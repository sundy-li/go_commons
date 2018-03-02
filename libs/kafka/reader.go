package kafka

//废弃,暂时不用,请用saram-cluster包
import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pborman/uuid"

	"github.com/Shopify/sarama"
	"github.com/samuel/go-zookeeper/zk"
)

type OnOffsetOverflow int

const (
	ThrowError    OnOffsetOverflow = 0 // 直接抛出 error
	ResetToOldest OnOffsetOverflow = 1 // 重置为最旧的 offset
	ResetToNewest OnOffsetOverflow = 2 // 重置为最新的 offset
)

var (
	MAX_ZK_CONSUMER_OCCUPY_RETRY = 20
	MAX_ZK_OFFSET_WRITE_TIME     = 10 * time.Second
	MAX_ZK_OFFSET_WRITE_COUNT    = int64(1000)
	MAX_ZK_CONNECT_TIMEOUT       = 180

	DefaultZkChroot = "/kafkago" // 避免与java的冲突
)

var (
	err_no_this_client = errors.New("在 zk 中没有该 client id")
)

type context struct {
	group string
	topic string

	threadSize int
	msgChans   []chan *sarama.ConsumerMessage //每个thread对应的输出端口

	kafkaConsumer sarama.Consumer
	kafkaAddr     []string

	zkConn             *zk.Conn
	zkAddr             []string
	zkChroot           string
	zkOffsetWriteTime  time.Duration
	zkOffsetWriteCount int64
	zkConnTimeout      int

	onOOf OnOffsetOverflow
}

// 控制 group topic中消息的读取与分配
type GroupTopicConsumer struct {
	*context

	clientId           string
	partitionConsumers map[int]*partitionConsumer
	partitionAssigned  map[int]int

	zkConsumerChan chan zk.Event

	exitChan, exitDoneChan chan string
	running                bool
}

type GroupTopicConfig struct {
	Topic     string
	Group     string
	ThreadNum int // 读取的线程数，与 MsgChans 返回的数目对应

	Kafka              []string
	Zookeeper          []string
	ZkChroot           string
	ZkOffsetWriteTime  time.Duration
	ZkOffsetWriteCount int64
	ZkConnTimeout      int

	OnOffsetOverflow OnOffsetOverflow
}

func NewGroupTopicConsumer(gtConf GroupTopicConfig) (*GroupTopicConsumer, error) {
	if !strings.HasPrefix(gtConf.ZkChroot, "/") {
		return nil, fmt.Errorf("conf.ZkChroot [%s] 不是以 / 为前缀", gtConf.ZkChroot)
	}

	kafkaConsumer, err := sarama.NewConsumer(gtConf.Kafka, nil)
	if err != nil {
		log.Printf("[NewGroupTopicConsumer] 连接kafka [%v] 失败:%s", gtConf.Kafka, err.Error())
		return nil, err
	}
	timeout := gtConf.ZkConnTimeout
	if timeout == 0 {
		timeout = MAX_ZK_CONNECT_TIMEOUT
	}
	zkConn, _, err := zk.Connect(gtConf.Zookeeper, time.Second*time.Duration(timeout))
	if err != nil {
		log.Printf("[NewGroupTopicConsumer] 连接zookeeper [%v] 失败:%s", gtConf.Kafka, err.Error())
		return nil, err
	}

	msgChans := make([]chan *sarama.ConsumerMessage, 0, gtConf.ThreadNum)
	for i := 0; i < gtConf.ThreadNum; i++ {
		msgChans = append(msgChans, make(chan *sarama.ConsumerMessage, 1000))
	}

	if gtConf.ZkChroot == "" {
		gtConf.ZkChroot = DefaultZkChroot
	}

	zkWT := gtConf.ZkOffsetWriteTime
	if zkWT == 0 {
		zkWT = MAX_ZK_OFFSET_WRITE_TIME
	}
	zkWC := gtConf.ZkOffsetWriteCount
	if zkWC == 0 {
		zkWC = MAX_ZK_OFFSET_WRITE_COUNT
	}

	ctx := &context{
		group: gtConf.Group,
		topic: gtConf.Topic,

		threadSize: gtConf.ThreadNum,
		msgChans:   msgChans,

		kafkaConsumer: kafkaConsumer,
		kafkaAddr:     gtConf.Kafka,

		zkConn:             zkConn,
		zkAddr:             gtConf.Zookeeper,
		zkChroot:           gtConf.ZkChroot,
		zkOffsetWriteCount: zkWC,
		zkOffsetWriteTime:  zkWT,
		zkConnTimeout:      timeout,

		onOOf: gtConf.OnOffsetOverflow,
	}
	this := &GroupTopicConsumer{
		context:        ctx,
		clientId:       genClientId(gtConf.Group),
		zkConsumerChan: make(chan zk.Event, 1),

		exitChan:     make(chan string, 1),
		exitDoneChan: make(chan string, 1),
	}
	go this.run()
	return this, nil
}

func (this *GroupTopicConsumer) name() string {
	return fmt.Sprintf("[GroupTopicConsumer:topic:%s, group:%s, clientid:%s]", this.topic, this.group, this.clientId)
}
func (this *GroupTopicConsumer) MsgChans() []<-chan *sarama.ConsumerMessage {
	ret := make([]<-chan *sarama.ConsumerMessage, 0, len(this.msgChans))
	for _, value := range this.msgChans {
		ret = append(ret, value)
	}
	return ret
}

func (this *GroupTopicConsumer) run() {
	zkChangeExit := make(chan string, 1)
	zkChangeDone := make(chan string, 1)
	firstWatch := make(chan string, 1)
	log.Printf("%s 开始监听变化", this.name())
	go this.watchZkConsumerChange(zkChangeExit, zkChangeDone, firstWatch)

	<-firstWatch

	zkHeartBeatChan := make(chan string, 1)
	zkHeartBeatDoneChan := make(chan string, 1)
	go this.zkHeartBeat(zkHeartBeatChan, zkHeartBeatDoneChan)

	log.Printf("%s 开始注册Consumer", this.name())
	err := this.registerConsumerInZK(this.clientId, map[string]int{this.topic: this.threadSize})
	if err != nil {
		log.Printf("注册 consumer 到 zk 中失败:%s", err.Error())
		return
	}
	this.running = true
	for {
		select {
		case <-this.zkConsumerChan:
			log.Printf("%s 开始 rebalance", this.name())
			assigned, err := this.rebalance(this.topic)
			if err != nil {
				if err == err_no_this_client {
					log.Printf("重新注册 consumer 到 zk")
					err := this.registerConsumerInZK(this.clientId, map[string]int{this.topic: this.threadSize})
					if err != nil {
						log.Printf("注册 consumer 到 zk 中失败:%s", err.Error())
					}
				} else {
					log.Printf("%s rebalance失败", this.name())
				}
				continue
			} else if reflect.DeepEqual(assigned, this.partitionAssigned) {
				log.Printf("rebalance 的结果和当前一样，不用重启Consumer")
				continue
			}
			this.closeAllPartitionConsumer()
			this.startPartitionConsumer(assigned)
			log.Printf("%s rebalance 完成", this.name())
		case <-this.exitChan:
			// 停止监控 zk change
			zkChangeExit <- "EXIT"
			<-zkChangeDone

			// 关闭 kafka
			this.closeAllPartitionConsumer()
			this.closeMsgChan()
			this.kafkaConsumer.Close()

			// 关闭 zk
			zkHeartBeatChan <- "EXIT"
			<-zkHeartBeatDoneChan
			this.zkConn.Close()

			this.exitDoneChan <- "DONE"
		}
	}
}

func (this *GroupTopicConsumer) closeMsgChan() {
	for _, ch := range this.msgChans {
		close(ch)
	}
}
func (this *GroupTopicConsumer) startPartitionConsumer(partMapThread map[int]int) {
	if len(this.partitionConsumers) > 0 {
		log.Printf("%s 试图重复启动 partitionConsumer", this.name())
		return
	}
	this.partitionAssigned = map[int]int{}
	this.partitionConsumers = map[int]*partitionConsumer{}
	for partitionNum, threadNum := range partMapThread {
		err := this.occupyPartition(this.clientId, threadNum, this.topic, partitionNum)
		if err != nil {
			log.Printf("%s 放弃占用 partition [%d] ", this.name(), partitionNum)
			continue
		}
		out := this.msgChans[threadNum]
		pc := newPartitionConsumer(this.context, partitionNum, out)
		this.partitionConsumers[partitionNum] = pc
		this.partitionAssigned[partitionNum] = threadNum
		go pc.run()
	}
}
func (this *GroupTopicConsumer) closeAllPartitionConsumer() {
	if len(this.partitionConsumers) <= 0 {
		return
	}
	var group sync.WaitGroup
	for _, partConsumer := range this.partitionConsumers {
		group.Add(1)
		go func(partConsumer *partitionConsumer) {
			partConsumer.exit()
			group.Done()
		}(partConsumer)
	}
	for partitionNum := range this.partitionConsumers {
		this.releasePartition(this.topic, partitionNum)
	}
	group.Wait()
	this.partitionConsumers = map[int]*partitionConsumer{}
}

func (this *GroupTopicConsumer) rebalance(topic string) (assigned map[int]int, err error) {
	clients, allClientThreads, err := this.getAllClientThreads(this.topic)
	if err != nil {
		return
	}
	if len(clients) == 0 {
		return assigned, err_no_this_client
	}
	var thisIn bool
	for _, c := range clients {
		if c == this.clientId {
			thisIn = true
		}
	}
	if !thisIn {
		return assigned, err_no_this_client
	}

	partitions32, err := this.kafkaConsumer.Partitions(topic)
	if err != nil {
		log.Printf("%s 获取 Topic [%s] Partition 失败:%s ", this.name(), topic, err.Error())
		return
	}
	partitions := make([]int, 0, len(partitions32))
	for _, p := range partitions32 {
		partitions = append(partitions, int(p))
	}
	sort.Ints(partitions)

	nPartsPerConsumer := len(partitions) / len(clients)
	nConsumersWithExtraPart := len(partitions) % len(clients)

	assigned = map[int]int{}
	for clientPosition, client := range clients {
		min := clientPosition
		if min > nConsumersWithExtraPart {
			min = nConsumersWithExtraPart
		}
		startParition := nPartsPerConsumer*clientPosition + min
		nParts := nPartsPerConsumer
		if clientPosition < nConsumersWithExtraPart {
			nParts += 1
		}

		if nParts <= 0 {
			continue
		}

		threadOfClient := allClientThreads[clientPosition]
		partPerThead := nParts / len(threadOfClient)
		exta := nParts % len(threadOfClient)
		for i, thread := range threadOfClient {
			e := i
			if i > exta {
				e = exta
			}
			sp := startParition + partPerThead*i + e

			np := partPerThead
			if i < exta {
				np += 1
			}

			for j := 0; j < np; j++ {
				log.Printf("%s Topic [%s] Partition [%d] 被 分配给了 Client [%s] Thread[%s]", this.name(), topic, partitions[sp+j], client, thread)
				if client == this.clientId {
					arr := strings.Split(thread, "-")
					assigned[partitions[sp+j]], _ = strconv.Atoi(arr[len(arr)-1])
				}
			}
		}
	}
	log.Printf("%s 分配结果:%v", this.name(), assigned)
	// for i := 0; i < len(partitions); i++ {
	// 	if _, ok := assigned[partitions[i]]; !ok {
	// 		continue
	// 	}
	// 	fmt.Println(partitions[i], ":", assigned[partitions[i]])
	// }
	return
}

func (this *GroupTopicConsumer) getAllClientThreads(topic string) (clients []string, clientThreads [][]string, err error) {
	clients, _, err = this.zkConn.Children(path.Join(this.zkChroot, "consumers", "ids"))
	if err != nil {
		return
	}
	sort.Strings(clients)
	for _, client := range clients {
		bin, _, err2 := this.zkConn.Get(path.Join(this.zkChroot, "consumers", "ids", client))
		if err2 != nil {
			err = err2
			return
		}
		data := map[string]interface{}{}
		json.Unmarshal(bin, &data)
		threadNum := 0

		// 多个consumer用了同一个group名称，但是不是同一个topic
		if _, ok := data["subscription"].(map[string]interface{})[topic]; !ok {
			clientThreads = append(clientThreads, []string{})
			continue
		}

		switch tmp := data["subscription"].(map[string]interface{})[topic].(type) {
		case int:
			threadNum = int(tmp)
		case float64:
			threadNum = int(tmp)
		default:
			panic(fmt.Sprintf("未知的类型 %T %v", tmp, tmp))
		}
		threads := []string{}
		for i := 0; i < threadNum; i++ {
			threads = append(threads, this.threadName(client, i))
		}
		clientThreads = append(clientThreads, threads)
	}

	// 移除没有占用该topic 的consumer
	newClients, newClientThreads := []string{}, [][]string{}
	for i, ct := range clientThreads {
		if len(ct) > 0 {
			newClients = append(newClients, clients[i])
			newClientThreads = append(newClientThreads, ct)
		}
	}
	return newClients, newClientThreads, err
}
func (this *GroupTopicConsumer) threadName(clientId string, threadNum int) string {
	return fmt.Sprintf("%s-%d", clientId, threadNum)
}

func (this *GroupTopicConsumer) occupyPartition(clientId string, threadNum int, topic string, partition int) (err error) {
	topicDir, err := mkZkDirIfNotExists(this.zkConn, this.zkChroot, "consumers", this.group, "owners", topic)
	if err != nil {
		return
	}

	partitionPath := topicDir + "/" + strconv.Itoa(partition)
	threadId := []byte(fmt.Sprintf("%s-%d", clientId, threadNum))
	i := 0
	for ; i < MAX_ZK_CONSUMER_OCCUPY_RETRY; i++ {
		_, err = this.zkConn.Create(partitionPath, threadId, zk.FlagEphemeral, zk.WorldACL(zk.PermAll))
		if err == zk.ErrNodeExists { // 还有其他节点占用了该 分区，等待其删除
			time.Sleep(time.Second * 1)
		} else {
			break
		}
	}
	if i == MAX_ZK_CONSUMER_OCCUPY_RETRY {
		log.Printf("%s 占用 partition [%d] 失败，一直有其他节点节点占用", this.name(), partition)
	}
	return
}

func (this *GroupTopicConsumer) releasePartition(topic string, partition int) (err error) {
	return this.zkConn.Delete(path.Join(this.zkChroot, "consumers", this.group, "owners", topic, strconv.Itoa(partition)), -1)
}

func (this *GroupTopicConsumer) getAllPartitions(topic string) (partitions []int, err error) {
	i32, err := this.kafkaConsumer.Partitions(topic)
	for _, p := range i32 {
		partitions = append(partitions, int(p))
	}
	return
}

func (this *GroupTopicConsumer) zkHeartBeat(closeChan, doneChan chan string) {
	for {
		select {
		case <-closeChan:
			log.Printf("%s 停止 zk 心跳", this.name())
			doneChan <- "EXITDONE"
			return
		case <-time.After(time.Second * 10):
			// log.Printf("%s zk 心跳", this.name())
			this.zkConn.Exists("/")
		}
	}
}
func (this *GroupTopicConsumer) registerConsumerInZK(clientId string, subscription map[string]int) (err error) {
	idsDir, err := mkZkDirIfNotExists(this.zkConn, this.zkChroot, "consumers", "ids")
	if err != nil {
		return
	}

	data, err := json.Marshal(map[string]interface{}{
		"version":      1,
		"subscription": subscription,
		"pattern":      "static",
		"timestamp":    strconv.FormatInt(time.Now().UnixNano()/1000/1000, 10),
	})

	if err != nil {
		return
	}
	_, err = this.zkConn.Create(idsDir+"/"+clientId, data, zk.FlagEphemeral, zk.WorldACL(zk.PermAll))
	log.Println("注册clientId：" + idsDir + "/" + clientId)
	return
}

// TODO: 添加 broker变化的watch
func (this *GroupTopicConsumer) watchZkConsumerChange(exitChan, doneChan, firstWatch chan string) (err error) {
	p, err := mkZkDirIfNotExists(this.zkConn, this.zkChroot, "consumers", "ids")
	if err != nil {
		log.Printf("监听 consumer [%s] 变化失败:%s", p, err.Error())
		return
	}
	log.Printf("监听 consumer [%s]变化", p)
	_, _, childrenChan, err := this.zkConn.ChildrenW(p)
	if err != nil {
		log.Printf("监听 consumer [%s] 变化失败:%s", p, err.Error())
		return
	}
	firstWatch <- "DONE"
	for {
		select {
		case e := <-childrenChan:
			log.Printf("监听到了 consumer [%s]变化", p)
			// 重新监听再返回变化信息，避免两个watch间的变化
			for {
				_, _, childrenChan, err = this.zkConn.ChildrenW(p)
				if err == nil {
					break
				}
				log.Printf("监听 consumer 变化失败:%s", err.Error())
				time.Sleep(time.Second * 10)
			}
			this.zkConsumerChan <- e
		case <-exitChan:
			log.Printf("停止监听 consumer 变化")
			doneChan <- "DONE"
			return
		}
	}
}

func (this *GroupTopicConsumer) Close() {
	if !this.running {
		return
	}
	this.exitChan <- "EXIT"
	select {
	case <-this.exitDoneChan:
	case <-time.After(time.Second * 10):
		log.Printf("[%s] 无法正常退出，强行退出", this.name())
	}
	this.running = false
}

type partitionConsumer struct {
	*context

	threadNum    int
	partitionNum int
	curOffset    int64

	kfkPartitionConsumer sarama.PartitionConsumer
	in                   <-chan *sarama.ConsumerMessage
	out                  chan<- *sarama.ConsumerMessage

	exitDoneChan chan string
	running      bool

	statusLock sync.Mutex // 用于防止初始化到一半已退出造成的kafka重复连接错误
}

func newPartitionConsumer(ctx *context, partitionNum int, out chan<- *sarama.ConsumerMessage) *partitionConsumer {
	this := &partitionConsumer{
		context:      ctx,
		out:          out,
		partitionNum: partitionNum,
		exitDoneChan: make(chan string, 1),
	}
	return this
}
func (this *partitionConsumer) name() string {
	return fmt.Sprintf("[PartitionConsumer-topic_%s-group_%s-parition_%d-thread-%d]", this.topic, this.group, this.partitionNum, this.threadNum)
}
func (this *partitionConsumer) run() (err error) {
	this.statusLock.Lock()
	curOffset, err := this.getOffset(this.topic, this.partitionNum)
	if err != nil {
		log.Printf("%s 获取offset失败，退出run:%s", this.name(), err.Error())
		this.statusLock.Unlock()
		return err
	}
	nextOne := curOffset + 1 //读取下一条
	if curOffset == -1 {
		nextOne = sarama.OffsetNewest // 没有设置，默认读取最新的
	}
	kfkPartitionConsumer, err := this.kafkaConsumer.ConsumePartition(this.topic, int32(this.partitionNum), nextOne)
	if err != nil {
		if strings.Contains(err.Error(), "The requested offset is outside the range of offsets") {
			switch this.onOOf {
			case ResetToNewest:
				kfkPartitionConsumer, err = this.kafkaConsumer.ConsumePartition(this.topic, int32(this.partitionNum), sarama.OffsetNewest)
			case ResetToOldest:
				kfkPartitionConsumer, err = this.kafkaConsumer.ConsumePartition(this.topic, int32(this.partitionNum), sarama.OffsetOldest)
			case ThrowError:
			default:
			}
		}

		if err != nil {
			highWater := int64(0)
			if tmpConsumer, e := this.kafkaConsumer.ConsumePartition(this.topic, int32(this.partitionNum), sarama.OffsetNewest); e == nil {
				highWater = tmpConsumer.HighWaterMarkOffset()
				tmpConsumer.Close()
			}
			log.Printf("%s 获取KafkaConsumer [topic:%s, partition:%d, requestOffset:%d, highWater:%d]失败，退出run: %s",
				this.name(), this.topic, this.partitionNum, nextOne, highWater, err.Error())
			return err
		}
	}

	this.curOffset = curOffset
	this.kfkPartitionConsumer = kfkPartitionConsumer
	this.in = this.kfkPartitionConsumer.Messages()

	var msg *sarama.ConsumerMessage
	var ok bool
	savedOffset := this.curOffset
	this.running = true
	this.statusLock.Unlock()
	for {
		select {
		case msg, ok = <-this.in:
			if !ok {
				if this.curOffset != savedOffset {
					savedOffset = this.updateOffset(this.curOffset)
				}
				this.exitDoneChan <- "DONE"
				this.running = false
				return nil
			}
			// log.Printf("%s 读取到一条消息:[%s],offset:[%d]", this.name(), string(msg.Value), msg.Offset)
			this.curOffset = msg.Offset

			if this.curOffset > savedOffset && (this.curOffset-savedOffset)%this.zkOffsetWriteCount == 0 {
				savedOffset = this.updateOffset(this.curOffset)
			}
			this.out <- msg
		case <-time.After(this.zkOffsetWriteTime):
			// fmt.Println(this.name(), "XXXXXXXX", this.curOffset, savedOffset)
			if this.curOffset != savedOffset {
				// fmt.Println(this.name(), "XXXXXXXXVVVVVVV")
				savedOffset = this.updateOffset(this.curOffset)
			}
		}
	}
	// this.running = false
	return nil
}
func (this *partitionConsumer) closeKafkaConnect() (err error) {
	if this.kfkPartitionConsumer != nil {
		return this.kfkPartitionConsumer.Close()
	}
	return nil
}

func (this *partitionConsumer) exit() {
	this.statusLock.Lock()
	err := this.closeKafkaConnect()
	if err != nil {
		log.Printf("%s 关闭 kafka 失败：%s", this.name(), err.Error())
	}
	this.statusLock.Unlock()
	if this.running {
		<-this.exitDoneChan
	}
	log.Printf("%s 成功退出", this.name())
}

func (this *partitionConsumer) updateOffset(offset int64) (savedOffset int64) {
	log.Printf("%s 更新offset [%d] ", this.name(), offset)
	if err := this.setOffset(this.topic, this.partitionNum, offset); err != nil {
		log.Printf("%s 更新Offset 失败:%s", this.name(), err.Error())
	}
	return offset
}

func (this *partitionConsumer) setOffset(topic string, partition int, offset int64) (err error) {
	partitionDir, err := mkZkDirIfNotExists(this.zkConn, this.zkChroot, "consumers", this.group, "offsets", topic)
	if err != nil {
		return
	}

	partitionPath := partitionDir + "/" + strconv.Itoa(partition)
	offsetBin := []byte(strconv.FormatInt(offset, 10))

	exists, _, err := this.zkConn.Exists(partitionPath)
	if err != nil {
		return
	}
	if !exists {
		_, err = this.zkConn.Create(partitionPath, offsetBin, 0, zk.WorldACL(zk.PermAll))
	} else {
		_, err = this.zkConn.Set(partitionPath, offsetBin, -1)
	}
	log.Printf("成功更新 Topic [%s] Partition [%d] Offset [%d]", topic, partition, offset)
	return
}

func (this *partitionConsumer) getOffset(topic string, partition int) (offset int64, err error) {
	partitionDir, err := mkZkDirIfNotExists(this.zkConn, this.zkChroot, "consumers", this.group, "offsets", topic)
	if err != nil {
		return
	}

	partitionPath := partitionDir + "/" + strconv.Itoa(partition)
	bin, _, err := this.zkConn.Get(partitionPath)
	switch err {
	case zk.ErrNoNode:
		return -1, nil
	case nil:
		return strconv.ParseInt(string(bin), 10, 64)
	default:
		return 0, err
	}
	// log.Printf("成功获取 Topic [%s] Partition [%d] Offset [%d]", topic, partition, offset)
	return
}

func mkZkDirIfNotExists(zkConn *zk.Conn, paths ...string) (finalPath string, err error) {
	curPath := ""
	for _, p := range paths {
		if curPath == "" {
			curPath = p
		} else {
			curPath = path.Join(curPath, p)
		}
		_, err = zkConn.Create(curPath, nil, 0, zk.WorldACL(zk.PermAll))
		if err != nil && err != zk.ErrNodeExists {
			return "", err
		}
	}
	return curPath, nil
}
func genClientId(group string) (clientId string) {
	hostname, _ := os.Hostname()
	return fmt.Sprintf("%s_%s-%d-%s", group, hostname, time.Now().UnixNano()/1000/1000, uuid.New()[:8])
}

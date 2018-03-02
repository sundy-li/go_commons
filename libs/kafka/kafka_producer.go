package kafka

import (
	"time"

	"github.com/Shopify/sarama"
)

// 把数据导入 kafka 的辅助 go 程

type KafkaImporter struct {
	client   sarama.Client
	producer sarama.AsyncProducer

	exitChan     chan string
	exitDoneChan chan string

	onErr func(topic string, msg []byte, err error)
}

//外部调用
func NewKafkaImporterWithConf(brokers []string, clientConf *sarama.Config, onErr func(topic string, msg []byte, err error)) (importer *KafkaImporter, err error) {
	client, err := sarama.NewClient(brokers, clientConf)
	if err != nil {
		return
	}

	producer, err := sarama.NewAsyncProducerFromClient(client)
	if err != nil {
		return
	}
	importer = &KafkaImporter{
		client:   client,
		producer: producer,

		// logger:       logger,
		exitChan:     make(chan string, 1),
		exitDoneChan: make(chan string, 1),

		onErr: onErr,
	}
	go importer._checkErr()
	return
}

func NewKafkaImporter(clientId string, brokers []string, onErr func(topic string, msg []byte, err error)) (importer *KafkaImporter, err error) {
	clientConf := sarama.NewConfig()
	clientConf.ClientID = clientId

	clientConf.Producer.MaxMessageBytes = 1000 * 100
	clientConf.Producer.Compression = sarama.CompressionSnappy
	clientConf.Producer.Partitioner = sarama.NewRandomPartitioner

	clientConf.Producer.Flush.Frequency = 10 * time.Second
	clientConf.Producer.Flush.MaxMessages = 100
	clientConf.Producer.Flush.Bytes = 1000 * 100

	return NewKafkaImporterWithConf(brokers, clientConf, onErr)
}

func (this *KafkaImporter) _checkErr() {
	var producerErr *sarama.ProducerError
FOREVER:
	for {
		select {
		case producerErr = <-this.producer.Errors():
			if producerErr != nil && producerErr.Err != nil && this.onErr != nil {
				valueBin, _ := producerErr.Msg.Value.Encode()
				this.onErr(producerErr.Msg.Topic, valueBin, producerErr.Err)
			}
		case <-this.exitChan:
			break FOREVER
		}
	}
	this.exitDoneChan <- "EXITDONE"
}

func (this *KafkaImporter) Exit() {
	this.exitChan <- "EXIT"
	<-this.exitDoneChan
	this.cleanup()
}

func (this *KafkaImporter) cleanup() {
	if this.producer != nil {
		this.producer.Close()
	}
	if this.client != nil && !this.client.Closed() {
		this.client.Close()
	}
}

func (this *KafkaImporter) Save(topic string, msg []byte) {
	this.SaveMsg(&sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder(msg),
	})
}
func (this *KafkaImporter) SaveMsg(msg *sarama.ProducerMessage) {
	this.producer.Input() <- msg
}

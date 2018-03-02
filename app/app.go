package app

import (
	"flag"
	"log"
	"runtime"

	"go_commons/utils"
)

var CPU_NUM int

func init() {
	flag.IntVar(&CPU_NUM, "cpunum", 1, "指定cpu数量，默认一个cpu")
}

func ParseFlag() {
	if !flag.Parsed() {
		flag.Parse()
	}
	runtime.GOMAXPROCS(CPU_NUM)
}

func Run(appName string, initFunc, jobFunc, cleanupFunc func() error) {
	log.Printf("初始化 [%s]", appName)
	if err := initFunc(); err != nil {
		log.Printf("初始化 [%s] 失败：[%s]", appName, err)
		panic(err)
	}
	log.Printf("初始化 [%s] 完成", appName)
	go func() {
		if err := jobFunc(); err != nil {
			log.Printf("[%s] 运行出错：[%v]", appName, err)
			panic(err)
		}
	}()

	utils.WaitForExitSign()
	log.Printf("[%s] 监听到退出信息，开始清理", appName)
	if err := cleanupFunc(); err != nil {
		log.Printf("[%s] 清理失败：[%v]", appName, err)
		panic(err)
	}
	log.Printf("[%s] 清理完成，成功退出", appName)
}

func Funcs(funcs ...func() error) func() error {
	return func() error {
		for _, fun := range funcs {
			if err := fun(); err != nil {
				return err
			}
		}
		return nil
	}
}
func LogWrapper(msg string, fun func() error) func() error {
	return func() error {
		log.Println(msg + " 开始")
		if err := fun(); err != nil {
			log.Printf("%s 失败:%v", msg, err)
			return err
		}
		log.Println(msg + " 完成")
		return nil
	}
}

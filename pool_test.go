package goroutine_pool_demo

import (
	"log"
	"runtime"
	"testing"
	"time"
)

func TestGoPool_AddTask(t *testing.T) {
	// 启动监控goroutine每隔一秒输出当前所有协程数量
	waitC := make(chan bool)
	go func() {
		for {
			log.Printf("[main] Total current goroutine: %d", runtime.NumGoroutine())
			time.Sleep(1 * time.Second)
		}
	}()

	// 启动并发3的池子
	myPool := NewGoPool(WithPoolSize(3), WithJobQueueSize(2048))
	myPool.Run()

	// 发送一共5个任务
	resultC := make(chan JobResult, 5000)
	for i := 0; i < 50; i++ {
		myPool.AddTask(func(resultCh chan<- JobResult, args []interface{}) {
			taskID := args[0].(int)
			log.Println("执行任务", taskID)
			time.Sleep(3 * time.Second)
			resultCh <- JobResult{taskID, taskID * 2}
		},
			resultC, i)
	}

	go func() {
		// 收集结果信息
		for i := 0; i < 5000; i++ {
			res := <-resultC
			log.Println("已完成任务：", res.id)
		}
	}()

	// 10s后扩容
	time.Sleep(10 * time.Second)
	myPool.SetPoolSize(10)

	// 20s后缩容
	time.Sleep(10 * time.Second)
	myPool.SetPoolSize(5)

	// 30s后，终止协程池
	time.Sleep(10 * time.Second)
	myPool.Stop()
	<-waitC
}

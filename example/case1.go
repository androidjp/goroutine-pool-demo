package main

import (
	"fmt"
	goroutine_pool_demo "goroutine-pool-demo"
	"time"
)

func main() {
	pool := goroutine_pool_demo.NewGoPool("myPool")
	pool.Run()

	pool.AddTask(func(resultCh chan<- goroutine_pool_demo.JobResult, args []interface{}) {
		fmt.Println("do sth")
	}, nil)

	time.Sleep(3 * time.Second)
	pool.Stop()
}

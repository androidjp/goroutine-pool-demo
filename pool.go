package goroutine_pool_demo

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"
)

type JobExecFunc func(resultCh chan<- JobResult, args []interface{})

type Job struct {
	execFunc JobExecFunc
	resultCh chan<- JobResult
	args     []interface{}
}

func (p *GoPool) Refresh() {
	newCnt := p.opts.PoolSize
	p.lock.Lock()
	defer p.lock.Unlock()
	// 根据目前最大协程数，扩缩容所有运行中的协程
	curRunningWorkerCnt := len(p.CurRunningWorkers)
	if curRunningWorkerCnt == newCnt {
		fmt.Println("当前协程数没变化，忽略")
		return
	}
	if curRunningWorkerCnt < newCnt {
		fmt.Println("当前协程数是：", curRunningWorkerCnt, ", 最新协程数配置是：", newCnt, "，准备扩容")
		// 扩容
		needIncCnt := newCnt - curRunningWorkerCnt
		for i := 0; i < needIncCnt; i++ {
			p.StartNewWorker()
		}
	} else {
		fmt.Println("当前协程数是：", curRunningWorkerCnt, ", 最新协程数配置是：", newCnt, "，准备缩容")
		// 缩容
		needExitCnt := curRunningWorkerCnt - newCnt

		cur := 0
		for key, cancelFunc := range p.CurRunningWorkers {
			if cur == needExitCnt {
				break
			}
			// 停止对应协程
			cancelFunc()
			fmt.Println("尝试关闭worker：", key)
			cur++
		}
		// 最终，阻塞地删除这几个对应的key
		for i := 0; i < needExitCnt; i++ {
			delete(p.CurRunningWorkers, <-p.workerExitSuccessfullyCh)
		}
	}
}

type GoPool struct {
	opts                     *GoPoolOptions                // 配置参数
	jobChan                  chan Job                      // 任务通道(给到所有的协程 来争抢消费)
	CurRunningWorkers        map[string]context.CancelFunc // 正在运行中的协程
	workerExitSuccessfullyCh chan string                   // worker退出成功的通道
	stopCh                   chan struct{}                 // 退出
	lock                     sync.Mutex                    // 悲观锁
}

func NewGoPool(opt ...GoPoolOption) *GoPool {
	opts := NewGoPoolOptions()
	for _, o := range opt {
		o(opts)
	}

	return &GoPool{
		opts:                     opts,
		jobChan:                  make(chan Job, opts.JobQueueSize),
		workerExitSuccessfullyCh: make(chan string),
		stopCh:                   make(chan struct{}),
		CurRunningWorkers:        make(map[string]context.CancelFunc),
	}
}

// SetPoolSize 设置协程池大小
func (p *GoPool) SetPoolSize(size int) {
	p.opts.PoolSize = size
}

func (p *GoPool) AddTask(jobFunc JobExecFunc, resultCh chan<- JobResult, args ...interface{}) {
	ji := Job{
		execFunc: jobFunc,
		resultCh: resultCh,
		args:     args,
	}
	p.jobChan <- ji
}

func (p *GoPool) Stop() {
	close(p.jobChan)
	close(p.stopCh)
}

func (p *GoPool) Run() {
	for i := 0; i < p.opts.PoolSize; i++ {
		p.StartNewWorker()
	}
	// 启动一个专属协程，去后台不断监测配置的变更，一旦有扩容或者缩容，则响应
	go func() {
		for {
			select {
			case <-p.stopCh:
				// 后台扩缩容协程退出
				fmt.Println("后台扩缩容协程退出")
				return
			default:
				time.Sleep(4 * time.Second)
				fmt.Println("扩缩容监测协程启动")
				p.Refresh()
			}
		}
	}()
}

func (p *GoPool) StartNewWorker() {
	ctxWithCancel, cancelFunc := context.WithCancel(context.Background())
	key := fmt.Sprintf("worker-%s", time.Now().String())
	p.CurRunningWorkers[key] = cancelFunc
	go func(key string, ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				fmt.Printf("此worker[%s] 由于缩容，退出工作\n", key)
				p.workerExitSuccessfullyCh <- key // 通知维护器，此协程退出了
				return
			case <-p.stopCh:
				fmt.Printf("随着协程池退出，worker[%s]退出", key)
				return
			default:
				if j, ok := <-p.jobChan; ok {
					// something normal
					j.execFunc(j.resultCh, j.args)
				} else {
					// 当通道已经被关闭，则可以手动结束此协程
					// something bad , need some deals
					// 我们清除掉系统正在运行的goroutine，手动调用runtime.Goexit().
					runtime.Goexit() // clear the goroutine
				}
			}
		}
	}(key, ctxWithCancel)
}

type JobResult struct {
	id    int
	value int
}

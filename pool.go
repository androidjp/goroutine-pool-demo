package goroutine_pool_demo

import (
	"context"
	"fmt"
	gonanoid "github.com/matoous/go-nanoid/v2"
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

type JobResult struct {
	err  error
	msg  string
	data interface{}
}

type GoPool struct {
	Name                     string                        // 协程池名称
	opts                     *GoPoolOptions                // 配置参数
	jobChan                  chan Job                      // 任务通道(给到所有的协程 来争抢消费)
	CurRunningWorkers        map[string]context.CancelFunc // 正在运行中的协程
	workerExitSuccessfullyCh chan string                   // worker退出成功的通道
	stopCh                   chan struct{}                 // 退出
	lock                     sync.Mutex                    // 悲观锁
}

func NewGoPool(name string, opt ...GoPoolOption) *GoPool {
	opts := NewGoPoolOptions()
	for _, o := range opt {
		o(opts)
	}

	return &GoPool{
		Name:                     name,
		opts:                     opts,
		jobChan:                  make(chan Job, opts.JobQueueSize),
		workerExitSuccessfullyCh: make(chan string),
		stopCh:                   make(chan struct{}),
		CurRunningWorkers:        make(map[string]context.CancelFunc),
	}
}

func (p *GoPool) Refresh() {
	newCnt := p.opts.PoolSize
	curRunningWorkerCnt := len(p.CurRunningWorkers)
	if curRunningWorkerCnt == newCnt {
		return
	}
	p.lock.Lock()
	defer p.lock.Unlock()
	curRunningWorkerCnt = len(p.CurRunningWorkers) // 加锁后二次确认是否有扩容缩容需求
	if curRunningWorkerCnt == newCnt {
		return
	}
	// 根据目前最大协程数，扩缩容所有运行中的协程
	if curRunningWorkerCnt < newCnt {
		//log.WithField("pool_name", p.Name).Debug("当前协程数是：", curRunningWorkerCnt, ", 最新协程数配置是：", newCnt, "，准备扩容")
		// 扩容
		needIncCnt := newCnt - curRunningWorkerCnt
		for i := 0; i < needIncCnt; i++ {
			p.startNewWorker()
		}
	} else {
		//log.WithField("pool_name", p.Name).Debug("当前协程数是：", curRunningWorkerCnt, ", 最新协程数配置是：", newCnt, "，准备缩容")
		// 缩容
		needExitCnt := curRunningWorkerCnt - newCnt

		cur := 0
		for key, cancelFunc := range p.CurRunningWorkers {
			if cur == needExitCnt {
				break
			}
			// 停止对应协程
			cancelFunc()
			//log.WithField("pool_name", p.Name).Debug("尝试关闭worker：", key)
			fmt.Println("key: ", key)
			cur++
		}
		// 最终，阻塞地删除这几个对应的key
		for i := 0; i < needExitCnt; i++ {
			delete(p.CurRunningWorkers, <-p.workerExitSuccessfullyCh)
		}
	}
}

func (p *GoPool) AddTask(jobFunc JobExecFunc, resultCh chan<- JobResult, args ...interface{}) {
	ji := Job{
		execFunc: jobFunc,
		resultCh: resultCh,
		args:     args,
	}
	for {
		select {
		case p.jobChan <- ji:
			//log.WithField("pool_name", p.Name).Debugf("成功写入jobChan!!")
			return
		default:
			//log.WithField("pool_name", p.Name).Warn("等待写入jobChan.........")
			time.Sleep(1 * time.Second)
		}
	}
}

func (p *GoPool) Stop() {
	close(p.jobChan)
	close(p.stopCh)
}

func (p *GoPool) Run() {
	for i := 0; i < p.opts.PoolSize; i++ {
		p.startNewWorker()
	}
	if p.opts.EnableRefresh {
		go func() {
			for {
				// 启动一个专属协程，去后台不断监测配置的变更，一旦有扩容或者缩容，则响应
				select {
				case <-p.stopCh:
					// 后台扩缩容协程退出
					//log.WithField("pool_name", p.Name).Debug("后台扩缩容协程退出")
					return
				default:
					time.Sleep(time.Duration(p.opts.RefreshIntervalSec) * time.Second)
					// 尝试调用opts更新函数，更新下opts的参数
					if p.opts.OptsRefreshFunc != nil {
						p.opts.OptsRefreshFunc(p.opts)
					}
					p.Refresh()
				}
			}
		}()
	}
}

func (p *GoPool) startNewWorker() {
	ctxWithCancel, cancelFunc := context.WithCancel(context.Background())
	nanoID, _ := gonanoid.New()
	key := fmt.Sprintf("%s-worker-%d-%s", p.Name, time.Now().Unix(), nanoID)
	p.CurRunningWorkers[key] = cancelFunc
	//log.WithField("pool_name", p.Name).Debug("生成 worker:", key)
	go func(key string, ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				//log.WithField("pool_name", p.Name).Debugf("此worker[%s] 由于缩容，退出工作\n", key)
				p.workerExitSuccessfullyCh <- key // 通知维护器，此协程退出了
				return
			case <-p.stopCh:
				//log.WithField("pool_name", p.Name).Debugf("随着协程池退出，worker[%s]退出", key)
				return
			case j, ok := <-p.jobChan:
				//log.WithField("pool_name", p.Name).Debugf("此worker[%s] 收到了任务\n", key)
				if ok {
					// something normal
					//log.WithField("pool_name", p.Name).Debugf("此worker[%s] 收到并准备执行任务 len:%d, cap:%d\n", key, len(p.jobChan), cap(p.jobChan))
					j.execFunc(j.resultCh, j.args)
					//log.WithField("pool_name", p.Name).Debugf("此worker[%s] 执行任务完毕！！！！！！！\n", key)
				} else {
					runtime.Goexit() // clear the goroutine
				}
			}
		}
	}(key, ctxWithCancel)
}

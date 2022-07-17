package goroutine_pool_demo

type GoPoolOption func(opts *GoPoolOptions)

type GoPoolOptions struct {
	PoolSize           int                      // 协程池大小（默认：5）
	JobQueueSize       int                      // 任务投递队列大小（默认：100）
	EnableRefresh      bool                     // 是否启用刷新机制（扩缩容）
	RefreshIntervalSec int64                    // （扩容/缩容）后台协程的刷新时间间隔（单位：秒）
	OptsRefreshFunc    GoPoolOptionsRefreshFunc // GoPoolOptions本身配置拉取同步的函数
}

func WithPoolSize(size int) GoPoolOption {
	return func(opts *GoPoolOptions) {
		opts.PoolSize = size
	}
}

func WithJobQueueSize(size int) GoPoolOption {
	return func(opts *GoPoolOptions) {
		opts.JobQueueSize = size
	}
}

func WithRefresh(enable bool, intervalSec int64, f GoPoolOptionsRefreshFunc) GoPoolOption {
	return func(opts *GoPoolOptions) {
		opts.EnableRefresh = enable
		opts.RefreshIntervalSec = intervalSec
		opts.OptsRefreshFunc = f
	}
}

type GoPoolOptionsRefreshFunc func(opts *GoPoolOptions)

func NewGoPoolOptions() *GoPoolOptions {
	return &GoPoolOptions{
		PoolSize:           5,     // 默认协程池协程数量：5
		JobQueueSize:       100,   // 默认任务队列缓存任务数量是100
		EnableRefresh:      false, // 是否启动刷新机制，默认是关闭
		RefreshIntervalSec: 5,     // 默认5s 刷新一次
		OptsRefreshFunc:    nil,
	}
}

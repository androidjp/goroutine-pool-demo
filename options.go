package goroutine_pool_demo

type GoPoolOption func(opts *GoPoolOptions)

type GoPoolOptions struct {
	PoolSize     int // 协程池大小（默认：5）
	JobQueueSize int // 任务投递队列大小（默认：100）
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

func NewGoPoolOptions() *GoPoolOptions {
	return &GoPoolOptions{
		PoolSize:     5,
		JobQueueSize: 100,
	}
}

package gopool

import (
	"context"
	"errors"
	"sync"
	"time"
)

// 2种任务定义

// 无参、无返回值
type Job = func()

// 有参、有返回值
type Callable = func(ctx context.Context) interface{}

var (
	ErrFutureGetTimeout = errors.New("future get timed out")
	ErrPoolClosed       = errors.New("goroutine pool closed")
)

var defaultPool = NewPool(DefaultWorkerPool(), defaultConfig)

func Submit(job Job) error {
	return defaultPool.Submit(job)
}

func Call(ctx context.Context, callable Callable) (Future, error) {
	return defaultPool.Call(ctx, callable)
}

func Shutdown() {
	defaultPool.Shutdown()
}

// Future接口，调用Callable任务时返回，用来取消执行、获取执行结果
type Future interface {
	Cancel()                                             // 取消执行
	Get() interface{}                                    // 阻塞方法
	GetWithTimeout(d time.Duration) (interface{}, error) // 带超时
}

// future implement Future interface
type future struct {
	v      interface{}        // 函数返回值
	ch     chan interface{}   // 接收函数返回值
	once   sync.Once          // do cancel() once
	ctx    context.Context    // 判断此future是否cancel
	cancel context.CancelFunc // ctx的cancelFunc，用来执行cancel
}

// 取消执行，但不会中断已经执行的任务
func (f *future) Cancel() {
	f.once.Do(func() {
		f.cancel()
	})
}

func (f *future) Get() interface{} {
	select {
	case f.v = <-f.ch:
		f.Cancel()
	case <-f.ctx.Done():
	}
	return f.v
}

func (f *future) GetWithTimeout(d time.Duration) (interface{}, error) {
	if f.v != nil {
		return f.v, nil
	}
	timer := time.NewTimer(d)
	select {
	case f.v = <-f.ch:
		f.Cancel()
	case <-f.ctx.Done():
		timer.Stop()
	case <-timer.C:
		timer.Stop()
		return nil, ErrFutureGetTimeout
	}
	return f.v, nil
}

type Config struct {
	MaxWorkerNum   int
	WorkerIdleTime time.Duration
}

var defaultConfig = &Config{
	MaxWorkerNum: 1000,
	WorkerIdleTime: 10 * time.Second,
}

type WorkerPool interface {
	Init(*Config)
	Destroy()
	Dispatch(Job)
}

type Pool struct {
	wp   WorkerPool
	once sync.Once
	done chan struct{}
}

// 提供一个wp的实例和一份配置
func NewPool(wp WorkerPool, conf *Config) *Pool {
	wp.Init(conf)
	p := &Pool{
		wp:   wp,
		done: make(chan struct{}),
	}
	return p
}

func (p *Pool) Shutdown() {
	p.once.Do(func() {
		close(p.done)
		p.wp.Destroy()
	})
}

func (p *Pool) Call(ctx context.Context, callable Callable) (Future, error) {
	select {
	case <-p.done:
		return nil, ErrPoolClosed
	default:
	}

	ch := make(chan interface{})
	c, cf := context.WithCancel(context.Background()) // c 与 ctx 没有任何关系
	job := func() {
		select {
		case <-c.Done(): // 若任务开始前已取消，则不执行任务
			return
		default:
			ch <- callable(ctx) // 任务开始后再取消，任务仍将执行
		}
	}
	p.wp.Dispatch(job)
	return &future{
		v:      nil,
		ch:     ch,
		ctx:    c,
		cancel: cf,
	}, nil
}

func (p *Pool) Submit(job Job) error {
	select {
	case <-p.done:
		return ErrPoolClosed
	default:
		p.wp.Dispatch(job)
		return nil
	}
}

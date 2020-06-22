package gopool

import (
	"runtime"
	"time"
)

var workerChanCap int

func init() {
	workerChanCap = func() int {
		if runtime.GOMAXPROCS(0) == 1 {
			return 0
		}
		return 1
	}()
}

type worker struct {
	wp   *workerpool
	id   int
	job  chan Job
	last time.Time
}

func newworker(wp *workerpool, id int) *worker {
	w := &worker{
		wp:  wp,
		id:  id,
		job: make(chan Job, workerChanCap),
	}
	return w
}

func (w *worker) run() {
	go func() {
		defer func() {
			recover()
		}()

		w.last = time.Now()
		for job := range w.job {
			if job != nil {
				job()
				w.last = time.Now()
				w.wp.ready <- w.id
			} else {
				w.wp.exited <- w.id
				return
			}
		}
	}()
}

type workerpool struct {
	conf   *Config
	ws     []*worker
	exited chan int
	ready  chan int
	done   chan struct{}
}

func (wp *workerpool) Init(config *Config) {
	wp.conf = config
	wp.ws = make([]*worker, config.MaxWorkerNum)
	wp.exited = make(chan int, config.MaxWorkerNum)
	wp.ready = make(chan int, config.MaxWorkerNum)
	wp.done = make(chan struct{})

	for i := 0; i < config.MaxWorkerNum; i++ {
		wp.ws[i] = newworker(wp, i)
		wp.exited <- i
	}

	go func() {
		for {
			select {
			case <-wp.done:
				return
			default:
				time.Sleep(100 * time.Millisecond)
			}

			for ready := range wp.ready { // worker不会被立即回收，无伤大雅
				w := wp.ws[ready]
				if time.Now().Sub(w.last) < w.wp.conf.WorkerIdleTime {
					wp.ready <- ready
					break
				}
				w.job <- nil
			}
		}
	}()
}

func (wp *workerpool) Dispatch(job Job) {
	select {
	case <-wp.done:
		return
	case i := <-wp.ready:
		wp.ws[i].job <- job
		return
	case ei := <-wp.exited:
		w := wp.ws[ei]
		w.run()
		w.job <- job
		return
	}
}

func (wp *workerpool) Destroy() {
	close(wp.done)
	for len(wp.ready) > 0 {
		ready := <-wp.ready
		wp.ws[ready].job <- nil
	}
}

func DefaultWorkerPool() WorkerPool {
	return &workerpool{}
}
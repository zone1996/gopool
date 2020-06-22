package gopool

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestSubmit(t *testing.T) {
	p := NewPool(DefaultWorkerPool(), &Config{
		MaxWorkerNum:   100,
		WorkerIdleTime: 5 * time.Second,
	})
	wg := sync.WaitGroup{}
	f := func() {
		time.Sleep(time.Second)
		wg.Done()
	}
	workernum := 10
	wg.Add(workernum)
	for i := 0; i < workernum; i++ {
		go p.Submit(f)
	}
	wg.Wait()
	p.Shutdown()
	fmt.Println("Over")
}

func TestCall(t *testing.T) {
	p := NewPool(DefaultWorkerPool(), &Config{
		MaxWorkerNum:   100,
		WorkerIdleTime: 5 * time.Second,
	})

	call := func(ctx context.Context) interface{} {
		time.Sleep(3 * time.Second)
		return 10
	}

	step := 0
	f, err := p.Call(context.Background(), call)
	fmt.Println(" time:", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Println(step, " ", err)

	step += 1
	v, err := f.GetWithTimeout(1 * time.Second)
	fmt.Println(step, " ", err)

	v = f.Get()
	fmt.Println(v, " time:", time.Now().Format("2006-01-02 15:04:05"))
}

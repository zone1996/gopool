package gopool

import (
	"sync"
	"testing"
	"time"
)

const (
	jobNum         = 1000000
	maxWorkerNum   = 20000 // 5w  10w, 2w is good
	workerIdleTime = 10 * time.Second
)

func testJob() {
	time.Sleep(time.Duration(10) * time.Millisecond)
}

func BenchmarkGoroutines(b *testing.B) {
	var wg sync.WaitGroup

	for i := 0; i < b.N; i++ {
		wg.Add(jobNum)
		for j := 0; j < jobNum; j++ {
			go func() {
				testJob()
				wg.Done()
			}()
		}
		wg.Wait()
	}
}

func BenchmarkGoPool(b *testing.B) {
	var wg sync.WaitGroup
	p := NewPool(DefaultWorkerPool(), &Config{
		MaxWorkerNum:   maxWorkerNum,
		WorkerIdleTime: workerIdleTime,
	})
	defer p.Shutdown()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wg.Add(jobNum)
		for j := 0; j < jobNum; j++ {
			_ = p.Submit(func() {
				testJob()
				wg.Done()
			})
		}
		wg.Wait()
	}
	b.StopTimer()
}

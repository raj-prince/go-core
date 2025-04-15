package main_test

import (
	"fmt"
	"sync"
	"testing"
)

const NumWorkers = 200
const TaskPool = 99

// Task to be executed
func basicTask() {
	x := 0
	x++
}

// Static Pool
type staticPool struct {
	work chan func()
	wg   sync.WaitGroup
}

func newStaticPool(numWorkers int) *staticPool {
	pool := &staticPool{
		work: make(chan func(), TaskPool),
	}

	for i := 0; i < numWorkers; i++ {
		pool.wg.Add(1)
		go func() {
			defer pool.wg.Done()
			for task := range pool.work {
				task()
			}
		}()
	}
	return pool
}

func (p *staticPool) submit(task func()) {
	p.work <- task
}

func (p *staticPool) close() {
	close(p.work)
	p.wg.Wait()
}

// Dynamic Pool
type dynamicPool struct {
	maxWorkers int
	work       chan func()
	wg         sync.WaitGroup
	active     int
}

func newDynamicPool(maxWorkers int) *dynamicPool {
	pool := &dynamicPool{
		maxWorkers: maxWorkers,
		work:       make(chan func(), TaskPool),
	}

	go func() {
		for task := range pool.work {
			if pool.active < pool.maxWorkers {
				pool.active++
				pool.wg.Add(1)
				go func() {
					defer func() {
						pool.active--
						pool.wg.Done()
					}()
					task()
				}()
			} else {
				fmt.Println("Pool is full, task rejected.")
			}
		}
	}()
	return pool

}

func (p *dynamicPool) submit(task func()) {
	p.work <- task
}

func (p *dynamicPool) close() {
	close(p.work)
	p.wg.Wait()
}

func BenchmarkDynamicPoolScheduling(b *testing.B) {
	pool := newDynamicPool(NumWorkers)
	defer pool.close()

	for i := 0; i < b.N; i++ {
		pool.submit(basicTask)
	}
}

func BenchmarkStaticPoolScheduling(b *testing.B) {
	pool := newStaticPool(NumWorkers)
	defer pool.close()

	for i := 0; i < b.N; i++ {
		pool.submit(basicTask)
	}
}

func BenchmarkDynamicPoolCreation(b *testing.B) {

	for i := 0; i < b.N; i++ {
		pool := newDynamicPool(NumWorkers)
		pool.close()
	}
}

func BenchmarkStaticPoolCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		pool := newStaticPool(NumWorkers)
		pool.close()
	}
}

/**
Command: go test -bench . -test.benchmem

Output:
BenchmarkDynamicPoolScheduling-96        2235728               544.1 ns/op            24 B/op          1 allocs/op
BenchmarkStaticPoolScheduling-96         2588268               400.1 ns/op             0 B/op          0 allocs/op
BenchmarkDynamicPoolCreation-96           900556              1287 ns/op            1072 B/op          4 allocs/op
BenchmarkStaticPoolCreation-96              5875            195249 ns/op            4341 B/op        203 allocs/op

Conclusion:
1. Scheduling cost is higher for dynamic pool.
2. Creation cpu/mem cost is higher for staticPool. Also, memory of static pool creation is not that high, so I would be
more inclined towards implementation static pool. As not much memory, and clean implementation of priority queue.
*/

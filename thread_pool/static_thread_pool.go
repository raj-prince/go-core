package thread_pool

import (
	"log"
	"sync"
)

// StaticThreadPool is a group of workers that can be used to execute a task
type StaticThreadPool struct {
	// Number of workers running in this group
	worker uint32

	// Channel to close all the workers
	close chan int

	// Wait group to wait for all workers to finish
	wg sync.WaitGroup

	// Channel to hold pending requests
	priorityCh chan Task
	normalCh   chan Task
}

// newStaticThreadPool creates a new thread pool
func NewStaticThreadPool(count uint32) *StaticThreadPool {
	log.Printf("StaticThreadpool: creating with worker: %d\n", count)
	if count == 0 {
		return nil
	}

	return &StaticThreadPool{
		worker:     count,
		close:      make(chan int, count),
		priorityCh: make(chan Task, count*2),
		normalCh:   make(chan Task, count*5000),
	}
}

// Start all the workers and wait till they start receiving requests
func (t *StaticThreadPool) Start() {
	// 10% threads will listen only on high priority channel
	highPriority := (t.worker * 10) / 100

	for i := uint32(0); i < t.worker; i++ {
		t.wg.Add(1)
		go t.Do(i < highPriority)
	}
}

// Stop all the workers threads
func (t *StaticThreadPool) Stop() {
	for i := uint32(0); i < t.worker; i++ {
		t.close <- 1
	}

	t.wg.Wait()

	close(t.close)
	close(t.priorityCh)
	close(t.normalCh)
}

// Schedule the download of a block
func (t *StaticThreadPool) Schedule(urgent bool, item Task) {
	// urgent specifies the priority of this task.
	// true means high priority and false means low priority
	if urgent {
		t.priorityCh <- item
	} else {
		t.normalCh <- item
	}
}

// Do is the core task to be executed by each worker thread
func (t *StaticThreadPool) Do(priority bool) {
	defer t.wg.Done()

	if priority {
		// This thread will work only on high priority channel
		for {
			select {
			case item := <-t.priorityCh:
				item.Execute()
			case <-t.close:
				return
			}
		}
	} else {
		// This thread will work only on both high and low priority channel
		for {
			select {
			case item := <-t.priorityCh:
				item.Execute()
			case item := <-t.normalCh:
				item.Execute()
			case <-t.close:
				return
			}
		}
	}
}

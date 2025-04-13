package thread_pool

import (
	"log"
	"sync"
	"sync/atomic"
)

// With current implementation, normalworker can't pick the priority job.

// DynamicThreadPool manages a pool of workers created on demand,
// with separate concurrency limits for priority and normal tasks.
// Workers execute one task and terminate.
type DynamicThreadPool struct {
	maxPriorityWorkers uint32 // Max concurrent workers for priority tasks.
	maxNormalWorkers   uint32 // Max concurrent workers for normal tasks.

	priorityCh chan Task     // Channel for high-priority tasks.
	normalCh   chan Task     // Channel for normal-priority tasks.
	closeCh    chan struct{} // Channel to signal workers to stop.

	wg sync.WaitGroup // Waits for all active workers to finish.

	prioritySem chan struct{} // Semaphore limiting priority workers.
	normalSem   chan struct{} // Semaphore limiting normal workers.

	workerCount atomic.Uint32 // Current total count of active workers.
	stopOnce    sync.Once     // Ensures Stop logic runs only once.
	isStopped   atomic.Bool   // Flag to indicate if the pool has been stopped.
}

// NewDynamicThreadPool creates a new dynamic thread pool with separate limits.
// maxPriorityWorkers: Max concurrent goroutines processing priority tasks. Must be > 0.
// maxNormalWorkers: Max concurrent goroutines processing normal tasks. Must be > 0.
func NewDynamicThreadPool(maxPriorityWorkers, maxNormalWorkers uint32) *DynamicThreadPool {
	if maxPriorityWorkers == 0 {
		log.Println("DynamicThreadPool: maxPriorityWorkers cannot be zero")
		return nil
	}
	if maxNormalWorkers == 0 {
		log.Println("DynamicThreadPool: maxNormalWorkers cannot be zero")
		return nil
	}

	log.Printf("DynamicThreadPool: Creating with maxPriorityWorkers: %d, maxNormalWorkers: %d\n",
		maxPriorityWorkers, maxNormalWorkers)

	return &DynamicThreadPool{
		maxPriorityWorkers: maxPriorityWorkers,
		maxNormalWorkers:   maxNormalWorkers,
		// Buffer channels appropriately. Sizes are examples.
		priorityCh:  make(chan Task, maxPriorityWorkers*2), // Example buffer size
		normalCh:    make(chan Task, maxNormalWorkers*10),  // Example buffer size
		closeCh:     make(chan struct{}),
		prioritySem: make(chan struct{}, maxPriorityWorkers), // Semaphore for priority tasks
		normalSem:   make(chan struct{}, maxNormalWorkers),   // Semaphore for normal tasks
	}
}

// Start prepares the pool to accept tasks. No workers are started initially.
func (t *DynamicThreadPool) Start() {
	log.Println("DynamicThreadPool: Started. Workers will be created per task.")
}

// Schedule adds a task to the appropriate queue and attempts to launch
// a corresponding worker if the concurrency limit for that type allows.
// Returns false if the pool is stopped, true otherwise.
func (t *DynamicThreadPool) Schedule(urgent bool, item Task) bool {
	if t.isStopped.Load() {
		// log.Println("DynamicThreadPool: Cannot schedule task on stopped pool") // Optional: Reduce log noise
		return false
	}

	if urgent {
		// Try to queue priority task
		select {
		case t.priorityCh <- item:
			t.tryLaunchPriorityWorker() // Attempt to launch a PRIORITY worker
			return true
		case <-t.closeCh:
			log.Println("DynamicThreadPool: Pool stopped while trying to schedule priority task")
			return false
		}
	} else {
		// Try to queue normal task
		select {
		case t.normalCh <- item:
			t.tryLaunchNormalWorker() // Attempt to launch a NORMAL worker
			return true
		case <-t.closeCh:
			log.Println("DynamicThreadPool: Pool stopped while trying to schedule normal task")
			return false
		}
	}
}

// tryLaunchPriorityWorker attempts to acquire the priority semaphore and start a priority worker.
func (t *DynamicThreadPool) tryLaunchPriorityWorker() {
	if t.isStopped.Load() { // Check if stopped before trying to launch
		return
	}

	// Acquired priority semaphore, start a new priority worker goroutine
	t.prioritySem <- struct{}{}
	t.workerCount.Add(1)
	t.wg.Add(1)
	go t.priorityWorkerTask()
	log.Printf("DynamicThreadPool: Launched priority worker. Active count: %d\n", t.workerCount.Load())
}

// tryLaunchNormalWorker attempts to acquire the normal semaphore and start a normal worker.
func (t *DynamicThreadPool) tryLaunchNormalWorker() {
	if t.isStopped.Load() { // Check if stopped before trying to launch
		return
	}
	// Acquired normal semaphore, start a new normal worker goroutine
	t.normalSem <- struct{}{}
	t.workerCount.Add(1)
	t.wg.Add(1)
	go t.normalWorkerTask()
	log.Printf("DynamicThreadPool: Launched normal worker. Active count: %d\n", t.workerCount.Load())
}

// priorityWorkerTask fetches and executes exactly one task from the priority queue.
func (t *DynamicThreadPool) priorityWorkerTask() {
	// Ensure semaphore is released, WG is decremented, and count updated when done.
	defer func() {
		<-t.prioritySem               // Release PRIORITY semaphore slot
		t.workerCount.Add(^uint32(0)) // Decrement total worker count
		t.wg.Done()
		log.Printf("DynamicThreadPool: Priority worker finished. Active count: %d\n", t.workerCount.Load())
	}()

	// This worker tries to grab exactly one priority task.
	select {
	case <-t.closeCh: // Highest priority: Shutdown signal
		// log.Println("DynamicThreadPool: Priority worker received stop signal before processing task.")
		return // Exit immediately

	case task, ok := <-t.priorityCh: // Read ONLY from priority channel
		if !ok {
			// log.Println("DynamicThreadPool: Priority channel closed while priority worker waiting, exiting.")
			return // Channel closed
		}
		task.Execute()
		return // Worker terminates after executing one task
	}
}

// normalWorkerTask fetches and executes exactly one task from the normal queue.
func (t *DynamicThreadPool) normalWorkerTask() {
	// Ensure semaphore is released, WG is decremented, and count updated when done.
	defer func() {
		<-t.normalSem                 // Release NORMAL semaphore slot
		t.workerCount.Add(^uint32(0)) // Decrement total worker count
		t.wg.Done()
		log.Printf("DynamicThreadPool: Normal worker finished. Active count: %d\n", t.workerCount.Load())
	}()

	// This worker tries to grab exactly one normal task.
	select {
	case <-t.closeCh: // Highest priority: Shutdown signal
		// log.Println("DynamicThreadPool: Normal worker received stop signal before processing task.")
		return // Exit immediately
	case task, ok := <-t.normalCh: // Read ONLY from normal channel
		if !ok {
			// log.Println("DynamicThreadPool: Normal channel closed while normal worker waiting, exiting.")
			return // Channel closed
		}
		task.Execute()
		return // Worker terminates after executing one task
	}
}

// Stop signals workers to terminate and waits for currently executing workers to finish.
func (t *DynamicThreadPool) Stop() {
	t.stopOnce.Do(func() {
		log.Println("DynamicThreadPool: Stopping...")
		t.isStopped.Store(true) // Mark as stopped first

		// Close closeCh to signal any workers currently blocked waiting for tasks.
		close(t.closeCh)

		// Wait for all worker goroutines currently executing tasks to finish
		t.wg.Wait()

		log.Println("DynamicThreadPool: All active workers stopped.")

		// Close task channels safely after workers are done
		close(t.priorityCh)
		close(t.normalCh)

		// Close semaphore channels
		close(t.prioritySem)
		close(t.normalSem)

		log.Println("DynamicThreadPool: Pool stopped completely.")
	})
}

// GetActiveWorkers returns the current total number of worker goroutines executing tasks.
func (t *DynamicThreadPool) GetActiveWorkers() uint32 {
	return t.workerCount.Load()
}

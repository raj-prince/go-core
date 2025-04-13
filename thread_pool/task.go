package thread_pool

import (
	"math/rand"
	"time"
)

// Task is an interface for tasks to be executed by the thread pool.
type Task interface {
	Execute()
}

// PrefetchTask is a concrete implementation of the Task interface.
type PrefetchTask struct {
	failCnt int32
}

// Execute implements the Task interface for PrefetchTask.
func (t PrefetchTask) Execute() {
	// Simulate some work.
	time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
}

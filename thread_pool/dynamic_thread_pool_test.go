// dynamic_thread_pool_test.go
package thread_pool

import (
	"fmt"
	"log"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// --- Test Task Implementation ---

// mockTask is a simple task for testing that increments a counter.
type mockTask struct {
	id          int
	counter     *atomic.Int32 // Pointer to a shared counter
	workTime    time.Duration // Simulate work
	panicOnExec bool          // Flag to cause panic during execution
}

func (m *mockTask) Execute() {
	if m.panicOnExec {
		panic(fmt.Sprintf("mockTask %d panicking as requested", m.id))
	}
	if m.workTime > 0 {
		time.Sleep(m.workTime)
	}
	if m.counter != nil {
		m.counter.Add(1)
	}
	// log.Printf("Task %d executed", m.id) // Optional: for debugging
}

// --- Test Suite Setup ---

type DynamicThreadPoolTestSuite struct {
	suite.Suite
	assert *assert.Assertions
}

func (suite *DynamicThreadPoolTestSuite) SetupTest() {
	suite.assert = assert.New(suite.T())
	// Disable log output during tests unless debugging
	// log.SetOutput(io.Discard)
}

func (suite *DynamicThreadPoolTestSuite) TearDownTest() {
	// Re-enable log output if needed
	// log.SetOutput(os.Stderr)
}

// --- Test Cases ---

func (suite *DynamicThreadPoolTestSuite) TestCreate() {
	tpNil1 := NewDynamicThreadPool(0, 5)
	suite.assert.Nil(tpNil1, "Pool should be nil if maxPriorityWorkers is zero")

	tpNil2 := NewDynamicThreadPool(5, 0)
	suite.assert.Nil(tpNil2, "Pool should be nil if maxNormalWorkers is zero")

	tp := NewDynamicThreadPool(2, 3)
	suite.assert.NotNil(tp, "Pool should not be nil with valid inputs")
	suite.assert.Equal(uint32(2), tp.maxPriorityWorkers)
	suite.assert.Equal(uint32(3), tp.maxNormalWorkers)
	suite.assert.NotNil(tp.priorityCh)
	suite.assert.NotNil(tp.normalCh)
	suite.assert.NotNil(tp.closeCh)
	suite.assert.NotNil(tp.prioritySem)
	suite.assert.NotNil(tp.normalSem)
	suite.assert.False(tp.isStopped.Load(), "Pool should not be stopped initially")
}

func (suite *DynamicThreadPoolTestSuite) TestStartStop() {
	tp := NewDynamicThreadPool(1, 1)
	suite.assert.NotNil(tp)

	tp.Start() // Start is currently a no-op, but call it for completeness

	// Stop should work even if no tasks were scheduled
	tp.Stop()
	suite.assert.True(tp.isStopped.Load(), "Pool should be stopped after Stop()")

	// Calling Stop again should be safe (idempotent)
	tp.Stop()
	suite.assert.True(tp.isStopped.Load(), "Pool should remain stopped")
}

func (suite *DynamicThreadPoolTestSuite) TestScheduleSimple() {
	tp := NewDynamicThreadPool(1, 1)
	suite.assert.NotNil(tp)
	tp.Start()

	var counter atomic.Int32
	task1 := &mockTask{id: 1, counter: &counter}
	task2 := &mockTask{id: 2, counter: &counter}

	scheduled1 := tp.Schedule(true, task1)  // Priority
	scheduled2 := tp.Schedule(false, task2) // Normal

	suite.assert.True(scheduled1, "Scheduling priority task should succeed")
	suite.assert.True(scheduled2, "Scheduling normal task should succeed")

	// Reliable wait for tasks to complete
	suite.waitForCounter(2, &counter, 3*time.Second)

	tp.Stop()

	suite.assert.Equal(int32(2), counter.Load(), "Both tasks should have executed")
}

func (suite *DynamicThreadPoolTestSuite) TestScheduleAfterStop() {
	tp := NewDynamicThreadPool(1, 1)
	suite.assert.NotNil(tp)
	tp.Start()
	tp.Stop() // Stop the pool first

	suite.assert.True(tp.isStopped.Load(), "Pool should be marked as stopped")

	var counter atomic.Int32
	task1 := &mockTask{id: 1, counter: &counter}

	scheduled := tp.Schedule(true, task1)

	suite.assert.False(scheduled, "Scheduling should fail on a stopped pool")
	// Give a tiny moment to ensure no worker could have possibly started
	time.Sleep(50 * time.Millisecond)
	suite.assert.Equal(int32(0), counter.Load(), "Task should not execute on stopped pool")
}

func (suite *DynamicThreadPoolTestSuite) TestManyTasks() {
	maxPri := uint32(5)
	maxNorm := uint32(10)
	tp := NewDynamicThreadPool(maxPri, maxNorm)
	suite.assert.NotNil(tp)
	tp.Start()

	totalTasks := 200
	var counter atomic.Int32

	for i := 0; i < totalTasks; i++ {
		task := &mockTask{
			id:       i,
			counter:  &counter,
			workTime: time.Duration(i%5) * time.Millisecond, // Vary work time slightly
		}
		// Mix priority and normal tasks
		isUrgent := i%3 == 0
		scheduled := tp.Schedule(isUrgent, task)
		suite.assert.True(scheduled, "Scheduling task %d should succeed", i)
	}

	// Wait for all tasks
	suite.waitForCounter(int32(totalTasks), &counter, 10*time.Second) // Generous timeout

	tp.Stop()

	suite.assert.Equal(int32(totalTasks), counter.Load(), "All scheduled tasks should have executed")
	suite.assert.Equal(uint32(0), tp.GetActiveWorkers(), "Should have 0 active workers after stop and wait")
}

func (suite *DynamicThreadPoolTestSuite) TestConcurrentScheduling() {
	maxPri := uint32(10)
	maxNorm := uint32(20)
	tp := NewDynamicThreadPool(maxPri, maxNorm)
	suite.assert.NotNil(tp)
	tp.Start()

	numGoroutines := 50
	tasksPerGoroutine := 10
	totalTasks := numGoroutines * tasksPerGoroutine
	var counter atomic.Int32
	var scheduleWg sync.WaitGroup

	scheduleWg.Add(numGoroutines)
	for g := 0; g < numGoroutines; g++ {
		go func(goroutineID int) {
			defer scheduleWg.Done()
			for i := 0; i < tasksPerGoroutine; i++ {
				taskID := goroutineID*tasksPerGoroutine + i
				task := &mockTask{
					id:       taskID,
					counter:  &counter,
					workTime: time.Duration(taskID%3) * time.Millisecond,
				}
				isUrgent := taskID%4 == 0 // Mix priorities
				scheduled := tp.Schedule(isUrgent, task)
				// In high contention, scheduling might fail if the pool stops *during* the test
				// but ideally, it should succeed until Stop() is called.
				// We check the final count instead of asserting every schedule call here.
				if !scheduled && !tp.isStopped.Load() {
					log.Printf("Warning: Scheduling failed unexpectedly for task %d", taskID)
				}
				runtime.Gosched() // Yield to allow other goroutines to schedule
			}
		}(g)
	}

	scheduleWg.Wait() // Wait for all goroutines to finish scheduling

	// Wait for all tasks to complete execution
	suite.waitForCounter(int32(totalTasks), &counter, 15*time.Second) // Increased timeout for concurrency

	tp.Stop()

	suite.assert.Equal(int32(totalTasks), counter.Load(), "All concurrently scheduled tasks should have executed")
	suite.assert.Equal(uint32(0), tp.GetActiveWorkers(), "Should have 0 active workers after stop and wait")
}

func (suite *DynamicThreadPoolTestSuite) TestWorkerCountLifecycle() {
	// Use small limits to observe behavior easily
	tp := NewDynamicThreadPool(1, 1)
	suite.assert.NotNil(tp)
	tp.Start()

	var counter atomic.Int32

	suite.assert.Equal(uint32(0), tp.GetActiveWorkers(), "Should start with 0 active workers")

	// Schedule one task
	task1 := &mockTask{id: 1, counter: &counter, workTime: 50 * time.Millisecond}
	tp.Schedule(false, task1)

	// Wait briefly, worker should be active
	time.Sleep(10 * time.Millisecond)
	suite.assert.Equal(uint32(1), tp.GetActiveWorkers(), "Should have 1 active worker after scheduling")

	// Wait for task to finish
	suite.waitForCounter(1, &counter, 1*time.Second)
	// Wait a bit longer for the worker goroutine to fully exit and release semaphore
	time.Sleep(50 * time.Millisecond)
	suite.assert.Equal(uint32(0), tp.GetActiveWorkers(), "Should have 0 active workers after task completion")

	// Schedule two tasks (one priority, one normal) - should use both semaphores
	task2 := &mockTask{id: 2, counter: &counter, workTime: 50 * time.Millisecond}
	task3 := &mockTask{id: 3, counter: &counter, workTime: 50 * time.Millisecond}
	tp.Schedule(true, task2)
	tp.Schedule(false, task3)

	// Wait briefly, both workers should be active
	time.Sleep(10 * time.Millisecond)
	suite.assert.Equal(uint32(2), tp.GetActiveWorkers(), "Should have 2 active workers for separate types")

	// Wait for both tasks
	suite.waitForCounter(3, &counter, 1*time.Second) // Counter is now 3
	time.Sleep(50 * time.Millisecond)
	suite.assert.Equal(uint32(0), tp.GetActiveWorkers(), "Should have 0 active workers after both tasks complete")

	tp.Stop()
}

// --- Helper Methods ---

// waitForCounter polls an atomic counter until it reaches the target value or times out.
func (suite *DynamicThreadPoolTestSuite) waitForCounter(target int32, counter *atomic.Int32, timeout time.Duration) {
	startTime := time.Now()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		currentVal := counter.Load()
		if currentVal >= target { // Use >= in case of over-increment (shouldn't happen here)
			return
		}
		if time.Since(startTime) > timeout {
			suite.assert.Failf("Timeout", "Timed out waiting for counter. Expected: %d, Got: %d", target, currentVal)
			return // Exit loop on timeout
		}
		<-ticker.C // Wait for the next tick
	}
}

// --- Test Runner ---

func TestDynamicThreadPoolSuite(t *testing.T) {
	suite.Run(t, new(DynamicThreadPoolTestSuite))
}

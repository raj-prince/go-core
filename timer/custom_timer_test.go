// custom_timer_test.go
package timer

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// --- Test Suite Setup ---

type CustomTimerTestSuite struct {
	suite.Suite
	assert *assert.Assertions
}

func (suite *CustomTimerTestSuite) SetupTest() {
	suite.assert = assert.New(suite.T())
}

// --- Test Cases ---

func (suite *CustomTimerTestSuite) TestNewCustomTimer() {
	duration := 5 * time.Second
	callbackExecuted := false
	cb := func() { callbackExecuted = true }

	ct := NewCustomTimer(duration, cb)

	suite.assert.NotNil(ct)
	suite.assert.Equal(duration, ct.duration)
	suite.assert.Nil(ct.timer, "Timer should be nil initially")
	suite.assert.False(ct.paused, "Should not be paused initially")
	suite.assert.NotNil(ct.callback, "Callback should be set")

	// Check if callback works (though it shouldn't be called yet)
	ct.callback()
	suite.assert.True(callbackExecuted, "Callback function should be callable")
}

func (suite *CustomTimerTestSuite) TestStartAndFire() {
	duration := 50 * time.Millisecond
	var callbackCount atomic.Int32
	cb := func() {
		callbackCount.Add(1)
	}

	ct := NewCustomTimer(duration, cb)
	ct.Start()
	suite.assert.NotNil(ct.timer, "Timer should be active after Start")

	// Wait longer than the duration
	time.Sleep(duration * 2)

	suite.assert.Equal(int32(1), callbackCount.Load(), "Callback should have been called exactly once")

	// Calling start again should have no effect
	ct.Start()
	time.Sleep(duration * 2)
	suite.assert.Equal(int32(1), callbackCount.Load(), "Callback should not be called again on second Start")
}

func (suite *CustomTimerTestSuite) TestPauseBeforeFire() {
	duration := 100 * time.Millisecond
	var callbackCount atomic.Int32
	cb := func() {
		callbackCount.Add(1)
	}

	ct := NewCustomTimer(duration, cb)
	ct.Start()
	suite.assert.NotNil(ct.timer)
	suite.assert.False(ct.paused)

	// Pause shortly after starting
	time.Sleep(duration / 4)
	ct.Pause()
	suite.assert.True(ct.paused, "Timer should be paused")

	// Wait longer than the original duration
	time.Sleep(duration * 2)

	suite.assert.Equal(int32(0), callbackCount.Load(), "Callback should not have been called after pause")

	// Calling pause again should do nothing
	ct.Pause()
	suite.assert.True(ct.paused, "Timer should remain paused")
}

func (suite *CustomTimerTestSuite) TestPauseUnstarted() {
	duration := 50 * time.Millisecond
	var callbackCount atomic.Int32
	cb := func() { callbackCount.Add(1) }

	ct := NewCustomTimer(duration, cb)
	ct.Pause() // Pause before starting

	suite.assert.Nil(ct.timer, "Timer should still be nil")
	suite.assert.False(ct.paused, "Paused flag should not be set if timer wasn't running")

	ct.Start() // Now start it
	time.Sleep(duration * 2)
	suite.assert.Equal(int32(1), callbackCount.Load(), "Callback should fire normally if pause was called before start")
}

func (suite *CustomTimerTestSuite) TestResume() {
	duration := 100 * time.Millisecond
	pauseTime := duration / 4  // Pause after 25ms
	resumeWait := duration / 3 // Wait another 25ms after resuming

	var callbackCount atomic.Int32
	callbackCh := make(chan bool, 1) // Use channel for signaling
	cb := func() {
		callbackCount.Add(1)
		callbackCh <- true
	}

	ct := NewCustomTimer(duration, cb)
	ct.Start()

	// Pause part way through
	time.Sleep(pauseTime)
	ct.Pause()
	suite.assert.True(ct.paused)

	// Wait a bit while paused - this time shouldn't count
	time.Sleep(100 * time.Millisecond)
	suite.assert.Equal(int32(0), callbackCount.Load(), "Callback shouldn't fire while paused")

	// Resume
	ct.Resume()
	suite.assert.False(ct.paused, "Timer should not be paused after resume")
	suite.assert.NotNil(ct.timer, "Timer should be active after resume")

	// Check it doesn't fire too early (e.g., immediately after resume)
	time.Sleep(resumeWait) // Wait less than remaining time
	suite.assert.Equal(int32(0), callbackCount.Load(), "Callback shouldn't fire before remaining duration passes")

	// Wait for callback using channel with timeout
	select {
	case <-callbackCh:
		// Callback fired as expected
		suite.assert.Equal(int32(1), callbackCount.Load(), "Callback count should be 1 after resume and wait")
	case <-time.After(duration): // Wait remaining duration + buffer
		suite.assert.Fail("Timeout waiting for callback after resume")
	}
}

func (suite *CustomTimerTestSuite) TestResumeUnpaused() {
	duration := 50 * time.Millisecond
	var callbackCount atomic.Int32
	cb := func() { callbackCount.Add(1) }

	ct := NewCustomTimer(duration, cb)
	ct.Start()
	time.Sleep(duration / 2)

	ct.Resume() // Call resume when not paused
	suite.assert.False(ct.paused, "Timer should remain unpaused")

	// Wait for original duration to pass
	time.Sleep(duration)
	suite.assert.Equal(int32(1), callbackCount.Load(), "Callback should fire normally if resume called when not paused")
}

func (suite *CustomTimerTestSuite) TestResetRunning() {
	duration := 100 * time.Millisecond
	resetTime := duration / 2
	var callbackCount atomic.Int32
	callbackCh := make(chan bool, 1)
	cb := func() {
		callbackCount.Add(1)
		// Use non-blocking send in case test times out before callback
		select {
		case callbackCh <- true:
		default:
		}
	}

	ct := NewCustomTimer(duration, cb)
	ct.Start()

	// Reset part way through
	time.Sleep(resetTime)
	ct.Reset()
	suite.assert.False(ct.paused, "Timer should not be paused after reset")
	suite.assert.NotNil(ct.timer, "Timer should be active after reset")

	// Wait less than the full duration *after* reset
	time.Sleep(duration / 2)
	suite.assert.Equal(int32(0), callbackCount.Load(), "Callback should not fire before full duration after reset")

	// Wait for callback using channel with timeout (duration + buffer from reset time)
	select {
	case <-callbackCh:
		// Callback fired as expected
		suite.assert.Equal(int32(1), callbackCount.Load(), "Callback count should be 1 after reset and wait")
	case <-time.After(duration * 2): // Generous timeout
		suite.assert.Fail("Timeout waiting for callback after reset")
	}
}

func (suite *CustomTimerTestSuite) TestResetPaused() {
	duration := 100 * time.Millisecond
	pauseTime := duration / 3
	var callbackCount atomic.Int32
	callbackCh := make(chan bool, 1)
	cb := func() {
		callbackCount.Add(1)
		select {
		case callbackCh <- true:
		default:
		}
	}

	ct := NewCustomTimer(duration, cb)
	ct.Start()
	time.Sleep(pauseTime)
	ct.Pause()
	suite.assert.True(ct.paused)

	// Reset while paused
	ct.Reset()
	suite.assert.False(ct.paused, "Timer should not be paused after reset")
	suite.assert.NotNil(ct.timer, "Timer should be active after reset")

	// Wait less than the full duration *after* reset
	time.Sleep(duration / 2)
	suite.assert.Equal(int32(0), callbackCount.Load(), "Callback should not fire before full duration after reset")

	// Wait for callback using channel with timeout
	select {
	case <-callbackCh:
		// Callback fired as expected
		suite.assert.Equal(int32(1), callbackCount.Load(), "Callback count should be 1 after reset and wait")
	case <-time.After(duration * 2): // Generous timeout
		suite.assert.Fail("Timeout waiting for callback after reset")
	}
}

// --- Test Runner ---

func TestCustomTimerSuite(t *testing.T) {
	suite.Run(t, new(CustomTimerTestSuite))
}

// Note: The current implementation of CustomTimer might have race conditions
// if methods (Pause, Resume, Reset, Start) are called concurrently from multiple
// goroutines, as fields like `timer`, `paused`, and `lastStartTime` are accessed
// without mutex protection. These tests primarily check sequential logic.
// Fully testing concurrent safety would require more complex test setups or
// modifications to CustomTimer to add locking.

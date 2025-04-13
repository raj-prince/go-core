package timer

import (
	"time"
)

// CustomTimer represents a custom timer with pause/resume functionality.
type CustomTimer struct {
	duration      time.Duration
	timer         *time.Timer
	callback      func()
	paused        bool
	lastStartTime time.Time
	activeElapsed time.Duration
}

// NewCustomTimer creates a new CustomTimer.
func NewCustomTimer(duration time.Duration, callback func()) *CustomTimer {
	return &CustomTimer{
		duration: duration,
		callback: callback,
	}
}

// Start starts the timer.
func (t *CustomTimer) Start() {
	if t.timer == nil && !t.paused {
		t.lastStartTime = time.Now()
		t.timer = time.NewTimer(t.duration)
		go t.run()
	}
}

// Pause pauses the timer.
func (t *CustomTimer) Pause() {
	if t.timer != nil {
		if !t.paused {
			t.timer.Stop()
			t.activeElapsed += time.Since(t.lastStartTime)
			t.paused = true
		}
	}
}

// Resume resumes the timer.
func (t *CustomTimer) Resume() {
	if t.paused {
		t.paused = false
		remainingDuration := t.duration - t.activeElapsed
		if remainingDuration > 0 {
			t.timer = time.NewTimer(remainingDuration)
			t.lastStartTime = time.Now()
			go t.run()
		} else {
			t.callback()
		}
	}
}

// Reset resets the timer.
func (t *CustomTimer) Reset() {
	if t.timer != nil {
		t.timer.Stop()
	}
	t.paused = false
	t.lastStartTime = time.Now()
	t.timer = time.NewTimer(t.duration)
	go t.run()
}

// run is a helper function that waits for the timer to expire and calls the callback.
func (t *CustomTimer) run() {
	<-t.timer.C
	t.callback()
}

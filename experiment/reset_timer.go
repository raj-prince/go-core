package main

import (
	"fmt"
	"time"
)

func main() {
	var (
		count int
		timer *time.Timer // Declare timer as a pointer
	)

	discardOnce := false

	// Using AfterFunc
	timer = time.AfterFunc(time.Millisecond, func() {
		if discardOnce {
			discardOnce = false
			return
		}
		time.Sleep(time.Second)
		count++
		fmt.Println("Timer fired:", count)
	})

	time.Sleep(time.Millisecond * 2)

	// Reset the timer immediately, regardless of its current state.
	ok := timer.Reset(time.Millisecond)
	if !ok {
		discardOnce = true
	}

	time.Sleep(time.Second * 3)
	timer.Stop()
	fmt.Println("timer stopped")
}

/**
Command: go run reset_timer.go

Output:
Timer fired: 1
Timer fired: 2
Timer stopped

Conclusion: a new timer callback is started if timer.Reset returns false.


If you uncomment the line#32: discardOnce = true

Output:
Timer fired: 1
Timer stopped.
*/

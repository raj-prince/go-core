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

	// Using AfterFunc
	timer = time.AfterFunc(time.Millisecond, func() {
		time.Sleep(time.Second)
		count++
		fmt.Println("Timer fired:", count)
	})

	time.Sleep(time.Millisecond * 2)

	// Reset the timer immediately, regardless of its current state.
	ok := timer.Reset(time.Millisecond)
	fmt.Println("timer reset returns: ", ok)

	time.Sleep(time.Second * 3)
	timer.Stop()
	fmt.Println("timer stopped")
}

/**
Command: go run reset_timer.go

Output:
timer reset returns:  false
Timer fired: 1
Timer fired: 2
timer stopped

Conclusion: a new timer callback is started if timer.Reset returns false.
*/

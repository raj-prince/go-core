package main_test

import (
	"sync"
	"testing"
	"time"
)

func BenchmarkNewTimer(b *testing.B) {
	timer := time.NewTimer(time.Hour)
	defer timer.Stop()
	for i := 0; i < b.N; i++ {
		timer = time.NewTimer(time.Hour)
	}
}

func BenchmarkResetTimer(b *testing.B) {
	timer := time.NewTimer(time.Hour)
	defer timer.Stop()
	for i := 0; i < b.N; i++ {
		timer.Reset(time.Second)
	}
}

func BenchmarkUpdateTime(b *testing.B) {
	var lastTime time.Time
	defer func() {
		ok := false
		if ok {
			b.Log(lastTime)
		}
	}()
	for i := 0; i < b.N; i++ {
		lastTime = time.Now()
	}
}

func BenchmarkChannelFromTime(b *testing.B) {
	var chanTime <-chan time.Time
	// Create a timer channel
	defer func() {
		ok := false
		if ok {
			b.Log(chanTime)
		}
	}()
	for i := 0; i < b.N; i++ {
		chanTime = time.After(time.Nanosecond)
		// <-chanTime
	}
}

func BenchmarkTimerStop(b *testing.B) {
	timer := time.NewTimer(time.Hour)
	for i := 0; i < b.N; i++ {
		timer.Stop()
	}
}

func BenchmarkMutexLockUnlock(b *testing.B) {
	mutex := new(sync.Mutex)
	go func() { // Making the scenarion realistic by sharing the mutext with other routine
		mutex.Lock()
		defer mutex.Unlock()
		time.Sleep(time.Millisecond)
	}()

	for i := 0; i < b.N; i++ {
		mutex.Lock()
		mutex.Unlock()
	}
}

/**
Command: go test -bench . -test.benchmem

Output:
goos: linux
goarch: amd64
pkg: go-core/experiment
cpu: Intel(R) Xeon(R) CPU @ 2.60GHz
BenchmarkNewTimer-96                     4462598               268.0 ns/op           248 B/op          3 allocs/op
BenchmarkResetTimer-96                  18103489                65.78 ns/op            0 B/op          0 allocs/op
BenchmarkUpdateTime-96                  31013109                38.68 ns/op            0 B/op          0 allocs/op
BenchmarkChannelFromTime-96              4458170               270.9 ns/op           248 B/op          3 allocs/op
BenchmarkTimerStop-96                   29053162                41.45 ns/op            0 B/op          0 allocs/op
BenchmarkMutexLockUnlock-96             86282263                12.69 ns/op            0 B/op          0 allocs/op
PASS
*/

package main_test

import (
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
PASS
*/

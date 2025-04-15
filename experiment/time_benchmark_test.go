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

/**
Command: go test -bench . -test.benchmem

Output:
goos: linux
goarch: amd64
pkg: go-core/experiment
cpu: Intel(R) Xeon(R) CPU @ 2.60GHz
BenchmarkNewTimer-96             6392534               188.3 ns/op           248 B/op          3 allocs/op
BenchmarkResetTimer-96          18149168                65.89 ns/op            0 B/op          0 allocs/op
BenchmarkUpdateTime-96          30910446                38.70 ns/op            0 B/op          0 allocs/op
PASS
ok      go-core/experiment      3.903s
*/

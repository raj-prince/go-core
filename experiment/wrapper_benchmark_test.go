package main_test

import (
	"testing"
)

type runner interface {
	Increment()
}

type simpleRunner struct {
	val int64
}

func (sr *simpleRunner) Increment() {
	sr.val++
}

type wrapperRunner struct {
	wrapped runner
}

func (wr *wrapperRunner) Increment() {
	wr.wrapped.Increment()
}

func createSimpleRunner() runner {
	return &simpleRunner{val: 0}
}

func createWrapperRunner(wrapped runner) runner {
	return &wrapperRunner{wrapped: wrapped}
}

func BenchmarkWrapperIncrement(b *testing.B) {
	runner := createWrapperRunner(createSimpleRunner())
	for i := 0; i < b.N; i++ {
		runner.Increment()
	}
}

func BenchmarkMethodIncrement(b *testing.B) {
	runner := createSimpleRunner()
	for i := 0; i < b.N; i++ {
		runner.Increment()
	}
}

func BenchmarkSimpleIncrement(b *testing.B) {
	val := int64(0)
	for i := 0; i < b.N; i++ {
		val++
	}
}

/**
Command: go test -bench . -test.benchmem

BenchmarkWrapperIncrement-96                                    587946856                2.102 ns/op           0 B/op          0 allocs/op
BenchmarkMethodIncrement-96                                     608669430                2.008 ns/op           0 B/op          0 allocs/op
BenchmarkSimpleIncrement-96                                     1000000000               0.2955 ns/op          0 B/op          0 allocs/op
*/

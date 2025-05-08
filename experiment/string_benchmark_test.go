package main

import (
	"strings"
	"testing"
)

func BenchmarkStringConcatenation(b *testing.B) {
	var result string
	for i := 0; i < b.N; i++ {
		result = "hell0000rjhjghjkhgdfffffffkjhjkhkjhfdkjhgkjhfgo" + " " + "world"
	}
	_ = result
}

func BenchmarkStringBuilder(b *testing.B) {
	var result string
	for i := 0; i < b.N; i++ {
		var sb strings.Builder
		sb.WriteString("hello")
		sb.WriteString(" ")
		sb.WriteString("world")
		result = sb.String()
	}
	_ = result
}

func BenchmarkStringJoin(b *testing.B) {
	var result string
	for i := 0; i < b.N; i++ {
		result = strings.Join([]string{"hello", " ", "world"}, "")
	}
	_ = result
}

func BenchmarkStringSprintf(b *testing.B) {
	var result string
	for i := 0; i < b.N; i++ {
		result = Sprintf("hello %s %s", " ", "world")
	}
	_ = result
}

func Sprintf(format string, a ...any) string {
	return format
}

/**
Output:

goos: linux
goarch: amd64
pkg: go-core/experiment
cpu: Intel(R) Xeon(R) CPU @ 2.60GHz
BenchmarkStringConcatenation-96                                 1000000000               0.2956 ns/op          0 B/op          0 allocs/op
BenchmarkStringBuilder-96                                       24481148                48.05 ns/op           24 B/op          2 allocs/op
BenchmarkStringJoin-96                                          25624930                45.98 ns/op           16 B/op          1 allocs/op
BenchmarkStringSprintf-96                                       1000000000               0.2955 ns/op          0 B/op          0 allocs/op

*/
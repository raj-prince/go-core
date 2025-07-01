package main_test

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Common fields for a more realistic logging scenario
var (
	testMessage = "This is a test log message."
	testTime    = time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	testInt     = 123
	testString  = "test_string"
)

const readOperationTime = 0 * time.Millisecond

func ReadWithSlog(logger *slog.Logger) {
	testMessage := fmt.Sprintf("%d %s %s", testInt, testString, testMessage)
	logger.Info(testMessage,
		slog.Time("time", testTime),
		slog.Int("int", testInt),
		slog.String("string", testString),
	)
	time.Sleep(readOperationTime)
	logger.Info(testMessage,
		slog.Time("time", testTime),
		slog.Int("int", testInt),
		slog.String("string", testString),
	)
}

// BenchmarkSlog measures the performance of the standard library's slog logger.
// The output is sent to io.Discard to measure the overhead of the logger itself
// (serialization, field handling, etc.) including file I/O latency.
func BenchmarkSlog(b *testing.B) {
	logFile, err := os.CreateTemp(b.TempDir(), "slog-*.log")
	if err != nil {
		b.Fatalf("failed to create temp file: %v", err)
	}
	b.Cleanup(func() { _ = logFile.Close() })

	logger := slog.New(slog.NewJSONHandler(logFile, nil))
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ReadWithSlog(logger)
		}
	})
}

// BenchmarkSlogAsync measures slog performance with an asynchronous writer.
func BenchmarkSlogAsync(b *testing.B) {
	logFile, err := os.CreateTemp(b.TempDir(), "slog-async-*.log")
	if err != nil {
		b.Fatalf("failed to create temp file: %v", err)
	}

	asyncWriter := NewAsyncWriter(logFile, 819200) // 8K buffer
	b.Cleanup(func() { _ = asyncWriter.Close() })

	logger := slog.New(slog.NewJSONHandler(asyncWriter, nil))
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ReadWithSlog(logger)
		}
	})
}

func ReadWithZerolog(logger *zerolog.Logger) {
	testMessage := fmt.Sprintf("%d %s %s", testInt, testString, testMessage)
	logger.Info().
		Time("time", testTime).
		Int("int", testInt).
		Str("string", testString).
		Msg(testMessage)
	time.Sleep(readOperationTime)
	logger.Info().
		Time("time", testTime).
		Int("int", testInt).
		Str("string", testString).
		Msg(testMessage)
}

// BenchmarkZerolog measures the performance of the zerolog logger.
// The output is sent to io.Discard to measure the overhead of the logger itself
// (serialization, field handling, etc.) including file I/O latency.
func BenchmarkZerolog(b *testing.B) {
	logFile, err := os.CreateTemp(b.TempDir(), "zerolog-*.log")
	if err != nil {
		b.Fatalf("failed to create temp file: %v", err)
	}
	b.Cleanup(func() { _ = logFile.Close() })

	logger := zerolog.New(logFile)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ReadWithZerolog(&logger)
		}
	})
}

// BenchmarkZerologAsync measures zerolog performance with an asynchronous writer.
func BenchmarkZerologAsync(b *testing.B) {
	logFile, err := os.CreateTemp(b.TempDir(), "zerolog-async-*.log")
	if err != nil {
		b.Fatalf("failed to create temp file: %v", err)
	}

	asyncWriter := NewAsyncWriter(logFile, 819200) // 8K buffer
	b.Cleanup(func() { _ = asyncWriter.Close() })

	logger := zerolog.New(asyncWriter)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ReadWithZerolog(&logger)
		}
	})
}

func ReadWithZaplog(logger *zap.Logger) {
	testMessage := fmt.Sprintf("%d %s %s", testInt, testString, testMessage)
	logger.Info(testMessage,
		zap.Time("time", testTime),
		zap.Int("int", testInt),
		zap.String("string", testString),
	)
	time.Sleep(readOperationTime)
	logger.Info(testMessage,
		zap.Time("time", testTime),
		zap.Int("int", testInt),
		zap.String("string", testString),
	)
}

// BenchmarkZap measures the performance of the zap logger.
// The output is sent to io.Discard to measure the overhead of the logger itself
// (serialization, field handling, etc.) including file I/O latency.
func BenchmarkZap(b *testing.B) {
	logFile, err := os.CreateTemp(b.TempDir(), "zap-*.log")
	if err != nil {
		b.Fatalf("failed to create temp file: %v", err)
	}
	b.Cleanup(func() { _ = logFile.Close() })

	// Using a direct core configuration is a common way to set up zap
	// for performance-critical applications.
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(logFile),
		zap.InfoLevel,
	)
	logger := zap.New(core)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ReadWithZaplog(logger)
		}
	})
}

// BenchmarkZapAsync measures zap performance with an asynchronous writer.
func BenchmarkZapAsync(b *testing.B) {
	logFile, err := os.CreateTemp(b.TempDir(), "zap-async-*.log")
	if err != nil {
		b.Fatalf("failed to create temp file: %v", err)
	}

	asyncWriter := NewAsyncWriter(logFile, 819200)
	b.Cleanup(func() { _ = asyncWriter.Close() })

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(asyncWriter),
		zap.InfoLevel,
	)
	logger := zap.New(core)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ReadWithZaplog(logger)
		}
	})
}

// AsyncWriter provides an asynchronous, buffered writer.
// It wraps an io.Writer and performs write operations in a separate goroutine.
type AsyncWriter struct {
	writer    io.Writer
	ch        chan []byte
	wg        sync.WaitGroup
	closeOnce sync.Once
	closed    chan struct{}
}

// NewAsyncWriter creates and starts a new AsyncWriter.
// It takes an underlying io.Writer to write to and a bufferSize for the
// internal channel.
func NewAsyncWriter(w io.Writer, bufferSize int) *AsyncWriter {
	if bufferSize <= 0 {
		bufferSize = 1024 // Default buffer size
	}
	aw := &AsyncWriter{
		writer: w,
		ch:     make(chan []byte, bufferSize),
		closed: make(chan struct{}),
	}
	aw.wg.Add(1)
	go aw.run()
	return aw
}

// run is the background worker goroutine that reads from the channel and
// writes to the underlying writer.
func (aw *AsyncWriter) run() {
	defer aw.wg.Done()
	for data := range aw.ch {
		if _, err := aw.writer.Write(data); err != nil {
			// In a real-world scenario, you might want a more robust error handling strategy.
			fmt.Fprintf(os.Stderr, "AsyncWriter: write error: %v\n", err)
		}
	}
}

// Write sends data to the writer's buffer. It is non-blocking unless the
// buffer is full. It makes a copy of the provided byte slice, so the caller
// is free to reuse the original slice.
func (aw *AsyncWriter) Write(p []byte) (int, error) {
	select {
	case <-aw.closed:
		return 0, io.ErrClosedPipe
	default:
	}

	// Make a copy of the data, as the caller might reuse the buffer p.
	data := make([]byte, len(p))
	copy(data, p)

	select {
	case aw.ch <- data:
		return len(p), nil
	case <-aw.closed:
		return 0, io.ErrClosedPipe
	}
}

// Close flushes any buffered data to the underlying writer, waits for the
// writer goroutine to exit, and closes the underlying writer if it
// implements io.Closer.
func (aw *AsyncWriter) Close() error {
	aw.closeOnce.Do(func() {
		close(aw.closed)
		close(aw.ch)
	})

	aw.wg.Wait()

	if closer, ok := aw.writer.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

// goos: linux
// goarch: amd64
// pkg: go-core/experiment
// cpu: INTEL(R) XEON(R) PLATINUM 8581C CPU @ 2.30GHz
// BenchmarkSlog-192             	  317107	      3560 ns/op	     375 B/op	       9 allocs/op
// BenchmarkSlogAsync-192        	  370838	      2987 ns/op	     722 B/op	      11 allocs/op
// BenchmarkZerolog-192          	  306411	      3858 ns/op	      81 B/op	       3 allocs/op
// BenchmarkZerologAsync-192     	  597055	      2221 ns/op	     369 B/op	       5 allocs/op
// BenchmarkZap-192              	  233199	      4719 ns/op	     477 B/op	       5 allocs/op
// BenchmarkZapAsync-192         	  484533	      2331 ns/op	     756 B/op	       7 allocs/op
// PASS
// ok  	go-core/experiment	8.660s

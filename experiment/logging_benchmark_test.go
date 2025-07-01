package main_test

import (
	"io"
	"log/slog"
	"os"
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

const readOperationTime = time.Millisecond

func ReadWithSlog(logger *slog.Logger) {
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

	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ReadWithSlog(logger)
		}
	})
}

func ReadWithZerolog(logger *zerolog.Logger) {
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

	logger := zerolog.New(io.Discard)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ReadWithZerolog(&logger)
		}
	})
}

func ReadWithZaplog(logger *zap.Logger) {
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
		zapcore.AddSync(io.Discard),
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

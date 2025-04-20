package main_test

import (
	"fmt"
	"syscall"
	"testing"
)

const (
	MIB = 1024 * 1024
)

type Block struct {
	data      []byte
	writeSeek uint64
}

func AllocateBlockWithMmap(size uint64) (*Block, error) {
	if size == 0 {
		return nil, fmt.Errorf("invalid size")
	}

	prot, flags := syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_ANON|syscall.MAP_PRIVATE
	addr, err := syscall.Mmap(-1, 0, int(size), prot, flags)

	if err != nil {
		return nil, fmt.Errorf("mmap error: %v", err)
	}

	block := &Block{
		data: addr,
	}

	// we do not create channel here, as that will be created when buffer is retrieved
	// reinit will always be called before use and that will create the channel as well.
	return block, nil
}

// Create a method to allocate block in go lang for a given size without MMap
func AllocateBlockWithoutMmap(size uint64) (*Block, error) {
	if size == 0 {
		return nil, fmt.Errorf("invalid size")
	}

	block := &Block{
		data: make([]byte, size),
	}

	return block, nil
}

func BenchmarkCopyCostOfMmapMemoryToOneMiBBlock(b *testing.B) {
	srcBlock, err := AllocateBlockWithMmap(16 * MIB)
	if err != nil {
		b.Fatalf("failed to allocate source block: %v", err)
	}
	defer syscall.Munmap(srcBlock.data)

	dst := make([]byte, MIB)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 16; j += MIB {
			copy(dst, srcBlock.data[j:j+MIB])
		}
	}
}

func BenchmarkCopyCostOfWithoutMmapMemoryToOneMiBBlock(b *testing.B) {
	srcBlock, err := AllocateBlockWithoutMmap(16 * MIB)
	if err != nil {
		b.Fatalf("failed to allocate source block: %v", err)
	}

	dst := make([]byte, MIB)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 16; j += MIB {
			copy(dst, srcBlock.data[j:j+MIB])
		}
	}
}

/**
Command: go test -bench . -test.benchmem

BenchmarkCopyCostOfMmapMemoryToOneMiBBlock-96                      32607             39071 ns/op               0 B/op          0 allocs/op
BenchmarkCopyCostOfWithoutMmapMemoryToOneMiBBlock-96               20229             57384 ns/op               0 B/op          0 allocs/op

Conclusion: with mmap performance is better than without. The ration of latency with/without is 3:2.
*/

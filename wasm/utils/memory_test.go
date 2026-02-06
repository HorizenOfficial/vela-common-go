package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Note: BytesToPtr and functions that convert int32 pointers back to *byte cannot be
// properly tested in native 64-bit Go because pointers are truncated to 32 bits.
// These are tested through WASM module integration tests where pointers are actually 32-bit.

func TestAllocate(t *testing.T) {
	t.Run("positive size tracks allocation", func(t *testing.T) {
		resetAllocatorState()

		ptr := Allocate(100)
		require.NotEqual(t, int32(0), ptr)

		mapSize, totalBytes := GetAllocatedMemoryStats()
		require.Equal(t, int64(1), mapSize)
		require.Equal(t, int64(100), totalBytes)
	})

	t.Run("zero size returns null", func(t *testing.T) {
		resetAllocatorState()

		ptr := Allocate(0)
		require.Equal(t, int32(0), ptr)

		mapSize, totalBytes := GetAllocatedMemoryStats()
		require.Equal(t, int64(0), mapSize)
		require.Equal(t, int64(0), totalBytes)
	})

	t.Run("negative size returns null", func(t *testing.T) {
		resetAllocatorState()

		ptr := Allocate(-10)
		require.Equal(t, int32(0), ptr)

		mapSize, totalBytes := GetAllocatedMemoryStats()
		require.Equal(t, int64(0), mapSize)
		require.Equal(t, int64(0), totalBytes)
	})

	t.Run("multiple allocations", func(t *testing.T) {
		resetAllocatorState()

		ptr1 := Allocate(50)
		ptr2 := Allocate(75)
		ptr3 := Allocate(25)

		require.NotEqual(t, int32(0), ptr1)
		require.NotEqual(t, int32(0), ptr2)
		require.NotEqual(t, int32(0), ptr3)

		mapSize, totalBytes := GetAllocatedMemoryStats()
		require.Equal(t, int64(3), mapSize)
		require.Equal(t, int64(150), totalBytes)
	})
}

func TestDeallocate(t *testing.T) {
	t.Run("nil pointer is handled gracefully", func(t *testing.T) {
		resetAllocatorState()
		Allocate(50)
		initialMapSize, initialBytes := GetAllocatedMemoryStats()

		Deallocate(nil, 100) // Should not panic, just warn

		mapSize, totalBytes := GetAllocatedMemoryStats()
		require.Equal(t, initialMapSize, mapSize)
		require.Equal(t, initialBytes, totalBytes)
	})

	t.Run("pointer not in map is handled gracefully", func(t *testing.T) {
		resetAllocatorState()
		Allocate(50)
		initialMapSize, initialBytes := GetAllocatedMemoryStats()

		// Create a pointer that was never allocated
		var dummy byte
		Deallocate(&dummy, 10) // Should not panic, just warn

		mapSize, totalBytes := GetAllocatedMemoryStats()
		require.Equal(t, initialMapSize, mapSize)
		require.Equal(t, initialBytes, totalBytes)
	})
}

func TestGetAllocatedMemoryStats(t *testing.T) {
	resetAllocatorState()

	mapSize, totalBytes := GetAllocatedMemoryStats()
	require.Equal(t, int64(0), mapSize)
	require.Equal(t, int64(0), totalBytes)

	Allocate(100)
	Allocate(200)

	mapSize, totalBytes = GetAllocatedMemoryStats()
	require.Equal(t, int64(2), mapSize)
	require.Equal(t, int64(300), totalBytes)
}

func TestPtrToString(t *testing.T) {
	t.Run("valid string", func(t *testing.T) {
		data := []byte("hello world")
		result := PtrToString(&data[0], int32(len(data)))
		require.Equal(t, "hello world", result)
	})

	t.Run("nil pointer", func(t *testing.T) {
		result := PtrToString(nil, 10)
		require.Equal(t, "", result)
	})

	t.Run("zero length", func(t *testing.T) {
		data := []byte("hello")
		result := PtrToString(&data[0], 0)
		require.Equal(t, "", result)
	})

	t.Run("negative length", func(t *testing.T) {
		data := []byte("hello")
		result := PtrToString(&data[0], -1)
		require.Equal(t, "", result)
	})

	t.Run("partial string", func(t *testing.T) {
		data := []byte("hello world")
		result := PtrToString(&data[0], 5)
		require.Equal(t, "hello", result)
	})
}

func TestBytesToPtr(t *testing.T) {
	t.Run("empty data returns nil", func(t *testing.T) {
		ptr := BytesToPtr([]byte{})
		require.Nil(t, ptr)
	})

	t.Run("nil data returns nil", func(t *testing.T) {
		ptr := BytesToPtr(nil)
		require.Nil(t, ptr)
	})

	// Note: Testing the actual data written by BytesToPtr requires a WASM environment
	// where int32 pointers are valid. In native 64-bit Go, the pointer truncation
	// causes segfaults when dereferencing.
}

// resetAllocatorState clears the allocator state for test isolation
func resetAllocatorState() {
	allocatedMemory = make(map[uintptr][]byte)
	cumulativeAllocSize = 0
}

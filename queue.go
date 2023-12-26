// Package lfg implements a lock-free, multiple-producer, multiple-consumer queue.
package lfg

import (
	"sync/atomic"

	"golang.org/x/sys/cpu"
)

// Queue[T any] is a lock-free, multiple-producer, multiple-consumer queue.
type Queue[T any] struct {
	buf []T

	mask int64

	_               cpu.CacheLinePad
	consumerBarrier atomic.Int64
	_               cpu.CacheLinePad
	consumerCursor  atomic.Int64

	_               cpu.CacheLinePad
	producerBarrier atomic.Int64
	_               cpu.CacheLinePad
	producerCursor  atomic.Int64
}

// NewQueue creates a new Queue[T any] with the given size. The size must be a power of two.
func NewQueue[T any](size uint) *Queue[T] {
	if !isPot(size) {
		panic("size must be a power of two")
	}

	return &Queue[T]{
		buf:  make([]T, size),
		mask: int64(size - 1),
	}
}

// Enqueue adds an item to the queue. It returns false if the buffer is full.
func (b *Queue[T]) Enqueue(v T) bool {
	var pc, cb int64

	for {
		pc = b.producerCursor.Load()
		cb = b.consumerBarrier.Load()

		if pc-cb >= b.mask {
			return false
		}

		if b.producerCursor.CompareAndSwap(pc, pc+1) {
			break
		}
	}

	b.buf[(pc+1)&b.mask] = v

	for {
		if b.producerBarrier.CompareAndSwap(pc, pc+1) {
			break
		}
	}

	return true
}

// Dequeue removes an item from the queue. It returns false if the buffer is empty.
func (b *Queue[T]) Dequeue() (T, bool) {
	var cc, pb int64

	for {
		cc = b.consumerCursor.Load()
		pb = b.producerBarrier.Load()

		if pb == cc {
			var zero T
			return zero, false
		}

		if b.consumerCursor.CompareAndSwap(cc, cc+1) {
			break
		}
	}

	v := b.buf[(cc+1)&b.mask]

	for {
		if b.consumerBarrier.CompareAndSwap(cc, cc+1) {
			break
		}
	}

	return v, true
}

func isPot(n uint) bool {
	if n == 0 {
		return false
	}
	return (n & (n - 1)) == 0
}

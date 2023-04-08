// Package lfg implements a lock-free, multiple-producer, multiple-consumer queue.
// Queue[T any] is a generic type, where the items added to the queue are of type *T.
package lfg

import (
	"sync/atomic"
	"unsafe"
)

// Queue[T any] is a lock-free, multiple-producer, multiple-consumer queue.
type Queue[T any] struct {
	buf []*T

	mask int64

	consumerBarrier atomic.Int64
	consumerCursor  atomic.Int64

	producerBarrier atomic.Int64
	producerCursor  atomic.Int64
}

// NewQueue creates a new Queue[T any] with the given size. The size must be a power of two.
func NewQueue[T any](size uint) *Queue[T] {
	if !isPot(size) {
		panic("size must be a power of two")
	}

	return &Queue[T]{
		buf:  make([]*T, size),
		mask: int64(size - 1),
	}
}

// Enqueue adds an item to the queue. It returns false if the buffer is full.
func (b *Queue[T]) Enqueue(v *T) bool {
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

	atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&b.buf[(pc+1)&b.mask])), unsafe.Pointer(v))

	for {
		if b.producerBarrier.CompareAndSwap(pc, pc+1) {
			break
		}
	}

	return true
}

// Dequeue removes an item from the queue. It returns false if the buffer is empty.
func (b *Queue[T]) Dequeue() (*T, bool) {
	var cc, pb int64

	for {
		cc = b.consumerCursor.Load()
		pb = b.producerBarrier.Load()

		if pb == cc {
			return nil, false
		}

		if b.consumerCursor.CompareAndSwap(cc, cc+1) {
			break
		}
	}

	v := atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&b.buf[(cc+1)&b.mask])))

	for {
		if b.consumerBarrier.CompareAndSwap(cc, cc+1) {
			break
		}
	}

	return (*T)(v), true
}

func isPot(n uint) bool {
	if n == 0 {
		return false
	}
	return (n & (n - 1)) == 0
}

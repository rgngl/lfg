package lfg

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testMsg struct{}

func TestRingBufferSingleThread(t *testing.T) {
	b := NewQueue[int](4)

	t.Run("empty queue must return false", func(t *testing.T) {
		v, ok := b.Dequeue()
		assert.False(t, ok)
		assert.Nil(t, v)
	})

	t.Run("enqueue one and dequeue one", func(t *testing.T) {
		ok := b.Enqueue(intPtr(0))
		assert.True(t, ok)

		v, ok := b.Dequeue()
		assert.True(t, ok)
		assert.Equal(t, 0, *v)
	})

	t.Run("enqueue until buffer is full", func(t *testing.T) {
		ok := b.Enqueue(intPtr(1))
		assert.True(t, ok)
		ok = b.Enqueue(intPtr(2))
		assert.True(t, ok)
		ok = b.Enqueue(intPtr(3))
		assert.True(t, ok)
		ok = b.Enqueue(intPtr(4))
		assert.False(t, ok)
	})

	t.Run("dequeue one and check value", func(t *testing.T) {
		v, ok := b.Dequeue()
		assert.True(t, ok)
		assert.Equal(t, 1, *v)
	})

	t.Run("enqueing one after dequeueing one must succeed", func(t *testing.T) {
		ok := b.Enqueue(intPtr(5))
		assert.True(t, ok)
	})

	t.Run("dequeue until buffer is empty", func(t *testing.T) {
		v, ok := b.Dequeue()
		assert.True(t, ok)
		assert.Equal(t, 2, *v)
		v, ok = b.Dequeue()
		assert.True(t, ok)
		assert.Equal(t, 3, *v)
		v, ok = b.Dequeue()
		assert.True(t, ok)
		assert.Equal(t, 5, *v)

		v, ok = b.Dequeue()
		assert.False(t, ok)
		assert.Nil(t, v)
	})

	t.Run("creating queues with invalid sizes must panic", func(t *testing.T) {
		assert.Panics(t, func() {
			_ = NewQueue[int](3)
		})

		assert.Panics(t, func() {
			_ = NewQueue[int](0)
		})
	})
}

func TestRingBufferSPSC(t *testing.T) {
	b := NewQueue[int](4)

	wg := sync.WaitGroup{}
	wg.Add(2)

	count := 1_000

	go func() {
		defer wg.Done()

		for i := 0; i < count; {
			ok := b.Enqueue(intPtr(i))
			if ok {
				i++
			}
		}
	}()

	go func() {
		defer wg.Done()

		var expected int

		for i := 0; i < count; {
			v, ok := b.Dequeue()
			if ok {
				i++
				if expected != *v {
					panic("unexpected value")
				}
				expected++
			}
		}
	}()

	wg.Wait()
}

func BenchmarkRingBufferMPSC(b *testing.B) {
	buf := NewQueue[testMsg](1024)

	const producerCount = 4
	countPerProducer := b.N
	expectedCount := countPerProducer * producerCount

	wg := sync.WaitGroup{}
	wg.Add(producerCount + 1)

	go func() {
		defer wg.Done()

		for i := 0; i < expectedCount; {
			_, ok := buf.Dequeue()
			if ok {
				i++
			}
		}
	}()

	for i := 0; i < producerCount; i++ {
		go func() {
			defer wg.Done()

			msg := &testMsg{}

			for j := 0; j < countPerProducer; {
				ok := buf.Enqueue(msg)
				if ok {
					j++
				}
			}
		}()
	}

	wg.Wait()
}

func BenchmarkRingBufferSPSC(b *testing.B) {
	buf := NewQueue[testMsg](1024)

	wg := sync.WaitGroup{}
	wg.Add(2)

	b.ResetTimer()

	go func() {
		defer wg.Done()

		msg := &testMsg{}

		for i := 0; i < b.N; {
			ok := buf.Enqueue(msg)
			if ok {
				i++
			}
		}
	}()

	go func() {
		defer wg.Done()

		for i := 0; i < b.N; {
			_, ok := buf.Dequeue()
			if ok {
				i++
			}
		}
	}()

	wg.Wait()
}

func intPtr(i int) *int {
	return &i
}

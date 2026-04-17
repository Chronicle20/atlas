package extraction

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestAcquire_SerializesSameKey(t *testing.T) {
	key := "serialize-same-key"
	const N = 128
	var inside int32
	var maxInside int32
	var wg sync.WaitGroup
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func() {
			defer wg.Done()
			m := Acquire(key)
			defer Release(m)
			cur := atomic.AddInt32(&inside, 1)
			for {
				prev := atomic.LoadInt32(&maxInside)
				if cur <= prev || atomic.CompareAndSwapInt32(&maxInside, prev, cur) {
					break
				}
			}
			time.Sleep(100 * time.Microsecond)
			atomic.AddInt32(&inside, -1)
		}()
	}
	wg.Wait()
	if got := atomic.LoadInt32(&maxInside); got != 1 {
		t.Errorf("same-key concurrent Acquire held > 1 at once: max=%d", got)
	}
}

func TestAcquire_DifferentKeysDoNotBlock(t *testing.T) {
	var wg sync.WaitGroup
	start := make(chan struct{})
	results := make(chan string, 2)

	wg.Add(2)
	go func() {
		defer wg.Done()
		m := Acquire("key-a")
		defer Release(m)
		<-start
		results <- "a"
	}()
	go func() {
		defer wg.Done()
		m := Acquire("key-b")
		defer Release(m)
		<-start
		results <- "b"
	}()

	close(start)
	wg.Wait()
	close(results)

	if len(results) != 2 {
		t.Fatalf("expected both goroutines to make progress")
	}
}

func TestTryAcquire_FailsWhenHeld(t *testing.T) {
	key := "tryacquire-held"
	m1 := Acquire(key)
	defer Release(m1)

	m2, ok := TryAcquire(key)
	if ok {
		Release(m2)
		t.Fatalf("expected TryAcquire to fail while key held")
	}
}

func TestTryAcquire_SucceedsAfterRelease(t *testing.T) {
	key := "tryacquire-free"
	m1 := Acquire(key)
	Release(m1)

	m2, ok := TryAcquire(key)
	if !ok {
		t.Fatalf("expected TryAcquire to succeed after Release")
	}
	Release(m2)
}

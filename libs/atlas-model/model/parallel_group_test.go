package model

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"context"
)

func TestGroup_TwoSuccessfulProviders(t *testing.T) {
	g, _ := NewGroup(context.Background())
	fa := Submit(g, func() (int, error) { return 1, nil })
	fb := Submit(g, func() (string, error) { return "ok", nil })

	if err := g.Wait(); err != nil {
		t.Fatalf("Wait() unexpected error: %v", err)
	}
	if fa.Get() != 1 {
		t.Errorf("fa.Get() = %d, want 1", fa.Get())
	}
	if fb.Get() != "ok" {
		t.Errorf("fb.Get() = %q, want %q", fb.Get(), "ok")
	}
}

func TestGroup_OneProviderErrors(t *testing.T) {
	wantErr := errors.New("boom")
	g, _ := NewGroup(context.Background())
	_ = Submit(g, func() (int, error) { return 0, wantErr })
	_ = Submit(g, func() (int, error) { return 7, nil })

	err := g.Wait()
	if !errors.Is(err, wantErr) {
		t.Fatalf("Wait() = %v, want %v", err, wantErr)
	}
}

func TestGroup_BothProvidersError(t *testing.T) {
	errA := errors.New("a")
	errB := errors.New("b")
	g, _ := NewGroup(context.Background())
	_ = Submit(g, func() (int, error) { return 0, errA })
	_ = Submit(g, func() (int, error) { return 0, errB })

	err := g.Wait()
	if err == nil {
		t.Fatal("Wait() returned nil, want either errA or errB")
	}
	if !errors.Is(err, errA) && !errors.Is(err, errB) {
		t.Fatalf("Wait() = %v, want errA or errB", err)
	}
}

func TestGroup_ThreeProviders_AllSucceed(t *testing.T) {
	g, _ := NewGroup(context.Background())
	fa := Submit(g, func() (int, error) { return 1, nil })
	fb := Submit(g, func() (int, error) { return 2, nil })
	fc := Submit(g, func() (int, error) { return 3, nil })

	if err := g.Wait(); err != nil {
		t.Fatalf("Wait() unexpected error: %v", err)
	}
	got := fa.Get() + fb.Get() + fc.Get()
	if got != 6 {
		t.Errorf("sum = %d, want 6", got)
	}
}

func TestGroup_ConcurrencyProof(t *testing.T) {
	const sleep = 50 * time.Millisecond
	const tolerance = 40 * time.Millisecond // wall-clock slack

	var inFlight int32
	var maxConcurrent int32

	tick := func() (int, error) {
		cur := atomic.AddInt32(&inFlight, 1)
		// track the high-water mark
		for {
			old := atomic.LoadInt32(&maxConcurrent)
			if cur <= old || atomic.CompareAndSwapInt32(&maxConcurrent, old, cur) {
				break
			}
		}
		time.Sleep(sleep)
		atomic.AddInt32(&inFlight, -1)
		return 0, nil
	}

	start := time.Now()
	g, _ := NewGroup(context.Background())
	_ = Submit(g, tick)
	_ = Submit(g, tick)
	if err := g.Wait(); err != nil {
		t.Fatalf("Wait() unexpected error: %v", err)
	}
	elapsed := time.Since(start)

	if elapsed >= 2*sleep-tolerance {
		t.Fatalf("Wait() took %v; expected <%v (parallel execution)", elapsed, 2*sleep-tolerance)
	}
	if atomic.LoadInt32(&maxConcurrent) < 2 {
		t.Fatalf("maxConcurrent = %d, want >= 2", maxConcurrent)
	}
}

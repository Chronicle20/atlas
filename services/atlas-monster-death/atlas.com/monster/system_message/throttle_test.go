package system_message

import (
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestThrottle_Allow(t *testing.T) {
	tA := uuid.New()
	tB := uuid.New()

	tests := []struct {
		name string
		run  func(th *Throttle, clock *time.Time) []bool
	}{
		{
			name: "first call allowed",
			run: func(th *Throttle, clock *time.Time) []bool {
				return []bool{th.Allow(tA, 1)}
			},
		},
		{
			name: "second call inside window denied",
			run: func(th *Throttle, clock *time.Time) []bool {
				first := th.Allow(tA, 1)
				*clock = clock.Add(30 * time.Second)
				second := th.Allow(tA, 1)
				return []bool{first, second}
			},
		},
		{
			name: "call after window allowed",
			run: func(th *Throttle, clock *time.Time) []bool {
				first := th.Allow(tA, 1)
				*clock = clock.Add(61 * time.Second)
				second := th.Allow(tA, 1)
				return []bool{first, second}
			},
		},
		{
			name: "boundary: exactly the window is allowed",
			run: func(th *Throttle, clock *time.Time) []bool {
				first := th.Allow(tA, 1)
				*clock = clock.Add(60 * time.Second)
				second := th.Allow(tA, 1)
				return []bool{first, second}
			},
		},
		{
			name: "distinct characters are independent",
			run: func(th *Throttle, clock *time.Time) []bool {
				first := th.Allow(tA, 1)
				second := th.Allow(tA, 2)
				return []bool{first, second}
			},
		},
		{
			name: "distinct tenants are independent",
			run: func(th *Throttle, clock *time.Time) []bool {
				first := th.Allow(tA, 1)
				second := th.Allow(tB, 1)
				return []bool{first, second}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			clock := time.Now()
			clockFn := func() time.Time { return clock }
			th := NewThrottle(time.Minute, 4096, clockFn)

			got := tc.run(th, &clock)

			switch tc.name {
			case "first call allowed":
				if len(got) != 1 || got[0] != true {
					t.Fatalf("got %v, want [true]", got)
				}
			case "second call inside window denied":
				if len(got) != 2 || got[0] != true || got[1] != false {
					t.Fatalf("got %v, want [true false]", got)
				}
			case "call after window allowed":
				if len(got) != 2 || got[0] != true || got[1] != true {
					t.Fatalf("got %v, want [true true]", got)
				}
			case "boundary: exactly the window is allowed":
				if len(got) != 2 || got[0] != true || got[1] != true {
					t.Fatalf("got %v, want [true true]", got)
				}
			case "distinct characters are independent":
				if len(got) != 2 || got[0] != true || got[1] != true {
					t.Fatalf("got %v, want [true true]", got)
				}
			case "distinct tenants are independent":
				if len(got) != 2 || got[0] != true || got[1] != true {
					t.Fatalf("got %v, want [true true]", got)
				}
			}
		})
	}
}

func TestThrottle_SweepsWhenOverCapacity(t *testing.T) {
	tA := uuid.New()
	clock := time.Now()
	clockFn := func() time.Time { return clock }
	th := NewThrottle(time.Minute, 4, clockFn)

	for i := uint32(1); i <= 4; i++ {
		if !th.Allow(tA, i) {
			t.Fatalf("expected Allow(tA, %d) to be true", i)
		}
	}

	clock = clock.Add(61 * time.Second)
	if !th.Allow(tA, 5) {
		t.Fatalf("expected Allow(tA, 5) to be true after window elapsed")
	}
	if len(th.last) > 4 {
		t.Fatalf("expected stale entries to be swept, len(th.last) = %d", len(th.last))
	}

	if th.Allow(tA, 5) {
		t.Fatalf("expected Allow(tA, 5) to be false immediately after being recorded")
	}
}

func TestThrottle_ConcurrentAllowIsRaceFree(t *testing.T) {
	tA := uuid.New()
	th := NewThrottle(time.Minute, 4096, time.Now)

	var wg sync.WaitGroup
	results := make([]bool, 50)
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i] = th.Allow(tA, uint32(i))
		}(i)
	}
	wg.Wait()

	admitted := 0
	for _, r := range results {
		if r {
			admitted++
		}
	}
	if admitted != 50 {
		t.Fatalf("expected 50 admitted characters, got %d", admitted)
	}
}

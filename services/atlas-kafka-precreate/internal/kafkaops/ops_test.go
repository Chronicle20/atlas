package kafkaops

import (
	"context"
	"errors"
	"testing"
	"time"

	kafka "github.com/segmentio/kafka-go"
)

// fakeClock provides a deterministic Sleep/Now pair for retry tests. No test
// in this file sleeps for real.
type fakeClock struct {
	now   time.Time
	slept []time.Duration
}

func newFakeClock() *fakeClock {
	return &fakeClock{now: time.Unix(0, 0)}
}

func (c *fakeClock) Sleep(d time.Duration) {
	c.now = c.now.Add(d)
	c.slept = append(c.slept, d)
}

func (c *fakeClock) Now() time.Time {
	return c.now
}

func durationsEqual(a, b []time.Duration) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestWithCoordinatorRetry(t *testing.T) {
	transportErr := errors.New("dial tcp: connection refused")

	tests := []struct {
		name          string
		fnResults     []error
		expectErr     bool
		expectErrIs   error
		expectExact   error
		expectCalls   int
		minCalls      bool
		expectBackoff []time.Duration
	}{
		{
			name:          "succeeds first try",
			fnResults:     []error{nil},
			expectErr:     false,
			expectCalls:   1,
			expectBackoff: []time.Duration{},
		},
		{
			name:          "retries NotCoordinatorForGroup",
			fnResults:     []error{kafka.NotCoordinatorForGroup, nil},
			expectErr:     false,
			expectCalls:   2,
			expectBackoff: []time.Duration{250 * time.Millisecond},
		},
		{
			name:          "retries GroupCoordinatorNotAvailable",
			fnResults:     []error{kafka.GroupCoordinatorNotAvailable, nil},
			expectErr:     false,
			expectCalls:   2,
			expectBackoff: []time.Duration{250 * time.Millisecond},
		},
		{
			name: "backoff doubles and caps",
			fnResults: []error{
				kafka.NotCoordinatorForGroup,
				kafka.NotCoordinatorForGroup,
				kafka.NotCoordinatorForGroup,
				kafka.NotCoordinatorForGroup,
				kafka.NotCoordinatorForGroup,
				kafka.NotCoordinatorForGroup,
				nil,
			},
			expectErr:   false,
			expectCalls: 7,
			expectBackoff: []time.Duration{
				250 * time.Millisecond,
				500 * time.Millisecond,
				1 * time.Second,
				2 * time.Second,
				2 * time.Second,
				2 * time.Second,
			},
		},
		{
			name:          "does not retry an unrelated error",
			fnResults:     []error{kafka.UnknownMemberId},
			expectErr:     true,
			expectErrIs:   kafka.UnknownMemberId,
			expectCalls:   1,
			expectBackoff: []time.Duration{},
		},
		{
			name:          "does not retry a transport error",
			fnResults:     []error{transportErr},
			expectErr:     true,
			expectExact:   transportErr,
			expectCalls:   1,
			expectBackoff: []time.Duration{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			clock := newFakeClock()
			calls := 0
			fn := func() error {
				idx := calls
				calls++
				if idx < len(tc.fnResults) {
					return tc.fnResults[idx]
				}
				return tc.fnResults[len(tc.fnResults)-1]
			}

			cfg := RetryConfig{
				Base:   250 * time.Millisecond,
				Max:    2 * time.Second,
				Budget: 60 * time.Second,
				Sleep:  clock.Sleep,
				Now:    clock.Now,
			}

			err := WithCoordinatorRetry(context.Background(), cfg, fn)

			if tc.expectErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tc.expectErrIs != nil && !errors.Is(err, tc.expectErrIs) {
					t.Errorf("expected errors.Is(err, %v) to be true, got err=%v", tc.expectErrIs, err)
				}
				if tc.expectExact != nil && !errors.Is(err, tc.expectExact) {
					t.Errorf("expected err to be exactly %v, got %v", tc.expectExact, err)
				}
			} else if err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}

			if calls != tc.expectCalls {
				t.Errorf("expected %d calls, got %d", tc.expectCalls, calls)
			}

			if !durationsEqual(clock.slept, tc.expectBackoff) {
				t.Errorf("expected backoffs %v, got %v", tc.expectBackoff, clock.slept)
			}
		})
	}
}

func TestWithCoordinatorRetry_GivesUpAtBudget(t *testing.T) {
	clock := newFakeClock()
	start := clock.now

	calls := 0
	fn := func() error {
		calls++
		return kafka.NotCoordinatorForGroup
	}

	cfg := RetryConfig{
		Base:   250 * time.Millisecond,
		Max:    2 * time.Second,
		Budget: 60 * time.Second,
		Sleep:  clock.Sleep,
		Now:    clock.Now,
	}

	err := WithCoordinatorRetry(context.Background(), cfg, fn)

	if err == nil {
		t.Fatal("expected non-nil error when budget is exhausted")
	}
	if !errors.Is(err, kafka.NotCoordinatorForGroup) {
		t.Errorf("expected returned error to unwrap to kafka.NotCoordinatorForGroup, got %v", err)
	}
	if calls < 2 {
		t.Errorf("expected at least 2 calls, got %d", calls)
	}
	if clock.now.Sub(start) > 60*time.Second {
		t.Errorf("expected elapsed time <= 60s, got %s", clock.now.Sub(start))
	}
}

func TestWithLeaderRetry(t *testing.T) {
	transportErr := errors.New("dial tcp: connection refused")

	tests := []struct {
		name          string
		fnResults     []error
		expectErr     bool
		expectErrIs   error
		expectExact   error
		expectCalls   int
		expectBackoff []time.Duration
	}{
		{
			name:          "succeeds first try",
			fnResults:     []error{nil},
			expectErr:     false,
			expectCalls:   1,
			expectBackoff: []time.Duration{},
		},
		{
			name:          "retries NotLeaderForPartition",
			fnResults:     []error{kafka.NotLeaderForPartition, nil},
			expectErr:     false,
			expectCalls:   2,
			expectBackoff: []time.Duration{250 * time.Millisecond},
		},
		{
			name:          "retries LeaderNotAvailable",
			fnResults:     []error{kafka.LeaderNotAvailable, nil},
			expectErr:     false,
			expectCalls:   2,
			expectBackoff: []time.Duration{250 * time.Millisecond},
		},
		{
			name:          "does not retry an unrelated error",
			fnResults:     []error{kafka.UnknownTopicOrPartition},
			expectErr:     true,
			expectErrIs:   kafka.UnknownTopicOrPartition,
			expectCalls:   1,
			expectBackoff: []time.Duration{},
		},
		{
			name:          "does not retry a coordinator error",
			fnResults:     []error{kafka.NotCoordinatorForGroup},
			expectErr:     true,
			expectErrIs:   kafka.NotCoordinatorForGroup,
			expectCalls:   1,
			expectBackoff: []time.Duration{},
		},
		{
			name:          "does not retry a transport error",
			fnResults:     []error{transportErr},
			expectErr:     true,
			expectExact:   transportErr,
			expectCalls:   1,
			expectBackoff: []time.Duration{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			clock := newFakeClock()
			calls := 0
			fn := func() error {
				idx := calls
				calls++
				if idx < len(tc.fnResults) {
					return tc.fnResults[idx]
				}
				return tc.fnResults[len(tc.fnResults)-1]
			}

			cfg := RetryConfig{
				Base:   250 * time.Millisecond,
				Max:    2 * time.Second,
				Budget: 60 * time.Second,
				Sleep:  clock.Sleep,
				Now:    clock.Now,
			}

			err := WithLeaderRetry(context.Background(), cfg, fn)

			if tc.expectErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tc.expectErrIs != nil && !errors.Is(err, tc.expectErrIs) {
					t.Errorf("expected errors.Is(err, %v) to be true, got err=%v", tc.expectErrIs, err)
				}
				if tc.expectExact != nil && !errors.Is(err, tc.expectExact) {
					t.Errorf("expected err to be exactly %v, got %v", tc.expectExact, err)
				}
			} else if err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}

			if calls != tc.expectCalls {
				t.Errorf("expected %d calls, got %d", tc.expectCalls, calls)
			}

			if !durationsEqual(clock.slept, tc.expectBackoff) {
				t.Errorf("expected backoffs %v, got %v", tc.expectBackoff, clock.slept)
			}
		})
	}
}

func TestWithLeaderRetry_GivesUpAtBudget(t *testing.T) {
	clock := newFakeClock()
	start := clock.now

	calls := 0
	fn := func() error {
		calls++
		return kafka.NotLeaderForPartition
	}

	cfg := RetryConfig{
		Base:   250 * time.Millisecond,
		Max:    2 * time.Second,
		Budget: 60 * time.Second,
		Sleep:  clock.Sleep,
		Now:    clock.Now,
	}

	err := WithLeaderRetry(context.Background(), cfg, fn)

	if err == nil {
		t.Fatal("expected non-nil error when budget is exhausted")
	}
	if !errors.Is(err, kafka.NotLeaderForPartition) {
		t.Errorf("expected returned error to unwrap to kafka.NotLeaderForPartition, got %v", err)
	}
	if calls < 2 {
		t.Errorf("expected at least 2 calls, got %d", calls)
	}
	if clock.now.Sub(start) > 60*time.Second {
		t.Errorf("expected elapsed time <= 60s, got %s", clock.now.Sub(start))
	}
}

func TestWithCoordinatorRetry_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	clock := newFakeClock()
	calls := 0
	fn := func() error {
		calls++
		return kafka.NotCoordinatorForGroup
	}

	cfg := RetryConfig{
		Base:   250 * time.Millisecond,
		Max:    2 * time.Second,
		Budget: 60 * time.Second,
		Sleep:  clock.Sleep,
		Now:    clock.Now,
	}

	err := WithCoordinatorRetry(ctx, cfg, fn)

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected errors.Is(err, context.Canceled) to be true, got %v", err)
	}
	if calls > 1 {
		t.Errorf("expected fn called at most once, got %d calls", calls)
	}
}

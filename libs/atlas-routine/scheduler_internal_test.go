package routine

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
)

// blockingTask's Run blocks on a channel the test never closes, ignoring ctx.
// started is closed the first time Run is entered, so the test can wait for
// the run to actually be in flight before cancelling.
type blockingTask struct {
	mu      sync.Mutex
	sleep   time.Duration
	block   chan struct{}
	started chan struct{}
	once    sync.Once
}

func (b *blockingTask) Run(context.Context) {
	b.once.Do(func() { close(b.started) })
	<-b.block
}

func (b *blockingTask) SleepTime() time.Duration {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.sleep
}

// releasingTask's Run returns as soon as ctx is done.
type releasingTask struct {
	sleep time.Duration
}

func (r *releasingTask) Run(ctx context.Context) {
	<-ctx.Done()
}

func (r *releasingTask) SleepTime() time.Duration {
	return r.sleep
}

// zeroSleepTask always reports a non-positive SleepTime.
type zeroSleepTask struct {
	mu    sync.Mutex
	count int
}

func (z *zeroSleepTask) Run(context.Context) {
	z.mu.Lock()
	z.count++
	z.mu.Unlock()
}

func (z *zeroSleepTask) runCount() int {
	z.mu.Lock()
	defer z.mu.Unlock()
	return z.count
}

func (z *zeroSleepTask) SleepTime() time.Duration {
	return 0
}

func TestDrainTimeoutAbandonsBlockedRun(t *testing.T) {
	orig := drainTimeout
	drainTimeout = 100 * time.Millisecond
	defer func() { drainTimeout = orig }()

	l, hook := test.NewNullLogger()
	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	task := &blockingTask{sleep: 10 * time.Millisecond, block: make(chan struct{}), started: make(chan struct{})}
	defer close(task.block)

	Register(l, ctx, &wg)(task)

	select {
	case <-task.started:
	case <-time.After(2 * time.Second):
		t.Fatal("task never started")
	}

	cancel()
	start := time.Now()
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		elapsed := time.Since(start)
		if elapsed < 100*time.Millisecond {
			t.Fatalf("wg.Wait() returned in %s, want >= 100ms", elapsed)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("wg.Wait() did not return within 2s")
	}

	found := false
	for _, e := range hook.AllEntries() {
		if e.Level == logrus.WarnLevel {
			if task, _ := e.Data["task"].(string); task == "*routine.blockingTask" && strings.Contains(e.Message, "abandoning drain") {
				found = true
				break
			}
		}
	}
	if !found {
		t.Fatal("Warn entry with task == \"*routine.blockingTask\" and message containing \"abandoning drain\" was not logged")
	}
}

func TestDrainSuccessReleasesEarly(t *testing.T) {
	orig := drainTimeout
	drainTimeout = 5 * time.Second
	defer func() { drainTimeout = orig }()

	l, hook := test.NewNullLogger()
	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	task := &releasingTask{sleep: 10 * time.Millisecond}
	Register(l, ctx, &wg)(task)

	cancel()
	start := time.Now()
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		if elapsed := time.Since(start); elapsed >= time.Second {
			t.Fatalf("wg.Wait() took %s, want well under 1s", elapsed)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("wg.Wait() did not return within 2s")
	}

	for _, e := range hook.AllEntries() {
		if e.Level == logrus.WarnLevel {
			t.Fatalf("unexpected Warn entry: %s", e.Message)
		}
	}
}

func TestNonPositiveSleepTimeClamps(t *testing.T) {
	origMinSleep := minSleep
	minSleep = 50 * time.Millisecond
	defer func() { minSleep = origMinSleep }()

	l, hook := test.NewNullLogger()
	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	task := &zeroSleepTask{}
	Register(l, ctx, &wg)(task)

	time.Sleep(200 * time.Millisecond)
	if count := task.runCount(); count > 5 {
		t.Fatalf("runCount over 200ms = %d, want <= 5 (busy-spin)", count)
	}

	found := false
	for _, e := range hook.AllEntries() {
		if e.Level == logrus.WarnLevel {
			if taskField, _ := e.Data["task"].(string); taskField == "*routine.zeroSleepTask" && strings.Contains(e.Message, "non-positive SleepTime") {
				found = true
				break
			}
		}
	}
	if !found {
		t.Fatal("Warn entry with non-positive SleepTime message was not logged")
	}

	// Cancel and wait for the loop to fully stop before the deferred
	// restore of minSleep runs, so the background goroutine's last read of
	// minSleep happens-before the restore (avoids a data race).
	cancel()
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("wg.Wait() did not return within 2s")
	}
}

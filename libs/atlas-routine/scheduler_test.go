package routine_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"

	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
)

// fakeTask is a Task double for scheduler tests. All mutable state is
// guarded by mu since the scheduler goroutine and the test goroutine both
// touch it (the race detector runs on this package).
type fakeTask struct {
	mu    sync.Mutex
	sleep time.Duration
	runs  []context.Context     // one entry per Run, in order
	fn    func(context.Context) // optional per-test body; nil == no-op
}

func (f *fakeTask) Run(ctx context.Context) {
	f.mu.Lock()
	f.runs = append(f.runs, ctx)
	f.mu.Unlock()
	if f.fn != nil {
		f.fn(ctx)
	}
}

func (f *fakeTask) SleepTime() time.Duration {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.sleep
}

func (f *fakeTask) runCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.runs)
}

func (f *fakeTask) lastCtx() context.Context {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.runs) == 0 {
		return nil
	}
	return f.runs[len(f.runs)-1]
}

func TestRegisterSleepFirst(t *testing.T) {
	l, _ := test.NewNullLogger()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var wg sync.WaitGroup

	task := &fakeTask{sleep: 200 * time.Millisecond}
	routine.Register(l, ctx, &wg)(task)

	if task.runCount() != 0 {
		t.Fatalf("runCount immediately after Register = %d, want 0", task.runCount())
	}
	time.Sleep(50 * time.Millisecond)
	if task.runCount() != 0 {
		t.Fatalf("runCount at 50ms = %d, want 0", task.runCount())
	}
	waitFor(t, func() bool { return task.runCount() >= 1 }, "task never ran")
}

func TestRegisterAddsBeforeReturning(t *testing.T) {
	l, _ := test.NewNullLogger()
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	task := &fakeTask{sleep: 200 * time.Millisecond}
	routine.Register(l, ctx, &wg)(task)

	waitDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitDone)
	}()

	select {
	case <-waitDone:
		t.Fatal("wg.Wait() completed within 100ms; Register did not Add before returning")
	case <-time.After(100 * time.Millisecond):
	}

	cancel()
	select {
	case <-waitDone:
	case <-time.After(2 * time.Second):
		t.Fatal("wg.Wait() did not complete after cancel")
	}
}

func TestCancellationStopsLoop(t *testing.T) {
	l, hook := test.NewNullLogger()
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	task := &fakeTask{sleep: 10 * time.Millisecond}
	routine.Register(l, ctx, &wg)(task)

	waitFor(t, func() bool { return task.runCount() >= 1 }, "task never ran")
	cancel()

	waitFor(t, func() bool {
		for _, e := range hook.AllEntries() {
			if e.Message == "Stopping task execution." {
				return true
			}
		}
		return false
	}, "stop message not logged")

	count := task.runCount()
	time.Sleep(100 * time.Millisecond)
	if task.runCount() != count {
		t.Fatalf("runCount changed after cancel: was %d, now %d", count, task.runCount())
	}
}

func TestWaitGroupReachesZero(t *testing.T) {
	l, _ := test.NewNullLogger()
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	task := &fakeTask{sleep: 10 * time.Millisecond}
	routine.Register(l, ctx, &wg)(task)

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

func TestRunReceivesLiveContext(t *testing.T) {
	l, _ := test.NewNullLogger()
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	var mu sync.Mutex
	var tickErr error
	task := &fakeTask{sleep: 10 * time.Millisecond}
	task.fn = func(c context.Context) {
		mu.Lock()
		tickErr = c.Err()
		mu.Unlock()
	}

	routine.Register(l, ctx, &wg)(task)
	waitFor(t, func() bool { return task.runCount() >= 1 }, "task never ran")

	mu.Lock()
	err := tickErr
	mu.Unlock()
	if err != nil {
		t.Fatalf("ctx.Err() during tick = %v, want nil", err)
	}

	cancel()
	waitFor(t, func() bool {
		lc := task.lastCtx()
		return lc != nil && lc.Err() == context.Canceled
	}, "lastCtx().Err() never became context.Canceled")
}

func TestAlreadyCancelledContext(t *testing.T) {
	l, hook := test.NewNullLogger()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var wg sync.WaitGroup

	task := &fakeTask{sleep: 10 * time.Millisecond}
	routine.Register(l, ctx, &wg)(task)

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

	if task.runCount() != 0 {
		t.Fatalf("runCount = %d, want 0", task.runCount())
	}

	found := false
	for _, e := range hook.AllEntries() {
		if e.Message == "Stopping task execution." {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("\"Stopping task execution.\" was not logged")
	}
}

func TestPanicStopsOnlyThatTask(t *testing.T) {
	l, hook := test.NewNullLogger()
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	var panicked sync.Once
	taskA := &fakeTask{sleep: 10 * time.Millisecond}
	taskA.fn = func(context.Context) {
		panicked.Do(func() { panic("boom") })
	}
	taskB := &fakeTask{sleep: 10 * time.Millisecond}

	register := routine.Register(l, ctx, &wg)
	register(taskA)
	register(taskB)

	waitFor(t, func() bool { return taskA.runCount() >= 1 }, "task A never ran")

	countA := taskA.runCount()
	time.Sleep(200 * time.Millisecond)
	if taskA.runCount() != countA {
		t.Fatalf("task A runCount changed after panic: was %d, now %d", countA, taskA.runCount())
	}

	waitFor(t, func() bool { return taskB.runCount() >= 2 }, "task B did not keep running")

	found := false
	for _, e := range hook.AllEntries() {
		if e.Level == logrus.ErrorLevel {
			if p, _ := e.Data["panic"].(string); p == "boom" {
				found = true
				break
			}
		}
	}
	if !found {
		t.Fatal("Error level entry with panic == \"boom\" was not logged")
	}

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

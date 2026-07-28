package routine_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"

	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
)

type ctxKey string

// waitFor polls cond until it returns true or the deadline passes.
func waitFor(t *testing.T, cond func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal(msg)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestGoRunsFnWithGivenContext(t *testing.T) {
	l, _ := test.NewNullLogger()
	ctx := context.WithValue(context.Background(), ctxKey("k"), "v")
	got := make(chan context.Context, 1)
	routine.Go(l, ctx, func(c context.Context) { got <- c })
	select {
	case c := <-got:
		if c.Value(ctxKey("k")) != "v" {
			t.Fatalf("ctx not passed through unmodified: got %v", c.Value(ctxKey("k")))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("fn never ran")
	}
}

func TestGoPanicDoesNotPropagate(t *testing.T) {
	l, _ := test.NewNullLogger()
	panicked := make(chan struct{})
	routine.Go(l, context.Background(), func(context.Context) {
		defer close(panicked)
		panic("boom")
	})
	<-panicked
	// Sibling work keeps running after a contained panic.
	ok := make(chan struct{})
	routine.Go(l, context.Background(), func(context.Context) { close(ok) })
	select {
	case <-ok:
	case <-time.After(2 * time.Second):
		t.Fatal("sibling goroutine did not run after a panic")
	}
}

func TestGoPanicIsLogged(t *testing.T) {
	l, hook := test.NewNullLogger()
	routine.Go(l, context.Background(), func(context.Context) {
		panic("kaboom-sentinel")
	})
	waitFor(t, func() bool { return hook.LastEntry() != nil }, "panic was not logged")
	e := hook.LastEntry()
	if e.Level != logrus.ErrorLevel {
		t.Fatalf("expected Error level, got %v", e.Level)
	}
	if e.Message != "Recovered panic in background goroutine." {
		t.Fatalf("unexpected message: %q", e.Message)
	}
	if p, _ := e.Data["panic"].(string); p != "kaboom-sentinel" {
		t.Fatalf("panic field = %q, want %q", p, "kaboom-sentinel")
	}
	stack, _ := e.Data["stack"].(string)
	if !strings.Contains(stack, "TestGoPanicIsLogged") {
		t.Fatalf("stack field missing this test's frame: %q", stack)
	}
}

func TestGoFnDefersRunBeforeRecoverLog(t *testing.T) {
	l, hook := test.NewNullLogger()
	deferRan := make(chan struct{})
	routine.Go(l, context.Background(), func(context.Context) {
		defer close(deferRan)
		panic("boom")
	})
	waitFor(t, func() bool { return hook.LastEntry() != nil }, "panic was not logged")
	// The log entry exists, so recover already ran; fn's defer must have run first.
	select {
	case <-deferRan:
	default:
		t.Fatal("fn's defer did not run before the helper's recover")
	}
}

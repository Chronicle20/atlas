package service

import (
	"syscall"
	"testing"
	"time"
)

// TestBootstrapLifecycleSIGTERMFlipsReadiness sends a real SIGTERM to the
// test process and drives Manager.Wait() end-to-end: teardown funcs fire,
// the readiness controller flips, Wait returns. MUST run last in the
// package (the zz_ filename enforces source ordering) because the teardown
// manager singleton cannot be re-armed.
func TestBootstrapLifecycleSIGTERMFlipsReadiness(t *testing.T) {
	rt := Bootstrap("atlas-test", WithoutTracer())
	rt.Logger().SetOutput(testWriter{t})

	if !rt.Ready() {
		t.Fatal("must be Ready before SIGTERM")
	}

	done := make(chan struct{})
	go func() {
		rt.Wait()
		close(done)
	}()
	if err := syscall.Kill(syscall.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("Wait() did not return after SIGTERM")
	}
	if rt.Ready() {
		t.Fatal("Ready() must be false after teardown")
	}
}

type testWriter struct{ t *testing.T }

func (w testWriter) Write(p []byte) (int, error) {
	w.t.Log(string(p))
	return len(p), nil
}

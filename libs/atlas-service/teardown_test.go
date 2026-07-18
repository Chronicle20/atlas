package service

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"
)

func TestGetTeardownManager(t *testing.T) {
	manager := GetTeardownManager()

	if manager == nil {
		t.Error("GetTeardownManager() returned nil")
	}

	// Test that we get the same instance (singleton)
	manager2 := GetTeardownManager()
	if manager != manager2 {
		t.Error("GetTeardownManager() did not return the same instance")
	}
}

func TestTeardownManager_TeardownFunc(t *testing.T) {
	// Use a local Manager, not GetTeardownManager()'s shared singleton:
	// registering a teardown on the singleton parks a goroutine on its
	// doneChan, which another test (TestBootstrapLifecycleSIGTERMFlipsReadiness)
	// later closes — racing on this frame's `called`. A local Manager keeps
	// the teardown isolated to this test.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager := &Manager{
		termChan:  make(chan os.Signal, 1),
		doneChan:  make(chan struct{}),
		waitGroup: &sync.WaitGroup{},
		context:   ctx,
		cancel:    cancel,
	}

	called := false
	teardownFunc := func() {
		called = true
	}

	// Test that TeardownFunc doesn't panic and defers execution.
	manager.TeardownFunc(teardownFunc)

	// The function must not run until teardown (doneChan close).
	if called {
		t.Error("Teardown function was called immediately, expected to be deferred")
	}

	// Drain the parked teardown goroutine so it does not outlive the test.
	// The close happens-before the goroutine's write, which happens-before
	// Wait() returns — so this is race-free.
	close(manager.doneChan)
	manager.waitGroup.Wait()
}

func TestTeardownManager_Wait(t *testing.T) {
	// Create a separate context for testing
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager := &Manager{
		termChan:  make(chan os.Signal, 1),
		doneChan:  make(chan struct{}),
		waitGroup: &sync.WaitGroup{},
		context:   ctx,
		cancel:    cancel,
	}

	called := false
	teardownFunc := func() {
		called = true
	}

	manager.TeardownFunc(teardownFunc)

	// Test Wait with a timeout to avoid hanging
	done := make(chan bool)
	go func() {
		// Send signal to trigger termination
		manager.termChan <- os.Interrupt
		manager.Wait()
		done <- true
	}()

	// Give it a moment to process
	select {
	case <-done:
		// Wait a bit for teardown function to execute
		time.Sleep(10 * time.Millisecond)
		if !called {
			t.Error("Teardown function was not called")
		}
	case <-time.After(1 * time.Second):
		t.Error("Wait() took too long")
	}
}

func TestTeardownManager_WaitGroup(t *testing.T) {
	manager := GetTeardownManager()

	wg := manager.WaitGroup()
	if wg == nil {
		t.Error("WaitGroup() returned nil")
	}

	// Test that we get the same instance
	wg2 := manager.WaitGroup()
	if wg != wg2 {
		t.Error("WaitGroup() did not return the same instance")
	}
}

func TestTeardownManager_Context(t *testing.T) {
	manager := GetTeardownManager()

	ctx := manager.Context()
	if ctx == nil {
		t.Error("Context() returned nil")
	}

	// Test that we get the same instance
	ctx2 := manager.Context()
	if ctx != ctx2 {
		t.Error("Context() did not return the same instance")
	}
}

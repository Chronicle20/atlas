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
	manager := GetTeardownManager()

	called := false
	teardownFunc := func() {
		called = true
	}

	// Test that TeardownFunc doesn't panic
	manager.TeardownFunc(teardownFunc)

	// We can't directly test the teardown execution without calling Wait()
	// But we can test that the function was registered without error
	if called {
		t.Error("Teardown function was called immediately, expected to be deferred")
	}
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

package testutil

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Chronicle20/atlas-model/async"
	"github.com/Chronicle20/atlas-model/model"
)

// GoroutineLeakDetector helps detect goroutine leaks in tests
type GoroutineLeakDetector struct {
	initialCount int
	t            *testing.T
}

// NewGoroutineLeakDetector creates a new goroutine leak detector
func NewGoroutineLeakDetector(t *testing.T) *GoroutineLeakDetector {
	return &GoroutineLeakDetector{
		initialCount: runtime.NumGoroutine(),
		t:            t,
	}
}

// Check verifies that no goroutines were leaked since creation
func (g *GoroutineLeakDetector) Check() {
	g.CheckWithRetries(3)
}

// CheckWithRetries verifies that no goroutines were leaked, with retries to handle timing
func (g *GoroutineLeakDetector) CheckWithRetries(retries int) {
	for i := 0; i < retries; i++ {
		runtime.GC()
		time.Sleep(time.Millisecond * 10) // Give goroutines time to cleanup
		
		currentCount := runtime.NumGoroutine()
		if currentCount <= g.initialCount {
			return
		}
		
		if i == retries-1 {
			g.t.Errorf("Goroutine leak detected: started with %d goroutines, now have %d", g.initialCount, currentCount)
		}
	}
}

// TimeoutContext creates a context with timeout and returns both context and cancel function
func TimeoutContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

// CancelledContext returns a pre-cancelled context for testing cancellation scenarios
func CancelledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

// TestError creates a standard test error with consistent formatting
func TestError(message string) error {
	return errors.New("test error: " + message)
}

// TestErrorWithValue creates a test error that includes a value for validation
func TestErrorWithValue(value interface{}) error {
	return fmt.Errorf("test error with value: %v", value)
}

// Provider Generators

// ErrorProvider creates a model provider that always returns an error
func ErrorProvider[T any](err error) model.Provider[T] {
	return model.ErrorProvider[T](err)
}

// FixedProvider creates a model provider that returns a fixed value
func FixedProvider[T any](value T) model.Provider[T] {
	return model.FixedProvider(value)
}

// DelayedProvider creates a provider that returns after a delay
func DelayedProvider[T any](value T, delay time.Duration) model.Provider[T] {
	return func() (T, error) {
		time.Sleep(delay)
		return value, nil
	}
}

// CancellableProvider creates a provider that respects context cancellation
func CancellableProvider[T any](ctx context.Context, value T, delay time.Duration) model.Provider[T] {
	return func() (T, error) {
		select {
		case <-ctx.Done():
			var zero T
			return zero, ctx.Err()
		case <-time.After(delay):
			return value, nil
		}
	}
}

// CountingProvider creates a provider that increments a counter when called
func CountingProvider[T any](value T, counter *int64) model.Provider[T] {
	return func() (T, error) {
		atomic.AddInt64(counter, 1)
		return value, nil
	}
}

// SliceProvider creates a provider that returns a slice of values
func SliceProvider[T any](values []T) model.Provider[[]T] {
	return model.FixedProvider(values)
}

// Async Provider Generators

// AsyncErrorProvider creates an async provider that returns an error
func AsyncErrorProvider[T any](err error) async.Provider[T] {
	return func(ctx context.Context, rchan chan T, echan chan error) {
		echan <- err
	}
}

// AsyncDelayedProvider creates an async provider with a delay
func AsyncDelayedProvider[T any](value T, delay time.Duration) async.Provider[T] {
	return func(ctx context.Context, rchan chan T, echan chan error) {
		select {
		case <-ctx.Done():
			echan <- ctx.Err()
		case <-time.After(delay):
			rchan <- value
		}
	}
}

// AsyncCountingProvider creates an async provider that increments a counter
func AsyncCountingProvider[T any](value T, counter *int64) async.Provider[T] {
	return func(ctx context.Context, rchan chan T, echan chan error) {
		atomic.AddInt64(counter, 1)
		rchan <- value
	}
}

// Test Data Generators

// GenerateSlice creates a slice of sequential values for testing
func GenerateSlice(start, count int) []uint32 {
	result := make([]uint32, count)
	for i := 0; i < count; i++ {
		result[i] = uint32(start + i)
	}
	return result
}

// GenerateLargeSlice creates a large slice for stress testing
func GenerateLargeSlice(count int) []uint32 {
	return GenerateSlice(1, count)
}

// Concurrent Testing Helpers

// ConcurrentRunner helps run concurrent tests with proper synchronization
type ConcurrentRunner struct {
	wg     sync.WaitGroup
	mu     sync.Mutex
	errors []error
}

// NewConcurrentRunner creates a new concurrent test runner
func NewConcurrentRunner() *ConcurrentRunner {
	return &ConcurrentRunner{
		errors: make([]error, 0),
	}
}

// Add increments the wait group counter
func (cr *ConcurrentRunner) Add(delta int) {
	cr.wg.Add(delta)
}

// Go runs a function in a goroutine and tracks any errors
func (cr *ConcurrentRunner) Go(fn func() error) {
	go func() {
		defer cr.wg.Done()
		if err := fn(); err != nil {
			cr.mu.Lock()
			cr.errors = append(cr.errors, err)
			cr.mu.Unlock()
		}
	}()
}

// Wait waits for all goroutines to complete
func (cr *ConcurrentRunner) Wait() {
	cr.wg.Wait()
}

// Errors returns all collected errors
func (cr *ConcurrentRunner) Errors() []error {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	result := make([]error, len(cr.errors))
	copy(result, cr.errors)
	return result
}

// CheckErrors fails the test if any errors were collected
func (cr *ConcurrentRunner) CheckErrors(t *testing.T) {
	errors := cr.Errors()
	if len(errors) > 0 {
		for i, err := range errors {
			t.Errorf("Concurrent operation %d failed: %v", i, err)
		}
	}
}

// Memory Pressure Testing

// MemoryPressureTest runs a function under memory pressure
func MemoryPressureTest(t *testing.T, fn func() error) {
	// Force garbage collection before test
	runtime.GC()
	runtime.GC() // Run twice to be thorough
	
	var m1, m2 runtime.MemStats
	runtime.ReadMemStats(&m1)
	
	err := fn()
	if err != nil {
		t.Fatalf("Function failed under memory pressure: %v", err)
	}
	
	runtime.GC()
	runtime.ReadMemStats(&m2)
	
	// Check for excessive memory growth (more than 10MB)
	if m2.Alloc > m1.Alloc+10*1024*1024 {
		t.Errorf("Excessive memory growth detected: %d bytes -> %d bytes", m1.Alloc, m2.Alloc)
	}
}

// Channel Testing Helpers

// ChannelMonitor helps monitor channel operations for testing
type ChannelMonitor struct {
	channelOps int64
	mu         sync.RWMutex
}

// NewChannelMonitor creates a new channel monitor
func NewChannelMonitor() *ChannelMonitor {
	return &ChannelMonitor{}
}

// IncrementOps atomically increments the operation counter
func (cm *ChannelMonitor) IncrementOps() {
	atomic.AddInt64(&cm.channelOps, 1)
}

// GetOps returns the current operation count
func (cm *ChannelMonitor) GetOps() int64 {
	return atomic.LoadInt64(&cm.channelOps)
}

// Reset resets the operation counter
func (cm *ChannelMonitor) Reset() {
	atomic.StoreInt64(&cm.channelOps, 0)
}

// Performance Testing Helpers

// BenchmarkHelper provides utilities for benchmark tests
type BenchmarkHelper struct {
	b      *testing.B
	start  time.Time
	paused bool
}

// NewBenchmarkHelper creates a new benchmark helper
func NewBenchmarkHelper(b *testing.B) *BenchmarkHelper {
	return &BenchmarkHelper{
		b:     b,
		start: time.Now(),
	}
}

// StartTimer starts the benchmark timer
func (bh *BenchmarkHelper) StartTimer() {
	if bh.paused {
		bh.b.StartTimer()
		bh.paused = false
	}
}

// StopTimer stops the benchmark timer
func (bh *BenchmarkHelper) StopTimer() {
	if !bh.paused {
		bh.b.StopTimer()
		bh.paused = true
	}
}

// ResetTimer resets the benchmark timer
func (bh *BenchmarkHelper) ResetTimer() {
	bh.b.ResetTimer()
	bh.start = time.Now()
	bh.paused = false
}

// Elapsed returns the elapsed time
func (bh *BenchmarkHelper) Elapsed() time.Duration {
	return time.Since(bh.start)
}

// Context Testing Helpers

// ContextWithValues creates a context with test values
func ContextWithValues(keyValues map[string]interface{}) context.Context {
	ctx := context.Background()
	for key, value := range keyValues {
		ctx = context.WithValue(ctx, key, value)
	}
	return ctx
}

// ExpiredContext returns a context that has already expired
func ExpiredContext() context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	time.Sleep(time.Millisecond) // Ensure context expires
	return ctx
}

// Error Simulation Helpers

// ErrorGenerator generates different types of test errors
type ErrorGenerator struct {
	counter int64
}

// NewErrorGenerator creates a new error generator
func NewErrorGenerator() *ErrorGenerator {
	return &ErrorGenerator{}
}

// Next generates the next error in sequence
func (eg *ErrorGenerator) Next() error {
	count := atomic.AddInt64(&eg.counter, 1)
	return fmt.Errorf("generated error #%d", count)
}

// NetworkError simulates a network error
func (eg *ErrorGenerator) NetworkError() error {
	count := atomic.AddInt64(&eg.counter, 1)
	return fmt.Errorf("network error #%d: connection timeout", count)
}

// TimeoutError simulates a timeout error
func (eg *ErrorGenerator) TimeoutError() error {
	count := atomic.AddInt64(&eg.counter, 1)
	return fmt.Errorf("timeout error #%d: operation timed out", count)
}

// ValidationError simulates a validation error
func (eg *ErrorGenerator) ValidationError() error {
	count := atomic.AddInt64(&eg.counter, 1)
	return fmt.Errorf("validation error #%d: invalid input", count)
}
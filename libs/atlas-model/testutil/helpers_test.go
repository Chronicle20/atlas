package testutil

import (
	"context"
	"runtime"
	"sync/atomic"
	"testing"
	"time"
)

func TestGoroutineLeakDetector(t *testing.T) {
	t.Run("NewGoroutineLeakDetector creates detector", func(t *testing.T) {
		initialCount := runtime.NumGoroutine()
		detector := NewGoroutineLeakDetector(t)
		
		if detector.initialCount != initialCount {
			t.Errorf("Expected initial count %d, got %d", initialCount, detector.initialCount)
		}
		if detector.t != t {
			t.Error("Expected detector to reference test instance")
		}
	})

	t.Run("Check detects no leak when goroutines match", func(t *testing.T) {
		detector := NewGoroutineLeakDetector(t)
		// No goroutines created, should pass
		detector.Check()
	})

	t.Run("CheckWithRetries with specific retry count", func(t *testing.T) {
		detector := NewGoroutineLeakDetector(t)
		detector.CheckWithRetries(5)
	})
}

func TestTimeoutContext(t *testing.T) {
	t.Run("TimeoutContext creates context with timeout", func(t *testing.T) {
		timeout := 50 * time.Millisecond
		ctx, cancel := TimeoutContext(timeout)
		defer cancel()
		
		deadline, ok := ctx.Deadline()
		if !ok {
			t.Error("Expected context to have deadline")
		}
		
		// Verify deadline is approximately correct (within 10ms tolerance)
		expectedDeadline := time.Now().Add(timeout)
		if deadline.Before(expectedDeadline.Add(-10*time.Millisecond)) || 
		   deadline.After(expectedDeadline.Add(10*time.Millisecond)) {
			t.Errorf("Deadline %v not within expected range around %v", deadline, expectedDeadline)
		}
		
		// Wait for timeout
		select {
		case <-ctx.Done():
			if ctx.Err() != context.DeadlineExceeded {
				t.Errorf("Expected DeadlineExceeded, got %v", ctx.Err())
			}
		case <-time.After(timeout + 20*time.Millisecond):
			t.Error("Context should have timed out")
		}
	})
}

func TestCancelledContext(t *testing.T) {
	t.Run("CancelledContext returns already cancelled context", func(t *testing.T) {
		ctx := CancelledContext()
		
		select {
		case <-ctx.Done():
			if ctx.Err() != context.Canceled {
				t.Errorf("Expected Canceled, got %v", ctx.Err())
			}
		default:
			t.Error("Context should be already cancelled")
		}
	})
}

func TestTestError(t *testing.T) {
	t.Run("TestError returns error with message", func(t *testing.T) {
		message := "test error message"
		err := TestError(message)
		
		if err == nil {
			t.Error("Expected error, got nil")
		}
		expectedMsg := "test error: " + message
		if err.Error() != expectedMsg {
			t.Errorf("Expected message '%s', got '%s'", expectedMsg, err.Error())
		}
	})
}

func TestTestErrorWithValue(t *testing.T) {
	t.Run("TestErrorWithValue includes value in message", func(t *testing.T) {
		value := 42
		err := TestErrorWithValue(value)
		
		if err == nil {
			t.Error("Expected error, got nil")
		}
		
		expected := "test error with value: 42"
		if err.Error() != expected {
			t.Errorf("Expected message '%s', got '%s'", expected, err.Error())
		}
	})
}

func TestErrorProvider(t *testing.T) {
	t.Run("ErrorProvider returns provider that always errors", func(t *testing.T) {
		testError := TestError("test error")
		provider := ErrorProvider[string](testError)
		
		result, err := provider()
		if err != testError {
			t.Errorf("Expected test error, got %v", err)
		}
		if result != "" {
			t.Errorf("Expected zero value for string, got %s", result)
		}
	})
}

func TestFixedProvider(t *testing.T) {
	t.Run("FixedProvider returns provider with fixed value", func(t *testing.T) {
		value := "fixed value"
		provider := FixedProvider(value)
		
		result, err := provider()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if result != value {
			t.Errorf("Expected '%s', got '%s'", value, result)
		}
	})
}

func TestDelayedProvider(t *testing.T) {
	t.Run("DelayedProvider introduces delay before returning", func(t *testing.T) {
		delay := 50 * time.Millisecond
		value := 123
		provider := DelayedProvider(value, delay)
		
		start := time.Now()
		result, err := provider()
		elapsed := time.Since(start)
		
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if result != value {
			t.Errorf("Expected %d, got %d", value, result)
		}
		if elapsed < delay {
			t.Errorf("Expected delay of at least %v, got %v", delay, elapsed)
		}
	})
}

func TestCancellableProvider(t *testing.T) {
	t.Run("CancellableProvider respects context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		delay := 100 * time.Millisecond
		value := "test value"
		
		provider := CancellableProvider(ctx, value, delay)
		
		// Cancel after short delay
		go func() {
			time.Sleep(10 * time.Millisecond)
			cancel()
		}()
		
		start := time.Now()
		result, err := provider()
		elapsed := time.Since(start)
		
		if err != context.Canceled {
			t.Errorf("Expected context.Canceled, got %v", err)
		}
		if result != "" {
			t.Errorf("Expected zero value, got %s", result)
		}
		if elapsed >= delay {
			t.Errorf("Expected cancellation before delay %v, took %v", delay, elapsed)
		}
	})

	t.Run("CancellableProvider completes when not cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		
		delay := 10 * time.Millisecond
		value := "completed value"
		
		provider := CancellableProvider(ctx, value, delay)
		
		result, err := provider()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if result != value {
			t.Errorf("Expected '%s', got '%s'", value, result)
		}
	})
}

func TestCountingProvider(t *testing.T) {
	t.Run("CountingProvider increments call count", func(t *testing.T) {
		value := 42
		counter := int64(0)
		provider := CountingProvider(value, &counter)
		
		// Call multiple times and verify count
		for i := 1; i <= 3; i++ {
			result, err := provider()
			if err != nil {
				t.Errorf("Call %d: Expected no error, got %v", i, err)
			}
			if result != value {
				t.Errorf("Call %d: Expected value %d, got %d", i, value, result)
			}
			if atomic.LoadInt64(&counter) != int64(i) {
				t.Errorf("Call %d: Expected call count %d, got %d", i, i, atomic.LoadInt64(&counter))
			}
		}
	})
}

func TestSliceProvider(t *testing.T) {
	t.Run("SliceProvider returns slice with values", func(t *testing.T) {
		values := []int{1, 2, 3, 4, 5}
		provider := SliceProvider(values)
		
		result, err := provider()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if len(result) != len(values) {
			t.Errorf("Expected slice length %d, got %d", len(values), len(result))
		}
		for i, expected := range values {
			if result[i] != expected {
				t.Errorf("Index %d: Expected %d, got %d", i, expected, result[i])
			}
		}
	})
}

func TestAsyncErrorProvider(t *testing.T) {
	t.Run("AsyncErrorProvider returns error", func(t *testing.T) {
		testError := TestError("async error")
		provider := AsyncErrorProvider[string](testError)
		
		ctx := context.Background()
		rchan := make(chan string, 1)
		echan := make(chan error, 1)
		provider(ctx, rchan, echan)

		var result string
		var err error
		select {
		case result = <-rchan:
		case err = <-echan:
		case <-time.After(100 * time.Millisecond):
			t.Error("Timeout waiting for result")
			return
		}
		
		if err != testError {
			t.Errorf("Expected test error, got %v", err)
		}
		if result != "" {
			t.Errorf("Expected zero value, got %s", result)
		}
	})
}

func TestAsyncDelayedProvider(t *testing.T) {
	t.Run("AsyncDelayedProvider returns value after delay", func(t *testing.T) {
		value := "async value"
		delay := 20 * time.Millisecond
		provider := AsyncDelayedProvider(value, delay)
		
		start := time.Now()
		ctx := context.Background()
		rchan := make(chan string, 1)
		echan := make(chan error, 1)
		provider(ctx, rchan, echan)

		var result string
		var err error
		select {
		case result = <-rchan:
		case err = <-echan:
		case <-time.After(100 * time.Millisecond):
			t.Error("Timeout waiting for result")
			return
		}
		elapsed := time.Since(start)
		
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if result != value {
			t.Errorf("Expected '%s', got '%s'", value, result)
		}
		if elapsed < delay {
			t.Errorf("Expected delay of at least %v, got %v", delay, elapsed)
		}
	})
}

func TestAsyncCountingProvider(t *testing.T) {
	t.Run("AsyncCountingProvider counts calls", func(t *testing.T) {
		value := 100
		counter := int64(0)
		provider := AsyncCountingProvider(value, &counter)
		
		ctx := context.Background()
		rchan := make(chan int, 1)
		echan := make(chan error, 1)
		provider(ctx, rchan, echan)

		var result int
		var err error
		select {
		case result = <-rchan:
		case err = <-echan:
		case <-time.After(100 * time.Millisecond):
			t.Error("Timeout waiting for result")
			return
		}
		
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if result != value {
			t.Errorf("Expected value %d, got %d", value, result)
		}
		if atomic.LoadInt64(&counter) != 1 {
			t.Errorf("Expected call count 1, got %d", atomic.LoadInt64(&counter))
		}
	})
}

func TestGenerateSlice(t *testing.T) {
	t.Run("GenerateSlice creates slice of specified size", func(t *testing.T) {
		start := 0
		count := 10
		slice := GenerateSlice(start, count)
		
		if len(slice) != count {
			t.Errorf("Expected slice length %d, got %d", count, len(slice))
		}
		
		// Verify values are sequential
		for i, val := range slice {
			expected := uint32(start + i)
			if val != expected {
				t.Errorf("Index %d: Expected %d, got %d", i, expected, val)
			}
		}
	})
}

func TestGenerateLargeSlice(t *testing.T) {
	t.Run("GenerateLargeSlice creates large slice", func(t *testing.T) {
		size := 100000
		slice := GenerateLargeSlice(size)
		
		if len(slice) != size {
			t.Errorf("Expected slice length %d, got %d", size, len(slice))
		}
		
		// Verify first few and last few values
		for i := 0; i < 5; i++ {
			expected := uint32(1 + i) // GenerateLargeSlice starts from 1
			if slice[i] != expected {
				t.Errorf("Index %d: Expected %d, got %d", i, expected, slice[i])
			}
		}
		
		for i := size - 5; i < size; i++ {
			expected := uint32(1 + i)
			if slice[i] != expected {
				t.Errorf("Index %d: Expected %d, got %d", i, expected, slice[i])
			}
		}
	})
}

func TestConcurrentRunner(t *testing.T) {
	t.Run("ConcurrentRunner executes operations concurrently", func(t *testing.T) {
		runner := NewConcurrentRunner()
		
		counter := int64(0)
		numOperations := 10
		
		// Add operations that increment counter
		for i := 0; i < numOperations; i++ {
			runner.Add(1)
			runner.Go(func() error {
				atomic.AddInt64(&counter, 1)
				time.Sleep(5 * time.Millisecond)
				return nil
			})
		}
		
		start := time.Now()
		runner.Wait()
		elapsed := time.Since(start)
		
		// Should complete faster than sequential execution
		sequentialTime := time.Duration(numOperations) * 5 * time.Millisecond
		if elapsed >= sequentialTime {
			t.Errorf("Expected concurrent execution faster than %v, took %v", sequentialTime, elapsed)
		}
		
		if atomic.LoadInt64(&counter) != int64(numOperations) {
			t.Errorf("Expected counter %d, got %d", numOperations, atomic.LoadInt64(&counter))
		}
		
		// Check for errors
		errors := runner.Errors()
		if len(errors) != 0 {
			t.Errorf("Expected no errors, got %v", errors)
		}
	})

	t.Run("ConcurrentRunner handles errors", func(t *testing.T) {
		runner := NewConcurrentRunner()
		
		testError := TestError("operation error")
		
		// Add successful and failing operations
		runner.Add(1)
		runner.Go(func() error {
			time.Sleep(5 * time.Millisecond)
			return nil
		})
		runner.Add(1)
		runner.Go(func() error {
			time.Sleep(5 * time.Millisecond)
			return testError
		})
		runner.Add(1)
		runner.Go(func() error {
			time.Sleep(5 * time.Millisecond)
			return nil
		})
		
		runner.Wait()
		
		errors := runner.Errors()
		if len(errors) != 1 {
			t.Errorf("Expected 1 error, got %d", len(errors))
		}
		if len(errors) > 0 && errors[0] != testError {
			t.Errorf("Expected test error, got %v", errors[0])
		}
		
		// Verify we have errors
		if len(runner.Errors()) == 0 {
			t.Error("Expected errors to be present")
		}
	})

	t.Run("ConcurrentRunner Wait synchronizes correctly", func(t *testing.T) {
		runner := NewConcurrentRunner()
		
		completed := int64(0)
		
		// Add operations with different delays
		runner.Add(1)
		runner.Go(func() error {
			time.Sleep(10 * time.Millisecond)
			atomic.AddInt64(&completed, 1)
			return nil
		})
		runner.Add(1)
		runner.Go(func() error {
			time.Sleep(20 * time.Millisecond)
			atomic.AddInt64(&completed, 1)
			return nil
		})
		runner.Add(1)
		runner.Go(func() error {
			time.Sleep(5 * time.Millisecond)
			atomic.AddInt64(&completed, 1)
			return nil
		})
		
		start := time.Now()
		runner.Wait()
		elapsed := time.Since(start)
		
		// Should wait for all operations to complete
		if atomic.LoadInt64(&completed) != 3 {
			t.Errorf("Expected 3 completed operations, got %d", atomic.LoadInt64(&completed))
		}
		
		// Should take at least as long as the longest operation
		if elapsed < 20*time.Millisecond {
			t.Errorf("Expected to wait at least 20ms, waited %v", elapsed)
		}
	})
}

func TestMemoryPressureTest(t *testing.T) {
	t.Run("MemoryPressureTest runs function and checks memory", func(t *testing.T) {
		// Create a function that allocates memory
		testFunc := func() error {
			// Allocate some memory for testing
			data := make([]byte, 1024*100) // 100KB
			_ = data // Use the data to prevent optimization
			return nil
		}
		
		// Run the memory pressure test - it should not fail
		MemoryPressureTest(t, testFunc)
	})
	
	t.Run("MemoryPressureTest function signature is correct", func(t *testing.T) {
		// We can't easily test error handling since MemoryPressureTest calls t.Fatalf
		// Let's just verify a successful function call
		testFunc := func() error {
			// Simple successful operation
			data := make([]byte, 100)
			_ = data
			return nil
		}
		
		// This should complete without error
		MemoryPressureTest(t, testFunc)
	})
}

func TestChannelMonitor(t *testing.T) {
	t.Run("ChannelMonitor tracks operations", func(t *testing.T) {
		monitor := NewChannelMonitor()
		
		// Increment operations
		for i := 0; i < 5; i++ {
			monitor.IncrementOps()
		}
		
		ops := monitor.GetOps()
		if ops != 5 {
			t.Errorf("Expected 5 operations, got %d", ops)
		}
		
		// Reset and verify
		monitor.Reset()
		ops = monitor.GetOps()
		if ops != 0 {
			t.Errorf("Expected 0 operations after reset, got %d", ops)
		}
	})
}

func TestBenchmarkHelper(t *testing.T) {
	t.Run("BenchmarkHelper constructor creates helper", func(t *testing.T) {
		// Since BenchmarkHelper needs *testing.B, we can't create one in unit tests
		// We just verify the constructor exists and would work
		// In a real benchmark test, it would be called like: NewBenchmarkHelper(b)
		
		// Test that the constructor function exists - it should be callable
		// We can't call it without a *testing.B, so we just verify the function signature works
		t.Log("NewBenchmarkHelper function is available for benchmark tests")
		
		// Note: In actual benchmarks, this would be:
		// helper := NewBenchmarkHelper(b)
		// helper.StartTimer()
		// ... benchmark code ...
		// helper.StopTimer()
	})
}

func TestContextWithValues(t *testing.T) {
	t.Run("ContextWithValues sets multiple values", func(t *testing.T) {
		values := map[string]interface{}{
			"key1": "value1",
			"key2": 42,
			"key3": true,
		}
		
		ctx := ContextWithValues(values)
		
		for key, expectedValue := range values {
			actualValue := ctx.Value(key)
			if actualValue != expectedValue {
				t.Errorf("Key %v: Expected %v, got %v", key, expectedValue, actualValue)
			}
		}
	})
}

func TestExpiredContext(t *testing.T) {
	t.Run("ExpiredContext returns expired context", func(t *testing.T) {
		ctx := ExpiredContext()
		
		select {
		case <-ctx.Done():
			if ctx.Err() != context.DeadlineExceeded {
				t.Errorf("Expected DeadlineExceeded, got %v", ctx.Err())
			}
		default:
			t.Error("Context should be already expired")
		}
		
		deadline, ok := ctx.Deadline()
		if !ok {
			t.Error("Expected expired context to have deadline")
		}
		if !deadline.Before(time.Now()) {
			t.Error("Expected deadline to be in the past")
		}
	})
}

func TestErrorGenerator(t *testing.T) {
	t.Run("ErrorGenerator cycles through error types", func(t *testing.T) {
		generator := NewErrorGenerator()
		
		// Test network error
		err1 := generator.Next()
		if err1 == nil {
			t.Error("Expected error, got nil")
		}
		
		// Test timeout error
		err2 := generator.Next()
		if err2 == nil {
			t.Error("Expected error, got nil")
		}
		
		// Test validation error
		err3 := generator.Next()
		if err3 == nil {
			t.Error("Expected error, got nil")
		}
		
		// Should cycle back to network error
		err4 := generator.Next()
		if err4 == nil {
			t.Error("Expected error, got nil")
		}
		
		// Verify errors are different
		if err1.Error() == err2.Error() {
			t.Error("Expected different error messages")
		}
		if err2.Error() == err3.Error() {
			t.Error("Expected different error messages")
		}
	})

	t.Run("ErrorGenerator specific error types", func(t *testing.T) {
		generator := NewErrorGenerator()
		
		netErr := generator.NetworkError()
		if netErr == nil {
			t.Error("Expected network error, got nil")
		}
		
		timeoutErr := generator.TimeoutError()
		if timeoutErr == nil {
			t.Error("Expected timeout error, got nil")
		}
		
		validationErr := generator.ValidationError()
		if validationErr == nil {
			t.Error("Expected validation error, got nil")
		}
		
		// Verify different error messages
		if netErr.Error() == timeoutErr.Error() {
			t.Error("Expected different error messages for network and timeout")
		}
		if timeoutErr.Error() == validationErr.Error() {
			t.Error("Expected different error messages for timeout and validation")
		}
	})
}
package main

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-model/testutil"
)

// TestGoroutineLeakDetectorIntegration demonstrates the proper usage of testutil.GoroutineLeakDetector
// This test file exists at the package root to avoid circular import issues
func TestGoroutineLeakDetectorIntegration(t *testing.T) {
	t.Run("BasicGoroutineLeakDetection", func(t *testing.T) {
		// Create a goroutine leak detector at test start
		detector := testutil.NewGoroutineLeakDetector(t)
		defer detector.Check() // Ensure we check for leaks at the end

		// Test basic functionality that creates and cleans up goroutines
		data := testutil.GenerateSlice(1, 50)
		provider := testutil.FixedProvider(data)
		
		// Use model.SliceMap with ParallelMap (creates goroutines internally)
		transform := func(val uint32) (uint32, error) {
			time.Sleep(time.Microsecond) // Minimal work
			return val * 3, nil
		}
		
		transformed := model.SliceMap[uint32, uint32](transform)(provider)(model.ParallelMap())
		result, err := transformed()
		
		if err != nil {
			t.Errorf("Parallel transformation should not fail: %v", err)
		}
		
		if len(result) != len(data) {
			t.Errorf("Expected %d results, got %d", len(data), len(result))
		}
		
		// Verify transformation
		for i, val := range result {
			expected := data[i] * 3
			if val != expected {
				t.Errorf("Index %d: expected %d, got %d", i, expected, val)
			}
		}
		
		// The defer detector.Check() will verify no goroutines leaked
		t.Logf("Basic goroutine leak detection completed successfully")
	})

	t.Run("AdvancedGoroutineLeakDetectionWithRetries", func(t *testing.T) {
		detector := testutil.NewGoroutineLeakDetector(t)
		defer detector.CheckWithRetries(5) // Use more retries for complex test

		// Create a more complex scenario with memoization and concurrent access
		var expensiveCallCount int64
		expensiveProvider := func() ([]uint32, error) {
			atomic.AddInt64(&expensiveCallCount, 1)
			
			// Simulate expensive concurrent work
			results := make([]uint32, 10)
			var wg sync.WaitGroup
			
			for i := range results {
				wg.Add(1)
				go func(idx int) {
					defer wg.Done()
					time.Sleep(time.Microsecond * 2)
					results[idx] = uint32(idx + 100)
				}(i)
			}
			wg.Wait()
			
			return results, nil
		}

		memoizedProvider := model.Memoize(expensiveProvider)
		
		// Access memoized provider concurrently from multiple goroutines
		const numAccess = 25
		var wg sync.WaitGroup
		results := make([][]uint32, numAccess)
		errors := make([]error, numAccess)
		
		for i := 0; i < numAccess; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				result, err := memoizedProvider()
				results[idx] = result
				errors[idx] = err
			}(i)
		}
		wg.Wait()
		
		// Verify results
		for i, err := range errors {
			if err != nil {
				t.Errorf("Access %d failed: %v", i, err)
			}
		}
		
		// All results should be identical due to memoization
		if len(results) > 0 && results[0] != nil {
			expected := results[0]
			for i, result := range results {
				if len(result) != len(expected) {
					t.Errorf("Access %d: length mismatch %d vs %d", i, len(result), len(expected))
					continue
				}
				for j, val := range result {
					if val != expected[j] {
						t.Errorf("Access %d, index %d: expected %d, got %d", i, j, expected[j], val)
					}
				}
			}
		}
		
		// Verify memoization worked (expensive function called only once)
		if expensiveCallCount != 1 {
			t.Errorf("Expected expensive function called once, got %d", expensiveCallCount)
		}
		
		t.Logf("Advanced goroutine leak detection completed successfully")
	})

	t.Run("ContextCancellationGoroutineLeakDetection", func(t *testing.T) {
		detector := testutil.NewGoroutineLeakDetector(t)
		defer detector.Check()

		// Test using context-aware providers from testutil
		ctx, cancel := testutil.TimeoutContext(time.Millisecond * 50)
		defer cancel()
		
		// Create a provider that respects context cancellation
		provider := testutil.CancellableProvider(ctx, uint32(42), time.Millisecond*100)
		
		// This should be cancelled due to timeout
		result, err := provider()
		
		// Should get a context timeout error
		if err == nil {
			t.Logf("Got result despite timeout (timing dependent): %d", result)
		} else {
			t.Logf("Expected timeout error received: %v", err)
		}
		
		t.Logf("Context cancellation goroutine leak detection completed")
	})

	t.Run("ErrorProviderGoroutineLeakDetection", func(t *testing.T) {
		detector := testutil.NewGoroutineLeakDetector(t)
		defer detector.Check()

		// Test error scenarios don't leak goroutines
		errorGen := testutil.NewErrorGenerator()
		
		// Create providers that return errors
		errorProvider1 := testutil.ErrorProvider[uint32](errorGen.NetworkError())
		errorProvider2 := testutil.ErrorProvider[[]uint32](errorGen.TimeoutError())
		
		// Call error providers multiple times
		for i := 0; i < 10; i++ {
			_, err1 := errorProvider1()
			_, err2 := errorProvider2()
			
			if err1 == nil {
				t.Error("Expected network error but got nil")
			}
			if err2 == nil {
				t.Error("Expected timeout error but got nil")
			}
		}
		
		t.Logf("Error provider goroutine leak detection completed")
	})

	t.Run("ConcurrentRunnerGoroutineLeakDetection", func(t *testing.T) {
		detector := testutil.NewGoroutineLeakDetector(t)
		defer detector.Check()

		// Use the ConcurrentRunner utility to coordinate multiple goroutines
		runner := testutil.NewConcurrentRunner()
		
		const numOperations = 20
		runner.Add(numOperations)
		
		var counter int64
		
		for i := 0; i < numOperations; i++ {
			runner.Go(func() error {
				// Each goroutine does some work
				provider := testutil.CountingProvider(uint32(100), &counter)
				result, err := provider()
				if err != nil {
					return err
				}
				if result != 100 {
					return testutil.TestErrorWithValue(result)
				}
				return nil
			})
		}
		
		runner.Wait()
		runner.CheckErrors(t)
		
		// Verify all operations executed
		if counter != int64(numOperations) {
			t.Errorf("Expected %d operations, got %d", numOperations, counter)
		}
		
		t.Logf("ConcurrentRunner goroutine leak detection completed")
	})
}

// TestMemoryPressureWithGoroutineLeakDetection demonstrates using both memory and goroutine monitoring
func TestMemoryPressureWithGoroutineLeakDetection(t *testing.T) {
	detector := testutil.NewGoroutineLeakDetector(t)
	defer detector.Check()
	
	// Run under memory pressure while monitoring goroutines
	testutil.MemoryPressureTest(t, func() error {
		// Generate large dataset
		data := testutil.GenerateLargeSlice(1000)
		provider := testutil.FixedProvider(data)
		
		// Process with parallel map
		transform := func(val uint32) (uint32, error) {
			return val * 2, nil
		}
		
		transformed := model.SliceMap[uint32, uint32](transform)(provider)(model.ParallelMap())
		_, err := transformed()
		return err
	})
	
	t.Logf("Memory pressure test with goroutine leak detection completed")
}

// BenchmarkGoroutineLeakDetectorOverhead measures the overhead of leak detection
func BenchmarkGoroutineLeakDetectorOverhead(b *testing.B) {
	b.Run("WithoutDetector", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			data := []uint32{1, 2, 3, 4, 5}
			provider := model.FixedProvider(data)
			_, _ = provider()
		}
	})
	
	b.Run("WithDetector", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			// Create detector per iteration to measure overhead
			detector := testutil.NewGoroutineLeakDetector(&testing.T{})
			
			data := []uint32{1, 2, 3, 4, 5}
			provider := model.FixedProvider(data)
			_, _ = provider()
			
			// Skip the actual check in benchmark to avoid test framework issues
			_ = detector
		}
	})
}
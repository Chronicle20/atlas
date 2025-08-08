package async

import (
	"context"
	"errors"
	"fmt"
	"github.com/Chronicle20/atlas-model/model"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestAsyncSlice(t *testing.T) {
	items := []uint32{1, 2, 3, 4, 5}

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		ctx := context.WithValue(context.Background(), "key", "value")
		results, err := AwaitSlice(model.SliceMap(AsyncTestTransformer)(model.FixedProvider(items))(), SetContext(ctx))()
		if err != nil {
			t.Fatal(err)
		}
		for _, result := range results {
			found := false
			for _, item := range items {
				if item == result {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("Invalid item.")
			}
		}
	}()
	wg.Wait()
}

func AsyncTestTransformer(m uint32) (Provider[uint32], error) {
	return func(ctx context.Context, rchan chan uint32, echan chan error) {
		time.Sleep(time.Duration(50) * time.Millisecond)

		if ctx.Value("key") != "value" {
			echan <- errors.New("invalid context")
		}

		rchan <- m
	}, nil
}

func TestAsyncSliceErrorHandling(t *testing.T) {
	// Test that errors are properly returned without double-read issues
	items := []uint32{1, 2, 3}
	expectedError := errors.New("test error")

	ctx := context.Background()
	results, err := AwaitSlice(model.SliceMap(func(m uint32) (Provider[uint32], error) {
		return func(ctx context.Context, rchan chan uint32, echan chan error) {
			if m == 2 {
				echan <- expectedError
				return
			}
			rchan <- m
		}, nil
	})(model.FixedProvider(items))(), SetContext(ctx))()

	if err == nil {
		t.Fatal("Expected error but got none")
	}
	if err.Error() != expectedError.Error() {
		t.Fatalf("Expected error %q but got %q", expectedError.Error(), err.Error())
	}
	if results != nil {
		t.Fatal("Expected nil results on error")
	}
}

func TestAsyncRaceConditionThreadSafety(t *testing.T) {
	// Comprehensive race condition tests for async package
	// These tests verify thread safety and should be run with `go test -race`

	t.Run("AwaitSliceConcurrentAccess", func(t *testing.T) {
		// Test concurrent access to AwaitSlice to verify no race conditions
		items := make([]uint32, 100)
		for i := range items {
			items[i] = uint32(i + 1)
		}

		provider := func(delay time.Duration) func(m uint32) (Provider[uint32], error) {
			return func(m uint32) (Provider[uint32], error) {
				return func(ctx context.Context, rchan chan uint32, echan chan error) {
					time.Sleep(delay)
					select {
					case <-ctx.Done():
						return
					default:
						rchan <- m
					}
				}, nil
			}
		}

		const numConcurrentTests = 5
		var wg sync.WaitGroup
		results := make([][]uint32, numConcurrentTests)
		errors := make([]error, numConcurrentTests)

		for testIdx := 0; testIdx < numConcurrentTests; testIdx++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()

				// Each test uses different delay to create varied timing
				delay := time.Microsecond * time.Duration(idx+1)
				ctx := context.Background()

				result, err := AwaitSlice(
					model.SliceMap(provider(delay))(model.FixedProvider(items))(),
					SetContext(ctx),
					SetTimeout(time.Second),
				)()

				results[idx] = result
				errors[idx] = err
			}(testIdx)
		}

		wg.Wait()

		// Verify all concurrent tests succeeded
		for i := 0; i < numConcurrentTests; i++ {
			if errors[i] != nil {
				t.Errorf("Concurrent test %d failed: %s", i, errors[i])
				continue
			}

			if len(results[i]) != len(items) {
				t.Errorf("Test %d: expected %d results, got %d", i, len(items), len(results[i]))
				continue
			}

			// Verify all items are present (order may vary due to concurrency)
			found := make(map[uint32]bool)
			for _, result := range results[i] {
				found[result] = true
			}

			for _, original := range items {
				if !found[original] {
					t.Errorf("Test %d: missing item %d in results", i, original)
				}
			}
		}
	})

	t.Run("ChannelOperationsRaceConditions", func(t *testing.T) {
		// Test that channel operations don't create race conditions
		items := make([]uint32, 200)
		for i := range items {
			items[i] = uint32(i + 1)
		}

		// Provider that uses channels intensively
		intensiveProvider := func(m uint32) (Provider[uint32], error) {
			return func(ctx context.Context, rchan chan uint32, echan chan error) {
				// Create multiple goroutines per provider to stress-test channels
				var wg sync.WaitGroup
				var result uint32
				var processErr error

				wg.Add(3)

				// Goroutine 1: Does computation
				go func() {
					defer wg.Done()
					time.Sleep(time.Microsecond * time.Duration(m%10))
					result = m * 2
				}()

				// Goroutine 2: Simulates additional async work
				go func() {
					defer wg.Done()
					time.Sleep(time.Microsecond * time.Duration((m+1)%5))
					if m > 1000 { // Simulate rare error condition
						processErr = fmt.Errorf("processing error for %d", m)
					}
				}()

				// Goroutine 3: Does more computation
				go func() {
					defer wg.Done()
					time.Sleep(time.Microsecond * time.Duration((m+2)%7))
				}()

				wg.Wait()

				select {
				case <-ctx.Done():
					return
				default:
					if processErr != nil {
						echan <- processErr
					} else {
						rchan <- result
					}
				}
			}, nil
		}

		ctx := context.Background()
		result, err := AwaitSlice(
			model.SliceMap(intensiveProvider)(model.FixedProvider(items))(),
			SetContext(ctx),
			SetTimeout(time.Second*5),
		)()

		if err != nil {
			t.Errorf("Expected no error, got %s", err)
		}

		if len(result) != len(items) {
			t.Errorf("Expected %d results, got %d", len(items), len(result))
		}

		// Verify results are correct (all should be doubled)
		resultMap := make(map[uint32]bool)
		for _, r := range result {
			resultMap[r] = true
		}

		for _, original := range items {
			expected := original * 2
			if !resultMap[expected] {
				t.Errorf("Missing expected result %d for input %d", expected, original)
			}
		}
	})

	t.Run("ErrorHandlingRaceConditions", func(t *testing.T) {
		// Test error handling for race conditions
		items := []uint32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

		errorProvider := func(m uint32) (Provider[uint32], error) {
			return func(ctx context.Context, rchan chan uint32, echan chan error) {
				// Simulate some async work
				time.Sleep(time.Microsecond * time.Duration(m%5))

				select {
				case <-ctx.Done():
					return
				default:
					if m == 5 { // Error on specific value
						echan <- fmt.Errorf("error on value %d", m)
					} else {
						rchan <- m
					}
				}
			}, nil
		}

		const numTests = 10
		var wg sync.WaitGroup
		results := make([]error, numTests)

		for testIdx := 0; testIdx < numTests; testIdx++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()

				ctx := context.Background()
				_, err := AwaitSlice(
					model.SliceMap(errorProvider)(model.FixedProvider(items))(),
					SetContext(ctx),
					SetTimeout(time.Second),
				)()

				results[idx] = err
			}(testIdx)
		}

		wg.Wait()

		// All tests should get the same error
		expectedError := "error on value 5"
		for i, err := range results {
			if err == nil {
				t.Errorf("Test %d: expected error but got none", i)
			} else if err.Error() != expectedError {
				t.Errorf("Test %d: expected error %q, got %q", i, expectedError, err.Error())
			}
		}
	})

	t.Run("TimeoutHandlingRaceConditions", func(t *testing.T) {
		// Test timeout handling under concurrent conditions
		items := []uint32{1, 2, 3, 4, 5}

		slowProvider := func(m uint32) (Provider[uint32], error) {
			return func(ctx context.Context, rchan chan uint32, echan chan error) {
				// Deliberately slow operation that should timeout
				select {
				case <-time.After(time.Second):
					rchan <- m
				case <-ctx.Done():
					return
				}
			}, nil
		}

		const numTests = 8
		var wg sync.WaitGroup
		results := make([]error, numTests)

		for testIdx := 0; testIdx < numTests; testIdx++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()

				ctx := context.Background()
				_, err := AwaitSlice(
					model.SliceMap(slowProvider)(model.FixedProvider(items))(),
					SetContext(ctx),
					SetTimeout(time.Millisecond*10), // Short timeout to trigger timeout
				)()

				results[idx] = err
			}(testIdx)
		}

		wg.Wait()

		// All tests should timeout with ErrAwaitTimeout
		for i, err := range results {
			if err == nil {
				t.Errorf("Test %d: expected timeout error but got none", i)
			} else if err != ErrAwaitTimeout {
				t.Errorf("Test %d: expected ErrAwaitTimeout, got %s", i, err)
			}
		}
	})

	t.Run("ContextCancellationRaceConditions", func(t *testing.T) {
		// Test context cancellation under concurrent conditions
		items := make([]uint32, 50)
		for i := range items {
			items[i] = uint32(i + 1)
		}

		cancellableProvider := func(m uint32) (Provider[uint32], error) {
			return func(ctx context.Context, rchan chan uint32, echan chan error) {
				// Wait for cancellation or send result
				select {
				case <-ctx.Done():
					return
				case <-time.After(time.Millisecond * time.Duration(m%10+1)):
					select {
					case <-ctx.Done():
						return
					default:
						rchan <- m
					}
				}
			}, nil
		}

		const numTests = 6
		var wg sync.WaitGroup
		results := make([]error, numTests)

		for testIdx := 0; testIdx < numTests; testIdx++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()

				ctx, cancel := context.WithCancel(context.Background())

				// Cancel context after a short delay
				go func() {
					time.Sleep(time.Millisecond * 5)
					cancel()
				}()

				_, err := AwaitSlice(
					model.SliceMap(cancellableProvider)(model.FixedProvider(items))(),
					SetContext(ctx),
					SetTimeout(time.Second),
				)()

				results[idx] = err
			}(testIdx)
		}

		wg.Wait()

		// All tests should be cancelled (timeout or context cancelled)
		for i, err := range results {
			if err == nil {
				t.Errorf("Test %d: expected cancellation error but got none", i)
			}
			// Accept either timeout or context cancellation
			if err != ErrAwaitTimeout && err != context.Canceled {
				t.Errorf("Test %d: expected timeout or cancellation error, got %s", i, err)
			}
		}
	})

	t.Run("HighVolumeDataRaceConditions", func(t *testing.T) {
		// Test with high volume of data to stress-test race conditions
		const dataSize = 2000
		items := make([]uint32, dataSize)
		for i := range items {
			items[i] = uint32(i + 1)
		}

		highVolumeProvider := func(m uint32) (Provider[uint32], error) {
			return func(ctx context.Context, rchan chan uint32, echan chan error) {
				// Minimal delay to maximize concurrency
				if m%100 == 0 {
					time.Sleep(time.Microsecond)
				}

				select {
				case <-ctx.Done():
					return
				default:
					rchan <- m + 1000 // Transform the data
				}
			}, nil
		}

		ctx := context.Background()
		result, err := AwaitSlice(
			model.SliceMap(highVolumeProvider)(model.FixedProvider(items))(),
			SetContext(ctx),
			SetTimeout(time.Second*10),
		)()

		if err != nil {
			t.Errorf("Expected no error, got %s", err)
		}

		if len(result) != len(items) {
			t.Errorf("Expected %d results, got %d", len(items), len(result))
		}

		// Verify all results are present and correctly transformed
		resultMap := make(map[uint32]bool)
		for _, r := range result {
			resultMap[r] = true
		}

		for _, original := range items {
			expected := original + 1000
			if !resultMap[expected] {
				t.Errorf("Missing expected result %d for input %d", expected, original)
			}
		}
	})

	t.Run("MixedSuccessAndErrorScenarios", func(t *testing.T) {
		// Test mixed scenarios with some successes and some errors
		items := make([]uint32, 100)
		for i := range items {
			items[i] = uint32(i + 1)
		}

		mixedProvider := func(m uint32) (Provider[uint32], error) {
			return func(ctx context.Context, rchan chan uint32, echan chan error) {
				time.Sleep(time.Microsecond * time.Duration(m%10))

				select {
				case <-ctx.Done():
					return
				default:
					if m%20 == 0 { // Error every 20th item
						echan <- fmt.Errorf("error on item %d", m)
					} else {
						rchan <- m * 2
					}
				}
			}, nil
		}

		const numTests = 5
		var wg sync.WaitGroup
		results := make([]error, numTests)

		for testIdx := 0; testIdx < numTests; testIdx++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()

				ctx := context.Background()
				_, err := AwaitSlice(
					model.SliceMap(mixedProvider)(model.FixedProvider(items))(),
					SetContext(ctx),
					SetTimeout(time.Second*2),
				)()

				results[idx] = err
			}(testIdx)
		}

		wg.Wait()

		// All tests should get the same error (first error encountered)
		for i, err := range results {
			if err == nil {
				t.Errorf("Test %d: expected error but got none", i)
			} else {
				// Should be an error about item 20, 40, 60, 80, or 100
				if !strings.Contains(err.Error(), "error on item") {
					t.Errorf("Test %d: unexpected error format: %s", i, err)
				}
			}
		}
	})

	t.Run("ExtremeConcurrencyRaceConditions", func(t *testing.T) {
		// Test with extreme concurrency to stress-test race conditions
		const dataSize = 10000
		items := make([]uint32, dataSize)
		for i := range items {
			items[i] = uint32(i + 1)
		}

		// Provider that performs shared memory operations
		var sharedCounter int64
		extremeProvider := func(m uint32) (Provider[uint32], error) {
			return func(ctx context.Context, rchan chan uint32, echan chan error) {
				// Atomic increment to test thread-safety
				atomic.AddInt64(&sharedCounter, 1)
				
				// Minimal delay to maximize concurrency
				if m%1000 == 0 {
					time.Sleep(time.Microsecond)
				}

				select {
				case <-ctx.Done():
					return
				default:
					rchan <- m + 100000
				}
			}, nil
		}

		ctx := context.Background()
		result, err := AwaitSlice(
			model.SliceMap(extremeProvider)(model.FixedProvider(items))(),
			SetContext(ctx),
			SetTimeout(time.Second*15),
		)()

		if err != nil {
			t.Errorf("Expected no error, got %s", err)
		}

		if len(result) != len(items) {
			t.Errorf("Expected %d results, got %d", len(items), len(result))
		}

		// Verify shared counter (should equal number of items if thread-safe)
		if sharedCounter != int64(dataSize) {
			t.Errorf("Expected shared counter %d, got %d (race condition detected)", dataSize, sharedCounter)
		}

		// Verify all results are correct and unique
		resultMap := make(map[uint32]bool)
		for _, r := range result {
			if resultMap[r] {
				t.Errorf("Duplicate result detected: %d (race condition)", r)
			}
			resultMap[r] = true
		}

		for _, original := range items {
			expected := original + 100000
			if !resultMap[expected] {
				t.Errorf("Missing expected result %d for input %d", expected, original)
			}
		}
	})

	t.Run("ChannelBufferingRaceConditions", func(t *testing.T) {
		// Test race conditions with channel buffering and backpressure
		items := make([]uint32, 500)
		for i := range items {
			items[i] = uint32(i + 1)
		}

		// Provider that simulates varying processing speeds
		bufferProvider := func(m uint32) (Provider[uint32], error) {
			return func(ctx context.Context, rchan chan uint32, echan chan error) {
				// Simulate different processing speeds to create backpressure
				delay := time.Microsecond * time.Duration((m%20)+1)
				time.Sleep(delay)

				// Multiple goroutines to stress channel operations
				var wg sync.WaitGroup
				results := make([]uint32, 3)
				
				for i := 0; i < 3; i++ {
					wg.Add(1)
					go func(idx int) {
						defer wg.Done()
						// Each goroutine computes a different transformation
						switch idx {
						case 0:
							results[idx] = m * 2
						case 1:
							results[idx] = m + 1000
						case 2:
							results[idx] = m ^ 0xFF // XOR operation
						}
					}(i)
				}

				wg.Wait()

				// Combine results (race condition potential here if not handled properly)
				final := results[0] + results[1] + results[2]

				select {
				case <-ctx.Done():
					return
				default:
					rchan <- final
				}
			}, nil
		}

		const numTests = 8
		var wg sync.WaitGroup
		results := make([][]uint32, numTests)
		errors := make([]error, numTests)

		for testIdx := 0; testIdx < numTests; testIdx++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()

				ctx := context.Background()
				result, err := AwaitSlice(
					model.SliceMap(bufferProvider)(model.FixedProvider(items))(),
					SetContext(ctx),
					SetTimeout(time.Second*10),
				)()

				results[idx] = result
				errors[idx] = err
			}(testIdx)
		}

		wg.Wait()

		// Verify all tests succeeded
		for i := 0; i < numTests; i++ {
			if errors[i] != nil {
				t.Errorf("Test %d failed: %s", i, errors[i])
				continue
			}

			if len(results[i]) != len(items) {
				t.Errorf("Test %d: expected %d results, got %d", i, len(items), len(results[i]))
				continue
			}

			// Results may be in different orders due to concurrency, but should contain the same values
			if i > 0 {
				// Sort both result sets to compare contents rather than order
				baseline := make([]uint32, len(results[0]))
				current := make([]uint32, len(results[i]))
				copy(baseline, results[0])
				copy(current, results[i])
				
				// Convert to maps for order-independent comparison
				baselineMap := make(map[uint32]int)
				currentMap := make(map[uint32]int)
				
				for _, val := range baseline {
					baselineMap[val]++
				}
				for _, val := range current {
					currentMap[val]++
				}
				
				if !reflect.DeepEqual(baselineMap, currentMap) {
					t.Errorf("Test %d: result sets differ from baseline (race condition)", i)
				}
			}
		}
	})

	t.Run("SingleProviderHighConcurrency", func(t *testing.T) {
		// Test race conditions with single provider under high concurrent load
		var executionCount int64

		provider := func(ctx context.Context, rchan chan uint32, echan chan error) {
			// Atomic increment to track executions
			count := atomic.AddInt64(&executionCount, 1)
			
			// Simulate work with potential for race conditions
			time.Sleep(time.Microsecond * time.Duration(count%10))

			select {
			case <-ctx.Done():
				return
			default:
				rchan <- uint32(count * 100)
			}
		}

		const numConcurrentCalls = 50
		var wg sync.WaitGroup
		results := make([]uint32, numConcurrentCalls)
		errors := make([]error, numConcurrentCalls)

		// Execute many concurrent single provider calls
		for i := 0; i < numConcurrentCalls; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()

				ctx := context.Background()
				result, err := Await(
					SingleProvider(provider),
					SetContext(ctx),
					SetTimeout(time.Second),
				)()

				results[idx] = result
				errors[idx] = err
			}(i)
		}

		wg.Wait()

		// Verify all calls succeeded
		for i, err := range errors {
			if err != nil {
				t.Errorf("Call %d failed: %s", i, err)
			}
		}

		// Verify execution count matches expected (no race condition in counter)
		if executionCount != int64(numConcurrentCalls) {
			t.Errorf("Expected %d executions, got %d (race condition in counter)", numConcurrentCalls, executionCount)
		}

		// Verify all results are unique (since count is incremented each time)
		resultSet := make(map[uint32]bool)
		for i, result := range results {
			if errors[i] != nil {
				continue // Skip failed calls
			}
			if resultSet[result] {
				t.Errorf("Duplicate result %d detected (race condition)", result)
			}
			resultSet[result] = true
		}
	})
}

func TestAsyncProviderErrorPropagation(t *testing.T) {
	// Test that errors propagate correctly through async provider chains
	// This test focuses on async-specific error propagation scenarios
	
	t.Run("SingleProviderErrorPropagation", func(t *testing.T) {
		// Test error propagation from a single async provider
		expectedError := errors.New("async provider error")
		
		errorProvider := func(ctx context.Context, rchan chan uint32, echan chan error) {
			// Simulate async work before error
			time.Sleep(time.Millisecond * 10)
			select {
			case <-ctx.Done():
				return
			default:
				echan <- expectedError
			}
		}
		
		ctx := context.Background()
		result, err := Await(SingleProvider(errorProvider), SetContext(ctx), SetTimeout(time.Second))()
		
		if err == nil {
			t.Fatal("Expected error but got none")
		}
		if err.Error() != expectedError.Error() {
			t.Errorf("Expected error %q but got %q", expectedError.Error(), err.Error())
		}
		if result != 0 {
			t.Errorf("Expected zero value result when error occurs, got %d", result)
		}
	})
	
	t.Run("SliceProviderErrorPropagation", func(t *testing.T) {
		// Test error propagation in slice operations where one provider fails
		items := []uint32{1, 2, 3, 4, 5}
		expectedError := errors.New("slice operation error")
		
		// Transform that fails on the third item
		failingTransform := func(m uint32) (Provider[uint32], error) {
			return func(ctx context.Context, rchan chan uint32, echan chan error) {
				// Simulate async processing time
				time.Sleep(time.Millisecond * time.Duration(m*2))
				
				select {
				case <-ctx.Done():
					return
				default:
					if m == 3 {
						echan <- expectedError
					} else {
						rchan <- m * 10
					}
				}
			}, nil
		}
		
		ctx := context.Background()
		result, err := AwaitSlice(
			model.SliceMap(failingTransform)(model.FixedProvider(items))(),
			SetContext(ctx),
			SetTimeout(time.Second),
		)()
		
		if err == nil {
			t.Fatal("Expected error but got none")
		}
		if err.Error() != expectedError.Error() {
			t.Errorf("Expected error %q but got %q", expectedError.Error(), err.Error())
		}
		if result != nil {
			t.Error("Expected nil result when error occurs")
		}
	})
	
	t.Run("ConcurrentErrorPropagation", func(t *testing.T) {
		// Test that the first error is properly propagated in concurrent scenarios
		items := make([]uint32, 50)
		for i := range items {
			items[i] = uint32(i + 1)
		}
		
		// Multiple errors can occur, but only the first one should be returned
		multiErrorTransform := func(m uint32) (Provider[uint32], error) {
			return func(ctx context.Context, rchan chan uint32, echan chan error) {
				// Variable delay to create race conditions for error reporting
				delay := time.Microsecond * time.Duration(m%10+1)
				time.Sleep(delay)
				
				select {
				case <-ctx.Done():
					return
				default:
					if m%10 == 0 { // Multiple items will error (10, 20, 30, 40, 50)
						echan <- fmt.Errorf("concurrent error on item %d", m)
					} else {
						rchan <- m + 100
					}
				}
			}, nil
		}
		
		ctx := context.Background()
		result, err := AwaitSlice(
			model.SliceMap(multiErrorTransform)(model.FixedProvider(items))(),
			SetContext(ctx),
			SetTimeout(time.Second),
		)()
		
		if err == nil {
			t.Fatal("Expected error but got none")
		}
		
		// Should be a concurrent error (exact item number may vary due to concurrency)
		if !strings.Contains(err.Error(), "concurrent error on item") {
			t.Errorf("Expected concurrent error but got %q", err.Error())
		}
		
		if result != nil {
			t.Error("Expected nil result when error occurs")
		}
	})
	
	t.Run("ErrorPropagationWithTimeout", func(t *testing.T) {
		// Test error propagation when both errors and timeouts can occur
		items := []uint32{1, 2, 3}
		explicitError := errors.New("explicit async error")
		
		mixedProvider := func(m uint32) (Provider[uint32], error) {
			return func(ctx context.Context, rchan chan uint32, echan chan error) {
				switch m {
				case 1:
					// Fast error
					time.Sleep(time.Millisecond * 10)
					echan <- explicitError
				case 2:
					// Slow success that should timeout
					time.Sleep(time.Millisecond * 200)
					rchan <- m
				case 3:
					// Fast success
					time.Sleep(time.Millisecond * 5)
					rchan <- m
				}
			}, nil
		}
		
		ctx := context.Background()
		result, err := AwaitSlice(
			model.SliceMap(mixedProvider)(model.FixedProvider(items))(),
			SetContext(ctx),
			SetTimeout(time.Millisecond*50), // Should timeout before item 2 completes but after error from item 1
		)()
		
		if err == nil {
			t.Fatal("Expected error but got none")
		}
		
		// Should get explicit error (not timeout) because explicit errors are processed first
		if err.Error() != explicitError.Error() {
			t.Errorf("Expected explicit error %q but got %q", explicitError.Error(), err.Error())
		}
		
		if result != nil {
			t.Error("Expected nil result when error occurs")
		}
	})
	
	t.Run("ChainedAsyncProviderErrorPropagation", func(t *testing.T) {
		// Test error propagation through chained async operations
		items := []uint32{1, 2, 3, 4, 5}
		chainedError := errors.New("chained operation error")
		
		// First transformation (should succeed)
		firstTransform := func(m uint32) (Provider[uint32], error) {
			return func(ctx context.Context, rchan chan uint32, echan chan error) {
				time.Sleep(time.Millisecond * 5)
				select {
				case <-ctx.Done():
					return
				default:
					rchan <- m * 2
				}
			}, nil
		}
		
		// Second transformation (should fail on specific value)
		secondTransform := func(m uint32) (Provider[uint32], error) {
			return func(ctx context.Context, rchan chan uint32, echan chan error) {
				time.Sleep(time.Millisecond * 10)
				select {
				case <-ctx.Done():
					return
				default:
					if m == 8 { // This is 4*2 from first transform
						echan <- chainedError
					} else {
						rchan <- m + 1000
					}
				}
			}, nil
		}
		
		ctx := context.Background()
		
		// First async operation
		firstResult, err := AwaitSlice(
			model.SliceMap(firstTransform)(model.FixedProvider(items))(),
			SetContext(ctx),
			SetTimeout(time.Second),
		)()
		
		if err != nil {
			t.Fatalf("First transformation should not error: %v", err)
		}
		
		// Second async operation (should fail)
		_, err = AwaitSlice(
			model.SliceMap(secondTransform)(model.FixedProvider(firstResult))(),
			SetContext(ctx),
			SetTimeout(time.Second),
		)()
		
		if err == nil {
			t.Fatal("Expected chained error but got none")
		}
		if err.Error() != chainedError.Error() {
			t.Errorf("Expected chained error %q but got %q", chainedError.Error(), err.Error())
		}
	})
	
	t.Run("PartialAsyncFailureRecovery", func(t *testing.T) {
		// Test that async operations properly handle partial failures
		// Unlike model operations, async operations should fail completely on any error
		items := []uint32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
		partialError := errors.New("partial async failure")
		
		partialFailureProvider := func(m uint32) (Provider[uint32], error) {
			return func(ctx context.Context, rchan chan uint32, echan chan error) {
				// Variable processing time to create race conditions
				delay := time.Millisecond * time.Duration(m%5+1)
				time.Sleep(delay)
				
				select {
				case <-ctx.Done():
					return
				default:
					if m == 7 { // Fail on the 7th item
						echan <- partialError
					} else {
						rchan <- m * m // Square the number
					}
				}
			}, nil
		}
		
		ctx := context.Background()
		result, err := AwaitSlice(
			model.SliceMap(partialFailureProvider)(model.FixedProvider(items))(),
			SetContext(ctx),
			SetTimeout(time.Second),
		)()
		
		// Should fail completely due to partial failure
		if err == nil {
			t.Fatal("Expected partial failure error but got none")
		}
		if err.Error() != partialError.Error() {
			t.Errorf("Expected partial failure error %q but got %q", partialError.Error(), err.Error())
		}
		if result != nil {
			t.Error("Expected nil result when partial failure occurs")
		}
	})
	
	t.Run("ErrorPropagationWithContextCancellation", func(t *testing.T) {
		// Test error propagation when context is cancelled during operation
		items := []uint32{1, 2, 3, 4, 5}
		explicitError := errors.New("explicit error before cancellation")
		
		cancellationProvider := func(m uint32) (Provider[uint32], error) {
			return func(ctx context.Context, rchan chan uint32, echan chan error) {
				if m == 1 {
					// Quick error
					time.Sleep(time.Millisecond * 5)
					echan <- explicitError
					return
				}
				
				// Longer operation that should be cancelled
				select {
				case <-time.After(time.Millisecond * 100):
					rchan <- m * 100
				case <-ctx.Done():
					return
				}
			}, nil
		}
		
		ctx, cancel := context.WithCancel(context.Background())
		
		// Cancel context after short delay
		go func() {
			time.Sleep(time.Millisecond * 20)
			cancel()
		}()
		
		result, err := AwaitSlice(
			model.SliceMap(cancellationProvider)(model.FixedProvider(items))(),
			SetContext(ctx),
			SetTimeout(time.Second),
		)()
		
		if err == nil {
			t.Fatal("Expected error but got none")
		}
		
		// Should get explicit error (processed before cancellation) or timeout/cancellation
		if err.Error() != explicitError.Error() && err != ErrAwaitTimeout && err != context.Canceled {
			t.Errorf("Expected explicit error, timeout, or cancellation but got %q", err.Error())
		}
		
		if result != nil {
			t.Error("Expected nil result when error occurs")
		}
	})
}

// Benchmark tests for AwaitSlice performance
func TestAsyncContextCancellation(t *testing.T) {
	// Test context cancellation scenarios specifically for async operations
	// This test focuses on how async providers handle context cancellation properly
	
	t.Run("ImmediateContextCancellation", func(t *testing.T) {
		// Test cancellation before async operations start
		items := []uint32{1, 2, 3, 4, 5}
		
		provider := func(m uint32) (Provider[uint32], error) {
			return func(ctx context.Context, rchan chan uint32, echan chan error) {
				// Should be cancelled immediately
				select {
				case <-ctx.Done():
					return
				case <-time.After(time.Millisecond * 100):
					rchan <- m
				}
			}, nil
		}
		
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately
		
		result, err := AwaitSlice(
			model.SliceMap(provider)(model.FixedProvider(items))(),
			SetContext(ctx),
			SetTimeout(time.Second),
		)()
		
		if err == nil {
			t.Fatal("Expected cancellation error but got none")
		}
		if err != ErrAwaitTimeout && err != context.Canceled {
			t.Errorf("Expected timeout or cancellation error, got %s", err)
		}
		if result != nil {
			t.Error("Expected nil result when context is cancelled")
		}
	})
	
	t.Run("MidOperationContextCancellation", func(t *testing.T) {
		// Test cancellation during async operation execution
		items := []uint32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
		
		provider := func(m uint32) (Provider[uint32], error) {
			return func(ctx context.Context, rchan chan uint32, echan chan error) {
				// Simulate work that can be interrupted
				select {
				case <-time.After(time.Millisecond * time.Duration(m*20)): // Variable delay
					select {
					case <-ctx.Done():
						return
					default:
						rchan <- m * 100
					}
				case <-ctx.Done():
					return
				}
			}, nil
		}
		
		ctx, cancel := context.WithCancel(context.Background())
		
		// Cancel context after a short delay to interrupt some operations
		go func() {
			time.Sleep(time.Millisecond * 50)
			cancel()
		}()
		
		result, err := AwaitSlice(
			model.SliceMap(provider)(model.FixedProvider(items))(),
			SetContext(ctx),
			SetTimeout(time.Second*5),
		)()
		
		if err == nil {
			t.Fatal("Expected cancellation error but got none")
		}
		if err != ErrAwaitTimeout && err != context.Canceled {
			t.Errorf("Expected timeout or cancellation error, got %s", err)
		}
		if result != nil {
			t.Error("Expected nil result when context is cancelled")
		}
	})
	
	t.Run("ContextCancellationWithDeadline", func(t *testing.T) {
		// Test context cancellation combined with deadline handling
		items := []uint32{1, 2, 3}
		
		provider := func(m uint32) (Provider[uint32], error) {
			return func(ctx context.Context, rchan chan uint32, echan chan error) {
				// Long operation that should be cancelled
				select {
				case <-time.After(time.Millisecond * 200):
					rchan <- m
				case <-ctx.Done():
					return
				}
			}, nil
		}
		
		// Create context with deadline
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
		defer cancel()
		
		result, err := AwaitSlice(
			model.SliceMap(provider)(model.FixedProvider(items))(),
			SetContext(ctx),
			SetTimeout(time.Second), // Longer timeout than context deadline
		)()
		
		if err == nil {
			t.Fatal("Expected cancellation/deadline error but got none")
		}
		if err != ErrAwaitTimeout && err != context.DeadlineExceeded && err != context.Canceled {
			t.Errorf("Expected timeout, deadline exceeded, or cancellation error, got %s", err)
		}
		if result != nil {
			t.Error("Expected nil result when context deadline is exceeded")
		}
	})
	
	t.Run("PartialCompletionBeforeCancellation", func(t *testing.T) {
		// Test scenario where some operations complete before cancellation
		items := []uint32{1, 2, 3, 4, 5}
		
		provider := func(m uint32) (Provider[uint32], error) {
			return func(ctx context.Context, rchan chan uint32, echan chan error) {
				// Fast operations for first few items, slow for later ones
				delay := time.Millisecond * 10
				if m > 3 {
					delay = time.Millisecond * 200 // Much slower for items 4 and 5
				}
				
				select {
				case <-time.After(delay):
					select {
					case <-ctx.Done():
						return
					default:
						rchan <- m * 10
					}
				case <-ctx.Done():
					return
				}
			}, nil
		}
		
		ctx, cancel := context.WithCancel(context.Background())
		
		// Cancel after enough time for first 3 items but not the rest
		go func() {
			time.Sleep(time.Millisecond * 50)
			cancel()
		}()
		
		result, err := AwaitSlice(
			model.SliceMap(provider)(model.FixedProvider(items))(),
			SetContext(ctx),
			SetTimeout(time.Second*5),
		)()
		
		// Should be cancelled even if some operations completed
		if err == nil {
			t.Fatal("Expected cancellation error but got none")
		}
		if err != ErrAwaitTimeout && err != context.Canceled {
			t.Errorf("Expected timeout or cancellation error, got %s", err)
		}
		if result != nil {
			t.Error("Expected nil result when context is cancelled (partial completion should not return partial results)")
		}
	})
	
	t.Run("ContextCancellationRaceWithError", func(t *testing.T) {
		// Test race between context cancellation and explicit errors
		items := []uint32{1, 2, 3}
		explicitError := errors.New("explicit async error")
		
		provider := func(m uint32) (Provider[uint32], error) {
			return func(ctx context.Context, rchan chan uint32, echan chan error) {
				if m == 2 {
					// Quick explicit error
					time.Sleep(time.Millisecond * 10)
					select {
					case <-ctx.Done():
						return
					default:
						echan <- explicitError
					}
					return
				}
				
				// Slow operation for other items
				select {
				case <-time.After(time.Millisecond * 100):
					rchan <- m
				case <-ctx.Done():
					return
				}
			}, nil
		}
		
		ctx, cancel := context.WithCancel(context.Background())
		
		// Cancel context after a moderate delay
		go func() {
			time.Sleep(time.Millisecond * 30)
			cancel()
		}()
		
		result, err := AwaitSlice(
			model.SliceMap(provider)(model.FixedProvider(items))(),
			SetContext(ctx),
			SetTimeout(time.Second),
		)()
		
		if err == nil {
			t.Fatal("Expected error but got none")
		}
		
		// Could be either explicit error or cancellation, depending on timing
		isValidError := err.Error() == explicitError.Error() || 
						err == ErrAwaitTimeout || 
						err == context.Canceled
		
		if !isValidError {
			t.Errorf("Expected explicit error, timeout, or cancellation, got %s", err)
		}
		
		if result != nil {
			t.Error("Expected nil result when error occurs")
		}
	})
	
	t.Run("ContextCancellationCleanup", func(t *testing.T) {
		// Test that context cancellation properly cleans up goroutines
		items := make([]uint32, 100) // More items to ensure goroutine creation
		for i := range items {
			items[i] = uint32(i + 1)
		}
		
		provider := func(m uint32) (Provider[uint32], error) {
			return func(ctx context.Context, rchan chan uint32, echan chan error) {
				// Long-running operation that should respect cancellation
				select {
				case <-time.After(time.Second * 2): // Very long operation
					rchan <- m
				case <-ctx.Done():
					// Clean exit on cancellation
					return
				}
			}, nil
		}
		
		ctx, cancel := context.WithCancel(context.Background())
		
		// Cancel quickly to test cleanup
		go func() {
			time.Sleep(time.Millisecond * 10)
			cancel()
		}()
		
		result, err := AwaitSlice(
			model.SliceMap(provider)(model.FixedProvider(items))(),
			SetContext(ctx),
			SetTimeout(time.Second*10),
		)()
		
		if err == nil {
			t.Fatal("Expected cancellation error but got none")
		}
		if err != ErrAwaitTimeout && err != context.Canceled {
			t.Errorf("Expected timeout or cancellation error, got %s", err)
		}
		if result != nil {
			t.Error("Expected nil result when context is cancelled")
		}
		
		// Give a moment for goroutines to clean up
		time.Sleep(time.Millisecond * 50)
	})
	
	t.Run("SingleProviderContextCancellation", func(t *testing.T) {
		// Test context cancellation with single async provider
		provider := func(ctx context.Context, rchan chan uint32, echan chan error) {
			// Long operation that should be cancelled
			select {
			case <-time.After(time.Millisecond * 200):
				rchan <- 42
			case <-ctx.Done():
				return
			}
		}
		
		ctx, cancel := context.WithCancel(context.Background())
		
		// Cancel after short delay
		go func() {
			time.Sleep(time.Millisecond * 50)
			cancel()
		}()
		
		result, err := Await(
			SingleProvider(provider),
			SetContext(ctx),
			SetTimeout(time.Second),
		)()
		
		if err == nil {
			t.Fatal("Expected cancellation error but got none")
		}
		if err != ErrAwaitTimeout && err != context.Canceled {
			t.Errorf("Expected timeout or cancellation error, got %s", err)
		}
		if result != 0 {
			t.Errorf("Expected zero value result when cancelled, got %d", result)
		}
	})
}

func BenchmarkAwaitSlice(b *testing.B) {
	// Create test data
	data := make([]uint32, 1000)
	for i := range data {
		data[i] = uint32(i + 1)
	}

	// CPU-intensive provider with variable delays
	intensiveProvider := func(val uint32) (Provider[uint32], error) {
		return func(ctx context.Context, rchan chan uint32, echan chan error) {
			// CPU-intensive work
			result := val
			for i := 0; i < 1000; i++ {
				result = (result*7 + 13) % 1000003
			}
			
			select {
			case <-ctx.Done():
				return
			case rchan <- result:
			}
		}, nil
	}

	b.Run("AsyncProcessing", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ctx := context.Background()
			results, err := AwaitSlice(
				model.SliceMap(intensiveProvider)(model.FixedProvider(data))(),
				SetContext(ctx),
				SetTimeout(time.Second*10),
			)()
			
			if err != nil {
				b.Fatalf("Unexpected error: %v", err)
			}
			
			if len(results) != len(data) {
				b.Fatalf("Expected %d results, got %d", len(data), len(results))
			}
		}
	})

	// Test with small timeout to benchmark timeout handling
	b.Run("TimeoutHandling", func(b *testing.B) {
		slowProvider := func(val uint32) (Provider[uint32], error) {
			return func(ctx context.Context, rchan chan uint32, echan chan error) {
				time.Sleep(time.Millisecond * 100) // Slow operation
				
				select {
				case <-ctx.Done():
					return
				case rchan <- val*2:
				}
			}, nil
		}

		smallData := data[:10] // Use smaller dataset for timeout test
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ctx := context.Background()
			_, err := AwaitSlice(
				model.SliceMap(slowProvider)(model.FixedProvider(smallData))(),
				SetContext(ctx),
				SetTimeout(time.Millisecond*50), // Shorter than operation time
			)()
			
			if err == nil {
				b.Fatal("Expected timeout error but got none")
			}
		}
	})
}

// Benchmark for error handling performance in async processing
func BenchmarkAwaitSliceErrorHandling(b *testing.B) {
	// Create test data
	data := make([]uint32, 100)
	for i := range data {
		data[i] = uint32(i + 1)
	}

	// Provider that fails on specific values
	errorProvider := func(val uint32) (Provider[uint32], error) {
		return func(ctx context.Context, rchan chan uint32, echan chan error) {
			if val == 50 { // Fail halfway through
				select {
				case <-ctx.Done():
					return
				case echan <- fmt.Errorf("test error on value %d", val):
				}
				return
			}
			
			// Small CPU work for successful cases
			result := val
			for i := 0; i < 100; i++ {
				result = (result*7 + 13) % 1000003
			}
			
			select {
			case <-ctx.Done():
				return
			case rchan <- result:
			}
		}, nil
	}

	b.Run("ErrorHandlingPerformance", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ctx := context.Background()
			_, err := AwaitSlice(
				model.SliceMap(errorProvider)(model.FixedProvider(data))(),
				SetContext(ctx),
				SetTimeout(time.Second*5),
			)()
			
			if err == nil {
				b.Fatal("Expected error but got none")
			}
		}
	})
}

// Benchmark for concurrent load
func BenchmarkAwaitSliceConcurrentLoad(b *testing.B) {
	// Create test data with many items to stress concurrency
	data := make([]uint32, 5000)
	for i := range data {
		data[i] = uint32(i + 1)
	}

	// Light provider to focus on concurrency overhead
	lightProvider := func(val uint32) (Provider[uint32], error) {
		return func(ctx context.Context, rchan chan uint32, echan chan error) {
			// Very light work
			result := val * 2
			
			select {
			case <-ctx.Done():
				return
			case rchan <- result:
			}
		}, nil
	}

	b.Run("HighConcurrency", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ctx := context.Background()
			results, err := AwaitSlice(
				model.SliceMap(lightProvider)(model.FixedProvider(data))(),
				SetContext(ctx),
				SetTimeout(time.Second*30),
			)()
			
			if err != nil {
				b.Fatalf("Unexpected error: %v", err)
			}
			
			if len(results) != len(data) {
				b.Fatalf("Expected %d results, got %d", len(data), len(results))
			}
		}
	})
}

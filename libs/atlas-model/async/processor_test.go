package async

import (
	"context"
	"errors"
	"fmt"
	"github.com/Chronicle20/atlas-model/model"
	"strings"
	"sync"
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
}

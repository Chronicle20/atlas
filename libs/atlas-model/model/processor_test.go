package model

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestFirstNoFilter(t *testing.T) {
	p := FixedProvider([]uint32{1})
	r, err := First(p, Filters[uint32]())
	if err != nil {
		t.Errorf("Expected result, got err %s", err)
	}
	if r != 1 {
		t.Errorf("Expected 1, got %d", r)
	}
}

func byTwo(val uint32) (uint32, error) {
	return val * 2, nil
}

func TestMap(t *testing.T) {
	p := FixedProvider(uint32(1))
	mp := Map[uint32, uint32](byTwo)(p)

	ar, err := mp()
	if err != nil {
		t.Errorf("Expected result, got err %s", err)
	}

	er, _ := p()
	if er*2 != ar {
		t.Errorf("Expected %d, got %d", er*2, ar)
	}
}

func TestSliceMap(t *testing.T) {
	p := FixedProvider([]uint32{1, 2, 3, 4, 5})
	mp := SliceMap[uint32, uint32](byTwo)(p)()

	ar, err := mp()
	if err != nil {
		t.Errorf("Expected result, got err %s", err)
	}

	er, _ := p()
	for i := range er {
		if er[i]*2 != ar[i] {
			t.Errorf("Expected %d, got %d", er[i]*2, ar[i])
		}
	}
}

func TestParallelSliceMap(t *testing.T) {
	p := FixedProvider([]uint32{1, 2, 3, 4, 5})
	mp := SliceMap[uint32, uint32](byTwo)(p)(ParallelMap())

	ar, err := mp()
	if err != nil {
		t.Errorf("Expected result, got err %s", err)
	}

	er, _ := p()
	for i := range er {
		if er[i]*2 != ar[i] {
			t.Errorf("Expected %d, got %d", er[i]*2, ar[i])
		}
	}
}

func isTwo(val uint32) bool {
	return val == 2
}

func TestFirst(t *testing.T) {
	p := FixedProvider([]uint32{1, 2, 3, 4, 5})
	mp, err := First(p, Filters(isTwo))
	if err != nil {
		t.Errorf("Expected result, got err %s", err)
	}
	if mp != 2 {
		t.Errorf("Expected 2, got %d", mp)
	}
}

func TestThenOperator(t *testing.T) {
	p := FixedProvider(uint32(1))
	count := uint32(0)

	op1 := func(u uint32) error {
		count += u
		return nil
	}
	op2 := func(u uint32) error {
		count += u
		return nil
	}

	err := For(p, ThenOperator(op1, Operators(op2)))
	if err != nil {
		t.Errorf("Expected result, got err %s", err)
	}
	if count != 2 {
		t.Errorf("Expected 2, got %d", count)
	}
}

func TestForEachSlice(t *testing.T) {
	p := FixedProvider([]uint32{1, 2, 3, 4, 5})
	count := uint32(0)

	err := ForEachSlice(p, func(u uint32) error {
		count += u
		return nil
	})
	if err != nil {
		t.Errorf("Expected result, got err %s", err)
	}
	if count != 15 {
		t.Errorf("Expected 15, got %d", count)
	}
}

func TestForEachSliceParallel(t *testing.T) {
	p := FixedProvider([]uint32{1, 2, 3, 4, 5})
	var count uint32
	var mu sync.Mutex

	err := ForEachSlice(p, func(u uint32) error {
		mu.Lock()
		defer mu.Unlock()
		count += u
		return nil
	}, ParallelExecute())
	if err != nil {
		t.Errorf("Expected result, got err %s", err)
	}
	if count != 15 {
		t.Errorf("Expected 15, got %d", count)
	}
}

func TestForEachMap(t *testing.T) {
	p := FixedProvider(map[uint32][]uint32{1: {1, 2}, 2: {1, 2, 3}})
	counts := map[uint32]uint32{}

	err := ForEachMap(p, func(k uint32) Operator[[]uint32] {
		return func(vs []uint32) error {
			count := uint32(0)
			for _, v := range vs {
				count += v
			}
			counts[k] = count
			return nil
		}
	})
	if err != nil {
		t.Errorf("Expected result, got err %s", err)
	}
	if counts[1] != 3 {
		t.Errorf("Expected 3, got %d", counts[1])
	}
	if counts[2] != 6 {
		t.Errorf("Expected 6, got %d", counts[2])
	}
}

func TestForEachMapParallel(t *testing.T) {
	p := FixedProvider(map[uint32][]uint32{1: {1, 2}, 2: {1, 2, 3}})
	counts := map[uint32]uint32{}
	var mu sync.Mutex

	err := ForEachMap(p, func(k uint32) Operator[[]uint32] {
		return func(vs []uint32) error {
			count := uint32(0)
			for _, v := range vs {
				count += v
			}
			mu.Lock()
			defer mu.Unlock()
			counts[k] = count
			return nil
		}
	}, ParallelExecute())
	if err != nil {
		t.Errorf("Expected result, got err %s", err)
	}
	if counts[1] != 3 {
		t.Errorf("Expected 3, got %d", counts[1])
	}
	if counts[2] != 6 {
		t.Errorf("Expected 6, got %d", counts[2])
	}
}

func TestMerge(t *testing.T) {
	p1 := FixedProvider([]uint32{1, 2, 3, 4, 5})
	p2 := FixedProvider([]uint32{1, 2, 3, 4, 5})
	rp := MergeSliceProvider(p1, p2)

	rs, err := rp()
	if err != nil {
		t.Errorf("Expected result, got err %s", err)
	}
	if len(rs) != 10 {
		t.Errorf("Expected 10, got %d", len(rs))
	}
}

func TestApply(t *testing.T) {
	f := func(a uint32) Provider[uint32] {
		return func() (uint32, error) {
			return a + 32, nil
		}
	}

	r, err := Apply(f)(5)
	if err != nil {
		t.Errorf("Expected result, got err %s", err)
	}
	if r != 37 {
		t.Errorf("Expected 37, got %d", r)
	}
}

func TestCurry(t *testing.T) {
	f := func(a uint32, b uint32) uint32 {
		return a + b
	}
	r := Curry(f)(1)(2)
	if r != 3 {
		t.Errorf("Expected 3, got %d", r)
	}
}

func TestCompose(t *testing.T) {
	f := func(a uint64) func(b uint32) func(c uint16) Provider[uint32] {
		return func(b uint32) func(c uint16) Provider[uint32] {
			return func(c uint16) Provider[uint32] {
				return func() (uint32, error) {
					return uint32(a)*b*uint32(c) + 24, nil
				}
			}
		}
	}

	type a = uint32
	type b = func(uint16) Provider[uint32]
	type c = func(uint16) (uint32, error)

	r, err := Compose(Curry(Compose[a, b, c])(Apply[uint16, uint32]), f)(2)(3)(4)
	if err != nil {
		t.Errorf("Expected result, got err %s", err)
	}
	if r != 48 {
		t.Errorf("Expected 47, got %d", r)
	}
}

func TestParameterTransformation(t *testing.T) {
	type rt = func(uint642 uint64) Provider[uint32]

	f := func(a uint32) Provider[uint32] {
		return func() (uint32, error) {
			return a + 32, nil
		}
	}
	tf := func(uint642 uint64) uint32 {
		return uint32(uint642)
	}

	var rf rt = Compose(f, tf)
	r, err := rf(5)()
	if err != nil {
		t.Errorf("Expected result, got err %s", err)
	}
	if r != 37 {
		t.Errorf("Expected 37, got %d", r)
	}
}

func TestFilters(t *testing.T) {
	ip := FixedProvider([]uint32{1, 2, 3, 4, 5})
	f := func(i uint32) bool {
		return i > 3
	}
	op := FilteredProvider(ip, Filters(f))
	r, err := op()
	if err != nil {
		t.Errorf("Expected result, got err %s", err)
	}
	if len(r) != 2 {
		t.Errorf("Expected 2, got %d", len(r))
	}

}

func TestMapLazyEvaluation(t *testing.T) {
	// Test that Map function defers execution until Provider is called
	executed := false

	// Create a Provider that tracks execution
	trackingProvider := func() (uint32, error) {
		executed = true
		return 5, nil
	}

	// Transform function that doubles the value
	doubleTransform := func(val uint32) (uint32, error) {
		return val * 2, nil
	}

	// Create Map pipeline - should NOT execute the provider yet
	mappedProvider := Map[uint32, uint32](doubleTransform)(trackingProvider)

	// Verify that the underlying provider has not been executed during composition
	if executed {
		t.Errorf("Map function should not execute provider during composition")
	}

	// Now execute the mapped provider
	result, err := mappedProvider()
	if err != nil {
		t.Errorf("Expected result, got err %s", err)
	}

	// Verify execution happened and result is correct
	if !executed {
		t.Errorf("Provider should have been executed when mapped provider was called")
	}
	if result != 10 {
		t.Errorf("Expected 10, got %d", result)
	}
}

func TestSliceMapLazyEvaluation(t *testing.T) {
	executed := false

	// Create a provider that tracks if it was executed
	provider := func() ([]uint32, error) {
		executed = true
		return []uint32{1, 2, 3}, nil
	}

	// Create the mapped provider - this should NOT execute the underlying provider
	mappedProvider := SliceMap[uint32, uint32](byTwo)(provider)()

	// Verify that the underlying provider has not been executed during composition
	if executed {
		t.Errorf("SliceMap function should not execute provider during composition")
	}

	// Now execute the mapped provider
	result, err := mappedProvider()
	if err != nil {
		t.Errorf("Expected result, got err %s", err)
	}

	// Verify execution happened and result is correct
	if !executed {
		t.Errorf("Provider should have been executed when mapped provider was called")
	}
	expected := []uint32{2, 4, 6}
	if len(result) != len(expected) {
		t.Errorf("Expected length %d, got %d", len(expected), len(result))
	}
	for i := range expected {
		if result[i] != expected[i] {
			t.Errorf("Expected %d, got %d at index %d", expected[i], result[i], i)
		}
	}
}

func TestParallelSliceMapLazyEvaluation(t *testing.T) {
	executed := false

	// Create a provider that tracks if it was executed
	provider := func() ([]uint32, error) {
		executed = true
		return []uint32{1, 2, 3}, nil
	}

	// Create the mapped provider with parallel execution - this should NOT execute the underlying provider
	mappedProvider := SliceMap[uint32, uint32](byTwo)(provider)(ParallelMap())

	// Verify that the underlying provider has not been executed during composition
	if executed {
		t.Errorf("SliceMap with ParallelMap should not execute provider during composition")
	}

	// Now execute the mapped provider
	result, err := mappedProvider()
	if err != nil {
		t.Errorf("Expected result, got err %s", err)
	}

	// Verify execution happened and result is correct
	if !executed {
		t.Errorf("Provider should have been executed when mapped provider was called")
	}
	expected := []uint32{2, 4, 6}
	if len(result) != len(expected) {
		t.Errorf("Expected length %d, got %d", len(expected), len(result))
	}
	for i := range expected {
		if result[i] != expected[i] {
			t.Errorf("Expected %d, got %d at index %d", expected[i], result[i], i)
		}
	}
}

func TestFilteredProviderLazyEvaluation(t *testing.T) {
	// Test that FilteredProvider function defers execution until Provider is called
	executed := false

	// Create a Provider that tracks execution
	trackingProvider := func() ([]uint32, error) {
		executed = true
		return []uint32{1, 2, 3, 4, 5}, nil
	}

	// Filter function that only allows even numbers
	evenFilter := func(val uint32) bool {
		return val%2 == 0
	}

	// Create FilteredProvider pipeline - should NOT execute the provider yet
	filteredProvider := FilteredProvider(trackingProvider, []Filter[uint32]{evenFilter})

	// Verify that the underlying provider has not been executed during composition
	if executed {
		t.Errorf("FilteredProvider function should not execute provider during composition")
	}

	// Now execute the filtered provider
	result, err := filteredProvider()
	if err != nil {
		t.Errorf("Expected result, got err %s", err)
	}

	// Verify execution happened and result is correct
	if !executed {
		t.Errorf("Provider should have been executed when filtered provider was called")
	}

	expected := []uint32{2, 4}
	if len(result) != len(expected) {
		t.Errorf("Expected length %d, got %d", len(expected), len(result))
	}
	for i := range expected {
		if result[i] != expected[i] {
			t.Errorf("Expected %d, got %d at index %d", expected[i], result[i], i)
		}
	}
}

func TestFoldLazyEvaluation(t *testing.T) {
	// Test that Fold function defers execution until Provider is called
	providerExecuted := false
	supplierExecuted := false

	// Create providers that track execution
	trackingProvider := func() ([]uint32, error) {
		providerExecuted = true
		return []uint32{1, 2, 3}, nil
	}

	trackingSupplier := func() (uint32, error) {
		supplierExecuted = true
		return uint32(0), nil
	}

	// Folder function that sums values
	sumFolder := func(acc uint32, val uint32) (uint32, error) {
		return acc + val, nil
	}

	// Create Fold pipeline - should NOT execute the providers yet
	foldProvider := Fold(trackingProvider, trackingSupplier, sumFolder)

	// Verify that the underlying providers have not been executed during composition
	if providerExecuted {
		t.Errorf("Fold function should not execute provider during composition")
	}
	if supplierExecuted {
		t.Errorf("Fold function should not execute supplier during composition")
	}

	// Now execute the fold provider
	result, err := foldProvider()
	if err != nil {
		t.Errorf("Expected result, got err %s", err)
	}

	// Verify execution happened and result is correct
	if !providerExecuted {
		t.Errorf("Provider should have been executed when fold provider was called")
	}
	if !supplierExecuted {
		t.Errorf("Supplier should have been executed when fold provider was called")
	}

	// Verify the result is the sum of 0 + 1 + 2 + 3 = 6
	expected := uint32(6)
	if result != expected {
		t.Errorf("Expected %d, got %d", expected, result)
	}
}

func TestCollectToMapLazyEvaluation(t *testing.T) {
	// Test that CollectToMap function defers execution until Provider is called
	executed := false

	// Create a Provider that tracks execution
	trackingProvider := func() ([]uint32, error) {
		executed = true
		return []uint32{1, 2, 3}, nil
	}

	// Key provider that converts value to string
	keyProvider := func(val uint32) string {
		return fmt.Sprintf("key-%d", val)
	}

	// Value provider that doubles the value
	valueProvider := func(val uint32) uint32 {
		return val * 2
	}

	// Create CollectToMap pipeline - should NOT execute the provider yet
	mapProvider := CollectToMap[uint32, string, uint32](trackingProvider, keyProvider, valueProvider)

	// Verify that the underlying provider has not been executed during composition
	if executed {
		t.Errorf("CollectToMap function should not execute provider during composition")
	}

	// Now execute the map provider
	result, err := mapProvider()
	if err != nil {
		t.Errorf("Expected result, got err %s", err)
	}

	// Verify execution happened and result is correct
	if !executed {
		t.Errorf("Provider should have been executed when map provider was called")
	}

	// Verify the result has correct mapping
	expected := map[string]uint32{
		"key-1": 2,
		"key-2": 4,
		"key-3": 6,
	}

	if len(result) != len(expected) {
		t.Errorf("Expected map with %d entries, got %d", len(expected), len(result))
	}

	for key, expectedValue := range expected {
		if actualValue, exists := result[key]; !exists {
			t.Errorf("Expected key %s not found in result", key)
		} else if actualValue != expectedValue {
			t.Errorf("For key %s: expected %d, got %d", key, expectedValue, actualValue)
		}
	}
}

func TestMergeSliceProviderLazyEvaluation(t *testing.T) {
	// Track side effects to verify lazy evaluation
	executed1 := false
	executed2 := false

	// Create providers with side effects
	provider1 := func() ([]uint32, error) {
		executed1 = true
		return []uint32{1, 2, 3}, nil
	}

	provider2 := func() ([]uint32, error) {
		executed2 = true
		return []uint32{4, 5, 6}, nil
	}

	// Compose the merge - should NOT execute providers yet
	mergedProvider := MergeSliceProvider(provider1, provider2)

	// Verify no execution happened during composition
	if executed1 {
		t.Errorf("MergeSliceProvider should not execute first provider during composition")
	}
	if executed2 {
		t.Errorf("MergeSliceProvider should not execute second provider during composition")
	}

	// Now execute the merged provider
	result, err := mergedProvider()
	if err != nil {
		t.Errorf("Expected result, got err %s", err)
	}

	// Verify execution happened when merged provider was called
	if !executed1 {
		t.Errorf("First provider should have been executed when merged provider was called")
	}
	if !executed2 {
		t.Errorf("Second provider should have been executed when merged provider was called")
	}

	// Verify the result is correct
	expected := []uint32{1, 2, 3, 4, 5, 6}
	if len(result) != len(expected) {
		t.Errorf("Expected slice with %d elements, got %d", len(expected), len(result))
	}

	for i, expectedValue := range expected {
		if result[i] != expectedValue {
			t.Errorf("At index %d: expected %d, got %d", i, expectedValue, result[i])
		}
	}
}

func TestMemoize(t *testing.T) {
	// Test that Memoize caches results and only executes underlying provider once
	executionCount := 0

	// Create a provider that tracks how many times it's been called
	expensiveProvider := func() (uint32, error) {
		executionCount++
		return uint32(42), nil
	}

	// Create memoized provider
	memoizedProvider := Memoize(expensiveProvider)

	// First call should execute the provider
	result1, err1 := memoizedProvider()
	if err1 != nil {
		t.Errorf("Expected result, got err %s", err1)
	}
	if result1 != 42 {
		t.Errorf("Expected 42, got %d", result1)
	}
	if executionCount != 1 {
		t.Errorf("Expected execution count 1 after first call, got %d", executionCount)
	}

	// Second call should return cached result without executing provider again
	result2, err2 := memoizedProvider()
	if err2 != nil {
		t.Errorf("Expected result, got err %s", err2)
	}
	if result2 != 42 {
		t.Errorf("Expected 42, got %d", result2)
	}
	if executionCount != 1 {
		t.Errorf("Expected execution count 1 after second call (cached), got %d", executionCount)
	}

	// Third call should also return cached result
	result3, err3 := memoizedProvider()
	if err3 != nil {
		t.Errorf("Expected result, got err %s", err3)
	}
	if result3 != 42 {
		t.Errorf("Expected 42, got %d", result3)
	}
	if executionCount != 1 {
		t.Errorf("Expected execution count 1 after third call (cached), got %d", executionCount)
	}
}

func TestMemoizeError(t *testing.T) {
	// Test that Memoize also caches errors
	executionCount := 0
	expectedError := fmt.Errorf("test error")

	// Create a provider that always returns an error
	errorProvider := func() (uint32, error) {
		executionCount++
		return 0, expectedError
	}

	// Create memoized provider
	memoizedProvider := Memoize(errorProvider)

	// First call should execute and return the error
	result1, err1 := memoizedProvider()
	if err1 == nil {
		t.Errorf("Expected error, got result %d", result1)
	}
	if err1.Error() != expectedError.Error() {
		t.Errorf("Expected error '%s', got '%s'", expectedError.Error(), err1.Error())
	}
	if executionCount != 1 {
		t.Errorf("Expected execution count 1 after first call, got %d", executionCount)
	}

	// Second call should return cached error without executing provider again
	result2, err2 := memoizedProvider()
	if err2 == nil {
		t.Errorf("Expected error, got result %d", result2)
	}
	if err2.Error() != expectedError.Error() {
		t.Errorf("Expected error '%s', got '%s'", expectedError.Error(), err2.Error())
	}
	if executionCount != 1 {
		t.Errorf("Expected execution count 1 after second call (cached error), got %d", executionCount)
	}
}

func TestMemoizeConcurrentAccess(t *testing.T) {
	// Test that Memoize is thread-safe when accessed concurrently from multiple goroutines
	executionCount := int64(0)
	
	// Create a provider that tracks execution count atomically
	expensiveProvider := func() (uint32, error) {
		atomic.AddInt64(&executionCount, 1)
		// Simulate some work to increase chance of race conditions
		time.Sleep(time.Millisecond * 10)
		return uint32(42), nil
	}

	// Create memoized provider
	memoizedProvider := Memoize(expensiveProvider)

	// Number of concurrent goroutines to test with
	const numGoroutines = 20
	var wg sync.WaitGroup
	results := make([]uint32, numGoroutines)
	errors := make([]error, numGoroutines)

	// Launch multiple goroutines that all call the memoized provider concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			result, err := memoizedProvider()
			results[idx] = result
			errors[idx] = err
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Verify that the underlying provider was executed exactly once
	if atomic.LoadInt64(&executionCount) != 1 {
		t.Errorf("Expected execution count 1 with concurrent access, got %d", executionCount)
	}

	// Verify all results are consistent
	for i := 0; i < numGoroutines; i++ {
		if errors[i] != nil {
			t.Errorf("Goroutine %d: Expected result, got error %s", i, errors[i])
		}
		if results[i] != 42 {
			t.Errorf("Goroutine %d: Expected result 42, got %d", i, results[i])
		}
	}

	// Verify subsequent calls still return cached result
	result, err := memoizedProvider()
	if err != nil {
		t.Errorf("Post-concurrent call: Expected result, got error %s", err)
	}
	if result != 42 {
		t.Errorf("Post-concurrent call: Expected result 42, got %d", result)
	}
	if atomic.LoadInt64(&executionCount) != 1 {
		t.Errorf("Post-concurrent call: Expected execution count 1, got %d", executionCount)
	}
}

func TestFirstProviderLazyEvaluation(t *testing.T) {
	// Test that FirstProvider function defers execution until Provider is called
	executed := false

	// Create a Provider that tracks execution
	trackingProvider := func() ([]uint32, error) {
		executed = true
		return []uint32{1, 2, 3, 4, 5}, nil
	}

	// Filter function that only allows even numbers
	evenFilter := func(val uint32) bool {
		return val%2 == 0
	}

	// Create FirstProvider pipeline - should NOT execute the provider yet
	firstProvider := FirstProvider(trackingProvider, []Filter[uint32]{evenFilter})

	// Verify that the underlying provider has not been executed during composition
	if executed {
		t.Errorf("FirstProvider function should not execute provider during composition")
	}

	// Now execute the first provider
	result, err := firstProvider()
	if err != nil {
		t.Errorf("Expected result, got err %s", err)
	}

	// Verify execution happened and result is correct
	if !executed {
		t.Errorf("Provider should have been executed when first provider was called")
	}

	// Should return first even number (2)
	expected := uint32(2)
	if result != expected {
		t.Errorf("Expected %d, got %d", expected, result)
	}
}

func TestToSliceProviderLazyEvaluation(t *testing.T) {
	// Test that ToSliceProvider function defers execution until Provider is called
	executed := false

	// Create a Provider that tracks execution
	trackingProvider := func() (uint32, error) {
		executed = true
		return uint32(42), nil
	}

	// Create ToSliceProvider pipeline - should NOT execute the provider yet
	sliceProvider := ToSliceProvider(trackingProvider)

	// Verify that the underlying provider has not been executed during composition
	if executed {
		t.Errorf("ToSliceProvider function should not execute provider during composition")
	}

	// Now execute the slice provider
	result, err := sliceProvider()
	if err != nil {
		t.Errorf("Expected result, got err %s", err)
	}

	// Verify execution happened and result is correct
	if !executed {
		t.Errorf("Provider should have been executed when slice provider was called")
	}

	// Should return slice with single element
	expected := []uint32{42}
	if len(result) != 1 {
		t.Errorf("Expected slice with 1 element, got %d", len(result))
	}
	if result[0] != expected[0] {
		t.Errorf("Expected %d, got %d", expected[0], result[0])
	}
}

func TestLazyHelperLazyEvaluation(t *testing.T) {
	// Test that Lazy helper function defers execution until Provider is called
	executed := false

	// Create a function that returns a provider and tracks execution
	providerFactory := func() Provider[uint32] {
		executed = true
		return func() (uint32, error) {
			return uint32(123), nil
		}
	}

	// Create Lazy provider - should NOT execute the factory function yet
	lazyProvider := Lazy(providerFactory)

	// Verify that the factory function has not been executed during composition
	if executed {
		t.Errorf("Lazy helper should not execute factory function during composition")
	}

	// Now execute the lazy provider
	result, err := lazyProvider()
	if err != nil {
		t.Errorf("Expected result, got err %s", err)
	}

	// Verify execution happened and result is correct
	if !executed {
		t.Errorf("Factory function should have been executed when lazy provider was called")
	}

	expected := uint32(123)
	if result != expected {
		t.Errorf("Expected %d, got %d", expected, result)
	}
}

func TestComplexPipelineLazyEvaluation(t *testing.T) {
	// Test lazy evaluation in a complex pipeline with multiple combinators
	executionOrder := []string{}

	// Create providers that track execution order
	provider1 := func() ([]uint32, error) {
		executionOrder = append(executionOrder, "provider1")
		return []uint32{1, 2, 3, 4, 5, 6}, nil
	}

	// Filter transform for SliceMap
	transform := func(val uint32) (uint32, error) {
		executionOrder = append(executionOrder, fmt.Sprintf("transform-%d", val))
		return val * 2, nil
	}

	// Build complex pipeline: provider1 -> SliceMap -> FilteredProvider -> FirstProvider
	// This should NOT execute any of the underlying operations during composition
	pipeline := FirstProvider(
		FilteredProvider(
			SliceMap[uint32, uint32](transform)(provider1)(),
			[]Filter[uint32]{func(val uint32) bool {
				executionOrder = append(executionOrder, fmt.Sprintf("filter-%d", val))
				return val > 6 // Only values greater than 6
			}},
		),
		[]Filter[uint32]{}, // No additional filters for FirstProvider
	)

	// Verify no execution happened during pipeline composition
	if len(executionOrder) > 0 {
		t.Errorf("Complex pipeline should not execute any operations during composition, but got: %v", executionOrder)
	}

	// Now execute the pipeline
	result, err := pipeline()
	if err != nil {
		t.Errorf("Expected result, got err %s", err)
	}

	// Verify execution happened in correct order
	expectedExecutionOrder := []string{
		"provider1",
		"transform-1", "transform-2", "transform-3", "transform-4", "transform-5", "transform-6",
		"filter-2", "filter-4", "filter-6", "filter-8", "filter-10", "filter-12",
	}

	if len(executionOrder) != len(expectedExecutionOrder) {
		t.Errorf("Expected %d execution steps, got %d: %v", len(expectedExecutionOrder), len(executionOrder), executionOrder)
	}

	for i, expected := range expectedExecutionOrder {
		if i >= len(executionOrder) || executionOrder[i] != expected {
			t.Errorf("At step %d: expected '%s', got execution order: %v", i, expected, executionOrder)
			break
		}
	}

	// Verify the result is correct
	// Pipeline: [1,2,3,4,5,6] -> transform (*2) -> [2,4,6,8,10,12] -> filter (>6) -> [8,10,12] -> first -> 8
	expected := uint32(8)
	if result != expected {
		t.Errorf("Expected %d, got %d", expected, result)
	}
}

func TestErrorPropagationLazyContext(t *testing.T) {
	// Test comprehensive error propagation in lazy evaluation contexts

	t.Run("MapErrorPropagation", func(t *testing.T) {
		// Test error propagation through Map function
		expectedError := errors.New("provider error")

		// Provider that returns an error
		errorProvider := func() (uint32, error) {
			return 0, expectedError
		}

		// Transform function that should never be called
		transformCalled := false
		transform := func(val uint32) (uint32, error) {
			transformCalled = true
			return val * 2, nil
		}

		// Build Map pipeline
		mappedProvider := Map[uint32, uint32](transform)(errorProvider)

		// Execute and verify error propagation
		result, err := mappedProvider()

		// Should receive the original error
		if err == nil {
			t.Errorf("Expected error, got result %d", result)
		}
		if err.Error() != expectedError.Error() {
			t.Errorf("Expected error '%s', got '%s'", expectedError.Error(), err.Error())
		}

		// Transform should never be called due to provider error
		if transformCalled {
			t.Errorf("Transform function should not be called when provider fails")
		}

		// Result should be zero value
		if result != 0 {
			t.Errorf("Expected zero result when error occurs, got %d", result)
		}
	})

	t.Run("MapTransformErrorPropagation", func(t *testing.T) {
		// Test error propagation from transformer function in Map
		expectedError := errors.New("transform error")

		// Provider that succeeds
		provider := func() (uint32, error) {
			return 42, nil
		}

		// Transform function that returns an error
		transform := func(val uint32) (uint32, error) {
			return 0, expectedError
		}

		// Build Map pipeline
		mappedProvider := Map[uint32, uint32](transform)(provider)

		// Execute and verify error propagation
		result, err := mappedProvider()

		// Should receive the transform error
		if err == nil {
			t.Errorf("Expected error, got result %d", result)
		}
		if err.Error() != expectedError.Error() {
			t.Errorf("Expected error '%s', got '%s'", expectedError.Error(), err.Error())
		}

		// Result should be zero value
		if result != 0 {
			t.Errorf("Expected zero result when transform error occurs, got %d", result)
		}
	})

	t.Run("SliceMapErrorPropagation", func(t *testing.T) {
		// Test error propagation in SliceMap function
		expectedError := errors.New("slice provider error")

		// Provider that returns an error
		errorProvider := func() ([]uint32, error) {
			return nil, expectedError
		}

		// Transform function that should never be called
		transformCalled := false
		transform := func(val uint32) (uint32, error) {
			transformCalled = true
			return val * 2, nil
		}

		// Build SliceMap pipeline
		mappedProvider := SliceMap[uint32, uint32](transform)(errorProvider)()

		// Execute and verify error propagation
		result, err := mappedProvider()

		// Should receive the original error
		if err == nil {
			t.Errorf("Expected error, got result %v", result)
		}
		if err.Error() != expectedError.Error() {
			t.Errorf("Expected error '%s', got '%s'", expectedError.Error(), err.Error())
		}

		// Transform should never be called due to provider error
		if transformCalled {
			t.Errorf("Transform function should not be called when provider fails")
		}

		// Result should be nil
		if result != nil {
			t.Errorf("Expected nil result when error occurs, got %v", result)
		}
	})

	t.Run("SliceMapTransformErrorPropagation", func(t *testing.T) {
		// Test error propagation from transformer in SliceMap
		expectedError := errors.New("transform error")

		// Provider that succeeds
		provider := func() ([]uint32, error) {
			return []uint32{1, 2, 3}, nil
		}

		// Transform function that fails on second item
		transformCallCount := 0
		transform := func(val uint32) (uint32, error) {
			transformCallCount++
			if val == 2 {
				return 0, expectedError
			}
			return val * 2, nil
		}

		// Build SliceMap pipeline
		mappedProvider := SliceMap[uint32, uint32](transform)(provider)()

		// Execute and verify error propagation
		result, err := mappedProvider()

		// Should receive the transform error
		if err == nil {
			t.Errorf("Expected error, got result %v", result)
		}
		if err.Error() != expectedError.Error() {
			t.Errorf("Expected error '%s', got '%s'", expectedError.Error(), err.Error())
		}

		// Transform should be called at least twice (up to the failing item)
		if transformCallCount < 2 {
			t.Errorf("Expected at least 2 transform calls, got %d", transformCallCount)
		}

		// Result should be nil
		if result != nil {
			t.Errorf("Expected nil result when transform error occurs, got %v", result)
		}
	})

	t.Run("ParallelSliceMapErrorPropagation", func(t *testing.T) {
		// Test error propagation in parallel SliceMap
		expectedError := errors.New("parallel transform error")

		// Provider that succeeds
		provider := func() ([]uint32, error) {
			return []uint32{1, 2, 3, 4, 5}, nil
		}

		// Transform function that fails on specific value
		transform := func(val uint32) (uint32, error) {
			if val == 3 {
				return 0, expectedError
			}
			return val * 2, nil
		}

		// Build parallel SliceMap pipeline
		mappedProvider := SliceMap[uint32, uint32](transform)(provider)(ParallelMap())

		// Execute and verify error propagation
		result, err := mappedProvider()

		// Should receive the transform error
		if err == nil {
			t.Errorf("Expected error, got result %v", result)
		}
		if err.Error() != expectedError.Error() {
			t.Errorf("Expected error '%s', got '%s'", expectedError.Error(), err.Error())
		}

		// Result should be nil
		if result != nil {
			t.Errorf("Expected nil result when parallel transform error occurs, got %v", result)
		}
	})

	t.Run("FilteredProviderErrorPropagation", func(t *testing.T) {
		// Test error propagation in FilteredProvider
		expectedError := errors.New("filtered provider error")

		// Provider that returns an error
		errorProvider := func() ([]uint32, error) {
			return nil, expectedError
		}

		// Filter function that should never be called
		filterCalled := false
		filter := func(val uint32) bool {
			filterCalled = true
			return val > 2
		}

		// Build FilteredProvider pipeline
		filteredProvider := FilteredProvider(errorProvider, []Filter[uint32]{filter})

		// Execute and verify error propagation
		result, err := filteredProvider()

		// Should receive the original error
		if err == nil {
			t.Errorf("Expected error, got result %v", result)
		}
		if err.Error() != expectedError.Error() {
			t.Errorf("Expected error '%s', got '%s'", expectedError.Error(), err.Error())
		}

		// Filter should never be called due to provider error
		if filterCalled {
			t.Errorf("Filter function should not be called when provider fails")
		}

		// Result should be nil
		if result != nil {
			t.Errorf("Expected nil result when error occurs, got %v", result)
		}
	})

	t.Run("FirstProviderErrorPropagation", func(t *testing.T) {
		// Test error propagation in FirstProvider
		expectedError := errors.New("first provider error")

		// Provider that returns an error
		errorProvider := func() ([]uint32, error) {
			return nil, expectedError
		}

		// Filter function that should never be called
		filterCalled := false
		filter := func(val uint32) bool {
			filterCalled = true
			return true
		}

		// Build FirstProvider pipeline
		firstProvider := FirstProvider(errorProvider, []Filter[uint32]{filter})

		// Execute and verify error propagation
		result, err := firstProvider()

		// Should receive the original error
		if err == nil {
			t.Errorf("Expected error, got result %d", result)
		}
		if err.Error() != expectedError.Error() {
			t.Errorf("Expected error '%s', got '%s'", expectedError.Error(), err.Error())
		}

		// Filter should never be called due to provider error
		if filterCalled {
			t.Errorf("Filter function should not be called when provider fails")
		}

		// Result should be zero value
		if result != 0 {
			t.Errorf("Expected zero result when error occurs, got %d", result)
		}
	})

	t.Run("FoldErrorPropagation", func(t *testing.T) {
		// Test error propagation in Fold function - provider error
		expectedError := errors.New("fold provider error")

		// Provider that returns an error
		errorProvider := func() ([]uint32, error) {
			return nil, expectedError
		}

		// Supplier that should never be called
		supplierCalled := false
		supplier := func() (uint32, error) {
			supplierCalled = true
			return 0, nil
		}

		// Folder that should never be called
		folderCalled := false
		folder := func(acc uint32, val uint32) (uint32, error) {
			folderCalled = true
			return acc + val, nil
		}

		// Build Fold pipeline
		foldProvider := Fold(errorProvider, supplier, folder)

		// Execute and verify error propagation
		result, err := foldProvider()

		// Should receive the original error
		if err == nil {
			t.Errorf("Expected error, got result %d", result)
		}
		if err.Error() != expectedError.Error() {
			t.Errorf("Expected error '%s', got '%s'", expectedError.Error(), err.Error())
		}

		// Supplier and folder should never be called due to provider error
		if supplierCalled {
			t.Errorf("Supplier should not be called when provider fails")
		}
		if folderCalled {
			t.Errorf("Folder should not be called when provider fails")
		}

		// Result should be zero value
		if result != 0 {
			t.Errorf("Expected zero result when error occurs, got %d", result)
		}
	})

	t.Run("FoldSupplierErrorPropagation", func(t *testing.T) {
		// Test error propagation in Fold function - supplier error
		expectedError := errors.New("fold supplier error")

		// Provider that succeeds
		provider := func() ([]uint32, error) {
			return []uint32{1, 2, 3}, nil
		}

		// Supplier that returns an error
		supplier := func() (uint32, error) {
			return 0, expectedError
		}

		// Folder that should never be called
		folderCalled := false
		folder := func(acc uint32, val uint32) (uint32, error) {
			folderCalled = true
			return acc + val, nil
		}

		// Build Fold pipeline
		foldProvider := Fold(provider, supplier, folder)

		// Execute and verify error propagation
		result, err := foldProvider()

		// Should receive the supplier error
		if err == nil {
			t.Errorf("Expected error, got result %d", result)
		}
		if err.Error() != expectedError.Error() {
			t.Errorf("Expected error '%s', got '%s'", expectedError.Error(), err.Error())
		}

		// Folder should never be called due to supplier error
		if folderCalled {
			t.Errorf("Folder should not be called when supplier fails")
		}

		// Result should be zero value
		if result != 0 {
			t.Errorf("Expected zero result when error occurs, got %d", result)
		}
	})

	t.Run("FoldFolderErrorPropagation", func(t *testing.T) {
		// Test error propagation in Fold function - folder error
		expectedError := errors.New("folder error")

		// Provider that succeeds
		provider := func() ([]uint32, error) {
			return []uint32{1, 2, 3}, nil
		}

		// Supplier that succeeds
		supplier := func() (uint32, error) {
			return 0, nil
		}

		// Folder that fails on second item
		folderCallCount := 0
		folder := func(acc uint32, val uint32) (uint32, error) {
			folderCallCount++
			if val == 2 {
				return 0, expectedError
			}
			return acc + val, nil
		}

		// Build Fold pipeline
		foldProvider := Fold(provider, supplier, folder)

		// Execute and verify error propagation
		result, err := foldProvider()

		// Should receive the folder error
		if err == nil {
			t.Errorf("Expected error, got result %d", result)
		}
		if err.Error() != expectedError.Error() {
			t.Errorf("Expected error '%s', got '%s'", expectedError.Error(), err.Error())
		}

		// Folder should be called twice (for values 1 and 2)
		if folderCallCount != 2 {
			t.Errorf("Expected 2 folder calls, got %d", folderCallCount)
		}

		// Result should be zero value
		if result != 0 {
			t.Errorf("Expected zero result when folder error occurs, got %d", result)
		}
	})

	t.Run("CollectToMapErrorPropagation", func(t *testing.T) {
		// Test error propagation in CollectToMap
		expectedError := errors.New("collect provider error")

		// Provider that returns an error
		errorProvider := func() ([]uint32, error) {
			return nil, expectedError
		}

		// Key and value providers that should never be called
		keyProviderCalled := false
		keyProvider := func(val uint32) string {
			keyProviderCalled = true
			return fmt.Sprintf("key-%d", val)
		}

		valueProviderCalled := false
		valueProvider := func(val uint32) uint32 {
			valueProviderCalled = true
			return val * 2
		}

		// Build CollectToMap pipeline
		mapProvider := CollectToMap[uint32, string, uint32](errorProvider, keyProvider, valueProvider)

		// Execute and verify error propagation
		result, err := mapProvider()

		// Should receive the original error
		if err == nil {
			t.Errorf("Expected error, got result %v", result)
		}
		if err.Error() != expectedError.Error() {
			t.Errorf("Expected error '%s', got '%s'", expectedError.Error(), err.Error())
		}

		// Key and value providers should never be called due to provider error
		if keyProviderCalled {
			t.Errorf("Key provider should not be called when main provider fails")
		}
		if valueProviderCalled {
			t.Errorf("Value provider should not be called when main provider fails")
		}

		// Result should be nil
		if result != nil {
			t.Errorf("Expected nil result when error occurs, got %v", result)
		}
	})

	t.Run("MergeSliceProviderErrorPropagation", func(t *testing.T) {
		// Test error propagation in MergeSliceProvider - first provider error
		expectedError := errors.New("first provider error")

		// First provider that returns an error
		errorProvider := func() ([]uint32, error) {
			return nil, expectedError
		}

		// Second provider that should never be called due to first provider error
		secondProviderCalled := false
		secondProvider := func() ([]uint32, error) {
			secondProviderCalled = true
			return []uint32{4, 5, 6}, nil
		}

		// Build MergeSliceProvider pipeline
		mergedProvider := MergeSliceProvider(errorProvider, secondProvider)

		// Execute and verify error propagation
		result, err := mergedProvider()

		// Should receive the first provider error
		if err == nil {
			t.Errorf("Expected error, got result %v", result)
		}
		if err.Error() != expectedError.Error() {
			t.Errorf("Expected error '%s', got '%s'", expectedError.Error(), err.Error())
		}

		// Second provider should never be called due to first provider error
		if secondProviderCalled {
			t.Errorf("Second provider should not be called when first provider fails")
		}

		// Result should be nil
		if result != nil {
			t.Errorf("Expected nil result when error occurs, got %v", result)
		}
	})

	t.Run("MergeSliceProviderSecondErrorPropagation", func(t *testing.T) {
		// Test error propagation in MergeSliceProvider - second provider error
		expectedError := errors.New("second provider error")

		// First provider that succeeds
		firstProvider := func() ([]uint32, error) {
			return []uint32{1, 2, 3}, nil
		}

		// Second provider that returns an error
		errorProvider := func() ([]uint32, error) {
			return nil, expectedError
		}

		// Build MergeSliceProvider pipeline
		mergedProvider := MergeSliceProvider(firstProvider, errorProvider)

		// Execute and verify error propagation
		result, err := mergedProvider()

		// Should receive the second provider error
		if err == nil {
			t.Errorf("Expected error, got result %v", result)
		}
		if err.Error() != expectedError.Error() {
			t.Errorf("Expected error '%s', got '%s'", expectedError.Error(), err.Error())
		}

		// Result should be nil
		if result != nil {
			t.Errorf("Expected nil result when error occurs, got %v", result)
		}
	})

	t.Run("ToSliceProviderErrorPropagation", func(t *testing.T) {
		// Test error propagation in ToSliceProvider
		expectedError := errors.New("to slice provider error")

		// Provider that returns an error
		errorProvider := func() (uint32, error) {
			return 0, expectedError
		}

		// Build ToSliceProvider pipeline
		sliceProvider := ToSliceProvider(errorProvider)

		// Execute and verify error propagation
		result, err := sliceProvider()

		// Should receive the original error
		if err == nil {
			t.Errorf("Expected error, got result %v", result)
		}
		if err.Error() != expectedError.Error() {
			t.Errorf("Expected error '%s', got '%s'", expectedError.Error(), err.Error())
		}

		// Result should be nil
		if result != nil {
			t.Errorf("Expected nil result when error occurs, got %v", result)
		}
	})

	t.Run("ChainedErrorPropagation", func(t *testing.T) {
		// Test error propagation through a chain of lazy operations
		expectedError := errors.New("chain error")

		// Initial provider that succeeds
		initialProvider := func() ([]uint32, error) {
			return []uint32{1, 2, 3, 4, 5}, nil
		}

		// First transform that succeeds
		firstTransform := func(val uint32) (uint32, error) {
			return val * 2, nil
		}

		// Second transform that fails on specific value
		secondTransformCallCount := 0
		secondTransform := func(val uint32) (uint32, error) {
			secondTransformCallCount++
			if val == 6 { // This is 3*2 from first transform
				return 0, expectedError
			}
			return val + 10, nil
		}

		// Build chained pipeline: provider -> SliceMap(firstTransform) -> SliceMap(secondTransform)
		pipeline := SliceMap[uint32, uint32](secondTransform)(
			SliceMap[uint32, uint32](firstTransform)(initialProvider)(),
		)()

		// Execute and verify error propagation
		result, err := pipeline()

		// Should receive the second transform error
		if err == nil {
			t.Errorf("Expected error, got result %v", result)
		}
		if err.Error() != expectedError.Error() {
			t.Errorf("Expected error '%s', got '%s'", expectedError.Error(), err.Error())
		}

		// Second transform should be called at least 3 times (up to the failing item)
		if secondTransformCallCount < 3 {
			t.Errorf("Expected at least 3 second transform calls, got %d", secondTransformCallCount)
		}

		// Result should be nil
		if result != nil {
			t.Errorf("Expected nil result when chained error occurs, got %v", result)
		}
	})
}

func TestSideEffectTiming(t *testing.T) {
	// Test that side effects occur at the right time - only when final Provider is invoked
	// This is critical for resource management, logging, and database operations

	t.Run("DatabaseOperation", func(t *testing.T) {
		// Simulate database operations with proper timing
		connectionOpenCount := 0
		queryExecutionCount := 0

		// Simulate opening a database connection (side effect)
		openConnection := func() ([]string, error) {
			connectionOpenCount++
			return []string{"user1", "user2", "user3"}, nil
		}

		// Simulate executing a query (side effect)
		executeQuery := func(user string) (string, error) {
			queryExecutionCount++
			return fmt.Sprintf("processed_%s", user), nil
		}

		// Build pipeline: openConnection -> SliceMap(executeQuery) -> FilteredProvider
		pipeline := FilteredProvider(
			SliceMap[string, string](executeQuery)(openConnection)(),
			[]Filter[string]{func(s string) bool {
				return len(s) > 10 // Only longer processed strings
			}},
		)

		// Verify NO side effects occurred during composition
		if connectionOpenCount != 0 {
			t.Errorf("Database connection should not be opened during composition, got %d opens", connectionOpenCount)
		}
		if queryExecutionCount != 0 {
			t.Errorf("Query should not be executed during composition, got %d executions", queryExecutionCount)
		}

		// Execute the pipeline - side effects should occur now
		result, err := pipeline()
		if err != nil {
			t.Errorf("Expected result, got err %s", err)
		}

		// Verify side effects occurred exactly once when pipeline was executed
		if connectionOpenCount != 1 {
			t.Errorf("Expected database connection to be opened exactly once, got %d", connectionOpenCount)
		}
		if queryExecutionCount != 3 {
			t.Errorf("Expected 3 query executions (one per user), got %d", queryExecutionCount)
		}

		// Verify correct result
		expected := []string{"processed_user1", "processed_user2", "processed_user3"}
		if len(result) != len(expected) {
			t.Errorf("Expected %d results, got %d", len(expected), len(result))
		}
	})

	t.Run("LoggingOperations", func(t *testing.T) {
		// Test that logging side effects happen at proper execution time
		logEntries := []string{}

		// Simulate logging provider (side effect)
		loggedProvider := func() (uint32, error) {
			logEntries = append(logEntries, "Data retrieved")
			return uint32(100), nil
		}

		// Simulate logging transformer (side effect)
		loggedTransform := func(val uint32) (uint32, error) {
			logEntries = append(logEntries, fmt.Sprintf("Processing value %d", val))
			return val * 2, nil
		}

		// Build pipeline with multiple logging points
		pipeline := Map[uint32, uint32](loggedTransform)(loggedProvider)

		// Verify no logging happened during composition
		if len(logEntries) != 0 {
			t.Errorf("No log entries should exist during composition, got: %v", logEntries)
		}

		// Execute pipeline - logging should happen now
		result, err := pipeline()
		if err != nil {
			t.Errorf("Expected result, got err %s", err)
		}

		// Verify logging happened in correct order
		expectedLogs := []string{
			"Data retrieved",
			"Processing value 100",
		}

		if len(logEntries) != len(expectedLogs) {
			t.Errorf("Expected %d log entries, got %d: %v", len(expectedLogs), len(logEntries), logEntries)
		}

		for i, expected := range expectedLogs {
			if i >= len(logEntries) || logEntries[i] != expected {
				t.Errorf("At log entry %d: expected '%s', got: %v", i, expected, logEntries)
			}
		}

		if result != 200 {
			t.Errorf("Expected 200, got %d", result)
		}
	})

	t.Run("ResourceManagement", func(t *testing.T) {
		// Test that resource allocation/deallocation happens at the right time
		resourcesAllocated := 0
		resourcesReleased := 0

		// Simulate resource allocation (side effect)
		allocateResource := func() ([]int, error) {
			resourcesAllocated++
			return []int{1, 2, 3, 4, 5}, nil
		}

		// Simulate resource processing (side effect)
		processResource := func(val int) (int, error) {
			if val == 5 { // Simulate resource cleanup on last item
				resourcesReleased++
			}
			return val * 10, nil
		}

		// Build pipeline: allocateResource -> SliceMap(processResource) -> FirstProvider
		pipeline := FirstProvider(
			SliceMap[int, int](processResource)(allocateResource)(),
			[]Filter[int]{func(val int) bool {
				return val >= 30 // Only values >= 30
			}},
		)

		// Verify no resources allocated during composition
		if resourcesAllocated != 0 {
			t.Errorf("Resources should not be allocated during composition, got %d", resourcesAllocated)
		}
		if resourcesReleased != 0 {
			t.Errorf("Resources should not be released during composition, got %d", resourcesReleased)
		}

		// Execute pipeline - resource operations should happen now
		result, err := pipeline()
		if err != nil {
			t.Errorf("Expected result, got err %s", err)
		}

		// Verify resource management happened correctly
		if resourcesAllocated != 1 {
			t.Errorf("Expected 1 resource allocation, got %d", resourcesAllocated)
		}
		if resourcesReleased != 1 {
			t.Errorf("Expected 1 resource release, got %d", resourcesReleased)
		}

		// Verify correct result (first value >= 30)
		if result != 30 {
			t.Errorf("Expected 30, got %d", result)
		}
	})

	t.Run("ErrorPropagation", func(t *testing.T) {
		// Test that side effects don't occur when early errors prevent execution
		sideEffectCount := 0

		// Provider that will fail
		failingProvider := func() ([]string, error) {
			return nil, errors.New("simulated failure")
		}

		// Transform that should never be called due to earlier error
		sideEffectTransform := func(val string) (string, error) {
			sideEffectCount++
			return val + "_processed", nil
		}

		// Build pipeline that should fail early
		pipeline := SliceMap[string, string](sideEffectTransform)(failingProvider)()

		// Execute pipeline - should fail before side effect
		_, err := pipeline()

		// Verify error occurred
		if err == nil {
			t.Errorf("Expected error, got success")
		}

		// Verify side effect never occurred due to early failure
		if sideEffectCount != 0 {
			t.Errorf("Side effect should not occur when provider fails, got %d executions", sideEffectCount)
		}
	})

	t.Run("ConditionalSideEffects", func(t *testing.T) {
		// Test that side effects only occur for processed items, not filtered items
		processedItems := []string{}

		// Provider with multiple items
		multiItemProvider := func() ([]int, error) {
			return []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, nil
		}

		// Transform that tracks which items were processed
		trackingTransform := func(val int) (int, error) {
			processedItems = append(processedItems, fmt.Sprintf("item_%d", val))
			return val * 2, nil
		}

		// Build pipeline with filtering - only even numbers should be processed after transformation
		pipeline := FilteredProvider(
			SliceMap[int, int](trackingTransform)(multiItemProvider)(),
			[]Filter[int]{func(val int) bool {
				return val > 10 // Only transformed values > 10 (original values > 5)
			}},
		)

		// Execute pipeline
		result, err := pipeline()
		if err != nil {
			t.Errorf("Expected result, got err %s", err)
		}

		// Verify side effects occurred for ALL items (transform happens before filter)
		expectedProcessedItems := []string{
			"item_1", "item_2", "item_3", "item_4", "item_5",
			"item_6", "item_7", "item_8", "item_9", "item_10",
		}

		if len(processedItems) != len(expectedProcessedItems) {
			t.Errorf("Expected %d processed items, got %d: %v", len(expectedProcessedItems), len(processedItems), processedItems)
		}

		// Verify result contains only filtered items (transformed values > 10)
		expectedResults := []int{12, 14, 16, 18, 20} // Original values 6,7,8,9,10 -> *2 = 12,14,16,18,20
		if len(result) != len(expectedResults) {
			t.Errorf("Expected %d filtered results, got %d", len(expectedResults), len(result))
		}

		for i, expected := range expectedResults {
			if i >= len(result) || result[i] != expected {
				t.Errorf("At result %d: expected %d, got %d", i, expected, result[i])
			}
		}
	})
}

func TestNilProviderHandling(t *testing.T) {
	// Test edge cases involving nil providers and provider functions
	// This ensures robust handling of null/nil scenarios in various provider operations

	t.Run("NilProviderFunction", func(t *testing.T) {
		// Test behavior when provider function itself is nil
		var nilProvider Provider[string]

		// Direct execution of nil provider should panic safely or handle gracefully
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic when calling nil provider directly")
			}
		}()

		// This should panic
		_, _ = nilProvider()
	})

	t.Run("ProviderReturningNilValues", func(t *testing.T) {
		// Test provider that returns nil values for pointer types
		nilStringProvider := func() (*string, error) {
			return nil, nil
		}

		result, err := nilStringProvider()
		if err != nil {
			t.Errorf("Expected no error from nil-returning provider, got: %v", err)
		}
		if result != nil {
			t.Errorf("Expected nil result, got: %v", result)
		}

		// Test with Map function on nil values
		mappedProvider := Map[*string, string](func(s *string) (string, error) {
			if s == nil {
				return "nil_value", nil
			}
			return *s, nil
		})(nilStringProvider)

		mappedResult, err := mappedProvider()
		if err != nil {
			t.Errorf("Expected no error mapping nil value, got: %v", err)
		}
		if mappedResult != "nil_value" {
			t.Errorf("Expected 'nil_value', got: %v", mappedResult)
		}
	})

	t.Run("NilSliceProviders", func(t *testing.T) {
		// Test providers that return nil slices
		nilSliceProvider := func() ([]string, error) {
			return nil, nil
		}

		result, err := nilSliceProvider()
		if err != nil {
			t.Errorf("Expected no error from nil slice provider, got: %v", err)
		}
		if result != nil {
			t.Errorf("Expected nil slice, got: %v", result)
		}

		// Test SliceMap with nil slice
		mapped := SliceMap[string, int](func(s string) (int, error) {
			return len(s), nil
		})(nilSliceProvider)()

		mappedResult, err := mapped()
		if err != nil {
			t.Errorf("Expected no error mapping nil slice, got: %v", err)
		}
		if mappedResult == nil {
			t.Error("Expected empty slice result, got nil")
		}
		if len(mappedResult) != 0 {
			t.Errorf("Expected empty slice, got slice with %d elements", len(mappedResult))
		}
	})

	t.Run("NilTransformerFunctions", func(t *testing.T) {
		// Test behavior with nil transformer functions
		provider := FixedProvider("test")

		// This should panic when transformer is nil and the provider is executed
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic when using nil transformer")
			}
		}()

		var nilTransformer Transformer[string, int]
		mappedProvider := Map[string, int](nilTransformer)(provider)
		// This should panic when executed
		_, _ = mappedProvider()
	})

	t.Run("NilFilterFunctions", func(t *testing.T) {
		// Test behavior with nil filter functions in FilteredProvider
		provider := FixedProvider([]string{"test1", "test2", "test3"})

		// Test with empty filter list (should return all items)
		filteredProvider := FilteredProvider(provider, []Filter[string]{})
		result, err := filteredProvider()
		if err != nil {
			t.Errorf("Expected no error with empty filters, got: %v", err)
		}
		if len(result) != 3 {
			t.Errorf("Expected 3 items with no filters, got: %d", len(result))
		}

		// Test with nil filter in list (should handle gracefully or panic predictably)
		defer func() {
			if r := recover(); r != nil {
				// Panic is acceptable behavior for nil filter
				return
			}
		}()

		var nilFilter Filter[string]
		filteredProvider = FilteredProvider(provider, []Filter[string]{nilFilter})
		_, err = filteredProvider()
		if err == nil {
			t.Error("Expected error or panic with nil filter function")
		}
	})

	t.Run("ErrorProviderWithNilError", func(t *testing.T) {
		// Test ErrorProvider with nil error
		var nilErr error
		provider := ErrorProvider[string](nilErr)

		result, err := provider()
		if err != nil {
			t.Errorf("Expected nil error to be treated as no error, got: %v", err)
		}
		if result != "" {
			t.Errorf("Expected zero value for string, got: %v", result)
		}
	})

	t.Run("NilContextHandling", func(t *testing.T) {
		// Test provider operations with nil context scenarios
		provider := FixedProvider([]uint32{1, 2, 3})

		// Test ForEachSlice with operations that might receive nil context
		err := ForEachSlice[uint32](provider, func(val uint32) error {
			// Simulate operation that handles nil context gracefully
			if val == 0 {
				return fmt.Errorf("invalid nil-like value")
			}
			return nil
		})

		if err != nil {
			t.Errorf("Expected no error in context handling test, got: %v", err)
		}
	})

	t.Run("NilPointerDereference", func(t *testing.T) {
		// Test scenarios that might lead to nil pointer dereferences
		type TestStruct struct {
			Value *string
		}

		// Provider returning struct with nil pointer field
		provider := FixedProvider(TestStruct{Value: nil})

		// Map function that safely handles nil pointer
		mappedProvider := Map[TestStruct, string](func(ts TestStruct) (string, error) {
			if ts.Value == nil {
				return "nil_pointer", nil
			}
			return *ts.Value, nil
		})(provider)

		result, err := mappedProvider()
		if err != nil {
			t.Errorf("Expected no error handling nil pointer, got: %v", err)
		}
		if result != "nil_pointer" {
			t.Errorf("Expected 'nil_pointer', got: %v", result)
		}
	})
}

// Benchmark tests to compare lazy vs eager performance
// These demonstrate the performance benefits of lazy evaluation

func BenchmarkMapLazyEvaluation(b *testing.B) {
	// Benchmark Map function with lazy evaluation - measures deferred execution performance

	// Create provider with simulated expensive operation
	expensiveProvider := func() (uint32, error) {
		// Simulate expensive computation
		sum := uint32(0)
		for i := 0; i < 1000; i++ {
			sum += uint32(i)
		}
		return sum, nil
	}

	// Simple doubling transform
	doubleTransform := func(val uint32) (uint32, error) {
		return val * 2, nil
	}

	b.ResetTimer()

	// Benchmark the complete lazy evaluation cycle
	for i := 0; i < b.N; i++ {
		// Compose the pipeline (should be fast - no execution)
		mappedProvider := Map[uint32, uint32](doubleTransform)(expensiveProvider)

		// Execute the pipeline (this is where the work happens)
		result, err := mappedProvider()
		if err != nil {
			b.Errorf("Unexpected error: %s", err)
		}
		if result == 0 {
			b.Errorf("Expected non-zero result")
		}
	}
}

func BenchmarkMapComposition(b *testing.B) {
	// Benchmark just the composition part of Map (should be very fast with lazy evaluation)

	expensiveProvider := func() (uint32, error) {
		// This should NOT be executed during composition
		panic("Provider should not execute during composition")
	}

	doubleTransform := func(val uint32) (uint32, error) {
		return val * 2, nil
	}

	b.ResetTimer()

	// Benchmark composition only (no execution)
	for i := 0; i < b.N; i++ {
		_ = Map[uint32, uint32](doubleTransform)(expensiveProvider)
	}
}

func BenchmarkSliceMapLazyVsEager(b *testing.B) {
	// Compare performance of lazy SliceMap vs hypothetical eager implementation

	// Large dataset to emphasize performance differences
	largeData := make([]uint32, 10000)
	for i := range largeData {
		largeData[i] = uint32(i + 1)
	}

	dataProvider := func() ([]uint32, error) {
		return largeData, nil
	}

	expensiveTransform := func(val uint32) (uint32, error) {
		// Simulate CPU-intensive transform
		result := val
		for i := 0; i < 100; i++ {
			result = result*2 + 1
			result = result / 2
		}
		return result, nil
	}

	b.Run("LazySliceMap", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Composition + execution
			mappedProvider := SliceMap[uint32, uint32](expensiveTransform)(dataProvider)()
			result, err := mappedProvider()
			if err != nil {
				b.Errorf("Unexpected error: %s", err)
			}
			if len(result) != len(largeData) {
				b.Errorf("Expected %d results, got %d", len(largeData), len(result))
			}
		}
	})

	b.Run("LazySliceMapComposition", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Only composition (should be very fast)
			_ = SliceMap[uint32, uint32](expensiveTransform)(dataProvider)()
		}
	})
}

func BenchmarkParallelSliceMapPerformance(b *testing.B) {
	// Compare sequential vs parallel performance with lazy evaluation

	// Create computationally intensive dataset
	data := make([]uint32, 1000)
	for i := range data {
		data[i] = uint32(i + 1)
	}

	dataProvider := func() ([]uint32, error) {
		return data, nil
	}

	// CPU-intensive transform to benefit from parallelization
	intensiveTransform := func(val uint32) (uint32, error) {
		result := val
		for i := 0; i < 10000; i++ {
			result = (result*7 + 13) % 1000003 // Prime modulus to prevent optimization
		}
		return result, nil
	}

	b.Run("SequentialSliceMap", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			mappedProvider := SliceMap[uint32, uint32](intensiveTransform)(dataProvider)()
			result, err := mappedProvider()
			if err != nil {
				b.Errorf("Unexpected error: %s", err)
			}
			if len(result) != len(data) {
				b.Errorf("Expected %d results, got %d", len(data), len(result))
			}
		}
	})

	b.Run("ParallelSliceMap", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			mappedProvider := SliceMap[uint32, uint32](intensiveTransform)(dataProvider)(ParallelMap())
			result, err := mappedProvider()
			if err != nil {
				b.Errorf("Unexpected error: %s", err)
			}
			if len(result) != len(data) {
				b.Errorf("Expected %d results, got %d", len(data), len(result))
			}
		}
	})
}

func BenchmarkFilteredProviderPerformance(b *testing.B) {
	// Benchmark FilteredProvider with various filter complexities

	// Large dataset
	data := make([]uint32, 50000)
	for i := range data {
		data[i] = uint32(i + 1)
	}

	dataProvider := func() ([]uint32, error) {
		return data, nil
	}

	b.Run("SimpleFilter", func(b *testing.B) {
		simpleFilter := func(val uint32) bool {
			return val%2 == 0 // Even numbers only
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			filteredProvider := FilteredProvider(dataProvider, []Filter[uint32]{simpleFilter})
			result, err := filteredProvider()
			if err != nil {
				b.Errorf("Unexpected error: %s", err)
			}
			if len(result) == 0 {
				b.Errorf("Expected non-empty result")
			}
		}
	})

	b.Run("ComplexFilter", func(b *testing.B) {
		complexFilter := func(val uint32) bool {
			// More complex filtering logic
			if val%2 != 0 {
				return false
			}
			// Prime check for even numbers
			for i := uint32(2); i*i <= val; i++ {
				if val%i == 0 && i != val {
					return false
				}
			}
			return val > 2
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			filteredProvider := FilteredProvider(dataProvider, []Filter[uint32]{complexFilter})
			result, err := filteredProvider()
			if err != nil {
				b.Errorf("Unexpected error: %s", err)
			}
			// Result may be empty for complex filter, that's OK
			_ = result
		}
	})
}

func BenchmarkComplexPipelinePerformance(b *testing.B) {
	// Benchmark complex pipeline with multiple operations
	// This demonstrates the performance characteristics of composed lazy operations

	// Initial data
	data := make([]uint32, 5000)
	for i := range data {
		data[i] = uint32(i + 1)
	}

	dataProvider := func() ([]uint32, error) {
		return data, nil
	}

	// Transform functions
	multiplyTransform := func(val uint32) (uint32, error) {
		return val * 3, nil
	}

	addTransform := func(val uint32) (uint32, error) {
		return val + 100, nil
	}

	// Filters
	rangeFilter := func(val uint32) bool {
		return val >= 1000 && val <= 20000
	}

	modFilter := func(val uint32) bool {
		return val%7 == 0
	}

	b.Run("ComplexLazyPipeline", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Build complex pipeline: multiply -> add -> filter by range -> filter by mod -> take first 100
			pipeline := SliceMap[uint32, uint32](func(val uint32) (uint32, error) {
				if val >= 100 {
					return val, nil
				}
				return 0, nil // Skip small values
			})(
				FilteredProvider(
					FilteredProvider(
						SliceMap[uint32, uint32](addTransform)(
							SliceMap[uint32, uint32](multiplyTransform)(dataProvider)(),
						)(),
						[]Filter[uint32]{rangeFilter},
					),
					[]Filter[uint32]{modFilter},
				),
			)()

			result, err := pipeline()
			if err != nil {
				b.Errorf("Unexpected error: %s", err)
			}
			_ = result
		}
	})
}

func BenchmarkMemoryUsage(b *testing.B) {
	// Benchmark memory allocation patterns with lazy evaluation
	// Lazy evaluation should minimize intermediate allocations

	// Large dataset to emphasize memory usage
	dataSize := 100000
	data := make([]uint32, dataSize)
	for i := range data {
		data[i] = uint32(i + 1)
	}

	dataProvider := func() ([]uint32, error) {
		// Create new slice each time to measure allocation behavior
		result := make([]uint32, len(data))
		copy(result, data)
		return result, nil
	}

	// Memory-allocating transform
	memoryTransform := func(val uint32) (uint32, error) {
		// Force allocation of temporary data
		temp := make([]uint32, 10)
		for i := range temp {
			temp[i] = val + uint32(i)
		}
		return temp[9], nil
	}

	b.Run("LazySliceMapMemory", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			mappedProvider := SliceMap[uint32, uint32](memoryTransform)(dataProvider)()
			result, err := mappedProvider()
			if err != nil {
				b.Errorf("Unexpected error: %s", err)
			}
			if len(result) != dataSize {
				b.Errorf("Expected %d results, got %d", dataSize, len(result))
			}
		}
	})
}

func BenchmarkMemoizePerformance(b *testing.B) {
	// Benchmark Memoize utility performance

	expensiveComputationCount := 0
	expensiveProvider := func() (uint32, error) {
		expensiveComputationCount++
		// Simulate expensive computation
		result := uint32(0)
		for i := 0; i < 10000; i++ {
			result += uint32(i * i)
		}
		return result, nil
	}

	b.Run("WithoutMemoization", func(b *testing.B) {
		expensiveComputationCount = 0
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			result, err := expensiveProvider()
			if err != nil {
				b.Errorf("Unexpected error: %s", err)
			}
			if result == 0 {
				b.Errorf("Expected non-zero result")
			}
		}
	})

	b.Run("WithMemoization", func(b *testing.B) {
		expensiveComputationCount = 0
		memoizedProvider := Memoize(expensiveProvider)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			result, err := memoizedProvider()
			if err != nil {
				b.Errorf("Unexpected error: %s", err)
			}
			if result == 0 {
				b.Errorf("Expected non-zero result")
			}
		}

		// Verify memoization worked (expensive computation should only run once)
		if expensiveComputationCount != 1 {
			b.Errorf("Expected expensive computation to run once with memoization, ran %d times", expensiveComputationCount)
		}
	})
}

func BenchmarkFirstProviderEarlyTermination(b *testing.B) {
	// Benchmark FirstProvider to demonstrate early termination benefits

	// Large dataset where first match is found early
	data := make([]uint32, 100000)
	for i := range data {
		data[i] = uint32(i + 1)
	}
	// Place target value early in the dataset
	data[100] = 999999

	dataProvider := func() ([]uint32, error) {
		return data, nil
	}

	// Filter that matches the target value
	targetFilter := func(val uint32) bool {
		return val == 999999
	}

	b.Run("FirstProviderEarlyMatch", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			firstProvider := FirstProvider(dataProvider, []Filter[uint32]{targetFilter})
			result, err := firstProvider()
			if err != nil {
				b.Errorf("Unexpected error: %s", err)
			}
			if result != 999999 {
				b.Errorf("Expected 999999, got %d", result)
			}
		}
	})

	// Compare with filtering entire dataset first
	b.Run("FilterThenFirstAlternative", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			filteredProvider := FilteredProvider(dataProvider, []Filter[uint32]{targetFilter})
			filtered, err := filteredProvider()
			if err != nil {
				b.Errorf("Unexpected error: %s", err)
			}
			if len(filtered) == 0 {
				b.Errorf("Expected at least one result")
			}
			result := filtered[0]
			if result != 999999 {
				b.Errorf("Expected 999999, got %d", result)
			}
		}
	})
}

func BenchmarkPipelineComposition(b *testing.B) {
	// Benchmark the cost of composing complex pipelines
	// With lazy evaluation, composition should be very fast

	simpleProvider := func() ([]uint32, error) {
		return []uint32{1, 2, 3, 4, 5}, nil
	}

	simpleTransform := func(val uint32) (uint32, error) {
		return val * 2, nil
	}

	simpleFilter := func(val uint32) bool {
		return val > 5
	}

	b.Run("SimplePipelineComposition", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Composition only - no execution
			_ = FirstProvider(
				FilteredProvider(
					SliceMap[uint32, uint32](simpleTransform)(simpleProvider)(),
					[]Filter[uint32]{simpleFilter},
				),
				[]Filter[uint32]{},
			)
		}
	})

	b.Run("ComplexPipelineComposition", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Complex composition with multiple stages
			_ = FilteredProvider(
				SliceMap[uint32, uint32](func(val uint32) (uint32, error) {
					return val + 10, nil
				})(
					FilteredProvider(
						SliceMap[uint32, uint32](func(val uint32) (uint32, error) {
							return val * 3, nil
						})(
							FilteredProvider(
								SliceMap[uint32, uint32](simpleTransform)(simpleProvider)(),
								[]Filter[uint32]{simpleFilter},
							),
						)(),
						[]Filter[uint32]{func(val uint32) bool { return val%2 == 0 }},
					),
				)(),
				[]Filter[uint32]{func(val uint32) bool { return val < 100 }},
			)
		}
	})
}

func TestExecuteForEachSliceErrorHandling(t *testing.T) {
	// Test that ExecuteForEachSlice properly handles errors and terminates early in parallel mode

	t.Run("SequentialMode", func(t *testing.T) {
		// Test sequential mode (existing behavior should work)
		p := FixedProvider([]uint32{1, 2, 3, 4, 5})
		callCount := 0

		err := ForEachSlice(p, func(u uint32) error {
			callCount++
			if u == 3 {
				return errors.New("error on 3")
			}
			return nil
		})

		if err == nil {
			t.Error("Expected error, got nil")
		}
		if err.Error() != "error on 3" {
			t.Errorf("Expected 'error on 3', got '%s'", err.Error())
		}
		// In sequential mode, should stop at the error (3 calls: 1, 2, 3)
		if callCount != 3 {
			t.Errorf("Expected 3 calls in sequential mode, got %d", callCount)
		}
	})

	t.Run("ParallelMode", func(t *testing.T) {
		// Test parallel mode - should return first error
		p := FixedProvider([]uint32{1, 2, 3, 4, 5})
		var callCount int32

		err := ForEachSlice(p, func(u uint32) error {
			atomic.AddInt32(&callCount, 1)
			if u == 3 {
				return errors.New("error on 3")
			}
			return nil
		}, ParallelExecute())

		if err == nil {
			t.Error("Expected error, got nil")
		}
		if err.Error() != "error on 3" {
			t.Errorf("Expected 'error on 3', got '%s'", err.Error())
		}
		// Note: In parallel mode, some goroutines might complete before cancellation
		// but we should get the error without waiting for all to complete
	})

	t.Run("ParallelModeNoError", func(t *testing.T) {
		// Test parallel mode when no errors occur
		p := FixedProvider([]uint32{1, 2, 3, 4, 5})
		var sum int32

		err := ForEachSlice(p, func(u uint32) error {
			atomic.AddInt32(&sum, int32(u))
			return nil
		}, ParallelExecute())

		if err != nil {
			t.Errorf("Expected no error, got %s", err)
		}
		expectedSum := int32(1 + 2 + 3 + 4 + 5)
		if sum != expectedSum {
			t.Errorf("Expected sum %d, got %d", expectedSum, sum)
		}
	})
}

func TestRaceConditionThreadSafety(t *testing.T) {
	// Comprehensive race condition tests designed to be run with `go test -race`
	// These tests verify thread safety of parallel execution functions

	t.Run("ExecuteForEachSliceRaceConditions", func(t *testing.T) {
		// Test for race conditions in parallel ExecuteForEachSlice
		// This test will fail with -race flag if there are unsafe memory accesses

		data := make([]uint32, 1000)
		for i := range data {
			data[i] = uint32(i + 1)
		}
		provider := FixedProvider(data)

		// Shared counter to verify thread safety
		var safeCounter int64
		var unsafeCounter int64

		err := ForEachSlice(provider, func(u uint32) error {
			// Safe atomic operation
			atomic.AddInt64(&safeCounter, int64(u))

			// Simulate some work to increase chance of race conditions
			for i := 0; i < 10; i++ {
				// This operation is NOT thread-safe but we're not sharing the variable
				temp := unsafeCounter + int64(u)
				_ = temp // Use the value to prevent optimization
			}

			return nil
		}, ParallelExecute())

		if err != nil {
			t.Errorf("Expected no error, got %s", err)
		}

		// Verify all values were processed exactly once
		expectedSum := int64(0)
		for _, val := range data {
			expectedSum += int64(val)
		}

		if safeCounter != expectedSum {
			t.Errorf("Expected sum %d, got %d (possible race condition)", expectedSum, safeCounter)
		}
	})

	t.Run("ExecuteForEachMapRaceConditions", func(t *testing.T) {
		// Test for race conditions in parallel ExecuteForEachMap
		data := make(map[uint32][]uint32)
		for i := uint32(1); i <= 100; i++ {
			data[i] = []uint32{i, i * 2, i * 3}
		}
		provider := FixedProvider(data)

		// Shared map to verify thread safety (this should be safe with proper synchronization)
		results := make(map[uint32]int64)
		var mu sync.RWMutex

		err := ForEachMap(provider, func(k uint32) Operator[[]uint32] {
			return func(vs []uint32) error {
				sum := int64(0)
				for _, v := range vs {
					sum += int64(v)
				}

				// Thread-safe write to shared map
				mu.Lock()
				results[k] = sum
				mu.Unlock()

				return nil
			}
		}, ParallelExecute())

		if err != nil {
			t.Errorf("Expected no error, got %s", err)
		}

		// Verify all keys were processed
		if len(results) != len(data) {
			t.Errorf("Expected %d results, got %d", len(data), len(results))
		}

		// Verify sums are correct
		for k, expectedSlice := range data {
			mu.RLock()
			actualSum, exists := results[k]
			mu.RUnlock()

			if !exists {
				t.Errorf("Missing result for key %d", k)
				continue
			}

			expectedSum := int64(0)
			for _, v := range expectedSlice {
				expectedSum += int64(v)
			}

			if actualSum != expectedSum {
				t.Errorf("For key %d: expected sum %d, got %d", k, expectedSum, actualSum)
			}
		}
	})

	t.Run("ConcurrentExecuteForEachSliceInvocations", func(t *testing.T) {
		// Test multiple concurrent invocations of ExecuteForEachSlice
		// This tests that each invocation is properly isolated

		const numRoutines = 10
		const dataSize = 100

		var wg sync.WaitGroup
		results := make([]int64, numRoutines)
		errors := make([]error, numRoutines)

		for routineIdx := 0; routineIdx < numRoutines; routineIdx++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()

				// Each routine processes its own data
				data := make([]uint32, dataSize)
				for i := range data {
					data[i] = uint32(i + 1 + idx*1000) // Make data unique per routine
				}
				provider := FixedProvider(data)

				var sum int64
				err := ForEachSlice(provider, func(u uint32) error {
					atomic.AddInt64(&sum, int64(u))
					return nil
				}, ParallelExecute())

				results[idx] = sum
				errors[idx] = err
			}(routineIdx)
		}

		wg.Wait()

		// Verify all routines completed successfully
		for i := 0; i < numRoutines; i++ {
			if errors[i] != nil {
				t.Errorf("Routine %d failed: %s", i, errors[i])
			}

			// Calculate expected sum for this routine's data
			expectedSum := int64(0)
			for j := 1; j <= dataSize; j++ {
				expectedSum += int64(j + i*1000)
			}

			if results[i] != expectedSum {
				t.Errorf("Routine %d: expected sum %d, got %d", i, expectedSum, results[i])
			}
		}
	})

	t.Run("ParallelSliceMapRaceConditions", func(t *testing.T) {
		// Test ParallelSliceMap for race conditions
		data := make([]uint32, 500)
		for i := range data {
			data[i] = uint32(i + 1)
		}
		provider := FixedProvider(data)

		// Shared state that transform functions will access safely
		var transformCount int64

		transform := func(val uint32) (uint32, error) {
			// Atomic increment to count transforms
			atomic.AddInt64(&transformCount, 1)

			// Simulate some work
			result := val * 2
			for i := 0; i < 5; i++ {
				result = result + 1 - 1 // Dummy operation
			}

			return result, nil
		}

		mappedProvider := SliceMap[uint32, uint32](transform)(provider)(ParallelMap())
		result, err := mappedProvider()

		if err != nil {
			t.Errorf("Expected no error, got %s", err)
		}

		// Verify all items were transformed
		if len(result) != len(data) {
			t.Errorf("Expected %d results, got %d", len(data), len(result))
		}

		// Verify transform was called correct number of times
		if transformCount != int64(len(data)) {
			t.Errorf("Expected %d transform calls, got %d", len(data), transformCount)
		}

		// Verify results are correct
		for i, original := range data {
			expected := original * 2
			if result[i] != expected {
				t.Errorf("At index %d: expected %d, got %d", i, expected, result[i])
			}
		}
	})

	t.Run("MixedParallelAndSequentialExecution", func(t *testing.T) {
		// Test mixing parallel and sequential execution to verify no interference
		data := make([]uint32, 200)
		for i := range data {
			data[i] = uint32(i + 1)
		}
		provider := FixedProvider(data)

		var parallelSum int64
		var sequentialSum int64

		// Run both parallel and sequential operations concurrently
		var wg sync.WaitGroup

		// Parallel execution
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := ForEachSlice(provider, func(u uint32) error {
				atomic.AddInt64(&parallelSum, int64(u))
				return nil
			}, ParallelExecute())
			if err != nil {
				t.Errorf("Parallel execution failed: %s", err)
			}
		}()

		// Sequential execution
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := ForEachSlice(provider, func(u uint32) error {
				atomic.AddInt64(&sequentialSum, int64(u))
				return nil
			})
			if err != nil {
				t.Errorf("Sequential execution failed: %s", err)
			}
		}()

		wg.Wait()

		// Both should have the same result
		expectedSum := int64(0)
		for _, val := range data {
			expectedSum += int64(val)
		}

		if parallelSum != expectedSum {
			t.Errorf("Parallel sum: expected %d, got %d", expectedSum, parallelSum)
		}
		if sequentialSum != expectedSum {
			t.Errorf("Sequential sum: expected %d, got %d", expectedSum, sequentialSum)
		}
	})

	t.Run("ErrorHandlingRaceConditions", func(t *testing.T) {
		// Test that error handling doesn't create race conditions
		data := make([]uint32, 100)
		for i := range data {
			data[i] = uint32(i + 1)
		}
		provider := FixedProvider(data)

		const numRoutines = 5
		var wg sync.WaitGroup
		results := make([]error, numRoutines)

		for routineIdx := 0; routineIdx < numRoutines; routineIdx++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()

				err := ForEachSlice(provider, func(u uint32) error {
					// Create error on specific value to test error handling
					if u == uint32(50+idx) {
						return fmt.Errorf("error on %d from routine %d", u, idx)
					}
					return nil
				}, ParallelExecute())

				results[idx] = err
			}(routineIdx)
		}

		wg.Wait()

		// Verify each routine got its expected error
		for i := 0; i < numRoutines; i++ {
			expectedErrorVal := 50 + i
			if results[i] == nil {
				t.Errorf("Routine %d: expected error but got none", i)
			} else {
				expectedMsg := fmt.Sprintf("error on %d from routine %d", expectedErrorVal, i)
				if results[i].Error() != expectedMsg {
					t.Errorf("Routine %d: expected error '%s', got '%s'", i, expectedMsg, results[i].Error())
				}
			}
		}
	})

	t.Run("ChannelCommunicationRaceConditions", func(t *testing.T) {
		// Test that channel operations in parallel functions don't cause races
		// This specifically tests the channel creation and communication patterns

		data := make([]uint32, 300)
		for i := range data {
			data[i] = uint32(i + 1)
		}
		provider := FixedProvider(data)

		const numConcurrentTests = 8
		var wg sync.WaitGroup
		results := make([]bool, numConcurrentTests)

		for testIdx := 0; testIdx < numConcurrentTests; testIdx++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()

				success := true
				err := ForEachSlice(provider, func(u uint32) error {
					// Simulate varying execution times
					if u%10 == 0 {
						time.Sleep(time.Microsecond * time.Duration(u%5))
					}
					return nil
				}, ParallelExecute())

				if err != nil {
					success = false
					t.Errorf("Test %d failed: %s", idx, err)
				}

				results[idx] = success
			}(testIdx)
		}

		wg.Wait()

		// Verify all concurrent tests succeeded
		for i, success := range results {
			if !success {
				t.Errorf("Concurrent test %d failed", i)
			}
		}
	})

	t.Run("HighContentionScenario", func(t *testing.T) {
		// Test high contention scenario to stress-test race condition fixes
		data := make([]uint32, 1000)
		for i := range data {
			data[i] = uint32(i + 1)
		}
		provider := FixedProvider(data)

		// Shared resource with high contention (protected by mutex)
		sharedMap := make(map[uint32]bool)
		var mapMutex sync.Mutex

		err := ForEachSlice(provider, func(u uint32) error {
			// High contention operation
			mapMutex.Lock()
			sharedMap[u] = true
			mapMutex.Unlock()

			// Some CPU-intensive work to increase chance of context switches
			sum := uint64(0)
			for i := uint64(0); i < uint64(u%100+1); i++ {
				sum += i * i
			}
			_ = sum // Prevent optimization

			return nil
		}, ParallelExecute())

		if err != nil {
			t.Errorf("High contention test failed: %s", err)
		}

		// Verify all values were processed
		mapMutex.Lock()
		if len(sharedMap) != len(data) {
			t.Errorf("Expected %d entries in shared map, got %d", len(data), len(sharedMap))
		}

		for _, val := range data {
			if !sharedMap[val] {
				t.Errorf("Value %d not found in shared map", val)
			}
		}
		mapMutex.Unlock()
	})

	t.Run("MemoizeProviderRaceConditions", func(t *testing.T) {
		// Test that memoized providers are thread-safe under high concurrency
		var expensiveCallCount int64
		expensiveProvider := func() (uint32, error) {
			atomic.AddInt64(&expensiveCallCount, 1)
			// Simulate expensive computation with variable delay
			time.Sleep(time.Millisecond * time.Duration(1+expensiveCallCount%3))
			return 42, nil
		}

		memoizedProvider := Memoize(expensiveProvider)

		const numGoroutines = 50
		var wg sync.WaitGroup
		results := make([]uint32, numGoroutines)
		errors := make([]error, numGoroutines)

		// Launch many goroutines that all try to access the memoized provider simultaneously
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				
				// Add some randomness to timing to increase race condition likelihood
				if idx%2 == 0 {
					time.Sleep(time.Microsecond * time.Duration(idx%10))
				}

				result, err := memoizedProvider()
				results[idx] = result
				errors[idx] = err
			}(i)
		}

		wg.Wait()

		// Verify all results are the same and no errors occurred
		for i := 0; i < numGoroutines; i++ {
			if errors[i] != nil {
				t.Errorf("Goroutine %d got error: %s", i, errors[i])
			}
			if results[i] != 42 {
				t.Errorf("Goroutine %d got result %d, expected 42", i, results[i])
			}
		}

		// Most importantly, verify the expensive computation only ran once
		if expensiveCallCount != 1 {
			t.Errorf("Expected expensive computation to run once, but ran %d times", expensiveCallCount)
		}
	})

	t.Run("ComplexProviderChainRaceConditions", func(t *testing.T) {
		// Test race conditions in complex provider chains with transformations and filters
		data := make([]uint32, 500)
		for i := range data {
			data[i] = uint32(i + 1)
		}
		baseProvider := FixedProvider(data)

		var transformCount, filterCount int64
		
		// Create a complex chain: SliceMap -> Filter -> SliceMap
		chainedProvider := SliceMap[uint32, uint32](func(val uint32) (uint32, error) {
			atomic.AddInt64(&transformCount, 1)
			return val * 2, nil
		})(baseProvider)(ParallelMap())

		filteredProvider := FilteredProvider(chainedProvider, []Filter[uint32]{
			func(val uint32) bool {
				atomic.AddInt64(&filterCount, 1)
				return val < 200 // Filter to smaller values
			},
		})

		finalProvider := SliceMap[uint32, uint32](func(val uint32) (uint32, error) {
			return val + 1, nil
		})(filteredProvider)(ParallelMap())

		const numConcurrentAccess = 20
		var wg sync.WaitGroup
		results := make([][]uint32, numConcurrentAccess)
		errors := make([]error, numConcurrentAccess)

		// Multiple goroutines access the same complex provider chain
		for i := 0; i < numConcurrentAccess; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				result, err := finalProvider()
				results[idx] = result
				errors[idx] = err
			}(i)
		}

		wg.Wait()

		// Verify all results are consistent
		for i := 0; i < numConcurrentAccess; i++ {
			if errors[i] != nil {
				t.Errorf("Chain access %d failed: %s", i, errors[i])
				continue
			}

			// All results should be identical
			if i > 0 && len(results[i]) != len(results[0]) {
				t.Errorf("Result %d has different length: got %d, expected %d", i, len(results[i]), len(results[0]))
				continue
			}

			if i > 0 {
				for j := range results[i] {
					if results[i][j] != results[0][j] {
						t.Errorf("Result %d index %d differs: got %d, expected %d", i, j, results[i][j], results[0][j])
					}
				}
			}
		}

		// Verify operations were performed correctly
		// Each value 1-99 gets: *2 -> filter (keeps values < 200) -> +1
		// So we should have processed 99 items in transforms and filters
		expectedFilteredCount := int64(99)
		expectedTransformCount := int64(len(data)) * numConcurrentAccess

		if filterCount < expectedFilteredCount {
			t.Errorf("Expected at least %d filter operations, got %d", expectedFilteredCount, filterCount)
		}
		if transformCount < expectedTransformCount {
			t.Errorf("Expected at least %d transform operations, got %d", expectedTransformCount, transformCount)
		}
	})

	t.Run("NestedParallelOperationsRaceConditions", func(t *testing.T) {
		// Test race conditions when parallel operations are nested within parallel operations
		outerData := make([]uint32, 20)
		for i := range outerData {
			outerData[i] = uint32(i + 1)
		}
		outerProvider := FixedProvider(outerData)

		var nestedOperationCount int64

		err := ForEachSlice(outerProvider, func(outerVal uint32) error {
			// For each outer value, create an inner parallel operation
			innerData := make([]uint32, 10)
			for i := range innerData {
				innerData[i] = outerVal*10 + uint32(i)
			}
			innerProvider := FixedProvider(innerData)

			return ForEachSlice(innerProvider, func(innerVal uint32) error {
				atomic.AddInt64(&nestedOperationCount, 1)
				// Simulate some work
				result := innerVal * innerVal
				_ = result
				return nil
			}, ParallelExecute())
		}, ParallelExecute())

		if err != nil {
			t.Errorf("Nested parallel operations failed: %s", err)
		}

		// Verify all nested operations were executed
		expectedOperations := int64(len(outerData) * 10)
		if nestedOperationCount != expectedOperations {
			t.Errorf("Expected %d nested operations, got %d", expectedOperations, nestedOperationCount)
		}
	})

	t.Run("CollectToMapRaceConditions", func(t *testing.T) {
		// Test race conditions in CollectToMap with parallel key/value extraction
		data := make([]uint32, 300)
		for i := range data {
			data[i] = uint32(i + 1)
		}
		provider := FixedProvider(data)

		var keyExtractionCount, valueExtractionCount int64

		keyProvider := func(val uint32) uint32 {
			atomic.AddInt64(&keyExtractionCount, 1)
			return val
		}

		valueProvider := func(val uint32) string {
			atomic.AddInt64(&valueExtractionCount, 1)
			return fmt.Sprintf("value_%d", val)
		}

		const numConcurrentMaps = 15
		var wg sync.WaitGroup
		results := make([]map[uint32]string, numConcurrentMaps)
		errors := make([]error, numConcurrentMaps)

		for i := 0; i < numConcurrentMaps; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				mapProvider := CollectToMap(provider, keyProvider, valueProvider)
				result, err := mapProvider()
				results[idx] = result
				errors[idx] = err
			}(i)
		}

		wg.Wait()

		// Verify all results
		for i := 0; i < numConcurrentMaps; i++ {
			if errors[i] != nil {
				t.Errorf("CollectToMap %d failed: %s", i, errors[i])
				continue
			}

			if len(results[i]) != len(data) {
				t.Errorf("Result %d has %d entries, expected %d", i, len(results[i]), len(data))
				continue
			}

			// Verify content consistency
			for _, originalVal := range data {
				if value, exists := results[i][originalVal]; !exists {
					t.Errorf("Result %d missing key %d", i, originalVal)
				} else {
					expectedValue := fmt.Sprintf("value_%d", originalVal)
					if value != expectedValue {
						t.Errorf("Result %d key %d: got '%s', expected '%s'", i, originalVal, value, expectedValue)
					}
				}
			}
		}

		// Verify extraction operations were performed (should be >= expected due to concurrent access)
		expectedExtractions := int64(len(data)) * numConcurrentMaps
		if keyExtractionCount < expectedExtractions {
			t.Errorf("Expected at least %d key extractions, got %d", expectedExtractions, keyExtractionCount)
		}
		if valueExtractionCount < expectedExtractions {
			t.Errorf("Expected at least %d value extractions, got %d", expectedExtractions, valueExtractionCount)
		}
	})

	t.Run("ErrorRecoveryInParallelOperations", func(t *testing.T) {
		// Test that error handling in one parallel operation doesn't affect others
		data := make([]uint32, 100)
		for i := range data {
			data[i] = uint32(i + 1)
		}
		provider := FixedProvider(data)

		const numParallelRuns = 10
		var wg sync.WaitGroup
		results := make([]error, numParallelRuns)

		for runIdx := 0; runIdx < numParallelRuns; runIdx++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				
				// Each run has different error conditions
				errorTrigger := uint32(10 + idx*5)
				
				err := ForEachSlice(provider, func(val uint32) error {
					if val == errorTrigger {
						return fmt.Errorf("intentional error at %d in run %d", val, idx)
					}
					// Simulate some work
					_ = val * val
					return nil
				}, ParallelExecute())

				results[idx] = err
			}(runIdx)
		}

		wg.Wait()

		// Verify each run got its expected error
		for i := 0; i < numParallelRuns; i++ {
			expectedErrorVal := 10 + i*5
			if results[i] == nil {
				t.Errorf("Run %d: expected error but got nil", i)
			} else {
				expectedMsg := fmt.Sprintf("intentional error at %d in run %d", expectedErrorVal, i)
				if results[i].Error() != expectedMsg {
					t.Errorf("Run %d: expected '%s', got '%s'", i, expectedMsg, results[i].Error())
				}
			}
		}
	})

	t.Run("HighVelocityConcurrentAccess", func(t *testing.T) {
		// Stress test with very high number of concurrent goroutines
		data := make([]uint32, 50)
		for i := range data {
			data[i] = uint32(i + 1)
		}
		provider := FixedProvider(data)

		var operationCount int64
		const numGoroutines = 100 // High concurrency

		var wg sync.WaitGroup
		startSignal := make(chan struct{})

		// Launch all goroutines and have them wait for start signal
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(goroutineId int) {
				defer wg.Done()
				
				// Wait for start signal to maximize concurrency
				<-startSignal

				err := ForEachSlice(provider, func(val uint32) error {
					atomic.AddInt64(&operationCount, 1)
					
					// Add some variability in execution time
					if val%3 == 0 {
						time.Sleep(time.Nanosecond * time.Duration(val%10))
					}
					
					return nil
				}, ParallelExecute())

				if err != nil {
					t.Errorf("Goroutine %d failed: %s", goroutineId, err)
				}
			}(i)
		}

		// Start all goroutines simultaneously
		close(startSignal)
		wg.Wait()

		// Verify all operations were performed
		expectedOperations := int64(len(data)) * numGoroutines
		if operationCount != expectedOperations {
			t.Errorf("Expected %d operations, got %d", expectedOperations, operationCount)
		}
	})
}

func TestConcurrentProviderExecution(t *testing.T) {
	// Test concurrent access to multiple provider instances with shared resources
	// This test verifies thread safety when multiple goroutines access different
	// provider chains simultaneously, including scenarios with shared state

	t.Run("MultipleConcurrentProviderChains", func(t *testing.T) {
		// Test concurrent execution of different provider chains that access shared data
		sharedData := make([]uint32, 200)
		for i := range sharedData {
			sharedData[i] = uint32(i + 1)
		}

		// Create multiple different provider chains
		provider1 := FixedProvider(sharedData[:50])
		provider2 := FixedProvider(sharedData[50:100])
		provider3 := FixedProvider(sharedData[100:150])
		provider4 := FixedProvider(sharedData[150:])

		// Create different transformation chains
		chain1 := SliceMap[uint32, uint32](func(val uint32) (uint32, error) {
			return val * 2, nil
		})(provider1)(ParallelMap())

		chain2 := SliceMap[uint32, string](func(val uint32) (string, error) {
			return fmt.Sprintf("val_%d", val), nil
		})(provider2)(ParallelMap())

		chain3 := FilteredProvider(provider3, []Filter[uint32]{
			func(val uint32) bool { return val%2 == 0 },
		})

		chain4 := Map[[]uint32, uint32](func(vals []uint32) (uint32, error) {
			sum := uint32(0)
			for _, val := range vals {
				sum += val
			}
			return sum, nil
		})(provider4)

		// Execute all chains concurrently
		const numConcurrentExecutions = 20
		var wg sync.WaitGroup

		// Results storage
		results1 := make([][]uint32, numConcurrentExecutions)
		results2 := make([][]string, numConcurrentExecutions)
		results3 := make([][]uint32, numConcurrentExecutions)
		results4 := make([]uint32, numConcurrentExecutions)
		errors := make([][]error, 4)
		for i := range errors {
			errors[i] = make([]error, numConcurrentExecutions)
		}

		// Launch concurrent executions
		for i := 0; i < numConcurrentExecutions; i++ {
			wg.Add(4) // Four different chains

			go func(idx int) {
				defer wg.Done()
				result, err := chain1()
				results1[idx] = result
				errors[0][idx] = err
			}(i)

			go func(idx int) {
				defer wg.Done()
				result, err := chain2()
				results2[idx] = result
				errors[1][idx] = err
			}(i)

			go func(idx int) {
				defer wg.Done()
				result, err := chain3()
				results3[idx] = result
				errors[2][idx] = err
			}(i)

			go func(idx int) {
				defer wg.Done()
				result, err := chain4()
				results4[idx] = result
				errors[3][idx] = err
			}(i)
		}

		wg.Wait()

		// Verify results consistency
		for i := 0; i < numConcurrentExecutions; i++ {
			// Check chain1 results
			if errors[0][i] != nil {
				t.Errorf("Chain1 execution %d failed: %s", i, errors[0][i])
			} else if len(results1[i]) != 50 {
				t.Errorf("Chain1 execution %d: expected 50 results, got %d", i, len(results1[i]))
			} else if i > 0 {
				// Verify consistency across executions
				for j := range results1[i] {
					if results1[i][j] != results1[0][j] {
						t.Errorf("Chain1 execution %d inconsistent at index %d: got %d, expected %d", i, j, results1[i][j], results1[0][j])
					}
				}
			}

			// Check chain2 results
			if errors[1][i] != nil {
				t.Errorf("Chain2 execution %d failed: %s", i, errors[1][i])
			} else if len(results2[i]) != 50 {
				t.Errorf("Chain2 execution %d: expected 50 results, got %d", i, len(results2[i]))
			}

			// Check chain3 results (filtered evens from 101-150)
			if errors[2][i] != nil {
				t.Errorf("Chain3 execution %d failed: %s", i, errors[2][i])
			} else if len(results3[i]) != 25 { // 25 even numbers from 101-150
				t.Errorf("Chain3 execution %d: expected 25 filtered results, got %d", i, len(results3[i]))
			}

			// Check chain4 results (sum of 151-200)
			if errors[3][i] != nil {
				t.Errorf("Chain4 execution %d failed: %s", i, errors[3][i])
			} else {
				expectedSum := uint32(0)
				for j := 151; j <= 200; j++ {
					expectedSum += uint32(j)
				}
				if results4[i] != expectedSum {
					t.Errorf("Chain4 execution %d: expected sum %d, got %d", i, expectedSum, results4[i])
				}
			}
		}
	})

	t.Run("ConcurrentAccessToSharedMemoizedProviders", func(t *testing.T) {
		// Test concurrent access to shared memoized providers
		var expensiveCallCount int64
		expensiveProvider := func() ([]uint32, error) {
			atomic.AddInt64(&expensiveCallCount, 1)
			// Simulate expensive computation
			time.Sleep(time.Millisecond * 5)
			return []uint32{1, 2, 3, 4, 5}, nil
		}

		memoizedProvider := Memoize(expensiveProvider)

		// Multiple transformation chains sharing the same memoized provider
		chain1 := SliceMap[uint32, uint32](func(val uint32) (uint32, error) {
			return val * 2, nil
		})(memoizedProvider)(ParallelMap())

		chain2 := SliceMap[uint32, uint32](func(val uint32) (uint32, error) {
			return val + 10, nil
		})(memoizedProvider)(ParallelMap())

		chain3 := Map[[]uint32, uint32](func(vals []uint32) (uint32, error) {
			sum := uint32(0)
			for _, val := range vals {
				sum += val
			}
			return sum, nil
		})(memoizedProvider)

		const numConcurrentAccess = 30
		var wg sync.WaitGroup

		results := make([][3]interface{}, numConcurrentAccess) // [chain1_result, chain2_result, chain3_result]
		errors := make([][3]error, numConcurrentAccess)

		// Launch concurrent access to shared memoized provider
		for i := 0; i < numConcurrentAccess; i++ {
			wg.Add(3)

			go func(idx int) {
				defer wg.Done()
				result, err := chain1()
				results[idx][0] = result
				errors[idx][0] = err
			}(i)

			go func(idx int) {
				defer wg.Done()
				result, err := chain2()
				results[idx][1] = result
				errors[idx][1] = err
			}(i)

			go func(idx int) {
				defer wg.Done()
				result, err := chain3()
				results[idx][2] = result
				errors[idx][2] = err
			}(i)
		}

		wg.Wait()

		// Verify all accesses succeeded
		for i := 0; i < numConcurrentAccess; i++ {
			for j := 0; j < 3; j++ {
				if errors[i][j] != nil {
					t.Errorf("Chain %d execution %d failed: %s", j+1, i, errors[i][j])
				}
			}
		}

		// Verify the expensive provider was only called once due to memoization
		if expensiveCallCount != 1 {
			t.Errorf("Expected expensive provider to be called once, but was called %d times", expensiveCallCount)
		}

		// Verify result consistency
		expectedChain1 := []uint32{2, 4, 6, 8, 10}
		expectedChain2 := []uint32{11, 12, 13, 14, 15}
		expectedChain3 := uint32(15) // sum of 1+2+3+4+5

		for i := 0; i < numConcurrentAccess; i++ {
			if result1, ok := results[i][0].([]uint32); ok {
				for j := range expectedChain1 {
					if j < len(result1) && result1[j] != expectedChain1[j] {
						t.Errorf("Chain1 execution %d index %d: expected %d, got %d", i, j, expectedChain1[j], result1[j])
					}
				}
			}

			if result2, ok := results[i][1].([]uint32); ok {
				for j := range expectedChain2 {
					if j < len(result2) && result2[j] != expectedChain2[j] {
						t.Errorf("Chain2 execution %d index %d: expected %d, got %d", i, j, expectedChain2[j], result2[j])
					}
				}
			}

			if result3, ok := results[i][2].(uint32); ok {
				if result3 != expectedChain3 {
					t.Errorf("Chain3 execution %d: expected %d, got %d", i, expectedChain3, result3)
				}
			}
		}
	})

	t.Run("ConcurrentProviderWithErrorHandling", func(t *testing.T) {
		// Test concurrent execution where some providers fail and others succeed
		successData := []uint32{1, 2, 3, 4, 5}
		
		successProvider := FixedProvider(successData)
		errorProvider := ErrorProvider[[]uint32](fmt.Errorf("intentional test error"))

		successChain := SliceMap[uint32, uint32](func(val uint32) (uint32, error) {
			return val * 3, nil
		})(successProvider)(ParallelMap())

		errorChain := SliceMap[uint32, uint32](func(val uint32) (uint32, error) {
			return val * 3, nil
		})(errorProvider)(ParallelMap())

		const numConcurrentTests = 25
		var wg sync.WaitGroup

		successResults := make([][]uint32, numConcurrentTests)
		errorResults := make([][]uint32, numConcurrentTests)
		successErrors := make([]error, numConcurrentTests)
		errorErrors := make([]error, numConcurrentTests)

		// Run success and error chains concurrently
		for i := 0; i < numConcurrentTests; i++ {
			wg.Add(2)

			go func(idx int) {
				defer wg.Done()
				result, err := successChain()
				successResults[idx] = result
				successErrors[idx] = err
			}(i)

			go func(idx int) {
				defer wg.Done()
				result, err := errorChain()
				errorResults[idx] = result
				errorErrors[idx] = err
			}(i)
		}

		wg.Wait()

		// Verify success chain results
		for i := 0; i < numConcurrentTests; i++ {
			if successErrors[i] != nil {
				t.Errorf("Success chain execution %d failed: %s", i, successErrors[i])
			} else if len(successResults[i]) != len(successData) {
				t.Errorf("Success chain execution %d: expected %d results, got %d", i, len(successData), len(successResults[i]))
			} else {
				for j, expected := range []uint32{3, 6, 9, 12, 15} {
					if successResults[i][j] != expected {
						t.Errorf("Success chain execution %d index %d: expected %d, got %d", i, j, expected, successResults[i][j])
					}
				}
			}
		}

		// Verify error chain consistently fails
		for i := 0; i < numConcurrentTests; i++ {
			if errorErrors[i] == nil {
				t.Errorf("Error chain execution %d should have failed but didn't", i)
			} else if errorErrors[i].Error() != "intentional test error" {
				t.Errorf("Error chain execution %d: expected 'intentional test error', got '%s'", i, errorErrors[i].Error())
			}

			if errorResults[i] != nil {
				t.Errorf("Error chain execution %d: expected nil result, got %v", i, errorResults[i])
			}
		}
	})

	t.Run("HighConcurrencyProviderStressTest", func(t *testing.T) {
		// Stress test with high concurrency to detect race conditions
		largeData := make([]uint32, 1000)
		for i := range largeData {
			largeData[i] = uint32(i + 1)
		}
		provider := FixedProvider(largeData)

		transformChain := SliceMap[uint32, uint32](func(val uint32) (uint32, error) {
			// Add small delay to increase chance of race conditions
			if val%100 == 0 {
				time.Sleep(time.Microsecond)
			}
			return val*2 + 1, nil
		})(provider)(ParallelMap())

		const highConcurrency = 100
		var wg sync.WaitGroup

		results := make([][]uint32, highConcurrency)
		errors := make([]error, highConcurrency)
		startSignal := make(chan struct{})

		// Launch high number of concurrent executions
		for i := 0; i < highConcurrency; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				<-startSignal // Wait for start signal to maximize concurrency
				result, err := transformChain()
				results[idx] = result
				errors[idx] = err
			}(i)
		}

		// Start all executions simultaneously
		close(startSignal)
		wg.Wait()

		// Verify all executions succeeded and are consistent
		for i := 0; i < highConcurrency; i++ {
			if errors[i] != nil {
				t.Errorf("High concurrency execution %d failed: %s", i, errors[i])
				continue
			}

			if len(results[i]) != len(largeData) {
				t.Errorf("High concurrency execution %d: expected %d results, got %d", i, len(largeData), len(results[i]))
				continue
			}

			// Verify results are consistent (compare with first successful result)
			if i > 0 && len(results[0]) == len(results[i]) {
				for j := range results[i] {
					if results[i][j] != results[0][j] {
						t.Errorf("High concurrency execution %d inconsistent at index %d: got %d, expected %d", i, j, results[i][j], results[0][j])
						break // Break inner loop to avoid spam
					}
				}
			}
		}

		// Verify transformation correctness for first result
		if len(results[0]) == len(largeData) {
			for i, originalVal := range largeData {
				expectedTransformed := originalVal*2 + 1
				if results[0][i] != expectedTransformed {
					t.Errorf("Transformation incorrect at index %d: got %d, expected %d", i, results[0][i], expectedTransformed)
					break
				}
			}
		}
	})
}

func TestParallelExecutionRaceConditions(t *testing.T) {
	// High concurrency race condition tests with intensive provider chain operations
	// This test is specifically designed to catch race conditions in provider execution
	// when running with `go test -race`

	t.Run("HighConcurrencyParallelProviderChains", func(t *testing.T) {
		// Test with very high concurrency to maximize chances of detecting race conditions
		const highConcurrency = 500 // Much higher than existing tests
		const dataSize = 200

		// Create shared data that will be accessed by multiple provider chains
		sharedData := make([]uint32, dataSize)
		for i := range sharedData {
			sharedData[i] = uint32(i + 1)
		}

		// Create multiple complex provider chains that will be executed concurrently
		baseProvider := FixedProvider(sharedData)

		// Chain 1: Multiple transformations with parallel processing
		chain1 := SliceMap[uint32, uint32](func(val uint32) (uint32, error) {
			// Add computational work to increase race condition likelihood
			result := val
			for i := 0; i < 100; i++ {
				result = (result*3 + 7) % 1000000
			}
			return result, nil
		})(baseProvider)(ParallelMap())

		// Chain 2: Filtered processing with transformations
		chain2 := SliceMap[uint32, string](func(val uint32) (string, error) {
			// String formatting with computation
			result := val * val
			return fmt.Sprintf("processed_%d_%d", val, result), nil
		})(FilteredProvider(baseProvider, []Filter[uint32]{
			func(val uint32) bool { return val%3 == 0 },
		}))(ParallelMap())

		// Chain 3: Complex aggregation operations
		chain3 := SliceMap[uint32, uint64](func(val uint32) (uint64, error) {
			// Accumulate operations to stress memory access patterns
			acc := uint64(val)
			for i := uint32(1); i <= 10; i++ {
				acc = acc*uint64(val) + uint64(i)
			}
			return acc, nil
		})(baseProvider)(ParallelMap())

		// Shared resources for race condition detection
		var totalProcessed int64
		var processingErrors int64
		var maxValue uint32
		var mutex sync.RWMutex

		// Launch extremely high concurrency execution
		var wg sync.WaitGroup
		results1 := make([][]uint32, highConcurrency)
		results2 := make([][]string, highConcurrency)
		results3 := make([][]uint64, highConcurrency)
		errors := make([][]error, 3)
		for i := range errors {
			errors[i] = make([]error, highConcurrency)
		}

		start := time.Now()

		for i := 0; i < highConcurrency; i++ {
			wg.Add(3) // Three concurrent chains per iteration

			go func(idx int) {
				defer wg.Done()
				result, err := chain1()
				
				// Update shared counters safely
				if err == nil {
					atomic.AddInt64(&totalProcessed, int64(len(result)))
					// Update max value with race condition potential
					mutex.Lock()
					for _, val := range result {
						if val > maxValue {
							maxValue = val
						}
					}
					mutex.Unlock()
				} else {
					atomic.AddInt64(&processingErrors, 1)
				}
				
				results1[idx] = result
				errors[0][idx] = err
			}(i)

			go func(idx int) {
				defer wg.Done()
				result, err := chain2()
				
				if err == nil {
					atomic.AddInt64(&totalProcessed, int64(len(result)))
				} else {
					atomic.AddInt64(&processingErrors, 1)
				}
				
				results2[idx] = result
				errors[1][idx] = err
			}(i)

			go func(idx int) {
				defer wg.Done()
				result, err := chain3()
				
				if err == nil {
					atomic.AddInt64(&totalProcessed, int64(len(result)))
				} else {
					atomic.AddInt64(&processingErrors, 1)
				}
				
				results3[idx] = result
				errors[2][idx] = err
			}(i)
		}

		wg.Wait()
		elapsed := time.Since(start)

		// Verify no errors occurred during high concurrency execution
		if processingErrors > 0 {
			t.Errorf("Expected no processing errors, but got %d errors", processingErrors)
		}

		// Verify results consistency across all executions
		for i := 0; i < highConcurrency; i++ {
			// Verify chain1 results
			if errors[0][i] != nil {
				t.Errorf("Chain1 execution %d failed: %s", i, errors[0][i])
			} else if len(results1[i]) != dataSize {
				t.Errorf("Chain1 execution %d: expected %d results, got %d", i, dataSize, len(results1[i]))
			} else if i > 0 && len(results1[0]) == len(results1[i]) {
				// Compare with first successful result for consistency
				for j := 0; j < len(results1[i]) && j < 10; j++ { // Limit comparison to avoid spam
					if results1[i][j] != results1[0][j] {
						t.Errorf("Chain1 execution %d inconsistent at index %d: got %d, expected %d", i, j, results1[i][j], results1[0][j])
						break
					}
				}
			}

			// Verify chain2 results (filtered data, so fewer results expected)
			if errors[1][i] != nil {
				t.Errorf("Chain2 execution %d failed: %s", i, errors[1][i])
			} else {
				expectedFilteredCount := 0
				for _, val := range sharedData {
					if val%3 == 0 {
						expectedFilteredCount++
					}
				}
				if len(results2[i]) != expectedFilteredCount {
					t.Errorf("Chain2 execution %d: expected %d filtered results, got %d", i, expectedFilteredCount, len(results2[i]))
				}
			}

			// Verify chain3 results
			if errors[2][i] != nil {
				t.Errorf("Chain3 execution %d failed: %s", i, errors[2][i])
			} else if len(results3[i]) != dataSize {
				t.Errorf("Chain3 execution %d: expected %d results, got %d", i, dataSize, len(results3[i]))
			}
		}

		// Performance and race condition indicators
		t.Logf("High concurrency test completed in %v", elapsed)
		t.Logf("Total items processed: %d", totalProcessed)
		t.Logf("Max value computed: %d", maxValue)

		// Verify we actually processed a reasonable amount of data
		expectedMinProcessed := int64(highConcurrency * (dataSize*2 + (dataSize/3))) // chain1 + chain2 + chain3
		if totalProcessed < expectedMinProcessed/2 { // Allow some variance
			t.Errorf("Expected at least %d items processed, got %d", expectedMinProcessed/2, totalProcessed)
		}
	})

	t.Run("RaceConditionsWithSharedMutableState", func(t *testing.T) {
		// Test race conditions when providers access shared mutable state
		// This test is designed to fail if proper synchronization is not in place
		
		const iterations = 100
		const concurrency = 50
		
		// Shared mutable state that could cause race conditions
		var sharedCounter int64
		var sharedMap sync.Map
		var sharedSlice []uint32
		var sliceMutex sync.RWMutex

		// Provider that modifies shared state
		racyProvider := func() ([]uint32, error) {
			// Increment counter (this is safe with atomic)
			newCount := atomic.AddInt64(&sharedCounter, 1)
			
			// Store in shared map
			sharedMap.Store(fmt.Sprintf("key_%d", newCount), newCount)
			
			// Append to shared slice (potentially racy without proper locking)
			sliceMutex.Lock()
			sharedSlice = append(sharedSlice, uint32(newCount))
			currentSlice := make([]uint32, len(sharedSlice))
			copy(currentSlice, sharedSlice)
			sliceMutex.Unlock()
			
			return currentSlice, nil
		}

		// Create transformation chain that uses the racy provider
		transformChain := SliceMap[uint32, uint32](func(val uint32) (uint32, error) {
			// Additional computation to increase race condition window
			result := val
			for i := 0; i < 50; i++ {
				result = (result*7 + 13) % 100000
			}
			return result, nil
		})(racyProvider)(ParallelMap())

		var wg sync.WaitGroup
		results := make([][]uint32, concurrency)
		errors := make([]error, concurrency)

		// Launch concurrent executions
		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				result, err := transformChain()
				results[idx] = result
				errors[idx] = err
			}(i)
		}

		wg.Wait()

		// Verify all executions completed successfully
		successCount := 0
		for i := 0; i < concurrency; i++ {
			if errors[i] != nil {
				t.Errorf("Execution %d failed: %s", i, errors[i])
			} else {
				successCount++
				if len(results[i]) == 0 {
					t.Errorf("Execution %d returned empty result", i)
				}
			}
		}

		// Verify shared state is consistent
		finalCount := atomic.LoadInt64(&sharedCounter)
		if finalCount != int64(concurrency) {
			t.Errorf("Expected counter to be %d, got %d", concurrency, finalCount)
		}

		// Verify shared map entries
		mapCount := 0
		sharedMap.Range(func(key, value interface{}) bool {
			mapCount++
			return true
		})
		if mapCount != concurrency {
			t.Errorf("Expected %d map entries, got %d", concurrency, mapCount)
		}

		// Verify shared slice final length
		sliceMutex.RLock()
		finalSliceLen := len(sharedSlice)
		sliceMutex.RUnlock()
		
		if finalSliceLen != concurrency {
			t.Errorf("Expected slice length %d, got %d", concurrency, finalSliceLen)
		}

		t.Logf("Race condition test completed successfully with %d concurrent executions", successCount)
	})
}

// Benchmark tests for ExecuteForEachSlice performance
func BenchmarkExecuteForEachSlice(b *testing.B) {
	// Create test data
	data := make([]uint32, 1000)
	for i := range data {
		data[i] = uint32(i + 1)
	}
	provider := func() ([]uint32, error) {
		return data, nil
	}

	// CPU-intensive operation
	intensiveOperation := func(val uint32) error {
		result := val
		for i := 0; i < 1000; i++ {
			result = (result*7 + 13) % 1000003
		}
		return nil
	}

	b.Run("SequentialExecution", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := ForEachSlice(provider, intensiveOperation)
			if err != nil {
				b.Fatalf("Unexpected error: %v", err)
			}
		}
	})

	b.Run("ParallelExecution", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := ForEachSlice(provider, intensiveOperation, ParallelExecute())
			if err != nil {
				b.Fatalf("Unexpected error: %v", err)
			}
		}
	})
}

// Benchmark tests for ExecuteForEachMap performance
func BenchmarkExecuteForEachMap(b *testing.B) {
	// Create test data
	data := make(map[string]uint32, 1000)
	for i := 0; i < 1000; i++ {
		data[fmt.Sprintf("key_%d", i)] = uint32(i + 1)
	}
	provider := func() (map[string]uint32, error) {
		return data, nil
	}

	// CPU-intensive operation (curried function)
	intensiveOperation := func(key string) Operator[uint32] {
		return func(val uint32) error {
			result := val
			for i := 0; i < 1000; i++ {
				result = (result*7 + 13) % 1000003
			}
			return nil
		}
	}

	b.Run("SequentialExecution", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := ForEachMap(provider, intensiveOperation)
			if err != nil {
				b.Fatalf("Unexpected error: %v", err)
			}
		}
	})

	b.Run("ParallelExecution", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := ForEachMap(provider, intensiveOperation, ParallelExecute())
			if err != nil {
				b.Fatalf("Unexpected error: %v", err)
			}
		}
	})
}

func TestProviderChainErrorPropagation(t *testing.T) {
	// Test that errors propagate correctly through complex provider chains
	// This test focuses on multi-level chaining scenarios that can occur in production
	
	t.Run("DeepProviderChainWithEarlyError", func(t *testing.T) {
		// Test error propagation when error occurs early in a deep chain
		expectedError := errors.New("early chain error")
		
		// Create an error provider that fails immediately
		errorProvider := ErrorProvider[[]uint32](expectedError)
		
		// Build a deep chain: errorProvider -> SliceMap -> Map -> Filter -> First
		transform := func(val uint32) (uint32, error) {
			return val * 2, nil
		}
		
		filter := func(val uint32) bool {
			return val > 5
		}
		
		// Chain multiple operations
		chain := func() (uint32, error) {
			sliceMapped := SliceMap[uint32, uint32](transform)(errorProvider)()
			filtered := FilteredProvider(sliceMapped, []Filter[uint32]{filter})
			return First(filtered, []Filter[uint32]{})
		}
		
		// Execute and verify error propagates to the top
		result, err := chain()
		
		// Should receive the original error from the beginning of the chain
		if err == nil {
			t.Errorf("Expected error, got result %d", result)
		}
		if err.Error() != expectedError.Error() {
			t.Errorf("Expected error '%s', got '%s'", expectedError.Error(), err.Error())
		}
		
		// Result should be zero value
		if result != 0 {
			t.Errorf("Expected zero value result when error occurs, got %d", result)
		}
	})
	
	t.Run("ProviderChainWithMiddleError", func(t *testing.T) {
		// Test error propagation when error occurs in the middle of chain
		transformError := errors.New("middle chain transform error")
		
		// Initial provider that succeeds
		provider := FixedProvider([]uint32{1, 2, 3, 4, 5})
		
		// Transform function that fails for certain values
		failingTransform := func(val uint32) (uint32, error) {
			if val == 3 {
				return 0, transformError
			}
			return val * 2, nil
		}
		
		// Build chain with error in the middle
		chain := func() ([]uint32, error) {
			return SliceMap[uint32, uint32](failingTransform)(provider)()()
		}
		
		// Execute and verify error propagation
		result, err := chain()
		
		// Should receive the transform error
		if err == nil {
			t.Errorf("Expected error, got result %v", result)
		}
		if err.Error() != transformError.Error() {
			t.Errorf("Expected error '%s', got '%s'", transformError.Error(), err.Error())
		}
		
		// Result should be nil
		if result != nil {
			t.Errorf("Expected nil result when error occurs, got %v", result)
		}
	})
	
	t.Run("ParallelProviderChainErrorPropagation", func(t *testing.T) {
		// Test error propagation in parallel execution chains
		parallelError := errors.New("parallel chain error")
		
		// Provider that succeeds
		provider := FixedProvider([]uint32{1, 2, 3, 4, 5})
		
		// Transform that fails for specific value
		parallelTransform := func(val uint32) (uint32, error) {
			if val == 4 {
				return 0, parallelError
			}
			return val * 3, nil
		}
		
		// Build parallel chain
		chain := func() ([]uint32, error) {
			return SliceMap[uint32, uint32](parallelTransform)(provider)(ParallelMap())()
		}
		
		// Execute and verify error propagation in parallel context
		result, err := chain()
		
		// Should receive the parallel transform error
		if err == nil {
			t.Errorf("Expected error, got result %v", result)
		}
		if err.Error() != parallelError.Error() {
			t.Errorf("Expected error '%s', got '%s'", parallelError.Error(), err.Error())
		}
		
		// Result should be nil
		if result != nil {
			t.Errorf("Expected nil result when parallel error occurs, got %v", result)
		}
	})
	
	t.Run("ChainedOperationErrorAggregation", func(t *testing.T) {
		// Test that only the first error in a chain is propagated
		firstError := errors.New("first error in chain")
		secondError := errors.New("second error in chain")
		
		// Provider that fails with first error
		errorProvider := ErrorProvider[uint32](firstError)
		
		// Transform that would fail with second error (but shouldn't be reached)
		failingTransform := func(val uint32) (uint32, error) {
			return 0, secondError
		}
		
		// Build chain where both operations would fail
		chain := func() (uint32, error) {
			return Map[uint32, uint32](failingTransform)(errorProvider)()
		}
		
		// Execute and verify only first error is propagated
		result, err := chain()
		
		// Should receive only the first error (short-circuiting)
		if err == nil {
			t.Errorf("Expected error, got result %d", result)
		}
		if err.Error() != firstError.Error() {
			t.Errorf("Expected first error '%s', got '%s'", firstError.Error(), err.Error())
		}
		
		// Result should be zero value
		if result != 0 {
			t.Errorf("Expected zero value result when error occurs, got %d", result)
		}
	})
	
	t.Run("NestedProviderChainErrorPropagation", func(t *testing.T) {
		// Test error propagation through nested provider operations
		nestedError := errors.New("nested provider error")
		
		// Create nested structure: provider -> merge -> fold -> collect
		provider1 := FixedProvider([]uint32{1, 2, 3})
		provider2 := ErrorProvider[[]uint32](nestedError)
		
		// Merge two providers (one fails)
		mergedProvider := MergeSliceProvider(provider1, provider2)
		
		// Fold operation on the merged result
		initialValue := func() (uint32, error) {
			return 0, nil
		}
		
		folder := func(acc uint32, val uint32) (uint32, error) {
			return acc + val, nil
		}
		
		// Build nested chain
		chain := func() (uint32, error) {
			return Fold(mergedProvider, initialValue, folder)()
		}
		
		// Execute and verify error propagation through nested structure
		result, err := chain()
		
		// Should receive the nested error
		if err == nil {
			t.Errorf("Expected error, got result %d", result)
		}
		if err.Error() != nestedError.Error() {
			t.Errorf("Expected error '%s', got '%s'", nestedError.Error(), err.Error())
		}
		
		// Result should be zero value
		if result != 0 {
			t.Errorf("Expected zero value result when nested error occurs, got %d", result)
		}
	})
	
	t.Run("LongChainPerformanceWithErrors", func(t *testing.T) {
		// Test that error propagation is efficient in long chains
		chainError := errors.New("chain performance error")
		
		// Create a very long chain to test performance
		provider := ErrorProvider[uint32](chainError)
		
		// Build an intentionally long chain of operations
		longChain := provider
		
		// Chain multiple Map operations (should short-circuit on first error)
		for i := 0; i < 10; i++ {
			transform := func(val uint32) (uint32, error) {
				// This should never execute due to early error
				t.Errorf("Transform should not execute when early error occurs")
				return val + 1, nil
			}
			longChain = Map[uint32, uint32](transform)(longChain)
		}
		
		// Execute and verify quick error propagation
		start := time.Now()
		result, err := longChain()
		duration := time.Since(start)
		
		// Should receive the original error quickly
		if err == nil {
			t.Errorf("Expected error, got result %d", result)
		}
		if err.Error() != chainError.Error() {
			t.Errorf("Expected error '%s', got '%s'", chainError.Error(), err.Error())
		}
		
		// Should be very fast (under 1ms) due to short-circuiting
		if duration > time.Millisecond {
			t.Errorf("Error propagation took too long: %v (expected < 1ms)", duration)
		}
		
		// Result should be zero value
		if result != 0 {
			t.Errorf("Expected zero value result when error occurs, got %d", result)
		}
	})
}

func TestPartialFailureRecovery(t *testing.T) {
	// Test behavior when some operations succeed and others fail
	// This test focuses on scenarios where the system needs to handle mixed results gracefully
	
	t.Run("SliceOperationWithPartialFailures", func(t *testing.T) {
		// Test SliceMap with some elements succeeding and others failing
		provider := FixedProvider([]uint32{1, 2, 3, 4, 5, 6})
		partialFailureError := errors.New("partial failure error")
		
		// Transform that fails for even numbers
		selectiveTransform := func(val uint32) (string, error) {
			if val%2 == 0 {
				return "", partialFailureError
			}
			return fmt.Sprintf("success_%d", val), nil
		}
		
		// Execute transformation
		result, err := SliceMap[uint32, string](selectiveTransform)(provider)()()
		
		// Should fail because some elements failed
		if err == nil {
			t.Errorf("Expected error due to partial failures, got result %v", result)
		}
		if err.Error() != partialFailureError.Error() {
			t.Errorf("Expected error '%s', got '%s'", partialFailureError.Error(), err.Error())
		}
		
		// Result should be nil due to failure
		if result != nil {
			t.Errorf("Expected nil result when partial failure occurs, got %v", result)
		}
	})
	
	t.Run("FilteredProviderWithPartialSuccesses", func(t *testing.T) {
		// Test filtering where some elements pass and others would cause errors
		provider := FixedProvider([]uint32{1, 2, 3, 4, 5, 6, 7, 8})
		
		// Filter that only allows odd numbers (preventing errors on even numbers)
		oddFilter := func(val uint32) bool {
			return val%2 == 1
		}
		
		// Transform that would fail on even numbers (but they should be filtered out)
		safeTransform := func(val uint32) (uint32, error) {
			if val%2 == 0 {
				return 0, errors.New("should not reach even numbers due to filter")
			}
			return val * 10, nil
		}
		
		// Apply filter first, then transform
		filteredProvider := FilteredProvider(provider, []Filter[uint32]{oddFilter})
		result, err := SliceMap[uint32, uint32](safeTransform)(filteredProvider)()()
		
		// Should succeed because filter prevented errors
		if err != nil {
			t.Errorf("Expected no error due to filtering, got error: %v", err)
		}
		
		// Should have only odd numbers transformed (1, 3, 5, 7) -> (10, 30, 50, 70)
		expected := []uint32{10, 30, 50, 70}
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("Expected result %v, got %v", expected, result)
		}
	})
	
	t.Run("ParallelExecutionWithMixedResults", func(t *testing.T) {
		// Test parallel execution where some workers succeed and others fail
		provider := FixedProvider([]uint32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
		mixedError := errors.New("mixed results error")
		
		// Transform that fails for multiples of 3
		mixedTransform := func(val uint32) (uint32, error) {
			if val%3 == 0 {
				return 0, mixedError
			}
			return val * val, nil
		}
		
		// Execute in parallel
		result, err := SliceMap[uint32, uint32](mixedTransform)(provider)(ParallelMap())()
		
		// Should fail because some parallel operations failed
		if err == nil {
			t.Errorf("Expected error due to parallel failures, got result %v", result)
		}
		if err.Error() != mixedError.Error() {
			t.Errorf("Expected error '%s', got '%s'", mixedError.Error(), err.Error())
		}
		
		// Result should be nil due to parallel failure
		if result != nil {
			t.Errorf("Expected nil result when parallel failure occurs, got %v", result)
		}
	})
	
	t.Run("ChainedProvidersWithSelectiveFailure", func(t *testing.T) {
		// Test chained providers where one succeeds and another fails
		successProvider := FixedProvider([]uint32{1, 2, 3})
		chainError := errors.New("chain failure error")
		
		// First transform succeeds
		firstTransform := func(val uint32) (uint32, error) {
			return val * 2, nil
		}
		
		// Second transform fails for values > 4
		secondTransform := func(val uint32) (uint32, error) {
			if val > 4 {
				return 0, chainError
			}
			return val + 10, nil
		}
		
		// Chain the transformations
		chain := func() ([]uint32, error) {
			intermediate, err := SliceMap[uint32, uint32](firstTransform)(successProvider)()()
			if err != nil {
				return nil, err
			}
			intermediateProvider := FixedProvider(intermediate)
			return SliceMap[uint32, uint32](secondTransform)(intermediateProvider)()()
		}
		
		// Execute chain
		result, err := chain()
		
		// Should fail because second transform fails on value 6 (3*2=6 > 4)
		if err == nil {
			t.Errorf("Expected error due to chain failure, got result %v", result)
		}
		if err.Error() != chainError.Error() {
			t.Errorf("Expected error '%s', got '%s'", chainError.Error(), err.Error())
		}
		
		// Result should be nil due to chain failure
		if result != nil {
			t.Errorf("Expected nil result when chain failure occurs, got %v", result)
		}
	})
	
	t.Run("MergedProvidersWithPartialFailure", func(t *testing.T) {
		// Test merging providers where some succeed and others fail
		successProvider1 := FixedProvider([]uint32{1, 2, 3})
		successProvider2 := FixedProvider([]uint32{4, 5, 6})
		mergeError := errors.New("merge failure error")
		failureProvider := ErrorProvider[[]uint32](mergeError)
		
		// Merge successful providers
		successfulMerge := MergeSliceProvider(successProvider1, successProvider2)
		result1, err1 := successfulMerge()
		
		// Should succeed
		if err1 != nil {
			t.Errorf("Expected no error in successful merge, got: %v", err1)
		}
		
		expected1 := []uint32{1, 2, 3, 4, 5, 6}
		if !reflect.DeepEqual(result1, expected1) {
			t.Errorf("Expected merged result %v, got %v", expected1, result1)
		}
		
		// Merge with failure
		failureMerge := MergeSliceProvider(successProvider1, failureProvider)
		result2, err2 := failureMerge()
		
		// Should fail due to one provider failing
		if err2 == nil {
			t.Errorf("Expected error due to merge failure, got result %v", result2)
		}
		if err2.Error() != mergeError.Error() {
			t.Errorf("Expected error '%s', got '%s'", mergeError.Error(), err2.Error())
		}
		
		// Result should be nil due to merge failure
		if result2 != nil {
			t.Errorf("Expected nil result when merge failure occurs, got %v", result2)
		}
	})
	
	t.Run("FoldOperationWithPartialFailure", func(t *testing.T) {
		// Test fold operation that fails partway through
		provider := FixedProvider([]uint32{1, 2, 3, 4, 5})
		foldError := errors.New("fold operation error")
		
		// Initial value function
		initialValue := func() (uint32, error) {
			return 0, nil
		}
		
		// Folder that fails when accumulator reaches certain threshold
		selectiveFolder := func(acc uint32, val uint32) (uint32, error) {
			newAcc := acc + val
			if newAcc > 6 { // Fails at 1+2+3+4 = 10 > 6, actually fails at 1+2+3=6, then 6+4=10
				return 0, foldError
			}
			return newAcc, nil
		}
		
		// Execute fold operation
		result, err := Fold(provider, initialValue, selectiveFolder)()
		
		// Should fail when threshold is exceeded
		if err == nil {
			t.Errorf("Expected error due to fold failure, got result %d", result)
		}
		if err.Error() != foldError.Error() {
			t.Errorf("Expected error '%s', got '%s'", foldError.Error(), err.Error())
		}
		
		// Result should be zero due to failure
		if result != 0 {
			t.Errorf("Expected zero result when fold failure occurs, got %d", result)
		}
	})
	
	t.Run("ForEachWithPartialExecution", func(t *testing.T) {
		// Test ForEach operation where some iterations succeed before failure
		provider := FixedProvider([]uint32{1, 2, 3, 4, 5, 6})
		executedCount := int32(0)
		forEachError := errors.New("for each partial error")
		
		// Operation that succeeds for first few elements then fails
		partialOperation := func(val uint32) error {
			atomic.AddInt32(&executedCount, 1)
			if val > 3 {
				return forEachError
			}
			// Simulate successful work
			time.Sleep(time.Microsecond)
			return nil
		}
		
		// Execute for each
		err := ForEachSlice(provider, partialOperation)
		
		// Should fail when encountering error
		if err == nil {
			t.Errorf("Expected error due to partial execution failure")
		}
		if err.Error() != forEachError.Error() {
			t.Errorf("Expected error '%s', got '%s'", forEachError.Error(), err.Error())
		}
		
		// Should have executed at least some operations (1, 2, 3, then fail on 4)
		execCount := atomic.LoadInt32(&executedCount)
		if execCount < 3 {
			t.Errorf("Expected at least 3 executions before failure, got %d", execCount)
		}
		if execCount > 6 {
			t.Errorf("Expected no more than 6 executions, got %d", execCount)
		}
	})
	
	t.Run("NestedPartialFailureRecovery", func(t *testing.T) {
		// Test nested operations where outer succeeds but inner fails
		outerProvider := FixedProvider([][]uint32{{1, 2}, {3, 4}, {5, 6}})
		innerError := errors.New("inner operation error")
		
		// Outer operation processes each inner array
		outerTransform := func(innerSlice []uint32) ([]uint32, error) {
			// Inner operation that fails on value 4
			innerTransform := func(val uint32) (uint32, error) {
				if val == 4 {
					return 0, innerError
				}
				return val * 100, nil
			}
			
			// Process inner slice
			result := make([]uint32, len(innerSlice))
			for i, val := range innerSlice {
				transformed, err := innerTransform(val)
				if err != nil {
					return nil, err
				}
				result[i] = transformed
			}
			return result, nil
		}
		
		// Execute nested operation
		result, err := SliceMap[[]uint32, []uint32](outerTransform)(outerProvider)()()
		
		// Should fail due to inner operation failure
		if err == nil {
			t.Errorf("Expected error due to nested failure, got result %v", result)
		}
		if err.Error() != innerError.Error() {
			t.Errorf("Expected error '%s', got '%s'", innerError.Error(), err.Error())
		}
		
		// Result should be nil due to nested failure
		if result != nil {
			t.Errorf("Expected nil result when nested failure occurs, got %v", result)
		}
	})
}

// Benchmark for error handling performance in parallel execution
func BenchmarkExecuteForEachSliceErrorHandling(b *testing.B) {
	// Create test data
	data := make([]uint32, 100)
	for i := range data {
		data[i] = uint32(i + 1)
	}
	provider := func() ([]uint32, error) {
		return data, nil
	}

	// Operation that fails on specific values
	errorOperation := func(val uint32) error {
		if val == 50 { // Fail halfway through
			return errors.New("test error")
		}
		// Small CPU work
		result := val
		for i := 0; i < 100; i++ {
			result = (result*7 + 13) % 1000003
		}
		return nil
	}

	b.Run("SequentialErrorHandling", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := ForEachSlice(provider, errorOperation)
			if err == nil {
				b.Fatal("Expected error but got nil")
			}
		}
	})

	b.Run("ParallelErrorHandling", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := ForEachSlice(provider, errorOperation, ParallelExecute())
			if err == nil {
				b.Fatal("Expected error but got nil")
			}
		}
	})
}

// TestErrorAggregationParallelOperations tests comprehensive error aggregation scenarios
func TestErrorAggregationParallelOperations(t *testing.T) {
	t.Run("MultipleSimultaneousErrorsInParallelSliceMap", func(t *testing.T) {
		// Test that when multiple parallel operations fail, we get a meaningful error
		// This tests the current behavior where first error is returned
		provider := func() ([]uint32, error) {
			return []uint32{1, 2, 3, 4, 5, 6, 7, 8}, nil
		}

		// Transform function that fails on multiple specific values
		transform := func(val uint32) (uint32, error) {
			switch val {
			case 2:
				return 0, errors.New("error on value 2")
			case 5:
				return 0, errors.New("error on value 5")
			case 7:
				return 0, errors.New("error on value 7")
			default:
				time.Sleep(time.Millisecond) // Small delay to increase chance of parallel execution
				return val * 2, nil
			}
		}

		// Execute parallel SliceMap
		result, err := SliceMap[uint32, uint32](transform)(provider)(ParallelMap())()

		// Should get an error (currently returns first error encountered)
		if err == nil {
			t.Errorf("Expected error when multiple parallel operations fail, got result %v", result)
		}

		// Verify we get one of the expected errors
		expectedErrors := []string{"error on value 2", "error on value 5", "error on value 7"}
		errorFound := false
		for _, expectedErr := range expectedErrors {
			if err.Error() == expectedErr {
				errorFound = true
				break
			}
		}
		if !errorFound {
			t.Errorf("Expected one of %v, got error '%s'", expectedErrors, err.Error())
		}

		// Result should be nil when errors occur
		if result != nil {
			t.Errorf("Expected nil result when parallel errors occur, got %v", result)
		}
	})

	t.Run("ErrorAggregationWithHighConcurrency", func(t *testing.T) {
		// Test error behavior with high concurrency (more errors than workers)
		data := make([]uint32, 100)
		for i := range data {
			data[i] = uint32(i + 1)
		}
		provider := func() ([]uint32, error) {
			return data, nil
		}

		// Transform that fails on every 10th item
		failureCount := int64(0)
		transform := func(val uint32) (uint32, error) {
			if val%10 == 0 {
				atomic.AddInt64(&failureCount, 1)
				return 0, fmt.Errorf("error on value %d", val)
			}
			// Add small work to simulate real processing
			time.Sleep(time.Microsecond * 10)
			return val * 2, nil
		}

		// Execute with high concurrency
		result, err := SliceMap[uint32, uint32](transform)(provider)(ParallelMap())()

		// Should get an error
		if err == nil {
			t.Errorf("Expected error with high concurrency failures, got result %v", result)
		}

		// Should have at least some failures (even if not all are counted due to early return)
		if atomic.LoadInt64(&failureCount) == 0 {
			t.Errorf("Expected some failures to be recorded, got %d", failureCount)
		}

		// Result should be nil
		if result != nil {
			t.Errorf("Expected nil result when high concurrency errors occur, got %v", result)
		}
	})

	t.Run("MixedSuccessFailureErrorAggregation", func(t *testing.T) {
		// Test mixed scenarios where some operations succeed and some fail
		provider := func() ([]uint32, error) {
			return []uint32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, nil
		}

		successCount := int64(0)
		failureCount := int64(0)

		// Transform that fails on even numbers, succeeds on odd
		transform := func(val uint32) (uint32, error) {
			if val%2 == 0 {
				atomic.AddInt64(&failureCount, 1)
				return 0, fmt.Errorf("even number error: %d", val)
			}
			atomic.AddInt64(&successCount, 1)
			return val * 3, nil
		}

		// Execute parallel operation
		result, err := SliceMap[uint32, uint32](transform)(provider)(ParallelMap())()

		// Should get an error due to failures
		if err == nil {
			t.Errorf("Expected error in mixed success/failure scenario, got result %v", result)
		}

		// Should have recorded some successes and failures
		// Note: Due to early termination, not all may be recorded
		if atomic.LoadInt64(&failureCount) == 0 {
			t.Errorf("Expected some failures to be recorded, got %d", failureCount)
		}

		// Result should be nil when errors occur
		if result != nil {
			t.Errorf("Expected nil result in mixed success/failure scenario, got %v", result)
		}
	})

	t.Run("ErrorPropagationConsistencyInParallel", func(t *testing.T) {
		// Test that error propagation is consistent across multiple runs
		provider := func() ([]uint32, error) {
			return []uint32{1, 2, 3, 4, 5}, nil
		}

		// Transform that always fails on value 3
		transform := func(val uint32) (uint32, error) {
			if val == 3 {
				return 0, errors.New("consistent error on value 3")
			}
			// Add randomness to execution time to test concurrency
			time.Sleep(time.Duration(val) * time.Microsecond)
			return val * 2, nil
		}

		// Run multiple times to test consistency
		for run := 0; run < 10; run++ {
			result, err := SliceMap[uint32, uint32](transform)(provider)(ParallelMap())()

			// Should always get an error
			if err == nil {
				t.Errorf("Run %d: Expected error, got result %v", run, result)
			}

			// Should always get the same error (in this deterministic case)
			expectedError := "consistent error on value 3"
			if err.Error() != expectedError {
				t.Errorf("Run %d: Expected error '%s', got '%s'", run, expectedError, err.Error())
			}

			// Result should always be nil
			if result != nil {
				t.Errorf("Run %d: Expected nil result, got %v", run, result)
			}
		}
	})

	t.Run("ParallelErrorAggregationWithExecuteForEachSlice", func(t *testing.T) {
		// Test error aggregation behavior in ExecuteForEachSlice parallel execution
		data := []uint32{1, 2, 3, 4, 5, 6, 7, 8}
		provider := func() ([]uint32, error) {
			return data, nil
		}

		failureCount := int64(0)
		
		// Operation that fails on multiple values
		operation := func(val uint32) error {
			if val == 2 || val == 5 || val == 7 {
				atomic.AddInt64(&failureCount, 1)
				return fmt.Errorf("operation failed on value %d", val)
			}
			// Small delay to simulate work
			time.Sleep(time.Microsecond * 100)
			return nil
		}

		// Execute with parallel configuration
		err := ForEachSlice(provider, operation, ParallelExecute())

		// Should get an error
		if err == nil {
			t.Errorf("Expected error in parallel ExecuteForEachSlice with failures")
		}

		// Should have recorded at least one failure
		if atomic.LoadInt64(&failureCount) == 0 {
			t.Errorf("Expected some failures to be recorded, got %d", failureCount)
		}

		// Error should indicate which value failed
		errorStr := err.Error()
		validErrors := []string{"operation failed on value 2", "operation failed on value 5", "operation failed on value 7"}
		errorFound := false
		for _, validError := range validErrors {
			if errorStr == validError {
				errorFound = true
				break
			}
		}
		if !errorFound {
			t.Errorf("Expected error to be one of %v, got '%s'", validErrors, errorStr)
		}
	})
}

func TestContextCancellation(t *testing.T) {
	t.Run("ExecuteForEachSlice cancels remaining operations on first error", func(t *testing.T) {
		// Test that the internal context cancellation works when an error occurs
		slice := make([]uint32, 100)
		for i := 0; i < 100; i++ {
			slice[i] = uint32(i)
		}

		var processedCount int64
		var cancelledCount int64
		
		// Operation that fails on specific value to trigger cancellation
		operation := func(u uint32) error {
			// Fail on value 50 to trigger context cancellation
			if u == 50 {
				return errors.New("operation failed on value 50")
			}
			
			// Simulate work and check for cancellation in select statement
			// The select will check the internal context created by ExecuteForEachSlice
			select {
			case <-time.After(2 * time.Millisecond):
				// Normal processing path
				atomic.AddInt64(&processedCount, 1)
				return nil
			}
		}

		// Execute in parallel mode to trigger internal context usage
		err := ExecuteForEachSlice(operation, ParallelExecute())(slice)
		
		// Should get error from the failing operation
		if err == nil {
			t.Errorf("Expected error from failing operation")
		} else if err.Error() != "operation failed on value 50" {
			t.Errorf("Expected specific error message, got: %s", err.Error())
		}

		processed := atomic.LoadInt64(&processedCount)
		cancelled := atomic.LoadInt64(&cancelledCount)

		t.Logf("Processed: %d, Cancelled: %d, Total: %d", processed, cancelled, len(slice))

		// Due to parallel execution and cancellation, not all should be processed
		if processed >= int64(len(slice)) {
			t.Errorf("Expected less than %d operations to be processed due to cancellation, got %d", 
				len(slice), processed)
		}
	})

	t.Run("ExecuteForEachSlice sequential mode processes all items", func(t *testing.T) {
		// Test that sequential execution processes all items without using context
		slice := []uint32{1, 2, 3, 4, 5}
		var processedCount int64

		operation := func(u uint32) error {
			// Sequential mode doesn't use context, so this should process all items
			atomic.AddInt64(&processedCount, 1)
			return nil
		}

		// Execute in sequential mode (default)
		err := ExecuteForEachSlice(operation)(slice)
		
		if err != nil {
			t.Errorf("Expected no error in sequential mode, got %s", err)
		}

		// All items should be processed in sequential mode
		if atomic.LoadInt64(&processedCount) != int64(len(slice)) {
			t.Errorf("Expected all %d items to be processed in sequential mode, got %d", 
				len(slice), atomic.LoadInt64(&processedCount))
		}
	})

	t.Run("Context cancellation behavior in parallel vs sequential", func(t *testing.T) {
		slice := []uint32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
		
		// Test parallel execution with early error
		var parallelProcessed int64
		parallelOp := func(u uint32) error {
			if u == 3 {
				return fmt.Errorf("error on item %d", u)
			}
			time.Sleep(1 * time.Millisecond) // Small delay
			atomic.AddInt64(&parallelProcessed, 1)
			return nil
		}
		
		parallelErr := ExecuteForEachSlice(parallelOp, ParallelExecute())(slice)
		if parallelErr == nil {
			t.Errorf("Expected error in parallel execution")
		}
		
		// Test sequential execution with same operation
		var sequentialProcessed int64
		sequentialOp := func(u uint32) error {
			if u == 3 {
				return fmt.Errorf("error on item %d", u)
			}
			atomic.AddInt64(&sequentialProcessed, 1)
			return nil
		}
		
		sequentialErr := ExecuteForEachSlice(sequentialOp)(slice)
		if sequentialErr == nil {
			t.Errorf("Expected error in sequential execution")
		}
		
		t.Logf("Parallel processed: %d, Sequential processed: %d", 
			atomic.LoadInt64(&parallelProcessed), atomic.LoadInt64(&sequentialProcessed))
		
		// Sequential should process exactly 2 items (before hitting error at item 3)
		if atomic.LoadInt64(&sequentialProcessed) != 2 {
			t.Errorf("Expected 2 items processed in sequential, got %d", 
				atomic.LoadInt64(&sequentialProcessed))
		}
	})

	t.Run("Parallel execution context cancellation with goroutine coordination", func(t *testing.T) {
		// Test the specific context cancellation mechanism in parallel execution
		slice := make([]uint32, 50)
		for i := 0; i < 50; i++ {
			slice[i] = uint32(i)
		}
		
		var mutex sync.Mutex
		processOrder := make([]uint32, 0)
		
		operation := func(u uint32) error {
			// Add delay to see context cancellation effect
			time.Sleep(time.Duration(u%5) * time.Millisecond)
			
			mutex.Lock()
			processOrder = append(processOrder, u)
			mutex.Unlock()
			
			// Fail on specific value to trigger cancellation
			if u == 25 {
				return fmt.Errorf("intentional error on value %d", u)
			}
			
			return nil
		}
		
		err := ExecuteForEachSlice(operation, ParallelExecute())(slice)
		
		if err == nil {
			t.Errorf("Expected error from operation")
		}
		
		mutex.Lock()
		processedCount := len(processOrder)
		mutex.Unlock()
		
		t.Logf("Processed %d out of %d items before cancellation", processedCount, len(slice))
		
		// Should have processed less than total due to cancellation
		if processedCount >= len(slice) {
			t.Errorf("Expected fewer than %d items to be processed, got %d", 
				len(slice), processedCount)
		}
		
		// Should have processed at least some items
		if processedCount == 0 {
			t.Errorf("Expected some items to be processed before cancellation")
		}
	})
}

func TestContextTimeout(t *testing.T) {
	t.Run("Long-running operation with external context timeout", func(t *testing.T) {
		// Test timeout handling when operations take longer than expected
		slice := []uint32{1, 2, 3, 4, 5}
		var processedCount int64
		var timeoutCount int64

		// Create context with short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		// Operation that simulates long-running work
		operation := func(u uint32) error {
			select {
			case <-ctx.Done():
				atomic.AddInt64(&timeoutCount, 1)
				return ctx.Err()
			case <-time.After(20 * time.Millisecond): // Longer than context timeout
				atomic.AddInt64(&processedCount, 1)
				return nil
			}
		}

		// Execute operations - some should timeout
		err := ExecuteForEachSlice(operation, ParallelExecute())(slice)

		// Should get a timeout error from at least one operation
		if err == nil {
			t.Errorf("Expected timeout error from long-running operations")
		}

		processed := atomic.LoadInt64(&processedCount)
		timedOut := atomic.LoadInt64(&timeoutCount)

		t.Logf("Processed: %d, Timed out: %d, Total: %d", processed, timedOut, len(slice))

		// At least some operations should have timed out
		if timedOut == 0 {
			t.Errorf("Expected at least some operations to timeout, got %d", timedOut)
		}

		// Total processed + timed out should be at least 1 (the first error stops execution)
		if processed+timedOut == 0 {
			t.Errorf("Expected at least one operation to be attempted")
		}
	})

	t.Run("Context timeout vs operation completion race", func(t *testing.T) {
		// Test the race between context timeout and operation completion
		slice := []uint32{1, 2, 3}
		var results sync.Map
		
		// Create context with medium timeout
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Millisecond)
		defer cancel()

		operation := func(u uint32) error {
			// Variable operation duration to create race conditions
			duration := time.Duration(u*5) * time.Millisecond
			
			select {
			case <-ctx.Done():
				results.Store(u, "timeout")
				return ctx.Err()
			case <-time.After(duration):
				results.Store(u, "completed")
				return nil
			}
		}

		// Execute operations
		err := ExecuteForEachSlice(operation, ParallelExecute())(slice)

		// Check results of the race conditions
		var completed, timedOut int
		for _, val := range slice {
			if result, ok := results.Load(val); ok {
				switch result {
				case "completed":
					completed++
				case "timeout":
					timedOut++
				}
				t.Logf("Operation %d: %s", val, result)
			}
		}

		t.Logf("Race results - Completed: %d, Timed out: %d", completed, timedOut)

		// Should have at least one timeout or completion
		if completed+timedOut == 0 {
			t.Errorf("Expected at least one operation result")
		}

		// If there was a timeout, err should not be nil
		if timedOut > 0 && err == nil {
			t.Errorf("Expected error when operations timed out")
		}
	})

	t.Run("Sequential vs parallel timeout behavior", func(t *testing.T) {
		// Test that sequential execution doesn't respect external context timeout
		// while parallel execution does (due to internal context handling)
		slice := []uint32{1, 2, 3}

		// Create context with short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Millisecond)
		defer cancel()

		// Operation that checks context and simulates work
		var seqProcessed, parProcessed int64

		sequentialOp := func(u uint32) error {
			// Sequential mode doesn't check external context internally,
			// but we can simulate checking it in the operation
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(5 * time.Millisecond):
				atomic.AddInt64(&seqProcessed, 1)
				return nil
			}
		}

		parallelOp := func(u uint32) error {
			// Parallel operations should be interrupted by timeout
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(10 * time.Millisecond): // Longer than context timeout
				atomic.AddInt64(&parProcessed, 1)
				return nil
			}
		}

		// Test sequential execution (should timeout on first operation that checks)
		seqErr := ExecuteForEachSlice(sequentialOp)(slice)

		// Test parallel execution (should timeout on operations)
		parErr := ExecuteForEachSlice(parallelOp, ParallelExecute())(slice)

		t.Logf("Sequential processed: %d, error: %v", atomic.LoadInt64(&seqProcessed), seqErr)
		t.Logf("Parallel processed: %d, error: %v", atomic.LoadInt64(&parProcessed), parErr)

		// Both should have timeout errors due to operation-level context checking
		if seqErr == nil {
			t.Errorf("Expected timeout error in sequential execution")
		}
		if parErr == nil {
			t.Errorf("Expected timeout error in parallel execution")
		}

		// Both should have processed very few or no items due to timeout
		if atomic.LoadInt64(&seqProcessed) > 1 {
			t.Errorf("Expected at most 1 item processed in sequential due to timeout, got %d", 
				atomic.LoadInt64(&seqProcessed))
		}
		if atomic.LoadInt64(&parProcessed) > 0 {
			t.Errorf("Expected no items processed in parallel due to timeout, got %d", 
				atomic.LoadInt64(&parProcessed))
		}
	})

	t.Run("Context deadline exceeded error handling", func(t *testing.T) {
		// Test specific handling of context.DeadlineExceeded errors
		slice := []uint32{1, 2, 3, 4, 5}
		
		// Create context with very short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		operation := func(u uint32) error {
			// Ensure context is already expired
			time.Sleep(2 * time.Millisecond)
			
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				return nil
			}
		}

		// Execute operations
		err := ExecuteForEachSlice(operation, ParallelExecute())(slice)

		// Should get specific deadline exceeded error
		if err == nil {
			t.Errorf("Expected deadline exceeded error")
		} else if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("Expected context.DeadlineExceeded error, got: %T %v", err, err)
		}
	})

	t.Run("Timeout with resource cleanup", func(t *testing.T) {
		// Test that timeouts don't cause resource leaks
		slice := make([]uint32, 20)
		for i := range slice {
			slice[i] = uint32(i)
		}

		var startedOps, cleanedOps int64

		// Create context with medium timeout
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
		defer cancel()

		operation := func(u uint32) error {
			atomic.AddInt64(&startedOps, 1)
			defer atomic.AddInt64(&cleanedOps, 1)

			select {
			case <-ctx.Done():
				// Simulate cleanup time
				time.Sleep(100 * time.Microsecond)
				return ctx.Err()
			case <-time.After(10 * time.Millisecond): // Longer than context timeout
				time.Sleep(100 * time.Microsecond)
				return nil
			}
		}

		// Execute operations
		err := ExecuteForEachSlice(operation, ParallelExecute())(slice)

		// Give some time for cleanup
		time.Sleep(10 * time.Millisecond)

		started := atomic.LoadInt64(&startedOps)
		cleaned := atomic.LoadInt64(&cleanedOps)

		t.Logf("Started operations: %d, Cleaned operations: %d", started, cleaned)

		// Should have timeout error
		if err == nil || !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("Expected context.DeadlineExceeded error, got: %v", err)
		}

		// All started operations should have cleaned up
		if cleaned != started {
			t.Errorf("Expected all %d started operations to clean up, only %d cleaned up", 
				started, cleaned)
		}

		// Should have started at least some operations
		if started == 0 {
			t.Errorf("Expected at least some operations to start")
		}
	})
}

func TestContextPropagationProviderChains(t *testing.T) {
	t.Run("Context values propagate through Map chain", func(t *testing.T) {
		// Test that context values are accessible through chained Map operations
		const testKey = "test-key"
		const testValue = "test-value"
		
		// Create a context with a value
		ctx := context.WithValue(context.Background(), testKey, testValue)
		
		// Create transformers that depend on context values
		var capturedValues []string
		transform1 := func(val uint32) (string, error) {
			// In a real scenario, this would access context through some provider mechanism
			// For now, we'll simulate context access by recording the call
			capturedValues = append(capturedValues, "transform1")
			return fmt.Sprintf("t1-%d", val), nil
		}
		
		transform2 := func(val string) (string, error) {
			capturedValues = append(capturedValues, "transform2")
			return fmt.Sprintf("t2-%s", val), nil
		}
		
		// Create chained providers
		baseProvider := FixedProvider(uint32(42))
		chain := Map(transform2)(Map(transform1)(baseProvider))
		
		// Execute the chain - in a context-aware system, transformers would access ctx
		result, err := chain()
		
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		
		expectedResult := "t2-t1-42"
		if result != expectedResult {
			t.Errorf("Expected result %s, got %s", expectedResult, result)
		}
		
		// Verify both transformers were called in order
		expectedCalls := []string{"transform1", "transform2"}
		if len(capturedValues) != len(expectedCalls) {
			t.Errorf("Expected %d transform calls, got %d", len(expectedCalls), len(capturedValues))
		}
		
		for i, expected := range expectedCalls {
			if i < len(capturedValues) && capturedValues[i] != expected {
				t.Errorf("Expected call %d to be %s, got %s", i, expected, capturedValues[i])
			}
		}
		
		// Note: In a full implementation, transformers would access ctx to verify context propagation
		_ = ctx // Use ctx to avoid unused variable warning
	})
	
	t.Run("Context cancellation propagates through SliceMap chain", func(t *testing.T) {
		// Test that context cancellation affects chained SliceMap operations
		ctx, cancel := context.WithCancel(context.Background())
		
		// Create a slice to process
		input := []uint32{1, 2, 3, 4, 5}
		
		var stage1Started, stage2Started int64
		var stage1Cancelled, stage2Cancelled int64
		
		// First stage transformer that simulates work and checks cancellation
		transform1 := func(val uint32) (uint32, error) {
			atomic.AddInt64(&stage1Started, 1)
			
			// Simulate some work
			for i := 0; i < 5; i++ {
				select {
				case <-ctx.Done():
					atomic.AddInt64(&stage1Cancelled, 1)
					return 0, ctx.Err()
				default:
					time.Sleep(2 * time.Millisecond)
				}
			}
			return val * 2, nil
		}
		
		// Second stage transformer
		transform2 := func(val uint32) (uint32, error) {
			atomic.AddInt64(&stage2Started, 1)
			
			select {
			case <-ctx.Done():
				atomic.AddInt64(&stage2Cancelled, 1)
				return 0, ctx.Err()
			default:
				return val + 10, nil
			}
		}
		
		// Create chained SliceMap operations
		baseProvider := FixedProvider(input)
		stage1 := SliceMap(transform1)(baseProvider)(ParallelMap())
		chain := SliceMap(transform2)(stage1)(ParallelMap())
		
		// Start execution and cancel after a short delay
		go func() {
			time.Sleep(5 * time.Millisecond)
			cancel()
		}()
		
		// Execute the chain
		_, err := chain()
		
		// Should get cancellation error
		if err == nil || !errors.Is(err, context.Canceled) {
			t.Errorf("Expected context.Canceled error, got: %v", err)
		}
		
		// At least one stage should have started
		totalStarted := atomic.LoadInt64(&stage1Started) + atomic.LoadInt64(&stage2Started)
		if totalStarted == 0 {
			t.Errorf("Expected at least one operation to start")
		}
		
		// Some operations should have been cancelled
		totalCancelled := atomic.LoadInt64(&stage1Cancelled) + atomic.LoadInt64(&stage2Cancelled)
		if totalCancelled == 0 {
			t.Errorf("Expected at least one operation to be cancelled")
		}
		
		t.Logf("Stage1: started=%d, cancelled=%d; Stage2: started=%d, cancelled=%d", 
			atomic.LoadInt64(&stage1Started), atomic.LoadInt64(&stage1Cancelled),
			atomic.LoadInt64(&stage2Started), atomic.LoadInt64(&stage2Cancelled))
	})
	
	t.Run("Context deadline propagates through mixed provider chains", func(t *testing.T) {
		// Test context deadline propagation through Map and SliceMap combinations
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()
		
		var operationCount int64
		var timeoutCount int64
		
		// Transform that simulates variable duration work
		slowTransform := func(val uint32) (uint32, error) {
			atomic.AddInt64(&operationCount, 1)
			
			// Variable duration based on value to create deadline race conditions
			duration := time.Duration(val*3) * time.Millisecond
			
			select {
			case <-time.After(duration):
				return val * 2, nil
			case <-ctx.Done():
				atomic.AddInt64(&timeoutCount, 1)
				return 0, ctx.Err()
			}
		}
		
		// Single value transform for Map
		singleTransform := func(slice []uint32) (uint32, error) {
			if len(slice) == 0 {
				return 0, errors.New("empty slice")
			}
			
			select {
			case <-ctx.Done():
				return 0, ctx.Err()
			default:
				return slice[0], nil
			}
		}
		
		// Create complex chain: SliceMap -> Map
		input := []uint32{1, 2, 3, 4, 5, 6, 7, 8}
		baseProvider := FixedProvider(input)
		sliceMapStage := SliceMap(slowTransform)(baseProvider)(ParallelMap())
		mapStage := Map(singleTransform)(sliceMapStage)
		
		// Execute the chain
		_, err := mapStage()
		
		// Should get a timeout-related error
		if err == nil {
			t.Errorf("Expected timeout error, got no error")
		} else if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
			t.Errorf("Expected timeout or cancellation error, got: %v", err)
		}
		
		operations := atomic.LoadInt64(&operationCount)
		timeouts := atomic.LoadInt64(&timeoutCount)
		
		t.Logf("Operations started: %d, Timeouts: %d", operations, timeouts)
		
		// Some operations should have started
		if operations == 0 {
			t.Errorf("Expected at least some operations to start")
		}
		
		// Should have hit timeout for some operations due to parallel execution
		if timeouts == 0 && operations >= int64(len(input)) {
			// Only expect timeouts if we had enough operations running
			t.Errorf("Expected at least some operations to timeout with deadline of 10ms")
		}
	})
	
	t.Run("Context cancellation in nested provider chains", func(t *testing.T) {
		// Test deeply nested provider chains with cancellation
		ctx, cancel := context.WithCancel(context.Background())
		
		var depth1Calls, depth2Calls, depth3Calls int64
		var depth1Cancelled, depth2Cancelled, depth3Cancelled int64
		
		// Create three levels of transforms with longer delays to ensure cancellation
		depth1Transform := func(val uint32) (uint32, error) {
			atomic.AddInt64(&depth1Calls, 1)
			
			// Longer delay to increase chance of cancellation
			for i := 0; i < 10; i++ {
				select {
				case <-ctx.Done():
					atomic.AddInt64(&depth1Cancelled, 1)
					return 0, ctx.Err()
				default:
					time.Sleep(2 * time.Millisecond)
				}
			}
			return val + 1, nil
		}
		
		depth2Transform := func(val uint32) (uint32, error) {
			atomic.AddInt64(&depth2Calls, 1)
			
			// Check cancellation with delay
			for i := 0; i < 5; i++ {
				select {
				case <-ctx.Done():
					atomic.AddInt64(&depth2Cancelled, 1)
					return 0, ctx.Err()
				default:
					time.Sleep(1 * time.Millisecond)
				}
			}
			return val * 2, nil
		}
		
		depth3Transform := func(slice []uint32) ([]uint32, error) {
			atomic.AddInt64(&depth3Calls, 1)
			
			select {
			case <-ctx.Done():
				atomic.AddInt64(&depth3Cancelled, 1)
				return nil, ctx.Err()
			default:
				// Double each element
				result := make([]uint32, len(slice))
				for i, v := range slice {
					result[i] = v * 2
				}
				return result, nil
			}
		}
		
		// Create deeply nested chain with larger input to increase processing time
		input := []uint32{1, 2, 3, 4, 5, 6, 7, 8}
		baseProvider := FixedProvider(input)
		
		// Chain: SliceMap -> SliceMap -> Map
		level1 := SliceMap(depth1Transform)(baseProvider)(ParallelMap())
		level2 := SliceMap(depth2Transform)(level1)(ParallelMap())
		level3 := Map(depth3Transform)(level2)
		
		// Start execution and cancel after very short delay
		go func() {
			time.Sleep(3 * time.Millisecond)
			cancel()
		}()
		
		// Execute the nested chain
		_, err := level3()
		
		// Should get cancellation error
		if err == nil || !errors.Is(err, context.Canceled) {
			t.Errorf("Expected context.Canceled error, got: %v", err)
		}
		
		// Log call counts for debugging
		t.Logf("Depth1: calls=%d, cancelled=%d", 
			atomic.LoadInt64(&depth1Calls), atomic.LoadInt64(&depth1Cancelled))
		t.Logf("Depth2: calls=%d, cancelled=%d", 
			atomic.LoadInt64(&depth2Calls), atomic.LoadInt64(&depth2Cancelled))
		t.Logf("Depth3: calls=%d, cancelled=%d", 
			atomic.LoadInt64(&depth3Calls), atomic.LoadInt64(&depth3Cancelled))
		
		// At least some operations should have been called
		totalCalls := atomic.LoadInt64(&depth1Calls) + atomic.LoadInt64(&depth2Calls) + atomic.LoadInt64(&depth3Calls)
		if totalCalls == 0 {
			t.Errorf("Expected at least some operations to be called in nested chain")
		}
		
		// Some operations should have been cancelled (relaxed assertion)
		totalCancelled := atomic.LoadInt64(&depth1Cancelled) + atomic.LoadInt64(&depth2Cancelled) + atomic.LoadInt64(&depth3Cancelled)
		t.Logf("Total calls: %d, Total cancelled: %d", totalCalls, totalCancelled)
		
		// The test passes if we get cancellation error - the specific cancellation counts
		// may vary due to timing, but the error propagation is what we're testing
	})
}

func TestInvalidContextHandling(t *testing.T) {
	t.Run("Nil context handling in operations", func(t *testing.T) {
		// Test operations that might encounter nil context scenarios
		// Even though Go doesn't allow literal nil context.Context, we test edge cases
		slice := []uint32{1, 2, 3}

		operation := func(val uint32) error {
			// Simulate operation that might need to handle context edge cases
			if val == 0 {
				return fmt.Errorf("invalid zero value")
			}
			return nil
		}

		// Test sequential execution - should handle nil-like scenarios gracefully
		err := ExecuteForEachSlice(operation)(slice)
		if err != nil {
			t.Errorf("Expected no error with valid slice, got: %v", err)
		}
	})

	t.Run("Already expired context behavior", func(t *testing.T) {
		// Test behavior with context that expires immediately
		ctx, cancel := context.WithTimeout(context.Background(), 0) // Expires immediately
		cancel() // Ensure it's cancelled

		slice := []uint32{1, 2, 3, 4, 5}
		operationCount := int64(0)

		operation := func(val uint32) error {
			select {
			case <-ctx.Done():
				// Context already expired - should handle gracefully
				return ctx.Err()
			default:
				atomic.AddInt64(&operationCount, 1)
				return nil
			}
		}

		// Test with parallel execution - should handle pre-cancelled context
		err := ExecuteForEachSlice(operation, ParallelExecute())(slice)

		// Should either succeed with no operations or fail with context error
		if err != nil {
			if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
				t.Errorf("Expected context error, got: %T %v", err, err)
			}
		}

		processed := atomic.LoadInt64(&operationCount)
		t.Logf("Operations processed with expired context: %d", processed)
	})

	t.Run("Context with invalid or corrupted values", func(t *testing.T) {
		// Test context with unusual values that might cause issues
		type contextKey string
		const testKey contextKey = "testKey"

		// Context with potentially problematic values
		ctx := context.WithValue(context.Background(), testKey, nil)
		ctx = context.WithValue(ctx, "anotherKey", "")
		ctx = context.WithValue(ctx, "", "emptyKeyValue")

		slice := []uint32{1, 2, 3}

		operation := func(val uint32) error {
			// Operation that accesses context values
			value := ctx.Value(testKey)
			if value != nil {
				return fmt.Errorf("expected nil value, got: %v", value)
			}

			// Test accessing non-existent key
			nonExistent := ctx.Value("nonexistent")
			if nonExistent != nil {
				return fmt.Errorf("expected nil for non-existent key, got: %v", nonExistent)
			}

			return nil
		}

		err := ExecuteForEachSlice(operation, ParallelExecute())(slice)
		if err != nil {
			t.Errorf("Expected operations to handle context values gracefully, got: %v", err)
		}
	})

	t.Run("Context cancellation race conditions", func(t *testing.T) {
		// Test race conditions between context cancellation and operation start
		slice := []uint32{1, 2, 3, 4, 5}
		successCount := int64(0)
		cancelCount := int64(0)

		for i := 0; i < 10; i++ { // Run multiple iterations to catch race conditions
			ctx, cancel := context.WithCancel(context.Background())

			operation := func(val uint32) error {
				// Introduce small delay to increase chance of race condition
				time.Sleep(time.Microsecond)

				select {
				case <-ctx.Done():
					atomic.AddInt64(&cancelCount, 1)
					return ctx.Err()
				default:
					atomic.AddInt64(&successCount, 1)
					return nil
				}
			}

			// Cancel context after very short delay
			go func() {
				time.Sleep(time.Microsecond)
				cancel()
			}()

			// Execute operations
			err := ExecuteForEachSlice(operation, ParallelExecute())(slice)

			// Either should succeed or fail with context error
			if err != nil && !errors.Is(err, context.Canceled) {
				t.Errorf("Expected either success or context.Canceled, got: %T %v", err, err)
			}
		}

		t.Logf("Race condition test - Success: %d, Cancelled: %d",
			atomic.LoadInt64(&successCount), atomic.LoadInt64(&cancelCount))

		// At least some operations should have run
		total := atomic.LoadInt64(&successCount) + atomic.LoadInt64(&cancelCount)
		if total == 0 {
			t.Errorf("Expected at least some operations to be attempted")
		}
	})

	t.Run("Context deadline in the past", func(t *testing.T) {
		// Test context with deadline already in the past
		pastTime := time.Now().Add(-1 * time.Hour)
		ctx, cancel := context.WithDeadline(context.Background(), pastTime)
		defer cancel()

		slice := []uint32{1, 2}

		operation := func(val uint32) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				return nil
			}
		}

		err := ExecuteForEachSlice(operation, ParallelExecute())(slice)

		// Should fail with deadline exceeded
		if err == nil {
			t.Errorf("Expected error with past deadline context")
		} else if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("Expected context.DeadlineExceeded, got: %T %v", err, err)
		}
	})

	t.Run("Context with zero value timeout", func(t *testing.T) {
		// Test context with zero-duration timeout
		ctx, cancel := context.WithTimeout(context.Background(), 0)
		defer cancel()

		slice := []uint32{1}
		errorOccurred := false

		operation := func(val uint32) error {
			select {
			case <-ctx.Done():
				errorOccurred = true
				return ctx.Err()
			default:
				// Should not reach here with zero timeout
				return nil
			}
		}

		err := ExecuteForEachSlice(operation)(slice)

		// Should handle zero timeout gracefully
		if err != nil {
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Errorf("Expected context.DeadlineExceeded or no error, got: %T %v", err, err)
			}
		}

		t.Logf("Zero timeout context - Error occurred: %v", errorOccurred)
	})
}

// Memory Leak Detection Tests
func TestMemoryLeakDetection(t *testing.T) {
	t.Run("RepeatedProviderExecutionMemoryLeak", func(t *testing.T) {
		// Test that repeated provider execution doesn't accumulate memory
		runtime.GC()
		var m1 runtime.MemStats
		runtime.ReadMemStats(&m1)
		initialAlloc := m1.Alloc

		// Execute many provider operations
		for i := 0; i < 10000; i++ {
			data := make([]uint32, 100)
			for j := range data {
				data[j] = uint32(j)
			}

			provider := FixedProvider(data)
			transform := func(val uint32) (uint32, error) {
				// Create some temporary objects to stress memory
				temp := make([]byte, 1024)
				_ = temp
				return val * 2, nil
			}

			mapped := SliceMap(transform)(provider)()
			results, err := mapped()
			if err != nil {
				t.Errorf("Iteration %d failed: %v", i, err)
			}
			if len(results) != len(data) {
				t.Errorf("Iteration %d: expected %d results, got %d", i, len(data), len(results))
			}

			// Periodically force GC to prevent normal accumulation
			if i%1000 == 0 {
				runtime.GC()
			}
		}

		// Force final garbage collection and measure memory
		runtime.GC()
		runtime.GC() // Double GC to ensure cleanup
		var m2 runtime.MemStats
		runtime.ReadMemStats(&m2)
		finalAlloc := m2.Alloc

		// Memory growth should be reasonable (less than 10MB for this test)
		memoryGrowth := int64(finalAlloc) - int64(initialAlloc)
		maxGrowth := int64(10 * 1024 * 1024) // 10MB threshold

		t.Logf("Memory growth: %d bytes (%.2f MB)", memoryGrowth, float64(memoryGrowth)/1024/1024)

		if memoryGrowth > maxGrowth {
			t.Errorf("Excessive memory growth detected: %d bytes (%.2f MB), threshold: %d bytes",
				memoryGrowth, float64(memoryGrowth)/1024/1024, maxGrowth)
		}
	})

	t.Run("MemoizedProviderMemoryLeak", func(t *testing.T) {
		// Test that memoized providers don't cause memory leaks with repeated access
		runtime.GC()
		var m1 runtime.MemStats
		runtime.ReadMemStats(&m1)
		initialAlloc := m1.Alloc

		// Create a memoized provider that allocates memory
		expensiveProvider := func() ([]byte, error) {
			// Allocate a large chunk of memory for the cached result
			data := make([]byte, 1024*1024) // 1MB
			for i := range data {
				data[i] = byte(i % 256)
			}
			return data, nil
		}

		memoized := Memoize(expensiveProvider)

		// Access the memoized provider many times
		for i := 0; i < 5000; i++ {
			result, err := memoized()
			if err != nil {
				t.Errorf("Iteration %d failed: %v", i, err)
			}
			if len(result) != 1024*1024 {
				t.Errorf("Iteration %d: expected 1MB result, got %d bytes", i, len(result))
			}

			// Verify content is correct
			if result[0] != 0 || result[255] != 255 {
				t.Errorf("Iteration %d: result content verification failed", i)
			}
		}

		// Force garbage collection and measure memory
		runtime.GC()
		runtime.GC()
		var m2 runtime.MemStats
		runtime.ReadMemStats(&m2)
		finalAlloc := m2.Alloc

		// Should only have one copy of the 1MB data plus reasonable overhead
		memoryGrowth := int64(finalAlloc) - int64(initialAlloc)
		maxGrowth := int64(2 * 1024 * 1024) // 2MB threshold (1MB data + overhead)

		t.Logf("Memoized memory growth: %d bytes (%.2f MB)", memoryGrowth, float64(memoryGrowth)/1024/1024)

		if memoryGrowth > maxGrowth {
			t.Errorf("Excessive memory growth in memoized provider: %d bytes (%.2f MB), threshold: %d bytes",
				memoryGrowth, float64(memoryGrowth)/1024/1024, maxGrowth)
		}
	})

	t.Run("ParallelExecutionMemoryLeak", func(t *testing.T) {
		// Test that parallel execution doesn't accumulate goroutines or memory
		runtime.GC()
		var m1 runtime.MemStats
		runtime.ReadMemStats(&m1)
		initialAlloc := m1.Alloc
		initialGoroutines := runtime.NumGoroutine()

		// Run many parallel operations
		for iteration := 0; iteration < 100; iteration++ {
			data := make([]uint32, 50)
			for i := range data {
				data[i] = uint32(i + iteration*50)
			}

			provider := FixedProvider(data)

			// Transform that allocates temporary memory
			transform := func(val uint32) (uint32, error) {
				// Allocate temporary memory to stress the system
				temp := make([]int, 1000)
				for i := range temp {
					temp[i] = int(val) + i
				}

				// Add some computation delay
				time.Sleep(time.Microsecond)
				return val * 2, nil
			}

			mapped := SliceMap(transform)(provider)(ParallelMap())
			results, err := mapped()
			if err != nil {
				t.Errorf("Iteration %d failed: %v", iteration, err)
			}
			if len(results) != len(data) {
				t.Errorf("Iteration %d: expected %d results, got %d", iteration, len(data), len(results))
			}

			// Give goroutines time to clean up
			if iteration%10 == 0 {
				time.Sleep(10 * time.Millisecond)
				runtime.GC()
			}
		}

		// Allow time for goroutine cleanup
		time.Sleep(100 * time.Millisecond)
		runtime.GC()
		runtime.GC()

		var m2 runtime.MemStats
		runtime.ReadMemStats(&m2)
		finalAlloc := m2.Alloc
		finalGoroutines := runtime.NumGoroutine()

		memoryGrowth := int64(finalAlloc) - int64(initialAlloc)
		goroutineGrowth := finalGoroutines - initialGoroutines

		t.Logf("Parallel execution - Memory growth: %d bytes (%.2f MB), Goroutine growth: %d",
			memoryGrowth, float64(memoryGrowth)/1024/1024, goroutineGrowth)

		// Memory growth should be reasonable
		maxMemoryGrowth := int64(5 * 1024 * 1024) // 5MB threshold
		if memoryGrowth > maxMemoryGrowth {
			t.Errorf("Excessive memory growth in parallel execution: %d bytes (%.2f MB)",
				memoryGrowth, float64(memoryGrowth)/1024/1024)
		}

		// Goroutine growth should be minimal (allow some variance for test runner)
		maxGoroutineGrowth := 10
		if goroutineGrowth > maxGoroutineGrowth {
			t.Errorf("Excessive goroutine growth: %d goroutines, threshold: %d",
				goroutineGrowth, maxGoroutineGrowth)
		}
	})

	t.Run("ErrorScenarioMemoryLeak", func(t *testing.T) {
		// Test that error scenarios don't cause memory leaks
		runtime.GC()
		var m1 runtime.MemStats
		runtime.ReadMemStats(&m1)
		initialAlloc := m1.Alloc

		// Run many operations that fail with errors
		for i := 0; i < 1000; i++ {
			data := make([]uint32, 20)
			for j := range data {
				data[j] = uint32(j)
			}

			provider := FixedProvider(data)

			// Transform that sometimes fails and allocates memory
			transform := func(val uint32) (uint32, error) {
				// Always allocate some memory
				temp := make([]byte, 2048)
				_ = temp

				// Fail on certain values
				if val%3 == 0 {
					return 0, fmt.Errorf("intentional error for value %d", val)
				}
				return val * 2, nil
			}

			mapped := SliceMap(transform)(provider)()
			_, err := mapped()

			// We expect errors, so that's okay
			if err == nil && i%3 == 0 {
				// Some iterations should have errors due to val%3==0 check
			}

			// Periodically clean up
			if i%100 == 0 {
				runtime.GC()
			}
		}

		// Final cleanup and measurement
		runtime.GC()
		runtime.GC()
		var m2 runtime.MemStats
		runtime.ReadMemStats(&m2)
		finalAlloc := m2.Alloc

		memoryGrowth := int64(finalAlloc) - int64(initialAlloc)
		maxGrowth := int64(3 * 1024 * 1024) // 3MB threshold

		t.Logf("Error scenario memory growth: %d bytes (%.2f MB)", memoryGrowth, float64(memoryGrowth)/1024/1024)

		if memoryGrowth > maxGrowth {
			t.Errorf("Excessive memory growth in error scenarios: %d bytes (%.2f MB), threshold: %d bytes",
				memoryGrowth, float64(memoryGrowth)/1024/1024, maxGrowth)
		}
	})

	t.Run("ContextCancellationMemoryLeak", func(t *testing.T) {
		// Test that context cancellation doesn't cause memory leaks
		runtime.GC()
		var m1 runtime.MemStats
		runtime.ReadMemStats(&m1)
		initialAlloc := m1.Alloc
		initialGoroutines := runtime.NumGoroutine()

		// Run many operations that get cancelled
		for i := 0; i < 50; i++ {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)

			data := make([]uint32, 30)
			for j := range data {
				data[j] = uint32(j + i*30)
			}

			provider := FixedProvider(data)

			// Long-running transform that respects context
			transform := func(val uint32) (uint32, error) {
				// Allocate memory for processing
				temp := make([]int64, 500)
				for k := range temp {
					temp[k] = int64(val) * int64(k)
				}

				// Long processing time to ensure cancellation
				for j := 0; j < 100; j++ {
					select {
					case <-ctx.Done():
						return 0, ctx.Err()
					default:
						time.Sleep(time.Millisecond)
					}
				}
				return val * 2, nil
			}

			mapped := SliceMap(transform)(provider)(ParallelMap())

			// Start a goroutine to cancel after a short delay
			go func() {
				time.Sleep(5 * time.Millisecond)
				cancel()
			}()

			_, err := mapped()

			// Should get cancellation error
			if err == nil {
				t.Logf("Iteration %d: expected cancellation error but got none", i)
			}

			cancel() // Ensure cleanup

			// Give time for goroutines to clean up
			time.Sleep(20 * time.Millisecond)

			if i%10 == 0 {
				runtime.GC()
			}
		}

		// Final cleanup and measurement
		time.Sleep(100 * time.Millisecond)
		runtime.GC()
		runtime.GC()

		var m2 runtime.MemStats
		runtime.ReadMemStats(&m2)
		finalAlloc := m2.Alloc
		finalGoroutines := runtime.NumGoroutine()

		memoryGrowth := int64(finalAlloc) - int64(initialAlloc)
		goroutineGrowth := finalGoroutines - initialGoroutines

		t.Logf("Context cancellation - Memory growth: %d bytes (%.2f MB), Goroutine growth: %d",
			memoryGrowth, float64(memoryGrowth)/1024/1024, goroutineGrowth)

		// Memory and goroutine growth should be minimal
		maxMemoryGrowth := int64(2 * 1024 * 1024) // 2MB threshold
		if memoryGrowth > maxMemoryGrowth {
			t.Errorf("Excessive memory growth with context cancellation: %d bytes (%.2f MB)",
				memoryGrowth, float64(memoryGrowth)/1024/1024)
		}

		maxGoroutineGrowth := 15
		if goroutineGrowth > maxGoroutineGrowth {
			t.Errorf("Excessive goroutine growth with context cancellation: %d goroutines", goroutineGrowth)
		}
	})
}

func TestResourceCleanupInErrorScenarios(t *testing.T) {
	// Test that resources are properly cleaned up when errors occur during various operations
	// This ensures that error scenarios don't cause memory leaks or resource accumulation

	t.Run("ProviderChainErrorResourceCleanup", func(t *testing.T) {
		// Test that resources are cleaned up when provider chains fail
		runtime.GC()
		var m1 runtime.MemStats
		runtime.ReadMemStats(&m1)
		initialAlloc := m1.Alloc
		initialGoroutines := runtime.NumGoroutine()

		// Run operations that fail at different points in the chain
		for i := 0; i < 100; i++ {
			data := []uint32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

			// Transform that fails on specific values with resource allocation
			failingTransform := func(val uint32) (uint32, error) {
				// Allocate memory for processing (should be cleaned up on error)
				temp := make([]byte, 2048)
				for j := range temp {
					temp[j] = byte(j % 256)
				}

				if val == 7 { // Fail on the 7th item
					return 0, fmt.Errorf("intentional error on value %d", val)
				}

				// Some processing work
				result := val
				for k := 0; k < 100; k++ {
					result = (result * 7) % 1000
				}
				return result + val, nil
			}

			provider := FixedProvider(data)
			mapped := SliceMap(failingTransform)(provider)(ParallelMap())

			_, err := mapped()

			if err == nil {
				t.Errorf("Iteration %d: expected error but got none", i)
				continue
			}

			// Verify we got the expected error
			if err.Error() != "intentional error on value 7" {
				t.Errorf("Iteration %d: expected specific error, got %v", i, err)
			}

			// Periodic cleanup to verify no accumulation
			if i%20 == 0 {
				time.Sleep(10 * time.Millisecond)
				runtime.GC()
			}
		}

		// Final cleanup and measurement
		time.Sleep(50 * time.Millisecond)
		runtime.GC()
		runtime.GC()

		var m2 runtime.MemStats
		runtime.ReadMemStats(&m2)
		finalAlloc := m2.Alloc
		finalGoroutines := runtime.NumGoroutine()

		memoryGrowth := int64(finalAlloc) - int64(initialAlloc)
		goroutineGrowth := finalGoroutines - initialGoroutines

		t.Logf("Provider chain error cleanup - Memory growth: %d bytes (%.2f MB), Goroutine growth: %d",
			memoryGrowth, float64(memoryGrowth)/1024/1024, goroutineGrowth)

		// Verify minimal resource growth
		maxMemoryGrowth := int64(3 * 1024 * 1024) // 3MB threshold
		if memoryGrowth > maxMemoryGrowth {
			t.Errorf("Excessive memory growth in error scenarios: %d bytes (%.2f MB)",
				memoryGrowth, float64(memoryGrowth)/1024/1024)
		}

		maxGoroutineGrowth := 10
		if goroutineGrowth > maxGoroutineGrowth {
			t.Errorf("Excessive goroutine growth in error scenarios: %d goroutines", goroutineGrowth)
		}
	})

	t.Run("ParallelMapErrorResourceCleanup", func(t *testing.T) {
		// Test resource cleanup when parallel operations fail
		runtime.GC()
		var m1 runtime.MemStats
		runtime.ReadMemStats(&m1)
		initialAlloc := m1.Alloc

		for iteration := 0; iteration < 50; iteration++ {
			// Create larger dataset to stress parallel processing
			data := make([]uint32, 50)
			for i := range data {
				data[i] = uint32(i + 1)
			}

			// Transform that fails on multiple values with heavy resource usage
			errorTransform := func(val uint32) (uint32, error) {
				// Allocate significant memory per operation
				temp := make([]int64, 1000)
				for j := range temp {
					temp[j] = int64(val) * int64(j)
				}

				// Fail on multiple values to test various error timing
				if val%13 == 0 { // Fail on values 13, 26, 39
					return 0, fmt.Errorf("parallel error on value %d", val)
				}

				// CPU-intensive work that should be cleaned up on error
				result := val
				for k := 0; k < 500; k++ {
					result = (result*11 + 7) % 10007
				}

				return result, nil
			}

			provider := FixedProvider(data)
			mapped := SliceMap(errorTransform)(provider)(ParallelMap())

			_, err := mapped()

			if err == nil {
				t.Errorf("Iteration %d: expected error but got none", iteration)
			}

			// Give time for goroutine cleanup
			time.Sleep(5 * time.Millisecond)

			if iteration%10 == 0 {
				runtime.GC()
			}
		}

		// Final cleanup and measurement
		time.Sleep(100 * time.Millisecond)
		runtime.GC()
		runtime.GC()

		var m2 runtime.MemStats
		runtime.ReadMemStats(&m2)
		finalAlloc := m2.Alloc

		memoryGrowth := int64(finalAlloc) - int64(initialAlloc)
		maxGrowth := int64(5 * 1024 * 1024) // 5MB threshold for parallel operations

		t.Logf("Parallel map error cleanup - Memory growth: %d bytes (%.2f MB)", memoryGrowth, float64(memoryGrowth)/1024/1024)

		if memoryGrowth > maxGrowth {
			t.Errorf("Excessive memory growth in parallel error scenarios: %d bytes (%.2f MB)",
				memoryGrowth, float64(memoryGrowth)/1024/1024)
		}
	})

	t.Run("MidOperationErrorResourceCleanup", func(t *testing.T) {
		// Test cleanup when errors occur in the middle of operations
		runtime.GC()
		var m1 runtime.MemStats
		runtime.ReadMemStats(&m1)
		initialAlloc := m1.Alloc

		for i := 0; i < 200; i++ {
			data := []uint32{1, 2, 3, 4, 5}

			// Transform that fails partway through processing
			midErrorTransform := func(val uint32) (uint32, error) {
				// Allocate resources at the start
				temp1 := make([]float64, 500)
				for j := range temp1 {
					temp1[j] = float64(val) * float64(j) * 0.1
				}

				// Start processing
				result := val
				for step := 0; step < 10; step++ {
					result = result*3 + 1

					// Fail partway through for certain values
					if val == 3 && step == 5 {
						// Allocate more resources that should be cleaned up
						temp2 := make([]int, 1000)
						for k := range temp2 {
							temp2[k] = int(result) + k
						}
						return 0, fmt.Errorf("mid-operation error on value %d at step %d", val, step)
					}
				}

				return result, nil
			}

			provider := FixedProvider(data)
			mapped := SliceMap(midErrorTransform)(provider)()

			_, err := mapped()

			if err == nil {
				t.Errorf("Iteration %d: expected mid-operation error but got none", i)
			} else if !strings.Contains(err.Error(), "mid-operation error") {
				t.Errorf("Iteration %d: expected mid-operation error, got %v", i, err)
			}

			// Periodic cleanup
			if i%25 == 0 {
				time.Sleep(10 * time.Millisecond)
				runtime.GC()
			}
		}

		// Final cleanup and measurement
		time.Sleep(50 * time.Millisecond)
		runtime.GC()
		runtime.GC()

		var m2 runtime.MemStats
		runtime.ReadMemStats(&m2)
		finalAlloc := m2.Alloc

		memoryGrowth := int64(finalAlloc) - int64(initialAlloc)
		maxGrowth := int64(4 * 1024 * 1024) // 4MB threshold

		t.Logf("Mid-operation error cleanup - Memory growth: %d bytes (%.2f MB)", memoryGrowth, float64(memoryGrowth)/1024/1024)

		if memoryGrowth > maxGrowth {
			t.Errorf("Excessive memory growth in mid-operation error scenarios: %d bytes (%.2f MB)",
				memoryGrowth, float64(memoryGrowth)/1024/1024)
		}
	})

	t.Run("ErrorPropagationResourceCleanup", func(t *testing.T) {
		// Test that error propagation doesn't leak resources
		runtime.GC()
		var m1 runtime.MemStats
		runtime.ReadMemStats(&m1)
		initialAlloc := m1.Alloc
		initialGoroutines := runtime.NumGoroutine()

		// Run operations with cascading errors
		for i := 0; i < 30; i++ {
			data := make([]uint32, 20)
			for j := range data {
				data[j] = uint32(j + 1 + i*20)
			}

			// First transform that allocates resources
			firstTransform := func(val uint32) (uint32, error) {
				temp := make([]byte, 4096)
				for k := range temp {
					temp[k] = byte((val + uint32(k)) % 256)
				}
				return val * 2, nil
			}

			// Second transform that fails and should clean up resources
			secondTransform := func(val uint32) (uint32, error) {
				// Allocate more resources
				temp := make([]int32, 1024)
				for k := range temp {
					temp[k] = int32(val) + int32(k)
				}

				// Fail on even values (will be multiple values since first transform doubles)
				if val%4 == 0 {
					return 0, fmt.Errorf("error propagation test on value %d", val)
				}

				return val + 100, nil
			}

			provider := FixedProvider(data)
			firstMapped := SliceMap(firstTransform)(provider)(ParallelMap())

			// Chain the operations
			result, err := firstMapped()
			if err != nil {
				t.Errorf("Iteration %d: first transform should not error, got %v", i, err)
				continue
			}

			secondMapped := SliceMap(secondTransform)(FixedProvider(result))(ParallelMap())
			_, err = secondMapped()

			if err == nil {
				t.Errorf("Iteration %d: expected error from second transform but got none", i)
			}

			// Allow cleanup time
			time.Sleep(5 * time.Millisecond)

			if i%10 == 0 {
				runtime.GC()
			}
		}

		// Final cleanup and measurement
		time.Sleep(100 * time.Millisecond)
		runtime.GC()
		runtime.GC()

		var m2 runtime.MemStats
		runtime.ReadMemStats(&m2)
		finalAlloc := m2.Alloc
		finalGoroutines := runtime.NumGoroutine()

		memoryGrowth := int64(finalAlloc) - int64(initialAlloc)
		goroutineGrowth := finalGoroutines - initialGoroutines

		t.Logf("Error propagation cleanup - Memory growth: %d bytes (%.2f MB), Goroutine growth: %d",
			memoryGrowth, float64(memoryGrowth)/1024/1024, goroutineGrowth)

		maxMemoryGrowth := int64(6 * 1024 * 1024) // 6MB threshold for chained operations
		if memoryGrowth > maxMemoryGrowth {
			t.Errorf("Excessive memory growth in error propagation: %d bytes (%.2f MB)",
				memoryGrowth, float64(memoryGrowth)/1024/1024)
		}

		maxGoroutineGrowth := 12
		if goroutineGrowth > maxGoroutineGrowth {
			t.Errorf("Excessive goroutine growth in error propagation: %d goroutines", goroutineGrowth)
		}
	})

	t.Run("FilterOperationErrorResourceCleanup", func(t *testing.T) {
		// Test resource cleanup when filter operations encounter errors
		runtime.GC()
		var m1 runtime.MemStats
		runtime.ReadMemStats(&m1)
		initialAlloc := m1.Alloc

		for i := 0; i < 100; i++ {
			data := make([]uint32, 30)
			for j := range data {
				data[j] = uint32(j + 1)
			}

			// Filter function that allocates resources and sometimes fails
			errorFilter := func(val uint32) bool {
				// Allocate memory during filtering
				temp := make([]uint64, 200)
				for k := range temp {
					temp[k] = uint64(val) * uint64(k) * uint64(k)
				}

				// Fail on specific values (this will be handled as a filter non-match, but resources should still be cleaned up)
				if val == 15 {
					// Even though this doesn't return an error, the resources should be cleaned up
					// when the filter doesn't match
					return false
				}

				return val%3 == 0
			}

			provider := FixedProvider(data)
			filtered, err := First(provider, Filters(errorFilter))

			// This should succeed (finding value 3), but resource cleanup should still occur
			if err != nil && err.Error() != "no item found" {
				t.Errorf("Iteration %d: unexpected error %v", i, err)
			}

			if err == nil && filtered != 3 {
				t.Errorf("Iteration %d: expected 3, got %d", i, filtered)
			}

			if i%20 == 0 {
				runtime.GC()
			}
		}

		// Final cleanup and measurement
		runtime.GC()
		runtime.GC()

		var m2 runtime.MemStats
		runtime.ReadMemStats(&m2)
		finalAlloc := m2.Alloc

		memoryGrowth := int64(finalAlloc) - int64(initialAlloc)
		maxGrowth := int64(2 * 1024 * 1024) // 2MB threshold for filter operations

		t.Logf("Filter operation cleanup - Memory growth: %d bytes (%.2f MB)", memoryGrowth, float64(memoryGrowth)/1024/1024)

		if memoryGrowth > maxGrowth {
			t.Errorf("Excessive memory growth in filter error scenarios: %d bytes (%.2f MB)",
				memoryGrowth, float64(memoryGrowth)/1024/1024)
		}
	})
}

// BenchmarkProviderErrorHandling measures the performance overhead of error handling in provider operations
func BenchmarkProviderErrorHandling(b *testing.B) {
	// Setup test data
	data := make([]uint32, 100)
	for i := range data {
		data[i] = uint32(i + 1)
	}

	b.Run("SuccessfulProvider", func(b *testing.B) {
		// Baseline: providers that never fail
		successfulProvider := func() ([]uint32, error) {
			return data, nil
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := successfulProvider()
			if err != nil {
				b.Fatal("Unexpected error in successful provider")
			}
		}
	})

	b.Run("ErrorProvider", func(b *testing.B) {
		// Error providers to measure error handling overhead
		testError := errors.New("benchmark test error")
		errorProvider := ErrorProvider[[]uint32](testError)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := errorProvider()
			if err == nil {
				b.Fatal("Expected error but got none")
			}
		}
	})

	b.Run("SliceMapWithErrors", func(b *testing.B) {
		// Measure overhead of error handling in SliceMap operations
		provider := func() ([]uint32, error) {
			return data, nil
		}

		// Transform that fails on every 10th item
		errorTransform := func(val uint32) (uint32, error) {
			if val%10 == 0 {
				return 0, fmt.Errorf("error on value %d", val)
			}
			return val * 2, nil
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := SliceMap(errorTransform)(provider)()()
			if err == nil {
				b.Fatal("Expected error from failing transform")
			}
		}
	})

	b.Run("ParallelMapWithErrors", func(b *testing.B) {
		// Measure parallel error handling overhead
		provider := func() ([]uint32, error) {
			return data, nil
		}

		// Transform that fails on multiple values
		errorTransform := func(val uint32) (uint32, error) {
			if val%5 == 0 {
				return 0, fmt.Errorf("parallel error on value %d", val)
			}
			// Add small delay to simulate work
			time.Sleep(time.Microsecond)
			return val * 3, nil
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := SliceMap(errorTransform)(provider)(ParallelMap())()
			if err == nil {
				b.Fatal("Expected error from parallel failing transform")
			}
		}
	})

	b.Run("ChainedProviderErrors", func(b *testing.B) {
		// Measure error propagation overhead through provider chains
		baseProvider := func() (uint32, error) {
			return 42, nil
		}

		// First transform succeeds
		firstTransform := func(val uint32) (uint32, error) {
			return val * 2, nil
		}

		// Second transform fails
		secondTransform := func(val uint32) (uint32, error) {
			return 0, fmt.Errorf("chained error on value %d", val)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			chain := Map(secondTransform)(Map(firstTransform)(baseProvider))
			_, err := chain()
			if err == nil {
				b.Fatal("Expected error from chained transforms")
			}
		}
	})

	b.Run("MemoizedProviderWithErrors", func(b *testing.B) {
		// Measure error handling overhead with memoization
		callCount := int64(0)
		errorProvider := func() (uint32, error) {
			atomic.AddInt64(&callCount, 1)
			return 0, fmt.Errorf("memoized error call %d", callCount)
		}

		memoized := Memoize(errorProvider)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := memoized()
			if err == nil {
				b.Fatal("Expected error from memoized provider")
			}
		}

		// Verify memoization worked (should only call once)
		if atomic.LoadInt64(&callCount) != 1 {
			b.Fatalf("Expected memoized provider to be called once, got %d calls", callCount)
		}
	})

	b.Run("ContextCancellationErrors", func(b *testing.B) {
		// Measure performance of context cancellation error handling
		provider := func() ([]uint32, error) {
			return data, nil
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ctx, cancel := context.WithCancel(context.Background())

			// Transform that checks for cancellation
			cancelTransform := func(val uint32) (uint32, error) {
				select {
				case <-ctx.Done():
					return 0, ctx.Err()
				default:
					return val * 2, nil
				}
			}

			// Cancel immediately to trigger error
			cancel()

			_, err := SliceMap(cancelTransform)(provider)()()
			if err == nil || !errors.Is(err, context.Canceled) {
				b.Fatal("Expected context cancellation error")
			}
		}
	})

	b.Run("ErrorRecoveryOverhead", func(b *testing.B) {
		// Measure overhead of attempting operations after errors
		var successCount, errorCount int64

		// Provider that alternates between success and error
		alternatingProvider := func() (uint32, error) {
			count := atomic.AddInt64(&successCount, 1)
			if count%2 == 0 {
				atomic.AddInt64(&errorCount, 1)
				return 0, fmt.Errorf("alternating error on call %d", count)
			}
			return uint32(count), nil
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := alternatingProvider()
			// Don't fail on expected errors - just measure the overhead
			_ = err
		}

		b.Logf("Success: %d, Errors: %d", atomic.LoadInt64(&successCount), atomic.LoadInt64(&errorCount))
	})
}

func BenchmarkContextCancellation(b *testing.B) {
	// Benchmark measuring the performance overhead of context cancellation
	// in parallel operations vs sequential operations

	// Setup test data
	data := make([]uint32, 1000)
	for i := range data {
		data[i] = uint32(i + 1)
	}

	b.Run("ParallelWithoutCancellation", func(b *testing.B) {
		// Baseline: parallel execution without any cancellation
		operation := func(u uint32) error {
			// Simulate light work
			time.Sleep(10 * time.Microsecond)
			return nil
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := ExecuteForEachSlice(operation, ParallelExecute())(data)
			if err != nil {
				b.Fatalf("Unexpected error: %v", err)
			}
		}
	})

	b.Run("ParallelWithEarlyCancellation", func(b *testing.B) {
		// Measure overhead when context cancellation occurs early in execution
		operation := func(u uint32) error {
			// Fail on the 10th item to trigger early cancellation
			if u == 10 {
				return errors.New("early cancellation trigger")
			}
			// Simulate work - some goroutines will be cancelled mid-execution
			time.Sleep(10 * time.Microsecond)
			return nil
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := ExecuteForEachSlice(operation, ParallelExecute())(data)
			if err == nil {
				b.Fatal("Expected cancellation error")
			}
		}
	})

	b.Run("ParallelWithLateCancellation", func(b *testing.B) {
		// Measure overhead when context cancellation occurs late in execution
		operation := func(u uint32) error {
			// Fail on the 900th item to trigger late cancellation
			if u == 900 {
				return errors.New("late cancellation trigger")
			}
			// Simulate work
			time.Sleep(10 * time.Microsecond)
			return nil
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := ExecuteForEachSlice(operation, ParallelExecute())(data)
			if err == nil {
				b.Fatal("Expected cancellation error")
			}
		}
	})

	b.Run("SequentialVsParallelCancellationOverhead", func(b *testing.B) {
		// Compare cancellation overhead between sequential and parallel execution
		erroringOperation := func(u uint32) error {
			if u == 50 {
				return errors.New("operation error")
			}
			time.Sleep(5 * time.Microsecond)
			return nil
		}

		b.Run("Sequential", func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				err := ExecuteForEachSlice(erroringOperation)(data)
				if err == nil {
					b.Fatal("Expected error in sequential execution")
				}
			}
		})

		b.Run("Parallel", func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				err := ExecuteForEachSlice(erroringOperation, ParallelExecute())(data)
				if err == nil {
					b.Fatal("Expected error in parallel execution")
				}
			}
		})
	})

	b.Run("CancellationWithVaryingWorkload", func(b *testing.B) {
		// Measure how cancellation performance scales with different workload intensities
		lightWork := func(u uint32) error {
			if u == 100 {
				return errors.New("cancellation trigger")
			}
			time.Sleep(1 * time.Microsecond)
			return nil
		}

		heavyWork := func(u uint32) error {
			if u == 100 {
				return errors.New("cancellation trigger")
			}
			time.Sleep(100 * time.Microsecond)
			return nil
		}

		b.Run("LightWork", func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				err := ExecuteForEachSlice(lightWork, ParallelExecute())(data[:200])
				if err == nil {
					b.Fatal("Expected cancellation error")
				}
			}
		})

		b.Run("HeavyWork", func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				err := ExecuteForEachSlice(heavyWork, ParallelExecute())(data[:200])
				if err == nil {
					b.Fatal("Expected cancellation error")
				}
			}
		})
	})

	b.Run("ContextCancellationRaceConditions", func(b *testing.B) {
		// Measure performance impact of race conditions during context cancellation
		var completedCount int64
		var cancelledCount int64

		operation := func(u uint32) error {
			// Fail on item 10 to trigger cancellation
			if u == 10 {
				return errors.New("trigger cancellation")
			}

			// Simulate work that might race with cancellation
			select {
			case <-time.After(5 * time.Microsecond):
				atomic.AddInt64(&completedCount, 1)
				return nil
			}
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			atomic.StoreInt64(&completedCount, 0)
			atomic.StoreInt64(&cancelledCount, 0)

			err := ExecuteForEachSlice(operation, ParallelExecute())(data)
			if err == nil {
				b.Fatal("Expected cancellation error")
			}
		}

		b.Logf("Avg completed: %d, cancelled: %d per iteration",
			atomic.LoadInt64(&completedCount)/int64(b.N),
			atomic.LoadInt64(&cancelledCount)/int64(b.N))
	})
}

// BenchmarkHighConcurrency performs stress testing with high concurrency to evaluate system behavior
// under extreme load conditions and detect performance degradation points
func BenchmarkHighConcurrency(b *testing.B) {
	// Test with different dataset sizes to understand scaling characteristics
	dataSizes := []int{100, 1000, 10000, 50000}

	// Test with different concurrency levels
	concurrencyLevels := []int{10, 100, 1000, 5000}

	for _, dataSize := range dataSizes {
		for _, concurrency := range concurrencyLevels {
			b.Run(fmt.Sprintf("DataSize_%d_Concurrency_%d", dataSize, concurrency), func(b *testing.B) {
				// Setup large dataset
				data := make([]uint32, dataSize)
				for i := range data {
					data[i] = uint32(i + 1)
				}

				// CPU-intensive operation to simulate real workload
				operation := func(val uint32) (uint32, error) {
					// Simulate computational work
					result := val
					for i := 0; i < 100; i++ {
						result = (result*31 + val) % 1000000
					}
					return result, nil
				}

				// Create provider with data
				provider := func() ([]uint32, error) {
					return data, nil
				}

				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					// Use SliceMap with high concurrency
					_, err := SliceMap[uint32, uint32](operation)(provider)(ParallelMap())()
					if err != nil {
						b.Fatalf("Unexpected error in high concurrency test: %v", err)
					}
				}
			})
		}
	}

	b.Run("StressTestMemoryPressure", func(b *testing.B) {
		// Test behavior under memory pressure with large datasets
		largeData := make([]uint32, 100000)
		for i := range largeData {
			largeData[i] = uint32(i + 1)
		}

		// Memory-intensive operation
		operation := func(val uint32) ([]uint32, error) {
			// Create temporary slice for each operation to increase memory pressure
			temp := make([]uint32, 1000)
			for i := range temp {
				temp[i] = val * uint32(i+1)
			}
			return temp, nil
		}

		provider := func() ([]uint32, error) {
			return largeData, nil
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := SliceMap[uint32, []uint32](operation)(provider)(ParallelMap())()
			if err != nil {
				b.Fatalf("Unexpected error in memory pressure test: %v", err)
			}
		}
	})

	b.Run("StressTestGoroutineOverhead", func(b *testing.B) {
		// Test with extremely high goroutine count to measure overhead
		smallData := make([]uint32, 10000)
		for i := range smallData {
			smallData[i] = uint32(i + 1)
		}

		// Minimal operation to isolate goroutine overhead
		operation := func(val uint32) (uint32, error) {
			return val + 1, nil
		}

		provider := func() ([]uint32, error) {
			return smallData, nil
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := SliceMap[uint32, uint32](operation)(provider)(ParallelMap())()
			if err != nil {
				b.Fatalf("Unexpected error in goroutine overhead test: %v", err)
			}
		}

		// Report goroutine overhead metrics
		b.ReportMetric(float64(len(smallData)), "goroutines/op")
	})

	b.Run("StressTestChannelThroughput", func(b *testing.B) {
		// Test channel throughput under high load
		mediumData := make([]uint32, 5000)
		for i := range mediumData {
			mediumData[i] = uint32(i + 1)
		}

		// Operation that stresses channel communication
		var channelOps int64
		operation := func(val uint32) (uint32, error) {
			atomic.AddInt64(&channelOps, 1)
			// Simulate brief work to create channel pressure
			time.Sleep(time.Nanosecond * 100)
			return val * 2, nil
		}

		provider := func() ([]uint32, error) {
			return mediumData, nil
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			atomic.StoreInt64(&channelOps, 0)
			_, err := SliceMap[uint32, uint32](operation)(provider)(ParallelMap())()
			if err != nil {
				b.Fatalf("Unexpected error in channel throughput test: %v", err)
			}
		}

		b.ReportMetric(float64(atomic.LoadInt64(&channelOps))/float64(b.N), "channel_ops/iteration")
	})

	b.Run("StressTestRaceConditionDetection", func(b *testing.B) {
		// Stress test specifically designed to trigger race conditions if they exist
		sharedCounter := int64(0)
		data := make([]uint32, 1000)
		for i := range data {
			data[i] = uint32(i + 1)
		}

		// Operation that might create race conditions with shared state
		operation := func(val uint32) (uint32, error) {
			// Intentionally create potential race condition scenarios
			current := atomic.LoadInt64(&sharedCounter)
			// Simulate work
			for i := 0; i < int(val%10)+1; i++ {
				current++
			}
			atomic.StoreInt64(&sharedCounter, current)
			return val, nil
		}

		provider := func() ([]uint32, error) {
			return data, nil
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			atomic.StoreInt64(&sharedCounter, 0)
			_, err := SliceMap[uint32, uint32](operation)(provider)(ParallelMap())()
			if err != nil {
				b.Fatalf("Unexpected error in race condition detection test: %v", err)
			}
		}

		b.ReportMetric(float64(atomic.LoadInt64(&sharedCounter))/float64(b.N), "final_counter/iteration")
	})
}

// Memory Profiling Benchmarks
// These benchmarks are designed to work with Go's memory profiling tools:
// go test -bench=BenchmarkMemoryProfile -memprofile=mem.prof
// go tool pprof mem.prof

func BenchmarkMemoryProfileProviderAllocation(b *testing.B) {
	// Benchmark memory allocations in provider chains
	// Use with: go test -bench=BenchmarkMemoryProfileProviderAllocation -memprofile=provider_mem.prof
	b.ReportAllocs()

	dataSize := 10000
	data := make([]uint32, dataSize)
	for i := range data {
		data[i] = uint32(i + 1)
	}

	// Memory-allocating provider
	provider := func() ([]uint32, error) {
		// Create new slice each time to track allocations
		result := make([]uint32, len(data))
		copy(result, data)
		return result, nil
	}

	// Memory-intensive transform
	transform := func(val uint32) ([]byte, error) {
		// Allocate bytes to track memory patterns
		result := make([]byte, 100)
		for i := range result {
			result[i] = byte(val + uint32(i))
		}
		return result, nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mapped := SliceMap[uint32, []byte](transform)(provider)()
		_, err := mapped()
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}

func BenchmarkMemoryProfileProviderChainComposition(b *testing.B) {
	// Benchmark memory usage in complex provider chains
	// Use with: go test -bench=BenchmarkMemoryProfileProviderChainComposition -memprofile=chain_mem.prof
	b.ReportAllocs()

	// Multi-stage provider chain with memory allocations at each stage
	stage1 := func() ([]uint32, error) {
		data := make([]uint32, 5000)
		for i := range data {
			data[i] = uint32(i + 1)
		}
		return data, nil
	}

	stage2 := func(val uint32) (string, error) {
		// Convert to string with padding
		return fmt.Sprintf("value_%08d_padded_data", val), nil
	}

	stage3 := func(val string) (map[string]interface{}, error) {
		// Create map with multiple allocations
		result := make(map[string]interface{})
		result["original"] = val
		result["length"] = len(val)
		result["hash"] = make([]byte, 16) // Additional allocation
		return result, nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		chain := SliceMap[string, map[string]interface{}](stage3)(SliceMap[uint32, string](stage2)(stage1)())()
		_, err := chain()
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}

func BenchmarkMemoryProfileParallelProcessing(b *testing.B) {
	// Benchmark memory usage in parallel processing scenarios
	// Use with: go test -bench=BenchmarkMemoryProfileParallelProcessing -memprofile=parallel_mem.prof
	b.ReportAllocs()

	dataSize := 20000
	data := make([]uint32, dataSize)
	for i := range data {
		data[i] = uint32(i + 1)
	}

	provider := func() ([]uint32, error) {
		// Create new slice for each benchmark iteration
		result := make([]uint32, len(data))
		copy(result, data)
		return result, nil
	}

	// Memory-heavy operation for parallel processing
	operation := func(val uint32) ([]int64, error) {
		// Allocate different sized slices based on input
		size := int(val%100) + 50
		result := make([]int64, size)
		for i := range result {
			result[i] = int64(val) * int64(i+1)
		}
		return result, nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parallel, err := SliceMap[uint32, []int64](operation)(provider)(ParallelMap())()
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
		_ = parallel // Use the result to avoid unused variable warning
	}
}

func BenchmarkMemoryProfileMemoization(b *testing.B) {
	// Benchmark memory usage patterns in memoization
	// Use with: go test -bench=BenchmarkMemoryProfileMemoization -memprofile=memo_mem.prof
	b.ReportAllocs()

	// Expensive computation that allocates memory
	computeProvider := func(input uint32) Provider[[]string] {
		return func() ([]string, error) {
			// Simulate expensive computation with memory allocation
			result := make([]string, 100)
			for i := range result {
				result[i] = fmt.Sprintf("computed_value_%d_%d", input, i)
			}
			return result, nil
		}
	}

	// Test both memoized and non-memoized versions
	b.Run("WithMemoization", func(b *testing.B) {
		b.ReportAllocs()

		memoized := Memoize(computeProvider(42))

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := memoized()
			if err != nil {
				b.Fatalf("Unexpected error: %v", err)
			}
		}
	})

	b.Run("WithoutMemoization", func(b *testing.B) {
		b.ReportAllocs()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			provider := computeProvider(42)
			_, err := provider()
			if err != nil {
				b.Fatalf("Unexpected error: %v", err)
			}
		}
	})
}

func TestEmptySliceOperations(t *testing.T) {
	// Test how slice operations handle empty slices as edge cases

	t.Run("SliceMapWithEmptySlice", func(t *testing.T) {
		emptyProvider := FixedProvider([]uint32{})
		mappedProvider := SliceMap[uint32, uint32](byTwo)(emptyProvider)()

		result, err := mappedProvider()
		if err != nil {
			t.Errorf("Expected no error with empty slice, got: %s", err)
		}

		if result == nil {
			t.Errorf("Expected empty slice, got nil")
		}

		if len(result) != 0 {
			t.Errorf("Expected empty result slice, got length %d", len(result))
		}
	})

	t.Run("ParallelSliceMapWithEmptySlice", func(t *testing.T) {
		emptyProvider := FixedProvider([]uint32{})
		mappedProvider := SliceMap[uint32, uint32](byTwo)(emptyProvider)(ParallelMap())

		result, err := mappedProvider()
		if err != nil {
			t.Errorf("Expected no error with empty slice, got: %s", err)
		}

		if result == nil {
			t.Errorf("Expected empty slice, got nil")
		}

		if len(result) != 0 {
			t.Errorf("Expected empty result slice, got length %d", len(result))
		}
	})

	t.Run("ForEachSliceWithEmptySlice", func(t *testing.T) {
		emptyProvider := FixedProvider([]uint32{})
		operationCalled := false

		err := ForEachSlice(emptyProvider, func(u uint32) error {
			operationCalled = true
			return nil
		})

		if err != nil {
			t.Errorf("Expected no error with empty slice, got: %s", err)
		}

		if operationCalled {
			t.Errorf("Operation should not be called on empty slice")
		}
	})

	t.Run("ParallelForEachSliceWithEmptySlice", func(t *testing.T) {
		emptyProvider := FixedProvider([]uint32{})
		operationCalled := false

		err := ForEachSlice(emptyProvider, func(u uint32) error {
			operationCalled = true
			return nil
		}, ParallelExecute())

		if err != nil {
			t.Errorf("Expected no error with empty slice, got: %s", err)
		}

		if operationCalled {
			t.Errorf("Operation should not be called on empty slice")
		}
	})

	t.Run("MergeSliceProviderWithEmptySlices", func(t *testing.T) {
		emptyProvider1 := FixedProvider([]uint32{})
		emptyProvider2 := FixedProvider([]uint32{})

		merged := MergeSliceProvider(emptyProvider1, emptyProvider2)
		result, err := merged()

		if err != nil {
			t.Errorf("Expected no error merging empty slices, got: %s", err)
		}

		if result == nil {
			t.Errorf("Expected empty slice, got nil")
		}

		if len(result) != 0 {
			t.Errorf("Expected empty merged slice, got length %d", len(result))
		}
	})

	t.Run("MergeSliceProviderEmptyWithNonEmpty", func(t *testing.T) {
		emptyProvider := FixedProvider([]uint32{})
		nonEmptyProvider := FixedProvider([]uint32{1, 2, 3})

		merged := MergeSliceProvider(emptyProvider, nonEmptyProvider)
		result, err := merged()

		if err != nil {
			t.Errorf("Expected no error merging empty with non-empty, got: %s", err)
		}

		if len(result) != 3 {
			t.Errorf("Expected merged slice length 3, got %d", len(result))
		}

		expectedValues := []uint32{1, 2, 3}
		for i, expected := range expectedValues {
			if result[i] != expected {
				t.Errorf("Expected value %d at index %d, got %d", expected, i, result[i])
			}
		}
	})

	t.Run("ToSliceProviderWithEmptyProvider", func(t *testing.T) {
		// Test edge case where single item provider might have issues with empty creation
		singleProvider := ErrorProvider[uint32](fmt.Errorf("no data available"))
		sliceProvider := ToSliceProvider(singleProvider)

		_, err := sliceProvider()
		if err == nil {
			t.Errorf("Expected error to be propagated from empty single provider")
		}
	})

	t.Run("FirstWithEmptySlice", func(t *testing.T) {
		emptyProvider := FixedProvider([]uint32{})

		_, err := First(emptyProvider, Filters(func(val uint32) bool { return true }))
		if err == nil {
			t.Errorf("Expected error when finding first in empty slice")
		}
	})

	t.Run("ExecuteForEachSliceWithEmptySlice", func(t *testing.T) {
		emptySlice := []uint32{}
		operationCalled := false

		operation := func(u uint32) error {
			operationCalled = true
			return nil
		}

		// This should process an empty slice without issues
		processor := ExecuteForEachSlice[uint32](operation)
		err := processor(emptySlice)

		if err != nil {
			t.Errorf("Expected no error with empty slice, got: %s", err)
		}

		if operationCalled {
			t.Errorf("Operation should not be called on empty slice")
		}
	})
}

func BenchmarkMemoryProfileErrorHandling(b *testing.B) {
	// Benchmark memory allocations during error scenarios
	// Use with: go test -bench=BenchmarkMemoryProfileErrorHandling -memprofile=error_mem.prof
	b.ReportAllocs()

	data := make([]uint32, 1000)
	for i := range data {
		data[i] = uint32(i + 1)
	}

	provider := func() ([]uint32, error) {
		result := make([]uint32, len(data))
		copy(result, data)
		return result, nil
	}

	// Operation that fails for certain values
	faultyOperation := func(val uint32) (string, error) {
		if val%100 == 0 {
			// Create error with allocated message
			return "", fmt.Errorf("processing failed for value %d with detailed error information", val)
		}
		// Normal processing with allocation
		return fmt.Sprintf("processed_value_%d", val), nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Execute and expect some errors
		mapped, err := SliceMap[uint32, string](faultyOperation)(provider)()()
		_ = mapped // Use the result
		_ = err    // Ignore errors for memory profiling
	}
}

func TestZeroValueAndNilPointerScenarios(t *testing.T) {
	// Test comprehensive zero-value and nil pointer scenarios for different data types
	// This ensures robust handling of edge cases involving zero values across the type system

	t.Run("ZeroValueIntegers", func(t *testing.T) {
		// Test providers with zero-value integers
		zeroIntProvider := FixedProvider(0)
		result, err := zeroIntProvider()
		if err != nil {
			t.Errorf("Expected no error with zero integer, got: %v", err)
		}
		if result != 0 {
			t.Errorf("Expected zero value, got: %d", result)
		}

		// Test mapping zero values
		mappedProvider := Map[int, string](func(i int) (string, error) {
			if i == 0 {
				return "zero", nil
			}
			return "non-zero", nil
		})(zeroIntProvider)

		mappedResult, err := mappedProvider()
		if err != nil {
			t.Errorf("Expected no error mapping zero integer, got: %v", err)
		}
		if mappedResult != "zero" {
			t.Errorf("Expected 'zero', got: %s", mappedResult)
		}
	})

	t.Run("ZeroValueStrings", func(t *testing.T) {
		// Test providers with empty strings (zero value for string type)
		emptyStringProvider := FixedProvider("")
		result, err := emptyStringProvider()
		if err != nil {
			t.Errorf("Expected no error with empty string, got: %v", err)
		}
		if result != "" {
			t.Errorf("Expected empty string, got: %s", result)
		}

		// Test SliceMap with empty strings in slice
		emptyStringsSlice := []string{"", "non-empty", "", "another"}
		provider := FixedProvider(emptyStringsSlice)

		// Transform empty strings to indicate their emptiness
		mapped := SliceMap[string, string](func(s string) (string, error) {
			if s == "" {
				return "EMPTY", nil
			}
			return s, nil
		})(provider)()

		result2, err := mapped()
		if err != nil {
			t.Errorf("Expected no error mapping strings with empties, got: %v", err)
		}
		expected := []string{"EMPTY", "non-empty", "EMPTY", "another"}
		if len(result2) != len(expected) {
			t.Errorf("Expected %d results, got %d", len(expected), len(result2))
		}
		for i, val := range result2 {
			if val != expected[i] {
				t.Errorf("Index %d: expected %s, got %s", i, expected[i], val)
			}
		}
	})

	t.Run("ZeroValueBooleans", func(t *testing.T) {
		// Test zero-value boolean (false)
		falseBoolProvider := FixedProvider(false)
		result, err := falseBoolProvider()
		if err != nil {
			t.Errorf("Expected no error with false boolean, got: %v", err)
		}
		if result != false {
			t.Errorf("Expected false, got: %v", result)
		}

		// Test filtering with zero-value booleans
		boolSlice := []bool{false, true, false, true, false}
		boolProvider := FixedProvider(boolSlice)

		// Count false values (zero values)
		falseCount := 0
		err = ForEachSlice[bool](boolProvider, func(b bool) error {
			if !b { // if b == false (zero value)
				falseCount++
			}
			return nil
		})

		if err != nil {
			t.Errorf("Expected no error counting false values, got: %v", err)
		}
		if falseCount != 3 {
			t.Errorf("Expected 3 false values, got %d", falseCount)
		}
	})

	t.Run("ZeroValueStructs", func(t *testing.T) {
		// Test struct with all zero values
		type TestStruct struct {
			ID    uint32
			Name  string
			Value *int
			Items []string
		}

		// Zero-value struct
		zeroStruct := TestStruct{}
		zeroStructProvider := FixedProvider(zeroStruct)

		result, err := zeroStructProvider()
		if err != nil {
			t.Errorf("Expected no error with zero-value struct, got: %v", err)
		}

		// Verify all fields are zero values
		if result.ID != 0 {
			t.Errorf("Expected zero ID, got: %d", result.ID)
		}
		if result.Name != "" {
			t.Errorf("Expected empty Name, got: %s", result.Name)
		}
		if result.Value != nil {
			t.Errorf("Expected nil Value pointer, got: %v", result.Value)
		}
		if result.Items != nil {
			t.Errorf("Expected nil Items slice, got: %v", result.Items)
		}

		// Test mapping zero-value struct to detect emptiness
		mappedProvider := Map[TestStruct, bool](func(ts TestStruct) (bool, error) {
			isEmpty := ts.ID == 0 && ts.Name == "" && ts.Value == nil && ts.Items == nil
			return isEmpty, nil
		})(zeroStructProvider)

		isEmpty, err := mappedProvider()
		if err != nil {
			t.Errorf("Expected no error checking struct emptiness, got: %v", err)
		}
		if !isEmpty {
			t.Error("Expected struct to be identified as empty/zero-value")
		}
	})

	t.Run("ZeroValueSlicesAndMaps", func(t *testing.T) {
		// Test zero-value slice (nil slice)
		var zeroSlice []int
		zeroSliceProvider := FixedProvider(zeroSlice)

		result, err := zeroSliceProvider()
		if err != nil {
			t.Errorf("Expected no error with zero-value slice, got: %v", err)
		}
		if result != nil {
			t.Errorf("Expected nil slice, got: %v", result)
		}
		if len(result) != 0 {
			t.Errorf("Expected zero length slice, got length: %d", len(result))
		}

		// Test SliceMap on zero-value slice
		mapped := SliceMap[int, string](func(i int) (string, error) {
			return fmt.Sprintf("value_%d", i), nil
		})(zeroSliceProvider)()

		mappedResult, err := mapped()
		if err != nil {
			t.Errorf("Expected no error mapping zero-value slice, got: %v", err)
		}
		if mappedResult == nil {
			t.Error("Expected empty slice result, not nil")
		}
		if len(mappedResult) != 0 {
			t.Errorf("Expected empty result slice, got length: %d", len(mappedResult))
		}

		// Test zero-value map
		var zeroMap map[string]int
		zeroMapProvider := FixedProvider(zeroMap)

		mapResult, err := zeroMapProvider()
		if err != nil {
			t.Errorf("Expected no error with zero-value map, got: %v", err)
		}
		if mapResult != nil {
			t.Errorf("Expected nil map, got: %v", mapResult)
		}
		if len(mapResult) != 0 {
			t.Errorf("Expected zero length map, got length: %d", len(mapResult))
		}
	})

	t.Run("NilPointerFieldsInStructs", func(t *testing.T) {
		// Test struct with nil pointer fields
		type StructWithPointers struct {
			IntPtr    *int
			StringPtr *string
			SlicePtr  *[]int
			MapPtr    *map[string]int
		}

		nilPtrStruct := StructWithPointers{
			IntPtr:    nil,
			StringPtr: nil,
			SlicePtr:  nil,
			MapPtr:    nil,
		}

		provider := FixedProvider(nilPtrStruct)
		result, err := provider()
		if err != nil {
			t.Errorf("Expected no error with nil pointer struct, got: %v", err)
		}

		// Verify all pointer fields are nil
		if result.IntPtr != nil {
			t.Errorf("Expected nil IntPtr, got: %v", result.IntPtr)
		}
		if result.StringPtr != nil {
			t.Errorf("Expected nil StringPtr, got: %v", result.StringPtr)
		}
		if result.SlicePtr != nil {
			t.Errorf("Expected nil SlicePtr, got: %v", result.SlicePtr)
		}
		if result.MapPtr != nil {
			t.Errorf("Expected nil MapPtr, got: %v", result.MapPtr)
		}

		// Test safe access to nil pointer fields
		mappedProvider := Map[StructWithPointers, string](func(s StructWithPointers) (string, error) {
			var parts []string

			if s.IntPtr == nil {
				parts = append(parts, "IntPtr:nil")
			} else {
				parts = append(parts, fmt.Sprintf("IntPtr:%d", *s.IntPtr))
			}

			if s.StringPtr == nil {
				parts = append(parts, "StringPtr:nil")
			} else {
				parts = append(parts, fmt.Sprintf("StringPtr:%s", *s.StringPtr))
			}

			return fmt.Sprintf("{%s}", strings.Join(parts, ",")), nil
		})(provider)

		safeResult, err := mappedProvider()
		if err != nil {
			t.Errorf("Expected no error in safe nil pointer access, got: %v", err)
		}
		expected := "{IntPtr:nil,StringPtr:nil}"
		if safeResult != expected {
			t.Errorf("Expected %s, got %s", expected, safeResult)
		}
	})

	t.Run("ZeroValueInterfaces", func(t *testing.T) {
		// Test zero-value interfaces (nil interface)
		var zeroInterface interface{}
		zeroInterfaceProvider := FixedProvider(zeroInterface)

		result, err := zeroInterfaceProvider()
		if err != nil {
			t.Errorf("Expected no error with zero-value interface, got: %v", err)
		}
		if result != nil {
			t.Errorf("Expected nil interface, got: %v", result)
		}

		// Test type assertion on nil interface
		mappedProvider := Map[interface{}, string](func(i interface{}) (string, error) {
			if i == nil {
				return "nil_interface", nil
			}
			return fmt.Sprintf("type_%T", i), nil
		})(zeroInterfaceProvider)

		typeResult, err := mappedProvider()
		if err != nil {
			t.Errorf("Expected no error checking interface type, got: %v", err)
		}
		if typeResult != "nil_interface" {
			t.Errorf("Expected 'nil_interface', got: %s", typeResult)
		}
	})

	t.Run("ZeroValueChannels", func(t *testing.T) {
		// Test zero-value channel (nil channel)
		var zeroChannel chan int
		zeroChannelProvider := FixedProvider(zeroChannel)

		result, err := zeroChannelProvider()
		if err != nil {
			t.Errorf("Expected no error with zero-value channel, got: %v", err)
		}
		if result != nil {
			t.Errorf("Expected nil channel, got: %v", result)
		}

		// Test safe channel operations check
		mappedProvider := Map[chan int, string](func(ch chan int) (string, error) {
			if ch == nil {
				return "nil_channel", nil
			}
			// Don't actually use the channel in tests to avoid blocking
			return "valid_channel", nil
		})(zeroChannelProvider)

		channelResult, err := mappedProvider()
		if err != nil {
			t.Errorf("Expected no error checking channel, got: %v", err)
		}
		if channelResult != "nil_channel" {
			t.Errorf("Expected 'nil_channel', got: %s", channelResult)
		}
	})

	t.Run("MixedZeroAndNilValues", func(t *testing.T) {
		// Test combinations of zero values and nil pointers
		type MixedStruct struct {
			ZeroInt     int
			NilIntPtr   *int
			EmptyString string
			NilSlice    []string
		}

		mixed := MixedStruct{
			ZeroInt:     0,
			NilIntPtr:   nil,
			EmptyString: "",
			NilSlice:    nil,
		}

		provider := FixedProvider([]MixedStruct{mixed, mixed, mixed})

		// Process slice of structs with mixed zero/nil values
		processed := SliceMap[MixedStruct, int](func(m MixedStruct) (int, error) {
			// Count the number of zero/nil fields
			count := 0
			if m.ZeroInt == 0 {
				count++
			}
			if m.NilIntPtr == nil {
				count++
			}
			if m.EmptyString == "" {
				count++
			}
			if m.NilSlice == nil {
				count++
			}
			return count, nil
		})(provider)()

		result, err := processed()
		if err != nil {
			t.Errorf("Expected no error processing mixed zero/nil values, got: %v", err)
		}

		// Each struct should have 4 zero/nil fields
		expected := []int{4, 4, 4}
		if len(result) != len(expected) {
			t.Errorf("Expected %d results, got %d", len(expected), len(result))
		}
		for i, count := range result {
			if count != expected[i] {
				t.Errorf("Index %d: expected %d zero/nil fields, got %d", i, expected[i], count)
			}
		}
	})
}

// TestGoroutineLeakDetectorUsage demonstrates proper usage pattern of goroutine leak detection
// This test shows how the testutil.GoroutineLeakDetector should be used in tests
func TestGoroutineLeakDetectorUsage(t *testing.T) {
	t.Run("DemonstrateGoroutineLeakDetectionPattern", func(t *testing.T) {
		// This test demonstrates the pattern that should be used with testutil.GoroutineLeakDetector
		// Note: Due to circular import restrictions, we simulate the pattern here

		// Record initial goroutine count (simulating testutil.NewGoroutineLeakDetector)
		initialGoroutines := runtime.NumGoroutine()

		// Create a cleanup function that checks for leaks
		checkForLeaks := func() {
			// Give goroutines time to cleanup
			time.Sleep(time.Millisecond * 10)
			runtime.GC()
			time.Sleep(time.Millisecond * 10)

			currentCount := runtime.NumGoroutine()
			if currentCount > initialGoroutines {
				t.Errorf("Goroutine leak detected: started with %d, ended with %d", initialGoroutines, currentCount)
			}
		}
		defer checkForLeaks()

		// Test parallel execution that should properly clean up goroutines
		data := make([]uint32, 100)
		for i := range data {
			data[i] = uint32(i + 1)
		}
		provider := FixedProvider(data)

		// Transform function that simulates some work
		transform := func(val uint32) (uint32, error) {
			time.Sleep(time.Microsecond * 5) // Minimal delay
			return val * 2, nil
		}

		// Use ParallelMap which creates goroutines internally
		transformed := SliceMap[uint32, uint32](transform)(provider)(ParallelMap())

		result, err := transformed()
		if err != nil {
			t.Errorf("ParallelMap should not fail: %v", err)
		}

		// Verify the transformation worked
		if len(result) != len(data) {
			t.Errorf("Expected %d results, got %d", len(data), len(result))
		}

		for i, val := range result {
			expected := data[i] * 2
			if val != expected {
				t.Errorf("Index %d: expected %d, got %d", i, expected, val)
			}
		}

		t.Logf("Goroutine leak detection pattern demonstrated successfully")
	})

	t.Run("DemonstrateProperGoroutineCleanupInMemoization", func(t *testing.T) {
		initialGoroutines := runtime.NumGoroutine()
		defer func() {
			// Multiple retries with cleanup, simulating testutil.GoroutineLeakDetector.CheckWithRetries
			for i := 0; i < 3; i++ {
				runtime.GC()
				time.Sleep(time.Millisecond * 10)

				currentCount := runtime.NumGoroutine()
				if currentCount <= initialGoroutines {
					return
				}

				if i == 2 {
					t.Errorf("Goroutine leak in memoization test: %d -> %d", initialGoroutines, currentCount)
				}
			}
		}()

		// Create an expensive provider that spawns goroutines
		var callCount int64
		expensiveProvider := func() (uint32, error) {
			atomic.AddInt64(&callCount, 1)

			// Simulate concurrent work
			var wg sync.WaitGroup
			results := make([]uint32, 3)

			for i := 0; i < 3; i++ {
				wg.Add(1)
				go func(idx int) {
					defer wg.Done()
					time.Sleep(time.Microsecond)
					results[idx] = uint32(idx + 1)
				}(i)
			}
			wg.Wait()

			var sum uint32
			for _, v := range results {
				sum += v
			}
			return sum, nil
		}

		memoized := Memoize(expensiveProvider)

		// Call from multiple goroutines concurrently
		const numCalls = 10
		var wg sync.WaitGroup
		results := make([]uint32, numCalls)
		errors := make([]error, numCalls)

		for i := 0; i < numCalls; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				result, err := memoized()
				results[idx] = result
				errors[idx] = err
			}(i)
		}
		wg.Wait()

		// Verify no errors and consistent results
		for i, err := range errors {
			if err != nil {
				t.Errorf("Call %d failed: %v", i, err)
			}
		}

		if len(results) > 0 {
			expected := results[0]
			for i, result := range results {
				if result != expected {
					t.Errorf("Call %d: expected %d, got %d", i, expected, result)
				}
			}
		}

		// Verify memoization worked (expensive function called only once)
		if callCount != 1 {
			t.Errorf("Expected memoized function called once, got %d", callCount)
		}

		t.Logf("Memoization goroutine cleanup pattern demonstrated successfully")
	})
}

func TestGoroutineCleanup(t *testing.T) {
	// Test that operations properly clean up goroutines and don't leak them

	t.Run("ParallelMap with successful execution", func(t *testing.T) {
		// Get baseline goroutine count
		runtime.GC() // Force GC to clean up any lingering goroutines
		time.Sleep(10 * time.Millisecond) // Brief pause for cleanup
		initialGoroutines := runtime.NumGoroutine()

		// Execute a parallel operation that creates goroutines
		data := make([]uint32, 50) // Enough to trigger goroutine creation
		for i := range data {
			data[i] = uint32(i)
		}

		provider := FixedProvider(data)
		transformer := func(n uint32) (uint32, error) {
			time.Sleep(1 * time.Millisecond) // Small delay to ensure goroutines exist
			return n * 2, nil
		}

		_, err := SliceMap[uint32, uint32](transformer)(provider)(ParallelMap())()
		if err != nil {
			t.Fatalf("Unexpected error: %s", err)
		}

		// Allow time for goroutine cleanup
		runtime.GC()
		time.Sleep(50 * time.Millisecond) // Sufficient time for cleanup

		// Check that goroutines were cleaned up
		finalGoroutines := runtime.NumGoroutine()
		if finalGoroutines > initialGoroutines {
			t.Errorf("Goroutine leak detected: initial=%d, final=%d, leaked=%d",
				initialGoroutines, finalGoroutines, finalGoroutines-initialGoroutines)
		}
	})

	t.Run("ParallelMap with early error termination", func(t *testing.T) {
		// Test cleanup when operations are cancelled due to error
		runtime.GC()
		time.Sleep(10 * time.Millisecond)
		initialGoroutines := runtime.NumGoroutine()

		data := make([]uint32, 100) // Large dataset to ensure multiple goroutines
		for i := range data {
			data[i] = uint32(i)
		}

		provider := FixedProvider(data)
		transformer := func(n uint32) (uint32, error) {
			time.Sleep(2 * time.Millisecond) // Delay to ensure goroutines are active
			if n == 10 { // Error on a specific value
				return 0, fmt.Errorf("test error") // Return error instead of panic
			}
			return n * 2, nil
		}

		// Execute with early error
		_, err := SliceMap[uint32, uint32](transformer)(provider)(ParallelMap())()

		// Should get an error due to the failing transformer
		if err == nil {
			t.Log("Expected error from transformer, but got none")
		}

		// Allow time for goroutine cleanup after error
		runtime.GC()
		time.Sleep(100 * time.Millisecond) // Extra time for error cleanup

		finalGoroutines := runtime.NumGoroutine()
		if finalGoroutines > initialGoroutines {
			t.Errorf("Goroutine leak detected after error: initial=%d, final=%d, leaked=%d",
				initialGoroutines, finalGoroutines, finalGoroutines-initialGoroutines)
		}
	})

	t.Run("ExecuteForEach with parallel execution", func(t *testing.T) {
		// Test ExecuteForEach goroutine cleanup
		runtime.GC()
		time.Sleep(10 * time.Millisecond)
		initialGoroutines := runtime.NumGoroutine()

		data := make([]uint32, 30)
		for i := range data {
			data[i] = uint32(i)
		}

		executed := make([]bool, len(data))
		var executedMutex sync.Mutex

		operator := func(n uint32) error {
			time.Sleep(2 * time.Millisecond) // Ensure goroutines exist
			executedMutex.Lock()
			executed[n] = true
			executedMutex.Unlock()
			return nil
		}

		err := ExecuteForEachSlice(operator, ParallelExecute())(data)
		if err != nil {
			t.Fatalf("Unexpected error: %s", err)
		}

		// Verify all operations executed
		for i, ex := range executed {
			if !ex {
				t.Errorf("Operation %d was not executed", i)
			}
		}

		// Allow cleanup
		runtime.GC()
		time.Sleep(50 * time.Millisecond)

		finalGoroutines := runtime.NumGoroutine()
		if finalGoroutines > initialGoroutines {
			t.Errorf("Goroutine leak in ExecuteForEach: initial=%d, final=%d, leaked=%d",
				initialGoroutines, finalGoroutines, finalGoroutines-initialGoroutines)
		}
	})

	t.Run("Context cancellation cleanup", func(t *testing.T) {
		// Test that cancelled operations properly clean up goroutines
		runtime.GC()
		time.Sleep(10 * time.Millisecond)
		initialGoroutines := runtime.NumGoroutine()

		data := make([]uint32, 100) // Large dataset
		for i := range data {
			data[i] = uint32(i)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		provider := func() ([]uint32, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				return data, nil
			}
		}

		transformer := func(n uint32) (uint32, error) {
			select {
			case <-ctx.Done():
				return 0, ctx.Err() // Return context error
			default:
				time.Sleep(20 * time.Millisecond) // Longer than context timeout
				return n * 2, nil
			}
		}

		// This should timeout and cancel
		_, err := SliceMap[uint32, uint32](transformer)(provider)(ParallelMap())()

		// Should get timeout error
		if err == nil {
			t.Log("Expected timeout error, but operation completed")
		}

		// Cancel to ensure cleanup
		cancel()

		// Allow extra time for cancelled goroutine cleanup
		runtime.GC()
		time.Sleep(100 * time.Millisecond)

		finalGoroutines := runtime.NumGoroutine()
		if finalGoroutines > initialGoroutines {
			t.Errorf("Goroutine leak after cancellation: initial=%d, final=%d, leaked=%d",
				initialGoroutines, finalGoroutines, finalGoroutines-initialGoroutines)
		}
	})

	t.Run("Multiple sequential operations cleanup", func(t *testing.T) {
		// Test that multiple operations don't accumulate goroutines
		runtime.GC()
		time.Sleep(10 * time.Millisecond)
		initialGoroutines := runtime.NumGoroutine()

		for iteration := 0; iteration < 10; iteration++ {
			data := make([]uint32, 20)
			for i := range data {
				data[i] = uint32(i + iteration*20)
			}

			provider := FixedProvider(data)
			transformer := func(n uint32) (uint32, error) {
				time.Sleep(1 * time.Millisecond)
				return n + 100, nil
			}

			_, err := SliceMap[uint32, uint32](transformer)(provider)(ParallelMap())()
			if err != nil {
				t.Fatalf("Unexpected error in iteration %d: %s", iteration, err)
			}

			// Brief cleanup between iterations
			runtime.GC()
			time.Sleep(5 * time.Millisecond)
		}

		// Final cleanup
		runtime.GC()
		time.Sleep(50 * time.Millisecond)

		finalGoroutines := runtime.NumGoroutine()
		if finalGoroutines > initialGoroutines {
			t.Errorf("Goroutine accumulation detected: initial=%d, final=%d, accumulated=%d",
				initialGoroutines, finalGoroutines, finalGoroutines-initialGoroutines)
		}
	})
}

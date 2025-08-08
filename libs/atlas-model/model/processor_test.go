package model

import (
	"errors"
	"fmt"
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

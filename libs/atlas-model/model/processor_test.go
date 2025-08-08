package model

import (
	"fmt"
	"testing"
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
	count := uint32(0)

	err := ForEachSlice(p, func(u uint32) error {
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

	err := ForEachMap(p, func(k uint32) Operator[[]uint32] {
		return func(vs []uint32) error {
			count := uint32(0)
			for _, v := range vs {
				count += v
			}
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
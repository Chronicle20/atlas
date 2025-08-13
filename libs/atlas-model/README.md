# atlas-model

Module which provides uniform operations on object models.

## Overview

This package provides functional programming utilities with lazy evaluation for data processing pipelines. All operations are designed with thread-safety in mind, enabling safe concurrent usage in Go applications.

## Key Features

- **Lazy Evaluation**: Operations are deferred until the final Provider is invoked
- **Parallel Execution**: Thread-safe parallel processing for improved performance
- **Functional Composition**: Build complex processing pipelines through function composition
- **Race-Condition-Free**: Comprehensive thread-safety guarantees for all parallel operations

## Thread Safety Guarantees

### Parallel Execution Functions

#### `ExecuteForEachSlice()` and `ExecuteForEachMap()`
- **Thread-safe**: Safe for concurrent invocation from multiple goroutines
- **Error Handling**: First error encountered terminates all parallel operations using context cancellation
- **Channel Safety**: Proper channel buffering and synchronization to prevent race conditions
- **Memory Safety**: No shared mutable state between goroutines

#### `SliceMap()` with `ParallelMap()`
- **Ordered Results**: Maintains slice element order despite parallel processing
- **Thread-safe Transformations**: Each element processed in isolation with no shared state
- **Error Propagation**: Any transformation error terminates all parallel work

#### `Memoize()`
- **Thread-safe Caching**: Multiple goroutines can safely call memoized Provider concurrently
- **Once Semantics**: Underlying computation executes exactly once using `sync.Once`

### Validation

All parallel execution functions have been thoroughly tested with Go's race detector (`go test -race`) to ensure thread-safety. The test suite includes:

- Concurrent access patterns
- High-contention scenarios  
- Error handling under concurrent conditions
- Channel communication safety
- Context cancellation race conditions

### Usage Guidelines

- All Provider functions are safe for concurrent invocation
- Parallel execution variants (`ParallelExecute()`, `ParallelMap()`) are production-ready
- Error handling properly terminates parallel operations without race conditions
- Channel operations use appropriate buffering and synchronization

## Testing Guidelines

### Running Tests

#### Basic Test Execution
```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run specific package tests
go test ./model
go test ./async
```

#### Race Condition Detection
```bash
# Run tests with race detector (required for CI)
go test -race ./...

# Run with extended race detection
go test -race -v ./...
```

#### Coverage Analysis
```bash
# Generate coverage report
go test -cover ./...

# Generate detailed coverage profile
go test -cover -coverprofile=coverage.out ./...

# View coverage in browser
go tool cover -html=coverage.out
```

### Performance Testing

#### Benchmark Execution
```bash
# Run all benchmarks
go test -bench=. ./...

# Run benchmarks with memory profiling
go test -bench=. -benchmem ./...

# Run benchmarks with extended execution time
go test -bench=. -benchtime=3s -count=5 ./...
```

#### Automated Regression Detection
The project includes automated benchmark regression detection:

```bash
# Run benchmark regression detection (as used in CI)
./scripts/benchmark-regression.sh run

# Create/update baseline for current branch
./scripts/benchmark-regression.sh baseline
```

See `scripts/README.md` for detailed information about the regression detection system.

#### Memory Profiling
```bash
# Generate memory profiles for tests and benchmarks
./scripts/memory-profiling.sh both

# View memory profile
go tool pprof profiles/test_memory_model_*.memprof
```

### Test Utilities

The `testutil` package provides comprehensive testing helpers:

#### Goroutine Leak Detection
```go
func TestExample(t *testing.T) {
    detector := testutil.NewGoroutineLeakDetector(t)
    defer detector.Check()
    
    // Your test code that might spawn goroutines
}
```

#### Context Testing
```go
// Test with cancellation
ctx := testutil.CancelledContext()
result, err := provider(ctx)

// Test with timeout
ctx, cancel := testutil.TimeoutContext(100 * time.Millisecond)
defer cancel()
```

#### Concurrent Testing
```go
func TestConcurrent(t *testing.T) {
    runner := testutil.NewConcurrentRunner()
    runner.Add(10)
    
    for i := 0; i < 10; i++ {
        runner.Go(func() error {
            // Concurrent test logic
            return nil
        })
    }
    
    runner.Wait()
    runner.CheckErrors(t)
}
```

#### Provider Testing
```go
// Create test providers
errorProvider := testutil.ErrorProvider[int](testutil.TestError("failure"))
fixedProvider := testutil.FixedProvider(42)
delayedProvider := testutil.DelayedProvider(100, 50*time.Millisecond)
```

### Testing Best Practices

#### Required Practices
1. **Always use race detector**: All tests must pass `go test -race`
2. **Maintain coverage**: Minimum 75% test coverage required
3. **Test error scenarios**: Include comprehensive error handling tests
4. **Verify resource cleanup**: Use goroutine leak detection for async operations
5. **Test edge cases**: Nil values, empty slices, cancelled contexts

#### Recommended Practices
1. **Use table-driven tests** for multiple input scenarios
2. **Test concurrent access patterns** for shared resources
3. **Include performance benchmarks** for critical paths
4. **Test context cancellation** in long-running operations
5. **Verify memory usage** with memory profiling

#### Test Organization
```go
func TestProviderOperation(t *testing.T) {
    tests := []struct {
        name     string
        input    []int
        expected int
        wantErr  bool
    }{
        {"normal case", []int{1, 2, 3}, 6, false},
        {"empty slice", []int{}, 0, false},
        {"nil slice", nil, 0, true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            detector := testutil.NewGoroutineLeakDetector(t)
            defer detector.Check()
            
            result, err := operation(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("expected error %v, got %v", tt.wantErr, err)
            }
            if result != tt.expected {
                t.Errorf("expected %v, got %v", tt.expected, result)
            }
        })
    }
}
```

### CI/CD Integration

The project uses GitHub Actions for automated testing:

#### Pull Request Testing
- Race condition detection with `go test -race`
- Coverage analysis with minimum 75% threshold
- Benchmark regression detection
- Memory profiling and leak detection

#### Baseline Maintenance
- Main branch updates create new benchmark baselines
- Performance profiles are archived for analysis
- Coverage reports are uploaded to Codecov

#### Artifacts
- Benchmark results (30-day retention)
- Memory profiles (30-day retention on PRs, 90-day on main)
- Coverage reports

### Coverage Requirements

- **Minimum Coverage**: 75% overall
- **Critical Paths**: >90% coverage for core provider operations
- **Error Handling**: All error paths must be tested
- **Edge Cases**: Comprehensive testing of boundary conditions

### Debugging Test Failures

#### Race Conditions
```bash
# Run problematic test with race detector
go test -race -run TestSpecificTest ./package

# Increase iterations to catch intermittent races
go test -race -count=100 -run TestSpecificTest ./package
```

#### Memory Leaks
```bash
# Run with memory profiling
go test -memprofile=mem.prof -run TestSpecificTest ./package

# Analyze memory usage
go tool pprof mem.prof
```

#### Performance Regressions
```bash
# Compare benchmarks manually
go test -bench=BenchmarkSpecific -benchtime=5s -count=10 ./package

# Use regression detection script
./scripts/benchmark-regression.sh run
```

For more detailed information about the testing infrastructure, see:
- `scripts/README.md` - Benchmark regression detection system
- `testutil/helpers.go` - Available test utilities
- `.github/workflows/` - CI/CD configuration
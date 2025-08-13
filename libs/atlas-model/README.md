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

### Test Coverage Requirements

The project maintains strict test coverage requirements to ensure reliability and production readiness.

#### Overall Coverage Targets

- **Minimum Overall Coverage**: 75% across all packages
- **Critical Paths Coverage**: >90% for core provider operations and parallel execution functions
- **New Code Coverage**: 100% coverage required for all new functions and methods
- **Package-Specific Targets**:
  - `model` package: Minimum 80% coverage
  - `async` package: Minimum 85% coverage (higher due to concurrency complexity)
  - `testutil` package: Minimum 90% coverage

#### Coverage Categories

##### 1. Functional Coverage
- **All Public APIs**: Every exported function must have test coverage
- **Return Value Paths**: All possible return scenarios must be tested
- **Parameter Combinations**: Test various input parameter combinations
- **State Transitions**: Cover all possible state changes in stateful operations

##### 2. Error Path Coverage
- **Error Propagation**: Test that errors bubble up correctly through provider chains
- **Error Recovery**: Verify graceful handling and recovery from error conditions
- **Partial Failures**: Test scenarios where some operations succeed and others fail
- **Resource Cleanup**: Ensure proper cleanup occurs even when errors are encountered

##### 3. Edge Case Coverage
- **Boundary Conditions**: Test with boundary values (zero, max, min values)
- **Nil and Empty Inputs**: Test behavior with nil pointers, empty slices, and zero-value structs
- **Invalid Contexts**: Test with cancelled, expired, and nil contexts
- **Concurrent Edge Cases**: Test race conditions, concurrent access, and high-contention scenarios

##### 4. Integration Coverage
- **Provider Chain Integration**: Test end-to-end provider chain execution
- **Cross-Package Integration**: Test interactions between `model` and `async` packages
- **External Dependencies**: Test integration with external systems (mocked appropriately)

#### Coverage Enforcement

##### CI/CD Enforcement
```yaml
# Coverage is enforced in GitHub Actions workflows
- Coverage check fails the build if below minimum thresholds
- Coverage reports are generated for all pull requests
- Coverage trends are tracked over time
```

##### Local Development
```bash
# Check coverage locally before committing
go test -cover ./...

# Generate detailed coverage report
go test -cover -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Check coverage by function
go tool cover -func=coverage.out
```

##### Coverage Exclusions
Some code may be excluded from coverage requirements:
- **Generated Code**: Auto-generated files (marked with `//go:generate`)
- **Debug/Development Code**: Code within `// +build debug` tags
- **Deprecated Functions**: Functions marked for deprecation (must be documented)

#### Coverage Quality Standards

##### 1. Meaningful Tests
- Tests must actually exercise the code, not just achieve coverage metrics
- Avoid "coverage theater" - tests that increase coverage but don't validate behavior
- Each test should have clear assertions and expected outcomes

##### 2. Test Types Required
- **Unit Tests**: Test individual functions and methods in isolation
- **Integration Tests**: Test component interactions and workflows
- **Property Tests**: Test invariants and properties across input ranges
- **Regression Tests**: Test for previously discovered bugs

##### 3. Coverage Reporting
- **Pull Request Reports**: Coverage changes highlighted in PR reviews
- **Trend Analysis**: Coverage tracked over time to prevent regression
- **Package Breakdown**: Coverage reported per package and per function
- **Differential Coverage**: Focus on coverage of changed lines in PRs

#### Monitoring and Maintenance

##### Coverage Tracking
- Coverage metrics stored in CI artifacts for historical analysis
- Weekly reports on coverage trends and package-level changes
- Alerts for significant coverage decreases

##### Review Process
- PRs with coverage decreases require additional review
- New features without adequate test coverage are rejected
- Coverage exceptions require team approval and documentation

##### Technical Debt
- Areas below coverage thresholds are tracked as technical debt
- Regular coverage improvement sprints to address gaps
- Refactoring efforts must maintain or improve coverage

#### Coverage Tools and Integration

##### Supported Tools
- **Built-in Coverage**: Standard Go coverage tools (`go test -cover`)
- **Coverage Visualization**: HTML reports and function-level analysis
- **CI Integration**: GitHub Actions with coverage reporting
- **External Services**: Integration with Codecov for advanced analytics

##### Coverage Commands
```bash
# Package-specific coverage
go test -cover ./model
go test -cover ./async

# Detailed function coverage
go test -cover -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | grep -E "(TOTAL|model/|async/)"

# Coverage with race detection
go test -race -cover ./...

# Benchmark coverage
go test -bench=. -cover ./...
```

#### Coverage Violation Handling

##### Pull Request Failures
When coverage requirements are not met:
1. **Immediate Action**: PR is blocked from merging
2. **Developer Guidance**: Clear feedback on which areas need more tests
3. **Resolution Path**: Add tests or request coverage exception approval

##### Exception Process
For legitimate coverage exceptions:
1. **Documentation**: Clearly document why coverage cannot be achieved
2. **Team Review**: Technical lead approval required for exceptions  
3. **Tracking**: Exceptions tracked and reviewed quarterly
4. **Mitigation**: Plan for future coverage improvement where possible

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
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
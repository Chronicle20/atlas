# HTTP Client Timeouts Plan

Last Updated: 2026-02-19

## Executive Summary

All 52+ Atlas microservices make cross-service REST calls through `http.DefaultClient` which has **zero timeout configuration**. A single slow or unresponsive downstream service can block goroutines indefinitely, cascading failures across the ecosystem. This plan introduces configurable client-side timeouts in `libs/atlas-rest` with sensible defaults, requiring zero changes to consuming services.

## Current State Analysis

### HTTP Client Usage

Three call sites in `libs/atlas-rest/requests/` use `http.DefaultClient.Do(req)`:
- `get.go:41` — GET requests
- `post.go:44` — POST/PUT/PATCH requests (shared `createOrUpdate`)
- `delete.go:37` — DELETE requests

`http.DefaultClient` has no `Timeout`, no custom `Transport`, no connection pool limits.

### Context Flow

Contexts are attached to requests via `req.WithContext(ctx)`, enabling cancellation. However, **no service creates contexts with timeouts or deadlines** for outbound calls. Contexts carry tenant/trace metadata only.

### Existing Timeout Protections

| Layer | Timeout | Protection Level |
|-------|---------|-----------------|
| Client-side (outbound) | **None** | No protection |
| Server-side (inbound) | 5s read, 10s write, 120s idle | Protects the server |
| Nginx ingress | 1800s | No meaningful protection |
| Retry | 1 attempt (no retry), 1s sleep | Minimal |

### Service Consumption Pattern

Every service wraps `atlas-rest` in an identical `rest/request.go` that adds span/tenant header decorators. The Configurator pattern (`SetRetries`, `AddHeaderDecorator`) is already in place for extending configuration. All services consume via factory functions that return `Request[A]` closures.

## Proposed Future State

### Architecture

Replace `http.DefaultClient` with a package-level configured `*http.Client` that includes:
1. A default request timeout (10 seconds)
2. A configured `http.Transport` with connection pool limits
3. A `Configurator` option to override timeout per-request

### Design Decisions

1. **Package-level client, not per-request** — Creating an `http.Client` per request is wasteful. A single shared client with sensible defaults mirrors what `http.DefaultClient` provides but with timeouts.

2. **Context deadline over Client.Timeout** — Use `context.WithTimeout` wrapping the caller's context rather than `http.Client.Timeout`. This is more idiomatic Go and composes with existing context cancellation. The `http.Client.Timeout` is set as a safety net.

3. **Configurator pattern for per-request overrides** — `SetTimeout(d time.Duration)` allows specific calls (e.g., long-running operations) to override the default.

4. **Zero changes to services** — The default timeout applies transparently. Services only need changes if they want to override the timeout for specific calls.

## Implementation Phases

### Phase 1: Client Configuration (Core)

Add a configured HTTP client and timeout support to `libs/atlas-rest/requests/`.

#### 1.1 Add timeout to configuration struct

In `config.go`, add a `timeout` field to `configuration` and a `SetTimeout` Configurator:

```go
type configuration struct {
    retries          int
    timeout          time.Duration
    headerDecorators []HeaderDecorator
}

func SetTimeout(d time.Duration) Configurator {
    return func(c *configuration) {
        c.timeout = d
    }
}
```

#### 1.2 Create package-level HTTP client

Add a new file `client.go` in `requests/` that initializes a shared `*http.Client`:

```go
package requests

import (
    "net/http"
    "time"
)

const (
    DefaultTimeout         = 10 * time.Second
    DefaultMaxIdleConns    = 100
    DefaultIdleConnTimeout = 90 * time.Second
)

var client = &http.Client{
    Timeout: 30 * time.Second, // absolute safety net
    Transport: &http.Transport{
        MaxIdleConns:        DefaultMaxIdleConns,
        MaxIdleConnsPerHost: 10,
        IdleConnTimeout:     DefaultIdleConnTimeout,
    },
}
```

#### 1.3 Update request functions to use client with context timeout

Replace `http.DefaultClient.Do(req)` in `get.go`, `post.go`, and `delete.go` with:

```go
reqCtx, cancel := context.WithTimeout(ctx, c.timeout)
defer cancel()
req = req.WithContext(reqCtx)
r, err = client.Do(req)
```

The default `c.timeout` is `DefaultTimeout` (10s). Per-request overrides use `SetTimeout()`.

### Phase 2: Transport Tuning

#### 2.1 Connection pool configuration

The default `http.Transport` has `MaxIdleConnsPerHost: 2`, which is too low for services making many concurrent requests to the same downstream. Set to 10.

#### 2.2 Environment variable override

Add an optional `HTTP_CLIENT_TIMEOUT` environment variable that overrides `DefaultTimeout` at startup:

```go
func init() {
    if v := os.Getenv("HTTP_CLIENT_TIMEOUT"); v != "" {
        if d, err := time.ParseDuration(v); err == nil {
            DefaultTimeout = d
        }
    }
}
```

### Phase 3: Testing

#### 3.1 Unit tests for timeout behavior

- Test that requests fail with context deadline exceeded after timeout
- Test that `SetTimeout` overrides the default
- Test that caller context cancellation still works
- Test that the retry mechanism respects timeout boundaries

#### 3.2 Integration validation

- Build all services to verify no compilation errors
- Deploy to dev environment and verify no behavioral regressions

## Detailed Task Breakdown

### Phase 1 Tasks

| # | Task | Effort | Depends On |
|---|------|--------|------------|
| 1.1 | Add `timeout` field to `configuration` and `SetTimeout` Configurator in `config.go` | S | — |
| 1.2 | Create `client.go` with configured `*http.Client` and constants | S | — |
| 1.3 | Update `get.go` to use `client` with context timeout | S | 1.1, 1.2 |
| 1.4 | Update `post.go` (`createOrUpdate`) to use `client` with context timeout | S | 1.1, 1.2 |
| 1.5 | Update `delete.go` to use `client` with context timeout | S | 1.1, 1.2 |

### Phase 2 Tasks

| # | Task | Effort | Depends On |
|---|------|--------|------------|
| 2.1 | Add env var override for default timeout in `client.go` | S | 1.2 |

### Phase 3 Tasks

| # | Task | Effort | Depends On |
|---|------|--------|------------|
| 3.1 | Write unit tests for timeout behavior | M | 1.3-1.5 |
| 3.2 | Run `go build` for all 52+ services to verify compilation | M | 1.3-1.5 |
| 3.3 | Run existing test suites | S | 1.3-1.5 |

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Default timeout too aggressive for some calls | Medium | Medium | 10s default is generous; `SetTimeout()` available for overrides; env var fallback |
| Connection pool exhaustion with new Transport | Low | Medium | Conservative pool sizes (100 total, 10 per host); monitor in dev |
| Breaking retry behavior | Low | High | Retry loop already catches transport errors; context deadline exceeded is a transport error and will trigger retry |
| Services depend on indefinite blocking | Low | Low | No service intentionally relies on unbounded waits |

## Success Metrics

1. All outbound HTTP requests have a bounded lifetime (default 10s)
2. Zero compilation errors across all services
3. All existing tests pass
4. No service code changes required for default behavior
5. Per-request timeout override available via `SetTimeout()` Configurator

## Required Resources and Dependencies

- **Code changes**: `libs/atlas-rest/requests/` only (3-4 files modified, 1 new file)
- **No new dependencies**: Uses only stdlib `net/http`, `context`, `time`, `os`
- **No service changes**: Default behavior applies transparently
- **Testing**: Existing test infrastructure sufficient; new tests use stdlib `net/http/httptest`

## Timeline Estimate

- Phase 1 (Core): 1-2 hours
- Phase 2 (Env override): 30 minutes
- Phase 3 (Testing + validation): 1-2 hours
- **Total: ~3-4 hours**

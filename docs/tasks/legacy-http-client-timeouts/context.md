# HTTP Client Timeouts — Context

Last Updated: 2026-02-19

## Key Files

### Library (modify)

| File | Purpose |
|------|---------|
| `libs/atlas-rest/requests/config.go` | Configuration struct + Configurator pattern — add `timeout` field |
| `libs/atlas-rest/requests/get.go` | GET requests — replace `http.DefaultClient` with configured client |
| `libs/atlas-rest/requests/post.go` | POST/PUT/PATCH requests — replace `http.DefaultClient` with configured client |
| `libs/atlas-rest/requests/delete.go` | DELETE requests — replace `http.DefaultClient` with configured client |
| `libs/atlas-rest/requests/client.go` | **NEW** — Package-level configured `*http.Client` |
| `libs/atlas-rest/retry/retry.go` | Retry logic — no changes needed, context deadline errors propagate naturally |

### Library (reference only)

| File | Purpose |
|------|---------|
| `libs/atlas-rest/requests/header.go` | Header decorators — unchanged |
| `libs/atlas-rest/requests/response.go` | Response processing — unchanged |
| `libs/atlas-rest/requests/provider.go` | Provider bridge — unchanged |
| `libs/atlas-rest/requests/url.go` | URL resolution — unchanged |
| `libs/atlas-rest/server/server.go` | Server-side timeouts (already configured) — reference for consistency |

### Service wrappers (no changes needed)

| Pattern | Example |
|---------|---------|
| `rest/request.go` | `services/atlas-character/atlas.com/character/rest/request.go` |

All 52+ services follow the identical wrapper pattern. Changes to `libs/atlas-rest` propagate automatically.

## Key Decisions

1. **Package-level client** — Single shared `*http.Client` replaces `http.DefaultClient`. Connection pooling is shared across all requests within a process.

2. **Context timeout (primary) + Client.Timeout (safety net)** — `context.WithTimeout` wrapping caller's context for per-request control. `http.Client.Timeout` set to 30s as an absolute backstop.

3. **Default 10 seconds** — Conservative enough to avoid false positives. Aggressive enough to prevent indefinite blocking. Overridable via `SetTimeout()` or `HTTP_CLIENT_TIMEOUT` env var.

4. **Zero service changes** — The default timeout applies transparently through the existing Configurator pattern. Services only need to add `SetTimeout()` if they have unusual latency requirements.

## Dependencies

- No new Go module dependencies
- No changes to `go.mod`
- No changes to any service code
- Uses only stdlib packages: `net/http`, `context`, `time`, `os`

## Interaction with Existing Features

### Retry
The retry mechanism (`retry.Try`) catches errors from `client.Do()`. Context deadline exceeded is returned as an error, which triggers retry if retries > 1. Each retry attempt gets a fresh context timeout (the `context.WithTimeout` is inside the retry loop's function).

### Header Decorators
No interaction — headers are applied before the request is sent, unaffected by timeout configuration.

### OpenTelemetry Spans
Span propagation via `SpanHeaderDecorator` is unaffected. The span context flows through the caller's context, which is the parent of the timeout context.

### Tenant Context
Tenant extraction via `TenantHeaderDecorator` reads from the caller's context, which is the parent of the timeout context. Unaffected.

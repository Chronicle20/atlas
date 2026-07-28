# task-168 Context — DB Connection Resilience

Companion to `plan.md`. Key files, locked decisions, dependencies, and gotchas an
implementer (or subagent) needs beyond the plan's step text.

## Worktree

All work happens in `.worktrees/task-168-db-connection-resilience` on branch
`task-168-db-connection-resilience`. Subagent prompts must cd there and verify
`git branch --show-current` after every commit.

## Key files

| File | Role |
|---|---|
| `libs/atlas-retry/retry.go` | `Try` loop (`attempt <= MaxRetries` — "retries: 1" means ONE attempt total); gains `WithDelayHint` (Task 1) |
| `libs/atlas-database/connection.go` | `Connect()` — currently `gorm.Open(postgres.Open(dsn))`; swapped to `sql.OpenDB(retryConnector)` + `postgres.New(postgres.Config{Conn:...})` (Task 4). Has `getIntEnv`/`getDurationEnv`/`try` helpers already |
| `libs/atlas-database/transient.go` (new) | classifier (Task 2) |
| `libs/atlas-database/metrics.go` (new) | counters + `CountTransient` + `registerDBStats` (Task 3) |
| `libs/atlas-database/connector.go` (new) | `newRetryConnector` (Task 4) |
| `libs/atlas-rest/requests/get.go` | GET path; default `retries: 1` today → 3; gains 503 handling (Task 6) |
| `libs/atlas-rest/requests/client_test.go` | httptest precedent for client tests (`delete` used as no-body-parsing proxy) |
| `libs/atlas-rest/server/error.go` (new) | 503 contract (Task 5) |
| `libs/atlas-rest/server/server.go` | Builder; `New()` gets the `/metrics` auto-mount (Task 7) |
| `libs/atlas-rest/server/context.go` | `NewHandlerDependency`/`NewHandlerContext` — exported, make handler tests possible |
| `libs/atlas-rest/degrade/` (new) | `Observe` (Task 9) |
| `libs/atlas-model/model/processor.go:101` | `Decorator[M]`; gains `ErrDecorator` (Task 8). Module must stay dependency-free |
| `services/atlas-login/atlas.com/login/character/processor.go:108` | the incident's silent decorator (Task 10) |
| `services/atlas-login/atlas.com/login/inventory/requests.go` | uses `requests.RootUrl("INVENTORY")` → env `INVENTORY_SERVICE_URL` (tests point this at httptest) |
| `services/atlas-inventory/atlas.com/inventory/{inventory,compartment,asset}/resource.go` | inline 500 writes → `server.WriteErrorResponse` (Task 11) |
| `services/atlas-inventory/atlas.com/inventory/main.go:62` | `database.Connect` — classifier registration goes right after |
| `services/atlas-{channel,summons,doors,monsters}/.../main.go` | the only 4 explicit `/metrics` mounts (lines 345/86/91/96) — removed in Task 7 |
| `.claude/agents/backend-guidelines-reviewer.md` | DOM-25 is the highest existing item → new ones are DOM-26/DOM-27 |

## Locked decisions (from design.md, plus planning-phase findings)

- **Retry layer = `driver.Connector.Connect` wrapper.** The pool only calls it
  before any SQL is sent, so FR-2.2 (never retry ambiguous work) holds
  structurally. GORM callbacks / query-helper retry were rejected in design.
- **Classifier order matters:** `*pgconn.PgError` (strict SQLSTATE allow-list:
  53300, 57P03, 08001, 08006) is checked BEFORE `*pgconn.ConnectError`, so an
  auth failure (28P01) raised during connect is NOT transient.
- **Classifier injection, not import:** `atlas-rest/server` must NOT import
  `atlas-database` (would drag gorm+pgx into DB-less services). main.go
  registers `func(err) bool` composing `IsTransientConnectionError` +
  `CountTransient`.
- **GET-only client retry**, default attempts 1→3, backoff 200ms→2s cap,
  `Retry-After` honored via `retry.WithDelayHint` (min(max(hint, jitter),
  MaxDelay)). Non-GET verbs byte-for-byte untouched. Exhaustion sentinel:
  `requests.ErrServiceUnavailable`.
- **`/metrics` fleet-wide** via `server.New()` seeding `routeInitializers`
  with `MountHandler("/metrics", promhttp.Handler())`; the 4 explicit mounts
  removed in the same task to avoid duplicate routes.
- **Env knobs:** `DB_ACQUIRE_RETRY_ATTEMPTS=3` (≤1 disables, wrapper returns
  base connector), `DB_ACQUIRE_RETRY_INITIAL_DELAY=100ms`,
  `DB_ACQUIRE_RETRY_MAX_DELAY=400ms`. `Retry-After: 1`
  (`server.TransientRetryAfterSeconds`).
- **Metric names fixed** (process-level, never tenant/entity labels):
  `atlas_db_acquire_retries_total{sqlstate}`,
  `atlas_db_transient_errors_total{sqlstate}`,
  `atlas_rest_client_retries_total{reason}`,
  `atlas_enrichment_degraded_total{component}`, `go_sql_*{db_name}` via stock
  `collectors.NewDBStatsCollector`.
- **DEVIATION from design.md:** atlas-login has **no database** (no
  `database.Connect` in its main.go), so design §2.3's note that login
  registers the classifier is wrong. Login only gets the decorator fix;
  atlas-inventory is the classifier-registration reference.
- prometheus version: `client_golang v1.23.2` (matches atlas-lock/atlas-seeder).

## Dependency/DAG notes for execution ordering

- Task 6 (client retry) needs Task 1 (`WithDelayHint`).
- Task 4 (connector) needs Tasks 2+3 and adds the `atlas-retry` require +
  `replace ../atlas-retry` to `libs/atlas-database/go.mod`.
- Task 10 needs Tasks 6, 8, 9. Task 11 needs Tasks 2, 3, 5. Task 12 needs 8, 9.
- Tasks 1, 2, 5, 8 have no intra-task dependencies and can start immediately.
- Task 14 (tidy + bake) must be last: the lib dep additions (prometheus into
  atlas-rest and atlas-database, pgx direct, atlas-retry) ripple into nearly
  every service's `go.sum`, and docker bake resolves from per-service
  go.mod/go.sum — so the sweep + `docker buildx bake all-go-services` is the
  only honest full verification.

## Gotchas discovered during planning

- `pgconn.ConnectError.err` is **unexported** and `Error()` derefs `Config` —
  you cannot construct one literally. Tests obtain a real one by
  `pgconn.Connect(ctx, "postgres://user:pass@127.0.0.1:1/db")` (closed port).
- `database.try()` returns **nil** (not the error) when the closure says
  don't-continue; that's why `Connect`'s DSN-parse failure is handled outside
  the closure (Fatalf) rather than returned with `(false, err)`.
- `retry.Try` uses `attempt <= MaxRetries`, so today's client default
  `retries: 1` = one attempt, zero retries — even transport errors are not
  retried on GET before this task.
- `gorm.Open` pings by default (`DisableAutomaticPing=false`), so the
  bootstrap `try(..., 10)` loop still works after the connector swap, and the
  ping itself exercises the retry connector.
- The retry counter and `errServiceUnavailableAttempt` sentinel survive
  `retry.Try`'s `%w` wrapping and the delay-hint wrapper because both
  implement/preserve `Unwrap` — `errors.Is` works end-to-end.
- Duplicate Prometheus registration (tests, repeated `Connect`) must not
  panic: `registerDBStats` uses `Register` + Warn, not `MustRegister`.
- api2go fixture marshaling: build JSON:API response bodies in tests with
  `jsonapi.Marshal(restModel)` (`github.com/jtumidanski/api2go/jsonapi`).
- atlas-login character tests are **in-package** (`package character`) so they
  can build `Model{id: 42}` and inspect the un-exported inventory field via
  `reflect.DeepEqual` against the original model.
- Never `go work sync`; `go mod tidy` per-module only, after imports exist.
  `go mod tidy` does NOT add `replace` directives — if a service module fails
  on the new transitive `atlas-retry`/pgx requirements, copy the replace line
  shape already present in that go.mod (both atlas-login and atlas-inventory
  already have the atlas-retry replace).
- `tools/redis-key-guard.sh` runs from the repo root (worktree root here),
  without a global `GOWORK=off`.

## Verification battery (root CLAUDE.md, mandatory before "done")

1. `go test -race ./...`, `go vet ./...`, `go build ./...` in every changed
   module (4 libs + all touched services).
2. `docker buildx bake all-go-services` (the tidy sweep touches ~all service
   go.mods, so bake everything).
3. `tools/redis-key-guard.sh` clean.
4. Code review (`superpowers:requesting-code-review`) before any PR.

# atlas-database

Shared GORM/postgres connection layer with multi-tenant scoping, transient
error classification, and acquire-phase retry.

## Connection

`database.Connect(l, configurators...)` builds the DSN from env, opens the
pool through a retrying `driver.Connector` (pgx stdlib), registers tenant
callbacks, runs migrations, and registers `go_sql_*` DBStats gauges.

## Environment knobs

| Env | Default | Meaning |
|---|---|---|
| `DB_USER` / `DB_PASSWORD` / `DB_HOST` / `DB_PORT` / `DB_NAME` | — | DSN components |
| `DB_MAX_OPEN_CONNS` | `10` | pool max open connections |
| `DB_MAX_IDLE_CONNS` | `5` | pool max idle connections |
| `DB_CONN_MAX_LIFETIME` | `5m` | recycle connections after this age |
| `DB_CONN_MAX_IDLE_TIME` | `3m` | close idle connections after this |
| `DB_ACQUIRE_RETRY_ATTEMPTS` | `3` | total acquire attempts; `0`/`1` disables retry |
| `DB_ACQUIRE_RETRY_INITIAL_DELAY` | `100ms` | retry backoff initial delay (full jitter) |
| `DB_ACQUIRE_RETRY_MAX_DELAY` | `400ms` | retry backoff cap |

Pool budgeting: `max_open × replicas × namespaces` summed over services must
fit postgres `max_connections` minus reserved slots.

## Transient classification

`IsTransientConnectionError(err) bool` — true only for acquire-phase failures:

| Condition | Transient |
|---|---|
| SQLSTATE `53300` (too_many_connections) | yes |
| SQLSTATE `57P03` (cannot_connect_now) | yes |
| SQLSTATE `08001` / `08006` (connect failure) | yes |
| `*pgconn.ConnectError` (connect never completed) | yes |
| net dial errors / ECONNREFUSED / ECONNRESET | yes |
| any other SQLSTATE (23xxx, 40001, 57014, ...) | no |
| bare `context.DeadlineExceeded`, `gorm.ErrRecordNotFound`, nil | no |

Coded errors are classified strictly by SQLSTATE (checked before the
connect-error shape), so an auth failure during connect is not transient.

`TransientSQLState(err) string` returns the classifying SQLSTATE ("" for
dial-shape). `CountTransient(err)` increments
`atlas_db_transient_errors_total{sqlstate}` — call only after the predicate
returned true.

## Retry behavior

Retry wraps `driver.Connector.Connect` ONLY — the pool calls it before any
SQL is sent, so retried work can never double-apply. Mid-statement errors
are structurally unreachable from this layer. Each retry logs Warn and
increments `atlas_db_acquire_retries_total{sqlstate}`.

## Metrics

`go_sql_*{db_name}` DBStats gauges, `atlas_db_acquire_retries_total{sqlstate}`,
`atlas_db_transient_errors_total{sqlstate}`. Process-level; never
tenant-labeled. Exposed via the rest-server Builder's automatic `/metrics`
mount.

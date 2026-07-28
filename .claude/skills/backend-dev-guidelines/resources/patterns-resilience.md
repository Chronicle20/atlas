# DB & Downstream Resilience Patterns

Source task: task-168 (atlas-pr-901 naked-character incident). These patterns
are mandatory for new code and enforced by DOM-27/DOM-28.

## Transient DB error classification

`libs/atlas-database` exports:

- `database.IsTransientConnectionError(err error) bool` — true only for
  acquire-phase failures: SQLSTATE 53300, 57P03, 08001, 08006, pgx
  `*pgconn.ConnectError`, dial-shape net errors (ECONNREFUSED/ECONNRESET).
  Anything that may have started executing (constraint violations,
  serialization failures, statement timeouts, bare context deadlines) is
  NOT transient. Never retry ambiguous work.
- `database.TransientSQLState(err) string` — metric label helper.
- `database.CountTransient(err)` — increments
  `atlas_db_transient_errors_total{sqlstate}`; call only after the
  predicate returned true.

## Acquire-phase DB retry (automatic)

`database.Connect` wraps the pgx connector so transient acquire failures are
retried transparently: max `DB_ACQUIRE_RETRY_ATTEMPTS` (default 3) attempts,
full-jitter backoff `DB_ACQUIRE_RETRY_INITIAL_DELAY` (100ms) →
`DB_ACQUIRE_RETRY_MAX_DELAY` (400ms). `0`/`1` disables. Every retry logs Warn
and increments `atlas_db_acquire_retries_total{sqlstate}`. Mid-statement
errors never reach this layer (the wrapper sits on `driver.Connector.Connect`,
which the pool only calls before any SQL is sent).

## The 503 transient-error contract (server side)

Transient DB errors MUST surface as `503 Service Unavailable` +
`Retry-After: 1` with a JSON:API error body — never a generic 500. In
handlers, replace `w.WriteHeader(http.StatusInternalServerError)` with:

    server.WriteErrorResponse(d.Logger())(w)(err)

and register the classifier once in main.go (services with a DB):

    server.RegisterTransientErrorClassifier(func(err error) bool {
        if database.IsTransientConnectionError(err) {
            database.CountTransient(err)
            return true
        }
        return false
    })

Keep 404/400 branches as they are. Non-transient errors still map to 500
(now with a JSON:API body). Reference implementation: atlas-inventory
(main.go + inventory/compartment/asset resource.go). As of task-168 this is
adopted by **every DB-backed service** — see
`docs/tasks/task-168-db-connection-resilience/fleet-503-adoption.md` for the
full list and the handful of non-mechanical sites. New DB-backed services MUST
follow it (DOM-27).

## Client retry semantics (automatic, GET only)

The shared REST client (`libs/atlas-rest/requests`) retries GETs on 503
(and transport errors) — 3 attempts default, jittered backoff capped at 2s,
`Retry-After` honored (capped). Exhaustion returns
`requests.ErrServiceUnavailable` (check with `errors.Is`). POST/PATCH/PUT/
DELETE are never retried on 503. Do not add per-call retry loops around the
client; if a GET must not retry, pass `requests.SetRetries(1)`.

## No silent degradation (decorator policy)

A decorator or enrichment step that fails its fetch MUST NOT silently return
the un-enriched model. Use the combinator + observer pair:

    func (p *ProcessorImpl) XDecorator() model.Decorator[Model] {
        return model.ErrDecorator(
            func(m Model) (Model, error) {
                x, err := p.dep.GetById(m.Id())
                if err != nil { return m, err }
                return m.SetX(x), nil
            },
            func(m Model, err error) {
                degrade.Observe(p.l, "<svc>.<domain>.<enrichment>", m.Id(), err)
            },
        )
    }

Degrading (returning the un-enriched model) remains the correct fallback —
but it logs Warn with the entity id and increments
`atlas_enrichment_degraded_total{component}`. Component strings are static
and low-cardinality; entity ids go in the log line only. Reference:
atlas-login `character.InventoryDecorator`.

## Pool sizing guidance

Defaults: `DB_MAX_OPEN_CONNS=10`, `DB_MAX_IDLE_CONNS=5`,
`DB_CONN_MAX_LIFETIME=5m`, `DB_CONN_MAX_IDLE_TIME=3m`. Budget rule: the sum
over all DB services of `max_open × replicas × namespaces` must fit inside
postgres `max_connections` minus reserved slots — dozens of services ×
multiple ephemeral namespaces WILL exhaust slots under burst if left
unbudgeted. Watch `go_sql_wait_count_total` / `go_sql_wait_duration_seconds_total`
and `atlas_db_acquire_retries_total` for pressure before it bites.
Infrastructure-side budgets (postgres max_connections, PgBouncer) are infra
concerns — the service-side knob is `DB_MAX_OPEN_CONNS`.

## Observability summary

| Metric | Meaning |
|---|---|
| `go_sql_*{db_name}` | pool gauges (open/in-use/idle/wait) per service |
| `atlas_db_acquire_retries_total{sqlstate}` | DB-side transparent retries — rising = chronic undersizing |
| `atlas_db_transient_errors_total{sqlstate}` | transient classifications (retried or surfaced) |
| `atlas_rest_client_retries_total{reason}` | client-side 503 retries |
| `atlas_enrichment_degraded_total{component}` | loud degradations — should be ~0 |

Every REST-serving service exposes `/metrics` automatically (mounted by the
rest-server Builder); no per-service mount is needed.

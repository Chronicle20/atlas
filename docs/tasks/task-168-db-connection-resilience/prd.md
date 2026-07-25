# DB Connection Resilience — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-07-12
---

## 1. Overview

On 2026-07-12 in ephemeral env `atlas-pr-901`, a freshly created character rendered naked in
character-select despite being fully equipped in the database. The root cause chain: character
creation's DB burst momentarily exhausted the shared postgres connection pool
(`SQLSTATE 53300 — remaining connection slots are reserved`), atlas-inventory returned a generic
500, atlas-login's single-attempt inventory fetch failed, and `InventoryDecorator`
(`services/atlas-login/atlas.com/login/character/processor.go:108-116`) silently swallowed the
error — pushing a character entry with no equipment. The failure was transient (2 error lines,
self-healed within seconds) but the user-visible corruption was silent and misleading.

The incident exposed four cross-cutting gaps, none specific to atlas-login or atlas-inventory:

1. **No pool budgeting.** All ~58 Go modules connect through `libs/atlas-database/connection.go`
   with defaults of 10 open / 5 idle connections per service; no deployment sets
   `DB_MAX_OPEN_CONNS`. Dozens of services × multiple ephemeral namespaces against one shared
   postgres makes slot exhaustion inevitable under burst.
2. **No transient-error contract.** A retryable condition (pool exhaustion) surfaces as a generic
   500, indistinguishable from a real bug, so no caller can make an informed retry decision.
3. **No response-level retry.** The shared REST client (`libs/atlas-rest/requests/get.go`)
   retries only transport errors; any 5xx response is terminal on the first attempt
   (default `retries: 1`).
4. **Silent degradation.** `model.Decorator[M]` has signature `func(M) M` — no error channel —
   so every decorator that fetches remote data can only drop it silently on failure. Nothing
   logs, nothing counts, and the degraded result is indistinguishable from a correct one.

This task hardens the shared libraries so all services inherit the fixes, establishes the
transient-error contract as a documented pattern for future implementations, and audits existing
silent-degrade call sites. Infrastructure changes (postgres `max_connections`, PgBouncer,
namespace budgets) are explicitly out of scope.

## 2. Goals

Primary goals:

- **Prevent** connection exhaustion from being amplified by the application layer: explicit,
  documented per-service pool budgets; limited, safe, acquire-phase-only DB retry.
- **Contract**: transient DB errors surface as `503 Service Unavailable` (+ `Retry-After`)
  instead of generic 500, distinguishing "retry me" from "I'm broken".
- **Recover**: the shared REST client retries idempotent requests on 503 with jittered backoff.
- **Honesty**: no decorator or fallback path degrades silently — every degradation logs at
  Warn-or-above and increments a metric; all existing decorator call sites audited.
- **Observe**: pool saturation, DB retries, and degraded responses are visible in Prometheus.
- **Document**: the patterns above land in the repo's agentic documentation
  (`.claude/skills/backend-dev-guidelines/`, reviewer checklist) so new implementations follow
  them by default.

Non-goals:

- Postgres server configuration (`max_connections`), PgBouncer/pooler deployment, per-namespace
  connection budgets, or any kustomize/infra manifest changes (out of scope per scoping decision;
  pool-sizing *guidance* is documented, applying it to deployments is not this task).
- Rearchitecting `model.Decorator`'s `func(M) M` signature across the codebase.
- Retry on 5xx codes other than 503, or retry of non-idempotent methods.
- Resilience for non-Postgres resources (Redis, Kafka) — existing mechanisms unchanged.
- Any change to the character-creation saga itself (it completed correctly in the incident).
- The task-126 branch (ruled out as cause).

## 3. User Stories

- As a player, I want the character-select screen to show my character's real equipment even when
  the database is momentarily saturated, so that I don't think my items were lost.
- As a service developer, I want a single shared predicate that tells me whether a DB error is
  transient, so that I map it to 503 without reimplementing SQLSTATE classification.
- As a service developer, I want inter-service GETs to automatically ride out a brief 503 from a
  dependency, so that momentary DB pressure doesn't cascade into user-visible failures.
- As an operator, I want pool-saturation and degraded-response metrics, so that I can see
  connection pressure building before (and when) it bites, instead of diagnosing from two FATAL
  log lines after the fact.
- As a reviewer (human or agent), I want the transient-error and no-silent-degrade patterns in
  the backend guidelines, so that new code is checked against them automatically.

## 4. Functional Requirements

### FR-1: Transient DB error classification (`libs/atlas-database`)

- FR-1.1: Export a predicate `IsTransientConnectionError(err error) bool` (name finalized in
  design) that returns true for acquire-phase connection failures:
  - `SQLSTATE 53300` (too_many_connections / reserved slots)
  - `SQLSTATE 57P03` (cannot_connect_now)
  - `SQLSTATE 08001` / `08006` (connection establishment/failure), driver dial errors
    (connection refused/reset, i/o timeout on connect)
  - `database/sql` pool-acquire context deadline (`ErrConnDone`-class conditions as identified
    in design).
- FR-1.2: The predicate MUST return false for any error produced after a statement began
  executing (constraint violations, serialization failures, statement timeouts, mid-query
  connection drops). Ambiguous-completion errors are NOT transient for retry purposes.
- FR-1.3: Classification is table-driven and unit-tested per SQLSTATE with real
  `pgconn.PgError` / driver error shapes, not string matching on full messages (message-prefix
  matching acceptable only where the driver exposes no code).

### FR-2: Limited acquire-phase DB retry (`libs/atlas-database`)

- FR-2.1: Connection-acquire failures classified transient by FR-1 are retried transparently:
  max 3 total attempts, jittered exponential backoff via `libs/atlas-retry`, initial delay
  ~100ms, max cumulative added latency ≤ ~1s (exact budget fixed in design).
- FR-2.2: Retry applies ONLY to the acquire phase (before any SQL is sent). Errors returned
  after statement execution begins are never retried by this layer (protects against
  double-applied writes on ambiguous COMMIT).
- FR-2.3: Every retry increments a Prometheus counter (FR-6) and logs at Warn with the SQLSTATE,
  so transparent retry cannot mask chronic undersizing.
- FR-2.4: Retry budget is env-tunable (`DB_ACQUIRE_RETRY_*`, names finalized in design) with the
  defaults above; setting retries to 0 disables the behavior.

### FR-3: 503 transient-error contract (`libs/atlas-rest` server side)

- FR-3.1: A shared helper maps an error to an HTTP status such that FR-1-transient errors
  produce `503 Service Unavailable` with a `Retry-After` header (small integer seconds; value
  fixed in design); all other unexpected errors keep their current mapping (500).
- FR-3.2: The helper is adopted in the resource handlers of the services on the incident path —
  at minimum atlas-inventory's inventory-by-character endpoint — as the reference
  implementation. Full-fleet handler adoption is driven by the documented pattern (FR-7) and
  the reviewer checklist, not bulk-edited in this task unless design finds a single choke point
  (e.g. a shared server error-translation layer) that upgrades all services at once. If such a
  choke point exists, adopt it there.
- FR-3.3: A 503 response body still follows JSON:API error-object conventions.

### FR-4: REST client retry on 503 (`libs/atlas-rest/requests`)

- FR-4.1: The shared GET path treats a 503 response as a retryable attempt: retried with the
  existing `atlas-retry` backoff+jitter, honoring `Retry-After` when present (capped at the
  configured max delay).
- FR-4.2: Retry-on-503 applies to GET only in this task. POST/PATCH are never retried on 503.
  (PUT/DELETE may be included if design confirms all in-repo usages are idempotent; default is
  GET-only.)
- FR-4.3: No other status code becomes retryable (5xx other than 503, 4xx unchanged). Transport
  -error retry behavior is unchanged.
- FR-4.4: Effective attempt count for a 503-retried GET is bounded (align with the existing
  `retries` configurator; design decides whether the default changes from 1) and total added
  latency is bounded so login-path callers cannot hang user-visible flows for more than a small
  number of seconds.
- FR-4.5: Client-side 503 retries increment a metric or emit a structured Warn log (design picks
  based on what the client layer can access).

### FR-5: No silent degradation — decorator policy + audit

- FR-5.1: Policy (documented in FR-7): a decorator or enrichment step that fails to fetch data
  MUST (a) first retry via the shared client behavior (FR-4 gives this for free), and on final
  failure (b) log at Warn or above with character/entity id and cause, and (c) increment a
  degradation metric. Returning the un-enriched model remains the accepted fallback (option C:
  degrade loudly; do not fail the flow).
- FR-5.2: `atlas-login` `InventoryDecorator` (`character/processor.go:108-116`) is upgraded to
  the policy as the reference implementation.
- FR-5.3: Audit ALL `model.Decorator` implementations across `services/` that perform fallible
  fetches (REST/DB/Redis) and currently swallow the error. Deliverable: an audit table in the
  task folder (`decorator-audit.md`) listing every site (file:line), whether it degrades
  silently, and its disposition; every silent site is either fixed to the policy in this task or
  listed with an explicit justification for why degradation is correct AND still gains the
  log+metric. No site may remain silent.
- FR-5.4: The audit also covers non-decorator silent-degrade shapes on the character-select
  path (e.g. seed-event handlers that build entries from partial data), flagged in the same
  table.

### FR-6: Observability

- FR-6.1: `libs/atlas-database` exposes `sql.DBStats`-derived Prometheus gauges per service:
  open connections, in-use, idle, wait count, wait duration, max-open setting. Exposition
  mechanism follows the existing Prometheus usage pattern in the repo (confirmed present in
  e.g. `libs/atlas-lock`, `libs/atlas-seeder`; design confirms the registration/exposition
  path for services that lack a metrics endpoint).
- FR-6.2: Counters: DB acquire retries (FR-2.3), transient-errors-classified (by SQLSTATE),
  decorator degradations (FR-5.1c), client 503 retries (FR-4.5) — exact metric names/labels
  fixed in design; labels MUST NOT explode cardinality (no per-character labels).
- FR-6.3: Metrics are tenant-agnostic (process-level); no tenant id in labels.

### FR-7: Documentation & agentic-guidelines updates

- FR-7.1: `.claude/skills/backend-dev-guidelines/` gains a "DB & downstream resilience" section
  covering: the FR-1 predicate, the 503 contract (map transient → 503 + Retry-After; never
  generic 500 for transient), client retry semantics (503 + GET only), the no-silent-degrade
  decorator policy, and pool-sizing guidance (how to reason about `DB_MAX_OPEN_CONNS` against a
  shared postgres; defaults documented).
- FR-7.2: `.claude/agents/backend-guidelines-reviewer.md` gains corresponding DOM-* checklist
  item(s) so the reviewer agent enforces: transient DB errors mapped to 503, and no silent
  degradation in decorators/enrichment paths.
- FR-7.3: `libs/atlas-database/README` (or CLAUDE.md within the lib, matching existing lib doc
  conventions) documents the retry behavior, env knobs, and classification table.
- FR-7.4: Root `CLAUDE.md` is updated only if design concludes a build/verification step changed
  (not expected); the pattern documentation itself lives in the skill per FR-7.1.

## 5. API Surface

No new endpoints. Changed behavior on existing endpoints:

- **All JSON:API resource endpoints (adopting services):** transient DB failures return
  `503 Service Unavailable` with `Retry-After: <seconds>` and a JSON:API error object
  (`errors[0].status = "503"`, `title` = stable machine-readable slug, e.g.
  `"temporarily unavailable"`), instead of 500. Non-transient failures keep returning 500.
- **Shared REST client (`libs/atlas-rest/requests`):** GETs transparently retry 503 responses;
  callers see either eventual success or the terminal error after bounded attempts. New exported
  error sentinel for exhausted-503 (design decides, e.g. `ErrServiceUnavailable`) so callers can
  distinguish it from `ErrBadRequest`/`ErrNotFound`.
- New exported predicate in `libs/atlas-database` (FR-1.1) and env knobs
  (`DB_ACQUIRE_RETRY_*`; existing `DB_MAX_OPEN_CONNS`/`DB_MAX_IDLE_CONNS`/
  `DB_CONN_MAX_LIFETIME`/`DB_CONN_MAX_IDLE_TIME` unchanged but now documented).

## 6. Data Model

No schema changes, no migrations, no new entities. Multi-tenancy untouched (classification and
retry operate below the tenant layer; metrics are process-level per FR-6.3).

## 7. Service Impact

| Area | Change |
|------|--------|
| `libs/atlas-database` | Transient classifier (FR-1), acquire-phase retry (FR-2), DBStats metrics (FR-6.1), docs (FR-7.3) |
| `libs/atlas-rest` | Server-side 503 mapping helper (FR-3), client GET retry-on-503 (FR-4) |
| `libs/atlas-retry` | Reused as-is (no changes expected; `Retry-After` honoring may add a small hook if needed) |
| `services/atlas-login` | `InventoryDecorator` reference fix (FR-5.2); consumes client retry automatically |
| `services/atlas-inventory` | Adopts 503 mapping on inventory endpoints (FR-3.2 reference) |
| All other Go services | Inherit client retry + DB retry + metrics via lib bump; decorator audit touches those with silent-degrade sites (FR-5.3 table drives the list) |
| `.claude/skills/`, `.claude/agents/` | Guidelines + reviewer checklist updates (FR-7) |

Every service whose `go.mod` is touched requires `docker buildx bake atlas-<svc>` per the root
CLAUDE.md verification rules; since `libs/atlas-database` and `libs/atlas-rest` are already in
the shared Dockerfile COPY list, no Dockerfile changes are expected.

## 8. Non-Functional Requirements

- **Latency bounds:** combined worst-case added latency on a user-visible path (DB acquire retry
  + client 503 retry) MUST stay under ~5s; individual budgets set in design so the login/char-
  select flow cannot hang the client socket past its own timeouts.
- **No retry storms:** all retries jittered (full jitter, already in `atlas-retry`); attempt
  counts hard-capped; DB-side retry disabled-able via env.
- **Safety:** no retry of any operation whose completion state is ambiguous (FR-2.2). This is
  the load-bearing safety property; it gets explicit unit tests.
- **Backward compatibility:** services that don't adopt the 503 helper keep current behavior;
  the client treats their 500s as terminal exactly as today. Lib changes must not alter any
  existing exported signature (additive only).
- **Performance:** classification is O(1) per error; metrics collection uses the standard
  `sql.DBStats` snapshot (no extra queries).
- **Multi-tenancy:** no tenant-scoped behavior differences; no tenant ids in metrics or logs
  beyond existing logging conventions.
- **Testing:** unit tests per SQLSTATE for the classifier; retry-path tests with fake drivers/
  round-trippers (Builder-pattern test setup per project convention, no `*_testhelpers.go`);
  byte-for-byte identical behavior for non-503 statuses in the client.

## 9. Open Questions

1. Does a single server-side error-translation choke point exist in `libs/atlas-rest/server`
   through which all JSON:API handlers already flow? If yes, FR-3.2 becomes a fleet-wide upgrade
   in one edit; if no, adoption is per-service via the documented pattern. (Design phase
   answers.)
2. Do all services already expose a Prometheus scrape endpoint, or does FR-6 need to add
   exposition plumbing in `libs/atlas-rest/server` (or the bootstrap lib from task-118)?
3. Should PUT/DELETE join GET in 503 retry (idempotent per HTTP semantics) — depends on an
   in-repo usage audit during design.
4. Exact scale of FR-5.3: 229 files reference `model.Decorator`; the audit must separate
   fallible-fetch decorators from pure in-memory ones. Effort is bounded by producing the table
   first, then fixing the silent subset.
5. Whether the default client `retries: 1` changes globally or only 503 handling gains attempts
   (interaction with existing transport-error retry semantics).

## 10. Acceptance Criteria

- [ ] `IsTransientConnectionError` exists in `libs/atlas-database`, table-driven, with unit
      tests covering 53300, 57P03, 08001/08006, dial errors, and negative cases (constraint
      violation, statement timeout, mid-query drop) — negatives MUST classify false.
- [ ] Acquire-phase DB retry: a simulated 53300 on acquire succeeds on a later attempt within
      the latency budget; a mid-statement error is provably NOT retried (test asserts single
      execution); retry counter increments; env knobs honored; retries=0 disables.
- [ ] atlas-inventory returns 503 + `Retry-After` + JSON:API error body for a transient DB
      failure (unit/integration test with injected classifier-true error), 500 for
      non-transient.
- [ ] Shared client GET retries a 503 (test: round-tripper returns 503 then 200 → caller sees
      200; 503×N → terminal sentinel error), honors `Retry-After`, never retries POST/PATCH on
      503, and treats all other statuses exactly as before (regression tests).
- [ ] `InventoryDecorator` on final inventory-fetch failure logs Warn with character id and
      increments the degradation metric; behavior verified by test.
- [ ] `decorator-audit.md` exists in the task folder listing every fallible-fetch decorator
      site with disposition; zero sites remain silent (each fixed or justified-with-logging).
- [ ] DBStats gauges + the four counters from FR-6.2 are registered and observable (test or
      local scrape evidence).
- [ ] `.claude/skills/backend-dev-guidelines/` and `.claude/agents/backend-guidelines-reviewer.md`
      contain the new resilience section / checklist items.
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...` clean in every changed module;
      `docker buildx bake` for every service whose `go.mod` changed; `tools/redis-key-guard.sh`
      clean.
- [ ] Incident-path replay: with atlas-inventory forced to return 503 once, a character-create →
      char-select flow in a test (or documented manual verification) produces a fully equipped
      entry (client retry absorbs the blip) — the original symptom cannot reproduce from a
      single transient failure.

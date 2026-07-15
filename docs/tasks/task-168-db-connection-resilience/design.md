# DB Connection Resilience — Design

Task: task-168-db-connection-resilience
Status: Approved PRD → Design
Date: 2026-07-12

---

## 1. Context

The PRD (prd.md) is approved. This design fixes the architecture for the four cross-cutting
gaps exposed by the atlas-pr-901 naked-character incident: no pool budgeting visibility, no
transient-error contract, no response-level retry, and silent decorator degradation.

Design-phase code survey results that drive the decisions below:

- `libs/atlas-database/connection.go` — `Connect()` opens gorm directly via
  `gorm.Open(postgres.Open(dsn))`; pool knobs (`DB_MAX_OPEN_CONNS` etc.) already exist but are
  undocumented. `gorm.io/driver/postgres v1.6.0`, `gorm.io/gorm v1.31.2`, `pgx/v5 v5.7.4`
  (currently indirect).
- `libs/atlas-rest/requests/get.go` — default `retries: 1` passed to `retry.Try` whose loop is
  `attempt <= MaxRetries`, so **a GET today makes exactly one attempt; even transport errors are
  not retried by default**. 5xx statuses fall through to `errors.New("unknown error")`.
- `libs/atlas-rest/server` — **no error-translation choke point exists** (PRD Open Question 1
  answered: NO). Every resource handler writes `w.WriteHeader(http.StatusInternalServerError)`
  inline (e.g. `services/atlas-inventory/.../inventory/resource.go:45`).
- Metrics exposition (Open Question 2): only 4 services mount `/metrics`
  (atlas-channel, atlas-summons, atlas-doors, atlas-monsters), each via
  `AddRouteInitializer(restserver.MountHandler("/metrics", promhttp.Handler()))`. The rest
  server `Builder` in `libs/atlas-rest/server/server.go` is the natural choke point to mount it
  fleet-wide.
- `libs/atlas-retry/retry.go` — `Try(ctx, cfg, fn)` with full jitter; no per-attempt delay
  override hook (needed for `Retry-After`).
- `libs/atlas-model/model/processor.go:101` — `type Decorator[M any] func(M) M`; the module is
  dependency-free (`golang.org/x/sync` only). Keep it that way.
- Prometheus precedent: `libs/atlas-lock/metrics.go` and `libs/atlas-seeder/metrics.go` use
  `promauto.NewCounterVec` with `atlas_<domain>_*` naming against the default registry.

Existing-lib audit (per project rule "audit existing libs before a new one"): every change in
this design lands in an existing module — `atlas-database`, `atlas-rest`, `atlas-retry`,
`atlas-model`. **No new lib module is created**, so no Dockerfile/`go.work` edits.

---

## 2. Component Design

### 2.1 FR-1 — Transient classifier (`libs/atlas-database/transient.go`)

**Exported API:**

```go
// IsTransientConnectionError reports whether err is a connection-acquire-phase
// failure that is safe to retry (no statement was ever sent).
func IsTransientConnectionError(err error) bool
// TransientSQLState returns the SQLSTATE that classified err transient ("" if
// classified by dial-error shape). Used for metric labels.
func TransientSQLState(err error) string
```

**Classification table (table-driven, in order):**

| Condition | Detection | Transient |
|---|---|---|
| SQLSTATE `53300` (too_many_connections) | `errors.As → *pgconn.PgError`, `.Code` | yes |
| SQLSTATE `57P03` (cannot_connect_now) | same | yes |
| SQLSTATE `08001` / `08006` (connect failure) | same | yes |
| pgx connect-phase failure | `errors.As → *pgconn.ConnectError` (wraps any error raised during connection establishment, incl. dial refused/reset, i/o timeout on connect, and server startup errors) | yes |
| net dial errors | `errors.As → *net.OpError` with `.Op == "dial"`; `syscall.ECONNREFUSED`/`ECONNRESET` via `errors.Is` | yes |
| Everything else (constraint violations 23xxx, serialization 40001, statement timeout 57014, mid-query drops, `context.DeadlineExceeded`, `gorm.ErrRecordNotFound`, nil) | default | **no** |

Notes:

- FR-1.2 safety comes structurally from the table: any SQLSTATE produced after a statement
  begins executing is simply not in the allow-list. A **bare** `context.DeadlineExceeded` is NOT
  transient (it is ambiguous — could be mid-query); a deadline **inside** a
  `*pgconn.ConnectError` IS (connect never completed) — this resolves the FR-1.1
  "ErrConnDone-class" bullet: we classify by *error shape proving the acquire phase*, never by
  timing guesses.
- Tests construct real `&pgconn.PgError{Code: "53300"}` / `pgconn.ConnectError` values (pgx
  moves from indirect to direct dependency; it is already in the module graph, no new COPY
  lines).
- No full-message string matching. `pgconn` exposes codes for everything in the table, so the
  message-prefix escape hatch in FR-1.3 is not needed.

**Alternative considered:** classify via `errors.Is(err, driver.ErrBadConn)`. Rejected:
`database/sql` swallows/retries `ErrBadConn` internally and gorm never surfaces it reliably;
SQLSTATE + connect-error shape is precise and testable.

### 2.2 FR-2 — Acquire-phase retry (`libs/atlas-database/connector.go`)

**Chosen approach: retry inside a `driver.Connector` wrapper.**

`Connect()` changes from `gorm.Open(postgres.Open(dsn))` to:

```go
pgxCfg, _ := pgx.ParseConfig(dsn)                      // pgx/v5
base := stdlib.GetConnector(*pgxCfg)                   // pgx/v5/stdlib
sqlDB := sql.OpenDB(newRetryConnector(l, base))        // retry wrapper
db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{})
```

`retryConnector.Connect(ctx)` wraps `base.Connect(ctx)` in `retry.Try` with:
`IsTransientConnectionError(err)` → retry; anything else → fail immediately.

**Why this is the right layer (and the load-bearing safety argument):** `database/sql` calls
`Connector.Connect` *only* when the pool needs a new physical connection — before any SQL is
sent on it. Every error the wrapper sees is by construction acquire-phase. Server-side slot
exhaustion (`53300`, the incident error) is raised by postgres during connection startup, so it
surfaces exactly here. Mid-statement errors never pass through `Connect`, so FR-2.2
("never retry ambiguous work") is guaranteed by the type system, not by error-inspection
heuristics.

**Alternatives rejected:**

- *GORM callback wrapping every operation* — would see post-execution errors, forcing fragile
  "has the statement started?" inference; violates FR-2.2 by design.
- *Retry inside `database.Query`/`SliceQuery` helpers* — same ambiguity problem, plus misses
  raw `db.Where(...)` call sites (the majority).

**Budget & knobs** (FR-2.1/2.4), read once at `Connect()`:

| Env | Default | Meaning |
|---|---|---|
| `DB_ACQUIRE_RETRY_ATTEMPTS` | 3 | total attempts; `0` or `1` disables retry (wrapper passes through) |
| `DB_ACQUIRE_RETRY_INITIAL_DELAY` | `100ms` | atlas-retry initial delay |
| `DB_ACQUIRE_RETRY_MAX_DELAY` | `400ms` | atlas-retry max delay |

Worst-case added latency: full-jitter delays ≤ 100ms + 200ms(capped 400ms) ≈ **≤ 0.6s**, under
the ~1s FR-2.1 budget (failed `53300` connect attempts themselves are fast server rejections).

Each retry logs Warn with the SQLSTATE and increments `atlas_db_acquire_retries_total`
(§2.6). `Connect()`'s existing 10×1s bootstrap loop (`try`) is unchanged — it guards initial
service startup, a different concern.

### 2.3 FR-3 — Server-side 503 contract (`libs/atlas-rest/server/error.go`)

Open Question 1 is answered NO (no choke point), so per FR-3.2 adoption is per-handler via a
new shared helper, with atlas-inventory as the reference. To keep `atlas-rest` from importing
`atlas-database` (that would drag gorm+pgx into every REST consumer, including DB-less
services), classification is **injected once per process**:

```go
// server package
// RegisterTransientErrorClassifier installs the process-wide predicate used by
// WriteErrorResponse to map errors to 503. Typically called once from main.go:
//   server.RegisterTransientErrorClassifier(database.IsTransientConnectionError)
func RegisterTransientErrorClassifier(f func(error) bool)

// WriteErrorResponse maps err to a JSON:API error response:
//   classifier(err) == true → 503 + Retry-After: 1 + {"errors":[{"status":"503",
//     "title":"temporarily unavailable"}]}
//   otherwise               → 500 (current behavior, now with a JSON:API body)
func WriteErrorResponse(l logrus.FieldLogger) func(w http.ResponseWriter) func(err error)
```

- `Retry-After: 1` (seconds, fixed constant `TransientRetryAfterSeconds = 1`).
- Body follows JSON:API error-object conventions (FR-3.3) via a small local struct (api2go's
  `jsonapi` error types if they marshal cleanly, else a hand-rolled
  `{"errors":[{status,title}]}` — decided at implementation by what api2go exposes; the wire
  shape above is the contract).
- Services that never call `RegisterTransientErrorClassifier` get the nil-classifier default:
  everything stays 500 — backward compatible (NFR).
- Reference adoption: atlas-inventory `inventory/resource.go`, `compartment/resource.go`,
  `asset/resource.go` GET paths replace inline `w.WriteHeader(http.StatusInternalServerError)`
  with `server.WriteErrorResponse(...)`; atlas-inventory `main.go` registers the classifier.
  atlas-login registers it too (it has a DB). Fleet-wide handler adoption remains
  documentation-driven (FR-7), as the PRD prescribes.

**Alternative considered:** marker-interface (`interface{ Transient() bool }`) wrapped onto
errors by atlas-database. Rejected: gorm surfaces raw pgconn errors from dozens of paths; there
is no single wrap point short of a GORM callback on every operation, which reintroduces the
ambiguity problem of §2.2. Process-level registration is one line in main.go and testable.

### 2.4 FR-4 — Client GET retry on 503 (`libs/atlas-rest/requests/get.go`)

Changes confined to the GET path (`get.go`) plus one additive hook in `atlas-retry`:

1. Inside the attempt closure, after a successful transport exchange: if
   `statusCode == http.StatusServiceUnavailable`, log Warn, increment
   `atlas_rest_client_retries_total{reason="503"}`, and return
   `(true, retry.WithDelayHint(errServiceUnavailableAttempt, parsedRetryAfter))` so the
   attempt is retried. All other statuses return `(false, nil)` exactly as today —
   byte-for-byte identical handling (regression-tested).
2. After `retry.Try` returns: if attempts were exhausted with the 503 sentinel, return the new
   exported `requests.ErrServiceUnavailable` (sibling of `ErrBadRequest`/`ErrNotFound`) so
   callers can distinguish terminal saturation.
3. **Default attempts for GET change from 1 → 3** (Open Question 5 answered: yes, GET only).
   Today's default performs zero retries even on transport errors, which is exactly the
   single-attempt fragility from the incident. GETs in this codebase are reads of JSON:API
   resources — idempotent by construction. `SetRetries` remains the per-call override.
   POST (`post.go`), PATCH (`patch.go`), PUT (`put.go`), DELETE (`delete.go`) are untouched:
   default 1 attempt, 503 not retryable (Open Question 3 answered: **GET-only**; a repo grep
   found no PUT/DELETE usage pattern that would justify widening the safety envelope now —
   revisit only if a real need appears).
4. `Retry-After` honoring — additive `atlas-retry` hook (the "small hook" the PRD anticipated):

```go
// retry package
// WithDelayHint wraps err so Try waits at least d (capped at cfg.MaxDelay)
// before the next attempt, instead of the jittered backoff if that is smaller.
func WithDelayHint(err error, d time.Duration) error
```

   `Try` checks `errors.As` for the hint wrapper; delay = `min(max(hint, jittered), MaxDelay)`.
   No existing signature changes.

**Latency budget (NFR ~5s):** GET retry config becomes initial 200ms, **MaxDelay 2s** (down
from 5s), 3 attempts → worst added inter-attempt wait ≤ 2×2s = 4s (only when the server sends
`Retry-After` ≥ 2; jitter-only worst case ≤ 0.6s), plus per-attempt `c.timeout`. Combined with
§2.2's ≤0.6s DB-side budget this stays inside ~5s for the login path.

**Alternative considered:** sleeping `Retry-After` inside the attempt closure (no atlas-retry
change). Rejected: bypasses `ctx` cancellation during the sleep and hides the wait from the
retry layer's accounting; the hint hook is 15 lines and reusable.

### 2.5 FR-5 — Loud degradation: combinator + observer + audit

Two additive pieces, then the audit:

**(a) Pure combinator in `libs/atlas-model/model`** (module stays dependency-free):

```go
// ErrDecorator adapts a fallible enrichment into a Decorator. On error it
// invokes onErr (never nil) and returns m unchanged — degrade loudly, don't fail.
func ErrDecorator[M any](f func(M) (M, error), onErr func(M, error)) Decorator[M]
```

**(b) Observer in `libs/atlas-rest/degrade`** (new package in the existing atlas-rest module,
which every service already imports; it gains the prometheus dep anyway for §2.6):

```go
// Observe logs the degradation at Warn with the entity id and cause, and
// increments atlas_enrichment_degraded_total{component}.
func Observe(l logrus.FieldLogger, component string, entityId uint32, err error)
```

`component` is a low-cardinality static string (e.g. `"login.character.inventory"`); entity id
goes only into the log line, never a label (FR-6.2).

**(c) Reference fix — atlas-login `InventoryDecorator`
(`services/atlas-login/atlas.com/login/character/processor.go:108`):**

```go
func (p *ProcessorImpl) InventoryDecorator() model.Decorator[Model] {
    return model.ErrDecorator(
        func(m Model) (Model, error) {
            i, err := p.ip.GetByCharacterId(m.Id())
            if err != nil { return m, err }
            return m.SetInventory(i), nil
        },
        func(m Model, err error) { degrade.Observe(p.l, "login.character.inventory", m.Id(), err) },
    )
}
```

Retry-before-degrade (FR-5.1a) comes for free: `GetByCharacterId` rides the shared GET client,
which now retries 503s (§2.4).

**(d) Audit (FR-5.3/5.4)** — deliverable `decorator-audit.md` in the task folder. Method:

1. `grep -rn "model.Decorator\[" services/` → implementations (not the 229 mere references).
2. Classify each: **fallible fetch** (body calls a processor/requests/DB/Redis and branches on
   `err`) vs **pure** (in-memory transform).
3. Table columns: `service`, `file:line`, `decorator`, `fetch kind`, `silent today?`,
   `disposition` (fixed-in-task / justified-but-now-loud). Every silent fallible site is
   converted to the (a)+(b) pattern in this task; "justified" sites still gain
   `degrade.Observe`. Zero rows may end silent.
4. Same table gets a second section for non-decorator silent-degrade shapes on the
   character-select path (login seed/entry builders), found by tracing the char-select flow in
   atlas-login (FR-5.4).

The audit's fix-list defines which extra services get touched (and therefore baked) in the
implementation plan.

**Alternative considered:** changing `Decorator[M]` to `func(M) (M, error)`. Explicit PRD
non-goal; the combinator gives the same honesty without a 229-file refactor.

### 2.6 FR-6 — Observability

**Metric homes** (all `promauto` against the default registry, `atlas_*` naming per
`atlas-lock` precedent; process-level, no tenant labels):

| Metric | Type | Labels | Home |
|---|---|---|---|
| `go_sql_*` DBStats family (open/in-use/idle/wait count/wait duration/max-open) | gauges/counters | `db_name` | `libs/atlas-database` — register `collectors.NewDBStatsCollector(sqlDB, dbName)` (stock client_golang collector, snapshot-based, no extra queries) in `Connect()` |
| `atlas_db_acquire_retries_total` | counter | `sqlstate` | `libs/atlas-database/metrics.go` |
| `atlas_db_transient_errors_total` | counter | `sqlstate` | `libs/atlas-database/metrics.go` (incremented by the classifier's callers: connector retry + a public `CountTransient(err)` used by the 503-mapping registration) |
| `atlas_rest_client_retries_total` | counter | `reason` (`"503"`) | `libs/atlas-rest/requests/metrics.go` |
| `atlas_enrichment_degraded_total` | counter | `component` | `libs/atlas-rest/degrade` |

**Exposition (Open Question 2 answered):** the rest-server `Builder` in
`libs/atlas-rest/server` mounts `/metrics` → `promhttp.Handler()` unconditionally (same
`MountHandler` mechanism the 4 services use today). The 4 explicit per-service mounts are
removed in the same change to avoid duplicate route registration. This is the single choke
point that gives every REST-serving service an endpoint; Prometheus *scrape configuration* is
infra and stays out of scope per the PRD.

### 2.7 FR-7 — Documentation

- `.claude/skills/backend-dev-guidelines/` gains a **"DB & downstream resilience"** section:
  the classifier, the 503 + `Retry-After` contract (never generic 500 for transient), client
  retry semantics (503 + GET only, `ErrServiceUnavailable`), the loud-degrade decorator policy
  (`model.ErrDecorator` + `degrade.Observe`), and pool-sizing guidance
  (`DB_MAX_OPEN_CONNS` reasoning: sum of per-service max-open × replicas × namespaces must fit
  postgres `max_connections` minus reserved slots; defaults 10 open / 5 idle documented).
- `.claude/agents/backend-guidelines-reviewer.md` gains two checklist items (next free DOM-*
  numbers, verified at implementation time): (1) transient DB errors map to 503 via
  `server.WriteErrorResponse` + registered classifier, never bare 500; (2) no silent
  degradation — every fallible enrichment logs Warn + increments the degradation metric.
- `libs/atlas-database/README.md` (new, matching lib doc conventions): classification table,
  retry behavior, all `DB_*` env knobs including the pre-existing pool knobs.
- Root `CLAUDE.md`: no change (no build/verification step changed).

---

## 3. Data Flow (incident replay, post-fix)

```
char-create burst → postgres slots exhausted
  ├─ atlas-inventory: pool needs new conn → retryConnector.Connect → 53300
  │    → classified transient → jittered retry (≤3 attempts, ≤0.6s) → usually heals here
  │    → if still failing: handler → WriteErrorResponse → 503 + Retry-After: 1
  ├─ atlas-login GET inventory: 503 received → client retries (≤3 attempts, honors Retry-After)
  │    → usually heals here
  └─ if ALL retries exhausted: InventoryDecorator → degrade.Observe (Warn + metric)
       → character entry without equipment, but loudly — never silently
```

Every layer that absorbs a failure emits a counter, so "transparent" never means "invisible"
(FR-2.3): chronic undersizing shows up as a rising `atlas_db_acquire_retries_total` /
`atlas_rest_client_retries_total` even while users see no errors.

---

## 4. Testing Strategy

Per project conventions (Builder-pattern setup, no `*_testhelpers.go`):

- **Classifier** (`libs/atlas-database`): table-driven over real `&pgconn.PgError{Code: ...}`
  and `pgconn.ConnectError` values — positives 53300/57P03/08001/08006/dial; negatives 23505,
  40001, 57014, bare `context.DeadlineExceeded`, `gorm.ErrRecordNotFound`, nil.
- **Connector retry**: fake `driver.Connector` whose `Connect` fails N times with a transient
  error then succeeds → asserts attempt count, success, counter increment; a non-transient
  error asserts exactly 1 attempt; `DB_ACQUIRE_RETRY_ATTEMPTS=0` asserts pass-through. The
  "mid-statement error is not retried" property is asserted structurally: a fake driver conn
  whose `QueryContext` fails is exercised through `sql.OpenDB(wrapper)` and the test asserts
  single execution (the wrapper never sees it).
- **503 mapping** (`libs/atlas-rest/server`): `httptest.ResponseRecorder` — classifier-true
  error → 503 + `Retry-After: 1` + JSON:API body; classifier-false → 500; no classifier
  registered → 500.
- **Client retry** (`libs/atlas-rest/requests`): fake round-tripper (existing `client_test.go`
  precedent) — 503→200 yields 200 to the caller; 503×3 yields `ErrServiceUnavailable`;
  `Retry-After` respected and capped; POST/PATCH on 503 not retried; regression matrix
  asserting 200/202/400/404/500/other behave byte-for-byte as before.
- **`retry.WithDelayHint`** (`libs/atlas-retry`): hint respected, capped at MaxDelay, absent →
  jitter unchanged.
- **Decorator policy**: atlas-login test — inventory fetch error → model returned un-enriched,
  Warn logged with character id, counter incremented (via `prometheus/testutil`).
- **Incident replay** (acceptance): atlas-login test with a fake inventory dependency returning
  503 once then 200 → decorated character has full equipment.
- Full verification battery per root CLAUDE.md: `go test -race`, `go vet`, `go build` in every
  changed module; `docker buildx bake` for atlas-login, atlas-inventory, and every service the
  decorator audit touches; `tools/redis-key-guard.sh`.

---

## 5. Resolved Open Questions

| # | Question | Resolution |
|---|---|---|
| 1 | Server-side choke point? | **No** — handlers write statuses inline. New `server.WriteErrorResponse` helper + per-process classifier registration; reference adoption in atlas-inventory/atlas-login; fleet adoption via docs + reviewer checklist. |
| 2 | Prometheus exposition everywhere? | **No** (4 services only). Rest-server `Builder` mounts `/metrics` unconditionally; 4 explicit mounts removed. |
| 3 | PUT/DELETE join 503 retry? | **No** — GET-only. No in-repo usage justifies widening; revisit on demand. |
| 4 | FR-5.3 audit scale | Grep for `model.Decorator[` *implementations*, classify fallible vs pure; table drives the fix list. Bounded and produced before fixes. |
| 5 | Does default client `retries` change? | **GET default 1 → 3** (transport + 503). Non-GET verbs unchanged at 1. |

## 6. Risks & Mitigations

- **Connector swap changes DSN parsing path** (`postgres.Open(dsn)` → `pgx.ParseConfig` +
  `stdlib.GetConnector`): both parse libpq DSNs via pgx; the DSN builder output
  (`host= user= ... sslmode=disable`) is plain libpq syntax. Covered by every service's
  existing integration usage + bake verification.
- **`/metrics` auto-mount collides with the 4 explicit mounts**: removed in the same commit;
  gorilla/mux duplicate routes would otherwise shadow, not crash — still cleaned up.
- **Retry amplification under sustained outage**: all retries are jittered, hard-capped
  (3 DB-side × 3 client-side worst case = bounded, ~seconds), and DB-side is env-disableable.
  503 `Retry-After: 1` spreads client herd behind full jitter.
- **atlas-rest gains a prometheus dependency**: acceptable — every service already links
  client_golang transitively via other libs (lock/seeder) or its own main; and FR-6 requires
  exposition from the rest server anyway.

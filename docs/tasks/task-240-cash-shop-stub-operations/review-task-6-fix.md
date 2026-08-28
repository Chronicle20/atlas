# Re-review: Task 6 fix — bound backfill's boot-time query

Commit under review: `39de4adc1` (`fix(cashshop): bound backfill's boot-time query to a single SQL aggregate`)
Range given: `cb9eeb9..39de4adc1` (this range also contains `4871c8581`, an unrelated channel-side commit for a
different task; I scoped the review to `39de4adc1` itself, confirmed via `git show --stat 39de4adc1`).
Prior review: `docs/tasks/task-240-cash-shop-stub-operations/review-task-6.md` (verdict `CHANGES_REQUIRED`)
Brief: `.superpowers/sdd/plan/task-6-brief.md`
Report: `.superpowers/sdd/plan/task-6-report.md`

## Scope

`39de4adc1` touches exactly two files, both already in the prior review's surface:

```
services/atlas-cashshop/atlas.com/cashshop/purchaserecord/backfill.go      | 150 ++++++++--
services/atlas-cashshop/atlas.com/cashshop/purchaserecord/backfill_test.go |  32 ++-
```

`main.go` is untouched by this commit (confirmed by `git show --stat 39de4adc1` — no `main.go` line), matching
the report's claim that the sequencing constraint imposed on it ("don't touch `main.go`") was unnecessary since
`Backfill`'s signature (`func Backfill(l logrus.FieldLogger, db *gorm.DB) (int, error)`) is unchanged. Scope
matches the fix brief. No drift.

## Verified with tooling

```
cd services/atlas-cashshop/atlas.com/cashshop && go clean -testcache && go build ./... && go test ./...
```
All packages `ok`, including `atlas-cashshop/purchaserecord`. Re-ran targeted:
```
go test ./purchaserecord/... -run TestBackfill -v -count=1
```
`TestBackfill` (4 subtests), `TestBackfillCountsDuplicatesAndSoftDeletes` (+ `is_idempotent` subtest), and
`TestBackfillUsesSQLiteFallbackUnderTest` all PASS.

## Findings, against the six judged questions

### 1. Is the production path now bounded? — YES, closed

`backfillGroupsSQL` (`purchaserecord/backfill.go:119-129`) performs the grouping/aggregation entirely in SQL —
`Select(... COUNT(*) ... MIN(cash_assets.created_at) ... MAX(cash_assets.created_at) ...)`, `Joins`, `Where`,
`Group`, single `Scan(&groups)` — with no intermediate ungrouped Go slice. Result set is bounded to the number of
distinct `(tenant_id, account_id, commodity_id)` groups, matching the brief's original design. This is the branch
production actually takes (see #2). The old unbounded `assetRows []backfillAssetRow` load survives only inside
`backfillGroupsSQLite`, and that function is reachable only from the sqlite branch. Blocking finding #1 from the
prior review is closed.

### 2. Dialect branch selection — sound, cannot misfire

`backfill.go:70`: `if db.Dialector.Name() == "sqlite" { ... } else { backfillGroupsSQL(db) }`.

Checked both drivers' `Name()` implementations directly (not inferred):

- `gorm.io/driver/postgres@v1.6.2/postgres.go:50-52` — `func (dialector Dialector) Name() string { return "postgres" }` — a hardcoded literal, not derived from DSN, env var, or build tag.
- `gorm.io/driver/sqlite@v1.6.0/sqlite.go:41-43` — `func (dialector Dialector) Name() string { return "sqlite" }` — likewise hardcoded.

Traced production's actual driver: `services/atlas-cashshop/.../main.go:64` calls `database.Connect(...)`
(`github.com/Chronicle20/atlas/libs/atlas-database`), whose `Connect` (`libs/atlas-database/connection.go:125`)
unconditionally does `gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{})`. There is no code
path in this service that opens a sqlite `*gorm.DB` outside test code (`databasetest.NewInMemoryTenantDB`, used
only by `backfill_test.go` and other `_test.go` files). So: production always takes the `else` (SQL aggregate)
branch; the module's test suite always takes the `sqlite` (Go-side) branch. The selector is a compile-time-fixed
string per driver, not runtime-derived — there is no plausible path for production to fall into the unbounded
fallback, nor for tests to silently exercise the SQL-aggregate branch. Sound.

### 3. Are the two branches semantically identical? — YES

Compared `backfillGroupsSQL` (`backfill.go:119-129`) and `backfillGroupsSQLite` (`backfill.go:137-186`) field by
field:

| aspect | SQL path | sqlite path |
|---|---|---|
| soft-deleted inclusion | `db.Unscoped()` | `db.Unscoped()` |
| join | `JOIN cash_compartments ON cash_compartments.id = cash_assets.compartment_id` | identical |
| zero-commodity filter | `WHERE cash_assets.commodity_id <> 0` | identical |
| grouping key | `tenant_id, account_id, commodity_id` (SQL `GROUP BY`) | same three fields as Go map key (`backfillGroupKey`) |
| count | `COUNT(*)` | `a.count++` per row |
| first/last | `MIN`/`MAX(created_at)` | manual `Before`/`After` comparison seeded from the first row in the group |

Both feed the same `backfillGroupRow` struct into the same downstream `entity{}`-building and `db.Create(&entities)`
call (`backfill.go:93-108`), which is unchanged by this fix and untouched by the dialect branch — so tenant
scoping in the write path and idempotency (the boot-time `existing > 0` short-circuit at `backfill.go:59-66`,
also unchanged) apply identically regardless of which grouping branch ran. No divergence found.

### 4. Does `TestBackfillUsesSQLiteFallbackUnderTest` pin what it claims? — partially; non-blocking

The test (`backfill_test.go:233-238`) only asserts `db.Dialector.Name() == "sqlite"` on the test harness's own
`*gorm.DB` — it never calls `Backfill` and never inspects which branch `Backfill` actually took. Given
`gorm.io/driver/sqlite`'s `Name()` is a hardcoded literal (see #2), this assertion is already an invariant of
`databasetest.NewInMemoryTenantDB` and cannot fail from any change to `Backfill`'s own `if` condition — e.g. if a
future edit changed `db.Dialector.Name() == "sqlite"` to a typo'd string that never matches, this test would still
pass while `Backfill` silently fell through to the unbounded/broken `else` branch under the test suite. In that
scenario, the *existing* `TestBackfill` and `TestBackfillCountsDuplicatesAndSoftDeletes` would still catch the
regression — they'd fail with the sqlite driver's `MIN()/MAX()` scan error the fix's own comments describe — so
production behavior is not left unguarded. But the new test's docstring claim ("pins the dialect check so a
future change ... fails loudly here instead of surfacing as a flaky CI failure") overstates what it verifies: it
pins the driver's dialect string, not `Backfill`'s branch selection. Non-blocking — coverage of the actual risk
exists via the other two tests, the new test just isn't the mechanism providing it.

### 5. Regression check — soft-deleted rows / idempotency / tenant scoping / non-fatal main.go handling

All confirmed unchanged and still passing:

- Soft-deleted inclusion: `Unscoped()` preserved on both branches (`backfill.go:121, 139`); `TestBackfillCountsDuplicatesAndSoftDeletes` (`backfill_test.go:171-192`) creates a soft-deleted asset via `db.Delete` and asserts `Get(...) == 1` after `Backfill` — PASS, re-ran independently.
- Idempotency: the `existing > 0` short-circuit (`backfill.go:59-66`) is untouched by this commit; `TestBackfillCountsDuplicatesAndSoftDeletes/is_idempotent` (`backfill_test.go:222-244`) runs `Backfill` twice, asserts second call returns 0 and prior counts unchanged — PASS.
- Tenant scoping (write path): `TenantId` still read per-row (`g.TenantId` from `backfillGroupRow`, sourced from `cash_assets.tenant_id` in both branches) and written per-entity (`backfill.go:97`) — no merge across tenants; unchanged from `cb9eeb9`.
- `main.go` non-fatal handling: `main.go` is not touched by `39de4adc1` at all (confirmed by `git show --stat`), so the prior review's PASS on this item (error logged as warning, doesn't block startup) still holds unmodified.

### 6. Prior non-blocking items — closed as reported

- **#1 (global vs per-tenant idempotency gate)**: now documented at `backfill.go:50-57`, directly above `Backfill`. Read the comment text verbatim — it accurately states the gate is global (`cash_purchase_records` count, not per-tenant), names both writers to that table (`Backfill` and `Record`, "wired since task-5"), and states the ship-together assumption and its failure mode ("a permanent no-op for every other tenant on every later boot") if that assumption is ever violated. This matches the prior review's Finding 2 concern precisely — the comment is accurate, not just present.
- **#2 (misleading subtest names)**: `TestBackfill`'s two previously-misleading subtests are renamed to `"a single purchase stays counted once"` and `"a commodity with no assets has no record"` (`backfill_test.go:96, 106`), both of which now accurately describe what they check (leftover-state re-assertions, explicitly called out as such in a new comment at `backfill_test.go:88-92`). The real duplicate-counting and soft-delete-inclusion coverage still lives in `TestBackfillCountsDuplicatesAndSoftDeletes`, unchanged. Verified via `-v` run: no subtest or fixture was dropped — `TestBackfill` still has 4 subtests (was 4 before, one being `"skips zero commodity id"` retained verbatim), `TestBackfillCountsDuplicatesAndSoftDeletes` still has its `is_idempotent` subtest. Nothing lost.

## Not evaluable

- **Behavior of `backfillGroupsSQL` against a live Postgres instance** — this environment has no Postgres to run
  it against. What's verified instead: (a) `db.Create(&entities)`/downstream write path is identical to what
  `cb9eeb9` already exercised and was reviewed against sqlite; (b) the SQL text itself (`Select`/`Joins`/`Where`/
  `Group`) is syntactically ordinary GROUP BY/MIN/MAX SQL with no Postgres-specific extensions, and matches the
  brief's original Step 3 instruction verbatim; (c) driver dispatch is proven sound (see #2) so *if* the SQL is
  correct, production will reach it. What would close this: a `go test` run (or a manual `Backfill` invocation)
  against a real Postgres instance — e.g. via `docker compose` bringing up Postgres and pointing this service's
  test harness at it, or an integration test tier this module doesn't currently have. This was also flagged
  not-evaluable in the prior review for the same reason and is unchanged by this fix; the final whole-branch
  review should inherit it.
- Real production `cash_assets` row counts (whether the original unbounded load would have caused a real
  incident) — no access to production DB metrics from this review surface. Now moot for the production path
  since it's bounded regardless, but still not evaluable were it ever needed for the sqlite-fallback comment's
  "test-fixture concern, not a production one" claim to be double-checked against actual CI fixture sizes (which
  are small by construction and not a risk in practice, but not independently measured here).

## Verdict rationale

The blocking finding is closed: production (Postgres, confirmed via `database.Connect`'s hardcoded
`postgres.New(...)`) now runs a single bounded SQL aggregate query, and the mechanism selecting that path
(`Dialector.Name()`, hardcoded per-driver, not derived from any runtime-configurable value) cannot misfire in
either direction. The two grouping branches are semantically identical field-by-field, so the sqlite-only test
suite is a faithful proxy for production behavior on this specific SQL. Both prior non-blocking items were
genuinely addressed, not just marked closed. One minor non-blocking observation on the new test's claimed
regression-protection is noted (#4 above) but is not a defect — the actual regression it describes is still
caught, just by a different existing test. The one honest not-evaluable (no live Postgres in this environment)
is a genuine environment limitation, not a corner cut by the implementer, and is inherited to the whole-branch
review as noted above.

```text
verdict: APPROVED_WITH_FINDINGS
artifact: docs/tasks/task-240-cash-shop-stub-operations/review-task-6-fix.md
scope_confirmed: reviewed 39de4adc1 in isolation (backfill.go, backfill_test.go only; main.go untouched by this commit, confirmed via git show --stat) against the prior review's blocking finding and the six judged questions; ran go build/go test independently from services/atlas-cashshop/atlas.com/cashshop
blocking: 0
non_blocking: 1
  - services/atlas-cashshop/atlas.com/cashshop/purchaserecord/backfill_test.go:233-238 — TestBackfillUsesSQLiteFallbackUnderTest asserts db.Dialector.Name() on the test harness's own DB, never invokes Backfill or inspects which branch it took; the test's docstring claims it pins Backfill's dialect check, but a typo'd condition inside Backfill would not be caught by this test (it would still be caught by the other two Backfill tests via a scan-error failure, so the actual risk is not unguarded).
not_evaluable: 2
  - backfillGroupsSQL's SQL text has not been run against a live Postgres instance from this environment; would be closed by a docker-compose Postgres integration run.
  - Production cash_assets row-count-driven risk is now moot for the production path (bounded regardless) but was not independently re-measured.
```

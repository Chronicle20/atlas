# Review: Task 6 — Backfill purchase records from existing locker rows

Commit range: `e1a6d590a..cb9eeb9` (one commit, `cb9eeb9`)
Brief: `.superpowers/sdd/plan/task-6-brief.md`
Report: `.superpowers/sdd/plan/task-6-report.md`

## Scope

`git diff --stat e1a6d590a..cb9eeb9`:

```
services/atlas-cashshop/atlas.com/cashshop/main.go                          |  10 +
.../atlas.com/cashshop/purchaserecord/backfill.go                           | 110 +++
.../atlas.com/cashshop/purchaserecord/backfill_test.go                      | 236 +++
```

Matches the brief's file list exactly (`backfill.go`, `backfill_test.go` new; `main.go` wired). No scope drift.

## Verified with tooling

```
cd services/atlas-cashshop/atlas.com/cashshop && go build ./... && go test ./...
```
All packages `ok`, `atlas-cashshop/purchaserecord` included. Re-ran `go test ./purchaserecord/... -run TestBackfill -v -count=1` (not cached): all 6 subtests PASS.

## Findings

### 1. BLOCKING — unbounded full-table load into memory at boot

`purchaserecord/backfill.go:50-56`:

```go
var assetRows []backfillAssetRow
err := db.Unscoped().
    Table("cash_assets").
    Select("cash_assets.tenant_id AS tenant_id, cash_compartments.account_id AS account_id, cash_assets.commodity_id AS commodity_id, cash_assets.created_at AS created_at").
    Joins("JOIN cash_compartments ON cash_compartments.id = cash_assets.compartment_id").
    Where("cash_assets.commodity_id <> 0").
    Scan(&assetRows).Error
```

This pulls every un-grouped `cash_assets` row (all tenants, all accounts, live and soft-deleted) into a single Go slice with no `LIMIT`, no cursor (`Rows()`), and no batching (`FindInBatches`), then groups/aggregates it in application memory (`backfill.go:66-88`). Memory use is proportional to the *entire historical row count* of `cash_assets` across the whole system, not to the number of distinct `(tenant, account, commodity)` groups the brief's SQL `GROUP BY` would have returned. `cash_assets` rows are soft-deleted, never purged, on every withdrawal and rebate, so this table only grows over the service's lifetime — this runs at boot, before Kafka consumers are wired (`main.go:70`), so a large-enough table risks a slow or OOM-killed startup, i.e. it directly threatens the "must not break startup" requirement the brief itself calls out for the error path.

The report's own justification (`task-6-report.md:21`) confirms this is not a production constraint: *"Production runs on Postgres where the aggregate query would have worked, but the module's own test suite runs on sqlite, so this had to work here too."* That is a test-driver limitation (`mattn/go-sqlite3`'s inability to scan `MIN()`/`MAX()` over `DATETIME` back into `time.Time`) driving a production-path design regression. No dialect check, no chunked/cursor read, and no attempt to keep the SQL aggregate on the real driver while working around sqlite only in the test harness (e.g. `db.Dialector.Name()` branch, or `FindInBatches`/`Rows()` to bound memory even in the Go-side-aggregation fallback) was made. The Go-side aggregation is semantically identical to what the SQL would have produced (same grouping keys, same count/min/max) — the defect is scalability, not correctness of the arithmetic.

This is exactly the risk the task brief for this review flagged as worth judging hard, and I judge it a blocking defect: it silently downgrades a boot-time production property (bounded memory, bounded query cost) to accommodate a test artifact, with no bound and no fallback plan documented.

### 2. Non-blocking — the idempotency gate is global, not per-tenant, and interacts unsafely with `Record`'s live write path

`purchaserecord/backfill.go:41-48`:

```go
var existing int64
if err := db.Model(&entity{}).Count(&existing).Error; err != nil {
    return 0, err
}
if existing > 0 {
    ...
    return 0, nil
}
```

This matches the brief's literal instruction ("Short-circuit ... when `cash_purchase_records` already has at least one row") verbatim, so it is not a deviation — but it is worth flagging as a design risk. `purchaserecord.Record` (`purchaserecord/administrator.go:17-37`) is the normal live-purchase path and writes to the same table on every purchase, already wired since Task 5 (`e1a6d590a`). If any tenant's purchase gets recorded via `Record` before `Backfill`'s first successful run for any reason (staggered rollout of Task 5 vs Task 6, a crash-loop that restarts after a live purchase already landed, etc.), `Backfill` will see `existing > 0` and permanently skip seeding history for every other tenant on every subsequent boot — silently, with only a Debug-level log line. Given Task 5 and Task 6 land in the same branch/PR here, this is unlikely to bite on the very first production rollout, but the gate has no defense if that assumption is ever violated later (e.g. a hotfix redeploy, a delayed Task 6 rollout). Worth a one-line comment recording the "must ship atomically with Task 5, or the backfill is a permanent no-op" assumption, or scoping the gate per tenant.

### 3. Non-blocking — misleading subtest names in `TestBackfill`

`purchaserecord/backfill_test.go:92-120`: the `"counts duplicates"` and `"includes soft-deleted assets"` subtests inside `TestBackfill` do not create any duplicate-commodity or soft-deleted fixtures — `TestBackfill` only ever creates two live assets (commodity 10000, 20000) in its one `"seeds from live assets"` subtest. These two later subtests just re-assert leftover state (`count(10000) == 1`, `count(30000) == 0`) that was never exercised as duplicates or soft-deletes in this test function. Confirmed by running with `-v`: neither subtest logs any new asset creation or backfill call. The actual duplicate-counting and soft-delete-inclusion coverage lives in the separately-named `TestBackfillCountsDuplicatesAndSoftDeletes` (`backfill_test.go:126-215`), which does exercise both correctly (3 duplicate rows → count 3; one soft-deleted row → count 1, confirmed via `db.Delete` then `Get`). Functional coverage exists and is real (not fake-green), but the names inside `TestBackfill` are misleading — a future reader trusting the subtest name over its body could think duplicate/soft-delete handling is covered in a place it isn't.

## Checklist (per review brief)

| Check | Result |
|---|---|
| `Backfill(l logrus.FieldLogger, db *gorm.DB) (int, error)` signature | PASS — `backfill.go:40`, exact match. |
| Soft-deleted rows included | PASS — `Unscoped()` at `backfill.go:51`; proven by `TestBackfillCountsDuplicatesAndSoftDeletes` (`backfill_test.go:150-153, 174-180`): asset soft-deleted via `db.Delete`, `Get` still returns count 1 after backfill. |
| Idempotency | PASS — `TestBackfillCountsDuplicatesAndSoftDeletes/is_idempotent` (`backfill_test.go:190-214`) runs `Backfill` twice, asserts second call returns 0 and prior counts are unchanged. Re-ran independently, confirmed PASS. |
| Tenant scoping (write path) | PASS — `TenantId` read per-row from `cash_assets.tenant_id`, grouped and written per-row (`backfill.go:74,95`); no merge across tenants in the write path. See Finding 2 for the separate short-circuit-gate concern (not a write-path bug). |
| `main.go` error handling | PASS and matches brief exactly — `main.go:70-74`: `Backfill` error is logged via `l.WithError(err).Warn(...)` and does not stop startup; success count logged at Info. Runs after `database.Connect` (migrations) and before Kafka consumer wiring (`main.go:106+`), so no live-write race within a single process. |
| Go-side aggregation semantics vs. brief's SQL | PASS on correctness, FAIL on scalability — see Finding 1. |
| Zero-commodity skip | PASS — `Where("cash_assets.commodity_id <> 0")` (`backfill.go:55`); tested at `backfill_test.go:112-120,182-188`. |
| Build/tests | PASS — `go build ./...` clean; `go test ./...` all `ok`; `go test ./purchaserecord/... -run TestBackfill -v -count=1` all 6 subtests PASS (verified independently, not cached). |

## Not evaluable

- Behavior against the real Postgres driver in production (whether the original brief's SQL `GROUP BY`/`MIN`/`MAX` genuinely fails only on sqlite) was not independently verified against a live Postgres instance — taken on the report's assertion, which is plausible (documented, well-known class of driver-level type-erasure on aggregates) but not something I could confirm from this review surface.
- Actual production `cash_assets` row counts / whether the unbounded load in Finding 1 would materialize as a real incident versus a theoretical risk — no access to production DB metrics from this review surface.

## Verdict rationale

Finding 1 (unbounded memory load at boot, driven by a test-driver workaround rather than a production constraint) is a genuine regression against a property the brief's design implicitly protected (bounded, single-aggregate-query backfill) and against the "must not break startup" concern this review was asked to check hard. It requires a fix — either a dialect-conditional query that keeps the SQL aggregate on Postgres, or a batched/cursor read that bounds memory even in the Go-side-aggregation fallback — before this is safe to ship at boot on a production-sized table.

---

```text
verdict: CHANGES_REQUIRED
artifact: docs/tasks/task-240-cash-shop-stub-operations/review-task-6.md
scope_confirmed: reviewed the full diff of cb9eeb9 (backfill.go, backfill_test.go, main.go wiring) against task-6-brief.md; ran go build/go test independently from services/atlas-cashshop/atlas.com/cashshop
blocking: 1
  - services/atlas-cashshop/atlas.com/cashshop/purchaserecord/backfill.go:50-56 — Backfill scans the entire (unscoped, ungrouped) cash_assets table into a Go slice with no LIMIT/cursor/batching before aggregating in memory; this runs at boot and is unbounded on a production-sized table, a scalability regression introduced solely to work around a sqlite test-driver limitation (the report itself confirms Postgres would have handled the brief's SQL GROUP BY/MIN/MAX fine).
non_blocking: 2
not_evaluable: 2
```


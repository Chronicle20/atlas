# DB Transaction Coverage for Multi-Entity Mutations — Design

Version: v1
Status: Proposed
Created: 2026-07-02
PRD: docs/tasks/task-119-db-transaction-coverage/prd.md

---

## 1. Summary

The PRD scopes this task as: audit the 14 database-backed services with zero `database.ExecuteTransaction` call sites, wrap every multi-statement mutation, and prove each wrap with a rollback test. Design-phase exploration produced two discoveries that reshape the work:

1. **`ExecuteTransaction` has never opened a transaction.** Its `isTransaction` guard returns `true` for every `*gorm.DB`, so all 53 existing call sites across 18 services run their "transactions" as plain sequential statements. Fixing `libs/atlas-database/transaction.go` is the foundational deliverable of this task — without it, every wrap this task adds is decoration and every FR-3.1 rollback test fails. This also silently undermines task-114's outbox enqueue-in-tx atomicity, so the fix must land before task-114's remediation is meaningful (§3.4).

2. **The PRD's grep premise ("zero call sites" = "no transactions") is incomplete.** Five of the 14 services already wrap their multi-statement flows in raw GORM transactions (`db.Transaction(...)` or manual `Begin`/`Commit`), which *do* work. Their remediation is standardization onto the helper (plus fixing emit-inside-transaction defects), not gap-filling. The genuine unwrapped multi-write gaps concentrate in **atlas-storage**; several other services' write-verb hits turn out not to be DB writes at all (in-memory registries, saga REST calls).

The resulting design: fix the lib (D0), run the full audit with a refined taxonomy (§5), remediate in three categories — standardize, wrap, justified-no-change (§6) — and verify with a shared fault-injection rollback-test pattern built on the existing `databasetest` helper (§7).

## 2. Discovery: `ExecuteTransaction` is a no-op

### 2.1 The defect

libs/atlas-database/transaction.go:9-18:

```go
func ExecuteTransaction(db *gorm.DB, fn func(tx *gorm.DB) error) error {
	if isTransaction(db) {
		return fn(db)
	}
	return db.Transaction(fn)
}

func isTransaction(db *gorm.DB) bool {
	return db.Statement != nil && db.Statement.ConnPool != nil
}
```

`gorm.Open` initializes the root `*gorm.DB` with a non-nil `Statement` whose `ConnPool` is the connection pool, and every derived handle (`WithContext`, `Session`) clones that. `Statement.ConnPool` is therefore **never nil**, `isTransaction` is always true, and `fn(db)` runs with no transaction — ever.

### 2.2 Evidence

Empirical probe run in this worktree (gorm v1.31.2, sqlite in-memory), temporary test in `libs/atlas-database`:

```
root db:           isTransaction=true
db.WithContext:    isTransaction=true
db.Session:        isTransaction=true
inside Transaction: isTransaction=true
rows after failed ExecuteTransaction: 1   ← the write survived fn returning an error
```

An `ExecuteTransaction` whose `fn` does `Create` then returns an error leaves the row committed. The bug has been present since the lib was introduced (commit 67a54a173, the shared-atlas-database migration).

### 2.3 Blast radius

- All 53 `ExecuteTransaction` call sites in the 18 "already transactional" services are non-atomic today. Their code is structured correctly (tx threaded via `WithTransaction`), so activating real transactions requires no call-site change.
- The PRD's FR-2.3 re-entrancy claim ("it detects an existing transaction and joins it") describes intent, not current behavior — currently *everything* "joins" a transaction that does not exist.
- task-114 (outbox adoption) builds `Enqueue(tx, ...)` atomicity on these same transactions. With the bug, an outbox row commits even when the enclosing "tx" fails. task-114 is unmerged, so it can rebase onto the fix.
- No rollback test exists anywhere in the repo that would have caught this (the only `ExecuteTransaction` test, services/atlas-data/atlas.com/data/commodity/processor_test.go, is a source-text assertion). FR-3.1's rollback tests are exactly the missing verification class.

### 2.4 Decision D0 — fix `isTransaction` via `gorm.TxCommitter`

```go
func isTransaction(db *gorm.DB) bool {
	committer, ok := db.Statement.ConnPool.(gorm.TxCommitter)
	return ok && committer != nil
}
```

Inside a real transaction, `Statement.ConnPool` is a `*sql.Tx` (implements `gorm.TxCommitter`); on the root pool it is `*sql.DB`/prepared-stmt pool (does not). This is GORM's own idiom for the same check in `finisher_api.go`.

**Alternatives considered and rejected:**

- *Delegate unconditionally to `db.Transaction(fn)`.* GORM natively handles nesting via savepoints, so the helper could be one line. Rejected: changes nested-composition semantics from "join the outer tx" (documented contract, what all composed processor code expects) to "savepoint per nesting level" — a behavior change with no requirement behind it, and savepoint round-trips on every nested call.
- *Leave the lib broken; use raw `db.Transaction` for this task's wraps.* Rejected outright: abandons the project-standard helper, leaves 18 services non-atomic, contradicts the PRD's acceptance criteria and DL-4's purpose.

**Regression tests** (new `libs/atlas-database/transaction_test.go`):
1. Failed `fn` from a root handle rolls back all writes (the probe scenario, made permanent).
2. Successful `fn` commits.
3. Nested `ExecuteTransaction` inside an outer transaction joins it (no new tx; outer rollback discards inner writes).
4. All of the above with tenant callbacks registered via `databasetest.NewInMemoryTenantDB`.

**Fleet verification for the lib change:** the fix alters runtime behavior of every importer. Gate: `go test -race ./...` in **all** modules that call `ExecuteTransaction` (the 18) plus the 14 audited ones; `go vet`; `docker buildx bake all-go-services` once (the lib is COPY'd into every image). Expected risk is low — code paths become *more* atomic, and no call-site code changes — but a test anywhere that accidentally depended on partial-write survival would surface here, and that is the point.

**Delivery vehicle:** the fix + regression tests are the **first commit on this branch**, and the recommendation is to cut them into an immediate small standalone PR (rebase-cut of commit 1, per the one-worktree/rebase-at-PR-time convention) so task-114 rebases onto real transaction semantics before its own remediation merges. If the owner prefers to keep everything in the task-119 PR, the sequencing risk is that task-114 lands "atomic" outbox enqueues that aren't — flagged for decision at PR time; the code is identical either way.

**Declared PRD deviation:** the PRD non-goal "Touching the 18 services that already use ExecuteTransaction" is honored at the code level (zero edits there), but the lib fix necessarily activates their dormant transaction semantics. This is unavoidable: the PRD's own acceptance criteria (rollback tests pass) are unsatisfiable against the broken lib.

## 3. Verified current state of the 14 services

Full survey performed in this worktree (design-phase reconnaissance; the audit phase re-verifies exhaustively). Classification legend in §5.

| Service | Design-phase finding | Key evidence |
|---|---|---|
| atlas-npc-conversations | Already transactional via raw `db.Transaction` — 6 sites, all class A (conversation + derived recipe rows) | conversation/npc/processor.go:128,149,173,197,215,267 |
| atlas-keys | Already transactional via raw `db.Transaction` — 4 sites, class B (reset = delete-all + loop-create; RMW change-key) | key/processor.go:72,91,105,117 |
| atlas-families | Already transactional via raw `db.Transaction` — 3 sites, class B (senior+junior member saves); has `WithTransaction` (family/processor.go:73) | family/processor.go:173,245,318 |
| atlas-marriages | **Manual `Begin`/`Rollback`/`Commit`** in `executeInTransaction`; proposal-accept + marriage-create + **Kafka emit inside the tx** | marriage/processor.go:1581,1688-1717 |
| atlas-monster-book | Raw `db.Transaction` at the consumer layer with **`message.Emit` nested inside** (publish-before-commit); both processors already expose `WithTransaction` | kafka/consumer/monsterbook/consumer.go:56-72, kafka/consumer/character/consumer.go:49; card/processor.go:45, collection/processor.go:89 |
| atlas-storage | **Unwrapped multi-write flows** — the real gap of the group: `ExpireAndEmit` (delete asset → emit mid-flow → create replacement), `MergeAndSort` (loop of quantity-updates + deletes + re-slotting), `GetOrCreateStorageId` (read-then-create) | storage/processor.go:720,483; asset/processor.go:70 |
| atlas-account | Single-table RMW flows (GetById→Updates; GetByName→Create); each has exactly one write statement | account/processor.go:142,189,268,296,414 |
| atlas-ban | Single-statement CRUD + background delete-sweep tickers; possible ban↔history pairing to confirm in audit | ban/processor.go:63,86; ban/task.go:31, history/task.go:33 |
| atlas-maps | Single-statement writes (location save, visit delete) | character/location/administrator.go, visit/administrator.go |
| atlas-map-actions | DB writes are seeder-cycle only (delete-all + bulk-create via `libs/atlas-seeder`); runtime actions are saga REST | script/administrator.go:78,105; libs/atlas-seeder/seed.go:85-120 |
| atlas-portal-actions | Same seeder-cycle shape | script/administrator.go:83, script/subdomain.go:28,83 |
| atlas-reactor-actions | Same seeder-cycle shape | script/administrator.go:82, script/subdomain.go:90 |
| atlas-party-quests | Seeder-cycle on `definition/*` only; the entire instance state machine (`instance/processor.go`) is **in-memory registry, not DB** | definition/administrator.go:73, definition/subdomain.go:59 |
| atlas-saga-orchestrator | Single-statement conditional writes guarded by optimistic versioning (in-memory `s.ver` map + version-checked `Updates`); most handler "writes" are outbound REST | saga/store.go:100-186,240,290 |

Two cross-cutting facts the audit must encode:

- **The seeder cycle is a shared-lib semantic, not a per-service flow.** `libs/atlas-seeder` `runSubdomain` (seed.go:85-120) deliberately runs `DeleteAllForTenant` + per-file `BulkCreate` non-atomically, with per-file error accounting (`counts.Failed`, continue-on-error) and a per-(tenant,group) mutex serializing concurrent seeds (seed.go:22-39). Wrapping it would change semantics for *every* seeder consumer, including services outside this task's scope.
- **Write-verb greps overcount.** Saga REST `.Create(...)` calls, in-memory registry mutations, and `tenant.Create` are not DB writes. The audit records these exclusions explicitly so the sweep is verifiably full.

## 4. Architecture of the change

Three layers, smallest possible surface each:

1. **`libs/atlas-database`** — the D0 one-function fix + regression tests, plus one new test-support helper in `databasetest` (§7.2). No signature changes; every existing caller compiles unchanged.
2. **Service write paths** — transaction boundaries standardized or added inside the existing Processor pattern (`WithTransaction(tx)` threading, buffered `Method(mb)` composition). No new abstractions, no new libs (per the audit-existing-libs rule: `atlas-database` already owns this concern).
3. **Audit artifact** — `docs/tasks/task-119-db-transaction-coverage/audit.md`, the DL-4 closure evidence.

## 5. Audit design

### 5.1 Methodology (full sweep, per service)

1. Enumerate write statements: grep `.Create(`, `.Save(`, `.Update(`, `.Updates(`, `.Delete(`, `.Exec(`, plus raw-transaction markers `.Transaction(`, `.Begin()`, `.Commit()`, `.Rollback()` in non-test Go files.
2. Discard non-DB hits (REST clients, in-memory registries, `tenant.Create`) — recorded in the audit as exclusions with file:line, so "full sweep" is checkable.
3. Resolve each remaining write to its triggering entry point (REST handler, Kafka consumer, ticker, seeder) and group writes by entry point.
4. Classify each entry point (below) with file:line citations.

### 5.2 Refined taxonomy

PRD classes, with two refinements forced by the survey and one by the PRD's own test requirement:

- **A — multi-table**: ≥2 writes across ≥2 tables in one logical operation. → Wrap.
- **B — multi-statement single-table**: **≥2 write statements** to one table where mid-flow failure leaves inconsistent state. → Wrap.
  *Refinement:* a read-modify-write with exactly **one** write statement is **not** class B — there is no "earlier write" to roll back, FR-3.1's rollback test is undefinable for it, and a transaction wrap alone does not close the concurrent-interleaving gap (that needs row locking or unique constraints, both behavior/schema changes the PRD excludes). Such flows are class C with a mandatory **race annotation** in the audit (e.g. atlas-account `GetOrCreate` needs a unique constraint on account name to be race-free — documented, not fixed here).
- **C — single-statement**: no change; race annotation where an RMW gap exists.
- **D — intentionally non-atomic**: written justification required. Known members: saga-orchestrator persistence (optimistic version guard is the concurrency mechanism; single-statement writes; saga compensation handles partial cross-service state) and the **seeder cycle** in the four actions services + party-quests definitions (shared `atlas-seeder` semantics: per-file error accounting, deliberate continue-on-error, per-tenant lock; changing it is an `atlas-seeder` design change affecting out-of-scope consumers — recorded as a candidate follow-up in the audit, explicitly not smuggled into this task).

Orthogonal flags, recorded per entry point:

- **[T] already-transactional, non-standard** — raw `db.Transaction` or manual `Begin`/`Commit`. Remediation: convert to `ExecuteTransaction` (FR-2.1/FR-2.3; manual `Begin`/`Commit` elimination is an explicit acceptance criterion).
- **[E] emit-inside-transaction** — Kafka publish happens before commit (monster-book consumers: `db.Transaction` outer, `message.Emit` inner; marriages: emit inside `executeInTransaction`). Remediation: invert nesting so publish follows commit (§6.1). This changes failure-mode behavior only (no event for a rolled-back write — which is FR-2.2's requirement); happy-path events are byte-identical, satisfying FR-2.4.

### 5.3 audit.md format

One section per service: (a) write-statement inventory table (file:line → entry point → class + flags), (b) exclusions list, (c) verdicts with justification for C/D, (d) remediation pointer (commit) for A/B/[T]/[E]. A closing matrix summarizes all 14 for DL-4 closure.

## 6. Remediation design

### 6.1 Canonical composition (the one pattern every wrap follows)

The exemplar is atlas-guilds (services/atlas-guilds/atlas.com/guilds/guild/processor.go:253,398): **`Emit` outside, `ExecuteTransaction` inside, buffer-puts inside the tx closure**. Buffered messages only reach Kafka after `ExecuteTransaction` returns nil:

```go
func (p *ProcessorImpl) DoThingAndEmit(args...) error {
	return message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(mb *message.Buffer) error {
		return p.DoThing(mb)(args...)          // Emit publishes only if this returns nil
	})
}

func (p *ProcessorImpl) DoThing(mb *message.Buffer) func(args...) error {
	return func(args...) error {
		return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
			// all writes via p.WithTransaction(tx) / collaborator.NewProcessor(l, ctx, tx)
			// mb.Put(...) is buffering, not publishing — safe inside the tx
		})
	}
}
```

Rules: no manual `Begin`/`Commit`; `tx` handles derive from `p.db.WithContext(p.ctx)` so tenant callbacks stay active (NFR multi-tenancy); nested processors receive `tx` via `WithTransaction`/constructor injection and join via the (fixed) re-entrancy check; error from `fn` propagates unswallowed to the existing logging path.

Monster-book's consumers currently have this composition **inverted** (`db.Transaction` outer, `message.Emit` inner — publish-before-commit, kafka/consumer/monsterbook/consumer.go:56-57). The fix is mechanical inversion into the pattern above.

### 6.2 Category 1 — standardize (5 services, flows already atomic)

| Service | Change |
|---|---|
| atlas-npc-conversations | 6× raw `db.Transaction` → `ExecuteTransaction`; emit placement already correct (verify in audit) |
| atlas-keys | 4× raw `db.Transaction` → `ExecuteTransaction` |
| atlas-families | 3× raw `db.Transaction` → `ExecuteTransaction`; keep existing `WithTransaction` threading |
| atlas-monster-book | 2 consumer sites → canonical composition (fixes [E] as well); processors untouched (`WithTransaction` already present) |
| atlas-marriages | Replace `executeInTransaction`'s manual `Begin`/`Rollback`/`Commit` (marriage/processor.go:1688-1717) with `ExecuteTransaction`; keep the shadow-processor injection it already does; move the buffered emit outside the tx ([E]) |

These flows keep their existing boundaries — same writes, same order (FR-2.4). Only the wrapper API and emit placement change.

### 6.3 Category 2 — wrap genuine gaps

- **atlas-storage** (the substantive work): wrap `ExpireAndEmit` (storage/processor.go:720 — delete + replacement-create become one tx; the mid-flow emit moves to a buffer published after commit), `MergeAndSort` (storage/processor.go:483 — the whole merge/compact/sort write sequence becomes one tx), and `GetOrCreateStorageId` (asset/processor.go:70 — joins the caller's tx via re-entrancy). Storage/asset processors gain `WithTransaction` following the monster-book shape, since neither has it today.
- **atlas-account, atlas-ban, atlas-maps**: expected mostly class C per the survey; any ≥2-write flow the full audit surfaces (e.g. a confirmed ban+history pairing) gets the canonical wrap. No pre-commitment — the audit decides.
- If the audit finds a high-frequency flow, its call rate is recorded before wrapping (PRD NFR); the survey found none (storage flows are UI/NPC-driven, not per-packet).

### 6.4 Category 3 — justified no-change (class D verdicts)

atlas-saga-orchestrator (optimistic-lock store), seeder cycles in map-actions/portal-actions/reactor-actions/party-quests, party-quests in-memory instance machinery, ban/history sweep tickers (single-statement). Each gets its written justification in audit.md per FR-2.5.

### 6.5 Emit convention vs task-114

task-114's migration list (its PRD §7) covers the 18 already-transactional services — **none of this task's 14 are in it**. Default convention here is therefore the existing buffer/`Emit`-after-commit pattern (§6.1), which satisfies FR-2.2 with no outbox dependency. At rebase time (after task-114 merges), each touched service is re-checked: if task-114's FR-3.1 sweep did migrate it to the outbox, the wrap uses the outbox `Emit`-shaped wrapper (enqueue-in-tx) instead, cited in audit.md. The composition in §6.1 makes this a one-line provider swap, not a restructure.

## 7. Testing design

### 7.1 Rollback test pattern (one per wrapped flow, FR-3.1)

Tests exercise the **production entry point** (not a synthetic transaction) against `databasetest.NewInMemoryTenantDB`, with failure injected between the flow's writes via a GORM callback:

```go
db := databasetest.NewInMemoryTenantDB(t, /* the service's migrations */)
databasetest.FailWritesOn(t, db, "second_table")   // §7.2 helper — fail the flow's later write

err := invokeTheProductionEntryPoint(l, ctx, db)   // e.g. the monster-book consumer handler
require.Error(t, err)

// the write that preceded the injected failure must be gone
require.Zero(t, countRows(t, db, "first_table"))
```

Concrete first instance: monster-book's card-picked-up flow — fail writes on `collections`, drive the consumer handler, assert the `cards` upsert rolled back.

The injection point is the *second* table (or second statement) of the flow, so the assertion proves the *first* write rolled back — exactly FR-3.1. sqlite in-memory supports real transactions, so these tests also double as living proof the D0 fix works in every service module.

### 7.2 One shared helper: `databasetest.FailWritesOn`

A single addition to the existing `libs/atlas-database/databasetest` package (precedent: `NewInMemoryTenantDB`, `TenantContext` already live there):

```go
// FailWritesOn registers create/update/delete callbacks that fail any write
// to the named table, for rollback testing.
func FailWritesOn(t *testing.T, db *gorm.DB, table string)
```

Implementation: `db.Callback().Create().Before("gorm:create").Register(...)` (and update/delete equivalents) calling `d.AddError(...)` when `d.Statement.Table == table`. This avoids 14 hand-rolled copies and respects the no-`*_testhelpers.go` rule — it is shared lib infrastructure, not per-service test-only constructors; entity setup in tests continues to use the services' Builder patterns.

For class-B same-table flows (e.g. keys reset: delete + create on `keys`), the helper takes the write *verb* into account — failing only `Create` on the table lets the preceding `Delete` execute and then roll back. If a flow needs finer granularity (fail the Nth statement), the test registers its own counting callback inline; no extra abstraction until a second consumer needs it (YAGNI).

### 7.3 Verification gates

- New: lib regression tests (§2.4), one rollback test per wrapped/standardized flow.
- Existing tests stay green: `go test -race ./...` in every changed module **and** in all 18 `ExecuteTransaction`-calling modules (lib behavior change).
- `go vet ./...`, `go build ./...` per changed module; `docker buildx bake all-go-services` once for the lib change, plus per-service bakes for touched services; `tools/redis-key-guard.sh`.
- FR-2.4 (no observable behavior change) is checked per touched flow by asserting identical event emission on the happy path in existing consumer/processor tests.

## 8. Sequencing and risks

Order of work:

1. **Commit 1**: D0 lib fix + regression tests + `FailWritesOn` helper. Decision point: cut as immediate standalone PR (recommended — unblocks task-114's semantics) or carry in the task PR.
2. **Audit phase** (can start immediately, independent of task-114/116): full sweep, `audit.md` committed.
3. **Wait for task-114 and task-116 to merge; rebase.** task-116 rewrites processor files in these same services (gen3 unification); remediation diffs are authored against the post-116 shape. Re-verify emit conventions per §6.5 against what task-114 actually landed.
4. **Remediation commits**, one commit per service, category 1 (mechanical) before category 2 (storage), rollback test in the same commit as its wrap (TDD: test first, red against the unwrapped flow, green after the wrap).
5. Full verification gate; code review; PR.

Risks:

| Risk | Mitigation |
|---|---|
| D0 fix changes runtime behavior of 18 untouched services | Fleet-wide `go test -race`; the activated semantics are exactly what those call sites were written to assume; any test relying on partial-write survival fails loudly at step 1, not in production |
| task-114 merges built on no-op transactions | Early standalone PR of commit 1 (recommendation); at minimum, flag to owner before task-114's PR lands |
| task-116 rewrites the same processor files | Remediation deferred until after its merge (PRD sequencing requirement); audit is file-line-cited against current main and re-checked at rebase |
| Emit-placement fixes ([E]) alter failure-mode behavior | That is FR-2.2's requirement; happy-path emission verified unchanged (FR-2.4); called out per flow in audit.md |
| sqlite (tests) vs Postgres (prod) transaction differences | Join-semantics only (no savepoints used); both engines give identical rollback behavior for this pattern |
| Long-lived transactions holding connections (e.g. storage `MergeAndSort` loop) | Flows are bounded (≤ storage capacity, ~100 rows); no network I/O inside any introduced tx (emits are buffered); noted per flow in audit.md |

## 9. Resolution of PRD open questions

- **§9.1 task-114 convention**: it lands outbox enqueue-in-tx via `Emit`-shaped wrappers, but its migration scope does not include these 14 services. Default here: buffer + publish-after-commit (§6.1); per-service re-check at rebase (§6.5).
- **§9.2 latency-sensitive flows in maps/map-actions**: moot. atlas-maps writes are single-statement (class C); atlas-map-actions' runtime path does no DB writes (saga REST) — its only multi-write path is seed-time reindex, class D via the shared seeder.

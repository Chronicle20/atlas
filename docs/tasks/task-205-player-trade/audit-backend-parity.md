# Backend Audit — task-205 meso-custody parity pass

- **Scope:** Go files changed by commits `efec8332a..HEAD` (`bce674d0d`…`349748892`), i.e.:
  - `services/atlas-trades/atlas.com/trades/escrow/{administrator,entity,model,processor,provider}.go` (+tests, +migration_test.go)
  - `services/atlas-trades/atlas.com/trades/trade/{processor,settlement}.go` (+tests)
  - `services/atlas-trades/atlas.com/trades/kafka/consumer/saga/consumer.go`
  - `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/{timer,compensator}.go` (+tests)
  - `libs/atlas-database/databasetest/failwrites.go`
- **Guidelines Source:** backend-dev-guidelines skill (DOM-*, SUB-*, FILE-*, SEC-*)
- **Date:** 2026-08-11
- **Build:** PASS — `go build ./...` clean in `atlas-trades` and `atlas-saga-orchestrator`
- **Vet:** PASS — `go vet ./...` clean in both modules
- **Overall:** NEEDS-WORK (one blocking finding; see Summary)

Default posture is FAIL until evidence is cited; every row below carries a file:line citation.

## escrow package (support/custody package — has `model.go`, treated as domain package)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | `builder.go` exists | PASS (pre-existing, unmodified by diff) | `escrow/builder.go:21-27` `ItemBuilder`/`NewItemBuilder`. **Not extended** for the two new entity types this diff introduces — see finding F1 below. |
| DOM-02 | `ToEntity()` on Model | PARTIAL | `ItemModel`/`MesoModel`/`MesoStakeModel` are read-back-only (`Make*` in `entity.go`→`model.go`); none of the four models ever calls `.ToEntity()` — the package writes entities directly in `administrator.go` (`toItemEntity`, `entity.go:26-73`, and inline literals for `MesoEntity`/`MesoStakeEntity`/`MesoRefundEntity`). This is the package's pre-existing pattern (unchanged by the diff) and internally consistent, so not flagged as new. |
| DOM-03 | `Make(Entity) (Model, error)` | FAIL for `MesoRefundEntity` | `model.go` defines `MakeItem` (142-192), `MakeMeso` (195-207), `MakeMesoStake` (210-223) — **no `MakeMesoRefund`, and no `MesoRefundModel` type exists anywhere in `model.go`.** `MesoRefundEntity` (`entity.go:303-319`, new in this diff) is read, written and deleted as a bare GORM struct throughout `administrator.go` (`RecordMesoRefund` 154-169, `RestoreMesoRefunds` 181-217 reads `[]MesoRefundEntity` directly at line 185-188, `DiscardMesoRefunds` 221-226). It never passes through the immutable-model layer the other three entities use. See F1. |
| DOM-04/05 | `Transform`/`TransformSlice` in `rest.go` | N/A | No `rest.go` in `escrow` — the package is Kafka/DB-only, never exposed over JSON:API. Correctly absent. |
| DOM-06 | Processor accepts `FieldLogger` | PASS | `escrow/processor.go:79` — `l logrus.FieldLogger` field on `ProcessorImpl`; `NewProcessor(l logrus.FieldLogger, ...)` at `processor.go:86`. |
| DOM-10 | Test DB has tenant callbacks | PASS | `escrow/administrator_test.go:19-22` `testDb` → `databasetest.NewInMemoryTenantDB(t, Migration)` (helper lives in `libs/atlas-database/databasetest`, registers tenant callbacks internally — confirmed by `TestCreateItemIsTenantScoped`, `administrator_test.go:93-121`, actually asserting cross-tenant isolation). |
| DOM-11 | Providers use lazy evaluation | PASS | `provider.go:15-27` (`ItemsByRoom`), `142-154` (`MesoStakesByOwner`), `156-164` (`AllMesoStakes`) all return curried functions over `*gorm.DB`, using `model.SliceMap(...)(model.FixedProvider(entities))(model.ParallelMap())()` per existing convention. |
| DOM-21 | Reuse atlas-constants types | PASS | `entity.go:27-31` imports `asset`, `character`, `inventory`, `inventory/slot`, `item` from `libs/atlas-constants`; `MesoStakeEntity.OwnerId character.Id` (`entity.go:277`), `MesoRefundEntity.OwnerId character.Id` (`entity.go:313`). No meso-specific type exists in `libs/atlas-constants` (confirmed: no `meso` package, README has no entry) so the new `Amount int64`/`Delta int32` fields on `MesoStakeEntity`/`MesoEntity` are legitimately raw scalars, not a reinvention. |
| — | Multi-tenancy: manual `tenant_id` filtering vs. automatic context filtering | Pre-existing, not a new-diff finding | `provider.go`/`administrator.go` thread `tenantId uuid.UUID` explicitly through every function signature and manual `Where("tenant_id = ?", …)` clauses rather than the documented automatic-GORM-callback pattern (`patterns-multitenancy-context.md:28-53`). Confirmed via `git show efec8332a~1:…/provider.go` — this shape predates the diff (baseline `ItemsByRoom`/`ItemById` already threaded `tenantId` identically). All functions **added** by this diff (`ArmMesoStake`, `CommitMesoStake`, `ClaimMesoForReturn`, `RecordMesoRefund`, `RestoreMesoRefunds`, etc.) follow the same pre-existing convention consistently, so graded as consistent-with-file, not a new deviation. |
| — | Un-tenant-scoped reads (`MesoStakeById`, `AllMesoStakes`) | PASS (reason holds) | `provider.go:74-99` (doc comment) and `156-164`. Judged on the merits rather than mechanically: `MesoStakeById` is looked up only by the saga-issued `stakeId` from a terminal Kafka status (`trade/processor.go` `resolveMesoStake`, called via `escrow.NewProcessor(...).MesoStakeById(stakeId)`), where the room — and therefore the tenant — may already be torn down; the row itself carries the tenant quad (`MesoStakeModel.Tenant()`, `model.go:138-140`) so the tenant is rebuilt post-lookup rather than pre-filtered. This mirrors the pre-existing, already-reviewed `AllItems`/`AllMesos` justification (`provider.go:166-181`) verbatim and is reachable only from the same two paths (saga-status consumer, boot/ticker sweep). Reasoning holds. |

## trade package (domain package — `settlement.go`, `processor.go` changes)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06 | Processor accepts `FieldLogger` | PASS (unchanged by diff) | `trade` package `ProcessorImpl` predates this diff; not touched for this check. |
| DOM-13 | No cross-domain logic in handlers | N/A | No `resource.go` touched in this diff. |
| — | Transaction discipline / no second pooled connection inside a tx | PASS | `processor.go:411-418` `emit()` opens one `database.ExecuteTransaction(p.db.WithContext(p.ctx), …)`, hands a `withTx`-rebound copy (`processor.go:423-426`, `cp.db = tx`). All new call sites that build `escrow.NewProcessor(p.l, p.ctx, p.db)` (`processor.go:1328`, `1386`, `1445`; `settlement.go:562`, `681`, `1113`, `1490`) run inside methods reached only through `p.emit(func(txp *ProcessorImpl, …) {…})` (verified call chain: `Attest→attest→settle→settlementPayload` all execute on the `txp` receiver at `settlement.go:279-281,318,495`; `addMeso`/`resolveMesoStake` reached via `processor.go:1264-1265,1425-1438`; `abandonSettlement`/`emitUnwind` via `settlement.go:375-396`; `unwindRecord` via the terminal-status handler at `settlement.go:1080`; `unwindStranded` via `settlement.go:1369-1370`). Because `p.db` is already the tx handle by the time these constructors run, `escrow.NewProcessor(p.l, p.ctx, p.db)` reuses the **same** pooled connection rather than opening a second one — no deadlock risk under pool size 1 (the exact class of bug in `bug_tx_scoped_reader_not_rebound_deadlocks.md`). |
| — | Deliberate escrow-processor-not-escrowStore-seam read (`addMeso`) | PASS (documented) | `processor.go:1328-1345` (diff) — code comment explains reading `InFlightMesoDelta`/`EffectiveMesoByOwner` through the same `escrow.Processor` that arms the stake below, rather than through the fakeable `escrowStore` test seam, specifically so a test double cannot silently diverge from the arm path. Deliberate, documented trade-off, not a guideline violation. |
| — | Compare-and-set correctness (no read-then-write races) | PASS | `escrow/administrator.go` `CommitMesoStake` (403-431), `ClaimMesoForReturn` (479-507, row `FOR UPDATE` lock), `DeleteResolvedMeso` (580-593, condition inside the `DELETE` WHERE) — all documented and implemented as single-statement compare-and-set, consistent with the design rationale in the doc comments. |

## saga-orchestrator `saga` package

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| — | Single source of truth for reverse-walk routing | PASS (test-enforced) | `timer.go:147-181` `dispatchTimeoutRollbacks` switches on `s.SagaType()`; `reverseWalkSagaTypes` (`timer.go:163-169`) is a parallel list consumed only by `timer_test.go:134` (`TestEverySagaTypeWithAReverseWalkIsDispatchedOnTimeout`) to assert every entry dispatches something. Two enumeration sites still exist (`var` list and `switch`), but the test iterates the list and fails loudly if the switch and the list diverge — same shape of guard that this task added specifically because the *previous* pair of parallel lists (`CompensateFailedStep` routing vs. the old `if` chain) had silently diverged for `TradeStaging`. Acceptable. |
| DOM-20 | Table-driven tests | PASS | `timer_test.go:124-142` `for _, st := range reverseWalkSagaTypes { t.Run(string(st), …) }`. |
| — | Builder pattern for test setup | PASS | `timer_test.go:136` `NewBuilder().SetSagaType(st).SetInitiatedBy("test").Build()`. No `*_testhelpers.go` file added (`git diff --name-only` shows only `compensator_test.go`/`timer_test.go`, both ordinary `_test.go` files). |
| — | Reverse-walk dispatch is fire-and-forget/idempotent | PASS (documented) | `compensator.go:2470-2494` new `AwardMesos` arm guarded by `c.claimTradeRollback(s, step)` (per-step marker) before dispatching `AwardMesosAndEmit`, matching the idempotency reasoning already documented for the sibling arms. |

## Test-convention checklist (whole diff)

| Check | Status | Evidence |
|---|---|---|
| Builder pattern used for domain-model test setup | PASS | `escrow/administrator_test.go:37-43` `testItem` → `NewItemBuilder(...).SetTradeSlot(...).SetSource(...).SetSnapshot(...).Build()`; `saga/timer_test.go:136` `NewBuilder()...Build()`. |
| No `*_testhelpers.go` files | PASS | `git diff --name-only efec8332a~1..HEAD -- '*_test.go'` lists only `administrator_test.go`, `migration_test.go`, `processor_settlement_test.go`, `processor_staging_test.go`, `settlement_reconcile_test.go`, `compensator_test.go`, `timer_test.go` — no `_testhelpers.go`. `testDb`/`testTenant` helpers live inside `administrator_test.go` itself (lines 19-31), not a separate helpers file. |
| Migration test correctness/idempotency | PASS | `escrow/migration_test.go` — `TestMigrationDropsStaleColumns` (26-44) and `TestMigrationLiftsAnArmedStakeOutOfTheOldSlot` (74-149) both seed a `legacy*Entity` shape, run `Migration(db)` **twice** (149: "a second pass must not duplicate the lift") and assert the backfilled stake row count stays at 1 — directly exercises the `ON CONFLICT (id) DO NOTHING` idempotency claimed in `entity.go:390-396`, and the `m.HasColumn` early-outs (`entity.go:356-371`) that make a re-run safe on an already-migrated DB. |
| `libs/atlas-database/databasetest/failwrites.go` — new shared test lib, not a per-service `_testhelpers.go` | PASS | It is a genuine shared library addition under `libs/atlas-database/databasetest/`, mirroring the existing `NewInMemoryTenantDB` helper in the same package — not a service-local test-only constructor file the CLAUDE.md rule targets. |

## Findings

### F1 — MesoRefundEntity has no domain Model (BLOCKING, Important)

`entity.go:303-319` introduces `MesoRefundEntity`, a brand-new persisted concept for this task ("records what a trade_unwind took, so a failed unwind can restore it"). Every other entity in the package (`ItemEntity`, `MesoEntity`, `MesoStakeEntity`) has a paired immutable `*Model` type with private fields + getters + a `Make*(Entity) (Model, error)` function (`model.go:19-223`), per `file-responsibilities.md` (`entity.go` → "Provides `Make(Entity) (Model, error)` and `Model.ToEntity()`"; `model.go` → "immutable domain objects with private fields and accessor methods"). `MesoRefundEntity` has neither:

- `RecordMesoRefund` (`administrator.go:154-169`) constructs the entity inline and writes it directly.
- `RestoreMesoRefunds` (`administrator.go:181-217`) reads `var rows []MesoRefundEntity` straight off GORM (line 185) and operates on the raw entity fields (`r.TenantId`, `r.Amount`, etc., lines 194-199) with no `Make`/model step at all.
- `DiscardMesoRefunds` (`administrator.go:221-226`) deletes by raw filter, never touching a model.
- No `MesoRefundModel` type, no `MakeMesoRefund` function, anywhere in `model.go`.

This breaks the file-responsibilities pattern for the one new domain concept this diff actually owns end-to-end (as opposed to `MesoStakeEntity`, which correctly got `MesoStakeModel` + `MakeMesoStake`). It is a structural deviation, not a prevalence argument — `MesoStakeEntity` in the very same diff proves the correct shape was known and applied elsewhere. Per audit rules, a File-Responsibilities violation defaults to Important severity and blocks PASS.

**Fix:** add `MesoRefundModel` (private fields: id, transactionId, tenantId/region/major/minor, roomId, ownerId, amount, createdAt) with getters, and `MakeMesoRefund(MesoRefundEntity) (MesoRefundModel, error)` in `model.go`; have `RestoreMesoRefunds`/`DiscardMesoRefunds` route through it instead of touching `MesoRefundEntity` fields directly.

## Summary

### Blocking (must fix)
- **F1 / DOM-03:** `MesoRefundEntity` (`escrow/entity.go:303-319`) has no `MesoRefundModel` and no `MakeMesoRefund`; `administrator.go:181-217` manipulates the raw entity directly, bypassing the immutable-model layer every sibling entity in the package uses.

### Non-Blocking (should fix / observations)
- The escrow package's whole-package convention of manually threading `tenantId` through every provider/administrator signature instead of the automatic GORM-callback filtering documented in `patterns-multitenancy-context.md` predates this diff and is applied consistently by the new code; flagged for awareness only, not attributed to this change.
- `timer.go`'s `reverseWalkSagaTypes` list and its `dispatchTimeoutRollbacks` switch are two separate enumerations kept in sync only by a test (`timer_test.go:134`) rather than by construction (e.g., driving the switch off the list). Test-enforced, so not a blocking finding, but a future addition that updates one and not the other would only be caught if the test suite runs.

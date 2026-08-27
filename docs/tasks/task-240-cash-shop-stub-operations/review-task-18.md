# Review: Task 18 — the ring-pairing domain

Commit range reviewed: `20746105d..8806b742e` (single commit `8806b742e feat(cashshop): add the ring pairing domain`).

Brief: `.superpowers/sdd/plan/task-18-brief.md`
Report: `.superpowers/sdd/plan/task-18-report.md`

## Scope

`git diff --stat 20746105d..8806b742e`:

```
services/atlas-cashshop/atlas.com/cashshop/main.go                 |   3 +-
services/atlas-cashshop/atlas.com/cashshop/ring/administrator.go   |  60 ++++++
services/atlas-cashshop/atlas.com/cashshop/ring/administrator_test.go | 207 +++++++
services/atlas-cashshop/atlas.com/cashshop/ring/entity.go          |  48 +++++
services/atlas-cashshop/atlas.com/cashshop/ring/model.go           |  76 +++++++
services/atlas-cashshop/atlas.com/cashshop/ring/processor.go       |  53 +++++
services/atlas-cashshop/atlas.com/cashshop/ring/provider.go        |  46 +++++
7 files changed, 492 insertions(+), 1 deletion(-)
```

Scope matches the brief's file list exactly (six new files under `ring/`, one line in `main.go`). No unrelated files touched.

## 1. FR-RING-4 atomicity — reproduced independently

`ring/administrator.go:56` inserts both `Entity` rows in a single `db.Create(&rows)` call over the `[]Entity` slice (one multi-row `INSERT`).

I did not take the implementer's mutation claim on trust. I mutated `CreatePair` myself:

```go
if err := db.Create(&rows[0]).Error; err != nil {
    return uuid.Nil, err
}
if err := db.Create(&rows[1]).Error; err != nil {
    return uuid.Nil, err
}
```

and ran `go test ./ring/... -run TestCreatePairIsAtomic -v`:

```
=== RUN   TestCreatePairIsAtomic
    administrator_test.go:197: GetByCharacterId(42) = 1 rows, want 0 (partial pair persisted)
--- FAIL: TestCreatePairIsAtomic (0.00s)
FAIL
```

The test genuinely discriminates: with two separate `Create` calls, character 42's half survives the injected failure on character 77's row, and the test catches exactly that. I reverted the mutation (`git checkout -- ring/administrator.go`) and reran the full package suite, which passes (`go test ./ring/... -v` — all PASS, `git status --porcelain` on `ring/` is clean).

The test's mechanism (`failWritesForCharacter` in `administrator_test.go:141-172`, a targeted `Before("gorm:create")` callback keyed on `CharacterId == 77`, inspecting `*Entity`/`[]Entity`/`*[]Entity` dest shapes) is a deliberate departure from the brief's suggested "wrap in a transaction and roll back" shape — correctly so, since a rollback-based test would pass under *either* implementation (batched or two-call) and would not discriminate. This is documented candidly in the report and in a code comment (`administrator_test.go:132-140`). **PASS.**

## 2. Processor design choice

The brief specified only free functions (`CreatePair`, `GetByCharacterId`, `GetById`); the `Processor` interface shape (`ring/processor.go:16-20`) was the implementer's own design, flagged as a concern in the report.

Compared against `wishlist/processor.go`, the ring processor omits the `message.Buffer`/`AndEmit`/`database.ExecuteTransaction` machinery entirely. Taken against wishlist alone this looks like a divergence. But `purchaserecord/processor.go` — a sibling package in the same directory, also a write-inside-caller's-transaction + tenant-scoped-read shape with no Kafka surface — is structurally identical to what was produced here: same field set (`l`, `ctx`, `db`, `t`), same `NewProcessor` signature, same pattern of the write method taking an explicit `db` argument ("caller-supplied handle... called inside an existing transaction") while the read methods use `p.db.WithContext(p.ctx)`. `ring` has no Kafka topic and no downstream event consumer yet (matches the brief, which describes only "a future effects consumer reads the pair over REST"), so the wishlist emit machinery would be a needless divergence in the other direction. Judged against the closer-matching precedent in the same package, this is not a finding. **PASS** (with the implementer's self-flagged concern read and accepted).

## 3. Multi-tenancy

- `provider.go:16` — `byCharacterIdProvider`: `db.Where("tenant_id = ? AND character_id = ?", ...)`.
- `provider.go:26` — `byIdProvider`: `db.Where("tenant_id = ? AND id = ?", ...)`.
- `administrator.go:32,44` — both `Entity` rows in `CreatePair` set `TenantId: tenantId` from the caller-supplied argument.
- `processor.go:34,44,48,52` — `ProcessorImpl` resolves `t tenant.Model` via `tenant.MustFromContext(ctx)` once in `NewProcessor` and threads `p.t.Id()` into every free-function call; no path bypasses it.

Test table: `administrator_test.go:111-119` (`another tenant sees nothing`) creates a fresh `uuid.New()` tenant and asserts `GetByCharacterId` returns zero rows for character 42, who has rings under tenant `a`. Present and matches the brief's Step 1 table. **PASS.**

## 4. Interfaces produced vs. brief

All read directly from source, compared line-for-line against the brief's block:

- `Type`, `TypeCouple`, `TypeFriendship` — `model.go:12-17`. Match.
- `State`, `StateActive`, `StateBroken`, `StateExpired` — `model.go:21-26`. Match.
- `func Migration(db *gorm.DB) error` — `entity.go:10-12`. Match.
- `func CreatePair(db *gorm.DB, tenantId uuid.UUID, ringType Type, a Half, b Half) (uuid.UUID, error)` — `administrator.go:25`. Match.
- `type Half struct { CharacterId, AssetId, ItemTemplateId uint32 }` — `administrator.go:11-15`. Match.
- `func GetByCharacterId(db *gorm.DB, tenantId uuid.UUID, characterId uint32) ([]Model, error)` — `provider.go:35`. Match.
- `func GetById(db *gorm.DB, tenantId uuid.UUID, id uuid.UUID) (Model, error)` — `provider.go:40`. Match.

All five brief test-table subtests are present verbatim in `administrator_test.go:26-129` (`creates two halves sharing a pair id`, `the partner half mirrors it`, `friendship pairs are distinguishable`, `another tenant sees nothing`, `a character with no rings is not an error`). **PASS.**

## 5. Migration wiring

`main.go` diff:

```go
-	db := database.Connect(l, database.SetMigrations(wallet.Migration, wishlist.Migration, compartment.Migration, asset.Migration, opening.Migration, purchaserecord.Migration, coupon.Migration, batch.Migration, redemption.Migration, outboxlib.Migration, database.IdempotencyMigration))
+	db := database.Connect(l, database.SetMigrations(wallet.Migration, wishlist.Migration, compartment.Migration, asset.Migration, opening.Migration, purchaserecord.Migration, ring.Migration, coupon.Migration, batch.Migration, redemption.Migration, outboxlib.Migration, database.IdempotencyMigration))
```

`ring.Migration` is in the real `database.SetMigrations` call (currently at `main.go:65`, not the brief's stale `main.go:62` — the implementer correctly re-grepped the live call site rather than trusting the plan-time line number). Import added at the top of `main.go`. **PASS.**

## 6. Shared constants library

`git diff --stat 20746105d..8806b742e -- libs/atlas-constants/` is empty — nothing added. `Type`/`State` remain service-local typed strings per the plan-time constants check. **PASS.**

## Standard backend expectations

- **Model/entity separation**: `Entity` (`entity.go:21-32`, exported gorm-tagged fields, table `cash_rings`) is distinct from `Model` (`model.go:30-40`, unexported fields, accessor methods only — `Id()`, `PairId()`, `CharacterId()`, `PartnerCharacterId()`, `AssetId()`, `ItemTemplateId()`, `Type()`, `State()`, `CreatedAt()`). `Make(Entity) (Model, error)` converts one to the other (`entity.go:36-48`). Correct shape. **PASS.**
- **No test-only constructors / `*_testhelpers.go`**: no such file added; `testDatabase(t)` lives inside `administrator_test.go` itself and calls the pre-existing shared `databasetest.NewInMemoryTenantDB` helper (confirmed pre-existing at `libs/atlas-database/databasetest/testdb.go:22`, introduced in commit `cbf198138`, task-205 — not new in this commit). **PASS.**
- **Builder pattern for test setup**: not applicable — `Half` is already a flat, brief-specified public struct with exported fields; there is no multi-field construction complex enough to warrant a builder, and the test constructs `Half{...}` literals directly matching the brief's own test table values. Not a finding.

## Findings

None blocking. None non-blocking beyond what is already self-flagged and resolved above (the processor shape, addressed in section 2).

## Not evaluable

- The future ring-purchase transaction path that will actually call `CreatePair`/`Processor.CreatePair` inside a real multi-statement transaction (wallet debit + two cash assets + pair insert) does not exist yet — this task is domain-only, per the brief. Cannot evaluate the real caller-side transaction wiring because it has not been written; nothing in this diff claims to provide it.
- The REST/Kafka consumer side that will read a pair (`GetById` from "a future effects consumer") is out of scope for this task and not present in the diff.

## Verdict rationale

Every brief interface is present with the exact signature. The load-bearing FR-RING-4 atomicity claim was independently reproduced by mutation (batched insert → two separate `Create` calls) and confirmed the test fails for the right reason, then the mutation was reverted and the tree left clean. Multi-tenancy is scoped on every query and covered by a test. The migration is wired at the real call site. No constants leaked into the shared library. The self-flagged processor design choice follows the closer sibling precedent (`purchaserecord`) rather than diverging from it.

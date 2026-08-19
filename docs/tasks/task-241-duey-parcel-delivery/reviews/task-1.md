# Review — Task 1: atlas-parcel module skeleton, entity and migration

Range: `b8b0bea22..c16e1e256` (1 commit, `c16e1e256`)
Reviewer: atlas-reviewer (Sonnet)

## Scope

Reviewed the full diff (9 files, 725 insertions): `go.work`, `go.work.sum`,
`services/atlas-parcel/atlas.com/parcel/{go.mod,go.sum,main.go,rest/handler.go,
parcel/asset_data.go,parcel/entity.go,parcel/entity_test.go}`. Cross-referenced
against the referenced read-only patterns (`atlas-mts/main.go`,
`atlas-mts/rest/handler.go`, `atlas-merchant/frederick/entity.go`,
`atlas-merchant/kafka/message/asset/kafka.go`) and against
`docs/tasks/task-241-duey-parcel-delivery/design.md` §3 (the entity/index
contract, source of truth over the brief's paraphrase). Scope matches the
brief; no drift.

## Findings

### 1. PASS — Entity field set and types match design §3 exactly

`parcel/entity.go:27-58` reproduces the design's struct verbatim: field names,
types (`WorldId byte` — matches the `byte` convention used across
`atlas-mts`, e.g. `services/atlas-mts/atlas.com/mts/serial/entity.go:43`),
`ItemId *uint32` pointer rationale, `Message`/`MesoAmount`/`FeePaid`
nullability shape, `ResolvedAt`/`LastNotified` as `*time.Time`.

### 2. PASS — Index shape matches design §3's three justified indexes, verified against real migrator output

Design specifies exactly `(tenant_id, recipient_id, status)`,
`(tenant_id, sender_id, status)`, `(tenant_id, status, expires_at)`. I ran a
scratch `db.Migrator().GetIndexes(&Entity{})` against the actual migration
(removed afterward, tree confirmed clean) and got:

```
idx_parcels_recipient columns=[tenant_id recipient_id status]
idx_parcels_sender    columns=[tenant_id sender_id status]
idx_parcels_sweep     columns=[tenant_id status expires_at]
```

Exact match, composite (not three separate single-column indexes).

### 3. PASS (verified, not just "looks correct") — `Id uuid.UUID` vs. `gorm.Model.ID uint` name collision resolves correctly

`Entity` embeds `gorm.Model` (which carries `ID uint`) and also declares an
explicit `Id uuid.UUID` field (`entity.go:29`) tagged `primaryKey`. Go
identifier `Id` != `ID`, so this isn't a compile-time shadow, and both fields
independently snake_case to column `id` — a real risk of ambiguous/duplicate
primary-key columns. I ran a scratch test that migrates, inserts, and
re-fetches an entity: GORM's schema builder resolves the collision by column
name (case-insensitive) in favor of the shallower/explicit field, producing a
single `id` column of type `uuid`, and round-trips the UUID correctly
(`gormModelID` stays `0`, unused). This exactly mirrors the already-shipped
`atlas-merchant/frederick/entity.go` pattern the brief named as the source —
so it is proven-safe by precedent and now also proven-safe by direct test,
not just assumed.

### 4. PASS — `AssetData` is a byte-for-byte move, not a cross-service import

`diff services/atlas-merchant/atlas.com/merchant/kafka/message/asset/kafka.go
services/atlas-parcel/atlas.com/parcel/parcel/asset_data.go` differs only in
package name and an added doc comment. All 30 fields, `WithQuantity`,
`Value`, `Scan` are identical. Satisfies CLAUDE.md's "straightforward move,
not a cross-service import."

### 5. PASS — Tenant scoping follows the same seam as `atlas-mts`/`atlas-merchant`

`rest/handler.go:74` (`RegisterHandler`) and `:88` (`RegisterInputHandler`)
both wrap the handler in `server.ParseTenant(...)`, identical to
`atlas-mts/rest/handler.go`'s shape. `TenantId` is the first-priority column
in all three composite indexes (finding 2), so a tenant-scoped mailbox/sweep
query is index-covered from this commit forward. Multi-tenancy is
structurally present, not deferred.

### 6. PASS — `main.go` bootstrap shape genuinely admits the additive registrations tasks 4/15/23/24 need

- `db := database.Connect(l, database.SetMigrations(parcel.Migration))` is
  already bound and in scope; Task 4 replacing `_ = db` with a real
  `AddRouteInitializer` costs nothing structural.
- `cmf := consumer.GetManager().AddConsumer(...)` is currently discarded
  (`_ = ...`) — Task 15 assigning it to a named variable and chaining
  `InitConsumers`/`InitHandlers` matches the exact shape used in
  `atlas-mts/main.go:85-96`.
- `srv := server.New(l)....AddRouteInitializer(...)` is built, then
  `.Run()` is called; inserting another `.AddRouteInitializer(...)` call
  before `.Run()` (Task 4) is mechanical.
- Periodic-task registration (`task.NewPeriodicTask(l, rt.Context(), db,
  interval)` / `.Start()` / `rt.TeardownFunc(.Stop)`) as used at
  `atlas-mts/main.go:105-107` for tasks 23/24 needs only `db`, `rt.Context()`,
  `rt.TeardownFunc` — all already present and unconsumed.
- `serviceName = "atlas-parcel"` and `consumerGroupId =
  consumergroup.Resolve("Parcel Service")` (`main.go:16,18`) are literal,
  statically readable — satisfies the stated Task 5 dependency on reading
  these values out of `main.go` source.

### 7. PASS — TDD evidence is credible

Report's RED log shows compile failures for `Entity`/`Migration`/status
constants/timer constants — genuinely would not compile without this commit's
code. GREEN log matches `go test ./... -v` output for the four new tests. I
did not re-run these (per instructions); the failure modes named are
consistent with the actual code shape.

### 8. Non-blocking — brief's go.mod guidance ("replace-free") doesn't match actual repo convention, and the implementer correctly deviated

Brief step 3 says go.mod should use "the same `replace`-free shared-lib
requirements" atlas-mts's go.mod uses. `services/atlas-mts/atlas.com/mts/
go.mod` in fact has 17 `replace` directives, same count/shape as the new
`services/atlas-parcel/atlas.com/parcel/go.mod`. The implementer noted this
discrepancy in the report and followed the actual established pattern
(`atlas-mts`/`atlas-buddies`) instead of the brief's inaccurate paraphrase.
Correct call, but worth recording so the brief text gets fixed for future
tasks referencing it.

### 9. Non-blocking — `go build`/`go vet`/`gofmt` re-verified clean

Ran `go build ./... && go vet ./... && gofmt -l .` in
`services/atlas-parcel/atlas.com/parcel` — clean, matches report. (Not
re-running the implementer's *tests* per instructions, but build/vet/fmt are
cheap sanity gates a reviewer should independently confirm rather than trust
blind.)

## Not evaluable

- Message length (100 char), meso ceiling (100,000,000), mailbox capacity
  (10 pending), Quick Delivery Ticket item id (5330000), Duey NPC id
  (9010009) — none of these are enforced or referenced in this task's diff.
  That's correct per brief scope (Task 1 is entity/migration only; validation
  and business rules land in Tasks 2/3), not a gap in this unit, but I could
  not evaluate whether the entity's *types* (e.g. `uint32 MesoAmount`,
  `string Message`) are sufficient to hold those ceilings without
  independently re-deriving Task 2/3's validation code, which is out of this
  task's diff.
- Postgres-specific behavior (the `uuid`/`jsonb` column types, the `Id`/`ID`
  collision resolution) was verified against SQLite in-memory (matching the
  test suite's own approach) — not against a live Postgres instance. GORM's
  schema-builder collision resolution (finding 3) is driver-agnostic
  (happens before SQL generation), so this is low-risk, but a Postgres-level
  confirmation was outside this task's own test surface and mine.

## Verdict

APPROVED_WITH_FINDINGS — no blocking defects. Entity, indexes, tenant
scoping, and `main.go`'s extensibility for later tasks all verified against
the design and, where verification was possible, against actual GORM/SQLite
runtime behavior rather than static reading alone.

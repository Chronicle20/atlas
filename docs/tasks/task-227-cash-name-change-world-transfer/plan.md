# Cash Name Change & World Transfer — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the Cash/0540 character-imprint services — name change and world transfer — end to end with GMS-authentic deferred application, an operator-only cancel path, and full packet coverage for the eight-row check/cancel family.

**Architecture:** One `character_pending_changes` table in atlas-character carries both request types through a `PENDING` → terminal lifecycle; two partial unique indexes *are* the one-pending-per-type rule and the name soft-reservation. Requests are created from atlas-channel (item use or cash-shop purchase) over REST, applied at the character's `LOGOUT` status event, and every non-`APPLIED` exit refunds the coupon and notifies the player. Name change applies locally; world transfer runs a four-step compensating saga through atlas-saga-orchestrator.

**Tech Stack:** Go 1.24 microservices (immutable models + Builder, processor Interface+Impl, GORM entities, JSON:API via api2go, Kafka with `message.Buffer`/outbox), `libs/atlas-packet` codecs, atlas-ui (React 19 + TanStack Query + shadcn/ui).

**Spec:** [`design.md`](design.md) (PRD: [`prd.md`](prd.md))

## Global Constraints

- Multi-tenancy: every table, query, Kafka message and REST call is tenant-scoped via `tenant.MustFromContext(ctx)`. No cross-tenant transfer is representable.
- Version gates in **new** code use the `MajorAtLeast` idiom — never a raw `t.MajorVersion() >= N`. Existing sibling branches in `GetCashSlotItemType` are explicitly out of scope (design §10).
- Client wire values (opcodes, dispatcher mode bytes, reason codes) are config-resolved through the tenant socket-config `options` tables (DOM-25) — never literals in Go.
- Before defining any new domain type/alias/constant, check `libs/atlas-constants/` for an existing equivalent (DOM-21). `item.ClassificationCharacterImprints` (540) already exists at `libs/atlas-constants/item/constants.go:103`.
- Test setup uses the project's Builder pattern. No `*_testhelpers.go` files with test-only constructors.
- Goroutines are spawned via `routine.Go` only (`tools/goroutine-guard.sh`).
- No `// TODO`, stubbed handlers, or 501s in landed commits.
- Never write literal home/absolute paths into committed files — repo-relative only.
- Server-side reason strings come from the closed taxonomy of design §6. No free text.
- The account second-password (SPW) field carried by the check packets must never reach a log line; the decoded struct's `String()` redacts it.
- Expiry default: **7 days** (168h).
- Name validation rules (unchanged, owned by `ProcessorImpl.CheckNameValidity`): length 3–12, regex `^[A-Za-z0-9぀-ゟ゠-ヿ一-龯]{3,12}$`.

## Reason taxonomy (design §6 — the closed set)

`name_invalid_length` · `name_invalid_charset` · `name_taken` · `name_reserved` ·
`already_pending` · `world_same` · `world_unknown` · `world_full` · `no_character_slot` ·
`banned` · `is_guild_master` · `is_gm` · `in_family` · `trade_open` · `merchant_open` ·
`mts_listings_open` · `operator_cancelled` · `expired` · `saga_failed`

## Deviations from the design, stated up front

- **design §3.3 calls atlas-character self-consuming its own status topic "a new pattern."** It is not — `kafka/consumer/character/consumer.go:24` already subscribes `character_event_status` and `handleLevelChangedStatusEvent` / `handleJobChangedStatusEvent` already handle it. The applier is a third handler on an existing consumer, not new machinery. This makes design risk **R4** substantially smaller than stated.
- **PRD §5's `GET /characters/name-availability` is not created.** Design §3.4 extends the existing `GET /characters/name-validity` instead. Confirmed present at `services/atlas-character/atlas.com/character/character/name_validity_resource.go`.
- **PRD FR-4.5's inventory re-home step is a no-op** and produces no saga step (design §1.5 — atlas-inventory has no world column).

---

## File Structure

**New — atlas-character** (`services/atlas-character/atlas.com/character/`):
- `pending_change/entity.go` — GORM entity, `Migration` incl. the two partial unique indexes
- `pending_change/model.go`, `builder.go` — immutable model + Builder
- `pending_change/administrator.go` — writes (create / transition)
- `pending_change/provider.go` — reads (by id, by character, pending-by-name, sweep, unnotified)
- `pending_change/processor.go` — Interface+Impl: create, resolve, apply, sweep
- `pending_change/eligibility.go` — world-transfer gate evaluation
- `pending_change/producer.go` — `PENDING_CHANGE_CREATED` / `PENDING_CHANGE_RESOLVED` / `WORLD_CHANGED` providers
- `pending_change/resource.go`, `rest.go` — JSON:API surface
- `pending_change/task.go` — expiry sweep ticker
- `configuration/` (model, rest, requests, registry) — imprint-config client, modelled on `services/atlas-trades/atlas.com/trades/configuration/`
- `kafka/message/pending_change/kafka.go` — the new message contract

**Modified — atlas-character**: `character/processor.go` (NameScope + reservation), `character/name_validity_resource.go` (scope param + `reserved`), `kafka/consumer/character/consumer.go` (applier handlers), `main.go` (migration, resource, ticker, config registry).

**New — atlas-tenants**: `imprint-configs` configuration resource (handler, provider, kafka event types, seed dir + default file).

**Modified — libs/atlas-saga**: `model.go` — `WorldTransfer` saga type, four actions + payloads.

**Modified — atlas-saga-orchestrator**: `saga/model.go` (aliases + payload unmarshal), `saga/handler.go` (4 handlers), `saga/compensator.go` (compensations), `saga/event_acceptance.go` (event kinds), plus a `pending_change/` REST-client package for the character-world update.

**Modified — atlas-channel**: `socket/handler/character_cash_item_use.go` (540 branch + classifier fix), `socket/handler/cash_shop_operation.go` (two real bodies), new `socket/handler/cash_shop_check_name_change.go` + `cash_shop_check_world_transfer.go`, new `pendingchange/` REST client, new `kafka/consumer/pendingchange/`.

**New codecs — libs/atlas-packet**: `cash/serverbound/check_name_change_possible.go`, `cash/serverbound/check_transfer_world_possible.go`, and clientbound bodies for the three `CASHSHOP_CHECK_*` arms plus the three `CANCEL_*` packets (exact file placement is settled by the derivation pass of Task 1, since `CASHSHOP_CHECK_NAME_CHANGE` may be a dispatcher family).

**Modified — atlas-guilds / atlas-buddies / atlas-rankings / atlas-mts**: a `NAME_CHANGED` status-event consumer each.

**New — atlas-ui**: `src/services/api/pending-changes.service.ts`, `src/lib/hooks/api/usePendingChanges.ts`, `src/components/features/characters/PendingChangesPanel.tsx`, `src/components/features/characters/CancelPendingChangeDialog.tsx`, plus tests. Modified: `src/pages/CharacterDetailPage.tsx`.

---

## Phase A — Derivation (blocking; nothing downstream may start without it)

### Task 1: Resolve the symbol transposition and the 540 prefix→feature split

Design §1.8 found `@0x47359c`, symboled `CCashShop::SendCheckNameChangePossiblePacket`, building `COutPacket(18)` = `0x012`, which the matrix records as `WORLD_TRANSFER`. Either the IDB symbols are transposed or the matrix is. Writing codecs against a transposed pair yields two structurally-plausible decoders that are silently swapped and whose byte-fixture tests both pass. Nothing else in this plan may begin.

**Files:**
- Create: `docs/tasks/task-227-cash-name-change-world-transfer/derivation.md`
- Create: `docs/tasks/task-227-cash-name-change-world-transfer/coverage-manifest.yaml`
- Read only: `docs/packets/audits/STATUS.md` rows 157, 161, 204, 409, 413, 476, 528, 532

**Interfaces:**
- Produces: `derivation.md` §1 — the authoritative `NAME_TRANSFER` / `WORLD_TRANSFER` opcode assignment per version, each with an IDB address and the decompiled `COutPacket(n)` line quoted. §2 — the field read order of each of the eight ops per version. §3 — which of `5400xxx` / `5401xxx` is name-change and which is world-transfer, with its evidence route. §4 — the reason-code enumeration for the two `*_POSSIBLE_RESULT` packets (OQ-7). §5 — a yes/no on whether `CASHSHOP_CHECK_NAME_CHANGE` (row 409, three receivers incl. `OnCheckDuplicatedIDResult`) is a dispatcher family per `docs/packets/DISPATCHER_FAMILY.md`.
- Produces: `coverage-manifest.yaml` declaring exactly the 59 cells.

- [ ] **Step 1: Resolve the IDA session by binary name**

`select_instance(port)` is dead. Use `mcp__ida-pro__idb_list`, find the row whose binary name is the GMS v83 client, and pass that session as the `database` parameter to every subsequent call.

- [ ] **Step 2: Read both send functions and record the actual opcode**

```
mcp__ida-pro__func_query(database=<v83 session>, name_regex="CCashShop::SendCheck(NameChange|TransferWorld)PossiblePacket")
mcp__ida-pro__decompile(database=<v83 session>, address=<each>)
```

Quote the literal `COutPacket(n)` argument and the full `Encode*` sequence for each. Do **not** trust the symbol name — `docs/packets/PROCESS.md` and the standing "don't infer feature from IDB names" rule both say the read order is the evidence.

- [ ] **Step 3: Disambiguate by call site**

Find each function's callers (`mcp__ida-pro__xrefs_to`). A send reached from `CCashShop::CheckTransferWorldPossible` (`@0x4734e5`, the function that formats `"Guild Master can not transfer worlds."`) is the world-transfer send regardless of its symbol. Record the call chain in `derivation.md` §1.

- [ ] **Step 4: Reconcile with the matrix, and amend whichever is wrong**

If the IDB read order proves the matrix's v83 `NAME_TRANSFER`=`0x010` / `WORLD_TRANSFER`=`0x012` assignment is transposed, amend `docs/packets/audits/` — do not "pick one." Repeat Steps 2–3 on each other version's IDB (v83/v84/v87/v92/v95 for `WORLD_TRANSFER`; those plus jms_v185 for `NAME_TRANSFER`) and record per-version addresses.

- [ ] **Step 5: Re-derive the whole `ClassificationCharacterImprints` classifier arm**

Read the client's `get_cashslot_item_type` arm for classification 540 on v83 and v95. Record which item-id prefixes it recognises and what enum value each yields. `character_cash_item_use.go:1132-1138` is an exact duplicate of the `5401` branch above it — the client's own arm settles whether a third prefix was intended.

- [ ] **Step 6: Settle the prefix→feature assignment**

Two independent routes; use whichever answers:
(a) query atlas-data for classification-540 templates per version and read their `String.wz` names;
(b) the client classifier arm from Step 5 cross-referenced against which `CashSlotItemType` each of `CCashShop::OnBuyNameChange` / the transfer path consumes.
Record the route used. If neither answers, stop and report BLOCKED — do not guess an item id.

- [ ] **Step 7: Derive the reason-code enumerations (OQ-7)**

Decompile `CCashShop::OnCheckNameChangePossibleResult` and `CCashShop::OnCheckTransferWorldPossibleResult`; record every code the client branches on and the string it renders, per version. This is the input the codec tasks map design §6's server-side reasons onto.

- [ ] **Step 8: Write coverage-manifest.yaml**

Follow the format of `docs/tasks/task-206-cash-shop-coupon-codes/coverage-manifest.yaml`: `ops`, `versions`, `fields`, `out_of_scope`. Declare exactly these 59 cells:

| row | cells | versions |
|---|---|---|
| `CANCEL_NAME_CHANGE_RESULT` | 8 | v61 v72 v79 v83 v84 v87 v92 v95 |
| `CANCEL_TRANSFER_WORLD_RESULT` | 8 | v61 v72 v79 v83 v84 v87 v92 v95 |
| `CANCEL_NAME_CHANGE_BY_OTHER` | 7 | v72 v79 v83 v84 v87 v92 v95 |
| `CASHSHOP_CHECK_NAME_CHANGE` | 9 | v48 v61 v72 v79 v83 v84 v87 v92 v95 |
| `CASHSHOP_CHECK_TRANSFER_WORLD_POSSIBLE_RESULT` | 10 | v48 v61 v72 v79 v83 v84 v87 v92 v95 jms_v185 |
| `CASHSHOP_CHECK_NAME_CHANGE_POSSIBLE_RESULT` | 6 | v79 v83 v84 v87 v92 v95 |
| `NAME_TRANSFER` (sb) | 6 | v83 v84 v87 v92 v95 jms_v185 |
| `WORLD_TRANSFER` (sb) | 5 | v83 v84 v87 v92 v95 |

Under `out_of_scope`, list the `character_pending_changes` table, the saga, the atlas-ui panel and the `NAME_CHANGED` consumers — they are this task's non-packet surface and would otherwise read as unclaimed changes.

- [ ] **Step 9: Commit**

```bash
git add docs/tasks/task-227-cash-name-change-world-transfer/derivation.md \
        docs/tasks/task-227-cash-name-change-world-transfer/coverage-manifest.yaml \
        docs/packets/audits/
git commit -m "docs(task-227): derive imprint check/cancel opcodes and 540 prefix split"
```

---

## Phase B — atlas-character: the pending-change record and its lifecycle

All Phase B work is inside `services/atlas-character/atlas.com/character/`. Paths below are relative to that directory unless stated otherwise.

### Task 2: The `pending_change` entity, migration, and the two partial unique indexes

The indexes are not belt-and-braces on application logic — they *are* FR-2.3 and FR-3.3. Application code catches the unique violation and maps it to a reason; it never pre-checks and races.

**Files:**
- Create: `pending_change/entity.go`
- Create: `pending_change/entity_test.go`
- Modify: `main.go:72` (add `pending_change.Migration` to `database.SetMigrations`)

**Interfaces:**
- Produces: `pending_change.Migration(db *gorm.DB) error`; unexported `entity` with `TableName() "character_pending_changes"`; exported status and type constants.

- [ ] **Step 1: Write the failing migration test**

Create `pending_change/entity_test.go`. This test needs a real Postgres — the partial indexes cannot be exercised on SQLite. Follow the container/DSN setup already used by `character/kafka_integration_test.go` in this module.

```go
package pending_change

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

func TestMigrationCreatesPartialUniqueIndexes(t *testing.T) {
	db := newTestDB(t) // same helper style as character/kafka_integration_test.go
	if err := Migration(db); err != nil {
		t.Fatalf("Migration: %v", err)
	}
	if err := Migration(db); err != nil {
		t.Fatalf("Migration is not idempotent: %v", err)
	}

	tid := uuid.New()
	name := "Alpha"
	lower := "alpha"
	mk := func(charId uint32, status string) *entity {
		return &entity{
			Id: uuid.New(), TenantId: tid, CharacterId: charId,
			Type: TypeNameChange, Status: status,
			RequestedName: &name, RequestedNameLower: &lower,
			SourceWorldId: world.Id(0),
			CreatedAt:     time.Now(), ExpiresAt: time.Now().Add(time.Hour),
		}
	}

	if err := db.Create(mk(1, StatusPending)).Error; err != nil {
		t.Fatalf("first pending insert: %v", err)
	}
	// Same name, different character, still PENDING -> reservation index must reject.
	if err := db.Create(mk(2, StatusPending)).Error; err == nil {
		t.Fatal("expected reservation unique violation for a duplicate pending name")
	}
	// Same name once the first is terminal -> allowed.
	if err := db.Model(&entity{}).Where("character_id = ?", uint32(1)).
		Update("status", StatusCancelled).Error; err != nil {
		t.Fatalf("transition: %v", err)
	}
	if err := db.Create(mk(2, StatusPending)).Error; err != nil {
		t.Fatalf("expected reservation released after terminal transition: %v", err)
	}
}

func TestMigrationEnforcesOnePendingPerType(t *testing.T) {
	db := newTestDB(t)
	if err := Migration(db); err != nil {
		t.Fatalf("Migration: %v", err)
	}
	tid := uuid.New()
	wid := world.Id(1)
	mk := func() *entity {
		return &entity{
			Id: uuid.New(), TenantId: tid, CharacterId: 7,
			Type: TypeWorldTransfer, Status: StatusPending,
			DestinationWorldId: &wid, SourceWorldId: world.Id(0),
			CreatedAt:          time.Now(), ExpiresAt: time.Now().Add(time.Hour),
		}
	}
	if err := db.Create(mk()).Error; err != nil {
		t.Fatalf("first insert: %v", err)
	}
	if err := db.Create(mk()).Error; err == nil {
		t.Fatal("expected one-pending-per-type unique violation")
	}
}
```

- [ ] **Step 2: Run the tests and confirm they fail**

Run: `cd services/atlas-character/atlas.com/character && go test ./pending_change/ -run TestMigration -v`
Expected: FAIL — package `pending_change` does not exist.

- [ ] **Step 3: Write the entity and migration**

```go
package pending_change

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// Request types.
const (
	TypeNameChange    = "NAME_CHANGE"
	TypeWorldTransfer = "WORLD_TRANSFER"
)

// Lifecycle statuses. Transitions out of StatusPending are one-way; a terminal
// record is never reopened (FR-2.2).
const (
	StatusPending   = "PENDING"
	StatusApplied   = "APPLIED"
	StatusCancelled = "CANCELLED"
	StatusRejected  = "REJECTED"
	StatusExpired   = "EXPIRED"
)

// Migration runs AutoMigrate, then creates the two partial unique indexes by
// raw DDL. GORM tags cannot express a WHERE clause, and these indexes are the
// mechanism behind FR-2.3 (one pending request per type) and FR-3.3 (the name
// soft reservation) — not a redundant guard on application code. Idempotent.
func Migration(db *gorm.DB) error {
	if err := db.AutoMigrate(&entity{}); err != nil {
		return err
	}
	stmts := []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_pc_one_pending_per_type
		   ON character_pending_changes (tenant_id, character_id, type)
		   WHERE status = 'PENDING'`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_pc_name_reservation
		   ON character_pending_changes (tenant_id, requested_name_lower)
		   WHERE status = 'PENDING' AND type = 'NAME_CHANGE'`,
	}
	for _, s := range stmts {
		if err := db.Exec(s).Error; err != nil {
			return err
		}
	}
	return nil
}

type entity struct {
	Id          uuid.UUID `gorm:"type:uuid;primaryKey;not null"`
	TenantId    uuid.UUID `gorm:"not null;index:idx_pc_tenant_char,priority:1"`
	CharacterId uint32    `gorm:"not null;index:idx_pc_tenant_char,priority:2"`
	Type        string    `gorm:"not null"`
	Status      string    `gorm:"not null;index:idx_pc_status"`

	// RequestedNameLower is a stored column rather than a functional index on
	// lower(requested_name) so the reservation lookup and idx_pc_name_reservation
	// agree exactly, and the "reserved" check in CheckNameValidity stays a plain
	// equality predicate.
	RequestedName      *string
	RequestedNameLower *string

	DestinationWorldId *world.Id
	SourceWorldId      world.Id `gorm:"not null"`

	// AssetId is null on the cash-shop purchase path, which carries an
	// entitlement reference correlated by TransactionId instead of an asset.
	AssetId *uint32

	Reason        string    `gorm:"not null;default:''"`
	TransactionId uuid.UUID `gorm:"not null"`

	CreatedAt  time.Time `gorm:"not null"`
	ExpiresAt  time.Time `gorm:"not null"`
	ResolvedAt *time.Time
	NotifiedAt *time.Time
}

func (e entity) TableName() string {
	return "character_pending_changes"
}
```

- [ ] **Step 4: Register the migration**

In `main.go`, add the import `"atlas-character/pending_change"` and extend line 72:

```go
db := database.Connect(l, database.SetMigrations(character.Migration, history.Migration, saved_location.Migration, teleport_rock.Migration, pending_change.Migration, outboxlib.Migration))
```

- [ ] **Step 5: Run the tests and confirm they pass**

Run: `cd services/atlas-character/atlas.com/character && go test -race ./pending_change/ -run TestMigration -v`
Expected: PASS (both tests).

- [ ] **Step 6: Commit**

```bash
git add services/atlas-character/atlas.com/character/pending_change/ \
        services/atlas-character/atlas.com/character/main.go
git commit -m "feat(task-227): character_pending_changes entity with partial unique indexes"
```

### Task 3: Model, Builder, administrator, provider

**Files:**
- Create: `pending_change/model.go`, `pending_change/builder.go`, `pending_change/administrator.go`, `pending_change/provider.go`
- Create: `pending_change/administrator_test.go`

**Interfaces:**
- Produces:
  - `Model` with getters `Id() uuid.UUID`, `CharacterId() uint32`, `Type() string`, `Status() string`, `RequestedName() string`, `DestinationWorldId() world.Id`, `SourceWorldId() world.Id`, `AssetId() uint32`, `HasAsset() bool`, `Reason() string`, `TransactionId() uuid.UUID`, `CreatedAt() time.Time`, `ExpiresAt() time.Time`, `ResolvedAt() *time.Time`, `NotifiedAt() *time.Time`
  - `NewBuilder() *modelBuilder` with `SetId/SetCharacterId/SetType/SetStatus/SetRequestedName/SetDestinationWorldId/SetSourceWorldId/SetAssetId/SetReason/SetTransactionId/SetCreatedAt/SetExpiresAt/SetResolvedAt/SetNotifiedAt` and `Build() Model`
  - `create(db *gorm.DB, tenantId uuid.UUID, m Model) (Model, error)` — returns `ErrAlreadyPending` / `ErrNameReserved` on unique violation
  - `transition(db *gorm.DB, id uuid.UUID, status string, reason string, at time.Time) (Model, bool, error)` — the bool reports whether a row actually moved out of `PENDING`
  - `markNotified(db *gorm.DB, id uuid.UUID, at time.Time) error`
  - `getById(db *gorm.DB, id uuid.UUID) (Model, error)`
  - `getByCharacterId(db *gorm.DB, characterId uint32) ([]Model, error)`
  - `getPendingByNameLower(db *gorm.DB, nameLower string) (Model, error)`
  - `getExpired(db *gorm.DB, now time.Time) ([]Model, error)`
  - `getResolvedUnnotified(db *gorm.DB, characterId uint32) ([]Model, error)`
  - `getPendingByCharacterId(db *gorm.DB, characterId uint32) ([]Model, error)`
  - Sentinels `ErrAlreadyPending`, `ErrNameReserved`, `ErrNotFound`, `ErrAlreadyTerminal`

- [ ] **Step 1: Write the failing administrator test**

`transition` is the idempotency mechanism for the entire refund path (design §3.10): the refund is emitted only by the transition that actually moves `status` away from `PENDING`, so a redelivered command must observe `moved == false`.

```go
package pending_change

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

func TestTransitionIsOneWayAndReportsWhetherItMoved(t *testing.T) {
	db := newTestDB(t)
	if err := Migration(db); err != nil {
		t.Fatalf("Migration: %v", err)
	}
	tid := uuid.New()

	m, err := create(db, tid, NewBuilder().
		SetId(uuid.New()).
		SetCharacterId(11).
		SetType(TypeNameChange).
		SetStatus(StatusPending).
		SetRequestedName("Bravo").
		SetSourceWorldId(world.Id(0)).
		SetTransactionId(uuid.New()).
		SetCreatedAt(time.Now()).
		SetExpiresAt(time.Now().Add(time.Hour)).
		Build())
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	now := time.Now()
	got, moved, err := transition(db, m.Id(), StatusCancelled, "operator_cancelled", now)
	if err != nil {
		t.Fatalf("transition: %v", err)
	}
	if !moved {
		t.Fatal("expected the first transition to move the row")
	}
	if got.Status() != StatusCancelled || got.Reason() != "operator_cancelled" {
		t.Fatalf("unexpected post-state: %s / %s", got.Status(), got.Reason())
	}
	if got.ResolvedAt() == nil {
		t.Fatal("expected resolved_at to be stamped")
	}

	// A redelivered cancel finds a terminal row: nothing moves, nothing is
	// re-stamped, and the caller must not emit a refund.
	_, moved, err = transition(db, m.Id(), StatusCancelled, "operator_cancelled", now.Add(time.Minute))
	if err != nil {
		t.Fatalf("second transition returned an error: %v", err)
	}
	if moved {
		t.Fatal("expected the second transition to be a no-op")
	}
}

func TestCreateMapsUniqueViolationsToSentinels(t *testing.T) {
	db := newTestDB(t)
	if err := Migration(db); err != nil {
		t.Fatalf("Migration: %v", err)
	}
	tid := uuid.New()
	base := func(charId uint32, name string) Model {
		return NewBuilder().
			SetId(uuid.New()).SetCharacterId(charId).SetType(TypeNameChange).
			SetStatus(StatusPending).SetRequestedName(name).
			SetSourceWorldId(world.Id(0)).SetTransactionId(uuid.New()).
			SetCreatedAt(time.Now()).SetExpiresAt(time.Now().Add(time.Hour)).Build()
	}
	if _, err := create(db, tid, base(21, "Charlie")); err != nil {
		t.Fatalf("create: %v", err)
	}
	// Same character, same type.
	if _, err := create(db, tid, base(21, "Delta")); !errors.Is(err, ErrAlreadyPending) {
		t.Fatalf("expected ErrAlreadyPending, got %v", err)
	}
	// Different character, same name — case-insensitively.
	if _, err := create(db, tid, base(22, "cHaRlIe")); !errors.Is(err, ErrNameReserved) {
		t.Fatalf("expected ErrNameReserved, got %v", err)
	}
}
```

- [ ] **Step 2: Run the tests and confirm they fail**

Run: `go test ./pending_change/ -run 'TestTransition|TestCreateMaps' -v`
Expected: FAIL — `create`, `transition`, `NewBuilder` undefined.

- [ ] **Step 3: Write model.go and builder.go**

Mirror `saved_location/model.go` and `saved_location/builder.go` exactly in shape: private fields, value-receiver getters, a `modelBuilder` with chained setters and `Build()`. `RequestedName()` returns `""` when the pointer is nil; `DestinationWorldId()` returns `world.Id(0)`; `AssetId()` returns `0` and `HasAsset()` reports the pointer's presence — nil-vs-zero matters for the refund decision, so keep both accessors.

Add `modelFromEntity(e entity) (Model, error)` in `provider.go`, as `saved_location/provider.go` does.

- [ ] **Step 4: Write administrator.go**

```go
package pending_change

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	// ErrAlreadyPending maps idx_pc_one_pending_per_type (FR-2.3).
	ErrAlreadyPending = errors.New("request already pending")
	// ErrNameReserved maps idx_pc_name_reservation (FR-3.3).
	ErrNameReserved = errors.New("name reserved")
	// ErrNotFound is returned for an unknown pending-change id.
	ErrNotFound = errors.New("pending change not found")
	// ErrAlreadyTerminal is returned when the caller asked for a transition on
	// a record that has already left PENDING. The REST layer maps it to 409.
	ErrAlreadyTerminal = errors.New("pending change already terminal")
)

func create(db *gorm.DB, tenantId uuid.UUID, m Model) (Model, error) {
	e := &entity{
		Id:            m.Id(),
		TenantId:      tenantId,
		CharacterId:   m.CharacterId(),
		Type:          m.Type(),
		Status:        m.Status(),
		SourceWorldId: m.SourceWorldId(),
		Reason:        m.Reason(),
		TransactionId: m.TransactionId(),
		CreatedAt:     m.CreatedAt(),
		ExpiresAt:     m.ExpiresAt(),
	}
	if n := m.RequestedName(); n != "" {
		lower := strings.ToLower(n)
		e.RequestedName = &n
		e.RequestedNameLower = &lower
	}
	if m.Type() == TypeWorldTransfer {
		d := m.DestinationWorldId()
		e.DestinationWorldId = &d
	}
	if m.HasAsset() {
		a := m.AssetId()
		e.AssetId = &a
	}

	if err := db.Create(e).Error; err != nil {
		return Model{}, mapUniqueViolation(err)
	}
	return modelFromEntity(*e)
}

// mapUniqueViolation turns a partial-unique-index violation into the reason the
// caller reports. Discriminated on the index name, because both indexes are
// 23505 and only the name says which invariant the insert hit.
func mapUniqueViolation(err error) error {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "idx_pc_one_pending_per_type"):
		return ErrAlreadyPending
	case strings.Contains(msg, "idx_pc_name_reservation"):
		return ErrNameReserved
	default:
		return err
	}
}

// transition moves a record out of PENDING exactly once. The returned bool is
// the idempotency signal for the whole refund path (design §3.10): callers emit
// the refund and the resolved notification ONLY when it is true, so a
// redelivered Kafka command mints nothing.
func transition(db *gorm.DB, id uuid.UUID, status string, reason string, at time.Time) (Model, bool, error) {
	res := db.Model(&entity{}).
		Where("id = ? AND status = ?", id, StatusPending).
		Updates(map[string]interface{}{"status": status, "reason": reason, "resolved_at": at})
	if res.Error != nil {
		return Model{}, false, res.Error
	}
	m, err := getById(db, id)
	if err != nil {
		return Model{}, false, err
	}
	return m, res.RowsAffected == 1, nil
}

func markNotified(db *gorm.DB, id uuid.UUID, at time.Time) error {
	return db.Model(&entity{}).
		Where("id = ? AND notified_at IS NULL", id).
		Update("notified_at", at).Error
}
```

- [ ] **Step 5: Write provider.go**

Plain `db.Where(...)` reads returning `Model` / `[]Model` through `modelFromEntity`, following `saved_location/administrator.go`'s query style. `getById` maps `gorm.ErrRecordNotFound` to `ErrNotFound`. `getPendingByNameLower` filters `status = 'PENDING' AND type = 'NAME_CHANGE' AND requested_name_lower = ?`. `getExpired` filters `status = 'PENDING' AND expires_at < ?`. `getResolvedUnnotified` filters `character_id = ? AND resolved_at IS NOT NULL AND notified_at IS NULL`. Tenant scoping comes from the request-scoped `db` the processor hands in, exactly as the sibling packages do.

- [ ] **Step 6: Run the tests and confirm they pass**

Run: `go test -race ./pending_change/ -v`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-character/atlas.com/character/pending_change/
git commit -m "feat(task-227): pending-change model, builder, administrator, provider"
```

### Task 4: Kafka contract and producers for the pending-change lifecycle

**Files:**
- Create: `kafka/message/pending_change/kafka.go`
- Create: `pending_change/producer.go`
- Modify: `kafka/message/character/kafka.go` (add `StatusEventTypeWorldChanged` + its body)
- Modify: `character/producer.go` (add `worldChangedEventProvider`)
- Create: `pending_change/producer_test.go`

**Interfaces:**
- Consumes: `pending_change.Model` (Task 3).
- Produces:
  - `pending_change.EnvEventTopic = "EVENT_TOPIC_CHARACTER_PENDING_CHANGE"`, `EventTypeCreated = "PENDING_CHANGE_CREATED"`, `EventTypeResolved = "PENDING_CHANGE_RESOLVED"`
  - `StatusEvent[E any]{TransactionId uuid.UUID; CharacterId uint32; WorldId world.Id; Type string; Body E}`
  - `CreatedEventBody{PendingChangeId uuid.UUID; ChangeType string; RequestedName string; DestinationWorldId world.Id; ExpiresAt time.Time}`
  - `ResolvedEventBody{PendingChangeId uuid.UUID; ChangeType string; Status string; Reason string; RequestedName string; DestinationWorldId world.Id}`
  - `character2.StatusEventTypeWorldChanged = "WORLD_CHANGED"` with `StatusEventWorldChangedBody{OldWorldId world.Id; NewWorldId world.Id}`
  - `createdEventProvider(m Model) model.Provider[[]kafka.Message]`, `resolvedEventProvider(m Model) model.Provider[[]kafka.Message]`

- [ ] **Step 1: Write the failing producer test**

```go
package pending_change

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	pendingchange2 "atlas-character/kafka/message/pending_change"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

func TestResolvedEventCarriesStatusAndReason(t *testing.T) {
	m := NewBuilder().
		SetId(uuid.New()).
		SetCharacterId(42).
		SetType(TypeNameChange).
		SetStatus(StatusRejected).
		SetReason("name_taken").
		SetRequestedName("Echo").
		SetSourceWorldId(world.Id(3)).
		SetTransactionId(uuid.New()).
		SetCreatedAt(time.Now()).
		SetExpiresAt(time.Now().Add(time.Hour)).
		Build()

	msgs, err := resolvedEventProvider(m)()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}

	var ev pendingchange2.StatusEvent[pendingchange2.ResolvedEventBody]
	if err := json.Unmarshal(msgs[0].Value, &ev); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if ev.Type != pendingchange2.EventTypeResolved {
		t.Fatalf("type = %s", ev.Type)
	}
	if ev.CharacterId != 42 || ev.WorldId != world.Id(3) {
		t.Fatalf("routing fields = %d / %d", ev.CharacterId, ev.WorldId)
	}
	if ev.Body.Status != StatusRejected || ev.Body.Reason != "name_taken" {
		t.Fatalf("body = %s / %s", ev.Body.Status, ev.Body.Reason)
	}
	if ev.Body.RequestedName != "Echo" {
		t.Fatalf("requestedName = %s", ev.Body.RequestedName)
	}
}
```

- [ ] **Step 2: Run the test and confirm it fails**

Run: `go test ./pending_change/ -run TestResolvedEvent -v`
Expected: FAIL — `resolvedEventProvider` undefined.

- [ ] **Step 3: Write the message contract**

`kafka/message/pending_change/kafka.go` follows the shape of `kafka/message/character/kafka.go`: an `EnvEventTopic` const, event-type consts, a generic `StatusEvent[E any]` with json tags `transactionId`, `characterId`, `worldId`, `type`, `body`, and the two body structs. Keys use `producer.CreateKey(int(characterId))` so a character's events stay ordered on one partition.

- [ ] **Step 4: Write `pending_change/producer.go`**

Two providers, mirroring `character/producer.go`'s `nameChangedEventProvider` shape (`producer.SingleMessageProvider(key, value)` over a `&StatusEvent[...]{}`). `createdEventProvider` populates `ExpiresAt`; `resolvedEventProvider` populates `Status` and `Reason`. Both carry `RequestedName` and `DestinationWorldId` so atlas-channel can name the requested value in the player-facing pink text without a REST round trip.

- [ ] **Step 5: Add `WORLD_CHANGED` to the character contract**

In `kafka/message/character/kafka.go`, add `StatusEventTypeWorldChanged = "WORLD_CHANGED"` alongside `StatusEventTypeNameChanged` (line 235) and a `StatusEventWorldChangedBody{OldWorldId world.Id \`json:"oldWorldId"\`; NewWorldId world.Id \`json:"newWorldId"\`}`. In `character/producer.go`, add `worldChangedEventProvider(transactionId uuid.UUID, characterId uint32, oldWorldId world.Id, newWorldId world.Id)` copying `nameChangedEventProvider` (line 261) verbatim in shape. Set the event's `WorldId` to the **new** world — consumers route on it and the character now lives there.

- [ ] **Step 6: Run the test and confirm it passes**

Run: `go test -race ./pending_change/ ./kafka/... ./character/ -run 'TestResolvedEvent|TestWorld' -v`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-character/atlas.com/character/kafka/message/ \
        services/atlas-character/atlas.com/character/pending_change/ \
        services/atlas-character/atlas.com/character/character/producer.go
git commit -m "feat(task-227): pending-change event contract and WORLD_CHANGED status event"
```

### Task 5: `NameScope` and the reservation check in `CheckNameValidity`

FR-3.2 makes a name-change name unique **tenant-wide**, deliberately stricter than character creation, so a later world transfer can never collide. FR-3.3's reservation must be visible to character creation too — that is the one behaviour change creation sees, and it is required.

**Files:**
- Modify: `character/processor.go:268-289` (`CheckNameValidity`)
- Modify: `character/name_validity_resource.go`
- Modify: `character/name_validity_resource_test.go`
- Modify: `services/atlas-character-factory/atlas.com/character-factory/character/name_validity_requests.go`

**Interfaces:**
- Consumes: `pending_change.getPendingByNameLower` — exposed to the `character` package as a narrow injected function to avoid an import cycle (`pending_change` will import `character` for the apply path). Define in `pending_change/processor.go` (Task 6) and inject from `main.go`:
  ```go
  // NameReservedFunc reports whether name is held by a live pending name-change
  // reservation. Injected rather than imported: pending_change already depends on
  // character for the apply path, so a direct import here would cycle.
  type NameReservedFunc func(l logrus.FieldLogger, ctx context.Context, name string) (bool, error)
  ```
- Produces: `type NameScope string`, `NameScopeWorld`, `NameScopeTenant`; `CheckNameValidity(name string, worldId world.Id, scope NameScope) (NameValidityResult, error)`; `NameValidityResult.Reason` gains `"reserved"`.

- [ ] **Step 1: Write the failing tests**

Append to `character/name_validity_resource_test.go`:

```go
func TestNameValidityTenantScopeRejectsOtherWorldDuplicate(t *testing.T) {
	db := newTestDB(t)
	// A character named "Foxtrot" in world 1.
	seedCharacter(t, db, 1, "Foxtrot")

	p := NewProcessor(testLogger(t), testContext(t), db)

	// World scope: creation in world 0 may reuse the name (today's behaviour).
	res, err := p.CheckNameValidity("Foxtrot", world.Id(0), NameScopeWorld)
	if err != nil {
		t.Fatalf("world scope: %v", err)
	}
	if !res.Valid {
		t.Fatalf("expected world-scoped check to allow a cross-world duplicate, got %s", res.Reason)
	}

	// Tenant scope: a name change may not.
	res, err = p.CheckNameValidity("Foxtrot", world.Id(0), NameScopeTenant)
	if err != nil {
		t.Fatalf("tenant scope: %v", err)
	}
	if res.Valid || res.Reason != "duplicate" {
		t.Fatalf("expected tenant-scoped duplicate, got valid=%v reason=%s", res.Valid, res.Reason)
	}
}

func TestNameValidityRejectsAReservedNameInBothScopes(t *testing.T) {
	db := newTestDB(t)
	p := NewProcessor(testLogger(t), testContext(t), db).
		WithNameReserved(func(_ logrus.FieldLogger, _ context.Context, name string) (bool, error) {
			return strings.EqualFold(name, "Golf"), nil
		})

	for _, scope := range []NameScope{NameScopeWorld, NameScopeTenant} {
		res, err := p.CheckNameValidity("golf", world.Id(0), scope)
		if err != nil {
			t.Fatalf("scope %s: %v", scope, err)
		}
		if res.Valid || res.Reason != "reserved" {
			t.Fatalf("scope %s: expected reserved, got valid=%v reason=%s", scope, res.Valid, res.Reason)
		}
	}
}
```

- [ ] **Step 2: Run the tests and confirm they fail**

Run: `go test ./character/ -run TestNameValidity -v`
Expected: FAIL — `NameScope` undefined, `CheckNameValidity` takes two arguments.

- [ ] **Step 3: Implement the scope and reservation check**

```go
// NameScope selects the uniqueness scope of a name check. Character creation
// uses NameScopeWorld (a name may repeat across worlds); a name change uses
// NameScopeTenant (FR-3.2), which is deliberately stricter so that a later
// world transfer can never produce a collision.
type NameScope string

const (
	NameScopeWorld  NameScope = "WORLD"
	NameScopeTenant NameScope = "TENANT"
)

func (p *ProcessorImpl) CheckNameValidity(name string, worldId world.Id, scope NameScope) (NameValidityResult, error) {
	if len(name) < 3 || len(name) > 12 {
		return NameValidityResult{Valid: false, Reason: "length", Detail: "Name must be 3-12 characters."}, nil
	}
	m, err := regexp.MatchString("^[A-Za-z0-9぀-ゟ゠-ヿ一-龯]{3,12}$", name)
	if err != nil {
		return NameValidityResult{}, err
	}
	if !m {
		return NameValidityResult{Valid: false, Reason: "regex", Detail: "Name contains invalid characters."}, nil
	}

	// getForName is already tenant-wide and already LOWER(name) = LOWER(?)
	// (provider.go:30-39); NameScopeWorld re-applies the world filter the
	// processor has always applied, byte for byte.
	cs, err := p.GetForName()(name)
	if err != nil {
		return NameValidityResult{}, err
	}
	for _, c := range cs {
		if scope == NameScopeTenant || c.WorldId() == worldId {
			return NameValidityResult{Valid: false, Reason: "duplicate", Detail: "Name already taken."}, nil
		}
	}

	// A live reservation blocks BOTH scopes (FR-3.3): a rename in flight must
	// block a creation of the same name, or the rename loses its own race at
	// apply time.
	if p.nameReserved != nil {
		reserved, err := p.nameReserved(p.l, p.ctx, name)
		if err != nil {
			return NameValidityResult{}, err
		}
		if reserved {
			return NameValidityResult{Valid: false, Reason: "reserved", Detail: "Name is reserved by a pending name change."}, nil
		}
	}

	return NameValidityResult{Valid: true}, nil
}
```

Add the `nameReserved NameReservedFunc` field to `ProcessorImpl` and a `WithNameReserved(f NameReservedFunc) *ProcessorImpl` setter that returns a shallow copy, following the module's existing `WithTransaction` convention. A nil `nameReserved` skips the check, so every existing `NewProcessor` call site keeps compiling and keeps its current behaviour.

- [ ] **Step 4: Update every `CheckNameValidity` call site**

Grep it: `grep -rn "CheckNameValidity" services/`. `name_validity_resource.go` reads an optional `scope` query parameter, defaulting to `WORLD` so the existing atlas-character-factory client is unaffected, and rejects any other value with 400. Add `Reserved bool` to `NameValidityResponse` so the PRD §5 `available|taken|reserved|invalid` contract is expressible without a second endpoint; set it from `res.Reason == "reserved"`.

- [ ] **Step 5: Pass the scope explicitly from atlas-character-factory**

In `services/atlas-character-factory/atlas.com/character-factory/character/name_validity_requests.go`, append `&scope=WORLD` to the request URL. Explicit beats implicit: creation's scope is now a stated choice rather than a default it inherits.

- [ ] **Step 6: Run the tests and confirm they pass**

Run: `go test -race ./character/ -run TestNameValidity -v` in atlas-character, then `go build ./...` in atlas-character-factory.
Expected: PASS, clean build.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-character/atlas.com/character/character/ \
        services/atlas-character-factory/atlas.com/character-factory/character/name_validity_requests.go
git commit -m "feat(task-227): tenant-scoped name checks and pending-name reservation"
```

### Task 6: The processor — create, resolve, and the refund/notify emission

This is where the highest-risk requirement lands: **refund idempotency** (FR-2.8, design §3.10). The refund is emitted only by the transition that actually moves `status` away from `PENDING`, inside the same transaction, through the outbox. A redelivered cancel finds `moved == false`, transitions nothing, and emits nothing.

**Files:**
- Create: `pending_change/processor.go`
- Create: `pending_change/processor_test.go`
- Create: `pending_change/refund_idempotency_test.go`

**Interfaces:**
- Consumes: `create`, `transition`, `getById`, `getPendingByNameLower`, `getPendingByCharacterId`, `getExpired`, `getResolvedUnnotified`, `markNotified` (Task 3); `createdEventProvider`, `resolvedEventProvider` (Task 4); `character.NewProcessor(...).CheckNameValidity(name, worldId, character.NameScopeTenant)` (Task 5).
- Produces:
  ```go
  type Processor interface {
      Create(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, changeType string, requestedName string, destinationWorldId world.Id, assetId *uint32) (Model, error)
      CreateAndEmit(transactionId uuid.UUID, characterId uint32, changeType string, requestedName string, destinationWorldId world.Id, assetId *uint32) (Model, error)
      Resolve(mb *message.Buffer) func(id uuid.UUID, status string, reason string) (Model, bool, error)
      ResolveAndEmit(id uuid.UUID, status string, reason string) (Model, bool, error)
      ApplyForCharacter(characterId uint32) error
      RenotifyForCharacter(characterId uint32) error
      Sweep(now time.Time) error
      GetByCharacterId(characterId uint32) ([]Model, error)
      GetById(id uuid.UUID) (Model, error)
      NameReserved(name string) (bool, error)
      WithTransaction(tx *gorm.DB) Processor
  }
  func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor
  // NameReservedFunc adapter for character.ProcessorImpl.WithNameReserved:
  func NameReservedFor(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, name string) (bool, error)
  ```
  Typed rejections re-exported for the REST layer: `ErrAlreadyPending`, `ErrNameReserved`, `ErrNotFound`, `ErrAlreadyTerminal`, plus `ErrIneligible` carrying a reason (`type IneligibleError struct{ Reason string }`).

- [ ] **Step 1: Write the failing refund-idempotency test**

This is the single most important test in the task. The known failure mode in this codebase is Kafka at-least-once redelivery through a non-idempotent handler duplicating items.

```go
package pending_change

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// A redelivered cancel must not mint a second coupon. The guard is the
// transition's RowsAffected, not a handler-level dedupe: only the transition
// that actually moves status out of PENDING emits the refund.
func TestRedeliveredCancelRefundsExactlyOnce(t *testing.T) {
	db := newTestDB(t)
	if err := Migration(db); err != nil {
		t.Fatalf("Migration: %v", err)
	}
	l, ctx := testLogger(t), testContext(t)

	assetId := uint32(9001)
	p := NewProcessor(l, ctx, db)
	m, err := p.CreateAndEmit(uuid.New(), 55, TypeNameChange, "Hotel", world.Id(0), &assetId)
	if err != nil {
		t.Fatalf("CreateAndEmit: %v", err)
	}

	awardsBefore := countOutboxMessagesMatching(t, db, "award_asset")

	for i := 0; i < 3; i++ {
		if _, moved, err := p.ResolveAndEmit(m.Id(), StatusCancelled, "operator_cancelled"); err != nil {
			t.Fatalf("delivery %d: %v", i, err)
		} else if moved != (i == 0) {
			t.Fatalf("delivery %d: moved = %v, want %v", i, moved, i == 0)
		}
	}

	if got := countOutboxMessagesMatching(t, db, "award_asset") - awardsBefore; got != 1 {
		t.Fatalf("expected exactly 1 refund emission across 3 deliveries, got %d", got)
	}
	if got := countOutboxMessagesMatching(t, db, "PENDING_CHANGE_RESOLVED"); got != 1 {
		t.Fatalf("expected exactly 1 resolved notification, got %d", got)
	}
}

// A purchase-path record carries no asset id; resolution must still notify, and
// must not emit a refund with a zero asset.
func TestPurchasePathResolutionEmitsNoAssetRefund(t *testing.T) {
	db := newTestDB(t)
	if err := Migration(db); err != nil {
		t.Fatalf("Migration: %v", err)
	}
	p := NewProcessor(testLogger(t), testContext(t), db)
	m, err := p.CreateAndEmit(uuid.New(), 56, TypeWorldTransfer, "", world.Id(2), nil)
	if err != nil {
		t.Fatalf("CreateAndEmit: %v", err)
	}
	if _, moved, err := p.ResolveAndEmit(m.Id(), StatusExpired, "expired"); err != nil || !moved {
		t.Fatalf("ResolveAndEmit: moved=%v err=%v", moved, err)
	}
	if got := countOutboxMessagesMatching(t, db, "award_asset"); got != 0 {
		t.Fatalf("expected no asset refund on the purchase path, got %d", got)
	}
	if got := countOutboxMessagesMatching(t, db, "PENDING_CHANGE_RESOLVED"); got != 1 {
		t.Fatalf("expected 1 resolved notification, got %d", got)
	}
}
```

Write `countOutboxMessagesMatching(t, db, substr)` in the same file as a plain query over the outbox table populated by `outbox.EmitProvider` — a test-local query helper, not a test-only constructor, so the "no `*_testhelpers.go`" rule is respected.

- [ ] **Step 2: Run the test and confirm it fails**

Run: `go test ./pending_change/ -run 'TestRedeliveredCancel|TestPurchasePath' -v`
Expected: FAIL — `NewProcessor` undefined.

- [ ] **Step 3: Write `Create` / `CreateAndEmit`**

```go
func (p *ProcessorImpl) Create(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, changeType string, requestedName string, destinationWorldId world.Id, assetId *uint32) (Model, error) {
	return func(transactionId uuid.UUID, characterId uint32, changeType string, requestedName string, destinationWorldId world.Id, assetId *uint32) (Model, error) {
		t := tenant.MustFromContext(p.ctx)

		c, err := character.NewProcessor(p.l, p.ctx, p.db).GetById()(characterId)
		if err != nil {
			return Model{}, err
		}

		switch changeType {
		case TypeNameChange:
			res, err := character.NewProcessor(p.l, p.ctx, p.db).
				WithNameReserved(NameReservedFor(p.db)).
				CheckNameValidity(requestedName, c.WorldId(), character.NameScopeTenant)
			if err != nil {
				return Model{}, err
			}
			if !res.Valid {
				return Model{}, IneligibleError{Reason: reasonForNameValidity(res.Reason)}
			}
		case TypeWorldTransfer:
			if reason, ok := p.evaluateTransferEligibility(c, destinationWorldId); !ok {
				return Model{}, IneligibleError{Reason: reason}
			}
		default:
			return Model{}, IneligibleError{Reason: "unknown_change_type"}
		}

		now := time.Now()
		b := NewBuilder().
			SetId(uuid.New()).
			SetCharacterId(characterId).
			SetType(changeType).
			SetStatus(StatusPending).
			SetRequestedName(requestedName).
			SetDestinationWorldId(destinationWorldId).
			SetSourceWorldId(c.WorldId()).
			SetTransactionId(transactionId).
			SetCreatedAt(now).
			SetExpiresAt(now.Add(p.expiry))
		if assetId != nil {
			b = b.SetAssetId(*assetId)
		}

		m, err := create(p.db, t.Id(), b.Build())
		if err != nil {
			return Model{}, err
		}

		if err := mb.Put(pendingchange2.EnvEventTopic, createdEventProvider(m)); err != nil {
			return Model{}, err
		}
		// Consumption is at request acceptance (FR-2.8). The purchase path has
		// no asset to destroy — atlas-cashshop consumes the entitlement off the
		// PENDING_CHANGE_CREATED event instead.
		if assetId != nil {
			if err := mb.Put(sagamsg.EnvCommandTopic, destroyAssetCommandProvider(m)); err != nil {
				return Model{}, err
			}
		}
		p.l.Infof("Created pending change [%s] type [%s] for character [%d] in world [%d].", m.Id(), changeType, characterId, c.WorldId())
		return m, nil
	}
}
```

`CreateAndEmit` wraps it in `database.ExecuteTransaction` + `message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))`, exactly as `character/processor.go:1888-1893` does. `reasonForNameValidity` maps `"length"`→`name_invalid_length`, `"regex"`→`name_invalid_charset`, `"duplicate"`→`name_taken`, `"reserved"`→`name_reserved`.

- [ ] **Step 4: Write `Resolve` / `ResolveAndEmit`**

```go
// Resolve moves a record to a terminal status and, ONLY when the row actually
// moved, emits the refund and the resolved notification. This is the whole of
// the idempotency contract: a redelivered command sees moved == false, emits
// nothing, and mints nothing (design §3.10).
func (p *ProcessorImpl) Resolve(mb *message.Buffer) func(id uuid.UUID, status string, reason string) (Model, bool, error) {
	return func(id uuid.UUID, status string, reason string) (Model, bool, error) {
		m, moved, err := transition(p.db, id, status, reason, time.Now())
		if err != nil {
			return Model{}, false, err
		}
		if !moved {
			p.l.Debugf("Pending change [%s] already terminal (%s); skipping refund and notification.", id, m.Status())
			return m, false, nil
		}

		if status != StatusApplied && m.HasAsset() {
			if err := mb.Put(sagamsg.EnvCommandTopic, awardAssetCommandProvider(m)); err != nil {
				return Model{}, false, err
			}
		}
		if err := mb.Put(pendingchange2.EnvEventTopic, resolvedEventProvider(m)); err != nil {
			return Model{}, false, err
		}
		p.l.Infof("Pending change [%s] for character [%d] transitioned PENDING -> %s, reason [%s].", m.Id(), m.CharacterId(), status, reason)
		return m, true, nil
	}
}
```

Note the two emissions are inside the transaction that performed the `UPDATE`, so the outbox row and the status change commit together or not at all.

- [ ] **Step 5: Write `ApplyForCharacter`, `RenotifyForCharacter`, `Sweep`, and the getters**

`ApplyForCharacter` loads `getPendingByCharacterId` and, per record:
- `TypeNameChange` — re-validate with `NameScopeTenant`; on failure `Resolve(..., StatusRejected, "name_taken")`; on success PATCH the name through `character.NewProcessor(...).Update(mb)(transactionId, characterId, RestModel{Name: requestedName})` — which already emits `NAME_CHANGED` (`character/processor.go:1926`) — then `Resolve(..., StatusApplied, "")`, releasing the reservation.
- `TypeWorldTransfer` — start the `WorldTransfer` saga (Task 10) and leave the record `PENDING`; the saga's terminal event drives `Resolve`.

`RenotifyForCharacter` loads `getResolvedUnnotified` and re-emits `resolvedEventProvider` for each (FR-2.9). `Sweep(now)` loads `getExpired` and calls `ResolveAndEmit(..., StatusExpired, "expired")` per row. `NameReserved(name)` is `getPendingByNameLower(strings.ToLower(name))` mapped to a bool. `p.expiry` comes from the configuration registry of Task 8; `NewProcessor` defaults it to `168 * time.Hour` when the registry has no entry.

- [ ] **Step 6: Run the tests and confirm they pass**

Run: `go test -race ./pending_change/ -v`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-character/atlas.com/character/pending_change/
git commit -m "feat(task-227): pending-change processor with transition-gated refund emission"
```

### Task 7: REST resource — GET / POST / DELETE, and main.go wiring

**Files:**
- Create: `pending_change/rest.go`, `pending_change/resource.go`
- Create: `pending_change/resource_test.go`
- Modify: `main.go` (route initializer)

**Interfaces:**
- Consumes: `Processor` (Task 6).
- Produces: `RestModel` with `GetName() "pending-changes"`; `InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer`; routes `GET|POST /characters/{characterId}/pending-changes` and `DELETE /characters/{characterId}/pending-changes/{id}`.

- [ ] **Step 1: Write the failing resource test**

```go
func TestCreatePendingChangeMapsRejectionsToStatusCodes(t *testing.T) {
	cases := []struct {
		name    string
		seed    func(t *testing.T, db *gorm.DB)
		input   CreateInputRestModel
		want    int
		reason  string
	}{
		{name: "unknown character", input: CreateInputRestModel{Type: TypeNameChange, RequestedName: "India"}, want: http.StatusNotFound},
		{name: "invalid name", seed: seedChar, input: CreateInputRestModel{Type: TypeNameChange, RequestedName: "ab"}, want: http.StatusUnprocessableEntity, reason: "name_invalid_length"},
		{name: "already pending", seed: seedCharWithPending, input: CreateInputRestModel{Type: TypeNameChange, RequestedName: "Juliet"}, want: http.StatusConflict, reason: "already_pending"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// ... exercise the handler, assert status and the reason in the error body
		})
	}
}

func TestDeletePendingChangeIsConflictOnceTerminal(t *testing.T) {
	// First DELETE -> 204 and status CANCELLED. Second DELETE -> 409.
}
```

- [ ] **Step 2: Run it and confirm it fails**

Run: `go test ./pending_change/ -run 'TestCreatePendingChange|TestDeletePendingChange' -v`
Expected: FAIL — `CreateInputRestModel` undefined.

- [ ] **Step 3: Write rest.go**

`RestModel` fields: `Id string \`json:"-"\``, `CharacterId uint32`, `Type string`, `Status string`, `RequestedName string`, `DestinationWorldId world.Id`, `SourceWorldId world.Id`, `Reason string`, `CreatedAt time.Time`, `ExpiresAt time.Time`, `ResolvedAt *time.Time`. `CreateInputRestModel` carries `Type`, `RequestedName`, `DestinationWorldId`, `AssetId *uint32`. `Transform(m Model) (RestModel, error)` and `Extract` follow `saved_location/rest.go`.

- [ ] **Step 4: Write resource.go**

Model it on `teleport_rock/resource.go`: `registerGet := rest.RegisterHandler(l)(db)(si)`, a `router.PathPrefix("/characters/{characterId}/pending-changes").Subrouter()`, and `rest.ParseCharacterId` wrapping each handler. Status mapping:

| processor error | HTTP |
|---|---|
| `character` not found | 404 |
| `IneligibleError` | 422, body carries `reason` |
| `ErrAlreadyPending` / `ErrNameReserved` | 409, body carries `already_pending` / `name_reserved` |
| `ErrAlreadyTerminal` | 409 |
| `ErrNotFound` | 404 |

The DELETE handler calls `ResolveAndEmit(id, StatusCancelled, "operator_cancelled")` and returns 409 when `moved == false`. Add the doc comment that makes the security property explicit:

```go
// handleCancelPendingChange is the ONLY cancel path in the system. The game
// client has no SendCancel* of any kind on any version (design §4.2.1), so this
// route is operator-facing and MUST NOT be reachable from a socket handler. The
// cancel-unreachability test in atlas-channel asserts that machine-checkably.
```

- [ ] **Step 5: Wire main.go**

Add `AddRouteInitializer(pending_change.InitResource(GetServer())(db)).` to the `server.New(l)` chain, and inject the reservation check into the character processor used by `name_validity_resource` by having `character.NewProcessor` pick up `pending_change.NameReservedFor(db)` at its call site there.

- [ ] **Step 6: Run the tests and confirm they pass**

Run: `go test -race ./... && go vet ./...` in `services/atlas-character/atlas.com/character`
Expected: PASS, clean.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-character/atlas.com/character/
git commit -m "feat(task-227): pending-changes REST resource with operator-only cancel"
```

### Task 8: Expiry configuration — the `imprint-configs` tenant resource and its client

FR-2.6 requires a configurable expiry. atlas-character consumes no tenant configuration today, so this adds the smallest complete instance of the existing pattern: a resource on atlas-tenants and a registry-backed client, modelled file-for-file on `services/atlas-trades/atlas.com/trades/configuration/`.

**Files:**
- Create: `services/atlas-tenants/atlas.com/tenants/configuration/imprint_handler.go`
- Modify: `services/atlas-tenants/atlas.com/tenants/configuration/{rest.go,provider.go,kafka.go,seed.go,resource.go}`
- Create: `services/atlas-tenants/configurations/imprint-configs/default.json`
- Create: `services/atlas-character/atlas.com/character/configuration/{model.go,rest.go,requests.go,registry.go}`
- Create: `services/atlas-character/atlas.com/character/configuration/model_test.go`
- Modify: `services/atlas-character/atlas.com/character/pending_change/processor.go` (read `p.expiry` from the registry)

**Interfaces:**
- Produces: resource name `imprint-configs`; `ImprintConfigRestModel{Id string; PendingExpiryHours int \`json:"pendingExpiryHours"\`}`; on the character side `configuration.Model` with `PendingExpiry() time.Duration`, `DefaultConfig()` returning 168h, `Extract(r RestModel) Model` zero-folding to the default, and `configuration.GetRegistry().Get(tenantId)`.

- [ ] **Step 1: Write the failing zero-fold test**

```go
package configuration

import (
	"testing"
	"time"
)

func TestExtractFoldsAbsentExpiryToTheDefault(t *testing.T) {
	if got := Extract(RestModel{}).PendingExpiry(); got != 168*time.Hour {
		t.Fatalf("PendingExpiry = %v, want 168h", got)
	}
}

func TestExtractHonoursAnOperatorOverride(t *testing.T) {
	if got := Extract(RestModel{PendingExpiryHours: 24}).PendingExpiry(); got != 24*time.Hour {
		t.Fatalf("PendingExpiry = %v, want 24h", got)
	}
}
```

- [ ] **Step 2: Run it and confirm it fails**

Run: `go test ./configuration/ -v` in `services/atlas-character/atlas.com/character`
Expected: FAIL — package does not exist.

- [ ] **Step 3: Write the character-side client**

Copy the four files from `services/atlas-trades/atlas.com/trades/configuration/` and reduce to the single knob. `requests.go` builds `GET {TENANTS}/tenants/{tenantId}/configurations/imprint-configs`. `registry.go` keeps the `sync.Once` + `sync.RWMutex` singleton shape and falls back to `DefaultConfig()` when the fetch misses, so a tenant with no seeded resource gets 7 days rather than a zero expiry.

- [ ] **Step 4: Add the atlas-tenants side**

Register `imprint-configs` alongside `trade-configs`: a `GetImprintConfigByIdProvider` / `GetAllImprintConfigsProvider` pair in `provider.go` over `GetByTenantIdAndResourceNameProvider(tenantID, "imprint-configs")`, the three `EventTypeImprintConfig{Created,Updated,Deleted}` consts plus `CreateImprintConfigStatusEventProvider` in `kafka.go`, a `defaultImprintConfigsPath = "/configurations/imprint-configs"` + `LoadImprintConfigFiles()` in `seed.go`, the rest model in `rest.go`, and the routes in `resource.go` — each mirroring its `trade-configs` neighbour.

`services/atlas-tenants/configurations/imprint-configs/default.json` holds `{"pendingExpiryHours": 168}`.

- [ ] **Step 5: Bootstrap the registry and consume the expiry**

In atlas-character `main.go`, initialise the configuration registry next to the other registries. In `pending_change.NewProcessor`, set `expiry` from `configuration.GetRegistry().Get(tenant.MustFromContext(ctx).Id()).PendingExpiry()`.

- [ ] **Step 6: Run the tests and confirm they pass**

Run: `go test -race ./... && go vet ./...` in both atlas-character and atlas-tenants.
Expected: PASS, clean.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-tenants/ services/atlas-character/
git commit -m "feat(task-227): tenant-configurable pending-change expiry via imprint-configs"
```

### Task 9: The applier — LOGOUT apply and LOGIN catch-up

Design §3.3: both entry points require a live channel session, so a pending request is always created while the character is online and the next transition is necessarily their logout. LOGIN is a catch-up so a crashed applier or a Kafka gap self-heals rather than stranding the coupon until expiry.

**Files:**
- Modify: `kafka/consumer/character/consumer.go` (two new handlers on the already-subscribed status topic)
- Create: `kafka/consumer/character/pending_change_applier_test.go`

**Interfaces:**
- Consumes: `pending_change.NewProcessor(...).ApplyForCharacter`, `.RenotifyForCharacter` (Task 6).
- Produces: `handleLogoutApplyPendingChanges(db *gorm.DB) message.Handler[character2.StatusEvent[character2.StatusEventLogoutBody]]` and `handleLoginPendingChangeCatchUp(db *gorm.DB) message.Handler[character2.StatusEvent[character2.StatusEventLoginBody]]`.

- [ ] **Step 1: Write the failing applier test**

```go
// FR-2.4: application must never mutate a character that is live in a channel.
// The LOGOUT event is the proof of absence; the applier is the only writer.
func TestLogoutAppliesAPendingNameChange(t *testing.T) {
	db := newTestDB(t)
	seedCharacter(t, db, 1, "Kilo")
	pcp := pending_change.NewProcessor(l, ctx, db)
	m, err := pcp.CreateAndEmit(uuid.New(), 1, pending_change.TypeNameChange, "Lima", world.Id(0), nil)
	if err != nil {
		t.Fatalf("CreateAndEmit: %v", err)
	}

	handleLogoutApplyPendingChanges(db)(l, ctx, character2.StatusEvent[character2.StatusEventLogoutBody]{
		TransactionId: uuid.New(), CharacterId: 1, WorldId: world.Id(0),
		Type:          character2.StatusEventTypeLogout,
	})

	c, err := character.NewProcessor(l, ctx, db).GetById()(1)
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if c.Name() != "Lima" {
		t.Fatalf("name = %s, want Lima", c.Name())
	}
	got, err := pcp.GetById(m.Id())
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if got.Status() != pending_change.StatusApplied {
		t.Fatalf("status = %s, want APPLIED", got.Status())
	}
	if countOutboxMessagesMatching(t, db, "NAME_CHANGED") != 1 {
		t.Fatal("expected exactly one NAME_CHANGED emission")
	}
}

// FR-2.7 / §5.2: the name was taken between reservation and apply.
func TestLogoutRejectsAndRefundsWhenTheNameWasTaken(t *testing.T) {
	db := newTestDB(t)
	seedCharacter(t, db, 1, "Mike")
	assetId := uint32(9002)
	pcp := pending_change.NewProcessor(l, ctx, db)
	m, _ := pcp.CreateAndEmit(uuid.New(), 1, pending_change.TypeNameChange, "November", world.Id(0), &assetId)

	// Another character takes the name in the interim.
	seedCharacter(t, db, 2, "November")

	handleLogoutApplyPendingChanges(db)(l, ctx, character2.StatusEvent[character2.StatusEventLogoutBody]{
		TransactionId: uuid.New(), CharacterId: 1, WorldId: world.Id(0),
		Type:          character2.StatusEventTypeLogout,
	})

	got, _ := pcp.GetById(m.Id())
	if got.Status() != pending_change.StatusRejected || got.Reason() != "name_taken" {
		t.Fatalf("got %s / %s, want REJECTED / name_taken", got.Status(), got.Reason())
	}
	if countOutboxMessagesMatching(t, db, "award_asset") != 1 {
		t.Fatal("expected the coupon to be refunded exactly once")
	}
}

// FR-2.9: a resolution that happened while the player was offline is delivered
// on their next login, not discarded.
func TestLoginReemitsAnUnnotifiedResolution(t *testing.T) {
	db := newTestDB(t)
	seedCharacter(t, db, 1, "Oscar")
	pcp := pending_change.NewProcessor(l, ctx, db)
	m, _ := pcp.CreateAndEmit(uuid.New(), 1, pending_change.TypeNameChange, "Papa", world.Id(0), nil)
	_, _, _ = pcp.ResolveAndEmit(m.Id(), pending_change.StatusCancelled, "operator_cancelled")

	before := countOutboxMessagesMatching(t, db, "PENDING_CHANGE_RESOLVED")
	handleLoginPendingChangeCatchUp(db)(l, ctx, character2.StatusEvent[character2.StatusEventLoginBody]{
		TransactionId: uuid.New(), CharacterId: 1, WorldId: world.Id(0),
		Type:          character2.StatusEventTypeLogin,
	})
	if got := countOutboxMessagesMatching(t, db, "PENDING_CHANGE_RESOLVED") - before; got != 1 {
		t.Fatalf("expected 1 re-emission, got %d", got)
	}
}
```

- [ ] **Step 2: Run the tests and confirm they fail**

Run: `go test ./kafka/consumer/character/ -run 'TestLogoutApplies|TestLogoutRejects|TestLoginReemits' -v`
Expected: FAIL — handlers undefined.

- [ ] **Step 3: Write the two handlers**

```go
// handleLogoutApplyPendingChanges is the safe point of FR-2.4: the LOGOUT status
// event is atlas-character's own statement that the character left every
// channel, so it is the earliest moment a rename or a world transfer may touch
// the record. The applier tolerates redelivery — the status transition is the
// idempotency key (pending_change.transition).
func handleLogoutApplyPendingChanges(db *gorm.DB) message.Handler[character2.StatusEvent[character2.StatusEventLogoutBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.StatusEventLogoutBody]) {
		if e.Type != character2.StatusEventTypeLogout {
			return
		}
		if err := pending_change.NewProcessor(l, ctx, db).ApplyForCharacter(e.CharacterId); err != nil {
			l.WithError(err).Errorf("Unable to apply pending changes for character [%d] at logout.", e.CharacterId)
		}
	}
}

// handleLoginPendingChangeCatchUp does two things, both catch-ups rather than
// primary paths: it re-emits a resolution the player was offline for (FR-2.9),
// and it re-attempts any PENDING row whose apply previously failed, so a crashed
// applier or a Kafka gap self-heals instead of stranding the coupon until expiry.
func handleLoginPendingChangeCatchUp(db *gorm.DB) message.Handler[character2.StatusEvent[character2.StatusEventLoginBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.StatusEventLoginBody]) {
		if e.Type != character2.StatusEventTypeLogin {
			return
		}
		p := pending_change.NewProcessor(l, ctx, db)
		if err := p.RenotifyForCharacter(e.CharacterId); err != nil {
			l.WithError(err).Errorf("Unable to re-emit unnotified pending-change resolutions for character [%d].", e.CharacterId)
		}
	}
}
```

Register both in `InitHandlers` on the status topic, next to `handleLevelChangedStatusEvent` (`consumer.go:301`). Note that the LOGIN handler deliberately does **not** call `ApplyForCharacter` — the character is live at that moment, and FR-2.4 forbids applying to a live character; the stranded-PENDING catch-up happens on their next logout, which the LOGOUT handler already covers.

- [ ] **Step 4: Run the tests and confirm they pass**

Run: `go test -race ./kafka/consumer/character/ -v`
Expected: PASS.

- [ ] **Step 5: Verify the consumer group does not collide**

`consumerGroupId` is `consumergroup.Resolve("Character Service")` (`main.go:39`) and is already used for this topic by the two existing status handlers, so no new group is introduced and design risk R4's collision concern does not apply. Confirm by reading `main.go:96-113` and note it in the commit body.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-character/atlas.com/character/kafka/consumer/character/
git commit -m "feat(task-227): apply pending changes at logout, re-notify at login"
```

### Task 10: The expiry sweep ticker

**Files:**
- Create: `pending_change/task.go`
- Create: `pending_change/task_test.go`
- Modify: `main.go` (register the ticker)

**Interfaces:**
- Consumes: `Processor.Sweep(now time.Time) error` (Task 6).
- Produces: `const ExpiryTask = "pending-change-expiry"`; `NewExpiry(l logrus.FieldLogger, db *gorm.DB, interval time.Duration) *Expiry` with `Run()` and `SleepTime() time.Duration`, matching the `session.Timeout` shape at `session/task.go`.

- [ ] **Step 1: Write the failing sweep test**

```go
func TestSweepExpiresAndRefundsAPastDueRequest(t *testing.T) {
	db := newTestDB(t)
	if err := Migration(db); err != nil {
		t.Fatalf("Migration: %v", err)
	}
	assetId := uint32(9003)
	p := NewProcessor(testLogger(t), testContext(t), db)
	m, err := p.CreateAndEmit(uuid.New(), 77, TypeNameChange, "Quebec", world.Id(0), &assetId)
	if err != nil {
		t.Fatalf("CreateAndEmit: %v", err)
	}
	// Backdate expires_at rather than sleeping.
	if err := db.Model(&entity{}).Where("id = ?", m.Id()).
		Update("expires_at", time.Now().Add(-time.Minute)).Error; err != nil {
		t.Fatalf("backdate: %v", err)
	}

	if err := p.Sweep(time.Now()); err != nil {
		t.Fatalf("Sweep: %v", err)
	}

	got, _ := p.GetById(m.Id())
	if got.Status() != StatusExpired || got.Reason() != "expired" {
		t.Fatalf("got %s / %s, want EXPIRED / expired", got.Status(), got.Reason())
	}
	if countOutboxMessagesMatching(t, db, "award_asset") != 1 {
		t.Fatal("expected exactly one refund")
	}

	// Idempotent: a second sweep must not refund again.
	if err := p.Sweep(time.Now()); err != nil {
		t.Fatalf("second Sweep: %v", err)
	}
	if countOutboxMessagesMatching(t, db, "award_asset") != 1 {
		t.Fatal("second sweep refunded again")
	}
}
```

- [ ] **Step 2: Run it and confirm it fails**

Run: `go test ./pending_change/ -run TestSweepExpires -v`
Expected: FAIL — `Sweep` behaviour not yet exercised end to end / `NewExpiry` undefined.

- [ ] **Step 3: Write task.go**

Copy the structure of `session/task.go` verbatim: an otel span named by the task const, then iterate. Unlike `session.Timeout`, the sweep has no in-memory registry to walk, so it must resolve a tenant context per row. Query the distinct `tenant_id` values with a pending expired row, build `tenant.WithContext(sctx, m)` per tenant using the tenant model fetched from the tenant registry, and call `Sweep` per tenant. Never run an untenanted query — the processor asserts `tenant.MustFromContext`.

- [ ] **Step 4: Register the ticker in main.go**

Next to the existing `session.NewTimeout` registration:

```go
routine.Go(l, rt.Context(), func(_ context.Context) {
	tasks.Register(l, rt.Context())(pending_change.NewExpiry(l, db, time.Minute*15))
})
```

15 minutes is chosen against a 7-day default expiry — the sweep's latency budget is hours, and a tighter interval buys nothing but load.

- [ ] **Step 5: Run the tests and confirm they pass**

Run: `go test -race ./... && go vet ./...` in atlas-character, then `tools/goroutine-guard.sh` from the repo root.
Expected: PASS, clean, guard exit 0.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-character/atlas.com/character/
git commit -m "feat(task-227): expiry sweep ticker for pending changes"
```

---

## Phase C — World transfer: eligibility gates and the compensating saga

### Task 11: World-transfer eligibility

Design §3.6 makes the escrow systems **blocking** rather than auto-settled, a deliberate departure from PRD FR-4.4: each is a player-visible, player-fixable state with its own close flow, auto-settling an auction someone already bid on is not reversible by compensation, and it removes the three highest-risk compensating steps from the saga. Design §1.6 adds the GM and **family** gates, which the PRD does not carry — the v83 client's own `CCashShop::CheckTransferWorldPossible` (`@0x4734e5`) refuses to even send the request in those states, so a server that permits it produces a state the client considers impossible.

**Files:**
- Create: `pending_change/eligibility.go`
- Create: `pending_change/eligibility_test.go`
- Create: `pending_change/requests.go` (REST clients for the gate lookups)

**Interfaces:**
- Consumes: `character.Model` (for `WorldId()`, `Gm()`, `AccountId()`).
- Produces: `evaluateTransferEligibility(c character.Model, destinationWorldId world.Id) (reason string, ok bool)` on `ProcessorImpl`, plus one narrow REST client per gate.

Gate table — evaluated in this order (cheapest and most local first, so an obviously-invalid request never fans out):

| # | condition | source | reason |
|---|---|---|---|
| 1 | destination == source | local | `world_same` |
| 2 | character is a GM (`characters.gm != 0`) | local | `is_gm` |
| 3 | destination world unknown or full | atlas-world `GET /worlds/{worldId}` | `world_unknown` / `world_full` |
| 4 | no free character slot in destination | atlas-account `GET /accounts/{accountId}` | `no_character_slot` |
| 5 | name taken in destination world | local `GetForName` + world filter | `name_taken` |
| 6 | character banned | atlas-ban `GET /bans?characterId=` | `banned` |
| 7 | guild **master** | atlas-guilds `GET /guilds` member lookup, `members.title == 1` | `is_guild_master` |
| 8 | in a family | atlas-families `GET /families/tree/{characterId}` | `in_family` |
| 9 | open trade | atlas-trades `GET /trades/rooms` | `trade_open` |
| 10 | open hired merchant | atlas-merchant `GET /characters/{characterId}/...` | `merchant_open` |
| 11 | live MTS listings or bids | atlas-mts `GET /characters/{characterId}/mts/holding` | `mts_listings_open` |

The exact sub-paths for gates 3, 4, 6, 7, 9, 10, 11 must be read from each service's `resource.go` (route prefixes confirmed present: `/worlds`, `/accounts`, `/bans`, `/guilds`, `/trades/rooms`, `/characters/{characterId}` in atlas-merchant, `/characters/{characterId}/mts/holding`). Do not guess a route — read the registration.

- [ ] **Step 1: Write the failing eligibility test**

Table-driven, one row per gate, with each REST client stubbed via the injected-function pattern already used by `teleport_rock.WorldIdOf`:

```go
func TestEligibilityGates(t *testing.T) {
	cases := []struct {
		name string
		deps gateDeps // the injected REST lookups, stubbed per case
		dest world.Id
		want string   // "" means eligible
	}{
		{name: "same world", dest: world.Id(0), want: "world_same"},
		{name: "gm blocked", deps: gateDeps{gm: 1}, dest: world.Id(1), want: "is_gm"},
		{name: "destination full", deps: gateDeps{worldFull: true}, dest: world.Id(1), want: "world_full"},
		{name: "no slot", deps: gateDeps{slotsFree: 0}, dest: world.Id(1), want: "no_character_slot"},
		{name: "name taken in destination", deps: gateDeps{nameTakenInDest: true}, dest: world.Id(1), want: "name_taken"},
		{name: "banned", deps: gateDeps{banned: true}, dest: world.Id(1), want: "banned"},
		{name: "guild master", deps: gateDeps{guildTitle: 1}, dest: world.Id(1), want: "is_guild_master"},
		{name: "in family", deps: gateDeps{inFamily: true}, dest: world.Id(1), want: "in_family"},
		{name: "open trade", deps: gateDeps{tradeOpen: true}, dest: world.Id(1), want: "trade_open"},
		{name: "merchant open", deps: gateDeps{merchantOpen: true}, dest: world.Id(1), want: "merchant_open"},
		{name: "mts listings", deps: gateDeps{mtsHolding: true}, dest: world.Id(1), want: "mts_listings_open"},
		{name: "eligible", deps: gateDeps{slotsFree: 3}, dest: world.Id(1), want: ""},
	}
	// Assert the returned reason equals want and ok == (want == "").
}

// A non-master guild member is severed by the saga, not blocked (design §3.6).
func TestGuildMemberIsNotBlocked(t *testing.T) {
	reason, ok := eval(gateDeps{guildTitle: 3, slotsFree: 2}, world.Id(1))
	if !ok || reason != "" {
		t.Fatalf("expected a rank-3 guild member to be eligible, got %s", reason)
	}
}
```

- [ ] **Step 2: Run it and confirm it fails**

Run: `go test ./pending_change/ -run 'TestEligibilityGates|TestGuildMember' -v`
Expected: FAIL — `gateDeps` undefined.

- [ ] **Step 3: Implement the gates**

`gateDeps` is a struct of function fields, one per remote lookup, so the test stubs them without an HTTP round trip and production wires the real REST clients. `evaluateTransferEligibility` short-circuits on the first failing gate and returns its reason from the closed taxonomy. Log each rejection at info with tenant, character, destination and reason (design §8).

- [ ] **Step 4: Add the eligibility REST endpoint**

`GET /characters/{characterId}/transfer-eligibility?destinationWorldId={id}` on the pending-changes resource, returning `{eligible bool, reason string}`. This backs the synchronous `WORLD_TRANSFER` availability check of design §3.5 — atlas-channel calls it and writes the result packet in the same handler invocation, with no Kafka round trip, because nothing is mutated.

- [ ] **Step 5: Re-check at apply time**

`ApplyForCharacter` (Task 6, Step 5) must call `evaluateTransferEligibility` again before starting the saga. FR-4.2 requires the destination-name check specifically as an apply-time safety net; a stale gate that passed at request time and fails now resolves the record to `REJECTED` with that gate's reason.

- [ ] **Step 6: Run the tests and confirm they pass**

Run: `go test -race ./pending_change/ -v && go vet ./...`
Expected: PASS, clean.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-character/atlas.com/character/pending_change/
git commit -m "feat(task-227): world-transfer eligibility gates incl. GM and family"
```

### Task 12: `libs/atlas-saga` — the `WorldTransfer` saga type, actions, and payloads

**Files:**
- Modify: `libs/atlas-saga/model.go`
- Create: `libs/atlas-saga/world_transfer_test.go`

**Interfaces:**
- Produces:
  ```go
  WorldTransfer Type = "world_transfer"

  ValidateWorldTransfer  Action = "validate_world_transfer"
  LeaveGuildForTransfer  Action = "leave_guild_for_transfer"
  LeavePartyForTransfer  Action = "leave_party_for_transfer"
  SeverBuddiesForTransfer Action = "sever_buddies_for_transfer"
  ChangeCharacterWorld   Action = "change_character_world"

  type ValidateWorldTransferPayload struct {
      CharacterId        uint32   `json:"characterId"`
      SourceWorldId      world.Id `json:"sourceWorldId"`
      DestinationWorldId world.Id `json:"destinationWorldId"`
      PendingChangeId    uuid.UUID `json:"pendingChangeId"`
  }
  type LeaveGuildForTransferPayload struct {
      CharacterId uint32   `json:"characterId"`
      WorldId     world.Id `json:"worldId"`
      GuildId     uint32   `json:"guildId"`
      // Title is recorded so the compensation can re-add the member at the rank
      // they held. A guild re-join is not a client-driveable recovery, so this
      // is the one severance whose compensation must be exact.
      Title byte `json:"title"`
  }
  type LeavePartyForTransferPayload struct {
      CharacterId uint32   `json:"characterId"`
      WorldId     world.Id `json:"worldId"`
      PartyId     uint32   `json:"partyId"`
  }
  type SeverBuddiesForTransferPayload struct {
      CharacterId uint32   `json:"characterId"`
      WorldId     world.Id `json:"worldId"`
      // BuddyIds is captured before severance so the compensation can restore
      // entries in both directions.
      BuddyIds []uint32 `json:"buddyIds"`
  }
  type ChangeCharacterWorldPayload struct {
      CharacterId        uint32    `json:"characterId"`
      SourceWorldId      world.Id  `json:"sourceWorldId"`
      DestinationWorldId world.Id  `json:"destinationWorldId"`
      PendingChangeId    uuid.UUID `json:"pendingChangeId"`
  }
  ```

Step order is fixed and load-bearing: `validate` → `leave_guild` → `leave_party` → `sever_buddies` → `change_character_world`. The world update is **last** and is a single-row update, so a failure anywhere leaves the character wholly in the source world with only recoverable severances applied — the character is never in two worlds and never in none.

- [ ] **Step 1: Write the failing round-trip test**

```go
func TestWorldTransferPayloadsRoundTrip(t *testing.T) {
	for _, p := range []any{
		ValidateWorldTransferPayload{CharacterId: 1, SourceWorldId: 0, DestinationWorldId: 1, PendingChangeId: uuid.New()},
		LeaveGuildForTransferPayload{CharacterId: 1, WorldId: 0, GuildId: 5, Title: 3},
		LeavePartyForTransferPayload{CharacterId: 1, WorldId: 0, PartyId: 9},
		SeverBuddiesForTransferPayload{CharacterId: 1, WorldId: 0, BuddyIds: []uint32{2, 3}},
		ChangeCharacterWorldPayload{CharacterId: 1, SourceWorldId: 0, DestinationWorldId: 1, PendingChangeId: uuid.New()},
	} {
		b, err := json.Marshal(p)
		if err != nil {
			t.Fatalf("marshal %T: %v", p, err)
		}
		out := reflect.New(reflect.TypeOf(p)).Interface()
		if err := json.Unmarshal(b, out); err != nil {
			t.Fatalf("unmarshal %T: %v", p, err)
		}
		if !reflect.DeepEqual(p, reflect.ValueOf(out).Elem().Interface()) {
			t.Fatalf("%T did not round-trip", p)
		}
	}
}
```

- [ ] **Step 2: Run it and confirm it fails**

Run: `cd libs/atlas-saga && go test ./ -run TestWorldTransferPayloads -v`
Expected: FAIL — types undefined.

- [ ] **Step 3: Add the type, actions and payloads**

Append the `Type` const to the block at `libs/atlas-saga/model.go:14-49` and the five `Action` consts to their own commented group, following the file's existing grouping convention.

- [ ] **Step 4: Run the test and confirm it passes**

Run: `go test -race ./ && go vet ./` in `libs/atlas-saga`
Expected: PASS, clean.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-saga/
git commit -m "feat(task-227): world-transfer saga type, actions and payloads"
```

### Task 13: atlas-saga-orchestrator — the five step handlers

**Files:**
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/model.go` (aliases + `Step.UnmarshalJSON` payload arms)
- Modify: `.../saga/handler.go` (interface entries, dispatch arms, five handlers)
- Modify: `.../saga/event_acceptance.go` (action → expected event kinds)
- Create: `.../pending_change/` (REST client: world update + resolve)
- Create: `.../saga/world_transfer_test.go`

**Interfaces:**
- Consumes: the five actions and payloads from Task 12.
- Produces on `HandlerImpl`: `handleValidateWorldTransfer`, `handleLeaveGuildForTransfer`, `handleLeavePartyForTransfer`, `handleSeverBuddiesForTransfer`, `handleChangeCharacterWorld` — each `func(s Saga, st Step[any]) error`, matching `handleIncreaseBuddyCapacity` (`handler.go:1371`) in shape.

- [ ] **Step 1: Write the failing handler-dispatch test**

```go
// Every new action must resolve to a handler; an unmapped action silently stalls
// the saga at that step, which is indistinguishable from a slow downstream.
func TestWorldTransferActionsAllResolveToHandlers(t *testing.T) {
	h := &HandlerImpl{}
	for _, a := range []Action{
		ValidateWorldTransfer, LeaveGuildForTransfer, LeavePartyForTransfer,
		SeverBuddiesForTransfer, ChangeCharacterWorld,
	} {
		if _, ok := h.GetHandler(a); !ok {
			t.Fatalf("action %s has no handler", a)
		}
	}
}

// Payload unmarshal must produce the concrete type, not map[string]interface{} —
// every handler type-asserts and returns "invalid payload" otherwise.
func TestWorldTransferStepsUnmarshalToConcretePayloads(t *testing.T) {
	raw := `{"action":"change_character_world","payload":{"characterId":1,"sourceWorldId":0,"destinationWorldId":1,"pendingChangeId":"` + uuid.New().String() + `"}}`
	var st Step[any]
	if err := json.Unmarshal([]byte(raw), &st); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := st.Payload().(ChangeCharacterWorldPayload); !ok {
		t.Fatalf("payload type = %T, want ChangeCharacterWorldPayload", st.Payload())
	}
}
```

- [ ] **Step 2: Run the tests and confirm they fail**

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test ./saga/ -run TestWorldTransfer -v`
Expected: FAIL — actions undefined.

- [ ] **Step 3: Add the aliases and the unmarshal arms**

In `saga/model.go`, add the five `= sharedsaga.X` action aliases next to `IncreaseBuddyCapacity` (line 95), the five payload aliases next to `IncreaseBuddyCapacityPayload` (line 263), and one `case` per action in the payload switch at line 1140. The existing `unmarshal_completeness_test.go` will catch any action left out of that switch — run it.

The dispatch method is `func (h *HandlerImpl) GetHandler(action Action) (ActionHandler, bool)` — add one `case` per action to its switch, alongside `case IncreaseBuddyCapacity: return h.handleIncreaseBuddyCapacity, true` (`handler.go:824`). Also add the five method entries to the `Handler` interface (`handler.go:115`).

- [ ] **Step 4: Write the five handlers**

Each mirrors `handleIncreaseBuddyCapacity`: type-assert the payload, call a processor, `h.logActionError(s, st, err, "...")` on failure, return the error.

- `handleValidateWorldTransfer` — calls atlas-character's `GET /characters/{id}/transfer-eligibility` (Task 11 Step 4). Read-only, so no compensation is needed. An `eligible: false` response is an error carrying the gate's reason, which becomes the saga's failure reason and hence the record's `REJECTED` reason.
- `handleLeaveGuildForTransfer` — emits `COMMAND_TOPIC_GUILD` `LEAVE` (`services/atlas-guilds/.../kafka/message/guild/kafka.go:22`). Skip cleanly when `GuildId == 0`.
- `handleLeavePartyForTransfer` — emits `COMMAND_TOPIC_PARTY` `LEAVE` (`services/atlas-parties/atlas.com/parties/party/kafka.go:9`). Skip when `PartyId == 0`.
- `handleSeverBuddiesForTransfer` — emits `COMMAND_TOPIC_BUDDY_LIST` `REQUEST_DELETE` (`services/atlas-buddies/.../kafka/message/list/kafka.go:18`) once per id in `BuddyIds`, **and** once in the reverse direction for each, since FR-4.3 requires severance in both directions.
- `handleChangeCharacterWorld` — PATCHes the character's world through the new orchestrator-side `pending_change` REST client, then resolves the pending-change record to `APPLIED`. `WORLD_CHANGED` is emitted by atlas-character on that update (Task 4).

- [ ] **Step 5: Add the event-acceptance entries**

In `event_acceptance.go`, add a row per action to the map at line ~140:

```go
sharedsaga.ValidateWorldTransfer:   {},                             // read-only, no event
sharedsaga.LeaveGuildForTransfer:   {EventKindGuildMemberLeft},
sharedsaga.LeavePartyForTransfer:   {EventKindPartyMemberLeft},
sharedsaga.SeverBuddiesForTransfer: {EventKindBuddyDeleted},
sharedsaga.ChangeCharacterWorld:    {EventKindCharacterWorldChanged},
```

Add any missing `EventKind*` const and its matching event-kind derivation, following the existing `EventKindBuddyCapacityChanged` wiring. `EventKindCharacterWorldChanged` derives from the new `WORLD_CHANGED` status event of Task 4.

- [ ] **Step 6: Run the tests and confirm they pass**

Run: `go test -race ./saga/ -v && go vet ./...`
Expected: PASS including `unmarshal_completeness_test.go`.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-saga-orchestrator/
git commit -m "feat(task-227): world-transfer saga step handlers"
```

### Task 14: atlas-saga-orchestrator — compensations and the mid-saga failure proof

FR-4.8 and the PRD's acceptance criterion both require this to be proven by a test, not by inspection.

**Files:**
- Modify: `.../saga/compensator.go`
- Create: `.../saga/world_transfer_compensation_test.go`

**Interfaces:**
- Consumes: the five handlers of Task 13.
- Produces: `compensateWorldTransfer(s Saga, failedStep Step[any]) error` on `CompensatorImpl`, dispatched from the saga-type switch alongside `compensateMesoSackUse` (`compensator.go:1631`).

Compensation table (design §3.11):

| step | compensation |
|---|---|
| `validate_world_transfer` | none — read-only |
| `leave_guild_for_transfer` | re-add the member at `Title` from the payload |
| `leave_party_for_transfer` | **none** — party membership is not durable state and is not restorable; recorded as a deliberate gap, not an omission |
| `sever_buddies_for_transfer` | restore every id in `BuddyIds`, both directions |
| `change_character_world` | set the world back to `SourceWorldId` |

Compensation runs in reverse step order, and every compensating action logs what it undid (design §8).

- [ ] **Step 1: Write the failing mid-saga failure test**

```go
// FR-4.8 / PRD acceptance: an injected failure at the last step must leave the
// character WHOLLY in the source world, with every recoverable severance undone
// and the coupon refunded. Never half-moved.
func TestChangeCharacterWorldFailureCompensatesEverySeverance(t *testing.T) {
	env := newWorldTransferTestEnv(t)
	env.failAt(ChangeCharacterWorld)

	s := env.startWorldTransfer(worldTransferFixture{
		CharacterId:        1,
		SourceWorldId:      world.Id(0),
		DestinationWorldId: world.Id(1),
		GuildId:            5,
		GuildTitle:         3,
		PartyId:            9,
		BuddyIds:           []uint32{2, 3},
	})

	env.awaitTerminal(s)

	if got := env.characterWorld(1); got != world.Id(0) {
		t.Fatalf("character world = %d, want 0 (source)", got)
	}
	if !env.guildMemberRestored(5, 1, 3) {
		t.Fatal("guild membership was not restored at the prior title")
	}
	if !env.buddiesRestored(1, []uint32{2, 3}) {
		t.Fatal("buddy entries were not restored in both directions")
	}
	if env.pendingChangeStatus() != "REJECTED" || env.pendingChangeReason() != "saga_failed" {
		t.Fatalf("record = %s / %s, want REJECTED / saga_failed",
			env.pendingChangeStatus(), env.pendingChangeReason())
	}
	if env.refundCount() != 1 {
		t.Fatalf("refund count = %d, want exactly 1", env.refundCount())
	}
}

// A failure at the FIRST severance must not attempt to compensate steps that
// never ran — an over-eager compensator re-adds a guild membership the character
// never left, or restores a buddy that was never severed.
func TestEarlyFailureCompensatesOnlyCompletedSteps(t *testing.T) {
	env := newWorldTransferTestEnv(t)
	env.failAt(LeaveGuildForTransfer)

	s := env.startWorldTransfer(worldTransferFixture{
		CharacterId: 1, SourceWorldId: world.Id(0), DestinationWorldId: world.Id(1),
		GuildId: 5, GuildTitle: 3, PartyId: 9, BuddyIds: []uint32{2, 3},
	})
	env.awaitTerminal(s)

	if env.buddyRestoreAttempts() != 0 {
		t.Fatal("compensator restored buddies that were never severed")
	}
	if env.characterWorld(1) != world.Id(0) {
		t.Fatalf("character world changed despite an early failure")
	}
}
```

Build `newWorldTransferTestEnv` on the existing harness in `saga/trade_compensation_test.go` / `meso_sack_compensation_test.go` — those already stub the downstream processors and drive a saga to a terminal state. Do not introduce a new harness.

- [ ] **Step 2: Run the tests and confirm they fail**

Run: `go test ./saga/ -run 'TestChangeCharacterWorldFailure|TestEarlyFailure' -v`
Expected: FAIL — `compensateWorldTransfer` undefined.

- [ ] **Step 3: Write the compensator**

Follow `compensateMesoSackUse` (`compensator.go:1631`) in shape: walk the saga's completed steps in reverse, switch on each step's action, and skip any step that did not complete. Add the `case WorldTransfer:` arm to the saga-type dispatch. The `leave_party_for_transfer` arm is an explicit no-op **with a comment saying why** — a silent absence reads as a missing case.

- [ ] **Step 4: Resolve the record on saga failure**

The saga's terminal-failure path calls atlas-character's `DELETE`-equivalent resolve with `REJECTED` / `saga_failed`, which triggers the refund exactly once through the transition gate of Task 6.

- [ ] **Step 5: Run the tests and confirm they pass**

Run: `go test -race ./saga/ -v && go vet ./...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-saga-orchestrator/
git commit -m "feat(task-227): world-transfer compensations with mid-saga failure proof"
```

---

## Phase D — Packets: eight rows, 59 cells

Every task in this phase follows the project's standing playbooks verbatim. Do **not** restate or improvise a procedure:

- New codec per op: [`docs/packets/IMPLEMENTING_A_PACKET.md`](../../packets/IMPLEMENTING_A_PACKET.md), driven by the `packet-implementer` agent (`/implement-packet`).
- Promoting one op × version cell to `✅`: [`docs/packets/audits/VERIFYING_A_PACKET.md`](../../packets/audits/VERIFYING_A_PACKET.md), driven by the `packet-verifier` agent (`/verify-packet`), fanned out one agent per cell and batched per IDB.
- If Task 1 §5 found `CASHSHOP_CHECK_NAME_CHANGE` to be a mode-prefix dispatcher family, that row instead follows [`docs/packets/DISPATCHER_FAMILY.md`](../../packets/DISPATCHER_FAMILY.md) via the `dispatcher-family-implementer` agent — and dispatcher-family agents must be serialized, never run two in parallel (shared `run.go` / `families.yaml` / global IDA instance).

Constraints that hold for every Phase D task:

- Field order comes from Task 1's `derivation.md` §2, never from a symbol name.
- Every struct is immutable and carries **both** `Encode` and `Decode`.
- Version-divergent fields use the `MajorAtLeast` idiom.
- **No wire change to an already-verified version.** All 59 cells are currently `❌` or `⬜ n-a`, so nothing in this phase may alter an existing verified codec — if a change appears necessary, stop and report it.
- Each op's handlers/writers are registered in every applicable template under `services/atlas-configurations/seed-data/templates/` at their **sorted** `opCode` position (never appended next to a semantically-related entry), with a non-empty `fname`. Then `tools/template-opcode-order-guard.sh` and `tools/template-duplicate-binding-guard.sh` must both exit 0.
- A cell that does not promote in the matrix is a failure report, not a prose claim. `⬜ n-a` cells stay `n-a` unless evidence proves the client sends/receives the op — and then the `n-a` consistency gate applies.

### Task 15: `NAME_TRANSFER` serverbound codec — 6 cells (v83 v84 v87 v92 v95 jms_v185)

**Files:**
- Create: `libs/atlas-packet/cash/serverbound/check_name_change_possible.go` + `_test.go`
- Modify: the six applicable templates
- Modify: `docs/packets/audits/status.json`, `docs/packets/audits/STATUS.md` (regenerated, not hand-edited)

**Interfaces:**
- Produces: an immutable struct with `Decode` and `Encode`, accessors for the candidate name and the SPW credential, and a `String()` that **redacts the SPW**.

- [ ] **Step 1: Confirm the opcode assignment from Task 1**

Read `derivation.md` §1. If the derivation amended the matrix, work from the amended values. Do not proceed on the pre-derivation assumption.

- [ ] **Step 2: Run the Step-0 already-implemented check**

`IMPLEMENTING_A_PACKET.md` §0: a serverbound `❌` is usually an unverified shared decoder, not a missing one. Grep `libs/atlas-packet/cash/serverbound/` for an existing decoder over the same read order before writing a new file.

- [ ] **Step 3: Write the codec with SPW redaction**

Design §1.7: v83 `CCashShop::OnBuyNameChange` (`@0x470480`) calls `ask_SPW()` and passes the result into the send, so this body carries the account's second password. Every handler in atlas-channel logs `p.String()` at debug, so redaction belongs in the struct, not at the call site:

```go
// String redacts the SPW. This body carries the account's second password
// (design §1.7, v83 CCashShop::OnBuyNameChange @0x470480 -> ask_SPW), and every
// serverbound handler in atlas-channel logs p.String() at debug level. The
// credential must never reach a log line.
func (m CheckNameChangePossible) String() string {
	return fmt.Sprintf("Name [%s], SPW [REDACTED]", m.name)
}
```

- [ ] **Step 4: Write the byte-fixture test per version**

One fixture per cell, each with a `packet-audit:verify` marker, per the playbook. Assert `Decode` produces the derived field values and that `String()` contains neither the SPW value nor any substring of it.

- [ ] **Step 5: Register the handler in all six templates**

At the sorted `opCode` position, with a `fname` and the `services` array. Then run both template guards and show exit 0.

- [ ] **Step 6: Fan out `packet-verifier`, one agent per cell**

Six cells. Batch per IDB. Each agent pins its evidence record, regenerates the matrix, and commits its three artifacts together.

- [ ] **Step 7: Confirm all six cells promoted**

Run: `grep '^| NAME_TRANSFER ' docs/packets/audits/STATUS.md`
Expected: six `✅` in the v83/v84/v87/v92/v95/JMS185 columns, no `❌` remaining on that row.

### Task 16: `WORLD_TRANSFER` serverbound codec — 5 cells (v83 v84 v87 v92 v95)

Same seven steps as Task 15, with:
- `libs/atlas-packet/cash/serverbound/check_transfer_world_possible.go`
- The body carries the destination world and (per design §1.7's `ask_SPW` pattern) may carry the SPW — the derivation settles it. If it does, redact it in `String()` identically.
- Verify with: `grep '^| WORLD_TRANSFER ' docs/packets/audits/STATUS.md` → five `✅`.

### Task 17: `CASHSHOP_CHECK_NAME_CHANGE_POSSIBLE_RESULT` clientbound — 6 cells (v79 v83 v84 v87 v92 v95)

Same shape, clientbound. Additionally:
- The reason-code enumeration comes from `derivation.md` §4 and is expressed as a config-resolved table in the templates (DOM-25) — not as Go constants with literal values.
- Map design §6's server-side reasons (`name_taken`, `name_reserved`, `name_invalid_length`, `name_invalid_charset`) onto the derived wire codes in the body builder, and record the mapping in a doc comment.
- Verify: `grep '^| CASHSHOP_CHECK_NAME_CHANGE_POSSIBLE_RESULT ' docs/packets/audits/STATUS.md` → six `✅`.

### Task 18: `CASHSHOP_CHECK_TRANSFER_WORLD_POSSIBLE_RESULT` clientbound — 10 cells (all nine GMS + jms_v185)

Same shape. The reason mapping covers `world_same`, `world_unknown`, `world_full`, `no_character_slot`, `banned`, `is_guild_master`, `is_gm`, `in_family`, `trade_open`, `merchant_open`, `mts_listings_open`, `name_taken`. Design §1.6 recorded the three strings the v83 client itself formats — guild master, GM, and `SP_5017_YOU_HAVE_TO_QUIT_FAMILY__R_NTO_MOVE_TO_ANOTHER_WORLD` — so those three must land on the codes that render them.

Verify: `grep '^| CASHSHOP_CHECK_TRANSFER_WORLD_POSSIBLE_RESULT ' docs/packets/audits/STATUS.md` → ten `✅`.

### Task 19: `CASHSHOP_CHECK_NAME_CHANGE` clientbound — 9 cells (v48 v61 v72 v79 v83 v84 v87 v92 v95)

Row 409 lists three receivers — `CCashShop::OnCheckDuplicatedIDResult`, `CCashShop::OnCheckNameChangePossibleResult`, plus `sub_455A7F` / `sub_463900` / `sub_473519`. Task 1 §5 decided whether this is a single body or a dispatcher family.

- If **dispatcher family**: follow `DISPATCHER_FAMILY.md` via `dispatcher-family-implementer` — a discrete struct per mode, config-resolved mode byte, per-mode body function, per-mode verification. `packet-audit dispatcher-lint` plus the matrix / fname-doc / operations `--check` runs must all show exit 0 before the family is called done. Serialize this agent against any other dispatcher-family agent.
- If **single body**: the ordinary seven-step shape of Task 17.

Name the unnamed IDB functions (`sub_455A7F`, `sub_463900`, `sub_473519`) while reversing rather than leaving them anonymous — an unnamed function is a prerequisite this task can produce itself.

Verify: `grep '^| CASHSHOP_CHECK_NAME_CHANGE ' docs/packets/audits/STATUS.md` → nine `✅`.

### Task 20: `CANCEL_NAME_CHANGE_RESULT` clientbound — 8 cells (v61 v72 v79 v83 v84 v87 v92 v95)

Ordinary seven-step shape. `CWvsContext::OnCancelNameChangeResult` is a **notification** with no serverbound counterpart on any version (design §4.2.1) — the codec is encode-only in practice but still carries `Decode` for the fixture test, per the standing rule.

Design §3.9's belt-and-braces applies at the *handler* level, not here: OQ-9 (whether the client accepts this packet outside the cash-shop UI) is derived during this task and recorded in `derivation.md`. If the client ignores it outside the cash shop, note it — Task 26's consumer sends a pink-text message alongside the packet regardless, so no behaviour changes either way, but the finding must be written down rather than left as an open question.

Verify: `grep '^| CANCEL_NAME_CHANGE_RESULT ' docs/packets/audits/STATUS.md` → eight `✅`.

### Task 21: `CANCEL_TRANSFER_WORLD_RESULT` clientbound — 8 cells (v61 v72 v79 v83 v84 v87 v92 v95)

Same as Task 20 for `CWvsContext::OnCancelTransferWorldResult`.

Verify: `grep '^| CANCEL_TRANSFER_WORLD_RESULT ' docs/packets/audits/STATUS.md` → eight `✅`.

### Task 22: `CANCEL_NAME_CHANGE_BY_OTHER` clientbound — 7 cells (v72 v79 v83 v84 v87 v92 v95)

Same as Task 20 for `CWvsContext::OnCancelNameChangebyOther`. This is the packet FR-2.7 requires specifically when a name change was invalidated because another character took the name.

Verify: `grep '^| CANCEL_NAME_CHANGE_BY_OTHER ' docs/packets/audits/STATUS.md` → seven `✅`.

### Task 23: Packet-phase completeness gate

**Files:**
- Modify: `docs/tasks/task-227-cash-name-change-world-transfer/coverage-manifest.yaml` (only if the derivation changed the declared set)
- Create: `docs/tasks/task-227-cash-name-change-world-transfer/completeness-critic.md` (written by the agent)

- [ ] **Step 1: Confirm every declared cell promoted**

Run:
```bash
grep -E '^\| (NAME_TRANSFER|WORLD_TRANSFER|CASHSHOP_CHECK_NAME_CHANGE|CASHSHOP_CHECK_NAME_CHANGE_POSSIBLE_RESULT|CASHSHOP_CHECK_TRANSFER_WORLD_POSSIBLE_RESULT|CANCEL_NAME_CHANGE_RESULT|CANCEL_TRANSFER_WORLD_RESULT|CANCEL_NAME_CHANGE_BY_OTHER) ' docs/packets/audits/STATUS.md
```
Expected: 59 `✅` across the eight rows and no `❌` on any of them.

- [ ] **Step 2: Run the template guards**

Run:
```bash
tools/template-opcode-order-guard.sh
tools/template-duplicate-binding-guard.sh
tools/template-movement-types-guard.sh
```
Expected: exit 0 each. The movement guard runs because templates changed, even though this task touches no move handler — a template edit that reordered an array can break it.

- [ ] **Step 3: Dispatch `packet-completeness-critic`**

Pin it to Sonnet (project model preference for review workflows). It diffs `coverage-manifest.yaml` against the branch's actual git delta over `libs/atlas-packet` and the matrix delta over `status.json`, and writes `completeness-critic.md`.

- [ ] **Step 4: Resolve every finding**

Expected: no CHANGED-BUT-UNCLAIMED and no CLAIMED-BUT-UNVERIFIED. A CHANGED-BUT-UNCLAIMED finding means the manifest is incomplete — extend it. A CLAIMED-BUT-UNVERIFIED finding means a cell did not actually promote — go back to that op's task. Do not accept either as a documented gap.

- [ ] **Step 5: Commit**

```bash
git add docs/tasks/task-227-cash-name-change-world-transfer/ docs/packets/audits/
git commit -m "docs(task-227): packet coverage manifest reconciled, 59 cells verified"
```

---

## Phase E — atlas-channel: the entry points and the client feedback

### Task 24: The Cash/0540 item-use branch, and the classifier dead-code fix

`GetCashSlotItemType` already handles `item.ClassificationCharacterImprints` and already version-scopes the enum (52/53 pre-v95, 53/54 on GMS ≥ 95), so FR-1.2 is already satisfied by existing code. What is missing is only the dispatch arm.

Two defects at `character_cash_item_use.go:1117-1140`, both in scope:
- Lines 1132-1138 are an **exact duplicate** of the `5401` branch at 1125-1131 — dead code. Task 1 Step 5 re-derived the client's own arm; fix per that finding.
- The version gate is a raw `t.MajorVersion() >= 95`. It is consistent with its ~40 siblings in the same function, so this task does **not** rewrite them (design §10) — but the two new helpers it adds use `MajorAtLeast`.

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go`
- Create: `services/atlas-channel/atlas.com/channel/pendingchange/{requests.go,rest.go,processor.go}`
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_imprint_test.go`

**Interfaces:**
- Consumes: atlas-character `POST /characters/{characterId}/pending-changes` (Task 7).
- Produces:
  ```go
  // nameChangeCashSlotItemType / worldTransferCashSlotItemType return the
  // version-scoped CashSlotItemType for each Cash/0540 sub-flow. Which item-id
  // prefix maps to which flow is settled by task-227 derivation.md §3 — do not
  // reorder these without re-reading it.
  func nameChangeCashSlotItemType(t tenant.Model) CashSlotItemType
  func worldTransferCashSlotItemType(t tenant.Model) CashSlotItemType
  ```
  plus `pendingchange.NewProcessor(l, ctx).RequestNameChange(...)` / `.RequestWorldTransfer(...)`.

- [ ] **Step 1: Write the failing classifier test**

```go
// The classifier's 540 arm had an exact duplicate of the 5401 branch (dead code
// at lines 1132-1138 before task-227). Assert the arm the client actually
// implements, per derivation.md §3.
func TestCharacterImprintClassifierMatchesTheClient(t *testing.T) {
	cases := []struct {
		name   string
		tenant tenant.Model
		itemId item.Id
		want   CashSlotItemType
	}{
		{name: "5400 pre-v95", tenant: testTenant(t, "GMS", 83, 1), itemId: 5400000, want: CashSlotItemType(52)},
		{name: "5400 v95", tenant: testTenant(t, "GMS", 95, 1), itemId: 5400000, want: CashSlotItemType(53)},
		{name: "5401 pre-v95", tenant: testTenant(t, "GMS", 83, 1), itemId: 5401000, want: CashSlotItemType(53)},
		{name: "5401 v95", tenant: testTenant(t, "GMS", 95, 1), itemId: 5401000, want: CashSlotItemType(54)},
		{name: "unknown 540 prefix", tenant: testTenant(t, "GMS", 83, 1), itemId: 5409000, want: CashSlotItemType(0)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := GetCashSlotItemType(tc.tenant)(tc.itemId); got != tc.want {
				t.Fatalf("got %d, want %d", got, tc.want)
			}
		})
	}
}

// The two sub-flow helpers must never collide on one version — a collision
// routes a world transfer into the rename flow.
func TestImprintSubFlowTypesNeverCollide(t *testing.T) {
	for _, tn := range []tenant.Model{
		testTenant(t, "GMS", 83, 1), testTenant(t, "GMS", 95, 1), testTenant(t, "JMS", 185, 1),
	} {
		if nameChangeCashSlotItemType(tn) == worldTransferCashSlotItemType(tn) {
			t.Fatalf("collision on %s v%d", tn.Region(), tn.MajorVersion())
		}
	}
}
```

If `derivation.md` §3 shows the client recognises a third prefix, add its row and drop the duplicate accordingly — the test above is written against the two-prefix arm the current code claims, and the derivation is the authority.

- [ ] **Step 2: Run the tests and confirm they fail**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run 'TestCharacterImprintClassifier|TestImprintSubFlow' -v`
Expected: FAIL — the sub-flow helpers are undefined; the classifier test may already pass, which is fine, it is the regression pin.

- [ ] **Step 3: Remove the dead duplicate and add the two helpers**

Delete lines 1132-1138. Add the two helpers near `viciousHammerCashSlotItemType` (`:867`), using `MajorAtLeast` rather than the raw comparison, and citing `derivation.md` §3 in the doc comment for the prefix→flow assignment.

- [ ] **Step 4: Add the dispatch arm**

Place it with the other cash-slot-type arms (before the classification-first block at `:614`), following the expiration-extender arm (`:334`) in shape:

```go
if it == nameChangeCashSlotItemType(t) {
	sp := cashsb.NewItemUseNameChange(updateTimeFirst)
	sp.Decode(l, ctx)(r, readerOptions)
	handleImprintNameChangeUse(l, ctx, wp)(s, itemId, source, sp.NewName())
	return
}
if it == worldTransferCashSlotItemType(t) {
	sp := cashsb.NewItemUseWorldTransfer(updateTimeFirst)
	sp.Decode(l, ctx)(r, readerOptions)
	handleImprintWorldTransferUse(l, ctx, wp)(s, itemId, source, sp.TargetWorld())
	return
}
```

The sub-body shapes come from the client's `SendConsumeCashItemUseRequest` arm for these two cash-slot types — derive them the same way Task 15 derives its bodies, and add the codec to `libs/atlas-packet/cash/serverbound/` with fixtures if one does not already exist. If the arm sends no sub-body (as the meso-sack arm does, `:595`), decode nothing and note it in a comment.

- [ ] **Step 5: Respect the EnableActions/ExclRequest contract (FR-5.3)**

`CWvsContext::SendConsumeCashItemUseRequest` is the sole caller of `SetExclRequestSent`, so the excl lock is already armed when this arm runs and the client has no timeout. Neither sub-flow warps and neither mutates inventory synchronously (the asset destruction is a saga step driven by atlas-character), so **every** exit from these two handlers — accepted and rejected alike — must call `enableActions`. Do not unlock an outcome that warps; neither of these does.

On rejection, additionally announce the reason via `chatpkt.WorldMessageWriter` with `writer.WorldMessagePopUpBody(...)`, as the expiration-extender arm does at `:363`, so a rejected use is never a dead click.

- [ ] **Step 6: Write the atlas-channel REST client**

`pendingchange/requests.go` + `rest.go` + `processor.go`, following `character/requests.go` in shape. `RequestNameChange` / `RequestWorldTransfer` POST the JSON:API envelope and map the response status to a reason: 409 → `already_pending`, 422 → the `reason` in the error body, 404 → unknown character.

- [ ] **Step 7: Run the tests and confirm they pass**

Run: `go test -race ./... && go vet ./...` in atlas-channel
Expected: PASS, clean.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/
git commit -m "feat(task-227): Cash/0540 item-use dispatch and classifier dead-code fix"
```

### Task 25: `BUY_NAME_CHANGE` (46) and `BUY_WORLD_TRANSFER` (49) stop being log-only

Both serverbound codecs already exist and already decode in full (`ShopOperationBuyNameChange`: `OldName`, `NewName`, `SerialNumber`; `ShopOperationBuyWorldTransfer`: `TargetWorld`, `SerialNumber`), and both mode bytes are already config-resolved through `isCashShopOperation` → `options["operations"]`, so DOM-25 is already honoured on this path. IDA corroborates mode 49: v83 `CCashShop::SendBuyTransferWorldItemPacket` (`@0x473601`) builds `COutPacket(0xE5)` then `Encode1(0x31)` = 49. **No new serverbound codec is needed here** — the work is behavioural only.

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_operation.go:182-194`
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_operation_imprint_test.go`

**Interfaces:**
- Consumes: `pendingchange.NewProcessor(...)` (Task 24).

- [ ] **Step 1: Write the failing handler test**

```go
// FR-1.4: both purchase ops must create the same request the item path creates,
// and must no longer be log-only.
func TestBuyNameChangeCreatesAPendingRequest(t *testing.T) {
	env := newCashShopHandlerEnv(t)
	env.dispatch(cashShopBuyNameChangePacket(t, "Romeo", "Sierra", 12345))

	if got := env.pendingChangeRequests(); len(got) != 1 {
		t.Fatalf("expected 1 request, got %d", len(got))
	}
	req := env.pendingChangeRequests()[0]
	if req.Type != "NAME_CHANGE" || req.RequestedName != "Sierra" {
		t.Fatalf("request = %s / %s", req.Type, req.RequestedName)
	}
}

func TestBuyWorldTransferCreatesAPendingRequest(t *testing.T) {
	env := newCashShopHandlerEnv(t)
	env.dispatch(cashShopBuyWorldTransferPacket(t, 2, 12346))

	req := env.pendingChangeRequests()[0]
	if req.Type != "WORLD_TRANSFER" || req.DestinationWorldId != 2 {
		t.Fatalf("request = %s / %d", req.Type, req.DestinationWorldId)
	}
}

// FR-5.1: no rejection may be a silent drop.
func TestBuyNameChangeRejectionReachesTheClient(t *testing.T) {
	env := newCashShopHandlerEnv(t)
	env.rejectWith(422, "name_taken")
	env.dispatch(cashShopBuyNameChangePacket(t, "Romeo", "Tango", 12347))

	if !env.wroteFailureWithReason("name_taken") {
		t.Fatal("expected a failure arm carrying the reason")
	}
}
```

- [ ] **Step 2: Run the tests and confirm they fail**

Run: `go test ./socket/handler/ -run 'TestBuyNameChange|TestBuyWorldTransfer' -v`
Expected: FAIL — the handlers still only log.

- [ ] **Step 3: Replace the two log-only bodies**

Both arms converge on the same processor calls Task 24's item arm uses (FR-1.4 — one service-side request, two entry points). `OldName` is validated against the session's character name server-side and the request is refused if it disagrees; never trust the client's copy.

- [ ] **Step 4: Write the success and failure arms**

Success: `CashShopNameChangeBuyDoneBody` / `CashShopTransferWorldDoneBody` (`libs/atlas-packet/cash/clientbound/shop_operation_body.go:658,667`), which are item-blob bodies. Failure: `CashShopTransferWorldFailedBody` (`:454`) for the transfer, and for the rename the failure arm the derivation identified. Both are already config-resolved through `atlas_packet.WithResolvedCode("operations", ...)`.

- [ ] **Step 5: Run the tests and confirm they pass**

Run: `go test -race ./... && go vet ./...`
Expected: PASS, clean.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/
git commit -m "feat(task-227): BUY_NAME_CHANGE and BUY_WORLD_TRANSFER create real requests"
```

### Task 26: The availability-check handlers

Design §3.5: these are interactive — the client blocks its dialog on the result — so the handler calls atlas-character over REST and writes the result packet in the same invocation. No Kafka round trip; a saga here would add hundreds of milliseconds to a keystroke-batch interaction for no atomicity benefit, because nothing is mutated.

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_check_name_change.go` + `_test.go`
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_check_world_transfer.go` + `_test.go`
- Modify: the applicable templates (handler registration for the two serverbound opcodes)

**Interfaces:**
- Consumes: the Task 15/16 codecs; atlas-character `GET /characters/name-validity?name=&worldId=&scope=TENANT` (Task 5) and `GET /characters/{id}/transfer-eligibility?destinationWorldId=` (Task 11).
- Produces: `CashShopCheckNameChangeHandleFunc(l, ctx, wp)` and `CashShopCheckWorldTransferHandleFunc(l, ctx, wp)`, both `func(s session.Model, r *request.Reader, readerOptions map[string]interface{})`.

- [ ] **Step 1: Write the failing handler tests**

```go
// FR-3.5: unavailable for an existing name in ANY world of the tenant, for an
// active reservation, and for a name failing validation.
func TestNameChangeCheckReportsEveryUnavailableCause(t *testing.T) {
	cases := []struct {
		name        string
		validity    nameValidityResponse
		wantCode    string
	}{
		{name: "available", validity: nameValidityResponse{Valid: true}, wantCode: "available"},
		{name: "taken in another world", validity: nameValidityResponse{Reason: "duplicate"}, wantCode: "name_taken"},
		{name: "reserved", validity: nameValidityResponse{Reason: "reserved"}, wantCode: "name_reserved"},
		{name: "too short", validity: nameValidityResponse{Reason: "length"}, wantCode: "name_invalid_length"},
		{name: "bad charset", validity: nameValidityResponse{Reason: "regex"}, wantCode: "name_invalid_charset"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := newCheckHandlerEnv(t)
			env.nameValidityReturns(tc.validity)
			env.dispatchNameCheck("Uniform", "1234")
			if got := env.lastResultReason(); got != tc.wantCode {
				t.Fatalf("reason = %s, want %s", got, tc.wantCode)
			}
		})
	}
}

// The check request carries the account's second password (design §1.7). It must
// be validated before the check is answered, and must never be logged.
func TestNameChangeCheckValidatesTheSpwAndNeverLogsIt(t *testing.T) {
	env := newCheckHandlerEnv(t)
	env.dispatchNameCheck("Victor", "s3cr3t")

	if !env.spwWasValidated() {
		t.Fatal("expected the SPW to be validated against the account before answering")
	}
	if strings.Contains(env.capturedLogs(), "s3cr3t") {
		t.Fatal("the SPW reached a log line")
	}
}

func TestWorldTransferCheckSurfacesTheGateReason(t *testing.T) {
	env := newCheckHandlerEnv(t)
	env.eligibilityReturns(false, "is_guild_master")
	env.dispatchWorldCheck(world.Id(2))
	if got := env.lastResultReason(); got != "is_guild_master" {
		t.Fatalf("reason = %s, want is_guild_master", got)
	}
}

// FR-4.7: the last-character-in-source-world case warns via pink text, and the
// warning is advisory — it does not make the check fail.
func TestWorldTransferCheckWarnsWhenStrandingStorage(t *testing.T) {
	env := newCheckHandlerEnv(t)
	env.eligibilityReturns(true, "")
	env.lastCharacterInSourceWorld(true)
	env.dispatchWorldCheck(world.Id(2))

	if !env.wrotePinkText() {
		t.Fatal("expected a pink-text storage warning")
	}
	if env.lastResultReason() != "available" {
		t.Fatal("the warning must not make the check fail")
	}
}
```

- [ ] **Step 2: Run the tests and confirm they fail**

Run: `go test ./socket/handler/ -run 'TestNameChangeCheck|TestWorldTransferCheck' -v`
Expected: FAIL — handlers undefined.

- [ ] **Step 3: Write the two handlers**

Each decodes its body, validates the SPW against the account (atlas-account), calls the atlas-character endpoint, maps the response to the derived wire reason code (Task 17/18's mapping), and writes the result packet. Log the outcome at info with tenant, character and reason — and never the SPW; the codec's redacting `String()` (Task 15 Step 3) makes the routine `l.Debugf("[%s] read [%s]", ...)` line safe.

- [ ] **Step 4: Add the FR-4.7 pink-text warning**

Resolve whether the character is the account's last in the source world, and if so announce `writer.WorldMessagePinkTextBody` (`services/atlas-channel/atlas.com/channel/socket/writer/world_message.go:107`) stating that storage in the source world will become inaccessible. Storage is keyed `(tenant, world, account)` and shared by every character the account owns in that world, so it never moves (FR-4.6) — the warning is the whole mitigation, and it is advisory: it does not block.

- [ ] **Step 5: Register both handlers in the templates**

Only the versions where the op is not `⬜ n-a`: `NAME_TRANSFER` on v83/v84/v87/v92/v95/jms_v185, `WORLD_TRANSFER` on v83/v84/v87/v92/v95. Sorted `opCode` position, non-empty `fname`, and a **validator** — a handler with a missing validator is silently dropped at config load. Run both template guards.

- [ ] **Step 6: Run the tests and confirm they pass**

Run: `go test -race ./... && go vet ./...` in atlas-channel; `tools/template-opcode-order-guard.sh` and `tools/template-duplicate-binding-guard.sh` from the repo root.
Expected: PASS, clean, guards exit 0.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-channel/ services/atlas-configurations/seed-data/templates/
git commit -m "feat(task-227): name-change and world-transfer availability checks"
```

### Task 27: The `PENDING_CHANGE_RESOLVED` consumer — CANCEL_* packets plus pink text

Design §3.9: atlas-channel writes the version-appropriate `CANCEL_*` packet **and** a pink-text world message. This is belt and braces against OQ-9 — if the client ignores the `CANCEL_*` packet outside the cash-shop UI, the player still learns why their coupon came back. An unexplained coupon reappearing in the cash inventory is the failure this prevents.

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/kafka/consumer/pendingchange/{consumer.go,kafka.go}`
- Create: `services/atlas-channel/atlas.com/channel/kafka/consumer/pendingchange/consumer_test.go`
- Modify: `services/atlas-channel/atlas.com/channel/main.go` (register the consumer)

**Interfaces:**
- Consumes: `EVENT_TOPIC_CHARACTER_PENDING_CHANGE` / `PENDING_CHANGE_RESOLVED` with `ResolvedEventBody` (Task 4). The contract struct is duplicated here across a module boundary — mirror the field names and json tags **exactly**, as the trade/mist/npc-shop mirrors do, or the body decodes zero-valued at runtime with no build error.
- Produces: `InitConsumers(l)(cmf)(consumerGroupId)` and `InitHandlers(l)(rf)` following `services/atlas-character/atlas.com/character/kafka/consumer/character/consumer.go` in shape.

- [ ] **Step 1: Write the failing consumer test**

```go
// FR-2.9: an operator cancellation reaches an online player as the CANCEL_*
// packet AND as pink text (design §3.9 belt-and-braces for OQ-9).
func TestResolvedCancellationWritesBothThePacketAndPinkText(t *testing.T) {
	env := newPendingChangeConsumerEnv(t)
	env.characterOnline(1)

	env.deliver(resolvedEvent{
		CharacterId: 1, ChangeType: "NAME_CHANGE",
		Status: "CANCELLED", Reason: "operator_cancelled", RequestedName: "Whiskey",
	})

	if !env.wrote("CANCEL_NAME_CHANGE_RESULT") {
		t.Fatal("expected CANCEL_NAME_CHANGE_RESULT")
	}
	if !env.wrotePinkTextContaining("Whiskey") {
		t.Fatal("expected pink text naming the requested value")
	}
}

// FR-2.7: a name change invalidated because someone else took the name uses the
// BY_OTHER packet specifically, not the generic cancel result.
func TestNameTakenRejectionUsesCancelByOther(t *testing.T) {
	env := newPendingChangeConsumerEnv(t)
	env.characterOnline(1)
	env.deliver(resolvedEvent{
		CharacterId: 1, ChangeType: "NAME_CHANGE",
		Status: "REJECTED", Reason: "name_taken", RequestedName: "Xray",
	})
	if !env.wrote("CANCEL_NAME_CHANGE_BY_OTHER") {
		t.Fatal("expected CANCEL_NAME_CHANGE_BY_OTHER")
	}
	if env.wrote("CANCEL_NAME_CHANGE_RESULT") {
		t.Fatal("must not also send the generic cancel result")
	}
}

func TestWorldTransferResolutionUsesTheTransferCancelPacket(t *testing.T) {
	env := newPendingChangeConsumerEnv(t)
	env.characterOnline(1)
	env.deliver(resolvedEvent{
		CharacterId: 1, ChangeType: "WORLD_TRANSFER",
		Status: "EXPIRED", Reason: "expired", DestinationWorldId: 2,
	})
	if !env.wrote("CANCEL_TRANSFER_WORLD_RESULT") {
		t.Fatal("expected CANCEL_TRANSFER_WORLD_RESULT")
	}
}

// An APPLIED resolution is not a cancellation and must send neither.
func TestAppliedResolutionSendsNoCancelPacket(t *testing.T) {
	env := newPendingChangeConsumerEnv(t)
	env.characterOnline(1)
	env.deliver(resolvedEvent{CharacterId: 1, ChangeType: "NAME_CHANGE", Status: "APPLIED"})
	if env.wroteAnyCancelPacket() {
		t.Fatal("APPLIED must not produce a cancel notification")
	}
}

// No live session: nothing is written and nothing is acked, so atlas-character
// leaves notified_at null and re-emits at the player's next login.
func TestOfflineCharacterWritesNothing(t *testing.T) {
	env := newPendingChangeConsumerEnv(t)
	env.characterOffline(1)
	env.deliver(resolvedEvent{
		CharacterId: 1, ChangeType: "NAME_CHANGE",
		Status: "CANCELLED", Reason: "operator_cancelled",
	})
	if env.wroteAnything() {
		t.Fatal("expected no writes for an offline character")
	}
}
```

- [ ] **Step 2: Run the tests and confirm they fail**

Run: `go test ./kafka/consumer/pendingchange/ -v`
Expected: FAIL — package does not exist.

- [ ] **Step 3: Write the consumer**

Packet selection: `NAME_CHANGE` + `REJECTED`/`name_taken` → `CANCEL_NAME_CHANGE_BY_OTHER`; `NAME_CHANGE` otherwise-terminal → `CANCEL_NAME_CHANGE_RESULT`; `WORLD_TRANSFER` terminal → `CANCEL_TRANSFER_WORLD_RESULT`; `APPLIED` → neither. Pink text accompanies every cancel packet and names the requested value, so the message is self-explanatory without the packet.

The consumer must ack only after writing, so that atlas-character's `notified_at` stamp reflects an actual delivery. When there is no live session it returns without writing and without stamping — the login catch-up (Task 9) handles it.

- [ ] **Step 4: Register the consumer in main.go**

Follow the existing consumer registrations in atlas-channel's `main.go`.

- [ ] **Step 5: Register the three CANCEL_* writers in the templates**

`CANCEL_NAME_CHANGE_RESULT` and `CANCEL_TRANSFER_WORLD_RESULT` on v61…v95; `CANCEL_NAME_CHANGE_BY_OTHER` on v72…v95. Sorted `opCode` position, non-empty `fname` (the seed template writers require it and validate against an exact corpus count). Run both template guards.

- [ ] **Step 6: Run the tests and confirm they pass**

Run: `go test -race ./... && go vet ./...` in atlas-channel; both template guards from the repo root.
Expected: PASS, clean, exit 0.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-channel/ services/atlas-configurations/seed-data/templates/
git commit -m "feat(task-227): deliver pending-change resolutions as CANCEL_* plus pink text"
```

### Task 28: The operator-cancel-unreachability guard

> **REWRITTEN by controller ruling, 2026-08-14. Do not restore the previous text.**
> The original task built a guard asserting *"no client cancel path exists — the
> cancel route is operator-only REST."* **That property is now proven FALSE.**
> Commit `4a5d9ff65` landed a real client-initiated cancel path, on the strength of
> two IDA derivations committed in this task folder:
> [`cancel-entry-point.md`](cancel-entry-point.md) and
> [`cancel-confirm-semantics.md`](cancel-confirm-semantics.md). The client's
> name-change/world-transfer coupon item-use arm builds a double-confirm
> `CANCELREQUESTS_*` dialog chain and, only on full confirmation, appends an
> invariant trailing byte to the generic cash item-use packet.
>
> A green guard asserting a falsehood is worse than no guard: it manufactures
> confidence in a property the code does not have. The **security intent** of the
> original task is real and worth keeping, so the property is narrowed rather than
> dropped — see below.

**The property this guard actually enforces:** the **operator** cancel route is not
reachable from any socket handler. There are two distinct cancel routes, and only
one of them is operator-privileged (verified at rewrite time):

| Route | Reason emitted | Who may reach it |
|---|---|---|
| `POST /characters/{id}/pending-changes/cancel` (self-scoped; resolves the caller's own PENDING record by `(characterId, type)`) | `player_cancelled` — `services/atlas-character/atlas.com/character/pending_change/processor.go:349` | the game client, via the coupon item-use arm. **Legitimate.** |
| `DELETE` on the id-based operator route | `operator_cancelled` — `services/atlas-character/atlas.com/character/pending_change/resource.go:131,164` | operators over REST **only**. Never a socket handler. |

The security property (design §8) is that a game-client packet path may never reach
the *second* row. The first row is the feature working as derived.

**Files:**
- Create: `tools/operator-cancel-path-guard.sh`
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/operator_cancel_unreachable_test.go`
- Modify: `CLAUDE.md` (register the guard as build-verification item 16)

Patterns to copy: an existing repo guard under `tools/` for the shell script's
shape and exit-code convention; `tools/template-duplicate-binding-guard.sh` for
template-walking.

- [ ] **Step 1: Write the guard test**

Assert BOTH halves, and scope each to the operator route only:

  (a) **No Go file under `services/atlas-channel/atlas.com/channel/socket/handler/`
      may reference the operator route or its reason.** Match on the
      `operator_cancelled` reason string and on an id-bearing `DELETE` against
      `pending-changes`. It must NOT match the legitimate self-scoped
      `POST .../pending-changes/cancel` or the `player_cancelled` reason — add an
      explicit positive test that a handler using the self-scoped route passes the
      guard, so a future tightening cannot silently re-ban the real feature.

  (b) **No tenant socket-config template may bind an operator-cancel handler.**
      Walk the templates as `tools/template-duplicate-binding-guard.sh` does.
      Note the `Cancel*ResultWriter` constants in
      `libs/atlas-packet/cash/clientbound/` are **clientbound writers**; a template
      naming one as a *handler* is the defect this catches.

Cite both derivation docs by path in the test file's header comment and in the
shell script's header, with one line on why the original property was withdrawn.
Nobody should be able to "restore" the old assertion without reading why it fell.

- [ ] **Step 2: Prove the guard can fail**

Run it on the clean tree → PASS. Then temporarily introduce a violation of each
half (an `operator_cancelled` reference in a handler file; a template binding an
operator-cancel handler), re-run → FAIL each time. Revert both. **A guard that
cannot fail is not a guard** — and this task exists precisely because the previous
version of it could only ever pass. Record both red runs in your report.

- [ ] **Step 3: Write `tools/operator-cancel-path-guard.sh`**

Portable POSIX shell, same two checks tree-wide so the property holds in CI and not
only in one module's test run. Exit non-zero with the offending file and line.

- [ ] **Step 4: Register it in CLAUDE.md**

Add as build-verification item 16, in the same style as items 13–15: what it bans,
why the failure is silent otherwise, and when to run it (whenever `socket/handler`
or a template changed). State the narrowed property, not the withdrawn one.

- [ ] **Step 5: Run the guard**

Run: `tools/operator-cancel-path-guard.sh`
Expected: exit 0.

- [ ] **Step 6: Commit**

```bash
git add tools/operator-cancel-path-guard.sh CLAUDE.md services/atlas-channel/
git commit -m "test(task-227): machine-check that the OPERATOR cancel route is unreachable from socket handlers"
```

---

## Phase F — `NAME_CHANGED` consumers

`nameChangedEventProvider` has been emitted since before this task, and a repo-wide grep finds **zero consumers outside atlas-character**. A stale name copy is the primary failure mode of this feature.

Design §1.4 surveyed every service's `entity.go` for a persisted denormalised name. Four services need a consumer:

| service | column | file |
|---|---|---|
| atlas-guilds | `members.name` | `services/atlas-guilds/atlas.com/guilds/guild/member/entity.go:16` |
| atlas-buddies | `buddy.character_name` | `services/atlas-buddies/atlas.com/buddies/buddy/entity.go:28` |
| atlas-rankings | `ranking.name` | `services/atlas-rankings/atlas.com/rankings/ranking/entity.go:23` |
| atlas-mts | `listing.seller_name` | `services/atlas-mts/atlas.com/mts/listing/entity.go:47` |

**atlas-parties, atlas-messengers, atlas-marriages and atlas-trades need no consumer**, and this is a reasoned exclusion rather than an oversight: they hold the name only in in-memory registries rebuilt at login (`parties/character/registry.go:36`, `messengers/character/registry.go:35`, `marriages/character/model.go:5`, `trades/trade/builder.go:55`). Because FR-2.4 forbids applying a rename to a live character, those caches are always rebuilt *after* the rename lands. **This is load-bearing on FR-2.4** — if the offline-only constraint is ever relaxed, those four become required consumers. Record that in each service's absence rather than nowhere.

**atlas-merchant is excluded by decision** (design §3.8, §10): its `blacklists.name` and `merchant_visits.name` are name-**keyed** rows, not name-carrying rows. Rewriting a blacklist entry on rename is a behaviour change to a moderation feature ("does a blocked player escape their block by renaming?" is a product question), and `merchant_visits` is a log.

### Task 29: atlas-guilds consumes `NAME_CHANGED`

**Files:**
- Create: `services/atlas-guilds/atlas.com/guilds/kafka/consumer/character/{consumer.go,kafka.go}`
- Create: `services/atlas-guilds/atlas.com/guilds/kafka/consumer/character/consumer_test.go`
- Modify: `services/atlas-guilds/atlas.com/guilds/main.go`

**Interfaces:**
- Consumes: `EVENT_TOPIC_CHARACTER_STATUS` / `NAME_CHANGED` with body `{oldName, newName}` — mirror the field names and json tags from `services/atlas-character/atlas.com/character/kafka/message/character/kafka.go` exactly; this is a cross-module contract copy with no compile-time link.
- Produces: `handleCharacterNameChanged(db *gorm.DB) message.Handler[character2.StatusEvent[character2.StatusEventNameChangedBody]]`.

- [ ] **Step 1: Write the failing test**

```go
func TestNameChangedUpdatesTheGuildRosterName(t *testing.T) {
	db := newTestDB(t)
	seedGuildMember(t, db, 5 /*guild*/, 1 /*character*/, "Yankee")

	handleCharacterNameChanged(db)(l, ctx, character2.StatusEvent[character2.StatusEventNameChangedBody]{
		TransactionId: uuid.New(), CharacterId: 1, WorldId: world.Id(0),
		Type:          character2.StatusEventTypeNameChanged,
		Body:          character2.StatusEventNameChangedBody{OldName: "Yankee", NewName: "Zulu"},
	})

	if got := guildMemberName(t, db, 5, 1); got != "Zulu" {
		t.Fatalf("roster name = %s, want Zulu", got)
	}
}

// At-least-once delivery: a redelivered event must be a harmless no-op.
func TestNameChangedIsIdempotent(t *testing.T) {
	db := newTestDB(t)
	seedGuildMember(t, db, 5, 1, "Yankee")
	ev := character2.StatusEvent[character2.StatusEventNameChangedBody]{
		TransactionId: uuid.New(), CharacterId: 1, WorldId: world.Id(0),
		Type:          character2.StatusEventTypeNameChanged,
		Body:          character2.StatusEventNameChangedBody{OldName: "Yankee", NewName: "Zulu"},
	}
	handleCharacterNameChanged(db)(l, ctx, ev)
	handleCharacterNameChanged(db)(l, ctx, ev)
	if got := guildMemberName(t, db, 5, 1); got != "Zulu" {
		t.Fatalf("roster name = %s after redelivery, want Zulu", got)
	}
}

// A character with no guild membership must not error or create a row.
func TestNameChangedForANonMemberIsANoOp(t *testing.T) {
	db := newTestDB(t)
	handleCharacterNameChanged(db)(l, ctx, character2.StatusEvent[character2.StatusEventNameChangedBody]{
		TransactionId: uuid.New(), CharacterId: 99, WorldId: world.Id(0),
		Type:          character2.StatusEventTypeNameChanged,
		Body:          character2.StatusEventNameChangedBody{OldName: "A", NewName: "B"},
	})
	if guildMemberRowCount(t, db) != 0 {
		t.Fatal("handler created a membership row")
	}
}
```

- [ ] **Step 2: Run the tests and confirm they fail**

Run: `cd services/atlas-guilds/atlas.com/guilds && go test ./kafka/consumer/character/ -v`
Expected: FAIL — package does not exist.

- [ ] **Step 3: Write the handler**

A narrow `UPDATE members SET name = ? WHERE tenant_id = ? AND character_id = ?`, idempotent by construction. Guard on `e.Type != StatusEventTypeNameChanged` and return early, since the status topic carries every character status event. Log at info with tenant, character, old and new name.

- [ ] **Step 4: Register the consumer and handler in main.go**

Add the status-topic subscription and the handler registration, following the service's existing `InitConsumers` / `InitHandlers` wiring.

- [ ] **Step 5: Run the tests and confirm they pass**

Run: `go test -race ./... && go vet ./...`
Expected: PASS, clean.

- [ ] **Step 6: `docker buildx bake atlas-guilds`**

Required whenever a service's `go.mod` is touched. If no dependency was added and `go.mod` is unchanged, note that and skip.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-guilds/
git commit -m "feat(task-227): atlas-guilds consumes NAME_CHANGED for roster names"
```

### Task 30: atlas-buddies consumes `NAME_CHANGED`

Same six-step shape as Task 29, updating `buddy.character_name` (`services/atlas-buddies/atlas.com/buddies/buddy/entity.go:28`).

One extra consideration: a buddy row is keyed by the *owner* and names the *buddy*, so the update matches on the **buddy's** character id, not the owner's. Write a test that seeds two owners who both have character 1 as a buddy and asserts both rows update — a `WHERE character_id = ?` written against the wrong column updates zero rows and passes a single-row test.

```go
func TestNameChangedUpdatesEveryOwnersCopyOfTheBuddyName(t *testing.T) {
	db := newTestDB(t)
	seedBuddy(t, db, 10 /*owner*/, 1 /*buddy*/, "Yankee")
	seedBuddy(t, db, 11 /*owner*/, 1 /*buddy*/, "Yankee")

	handleCharacterNameChanged(db)(l, ctx, nameChangedEvent(1, "Yankee", "Zulu"))

	for _, owner := range []uint32{10, 11} {
		if got := buddyName(t, db, owner, 1); got != "Zulu" {
			t.Fatalf("owner %d copy = %s, want Zulu", owner, got)
		}
	}
}
```

Verify with `go test -race ./... && go vet ./...` and commit as `feat(task-227): atlas-buddies consumes NAME_CHANGED`.

### Task 31: atlas-rankings consumes `NAME_CHANGED`

Same six-step shape, updating `ranking.name` (`services/atlas-rankings/atlas.com/rankings/ranking/entity.go:23`).

A ranking row may not exist for a character yet — assert the no-op case explicitly, as in Task 29's third test. Commit as `feat(task-227): atlas-rankings consumes NAME_CHANGED`.

### Task 32: atlas-mts consumes `NAME_CHANGED`

Same six-step shape, updating `listing.seller_name` (`services/atlas-mts/atlas.com/mts/listing/entity.go:47`).

A seller may hold several listings; assert that **all** of them update:

```go
func TestNameChangedUpdatesEverySellerListing(t *testing.T) {
	db := newTestDB(t)
	seedListing(t, db, 1 /*seller*/, "Yankee")
	seedListing(t, db, 1, "Yankee")

	handleCharacterNameChanged(db)(l, ctx, nameChangedEvent(1, "Yankee", "Zulu"))

	for _, id := range listingIdsForSeller(t, db, 1) {
		if got := listingSellerName(t, db, id); got != "Zulu" {
			t.Fatalf("listing %d seller = %s, want Zulu", id, got)
		}
	}
}
```

Commit as `feat(task-227): atlas-mts consumes NAME_CHANGED`.

---

## Phase G — atlas-ui: the operator panel

FR-2.10 scopes this to **read + cancel only**. It must not be able to create a rename or transfer request, nor edit a requested value — operators do not grant renames or transfers from the console.

### Task 33: The service layer and the React Query hooks

**Files:**
- Create: `services/atlas-ui/src/services/api/pending-changes.service.ts`
- Create: `services/atlas-ui/src/lib/hooks/api/usePendingChanges.ts`
- Create: `services/atlas-ui/src/lib/hooks/api/__tests__/usePendingChanges.test.ts`
- Modify: `services/atlas-ui/src/services/api/index.ts` (export the service)

**Interfaces:**
- Consumes: atlas-character `GET|POST /characters/{characterId}/pending-changes`, `DELETE .../{id}` (Task 7). The UI uses only GET and DELETE.
- Produces:
  ```ts
  export type PendingChangeType = "NAME_CHANGE" | "WORLD_TRANSFER";
  export type PendingChangeStatus = "PENDING" | "APPLIED" | "CANCELLED" | "REJECTED" | "EXPIRED";

  export interface PendingChange {
    id: string;
    characterId: number;
    type: PendingChangeType;
    status: PendingChangeStatus;
    requestedName: string;
    destinationWorldId: number;
    sourceWorldId: number;
    reason: string;
    createdAt: string;
    expiresAt: string;
    resolvedAt: string | null;
  }

  export const pendingChangesService = {
    getByCharacterId(characterId: string): Promise<PendingChange[]>,
    cancel(characterId: string, id: string): Promise<void>,
  };

  export const pendingChangeKeys = {
    all: ["pending-changes"] as const,
    detail: (tenantId: string | undefined, characterId: string) => readonly unknown[],
  };
  export function usePendingChanges(tenant: Tenant | null | undefined, characterId: string): UseQueryResult<PendingChange[], Error>;
  export function useCancelPendingChange(): UseMutationResult<void, Error, { tenant: Tenant | null | undefined; characterId: string; id: string }>;
  ```

- [ ] **Step 1: Write the failing hook test**

```ts
import { describe, expect, it, vi } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { usePendingChanges, useCancelPendingChange, pendingChangeKeys } from "../usePendingChanges";
import { pendingChangesService } from "@/services/api/pending-changes.service";

describe("usePendingChanges", () => {
  it("is disabled until a tenant is selected", () => {
    const spy = vi.spyOn(pendingChangesService, "getByCharacterId");
    const { result } = renderHook(() => usePendingChanges(null, "1"), { wrapper });
    expect(spy).not.toHaveBeenCalled();
    expect(result.current.isPending).toBe(true);
  });

  it("invalidates the detail key after a cancel so the panel reflects CANCELLED without a reload", async () => {
    vi.spyOn(pendingChangesService, "cancel").mockResolvedValue(undefined);
    const qc = makeQueryClient();
    const invalidate = vi.spyOn(qc, "invalidateQueries");
    const { result } = renderHook(() => useCancelPendingChange(), { wrapper: wrapperWith(qc) });

    await result.current.mutateAsync({ tenant, characterId: "1", id: "pc-1" });

    await waitFor(() =>
      expect(invalidate).toHaveBeenCalledWith({
        queryKey: pendingChangeKeys.detail(tenant.id, "1"),
      }),
    );
  });
});
```

- [ ] **Step 2: Run it and confirm it fails**

Run: `cd services/atlas-ui && npm run test -- usePendingChanges`
Expected: FAIL — module not found. (`npm` needs nvm 22 in this repo; use it.)

- [ ] **Step 3: Write the service**

Follow `teleport-rocks.service.ts` exactly, including its `unwrap` helper: `api.getOne` unwraps the JSON:API `.data` envelope but `api.post` / `api.delete` pass the raw response through, so read and write calls resolve to different shapes and both must be normalized. No `any` anywhere in the layer — the envelope is typed with a `PendingChangeResource` interface.

- [ ] **Step 4: Write the hooks**

Follow `useTeleportRocks.ts`: a `pendingChangeKeys` object, `enabled: !!tenant?.id && !!characterId`, and `invalidateQueries` (not `setQueryData`) on cancel success — the DELETE returns no body, so there is nothing to write into the cache and a refetch is the correct move.

- [ ] **Step 5: Run the tests and confirm they pass**

Run: `npm run test -- usePendingChanges`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-ui/src/services/api/ services/atlas-ui/src/lib/hooks/api/
git commit -m "feat(task-227): atlas-ui pending-changes service and hooks"
```

### Task 34: The panel, the confirm dialog, and the page wiring

**Files:**
- Create: `services/atlas-ui/src/components/features/characters/PendingChangesPanel.tsx`
- Create: `services/atlas-ui/src/components/features/characters/CancelPendingChangeDialog.tsx`
- Create: `services/atlas-ui/src/components/features/characters/__tests__/PendingChangesPanel.test.tsx`
- Modify: `services/atlas-ui/src/pages/CharacterDetailPage.tsx`

**Interfaces:**
- Consumes: `usePendingChanges`, `useCancelPendingChange`, `PendingChange` (Task 33).
- Produces: `<PendingChangesPanel characterId={string} />` and `<CancelPendingChangeDialog change={PendingChange} characterName={string} characterId={string} />`.

- [ ] **Step 1: Write the failing panel test**

```tsx
describe("PendingChangesPanel", () => {
  it("lists a pending name change with its requested value and expiry", async () => {
    mockPendingChanges([pendingNameChange({ requestedName: "Zulu", expiresAt: "2026-08-21T00:00:00Z" })]);
    render(<PendingChangesPanel characterId="1" />, { wrapper });

    expect(await screen.findByText("Name Change")).toBeInTheDocument();
    expect(screen.getByText("Zulu")).toBeInTheDocument();
    expect(screen.getByText(/PENDING/)).toBeInTheDocument();
  });

  it("shows the rejection reason on a resolved record so an operator can answer 'what happened to my coupon?'", async () => {
    mockPendingChanges([rejectedNameChange({ requestedName: "Zulu", reason: "name_taken" })]);
    render(<PendingChangesPanel characterId="1" />, { wrapper });

    expect(await screen.findByText(/REJECTED/)).toBeInTheDocument();
    expect(screen.getByText(/name.taken/i)).toBeInTheDocument();
  });

  it("offers Cancel only on a PENDING record", async () => {
    mockPendingChanges([
      pendingNameChange({ id: "pc-1" }),
      appliedNameChange({ id: "pc-2" }),
    ]);
    render(<PendingChangesPanel characterId="1" />, { wrapper });

    const buttons = await screen.findAllByRole("button", { name: /cancel/i });
    expect(buttons).toHaveLength(1);
  });

  it("names the character and the requested value in the confirm dialog before cancelling", async () => {
    const cancel = vi.spyOn(pendingChangesService, "cancel").mockResolvedValue(undefined);
    mockPendingChanges([pendingNameChange({ id: "pc-1", requestedName: "Zulu" })]);
    render(<PendingChangesPanel characterId="1" />, { wrapper });

    await userEvent.click(await screen.findByRole("button", { name: /cancel/i }));

    const dialog = await screen.findByRole("alertdialog");
    expect(dialog).toHaveTextContent("Zulu");
    expect(cancel).not.toHaveBeenCalled();          // not until confirmed

    await userEvent.click(screen.getByRole("button", { name: /^cancel request$/i }));
    expect(cancel).toHaveBeenCalledWith("1", "pc-1");
  });

  it("exposes no create or edit affordance (FR-2.10 is read + cancel only)", async () => {
    mockPendingChanges([pendingNameChange({})]);
    render(<PendingChangesPanel characterId="1" />, { wrapper });
    await screen.findByText("Name Change");

    expect(screen.queryByRole("button", { name: /new|create|add|edit/i })).toBeNull();
  });

  it("renders an empty state rather than a broken table when there are no records", async () => {
    mockPendingChanges([]);
    render(<PendingChangesPanel characterId="1" />, { wrapper });
    expect(await screen.findByText(/no pending changes/i)).toBeInTheDocument();
  });
});
```

- [ ] **Step 2: Run the tests and confirm they fail**

Run: `npm run test -- PendingChangesPanel`
Expected: FAIL — component not found.

- [ ] **Step 3: Write the panel**

Follow `TeleportRockListCard.tsx` in structure: a shadcn/ui `Card` with `CardHeader`/`CardTitle`/`CardContent`, a `Badge` per status, and `toast` from `sonner` for outcomes. Tenant comes from `useTenant()`'s `activeTenant` — never a hard-coded tenant. Errors go through `createErrorFromUnknown` as the sibling components do.

Render the requested value per type: `requestedName` for `NAME_CHANGE`, the destination world for `WORLD_TRANSFER`. Show `createdAt` and `expiresAt` on pending rows, `resolvedAt` and `reason` on resolved ones.

- [ ] **Step 4: Write the confirm dialog**

A shadcn/ui `AlertDialog` — the destructive-confirm primitive used elsewhere in this codebase — naming both the character and the requested value in its description, so an operator cannot cancel the wrong record by muscle memory. The confirm action calls `useCancelPendingChange().mutateAsync`.

- [ ] **Step 5: Wire it into CharacterDetailPage**

Add the import alongside the other `@/components/features/characters/*` imports and render `<PendingChangesPanel characterId={String(id)} />` in the page's card column, following the placement of `TeleportRockListCard`.

- [ ] **Step 6: Run the tests and the build**

Run:
```bash
cd services/atlas-ui
npm run test
npm run build
```
Expected: PASS both. `npm run build` type-checks the tests in this repo, so a passing Vitest run alone is **not** sufficient verification for a UI change here.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-ui/src/
git commit -m "feat(task-227): operator pending-changes panel with confirmed cancel"
```

---

## Phase H — Verification and review

### Task 35: Full gate sweep

Every command below must be run and its output read. A gate that was not run is not a gate, and a spot-check presented as a full sweep is a false "verified."

- [ ] **Step 1: Per-module Go tests and vet**

For each changed module — atlas-character, atlas-character-factory, atlas-channel, atlas-saga-orchestrator, atlas-tenants, atlas-guilds, atlas-buddies, atlas-rankings, atlas-mts, `libs/atlas-saga`, `libs/atlas-packet`:

```bash
go test -race ./...
go vet ./...
go build ./...
```

- [ ] **Step 2: Docker bake for every service whose `go.mod` was touched**

```bash
docker buildx bake atlas-character
docker buildx bake atlas-channel
docker buildx bake atlas-saga-orchestrator
docker buildx bake atlas-tenants
# plus atlas-guilds / atlas-buddies / atlas-rankings / atlas-mts / atlas-character-factory
# if and only if their go.mod changed
```

`go build` against the workspace `go.work` will **not** catch a missing `COPY libs/...` line in the shared root `Dockerfile` — only the bake will. Determine the list with `git diff --name-only main... -- '**/go.mod'` rather than from memory.

- [ ] **Step 3: Repo-root guards**

```bash
tools/redis-key-guard.sh
tools/goroutine-guard.sh
tools/template-opcode-order-guard.sh
tools/template-duplicate-binding-guard.sh
tools/template-movement-types-guard.sh
tools/skill-job-id-guard.sh
tools/buff-duration-guard.sh
tools/cancel-path-guard.sh          # added by Task 28
tools/lint.sh --check
```

`tools/lint.sh --check` false-fails without nvm — load nvm 22 first. Run `tools/lint.sh` (no flags) to fix formatting in place before committing. `tools/service-registration-guard.sh` is **not** required: this task adds no new service.

- [ ] **Step 4: atlas-ui**

```bash
cd services/atlas-ui && npm run build && npm run test
```

- [ ] **Step 5: Packet matrix**

Re-run the Task 23 Step 1 grep and confirm 59 `✅` still stand after all subsequent commits. A registry `fname` edit stales the matrix, and Phase E registered writers — so this re-check is not redundant.

- [ ] **Step 6: Record the results**

Write the actual command output — not a paraphrase — into `docs/tasks/task-227-cash-name-change-world-transfer/verification.md`. Quote failures verbatim and fix them; do not summarise a failure as a caveat.

- [ ] **Step 7: Commit**

```bash
git add docs/tasks/task-227-cash-name-change-world-transfer/verification.md
git commit -m "docs(task-227): full verification sweep results"
```

### Task 36: Code review before PR

- [ ] **Step 1: Confirm the worktree is the right one and is clean**

```bash
git rev-parse --show-toplevel   # must end with /.worktrees/task-227-cash-name-change-world-transfer
git branch --show-current       # must be task-227-cash-name-change-world-transfer
git status --short               # must be empty
```

- [ ] **Step 2: Invoke `superpowers:requesting-code-review`**

It dispatches the appropriate subset. All three apply here: `plan-adherence-reviewer` (this plan), `backend-guidelines-reviewer` (Go changed across nine modules), and `frontend-guidelines-reviewer` (atlas-ui TS changed). Pin the reviewers to Sonnet per the project's model preference for review workflows. Findings land in `docs/tasks/task-227-cash-name-change-world-transfer/audit.md`.

- [ ] **Step 3: Verify the reviewers wrote into the worktree, not main**

```bash
ls docs/tasks/task-227-cash-name-change-world-transfer/audit.md
git -C ../.. status --short      # main repo must be untouched
```

- [ ] **Step 4: Resolve every finding**

Address each on this branch. Do not open a follow-up task to avoid completing this one, and do not accept a finding as a documented gap when the blocker is something this branch can produce.

- [ ] **Step 5: Re-run the Task 35 gates after the fixes**

Any code change after a verification sweep invalidates it.

- [ ] **Step 6: Commit and open the PR**

```bash
git add docs/tasks/task-227-cash-name-change-world-transfer/
git commit -m "docs(task-227): code review findings and resolutions"
```

Then open the PR with `env -u GH_TOKEN -u GITHUB_TOKEN gh pr create`.

---

## Self-review notes

**Spec coverage.** Every PRD functional requirement maps to a task:

| FR | Task |
|---|---|
| FR-1.1, FR-1.2, FR-1.3 | 24 (dispatch arm, version-scoped types, derivation-sourced item ids) |
| FR-1.4 | 25 |
| FR-1.5, FR-1.6 | 15, 16, 17, 18, 19, 26 |
| FR-1.7 | 20, 21, 22, 27 |
| FR-1.8 | 15–22 (per-op registration), 23 (guards) |
| FR-2.1, FR-2.2 | 2, 3 |
| FR-2.3 | 2 (index), 3 (sentinel), 7 (409) |
| FR-2.4 | 9 |
| FR-2.5 | 7 (DELETE), 34 (UI) |
| FR-2.6 | 8 (config), 10 (sweep) |
| FR-2.7 | 6, 9, 27 |
| FR-2.8 | 6 (transition-gated emission + idempotency test) |
| FR-2.9 | 6, 9, 27 |
| FR-2.10 | 33, 34 |
| FR-3.1, FR-3.2, FR-3.3, FR-3.4, FR-3.5 | 5, 2 (reservation index), 6 |
| FR-3.6 | 29, 30, 31, 32 |
| FR-3.7 | satisfied by FR-2.4 — no live-broadcast path exists to build |
| FR-4.1, FR-4.2, FR-4.3 | 11 |
| FR-4.4 | 11 — **blocking rather than settling** (design §3.6, R5) |
| FR-4.5 | 13 (character world update); inventory step is a no-op (design §1.5) |
| FR-4.6 | no work — storage is never touched; asserted by Task 14's test env |
| FR-4.7 | 26 |
| FR-4.8 | 12, 13, 14 |
| FR-4.9 | 13, 14 |
| FR-5.1, FR-5.2, FR-5.3 | 24, 25, 26, 27 |
| §5 API surface | 7, 11 |
| §6 data model | 2 |
| §8 non-functional | throughout; tenancy and idempotency have dedicated tests in 6, 9, 29–32 |

Every PRD §10 acceptance criterion has a test named in the task that owns it. The two criteria that are properties rather than behaviours — "no serverbound handler is wired to the cancel endpoint" and "a mid-saga failure leaves the character wholly in the source world" — are machine-checked in Tasks 28 and 14 respectively, per the PRD's own insistence that they be proven by test and not by inspection.

**Open questions.** OQ-2, OQ-3, OQ-4, OQ-6 were resolved in design and are recorded as such in the Deviations section and in `context.md`. OQ-1, OQ-7, OQ-9 are resolved by Task 1's derivation, which is why Task 1 blocks everything. OQ-5 is resolved by design's choice of a query-level tenant-wide check with no new index on `characters` (Task 5, Step 3) — correct whether or not live duplicates exist. OQ-8 is resolved by the `notified_at` column (Tasks 2, 6, 9).

**Risks.** R1 is Task 1, scheduled first and blocking. R2 has two independent evidence routes (Task 1, Step 6) and stops with a BLOCKED report rather than a guess if neither answers. R3 is decomposed into eight per-op tasks with per-cell `packet-verifier` fan-out, never one task. R4 is smaller than stated — the consumer already exists (Task 9, Step 5 verifies the group id). R5 is a stated departure from FR-4.4 with its reasoning in Task 11's preamble; if the intent was genuinely auto-settlement, that is the decision to revisit before Task 11 begins.

---

## Phase H — Option A rework of Task 25's done-body emit

**Added by controller ruling, 2026-08-14.** Task 25 (`98213d81e`) shipped
`BUY_NAME_CHANGE` / `BUY_WORLD_TRANSFER` arms that emit a done body carrying a
`CashInventoryItem` with `CashId = 0`. That is **confirmed unsafe** — see
[`cash-inventory-item-zero-fields.md`](cash-inventory-item-zero-fields.md): the
client `DecodeBuffer`s the item into its locker array (v83 `0x47bccb`, `0x47bfa2`),
and `CCSWnd_Locker::OnMouseButton` (`0x4b053b`) later reads the clicked slot's
`CashId` and echoes it back on the locker-withdraw op, which our
`MoveFromCashInventory` consumes as the serial number. Two `CashId == 0` entries
make that withdraw ambiguous, and in every case a fabricated id crosses the wire.

Nothing in this codebase fabricates a `CashId`; every sibling site resolves a real
`asset.Model` and reads `.CashId()`. These two arms cannot, because the flow as
designed (design §3.1/§5.1) **never creates a cash asset at all**. Task 25
implemented the design faithfully — the design is what is wrong.

**The user ruled Option A: build it properly.** Route `BUY_*` through
`cashshop.RequestPurchase` so a real asset exists, and emit the done body from the
purchase-success consumer using the real `AssetId`.

### Two resolved blockers — do not relitigate these

**Currency: DERIVED, not a guess.** See
[`buy-currency-derivation.md`](buy-currency-derivation.md). The client HARD-CODES
NX Prepaid for both ops. `CCashShop::OnBuyNameChange` (v83 `0x47031e`, v87
`0x47ab57`, v95 `0x491200`) and `CCashShop::OnBuyTransferWorld` (v83 `0x470480`)
build a confirmation dialog from the hard-coded literal
`"You will spend %d NX Prepaid.\r\nWould you like to proceed?"`, gated on a fixed
balance field which on the PDB-backed v95 IDB resolves **by name** to
`CCashShop::m_nPrepaidNXCash`. The send paths (v83 `0x47342f`, `0x473601`) carry no
currency field, matching our decoders exactly. The generic `BUY` op
(`CCashShop::OnBuy`) genuinely *does* decode `isPoints` + `currency` off the wire —
the two op families differ architecturally; this is not an oversight.
=> **Call `RequestPurchase` with `isPoints = false` and `currency = 0`.** Zero is
neither 1 nor 2, so `services/atlas-cashshop/atlas.com/cashshop/wallet/model.go:37-58`
falls through to prepaid. Do not invent a wallet selector on the wire.

**Ordering: INSERT-FIRST. Purchase-first is not available.** `Resolve()` mints a
refund only when `status != StatusApplied && m.HasAsset()`
(`pending_change/processor.go:287-306`), and `HasAsset()` is **false on the purchase
path by construction** (`entity.go:69-74`). atlas-cashshop has **no void/refund
command at all** — its full command set is RequestPurchase,
RequestInventory/Storage/CharacterSlot Increase*, Expire, OpenSurprise,
RequestCouponRedemption (`consumer.go:34-65`), and `Expire` deletes the asset
without ever crediting the wallet. So purchase-first plus a name-taken 409 leaves
the player charged with nothing to show, reversible only by building new refund
machinery. Insert-first plus a purchase failure releases the unpaid PENDING row via
the already-tested cancel path, minting no spurious refund.

### Task 37: A correlation id on the purchase command and its two outcome events

Today `handleStatusEventPurchase` keys only off `CharacterId` and **cannot tell a
name-change buy from any other concurrent BUY**; `ErrorEventBody` is equally
op-blind. Moving the done body to the consumer therefore requires a correlation
carrier. This is a cross-service wire-format change — the first on this branch —
so it lands as its own task, with no behaviour change, so the diff is reviewable in
isolation.

Precedent to copy: `OpenSurpriseCommandBody.TransactionId uuid.UUID`
(`kafka/message/cashshop/kafka.go:57-64` in **both** services) — an opaque UUID
minted by the caller. Note its success event does **not** echo it back; that is the
gap this task closes for the purchase family.

**Files:**
- Modify: `services/atlas-cashshop/atlas.com/cashshop/kafka/message/cashshop/kafka.go` (`RequestPurchaseCommandBody`, `PurchaseEventBody`, `ErrorEventBody`)
- Modify: `services/atlas-channel/atlas.com/channel/kafka/message/cashshop/kafka.go` — the **mirror** of the same three bodies
- Modify: `services/atlas-cashshop/atlas.com/cashshop/kafka/producer/cashshop/producer.go` (`PurchaseStatusEventProvider`, ~line 39, and the error-event provider beside it)
- Modify: `services/atlas-cashshop/atlas.com/cashshop/cashshop/processor.go` (`RequestPurchase`, ~lines 98-127) — thread the id from command to outcome
- Modify: `services/atlas-cashshop/atlas.com/cashshop/kafka/consumer/cashshop/consumer.go` — the RequestPurchase arm
- Tests beside each changed file.

Patterns to copy: `OpenSurpriseCommandBody` for the field shape and its doc comment
style; `producer.go:39-51` for the provider signature convention.

- [ ] **Step 1: Write the failing round-trip tests**

Assert the id survives command → processor → **both** outcome events, and that two
concurrent purchases for the SAME character are distinguishable by it. Assert the
zero UUID is accepted and means "no correlation" (every existing caller), so this
change is backward compatible on the wire.

- [ ] **Step 2: Add the field to all six struct definitions**

`TransactionId uuid.UUID \`json:"transactionId,omitempty"\`` on
`RequestPurchaseCommandBody`, `PurchaseEventBody` and `ErrorEventBody`, in **both**
services. The two services' copies are hand-mirrored, not generated — a field added
to one and not the other is silently dropped at the JSON boundary.

> **Verified at authoring time, and a live example of exactly that failure:**
> channel-side `PurchaseEventBody` already declares `ItemId uint32` (`kafka.go:128`)
> which the cashshop producer **never sets** (`producer.go:44-49`) and the channel
> consumer **never reads**. It is dead on both sides today, so leave it alone — but
> it is the proof that this mirror drifts silently. Add the new field to both.

- [ ] **Step 3: Thread it through the producer and processor**

`PurchaseStatusEventProvider` and the error provider take the id and set it. The
processor carries it from the command body to whichever outcome it emits. **Both**
arms — a failure that drops the correlation is the same defect as never adding it.

- [ ] **Step 4: Run the tests, then commit**

`go build ./... && go test ./... && go vet ./...` in each changed module.
`git commit -m "feat(task-227): correlate cash purchase outcomes with a transaction id"`

### Task 38: `BUY_*` charges the player through `RequestPurchase` (insert-first)

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_operation.go` — `handleBuyNameChange`, `handleBuyWorldTransfer`
- Modify: the channel-side cashshop processor that issues `RequestPurchase`
- Tests beside them.

- [ ] **Step 1: Write the failing tests**

Assert, for each of the two arms, in this order: (a) the PENDING record is inserted
**before** the purchase command is emitted; (b) the emitted command carries
`isPoints = false`, `currency = 0`, the right `serialNumber`, and a **non-zero**
`TransactionId`; (c) that id is recorded against the PENDING record so the consumer
can resolve back to it; (d) **no done body is written from the handler** — the
client is answered by the consumer now. Assert (d) explicitly: it is the whole
point of the rework and the easiest thing to leave behind.

- [ ] **Step 2: Implement**

Insert-first, then emit. Mint the `TransactionId` at the handler. Do **not** await
the outcome — `RequestPurchase` is fully async; the handler discards the return and
every outcome arrives as a status event.

- [ ] **Step 3: Run the tests, then commit**

### Task 39: The purchase-outcome consumer answers the client

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/cashshop/consumer.go` — `handleStatusEventPurchase` and the error arm
- Modify: `services/atlas-channel/atlas.com/channel/main.go` — register any writer this consumer newly emits, per the standing "registration follows whoever EMITS" rule
- Tests beside them.

- [ ] **Step 1: Write the failing tests**

  (a) **Success:** a purchase event whose `TransactionId` resolves to a PENDING
      name-change record emits the done body built from the event's **real
      `AssetId`** — mirroring `consumer.go:135-142`, which resolves an
      `asset.Model` and reads `.CashId()`. Assert the emitted `CashId` is
      non-zero and equals the resolved asset's. **A test asserting `CashId != 0`
      is the regression gate for the whole defect** — name it so.
  (b) **Success, unrelated buy:** a purchase event with a zero or unknown
      `TransactionId` takes the pre-existing path unchanged. No name-change done
      body is emitted. This is the concurrency case the old code could not tell apart.
  (c) **Failure:** an error event whose `TransactionId` resolves to a PENDING
      record **releases that record via the existing self-scoped cancel path**
      (reason `player_cancelled`, `pending_change/processor.go:349`) and answers
      the client with the failure packet. Assert **no refund is minted** —
      `HasAsset()` is false on this path, and a spurious refund here is the
      failure mode insert-first exists to prevent.
  (d) `Expiration` on the emitted `CashInventoryItem` is **UNVERIFIED** — no
      client read path was found for it, which is not proof it is unused. Use
      whatever the resolved asset carries rather than a literal, and note it in
      the report. Do not fabricate a value.

- [ ] **Step 2: Implement, then delete the fabricated emit**

Remove the `CashId = 0` done-body construction Task 25 left in the handlers. Grep
the tree afterward and confirm no construction site remains that sets a `CashId`
from anything but a resolved `asset.Model`.

- [ ] **Step 3: Run the tests, then commit**

### Note on the unrelated pre-existing hole

`pending_change/processor.go:250-256` states the purchase path's entitlement is
consumed by atlas-cashshop off the `PENDING_CHANGE_CREATED` event. **That consumer
does not exist** — grep for `PENDING_CHANGE_CREATED` / `PendingChangeCreated`
across `services/atlas-cashshop/` returns nothing, and `TransactionId` is minted
inside atlas-character and never returned (the channel-side `pendingchange.RestModel`
has no such field). That is a *different* unbuilt design from the channel-driven
shape these three tasks build. It is **out of scope here** — but the stale comment
is not: whichever task last touches that file must correct it to describe what the
code actually does, or the next reader inherits the same false map that produced
this rework.

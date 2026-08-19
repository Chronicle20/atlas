# Duey Parcel Delivery Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build Duey — asynchronous player-to-player parcel delivery — as a new
`atlas-parcel` custody service, two packet families, two saga types, and a
world-transfer eligibility gate.

**Architecture:** A new `atlas-parcel` service owns the `parcels` table, the
state machine, and the expiry/return + notification sweeps. Item custody rides
the existing four-action saga custody protocol
(`release_from_character` / `accept_to_parcel` / `release_from_parcel` /
`accept_to_character`) so an item's stats survive the trip. `atlas-channel`
owns the wire: a `DUEY_ACTION` serverbound handler and a `PARCEL` clientbound
mode-prefix dispatcher. `atlas-character` gains gate 12 `parcel_pending`.

**Tech Stack:** Go, GORM + Postgres, Kafka (`atlas-kafka`), JSON:API
(`atlas-rest`), the Atlas saga orchestrator, `libs/atlas-packet`,
`tools/packet-audit`.

**Spec:** `docs/tasks/task-241-duey-parcel-delivery/design.md`
(PRD: `docs/tasks/task-241-duey-parcel-delivery/prd.md`)

## Global Constraints

Every task's requirements implicitly include this section. Values are copied
verbatim from the design.

- **Fee tiers** (design §6.3), applied in DESCENDING order, computed in
  `float64` and truncated by a `uint32` conversion — deliberately NOT integer
  arithmetic, because the client quotes the player a double-derived figure
  before the packet is sent (v72 `sub_6590A1` @0x6590A1, v83 `sub_6EEDFE`
  @0x6EEDFE):
  `m >= 100_000_000 → 0.06`, `m >= 25_000_000 → 0.05`,
  `m >= 10_000_000 → 0.04`, `m >= 5_000_000 → 0.03`,
  `m >= 1_000_000 → 0.018`, `m >= 100_000 → 0.008`, otherwise `0`.
- **Surcharge:** a non-quick (NPC) send adds a flat `5000`. A quick send adds
  nothing and consumes a Quick Delivery Ticket instead (design §0 finding B).
- **Per-parcel meso ceiling:** `100_000_000`. Above it → `INCORRECT_REQUEST`.
- **Level meso limit:** a character at level `<= 15` may send at most
  `1_000_000` meso; above → `MESO_LIMIT` (design §6.4).
- **Mailbox capacity:** a recipient may hold at most **10** pending parcels;
  an 11th send → `RECEIVER_STORAGE_FULL` (design §0 finding G).
- **Message length:** max 100 characters. The message exists ONLY on the
  quick-delivery serverbound arm (design §0 finding C).
- **Timers:** `ReceivableAt = CreatedAt + 24h` on a normal parcel,
  `ReceivableAt = CreatedAt` on a return leg; `ExpiresAt = CreatedAt + 30d`
  (subject to the RISK-4 polarity check in Task 23).
- **Never disconnect on a malformed request** (NFR-5). Reject with the
  matching `PARCEL` result arm and log at warn.
- **Quick Delivery Ticket item id:** `5330000`. **Duey NPC id:** `9010009`.
- **Version span:** `PARCEL` and `DUEY_ACTION` apply to gms v72, v79, v83,
  v84, v87, v92, v95 and jms v185. v48 and v61 are out of span (`⬜`).
- **Never hard-code a mode byte.** Every `PARCEL` arm resolves its mode via
  `atlas_packet.WithResolvedCode("operations", <fixed key const>, …)`.
- Test setup uses the project's Builder pattern. No `*_testhelpers.go` files.
- Implementers run module-local `go build ./... && go test ./...` only.
  Repo-wide verification is the controller's job.

---

## Task 1: atlas-parcel module skeleton, entity and migration

### Files

- `services/atlas-parcel/atlas.com/parcel/go.mod` — new file; module `atlas-parcel`
- `services/atlas-parcel/atlas.com/parcel/main.go` — new file
- `services/atlas-parcel/atlas.com/parcel/rest/handler.go` — new file
- `services/atlas-parcel/atlas.com/parcel/parcel/entity.go` — new file
- `services/atlas-parcel/atlas.com/parcel/parcel/asset_data.go` — new file
- `services/atlas-parcel/atlas.com/parcel/parcel/entity_test.go` — new file

Patterns to copy:
- `services/atlas-mts/atlas.com/mts/main.go` (bootstrap, `database.Connect`,
  `server.New`, `consumergroup.Resolve`) — read-only
- `services/atlas-mts/atlas.com/mts/rest/handler.go` — read-only
- `services/atlas-merchant/atlas.com/merchant/frederick/entity.go`
  (`gorm.Model` + `TenantId` + jsonb snapshot + `Migration`) — read-only
- `services/atlas-merchant/atlas.com/merchant/kafka/message/asset/kafka.go:10`
  (`AssetData` with `Value()`/`Scan()` driver plumbing) — read-only; copy the
  type into `parcel/asset_data.go` (a straightforward move, not a cross-service
  import — see CLAUDE.md "don't call another layer's internals")

Module root for `go build`/`go test`:
`services/atlas-parcel/atlas.com/parcel`.

**Interfaces produced:** `parcel.Entity` (fields per design §3),
`parcel.AssetData`, `parcel.Migration(db *gorm.DB) error`,
`parcel.StatusPending/StatusReceived/StatusDiscarded/StatusExpired` string
constants, `parcel.ReceivableDelay = 24 * time.Hour`,
`parcel.ExpiryWindow = 30 * 24 * time.Hour`.

### Steps

- [ ] **Step 1: Write the failing test**

`TestEntityTableName` and `TestMigrationCreatesParcels` in
`parcel/entity_test.go`. Setup copied from
`services/atlas-mts/atlas.com/mts/test/database.go` (sqlite in-memory GORM
handle) — if that helper is not importable from a fresh module, use the same
`gorm.Open(sqlite.Open(":memory:"))` shape it uses.

| case | assertion |
|---|---|
| `table name` | `(&Entity{}).TableName()` == `"parcels"` |
| `migration` | `Migration(db)` returns nil and `db.Migrator().HasTable("parcels")` is true |
| `status constants` | `StatusPending`=="pending", `StatusReceived`=="received", `StatusDiscarded`=="discarded", `StatusExpired`=="expired" |
| `timers` | `ReceivableDelay` == `24*time.Hour`, `ExpiryWindow` == `720*time.Hour` |

- [ ] **Step 2: Run the test and confirm it fails**

Run: `cd services/atlas-parcel/atlas.com/parcel && go test ./parcel/...`
Expected: compile failure — package `parcel` does not exist.

- [ ] **Step 3: Write `go.mod`, `main.go`, `rest/handler.go`**

`go.mod` declares `module atlas-parcel` with the same Go version and the same
`replace`-free shared-lib requirements `services/atlas-mts/atlas.com/mts/go.mod`
uses (the workspace resolves them). `main.go` is `atlas-mts`'s main.go with the
MTS-specific packages removed: `serviceName = "atlas-parcel"`,
`consumerGroupId = consumergroup.Resolve("Parcel Service")`,
`database.Connect(l, database.SetMigrations(parcel.Migration))`, the readiness
and `/debug/consumers` route initializers, and nothing else yet. Route
initializers, consumers and tasks are added by later tasks.

Add the module to `go.work` in this task so the workspace resolves it:
`./services/atlas-parcel/atlas.com/parcel`.

- [ ] **Step 4: Write `parcel/asset_data.go` and `parcel/entity.go`**

`asset_data.go` is the `AssetData` struct from
`services/atlas-merchant/atlas.com/merchant/kafka/message/asset/kafka.go`
verbatim (all 30 fields plus `WithQuantity`, `Value`, `Scan`), in package
`parcel`.

`entity.go` declares the entity exactly as design §3:

```go
type Entity struct {
	gorm.Model
	Id       uuid.UUID `gorm:"type:uuid;primaryKey"`
	TenantId uuid.UUID `gorm:"type:uuid;not null;index:idx_parcels_recipient,priority:1;index:idx_parcels_sender,priority:1;index:idx_parcels_sweep,priority:1"`
	WorldId  byte      `gorm:"not null"`

	SenderId        uint32 `gorm:"not null;index:idx_parcels_sender,priority:2"`
	SenderAccountId uint32 `gorm:"not null"`
	SenderName      string `gorm:"not null"`

	RecipientId        uint32 `gorm:"not null;index:idx_parcels_recipient,priority:2"`
	RecipientAccountId uint32 `gorm:"not null"`

	Message    string
	MesoAmount uint32
	FeePaid    uint32

	ItemId       *uint32
	ItemType     byte
	Quantity     uint16
	ItemSnapshot AssetData `gorm:"type:jsonb"`

	Status   string `gorm:"not null;index:idx_parcels_recipient,priority:3;index:idx_parcels_sender,priority:3;index:idx_parcels_sweep,priority:2"`
	Quick    bool   `gorm:"not null"`
	Returned bool   `gorm:"not null"`

	CreatedAt    time.Time `gorm:"not null"`
	ReceivableAt time.Time `gorm:"not null"`
	ExpiresAt    time.Time `gorm:"not null;index:idx_parcels_sweep,priority:3"`
	ResolvedAt   *time.Time
	LastNotified *time.Time
}
```

plus `TableName() string { return "parcels" }`, the four status constants, the
two duration constants, and
`func Migration(db *gorm.DB) error { return db.AutoMigrate(&Entity{}) }`.

`ItemId` is a pointer deliberately: a meso-only parcel has no item and `0` is
not a legal sentinel (design §3).

- [ ] **Step 5: Run the tests and confirm they pass**

Run: `cd services/atlas-parcel/atlas.com/parcel && go build ./... && go test ./...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-parcel go.work
git commit -m "feat(parcel): atlas-parcel module skeleton, parcels entity and migration"
```

---

## Task 2: atlas-parcel model, builder, provider, administrator

### Files

- `services/atlas-parcel/atlas.com/parcel/parcel/model.go` — new file
- `services/atlas-parcel/atlas.com/parcel/parcel/builder.go` — new file
- `services/atlas-parcel/atlas.com/parcel/parcel/provider.go` — new file
- `services/atlas-parcel/atlas.com/parcel/parcel/administrator.go` — new file
- `services/atlas-parcel/atlas.com/parcel/parcel/builder_test.go` — new file
- `services/atlas-parcel/atlas.com/parcel/parcel/provider_tenant_test.go` — new file

Patterns to copy:
- `services/atlas-merchant/atlas.com/merchant/frederick/model.go`,
  `builder.go`, `provider.go`, `administrator.go` — read-only, the closest
  shape (tenant-scoped custody rows with a jsonb snapshot)
- `services/atlas-merchant/atlas.com/merchant/frederick/builder_test.go` — read-only
- `services/atlas-mts/atlas.com/mts/listing/provider.go` for the world-scoped
  and status-scoped provider filters — read-only

Module root: `services/atlas-parcel/atlas.com/parcel`.

**Interfaces consumed:** Task 1's `Entity`, `AssetData`, status constants.

**Interfaces produced:**
- `parcel.Model` — immutable, with getters `Id() uuid.UUID`,
  `WorldId() world.Id`, `SenderId() uint32`, `SenderAccountId() uint32`,
  `SenderName() string`, `RecipientId() uint32`,
  `RecipientAccountId() uint32`, `Message() string`, `MesoAmount() uint32`,
  `FeePaid() uint32`, `ItemId() *uint32`, `ItemType() byte`,
  `Quantity() uint16`, `ItemSnapshot() AssetData`, `Status() string`,
  `Quick() bool`, `Returned() bool`, `CreatedAt() time.Time`,
  `ReceivableAt() time.Time`, `ExpiresAt() time.Time`,
  `ResolvedAt() *time.Time`, `LastNotified() *time.Time`
- `parcel.NewBuilder() *Builder` with a `SetX` per field and
  `Build() (Model, error)`
- `parcel.Make(e Entity) (Model, error)`
- Providers: `ById(id uuid.UUID)`, `ByRecipient(recipientId uint32, worldId world.Id, status string)`,
  `BySender(senderId uint32, status string)`,
  `ReceivableByRecipient(recipientId uint32, worldId world.Id, now time.Time)`
  — each `func(db *gorm.DB) model.Provider[...]` following frederick's shape
- Administrator: `Create(db *gorm.DB) func(m Model) (Model, error)`,
  `UpdateStatus(db *gorm.DB) func(id uuid.UUID, status string, resolvedAt time.Time) error`,
  `StampNotified(db *gorm.DB) func(ids []uuid.UUID, at time.Time) error`

### Steps

- [ ] **Step 1: Write the failing tests**

`TestBuilderRoundTrip` in `builder_test.go` — table-driven.

| case | input | expectation |
|---|---|---|
| `full parcel` | every field set, `ItemId` = pointer to 1302000, quantity 1 | every getter returns the value set |
| `meso only` | `ItemId` nil, `MesoAmount` 5000 | `ItemId()` is nil, `MesoAmount()` == 5000 |
| `return leg` | `Returned` true, `FeePaid` 0 | `Returned()` true, `FeePaid()` == 0 |
| `make from entity` | `Make(Entity{...})` | round-trips every field, including `ItemSnapshot` |

`TestProviderTenantIsolation` in `provider_tenant_test.go` — modelled on
`services/atlas-merchant/atlas.com/merchant/frederick`'s tenant test and
`services/atlas-storage/atlas.com/storage/storage/provider_tenant_test.go`.
Seed two parcels for the same `RecipientId` under two different tenant ids;
assert `ByRecipient` under tenant A returns exactly one row and it is A's.

| case | seeded | query | expect |
|---|---|---|---|
| `recipient scoped to tenant` | tenant A parcel + tenant B parcel, same recipientId 100 | `ByRecipient(100, world 0, "pending")` in tenant-A ctx | 1 model, `Id` == A's id |
| `sender scoped to tenant` | same | `BySender(200, "pending")` in tenant-B ctx | 1 model, `Id` == B's id |
| `receivable filter` | one parcel `ReceivableAt` = now+1h, one = now-1h, same recipient/tenant | `ReceivableByRecipient(100, 0, now)` | 1 model, the past one |

- [ ] **Step 2: Run the tests and confirm they fail**

Run: `cd services/atlas-parcel/atlas.com/parcel && go test ./parcel/...`
Expected: compile failure — `NewBuilder`, `Make`, `ByRecipient` undefined.

- [ ] **Step 3: Write model, builder, provider, administrator**

Follow frederick's file-for-file shape. The provider functions apply
`db.Where(...)` on the indexed columns only — `(tenant_id, recipient_id,
status)` for `ByRecipient`, `(tenant_id, sender_id, status)` for `BySender`,
so NFR-4 is served by `idx_parcels_recipient` with no table scan. Tenant
scoping comes from the `atlas-database` tenant filter on the context, exactly
as frederick's providers get it — the providers must NOT add a manual
`tenant_id` predicate.

`ReceivableByRecipient` adds `receivable_at <= ?` and `status = 'pending'`.

- [ ] **Step 4: Run the tests and confirm they pass**

Run: `cd services/atlas-parcel/atlas.com/parcel && go build ./... && go test ./...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-parcel
git commit -m "feat(parcel): parcel model, builder, providers and administrator"
```

---

## Task 3: atlas-parcel processor — state machine and injectable clock

### Files

- `services/atlas-parcel/atlas.com/parcel/parcel/processor.go` — new file
- `services/atlas-parcel/atlas.com/parcel/parcel/processor_test.go` — new file
- `services/atlas-parcel/atlas.com/parcel/parcel/errors.go` — new file
- `services/atlas-parcel/atlas.com/parcel/parcel/model.go` — created in Task 2;
  modify: add `Receivable(now time.Time) bool`
- `services/atlas-mts/atlas.com/mts/listing/errors.go` — read-only; the
  sentinel-error shape to copy

Patterns to copy:
- `services/atlas-merchant/atlas.com/merchant/frederick/processor.go` and
  `processor_test.go` — read-only
- `services/atlas-character/atlas.com/character/pending_change/processor_eligibility.go:73`
  (`withTransferEligibilityGates`) — read-only; the unexported with-style seam
  the clock override copies (design §8.3)

Module root: `services/atlas-parcel/atlas.com/parcel`.

**Interfaces consumed:** Task 2's `Model`, providers, administrator.

**Interfaces produced:**
- `parcel.Processor` interface with `GetById(id uuid.UUID) (Model, error)`,
  `GetForRecipient(recipientId uint32, worldId world.Id) ([]Model, error)`,
  `GetPendingForSender(senderId uint32) ([]Model, error)`,
  `HasInFlight(characterId uint32) (bool, error)`,
  `Create(m Model) (Model, error)`,
  `Receive(id uuid.UUID, recipientId uint32) (Model, error)`,
  `Discard(id uuid.UUID, recipientId uint32) (Model, error)`
- `parcel.NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor`
- unexported `withClock(now func() time.Time) Processor` on `*ProcessorImpl`
- sentinel errors in `errors.go`: `ErrNotFound`, `ErrNotPending`,
  `ErrNotRecipient`, `ErrNotYetReceivable`

### Steps

- [ ] **Step 1: Write the failing tests**

`TestProcessorReceive`, `TestProcessorDiscard`, `TestProcessorHasInFlight` in
`processor_test.go`. Setup uses `NewBuilder` from Task 2 and the in-memory DB
helper from Task 1's test; the clock is overridden via `withClock`.

`TestProcessorReceive` cases — fixed clock `T = 2026-01-02T00:00:00Z`:

| subtest | seeded parcel | call | expect |
|---|---|---|---|
| `happy path` | pending, recipient 100, `ReceivableAt` T-1h | `Receive(id, 100)` | model with `Status()`=="received", `ResolvedAt()` == T |
| `wrong recipient` | pending, recipient 100 | `Receive(id, 999)` | `ErrNotRecipient`, row still `pending` |
| `not yet receivable` | pending, recipient 100, `ReceivableAt` T+1h | `Receive(id, 100)` | `ErrNotYetReceivable`, row still `pending` |
| `already received` | `Status` `received` | `Receive(id, 100)` | `ErrNotPending` |
| `missing` | none | `Receive(uuid.New(), 100)` | `ErrNotFound` |

`TestProcessorDiscard` cases:

| subtest | seeded parcel | call | expect |
|---|---|---|---|
| `happy path` | pending, recipient 100, `ReceivableAt` T-1h | `Discard(id, 100)` | `Status()`=="discarded", `ResolvedAt()` == T |
| `wrong recipient` | pending, recipient 100 | `Discard(id, 999)` | `ErrNotRecipient` |
| `already discarded` | `Status` `discarded` | `Discard(id, 100)` | `ErrNotPending` |

`TestProcessorHasInFlight` cases — this is what gate 12 consumes (design §9.1:
"outbound `pending`, or inbound `pending` with `ReceivableAt <= now`"):

| subtest | seeded | call | expect |
|---|---|---|---|
| `outbound pending` | pending parcel, sender 100 | `HasInFlight(100)` | true |
| `inbound receivable` | pending parcel, recipient 100, `ReceivableAt` T-1h | `HasInFlight(100)` | true |
| `inbound not yet receivable` | pending parcel, recipient 100, `ReceivableAt` T+1h | `HasInFlight(100)` | false |
| `resolved only` | received parcel sender 100, discarded parcel recipient 100 | `HasInFlight(100)` | false |
| `no parcels` | none | `HasInFlight(100)` | false |

Note the asymmetry is intentional: an outbound parcel is in flight from the
instant it is sent, while an inbound one only becomes the character's problem
once it is receivable.

- [ ] **Step 2: Run the tests and confirm they fail**

Run: `cd services/atlas-parcel/atlas.com/parcel && go test ./parcel/... -run 'TestProcessor'`
Expected: compile failure — `NewProcessor` undefined.

- [ ] **Step 3: Write `errors.go` and `processor.go`**

`ProcessorImpl` carries `l`, `ctx`, `db`, and `now func() time.Time`.
`NewProcessor` sets `now: time.Now`. `withClock` returns a copy with `now`
replaced — unexported, mirroring `withTransferEligibilityGates`.

`Receive` and `Discard` each run one `database.ExecuteTransaction`: re-read the
row inside the transaction, validate recipient / status / receivability, then
`UpdateStatus`. Re-reading inside the transaction is what makes a replayed
receive award once (NFR-3) — the second delivery finds `status != pending` and
returns `ErrNotPending`.

Add `Model.Receivable(now time.Time) bool` (`Status()==StatusPending &&
!ReceivableAt().After(now)`) in `model.go`.

- [ ] **Step 4: Run the tests and confirm they pass**

Run: `cd services/atlas-parcel/atlas.com/parcel && go build ./... && go test ./...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-parcel
git commit -m "feat(parcel): parcel processor state machine with injectable clock"
```

---

## Task 4: atlas-parcel REST surface

### Files

- `services/atlas-parcel/atlas.com/parcel/parcel/rest.go` — new file
- `services/atlas-parcel/atlas.com/parcel/parcel/resource.go` — new file
- `services/atlas-parcel/atlas.com/parcel/parcel/resource_test.go` — new file
- `services/atlas-parcel/atlas.com/parcel/main.go` — created in Task 1; modify:
  add `AddRouteInitializer(parcel.InitResource(GetServer())(db))`

Patterns to copy:
- `services/atlas-mts/atlas.com/mts/holding/rest.go`, `resource.go`,
  `resource_test.go` — read-only, the JSON:API + filter-query shape
- `services/atlas-merchant/atlas.com/merchant/frederick/rest.go`,
  `resource.go` — read-only

Module root: `services/atlas-parcel/atlas.com/parcel`.

**Interfaces consumed:** Task 3's `Processor`.

**Interfaces produced:**
- `parcel.RestModel` with JSON:API tags for every model field
  (`Transform(m Model) (RestModel, error)`), `GetID()`/`SetID()`
- `parcel.InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer`
- routes:
  - `GET /parcels?filter[recipientId]=&filter[worldId]=&filter[status]=`
  - `GET /parcels?filter[senderId]=&filter[status]=`
  - `GET /parcels/{parcelId}`
  - `GET /characters/{characterId}/parcel-status`

### Steps

- [ ] **Step 1: Write the failing test**

`TestParcelResource` in `resource_test.go` — table-driven, HTTP-level, setup
copied from `services/atlas-mts/atlas.com/mts/holding/resource_test.go:1-80`
(router construction, tenant header injection, in-memory DB).

| subtest | seeded | request | expect |
|---|---|---|---|
| `list by recipient` | 2 pending parcels recipient 100 world 0, 1 recipient 200 | `GET /parcels?filter[recipientId]=100&filter[worldId]=0&filter[status]=pending` | 200, `data` length 2 |
| `list by sender` | 1 pending parcel sender 300 | `GET /parcels?filter[senderId]=300&filter[status]=pending` | 200, `data` length 1 |
| `get by id` | 1 parcel | `GET /parcels/{id}` | 200, `data.id` == the parcel id |
| `get by id missing` | none | `GET /parcels/{random uuid}` | 404 |
| `parcel-status true` | 1 pending parcel sender 100 | `GET /characters/100/parcel-status` | 200, `data.attributes.inFlight` == true |
| `parcel-status false` | none | `GET /characters/100/parcel-status` | 200, `data.attributes.inFlight` == false |
| `tenant isolation` | parcel under tenant A | `GET /parcels?filter[recipientId]=100` with tenant B header | 200, `data` length 0 |

- [ ] **Step 2: Run the test and confirm it fails**

Run: `cd services/atlas-parcel/atlas.com/parcel && go test ./parcel/... -run TestParcelResource`
Expected: compile failure — `InitResource` undefined.

- [ ] **Step 3: Write `rest.go` and `resource.go`**

`rest.go` mirrors `holding/rest.go`: a `RestModel` with `jsonapi` tags,
`GetName() string { return "parcels" }`, and `Transform`. The item snapshot is
exposed as a nested attribute object.

`resource.go` registers the four routes on the `server.RouteInitializer`.
`parcel-status` returns a distinct one-attribute rest model
(`parcelStatusRestModel{InFlight bool}`, `GetName() "parcel-statuses"`) so the
gate has one narrow round trip, matching how `mtsHoldingOpen` is a narrow
lookup (`services/atlas-character/atlas.com/character/pending_change/requests.go:184`).

- [ ] **Step 4: Wire the route initializer into `main.go`**

Add `AddRouteInitializer(parcel.InitResource(GetServer())(db))` to the
`server.New(l)` chain, before the `/debug/consumers` and `/readyz` mounts.

- [ ] **Step 5: Run the tests and confirm they pass**

Run: `cd services/atlas-parcel/atlas.com/parcel && go build ./... && go test ./...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-parcel
git commit -m "feat(parcel): JSON:API REST surface for parcels and parcel-status"
```

---

## Task 5: Register atlas-parcel across build, CI, k8s and databases

This task is deliberately larger than the six-file guideline — it is the
`docs/adding-a-new-service.md` checklist, which is one indivisible unit: a
partial registration is exactly the silent-failure mode that doc exists to
prevent (atlas-mts, task-121). See `context.md`.

### Files

- `.github/config/services.json` — add the `atlas-parcel` entry
- `docker-bake.hcl` — add `"atlas-parcel"` to `go_services`
- `deploy/k8s/base/atlas-parcel.yaml` — new file (Deployment + Service)
- `deploy/k8s/base/kustomization.yaml` — add `atlas-parcel.yaml` to `resources:`
- `deploy/k8s/base/env-configmap.yaml` — add `COMMAND_TOPIC_PARCEL_CUSTODY`,
  `EVENT_TOPIC_PARCEL_CUSTODY_STATUS`, `EVENT_TOPIC_PARCEL_STATUS`
- `deploy/k8s/overlays/main/patches/db-name-suffix.yaml` — add the parcel doc
- `deploy/k8s/overlays/main/patches/atlas-env-env.yaml` — add the parcel doc
- `deploy/k8s/overlays/main/kustomization.yaml` — `images:` entry +
  `configMapGenerator` literals for the three topics
- `deploy/k8s/overlays/pr/kustomization.yaml` — `ATLAS_DB_NAMES` +
  `images:` entry (topic literals come from the generator)
- `deploy/shared/routes.conf` — nginx location block for `atlas-parcel`
- `deploy/k8s/base/routes.conf.template.generated` — regenerated
- `tools/db-bootstrap.sh` — add `atlas-parcel` to the `DBS` list
- `go.work` — already added in Task 1; verify present

Read-only references: `deploy/k8s/base/atlas-mts.yaml` (copy as the template
for the Deployment/Service), `docs/adding-a-new-service.md` (the checklist).

Generator-owned files — **do not hand-edit**, re-run the script:
- `deploy/k8s/overlays/pr/patches/db-name-suffix.yaml` →
  `deploy/k8s/overlays/pr/scripts/gen-db-name-suffix.sh`
- `deploy/k8s/overlays/pr/patches/consumer-group-env.yaml` →
  `deploy/k8s/overlays/pr/scripts/gen-consumer-group-patch.sh`
- the PR overlay topic literals →
  `deploy/k8s/overlays/pr/scripts/gen-topic-config.sh`
- `dev/cluster-infra-coordination/atlas-pr-cleanup-env.example.yaml` →
  `deploy/k8s/overlays/pr/scripts/gen-cleanup-env.sh`

**Interfaces consumed:** Task 1's `main.go` (the generator reads
`consumerGroupId = consumergroup.Resolve("Parcel Service")` from it).

**Database name:** `atlas-parcel`. **Container name:** `parcel`.

### Steps

- [ ] **Step 1: Establish the failing check**

Run: `tools/service-registration-guard.sh`
Expected: FAIL, reporting `atlas-parcel` present in `go.work` but missing from
`services.json`, `docker-bake.hcl`, the base manifest and both overlays.

- [ ] **Step 2: Add the build and CI entries**

`.github/config/services.json`: an entry with
`name: "atlas-parcel"`, `type: "go-service"`,
`path: "services/atlas-parcel/atlas.com/parcel"`,
`module_path: "atlas-parcel"`,
`docker_image: "ghcr.io/chronicle20/atlas-parcel/atlas-parcel"`,
`docker_context: "."` — copy the atlas-mts entry's exact field set and shape.

`docker-bake.hcl`: add `"atlas-parcel"` to the hardcoded `go_services` list.

- [ ] **Step 3: Add the k8s base**

`deploy/k8s/base/atlas-parcel.yaml` copied from `atlas-mts.yaml` with:
container `name: parcel`, image
`ghcr.io/chronicle20/atlas-parcel/atlas-parcel`, `DB_NAME: "atlas-parcel"`
(UNSUFFIXED — the overlays patch the suffix), and the three topic env vars.
No `namespace:`.

Add `atlas-parcel.yaml` to `resources:` in `deploy/k8s/base/kustomization.yaml`
and the three `KEY: "KEY"` identity entries to
`deploy/k8s/base/env-configmap.yaml`.

- [ ] **Step 4: Add the main overlay**

New patch documents in `patches/db-name-suffix.yaml`
(`DB_NAME: "atlas-parcel-main"`, container `parcel`) and
`patches/atlas-env-env.yaml` (`ATLAS_ENV: "main"`). In
`overlays/main/kustomization.yaml`, add the `images:` entry
(`- name: ghcr.io/chronicle20/atlas-parcel/atlas-parcel` with `newTag:` set to
the current fleet `main-<sha>`, confirmed to exist on ghcr) and the three
`KEY=KEY-main` `configMapGenerator` literals. Do NOT add
`KAFKA_CONSUMER_GROUP` on main.

- [ ] **Step 5: Add the PR overlay and re-run its generators**

Hand-edit only `ATLAS_DB_NAMES` (add `atlas-parcel`) and `images:` in
`overlays/pr/kustomization.yaml`. Then run, in order:

```bash
deploy/k8s/overlays/pr/scripts/gen-topic-config.sh
deploy/k8s/overlays/pr/scripts/gen-db-name-suffix.sh
deploy/k8s/overlays/pr/scripts/gen-consumer-group-patch.sh
deploy/k8s/overlays/pr/scripts/gen-cleanup-env.sh
```

and commit whatever they rewrite. `gen-topic-config.sh` emits the whole
`KEY=KEY-PLACEHOLDER_ATLAS_ENV` block — paste its output into the atlas-env
generator wholesale rather than editing individual literals.

- [ ] **Step 6: Add ingress and the local DB bootstrap**

Add an alphabetically-placed location block for `http://atlas-parcel:8080` in
`deploy/shared/routes.conf`, then run `tools/gen-routes.sh` and commit both
files. Add `atlas-parcel` to the hand-edited `DBS` list in
`tools/db-bootstrap.sh`.

- [ ] **Step 7: Verify**

```bash
tools/service-registration-guard.sh
kubectl kustomize deploy/k8s/overlays/main | grep -A8 "name: atlas-parcel$"
kubectl kustomize deploy/k8s/overlays/pr > /dev/null
docker buildx bake atlas-parcel
```

Expected: guard exits 0; the main render shows `DB_NAME=atlas-parcel-main`,
`ATLAS_ENV=main` and a pinned `main-<sha>` image; the pr render is clean; the
bake succeeds.

Two steps no tool can do and which are **out of scope for the implementer** —
record them in the PR description as operator follow-ups, do not attempt them:
creating `atlas-parcel-main` on postgres.home (§6.1) and flipping the new GHCR
package to public after the first push (§6b).

- [ ] **Step 8: Commit**

```bash
git add .github/config/services.json docker-bake.hcl deploy tools/db-bootstrap.sh dev
git commit -m "chore(parcel): register atlas-parcel across build, k8s, ingress and databases"
```

---

## Task 6: Packet record — dispatcher yamls, registry entries, template tables

### Files

- `docs/packets/dispatchers/parcel.yaml` — new file
- `docs/packets/dispatchers/duey_action.yaml` — new file
- `docs/packets/registry/gms_v72.yaml` — add the `DUEY_ACTION` serverbound
  entry (opcode 64 / `0x040`, `provenance: ida-discovered`,
  `ida.address: 0x65AF41`)
- `docs/packets/registry/gms_v79.yaml` — add the `DUEY_ACTION` serverbound
  entry once derived (see Step 2)
- `services/atlas-configurations/seed-data/templates/template_gms_72_1.json`
  through `template_gms_95_1.json` and `template_jms_185_1.json` — regenerated
  `operations` tables (generator-owned; do not hand-edit)

Read-only references:
- `docs/packets/dispatchers/storage_operation.yaml` — the per-version mode
  table with a documented column divergence, the shape to copy
- `docs/packets/dispatchers/note_operation.yaml` — the minimal shape, including
  the `opcodes:` block that lets `operations` ADD a missing writer
- `docs/packets/dispatchers/README.md` — the `alias_of` and regeneration rules
- `docs/packets/registry/gms_v83.yaml:2228` — the existing `DUEY_ACTION`
  entry with its `fname_alts` list
- `docs/packets/DISPATCHER_FAMILY.md`

**Interfaces produced:** the `operations` keys every later packet task
resolves against — `OPEN`, `SEND_ENABLE_ACTIONS`, `NOT_ENOUGH_MESOS`,
`INCORRECT_REQUEST`, `NAME_DOES_NOT_EXIST`, `SAME_ACCOUNT`,
`RECEIVER_STORAGE_FULL`, `RECEIVER_UNABLE_TO_RECEIVE`,
`SENDER_UNIQUE_CONFLICT`, `MESO_LIMIT`, `SUCCESSFULLY_SENT`, `UNKNOWN_ERROR`,
`RECV_ENABLE_ACTIONS`, `RECV_NO_FREE_SLOTS`, `RECV_UNIQUE_CONFLICT`,
`PARCEL_REMOVED`, `PARCEL_ARRIVED`, `ALARM_NAMED`, `OPEN_QUICK`,
`ALARM_GENERIC`, `UNKNOWN_ERROR_2`; and the serverbound routing keys `SEND`,
`RECEIVE`, `DISCARD`, `CLOSE`.

### Steps

- [ ] **Step 1: Write `docs/packets/dispatchers/parcel.yaml`**

Header block: `writer: Parcel`, `fname: CParcelDlg::OnPacket`, `op: PARCEL`,
`direction: clientbound`, and an `opcodes:` map from the registry:
`gms_v72: "0x120"`, `gms_v79: "0x12C"`, `gms_v83: "0x142"`,
`gms_v84: "0x149"`, `gms_v87: "0x153"`, `gms_v92: "0x175"`,
`gms_v95: "0x17D"`, `jms_v185: "0x160"`.

The `operations:` list is the design §5.2 arm set, all twenty-one keys with the
v83-derived mode values 0x08–0x1C (decimal 8–28). Seed **only the `gms_v83`
column** from the design, then derive each remaining column in Step 3 — the
design is explicit that every mode above is v83-derived and each other version
re-derives its own arm set.

A leading comment records: the source addresses
(`CParcelDlg::OnPacket` @0x6F56EA, `CParcelDlg::NoticeResult` @0x6F5BE2 on
GMS v83), the fact that arms 0x00–0x07 do not exist because `OnPacket`'s
`default` runs `NoticeResult` which returns without a notice below 0x0A, and
the correction against Cosmic's enum (0x17 has a body; 0x18/0x19/0x1B are
notification arms; 0x1A and 0x1C exist at all).

- [ ] **Step 2: Write `docs/packets/dispatchers/duey_action.yaml` and close the v72/v79 registry gap**

`writer: DueyAction`, `fname: CTabSend::SendParcel`, `op: DUEY_ACTION`,
**`direction: serverbound`** — the explicit `serverbound` marker is what opts
the file out of the FAM-CAP check; omitting it would demote already-verified
cells (`DISPATCHER_FAMILY.md`, "Serverbound dispatcher files are out of scope
for FAM-CAP").

`operations:` — `SEND: 2`, `RECEIVE: 4`, `DISCARD: 5`, `CLOSE: 7` on gms_v83.

Add the missing v72 registry entry to `docs/packets/registry/gms_v72.yaml`, in
opcode order, in the same shape as the v83 entry:

```yaml
- op: DUEY_ACTION
  direction: serverbound
  opcode: 64
  fname: CTabReceive::ReceiveParcel
  fname_alts:
    - CTabSend::SendParcel
  provenance: ida-discovered
  ida:
    address: 6663489
```

(`0x65AF41` = 6663489; `CTabSend::SendParcel` @0x65D940 constructs the same
opcode with mode 2 — design §5.4.) Derive the v79 opcode the same way from the
v79 IDB before adding its entry; if the v79 build genuinely lacks the op,
record that in the yaml comment and leave v79 out of span rather than guessing.

- [ ] **Step 3: Derive every remaining version column**

For each of gms_v72, v79, v84, v87, v92 and jms_v185, decompile that build's
`CParcelDlg::OnPacket` and fill its column in `parcel.yaml`; do the same for
each build's `CTabSend::SendParcel` / `CTabReceive::ReceiveParcel` /
`CTabReceive::DiscardParcel` / `CParcelDlg::CloseParcelDlg` to fill
`duey_action.yaml`. JMS is expected to shift (`storage_operation.yaml`'s jms
column is the precedent for recording a shift rather than assuming GMS
values). A column whose client has no case for a key omits that key — never
invent a byte for a version-absent arm.

- [ ] **Step 4: Regenerate the template operations tables**

```bash
go run ./tools/packet-audit operations
go run ./tools/packet-audit operations --check
go run ./tools/packet-audit fname-doc --check
```

Expected: `operations` rewrites the `Parcel` / `DueyAction` tables into each
in-span template and adds the writer at the `opcodes:` hex where it is absent;
`--check` then exits 0 with no `EXTRA` or `MISSING` rows.

- [ ] **Step 5: Commit**

```bash
git add docs/packets services/atlas-configurations/seed-data/templates
git commit -m "docs(packets): parcel and duey_action dispatcher tables, v72 DUEY_ACTION registry entry"
```

---

## Task 7: `PARCEL` struct and the OPEN / OPEN_QUICK arms

### Files

- `libs/atlas-packet/parcel/parcel.go` — new file; the shared `Parcel` wire
  struct (the 234-byte fixed block + optional item)
- `libs/atlas-packet/parcel/clientbound/parcel.go` — new file; `Open` and
  `OpenQuick` discrete structs
- `libs/atlas-packet/parcel/clientbound/parcel_body.go` — new file; the
  operation key constants and `ParcelOpenBody` / `ParcelOpenQuickBody`
- `libs/atlas-packet/parcel/clientbound/parcel_test.go` — new file
- `tools/packet-audit/cmd/run.go` — modify `candidatesFromFName`: add the
  `CParcelDlg::OnPacket#Open` and `#OpenQuick` cases

Patterns to copy:
- `libs/atlas-packet/note/clientbound/operation.go` (discrete struct shape,
  `packet-audit:fname` marker, `Encode`/`Decode` pair) — read-only
- `libs/atlas-packet/note/clientbound/operation_body.go`
  (`atlas_packet.WithResolvedCode("operations", KEY, func(mode byte)…)`) — read-only
- `libs/atlas-packet/storage/clientbound/show.go` — read-only; a clientbound
  arm carrying a list of items
- `tools/packet-audit/cmd/run.go:1324-1340` — the `#`-entry case shape

Module root: `libs/atlas-packet`.

**Interfaces produced:**
- `parcel.Parcel` struct with a `NewParcel(...)` constructor and
  `Encode(l, ctx) func(map[string]interface{}) []byte`
- `clientbound.ParcelWriter = "Parcel"` (the writer name)
- `clientbound.NewParcelOpen(mode byte, quickEnabled bool, mailbox []parcel.Parcel, arrived []parcel.Parcel) Open`
- `clientbound.NewParcelOpenQuick(mode byte) OpenQuick`
- `clientbound.ParcelOpenBody(quickEnabled bool, mailbox []parcel.Parcel, arrived []parcel.Parcel) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte`
- `clientbound.ParcelOpenQuickBody() func(...)` — same return shape
- operation key constants `ParcelOperationOpen = "OPEN"`,
  `ParcelOperationOpenQuick = "OPEN_QUICK"`

### Steps

- [ ] **Step 1: Derive the fixed block's exact layout**

The design pins five offsets and defers the rest. Before writing any code,
decompile `PARCEL::Decode` @0x4E4345 (GMS v83) and
`CTabReceive::SetParcel` @0x6EF69C and write down the complete layout of the
234-byte block. Confirmed anchors, each cited to its consumer:

| offset | field | evidence |
|---|---|---|
| +0 | `uint32 parcelId` | receive/discard encode `*(parcel+0)` as the id (v83 @0x6F0CA3, @0x6F0DC3) |
| +4 | `char[13] senderName` | `SetParcel` formats SP_3878 with `parcel+4` |
| +17 | `uint32 mesos` | SP_3879 formatted with `*(parcel+17)` |
| +21 | `FILETIME sentAt` | v72 `ReceiveParcel` computes its 30-day window from `*(uint64*)(parcel+21)` |
| +29..233 | message + padding | the remainder of the fixed block |
| +234 | `bool hasItem` then `GW_ItemSlotBase` | design §5.3 |

Record the derived +29..233 breakdown as a comment block in `parcel.go`. Do
not guess: if a sub-field's meaning cannot be established, encode it as
explicit zero padding of the exact derived width and say so in the comment.

- [ ] **Step 2: Write the failing test**

`TestParcelEncode` and `TestParcelOpenEncode` in `parcel_test.go` —
byte-fixture tests carrying `// packet-audit:verify` markers, setup copied
from `libs/atlas-packet/note/clientbound/operation_test.go` (writer options
map with an `operations` table, `Encode(l, ctx)(options)` and a `[]byte`
comparison).

| subtest | input | expected bytes |
|---|---|---|
| `parcel no item` | id 7, sender "Alice", mesos 1000, sentAt a fixed FILETIME, message "hi", no item | `04 bytes id LE` + 13-byte name field + `04 bytes meso LE` + 8-byte FILETIME + the derived message/padding block + `00` |
| `parcel with item` | same plus a 1302000 equip | the same prefix + `01` + the `GW_ItemSlotBase` encoding the existing encoder produces |
| `open empty mailbox` | mode 8, quickEnabled true, mailbox empty, arrived empty | `08 01 00 00` |
| `open one parcel` | mode 8, quickEnabled false, one mailbox parcel, no arrived | `08 00 01` + that parcel's bytes + `00` |
| `open with arrived` | mode 8, quickEnabled false, one mailbox parcel, the same parcel also in arrived | `08 00 01` + parcel bytes + `01` + parcel bytes |
| `open quick` | mode 26 (0x1A) | `1A` |

The exact byte strings for the two `parcel` cases come from the Step-1
derivation; write them out in full in the test rather than computing them.
The `open` cases' expected bytes are `mode` + `bool` + `byte count` + N×parcel
+ `byte newCount` + M×parcel, per design §5.3.

- [ ] **Step 3: Run the test and confirm it fails**

Run: `cd libs/atlas-packet && go test ./parcel/...`
Expected: compile failure — package `parcel` does not exist.

- [ ] **Step 4: Write `parcel.go`, `parcel.go` (clientbound) and `parcel_body.go`**

`Open` holds `mode byte`, `quickEnabled bool`, `mailbox []parcel.Parcel`,
`arrived []parcel.Parcel`; its `Encode` writes mode, the bool, the mailbox
count byte, each parcel, the arrived count byte, each arrived parcel.
`OpenQuick` holds only `mode byte`. Both carry a
`// packet-audit:fname CParcelDlg::OnPacket#Open` / `#OpenQuick` marker and
both implement `Operation() string { return ParcelWriter }`.

The body functions use `WithResolvedCode` with a **fixed const key** — no
caller-supplied selector, per INV-3.

- [ ] **Step 5: Add the `run.go` candidate entries**

In `candidatesFromFName`, next to the note family's block, add:

```go
case "CParcelDlg::OnPacket#Open":
	return []candidate{{name: "Open", pkg: "parcel", dir: csvpkg.DirClientbound}}
case "CParcelDlg::OnPacket#OpenQuick":
	return []candidate{{name: "OpenQuick", pkg: "parcel", dir: csvpkg.DirClientbound}}
```

- [ ] **Step 6: Run the tests and the linter**

```bash
cd libs/atlas-packet && go build ./... && go test ./parcel/...
cd - && go run ./tools/packet-audit dispatcher-lint
```

Expected: tests PASS; `dispatcher-lint: clean`.

- [ ] **Step 7: Commit**

```bash
git add libs/atlas-packet/parcel tools/packet-audit/cmd/run.go
git commit -m "feat(packet): PARCEL wire struct and the OPEN/OPEN_QUICK arms"
```

---

## Task 8: `PARCEL` bodyless result arms

### Files

- `libs/atlas-packet/parcel/clientbound/parcel.go` — created in Task 7; modify:
  add the fifteen bodyless result structs
- `libs/atlas-packet/parcel/clientbound/parcel_body.go` — created in Task 7;
  modify: add their key constants and body functions
- `libs/atlas-packet/parcel/clientbound/parcel_result_test.go` — new file
- `tools/packet-audit/cmd/run.go` — modify: fifteen more `#`-entry cases

Patterns to copy: `libs/atlas-packet/note/clientbound/operation.go`'s
`SendSuccess` / `Refresh` (a `struct { mode byte }` whose `Encode` writes one
byte) — read-only.

Module root: `libs/atlas-packet`.

**Interfaces consumed:** Task 7's `ParcelWriter`, the resolved-mode idiom.

**Interfaces produced:** one struct + one constructor + one body func + one
operation-key const per arm, for each of:

| key const | struct | mode (v83) |
|---|---|---|
| `ParcelOperationSendEnableActions` = `"SEND_ENABLE_ACTIONS"` | `SendEnableActions` | 0x09 |
| `ParcelOperationNotEnoughMesos` = `"NOT_ENOUGH_MESOS"` | `NotEnoughMesos` | 0x0A |
| `ParcelOperationIncorrectRequest` = `"INCORRECT_REQUEST"` | `IncorrectRequest` | 0x0B |
| `ParcelOperationNameDoesNotExist` = `"NAME_DOES_NOT_EXIST"` | `NameDoesNotExist` | 0x0C |
| `ParcelOperationSameAccount` = `"SAME_ACCOUNT"` | `SameAccount` | 0x0D |
| `ParcelOperationReceiverStorageFull` = `"RECEIVER_STORAGE_FULL"` | `ReceiverStorageFull` | 0x0E |
| `ParcelOperationReceiverUnableToReceive` = `"RECEIVER_UNABLE_TO_RECEIVE"` | `ReceiverUnableToReceive` | 0x0F |
| `ParcelOperationSenderUniqueConflict` = `"SENDER_UNIQUE_CONFLICT"` | `SenderUniqueConflict` | 0x10 |
| `ParcelOperationMesoLimit` = `"MESO_LIMIT"` | `MesoLimit` | 0x11 |
| `ParcelOperationSuccessfullySent` = `"SUCCESSFULLY_SENT"` | `SuccessfullySent` | 0x12 |
| `ParcelOperationUnknownError` = `"UNKNOWN_ERROR"` | `UnknownError` | 0x13 |
| `ParcelOperationRecvEnableActions` = `"RECV_ENABLE_ACTIONS"` | `RecvEnableActions` | 0x14 |
| `ParcelOperationRecvNoFreeSlots` = `"RECV_NO_FREE_SLOTS"` | `RecvNoFreeSlots` | 0x15 |
| `ParcelOperationRecvUniqueConflict` = `"RECV_UNIQUE_CONFLICT"` | `RecvUniqueConflict` | 0x16 |
| `ParcelOperationUnknownError2` = `"UNKNOWN_ERROR_2"` | `UnknownError2` | 0x1C |

Body function names are `Parcel<StructName>Body()`, each taking **no
arguments** — a caller must not be able to pick which result the client sees
(AP-4/INV-3). Each is its own discrete struct even though all fifteen share a
wire shape: "discrete means *discrete*, even when two arms share a wire shape"
(`DISPATCHER_FAMILY.md`).

### Steps

- [ ] **Step 1: Write the failing test**

`TestParcelResultArms` in `parcel_result_test.go` — one table over all fifteen
arms, carrying a `// packet-audit:verify` marker. Setup copied from Task 7's
`parcel_test.go`.

Each case supplies an `operations` table mapping the arm's key to its v83
mode, calls the arm's body function, and asserts the output is exactly the
single byte of that mode. Case names are the key strings; expected bytes are
the mode column above (`{0x09}`, `{0x0A}`, … `{0x1C}`).

Add one negative case, `unresolved key falls back`, asserting that when the
`operations` table is missing the key the encoder still produces one byte and
does not panic — matching `ResolveCode`'s documented miss behaviour.

- [ ] **Step 2: Run the test and confirm it fails**

Run: `cd libs/atlas-packet && go test ./parcel/... -run TestParcelResultArms`
Expected: compile failure — `ParcelNotEnoughMesosBody` undefined.

- [ ] **Step 3: Write the structs and body functions**

Fifteen `struct { mode byte }` types in the single consolidated
`clientbound/parcel.go` (no file sprawl — AP-8), each with a
`// packet-audit:fname CParcelDlg::OnPacket#<StructName>` marker,
`Mode()`, `Operation()`, `String()`, `Encode`, `Decode`. Fifteen body
functions in `parcel_body.go`, each
`WithResolvedCode("operations", <fixed const>, func(mode byte) packet.Encoder {…})`.

- [ ] **Step 4: Add the fifteen `run.go` cases**

One `case "CParcelDlg::OnPacket#<StructName>":` per arm, each returning
`[]candidate{{name: "<StructName>", pkg: "parcel", dir: csvpkg.DirClientbound}}`,
with a one-line comment naming the mode and the client's StringPool effect from
design §5.2.

- [ ] **Step 5: Run the tests and the linter**

```bash
cd libs/atlas-packet && go build ./... && go test ./parcel/...
cd - && go run ./tools/packet-audit dispatcher-lint
```

Expected: PASS; `dispatcher-lint: clean` (INV-1 no shared struct, INV-2 no
`mode: 0x` literal, INV-3 no caller-supplied key, INV-5 every struct
constructed by a body func).

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-packet/parcel tools/packet-audit/cmd/run.go
git commit -m "feat(packet): PARCEL bodyless result arms"
```

---

## Task 9: `PARCEL` arms that carry a body — removed, arrived, alarms

### Files

- `libs/atlas-packet/parcel/clientbound/parcel.go` — created in Task 7; modify:
  add four structs
- `libs/atlas-packet/parcel/clientbound/parcel_body.go` — created in Task 7;
  modify: add their keys and body functions
- `libs/atlas-packet/parcel/clientbound/parcel_notify_test.go` — new file
- `tools/packet-audit/cmd/run.go` — modify: four more `#`-entry cases

Patterns to copy: `libs/atlas-packet/note/clientbound/operation.go`'s
`SendError` (a struct whose `Encode` writes mode plus a body field) — read-only.

Module root: `libs/atlas-packet`.

**Interfaces consumed:** Task 7's `parcel.Parcel` and `ParcelWriter`.

**Interfaces produced:**

| key const | struct | mode | body |
|---|---|---|---|
| `ParcelOperationParcelRemoved` = `"PARCEL_REMOVED"` | `ParcelRemoved` | 0x17 | `uint32 parcelId`, `byte kind` |
| `ParcelOperationParcelArrived` = `"PARCEL_ARRIVED"` | `ParcelArrived` | 0x18 | one `parcel.Parcel` |
| `ParcelOperationAlarmNamed` = `"ALARM_NAMED"` | `AlarmNamed` | 0x19 | `string senderName`, `bool hasItem` |
| `ParcelOperationAlarmGeneric` = `"ALARM_GENERIC"` | `AlarmGeneric` | 0x1B | `bool hasItem` |

Body functions:
- `ParcelRemovedBody(parcelId uint32, kind byte)`
- `ParcelArrivedBody(p parcel.Parcel)`
- `ParcelAlarmNamedBody(senderName string, hasItem bool)`
- `ParcelAlarmGenericBody(hasItem bool)`

`kind` is a client-interpreted discriminator, not an operations-table code: the
client shows SP_3899 "deleted" when `kind == 3` and SP_3900 "claimed"
otherwise (design §5.2). It stays a plain body byte and is NOT resolved
through the operations table — INV-3's semantic signal only fires on a
parameter that flows into a `WithResolvedCode` key, which this one does not.
Export the two callers need as named constants so no call site writes a bare
literal: `ParcelRemovedKindDiscarded = byte(3)` (the design pins 3 as the
discard value) and `ParcelRemovedKindClaimed`, **whose value must be read off
`CParcelDlg::OnPacket`'s 0x17 arm before it is written down** — the design
establishes only that the client shows "claimed" for anything other than 3,
not which value the server sends. Derive it, then use it in the fixture below.

### Steps

- [ ] **Step 1: Write the failing test**

`TestParcelNotifyArms` in `parcel_notify_test.go`, with a
`// packet-audit:verify` marker.

| subtest | call | expected bytes |
|---|---|---|
| `removed claimed` | `ParcelRemovedBody(7, ParcelRemovedKindClaimed)`, operations `PARCEL_REMOVED`=0x17 | `17 07 00 00 00` + the derived `ParcelRemovedKindClaimed` byte |
| `removed discarded` | `ParcelRemovedBody(7, ParcelRemovedKindDiscarded)` | `17 07 00 00 00 03` |
| `arrived` | `ParcelArrivedBody(p)` with the Task-7 `parcel no item` fixture, operations `PARCEL_ARRIVED`=0x18 | `18` + that fixture's exact bytes |
| `alarm named with item` | `ParcelAlarmNamedBody("Alice", true)`, operations `ALARM_NAMED`=0x19 | `19` + the length-prefixed `"Alice"` the response writer emits + `01` |
| `alarm named no item` | `ParcelAlarmNamedBody("Alice", false)` | same with a trailing `00` |
| `alarm generic` | `ParcelAlarmGenericBody(true)`, operations `ALARM_GENERIC`=0x1B | `1B 01` |

The string encoding is whatever `response.Writer`'s string method already
emits — read it from `libs/atlas-socket/response` and write the exact bytes
into the fixture rather than describing them.

- [ ] **Step 2: Run the test and confirm it fails**

Run: `cd libs/atlas-packet && go test ./parcel/... -run TestParcelNotifyArms`
Expected: compile failure — `ParcelRemovedBody` undefined.

- [ ] **Step 3: Write the four structs and body functions**

Each with its `packet-audit:fname` marker and a `WithResolvedCode` body func
keyed on its fixed const.

- [ ] **Step 4: Add the four `run.go` cases**

- [ ] **Step 5: Run the tests and the linter**

```bash
cd libs/atlas-packet && go build ./... && go test ./parcel/...
cd - && go run ./tools/packet-audit dispatcher-lint
```

Expected: PASS; `dispatcher-lint: clean`.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-packet/parcel tools/packet-audit/cmd/run.go
git commit -m "feat(packet): PARCEL removed/arrived/alarm arms"
```

---

## Task 10: `DUEY_ACTION` serverbound codecs

### Files

- `libs/atlas-packet/parcel/serverbound/action.go` — new file; the mode-byte
  dispatcher struct
- `libs/atlas-packet/parcel/serverbound/action_send.go` — new file
- `libs/atlas-packet/parcel/serverbound/action_parcel_id.go` — new file
  (receive and discard share the `uint32 parcelId` body but are separate
  structs)
- `libs/atlas-packet/parcel/serverbound/action_test.go` — new file
- `tools/packet-audit/cmd/run.go` — modify: the `CTabSend::SendParcel`,
  `CTabReceive::ReceiveParcel`, `CTabReceive::DiscardParcel` and
  `CParcelDlg::CloseParcelDlg` cases

Patterns to copy:
- `libs/atlas-packet/storage/serverbound/operation.go` (the `Operation`
  mode-byte dispatcher plus its `Handle` const) — read-only
- `libs/atlas-packet/storage/serverbound/operation_store_asset.go` and its
  test — read-only

Module root: `libs/atlas-packet`.

**Interfaces produced:**
- `serverbound.DueyActionHandle = "DueyActionHandle"`
- `serverbound.Action` with `Mode() byte`, `Operation()`, `Decode`
- `serverbound.ActionSend` with `InventoryType() byte`, `Slot() uint16`,
  `Quantity() uint16`, `Mesos() uint32`, `RecipientName() string`,
  `Quick() bool`, `Message() string`, `TicketRef() uint32`
- `serverbound.ActionReceive` and `serverbound.ActionDiscard`, each with
  `ParcelId() uint32`
- `serverbound.ActionClose` (no body)

The `SEND` body is asymmetric (design §5.4, §0 finding C): the NPC arm
(`quick == false`) **stops after the flag** — no message, no ticket ref
(v83 @0x6F36A8). Only the quick arm (@0x6F1DF5) carries
`string message` then `uint32 ticketRef`. `ActionSend.Decode` must read the
two trailing fields **only when the quick flag is set**; reading them
unconditionally desynchronises the reader on every NPC send.

### Steps

- [ ] **Step 1: Write the failing test**

`TestDueyActionDecode` in `action_test.go`, table-driven, setup copied from
`libs/atlas-packet/storage/serverbound/operation_store_asset_test.go`.

| subtest | input bytes | expect |
|---|---|---|
| `mode send` | `02` + the send body below | `Action.Mode()` == 2 |
| `send npc` | `01` (invType) `05 00` (slot) `01 00` (qty) `E8 03 00 00` (mesos 1000) + `"Bob"` + `00` (quick) | invType 1, slot 5, qty 1, mesos 1000, name "Bob", quick false, message "", ticketRef 0 |
| `send quick` | same prefix with `01` (quick) + `"hi"` + `40 42 0F 00` (ticketRef 1000000) | quick true, message "hi", ticketRef 1000000 |
| `send npc trailing garbage` | the `send npc` bytes followed by four extra bytes | the extra bytes are NOT consumed — assert the reader's remaining length is 4 after `Decode` |
| `receive` | `07 00 00 00` | `ActionReceive.ParcelId()` == 7 |
| `discard` | `07 00 00 00` | `ActionDiscard.ParcelId()` == 7 |
| `close` | empty | `Decode` consumes nothing and does not panic |

The `"Bob"` / `"hi"` byte sequences are whatever `request.Reader`'s string
method expects — read `libs/atlas-socket/request` and write the exact bytes.

- [ ] **Step 2: Run the test and confirm it fails**

Run: `cd libs/atlas-packet && go test ./parcel/serverbound/...`
Expected: compile failure — package does not exist.

- [ ] **Step 3: Write the four structs**

`Action` mirrors `storage/serverbound.Operation` exactly: one `mode byte`,
`Operation() string { return DueyActionHandle }`, `Decode` reads one byte.

Each struct carries a `packet-audit:fname` marker naming its client call site:
`CTabSend::SendParcel`, `CTabReceive::ReceiveParcel`,
`CTabReceive::DiscardParcel`, `CParcelDlg::CloseParcelDlg`.

- [ ] **Step 4: Add the `run.go` cases**

Four cases returning
`[]candidate{{name: "<Struct>", pkg: "parcel", dir: csvpkg.DirServerbound}}`.

- [ ] **Step 5: Run the tests**

```bash
cd libs/atlas-packet && go build ./... && go test ./parcel/...
cd - && go run ./tools/packet-audit dispatcher-lint
```

Expected: PASS; clean.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-packet/parcel tools/packet-audit/cmd/run.go
git commit -m "feat(packet): DUEY_ACTION serverbound codecs"
```

---

## Task 11: Saga library — parcel custody actions, types and payloads

### Files

- `libs/atlas-saga/model.go` — modify: add the two `Type` constants and the
  four `Action` constants
- `libs/atlas-saga/payloads.go` — modify: add the four payload structs
- `libs/atlas-saga/unmarshal.go` — modify: add the four `case` arms
- `libs/atlas-saga/unmarshal_test.go` — modify: add the four unmarshal tests
- `libs/atlas-saga/world_transfer_test.go` — modify: add the four actions to
  the known-action list at line 97-118

Patterns to copy:
- `libs/atlas-saga/model.go` — read-only; lines 213-232, the MTS custody action block, including
  its comment explaining composite vs atomic) — read-only
- `libs/atlas-saga/payloads.go:844-874` (`TransferToMtsPayload`,
  `WithdrawFromMtsPayload`, `AcceptToMtsListingPayload`,
  `ReleaseFromMtsHoldingPayload`) — read-only
- `libs/atlas-saga/unmarshal.go` — read-only; lines 432-455
- `libs/atlas-saga/unmarshal_test.go` — read-only; lines 373-620

Module root: `libs/atlas-saga`.

**Interfaces produced:**
- `saga.ParcelSend Type = "parcel_send"`, `saga.ParcelReceive Type = "parcel_receive"`
- `saga.TransferToParcel Action = "transfer_to_parcel"` (COMPOSITE)
- `saga.AcceptToParcel Action = "accept_to_parcel"` (atomic)
- `saga.ReleaseFromParcel Action = "release_from_parcel"` (atomic)
- `saga.WithdrawFromParcel Action = "withdraw_from_parcel"` (COMPOSITE)
- `saga.TransferToParcelPayload` — `TransactionId uuid.UUID`,
  `ParcelId uuid.UUID`, `CharacterId uint32`, `WorldId world.Id`,
  `SourceInventoryType byte`, `AssetId uint32`, `Quantity uint32`,
  `SenderAccountId uint32`, `SenderName string`, `RecipientId uint32`,
  `RecipientAccountId uint32`, `MesoAmount uint32`, `FeePaid uint32`,
  `Quick bool`, `Message string`, `ReceivableAt time.Time`,
  `ExpiresAt time.Time`
- `saga.AcceptToParcelPayload` — the same delivery parameters plus the
  captured item snapshot fields (the same equip stat block
  `AcceptToMtsListingPayload` carries: `TemplateId`, `Quantity`, `Strength`,
  `Dexterity`, `Intelligence`, `Luck`, `HP`, `MP`, `WeaponAttack`,
  `MagicAttack`, `WeaponDefense`, `MagicDefense`, `Accuracy`, `Avoidability`,
  `Hands`, `Speed`, `Jump`, `Slots`, `Level`, `ItemLevel`, `ItemExp`,
  `RingId`, `ViciousCount`, `Flags`, `Owner`) and `HasItem bool`
- `saga.ReleaseFromParcelPayload` — `TransactionId uuid.UUID`,
  `ParcelId uuid.UUID`, `RecipientId uint32`
- `saga.WithdrawFromParcelPayload` — `TransactionId uuid.UUID`,
  `ParcelId uuid.UUID`, `CharacterId uint32`, `WorldId world.Id`,
  `InventoryType byte`

`AcceptToParcelPayload.HasItem` is the meso-only escape hatch (RISK-2): the
composite expansion sets it false and leaves the snapshot zero-valued, and
atlas-parcel then creates a row with a nil `ItemId`.

### Steps

- [ ] **Step 1: Write the failing tests**

Four tests in `unmarshal_test.go`, each modelled on
`TestUnmarshalTransferToMtsStep` (line 373): build a JSON step envelope with
the action string, unmarshal it, assert `step.Action` and that
`step.Payload.(<Payload>)` type-asserts and carries the values set.

| test | action string | payload type | asserted fields |
|---|---|---|---|
| `TestUnmarshalTransferToParcelStep` | `transfer_to_parcel` | `TransferToParcelPayload` | `ParcelId`, `CharacterId`, `AssetId`, `MesoAmount`, `FeePaid`, `Quick`, `ReceivableAt` |
| `TestUnmarshalAcceptToParcelStep` | `accept_to_parcel` | `AcceptToParcelPayload` | `ParcelId`, `RecipientId`, `TemplateId`, `HasItem`, `Owner` |
| `TestUnmarshalReleaseFromParcelStep` | `release_from_parcel` | `ReleaseFromParcelPayload` | `ParcelId`, `RecipientId` |
| `TestUnmarshalWithdrawFromParcelStep` | `withdraw_from_parcel` | `WithdrawFromParcelPayload` | `ParcelId`, `CharacterId`, `InventoryType` |

Add a fifth, `TestParcelActionsAreKnown`, asserting all four actions appear in
the action list `world_transfer_test.go:97-118` enumerates.

- [ ] **Step 2: Run the tests and confirm they fail**

Run: `cd libs/atlas-saga && go test ./... -run 'Parcel'`
Expected: compile failure — `TransferToParcel` undefined.

- [ ] **Step 3: Add the constants, payloads and unmarshal arms**

Place the action block immediately after the MTS block in `model.go`, with a
comment matching the design §4.2 wording: `transfer_to_parcel` is a COMPOSITE
expanded into `release_from_character` + `accept_to_parcel`, the same shape as
`transfer_to_mts`.

- [ ] **Step 4: Run the tests and confirm they pass**

Run: `cd libs/atlas-saga && go build ./... && go test ./...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-saga
git commit -m "feat(saga): parcel custody actions, saga types and payloads"
```

---

## Task 12: Orchestrator — composite expansion for parcel custody

### Files

- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/model.go` — modify:
  add the four action aliases (near line 190) and the four payload aliases
  (near line 326), plus the `case` arms in the payload switch near line 1529
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/processor.go` — modify:
  add `TransferToParcel`/`WithdrawFromParcel` to the composite list near
  line 1182, the two `case` arms near line 1217, and the two new
  `expandTransferToParcel` / `expandWithdrawFromParcel` functions
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/parcel_expansion_test.go` — new file

Patterns to copy:
- `saga/processor.go:1943-2049` (`expandTransferToMts`) and `:2050-2130`
  (`expandWithdrawFromMts`) — read-only, the exact shape to mirror
- `saga/mts_expansion_test.go` — read-only, the expansion test shape
- `saga/storage_expansion_test.go` — read-only

Module root: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`.

**Interfaces consumed:** Task 11's actions and payloads.

**Interfaces produced:**
- `expandTransferToParcel(st Step[any]) ([]Step[any], error)` → two steps:
  `release_from_character` (`ReleaseFromCharacterPayload`) then
  `accept_to_parcel` (`AcceptToParcelPayload` carrying the snapshot looked up
  from the character's compartment by `AssetId`)
- `expandWithdrawFromParcel(st Step[any]) ([]Step[any], error)` → two steps:
  `release_from_parcel` (`ReleaseFromParcelPayload`) then
  `accept_to_character` (`AcceptToCharacterPayload`)

### Steps

- [ ] **Step 1: Write the failing test**

`TestExpandTransferToParcel` and `TestExpandWithdrawFromParcel` in
`parcel_expansion_test.go`, setup copied from `mts_expansion_test.go` (the
compartment-lookup stub and the `ProcessorImpl` construction it uses).

| subtest | payload | expect |
|---|---|---|
| `item parcel` | `TransferToParcelPayload` with `AssetId` 42 present in the stubbed compartment as template 1302000 with `Strength` 5 and `Owner` "Alice" | 2 steps; step 0 action `ReleaseFromCharacter` with `AssetId` 42; step 1 action `AcceptToParcel` with `HasItem` true, `TemplateId` 1302000, `Strength` 5, `Owner` "Alice" |
| `meso only` | `TransferToParcelPayload` with `AssetId` 0 and `Quantity` 0 | **1 step**: only `accept_to_parcel`, with `HasItem` false and a zero-valued snapshot — no `release_from_character` is emitted (RISK-2) |
| `asset missing` | `AssetId` 99 not in the compartment | error, no steps |
| `wrong payload type` | a `TransferToMtsPayload` | error `invalid payload type for TransferToParcel` |
| `withdraw` | `WithdrawFromParcelPayload` with `ParcelId` P, character 100, inventory type 1 | 2 steps; step 0 `ReleaseFromParcel` with `ParcelId` P; step 1 `AcceptToCharacter` for character 100 |

The `meso only` case is the sharpest edge in the design (RISK-2) and this test
is what pins it: a release/accept pair whose purpose is moving an asset must
not dispatch a meaningless release when there is no asset.

- [ ] **Step 2: Run the tests and confirm they fail**

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test ./saga/... -run Parcel`
Expected: compile failure — `expandTransferToParcel` undefined.

- [ ] **Step 3: Add the aliases in `saga/model.go`**

Four `Action` aliases and four payload type aliases to `sharedsaga`, in the
same block-per-subsystem style the MTS aliases use (lines 190-193, 326-329),
plus the four `case` arms in the payload-unmarshal switch (near line 1529).

- [ ] **Step 4: Write the two expansions in `saga/processor.go`**

`expandTransferToParcel` mirrors `expandTransferToMts`: call
`compartment.RequestCompartment(p.l, p.ctx)(payload.CharacterId,
payload.SourceInventoryType)`, linear-search `comp.Assets` for
`payload.AssetId`, and build the two steps from the found asset. Guard the
lookup entirely behind `payload.AssetId != 0` so a meso-only parcel does no
inventory round trip and emits one step.

Register both actions in the composite list near line 1182 and add the two
dispatch `case` arms near line 1217.

- [ ] **Step 5: Run the tests and confirm they pass**

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go build ./... && go test ./saga/...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-saga-orchestrator
git commit -m "feat(orchestrator): expand transfer_to_parcel and withdraw_from_parcel"
```

---

## Task 13: Orchestrator — parcel custody dispatch package and status consumer

### Files

- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/parcel/processor.go` — new file
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/parcel/producer.go` — new file
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/parcel/custody/kafka.go` — new file
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/consumer/parcel/custody/consumer.go` — new file
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/main.go` — modify:
  register the consumer (mirroring `mtsCustody` at lines 30, 126, 155)
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/parcel/processor_test.go` — new file

Patterns to copy:
- `saga-orchestrator/mts/processor.go` (the `Params` struct + `AndEmit`
  wrappers + `Buffer` methods) — read-only
- `saga-orchestrator/mts/producer.go` — read-only
- `saga-orchestrator/kafka/message/mts/custody/kafka.go` — read-only
- `saga-orchestrator/kafka/consumer/mts/custody/consumer.go` — read-only

Module root: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`.

**Interfaces consumed:** Task 11's payloads.

**Interfaces produced:**
- `custody.EnvCommandTopic = "COMMAND_TOPIC_PARCEL_CUSTODY"`
- `custody.EnvStatusTopic = "EVENT_TOPIC_PARCEL_CUSTODY_STATUS"`
- `custody.CommandAcceptToParcel = "ACCEPT_TO_PARCEL"`,
  `custody.CommandReleaseFromParcel = "RELEASE_FROM_PARCEL"`,
  `custody.CommandRestoreParcel = "RESTORE_PARCEL"`,
  `custody.CommandRemoveParcel = "REMOVE_PARCEL"`
- `custody.Command[E]` envelope + `AcceptToParcelCommandBody`,
  `ReleaseFromParcelCommandBody`, `RestoreParcelCommandBody`,
  `RemoveParcelCommandBody`
- status event types `StatusEventAccepted`, `StatusEventReleased`,
  `StatusEventError`
- `parcel.Processor` with `AcceptToParcelAndEmit`, `AcceptToParcel(mb)`,
  `ReleaseFromParcelAndEmit`, `ReleaseFromParcel(mb)`,
  `RestoreParcelAndEmit`, `RemoveParcelAndEmit` — the last two exist for the
  compensator (Task 14)
- `parcel.AcceptToParcelParams` — the flattened field set
  `AcceptToParcelPayload` carries

### Steps

- [ ] **Step 1: Write the failing test**

`TestParcelProcessorDispatch` in `parcel/processor_test.go`, setup copied from
`saga-orchestrator/mts/requests_drain_test.go` (buffer capture without a live
Kafka).

| subtest | call | expect on the buffer |
|---|---|---|
| `accept` | `AcceptToParcel(mb)(txId, params)` with `HasItem` true | one message on `EnvCommandTopic` whose `Type` is `ACCEPT_TO_PARCEL` and whose body carries `ParcelId`, `RecipientId`, `TemplateId` |
| `accept meso only` | same with `HasItem` false, `TemplateId` 0 | body `HasItem` false, `TemplateId` 0 |
| `release` | `ReleaseFromParcel(mb)(txId, parcelId, recipientId)` | one `RELEASE_FROM_PARCEL` message carrying both ids |
| `restore` | `RestoreParcel(mb)(txId, parcelId)` | one `RESTORE_PARCEL` message |
| `remove` | `RemoveParcel(mb)(txId, parcelId)` | one `REMOVE_PARCEL` message |

- [ ] **Step 2: Run the test and confirm it fails**

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test ./parcel/...`
Expected: compile failure — package does not exist.

- [ ] **Step 3: Write the kafka message package**

`kafka/message/parcel/custody/kafka.go` copies the MTS custody envelope
verbatim, renamed. `RestoreParcel` and `RemoveParcel` exist for the same
reason their MTS twins do and their doc comments must say so: `RESTORE_PARCEL`
un-resolves a parcel released by a `withdraw_from_parcel` whose
`accept_to_character` then failed (otherwise the item is lost);
`REMOVE_PARCEL` hard-deletes a still-`pending` parcel row created by a late
`accept_to_parcel` after its saga already compensated (otherwise the item is
duplicated). Both are idempotent: 0 rows affected is success.

- [ ] **Step 4: Write the processor and producer**

Mirror `mts/processor.go` exactly — pure `Buffer` methods plus `AndEmit`
wrappers, no direct producer calls inside the buffer methods.

- [ ] **Step 5: Write the status consumer and register it in `main.go`**

`kafka/consumer/parcel/custody/consumer.go` mirrors
`kafka/consumer/mts/custody/consumer.go`: subscribe to `EnvStatusTopic`,
translate each event into the orchestrator's step-completion call. Register it
in `main.go` at the three sites the `mtsCustody` import/`InitConsumers`/
`InitHandlers` lines occupy (30, 126, 155).

- [ ] **Step 6: Run the tests and confirm they pass**

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go build ./... && go test ./...`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-saga-orchestrator
git commit -m "feat(orchestrator): parcel custody command dispatch and status consumer"
```

---

## Task 14: Orchestrator — handler dispatch, event acceptance, compensation

### Files

- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go` — modify:
  add `handleAcceptToParcel` / `handleReleaseFromParcel` to the interface
  (near line 148), the dispatch `case` arms (near line 923) and the two
  handler implementations (near line 2427)
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/event_acceptance.go` — modify:
  add the four `EventKindParcelCustody*` constants (near line 80) and the
  action→event-kind entries (near line 245) and outcome entries (near line 431)
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/compensator.go` — modify:
  add the reverse-walk arms and the late-compensation entries
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/parcel_compensation_test.go` — new file

Patterns to copy:
- `saga/handler.go:2427-2510` (`handleAcceptToMtsListing`,
  `handleReleaseFromMtsHolding`) — read-only
- `saga/event_acceptance.go:80-83, 242-247, 431-433` — read-only
- `saga/compensator.go:2340-2430` (the MTS reverse walk) and `:2660-2700`
  (the late-compensation table) — read-only
- `saga/mts_dupe_safety_test.go`, `saga/trade_compensation_test.go` — read-only

Module root: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`.

**Interfaces consumed:** Task 13's `parcel.Processor`.

**Interfaces produced:**
- `EventKindParcelCustodyAccepted = "parcel.custody_accepted"`,
  `EventKindParcelCustodyReleased = "parcel.custody_released"`,
  `EventKindParcelCustodyError = "parcel.custody_error"`
- `HandlerImpl.parcelP parcel.Processor` field, wired wherever `mtsP` is wired

### Steps

- [ ] **Step 1: Write the failing test**

`TestParcelSendCompensation` and `TestParcelReceiveCompensation` in
`parcel_compensation_test.go`, setup copied from `mts_dupe_safety_test.go`
(the mock processors and the saga fixture builder it uses).

| subtest | saga state | expect |
|---|---|---|
| `send fails at accept` | `parcel_send` with `award_mesos` Completed, `release_from_character` Completed, `accept_to_parcel` Failed | compensation re-credits `mesoAmount + fee` AND re-grants the item from the `AcceptToParcelPayload` snapshot (stats preserved — Strength 5 and Owner "Alice" survive) |
| `send meso-only fails at accept` | `parcel_send` with `award_mesos` Completed and `accept_to_parcel` Failed, no `release_from_character` step | compensation re-credits the meso and attempts **no** item re-grant (RISK-2) |
| `send late accept` | `parcel_send` already compensated, a late `accept_to_parcel` success arrives | a `REMOVE_PARCEL` command is dispatched |
| `receive fails at accept_to_character` | `parcel_receive` with `release_from_parcel` Completed and `accept_to_character` Failed | a `RESTORE_PARCEL` command is dispatched for that parcel id |
| `receive fails at award_mesos` | `parcel_receive` with both custody steps Completed and `award_mesos` Failed | the reverse walk restores the parcel and the item is not left in the recipient's inventory |

The first case is the test the design calls out as earning its keep (§10): a
re-awarded item loses its stats and a released/re-accepted one does not.

- [ ] **Step 2: Run the tests and confirm they fail**

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test ./saga/... -run ParcelCompensation`
Expected: compile failure — `EventKindParcelCustodyAccepted` undefined.

- [ ] **Step 3: Add the handler arms**

`handleAcceptToParcel` type-asserts `AcceptToParcelPayload` and calls
`h.parcelP.AcceptToParcelAndEmit`; `handleReleaseFromParcel` does the same for
`ReleaseFromParcelPayload` and `ReleaseFromParcelAndEmit`. Both mirror their
MTS twins including the invalid-payload error text shape. Add the composite
actions to the "expanded, not handled" list near line 936.

- [ ] **Step 4: Add the event-acceptance entries**

```
sharedsaga.TransferToParcel:   {} // composite: expanded into release_from_character + accept_to_parcel
sharedsaga.WithdrawFromParcel: {} // composite: expanded into release_from_parcel + accept_to_character
sharedsaga.AcceptToParcel:     {EventKindParcelCustodyAccepted, EventKindParcelCustodyError}
sharedsaga.ReleaseFromParcel:  {EventKindParcelCustodyReleased, EventKindParcelCustodyError}
```

plus `EventKindParcelCustodyAccepted`/`Released` → `OutcomeSuccess` in the
outcome table.

- [ ] **Step 5: Add the compensation arms**

In the reverse walk: `ReleaseFromCharacter` in a `parcel_send` saga re-grants
from the sibling `AcceptToParcelPayload` snapshot (mirroring the
`assetDataFromMtsListingSnapshot` helper — add
`assetDataFromParcelSnapshot(p AcceptToParcelPayload) asset2.AssetData`);
`ReleaseFromParcel` compensates to `RestoreParcel`. In the late-compensation
table, `AcceptToParcel` compensates to `RemoveParcel` and `ReleaseFromParcel`
to `RestoreParcel`.

Guard the re-grant on `HasItem`: a meso-only `parcel_send` has no
`ReleaseFromCharacter` step at all, and a snapshot with `HasItem == false`
must never produce an award.

- [ ] **Step 6: Run the tests and confirm they pass**

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go build ./... && go test ./...`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-saga-orchestrator
git commit -m "feat(orchestrator): parcel custody handlers, event acceptance and compensation"
```

---

## Task 15: atlas-parcel custody consumer

### Files

- `services/atlas-parcel/atlas.com/parcel/kafka/consumer/consumer.go` — new file
  (the shared `NewConfig` helper)
- `services/atlas-parcel/atlas.com/parcel/kafka/message/custody/kafka.go` — new file
- `services/atlas-parcel/atlas.com/parcel/kafka/consumer/custody/consumer.go` — new file
- `services/atlas-parcel/atlas.com/parcel/kafka/producer/custody/producer.go` — new file
- `services/atlas-parcel/atlas.com/parcel/parcel/processor_custody.go` — new file
- `services/atlas-parcel/atlas.com/parcel/kafka/consumer/custody/consumer_test.go` — new file
- `services/atlas-parcel/atlas.com/parcel/main.go` — created in Task 1; modify:
  register the consumer and handlers

Patterns to copy:
- `services/atlas-mts/atlas.com/mts/kafka/consumer/consumer.go`,
  `kafka/consumer/custody/consumer.go`, `kafka/producer/custody/producer.go`,
  `kafka/message/custody/kafka.go` — read-only
- `services/atlas-mts/atlas.com/mts/holding/processor_custody.go`
  (`Release`/`RestoreHolding` and their idempotency comments) — read-only
- `services/atlas-mts/atlas.com/mts/kafka/consumer/custody/dupe_safety_test.go` — read-only

Module root: `services/atlas-parcel/atlas.com/parcel`.

**Interfaces consumed:** Task 3's `Processor`, Task 13's command envelope
(the wire shape must match field for field; the two packages are independent
Go types over the same JSON).

**Interfaces produced:**
- `parcel.ProcessorImpl.AcceptCustody(params AcceptParams) (Model, error)` —
  creates the `pending` row from the command body alone
- `parcel.ProcessorImpl.ReleaseCustody(parcelId uuid.UUID, recipientId uint32) (Model, error)` —
  transitions to `received` and returns the row's snapshot
- `parcel.ProcessorImpl.RestoreCustody(parcelId uuid.UUID) error` —
  `received` → `pending`, idempotent
- `parcel.ProcessorImpl.RemoveCustody(parcelId uuid.UUID) error` —
  hard-deletes a still-`pending` row, idempotent

Design §4.3: "The parcel row transitions to `received` as part of
`release_from_parcel`, inside atlas-parcel's own transaction — the status
change and the custody release are the same fact and must not be two steps
that can disagree."

### Steps

- [ ] **Step 1: Write the failing test**

`TestCustodyCommands` in `kafka/consumer/custody/consumer_test.go`, setup
copied from `services/atlas-mts/atlas.com/mts/kafka/consumer/custody/consumer_test.go`
(in-memory DB, a captured producer, a hand-built command message).

| subtest | command | expect |
|---|---|---|
| `accept with item` | `ACCEPT_TO_PARCEL`, `HasItem` true, template 1302000, Strength 5 | one `pending` row with `ItemId` non-nil == 1302000 and `ItemSnapshot.Strength` == 5; one `parcel.custody_accepted` status event |
| `accept meso only` | `ACCEPT_TO_PARCEL`, `HasItem` false, `MesoAmount` 5000 | one `pending` row with `ItemId` **nil** and `MesoAmount` 5000 |
| `accept replay` | the same `ACCEPT_TO_PARCEL` delivered twice | exactly one row (the `ParcelId` is allocated by the initiator, so creation is idempotent on replay) |
| `release` | `RELEASE_FROM_PARCEL` for a pending row | the row is `received`, `ResolvedAt` set; a `parcel.custody_released` event carrying the snapshot |
| `release replay` | `RELEASE_FROM_PARCEL` twice | second delivery affects 0 rows and still reports success — no second event |
| `release wrong recipient` | `RELEASE_FROM_PARCEL` with a recipient id that does not own the row | a `parcel.custody_error` event; the row stays `pending` |
| `restore` | `RESTORE_PARCEL` for a `received` row | the row is `pending` again, `ResolvedAt` cleared |
| `restore on a pending row` | `RESTORE_PARCEL` for an already-`pending` row | 0 rows affected, success |
| `remove` | `REMOVE_PARCEL` for a `pending` row | the row is gone |
| `remove on a received row` | `REMOVE_PARCEL` for a `received` row | the row is untouched, success (the guard is `status = pending`) |

- [ ] **Step 2: Run the test and confirm it fails**

Run: `cd services/atlas-parcel/atlas.com/parcel && go test ./kafka/...`
Expected: compile failure — package does not exist.

- [ ] **Step 3: Write the kafka message, consumer, producer and processor_custody**

Each handler runs one `database.ExecuteTransaction`, then emits its status
event through the per-delivery `producer.ProviderImpl(l)` factory so the ack
carries the right tenant/span headers (the MTS consumer's `providerFn`
comment explains why).

- [ ] **Step 4: Register the consumer in `main.go`**

`custodyConsumer.InitConsumers(l)(cmf)(consumerGroupId)` and
`custodyConsumer.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler)`,
mirroring `services/atlas-mts/atlas.com/mts/main.go`.

- [ ] **Step 5: Run the tests and confirm they pass**

Run: `cd services/atlas-parcel/atlas.com/parcel && go build ./... && go test ./...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-parcel
git commit -m "feat(parcel): custody command consumer and status events"
```

---

## Task 16: atlas-channel — fee, meso limit and send validation

### Files

- `services/atlas-channel/atlas.com/channel/parcel/fee.go` — new file
- `services/atlas-channel/atlas.com/channel/parcel/validation.go` — new file
- `services/atlas-channel/atlas.com/channel/parcel/fee_test.go` — new file
- `services/atlas-channel/atlas.com/channel/parcel/validation_test.go` — new file

Module root: `services/atlas-channel/atlas.com/channel`.

This task is pure functions with no I/O, so it is split from the handler
(Task 17) that calls them: the fee table is the most heavily asserted piece of
behaviour in the feature and deserves its own review surface.

**Interfaces produced:**
- `parcel.Fee(mesoAmount uint32) uint32` — the tiered fee alone
- `parcel.TotalCost(mesoAmount uint32, quick bool) (uint64, bool)` —
  `mesoAmount + Fee + (quick ? 0 : 5000)` computed in `uint64`; the bool is
  false when the result exceeds `math.MaxUint32`
- `parcel.SendSurcharge = uint32(5000)`, `parcel.MaxParcelMeso = uint32(100_000_000)`,
  `parcel.MesoLimitLevel = byte(15)`, `parcel.MesoLimitAmount = uint32(1_000_000)`,
  `parcel.MaxMessageLength = 100`, `parcel.MailboxCapacity = 10`
- `parcel.RejectReason` string type with the constants
  `RejectNone`, `RejectIncorrectRequest`, `RejectMesoLimit`,
  `RejectNotEnoughMesos`
- `parcel.ValidateSend(in SendInput) RejectReason` where `SendInput` carries
  `MesoAmount uint32`, `Quantity uint16`, `Quick bool`, `Message string`,
  `SenderLevel byte`, `SenderMeso uint32`

`ValidateSend` covers only the checks that need no remote lookup. Recipient
resolution, the same-account check, the mailbox count and the ticket check
live in Task 17 because each needs a round trip.

### Steps

- [ ] **Step 1: Write the failing fee test**

`TestFee` in `fee_test.go` — table-driven over every tier boundary the PRD
acceptance criteria name. The expected values are `uint32(float64(m) * rate)`
with the design §6.3 rates; if the implementation disagrees at any boundary
that is an implementation bug, not a table error — verify against the formula
directly, do not adjust the table to match the code.

| meso | expected fee |
|---|---|
| 0 | 0 |
| 99,999 | 0 |
| 100,000 | 800 |
| 999,999 | 7,999 |
| 1,000,000 | 18,000 |
| 4,999,999 | 89,999 |
| 5,000,000 | 150,000 |
| 9,999,999 | 299,999 |
| 10,000,000 | 400,000 |
| 24,999,999 | 999,999 |
| 25,000,000 | 1,250,000 |
| 99,999,999 | 4,999,999 |
| 100,000,000 | 6,000,000 |

`TestTotalCost` — the surcharge direction (FR-3, design §0 finding B):

| meso | quick | expected total | ok |
|---|---|---|---|
| 0 | false | 5,000 | true |
| 0 | true | 0 | true |
| 100,000 | false | 105,800 | true |
| 100,000 | true | 100,800 | true |
| 100,000,000 | false | 106,005,000 | true |
| 4,294,000,000 | false | — | **false** (overflows `uint32`) |

- [ ] **Step 2: Write the failing validation test**

`TestValidateSend` in `validation_test.go`. Baseline input: meso 1,000,
quantity 1, quick false, message "", sender level 30, sender meso 1,000,000.

| subtest | change from baseline | expect |
|---|---|---|
| `valid` | — | `RejectNone` |
| `nothing attached` | meso 0, quantity 0 | `RejectIncorrectRequest` |
| `item only` | meso 0, quantity 1 | `RejectNone` |
| `meso only` | meso 1000, quantity 0 | `RejectNone` |
| `over the parcel cap` | meso 100,000,001 | `RejectIncorrectRequest` |
| `at the parcel cap` | meso 100,000,000, sender meso 4,294,967,295 | `RejectNone` |
| `overflow` | meso 4,294,000,000 | `RejectIncorrectRequest` |
| `low level over limit` | level 15, meso 1,000,001, sender meso 4,294,967,295 | `RejectMesoLimit` |
| `low level at limit` | level 15, meso 1,000,000, sender meso 4,294,967,295 | `RejectNone` |
| `level 16 over limit` | level 16, meso 1,000,001, sender meso 4,294,967,295 | `RejectNone` |
| `cannot afford` | meso 1,000,000, sender meso 1,000,000 (fee 18,000 + 5,000 surcharge unaffordable) | `RejectNotEnoughMesos` |
| `can afford exactly` | meso 1,000,000, sender meso 1,023,000 | `RejectNone` |
| `message too long` | quick true, message of 101 characters | `RejectIncorrectRequest` |
| `message at the limit` | quick true, message of 100 characters | `RejectNone` |
| `message on a non-quick send` | quick false, message of 101 characters | `RejectNone` — the NPC arm carries no message, so there is nothing to validate (design §0 finding C) |

Check order matters: the parcel cap and the overflow check run before the
affordability check, so an absurd meso amount reports
`INCORRECT_REQUEST` rather than `NOT_ENOUGH_MESOS`. The table above pins that
order.

- [ ] **Step 3: Run the tests and confirm they fail**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./parcel/...`
Expected: compile failure — package `parcel` does not exist.

- [ ] **Step 4: Write `fee.go` and `validation.go`**

`Fee` is a descending `if` chain over the seven tiers, each returning
`uint32(float64(m) * rate)` with the rate as an untyped float constant. A
comment above it records why this is float and not integer arithmetic
(design §6.3: the client quotes the player its own double-derived figure
before the packet is sent; charging an integer-derived number charges a number
the player was not shown) and cites v72 `sub_6590A1` @0x6590A1 and v83
`sub_6EEDFE` @0x6EEDFE.

- [ ] **Step 5: Run the tests and confirm they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./parcel/...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/parcel
git commit -m "feat(channel): duey fee table and send validation"
```

---

## Task 17: atlas-channel — DUEY_ACTION handler and the ParcelSend saga

### Files

- `services/atlas-channel/atlas.com/channel/socket/handler/duey_action.go` — new file
- `services/atlas-channel/atlas.com/channel/socket/handler/duey_action_send.go` — new file
- `services/atlas-channel/atlas.com/channel/socket/handler/duey_action_send_test.go` — new file
- `services/atlas-channel/atlas.com/channel/parcel/requests.go` — new file
  (the atlas-parcel REST client)
- `services/atlas-channel/atlas.com/channel/parcel/processor.go` — new file
- `services/atlas-channel/atlas.com/channel/main.go` — modify: add
  `handlerMap[parcelsb.DueyActionHandle] = handler.DueyActionHandleFunc`
  beside the storage entry (line 1004)

Patterns to copy:
- `services/atlas-channel/atlas.com/channel/socket/handler/storage_operation.go`
  — read-only; lines 31-65, the mode dispatcher + `isStorageOperation` resolved-mode comparison) — read-only
- `services/atlas-channel/atlas.com/channel/socket/handler/note_send.go`
  (pre-flight → announce-inline-on-reject → build saga) — read-only
- `services/atlas-channel/atlas.com/channel/socket/handler/note_send_test.go` — read-only
- `services/atlas-character/atlas.com/character/pending_change/requests.go:184`
  (a narrow REST lookup) — read-only

Module root: `services/atlas-channel/atlas.com/channel`.

**Interfaces consumed:** Task 10's serverbound codecs, Task 8/9's clientbound
body functions, Task 16's `Fee`/`TotalCost`/`ValidateSend`, Task 11's saga
actions.

**Interfaces produced:**
- `handler.DueyActionHandleFunc` — the mode dispatcher, registered under
  `parcelsb.DueyActionHandle`
- `handler.DueyActionMode*` string constants `"SEND"`, `"RECEIVE"`,
  `"DISCARD"`, `"CLOSE"`
- `parcel.Processor` (channel-side) with
  `GetForRecipient(recipientId uint32, worldId world.Id) ([]Model, error)`,
  `CountPending(recipientId uint32) (int, error)`,
  `GetById(id uuid.UUID) (Model, error)`
- `handler.buildParcelSendSaga(...) saga.Saga`

### Steps

- [ ] **Step 1: Write the failing test**

`TestDueyActionSend` in `duey_action_send_test.go`, setup copied from
`note_send_test.go` (the session fixture, the writer capture, the saga-create
seam).

Baseline: sender character 100, account 1, world 0, level 30, meso 1,000,000;
recipient "Bob" resolving to character 200, account 2, world 0; the recipient
has 0 pending parcels; the sender holds one Quick Delivery Ticket.

| subtest | change | expect announced arm | expect saga |
|---|---|---|---|
| `npc send item and meso` | meso 1,000, item slot 5 qty 1, quick false | none until the saga completes | a `parcel_send` saga: step 0 `award_mesos` with `Amount` `-(1000+5000)`, step 1 `transfer_to_parcel` with `Quick` false; **no** `destroy_asset` step |
| `quick send` | quick true, message "hi" | none | steps: `award_mesos` `-(1000+0)`, `destroy_asset` of 5330000 qty 1, `transfer_to_parcel` with `Quick` true and `Message` "hi" |
| `meso only` | qty 0, meso 1,000 | none | `transfer_to_parcel` with `AssetId` 0 |
| `nothing attached` | qty 0, meso 0 | `INCORRECT_REQUEST` | no saga |
| `cannot afford` | sender meso 100 | `NOT_ENOUGH_MESOS` | no saga |
| `meso limit` | level 15, meso 1,000,001 | `MESO_LIMIT` | no saga |
| `unknown recipient` | "Nobody" resolves to nothing | `NAME_DOES_NOT_EXIST` | no saga |
| `recipient in another world` | "Bob" resolves only in world 1 | `NAME_DOES_NOT_EXIST` | no saga |
| `same account` | recipient account 1 | `SAME_ACCOUNT` | no saga |
| `mailbox full` | recipient has 10 pending parcels | `RECEIVER_STORAGE_FULL` | no saga |
| `mailbox at nine` | recipient has 9 pending parcels | none | saga created |
| `quick without a ticket` | quick true, sender holds no 5330000 | `INCORRECT_REQUEST` | no saga, and a warn-level log |
| `message too long` | quick true, 101-character message | `INCORRECT_REQUEST` | no saga |

**Every rejection subtest additionally asserts the session was NOT closed**
(NFR-5) — Cosmic disconnects and autobans on the packet-edit cases; Atlas
rejects and logs. Assert on the session fixture's close counter, exactly as
the note-send rejection tests do.

- [ ] **Step 2: Run the test and confirm it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/... -run TestDueyActionSend`
Expected: compile failure — `DueyActionHandleFunc` undefined.

- [ ] **Step 3: Write `parcel/requests.go` and `parcel/processor.go`**

The REST client hits `GET /parcels?filter[recipientId]=&filter[worldId]=&filter[status]=pending`
and `GET /parcels/{id}` on atlas-parcel. `CountPending` is the length of the
first call's result — the only pre-flight besides name resolution that leaves
the channel (design §6.2).

- [ ] **Step 4: Write `duey_action.go`**

Mirror `storage_operation.go`'s dispatcher exactly: decode
`parcelsb.Action`, read `p.Mode()`, and compare against each resolved
operation key with the same `isStorageOperation`-shaped helper
(`isDueyAction(l)(readerOptions, mode, DueyActionModeSend)`), never against a
literal. An unmatched mode logs at warn and returns.

- [ ] **Step 5: Write `duey_action_send.go`**

Order the pre-flight exactly as design §6.2 tabulates, with the local checks
(Task 16's `ValidateSend`) before the remote ones, then recipient resolution,
then same-account, then mailbox capacity, then the ticket check. Resolve the
recipient with
`character.NewProcessor(l, ctx).ByNameProvider(name)()` and filter the result
to `s.WorldId()` in the channel (design §6.1) — `ByNameProvider` is
tenant-scoped and name-filtered but not world-filtered, and the model already
exposes `WorldId()` and `AccountId()`
(`services/atlas-channel/atlas.com/channel/character/model.go:241,269`), which
is what makes the same-account check possible without a second lookup.

The ticket check reads the sender's ETC compartment via
`compartment.NewProcessor(l, ctx).GetByType(...)` and
`Model.FindFirstByItemId(item.QuickDeliveryTicketId)` (the constant Task 22
adds — if Task 22 has not landed, define it there and have Task 22 remove the
duplicate).

`buildParcelSendSaga` emits the design §4.3 step order — `award_mesos` first so
the irreversible-looking step's compensation is a credit rather than an item
re-mint (the same reasoning `note_send.go:17-19` records for destroy-first),
then the conditional `destroy_asset`, then `transfer_to_parcel`.

- [ ] **Step 6: Register the handler in `main.go`**

- [ ] **Step 7: Run the tests and confirm they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./socket/handler/... ./parcel/...`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-channel
git commit -m "feat(channel): DUEY_ACTION handler and the parcel_send saga"
```

---

## Task 18: atlas-channel — receive, discard and close arms

### Files

- `services/atlas-channel/atlas.com/channel/socket/handler/duey_action_receive.go` — new file
- `services/atlas-channel/atlas.com/channel/socket/handler/duey_action_receive_test.go` — new file
- `services/atlas-channel/atlas.com/channel/socket/handler/duey_action.go` (created in Task 17)
  — modify: route the `RECEIVE`, `DISCARD` and `CLOSE` modes
- `services/atlas-channel/atlas.com/channel/parcel/requests.go` — created in
  Task 17; modify: add the discard `PATCH` call

Patterns to copy: `storage_operation.go:83-160` (`handleRetrieveAsset` — a
saga-driven withdrawal with a pre-flight) — read-only.

Module root: `services/atlas-channel/atlas.com/channel`.

**Interfaces consumed:** Task 17's dispatcher and channel-side
`parcel.Processor`; Task 11's `WithdrawFromParcel`.

**Interfaces produced:**
- `handler.buildParcelReceiveSaga(...) saga.Saga` — `withdraw_from_parcel`
  then `award_mesos` (positive `MesoAmount`), per design §4.3
- `handler.handleParcelReceive`, `handler.handleParcelDiscard`,
  `handler.handleParcelClose`

### Steps

- [ ] **Step 1: Write the failing test**

`TestDueyActionReceive` and `TestDueyActionDiscard` in
`duey_action_receive_test.go`.

Baseline: character 100 in world 0; parcel P addressed to 100, `pending`,
`ReceivableAt` in the past, holding one equip of inventory type 1 and 5,000
meso; the recipient's EQUIP compartment has free slots.

| subtest | change | expect announced arm | expect saga |
|---|---|---|---|
| `receive happy path` | — | `PARCEL_REMOVED` with `kind` == `ParcelRemovedKindClaimed` on saga completion | `parcel_receive`: `withdraw_from_parcel` for P, then `award_mesos` `+5000` |
| `receive meso only` | P has no item | same | `withdraw_from_parcel` with `InventoryType` 0 and `award_mesos` `+5000` |
| `no free slot` | the EQUIP compartment is full | `RECV_NO_FREE_SLOTS` | no saga |
| `unique conflict` | the recipient already holds the one-of-a-kind item | `RECV_UNIQUE_CONFLICT` | no saga |
| `not receivable yet` | `ReceivableAt` in the future | `INCORRECT_REQUEST` | no saga |
| `not addressed to me` | P addressed to 999 | `INCORRECT_REQUEST` | no saga |
| `already received` | P `status` `received` | `INCORRECT_REQUEST` | no saga |
| `discard happy path` | — | `PARCEL_REMOVED` with `kind` == `ParcelRemovedKindDiscarded` | no saga; a `PATCH /parcels/{id}` setting `discarded` |
| `discard not mine` | P addressed to 999 | `INCORRECT_REQUEST` | no PATCH |
| `close` | mode CLOSE | nothing announced | no saga |

Every rejection subtest asserts the parcel stays `pending` (FR-15, FR-16) and
the session is not closed.

- [ ] **Step 2: Run the tests and confirm they fail**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/... -run 'TestDueyAction(Receive|Discard)'`
Expected: compile failure — `handleParcelReceive` undefined.

- [ ] **Step 3: Write `duey_action_receive.go`**

Pre-flight per design §7.2, in that order: free-slot check, unique-item check,
then the parcel-state check. Both inventory checks use the existing
`compartment.NewProcessor(l, ctx).GetByType` plus `Model.Capacity()` /
`Model.Assets()` / `Model.FindFirstByItemId` — no new endpoint.

Discard is not a saga (design §4.4): the contents are destroyed, nothing
leaves custody, so the channel PATCHes atlas-parcel directly and announces
`PARCEL_REMOVED` with `kind == 3`. The client already confirms via
`CUIFadeYesNo` (SP_3889) before sending, so no server-side confirmation exists
or is needed.

`CLOSE` clears whatever session-side dialog state the channel tracks and
announces nothing.

- [ ] **Step 4: Route the three modes in `duey_action.go`**

- [ ] **Step 5: Run the tests and confirm they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./socket/handler/... ./parcel/...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel
git commit -m "feat(channel): duey receive, discard and close arms"
```

---

## Task 19: Saga `ShowParcel` action and the SHOW_PARCEL command

### Files

- `libs/atlas-saga/model.go` — modify: add `ShowParcel Action = "show_parcel"`
- `libs/atlas-saga/payloads.go` — modify: add `ShowParcelPayload`
- `libs/atlas-saga/unmarshal.go` — modify: add the `case`
- `libs/atlas-saga/unmarshal_test.go` — modify: add the unmarshal test
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/parcel/kafka.go` — new file
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/parcel/processor.go` (created in Task 13)
  — modify: add `ShowParcelAndEmit` / `ShowParcel(mb)`
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go` — modify:
  add `handleShowParcel` and its dispatch `case`
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/event_acceptance.go` — modify:
  mark `ShowParcel` self-completing, like `ShowStorage`

Patterns to copy:
- `libs/atlas-saga/payloads.go:544-551` (`ShowStoragePayload`) — read-only
- `saga-orchestrator/storage/processor.go:103-130` (`ShowStorageAndEmit`) — read-only
- `saga-orchestrator/kafka/message/storage/kafka.go:19,110-120`
  (`CommandTypeShowStorage`, `ShowStorageCommand`) — read-only

Module roots: `libs/atlas-saga` and
`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`.

**Interfaces produced:**
- `saga.ShowParcel Action = "show_parcel"`
- `saga.ShowParcelPayload` — `CharacterId uint32`, `NpcId uint32`,
  `WorldId world.Id`, `ChannelId channel.Id`, `Quick bool`
- `parcelmsg.EnvCommandTopic = "COMMAND_TOPIC_PARCEL"`,
  `parcelmsg.CommandTypeShowParcel = "SHOW_PARCEL"`, `parcelmsg.ShowParcelCommand`

`Quick` on the payload lets one command serve both entry points: the NPC path
sends `Quick: false` (the channel announces `PARCEL[OPEN]` with the mailbox)
and the Quick Delivery Ticket path sends `Quick: true` (the channel announces
`PARCEL[OPEN_QUICK]`, mode 0x1A, the quick-send-only dialog with no list —
design §5.2, §9.5).

`ShowParcel` is **self-completing**, like `ShowStorage` and unlike
`OpenNpcShop`: nothing downstream depends on the dialog having opened, because
the ticket is consumed by the `parcel_send` saga and not by opening the
interface (FR-26).

### Steps

- [ ] **Step 1: Write the failing test**

`TestUnmarshalShowParcelStep` in `libs/atlas-saga/unmarshal_test.go`, modelled
on the existing `ShowStorage` unmarshal test: assert the action and that the
payload type-asserts with `CharacterId`, `NpcId`, `WorldId`, `ChannelId` and
`Quick` intact, including a `Quick: true` case.

- [ ] **Step 2: Run it and confirm it fails**

Run: `cd libs/atlas-saga && go test ./... -run ShowParcel`
Expected: compile failure — `ShowParcel` undefined.

- [ ] **Step 3: Add the action, payload and unmarshal arm**

- [ ] **Step 4: Add the orchestrator command message, processor method and handler**

`handleShowParcel` type-asserts `ShowParcelPayload`, calls
`h.parcelP.ShowParcelAndEmit`, and self-completes the step — mirroring
`handleShowStorage`.

- [ ] **Step 5: Run both modules' tests**

```bash
cd libs/atlas-saga && go build ./... && go test ./...
cd ../../services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go build ./... && go test ./...
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-saga services/atlas-saga-orchestrator
git commit -m "feat(saga): show_parcel action and the SHOW_PARCEL command"
```

---

## Task 20: NPC conversation entry point for Duey

### Files

- `services/atlas-npc-conversations/atlas.com/npc/conversation/operation_executor.go` — modify:
  add the `open_duey` case beside `open_storage` (line 2101)
- `services/atlas-npc-conversations/atlas.com/npc/conversation/operation_executor_test.go` — modify:
  add the `open_duey` test
- `deploy/seed/gms/72_1/npc-conversations/npc/npc-9010009.json` — new file
- `deploy/seed/gms/79_1/npc-conversations/npc/npc-9010009.json` — new file
- `deploy/seed/gms/83_1/npc-conversations/npc/npc-9010009.json` — new file
- `deploy/seed/gms/84_1/npc-conversations/npc/npc-9010009.json` — new file
- `deploy/seed/gms/87_1/npc-conversations/npc/npc-9010009.json` — new file
- `deploy/seed/gms/92_1/npc-conversations/npc/npc-9010009.json` — new file
- `deploy/seed/gms/95_1/npc-conversations/npc/npc-9010009.json` — new file
- `deploy/seed/jms/185_1/npc-conversations/npc/npc-9010009.json` — new file

Read-only reference:
`deploy/seed/gms/83_1/npc-conversations/npc/npc-2020004.json` — the
`genericAction` + `open_storage` seed to copy.

Module root: `services/atlas-npc-conversations/atlas.com/npc`.

Eight seed files is over the six-file guideline, but they are the same
mechanical file repeated per version — the case Step 5a explicitly allows to
batch. The NPC id lives in the seed data, where every other NPC id lives,
rather than becoming a Go constant (design §9.6).

**Interfaces consumed:** Task 19's `saga.ShowParcel` / `ShowParcelPayload`.

### Steps

- [ ] **Step 1: Write the failing test**

`TestExecuteOpenDuey` in `operation_executor_test.go`, setup copied from the
existing `open_storage` executor test.

| subtest | conversation context | expect |
|---|---|---|
| `emits show_parcel` | NPC 9010009, character 100, world 0, channel 1 | returns action `saga.ShowParcel`, status `saga.Pending`, and a `ShowParcelPayload` with `CharacterId` 100, `NpcId` 9010009, `WorldId` 0, `ChannelId` 1, `Quick` false |
| `missing context` | no previous conversation context | an error mentioning the conversation context, and no payload |

- [ ] **Step 2: Run it and confirm it fails**

Run: `cd services/atlas-npc-conversations/atlas.com/npc && go test ./conversation/... -run TestExecuteOpenDuey`
Expected: FAIL — the executor returns an unknown-operation error.

- [ ] **Step 3: Add the `open_duey` case**

Modelled on `open_storage` (operation_executor.go:2101) but with no params:
read the NPC id from `GetRegistry().GetPreviousContext(e.ctx, characterId)`,
build the payload from it plus `f.WorldId()` / `f.ChannelId()`, and return
`saga.ShowParcel` as a `Pending` step. `Quick` is false on this path.

- [ ] **Step 4: Write the eight seed files**

Each is `npc-2020004.json`'s shape with `npcId` 9010009, `startState`
`"openDuey"`, one `genericAction` state whose single operation is
`{"type": "open_duey", "params": {}}` and whose single outcome has
`"nextState": null`.

- [ ] **Step 5: Run the tests and confirm they pass**

Run: `cd services/atlas-npc-conversations/atlas.com/npc && go build ./... && go test ./...`
Expected: PASS.

- [ ] **Step 6: Verify the seed files parse**

Run: `for f in deploy/seed/*/*/npc-conversations/npc/npc-9010009.json; do python3 -m json.tool "$f" > /dev/null || echo "BAD $f"; done`
Expected: no output.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-npc-conversations deploy/seed
git commit -m "feat(npc): open_duey operation and NPC 9010009 conversations"
```

---

## Task 21: atlas-channel — SHOW_PARCEL consumer

### Files

- `services/atlas-channel/atlas.com/channel/kafka/message/parcel/kafka.go` — new file
- `services/atlas-channel/atlas.com/channel/kafka/consumer/parcel/consumer.go` — new file
- `services/atlas-channel/atlas.com/channel/kafka/consumer/parcel/consumer_test.go` — new file
- `services/atlas-channel/atlas.com/channel/parcel/model.go` — new file
  (the channel-side parcel model + a `ToPacket` mapper onto
  `libs/atlas-packet/parcel.Parcel`)
- `services/atlas-channel/atlas.com/channel/main.go` — modify: register the
  consumer alongside the storage consumer registration

Patterns to copy:
- `services/atlas-channel/atlas.com/channel/kafka/consumer/storage/consumer.go`
  (the SHOW_STORAGE command consumer) — read-only
- `services/atlas-channel/atlas.com/channel/kafka/consumer/merchant/consumer.go:454`
  (the tenant guard + `IfPresentByCharacterId` no-op) — read-only

Module root: `services/atlas-channel/atlas.com/channel`.

**Interfaces consumed:** Task 19's `SHOW_PARCEL` command, Task 7's
`ParcelOpenBody` / `ParcelOpenQuickBody`, Task 17's channel-side
`parcel.Processor`.

**Interfaces produced:**
- `parcel.Model.ToPacket() packetparcel.Parcel`
- the consumer handler `handleShowParcelCommand`

Design §5.3: the `OPEN` body's second list is the client's own "what showed up
while you were away" mechanism. Atlas populates it from parcels whose
`LastNotified` is null and stamps `LastNotified` when the open packet is
built — the cheapest correct implementation of FR-24, needing no extra packet.

### Steps

- [ ] **Step 1: Write the failing test**

`TestShowParcelCommand` in `consumer_test.go`, setup copied from the storage
consumer's test (the tenant-context harness and the announce capture).

| subtest | command / state | expect |
|---|---|---|
| `open with mailbox` | `Quick` false; the recipient has 2 receivable parcels, both `LastNotified` non-null | one announce of `PARCEL[OPEN]` with `quickEnabled` false, mailbox length 2, arrived length 0 |
| `open with new arrivals` | 2 receivable parcels, one with `LastNotified` null | mailbox length 2, arrived length 1 — and that parcel's `LastNotified` is stamped afterwards |
| `open quick` | `Quick` true | one announce of `PARCEL[OPEN_QUICK]`, no REST mailbox fetch |
| `not yet receivable excluded` | one parcel `ReceivableAt` in the future | mailbox length 0 (FR-12) |
| `wrong tenant` | the command's tenant differs from `sc.Tenant()` | no announce |
| `recipient offline` | the character has no session on this channel | no announce, no error |

- [ ] **Step 2: Run it and confirm it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./kafka/consumer/parcel/...`
Expected: compile failure — package does not exist.

- [ ] **Step 3: Write the message package, model mapper and consumer**

The consumer guards on `t.Is(sc.Tenant())` and announces through
`session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(...)`, so
an offline recipient is a silent no-op — the same shape as the frederick
notification handler.

`quickEnabled` on the `OPEN` body reflects whether the tenant's client
supports the quick tab; derive it from the same condition Task 22 uses to
decide whether the ticket path is live, and pass it through rather than
hard-coding true.

- [ ] **Step 4: Register the consumer in `main.go`**

- [ ] **Step 5: Run the tests and confirm they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./kafka/consumer/parcel/... ./parcel/...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel
git commit -m "feat(channel): SHOW_PARCEL consumer announcing PARCEL[OPEN]"
```

---

## Task 22: Quick Delivery Ticket — the classification-533 branch

### Files

- `libs/atlas-constants/item/constants.go` — modify: add
  `QuickDeliveryTicketId = uint32(5330000)` beside `ClassificationDueyCoupon`
  (line 109)
- `libs/atlas-constants/item/constants_test.go` — modify: add the constant test
- `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_duey.go` — new file
- `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_duey_test.go` — new file
- `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go` — modify:
  add the classification-first dispatch beside `handleRemoteMerchantUse`
  (line 787)

Patterns to copy:
- `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_remote_merchant.go`
  (the classification-first handler, the `<feature>Enabled(t tenant.Model)`
  version gate, the saga-create test seam) — read-only
- `character_cash_item_use_remote_merchant_test.go` — read-only

Module roots: `libs/atlas-constants` and
`services/atlas-channel/atlas.com/channel`.

**Interfaces consumed:** Task 19's `saga.ShowParcel` with `Quick: true`.

**Interfaces produced:**
- `item.QuickDeliveryTicketId`
- `handler.handleDueyCouponUse(l, ctx, wp)(s, t, itemId, source, it)`
- `handler.dueyCouponEnabled(t tenant.Model) bool`

Design §9.5: the handler announces the quick dialog and **consumes nothing** —
the ticket is destroyed by the `parcel_send` saga (FR-26), and the client
itself pre-checks `CWvsContext::IsExist(5330000)` before letting the player
send. `GetCashSlotItemType` already maps `ClassificationDueyCoupon` → 31
(`character_cash_item_use.go:1410`); that mapping does not change. Dispatch is
classification-first for the reason the file already documents at line 780:
the cash-slot type byte collides across features.

### Steps

- [ ] **Step 1: Write the failing tests**

`TestQuickDeliveryTicketId` in `libs/atlas-constants/item/constants_test.go`:
assert `QuickDeliveryTicketId == 5330000` and
`GetClassification(QuickDeliveryTicketId) == ClassificationDueyCoupon`.

`TestHandleDueyCouponUse` in
`character_cash_item_use_duey_test.go`, setup copied from
`character_cash_item_use_remote_merchant_test.go`:

| subtest | tenant / input | expect |
|---|---|---|
| `emits show_parcel quick` | GMS v83, item 5330000 | a saga with one `show_parcel` step whose payload has `Quick` true, `NpcId` 0, and the session's world/channel |
| `consumes nothing` | same | no `destroy_asset` step in the saga (FR-26) |
| `out of span` | GMS v61 | no saga, a debug/warn log, and the session is not closed |

- [ ] **Step 2: Run them and confirm they fail**

```bash
cd libs/atlas-constants && go test ./item/... -run TestQuickDeliveryTicketId
cd ../../services/atlas-channel/atlas.com/channel && go test ./socket/handler/... -run TestHandleDueyCouponUse
```

Expected: both fail — `QuickDeliveryTicketId` and `handleDueyCouponUse`
undefined.

- [ ] **Step 3: Add the constant**

- [ ] **Step 4: Write the handler and its version gate**

`dueyCouponEnabled` gates on the `PARCEL` span: `t.IsRegion("GMS") &&
t.MajorAtLeast(72)`, plus JMS 185. Derive the JMS condition the same way
`remoteMerchantEnabled` derives its GMS one, from the client's own
`get_cashslot_item_type` handling of classification 533 — if the JMS build
does not route 533, gate JMS off and record why in the function's doc comment.

- [ ] **Step 5: Add the dispatch branch**

Immediately after the `ClassificationRemoteMerchant` branch at line 787:

```go
if category == item.ClassificationDueyCoupon {
	handleDueyCouponUse(l, ctx, wp)(s, t, itemId, source, it)
	return
}
```

- [ ] **Step 6: Run the tests and confirm they pass**

```bash
cd libs/atlas-constants && go build ./... && go test ./...
cd ../../services/atlas-channel/atlas.com/channel && go build ./... && go test ./socket/handler/...
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add libs/atlas-constants services/atlas-channel
git commit -m "feat(channel): Quick Delivery Ticket opens the quick send dialog"
```

---

## Task 23: atlas-parcel expiry and return-to-sender sweep

### Files

- `services/atlas-parcel/atlas.com/parcel/parcel/task.go` — new file
- `services/atlas-parcel/atlas.com/parcel/parcel/task_test.go` — new file
- `services/atlas-parcel/atlas.com/parcel/parcel/administrator.go` — created in
  Task 2; modify: add the claim-by-update sweep
- `services/atlas-parcel/atlas.com/parcel/main.go` — created in Task 1; modify:
  start the task

Patterns to copy:
- `services/atlas-merchant/atlas.com/merchant/frederick/task.go`
  (the `Run()` + `SleepTime()` pair driven off
  `database.WithoutTenantFilter(ctx)`) — read-only
- `services/atlas-mts/atlas.com/mts/task/periodic.go` and its `periodic_test.go`
  — read-only

Module root: `services/atlas-parcel/atlas.com/parcel`.

**Interfaces consumed:** Task 2's administrator, Task 3's clock seam.

**Interfaces produced:**
- `parcel.DefaultExpiryInterval = 1 * time.Hour`
- `parcel.NewExpiryTask(l, ctx, db, interval) *ExpiryTask` with `Run()` and
  `SleepTime()`
- `parcel.ClaimExpired(db *gorm.DB) func(now time.Time, batch int) ([]Model, error)` —
  the claim-by-update from design §8.1

### Steps

- [ ] **Step 1: Resolve RISK-4 before writing the sweep**

Design §7.4 flags that v72's `CTabReceive::ReceiveParcel` @0x65AF41 refuses to
send a receive request when the parcel's timestamp falls outside a 30-day
window (`(parcelTime - now) / 864000000000 < 30`), and that the polarity of
that comparison decides whether server expiry stays at 30 days or moves to 29.
Pin the polarity from the IDB, then either keep `ExpiryWindow` at
`30 * 24 * time.Hour` or reduce it to `29 * 24 * time.Hour`, and record the
finding and the decision in `docs/tasks/task-241-duey-parcel-delivery/context.md`
under a "RISK-4 resolution" heading. **Do not assume** — this is a one-line
decision that must come from the client.

- [ ] **Step 2: Write the failing test**

`TestExpirySweep` in `task_test.go`, with the task's clock fixed at
`T = 2026-03-01T00:00:00Z` via Task 3's `withClock` seam.

| subtest | seeded | after one `Run()` |
|---|---|---|
| `expires and returns` | pending parcel, sender 100 ("Alice"), recipient 200 ("Bob"), `ExpiresAt` T-1h, item 1302000, meso 5000, `FeePaid` 800, `Returned` false | the original row is `expired` with `ResolvedAt` T; **one new** `pending` row with `SenderId` 200, `RecipientId` 100, `SenderName` "Bob", `Message` "Unclaimed parcel returned.", `Returned` true, `FeePaid` **0**, the same item snapshot and meso, `ReceivableAt` == `CreatedAt` (no 24-hour delay), `ExpiresAt` == `CreatedAt + ExpiryWindow` |
| `return leg expires into nothing` | pending parcel with `Returned` true, `ExpiresAt` T-1h | the row is `expired`; **no** new row |
| `not yet expired` | pending parcel, `ExpiresAt` T+1h | unchanged, no new row |
| `already resolved` | `received` parcel, `ExpiresAt` T-1h | unchanged, no new row |
| `meso-only return` | pending, no item, meso 5000, `ExpiresAt` T-1h | the return leg has a nil `ItemId` and meso 5000 |
| `batch bound` | 5 expired parcels, batch size 2 | exactly 2 rows claimed per `Run()`; three `Run()`s drain them all |
| `concurrent claim` | 1 expired parcel, two `Run()`s racing on the same DB | exactly one return leg is created |

The last case is what design §8.1 (NFR-7) buys: only one replica's `UPDATE`
claims a given row, so no leader election is needed and the operation stays
safe under multiple replicas.

- [ ] **Step 3: Run it and confirm it fails**

Run: `cd services/atlas-parcel/atlas.com/parcel && go test ./parcel/... -run TestExpirySweep`
Expected: compile failure — `NewExpiryTask` undefined.

- [ ] **Step 4: Write `ClaimExpired` and the task**

`ClaimExpired` is one claim-by-update, batched:

```sql
UPDATE parcels SET status = 'expired', resolved_at = ?
 WHERE status = 'pending' AND expires_at <= ?
 LIMIT ?
 RETURNING *;
```

`Run()` calls it under `database.WithoutTenantFilter(t.ctx)`, then for each
claimed row with `Returned == false` re-enters that row's tenant context and
inserts the return leg. `SenderName` on the return leg is set to the original
**recipient's** name so the parcel reads as coming back from the person who
never claimed it, and `Message` is the server-authored
`"Unclaimed parcel returned."` — there is no wire field for "this is a
return", so the distinction is carried by the two fields that do exist
(design §7.4, OQ-7).

- [ ] **Step 5: Start the task in `main.go`**

Mirror atlas-mts's `expirationTask` block: construct, `Start()`, and register
a `rt.TeardownFunc(task.Stop)`. The interval reads
`PARCEL_EXPIRY_INTERVAL_SECONDS` and falls back to `DefaultExpiryInterval`.

- [ ] **Step 6: Run the tests and confirm they pass**

Run: `cd services/atlas-parcel/atlas.com/parcel && go build ./... && go test ./...`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-parcel docs/tasks/task-241-duey-parcel-delivery/context.md
git commit -m "feat(parcel): expiry sweep with return-to-sender"
```

---

## Task 24: atlas-parcel notification sweep

### Files

- `services/atlas-parcel/atlas.com/parcel/parcel/notification_task.go` — new file
- `services/atlas-parcel/atlas.com/parcel/parcel/notification_task_test.go` — new file
- `services/atlas-parcel/atlas.com/parcel/kafka/message/parcel/kafka.go` — new file
- `services/atlas-parcel/atlas.com/parcel/kafka/producer/parcel/producer.go` — new file
- `services/atlas-parcel/atlas.com/parcel/main.go` — created in Task 1; modify:
  start the task

Patterns to copy:
- `services/atlas-merchant/atlas.com/merchant/frederick/notification_task.go`
  and `notification_task_test.go` (the `envContext` re-entry before any Kafka
  emit, the topic-not-configured early return) — read-only

Module root: `services/atlas-parcel/atlas.com/parcel`.

**Interfaces consumed:** Task 2's administrator (`StampNotified`).

**Interfaces produced:**
- `parcel.DefaultNotificationInterval = 5 * time.Minute`
- `parcel.NewNotificationTask(l, ctx, db, interval, envContext) *NotificationTask`
- `parcelmsg.EnvStatusEventTopic = "EVENT_TOPIC_PARCEL_STATUS"`,
  `parcelmsg.StatusEventParcelArrived = "PARCEL_ARRIVED"`,
  `parcelmsg.StatusEvent[E]` + `StatusEventParcelArrivedBody{SenderName string, HasItem bool}`

Design §7.1: there is no notification tier ladder. Frederick escalates because
a hired merchant's goods rot for 100 days; a parcel has one arrival event and
one 30-day death, so a single nullable `LastNotified` is enough, and it has one
meaning — "the player has been told about this parcel once".

### Steps

- [ ] **Step 1: Write the failing test**

`TestNotificationSweep` in `notification_task_test.go`, clock fixed at `T`.

| subtest | seeded | after one `Run()` |
|---|---|---|
| `notifies a newly receivable parcel` | pending, `ReceivableAt` T-1h, `LastNotified` nil, sender "Alice", has item | one `PARCEL_ARRIVED` event addressed to the recipient with `SenderName` "Alice" and `HasItem` true; `LastNotified` stamped to T |
| `does not renotify` | the same parcel with `LastNotified` T-1h | no event emitted |
| `not yet receivable` | pending, `ReceivableAt` T+1h, `LastNotified` nil | no event, `LastNotified` still nil |
| `resolved parcel` | `received`, `LastNotified` nil | no event |
| `offline recipient still stamps` | the recipient has no session anywhere | the event is still emitted and `LastNotified` is still stamped — FR-24 is served by the OPEN packet's second list, not by this sweep (design §7.1) |
| `topic not configured` | `EVENT_TOPIC_PARCEL_STATUS` unset | `Run()` warns and returns without stamping anything |
| `concurrent claim` | one due parcel, two `Run()`s racing | exactly one event |

- [ ] **Step 2: Run it and confirm it fails**

Run: `cd services/atlas-parcel/atlas.com/parcel && go test ./parcel/... -run TestNotificationSweep`
Expected: compile failure — `NewNotificationTask` undefined.

- [ ] **Step 3: Write the message package, producer and task**

The sweep uses the same claim-by-update shape as Task 23 (design §8.1), on
`last_notified IS NULL AND status = 'pending' AND receivable_at <= NOW()`.
Query under `database.WithoutTenantFilter`, then re-enter the per-tenant
context via `envContext` before any Kafka emit — frederick's task does exactly
this and the ordering is load-bearing for tenant headers.

- [ ] **Step 4: Start the task in `main.go`**

Interval from `PARCEL_NOTIFICATION_INTERVAL_SECONDS`, falling back to
`DefaultNotificationInterval`.

- [ ] **Step 5: Run the tests and confirm they pass**

Run: `cd services/atlas-parcel/atlas.com/parcel && go build ./... && go test ./...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-parcel
git commit -m "feat(parcel): arrival notification sweep"
```

---

## Task 25: atlas-channel — parcel arrival alarm

### Files

- `services/atlas-channel/atlas.com/channel/kafka/message/parcel/kafka.go` (created in Task 21)
  — modify: add the status-event types
- `services/atlas-channel/atlas.com/channel/kafka/consumer/parcel/consumer.go` (created in Task 21)
  — modify: add the status-event consumer and handler
- `services/atlas-channel/atlas.com/channel/kafka/consumer/parcel/status_test.go` — new file
- `services/atlas-channel/atlas.com/channel/main.go` — modify: register the
  status consumer

Patterns to copy:
- `services/atlas-channel/atlas.com/channel/kafka/consumer/merchant/consumer.go`
  — read-only; lines 454-482, `handleFrederickNotificationEvent` — the tenant guard plus
  `IfPresentByCharacterId`) — read-only

Module root: `services/atlas-channel/atlas.com/channel`.

**Interfaces consumed:** Task 24's `PARCEL_ARRIVED` status event, Task 9's
`ParcelAlarmNamedBody`.

### Steps

- [ ] **Step 1: Write the failing test**

`TestParcelArrivedEvent` in `status_test.go`.

| subtest | event / state | expect |
|---|---|---|
| `online recipient` | `PARCEL_ARRIVED`, sender "Alice", hasItem true, recipient has a session on this channel | one announce of `PARCEL[ALARM_NAMED]` with `senderName` "Alice" and `hasItem` true |
| `no item` | hasItem false | `ALARM_NAMED` with `hasItem` false |
| `offline recipient` | no session on this channel | no announce, no error |
| `wrong tenant` | the event's tenant differs from `sc.Tenant()` | no announce |
| `wrong event type` | a different `Type` on the envelope | no announce |

- [ ] **Step 2: Run it and confirm it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./kafka/consumer/parcel/... -run TestParcelArrivedEvent`
Expected: FAIL — the handler does not exist.

- [ ] **Step 3: Write the handler**

Mirror `handleFrederickNotificationEvent`: guard on the event type, guard on
`t.Is(sc.Tenant())`, then announce through `IfPresentByCharacterId`.

The channel always sends `ALARM_NAMED` (0x19), never `PARCEL_ARRIVED` (0x18).
Design §7.1 makes this an explicit, documented trade: 0x18 both appends the
row and raises SP_3902, but choosing it needs the channel to know the session
has an open parcel dialog, which it does not track. A player with the dialog
open sees a toast rather than a live row — low severity. Record that in the
handler's doc comment.

- [ ] **Step 4: Register the consumer in `main.go`**

- [ ] **Step 5: Run the tests and confirm they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./kafka/consumer/parcel/...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel
git commit -m "feat(channel): parcel arrival alarm consumer"
```

---

## Task 26: atlas-character — world-transfer gate 12 `parcel_pending`

### Files

- `services/atlas-character/atlas.com/character/pending_change/processor_eligibility.go` — modify:
  add the `parcelPending` field to `gateDeps` (after `mtsHolding`, line 49),
  wire it in `productionGateDeps()` (line 65), add `checkParcelPending`
  (after `checkMtsHolding`, line 385), and append it as gate 12 in **BOTH**
  `evaluateTransferEligibility` (after line 201) and
  `evaluateTransferEligibilityIndependent` (after line 224)
- `services/atlas-character/atlas.com/character/pending_change/requests.go` — modify:
  add `parcelPending` beside `mtsHoldingOpen` (line 184)
- `services/atlas-character/atlas.com/character/pending_change/processor_eligibility_test.go` — modify:
  add the gate-12 tests
- `services/atlas-character/atlas.com/character/pending_change/requests_test.go` — modify:
  add the REST-client test

Read-only references: `processor_eligibility.go:373-385` (`checkMtsHolding` —
the exact method shape to copy) and `requests.go:180-197` (`mtsHoldingOpen`).

Module root: `services/atlas-character/atlas.com/character`.

**Interfaces consumed:** Task 4's `GET /characters/{characterId}/parcel-status`.

**Interfaces produced:**
- `gateDeps.parcelPending func(l logrus.FieldLogger, ctx context.Context, characterId uint32) (bool, error)`
- `func parcelPending(l logrus.FieldLogger, ctx context.Context, characterId uint32) (bool, error)` in `requests.go`
- `(*ProcessorImpl).checkParcelPending(c character.Model) (string, bool, error)`
  returning `("parcel_pending", true, nil)` when a parcel is in flight

Gate 12 is **destination-INDEPENDENT**. It must appear in both evaluate
functions in the same relative order — that is the file's documented invariant
and the reason the CHECK-time handler
(`cash_shop_check_transfer_world_possible.go`) and the BUY-time handler agree
(FR-28). A dependency error propagates so `runGates` converts it to
`check_unavailable`, failing closed without asserting a reason the server does
not actually know to hold.

Blocking rather than auto-cancelling is deliberate (FR-31): the player can
resolve it themselves by receiving or discarding, or by waiting out expiry,
and a silent auto-return during a transfer would be a cross-world asset
movement — exactly what same-world delivery forbids.

### Steps

- [ ] **Step 1: Write the failing tests**

Add to `processor_eligibility_test.go`, using
`withTransferEligibilityGates` to stub `parcelPending` and leaving every other
gate passing. Test both entry points.

| subtest | `parcelPending` stub | call | expect |
|---|---|---|---|
| `blocks buy-time` | `(true, nil)` | `CheckTransferEligibility(id, dest)` | `ok` false, reason `"parcel_pending"` |
| `blocks check-time` | `(true, nil)` | `CheckTransferEligibilityIndependent(id)` | `ok` false, reason `"parcel_pending"` |
| `passes buy-time` | `(false, nil)` | `CheckTransferEligibility(id, dest)` | `ok` true, reason `""` |
| `passes check-time` | `(false, nil)` | `CheckTransferEligibilityIndependent(id)` | `ok` true, reason `""` |
| `dependency error buy-time` | `(false, errors.New("boom"))` | `CheckTransferEligibility(id, dest)` | `ok` false, reason `"check_unavailable"` — **not** `"parcel_pending"` |
| `dependency error check-time` | `(false, errors.New("boom"))` | `CheckTransferEligibilityIndependent(id)` | `ok` false, reason `"check_unavailable"` |
| `runs after mts` | both `mtsHolding` and `parcelPending` return true | either entry point | reason `"mts_listings_open"` — gate 11 precedes gate 12, so the earlier gate's reason wins |

Add to `requests_test.go`:

| subtest | stubbed HTTP response | expect |
|---|---|---|
| `in flight` | 200 with `data.attributes.inFlight` true | `(true, nil)` |
| `not in flight` | 200 with `inFlight` false | `(false, nil)` |
| `service down` | 503 | `(false, err)` — never `(false, nil)`, which would silently permit the transfer |

- [ ] **Step 2: Run the tests and confirm they fail**

Run: `cd services/atlas-character/atlas.com/character && go test ./pending_change/... -run Parcel`
Expected: compile failure — `parcelPending` undefined.

- [ ] **Step 3: Add the dependency, the REST client and the gate**

`checkParcelPending` is `checkMtsHolding` with the names changed, including
the error-log line shape. Append the gate to both slices with a comment
matching the file's style:

```go
// Gate 12 (destination-INDEPENDENT): a parcel in flight in either
// direction. Same-world delivery means a transfer would strand it, and
// auto-returning during a transfer would itself be a cross-world asset
// movement (design §9.1, FR-31).
func() (string, bool, error) { return p.checkParcelPending(c) },
```

- [ ] **Step 4: Run the tests and confirm they pass**

Run: `cd services/atlas-character/atlas.com/character && go build ./... && go test ./pending_change/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-character
git commit -m "feat(character): world-transfer gate 12 parcel_pending"
```

---

## Task 27: Surface `parcel_pending` in every seed template

### Files

- `services/atlas-configurations/seed-data/templates/template_gms_48_1.json:3339`
- `services/atlas-configurations/seed-data/templates/template_gms_61_1.json:4165`
- `services/atlas-configurations/seed-data/templates/template_gms_72_1.json:4395`
- `services/atlas-configurations/seed-data/templates/template_gms_79_1.json:4694`
- `services/atlas-configurations/seed-data/templates/template_gms_83_1.json:5042`
- `services/atlas-configurations/seed-data/templates/template_gms_84_1.json:5108`
- `services/atlas-configurations/seed-data/templates/template_gms_87_1.json:4814`
- `services/atlas-configurations/seed-data/templates/template_gms_92_1.json:3366`
- `services/atlas-configurations/seed-data/templates/template_gms_95_1.json:4921`

Nine files, one mechanical edit each — the batching case Step 5a allows.
It is deliberately its own task because a missed file is **silent**: that
version shows "please try again" instead of "cannot transfer out" and nothing
fails (design RISK-5).

`template_jms_185_1.json` and `template_gms_12_1.json` carry no world-transfer
reason map at all (verified: `grep -n "mts_listings_open"` matches only the
nine files above) and are correctly untouched.

**Interfaces consumed:** Task 26's `"parcel_pending"` reason string.

### Steps

- [ ] **Step 1: Establish the failing check**

List the nine files that carry the reason map:

```
grep -rl mts_listings_open services/atlas-configurations/seed-data/templates
```

Expected: exactly the nine `template_gms_{48,61,72,79,83,84,87,92,95}_1.json`
files above, and `grep -c parcel_pending` over the same set prints `0` for each.

- [ ] **Step 2: Add the key to each file**

In each template, `parcel_pending` joins the `CANNOT_TRANSFER_OUT` bucket —
the same numeric value `merchant_open` and `mts_listings_open` already carry
in that file. **Read the value from each file individually; never copy it
across files** — the bucket's number differs per template (222 in gms_83,
231 in gms_84, 237 in gms_87, 60 in gms_92 and gms_95, 197 in gms_72, 211 in
gms_79, 179 in gms_61, 155 in gms_48 — re-read each rather than trusting this
list). Insert it immediately after the `mts_listings_open` line, preserving
the file's existing trailing-comma shape (`gms_48` and `gms_61` have
`mts_listings_open` as the last key in its object, so the comma moves).

- [ ] **Step 3: Verify all nine and check the JSON still parses**

```bash
grep -c parcel_pending services/atlas-configurations/seed-data/templates/template_gms_48_1.json \
  services/atlas-configurations/seed-data/templates/template_gms_61_1.json \
  services/atlas-configurations/seed-data/templates/template_gms_72_1.json \
  services/atlas-configurations/seed-data/templates/template_gms_79_1.json \
  services/atlas-configurations/seed-data/templates/template_gms_83_1.json \
  services/atlas-configurations/seed-data/templates/template_gms_84_1.json \
  services/atlas-configurations/seed-data/templates/template_gms_87_1.json \
  services/atlas-configurations/seed-data/templates/template_gms_92_1.json \
  services/atlas-configurations/seed-data/templates/template_gms_95_1.json
for f in services/atlas-configurations/seed-data/templates/template_*.json; do python3 -m json.tool "$f" > /dev/null || echo "BAD $f"; done
```

Expected: `1` for each of the nine; no `BAD` lines.

Then, for each of the nine, confirm the value written equals that file's own
`mts_listings_open` value — read both keys out of the file and compare. A value
copied from another template is the RISK-5 failure this task exists to prevent.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-configurations/seed-data/templates
git commit -m "feat(config): map parcel_pending to CANNOT_TRANSFER_OUT in all nine templates"
```

---

## Task 28: Promote the coverage matrix and close out the packet record

### Files

- `docs/packets/audits/` — regenerated reports and evidence records
- `docs/packets/audits/STATUS.md` and `status.json` — regenerated
- `docs/tasks/task-241-duey-parcel-delivery/coverage-manifest.yaml` — new file

Read-only references: `docs/packets/audits/VERIFYING_A_PACKET.md`,
`docs/packets/DISPATCHER_FAMILY.md` ("Family complete" checklist).

This task runs last because the matrix grades codec byte-correctness against
committed fixtures, and every fixture must already exist (Tasks 7–10).

### Steps

- [ ] **Step 1: Write the coverage manifest**

`coverage-manifest.yaml` declares every op × version this task claims:
`PARCEL` clientbound and `DUEY_ACTION` serverbound across gms v72, v79, v83,
v84, v87, v92, v95 and jms v185. This is what
`packet-completeness-critic` diffs against the branch's actual git and matrix
delta before the PR.

- [ ] **Step 2: Verify each arm per version**

For each `PARCEL` arm and each in-span version, follow
`VERIFYING_A_PACKET.md`: the synthetic `#`-suffixed export entry, the audit
report, the byte fixture with its `// packet-audit:verify` marker, and the
pinned evidence record. Same for the four `DUEY_ACTION` arms.

A cell that does not promote is a failure report, never a prose claim. If a
version's client genuinely lacks an arm, mark it version-absent (`⬜`) in
`parcel.yaml` with the decompile address that shows the switch has no case for
it — do not claim a cell the client does not have.

- [ ] **Step 3: Regenerate and check**

```bash
go run ./tools/packet-audit matrix
go run ./tools/packet-audit matrix --check
go run ./tools/packet-audit dispatcher-lint
go run ./tools/packet-audit fname-doc --check
go run ./tools/packet-audit operations --check
```

Expected: all five exit 0, and `PARCEL` is **not** present in
`docs/packets/dispatcher-lint-baseline.yaml` (the baseline is empty and only
shrinks — a new family is authored discrete-per-mode from the start).

- [ ] **Step 4: Confirm the family-complete checklist**

Walk `DISPATCHER_FAMILY.md`'s "Family complete" checklist and reproduce it in
the PR description: one discrete struct per mode in one consolidated file;
each `Encode` writes the full arm body; every constructor takes `mode byte`
and every body func resolves it (zero `mode: 0x` literals, zero
`func(_ byte)`); no body func takes a caller-supplied selector; no struct
serves more than one mode; per-mode `#`-entry, export entry, report, fixture
and evidence; the four commands exit 0.

- [ ] **Step 5: Commit**

```bash
git add docs/packets docs/tasks/task-241-duey-parcel-delivery
git commit -m "docs(packets): promote PARCEL and DUEY_ACTION coverage cells"
```

---

## Final verification

After Task 28, before requesting review:

```bash
tools/verify.sh
```

Only the flagless invocation counts. Then run the code-review step
(`superpowers:requesting-code-review`), which dispatches
`backend-guidelines-reviewer`, `plan-adherence-reviewer` and — because this is
a packet task — `packet-completeness-critic` against
`coverage-manifest.yaml`.

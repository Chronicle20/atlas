# Mystic Door (Priest) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement Mystic Door (skill `2311002`) as a new version-agnostic engine service `atlas-doors` plus an `atlas-channel` packet edge, so a Priest can deploy a party-shared two-map town portal on every supported tenant version.

**Architecture:** A new Redis-backed, per-tenant `atlas-doors` service owns door lifecycle (registry, shared object-id allocation, per-party town-slot allocation, leader-elected expiry, Kafka command/event topics, REST) — modeled one-for-one on the in-flight `atlas-summons`. `atlas-channel` stays the thin per-version packet edge: it routes the cast to a SPAWN command, decodes the enter-door packet and warps via the existing portal path, and consumes door status events to broadcast spawn/remove/party-minimap packets to eligible (same-channel party) sessions. All version variance lives in a new `libs/atlas-packet/door` package plus per-version tenant socket-template opcodes.

**Tech Stack:** Go 1.22+, libs/atlas-redis (Registry/KeyedSet), libs/atlas-object-id, libs/atlas-lock (leader election), libs/atlas-kafka, libs/atlas-model, libs/atlas-rest (JSON:API via api2go), libs/atlas-tenant, libs/atlas-constants, libs/atlas-packet, libs/atlas-socket.

**Read first:** `context.md` (in this folder) — it maps the atlas-summons template location, the channel seams, the atlas-data portal facts, and the locked-in decisions. Every "copy from atlas-summons" instruction below refers to `.worktrees/task-088-player-summons/services/atlas-summons/atlas.com/summons/`.

---

## Phase ordering & rationale

- **Phase 0** — de-risk the one open data question (town door portals) and stand up the empty service so later phases compile.
- **Phase 1** — the version-agnostic engine domain (model, registry, slot, id allocation, data clients, processor). Pure logic, fully TDD-able offline.
- **Phase 2** — wire the engine into Kafka consumers + REST + leader task + `main.go`.
- **Phase 3** — `libs/atlas-packet/door` encoders/decoder (v83 from Cosmic; golden-byte tests).
- **Phase 4** — the `atlas-channel` edge (cast handler, enter-door handler, broadcast consumer, map-enter spawn, partyPortal).
- **Phase 5** — per-version opcode resolution + tenant template/live-config wiring (the OQ-5 matrix).
- **Phase 6** — full verification + service registration completeness.

Commit after every task. Run `go test -race ./...` in the changed module after each implementation step that the task says to.

---

## Phase 0 — De-risk & scaffold

### Task 0: Verify town door-portal data (OQ-3)

**No code.** This is the highest-priority pre-implementation data check (design §6.3, §12). Findings drive the slot→portal fallback in Task 9.

**Files:**
- Modify: `docs/tasks/task-093-mystic-door/context.md` (append a "## Town portal verification (Task 0)" section with the findings table)

- [ ] **Step 1: Inspect the door portals for the canonical return towns.** Use the verified inspection method (memory `reference_atlas_data_wz_inspection`). Either query a live tenant or read local WZ dumps. For each of these town map ids, count portals with `pt == 6` (door type):

  Towns to check: `100000000` Henesys, `101000000` Ellinia, `102000000` Perion, `103000000` Kerning City, `104000000` Lith Harbor, `105040300` Sleepywood, `120000000` Nautilus, `200000000` Orbis, `220000000` Ludibrium, `230000000` Aquarium.

  Live-tenant method (throwaway curl pod per `reference_atlas_data_wz_inspection`), example:
  ```
  GET /api/data/maps/100000000/portals
  Headers: TENANT_ID, REGION=GMS, MAJOR_VERSION=83, MINOR_VERSION=1
  ```
  Count entries where `attributes.type == 6`.

- [ ] **Step 2: Record the count per town per version** (`gms_v83`, and spot-check `gms_v95` + `jms_v185`) in a table in context.md.

- [ ] **Step 3: Decide the fallback.** If every checked town exposes ≥6 door portals, the §6.3 happy path always applies and the fallback (Task 9) is defensive only. If any town has <6, document which, and confirm the Task 9 fallback (default door position near the town's spawn portal) is acceptable. Note the conclusion in context.md.

- [ ] **Step 4: Commit**
  ```bash
  git add docs/tasks/task-093-mystic-door/context.md
  git commit -m "docs(task-093): record town door-portal verification (OQ-3)"
  ```

---

### Task 1: Service registration + empty `atlas-doors` module

Stand up a compiling, empty module so subsequent tasks have a home. No domain logic yet.

**Files:**
- Create: `services/atlas-doors/atlas.com/doors/go.mod`
- Create: `services/atlas-doors/atlas.com/doors/main.go`
- Create: `services/atlas-doors/atlas.com/doors/logger/logger.go` (copy from atlas-summons `logger/`)
- Modify: `go.work`
- Modify: `.github/config/services.json`
- Modify: `docker-bake.hcl`

- [ ] **Step 1: Copy the module skeleton from atlas-summons.** Copy `go.mod`, `main.go`, and `logger/` from `.worktrees/task-088-player-summons/services/atlas-summons/atlas.com/summons/` into `services/atlas-doors/atlas.com/doors/`. Then:
  - In `go.mod`, change `module atlas-summons` → `module atlas-doors`. Keep the same `require` block (atlas-redis, atlas-object-id, atlas-lock, atlas-kafka, atlas-model, atlas-rest, atlas-service, atlas-tenant, atlas-constants, atlas-tracing, gorilla/mux, api2go, go-redis, kafka-go, logrus, ecslogrus, uuid). Remove any require atlas-summons does not actually share with doors only if `go mod tidy` later flags it — do not prune by hand now.
  - In `main.go`, strip the body down to a minimal bootstrap that compiles: logger init, redis connect, tracing init, an empty REST server on base path `/api/` with `/metrics` + readiness, and a `select{}`/signal wait. Remove all summon-specific init (InitIdAllocator/InitRegistry/consumers/tasks) — they're re-added in Task 12. Replace the `serviceName` string with `"atlas-doors"`.

- [ ] **Step 2: Register the service path in `go.work`.** Add the line (keep the list sorted near the other services):
  ```
  	./services/atlas-doors/atlas.com/doors
  ```

- [ ] **Step 3: Add the services.json entry.** In `.github/config/services.json`, add (mirroring the atlas-mounts entry shape):
  ```json
  {
    "name": "atlas-doors",
    "type": "go-service",
    "path": "services/atlas-doors",
    "module_path": "services/atlas-doors/atlas.com/doors",
    "docker_image": "ghcr.io/chronicle20/atlas-doors/atlas-doors",
    "docker_context": "."
  }
  ```

- [ ] **Step 4: Add `atlas-doors` to docker-bake.hcl.** In `docker-bake.hcl`, add `"atlas-doors"` to the hardcoded `go_services = [ ... ]` list (line ~35). HCL cannot read JSON, so both files must list it (memory `reference_docker_bake_hand_synced`).

- [ ] **Step 5: Tidy and build.**
  Run:
  ```bash
  cd services/atlas-doors/atlas.com/doors && go mod tidy && go build ./... && cd -
  ```
  Expected: clean build, an `atlas-doors` binary package compiles.

- [ ] **Step 6: Verify the bake target resolves.**
  Run from the worktree root:
  ```bash
  docker buildx bake atlas-doors --print
  ```
  Expected: prints a target named `atlas-doors` (no "target not found"). A full bake happens in Phase 6.

- [ ] **Step 7: Commit**
  ```bash
  git add services/atlas-doors go.work .github/config/services.json docker-bake.hcl
  git commit -m "feat(atlas-doors): register service + empty module skeleton"
  ```

---

## Phase 1 — Engine domain (version-agnostic)

All Phase-1 code lives under `services/atlas-doors/atlas.com/doors/door/` (and `data/`, `party/`). Pure logic; TDD throughout. Use the project Builder pattern for fixtures — no `*_testhelpers.go`.

### Task 2: Domain model + builder

**Files:**
- Create: `services/atlas-doors/atlas.com/doors/door/model.go`
- Create: `services/atlas-doors/atlas.com/doors/door/builder.go`
- Test: `services/atlas-doors/atlas.com/doors/door/model_test.go`

- [ ] **Step 1: Write the failing test.**
  ```go
  package door

  import (
  	"testing"
  	"time"

  	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
  )

  func TestBuilderBuildAndGetters(t *testing.T) {
  	f := field.NewBuilder(1, 2, 100000000).Build()
  	deploy := time.Unix(1000, 0)
  	m := NewBuilder().
  		SetAreaDoorId(1000001).
  		SetTownDoorId(1000002).
  		SetOwnerCharacterId(42).
  		SetPartyId(7).
  		SetSkillId(2311002).
  		SetSkillLevel(30).
  		SetField(f).
  		SetTownMapId(104000000).
  		SetSlot(3).
  		SetTownPortalId(0x83).
  		SetAreaXY(500, -200).
  		SetTownXY(10, 20).
  		SetDeployTime(deploy).
  		SetExpiresAt(deploy.Add(2 * time.Minute)).
  		Build()

  	if m.AreaDoorId() != 1000001 || m.TownDoorId() != 1000002 {
  		t.Fatalf("oids wrong: %d/%d", m.AreaDoorId(), m.TownDoorId())
  	}
  	if m.PairId() != 1000001 {
  		t.Fatalf("pairId should default to areaDoorId, got %d", m.PairId())
  	}
  	if m.OwnerCharacterId() != 42 || m.PartyId() != 7 {
  		t.Fatalf("owner/party wrong")
  	}
  	if m.Slot() != 3 || m.TownPortalId() != 0x83 {
  		t.Fatalf("slot/portal wrong")
  	}
  	if m.AreaX() != 500 || m.AreaY() != -200 || m.TownX() != 10 || m.TownY() != 20 {
  		t.Fatalf("positions wrong")
  	}
  	if !m.ExpiresAt().Equal(deploy.Add(2 * time.Minute)) {
  		t.Fatalf("expiry wrong")
  	}
  }

  func TestReslotReturnsCopy(t *testing.T) {
  	m := NewBuilder().SetAreaDoorId(1000001).SetSlot(0).SetTownPortalId(0x80).SetTownXY(1, 2).Build()
  	n := m.Reslot(4, 0x84, 99, 88)
  	if m.Slot() != 0 || m.TownPortalId() != 0x80 {
  		t.Fatalf("original mutated")
  	}
  	if n.Slot() != 4 || n.TownPortalId() != 0x84 || n.TownX() != 99 || n.TownY() != 88 {
  		t.Fatalf("reslot copy wrong: slot %d portal %d", n.Slot(), n.TownPortalId())
  	}
  	if n.AreaDoorId() != m.AreaDoorId() {
  		t.Fatalf("reslot dropped identity")
  	}
  }
  ```

- [ ] **Step 2: Run test to verify it fails.**
  Run: `cd services/atlas-doors/atlas.com/doors && go test ./door/ -run TestBuilder -v`
  Expected: FAIL — `undefined: NewBuilder`.

- [ ] **Step 3: Implement model.go.**
  ```go
  package door

  import (
  	"time"

  	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
  	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
  )

  // Model is the immutable representation of a Mystic Door pair (area door in the
  // source field + town door in the return town). Both halves share one record
  // and one pairId so expiry/removal is a single atomic operation.
  type Model struct {
  	areaDoorId       uint32
  	townDoorId       uint32
  	ownerCharacterId uint32
  	partyId          uint32 // 0 = solo
  	skillId          uint32
  	skillLevel       byte
  	fld              field.Model
  	townMapId        _map.Id
  	slot             byte
  	townPortalId     uint32
  	areaX            int16
  	areaY            int16
  	townX            int16
  	townY            int16
  	deployTime       time.Time
  	expiresAt        time.Time
  }

  func (m Model) AreaDoorId() uint32       { return m.areaDoorId }
  func (m Model) TownDoorId() uint32       { return m.townDoorId }
  func (m Model) PairId() uint32           { return m.areaDoorId }
  func (m Model) OwnerCharacterId() uint32 { return m.ownerCharacterId }
  func (m Model) PartyId() uint32          { return m.partyId }
  func (m Model) SkillId() uint32          { return m.skillId }
  func (m Model) SkillLevel() byte         { return m.skillLevel }
  func (m Model) Field() field.Model       { return m.fld }
  func (m Model) TownMapId() _map.Id       { return m.townMapId }
  func (m Model) Slot() byte               { return m.slot }
  func (m Model) TownPortalId() uint32     { return m.townPortalId }
  func (m Model) AreaX() int16             { return m.areaX }
  func (m Model) AreaY() int16             { return m.areaY }
  func (m Model) TownX() int16             { return m.townX }
  func (m Model) TownY() int16             { return m.townY }
  func (m Model) DeployTime() time.Time    { return m.deployTime }
  func (m Model) ExpiresAt() time.Time     { return m.expiresAt }

  // Reslot returns a copy with new town-slot placement. Used by the party
  // membership re-slot path. Identity (oids, owner, field) is preserved.
  func (m Model) Reslot(slot byte, townPortalId uint32, townX, townY int16) Model {
  	n := m
  	n.slot = slot
  	n.townPortalId = townPortalId
  	n.townX = townX
  	n.townY = townY
  	return n
  }
  ```

- [ ] **Step 4: Implement builder.go.**
  ```go
  package door

  import (
  	"time"

  	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
  	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
  )

  type ModelBuilder struct {
  	m Model
  }

  func NewBuilder() *ModelBuilder { return &ModelBuilder{} }

  func Clone(m Model) *ModelBuilder { return &ModelBuilder{m: m} }

  func (b *ModelBuilder) SetAreaDoorId(v uint32) *ModelBuilder       { b.m.areaDoorId = v; return b }
  func (b *ModelBuilder) SetTownDoorId(v uint32) *ModelBuilder       { b.m.townDoorId = v; return b }
  func (b *ModelBuilder) SetOwnerCharacterId(v uint32) *ModelBuilder { b.m.ownerCharacterId = v; return b }
  func (b *ModelBuilder) SetPartyId(v uint32) *ModelBuilder          { b.m.partyId = v; return b }
  func (b *ModelBuilder) SetSkillId(v uint32) *ModelBuilder          { b.m.skillId = v; return b }
  func (b *ModelBuilder) SetSkillLevel(v byte) *ModelBuilder         { b.m.skillLevel = v; return b }
  func (b *ModelBuilder) SetField(v field.Model) *ModelBuilder       { b.m.fld = v; return b }
  func (b *ModelBuilder) SetTownMapId(v _map.Id) *ModelBuilder       { b.m.townMapId = v; return b }
  func (b *ModelBuilder) SetSlot(v byte) *ModelBuilder               { b.m.slot = v; return b }
  func (b *ModelBuilder) SetTownPortalId(v uint32) *ModelBuilder     { b.m.townPortalId = v; return b }
  func (b *ModelBuilder) SetAreaXY(x, y int16) *ModelBuilder         { b.m.areaX = x; b.m.areaY = y; return b }
  func (b *ModelBuilder) SetTownXY(x, y int16) *ModelBuilder         { b.m.townX = x; b.m.townY = y; return b }
  func (b *ModelBuilder) SetDeployTime(v time.Time) *ModelBuilder    { b.m.deployTime = v; return b }
  func (b *ModelBuilder) SetExpiresAt(v time.Time) *ModelBuilder     { b.m.expiresAt = v; return b }

  func (b *ModelBuilder) Build() Model { return b.m }
  ```

- [ ] **Step 5: Run tests to verify they pass.**
  Run: `cd services/atlas-doors/atlas.com/doors && go test ./door/ -run 'TestBuilder|TestReslot' -v`
  Expected: PASS.

- [ ] **Step 6: Commit**
  ```bash
  git add services/atlas-doors/atlas.com/doors/door/model.go services/atlas-doors/atlas.com/doors/door/builder.go services/atlas-doors/atlas.com/doors/door/model_test.go
  git commit -m "feat(atlas-doors): immutable door Model + Builder"
  ```

---

### Task 3: Town slot computation

Pure function: party door slot from the history-sorted member ordering, and the wire town portal id.

**Files:**
- Create: `services/atlas-doors/atlas.com/doors/door/slot.go`
- Test: `services/atlas-doors/atlas.com/doors/door/slot_test.go`

- [ ] **Step 1: Write the failing test.**
  ```go
  package door

  import "testing"

  func TestSlotForOwner(t *testing.T) {
  	members := []uint32{10, 20, 30, 40}
  	if got := SlotForOwner(members, 10); got != 0 {
  		t.Fatalf("leader slot = %d, want 0", got)
  	}
  	if got := SlotForOwner(members, 30); got != 2 {
  		t.Fatalf("third member slot = %d, want 2", got)
  	}
  }

  func TestSlotForOwnerSoloOrMissing(t *testing.T) {
  	if got := SlotForOwner(nil, 99); got != 0 {
  		t.Fatalf("solo slot = %d, want 0", got)
  	}
  	if got := SlotForOwner([]uint32{1, 2}, 99); got != 0 {
  		t.Fatalf("non-member slot = %d, want 0", got)
  	}
  }

  func TestTownPortalId(t *testing.T) {
  	if TownPortalId(0) != 0x80 {
  		t.Fatalf("slot0 portal = %#x, want 0x80", TownPortalId(0))
  	}
  	if TownPortalId(5) != 0x85 {
  		t.Fatalf("slot5 portal = %#x, want 0x85", TownPortalId(5))
  	}
  }
  ```

- [ ] **Step 2: Run test to verify it fails.**
  Run: `cd services/atlas-doors/atlas.com/doors && go test ./door/ -run 'TestSlot|TestTownPortal' -v`
  Expected: FAIL — `undefined: SlotForOwner`.

- [ ] **Step 3: Implement slot.go.**
  ```go
  package door

  // townDoorPortalBase is the wire portal id the client expects for the first
  // party door slot (Cosmic MapleMap.getDoorPortal: portals.get(0x80 + slot)).
  const townDoorPortalBase uint32 = 0x80

  // MaxPartySlots caps door slots at the MapleStory party size (6 → slots 0..5,
  // portals 0x80..0x85), guaranteeing no intra-party town-door overlap.
  const MaxPartySlots = 6

  // SlotForOwner returns the owner's 0-based index in the history-sorted member
  // list (Cosmic Party.getPartyDoor). A solo caster, or an owner not found in the
  // list, takes slot 0.
  func SlotForOwner(orderedMemberIds []uint32, ownerCharacterId uint32) byte {
  	for i, id := range orderedMemberIds {
  		if id == ownerCharacterId {
  			return byte(i)
  		}
  	}
  	return 0
  }

  // TownPortalId maps a party door slot to the wire portal id the client expects.
  func TownPortalId(slot byte) uint32 {
  	return townDoorPortalBase + uint32(slot)
  }
  ```

- [ ] **Step 4: Run tests to verify they pass.**
  Run: `cd services/atlas-doors/atlas.com/doors && go test ./door/ -run 'TestSlot|TestTownPortal' -v`
  Expected: PASS.

- [ ] **Step 5: Commit**
  ```bash
  git add services/atlas-doors/atlas.com/doors/door/slot.go services/atlas-doors/atlas.com/doors/door/slot_test.go
  git commit -m "feat(atlas-doors): party town-slot + wire portal-id computation"
  ```

---

### Task 4: Object-id allocator (two allocations per door)

Wrap `libs/atlas-object-id`, exactly like `summon/id_allocator.go`, but expose an `AllocatePair` that fails (no MinId fallback) on error.

**Files:**
- Create: `services/atlas-doors/atlas.com/doors/door/id_allocator.go` (adapt from atlas-summons `id_allocator.go`)
- Test: `services/atlas-doors/atlas.com/doors/door/id_allocator_test.go`

- [ ] **Step 1: Copy and adapt.** Copy `summon/id_allocator.go` → `door/id_allocator.go`. Rename the type to `IdAllocator`, package to `door`, and the singleton funcs to `InitIdAllocator(rc)` / `GetIdAllocator()`. Keep `Allocate(ctx, t) (uint32, error)` and `Release(ctx, t, id)` but change the error policy: **propagate** the underlying allocator error instead of returning `MinId`. Add:
  ```go
  // AllocatePair allocates two distinct object ids (area + town). On any failure
  // it releases whatever it already took and returns the error — the spawn must
  // fail cleanly rather than collide on a MinId fallback.
  func (a *IdAllocator) AllocatePair(ctx context.Context, t tenant.Model) (uint32, uint32, error) {
  	area, err := a.Allocate(ctx, t)
  	if err != nil {
  		return 0, 0, err
  	}
  	town, err := a.Allocate(ctx, t)
  	if err != nil {
  		a.Release(ctx, t, area)
  		return 0, 0, err
  	}
  	return area, town, nil
  }
  ```
  (If atlas-summons' `Allocate` signature returns only `uint32`, change it here to return `(uint32, error)` and surface the inner error.)

- [ ] **Step 2: Write a test that AllocatePair returns two distinct ids ≥ MinId.** Use a miniredis or the same test harness atlas-summons' allocator test uses (copy `id_allocator_test.go` and adapt). If atlas-summons has no allocator test, write one with `miniredis`:
  ```go
  package door

  import (
  	"context"
  	"testing"

  	objectid "github.com/Chronicle20/atlas/libs/atlas-object-id"
  	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
  	"github.com/alicebob/miniredis/v2"
  	goredis "github.com/redis/go-redis/v9"
  	"github.com/google/uuid"
  )

  func TestAllocatePairDistinct(t *testing.T) {
  	mr, _ := miniredis.Run()
  	defer mr.Close()
  	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
  	InitIdAllocator(rc)
  	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
  	ctx := tenant.WithContext(context.Background(), te)
  	a, b, err := GetIdAllocator().AllocatePair(ctx, te)
  	if err != nil {
  		t.Fatalf("alloc err: %v", err)
  	}
  	if a == b {
  		t.Fatalf("ids not distinct: %d", a)
  	}
  	if a < objectid.MinId || b < objectid.MinId {
  		t.Fatalf("ids below MinId: %d %d", a, b)
  	}
  }
  ```
  (Match the exact `tenant.Create` / `tenant.WithContext` signatures used in atlas-summons tests; adjust if the constructor differs.)

- [ ] **Step 3: Run test, expect FAIL** (`undefined: AllocatePair`), then implement, then re-run.
  Run: `cd services/atlas-doors/atlas.com/doors && go test ./door/ -run TestAllocatePair -v`
  Expected after impl: PASS.

- [ ] **Step 4: Commit**
  ```bash
  git add services/atlas-doors/atlas.com/doors/door/id_allocator.go services/atlas-doors/atlas.com/doors/door/id_allocator_test.go
  git commit -m "feat(atlas-doors): object-id allocator with fail-fast AllocatePair"
  ```

---

### Task 5: Redis registry + indices

Mirror `summon/registry.go`. Four key spaces: primary record, field index, owner index, town+party slot index. All via `libs/atlas-redis` (rediskeyguard-clean).

**Files:**
- Create: `services/atlas-doors/atlas.com/doors/door/registry.go` (adapt from atlas-summons `registry.go`)
- Test: `services/atlas-doors/atlas.com/doors/door/registry_test.go`

- [ ] **Step 1: Copy and adapt the storedDoor + Registry.** Copy `summon/registry.go` → `door/registry.go`. Define `storedDoor` (all Model fields; `field.Model` flattened to world/channel/map/instance; times as unix-milli). Keep the singleton pattern (`InitRegistry(rc)`/`GetRegistry()`), the `atlasredis.Registry[string, storedDoor]` primary store, and `atlasredis.KeyedSet[string]` indices. Use these key formats:
  ```
  door:{tenant}:{areaDoorId}                               (primary, via atlasredis.Registry)
  door-field:{tenant}:{world}:{channel}:{map}:{instance}   (field index set)
  door-owner:{tenant}:{characterId}                        (owner index set)
  door-town:{tenant}:{world}:{channel}:{townMap}:{scope}   (town+party slot index set)
  ```
  where `{scope}` = `partyId` when `partyId != 0`, else `solo-{ownerCharacterId}` (per-party slot isolation; solo casters namespaced by owner — design §4.3).

- [ ] **Step 2: Implement the Registry methods** (signatures mirror summons):
  ```go
  func (r *Registry) Put(ctx context.Context, t tenant.Model, m Model) error
  func (r *Registry) Get(ctx context.Context, t tenant.Model, areaDoorId uint32) (Model, error)
  func (r *Registry) GetInField(ctx context.Context, t tenant.Model, f field.Model) ([]Model, error)
  func (r *Registry) GetByOwner(ctx context.Context, t tenant.Model, characterId uint32) ([]Model, error)
  func (r *Registry) GetInTown(ctx context.Context, t tenant.Model, worldId world.Id, channelId channel.Id, townMapId _map.Id, scope string) ([]Model, error)
  func (r *Registry) Remove(ctx context.Context, t tenant.Model, areaDoorId uint32) error
  func (r *Registry) GetAll(ctx context.Context) (map[tenant.Model][]Model, error)
  ```
  `Put` adds the record + inserts into all three index sets. `Remove` deletes the record + removes from all three sets. Add a helper `townScope(m Model) string` returning `partyId` or `solo-{owner}`.

- [ ] **Step 3: Write the failing test** (use miniredis):
  ```go
  func TestRegistryIndices(t *testing.T) {
  	// Put a door; assert GetInField, GetByOwner, GetInTown all return it;
  	// Remove; assert all three are now empty.
  }

  func TestRegistryPerPartySlotIsolation(t *testing.T) {
  	// Two doors, same townMap+world+channel, DIFFERENT partyId, both slot 0 /
  	// townPortalId 0x80. GetInTown(partyA) returns only A; GetInTown(partyB)
  	// returns only B. Both legitimately occupy portal 0x80.
  }

  func TestRegistrySoloNonCollision(t *testing.T) {
  	// Two solo doors (partyId 0), different owners, same town, both slot 0.
  	// GetInTown(scope solo-owner1) returns only owner1's door.
  }
  ```
  Build fixtures with `NewBuilder()`. Fill in the assertions concretely.

- [ ] **Step 4: Run, expect FAIL, implement, re-run.**
  Run: `cd services/atlas-doors/atlas.com/doors && go test ./door/ -run TestRegistry -v`
  Expected after impl: PASS.

- [ ] **Step 5: Verify redis-key-guard stays clean.**
  Run from the worktree root:
  ```bash
  GOWORK=off tools/redis-key-guard.sh
  ```
  Expected: no findings (all keyed access goes through `libs/atlas-redis`).

- [ ] **Step 6: Commit**
  ```bash
  git add services/atlas-doors/atlas.com/doors/door/registry.go services/atlas-doors/atlas.com/doors/door/registry_test.go
  git commit -m "feat(atlas-doors): Redis registry with field/owner/town-party indices"
  ```

---

### Task 6: atlas-data map client (return town + door portals)

A REST client to atlas-data for the source map's return/forced-return/town/field-limit and the town map's door portals. Mirror the atlas-summons `data/skill` client structure (requests.Provider + Extract).

**Files:**
- Create: `services/atlas-doors/atlas.com/doors/data/map/model.go`
- Create: `services/atlas-doors/atlas.com/doors/data/map/rest.go` (RestModel + Extract)
- Create: `services/atlas-doors/atlas.com/doors/data/map/requests.go`
- Create: `services/atlas-doors/atlas.com/doors/data/map/processor.go`
- Test: `services/atlas-doors/atlas.com/doors/data/map/processor_test.go`

- [ ] **Step 1: Define the Model + RestModel.** Model exposes the fields the engine needs:
  ```go
  // model.go
  package _map

  import mapconst "github.com/Chronicle20/atlas/libs/atlas-constants/map"

  type Model struct {
  	id                mapconst.Id
  	town              bool
  	returnMapId       mapconst.Id
  	forcedReturnMapId mapconst.Id
  	fieldLimit        uint32
  	doorPortals       []Portal // Type==6, in load order
  }
  func (m Model) Id() mapconst.Id                { return m.id }
  func (m Model) Town() bool                     { return m.town }
  func (m Model) ReturnMapId() mapconst.Id       { return m.returnMapId }
  func (m Model) ForcedReturnMapId() mapconst.Id { return m.forcedReturnMapId }
  func (m Model) FieldLimit() uint32             { return m.fieldLimit }
  func (m Model) DoorPortals() []Portal          { return m.doorPortals }

  type Portal struct {
  	Id   uint32
  	Type uint8
  	X    int16
  	Y    int16
  }
  ```
  The RestModel mirrors `services/atlas-channel/atlas.com/channel/data/map/rest.go` (fields `returnMapId`, `forcedReturnMapId`, `fieldLimit`, `town`) plus a separate portals fetch. Door portals come from `GET /data/maps/{mapId}/portals` filtered to `type == 6`.

- [ ] **Step 2: requests.go** — two requests:
  ```go
  func requestMap(mapId mapconst.Id) requests.Request[RestModel]          // GET {DATA}/data/maps/{mapId}
  func requestPortals(mapId mapconst.Id) requests.Request[[]PortalRestModel] // GET {DATA}/data/maps/{mapId}/portals
  ```
  Base URL from the same env the atlas-summons `data/skill` client uses (`requests.RootUrl("DATA")` or equivalent — match the atlas-summons pattern exactly).

- [ ] **Step 3: processor.go** — `Processor` interface + Impl:
  ```go
  type Processor interface {
  	GetById(mapId mapconst.Id) (Model, error)            // map metadata only
  	GetDoorPortals(mapId mapconst.Id) ([]Portal, error)  // Type==6 in load order
  }
  func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor
  ```
  `GetDoorPortals` fetches portals, filters `Type == PortalTypeDoor (6)`, preserves order.

- [ ] **Step 4: Write the failing test.** Use an `httptest.Server` returning a fixed JSON:API map + portals payload (copy the JSON:API envelope shape from a real atlas-data response; door portals as `type: "portals"` with `attributes.type == 6`). Assert `GetById` returns the right return/forced/town/fieldLimit, and `GetDoorPortals` returns only the 6-type portals in order. Inject the base URL via the env var the requests use.
  ```go
  func TestGetByIdParsesReturnAndTownAndLimit(t *testing.T) { /* ... */ }
  func TestGetDoorPortalsFiltersTypeSixInOrder(t *testing.T) { /* ... */ }
  ```

- [ ] **Step 5: Run, expect FAIL, implement, re-run.**
  Run: `cd services/atlas-doors/atlas.com/doors && go test ./data/map/ -v`
  Expected after impl: PASS.

- [ ] **Step 6: Commit**
  ```bash
  git add services/atlas-doors/atlas.com/doors/data/map
  git commit -m "feat(atlas-doors): atlas-data map client (return town + door portals)"
  ```

---

### Task 7: atlas-data skill client (duration by level)

Mirror the atlas-summons `data/skill` client; expose duration (and confirm MP/item cost are read-only here — cost is consumed channel-side, OQ-1).

**Files:**
- Create: `services/atlas-doors/atlas.com/doors/data/skill/{model.go,rest.go,requests.go,processor.go}` (copy from atlas-summons `data/skill`, prune to what doors needs)
- Test: `services/atlas-doors/atlas.com/doors/data/skill/processor_test.go`

- [ ] **Step 1: Copy the atlas-summons `data/skill` package** into `services/atlas-doors/.../data/skill/` and rename the Go package import path references to `atlas-doors/data/skill`. Keep the effect Model with at least `Duration() int32`. Keep `GetEffect(skillId uint32, level byte) (effect.Model, error)`.

- [ ] **Step 2: Write a failing test** that, via `httptest.Server`, asserts `GetEffect(2311002, 30).Duration()` returns the value from the served payload (pick a representative number, e.g. duration in ms). Reuse the atlas-summons skill-client test as a template if present.

- [ ] **Step 3: Run, expect FAIL, implement/adjust, re-run.**
  Run: `cd services/atlas-doors/atlas.com/doors && go test ./data/skill/ -v`
  Expected after impl: PASS.

- [ ] **Step 4: Commit**
  ```bash
  git add services/atlas-doors/atlas.com/doors/data/skill
  git commit -m "feat(atlas-doors): atlas-data skill client (door duration by level)"
  ```

---

### Task 8: atlas-parties client (history-sorted members)

Read the owner's party id + the history-sorted member id ordering (for slot assignment). Confirm against atlas-parties whether members are already returned in stable history order; if not, sort by the join/history field the API exposes.

**Files:**
- Create: `services/atlas-doors/atlas.com/doors/party/{model.go,rest.go,requests.go,processor.go}`
- Test: `services/atlas-doors/atlas.com/doors/party/processor_test.go`

- [ ] **Step 1: Inspect the atlas-parties REST contract.** Read `services/atlas-parties/atlas.com/parties/` (resource + rest model) to find the party-by-member endpoint and the member ordering field. Note in the processor doc comment whether the API returns members in history order natively or whether we sort.

- [ ] **Step 2: Define the client.**
  ```go
  type Processor interface {
  	// GetByMember returns the partyId (0 if not in a party) and the
  	// history-sorted member ids (Cosmic Party order). Solo → (0, nil).
  	GetByMember(characterId uint32) (partyId uint32, orderedMemberIds []uint32, err error)
  }
  func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor
  ```
  Mirror the channel-side party read pattern only for structure; this is a fresh atlas-doors client.

- [ ] **Step 3: Write the failing test** with `httptest.Server` returning a party with 3 members in a known order; assert `GetByMember` returns the partyId and the ordered ids; assert a not-in-party member returns `(0, nil, nil)`.

- [ ] **Step 4: Run, expect FAIL, implement, re-run.**
  Run: `cd services/atlas-doors/atlas.com/doors && go test ./party/ -v`
  Expected after impl: PASS.

- [ ] **Step 5: Commit**
  ```bash
  git add services/atlas-doors/atlas.com/doors/party
  git commit -m "feat(atlas-doors): atlas-parties client (history-sorted members)"
  ```

---

### Task 9: Kafka envelopes (commands + events)

Define the command/event envelope types and topic env constants before the processor (the processor emits them).

**Files:**
- Create: `services/atlas-doors/atlas.com/doors/door/kafka.go`

- [ ] **Step 1: Implement kafka.go.** Mirror the atlas-summons `kafka.go` envelope shape (`Command[E]` / `StatusEvent[E]` with tenant/span via headers, body typed by `Type`). Topic env consts and types:
  ```go
  package door

  import (
  	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
  	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
  	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
  	"github.com/google/uuid"
  )

  const (
  	EnvCommandTopicDoor       = "COMMAND_TOPIC_DOOR"
  	EnvEventTopicDoorStatus   = "EVENT_TOPIC_DOOR_STATUS"
  )

  // Command types
  const (
  	CommandTypeSpawn  = "SPAWN"
  	CommandTypeRemove = "REMOVE"
  )

  type Command[E any] struct {
  	WorldId   world.Id   `json:"worldId"`
  	ChannelId channel.Id `json:"channelId"`
  	MapId     _map.Id    `json:"mapId"`
  	Instance  uuid.UUID  `json:"instance"`
  	Type      string     `json:"type"`
  	Body      E          `json:"body"`
  }

  type SpawnCommandBody struct {
  	OwnerCharacterId uint32 `json:"ownerCharacterId"`
  	X                int16  `json:"x"`
  	Y                int16  `json:"y"`
  	SkillId          uint32 `json:"skillId"`
  	SkillLevel       byte   `json:"skillLevel"`
  }

  type RemoveCommandBody struct {
  	OwnerCharacterId uint32 `json:"ownerCharacterId"`
  	Reason           string `json:"reason"`
  }

  // Event types
  const (
  	EventTypeCreated     = "CREATED"
  	EventTypeRemoved     = "REMOVED"
  	EventTypeSlotChanged = "SLOT_CHANGED"
  )

  // Removal reasons
  const (
  	ReasonExpiry         = "EXPIRY"
  	ReasonDisconnect     = "DISCONNECT"
  	ReasonChannelChanged = "CHANNEL_CHANGED"
  	ReasonLeftField      = "LEFT_FIELD"
  	ReasonRecast         = "RECAST"
  )

  type StatusEvent[E any] struct {
  	WorldId          world.Id   `json:"worldId"`
  	ChannelId        channel.Id `json:"channelId"`
  	MapId            _map.Id    `json:"mapId"` // area field map; CreateKey on this
  	Instance         uuid.UUID  `json:"instance"`
  	PairId           uint32     `json:"pairId"`
  	OwnerCharacterId uint32     `json:"ownerCharacterId"`
  	PartyId          uint32     `json:"partyId"`
  	Type             string     `json:"type"`
  	Body             E          `json:"body"`
  }

  type CreatedBody struct {
  	TownMapId    _map.Id `json:"townMapId"`
  	Slot         byte    `json:"slot"`
  	AreaDoorId   uint32  `json:"areaDoorId"`
  	TownDoorId   uint32  `json:"townDoorId"`
  	TownPortalId uint32  `json:"townPortalId"`
  	AreaX        int16   `json:"areaX"`
  	AreaY        int16   `json:"areaY"`
  	TownX        int16   `json:"townX"`
  	TownY        int16   `json:"townY"`
  	ExpiresAt    int64   `json:"expiresAt"` // unix-milli
  }

  type RemovedBody struct {
  	TownMapId  _map.Id `json:"townMapId"`
  	AreaDoorId uint32  `json:"areaDoorId"`
  	TownDoorId uint32  `json:"townDoorId"`
  	Slot       byte    `json:"slot"`
  	Reason     string  `json:"reason"`
  }

  type SlotChangedBody struct {
  	TownMapId    _map.Id `json:"townMapId"`
  	OldSlot      byte    `json:"oldSlot"`
  	NewSlot      byte    `json:"newSlot"`
  	TownPortalId uint32  `json:"townPortalId"`
  	TownDoorId   uint32  `json:"townDoorId"`
  	TownX        int16   `json:"townX"`
  	TownY        int16   `json:"townY"`
  }
  ```
  (Match field tag/casing conventions to atlas-summons' `kafka.go` exactly — copy its header/CreateKey helper functions and adapt.)

- [ ] **Step 2: Build.**
  Run: `cd services/atlas-doors/atlas.com/doors && go build ./door/`
  Expected: clean.

- [ ] **Step 3: Commit**
  ```bash
  git add services/atlas-doors/atlas.com/doors/door/kafka.go
  git commit -m "feat(atlas-doors): Kafka command/event envelopes + topic consts"
  ```

---

### Task 10: Event producers

Provider helpers that turn a `door.Model` into `[]kafka.Message` for CREATED/REMOVED/SLOT_CHANGED. Mirror atlas-summons `producer.go`.

**Files:**
- Create: `services/atlas-doors/atlas.com/doors/door/producer.go`
- Test: `services/atlas-doors/atlas.com/doors/door/producer_test.go`

- [ ] **Step 1: Implement the providers** (mirror atlas-summons signature `model.Provider[[]kafka.Message]`):
  ```go
  func createdEventProvider(m Model) model.Provider[[]kafka.Message]
  func removedEventProvider(m Model, reason string) model.Provider[[]kafka.Message]
  func slotChangedEventProvider(m Model, oldSlot byte) model.Provider[[]kafka.Message]
  ```
  Each builds a `StatusEvent[...]` keyed by the area field map id (`CreateKey`), with the body filled from `m`. Use the same `producer.SingleMessageProvider`/key helper atlas-summons uses.

- [ ] **Step 2: Write a test** asserting `createdEventProvider(m)()` yields one message whose key encodes the area map id and whose decoded value has `Type == EventTypeCreated` and the expected body fields. (Decode the message value JSON back into `StatusEvent[CreatedBody]`.)

- [ ] **Step 3: Run, expect FAIL, implement, re-run.**
  Run: `cd services/atlas-doors/atlas.com/doors && go test ./door/ -run TestProducer -v`
  Expected after impl: PASS.

- [ ] **Step 4: Commit**
  ```bash
  git add services/atlas-doors/atlas.com/doors/door/producer.go services/atlas-doors/atlas.com/doors/door/producer_test.go
  git commit -m "feat(atlas-doors): door status event producers"
  ```

---

### Task 11: Processor — Spawn / Remove / Reslot / cleanup

The engine core. Pure orchestration over registry + data clients + party client + id allocator + producers, with injectable seams for tests (mirror atlas-summons' ProcessorImpl fields).

**Files:**
- Create: `services/atlas-doors/atlas.com/doors/door/processor.go`
- Test: `services/atlas-doors/atlas.com/doors/door/processor_test.go`

- [ ] **Step 1: Define the interface + Impl.**
  ```go
  type Processor interface {
  	GetById(areaDoorId uint32) (Model, error)
  	GetInField(f field.Model) ([]Model, error)
  	Spawn(f field.Model, ownerCharacterId, skillId uint32, skillLevel byte, x, y int16) (Model, error)
  	RemoveByOwner(ownerCharacterId uint32, reason string) error
  	RemoveAllForOwnerLeavingField(ownerCharacterId uint32, newField field.Model) error
  	ReslotForParty(partyId uint32) error
  	ReslotOwnerToSolo(ownerCharacterId uint32) error
  }
  func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor
  ```
  ProcessorImpl carries `l, ctx, t tenant.Model, emit emitter` plus injectable seams: `mapData mapSource`, `skillData skillSource`, `partyData partySource`, `clock func() time.Time`. Provide a production `NewProcessor` that wires the real clients and a test constructor that accepts the seams (unexported struct literal in tests — no `*_testhelpers.go`).

- [ ] **Step 2: Implement `Spawn` (the heart):**
  1. Resolve party: `partyId, ordered, err := partyData.GetByMember(owner)`.
  2. **Recast replace:** `RemoveByOwner(owner, ReasonRecast)` first (best-effort; ignore not-found).
  3. Resolve source map metadata: `srcMap, _ := mapData.GetById(f.MapId())`. Reject (return a sentinel `ErrIneligible`) if `srcMap.Town()` or no valid return map. (Field-limit/town are also gated channel-side, Task 17 — re-check here defensively.)
  4. `townMapId` = `ForcedReturnMapId` if it resolves to a real map, else `ReturnMapId`.
  5. `slot := SlotForOwner(ordered, owner)` (solo → 0). If `slot >= MaxPartySlots` reject (no slot).
  6. Resolve town door portals: `portals, _ := mapData.GetDoorPortals(townMapId)`. Pick `townX/townY`:
     - if `int(slot) < len(portals)` → `portals[slot].X/Y`;
     - else **fallback** (OQ-3): use `portals[0]` if any, else a default near the town spawn (document the chosen default in code per Task 0 findings).
  7. `townPortalId := TownPortalId(slot)` (wire id `0x80+slot` regardless of atlas-data's internal id).
  8. Duration: `eff, _ := skillData.GetEffect(skillId, skillLevel)`; `now := clock()`; `expiresAt := now.Add(time.Duration(eff.Duration()) * time.Millisecond)`.
  9. Allocate: `area, town, err := GetIdAllocator().AllocatePair(ctx, t)`; on error, emit nothing and return err (clean no-op).
  10. Build the Model, `GetRegistry().Put(...)`, `emit(EnvEventTopicDoorStatus, createdEventProvider(m))`, return m.

- [ ] **Step 3: Implement `RemoveByOwner`:** look up `GetByOwner`; for the door (there is at most one), `Release` both oids, `Registry.Remove`, emit `removedEventProvider(m, reason)`. The deploy-grace (FR-6.3) is honored on the **broadcast** side via the channel consumer (Task 19) — but the engine still records `deployTime`; if `reason == ReasonRecast` and `now-deployTime < grace`, the channel must defer the *remove broadcast* (note this in the event so the channel can delay). Simplest: include `deployTime` (unix-milli) in `RemovedBody` is unnecessary — instead the channel delays based on its own CREATED bookkeeping. Keep the engine simple: remove immediately, emit REMOVED.
  > Design note: the grace delay is a **client-crash guard on the spawn→remove broadcast sequence**, owned by the channel consumer (Task 19). The engine removes state synchronously.

- [ ] **Step 4: Implement `RemoveAllForOwnerLeavingField`:** for the owner's door, remove **only if** `newField` is neither the door's source field nor (the door's town map on the same world/channel). Walking into the town the door spans is a warp, not abandonment (design §5.3). Otherwise no-op.

- [ ] **Step 5: Implement `ReslotForParty`:** recompute every party member's slot from fresh `partyData` ordering; for each owner whose door's slot changed, `Reslot` the Model, re-resolve town portal position, `Put`, emit `slotChangedEventProvider(m, oldSlot)`.

- [ ] **Step 6: Implement `ReslotOwnerToSolo`:** set the owner's door to solo scope, slot 0, portal `0x80`, town position from `portals[0]`; `Put`; emit `slotChangedEventProvider(m, oldSlot)`.

- [ ] **Step 7: Write the failing tests** (table-driven where natural; inject seams returning fixed data; capture emitted events with a fake emitter):
  ```go
  func TestSpawnSoloSlotZero(t *testing.T) { /* slot 0, portal 0x80, CREATED emitted */ }
  func TestSpawnPartyMemberSlotIndex(t *testing.T) { /* owner is 3rd → slot 2, portal 0x82 */ }
  func TestSpawnRejectsTownMap(t *testing.T) { /* srcMap.Town()==true → ErrIneligible, no event */ }
  func TestSpawnRejectsNoReturnMap(t *testing.T) { /* returnMap==0 → ErrIneligible */ }
  func TestSpawnUsesForcedReturnWhenSet(t *testing.T) { /* townMapId == forced */ }
  func TestSpawnAllocFailureNoEvent(t *testing.T) { /* allocator errors → err, zero events */ }
  func TestSpawnRecastReplacesPrior(t *testing.T) { /* prior owner door removed (REMOVED reason RECAST) before CREATED */ }
  func TestRemoveByOwnerReleasesAndEmits(t *testing.T) {}
  func TestLeaveFieldRemovesOnlyWhenLeavingSource(t *testing.T) { /* warp to town map → no remove; warp elsewhere → remove */ }
  func TestReslotForPartyEmitsSlotChangedForMovedDoors(t *testing.T) {}
  func TestReslotOwnerToSoloResetsSlotZero(t *testing.T) {}
  ```
  Use miniredis for the registry and fakes for map/skill/party/emitter/clock.

- [ ] **Step 8: Run, expect FAIL, implement, re-run.**
  Run: `cd services/atlas-doors/atlas.com/doors && go test ./door/ -run 'TestSpawn|TestRemove|TestLeave|TestReslot' -v`
  Expected after impl: PASS. Then `go test -race ./door/`.

- [ ] **Step 9: Commit**
  ```bash
  git add services/atlas-doors/atlas.com/doors/door/processor.go services/atlas-doors/atlas.com/doors/door/processor_test.go
  git commit -m "feat(atlas-doors): door processor (spawn/remove/reslot/cleanup)"
  ```

---

## Phase 2 — Engine wiring (Kafka, REST, leader task, main)

### Task 12: Command consumer (SPAWN / REMOVE)

**Files:**
- Create: `services/atlas-doors/atlas.com/doors/kafka/consumer/consumer.go` (copy from atlas-summons — NewConfig curry + LookupBrokers)
- Create: `services/atlas-doors/atlas.com/doors/kafka/consumer/door/consumer.go`
- Test: `services/atlas-doors/atlas.com/doors/kafka/consumer/door/consumer_test.go`

- [ ] **Step 1: Copy the shared `kafka/consumer/consumer.go`** from atlas-summons (rename package import paths to `atlas-doors/...`).

- [ ] **Step 2: Implement the door command consumer** mirroring atlas-summons' `kafka/consumer/summon/consumer.go`:
  - `InitConsumers(l)` registers consumer name `"door_command"`, topic env `door.EnvCommandTopicDoor`, header parsers `SpanHeaderParser, TenantHeaderParser`.
  - `InitHandlers(l)` registers two handlers on the topic: `handleSpawnCommand` (`Command[SpawnCommandBody]`) → `door.NewProcessor(l, ctx).Spawn(...)`; `handleRemoveCommand` (`Command[RemoveCommandBody]`) → `RemoveByOwner(...)`.
  - The handler builds `field.Model` from the command's world/channel/map/instance.

- [ ] **Step 3: Write a test** that feeds a decoded `Command[SpawnCommandBody]` into `handleSpawnCommand` (with an injected processor seam) and asserts `Spawn` is called with the right args. (Mirror the atlas-summons command-consumer test if one exists; otherwise test the body→processor mapping directly.)

- [ ] **Step 4: Run, expect FAIL, implement, re-run.**
  Run: `cd services/atlas-doors/atlas.com/doors && go test ./kafka/consumer/door/ -v`
  Expected after impl: PASS.

- [ ] **Step 5: Commit**
  ```bash
  git add services/atlas-doors/atlas.com/doors/kafka/consumer/consumer.go services/atlas-doors/atlas.com/doors/kafka/consumer/door
  git commit -m "feat(atlas-doors): COMMAND_TOPIC_DOOR consumer (SPAWN/REMOVE)"
  ```

---

### Task 13: Character-status consumer (cleanup)

**Files:**
- Create: `services/atlas-doors/atlas.com/doors/kafka/consumer/character/consumer.go` (adapt from atlas-summons `kafka/consumer/character/`)
- Test: `services/atlas-doors/atlas.com/doors/kafka/consumer/character/consumer_test.go`

- [ ] **Step 1: Adapt the atlas-summons character-status consumer.** Topic env `EVENT_TOPIC_CHARACTER_STATUS`, header parsers Span/Tenant. Three handlers:
  - `handleLogout` → `RemoveByOwner(charId, ReasonDisconnect)`.
  - `handleChannelChanged` → `RemoveByOwner(charId, ReasonChannelChanged)`.
  - `handleMapChanged` → `RemoveAllForOwnerLeavingField(charId, newField)` where `newField` is built from the event's target world/channel/map/instance. (This is the only behavioral delta from summons, which despawns on every map change — doors persist when the owner warps into their own town.)

- [ ] **Step 2: Write tests:** logout/channel-change always remove; map-change removes only when leaving the source field (reuse the processor's leave-field logic via an injected processor seam — assert the right method is called with the right field).

- [ ] **Step 3: Run, expect FAIL, implement, re-run.**
  Run: `cd services/atlas-doors/atlas.com/doors && go test ./kafka/consumer/character/ -v`
  Expected after impl: PASS.

- [ ] **Step 4: Commit**
  ```bash
  git add services/atlas-doors/atlas.com/doors/kafka/consumer/character
  git commit -m "feat(atlas-doors): character-status consumer (door cleanup)"
  ```

---

### Task 14: Party-status consumer (re-slot)

**Files:**
- Create: `services/atlas-doors/atlas.com/doors/kafka/consumer/party/consumer.go`
- Test: `services/atlas-doors/atlas.com/doors/kafka/consumer/party/consumer_test.go`

- [ ] **Step 1: Inspect the atlas-parties event contract.** Read the atlas-parties producer / status-event types to find the membership-change event (`EVENT_TOPIC_PARTY_STATUS`) and its body shape (joined/left/leader/disband + the affected character + party id). Use those types; do not invent new ones.

- [ ] **Step 2: Implement the consumer.** Header parsers Span/Tenant. On any membership change for a party that may have live doors:
  - join → `ReslotForParty(partyId)` (the joiner's own door, if any, is re-slotted; existing members re-slotted).
  - leave → `ReslotOwnerToSolo(leaverCharId)` then `ReslotForParty(partyId)` (remaining members).
  - leader change → `ReslotForParty(partyId)`.
  - disband → for each former member with a door, `ReslotOwnerToSolo`.
  > Visibility revocation for a leaver (sending `removeDoor` to that one session) is a **channel** concern (Task 19/20), driven by the party event the channel already consumes — not the engine. The engine only owns slot state.

- [ ] **Step 3: Write tests** with an injected processor seam asserting the right reslot calls per event type.

- [ ] **Step 4: Run, expect FAIL, implement, re-run.**
  Run: `cd services/atlas-doors/atlas.com/doors && go test ./kafka/consumer/party/ -v`
  Expected after impl: PASS.

- [ ] **Step 5: Commit**
  ```bash
  git add services/atlas-doors/atlas.com/doors/kafka/consumer/party
  git commit -m "feat(atlas-doors): party-status consumer (door re-slotting)"
  ```

---

### Task 15: Leader-elected expiry sweep + tasks + leaderconfig

**Files:**
- Create: `services/atlas-doors/atlas.com/doors/leaderconfig.go` (copy from atlas-summons, rename SUMMON→DOOR)
- Create: `services/atlas-doors/atlas.com/doors/tasks/task.go` (copy verbatim from atlas-summons)
- Create: `services/atlas-doors/atlas.com/doors/door/expiry_task.go` (adapt from atlas-summons `expiry_task.go`)
- Test: `services/atlas-doors/atlas.com/doors/door/expiry_task_test.go`

- [ ] **Step 1: Copy `tasks/task.go` verbatim** (Task interface + Register goroutine loop).

- [ ] **Step 2: Copy `leaderconfig.go`**, renaming env vars: `DOOR_LEADER_ELECTION_ENABLED`, `DOOR_LEADER_TTL`, `DOOR_LEADER_REFRESH`, `DOOR_LEADER_BACKOFF` (same defaults as summons).

- [ ] **Step 3: Adapt `expiry_task.go`.** `NewExpiryTask(l, ctx, time.Second)`; `Run()` enumerates `GetRegistry().GetAll()` grouped by tenant; for each door where `now.After(expiresAt)`, call `NewProcessor(l, tenantCtx).RemoveByOwner(owner, ReasonExpiry)` (or a direct `Registry.Remove` + emit — match how summons' expiry removes). This is also the orphan backstop.

- [ ] **Step 4: Write a test** with two doors (one expired, one not) and a fixed clock; assert `Run()` removes only the expired one and emits one REMOVED with reason EXPIRY.

- [ ] **Step 5: Run, expect FAIL, implement, re-run.**
  Run: `cd services/atlas-doors/atlas.com/doors && go test ./door/ -run TestExpiry -v`
  Expected after impl: PASS.

- [ ] **Step 6: Commit**
  ```bash
  git add services/atlas-doors/atlas.com/doors/leaderconfig.go services/atlas-doors/atlas.com/doors/tasks services/atlas-doors/atlas.com/doors/door/expiry_task.go services/atlas-doors/atlas.com/doors/door/expiry_task_test.go
  git commit -m "feat(atlas-doors): leader-elected expiry sweep + tasks + leaderconfig"
  ```

---

### Task 16: REST (GET door, GET doors in field) + main.go wiring

**Files:**
- Create: `services/atlas-doors/atlas.com/doors/door/resource.go` (RestModel + Transform)
- Create: `services/atlas-doors/atlas.com/doors/door/rest.go` (GET /doors/{doorId})
- Create: `services/atlas-doors/atlas.com/doors/world/resource.go` (GET .../maps/{m}/instances/{i}/doors)
- Create: `services/atlas-doors/atlas.com/doors/rest/handler.go` (type aliases + ParseDoorId etc., copy from atlas-summons)
- Modify: `services/atlas-doors/atlas.com/doors/main.go`
- Test: `services/atlas-doors/atlas.com/doors/door/resource_test.go`

- [ ] **Step 1: Copy `rest/handler.go`** from atlas-summons; keep the parse helpers needed (`ParseWorldId`, `ParseChannelId`, `ParseMapId`, `ParseInstanceId`) and add `ParseDoorId`.

- [ ] **Step 2: Implement `resource.go`** — `RestModel` with `GetName() string { return "doors" }`, `GetID/SetID` over `areaDoorId` (string), and `Transform(Model) (RestModel, error)` exposing owner, party, field (world/channel/map/instance), townMapId, slot, positions, pairId, townPortalId, expiresAt (unix-milli).

- [ ] **Step 3: Implement `door/rest.go`** — `InitResource(si)` registering `GET /doors/{doorId}` → `GetById` → Transform → JSON:API (mirror summons `rest.go`).

- [ ] **Step 4: Implement `world/resource.go`** — `GET /worlds/{w}/channels/{c}/maps/{m}/instances/{i}/doors` → `GetInField(f)` → `[]RestModel` (mirror summons `world/resource.go`).

- [ ] **Step 5: Write a Transform test** asserting RestModel fields map from a built Model (GetName=="doors", positions/slot/portal/expiry correct).

- [ ] **Step 6: Wire `main.go`.** Re-add (restoring the bits stripped in Task 1, adapted): `door.InitIdAllocator(rc)`, `door.InitRegistry(rc)`; register the three consumers (`doorcmd.InitConsumers`/`InitHandlers`, `characterevt.InitConsumers`/`InitHandlers`, `partyevt.InitConsumers`/`InitHandlers`); register routes (`door.InitResource`, `world.InitResource`); register the leader-elected expiry task via `tasks.Register` gated by `leaderconfig`. Match the atlas-summons bootstrap ordering.

- [ ] **Step 7: Build + test + vet the whole module.**
  Run:
  ```bash
  cd services/atlas-doors/atlas.com/doors && go build ./... && go vet ./... && go test -race ./... && cd -
  ```
  Expected: all clean.

- [ ] **Step 8: Commit**
  ```bash
  git add services/atlas-doors
  git commit -m "feat(atlas-doors): REST resources + main.go wiring (engine complete)"
  ```

---

## Phase 3 — Packet library (`libs/atlas-packet/door`)

v83 byte layouts are transcribed from Cosmic (`~/source/Cosmic` `tools/PacketCreator.java` and `net/server/channel/handlers/DoorHandler.java`). **Read the actual Cosmic source for each packet — do not guess byte order.** Each encoder/decoder reads tenant from context and branches on version where structures diverge (pattern: `libs/atlas-packet/monster/clientbound/spawn.go`). v84≡v83 structurally; use `MajorVersion() >= 87` (not `> 83`) for v87+ branches (memory `bug_majorversion_gt83_is_off_by_one_v87`).

### Task 17: spawnDoor + removeDoor encoders

**Files:**
- Create: `libs/atlas-packet/door/clientbound/spawn.go`
- Create: `libs/atlas-packet/door/clientbound/remove.go`
- Test: `libs/atlas-packet/door/clientbound/spawn_test.go`
- Test: `libs/atlas-packet/door/clientbound/remove_test.go`

- [ ] **Step 1: Read Cosmic `PacketCreator.spawnDoor` and `removeDoor`.** Transcribe the exact field order for v83. spawnDoor (Cosmic shape): `oid (int)`, `townFlag/in-town (byte or via linkedPortalId==-1)`, `townMapId (int)`, `x (short)`, `y (short)` — confirm exact fields/order against the source. removeDoor: `oid (int)` + flag — confirm.

- [ ] **Step 2: Write the failing golden-byte test (v83).** Build the packet, encode under a v83 tenant context, assert the exact byte slice matches the transcribed Cosmic layout. Example skeleton:
  ```go
  func TestSpawnDoorV83Bytes(t *testing.T) {
  	ctx := tenantContext(t, "GMS", 83, 1) // helper builds tenant ctx
  	pkt := NewSpawnDoor(/* oid, inTown, townMapId, x, y, ... */)
  	got := pkt.Encode(testLogger(), ctx)(map[string]interface{}{})
  	want := []byte{ /* exact bytes from Cosmic spawnDoor layout */ }
  	if !bytes.Equal(got, want) {
  		t.Fatalf("v83 spawnDoor bytes:\n got %v\nwant %v", got, want)
  	}
  }
  ```
  (Provide a small `tenantContext`/`testLogger` test helper in the test file — not a `*_testhelpers.go` production file.)

- [ ] **Step 3: Run, expect FAIL, implement the encoders, re-run.** Implement `NewSpawnDoor(...)` / `NewRemoveDoor(...)` + `Encode` with `response.NewWriter`, `tenant.MustFromContext(ctx)`, and version branches. Writer name consts:
  ```go
  const SpawnDoorWriter = "SpawnDoor"
  const RemoveDoorWriter = "RemoveDoor"
  ```
  Run: `go test ./libs/atlas-packet/door/clientbound/ -run 'SpawnDoor|RemoveDoor' -v`
  Expected: PASS for v83.

- [ ] **Step 4: Commit**
  ```bash
  git add libs/atlas-packet/door/clientbound/spawn.go libs/atlas-packet/door/clientbound/remove.go libs/atlas-packet/door/clientbound/spawn_test.go libs/atlas-packet/door/clientbound/remove_test.go
  git commit -m "feat(atlas-packet): door spawn/remove encoders (v83 golden bytes)"
  ```

---

### Task 18: spawnPortal + playPortalSound encoders

**Files:**
- Create: `libs/atlas-packet/door/clientbound/spawn_portal.go`
- Create: `libs/atlas-packet/door/clientbound/play_portal_sound.go`
- Test: `libs/atlas-packet/door/clientbound/spawn_portal_test.go`
- Test: `libs/atlas-packet/door/clientbound/play_portal_sound_test.go`

- [ ] **Step 1: Read Cosmic `PacketCreator.spawnPortal` and `playPortalSound`.** Transcribe v83 layouts. spawnPortal (town↔target minimap portal): `townMapId (int)`, `targetMapId (int)`, `x (short)`, `y (short)` — confirm. playPortalSound: opcode + (typically) a sound-effect selector — confirm against source.

- [ ] **Step 2: Write failing v83 golden-byte tests** for both (same pattern as Task 17).

- [ ] **Step 3: Run, expect FAIL, implement, re-run.** Writer name consts:
  ```go
  const SpawnPortalWriter = "SpawnPortal"
  const PlayPortalSoundWriter = "PlayPortalSound" // confirm Cosmic uses a dedicated op vs. a field-effect op
  ```
  Run: `go test ./libs/atlas-packet/door/clientbound/ -run 'SpawnPortal|PlayPortalSound' -v`
  Expected: PASS for v83.

- [ ] **Step 4: Commit**
  ```bash
  git add libs/atlas-packet/door/clientbound/spawn_portal.go libs/atlas-packet/door/clientbound/play_portal_sound.go libs/atlas-packet/door/clientbound/spawn_portal_test.go libs/atlas-packet/door/clientbound/play_portal_sound_test.go
  git commit -m "feat(atlas-packet): spawnPortal + playPortalSound encoders (v83 golden bytes)"
  ```

---

### Task 19: enter-door serverbound decoder + partyPortal door fields

**Files:**
- Create: `libs/atlas-packet/door/serverbound/enter.go`
- Test: `libs/atlas-packet/door/serverbound/enter_test.go`
- Modify: `libs/atlas-packet/party/clientbound/created.go`
- Test: `libs/atlas-packet/party/clientbound/created_test.go`

- [ ] **Step 1: Read Cosmic `DoorHandler.handlePacket`.** The serverbound shape is `int ownerId`, `byte direction` (`1` = town→target, `0` = target→town). Implement the decoder:
  ```go
  package serverbound

  type Enter struct {
  	ownerId   uint32
  	direction byte
  }
  func (m Enter) OwnerId() uint32 { return m.ownerId }
  func (m Enter) Direction() byte { return m.direction }
  func (m *Enter) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, opts map[string]interface{}) {
  	return func(r *request.Reader, opts map[string]interface{}) {
  		m.ownerId = r.ReadUint32()
  		m.direction = r.ReadByte()
  	}
  }
  const EnterDoorHandle = "EnterDoor"
  ```

- [ ] **Step 2: Write a decode test** feeding `int ownerId=42, byte direction=1` bytes and asserting the decoded fields. (v83; confirm whether any version prepends/extends — if IDA shows a delta in Phase 5, add a branch then.)

- [ ] **Step 3: Wire partyPortal door fields.** The party-operation `Created` packet (`libs/atlas-packet/party/clientbound/created.go`) currently hard-zeros the door fields. Add an optional door-state carrier so the channel can populate door map x/y (int) + minimap x/y (short) when a door exists for that party member. Keep the zero-fill path for the no-door case (backward compatible). Add a constructor variant:
  ```go
  func NewCreatedWithDoor(mode byte, partyId uint32, doorMapX, doorMapY int32, doorMiniX, doorMiniY int16) Created
  ```
  and have `Encode` write the provided values instead of the four zeros. Update `Decode` comments to match.

- [ ] **Step 4: Write a test** asserting `NewCreatedWithDoor` encodes the door fields (and that `NewCreated` still encodes zeros — no regression).

- [ ] **Step 5: Run, expect FAIL, implement, re-run.**
  Run: `go test ./libs/atlas-packet/door/serverbound/ ./libs/atlas-packet/party/clientbound/ -v`
  Expected: PASS.

- [ ] **Step 6: Commit**
  ```bash
  git add libs/atlas-packet/door/serverbound libs/atlas-packet/party/clientbound/created.go libs/atlas-packet/party/clientbound/created_test.go
  git commit -m "feat(atlas-packet): enter-door decoder + party-portal door fields"
  ```

---

## Phase 4 — atlas-channel edge

### Task 20: Cast routing — door Lookup handler

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/skill/handler/door/door.go`
- Create: `services/atlas-channel/atlas.com/channel/skill/handler/door/producer.go` (COMMAND_TOPIC_DOOR SPAWN emitter)
- Modify: `services/atlas-channel/atlas.com/channel/skill/handler/registrations/registrations.go`
- Test: `services/atlas-channel/atlas.com/channel/skill/handler/door/door_test.go`

- [ ] **Step 1: Implement the SPAWN producer** in `door/producer.go`: a `model.Provider[[]kafka.Message]` building `door.Command[door.SpawnCommandBody]` (import the envelope types from `atlas-doors`? No — atlas-channel must not import atlas-doors. Re-declare the command envelope + body + topic const locally in this package, matching the JSON shape exactly, the same way atlas-summons consumers and producers mirror types across services). Topic env: `COMMAND_TOPIC_DOOR`.

- [ ] **Step 2: Implement the handler** `door.go` registering for `skill2.PriestMysticDoorId` via the per-skill `handler.Register`. The handler:
  1. Loads the map metadata for `f.MapId()` via the channel's `data/map` processor.
  2. **Channel-side rejections (emit nothing, client already re-enabled):** if `mapModel.FieldLimit() & map.FieldLimitNoMysticDoor != 0`, or `mapModel.Town()`, or no valid return map → return nil (no SPAWN).
  3. Otherwise load the caster position (`character.NewProcessor(l, ctx).GetById()(characterId)` → `c.X(), c.Y()`) and emit `COMMAND_TOPIC_DOOR / SPAWN{ownerCharacterId, x, y, skillId, skillLevel}`.
  Use the `Handler` signature from `registry.go`. Register via `init()`:
  ```go
  func init() { handler.Register(skill2.PriestMysticDoorId, Handle) }
  ```

- [ ] **Step 3: Add the blank import** to `registrations/registrations.go`:
  ```go
  _ "atlas-channel/skill/handler/door" // Priest Mystic Door — task 093
  ```

- [ ] **Step 4: Write the failing test** (mirror the heal handler tests; inject the map-load and emitter seams like `loadCasterFunc`):
  ```go
  func TestDoorHandlerEmitsSpawnWithCasterPosition(t *testing.T) { /* non-town, no field-limit → one SPAWN with x/y */ }
  func TestDoorHandlerRejectsOnFieldLimit(t *testing.T) { /* FieldLimitNoMysticDoor set → zero emits */ }
  func TestDoorHandlerRejectsOnTownMap(t *testing.T) { /* Town()==true → zero emits */ }
  func TestDoorHandlerRejectsOnNoReturnMap(t *testing.T) { /* returnMap invalid → zero emits */ }
  ```

- [ ] **Step 5: Run, expect FAIL, implement, re-run.**
  Run: `cd services/atlas-channel/atlas.com/channel && go test ./skill/handler/door/ -v`
  Expected after impl: PASS.

- [ ] **Step 6: Commit**
  ```bash
  git add services/atlas-channel/atlas.com/channel/skill/handler/door services/atlas-channel/atlas.com/channel/skill/handler/registrations/registrations.go
  git commit -m "feat(atlas-channel): route Mystic Door cast to SPAWN command"
  ```

---

### Task 21: Enter-door inbound handler (validate + warp)

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/door_enter.go`
- Create: `services/atlas-channel/atlas.com/channel/door/processor.go` (REST client → atlas-doors GET)
- Modify: `services/atlas-channel/atlas.com/channel/main.go` (register handler name→func in `produceHandlers`)
- Test: `services/atlas-channel/atlas.com/channel/socket/handler/door_enter_test.go`

- [ ] **Step 1: Implement the channel→atlas-doors REST client** `door/processor.go`:
  ```go
  type Processor interface {
  	GetInField(f field.Model) ([]Model, error) // GET /worlds/.../maps/{m}/instances/{i}/doors
  	GetById(areaDoorId uint32) (Model, error)  // GET /doors/{id}
  }
  func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor
  ```
  Model mirrors the atlas-doors RestModel fields the channel needs (owner, party, field, townMapId, slot, area/town positions, townPortalId, pairId, areaDoorId, townDoorId). Base URL via `DOORS_SERVICE_URL`/`BASE_SERVICE_URL` fallback (the standard channel REST pattern).

- [ ] **Step 2: Implement `door_enter.go`.** Signature like `buddy_operation.go`:
  ```go
  func DoorEnterHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, opts map[string]interface{})
  ```
  Decode via `serverbound.Enter`. Flow:
  1. `GetInField(s.Field())` from atlas-doors; find the door whose `OwnerCharacterId == ownerId` present on the character's current map (area door map OR town map matching current map).
  2. Validate requester is the owner **or** a current same-channel party member (read party via the existing channel party processor). If not, or if mid-map-change/banned → send the blocked message + re-enable actions (use the existing block helper) and return.
  3. Determine destination from `direction` + which side the character is on: town side → warp to the area door map at area x/y; area side → warp to town map at town x/y. Use `portal.NewProcessor(l, ctx).Warp(s.Field(), s.CharacterId(), targetMapId)`.
  4. Play the portal sound: `session.Announce(...)(clientbound.PlayPortalSoundWriter)(...)` to the entering session.

- [ ] **Step 3: Register the handler name→func** in `main.go` `produceHandlers`:
  ```go
  handlerMap[doorsb.EnterDoorHandle] = handler.DoorEnterHandleFunc
  ```
  (Import the `door/serverbound` package as `doorsb`.) The opcode→validator mapping for `EnterDoorHandle` is added to tenant templates in Phase 5 with `LoggedInValidator`.

- [ ] **Step 4: Write the failing test** (inject the door-REST and party seams; assert warp called with the right target for each direction; assert ineligible requester is rejected without a warp):
  ```go
  func TestEnterDoorOwnerAreaToTownWarps(t *testing.T) {}
  func TestEnterDoorTownToAreaWarps(t *testing.T) {}
  func TestEnterDoorPartyMemberAllowed(t *testing.T) {}
  func TestEnterDoorStrangerRejectedNoWarp(t *testing.T) {}
  ```

- [ ] **Step 5: Run, expect FAIL, implement, re-run.**
  Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run TestEnterDoor -v`
  Expected after impl: PASS.

- [ ] **Step 6: Commit**
  ```bash
  git add services/atlas-channel/atlas.com/channel/socket/handler/door_enter.go services/atlas-channel/atlas.com/channel/door services/atlas-channel/atlas.com/channel/main.go services/atlas-channel/atlas.com/channel/socket/handler/door_enter_test.go
  git commit -m "feat(atlas-channel): enter-door handler (validate + warp via portal)"
  ```

---

### Task 22: Door status consumer → broadcast

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/kafka/consumer/door/consumer.go`
- Create: `services/atlas-channel/atlas.com/channel/kafka/consumer/door/kafka.go` (mirror the atlas-doors StatusEvent envelope locally)
- Modify: `services/atlas-channel/atlas.com/channel/main.go` (register the consumer + handlers)
- Test: `services/atlas-channel/atlas.com/channel/kafka/consumer/door/consumer_test.go`

- [ ] **Step 1: Mirror the StatusEvent envelope** in `kafka.go` (CREATED/REMOVED/SLOT_CHANGED + bodies), matching the atlas-doors JSON shape. Topic env `EVENT_TOPIC_DOOR_STATUS`.

- [ ] **Step 2: Implement the consumer** mirroring `kafka/consumer/monster/consumer.go`: `SetHeaderParsers(Span, Tenant)`, `InitConsumers`/`InitHandlers`. Handlers:
  - **CREATED:** resolve eligible viewers = caster ∪ same-channel party members present in (a) the source field and (b) the town map. To the field viewers, `Announce` `SpawnDoorWriter` (area door). To the town viewers, `Announce` `SpawnDoorWriter` (town door) + `SpawnPortalWriter` (minimap portal). To all party members, send the party-portal update (`party/clientbound.NewCreatedWithDoor`). Eligibility = intersect `_map.NewProcessor.ForSessionsInMap(field)` with party membership (caster always included) — use `party.MemberInMap`/`OtherMemberInMap` filters (see context.md).
  - **REMOVED:** `Announce` `RemoveDoorWriter` to both maps' eligible viewers + clear the party-portal (`NewCreated` zero-fill). Honor the deploy grace (FR-6.3): if the elapsed time since the matching CREATED is < ~3000ms, defer the remove broadcast by the remainder (track CREATED deploy time in a small per-pair in-memory map, or compare against `expiresAt - duration`). Document the chosen mechanism.
  - **SLOT_CHANGED:** `Announce` `RemoveDoorWriter` for the old town placement then `SpawnDoorWriter`+`SpawnPortalWriter` for the new town placement to town viewers; update the party-portal.

- [ ] **Step 3: Register the consumer** in `main.go` alongside the other channel status consumers (monster/summon).

- [ ] **Step 4: Write tests** with injected session-enumeration + party seams: CREATED announces spawnDoor to an in-field party member but not to a non-party in-field session; REMOVED announces removeDoor; SLOT_CHANGED re-announces at the new portal. Assert the writer names + target sessions.

- [ ] **Step 5: Run, expect FAIL, implement, re-run.**
  Run: `cd services/atlas-channel/atlas.com/channel && go test ./kafka/consumer/door/ -v`
  Expected after impl: PASS.

- [ ] **Step 6: Commit**
  ```bash
  git add services/atlas-channel/atlas.com/channel/kafka/consumer/door services/atlas-channel/atlas.com/channel/main.go services/atlas-channel/atlas.com/channel/kafka/consumer/door/consumer_test.go
  git commit -m "feat(atlas-channel): door status consumer → eligible-viewer broadcast"
  ```

---

### Task 23: Map-enter spawn for self (FR-3.4)

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go` (`SpawnForSelf`, after line ~310)
- Test: `services/atlas-channel/atlas.com/channel/kafka/consumer/map/door_spawn_test.go`

- [ ] **Step 1: Add a door-spawn goroutine to `SpawnForSelf`.** After the existing spawn blocks, query atlas-doors `GetInField(f)` for the entering map; filter to doors whose owner is self or a same-channel party member; for each, `Announce` the appropriate spawn packets to the entering session only: area door (`SpawnDoorWriter`) if the map is the door's source field; town door (`SpawnDoorWriter` + `SpawnPortalWriter`) if the map is the door's town map. Reuse the same eligibility filter as Task 22 (extract a shared helper `eligibleDoorsFor(l, ctx, s, doors)` to keep DRY).

- [ ] **Step 2: Write a test** asserting an entering session that is a party member receives spawn packets for an existing in-field door, and a non-party entrant receives none. Inject the door-REST + party seams.

- [ ] **Step 3: Run, expect FAIL, implement, re-run.**
  Run: `cd services/atlas-channel/atlas.com/channel && go test ./kafka/consumer/map/ -run TestDoorSpawn -v`
  Expected after impl: PASS.

- [ ] **Step 4: Commit**
  ```bash
  git add services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go services/atlas-channel/atlas.com/channel/kafka/consumer/map/door_spawn_test.go
  git commit -m "feat(atlas-channel): spawn eligible doors to entering session (FR-3.4)"
  ```

---

### Task 24: Leaver visibility revocation (FR-6.4)

The engine re-slots on party change; the channel revokes a leaver's visibility (sends `removeDoor` to the leaver only, without destroying the door). Driven by the party event the channel already consumes.

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/party/consumer.go` (or wherever the channel handles party membership events)
- Test: `services/atlas-channel/atlas.com/channel/kafka/consumer/party/door_visibility_test.go`

- [ ] **Step 1: Locate the channel's party-status consumer.** Find where atlas-channel reacts to party membership changes (search `EVENT_TOPIC_PARTY_STATUS` consumers under `kafka/consumer/`). If none reacts to leaves yet, add a handler.

- [ ] **Step 2: On a leave event,** for the leaver's session, query atlas-doors for the party's other doors visible to the leaver and `Announce` `RemoveDoorWriter` to the leaver only (visibility revocation, not destruction). On a join event, (re)spawn the party's existing doors to the joiner's session (reuse the Task 23 eligible-spawn helper).

- [ ] **Step 3: Write a test** asserting the leaver receives removeDoor for a remaining party door (door still exists in the registry — not removed) and a joiner receives spawnDoor.

- [ ] **Step 4: Run, expect FAIL, implement, re-run.**
  Run: `cd services/atlas-channel/atlas.com/channel && go test ./kafka/consumer/party/ -run TestDoorVisibility -v`
  Expected after impl: PASS.

- [ ] **Step 5: Commit**
  ```bash
  git add services/atlas-channel/atlas.com/channel/kafka/consumer/party
  git commit -m "feat(atlas-channel): party join/leave door visibility delta (FR-6.4)"
  ```

---

### Task 25: atlas-channel module verification

**Files:** none (verification only)

- [ ] **Step 1: Build, vet, test the channel module.**
  Run:
  ```bash
  cd services/atlas-channel/atlas.com/channel && go build ./... && go vet ./... && go test -race ./... && cd -
  ```
  Expected: all clean. Fix any wiring/regression (existing tests reference internal handler names — adjust call sites, don't rename blindly; memory: "Test files reference internal functions").

- [ ] **Step 2: Build the packet lib.**
  Run: `cd libs/atlas-packet && go build ./... && go vet ./... && go test -race ./... && cd -`
  Expected: clean.

- [ ] **Step 3: Commit (if any fixes were needed).**
  ```bash
  git add -A
  git commit -m "test(atlas-channel): fix call sites after door wiring"
  ```

---

## Phase 5 — Per-version opcode coverage (OQ-5)

This phase resolves opcodes + byte structures for the **non-v83** versions and wires every packet into tenant templates and live config. It is investigative (IDA/WZ). **An unresolved fname is a STOP-AND-ESCALATE — park that version/packet, do not guess** (memory `feedback_unresolved_fname_escalate`).

### Task 26: Per-version packet structure verification matrix

**Files:**
- Modify: `docs/tasks/task-093-mystic-door/context.md` (append a "## Packet verification matrix" table)
- Add per-version branches + golden tests to the Phase-3 packet files as needed.

- [ ] **Step 1: Build the matrix.** For each version `gms_v84, gms_v87, gms_v92, gms_v95, jms_v185` × each packet (`spawnDoor, removeDoor, spawnPortal, playPortalSound, partyPortal, enter-door`):
  1. Select the version's IDB (`mcp__ida-pro__select_instance(port)` — v83/v87/v95/jms per memory `reference_ida_mcp_new_api`).
  2. Resolve the opcode (handler + writer) and confirm the byte structure against the decompile. v84≡v83 for structure (verify the opcode row only).
  3. Record opcode + "structure same/diff vs v83" in the matrix table.
  4. If an fname doesn't resolve → mark the cell **PARKED** with the reason; do not invent bytes.

- [ ] **Step 2: Add per-version encode/decode branches** to the Phase-3 packet files wherever structure diverges from v83, each guarded by `MajorVersion() >= 87` / region checks (not `> 83`). Add a golden-byte test per (version × packet) that has a confirmed structure.

- [ ] **Step 3: Run the packet lib tests.**
  Run: `cd libs/atlas-packet && go test -race ./door/... ./party/clientbound/... && cd -`
  Expected: PASS for every non-parked cell.

- [ ] **Step 4: Commit**
  ```bash
  git add libs/atlas-packet/door docs/tasks/task-093-mystic-door/context.md
  git commit -m "feat(atlas-packet): per-version door packet structures + matrix (OQ-5)"
  ```

---

### Task 27: Tenant socket template opcode rows

**Files:**
- Modify: the tenant socket templates for each version (handlers + writers). Locate them: `grep -rln "EnterDoor\|socket.handlers\|socket.writers" deploy/ services/atlas-tenants` and the seed templates (search for where other handler/writer opcodes are defined per version).

- [ ] **Step 1: Find the template source.** Identify where per-version `socket.handlers` and `socket.writers` rows live (seed templates / tenant config). Confirm the structure with an existing entry (e.g. a recently-added writer like a monster/mount packet).

- [ ] **Step 2: Add writer rows** for `SpawnDoor`, `RemoveDoor`, `SpawnPortal`, `PlayPortalSound` (and confirm the party-operation writer already exists — partyPortal reuses `PartyOperation`) for every non-parked version, using the opcodes from Task 26.

- [ ] **Step 3: Add the handler row** for `EnterDoor` for every non-parked version, **each with a validator** (`LoggedInValidator`). A validator-less handler is silently dropped (memory `bug_socket_handler_missing_validator_silently_dropped`).

- [ ] **Step 4: Build/validate the template format** (run whatever schema check the repo provides for templates, or at minimum `go build` of atlas-tenants if templates are Go-embedded).

- [ ] **Step 5: Commit**
  ```bash
  git add -A
  git commit -m "feat(tenants): door packet handler/writer opcode rows (all versions)"
  ```

---

### Task 28: Live tenant config patch (deploy step) + k8s manifest

**Files:**
- Create: `deploy/k8s/base/atlas-doors.yaml`
- Modify: any kustomize overlay that enumerates services (if the repo lists deployments per overlay)
- Document: append a "## Deploy / live-config steps" section to context.md

- [ ] **Step 1: Author `deploy/k8s/base/atlas-doors.yaml`** — Deployment + Service mirroring `deploy/k8s/base/atlas-summons.yaml`. Env: Redis, Kafka brokers, `COMMAND_TOPIC_DOOR`, `EVENT_TOPIC_DOOR_STATUS`, `EVENT_TOPIC_CHARACTER_STATUS`, `EVENT_TOPIC_PARTY_STATUS`, `DOOR_LEADER_*`, and DATA/PARTY service URLs via `BASE_SERVICE_URL` fallback (do NOT hard-code `*_SERVICE_URL` from the kustomize base — memory `bug_service_url_hardcoded_base_namespace`). Readiness probe path **`/api/readyz`** (memory `bug_readiness_probe_path_under_api_basepath`).

- [ ] **Step 2: Add the channel env** for `COMMAND_TOPIC_DOOR` / `EVENT_TOPIC_DOOR_STATUS` / `DOORS_SERVICE_URL` to the atlas-channel deployment manifest if topic/URL envs are declared there.

- [ ] **Step 3: Document the live-config patch** in context.md: existing tenants do NOT auto-receive the new handler/writer opcodes — they must be patched into live tenant config and the **channel restarted** (projection does not hot-reload handlers/writers — memory `bug_new_opcodes_not_in_live_tenant_config`). List the exact opcodes per version to patch. (This is an operational step performed at deploy time, not in code.)

- [ ] **Step 4: Validate kustomize.**
  Run: `kubectl kustomize deploy/k8s/base >/dev/null` (or the repo's standard kustomize validation).
  Expected: no error; `atlas-doors` resources render.

- [ ] **Step 5: Commit**
  ```bash
  git add deploy/k8s docs/tasks/task-093-mystic-door/context.md
  git commit -m "feat(deploy): atlas-doors k8s manifest + live-config notes"
  ```

---

## Phase 6 — Full verification

### Task 29: Full build/test/bake/guard sweep

**Files:** none (verification only)

- [ ] **Step 1: Test + vet + build every changed module.**
  Run (from worktree root):
  ```bash
  for d in services/atlas-doors/atlas.com/doors services/atlas-channel/atlas.com/channel libs/atlas-packet; do
    echo "== $d =="; (cd "$d" && go test -race ./... && go vet ./... && go build ./...) || break
  done
  ```
  Expected: all clean.

- [ ] **Step 2: redis-key-guard.**
  Run: `GOWORK=off tools/redis-key-guard.sh`
  Expected: no findings.

- [ ] **Step 3: docker buildx bake the touched go-mod services.**
  Run (from worktree root):
  ```bash
  docker buildx bake atlas-doors
  docker buildx bake atlas-channel
  ```
  Expected: both succeed. (atlas-doors adds no new shared lib, so the root Dockerfile needs no edit — confirm the build proves it.)

- [ ] **Step 4: Confirm registration completeness.** Verify `atlas-doors` appears in: `.github/config/services.json`, `docker-bake.hcl` `go_services`, `go.work`, and `deploy/k8s/base/atlas-doors.yaml`. Confirm the root `Dockerfile` was NOT edited (no new lib).
  ```bash
  grep -q atlas-doors .github/config/services.json && grep -q '"atlas-doors"' docker-bake.hcl && grep -q atlas-doors go.work && test -f deploy/k8s/base/atlas-doors.yaml && echo "registration OK"
  ```
  Expected: `registration OK`.

- [ ] **Step 5: Acceptance-criteria self-check.** Walk the PRD §10 acceptance checklist against the implemented tasks; note any gap in context.md. (Code review in the next step is the formal gate.)

- [ ] **Step 6: Commit any final fixes.**
  ```bash
  git add -A && git commit -m "chore(task-093): final verification sweep" || echo "nothing to commit"
  ```

---

### Task 30: Code review

**Files:** `docs/tasks/task-093-mystic-door/audit.md` (written by reviewers)

- [ ] **Step 1: Run the code-review step BEFORE opening a PR** (CLAUDE.md "Code Review Before PR"). Invoke `superpowers:requesting-code-review`; it dispatches `plan-adherence-reviewer` + `backend-guidelines-reviewer` (Go files changed). Address findings via `superpowers:receiving-code-review`.

- [ ] **Step 2: Re-run the Task 29 verification sweep** after addressing review findings.

- [ ] **Step 3: Only then** proceed to `superpowers:finishing-a-development-branch`.

---

## Self-review (plan author)

**Spec coverage (PRD §4 FRs → tasks):**
- FR-1.1 cast routing → Task 20. FR-1.2 rejections → Task 20 (channel) + Task 11 (engine re-check). FR-1.3 MP/Magic Rock consume → no task needed (OQ-1: existing `UseSkill` path; verified in design §1). FR-1.4 recast replace → Task 11 Step 2.
- FR-2.1/2.2/2.3 paired door, shared oid, return town → Tasks 2, 4, 11, 6.
- FR-3.1–3.4 visibility/broadcast/map-enter → Tasks 22, 23.
- FR-4.1–4.3 slot assignment + re-slot → Tasks 3, 11, 14.
- FR-5.1–5.4 enter/warp → Tasks 19, 21.
- FR-6.1 expiry → Task 15. FR-6.2 disconnect/channel/leave-field cleanup → Task 13. FR-6.3 deploy grace → Task 22 Step 2. FR-6.4 party membership → Tasks 14, 24. FR-6.5 per-channel → enforced by channel eligibility (Tasks 22/23). FR-6.6 ephemeral → Task 5 (Redis-only).
- FR-7.1–7.4 version coverage → Tasks 17–19 (v83) + 26–28 (other versions, templates, live config).
- API §5: atlas-doors REST → Task 16; Kafka topics → Tasks 9, 12; channel edge → Tasks 20–24; cross-service reads → Tasks 6, 7, 8.
- Data model §6 → Tasks 2, 5. Service impact §7 → all phases. NFRs §8 → Tasks 5 (multi-tenancy/redis-guard), 11 (concurrency/resilience), 15 (cleanup backstop), 22 (client safety). Registration §10 → Tasks 1, 28.

**Placeholder scan:** Packet byte tasks (17–19, 26) intentionally instruct "read Cosmic/IDA, transcribe exact bytes, golden-byte test" rather than inlining fabricated bytes — this honors the project's Verification-Over-Memory rule (fabricating packet bytes would be the real failure). All other code steps contain concrete code.

**Type consistency:** `door.Model` getters/builder (Task 2) are reused verbatim in Tasks 3–16. Envelope types (Task 9) are consumed by Tasks 10–14 and mirrored (not imported) channel-side in Tasks 20–22. Writer-name consts (`SpawnDoorWriter`, `RemoveDoorWriter`, `SpawnPortalWriter`, `PlayPortalSoundWriter`, Tasks 17–18) and `EnterDoorHandle` (Task 19) are referenced consistently in Tasks 21–23, 27.

**Known gap flagged for execution:** the atlas-summons template lives in the task-088 sibling worktree (not merged). Every "copy from atlas-summons" step depends on that worktree existing at execution time; if it is gone, fall back to `atlas-monsters` patterns (context.md). atlas-doors must never import atlas-summons.

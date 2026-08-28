# Attack-Path Snapshots (PS-1) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove all REST calls from the steady-state attack path in atlas-channel by serving character/skills/inventory/buffs from a session-scoped, event-maintained snapshot, monster reads from the (extended) task-120 live mirror, and skill-effect reads from an in-process TTL cache — with REST retained only as a counted, backfilling miss-fallback.

**Architecture:** A new `character/snapshot` package holds a tenant-scoped singleton registry keyed by character id with four independently-validated components (core `character.Model`, `[]skill.Model`, `inventory.Model`, `[]buff.Model`) plus a locally-fed position overlay. Additive Kafka handlers on the existing character/skill/asset/compartment/buff consumers maintain it (rich events apply absolute values in place; thin events invalidate; every mutation bumps a per-component generation counter that guards REST backfills). `processAttack` composes the exact decorated model it fetches today from the snapshot; reflect/MP-Eater monster reads go through one memoized resolve per damaged monster against the task-120 `LiveMirror` (extended with X/Y); `data/skill` gains a task-060-semantics in-process TTL cache. One bounded producer change in atlas-character populates `Values` on every STAT_CHANGED emission so MP-consuming swings do not self-invalidate.

**Tech Stack:** Go, Kafka (segmentio/kafka-go via libs/atlas-kafka), Prometheus client_golang (arrives with task-120), gorm+sqlite test harness (atlas-character only).

**Spec:** `docs/tasks/task-122-attack-path-snapshots/design.md` (PRD: `prd.md`, FR-2 audit: `event-coverage.md`, same folder).

## Global Constraints

- **task-120 must be merged into this branch before any code task runs.** Task 1 is a hard gate: it rebases onto main and reconciles every task-120 API name this plan consumes (design FR-5.2). If task-120 is not on main yet, STOP and report BLOCKED.
- Steady-state attack path performs **zero REST calls**; every remaining REST call on the path is a counted fallback that backfills local state (PRD FR-4).
- **Wire-level equivalence** (PRD FR-4.6): for identical inputs, damage-application commands, status applies, reflect emissions, projectile consumption emissions, and broadcast packet bytes are identical to pre-change behavior.
- Snapshot event handlers are **update-only**: they never create entries (consumers run from `kafka.LastOffset`; entries are created only by the lazy `Get` populate). Every component mutation — in-place update *or* invalidation — **bumps that component's generation counter**; REST backfills apply only if the generation is unchanged (design §4.4).
- In-place event updates carry **absolute values only** (idempotent under at-least-once redelivery, PRD NFR-4). Deltas and un-appliable types invalidate instead. Silently caching over a known gap is prohibited (FR-2.4).
- All snapshot/cache state is tenant-scoped (`tenant.MustFromContext`); no cross-tenant reads (PRD NFR-5). Registry discipline: `sync.Once` singleton + `sync.RWMutex`, value-copy reads, no unbounded goroutines (NFR-6).
- `Values` key convention (established by `services/atlas-character/atlas.com/character/character/processor.go:905,1408,1623,1867`): **snake_case stat names** — `skin, face, hair, level, job, strength, dexterity, intelligence, luck, hp, max_hp, mp, max_mp, available_ap, available_sp, experience, fame, meso, gachapon_experience`. JSON round-trip delivers numbers as `float64`.
- Metric names exactly (all labeled `tenant`; design §8):
  - `atlas_channel_char_snapshot_reads_total{tenant, component, outcome="hit|miss|fallback_success|fallback_failure"}`
  - `atlas_channel_char_snapshot_updates_total{tenant, component, kind="event_update|invalidation|backfill|backfill_discarded"}`
  - `atlas_channel_skill_data_cache_total{tenant, outcome="hit|negative_hit|miss"}`
  - `atlas_channel_char_snapshot_divergence_total{tenant, component}`
  - `component` label values: `core|skills|inventory|buffs|position`.
- Env vars: `SKILL_DATA_CACHE_ENABLED` (default `true`), `SKILL_DATA_CACHE_TTL` (default `5m`, clamp `[1s, 24h]`), `SKILL_DATA_CACHE_NEGATIVE_TTL` (default `30s`, clamp `[0s, 5m]`), `CHAR_SNAPSHOT_SHADOW_SAMPLE_RATE` (float, default `0`). Invalid values warn once and fall back to defaults.
- Test setup uses the project Builder pattern; **no `*_testhelpers.go` files** (CLAUDE.md).
- This repo is a Go workspace: run `go test`/`go vet`/`go build` from the module directory (`services/atlas-channel/atlas.com/channel` or `services/atlas-character/atlas.com/character`).
- Verification gate before "done": `go test -race ./...`, `go vet ./...`, `go build ./...` clean in both changed modules; `docker buildx bake atlas-channel atlas-character` from the worktree root; `tools/redis-key-guard.sh` from the worktree root (CLAUDE.md).
- No `// TODO`, stubs, or deferred-but-producible work in landed commits. Never write literal home/absolute paths into committed files.

---

### Task 1: task-120 reconciliation gate + execution-time verifications

**Files:**
- Modify: `docs/tasks/task-122-attack-path-snapshots/context.md` (append a "Task 1 findings" section)
- Modify: `docs/tasks/task-122-attack-path-snapshots/event-coverage.md` (resolve the VERIFY-AT-EXECUTION markers)

**Interfaces:**
- Consumes: task-120's landed code (`monster/live_mirror.go`, `monster/metrics.go`, `monster/information/cache.go`, `/api/metrics` mount in `main.go`).
- Produces: a confirmed API inventory that Tasks 2–12 rely on verbatim: `monster.GetLiveMirror()`, `(*LiveMirror).Lookup(t tenant.Model, uniqueId uint32) (LiveEntry, bool)`, `Put`, `Remove`, `EvictTenant`, `monster.LiveEntryFromModel(mo Model) LiveEntry`, `monster.RecordMirrorFallback(t tenant.Model, success bool)`, `LiveEntry{Field, MonsterId, Mp, MaxMp, ControllerHasAggro, LastWrite}`, prometheus dep + `promhttp` mount.

- [x] **Step 1: Verify task-120 is merged and rebase**

```bash
cd .worktrees/task-122-attack-path-snapshots   # worktree root (repo-relative)
git fetch origin main
git log origin/main --oneline --grep "task-120" | head -5
```
Expected: at least one task-120 commit on origin/main. If none: **STOP — report BLOCKED** ("task-122 depends on task-120 landing first; owner decision, PRD FR-5.1"). Do not improvise against the task-120 plan.

```bash
git rebase origin/main
ls services/atlas-channel/atlas.com/channel/monster/live_mirror.go \
   services/atlas-channel/atlas.com/channel/monster/metrics.go \
   services/atlas-channel/atlas.com/channel/monster/information/cache.go
```
Expected: all three files exist after rebase. Resolve any rebase conflicts (docs-only conflicts expected at worst).

- [x] **Step 2: Reconcile the mirror API against this plan**

Read `services/atlas-channel/atlas.com/channel/monster/live_mirror.go` as landed and confirm each name in the Interfaces block above exists with the same signature. Also confirm in `services/atlas-channel/atlas.com/channel/main.go`: the `listener.RegisterEvictor` block calls `monsterDomain.GetLiveMirror().EvictTenant(tid)`, and an `AddRouteInitializer(restserver.MountHandler("/metrics", promhttp.Handler()))` line exists (endpoint `/api/metrics`). If any name/signature differs, update the affected steps in Tasks 10–11 of this plan file (and note the delta in context.md) BEFORE executing them — do not guess mid-task.

- [x] **Step 3: VERIFY (design §10.2) — inventory REST does not net out reservations**

Read the atlas-inventory REST read path: `services/atlas-inventory/atlas.com/inventory/inventory/` (rest.go/resource.go/processor.go — follow `GET /characters/{id}/inventory`). Confirm asset quantities returned are the raw asset rows, not quantity-minus-reservations. Record the answer with file:line in event-coverage.md §4 (replace the VERIFY-AT-EXECUTION marker). If reservations ARE netted out: reservation events must join the inventory component — add `RESERVED`/`RESERVATION_CANCELLED` → `InvalidateInventory` handlers to Task 7 and note the change in context.md.

- [x] **Step 4: VERIFY (design §10.3) — Values key convention**

Confirm at `services/atlas-character/atlas.com/character/character/processor.go:905,1408,1623,1867` that the populated `values` maps use the snake_case keys listed in Global Constraints (`"luck"`, `"max_hp"`, `"max_mp"`, `"intelligence"`, `"strength"`, `"dexterity"` are the ones in evidence today). Record confirmation in context.md.

- [x] **Step 5: VERIFY (design §10.4) — MAP_CHANGED UseTargetPosition=false path**

Re-read `services/atlas-channel/atlas.com/channel/kafka/message/character/kafka.go:114-127` (`StatusEventMapChangedBody`) and confirm `UseTargetPosition/TargetX/TargetY` fields. This plan's Task 5 invalidates BOTH position and core on `UseTargetPosition=false` so the next attack's core refetch carries fresh REST X/Y (reflect-before-first-move falls back to exactly today's source). Record in event-coverage.md §2 that the disposition is implemented as position-invalidate + core-invalidate.

- [x] **Step 6: Escalate the RequestReserve finding (design §10.5)** (finding retracted, not escalated as written: re-confirmed at execution time that `RequestReserve` was already fixed by task-205 on `main` before task-122 began — `compartment/processor.go:810-853` loops over every request, not just the first; see context.md "Task 1 findings" Step 6)

Confirm `services/atlas-inventory/atlas.com/inventory/compartment/processor.go:767` still returns inside its request loop. Add one line to context.md under "Escalations for owner": "`RequestReserve` processes only the first batched reserve request (`compartment/processor.go:767`) — candidate pre-existing atlas-inventory bug, NOT addressed by task-122 (not snapshot-relevant; reservations are registry-only)." Do not fix it in this task.

- [x] **Step 7: Commit**

```bash
git add docs/tasks/task-122-attack-path-snapshots/context.md docs/tasks/task-122-attack-path-snapshots/event-coverage.md
git commit -m "docs(task-122): task-120 reconciliation + execution-time audit verifications"
```

---

### Task 2: Snapshot registry — entry, mutators, metrics, builder enablers

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/character/snapshot/registry.go`
- Create: `services/atlas-channel/atlas.com/channel/character/snapshot/metrics.go`
- Create: `services/atlas-channel/atlas.com/channel/character/snapshot/registry_test.go`
- Create: `services/atlas-channel/atlas.com/channel/character/skill/builder.go`
- Modify: `services/atlas-channel/atlas.com/channel/character/builder.go` (add `SetX`/`SetY` setters)

**Interfaces:**
- Consumes: `character.Model`/`character.CloneModel` (`character/builder.go`), `skill.Model` (`character/skill/model.go`), `inventory.Model` + builders, `compartment.Model` + builders, `buff.Model`/`buff.NewBuff` (`character/buff/model.go`), `stat.Type` constants (`libs/atlas-constants/stat`).
- Produces (used by Tasks 3–8, 11, 12):
  - `snapshot.GetRegistry() *Registry`
  - `type ComponentView struct { Core character.Model; CoreValid bool; CoreGen uint64; Skills []skill.Model; SkillsValid bool; SkillsGen uint64; Inv inventory.Model; InvValid bool; InvGen uint64; Buffs []buff.Model; BuffsValid bool; BuffsGen uint64; PosX, PosY int16; PosValid bool }`
  - `(*Registry).View(t tenant.Model, characterId uint32) ComponentView` — creates the (all-invalid) entry if absent; the ONLY entry-creating call.
  - `(*Registry).ComposedIfValid(t tenant.Model, characterId uint32) (character.Model, bool)`
  - Backfills (generation-checked): `BackfillCore(t, characterId, m character.Model, gen uint64) bool`, `BackfillSkills(t, characterId, ms []skill.Model, gen uint64) bool`, `BackfillInventory(t, characterId, inv inventory.Model, gen uint64) bool`, `BackfillBuffs(t, characterId, bs []buff.Model, gen uint64) bool`
  - Update-only mutators (no-op when entry absent; ALL bump the component gen and clear the composed cache): `ApplyStatChanged(t, characterId, updates []stat.Type, values map[string]interface{})`, `SetLevel(t, characterId, level byte)`, `SetExperience(t, characterId, exp uint32)`, `InvalidateCore(t, characterId)`, `UpsertSkill(t, characterId, sm skill.Model)`, `RemoveSkill(t, characterId, skillId skillconst.Id)`, `InvalidateSkills(t, characterId)`, `UpsertAsset(t, characterId, compartmentId uuid.UUID, a asset.Model)`, `SetAssetQuantity(t, characterId, assetId uint32, quantity uint32)`, `SetAssetSlot(t, characterId, assetId uint32, slot int16)`, `RemoveAsset(t, characterId, assetId uint32)`, `InvalidateInventory(t, characterId)`, `UpsertBuff(t, characterId, b buff.Model)`, `RemoveBuff(t, characterId, sourceId int32)`, `InvalidateBuffs(t, characterId)`, `SetPosition(t, characterId, x, y int16)`, `InvalidatePosition(t, characterId)`
  - `(*Registry).Evict(t tenant.Model, characterId uint32)`, `(*Registry).EvictTenant(tid uuid.UUID)`
  - `snapshot.recordRead(t, component, outcome string)` / `snapshot.recordUpdate(t, component, kind string)` (package-internal, used by Task 3)
  - Component/outcome constants: `componentCore/componentSkills/componentInventory/componentBuffs/componentPosition`, `outcomeHit/outcomeMiss/outcomeFallbackSuccess/outcomeFallbackFailure`, `kindEventUpdate/kindInvalidation/kindBackfill/kindBackfillDiscarded`
  - `character` builder: `SetX(v int16)`, `SetY(v int16)`
  - `character/skill` builder: `skill.NewModelBuilder(id skillconst.Id) *modelBuilder` with `SetLevel/SetMasterLevel/SetExpiration/SetCooldownExpiresAt/Build/MustBuild`, and `skill.Clone(m Model) *modelBuilder`

- [x] **Step 1: Add the trivial builder enablers**

In `services/atlas-channel/atlas.com/channel/character/builder.go`, next to `SetMeso` (line ~138), add:

```go
func (b *modelBuilder) SetX(v int16) *modelBuilder { b.x = v; return b }
func (b *modelBuilder) SetY(v int16) *modelBuilder { b.y = v; return b }
```

Create `services/atlas-channel/atlas.com/channel/character/skill/builder.go`:

```go
package skill

import (
	"errors"
	"time"

	skillconst "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

var ErrInvalidSkillId = errors.New("skill id must be greater than 0")

type modelBuilder struct {
	id                skillconst.Id
	level             byte
	masterLevel       byte
	expiration        time.Time
	cooldownExpiresAt time.Time
}

// NewModelBuilder creates a new builder instance
func NewModelBuilder(id skillconst.Id) *modelBuilder {
	return &modelBuilder{id: id}
}

// Clone creates a builder initialized with the Model's values
func Clone(m Model) *modelBuilder {
	return &modelBuilder{
		id:                m.id,
		level:             m.level,
		masterLevel:       m.masterLevel,
		expiration:        m.expiration,
		cooldownExpiresAt: m.cooldownExpiresAt,
	}
}

func (b *modelBuilder) SetLevel(v byte) *modelBuilder                 { b.level = v; return b }
func (b *modelBuilder) SetMasterLevel(v byte) *modelBuilder           { b.masterLevel = v; return b }
func (b *modelBuilder) SetExpiration(v time.Time) *modelBuilder       { b.expiration = v; return b }
func (b *modelBuilder) SetCooldownExpiresAt(v time.Time) *modelBuilder { b.cooldownExpiresAt = v; return b }

func (b *modelBuilder) Build() (Model, error) {
	if b.id == 0 {
		return Model{}, ErrInvalidSkillId
	}
	return Model{
		id:                b.id,
		level:             b.level,
		masterLevel:       b.masterLevel,
		expiration:        b.expiration,
		cooldownExpiresAt: b.cooldownExpiresAt,
	}, nil
}

func (b *modelBuilder) MustBuild() Model {
	m, err := b.Build()
	if err != nil {
		panic(err)
	}
	return m
}
```

- [x] **Step 2: Write the failing registry tests**

Create `services/atlas-channel/atlas.com/channel/character/snapshot/registry_test.go`:

```go
package snapshot

import (
	"sync"
	"testing"

	"atlas-channel/asset"
	"atlas-channel/character"
	"atlas-channel/character/buff"
	"atlas-channel/character/skill"
	"atlas-channel/compartment"
	"atlas-channel/inventory"

	invconst "github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	skillconst "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-constants/stat"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
)

// newTestRegistry bypasses the singleton for test isolation.
func newTestRegistry() *Registry {
	return &Registry{perTenant: map[uuid.UUID]map[uint32]*entry{}}
}

func newTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return tm
}

func testCore(t *testing.T, id uint32) character.Model {
	t.Helper()
	return character.NewModelBuilder().
		SetId(id).SetLevel(30).SetJobId(312).
		SetMp(500).SetMaxMp(800).SetX(100).SetY(-50).
		MustBuild()
}

func testInventory(t *testing.T, characterId uint32) (inventory.Model, uuid.UUID, asset.Model) {
	t.Helper()
	compId := uuid.New()
	a := asset.NewModelBuilder(9001, compId, 2060000).SetSlot(2).SetQuantity(400).MustBuild()
	comp := compartment.NewBuilder(compId, characterId, invconst.TypeValueUse, 96).AddAsset(a).MustBuild()
	inv := inventory.NewBuilder(characterId).
		SetEquipable(compartment.NewBuilder(uuid.New(), characterId, invconst.TypeValueEquip, 96).MustBuild()).
		SetConsumable(comp).
		SetSetup(compartment.NewBuilder(uuid.New(), characterId, invconst.TypeValueSetup, 96).MustBuild()).
		SetEtc(compartment.NewBuilder(uuid.New(), characterId, invconst.TypeValueETC, 96).MustBuild()).
		SetCash(compartment.NewBuilder(uuid.New(), characterId, invconst.TypeValueCash, 96).MustBuild()).
		MustBuild()
	return inv, compId, a
}

// populate drives a full backfill so the entry is valid for mutation tests.
func populate(t *testing.T, r *Registry, tm tenant.Model, characterId uint32) ComponentView {
	t.Helper()
	v := r.View(tm, characterId)
	if v.CoreValid || v.SkillsValid || v.InvValid || v.BuffsValid || v.PosValid {
		t.Fatalf("fresh entry must start all-invalid: %+v", v)
	}
	if !r.BackfillCore(tm, characterId, testCore(t, characterId), v.CoreGen) {
		t.Fatalf("core backfill rejected")
	}
	sk := skill.NewModelBuilder(skillconst.Id(3121004)).SetLevel(10).MustBuild()
	if !r.BackfillSkills(tm, characterId, []skill.Model{sk}, v.SkillsGen) {
		t.Fatalf("skills backfill rejected")
	}
	inv, _, _ := testInventory(t, characterId)
	if !r.BackfillInventory(tm, characterId, inv, v.InvGen) {
		t.Fatalf("inventory backfill rejected")
	}
	if !r.BackfillBuffs(tm, characterId, []buff.Model{}, v.BuffsGen) {
		t.Fatalf("buffs backfill rejected")
	}
	return r.View(tm, characterId)
}

func TestRegistry_MutatorsNoOpWhenEntryAbsent(t *testing.T) {
	r := newTestRegistry()
	tm := newTestTenant(t)
	r.SetLevel(tm, 7, 31)
	r.InvalidateCore(tm, 7)
	r.SetPosition(tm, 7, 1, 2)
	r.mu.RLock()
	defer r.mu.RUnlock()
	if len(r.perTenant) != 0 {
		t.Fatalf("mutators must never create entries")
	}
}

func TestRegistry_ViewCreatesAllInvalidEntryAndBackfillValidates(t *testing.T) {
	r := newTestRegistry()
	tm := newTestTenant(t)
	v := populate(t, r, tm, 7)
	if !v.CoreValid || !v.SkillsValid || !v.InvValid || !v.BuffsValid {
		t.Fatalf("all components must be valid after backfill: %+v", v)
	}
	if v.PosValid {
		t.Fatalf("position starts invalid until fed by movement")
	}
}

func TestRegistry_StaleBackfillDiscarded(t *testing.T) {
	r := newTestRegistry()
	tm := newTestTenant(t)
	v := r.View(tm, 7)
	// A concurrent invalidation (any mutation) bumps the gen…
	r.InvalidateCore(tm, 7)
	// …so the in-flight backfill recorded at v.CoreGen must be discarded.
	if r.BackfillCore(tm, 7, testCore(t, 7), v.CoreGen) {
		t.Fatalf("stale backfill must be discarded after a concurrent mutation")
	}
	after := r.View(tm, 7)
	if after.CoreValid {
		t.Fatalf("discarded backfill must leave the component invalid")
	}
}

func TestRegistry_ApplyStatChanged_CompleteValuesAppliesInPlace(t *testing.T) {
	r := newTestRegistry()
	tm := newTestTenant(t)
	populate(t, r, tm, 7)
	// JSON round-trip delivers float64.
	r.ApplyStatChanged(tm, 7, []stat.Type{stat.TypeMp}, map[string]interface{}{"mp": float64(463)})
	v := r.View(tm, 7)
	if !v.CoreValid {
		t.Fatalf("complete values must not invalidate")
	}
	if v.Core.Mp() != 463 {
		t.Fatalf("mp not applied: %d", v.Core.Mp())
	}
	// Redelivery is idempotent (absolute value).
	r.ApplyStatChanged(tm, 7, []stat.Type{stat.TypeMp}, map[string]interface{}{"mp": float64(463)})
	if got := r.View(tm, 7); got.Core.Mp() != 463 {
		t.Fatalf("redelivery corrupted mp: %d", got.Core.Mp())
	}
}

func TestRegistry_ApplyStatChanged_MissingOrUnappliableValueInvalidates(t *testing.T) {
	cases := []struct {
		name    string
		updates []stat.Type
		values  map[string]interface{}
	}{
		{"nil values", []stat.Type{stat.TypeMp}, nil},
		{"missing key", []stat.Type{stat.TypeMp, stat.TypeHp}, map[string]interface{}{"mp": float64(1)}},
		{"non-numeric", []stat.Type{stat.TypeMp}, map[string]interface{}{"mp": "463"}},
		{"available_sp unappliable", []stat.Type{stat.TypeAvailableSP}, map[string]interface{}{"available_sp": float64(3)}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := newTestRegistry()
			tm := newTestTenant(t)
			populate(t, r, tm, 7)
			r.ApplyStatChanged(tm, 7, tc.updates, tc.values)
			if v := r.View(tm, 7); v.CoreValid {
				t.Fatalf("must invalidate core")
			}
		})
	}
}

func TestRegistry_ApplyStatChanged_EmptyUpdatesIsNoOp(t *testing.T) {
	r := newTestRegistry()
	tm := newTestTenant(t)
	before := populate(t, r, tm, 7)
	r.ApplyStatChanged(tm, 7, []stat.Type{}, nil)
	after := r.View(tm, 7)
	if !after.CoreValid || after.CoreGen != before.CoreGen {
		t.Fatalf("empty updates must not mutate: before=%+v after=%+v", before, after)
	}
}

func TestRegistry_ApplyStatChanged_PetSnIsSkipped(t *testing.T) {
	r := newTestRegistry()
	tm := newTestTenant(t)
	populate(t, r, tm, 7)
	r.ApplyStatChanged(tm, 7, []stat.Type{stat.TypePetSn1}, nil)
	if v := r.View(tm, 7); !v.CoreValid {
		t.Fatalf("PET_SN updates are not core fields and must be skipped, not invalidate")
	}
}

func TestRegistry_SetLevelAndExperience_ApplyWhenValid(t *testing.T) {
	r := newTestRegistry()
	tm := newTestTenant(t)
	populate(t, r, tm, 7)
	r.SetLevel(tm, 7, 31)
	r.SetExperience(tm, 7, 123456)
	v := r.View(tm, 7)
	if v.Core.Level() != 31 || v.Core.Experience() != 123456 {
		t.Fatalf("level/exp not applied: %d/%d", v.Core.Level(), v.Core.Experience())
	}
}

func TestRegistry_InPlaceUpdateOnInvalidComponentOnlyBumpsGen(t *testing.T) {
	r := newTestRegistry()
	tm := newTestTenant(t)
	populate(t, r, tm, 7)
	r.InvalidateCore(tm, 7)
	g := r.View(tm, 7).CoreGen
	r.SetLevel(tm, 7, 99)
	v := r.View(tm, 7)
	if v.CoreValid {
		t.Fatalf("in-place update must not validate a stale component")
	}
	if v.CoreGen == g {
		t.Fatalf("in-place update on invalid component must still bump gen (kills in-flight backfill)")
	}
}

func TestRegistry_SkillUpsertRemoveAndCooldownPreserved(t *testing.T) {
	r := newTestRegistry()
	tm := newTestTenant(t)
	populate(t, r, tm, 7)
	up := skill.NewModelBuilder(skillconst.Id(3121004)).SetLevel(20).SetMasterLevel(30).MustBuild()
	r.UpsertSkill(tm, 7, up)
	v := r.View(tm, 7)
	if len(v.Skills) != 1 || v.Skills[0].Level() != 20 || v.Skills[0].MasterLevel() != 30 {
		t.Fatalf("upsert mismatch: %+v", v.Skills)
	}
	newSkill := skill.NewModelBuilder(skillconst.Id(3121002)).SetLevel(1).MustBuild()
	r.UpsertSkill(tm, 7, newSkill)
	if v = r.View(tm, 7); len(v.Skills) != 2 {
		t.Fatalf("insert mismatch: %+v", v.Skills)
	}
	r.RemoveSkill(tm, 7, skillconst.Id(3121004))
	if v = r.View(tm, 7); len(v.Skills) != 1 || v.Skills[0].Id() != skillconst.Id(3121002) {
		t.Fatalf("remove mismatch: %+v", v.Skills)
	}
}

func TestRegistry_AssetLifecycle(t *testing.T) {
	r := newTestRegistry()
	tm := newTestTenant(t)
	v := populate(t, r, tm, 7)
	compId := v.Inv.Consumable().Id()

	// Quantity absolute (idempotent).
	r.SetAssetQuantity(tm, 7, 9001, 380)
	r.SetAssetQuantity(tm, 7, 9001, 380)
	got := r.View(tm, 7)
	a, ok := got.Inv.Consumable().FindById(9001)
	if !ok || a.Quantity() != 380 {
		t.Fatalf("quantity not applied: %+v ok=%v", a, ok)
	}

	// Slot absolute by AssetId.
	r.SetAssetSlot(tm, 7, 9001, 5)
	got = r.View(tm, 7)
	a, _ = got.Inv.Consumable().FindById(9001)
	if a.Slot() != 5 {
		t.Fatalf("slot not applied: %d", a.Slot())
	}

	// Upsert (full replace by AssetId) + insert of a new asset.
	repl := asset.NewModelBuilder(9001, compId, 2061000).SetSlot(5).SetQuantity(111).MustBuild()
	r.UpsertAsset(tm, 7, compId, repl)
	ins := asset.NewModelBuilder(9002, compId, 2060001).SetSlot(6).SetQuantity(200).MustBuild()
	r.UpsertAsset(tm, 7, compId, ins)
	got = r.View(tm, 7)
	if len(got.Inv.Consumable().Assets()) != 2 {
		t.Fatalf("expected 2 assets: %+v", got.Inv.Consumable().Assets())
	}
	a, _ = got.Inv.Consumable().FindById(9001)
	if a.TemplateId() != 2061000 || a.Quantity() != 111 {
		t.Fatalf("upsert replace mismatch: %+v", a)
	}

	// Upsert into an unknown compartment invalidates instead of guessing.
	r.UpsertAsset(tm, 7, uuid.New(), ins)
	if got = r.View(tm, 7); got.InvValid {
		t.Fatalf("unknown compartment must invalidate inventory")
	}

	// Refill, then delete.
	inv2, _, _ := testInventory(t, 7)
	if !r.BackfillInventory(tm, 7, inv2, got.InvGen) {
		t.Fatalf("re-backfill rejected")
	}
	r.RemoveAsset(tm, 7, 9001)
	got = r.View(tm, 7)
	if _, ok = got.Inv.Consumable().FindById(9001); ok {
		t.Fatalf("delete must remove the asset")
	}
}

func TestRegistry_AssetMutationDoesNotAliasPriorReads(t *testing.T) {
	// A composed/read model handed out earlier must not observe later
	// mutations (value-copy semantics; guards the shared-map/slice hazard in
	// inventory.CloneModel / compartment.CloneModel).
	r := newTestRegistry()
	tm := newTestTenant(t)
	populate(t, r, tm, 7)
	before := r.View(tm, 7)
	beforeQty := before.Inv.Consumable().Assets()[0].Quantity()
	r.SetAssetQuantity(tm, 7, 9001, 42)
	if got := before.Inv.Consumable().Assets()[0].Quantity(); got != beforeQty {
		t.Fatalf("prior read mutated in place: %d -> %d", beforeQty, got)
	}
}

func TestRegistry_BuffUpsertRemoveBySourceId(t *testing.T) {
	r := newTestRegistry()
	tm := newTestTenant(t)
	populate(t, r, tm, 7)
	b := buff.NewBuff(3111004, 20, 60_000, nil, timeNowForTest(), timeNowForTest().Add(60_000_000_000))
	r.UpsertBuff(tm, 7, b)
	r.UpsertBuff(tm, 7, b) // redelivery: replace, not duplicate
	v := r.View(tm, 7)
	if len(v.Buffs) != 1 {
		t.Fatalf("upsert must replace by sourceId: %+v", v.Buffs)
	}
	r.RemoveBuff(tm, 7, 3111004)
	if v = r.View(tm, 7); len(v.Buffs) != 0 {
		t.Fatalf("remove mismatch: %+v", v.Buffs)
	}
}

func TestRegistry_PositionOverlayAndComposedCache(t *testing.T) {
	r := newTestRegistry()
	tm := newTestTenant(t)
	populate(t, r, tm, 7)
	if _, ok := r.ComposedIfValid(tm, 7); !ok {
		t.Fatalf("all-valid entry must compose")
	}
	r.SetPosition(tm, 7, 333, -444)
	m, ok := r.ComposedIfValid(tm, 7)
	if !ok {
		t.Fatalf("compose after position update")
	}
	if m.X() != 333 || m.Y() != -444 {
		t.Fatalf("position overlay not applied: %d/%d", m.X(), m.Y())
	}
	// Composition preserves the decorated shape: inventory + skills present.
	if len(m.Skills()) != 1 || len(m.Inventory().Consumable().Assets()) != 1 {
		t.Fatalf("composed model missing decorations")
	}
	r.InvalidatePosition(tm, 7)
	m, ok = r.ComposedIfValid(tm, 7)
	if !ok {
		t.Fatalf("pos-invalid entry still composes from core X/Y")
	}
	if m.X() != 100 || m.Y() != -50 {
		t.Fatalf("expected core REST X/Y when position invalid: %d/%d", m.X(), m.Y())
	}
}

func TestRegistry_ComposedNotServedWhenComponentInvalid(t *testing.T) {
	r := newTestRegistry()
	tm := newTestTenant(t)
	populate(t, r, tm, 7)
	r.InvalidateInventory(tm, 7)
	if _, ok := r.ComposedIfValid(tm, 7); ok {
		t.Fatalf("must not serve composed model over an invalid component")
	}
}

func TestRegistry_EvictAndTenantIsolation(t *testing.T) {
	r := newTestRegistry()
	t1 := newTestTenant(t)
	t2 := newTestTenant(t)
	populate(t, r, t1, 7)
	populate(t, r, t2, 7)
	r.Evict(t1, 7)
	if _, ok := r.ComposedIfValid(t1, 7); ok {
		t.Fatalf("evicted entry must be gone")
	}
	if _, ok := r.ComposedIfValid(t2, 7); !ok {
		t.Fatalf("t2 must survive t1 evict")
	}
	r.EvictTenant(t2.Id())
	if _, ok := r.ComposedIfValid(t2, 7); ok {
		t.Fatalf("tenant evict must drop entries")
	}
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	r := newTestRegistry()
	tm := newTestTenant(t)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				id := uint32(j%5 + 1)
				v := r.View(tm, id)
				_ = r.BackfillCore(tm, id, testCore(t, id), v.CoreGen)
				r.ApplyStatChanged(tm, id, []stat.Type{stat.TypeMp}, map[string]interface{}{"mp": float64(j)})
				r.SetPosition(tm, id, int16(j), int16(-j))
				_, _ = r.ComposedIfValid(tm, id)
				if j%50 == 0 {
					r.Evict(tm, id)
				}
			}
		}(i)
	}
	wg.Wait()
}
```

Add this small helper at the bottom of the test file (keeps `time` usage in one place):

```go
func timeNowForTest() time.Time { return time.Now() }
```
(add `"time"` to the test imports).

- [x] **Step 3: Run tests to verify they fail**

Run (from `services/atlas-channel/atlas.com/channel`): `go test ./character/snapshot/ -v`
Expected: FAIL to compile — `Registry`, `entry`, `ComponentView` undefined (and `SetX` undefined if Step 1 skipped).

- [x] **Step 4: Implement metrics.go**

Create `services/atlas-channel/atlas.com/channel/character/snapshot/metrics.go`:

```go
package snapshot

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const (
	componentCore      = "core"
	componentSkills    = "skills"
	componentInventory = "inventory"
	componentBuffs     = "buffs"
	componentPosition  = "position"

	outcomeHit             = "hit"
	outcomeMiss            = "miss"
	outcomeFallbackSuccess = "fallback_success"
	outcomeFallbackFailure = "fallback_failure"

	kindEventUpdate       = "event_update"
	kindInvalidation      = "invalidation"
	kindBackfill          = "backfill"
	kindBackfillDiscarded = "backfill_discarded"
)

var (
	snapshotReadsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "atlas_channel_char_snapshot_reads_total",
			Help: "Character snapshot reads by tenant, component, and outcome.",
		},
		[]string{"tenant", "component", "outcome"},
	)

	snapshotUpdatesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "atlas_channel_char_snapshot_updates_total",
			Help: "Character snapshot state transitions by tenant, component, and kind.",
		},
		[]string{"tenant", "component", "kind"},
	)
)

func recordRead(t tenant.Model, component, outcome string) {
	snapshotReadsTotal.WithLabelValues(t.Id().String(), component, outcome).Inc()
}

func recordUpdate(t tenant.Model, component, kind string) {
	snapshotUpdatesTotal.WithLabelValues(t.Id().String(), component, kind).Inc()
}
```

- [x] **Step 5: Implement registry.go**

Create `services/atlas-channel/atlas.com/channel/character/snapshot/registry.go`:

```go
// Package snapshot holds the session-scoped character snapshot (task-122,
// PS-1): a per-pod, tenant-scoped projection of character core / skills /
// inventory / buffs plus a locally-fed position overlay, maintained from
// Kafka events and REST miss-fallbacks. Entries are created ONLY by the
// lazy read path (View) and evicted with the session; event mutators are
// update-only. Every mutation bumps the touched component's generation so
// an in-flight REST backfill recorded against an older generation is
// discarded instead of clobbering newer event-driven state (design §4.4).
package snapshot

import (
	"sync"

	"atlas-channel/asset"
	"atlas-channel/character"
	"atlas-channel/character/buff"
	"atlas-channel/character/skill"
	"atlas-channel/compartment"
	"atlas-channel/inventory"

	skillconst "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-constants/stat"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
)

type entry struct {
	core      character.Model
	coreGen   uint64
	coreValid bool

	skills      []skill.Model
	skillsGen   uint64
	skillsValid bool

	inv      inventory.Model
	invGen   uint64
	invValid bool

	buffs      []buff.Model
	buffsGen   uint64
	buffsValid bool

	posX, posY int16
	posValid   bool

	composed      character.Model
	composedValid bool
}

type Registry struct {
	mu        sync.RWMutex
	perTenant map[uuid.UUID]map[uint32]*entry
}

var (
	registryOnce sync.Once
	registry     *Registry
)

func GetRegistry() *Registry {
	registryOnce.Do(func() {
		registry = &Registry{perTenant: map[uuid.UUID]map[uint32]*entry{}}
	})
	return registry
}

// ComponentView is a value-copy view of one entry's components, validity,
// and generations, used by the read path to fetch and backfill outside the
// lock.
type ComponentView struct {
	Core      character.Model
	CoreValid bool
	CoreGen   uint64

	Skills      []skill.Model
	SkillsValid bool
	SkillsGen   uint64

	Inv      inventory.Model
	InvValid bool
	InvGen   uint64

	Buffs      []buff.Model
	BuffsValid bool
	BuffsGen   uint64

	PosX, PosY int16
	PosValid   bool
}

// View returns the current component view for characterId, creating an
// all-invalid entry when absent. This is the ONLY call that creates
// entries: events must never create them (consumers start at LastOffset,
// so an event-created entry would be a partial hallucination).
func (r *Registry) View(t tenant.Model, characterId uint32) ComponentView {
	r.mu.Lock()
	defer r.mu.Unlock()
	e := r.entryLocked(t, characterId, true)
	return ComponentView{
		Core: e.core, CoreValid: e.coreValid, CoreGen: e.coreGen,
		Skills: e.skills, SkillsValid: e.skillsValid, SkillsGen: e.skillsGen,
		Inv: e.inv, InvValid: e.invValid, InvGen: e.invGen,
		Buffs: e.buffs, BuffsValid: e.buffsValid, BuffsGen: e.buffsGen,
		PosX: e.posX, PosY: e.posY, PosValid: e.posValid,
	}
}

// entryLocked fetches (optionally creating) the entry. Caller holds mu.
func (r *Registry) entryLocked(t tenant.Model, characterId uint32, create bool) *entry {
	tm, ok := r.perTenant[t.Id()]
	if !ok {
		if !create {
			return nil
		}
		tm = map[uint32]*entry{}
		r.perTenant[t.Id()] = tm
	}
	e, ok := tm[characterId]
	if !ok {
		if !create {
			return nil
		}
		e = &entry{}
		tm[characterId] = e
	}
	return e
}

// ComposedIfValid returns the composed decorated model when every
// component is valid, rebuilding the cached composition if stale. Position
// is an overlay: when invalid, the core model's REST-sourced X/Y are used
// (exactly today's source).
func (r *Registry) ComposedIfValid(t tenant.Model, characterId uint32) (character.Model, bool) {
	r.mu.RLock()
	e := r.entryLocked(t, characterId, false)
	if e == nil || !(e.coreValid && e.skillsValid && e.invValid && e.buffsValid) {
		r.mu.RUnlock()
		return character.Model{}, false
	}
	if e.composedValid {
		m := e.composed
		r.mu.RUnlock()
		return m, true
	}
	r.mu.RUnlock()

	r.mu.Lock()
	defer r.mu.Unlock()
	e = r.entryLocked(t, characterId, false)
	if e == nil || !(e.coreValid && e.skillsValid && e.invValid && e.buffsValid) {
		return character.Model{}, false
	}
	if !e.composedValid {
		e.composed = composeLocked(e)
		e.composedValid = true
	}
	return e.composed, true
}

// composeLocked builds the same decorated model
// GetById(InventoryDecorator, SkillModelDecorator) returns today:
// base core -> position overlay -> SetInventory (rebuilds the equipment
// map from negative slots, character/model.go:284) -> SetSkills.
func composeLocked(e *entry) character.Model {
	m := e.core
	if e.posValid {
		m = character.CloneModel(m).SetX(e.posX).SetY(e.posY).MustBuild()
	}
	m = m.SetInventory(e.inv)
	m = m.SetSkills(e.skills)
	return m
}

// --- Backfills (generation-checked) -----------------------------------------

func (r *Registry) BackfillCore(t tenant.Model, characterId uint32, m character.Model, gen uint64) bool {
	return r.backfill(t, characterId, componentCore,
		func(e *entry) bool { return e.coreGen == gen },
		func(e *entry) { e.core = m; e.coreValid = true })
}

func (r *Registry) BackfillSkills(t tenant.Model, characterId uint32, ms []skill.Model, gen uint64) bool {
	cp := append([]skill.Model(nil), ms...)
	return r.backfill(t, characterId, componentSkills,
		func(e *entry) bool { return e.skillsGen == gen },
		func(e *entry) { e.skills = cp; e.skillsValid = true })
}

func (r *Registry) BackfillInventory(t tenant.Model, characterId uint32, inv inventory.Model, gen uint64) bool {
	return r.backfill(t, characterId, componentInventory,
		func(e *entry) bool { return e.invGen == gen },
		func(e *entry) { e.inv = inv; e.invValid = true })
}

func (r *Registry) BackfillBuffs(t tenant.Model, characterId uint32, bs []buff.Model, gen uint64) bool {
	cp := append([]buff.Model(nil), bs...)
	return r.backfill(t, characterId, componentBuffs,
		func(e *entry) bool { return e.buffsGen == gen },
		func(e *entry) { e.buffs = cp; e.buffsValid = true })
}

func (r *Registry) backfill(t tenant.Model, characterId uint32, component string, genOK func(*entry) bool, apply func(*entry)) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	e := r.entryLocked(t, characterId, false)
	if e == nil {
		// Entry evicted while the fetch was in flight (session destroyed):
		// do not resurrect it.
		recordUpdate(t, component, kindBackfillDiscarded)
		return false
	}
	if !genOK(e) {
		recordUpdate(t, component, kindBackfillDiscarded)
		return false
	}
	apply(e)
	e.composedValid = false
	recordUpdate(t, component, kindBackfill)
	return true
}

// --- Update-only event mutators ----------------------------------------------
//
// mutate runs f against an existing entry (no-op when absent), bumps the
// component generation via bump, and clears the composed cache. In-place
// updates must apply only when the component is valid; when it is invalid
// the gen bump alone matters (it kills any in-flight backfill so the next
// read refetches state that includes this event's effect).

func (r *Registry) mutate(t tenant.Model, characterId uint32, f func(e *entry)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	e := r.entryLocked(t, characterId, false)
	if e == nil {
		return
	}
	f(e)
	e.composedValid = false
}

// statValueKeys maps stat update types to the snake_case Values keys the
// atlas-character producer uses (convention pinned by the existing
// populated sites; task-122 event-coverage.md §1).
var statValueKeys = map[stat.Type]string{
	stat.TypeSkin:               "skin",
	stat.TypeFace:               "face",
	stat.TypeHair:               "hair",
	stat.TypeLevel:              "level",
	stat.TypeJob:                "job",
	stat.TypeStrength:           "strength",
	stat.TypeDexterity:          "dexterity",
	stat.TypeIntelligence:       "intelligence",
	stat.TypeLuck:               "luck",
	stat.TypeHp:                 "hp",
	stat.TypeMaxHp:              "max_hp",
	stat.TypeMp:                 "mp",
	stat.TypeMaxMp:              "max_mp",
	stat.TypeAvailableAP:        "available_ap",
	stat.TypeExperience:         "experience",
	stat.TypeFame:               "fame",
	stat.TypeMeso:               "meso",
	stat.TypeGachaponExperience: "gachapon_experience",
}

// applyStat returns the model with one absolute stat value applied. The
// second return is false for types the snapshot cannot apply in place
// (AVAILABLE_SP is a per-book string table on the model; unknown types are
// fail-safe).
func applyStat(m character.Model, u stat.Type, v float64) (character.Model, bool) {
	switch u {
	case stat.TypeSkin:
		return character.CloneModel(m).SetSkinColor(byte(v)).MustBuild(), true
	case stat.TypeFace:
		return character.CloneModel(m).SetFace(uint32(v)).MustBuild(), true
	case stat.TypeHair:
		return character.CloneModel(m).SetHair(uint32(v)).MustBuild(), true
	case stat.TypeLevel:
		return character.CloneModel(m).SetLevel(byte(v)).MustBuild(), true
	case stat.TypeJob:
		return character.CloneModel(m).SetJobId(job.Id(uint16(v))).MustBuild(), true
	case stat.TypeStrength:
		return character.CloneModel(m).SetStrength(uint16(v)).MustBuild(), true
	case stat.TypeDexterity:
		return character.CloneModel(m).SetDexterity(uint16(v)).MustBuild(), true
	case stat.TypeIntelligence:
		return character.CloneModel(m).SetIntelligence(uint16(v)).MustBuild(), true
	case stat.TypeLuck:
		return character.CloneModel(m).SetLuck(uint16(v)).MustBuild(), true
	case stat.TypeHp:
		return character.CloneModel(m).SetHp(uint16(v)).MustBuild(), true
	case stat.TypeMaxHp:
		return character.CloneModel(m).SetMaxHp(uint16(v)).MustBuild(), true
	case stat.TypeMp:
		return character.CloneModel(m).SetMp(uint16(v)).MustBuild(), true
	case stat.TypeMaxMp:
		return character.CloneModel(m).SetMaxMp(uint16(v)).MustBuild(), true
	case stat.TypeAvailableAP:
		return character.CloneModel(m).SetAp(uint16(v)).MustBuild(), true
	case stat.TypeExperience:
		return character.CloneModel(m).SetExperience(uint32(v)).MustBuild(), true
	case stat.TypeFame:
		return character.CloneModel(m).SetFame(int16(v)).MustBuild(), true
	case stat.TypeMeso:
		return character.CloneModel(m).SetMeso(uint32(v)).MustBuild(), true
	case stat.TypeGachaponExperience:
		return character.CloneModel(m).SetGachaponExperience(uint32(v)).MustBuild(), true
	}
	return m, false
}

// isPetSnStat reports stat types that are not fields of the base character
// model (pets arrive only via decorators the attack path never applies) —
// skipped rather than invalidating.
func isPetSnStat(u stat.Type) bool {
	return u == stat.TypePetSn1 || u == stat.TypePetSn2 || u == stat.TypePetSn3
}

// ApplyStatChanged applies a STAT_CHANGED event: complete absolute Values
// update the core in place; anything less invalidates the core component
// (invalidate-and-refetch — never guess). Empty Updates (error-path
// emissions) are a no-op.
func (r *Registry) ApplyStatChanged(t tenant.Model, characterId uint32, updates []stat.Type, values map[string]interface{}) {
	if len(updates) == 0 {
		return
	}
	r.mutate(t, characterId, func(e *entry) {
		e.coreGen++
		if !e.coreValid {
			return
		}
		m := e.core
		for _, u := range updates {
			if isPetSnStat(u) {
				continue
			}
			key, known := statValueKeys[u]
			if !known {
				e.coreValid = false
				recordUpdate(t, componentCore, kindInvalidation)
				return
			}
			raw, present := values[key]
			if !present {
				e.coreValid = false
				recordUpdate(t, componentCore, kindInvalidation)
				return
			}
			f, isNum := raw.(float64)
			if !isNum {
				e.coreValid = false
				recordUpdate(t, componentCore, kindInvalidation)
				return
			}
			next, applied := applyStat(m, u, f)
			if !applied {
				e.coreValid = false
				recordUpdate(t, componentCore, kindInvalidation)
				return
			}
			m = next
		}
		e.core = m
		recordUpdate(t, componentCore, kindEventUpdate)
	})
}

func (r *Registry) SetLevel(t tenant.Model, characterId uint32, level byte) {
	r.mutate(t, characterId, func(e *entry) {
		e.coreGen++
		if !e.coreValid {
			return
		}
		e.core = character.CloneModel(e.core).SetLevel(level).MustBuild()
		recordUpdate(t, componentCore, kindEventUpdate)
	})
}

func (r *Registry) SetExperience(t tenant.Model, characterId uint32, exp uint32) {
	r.mutate(t, characterId, func(e *entry) {
		e.coreGen++
		if !e.coreValid {
			return
		}
		e.core = character.CloneModel(e.core).SetExperience(exp).MustBuild()
		recordUpdate(t, componentCore, kindEventUpdate)
	})
}

func (r *Registry) InvalidateCore(t tenant.Model, characterId uint32) {
	r.mutate(t, characterId, func(e *entry) {
		e.coreGen++
		e.coreValid = false
		recordUpdate(t, componentCore, kindInvalidation)
	})
}

func (r *Registry) UpsertSkill(t tenant.Model, characterId uint32, sm skill.Model) {
	r.mutate(t, characterId, func(e *entry) {
		e.skillsGen++
		if !e.skillsValid {
			return
		}
		out := make([]skill.Model, 0, len(e.skills)+1)
		replaced := false
		for _, s := range e.skills {
			if s.Id() == sm.Id() {
				// Preserve the cooldown the REST populate carried; skill
				// CREATED/UPDATED events do not include it (v1: cooldown
				// events are ignored, not in the attack read-set).
				out = append(out, skill.Clone(sm).SetCooldownExpiresAt(s.CooldownExpiresAt()).MustBuild())
				replaced = true
			} else {
				out = append(out, s)
			}
		}
		if !replaced {
			out = append(out, sm)
		}
		e.skills = out
		recordUpdate(t, componentSkills, kindEventUpdate)
	})
}

func (r *Registry) RemoveSkill(t tenant.Model, characterId uint32, skillId skillconst.Id) {
	r.mutate(t, characterId, func(e *entry) {
		e.skillsGen++
		if !e.skillsValid {
			return
		}
		out := make([]skill.Model, 0, len(e.skills))
		for _, s := range e.skills {
			if s.Id() != skillId {
				out = append(out, s)
			}
		}
		e.skills = out
		recordUpdate(t, componentSkills, kindEventUpdate)
	})
}

func (r *Registry) InvalidateSkills(t tenant.Model, characterId uint32) {
	r.mutate(t, characterId, func(e *entry) {
		e.skillsGen++
		e.skillsValid = false
		recordUpdate(t, componentSkills, kindInvalidation)
	})
}

// replaceCompartment rebuilds the inventory with comp swapped in. A fresh
// builder is mandatory: inventory.CloneModel shares the compartments map
// with the source model, so building through it would mutate models
// already handed to readers (see TestRegistry_AssetMutationDoesNotAliasPriorReads).
func replaceCompartment(inv inventory.Model, comp compartment.Model) inventory.Model {
	b := inventory.NewBuilder(inv.CharacterId())
	for _, c := range inv.Compartments() {
		if c.Id() == comp.Id() {
			b.SetCompartment(comp)
		} else {
			b.SetCompartment(c)
		}
	}
	return b.MustBuild()
}

// mutateAssetInInventory finds the compartment holding assetId, applies
// transform to a fresh copy of its asset slice, and swaps the rebuilt
// compartment in. Returns false when no compartment holds the asset.
func mutateAssetInInventory(inv inventory.Model, assetId uint32, transform func(a asset.Model) asset.Model) (inventory.Model, bool) {
	for _, c := range inv.Compartments() {
		if _, ok := c.FindById(assetId); !ok {
			continue
		}
		out := make([]asset.Model, 0, len(c.Assets()))
		for _, a := range c.Assets() {
			if a.Id() == assetId {
				out = append(out, transform(a))
			} else {
				out = append(out, a)
			}
		}
		return replaceCompartment(inv, compartment.CloneModel(c).SetAssets(out).MustBuild()), true
	}
	return inv, false
}

// UpsertAsset replaces (by AssetId) or inserts the full asset into the
// compartment identified by compartmentId. An unknown compartment
// invalidates the inventory component instead of guessing.
func (r *Registry) UpsertAsset(t tenant.Model, characterId uint32, compartmentId uuid.UUID, a asset.Model) {
	r.mutate(t, characterId, func(e *entry) {
		e.invGen++
		if !e.invValid {
			return
		}
		comp, ok := e.inv.CompartmentById(compartmentId)
		if !ok {
			e.invValid = false
			recordUpdate(t, componentInventory, kindInvalidation)
			return
		}
		out := make([]asset.Model, 0, len(comp.Assets())+1)
		replaced := false
		for _, existing := range comp.Assets() {
			if existing.Id() == a.Id() {
				out = append(out, a)
				replaced = true
			} else {
				out = append(out, existing)
			}
		}
		if !replaced {
			out = append(out, a)
		}
		e.inv = replaceCompartment(e.inv, compartment.CloneModel(comp).SetAssets(out).MustBuild())
		recordUpdate(t, componentInventory, kindEventUpdate)
	})
}

func (r *Registry) SetAssetQuantity(t tenant.Model, characterId uint32, assetId uint32, quantity uint32) {
	r.mutate(t, characterId, func(e *entry) {
		e.invGen++
		if !e.invValid {
			return
		}
		next, ok := mutateAssetInInventory(e.inv, assetId, func(a asset.Model) asset.Model {
			return asset.Clone(a).SetQuantity(quantity).MustBuild()
		})
		if !ok {
			e.invValid = false
			recordUpdate(t, componentInventory, kindInvalidation)
			return
		}
		e.inv = next
		recordUpdate(t, componentInventory, kindEventUpdate)
	})
}

func (r *Registry) SetAssetSlot(t tenant.Model, characterId uint32, assetId uint32, slot int16) {
	r.mutate(t, characterId, func(e *entry) {
		e.invGen++
		if !e.invValid {
			return
		}
		next, ok := mutateAssetInInventory(e.inv, assetId, func(a asset.Model) asset.Model {
			return asset.Clone(a).SetSlot(slot).MustBuild()
		})
		if !ok {
			e.invValid = false
			recordUpdate(t, componentInventory, kindInvalidation)
			return
		}
		e.inv = next
		recordUpdate(t, componentInventory, kindEventUpdate)
	})
}

func (r *Registry) RemoveAsset(t tenant.Model, characterId uint32, assetId uint32) {
	r.mutate(t, characterId, func(e *entry) {
		e.invGen++
		if !e.invValid {
			return
		}
		for _, c := range e.inv.Compartments() {
			if _, ok := c.FindById(assetId); !ok {
				continue
			}
			out := make([]asset.Model, 0, len(c.Assets()))
			for _, a := range c.Assets() {
				if a.Id() != assetId {
					out = append(out, a)
				}
			}
			e.inv = replaceCompartment(e.inv, compartment.CloneModel(c).SetAssets(out).MustBuild())
			recordUpdate(t, componentInventory, kindEventUpdate)
			return
		}
		// Deleting an asset we never held: harmless for correctness of the
		// planner (it can only over-refetch), but the state is suspect —
		// invalidate.
		e.invValid = false
		recordUpdate(t, componentInventory, kindInvalidation)
	})
}

func (r *Registry) InvalidateInventory(t tenant.Model, characterId uint32) {
	r.mutate(t, characterId, func(e *entry) {
		e.invGen++
		e.invValid = false
		recordUpdate(t, componentInventory, kindInvalidation)
	})
}

func (r *Registry) UpsertBuff(t tenant.Model, characterId uint32, b buff.Model) {
	r.mutate(t, characterId, func(e *entry) {
		e.buffsGen++
		if !e.buffsValid {
			return
		}
		out := make([]buff.Model, 0, len(e.buffs)+1)
		replaced := false
		for _, existing := range e.buffs {
			if existing.SourceId() == b.SourceId() {
				out = append(out, b)
				replaced = true
			} else {
				out = append(out, existing)
			}
		}
		if !replaced {
			out = append(out, b)
		}
		e.buffs = out
		recordUpdate(t, componentBuffs, kindEventUpdate)
	})
}

func (r *Registry) RemoveBuff(t tenant.Model, characterId uint32, sourceId int32) {
	r.mutate(t, characterId, func(e *entry) {
		e.buffsGen++
		if !e.buffsValid {
			return
		}
		out := make([]buff.Model, 0, len(e.buffs))
		for _, existing := range e.buffs {
			if existing.SourceId() != sourceId {
				out = append(out, existing)
			}
		}
		e.buffs = out
		recordUpdate(t, componentBuffs, kindEventUpdate)
	})
}

func (r *Registry) InvalidateBuffs(t tenant.Model, characterId uint32) {
	r.mutate(t, characterId, func(e *entry) {
		e.buffsGen++
		e.buffsValid = false
		recordUpdate(t, componentBuffs, kindInvalidation)
	})
}

// SetPosition feeds the locally-observed movement fold (zero hops; strictly
// fresher than the REST projection of the same packets — FR-2.5).
func (r *Registry) SetPosition(t tenant.Model, characterId uint32, x, y int16) {
	r.mutate(t, characterId, func(e *entry) {
		e.posX, e.posY = x, y
		e.posValid = true
		recordUpdate(t, componentPosition, kindEventUpdate)
	})
}

func (r *Registry) InvalidatePosition(t tenant.Model, characterId uint32) {
	r.mutate(t, characterId, func(e *entry) {
		e.posValid = false
		recordUpdate(t, componentPosition, kindInvalidation)
	})
}

// Evict removes one character's snapshot (session destroy: logout,
// disconnect, channel change all funnel through session.Processor.Destroy).
func (r *Registry) Evict(t tenant.Model, characterId uint32) {
	r.mu.Lock()
	defer r.mu.Unlock()
	tm, ok := r.perTenant[t.Id()]
	if !ok {
		return
	}
	delete(tm, characterId)
	if len(tm) == 0 {
		delete(r.perTenant, t.Id())
	}
}

// EvictTenant drops every entry for the tenant (listener drain).
func (r *Registry) EvictTenant(tid uuid.UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.perTenant, tid)
}
```

Add `"github.com/Chronicle20/atlas/libs/atlas-constants/job"` to the imports (used by `applyStat`'s `stat.TypeJob` case).

- [x] **Step 6: Run tests to verify they pass**

Run (from `services/atlas-channel/atlas.com/channel`): `go test -race ./character/snapshot/ ./character/... -v`
Expected: PASS — all new registry tests plus all pre-existing character-package tests.

- [x] **Step 7: Build, vet, commit**

Run (from `services/atlas-channel/atlas.com/channel`): `go build ./... && go vet ./...`
Expected: clean.

```bash
git add services/atlas-channel/atlas.com/channel/character/snapshot/registry.go \
        services/atlas-channel/atlas.com/channel/character/snapshot/metrics.go \
        services/atlas-channel/atlas.com/channel/character/snapshot/registry_test.go \
        services/atlas-channel/atlas.com/channel/character/skill/builder.go \
        services/atlas-channel/atlas.com/channel/character/builder.go
git commit -m "feat(task-122): session-scoped character snapshot registry with generation-guarded components"
```

---

### Task 3: Snapshot processor — Get, composition, per-component fallback + backfill, GetBuffs

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/character/snapshot/processor.go`
- Create: `services/atlas-channel/atlas.com/channel/character/snapshot/processor_test.go`

**Interfaces:**
- Consumes (Task 2): `GetRegistry()`, `View`, `ComposedIfValid`, `Backfill*`, metrics helpers/constants.
- Consumes (existing): `character.NewProcessor(l, ctx).GetById()` (base, no decorators — `character/processor.go:65`), `inventory.NewProcessor(l, ctx).GetByCharacterId` (`inventory/processor.go`), `skill.NewProcessor(l, ctx).GetByCharacterId` (`character/skill/processor.go:41`), `buff.NewProcessor(l, ctx).GetByCharacterId` (`character/buff/processor.go:41`).
- Produces (used by Tasks 11–12):
  - `snapshot.NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor`
  - `(*Processor).Get(characterId uint32) (character.Model, error)` — the attack path's one read (FR-3.5 reusable API).
  - `(*Processor).GetBuffs(characterId uint32) ([]buff.Model, error)` and `(*Processor).BuffsProvider(characterId uint32) model.Provider[[]buff.Model]`
  - Package-level fetch seams (test-swappable): `coreFetchFn`, `inventoryFetchFn`, `skillsFetchFn`, `buffsFetchFn` — each `func(l logrus.FieldLogger, ctx context.Context, characterId uint32) (T, error)`.

- [x] **Step 1: Write the failing tests**

Create `services/atlas-channel/atlas.com/channel/character/snapshot/processor_test.go`:

```go
package snapshot

import (
	"context"
	"errors"
	"testing"

	"atlas-channel/character"
	"atlas-channel/character/buff"
	"atlas-channel/character/skill"
	"atlas-channel/inventory"

	skillconst "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type fetchCounts struct{ core, inv, skills, buffs int }

// installFetchSeams wires happy-path fakes and returns a counter. Callers
// override individual seams after this for failure cases.
func installFetchSeams(t *testing.T, characterId uint32) *fetchCounts {
	t.Helper()
	counts := &fetchCounts{}
	prevCore, prevInv, prevSkills, prevBuffs := coreFetchFn, inventoryFetchFn, skillsFetchFn, buffsFetchFn
	coreFetchFn = func(_ logrus.FieldLogger, _ context.Context, id uint32) (character.Model, error) {
		counts.core++
		return testCore(t, id), nil
	}
	inventoryFetchFn = func(_ logrus.FieldLogger, _ context.Context, id uint32) (inventory.Model, error) {
		counts.inv++
		inv, _, _ := testInventory(t, id)
		return inv, nil
	}
	skillsFetchFn = func(_ logrus.FieldLogger, _ context.Context, id uint32) ([]skill.Model, error) {
		counts.skills++
		return []skill.Model{skill.NewModelBuilder(skillconst.Id(3121004)).SetLevel(10).MustBuild()}, nil
	}
	buffsFetchFn = func(_ logrus.FieldLogger, _ context.Context, id uint32) ([]buff.Model, error) {
		counts.buffs++
		return []buff.Model{}, nil
	}
	t.Cleanup(func() {
		coreFetchFn, inventoryFetchFn, skillsFetchFn, buffsFetchFn = prevCore, prevInv, prevSkills, prevBuffs
	})
	return counts
}

func newTestProcessor(t *testing.T) (*Processor, tenant.Model) {
	t.Helper()
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	return NewProcessor(logrus.New(), ctx), tm
}

func TestProcessor_FirstGetPopulatesLazily_SecondGetIsZeroRest(t *testing.T) {
	resetRegistryForTest(t)
	p, _ := newTestProcessor(t)
	counts := installFetchSeams(t, 7)

	m, err := p.Get(7)
	if err != nil {
		t.Fatalf("first get: %v", err)
	}
	if m.Id() != 7 || len(m.Skills()) != 1 || len(m.Inventory().Consumable().Assets()) != 1 {
		t.Fatalf("first get returned undecorated model: %+v", m)
	}
	if counts.core != 1 || counts.inv != 1 || counts.skills != 1 {
		t.Fatalf("first get must fetch each component once: %+v", counts)
	}

	if _, err = p.Get(7); err != nil {
		t.Fatalf("second get: %v", err)
	}
	if counts.core != 1 || counts.inv != 1 || counts.skills != 1 {
		t.Fatalf("second get performed REST: %+v", counts)
	}
}

func TestProcessor_ComposedMatchesDecoratorPath(t *testing.T) {
	// FR-4.6 seed: the snapshot-composed model must equal the model built by
	// today's decorator chain for the same inputs.
	resetRegistryForTest(t)
	p, _ := newTestProcessor(t)
	installFetchSeams(t, 7)

	got, err := p.Get(7)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	inv, _, _ := testInventory(t, 7)
	want := testCore(t, 7).
		SetInventory(inv).
		SetSkills([]skill.Model{skill.NewModelBuilder(skillconst.Id(3121004)).SetLevel(10).MustBuild()})

	if got.Level() != want.Level() || got.JobId() != want.JobId() ||
		got.X() != want.X() || got.Y() != want.Y() {
		t.Fatalf("core mismatch: got %+v want %+v", got, want)
	}
	gw, gok := got.Equipment().Get("weapon")
	ww, wok := want.Equipment().Get("weapon")
	if gok != wok || (gok && gw.Equipable != nil) != (wok && ww.Equipable != nil) {
		t.Fatalf("equipment derivation mismatch")
	}
	if len(got.Skills()) != len(want.Skills()) || got.Skills()[0].Level() != want.Skills()[0].Level() {
		t.Fatalf("skills mismatch")
	}
	ga, wa := got.Inventory().Consumable().Assets(), want.Inventory().Consumable().Assets()
	if len(ga) != len(wa) || ga[0].TemplateId() != wa[0].TemplateId() || ga[0].Quantity() != wa[0].Quantity() || ga[0].Slot() != wa[0].Slot() {
		t.Fatalf("consumable assets mismatch: %+v vs %+v", ga, wa)
	}
}

func TestProcessor_PerComponentFallbackOnlyRefetchesInvalidComponent(t *testing.T) {
	resetRegistryForTest(t)
	p, tm := newTestProcessor(t)
	counts := installFetchSeams(t, 7)

	if _, err := p.Get(7); err != nil {
		t.Fatalf("populate: %v", err)
	}
	GetRegistry().InvalidateSkills(tm, 7)

	if _, err := p.Get(7); err != nil {
		t.Fatalf("refetch: %v", err)
	}
	if counts.skills != 2 {
		t.Fatalf("skills must refetch after invalidation: %+v", counts)
	}
	if counts.core != 1 || counts.inv != 1 {
		t.Fatalf("valid components must NOT refetch: %+v", counts)
	}
}

func TestProcessor_FallbackFailureSurfacesError(t *testing.T) {
	// FR-3.4: a REST fallback failure surfaces exactly as today's error path;
	// the snapshot never converts a hard failure into stale-success.
	resetRegistryForTest(t)
	p, tm := newTestProcessor(t)
	counts := installFetchSeams(t, 7)

	if _, err := p.Get(7); err != nil {
		t.Fatalf("populate: %v", err)
	}
	GetRegistry().InvalidateInventory(tm, 7)

	wantErr := errors.New("inventory service down")
	inventoryFetchFn = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (inventory.Model, error) {
		return inventory.Model{}, wantErr
	}
	if _, err := p.Get(7); !errors.Is(err, wantErr) {
		t.Fatalf("fallback failure must propagate, got %v", err)
	}
	_ = counts
}

func TestProcessor_StaleBackfillStillServesThisCaller(t *testing.T) {
	// Design §4.4: if an event bumps the gen mid-fetch, the backfill is
	// discarded but the fetched value is returned to THIS caller (it is
	// exactly what REST would have returned today).
	resetRegistryForTest(t)
	p, tm := newTestProcessor(t)
	installFetchSeams(t, 7)
	if _, err := p.Get(7); err != nil {
		t.Fatalf("populate: %v", err)
	}
	GetRegistry().InvalidateSkills(tm, 7)

	prev := skillsFetchFn
	skillsFetchFn = func(l logrus.FieldLogger, ctx context.Context, id uint32) ([]skill.Model, error) {
		// Concurrent event arrives while the fetch is in flight.
		GetRegistry().InvalidateSkills(tm, 7)
		return prev(l, ctx, id)
	}
	m, err := p.Get(7)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(m.Skills()) != 1 {
		t.Fatalf("caller must still receive the fetched skills: %+v", m.Skills())
	}
	if v := GetRegistry().View(tm, 7); v.SkillsValid {
		t.Fatalf("discarded backfill must leave the component invalid for the next read")
	}
}

func TestProcessor_PositionOverlayServesLocalFold(t *testing.T) {
	resetRegistryForTest(t)
	p, tm := newTestProcessor(t)
	installFetchSeams(t, 7)
	if _, err := p.Get(7); err != nil {
		t.Fatalf("populate: %v", err)
	}
	GetRegistry().SetPosition(tm, 7, 555, -666)
	m, err := p.Get(7)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if m.X() != 555 || m.Y() != -666 {
		t.Fatalf("position overlay not served: %d/%d", m.X(), m.Y())
	}
}

func TestProcessor_GetBuffs_LazySeedThenEventMaintained(t *testing.T) {
	resetRegistryForTest(t)
	p, tm := newTestProcessor(t)
	counts := installFetchSeams(t, 7)

	bs, err := p.GetBuffs(7)
	if err != nil {
		t.Fatalf("first buffs: %v", err)
	}
	if len(bs) != 0 || counts.buffs != 1 {
		t.Fatalf("first buffs read must seed via REST: %+v %+v", bs, counts)
	}

	b := buff.NewBuff(3111004, 20, 60_000, nil, timeNowForTest(), timeNowForTest().Add(60_000_000_000))
	GetRegistry().UpsertBuff(tm, 7, b)
	bs, err = p.GetBuffs(7)
	if err != nil {
		t.Fatalf("second buffs: %v", err)
	}
	if len(bs) != 1 || counts.buffs != 1 {
		t.Fatalf("second buffs read must be event-served, zero REST: %+v %+v", bs, counts)
	}
}

func TestProcessor_GetBuffs_SelfExpiresPastExpiresAt(t *testing.T) {
	// event-coverage.md §5: a lost EXPIRED event degrades to at most the
	// buff's natural duration — reads self-filter expired entries.
	resetRegistryForTest(t)
	p, tm := newTestProcessor(t)
	installFetchSeams(t, 7)
	if _, err := p.GetBuffs(7); err != nil {
		t.Fatalf("seed: %v", err)
	}
	expired := buff.NewBuff(3111004, 20, 1, nil, timeNowForTest().Add(-2_000_000_000), timeNowForTest().Add(-1_000_000_000))
	GetRegistry().UpsertBuff(tm, 7, expired)
	bs, err := p.GetBuffs(7)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(bs) != 0 {
		t.Fatalf("expired buffs must be filtered: %+v", bs)
	}
}
```

Add this helper to `registry_test.go` (same package; the singleton must be resettable between processor tests):

```go
// resetRegistryForTest swaps the singleton for a fresh registry.
func resetRegistryForTest(t *testing.T) {
	t.Helper()
	registryOnce.Do(func() {})
	registry = &Registry{perTenant: map[uuid.UUID]map[uint32]*entry{}}
}
```

- [x] **Step 2: Run tests to verify they fail**

Run (from `services/atlas-channel/atlas.com/channel`): `go test ./character/snapshot/ -run TestProcessor -v`
Expected: FAIL to compile — `Processor`, `coreFetchFn` undefined.

- [x] **Step 3: Implement processor.go** (landed with a deliberate, documented deviation from this step's literal code block: core fallback failure still errors, but inventory/skills fallback failure now degrades to a partial model instead of erroring — commit `844a59b72`, matching the real pre-existing `character.ProcessorImpl` decorator behavior; confirmed correct by the plan-adherence audit)

Create `services/atlas-channel/atlas.com/channel/character/snapshot/processor.go`:

```go
package snapshot

import (
	"context"

	"atlas-channel/character"
	"atlas-channel/character/buff"
	"atlas-channel/character/skill"
	"atlas-channel/inventory"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

// Fetch seams: the exact per-component REST providers today's decorator
// chain uses (character/processor.go GetById + InventoryDecorator +
// SkillModelDecorator; character/buff/processor.go). Package-level vars so
// tests prove zero-REST behavior without HTTP fakes (project precedent:
// monsterByIdFn in movement, task-120).
var (
	coreFetchFn = func(l logrus.FieldLogger, ctx context.Context, characterId uint32) (character.Model, error) {
		return character.NewProcessor(l, ctx).GetById()(characterId)
	}
	inventoryFetchFn = func(l logrus.FieldLogger, ctx context.Context, characterId uint32) (inventory.Model, error) {
		return inventory.NewProcessor(l, ctx).GetByCharacterId(characterId)
	}
	skillsFetchFn = func(l logrus.FieldLogger, ctx context.Context, characterId uint32) ([]skill.Model, error) {
		return skill.NewProcessor(l, ctx).GetByCharacterId(characterId)
	}
	buffsFetchFn = func(l logrus.FieldLogger, ctx context.Context, characterId uint32) ([]buff.Model, error) {
		return buff.NewProcessor(l, ctx).GetByCharacterId(characterId)
	}
)

// Processor is the snapshot's read API (FR-3.5): any handler can resolve a
// locally-sessioned character's decorated model or active buffs from
// in-process state, with per-component REST miss-fallback that backfills.
type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor {
	return &Processor{l: l, ctx: ctx, t: tenant.MustFromContext(ctx)}
}

// Get returns the composed decorated character model — the same shape
// cp.GetById(cp.InventoryDecorator, cp.SkillModelDecorator) returns today.
// Fast path: all components valid → in-process composition. Slow path:
// only invalid components are refetched over REST (counted), backfilled
// under a generation check, and the fetched values are served to this
// caller even if a concurrent event discarded the backfill.
func (p *Processor) Get(characterId uint32) (character.Model, error) {
	r := GetRegistry()
	if m, ok := r.ComposedIfValid(p.t, characterId); ok {
		recordRead(p.t, componentCore, outcomeHit)
		recordRead(p.t, componentSkills, outcomeHit)
		recordRead(p.t, componentInventory, outcomeHit)
		return m, nil
	}

	v := r.View(p.t, characterId)

	core := v.Core
	if v.CoreValid {
		recordRead(p.t, componentCore, outcomeHit)
	} else {
		recordRead(p.t, componentCore, outcomeMiss)
		fetched, err := coreFetchFn(p.l, p.ctx, characterId)
		if err != nil {
			recordRead(p.t, componentCore, outcomeFallbackFailure)
			p.l.WithError(err).Debugf("Snapshot core fallback failed for character [%d].", characterId)
			return character.Model{}, err
		}
		recordRead(p.t, componentCore, outcomeFallbackSuccess)
		p.l.Debugf("Snapshot fallback: core refetched for character [%d].", characterId)
		r.BackfillCore(p.t, characterId, fetched, v.CoreGen)
		core = fetched
	}

	inv := v.Inv
	if v.InvValid {
		recordRead(p.t, componentInventory, outcomeHit)
	} else {
		recordRead(p.t, componentInventory, outcomeMiss)
		fetched, err := inventoryFetchFn(p.l, p.ctx, characterId)
		if err != nil {
			recordRead(p.t, componentInventory, outcomeFallbackFailure)
			p.l.WithError(err).Debugf("Snapshot inventory fallback failed for character [%d].", characterId)
			return character.Model{}, err
		}
		recordRead(p.t, componentInventory, outcomeFallbackSuccess)
		p.l.Debugf("Snapshot fallback: inventory refetched for character [%d].", characterId)
		r.BackfillInventory(p.t, characterId, fetched, v.InvGen)
		inv = fetched
	}

	skills := v.Skills
	if v.SkillsValid {
		recordRead(p.t, componentSkills, outcomeHit)
	} else {
		recordRead(p.t, componentSkills, outcomeMiss)
		fetched, err := skillsFetchFn(p.l, p.ctx, characterId)
		if err != nil {
			recordRead(p.t, componentSkills, outcomeFallbackFailure)
			p.l.WithError(err).Debugf("Snapshot skills fallback failed for character [%d].", characterId)
			return character.Model{}, err
		}
		recordRead(p.t, componentSkills, outcomeFallbackSuccess)
		p.l.Debugf("Snapshot fallback: skills refetched for character [%d].", characterId)
		r.BackfillSkills(p.t, characterId, fetched, v.SkillsGen)
		skills = fetched
	}

	m := core
	if v.PosValid {
		m = character.CloneModel(m).SetX(v.PosX).SetY(v.PosY).MustBuild()
	}
	m = m.SetInventory(inv)
	m = m.SetSkills(skills)
	return m, nil
}

// GetBuffs returns the character's active buffs from the snapshot,
// lazy-seeding via REST on miss. Reads self-filter entries past their
// ExpiresAt (bounds the atlas-buffs-restart residual, event-coverage §5).
func (p *Processor) GetBuffs(characterId uint32) ([]buff.Model, error) {
	r := GetRegistry()
	v := r.View(p.t, characterId)
	if v.BuffsValid {
		recordRead(p.t, componentBuffs, outcomeHit)
		return filterActive(v.Buffs), nil
	}
	recordRead(p.t, componentBuffs, outcomeMiss)
	fetched, err := buffsFetchFn(p.l, p.ctx, characterId)
	if err != nil {
		recordRead(p.t, componentBuffs, outcomeFallbackFailure)
		return nil, err
	}
	recordRead(p.t, componentBuffs, outcomeFallbackSuccess)
	p.l.Debugf("Snapshot fallback: buffs refetched for character [%d].", characterId)
	r.BackfillBuffs(p.t, characterId, fetched, v.BuffsGen)
	return filterActive(fetched), nil
}

// BuffsProvider adapts GetBuffs to the provider shape used across the
// codebase (FR-3.5 naming per design §4.2).
func (p *Processor) BuffsProvider(characterId uint32) model.Provider[[]buff.Model] {
	return func() ([]buff.Model, error) {
		return p.GetBuffs(characterId)
	}
}

func filterActive(bs []buff.Model) []buff.Model {
	out := make([]buff.Model, 0, len(bs))
	for _, b := range bs {
		if b.Expired() {
			continue
		}
		out = append(out, b)
	}
	return out
}
```

- [x] **Step 4: Run tests to verify they pass**

Run (from `services/atlas-channel/atlas.com/channel`): `go test -race ./character/snapshot/ -v`
Expected: PASS.

- [x] **Step 5: Build, vet, commit**

Run (from `services/atlas-channel/atlas.com/channel`): `go build ./... && go vet ./...`
Expected: clean.

```bash
git add services/atlas-channel/atlas.com/channel/character/snapshot/processor.go \
        services/atlas-channel/atlas.com/channel/character/snapshot/processor_test.go \
        services/atlas-channel/atlas.com/channel/character/snapshot/registry_test.go
git commit -m "feat(task-122): snapshot read API with per-component REST fallback and backfill"
```

---

### Task 4: Lifecycle — session-destroy eviction, tenant evictor, movement position feed

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/session/processor.go:330-348` (`Destroy`)
- Modify: `services/atlas-channel/atlas.com/channel/main.go` (`listener.RegisterEvictor` block, ~line 287)
- Modify: `services/atlas-channel/atlas.com/channel/movement/processor.go:46-65` (`ForCharacter`)
- Test: `services/atlas-channel/atlas.com/channel/movement/processor_test.go` (append)

**Interfaces:**
- Consumes (Task 2): `snapshot.GetRegistry()`, `Evict`, `EvictTenant`, `SetPosition`, `View`.
- Produces: no new API — behavioral guarantees: snapshot evicted inside `session.Destroy` itself (design §11 — not a parallel code path), position fed synchronously before the movement Kafka emit (FR-2.5).

- [x] **Step 1: Write the failing movement test**

Append to `services/atlas-channel/atlas.com/channel/movement/processor_test.go` (this file exists post-task-120 with `newMovementTestTenant`/`newMovementTestProcessor`/`movementTestField` helpers — reuse them; add imports `atlas-channel/character/snapshot` and `github.com/Chronicle20/atlas/libs/atlas-packet/model` as needed):

```go
func TestForCharacter_FeedsSnapshotPositionSynchronously(t *testing.T) {
	p, tm := newMovementTestProcessor(t)
	f := movementTestField()

	// Entry must exist (events/feeds never create entries): simulate the
	// lazy populate by creating the entry, then validate core via backfill
	// so the position lands on a live entry.
	v := snapshot.GetRegistry().View(tm, 9001)
	_ = v

	mv := model.Movement{StartX: 10, StartY: 20}
	// No elements: the fold returns the start position.
	if err := p.ForCharacter(f, 9001, mv); err != nil {
		t.Fatalf("ForCharacter: %v", err)
	}

	got := snapshot.GetRegistry().View(tm, 9001)
	if !got.PosValid || got.PosX != 10 || got.PosY != 20 {
		t.Fatalf("position must be fed synchronously before ForCharacter returns: %+v", got)
	}
}

func TestForCharacter_NoEntryNoCreate(t *testing.T) {
	p, tm := newMovementTestProcessor(t)
	f := movementTestField()
	if err := p.ForCharacter(f, 9002, model.Movement{StartX: 1, StartY: 2}); err != nil {
		t.Fatalf("ForCharacter: %v", err)
	}
	// 9002 was never Viewed/populated — the feed must not create an entry.
	// View creates, so check via ComposedIfValid which does not.
	if _, ok := snapshot.GetRegistry().ComposedIfValid(tm, 9002); ok {
		t.Fatalf("position feed must never create snapshot entries")
	}
}
```

NOTE: `TestForCharacter_FeedsSnapshotPositionSynchronously` asserts the position lands even while other components are invalid — `SetPosition` is update-only against the *entry* (created by `View`), not against component validity.

NOTE: the goroutines inside `ForCharacter` will attempt map/producer calls that fail fast in tests and only log — the assertion is on the synchronous `SetPosition`, which runs before either goroutine is spawned.

- [x] **Step 2: Run tests to verify they fail**

Run (from `services/atlas-channel/atlas.com/channel`): `go test ./movement/ -run TestForCharacter -v`
Expected: FAIL — position not fed (PosValid false).

- [x] **Step 3: Implement the movement feed**

In `services/atlas-channel/atlas.com/channel/movement/processor.go`, replace `ForCharacter` (lines 46-65) with:

```go
func (p *Processor) ForCharacter(f field.Model, characterId uint32, movement model.Movement) error {
	// Fold the packet to its final position synchronously and feed the
	// character snapshot BEFORE anything else observes this packet: this is
	// the same fold the Kafka command carries, minus the Kafka->Redis->REST
	// round-trip — the freshest position this pod can know (task-122
	// FR-2.5). Update-only: characters without a snapshot entry no-op.
	ms, foldErr := model2.Fold(model2.FixedProvider(movement.Elements), summaryProvider(movement.StartX, movement.StartY, 0), folder)()
	if foldErr == nil {
		snapshot.GetRegistry().SetPosition(p.t, characterId, ms.X, ms.Y)
	}

	go func() {
		op := session.Announce(p.l)(p.ctx)(p.wp)(charpkt.CharacterMovementWriter)(charpkt.NewCharacterMovement(characterId, movement).Encode)
		err := _map2.NewProcessor(p.l, p.ctx).ForOtherSessionsInMap(f, characterId, op)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to move character [%d] for characters in map [%d].", characterId, f.MapId())
		}
	}()
	go func() {
		if foldErr != nil {
			return
		}
		err := producer.ProviderImpl(p.l)(p.ctx)(movement2.EnvCommandCharacterMovement)(CommandProducer(f, uint64(characterId), characterId, ms.X, ms.Y, ms.Fh, ms.Stance))
		if err != nil {
			p.l.WithError(err).Errorf("Unable to issue movement command [%d].", characterId)
		}
	}()
	return nil
}
```

Add import `"atlas-channel/character/snapshot"`. Behavioral invariant: the emitted Kafka command is byte-identical (same fold, same producer call); the broadcast goroutine is untouched; the only change is the fold moving out of the goroutine (it was already computed per packet).

- [x] **Step 4: Wire eviction**

In `services/atlas-channel/atlas.com/channel/session/processor.go`, inside `Destroy` (line 330), immediately after `getRegistry().Remove(p.t.Id(), s.SessionId())` (line 332), add:

```go
	// Session-scoped character snapshot dies with the session (task-122
	// FR-3.2): logout, disconnect, and channel change all funnel here.
	// CharacterId can be 0 for pre-login sessions; Evict no-ops then.
	snapshot.GetRegistry().Evict(p.t, s.CharacterId())
```

Add import `"atlas-channel/character/snapshot"` to session/processor.go.

In `services/atlas-channel/atlas.com/channel/main.go`, in the `listener.RegisterEvictor` block (~line 287), next to `monsterDomain.GetLiveMirror().EvictTenant(tid)`, add:

```go
		snapshot.GetRegistry().EvictTenant(tid)
```

Add import `snapshot "atlas-channel/character/snapshot"`.

- [x] **Step 5: Run tests, build, vet**

Run (from `services/atlas-channel/atlas.com/channel`): `go test -race ./movement/ ./session/ ./character/snapshot/ && go build ./... && go vet ./...`
Expected: PASS / clean. (session package has existing tests; the Destroy addition must not break them.)

- [x] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/session/processor.go \
        services/atlas-channel/atlas.com/channel/main.go \
        services/atlas-channel/atlas.com/channel/movement/processor.go \
        services/atlas-channel/atlas.com/channel/movement/processor_test.go
git commit -m "feat(task-122): snapshot lifecycle eviction and synchronous movement position feed"
```

---

### Task 5: Character-status consumer snapshot handlers

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/character/consumer.go`
- Create: `services/atlas-channel/atlas.com/channel/kafka/consumer/character/consumer_test.go`

**Interfaces:**
- Consumes (Task 2): `snapshot.GetRegistry()`: `ApplyStatChanged`, `SetLevel`, `SetExperience`, `SetPosition`, `InvalidatePosition`, `InvalidateCore`, `View` (tests), `BackfillCore` (tests).
- Consumes (existing): event bodies in `kafka/message/character/kafka.go` (`StatusEventStatChangedBody.Values`, `LevelChangedStatusEventBody.Current`, `ExperienceChangedStatusEventBody.Current`, `StatusEventMapChangedBody.UseTargetPosition/TargetX/TargetY`), registration pattern `consumer.go:47-88`.
- Produces: four new handler funcs registered in `InitHandlers`: `handleSnapshotStatChanged`, `handleSnapshotLevelChanged`, `handleSnapshotExperienceChanged`, `handleSnapshotMapChanged`. Existing handlers untouched (FR-3.3). No new MESO/FAME handlers: every meso/fame mutation site also emits STAT_CHANGED(Meso|Fame) (verified `atlas-character character/processor.go:751,768,796,813`), which this task routes through `ApplyStatChanged` (nil Values → invalidate today; rich after Task 13).

- [x] **Step 1: Write the failing tests**

Create `services/atlas-channel/atlas.com/channel/kafka/consumer/character/consumer_test.go`:

```go
package character

import (
	"context"
	"testing"

	character3 "atlas-channel/character"
	"atlas-channel/character/snapshot"
	character2 "atlas-channel/kafka/message/character"
	"atlas-channel/server"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/stat"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func newTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return tm
}

func newTestServer(t *testing.T, tm tenant.Model) server.Model {
	t.Helper()
	ch := channel.NewModel(0, 1)
	return server.Register(tm, ch, "127.0.0.1", 8484)
}

// seedSnapshotCore creates a snapshot entry and validates its core so
// in-place event updates have a base to apply to.
func seedSnapshotCore(t *testing.T, tm tenant.Model, characterId uint32) {
	t.Helper()
	v := snapshot.GetRegistry().View(tm, characterId)
	core := character3.NewModelBuilder().
		SetId(characterId).SetLevel(30).SetMp(500).SetMaxMp(800).
		MustBuild()
	if !snapshot.GetRegistry().BackfillCore(tm, characterId, core, v.CoreGen) {
		t.Fatalf("seed backfill rejected")
	}
}

func TestHandleSnapshotStatChanged_RichValuesApplyInPlace(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)
	seedSnapshotCore(t, tm, 41)

	e := character2.StatusEvent[character2.StatusEventStatChangedBody]{
		WorldId: 0, CharacterId: 41, Type: character2.StatusEventTypeStatChanged,
		Body: character2.StatusEventStatChangedBody{
			ChannelId: 1,
			Updates:   []stat.Type{stat.TypeMp},
			Values:    map[string]interface{}{"mp": float64(463)},
		},
	}
	handleSnapshotStatChanged(sc, nil)(logrus.New(), ctx, e)

	v := snapshot.GetRegistry().View(tm, 41)
	if !v.CoreValid || v.Core.Mp() != 463 {
		t.Fatalf("rich STAT_CHANGED must apply in place: valid=%v mp=%d", v.CoreValid, v.Core.Mp())
	}
}

func TestHandleSnapshotStatChanged_NilValuesInvalidates(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)
	seedSnapshotCore(t, tm, 42)

	e := character2.StatusEvent[character2.StatusEventStatChangedBody]{
		WorldId: 0, CharacterId: 42, Type: character2.StatusEventTypeStatChanged,
		Body: character2.StatusEventStatChangedBody{ChannelId: 1, Updates: []stat.Type{stat.TypeMp}},
	}
	handleSnapshotStatChanged(sc, nil)(logrus.New(), ctx, e)

	if v := snapshot.GetRegistry().View(tm, 42); v.CoreValid {
		t.Fatalf("nil-Values STAT_CHANGED must invalidate (rollout safety)")
	}
}

func TestHandleSnapshotLevelAndExperience_ApplyAbsolute(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)
	seedSnapshotCore(t, tm, 43)

	le := character2.StatusEvent[character2.LevelChangedStatusEventBody]{
		WorldId: 0, CharacterId: 43, Type: character2.StatusEventTypeLevelChanged,
		Body: character2.LevelChangedStatusEventBody{ChannelId: 1, Amount: 1, Current: 31},
	}
	handleSnapshotLevelChanged(sc, nil)(logrus.New(), ctx, le)

	ee := character2.StatusEvent[character2.ExperienceChangedStatusEventBody]{
		WorldId: 0, CharacterId: 43, Type: character2.StatusEventTypeExperienceChanged,
		Body: character2.ExperienceChangedStatusEventBody{ChannelId: 1, Current: 999},
	}
	handleSnapshotExperienceChanged(sc, nil)(logrus.New(), ctx, ee)

	v := snapshot.GetRegistry().View(tm, 43)
	if v.Core.Level() != 31 || v.Core.Experience() != 999 {
		t.Fatalf("level/exp not applied: %d/%d", v.Core.Level(), v.Core.Experience())
	}
}

func TestHandleSnapshotMapChanged_TargetPositionSetsOverlay(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)
	seedSnapshotCore(t, tm, 44)

	e := character2.StatusEvent[character2.StatusEventMapChangedBody]{
		WorldId: 0, CharacterId: 44, Type: character2.StatusEventTypeMapChanged,
		Body: character2.StatusEventMapChangedBody{
			ChannelId: 1, TargetMapId: 100000000,
			UseTargetPosition: true, TargetX: 77, TargetY: -88,
		},
	}
	handleSnapshotMapChanged(sc, nil)(logrus.New(), ctx, e)

	v := snapshot.GetRegistry().View(tm, 44)
	if !v.PosValid || v.PosX != 77 || v.PosY != -88 {
		t.Fatalf("UseTargetPosition must set the overlay: %+v", v)
	}
	if !v.CoreValid {
		t.Fatalf("UseTargetPosition path must not invalidate core (overlay covers X/Y)")
	}
}

func TestHandleSnapshotMapChanged_PortalWarpInvalidatesPositionAndCore(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)
	seedSnapshotCore(t, tm, 45)
	snapshot.GetRegistry().SetPosition(tm, 45, 1, 2)

	e := character2.StatusEvent[character2.StatusEventMapChangedBody]{
		WorldId: 0, CharacterId: 45, Type: character2.StatusEventTypeMapChanged,
		Body: character2.StatusEventMapChangedBody{ChannelId: 1, TargetMapId: 100000000},
	}
	handleSnapshotMapChanged(sc, nil)(logrus.New(), ctx, e)

	v := snapshot.GetRegistry().View(tm, 45)
	if v.PosValid {
		t.Fatalf("portal warp must invalidate the position overlay")
	}
	if v.CoreValid {
		t.Fatalf("portal warp must invalidate core so the next read refetches fresh REST X/Y (design §10.4)")
	}
}

func TestSnapshotHandlers_IgnoreOtherWorlds(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm) // world 0
	seedSnapshotCore(t, tm, 46)

	e := character2.StatusEvent[character2.StatusEventStatChangedBody]{
		WorldId: 3, CharacterId: 46, Type: character2.StatusEventTypeStatChanged,
		Body: character2.StatusEventStatChangedBody{ChannelId: 1, Updates: []stat.Type{stat.TypeMp}},
	}
	handleSnapshotStatChanged(sc, nil)(logrus.New(), ctx, e)
	if v := snapshot.GetRegistry().View(tm, 46); !v.CoreValid {
		t.Fatalf("other-world events must be ignored")
	}
}
```

NOTE: check `server.Register`'s actual signature in `server/` (the monster consumer tests use `server.Register(tm, ch, "127.0.0.1", 8484)` — reuse exactly that form) and `channel.NewModel(0, 1)` argument types (`world.Id`, `channel.Id` are both `byte`-backed). Adjust literals to compile; do not change the assertions.

- [x] **Step 2: Run tests to verify they fail**

Run (from `services/atlas-channel/atlas.com/channel`): `go test ./kafka/consumer/character/ -v`
Expected: FAIL to compile — `handleSnapshotStatChanged` undefined.

- [x] **Step 3: Implement the handlers**

In `services/atlas-channel/atlas.com/channel/kafka/consumer/character/consumer.go`:

1. Add import `"atlas-channel/character/snapshot"`.

2. Append the four handlers at the end of the file:

```go
// --- task-122 snapshot maintenance (additive; packet behavior above is untouched) ---

// handleSnapshotStatChanged projects STAT_CHANGED into the character
// snapshot: complete absolute Values apply in place; anything less
// invalidates the core component (invalidate-and-refetch, never guess).
func handleSnapshotStatChanged(sc server.Model, _ writer.Producer) message.Handler[character2.StatusEvent[character2.StatusEventStatChangedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.StatusEventStatChangedBody]) {
		if e.Type != character2.StatusEventTypeStatChanged {
			return
		}
		t := tenant.MustFromContext(ctx)
		if !sc.IsWorld(t, e.WorldId) {
			return
		}
		snapshot.GetRegistry().ApplyStatChanged(t, e.CharacterId, e.Body.Updates, e.Body.Values)
	}
}

func handleSnapshotLevelChanged(sc server.Model, _ writer.Producer) message.Handler[character2.StatusEvent[character2.LevelChangedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.LevelChangedStatusEventBody]) {
		if e.Type != character2.StatusEventTypeLevelChanged {
			return
		}
		t := tenant.MustFromContext(ctx)
		if !sc.IsWorld(t, e.WorldId) {
			return
		}
		snapshot.GetRegistry().SetLevel(t, e.CharacterId, e.Body.Current)
	}
}

func handleSnapshotExperienceChanged(sc server.Model, _ writer.Producer) message.Handler[character2.StatusEvent[character2.ExperienceChangedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.ExperienceChangedStatusEventBody]) {
		if e.Type != character2.StatusEventTypeExperienceChanged {
			return
		}
		t := tenant.MustFromContext(ctx)
		if !sc.IsWorld(t, e.WorldId) {
			return
		}
		snapshot.GetRegistry().SetExperience(t, e.CharacterId, e.Body.Current)
	}
}

// handleSnapshotMapChanged keeps the position overlay honest across warps:
// Mystic-Door-style warps carry exact coordinates; portal warps invalidate
// position AND core so the next attack's core refetch carries fresh REST
// X/Y (exactly today's source) until the first movement packet re-feeds
// the overlay (design §10.4).
func handleSnapshotMapChanged(sc server.Model, _ writer.Producer) message.Handler[character2.StatusEvent[character2.StatusEventMapChangedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.StatusEventMapChangedBody]) {
		if e.Type != character2.StatusEventTypeMapChanged {
			return
		}
		t := tenant.MustFromContext(ctx)
		if !sc.IsWorld(t, e.WorldId) {
			return
		}
		r := snapshot.GetRegistry()
		if e.Body.UseTargetPosition {
			r.SetPosition(t, e.CharacterId, e.Body.TargetX, e.Body.TargetY)
			return
		}
		r.InvalidatePosition(t, e.CharacterId)
		r.InvalidateCore(t, e.CharacterId)
	}
}
```

3. Register them in `InitHandlers` after the existing six registrations (same pattern, appending to `handles`):

```go
					id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleSnapshotStatChanged(sc, wp))))
					if err != nil {
						return nil, err
					}
					handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
					id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleSnapshotLevelChanged(sc, wp))))
					if err != nil {
						return nil, err
					}
					handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
					id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleSnapshotExperienceChanged(sc, wp))))
					if err != nil {
						return nil, err
					}
					handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
					id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleSnapshotMapChanged(sc, wp))))
					if err != nil {
						return nil, err
					}
					handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
```

- [x] **Step 4: Run tests to verify they pass**

Run (from `services/atlas-channel/atlas.com/channel`): `go test -race ./kafka/consumer/character/ -v`
Expected: PASS.

- [x] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/consumer/character/consumer.go \
        services/atlas-channel/atlas.com/channel/kafka/consumer/character/consumer_test.go
git commit -m "feat(task-122): project character status events into the snapshot"
```

---

### Task 6: Skill consumer snapshot handlers (incl. new DELETED handler)

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/message/skill/kafka.go` (add `StatusEventTypeDeleted` + `StatusEventDeletedBody`)
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/skill/consumer.go`
- Create: `services/atlas-channel/atlas.com/channel/kafka/consumer/skill/consumer_test.go`

**Interfaces:**
- Consumes (Task 2): `snapshot.GetRegistry()`: `UpsertSkill`, `RemoveSkill`, `View`/`BackfillSkills` (tests); `skill.NewModelBuilder` (`character/skill/builder.go`).
- Consumes (existing): `StatusEventCreatedBody/UpdatedBody{Level, MasterLevel, Expiration}` (`kafka/message/skill/kafka.go:36-46`), envelope `SkillId` (`kafka.go:31`).
- Produces: `handleSnapshotSkillCreated`, `handleSnapshotSkillUpdated`, `handleSnapshotSkillDeleted` registered in `InitHandlers`; message constants `skill2.StatusEventTypeDeleted = "DELETED"`, `skill2.StatusEventDeletedBody struct{}`.

- [x] **Step 1: Add the DELETED message definitions**

In `services/atlas-channel/atlas.com/channel/kafka/message/skill/kafka.go`, extend the status-event const block:

```go
	StatusEventTypeDeleted         = "DELETED"
```

and append:

```go
type StatusEventDeletedBody struct {
}
```

VERIFY against the producer: `services/atlas-skills/atlas.com/skills/skill/producer.go` (emission at `skill/processor.go:284` per event-coverage §3) — confirm the event `Type` string is `"DELETED"` and the body carries no fields the handler needs. If the body has fields, mirror them (unused fields may be omitted; the handler needs only the envelope `CharacterId`/`SkillId`).

- [x] **Step 2: Write the failing tests**

Create `services/atlas-channel/atlas.com/channel/kafka/consumer/skill/consumer_test.go`:

```go
package skill

import (
	"context"
	"testing"
	"time"

	skillmodel "atlas-channel/character/skill"
	"atlas-channel/character/snapshot"
	skill2 "atlas-channel/kafka/message/skill"
	"atlas-channel/server"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	skillconst "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func newTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return tm
}

func newTestServer(t *testing.T, tm tenant.Model) server.Model {
	t.Helper()
	return server.Register(tm, channel.NewModel(0, 1), "127.0.0.1", 8484)
}

func seedSkills(t *testing.T, tm tenant.Model, characterId uint32, ms []skillmodel.Model) {
	t.Helper()
	v := snapshot.GetRegistry().View(tm, characterId)
	if !snapshot.GetRegistry().BackfillSkills(tm, characterId, ms, v.SkillsGen) {
		t.Fatalf("seed backfill rejected")
	}
}

func TestHandleSnapshotSkillCreatedAndUpdated_Upsert(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)
	seedSkills(t, tm, 51, nil)

	ce := skill2.StatusEvent[skill2.StatusEventCreatedBody]{
		CharacterId: 51, SkillId: 3121004, Type: skill2.StatusEventTypeCreated,
		Body: skill2.StatusEventCreatedBody{Level: 1, MasterLevel: 30, Expiration: time.Time{}},
	}
	handleSnapshotSkillCreated(sc, nil)(logrus.New(), ctx, ce)

	v := snapshot.GetRegistry().View(tm, 51)
	if len(v.Skills) != 1 || v.Skills[0].Id() != skillconst.Id(3121004) || v.Skills[0].Level() != 1 {
		t.Fatalf("CREATED upsert mismatch: %+v", v.Skills)
	}

	ue := skill2.StatusEvent[skill2.StatusEventUpdatedBody]{
		CharacterId: 51, SkillId: 3121004, Type: skill2.StatusEventTypeUpdated,
		Body: skill2.StatusEventUpdatedBody{Level: 2, MasterLevel: 30, Expiration: time.Time{}},
	}
	handleSnapshotSkillUpdated(sc, nil)(logrus.New(), ctx, ue)

	v = snapshot.GetRegistry().View(tm, 51)
	if len(v.Skills) != 1 || v.Skills[0].Level() != 2 {
		t.Fatalf("UPDATED upsert mismatch: %+v", v.Skills)
	}
}

func TestHandleSnapshotSkillDeleted_Removes(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)
	seedSkills(t, tm, 52, []skillmodel.Model{
		skillmodel.NewModelBuilder(skillconst.Id(3121004)).SetLevel(10).MustBuild(),
	})

	de := skill2.StatusEvent[skill2.StatusEventDeletedBody]{
		CharacterId: 52, SkillId: 3121004, Type: skill2.StatusEventTypeDeleted,
	}
	handleSnapshotSkillDeleted(sc, nil)(logrus.New(), ctx, de)

	if v := snapshot.GetRegistry().View(tm, 52); len(v.Skills) != 0 {
		t.Fatalf("DELETED must remove the skill: %+v", v.Skills)
	}
}
```

- [x] **Step 3: Run tests to verify they fail**

Run (from `services/atlas-channel/atlas.com/channel`): `go test ./kafka/consumer/skill/ -v`
Expected: FAIL to compile — `handleSnapshotSkillCreated` undefined.

- [x] **Step 4: Implement the handlers**

In `services/atlas-channel/atlas.com/channel/kafka/consumer/skill/consumer.go`, add imports `skillmodel "atlas-channel/character/skill"`, `"atlas-channel/character/snapshot"`, `skillconst "github.com/Chronicle20/atlas/libs/atlas-constants/skill"`, then append:

```go
// --- task-122 snapshot maintenance (additive) ---

func snapshotSkillFromEvent(skillId uint32, level byte, masterLevel byte, expiration time.Time) skillmodel.Model {
	return skillmodel.NewModelBuilder(skillconst.Id(skillId)).
		SetLevel(level).
		SetMasterLevel(masterLevel).
		SetExpiration(expiration).
		MustBuild()
}

func handleSnapshotSkillCreated(sc server.Model, _ writer.Producer) message.Handler[skill2.StatusEvent[skill2.StatusEventCreatedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e skill2.StatusEvent[skill2.StatusEventCreatedBody]) {
		if e.Type != skill2.StatusEventTypeCreated {
			return
		}
		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}
		snapshot.GetRegistry().UpsertSkill(t, e.CharacterId, snapshotSkillFromEvent(e.SkillId, e.Body.Level, e.Body.MasterLevel, e.Body.Expiration))
	}
}

func handleSnapshotSkillUpdated(sc server.Model, _ writer.Producer) message.Handler[skill2.StatusEvent[skill2.StatusEventUpdatedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e skill2.StatusEvent[skill2.StatusEventUpdatedBody]) {
		if e.Type != skill2.StatusEventTypeUpdated {
			return
		}
		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}
		snapshot.GetRegistry().UpsertSkill(t, e.CharacterId, snapshotSkillFromEvent(e.SkillId, e.Body.Level, e.Body.MasterLevel, e.Body.Expiration))
	}
}

// handleSnapshotSkillDeleted is atlas-channel's first consumer of the
// skill DELETED event (saga compensation path; the packet layer never
// needed it — event-coverage.md §3).
func handleSnapshotSkillDeleted(sc server.Model, _ writer.Producer) message.Handler[skill2.StatusEvent[skill2.StatusEventDeletedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e skill2.StatusEvent[skill2.StatusEventDeletedBody]) {
		if e.Type != skill2.StatusEventTypeDeleted {
			return
		}
		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}
		snapshot.GetRegistry().RemoveSkill(t, e.CharacterId, skillconst.Id(e.SkillId))
	}
}
```

Register all three in `InitHandlers` after the existing four registrations (same `rf(...)` + `handles = append(...)` pattern as Task 5 Step 3.3, one block per handler).

- [x] **Step 5: Run tests, build, vet, commit**

Run (from `services/atlas-channel/atlas.com/channel`): `go test -race ./kafka/consumer/skill/ && go build ./... && go vet ./...`
Expected: PASS / clean.

```bash
git add services/atlas-channel/atlas.com/channel/kafka/message/skill/kafka.go \
        services/atlas-channel/atlas.com/channel/kafka/consumer/skill/consumer.go \
        services/atlas-channel/atlas.com/channel/kafka/consumer/skill/consumer_test.go
git commit -m "feat(task-122): project skill status events (incl. DELETED) into the snapshot"
```

---

### Task 7: Asset + compartment consumer snapshot handlers

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/asset/consumer.go`
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/asset/consumer_test.go` (append)
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/compartment/consumer.go`
- Create or append: `services/atlas-channel/atlas.com/channel/kafka/consumer/compartment/consumer_test.go`

**Interfaces:**
- Consumes (Task 2): `snapshot.GetRegistry()`: `UpsertAsset`, `SetAssetQuantity`, `SetAssetSlot`, `RemoveAsset`, `InvalidateInventory`, `View`/`BackfillInventory` (tests).
- Consumes (existing): asset envelope `{CharacterId, CompartmentId, AssetId, TemplateId, Slot}` (`kafka/message/asset/kafka.go:21-29`); rich bodies CREATED/UPDATED/ACCEPTED (`kafka.go:31,65,111`); `QUANTITY_CHANGED.Quantity` (`kafka.go:106`); the existing full-asset builder `buildAssetFromCreatedBody` (`kafka/consumer/asset/consumer.go:119`) and its UPDATED/ACCEPTED siblings (locate by grep `NewModelBuilder(e.AssetId` in that file — reuse, do not duplicate); compartment event constants (`kafka/message/compartment/kafka.go:112-118`).
- Produces: asset handlers `handleSnapshotAssetCreated/Updated/Accepted/QuantityChanged/Moved/Deleted/Released/Expired`; compartment handlers `handleSnapshotCompartmentChanged` (one handler covering CREATED/DELETED/CAPACITY_CHANGED via three registrations OR three thin handlers — implement as three thin handlers to match the one-handler-per-body-type pattern) plus `handleSnapshotMergeComplete`, `handleSnapshotSortComplete`. RESERVED/RESERVATION_CANCELLED intentionally have no snapshot handlers (REST parity — Task 1 Step 3 verified).

- [x] **Step 1: Write the failing asset tests**

Append to `services/atlas-channel/atlas.com/channel/kafka/consumer/asset/consumer_test.go` (add the imports it needs: `context`, `atlas-channel/asset`, `atlas-channel/character/snapshot`, `atlas-channel/compartment`, `atlas-channel/inventory`, `atlas-channel/server`, `invconst "github.com/Chronicle20/atlas/libs/atlas-constants/inventory"`, `"github.com/Chronicle20/atlas/libs/atlas-constants/channel"`, `tenant`, `uuid`, `logrus`, `testing`):

```go
func newSnapshotTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return tm
}

func newSnapshotTestServer(t *testing.T, tm tenant.Model) server.Model {
	t.Helper()
	return server.Register(tm, channel.NewModel(0, 1), "127.0.0.1", 8484)
}

// seedInventory backfills a one-consumable-asset inventory and returns the
// consumable compartment id.
func seedInventory(t *testing.T, tm tenant.Model, characterId uint32) uuid.UUID {
	t.Helper()
	compId := uuid.New()
	a := asset.NewModelBuilder(9001, compId, 2060000).SetSlot(2).SetQuantity(400).MustBuild()
	comp := compartment.NewBuilder(compId, characterId, invconst.TypeValueUse, 96).AddAsset(a).MustBuild()
	inv := inventory.NewBuilder(characterId).
		SetEquipable(compartment.NewBuilder(uuid.New(), characterId, invconst.TypeValueEquip, 96).MustBuild()).
		SetConsumable(comp).
		SetSetup(compartment.NewBuilder(uuid.New(), characterId, invconst.TypeValueSetup, 96).MustBuild()).
		SetEtc(compartment.NewBuilder(uuid.New(), characterId, invconst.TypeValueETC, 96).MustBuild()).
		SetCash(compartment.NewBuilder(uuid.New(), characterId, invconst.TypeValueCash, 96).MustBuild()).
		MustBuild()
	v := snapshot.GetRegistry().View(tm, characterId)
	if !snapshot.GetRegistry().BackfillInventory(tm, characterId, inv, v.InvGen) {
		t.Fatalf("seed backfill rejected")
	}
	return compId
}

func TestHandleSnapshotAssetQuantityChanged_Absolute(t *testing.T) {
	tm := newSnapshotTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newSnapshotTestServer(t, tm)
	seedInventory(t, tm, 61)

	e := asset2.StatusEvent[asset2.QuantityChangedEventBody]{
		CharacterId: 61, AssetId: 9001, TemplateId: 2060000, Slot: 2,
		Type: asset2.StatusEventTypeQuantityChanged,
		Body: asset2.QuantityChangedEventBody{Quantity: 399},
	}
	handleSnapshotAssetQuantityChanged(sc, nil)(logrus.New(), ctx, e)
	handleSnapshotAssetQuantityChanged(sc, nil)(logrus.New(), ctx, e) // redelivery-idempotent

	v := snapshot.GetRegistry().View(tm, 61)
	a, ok := v.Inv.Consumable().FindById(9001)
	if !ok || a.Quantity() != 399 {
		t.Fatalf("quantity mismatch: %+v ok=%v", a, ok)
	}
}

func TestHandleSnapshotAssetMoved_SetsSlotAbsolute(t *testing.T) {
	tm := newSnapshotTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newSnapshotTestServer(t, tm)
	seedInventory(t, tm, 62)

	e := asset2.StatusEvent[asset2.MovedStatusEventBody]{
		CharacterId: 62, AssetId: 9001, TemplateId: 2060000, Slot: 7,
		Type: asset2.StatusEventTypeMoved,
		Body: asset2.MovedStatusEventBody{OldSlot: 2},
	}
	handleSnapshotAssetMoved(sc, nil)(logrus.New(), ctx, e)

	v := snapshot.GetRegistry().View(tm, 62)
	a, _ := v.Inv.Consumable().FindById(9001)
	if a.Slot() != 7 {
		t.Fatalf("slot mismatch: %d", a.Slot())
	}
}

func TestHandleSnapshotAssetCreatedAndDeleted(t *testing.T) {
	tm := newSnapshotTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newSnapshotTestServer(t, tm)
	compId := seedInventory(t, tm, 63)

	ce := asset2.StatusEvent[asset2.CreatedStatusEventBody]{
		CharacterId: 63, CompartmentId: compId, AssetId: 9002, TemplateId: 2070000, Slot: 3,
		Type: asset2.StatusEventTypeCreated,
		Body: asset2.CreatedStatusEventBody{Quantity: 500},
	}
	handleSnapshotAssetCreated(sc, nil)(logrus.New(), ctx, ce)

	v := snapshot.GetRegistry().View(tm, 63)
	a, ok := v.Inv.Consumable().FindById(9002)
	if !ok || a.Quantity() != 500 || a.Slot() != 3 || a.TemplateId() != 2070000 {
		t.Fatalf("CREATED upsert mismatch: %+v ok=%v", a, ok)
	}

	de := asset2.StatusEvent[asset2.DeletedStatusEventBody]{
		CharacterId: 63, CompartmentId: compId, AssetId: 9002, TemplateId: 2070000, Slot: 3,
		Type: asset2.StatusEventTypeDeleted,
	}
	handleSnapshotAssetDeleted(sc, nil)(logrus.New(), ctx, de)

	v = snapshot.GetRegistry().View(tm, 63)
	if _, ok = v.Inv.Consumable().FindById(9002); ok {
		t.Fatalf("DELETED must remove the asset")
	}
}

func TestHandleSnapshotAssetReleasedAndExpired_Invalidate(t *testing.T) {
	tm := newSnapshotTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newSnapshotTestServer(t, tm)
	seedInventory(t, tm, 64)

	re := asset2.StatusEvent[asset2.ReleasedStatusEventBody]{
		CharacterId: 64, AssetId: 9001, Type: asset2.StatusEventTypeReleased,
	}
	handleSnapshotAssetReleased(sc, nil)(logrus.New(), ctx, re)
	if v := snapshot.GetRegistry().View(tm, 64); v.InvValid {
		t.Fatalf("RELEASED (thin) must invalidate the inventory component")
	}
}
```

- [x] **Step 2: Write the failing compartment tests**

Create (or append to, if it exists) `services/atlas-channel/atlas.com/channel/kafka/consumer/compartment/consumer_test.go` with the same tenant/server helper pattern as Step 1 (fresh helpers in this package), then:

```go
func TestHandleSnapshotCompartmentEvents_Invalidate(t *testing.T) {
	tm := newSnapshotTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newSnapshotTestServer(t, tm)

	cases := []struct {
		name string
		fire func()
	}{
		{"capacity changed", func() {
			e := compartment2.StatusEvent[compartment2.CapacityChangedEventBody]{
				CharacterId: 71, Type: compartment2.StatusEventTypeCapacityChanged,
				Body: compartment2.CapacityChangedEventBody{Type: 2, Capacity: 128},
			}
			handleSnapshotCompartmentCapacityChanged(sc, nil)(logrus.New(), ctx, e)
		}},
		{"deleted", func() {
			e := compartment2.StatusEvent[compartment2.DeletedStatusEventBody]{
				CharacterId: 71, Type: compartment2.StatusEventTypeDeleted,
			}
			handleSnapshotCompartmentDeleted(sc, nil)(logrus.New(), ctx, e)
		}},
		{"created", func() {
			e := compartment2.StatusEvent[compartment2.CreatedStatusEventBody]{
				CharacterId: 71, Type: compartment2.StatusEventTypeCreated,
				Body: compartment2.CreatedStatusEventBody{Type: 2, Capacity: 96},
			}
			handleSnapshotCompartmentCreated(sc, nil)(logrus.New(), ctx, e)
		}},
		{"merge complete", func() {
			e := compartment2.StatusEvent[compartment2.MergeCompleteEventBody]{
				CharacterId: 71, Type: compartment2.StatusEventTypeMergeComplete,
				Body: compartment2.MergeCompleteEventBody{Type: 2},
			}
			handleSnapshotMergeComplete(sc, nil)(logrus.New(), ctx, e)
		}},
		{"sort complete", func() {
			e := compartment2.StatusEvent[compartment2.SortCompleteEventBody]{
				CharacterId: 71, Type: compartment2.StatusEventTypeSortComplete,
				Body: compartment2.SortCompleteEventBody{Type: 2},
			}
			handleSnapshotSortComplete(sc, nil)(logrus.New(), ctx, e)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			seedTestInventory(t, tm, 71) // same seed helper shape as the asset package test
			tc.fire()
			if v := snapshot.GetRegistry().View(tm, 71); v.InvValid {
				t.Fatalf("%s must invalidate the inventory component", tc.name)
			}
		})
	}
}
```

(Define `seedTestInventory` in this package's test file — same body as the asset package's `seedInventory`; the two packages cannot share unexported test helpers and CLAUDE.md forbids `*_testhelpers.go`. Match the compartment message import alias to the package's existing convention — check the top of `kafka/consumer/compartment/consumer.go`.)

- [x] **Step 3: Run tests to verify they fail**

Run (from `services/atlas-channel/atlas.com/channel`): `go test ./kafka/consumer/asset/ ./kafka/consumer/compartment/ -v`
Expected: FAIL to compile — snapshot handlers undefined.

- [x] **Step 4: Implement the asset handlers**

In `services/atlas-channel/atlas.com/channel/kafka/consumer/asset/consumer.go`, add imports `"atlas-channel/character/snapshot"`, `tenant` (if not present), then append (locate the existing UPDATED/ACCEPTED full-asset builder funcs next to `buildAssetFromCreatedBody:119` and reuse them — they exist for the packet handlers; if the ACCEPTED handler builds inline, extract its builder into `buildAssetFromAcceptedBody` rather than duplicating):

```go
// --- task-122 snapshot maintenance (additive) ---

func handleSnapshotAssetCreated(sc server.Model, _ writer.Producer) message.Handler[asset2.StatusEvent[asset2.CreatedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e asset2.StatusEvent[asset2.CreatedStatusEventBody]) {
		if e.Type != asset2.StatusEventTypeCreated {
			return
		}
		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}
		snapshot.GetRegistry().UpsertAsset(t, e.CharacterId, e.CompartmentId, buildAssetFromCreatedBody(e))
	}
}

func handleSnapshotAssetUpdated(sc server.Model, _ writer.Producer) message.Handler[asset2.StatusEvent[asset2.UpdatedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e asset2.StatusEvent[asset2.UpdatedStatusEventBody]) {
		if e.Type != asset2.StatusEventTypeUpdated {
			return
		}
		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}
		snapshot.GetRegistry().UpsertAsset(t, e.CharacterId, e.CompartmentId, buildAssetFromUpdatedBody(e))
	}
}

func handleSnapshotAssetAccepted(sc server.Model, _ writer.Producer) message.Handler[asset2.StatusEvent[asset2.AcceptedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e asset2.StatusEvent[asset2.AcceptedStatusEventBody]) {
		if e.Type != asset2.StatusEventTypeAccepted {
			return
		}
		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}
		snapshot.GetRegistry().UpsertAsset(t, e.CharacterId, e.CompartmentId, buildAssetFromAcceptedBody(e))
	}
}

func handleSnapshotAssetQuantityChanged(sc server.Model, _ writer.Producer) message.Handler[asset2.StatusEvent[asset2.QuantityChangedEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e asset2.StatusEvent[asset2.QuantityChangedEventBody]) {
		if e.Type != asset2.StatusEventTypeQuantityChanged {
			return
		}
		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}
		snapshot.GetRegistry().SetAssetQuantity(t, e.CharacterId, e.AssetId, e.Body.Quantity)
	}
}

// handleSnapshotAssetMoved sets the slot ABSOLUTE by AssetId (idempotent):
// each leg of a swap arrives as its own MOVED with the new slot in the
// envelope (event-coverage.md §4).
func handleSnapshotAssetMoved(sc server.Model, _ writer.Producer) message.Handler[asset2.StatusEvent[asset2.MovedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e asset2.StatusEvent[asset2.MovedStatusEventBody]) {
		if e.Type != asset2.StatusEventTypeMoved {
			return
		}
		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}
		snapshot.GetRegistry().SetAssetSlot(t, e.CharacterId, e.AssetId, e.Slot)
	}
}

func handleSnapshotAssetDeleted(sc server.Model, _ writer.Producer) message.Handler[asset2.StatusEvent[asset2.DeletedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e asset2.StatusEvent[asset2.DeletedStatusEventBody]) {
		if e.Type != asset2.StatusEventTypeDeleted {
			return
		}
		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}
		snapshot.GetRegistry().RemoveAsset(t, e.CharacterId, e.AssetId)
	}
}

// RELEASED/EXPIRED are thin (no replacement asset payload): invalidate.
func handleSnapshotAssetReleased(sc server.Model, _ writer.Producer) message.Handler[asset2.StatusEvent[asset2.ReleasedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e asset2.StatusEvent[asset2.ReleasedStatusEventBody]) {
		if e.Type != asset2.StatusEventTypeReleased {
			return
		}
		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}
		snapshot.GetRegistry().InvalidateInventory(t, e.CharacterId)
	}
}

func handleSnapshotAssetExpired(sc server.Model, _ writer.Producer) message.Handler[asset2.StatusEvent[asset2.ExpiredStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e asset2.StatusEvent[asset2.ExpiredStatusEventBody]) {
		if e.Type != asset2.StatusEventTypeExpired {
			return
		}
		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}
		snapshot.GetRegistry().InvalidateInventory(t, e.CharacterId)
	}
}
```

Register all eight in the asset `InitHandlers` (same pattern as Task 5). If `buildAssetFromUpdatedBody`/`buildAssetFromAcceptedBody` do not already exist as named funcs, extract them from the existing UPDATED/ACCEPTED packet handlers in the same file (pure refactor — the packet handlers then call the extracted funcs; behavior unchanged).

- [x] **Step 5: Implement the compartment handlers**

In `services/atlas-channel/atlas.com/channel/kafka/consumer/compartment/consumer.go`, add imports (`atlas-channel/character/snapshot`, tenant), then append five thin handlers, all with the body pattern:

```go
func handleSnapshotCompartmentCreated(sc server.Model, _ writer.Producer) message.Handler[compartment2.StatusEvent[compartment2.CreatedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e compartment2.StatusEvent[compartment2.CreatedStatusEventBody]) {
		if e.Type != compartment2.StatusEventTypeCreated {
			return
		}
		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}
		// A compartment the snapshot has never seen: refetch rather than
		// synthesize (bulk shape change — event-coverage.md §4).
		snapshot.GetRegistry().InvalidateInventory(t, e.CharacterId)
	}
}
```

…and identically-shaped `handleSnapshotCompartmentDeleted` (gate `StatusEventTypeDeleted`), `handleSnapshotCompartmentCapacityChanged` (gate `StatusEventTypeCapacityChanged`), `handleSnapshotMergeComplete` (gate `StatusEventTypeMergeComplete`, body `MergeCompleteEventBody`), `handleSnapshotSortComplete` (gate `StatusEventTypeSortComplete`, body `SortCompleteEventBody`) — each calling `InvalidateInventory`. Match this package's existing message import alias and `InitHandlers` registration pattern (read the file first; it already registers handlers for other compartment events). Register all five.

Do NOT add handlers for `RESERVED`/`RESERVATION_CANCELLED` (REST parity, Task 1 Step 3) unless Task 1 found reservations netted out — in that case both invalidate.

- [x] **Step 6: Run tests, build, vet, commit**

Run (from `services/atlas-channel/atlas.com/channel`): `go test -race ./kafka/consumer/asset/ ./kafka/consumer/compartment/ && go build ./... && go vet ./...`
Expected: PASS / clean.

```bash
git add services/atlas-channel/atlas.com/channel/kafka/consumer/asset/ \
        services/atlas-channel/atlas.com/channel/kafka/consumer/compartment/
git commit -m "feat(task-122): project asset and compartment events into the snapshot inventory component"
```

---

### Task 8: Buff consumer snapshot handlers

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/buff/consumer.go`
- Create: `services/atlas-channel/atlas.com/channel/kafka/consumer/buff/consumer_test.go`

**Interfaces:**
- Consumes (Task 2): `snapshot.GetRegistry()`: `UpsertBuff`, `RemoveBuff`, `View`/`BackfillBuffs` (tests); `buff.NewBuff` (`character/buff/model.go:55`), `stat.NewStat` (`character/buff/stat`).
- Consumes (existing): `AppliedStatusEventBody`/`ExpiredStatusEventBody` (`kafka/message/buff/kafka.go:58-75`) — both carry `SourceId, Level, Duration, Changes, CreatedAt, ExpiresAt` (rich).
- Produces: `handleSnapshotBuffApplied`, `handleSnapshotBuffExpired` registered in `InitHandlers`.

- [x] **Step 1: Write the failing tests**

Create `services/atlas-channel/atlas.com/channel/kafka/consumer/buff/consumer_test.go` (fresh tenant/server helpers as in Task 6):

```go
package buff

import (
	"context"
	"testing"
	"time"

	"atlas-channel/character/snapshot"
	buff2 "atlas-channel/kafka/message/buff"
	"atlas-channel/server"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func newTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return tm
}

func newTestServer(t *testing.T, tm tenant.Model) server.Model {
	t.Helper()
	return server.Register(tm, channel.NewModel(0, 1), "127.0.0.1", 8484)
}

func seedBuffs(t *testing.T, tm tenant.Model, characterId uint32) {
	t.Helper()
	v := snapshot.GetRegistry().View(tm, characterId)
	if !snapshot.GetRegistry().BackfillBuffs(tm, characterId, nil, v.BuffsGen) {
		t.Fatalf("seed backfill rejected")
	}
}

func TestHandleSnapshotBuffAppliedAndExpired(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)
	seedBuffs(t, tm, 81)

	ae := buff2.StatusEvent[buff2.AppliedStatusEventBody]{
		WorldId: 0, CharacterId: 81, Type: buff2.EventStatusTypeBuffApplied,
		Body: buff2.AppliedStatusEventBody{
			SourceId: 3111004, Level: 20, Duration: 60000,
			Changes:   []buff2.StatChange{{Type: "SOUL_ARROW", Amount: 1}},
			CreatedAt: time.Now(), ExpiresAt: time.Now().Add(time.Minute),
		},
	}
	handleSnapshotBuffApplied(sc, nil)(logrus.New(), ctx, ae)
	handleSnapshotBuffApplied(sc, nil)(logrus.New(), ctx, ae) // redelivery: no duplicate

	v := snapshot.GetRegistry().View(tm, 81)
	if len(v.Buffs) != 1 || v.Buffs[0].SourceId() != 3111004 {
		t.Fatalf("APPLIED upsert mismatch: %+v", v.Buffs)
	}
	if len(v.Buffs[0].Changes()) != 1 || v.Buffs[0].Changes()[0].Type() != "SOUL_ARROW" {
		t.Fatalf("changes not carried: %+v", v.Buffs[0].Changes())
	}

	ee := buff2.StatusEvent[buff2.ExpiredStatusEventBody]{
		WorldId: 0, CharacterId: 81, Type: buff2.EventStatusTypeBuffExpired,
		Body: buff2.ExpiredStatusEventBody{SourceId: 3111004},
	}
	handleSnapshotBuffExpired(sc, nil)(logrus.New(), ctx, ee)

	if v = snapshot.GetRegistry().View(tm, 81); len(v.Buffs) != 0 {
		t.Fatalf("EXPIRED must remove by sourceId: %+v", v.Buffs)
	}
}
```

- [x] **Step 2: Run tests to verify they fail**

Run (from `services/atlas-channel/atlas.com/channel`): `go test ./kafka/consumer/buff/ -v`
Expected: FAIL to compile.

- [x] **Step 3: Implement the handlers**

In `services/atlas-channel/atlas.com/channel/kafka/consumer/buff/consumer.go` append (imports: `"atlas-channel/character/snapshot"`; `buff`/`stat` already imported):

```go
// --- task-122 snapshot maintenance (additive) ---

func handleSnapshotBuffApplied(sc server.Model, _ writer.Producer) message.Handler[buff2.StatusEvent[buff2.AppliedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e buff2.StatusEvent[buff2.AppliedStatusEventBody]) {
		if e.Type != buff2.EventStatusTypeBuffApplied {
			return
		}
		t := tenant.MustFromContext(ctx)
		if !sc.IsWorld(t, e.WorldId) {
			return
		}
		changes := make([]stat.Model, 0, len(e.Body.Changes))
		for _, cm := range e.Body.Changes {
			changes = append(changes, stat.NewStat(cm.Type, cm.Amount))
		}
		snapshot.GetRegistry().UpsertBuff(t, e.CharacterId,
			buff.NewBuff(e.Body.SourceId, e.Body.Level, e.Body.Duration, changes, e.Body.CreatedAt, e.Body.ExpiresAt))
	}
}

func handleSnapshotBuffExpired(sc server.Model, _ writer.Producer) message.Handler[buff2.StatusEvent[buff2.ExpiredStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e buff2.StatusEvent[buff2.ExpiredStatusEventBody]) {
		if e.Type != buff2.EventStatusTypeBuffExpired {
			return
		}
		t := tenant.MustFromContext(ctx)
		if !sc.IsWorld(t, e.WorldId) {
			return
		}
		snapshot.GetRegistry().RemoveBuff(t, e.CharacterId, e.Body.SourceId)
	}
}
```

Register both in `InitHandlers` after the existing two registrations.

- [x] **Step 4: Run tests, build, vet, commit**

Run (from `services/atlas-channel/atlas.com/channel`): `go test -race ./kafka/consumer/buff/ && go build ./... && go vet ./...`
Expected: PASS / clean.

```bash
git add services/atlas-channel/atlas.com/channel/kafka/consumer/buff/consumer.go \
        services/atlas-channel/atlas.com/channel/kafka/consumer/buff/consumer_test.go
git commit -m "feat(task-122): project buff applied/expired events into the snapshot"
```

---

### Task 9: Skill-data in-process TTL cache

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/data/skill/cache.go`
- Create: `services/atlas-channel/atlas.com/channel/data/skill/metrics.go`
- Create: `services/atlas-channel/atlas.com/channel/data/skill/cache_test.go`
- Modify: `services/atlas-channel/atlas.com/channel/data/skill/processor.go:30-32` (`GetById` reads through the cache)
- Modify: `services/atlas-channel/atlas.com/channel/main.go` (evictor block)

**Interfaces:**
- Consumes (existing): `data/skill.Model`/`RestModel`/`Extract`/`requestById` (`data/skill/rest.go`, `requests.go`), `requests.ErrNotFound` (`libs/atlas-rest/requests`), the task-120 in-process cache exemplar `monster/information/cache.go` (env parsing + error classification — copy its `parseBoolEnv`/`parseDurationEnv`/`classifyError` shapes).
- Produces:
  - `Processor.GetById(uniqueId uint32) (Model, error)` — signature unchanged, now cached. `GetEffect` (processor.go:34) is automatically cached through it (attack `se`, MP-Eater effect, writer mastery — FR-4.3).
  - `skill.EvictTenant(tid uuid.UUID)` for main.go.
  - Test seam `upstreamFn func(l logrus.FieldLogger, ctx context.Context, skillId uint32) (Model, error)`.
- Semantics (task-060, re-expressed in-process per task-120 §5.4): positive TTL 5m / negative 30s; negative caching ONLY for `errors.Is(err, requests.ErrNotFound)`; transient errors never cached; negative hits synthesize `fmt.Errorf("skill %d not found: %w", id, requests.ErrNotFound)`; lazy expiry on read; no sweeper; no singleflight (concurrent same-key misses may duplicate the fetch — accepted, task-060 precedent). Env: `SKILL_DATA_CACHE_ENABLED`/`SKILL_DATA_CACHE_TTL`/`SKILL_DATA_CACHE_NEGATIVE_TTL` per Global Constraints.

- [x] **Step 1: Write the failing tests**

Create `services/atlas-channel/atlas.com/channel/data/skill/cache_test.go` — port the task-120 monster-info cache test suite (`monster/information/cache_test.go`, present after Task 1's rebase) to this package: same cases, renamed env vars/types. Required cases:

```go
// - TestSkillCache_PositiveHitAvoidsSecondFetch: two GetById(2301002) calls,
//   upstream counter == 1, second returns the cached Model (Effects() intact).
// - TestSkillCache_ExpiredEntryRefetches: force expiresAt into the past via
//   the internals (same-package test), third call refetches (counter == 2).
// - TestSkillCache_NegativeCachesNotFound: upstream returns
//   fmt.Errorf("x: %w", requests.ErrNotFound) once; second call does NOT hit
//   upstream and the returned error still satisfies
//   errors.Is(err, requests.ErrNotFound).
// - TestSkillCache_TransientErrorNotCached: upstream returns a plain error;
//   second call hits upstream again (counter == 2).
// - TestSkillCache_DisabledBypasses: with SKILL_DATA_CACHE_ENABLED=false
//   (t.Setenv BEFORE first getSkillCache() — reset the singleton first),
//   every call hits upstream.
// - TestSkillCache_TenantIsolation: same skillId under two tenants fetches
//   twice; EvictTenant(t1) evicts only t1.
// - TestSkillCache_ConcurrentAccess: 8 goroutines × 200 mixed GetById calls
//   under -race.
```

Write each of these as real test functions (the task-120 file is the concrete template — same reset helper pattern: `resetSkillCache()` zeroing `skillCacheOnce`/`skillCachePtr`, `t.Cleanup(resetSkillCache)` in every test, `t.Setenv` for env cases).

- [x] **Step 2: Run tests to verify they fail**

Run (from `services/atlas-channel/atlas.com/channel`): `go test ./data/skill/ -v`
Expected: FAIL to compile — `upstreamFn`, `getSkillCache` undefined.

- [x] **Step 3: Implement metrics.go and cache.go**

Create `services/atlas-channel/atlas.com/channel/data/skill/metrics.go`:

```go
package skill

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

var skillDataCacheTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "atlas_channel_skill_data_cache_total",
		Help: "Skill-data cache lookups by tenant and outcome.",
	},
	[]string{"tenant", "outcome"},
)

func recordCache(t tenant.Model, outcome string) {
	skillDataCacheTotal.WithLabelValues(t.Id().String(), outcome).Inc()
}
```

Create `services/atlas-channel/atlas.com/channel/data/skill/cache.go`:

```go
package skill

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// In-process TTL cache for immutable atlas-data skill templates (task-122
// FR-4.3). Semantics ported from task-060 (atlas-monsters
// monster/information/cache.go) re-expressed in memory per task-120 §5.4:
// positive/negative TTLs, ErrNotFound-only negative caching, env
// kill-switch, lazy expiry, no singleflight.

type cacheConfig struct {
	enabled     bool
	ttl         time.Duration
	negativeTTL time.Duration
}

const (
	envEnabled     = "SKILL_DATA_CACHE_ENABLED"
	envTTL         = "SKILL_DATA_CACHE_TTL"
	envNegativeTTL = "SKILL_DATA_CACHE_NEGATIVE_TTL"

	defaultTTL         = 5 * time.Minute
	defaultNegativeTTL = 30 * time.Second

	minTTL         = 1 * time.Second
	maxTTL         = 24 * time.Hour
	minNegativeTTL = 0 * time.Second
	maxNegativeTTL = 5 * time.Minute

	outcomeCacheHit         = "hit"
	outcomeCacheNegativeHit = "negative_hit"
	outcomeCacheMiss        = "miss"
)

var configLogger logrus.FieldLogger = logrus.StandardLogger()

func loadConfig() cacheConfig {
	return cacheConfig{
		enabled:     parseBoolEnv(envEnabled, true),
		ttl:         parseDurationEnv(envTTL, defaultTTL, minTTL, maxTTL),
		negativeTTL: parseDurationEnv(envNegativeTTL, defaultNegativeTTL, minNegativeTTL, maxNegativeTTL),
	}
}

func parseBoolEnv(name string, def bool) bool {
	v, ok := os.LookupEnv(name)
	if !ok || v == "" {
		return def
	}
	switch v {
	case "true", "TRUE", "True", "1", "yes", "y":
		return true
	case "false", "FALSE", "False", "0", "no", "n":
		return false
	default:
		configLogger.Warnf("invalid bool for %s=%q; using default %v", name, v, def)
		return def
	}
}

func parseDurationEnv(name string, def, lo, hi time.Duration) time.Duration {
	v, ok := os.LookupEnv(name)
	if !ok || v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		configLogger.Warnf("invalid duration for %s=%q; using default %s", name, v, def)
		return def
	}
	if d < lo || d > hi {
		configLogger.Warnf("%s=%s out of range [%s, %s]; using default %s", name, d, lo, hi, def)
		return def
	}
	return d
}

type cacheEntry struct {
	m         Model
	negative  bool
	expiresAt time.Time
}

type skillCache struct {
	cfg       cacheConfig
	mu        sync.RWMutex
	perTenant map[uuid.UUID]map[uint32]cacheEntry
}

var (
	skillCacheOnce sync.Once
	skillCachePtr  *skillCache
)

func getSkillCache() *skillCache {
	skillCacheOnce.Do(func() {
		skillCachePtr = &skillCache{
			cfg:       loadConfig(),
			perTenant: map[uuid.UUID]map[uint32]cacheEntry{},
		}
	})
	return skillCachePtr
}

// upstreamFn is the test seam for the real REST fetch (task-060 precedent).
var upstreamFn = func(l logrus.FieldLogger, ctx context.Context, skillId uint32) (Model, error) {
	return requests.Provider[RestModel, Model](l, ctx)(requestById(skillId), Extract)()
}

func notFoundError(skillId uint32) error {
	return fmt.Errorf("skill %d not found: %w", skillId, requests.ErrNotFound)
}

func (c *skillCache) get(t tenant.Model, skillId uint32) (cacheEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	tm, ok := c.perTenant[t.Id()]
	if !ok {
		return cacheEntry{}, false
	}
	e, ok := tm[skillId]
	if !ok || time.Now().After(e.expiresAt) {
		return cacheEntry{}, false
	}
	return e, true
}

func (c *skillCache) put(t tenant.Model, skillId uint32, e cacheEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	tm, ok := c.perTenant[t.Id()]
	if !ok {
		tm = map[uint32]cacheEntry{}
		c.perTenant[t.Id()] = tm
	}
	tm[skillId] = e
}

// EvictTenant drops the tenant's cached skill templates (listener drain).
func EvictTenant(tid uuid.UUID) {
	c := getSkillCache()
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.perTenant, tid)
}

// getByIdCached is the read-through used by Processor.GetById.
func getByIdCached(l logrus.FieldLogger, ctx context.Context, skillId uint32) (Model, error) {
	c := getSkillCache()
	if !c.cfg.enabled {
		return upstreamFn(l, ctx, skillId)
	}
	t := tenant.MustFromContext(ctx)
	if e, ok := c.get(t, skillId); ok {
		if e.negative {
			recordCache(t, outcomeCacheNegativeHit)
			return Model{}, notFoundError(skillId)
		}
		recordCache(t, outcomeCacheHit)
		return e.m, nil
	}
	recordCache(t, outcomeCacheMiss)
	m, err := upstreamFn(l, ctx, skillId)
	if err != nil {
		if errors.Is(err, requests.ErrNotFound) && c.cfg.negativeTTL > 0 {
			c.put(t, skillId, cacheEntry{negative: true, expiresAt: time.Now().Add(c.cfg.negativeTTL)})
		}
		return Model{}, err
	}
	c.put(t, skillId, cacheEntry{m: m, expiresAt: time.Now().Add(c.cfg.ttl)})
	return m, nil
}
```

In `services/atlas-channel/atlas.com/channel/data/skill/processor.go`, replace `GetById`:

```go
func (p *ProcessorImpl) GetById(uniqueId uint32) (Model, error) {
	return getByIdCached(p.l, p.ctx, uniqueId)
}
```

In `services/atlas-channel/atlas.com/channel/main.go`, in the evictor block, add (import alias — check how `data/skill` is imported in main.go; if absent add `dataskill "atlas-channel/data/skill"`):

```go
		dataskill.EvictTenant(tid)
```

- [x] **Step 4: Run tests to verify they pass**

Run (from `services/atlas-channel/atlas.com/channel`): `go test -race ./data/skill/... && go build ./... && go vet ./...`
Expected: PASS / clean. NOTE: every existing caller of `GetById`/`GetEffect` now reads through the cache — run the full handler/writer test suites too: `go test -race ./socket/...`.

- [x] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/data/skill/ \
        services/atlas-channel/atlas.com/channel/main.go
git commit -m "feat(task-122): in-process TTL cache for atlas-data skill templates"
```

---

### Task 10: Live-mirror X/Y extension + monster movement position feed

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/monster/live_mirror.go` (as landed by task-120 — reconciled in Task 1)
- Modify: `services/atlas-channel/atlas.com/channel/monster/live_mirror_test.go` (append)
- Modify: `services/atlas-channel/atlas.com/channel/movement/processor.go` (`ForMonster` — feed mirror position from the fold)
- Modify: `services/atlas-channel/atlas.com/channel/movement/processor_test.go` (append)

**Interfaces:**
- Consumes: task-120's `LiveEntry`/`LiveMirror`/`LiveEntryFromModel` as reconciled in Task 1; monster `Model.X()/Y()` (`monster/model.go`).
- Produces (used by Task 11): `LiveEntry.X int16`, `LiveEntry.Y int16`; `(*LiveMirror).UpdatePosition(t tenant.Model, uniqueId uint32, x, y int16)` (update-only); `LiveEntryFromModel` maps X/Y.

- [x] **Step 1: Write the failing tests**

Append to `services/atlas-channel/atlas.com/channel/monster/live_mirror_test.go`:

```go
func TestLiveEntryFromModel_MapsPosition(t *testing.T) {
	f := testField()
	mo, err := NewModelBuilder(7, f, 100100).SetX(123).SetY(-45).Build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	e := LiveEntryFromModel(mo)
	if e.X != 123 || e.Y != -45 {
		t.Fatalf("position not mapped: %d/%d", e.X, e.Y)
	}
}

func TestLiveMirror_UpdatePosition(t *testing.T) {
	m := newTestLiveMirror()
	tm := newTestTenant(t)
	m.UpdatePosition(tm, 7, 5, 6) // absent: must not create
	if _, ok := m.Lookup(tm, 7); ok {
		t.Fatalf("UpdatePosition must never create an entry")
	}
	m.Put(tm, 7, LiveEntry{Field: testField(), MonsterId: 100100, Mp: 9})
	m.UpdatePosition(tm, 7, 5, 6)
	got, _ := m.Lookup(tm, 7)
	if got.X != 5 || got.Y != 6 || got.Mp != 9 {
		t.Fatalf("position update mismatch: %+v", got)
	}
}
```

NOTE: `SetX`/`SetY` on the monster model builder — check `monster/builder.go`; if the setters do not exist, add them (one-liners next to `SetMp`, same pattern as Task 2 Step 1's character additions; the Model already carries x/y if `monster/model.go` exposes `X()`/`Y()` — it does, the attack reflect path reads them today via REST).

- [x] **Step 2: Run tests to verify they fail**

Run (from `services/atlas-channel/atlas.com/channel`): `go test ./monster/ -run 'TestLiveEntryFromModel_MapsPosition|TestLiveMirror_UpdatePosition' -v`
Expected: FAIL to compile — `X`/`UpdatePosition` undefined.

- [x] **Step 3: Extend the mirror**

In `services/atlas-channel/atlas.com/channel/monster/live_mirror.go`:

1. Add to `LiveEntry`:

```go
	// X/Y: last locally-known position, seeded from the CREATED fetch and
	// updated synchronously from this pod's monster-movement fold (task-122
	// FR-5.1 — serves the reflect bounds check without a REST read).
	X int16
	Y int16
```

2. In `LiveEntryFromModel`, add `X: mo.X(), Y: mo.Y(),`.

3. Add the update-only mutator (same shape as `UpdateMp`):

```go
// UpdatePosition sets the entry's last-known position. Update only — see
// UpdateMp for why events/feeds never create entries.
func (m *LiveMirror) UpdatePosition(t tenant.Model, uniqueId uint32, x, y int16) {
	m.mu.Lock()
	defer m.mu.Unlock()
	tenantMap, ok := m.perTenant[t.Id()]
	if !ok {
		return
	}
	e, ok := tenantMap[uniqueId]
	if !ok {
		return
	}
	e.X, e.Y = x, y
	e.LastWrite = time.Now()
	tenantMap[uniqueId] = e
}
```

- [x] **Step 4: Feed the mirror from the monster movement fold**

In `services/atlas-channel/atlas.com/channel/movement/processor.go`, inside `ForMonster`, at the point where the movement summary `ms` is computed for the Kafka command (post-task-120 shape — the fold at ~line 167 of the pre-task-120 file), add immediately after the successful fold:

```go
	// Keep the live mirror's position current from the same fold the
	// movement command carries (task-122: reflect reads it in-process).
	monster.GetLiveMirror().UpdatePosition(p.t, objectId, ms.X, ms.Y)
```

Placement rule: synchronously after the fold succeeds, before (or independent of) the command emission — mirror position must not depend on Kafka success. If task-120's landed `ForMonster` computes the fold inside a goroutine, hoist the fold synchronously exactly as Task 4 did for `ForCharacter` (same wire-invariant: identical command bytes). Add a movement test:

```go
func TestForMonster_FeedsMirrorPosition(t *testing.T) {
	p, tm := newMovementTestProcessor(t)
	f := movementTestField()
	monster.GetLiveMirror().Put(tm, 8101, monster.LiveEntry{Field: f, MonsterId: 100100, Mp: 3})

	mv := model.Movement{StartX: 44, StartY: -55}
	_ = p.ForMonster(f, 1, 8101, 0, false, 0, 0, 0, model.MultiTargetForBall{}, model.RandTimeForAreaAttack{}, mv)

	got, ok := monster.GetLiveMirror().Lookup(tm, 8101)
	if !ok || got.X != 44 || got.Y != -55 {
		t.Fatalf("mirror position not fed: %+v ok=%v", got, ok)
	}
}
```

NOTE: `ForMonster`'s signature and early body are task-120's; adapt the call literals to the landed signature (Task 1 reconciliation). The ack/inbox/snap/emit logic is untouched.

- [x] **Step 5: Run tests, build, vet, commit**

Run (from `services/atlas-channel/atlas.com/channel`): `go test -race ./monster/ ./movement/ && go build ./... && go vet ./...`
Expected: PASS / clean.

```bash
git add services/atlas-channel/atlas.com/channel/monster/live_mirror.go \
        services/atlas-channel/atlas.com/channel/monster/live_mirror_test.go \
        services/atlas-channel/atlas.com/channel/monster/builder.go \
        services/atlas-channel/atlas.com/channel/movement/processor.go \
        services/atlas-channel/atlas.com/channel/movement/processor_test.go
git commit -m "feat(task-122): extend live-monster mirror with locally-fed X/Y"
```

---

### Task 11: Attack-path adoption

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go`
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_projectile.go`
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common_test.go`, `character_attack_mp_eater_test.go`, `character_attack_projectile_test.go` (update seams/types in the same commit)
- Test: append equivalence/dedup/staleness tests to `character_attack_common_test.go`

**Interfaces:**
- Consumes: `snapshot.NewProcessor(l, ctx).Get / GetBuffs` (Task 3), `monster.GetLiveMirror().Lookup/Put`, `monster.LiveEntryFromModel`, `monster.RecordMirrorFallback`, `LiveEntry{X, Y, MonsterId, Mp, MaxMp}` (Tasks 1/10), `data/skill` cached `GetEffect` (Task 9 — no code change here, already routed).
- Produces:
  - `damageInfoEntryDeps.getMonster` retyped to `func(monsterId uint32) (monster.LiveEntry, error)`
  - `mpEaterTryProc(l, ctx, mp *monster.Processor, getMonster func(uint32) (monster.LiveEntry, error), c character.Model, monsterId uint32, f field.Model, characterId uint32)`
  - Package seam `attackMonsterByIdFn = func(l logrus.FieldLogger, ctx context.Context, uniqueId uint32) (monster.Model, error)` (REST fallback for the memoized resolver)
  - `ProjectileProcessorImpl.buffs func(characterId uint32) ([]buff.Model, error)` field replacing `bp buff.Processor`

- [x] **Step 1: Write the failing tests**

Append to `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common_test.go` (reuse this file's existing fixture helpers for `AttackInfo`/`DamageInfo`/deps — read them first; add snapshot/monster imports):

```go
// FR-7.3: one monster resolve per damaged monster serves BOTH the reflect
// check and MP Eater, mirror hit or not.
func TestProcessAttack_MonsterResolveDeduped(t *testing.T) {
	// Drive processDamageInfoEntry twice for the same monster through deps
	// whose getMonster counts calls; then assert the memoized resolver
	// (buildMonsterResolver) collapses them.
	tm := newHandlerTestTenant(t) // reuse/add the package's tenant helper
	calls := 0
	prev := attackMonsterByIdFn
	attackMonsterByIdFn = func(_ logrus.FieldLogger, _ context.Context, uniqueId uint32) (monster.Model, error) {
		calls++
		f := field.NewBuilder(0, 1, 100000000).Build()
		return monster.NewModelBuilder(uniqueId, f, 100100).SetMp(50).SetMaxMp(80).SetX(10).SetY(20).Build()
	}
	defer func() { attackMonsterByIdFn = prev }()

	ctx := tenant.WithContext(context.Background(), tm)
	resolve := buildMonsterResolver(logrus.New(), ctx, tm)

	e1, err := resolve(4001)
	if err != nil {
		t.Fatalf("first resolve: %v", err)
	}
	if _, err = resolve(4001); err != nil {
		t.Fatalf("second resolve: %v", err)
	}
	if calls != 1 {
		t.Fatalf("resolver must memoize per swing: %d REST calls", calls)
	}
	if e1.Mp != 50 || e1.MaxMp != 80 || e1.X != 10 || e1.Y != 20 {
		t.Fatalf("entry mismatch: %+v", e1)
	}
	// The fallback must have backfilled the mirror for the NEXT swing.
	if got, ok := monster.GetLiveMirror().Lookup(tm, 4001); !ok || got.Mp != 50 {
		t.Fatalf("fallback must backfill mirror: %+v ok=%v", got, ok)
	}
}

func TestBuildMonsterResolver_MirrorHitZeroRest(t *testing.T) {
	tm := newHandlerTestTenant(t)
	f := field.NewBuilder(0, 1, 100000000).Build()
	monster.GetLiveMirror().Put(tm, 4002, monster.LiveEntry{Field: f, MonsterId: 100100, Mp: 7, MaxMp: 9, X: 1, Y: 2})

	prev := attackMonsterByIdFn
	attackMonsterByIdFn = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (monster.Model, error) {
		t.Fatalf("mirror hit must not call REST")
		return monster.Model{}, nil
	}
	defer func() { attackMonsterByIdFn = prev }()

	ctx := tenant.WithContext(context.Background(), tm)
	resolve := buildMonsterResolver(logrus.New(), ctx, tm)
	e, err := resolve(4002)
	if err != nil || e.Mp != 7 || e.X != 1 {
		t.Fatalf("mirror entry mismatch: %+v err=%v", e, err)
	}
}

// FR-7.4 staleness window: a skill UPDATED event between two snapshot reads
// is reflected in the second read.
func TestSnapshotStalenessWindow_SkillLevelChangesBetweenAttacks(t *testing.T) {
	tm := newHandlerTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)

	v := snapshot.GetRegistry().View(tm, 91)
	core := character.NewModelBuilder().SetId(91).SetLevel(120).MustBuild()
	_ = snapshot.GetRegistry().BackfillCore(tm, 91, core, v.CoreGen)
	_ = snapshot.GetRegistry().BackfillSkills(tm, 91, []skill.Model{
		skill.NewModelBuilder(skillconst.Id(3121004)).SetLevel(10).MustBuild(),
	}, v.SkillsGen)
	inv, _, _ := newHandlerTestInventory(t, 91) // package-local inventory fixture (same shape as snapshot tests)
	_ = snapshot.GetRegistry().BackfillInventory(tm, 91, inv, v.InvGen)
	_ = snapshot.GetRegistry().BackfillBuffs(tm, 91, nil, v.BuffsGen)

	sp := snapshot.NewProcessor(logrus.New(), ctx)
	m1, err := sp.Get(91)
	if err != nil {
		t.Fatalf("first read: %v", err)
	}
	if s, _ := m1.SkillById(skillconst.Id(3121004)); s.Level() != 10 {
		t.Fatalf("first read level: %d", s.Level())
	}

	// Event lands between attacks.
	snapshot.GetRegistry().UpsertSkill(tm, 91,
		skill.NewModelBuilder(skillconst.Id(3121004)).SetLevel(11).MustBuild())

	m2, err := sp.Get(91)
	if err != nil {
		t.Fatalf("second read: %v", err)
	}
	if s, _ := m2.SkillById(skillconst.Id(3121004)); s.Level() != 11 {
		t.Fatalf("second attack must see the new level: %d", s.Level())
	}
}
```

Also update the EXISTING tests that construct `damageInfoEntryDeps` (in `character_attack_common_test.go` and `character_attack_mp_eater_test.go`): `getMonster` fakes now return `monster.LiveEntry{MonsterId: ..., X: ..., Y: ..., Mp: ..., MaxMp: ...}` instead of `monster.Model`. Keep every assertion identical — the reflected-damage math, bounds checks, and MP-Eater behavior are pinned by those assertions (FR-4.6 for the LiveEntry retype).

- [x] **Step 2: Write the failing wire-equivalence test (FR-7.2)**

Append to `character_attack_common_test.go`:

```go
// FR-4.6/FR-7.2: for the same logical state, the snapshot-composed model
// and the decorator-built model produce byte-identical broadcast bodies.
func TestWriterEquivalence_SnapshotComposedModel(t *testing.T) {
	tm := newHandlerTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)

	inv, _, _ := newHandlerTestInventory(t, 92)
	skills := []skill.Model{skill.NewModelBuilder(skillconst.Id(3121004)).SetLevel(10).MustBuild()}
	base := character.NewModelBuilder().SetId(92).SetLevel(120).SetJobId(322).SetX(5).SetY(-5).MustBuild()

	// Path A: today's decorator order (GetById -> InventoryDecorator -> SkillModelDecorator).
	want := base.SetInventory(inv).SetSkills(skills)

	// Path B: snapshot composition.
	v := snapshot.GetRegistry().View(tm, 92)
	_ = snapshot.GetRegistry().BackfillCore(tm, 92, base, v.CoreGen)
	_ = snapshot.GetRegistry().BackfillSkills(tm, 92, skills, v.SkillsGen)
	_ = snapshot.GetRegistry().BackfillInventory(tm, 92, inv, v.InvGen)
	_ = snapshot.GetRegistry().BackfillBuffs(tm, 92, nil, v.BuffsGen)
	got, err := snapshot.NewProcessor(logrus.New(), ctx).Get(92)
	if err != nil {
		t.Fatalf("snapshot get: %v", err)
	}

	ai := newRangedAttackInfoFixture(t) // reuse/extend the package's existing AttackInfo fixture

	// packet.Encode = func(l, ctx) func(options map[string]interface{}) []byte
	// (libs/atlas-socket/packet/encoder.go:9). Use the packet-test context
	// helper the writer tests use (socket/writer/character_info_test.go:31,
	// pt "github.com/Chronicle20/atlas/libs/atlas-packet/test").
	ptCtx := pt.CreateContext("GMS", 83, 1)
	lb := logrus.New()
	opts := map[string]interface{}{}
	wantBytes := writer.CharacterAttackRangedBody(want, ai)(lb, ptCtx)(opts)
	gotBytes := writer.CharacterAttackRangedBody(got, ai)(lb, ptCtx)(opts)
	if !bytes.Equal(wantBytes, gotBytes) {
		t.Fatalf("broadcast bytes diverge:\nwant %x\ngot  %x", wantBytes, gotBytes)
	}
}
```

NOTE: the writer's mastery path performs skill-data lookups through the Task 9 cache; in this test both encodes run under identical cache/upstream conditions (no HTTP server → transient error → identical `startingMastery` fallback both times), so byte-equality holds regardless. Prefer an `ai` fixture with `SkillId=0` so the mastery/skill-data path short-circuits entirely while inventory-derived bullet data still encodes.

- [x] **Step 3: Run tests to verify they fail**

Run (from `services/atlas-channel/atlas.com/channel`): `go test ./socket/handler/ -run 'TestProcessAttack_MonsterResolveDeduped|TestBuildMonsterResolver|TestSnapshotStalenessWindow|TestWriterEquivalence' -v`
Expected: FAIL to compile — `buildMonsterResolver`, `attackMonsterByIdFn` undefined; deps type mismatch.

- [x] **Step 4: Implement the attack-path swap** (two disclosed, reviewed deviations: (1) `drainTryHeal`/`pickPocketTryProc`/`mortalBlowDeps` remain REST-backed per controller ruling R7 — monster HP mirroring was ruled out of this plan's sizing, recorded in `task-11-review.md`; (2) projectile buffs are served via a `loadBuffs` closure parameter rather than a `ProjectileProcessorImpl` struct field — functionally equivalent, buffs still served from the snapshot)

In `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go`:

1. Add the seam + resolver builder (near `damageInfoEntryDeps`):

```go
// attackMonsterByIdFn is the REST-fallback seam for the per-swing monster
// resolver (precedent: monsterByIdFn in movement).
var attackMonsterByIdFn = func(l logrus.FieldLogger, ctx context.Context, uniqueId uint32) (monster.Model, error) {
	return monster.NewProcessor(l, ctx).GetById(uniqueId)
}

// buildMonsterResolver returns a per-swing memoized monster resolve backed
// by the live mirror with REST fallback + backfill (FR-4.2): one resolve
// per damaged monster serves both the reflect check and MP Eater. Not
// goroutine-safe — processAttack runs single-goroutine per packet.
func buildMonsterResolver(l logrus.FieldLogger, ctx context.Context, t tenant.Model) func(monsterId uint32) (monster.LiveEntry, error) {
	resolved := make(map[uint32]monster.LiveEntry)
	return func(monsterId uint32) (monster.LiveEntry, error) {
		if e, ok := resolved[monsterId]; ok {
			return e, nil
		}
		e, ok := monster.GetLiveMirror().Lookup(t, monsterId)
		if !ok {
			l.Debugf("Live mirror miss for monster [%d] on attack path; falling back to REST.", monsterId)
			mo, err := attackMonsterByIdFn(l, ctx, monsterId)
			if err != nil {
				monster.RecordMirrorFallback(t, false)
				return monster.LiveEntry{}, err
			}
			monster.RecordMirrorFallback(t, true)
			e = monster.LiveEntryFromModel(mo)
			monster.GetLiveMirror().Put(t, monsterId, e)
		}
		resolved[monsterId] = e
		return e, nil
	}
}
```

2. Retype `damageInfoEntryDeps.getMonster` to `func(monsterId uint32) (monster.LiveEntry, error)` and adjust the reflect branch in `processDamageInfoEntry` (lines 134-148):

```go
			mon, mErr := deps.getMonster(di.MonsterId())
			if mErr == nil {
				entry := make([]int32, 0, len(damages))
				for _, d := range damages {
					entry = append(entry, int32(d))
				}
				r, within := computeReflect(entry, info, casterX, casterY, mon.X, mon.Y)
				if within {
					l.Debugf("reflect: char [%d] hit monster [%d] for %d reflected damage.", casterId, di.MonsterId(), r)
					if eErr := deps.emitReflectDamage(f, di.MonsterId(), mon.MonsterId, casterId, uint32(r), info.Kind); eErr != nil {
						l.WithError(eErr).Errorf("Unable to emit DAMAGE_REFLECTED for monster [%d] / character [%d].", di.MonsterId(), casterId)
					}
					reflected = true
				}
			}
```

3. Retype `mpEaterTryProc`'s monster read (signature per Interfaces above); replace the `mp.GetById(monsterId)` block (lines 240-247) with:

```go
	mon, err := getMonster(monsterId)
	if err != nil {
		l.WithError(err).Debugf("MP Eater: monster [%d] snapshot fetch failed.", monsterId)
		return
	}
	if mon.MaxMp == 0 || mon.Mp == 0 {
		return
	}
```

…and `mpEaterAbsorbAmount(mon.MaxMp, eaterEffect.X())` below. `mp` stays a parameter (still needed for `DrainMp`).

4. In `processAttack`:

```go
					sp := snapshot.NewProcessor(l, ctx)
					c, err := sp.Get(s.CharacterId())
					if err != nil {
						return err
					}
```
(replacing the `cp := character.NewProcessor(...)`/`cp.GetById(...)` pair at lines 271-275). The HP/MP cost block still needs a character processor: keep `cp := character.NewProcessor(l, ctx)` immediately above the cost gate (it is only used for `ChangeHP`/`ChangeMP` command emission — reads are gone).

Then wire the resolver:

```go
					getMonster := buildMonsterResolver(l, ctx, t)
```
(after `t := tenant.MustFromContext(ctx)`), set `deps.getMonster: getMonster`, and pass it through `onDamageApplied`:

```go
						onDamageApplied: func(monsterId uint32) {
							if ai.AttackType() == packetmodel.AttackTypeMagic && ai.SkillId() > 0 {
								mpEaterTryProc(l, ctx, mp, getMonster, c, monsterId, s.Field(), s.CharacterId())
							}
						},
```

The broadcast loop, projectile Emit, and everything else stays byte-for-byte (FR-4.5: the writers keep consuming the already-fetched `c`).

5. In `character_attack_projectile.go`: replace the `bp buff.Processor` field with `buffs func(characterId uint32) ([]buff.Model, error)`; in `NewProjectileProcessor` set `buffs: snapshot.NewProcessor(l, ctx).GetBuffs`; in `Plan` replace `p.bp.GetByCharacterId(c.Id())` with `p.buffs(c.Id())` — the fail-open "assume no buffs" branch is unchanged (FR-4.4 verdict: buffs join the snapshot; venom stays REST-memoized — `loadVenomStats` untouched).

- [x] **Step 5: Update the existing seam-based tests and run everything**

Update `damageInfoEntryDeps` fakes and `mpEaterTryProc` call sites in the three existing test files to the `LiveEntry` shapes (assertions unchanged). Then:

Run (from `services/atlas-channel/atlas.com/channel`): `go test -race ./socket/... ./character/snapshot/ && go build ./... && go vet ./...`
Expected: PASS / clean — including every pre-existing attack test (reflect math, MP-Eater gating, cost gate, projectile planning).

- [x] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/
git commit -m "feat(task-122): serve the attack path from snapshot, live mirror, and skill cache"
```

---

### Task 12: Shadow verification (env-gated sampling)

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/character/snapshot/shadow.go`
- Create: `services/atlas-channel/atlas.com/channel/character/snapshot/shadow_test.go`
- Modify: `services/atlas-channel/atlas.com/channel/character/snapshot/processor.go` (hook on the fast path)
- Modify: `services/atlas-channel/atlas.com/channel/character/snapshot/metrics.go` (divergence counter)

**Interfaces:**
- Consumes: Task 3's fetch seams (shadow fetches REST equivalents through the same seams — tests stay HTTP-free).
- Produces: `CHAR_SNAPSHOT_SHADOW_SAMPLE_RATE` behavior — on a sampled full-hit `Get`, an async bounded goroutine fetches the REST projection and compares the attack-relevant projection, logging Warn + incrementing `atlas_channel_char_snapshot_divergence_total{tenant, component}` per diverging component. Default 0 = off (production); enabled in staging soaks. Position tolerance: ±100 px (chosen constant — position naturally drifts between fold and projection; anything beyond a screen-quadrant indicates real divergence).

- [x] **Step 1: Write the failing tests**

Create `services/atlas-channel/atlas.com/channel/character/snapshot/shadow_test.go`:

```go
package snapshot

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"atlas-channel/character"
	"atlas-channel/inventory"

	"github.com/sirupsen/logrus"
)

func TestShadow_DisabledByDefault(t *testing.T) {
	resetRegistryForTest(t)
	resetShadowForTest(t)
	p, _ := newTestProcessor(t)
	counts := installFetchSeams(t, 7)
	if _, err := p.Get(7); err != nil {
		t.Fatalf("populate: %v", err)
	}
	if _, err := p.Get(7); err != nil { // full hit — the only sample point
		t.Fatalf("hit: %v", err)
	}
	waitForShadowDrain(t)
	if counts.core != 1 {
		t.Fatalf("shadow must be off by default: %+v", counts)
	}
}

func TestShadow_SamplesAndCountsDivergence(t *testing.T) {
	resetRegistryForTest(t)
	resetShadowForTest(t)
	t.Setenv("CHAR_SNAPSHOT_SHADOW_SAMPLE_RATE", "1.0")
	p, _ := newTestProcessor(t)
	counts := installFetchSeams(t, 7)
	if _, err := p.Get(7); err != nil {
		t.Fatalf("populate: %v", err)
	}

	// Make the REST projection diverge on level.
	var diverged atomic.Int32
	prevCore := coreFetchFn
	coreFetchFn = func(l logrus.FieldLogger, ctx context.Context, id uint32) (character.Model, error) {
		diverged.Add(1)
		return character.CloneModel(testCore(t, id)).SetLevel(99).MustBuild(), nil
	}
	t.Cleanup(func() { coreFetchFn = prevCore })

	if _, err := p.Get(7); err != nil {
		t.Fatalf("hit: %v", err)
	}
	waitForShadowDrain(t)
	if diverged.Load() == 0 {
		t.Fatalf("rate=1.0 must shadow-fetch on a full hit")
	}
	_ = counts
	// Divergence is observable via compareProjection directly:
	inv, _, _ := testInventory(t, 7)
	snapM := testCore(t, 7).SetInventory(inv).SetSkills(nil)
	restM := character.CloneModel(testCore(t, 7)).SetLevel(99).MustBuild().SetInventory(inv).SetSkills(nil)
	div := compareProjection(snapM, restM, nil, nil)
	if len(div) != 1 || div[0] != componentCore {
		t.Fatalf("level divergence must flag core: %v", div)
	}
}

func TestCompareProjection_PositionToleranceBanded(t *testing.T) {
	inv, _, _ := testInventory(t, 7)
	a := testCore(t, 7).SetInventory(inv).SetSkills(nil)
	// Within band: no divergence.
	b := character.CloneModel(testCore(t, 7)).SetX(testCore(t, 7).X() + 90).MustBuild().SetInventory(inv).SetSkills(nil)
	if div := compareProjection(a, b, nil, nil); len(div) != 0 {
		t.Fatalf("within-band position must not diverge: %v", div)
	}
	// Beyond band: position divergence.
	c := character.CloneModel(testCore(t, 7)).SetX(testCore(t, 7).X() + 500).MustBuild().SetInventory(inv).SetSkills(nil)
	if div := compareProjection(a, c, nil, nil); len(div) != 1 || div[0] != componentPosition {
		t.Fatalf("out-of-band position must diverge: %v", div)
	}
}

// waitForShadowDrain waits for in-flight shadow goroutines (bounded by the
// semaphore) to finish.
func waitForShadowDrain(t *testing.T) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if shadowInFlight.Load() == 0 {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("shadow goroutines did not drain")
}

var _ = inventory.Model{} // keep import if unused by edits above
```

- [x] **Step 2: Run tests to verify they fail**

Run (from `services/atlas-channel/atlas.com/channel`): `go test ./character/snapshot/ -run TestShadow -v`
Expected: FAIL to compile — `resetShadowForTest`, `compareProjection`, `shadowInFlight` undefined.

- [x] **Step 3: Implement shadow.go** (this step's own prose above, "compareProjection treats nil-vs-nil as equal," restates controller ruling R9 — CONFIRMED WRONG AS WRITTEN: `servedBuffs` is `nil` at both `maybeShadow` call sites while `shadowCompare` fetches real REST buffs, so a literal nil-vs-nil compare would false-flag `componentBuffs` for every genuinely buffed character. Landed code instead gates with `if snapBuffs != nil` and skips the buffs component with a visible Debug field when unset — confirmed correct by `task-12-review.md`)

Add to `metrics.go`:

```go
var snapshotDivergenceTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "atlas_channel_char_snapshot_divergence_total",
		Help: "Shadow-verification divergences between snapshot and REST projection, by tenant and component.",
	},
	[]string{"tenant", "component"},
)

func recordDivergence(t tenant.Model, component string) {
	snapshotDivergenceTotal.WithLabelValues(t.Id().String(), component).Inc()
}
```

Create `services/atlas-channel/atlas.com/channel/character/snapshot/shadow.go`:

```go
package snapshot

import (
	"context"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"sync/atomic"

	"atlas-channel/character"
	"atlas-channel/character/buff"
	"atlas-channel/character/skill"

	charconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

// Shadow verification (design §8, resolves PRD Open Question 5): on a
// sampled full-hit read, asynchronously fetch the REST projection and
// compare the attack-relevant fields, logging + counting divergence. This
// answers the owner's accuracy concern with runtime evidence. Default off
// (rate 0); enabled in staging soaks via CHAR_SNAPSHOT_SHADOW_SAMPLE_RATE.

const (
	envShadowSampleRate = "CHAR_SNAPSHOT_SHADOW_SAMPLE_RATE"
	// positionToleranceBand allows for natural drift between this pod's
	// movement fold and the async REST projection of the same packets.
	positionToleranceBand = 100
	shadowMaxInFlight     = 4
)

var (
	shadowRateOnce sync.Once
	shadowRateVal  float64
	shadowSem      = make(chan struct{}, shadowMaxInFlight)
	shadowInFlight atomic.Int32
)

func shadowRate() float64 {
	shadowRateOnce.Do(func() {
		v, ok := os.LookupEnv(envShadowSampleRate)
		if !ok || v == "" {
			return
		}
		f, err := strconv.ParseFloat(v, 64)
		if err != nil || f < 0 || f > 1 {
			logrus.StandardLogger().Warnf("invalid %s=%q; shadow verification disabled", envShadowSampleRate, v)
			return
		}
		shadowRateVal = f
	})
	return shadowRateVal
}

// maybeShadow samples a full-hit read. Bounded: skips the sample when
// shadowMaxInFlight comparisons are already running (never queues, never
// blocks the attack path).
func (p *Processor) maybeShadow(characterId uint32, served character.Model, servedBuffs []buff.Model) {
	rate := shadowRate()
	if rate <= 0 || rand.Float64() >= rate {
		return
	}
	select {
	case shadowSem <- struct{}{}:
	default:
		return
	}
	shadowInFlight.Add(1)
	l, ctx, t := p.l, p.ctx, p.t
	go func() {
		defer func() {
			<-shadowSem
			shadowInFlight.Add(-1)
		}()
		shadowCompare(l, ctx, t, characterId, served, servedBuffs)
	}()
}

func shadowCompare(l logrus.FieldLogger, ctx context.Context, t tenant.Model, characterId uint32, served character.Model, servedBuffs []buff.Model) {
	core, err := coreFetchFn(l, ctx, characterId)
	if err != nil {
		return // shadow is best-effort; fallback health is already metered
	}
	inv, err := inventoryFetchFn(l, ctx, characterId)
	if err != nil {
		return
	}
	skills, err := skillsFetchFn(l, ctx, characterId)
	if err != nil {
		return
	}
	restBuffs, err := buffsFetchFn(l, ctx, characterId)
	if err != nil {
		return
	}
	restModel := core.SetInventory(inv).SetSkills(skills)
	for _, component := range compareProjection(served, restModel, servedBuffs, restBuffs) {
		recordDivergence(t, component)
		l.Warnf("Snapshot shadow divergence for character [%d] component [%s].", characterId, component)
	}
}

// compareProjection compares the attack-relevant projection of two
// decorated models (+ buff sets) and returns the diverging components.
func compareProjection(snap, rest character.Model, snapBuffs, restBuffs []buff.Model) []string {
	var out []string

	if snap.Level() != rest.Level() || snap.JobId() != rest.JobId() {
		out = append(out, componentCore)
	}

	dx, dy := int32(snap.X())-int32(rest.X()), int32(snap.Y())-int32(rest.Y())
	if dx < -positionToleranceBand || dx > positionToleranceBand || dy < -positionToleranceBand || dy > positionToleranceBand {
		out = append(out, componentPosition)
	}

	if !sameWeapon(snap, rest) || !sameAssetQuantities(snap, rest) {
		out = append(out, componentInventory)
	}

	if !sameSkillLevels(snap.Skills(), rest.Skills()) {
		out = append(out, componentSkills)
	}

	if hasGateBuff(snapBuffs, charconst.TemporaryStatTypeSoulArrow) != hasGateBuff(restBuffs, charconst.TemporaryStatTypeSoulArrow) ||
		hasGateBuff(snapBuffs, charconst.TemporaryStatTypeShadowPartner) != hasGateBuff(restBuffs, charconst.TemporaryStatTypeShadowPartner) {
		out = append(out, componentBuffs)
	}
	return out
}

func sameWeapon(a, b character.Model) bool {
	wa, oka := a.Equipment().Get("weapon")
	wb, okb := b.Equipment().Get("weapon")
	if oka != okb {
		return false
	}
	if !oka {
		return true
	}
	ta, tb := uint32(0), uint32(0)
	if wa.Equipable != nil {
		ta = wa.Equipable.TemplateId()
	}
	if wb.Equipable != nil {
		tb = wb.Equipable.TemplateId()
	}
	return ta == tb
}

func sameAssetQuantities(a, b character.Model) bool {
	type key struct {
		slot       int16
		templateId uint32
	}
	build := func(m character.Model) map[key]uint32 {
		out := map[key]uint32{}
		for _, as := range m.Inventory().Consumable().Assets() {
			out[key{as.Slot(), as.TemplateId()}] = as.Quantity()
		}
		for _, as := range m.Inventory().Cash().Assets() {
			out[key{as.Slot(), as.TemplateId()}] = as.Quantity()
		}
		return out
	}
	ma, mb := build(a), build(b)
	if len(ma) != len(mb) {
		return false
	}
	for k, v := range ma {
		if mb[k] != v {
			return false
		}
	}
	return true
}

func sameSkillLevels(a, b []skill.Model) bool {
	build := func(ms []skill.Model) map[uint32]byte {
		out := map[uint32]byte{}
		for _, s := range ms {
			out[uint32(s.Id())] = s.Level()
		}
		return out
	}
	ma, mb := build(a), build(b)
	if len(ma) != len(mb) {
		return false
	}
	for k, v := range ma {
		if mb[k] != v {
			return false
		}
	}
	return true
}

func hasGateBuff(bs []buff.Model, statType charconst.TemporaryStatType) bool {
	for _, b := range bs {
		if b.Expired() {
			continue
		}
		for _, c := range b.Changes() {
			if c.Type() == string(statType) {
				return true
			}
		}
	}
	return false
}
```

Add the test-reset helper to `shadow_test.go`'s package (or registry_test.go):

```go
func resetShadowForTest(t *testing.T) {
	t.Helper()
	shadowRateOnce = sync.Once{}
	shadowRateVal = 0
}
```

Hook the sampler in `processor.go`: in `Get`'s fast path, immediately before `return m, nil`:

```go
		p.maybeShadow(characterId, m, nil)
```

and in `GetBuffs`' hit path, before returning: `p.maybeShadow(characterId, character.Model{}, filterActive(v.Buffs))` is NOT wired — buffs are compared as part of the Get sample only when a served set is available; keep the single hook in `Get` and pass `nil` (compareProjection treats nil-vs-nil as equal; the buff comparison meaningfully engages only when a future caller passes served buffs). Simplicity over coverage here: the projectile gate's correctness is already covered by the FR-7 unit tests; shadow's primary target is core/inventory/skills/position drift.

- [x] **Step 4: Run tests, build, vet, commit**

Run (from `services/atlas-channel/atlas.com/channel`): `go test -race ./character/snapshot/ && go build ./... && go vet ./...`
Expected: PASS / clean.

```bash
git add services/atlas-channel/atlas.com/channel/character/snapshot/
git commit -m "feat(task-122): env-gated shadow verification with divergence metrics"
```

---

### Task 13: atlas-character — populate Values on every STAT_CHANGED emission

**Files:**
- Modify: `services/atlas-character/atlas.com/character/character/processor.go` (every `statChangedProvider` call site)
- Test: `services/atlas-character/atlas.com/character/character/stat_values_test.go` (create)

**Interfaces:**
- Consumes: `statChangedProvider(transactionId, channel, characterId, updates, values)` (`character/producer.go:249`) — signature unchanged; the change is that NO call site passes `nil` values alongside non-empty updates anymore.
- Produces: every STAT_CHANGED event carries a `Values` map with one snake_case key per `stat.Type` in `Updates`, holding the absolute post-mutation value (additive, `omitempty` — old consumers unaffected; mixed-version rollout degrades to invalidate-and-refetch on the channel side, never wrong data).

**Site inventory** (current line numbers in `character/processor.go`; every row = replace the `nil` with a `map[string]interface{}` or extend the existing map). "Value expression" is the absolute post-mutation value already computed in the surrounding scope — hoist a local when it currently lives inside the tx closure:

| Line | Flow | Updates | Values to attach |
|---|---|---|---|
| 475 | ChangeJob | Job | `{"job": jobId}` |
| 507 | ChangeHair | Hair | `{"hair": styleId}` |
| 539 | ChangeFace | Face | `{"face": styleId}` |
| 571 | ChangeSkin | Skin | `{"skin": styleId}` |
| 640 | AwardExperience | Experience | `{"experience": current}` |
| 684 | DeductExperience | Experience | `{"experience": current}` |
| 723 | AwardLevel | Level | `{"level": current}` |
| 751 | RequestChangeMeso | Meso | hoist `newMeso := uint32(int64(c.Meso()) + int64(amount))` (the `SetMeso` argument) → `{"meso": newMeso}` |
| 768 | AttemptMesoPickUp | Meso | hoist `newMeso := uint32(int64(c.Meso()) + int64(meso))` → `{"meso": newMeso}` |
| 796 | RequestDropMeso | Meso | hoist `newMeso := c.Meso() - amount` from the tx closure → `{"meso": newMeso}` |
| 813 | RequestChangeFame | Fame | hoist `total` (the `SetFame` argument) → `{"fame": total}` |
| 826, 830, 892 | AP-distribute error paths | (empty) | leave `nil` — empty Updates carry no keys (channel no-ops) |
| 901 | AP-distribute dynamicUpdate-failed | AvailableAP | `{"available_ap": c.AP()}` (update failed → value unchanged) |
| 905 | AP-distribute success | per-ability + AvailableAP | extend the existing `values` map: `values["available_ap"] = c.AP() - spent` |
| 931 | RequestDistributeSp | AvailableSP | `{"available_sp": c.SP(sb) - uint32(amount)}` (hoist `sb`/the computed value out of the tx closure; channel invalidates on AVAILABLE_SP regardless — populated for contract completeness) |
| 1155 | ChangeHP | Hp | `{"hp": adjusted}` |
| 1205 | SetHP | Hp | the `SetHealth` argument (hoist as `newHp`) → `{"hp": newHp}` |
| 1237 | ChangeMP | Mp | `{"mp": adjusted}` — **the attack-path-critical site** |
| 1270 | ClampHP | Hp | the clamped value (hoist) → `{"hp": newHp}` |
| 1304 | ClampMP | Mp | the clamped value (hoist) → `{"mp": newMp}` |
| 1408 | Level-up growth | dynamic `sus` | extend the existing `values` map so EVERY `stat.Type` appended to `sus` has its key: read the function body and add the matching absolute for each conditional append (`available_ap`, `available_sp`, `hp`, `mp`, `max_hp`, `max_mp`, `strength`, `dexterity`, `intelligence`, `luck` as applicable) |
| 1623 | Job-change growth | AP, SP, Hp, MaxHp, Mp, MaxMp | extend existing map: `values["available_ap"] = c.AP() + addedAP; values["available_sp"] = <new SP>; values["hp"] = newMaxHP; values["mp"] = newMaxMP` (SetHealth/SetMana set current to the new max in this flow) |
| 1867, 1918 | ResetStats / RebalanceAP | AvailableAP + abilities | extend existing maps: add `values["available_ap"] = <the SetAP argument>` |

Line numbers WILL drift as edits land — locate each site by the surrounding function name, not the number. After editing, `grep -n "statChangedProvider" character/processor.go` and assert: zero remaining sites pass `nil` with a non-empty `Updates` slice.

- [x] **Step 1: Write the failing test**

Create `services/atlas-character/atlas.com/character/character/stat_values_test.go`. Use the package's existing DB-backed harness (`testDatabase(t)` in `provider_test.go:20`; follow `kafka_integration_test.go` for the message-capture pattern — read both files first and reuse their setup idioms exactly). Drive at minimum these flows end-to-end and decode the captured STAT_CHANGED events:

```go
// TestStatChanged_ValuesCompleteOnHotPaths pins task-122's producer
// contract: every STAT_CHANGED carries one snake_case key per updated stat
// with the absolute post-mutation value.
//
// Flows driven (via the ProcessorImpl against testDatabase):
//   1. ChangeMP (the per-swing path): assert values["mp"] == post-change MP.
//   2. ChangeHP: values["hp"].
//   3. AwardExperience: values["experience"].
//   4. AwardLevel (via the level path): values["level"].
//   5. RequestChangeMeso: values["meso"].
//
// Assertion helper (shared by all cases): decode the captured kafka message
// into character2.StatusEvent[character2.StatusEventStatChangedBody] and
// require, for every u in Updates, statKeyFor(u) present in Values and
// numerically equal to the post-mutation column re-read from the DB.
```

Write the real test functions per that outline (the capture mechanism — producer seam or message.Buffer inspection — must be copied from `kafka_integration_test.go`'s working pattern; do not invent a new harness). Include a `statKeyFor(stat.Type) string` helper local to the test mirroring the channel-side key table.

- [x] **Step 2: Run tests to verify they fail**

Run (from `services/atlas-character/atlas.com/character`): `go test ./character/ -run TestStatChanged_Values -v`
Expected: FAIL — Values nil on the driven flows.

- [x] **Step 3: Edit every site per the inventory table** (the table's job-change-growth row (line 1623) restates controller ruling R10 — CONFIRMED WRONG AS WRITTEN: the landed code already carried `max_hp`/`max_mp` at that site, so R10's premise was stale; the implementer instead added the genuinely-missing `available_ap`/`available_sp`/`hp`/`mp` keys there. All 24 non-empty-`Updates` call sites confirmed populated and matching atlas-channel's `statValueKeys` table exactly per the plan-adherence audit)

Mechanical rule per site: build/extend the `map[string]interface{}` with the absolute post-mutation values named in the table, hoisting tx-closure locals where noted. Do not alter `Updates` slices, event ordering, or any other emission.

- [x] **Step 4: Run the full module suite**

Run (from `services/atlas-character/atlas.com/character`): `go test -race ./... && go build ./... && go vet ./...`
Expected: PASS / clean (existing producer/processor tests must not regress — `Values` is additive).

```bash
grep -n "statChangedProvider" character/processor.go
```
Expected: every call site passes a values map or has a provably-empty `Updates` slice (826/830/892 only).

- [x] **Step 5: Commit**

```bash
git add services/atlas-character/atlas.com/character/character/processor.go \
        services/atlas-character/atlas.com/character/character/stat_values_test.go
git commit -m "feat(task-122): populate absolute Values on every STAT_CHANGED emission"
```

---

### Task 14: Full verification gate

**Files:**
- Modify: `docs/tasks/task-122-attack-path-snapshots/context.md` (final state notes)

- [x] **Step 1: Full test/vet/build in both modules** (Task 14's own implementer ran module-local `go build`/`go vet`/`go test` only, per its dispatch contract — no `-race`. The `-race` run is covered by the repo-wide flagless `tools/verify.sh`, confirmed green at HEAD `74d07bbbf` and again at `e4ee83ffe` after the later DOM-20 conversion)

```bash
cd services/atlas-channel/atlas.com/channel && go test -race ./... && go vet ./... && go build ./...
cd ../../../atlas-character/atlas.com/character && go test -race ./... && go vet ./... && go build ./...
```
Expected: all clean. Fix anything that isn't before proceeding.

- [x] **Step 2: Docker bake (mandatory — go.mod-touching services)** (not run by Task 14's own implementer — deferred to the repo-wide flagless `tools/verify.sh`, per its dispatch contract; that run is what counts as verified per CLAUDE.md and passed at HEAD `74d07bbbf` and again at `e4ee83ffe`)

From the worktree root:

```bash
docker buildx bake atlas-channel atlas-character
```
Expected: both images build. (atlas-channel's go.mod gained the prometheus dep via task-120; bake regardless — CLAUDE.md requires it for every service whose module was touched.)

- [x] **Step 3: redis-key-guard**

From the worktree root:

```bash
tools/redis-key-guard.sh
```
Expected: clean (all new state is in-process; no new go-redis usage).

- [ ] **Step 4: Zero-REST assertion sweep** (left unchecked: this step's literal "Expected: no matches" is false — one of its own three greps, `mp.GetById` against `character_attack_common.go`, matches line 1221 (beacon `monsterExists`). Controller ruling R11 required a real sweep instead of the three hard-coded greps; it held against landed code and disclosed FOUR reachable REST reads by design — beacon `monsterExists` (`:1221`), venom `effective_stats.GetByCharacterId` (`:1027`), `AVAILABLE_SP` always-invalidate (`registry.go:246-265`), and a new find, Energy Charge's rejection-only `energyReannounceAuthoritative` REST read (`character_attack_energy_charge.go:198`) — all listed with rationale in context.md's "R11 — real zero-REST sweep" section, none a defect owed to task-122. "Zero-REST attack path" is not literally true; do not check this box as written)

```bash
grep -n "GetById(cp.InventoryDecorator" services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go
grep -rn "bp.GetByCharacterId" services/atlas-channel/atlas.com/channel/socket/handler/
grep -rn "mp.GetById" services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go
```
Expected: no matches (each read now routes through snapshot/mirror/cache). The venom `effective_stats.NewProcessor(...).GetByCharacterId` read at `character_attack_common.go:332` REMAINS by design (event-coverage §6 verdict) — confirm it is still the per-swing memoized lazy fetch.

- [x] **Step 5: Update context.md and commit**

Append to context.md: final component→handler map (which consumer file maintains which snapshot component), the metric names as wired, env-var defaults, and any deltas discovered during execution (task-120 API drift, VERIFY outcomes). Commit:

```bash
git add docs/tasks/task-122-attack-path-snapshots/context.md
git commit -m "docs(task-122): execution notes and verification results"
```

- [x] **Step 6: Code review**

Per CLAUDE.md, run `superpowers:requesting-code-review` before any PR (plan-adherence + backend-guidelines reviewers). Not optional.

---

## Spec coverage self-check (writing-plans review, executed at plan time)

- FR-1 read-set inventory → design.md §2 (committed); plan consumes it (Tasks 3, 11).
- FR-2 audit → event-coverage.md (committed); VERIFY-AT-EXECUTION items → Task 1.
- FR-2.4(a) STAT_CHANGED Values → Task 13. FR-2.4(b) effective-stats stays REST → Task 11 Step 4.5 + Task 14 Step 4. FR-2.5 position → Tasks 2 (overlay), 4 (feed), 5 (MAP_CHANGED).
- FR-3.1/3.6 registry+concurrency → Task 2. FR-3.2 lazy populate + eviction → Tasks 3, 4. FR-3.3 event maintenance → Tasks 5–8. FR-3.4 fallback error semantics → Task 3 (test: FallbackFailureSurfacesError). FR-3.5 reusable API → Task 3 (`Processor.Get`/`GetBuffs`/`BuffsProvider`).
- FR-4.1 → Task 11 Step 4.4. FR-4.2 dedup → Task 11 (resolver + FR-7.3 test). FR-4.3 TTL cache → Task 9. FR-4.4 buffs-join/venom-stays → Tasks 8, 11. FR-4.5 writers → Task 11 (no writer REST; same `c`). FR-4.6 → Task 11 Step 2 + preserved seam tests.
- FR-5.1/5.2 task-120 dependency → Task 1 gate + Task 10 extension.
- FR-6 observability → Tasks 2, 9, 12 (metric names in Global Constraints); FR-6.2 debug logs → Task 3 fallback logging.
- FR-7.1 → Tasks 2, 3, 5–9 unit tests. FR-7.2 → Task 11 Step 2. FR-7.3 → Task 11 Step 1. FR-7.4 → Task 11 Step 1 (staleness test).
- NFR-4 idempotency → absolute-value mutators + redelivery tests (Tasks 2, 7, 8). NFR-5 tenant scoping → all registries tenant-keyed + isolation tests. NFR-6 → `-race` everywhere, bounded shadow goroutines. NFR-7 → hit-rate verifiable at `/api/metrics` (Task 1 confirms mount).
- PRD acceptance checklist → covered by Tasks 1–14; build/bake/guard gate → Task 14.


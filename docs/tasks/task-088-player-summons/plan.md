# Player Summons Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stand up a new `atlas-summons` microservice plus the summon packet/opcode/roster surface so player skill casts spawn owner-bound summons (puppet, attacker, buff-aura) that move, attack, take damage, and despawn correctly across every supported client version.

**Architecture:** A new Redis-backed, per-tenant `atlas-summons` service (modeled on `atlas-monsters`) owns the summon lifecycle and all behavioral logic; it is version-agnostic. `atlas-channel` is the thin packet edge (skill-cast → spawn command, three inbound relay handlers, one status-event broadcast consumer, six clientbound writers). Version variance is confined to six packets in `libs/atlas-packet/summon/` and per-version opcodes in seven socket templates. One new cross-service command pair (`ADD_PUPPET`/`REMOVE_PUPPET` on `COMMAND_TOPIC_MONSTER`) is the only `atlas-monsters` change.

**Tech Stack:** Go (DDD immutable models + Builder, functional providers, JSON:API via api2go/jsonapi, Kafka via `message.Buffer`/`message.Emit`, Redis via `libs/atlas-redis`, object ids via `libs/atlas-object-id`, leader election via `lock`), context-based multi-tenancy (`tenant.MustFromContext`). Read `context.md` before starting — it maps every blueprint file:line, cross-service struct, and resolved decision referenced below.

**Conventions for this plan:**
- All paths are relative to the worktree root (`<repo-root>/.worktrees/task-088-player-summons`).
- Cosmic baseline = `~/source/Cosmic` (Java, v83).
- The service package path is `services/atlas-summons/atlas.com/summons/`. Its Go module name is `atlas-summons` (short), declared in `services/atlas-summons/atlas.com/summons/go.mod`.
- "Mirror `<file>:<lines>`" means open that blueprint file and reproduce its structure with summon types — `context.md` lists every blueprint location.
- Every phase ends with a **verification gate** task (build/test/vet/bake/redis-guard from the worktree root). Do not start the next phase until the gate is green.
- Each phase is independently committable and leaves the tree building green.

---

## Phase 0 — Scaffold `atlas-summons` (builds green, no behavior)

Goal: a registered, bootable service with model, registry, id-allocator, REST read
surface, and empty processor — no summon behavior yet. After Phase 0,
`docker buildx bake atlas-summons` succeeds and `GET /summons/{id}` returns 404
cleanly.

### Task 0.1: Register the new service in repo manifests

**Files:**
- Modify: `.github/config/services.json`
- Modify: `go.work`
- Create: `deploy/k8s/base/atlas-summons.yaml`
- Modify: `deploy/k8s/base/kustomization.yaml`
- Modify: `deploy/k8s/base/env-configmap.yaml`

- [ ] **Step 1: Add the service to `services.json`**

Open `.github/config/services.json`, find the `atlas-monsters` entry, and add an
analogous `atlas-summons` entry (same fields — name, path, any port/metadata keys
the file uses). `docker-bake.hcl` derives its bake target from this file, so no
`docker-bake.hcl` edit is needed (confirm by reading the top of `docker-bake.hcl`).

- [ ] **Step 2: Add the module to `go.work`**

Add the line (alphabetically, next to the other service modules):

```
	./services/atlas-summons/atlas.com/summons
```

- [ ] **Step 3: Create the k8s deployment manifest**

Copy `deploy/k8s/base/atlas-monsters.yaml` to `deploy/k8s/base/atlas-summons.yaml`
and replace every `monsters`/`monster` token with `summons`/`summon` (Deployment
name, container name, image `SERVICE` arg, Service name, labels, the
`*_LEADER_ELECTION_*` env var names → `SUMMON_LEADER_*`). Keep the same REST_PORT
convention. Do **not** invent ports the manifest doesn't already pattern.

- [ ] **Step 4: Register the manifest in kustomization**

In `deploy/k8s/base/kustomization.yaml`, add `- atlas-summons.yaml` to the
`resources:` list (alphabetical position near `atlas-monsters.yaml`).

- [ ] **Step 5: Add the two new topic env vars**

In `deploy/k8s/base/env-configmap.yaml`, add (near the other `*_TOPIC_*` entries):

```yaml
  COMMAND_TOPIC_SUMMON: "command.topic.summon"
  EVENT_TOPIC_SUMMON_STATUS: "event.topic.summon.status"
```

Match the existing naming/value convention in that file (read a couple of existing
topic entries first; if the file uses fully-qualified topic strings, follow that).

- [ ] **Step 6: Commit**

```bash
git add .github/config/services.json go.work deploy/k8s/base/atlas-summons.yaml deploy/k8s/base/kustomization.yaml deploy/k8s/base/env-configmap.yaml
git commit -m "chore(atlas-summons): register new service in repo manifests"
```

### Task 0.2: Create the Go module + logger + leader config

**Files:**
- Create: `services/atlas-summons/atlas.com/summons/go.mod`
- Create: `services/atlas-summons/atlas.com/summons/logger/init.go`
- Create: `services/atlas-summons/atlas.com/summons/leaderconfig.go`

- [ ] **Step 1: Create `go.mod`**

Copy `services/atlas-monsters/atlas.com/monsters/go.mod` to the new path, change the
module line to `module atlas-summons`, and keep the same `go` directive version.
Leave `require` blocks as-is for now (they resolve via `go.work`); prune unused
requires at the end of Phase 0 with `go mod tidy`.

- [ ] **Step 2: Mirror the logger init**

Copy `services/atlas-monsters/atlas.com/monsters/logger/init.go` to the new path
verbatim (it is service-name-agnostic — it reads the service name from the caller).
Confirm no `monsters` string is hard-coded; if one is, change it to `summons`.

- [ ] **Step 3: Mirror leader config**

Copy `services/atlas-monsters/atlas.com/monsters/leaderconfig.go`, renaming env var
constants `MONSTER_LEADER_*` → `SUMMON_LEADER_*` and the helper function names if they
embed "monster". Keep the same defaults (TTL 30s, refresh 10s, backoff 5s).

- [ ] **Step 4: Verify it compiles (no main yet)**

Run: `cd services/atlas-summons/atlas.com/summons && go build ./logger/... ./... 2>&1 | head -40`
Expected: compiles (or only "no Go files in ." at root, which is fine pre-main).

- [ ] **Step 5: Commit**

```bash
git add services/atlas-summons/atlas.com/summons/go.mod services/atlas-summons/atlas.com/summons/logger services/atlas-summons/atlas.com/summons/leaderconfig.go
git commit -m "feat(atlas-summons): module skeleton, logger, leader config"
```

### Task 0.3: Summon model + builder (immutable)

**Files:**
- Create: `services/atlas-summons/atlas.com/summons/summon/model.go`
- Create: `services/atlas-summons/atlas.com/summons/summon/builder.go`
- Test: `services/atlas-summons/atlas.com/summons/summon/model_test.go`

- [ ] **Step 1: Write the failing test**

```go
package summon

import (
	"testing"
	"time"

	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-model/model/field"
	"github.com/google/uuid"
)

func testField() field.Model {
	return field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100000000)).SetInstance(uuid.Nil).Build()
}

func TestBuilderRoundTrip(t *testing.T) {
	exp := time.Unix(1700000000, 0).UTC()
	m := NewBuilder().
		SetId(1000001).
		SetOwnerCharacterId(42).
		SetSkillId(3111002).
		SetSkillLevel(20).
		SetSummonType(SummonTypePuppet).
		SetMovementType(MovementStationary).
		SetField(testField()).
		SetX(100).SetY(-50).
		SetHp(800).SetMaxHp(800).
		SetExpiresAt(exp).
		Build()

	if m.Id() != 1000001 || m.OwnerCharacterId() != 42 || m.SkillId() != 3111002 {
		t.Fatalf("identity getters wrong: %+v", m)
	}
	if m.SummonType() != SummonTypePuppet || m.MovementType() != MovementStationary {
		t.Fatalf("classification getters wrong")
	}
	if m.Hp() != 800 || m.MaxHp() != 800 || m.X() != 100 || m.Y() != -50 {
		t.Fatalf("numeric getters wrong")
	}
	if !m.ExpiresAt().Equal(exp) {
		t.Fatalf("expiresAt wrong")
	}
}

func TestAddHPClampsAtZero(t *testing.T) {
	m := NewBuilder().SetHp(100).SetMaxHp(100).Build()
	m2 := m.AddHP(-250)
	if m2.Hp() != 0 {
		t.Fatalf("expected hp clamped to 0, got %d", m2.Hp())
	}
	// original unchanged (immutability)
	if m.Hp() != 100 {
		t.Fatalf("original mutated")
	}
}
```

- [ ] **Step 2: Run it — expect FAIL (undefined types)**

Run: `cd services/atlas-summons/atlas.com/summons && go test ./summon/ -run TestBuilder -v`
Expected: FAIL — undefined `NewBuilder`, `SummonTypePuppet`, etc.

- [ ] **Step 3: Implement `model.go`**

Mirror `monster/model.go` (private fields + getters; immutable transform methods
return `Clone(m)....Build()`). Define the enums and full field set:

```go
package summon

import (
	"time"

	"github.com/Chronicle20/atlas-model/model/field"
)

type SummonType string

const (
	SummonTypePuppet   SummonType = "PUPPET"
	SummonTypeAttacker SummonType = "ATTACKER"
	SummonTypeBuffAura SummonType = "BUFF_AURA"
)

type MovementType byte

const (
	MovementStationary   MovementType = 0
	MovementFollow       MovementType = 1
	MovementCircleFollow MovementType = 3
)

type Model struct {
	id               uint32
	ownerCharacterId uint32
	skillId          uint32
	skillLevel       byte
	summonType       SummonType
	movementType     MovementType
	fld              field.Model
	x                int16
	y                int16
	stance           byte
	hp               int32
	maxHp            int32
	animated         bool
	spawnTime        time.Time
	expiresAt        time.Time

	// Beholder-only aura snapshot (zero-valued for all other summons)
	nextHealAt    time.Time
	nextBuffAt    time.Time
	healAmount    int16
	healInterval  time.Duration
	buffInterval  time.Duration
	buffSourceId  int32
	buffLevel     byte
	buffDuration  int32
	buffChanges   []StatChange
}

// StatChange mirrors the buff command's change element (see context.md §5).
type StatChange struct {
	Type   string
	Amount int32
}

func (m Model) Id() uint32                 { return m.id }
func (m Model) OwnerCharacterId() uint32   { return m.ownerCharacterId }
func (m Model) SkillId() uint32            { return m.skillId }
func (m Model) SkillLevel() byte           { return m.skillLevel }
func (m Model) SummonType() SummonType     { return m.summonType }
func (m Model) MovementType() MovementType { return m.movementType }
func (m Model) Field() field.Model         { return m.fld }
func (m Model) X() int16                   { return m.x }
func (m Model) Y() int16                   { return m.y }
func (m Model) Stance() byte               { return m.stance }
func (m Model) Hp() int32                  { return m.hp }
func (m Model) MaxHp() int32               { return m.maxHp }
func (m Model) Animated() bool             { return m.animated }
func (m Model) SpawnTime() time.Time       { return m.spawnTime }
func (m Model) ExpiresAt() time.Time       { return m.expiresAt }
func (m Model) IsPuppet() bool             { return m.summonType == SummonTypePuppet }
func (m Model) IsBeholder() bool           { return m.summonType == SummonTypeBuffAura }
func (m Model) NextHealAt() time.Time      { return m.nextHealAt }
func (m Model) NextBuffAt() time.Time      { return m.nextBuffAt }
func (m Model) HealAmount() int16          { return m.healAmount }
func (m Model) HealInterval() time.Duration { return m.healInterval }
func (m Model) BuffInterval() time.Duration { return m.buffInterval }
func (m Model) BuffSourceId() int32        { return m.buffSourceId }
func (m Model) BuffLevel() byte            { return m.buffLevel }
func (m Model) BuffDuration() int32        { return m.buffDuration }
func (m Model) BuffChanges() []StatChange  { return m.buffChanges }

// Move returns a copy at the new position/stance (non-stationary summons only).
func (m Model) Move(x int16, y int16, stance byte) Model {
	return Clone(m).SetX(x).SetY(y).SetStance(stance).Build()
}

// AddHP returns a copy with hp adjusted by delta, clamped to [0, maxHp].
func (m Model) AddHP(delta int32) Model {
	hp := m.hp + delta
	if hp < 0 {
		hp = 0
	}
	if m.maxHp > 0 && hp > m.maxHp {
		hp = m.maxHp
	}
	return Clone(m).SetHp(hp).Build()
}
```

- [ ] **Step 4: Implement `builder.go`**

Mirror `monster/builder.go`: a `ModelBuilder` with every field, `NewBuilder()`,
`Clone(m)`, a `SetX` per field (slice fields copied), and `Build()`. Example shape:

```go
package summon

import (
	"time"

	"github.com/Chronicle20/atlas-model/model/field"
)

type ModelBuilder struct {
	id               uint32
	ownerCharacterId uint32
	skillId          uint32
	skillLevel       byte
	summonType       SummonType
	movementType     MovementType
	fld              field.Model
	x                int16
	y                int16
	stance           byte
	hp               int32
	maxHp            int32
	animated         bool
	spawnTime        time.Time
	expiresAt        time.Time
	nextHealAt       time.Time
	nextBuffAt       time.Time
	healAmount       int16
	healInterval     time.Duration
	buffInterval     time.Duration
	buffSourceId     int32
	buffLevel        byte
	buffDuration     int32
	buffChanges      []StatChange
}

func NewBuilder() *ModelBuilder { return &ModelBuilder{animated: true} }

func Clone(m Model) *ModelBuilder {
	changes := make([]StatChange, len(m.buffChanges))
	copy(changes, m.buffChanges)
	return &ModelBuilder{
		id: m.id, ownerCharacterId: m.ownerCharacterId, skillId: m.skillId,
		skillLevel: m.skillLevel, summonType: m.summonType, movementType: m.movementType,
		fld: m.fld, x: m.x, y: m.y, stance: m.stance, hp: m.hp, maxHp: m.maxHp,
		animated: m.animated, spawnTime: m.spawnTime, expiresAt: m.expiresAt,
		nextHealAt: m.nextHealAt, nextBuffAt: m.nextBuffAt, healAmount: m.healAmount,
		healInterval: m.healInterval, buffInterval: m.buffInterval,
		buffSourceId: m.buffSourceId, buffLevel: m.buffLevel, buffDuration: m.buffDuration,
		buffChanges: changes,
	}
}

func (b *ModelBuilder) SetId(v uint32) *ModelBuilder               { b.id = v; return b }
func (b *ModelBuilder) SetOwnerCharacterId(v uint32) *ModelBuilder { b.ownerCharacterId = v; return b }
func (b *ModelBuilder) SetSkillId(v uint32) *ModelBuilder          { b.skillId = v; return b }
func (b *ModelBuilder) SetSkillLevel(v byte) *ModelBuilder         { b.skillLevel = v; return b }
func (b *ModelBuilder) SetSummonType(v SummonType) *ModelBuilder   { b.summonType = v; return b }
func (b *ModelBuilder) SetMovementType(v MovementType) *ModelBuilder { b.movementType = v; return b }
func (b *ModelBuilder) SetField(v field.Model) *ModelBuilder       { b.fld = v; return b }
func (b *ModelBuilder) SetX(v int16) *ModelBuilder                 { b.x = v; return b }
func (b *ModelBuilder) SetY(v int16) *ModelBuilder                 { b.y = v; return b }
func (b *ModelBuilder) SetStance(v byte) *ModelBuilder             { b.stance = v; return b }
func (b *ModelBuilder) SetHp(v int32) *ModelBuilder                { b.hp = v; return b }
func (b *ModelBuilder) SetMaxHp(v int32) *ModelBuilder             { b.maxHp = v; return b }
func (b *ModelBuilder) SetAnimated(v bool) *ModelBuilder           { b.animated = v; return b }
func (b *ModelBuilder) SetSpawnTime(v time.Time) *ModelBuilder     { b.spawnTime = v; return b }
func (b *ModelBuilder) SetExpiresAt(v time.Time) *ModelBuilder     { b.expiresAt = v; return b }
func (b *ModelBuilder) SetNextHealAt(v time.Time) *ModelBuilder    { b.nextHealAt = v; return b }
func (b *ModelBuilder) SetNextBuffAt(v time.Time) *ModelBuilder    { b.nextBuffAt = v; return b }
func (b *ModelBuilder) SetHealAmount(v int16) *ModelBuilder        { b.healAmount = v; return b }
func (b *ModelBuilder) SetHealInterval(v time.Duration) *ModelBuilder { b.healInterval = v; return b }
func (b *ModelBuilder) SetBuffInterval(v time.Duration) *ModelBuilder { b.buffInterval = v; return b }
func (b *ModelBuilder) SetBuffSourceId(v int32) *ModelBuilder      { b.buffSourceId = v; return b }
func (b *ModelBuilder) SetBuffLevel(v byte) *ModelBuilder          { b.buffLevel = v; return b }
func (b *ModelBuilder) SetBuffDuration(v int32) *ModelBuilder      { b.buffDuration = v; return b }
func (b *ModelBuilder) SetBuffChanges(v []StatChange) *ModelBuilder { b.buffChanges = v; return b }

func (b *ModelBuilder) Build() Model {
	return Model{
		id: b.id, ownerCharacterId: b.ownerCharacterId, skillId: b.skillId,
		skillLevel: b.skillLevel, summonType: b.summonType, movementType: b.movementType,
		fld: b.fld, x: b.x, y: b.y, stance: b.stance, hp: b.hp, maxHp: b.maxHp,
		animated: b.animated, spawnTime: b.spawnTime, expiresAt: b.expiresAt,
		nextHealAt: b.nextHealAt, nextBuffAt: b.nextBuffAt, healAmount: b.healAmount,
		healInterval: b.healInterval, buffInterval: b.buffInterval,
		buffSourceId: b.buffSourceId, buffLevel: b.buffLevel, buffDuration: b.buffDuration,
		buffChanges: b.buffChanges,
	}
}
```

> Note: confirm the actual import path of `field.Model` by reading the top of
> `monster/model.go` (it imports the same `field` package). Use the identical import.

- [ ] **Step 5: Run the test — expect PASS**

Run: `cd services/atlas-summons/atlas.com/summons && go test ./summon/ -run 'TestBuilder|TestAddHP' -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-summons/atlas.com/summons/summon/model.go services/atlas-summons/atlas.com/summons/summon/builder.go services/atlas-summons/atlas.com/summons/summon/model_test.go
git commit -m "feat(atlas-summons): immutable summon model + builder"
```

### Task 0.4: Redis registry (store + field index + owner index)

**Files:**
- Create: `services/atlas-summons/atlas.com/summons/summon/registry.go`
- Test: `services/atlas-summons/atlas.com/summons/summon/registry_test.go`

- [ ] **Step 1: Write the failing test (miniredis-backed)**

Mirror how `monster/registry_test.go` (if present) or another service registry test
spins up `miniredis` and a `goredis.Client`. The test asserts: put a summon → it
appears in the field index and owner index; remove it → both indexes drop it.

```go
package summon

import (
	"context"
	"testing"

	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-model/model/field"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/google/uuid"
)

func newTestRegistry(t *testing.T) (*Registry, tenant.Model, context.Context) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mr.Close)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	reg := newRegistry(rc) // unexported constructor used by InitRegistry
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	return reg, ten, tenant.WithContext(context.Background(), ten)
}

func TestRegistryPutIndexesByFieldAndOwner(t *testing.T) {
	reg, ten, ctx := newTestRegistry(t)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100000000)).SetInstance(uuid.Nil).Build()
	m := NewBuilder().SetId(1000001).SetOwnerCharacterId(42).SetField(f).
		SetSummonType(SummonTypePuppet).SetMovementType(MovementStationary).Build()

	if err := reg.Put(ctx, ten, m); err != nil {
		t.Fatal(err)
	}
	inField, err := reg.GetInField(ctx, ten, f)
	if err != nil || len(inField) != 1 || inField[0].Id() != 1000001 {
		t.Fatalf("field index miss: %v %+v", err, inField)
	}
	byOwner, err := reg.GetByOwner(ctx, ten, 42)
	if err != nil || len(byOwner) != 1 {
		t.Fatalf("owner index miss: %v %+v", err, byOwner)
	}

	if err := reg.Remove(ctx, ten, 1000001); err != nil {
		t.Fatal(err)
	}
	inField, _ = reg.GetInField(ctx, ten, f)
	byOwner, _ = reg.GetByOwner(ctx, ten, 42)
	if len(inField) != 0 || len(byOwner) != 0 {
		t.Fatalf("indexes not cleared on remove")
	}
}
```

- [ ] **Step 2: Run it — expect FAIL**

Run: `cd services/atlas-summons/atlas.com/summons && go test ./summon/ -run TestRegistry -v`
Expected: FAIL — undefined `newRegistry`/`Registry`/methods.

- [ ] **Step 3: Implement `registry.go`**

Mirror `monster/registry.go`. Use a JSON-serializable `storedSummon` (export-free
struct with all Model fields) plus `toStored`/`fromStored`. Singleton via `sync.Once`.
Namespaces: store `"summon"`, field index `"summon-map"`, owner index
`"summon-owner"`. Key suffixes mirror monster (see `context.md` §3).

```go
package summon

import (
	"context"
	"fmt"
	"sync"

	atlasredis "github.com/Chronicle20/atlas-redis"
	"github.com/Chronicle20/atlas-model/model/field"
	tenant "github.com/Chronicle20/atlas-tenant"
	goredis "github.com/redis/go-redis/v9"
)

type Registry struct {
	reg      *atlasredis.Registry[string, storedSummon]
	fieldIdx *atlasredis.KeyedSet[string]
	ownerIdx *atlasredis.KeyedSet[string]
}

var registry *Registry
var once sync.Once

func newRegistry(rc *goredis.Client) *Registry {
	return &Registry{
		reg:      atlasredis.NewRegistry[string, storedSummon](rc, "summon", func(s string) string { return s }),
		fieldIdx: atlasredis.NewKeyedSet[string](rc, "summon-map", func(s string) string { return s }),
		ownerIdx: atlasredis.NewKeyedSet[string](rc, "summon-owner", func(s string) string { return s }),
	}
}

func InitRegistry(rc *goredis.Client) { once.Do(func() { registry = newRegistry(rc) }) }
func GetRegistry() *Registry          { return registry }

func storeSuffix(t tenant.Model, id uint32) string {
	return fmt.Sprintf("%s:%d", t.Id().String(), id)
}
func fieldSuffix(t tenant.Model, f field.Model) string {
	return fmt.Sprintf("%s:%d:%d:%d:%s", t.Id().String(), f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String())
}
func ownerSuffix(t tenant.Model, characterId uint32) string {
	return fmt.Sprintf("%s:%d", t.Id().String(), characterId)
}

func (r *Registry) Put(ctx context.Context, t tenant.Model, m Model) error {
	if err := r.reg.Put(ctx, storeSuffix(t, m.Id()), toStored(m)); err != nil {
		return err
	}
	member := fmt.Sprintf("%d", m.Id())
	if err := r.fieldIdx.Add(ctx, fieldSuffix(t, m.Field()), member); err != nil {
		return err
	}
	return r.ownerIdx.Add(ctx, ownerSuffix(t, m.OwnerCharacterId()), member)
}

func (r *Registry) Get(ctx context.Context, t tenant.Model, id uint32) (Model, error) {
	s, err := r.reg.Get(ctx, storeSuffix(t, id))
	if err != nil {
		return Model{}, err
	}
	return fromStored(s), nil
}

func (r *Registry) GetInField(ctx context.Context, t tenant.Model, f field.Model) ([]Model, error) {
	return r.loadMembers(ctx, t, r.fieldIdx, fieldSuffix(t, f))
}
func (r *Registry) GetByOwner(ctx context.Context, t tenant.Model, characterId uint32) ([]Model, error) {
	return r.loadMembers(ctx, t, r.ownerIdx, ownerSuffix(t, characterId))
}

func (r *Registry) loadMembers(ctx context.Context, t tenant.Model, set *atlasredis.KeyedSet[string], key string) ([]Model, error) {
	members, err := set.Members(ctx, key)
	if err != nil {
		return nil, err
	}
	out := make([]Model, 0, len(members))
	for _, member := range members {
		var id uint32
		if _, err := fmt.Sscanf(member, "%d", &id); err != nil {
			continue
		}
		m, err := r.Get(ctx, t, id)
		if err != nil {
			// stale index entry; skip
			continue
		}
		out = append(out, m)
	}
	return out, nil
}

func (r *Registry) Update(ctx context.Context, t tenant.Model, id uint32, fn func(Model) Model) (Model, error) {
	s, err := r.reg.Update(ctx, storeSuffix(t, id), func(cur storedSummon) storedSummon {
		return toStored(fn(fromStored(cur)))
	})
	if err != nil {
		return Model{}, err
	}
	return fromStored(s), nil
}

func (r *Registry) Remove(ctx context.Context, t tenant.Model, id uint32) error {
	m, err := r.Get(ctx, t, id)
	if err == nil {
		member := fmt.Sprintf("%d", id)
		_ = r.fieldIdx.Remove(ctx, fieldSuffix(t, m.Field()), member)
		_ = r.ownerIdx.Remove(ctx, ownerSuffix(t, m.OwnerCharacterId()), member)
	}
	return r.reg.Remove(ctx, storeSuffix(t, id))
}

// GetAll returns every stored summon across tenants (used by sweep tasks).
func (r *Registry) GetAll(ctx context.Context) ([]Model, error) {
	stored, err := r.reg.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Model, 0, len(stored))
	for _, s := range stored {
		out = append(out, fromStored(s))
	}
	return out, nil
}
```

Add the `storedSummon` struct + `toStored`/`fromStored` in the same file (or
`stored.go`). It must carry the tenant id is NOT needed inside the value (keys carry
it), but it MUST carry the full field (world/channel/map/instance) so `fromStored`
can rebuild `field.Model` via `field.NewBuilder(...).SetInstance(...).Build()`. Times
serialize as RFC3339 strings or unix; pick what `monster`'s stored model does for its
time fields and match it.

- [ ] **Step 4: Run the test — expect PASS**

Run: `cd services/atlas-summons/atlas.com/summons && go test ./summon/ -run TestRegistry -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-summons/atlas.com/summons/summon/registry.go services/atlas-summons/atlas.com/summons/summon/registry_test.go
git commit -m "feat(atlas-summons): redis registry with field + owner indexes"
```

### Task 0.5: Object-id allocator wrapper

**Files:**
- Create: `services/atlas-summons/atlas.com/summons/summon/id_allocator.go`

- [ ] **Step 1: Implement (mirror `monster/id_allocator.go` verbatim, renamed)**

```go
package summon

import (
	"context"
	"sync"

	objectid "github.com/Chronicle20/atlas-object-id"
	tenant "github.com/Chronicle20/atlas-tenant"
	goredis "github.com/redis/go-redis/v9"
)

type IdAllocator struct{ inner objectid.Allocator }

var idAllocator *IdAllocator
var idAllocatorOnce sync.Once

func InitIdAllocator(rc *goredis.Client) {
	idAllocatorOnce.Do(func() { idAllocator = &IdAllocator{inner: objectid.NewRedisAllocator(rc)} })
}
func GetIdAllocator() *IdAllocator { return idAllocator }

func (a *IdAllocator) Allocate(ctx context.Context, t tenant.Model) uint32 {
	id, err := a.inner.Allocate(ctx, t)
	if err != nil {
		return objectid.MinId
	}
	return id
}
func (a *IdAllocator) Release(ctx context.Context, t tenant.Model, id uint32) {
	_ = a.inner.Release(ctx, t, id)
}
```

Confirm the exact import path of `objectid` from `monster/id_allocator.go`.

- [ ] **Step 2: Build**

Run: `cd services/atlas-summons/atlas.com/summons && go build ./summon/...`
Expected: compiles.

- [ ] **Step 3: Commit**

```bash
git add services/atlas-summons/atlas.com/summons/summon/id_allocator.go
git commit -m "feat(atlas-summons): object-id allocator wrapper"
```

### Task 0.6: Kafka event topic + producer providers (CREATED / DESTROYED)

**Files:**
- Create: `services/atlas-summons/atlas.com/summons/summon/kafka.go`
- Create: `services/atlas-summons/atlas.com/summons/summon/producer.go`

- [ ] **Step 1: Implement `kafka.go` (event envelope + type consts)**

Mirror `monster/kafka.go`. Define the event topic env var, the `StatusEvent[E]`
envelope (exported — `atlas-channel` consumes it), the event-type constants, and the
body structs for the five event types (start with CREATED + DESTROYED bodies;
MOVED/ATTACKED/DAMAGED bodies are added in later phases).

```go
package summon

import (
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

const EnvEventTopicSummonStatus = "EVENT_TOPIC_SUMMON_STATUS"

const (
	EventSummonStatusCreated   = "CREATED"
	EventSummonStatusMoved     = "MOVED"
	EventSummonStatusAttacked  = "ATTACKED"
	EventSummonStatusDamaged   = "DAMAGED"
	EventSummonStatusDestroyed = "DESTROYED"
)

type StatusEvent[E any] struct {
	WorldId          world.Id   `json:"worldId"`
	ChannelId        channel.Id `json:"channelId"`
	MapId            _map.Id    `json:"mapId"`
	Instance         uuid.UUID  `json:"instance"`
	SummonId         uint32     `json:"summonId"`
	OwnerCharacterId uint32     `json:"ownerCharacterId"`
	SkillId          uint32     `json:"skillId"`
	Type             string     `json:"type"`
	Body             E          `json:"body"`
}

type StatusEventCreatedBody struct {
	SkillLevel   byte `json:"skillLevel"`
	MovementType byte `json:"movementType"`
	X            int16 `json:"x"`
	Y            int16 `json:"y"`
	Stance       byte  `json:"stance"`
	Puppet       bool  `json:"puppet"`
	Animated     bool  `json:"animated"`
}

type StatusEventDestroyedBody struct {
	Animated bool `json:"animated"`
}
```

- [ ] **Step 2: Implement `producer.go` (providers)**

Mirror `monster/producer.go`. Use `producer.CreateKey(int(f.MapId()))` and
`producer.SingleMessageProvider(key, &value)`.

```go
package summon

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func createdEventProvider(m Model) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(m.Field().MapId()))
	value := StatusEvent[StatusEventCreatedBody]{
		WorldId: m.Field().WorldId(), ChannelId: m.Field().ChannelId(),
		MapId: m.Field().MapId(), Instance: m.Field().Instance(),
		SummonId: m.Id(), OwnerCharacterId: m.OwnerCharacterId(), SkillId: m.SkillId(),
		Type: EventSummonStatusCreated,
		Body: StatusEventCreatedBody{
			SkillLevel: m.SkillLevel(), MovementType: byte(m.MovementType()),
			X: m.X(), Y: m.Y(), Stance: m.Stance(),
			Puppet: m.IsPuppet(), Animated: m.Animated(),
		},
	}
	return producer.SingleMessageProvider(key, &value)
}

func destroyedEventProvider(m Model, animated bool) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(m.Field().MapId()))
	value := StatusEvent[StatusEventDestroyedBody]{
		WorldId: m.Field().WorldId(), ChannelId: m.Field().ChannelId(),
		MapId: m.Field().MapId(), Instance: m.Field().Instance(),
		SummonId: m.Id(), OwnerCharacterId: m.OwnerCharacterId(), SkillId: m.SkillId(),
		Type: EventSummonStatusDestroyed,
		Body: StatusEventDestroyedBody{Animated: animated},
	}
	return producer.SingleMessageProvider(key, &value)
}
```

Confirm the exact import paths for `producer`, `model`, and `kafka.Message` by
copying them from `monster/producer.go`.

- [ ] **Step 3: Build**

Run: `cd services/atlas-summons/atlas.com/summons && go build ./summon/...`
Expected: compiles.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-summons/atlas.com/summons/summon/kafka.go services/atlas-summons/atlas.com/summons/summon/producer.go
git commit -m "feat(atlas-summons): summon status event envelope + providers"
```

### Task 0.7: Processor interface + skeleton impl

**Files:**
- Create: `services/atlas-summons/atlas.com/summons/summon/processor.go`

- [ ] **Step 1: Implement the interface + impl (read methods + emit closure now; behavior methods land in later phases)**

Mirror `monster/processor.go:79-93` for `NewProcessor`/`emit`. Define the full
interface signature surface up front so later phases only fill bodies:

```go
package summon

import (
	"context"

	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-model/model/field"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	GetById(id uint32) (Model, error)
	GetInField(f field.Model) ([]Model, error)
	Spawn(f field.Model, ownerCharacterId uint32, skillId uint32, skillLevel byte, x int16, y int16) (Model, error)
	Move(id uint32, senderCharacterId uint32, x int16, y int16, stance byte) error
	Attack(id uint32, senderCharacterId uint32, direction byte, targets []AttackTarget) error
	Damage(id uint32, senderCharacterId uint32, amount int32, monsterIdFrom uint32) error
	Despawn(id uint32, animated bool) error
	DespawnAllForOwner(ownerCharacterId uint32) error
}

// AttackTarget is one {monster, reported damage} pair from a summon-attack packet.
type AttackTarget struct {
	MonsterId uint32
	Damage    uint32
}

type ProcessorImpl struct {
	l    logrus.FieldLogger
	ctx  context.Context
	t    tenant.Model
	emit func(topic string, provider model.Provider[[]kafka.Message]) error
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l: l, ctx: ctx, t: tenant.MustFromContext(ctx),
		emit: func(topic string, provider model.Provider[[]kafka.Message]) error {
			return producer.ProviderImpl(l)(ctx)(topic)(provider)
		},
	}
}

func (p *ProcessorImpl) GetById(id uint32) (Model, error) {
	return GetRegistry().Get(p.ctx, p.t, id)
}
func (p *ProcessorImpl) GetInField(f field.Model) ([]Model, error) {
	return GetRegistry().GetInField(p.ctx, p.t, f)
}

// The behavioral methods are implemented in later phases. For Phase 0 they return nil
// so the service builds; each later phase replaces the body and adds a test.
func (p *ProcessorImpl) Spawn(f field.Model, ownerCharacterId uint32, skillId uint32, skillLevel byte, x int16, y int16) (Model, error) {
	return Model{}, nil
}
func (p *ProcessorImpl) Move(id uint32, senderCharacterId uint32, x int16, y int16, stance byte) error { return nil }
func (p *ProcessorImpl) Attack(id uint32, senderCharacterId uint32, direction byte, targets []AttackTarget) error { return nil }
func (p *ProcessorImpl) Damage(id uint32, senderCharacterId uint32, amount int32, monsterIdFrom uint32) error { return nil }
func (p *ProcessorImpl) Despawn(id uint32, animated bool) error { return nil }
func (p *ProcessorImpl) DespawnAllForOwner(ownerCharacterId uint32) error { return nil }
```

> These are not stubs left in a deliverable — every one is replaced with a real
> implementation + test in Phases 1–5. Phase 0's gate only requires the read methods
> to work. If the project's "no TODO" discipline is a concern, leave a clear
> `// implemented in Phase N` comment (not `// TODO`) on each, and ensure the phase
> that fills it has a task below.

- [ ] **Step 2: Build**

Run: `cd services/atlas-summons/atlas.com/summons && go build ./summon/...`
Expected: compiles.

- [ ] **Step 3: Commit**

```bash
git add services/atlas-summons/atlas.com/summons/summon/processor.go
git commit -m "feat(atlas-summons): processor interface + read methods"
```

### Task 0.8: JSON:API resource + REST read endpoints

**Files:**
- Create: `services/atlas-summons/atlas.com/summons/summon/resource.go`
- Create: `services/atlas-summons/atlas.com/summons/summon/rest.go`
- Create: `services/atlas-summons/atlas.com/summons/world/resource.go`
- Create: `services/atlas-summons/atlas.com/summons/rest/handler.go` (copy of monsters' helper if not shared)

- [ ] **Step 1: Implement the REST model + Transform + GET single + field list**

Mirror `monster/resource.go`/`rest.go` and `world/resource.go`. The resource type
name is `summons` (`GetName()` returns `"summons"`). RestModel attributes per PRD §5.1:
`ownerCharacterId`, `skillId`, `skillLevel`, `summonType`, `movementType`, `x`, `y`,
`hp`, `maxHp`, `expiresAt`, `worldId`, `channelId`, `mapId`, `instance`. Routes:
- `GET /summons/{summonId}` (single)
- `GET /worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/summons` (field list)

Copy the `rest/handler.go` helper package from `atlas-monsters` if it is service-local
(it provides `RegisterHandler`, `ParseXId`, `HandlerDependency`). Add a `ParseSummonId`
mirroring `ParseMonsterId`.

- [ ] **Step 2: Build**

Run: `cd services/atlas-summons/atlas.com/summons && go build ./...`
Expected: compiles (main still missing — that's next).

- [ ] **Step 3: Commit**

```bash
git add services/atlas-summons/atlas.com/summons/summon/resource.go services/atlas-summons/atlas.com/summons/summon/rest.go services/atlas-summons/atlas.com/summons/world services/atlas-summons/atlas.com/summons/rest
git commit -m "feat(atlas-summons): JSON:API summons resource + read endpoints"
```

### Task 0.9: `main.go` boot wiring + Phase 0 verification gate

**Files:**
- Create: `services/atlas-summons/atlas.com/summons/main.go`

- [ ] **Step 1: Implement `main.go`**

Mirror `monsters/main.go` exactly (boot order in `context.md` §2), but:
- service name `"atlas-summons"`, consumer group `consumergroup.Resolve("Summon Registry Service")`
- registry inits: `summon.InitIdAllocator(rc)`, `summon.InitRegistry(rc)`
- REST route initializers: `summon.InitResource(GetServer())`, `world.InitResource(GetServer())`, `/metrics`, `/debug/consumers`
- leader lock name `"summons-sweep"`; in Phase 0, `registerSweepTasks` is an empty
  function (the expiry sweep is registered in Phase 1). Wire the leader-election
  scaffolding now so later phases only append tasks.
- Kafka consumer registration: none in Phase 0 (added in Phase 1). Leave the `cmf`
  wiring present but with no `InitConsumers` calls yet, matching the monsters shape.

Provide the `GetServer()`/`GetServer().GetPrefix()` server-info helper (copy
`monsters`' equivalent — likely a small `server.go` or inline in `main.go`; replicate
whatever monsters does).

- [ ] **Step 2: `go mod tidy` and build the service**

Run:
```bash
cd services/atlas-summons/atlas.com/summons && go mod tidy && go build ./...
```
Expected: builds clean.

- [ ] **Step 3: Phase 0 verification gate (worktree root)**

Run from the worktree root:
```bash
go vet ./services/atlas-summons/... && \
go test -race ./services/atlas-summons/... && \
go build ./services/atlas-summons/... && \
docker buildx bake atlas-summons && \
GOWORK=off tools/redis-key-guard.sh
```
Expected: all clean. The bake step is mandatory — it is the only check that catches a
missing `services.json`/Dockerfile wiring for the new service.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-summons/atlas.com/summons/main.go services/atlas-summons/atlas.com/summons/go.mod services/atlas-summons/atlas.com/summons/go.sum
git commit -m "feat(atlas-summons): main boot wiring; Phase 0 scaffold builds green"
```

---

## Phase 1 — Roster + spawn/despawn lifecycle + v83 spawn/remove + channel wiring

Goal: casting a v83 summon skill spawns the correct summon (type/movement/HP/duration)
and broadcasts `SummonSpawn`; it despawns on logout/channel-change/map-change, on
re-cast (same skill), on conflicting-class cast, and on duration expiry, each
broadcasting `SummonRemove`. This phase wires the full skill-cast → spawn → broadcast
→ despawn loop for v83 only.

### Task 1.1: Roster table in `libs/atlas-constants/summon`

**Files:**
- Create: `libs/atlas-constants/summon/roster.go`
- Test: `libs/atlas-constants/summon/roster_test.go`

- [ ] **Step 1: Write the failing test**

```go
package summon

import "testing"

func TestLookupKnownSummon(t *testing.T) {
	e, ok := Lookup(3111002) // Ranger Puppet
	if !ok || e.Type != TypePuppet || e.Movement != MovementStationary {
		t.Fatalf("ranger puppet wrong: %+v ok=%v", e, ok)
	}
	e, ok = Lookup(1321007) // Beholder
	if !ok || e.Type != TypeBuffAura || e.Movement != MovementFollow {
		t.Fatalf("beholder wrong: %+v ok=%v", e, ok)
	}
	e, ok = Lookup(3111005) // Silver Hawk: attacker, stun, circle-follow
	if !ok || e.Type != TypeAttacker || e.Movement != MovementCircleFollow || !e.Stun {
		t.Fatalf("silver hawk wrong: %+v ok=%v", e, ok)
	}
}

func TestLookupUnknownSummon(t *testing.T) {
	if _, ok := Lookup(99999999); ok {
		t.Fatalf("expected miss for unknown id")
	}
	if IsSummonSkill(99999999) {
		t.Fatalf("IsSummonSkill should be false for unknown id")
	}
	if !IsSummonSkill(1321007) {
		t.Fatalf("IsSummonSkill should be true for Beholder")
	}
}

func TestRosterHas21Entries(t *testing.T) {
	if len(roster) != 21 {
		t.Fatalf("expected 21 roster entries, got %d", len(roster))
	}
}
```

- [ ] **Step 2: Run it — expect FAIL**

Run: `cd libs/atlas-constants && go test ./summon/ -v`
Expected: FAIL — undefined `Lookup`, `Entry`, etc.

- [ ] **Step 3: Implement `roster.go`**

Keys reference the existing skill-id constants in
`libs/atlas-constants/skill/constants.go` (verify each constant name by reading that
file; the literal ids below are from `design.md` Appendix A and must match). Type and
movement come from Cosmic `StatEffect.java:1766-1797` / `Summon.isPuppet()/isStationary()`.

```go
package summon

type Type string

const (
	TypePuppet   Type = "PUPPET"
	TypeAttacker Type = "ATTACKER"
	TypeBuffAura Type = "BUFF_AURA"
)

type Movement byte

const (
	MovementStationary   Movement = 0
	MovementFollow       Movement = 1
	MovementCircleFollow Movement = 3
)

type Entry struct {
	Type     Type
	Movement Movement
	Stun     bool // applies STUN monster status on hit
	Freeze   bool // applies FREEZE monster status on hit
	OneShot  bool // self-cancels after a single attack (Gaviota)
}

// roster: the 21 v83 summon skills. Adding a summon = one row here, no engine change.
var roster = map[uint32]Entry{
	3111002:  {Type: TypePuppet, Movement: MovementStationary},                 // Ranger Puppet
	3211002:  {Type: TypePuppet, Movement: MovementStationary},                 // Sniper Puppet
	13111004: {Type: TypePuppet, Movement: MovementStationary},                 // Wind Archer Puppet
	5211001:  {Type: TypeAttacker, Movement: MovementStationary},               // Octopus
	5220002:  {Type: TypeAttacker, Movement: MovementStationary},               // Wrath of the Octopi
	3111005:  {Type: TypeAttacker, Movement: MovementCircleFollow, Stun: true}, // Silver Hawk
	3211005:  {Type: TypeAttacker, Movement: MovementCircleFollow, Stun: true}, // Golden Eagle
	3121006:  {Type: TypeAttacker, Movement: MovementCircleFollow},             // Phoenix
	3221005:  {Type: TypeAttacker, Movement: MovementCircleFollow, Freeze: true}, // Frostprey
	2311006:  {Type: TypeAttacker, Movement: MovementCircleFollow},             // Summon Dragon
	5211002:  {Type: TypeAttacker, Movement: MovementCircleFollow, OneShot: true}, // Gaviota
	2121005:  {Type: TypeAttacker, Movement: MovementFollow, Freeze: true},     // Elquines
	2221005:  {Type: TypeAttacker, Movement: MovementFollow},                   // Ifrit (I/L)
	2321003:  {Type: TypeAttacker, Movement: MovementFollow},                   // Bahamut
	11001004: {Type: TypeAttacker, Movement: MovementFollow},                   // Dawn Warrior Soul
	12001004: {Type: TypeAttacker, Movement: MovementFollow},                   // Blaze Wizard Flame
	12111004: {Type: TypeAttacker, Movement: MovementFollow},                   // Blaze Wizard Ifrit
	13001004: {Type: TypeAttacker, Movement: MovementFollow},                   // Wind Archer Storm
	14001005: {Type: TypeAttacker, Movement: MovementFollow},                   // Night Walker Darkness
	15001004: {Type: TypeAttacker, Movement: MovementFollow},                   // Thunder Breaker Lightning
	1321007:  {Type: TypeBuffAura, Movement: MovementFollow},                   // Dark Knight Beholder
}

func Lookup(skillId uint32) (Entry, bool) {
	e, ok := roster[skillId]
	return e, ok
}

func IsSummonSkill(skillId uint32) bool {
	_, ok := roster[skillId]
	return ok
}
```

> After writing this, replace the literal map keys with the named constants from
> `libs/atlas-constants/skill/constants.go` (e.g. `skill.RangerPuppetId`) **if** they
> are importable without an import cycle. If the skill package importing here would
> cycle, keep the literals and add a comment naming each constant. Verify every id
> against `skill/constants.go` before finishing.

- [ ] **Step 4: Run the test — expect PASS**

Run: `cd libs/atlas-constants && go test ./summon/ -v`
Expected: PASS (all three tests).

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-constants/summon/roster.go libs/atlas-constants/summon/roster_test.go
git commit -m "feat(atlas-constants): summon roster classification table (21 v83 skills)"
```

### Task 1.2: Skill-effect data client in `atlas-summons`

**Files:**
- Create: `services/atlas-summons/atlas.com/summons/data/skill/effect/rest.go`
- Create: `services/atlas-summons/atlas.com/summons/data/skill/effect/model.go`
- Create: `services/atlas-summons/atlas.com/summons/data/skill/rest.go`
- Create: `services/atlas-summons/atlas.com/summons/data/skill/model.go`
- Create: `services/atlas-summons/atlas.com/summons/data/skill/requests.go`
- Create: `services/atlas-summons/atlas.com/summons/data/skill/processor.go`

- [ ] **Step 1: Implement the data client**

Mirror `services/atlas-channel/.../data/skill/` (rest/model/requests/processor) but
expose the fields summons needs. The effect `Model` getters required:
`Duration() int32`, `X() int16`, `Y() int16`, `Prop() float64`,
`MonsterStatus() map[string]uint32`, **and** `WeaponAttack() int16`,
`MagicAttack() int16` (the latter two are the additions over the channel-side model).
The REST request: base url env `DATA_SERVICE_URL`, path `data/skills/%d`,
`requests.GetRequest[RestModel]` + an `Extract` from RestModel → Model. Copy the
RestModel attribute tags from `services/atlas-channel/.../data/skill/effect/rest.go`
(it already deserializes `weaponAttack`/`magicAttack`/`duration`/`x`/`y`/`prop`/
`monsterStatus`).

The skill processor exposes `GetEffect(skillId uint32, level byte) (effect.Model, error)`
following the channel-side processor's REST fetch + per-level effect selection.

- [ ] **Step 2: Build**

Run: `cd services/atlas-summons/atlas.com/summons && go build ./data/...`
Expected: compiles.

- [ ] **Step 3: Commit**

```bash
git add services/atlas-summons/atlas.com/summons/data
git commit -m "feat(atlas-summons): skill-effect data client (exposes weaponAttack/magicAttack)"
```

### Task 1.3: Spawn + despawn processor logic

**Files:**
- Modify: `services/atlas-summons/atlas.com/summons/summon/processor.go`
- Test: `services/atlas-summons/atlas.com/summons/summon/processor_spawn_test.go`

- [ ] **Step 1: Write the failing test**

The test uses miniredis (as in Task 0.4) and a fake effect provider. It asserts:
- Spawn of a puppet skill creates a summon with `SummonTypePuppet`, stationary
  movement, hp = effect.X, persisted in field + owner indexes.
- A second spawn of the **same** skill for the same owner removes the first (FR-2.4).
- Spawn of a roster-miss skill id returns `(Model{}, nil)` and persists nothing (FR-1.3).

```go
func TestSpawnPuppetPersistsAndIndexes(t *testing.T) {
	// arrange miniredis + InitRegistry + InitIdAllocator + a stub effect provider
	// returning Duration=60000ms, X=800 for skill 3111002.
	// act: p.Spawn(field, owner=42, skillId=3111002, level=20, x=100, y=-50)
	// assert: returned model is puppet/stationary, hp==800, GetByOwner(42) len==1
}

func TestRecastReplacesSameSkill(t *testing.T) {
	// spawn 3111002 twice for owner 42; assert GetByOwner(42) len==1 and id differs
}

func TestSpawnUnknownSkillNoOp(t *testing.T) {
	// p.Spawn(..., skillId=99999999, ...) -> (Model{}, nil); GetByOwner len==0
}
```

> Implementer: flesh these out against the real `Spawn` signature. Inject the effect
> data via a small interface field on `ProcessorImpl` (default = the real
> `data/skill` processor; tests substitute a stub) so the spawn logic is unit-testable
> without a live atlas-data. Add that interface + default wiring in Step 3.

- [ ] **Step 2: Run it — expect FAIL**

Run: `cd services/atlas-summons/atlas.com/summons && go test ./summon/ -run 'TestSpawn|TestRecast' -v`
Expected: FAIL — Spawn returns empty.

- [ ] **Step 3: Implement `Spawn`, `Despawn`, `DespawnAllForOwner`**

Replace the Phase-0 bodies. Logic per `design.md` §7:

```go
func (p *ProcessorImpl) Spawn(f field.Model, ownerCharacterId uint32, skillId uint32, skillLevel byte, x int16, y int16) (Model, error) {
	entry, ok := summonconst.Lookup(skillId)
	if !ok {
		p.l.Debugf("Skill [%d] is not a summon; no spawn.", skillId) // FR-1.3 graceful no-op
		return Model{}, nil
	}

	// FR-2.4 / FR-2.5: remove same-skill instance and conflicting-mobility-class instance.
	existing, _ := GetRegistry().GetByOwner(p.ctx, p.t, ownerCharacterId)
	for _, e := range existing {
		if e.SkillId() == skillId || conflictsMobility(entry.Movement, e.MovementType()) {
			_ = p.Despawn(e.Id(), false)
		}
	}

	eff, err := p.effects.GetEffect(skillId, skillLevel)
	if err != nil {
		p.l.WithError(err).Warnf("No effect data for summon skill [%d]; aborting spawn.", skillId)
		return Model{}, err
	}

	id := GetIdAllocator().Allocate(p.ctx, p.t)
	now := time.Now()
	expires := now.Add(time.Duration(eff.Duration()) * time.Millisecond)

	hp := int32(0)
	if entry.Type == summonconst.TypePuppet {
		hp = int32(eff.X())
	} else if entry.Type == summonconst.TypeBuffAura {
		hp = int32(eff.X()) + 1 // Cosmic Beholder hp = x + 1
	}

	b := NewBuilder().
		SetId(id).SetOwnerCharacterId(ownerCharacterId).SetSkillId(skillId).SetSkillLevel(skillLevel).
		SetSummonType(SummonType(entry.Type)).SetMovementType(MovementType(entry.Movement)).
		SetField(f).SetX(x).SetY(y).SetHp(hp).SetMaxHp(hp).
		SetSpawnTime(now).SetExpiresAt(expires).SetAnimated(true)

	// Beholder aura snapshot is added in Phase 5 (this builder is extended there).
	m := b.Build()

	if err := GetRegistry().Put(p.ctx, p.t, m); err != nil {
		GetIdAllocator().Release(p.ctx, p.t, id)
		return Model{}, err
	}
	if err := p.emit(EnvEventTopicSummonStatus, createdEventProvider(m)); err != nil {
		p.l.WithError(err).Errorf("Unable to emit CREATED for summon [%d].", id)
	}
	// Puppet ADD_PUPPET emission is added in Phase 4.
	// Beholder timer init is added in Phase 5.
	return m, nil
}

func (p *ProcessorImpl) Despawn(id uint32, animated bool) error {
	m, err := GetRegistry().Get(p.ctx, p.t, id)
	if err != nil {
		return nil // already gone
	}
	if err := GetRegistry().Remove(p.ctx, p.t, id); err != nil {
		return err
	}
	GetIdAllocator().Release(p.ctx, p.t, id)
	if err := p.emit(EnvEventTopicSummonStatus, destroyedEventProvider(m, animated)); err != nil {
		p.l.WithError(err).Errorf("Unable to emit DESTROYED for summon [%d].", id)
	}
	// Puppet REMOVE_PUPPET emission is added in Phase 4.
	// Beholder timer cleanup is implicit (registry removal) — see Phase 5.
	return nil
}

func (p *ProcessorImpl) DespawnAllForOwner(ownerCharacterId uint32) error {
	ms, err := GetRegistry().GetByOwner(p.ctx, p.t, ownerCharacterId)
	if err != nil {
		return err
	}
	for _, m := range ms {
		_ = p.Despawn(m.Id(), false)
	}
	return nil
}

// conflictsMobility implements Cosmic StatEffect.java:1024-1029: a new stationary
// summon cancels the existing stationary one; a new non-stationary cancels the
// existing non-stationary one.
func conflictsMobility(newMove summonconst.Movement, existing MovementType) bool {
	newStationary := newMove == summonconst.MovementStationary
	existingStationary := existing == MovementStationary
	return newStationary == existingStationary
}
```

Add the `effects` interface field to `ProcessorImpl` and default-wire it in
`NewProcessor`:

```go
type effectSource interface {
	GetEffect(skillId uint32, level byte) (effect.Model, error)
}
// in ProcessorImpl: effects effectSource
// in NewProcessor: effects: skilldata.NewProcessor(l, ctx),
```

Use import alias `summonconst "github.com/Chronicle20/atlas-constants/summon"` and
`effect`/`skilldata` for the new `data/skill` packages.

- [ ] **Step 4: Run the tests — expect PASS**

Run: `cd services/atlas-summons/atlas.com/summons && go test ./summon/ -run 'TestSpawn|TestRecast' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-summons/atlas.com/summons/summon/processor.go services/atlas-summons/atlas.com/summons/summon/processor_spawn_test.go
git commit -m "feat(atlas-summons): spawn/despawn lifecycle with re-cast + conflict cancel"
```

### Task 1.4: `COMMAND_TOPIC_SUMMON` consumer (SPAWN)

**Files:**
- Create: `services/atlas-summons/atlas.com/summons/kafka/consumer/summon/kafka.go`
- Create: `services/atlas-summons/atlas.com/summons/kafka/consumer/summon/consumer.go`
- Modify: `services/atlas-summons/atlas.com/summons/main.go`

- [ ] **Step 1: Define the command envelope + SPAWN body**

Mirror `monsters/kafka/consumer/monster/kafka.go`. Command env var
`COMMAND_TOPIC_SUMMON`, type consts `SPAWN`/`MOVE`/`ATTACK`/`DAMAGE`, the `Command[E]`
envelope (world/channel/map/instance/summonId/type/body), and `SpawnCommandBody`:

```go
const EnvCommandTopic = "COMMAND_TOPIC_SUMMON"
const (
	CommandTypeSpawn  = "SPAWN"
	CommandTypeMove   = "MOVE"
	CommandTypeAttack = "ATTACK"
	CommandTypeDamage = "DAMAGE"
)

type Command[E any] struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
	SummonId  uint32     `json:"summonId"`
	Type      string     `json:"type"`
	Body      E          `json:"body"`
}

type SpawnCommandBody struct {
	OwnerCharacterId uint32 `json:"ownerCharacterId"`
	SkillId          uint32 `json:"skillId"`
	SkillLevel       byte   `json:"skillLevel"`
	X                int16  `json:"x"`
	Y                int16  `json:"y"`
}
```

- [ ] **Step 2: Implement the consumer + SPAWN handler**

Mirror `monsters/kafka/consumer/monster/consumer.go`: `InitConsumers(l)(cmf)(group)`
registers `COMMAND_TOPIC_SUMMON`; `InitHandlers(l)(rf)` registers `handleSpawnCommand`.

```go
func handleSpawnCommand(l logrus.FieldLogger, ctx context.Context, c Command[SpawnCommandBody]) {
	if c.Type != CommandTypeSpawn {
		return
	}
	f := field.NewBuilder(c.WorldId, c.ChannelId, c.MapId).SetInstance(c.Instance).Build()
	_, err := summon.NewProcessor(l, ctx).Spawn(f, c.Body.OwnerCharacterId, c.Body.SkillId, c.Body.SkillLevel, c.Body.X, c.Body.Y)
	if err != nil {
		l.WithError(err).Errorf("Failed to spawn summon for owner [%d] skill [%d].", c.Body.OwnerCharacterId, c.Body.SkillId)
	}
}
```

- [ ] **Step 3: Wire into `main.go`**

Add to the consumer block: `summoncmd.InitConsumers(l)(cmf)(consumerGroupId)` and
`summoncmd.InitHandlers(l)(consumer.GetManager().RegisterHandler)`.

- [ ] **Step 4: Build + test**

Run: `cd services/atlas-summons/atlas.com/summons && go build ./... && go test ./...`
Expected: clean.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-summons/atlas.com/summons/kafka/consumer/summon services/atlas-summons/atlas.com/summons/main.go
git commit -m "feat(atlas-summons): COMMAND_TOPIC_SUMMON consumer with SPAWN handler"
```

### Task 1.5: Character-lifecycle despawn-cascade consumer

**Files:**
- Create: `services/atlas-summons/atlas.com/summons/kafka/consumer/character/kafka.go`
- Create: `services/atlas-summons/atlas.com/summons/kafka/consumer/character/consumer.go`
- Modify: `services/atlas-summons/atlas.com/summons/main.go`

- [ ] **Step 1: Mirror the character status event envelope**

Define (matching `context.md` §5) `EnvEventTopicCharacterStatus = "EVENT_TOPIC_CHARACTER_STATUS"`,
the `StatusEvent[E]` envelope (`TransactionId`, `WorldId`, `CharacterId`, `Type`, `Body`),
the type consts `LOGOUT`/`CHANNEL_CHANGED`/`MAP_CHANGED`, and minimal body structs (the
cascade only needs `CharacterId` from the envelope, so the bodies can be `any`/empty
structs — but decode them faithfully to avoid Kafka parse errors).

- [ ] **Step 2: Implement the three handlers (all call `DespawnAllForOwner`)**

```go
func handleLogout(l logrus.FieldLogger, ctx context.Context, e StatusEvent[LogoutBody]) {
	if e.Type != StatusEventTypeLogout {
		return
	}
	_ = summon.NewProcessor(l, ctx).DespawnAllForOwner(e.CharacterId)
}
// handleChannelChanged + handleMapChanged identical, gated on their Type const.
```

> Despawn-all-by-owner is correct for all three: on channel/map change the summons must
> not follow (they are bound to the field at spawn). Re-cast in the new field is the
> player's responsibility, matching Cosmic `Character.java:3769-3791`.

- [ ] **Step 3: Wire into `main.go`** (consumer + handlers, same pattern as Task 1.4).

- [ ] **Step 4: Build + test** — `go build ./... && go test ./...` clean.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-summons/atlas.com/summons/kafka/consumer/character services/atlas-summons/atlas.com/summons/main.go
git commit -m "feat(atlas-summons): despawn cascade on logout/channel/map change"
```

### Task 1.6: Expiry sweep task

**Files:**
- Create: `services/atlas-summons/atlas.com/summons/tasks/task.go`
- Create: `services/atlas-summons/atlas.com/summons/summon/expiry_task.go`
- Modify: `services/atlas-summons/atlas.com/summons/main.go`
- Test: `services/atlas-summons/atlas.com/summons/summon/expiry_task_test.go`

- [ ] **Step 1: Copy the task runner**

Copy `monsters/tasks/task.go` verbatim (the `Task` interface + `Register(l,ctx)(t)`
tick loop). It is service-agnostic.

- [ ] **Step 2: Write the failing test**

```go
func TestExpirySweepDespawnsExpired(t *testing.T) {
	// miniredis + registry; put a summon with ExpiresAt in the past and one in the future.
	// run the sweep's processing once; assert expired one is gone, future one remains.
}
```

- [ ] **Step 3: Implement `expiry_task.go`**

```go
type ExpiryTask struct {
	l        logrus.FieldLogger
	ctx      context.Context
	interval time.Duration
}

func NewExpiryTask(l logrus.FieldLogger, ctx context.Context, interval time.Duration) *ExpiryTask {
	return &ExpiryTask{l: l, ctx: ctx, interval: interval}
}
func (t *ExpiryTask) SleepTime() time.Duration { return t.interval }
func (t *ExpiryTask) Run() {
	all, err := GetRegistry().GetAll(t.ctx)
	if err != nil {
		return
	}
	now := time.Now()
	for _, m := range all {
		if m.ExpiresAt().IsZero() || now.Before(m.ExpiresAt()) {
			continue
		}
		// Rebuild tenant-scoped context from the model's field is NOT possible (no tenant
		// in field); instead the sweep must iterate per-tenant. See note below.
	}
}
```

> **Tenant-context note:** `GetAll` returns models without tenant context, but
> `Despawn` needs a tenant-scoped processor. Match how `atlas-monsters`' sweep tasks
> obtain tenant context (read `monster/status_task.go` — its registry `GetMonsters()`
> returns a `map[tenant.Model][]Model`). **Change `GetRegistry().GetAll` to return
> `map[tenant.Model][]Model`** (mirror the monster registry's tenant-keyed accessor),
> so the sweep can build `tenant.WithContext(t.ctx, ten)` per tenant and call
> `summon.NewProcessor(l, tctx).Despawn(m.Id(), true)`. Update Task 0.4's `GetAll`
> accordingly and re-run its test. This is the one place the registry must be
> tenant-aware in its enumeration — copy the exact mechanism from
> `monster/registry.go`'s `GetMonsters()`.

- [ ] **Step 4: Register in `main.go`** inside `registerSweepTasks`:
`tasks.Register(l, ctx)(summon.NewExpiryTask(l, ctx, time.Second))`.

- [ ] **Step 5: Run the test — expect PASS**, then build.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-summons/atlas.com/summons/tasks services/atlas-summons/atlas.com/summons/summon/expiry_task.go services/atlas-summons/atlas.com/summons/summon/expiry_task_test.go services/atlas-summons/atlas.com/summons/summon/registry.go services/atlas-summons/atlas.com/summons/main.go
git commit -m "feat(atlas-summons): leader-elected expiry sweep + tenant-keyed enumeration"
```

### Task 1.7: v83 `SummonSpawn` + `SummonRemove` clientbound packets

**Files:**
- Create: `libs/atlas-packet/summon/clientbound/spawn.go`
- Create: `libs/atlas-packet/summon/clientbound/remove.go`
- Test: `libs/atlas-packet/summon/clientbound/spawn_test.go`
- Test: `libs/atlas-packet/summon/clientbound/remove_test.go`

- [ ] **Step 1: Write the failing round-trip tests (all variants)**

```go
package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas-packet/test"
)

func TestSummonSpawn(t *testing.T) {
	in := NewSummonSpawn(42, 1000001, 3111002, 20, 100, -50, 0, byte(MovementStationary), true /*puppet*/, false /*animated*/)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, in.Encode, in.Decode, nil)
		})
	}
}

func TestSummonRemove(t *testing.T) {
	in := NewSummonRemove(42, 1000001, true)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, in.Encode, in.Decode, nil)
		})
	}
}
```

> Each clientbound packet needs a matching `Decode` for the round-trip harness (the
> harness encodes then decodes and asserts zero leftover bytes). The `Decode` reads the
> same fields the `Encode` writes — see `monster/clientbound/spawn.go` which pairs
> `Encode`/`Decode`.

- [ ] **Step 2: Run — expect FAIL** (`cd libs/atlas-packet && go test ./summon/...`).

- [ ] **Step 3: Implement `spawn.go` (v83 layout; version branches stubbed to v83 for now)**

Encode per the Cosmic `:1149` layout (`context.md` §4). For Phase 1 the encoding is
v83-correct for **all** variants (v84/v86 are byte-identical to v83; the genuine
per-version deltas — the `0x0A` marker value and any field width changes — are
harvested and branched in Phase 6). Write the `0x0A` byte unconditionally now and add a
`// Phase 6: IDA-confirm per-version marker` comment (not a TODO). Pull
`t := tenant.MustFromContext(ctx)` even though Phase-1 branches are trivial, so the
structure is ready for Phase 6.

```go
type SummonSpawn struct {
	ownerId      uint32
	oid          uint32
	skillId      uint32
	level        byte
	x            int16
	y            int16
	stance       byte
	movementType byte
	puppet       bool
	animated     bool
}

func NewSummonSpawn(ownerId, oid, skillId uint32, level byte, x, y int16, stance, movementType byte, puppet, animated bool) SummonSpawn {
	return SummonSpawn{ownerId, oid, skillId, level, x, y, stance, movementType, puppet, animated}
}

func (m SummonSpawn) Encode(l logrus.FieldLogger, ctx context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	_ = tenant.MustFromContext(ctx)
	return func(map[string]interface{}) []byte {
		w.WriteInt(m.ownerId)
		w.WriteInt(m.oid)
		w.WriteInt(m.skillId)
		w.WriteByte(0x0A) // v83 marker; per-version value confirmed via IDA in Phase 6
		w.WriteByte(m.level)
		w.WriteInt16(m.x)
		w.WriteInt16(m.y)
		w.WriteByte(m.stance)
		w.WriteShort(0)
		w.WriteByte(m.movementType)
		w.WriteBool(!m.puppet)   // attack flag = !isPuppet
		w.WriteBool(!m.animated) // !animated
		return w.Bytes()
	}
}
// Decode mirrors the reads; see monster/clientbound/spawn.go for the pattern.
```

`remove.go` per `:1172`: int owner, int oid, byte (4 if animated else 1).

- [ ] **Step 4: Run — expect PASS** for every variant.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-packet/summon/clientbound/spawn.go libs/atlas-packet/summon/clientbound/remove.go libs/atlas-packet/summon/clientbound/spawn_test.go libs/atlas-packet/summon/clientbound/remove_test.go
git commit -m "feat(atlas-packet): SummonSpawn + SummonRemove clientbound (v83 layout)"
```

### Task 1.8: atlas-channel — skill-cast branch emits SPAWN

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/summon/processor.go`
- Create: `services/atlas-channel/atlas.com/channel/summon/producer.go`
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_skill_use.go`

- [ ] **Step 1: Implement the channel-side summon command emitter**

Mirror `services/atlas-channel/.../monster/processor.go:56-59` +
`monster/producer.go`. `Processor.Spawn(f, owner, skillId, level, x, y) error` emits
`COMMAND_TOPIC_SUMMON SPAWN` via `producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(SpawnCommandProvider(...))`.
Reuse the command envelope/body **shape** from the summons service (re-declare it
channel-side in this package — services do not import each other's internals; declare a
matching `Command[SpawnCommandBody]` here, keyed on `producer.CreateKey(int(owner))`).

- [ ] **Step 2: Add the branch in `character_skill_use.go`**

After `GetEffect` (line ~70), before/alongside the existing skill handling:

```go
if summon.IsSummonSkill(sui.SkillId()) {
	if err := summoncmd.NewProcessor(l, ctx).Spawn(s.Field(), s.CharacterId(), sui.SkillId(), byte(sui.SkillLevel()), s.X(), s.Y()); err != nil {
		l.WithError(err).Errorf("Unable to request summon spawn for character [%d] skill [%d].", s.CharacterId(), sui.SkillId())
	}
}
```

Use `summon "github.com/Chronicle20/atlas-constants/summon"` for the predicate and
`summoncmd` for the channel-side emitter package. Confirm the session getters for the
caster position (`s.X()`/`s.Y()` or via `s.Field()`/movement state — read
`session.Model` to get the right accessor; if position is not on the session, fetch the
caster's last movement position the same way other handlers do). The summon must NOT
short-circuit the existing skill effect application (buff/cooldown still apply).

- [ ] **Step 3: Build** `services/atlas-channel/...`.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/summon services/atlas-channel/atlas.com/channel/socket/handler/character_skill_use.go
git commit -m "feat(atlas-channel): route summon skill casts to COMMAND_TOPIC_SUMMON SPAWN"
```

### Task 1.9: atlas-channel — `EVENT_TOPIC_SUMMON_STATUS` consumer broadcasts spawn/remove

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/summon/kafka.go` (event envelope mirror)
- Create: `services/atlas-channel/atlas.com/channel/kafka/consumer/summon/consumer.go`
- Modify: `services/atlas-channel/atlas.com/channel/main.go`

- [ ] **Step 1: Mirror the status event envelope channel-side**

Re-declare `StatusEvent[E]` + the CREATED/DESTROYED bodies + type consts +
`EnvEventTopicSummonStatus` in `services/atlas-channel/.../summon/kafka.go` (matching the
producer side from Task 0.6).

- [ ] **Step 2: Implement the consumer + CREATED/DESTROYED handlers**

Mirror `services/atlas-channel/.../kafka/consumer/monster/consumer.go` (InitConsumers
34-40, InitHandlers 42-118, `handleStatusEventCreated`/`Destroyed` 120-156). CREATED →
broadcast `SummonSpawn` via `ForSessionsInMap`; DESTROYED → broadcast `SummonRemove`.

```go
func handleSummonCreated(sc server.Model, wp writer.Producer) message.Handler[summon.StatusEvent[summon.StatusEventCreatedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e summon.StatusEvent[summon.StatusEventCreatedBody]) {
		if e.Type != summon.EventSummonStatusCreated {
			return
		}
		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}
		_ = _map.NewProcessor(l, ctx).ForSessionsInMap(sc.Field(e.MapId, e.Instance),
			session.Announce(l)(ctx)(wp)(summonpkt.SummonSpawnWriter)(
				writer.SummonSpawnBody(e.OwnerCharacterId, e.SummonId, e.SkillId, e.Body)))
	}
}
```

Add the `writer.SummonSpawnBody`/`SummonRemoveBody` body builders next to the existing
monster body builders (they construct the packet struct from the event). Reference the
writer name constants from `libs/atlas-packet/summon/clientbound` (e.g.
`summonpkt.SummonSpawnWriter`, `SummonRemoveWriter` — add these string consts in the
packet package).

- [ ] **Step 3: Register consumer + handlers in `main.go`** (mirror the monster status
consumer registration block).

- [ ] **Step 4: Register the writers** — add `summonpkt.SummonSpawnWriter` and
`summonpkt.SummonRemoveWriter` to `produceWriters()` (`main.go:586-686`).

- [ ] **Step 5: Build** `services/atlas-channel/...`.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/summon services/atlas-channel/atlas.com/channel/kafka/consumer/summon services/atlas-channel/atlas.com/channel/main.go libs/atlas-packet/summon
git commit -m "feat(atlas-channel): broadcast SummonSpawn/SummonRemove from status events"
```

### Task 1.10: v83 opcodes for SummonSpawn/SummonRemove

**Files:**
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_83_1.json`

- [ ] **Step 1: Add writer opcode entries**

In `template_gms_83_1.json`'s `socket.writers[]`, add entries for `SummonSpawn` and
`SummonRemove`. **The opcode byte values must be the real v83 client send-opcodes,
harvested from IDA** (do not invent). Run the IDA harvest sub-step:
- Load the v83 IDB (`reference_ida_harvest_subagents`), find the client handler that
  reads `CUserPool::OnSummonedEnterField` / summon spawn and the corresponding outbound
  opcode table; record the send opcodes for spawn and remove.
- If a v83 reference packet capture or the existing Cosmic `SendOpcode` enum is
  available and cross-checks, use it to confirm.

Add:
```json
{ "opCode": "0x<spawn>", "writer": "SummonSpawn" },
{ "opCode": "0x<remove>", "writer": "SummonRemove" }
```

> The exact bytes are produced by the Phase-1 harvest; record them in
> `summon-packet-delta.md` (created in Phase 6) under a "v83 opcodes" section as you go.

- [ ] **Step 2: Validate the JSON** parses (`python3 -m json.tool < template_gms_83_1.json > /dev/null` or the repo's config-lint if one exists).

- [ ] **Step 3: Commit**

```bash
git add services/atlas-configurations/seed-data/templates/template_gms_83_1.json
git commit -m "feat(atlas-configurations): seed v83 SummonSpawn/SummonRemove opcodes"
```

### Task 1.11: Phase 1 verification gate

- [ ] **Step 1: Run the full gate (worktree root)**

```bash
go vet ./services/atlas-summons/... ./services/atlas-channel/... ./libs/atlas-packet/... ./libs/atlas-constants/... && \
go test -race ./services/atlas-summons/... ./libs/atlas-packet/summon/... ./libs/atlas-constants/summon/... && \
go build ./services/atlas-summons/... ./services/atlas-channel/... ./libs/... && \
docker buildx bake atlas-summons atlas-channel && \
GOWORK=off tools/redis-key-guard.sh
```
Expected: all clean.

- [ ] **Step 2: Commit (if any tidy changes)** — otherwise the phase is complete.

---

## Phase 2 — Movement (`SummonMove`)

Goal: the owner's client-driven summon movement is validated (ownership), the
authoritative position is updated, and the move is rebroadcast to other in-range
sessions. Stationary summons are never expected to move.

### Task 2.1: `SummonMove` clientbound + serverbound packets

**Files:**
- Create: `libs/atlas-packet/summon/clientbound/move.go` (+ test)
- Create: `libs/atlas-packet/summon/serverbound/move.go` (+ test)

- [ ] **Step 1: Write failing round-trip tests** for both, looping `test.Variants`
(mirror Task 1.7's test shape). The serverbound test builds a reader from a known byte
slice and asserts the decoded fields.

- [ ] **Step 2: Implement clientbound `move.go`** per Cosmic `:2284`: int cid, int oid,
startPos (short x, short y), raw movement bytes (carry the raw movement blob as
`[]byte` written via `WriteByteArray`). Implement serverbound `move.go` decoding the
inbound `MoveSummonHandler` layout: int oid, startPos, raw movement bytes (read the
remaining bytes). Confirm exact inbound layout from Cosmic `MoveSummonHandler.java:36-59`.

- [ ] **Step 3: Run tests — expect PASS** all variants.

- [ ] **Step 4: Commit**

```bash
git add libs/atlas-packet/summon/clientbound/move.go libs/atlas-packet/summon/serverbound/move.go libs/atlas-packet/summon/clientbound/move_test.go libs/atlas-packet/summon/serverbound/move_test.go
git commit -m "feat(atlas-packet): SummonMove clientbound + serverbound (v83 layout)"
```

### Task 2.2: summons `Move` processor + MOVED event

**Files:**
- Modify: `services/atlas-summons/atlas.com/summons/summon/processor.go`
- Modify: `services/atlas-summons/atlas.com/summons/summon/kafka.go` (MOVED body)
- Modify: `services/atlas-summons/atlas.com/summons/summon/producer.go` (MOVED provider)
- Test: `services/atlas-summons/atlas.com/summons/summon/processor_move_test.go`

- [ ] **Step 1: Write the failing test** — `Move` by the owner updates position and
emits MOVED; `Move` by a non-owner is rejected (no change, no emit).

```go
func TestMoveByOwnerUpdatesPosition(t *testing.T) { /* spawn, Move(owner), assert new x/y persisted */ }
func TestMoveByNonOwnerRejected(t *testing.T)    { /* Move(otherCid) returns nil, position unchanged */ }
```

- [ ] **Step 2: Run — expect FAIL.**

- [ ] **Step 3: Implement `Move`**

```go
func (p *ProcessorImpl) Move(id uint32, senderCharacterId uint32, x int16, y int16, stance byte) error {
	m, err := GetRegistry().Get(p.ctx, p.t, id)
	if err != nil {
		return nil
	}
	if m.OwnerCharacterId() != senderCharacterId {
		p.l.Infof("Character [%d] moved summon [%d] it does not own; dropping.", senderCharacterId, id) // §11 ownership
		return nil
	}
	updated, err := GetRegistry().Update(p.ctx, p.t, id, func(cur Model) Model {
		return cur.Move(x, y, stance)
	})
	if err != nil {
		return err
	}
	return p.emit(EnvEventTopicSummonStatus, movedEventProvider(updated, /*rawMovement*/ nil))
}
```

Add `StatusEventMovedBody{X,Y,Stance,RawMovement []byte}` and `movedEventProvider`. The
raw movement blob is carried through from the inbound packet so the rebroadcast is
byte-faithful — thread it through the `Move` signature (add a `rawMovement []byte`
param) and the command body.

- [ ] **Step 4: Run — expect PASS.**

- [ ] **Step 5: Commit**

```bash
git add services/atlas-summons/atlas.com/summons/summon/processor.go services/atlas-summons/atlas.com/summons/summon/kafka.go services/atlas-summons/atlas.com/summons/summon/producer.go services/atlas-summons/atlas.com/summons/summon/processor_move_test.go
git commit -m "feat(atlas-summons): Move with ownership check + MOVED event"
```

### Task 2.3: summons MOVE command + channel SummonMove handler/broadcast + opcodes

**Files:**
- Modify: `services/atlas-summons/atlas.com/summons/kafka/consumer/summon/*.go` (MOVE body + handler)
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/summon_move.go`
- Modify: `services/atlas-channel/.../summon/processor.go`+`producer.go` (emit MOVE)
- Modify: `services/atlas-channel/.../kafka/consumer/summon/consumer.go` (MOVED → broadcast `SummonMove`)
- Modify: `services/atlas-channel/.../main.go` (register handler + writer)
- Modify: `template_gms_83_1.json` (SummonMove writer opcode + MoveSummon handler opcode)

- [ ] **Step 1: summons consumer** — add `MoveCommandBody{SummonId, SenderCharacterId, X, Y, Stance, RawMovement}` and `handleMoveCommand` → `Move(...)`.

- [ ] **Step 2: channel inbound handler** — `summon_move.go` mirrors `character_move.go:14-22`: decode `serverbound.Move`, then `summoncmd.NewProcessor(l,ctx).Move(s.Field(), oid, s.CharacterId(), ...)` emitting `COMMAND_TOPIC_SUMMON MOVE`. Register `handlerMap[summonsb.SummonMoveHandle] = handler.SummonMoveHandleFunc` in `produceHandlers()`.

- [ ] **Step 3: channel broadcast** — MOVED handler broadcasts `SummonMove` to **other** sessions in map (use `ForOtherSessionsInMap` excluding the owner — the owner's client already rendered the move). Register `summonpkt.SummonMoveWriter` in `produceWriters()`.

- [ ] **Step 4: opcodes** — IDA-harvest the v83 inbound `CUserLocal::OnMoveSummon` recv opcode and the outbound `SummonMove` send opcode; add a handler entry (`"handler": "SummonMoveHandle"`) and a writer entry to `template_gms_83_1.json`. Record both in the running delta notes.

- [ ] **Step 5: Build both services.**

- [ ] **Step 6: Commit**

```bash
git add services/atlas-summons services/atlas-channel services/atlas-configurations/seed-data/templates/template_gms_83_1.json
git commit -m "feat(summons): movement relay (MOVE command, SummonMove broadcast, v83 opcodes)"
```

### Task 2.4: Phase 2 verification gate

- [ ] **Step 1: Run the gate**

```bash
go vet ./services/atlas-summons/... ./services/atlas-channel/... ./libs/atlas-packet/... && \
go test -race ./services/atlas-summons/... ./libs/atlas-packet/summon/... && \
go build ./services/... ./libs/... && \
docker buildx bake atlas-summons atlas-channel && \
GOWORK=off tools/redis-key-guard.sh
```
Expected: clean.

---

## Phase 3 — Attacker behavior (`SummonAttack` + owner-credited damage + ceiling)

Goal: an attacker summon's client-driven attack damages nearby monsters with damage
credited to the owner (XP/drops/kill credit), applies stun/freeze where the roster says
so, self-cancels for Gaviota, and is validated against a server-side per-hit ceiling
(clamp + alert). All 21 attackers are client-driven (Q2).

### Task 3.1: `SummonAttack` clientbound + serverbound packets

**Files:**
- Create: `libs/atlas-packet/summon/clientbound/attack.go` (+ test)
- Create: `libs/atlas-packet/summon/serverbound/attack.go` (+ test)

- [ ] **Step 1: Write failing round-trip tests** (loop `test.Variants`). Build an attack
with two targets and assert round-trip.

- [ ] **Step 2: Implement** per Cosmic `:2308`: clientbound = int cid, int oid, byte 0,
byte direction, byte count, per target {int oid, byte 6, int dmg}. Serverbound mirrors
`SummonDamageHandler` inbound read: int oid, byte (direction/animation), byte count,
per target {int monsterOid, int dmg} (confirm exact inbound layout from
`SummonDamageHandler.java:54-145`). Carry targets as a slice.

- [ ] **Step 3: Run tests — expect PASS** all variants.

- [ ] **Step 4: Commit**

```bash
git add libs/atlas-packet/summon/clientbound/attack.go libs/atlas-packet/summon/serverbound/attack.go libs/atlas-packet/summon/clientbound/attack_test.go libs/atlas-packet/summon/serverbound/attack_test.go
git commit -m "feat(atlas-packet): SummonAttack clientbound + serverbound (v83 layout)"
```

### Task 3.2: Owner combat-stats client (`atlas-effective-stats`)

**Files:**
- Create: `services/atlas-summons/atlas.com/summons/effectivestats/rest.go`
- Create: `services/atlas-summons/atlas.com/summons/effectivestats/model.go`
- Create: `services/atlas-summons/atlas.com/summons/effectivestats/requests.go`
- Create: `services/atlas-summons/atlas.com/summons/effectivestats/processor.go`

- [ ] **Step 1: Implement the client**

Mirror `services/atlas-channel/.../effective_stats/` (rest/requests). Endpoint
`GET /worlds/{w}/channels/{c}/characters/{id}/stats`, base url env `EFFECTIVE_STATS`,
path `worlds/%d/channels/%d/characters/%d/stats`. Model getters needed:
`WeaponAttack() uint32`, `MagicAttack() uint32`, `Strength()`, `Dexterity()`,
`Intelligence()`, `Luck()` (all uint32). Processor:
`GetByCharacter(worldId world.Id, channelId channel.Id, characterId uint32) (Model, error)`.

- [ ] **Step 2: Build** `./effectivestats/...`.

- [ ] **Step 3: Commit**

```bash
git add services/atlas-summons/atlas.com/summons/effectivestats
git commit -m "feat(atlas-summons): effective-stats client for damage ceiling"
```

### Task 3.3: Damage ceiling — conservative clamp (real, logged limitation)

**Files:**
- Create: `services/atlas-summons/atlas.com/summons/summon/ceiling.go`
- Test: `services/atlas-summons/atlas.com/summons/summon/ceiling_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestConservativeCeilingClampsExcess(t *testing.T) {
	// physical summon: watk=200, effect.weaponAttack=100
	// reported damage way above the bound is clamped; in-bound damage passes through.
	max := ConservativeMaxPerHit(false /*magic*/, 200 /*watk*/, 0 /*matk*/, 100 /*effWatk*/, 0 /*effMatk*/)
	if clampDamage(uint32(max)+5000, max) != uint32(max) {
		t.Fatalf("excess not clamped")
	}
	if clampDamage(uint32(max)-1, max) != uint32(max)-1 {
		t.Fatalf("in-bound damage altered")
	}
}
```

- [ ] **Step 2: Run — expect FAIL.**

- [ ] **Step 3: Implement the conservative ceiling + clamp**

This is the **interim** bound that ships before the full weapon-type port (Task 3.6).
It is a genuine clamp (not a stub), intentionally generous to avoid false positives,
and the limitation is logged where it is used.

```go
package summon

// ConservativeMaxPerHit is an interim per-hit ceiling pending the full weapon-type
// port (see Task 3.6 / design.md §8.3). It bounds damage by the attack-multiplier
// term of Cosmic's formula using a generous base-damage proxy, so blatant client
// inflation is clamped while legitimate hits pass. The exact Cosmic formula is the
// parity target and replaces this in Task 3.6.
func ConservativeMaxPerHit(magic bool, totalWatk, totalMatk uint32, effWeaponAttack, effMagicAttack int16) int64 {
	if magic {
		matk := totalMatk
		if matk < 14 {
			matk = 14
		}
		// generous proxy for maxBaseMagicDamage: matk * matk (Cosmic squares matk-ish term)
		base := int64(matk) * int64(matk)
		return base * 5 / 100 * int64(effMagicAttack)
	}
	watk := totalWatk
	if watk < 14 {
		watk = 14
	}
	// generous proxy for maxBaseDamage: a high multiplier on watk (4x covers high-mastery)
	base := int64(watk) * 4
	mod := int64(77) // 0.077 * 1000
	if base >= 438 {
		mod = 54 // 0.054 * 1000
	}
	return base * mod / 1000 * int64(effWeaponAttack)
}

func clampDamage(reported uint32, max int64) uint32 {
	if max <= 0 {
		return reported // no ceiling computable (e.g. stats fetch failed); do not clamp to 0
	}
	if int64(reported) > max {
		return uint32(max)
	}
	return reported
}
```

> **Honest-limitation requirement:** the conservative proxy is documented here and the
> call site (Task 3.4) MUST log at info when it clamps, including reported-vs-max, so the
> interim nature is visible in ops. Task 3.6 replaces `ConservativeMaxPerHit` with the
> faithful port; the call site switches to the real function with no other change.

- [ ] **Step 4: Run — expect PASS.**

- [ ] **Step 5: Commit**

```bash
git add services/atlas-summons/atlas.com/summons/summon/ceiling.go services/atlas-summons/atlas.com/summons/summon/ceiling_test.go
git commit -m "feat(atlas-summons): conservative per-hit damage ceiling (interim, logged)"
```

### Task 3.4: summons `Attack` processor (credit + status + Gaviota + clamp/alert)

**Files:**
- Modify: `services/atlas-summons/atlas.com/summons/summon/processor.go`
- Modify: `services/atlas-summons/atlas.com/summons/summon/kafka.go` (ATTACKED body)
- Modify: `services/atlas-summons/atlas.com/summons/summon/producer.go` (ATTACKED + monster DAMAGE + APPLY_STATUS providers)
- Create: `services/atlas-summons/atlas.com/summons/monster/producer.go` (emit to COMMAND_TOPIC_MONSTER)
- Test: `services/atlas-summons/atlas.com/summons/summon/processor_attack_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestAttackCreditsOwnerAndClamps(t *testing.T) {
	// spawn attacker (Silver Hawk 3111005) owner=42; effective-stats stub watk=200.
	// Attack with one target reporting absurd damage; assert:
	//  - a COMMAND_TOPIC_MONSTER DAMAGE message is emitted with CharacterId==42
	//  - the emitted damage equals the clamped value (< reported)
	//  - an ATTACKED status event is emitted
}
func TestGaviotaSelfCancels(t *testing.T) {
	// spawn Gaviota (5211002); after one Attack, GetById -> not found (despawned)
}
func TestAttackByNonOwnerDropped(t *testing.T) {
	// Attack(senderCid != owner) -> no DAMAGE emitted, no ATTACKED event
}
```

Use a capturing fake emitter + fake effective-stats source injected into `ProcessorImpl`
(extend the test seam from Task 1.3).

- [ ] **Step 2: Run — expect FAIL.**

- [ ] **Step 3: Implement `Attack`**

```go
func (p *ProcessorImpl) Attack(id uint32, senderCharacterId uint32, direction byte, targets []AttackTarget) error {
	m, err := GetRegistry().Get(p.ctx, p.t, id)
	if err != nil {
		return nil
	}
	if m.OwnerCharacterId() != senderCharacterId {
		p.l.Infof("Character [%d] attacked with summon [%d] it does not own; dropping.", senderCharacterId, id)
		return nil
	}
	eff, err := p.effects.GetEffect(m.SkillId(), m.SkillLevel())
	if err != nil {
		return err
	}
	stats, serr := p.stats.GetByCharacter(m.Field().WorldId(), m.Field().ChannelId(), m.OwnerCharacterId())
	var max int64
	if serr != nil {
		p.l.WithError(serr).Warnf("No effective-stats for owner [%d]; summon damage not clamped this hit.", m.OwnerCharacterId())
		max = 0 // clampDamage treats 0 as "no ceiling" (do not zero legit damage)
	} else {
		magic := eff.WeaponAttack() == 0
		max = ConservativeMaxPerHit(magic, stats.WeaponAttack(), stats.MagicAttack(), eff.WeaponAttack(), eff.MagicAttack())
	}

	clampedTargets := make([]AttackTarget, 0, len(targets))
	for _, tgt := range targets {
		dmg := clampDamage(tgt.Damage, max)
		if max > 0 && int64(tgt.Damage) > max {
			p.l.Infof("Summon [%d] owner [%d] reported damage [%d] > ceiling [%d] on mob [%d]; clamped. (FR-4.3 alert)",
				id, m.OwnerCharacterId(), tgt.Damage, max, tgt.MonsterId) // §8.4 alert (warn-only; ban NOT auto-fired)
		}
		clampedTargets = append(clampedTargets, AttackTarget{MonsterId: tgt.MonsterId, Damage: dmg})
		// FR-4.2: credit the owner via monster DAMAGE.
		if err := p.emit(monsterMsg.EnvCommandTopic, monsterDamageProvider(m.Field(), tgt.MonsterId, m.OwnerCharacterId(), []uint32{dmg})); err != nil {
			p.l.WithError(err).Errorf("Unable to emit monster DAMAGE for summon [%d] target [%d].", id, tgt.MonsterId)
		}
		// FR-4.4: stun/freeze.
		if statuses := monsterStatusFor(m.SkillId(), eff); len(statuses) > 0 && rollProc(eff.Prop()) {
			_ = p.emit(monsterMsg.EnvCommandTopic, monsterApplyStatusProvider(m.Field(), tgt.MonsterId, m, eff, statuses))
		}
	}

	if err := p.emit(EnvEventTopicSummonStatus, attackedEventProvider(m, direction, clampedTargets)); err != nil {
		p.l.WithError(err).Errorf("Unable to emit ATTACKED for summon [%d].", id)
	}

	// FR-4.5: Gaviota self-cancels after one attack.
	if e, ok := summonconst.Lookup(m.SkillId()); ok && e.OneShot {
		_ = p.Despawn(id, true)
	}
	return nil
}
```

Add helpers: `monsterStatusFor(skillId, eff)` returns the `map[string]int32` of
STUN/FREEZE (driven by the roster `Stun`/`Freeze` flags plus `eff.MonsterStatus()`);
`rollProc(prop float64) bool` (Cosmic prop-based roll — for unit-test determinism, make
the proc threshold injectable or treat `prop >= 1.0` as always-proc and gate the random
path behind a function field defaulting to a real RNG). Add `p.stats` field
(effective-stats source) to `ProcessorImpl` + default wiring in `NewProcessor`.

Create `monster/producer.go` in atlas-summons with `monsterDamageProvider` and
`monsterApplyStatusProvider` emitting the **monster** command bodies (re-declare the
`COMMAND_TOPIC_MONSTER` envelope + `damageCommandBody`/`applyStatusCommandBody` shapes
locally per `context.md` §5 — services don't import each other). `damageCommandBody`
sets `CharacterId = owner`.

- [ ] **Step 4: Run — expect PASS.**

- [ ] **Step 5: Commit**

```bash
git add services/atlas-summons/atlas.com/summons/summon services/atlas-summons/atlas.com/summons/monster
git commit -m "feat(atlas-summons): Attack with owner credit, status, clamp/alert, Gaviota self-cancel"
```

### Task 3.5: ATTACK command + channel handler/broadcast + opcodes

**Files:**
- Modify: `services/atlas-summons/.../kafka/consumer/summon/*.go` (ATTACK body + handler)
- Create: `services/atlas-channel/.../socket/handler/summon_attack.go`
- Modify: `services/atlas-channel/.../summon/processor.go`+`producer.go` (emit ATTACK)
- Modify: `services/atlas-channel/.../kafka/consumer/summon/consumer.go` (ATTACKED → broadcast `SummonAttack`)
- Modify: `services/atlas-channel/.../main.go` (register handler + writer)
- Modify: `template_gms_83_1.json` (writer + handler opcodes)

- [ ] **Step 1: summons consumer** — `AttackCommandBody{SummonId, SenderCharacterId, Direction, Targets []TargetEntry}` + `handleAttackCommand` → `Attack(...)`.

- [ ] **Step 2: channel inbound handler** — `summon_attack.go` mirrors the attack-handler
shape: decode `serverbound.Attack`, emit `COMMAND_TOPIC_SUMMON ATTACK`. Register
`handlerMap[summonsb.SummonAttackHandle]`.

- [ ] **Step 3: channel broadcast** — ATTACKED → broadcast `SummonAttack` to other
sessions in map (`ForOtherSessionsInMap`). Register `summonpkt.SummonAttackWriter`.

- [ ] **Step 4: opcodes** — IDA-harvest v83 inbound summon-attack recv opcode + outbound
`SummonAttack` send opcode; add handler + writer entries to `template_gms_83_1.json`.

- [ ] **Step 5: Build both services.**

- [ ] **Step 6: Commit**

```bash
git add services/atlas-summons services/atlas-channel services/atlas-configurations/seed-data/templates/template_gms_83_1.json
git commit -m "feat(summons): attack relay (ATTACK command, SummonAttack broadcast, v83 opcodes)"
```

### Task 3.6: Faithful weapon-type damage-ceiling port

**Files:**
- Modify: `services/atlas-summons/atlas.com/summons/summon/ceiling.go`
- Modify: `services/atlas-summons/atlas.com/summons/summon/processor.go` (switch call site)
- Test: `services/atlas-summons/atlas.com/summons/summon/ceiling_test.go` (add parity cases)

- [ ] **Step 1: Read the Cosmic source** `SummonDamageHandler.calcMaxDamage:123-145`
plus `Character.calculateMaxBaseDamage`/`calculateMaxBaseMagicDamage` (the weapon-type
+ mastery-aware base-damage math). Record the exact formula, including how weapon type
maps to the stat multiplier and how mastery factors in.

- [ ] **Step 2: Determine the owner's weapon type** — extend the effective-stats client
(or add an equipment/weapon-type field) to expose the equipped weapon's type, since the
physical formula is weapon-type-aware. Confirm whether `atlas-effective-stats` already
returns weapon type; if not, add a small `atlas-character` equipment-weapon lookup
client (`GET` the equipped weapon item id → classify via `libs/atlas-constants` weapon
type). Document the chosen source.

- [ ] **Step 3: Write parity tests** with concrete Cosmic-derived expected values for at
least: a physical summon at a known watk + weapon type, and a magic summon at a known
matk. Compute the expected max by hand from the Cosmic formula and assert equality.

- [ ] **Step 4: Implement `FaithfulMaxPerHit(...)`** replacing the conservative proxy
math with the ported `maxBaseDamage`/`maxBaseMagicDamage` + the `0.05*magicAttack` /
`mod*weaponAttack` terms. Switch the Task 3.4 call site from `ConservativeMaxPerHit` to
`FaithfulMaxPerHit` and remove the interim-limitation log (keep the clamp alert log).

- [ ] **Step 5: Run all ceiling + attack tests — expect PASS.**

- [ ] **Step 6: Commit**

```bash
git add services/atlas-summons/atlas.com/summons/summon/ceiling.go services/atlas-summons/atlas.com/summons/summon/ceiling_test.go services/atlas-summons/atlas.com/summons/summon/processor.go services/atlas-summons/atlas.com/summons/effectivestats
git commit -m "feat(atlas-summons): faithful weapon-type damage-ceiling port (Cosmic parity)"
```

### Task 3.7: Phase 3 verification gate

- [ ] **Step 1: Run the gate**

```bash
go vet ./services/atlas-summons/... ./services/atlas-channel/... ./libs/atlas-packet/... && \
go test -race ./services/atlas-summons/... ./libs/atlas-packet/summon/... && \
go build ./services/... ./libs/... && \
docker buildx bake atlas-summons atlas-channel && \
GOWORK=off tools/redis-key-guard.sh
```
Expected: clean.

---

## Phase 4 — Puppet behavior (`SummonDamage` + `ADD_PUPPET`/`REMOVE_PUPPET` aggro)

Goal: puppets draw monster aggro toward the owner, take client-reported damage, and are
destroyed at 0 HP. This phase adds the one `atlas-monsters` change.

### Task 4.1: `SummonDamage` clientbound + serverbound packets

**Files:**
- Create: `libs/atlas-packet/summon/clientbound/damage.go` (+ test)
- Create: `libs/atlas-packet/summon/serverbound/damage.go` (+ test)

- [ ] **Step 1: Write failing round-trip tests** (loop variants).

- [ ] **Step 2: Implement** per Cosmic `:4076`: clientbound = int cid, int oid, byte 12,
int dmg, int monsterIdFrom, byte 0. Serverbound mirrors `DamageSummonHandler.java:35-54`
inbound: int oid, byte, int dmg, int monsterIdFrom (confirm exact layout).

- [ ] **Step 3: Run — expect PASS.**

- [ ] **Step 4: Commit**

```bash
git add libs/atlas-packet/summon/clientbound/damage.go libs/atlas-packet/summon/serverbound/damage.go libs/atlas-packet/summon/clientbound/damage_test.go libs/atlas-packet/summon/serverbound/damage_test.go
git commit -m "feat(atlas-packet): SummonDamage clientbound + serverbound (v83 layout)"
```

### Task 4.2: `atlas-monsters` — `ADD_PUPPET`/`REMOVE_PUPPET` command + puppet set + controller bias

**Files:**
- Modify: `services/atlas-monsters/.../kafka/consumer/monster/kafka.go` (new command types + bodies)
- Modify: `services/atlas-monsters/.../kafka/consumer/monster/consumer.go` (handlers)
- Create: `services/atlas-monsters/.../monster/puppet_registry.go` (per-field puppet set, Redis)
- Modify: `services/atlas-monsters/.../monster/processor.go` (controller-selection bias)
- Test: `services/atlas-monsters/.../monster/puppet_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestAddPuppetBiasesController(t *testing.T) {
	// add a puppet at (x,y) in a field with a monster within vicinity (distanceSq < 177777)
	// of the puppet owner; assert the next controller pick prefers the puppet owner.
}
func TestRemovePuppetClearsBias(t *testing.T) { /* after REMOVE_PUPPET, no bias */ }
```

- [ ] **Step 2: Run — expect FAIL.**

- [ ] **Step 3: Implement**

Add command types `CommandTypeAddPuppet="ADD_PUPPET"`, `CommandTypeRemovePuppet="REMOVE_PUPPET"`
and bodies `{WorldId,ChannelId,MapId,Instance,OwnerCharacterId,X,Y}` (add) /
`{...,OwnerCharacterId}` (remove). The handlers update a per-field puppet set
(`atlasredis` KeyedSet, namespace `"monster-puppet"`, keyed like the map index) storing
`{ownerCharacterId,x,y}`. Extend controller selection (`processor.go` `FindNextController`/
`StartControl` path) so that when picking/repicking a controller, an in-vicinity puppet
owner (Cosmic `Monster.java:1804-1942`, `distanceSq < 177777`) is preferred. Implement
the minimal vicinity-bias port; the full visibility/repick nuance is the documented long
tail (design §9 phasing note) — land the bias + signaling first, not a stub.

- [ ] **Step 4: Run — expect PASS**, then build atlas-monsters.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters
git commit -m "feat(atlas-monsters): ADD_PUPPET/REMOVE_PUPPET command + vicinity controller bias"
```

### Task 4.3: summons emits ADD_PUPPET on spawn / REMOVE_PUPPET on despawn

**Files:**
- Modify: `services/atlas-summons/atlas.com/summons/monster/producer.go` (puppet providers)
- Modify: `services/atlas-summons/atlas.com/summons/summon/processor.go` (Spawn/Despawn hooks)
- Test: `services/atlas-summons/atlas.com/summons/summon/processor_puppet_test.go`

- [ ] **Step 1: Write the failing test** — spawning a puppet emits `ADD_PUPPET`;
despawning emits `REMOVE_PUPPET`; spawning a non-puppet emits neither.

- [ ] **Step 2: Run — expect FAIL.**

- [ ] **Step 3: Implement** — in `Spawn`, after CREATED, `if m.IsPuppet() { emit ADD_PUPPET }`.
In `Despawn`, before/after Remove, `if m.IsPuppet() { emit REMOVE_PUPPET }`. Add the two
providers to `monster/producer.go`.

- [ ] **Step 4: Run — expect PASS.**

- [ ] **Step 5: Commit**

```bash
git add services/atlas-summons/atlas.com/summons
git commit -m "feat(atlas-summons): signal ADD_PUPPET/REMOVE_PUPPET on puppet spawn/despawn"
```

### Task 4.4: summons `Damage` processor (HP decrement + destroy at 0)

**Files:**
- Modify: `services/atlas-summons/atlas.com/summons/summon/processor.go`
- Modify: `services/atlas-summons/atlas.com/summons/summon/kafka.go` (DAMAGED body)
- Modify: `services/atlas-summons/atlas.com/summons/summon/producer.go` (DAMAGED provider)
- Test: `services/atlas-summons/atlas.com/summons/summon/processor_damage_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestPuppetDamageDecrementsAndDestroysAtZero(t *testing.T) {
	// spawn puppet hp=100 owner=42; Damage(owner, 30) -> hp 70, DAMAGED emitted, alive.
	// Damage(owner, 100) -> hp 0, DESTROYED emitted, REMOVE_PUPPET emitted, gone.
}
func TestDamageByNonOwnerDropped(t *testing.T) { /* ownership check */ }
```

- [ ] **Step 2: Run — expect FAIL.**

- [ ] **Step 3: Implement `Damage`**

```go
func (p *ProcessorImpl) Damage(id uint32, senderCharacterId uint32, amount int32, monsterIdFrom uint32) error {
	m, err := GetRegistry().Get(p.ctx, p.t, id)
	if err != nil {
		return nil
	}
	if m.OwnerCharacterId() != senderCharacterId {
		p.l.Infof("Character [%d] damaged summon [%d] it does not own; dropping.", senderCharacterId, id)
		return nil
	}
	updated, err := GetRegistry().Update(p.ctx, p.t, id, func(cur Model) Model { return cur.AddHP(-amount) })
	if err != nil {
		return err
	}
	if err := p.emit(EnvEventTopicSummonStatus, damagedEventProvider(updated, amount, monsterIdFrom)); err != nil {
		p.l.WithError(err).Errorf("Unable to emit DAMAGED for summon [%d].", id)
	}
	if updated.Hp() <= 0 {
		return p.Despawn(id, true) // emits DESTROYED + REMOVE_PUPPET + oid release
	}
	return nil
}
```

Add `StatusEventDamagedBody{Damage int32, MonsterIdFrom uint32}` + `damagedEventProvider`.

- [ ] **Step 4: Run — expect PASS.**

- [ ] **Step 5: Commit**

```bash
git add services/atlas-summons/atlas.com/summons/summon
git commit -m "feat(atlas-summons): puppet Damage with HP decrement and destroy-at-zero"
```

### Task 4.5: DAMAGE command + channel handler/broadcast + opcodes

**Files:**
- Modify: `services/atlas-summons/.../kafka/consumer/summon/*.go` (DAMAGE body + handler)
- Create: `services/atlas-channel/.../socket/handler/summon_damage.go`
- Modify: `services/atlas-channel/.../summon/processor.go`+`producer.go` (emit DAMAGE)
- Modify: `services/atlas-channel/.../kafka/consumer/summon/consumer.go` (DAMAGED → `SummonDamage`)
- Modify: `services/atlas-channel/.../main.go` (handler + writer)
- Modify: `template_gms_83_1.json` (writer + handler opcodes)

- [ ] **Step 1: summons consumer** — `DamageCommandBody{SummonId, SenderCharacterId, Damage, MonsterIdFrom}` + `handleDamageCommand` → `Damage(...)`.

- [ ] **Step 2: channel inbound handler** — `summon_damage.go` decodes `serverbound.Damage`, emits `COMMAND_TOPIC_SUMMON DAMAGE`. Register `handlerMap[summonsb.SummonDamageHandle]`.

- [ ] **Step 3: channel broadcast** — DAMAGED → broadcast `SummonDamage` to other sessions; DESTROYED already broadcasts `SummonRemove` (Phase 1). Register `summonpkt.SummonDamageWriter`.

- [ ] **Step 4: opcodes** — IDA-harvest v83 inbound damage-summon recv opcode + outbound `SummonDamage` send opcode; add handler + writer entries to `template_gms_83_1.json`.

- [ ] **Step 5: Build both services.**

- [ ] **Step 6: Commit**

```bash
git add services/atlas-summons services/atlas-channel services/atlas-configurations/seed-data/templates/template_gms_83_1.json
git commit -m "feat(summons): puppet damage relay (DAMAGE command, SummonDamage broadcast, v83 opcodes)"
```

### Task 4.6: Phase 4 verification gate

- [ ] **Step 1: Run the gate** (now includes atlas-monsters bake)

```bash
go vet ./services/atlas-summons/... ./services/atlas-channel/... ./services/atlas-monsters/... ./libs/atlas-packet/... && \
go test -race ./services/atlas-summons/... ./services/atlas-monsters/... ./libs/atlas-packet/summon/... && \
go build ./services/... ./libs/... && \
docker buildx bake atlas-summons atlas-channel atlas-monsters && \
GOWORK=off tools/redis-key-guard.sh
```
Expected: clean.

---

## Phase 5 — Beholder buff aura (`SummonSkill` + heal/buff timers)

Goal: while Beholder (`1321007`) is deployed, on server-side intervals it heals its
owner (`CHANGE_HP`) and applies the Beholder buff (`APPLY` with a negated source id).
Heal/buff values come from the owner's `AURA_OF_BEHOLDER` (`1320008`) and
`HEX_OF_BEHOLDER` (`1320009`) skills (Cosmic `Character.java:4448-4491`), snapshotted at
spawn. Timers are leader-elected (single-fire). `SummonSkill` renders the buff effect.

### Task 5.1: Snapshot aura/hex effects at Beholder spawn

**Files:**
- Modify: `services/atlas-summons/atlas.com/summons/summon/processor.go` (Spawn Beholder branch)
- Modify: `services/atlas-summons/atlas.com/summons/data/skill/...` (resolve owner skill levels)
- Test: `services/atlas-summons/atlas.com/summons/summon/processor_beholder_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestBeholderSpawnSnapshotsAuraAndHex(t *testing.T) {
	// stub effect provider: AURA_OF_BEHOLDER(1320008) effect hp=200, x=4 (heal every 4s);
	// HEX_OF_BEHOLDER(1320009) effect changes=[{WATK,+20}], x=4, duration=...
	// spawn Beholder owner=42; assert model.HealAmount()==200, HealInterval()==4s,
	// BuffSourceId()==-1320009, BuffChanges() has WATK+20, NextHealAt/NextBuffAt set.
}
```

- [ ] **Step 2: Run — expect FAIL.**

- [ ] **Step 3: Implement the Beholder snapshot in `Spawn`**

After building the base model, when `entry.Type == TypeBuffAura`:
- resolve the owner's trained levels in `1320008`/`1320009`. Owner skill levels come
  from a small skill-level lookup (mirror how the channel resolves a character's skill
  level — add a `characterskill` client to atlas-summons, or pass the owner's relevant
  levels in the SPAWN command if the channel already has them; **prefer threading the
  two levels in the SPAWN command** since the channel holds the caster's skill book,
  avoiding a new client. Add `AuraLevel`/`HexLevel` to `SpawnCommandBody`).
- fetch `aura := effects.GetEffect(1320008, auraLevel)` and `hex := effects.GetEffect(1320009, hexLevel)`.
- `healAmount = aura.HealHp()` (the effect `hp` field — add an `Hp() int16` getter to
  the summons effect model, reading the WZ `hp`); `healInterval = aura.X() seconds`.
- `buffChanges` from `hex` stat changes; `buffInterval = hex.X() seconds`;
  `buffDuration = hex.Duration()`; `buffLevel = hexLevel`; `buffSourceId = -int32(1320009)`.
- `nextHealAt = now + healInterval`; `nextBuffAt = now + buffInterval`.

Use the named constants `skill.DarkKnightAuraOfBeholderId`/`...HexOfBeholderId` from
`libs/atlas-constants/skill/constants.go` (verify the exact constant names; ids are
1320008/1320009).

- [ ] **Step 4: Run — expect PASS.**

- [ ] **Step 5: Commit**

```bash
git add services/atlas-summons/atlas.com/summons
git commit -m "feat(atlas-summons): snapshot Beholder aura/hex effects at spawn"
```

### Task 5.2: Beholder-aura sweep task (heal + buff)

**Files:**
- Create: `services/atlas-summons/atlas.com/summons/summon/beholder_task.go`
- Create: `services/atlas-summons/atlas.com/summons/character/producer.go` (CHANGE_HP emit)
- Create: `services/atlas-summons/atlas.com/summons/buff/producer.go` (buff APPLY emit)
- Modify: `services/atlas-summons/atlas.com/summons/main.go` (register task)
- Test: `services/atlas-summons/atlas.com/summons/summon/beholder_task_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestBeholderSweepFiresHealAndBuffWhenDue(t *testing.T) {
	// put a Beholder with NextHealAt/NextBuffAt in the past.
	// run the sweep once with a capturing emitter; assert:
	//  - a COMMAND_TOPIC_CHARACTER CHANGE_HP with Amount==healAmount, CharacterId==owner
	//  - a COMMAND_TOPIC_CHARACTER_BUFF APPLY with SourceId==-1320009, FromId==owner
	//  - the model's NextHealAt/NextBuffAt advanced by the interval (persisted)
}
func TestBeholderSweepSkipsWhenNotDue(t *testing.T) { /* future timers -> no emit */ }
```

- [ ] **Step 2: Run — expect FAIL.**

- [ ] **Step 3: Implement the sweep task**

Mirror the expiry task. `Run()` iterates `GetAllByTenant`, filters `IsBeholder()`, and
for each: if `now >= NextHealAt`, emit CHANGE_HP and advance `NextHealAt`; if
`now >= NextBuffAt`, emit buff APPLY and advance `NextBuffAt`; persist the advanced
timers via `registry.Update`. Use the cross-service bodies from `context.md` §5:
- CHANGE_HP: `CharacterCommand[ChangeHPCommandBody]{CharacterId:owner, WorldId, Type:"CHANGE_HP", Body:{ChannelId, Amount: healAmount}}` on `COMMAND_TOPIC_CHARACTER`.
- buff APPLY: `Command[ApplyCommandBody]{...,CharacterId:owner, Type:"APPLY", Body:{FromId:owner, SourceId:-1320009, Level, Duration, Changes}}` on `COMMAND_TOPIC_CHARACTER_BUFF`.

Register in `registerSweepTasks`:
`tasks.Register(l, ctx)(summon.NewBeholderTask(l, ctx, time.Second))`. Leader election
(already wired) guarantees single-fire (NFR). Timer cleanup is implicit: `Despawn`
removes the model, so a removed Beholder is never swept again (no orphan ticks).

- [ ] **Step 4: Run — expect PASS.**

- [ ] **Step 5: Commit**

```bash
git add services/atlas-summons/atlas.com/summons
git commit -m "feat(atlas-summons): leader-elected Beholder heal/buff aura sweep"
```

### Task 5.3: `SummonSkill` clientbound packet + broadcast + opcode

**Files:**
- Create: `libs/atlas-packet/summon/clientbound/skill.go` (+ test)
- Modify: `services/atlas-summons/.../summon/kafka.go`+`producer.go` (a SKILL/buff status event, or reuse ATTACKED) 
- Modify: `services/atlas-channel/.../kafka/consumer/summon/consumer.go` (broadcast `SummonSkill`)
- Modify: `services/atlas-channel/.../main.go` (writer)
- Modify: `template_gms_83_1.json` (writer opcode)

- [ ] **Step 1: Round-trip test for `SummonSkill`** (loop variants).

- [ ] **Step 2: Implement** per Cosmic `:4569`: int cid, int summonSkillId, byte newStance.

- [ ] **Step 3: Emit + broadcast** — when the Beholder buff fires, the summons service
emits a status event carrying the new stance; channel broadcasts `SummonSkill`. Add a
`StatusEventSkillBody{NewStance byte}` + provider, or extend the buff sweep to emit it.
Register `summonpkt.SummonSkillWriter`; add its v83 writer opcode (IDA-harvested) to the
template.

- [ ] **Step 4: Build both services.**

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-packet/summon/clientbound/skill.go services/atlas-summons services/atlas-channel services/atlas-configurations/seed-data/templates/template_gms_83_1.json
git commit -m "feat(summons): SummonSkill packet for Beholder buff effect (v83 opcode)"
```

### Task 5.4: Phase 5 verification gate

- [ ] **Step 1: Run the gate**

```bash
go vet ./services/atlas-summons/... ./services/atlas-channel/... ./libs/atlas-packet/... && \
go test -race ./services/atlas-summons/... ./libs/atlas-packet/summon/... && \
go build ./services/... ./libs/... && \
docker buildx bake atlas-summons atlas-channel && \
GOWORK=off tools/redis-key-guard.sh
```
Expected: clean. At this point the full v83 feature is functional end-to-end.

---

## Phase 6 — Multi-version protocol, opcodes across all templates, per-variant tests

Goal: the six packets + three decoders encode/decode byte-correctly for GMS
v12/83/84/87/92/95 and JMS v185; opcodes are seeded in all seven templates; deltas are
documented. This phase is **research-driven** — per-version byte layouts and opcode
bytes are client-fixed and harvested from IDA, never invented.

### Task 6.1: IDA delta + opcode harvest → `summon-packet-delta.md`

**Files:**
- Create: `docs/tasks/task-088-player-summons/summon-packet-delta.md`

- [ ] **Step 1: Harvest per-version layouts (one IDB at a time)**

Following `reference_ida_harvest_subagents` (one IDB loaded at a time; the user switches
instances), and the task-083 precedent (`docs/tasks/task-083-gms-v84-tenant-support/v84-packet-delta.md`),
for each available IDB (v83, v84, v87, v95, JMS185) harvest:
- the **summon spawn/remove/move/attack/damage/skill packet layouts** (the v83-marker
  byte value per version — the `0x0A` in spawn — plus any field-width or
  field-presence deltas), and
- the **send opcodes** for the six writers and the **recv opcodes** for the three
  handlers.

Cross-reference `bug_majorversion_gt83_is_off_by_one_v87`: v84 (and v86) are
byte-identical to v83; gate any new structure on `>=87`. For v12 and v92 (no IDB),
derive from the nearest bracketing versions + existing template opcode patterns and mark
those rows **"derived, unverified — confirm against capture"** explicitly (no silent
guessing).

- [ ] **Step 2: Write `summon-packet-delta.md`** — one section per packet, a table of
{version → byte layout + opcode}, mirroring `v84-packet-delta.md`'s structure. Include
the running v83 opcodes recorded during Phases 1–5.

- [ ] **Step 3: Commit**

```bash
git add docs/tasks/task-088-player-summons/summon-packet-delta.md
git commit -m "docs(task-088): summon packet per-version delta + opcode harvest"
```

### Task 6.2: Version-conditional encode/decode for all six packets + three decoders

**Files:**
- Modify: `libs/atlas-packet/summon/clientbound/*.go`
- Modify: `libs/atlas-packet/summon/serverbound/*.go`
- Modify: their `_test.go` files (assert real per-version bytes, not just round-trip)

- [ ] **Step 1: For each packet, add the version branches** documented in
`summon-packet-delta.md`, using the idiom (`t.IsRegion`, `t.MajorAtLeast(87)`,
`t.MajorAtMost`, `t.MajorInRange`). Replace the Phase-1 `0x0A`-unconditional marker with
the per-version value. **Gate on `>=87`, never `>83`** (v84/v86 = v83).

- [ ] **Step 2: Strengthen tests** — beyond round-trip, add at least one explicit
byte-level assertion per packet for a representative pre-87 and post-87 variant (encode
→ compare to the expected bytes from the delta doc). This catches a wrong branch that a
round-trip alone would miss (per `reference_packet_audit_tool_mechanics`: round-trip is
necessary but not sufficient for mode/version-driven packets).

- [ ] **Step 3: Run the full packet suite — expect PASS** for every variant.

```bash
cd libs/atlas-packet && go test ./summon/...
```

- [ ] **Step 4: Commit**

```bash
git add libs/atlas-packet/summon
git commit -m "feat(atlas-packet): version-conditional summon encode/decode (all supported versions)"
```

### Task 6.3: Seed summon opcodes into the remaining six templates

**Files:**
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_12_1.json`
- Modify: `template_gms_84_1.json`, `template_gms_87_1.json`, `template_gms_92_1.json`, `template_gms_95_1.json`, `template_jms_185_1.json`

- [ ] **Step 1: Add the six writer + three handler opcode entries** to each remaining
template, using the per-version opcode bytes from `summon-packet-delta.md` (v83 was done
incrementally in Phases 1–5). Each template gets: `SummonSpawn`, `SummonRemove`,
`SummonMove`, `SummonAttack`, `SummonDamage`, `SummonSkill` writers + `SummonMoveHandle`,
`SummonAttackHandle`, `SummonDamageHandle` handlers.

- [ ] **Step 2: Validate every template parses** (loop `python3 -m json.tool`).

- [ ] **Step 3: Confirm writer/handler names match** the string consts registered in
`atlas-channel`'s `produceWriters`/`produceHandlers` and the packet package — a
name mismatch silently drops the packet.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-configurations/seed-data/templates/template_gms_12_1.json services/atlas-configurations/seed-data/templates/template_gms_84_1.json services/atlas-configurations/seed-data/templates/template_gms_87_1.json services/atlas-configurations/seed-data/templates/template_gms_92_1.json services/atlas-configurations/seed-data/templates/template_gms_95_1.json services/atlas-configurations/seed-data/templates/template_jms_185_1.json
git commit -m "feat(atlas-configurations): seed summon opcodes for all supported versions"
```

### Task 6.4: Graceful no-op regression test for later-version summons

**Files:**
- Test: `services/atlas-summons/atlas.com/summons/summon/processor_spawn_test.go` (add case)

- [ ] **Step 1: Add the test** — `Spawn` with a Dual-Blade/Evan summon skill id (a real
later-version id NOT in the roster) returns `(Model{}, nil)`, persists nothing, and logs
at debug — no error, no panic (FR-1.3 / Q5). Reuse `TestSpawnUnknownSkillNoOp`'s shape
with a concrete out-of-scope id.

- [ ] **Step 2: Run — expect PASS.**

- [ ] **Step 3: Commit**

```bash
git add services/atlas-summons/atlas.com/summons/summon/processor_spawn_test.go
git commit -m "test(atlas-summons): graceful no-op for later-version-only summon ids"
```

### Task 6.5: Final full verification gate

- [ ] **Step 1: Run the complete gate (worktree root)**

```bash
go vet ./... && \
go test -race ./services/atlas-summons/... ./services/atlas-monsters/... ./services/atlas-channel/... ./libs/atlas-packet/... ./libs/atlas-constants/... && \
go build ./... && \
docker buildx bake atlas-summons atlas-channel atlas-monsters atlas-configurations && \
GOWORK=off tools/redis-key-guard.sh
```
Expected: all clean.

- [ ] **Step 2: Code review** — before opening a PR, run
`superpowers:requesting-code-review` (dispatches `plan-adherence-reviewer` +
`backend-guidelines-reviewer`; no TS changed so no frontend reviewer). Address findings,
then finish the branch.

---

## Acceptance criteria → task map (self-review)

| PRD acceptance criterion | Implementing task(s) |
|---|---|
| Service exists/builds/registered; `bake atlas-summons` succeeds | 0.1, 0.9 |
| All 21 summons spawn correct type/movement/HP/duration | 1.1, 1.3, 1.7, 1.9 |
| Re-cast replaces; conflicting-class cancels | 1.3 (`conflictsMobility`, re-cast loop) |
| Puppets draw aggro, take damage, destroyed at 0 HP | 4.2, 4.3, 4.4, 4.5 |
| Attackers credit owner; XP/drops/kill; stun/freeze; Gaviota self-cancel | 3.4, 3.5 |
| Client damage > server max clamped + alert | 3.3, 3.4, 3.6 |
| Beholder heals + buffs; timers stop on removal | 5.1, 5.2 |
| Despawn on logout/channel/map/expiry; oids released every path | 1.5, 1.6, 1.3 (Despawn funnel) |
| Six packets byte-correct all versions; deltas documented | 1.7, 2.1, 3.1, 4.1, 5.3, 6.1, 6.2 |
| Opcodes seeded all versions; resolved per-tenant | 1.10, 2.3, 3.5, 4.5, 5.3, 6.3 |
| `go test -race`/`vet`/`build` clean; redis-guard clean | every phase gate |
| Later-version-only summon = graceful no-op | 1.3, 6.4 |

---

## Notes for the executor

- **No silent stubs:** the Phase-0 processor method bodies are placeholders ONLY for the
  scaffold-builds-green gate; every one is replaced with a tested implementation in
  Phases 1–5 (Spawn/Despawn 1.3, Move 2.2, Attack 3.4, Damage 4.4). Do not ship Phase 0
  as a standalone PR.
- **Conservative→faithful ceiling:** Task 3.3 ships a real (logged-as-interim) clamp;
  Task 3.6 replaces it with the Cosmic-parity port in the **same phase**. Both land
  before Phase 3's gate — there is no released state with a silent approximation.
- **Cross-service structs are re-declared, not imported:** each service declares its own
  copy of the command/event envelopes it emits/consumes (per the project's service-
  boundary rule). `context.md` §5 is the authoritative field list — keep JSON tags
  identical across producer and consumer or messages silently fail to decode.
- **IDA-harvested values are mandatory, not optional:** opcode bytes and per-version
  layout markers come from the binaries. Where no IDB exists (v12, v92), the value is
  marked "derived, unverified" in `summon-packet-delta.md` and flagged for capture
  confirmation — never silently guessed.
- **Live-tenant note:** seeding opcodes in templates only affects newly-created tenants;
  existing tenants need a config patch + channel restart
  (`bug_new_opcodes_not_in_live_tenant_config`). That is an operational follow-up, not a
  code task in this plan.

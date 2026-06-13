# Mystic Door (Priest) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship the Priest Mystic Door skill (`2311002`) as a new version-agnostic `atlas-doors` engine service plus an `atlas-channel` packet edge, so a Priest can cast a party-shared two-map town portal that works on every supported tenant version.

**Architecture:** `atlas-doors` (new, mirrored on in-tree `atlas-monsters`) owns door state in Redis: paired area+town door records, a shared object-id allocator, a leader-elected expiry sweep, per-party town-slot allocation, and Kafka command/event topics. `atlas-channel` routes the cast through the existing per-skill `Lookup` seam, decodes the enter-door packet, warps via the existing `portal.Warp` path, and broadcasts spawn/remove/party-minimap packets to party-scoped, same-channel viewers. All version variance lives in a new `libs/atlas-packet/door` package plus per-tenant socket-template opcodes (opcodes are config, not Go).

**Tech Stack:** Go (DDD immutable models + Builder, functional composition), `libs/atlas-redis`/`atlas-object-id`/`atlas-lock`/`atlas-kafka`/`atlas-tenant`/`atlas-constants`/`atlas-packet`, JSON:API (api2go), Kafka, Redis, multi-tenant context propagation, Docker buildx bake, k8s/kustomize.

> **Read `context.md` first.** It lists 12 load-bearing corrections to `design.md`
> discovered during planning (chief among them: **mirror `atlas-monsters`, not the
> non-present `atlas-summons`**; **opcodes are tenant config, not Go**; **`PlayPortalSound`
> already exists — don't add a packet**; **per-version door bytes are IDA-verify-or-escalate**).

---

## Conventions used in this plan

- All paths are relative to the worktree root `<repo-root>/.worktrees/task-093-mystic-door/`.
- New service module: `services/atlas-doors/atlas.com/doors/` (go.mod module name
  **`atlas-doors`**, short form).
- "Mirror `monsters/<file>`" means: open the named `atlas-monsters` file, copy its
  structure, and apply the substitutions called out in the step. Monster code is the
  in-tree source of truth for the registry/allocator/leader/kafka boilerplate; do not
  invent boilerplate that diverges from it.
- Each TDD task: write failing test → run (FAIL) → implement → run (PASS) → commit.
- Commit messages use the `feat(atlas-doors): …` / `feat(atlas-channel): …` /
  `feat(atlas-packet): …` form. Commit on the `task-093-mystic-door` branch only.
- After every commit, the executor verifies `git branch --show-current` is
  `task-093-mystic-door` and `git rev-parse --show-toplevel` ends with
  `/.worktrees/task-093-mystic-door`.

---

# PART A — Service scaffold & registration

Goal: a buildable, registered, empty `atlas-doors` service that boots, elects a leader,
serves `/api/`, and is wired into go.work / services.json / docker-bake / k8s. No door
logic yet.

### Task A1: Create the Go module

**Files:**
- Create: `services/atlas-doors/atlas.com/doors/go.mod`
- Create: `services/atlas-doors/atlas.com/doors/logger/init.go`
- Modify: `go.work`

- [ ] **Step 1: Create the module file** by copying the monsters go.mod header.

Run: `cat services/atlas-monsters/atlas.com/monsters/go.mod | head -40` and create
`services/atlas-doors/atlas.com/doors/go.mod` with module name `atlas-doors`, the same
`go` version line, and the same `require`/`replace` blocks (the workspace `replace`s for
`libs/atlas-*` are inherited from `go.work`, but copy any explicit `require`s monsters
lists for: `atlas-redis`, `atlas-object-id`, `atlas-lock`, `atlas-kafka`, `atlas-tenant`,
`atlas-constants`, `atlas-rest`, `atlas-model`, `atlas-service`, `atlas-tracing`,
`github.com/sirupsen/logrus`, `github.com/google/uuid`, `github.com/gorilla/mux`,
`github.com/redis/go-redis/v9`, `github.com/segmentio/kafka-go`, api2go/jsonapi).

- [ ] **Step 2: Add the module to the workspace.**

Edit `go.work` — add this line in the services block (after `./services/atlas-data/...`):

```
	./services/atlas-doors/atlas.com/doors
```

- [ ] **Step 3: Create the logger** by mirroring `monsters/logger/init.go` verbatim
(package `logger`, logrus + ECS hook, `func New(serviceName string) *logrus.Logger`).

- [ ] **Step 4: Verify the module resolves.**

Run: `cd services/atlas-doors/atlas.com/doors && GOFLAGS=-mod=mod go mod tidy && go build ./... ; cd -`
Expected: builds (only the logger package exists; no errors).

- [ ] **Step 5: Commit.**

```bash
git add go.work services/atlas-doors/atlas.com/doors/go.mod services/atlas-doors/atlas.com/doors/go.sum services/atlas-doors/atlas.com/doors/logger
git commit -m "feat(atlas-doors): scaffold module, workspace entry, logger"
```

### Task A2: Generic task runner + leader config

**Files:**
- Create: `services/atlas-doors/atlas.com/doors/tasks/task.go`
- Create: `services/atlas-doors/atlas.com/doors/leaderconfig.go`
- Create: `services/atlas-doors/atlas.com/doors/leaderconfig_test.go`

- [ ] **Step 1: Mirror `monsters/tasks/task.go` verbatim** — the `Task` interface and
`Register` goroutine loop:

```go
package tasks

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
)

type Task interface {
	Run()
	SleepTime() time.Duration
}

func Register(l logrus.FieldLogger, ctx context.Context) func(t Task) {
	return func(t Task) {
		go func(t Task) {
			for {
				select {
				case <-ctx.Done():
					l.Infof("Stopping task execution.")
					return
				case <-time.After(t.SleepTime()):
					t.Run()
				}
			}
		}(t)
	}
}
```

- [ ] **Step 2: Write the failing leaderconfig test.** Mirror
`monsters/leaderconfig_test.go`, renaming env vars to the `DOOR_LEADER_*` prefix.

```go
package main

import (
	"os"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestLeaderTTLDefault(t *testing.T) {
	os.Unsetenv("DOOR_LEADER_TTL")
	if got := leaderTTL(logrus.New()); got != defaultLeaderTTL {
		t.Fatalf("expected default %v, got %v", defaultLeaderTTL, got)
	}
}

func TestLeaderTTLClampLow(t *testing.T) {
	os.Setenv("DOOR_LEADER_TTL", "1s")
	defer os.Unsetenv("DOOR_LEADER_TTL")
	if got := leaderTTL(logrus.New()); got < 5*time.Second {
		t.Fatalf("expected clamp to >=5s, got %v", got)
	}
}
```

- [ ] **Step 3: Run the test to verify it fails.**

Run: `cd services/atlas-doors/atlas.com/doors && go test ./... ; cd -`
Expected: FAIL (`leaderTTL`/`defaultLeaderTTL` undefined).

- [ ] **Step 4: Create `leaderconfig.go`** by mirroring `monsters/leaderconfig.go`,
substituting the env prefix `MONSTER_`→`DOOR_`:

```go
package main

import (
	"os"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	envLeaderEnabled = "DOOR_LEADER_ELECTION_ENABLED"
	envLeaderTTL     = "DOOR_LEADER_TTL"
	envLeaderRefresh = "DOOR_LEADER_REFRESH"
	envLeaderBackoff = "DOOR_LEADER_BACKOFF"

	defaultLeaderTTL     = 30 * time.Second
	defaultLeaderRefresh = 10 * time.Second
	defaultLeaderBackoff = 5 * time.Second
)

// leaderEnabled, leaderTTL, leaderRefresh, leaderBackoff, parseDurationInRange:
// copy the bodies from monsters/leaderconfig.go unchanged (only the const names above
// differ). leaderEnabled defaults true; leaderTTL clamps to [5s,5m]; leaderRefresh
// defaults ttl/3 clamped to [1s, ttl/2]; leaderBackoff clamps to [1s,1m].
```

(Reproduce the function bodies exactly from `monsters/leaderconfig.go`.)

- [ ] **Step 5: Run the test to verify it passes.**

Run: `cd services/atlas-doors/atlas.com/doors && go test ./... ; cd -`
Expected: PASS.

- [ ] **Step 6: Commit.**

```bash
git add services/atlas-doors/atlas.com/doors/tasks services/atlas-doors/atlas.com/doors/leaderconfig.go services/atlas-doors/atlas.com/doors/leaderconfig_test.go
git commit -m "feat(atlas-doors): task runner + leader election config"
```

### Task A3: Shared Kafka consumer/producer plumbing

**Files:**
- Create: `services/atlas-doors/atlas.com/doors/kafka/consumer/consumer.go`
- Create: `services/atlas-doors/atlas.com/doors/kafka/producer/producer.go`
- Create: `services/atlas-doors/atlas.com/doors/rest/handler.go`

- [ ] **Step 1: Mirror `monsters/kafka/consumer/consumer.go` verbatim** — the `NewConfig`
factory and `LookupBrokers`:

```go
package consumer

// NewConfig(l)(name)(token)(groupId) consumer.Config + LookupBrokers() []string
// reading BOOTSTRAP_SERVERS. Copy from monsters unchanged.
```

- [ ] **Step 2: Mirror `monsters/kafka/producer/producer.go` verbatim** — the
`ProviderImpl(l)(ctx)(token)` wrapper around `libs/atlas-kafka/producer`.

- [ ] **Step 3: Mirror `monsters/rest/handler.go` verbatim** — the `server.*` type
aliases (`HandlerDependency`, `HandlerContext`, `RegisterHandler`,
`RegisterInputHandler`) and the typed path parsers. Add a `ParseDoorId` parser
(`server.ParseIntId[uint32]`) and keep `ParseWorldId/ParseChannelId/ParseMapId/
ParseInstanceId`.

- [ ] **Step 4: Verify it builds.**

Run: `cd services/atlas-doors/atlas.com/doors && go build ./... ; cd -`
Expected: builds.

- [ ] **Step 5: Commit.**

```bash
git add services/atlas-doors/atlas.com/doors/kafka services/atlas-doors/atlas.com/doors/rest
git commit -m "feat(atlas-doors): kafka consumer/producer + rest handler plumbing"
```

### Task A4: Register the service (build system)

**Files:**
- Modify: `.github/config/services.json`
- Modify: `docker-bake.hcl`

- [ ] **Step 1: Add the services.json entry.** Insert this object into the `.services[]`
array (after the `atlas-data` entry, mirroring `atlas-monsters`):

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

- [ ] **Step 2: Add to docker-bake.hcl `go_services`.** Insert `"atlas-doors",` into the
hardcoded list, alphabetically between `"atlas-data",` and the next entry:

```hcl
  "atlas-data",
  "atlas-doors",
```

- [ ] **Step 3: Verify bake config parses.**

Run: `docker buildx bake atlas-doors --print 2>&1 | head -30`
Expected: prints a valid target for `atlas-doors` (no HCL error). (It will fail to build
until `main.go` exists — that's fine; we only check the target resolves here.)

- [ ] **Step 4: Commit.**

```bash
git add .github/config/services.json docker-bake.hcl
git commit -m "feat(atlas-doors): register service in services.json + docker-bake"
```

### Task A5: k8s manifest + env config

**Files:**
- Create: `deploy/k8s/base/atlas-doors.yaml`
- Modify: `deploy/k8s/base/kustomization.yaml`
- Modify: `deploy/k8s/base/env-configmap.yaml`

- [ ] **Step 1: Create `deploy/k8s/base/atlas-doors.yaml`** by mirroring
`deploy/k8s/base/atlas-monsters.yaml` (Deployment + Service, `containerPort: 8080`,
`envFrom` the `atlas-env` ConfigMap, `LOG_LEVEL` env, Service port 8080). Substitute
`atlas-monsters`→`atlas-doors` and the image to
`ghcr.io/chronicle20/atlas-doors/atlas-doors`. **If** you add a readiness probe, the path
MUST be `/api/readyz` (the REST server base path is `/api/`).

- [ ] **Step 2: Add to kustomization.** In `deploy/k8s/base/kustomization.yaml`, add a
resource line in alpha order:

```yaml
  - atlas-doors.yaml
```

- [ ] **Step 3: Add the door topics to env-configmap.** In
`deploy/k8s/base/env-configmap.yaml`, under the existing `COMMAND_TOPIC_*` /
`EVENT_TOPIC_*` block, add:

```yaml
  COMMAND_TOPIC_DOOR: "command-topic-door"
  EVENT_TOPIC_DOOR_STATUS: "event-topic-door-status"
```

(Match the existing naming convention of neighbouring topic values.)

- [ ] **Step 4: Verify kustomize builds.**

Run: `kubectl kustomize deploy/k8s/base >/dev/null && echo OK`
Expected: `OK` (no kustomize error).

- [ ] **Step 5: Commit.**

```bash
git add deploy/k8s/base/atlas-doors.yaml deploy/k8s/base/kustomization.yaml deploy/k8s/base/env-configmap.yaml
git commit -m "feat(atlas-doors): k8s base manifest + door kafka topics"
```

---

# PART B — Domain model, registry, id allocator

Goal: the immutable `door.Model` (a pair record), its Builder, the Redis registry with
field/owner/town-party indices, and the object-id allocator.

### Task B1: `door.Model` + Builder

**Files:**
- Create: `services/atlas-doors/atlas.com/doors/door/model.go`
- Create: `services/atlas-doors/atlas.com/doors/door/builder.go`
- Create: `services/atlas-doors/atlas.com/doors/door/model_test.go`

- [ ] **Step 1: Write the failing model test.**

```go
package door

import (
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

func TestBuilderAndGettersAndReslotImmutable(t *testing.T) {
	f := field.NewBuilder(1, 2, 100000000).Build()
	deploy := time.Unix(1000, 0)
	m := NewBuilder().
		SetAreaDoorId(1_000_001).
		SetTownDoorId(1_000_002).
		SetOwnerCharacterId(42).
		SetPartyId(7).
		SetSkillId(2311002).
		SetSkillLevel(10).
		SetField(f).
		SetTownMapId(104000000).
		SetSlot(0).
		SetTownPortalId(0x80).
		SetAreaX(50).SetAreaY(60).
		SetTownX(-12).SetTownY(34).
		SetDeployTime(deploy).
		SetExpiresAt(deploy.Add(2 * time.Minute)).
		Build()

	if m.AreaDoorId() != 1_000_001 || m.TownDoorId() != 1_000_002 {
		t.Fatalf("door ids wrong: %d/%d", m.AreaDoorId(), m.TownDoorId())
	}
	if m.PairId() != m.AreaDoorId() {
		t.Fatalf("pairId must equal areaDoorId, got %d", m.PairId())
	}
	if m.Field().MapId() != 100000000 {
		t.Fatalf("field map wrong: %d", m.Field().MapId())
	}

	// Reslot returns a NEW model; original unchanged.
	n := m.Reslot(3, 0x83, -99, 88)
	if m.Slot() != 0 || m.TownPortalId() != 0x80 || m.TownX() != -12 {
		t.Fatalf("original mutated by Reslot")
	}
	if n.Slot() != 3 || n.TownPortalId() != 0x83 || n.TownX() != -99 || n.TownY() != 88 {
		t.Fatalf("reslot did not apply: slot=%d portal=%d x=%d y=%d", n.Slot(), n.TownPortalId(), n.TownX(), n.TownY())
	}
	// Reslot preserves identity fields.
	if n.AreaDoorId() != m.AreaDoorId() || n.OwnerCharacterId() != m.OwnerCharacterId() {
		t.Fatalf("reslot changed identity fields")
	}
}
```

- [ ] **Step 2: Run to verify it fails.**

Run: `cd services/atlas-doors/atlas.com/doors && go test ./door/ ; cd -`
Expected: FAIL (`NewBuilder` undefined).

- [ ] **Step 3: Implement `model.go`** (private fields + value-receiver getters; derived
`PairId()` returns `areaDoorId`; `Reslot` returns a Clone with new town slot fields):

```go
package door

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

type Model struct {
	areaDoorId       uint32
	townDoorId       uint32
	ownerCharacterId uint32
	partyId          uint32
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

func (m Model) Reslot(slot byte, townPortalId uint32, townX int16, townY int16) Model {
	return Clone(m).SetSlot(slot).SetTownPortalId(townPortalId).SetTownX(townX).SetTownY(townY).Build()
}
```

- [ ] **Step 4: Implement `builder.go`** (pointer-receiver fluent setters, `NewBuilder()`,
`Clone(m)`):

```go
package door

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

type ModelBuilder struct {
	areaDoorId       uint32
	townDoorId       uint32
	ownerCharacterId uint32
	partyId          uint32
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

func NewBuilder() *ModelBuilder { return &ModelBuilder{} }

func Clone(m Model) *ModelBuilder {
	return &ModelBuilder{
		areaDoorId: m.areaDoorId, townDoorId: m.townDoorId, ownerCharacterId: m.ownerCharacterId,
		partyId: m.partyId, skillId: m.skillId, skillLevel: m.skillLevel, fld: m.fld,
		townMapId: m.townMapId, slot: m.slot, townPortalId: m.townPortalId,
		areaX: m.areaX, areaY: m.areaY, townX: m.townX, townY: m.townY,
		deployTime: m.deployTime, expiresAt: m.expiresAt,
	}
}

func (b *ModelBuilder) SetAreaDoorId(v uint32) *ModelBuilder       { b.areaDoorId = v; return b }
func (b *ModelBuilder) SetTownDoorId(v uint32) *ModelBuilder       { b.townDoorId = v; return b }
func (b *ModelBuilder) SetOwnerCharacterId(v uint32) *ModelBuilder { b.ownerCharacterId = v; return b }
func (b *ModelBuilder) SetPartyId(v uint32) *ModelBuilder          { b.partyId = v; return b }
func (b *ModelBuilder) SetSkillId(v uint32) *ModelBuilder          { b.skillId = v; return b }
func (b *ModelBuilder) SetSkillLevel(v byte) *ModelBuilder         { b.skillLevel = v; return b }
func (b *ModelBuilder) SetField(v field.Model) *ModelBuilder       { b.fld = v; return b }
func (b *ModelBuilder) SetTownMapId(v _map.Id) *ModelBuilder       { b.townMapId = v; return b }
func (b *ModelBuilder) SetSlot(v byte) *ModelBuilder               { b.slot = v; return b }
func (b *ModelBuilder) SetTownPortalId(v uint32) *ModelBuilder     { b.townPortalId = v; return b }
func (b *ModelBuilder) SetAreaX(v int16) *ModelBuilder             { b.areaX = v; return b }
func (b *ModelBuilder) SetAreaY(v int16) *ModelBuilder             { b.areaY = v; return b }
func (b *ModelBuilder) SetTownX(v int16) *ModelBuilder             { b.townX = v; return b }
func (b *ModelBuilder) SetTownY(v int16) *ModelBuilder             { b.townY = v; return b }
func (b *ModelBuilder) SetDeployTime(v time.Time) *ModelBuilder    { b.deployTime = v; return b }
func (b *ModelBuilder) SetExpiresAt(v time.Time) *ModelBuilder     { b.expiresAt = v; return b }

func (b *ModelBuilder) Build() Model {
	return Model{
		areaDoorId: b.areaDoorId, townDoorId: b.townDoorId, ownerCharacterId: b.ownerCharacterId,
		partyId: b.partyId, skillId: b.skillId, skillLevel: b.skillLevel, fld: b.fld,
		townMapId: b.townMapId, slot: b.slot, townPortalId: b.townPortalId,
		areaX: b.areaX, areaY: b.areaY, townX: b.townX, townY: b.townY,
		deployTime: b.deployTime, expiresAt: b.expiresAt,
	}
}
```

- [ ] **Step 5: Run to verify it passes.**

Run: `cd services/atlas-doors/atlas.com/doors && go test ./door/ ; cd -`
Expected: PASS.

- [ ] **Step 6: Commit.**

```bash
git add services/atlas-doors/atlas.com/doors/door/model.go services/atlas-doors/atlas.com/doors/door/builder.go services/atlas-doors/atlas.com/doors/door/model_test.go
git commit -m "feat(atlas-doors): immutable door pair model + builder"
```

### Task B2: Object-id allocator wrapper

**Files:**
- Create: `services/atlas-doors/atlas.com/doors/door/id_allocator.go`

- [ ] **Step 1: Mirror `monsters/monster/id_allocator.go`** but **without** the silent
`MinId` fallback — door allocation must surface errors so the spawn can fail cleanly
(design §4.4):

```go
package door

import (
	"context"
	"sync"

	objectid "github.com/Chronicle20/atlas/libs/atlas-object-id"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	goredis "github.com/redis/go-redis/v9"
)

type IdAllocator struct{ inner objectid.Allocator }

var idAllocator *IdAllocator
var idAllocatorOnce sync.Once

func InitIdAllocator(rc *goredis.Client) {
	idAllocatorOnce.Do(func() { idAllocator = &IdAllocator{inner: objectid.NewRedisAllocator(rc)} })
}

func GetIdAllocator() *IdAllocator { return idAllocator }

// Allocate returns (id, nil) or (0, err). Callers MUST fail the spawn on error and
// release any prior allocation — never substitute MinId (collision bug, TODO.md).
func (a *IdAllocator) Allocate(ctx context.Context, t tenant.Model) (uint32, error) {
	return a.inner.Allocate(ctx, t)
}

func (a *IdAllocator) Release(ctx context.Context, t tenant.Model, id uint32) {
	_ = a.inner.Release(ctx, t, id)
}
```

(Confirm the exact `objectid.Allocator` interface + `NewRedisAllocator` signature against
`libs/atlas-object-id`; adapt import path/method names to match.)

- [ ] **Step 2: Verify it builds.**

Run: `cd services/atlas-doors/atlas.com/doors && go build ./door/ ; cd -`
Expected: builds.

- [ ] **Step 3: Commit.**

```bash
git add services/atlas-doors/atlas.com/doors/door/id_allocator.go
git commit -m "feat(atlas-doors): object-id allocator wrapper (fail-on-error)"
```

### Task B3: Redis registry + indices

**Files:**
- Create: `services/atlas-doors/atlas.com/doors/door/registry.go`
- Create: `services/atlas-doors/atlas.com/doors/door/registry_test.go`

The registry mirrors `monsters/monster/registry.go`: a primary
`atlasredis.Registry[string, storedDoor]` plus secondary `atlasredis.KeyedSet[string]`
indices. **Doors need three indices**: by field (area-door spawn + field broadcast), by
owner (recast/cleanup), and by town+party (slot allocation + town broadcast). Tenant id
goes in the key suffix.

- [ ] **Step 1: Write the failing registry test.** Use whatever in-memory redis harness
`monsters/monster/registry_test.go` uses. Assert: Put then Get round-trips all fields;
GetInField returns the door; GetByOwner returns it; two solo casters at the same town do
NOT collide in the town-party index; Remove clears all three indices.

```go
package door

import (
	"context"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	// reuse whatever in-memory redis the monsters registry test uses
)

func newTestRegistry(t *testing.T) (*Registry, context.Context, tenant.Model) {
	// mirror monsters/monster/registry_test.go: spin an in-memory redis client,
	// build a tenant, return newRegistry(rc), tenant.WithContext(...), tenant.
	panic("fill from monsters registry test harness")
}

func TestRegistryRoundTripAndIndices(t *testing.T) {
	r, ctx, ten := newTestRegistry(t)
	f := field.NewBuilder(1, 2, 100000000).Build()
	m := NewBuilder().SetAreaDoorId(1_000_001).SetTownDoorId(1_000_002).
		SetOwnerCharacterId(42).SetPartyId(0).SetField(f).
		SetTownMapId(104000000).SetSlot(0).SetTownPortalId(0x80).
		SetDeployTime(time.Unix(1000, 0)).SetExpiresAt(time.Unix(1120, 0)).Build()

	if err := r.Put(ctx, ten, m); err != nil { t.Fatal(err) }

	got, err := r.Get(ctx, ten, 1_000_001)
	if err != nil || got.OwnerCharacterId() != 42 || got.TownPortalId() != 0x80 {
		t.Fatalf("round-trip failed: %+v err=%v", got, err)
	}
	inField, _ := r.GetInField(ctx, ten, f)
	if len(inField) != 1 { t.Fatalf("field index: want 1 got %d", len(inField)) }
	byOwner, _ := r.GetByOwner(ctx, ten, 42)
	if len(byOwner) != 1 { t.Fatalf("owner index: want 1 got %d", len(byOwner)) }

	if err := r.Remove(ctx, ten, 1_000_001); err != nil { t.Fatal(err) }
	inField, _ = r.GetInField(ctx, ten, f)
	byOwner, _ = r.GetByOwner(ctx, ten, 42)
	if len(inField) != 0 || len(byOwner) != 0 {
		t.Fatalf("indices not cleared on remove: field=%d owner=%d", len(inField), len(byOwner))
	}
}
```

- [ ] **Step 2: Run to verify it fails.**

Run: `cd services/atlas-doors/atlas.com/doors && go test ./door/ -run TestRegistry ; cd -`
Expected: FAIL (`Registry`/`newRegistry` undefined).

- [ ] **Step 3: Implement `registry.go`** mirroring monsters, with the door
`storedDoor` struct (flattening tenant id/region/major/minor AND the full field +
townMapId + all coords + slot + portal + unix-milli times) and the three indices.
Key namespaces: `"door"` (store), `"door-field"`, `"door-owner"`, `"door-town"`.
The town-party index suffix is
`{tenant}:{world}:{channel}:{townMap}:{partyScope}` where `partyScope = partyId` for a
party door, or `"solo-{ownerCharacterId}"` for a solo door (so two solo slot-0 doors at
the same town don't collide — design §4.3).

```go
package door

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	atlasredis "github.com/Chronicle20/atlas/libs/atlas-redis"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	goredis "github.com/redis/go-redis/v9"
)

type storedDoor struct {
	// tenant
	TenantId string `json:"tenantId"`
	Region   string `json:"region"`
	Major    uint16 `json:"major"`
	Minor    uint16 `json:"minor"`
	// field
	WorldId   byte   `json:"worldId"`
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
	Instance  string `json:"instance"`
	// door
	AreaDoorId       uint32 `json:"areaDoorId"`
	TownDoorId       uint32 `json:"townDoorId"`
	OwnerCharacterId uint32 `json:"ownerCharacterId"`
	PartyId          uint32 `json:"partyId"`
	SkillId          uint32 `json:"skillId"`
	SkillLevel       byte   `json:"skillLevel"`
	TownMapId        uint32 `json:"townMapId"`
	Slot             byte   `json:"slot"`
	TownPortalId     uint32 `json:"townPortalId"`
	AreaX            int16  `json:"areaX"`
	AreaY            int16  `json:"areaY"`
	TownX            int16  `json:"townX"`
	TownY            int16  `json:"townY"`
	DeployMs         int64  `json:"deployMs"`
	ExpiresMs        int64  `json:"expiresMs"`
}

type Registry struct {
	reg      *atlasredis.Registry[string, storedDoor]
	fieldIdx *atlasredis.KeyedSet[string]
	ownerIdx *atlasredis.KeyedSet[string]
	townIdx  *atlasredis.KeyedSet[string]
}

var registry *Registry
var once sync.Once

func newRegistry(rc *goredis.Client) *Registry {
	id := func(s string) string { return s }
	return &Registry{
		reg:      atlasredis.NewRegistry[string, storedDoor](rc, "door", id),
		fieldIdx: atlasredis.NewKeyedSet[string](rc, "door-field", id),
		ownerIdx: atlasredis.NewKeyedSet[string](rc, "door-owner", id),
		townIdx:  atlasredis.NewKeyedSet[string](rc, "door-town", id),
	}
}

func InitRegistry(rc *goredis.Client) { once.Do(func() { registry = newRegistry(rc) }) }
func GetRegistry() *Registry          { return registry }

func partyScope(partyId, ownerCharacterId uint32) string {
	if partyId != 0 {
		return fmt.Sprintf("%d", partyId)
	}
	return fmt.Sprintf("solo-%d", ownerCharacterId)
}

func storeSuffix(t tenant.Model, id uint32) string {
	return fmt.Sprintf("%s:%d", t.Id().String(), id)
}
func fieldSuffix(t tenant.Model, f field.Model) string {
	return fmt.Sprintf("%s:%d:%d:%d:%s", t.Id().String(), f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String())
}
func ownerSuffix(t tenant.Model, characterId uint32) string {
	return fmt.Sprintf("%s:%d", t.Id().String(), characterId)
}
func townSuffix(t tenant.Model, f field.Model, townMapId _map.Id, partyId, ownerCharacterId uint32) string {
	return fmt.Sprintf("%s:%d:%d:%d:%s", t.Id().String(), f.WorldId(), f.ChannelId(), townMapId, partyScope(partyId, ownerCharacterId))
}

// Put/Get/GetInField/GetByOwner/GetInTownParty/Remove/GetAll:
// mirror monsters registry method bodies, using toStored/fromStored converters and the
// three indices. timeToMs(0 == zero time)/msToTime as in monsters. member ids stored as
// fmt.Sprintf("%d", areaDoorId). Remove reads the model first to know field/owner/town
// keys, removes from all three sets, then reg.Remove. GetAll regroups by rebuilt tenant.
```

Implement `toStored(t,m)`, `fromStored(s) (tenant.Model, Model, error)` (rebuild tenant
via the monsters helper and field via `field.NewBuilder(...).SetInstance(
uuid.MustParse(...)).Build()`), `timeToMs`/`msToTime`, and the methods:

```go
func (r *Registry) Put(ctx context.Context, t tenant.Model, m Model) error
func (r *Registry) Get(ctx context.Context, t tenant.Model, areaDoorId uint32) (Model, error)
func (r *Registry) GetInField(ctx context.Context, t tenant.Model, f field.Model) ([]Model, error)
func (r *Registry) GetByOwner(ctx context.Context, t tenant.Model, characterId uint32) ([]Model, error)
func (r *Registry) GetInTownParty(ctx context.Context, t tenant.Model, f field.Model, townMapId _map.Id, partyId, ownerCharacterId uint32) ([]Model, error)
func (r *Registry) Remove(ctx context.Context, t tenant.Model, areaDoorId uint32) error
func (r *Registry) GetAll(ctx context.Context) (map[tenant.Model][]Model, error)
func timeToMs(t time.Time) int64
func msToTime(ms int64) time.Time
```

- [ ] **Step 4: Run to verify it passes.**

Run: `cd services/atlas-doors/atlas.com/doors && go test ./door/ -run TestRegistry ; cd -`
Expected: PASS.

- [ ] **Step 5: Run rediskeyguard** to confirm no raw keyed go-redis calls leaked in.

Run: `GOWORK=off tools/redis-key-guard.sh`
Expected: clean.

- [ ] **Step 6: Commit.**

```bash
git add services/atlas-doors/atlas.com/doors/door/registry.go services/atlas-doors/atlas.com/doors/door/registry_test.go
git commit -m "feat(atlas-doors): redis registry with field/owner/town-party indices"
```

---

# PART C — Slot allocation, town resolution, cross-service data clients

Goal: pure slot/town logic (unit-tested), plus the atlas-data (map + portals + skill
effect) and atlas-parties REST clients atlas-doors needs.

### Task C1: atlas-data map + portal client

**Files:**
- Create: `services/atlas-doors/atlas.com/doors/data/map/model.go`
- Create: `services/atlas-doors/atlas.com/doors/data/map/rest.go`
- Create: `services/atlas-doors/atlas.com/doors/data/map/requests.go`
- Create: `services/atlas-doors/atlas.com/doors/data/map/processor.go`

Mirror the channel `data/map` + `data/portal` client pattern, but the map model must
expose `ReturnMapId()`, `ForcedReturnMapId()`, `Town()`, `FieldLimit()`, and
`Portals() []Portal` where `Portal` exposes `Id()`, `Name()`, `Type() uint8`, `X()`,
`Y()`, `TargetMapId()`.

- [ ] **Step 1: Implement the REST model + Extract** (`rest.go`) including the portals
to-many relationship. The map RestModel fields (verbatim tags from atlas-data):
`ReturnMapId _map.Id json:"returnMapId"`, `ForcedReturnMapId _map.Id
json:"forcedReturnMapId"`, `Town bool json:"town"`, `FieldLimit uint32
json:"fieldLimit"`, plus the portal sub-resource. Portal RestModel:
`Name string json:"name"`, `Type uint8 json:"type"`, `X int16 json:"x"`,
`Y int16 json:"y"`, `TargetMapId _map.Id json:"targetMapId"`. Add the api2go
`SetToOneReferenceID`/`SetToManyReferenceIDs` no-op stubs.

- [ ] **Step 2: Implement `requests.go`** with `requests.RootUrl("DATA")` and templates
`"data/maps/%d"` (with `?include=portals`) and `"data/maps/%d/portals"`.

- [ ] **Step 3: Implement `processor.go`**:

```go
type Processor interface {
	GetById(mapId _map.Id) (Model, error)
	GetPortals(mapId _map.Id) ([]Portal, error)
}
func NewProcessor(l logrus.FieldLogger, ctx context.Context) *ProcessorImpl
```

`GetById` fetches the map (with portals via include, or fetch portals separately and
attach). `GetPortals` fetches the `/portals` sub-resource.

- [ ] **Step 4: Write a small Extract test** asserting a portal of `Type==6` is read and
its X/Y/TargetMapId survive Extract (use a fixed RestModel, no network).

```go
func TestExtractDoorPortal(t *testing.T) {
	rm := PortalRestModel{Name: "tp", Type: 6, X: -100, Y: 200, TargetMapId: 999}
	p, err := ExtractPortal(rm)
	if err != nil || p.Type() != 6 || p.X() != -100 || p.TargetMapId() != 999 {
		t.Fatalf("portal extract wrong: %+v err=%v", p, err)
	}
}
```

- [ ] **Step 5: Run; implement until PASS.**

Run: `cd services/atlas-doors/atlas.com/doors && go test ./data/map/ ; cd -`
Expected: PASS.

- [ ] **Step 6: Commit.**

```bash
git add services/atlas-doors/atlas.com/doors/data/map
git commit -m "feat(atlas-doors): atlas-data map + portal client"
```

### Task C2: atlas-data skill-effect client (duration by level)

**Files:**
- Create: `services/atlas-doors/atlas.com/doors/data/skill/...` (model/rest/requests/processor)

- [ ] **Step 1: Mirror the channel `data/skill` client** but expose the door-relevant
effect getters: `Duration() int32` (ms; `-1` = none), `MPConsume() uint16`,
`ItemConsume() uint32`. `Processor.GetEffect(skillId uint32, level byte) (effect.Model,
error)` (level 1-based → `Effects()[level-1]`). URL `data/skills/%d`,
`requests.RootUrl("DATA")`.

- [ ] **Step 2: Write a test** that `GetEffect` returns the level-1 effect from a fixed
skill RestModel with two levels (no network — test `Extract` + the level indexing helper
directly).

- [ ] **Step 3: Run; implement until PASS.**

Run: `cd services/atlas-doors/atlas.com/doors && go test ./data/skill/... ; cd -`
Expected: PASS.

- [ ] **Step 4: Commit.**

```bash
git add services/atlas-doors/atlas.com/doors/data/skill
git commit -m "feat(atlas-doors): atlas-data skill-effect client (duration by level)"
```

### Task C3: atlas-parties client (history-sorted members)

**Files:**
- Create: `services/atlas-doors/atlas.com/doors/party/...` (model/rest/requests/processor)

- [ ] **Step 1: Mirror the channel `party` client.** Model exposes `Id()`, `LeaderId()`,
and `Members() []uint32` (ordered, join order; the registry seeds `[leaderId]` so the
slice is leader-first then join order — preserve it, do not re-sort). Requests:
`Resource="parties"`, `ByMemberId="parties?filter[members.id]=%d"`, `ById="parties/%d"`.
Processor: `GetByMemberId(characterId uint32) (Model, error)`, `GetById(partyId uint32)
(Model, error)`.

- [ ] **Step 2: Write a test** that `Extract` preserves member order across a fixed
two-member RestModel (member[0], member[1] order unchanged).

- [ ] **Step 3: Run; implement until PASS.**

Run: `cd services/atlas-doors/atlas.com/doors && go test ./party/... ; cd -`
Expected: PASS.

- [ ] **Step 4: Commit.**

```bash
git add services/atlas-doors/atlas.com/doors/party
git commit -m "feat(atlas-doors): atlas-parties client (ordered members)"
```

### Task C4: Slot + town-portal resolution (pure logic)

**Files:**
- Create: `services/atlas-doors/atlas.com/doors/door/slot.go`
- Create: `services/atlas-doors/atlas.com/doors/door/slot_test.go`

This is the heart of FR-4. Pure functions over already-fetched party members + town
portals, so they unit-test without network.

- [ ] **Step 1: Write the failing slot test** covering: solo → slot 0; member index in
party; 6-member saturation; town with ≥6 door portals indexes by slot; town with <6
door portals falls back; wire portal id is always `0x80+slot`.

```go
package door

import "testing"

func TestComputeSlotSolo(t *testing.T) {
	if got := ComputeSlot(0, []uint32{}, 42); got != 0 {
		t.Fatalf("solo slot want 0 got %d", got)
	}
}

func TestComputeSlotPartyIndex(t *testing.T) {
	members := []uint32{10, 20, 30}
	if got := ComputeSlot(7, members, 30); got != 2 {
		t.Fatalf("want slot 2 got %d", got)
	}
	if got := ComputeSlot(7, members, 10); got != 0 {
		t.Fatalf("want slot 0 got %d", got)
	}
}

func TestComputeSlotNotMemberFallsToZero(t *testing.T) {
	if got := ComputeSlot(7, []uint32{10, 20}, 99); got != 0 {
		t.Fatalf("non-member want 0 got %d", got)
	}
}

func TestResolveTownPortalWithEnoughDoorPortals(t *testing.T) {
	portals := []TownPortal{{X: -10, Y: 1}, {X: -20, Y: 2}, {X: -30, Y: 3},
		{X: -40, Y: 4}, {X: -50, Y: 5}, {X: -60, Y: 6}}
	wireId, x, y, ok := ResolveTownPortal(portals, 3, defaultTownX, defaultTownY)
	if !ok || wireId != 0x83 || x != -40 || y != 4 {
		t.Fatalf("want 0x83/-40/4 got %d/%d/%d ok=%v", wireId, x, y, ok)
	}
}

func TestResolveTownPortalFallbackWhenTooFew(t *testing.T) {
	portals := []TownPortal{{X: -10, Y: 1}} // only 1 door portal
	wireId, x, y, ok := ResolveTownPortal(portals, 3, 7, 8)
	// wire id still 0x80+slot; position falls back to provided default
	if !ok || wireId != 0x83 || x != 7 || y != 8 {
		t.Fatalf("fallback wrong: %d/%d/%d ok=%v", wireId, x, y, ok)
	}
}
```

- [ ] **Step 2: Run to verify it fails.**

Run: `cd services/atlas-doors/atlas.com/doors && go test ./door/ -run 'Slot|TownPortal' ; cd -`
Expected: FAIL (undefined).

- [ ] **Step 3: Implement `slot.go`.**

```go
package door

const (
	townPortalBase byte = 0x80
	maxPartySize        = 6
	// default door position when a town exposes too few door-type portals (design §6.3).
	defaultTownX int16 = 0
	defaultTownY int16 = 0
)

// TownPortal is an atlas-data door-type portal position (Type==6), in load order.
type TownPortal struct {
	X int16
	Y int16
}

// ComputeSlot returns the caster's 0-based party door slot (Cosmic Party.getPartyDoor).
// Solo (partyId==0) or non-member → slot 0.
func ComputeSlot(partyId uint32, members []uint32, ownerCharacterId uint32) byte {
	if partyId == 0 {
		return 0
	}
	for i, id := range members {
		if id == ownerCharacterId {
			if i >= maxPartySize {
				return maxPartySize - 1
			}
			return byte(i)
		}
	}
	return 0
}

// ResolveTownPortal maps a slot to the wire portal id (0x80+slot) and a town position.
// If the town has >slot door portals, use that portal's position; otherwise fall back to
// the provided default position (still encoding 0x80+slot on the wire). Always ok=true.
func ResolveTownPortal(doorPortals []TownPortal, slot byte, fallbackX, fallbackY int16) (wireId uint32, x int16, y int16, ok bool) {
	wireId = uint32(townPortalBase + slot)
	if int(slot) < len(doorPortals) {
		p := doorPortals[slot]
		return wireId, p.X, p.Y, true
	}
	return wireId, fallbackX, fallbackY, true
}
```

- [ ] **Step 4: Run to verify it passes.**

Run: `cd services/atlas-doors/atlas.com/doors && go test ./door/ -run 'Slot|TownPortal' ; cd -`
Expected: PASS.

- [ ] **Step 5: Commit.**

```bash
git add services/atlas-doors/atlas.com/doors/door/slot.go services/atlas-doors/atlas.com/doors/door/slot_test.go
git commit -m "feat(atlas-doors): party slot + town-portal resolution"
```

### Task C5: Town resolution helper (return vs forced-return)

**Files:**
- Create: `services/atlas-doors/atlas.com/doors/door/town.go`
- Create: `services/atlas-doors/atlas.com/doors/door/town_test.go`

- [ ] **Step 1: Write the failing test.** `ResolveTownMap(returnMapId, forcedReturnMapId)`:
forced-return wins when it is a real map; otherwise return map; "no valid return" sentinel
detection.

```go
package door

import "testing"

func TestResolveTownMapForcedWins(t *testing.T) {
	if got := ResolveTownMap(104000000, 100000000); got != 100000000 {
		t.Fatalf("forced should win, got %d", got)
	}
}
func TestResolveTownMapReturnWhenNoForced(t *testing.T) {
	if got := ResolveTownMap(104000000, noMap); got != 104000000 {
		t.Fatalf("want return map, got %d", got)
	}
}
func TestHasReturnMapFalseWhenNone(t *testing.T) {
	if HasValidReturn(noMap, noMap) {
		t.Fatalf("expected no valid return")
	}
}
```

- [ ] **Step 2: Run to verify it fails.**

Run: `cd services/atlas-doors/atlas.com/doors && go test ./door/ -run 'Town|Return' ; cd -`
Expected: FAIL.

- [ ] **Step 3: Implement `town.go`.** `noMap` is MapleStory's "no map" sentinel — confirm
the exact sentinel atlas-data emits for an absent return map (check
`libs/atlas-constants/map` for `EmptyMapId`; use that constant rather than a literal).

```go
package door

import _map "github.com/Chronicle20/atlas/libs/atlas-constants/map"

// confirm against libs/atlas-constants/map — use the EmptyMapId constant if present.
const noMap _map.Id = 999999999 // MapleStory "no map" sentinel

func HasValidReturn(returnMapId, forcedReturnMapId _map.Id) bool {
	return ResolveTownMap(returnMapId, forcedReturnMapId) != noMap
}

func ResolveTownMap(returnMapId, forcedReturnMapId _map.Id) _map.Id {
	if forcedReturnMapId != noMap && forcedReturnMapId != 0 {
		return forcedReturnMapId
	}
	return returnMapId
}
```

- [ ] **Step 4: Run to verify it passes; then commit.**

Run: `cd services/atlas-doors/atlas.com/doors && go test ./door/ -run 'Town|Return' ; cd -`
Expected: PASS.

```bash
git add services/atlas-doors/atlas.com/doors/door/town.go services/atlas-doors/atlas.com/doors/door/town_test.go
git commit -m "feat(atlas-doors): return/forced-return town resolution"
```

---

# PART D — Event contracts, processor, producer, REST

### Task D1: Event topic envelope + bodies

**Files:**
- Create: `services/atlas-doors/atlas.com/doors/door/kafka.go`

- [ ] **Step 1: Define the status-event contract** (mirror monsters `kafka.go` envelope
shape). This is the contract atlas-channel consumes — keep field names stable.

```go
package door

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

const EnvEventTopicDoorStatus = "EVENT_TOPIC_DOOR_STATUS"

const (
	EventDoorStatusCreated     = "CREATED"
	EventDoorStatusRemoved     = "REMOVED"
	EventDoorStatusSlotChanged = "SLOT_CHANGED"
)

// Removal reasons (FR-6.1/6.2).
const (
	RemoveReasonExpiry         = "EXPIRY"
	RemoveReasonLogout         = "LOGOUT"
	RemoveReasonChannelChanged = "CHANNEL_CHANGED"
	RemoveReasonLeftField      = "LEFT_FIELD"
	RemoveReasonRecast         = "RECAST"
)

type StatusEvent[E any] struct {
	WorldId          world.Id   `json:"worldId"`
	ChannelId        channel.Id `json:"channelId"`
	MapId            _map.Id    `json:"mapId"`     // area field map (event key)
	Instance         uuid.UUID  `json:"instance"`
	PairId           uint32     `json:"pairId"`
	OwnerCharacterId uint32     `json:"ownerCharacterId"`
	PartyId          uint32     `json:"partyId"`
	Type             string     `json:"type"`
	Body             E          `json:"body"`
}

type CreatedBody struct {
	AreaDoorId   uint32  `json:"areaDoorId"`
	TownDoorId   uint32  `json:"townDoorId"`
	TownMapId    _map.Id `json:"townMapId"`
	Slot         byte    `json:"slot"`
	TownPortalId uint32  `json:"townPortalId"`
	AreaX        int16   `json:"areaX"`
	AreaY        int16   `json:"areaY"`
	TownX        int16   `json:"townX"`
	TownY        int16   `json:"townY"`
	SkillId      uint32  `json:"skillId"`
	SkillLevel   byte    `json:"skillLevel"`
	ExpiresAt    int64   `json:"expiresAt"` // unix-milli
}

type RemovedBody struct {
	AreaDoorId uint32  `json:"areaDoorId"`
	TownDoorId uint32  `json:"townDoorId"`
	TownMapId  _map.Id `json:"townMapId"`
	Slot       byte    `json:"slot"`
	Reason     string  `json:"reason"`
}

type SlotChangedBody struct {
	AreaDoorId   uint32  `json:"areaDoorId"`
	TownDoorId   uint32  `json:"townDoorId"`
	TownMapId    _map.Id `json:"townMapId"`
	OldSlot      byte    `json:"oldSlot"`
	NewSlot      byte    `json:"newSlot"`
	TownPortalId uint32  `json:"townPortalId"`
	TownX        int16   `json:"townX"`
	TownY        int16   `json:"townY"`
}
```

- [ ] **Step 2: Verify build; commit.**

Run: `cd services/atlas-doors/atlas.com/doors && go build ./door/ ; cd -`

```bash
git add services/atlas-doors/atlas.com/doors/door/kafka.go
git commit -m "feat(atlas-doors): door status event contract"
```

### Task D2: Event providers

**Files:**
- Create: `services/atlas-doors/atlas.com/doors/door/producer.go`

- [ ] **Step 1: Implement the three providers** keyed by area-field map id (mirror
monsters `producer.go` using `producer.CreateKey` + `producer.SingleMessageProvider`):

```go
package door

import (
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func createdEventProvider(m Model) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(m.Field().MapId()))
	value := StatusEvent[CreatedBody]{
		WorldId: m.Field().WorldId(), ChannelId: m.Field().ChannelId(),
		MapId: m.Field().MapId(), Instance: m.Field().Instance(),
		PairId: m.PairId(), OwnerCharacterId: m.OwnerCharacterId(), PartyId: m.PartyId(),
		Type: EventDoorStatusCreated,
		Body: CreatedBody{
			AreaDoorId: m.AreaDoorId(), TownDoorId: m.TownDoorId(), TownMapId: m.TownMapId(),
			Slot: m.Slot(), TownPortalId: m.TownPortalId(),
			AreaX: m.AreaX(), AreaY: m.AreaY(), TownX: m.TownX(), TownY: m.TownY(),
			SkillId: m.SkillId(), SkillLevel: m.SkillLevel(), ExpiresAt: timeToMs(m.ExpiresAt()),
		},
	}
	return producer.SingleMessageProvider(key, &value)
}

func removedEventProvider(m Model, reason string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(m.Field().MapId()))
	value := StatusEvent[RemovedBody]{
		WorldId: m.Field().WorldId(), ChannelId: m.Field().ChannelId(),
		MapId: m.Field().MapId(), Instance: m.Field().Instance(),
		PairId: m.PairId(), OwnerCharacterId: m.OwnerCharacterId(), PartyId: m.PartyId(),
		Type: EventDoorStatusRemoved,
		Body: RemovedBody{AreaDoorId: m.AreaDoorId(), TownDoorId: m.TownDoorId(),
			TownMapId: m.TownMapId(), Slot: m.Slot(), Reason: reason},
	}
	return producer.SingleMessageProvider(key, &value)
}

func slotChangedEventProvider(m Model, oldSlot byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(m.Field().MapId()))
	value := StatusEvent[SlotChangedBody]{
		WorldId: m.Field().WorldId(), ChannelId: m.Field().ChannelId(),
		MapId: m.Field().MapId(), Instance: m.Field().Instance(),
		PairId: m.PairId(), OwnerCharacterId: m.OwnerCharacterId(), PartyId: m.PartyId(),
		Type: EventDoorStatusSlotChanged,
		Body: SlotChangedBody{AreaDoorId: m.AreaDoorId(), TownDoorId: m.TownDoorId(),
			TownMapId: m.TownMapId(), OldSlot: oldSlot, NewSlot: m.Slot(),
			TownPortalId: m.TownPortalId(), TownX: m.TownX(), TownY: m.TownY()},
	}
	return producer.SingleMessageProvider(key, &value)
}
```

- [ ] **Step 2: Build; commit.**

```bash
git add services/atlas-doors/atlas.com/doors/door/producer.go
git commit -m "feat(atlas-doors): door event providers"
```

### Task D3: Processor — Spawn (with recast replace) + Remove + Get + Reslot

**Files:**
- Create: `services/atlas-doors/atlas.com/doors/door/processor.go`
- Create: `services/atlas-doors/atlas.com/doors/door/resolver.go`
- Create: `services/atlas-doors/atlas.com/doors/door/processor_test.go`

The processor is field-injected (emit + a `resolver` source) so it unit-tests without
Kafka/REST (mirror the monsters processor seam).

- [ ] **Step 1: Write the failing processor test** for the core behaviors:
  - `Spawn` allocates two oids, persists, emits CREATED, returns the model with
    pairId==areaDoorId.
  - `Spawn` recast: an existing owner door is removed (REMOVED reason RECAST emitted)
    before the new one is deployed.
  - `Spawn` fails cleanly when the second (town) oid allocation fails (no persist, no
    CREATED, area oid released — allocate area first, then town).
  - `RemoveByOwner` removes + emits REMOVED with the given reason; idempotent.
  - `Reslot` updates slot/portal/pos + emits SLOT_CHANGED; no-op when slot unchanged.

Use a fake emitter capturing `(topic, decoded event Type)` and a fake `resolver`
returning canned town/slot inputs. Build the processor with the registry test harness
from B3 and a counter-based id allocator stub.

```go
package door

import (
	"context"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

type fakeResolver struct {
	partyId     uint32
	townMapId   _map.Id
	doorPortals []TownPortal
	members     []uint32
	durationMs  int32
}

func (f fakeResolver) PartyIdFor(_ context.Context, _ uint32) (uint32, error) { return f.partyId, nil }
func (f fakeResolver) ResolveSpawn(_ context.Context, _ field.Model, ownerCharacterId, partyId, _ uint32, _ byte) (spawnPlan, error) {
	slot := ComputeSlot(partyId, f.members, ownerCharacterId)
	wireId, tx, ty, _ := ResolveTownPortal(f.doorPortals, slot, defaultTownX, defaultTownY)
	return spawnPlan{townMapId: f.townMapId, slot: slot, townPortalId: wireId, townX: tx, townY: ty, durationMs: f.durationMs}, nil
}

func TestSpawnCreatesPairAndEmitsCreated(t *testing.T) { /* fill with harness */ }
func TestSpawnRecastReplacesExisting(t *testing.T)      { /* pre-seed owner door */ }
func TestSpawnFailsCleanlyOnAllocError(t *testing.T)    { /* 2nd alloc errors */ }
```

- [ ] **Step 2: Run to verify it fails.**

Run: `cd services/atlas-doors/atlas.com/doors && go test ./door/ -run Spawn ; cd -`
Expected: FAIL.

- [ ] **Step 3: Implement `processor.go`.**

```go
package door

import (
	"context"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	GetById(areaDoorId uint32) (Model, error)
	GetInField(f field.Model) ([]Model, error)
	Spawn(f field.Model, ownerCharacterId, skillId uint32, skillLevel byte, x, y int16) (Model, error)
	RemoveByOwner(ownerCharacterId uint32, reason string) error
	RemoveByOwnerIfLeftField(ownerCharacterId uint32, newField field.Model) error
	Reslot(areaDoorId uint32, newSlot byte, townPortalId uint32, townX, townY int16) error
}

type spawnPlan struct {
	townMapId    _map.Id
	slot         byte
	townPortalId uint32
	townX        int16
	townY        int16
	durationMs   int32
}

type resolver interface {
	ResolveSpawn(ctx context.Context, f field.Model, ownerCharacterId, partyId, skillId uint32, level byte) (spawnPlan, error)
	PartyIdFor(ctx context.Context, ownerCharacterId uint32) (uint32, error)
}

type emitter func(topic string, p model.Provider[[]kafka.Message]) error

type ProcessorImpl struct {
	l    logrus.FieldLogger
	ctx  context.Context
	t    tenant.Model
	emit emitter
	res  resolver
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *ProcessorImpl {
	return &ProcessorImpl{
		l: l, ctx: ctx, t: tenant.MustFromContext(ctx),
		emit: func(topic string, p model.Provider[[]kafka.Message]) error {
			return producer.ProviderImpl(l)(ctx)(topic)(p)
		},
		res: newRestResolver(l, ctx),
	}
}

func (p *ProcessorImpl) GetById(areaDoorId uint32) (Model, error) {
	return GetRegistry().Get(p.ctx, p.t, areaDoorId)
}
func (p *ProcessorImpl) GetInField(f field.Model) ([]Model, error) {
	return GetRegistry().GetInField(p.ctx, p.t, f)
}

func (p *ProcessorImpl) Spawn(f field.Model, ownerCharacterId, skillId uint32, skillLevel byte, x, y int16) (Model, error) {
	// FR-1.4 recast: remove any existing owner door first.
	if err := p.RemoveByOwner(ownerCharacterId, RemoveReasonRecast); err != nil {
		p.l.WithError(err).Warnf("recast cleanup failed for character %d", ownerCharacterId)
	}

	partyId, err := p.res.PartyIdFor(p.ctx, ownerCharacterId)
	if err != nil {
		partyId = 0
	}
	plan, err := p.res.ResolveSpawn(p.ctx, f, ownerCharacterId, partyId, skillId, skillLevel)
	if err != nil {
		p.l.WithError(err).Warnf("door spawn rejected (resolve) for character %d", ownerCharacterId)
		return Model{}, err
	}

	areaId, err := GetIdAllocator().Allocate(p.ctx, p.t)
	if err != nil {
		p.l.WithError(err).Errorf("door area oid alloc failed")
		return Model{}, err
	}
	townId, err := GetIdAllocator().Allocate(p.ctx, p.t)
	if err != nil {
		GetIdAllocator().Release(p.ctx, p.t, areaId)
		p.l.WithError(err).Errorf("door town oid alloc failed")
		return Model{}, err
	}

	now := time.Now()
	expires := now
	if plan.durationMs > 0 {
		expires = now.Add(time.Duration(plan.durationMs) * time.Millisecond)
	}
	m := NewBuilder().
		SetAreaDoorId(areaId).SetTownDoorId(townId).
		SetOwnerCharacterId(ownerCharacterId).SetPartyId(partyId).
		SetSkillId(skillId).SetSkillLevel(skillLevel).SetField(f).
		SetTownMapId(plan.townMapId).SetSlot(plan.slot).SetTownPortalId(plan.townPortalId).
		SetAreaX(x).SetAreaY(y).SetTownX(plan.townX).SetTownY(plan.townY).
		SetDeployTime(now).SetExpiresAt(expires).Build()

	if err := GetRegistry().Put(p.ctx, p.t, m); err != nil {
		GetIdAllocator().Release(p.ctx, p.t, areaId)
		GetIdAllocator().Release(p.ctx, p.t, townId)
		return Model{}, err
	}
	if err := p.emit(EnvEventTopicDoorStatus, createdEventProvider(m)); err != nil {
		p.l.WithError(err).Errorf("failed emitting CREATED for door %d", areaId)
	}
	return m, nil
}

func (p *ProcessorImpl) RemoveByOwner(ownerCharacterId uint32, reason string) error {
	doors, err := GetRegistry().GetByOwner(p.ctx, p.t, ownerCharacterId)
	if err != nil {
		return err
	}
	for _, m := range doors {
		if err := GetRegistry().Remove(p.ctx, p.t, m.AreaDoorId()); err != nil {
			p.l.WithError(err).Warnf("failed removing door %d", m.AreaDoorId())
			continue
		}
		GetIdAllocator().Release(p.ctx, p.t, m.AreaDoorId())
		GetIdAllocator().Release(p.ctx, p.t, m.TownDoorId())
		if err := p.emit(EnvEventTopicDoorStatus, removedEventProvider(m, reason)); err != nil {
			p.l.WithError(err).Errorf("failed emitting REMOVED for door %d", m.AreaDoorId())
		}
	}
	return nil
}

// RemoveByOwnerIfLeftField removes the owner's door only when newField is neither the
// door's source field nor its town map (walking into the town the door spans is a warp,
// not abandonment — FR-6.2 / design §5.3).
func (p *ProcessorImpl) RemoveByOwnerIfLeftField(ownerCharacterId uint32, newField field.Model) error {
	doors, err := GetRegistry().GetByOwner(p.ctx, p.t, ownerCharacterId)
	if err != nil {
		return err
	}
	for _, m := range doors {
		src := m.Field()
		sameSource := src.WorldId() == newField.WorldId() && src.ChannelId() == newField.ChannelId() &&
			src.MapId() == newField.MapId() && src.Instance() == newField.Instance()
		intoTown := newField.MapId() == m.TownMapId()
		if sameSource || intoTown {
			continue
		}
		if err := GetRegistry().Remove(p.ctx, p.t, m.AreaDoorId()); err != nil {
			continue
		}
		GetIdAllocator().Release(p.ctx, p.t, m.AreaDoorId())
		GetIdAllocator().Release(p.ctx, p.t, m.TownDoorId())
		_ = p.emit(EnvEventTopicDoorStatus, removedEventProvider(m, RemoveReasonLeftField))
	}
	return nil
}

func (p *ProcessorImpl) Reslot(areaDoorId uint32, newSlot byte, townPortalId uint32, townX, townY int16) error {
	m, err := GetRegistry().Get(p.ctx, p.t, areaDoorId)
	if err != nil {
		return err
	}
	oldSlot := m.Slot()
	if oldSlot == newSlot {
		return nil
	}
	n := m.Reslot(newSlot, townPortalId, townX, townY)
	if err := GetRegistry().Put(p.ctx, p.t, n); err != nil {
		return err
	}
	return p.emit(EnvEventTopicDoorStatus, slotChangedEventProvider(n, oldSlot))
}
```

- [ ] **Step 4: Implement `resolver.go`** — `newRestResolver(l, ctx) resolver` wiring the
`data/map`, `data/skill`, and `party` clients. `PartyIdFor` reads the party (0 on
not-found). `ResolveSpawn`: fetch map metadata; reject (error) if `!HasValidReturn` or
`Town` or `FieldLimitNoMysticDoor` set (defensive re-check — the channel pre-checks too);
resolve the town map; fetch town door portals (Type==6, load order) → `[]TownPortal`;
read party members; `ComputeSlot`; `ResolveTownPortal`; read the skill effect
`Duration()` for the level (treat `-1`/`<=0` as "no expiry" → `durationMs` 0). Return the
`spawnPlan`.

- [ ] **Step 5: Run to verify it passes.**

Run: `cd services/atlas-doors/atlas.com/doors && go test ./door/ -run Spawn ; cd -`
Expected: PASS.

- [ ] **Step 6: Commit.**

```bash
git add services/atlas-doors/atlas.com/doors/door/processor.go services/atlas-doors/atlas.com/doors/door/resolver.go services/atlas-doors/atlas.com/doors/door/processor_test.go
git commit -m "feat(atlas-doors): processor spawn/remove/reslot with recast + fail-clean alloc"
```

### Task D4: REST resource (GET door, GET doors-in-field)

**Files:**
- Create: `services/atlas-doors/atlas.com/doors/door/resource.go`
- Create: `services/atlas-doors/atlas.com/doors/door/rest.go`
- Create: `services/atlas-doors/atlas.com/doors/world/resource.go`
- Create: `services/atlas-doors/atlas.com/doors/door/resource_test.go`

- [ ] **Step 1: Write the failing Transform test.**

```go
package door

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

func TestTransform(t *testing.T) {
	f := field.NewBuilder(1, 2, 100000000).Build()
	m := NewBuilder().SetAreaDoorId(1_000_001).SetTownDoorId(1_000_002).
		SetOwnerCharacterId(42).SetTownMapId(104000000).SetSlot(2).
		SetTownPortalId(0x82).SetField(f).Build()
	rm, err := Transform(m)
	if err != nil || rm.GetID() != "1000001" || rm.OwnerCharacterId != 42 ||
		rm.TownPortalId != 0x82 || rm.MapId != 100000000 {
		t.Fatalf("transform wrong: %+v err=%v", rm, err)
	}
	if rm.GetName() != "doors" {
		t.Fatalf("resource name want doors got %s", rm.GetName())
	}
}
```

- [ ] **Step 2: Run to verify it fails.**

Run: `cd services/atlas-doors/atlas.com/doors && go test ./door/ -run TestTransform ; cd -`
Expected: FAIL.

- [ ] **Step 3: Implement `resource.go`** (RestModel + `GetID/SetID/GetName` returning
`"doors"` + `Transform(m) (RestModel, error)` exposing areaDoorId, townDoorId, pairId,
owner, partyId, world/channel/map/instance, townMapId, slot, townPortalId, area/town
coords, skill id/level, expiresAt). Implement `rest.go` (`InitResource(si)` subrouting
`/doors`, `GET /doors/{doorId}` → `ParseDoorId` → `GetById` → `MarshalResponse`).
Implement `world/resource.go` (`GET /worlds/{worldId}/channels/{channelId}/maps/{mapId}/
instances/{instanceId}/doors` → `GetInField` → `[]RestModel`).

- [ ] **Step 4: Run to verify it passes; commit.**

Run: `cd services/atlas-doors/atlas.com/doors && go test ./door/ -run TestTransform ; cd -`
Expected: PASS.

```bash
git add services/atlas-doors/atlas.com/doors/door/resource.go services/atlas-doors/atlas.com/doors/door/rest.go services/atlas-doors/atlas.com/doors/world services/atlas-doors/atlas.com/doors/door/resource_test.go
git commit -m "feat(atlas-doors): JSON:API door resource + in-field list route"
```

---

# PART E — Consumers, expiry sweep, main wiring

### Task E1: Door command consumer (SPAWN / REMOVE)

**Files:**
- Create: `services/atlas-doors/atlas.com/doors/kafka/consumer/door/kafka.go`
- Create: `services/atlas-doors/atlas.com/doors/kafka/consumer/door/consumer.go`

- [ ] **Step 1: Define the command contract** (`kafka.go`). This is the contract
atlas-channel emits.

```go
package door

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

const EnvCommandTopic = "COMMAND_TOPIC_DOOR"

const (
	CommandTypeSpawn  = "SPAWN"
	CommandTypeRemove = "REMOVE"
)

type Command[E any] struct {
	WorldId          world.Id   `json:"worldId"`
	ChannelId        channel.Id `json:"channelId"`
	MapId            _map.Id    `json:"mapId"`
	Instance         uuid.UUID  `json:"instance"`
	OwnerCharacterId uint32     `json:"ownerCharacterId"`
	Type             string     `json:"type"`
	Body             E          `json:"body"`
}

type SpawnBody struct {
	SkillId    uint32 `json:"skillId"`
	SkillLevel byte   `json:"skillLevel"`
	X          int16  `json:"x"`
	Y          int16  `json:"y"`
}

type RemoveBody struct {
	Reason string `json:"reason"`
}
```

- [ ] **Step 2: Implement `consumer.go`** mirroring monsters' command consumer:
`InitConsumers` with `SetHeaderParsers(Span, Tenant)` + `SetStartOffset(LastOffset)`;
`InitHandlers` registering `handleSpawn` and `handleRemove`, each guarding `c.Type`.

```go
func handleSpawn(l logrus.FieldLogger, ctx context.Context, c Command[SpawnBody]) {
	if c.Type != CommandTypeSpawn { return }
	f := field.NewBuilder(c.WorldId, c.ChannelId, c.MapId).SetInstance(c.Instance).Build()
	_, err := door.NewProcessor(l, ctx).Spawn(f, c.OwnerCharacterId, c.Body.SkillId, c.Body.SkillLevel, c.Body.X, c.Body.Y)
	if err != nil { l.WithError(err).Debugf("door spawn rejected for character %d", c.OwnerCharacterId) }
}

func handleRemove(l logrus.FieldLogger, ctx context.Context, c Command[RemoveBody]) {
	if c.Type != CommandTypeRemove { return }
	reason := c.Body.Reason
	if reason == "" { reason = door.RemoveReasonRecast }
	_ = door.NewProcessor(l, ctx).RemoveByOwner(c.OwnerCharacterId, reason)
}
```

- [ ] **Step 3: Build; commit.**

Run: `cd services/atlas-doors/atlas.com/doors && go build ./... ; cd -`

```bash
git add services/atlas-doors/atlas.com/doors/kafka/consumer/door
git commit -m "feat(atlas-doors): door command consumer (SPAWN/REMOVE)"
```

### Task E2: Character-status cleanup consumer

**Files:**
- Create: `services/atlas-doors/atlas.com/doors/kafka/consumer/character/kafka.go`
- Create: `services/atlas-doors/atlas.com/doors/kafka/consumer/character/consumer.go`

- [ ] **Step 1: Mirror monsters' character-status consumer** (`EVENT_TOPIC_CHARACTER_
STATUS`, LOGOUT/CHANNEL_CHANGED/MAP_CHANGED). Handlers:

```go
func handleLogout(l, ctx, e StatusEvent[LogoutBody]) {
	if e.Type != StatusEventTypeLogout { return }
	_ = door.NewProcessor(l, ctx).RemoveByOwner(e.CharacterId, door.RemoveReasonLogout)
}
func handleChannelChanged(l, ctx, e StatusEvent[ChannelChangedBody]) {
	if e.Type != StatusEventTypeChannelChanged { return }
	_ = door.NewProcessor(l, ctx).RemoveByOwner(e.CharacterId, door.RemoveReasonChannelChanged)
}
func handleMapChanged(l, ctx, e StatusEvent[MapChangedBody]) {
	if e.Type != StatusEventTypeMapChanged { return }
	f := field.NewBuilder(e.WorldId, e.Body.ChannelId, e.Body.TargetMapId).SetInstance(e.Body.Instance).Build()
	_ = door.NewProcessor(l, ctx).RemoveByOwnerIfLeftField(e.CharacterId, f)
}
```

(Confirm the exact `MAP_CHANGED` body field names against the real
`EVENT_TOPIC_CHARACTER_STATUS` contract in monsters/channel — the new field is the
*destination* map/channel/instance.)

- [ ] **Step 2: Build; commit.**

```bash
git add services/atlas-doors/atlas.com/doors/kafka/consumer/character
git commit -m "feat(atlas-doors): character-status cleanup consumer (logout/channel/map)"
```

### Task E3: Party-status reslot consumer

**Files:**
- Create: `services/atlas-doors/atlas.com/doors/kafka/consumer/party/kafka.go`
- Create: `services/atlas-doors/atlas.com/doors/kafka/consumer/party/consumer.go`
- Create: `services/atlas-doors/atlas.com/doors/door/reslot.go`
- Create: `services/atlas-doors/atlas.com/doors/door/reslot_test.go`

- [ ] **Step 1: Write the failing reslot-routine test.** `ReslotParty` recomputes every
party member's slot from new membership and `Reslot`s only owners whose slot changed
(FR-4.3/FR-6.4). Drive with a fake party member list + pre-seeded doors + a fake town
portal resolver, asserting which owners get SLOT_CHANGED.

```go
func TestReslotPartyRecomputesChangedSlots(t *testing.T) {
	// party members [A,B,C]; A has a door at slot 0, C has a door at slot 2.
	// new membership [B,C] (A left): C now slot 1 -> SLOT_CHANGED for C; A re-slotted solo 0.
}
```

- [ ] **Step 2: Run (FAIL); implement `reslot.go`** — `ReslotParty(ctx, partyId, members,
townPortalsByMap func(_map.Id) []TownPortal)`: for each member with a live door in this
party, compute the new slot from the new membership ordering, resolve the new town portal,
and `Reslot(...)`. A member no longer in the party → solo scope, slot 0.

- [ ] **Step 3: Implement `consumer.go`** consuming `EVENT_TOPIC_PARTY_STATUS` membership
changes (join/leave/leader/disband) and calling `ReslotParty`. Mirror the monsters/channel
party event envelope; confirm the exact event Type strings + member-list body shape.

- [ ] **Step 4: Run (PASS); commit.**

```bash
git add services/atlas-doors/atlas.com/doors/kafka/consumer/party services/atlas-doors/atlas.com/doors/door/reslot.go services/atlas-doors/atlas.com/doors/door/reslot_test.go
git commit -m "feat(atlas-doors): party-status reslot consumer + routine"
```

### Task E4: Leader-elected expiry sweep

**Files:**
- Create: `services/atlas-doors/atlas.com/doors/door/expiry_task.go`
- Create: `services/atlas-doors/atlas.com/doors/door/expiry_task_test.go`

- [ ] **Step 1: Write the failing expiry test.** Seed an expired door past grace, an
expired-but-within-grace door, and a future door; `Run()` removes only the first.

```go
func TestExpiryRemovesOnlyExpiredPastGrace(t *testing.T) {
	// door1: deployTime=now-10m, expiresAt=now-1m -> removed.
	// door2: deployTime=now, expiresAt=now-1ms (rapid cancel) -> NOT removed (grace).
	// door3: expiresAt in future -> NOT removed.
}
```

- [ ] **Step 2: Run (FAIL); implement `expiry_task.go`** mirroring monsters' expiry task
(`tasks.Task`, `GetRegistry().GetAll` grouped by tenant, `tenant.WithContext`,
`newProcessor` field-injected). Add the grace guard:

```go
const deployGrace = 3 * time.Second

func (t *ExpiryTask) Run() {
	all, err := GetRegistry().GetAll(t.ctx)
	if err != nil { t.l.WithError(err).Errorf("door expiry sweep failed"); return }
	now := time.Now()
	for ten, ms := range all {
		tctx := tenant.WithContext(t.ctx, ten)
		p := t.newProcessor(t.l, tctx)
		for _, m := range ms {
			if m.ExpiresAt().IsZero() || now.Before(m.ExpiresAt()) { continue }
			if now.Sub(m.DeployTime()) < deployGrace { continue } // FR-6.3
			if err := p.RemoveByOwner(m.OwnerCharacterId(), RemoveReasonExpiry); err != nil {
				t.l.WithError(err).Warnf("failed expiring door %d", m.AreaDoorId())
			}
		}
	}
}
```

- [ ] **Step 3: Run (PASS); commit.**

```bash
git add services/atlas-doors/atlas.com/doors/door/expiry_task.go services/atlas-doors/atlas.com/doors/door/expiry_task_test.go
git commit -m "feat(atlas-doors): leader-elected expiry sweep with deploy grace"
```

### Task E5: `main.go` wiring

**Files:**
- Create: `services/atlas-doors/atlas.com/doors/main.go`
- Create: `services/atlas-doors/atlas.com/doors/main_leader_test.go`

- [ ] **Step 1: Mirror `monsters/main.go`** substituting door packages: `InitIdAllocator`
+ `InitRegistry`; register the door command, character-status, and party-status consumers
(`InitConsumers` + `InitHandlers`); REST routes `door.InitResource` + `world.InitResource`
+ `/metrics` + `/debug/consumers`; leader block `lock.New(rc, "doors-sweep", …)` running
`tasks.Register(...)(door.NewExpiryTask(l, leaderCtx, time.Second))`; `serviceName =
"atlas-doors"`, `consumerGroupId = consumergroup.Resolve("Door Registry Service")`.

- [ ] **Step 2: Mirror `monsters/main_leader_test.go`** (asserts the leader gate
enable/disable behavior compiles + the sweep registration path).

- [ ] **Step 3: Build the whole service + run all tests.**

Run: `cd services/atlas-doors/atlas.com/doors && go build ./... && go vet ./... && go test -race ./... ; cd -`
Expected: builds, vet clean, tests pass.

- [ ] **Step 4: Bake the service image** (CLAUDE.md mandatory step).

Run: `docker buildx bake atlas-doors`
Expected: image builds.

- [ ] **Step 5: Commit.**

```bash
git add services/atlas-doors/atlas.com/doors/main.go services/atlas-doors/atlas.com/doors/main_leader_test.go
git commit -m "feat(atlas-doors): main wiring (consumers, REST, leader-elected expiry)"
```

---

# PART F — `libs/atlas-packet/door` packets

Goal: the version-branching clientbound encoders (`spawnDoor`, `removeDoor`,
`spawnPortal`) and the serverbound enter-door decoder, with per-version golden tests.
**`playPortalSound` is NOT a new packet** — the channel reuses the existing character
simple-effect (context.md #9). **Opcodes are config, not Go** (context.md #2) — these
packets are referenced by writer/handle name only.

> **Byte structure source:** Cosmic `tools/PacketCreator.java` (`spawnDoor`,
> `removeDoor`, `spawnPortal`) and `DoorHandler.java` (enter-door) give the v83 field
> order. **Per-version opcode values and any byte deltas are Part H IDA work** — do NOT
> invent them here. Where a version is known to diverge structurally, branch on
> `t.IsRegion("GMS") && t.MajorAtLeast(87)`; otherwise the v83 layout applies to
> v83–v86 (off-by-one). If IDA later shows a structural delta, add the branch then.

### Task F1: enter-door serverbound decoder

**Files:**
- Create: `libs/atlas-packet/door/serverbound/enter.go`
- Create: `libs/atlas-packet/door/serverbound/enter_test.go`

- [ ] **Step 1: Write the failing roundtrip test** across `pt.Variants`.

```go
package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestEnterDoorRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			in := Enter{ownerId: 4242, direction: 1}
			out := Enter{}
			pt.RoundTrip(t, ctx, in.Encode, out.Decode, nil)
			if out.OwnerId() != in.OwnerId() || out.Direction() != in.Direction() {
				t.Fatalf("roundtrip mismatch: %+v vs %+v", out, in)
			}
		})
	}
}
```

- [ ] **Step 2: Run to verify it fails.**

Run: `cd libs/atlas-packet && go test ./door/... ; cd -`
Expected: FAIL (package missing).

- [ ] **Step 3: Implement `enter.go`** (Cosmic `DoorHandler`: `int ownerId`, `byte
direction`):

```go
package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const EnterDoorHandle = "EnterDoorHandle"

type Enter struct {
	ownerId   uint32
	direction byte
}

func (m Enter) OwnerId() uint32   { return m.ownerId }
func (m Enter) Direction() byte   { return m.direction }
func (m Enter) Operation() string { return EnterDoorHandle }
func (m Enter) String() string    { return fmt.Sprintf("Enter{ownerId=%d direction=%d}", m.ownerId, m.direction) }

func (m *Enter) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.ownerId = r.ReadUint32()
		m.direction = r.ReadByte()
	}
}

// Encode is symmetric, for tests + completeness.
func (m Enter) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.ownerId)
		w.WriteByte(m.direction)
		return w.Bytes()
	}
}
```

- [ ] **Step 4: Run to verify it passes; commit.**

Run: `cd libs/atlas-packet && go test ./door/serverbound/ ; cd -`
Expected: PASS.

```bash
git add libs/atlas-packet/door/serverbound
git commit -m "feat(atlas-packet): enter-door serverbound decoder"
```

### Task F2: spawnDoor clientbound encoder

**Files:**
- Create: `libs/atlas-packet/door/clientbound/spawn.go`
- Create: `libs/atlas-packet/door/clientbound/spawn_test.go`

Cosmic `PacketCreator.spawnDoor(int oid, Point pos, boolean town)`. Confirm the exact v83
field order from `~/source/Cosmic/.../PacketCreator.java` `spawnDoor` before finalizing;
the encoder follows that order (oid int, position, town flag — adjust to the verified
Cosmic layout).

- [ ] **Step 1: Write the failing test** — roundtrip across `pt.Variants` plus a v83
golden-byte assertion and a v83≡v84..86 equality + v87(-path) assertion (mirror
`character/clientbound/version_bounds_test.go`). No timestamp in this packet, so use
`bytes.Equal`.

- [ ] **Step 2: Run to verify it fails.**

- [ ] **Step 3: Implement `spawn.go`** with `const SpawnDoorWriter = "SpawnDoor"`,
`NewSpawnDoor(oid uint32, x, y int16, town bool)`, getters, `Operation()`, `String()`,
and the version-branched `Encode`. Default (v83–v86) layout from Cosmic; add a
`MajorAtLeast(87)` branch only when IDA (Part H) shows a delta.

- [ ] **Step 4: Run to verify it passes; commit.**

```bash
git add libs/atlas-packet/door/clientbound/spawn.go libs/atlas-packet/door/clientbound/spawn_test.go
git commit -m "feat(atlas-packet): spawnDoor clientbound encoder"
```

### Task F3: removeDoor clientbound encoder

**Files:**
- Create: `libs/atlas-packet/door/clientbound/remove.go`
- Create: `libs/atlas-packet/door/clientbound/remove_test.go`

Cosmic `removeDoor(int oid, boolean town)`.

- [ ] **Step 1: Write the failing roundtrip + golden test** (mirror F2).
- [ ] **Step 2: Run (FAIL).**
- [ ] **Step 3: Implement `remove.go`** (`const RemoveDoorWriter = "RemoveDoor"`,
`NewRemoveDoor(oid uint32, town bool)`, version-branched `Encode`).
- [ ] **Step 4: Run (PASS); commit.**

```bash
git add libs/atlas-packet/door/clientbound/remove.go libs/atlas-packet/door/clientbound/remove_test.go
git commit -m "feat(atlas-packet): removeDoor clientbound encoder"
```

### Task F4: spawnPortal clientbound encoder

**Files:**
- Create: `libs/atlas-packet/door/clientbound/spawn_portal.go`
- Create: `libs/atlas-packet/door/clientbound/spawn_portal_test.go`

Cosmic `spawnPortal(int townId, int targetId, Point pos)` — the minimap town↔target
portal (two ints + position). This places the minimap door indicator for the *caster*; the
party-wide indicator goes through the party packet (Task G6).

- [ ] **Step 1: Write the failing roundtrip + golden test.**
- [ ] **Step 2: Run (FAIL).**
- [ ] **Step 3: Implement `spawn_portal.go`** (`const SpawnPortalWriter = "SpawnPortal"`,
`NewSpawnPortal(townMapId, targetMapId _map.Id, x, y int16)`, version-branched `Encode`:
two ints then position).
- [ ] **Step 4: Run (PASS); commit.**

```bash
git add libs/atlas-packet/door/clientbound/spawn_portal.go libs/atlas-packet/door/clientbound/spawn_portal_test.go
git commit -m "feat(atlas-packet): spawnPortal (minimap) clientbound encoder"
```

### Task F5: Populate the reserved party door block

**Files:**
- Modify: `libs/atlas-packet/party/clientbound/created.go`
- Modify: `libs/atlas-packet/party/clientbound/created_test.go`

The door fields in `created.go` are currently hard-zeroed (`EmptyMapId,EmptyMapId,0,0`).
Make them carry real door data when present (FR-3.3 — the party minimap indicator).

- [ ] **Step 1: Write the failing test** — a `Created` built with door fields encodes the
town map id, target map id, and minimap x/y instead of zeros; a `Created` with no door
still encodes the empty sentinel. Assert across `pt.Variants`.

- [ ] **Step 2: Run (FAIL).**

- [ ] **Step 3: Add door fields to the `Created` struct + constructor** (e.g.
`doorTownMapId`, `doorTargetMapId _map.Id`, `doorX`, `doorY int16`, defaulting to
`EmptyMapId,EmptyMapId,0,0`), and write them in `Encode` in place of the hard-zeros:

```go
w.WriteInt(uint32(m.doorTownMapId))
w.WriteInt(uint32(m.doorTargetMapId))
w.WriteShort(uint16(m.doorX))
w.WriteShort(uint16(m.doorY))
```

(Keep the existing zero-default path working for callers that don't set door fields.)

- [ ] **Step 4: Run (PASS); commit.**

```bash
git add libs/atlas-packet/party/clientbound/created.go libs/atlas-packet/party/clientbound/created_test.go
git commit -m "feat(atlas-packet): carry real door fields in party-created packet"
```

### Task F6: Package vet + lib test sweep

- [ ] **Step 1: Run the full packet lib test + vet.**

Run: `cd libs/atlas-packet && go vet ./... && go test -race ./... ; cd -`
Expected: clean.

- [ ] **Step 2: Commit any fixes** (`git commit -am "fix(atlas-packet): door package vet/test cleanup"`).

---

# PART G — atlas-channel edge

Goal: route the cast to a SPAWN command, decode + validate + warp on enter-door, consume
door status events and broadcast to party-scoped same-channel viewers, spawn doors to
arriving sessions, and feed door data into the party packet. **Reuse the existing
simple-effect for the portal sound** (no new packet).

### Task G1: Door SPAWN command emission (channel side)

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/kafka/message/door/kafka.go`
- Create: `services/atlas-channel/atlas.com/channel/door/producer.go`
- Create: `services/atlas-channel/atlas.com/channel/door/requests.go`
- Create: `services/atlas-channel/atlas.com/channel/door/processor.go`
- Create: `services/atlas-channel/atlas.com/channel/door/producer_test.go`

- [ ] **Step 1: Define the command envelope (channel copy)** mirroring the atlas-doors
contract from E1 (byte-identical JSON tags) in `kafka/message/door/kafka.go`
(`EnvDoorCommandTopic = "COMMAND_TOPIC_DOOR"`, `Command[SpawnBody]`, `CommandTypeSpawn`,
`SpawnBody{SkillId, SkillLevel, X, Y}`; plus `RemoveBody`+`CommandTypeRemove`).

- [ ] **Step 2: Write the failing provider test** — `SpawnCommandProvider` keys by the
area map id and carries owner + skill + position.

- [ ] **Step 3: Implement `door/producer.go`** mirroring `portal/producer.go`:

```go
func SpawnCommandProvider(f field.Model, ownerCharacterId, skillId uint32, level byte, x, y int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(f.MapId()))
	value := door.Command[door.SpawnBody]{
		WorldId: f.WorldId(), ChannelId: f.ChannelId(), MapId: f.MapId(), Instance: f.Instance(),
		OwnerCharacterId: ownerCharacterId, Type: door.CommandTypeSpawn,
		Body: door.SpawnBody{SkillId: skillId, SkillLevel: level, X: x, Y: y},
	}
	return producer.SingleMessageProvider(key, value)
}
```

Implement `door/processor.go` with `Spawn(f, ownerCharacterId, skillId, level, x, y) error`
calling `producer.ProviderImpl(l)(ctx)(door.EnvDoorCommandTopic)(SpawnCommandProvider(...))`,
plus a REST read client (`door/requests.go` with `requests.RootUrl("DOORS")`):
`GetInField(f) ([]Model, error)`, `GetByOwnerOnMap(f, ownerCharacterId) (Model, bool)` for
the enter-door validation (G3) and map-enter spawn (G5). Add `DOORS_SERVICE_URL`/BASE
fallback per the trimmed-client convention.

- [ ] **Step 4: Run (PASS); commit.**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/message/door services/atlas-channel/atlas.com/channel/door
git commit -m "feat(atlas-channel): door command producer + atlas-doors client"
```

### Task G2: Mystic Door cast handler

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/skill/handler/mysticdoor/mysticdoor.go`
- Create: `services/atlas-channel/atlas.com/channel/skill/handler/mysticdoor/mysticdoor_test.go`
- Modify: `services/atlas-channel/atlas.com/channel/skill/handler/registrations/registrations.go`

By the time this handler runs, `UseSkill` has consumed MP + Magic Rock and skipped the
buff (no statups). The handler does the cheap channel-side eligibility rejections
(field-limit, town map, no return map) and, if eligible, emits the SPAWN command with the
caster's position. Rejections emit nothing (client already re-enabled).

- [ ] **Step 1: Write the failing test** — the handler (with seam-injected map lookup +
spawn func + caster-position func) emits a SPAWN with the caster's X/Y when the map is
eligible, and emits nothing when the map has `FieldLimitNoMysticDoor`, is a `Town`, or has
no valid return map.

```go
func TestMysticDoorEmitsSpawnWhenEligible(t *testing.T) { /* eligible map -> spawn called with c.X/c.Y */ }
func TestMysticDoorRejectsFieldLimit(t *testing.T)      { /* fieldLimit&0x02 -> no spawn */ }
func TestMysticDoorRejectsTownMap(t *testing.T)         { /* Town==true -> no spawn */ }
func TestMysticDoorRejectsNoReturn(t *testing.T)        { /* no valid return -> no spawn */ }
```

- [ ] **Step 2: Run (FAIL).**

- [ ] **Step 3: Implement `mysticdoor.go`.**

```go
package mysticdoor

import (
	"context"

	channelhandler "atlas-channel/skill/handler"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/sirupsen/logrus"
	// field, writer, packetmodel, effect imports as used by sibling handlers
)

func init() {
	channelhandler.Register(skill2.PriestMysticDoorId, Apply)
}

// seams for tests
var loadCaster = func(l logrus.FieldLogger, ctx context.Context, characterId uint32) (int16, int16, error) { /* character.GetById -> X,Y */ }
var loadMap = func(l logrus.FieldLogger, ctx context.Context, mapId _map.Id) (fieldLimit uint32, town bool, hasReturn bool, err error) { /* data/map client */ }
var emitSpawn = func(l logrus.FieldLogger, ctx context.Context, f field.Model, characterId, skillId uint32, level byte, x, y int16) error { /* door.NewProcessor(l,ctx).Spawn(...) */ }

func Apply(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer, f field.Model, characterId uint32, info packetmodel.SkillUsageInfo, e effect.Model) error {
	return func(ctx context.Context) func(wp writer.Producer, f field.Model, characterId uint32, info packetmodel.SkillUsageInfo, e effect.Model) error {
		return func(wp writer.Producer, f field.Model, characterId uint32, info packetmodel.SkillUsageInfo, e effect.Model) error {
			fieldLimit, town, hasReturn, err := loadMap(l, ctx, f.MapId())
			if err != nil { l.WithError(err).Warnf("mystic door map lookup failed"); return nil }
			if town || !hasReturn || fieldLimit&_map.FieldLimitNoMysticDoor != 0 {
				l.Debugf("mystic door rejected: town=%v hasReturn=%v limit=0x%x", town, hasReturn, fieldLimit)
				return nil // client already re-enabled by UseSkill
			}
			x, y, err := loadCaster(l, ctx, characterId)
			if err != nil { return nil }
			return emitSpawn(l, ctx, f, characterId, uint32(info.SkillId()), skillLevel(info), x, y)
		}
	}
}
```

(Resolve the cast level the same way Heal does — `skillLevel(info)` is whatever accessor
`SkillUsageInfo` exposes; mirror `heal/heal.go`.)

- [ ] **Step 4: Register the blank import** in `registrations.go`:

```go
_ "atlas-channel/skill/handler/mysticdoor" // Priest Mystic Door — task-093
```

- [ ] **Step 5: Run (PASS); build; commit.**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./skill/handler/mysticdoor/ && go build ./... ; cd -`

```bash
git add services/atlas-channel/atlas.com/channel/skill/handler/mysticdoor services/atlas-channel/atlas.com/channel/skill/handler/registrations/registrations.go
git commit -m "feat(atlas-channel): route Mystic Door cast to door SPAWN command"
```

### Task G3: enter-door inbound handler

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/mystic_door_enter.go`
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/mystic_door_enter_test.go`
- Modify: `services/atlas-channel/atlas.com/channel/main.go` (`produceHandlers`)

- [ ] **Step 1: Write the failing handler test** — decode `ownerId,direction`; with a
seam-injected door lookup + party-membership check + warp func: a valid request (requester
is owner or same-channel party member, door present on current map) calls warp to the
linked map; an ineligible requester does NOT warp.

- [ ] **Step 2: Run (FAIL).**

- [ ] **Step 3: Implement the handler** (mirror `portal_script.go`), registered with
`LoggedInValidator`:

```go
func MysticDoorEnterHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := doorsb.Enter{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		d, ok := findDoorOnMap(l, ctx, s.Field(), p.OwnerId(), s.CharacterId())
		if !ok {
			// blocked: re-enable actions (mirror existing blocked-message pattern)
			return
		}
		targetMapId, _, _ := linkedDestination(d, s.Field()) // town side -> area; area side -> town
		if err := portal.NewProcessor(l, ctx).Warp(s.Field(), s.CharacterId(), targetMapId); err != nil {
			l.WithError(err).Warnf("mystic door warp failed")
			return
		}
		playPortalSoundForSession(l, ctx, s) // reuse character simple-effect (context.md #9)
	}
}
```

`findDoorOnMap` reads atlas-doors (the G1 client), confirms a door owned by `ownerId` is
present on `s.Field()` (area field or town map) and that `s.CharacterId()` is the owner or
a current same-channel party member (channel party read). `playPortalSoundForSession`
announces the existing portal-sound simple-effect — no new packet.

- [ ] **Step 4: Register the handler** in `main.go` `produceHandlers()`:

```go
handlerMap[doorsb.EnterDoorHandle] = handler.MysticDoorEnterHandleFunc
```

(Validator binding is per-tenant config — Part H — and must be `LoggedInValidator`.)

- [ ] **Step 5: Run (PASS); build; commit.**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/mystic_door_enter.go services/atlas-channel/atlas.com/channel/socket/handler/mystic_door_enter_test.go services/atlas-channel/atlas.com/channel/main.go
git commit -m "feat(atlas-channel): enter-door inbound handler (validate + warp + portal sound)"
```

### Task G4: door status consumer → broadcast

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/kafka/consumer/door/kafka.go`
- Create: `services/atlas-channel/atlas.com/channel/kafka/consumer/door/consumer.go`
- Create: `services/atlas-channel/atlas.com/channel/socket/writer/door.go`
- Create: `services/atlas-channel/atlas.com/channel/kafka/consumer/door/consumer_test.go`
- Modify: `services/atlas-channel/atlas.com/channel/main.go` (consumer init + handler init + `produceWriters`)

- [ ] **Step 1: Define the channel copy of the door status contract** (`kafka.go`,
byte-identical to atlas-doors D1 events: `StatusEvent[CreatedBody/RemovedBody/
SlotChangedBody]`, the type + reason consts).

- [ ] **Step 2: Add channel Body wrappers** in `socket/writer/door.go`:
`SpawnDoorBody(...)`, `RemoveDoorBody(...)`, `SpawnPortalBody(...)` mapping to the
`libs/atlas-packet/door/clientbound` encoders (mirror `socket/writer/character_spawn.go`).

- [ ] **Step 3: Write the failing consumer test** — `handleCreated` broadcasts `SpawnDoor`
(area) to eligible field viewers and `SpawnDoor`+`SpawnPortal` to eligible town viewers;
`handleRemoved` broadcasts `RemoveDoor` to both maps; eligibility = caster ∪ same-channel
party members present in the map. Use the package-var broadcaster seam (mirror the mist
consumer) so the test stubs session enumeration.

- [ ] **Step 4: Run (FAIL); implement `consumer.go`** mirroring
`kafka/consumer/mist/consumer.go`: `InitConsumers`/`InitHandlers`, `sc.Is(tenant,
WorldId, ChannelId)` guard (gives per-channel FR-6.5), then resolve eligible viewers and
`Announce` the door packets. SLOT_CHANGED re-broadcasts the town door at the new portal +
updates the party packet (G6).

```go
// broadcastDoorToEligible announces `op` to sessions in `f` whose character is the owner
// or a same-channel party member of the owner (caster always included).
var broadcastDoorToEligible = func(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, f field.Model, ownerCharacterId, partyId uint32, writerName string, enc packet.Encode) {
	members := partyMemberSet(l, ctx, ownerCharacterId, partyId) // includes owner
	_ = _map.NewProcessor(l, ctx).ForSessionsInMap(f, func(s session.Model) error {
		if _, ok := members[s.CharacterId()]; !ok { return nil }
		return session.Announce(l)(ctx)(wp)(writerName)(enc)(s)
	})
}
```

- [ ] **Step 5: Register** in `main.go`: the writer-name consts in `produceWriters()`
(`doorcb.SpawnDoorWriter`, `doorcb.RemoveDoorWriter`, `doorcb.SpawnPortalWriter`); the
consumer in the consumer-init block (`doorConsumer.InitConsumers(l)(cmf)(consumerGroupId)`)
and handler-init block (`register(doorConsumer.InitHandlers(fl)(sc)(wp)(rh))`); the import
alias.

- [ ] **Step 6: Run (PASS); build; commit.**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/consumer/door services/atlas-channel/atlas.com/channel/socket/writer/door.go services/atlas-channel/atlas.com/channel/main.go
git commit -m "feat(atlas-channel): door status consumer broadcasts spawn/remove to party-scoped viewers"
```

### Task G5: Map-enter spawn (FR-3.4)

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go` (SpawnForSelf block ~189-211)
- Create the `spawnDoorsForSession` operator alongside `spawnReactorsForSession`

- [ ] **Step 1: Write a failing test** of `spawnDoorsForSession`: a door whose owner is
the entering session's character (or a same-channel party member) yields a `SpawnDoor`
announce to that session; an ineligible door yields nothing.

- [ ] **Step 2: Run (FAIL); implement.** Add to the SpawnForSelf block:

```go
go func() {
	door.NewProcessor(l, ctx).ForEachInMap(f, spawnDoorsForSession(l)(ctx)(wp)(s))
}()
```

`spawnDoorsForSession` filters to doors the session may see (owner is self or a
same-channel party member of the owner) and announces the area `SpawnDoor` (or, in town,
`SpawnDoor`+`SpawnPortal`). Reuse the eligibility helper from G4. (Add a `ForEachInMap` to
the channel door client that reads `GetInField`.)

- [ ] **Step 3: Run (PASS); build; commit.**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go services/atlas-channel/atlas.com/channel/door
git commit -m "feat(atlas-channel): spawn eligible doors to arriving sessions"
```

### Task G6: party minimap door indicator (the long-standing TODO)

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/party_operation.go` (or wherever the channel builds the party-created/update packet)
- Modify: `docs/TODO.md` (update the stale entry at line ~146)

- [ ] **Step 1: Wire live door data into the party-created packet.** When the channel
emits a party-created/update for a member who owns a live door (read atlas-doors by owner),
populate the new door fields added in F5 (town map id, target map id, minimap x/y) instead
of the empty sentinel. Send to party members so the door shows on their minimap (FR-3.3).

- [ ] **Step 2: Update the stale TODO.** In `docs/TODO.md`, change the
`Write doors for party` entry (~line 146) to reflect the implemented wiring (point at
`libs/atlas-packet/party/clientbound/created.go` + this channel handler) and mark it
done/remove it per the /dev-docs format.

- [ ] **Step 3: Build; commit.**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/party_operation.go docs/TODO.md
git commit -m "feat(atlas-channel): party minimap door indicator (resolves doors-for-party TODO)"
```

### Task G7: channel build + test sweep

- [ ] **Step 1: Run.**

Run: `cd services/atlas-channel/atlas.com/channel && go vet ./... && go test -race ./... && go build ./... ; cd -`
Expected: clean.

- [ ] **Step 2: Bake the channel image** (its go.mod was touched).

Run: `docker buildx bake atlas-channel`
Expected: builds.

- [ ] **Step 3: Commit any fixes.**

---

# PART H — Per-version opcode wiring & verification matrix

Goal: make the four door packets (`SpawnDoor`, `RemoveDoor`, `SpawnPortal` writers +
`EnterDoor` handler) live on **every** supported version by adding tenant socket-template
opcode rows (with validators) and patching live tenant config — each opcode + byte layout
**verified against IDA**, never guessed.

> **This is the "version done" gate.** Versions: `gms_v83`, `gms_v84`, `gms_v87`,
> `gms_v92`, `gms_v95`, `jms_v185`. Per FR-7.4 and the memory rules, **an op whose fname
> does not resolve in the IDA export is a STOP-AND-ESCALATE** — do not auto-substitute an
> fname, re-export, or fake a hash. Park that version's opcode (like the v92 mount-food
> handler) and surface it to the user.

### Task H1: Locate the socket templates and the opcode-config shape

- [ ] **Step 1: Find the tenant socket templates** (seed configs) and the live-config
shape. Search the repo for where existing handler/writer opcode rows live for a known
packet (e.g. `PortalScriptHandle`, `CharacterSpawn`) across versions — those are the files
Part H edits. Document the exact path(s) and the row schema (writer name → opcode; handler
name → opcode + `validator`). (Per-version templates + live config must both be patched;
projection does not hot-reload handlers/writers.)

- [ ] **Step 2: Write down the matrix** as a checklist: 4 packets × 6 versions = 24 cells,
each needing (a) opcode resolved from IDA, (b) byte layout confirmed, (c) template row
added (handler rows include `LoggedInValidator`), (d) golden test, (e) live-config patch.

### Task H2: gms_v83 opcodes + golden bytes (baseline)

- [ ] **Step 1: Resolve the four v83 opcodes from the v83 IDA export** (ida-pro-mcp v83
instance; fname→opcode per `reference_packet_audit_tool_mechanics` /
`reference_ida_mcp_new_api`). If any fname does not resolve, STOP and escalate.

- [ ] **Step 2: Confirm the v83 byte layout** for each packet against the decompile and
finalize the `libs/atlas-packet/door` encoders' v83 path (adjust F2-F4 if IDA differs from
the Cosmic-derived layout — IDA is the source of truth).

- [ ] **Step 3: Add the v83 template rows** (3 writers + 1 handler with `LoggedInValidator`).

- [ ] **Step 4: Add/confirm the v83 golden-byte tests** in the packet lib (the
`// packet-audit:verify packet=… version=gms_v83 ida=0x…` comment lines).

- [ ] **Step 5: Commit.**

```bash
git add libs/atlas-packet/door <template files>
git commit -m "feat(task-093): gms_v83 door opcodes + verified golden bytes"
```

### Task H3: gms_v84 opcodes

- [ ] **Step 1: Resolve v84 opcodes from the v84/v83 IDA** — the opcode TABLE shifts vs v83
above ~0x3D (bug `v84_opcode_table_shifted_vs_v83`); structure is v83-identical
(off-by-one) but opcode values may differ. Verify each; escalate unresolved fnames.

- [ ] **Step 2: Add v84 template rows + golden tests** (structure == v83; assert
`bytes.Equal` to the v83 encoding). Commit.

### Task H4: gms_v87 opcodes

- [ ] **Step 1: Resolve v87 opcodes from the v87 IDA.** v87 is where structure may diverge
(`MajorAtLeast(87)` branch). If the door packets diverge structurally, add the branch in
F2-F4 + a v87 golden test; else assert v87 stays on the v87 path. Note bug
`v87_template_missing_core_opcodes` — v87 templates have known gaps; do not assume an
existing row. Escalate unresolved fnames.

- [ ] **Step 2: Add v87 template rows + golden tests. Commit.**

### Task H5: gms_v92 + gms_v95 opcodes

- [ ] **Step 1: Resolve v95 opcodes from the v95 IDA; resolve v92 opcodes** against
whatever v92 IDB is available — if none resolves an opcode, park that version per the
escalation rule.

- [ ] **Step 2: Add v92 + v95 template rows + golden tests. Commit.**

### Task H6: jms_v185 opcodes

- [ ] **Step 1: Resolve jms_v185 opcodes from the jms IDA** (audit-dir caveat:
`reference_packet_audit_jms_dirname_mismatch` — pass the explicit audit dir; the jms IDA
is a separate instance). jms diverges from gms — confirm the byte layout independently.
Escalate unresolved fnames.

- [ ] **Step 2: Add jms_v185 template rows + golden tests. Commit.**

### Task H7: Live tenant config patch

- [ ] **Step 1: For each existing live tenant**, patch the live socket config to add the 4
door opcode rows (3 writers + 1 handler with `LoggedInValidator`) — existing tenants do
NOT auto-receive new opcodes (bug `new_opcodes_not_in_live_tenant_config`). Document the
patch + the channel restart requirement (projection does not hot-reload handlers/writers).
Capture this as a deploy runbook step in the PR description (or commit it if live config is
repo-managed).

---

# PART I — Final verification & branch finish

### Task I1: Full multi-module verification

- [ ] **Step 1: atlas-doors.**

Run: `cd services/atlas-doors/atlas.com/doors && go test -race ./... && go vet ./... && go build ./... ; cd -`
Expected: all clean.

- [ ] **Step 2: atlas-channel.**

Run: `cd services/atlas-channel/atlas.com/channel && go test -race ./... && go vet ./... && go build ./... ; cd -`
Expected: all clean.

- [ ] **Step 3: libs/atlas-packet.**

Run: `cd libs/atlas-packet && go test -race ./... && go vet ./... && go build ./... ; cd -`
Expected: all clean.

- [ ] **Step 4: rediskeyguard.**

Run: `GOWORK=off tools/redis-key-guard.sh`
Expected: clean.

- [ ] **Step 5: Bake every touched service** (go.mod touched: atlas-doors + atlas-channel).

Run: `docker buildx bake atlas-doors && docker buildx bake atlas-channel`
Expected: both images build.

- [ ] **Step 6: kustomize.**

Run: `kubectl kustomize deploy/k8s/base >/dev/null && echo OK`
Expected: `OK`.

### Task I2: Acceptance-criteria walkthrough + review

- [ ] **Step 1: Tick the PRD §10 acceptance criteria** against the implementation, noting
runtime-only (live-client) items vs unit-covered, and the per-version matrix status (fully
verified vs parked-and-escalated).

- [ ] **Step 2: Code review.** Run `superpowers:requesting-code-review` (dispatches
`plan-adherence-reviewer` + `backend-guidelines-reviewer`; the latter enforces DOM-21
reuse of `libs/atlas-constants`). Address findings. **Do not open the PR before the
review** (CLAUDE.md "Code Review Before PR").

- [ ] **Step 3: Finish the branch** via `superpowers:finishing-a-development-branch` only
after every Build & Verification gate above is green.

---

## Spec-coverage map (self-review)

| PRD requirement | Task(s) |
|---|---|
| FR-1.1 route 2311002 to spawn | G2 |
| FR-1.2 cast rejections (field-limit/town/no-return/no-slot) | G2 (channel pre-checks), D3/resolver (engine re-check) |
| FR-1.3 MP + Magic Rock consume (data-derived, no double) | Existing `UseSkill` (context.md #7); C2 effect client |
| FR-1.4 recast replaces prior door | D3 `Spawn` recast |
| FR-2.1 paired area+town door | B1 model, D3 |
| FR-2.2 shared object-id pool (2 oids) | B2, D3 |
| FR-2.3 return town from atlas-data | C1, C5, resolver |
| FR-3.1/3.2 field+town visibility to party | G4, G5 |
| FR-3.3 spawnPortal + party minimap indicator | F4, F5, G4, G6 |
| FR-3.4 map-enter spawn | G5 |
| FR-4.1/4.2 slot = party index → 0x80+slot | C3, C4 |
| FR-4.3 re-slot on membership change | E3, D3 `Reslot` |
| FR-5.1 enter-door decode | F1 |
| FR-5.2 ownership/party validation | G3 |
| FR-5.3 warp + portal sound | G3 (warp via portal.Warp; sound via existing simple-effect) |
| FR-5.4 reject mid-map-change/banned | G3 |
| FR-6.1 expiry removal + broadcast | E4, G4 |
| FR-6.2 removal on logout/channel/left-field | E2, D3 `RemoveByOwnerIfLeftField` |
| FR-6.3 deploy grace | E4 |
| FR-6.4 join/leave/disband re-slot behavior | E3 |
| FR-6.5 per-channel | E2 (channel-changed removal), G4 (`sc.Is` channel guard) |
| FR-6.6 ephemeral (Redis, no relational) | B3 |
| FR-7.1 all packets all versions | F, H |
| FR-7.2 per-version opcodes + live config | H |
| FR-7.3 handler validator present | H (LoggedInValidator rows) |
| FR-7.4 bytes verified per version (no v83 assumption) | H (IDA), F golden tests |
| §5.1 REST GET door + in-field | D4 |
| §5.1 Kafka command/event topics | D1, E1 |
| §10 build/bake/vet/test/rediskeyguard clean | I1 |
| §10 service registration (services.json/bake/go.work/Dockerfile/k8s) | A4, A5 (Dockerfile: no edit — context.md) |

## Notes carried for the executor

- **Mirror `atlas-monsters`** for all engine boilerplate (registry, allocator, leader,
  kafka, tasks) — `atlas-summons` is NOT in this branch.
- **Opcodes are tenant config, not Go.** Go branches on structure only.
- **`playPortalSound` is the existing character simple-effect** — do not add a packet.
- **Per-version door bytes/opcodes: IDA-verify or escalate.** Never guess; never fake a
  hash; park a version whose fname won't resolve (like v92 mount-food).
- **Confirm Cosmic byte order** for `spawnDoor`/`removeDoor`/`spawnPortal`/enter-door
  against `~/source/Cosmic/.../PacketCreator.java` + `DoorHandler.java` before finalizing
  the F-task encoders; IDA (Part H) is the final arbiter.

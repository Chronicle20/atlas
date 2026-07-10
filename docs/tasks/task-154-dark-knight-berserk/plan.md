# Dark Knight Berserk (1320006) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** atlas-buffs tracks Dark Knights with Berserk leveled, re-evaluates `hp*100/effectiveMaxHp < x(level)` on the design's triggers, and drives a Cosmic-parity 5s-initial/3s-period broadcast by emitting one `BERSERK` event per tick on `EVENT_TOPIC_CHARACTER_BUFF_STATUS`; atlas-channel statelessly translates each event into own + foreign `EffectSkillUse` packets.

**Architecture:** New `berserk/` domain package in atlas-buffs with a Redis-backed `TenantRegistry` (namespace `buffs-berserk`), a 1s scan ticker with atomic per-entry claims (replica-safe via `TenantRegistry.Update`'s WATCH/MULTI), consumers that only mark registry state, and a ticker that does all REST I/O and emission. atlas-channel adds one handler to its existing buff consumer plus two `Announce*` helpers. Zero k8s manifest changes; zero packet/writer/template changes.

**Tech Stack:** Go, `libs/atlas-redis` TenantRegistry + Set, `libs/atlas-kafka` (message.Buffer/Emit, curried consumers), `libs/atlas-rest/requests` REST clients, `libs/atlas-constants` (skill/stat/world/channel/character), miniredis + producertest for tests.

## Global Constraints

- Worktree: all work happens in `.worktrees/task-154-dark-knight-berserk` on branch `task-154-dark-knight-berserk`. Every subagent must `cd` there first and verify `git branch --show-current` = `task-154-dark-knight-berserk` after each commit.
- No numeric skill-id literals outside `libs/atlas-constants` — use `skill.DarkKnightBerserkId` (`libs/atlas-constants/skill/constants.go:3006`). No hard-coded thresholds; `x` comes from atlas-data at runtime.
- Strict less-than comparison: `hp*100/effectiveMaxHp < x`. Equality is INACTIVE.
- All registry state and REST lookups tenant-scoped (`tenant.MustFromContext(ctx)`); Redis access only through `libs/atlas-redis` types (`tools/redis-key-guard.sh` must stay clean).
- Immutable models: private fields + getters + Builder. No `*_testhelpers.go` files; table-driven tests.
- Consumers never panic on missing data; failed lookups log-and-rearm, never crash or freeze on stale state.
- Cadence constants are named: initial delay 5s, period 3s, re-eval grace 2s, retry re-arm 1s, scan interval 1000ms.
- Commit after every task with prefix `feat(task-154): ...` (tests-only tasks may use `test(task-154): ...`).
- Never write literal home/absolute paths into committed files.
- Do not `go work sync`. Run `go` commands from the module dir (`services/atlas-buffs/atlas.com/buffs`, `services/atlas-channel/atlas.com/channel`).

## File Structure

atlas-buffs (`services/atlas-buffs/atlas.com/buffs/`), module `atlas-buffs`:

| File | Responsibility | Task |
|---|---|---|
| `berserk/model.go` | Immutable entry model + JSON round-trip + functional mutators | 1 |
| `berserk/builder.go` | Builder | 1 |
| `berserk/evaluate.go` | Pure FR-1 computation | 2 |
| `berserk/registry.go` | TenantRegistry wrapper: Track/Untrack/MarkDirty/UpdateChannel/UpdateSkillLevel/ClaimReeval/ClaimBroadcast/StoreEvaluation/GetAll/GetTenants | 3 |
| `external/character/`, `external/skills/`, `external/effectivestats/`, `external/dataskill/` | Read-only REST clients (requests.go + rest.go each) | 4 |
| `berserk/cache.go` | Per-tenant effect-`x` cache | 4 |
| `kafka/message/character/kafka.go` | + `EventStatusTypeBerserk` + `BerserkStatusEventBody` | 5 |
| `berserk/producer.go` | BERSERK event provider | 5 |
| `berserk/processor.go` | Processor: TrackOnLogin/Untrack/HandleStatChanged/HandleTransfer/HandleSkillUpdated/MarkMaxHpDirty/ProcessTicks + fan-out | 6 |
| `tasks/berserk.go` | 1s ticker (poison.go shape) | 6 |
| `kafka/message/characterstatus/kafka.go` | Local mirror of atlas-character status envelope + LOGIN/LOGOUT/STAT_CHANGED/MAP_CHANGED/CHANNEL_CHANGED bodies | 7 |
| `kafka/message/skillstatus/kafka.go` | Local mirror of atlas-skills status envelope + UPDATED/DELETED bodies | 7 |
| `kafka/consumer/characterstatus/consumer.go` | Character status consumer → registry marks | 7 |
| `kafka/consumer/skillstatus/consumer.go` | Skill status consumer → track/untrack/level updates | 7 |
| `main.go` | + berserk.InitRegistry, + BerserkTick, + 2 consumer registrations | 6, 7 |
| `character/maxhp.go` | `affectsMaxHp` + berserk dirty hook | 8 |
| `character/processor.go` | Hook calls in Apply/Cancel/CancelAll/CancelByStatTypes/ExpireBuffs | 8 |

atlas-channel (`services/atlas-channel/atlas.com/channel/`), module `atlas-channel`:

| File | Responsibility | Task |
|---|---|---|
| `kafka/message/buff/kafka.go` | + `EventStatusTypeBerserk` + `BerserkStatusEventBody` mirror | 9 |
| `kafka/message/buff/kafka_test.go` | Golden-JSON cross-service contract test | 9 |
| `socket/handler/effects.go` | + `AnnounceBerserkEffect` + `AnnounceForeignBerserkEffect` | 9 |
| `kafka/consumer/buff/consumer.go` | + `handleStatusEventBerserk` + registration | 9 |

Docs: `docs/tasks/task-154-dark-knight-berserk/context.md` (Task 10).

**Interfaces spine (types used across tasks):**

- `berserk.Model` getters: `WorldId() world.Id`, `ChannelId() channel.Id`, `ChannelKnown() bool`, `CharacterId() uint32`, `CharacterLevel() byte`, `SkillLevel() byte`, `Active() bool`, `DirtyAt() time.Time`, `NextBroadcastAt() time.Time`, `DirtyDue(now time.Time) bool`, `BroadcastDue(now time.Time) bool`.
- `berserk.Evaluate(skillLevel byte, hp uint16, effectiveMaxHp uint32, x int16) bool`.
- Exported constants: `berserk.InitialBroadcastDelay = 5 * time.Second`, `berserk.BroadcastPeriod = 3 * time.Second`, `berserk.ReevalGrace = 2 * time.Second`, `berserk.ReevalRetryDelay = time.Second`.
- Registry: `GetRegistry()` after `InitRegistry(client *goredis.Client)`; methods listed per Task 3.
- Processor: `berserk.NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor` with the interface in Task 6.
- Event body (both services, identical JSON): `BerserkStatusEventBody{TransactionId uuid.UUID, ChannelId channel.Id, SkillId uint32, CharacterLevel byte, SkillLevel byte, Active bool}` under envelope `StatusEvent[E]{WorldId world.Id, CharacterId uint32, Type string, Body E}`, `Type = "BERSERK"`, topic env `EVENT_TOPIC_CHARACTER_BUFF_STATUS`.

---

### Task 1: berserk entry model + builder

**Files:**
- Create: `services/atlas-buffs/atlas.com/buffs/berserk/model.go`
- Create: `services/atlas-buffs/atlas.com/buffs/berserk/builder.go`
- Test: `services/atlas-buffs/atlas.com/buffs/berserk/model_test.go`

**Interfaces:**
- Consumes: `world.Id`, `channel.Id` from `libs/atlas-constants`.
- Produces: `Model` (getters above), `NewBuilder(worldId world.Id, characterId uint32, skillLevel byte) *Builder` with `SetChannel(channel.Id)`, `SetCharacterLevel(byte)`, `SetDirtyAt(time.Time)`, `Build() Model`; package-private functional mutators `channelUpdated`, `skillLevelUpdated`, `dirtyMarked`, `dirtyCleared`, `evaluated`, `broadcastScheduled` (used by Task 3's registry closures); JSON round-trip via MarshalJSON/UnmarshalJSON (required because fields are private and the value is stored in Redis — same pattern as `character/model.go:47-76`).

- [ ] **Step 1: Write the failing test**

`berserk/model_test.go`:

```go
package berserk

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/stretchr/testify/assert"
)

func TestModelJSONRoundTrip(t *testing.T) {
	dirty := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	next := dirty.Add(5 * time.Second)
	m := NewBuilder(world.Id(1), 42, 10).
		SetChannel(channel.Id(2)).
		SetCharacterLevel(120).
		SetDirtyAt(dirty).
		Build()
	m = m.evaluated(true, 121, next)

	data, err := json.Marshal(m)
	assert.NoError(t, err)

	var got Model
	assert.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, world.Id(1), got.WorldId())
	assert.Equal(t, channel.Id(2), got.ChannelId())
	assert.True(t, got.ChannelKnown())
	assert.Equal(t, uint32(42), got.CharacterId())
	assert.Equal(t, byte(121), got.CharacterLevel())
	assert.Equal(t, byte(10), got.SkillLevel())
	assert.True(t, got.Active())
	assert.True(t, got.DirtyAt().Equal(dirty))
	assert.True(t, got.NextBroadcastAt().Equal(next))
}

func TestBuilderDefaults(t *testing.T) {
	m := NewBuilder(world.Id(0), 7, 1).Build()
	assert.False(t, m.ChannelKnown(), "channel unknown until a channel-bearing event")
	assert.False(t, m.Active())
	assert.True(t, m.DirtyAt().IsZero())
	assert.True(t, m.NextBroadcastAt().IsZero())
}

func TestMutators(t *testing.T) {
	now := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	m := NewBuilder(world.Id(0), 7, 1).Build()

	m2 := m.channelUpdated(world.Id(1), channel.Id(3))
	assert.True(t, m2.ChannelKnown())
	assert.Equal(t, channel.Id(3), m2.ChannelId())
	assert.False(t, m.ChannelKnown(), "original unchanged (immutability)")

	m3 := m2.dirtyMarked(now)
	assert.True(t, m3.DirtyAt().Equal(now))
	m4 := m3.dirtyCleared()
	assert.True(t, m4.DirtyAt().IsZero())

	m5 := m4.skillLevelUpdated(20)
	assert.Equal(t, byte(20), m5.SkillLevel())

	m6 := m5.evaluated(true, 130, now.Add(5*time.Second))
	assert.True(t, m6.Active())
	assert.Equal(t, byte(130), m6.CharacterLevel())

	m7 := m6.broadcastScheduled(now.Add(3 * time.Second))
	assert.True(t, m7.NextBroadcastAt().Equal(now.Add(3*time.Second)))
}

func TestDueHelpers(t *testing.T) {
	now := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	m := NewBuilder(world.Id(0), 7, 1).SetChannel(channel.Id(1)).Build()

	assert.False(t, m.DirtyDue(now), "zero dirtyAt = clean")
	assert.False(t, m.BroadcastDue(now), "zero nextBroadcastAt = not scheduled yet")

	assert.True(t, m.dirtyMarked(now).DirtyDue(now), "dirtyAt == now is due")
	assert.False(t, m.dirtyMarked(now.Add(time.Second)).DirtyDue(now), "future dirtyAt (grace) not due")

	sched := m.broadcastScheduled(now)
	assert.True(t, sched.BroadcastDue(now))

	unknown := NewBuilder(world.Id(0), 8, 1).Build().dirtyMarked(now).broadcastScheduled(now)
	assert.False(t, unknown.DirtyDue(now), "re-eval needs channelKnown (effective-stats route needs channel)")
	assert.False(t, unknown.BroadcastDue(now), "broadcast needs channelKnown (cannot route)")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run (from `services/atlas-buffs/atlas.com/buffs`): `go test ./berserk/... -run 'TestModel|TestBuilder|TestMutators|TestDue' -v`
Expected: FAIL — package does not exist / undefined identifiers.

- [ ] **Step 3: Write the implementation**

`berserk/model.go`:

```go
package berserk

import (
	"encoding/json"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// Broadcast cadence (Cosmic parity: Character.java:1867 — 5000ms delay, 3000ms
// period) plus service-local pacing knobs. Exported so the character package's
// buff hook and tests reference the same values.
const (
	InitialBroadcastDelay = 5 * time.Second
	BroadcastPeriod       = 3 * time.Second
	// ReevalGrace defers buff-origin re-evaluations so atlas-effective-stats can
	// consume the buff event and recompute max HP before we read it (design D5).
	ReevalGrace = 2 * time.Second
	// ReevalRetryDelay re-arms dirtyAt after a failed lookup so the re-evaluation
	// retries instead of silently freezing on stale state (design §4.1).
	ReevalRetryDelay = time.Second
)

// Model is one tracked Dark Knight. channelId is meaningless until
// channelKnown; entries created from skill UPDATED events (which carry no
// channel) stay unroutable until the next channel-bearing character event.
type Model struct {
	worldId         world.Id
	channelId       channel.Id
	channelKnown    bool
	characterId     uint32
	characterLevel  byte
	skillLevel      byte
	active          bool
	dirtyAt         time.Time
	nextBroadcastAt time.Time
}

func (m Model) WorldId() world.Id          { return m.worldId }
func (m Model) ChannelId() channel.Id      { return m.channelId }
func (m Model) ChannelKnown() bool         { return m.channelKnown }
func (m Model) CharacterId() uint32        { return m.characterId }
func (m Model) CharacterLevel() byte       { return m.characterLevel }
func (m Model) SkillLevel() byte           { return m.skillLevel }
func (m Model) Active() bool               { return m.active }
func (m Model) DirtyAt() time.Time         { return m.dirtyAt }
func (m Model) NextBroadcastAt() time.Time { return m.nextBroadcastAt }

// DirtyDue reports whether a re-evaluation is due. Requires channelKnown
// because the effective-stats route needs world/channel to resolve max HP.
func (m Model) DirtyDue(now time.Time) bool {
	return m.channelKnown && !m.dirtyAt.IsZero() && !m.dirtyAt.After(now)
}

// BroadcastDue reports whether a broadcast tick is due. Zero nextBroadcastAt
// means no evaluation has completed yet — nothing to broadcast.
func (m Model) BroadcastDue(now time.Time) bool {
	return m.channelKnown && !m.nextBroadcastAt.IsZero() && !m.nextBroadcastAt.After(now)
}

func (m Model) channelUpdated(worldId world.Id, channelId channel.Id) Model {
	m.worldId = worldId
	m.channelId = channelId
	m.channelKnown = true
	return m
}

func (m Model) skillLevelUpdated(level byte) Model {
	m.skillLevel = level
	return m
}

func (m Model) dirtyMarked(at time.Time) Model {
	m.dirtyAt = at
	return m
}

func (m Model) dirtyCleared() Model {
	m.dirtyAt = time.Time{}
	return m
}

func (m Model) evaluated(active bool, characterLevel byte, nextBroadcastAt time.Time) Model {
	m.active = active
	m.characterLevel = characterLevel
	m.nextBroadcastAt = nextBroadcastAt
	return m
}

func (m Model) broadcastScheduled(next time.Time) Model {
	m.nextBroadcastAt = next
	return m
}

type modelJSON struct {
	WorldId         world.Id   `json:"worldId"`
	ChannelId       channel.Id `json:"channelId"`
	ChannelKnown    bool       `json:"channelKnown"`
	CharacterId     uint32     `json:"characterId"`
	CharacterLevel  byte       `json:"characterLevel"`
	SkillLevel      byte       `json:"skillLevel"`
	Active          bool       `json:"active"`
	DirtyAt         time.Time  `json:"dirtyAt"`
	NextBroadcastAt time.Time  `json:"nextBroadcastAt"`
}

func (m Model) MarshalJSON() ([]byte, error) {
	return json.Marshal(modelJSON{
		WorldId:         m.worldId,
		ChannelId:       m.channelId,
		ChannelKnown:    m.channelKnown,
		CharacterId:     m.characterId,
		CharacterLevel:  m.characterLevel,
		SkillLevel:      m.skillLevel,
		Active:          m.active,
		DirtyAt:         m.dirtyAt,
		NextBroadcastAt: m.nextBroadcastAt,
	})
}

func (m *Model) UnmarshalJSON(data []byte) error {
	var aux modelJSON
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	m.worldId = aux.WorldId
	m.channelId = aux.ChannelId
	m.channelKnown = aux.ChannelKnown
	m.characterId = aux.CharacterId
	m.characterLevel = aux.CharacterLevel
	m.skillLevel = aux.SkillLevel
	m.active = aux.Active
	m.dirtyAt = aux.DirtyAt
	m.nextBroadcastAt = aux.NextBroadcastAt
	return nil
}
```

`berserk/builder.go`:

```go
package berserk

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

type Builder struct {
	worldId        world.Id
	channelId      channel.Id
	channelKnown   bool
	characterId    uint32
	characterLevel byte
	skillLevel     byte
	dirtyAt        time.Time
}

func NewBuilder(worldId world.Id, characterId uint32, skillLevel byte) *Builder {
	return &Builder{
		worldId:     worldId,
		characterId: characterId,
		skillLevel:  skillLevel,
	}
}

func (b *Builder) SetChannel(channelId channel.Id) *Builder {
	b.channelId = channelId
	b.channelKnown = true
	return b
}

func (b *Builder) SetCharacterLevel(level byte) *Builder {
	b.characterLevel = level
	return b
}

func (b *Builder) SetDirtyAt(at time.Time) *Builder {
	b.dirtyAt = at
	return b
}

func (b *Builder) Build() Model {
	return Model{
		worldId:        b.worldId,
		channelId:      b.channelId,
		channelKnown:   b.channelKnown,
		characterId:    b.characterId,
		characterLevel: b.characterLevel,
		skillLevel:     b.skillLevel,
		dirtyAt:        b.dirtyAt,
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./berserk/... -v`
Expected: PASS (all four tests).

- [ ] **Step 5: Commit**

```bash
git add berserk/model.go berserk/builder.go berserk/model_test.go
git commit -m "feat(task-154): berserk entry model and builder"
```

---

### Task 2: pure Evaluate function

**Files:**
- Create: `services/atlas-buffs/atlas.com/buffs/berserk/evaluate.go`
- Test: `services/atlas-buffs/atlas.com/buffs/berserk/evaluate_test.go`

**Interfaces:**
- Produces: `Evaluate(skillLevel byte, hp uint16, effectiveMaxHp uint32, x int16) bool` — the FR-1 computation. `hp` is `uint16` (atlas-character REST `hp`), `effectiveMaxHp` is `uint32` (effective-stats `maxHP`), `x` is `int16` (atlas-data effect `x`).

- [ ] **Step 1: Write the failing test**

`berserk/evaluate_test.go`:

```go
package berserk

import "testing"

// v83 reference values (verified from local WZ, design §2): skill 1320006 has
// 30 levels, x = 21 at level 1 rising to 50 at level 30. Values here are test
// inputs only — runtime x always comes from atlas-data.
func TestEvaluate(t *testing.T) {
	cases := []struct {
		name       string
		skillLevel byte
		hp         uint16
		maxHp      uint32
		x          int16
		want       bool
	}{
		{name: "below threshold is active", skillLevel: 30, hp: 499, maxHp: 1000, x: 50, want: true},
		{name: "equality is inactive (strict less-than, Character.java:1852)", skillLevel: 30, hp: 500, maxHp: 1000, x: 50, want: false},
		{name: "above threshold is inactive", skillLevel: 30, hp: 501, maxHp: 1000, x: 50, want: false},
		{name: "integer division truncates toward inactive edge", skillLevel: 30, hp: 509, maxHp: 1020, x: 49, want: false}, // 509*100/1020 = 49
		{name: "integer division one below", skillLevel: 30, hp: 499, maxHp: 1020, x: 49, want: true}, // 499*100/1020 = 48
		{name: "skill level zero never active", skillLevel: 0, hp: 1, maxHp: 1000, x: 50, want: false},
		{name: "dead (hp=0) never active (design D7)", skillLevel: 30, hp: 0, maxHp: 1000, x: 50, want: false},
		{name: "maxHp zero guarded", skillLevel: 30, hp: 100, maxHp: 0, x: 50, want: false},
		{name: "non-positive x guarded", skillLevel: 30, hp: 1, maxHp: 1000, x: 0, want: false},
		{name: "negative x guarded", skillLevel: 30, hp: 1, maxHp: 1000, x: -1, want: false},
		{name: "hyper body raises maxHp and activates with constant hp", skillLevel: 30, hp: 600, maxHp: 1900, x: 50, want: true},   // 600*100/1900 = 31
		{name: "hyper body expiry deactivates with constant hp", skillLevel: 30, hp: 600, maxHp: 1000, x: 50, want: false},          // 60
		{name: "max uint16 hp does not overflow uint32 math", skillLevel: 30, hp: 65535, maxHp: 99999, x: 50, want: false},          // 65535*100 = 6,553,500 < 2^32
		{name: "level 1 threshold", skillLevel: 1, hp: 209, maxHp: 1000, x: 21, want: true},
		{name: "level 1 threshold boundary", skillLevel: 1, hp: 210, maxHp: 1000, x: 21, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Evaluate(tc.skillLevel, tc.hp, tc.maxHp, tc.x); got != tc.want {
				t.Errorf("Evaluate(%d, %d, %d, %d) = %v, want %v", tc.skillLevel, tc.hp, tc.maxHp, tc.x, got, tc.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./berserk/... -run TestEvaluate -v`
Expected: FAIL — `undefined: Evaluate`.

- [ ] **Step 3: Write the implementation**

`berserk/evaluate.go`:

```go
package berserk

// Evaluate computes the berserk-active state (design D7 / PRD FR-1):
//
//	active := skillLevel > 0 && hp > 0 && hp*100/effectiveMaxHp < x
//
// Strict less-than is Cosmic parity (Character.java:1852): at exactly x% the
// aura is OFF. hp > 0 folds death handling into the formula — the
// death-accompanying STAT_CHANGED(HP=0) evaluates to inactive with no DIED
// consumer. effectiveMaxHp is buff-inclusive (atlas-effective-stats), so
// Hyper Body apply/expire can flip the state with hp constant. Integer math:
// hp is uint16 so hp*100 fits uint32 with no overflow.
func Evaluate(skillLevel byte, hp uint16, effectiveMaxHp uint32, x int16) bool {
	if skillLevel == 0 || hp == 0 || effectiveMaxHp == 0 || x <= 0 {
		return false
	}
	return uint32(hp)*100/effectiveMaxHp < uint32(x)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./berserk/... -run TestEvaluate -v`
Expected: PASS (15 subtests).

- [ ] **Step 5: Commit**

```bash
git add berserk/evaluate.go berserk/evaluate_test.go
git commit -m "feat(task-154): pure berserk state evaluation with strict-less-than boundary"
```

---

### Task 3: Redis-backed berserk registry with atomic claims

**Files:**
- Create: `services/atlas-buffs/atlas.com/buffs/berserk/registry.go`
- Create: `services/atlas-buffs/atlas.com/buffs/berserk/testmain_test.go`
- Test: `services/atlas-buffs/atlas.com/buffs/berserk/registry_test.go`

**Interfaces:**
- Consumes: `Model` + mutators (Task 1); `atlas.TenantRegistry[uint32, Model]` and `atlas.Set` from `libs/atlas-redis`. `TenantRegistry.Update` (`libs/atlas-redis/tenant_registry.go:130`) is a single-attempt WATCH/MULTI read-modify-write: on concurrent modification it returns go-redis `TxFailedErr` (no retry loop) — the claim methods treat any error as "did not win".
- Produces:
  - `InitRegistry(client *goredis.Client)`, `GetRegistry() *Registry`, `var ErrNotFound`
  - `Track(ctx, m Model) error` (also adds tenant to the shared `buffs:_tenants` set so ticker fan-out sees tenants whose only state is a Dark Knight)
  - `Untrack(ctx, characterId uint32) error`
  - `Get(ctx, characterId uint32) (Model, error)`
  - `GetAll(ctx) []Model`
  - `GetTenants(ctx) ([]tenant.Model, error)`
  - `MarkDirty(ctx, characterId uint32, at time.Time) error` — no-op on untracked (ErrNotFound → nil); last-writer-wins by design (design §5)
  - `UpdateChannel(ctx, characterId uint32, worldId world.Id, channelId channel.Id) error` — no-op on untracked
  - `UpdateSkillLevel(ctx, characterId uint32, level byte) error` — returns `ErrNotFound` when untracked (processor decides to Track)
  - `ClaimReeval(ctx, characterId uint32, now time.Time) (Model, bool)` — atomically clears dirtyAt iff due; exactly one caller wins per deadline
  - `ClaimBroadcast(ctx, characterId uint32, now time.Time) (Model, bool)` — atomically advances nextBroadcastAt by `BroadcastPeriod` iff due; returns the pre-advance state to emit
  - `StoreEvaluation(ctx, characterId uint32, active bool, characterLevel byte, nextBroadcastAt time.Time) error`

- [ ] **Step 1: Write the failing test**

`berserk/testmain_test.go` (same shape as `character/testmain_test.go`):

```go
package berserk

import (
	"os"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer/producertest"
)

func TestMain(m *testing.M) {
	producertest.InstallNoop()
	os.Exit(m.Run())
}
```

`berserk/registry_test.go`:

```go
package berserk

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func setupTestRegistry(t *testing.T) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRegistry(client)
}

func setupTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}
	return ten
}

func setupTestContext(t *testing.T, ten tenant.Model) context.Context {
	t.Helper()
	return tenant.WithContext(context.Background(), ten)
}

func trackedModel(characterId uint32) Model {
	return NewBuilder(world.Id(0), characterId, 10).SetChannel(channel.Id(1)).Build()
}

func TestTrackUntrackLifecycle(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))

	assert.NoError(t, GetRegistry().Track(ctx, trackedModel(42)))

	got, err := GetRegistry().Get(ctx, 42)
	assert.NoError(t, err)
	assert.Equal(t, uint32(42), got.CharacterId())

	all := GetRegistry().GetAll(ctx)
	assert.Len(t, all, 1)

	tenants, err := GetRegistry().GetTenants(ctx)
	assert.NoError(t, err)
	assert.Len(t, tenants, 1, "Track must register the tenant for ticker fan-out")

	assert.NoError(t, GetRegistry().Untrack(ctx, 42))
	_, err = GetRegistry().Get(ctx, 42)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestMarkDirtyAndUpdateChannelIgnoreUntracked(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))

	assert.NoError(t, GetRegistry().MarkDirty(ctx, 99, time.Now()))
	assert.NoError(t, GetRegistry().UpdateChannel(ctx, 99, world.Id(0), channel.Id(1)))
	assert.ErrorIs(t, GetRegistry().UpdateSkillLevel(ctx, 99, 5), ErrNotFound)
}

func TestClaimReeval(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))
	now := time.Now()

	assert.NoError(t, GetRegistry().Track(ctx, trackedModel(42)))

	_, ok := GetRegistry().ClaimReeval(ctx, 42, now)
	assert.False(t, ok, "clean entry is not claimable")

	assert.NoError(t, GetRegistry().MarkDirty(ctx, 42, now.Add(time.Second)))
	_, ok = GetRegistry().ClaimReeval(ctx, 42, now)
	assert.False(t, ok, "grace-deferred dirty not claimable early")

	assert.NoError(t, GetRegistry().MarkDirty(ctx, 42, now))
	m, ok := GetRegistry().ClaimReeval(ctx, 42, now)
	assert.True(t, ok)
	assert.True(t, m.DirtyAt().IsZero(), "claim clears dirtyAt")

	_, ok = GetRegistry().ClaimReeval(ctx, 42, now)
	assert.False(t, ok, "second claim on same deadline loses")
}

func TestClaimReevalRequiresChannel(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))
	now := time.Now()

	// channelKnown=false (skill-UPDATED-created entry): dirty but unroutable.
	m := NewBuilder(world.Id(0), 7, 10).SetDirtyAt(now).Build()
	assert.NoError(t, GetRegistry().Track(ctx, m))

	_, ok := GetRegistry().ClaimReeval(ctx, 7, now)
	assert.False(t, ok, "re-eval needs channelKnown for the effective-stats route")

	assert.NoError(t, GetRegistry().UpdateChannel(ctx, 7, world.Id(0), channel.Id(2)))
	_, ok = GetRegistry().ClaimReeval(ctx, 7, now)
	assert.True(t, ok, "dirtyAt survives until channel is known, then claims")
}

func TestClaimBroadcast(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))
	now := time.Now()

	assert.NoError(t, GetRegistry().Track(ctx, trackedModel(42)))
	_, ok := GetRegistry().ClaimBroadcast(ctx, 42, now)
	assert.False(t, ok, "no broadcast before first evaluation")

	assert.NoError(t, GetRegistry().StoreEvaluation(ctx, 42, true, 120, now))
	m, ok := GetRegistry().ClaimBroadcast(ctx, 42, now)
	assert.True(t, ok)
	assert.True(t, m.Active())
	assert.Equal(t, byte(120), m.CharacterLevel())

	_, ok = GetRegistry().ClaimBroadcast(ctx, 42, now)
	assert.False(t, ok, "claim advanced the deadline by BroadcastPeriod")

	stored, err := GetRegistry().Get(ctx, 42)
	assert.NoError(t, err)
	assert.True(t, stored.NextBroadcastAt().Equal(now.Add(BroadcastPeriod)))

	_, ok = GetRegistry().ClaimBroadcast(ctx, 42, now.Add(BroadcastPeriod))
	assert.True(t, ok, "due again one period later")
}

// TestConcurrentClaimSingleWinner is the cancel-reschedule race from the PRD's
// acceptance criteria: when two replicas scan the same due entry, exactly one
// claim wins.
func TestConcurrentClaimSingleWinner(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))
	now := time.Now()

	assert.NoError(t, GetRegistry().Track(ctx, trackedModel(42)))
	assert.NoError(t, GetRegistry().StoreEvaluation(ctx, 42, true, 120, now))

	const attempts = 8
	wins := make(chan bool, attempts)
	var wg sync.WaitGroup
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, ok := GetRegistry().ClaimBroadcast(ctx, 42, now)
			wins <- ok
		}()
	}
	wg.Wait()
	close(wins)

	winners := 0
	for w := range wins {
		if w {
			winners++
		}
	}
	assert.Equal(t, 1, winners, "exactly one claimant may emit per deadline")
}

func TestTenantIsolation(t *testing.T) {
	setupTestRegistry(t)
	tenA := setupTestTenant(t)
	tenB := setupTestTenant(t)
	ctxA := setupTestContext(t, tenA)
	ctxB := setupTestContext(t, tenB)

	assert.NoError(t, GetRegistry().Track(ctxA, trackedModel(42)))

	_, err := GetRegistry().Get(ctxB, 42)
	assert.ErrorIs(t, err, ErrNotFound, "same character id in another tenant is invisible")
	assert.Empty(t, GetRegistry().GetAll(ctxB))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./berserk/... -run 'TestTrack|TestMark|TestClaim|TestConcurrent|TestTenant' -v`
Expected: FAIL — `undefined: InitRegistry` etc.

- [ ] **Step 3: Write the implementation**

`berserk/registry.go`:

```go
package berserk

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	goredis "github.com/redis/go-redis/v9"
)

var ErrNotFound = errors.New("not found")

// Registry stores tracked Dark Knights in Redis (namespace buffs-berserk) so
// state is shared across the service's replicas (design D1). Tenants are
// registered in the same buffs:_tenants set the buff registry maintains, so
// ticker fan-out sees tenants whose only tracked state is a Dark Knight.
type Registry struct {
	entries *atlas.TenantRegistry[uint32, Model]
	tenants *atlas.Set
}

var registry *Registry

func InitRegistry(client *goredis.Client) {
	registry = &Registry{
		entries: atlas.NewTenantRegistry[uint32, Model](client, "buffs-berserk", func(k uint32) string {
			return strconv.FormatUint(uint64(k), 10)
		}),
		tenants: atlas.NewSet(client, "buffs:_tenants"),
	}
}

func GetRegistry() *Registry {
	return registry
}

func (r *Registry) Track(ctx context.Context, m Model) error {
	t := tenant.MustFromContext(ctx)
	if err := r.entries.Put(ctx, t, m.CharacterId(), m); err != nil {
		return err
	}
	if tb, err := json.Marshal(&t); err == nil {
		_ = r.tenants.Add(ctx, string(tb))
	}
	return nil
}

func (r *Registry) Untrack(ctx context.Context, characterId uint32) error {
	t := tenant.MustFromContext(ctx)
	return r.entries.Remove(ctx, t, characterId)
}

func (r *Registry) Get(ctx context.Context, characterId uint32) (Model, error) {
	t := tenant.MustFromContext(ctx)
	m, err := r.entries.Get(ctx, t, characterId)
	if errors.Is(err, atlas.ErrNotFound) {
		return Model{}, ErrNotFound
	}
	return m, err
}

func (r *Registry) GetAll(ctx context.Context) []Model {
	t := tenant.MustFromContext(ctx)
	vals, err := r.entries.GetAllValues(ctx, t)
	if err != nil {
		return nil
	}
	return vals
}

func (r *Registry) GetTenants(ctx context.Context) ([]tenant.Model, error) {
	members, err := r.tenants.Members(ctx)
	if err != nil {
		return nil, err
	}
	var tenants []tenant.Model
	for _, mb := range members {
		var t tenant.Model
		if err := json.Unmarshal([]byte(mb), &t); err != nil {
			continue
		}
		tenants = append(tenants, t)
	}
	return tenants, nil
}

// MarkDirty schedules a re-evaluation at/after `at`. Untracked characters are
// ignored (most characters are not Dark Knights). Last-writer-wins on dirtyAt
// is intentional: re-evaluations are idempotent and compute from current data,
// so which trigger fires one is immaterial (design §5).
func (r *Registry) MarkDirty(ctx context.Context, characterId uint32, at time.Time) error {
	t := tenant.MustFromContext(ctx)
	_, err := r.entries.Update(ctx, t, characterId, func(m Model) Model {
		return m.dirtyMarked(at)
	})
	if errors.Is(err, atlas.ErrNotFound) {
		return nil
	}
	return err
}

func (r *Registry) UpdateChannel(ctx context.Context, characterId uint32, worldId world.Id, channelId channel.Id) error {
	t := tenant.MustFromContext(ctx)
	_, err := r.entries.Update(ctx, t, characterId, func(m Model) Model {
		return m.channelUpdated(worldId, channelId)
	})
	if errors.Is(err, atlas.ErrNotFound) {
		return nil
	}
	return err
}

func (r *Registry) UpdateSkillLevel(ctx context.Context, characterId uint32, level byte) error {
	t := tenant.MustFromContext(ctx)
	_, err := r.entries.Update(ctx, t, characterId, func(m Model) Model {
		return m.skillLevelUpdated(level)
	})
	if errors.Is(err, atlas.ErrNotFound) {
		return ErrNotFound
	}
	return err
}

// ClaimReeval atomically claims a due re-evaluation: it clears dirtyAt and
// returns (entry, true) iff the entry was dirty, due, and routable. Update is
// a single-attempt WATCH/MULTI (tenant_registry.go:130): when two replicas
// race, the loser's transaction fails and we report not-claimed — at most one
// re-evaluation runs per deadline (design D2).
func (r *Registry) ClaimReeval(ctx context.Context, characterId uint32, now time.Time) (Model, bool) {
	t := tenant.MustFromContext(ctx)
	claimed := false
	m, err := r.entries.Update(ctx, t, characterId, func(m Model) Model {
		claimed = false
		if m.DirtyDue(now) {
			claimed = true
			return m.dirtyCleared()
		}
		return m
	})
	if err != nil || !claimed {
		return Model{}, false
	}
	return m, true
}

// ClaimBroadcast atomically claims a due broadcast tick, advancing the
// deadline by BroadcastPeriod. Returns the claimed state to emit. Same
// single-winner semantics as ClaimReeval.
func (r *Registry) ClaimBroadcast(ctx context.Context, characterId uint32, now time.Time) (Model, bool) {
	t := tenant.MustFromContext(ctx)
	claimed := false
	m, err := r.entries.Update(ctx, t, characterId, func(m Model) Model {
		claimed = false
		if m.BroadcastDue(now) {
			claimed = true
			return m.broadcastScheduled(now.Add(BroadcastPeriod))
		}
		return m
	})
	if err != nil || !claimed {
		return Model{}, false
	}
	return m, true
}

// StoreEvaluation writes the outcome of a re-evaluation: captured active
// state, refreshed character level, and a fresh initial-delay schedule
// (Cosmic parity: every re-evaluation replaces the schedule, design D2).
func (r *Registry) StoreEvaluation(ctx context.Context, characterId uint32, active bool, characterLevel byte, nextBroadcastAt time.Time) error {
	t := tenant.MustFromContext(ctx)
	_, err := r.entries.Update(ctx, t, characterId, func(m Model) Model {
		return m.evaluated(active, characterLevel, nextBroadcastAt)
	})
	if errors.Is(err, atlas.ErrNotFound) {
		return nil
	}
	return err
}
```

Note for the implementer: `ClaimBroadcast` returns the post-`Update` model (whose `nextBroadcastAt` is already advanced) — that is fine because the emitter only reads `WorldId/ChannelId/CharacterId/CharacterLevel/SkillLevel/Active`, none of which the claim touches.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -race ./berserk/... -v`
Expected: PASS, including `TestConcurrentClaimSingleWinner` under `-race`.

- [ ] **Step 5: Commit**

```bash
git add berserk/registry.go berserk/registry_test.go berserk/testmain_test.go
git commit -m "feat(task-154): redis-backed berserk registry with atomic reeval/broadcast claims"
```

---

### Task 4: external REST clients + effect-x cache

**Files:**
- Create: `services/atlas-buffs/atlas.com/buffs/external/character/requests.go`, `external/character/rest.go`
- Create: `services/atlas-buffs/atlas.com/buffs/external/skills/requests.go`, `external/skills/rest.go`
- Create: `services/atlas-buffs/atlas.com/buffs/external/effectivestats/requests.go`, `external/effectivestats/rest.go`
- Create: `services/atlas-buffs/atlas.com/buffs/external/dataskill/requests.go`, `external/dataskill/rest.go`
- Create: `services/atlas-buffs/atlas.com/buffs/berserk/cache.go`
- Test: `services/atlas-buffs/atlas.com/buffs/berserk/cache_test.go`

**Interfaces:**
- Consumes: `requests.RootUrl` / `requests.GetRequest` from `libs/atlas-rest/requests` (`RootUrl` resolves `<DOMAIN>_SERVICE_URL` with `BASE_SERVICE_URL` fallback — never hard-code URLs; known footgun). Template: `services/atlas-effective-stats/atlas.com/effective-stats/external/character/`.
- Produces:
  - `external/character.RequestById(id uint32) requests.Request[RestModel]` — `GET characters/{id}`, RestModel has `Hp uint16`, `Level byte`
  - `external/skills.RequestByCharacterAndSkill(characterId uint32, skillId uint32) requests.Request[RestModel]` — `GET characters/{cid}/skills/{sid}`, RestModel has `Level byte`
  - `external/effectivestats.RequestByCharacter(worldId world.Id, channelId channel.Id, characterId uint32) requests.Request[RestModel]` — `GET worlds/{w}/channels/{c}/characters/{id}/stats`, RestModel has `MaxHp uint32` (JSON tag `maxHP` — uppercase HP, verified against `services/atlas-effective-stats/.../stat/rest.go`)
  - `external/dataskill.RequestById(skillId uint32) requests.Request[RestModel]` — `GET data/skills/{id}`, RestModel has `Effects []EffectModel` with `X int16` (tag `x`)
  - `berserk.NewEffectXCache(fetch func(l logrus.FieldLogger, ctx context.Context) (dataskill.RestModel, error)) *EffectXCache` with `X(l, ctx, skillLevel byte) (int16, error)`; package singleton `GetEffectXCache()` (sync.Once) wired to `dataskill.RequestById(uint32(skill.DarkKnightBerserkId))`

- [ ] **Step 1: Write the failing cache test**

`berserk/cache_test.go`:

```go
package berserk

import (
	"context"
	"errors"
	"testing"

	"atlas-buffs/external/dataskill"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func fixedSkill(xs ...int16) dataskill.RestModel {
	effects := make([]dataskill.EffectModel, 0, len(xs))
	for _, x := range xs {
		effects = append(effects, dataskill.EffectModel{X: x})
	}
	return dataskill.RestModel{Effects: effects}
}

func cacheCtx(t *testing.T) context.Context {
	t.Helper()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	assert.NoError(t, err)
	return tenant.WithContext(context.Background(), ten)
}

func TestEffectXCacheResolvesPerLevel(t *testing.T) {
	calls := 0
	c := NewEffectXCache(func(_ logrus.FieldLogger, _ context.Context) (dataskill.RestModel, error) {
		calls++
		return fixedSkill(21, 22, 23), nil
	})
	l := logrus.New()
	ctx := cacheCtx(t)

	x, err := c.X(l, ctx, 1)
	assert.NoError(t, err)
	assert.Equal(t, int16(21), x)

	x, err = c.X(l, ctx, 3)
	assert.NoError(t, err)
	assert.Equal(t, int16(23), x)

	assert.Equal(t, 1, calls, "effect data is immutable per tenant: fetched once")
}

func TestEffectXCacheTenantScoped(t *testing.T) {
	calls := 0
	c := NewEffectXCache(func(_ logrus.FieldLogger, _ context.Context) (dataskill.RestModel, error) {
		calls++
		return fixedSkill(21), nil
	})
	l := logrus.New()

	_, err := c.X(l, cacheCtx(t), 1)
	assert.NoError(t, err)
	_, err = c.X(l, cacheCtx(t), 1)
	assert.NoError(t, err)
	assert.Equal(t, 2, calls, "one fetch per tenant")
}

func TestEffectXCacheBounds(t *testing.T) {
	c := NewEffectXCache(func(_ logrus.FieldLogger, _ context.Context) (dataskill.RestModel, error) {
		return fixedSkill(21, 22), nil
	})
	l := logrus.New()
	ctx := cacheCtx(t)

	_, err := c.X(l, ctx, 0)
	assert.Error(t, err, "level 0 has no effect entry")
	_, err = c.X(l, ctx, 3)
	assert.Error(t, err, "level beyond data is an error, not a panic")
}

func TestEffectXCacheFetchFailureNotCached(t *testing.T) {
	fail := true
	c := NewEffectXCache(func(_ logrus.FieldLogger, _ context.Context) (dataskill.RestModel, error) {
		if fail {
			return dataskill.RestModel{}, errors.New("boom")
		}
		return fixedSkill(21), nil
	})
	l := logrus.New()
	ctx := cacheCtx(t)

	_, err := c.X(l, ctx, 1)
	assert.Error(t, err)

	fail = false
	x, err := c.X(l, ctx, 1)
	assert.NoError(t, err, "failed fetch must not poison the cache")
	assert.Equal(t, int16(21), x)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./berserk/... -run TestEffectXCache -v`
Expected: FAIL — `atlas-buffs/external/dataskill` does not exist / `undefined: NewEffectXCache`.

- [ ] **Step 3: Write the clients and cache**

`external/character/rest.go`:

```go
package character

import "strconv"

// RestModel is the trimmed atlas-character projection this service reads:
// current HP and character level per re-evaluation (design D5).
type RestModel struct {
	Id    uint32 `json:"-"`
	Level byte   `json:"level"`
	Hp    uint16 `json:"hp"`
}

func (r RestModel) GetName() string {
	return "characters"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(idStr string) error {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}
```

`external/character/requests.go`:

```go
package character

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource = "characters"
	ById     = Resource + "/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("CHARACTERS")
}

func RequestById(id uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+ById, id))
}
```

`external/skills/rest.go`:

```go
package skills

import "strconv"

// RestModel is the trimmed atlas-skills projection: Berserk level at login.
type RestModel struct {
	Id    uint32 `json:"-"`
	Level byte   `json:"level"`
}

func (r RestModel) GetName() string {
	return "skills"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(idStr string) error {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}
```

`external/skills/requests.go`:

```go
package skills

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource          = "characters/%d/skills"
	ByCharacterSkill  = Resource + "/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("SKILLS")
}

func RequestByCharacterAndSkill(characterId uint32, skillId uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+ByCharacterSkill, characterId, skillId))
}
```

`external/effectivestats/rest.go`:

```go
package effectivestats

// RestModel is the trimmed atlas-effective-stats projection: buff-inclusive
// max HP (JSON tag maxHP per services/atlas-effective-stats stat/rest.go).
type RestModel struct {
	Id    string `json:"-"`
	MaxHp uint32 `json:"maxHP"`
}

func (r RestModel) GetName() string {
	return "effective-stats"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}
```

`external/effectivestats/requests.go`:

```go
package effectivestats

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	ByCharacter = "worlds/%d/channels/%d/characters/%d/stats"
)

func getBaseRequest() string {
	return requests.RootUrl("EFFECTIVE_STATS")
}

func RequestByCharacter(worldId world.Id, channelId channel.Id, characterId uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+ByCharacter, worldId, channelId, characterId))
}
```

`external/dataskill/rest.go`:

```go
package dataskill

import "strconv"

// RestModel is the trimmed atlas-data skill projection: per-level effect x
// (the berserk threshold percentage — the WZ `berserk` field is a dead type
// marker in Atlas and MUST NOT be read; design §2).
type RestModel struct {
	Id      uint32        `json:"-"`
	Effects []EffectModel `json:"effects"`
}

type EffectModel struct {
	X int16 `json:"x"`
}

func (r RestModel) GetName() string {
	return "skills"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(idStr string) error {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}
```

`external/dataskill/requests.go`:

```go
package dataskill

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource = "data/skills"
	ById     = Resource + "/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

func RequestById(skillId uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+ById, skillId))
}
```

`berserk/cache.go`:

```go
package berserk

import (
	"context"
	"fmt"
	"sync"

	"atlas-buffs/external/dataskill"

	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// EffectXCache caches Berserk's per-level effect x values per tenant. Effect
// data is immutable for a tenant's lifetime, so one atlas-data fetch per
// tenant suffices (design D5). Failed fetches are not cached.
type EffectXCache struct {
	mu       sync.RWMutex
	byTenant map[uuid.UUID][]int16
	fetch    func(l logrus.FieldLogger, ctx context.Context) (dataskill.RestModel, error)
}

func NewEffectXCache(fetch func(l logrus.FieldLogger, ctx context.Context) (dataskill.RestModel, error)) *EffectXCache {
	return &EffectXCache{
		byTenant: make(map[uuid.UUID][]int16),
		fetch:    fetch,
	}
}

var effectXCache *EffectXCache
var effectXCacheOnce sync.Once

func GetEffectXCache() *EffectXCache {
	effectXCacheOnce.Do(func() {
		effectXCache = NewEffectXCache(func(l logrus.FieldLogger, ctx context.Context) (dataskill.RestModel, error) {
			return dataskill.RequestById(uint32(skill.DarkKnightBerserkId))(l, ctx)
		})
	})
	return effectXCache
}

func (c *EffectXCache) X(l logrus.FieldLogger, ctx context.Context, skillLevel byte) (int16, error) {
	t := tenant.MustFromContext(ctx)

	c.mu.RLock()
	xs, ok := c.byTenant[t.Id()]
	c.mu.RUnlock()

	if !ok {
		rm, err := c.fetch(l, ctx)
		if err != nil {
			return 0, err
		}
		xs = make([]int16, 0, len(rm.Effects))
		for _, e := range rm.Effects {
			xs = append(xs, e.X)
		}
		c.mu.Lock()
		c.byTenant[t.Id()] = xs
		c.mu.Unlock()
	}

	if skillLevel == 0 || int(skillLevel) > len(xs) {
		return 0, fmt.Errorf("no effect data for skill [%d] level [%d]", uint32(skill.DarkKnightBerserkId), skillLevel)
	}
	return xs[skillLevel-1], nil
}
```

- [ ] **Step 4: Run tests and build**

Run: `go build ./... && go test ./berserk/... -run TestEffectXCache -v`
Expected: build clean, cache tests PASS.

- [ ] **Step 5: Commit**

```bash
git add external berserk/cache.go berserk/cache_test.go
git commit -m "feat(task-154): outbound REST clients and per-tenant effect-x cache"
```

---

### Task 5: BERSERK event contract + producer

**Files:**
- Modify: `services/atlas-buffs/atlas.com/buffs/kafka/message/character/kafka.go` (append after `ExpiredStatusEventBody`, before the `EnvCommandTopicCharacter` block)
- Create: `services/atlas-buffs/atlas.com/buffs/berserk/producer.go`
- Test: `services/atlas-buffs/atlas.com/buffs/berserk/producer_test.go`

**Interfaces:**
- Consumes: existing `StatusEvent[E]` envelope + `EnvEventStatusTopic` in `kafka/message/character/kafka.go:61-71`; `Model` (Task 1); `producer.CreateKey`/`producer.SingleMessageProvider` from `libs/atlas-kafka` (same shape as `character/producer.go`).
- Produces: `character2.EventStatusTypeBerserk = "BERSERK"`, `character2.BerserkStatusEventBody`; `berserk.berserkStatusEventProvider(transactionId uuid.UUID, m Model) model.Provider[[]kafka.Message]` (package-private; used by Task 6's ticker).

Design note (D6): a new event type on the EXISTING `EVENT_TOPIC_CHARACTER_BUFF_STATUS` topic — existing consumers type-guard and ignore unknown types; a dedicated topic would need configmap/overlay wiring (known failure family). The envelope has no transaction id, so it rides in the body. Key = character id → per-character ordering alongside the character's other buff events.

- [ ] **Step 1: Write the failing test**

`berserk/producer_test.go`:

```go
package berserk

import (
	"encoding/json"
	"testing"
	"time"

	character2 "atlas-buffs/kafka/message/character"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// Providers are pure message builders, so the on-the-wire JSON contract is
// asserted directly — this is the emit-side half of the golden contract test
// (atlas-channel's mirror decode is the consume-side half, Task 9).
func TestBerserkStatusEventProvider(t *testing.T) {
	txId := uuid.New()
	m := NewBuilder(world.Id(1), 42, 20).
		SetChannel(channel.Id(3)).
		SetCharacterLevel(135).
		Build().
		evaluated(true, 135, time.Time{})

	msgs, err := berserkStatusEventProvider(txId, m)()
	assert.NoError(t, err)
	assert.Len(t, msgs, 1)

	var e character2.StatusEvent[character2.BerserkStatusEventBody]
	assert.NoError(t, json.Unmarshal(msgs[0].Value, &e))
	assert.Equal(t, world.Id(1), e.WorldId)
	assert.Equal(t, uint32(42), e.CharacterId)
	assert.Equal(t, character2.EventStatusTypeBerserk, e.Type)
	assert.Equal(t, txId, e.Body.TransactionId)
	assert.Equal(t, channel.Id(3), e.Body.ChannelId)
	assert.Equal(t, uint32(skill.DarkKnightBerserkId), e.Body.SkillId)
	assert.Equal(t, byte(135), e.Body.CharacterLevel)
	assert.Equal(t, byte(20), e.Body.SkillLevel)
	assert.True(t, e.Body.Active)

	// Key must be the character id (per-character ordering on the topic).
	assert.NotEmpty(t, msgs[0].Key)
}

func TestBerserkStatusEventJSONFieldNames(t *testing.T) {
	body := character2.BerserkStatusEventBody{}
	data, err := json.Marshal(body)
	assert.NoError(t, err)
	for _, field := range []string{"transactionId", "channelId", "skillId", "characterLevel", "skillLevel", "active"} {
		assert.Contains(t, string(data), `"`+field+`"`, "JSON field names are the cross-service contract with atlas-channel")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./berserk/... -run TestBerserkStatus -v`
Expected: FAIL — `undefined: character2.BerserkStatusEventBody` / `berserkStatusEventProvider`.

- [ ] **Step 3: Write the implementation**

Append to `kafka/message/character/kafka.go` (after `ExpiredStatusEventBody`, line 90):

```go
const (
	EventStatusTypeBerserk = "BERSERK"
)

// BerserkStatusEventBody is one broadcast tick of Dark Knight Berserk aura
// state (task-154). Emitted every BroadcastPeriod per tracked Dark Knight
// with the state captured at the last re-evaluation; Active=false ticks are
// intentional — they clear the aura and keep late-joining observers
// consistent. ChannelId rides in the body because this topic's envelope has
// no channel; it lets atlas-channel guard with sc.Is(tenant, world, channel).
type BerserkStatusEventBody struct {
	TransactionId  uuid.UUID  `json:"transactionId"`
	ChannelId      channel.Id `json:"channelId"`
	SkillId        uint32     `json:"skillId"`
	CharacterLevel byte       `json:"characterLevel"`
	SkillLevel     byte       `json:"skillLevel"`
	Active         bool       `json:"active"`
}
```

(`uuid` and `channel` are already imported in that file.)

`berserk/producer.go`:

```go
package berserk

import (
	character2 "atlas-buffs/kafka/message/character"

	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func berserkStatusEventProvider(transactionId uuid.UUID, m Model) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(m.CharacterId()))
	value := &character2.StatusEvent[character2.BerserkStatusEventBody]{
		WorldId:     m.WorldId(),
		CharacterId: m.CharacterId(),
		Type:        character2.EventStatusTypeBerserk,
		Body: character2.BerserkStatusEventBody{
			TransactionId:  transactionId,
			ChannelId:      m.ChannelId(),
			SkillId:        uint32(skill.DarkKnightBerserkId),
			CharacterLevel: m.CharacterLevel(),
			SkillLevel:     m.SkillLevel(),
			Active:         m.Active(),
		},
	}
	return producer.SingleMessageProvider(key, value)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./berserk/... -run TestBerserkStatus -v && go build ./...`
Expected: PASS, build clean.

- [ ] **Step 5: Commit**

```bash
git add kafka/message/character/kafka.go berserk/producer.go berserk/producer_test.go
git commit -m "feat(task-154): BERSERK status event contract and producer"
```

---

### Task 6: berserk processor, scan ticker, main.go wiring

**Files:**
- Create: `services/atlas-buffs/atlas.com/buffs/berserk/processor.go`
- Create: `services/atlas-buffs/atlas.com/buffs/tasks/berserk.go`
- Modify: `services/atlas-buffs/atlas.com/buffs/main.go`
- Test: `services/atlas-buffs/atlas.com/buffs/berserk/processor_test.go`

**Interfaces:**
- Consumes: registry (Task 3), `Evaluate` (Task 2), external clients + `GetEffectXCache` (Task 4), `berserkStatusEventProvider` (Task 5), `message.Emit` pattern (`character/processor.go:49`), fan-out shape (`character/processor.go:190-205`), `requests.ErrNotFound` from `libs/atlas-rest/requests`.
- Produces:

```go
type Processor interface {
	TrackOnLogin(worldId world.Id, channelId channel.Id, characterId uint32) error
	Untrack(characterId uint32) error
	HandleStatChanged(worldId world.Id, channelId channel.Id, characterId uint32, updates []stat.Type) error
	HandleTransfer(worldId world.Id, channelId channel.Id, characterId uint32) error
	HandleSkillUpdated(worldId world.Id, characterId uint32, level byte) error
	MarkMaxHpDirty(characterId uint32) error
	ProcessTicks() error
}
```

plus `NewProcessor(l, ctx) Processor` and package function `ProcessBerserkTicks(l logrus.FieldLogger, ctx context.Context) error` (per-tenant fan-out, the ticker entry point). `stat` here is `github.com/Chronicle20/atlas/libs/atlas-constants/stat` (`TypeHp = "HP"`, `TypeMaxHp = "MAX_HP"`).

Design rules encoded here (D3/D5/D7/§4.1):
- Consumers do no REST except `TrackOnLogin`'s one skill lookup (no event carries the level at login).
- The ticker does the two REST reads per re-evaluation (HP+level from atlas-character, maxHP from effective-stats); effect `x` is cached.
- Lookup failure: warn + re-arm `dirtyAt = now + ReevalRetryDelay`; the existing schedule keeps broadcasting last-known state.
- A skills REST 404 means level 0 (character never learned the skill) — not an error.
- Per pass an entry either re-evaluates or broadcasts, never both (5s initial delay always separates them).

- [ ] **Step 1: Write the failing test**

`berserk/processor_test.go`:

```go
package berserk

import (
	"context"
	"errors"
	"testing"
	"time"

	extchar "atlas-buffs/external/character"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/stat"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// testProcessor builds a ProcessorImpl with deterministic time and stubbed
// externals. Same-package construction — no test helpers file (project rule);
// the Builder pattern is used for all Model setup.
func testProcessor(t *testing.T, ctx context.Context, now time.Time) *ProcessorImpl {
	t.Helper()
	return &ProcessorImpl{
		l:   logrus.New(),
		ctx: ctx,
		now: func() time.Time { return now },
		getCharacter: func(characterId uint32) (extchar.RestModel, error) {
			return extchar.RestModel{Id: characterId, Level: 120, Hp: 100}, nil
		},
		getSkillLevel: func(characterId uint32) (byte, error) { return 10, nil },
		getMaxHp:      func(worldId world.Id, channelId channel.Id, characterId uint32) (uint32, error) { return 1000, nil },
		getEffectX:    func(skillLevel byte) (int16, error) { return 30, nil },
	}
}

func TestTrackOnLoginSkillLevelZeroNotTracked(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))
	p := testProcessor(t, ctx, time.Now())
	p.getSkillLevel = func(uint32) (byte, error) { return 0, nil }

	assert.NoError(t, p.TrackOnLogin(world.Id(0), channel.Id(1), 42))
	_, err := GetRegistry().Get(ctx, 42)
	assert.ErrorIs(t, err, ErrNotFound, "level 0 characters generate no registry entries")
}

func TestTrackOnLoginTracksAndMarksDirty(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))
	now := time.Now()
	p := testProcessor(t, ctx, now)

	assert.NoError(t, p.TrackOnLogin(world.Id(0), channel.Id(1), 42))
	m, err := GetRegistry().Get(ctx, 42)
	assert.NoError(t, err)
	assert.Equal(t, byte(10), m.SkillLevel())
	assert.True(t, m.ChannelKnown())
	assert.True(t, m.DirtyDue(now))
}

func TestHandleStatChanged(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))
	now := time.Now()
	p := testProcessor(t, ctx, now)

	// Untracked: zero work, no error.
	assert.NoError(t, p.HandleStatChanged(world.Id(0), channel.Id(1), 99, []stat.Type{stat.TypeHp}))

	assert.NoError(t, GetRegistry().Track(ctx, NewBuilder(world.Id(0), 42, 10).SetChannel(channel.Id(1)).Build()))

	// Non-HP updates refresh channel but do not mark dirty.
	assert.NoError(t, p.HandleStatChanged(world.Id(0), channel.Id(2), 42, []stat.Type{stat.TypeStrength}))
	m, _ := GetRegistry().Get(ctx, 42)
	assert.Equal(t, channel.Id(2), m.ChannelId())
	assert.True(t, m.DirtyAt().IsZero())

	// HP update: dirty now.
	assert.NoError(t, p.HandleStatChanged(world.Id(0), channel.Id(2), 42, []stat.Type{stat.TypeHp}))
	m, _ = GetRegistry().Get(ctx, 42)
	assert.True(t, m.DirtyAt().Equal(now))

	// MAX_HP present: grace-deferred even when HP is also present (the
	// max-HP recompute in effective-stats is what we are waiting out).
	assert.NoError(t, p.HandleStatChanged(world.Id(0), channel.Id(2), 42, []stat.Type{stat.TypeHp, stat.TypeMaxHp}))
	m, _ = GetRegistry().Get(ctx, 42)
	assert.True(t, m.DirtyAt().Equal(now.Add(ReevalGrace)))
}

func TestHandleSkillUpdated(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))
	now := time.Now()
	p := testProcessor(t, ctx, now)

	// New (SP allocation 0→1): tracked without channel.
	assert.NoError(t, p.HandleSkillUpdated(world.Id(0), 42, 1))
	m, err := GetRegistry().Get(ctx, 42)
	assert.NoError(t, err)
	assert.Equal(t, byte(1), m.SkillLevel())
	assert.False(t, m.ChannelKnown())
	assert.True(t, m.DirtyAt().Equal(now))

	// Existing: level refresh + dirty.
	assert.NoError(t, GetRegistry().UpdateChannel(ctx, 42, world.Id(0), channel.Id(1)))
	assert.NoError(t, p.HandleSkillUpdated(world.Id(0), 42, 2))
	m, _ = GetRegistry().Get(ctx, 42)
	assert.Equal(t, byte(2), m.SkillLevel())
	assert.True(t, m.ChannelKnown(), "level update must not lose the channel")

	// Level 0 (SP reset): untracked.
	assert.NoError(t, p.HandleSkillUpdated(world.Id(0), 42, 0))
	_, err = GetRegistry().Get(ctx, 42)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestProcessTicksReevaluates(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))
	now := time.Now()
	p := testProcessor(t, ctx, now)
	// hp=100, maxHp=1000, x=30 → 10 < 30 → active.

	assert.NoError(t, GetRegistry().Track(ctx,
		NewBuilder(world.Id(0), 42, 10).SetChannel(channel.Id(1)).SetDirtyAt(now).Build()))

	assert.NoError(t, p.ProcessTicks())

	m, err := GetRegistry().Get(ctx, 42)
	assert.NoError(t, err)
	assert.True(t, m.Active())
	assert.Equal(t, byte(120), m.CharacterLevel(), "character level refreshed from REST")
	assert.True(t, m.DirtyAt().IsZero(), "claim cleared")
	assert.True(t, m.NextBroadcastAt().Equal(now.Add(InitialBroadcastDelay)), "fresh 5s schedule")
}

func TestProcessTicksReevalDoesNotBroadcastSamePass(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))
	now := time.Now()
	p := testProcessor(t, ctx, now)

	// Dirty AND broadcast-due: the re-evaluation wins the pass and replaces
	// the schedule (Cosmic cancel-and-replace semantics).
	m := NewBuilder(world.Id(0), 42, 10).SetChannel(channel.Id(1)).SetDirtyAt(now).Build().
		evaluated(false, 120, now)
	assert.NoError(t, GetRegistry().Track(ctx, m))

	assert.NoError(t, p.ProcessTicks())

	got, _ := GetRegistry().Get(ctx, 42)
	assert.True(t, got.NextBroadcastAt().Equal(now.Add(InitialBroadcastDelay)),
		"schedule replaced by re-evaluation, not advanced by a broadcast claim")
}

func TestProcessTicksBroadcastAdvancesSchedule(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))
	now := time.Now()
	p := testProcessor(t, ctx, now)

	m := NewBuilder(world.Id(0), 42, 10).SetChannel(channel.Id(1)).Build().
		evaluated(true, 120, now)
	assert.NoError(t, GetRegistry().Track(ctx, m))

	assert.NoError(t, p.ProcessTicks())

	got, _ := GetRegistry().Get(ctx, 42)
	assert.True(t, got.NextBroadcastAt().Equal(now.Add(BroadcastPeriod)))
	assert.True(t, got.Active(), "broadcast uses captured state, does not recompute")
}

func TestProcessTicksLookupFailureRearms(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))
	now := time.Now()
	p := testProcessor(t, ctx, now)
	p.getMaxHp = func(world.Id, channel.Id, uint32) (uint32, error) { return 0, errors.New("effective-stats down") }

	m := NewBuilder(world.Id(0), 42, 10).SetChannel(channel.Id(1)).SetDirtyAt(now).Build().
		evaluated(true, 120, now.Add(time.Minute))
	assert.NoError(t, GetRegistry().Track(ctx, m))

	assert.NoError(t, p.ProcessTicks(), "lookup failure never fails the pass")

	got, _ := GetRegistry().Get(ctx, 42)
	assert.True(t, got.DirtyAt().Equal(now.Add(ReevalRetryDelay)), "re-armed for retry")
	assert.True(t, got.Active(), "last-known state kept")
	assert.True(t, got.NextBroadcastAt().Equal(now.Add(time.Minute)), "existing schedule untouched")
}

func TestProcessTicksMaxHpZeroGuard(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))
	now := time.Now()
	p := testProcessor(t, ctx, now)
	p.getMaxHp = func(world.Id, channel.Id, uint32) (uint32, error) { return 0, nil }

	assert.NoError(t, GetRegistry().Track(ctx,
		NewBuilder(world.Id(0), 42, 10).SetChannel(channel.Id(1)).SetDirtyAt(now).Build()))

	assert.NoError(t, p.ProcessTicks())
	got, _ := GetRegistry().Get(ctx, 42)
	assert.True(t, got.DirtyAt().Equal(now.Add(ReevalRetryDelay)), "maxHp=0 treated as failed lookup")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./berserk/... -run 'TestTrackOnLogin|TestHandle|TestProcessTicks' -v`
Expected: FAIL — `undefined: ProcessorImpl`.

- [ ] **Step 3: Write the implementation**

`berserk/processor.go`:

```go
package berserk

import (
	"context"
	"errors"
	"time"

	extchar "atlas-buffs/external/character"
	exteffstats "atlas-buffs/external/effectivestats"
	extskills "atlas-buffs/external/skills"
	"atlas-buffs/kafka/message"
	character2 "atlas-buffs/kafka/message/character"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-constants/stat"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	TrackOnLogin(worldId world.Id, channelId channel.Id, characterId uint32) error
	Untrack(characterId uint32) error
	HandleStatChanged(worldId world.Id, channelId channel.Id, characterId uint32, updates []stat.Type) error
	HandleTransfer(worldId world.Id, channelId channel.Id, characterId uint32) error
	HandleSkillUpdated(worldId world.Id, characterId uint32, level byte) error
	MarkMaxHpDirty(characterId uint32) error
	ProcessTicks() error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	now func() time.Time

	getCharacter  func(characterId uint32) (extchar.RestModel, error)
	getSkillLevel func(characterId uint32) (byte, error)
	getMaxHp      func(worldId world.Id, channelId channel.Id, characterId uint32) (uint32, error)
	getEffectX    func(skillLevel byte) (int16, error)
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
		now: time.Now,
	}
	p.getCharacter = func(characterId uint32) (extchar.RestModel, error) {
		return extchar.RequestById(characterId)(l, ctx)
	}
	p.getSkillLevel = func(characterId uint32) (byte, error) {
		rm, err := extskills.RequestByCharacterAndSkill(characterId, uint32(skill.DarkKnightBerserkId))(l, ctx)
		if errors.Is(err, requests.ErrNotFound) {
			// The character never learned the skill: level 0, not an error.
			return 0, nil
		}
		if err != nil {
			return 0, err
		}
		return rm.Level, nil
	}
	p.getMaxHp = func(worldId world.Id, channelId channel.Id, characterId uint32) (uint32, error) {
		rm, err := exteffstats.RequestByCharacter(worldId, channelId, characterId)(l, ctx)
		if err != nil {
			return 0, err
		}
		return rm.MaxHp, nil
	}
	p.getEffectX = func(skillLevel byte) (int16, error) {
		return GetEffectXCache().X(l, ctx, skillLevel)
	}
	return p
}

// TrackOnLogin is the only consumer-driven REST call (design D3): no event
// carries the Berserk level at login. Level 0 (all non-Dark-Knights) is
// filtered here — no registry entry, no ticker work, no events.
func (p *ProcessorImpl) TrackOnLogin(worldId world.Id, channelId channel.Id, characterId uint32) error {
	level, err := p.getSkillLevel(characterId)
	if err != nil {
		return err
	}
	if level == 0 {
		return nil
	}
	m := NewBuilder(worldId, characterId, level).
		SetChannel(channelId).
		SetDirtyAt(p.now()).
		Build()
	p.l.Infof("Tracking berserk for character [%d] at skill level [%d].", characterId, level)
	return GetRegistry().Track(p.ctx, m)
}

func (p *ProcessorImpl) Untrack(characterId uint32) error {
	p.l.Infof("Untracking berserk for character [%d].", characterId)
	return GetRegistry().Untrack(p.ctx, characterId)
}

// HandleStatChanged refreshes the routing channel (design D8: every
// channel-bearing character event refreshes it) and marks dirty when the
// update touches HP. MAX_HP updates get the grace deferral even when HP moved
// too: the effective-stats MAX_HP recompute is exactly what the grace waits
// out (design D5).
func (p *ProcessorImpl) HandleStatChanged(worldId world.Id, channelId channel.Id, characterId uint32, updates []stat.Type) error {
	if err := GetRegistry().UpdateChannel(p.ctx, characterId, worldId, channelId); err != nil {
		return err
	}
	var dirtyAt time.Time
	for _, u := range updates {
		if u == stat.TypeMaxHp {
			dirtyAt = p.now().Add(ReevalGrace)
			break
		}
		if u == stat.TypeHp {
			dirtyAt = p.now()
		}
	}
	if dirtyAt.IsZero() {
		return nil
	}
	return GetRegistry().MarkDirty(p.ctx, characterId, dirtyAt)
}

// HandleTransfer covers MAP_CHANGED and CHANNEL_CHANGED (Cosmic re-checks on
// transfer).
func (p *ProcessorImpl) HandleTransfer(worldId world.Id, channelId channel.Id, characterId uint32) error {
	if err := GetRegistry().UpdateChannel(p.ctx, characterId, worldId, channelId); err != nil {
		return err
	}
	return GetRegistry().MarkDirty(p.ctx, characterId, p.now())
}

// HandleSkillUpdated tracks SP allocation into Berserk without a relog. New
// entries have no channel (the skill event carries none); the next
// channel-bearing character event fills it in (design D8).
func (p *ProcessorImpl) HandleSkillUpdated(worldId world.Id, characterId uint32, level byte) error {
	if level == 0 {
		return p.Untrack(characterId)
	}
	err := GetRegistry().UpdateSkillLevel(p.ctx, characterId, level)
	if errors.Is(err, ErrNotFound) {
		p.l.Infof("Tracking berserk for character [%d] at skill level [%d] (skill update).", characterId, level)
		return GetRegistry().Track(p.ctx,
			NewBuilder(worldId, characterId, level).SetDirtyAt(p.now()).Build())
	}
	if err != nil {
		return err
	}
	return GetRegistry().MarkDirty(p.ctx, characterId, p.now())
}

// MarkMaxHpDirty is the in-process hook for buff apply/expire/cancel whose
// stat-ups affect max HP (Hyper Body). Grace-deferred: atlas-buffs is the
// producer of the very event effective-stats consumes to recompute max HP
// (design D5).
func (p *ProcessorImpl) MarkMaxHpDirty(characterId uint32) error {
	return GetRegistry().MarkDirty(p.ctx, characterId, p.now().Add(ReevalGrace))
}

// ProcessTicks is one scan pass for one tenant: claim due re-evaluations
// (2 REST reads each), else claim due broadcasts and emit. Claims are atomic
// across replicas — at most one emitter per deadline (design D2).
func (p *ProcessorImpl) ProcessTicks() error {
	now := p.now()
	entries := GetRegistry().GetAll(p.ctx)

	return message.Emit(p.l, p.ctx)(func(buf *message.Buffer) error {
		for _, e := range entries {
			if e.DirtyDue(now) {
				if m, ok := GetRegistry().ClaimReeval(p.ctx, e.CharacterId(), now); ok {
					p.reevaluate(m, now)
				}
				continue
			}
			if e.BroadcastDue(now) {
				if m, ok := GetRegistry().ClaimBroadcast(p.ctx, e.CharacterId(), now); ok {
					if err := buf.Put(character2.EnvEventStatusTopic, berserkStatusEventProvider(uuid.New(), m)); err != nil {
						return err
					}
				}
			}
		}
		return nil
	})
}

// reevaluate runs the FR-1 computation for a claimed entry. Any lookup
// failure warns and re-arms dirtyAt so a later pass retries; the existing
// schedule keeps broadcasting the last-known state meanwhile (FR-5).
func (p *ProcessorImpl) reevaluate(m Model, now time.Time) {
	rearm := func(reason string, err error) {
		p.l.WithError(err).Warnf("Berserk re-evaluation for character [%d] failed (%s); retrying.", m.CharacterId(), reason)
		_ = GetRegistry().MarkDirty(p.ctx, m.CharacterId(), now.Add(ReevalRetryDelay))
	}

	x, err := p.getEffectX(m.SkillLevel())
	if err != nil {
		rearm("effect data", err)
		return
	}
	c, err := p.getCharacter(m.CharacterId())
	if err != nil {
		rearm("character", err)
		return
	}
	maxHp, err := p.getMaxHp(m.WorldId(), m.ChannelId(), m.CharacterId())
	if err != nil {
		rearm("effective stats", err)
		return
	}
	if maxHp == 0 {
		rearm("effective stats", errors.New("effective max HP is zero"))
		return
	}

	active := Evaluate(m.SkillLevel(), c.Hp, maxHp, x)
	if active != m.Active() {
		p.l.Debugf("Berserk state for character [%d] now [%v] (hp [%d] maxHp [%d] x [%d]).", m.CharacterId(), active, c.Hp, maxHp, x)
	}
	if err := GetRegistry().StoreEvaluation(p.ctx, m.CharacterId(), active, c.Level, now.Add(InitialBroadcastDelay)); err != nil {
		p.l.WithError(err).Warnf("Unable to store berserk evaluation for character [%d].", m.CharacterId())
	}
}

// ProcessBerserkTicks fans out one ProcessTicks per tenant (ticker entry
// point; same shape as character.ProcessPoisonTicks, processor.go:190-205).
func ProcessBerserkTicks(l logrus.FieldLogger, ctx context.Context) error {
	ts, err := GetRegistry().GetTenants(ctx)
	if err != nil {
		return err
	}

	for _, t := range ts {
		go func() {
			tctx := tenant.WithContext(ctx, t)
			if err := NewProcessor(l, tctx).ProcessTicks(); err != nil {
				l.WithError(err).Error("Failed to process berserk ticks for tenant.")
			}
		}()
	}
	return nil
}
```

`tasks/berserk.go`:

```go
package tasks

import (
	"atlas-buffs/berserk"
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
)

type BerserkTick struct {
	l        logrus.FieldLogger
	interval int
}

func NewBerserkTick(l logrus.FieldLogger, interval int) *BerserkTick {
	return &BerserkTick{l, interval}
}

func (r *BerserkTick) Run() {
	ctx, span := otel.GetTracerProvider().Tracer("atlas-buffs").Start(context.Background(), "berserk_tick_task")
	defer span.End()

	_ = berserk.ProcessBerserkTicks(r.l, ctx)
}

func (r *BerserkTick) SleepTime() time.Duration {
	return time.Millisecond * time.Duration(r.interval)
}
```

`main.go` — two edits:

After `character.InitRegistry(rc)` add:

```go
	berserk.InitRegistry(rc)
```

After `go tasks.Register(tasks.NewPoisonTick(l, 1000))` add:

```go
	go tasks.Register(tasks.NewBerserkTick(l, 1000))
```

Add `"atlas-buffs/berserk"` to the imports.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -race ./... && go vet ./... && go build ./...`
Expected: all PASS/clean.

- [ ] **Step 5: Commit**

```bash
git add berserk/processor.go berserk/processor_test.go tasks/berserk.go main.go
git commit -m "feat(task-154): berserk processor, 1s scan ticker, service wiring"
```

---

### Task 7: character-status and skill-status consumers in atlas-buffs

**Files:**
- Create: `services/atlas-buffs/atlas.com/buffs/kafka/message/characterstatus/kafka.go`
- Create: `services/atlas-buffs/atlas.com/buffs/kafka/message/skillstatus/kafka.go`
- Create: `services/atlas-buffs/atlas.com/buffs/kafka/consumer/characterstatus/consumer.go`
- Create: `services/atlas-buffs/atlas.com/buffs/kafka/consumer/skillstatus/consumer.go`
- Modify: `services/atlas-buffs/atlas.com/buffs/main.go`
- Test: `services/atlas-buffs/atlas.com/buffs/kafka/consumer/characterstatus/consumer_test.go`, `services/atlas-buffs/atlas.com/buffs/kafka/consumer/skillstatus/consumer_test.go` (+ a `testmain_test.go` in each)

**Interfaces:**
- Consumes: `berserk.NewProcessor(l, ctx)` (Task 6). Event shapes are verbatim mirrors of the producers' structs (verified):
  - atlas-character (`services/atlas-character/atlas.com/character/kafka/message/character/kafka.go:212-346`): envelope `StatusEvent[E]{TransactionId uuid.UUID, WorldId world.Id, CharacterId uint32, Type string, Body E}`; types `LOGIN`, `LOGOUT`, `STAT_CHANGED`, `MAP_CHANGED`, `CHANNEL_CHANGED`; the CHANNEL_CHANGED body is named `ChangeChannelEventLoginBody` upstream (mirrored here as `StatusEventChannelChangedBody` — JSON is what matters).
  - atlas-skills (`services/atlas-skills/atlas.com/skills/kafka/message/skill/kafka.go:52-90`): envelope carries top-level `SkillId uint32`; types `UPDATED` (body `{Level byte, MasterLevel byte, Expiration time.Time}`) and `DELETED` (empty body).
- Produces: `characterstatus.InitConsumers/InitHandlers`, `skillstatus.InitConsumers/InitHandlers` (curried, same shape as `kafka/consumer/character/consumer.go`).

Consumer semantics (design §4.1 trigger table):

| Event | Handler action |
|---|---|
| LOGIN | `TrackOnLogin(worldId, body.ChannelId, characterId)` — the one consumer-side REST call |
| LOGOUT | `Untrack(characterId)` |
| STAT_CHANGED | `HandleStatChanged(worldId, body.ChannelId, characterId, body.Updates)` — untracked characters are no-ops inside the registry |
| MAP_CHANGED | `HandleTransfer(worldId, body.ChannelId, characterId)` |
| CHANNEL_CHANGED | `HandleTransfer(worldId, body.ChannelId, characterId)` |
| skill UPDATED with envelope `SkillId == uint32(skill.DarkKnightBerserkId)` | `HandleSkillUpdated(worldId, characterId, body.Level)` |
| skill DELETED with berserk SkillId | `Untrack(characterId)` |

Both consumers use `consumer.SetStartOffset(kafka.LastOffset)` (transient signals; state reconstructs from the next trigger — same convention as atlas-channel's buff status consumer).

- [ ] **Step 1: Write the failing tests**

`kafka/consumer/characterstatus/testmain_test.go` and `kafka/consumer/skillstatus/testmain_test.go` (identical content, own package names `characterstatus`/`skillstatus`):

```go
package characterstatus

import (
	"os"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer/producertest"
)

func TestMain(m *testing.M) {
	producertest.InstallNoop()
	os.Exit(m.Run())
}
```

`kafka/consumer/characterstatus/consumer_test.go`:

```go
package characterstatus

import (
	"context"
	"testing"
	"time"

	"atlas-buffs/berserk"
	characterstatus2 "atlas-buffs/kafka/message/characterstatus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/stat"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func setup(t *testing.T) context.Context {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	berserk.InitRegistry(client)
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	assert.NoError(t, err)
	return tenant.WithContext(context.Background(), ten)
}

func tracked(t *testing.T, ctx context.Context, characterId uint32) {
	t.Helper()
	assert.NoError(t, berserk.GetRegistry().Track(ctx,
		berserk.NewBuilder(world.Id(0), characterId, 10).SetChannel(channel.Id(1)).Build()))
}

func TestHandleLogoutUntracks(t *testing.T) {
	ctx := setup(t)
	l := logrus.New()
	tracked(t, ctx, 42)

	handleStatusEventLogout(l, ctx, characterstatus2.StatusEvent[characterstatus2.StatusEventLogoutBody]{
		WorldId: world.Id(0), CharacterId: 42, Type: characterstatus2.StatusEventTypeLogout,
	})

	_, err := berserk.GetRegistry().Get(ctx, 42)
	assert.ErrorIs(t, err, berserk.ErrNotFound)
}

func TestHandleLogoutWrongTypeIsNoop(t *testing.T) {
	ctx := setup(t)
	l := logrus.New()
	tracked(t, ctx, 42)

	handleStatusEventLogout(l, ctx, characterstatus2.StatusEvent[characterstatus2.StatusEventLogoutBody]{
		WorldId: world.Id(0), CharacterId: 42, Type: characterstatus2.StatusEventTypeLogin,
	})

	_, err := berserk.GetRegistry().Get(ctx, 42)
	assert.NoError(t, err, "wrong-type event must not mutate the registry")
}

func TestHandleStatChangedHpMarksDirty(t *testing.T) {
	ctx := setup(t)
	l := logrus.New()
	tracked(t, ctx, 42)

	handleStatusEventStatChanged(l, ctx, characterstatus2.StatusEvent[characterstatus2.StatusEventStatChangedBody]{
		WorldId: world.Id(0), CharacterId: 42, Type: characterstatus2.StatusEventTypeStatChanged,
		Body: characterstatus2.StatusEventStatChangedBody{ChannelId: channel.Id(2), Updates: []stat.Type{stat.TypeHp}},
	})

	m, err := berserk.GetRegistry().Get(ctx, 42)
	assert.NoError(t, err)
	assert.Equal(t, channel.Id(2), m.ChannelId(), "channel refreshed")
	assert.False(t, m.DirtyAt().IsZero(), "HP change marks dirty")
	assert.True(t, m.DirtyDue(time.Now().Add(time.Second)))
}

func TestHandleStatChangedUntrackedIsNoop(t *testing.T) {
	ctx := setup(t)
	l := logrus.New()

	handleStatusEventStatChanged(l, ctx, characterstatus2.StatusEvent[characterstatus2.StatusEventStatChangedBody]{
		WorldId: world.Id(0), CharacterId: 99, Type: characterstatus2.StatusEventTypeStatChanged,
		Body: characterstatus2.StatusEventStatChangedBody{ChannelId: channel.Id(1), Updates: []stat.Type{stat.TypeHp}},
	})

	assert.Empty(t, berserk.GetRegistry().GetAll(ctx), "untracked characters generate no entries")
}

func TestHandleMapChangedRefreshesChannelAndMarksDirty(t *testing.T) {
	ctx := setup(t)
	l := logrus.New()
	tracked(t, ctx, 42)

	handleStatusEventMapChanged(l, ctx, characterstatus2.StatusEvent[characterstatus2.StatusEventMapChangedBody]{
		WorldId: world.Id(0), CharacterId: 42, Type: characterstatus2.StatusEventTypeMapChanged,
		Body: characterstatus2.StatusEventMapChangedBody{ChannelId: channel.Id(3)},
	})

	m, err := berserk.GetRegistry().Get(ctx, 42)
	assert.NoError(t, err)
	assert.Equal(t, channel.Id(3), m.ChannelId())
	assert.False(t, m.DirtyAt().IsZero(), "Cosmic re-checks on transfer")
}

func TestHandleChannelChangedRefreshesChannel(t *testing.T) {
	ctx := setup(t)
	l := logrus.New()
	tracked(t, ctx, 42)

	handleStatusEventChannelChanged(l, ctx, characterstatus2.StatusEvent[characterstatus2.StatusEventChannelChangedBody]{
		WorldId: world.Id(0), CharacterId: 42, Type: characterstatus2.StatusEventTypeChannelChanged,
		Body: characterstatus2.StatusEventChannelChangedBody{ChannelId: channel.Id(4)},
	})

	m, err := berserk.GetRegistry().Get(ctx, 42)
	assert.NoError(t, err)
	assert.Equal(t, channel.Id(4), m.ChannelId())
}
```

Note: the LOGIN handler is intentionally not exercised here — it is a one-line adapter over `TrackOnLogin`, whose logic (skills lookup, level-0 filter, dirty tracking) is covered with stubbed externals in `berserk/processor_test.go` (Task 6). A consumer-level LOGIN test would need a live skills endpoint.

`kafka/consumer/skillstatus/consumer_test.go`:

```go
package skillstatus

import (
	"context"
	"testing"

	"atlas-buffs/berserk"
	skillstatus2 "atlas-buffs/kafka/message/skillstatus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func setup(t *testing.T) context.Context {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	berserk.InitRegistry(client)
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	assert.NoError(t, err)
	return tenant.WithContext(context.Background(), ten)
}

func TestHandleUpdatedTracksBerserkSkill(t *testing.T) {
	ctx := setup(t)
	l := logrus.New()

	handleStatusEventUpdated(l, ctx, skillstatus2.StatusEvent[skillstatus2.StatusEventUpdatedBody]{
		WorldId: world.Id(0), CharacterId: 42, SkillId: uint32(skill.DarkKnightBerserkId),
		Type: skillstatus2.StatusEventTypeUpdated,
		Body: skillstatus2.StatusEventUpdatedBody{Level: 1},
	})

	m, err := berserk.GetRegistry().Get(ctx, 42)
	assert.NoError(t, err)
	assert.Equal(t, byte(1), m.SkillLevel())
	assert.False(t, m.ChannelKnown(), "skill events carry no channel (design D8)")
}

func TestHandleUpdatedIgnoresOtherSkills(t *testing.T) {
	ctx := setup(t)
	l := logrus.New()

	handleStatusEventUpdated(l, ctx, skillstatus2.StatusEvent[skillstatus2.StatusEventUpdatedBody]{
		WorldId: world.Id(0), CharacterId: 42, SkillId: uint32(skill.DarkKnightAchillesId),
		Type: skillstatus2.StatusEventTypeUpdated,
		Body: skillstatus2.StatusEventUpdatedBody{Level: 5},
	})

	assert.Empty(t, berserk.GetRegistry().GetAll(ctx))
}

func TestHandleUpdatedLevelZeroUntracks(t *testing.T) {
	ctx := setup(t)
	l := logrus.New()
	assert.NoError(t, berserk.GetRegistry().Track(ctx,
		berserk.NewBuilder(world.Id(0), 42, 10).SetChannel(channel.Id(1)).Build()))

	handleStatusEventUpdated(l, ctx, skillstatus2.StatusEvent[skillstatus2.StatusEventUpdatedBody]{
		WorldId: world.Id(0), CharacterId: 42, SkillId: uint32(skill.DarkKnightBerserkId),
		Type: skillstatus2.StatusEventTypeUpdated,
		Body: skillstatus2.StatusEventUpdatedBody{Level: 0},
	})

	_, err := berserk.GetRegistry().Get(ctx, 42)
	assert.ErrorIs(t, err, berserk.ErrNotFound)
}

func TestHandleDeletedUntracks(t *testing.T) {
	ctx := setup(t)
	l := logrus.New()
	assert.NoError(t, berserk.GetRegistry().Track(ctx,
		berserk.NewBuilder(world.Id(0), 42, 10).SetChannel(channel.Id(1)).Build()))

	handleStatusEventDeleted(l, ctx, skillstatus2.StatusEvent[skillstatus2.StatusEventDeletedBody]{
		WorldId: world.Id(0), CharacterId: 42, SkillId: uint32(skill.DarkKnightBerserkId),
		Type: skillstatus2.StatusEventTypeDeleted,
	})

	_, err := berserk.GetRegistry().Get(ctx, 42)
	assert.ErrorIs(t, err, berserk.ErrNotFound)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./kafka/consumer/... -v`
Expected: FAIL — packages do not exist.

- [ ] **Step 3: Write the implementation**

`kafka/message/characterstatus/kafka.go`:

```go
// Package characterstatus mirrors the atlas-character status events this
// service consumes (source of truth:
// services/atlas-character/atlas.com/character/kafka/message/character/kafka.go).
// Only the consumed types/fields are mirrored; unknown event types on the
// topic are ignored by the handlers' type guards.
package characterstatus

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/stat"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	EnvEventTopicCharacterStatus  = "EVENT_TOPIC_CHARACTER_STATUS"
	StatusEventTypeLogin          = "LOGIN"
	StatusEventTypeLogout         = "LOGOUT"
	StatusEventTypeChannelChanged = "CHANNEL_CHANGED"
	StatusEventTypeMapChanged     = "MAP_CHANGED"
	StatusEventTypeStatChanged    = "STAT_CHANGED"
)

type StatusEvent[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	WorldId       world.Id  `json:"worldId"`
	CharacterId   uint32    `json:"characterId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type StatusEventLoginBody struct {
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
}

type StatusEventLogoutBody struct {
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
}

// StatusEventChannelChangedBody mirrors the producer's
// ChangeChannelEventLoginBody (the upstream name is historical; the JSON
// shape is the contract).
type StatusEventChannelChangedBody struct {
	ChannelId    channel.Id `json:"channelId"`
	OldChannelId channel.Id `json:"oldChannelId"`
	MapId        _map.Id    `json:"mapId"`
	Instance     uuid.UUID  `json:"instance"`
}

type StatusEventMapChangedBody struct {
	ChannelId      channel.Id `json:"channelId"`
	OldMapId       _map.Id    `json:"oldMapId"`
	OldInstance    uuid.UUID  `json:"oldInstance"`
	TargetMapId    _map.Id    `json:"targetMapId"`
	TargetInstance uuid.UUID  `json:"targetInstance"`
	TargetPortalId uint32     `json:"targetPortalId"`
}

// StatusEventStatChangedBody: Values is populated only for level-up/job flows
// and never carries current HP (verified, design §2) — HP is read via REST in
// the ticker's re-evaluation, so it is not mirrored here.
type StatusEventStatChangedBody struct {
	ChannelId       channel.Id  `json:"channelId"`
	ExclRequestSent bool        `json:"exclRequestSent"`
	Updates         []stat.Type `json:"updates"`
}
```

`kafka/message/skillstatus/kafka.go`:

```go
// Package skillstatus mirrors the atlas-skills status events this service
// consumes (source of truth:
// services/atlas-skills/atlas.com/skills/kafka/message/skill/kafka.go).
package skillstatus

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	EnvStatusEventTopic    = "EVENT_TOPIC_SKILL_STATUS"
	StatusEventTypeUpdated = "UPDATED"
	StatusEventTypeDeleted = "DELETED"
)

type StatusEvent[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	WorldId       world.Id  `json:"worldId"`
	CharacterId   uint32    `json:"characterId"`
	SkillId       uint32    `json:"skillId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type StatusEventUpdatedBody struct {
	Level       byte      `json:"level"`
	MasterLevel byte      `json:"masterLevel"`
	Expiration  time.Time `json:"expiration"`
}

type StatusEventDeletedBody struct{}
```

`kafka/consumer/characterstatus/consumer.go`:

```go
package characterstatus

import (
	"atlas-buffs/berserk"
	consumer2 "atlas-buffs/kafka/consumer"
	characterstatus2 "atlas-buffs/kafka/message/characterstatus"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("character_status_event")(characterstatus2.EnvEventTopicCharacterStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		t, _ = topic.EnvProvider(l)(characterstatus2.EnvEventTopicCharacterStatus)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventLogin))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventLogout))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventStatChanged))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventMapChanged))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventChannelChanged))); err != nil {
			return err
		}
		return nil
	}
}

func handleStatusEventLogin(l logrus.FieldLogger, ctx context.Context, e characterstatus2.StatusEvent[characterstatus2.StatusEventLoginBody]) {
	if e.Type != characterstatus2.StatusEventTypeLogin {
		return
	}
	if err := berserk.NewProcessor(l, ctx).TrackOnLogin(e.WorldId, e.Body.ChannelId, e.CharacterId); err != nil {
		l.WithError(err).Errorf("Unable to evaluate berserk tracking for character [%d] at login.", e.CharacterId)
	}
}

func handleStatusEventLogout(l logrus.FieldLogger, ctx context.Context, e characterstatus2.StatusEvent[characterstatus2.StatusEventLogoutBody]) {
	if e.Type != characterstatus2.StatusEventTypeLogout {
		return
	}
	if err := berserk.NewProcessor(l, ctx).Untrack(e.CharacterId); err != nil {
		l.WithError(err).Errorf("Unable to untrack berserk for character [%d] at logout.", e.CharacterId)
	}
}

func handleStatusEventStatChanged(l logrus.FieldLogger, ctx context.Context, e characterstatus2.StatusEvent[characterstatus2.StatusEventStatChangedBody]) {
	if e.Type != characterstatus2.StatusEventTypeStatChanged {
		return
	}
	if err := berserk.NewProcessor(l, ctx).HandleStatChanged(e.WorldId, e.Body.ChannelId, e.CharacterId, e.Body.Updates); err != nil {
		l.WithError(err).Errorf("Unable to process stat change for berserk tracking of character [%d].", e.CharacterId)
	}
}

func handleStatusEventMapChanged(l logrus.FieldLogger, ctx context.Context, e characterstatus2.StatusEvent[characterstatus2.StatusEventMapChangedBody]) {
	if e.Type != characterstatus2.StatusEventTypeMapChanged {
		return
	}
	if err := berserk.NewProcessor(l, ctx).HandleTransfer(e.WorldId, e.Body.ChannelId, e.CharacterId); err != nil {
		l.WithError(err).Errorf("Unable to process map change for berserk tracking of character [%d].", e.CharacterId)
	}
}

func handleStatusEventChannelChanged(l logrus.FieldLogger, ctx context.Context, e characterstatus2.StatusEvent[characterstatus2.StatusEventChannelChangedBody]) {
	if e.Type != characterstatus2.StatusEventTypeChannelChanged {
		return
	}
	if err := berserk.NewProcessor(l, ctx).HandleTransfer(e.WorldId, e.Body.ChannelId, e.CharacterId); err != nil {
		l.WithError(err).Errorf("Unable to process channel change for berserk tracking of character [%d].", e.CharacterId)
	}
}
```

`kafka/consumer/skillstatus/consumer.go`:

```go
package skillstatus

import (
	"atlas-buffs/berserk"
	consumer2 "atlas-buffs/kafka/consumer"
	skillstatus2 "atlas-buffs/kafka/message/skillstatus"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("skill_status_event")(skillstatus2.EnvStatusEventTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		t, _ = topic.EnvProvider(l)(skillstatus2.EnvStatusEventTopic)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventUpdated))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventDeleted))); err != nil {
			return err
		}
		return nil
	}
}

func handleStatusEventUpdated(l logrus.FieldLogger, ctx context.Context, e skillstatus2.StatusEvent[skillstatus2.StatusEventUpdatedBody]) {
	if e.Type != skillstatus2.StatusEventTypeUpdated {
		return
	}
	if e.SkillId != uint32(skill.DarkKnightBerserkId) {
		return
	}
	if err := berserk.NewProcessor(l, ctx).HandleSkillUpdated(e.WorldId, e.CharacterId, e.Body.Level); err != nil {
		l.WithError(err).Errorf("Unable to process berserk skill update for character [%d].", e.CharacterId)
	}
}

func handleStatusEventDeleted(l logrus.FieldLogger, ctx context.Context, e skillstatus2.StatusEvent[skillstatus2.StatusEventDeletedBody]) {
	if e.Type != skillstatus2.StatusEventTypeDeleted {
		return
	}
	if e.SkillId != uint32(skill.DarkKnightBerserkId) {
		return
	}
	if err := berserk.NewProcessor(l, ctx).Untrack(e.CharacterId); err != nil {
		l.WithError(err).Errorf("Unable to untrack berserk for character [%d] after skill deletion.", e.CharacterId)
	}
}
```

`main.go` — after the existing `character2.InitConsumers`/`InitHandlers` block add:

```go
	characterstatus2.InitConsumers(l)(cmf)(consumerGroupId)
	if err := characterstatus2.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
	skillstatus2.InitConsumers(l)(cmf)(consumerGroupId)
	if err := skillstatus2.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
```

with imports:

```go
	characterstatus2 "atlas-buffs/kafka/consumer/characterstatus"
	skillstatus2 "atlas-buffs/kafka/consumer/skillstatus"
```

Deploy note (verified, design §2): `EVENT_TOPIC_CHARACTER_STATUS` and `EVENT_TOPIC_SKILL_STATUS` are already in the shared configmap (`deploy/k8s/base/env-configmap.yaml`) and atlas-buffs inherits via `envFrom` — no manifest changes.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -race ./... && go build ./...`
Expected: PASS/clean.

- [ ] **Step 5: Commit**

```bash
git add kafka/message/characterstatus kafka/message/skillstatus kafka/consumer/characterstatus kafka/consumer/skillstatus main.go
git commit -m "feat(task-154): character-status and skill-status consumers driving berserk registry"
```

---

### Task 8: buff-origin max-HP hook in the character processor

**Files:**
- Create: `services/atlas-buffs/atlas.com/buffs/character/maxhp.go`
- Modify: `services/atlas-buffs/atlas.com/buffs/character/processor.go`
- Test: `services/atlas-buffs/atlas.com/buffs/character/maxhp_test.go`

**Interfaces:**
- Consumes: `berserk.NewProcessor(l, ctx).MarkMaxHpDirty(characterId)` (Task 6); `stat.Model` from `atlas-buffs/buff/stat`; `character.TemporaryStatTypeHyperBodyHP` from `libs/atlas-constants/character` (import-aliased — the local package is also named `character`).
- Produces: `affectsMaxHp(changes []stat.Model) bool` and `markBerserkDirtyOnMaxHpChange(l, ctx, characterId, changeSets ...[]stat.Model)` (both package-private), wired into `Apply`, `Cancel`, `CancelAll`, `CancelByStatTypes`, and `ExpireBuffs`.

No import cycle: `berserk` does not import `atlas-buffs/character`.

- [ ] **Step 1: Write the failing test**

`character/maxhp_test.go`:

```go
package character

import (
	"testing"
	"time"

	"atlas-buffs/berserk"
	"atlas-buffs/buff/stat"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	constants "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func setupBothRegistries(t *testing.T) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRegistry(client)
	berserk.InitRegistry(client)
}

func TestAffectsMaxHp(t *testing.T) {
	cases := []struct {
		name    string
		changes []stat.Model
		want    bool
	}{
		{name: "hyper body hp", changes: []stat.Model{stat.NewStat(string(constants.TemporaryStatTypeHyperBodyHP), 60)}, want: true},
		{name: "hyper body mp only", changes: []stat.Model{stat.NewStat(string(constants.TemporaryStatTypeHyperBodyMP), 60)}, want: false},
		{name: "plain stat buff", changes: []stat.Model{stat.NewStat("STR", 10)}, want: false},
		{name: "mixed includes max hp", changes: []stat.Model{stat.NewStat("STR", 10), stat.NewStat(string(constants.TemporaryStatTypeHyperBodyHP), 60)}, want: true},
		{name: "empty", changes: nil, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, affectsMaxHp(tc.changes))
		})
	}
}

func TestApplyHyperBodyMarksTrackedBerserkDirty(t *testing.T) {
	setupBothRegistries(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := logrus.New()

	assert.NoError(t, berserk.GetRegistry().Track(ctx,
		berserk.NewBuilder(world.Id(0), 42, 10).SetChannel(channel.Id(1)).Build()))

	changes := []stat.Model{stat.NewStat(string(constants.TemporaryStatTypeHyperBodyHP), 60)}
	assert.NoError(t, NewProcessor(l, ctx).Apply(world.Id(0), channel.Id(1), 42, 42, 1301007, 30, 10, changes, false))

	m, err := berserk.GetRegistry().Get(ctx, 42)
	assert.NoError(t, err)
	assert.False(t, m.DirtyAt().IsZero(), "hyper body apply marks berserk dirty")
	assert.False(t, m.DirtyDue(time.Now()), "grace-deferred: effective-stats must recompute first")
	assert.True(t, m.DirtyDue(time.Now().Add(berserk.ReevalGrace+time.Second)))
}

func TestCancelHyperBodyMarksTrackedBerserkDirty(t *testing.T) {
	setupBothRegistries(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := logrus.New()

	assert.NoError(t, berserk.GetRegistry().Track(ctx,
		berserk.NewBuilder(world.Id(0), 42, 10).SetChannel(channel.Id(1)).Build()))

	changes := []stat.Model{stat.NewStat(string(constants.TemporaryStatTypeHyperBodyHP), 60)}
	p := NewProcessor(l, ctx)
	assert.NoError(t, p.Apply(world.Id(0), channel.Id(1), 42, 42, 1301007, 30, 10, changes, false))

	// Clear the apply-time dirty mark so the cancel effect is observable.
	assert.NoError(t, berserk.GetRegistry().StoreEvaluation(ctx, 42, false, 100, time.Now()))
	_, claimed := berserk.GetRegistry().ClaimReeval(ctx, 42, time.Now().Add(berserk.ReevalGrace+time.Second))
	assert.True(t, claimed)

	assert.NoError(t, p.Cancel(world.Id(0), 42, 1301007))

	m, err := berserk.GetRegistry().Get(ctx, 42)
	assert.NoError(t, err)
	assert.False(t, m.DirtyAt().IsZero(), "hyper body cancel marks berserk dirty")
}

func TestApplyNonMaxHpBuffDoesNotMarkDirty(t *testing.T) {
	setupBothRegistries(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := logrus.New()

	assert.NoError(t, berserk.GetRegistry().Track(ctx,
		berserk.NewBuilder(world.Id(0), 42, 10).SetChannel(channel.Id(1)).Build()))

	changes := []stat.Model{stat.NewStat("STR", 10)}
	assert.NoError(t, NewProcessor(l, ctx).Apply(world.Id(0), channel.Id(1), 42, 42, 2001001, 30, 10, changes, false))

	m, err := berserk.GetRegistry().Get(ctx, 42)
	assert.NoError(t, err)
	assert.True(t, m.DirtyAt().IsZero())
}
```

Note: `1301007` (Hyper Body) and `2001001` here are buff **source ids in test fixtures**, consistent with the existing `registry_test.go` usage of literal source ids in this package's tests; the production code path carries them opaquely.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./character/... -run 'TestAffectsMaxHp|TestApplyHyper|TestCancelHyper|TestApplyNonMaxHp' -v`
Expected: FAIL — `undefined: affectsMaxHp`.

- [ ] **Step 3: Write the implementation**

`character/maxhp.go`:

```go
package character

import (
	"atlas-buffs/berserk"
	"atlas-buffs/buff/stat"
	"context"

	constants "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/sirupsen/logrus"
)

// maxHpBuffStatTypes mirrors the MapBuffStatType cases in atlas-effective-stats
// that resolve to its max-HP stat (services/atlas-effective-stats/atlas.com/
// effective-stats/stat/model.go — currently only HYPER_BODY_HP). Keep in sync
// if effective-stats grows new max-HP-affecting buff mappings.
var maxHpBuffStatTypes = map[string]bool{
	string(constants.TemporaryStatTypeHyperBodyHP): true,
}

func affectsMaxHp(changes []stat.Model) bool {
	for _, c := range changes {
		if maxHpBuffStatTypes[c.Type()] {
			return true
		}
	}
	return false
}

// markBerserkDirtyOnMaxHpChange schedules a grace-deferred berserk
// re-evaluation when any change set affects max HP. This service is the
// producer of the buff event atlas-effective-stats consumes to recompute max
// HP, so an immediate re-evaluation would read the stale value (design D5).
// Untracked characters are no-ops inside the berserk registry.
func markBerserkDirtyOnMaxHpChange(l logrus.FieldLogger, ctx context.Context, characterId uint32, changeSets ...[]stat.Model) {
	for _, cs := range changeSets {
		if affectsMaxHp(cs) {
			if err := berserk.NewProcessor(l, ctx).MarkMaxHpDirty(characterId); err != nil {
				l.WithError(err).Warnf("Unable to mark berserk dirty for character [%d].", characterId)
			}
			return
		}
	}
}
```

`character/processor.go` — five call sites. In `Apply`, replace the final `return message.Emit(...)` with:

```go
	err := message.Emit(p.l, p.ctx)(func(buf *message.Buffer) error {
		applied, err := GetRegistry().Apply(p.ctx, worldId, channelId, characterId, sourceId, level, duration, changes, accumulate)
		if err != nil {
			return err
		}
		// One APPLIED per stored buff: default mode returns a single whole-source
		// buff; accumulate mode returns one buff per stat, each carrying its own
		// changes/expiry so the channel sets (and later cancels) each stat icon
		// independently.
		for _, b := range applied {
			if err := buf.Put(character2.EnvEventStatusTopic, appliedStatusEventProvider(worldId, characterId, fromId, b.SourceId(), b.Level(), b.Duration(), b.Changes(), b.CreatedAt(), b.ExpiresAt())); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	markBerserkDirtyOnMaxHpChange(p.l, p.ctx, characterId, changes)
	return nil
```

In `Cancel`, after the `cancelled, err := ...` block succeeds, replace the final `return message.Emit(...)` with the same emit-then-hook shape, collecting change sets:

```go
	err = message.Emit(p.l, p.ctx)(func(buf *message.Buffer) error {
		for _, b := range cancelled {
			if err := buf.Put(character2.EnvEventStatusTopic, expiredStatusEventProvider(worldId, characterId, b.SourceId(), b.Level(), b.Duration(), b.Changes(), b.CreatedAt(), b.ExpiresAt())); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	sets := make([][]stat.Model, 0, len(cancelled))
	for _, b := range cancelled {
		sets = append(sets, b.Changes())
	}
	markBerserkDirtyOnMaxHpChange(p.l, p.ctx, characterId, sets...)
	return nil
```

In `CancelAll`, replace the final `return message.Emit(...)` with:

```go
	err := message.Emit(p.l, p.ctx)(func(buf *message.Buffer) error {
		for _, b := range buffs {
			if err := buf.Put(character2.EnvEventStatusTopic, expiredStatusEventProvider(worldId, characterId, b.SourceId(), b.Level(), b.Duration(), b.Changes(), b.CreatedAt(), b.ExpiresAt())); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	sets := make([][]stat.Model, 0, len(buffs))
	for _, b := range buffs {
		sets = append(sets, b.Changes())
	}
	markBerserkDirtyOnMaxHpChange(p.l, p.ctx, characterId, sets...)
	return nil
```

In `CancelByStatTypes`, replace the final `return message.Emit(...)` with (note `err =`, not `err :=` — `err` is already declared earlier in that method):

```go
	err = message.Emit(p.l, p.ctx)(func(buf *message.Buffer) error {
		for _, b := range cancelled {
			if err := buf.Put(character2.EnvEventStatusTopic, expiredStatusEventProvider(worldId, characterId, b.SourceId(), b.Level(), b.Duration(), b.Changes(), b.CreatedAt(), b.ExpiresAt())); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	sets := make([][]stat.Model, 0, len(cancelled))
	for _, b := range cancelled {
		sets = append(sets, b.Changes())
	}
	markBerserkDirtyOnMaxHpChange(p.l, p.ctx, characterId, sets...)
	return nil
```

In `ExpireBuffs`, add the hook inside the per-character loop, after the inner `for _, eb := range ebs` loop:

```go
			if len(ebs) > 0 {
				sets := make([][]stat.Model, 0, len(ebs))
				for _, eb := range ebs {
					sets = append(sets, eb.Changes())
				}
				markBerserkDirtyOnMaxHpChange(p.l, p.ctx, c.Id(), sets...)
			}
```

(Comment-preserving note: keep the existing comments in these methods exactly as they are; only the return shape changes.)

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -race ./... && go vet ./...`
Expected: PASS/clean — including all pre-existing `character` package tests (the emit-then-hook refactor must not change any existing behavior).

- [ ] **Step 5: Commit**

```bash
git add character/maxhp.go character/maxhp_test.go character/processor.go
git commit -m "feat(task-154): grace-deferred berserk re-evaluation on max-HP buff changes"
```

---

### Task 9: atlas-channel — event mirror, announce helpers, berserk handler

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/message/buff/kafka.go` (append at end)
- Create: `services/atlas-channel/atlas.com/channel/kafka/message/buff/kafka_test.go`
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/effects.go` (append after `AnnounceForeignSkillUse`)
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/buff/consumer.go` (new handler + registration)

**Interfaces:**
- Consumes: `BerserkStatusEventBody` JSON contract (Task 5); `charpkt.CharacterSkillUseEffectBody(skillId, characterLevel, skillLevel, darkForceEffect, createOrDeleteDragon, left)` and `charpkt.CharacterSkillUseEffectForeignBody(characterId, ...)` from `libs/atlas-packet/character/effect_body.go:62-84` — the **`darkForceEffect` bool is the on/off aura flag**, encoded as a trailing byte only when the skill id is `skill.DarkKnightBerserkId` (the packet lib derives that gate internally; byte fixtures for v83/v84/v87 already pin the wire format in `clientbound/effect_skill_use_test.go`); `charcb.CharacterEffectWriter`/`CharacterEffectForeignWriter` (registered for every tenant — no writer/template work); `sc.Is(t tenant.Model, worldId world.Id, channelId channel.Id) bool` (`server/model.go:49`); `session.NewProcessor(l, ctx).IfPresentByCharacterId(ch channel.Model)(characterId, f)` (`session/processor.go:106`); `_map.NewProcessor(l, ctx).ForOtherSessionsInMap(f field.Model, referenceCharacterId, o)` (`map/processor.go:67`).
- Produces: `sockethandler.AnnounceBerserkEffect(l)(ctx)(wp)(skillId uint32, characterLevel byte, skillLevel byte, active bool) model2.Operator[session.Model]` and `AnnounceForeignBerserkEffect(l)(ctx)(wp)(characterId uint32, skillId uint32, characterLevel byte, skillLevel byte, active bool) model2.Operator[session.Model]`; `handleStatusEventBerserk` registered in the buff consumer.

Import note: `kafka/consumer/buff/consumer.go` already imports `libs/atlas-kafka/handler` as `handler`, so import the socket handlers as `sockethandler "atlas-channel/socket/handler"` — precedent: `kafka/consumer/monster/consumer.go:14` uses `socketHandler`; use the same casing as that file (`socketHandler`). No import cycle: `socket/handler` does not import any `kafka/consumer` package.

Testing scope note (deviation from design §6, justified): atlas-channel has no harness for session-touching consumer handlers — existing consumer tests (e.g. `kafka/consumer/drop/consumer_test.go`) cover extracted pure helpers only. The handler here is a guard + two announce calls whose shape mirrors the already-shipped `handleStatusEventApplied`/`handleStatusEventLevelChanged`; the wire bytes are pinned by the packet lib fixtures. What IS testable and load-bearing is the **cross-service JSON contract** — the golden decode test below (emit-side twin lives in Task 5).

- [ ] **Step 1: Write the failing contract test**

`kafka/message/buff/kafka_test.go`:

```go
package buff

import (
	"encoding/json"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// berserkEventJSON is a golden fixture of what atlas-buffs'
// berserkStatusEventProvider puts on EVENT_TOPIC_CHARACTER_BUFF_STATUS
// (see atlas-buffs berserk/producer_test.go — the emit-side twin of this
// test). If either side's struct drifts, one of the two tests breaks.
const berserkEventJSON = `{"worldId":1,"characterId":42,"type":"BERSERK","body":{"transactionId":"11111111-2222-3333-4444-555555555555","channelId":3,"skillId":1320006,"characterLevel":135,"skillLevel":20,"active":true}}`

func TestBerserkStatusEventDecode(t *testing.T) {
	var e StatusEvent[BerserkStatusEventBody]
	assert.NoError(t, json.Unmarshal([]byte(berserkEventJSON), &e))

	assert.Equal(t, world.Id(1), e.WorldId)
	assert.Equal(t, uint32(42), e.CharacterId)
	assert.Equal(t, EventStatusTypeBerserk, e.Type)
	assert.Equal(t, uuid.MustParse("11111111-2222-3333-4444-555555555555"), e.Body.TransactionId)
	assert.Equal(t, channel.Id(3), e.Body.ChannelId)
	assert.Equal(t, uint32(skill.DarkKnightBerserkId), e.Body.SkillId)
	assert.Equal(t, byte(135), e.Body.CharacterLevel)
	assert.Equal(t, byte(20), e.Body.SkillLevel)
	assert.True(t, e.Body.Active)
}

func TestBerserkStatusEventDecodeInactive(t *testing.T) {
	inactive := `{"worldId":0,"characterId":7,"type":"BERSERK","body":{"transactionId":"11111111-2222-3333-4444-555555555555","channelId":1,"skillId":1320006,"characterLevel":200,"skillLevel":30,"active":false}}`
	var e StatusEvent[BerserkStatusEventBody]
	assert.NoError(t, json.Unmarshal([]byte(inactive), &e))
	assert.False(t, e.Body.Active, "inactive ticks clear the aura — they are broadcast too")
}
```

(The `1320006` inside the JSON string is wire-fixture data, asserted against `skill.DarkKnightBerserkId` — the same pattern the packet byte fixtures use.)

- [ ] **Step 2: Run test to verify it fails**

Run (from `services/atlas-channel/atlas.com/channel`): `go test ./kafka/message/buff/... -v`
Expected: FAIL — `undefined: BerserkStatusEventBody`.

- [ ] **Step 3: Write the implementation**

Append to `kafka/message/buff/kafka.go`:

```go
const (
	EventStatusTypeBerserk = "BERSERK"
)

// BerserkStatusEventBody mirrors atlas-buffs' berserk broadcast tick
// (task-154). One event per 3s tick per tracked Dark Knight; Active=false
// ticks clear the aura and keep late-joining observers consistent. ChannelId
// enables the precise sc.Is(tenant, world, channel) guard.
type BerserkStatusEventBody struct {
	TransactionId  uuid.UUID  `json:"transactionId"`
	ChannelId      channel.Id `json:"channelId"`
	SkillId        uint32     `json:"skillId"`
	CharacterLevel byte       `json:"characterLevel"`
	SkillLevel     byte       `json:"skillLevel"`
	Active         bool       `json:"active"`
}
```

(`uuid` and `channel` are already imported in that file.)

Append to `socket/handler/effects.go` after `AnnounceForeignSkillUse`:

```go
// AnnounceBerserkEffect is the self-facing CharacterEffect broadcast carrying
// the Dark Knight Berserk aura flag. Identical to AnnounceSkillUse except the
// darkForceEffect bool is threaded through: the packet encoder writes it as a
// trailing byte only for skill.DarkKnightBerserkId (effect_body.go derives
// that gate from the skill id).
func AnnounceBerserkEffect(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(skillId uint32, characterLevel byte, skillLevel byte, active bool) model2.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(skillId uint32, characterLevel byte, skillLevel byte, active bool) model2.Operator[session.Model] {
		return func(wp writer.Producer) func(skillId uint32, characterLevel byte, skillLevel byte, active bool) model2.Operator[session.Model] {
			return func(skillId uint32, characterLevel byte, skillLevel byte, active bool) model2.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(charcb.CharacterEffectWriter)(charpkt.CharacterSkillUseEffectBody(skillId, characterLevel, skillLevel, active, false, false))
			}
		}
	}
}

// AnnounceForeignBerserkEffect is the same broadcast targeted at other
// sessions on the Dark Knight's map.
func AnnounceForeignBerserkEffect(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(characterId uint32, skillId uint32, characterLevel byte, skillLevel byte, active bool) model2.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(characterId uint32, skillId uint32, characterLevel byte, skillLevel byte, active bool) model2.Operator[session.Model] {
		return func(wp writer.Producer) func(characterId uint32, skillId uint32, characterLevel byte, skillLevel byte, active bool) model2.Operator[session.Model] {
			return func(characterId uint32, skillId uint32, characterLevel byte, skillLevel byte, active bool) model2.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(charcb.CharacterEffectForeignWriter)(charpkt.CharacterSkillUseEffectForeignBody(characterId, skillId, characterLevel, skillLevel, active, false, false))
			}
		}
	}
}
```

In `kafka/consumer/buff/consumer.go`:

1. Add the import `socketHandler "atlas-channel/socket/handler"`.
2. In `InitHandlers`, after the `handleStatusEventExpired` registration block, add:

```go
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventBerserk(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
```

3. Append the handler:

```go
// handleStatusEventBerserk translates one berserk broadcast tick into the own
// + foreign EffectSkillUse packets (task-154). Stateless by design (D4):
// atlas-buffs owns the schedule; the periodic re-broadcast covers late-joining
// observers, so there is no map-enter hook. No session means the character
// transferred or logged out between emit and consume — the next tick
// self-corrects.
func handleStatusEventBerserk(sc server.Model, wp writer.Producer) message.Handler[buff2.StatusEvent[buff2.BerserkStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e buff2.StatusEvent[buff2.BerserkStatusEventBody]) {
		if e.Type != buff2.EventStatusTypeBerserk {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.Body.ChannelId) {
			return
		}

		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, func(s session.Model) error {
			if err := socketHandler.AnnounceBerserkEffect(l)(ctx)(wp)(e.Body.SkillId, e.Body.CharacterLevel, e.Body.SkillLevel, e.Body.Active)(s); err != nil {
				l.WithError(err).Errorf("Unable to write berserk effect for character [%d].", e.CharacterId)
			}

			_ = _map.NewProcessor(l, ctx).ForOtherSessionsInMap(s.Field(), s.CharacterId(), socketHandler.AnnounceForeignBerserkEffect(l)(ctx)(wp)(e.CharacterId, e.Body.SkillId, e.Body.CharacterLevel, e.Body.SkillLevel, e.Body.Active))
			return nil
		})
	}
}
```

No `main.go` changes: the buff consumer/handlers are already registered (`main.go:200` region and `main.go:500` region).

- [ ] **Step 4: Run tests to verify they pass**

Run (from `services/atlas-channel/atlas.com/channel`): `go test -race ./... && go vet ./... && go build ./...`
Expected: PASS/clean.

- [ ] **Step 5: Commit**

```bash
git add kafka/message/buff/kafka.go kafka/message/buff/kafka_test.go socket/handler/effects.go kafka/consumer/buff/consumer.go
git commit -m "feat(task-154): translate berserk ticks into own+foreign EffectSkillUse packets"
```

---

### Task 10: full verification suite

**Files:** none created — verification only. Fix-and-recommit anything that fails, then re-run the full suite from the top.

- [ ] **Step 1: atlas-buffs module checks**

Run (from `services/atlas-buffs/atlas.com/buffs`):

```bash
go test -race ./... && go vet ./... && go build ./...
```

Expected: all PASS/clean.

- [ ] **Step 2: atlas-channel module checks**

Run (from `services/atlas-channel/atlas.com/channel`):

```bash
go test -race ./... && go vet ./... && go build ./...
```

Expected: all PASS/clean.

- [ ] **Step 3: Docker bake both services (mandatory — go.mod-touching or not, both services changed)**

Run (from the worktree root):

```bash
docker buildx bake atlas-buffs atlas-channel
```

Expected: both images build. This is the only check that catches shared-Dockerfile `COPY libs/...` gaps; `go build` against `go.work` will not. No new shared lib was added, so no Dockerfile/go.work edits are expected — if bake fails on a missing lib COPY, fix the root `Dockerfile` per CLAUDE.md and re-bake.

- [ ] **Step 4: Redis key guard**

Run (from the worktree root, no global GOWORK=off prefix):

```bash
tools/redis-key-guard.sh
```

Expected: clean. All new Redis access is via `atlas.TenantRegistry`/`atlas.Set`.

- [ ] **Step 5: Acceptance criteria sweep**

Walk the PRD's acceptance criteria against the implementation (full sweep, not spot-check). The mapping:

| PRD AC | Where verified |
|---|---|
| Below-threshold activates within one tick; foreign variant to others | processor reeval + broadcast tests (Task 6); handler shape (Task 9) |
| Equality is inactive (strict `<`) | `TestEvaluate` boundary rows (Task 2) |
| Hyper Body apply/expire re-evaluates with HP constant | maxhp hook tests (Task 8) + `TestEvaluate` hyper-body rows (Task 2) |
| SP 0→1 tracks without relog; level change re-resolves x | `TestHandleUpdatedTracksBerserkSkill` (Task 7), `TestHandleSkillUpdated` (Task 6); x re-read per re-evaluation via cache (Task 4) |
| Login restores; logout stops; transfer re-routes | `TestTrackOnLogin*` (Task 6), `TestHandleLogout*`, `TestHandleMapChanged*`, `TestHandleChannelChanged*` (Task 7) |
| Map-enterer sees aura ≤3s with no HP event | periodic re-broadcast: `TestProcessTicksBroadcastAdvancesSchedule` (Task 6) |
| Level-0 characters: no entries/tickers/events | `TestTrackOnLoginSkillLevelZeroNotTracked` (Task 6), `TestHandleStatChangedUntrackedIsNoop` (Task 7) |
| 5s initial delay, 3s period, schedule replaced per re-evaluation | registry claim tests (Task 3), `TestProcessTicksReevaluates`/`TestProcessTicksReevalDoesNotBroadcastSamePass` (Task 6) |
| No literals: skill id / x / mode byte all resolved | grep the diff: `grep -rn '1320006' services/ --include='*.go'` must hit only test fixtures + atlas-constants references |
| Death stops aura; revive re-establishes | `TestEvaluate` hp=0 row (Task 2); revive = STAT_CHANGED(HP) path (Task 7) |
| Tenant isolation | `TestTenantIsolation` (Task 3), tenant-scoped cache test (Task 4) |
| Test suite + bake + guard clean | Steps 1–4 above |
| Builder-pattern tests, boundary + cancel-reschedule race covered | `TestConcurrentClaimSingleWinner` (Task 3), all setup via `NewBuilder` |

Expected: every row checks out; any gap is a finding to fix now, not to defer.

- [ ] **Step 6: Commit any verification fixes**

```bash
git status --short   # must be clean, or commit fixes with feat(task-154)/fix(task-154)
git log --oneline -12
git branch --show-current   # must be task-154-dark-knight-berserk
```

After this task: run `superpowers:requesting-code-review` (mandatory before any PR — CLAUDE.md), which dispatches plan-adherence-reviewer + backend-guidelines-reviewer over the branch.

---

## Execution Notes

- Tasks 1→8 are strictly ordered within atlas-buffs (each consumes the previous task's interfaces). Task 9 (atlas-channel) depends only on Task 5's JSON contract and can run in parallel with Tasks 6–8 if desired; Task 10 requires everything.
- The runtime BERSERK emission path cannot be exercised by unit tests end-to-end (producertest installs a no-op writer). The provider JSON (Task 5) + the channel-side golden decode (Task 9) pin the contract; the claim/schedule state machine is covered in Tasks 3/6. Live verification on a deployed env: drop a Dark Knight below the threshold and watch `atlas-channel` debug logs / the client aura, and `atlas-buffs` logs for `Tracking berserk`/state-transition lines.
- Known live-config caveat (bug-pattern memory): seed templates and live tenant configs need NO changes here — the SKILL_USE writer + operations mode are already wired for every version; the consumed/emitted topics already exist in the shared configmap. If a deployed tenant somehow lacks `EVENT_TOPIC_SKILL_STATUS` in its env, that is a deploy-env problem, not a code change.
- GM-hide (Cosmic broadcasts to GMs only while hidden) is an accepted follow-up per PRD §9.1 — the foreign broadcast here is unconditional. Do NOT add a hide check speculatively.

# Homing Beacon / Bullseye (task-167) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Casting Homing Beacon (5211006) / Bullseye (5220011) grants the attacker a non-expiring `HOMING_BEACON` buff carrying the struck monster's object id; the lock is replaced on re-cast, canceled on map change, and survives unrelated buff gives/cancels on the wire.

**Architecture:** Four subsystems change: atlas-buffs gains first-class no-expiry buffs and a `MAP_CHANGED` consumer; libs/atlas-packet gains a populated GuidedBullet block, accurate cancel masks with a version-gated movement-byte filter, and the verified v95 two-state group; atlas-channel gains the attack-handler hook, a beacon mirror registry, local-give merge, and foreign suppression. Spec: `docs/tasks/task-167-homing-beacon-bullseye/design.md` (all IDA/WZ evidence lives there, §2–§3).

**Tech Stack:** Go workspaces (`go.work`), Kafka via `libs/atlas-kafka`, Redis registry via `libs/atlas-redis`, packet byte-fixtures via `libs/atlas-packet/test`, testify + miniredis in atlas-buffs.

## Global Constraints

- Worktree: ALL work happens in `.worktrees/task-167-homing-beacon-bullseye` on branch `task-167-homing-beacon-bullseye`. Every subagent must `cd` there first and verify `git branch --show-current` after each commit.
- No magic sentinel durations: no-expiry is an explicit `noExpiry` flag end-to-end (PRD FR-2.1 owner decision).
- No client-interpreted byte hard-coded in a service; all version gating lives in libs/atlas-packet (DOM-25).
- Existing finite-duration buff behavior must be unchanged (FR-2.6); existing byte fixtures must stay green except where a task explicitly extends them.
- Monster object ids come from `libs/atlas-object-id` (MinId 1,000,000; MaxId 0x7FFFFFFF) — always nonzero, always fits `int32` (design §2.5).
- Skill ids: `skill.OutlawHomingBeaconId = Id(5211006)`, `skill.CorsairBullseyeId = Id(5220011)` (`libs/atlas-constants/skill/constants.go:3225,3236`). Stat type: `character.TemporaryStatTypeHomingBeacon` = `"HOMING_BEACON"`.
- No new topics, no DB migrations, no new libs → no Dockerfile / go.work changes.
- Test style: project Builder pattern / existing test idioms; no `*_testhelpers.go` files.
- Final gates: `go test -race ./...`, `go vet ./...`, `go build ./...` per changed module; `docker buildx bake atlas-channel atlas-buffs`; `tools/redis-key-guard.sh` from repo root.

---

### Task 1: atlas-buffs — no-expiry domain model

**Files:**
- Modify: `services/atlas-buffs/atlas.com/buffs/buff/model.go`
- Test: `services/atlas-buffs/atlas.com/buffs/buff/model_test.go`

**Interfaces:**
- Consumes: existing `stat.Model`, `NewBuff` (unchanged).
- Produces: `NewNoExpiryBuff(sourceId int32, level byte, changes []stat.Model) (Model, error)`; `(Model).NoExpiry() bool`; `(Model).Expired()` returns false when noExpiry; JSON round-trips `noExpiry` (omitempty). Tasks 2, 3 rely on these exact names.

- [ ] **Step 1: Write the failing tests** — append to `model_test.go` (existing `setupTestChanges()` is in this file):

```go
func TestNewNoExpiryBuff(t *testing.T) {
	b, err := NewNoExpiryBuff(int32(5211006), byte(1), setupTestChanges())

	assert.NoError(t, err)
	assert.True(t, b.NoExpiry())
	assert.Equal(t, int32(0), b.Duration())
	assert.True(t, b.ExpiresAt().IsZero())
	assert.False(t, b.Expired(), "no-expiry buff must never report expired despite zero expiresAt")
	assert.Len(t, b.Changes(), 2)
}

func TestNewNoExpiryBuff_EmptyChanges(t *testing.T) {
	_, err := NewNoExpiryBuff(int32(5211006), byte(1), []stat.Model{})
	assert.ErrorIs(t, err, ErrEmptyChanges)
}

func TestNewBuff_StillRejectsNonPositiveDuration(t *testing.T) {
	_, err := NewBuff(int32(2001001), byte(5), 0, setupTestChanges())
	assert.ErrorIs(t, err, ErrInvalidDuration)
	_, err = NewBuff(int32(2001001), byte(5), -1, setupTestChanges())
	assert.ErrorIs(t, err, ErrInvalidDuration)
}

func TestNoExpiryBuff_JSONRoundTrip(t *testing.T) {
	b, err := NewNoExpiryBuff(int32(5220011), byte(10), setupTestChanges())
	assert.NoError(t, err)

	data, err := json.Marshal(b)
	assert.NoError(t, err)

	var out Model
	assert.NoError(t, json.Unmarshal(data, &out))
	assert.True(t, out.NoExpiry())
	assert.False(t, out.Expired())
}

// A finite buff marshalled before this change has no noExpiry field; it must
// unmarshal to noExpiry=false so previously Redis-persisted buffs are unaffected.
func TestFiniteBuff_JSONAbsentNoExpiryDefaultsFalse(t *testing.T) {
	b, err := NewBuff(int32(2001001), byte(5), 60000, setupTestChanges())
	assert.NoError(t, err)

	data, err := json.Marshal(b)
	assert.NoError(t, err)
	assert.NotContains(t, string(data), "noExpiry", "omitempty must keep finite-buff JSON unchanged")

	var out Model
	assert.NoError(t, json.Unmarshal(data, &out))
	assert.False(t, out.NoExpiry())
}
```

Add `"encoding/json"` to the test file's imports.

- [ ] **Step 2: Run tests to verify they fail**

Run (from the worktree): `cd services/atlas-buffs/atlas.com/buffs && go test ./buff/ -run 'NoExpiry|StillRejects' -v`
Expected: FAIL — `undefined: NewNoExpiryBuff`, `b.NoExpiry undefined`.

- [ ] **Step 3: Implement** — in `buff/model.go`:

Add field to `Model`:

```go
type Model struct {
	id        uuid.UUID
	sourceId  int32
	level     byte
	duration  int32
	changes   []stat.Model
	createdAt time.Time
	expiresAt time.Time
	noExpiry  bool
}
```

Add accessor and change `Expired()`:

```go
// NoExpiry reports whether this buff never expires on its own (e.g. the
// HOMING_BEACON lock). The expiration ticker must never reap it; it is
// removed only by explicit cancel flows.
func (m Model) NoExpiry() bool {
	return m.noExpiry
}

func (m Model) Expired() bool {
	if m.noExpiry {
		return false
	}
	return m.expiresAt.Before(time.Now())
}
```

Add constructor (below `NewBuff`; `NewBuff` itself is unchanged):

```go
// NewNoExpiryBuff builds a buff that never expires on its own. duration is 0
// and expiresAt is the zero time; Expired() short-circuits on the flag so the
// zero expiresAt is never consulted (FR-2.4).
func NewNoExpiryBuff(sourceId int32, level byte, changes []stat.Model) (Model, error) {
	if len(changes) == 0 {
		return Model{}, ErrEmptyChanges
	}
	return Model{
		id:        uuid.New(),
		sourceId:  sourceId,
		level:     level,
		duration:  0,
		changes:   changes,
		createdAt: time.Now(),
		expiresAt: time.Time{},
		noExpiry:  true,
	}, nil
}
```

Extend both JSON structs (Marshal and Unmarshal aux) with:

```go
	NoExpiry  bool         `json:"noExpiry,omitempty"`
```

and wire `NoExpiry: m.noExpiry` in `MarshalJSON` / `m.noExpiry = aux.NoExpiry` in `UnmarshalJSON`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./buff/ -v` (whole package — existing tests must stay green).
Expected: PASS, no failures in pre-existing tests.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-buffs/atlas.com/buffs/buff/model.go services/atlas-buffs/atlas.com/buffs/buff/model_test.go
git commit -m "feat(buffs): explicit no-expiry buff model (task-167 FR-2)"
```

---

### Task 2: atlas-buffs — noExpiry through registry, processor, Kafka contract

**Files:**
- Modify: `services/atlas-buffs/atlas.com/buffs/character/registry.go`
- Modify: `services/atlas-buffs/atlas.com/buffs/character/processor.go`
- Modify: `services/atlas-buffs/atlas.com/buffs/character/producer.go`
- Modify: `services/atlas-buffs/atlas.com/buffs/kafka/message/character/kafka.go`
- Modify: `services/atlas-buffs/atlas.com/buffs/kafka/consumer/character/consumer.go`
- Test: `services/atlas-buffs/atlas.com/buffs/character/registry_test.go`, `services/atlas-buffs/atlas.com/buffs/character/processor_test.go`

**Interfaces:**
- Consumes: `buff.NewNoExpiryBuff`, `buff.Model.NoExpiry()` (Task 1).
- Produces:
  - `Registry.Apply(ctx, worldId world.Id, channelId channel.Id, characterId uint32, sourceId int32, level byte, duration int32, changes []stat.Model, accumulate bool, noExpiry bool) ([]buff.Model, error)` — trailing `noExpiry` param appended.
  - `Processor.Apply(worldId world.Id, channelId channel.Id, characterId uint32, fromId uint32, sourceId int32, level byte, duration int32, changes []stat.Model, accumulate bool, noExpiry bool) error` — trailing `noExpiry` param appended.
  - Kafka contract: `ApplyCommandBody.NoExpiry bool` (`json:"noExpiry,omitempty"`), `AppliedStatusEventBody.NoExpiry`, `ExpiredStatusEventBody.NoExpiry` (same tag). Task 9 mirrors these in atlas-channel.
  - Event providers gain a trailing `noExpiry bool` param.
- There is no mock of `character.Processor` in this repo (verified: `services/atlas-buffs/atlas.com/buffs/character/` has no mock dir); the only callers of `Apply` are the consumer handler and tests.

- [ ] **Step 1: Write the failing tests**

Append to `character/registry_test.go` (uses this file's existing `setupTestRegistry` / `setupTestTenant` / `setupTestContext` / `setupTestChanges` helpers):

```go
func TestRegistry_ApplyNoExpiry(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)

	changes := []stat.Model{stat.NewStat("HOMING_BEACON", 1000001)}
	applied, err := GetRegistry().Apply(ctx, world.Id(0), channel.Id(0), uint32(1000), int32(5211006), byte(1), 0, changes, false, true)

	assert.NoError(t, err)
	assert.Len(t, applied, 1)
	assert.True(t, applied[0].NoExpiry())
	assert.False(t, applied[0].Expired())
}

// A reap pass must remove an already-expired finite buff sitting next to a
// no-expiry buff, and must keep the no-expiry buff (FR-2.3 regression).
func TestRegistry_GetExpiredKeepsNoExpiry(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	characterId := uint32(1001)

	// Finite 1ms buff → expired after the sleep below.
	_, err := GetRegistry().Apply(ctx, world.Id(0), channel.Id(0), characterId, int32(2001001), byte(5), 1, setupTestChanges(), false, false)
	assert.NoError(t, err)
	// No-expiry beacon.
	_, err = GetRegistry().Apply(ctx, world.Id(0), channel.Id(0), characterId, int32(5211006), byte(1), 0, []stat.Model{stat.NewStat("HOMING_BEACON", 1000001)}, false, true)
	assert.NoError(t, err)

	time.Sleep(5 * time.Millisecond)
	expired := GetRegistry().GetExpired(ctx, characterId)

	assert.Len(t, expired, 1)
	assert.Equal(t, int32(2001001), expired[0].SourceId())

	m, err := GetRegistry().Get(ctx, characterId)
	assert.NoError(t, err)
	assert.Len(t, m.Buffs(), 1)
}

func TestRegistry_CancelByStatTypesRemovesNoExpiry(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	characterId := uint32(1002)

	_, err := GetRegistry().Apply(ctx, world.Id(0), channel.Id(0), characterId, int32(5211006), byte(1), 0, []stat.Model{stat.NewStat("HOMING_BEACON", 1000001)}, false, true)
	assert.NoError(t, err)

	cancelled, err := GetRegistry().CancelByStatTypes(ctx, characterId, map[string]bool{"HOMING_BEACON": true})
	assert.NoError(t, err)
	assert.Len(t, cancelled, 1)
	assert.True(t, cancelled[0].NoExpiry())
}

// FR-3.4 evidence: whole-character cancel removes a no-expiry buff too.
func TestRegistry_CancelAllRemovesNoExpiry(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	characterId := uint32(1003)

	_, err := GetRegistry().Apply(ctx, world.Id(0), channel.Id(0), characterId, int32(5220011), byte(10), 0, []stat.Model{stat.NewStat("HOMING_BEACON", 1000001)}, false, true)
	assert.NoError(t, err)

	all := GetRegistry().CancelAll(ctx, characterId)
	assert.Len(t, all, 1)

	m, err := GetRegistry().Get(ctx, characterId)
	assert.NoError(t, err)
	assert.Len(t, m.Buffs(), 0)
}
```

If `Model` (in `character/model.go`) has no exported `Buffs()` accessor, check the existing tests for how buff counts are asserted (registry_test.go asserts on `applied`/`Get` today) and use the same accessor; add nothing new unless the file already exports one.

- [ ] **Step 2: Update ALL existing `Registry.Apply(...)` / `Processor.Apply(...)` call sites in tests**

Every existing call in `registry_test.go` / `processor_test.go` gets a trailing `, false` argument. Do this mechanically:

```bash
grep -n "Apply(ctx\|\.Apply(" services/atlas-buffs/atlas.com/buffs/character/*_test.go
```

and append `, false` to each call's argument list.

- [ ] **Step 3: Run tests to verify they fail**

Run: `cd services/atlas-buffs/atlas.com/buffs && go test ./character/ -run 'NoExpiry|CancelAllRemoves' -v`
Expected: FAIL — wrong number of arguments to `Apply` (compile error) until Step 4.

- [ ] **Step 4: Implement**

`character/registry.go` — extend `Apply` signature and use the flag:

```go
func (r *Registry) Apply(ctx context.Context, worldId world.Id, channelId channel.Id, characterId uint32, sourceId int32, level byte, duration int32, changes []stat.Model, accumulate bool, noExpiry bool) ([]buff.Model, error) {
```

Replace the two `buff.NewBuff(...)` construction sites with a small helper used by both branches:

```go
	newBuff := func(cs []stat.Model) (buff.Model, error) {
		if noExpiry {
			return buff.NewNoExpiryBuff(sourceId, level, cs)
		}
		return buff.NewBuff(sourceId, level, duration, cs)
	}
```

so the accumulate branch calls `newBuff([]stat.Model{c})` and the default branch calls `newBuff(changes)`. Everything else in `Apply` is unchanged.

`character/processor.go` — extend the interface method and impl:

```go
	Apply(worldId world.Id, channelId channel.Id, characterId uint32, fromId uint32, sourceId int32, level byte, duration int32, changes []stat.Model, accumulate bool, noExpiry bool) error
```

`ProcessorImpl.Apply` passes `noExpiry` through to `GetRegistry().Apply(...)` and passes `b.NoExpiry()` to the event provider (below).

`kafka/message/character/kafka.go` — add fields:

```go
type ApplyCommandBody struct {
	FromId   uint32       `json:"fromId"`
	SourceId int32        `json:"sourceId"`
	Level    byte         `json:"level"`
	Duration int32        `json:"duration"`
	Changes  []StatChange `json:"changes"`
	Accumulate bool `json:"accumulate,omitempty"`
	// NoExpiry marks an explicitly non-expiring buff (task-167 FR-2). When set,
	// Duration MUST be 0; the consumer rejects the command otherwise.
	NoExpiry bool `json:"noExpiry,omitempty"`
}
```

(keep the existing Accumulate doc comment as-is; only add the NoExpiry field + comment). Add `NoExpiry bool \`json:"noExpiry,omitempty"\`` to `AppliedStatusEventBody` and `ExpiredStatusEventBody` as well.

`character/producer.go` — both providers gain a trailing `noExpiry bool` parameter and set `NoExpiry: noExpiry` in their event bodies:

```go
func appliedStatusEventProvider(worldId world.Id, characterId uint32, fromId uint32, sourceId int32, level byte, duration int32, changes []stat.Model, createdAt time.Time, expiresAt time.Time, noExpiry bool) model.Provider[[]kafka.Message] {
```

```go
func expiredStatusEventProvider(worldId world.Id, characterId uint32, sourceId int32, level byte, duration int32, changes []stat.Model, createdAt time.Time, expiresAt time.Time, noExpiry bool) model.Provider[[]kafka.Message] {
```

Update ALL 5 call sites in `character/processor.go` (`Apply`, `Cancel`, `CancelAll`, `CancelByStatTypes`, `ExpireBuffs`) to pass `b.NoExpiry()` (or `eb.NoExpiry()` in `ExpireBuffs`) as the trailing argument.

`kafka/consumer/character/consumer.go` — `handleApply` validates the malformed combination before calling the processor (design §5.7):

```go
	if c.Body.NoExpiry && c.Body.Duration != 0 {
		l.Warnf("Rejecting malformed APPLY for character [%d] source [%d]: noExpiry with nonzero duration [%d].", c.CharacterId, c.Body.SourceId, c.Body.Duration)
		return
	}
```

and passes `c.Body.NoExpiry` as the trailing argument to `Apply`.

- [ ] **Step 5: Run the full package tests**

Run: `go test -race ./...` from `services/atlas-buffs/atlas.com/buffs`.
Expected: PASS (new tests + all pre-existing tests).

- [ ] **Step 6: Commit**

```bash
git add services/atlas-buffs/atlas.com/buffs
git commit -m "feat(buffs): thread noExpiry through registry, processor, and Kafka contract (task-167 FR-2)"
```

---

### Task 3: atlas-buffs — REST projection of no-expiry

**Files:**
- Modify: `services/atlas-buffs/atlas.com/buffs/buff/rest.go`
- Test: `services/atlas-buffs/atlas.com/buffs/buff/rest_test.go`

**Interfaces:**
- Consumes: `Model.NoExpiry()` (Task 1).
- Produces: `RestModel.NoExpiry bool` (`json:"noExpiry"`) — a REST reader can tell the buff does not expire (FR-2.5). Task 9 adds the matching field to atlas-channel's mirror RestModel.

- [ ] **Step 1: Write the failing test** — append to `rest_test.go` (follow the file's existing Transform test style; read it first):

```go
func TestTransform_NoExpiry(t *testing.T) {
	b, err := NewNoExpiryBuff(int32(5211006), byte(1), []stat.Model{stat.NewStat("HOMING_BEACON", 1000001)})
	assert.NoError(t, err)

	rm, err := Transform(b)
	assert.NoError(t, err)
	assert.True(t, rm.NoExpiry)
	assert.True(t, rm.ExpiresAt.IsZero())
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./buff/ -run TestTransform_NoExpiry -v`
Expected: FAIL — `rm.NoExpiry undefined`.

- [ ] **Step 3: Implement** — in `buff/rest.go` add to `RestModel`:

```go
	NoExpiry  bool             `json:"noExpiry"`
```

and in `Transform` set `NoExpiry: m.noExpiry`.

- [ ] **Step 4: Run tests**

Run: `go test ./buff/ -v` — Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-buffs/atlas.com/buffs/buff/rest.go services/atlas-buffs/atlas.com/buffs/buff/rest_test.go
git commit -m "feat(buffs): expose noExpiry on buff REST model (task-167 FR-2.5)"
```

---

### Task 4: atlas-buffs — MAP_CHANGED consumer cancels the beacon

**Files:**
- Create: `services/atlas-buffs/atlas.com/buffs/kafka/consumer/charstatus/kafka.go`
- Create: `services/atlas-buffs/atlas.com/buffs/kafka/consumer/charstatus/consumer.go`
- Create: `services/atlas-buffs/atlas.com/buffs/kafka/consumer/charstatus/consumer_test.go`
- Create: `services/atlas-buffs/atlas.com/buffs/kafka/consumer/charstatus/testmain_test.go`
- Modify: `services/atlas-buffs/atlas.com/buffs/main.go`

**Interfaces:**
- Consumes: `character.NewProcessor(l, ctx).CancelByStatTypes(worldId, characterId, types)` (existing), `buff.Model` via registry (Tasks 1–2).
- Produces: `charstatus.InitConsumers(l)(cmf)(consumerGroupId)` and `charstatus.InitHandlers(l)(rf)` registered from `main.go`.
- Deployment: **no k8s change needed.** atlas-buffs consumes the shared `atlas-env` ConfigMap via `envFrom` (`deploy/k8s/base/atlas-buffs.yaml`), and `EVENT_TOPIC_CHARACTER_STATUS` is already defined there (`deploy/k8s/base/env-configmap.yaml:94`). The design's §5.4 assumption that the deployment needs a new env var is corrected here — verified against the manifests.

- [ ] **Step 1: Create `kafka.go`** — local re-declaration mirroring `services/atlas-summons/atlas.com/summons/kafka/consumer/character/kafka.go` (envelope has `TransactionId`, unlike the buff-status envelope):

```go
package charstatus

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	EnvEventTopicCharacterStatus = "EVENT_TOPIC_CHARACTER_STATUS"
	StatusEventTypeMapChanged    = "MAP_CHANGED"
)

// StatusEvent mirrors the atlas-character EVENT_TOPIC_CHARACTER_STATUS
// envelope (see atlas-maps kafka/message/character/kafka.go). Only the fields
// the beacon cancel needs are consumed; the body is decoded faithfully to
// avoid Kafka parse errors.
type StatusEvent[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	WorldId       world.Id  `json:"worldId"`
	CharacterId   uint32    `json:"characterId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type MapChangedBody struct {
	ChannelId      channel.Id `json:"channelId"`
	OldMapId       _map.Id    `json:"oldMapId"`
	OldInstance    uuid.UUID  `json:"oldInstance"`
	TargetMapId    _map.Id    `json:"targetMapId"`
	TargetInstance uuid.UUID  `json:"targetInstance"`
	TargetPortalId uint32     `json:"targetPortalId"`
}
```

- [ ] **Step 2: Write the failing handler test** — `consumer_test.go`. The handler is exercised directly with a miniredis-backed registry and noop producer, mirroring `character/registry_test.go` setup:

```go
package charstatus

import (
	"atlas-buffs/buff/stat"
	"atlas-buffs/character"
	"context"
	"testing"

	charconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func setupTest(t *testing.T) context.Context {
	t.Helper()
	mr := miniredis.RunT(t)
	character.InitRegistry(goredis.NewClient(&goredis.Options{Addr: mr.Addr()}))
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	assert.NoError(t, err)
	return tenant.WithContext(context.Background(), ten)
}

func seedBeacon(t *testing.T, ctx context.Context, characterId uint32) {
	t.Helper()
	_, err := character.GetRegistry().Apply(ctx, world.Id(0), channel.Id(0), characterId, int32(5211006), byte(1),
		0, []stat.Model{stat.NewStat(string(charconst.TemporaryStatTypeHomingBeacon), 1000001)}, false, true)
	assert.NoError(t, err)
}

func TestHandleMapChanged_CancelsBeacon(t *testing.T) {
	ctx := setupTest(t)
	characterId := uint32(2000)
	seedBeacon(t, ctx, characterId)

	handleMapChanged(logrus.StandardLogger(), ctx, StatusEvent[MapChangedBody]{
		WorldId: world.Id(0), CharacterId: characterId, Type: StatusEventTypeMapChanged,
	})

	m, err := character.GetRegistry().Get(ctx, characterId)
	assert.NoError(t, err)
	assert.Len(t, m.Buffs(), 0)
}

// A non-MAP_CHANGED event on the same topic must be ignored.
func TestHandleMapChanged_GuardsType(t *testing.T) {
	ctx := setupTest(t)
	characterId := uint32(2001)
	seedBeacon(t, ctx, characterId)

	handleMapChanged(logrus.StandardLogger(), ctx, StatusEvent[MapChangedBody]{
		WorldId: world.Id(0), CharacterId: characterId, Type: "LOGOUT",
	})

	m, err := character.GetRegistry().Get(ctx, characterId)
	assert.NoError(t, err)
	assert.Len(t, m.Buffs(), 1)
}

// Only HOMING_BEACON is canceled on map change — other buffs survive.
func TestHandleMapChanged_LeavesOtherBuffs(t *testing.T) {
	ctx := setupTest(t)
	characterId := uint32(2002)
	seedBeacon(t, ctx, characterId)
	_, err := character.GetRegistry().Apply(ctx, world.Id(0), channel.Id(0), characterId, int32(2001001), byte(5),
		60000, []stat.Model{stat.NewStat("SPEED", 20)}, false, false)
	assert.NoError(t, err)

	handleMapChanged(logrus.StandardLogger(), ctx, StatusEvent[MapChangedBody]{
		WorldId: world.Id(0), CharacterId: characterId, Type: StatusEventTypeMapChanged,
	})

	m, err := character.GetRegistry().Get(ctx, characterId)
	assert.NoError(t, err)
	assert.Len(t, m.Buffs(), 1)
	for _, b := range m.Buffs() {
		assert.Equal(t, int32(2001001), b.SourceId())
	}
}
```

Note: `character.Model.Buffs()` returns `map[string]buff.Model` (`character/model.go:26`) — `assert.Len` works on maps; iterate to inspect entries.

`testmain_test.go` (same shape as `character/testmain_test.go`):

```go
package charstatus

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

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./kafka/consumer/charstatus/ -v`
Expected: FAIL — `undefined: handleMapChanged`.

- [ ] **Step 4: Implement `consumer.go`** — mirrors `atlas-summons/.../kafka/consumer/character/consumer.go` registration idiom, but registers only the MAP_CHANGED handler (design §5.4: no other stat gains map-change semantics):

```go
package charstatus

import (
	"atlas-buffs/character"
	consumer2 "atlas-buffs/kafka/consumer"
	"context"

	charconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("character_status_event")(EnvEventTopicCharacterStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		t, _ := topic.EnvProvider(l)(EnvEventTopicCharacterStatus)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleMapChanged))); err != nil {
			return err
		}
		return nil
	}
}

// handleMapChanged cancels any active HOMING_BEACON lock when the character
// completes a map transition (Cosmic PlayerMapTransitionHandler parity —
// design.md §2.2). Errors log-and-continue; the next map change or a
// logout/death cancel-all is the safety net (design §5.7).
func handleMapChanged(l logrus.FieldLogger, ctx context.Context, e StatusEvent[MapChangedBody]) {
	if e.Type != StatusEventTypeMapChanged {
		return
	}
	l.Debugf("Character [%d] changed maps [%d] -> [%d]; canceling HOMING_BEACON if present.", e.CharacterId, e.Body.OldMapId, e.Body.TargetMapId)
	if err := character.NewProcessor(l, ctx).CancelByStatTypes(e.WorldId, e.CharacterId, []string{string(charconst.TemporaryStatTypeHomingBeacon)}); err != nil {
		l.WithError(err).Errorf("Unable to cancel HOMING_BEACON for character [%d] on map change.", e.CharacterId)
	}
}
```

- [ ] **Step 5: Wire into `main.go`** — after the existing `character2.InitHandlers` block:

```go
	charstatus.InitConsumers(l)(cmf)(consumerGroupId)
	if err := charstatus.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
```

with import `"atlas-buffs/kafka/consumer/charstatus"`.

- [ ] **Step 6: Run tests + build**

Run: `go test -race ./... && go build ./...` from `services/atlas-buffs/atlas.com/buffs`.
Expected: PASS / clean.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-buffs/atlas.com/buffs
git commit -m "feat(buffs): cancel HOMING_BEACON on MAP_CHANGED (task-167 FR-3.1)"
```

---

### Task 5: atlas-packet — populated GuidedBullet block (pre-95 versions)

**Files:**
- Modify: `libs/atlas-packet/model/character_temporary_stat.go`
- Test: `libs/atlas-packet/model/character_temporary_stat_test.go`

**Interfaces:**
- Consumes: existing `GuidedBulletTemporaryStat` (17-byte block: base 13 + dwMobId 4), `twoStateGuidedBullet` kind, `m.stats` map.
- Produces: `NewGuidedBulletTemporaryStatWithOptions(nOption int32, rOption int32, dwMobId uint32) GuidedBulletTemporaryStat`; `getBaseTemporaryStats` emits it when `HOMING_BEACON` is active. Field mapping (IDA, design §2.3/§2.4): nOption = monster object id (nonzero satisfies `IsActivated`), rOption = skill id (passed to `CMob::SetGuided` as reason), dwMobId = monster object id.

- [ ] **Step 1: Write the failing tests** — append to `character_temporary_stat_test.go`:

```go
// TestCTSHomingBeaconPre95PopulatedBlock pins the populated GuidedBullet block
// for the classic 7-member two-state group (v83/v84/v87/JMS). The block is
// nOption=mobId | rOption=skillId | 5-byte time | dwMobId=mobId — 17 bytes,
// same size as the empty block, so total packet length is unchanged and the
// two-state mask bits (always set pre-95) are unchanged.
func TestCTSHomingBeaconPre95PopulatedBlock(t *testing.T) {
	pre95 := []struct {
		name    string
		region  string
		major   uint16
	}{
		{"GMS v83", "GMS", 83},
		{"GMS v84", "GMS", 84},
		{"GMS v87", "GMS", 87},
		{"JMS v185", "JMS", 185},
	}
	for _, v := range pre95 {
		t.Run(v.name, func(t *testing.T) {
			ctx := pt.CreateContext(v.region, v.major, 1)
			tn, _ := tenant.Create([16]byte{}, v.region, v.major, 1)
			input := NewCharacterTemporaryStat()
			// mobId 1000001 (0x000F4241), skill 5211006 (0x004F837E).
			input.AddStat(nil)(tn)(string(character.TemporaryStatTypeHomingBeacon), 5211006, 1000001, 1, time.Time{})

			got := input.Encode(nil, ctx)(nil)

			// nOption=1000001 then rOption=5211006 as consecutive LE int32s.
			head := []byte{0x41, 0x42, 0x0F, 0x00, 0x7E, 0x83, 0x4F, 0x00}
			idx := bytes.Index(got, head)
			if idx < 0 {
				t.Fatalf("populated GuidedBullet head (nOption=1000001,rOption=5211006) missing; got % x", got)
			}
			// dwMobId sits after the 5-byte DecodeTime: head(8) + time(5) = offset 13.
			mob := got[idx+13 : idx+17]
			if !bytes.Equal(mob, []byte{0x41, 0x42, 0x0F, 0x00}) {
				t.Fatalf("dwMobId: got % x want 41 42 0f 00", mob)
			}
		})
	}
}

// Without an active beacon the encode must stay byte-compatible with today:
// the GuidedBullet slot still emits an empty 17-byte block (nOption=0).
func TestCTSHomingBeaconPre95AbsentStaysEmpty(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	input := NewCharacterTemporaryStat()

	got := input.Encode(nil, ctx)(nil)

	// Empty CTS on v83: 16 mask + 2 leading + 7 base blocks
	// (15+15+15+13+20+17+15 = 110).
	if len(got) != 16+2+110 {
		t.Fatalf("empty v83 CTS length: got %d want %d", len(got), 16+2+110)
	}
}
```

Note: `AddStat` with a zero `expiresAt` is fine — the GuidedBullet block never encodes expiry (NoExpire client type, design §2.3).

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd libs/atlas-packet && go test ./model/ -run TestCTSHomingBeacon -v`
Expected: `TestCTSHomingBeaconPre95PopulatedBlock` FAILs (block is always empty today); `AbsentStaysEmpty` PASSes (pins status quo).

- [ ] **Step 3: Implement**

Add the constructor next to `NewGuidedBulletTemporaryStat`:

```go
// NewGuidedBulletTemporaryStatWithOptions builds a populated GuidedBullet
// block for an active HOMING_BEACON lock. nOption must be nonzero — the
// client's set path gates on IsActivated (nValue != 0) before calling
// CMob::SetGuided (IDA v83 @0xA202BE, v95 @0xA02FC0; design.md §2.3/§2.4).
func NewGuidedBulletTemporaryStatWithOptions(nOption int32, rOption int32, dwMobId uint32) GuidedBulletTemporaryStat {
	return GuidedBulletTemporaryStat{
		CharacterTemporaryStatBase: CharacterTemporaryStatBase{
			bDynamicTermSet: false,
			nOption:         nOption,
			rOption:         rOption,
			tLastUpdated:    time.Now().Unix(),
		},
		dwMobId: dwMobId,
	}
}
```

In `getBaseTemporaryStats`, replace the `twoStateGuidedBullet` case:

```go
		case twoStateGuidedBullet:
			// GuidedBullet / HOMING_BEACON: nOption = locked monster object id
			// (allocator range guarantees nonzero — IsActivated gate), rOption =
			// source skill id (SetGuided reason + icon), dwMobId = monster object
			// id. Absent stat -> empty block, byte-identical to the pre-task
			// encode. design.md §5.5.1.
			if s, ok := m.stats[bs.name]; ok {
				list = append(list, NewGuidedBulletTemporaryStatWithOptions(s.Value(), s.SourceId(), uint32(s.Value())))
			} else {
				list = append(list, NewGuidedBulletTemporaryStat()) // 17
			}
```

- [ ] **Step 4: Run the full lib tests**

Run: `go test -race ./...` from `libs/atlas-packet`.
Expected: PASS — all existing fixtures (mount, disease, v95 truncation) stay green.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-packet/model/character_temporary_stat.go libs/atlas-packet/model/character_temporary_stat_test.go
git commit -m "feat(packet): populate GuidedBullet block from active HOMING_BEACON (task-167 FR-4.1)"
```

---

### Task 6: atlas-packet — v95 two-state group extension (closes Task 41b / FR-4.3)

**Files:**
- Modify: `libs/atlas-packet/model/character_temporary_stat.go`
- Test: `libs/atlas-packet/model/character_temporary_stat_test.go`

**Interfaces:**
- Consumes: `NewGuidedBulletTemporaryStatWithOptions` (Task 5), existing `SpeedInfusionTemporaryStat` (its 20-byte wire shape — base 13 + tCurrentTime 5 + int16 2 — is exactly the IDA-verified v95 PartyBooster block, design §2.4 slot 4; reused with a comment, no new struct).
- Produces: v95 `twoStateBaseStats` = 4 unconditional members (status quo) + `PartyBooster` (kind `twoStatePartyBooster`, conditional) + `HomingBeacon` (kind `twoStateGuidedBullet`, conditional). Conditional = block written and mask bit set ONLY when the stat is active (the v95 trailer read is mask-gated per member, IDA @`0x73DBA0`, design §2.4). `decodeBaseTemporaryStats` gains the decoded mask as a parameter.
- Registry facts (already true, pinned by fixtures here): v95 shifts EnergyCharge=122, DashSpeed=123, DashJump=124, RideVehicle=125, PartyBooster=126, HomingBeacon=127 (`buildCharacterTemporaryStatRegistry`, two-state block at the end of the builder).

- [ ] **Step 1: Write the failing tests** — append to `character_temporary_stat_test.go`:

```go
// TestCTSHomingBeaconV95MaskAndBlock pins the v95 beacon give: bit 127
// (0x80000000 in wire dword[0]) joins the 4 always-set two-state bits
// (0x3C000000), and the trailer is the status-quo 58 bytes plus one populated
// 17-byte GuidedBullet block. IDA: v95 group @SecondaryStat::SecondaryStat
// 0x72F190, GuidedBullet DecodeForClient 0x727180, mask-gated tail read
// 0x73DBA0 (design.md §2.4).
func TestCTSHomingBeaconV95MaskAndBlock(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	tn, _ := tenant.Create([16]byte{}, "GMS", 95, 1)
	input := NewCharacterTemporaryStat()
	input.AddStat(nil)(tn)(string(character.TemporaryStatTypeHomingBeacon), 5220011, 1000001, 10, time.Time{})

	got := input.Encode(nil, ctx)(nil)

	// dword[0] = 0x3C000000 | 0x80000000 = 0xBC000000 -> LE bytes 00 00 00 BC.
	if !bytes.Equal(got[0:4], []byte{0x00, 0x00, 0x00, 0xBC}) {
		t.Fatalf("v95 mask dword[0]: got % x want 00 00 00 bc", got[0:4])
	}
	if !bytes.Equal(got[4:16], make([]byte, 12)) {
		t.Fatalf("v95 mask dwords[1..3] should be empty; got % x", got[4:16])
	}
	// 16 mask + 2 leading + 58 status-quo blocks + 17 GuidedBullet.
	if len(got) != 16+2+58+17 {
		t.Fatalf("v95 beacon packet length: got %d want %d", len(got), 16+2+58+17)
	}
	// Populated block: nOption=1000001, rOption=5220011 (0x004FA6AB).
	head := []byte{0x41, 0x42, 0x0F, 0x00, 0xAB, 0xA6, 0x4F, 0x00}
	idx := bytes.Index(got, head)
	if idx < 0 {
		t.Fatalf("v95 populated GuidedBullet head missing; got % x", got)
	}
	if !bytes.Equal(got[idx+13:idx+17], []byte{0x41, 0x42, 0x0F, 0x00}) {
		t.Fatalf("v95 dwMobId: got % x want 41 42 0f 00", got[idx+13:idx+17])
	}
}

// Non-beacon v95 traffic must stay byte-identical to the current truncated
// encode (regression safety for every existing v95 buff packet).
// TestCTSMonsterRidingV95MaskAndLayout (above) already pins the mount case;
// this pins the empty case.
func TestCTSEmptyV95StaysStatusQuo(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	input := NewCharacterTemporaryStat()
	got := input.Encode(nil, ctx)(nil)
	if len(got) != 16+2+58 {
		t.Fatalf("empty v95 CTS length: got %d want %d", len(got), 16+2+58)
	}
	if !bytes.Equal(got[0:4], []byte{0x00, 0x00, 0x00, 0x3C}) {
		t.Fatalf("empty v95 mask dword[0]: got % x want 00 00 00 3c", got[0:4])
	}
}

// TestCTSPartyBoosterV95Block pins the conditional PartyBooster member:
// bit 126 (0x40000000) and a 20-byte block (base 13 + tCurrentTime 5 +
// usExpireTerm 2 — IDA DecodeForClient 0x72C600). PartyBooster has no
// producer in atlas yet; this exercises the verified wire slot only.
func TestCTSPartyBoosterV95Block(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	tn, _ := tenant.Create([16]byte{}, "GMS", 95, 1)
	input := NewCharacterTemporaryStat()
	input.AddStat(nil)(tn)(string(character.TemporaryStatTypePartyBooster), 1005017, 20, 20, time.Now().Add(time.Minute))

	got := input.Encode(nil, ctx)(nil)

	if !bytes.Equal(got[0:4], []byte{0x00, 0x00, 0x00, 0x7C}) {
		t.Fatalf("v95 mask dword[0] with PartyBooster: got % x want 00 00 00 7c", got[0:4])
	}
	if len(got) != 16+2+58+20 {
		t.Fatalf("v95 PartyBooster packet length: got %d want %d", len(got), 16+2+58+20)
	}
}

// Decode must mirror the conditional read: beacon- and PartyBooster-bearing
// v95 payloads round-trip without desyncing the reader.
func TestCTSHomingBeaconV95RoundTrip(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	tn, _ := tenant.Create([16]byte{}, "GMS", 95, 1)
	input := NewCharacterTemporaryStat()
	input.AddStat(nil)(tn)(string(character.TemporaryStatTypeHomingBeacon), 5220011, 1000001, 10, time.Time{})
	output := NewCharacterTemporaryStat()
	pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
}

func TestCTSPartyBoosterV95RoundTrip(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	tn, _ := tenant.Create([16]byte{}, "GMS", 95, 1)
	input := NewCharacterTemporaryStat()
	input.AddStat(nil)(tn)(string(character.TemporaryStatTypePartyBooster), 1005017, 20, 20, time.Now().Add(time.Minute))
	output := NewCharacterTemporaryStat()
	pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./model/ -run 'V95' -v` from `libs/atlas-packet`.
Expected: new tests FAIL (beacon/PartyBooster emit nothing on v95 today); `TestCTSMonsterRidingV95MaskAndLayout` and `TestCTSEmptyV95StaysStatusQuo` PASS.

- [ ] **Step 3: Implement**

1. Add the kind and the conditional flag:

```go
const (
	twoStateDynamic       twoStateKind = iota // dynamic base block (15B): EnergyCharge, DashSpeed, DashJump, Undead
	twoStateMonsterRiding                     // non-dynamic base (13B): nOption=vehicle id, rOption=skill id
	twoStateSpeedInfusion                     // SpeedInfusion special block (20B)
	twoStateGuidedBullet                      // GuidedBullet special block (17B)
	twoStatePartyBooster                      // v95 PartyBooster block (20B: base 13 + tCurrentTime 5 + usExpireTerm 2)
)

type twoStateStat struct {
	name character.TemporaryStatType
	kind twoStateKind
	// conditional members (v95 PartyBooster/GuidedBullet) set their mask bit and
	// write their block ONLY when the stat is active. The v95 client's two-state
	// trailer read is mask-gated per member (IDA @0x73DBA0), so absent members are
	// simply skipped; pre-95 clients read all 7 blocks unconditionally, so pre-95
	// members are never conditional. design.md §2.4/§4.4.
	conditional bool
}
```

2. Update every `twoStateStat{...}` literal in `twoStateBaseStats` to three fields (existing members get `false`), replace the v95 early-return, and update the function's doc comment (delete the "not yet IDA-verified (Task 41b)" paragraph; state the verified 6-member group with block sizes 15/15/15/13/20/17 and bits 122–127, citing design.md §2.4):

```go
func twoStateBaseStats(t tenant.Model) []twoStateStat {
	stats := []twoStateStat{
		{character.TemporaryStatTypeEnergyCharge, twoStateDynamic, false},
		{character.TemporaryStatTypeDashSpeed, twoStateDynamic, false},
		{character.TemporaryStatTypeDashJump, twoStateDynamic, false},
		{character.TemporaryStatTypeMonsterRiding, twoStateMonsterRiding, false},
	}
	if t.Region() == "GMS" && t.MajorVersion() >= 95 {
		// v95 verified 6-member group (design.md §2.4): the 4 unconditional
		// members above stay always-written (status quo, fixture-locked);
		// PartyBooster(126) and GuidedBullet(127) are conditional. Undead has
		// no v95 wire slot (bit 128 overflows the mask).
		return append(stats,
			twoStateStat{character.TemporaryStatTypePartyBooster, twoStatePartyBooster, true},
			twoStateStat{character.TemporaryStatTypeHomingBeacon, twoStateGuidedBullet, true},
		)
	}
	return append(stats,
		twoStateStat{character.TemporaryStatTypeSpeedInfusion, twoStateSpeedInfusion, false},
		twoStateStat{character.TemporaryStatTypeHomingBeacon, twoStateGuidedBullet, false},
		twoStateStat{character.TemporaryStatTypeUndead, twoStateDynamic, false},
	)
}
```

3. `EncodeMask`: in the unconditional two-state loop, skip conditional members — their bit arrives via the active-stats loop that already ORs every entry of `m.stats`:

```go
		for _, bs := range twoStateBaseStats(t) {
			if bs.conditional {
				// Conditional members' bits are set by the active-stats loop
				// below only when the stat is present (v95 mask-gated read).
				continue
			}
			if st, ok := reg.byName[bs.name]; ok {
				mask = mask.Or(st.mask)
			}
		}
```

4. `getBaseTemporaryStats`: skip inactive conditional members and add the PartyBooster case (reusing `SpeedInfusionTemporaryStat` — identical 20-byte wire shape; `usExpireItem` carries the client's `usExpireTerm`):

```go
	for _, bs := range twoStateBaseStats(t) {
		if bs.conditional {
			if _, ok := m.stats[bs.name]; !ok {
				continue
			}
		}
		switch bs.kind {
		// (all existing cases — twoStateMonsterRiding, twoStateSpeedInfusion,
		// twoStateGuidedBullet, default/twoStateDynamic — stay exactly as they
		// are after Task 5; only this case is NEW:)
		case twoStatePartyBooster:
			// v95 PartyBooster (bit 126): 20-byte block, same wire shape as the
			// pre-95 SpeedInfusion block (base + tCurrentTime + usExpireTerm),
			// IDA DecodeForClient @0x72C600. Reached only when active
			// (conditional member).
			s := m.stats[bs.name]
			list = append(list, SpeedInfusionTemporaryStat{
				CharacterTemporaryStatBase: CharacterTemporaryStatBase{
					bDynamicTermSet: false,
					nOption:         s.Value(),
					rOption:         s.SourceId(),
					tLastUpdated:    time.Now().Unix(),
				},
			})
		}
	}
```

5. `decodeBaseTemporaryStats`: change the signature to accept the decoded mask and skip conditional members whose bit is unset; add the PartyBooster case:

```go
func (m *CharacterTemporaryStat) decodeBaseTemporaryStats(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}, mask tool.Uint128) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}, mask tool.Uint128) {
		reg := buildCharacterTemporaryStatRegistry(t)
		for _, bs := range twoStateBaseStats(t) {
			if bs.conditional {
				st, ok := reg.byName[bs.name]
				if !ok || mask.And(st.mask).IsZero() {
					continue
				}
			}
			switch bs.kind {
			case twoStateSpeedInfusion, twoStatePartyBooster:
				si := SpeedInfusionTemporaryStat{CharacterTemporaryStatBase: CharacterTemporaryStatBase{bDynamicTermSet: false}}
				si.Decode(l, ctx)(r, options)
			case twoStateGuidedBullet:
				gb := GuidedBulletTemporaryStat{CharacterTemporaryStatBase: CharacterTemporaryStatBase{bDynamicTermSet: false}}
				gb.Decode(l, ctx)(r, options)
			case twoStateMonsterRiding:
				base := CharacterTemporaryStatBase{bDynamicTermSet: false}
				base.Decode(l, ctx)(r, options)
			default: // twoStateDynamic
				base := CharacterTemporaryStatBase{bDynamicTermSet: true}
				base.Decode(l, ctx)(r, options)
			}
		}
	}
}
```

Update the two callers (`Decode`, `DecodeForeign`) to pass their local `mask`: `m.decodeBaseTemporaryStats(l, ctx)(r, options, mask)`.

6. `EncodeForeign` needs no code change for FR-4.5: it shares `EncodeMask`/`getBaseTemporaryStats`, and the channel never adds `HOMING_BEACON` to a foreign CTS (per-stat foreign path skips it via `baseStatNames`, and Task 12 suppresses beacon-only foreign announcements), so bit 127 is never set on a foreign v95 encode. Add this foreign regression test:

```go
// Foreign v95 encode must never carry the GuidedBullet block even if a beacon
// stat is (incorrectly) present upstream: HOMING_BEACON is caster-only and the
// remote-reader path is unverified (FR-4.5). The lib guarantees this by CTS
// construction (channel never AddStats it on foreign bodies); this test pins
// that an EMPTY foreign v95 CTS stays at the status-quo length.
func TestCTSForeignEmptyV95StaysStatusQuo(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	input := NewCharacterTemporaryStat()
	got := input.EncodeForeign(nil, ctx)(nil)
	if len(got) != 16+2+58 {
		t.Fatalf("empty foreign v95 CTS length: got %d want %d", len(got), 16+2+58)
	}
}
```

- [ ] **Step 4: Run the full lib tests**

Run: `go test -race ./...` from `libs/atlas-packet`.
Expected: PASS — including all pre-existing v95 fixtures (mount, truncation length 16+2+58).

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-packet/model/character_temporary_stat.go libs/atlas-packet/model/character_temporary_stat_test.go
git commit -m "feat(packet): verified v95 two-state group with conditional PartyBooster/GuidedBullet (task-167 FR-4.3, closes Task 41b)"
```

---

### Task 7: IDA — movement-affecting filter for v84/v87/JMS + evidence records

**Files:**
- Create: `docs/tasks/task-167-homing-beacon-bullseye/evidence/movement-filter.md`
- Create: `docs/tasks/task-167-homing-beacon-bullseye/evidence/v95-two-state-group.md`

**Interfaces:**
- Consumes: IDA-MCP (`mcp__ida-pro__*`). Instances: v83 = `MapleStory_dump.exe` port 13342, v95 = `GMS_v95.0_U_DEVM.exe` port 13341, JMS = port 13340 (use the `*_U_DEVM` build, NOT the SMC retail dump); v84/v87 — run `mcp__ida-pro__list_instances` and select by binary name; confirm the loaded binary matches the target version BEFORE reading (CLAUDE.md IDA rule). Use `func_query` with `name_regex` for lookups.
- Produces: the per-version movement-affecting stat lists Task 8 encodes into `movementAffectingStatNames`. Known from design §2.3/§2.4:
  - v83 (filter fn `sub_77DC78`): Speed, Jump, Stun, Weakness, Slow, Morph, Ghost, BasicStatUp, Attract, RideVehicle, and the two Dash bits.
  - v95 (`SecondaryStat::IsMovementAffectingStat` @`0x7208C0`): the v83 list + Flying, Frozen, YellowAura.
  - v84/v87/JMS: EXPECTED identical to v83's list, but must be verified, not assumed.

- [ ] **Step 1: Extract the v84 filter.** Select the v84 instance. Locate `CWvsContext::OnTemporaryStatReset` (known v84 reset address from the existing packet-audit marker: `0xa6bb24`, see `libs/atlas-packet/character/clientbound/buff_cancel_test.go`). Decompile it; the trailing-byte read is gated by a helper called with the decoded 16-byte mask (v83's equivalent is `sub_77DC78`). Decompile that helper and record every CTS mask constant it tests, resolving each constant's bit position to the client stat name via the CTS dynamic-initializer globals (same method the design used, §2.3).
- [ ] **Step 2: Extract the v87 filter.** Same procedure; v87 reset marker address `0xab7dc1`.
- [ ] **Step 3: Extract the JMS v185 filter.** Same procedure; JMS reset marker address `0xb07628`.
- [ ] **Step 4: Cross-check the atlas name mapping.** For each filter member, confirm the atlas `TemporaryStatType` whose registry shift equals the client constant's bit position on that version (`buildCharacterTemporaryStatRegistry`): expected Weakness→`Weaken`, Ghost→`GhostMorph`, BasicStatUp→`MapleWarrior`, Attract→`Seduce`, RideVehicle→`MonsterRiding`. If any shift does not line up, STOP and report — do not guess a mapping.
- [ ] **Step 5: Write `evidence/movement-filter.md`.** One section per version (v83, v84, v87, JMS, v95): filter function address, decompiled constant list, resolved stat names, and the atlas mapping table. v83/v95 sections come from design §2.3/§2.4 (already verified); v84/v87/JMS from Steps 1–3. State plainly for each version whether it matched v83's list; if one differs, the verified truth wins and Task 8's table must gate it.
- [ ] **Step 6: Write `evidence/v95-two-state-group.md`.** Record the §2.4 v95 facts (group membership table with addresses `0x72F190`, per-slot DecodeForClient addresses, block sizes 15/15/15/13/20/17, bits 122–127, mask-gated tail read `0x73DBA0-0x73DBF2`, set path `0xA02FC0`, reset path `0x9F2AB0`/`0x6572E0`) so the "Task 41b" closure has a standalone evidence record.
- [ ] **Step 7: Commit**

```bash
git add docs/tasks/task-167-homing-beacon-bullseye/evidence/
git commit -m "docs(task-167): IDA evidence — movement filters (all versions) + v95 two-state group"
```

---

### Task 8: atlas-packet — accurate cancel masks + conditional movement byte (F1)

**Files:**
- Modify: `libs/atlas-packet/model/character_temporary_stat.go`
- Modify: `libs/atlas-packet/character/clientbound/buff_cancel.go`
- Test: `libs/atlas-packet/model/character_temporary_stat_test.go`, `libs/atlas-packet/character/clientbound/buff_cancel_test.go`

**Interfaces:**
- Consumes: the per-version filter lists from Task 7's evidence file (v83/v95 already known from design §2.3/§2.4; v84/v87/JMS as extracted — plan below assumes they matched v83; if Task 7 found a difference, gate it the same way v95 is gated).
- Produces:
  - `func (m *CharacterTemporaryStat) CancelMask(t tenant.Model) tool.Uint128` — OR of ONLY the stats present in `m.stats` (no unconditional two-state bits).
  - `func (m *CharacterTemporaryStat) EncodeCancelMask(l logrus.FieldLogger, t tenant.Model, options map[string]interface{}) func(w *response.Writer)` — writes `CancelMask` in the same 4-dword layout as `EncodeMask`.
  - `func MovementAffectingMask(t tenant.Model) tool.Uint128` — exported; the version-gated movement filter as a mask.
  - `BuffCancel`/`BuffCancelForeign` use them; the trailing byte is written/read iff `cancelMask AND MovementAffectingMask != 0`.
- The give path (`EncodeMask`, BuffGive trailer) is untouched: give masks always contain the RideVehicle/Dash group bits (pre-95 unconditional; v95 4 always-set members), which are movement-affecting, so the client always reads the give trailer byte — status quo holds (design §5.5.3).

- [ ] **Step 1: Write the failing model tests** — append to `character_temporary_stat_test.go`:

```go
// F1 regression: a cancel mask must contain ONLY the canceled stats — never
// the unconditional two-state group bits. Under the old EncodeMask-reused
// cancel, ANY cancel cleared every two-state stat client-side (v83 reset
// @0xA2071F, v95 @0x9F2AB0 clear every masked stat).
func TestCancelMaskContainsOnlyActiveStats(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			tn, _ := tenant.Create([16]byte{}, v.Region, v.MajorVersion, v.MinorVersion)
			cts := NewCharacterTemporaryStat()
			cts.AddStat(nil)(tn)(string(character.TemporaryStatTypeInvincible), 2301003, 30, 20, time.Now().Add(time.Minute))

			mask := cts.CancelMask(tn)
			reg := buildCharacterTemporaryStatRegistry(tn)
			inv := reg.byName[character.TemporaryStatTypeInvincible]
			if mask.And(inv.mask).IsZero() {
				t.Fatal("cancel mask missing the canceled stat's bit")
			}
			riding := reg.byName[character.TemporaryStatTypeMonsterRiding]
			if !mask.And(riding.mask).IsZero() {
				t.Fatal("cancel mask must not contain inactive two-state bits (F1)")
			}
		})
	}
}

func TestCancelMaskEmptyForEmptyCTS(t *testing.T) {
	tn, _ := tenant.Create([16]byte{}, "GMS", 83, 1)
	cts := NewCharacterTemporaryStat()
	if !cts.CancelMask(tn).IsZero() {
		t.Fatal("empty CTS must produce an empty cancel mask")
	}
}

// Movement filter membership per version (IDA: v83 sub_77DC78, v95
// SecondaryStat::IsMovementAffectingStat @0x7208C0, v84/v87/JMS per
// docs/tasks/task-167-homing-beacon-bullseye/evidence/movement-filter.md).
func TestMovementAffectingMaskMembership(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			tn, _ := tenant.Create([16]byte{}, v.Region, v.MajorVersion, v.MinorVersion)
			reg := buildCharacterTemporaryStatRegistry(tn)
			mv := MovementAffectingMask(tn)

			in := []character.TemporaryStatType{
				character.TemporaryStatTypeSpeed,
				character.TemporaryStatTypeJump,
				character.TemporaryStatTypeStun,
				character.TemporaryStatTypeWeaken,
				character.TemporaryStatTypeSlow,
				character.TemporaryStatTypeMorph,
				character.TemporaryStatTypeGhostMorph,
				character.TemporaryStatTypeMapleWarrior,
				character.TemporaryStatTypeSeduce,
				character.TemporaryStatTypeMonsterRiding,
				character.TemporaryStatTypeDashSpeed,
				character.TemporaryStatTypeDashJump,
			}
			out := []character.TemporaryStatType{
				character.TemporaryStatTypeHomingBeacon,
				character.TemporaryStatTypeInvincible,
			}
			if tn.Region() == "GMS" && tn.MajorVersion() >= 95 {
				in = append(in,
					character.TemporaryStatTypeFlying,
					character.TemporaryStatTypeFrozen,
					character.TemporaryStatTypeYellowAura,
				)
			}
			for _, n := range in {
				st, ok := reg.byName[n]
				if !ok {
					continue // stat not enumerated on this version
				}
				if mv.And(st.mask).IsZero() {
					t.Errorf("%s should be movement-affecting on %s", n, v.Name)
				}
			}
			for _, n := range out {
				st, ok := reg.byName[n]
				if !ok {
					continue
				}
				if !mv.And(st.mask).IsZero() {
					t.Errorf("%s should NOT be movement-affecting on %s", n, v.Name)
				}
			}
		})
	}
}
```

- [ ] **Step 2: Write the failing writer tests** — append to `libs/atlas-packet/character/clientbound/buff_cancel_test.go`:

```go
// Beacon-only cancel: mask carries exactly the GuidedBullet bit (v83 shift 87
// -> wire dword[1] 0x00800000) and NO movement byte (GuidedBullet is not in
// the movement filter; the client reads the trailing byte only when
// sub_77DC78(mask) is true — design.md §2.3).
func TestBuffCancelBeaconOnlyV83(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	tn, _ := tenant.Create([16]byte{}, "GMS", 83, 1)
	cts := model.NewCharacterTemporaryStat()
	cts.AddStat(nil)(tn)(string(character.TemporaryStatTypeHomingBeacon), 5211006, 1000001, 1, time.Time{})

	got := NewBuffCancel(*cts).Encode(nil, ctx)(nil)

	want := []byte{
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x80, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("beacon-only cancel: got % x want % x (16 bytes, no movement byte)", got, want)
	}
}

// A movement-affecting cancel (Speed) carries the trailing byte.
func TestBuffCancelSpeedCarriesMovementByteV83(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	tn, _ := tenant.Create([16]byte{}, "GMS", 83, 1)
	cts := model.NewCharacterTemporaryStat()
	cts.AddStat(nil)(tn)(string(character.TemporaryStatTypeSpeed), 2001002, 20, 10, time.Now().Add(time.Minute))

	got := NewBuffCancel(*cts).Encode(nil, ctx)(nil)

	if len(got) != 17 {
		t.Fatalf("speed cancel length: got %d want 17 (mask + movement byte)", len(got))
	}
	// Speed is registry shift 7 -> mask.L low dword -> wire dword[3]
	// (bytes 12-15) = 80 00 00 00; dwords [0..2] empty.
	if !bytes.Equal(got[0:12], make([]byte, 12)) {
		t.Fatalf("speed cancel mask dwords[0..2] should be empty: got % x", got[0:12])
	}
	if !bytes.Equal(got[12:16], []byte{0x80, 0x00, 0x00, 0x00}) {
		t.Fatalf("speed cancel mask dword[3]: got % x", got[12:16])
	}
}

// A mount cancel carries the riding bit and the movement byte, and nothing else.
func TestBuffCancelMountV83(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	tn, _ := tenant.Create([16]byte{}, "GMS", 83, 1)
	cts := model.NewCharacterTemporaryStat()
	cts.AddStat(nil)(tn)(string(character.TemporaryStatTypeMonsterRiding), 1004, 1902000, 1, time.Now().Add(time.Hour))

	got := NewBuffCancel(*cts).Encode(nil, ctx)(nil)

	if len(got) != 17 {
		t.Fatalf("mount cancel length: got %d want 17", len(got))
	}
	// RideVehicle shift 85 -> dword[1] = 0x00200000 -> LE 00 00 20 00.
	if !bytes.Equal(got[4:8], []byte{0x00, 0x00, 0x20, 0x00}) {
		t.Fatalf("mount cancel mask dword[1]: got % x want 00 00 20 00", got[4:8])
	}
}

// v95 beacon-only cancel: exactly bit 127 (dword[0] 0x80000000), no movement byte.
func TestBuffCancelBeaconOnlyV95(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	tn, _ := tenant.Create([16]byte{}, "GMS", 95, 1)
	cts := model.NewCharacterTemporaryStat()
	cts.AddStat(nil)(tn)(string(character.TemporaryStatTypeHomingBeacon), 5220011, 1000001, 10, time.Time{})

	got := NewBuffCancel(*cts).Encode(nil, ctx)(nil)

	want := append([]byte{0x00, 0x00, 0x00, 0x80}, make([]byte, 12)...)
	if !bytes.Equal(got, want) {
		t.Fatalf("v95 beacon-only cancel: got % x want % x", got, want)
	}
}
```

Add imports `"bytes"`, `"time"`, and `"github.com/Chronicle20/atlas/libs/atlas-constants/character"`, `tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"` to the test file as needed.

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./model/ ./character/clientbound/ -run 'CancelMask|MovementAffecting|BuffCancel' -v` from `libs/atlas-packet`.
Expected: FAIL — `CancelMask`/`MovementAffectingMask` undefined; BuffCancel fixtures see 17 bytes with full two-state mask.

- [ ] **Step 4: Implement in `model/character_temporary_stat.go`**

```go
// CancelMask returns the mask of ONLY the stats present on this CTS. Cancel
// packets must use this instead of EncodeMask's give-shape mask: the client's
// TemporaryStatReset clears EVERY masked stat (v83 @0xA2071F, v95 @0x9F2AB0),
// so a cancel carrying the unconditional two-state group bits destroys any
// active mount/dash/energy-charge/beacon (design.md §3 F1).
func (m *CharacterTemporaryStat) CancelMask(t tenant.Model) tool.Uint128 {
	mask := tool.Uint128{}
	for _, v := range m.stats {
		mask = mask.Or(v.statType.mask)
	}
	return mask
}

// EncodeCancelMask writes CancelMask in the same 4-dword wire layout as
// EncodeMask.
func (m *CharacterTemporaryStat) EncodeCancelMask(l logrus.FieldLogger, t tenant.Model, options map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		writeMask(w, m.CancelMask(t))
	}
}

func writeMask(w *response.Writer, mask tool.Uint128) {
	w.WriteInt(uint32(mask.H >> 32))
	w.WriteInt(uint32(mask.H & 0xFFFFFFFF))
	w.WriteInt(uint32(mask.L >> 32))
	w.WriteInt(uint32(mask.L & 0xFFFFFFFF))
}
```

Refactor the tail of `EncodeMask` to call `writeMask(w, mask)` (bytes unchanged).

```go
// movementAffectingStatNames is the version-gated mirror of the client's
// movement filter: the reset/give trailing byte is read ONLY when the packet
// mask intersects this set (v83 sub_77DC78; v95 SecondaryStat::
// IsMovementAffectingStat @0x7208C0; v84/v87/JMS verified identical to v83 —
// docs/tasks/task-167-homing-beacon-bullseye/evidence/movement-filter.md).
// Client names map: Weakness->Weaken, Ghost->GhostMorph, BasicStatUp->
// MapleWarrior, Attract->Seduce, RideVehicle->MonsterRiding (shift-verified).
func movementAffectingStatNames(t tenant.Model) []character.TemporaryStatType {
	names := []character.TemporaryStatType{
		character.TemporaryStatTypeSpeed,
		character.TemporaryStatTypeJump,
		character.TemporaryStatTypeStun,
		character.TemporaryStatTypeWeaken,
		character.TemporaryStatTypeSlow,
		character.TemporaryStatTypeMorph,
		character.TemporaryStatTypeGhostMorph,
		character.TemporaryStatTypeMapleWarrior,
		character.TemporaryStatTypeSeduce,
		character.TemporaryStatTypeMonsterRiding,
		character.TemporaryStatTypeDashSpeed,
		character.TemporaryStatTypeDashJump,
	}
	if t.Region() == "GMS" && t.MajorVersion() >= 95 {
		names = append(names,
			character.TemporaryStatTypeFlying,
			character.TemporaryStatTypeFrozen,
			character.TemporaryStatTypeYellowAura,
		)
	}
	return names
}

// MovementAffectingMask returns the movement filter as a mask for this
// tenant's registry layout. Writers AND their packet mask against it to
// decide whether the client will read the trailing movement byte.
func MovementAffectingMask(t tenant.Model) tool.Uint128 {
	reg := buildCharacterTemporaryStatRegistry(t)
	mask := tool.Uint128{}
	for _, n := range movementAffectingStatNames(t) {
		if st, ok := reg.byName[n]; ok {
			mask = mask.Or(st.mask)
		}
	}
	return mask
}
```

If Task 7 found any version whose filter differs from v83's, add the corresponding gate here (same style as the v95 branch) — the evidence file is authoritative.

- [ ] **Step 5: Implement in `character/clientbound/buff_cancel.go`**

`BuffCancel.Encode`:

```go
func (m BuffCancel) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		m.cts.EncodeCancelMask(l, t, options)(w)
		// The trailing byte is the movement flag the client reads ONLY when
		// the mask intersects the movement filter (it was mislabeled
		// tSwallowBuffTime; the old always-full mask made it unconditionally
		// required). design.md §5.5.3.
		if !m.cts.CancelMask(t).And(model.MovementAffectingMask(t)).IsZero() {
			w.WriteByte(0)
		}
		return w.Bytes()
	}
}
```

`BuffCancel.Decode`:

```go
func (m *BuffCancel) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.cts = *model.NewCharacterTemporaryStat()
		mask := m.cts.DecodeMask(r)
		if !mask.And(model.MovementAffectingMask(t)).IsZero() {
			_ = r.ReadByte() // movement flag
		}
	}
}
```

Apply the identical mask/byte change to `BuffCancelForeign.Encode`/`Decode` (keeping the leading `characterId` int).

- [ ] **Step 6: Run the full lib tests**

Run: `go test -race ./...` from `libs/atlas-packet`.
Expected: PASS. The pre-existing `TestBuffCancelRoundTrip`/`TestBuffCancelForeignRoundTrip` (empty CTS) stay green: empty mask → no byte on encode, none consumed on decode. The `packet-audit:verify` marker comments on those tests remain valid — the pinned client reset handlers are exactly the mask-gated readers this change conforms to; do NOT touch the markers.

- [ ] **Step 7: Commit**

```bash
git add libs/atlas-packet/model/character_temporary_stat.go libs/atlas-packet/model/character_temporary_stat_test.go libs/atlas-packet/character/clientbound/buff_cancel.go libs/atlas-packet/character/clientbound/buff_cancel_test.go
git commit -m "fix(packet): accurate cancel masks + conditional movement byte (task-167 F1)"
```

---

### Task 9: atlas-channel — noExpiry contract mirror + buff mirror model

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/message/buff/kafka.go`
- Modify: `services/atlas-channel/atlas.com/channel/character/buff/model.go`
- Modify: `services/atlas-channel/atlas.com/channel/character/buff/rest.go`
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/buff/consumer.go` (NewBuff call sites only)
- Test: `services/atlas-channel/atlas.com/channel/character/buff/model_test.go` (create)

**Interfaces:**
- Consumes: the wire fields Task 2 added (`noExpiry` on APPLY command and APPLIED/EXPIRED events — same JSON tags).
- Produces:
  - channel `buff.NewBuff(sourceId int32, level byte, duration int32, changes []stat.Model, createdAt time.Time, expiresAt time.Time, noExpiry bool) Model` — trailing param appended; `(Model).NoExpiry() bool`; flag-aware `Expired()`.
  - channel kafka message types: `ApplyCommandBody.NoExpiry`, `AppliedStatusEventBody.NoExpiry`, `ExpiredStatusEventBody.NoExpiry` (all `json:"noExpiry,omitempty"`), plus `CommandTypeCancelByTypes = "CANCEL_BY_TYPES"` and `CancelByTypesCommandBody{Types []string}` (consumed by Task 10).
  - `RestModel.NoExpiry bool` (`json:"noExpiry"`) mapped in `Extract`.

- [ ] **Step 1: Write the failing test** — create `character/buff/model_test.go`:

```go
package buff

import (
	"testing"
	"time"

	"atlas-channel/character/buff/stat"
)

func TestNoExpiryMirrorNeverExpires(t *testing.T) {
	b := NewBuff(5211006, 1, 0, []stat.Model{stat.NewStat("HOMING_BEACON", 1000001)}, time.Now(), time.Time{}, true)
	if b.Expired() {
		t.Fatal("no-expiry mirror buff must not report expired")
	}
	if !b.NoExpiry() {
		t.Fatal("NoExpiry() must be true")
	}
}

func TestFiniteMirrorStillExpires(t *testing.T) {
	b := NewBuff(2001001, 5, 1000, []stat.Model{stat.NewStat("SPEED", 20)}, time.Now().Add(-2*time.Second), time.Now().Add(-time.Second), false)
	if !b.Expired() {
		t.Fatal("past-expiry finite mirror buff must report expired")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./character/buff/ -v`
Expected: FAIL — wrong number of arguments to `NewBuff` / `NoExpiry` undefined.

- [ ] **Step 3: Implement**

`kafka/message/buff/kafka.go` — add to the const block:

```go
	CommandTypeCancelByTypes = "CANCEL_BY_TYPES"
```

add the body type:

```go
type CancelByTypesCommandBody struct {
	Types []string `json:"types"`
}
```

and add `NoExpiry bool \`json:"noExpiry,omitempty"\`` to `ApplyCommandBody`, `AppliedStatusEventBody`, and `ExpiredStatusEventBody` (matching atlas-buffs' authoritative schema from Task 2).

`character/buff/model.go` — add `noExpiry bool` field, accessor, flag-aware `Expired()` (identical logic to Task 1's), and the trailing constructor param:

```go
func NewBuff(sourceId int32, level byte, duration int32, changes []stat.Model, createdAt time.Time, expiresAt time.Time, noExpiry bool) Model {
	return Model{
		sourceId:  sourceId,
		level:     level,
		duration:  duration,
		changes:   changes,
		createdAt: createdAt,
		expiresAt: expiresAt,
		noExpiry:  noExpiry,
	}
}
```

`character/buff/rest.go` — add `NoExpiry bool \`json:"noExpiry"\`` to `RestModel`; in `Extract`, populate the model's `noExpiry` field (struct literal gains `noExpiry: rm.NoExpiry`).

`kafka/consumer/buff/consumer.go` — the two `buff.NewBuff(...)` call sites (`handleStatusEventApplied` line ~73, `handleStatusEventExpired` line ~109) each gain the trailing argument `e.Body.NoExpiry`.

- [ ] **Step 4: Run tests + build**

Run: `go test -race ./... && go build ./...` from `services/atlas-channel/atlas.com/channel`.
Expected: PASS / clean (the compiler surfaces any missed `NewBuff` caller — grep confirmed only the two consumer sites plus `rest.go`'s struct literal).

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel
git commit -m "feat(channel): mirror noExpiry through buff contract and mirror model (task-167 FR-2.5)"
```

---

### Task 10: atlas-channel — CancelByTypes + ApplyNoExpiry buff commands

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/character/buff/producer.go`
- Modify: `services/atlas-channel/atlas.com/channel/character/buff/processor.go`

**Interfaces:**
- Consumes: `CommandTypeCancelByTypes` / `CancelByTypesCommandBody` / `ApplyCommandBody.NoExpiry` (Task 9); `statup.Model` (`Mask() string`, `Amount() int32`).
- Produces (Task 13 calls these):
  - `Processor.ApplyNoExpiry(f field.Model, fromId uint32, sourceId int32, level byte, statups []statup.Model) model.Operator[uint32]`
  - `Processor.CancelByTypes(f field.Model, characterId uint32, types []string) error`
- Deliberate deviation from atlas-buffs' "one code path" rule: the channel-side `Apply` has many existing callers (skill common path, mounts, mystic door), so a separate `ApplyNoExpiry` method avoids a signature ripple; atlas-buffs' internal single path (Task 2) is where the semantics live. There is no mock of this channel processor (verified: no mock dir under `character/buff/`).

- [ ] **Step 1: Implement the producers** — append to `producer.go` (shape copied from atlas-consumables' `cancelByTypesCommandProvider`, `services/atlas-consumables/atlas.com/consumables/character/buff/producer.go:57-72`):

```go
// ApplyNoExpiryCommandProvider emits an APPLY carrying the explicit noExpiry
// flag (Duration 0 — atlas-buffs rejects the flag with a nonzero duration).
func ApplyNoExpiryCommandProvider(f field.Model, characterId uint32, fromId uint32, sourceId int32, level byte, statups []statup.Model) model.Provider[[]kafka.Message] {
	changes := make([]buff.StatChange, 0)
	for _, su := range statups {
		changes = append(changes, buff.StatChange{
			Type:   su.Mask(),
			Amount: su.Amount(),
		})
	}

	key := producer.CreateKey(int(characterId))
	value := &buff.Command[buff.ApplyCommandBody]{
		WorldId:     f.WorldId(),
		ChannelId:   f.ChannelId(),
		MapId:       f.MapId(),
		Instance:    f.Instance(),
		CharacterId: characterId,
		Type:        buff.CommandTypeApply,
		Body: buff.ApplyCommandBody{
			FromId:   fromId,
			SourceId: sourceId,
			Level:    level,
			Duration: 0,
			Changes:  changes,
			NoExpiry: true,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func CancelByTypesCommandProvider(f field.Model, characterId uint32, types []string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &buff.Command[buff.CancelByTypesCommandBody]{
		WorldId:     f.WorldId(),
		ChannelId:   f.ChannelId(),
		MapId:       f.MapId(),
		Instance:    f.Instance(),
		CharacterId: characterId,
		Type:        buff.CommandTypeCancelByTypes,
		Body: buff.CancelByTypesCommandBody{
			Types: append([]string(nil), types...),
		},
	}
	return producer.SingleMessageProvider(key, value)
}
```

- [ ] **Step 2: Extend the processor** — in `processor.go`, add to the `Processor` interface:

```go
	ApplyNoExpiry(f field.Model, fromId uint32, sourceId int32, level byte, statups []statup.Model) model.Operator[uint32]
	CancelByTypes(f field.Model, characterId uint32, types []string) error
```

and implement on `ProcessorImpl`:

```go
func (p *ProcessorImpl) ApplyNoExpiry(f field.Model, fromId uint32, sourceId int32, level byte, statups []statup.Model) model.Operator[uint32] {
	return func(characterId uint32) error {
		p.l.Debugf("Character [%d] applying no-expiry effect from source [%d].", characterId, sourceId)
		return producer.ProviderImpl(p.l)(p.ctx)(buff2.EnvCommandTopic)(ApplyNoExpiryCommandProvider(f, characterId, fromId, sourceId, level, statups))
	}
}

func (p *ProcessorImpl) CancelByTypes(f field.Model, characterId uint32, types []string) error {
	p.l.Debugf("Character [%d] cancelling effects by types %v.", characterId, types)
	return producer.ProviderImpl(p.l)(p.ctx)(buff2.EnvCommandTopic)(CancelByTypesCommandProvider(f, characterId, types))
}
```

- [ ] **Step 3: Build + vet**

Run: `go build ./... && go vet ./...` from `services/atlas-channel/atlas.com/channel`.
Expected: clean.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/character/buff
git commit -m "feat(channel): CancelByTypes + no-expiry apply buff commands (task-167 FR-1.2/FR-1.4)"
```

---

### Task 11: atlas-channel — beacon mirror registry

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/character/buff/beacon.go`
- Test: `services/atlas-channel/atlas.com/channel/character/buff/beacon_test.go`

**Interfaces:**
- Consumes: nothing new (sync/RWMutex singleton, tenant model) — pattern copied from `monster.StatusMirror` (`services/atlas-channel/atlas.com/channel/monster/status_mirror.go`).
- Produces (Task 12 uses these):
  - `type BeaconEntry struct { SourceId int32; Level byte; MobId int32 }`
  - `GetBeaconMirror() *BeaconMirror`
  - `(*BeaconMirror).Set(t tenant.Model, characterId uint32, e BeaconEntry)`
  - `(*BeaconMirror).Clear(t tenant.Model, characterId uint32)`
  - `(*BeaconMirror).Get(t tenant.Model, characterId uint32) (BeaconEntry, bool)`
- Known limitation (accepted, design §5.7): process-local; after a channel restart it repopulates only from subsequent events, so an unrelated give to a still-locked character can drop the lock visual pre-95 until re-cast. Note this in the type's doc comment.

- [ ] **Step 1: Write the failing test** — `beacon_test.go`:

```go
package buff

import (
	"sync"
	"testing"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
)

func newTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	tn, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return tn
}

func TestBeaconMirrorSetGetClear(t *testing.T) {
	beaconMirrorOnce = sync.Once{}
	beaconMirror = nil
	m := GetBeaconMirror()
	tn := newTestTenant(t)

	if _, ok := m.Get(tn, 100); ok {
		t.Fatal("empty mirror must miss")
	}

	m.Set(tn, 100, BeaconEntry{SourceId: 5211006, Level: 1, MobId: 1000001})
	e, ok := m.Get(tn, 100)
	if !ok || e.MobId != 1000001 || e.SourceId != 5211006 {
		t.Fatalf("get after set: got %+v ok=%v", e, ok)
	}

	// Re-set replaces (re-cast on another monster).
	m.Set(tn, 100, BeaconEntry{SourceId: 5220011, Level: 10, MobId: 1000002})
	e, _ = m.Get(tn, 100)
	if e.MobId != 1000002 || e.SourceId != 5220011 {
		t.Fatalf("re-set must replace: got %+v", e)
	}

	m.Clear(tn, 100)
	if _, ok := m.Get(tn, 100); ok {
		t.Fatal("get after clear must miss")
	}
}

func TestBeaconMirrorTenantIsolation(t *testing.T) {
	beaconMirrorOnce = sync.Once{}
	beaconMirror = nil
	m := GetBeaconMirror()
	t1 := newTestTenant(t)
	t2 := newTestTenant(t)

	m.Set(t1, 100, BeaconEntry{SourceId: 5211006, Level: 1, MobId: 1000001})
	if _, ok := m.Get(t2, 100); ok {
		t.Fatal("tenant 2 must not see tenant 1's beacon")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./character/buff/ -run TestBeaconMirror -v`
Expected: FAIL — `undefined: beaconMirrorOnce`, `GetBeaconMirror`.

- [ ] **Step 3: Implement `beacon.go`** (singleton idiom copied from `monster/status_mirror.go:70-80`):

```go
package buff

import (
	"sync"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
)

// BeaconEntry is the channel-local projection of a character's active
// HOMING_BEACON lock (statup amount = locked monster object id).
type BeaconEntry struct {
	SourceId int32
	Level    byte
	MobId    int32
}

// BeaconMirror tracks each character's active beacon from buff APPLIED /
// EXPIRED events so the local-give path can re-carry the populated
// GuidedBullet block on unrelated gives (design.md §3 F2: pre-95 clients
// overwrite the stored beacon from every local give trailer).
//
// Process-local by design: after a channel restart it repopulates only from
// subsequent events, so an unrelated give to a still-locked character may
// drop the lock visual until re-cast (accepted degradation, design.md §5.7).
type BeaconMirror struct {
	mu        sync.RWMutex
	perTenant map[uuid.UUID]map[uint32]BeaconEntry
}

var (
	beaconMirror     *BeaconMirror
	beaconMirrorOnce sync.Once
)

// GetBeaconMirror returns the process-wide singleton mirror, lazily
// initialising it on first call.
func GetBeaconMirror() *BeaconMirror {
	beaconMirrorOnce.Do(func() {
		beaconMirror = &BeaconMirror{perTenant: make(map[uuid.UUID]map[uint32]BeaconEntry)}
	})
	return beaconMirror
}

func (m *BeaconMirror) Set(t tenant.Model, characterId uint32, e BeaconEntry) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.perTenant[t.Id()]
	if !ok {
		c = make(map[uint32]BeaconEntry)
		m.perTenant[t.Id()] = c
	}
	c[characterId] = e
}

func (m *BeaconMirror) Clear(t tenant.Model, characterId uint32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if c, ok := m.perTenant[t.Id()]; ok {
		delete(c, characterId)
	}
}

func (m *BeaconMirror) Get(t tenant.Model, characterId uint32) (BeaconEntry, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	c, ok := m.perTenant[t.Id()]
	if !ok {
		return BeaconEntry{}, false
	}
	e, ok := c[characterId]
	return e, ok
}
```

If `tenant.Model` has no `Id() uuid.UUID` accessor, check how `monster/status_mirror.go` keys its `perTenant` map and use the identical expression.

- [ ] **Step 4: Run tests**

Run: `go test -race ./character/buff/ -v` — Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/character/buff/beacon.go services/atlas-channel/atlas.com/channel/character/buff/beacon_test.go
git commit -m "feat(channel): beacon mirror registry (task-167 F2 groundwork)"
```

---

### Task 12: atlas-channel — local-give merge + foreign suppression in the buff consumer

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/buff/consumer.go`
- Test: `services/atlas-channel/atlas.com/channel/kafka/consumer/buff/consumer_test.go` (create)

**Interfaces:**
- Consumes: `buff.GetBeaconMirror()` / `buff.BeaconEntry` (Task 11); `e.Body.NoExpiry` (Task 9); `charconst.TemporaryStatTypeHomingBeacon`.
- Produces: pure helpers (unit-tested; the session-touching handler bodies wire them in):
  - `beaconChange(changes []buff2.StatChange) (buff2.StatChange, bool)` — first HOMING_BEACON change.
  - `isBeaconOnly(changes []buff2.StatChange) bool` — non-empty and every change is HOMING_BEACON.
  - `mergeBeacon(bs []buff.Model, e buff.BeaconEntry) []buff.Model` — appends the synthetic no-expiry beacon buff.

- [ ] **Step 1: Write the failing tests** — `consumer_test.go`:

```go
package buff

import (
	"testing"

	"atlas-channel/character/buff"
	buff2 "atlas-channel/kafka/message/buff"

	charconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
)

func TestBeaconChange(t *testing.T) {
	_, ok := beaconChange([]buff2.StatChange{{Type: "SPEED", Amount: 20}})
	if ok {
		t.Fatal("no beacon change expected")
	}
	c, ok := beaconChange([]buff2.StatChange{
		{Type: "SPEED", Amount: 20},
		{Type: string(charconst.TemporaryStatTypeHomingBeacon), Amount: 1000001},
	})
	if !ok || c.Amount != 1000001 {
		t.Fatalf("beacon change: got %+v ok=%v", c, ok)
	}
}

func TestIsBeaconOnly(t *testing.T) {
	if isBeaconOnly(nil) {
		t.Fatal("empty changes are not beacon-only")
	}
	if isBeaconOnly([]buff2.StatChange{{Type: "SPEED", Amount: 20}}) {
		t.Fatal("SPEED is not beacon-only")
	}
	if !isBeaconOnly([]buff2.StatChange{{Type: string(charconst.TemporaryStatTypeHomingBeacon), Amount: 1000001}}) {
		t.Fatal("single HOMING_BEACON change is beacon-only")
	}
	if isBeaconOnly([]buff2.StatChange{
		{Type: string(charconst.TemporaryStatTypeHomingBeacon), Amount: 1000001},
		{Type: "SPEED", Amount: 20},
	}) {
		t.Fatal("mixed changes are not beacon-only")
	}
}

func TestMergeBeacon(t *testing.T) {
	bs := []buff.Model{}
	out := mergeBeacon(bs, buff.BeaconEntry{SourceId: 5211006, Level: 1, MobId: 1000001})
	if len(out) != 1 {
		t.Fatalf("merge: got %d buffs want 1", len(out))
	}
	b := out[0]
	if b.SourceId() != 5211006 || !b.NoExpiry() {
		t.Fatalf("merged beacon buff wrong: sourceId=%d noExpiry=%v", b.SourceId(), b.NoExpiry())
	}
	if len(b.Changes()) != 1 || b.Changes()[0].Type() != string(charconst.TemporaryStatTypeHomingBeacon) || b.Changes()[0].Amount() != 1000001 {
		t.Fatalf("merged beacon statup wrong: %+v", b.Changes())
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./kafka/consumer/buff/ -v`
Expected: FAIL — `undefined: beaconChange`, `isBeaconOnly`, `mergeBeacon`.

- [ ] **Step 3: Implement the helpers** — in `consumer.go`:

```go
// beaconChange returns the first HOMING_BEACON stat change carried by an
// event, if any.
func beaconChange(changes []buff2.StatChange) (buff2.StatChange, bool) {
	for _, c := range changes {
		if c.Type == string(charconst.TemporaryStatTypeHomingBeacon) {
			return c, true
		}
	}
	return buff2.StatChange{}, false
}

// isBeaconOnly reports whether an event's changes carry nothing but the
// beacon stat. Such events are never announced to other players: the stat is
// caster-only and the foreign GuidedBullet read path is unverified (FR-4.5).
func isBeaconOnly(changes []buff2.StatChange) bool {
	if len(changes) == 0 {
		return false
	}
	for _, c := range changes {
		if c.Type != string(charconst.TemporaryStatTypeHomingBeacon) {
			return false
		}
	}
	return true
}

// mergeBeacon appends the character's active beacon as a synthetic no-expiry
// buff so an unrelated local give re-carries the populated GuidedBullet block
// (pre-95 clients overwrite the stored beacon from every local give trailer —
// design.md §3 F2). Idempotent client-side: SetGuided on the same mob is a
// re-apply.
func mergeBeacon(bs []buff.Model, e buff.BeaconEntry) []buff.Model {
	return append(bs, buff.NewBuff(e.SourceId, e.Level, 0,
		[]stat.Model{stat.NewStat(string(charconst.TemporaryStatTypeHomingBeacon), e.MobId)},
		time.Now(), time.Time{}, true))
}
```

Add imports: `charconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"`, `"time"`.

- [ ] **Step 4: Wire the handlers**

`handleStatusEventApplied` — the complete replacement for the `IfPresentByCharacterId` callback body (type/world guards above it are unchanged; note `bs` stays the UNMERGED slice for the foreign announce, and `localBs` carries the merge):

```go
		session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, func(s session.Model) error {
			t := tenant.MustFromContext(ctx)
			// Track the active beacon from its own APPLIED event.
			if bc, ok := beaconChange(e.Body.Changes); ok {
				buff.GetBeaconMirror().Set(t, e.CharacterId, buff.BeaconEntry{SourceId: e.Body.SourceId, Level: e.Body.Level, MobId: bc.Amount})
			}

			bs := make([]buff.Model, 0)
			changes := make([]stat.Model, 0)
			for _, cm := range e.Body.Changes {
				changes = append(changes, stat.NewStat(cm.Type, cm.Amount))
			}
			bs = append(bs, buff.NewBuff(e.Body.SourceId, e.Body.Level, e.Body.Duration, changes, e.Body.CreatedAt, e.Body.ExpiresAt, e.Body.NoExpiry))

			// F2: while locked, every LOCAL give must re-carry the populated
			// beacon block (pre-95 clients overwrite the stored beacon from
			// every local give trailer). Skip when this event is itself the
			// beacon — bs already carries it.
			localBs := bs
			if _, isBeacon := beaconChange(e.Body.Changes); !isBeacon {
				if entry, ok := buff.GetBeaconMirror().Get(t, e.CharacterId); ok {
					localBs = mergeBeacon(bs, entry)
				}
			}

			err := session.Announce(l)(ctx)(wp)(charpkt.CharacterBuffGiveWriter)(writer.CharacterBuffGiveBody(localBs))(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to write new character [%d] buffs.", e.CharacterId)
			}

			// Beacon-only events are never announced to other players: the
			// stat is caster-only and the remote GuidedBullet read path is
			// unverified (FR-4.5). The foreign body uses the UNMERGED bs.
			if !isBeaconOnly(e.Body.Changes) {
				_ = _map.NewProcessor(l, ctx).ForOtherSessionsInMap(s.Field(), s.CharacterId(), func(os session.Model) error {
					err = session.Announce(l)(ctx)(wp)(charpkt.CharacterBuffGiveForeignWriter)(writer.CharacterBuffGiveForeignBody(e.CharacterId, bs))(os)
					if err != nil {
						l.WithError(err).Errorf("Unable to write new character [%d] buffs.", e.CharacterId)
						return err
					}
					return nil
				})
			}
			return nil
		})
```

(the `tenant.MustFromContext` import already exists in this file.)

`handleStatusEventExpired` — two surgical edits to the existing callback body: (a) insert at the top, before `ebs` is built:

```go
			t := tenant.MustFromContext(ctx)
			if _, ok := beaconChange(e.Body.Changes); ok {
				buff.GetBeaconMirror().Clear(t, e.CharacterId)
			}
```

and (b) wrap the existing `ForOtherSessionsInMap` foreign-cancel block in `if !isBeaconOnly(e.Body.Changes) { ... }` (block contents unchanged). The LOCAL cancel path is unchanged — with Task 8's accurate masks a beacon-only EXPIRED sends exactly the beacon bit, which the client's reset path uses to clear the lock and icon (design §2.3/§2.4; answers FR-3.2 — no value-0 give needed).

- [ ] **Step 5: Run tests + build**

Run: `go test -race ./... && go build ./...` from `services/atlas-channel/atlas.com/channel`.
Expected: PASS / clean.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/consumer/buff
git commit -m "feat(channel): beacon merge on local gives, suppress beacon-only foreign announcements (task-167 F2/FR-4.5)"
```

---

### Task 13: atlas-channel — attack-handler hook

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go`
- Test: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common_test.go` (append; create if the handler package keeps these tests elsewhere — check `ls services/atlas-channel/atlas.com/channel/socket/handler/*_test.go` first and follow the existing file)

**Interfaces:**
- Consumes: `buff.Processor.CancelByTypes` / `ApplyNoExpiry` (Task 10); `statup.NewModel` (`data/skill/effect/statup`); `skill3.OutlawHomingBeaconId` / `skill3.CorsairBullseyeId`; `mp.GetById`; `ai.DamageInfo()` / `di.MonsterId()`; `sk.Level()`.
- Produces: `beaconTargetMonsterId(monsterIds []uint32, exists func(uint32) bool) (uint32, bool)` and `beaconTryApply(l logrus.FieldLogger, ai packetmodel.AttackInfo, skillLevel byte, f field.Model, characterId uint32, deps beaconApplyDeps)` — deps-struct style mirrors `damageInfoEntryDeps` so tests drive both emit paths without Kafka.

- [ ] **Step 1: Write the failing tests** — following the handler package's existing unit-test style (pure funcs + fakes):

```go
func TestBeaconTargetMonsterId(t *testing.T) {
	existsAll := func(uint32) bool { return true }

	// Whiff: no entries.
	if _, ok := beaconTargetMonsterId(nil, existsAll); ok {
		t.Fatal("whiff must yield no target")
	}
	// Zero monster ids are skipped.
	if _, ok := beaconTargetMonsterId([]uint32{0, 0}, existsAll); ok {
		t.Fatal("zero ids must yield no target")
	}
	// Single hit.
	id, ok := beaconTargetMonsterId([]uint32{1000001}, existsAll)
	if !ok || id != 1000001 {
		t.Fatalf("single hit: got %d ok=%v", id, ok)
	}
	// Multi-entry: LAST valid entry wins (Cosmic per-monster loop order,
	// design.md §5.2; WZ has no mobCount so this is a defensive rule).
	id, ok = beaconTargetMonsterId([]uint32{1000001, 1000002}, existsAll)
	if !ok || id != 1000002 {
		t.Fatalf("last-wins: got %d ok=%v", id, ok)
	}
	// Monster missing from the field registry is skipped.
	id, ok = beaconTargetMonsterId([]uint32{1000001, 1000002}, func(m uint32) bool { return m == 1000001 })
	if !ok || id != 1000001 {
		t.Fatalf("missing-monster skip: got %d ok=%v", id, ok)
	}
}

func TestBeaconTryApply(t *testing.T) {
	mkAttack := func(skillId uint32, monsterIds ...uint32) packetmodel.AttackInfo {
		ai := packetmodel.NewAttackInfo(packetmodel.AttackTypeRanged)
		ai.SetSkillId(skillId)
		for _, m := range monsterIds {
			di := packetmodel.NewDamageInfo(1)
			di.SetMonsterId(m)
			ai.AddDamageInfo(*di)
		}
		return *ai
	}

	type call struct {
		kind  string
		mobId int32
	}
	l := logrus.New()
	l.SetOutput(io.Discard)
	record := func(calls *[]call) beaconApplyDeps {
		return beaconApplyDeps{
			monsterExists: func(uint32) bool { return true },
			cancelByTypes: func(types []string) error {
				*calls = append(*calls, call{kind: "cancel"})
				return nil
			},
			applyNoExpiry: func(sourceId int32, level byte, mobId int32) error {
				*calls = append(*calls, call{kind: "apply", mobId: mobId})
				return nil
			},
		}
	}

	// Non-beacon skill: nothing emitted.
	var calls []call
	beaconTryApply(l, mkAttack(4001344, 1000001), 1, field.Model{}, 42, record(&calls))
	if len(calls) != 0 {
		t.Fatalf("non-beacon skill must emit nothing, got %v", calls)
	}

	// Whiff: nothing emitted, prior lock untouched (FR-1.5).
	calls = nil
	beaconTryApply(l, mkAttack(uint32(skill3.OutlawHomingBeaconId)), 1, field.Model{}, 42, record(&calls))
	if len(calls) != 0 {
		t.Fatalf("whiff must emit nothing, got %v", calls)
	}

	// Hit: CANCEL_BY_TYPES then APPLY, in that order (FR-1.4).
	calls = nil
	beaconTryApply(l, mkAttack(uint32(skill3.CorsairBullseyeId), 1000001), 10, field.Model{}, 42, record(&calls))
	if len(calls) != 2 || calls[0].kind != "cancel" || calls[1].kind != "apply" || calls[1].mobId != 1000001 {
		t.Fatalf("hit must emit cancel then apply(mobId): got %v", calls)
	}

	// Cancel failure: apply is not attempted (lock state stays consistent).
	calls = nil
	deps := record(&calls)
	deps.cancelByTypes = func([]string) error { return errors.New("kafka down") }
	beaconTryApply(l, mkAttack(uint32(skill3.OutlawHomingBeaconId), 1000001), 1, field.Model{}, 42, deps)
	for _, c := range calls {
		if c.kind == "apply" {
			t.Fatal("apply must not run after cancel failure")
		}
	}
}
```

Adjust the `mkAttack` construction to `packetmodel.AttackInfo`'s actual builder API — read `libs/atlas-packet/model/attack_info.go` first and use its real constructor/setters (`NewAttackInfo`, `SetSkillId`, `AddDamageInfo` are to be confirmed there; `DamageInfo` setters `NewDamageInfo(hits)`, `SetMonsterId` are confirmed at `damage_info.go:12,94`). The test imports `"errors"`, `"io"`, `"github.com/sirupsen/logrus"`, `"atlas-channel/socket/handler"`-local types, `packetmodel`, `field`, and `skill3` per the neighboring test files.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./socket/handler/ -run 'Beacon' -v`
Expected: FAIL — `undefined: beaconTargetMonsterId`, `beaconApplyDeps`, `beaconTryApply`.

- [ ] **Step 3: Implement** — in `character_attack_common.go` (near the MP Eater helpers, which are the pattern: errors logged and swallowed, pipeline unaffected):

```go
// beaconApplyDeps groups the emit closures beaconTryApply needs so tests can
// record ordering without Kafka. Mirrors the damageInfoEntryDeps pattern.
type beaconApplyDeps struct {
	monsterExists func(monsterId uint32) bool
	cancelByTypes func(types []string) error
	applyNoExpiry func(sourceId int32, level byte, mobId int32) error
}

// beaconTargetMonsterId picks the beacon lock target: the LAST damage entry
// whose monster id is nonzero and still exists in the field registry. WZ has
// no mobCount for 5211006/5220011 (single-target), so multiple entries only
// occur on malformed packets; last-valid-wins matches Cosmic's per-monster
// loop order (AbstractDealDamageHandler.java:346-348, design.md §5.2).
func beaconTargetMonsterId(monsterIds []uint32, exists func(uint32) bool) (uint32, bool) {
	var target uint32
	found := false
	for _, id := range monsterIds {
		if id == 0 || !exists(id) {
			continue
		}
		target = id
		found = true
	}
	return target, found
}

// beaconTryApply handles Homing Beacon (5211006) / Bullseye (5220011): on a
// valid strike it emits CANCEL_BY_TYPES(HOMING_BEACON) then a no-expiry APPLY
// whose statup amount is the struck monster's object id. Both commands share
// the character-keyed buff command topic, so ordering is guaranteed and the
// old lock (either skill id) is always cleared first (FR-1.4). A whiff emits
// nothing (FR-1.5). Failures are logged and swallowed — the attack pipeline
// (damage, projectile, broadcast) is already complete and must not be
// affected (FR-1.6).
func beaconTryApply(l logrus.FieldLogger, ai packetmodel.AttackInfo, skillLevel byte, f field.Model, characterId uint32, deps beaconApplyDeps) {
	sid := skill3.Id(ai.SkillId())
	if sid != skill3.OutlawHomingBeaconId && sid != skill3.CorsairBullseyeId {
		return
	}

	ids := make([]uint32, 0, len(ai.DamageInfo()))
	for _, di := range ai.DamageInfo() {
		ids = append(ids, di.MonsterId())
	}
	mobId, ok := beaconTargetMonsterId(ids, deps.monsterExists)
	if !ok {
		return
	}

	if err := deps.cancelByTypes([]string{string(charconst.TemporaryStatTypeHomingBeacon)}); err != nil {
		l.WithError(err).Errorf("Beacon: unable to cancel prior HOMING_BEACON for character [%d]; skipping apply.", characterId)
		return
	}
	if err := deps.applyNoExpiry(int32(ai.SkillId()), skillLevel, int32(mobId)); err != nil {
		l.WithError(err).Errorf("Beacon: unable to apply HOMING_BEACON (mob [%d]) for character [%d].", mobId, characterId)
	}
}
```

Add import `charconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"` (the file already imports `skill3`, `field`, `packetmodel`, `logrus`).

Wire the production call in `processAttack`, replacing the `// TODO Homing Beacon / Bullseye` line (after the projectile emit block; `mp`, `sk`, `s` in scope):

```go
					if ai.AttackType() == packetmodel.AttackTypeRanged && ai.SkillId() > 0 {
						bp := buff.NewProcessor(l, ctx)
						beaconTryApply(l, ai, sk.Level(), s.Field(), s.CharacterId(), beaconApplyDeps{
							monsterExists: func(monsterId uint32) bool {
								_, gErr := mp.GetById(monsterId)
								return gErr == nil
							},
							cancelByTypes: func(types []string) error {
								return bp.CancelByTypes(s.Field(), s.CharacterId(), types)
							},
							applyNoExpiry: func(sourceId int32, level byte, mobId int32) error {
								return bp.ApplyNoExpiry(s.Field(), s.CharacterId(), sourceId, level,
									[]statup.Model{statup.NewModel(string(charconst.TemporaryStatTypeHomingBeacon), mobId)})(s.CharacterId())
							},
						})
					}
```

with imports `"atlas-channel/character/buff"` and `"atlas-channel/data/skill/effect/statup"` added. `fromId` = the caster (`s.CharacterId()` — self-applied, matching Cosmic's `applyBeaconBuff(applyfrom=applyto)`). Delete the TODO line; leave every other TODO in the block untouched.

- [ ] **Step 4: Run tests + build**

Run: `go test -race ./... && go build ./... && go vet ./...` from `services/atlas-channel/atlas.com/channel`.
Expected: PASS / clean.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler
git commit -m "feat(channel): Homing Beacon / Bullseye lock-on via attack handler (task-167 FR-1)"
```

---

### Task 14: Full verification + acceptance checklist

**Files:**
- Modify: `docs/tasks/task-167-homing-beacon-bullseye/plan.md` (check off), no code.

- [ ] **Step 1: Module gates** — from the worktree root, for each changed module (`services/atlas-buffs/atlas.com/buffs`, `services/atlas-channel/atlas.com/channel`, `libs/atlas-packet`):

```bash
(cd <module> && go test -race ./... && go vet ./... && go build ./...)
```

Expected: all clean. (Do NOT prefix with a global `GOWORK=off`.)

- [ ] **Step 2: Redis key guard** — from the worktree root:

```bash
tools/redis-key-guard.sh
```

Expected: clean (the beacon mirror is a plain in-memory map, no Redis involvement).

- [ ] **Step 3: Docker bakes** — from the worktree root (mandatory, CLAUDE.md):

```bash
docker buildx bake atlas-buffs
docker buildx bake atlas-channel
```

Expected: both images build. No Dockerfile/go.work edits were needed (no new libs).

- [ ] **Step 4: Fixture sweep** — `go test ./... 2>&1 | tail -5` in `libs/atlas-packet` once more after everything is merged into the branch; any fixture that changed byte expectations without an explicit task step above is a defect, not a fixture update.

- [ ] **Step 5: Code review** — run `superpowers:requesting-code-review` (plan-adherence + backend-guidelines reviewers; Go-only change set) BEFORE any PR. Findings go to `docs/tasks/task-167-homing-beacon-bullseye/audit.md`.

- [ ] **Step 6: Live acceptance (v83 tenant, manual — record results in the task folder)** — from design §6:
  1. Outlaw casts Homing Beacon at a monster → subsequent shots visibly home.
  2. Corsair Bullseye behaves identically.
  3. Re-cast on another monster moves the lock; old target unflagged.
  4. All-miss cast → lock unchanged.
  5. Gain and expire an unrelated buff (e.g. speed potion) mid-lock → lock survives (F1/F2 proof).
  6. Map change → lock and icon clear; returning does not resurrect it.
  7. Kill the locked target → no server cancel (parity, FR-3.3).
  8. Death/respawn and logout clear the beacon (existing CANCEL_ALL flows).
  9. Icon renders with no duration bar (PRD Open Question 5 — only observable live).

- [ ] **Step 7: Commit any checklist/doc updates**

```bash
git add docs/tasks/task-167-homing-beacon-bullseye/
git commit -m "docs(task-167): verification results"
```

---

## Task order and independence

Strict order: 1 → 2 → 3 → 4 (buffs), 5 → 6 (packet encode), 7 → 8 (IDA then cancel masks), 9 → 10 → 11 → 12 → 13 (channel), 14 last. Cross-stream: Tasks 5–8 (lib) are independent of Tasks 1–4 (buffs) and may run in parallel streams; Task 9 depends on Task 2's field names; Task 12 depends on 8, 9, 11; Task 13 depends on 10.

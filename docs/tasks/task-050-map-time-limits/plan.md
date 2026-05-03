# Map Time Limits Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** End-to-end per-character map-stay timer for time-limited maps: register on map entry, render countdown via the v83 client, warp to `forcedReturnMapId` on expiration, and persist `forcedReturnMapId` on disconnect / channel-change so logout-camping can't bypass the limit.

**Architecture:** Owner is **atlas-maps** (deviates from PRD; see [`design.md`](./design.md) §1). atlas-maps holds an in-memory `Registry` of `Entry` values keyed by `(tenant.Id, characterId)`, each backed by a per-entry `time.AfterFunc` goroutine. atlas-maps reacts to `MAP_CHANGED` (start/cancel), `SESSION_DESTROYED` (forced-return on logout/channel-change), and `CHANNEL_CHANGED` (belt-and-suspenders fallback). atlas-maps publishes a new `MAP_TIMER_STARTED` event on `EVENT_TOPIC_MAP_STATUS` and a `CHANGE_MAP` command on `COMMAND_TOPIC_CHARACTER`. atlas-channel becomes a dumb renderer that hears `MAP_TIMER_STARTED` and emits `fieldcb.NewTimerClock(seconds)` to the live session.

**Tech Stack:** Go 1.22+, `libs/atlas-kafka` consumer/producer, `libs/atlas-tenant`, `libs/atlas-constants/{field,channel,map,world}`, `libs/atlas-packet/field/clientbound` (Clock writer), OpenTelemetry tracer (`go.opentelemetry.io/otel`).

---

## Conventions

- Worktree path is whatever `superpowers:using-git-worktrees` allocates. All `cd` commands assume **CWD = repo root** (the worktree root). Substitute as needed.
- Every commit message starts with `feat(atlas-maps):`, `feat(atlas-channel):`, `test(atlas-maps):`, etc.
- For each task, run the listed test command from inside the affected service directory (`services/<svc>/atlas.com/<svc>/`).
- Use `git add <specific files>` — never `git add -A`.

---

## Task 1: Add `data/map/info` package — Model, getters, and `IsTimeLimited` predicate (TDD red)

**Files:**
- Create: `services/atlas-maps/atlas.com/maps/data/map/info/model.go`
- Create: `services/atlas-maps/atlas.com/maps/data/map/info/model_test.go`

- [ ] **Step 1: Write the failing model test**

Create `services/atlas-maps/atlas.com/maps/data/map/info/model_test.go`:

```go
package info

import (
	"testing"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/stretchr/testify/require"
)

func TestModel_Getters(t *testing.T) {
	m := Model{
		id:                _map.Id(100000000),
		timeLimit:         600,
		forcedReturnMapId: _map.Id(100000201),
	}
	require.Equal(t, _map.Id(100000000), m.Id())
	require.Equal(t, int32(600), m.TimeLimit())
	require.Equal(t, _map.Id(100000201), m.ForcedReturnMapId())
}

func TestModel_IsTimeLimited_BothFieldsPresent(t *testing.T) {
	m := Model{timeLimit: 600, forcedReturnMapId: _map.Id(100000201)}
	require.True(t, m.IsTimeLimited())
}

func TestModel_IsTimeLimited_TimeLimitZero(t *testing.T) {
	m := Model{timeLimit: 0, forcedReturnMapId: _map.Id(100000201)}
	require.False(t, m.IsTimeLimited(), "timeLimit=0 must disable")
}

func TestModel_IsTimeLimited_TimeLimitNegative(t *testing.T) {
	m := Model{timeLimit: -1, forcedReturnMapId: _map.Id(100000201)}
	require.False(t, m.IsTimeLimited(), "negative timeLimit treated as disabled")
}

func TestModel_IsTimeLimited_ForcedReturnSentinel(t *testing.T) {
	m := Model{timeLimit: 600, forcedReturnMapId: _map.Id(999999999)}
	require.False(t, m.IsTimeLimited(), "999999999 sentinel must disable")
}

func TestModel_IsTimeLimited_BothMissing(t *testing.T) {
	m := Model{timeLimit: 0, forcedReturnMapId: _map.Id(999999999)}
	require.False(t, m.IsTimeLimited())
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run from `services/atlas-maps/atlas.com/maps/`:
```
go test ./data/map/info/...
```
Expected: build failure (`undefined: Model` and `undefined: id` etc.).

- [ ] **Step 3: Implement the model**

Create `services/atlas-maps/atlas.com/maps/data/map/info/model.go`:

```go
package info

import (
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

const noForcedReturnMapId = _map.Id(999999999)

type Model struct {
	id                _map.Id
	timeLimit         int32
	forcedReturnMapId _map.Id
}

func (m Model) Id() _map.Id {
	return m.id
}

func (m Model) TimeLimit() int32 {
	return m.timeLimit
}

func (m Model) ForcedReturnMapId() _map.Id {
	return m.forcedReturnMapId
}

func (m Model) IsTimeLimited() bool {
	return m.timeLimit > 0 && m.forcedReturnMapId != noForcedReturnMapId
}
```

- [ ] **Step 4: Run the test to verify it passes**

```
go test ./data/map/info/...
```
Expected: PASS.

- [ ] **Step 5: Commit**

```
git add services/atlas-maps/atlas.com/maps/data/map/info/model.go services/atlas-maps/atlas.com/maps/data/map/info/model_test.go
git commit -m "feat(atlas-maps): add data/map/info Model with IsTimeLimited predicate"
```

---

## Task 2: Add `data/map/info` REST DTO + Extract

**Files:**
- Create: `services/atlas-maps/atlas.com/maps/data/map/info/rest.go`
- Create: `services/atlas-maps/atlas.com/maps/data/map/info/rest_test.go`

- [ ] **Step 1: Write the failing rest test**

Create `services/atlas-maps/atlas.com/maps/data/map/info/rest_test.go`:

```go
package info

import (
	"testing"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/stretchr/testify/require"
)

func TestExtract_PopulatesAllFields(t *testing.T) {
	rm := RestModel{
		Id:                _map.Id(100000000),
		TimeLimit:         600,
		ForcedReturnMapId: _map.Id(100000201),
	}
	m, err := Extract(rm)
	require.NoError(t, err)
	require.Equal(t, _map.Id(100000000), m.Id())
	require.Equal(t, int32(600), m.TimeLimit())
	require.Equal(t, _map.Id(100000201), m.ForcedReturnMapId())
}

func TestRestModel_ImplementsJSONApiResource(t *testing.T) {
	rm := RestModel{Id: _map.Id(100000000)}
	require.Equal(t, "maps", rm.GetName())
	require.Equal(t, "100000000", rm.GetID())
	require.NoError(t, rm.SetID("200000000"))
	require.Equal(t, _map.Id(200000000), rm.Id)
}
```

- [ ] **Step 2: Run the test**

```
go test ./data/map/info/...
```
Expected: build failure (`undefined: RestModel`, `undefined: Extract`).

- [ ] **Step 3: Implement RestModel + Extract**

Create `services/atlas-maps/atlas.com/maps/data/map/info/rest.go`. Mirror `data/map/script/rest.go`:

```go
package info

import (
	"strconv"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

type RestModel struct {
	Id                _map.Id `json:"-"`
	TimeLimit         int32   `json:"timeLimit"`
	ForcedReturnMapId _map.Id `json:"forcedReturnMapId"`
}

func (r RestModel) GetName() string {
	return "maps"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = _map.Id(id)
	return nil
}

func (r *RestModel) SetToOneReferenceID(_ string, _ string) error {
	return nil
}

func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		id:                rm.Id,
		timeLimit:         rm.TimeLimit,
		forcedReturnMapId: rm.ForcedReturnMapId,
	}, nil
}
```

- [ ] **Step 4: Run the test**

```
go test ./data/map/info/...
```
Expected: PASS.

- [ ] **Step 5: Commit**

```
git add services/atlas-maps/atlas.com/maps/data/map/info/rest.go services/atlas-maps/atlas.com/maps/data/map/info/rest_test.go
git commit -m "feat(atlas-maps): add data/map/info RestModel + Extract"
```

---

## Task 3: Add `data/map/info` Processor + REST request

**Files:**
- Create: `services/atlas-maps/atlas.com/maps/data/map/info/requests.go`
- Create: `services/atlas-maps/atlas.com/maps/data/map/info/processor.go`

No new test (Processor is an HTTP shim covered by integration paths).

- [ ] **Step 1: Implement requests.go (mirror `data/map/script/requests.go`)**

```go
package info

import (
	"fmt"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const mapsResource = "data/maps/%d"

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

func requestMap(mapId _map.Id) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+mapsResource, mapId))
}
```

- [ ] **Step 2: Implement processor.go**

Mirror atlas-channel's `data/map/processor.go` (cache + per-key load mutex):

```go
package info

import (
	"context"
	"sync"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	GetById(mapId _map.Id) (Model, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

type cacheKey struct {
	tenantId uuid.UUID
	mapId    _map.Id
}

var (
	mapCache  sync.Map
	mapLoadMu sync.Map
)

func (p *ProcessorImpl) GetById(mapId _map.Id) (Model, error) {
	t := tenant.MustFromContext(p.ctx)
	key := cacheKey{tenantId: t.Id(), mapId: mapId}

	if cached, ok := mapCache.Load(key); ok {
		return cached.(Model), nil
	}

	muIface, _ := mapLoadMu.LoadOrStore(key, &sync.Mutex{})
	mu := muIface.(*sync.Mutex)
	mu.Lock()
	defer mu.Unlock()

	if cached, ok := mapCache.Load(key); ok {
		return cached.(Model), nil
	}

	m, err := requests.Provider[RestModel, Model](p.l, p.ctx)(requestMap(mapId), Extract)()
	if err != nil {
		return Model{}, err
	}
	mapCache.Store(key, m)
	return m, nil
}
```

- [ ] **Step 3: Verify it builds**

```
go build ./data/map/info/...
```
Expected: success.

- [ ] **Step 4: Commit**

```
git add services/atlas-maps/atlas.com/maps/data/map/info/processor.go services/atlas-maps/atlas.com/maps/data/map/info/requests.go
git commit -m "feat(atlas-maps): add data/map/info Processor with tenant-scoped cache"
```

---

## Task 4: Add `MAP_TIMER_STARTED` event constant + body to atlas-maps

**Files:**
- Modify: `services/atlas-maps/atlas.com/maps/kafka/message/map/kafka.go`
- Modify: `services/atlas-maps/atlas.com/maps/kafka/message/map/kafka_test.go`

- [ ] **Step 1: Write the failing serialization test**

Append to `services/atlas-maps/atlas.com/maps/kafka/message/map/kafka_test.go`:

```go
func TestStatusEvent_MapTimerStarted_Serialization(t *testing.T) {
	event := StatusEvent[MapTimerStarted]{
		TransactionId: uuid.MustParse("12345678-1234-5678-1234-567812345678"),
		WorldId:       world.Id(1),
		ChannelId:     channel.Id(2),
		MapId:         _map.Id(100000000),
		Type:          EventTopicMapStatusTypeMapTimerStarted,
		Body: MapTimerStarted{
			CharacterId: 12345,
			Seconds:     600,
		},
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal event: %v", err)
	}

	var decoded StatusEvent[MapTimerStarted]
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal event: %v", err)
	}
	if decoded.Type != EventTopicMapStatusTypeMapTimerStarted {
		t.Errorf("Expected Type %s, got %s", EventTopicMapStatusTypeMapTimerStarted, decoded.Type)
	}
	if decoded.Body.CharacterId != 12345 {
		t.Errorf("Expected CharacterId 12345, got %d", decoded.Body.CharacterId)
	}
	if decoded.Body.Seconds != 600 {
		t.Errorf("Expected Seconds 600, got %d", decoded.Body.Seconds)
	}
}

func TestEventTypeConstant_MapTimerStarted(t *testing.T) {
	if EventTopicMapStatusTypeMapTimerStarted != "MAP_TIMER_STARTED" {
		t.Errorf("Expected EventTopicMapStatusTypeMapTimerStarted to be 'MAP_TIMER_STARTED', got '%s'", EventTopicMapStatusTypeMapTimerStarted)
	}
}
```

- [ ] **Step 2: Run the test**

```
go test ./kafka/message/map/...
```
Expected: build failure (`undefined: EventTopicMapStatusTypeMapTimerStarted`, `undefined: MapTimerStarted`).

- [ ] **Step 3: Add the constant + body to `kafka.go`**

In `services/atlas-maps/atlas.com/maps/kafka/message/map/kafka.go`, add to the `const` block:

```go
	EventTopicMapStatusTypeMapTimerStarted = "MAP_TIMER_STARTED"
```

And add a new body type at the bottom of the file:

```go
type MapTimerStarted struct {
	CharacterId uint32 `json:"characterId"`
	Seconds     uint32 `json:"seconds"`
}
```

- [ ] **Step 4: Run the test**

```
go test ./kafka/message/map/...
```
Expected: PASS.

- [ ] **Step 5: Commit**

```
git add services/atlas-maps/atlas.com/maps/kafka/message/map/kafka.go services/atlas-maps/atlas.com/maps/kafka/message/map/kafka_test.go
git commit -m "feat(atlas-maps): add MAP_TIMER_STARTED event envelope"
```

---

## Task 5: Add `COMMAND_TOPIC_CHARACTER` envelope + ChangeMap command body to atlas-maps

**Files:**
- Modify: `services/atlas-maps/atlas.com/maps/kafka/message/character/kafka.go`
- Create: `services/atlas-maps/atlas.com/maps/kafka/message/character/command_test.go`

- [ ] **Step 1: Write the failing serialization test**

Create `services/atlas-maps/atlas.com/maps/kafka/message/character/command_test.go`:

```go
package character

import (
	"encoding/json"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

func TestCommand_ChangeMap_Serialization(t *testing.T) {
	cmd := Command[ChangeMapBody]{
		TransactionId: uuid.MustParse("12345678-1234-5678-1234-567812345678"),
		WorldId:       world.Id(1),
		CharacterId:   42,
		Type:          CommandChangeMap,
		Body: ChangeMapBody{
			ChannelId: channel.Id(2),
			MapId:     _map.Id(100000000),
			Instance:  uuid.Nil,
			PortalId:  0,
		},
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		t.Fatalf("Failed to marshal command: %v", err)
	}

	var decoded Command[ChangeMapBody]
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal command: %v", err)
	}

	if decoded.Type != CommandChangeMap {
		t.Errorf("Expected Type %s, got %s", CommandChangeMap, decoded.Type)
	}
	if decoded.Body.MapId != _map.Id(100000000) {
		t.Errorf("Expected MapId 100000000, got %d", decoded.Body.MapId)
	}
	if decoded.Body.Instance != uuid.Nil {
		t.Errorf("Expected Instance Nil, got %v", decoded.Body.Instance)
	}
}

func TestCommandTypeConstant_ChangeMap(t *testing.T) {
	if CommandChangeMap != "CHANGE_MAP" {
		t.Errorf("Expected CommandChangeMap to be 'CHANGE_MAP', got '%s'", CommandChangeMap)
	}
	if EnvCommandTopic != "COMMAND_TOPIC_CHARACTER" {
		t.Errorf("Expected EnvCommandTopic to be 'COMMAND_TOPIC_CHARACTER', got '%s'", EnvCommandTopic)
	}
}
```

- [ ] **Step 2: Run the test**

```
go test ./kafka/message/character/...
```
Expected: build failure (`undefined: Command`, `undefined: ChangeMapBody`, `undefined: CommandChangeMap`, `undefined: EnvCommandTopic`).

- [ ] **Step 3: Append the command envelope to `kafka.go`**

Append to `services/atlas-maps/atlas.com/maps/kafka/message/character/kafka.go`:

```go
const (
	EnvCommandTopic  = "COMMAND_TOPIC_CHARACTER"
	CommandChangeMap = "CHANGE_MAP"
)

type Command[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	WorldId       world.Id  `json:"worldId"`
	CharacterId   uint32    `json:"characterId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type ChangeMapBody struct {
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
	PortalId  uint32     `json:"portalId"`
}
```

(All four imports — `channel`, `_map`, `world`, `uuid` — are already imported by the existing file.)

- [ ] **Step 4: Run the test**

```
go test ./kafka/message/character/...
```
Expected: PASS.

- [ ] **Step 5: Commit**

```
git add services/atlas-maps/atlas.com/maps/kafka/message/character/kafka.go services/atlas-maps/atlas.com/maps/kafka/message/character/command_test.go
git commit -m "feat(atlas-maps): add CHANGE_MAP command envelope"
```

---

## Task 6: Add `kafka/message/session` consumer envelope to atlas-maps

**Files:**
- Create: `services/atlas-maps/atlas.com/maps/kafka/message/session/kafka.go`

No test — this file is type defs only; coverage comes from the consumer test in Task 14.

- [ ] **Step 1: Create the file**

Mirror `services/atlas-asset-expiration/atlas.com/asset-expiration/kafka/message/session/kafka.go`:

```go
package session

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	EnvEventTopicSessionStatus      = "EVENT_TOPIC_SESSION_STATUS"
	EventSessionStatusIssuerLogin   = "LOGIN"
	EventSessionStatusIssuerChannel = "CHANNEL"
	EventSessionStatusTypeCreated   = "CREATED"
	EventSessionStatusTypeDestroyed = "DESTROYED"
)

type StatusEvent struct {
	SessionId   uuid.UUID  `json:"sessionId"`
	AccountId   uint32     `json:"accountId"`
	CharacterId uint32     `json:"characterId"`
	WorldId     world.Id   `json:"worldId"`
	ChannelId   channel.Id `json:"channelId"`
	Issuer      string     `json:"issuer"`
	Type        string     `json:"type"`
}
```

- [ ] **Step 2: Verify build**

```
go build ./kafka/message/session/...
```
Expected: success.

- [ ] **Step 3: Commit**

```
git add services/atlas-maps/atlas.com/maps/kafka/message/session/kafka.go
git commit -m "feat(atlas-maps): add session-status kafka envelope"
```

---

## Task 7: Add timer Entry model + Builder

**Files:**
- Create: `services/atlas-maps/atlas.com/maps/map/timer/model.go`
- Create: `services/atlas-maps/atlas.com/maps/map/timer/model_test.go`

- [ ] **Step 1: Write the failing model test**

Create `services/atlas-maps/atlas.com/maps/map/timer/model_test.go`:

```go
package timer

import (
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func mkTenant(t *testing.T) tenant.Model {
	t.Helper()
	tt, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	return tt
}

func TestEntry_GettersExposeAllFields(t *testing.T) {
	tt := mkTenant(t)
	f := field.NewBuilder(0, 0, 100000000).SetInstance(uuid.Nil).Build()
	tok := uuid.New()
	expires := time.Now().Add(10 * time.Minute)

	e := NewEntryBuilder().
		SetTenant(tt).
		SetCharacterId(42).
		SetField(f).
		SetForcedReturnMapId(_map.Id(100000201)).
		SetSeconds(600).
		SetToken(tok).
		SetExpiresAt(expires).
		Build()

	require.Equal(t, tt, e.Tenant())
	require.Equal(t, uint32(42), e.CharacterId())
	require.True(t, e.Field().Equals(f))
	require.Equal(t, _map.Id(100000201), e.ForcedReturnMapId())
	require.Equal(t, uint32(600), e.Seconds())
	require.Equal(t, tok, e.Token())
	require.Equal(t, expires, e.ExpiresAt())
}
```

- [ ] **Step 2: Run the test**

```
go test ./map/timer/...
```
Expected: build failure.

- [ ] **Step 3: Implement `model.go`**

Create `services/atlas-maps/atlas.com/maps/map/timer/model.go`:

```go
package timer

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
)

type Entry struct {
	tenant            tenant.Model
	characterId       uint32
	field             field.Model
	forcedReturnMapId _map.Id
	seconds           uint32
	token             uuid.UUID
	expiresAt         time.Time
	timer             *time.Timer
}

func (e Entry) Tenant() tenant.Model        { return e.tenant }
func (e Entry) CharacterId() uint32         { return e.characterId }
func (e Entry) Field() field.Model          { return e.field }
func (e Entry) ForcedReturnMapId() _map.Id  { return e.forcedReturnMapId }
func (e Entry) Seconds() uint32             { return e.seconds }
func (e Entry) Token() uuid.UUID            { return e.token }
func (e Entry) ExpiresAt() time.Time        { return e.expiresAt }
func (e Entry) Timer() *time.Timer          { return e.timer }

type EntryBuilder struct {
	e Entry
}

func NewEntryBuilder() *EntryBuilder { return &EntryBuilder{} }

func (b *EntryBuilder) SetTenant(t tenant.Model) *EntryBuilder            { b.e.tenant = t; return b }
func (b *EntryBuilder) SetCharacterId(id uint32) *EntryBuilder            { b.e.characterId = id; return b }
func (b *EntryBuilder) SetField(f field.Model) *EntryBuilder              { b.e.field = f; return b }
func (b *EntryBuilder) SetForcedReturnMapId(id _map.Id) *EntryBuilder     { b.e.forcedReturnMapId = id; return b }
func (b *EntryBuilder) SetSeconds(s uint32) *EntryBuilder                 { b.e.seconds = s; return b }
func (b *EntryBuilder) SetToken(t uuid.UUID) *EntryBuilder                { b.e.token = t; return b }
func (b *EntryBuilder) SetExpiresAt(t time.Time) *EntryBuilder            { b.e.expiresAt = t; return b }
func (b *EntryBuilder) SetTimer(t *time.Timer) *EntryBuilder              { b.e.timer = t; return b }
func (b *EntryBuilder) Build() Entry                                       { return b.e }
```

- [ ] **Step 4: Run the test**

```
go test ./map/timer/...
```
Expected: PASS.

- [ ] **Step 5: Commit**

```
git add services/atlas-maps/atlas.com/maps/map/timer/model.go services/atlas-maps/atlas.com/maps/map/timer/model_test.go
git commit -m "feat(atlas-maps): add timer Entry model + builder"
```

---

## Task 8: Add timer Registry — basic Add/Get/Cancel

**Files:**
- Create: `services/atlas-maps/atlas.com/maps/map/timer/registry.go`
- Create: `services/atlas-maps/atlas.com/maps/map/timer/registry_test.go`

- [ ] **Step 1: Write the failing registry tests (Add + Cancel)**

Create `services/atlas-maps/atlas.com/maps/map/timer/registry_test.go`:

```go
package timer

import (
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func mkRegEntry(t *testing.T, tenantA interface{ /* placeholder */ }) Entry {
	t.Helper()
	f := field.NewBuilder(0, 0, 100000000).SetInstance(uuid.Nil).Build()
	return NewEntryBuilder().
		SetCharacterId(42).
		SetField(f).
		SetForcedReturnMapId(_map.Id(100000201)).
		SetSeconds(600).
		SetToken(uuid.New()).
		SetExpiresAt(time.Now().Add(10 * time.Minute)).
		Build()
}

func TestRegistry_Add_StoresEntry(t *testing.T) {
	tt := mkTenant(t)
	r := NewTestRegistry()
	e := mkRegEntry(t, tt)
	e2 := NewEntryBuilder().
		SetTenant(tt).
		SetCharacterId(e.CharacterId()).
		SetField(e.Field()).
		SetForcedReturnMapId(e.ForcedReturnMapId()).
		SetSeconds(e.Seconds()).
		SetToken(e.Token()).
		SetExpiresAt(e.ExpiresAt()).
		Build()

	require.NoError(t, r.Add(e2))

	got, ok := r.Get(tt, 42)
	require.True(t, ok)
	require.Equal(t, e2.Token(), got.Token())
}

func TestRegistry_Cancel_RemovesEntry(t *testing.T) {
	tt := mkTenant(t)
	r := NewTestRegistry()
	e := NewEntryBuilder().
		SetTenant(tt).
		SetCharacterId(42).
		SetForcedReturnMapId(_map.Id(100000201)).
		SetToken(uuid.New()).
		Build()
	require.NoError(t, r.Add(e))

	got, ok := r.Cancel(tt, 42)
	require.True(t, ok)
	require.Equal(t, e.Token(), got.Token())

	_, ok = r.Get(tt, 42)
	require.False(t, ok, "Cancel must remove the entry")
}

func TestRegistry_Cancel_AbsentIsNoOp(t *testing.T) {
	tt := mkTenant(t)
	r := NewTestRegistry()
	_, ok := r.Cancel(tt, 999)
	require.False(t, ok, "Cancel on absent key returns false")
}

func TestRegistry_Add_ReplacesExistingEntry(t *testing.T) {
	tt := mkTenant(t)
	r := NewTestRegistry()
	first := NewEntryBuilder().SetTenant(tt).SetCharacterId(42).SetToken(uuid.New()).Build()
	second := NewEntryBuilder().SetTenant(tt).SetCharacterId(42).SetToken(uuid.New()).Build()
	require.NoError(t, r.Add(first))
	require.NoError(t, r.Add(second))

	got, ok := r.Get(tt, 42)
	require.True(t, ok)
	require.Equal(t, second.Token(), got.Token(), "second Add overwrites prior entry")
}

func TestRegistry_TenantsIsolated(t *testing.T) {
	t1 := mkTenant(t)
	t2 := mkTenant(t)
	r := NewTestRegistry()
	require.NoError(t, r.Add(NewEntryBuilder().SetTenant(t1).SetCharacterId(42).SetToken(uuid.New()).Build()))
	_, ok := r.Get(t2, 42)
	require.False(t, ok, "Other tenant must not see entry")
}
```

- [ ] **Step 2: Run the tests**

```
go test ./map/timer/...
```
Expected: build failure (`undefined: NewTestRegistry`, `Add`, `Get`, `Cancel`).

- [ ] **Step 3: Implement `registry.go`**

Create `services/atlas-maps/atlas.com/maps/map/timer/registry.go`:

```go
package timer

import (
	"sync"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type tenantBucket struct {
	tenant  tenant.Model
	entries map[uint32]Entry
}

type Registry struct {
	mu        sync.RWMutex
	perTenant map[string]*tenantBucket
}

var (
	registryOnce sync.Once
	registry     *Registry
)

func GetRegistry() *Registry {
	registryOnce.Do(func() {
		registry = &Registry{perTenant: map[string]*tenantBucket{}}
	})
	return registry
}

func NewTestRegistry() *Registry {
	return &Registry{perTenant: map[string]*tenantBucket{}}
}

func tenantKey(t tenant.Model) string {
	return t.Id().String()
}

func (r *Registry) bucket(t tenant.Model) *tenantBucket {
	key := tenantKey(t)
	b, ok := r.perTenant[key]
	if !ok {
		b = &tenantBucket{tenant: t, entries: map[uint32]Entry{}}
		r.perTenant[key] = b
	}
	return b
}

// Add inserts or replaces the entry for (e.Tenant(), e.CharacterId()). Replacement
// is silent — callers that care about pre-existing entries should call Cancel
// first to obtain the prior entry's stop handle.
func (r *Registry) Add(e Entry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	b := r.bucket(e.Tenant())
	b.entries[e.CharacterId()] = e
	return nil
}

func (r *Registry) Get(t tenant.Model, characterId uint32) (Entry, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	b, ok := r.perTenant[tenantKey(t)]
	if !ok {
		return Entry{}, false
	}
	e, ok := b.entries[characterId]
	return e, ok
}

// Cancel atomically removes and returns the entry. The caller is responsible
// for stopping the entry's underlying time.Timer.
func (r *Registry) Cancel(t tenant.Model, characterId uint32) (Entry, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	b, ok := r.perTenant[tenantKey(t)]
	if !ok {
		return Entry{}, false
	}
	e, ok := b.entries[characterId]
	if !ok {
		return Entry{}, false
	}
	delete(b.entries, characterId)
	if len(b.entries) == 0 {
		delete(r.perTenant, tenantKey(t))
	}
	return e, true
}
```

- [ ] **Step 4: Run the tests**

```
go test ./map/timer/...
```
Expected: PASS.

- [ ] **Step 5: Commit**

```
git add services/atlas-maps/atlas.com/maps/map/timer/registry.go services/atlas-maps/atlas.com/maps/map/timer/registry_test.go
git commit -m "feat(atlas-maps): add timer Registry with Add/Get/Cancel"
```

---

## Task 9: Add `Claim(token)` and `ClaimAny` registry ops for race-safe expiration

**Files:**
- Modify: `services/atlas-maps/atlas.com/maps/map/timer/registry.go`
- Modify: `services/atlas-maps/atlas.com/maps/map/timer/registry_test.go`

- [ ] **Step 1: Write the failing claim tests**

Append to `registry_test.go`:

```go
func TestRegistry_Claim_MatchingTokenRemoves(t *testing.T) {
	tt := mkTenant(t)
	r := NewTestRegistry()
	tok := uuid.New()
	e := NewEntryBuilder().SetTenant(tt).SetCharacterId(42).SetToken(tok).Build()
	require.NoError(t, r.Add(e))

	got, claimed := r.Claim(tt, 42, tok)
	require.True(t, claimed)
	require.Equal(t, tok, got.Token())

	_, ok := r.Get(tt, 42)
	require.False(t, ok, "Claim must remove the entry")
}

func TestRegistry_Claim_StaleTokenIsNoOp(t *testing.T) {
	tt := mkTenant(t)
	r := NewTestRegistry()
	currentTok := uuid.New()
	staleTok := uuid.New()
	e := NewEntryBuilder().SetTenant(tt).SetCharacterId(42).SetToken(currentTok).Build()
	require.NoError(t, r.Add(e))

	_, claimed := r.Claim(tt, 42, staleTok)
	require.False(t, claimed, "Stale token must not claim")

	got, ok := r.Get(tt, 42)
	require.True(t, ok, "Entry must still be present")
	require.Equal(t, currentTok, got.Token())
}

func TestRegistry_Claim_AbsentIsNoOp(t *testing.T) {
	tt := mkTenant(t)
	r := NewTestRegistry()
	_, claimed := r.Claim(tt, 999, uuid.New())
	require.False(t, claimed)
}

func TestRegistry_ClaimAny_RemovesIgnoringToken(t *testing.T) {
	tt := mkTenant(t)
	r := NewTestRegistry()
	tok := uuid.New()
	e := NewEntryBuilder().SetTenant(tt).SetCharacterId(42).SetToken(tok).Build()
	require.NoError(t, r.Add(e))

	got, claimed := r.ClaimAny(tt, 42)
	require.True(t, claimed)
	require.Equal(t, tok, got.Token())

	_, ok := r.Get(tt, 42)
	require.False(t, ok)
}

func TestRegistry_ClaimAny_AbsentIsNoOp(t *testing.T) {
	tt := mkTenant(t)
	r := NewTestRegistry()
	_, claimed := r.ClaimAny(tt, 999)
	require.False(t, claimed)
}

func TestRegistry_ClaimRace_SecondClaimSeesEmpty(t *testing.T) {
	// Simulate Race B (timer goroutine vs SESSION_DESTROYED handler):
	// the loser sees an empty registry and bails.
	tt := mkTenant(t)
	r := NewTestRegistry()
	tok := uuid.New()
	require.NoError(t, r.Add(NewEntryBuilder().SetTenant(tt).SetCharacterId(42).SetToken(tok).Build()))

	_, claimedFirst := r.Claim(tt, 42, tok)
	require.True(t, claimedFirst)

	_, claimedSecond := r.ClaimAny(tt, 42)
	require.False(t, claimedSecond, "second claim sees no entry")
}
```

- [ ] **Step 2: Run the tests**

```
go test ./map/timer/...
```
Expected: build failure (`undefined: Claim`, `undefined: ClaimAny`).

- [ ] **Step 3: Implement Claim and ClaimAny in `registry.go`**

Append to `registry.go`:

```go
import "github.com/google/uuid" // already imported via Entry chain — confirm

// Claim atomically removes the entry only if its current token matches the
// supplied token. Used by the timer-fires goroutine to avoid acting on an
// entry that was replaced or cancelled while the timer was queued.
func (r *Registry) Claim(t tenant.Model, characterId uint32, token uuid.UUID) (Entry, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	b, ok := r.perTenant[tenantKey(t)]
	if !ok {
		return Entry{}, false
	}
	e, ok := b.entries[characterId]
	if !ok {
		return Entry{}, false
	}
	if e.Token() != token {
		return Entry{}, false
	}
	delete(b.entries, characterId)
	if len(b.entries) == 0 {
		delete(r.perTenant, tenantKey(t))
	}
	return e, true
}

// ClaimAny atomically removes any entry for the key regardless of token.
// Used by the SESSION_DESTROYED handler.
func (r *Registry) ClaimAny(t tenant.Model, characterId uint32) (Entry, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	b, ok := r.perTenant[tenantKey(t)]
	if !ok {
		return Entry{}, false
	}
	e, ok := b.entries[characterId]
	if !ok {
		return Entry{}, false
	}
	delete(b.entries, characterId)
	if len(b.entries) == 0 {
		delete(r.perTenant, tenantKey(t))
	}
	return e, true
}
```

If `uuid` is not yet imported in registry.go, add `"github.com/google/uuid"` to its import block.

- [ ] **Step 4: Run the tests**

```
go test ./map/timer/...
```
Expected: PASS.

- [ ] **Step 5: Commit**

```
git add services/atlas-maps/atlas.com/maps/map/timer/registry.go services/atlas-maps/atlas.com/maps/map/timer/registry_test.go
git commit -m "feat(atlas-maps): add Claim/ClaimAny race-safe registry ops"
```

---

## Task 10: Add timer producer functions (MAP_TIMER_STARTED + CHANGE_MAP)

**Files:**
- Create: `services/atlas-maps/atlas.com/maps/map/timer/producer.go`
- Create: `services/atlas-maps/atlas.com/maps/map/timer/producer_test.go`

- [ ] **Step 1: Write the failing producer tests**

Create `services/atlas-maps/atlas.com/maps/map/timer/producer_test.go`:

```go
package timer

import (
	"encoding/json"
	"testing"

	characterKafka "atlas-maps/kafka/message/character"
	mapKafka "atlas-maps/kafka/message/map"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMapTimerStartedProvider_BuildsCorrectEvent(t *testing.T) {
	txn := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	f := field.NewBuilder(world.Id(1), channel.Id(2), _map.Id(100000000)).SetInstance(uuid.Nil).Build()
	prov := mapTimerStartedProvider(txn, f, uint32(42), uint32(600))
	msgs, err := prov()
	require.NoError(t, err)
	require.Len(t, msgs, 1)

	var ev mapKafka.StatusEvent[mapKafka.MapTimerStarted]
	require.NoError(t, json.Unmarshal(msgs[0].Value, &ev))
	require.Equal(t, mapKafka.EventTopicMapStatusTypeMapTimerStarted, ev.Type)
	require.Equal(t, txn, ev.TransactionId)
	require.Equal(t, world.Id(1), ev.WorldId)
	require.Equal(t, channel.Id(2), ev.ChannelId)
	require.Equal(t, _map.Id(100000000), ev.MapId)
	require.Equal(t, uint32(42), ev.Body.CharacterId)
	require.Equal(t, uint32(600), ev.Body.Seconds)
}

func TestChangeMapProvider_BuildsCorrectCommand(t *testing.T) {
	txn := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	prov := changeMapProvider(txn, uint32(42), world.Id(1), channel.Id(2), _map.Id(100000201))
	msgs, err := prov()
	require.NoError(t, err)
	require.Len(t, msgs, 1)

	var cmd characterKafka.Command[characterKafka.ChangeMapBody]
	require.NoError(t, json.Unmarshal(msgs[0].Value, &cmd))
	require.Equal(t, characterKafka.CommandChangeMap, cmd.Type)
	require.Equal(t, txn, cmd.TransactionId)
	require.Equal(t, world.Id(1), cmd.WorldId)
	require.Equal(t, uint32(42), cmd.CharacterId)
	require.Equal(t, channel.Id(2), cmd.Body.ChannelId)
	require.Equal(t, _map.Id(100000201), cmd.Body.MapId)
	require.Equal(t, uuid.Nil, cmd.Body.Instance, "forced-return goes to non-instanced field")
	require.Equal(t, uint32(0), cmd.Body.PortalId, "default spawn portal")
}
```

- [ ] **Step 2: Run the tests**

```
go test ./map/timer/...
```
Expected: build failure (`undefined: mapTimerStartedProvider`, `undefined: changeMapProvider`).

- [ ] **Step 3: Implement `producer.go`**

Create `services/atlas-maps/atlas.com/maps/map/timer/producer.go`:

```go
package timer

import (
	characterKafka "atlas-maps/kafka/message/character"
	mapKafka "atlas-maps/kafka/message/map"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func mapTimerStartedProvider(transactionId uuid.UUID, f field.Model, characterId uint32, seconds uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &mapKafka.StatusEvent[mapKafka.MapTimerStarted]{
		TransactionId: transactionId,
		WorldId:       f.WorldId(),
		ChannelId:     f.ChannelId(),
		MapId:         f.MapId(),
		Instance:      f.Instance(),
		Type:          mapKafka.EventTopicMapStatusTypeMapTimerStarted,
		Body: mapKafka.MapTimerStarted{
			CharacterId: characterId,
			Seconds:     seconds,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func changeMapProvider(transactionId uuid.UUID, characterId uint32, worldId world.Id, channelId channel.Id, mapId _map.Id) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &characterKafka.Command[characterKafka.ChangeMapBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		CharacterId:   characterId,
		Type:          characterKafka.CommandChangeMap,
		Body: characterKafka.ChangeMapBody{
			ChannelId: channelId,
			MapId:     mapId,
			Instance:  uuid.Nil, // forced-return always targets non-instanced field
			PortalId:  0,        // default spawn portal
		},
	}
	return producer.SingleMessageProvider(key, value)
}
```

- [ ] **Step 4: Run the tests**

```
go test ./map/timer/...
```
Expected: PASS.

- [ ] **Step 5: Commit**

```
git add services/atlas-maps/atlas.com/maps/map/timer/producer.go services/atlas-maps/atlas.com/maps/map/timer/producer_test.go
git commit -m "feat(atlas-maps): add timer Kafka producers"
```

---

## Task 11: Timer Processor — `Register` (state-machine only, no goroutine)

**Files:**
- Create: `services/atlas-maps/atlas.com/maps/map/timer/processor.go`
- Create: `services/atlas-maps/atlas.com/maps/map/timer/processor_test.go`

- [ ] **Step 1: Write the failing Register test**

Create `services/atlas-maps/atlas.com/maps/map/timer/processor_test.go`:

```go
package timer

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	"atlas-maps/kafka/producer"
	mapKafka "atlas-maps/kafka/message/map"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	kafkaProducer "github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

// recordingProducer captures emitted messages by topic — copied from
// services/atlas-maps/atlas.com/maps/tasks/mist_tick_test.go.
type recordingProducer struct {
	mu       sync.Mutex
	messages map[string][]kafka.Message
}

func newRecordingProducer() *recordingProducer {
	return &recordingProducer{messages: map[string][]kafka.Message{}}
}

func (m *recordingProducer) Provider() producer.Provider {
	return func(token string) kafkaProducer.MessageProducer {
		return func(prov model.Provider[[]kafka.Message]) error {
			msgs, err := prov()
			if err != nil {
				return err
			}
			m.mu.Lock()
			defer m.mu.Unlock()
			m.messages[token] = append(m.messages[token], msgs...)
			return nil
		}
	}
}

func (m *recordingProducer) Messages(topic string) []kafka.Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]kafka.Message(nil), m.messages[topic]...)
}

func mkProcTenant(t *testing.T) tenant.Model {
	t.Helper()
	tt, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	return tt
}

func newTestProcessor(t *testing.T, reg *Registry, rec *recordingProducer, tt tenant.Model) Processor {
	t.Helper()
	logger, _ := test.NewNullLogger()
	tctx := tenant.WithContext(context.Background(), tt)
	return NewProcessorWithRegistry(logger, tctx, rec.Provider(), reg)
}

func TestProcessor_Register_InsertsEntryAndEmitsMapTimerStarted(t *testing.T) {
	tt := mkProcTenant(t)
	reg := NewTestRegistry()
	rec := newRecordingProducer()
	p := newTestProcessor(t, reg, rec, tt)

	f := field.NewBuilder(world.Id(1), channel.Id(2), _map.Id(100000000)).SetInstance(uuid.Nil).Build()
	require.NoError(t, p.Register(uuid.New(), uint32(42), f, _map.Id(100000201), uint32(600)))

	// Registry holds the entry.
	got, ok := reg.Get(tt, 42)
	require.True(t, ok)
	require.Equal(t, _map.Id(100000201), got.ForcedReturnMapId())
	require.Equal(t, uint32(600), got.Seconds())

	// Producer received MAP_TIMER_STARTED on EVENT_TOPIC_MAP_STATUS.
	msgs := rec.Messages(mapKafka.EnvEventTopicMapStatus)
	require.Len(t, msgs, 1)
	var ev mapKafka.StatusEvent[mapKafka.MapTimerStarted]
	require.NoError(t, json.Unmarshal(msgs[0].Value, &ev))
	require.Equal(t, mapKafka.EventTopicMapStatusTypeMapTimerStarted, ev.Type)
	require.Equal(t, uint32(42), ev.Body.CharacterId)
	require.Equal(t, uint32(600), ev.Body.Seconds)
}

func TestProcessor_Register_ReplacesPriorEntry(t *testing.T) {
	tt := mkProcTenant(t)
	reg := NewTestRegistry()
	rec := newRecordingProducer()
	p := newTestProcessor(t, reg, rec, tt)

	f1 := field.NewBuilder(0, 0, _map.Id(100000000)).SetInstance(uuid.Nil).Build()
	f2 := field.NewBuilder(0, 0, _map.Id(200000000)).SetInstance(uuid.Nil).Build()

	require.NoError(t, p.Register(uuid.New(), uint32(42), f1, _map.Id(100000201), uint32(600)))
	first, ok := reg.Get(tt, 42)
	require.True(t, ok)

	require.NoError(t, p.Register(uuid.New(), uint32(42), f2, _map.Id(200000201), uint32(300)))
	second, ok := reg.Get(tt, 42)
	require.True(t, ok)
	require.NotEqual(t, first.Token(), second.Token(), "second Register must mint a new token")
	require.Equal(t, _map.Id(200000201), second.ForcedReturnMapId(), "second Register replaces forcedReturnMapId")
}
```

- [ ] **Step 2: Run the tests**

```
go test ./map/timer/...
```
Expected: build failure (`undefined: Processor`, `undefined: NewProcessorWithRegistry`).

- [ ] **Step 3: Implement Processor + Register (no goroutine yet — schedule a no-op timer)**

Create `services/atlas-maps/atlas.com/maps/map/timer/processor.go`:

```go
package timer

import (
	"context"
	"time"

	"atlas-maps/kafka/message"
	mapKafka "atlas-maps/kafka/message/map"
	"atlas-maps/kafka/producer"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type Processor interface {
	Register(transactionId uuid.UUID, characterId uint32, f field.Model, forcedReturnMapId _map.Id, seconds uint32) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
	p   producer.Provider
	r   *Registry
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, p producer.Provider) Processor {
	return NewProcessorWithRegistry(l, ctx, p, GetRegistry())
}

func NewProcessorWithRegistry(l logrus.FieldLogger, ctx context.Context, p producer.Provider, r *Registry) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		t:   tenant.MustFromContext(ctx),
		p:   p,
		r:   r,
	}
}

// Register inserts (or replaces) the timer entry for (tenant, characterId),
// schedules a per-entry time.Timer, and publishes MAP_TIMER_STARTED so
// atlas-channel can render the countdown.
func (p *ProcessorImpl) Register(transactionId uuid.UUID, characterId uint32, f field.Model, forcedReturnMapId _map.Id, seconds uint32) error {
	_, span := otel.GetTracerProvider().Tracer("atlas-maps").Start(p.ctx, "MapTimer.Start")
	span.SetAttributes(
		attribute.String("tenant.id", p.t.Id().String()),
		attribute.Int("world.id", int(f.WorldId())),
		attribute.Int("map.id", int(f.MapId())),
		attribute.Int("forced.return.map.id", int(forcedReturnMapId)),
	)
	defer span.End()
	// Cancel any prior entry first so its timer is stopped before we replace it.
	if prior, ok := p.r.Cancel(p.t, characterId); ok {
		if prior.Timer() != nil {
			prior.Timer().Stop()
		}
	}

	tok := uuid.New()
	duration := time.Duration(seconds) * time.Second
	expiresAt := time.Now().Add(duration)
	t := time.AfterFunc(duration, func() {
		p.handleExpire(p.t, characterId, tok)
	})

	entry := NewEntryBuilder().
		SetTenant(p.t).
		SetCharacterId(characterId).
		SetField(f).
		SetForcedReturnMapId(forcedReturnMapId).
		SetSeconds(seconds).
		SetToken(tok).
		SetExpiresAt(expiresAt).
		SetTimer(t).
		Build()
	if err := p.r.Add(entry); err != nil {
		t.Stop()
		return err
	}

	if err := message.Emit(p.p)(func(buf *message.Buffer) error {
		return buf.Put(mapKafka.EnvEventTopicMapStatus, mapTimerStartedProvider(transactionId, f, characterId, seconds))
	}); err != nil {
		// Emission failure is logged but does NOT roll back the registry —
		// the timer is still authoritative for forced-return at expiry.
		p.l.WithError(err).Warnf("MapTimer.Register: failed to emit MAP_TIMER_STARTED for character [%d] map [%d].", characterId, f.MapId())
	}
	p.l.Infof("MapTimer.Start: tenant=[%s] character=[%d] map=[%d] forcedReturn=[%d] seconds=[%d].", p.t.Id(), characterId, f.MapId(), forcedReturnMapId, seconds)
	return nil
}

// handleExpire is the time.Timer callback. Stub for now — fleshed out in Task 13.
func (p *ProcessorImpl) handleExpire(t tenant.Model, characterId uint32, token uuid.UUID) {
	_, _ = p.r.Claim(t, characterId, token)
}
```

- [ ] **Step 4: Run the tests**

```
go test ./map/timer/...
```
Expected: PASS.

- [ ] **Step 5: Commit**

```
git add services/atlas-maps/atlas.com/maps/map/timer/processor.go services/atlas-maps/atlas.com/maps/map/timer/processor_test.go
git commit -m "feat(atlas-maps): add timer Processor.Register with MAP_TIMER_STARTED emission"
```

---

## Task 12: Timer Processor — `CancelIfTracked`

**Files:**
- Modify: `services/atlas-maps/atlas.com/maps/map/timer/processor.go`
- Modify: `services/atlas-maps/atlas.com/maps/map/timer/processor_test.go`

- [ ] **Step 1: Write the failing CancelIfTracked test**

Append to `processor_test.go`:

```go
func TestProcessor_CancelIfTracked_RemovesAndStops(t *testing.T) {
	tt := mkProcTenant(t)
	reg := NewTestRegistry()
	rec := newRecordingProducer()
	p := newTestProcessor(t, reg, rec, tt)

	f := field.NewBuilder(0, 0, _map.Id(100000000)).SetInstance(uuid.Nil).Build()
	require.NoError(t, p.Register(uuid.New(), uint32(42), f, _map.Id(100000201), uint32(600)))

	cancelled := p.CancelIfTracked(uint32(42))
	require.True(t, cancelled)

	_, ok := reg.Get(tt, 42)
	require.False(t, ok, "CancelIfTracked must remove entry")
}

func TestProcessor_CancelIfTracked_AbsentReturnsFalse(t *testing.T) {
	tt := mkProcTenant(t)
	reg := NewTestRegistry()
	rec := newRecordingProducer()
	p := newTestProcessor(t, reg, rec, tt)
	require.False(t, p.CancelIfTracked(uint32(999)))
}
```

- [ ] **Step 2: Run the tests**

```
go test ./map/timer/...
```
Expected: build failure (`p.CancelIfTracked` undefined on Processor).

- [ ] **Step 3: Add `CancelIfTracked` to the Processor interface and impl**

In `processor.go`, expand the interface:

```go
type Processor interface {
	Register(transactionId uuid.UUID, characterId uint32, f field.Model, forcedReturnMapId _map.Id, seconds uint32) error
	CancelIfTracked(characterId uint32) bool
}
```

Append the impl:

```go
func (p *ProcessorImpl) CancelIfTracked(characterId uint32) bool {
	prior, ok := p.r.Cancel(p.t, characterId)
	if !ok {
		return false
	}
	_, span := otel.GetTracerProvider().Tracer("atlas-maps").Start(p.ctx, "MapTimer.Cancel")
	span.SetAttributes(
		attribute.String("tenant.id", p.t.Id().String()),
		attribute.Int("world.id", int(prior.Field().WorldId())),
		attribute.Int("map.id", int(prior.Field().MapId())),
	)
	defer span.End()
	if prior.Timer() != nil {
		prior.Timer().Stop()
	}
	p.l.Infof("MapTimer.Cancel: tenant=[%s] character=[%d] map=[%d].", p.t.Id(), characterId, prior.Field().MapId())
	return true
}
```

- [ ] **Step 4: Run the tests**

```
go test ./map/timer/...
```
Expected: PASS.

- [ ] **Step 5: Commit**

```
git add services/atlas-maps/atlas.com/maps/map/timer/processor.go services/atlas-maps/atlas.com/maps/map/timer/processor_test.go
git commit -m "feat(atlas-maps): add timer Processor.CancelIfTracked"
```

---

## Task 13: Timer Processor — `ForceReturnIfTracked` and `handleExpire` emit CHANGE_MAP

**Files:**
- Modify: `services/atlas-maps/atlas.com/maps/map/timer/processor.go`
- Modify: `services/atlas-maps/atlas.com/maps/map/timer/processor_test.go`

- [ ] **Step 1: Write failing tests for ForceReturnIfTracked + expiration**

Append to `processor_test.go`:

```go
func TestProcessor_ForceReturnIfTracked_EmitsChangeMap(t *testing.T) {
	tt := mkProcTenant(t)
	reg := NewTestRegistry()
	rec := newRecordingProducer()
	p := newTestProcessor(t, reg, rec, tt)

	f := field.NewBuilder(world.Id(1), channel.Id(2), _map.Id(100000000)).SetInstance(uuid.Nil).Build()
	require.NoError(t, p.Register(uuid.New(), uint32(42), f, _map.Id(100000201), uint32(600)))

	forced := p.ForceReturnIfTracked(uint32(42))
	require.True(t, forced)

	// Registry empty.
	_, ok := reg.Get(tt, 42)
	require.False(t, ok)

	// Producer received CHANGE_MAP on COMMAND_TOPIC_CHARACTER.
	msgs := rec.Messages(characterKafka.EnvCommandTopic)
	require.Len(t, msgs, 1)
	var cmd characterKafka.Command[characterKafka.ChangeMapBody]
	require.NoError(t, json.Unmarshal(msgs[0].Value, &cmd))
	require.Equal(t, characterKafka.CommandChangeMap, cmd.Type)
	require.Equal(t, world.Id(1), cmd.WorldId)
	require.Equal(t, channel.Id(2), cmd.Body.ChannelId)
	require.Equal(t, _map.Id(100000201), cmd.Body.MapId)
	require.Equal(t, uuid.Nil, cmd.Body.Instance)
	require.Equal(t, uint32(0), cmd.Body.PortalId)
}

func TestProcessor_ForceReturnIfTracked_AbsentIsNoOp(t *testing.T) {
	tt := mkProcTenant(t)
	reg := NewTestRegistry()
	rec := newRecordingProducer()
	p := newTestProcessor(t, reg, rec, tt)
	require.False(t, p.ForceReturnIfTracked(uint32(999)))
	require.Empty(t, rec.Messages(characterKafka.EnvCommandTopic))
}

func TestProcessor_TimerFires_EmitsChangeMap(t *testing.T) {
	tt := mkProcTenant(t)
	reg := NewTestRegistry()
	rec := newRecordingProducer()
	p := newTestProcessor(t, reg, rec, tt)

	f := field.NewBuilder(world.Id(1), channel.Id(2), _map.Id(100000000)).SetInstance(uuid.Nil).Build()
	// 100ms duration so we can observe expiration with a short sleep.
	require.NoError(t, p.Register(uuid.New(), uint32(42), f, _map.Id(100000201), uint32(0)))

	// seconds=0 ⇒ time.AfterFunc(0) fires "immediately"; sleep enough to let it run.
	time.Sleep(150 * time.Millisecond)

	_, ok := reg.Get(tt, 42)
	require.False(t, ok, "expired entry must be removed by handleExpire")

	msgs := rec.Messages(characterKafka.EnvCommandTopic)
	require.Len(t, msgs, 1)
	var cmd characterKafka.Command[characterKafka.ChangeMapBody]
	require.NoError(t, json.Unmarshal(msgs[0].Value, &cmd))
	require.Equal(t, _map.Id(100000201), cmd.Body.MapId)
}

func TestProcessor_TimerFires_StaleTokenNoOp(t *testing.T) {
	tt := mkProcTenant(t)
	reg := NewTestRegistry()
	rec := newRecordingProducer()
	p := newTestProcessor(t, reg, rec, tt)

	f := field.NewBuilder(0, 0, _map.Id(100000000)).SetInstance(uuid.Nil).Build()
	// First Register schedules a timer with token A.
	require.NoError(t, p.Register(uuid.New(), uint32(42), f, _map.Id(100000201), uint32(0)))
	// Immediately Register again to mint token B and replace token A's entry —
	// this also Stops the prior *time.Timer; if Stop loses to AfterFunc the
	// stale callback still calls Claim with token A, which must no-op.
	f2 := field.NewBuilder(0, 0, _map.Id(200000000)).SetInstance(uuid.Nil).Build()
	require.NoError(t, p.Register(uuid.New(), uint32(42), f2, _map.Id(200000201), uint32(60)))

	time.Sleep(150 * time.Millisecond)

	got, ok := reg.Get(tt, 42)
	require.True(t, ok, "second entry must still be present")
	require.Equal(t, _map.Id(200000201), got.ForcedReturnMapId(), "second entry survived")
	// CHANGE_MAP must NOT have been emitted on behalf of the stale first entry.
	require.Empty(t, rec.Messages(characterKafka.EnvCommandTopic), "stale token must not emit CHANGE_MAP")
}
```

Add the import for `characterKafka "atlas-maps/kafka/message/character"` and `time` to the test file's import block if missing.

- [ ] **Step 2: Run the tests**

```
go test ./map/timer/...
```
Expected: build failure (`p.ForceReturnIfTracked` undefined; expiration test fails).

- [ ] **Step 3: Wire CHANGE_MAP into ForceReturnIfTracked + handleExpire**

In `processor.go`:

(a) Expand the interface:

```go
type Processor interface {
	Register(transactionId uuid.UUID, characterId uint32, f field.Model, forcedReturnMapId _map.Id, seconds uint32) error
	CancelIfTracked(characterId uint32) bool
	ForceReturnIfTracked(characterId uint32) bool
}
```

(b) Add ForceReturnIfTracked:

```go
func (p *ProcessorImpl) ForceReturnIfTracked(characterId uint32) bool {
	entry, ok := p.r.ClaimAny(p.t, characterId)
	if !ok {
		return false
	}
	_, span := otel.GetTracerProvider().Tracer("atlas-maps").Start(p.ctx, "MapTimer.Disconnect")
	span.SetAttributes(
		attribute.String("tenant.id", p.t.Id().String()),
		attribute.Int("world.id", int(entry.Field().WorldId())),
		attribute.Int("map.id", int(entry.Field().MapId())),
		attribute.Int("forced.return.map.id", int(entry.ForcedReturnMapId())),
	)
	defer span.End()
	if entry.Timer() != nil {
		entry.Timer().Stop()
	}
	if err := p.emitChangeMap(p.ctx, entry); err != nil {
		p.l.WithError(err).Errorf("MapTimer.Disconnect: failed to emit CHANGE_MAP for character [%d].", characterId)
	}
	p.l.Warnf("MapTimer.Disconnect: tenant=[%s] character=[%d] map=[%d] forcedReturn=[%d].", p.t.Id(), characterId, entry.Field().MapId(), entry.ForcedReturnMapId())
	return true
}
```

(c) Replace `handleExpire` with the real implementation:

```go
func (p *ProcessorImpl) handleExpire(tt tenant.Model, characterId uint32, token uuid.UUID) {
	entry, claimed := p.r.Claim(tt, characterId, token)
	if !claimed {
		return // stale token / already cancelled — race A or replaced.
	}
	// Detached goroutine ctx: do NOT inherit the consumer ctx (may be cancelled).
	sctx, span := otel.GetTracerProvider().Tracer("atlas-maps").Start(context.Background(), "MapTimer.Expire")
	span.SetAttributes(
		attribute.String("tenant.id", tt.Id().String()),
		attribute.Int("world.id", int(entry.Field().WorldId())),
		attribute.Int("map.id", int(entry.Field().MapId())),
		attribute.Int("forced.return.map.id", int(entry.ForcedReturnMapId())),
	)
	defer span.End()
	tctx := tenant.WithContext(sctx, tt)
	if err := p.emitChangeMap(tctx, entry); err != nil {
		p.l.WithError(err).Errorf("MapTimer.Expire: failed to emit CHANGE_MAP for character [%d].", characterId)
		return
	}
	p.l.Warnf("MapTimer.Expire: tenant=[%s] character=[%d] map=[%d] forcedReturn=[%d].", tt.Id(), characterId, entry.Field().MapId(), entry.ForcedReturnMapId())
}
```

(d) Add the helper:

```go
func (p *ProcessorImpl) emitChangeMap(ctx context.Context, entry Entry) error {
	// Build a tenant-scoped producer so the message inherits the correct
	// Kafka headers even when called from a detached goroutine.
	prov := producer.ProviderImpl(p.l)(ctx)
	return message.Emit(prov)(func(buf *message.Buffer) error {
		return buf.Put(characterKafka.EnvCommandTopic, changeMapProvider(uuid.New(), entry.CharacterId(), entry.Field().WorldId(), entry.Field().ChannelId(), entry.ForcedReturnMapId()))
	})
}
```

(e) Update the import block in `processor.go` to include:

```go
characterKafka "atlas-maps/kafka/message/character"
"go.opentelemetry.io/otel"
"go.opentelemetry.io/otel/attribute"
```

(f) For Tasks 11 and 12 (Register and CancelIfTracked), the impl was instrumented
inline with spans (`MapTimer.Start` and `MapTimer.Cancel`). Make sure the import
additions above were added when those tasks landed — if not, add now.

- [ ] **Step 4: Run the tests**

```
go test ./map/timer/... -count=1 -timeout 30s
```
Expected: PASS. (The `_count=1` disables the test cache so the 100ms timing tests aren't skipped on a re-run.)

- [ ] **Step 5: Commit**

```
git add services/atlas-maps/atlas.com/maps/map/timer/processor.go services/atlas-maps/atlas.com/maps/map/timer/processor_test.go
git commit -m "feat(atlas-maps): emit CHANGE_MAP on timer expiry and forced-return"
```

---

## Task 14: SESSION_DESTROYED consumer in atlas-maps

**Files:**
- Create: `services/atlas-maps/atlas.com/maps/kafka/consumer/session/consumer.go`
- Create: `services/atlas-maps/atlas.com/maps/kafka/consumer/session/consumer_test.go`

- [ ] **Step 1: Write the failing handler test**

Create `services/atlas-maps/atlas.com/maps/kafka/consumer/session/consumer_test.go`:

```go
package session

import (
	"context"
	"testing"

	sessionKafka "atlas-maps/kafka/message/session"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

type fakeForceReturner struct {
	calledFor uint32
	called    bool
	returnVal bool
}

func (f *fakeForceReturner) ForceReturnIfTracked(characterId uint32) bool {
	f.calledFor = characterId
	f.called = true
	return f.returnVal
}

func TestHandleSessionDestroyed_ForcesReturnForTrackedCharacter(t *testing.T) {
	logger, _ := test.NewNullLogger()
	tt, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), tt)

	fr := &fakeForceReturner{returnVal: true}
	h := newHandleSessionDestroyed(func(_ context.Context) ForceReturner { return fr })
	h(logger, ctx, sessionKafka.StatusEvent{
		SessionId:   uuid.New(),
		AccountId:   1,
		CharacterId: 42,
		WorldId:     world.Id(1),
		ChannelId:   channel.Id(2),
		Type:        sessionKafka.EventSessionStatusTypeDestroyed,
	})

	require.True(t, fr.called)
	require.Equal(t, uint32(42), fr.calledFor)
}

func TestHandleSessionDestroyed_IgnoresCreatedEvents(t *testing.T) {
	logger, _ := test.NewNullLogger()
	tt, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), tt)

	fr := &fakeForceReturner{}
	h := newHandleSessionDestroyed(func(_ context.Context) ForceReturner { return fr })
	h(logger, ctx, sessionKafka.StatusEvent{
		Type:        sessionKafka.EventSessionStatusTypeCreated,
		CharacterId: 42,
	})
	require.False(t, fr.called)
}

func TestHandleSessionDestroyed_SkipsZeroCharacterId(t *testing.T) {
	logger, _ := test.NewNullLogger()
	tt, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), tt)

	fr := &fakeForceReturner{}
	h := newHandleSessionDestroyed(func(_ context.Context) ForceReturner { return fr })
	h(logger, ctx, sessionKafka.StatusEvent{
		Type:        sessionKafka.EventSessionStatusTypeDestroyed,
		CharacterId: 0,
	})
	require.False(t, fr.called, "no-character-selected sessions must be skipped")
}
```

- [ ] **Step 2: Run the test**

```
go test ./kafka/consumer/session/...
```
Expected: build failure (`undefined: newHandleSessionDestroyed`, `undefined: ForceReturner`).

- [ ] **Step 3: Implement the consumer**

Create `services/atlas-maps/atlas.com/maps/kafka/consumer/session/consumer.go`:

```go
package session

import (
	"context"

	consumer2 "atlas-maps/kafka/consumer"
	sessionKafka "atlas-maps/kafka/message/session"
	"atlas-maps/kafka/producer"
	timer "atlas-maps/map/timer"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	kafkaMessage "github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/sirupsen/logrus"
)

// ForceReturner is the seam used to test the handler without standing up a
// real timer Processor. Production binds it to timer.Processor's
// ForceReturnIfTracked.
type ForceReturner interface {
	ForceReturnIfTracked(characterId uint32) bool
}

type forceReturnerProvider func(ctx context.Context) ForceReturner

func defaultForceReturnerProvider(l logrus.FieldLogger) forceReturnerProvider {
	return func(ctx context.Context) ForceReturner {
		return timer.NewProcessor(l, ctx, producer.ProviderImpl(l)(ctx))
	}
}

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("session_status")(sessionKafka.EnvEventTopicSessionStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		t, _ = topic.EnvProvider(l)(sessionKafka.EnvEventTopicSessionStatus)()
		if _, err := rf(t, kafkaMessage.AdaptHandler(kafkaMessage.PersistentConfig(newHandleSessionDestroyed(defaultForceReturnerProvider(l))))); err != nil {
			return err
		}
		return nil
	}
}

func newHandleSessionDestroyed(provider forceReturnerProvider) kafkaMessage.Handler[sessionKafka.StatusEvent] {
	return func(l logrus.FieldLogger, ctx context.Context, e sessionKafka.StatusEvent) {
		if e.Type != sessionKafka.EventSessionStatusTypeDestroyed {
			return
		}
		if e.CharacterId == 0 {
			// Pre-character-selection sessions never had a timer; skip.
			return
		}
		l.Debugf("SESSION_DESTROYED for character [%d] account [%d] world [%d] channel [%d].", e.CharacterId, e.AccountId, e.WorldId, e.ChannelId)
		provider(ctx).ForceReturnIfTracked(e.CharacterId)
	}
}
```

- [ ] **Step 4: Run the tests**

```
go test ./kafka/consumer/session/...
```
Expected: PASS.

- [ ] **Step 5: Commit**

```
git add services/atlas-maps/atlas.com/maps/kafka/consumer/session/consumer.go services/atlas-maps/atlas.com/maps/kafka/consumer/session/consumer_test.go
git commit -m "feat(atlas-maps): add SESSION_DESTROYED consumer to fire forced-return"
```

---

## Task 15: Wire timer hooks into MAP_CHANGED + CHANNEL_CHANGED handlers

**Files:**
- Modify: `services/atlas-maps/atlas.com/maps/kafka/consumer/character/consumer.go`

No new test file — the hook calls are thin glue, the registry+processor logic is already covered. Future end-to-end coverage comes from running atlas-maps' full test suite + manual smoke.

- [ ] **Step 1: Add the timer hook calls into `handleStatusEventMapChangedFunc`**

In `services/atlas-maps/atlas.com/maps/kafka/consumer/character/consumer.go`, modify the `handleStatusEventMapChangedFunc` body. The current handler (lines 77-88) just calls `TransitionMapAndEmit`. After the existing `TransitionMapAndEmit` line, add:

```go
				// --- map-time-limit timer hooks (task-050) ---
				tp := timer.NewProcessor(l, ctx, producer.ProviderImpl(l)(ctx))
				_ = tp.CancelIfTracked(event.CharacterId)
				md, err := info.NewProcessor(l, ctx).GetById(event.Body.TargetMapId)
				if err != nil {
					l.WithError(err).Debugf("Unable to fetch map info for [%d]; skipping time-limit registration.", event.Body.TargetMapId)
				} else if md.IsTimeLimited() {
					if rerr := tp.Register(transactionId, event.CharacterId, newField, md.ForcedReturnMapId(), uint32(md.TimeLimit())); rerr != nil {
						l.WithError(rerr).Warnf("Failed to register map-time-limit timer for character [%d] map [%d].", event.CharacterId, event.Body.TargetMapId)
					}
				}
				// --- end task-050 hooks ---
```

Note: `transactionId` and `newField` are already in scope at this point.

- [ ] **Step 2: Add the CHANNEL_CHANGED fallback**

In the same file, modify `handleStatusEventChannelChangedFunc` (lines 90-100). After the existing `TransitionChannelAndEmit` line, add:

```go
				// task-050 fallback: SESSION_DESTROYED normally clears the
				// registry first; if it didn't (race window), force-return now.
				tp := timer.NewProcessor(l, ctx, producer.ProviderImpl(l)(ctx))
				_ = tp.ForceReturnIfTracked(event.CharacterId)
```

- [ ] **Step 3: Update imports**

Add to the `import` block in `consumer.go`:

```go
"atlas-maps/data/map/info"
"atlas-maps/map/timer"
```

(`producer` is already imported.)

- [ ] **Step 4: Verify build + existing tests still pass**

```
go build ./...
go test ./...
```
Expected: success. Use `-timeout 60s` if any timer-driven test runs slow.

- [ ] **Step 5: Commit**

```
git add services/atlas-maps/atlas.com/maps/kafka/consumer/character/consumer.go
git commit -m "feat(atlas-maps): wire map-time-limit timer into MAP_CHANGED + CHANNEL_CHANGED handlers"
```

---

## Task 16: Wire SESSION_DESTROYED consumer into atlas-maps `main.go`

**Files:**
- Modify: `services/atlas-maps/atlas.com/maps/main.go`

- [ ] **Step 1: Add session consumer + handler init**

In `main.go`, find the existing consumer init block (around line 68-88). Add:

(a) New import:

```go
sessionConsumer "atlas-maps/kafka/consumer/session"
```

(b) After `mistConsumer.InitConsumers(l)(cmf)(consumerGroupId)`, add:

```go
	sessionConsumer.InitConsumers(l)(cmf)(consumerGroupId)
```

(c) After `mistConsumer.InitHandlers(...)` block, add:

```go
	if err := sessionConsumer.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register session-status kafka handlers.")
	}
```

- [ ] **Step 2: Verify build**

```
go build ./...
```
Expected: success.

- [ ] **Step 3: Commit**

```
git add services/atlas-maps/atlas.com/maps/main.go
git commit -m "feat(atlas-maps): wire SESSION_DESTROYED consumer in main"
```

---

## Task 17: Add `MAP_TIMER_STARTED` envelope to atlas-channel

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/message/map/kafka.go`
- Create: `services/atlas-channel/atlas.com/channel/kafka/message/map/kafka_test.go`

- [ ] **Step 1: Write the failing serialization test**

Create `services/atlas-channel/atlas.com/channel/kafka/message/map/kafka_test.go`:

```go
package _map

import (
	"encoding/json"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

func TestStatusEvent_MapTimerStarted_Serialization(t *testing.T) {
	event := StatusEvent[MapTimerStarted]{
		TransactionId: uuid.MustParse("12345678-1234-5678-1234-567812345678"),
		WorldId:       world.Id(1),
		ChannelId:     channel.Id(2),
		MapId:         _map.Id(100000000),
		Type:          EventTopicMapStatusTypeMapTimerStarted,
		Body: MapTimerStarted{
			CharacterId: 12345,
			Seconds:     600,
		},
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded StatusEvent[MapTimerStarted]
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}
	if decoded.Type != EventTopicMapStatusTypeMapTimerStarted {
		t.Errorf("Type mismatch")
	}
	if decoded.Body.CharacterId != 12345 || decoded.Body.Seconds != 600 {
		t.Errorf("Body mismatch")
	}
}

func TestEventTypeConstant_MapTimerStarted(t *testing.T) {
	if EventTopicMapStatusTypeMapTimerStarted != "MAP_TIMER_STARTED" {
		t.Errorf("Expected 'MAP_TIMER_STARTED', got '%s'", EventTopicMapStatusTypeMapTimerStarted)
	}
}
```

- [ ] **Step 2: Run the test**

```
go test ./kafka/message/map/...
```
Expected: build failure.

- [ ] **Step 3: Add the constant + body to atlas-channel's kafka.go**

In `services/atlas-channel/atlas.com/channel/kafka/message/map/kafka.go`, append `EventTopicMapStatusTypeMapTimerStarted = "MAP_TIMER_STARTED"` to the const block, and add at the bottom:

```go
type MapTimerStarted struct {
	CharacterId uint32 `json:"characterId"`
	Seconds     uint32 `json:"seconds"`
}
```

- [ ] **Step 4: Run the test**

```
go test ./kafka/message/map/...
```
Expected: PASS.

- [ ] **Step 5: Commit**

```
git add services/atlas-channel/atlas.com/channel/kafka/message/map/kafka.go services/atlas-channel/atlas.com/channel/kafka/message/map/kafka_test.go
git commit -m "feat(atlas-channel): add MAP_TIMER_STARTED event envelope"
```

---

## Task 18: atlas-channel — render TimerClock packet on `MAP_TIMER_STARTED`

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go`

No new test — the writer (`fieldcb.NewTimerClock`) is already covered, and the handler is a thin map between event → existing infrastructure. Smoke covered via cross-service integration testing.

- [ ] **Step 1: Add the handler**

In `services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go`, after `handleStatusEventCharacterEnter` (around line 84-98), add:

```go
func handleStatusEventMapTimerStarted(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, ctx context.Context, event _map3.StatusEvent[_map3.MapTimerStarted]) {
	return func(l logrus.FieldLogger, ctx context.Context, e _map3.StatusEvent[_map3.MapTimerStarted]) {
		if e.Type != _map3.EventTopicMapStatusTypeMapTimerStarted {
			return
		}
		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}
		l.Debugf("MAP_TIMER_STARTED for character [%d] map [%d] seconds [%d].", e.Body.CharacterId, e.MapId, e.Body.Seconds)
		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.Body.CharacterId, session.Announce(l)(ctx)(wp)(fieldcb.ClockWriter)(fieldcb.NewTimerClock(e.Body.Seconds).Encode))
	}
}
```

- [ ] **Step 2: Register the handler in `InitHandlers`**

In the same file's `InitHandlers` (around line 60-82), add a new line before `return nil`:

```go
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventMapTimerStarted(sc, wp)))); err != nil {
					return err
				}
```

- [ ] **Step 3: Verify build**

```
go build ./...
```
Expected: success. (No new imports needed — `fieldcb`, `session`, `tenant`, `message`, `_map3` are already imported.)

- [ ] **Step 4: Run all map-consumer tests**

```
go test ./kafka/consumer/map/...
```
Expected: PASS.

- [ ] **Step 5: Commit**

```
git add services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go
git commit -m "feat(atlas-channel): render TimerClock on MAP_TIMER_STARTED event"
```

---

## Task 19: Whole-service build + test pass for atlas-maps and atlas-channel

**Files:** none (verification only)

- [ ] **Step 1: Build atlas-maps**

Run from `services/atlas-maps/atlas.com/maps/`:
```
go build ./...
```
Expected: success.

- [ ] **Step 2: Test atlas-maps**

```
go test ./... -count=1 -timeout 90s
```
Expected: all pass. The 100ms-timer expiration tests in `map/timer/processor_test.go` may take ~150ms each.

- [ ] **Step 3: Build atlas-channel**

Run from `services/atlas-channel/atlas.com/channel/`:
```
go build ./...
```
Expected: success.

- [ ] **Step 4: Test atlas-channel**

```
go test ./... -count=1 -timeout 120s
```
Expected: all pass.

- [ ] **Step 5: If anything fails, fix root cause and re-test, then commit**

Fix issues with surgical commits (one logical fix per commit). If everything is green, no commit needed for this task.

---

## Task 20: Service docs refresh

**Files:**
- Modify (or create): `services/atlas-maps/docs/...`
- Modify (or create): `services/atlas-channel/docs/...`

- [ ] **Step 1: Refresh atlas-maps docs**

Run:
```
/service-doc atlas-maps
```
Wait for the agent to finish.

- [ ] **Step 2: Refresh atlas-channel docs**

```
/service-doc atlas-channel
```
Wait for the agent to finish.

- [ ] **Step 3: Review and commit doc changes**

Review the diff. If the agent made changes:
```
git add services/atlas-maps/docs services/atlas-channel/docs
git commit -m "docs: refresh atlas-maps and atlas-channel docs for map time limits"
```

If no changes: skip this commit.

---

## Self-review notes

**Spec coverage:**

| PRD acceptance criteria (§10) | Task |
|---|---|
| Map data exposes ForcedReturnMapId/TimeLimit/IsTimeLimited | Task 1, Task 2 |
| Timer registered on entry to time-limited map | Task 11 + Task 15 |
| No timer for non-time-limited maps | Task 1 (predicate) + Task 15 (gate) |
| Re-entering same map resets timer | Task 11 (Register replaces prior) |
| Portal-out cancels timer, no warp | Task 12 + Task 15 (CancelIfTracked) |
| Direct portal to other time-limited map | Task 11 + Task 15 (Cancel + Register) |
| Expiration → CHANGE_MAP → warp | Task 13 (handleExpire) |
| Logout → forced-return persisted | Task 13 + Task 14 + Task 16 (SESSION_DESTROYED → ForceReturn) |
| Channel change → arrival at forced-return | Task 14 (primary) + Task 15 (CHANNEL_CHANGED fallback) |
| Death uses returnMapId, not forcedReturnMapId | unchanged (no task — verified by NOT touching `respawn/processor.go`) |
| Graceful shutdown cancels timers, zero CHANGE_MAP | partially — `time.Timer.Stop` on Cancel; on full process shutdown Go runtime cancels pending `time.AfterFunc`. The handleExpire goroutine's `Claim` will run if AlreadyFired, but at process exit message.Emit will be a no-op or fail (caller logs and returns). Acceptable per design §4.5. |
| Cross-tenant character isolation | Task 8 (registry tenantKey) |
| Client-side countdown rendered | Task 17 + Task 18 |
| Logs/metrics fire | Task 11/12/13 (logs); spans on `MapTimer.Expire` (Task 13) — Start/Cancel spans deferred (see follow-ups below). |

**Placeholder scan:** none.

**Type consistency check:**

- `Processor.Register(transactionId, characterId, field, forcedReturnMapId, seconds)` signature stays consistent across Tasks 11–15.
- `ForceReturnIfTracked(characterId) bool` signature stays consistent across Tasks 13, 14, 15.
- `MapTimerStarted{CharacterId uint32; Seconds uint32}` body shape is identical in atlas-maps (Task 4) and atlas-channel (Task 17) — required for cross-service deserialization.
- `ChangeMapBody{ChannelId, MapId, Instance, PortalId}` in atlas-maps (Task 5) matches atlas-character's existing body field-for-field.
- `Entry.Token() uuid.UUID` is the value passed to `Claim(t, characterId, token)` in Task 9 and consumed by `handleExpire` in Task 13.

**Known follow-ups (out of v1 scope):**

- Crash-recovery (atlas-maps pod crash mid-timer) is accepted-loss per PRD §8.6. atlas-maps' restart loses in-flight timers; characters still on the time-limited map start a fresh timer on next MAP_CHANGED.
- `character.id` span attribute (omitted intentionally per task-040 §3 cardinality exclusion). It's available in trace bodies via the existing log lines; not promoted to a spanmetrics dimension.

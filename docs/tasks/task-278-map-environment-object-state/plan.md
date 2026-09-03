# Map Environment Object State & Field Obstacles — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Drive the four already-verified field-object packets (`SetObjectState`, `FieldObstacleOnOff`, `FieldObstacleOnOffList`, `FieldObstacleAllReset`) from reactor/map script operations through a saga action pair, with per-field-instance state owned by `atlas-maps` and replayed to late joiners.

**Architecture:** Script op → saga step (`move_environment` / `reset_environment`) → `atlas-saga-orchestrator` handler → `COMMAND_TOPIC_MAP` → `atlas-maps` in-memory `map/environment` registry → `EVENT_TOPIC_MAP_STATUS` → `atlas-channel` broadcast. Late joiners are served by a REST `GET` from `atlas-channel`'s `SpawnForSelf`. This is the existing weather/jukebox shape verbatim.

**Tech Stack:** Go 1.27, `libs/atlas-kafka` (segmentio/kafka-go), `libs/atlas-rest` (JSON:API via jtumidanski/api2go), `libs/atlas-model` provider/operator composition, `libs/atlas-tenant` context tenancy, gorilla/mux, sirupsen/logrus, testify.

**Spec:** `docs/tasks/task-278-map-environment-object-state/design.md` (PRD at `docs/tasks/task-278-map-environment-object-state/prd.md`)

## Global Constraints

- **`libs/atlas-packet` is unchanged by this branch.** All four codecs exist and are verified. No new codec, no new version gate, no `MajorAtLeast` idiom anywhere in this task.
- **Seed templates under `services/atlas-configurations/seed-data/templates/` are unchanged.** No new opcode is routed.
- **No version sniffing.** Writer selection falls back on a real `errors.New("writer not found")` from `writer.Producer` (`libs/atlas-socket/writer/writer.go:33`), never on a tenant version comparison. This replaces PRD FR-15 and the PRD acceptance criterion that mentions "GMS<61 per-obstacle `FieldObstacleOnOff`" — `gms_48_1` routes **no** `FieldObstacleOnOff` at all, so the fallback target is `SetObjectState`, not `FieldObstacleOnOff`. See design.md §1.3.
- **`SetObjectState` is routed on all nine templates**; `CField::OnSetObjectState` and `CField::OnFieldObstacleOnOff` are byte-for-byte the same client function (design.md §1.1), so the fallback is behaviourally identical.
- **Reset semantics:** `FieldObstacleAllReset` restores only the client's `m_lpObstacle` list, not arbitrary named objects (design.md §1.2). The `ENVIRONMENT_RESET` event therefore carries the cleared entries and the channel additionally sends `SetObjectState(name, 0)` for every one of them.
- **Tenancy:** every registry read/write goes through `tenant.MustFromContext(p.ctx)`; `Tenant` is part of the registry key.
- **No stubs.** No `// TODO`, no bare `return nil` placeholder, no unimplemented status response may be landed.
- Use the project's Builder pattern in tests. No `*_testhelpers.go`.
- Preserve existing line endings. Repo-relative paths only in committed files.
- Flagless `tools/verify.sh` must exit 0 before the branch is done.

**Naming contract used across tasks (defined in Task 1 / Task 2, consumed everywhere later):**

```go
// libs/atlas-constants/field
type ObjectKind string
const ObjectKindEnvironment ObjectKind = "ENVIRONMENT"
const ObjectKindObstacle    ObjectKind = "OBSTACLE"
func ParseObjectKind(s string) (ObjectKind, error)

// libs/atlas-saga
const MoveEnvironment  Action = "move_environment"
const ResetEnvironment Action = "reset_environment"
type MoveEnvironmentPayload struct{ WorldId; ChannelId; MapId; Instance; Kind; Name; State }
type ResetEnvironmentPayload struct{ WorldId; ChannelId; MapId; Instance }

// Kafka wire strings
"SET_ENVIRONMENT_STATE"      // COMMAND_TOPIC_MAP
"RESET_ENVIRONMENT"          // COMMAND_TOPIC_MAP
"ENVIRONMENT_STATE_CHANGED"  // EVENT_TOPIC_MAP_STATUS
"ENVIRONMENT_RESET"          // EVENT_TOPIC_MAP_STATUS
```

---

## Task 1: `field.ObjectKind` shared enum

### Files

- `libs/atlas-constants/field/constants.go` — add `ObjectKind`, the two constants, and `ParseObjectKind`
- `libs/atlas-constants/field/constants_test.go` — **new file**

Module root (`go build` / `go test` cwd): `libs/atlas-constants`
Patterns to copy: `libs/atlas-constants/field/model_test.go` (plain table-driven, no tenant context)

**Interfaces:**
- Consumes: nothing.
- Produces: `field.ObjectKind` (string type), `field.ObjectKindEnvironment`, `field.ObjectKindObstacle`, `field.ParseObjectKind(s string) (ObjectKind, error)`. Every later task imports these from `github.com/Chronicle20/atlas/libs/atlas-constants/field`.

`ParseObjectKind` is the single home of the default-to-`ENVIRONMENT` rule (design.md §6.4) so the two script executors, the Kafka handler, and the REST handler cannot drift.

- [ ] **Step 1: Write the failing test**

`libs/atlas-constants/field/constants_test.go`, package `field`. One table-driven test `TestParseObjectKind`, subtest name = the `name` column.

| subtest name | input `s` | want kind | want error |
|---|---|---|---|
| `empty defaults to environment` | `""` | `ObjectKindEnvironment` | nil |
| `environment` | `"ENVIRONMENT"` | `ObjectKindEnvironment` | nil |
| `obstacle` | `"OBSTACLE"` | `ObjectKindObstacle` | nil |
| `lowercase environment` | `"environment"` | `ObjectKindEnvironment` | nil |
| `lowercase obstacle` | `"obstacle"` | `ObjectKindObstacle` | nil |
| `mixed case obstacle` | `"Obstacle"` | `ObjectKindObstacle` | nil |
| `unknown` | `"GATE"` | `""` (zero value) | non-nil, message exactly `unrecognized object kind [GATE]` |
| `whitespace only` | `"   "` | `ObjectKindEnvironment` | nil |

Assert the error message with `err.Error() != "unrecognized object kind [GATE]"` — the exact string matters because Task 5 asserts a `400` is produced from it and Task 12/13 surface it to script authors.

Add a second test `TestObjectKindConstants` asserting `string(ObjectKindEnvironment) == "ENVIRONMENT"` and `string(ObjectKindObstacle) == "OBSTACLE"` — these two strings are the Kafka and JSON wire values and must not drift.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-constants && go test ./field/... -run 'TestParseObjectKind|TestObjectKindConstants' -v`
Expected: FAIL — `undefined: ParseObjectKind`, `undefined: ObjectKindEnvironment`.

- [ ] **Step 3: Write minimal implementation**

Append to `libs/atlas-constants/field/constants.go` (the file currently holds only `IdFormat` and `type Id string`; add `fmt` and `strings` imports):

```go
// ObjectKind selects which clientbound opcode carries a named field-object
// state change. Both kinds resolve to the same client dictionary
// (CMapLoadable::m_mNamedObj) and the same client function
// (CMapLoadable::SetObjectState), so the distinction is a transport
// preference on the server, not a behavioural one on the client.
type ObjectKind string

const (
	ObjectKindEnvironment ObjectKind = "ENVIRONMENT"
	ObjectKindObstacle    ObjectKind = "OBSTACLE"
)

// ParseObjectKind resolves a wire or script string to an ObjectKind. A blank
// value defaults to ObjectKindEnvironment: no authored script in the repo
// specifies a kind today, and the upstream conversion source
// (the Cosmic moveEnvironment script call) is the SetObjectState path.
func ParseObjectKind(s string) (ObjectKind, error) {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "":
		return ObjectKindEnvironment, nil
	case string(ObjectKindEnvironment):
		return ObjectKindEnvironment, nil
	case string(ObjectKindObstacle):
		return ObjectKindObstacle, nil
	default:
		return "", fmt.Errorf("unrecognized object kind [%s]", s)
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd libs/atlas-constants && go test ./field/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-constants/field/constants.go libs/atlas-constants/field/constants_test.go
git commit -m "feat(atlas-constants): add field.ObjectKind and ParseObjectKind"
```

---

## Task 2: `libs/atlas-saga` actions and payloads

### Files

- `libs/atlas-saga/model.go` — add two `Action` constants in the "Field effect actions" block (currently `model.go:285-289`)
- `libs/atlas-saga/payloads.go` — add two payload structs after `FieldEffectWeatherPayload` (`payloads.go:1300-1310`)
- `libs/atlas-saga/unmarshal.go` — add two `case` arms after the `FieldEffectWeather` arm (`unmarshal.go:636-641`)
- `libs/atlas-saga/unmarshal_test.go` — new test cases

Module root: `libs/atlas-saga`
Patterns to copy: `libs/atlas-saga/payloads.go:1300-1310` (`FieldEffectWeatherPayload` field layout and comment style); `libs/atlas-saga/unmarshal.go:636-641` (the `case` arm shape, repeated verbatim with the new type)

**Interfaces:**
- Consumes: `field.ObjectKind` from Task 1.
- Produces: `saga.MoveEnvironment`, `saga.ResetEnvironment`, `saga.MoveEnvironmentPayload`, `saga.ResetEnvironmentPayload`. Tasks 8, 12, 13 consume these.

Import aliases already in `payloads.go`: `uuid` (unaliased `github.com/google/uuid`), `channel`, `world` (unaliased), `_map` (aliased `github.com/Chronicle20/atlas/libs/atlas-constants/map`). Add `"github.com/Chronicle20/atlas/libs/atlas-constants/field"` unaliased.

- [ ] **Step 1: Write the failing test**

Append to `libs/atlas-saga/unmarshal_test.go`. Two tests, following the existing table-driven JSON round-trip shape at `libs/atlas-saga/unmarshal_test.go:1-10` (no tenant context, no logger — pure `json.Unmarshal` into `Step[any]`).

`TestUnmarshalMoveEnvironmentStep` — unmarshal this exact JSON into a `Step[any]`:

```json
{
  "stepId": "move-environment-gate01",
  "status": "pending",
  "action": "move_environment",
  "payload": {
    "worldId": 0,
    "channelId": 1,
    "mapId": 910010000,
    "instance": "00000000-0000-0000-0000-000000000000",
    "kind": "OBSTACLE",
    "name": "gate01",
    "state": 3
  }
}
```

Assertions, field by field:
- `s.Action == MoveEnvironment`
- `s.Payload` type-asserts to `MoveEnvironmentPayload` (`ok == true`)
- `p.WorldId == world.Id(0)`
- `p.ChannelId == channel.Id(1)`
- `p.MapId == _map.Id(910010000)`
- `p.Instance == uuid.Nil`
- `p.Kind == field.ObjectKindObstacle`
- `p.Name == "gate01"`
- `p.State == uint32(3)`

`TestUnmarshalResetEnvironmentStep` — same shape with `"action": "reset_environment"` and payload `{"worldId":0,"channelId":1,"mapId":910010000,"instance":"00000000-0000-0000-0000-000000000000"}`. Assert `s.Action == ResetEnvironment`, the payload type-asserts to `ResetEnvironmentPayload`, and `p.MapId == _map.Id(910010000)`.

Add a third case to whichever existing table enumerates action-string round trips (if one exists in `unmarshal_test.go`); otherwise the two tests above are sufficient.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-saga && go test ./... -run 'TestUnmarshal(Move|Reset)EnvironmentStep' -v`
Expected: FAIL — `undefined: MoveEnvironment`, `undefined: MoveEnvironmentPayload`.

- [ ] **Step 3: Write minimal implementation**

`libs/atlas-saga/model.go`, immediately after `PlayJukebox Action = "play_jukebox"` (currently line 289), inside the same const block:

```go
	// Environment object actions. Both are fire-and-forget: the step
	// completes when the command is produced, and neither has a
	// compensating action -- reversing a move is the script author's job
	// (a second move_environment, or reset_environment).
	MoveEnvironment  Action = "move_environment"
	ResetEnvironment Action = "reset_environment"
```

`libs/atlas-saga/payloads.go`, after `FieldEffectWeatherPayload` (add the `field` import):

```go
// MoveEnvironmentPayload represents the payload for setting the state of one
// named field object. Kind selects the clientbound opcode; see
// libs/atlas-constants/field.ObjectKind.
type MoveEnvironmentPayload struct {
	WorldId   world.Id         `json:"worldId"`   // WorldId of the field
	ChannelId channel.Id       `json:"channelId"` // ChannelId of the field
	MapId     _map.Id          `json:"mapId"`     // MapId of the field
	Instance  uuid.UUID        `json:"instance"`  // Instance UUID of the field
	Kind      field.ObjectKind `json:"kind"`      // ENVIRONMENT or OBSTACLE
	Name      string           `json:"name"`      // Opaque object name, not validated against WZ data
	State     uint32           `json:"state"`     // New object state
}

// ResetEnvironmentPayload represents the payload for clearing every tracked
// field object and restoring the field's objects to their default state.
type ResetEnvironmentPayload struct {
	WorldId   world.Id   `json:"worldId"`   // WorldId of the field
	ChannelId channel.Id `json:"channelId"` // ChannelId of the field
	MapId     _map.Id    `json:"mapId"`     // MapId of the field
	Instance  uuid.UUID  `json:"instance"`  // Instance UUID of the field
}
```

`libs/atlas-saga/unmarshal.go`, after the `case PlayJukebox:` arm (currently lines 642-647):

```go
	case MoveEnvironment:
		var payload MoveEnvironmentPayload
		if err := json.Unmarshal(aux.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.Action, err)
		}
		s.Payload = any(payload).(T)
	case ResetEnvironment:
		var payload ResetEnvironmentPayload
		if err := json.Unmarshal(aux.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.Action, err)
		}
		s.Payload = any(payload).(T)
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd libs/atlas-saga && go test ./... `
Expected: PASS (whole module, to catch any exhaustive-switch test elsewhere in the package).

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-saga/model.go libs/atlas-saga/payloads.go libs/atlas-saga/unmarshal.go libs/atlas-saga/unmarshal_test.go
git commit -m "feat(atlas-saga): add move_environment and reset_environment actions"
```

---

## Task 3: `atlas-maps` `map/environment` registry, processor, and REST model

### Files

- `services/atlas-maps/atlas.com/maps/map/environment/registry.go` — **new file**
- `services/atlas-maps/atlas.com/maps/map/environment/registry_test.go` — **new file**
- `services/atlas-maps/atlas.com/maps/map/environment/processor.go` — **new file**
- `services/atlas-maps/atlas.com/maps/map/environment/processor_test.go` — **new file**
- `services/atlas-maps/atlas.com/maps/map/environment/rest.go` — **new file**

Module root: `services/atlas-maps/atlas.com/maps`
Patterns to copy: `services/atlas-maps/atlas.com/maps/map/weather/registry.go:1-57` (singleton `once.Do`, `FieldKey`, RWMutex accessors — but **drop** the `GetExpired`/`ExpiredEntry` wall-clock machinery at `registry.go:59-84`, which has no analogue here); `services/atlas-maps/atlas.com/maps/map/weather/processor.go:1-45` (interface + `ProcessorImpl` + `NewProcessor` + `tenant.MustFromContext`); `services/atlas-maps/atlas.com/maps/map/weather/rest.go:1-30` (jsonapi `RestModel` with `GetID`/`GetName`/`SetID`/`Transform`); `services/atlas-maps/atlas.com/maps/map/jukebox/registry_test.go` (registry test shape).

**Interfaces:**
- Consumes: `field.ObjectKind` (Task 1).
- Produces, for Tasks 4/5/6:
  ```go
  package environment
  type FieldKey struct { Tenant tenant.Model; Field field.Model }
  type ObjectEntry struct { Kind field.ObjectKind; Name string; State uint32 }
  type Processor interface {
      Set(f field.Model, kind field.ObjectKind, name string, state uint32) (ObjectEntry, error)
      Reset(f field.Model) []ObjectEntry
      GetAll(f field.Model) []ObjectEntry
  }
  func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor
  type RestModel struct { Id string; Kind string; Name string; State uint32 }
  func Transform(e ObjectEntry) (RestModel, error)
  ```

Note two deliberate divergences from `map/weather`: `Registry.Get` returns a **copy** of the slice (weather's `Get` at `registry.go:46-51` returns the value directly — safe there because `WeatherEntry` is a scalar struct, unsafe here because the value is a slice header), and there is no expiry sweeper.

`Set` is keyed on the pair `(Kind, Name)` and replaces **in place**, preserving the entry's original index. Insertion order is client-observable on replay (design.md §3.2).

- [ ] **Step 1: Write the failing test**

Two new test files, package `environment`. Both use the `map/jukebox/registry_test.go` setup shape: `test.NewNullLogger()` for the logger, `tenant.Create(uuid.New(), "GMS", 83, 1)` + `tenant.WithContext(context.Background(), tn)` for the context, `field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(910010000)).SetInstance(uuid.Nil).Build()` for the field.

**Important:** the registry is a process-wide singleton. Every test must build a **fresh tenant** via `tenant.Create(uuid.New(), ...)` so tests cannot leak state into one another, and must not rely on the registry being empty for a tenant it did not create.

`registry_test.go`:

| test func | scenario | assertion |
|---|---|---|
| `TestRegistrySetAppendsNewEntry` | `Set(key, {Obstacle,"a",1})`, then `Set(key, {Environment,"b",2})` | `Get(key)` returns exactly `[{Obstacle,"a",1},{Environment,"b",2}]` in that order |
| `TestRegistrySetReplacesInPlace` | `Set` `{Obstacle,"a",1}`, `{Environment,"b",2}`, `{Obstacle,"a",7}` | `Get(key)` has `len == 2`; index 0 is `{Obstacle,"a",7}`; index 1 is `{Environment,"b",2}` — position preserved, not moved to the end |
| `TestRegistrySetSameNameDifferentKindAreDistinct` | `Set` `{Obstacle,"a",1}` then `{Environment,"a",2}` | `Get(key)` has `len == 2` — the key is the `(Kind, Name)` pair |
| `TestRegistryGetReturnsCopy` | `Set` one entry; take `got := Get(key)`; mutate `got[0].State = 99` | a second `Get(key)` still reports `State == 1` |
| `TestRegistryGetUntrackedReturnsEmptyNotNil` | `Get` on a key never `Set` | `len == 0` **and** the returned slice is non-nil (`got == nil` must be false) |
| `TestRegistryTenantIsolation` | two tenants from separate `tenant.Create(uuid.New(), ...)` calls, same `field.Model`; `Set {Environment,"a",1}` under tenant A only | `Get` with tenant B's key returns `len == 0`; `Get` with tenant A's key returns `len == 1` |
| `TestRegistryDeleteRemovesKey` | `Set` two entries, then `Delete(key)` | `Get(key)` returns `len == 0` |

`processor_test.go`:

| test func | scenario | assertion |
|---|---|---|
| `TestProcessorSetRejectsBlankName` | `Set(f, ObjectKindEnvironment, "", 1)` | returns non-nil error with message exactly `environment object name must not be blank`; `GetAll(f)` returns `len == 0` |
| `TestProcessorSetRejectsWhitespaceName` | `Set(f, ObjectKindEnvironment, "   ", 1)` | same error; nothing tracked |
| `TestProcessorSetReturnsEntry` | `Set(f, ObjectKindObstacle, "obs3", 2)` | returned `ObjectEntry` equals `{ObjectKindObstacle, "obs3", 2}`; nil error |
| `TestProcessorSetIsIdempotent` | `Set(f, ObjectKindObstacle, "obs3", 2)` twice | `GetAll(f)` has `len == 1` |
| `TestProcessorResetReturnsClearedAndEmpties` | `Set` `{Obstacle,"a",1}` and `{Environment,"b",2}`, then `Reset(f)` | returned slice is `[{Obstacle,"a",1},{Environment,"b",2}]` in insertion order; a following `GetAll(f)` returns `len == 0` |
| `TestProcessorResetOnUntrackedFieldReturnsEmpty` | `Reset(f)` with nothing tracked | returns `len == 0`, non-nil slice, no panic |
| `TestProcessorGetAllUntracked` | `GetAll(f)` with nothing tracked | returns `len == 0`, non-nil slice, no panic |

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-maps/atlas.com/maps && go test ./map/environment/... -v`
Expected: FAIL — the `environment` package does not exist (`no Go files in .../map/environment`).

- [ ] **Step 3: Write minimal implementation**

`registry.go`:

```go
package environment

import (
	"slices"
	"sync"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type FieldKey struct {
	Tenant tenant.Model
	Field  field.Model
}

type ObjectEntry struct {
	Kind  field.ObjectKind
	Name  string
	State uint32
}

type Registry struct {
	mutex   sync.RWMutex
	entries map[FieldKey][]ObjectEntry
}

var (
	registry *Registry
	once     sync.Once
)

func getRegistry() *Registry {
	once.Do(func() {
		registry = &Registry{}
		registry.entries = make(map[FieldKey][]ObjectEntry)
	})
	return registry
}

// Set replaces the entry for (entry.Kind, entry.Name) in place, preserving its
// original index, or appends it when it is new. Insertion order is preserved
// because replay order is observable to the client.
func (r *Registry) Set(key FieldKey, entry ObjectEntry) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	existing := r.entries[key]
	for i, e := range existing {
		if e.Kind == entry.Kind && e.Name == entry.Name {
			existing[i] = entry
			r.entries[key] = existing
			return
		}
	}
	r.entries[key] = append(existing, entry)
}

// Get returns a copy of the field's entries. ObjectEntry is a value type with
// no reference fields, so slices.Clone is a full deep copy and a concurrent
// Set cannot tear the caller's view.
func (r *Registry) Get(key FieldKey) []ObjectEntry {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	e, ok := r.entries[key]
	if !ok {
		return make([]ObjectEntry, 0)
	}
	return slices.Clone(e)
}

// Clear removes the key entirely and returns what it held, so a field with no
// tracked state occupies no map entry.
func (r *Registry) Clear(key FieldKey) []ObjectEntry {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	e, ok := r.entries[key]
	delete(r.entries, key)
	if !ok {
		return make([]ObjectEntry, 0)
	}
	return e
}

func (r *Registry) Delete(key FieldKey) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	delete(r.entries, key)
}
```

`processor.go`:

```go
package environment

import (
	"context"
	"errors"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// ErrBlankName is returned by Set for an empty or whitespace-only object name.
// Object names are otherwise opaque: no obstacle/environment name index exists
// in libs/atlas-wz or atlas-data, so a name the client does not know is a
// silent client-side no-op rather than a server error.
var ErrBlankName = errors.New("environment object name must not be blank")

type Processor interface {
	Set(f field.Model, kind field.ObjectKind, name string, state uint32) (ObjectEntry, error)
	Reset(f field.Model) []ObjectEntry
	GetAll(f field.Model) []ObjectEntry
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) Set(f field.Model, kind field.ObjectKind, name string, state uint32) (ObjectEntry, error) {
	if strings.TrimSpace(name) == "" {
		return ObjectEntry{}, ErrBlankName
	}
	t := tenant.MustFromContext(p.ctx)
	entry := ObjectEntry{Kind: kind, Name: name, State: state}
	getRegistry().Set(FieldKey{Tenant: t, Field: f}, entry)
	p.l.Debugf("Environment object [%s] kind [%s] set to state [%d] in map [%d] instance [%s].", name, kind, state, f.MapId(), f.Instance())
	return entry, nil
}

// Reset clears the field's tracked entries and returns what was cleared, so the
// caller can build the per-object reset sweep. FieldObstacleAllReset restores
// only the client's obstacle list, so non-obstacle named objects must be zeroed
// explicitly -- see design.md section 1.2.
func (p *ProcessorImpl) Reset(f field.Model) []ObjectEntry {
	t := tenant.MustFromContext(p.ctx)
	cleared := getRegistry().Clear(FieldKey{Tenant: t, Field: f})
	p.l.Debugf("Environment reset in map [%d] instance [%s]; cleared [%d] object(s).", f.MapId(), f.Instance(), len(cleared))
	return cleared
}

func (p *ProcessorImpl) GetAll(f field.Model) []ObjectEntry {
	t := tenant.MustFromContext(p.ctx)
	return getRegistry().Get(FieldKey{Tenant: t, Field: f})
}
```

`rest.go`:

```go
package environment

import "fmt"

type RestModel struct {
	Id    string `json:"-"`
	Kind  string `json:"kind"`
	Name  string `json:"name"`
	State uint32 `json:"state"`
}

func (m RestModel) GetID() string {
	return m.Id
}

func (m RestModel) GetName() string {
	return "environment-objects"
}

func (m *RestModel) SetID(idStr string) error {
	m.Id = idStr
	return nil
}

func Transform(e ObjectEntry) (RestModel, error) {
	return RestModel{
		Id:    fmt.Sprintf("%s:%s", e.Kind, e.Name),
		Kind:  string(e.Kind),
		Name:  e.Name,
		State: e.State,
	}, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-maps/atlas.com/maps && go test ./map/environment/... -v`
Expected: PASS, all cases.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-maps/atlas.com/maps/map/environment/
git commit -m "feat(atlas-maps): add map/environment registry, processor, and REST model"
```

---

## Task 4: `atlas-maps` Kafka command/event contract and consumer arms

### Files

- `services/atlas-maps/atlas.com/maps/kafka/message/map/command.go` — add two command type constants and two body structs
- `services/atlas-maps/atlas.com/maps/kafka/message/map/kafka.go` — add two status type constants and three structs
- `services/atlas-maps/atlas.com/maps/map/environment/producer.go` — **new file**
- `services/atlas-maps/atlas.com/maps/kafka/consumer/map/consumer.go` — two new handlers + two registrations in `InitHandlers`
- `services/atlas-maps/atlas.com/maps/kafka/consumer/map/consumer_test.go` — new test cases

Module root: `services/atlas-maps/atlas.com/maps`
Patterns to copy: `services/atlas-maps/atlas.com/maps/map/weather/producer.go:14-30` (`producer.CreateKey(int(f.MapId()))` + `producer.SingleMessageProvider`); `services/atlas-maps/atlas.com/maps/kafka/consumer/map/consumer.go:46-75` (`handleWeatherStartCommand`: `if c.Type != ...` early return, `field.NewBuilder(...)`, processor call, `producer.ProviderImpl(l)(ctx)(mapKafka.EnvEventTopicMapStatus)(...)` with an `Errorf` on failure); `consumer.go:32-44` (`InitHandlers` registration shape).

**Interfaces:**
- Consumes: `environment.Processor` and `environment.ObjectEntry` (Task 3), `field.ParseObjectKind` (Task 1).
- Produces, for Tasks 6/7/10/11:
  ```go
  // kafka/message/map
  const CommandTypeSetEnvironmentState = "SET_ENVIRONMENT_STATE"
  const CommandTypeResetEnvironment    = "RESET_ENVIRONMENT"
  type SetEnvironmentStateCommandBody struct { Kind string; Name string; State uint32 }
  type ResetEnvironmentCommandBody struct{}
  const EventTopicMapStatusTypeEnvironmentStateChanged = "ENVIRONMENT_STATE_CHANGED"
  const EventTopicMapStatusTypeEnvironmentReset        = "ENVIRONMENT_RESET"
  type EnvironmentStateChanged struct { Kind string; Name string; State uint32 }
  type EnvironmentObject struct { Kind string; Name string }
  type EnvironmentReset struct { Cleared []EnvironmentObject }
  // map/environment
  func EnvironmentStateChangedEventProvider(transactionId uuid.UUID, f field.Model, e ObjectEntry) model.Provider[[]kafka.Message]
  func EnvironmentResetEventProvider(transactionId uuid.UUID, f field.Model, cleared []ObjectEntry) model.Provider[[]kafka.Message]
  ```

`Kind` is a plain `string` on the wire, not `field.ObjectKind`, so an unrecognised value from a future producer deserialises and is rejected by `ParseObjectKind` in the handler rather than failing the whole decode.

- [ ] **Step 1: Write the failing test**

Append to `services/atlas-maps/atlas.com/maps/kafka/consumer/map/consumer_test.go`, package `_map`. Setup copied from `consumer_test.go:18-46`: `l, _ := test.NewNullLogger()`; `tn, _ := tenant.Create(uuid.New(), "GMS", 83, 1)`; `ctx := tenant.WithContext(context.Background(), tn)`; `f := field.NewBuilder(0, 1, 910010000).SetInstance(uuid.Nil).Build()`. Handlers are invoked directly (`handleSetEnvironmentStateCommand()(l, ctx, cmd)`) and asserted through `environment.NewProcessor(l, ctx).GetAll(f)`. Use a fresh `uuid.New()` tenant per test — the registry is a process-wide singleton.

| test func | command | assertion |
|---|---|---|
| `TestHandleSetEnvironmentStateCommand_Applies` | `Command[SetEnvironmentStateCommandBody]{Type: CommandTypeSetEnvironmentState, WorldId:0, ChannelId:1, MapId:910010000, Instance: uuid.Nil, Body:{Kind:"OBSTACLE", Name:"obs3", State:2}}` | `GetAll(f)` returns exactly one entry equal to `{field.ObjectKindObstacle, "obs3", 2}` |
| `TestHandleSetEnvironmentStateCommand_BlankKindDefaultsEnvironment` | same, `Body.Kind == ""` , `Name:"gate01"`, `State:1` | the single entry's `Kind == field.ObjectKindEnvironment` |
| `TestHandleSetEnvironmentStateCommand_WrongTypeIgnored` | same body, `Type: CommandTypeWeatherStart` | `GetAll(f)` returns `len == 0` |
| `TestHandleSetEnvironmentStateCommand_UnknownKindRejected` | `Body.Kind == "GATE"` | `GetAll(f)` returns `len == 0` (no mutation); handler returns without panicking |
| `TestHandleSetEnvironmentStateCommand_BlankNameRejected` | `Body{Kind:"ENVIRONMENT", Name:"", State:1}` | `GetAll(f)` returns `len == 0` |
| `TestHandleResetEnvironmentCommand_ClearsTracked` | pre-seed via `environment.NewProcessor(l, ctx).Set(f, ObjectKindObstacle, "a", 1)` and `Set(f, ObjectKindEnvironment, "b", 2)`, then dispatch `Command[ResetEnvironmentCommandBody]{Type: CommandTypeResetEnvironment, ...}` | `GetAll(f)` returns `len == 0` |
| `TestHandleResetEnvironmentCommand_WrongTypeIgnored` | pre-seed one entry, dispatch with `Type: CommandTypePlayJukebox` | `GetAll(f)` still returns `len == 1` |

Also add, in `services/atlas-maps/atlas.com/maps/map/environment/producer_test.go` (**new file**, setup copied from `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/map_command/producer_test.go:1-40` — no tenant context needed, assert by unmarshalling `msgs[0].Value`):

`TestEnvironmentStateChangedEventProvider`: build with `transactionId := uuid.New()`, `f` as above, `ObjectEntry{ObjectKindObstacle, "obs3", 2}`. Assert `len(msgs) == 1`; `string(msgs[0].Key) == string(producer.CreateKey(int(f.MapId())))`; unmarshalling `msgs[0].Value` into `mapKafka.StatusEvent[mapKafka.EnvironmentStateChanged]` yields `Type == "ENVIRONMENT_STATE_CHANGED"`, `WorldId == 0`, `ChannelId == 1`, `MapId == 910010000`, `Instance == uuid.Nil`, `Body.Kind == "OBSTACLE"`, `Body.Name == "obs3"`, `Body.State == 2`, `TransactionId == transactionId`.

`TestEnvironmentResetEventProvider`: cleared `[]ObjectEntry{{ObjectKindObstacle,"a",1},{ObjectKindEnvironment,"b",2}}`. Assert `Type == "ENVIRONMENT_RESET"`; `len(Body.Cleared) == 2`; `Body.Cleared[0] == EnvironmentObject{Kind:"OBSTACLE", Name:"a"}`; `Body.Cleared[1] == EnvironmentObject{Kind:"ENVIRONMENT", Name:"b"}` — order preserved.

`TestEnvironmentResetEventProvider_EmptyCleared`: cleared `[]ObjectEntry{}`. Assert `Body.Cleared` marshals to an empty JSON array, not `null` (the channel iterates it directly).

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-maps/atlas.com/maps && go test ./kafka/consumer/map/... ./map/environment/... -v`
Expected: FAIL — `undefined: handleSetEnvironmentStateCommand`, `undefined: mapKafka.CommandTypeSetEnvironmentState`, `undefined: EnvironmentStateChangedEventProvider`.

- [ ] **Step 3: Write minimal implementation**

`kafka/message/map/command.go` — add to the existing const block and append the bodies:

```go
	CommandTypeSetEnvironmentState = "SET_ENVIRONMENT_STATE"
	CommandTypeResetEnvironment    = "RESET_ENVIRONMENT"
```
```go
// SetEnvironmentStateCommandBody carries one named field-object state change.
// Kind is a plain string so an unrecognised value from a future producer
// deserialises and is rejected by the handler, rather than failing the decode.
type SetEnvironmentStateCommandBody struct {
	Kind  string `json:"kind"`
	Name  string `json:"name"`
	State uint32 `json:"state"`
}

// ResetEnvironmentCommandBody is empty; field routing comes from the envelope.
type ResetEnvironmentCommandBody struct{}
```

`kafka/message/map/kafka.go` — add to the existing const block and append:

```go
	EventTopicMapStatusTypeEnvironmentStateChanged = "ENVIRONMENT_STATE_CHANGED"
	EventTopicMapStatusTypeEnvironmentReset        = "ENVIRONMENT_RESET"
```
```go
type EnvironmentStateChanged struct {
	Kind  string `json:"kind"`
	Name  string `json:"name"`
	State uint32 `json:"state"`
}

type EnvironmentObject struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}

// EnvironmentReset carries every entry that was tracked. It is not an empty
// body: FieldObstacleAllReset restores only the client's obstacle list, so the
// channel must be told which non-obstacle objects to zero explicitly, and the
// channel keeps no registry of its own -- see design.md section 1.2.
type EnvironmentReset struct {
	Cleared []EnvironmentObject `json:"cleared"`
}
```

`map/environment/producer.go`:

```go
package environment

import (
	mapKafka "atlas-maps/kafka/message/map"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func EnvironmentStateChangedEventProvider(transactionId uuid.UUID, f field.Model, e ObjectEntry) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(f.MapId()))
	value := &mapKafka.StatusEvent[mapKafka.EnvironmentStateChanged]{
		TransactionId: transactionId,
		WorldId:       f.WorldId(),
		ChannelId:     f.ChannelId(),
		MapId:         f.MapId(),
		Instance:      f.Instance(),
		Type:          mapKafka.EventTopicMapStatusTypeEnvironmentStateChanged,
		Body: mapKafka.EnvironmentStateChanged{
			Kind:  string(e.Kind),
			Name:  e.Name,
			State: e.State,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func EnvironmentResetEventProvider(transactionId uuid.UUID, f field.Model, cleared []ObjectEntry) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(f.MapId()))
	objects := make([]mapKafka.EnvironmentObject, 0, len(cleared))
	for _, e := range cleared {
		objects = append(objects, mapKafka.EnvironmentObject{Kind: string(e.Kind), Name: e.Name})
	}
	value := &mapKafka.StatusEvent[mapKafka.EnvironmentReset]{
		TransactionId: transactionId,
		WorldId:       f.WorldId(),
		ChannelId:     f.ChannelId(),
		MapId:         f.MapId(),
		Instance:      f.Instance(),
		Type:          mapKafka.EventTopicMapStatusTypeEnvironmentReset,
		Body:          mapKafka.EnvironmentReset{Cleared: objects},
	}
	return producer.SingleMessageProvider(key, value)
}
```

`kafka/consumer/map/consumer.go` — register both in `InitHandlers` after the jukebox registration (currently `consumer.go:39-41`), following the same `if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(...))); err != nil { return err }` shape, then append the handlers (import `"atlas-maps/map/environment"`):

```go
func handleSetEnvironmentStateCommand() func(l logrus.FieldLogger, ctx context.Context, c mapKafka.Command[mapKafka.SetEnvironmentStateCommandBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, c mapKafka.Command[mapKafka.SetEnvironmentStateCommandBody]) {
		if c.Type != mapKafka.CommandTypeSetEnvironmentState {
			return
		}

		kind, err := field.ParseObjectKind(c.Body.Kind)
		if err != nil {
			l.WithError(err).Errorf("Rejecting environment state command for map [%d] instance [%s].", c.MapId, c.Instance)
			return
		}

		f := field.NewBuilder(c.WorldId, c.ChannelId, c.MapId).SetInstance(c.Instance).Build()
		entry, err := environment.NewProcessor(l, ctx).Set(f, kind, c.Body.Name, c.Body.State)
		if err != nil {
			l.WithError(err).Errorf("Rejecting environment state command for map [%d] instance [%s].", c.MapId, c.Instance)
			return
		}

		// Emitted unconditionally, including for a re-set to the same state:
		// scripts may rely on the re-broadcast to re-run the client animation.
		err = producer.ProviderImpl(l)(ctx)(mapKafka.EnvEventTopicMapStatus)(environment.EnvironmentStateChangedEventProvider(c.TransactionId, f, entry))
		if err != nil {
			l.WithError(err).Errorf("Unable to produce environment state changed event for map [%d] instance [%s].", c.MapId, c.Instance)
		}
	}
}

func handleResetEnvironmentCommand() func(l logrus.FieldLogger, ctx context.Context, c mapKafka.Command[mapKafka.ResetEnvironmentCommandBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, c mapKafka.Command[mapKafka.ResetEnvironmentCommandBody]) {
		if c.Type != mapKafka.CommandTypeResetEnvironment {
			return
		}

		f := field.NewBuilder(c.WorldId, c.ChannelId, c.MapId).SetInstance(c.Instance).Build()
		cleared := environment.NewProcessor(l, ctx).Reset(f)

		err := producer.ProviderImpl(l)(ctx)(mapKafka.EnvEventTopicMapStatus)(environment.EnvironmentResetEventProvider(c.TransactionId, f, cleared))
		if err != nil {
			l.WithError(err).Errorf("Unable to produce environment reset event for map [%d] instance [%s].", c.MapId, c.Instance)
		}
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-maps/atlas.com/maps && go build ./... && go test ./kafka/... ./map/environment/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-maps/atlas.com/maps/kafka services/atlas-maps/atlas.com/maps/map/environment
git commit -m "feat(atlas-maps): add environment command/event contract and consumer arms"
```

---

## Task 5: `atlas-maps` environment REST resource

### Files

- `services/atlas-maps/atlas.com/maps/map/environment/resource.go` — **new file**
- `services/atlas-maps/atlas.com/maps/map/environment/resource_test.go` — **new file**
- `services/atlas-maps/atlas.com/maps/main.go` — one `AddRouteInitializer(environment.InitResource(GetServer()))` line after `jukebox.InitResource` (currently `main.go:149`)

Module root: `services/atlas-maps/atlas.com/maps`
Patterns to copy: `services/atlas-maps/atlas.com/maps/map/weather/resource.go:22-57` (route prefix, `rest.RegisterHandler`, the `ParseWorldId → ParseChannelId → ParseMapId → ParseInstanceId` nest, `field.NewBuilder(...).SetInstance(instanceId).Build()`, `server.MarshalResponse`). `rest.RegisterInputHandler[M]` is at `services/atlas-maps/atlas.com/maps/rest/handler.go:30` and `rest.ParseInput[M]` at `handler.go:24`.

**Interfaces:**
- Consumes: `environment.Processor`, `environment.RestModel`, `environment.Transform` (Task 3); `environment.EnvironmentStateChangedEventProvider`, `environment.EnvironmentResetEventProvider` (Task 4); `field.ParseObjectKind` (Task 1).
- Produces: `func InitResource(si jsonapi.ServerInformation) server.RouteInitializer`, and the three routes below, which Task 9's channel REST client consumes.

Routes — `instanceId` is a required **path segment**, not a query parameter (design.md §5, correcting PRD §5):

```
GET    /worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/environment
POST   /worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/environment
DELETE /worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/environment
```

Status codes: `GET` → `200` with a possibly-empty `data` array, **never `404`** (a deliberate divergence from `weather/resource.go:39-42`, which 404s because weather is a singleton and environment is a collection). `POST` → `202 Accepted`; `400` on blank `name` or unparseable `kind`. `DELETE` → `204 No Content`, including for an untracked field.

Both mutating verbs call the same `environment.Processor` methods the Kafka handlers call and then produce the same status event, so "REST behaves exactly as the command path" is structural.

- [ ] **Step 1: Write the failing test**

`resource_test.go`, package `environment`. Setup shape copied from `services/atlas-maps/atlas.com/maps/rest/handler_test.go:1-45`: `httptest.NewRequest` + `mux.SetURLVars` + `test.NewNullLogger()`, plus a tenant context via `tenant.Create(uuid.New(), "GMS", 83, 1)` / `tenant.WithContext` because the processor requires one. Build the `rest.HandlerDependency` / `rest.HandlerContext` the way `handler_test.go` does. Use a fresh tenant per test (singleton registry).

The event producers require a Kafka connection, so these tests exercise the handler functions that do **not** produce (`GET`) directly, and for `POST`/`DELETE` assert the **status code and registry effect** only — construct the handler through the same nest but tolerate a produce failure (the handler logs it and still returns its status code; assert the code and the registry, not the emit).

| test func | request | assertion |
|---|---|---|
| `TestGetEnvironmentInMap_Empty` | `GET` on an untracked field | status `200`; body's `data` is present and an empty array; **not** `404` |
| `TestGetEnvironmentInMap_ReturnsTrackedInOrder` | pre-seed `Set(f, ObjectKindObstacle,"a",1)` then `Set(f, ObjectKindEnvironment,"b",2)`; `GET` | status `200`; `data` has 2 elements; `data[0].id == "OBSTACLE:a"`, `data[0].attributes.kind == "OBSTACLE"`, `.name == "a"`, `.state == 1`; `data[1].id == "ENVIRONMENT:b"`, `.state == 2` |
| `TestGetEnvironmentInMap_TenantIsolation` | seed under tenant A; `GET` with tenant B's context, same field | status `200`; `data` empty |
| `TestPostEnvironment_BlankName` | `POST` body attributes `{"kind":"ENVIRONMENT","name":"","state":1}` | status `400`; `GetAll(f)` returns `len == 0` |
| `TestPostEnvironment_WhitespaceName` | `name` = `"   "` | status `400`; nothing tracked |
| `TestPostEnvironment_BadKind` | `{"kind":"GATE","name":"gate01","state":1}` | status `400`; nothing tracked |
| `TestPostEnvironment_Accepted` | `{"kind":"OBSTACLE","name":"obs3","state":2}` | status `202`; `GetAll(f)` returns exactly `[{ObjectKindObstacle,"obs3",2}]` |
| `TestPostEnvironment_BlankKindDefaults` | `{"kind":"","name":"gate01","state":1}` | status `202`; the single entry's `Kind == field.ObjectKindEnvironment` |
| `TestDeleteEnvironment_Untracked` | `DELETE` on an untracked field | status `204` |
| `TestDeleteEnvironment_ClearsTracked` | seed two entries; `DELETE` | status `204`; `GetAll(f)` returns `len == 0` |

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-maps/atlas.com/maps && go test ./map/environment/... -run 'TestGetEnvironment|TestPostEnvironment|TestDeleteEnvironment' -v`
Expected: FAIL — `undefined: InitResource`, `undefined: handleGetEnvironmentInMap`.

- [ ] **Step 3: Write minimal implementation**

`resource.go` — three handlers registered on one path, following `weather/resource.go` structurally. Handler names: `getEnvironmentInMap`, `setEnvironmentInMap`, `resetEnvironmentInMap` (the `rest.RegisterHandler` handler-name strings). Key points the implementer must get right:

- `InitResource` registers all three methods on the same `r.HandleFunc(...)` path with `.Methods(http.MethodGet)`, `.Methods(http.MethodPost)`, `.Methods(http.MethodDelete)` respectively; `POST` uses `rest.RegisterInputHandler[RestModel](l)(si)(setEnvironmentInMap, handleSetEnvironmentInMap)`, the other two use `rest.RegisterHandler(l)(si)(...)`.
- `handleGetEnvironmentInMap`: build `f`, call `NewProcessor(d.Logger(), d.Context()).GetAll(f)`, map each `ObjectEntry` through `Transform`, and respond with `server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(r.URL.Query())(res)`. On zero entries respond with an empty `[]RestModel{}` and `200` — **never** write `http.StatusNotFound`.
- `handleSetEnvironmentInMap` (via `rest.ParseInput[RestModel]`): `kind, err := field.ParseObjectKind(input.Kind)`; on error `w.WriteHeader(http.StatusBadRequest); return`. Then `entry, err := p.Set(f, kind, input.Name, input.State)`; on `ErrBlankName` (or any error) `w.WriteHeader(http.StatusBadRequest); return`. On success produce `EnvironmentStateChangedEventProvider(uuid.New(), f, entry)` to `mapKafka.EnvEventTopicMapStatus` via `producer.ProviderImpl(d.Logger())(d.Context())(...)`, log an `Errorf` on produce failure, then `w.WriteHeader(http.StatusAccepted)`.
- `handleResetEnvironmentInMap`: `cleared := p.Reset(f)`, produce `EnvironmentResetEventProvider(uuid.New(), f, cleared)`, log an `Errorf` on produce failure, then `w.WriteHeader(http.StatusNoContent)`.

`main.go` — add the import `"atlas-maps/map/environment"` and one line to the builder chain immediately after `AddRouteInitializer(jukebox.InitResource(GetServer())).`:

```go
		AddRouteInitializer(environment.InitResource(GetServer())).
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-maps/atlas.com/maps && go build ./... && go test ./map/environment/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-maps/atlas.com/maps/map/environment services/atlas-maps/atlas.com/maps/main.go
git commit -m "feat(atlas-maps): expose environment object state over REST"
```

---

## Task 6: `atlas-maps` empty-field teardown

### Files

- `services/atlas-maps/atlas.com/maps/map/character/registry.go` — `RemoveCharacterFromAllMaps` returns the affected `[]MapKey` (currently void, `registry.go:105-114`)
- `services/atlas-maps/atlas.com/maps/map/character/processor.go` — `ExitAll` returns `[]MapKey` (currently void, `processor.go:66-69`); the `Processor` interface at `processor.go:22` changes with it
- `services/atlas-maps/atlas.com/maps/map/processor.go` — the empty-field check inside `Exit` (`processor.go:105-110`)
- `services/atlas-maps/atlas.com/maps/kafka/consumer/character/consumer.go` — the `ExitAll` call site (`consumer.go:194`) clears environment state for each affected key
- `services/atlas-maps/atlas.com/maps/map/character/processor_test.go` or `services/atlas-maps/atlas.com/maps/map/processor_test.go` — new test cases (use whichever already exists; create `map/environment_teardown_test.go` in package `_map` if neither is a fit)

Module root: `services/atlas-maps/atlas.com/maps`
Patterns to copy: `services/atlas-maps/atlas.com/maps/map/processor.go:105-110` (`Exit` is the single funnel — `ExitAndEmit`, `TransitionMap`, and `TransitionChannel` all route through it, so one check covers logout, warp, and channel change).

**Interfaces:**
- Consumes: `environment.Processor.Reset` (Task 3).
- Produces (breaking signature changes — only two call sites exist repo-wide):
  ```go
  // map/character
  func (r *Registry) RemoveCharacterFromAllMaps(t tenant.Model, characterId uint32) []MapKey
  func (p *ProcessorImpl) ExitAll(characterId uint32) []MapKey   // interface method changes too
  ```
  `ExitAll`'s only caller in the repo is `services/atlas-maps/atlas.com/maps/kafka/consumer/character/consumer.go:194`.

**Design correction recorded here:** design.md §3.4 describes this as a `CHARACTER_EXIT` hook and names `mapcharacter.Processor.ExitAndEmit`. Neither is accurate. There is no `CHARACTER_EXIT` handler in the maps character consumer; `consumer.go:144` is inside `handleStatusEventLogoutFunc`, and the method is `_map.Processor.ExitAndEmit` (`map/processor.go:112-116`), which delegates to `_map.ProcessorImpl.Exit`. Putting the check inside `Exit` is strictly better than patching the logout handler: it is one site instead of three and it also covers the warp and channel-change exits that `handleStatusEventLogoutFunc` does not.

**Teardown is registry-clear only, with no `ENVIRONMENT_RESET` event** — there is nobody left in the field to receive it, and the next entrant gets a clean (empty) replay anyway.

- [ ] **Step 1: Write the failing test**

Package `_map` (the `services/atlas-maps/atlas.com/maps/map` package). Setup shape copied from `services/atlas-maps/atlas.com/maps/map/processor_test.go` (existing tenant/logger/field helpers there); fresh tenant per test.

| test func | scenario | assertion |
|---|---|---|
| `TestExit_LastCharacterClearsEnvironment` | `Enter` character 1 into `f`; `environment.NewProcessor(l,ctx).Set(f, ObjectKindObstacle, "a", 1)`; `Exit` character 1 | `environment.NewProcessor(l,ctx).GetAll(f)` returns `len == 0` |
| `TestExit_RemainingCharacterKeepsEnvironment` | `Enter` characters 1 and 2 into `f`; `Set(f, ObjectKindObstacle, "a", 1)`; `Exit` character 1 | `GetAll(f)` still returns `len == 1` with `State == 1` |
| `TestExit_OtherFieldUnaffected` | two fields `f1` (map 910010000) and `f2` (map 910010100); `Enter` char 1 into both; `Set` one entry on each; `Exit` char 1 from `f1` only | `GetAll(f1)` is `len == 0`; `GetAll(f2)` is `len == 1` |
| `TestExit_NoEnvironmentTrackedIsNoop` | `Enter` then `Exit` character 1 with nothing tracked | no panic; `GetAll(f)` returns `len == 0` |

In package `character` (`map/character`), for the signature change:

| test func | scenario | assertion |
|---|---|---|
| `TestExitAllReturnsAffectedKeys` | `Enter` character 7 into `f1` and `f2` under tenant A; a different character 8 into `f3`; call `ExitAll(7)` | returned `[]MapKey` has `len == 2` and contains exactly the keys for `f1` and `f2` (order not asserted); it does **not** contain `f3`'s key |
| `TestExitAllIgnoresOtherTenants` | character 7 in `f1` under tenant A and under tenant B; call `ExitAll(7)` with tenant A's context | returned keys all have `Tenant` equal to tenant A; `len == 1` |

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-maps/atlas.com/maps && go test ./map/... -run 'TestExit_|TestExitAll' -v`
Expected: FAIL — the `Exit` tests fail because environment state survives; the `ExitAll` tests fail to compile (`ExitAll(7) used as value`).

- [ ] **Step 3: Write minimal implementation**

`map/character/registry.go` — `RemoveCharacterFromAllMaps` collects and returns the keys it actually touched:

```go
// RemoveCharacterFromAllMaps removes characterId from every map key that
// belongs to tenant t, and returns the keys the character was actually
// removed from so the caller can run per-field teardown.
func (r *Registry) RemoveCharacterFromAllMaps(t tenant.Model, characterId uint32) []MapKey {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	affected := make([]MapKey, 0)
	for mk := range r.characterRegister {
		if mk.Tenant != t {
			continue
		}
		before := len(r.characterRegister[mk])
		r.characterRegister[mk] = removeIfExists(r.characterRegister[mk], characterId)
		if len(r.characterRegister[mk]) != before {
			affected = append(affected, mk)
		}
	}
	return affected
}
```

`map/character/processor.go` — change the interface line `ExitAll(characterId uint32)` to `ExitAll(characterId uint32) []MapKey` and the impl to `return getRegistry().RemoveCharacterFromAllMaps(t, characterId)`.

`map/processor.go` — inside `Exit`, after `p.cp.Exit(...)` and before the `mb.Put(...)`. Import `"atlas-maps/map/environment"`:

```go
func (p *ProcessorImpl) Exit(mb *message.Buffer) func(transactionId uuid.UUID, f field.Model, characterId uint32) error {
	return func(transactionId uuid.UUID, f field.Model, characterId uint32) error {
		p.cp.Exit(transactionId, f, characterId)
		// When the field empties, drop its tracked environment state. This is a
		// registry clear only -- no ENVIRONMENT_RESET is emitted, because
		// nobody is left in the field to receive it and the next entrant
		// replays an empty state anyway. Exit is the single funnel for logout,
		// warp (TransitionMap), and channel change (TransitionChannel).
		if remaining, err := p.cp.GetCharactersInMap(transactionId, f); err == nil && len(remaining) == 0 {
			environment.NewProcessor(p.l, p.ctx).Reset(f)
		}
		return mb.Put(mapKafka.EnvEventTopicMapStatus, exitMapProvider(transactionId, f, characterId))
	}
}
```

(If `ProcessorImpl` in `map/processor.go` does not already carry `l`/`ctx` fields, read the struct and use whatever it does carry — `NewProcessor(l, ctx, p, db)` implies both are present.)

`kafka/consumer/character/consumer.go` at line 194 — capture the returned keys and clear each field. Import `"atlas-maps/map/environment"`:

```go
			affected := mapcharacter.NewProcessor(fl, ctx).ExitAll(event.CharacterId)
			fl.Debugf("Removed character [%d] from [%d] in-memory map registry entries.", event.CharacterId, len(affected))

			ep := environment.NewProcessor(fl, ctx)
			cp := mapcharacter.NewProcessor(fl, ctx)
			for _, mk := range affected {
				if remaining, err := cp.GetCharactersInMap(uuid.New(), mk.Field); err == nil && len(remaining) == 0 {
					ep.Reset(mk.Field)
				}
			}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-maps/atlas.com/maps && go build ./... && go test ./... `
Expected: PASS. (Whole-module run, because the `ExitAll` signature change can break an unrelated compile.)

- [ ] **Step 5: Commit**

```bash
git add services/atlas-maps/atlas.com/maps/map services/atlas-maps/atlas.com/maps/kafka/consumer/character/consumer.go
git commit -m "feat(atlas-maps): clear environment state when a field empties"
```

---

## Task 7: `atlas-saga-orchestrator` map command contract and producers

### Files

- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/map/kafka.go` — add two command type constants and two body structs (this service keeps its own mirror of the map message types)
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/map_command/producer.go` — two new providers
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/map_command/processor.go` — two new interface methods + impls
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/map_command/producer_test.go` — new test cases

Module root: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`
Patterns to copy: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/map_command/producer.go:14-30` (`WeatherStartCommandProvider`) and `map_command/processor.go:34-36` (`FieldEffectWeather` impl — a one-line `producer.ProviderImpl(p.l)(p.ctx)(mapKafka.EnvCommandTopicMap)(...)`).

The constants and body structs must match Task 4's `atlas-maps` definitions **byte for byte on the wire**: `"SET_ENVIRONMENT_STATE"`, `"RESET_ENVIRONMENT"`, and JSON keys `kind` / `name` / `state`.

**Interfaces:**
- Consumes: `field.ObjectKind` (Task 1).
- Produces, for Task 8:
  ```go
  SetEnvironmentState(transactionId uuid.UUID, f field.Model, kind field.ObjectKind, name string, state uint32) error
  ResetEnvironment(transactionId uuid.UUID, f field.Model) error
  ```

- [ ] **Step 1: Write the failing test**

Append to `map_command/producer_test.go`. Setup copied from the existing `TestPlayJukeboxCommandProvider` at `producer_test.go:1-40` — no tenant context, no logger; build `f := field.NewBuilder(0, 1, 910010000).SetInstance(uuid.Nil).Build()`, call the provider, assert on `msgs[0]`.

`TestSetEnvironmentStateCommandProvider` — call with `transactionId := uuid.New()`, `f`, `field.ObjectKindObstacle`, `"obs3"`, `uint32(2)`. Assert:
- `len(msgs) == 1`
- `string(msgs[0].Key) == string(producer.CreateKey(int(f.MapId())))`
- unmarshalling `msgs[0].Value` into `mapKafka.Command[mapKafka.SetEnvironmentStateCommandBody]` gives `TransactionId == transactionId`, `WorldId == 0`, `ChannelId == 1`, `MapId == 910010000`, `Instance == uuid.Nil`, `Type == "SET_ENVIRONMENT_STATE"`, `Body.Kind == "OBSTACLE"`, `Body.Name == "obs3"`, `Body.State == uint32(2)`

`TestResetEnvironmentCommandProvider` — call with `transactionId`, `f`. Assert the same envelope fields and `Type == "RESET_ENVIRONMENT"`.

`TestSetEnvironmentStateCommandProvider_EnvironmentKind` — same but `field.ObjectKindEnvironment`; assert `Body.Kind == "ENVIRONMENT"`.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test ./map_command/... -v`
Expected: FAIL — `undefined: SetEnvironmentStateCommandProvider`.

- [ ] **Step 3: Write minimal implementation**

`kafka/message/map/kafka.go` — this service keeps its map command types in `kafka.go`, not in a separate `command.go`; add to the existing const block and append the two bodies (identical text to Task 4's `atlas-maps` copy):

```go
	CommandTypeSetEnvironmentState = "SET_ENVIRONMENT_STATE"

	CommandTypeResetEnvironment = "RESET_ENVIRONMENT"
```
```go
type SetEnvironmentStateCommandBody struct {
	Kind  string `json:"kind"`
	Name  string `json:"name"`
	State uint32 `json:"state"`
}

type ResetEnvironmentCommandBody struct{}
```

`map_command/producer.go` — append (import `"github.com/Chronicle20/atlas/libs/atlas-constants/field"` is already present for `field.Model`):

```go
func SetEnvironmentStateCommandProvider(transactionId uuid.UUID, f field.Model, kind field.ObjectKind, name string, state uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(f.MapId()))
	value := &mapKafka.Command[mapKafka.SetEnvironmentStateCommandBody]{
		TransactionId: transactionId,
		WorldId:       f.WorldId(),
		ChannelId:     f.ChannelId(),
		MapId:         f.MapId(),
		Instance:      f.Instance(),
		Type:          mapKafka.CommandTypeSetEnvironmentState,
		Body: mapKafka.SetEnvironmentStateCommandBody{
			Kind:  string(kind),
			Name:  name,
			State: state,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func ResetEnvironmentCommandProvider(transactionId uuid.UUID, f field.Model) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(f.MapId()))
	value := &mapKafka.Command[mapKafka.ResetEnvironmentCommandBody]{
		TransactionId: transactionId,
		WorldId:       f.WorldId(),
		ChannelId:     f.ChannelId(),
		MapId:         f.MapId(),
		Instance:      f.Instance(),
		Type:          mapKafka.CommandTypeResetEnvironment,
		Body:          mapKafka.ResetEnvironmentCommandBody{},
	}
	return producer.SingleMessageProvider(key, value)
}
```

`map_command/processor.go` — add both to the `Processor` interface and implement:

```go
	SetEnvironmentState(transactionId uuid.UUID, f field.Model, kind field.ObjectKind, name string, state uint32) error
	ResetEnvironment(transactionId uuid.UUID, f field.Model) error
```
```go
func (p *ProcessorImpl) SetEnvironmentState(transactionId uuid.UUID, f field.Model, kind field.ObjectKind, name string, state uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(mapKafka.EnvCommandTopicMap)(SetEnvironmentStateCommandProvider(transactionId, f, kind, name, state))
}

func (p *ProcessorImpl) ResetEnvironment(transactionId uuid.UUID, f field.Model) error {
	return producer.ProviderImpl(p.l)(p.ctx)(mapKafka.EnvCommandTopicMap)(ResetEnvironmentCommandProvider(transactionId, f))
}
```

If a mock of `map_command.Processor` exists anywhere under `services/atlas-saga-orchestrator/`, add the two methods there too — find it with `grep -rn "map_command.Processor" services/atlas-saga-orchestrator/`.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go build ./... && go test ./map_command/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/map services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/map_command
git commit -m "feat(saga-orchestrator): add environment map command providers"
```

---

## Task 8: `atlas-saga-orchestrator` saga wiring and handlers

### Files

- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/model.go` — two action re-exports (after `PlayJukebox`, `model.go:255`), two payload re-exports (after `PlayJukeboxPayload`, `model.go:395`), two `case` arms in the payload unmarshaller (after the `PlayJukebox` arm, `model.go:1719-1724`)
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/event_acceptance.go` — two entries in the fire-and-forget set (after `sharedsaga.PlayJukebox: {}`, `event_acceptance.go:325`)
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go` — two interface methods (`handler.go:190`), two dispatch arms (`handler.go:1030`), two handler bodies (after `handlePlayJukebox`)
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler_test.go` — new test cases
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/model_test.go` — new round-trip test cases

Module root: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`
Patterns to copy: `saga/handler.go:3715-3742` (`handleFieldEffectWeather` — type-assert, log with `logrus.Fields`, `field.NewBuilder`, delegate to `h.mapCommandP`, `logActionError` + `return err` on failure, `StepCompleted(s.TransactionId(), true)` on success); `saga/handler_test.go:1683-1702` (`TestHandlePlayJukebox_InvalidPayload`, which uses `setupContext()` at `saga/processor_test.go:30-35`).

**Interfaces:**
- Consumes: `saga.MoveEnvironment` / `saga.ResetEnvironment` / the two payloads (Task 2); `map_command.Processor.SetEnvironmentState` / `.ResetEnvironment` (Task 7).
- Produces: nothing consumed by later tasks.

Both actions are fire-and-forget: they belong in the `{}`-valued (no correlating Kafka event) section of `eventAcceptance`, and neither has a compensating action.

- [ ] **Step 1: Write the failing test**

In `saga/handler_test.go`, setup via `setupContext()` (`saga/processor_test.go:30-35`).

| test func | scenario | assertion |
|---|---|---|
| `TestHandleMoveEnvironment_InvalidPayload` | step whose `Payload()` is a `PlayJukeboxPayload`, action `MoveEnvironment` | returns non-nil error with message exactly `invalid payload`; the `map_command` mock's `SetEnvironmentState` was **never** called |
| `TestHandleResetEnvironment_InvalidPayload` | step whose `Payload()` is a `MoveEnvironmentPayload`, action `ResetEnvironment` | returns non-nil error `invalid payload`; mock's `ResetEnvironment` never called |
| `TestHandleMoveEnvironment_DelegatesToMapCommand` | valid `MoveEnvironmentPayload{WorldId:0, ChannelId:1, MapId:910010000, Instance:uuid.Nil, Kind:field.ObjectKindObstacle, Name:"obs3", State:2}` | mock's `SetEnvironmentState` called once, with `f.WorldId()==0`, `f.ChannelId()==1`, `f.MapId()==910010000`, `f.Instance()==uuid.Nil`, `kind==field.ObjectKindObstacle`, `name=="obs3"`, `state==uint32(2)`; handler returns nil |
| `TestHandleResetEnvironment_DelegatesToMapCommand` | valid `ResetEnvironmentPayload{0,1,910010000,uuid.Nil}` | mock's `ResetEnvironment` called once with the matching field; handler returns nil |

Use whatever `map_command.Processor` test double `TestHandlePlayJukebox_*` already uses (a func-field mock assigned to `h.mapCommandP`); extend it with the two new methods.

In `saga/model_test.go`, following the existing action round-trip cases:

`TestUnmarshalStep_MoveEnvironment` — unmarshal a `Step[any]` JSON with `"action":"move_environment"` and the payload from Task 2's test; assert `s.Action() == MoveEnvironment` and `s.Payload()` type-asserts to `MoveEnvironmentPayload` with `Kind == field.ObjectKindObstacle`, `Name == "obs3"`, `State == uint32(2)`.

`TestUnmarshalStep_ResetEnvironment` — same with `"action":"reset_environment"`; assert the payload type-asserts to `ResetEnvironmentPayload` with `MapId == 910010000`.

`TestEventAcceptance_EnvironmentActionsAreFireAndForget` — assert `eventAcceptance[sharedsaga.MoveEnvironment]` is present with `len == 0`, and the same for `sharedsaga.ResetEnvironment`. (Match the map/variable name the file actually uses at `event_acceptance.go:321`.)

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test ./saga/... -run 'Environment' -v`
Expected: FAIL — `undefined: MoveEnvironment`, `h.mapCommandP.SetEnvironmentState undefined`.

- [ ] **Step 3: Write minimal implementation**

`saga/model.go` — after the `PlayJukebox` re-export block:

```go
	// Environment object actions
	MoveEnvironment  = sharedsaga.MoveEnvironment
	ResetEnvironment = sharedsaga.ResetEnvironment
```
after `PlayJukeboxPayload`:
```go
	MoveEnvironmentPayload              = sharedsaga.MoveEnvironmentPayload
	ResetEnvironmentPayload             = sharedsaga.ResetEnvironmentPayload
```
after the `case PlayJukebox:` arm in the payload unmarshaller:
```go
	case MoveEnvironment:
		var payload MoveEnvironmentPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case ResetEnvironment:
		var payload ResetEnvironmentPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
```

`saga/event_acceptance.go` — after `sharedsaga.PlayJukebox: {},`:
```go
	sharedsaga.MoveEnvironment:            {},
	sharedsaga.ResetEnvironment:           {},
```
(align the `{}` column with the surrounding entries; `gofmt` will settle it)

`saga/handler.go` — interface, after `handlePlayJukebox`:
```go
	handleMoveEnvironment(s Saga, st Step[any]) error
	handleResetEnvironment(s Saga, st Step[any]) error
```
dispatch, after `case PlayJukebox:`:
```go
	case MoveEnvironment:
		return h.handleMoveEnvironment, true
	case ResetEnvironment:
		return h.handleResetEnvironment, true
```
bodies, after `handlePlayJukebox`:
```go
// handleMoveEnvironment handles the MoveEnvironment action.
// Produces a SET_ENVIRONMENT_STATE command to COMMAND_TOPIC_MAP.
// Fire-and-forget: the step completes when the command is produced, and there
// is no compensating action -- reversing a move is the script author's job.
func (h *HandlerImpl) handleMoveEnvironment(s Saga, st Step[any]) error {
	payload, ok := st.Payload().(MoveEnvironmentPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	h.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId().String(),
		"map_id":         payload.MapId,
		"kind":           string(payload.Kind),
		"name":           payload.Name,
		"state":          payload.State,
	}).Debug("Moving field environment object")

	f := field.NewBuilder(payload.WorldId, payload.ChannelId, payload.MapId).
		SetInstance(payload.Instance).
		Build()

	err := h.mapCommandP.SetEnvironmentState(s.TransactionId(), f, payload.Kind, payload.Name, payload.State)
	if err != nil {
		h.logActionError(s, st, err, "Unable to move field environment object.")
		return err
	}

	_ = NewProcessor(h.l, h.ctx).StepCompleted(s.TransactionId(), true)
	return nil
}

// handleResetEnvironment handles the ResetEnvironment action.
// Produces a RESET_ENVIRONMENT command to COMMAND_TOPIC_MAP.
func (h *HandlerImpl) handleResetEnvironment(s Saga, st Step[any]) error {
	payload, ok := st.Payload().(ResetEnvironmentPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	h.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId().String(),
		"map_id":         payload.MapId,
	}).Debug("Resetting field environment objects")

	f := field.NewBuilder(payload.WorldId, payload.ChannelId, payload.MapId).
		SetInstance(payload.Instance).
		Build()

	err := h.mapCommandP.ResetEnvironment(s.TransactionId(), f)
	if err != nil {
		h.logActionError(s, st, err, "Unable to reset field environment objects.")
		return err
	}

	_ = NewProcessor(h.l, h.ctx).StepCompleted(s.TransactionId(), true)
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go build ./... && go test ./saga/... ./map_command/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga
git commit -m "feat(saga-orchestrator): handle move_environment and reset_environment actions"
```

---

## Task 9: `atlas-channel` environment REST client

### Files

- `services/atlas-channel/atlas.com/channel/environment/rest.go` — **new file**
- `services/atlas-channel/atlas.com/channel/environment/requests.go` — **new file**
- `services/atlas-channel/atlas.com/channel/environment/processor.go` — **new file**
- `services/atlas-channel/atlas.com/channel/environment/mock/processor.go` — **new file**
- `services/atlas-channel/atlas.com/channel/environment/requests_test.go` — **new file**

Module root: `services/atlas-channel/atlas.com/channel`
Patterns to copy: `services/atlas-channel/atlas.com/channel/weather/requests.go:1-26` (the `mapInstanceResource` constant, `requests.RootUrlFor(ctx, "MAPS")`, `requests.GetRequest`); `services/atlas-channel/atlas.com/channel/weather/processor.go:1-33` (interface, `ProcessorImpl`, `Extract`); `services/atlas-channel/atlas.com/channel/weather/mock/processor.go:1-20` (func-field mock with `var _ environment.Processor = (*ProcessorMock)(nil)`); `services/atlas-channel/atlas.com/channel/weather/requests_test.go` (the HTTP-stub test shape).

The only structural difference from `channel/weather` is that this returns a **slice**, so it uses `requests.SliceProvider` (`libs/atlas-rest/requests/provider.go:23`) rather than `requests.Provider`. Its signature is `SliceProvider[A, M any](l, ctx)(r Request[[]A], t model.Transformer[A, M], filters []model.Filter[M]) model.Provider[[]M]` — three arguments, with `nil` for filters.

**Interfaces:**
- Consumes: the Task 5 REST route.
- Produces, for Task 11:
  ```go
  package environment
  type RestModel struct { Id string; Kind string; Name string; State uint32 }
  type Processor interface { GetAll(f field.Model) ([]RestModel, error) }
  func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor
  func Extract(m RestModel) (RestModel, error)
  ```

- [ ] **Step 1: Write the failing test**

`requests_test.go`, package `environment`. Setup shape copied from `services/atlas-channel/atlas.com/channel/weather/requests_test.go`: stand up an `httptest.NewServer`, point the `"MAPS"` service root at it the same way that file does, build a tenant context with `tenant.Create(uuid.New(), "GMS", 83, 1)` + `tenant.WithContext`, and `test.NewNullLogger()`.

| test func | server response | assertion |
|---|---|---|
| `TestGetAll_ParsesCollection` | `200` with a JSON:API document whose `data` is two `environment-objects` resources: `{"id":"OBSTACLE:obs3","attributes":{"kind":"OBSTACLE","name":"obs3","state":2}}` then `{"id":"ENVIRONMENT:gate01","attributes":{"kind":"ENVIRONMENT","name":"gate01","state":1}}` | returns nil error and a slice of length 2; `[0].Kind=="OBSTACLE"`, `[0].Name=="obs3"`, `[0].State==uint32(2)`; `[1].Kind=="ENVIRONMENT"`, `[1].Name=="gate01"`, `[1].State==uint32(1)` — **order preserved** |
| `TestGetAll_EmptyCollection` | `200` with `{"data":[]}` | returns nil error and a slice of length 0 |
| `TestGetAll_ServerError` | `500` | returns a non-nil error |
| `TestGetAll_RequestsInstancePath` | any `200`; capture `r.URL.Path` on the server | the captured path ends with `/worlds/0/channels/1/maps/910010000/instances/00000000-0000-0000-0000-000000000000/environment` |

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./environment/... -v`
Expected: FAIL — the `environment` package does not exist.

- [ ] **Step 3: Write minimal implementation**

`rest.go`:

```go
package environment

type RestModel struct {
	Id    string `json:"-"`
	Kind  string `json:"kind"`
	Name  string `json:"name"`
	State uint32 `json:"state"`
}

func (m RestModel) GetID() string {
	return m.Id
}

func (m RestModel) GetName() string {
	return "environment-objects"
}

func (m *RestModel) SetID(idStr string) error {
	m.Id = idStr
	return nil
}
```

`requests.go`:

```go
package environment

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	mapInstanceResource            = "worlds/%d/channels/%d/maps/%d/instances/%s"
	mapInstanceEnvironmentResource = mapInstanceResource + "/environment"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "MAPS")
}

func requestEnvironmentInMap(ctx context.Context, f field.Model) requests.Request[[]RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[[]RestModel](err)
	}
	return requests.MakeGetRequest[[]RestModel](fmt.Sprintf(root+mapInstanceEnvironmentResource, f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String()))
}
```

**Verify the collection-request constructor name before writing it:** `requests.GetRequest[M]` returns a `Request[M]`, and `SliceProvider` needs a `Request[[]A]`. Run `grep -rn "SliceProvider" services/ libs/ --include="*.go" | grep -v _test` and copy the request constructor an existing `SliceProvider` caller uses. Use that name, not the placeholder above.

`processor.go`:

```go
package environment

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

type Processor interface {
	GetAll(f field.Model) ([]RestModel, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) GetAll(f field.Model) ([]RestModel, error) {
	return requests.SliceProvider[RestModel, RestModel](p.l, p.ctx)(requestEnvironmentInMap(p.ctx, f), Extract, nil)()
}

func Extract(m RestModel) (RestModel, error) {
	return m, nil
}
```

`mock/processor.go`:

```go
package mock

import (
	"atlas-channel/environment"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

type ProcessorMock struct {
	GetAllFunc func(f field.Model) ([]environment.RestModel, error)
}

var _ environment.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) GetAll(f field.Model) ([]environment.RestModel, error) {
	if m.GetAllFunc != nil {
		return m.GetAllFunc(f)
	}
	return []environment.RestModel{}, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./environment/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/environment
git commit -m "feat(atlas-channel): add environment REST client for atlas-maps"
```

---

## Task 10: `atlas-channel` writer-fallback helper and status-event broadcast

### Files

- `services/atlas-channel/atlas.com/channel/kafka/message/map/kafka.go` — mirror Task 4's two status constants and three structs (this service keeps its own copy of the map event types)
- `services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go` — `announceObjectState` helper, two status-event handlers, two registrations in `InitHandlers`
- `services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer_test.go` — new test cases

Module root: `services/atlas-channel/atlas.com/channel`
Patterns to copy: `services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go:989-1010` (`handleStatusEventWeatherStart`: type guard, `sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId)` guard, `field.NewBuilder`, `_map.NewProcessor(l, ctx).ForSessionsInMap(f, ...)`, `Errorf` on broadcast failure); `consumer.go:98-102` (`InitHandlers` registration, appending a `listener.HandlerHandle`); `consumer.go:769-779` (`doorAnnounce`, the package-level `var` announce seam that tests stub).

**Interfaces:**
- Consumes: `field.ObjectKind`/`ParseObjectKind` (Task 1); the writer constants `fieldcb.SetObjectStateWriter`, `fieldcb.FieldObstacleOnOffWriter`, `fieldcb.FieldObstacleAllResetWriter` and the wrappers `writer.SetObjectStateBody`, `writer.FieldObstacleOnOffBody`, `writer.FieldObstacleAllResetBody`.
- Produces, for Task 11:
  ```go
  func announceObjectState(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, kind field.ObjectKind, name string, state uint32, s session.Model) error
  ```

`announceObjectState` owns the §1.3 fallback so no call site can forget it. It resolves the kind-preferred writer name, probes `wp(writerName)`, and on error retries with `fieldcb.SetObjectStateWriter`. It emits through the existing `doorAnnounce` seam so tests can capture the writer name without a socket. Because `SetObjectState` is routed on all nine templates, the second probe failing is a genuine misconfiguration and logs at `Error`.

`handleStatusEventEnvironmentReset` sends `FieldObstacleAllReset` when routed (silently skipped on `gms_48_1` / `gms_61_1`), and then `announceObjectState(kind, name, 0)` for **every** entry in `e.Body.Cleared`, in the order the event carries. Step two is what restores non-obstacle named objects and what makes reset correct on the two templates that route no `FieldObstacleAllReset`.

- [ ] **Step 1: Write the failing test**

Append to `services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer_test.go`, package `_map`. Setup: `newTestCtx(t)` (`consumer_test.go:32-38`), `newTestField()` (`consumer_test.go:45-47`), `addFieldSession(t, ctx, l, 0, f)` (`consumer_test.go:56-66`). Stub `doorAnnounce` with the save/restore pattern at `consumer_test.go:254-262` to capture an ordered `[]string` of writer names. Build the `writer.Producer` stub directly — `writer.Producer` is a type alias for `func(name string) (writer.BodyFunc, error)`:

```go
routed := func(names ...string) writer.Producer {
    set := make(map[string]struct{}, len(names))
    for _, n := range names { set[n] = struct{}{} }
    return func(name string) (writer.BodyFunc, error) {
        if _, ok := set[name]; ok { return nil, nil }
        return nil, errors.New("writer not found")
    }
}
```

`TestAnnounceObjectState_WriterSelection` — table-driven, one `t.Run` per row, calling `announceObjectState(l, ctx, wp, kind, "obj", 1, session.Model{})` and asserting the single captured writer name:

| subtest name | kind | routed writers | expect writer name |
|---|---|---|---|
| `obstacle with obstacle writer routed` | `ObjectKindObstacle` | `SetObjectStateWriter`, `FieldObstacleOnOffWriter` | `fieldcb.FieldObstacleOnOffWriter` |
| `obstacle with obstacle writer unrouted falls back` | `ObjectKindObstacle` | `SetObjectStateWriter` only | `fieldcb.SetObjectStateWriter` |
| `environment always uses set object state` | `ObjectKindEnvironment` | `SetObjectStateWriter`, `FieldObstacleOnOffWriter` | `fieldcb.SetObjectStateWriter` |
| `environment with only obstacle writer routed still uses set object state` | `ObjectKindEnvironment` | `FieldObstacleOnOffWriter` only | none captured; the call returns a non-nil error |

`TestHandleStatusEventEnvironmentStateChanged_Broadcasts` — dispatch `_map3.StatusEvent[_map3.EnvironmentStateChanged]{Type: "ENVIRONMENT_STATE_CHANGED", WorldId:0, ChannelId:0, MapId:100000000, Instance: uuid.Nil, Body:{Kind:"OBSTACLE", Name:"obs3", State:2}}` with one session in the field and `FieldObstacleOnOffWriter` routed. Assert exactly one captured writer name, equal to `fieldcb.FieldObstacleOnOffWriter`.

`TestHandleStatusEventEnvironmentStateChanged_WrongTypeIgnored` — same event with `Type: "WEATHER_START"`. Assert zero captured writer names.

`TestHandleStatusEventEnvironmentStateChanged_BadKindIgnored` — `Body.Kind == "GATE"`. Assert zero captured writer names.

`TestHandleStatusEventEnvironmentReset_AllResetRouted` — event `Type: "ENVIRONMENT_RESET"`, `Body.Cleared == [{Kind:"OBSTACLE",Name:"a"},{Kind:"ENVIRONMENT",Name:"b"}]`, routed = `SetObjectStateWriter`, `FieldObstacleOnOffWriter`, `FieldObstacleAllResetWriter`. Assert the captured names are **exactly**, in order:
`[FieldObstacleAllResetWriter, FieldObstacleOnOffWriter, SetObjectStateWriter]`.

`TestHandleStatusEventEnvironmentReset_AllResetUnrouted` — same event, routed = `SetObjectStateWriter` only. Assert the captured names are exactly, in order: `[SetObjectStateWriter, SetObjectStateWriter]` — the `AllReset` is skipped and both cleared objects are zeroed via the fallback.

`TestHandleStatusEventEnvironmentReset_EmptyCleared` — `Body.Cleared` is an empty slice, routed includes `FieldObstacleAllResetWriter`. Assert exactly one captured name, `fieldcb.FieldObstacleAllResetWriter`.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./kafka/consumer/map/... -run 'Environment|AnnounceObjectState' -v`
Expected: FAIL — `undefined: announceObjectState`, `undefined: _map3.EnvironmentStateChanged`.

- [ ] **Step 3: Write minimal implementation**

`kafka/message/map/kafka.go` — add to the const block and append the three structs, copying Task 4's `atlas-maps` text exactly (same constants, same JSON tags):

```go
	EventTopicMapStatusTypeEnvironmentStateChanged = "ENVIRONMENT_STATE_CHANGED"
	EventTopicMapStatusTypeEnvironmentReset        = "ENVIRONMENT_RESET"
```
```go
type EnvironmentStateChanged struct {
	Kind  string `json:"kind"`
	Name  string `json:"name"`
	State uint32 `json:"state"`
}

type EnvironmentObject struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}

type EnvironmentReset struct {
	Cleared []EnvironmentObject `json:"cleared"`
}
```

`kafka/consumer/map/consumer.go` — the helper:

```go
// announceObjectState sends one (name, state) to s using the writer preferred
// for kind, falling back to SetObjectState when the tenant routes no writer for
// that opcode. The fallback is behaviourally identical: CField::OnSetObjectState
// and CField::OnFieldObstacleOnOff are the same client function
// (CMapLoadable::SetObjectState), differing only in opcode -- see design.md
// section 1.1. gms_48_1 routes no obstacle writer at all and gms_61_1 routes
// neither FieldObstacleOnOff nor FieldObstacleAllReset, so this is a branch on a
// real "writer not found" error rather than a version sniff.
func announceObjectState(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, kind fieldconst.ObjectKind, name string, state uint32, s session.Model) error {
	if kind == fieldconst.ObjectKindObstacle {
		if _, err := wp(fieldcb.FieldObstacleOnOffWriter); err == nil {
			return doorAnnounce(l, ctx, wp, fieldcb.FieldObstacleOnOffWriter, writer.FieldObstacleOnOffBody(name, state), s)
		}
	}
	if _, err := wp(fieldcb.SetObjectStateWriter); err != nil {
		l.WithError(err).Errorf("Tenant routes no [%s] writer; unable to set object [%s] state.", fieldcb.SetObjectStateWriter, name)
		return err
	}
	return doorAnnounce(l, ctx, wp, fieldcb.SetObjectStateWriter, writer.SetObjectStateBody(name, state), s)
}
```

(`fieldconst` is the import alias for `github.com/Chronicle20/atlas/libs/atlas-constants/field`; the file already imports that package as `field` for `field.NewBuilder`, so reuse `field` and drop the alias if there is no clash.)

The two handlers, following `handleStatusEventWeatherStart`:

```go
func handleStatusEventEnvironmentStateChanged(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, ctx context.Context, event _map3.StatusEvent[_map3.EnvironmentStateChanged]) {
	return func(l logrus.FieldLogger, ctx context.Context, e _map3.StatusEvent[_map3.EnvironmentStateChanged]) {
		if e.Type != _map3.EventTopicMapStatusTypeEnvironmentStateChanged {
			return
		}
		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}

		kind, err := field.ParseObjectKind(e.Body.Kind)
		if err != nil {
			l.WithError(err).Errorf("Unable to broadcast environment state change in map [%d] instance [%s].", e.MapId, e.Instance)
			return
		}

		l.Debugf("Environment object [%s] kind [%s] set to state [%d] in map [%d] instance [%s].", e.Body.Name, e.Body.Kind, e.Body.State, e.MapId, e.Instance)
		f := field.NewBuilder(e.WorldId, e.ChannelId, e.MapId).SetInstance(e.Instance).Build()
		err = _map.NewProcessor(l, ctx).ForSessionsInMap(f, func(s session.Model) error {
			return announceObjectState(l, ctx, wp, kind, e.Body.Name, e.Body.State, s)
		})
		if err != nil {
			l.WithError(err).Errorf("Unable to broadcast environment state change to map [%d] instance [%s].", e.MapId, e.Instance)
		}
	}
}

func handleStatusEventEnvironmentReset(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, ctx context.Context, event _map3.StatusEvent[_map3.EnvironmentReset]) {
	return func(l logrus.FieldLogger, ctx context.Context, e _map3.StatusEvent[_map3.EnvironmentReset]) {
		if e.Type != _map3.EventTopicMapStatusTypeEnvironmentReset {
			return
		}
		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}

		l.Debugf("Environment reset in map [%d] instance [%s]; restoring [%d] tracked object(s).", e.MapId, e.Instance, len(e.Body.Cleared))
		f := field.NewBuilder(e.WorldId, e.ChannelId, e.MapId).SetInstance(e.Instance).Build()
		err := _map.NewProcessor(l, ctx).ForSessionsInMap(f, func(s session.Model) error {
			// FieldObstacleAllReset restores only the client's own obstacle
			// list, and gms_48_1 / gms_61_1 route no such writer at all.
			if _, werr := wp(fieldcb.FieldObstacleAllResetWriter); werr == nil {
				_ = doorAnnounce(l, ctx, wp, fieldcb.FieldObstacleAllResetWriter, writer.FieldObstacleAllResetBody(), s)
			}
			// Explicitly zero every tracked object: this is the only thing that
			// restores non-obstacle named objects -- see design.md section 1.2.
			for _, o := range e.Body.Cleared {
				kind, kerr := field.ParseObjectKind(o.Kind)
				if kerr != nil {
					l.WithError(kerr).Errorf("Skipping cleared object [%s] in map [%d] instance [%s].", o.Name, e.MapId, e.Instance)
					continue
				}
				_ = announceObjectState(l, ctx, wp, kind, o.Name, 0, s)
			}
			return nil
		})
		if err != nil {
			l.WithError(err).Errorf("Unable to broadcast environment reset to map [%d] instance [%s].", e.MapId, e.Instance)
		}
	}
}
```

`InitHandlers` — register both after the `handleStatusEventJukeboxEnd` registration (`consumer.go:118-123`), using the identical `id, err = rf(...)` / `handles = append(...)` shape.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./kafka/consumer/map/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka
git commit -m "feat(atlas-channel): broadcast environment state changes with writer fallback"
```

---

## Task 11: `atlas-channel` enter replay

### Files

- `services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go` — `announceEnvironmentState` and one new `routine.Go` block in `SpawnForSelf`
- `services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer_test.go` — new test cases

Module root: `services/atlas-channel/atlas.com/channel`
Patterns to copy: `services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go:1156-1162` (`announceActiveJukebox` — fetch, return silently on error, `doorAnnounce`) and `consumer.go:377-383` (the `routine.Go` blocks in `SpawnForSelf`).

The replay site is `SpawnForSelf`, not `handleStatusEventCharacterEnter`: `SpawnForSelf` is the function that already owns "send the entering session everything it needs to render the field", it is called synchronously after `SetField`, and each of its blocks already returns silently on fetch failure. Because the block is inside `routine.Go`, a REST failure cannot block map entry.

**Interfaces:**
- Consumes: `environment.Processor.GetAll` (Task 9); `announceObjectState` (Task 10); `fieldcb.FieldObstacleOnOffListWriter`, `fieldcb.NewObstacleState`, `writer.FieldObstacleOnOffListBody`.
- Produces: nothing consumed later.

Replay order (design.md §4.4): obstacles first — one `FieldObstacleOnOffList` if that writer is routed, otherwise one `announceObjectState` per obstacle — then one `SetObjectState` per environment object, in insertion order. This gives `FieldObstacleOnOffListWriter` its emitting call site, satisfying PRD acceptance criterion 1 for the fourth writer.

- [ ] **Step 1: Write the failing test**

Append to `consumer_test.go`. Same setup as Task 10 (stubbed `doorAnnounce` capturing ordered writer names, `routed(...)` writer producer). The environment fetch is stubbed by making `announceEnvironmentState` take the processor's result — write it as `announceEnvironmentState(l, ctx, wp, entries []environment.RestModel, s session.Model)` so the test supplies entries directly and the `routine.Go` block in `SpawnForSelf` does the `GetAll` call and the error handling. This keeps the pure ordering logic testable without an HTTP stub.

`TestAnnounceEnvironmentState_ObstaclesThenEnvironment` — entries in this order: `{Kind:"OBSTACLE",Name:"a",State:1}`, `{Kind:"ENVIRONMENT",Name:"b",State:2}`, `{Kind:"OBSTACLE",Name:"c",State:3}`; routed = `SetObjectStateWriter`, `FieldObstacleOnOffWriter`, `FieldObstacleOnOffListWriter`. Assert the captured writer names are exactly, in order:
`[FieldObstacleOnOffListWriter, SetObjectStateWriter]` — one list packet carrying both obstacles (`a` then `c`, insertion order preserved), then one `SetObjectState` for `b`.

`TestAnnounceEnvironmentState_ListWriterUnrouted` — same entries, routed = `SetObjectStateWriter`, `FieldObstacleOnOffWriter` (no list writer). Assert exactly, in order:
`[FieldObstacleOnOffWriter, FieldObstacleOnOffWriter, SetObjectStateWriter]` — one per obstacle, then the environment object.

`TestAnnounceEnvironmentState_NoObstacleWritersAtAll` — same entries, routed = `SetObjectStateWriter` only (the `gms_48_1` shape). Assert exactly, in order:
`[SetObjectStateWriter, SetObjectStateWriter, SetObjectStateWriter]` — `a`, `c`, then `b`.

`TestAnnounceEnvironmentState_Empty` — entries is an empty slice, all writers routed. Assert zero captured writer names (PRD FR-19).

`TestAnnounceEnvironmentState_OnlyEnvironment` — entries `[{Kind:"ENVIRONMENT",Name:"b",State:2}]`, all writers routed. Assert exactly `[SetObjectStateWriter]` — no empty `FieldObstacleOnOffList` is sent.

`TestAnnounceEnvironmentState_BadKindSkipped` — entries `[{Kind:"GATE",Name:"x",State:1},{Kind:"ENVIRONMENT",Name:"b",State:2}]`, all routed. Assert exactly `[SetObjectStateWriter]` — the unparseable entry is skipped, the valid one still goes out.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./kafka/consumer/map/... -run 'TestAnnounceEnvironmentState' -v`
Expected: FAIL — `undefined: announceEnvironmentState`.

- [ ] **Step 3: Write minimal implementation**

```go
// announceEnvironmentState replays the field's tracked object state to a single
// entering session: obstacles first as one FieldObstacleOnOffList when that
// writer is routed, then one SetObjectState per environment object, in the
// insertion order atlas-maps preserved. Replay order is observable to the
// client. gms_48_1 routes no obstacle list writer, so the obstacle branch falls
// back to one announceObjectState per obstacle.
func announceEnvironmentState(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, entries []environment.RestModel, s session.Model) {
	obstacles := make([]fieldcb.ObstacleState, 0, len(entries))
	obstacleNames := make([]environment.RestModel, 0, len(entries))
	others := make([]environment.RestModel, 0, len(entries))
	for _, e := range entries {
		kind, err := field.ParseObjectKind(e.Kind)
		if err != nil {
			l.WithError(err).Errorf("Skipping environment object [%s] on enter replay.", e.Name)
			continue
		}
		if kind == field.ObjectKindObstacle {
			obstacles = append(obstacles, fieldcb.NewObstacleState(e.Name, e.State))
			obstacleNames = append(obstacleNames, e)
			continue
		}
		others = append(others, e)
	}

	if len(obstacles) > 0 {
		if _, err := wp(fieldcb.FieldObstacleOnOffListWriter); err == nil {
			_ = doorAnnounce(l, ctx, wp, fieldcb.FieldObstacleOnOffListWriter, writer.FieldObstacleOnOffListBody(obstacles), s)
		} else {
			for _, e := range obstacleNames {
				_ = announceObjectState(l, ctx, wp, field.ObjectKindObstacle, e.Name, e.State, s)
			}
		}
	}

	for _, e := range others {
		_ = announceObjectState(l, ctx, wp, field.ObjectKindEnvironment, e.Name, e.State, s)
	}
}
```

In `SpawnForSelf`, immediately after the `announceActiveJukebox` block (currently `consumer.go:381-383`):

```go
		routine.Go(l, ctx, func(_ context.Context) {
			entries, eerr := environment.NewProcessor(l, ctx).GetAll(f)
			if eerr != nil {
				// Fails open: an unreachable atlas-maps costs the replayed
				// object state, not the map entry.
				l.WithError(eerr).Errorf("Unable to retrieve environment state for map [%d] instance [%s].", f.MapId(), f.Instance())
				return
			}
			if len(entries) == 0 {
				return
			}
			announceEnvironmentState(l, ctx, wp, entries, s)
		})
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./... `
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/consumer/map
git commit -m "feat(atlas-channel): replay environment object state on map enter"
```

---

## Task 12: `atlas-reactor-actions` script operations

### Files

- `services/atlas-reactor-actions/atlas.com/reactor/script/executor.go` — implement `executeMoveEnvironment` (currently a stub at `executor.go:278-292`), add `executeResetEnvironment`, add the `reset_environment` `case` to the switch (`executor.go:49-86`)
- `services/atlas-reactor-actions/atlas.com/reactor/script/executor_test.go` — **new file**
- `services/atlas-reactor-actions/docs/reactor_script_schema.json` — add `reset_environment` to the operation-type enum (`:107-119`), add optional `kind` to the existing `move_environment` branch (`:218-236`)
- `services/atlas-reactor-actions/docs/domain.md` — replace the `move_environment` "(not yet implemented)" line (`:108`) and add `reset_environment`

Module root: `services/atlas-reactor-actions/atlas.com/reactor`
Patterns to copy: `services/atlas-reactor-actions/atlas.com/reactor/script/executor.go:220-243` (`executeSpawnMonster` — the `saga.NewBuilder().SetSagaType(saga.InventoryTransaction).SetInitiatedBy(...).AddStep(fmt.Sprintf(...), saga.Pending, saga.<Action>, saga.<Payload>{...}).Build()` shape ending in `return e.sagaP.Create(s)`); `executor.go:308-314` (`executeDropMessage`'s `params["x"], ok :=` missing-parameter error style).

`operation.Model`'s parameter API is `Params() map[string]string` — a plain string map with no typed accessors, so every value is hand-parsed.

**Interfaces:**
- Consumes: `saga.MoveEnvironment`, `saga.ResetEnvironment`, `saga.MoveEnvironmentPayload`, `saga.ResetEnvironmentPayload` (Task 2); `field.ParseObjectKind` (Task 1).
- Produces: nothing consumed later.

Parameters (PRD FR-26): `name` required and non-blank; `value` required, parsed with `strconv.ParseUint(v, 10, 32)` — a parse failure is an error, never a silent zero; `kind` optional, through `field.ParseObjectKind` (blank → `ENVIRONMENT`, unrecognised → error). `reset_environment` takes no parameters.

`executeKillAllMonsters` and `executeWeakenAreaBoss` are **not** touched (PRD non-goal); their "not yet implemented" doc lines stay.

- [ ] **Step 1: Write the failing test**

`executor_test.go`, package `script` — **new file**, and the package has no existing test. Build the executor with `NewOperationExecutor(l, ctx)` and then replace its `sagaP` field with a capture double (the field is unexported but the test is in-package). The double implements `reactorsaga.Processor` with a `CreateFunc func(s saga.Saga) error` recording the saga it was handed. Logger: `test.NewNullLogger()`. Context: plain `context.Background()` unless `reactorsaga.NewProcessor` needs a tenant — if it does, add `tenant.Create(uuid.New(), "GMS", 83, 1)` + `tenant.WithContext`.

Build `operation.Model` values through whatever constructor `libs/atlas-script-core/operation` exports (find it with `grep -rn "func New" libs/atlas-script-core/operation/`); it needs a type string and a `map[string]string` of params.

`rc := ReactorContext{Field: field.NewBuilder(0, 1, 910010000).SetInstance(uuid.Nil).Build(), ReactorId: 1, Classification: "2001000", ReactorName: "gate"}`.

`TestExecuteMoveEnvironment` — table-driven, one `t.Run` per row, calling `e.ExecuteOperation(rc, 1234, op)`:

| subtest name | params | expect |
|---|---|---|
| `creates saga step` | `{"name":"gate01","value":"3"}` | nil error; exactly one saga created; its single step has `Action == saga.MoveEnvironment`, `Status == saga.Pending`, `StepId == "move-environment-2001000-gate01"`; payload type-asserts to `saga.MoveEnvironmentPayload` with `WorldId==0`, `ChannelId==1`, `MapId==910010000`, `Instance==uuid.Nil`, `Kind==field.ObjectKindEnvironment`, `Name=="gate01"`, `State==uint32(3)` |
| `kind obstacle reaches payload` | `{"name":"obs3","value":"1","kind":"OBSTACLE"}` | payload `Kind == field.ObjectKindObstacle` |
| `omitted kind defaults environment` | `{"name":"gate01","value":"0"}` | payload `Kind == field.ObjectKindEnvironment`, `State == uint32(0)` |
| `missing name errors` | `{"value":"3"}` | non-nil error, message exactly `move_environment operation missing name parameter`; no saga created |
| `blank name errors` | `{"name":"","value":"3"}` | same error; no saga created |
| `missing value errors` | `{"name":"gate01"}` | non-nil error, message exactly `move_environment operation missing value parameter`; no saga created |
| `non-numeric value errors` | `{"name":"gate01","value":"up"}` | non-nil error mentioning `value`; no saga created; **not** a silent `State == 0` |
| `negative value errors` | `{"name":"gate01","value":"-1"}` | non-nil error; no saga created |
| `overflow value errors` | `{"name":"gate01","value":"4294967296"}` | non-nil error; no saga created |
| `max uint32 accepted` | `{"name":"gate01","value":"4294967295"}` | nil error; payload `State == uint32(4294967295)` |
| `bad kind errors` | `{"name":"gate01","value":"3","kind":"GATE"}` | non-nil error, message exactly `unrecognized object kind [GATE]`; no saga created |

`TestExecuteResetEnvironment`:

| subtest name | params | expect |
|---|---|---|
| `creates saga step` | `{}` | nil error; one saga; single step `Action == saga.ResetEnvironment`, `Status == saga.Pending`, `StepId == "reset-environment-2001000"`; payload type-asserts to `saga.ResetEnvironmentPayload` with `WorldId==0`, `ChannelId==1`, `MapId==910010000`, `Instance==uuid.Nil` |
| `extra params ignored` | `{"name":"ignored"}` | nil error; one saga; same payload |

`TestExecuteMoveEnvironment_NoLongerReturnsNilStub` — assert that a valid `move_environment` produces **exactly one** saga create call (this is the regression guard for today's bare `return nil`).

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-reactor-actions/atlas.com/reactor && go test ./script/... -v`
Expected: FAIL — `move_environment` currently returns nil without creating a saga, so the create-count assertions fail; `reset_environment` falls through to the `default` "Unknown operation type" arm.

- [ ] **Step 3: Write minimal implementation**

`executor.go` — add `case "reset_environment": return e.executeResetEnvironment(rc, characterId, op)` to the switch after the `move_environment` case, and replace `executeMoveEnvironment` wholesale:

```go
// executeMoveEnvironment sets the state of one named field object via a saga.
// Parameters: name (required, non-blank), value (required, uint32), kind
// (optional, ENVIRONMENT or OBSTACLE; blank defaults to ENVIRONMENT).
func (e *OperationExecutor) executeMoveEnvironment(rc ReactorContext, characterId uint32, op operation.Model) error {
	params := op.Params()

	name, ok := params["name"]
	if !ok || strings.TrimSpace(name) == "" {
		return fmt.Errorf("move_environment operation missing name parameter")
	}

	valueStr, ok := params["value"]
	if !ok || strings.TrimSpace(valueStr) == "" {
		return fmt.Errorf("move_environment operation missing value parameter")
	}
	value, err := strconv.ParseUint(strings.TrimSpace(valueStr), 10, 32)
	if err != nil {
		return fmt.Errorf("move_environment operation has non-numeric value parameter [%s]: %w", valueStr, err)
	}

	kind, err := field.ParseObjectKind(params["kind"])
	if err != nil {
		return err
	}

	e.l.Debugf("Moving environment object [%s] kind [%s] to state [%d] at reactor [%s].", name, kind, value, rc.Classification)

	s := saga.NewBuilder().
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("reactor-action-move-environment").
		AddStep(
			fmt.Sprintf("move-environment-%s-%s", rc.Classification, name),
			saga.Pending,
			saga.MoveEnvironment,
			saga.MoveEnvironmentPayload{
				WorldId:   rc.Field.WorldId(),
				ChannelId: rc.Field.ChannelId(),
				MapId:     rc.Field.MapId(),
				Instance:  rc.Field.Instance(),
				Kind:      kind,
				Name:      name,
				State:     uint32(value),
			},
		).Build()

	return e.sagaP.Create(s)
}

// executeResetEnvironment clears every tracked field object and restores the
// field's objects to their default state. Takes no parameters.
func (e *OperationExecutor) executeResetEnvironment(rc ReactorContext, characterId uint32, op operation.Model) error {
	e.l.Debugf("Resetting environment objects in map [%d] at reactor [%s].", rc.Field.MapId(), rc.Classification)

	s := saga.NewBuilder().
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("reactor-action-reset-environment").
		AddStep(
			fmt.Sprintf("reset-environment-%s", rc.Classification),
			saga.Pending,
			saga.ResetEnvironment,
			saga.ResetEnvironmentPayload{
				WorldId:   rc.Field.WorldId(),
				ChannelId: rc.Field.ChannelId(),
				MapId:     rc.Field.MapId(),
				Instance:  rc.Field.Instance(),
			},
		).Build()

	return e.sagaP.Create(s)
}
```

`strings` and `strconv` are already imported by this file; `field` is already imported for `field.Model`.

`docs/reactor_script_schema.json`:
- add `"reset_environment",` to the operation-type enum after `"move_environment",`
- in the existing `move_environment` `allOf` branch, add a third property alongside `name` and `value` (leaving `"required": ["name", "value"]` unchanged):
  ```json
                  "kind": {
                    "type": "string",
                    "enum": ["ENVIRONMENT", "OBSTACLE"],
                    "description": "Which clientbound opcode carries the change; defaults to ENVIRONMENT"
                  }
  ```
- add a new `allOf` branch for `reset_environment` matching the shape of the neighbouring branches, with no `required` params.

`docs/domain.md:108` — replace the line with:
```markdown
- `move_environment`: Sets the state of a named field object via saga; params: `name`, `value` (uint32); optional: `kind` (`ENVIRONMENT` default, or `OBSTACLE`)
- `reset_environment`: Clears tracked field object state and restores default appearance via saga; takes no params
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-reactor-actions/atlas.com/reactor && go build ./... && go test ./... `
Then confirm the schema is still valid JSON: `python3 -m json.tool services/atlas-reactor-actions/docs/reactor_script_schema.json > /dev/null && echo OK` (run from the worktree root).
Then confirm the stub is gone: `grep -n "This needs a new saga action for environment manipulation" services/atlas-reactor-actions/atlas.com/reactor/script/executor.go` — must print nothing (exit 1).
Expected: tests PASS, `OK`, and no grep hit.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-reactor-actions
git commit -m "feat(atlas-reactor-actions): implement move_environment and add reset_environment"
```

---

## Task 13: `atlas-map-actions` script operations

### Files

- `services/atlas-map-actions/atlas.com/map-actions/script/executor.go` — add `executeMoveEnvironment` and `executeResetEnvironment`, plus two `case` arms in the switch (`executor.go:35-50`)
- `services/atlas-map-actions/atlas.com/map-actions/script/executor_test.go` — **new file**
- `services/atlas-map-actions/docs/map_script_schema.json` — add both to the operation-type enum (`:102-110`) and add two `allOf` branches
- `services/atlas-map-actions/docs/domain.md` — two rows in the operation table (`:67-75`)

Module root: `services/atlas-map-actions/atlas.com/map-actions`
Patterns to copy: `services/atlas-map-actions/atlas.com/map-actions/script/executor.go:63-88` (`executeFieldEffect` — same `saga.NewBuilder()...AddStep(...).Build()` / `return e.sagaP.Create(s)` shape, and the `params["path"], ok :=` missing-parameter style).

Parameter semantics are **identical** to Task 12; the only structural difference is that `ExecuteOperation` takes `f field.Model` directly rather than a `ReactorContext`, so the payload is built from `f` and the step id has no reactor classification in it.

**Design correction:** design.md §6.3 cites the switch at `executor.go:36-52`; it is actually at `executor.go:35-50`.

**Interfaces:**
- Consumes: the same Task 1 / Task 2 symbols as Task 12.
- Produces: nothing consumed later.

- [ ] **Step 1: Write the failing test**

`executor_test.go`, package `script` — **new file**. Same double-and-capture approach as Task 12: build with `NewOperationExecutor(l, ctx)`, replace the unexported `sagaP` with a `mapactionsaga.Processor` double recording each `Create(s saga.Saga)`. `f := field.NewBuilder(0, 1, 910010000).SetInstance(uuid.Nil).Build()`; calls are `e.ExecuteOperation(f, 1234, op)`.

`TestExecuteMoveEnvironment` — the **same eleven rows** as Task 12's table, with these differences in the expectations: `StepId == "move-environment-gate01"` (no classification segment), `SetInitiatedBy("map-action-move-environment")`, and the error strings unchanged (`move_environment operation missing name parameter`, `move_environment operation missing value parameter`, `unrecognized object kind [GATE]`). Payload assertions are identical: `WorldId==0`, `ChannelId==1`, `MapId==910010000`, `Instance==uuid.Nil`, plus the row's `Kind`/`Name`/`State`.

`TestExecuteResetEnvironment`:

| subtest name | params | expect |
|---|---|---|
| `creates saga step` | `{}` | nil error; one saga; step `Action == saga.ResetEnvironment`, `Status == saga.Pending`, `StepId == "reset-environment-910010000"`; payload type-asserts to `saga.ResetEnvironmentPayload` with `WorldId==0`, `ChannelId==1`, `MapId==910010000`, `Instance==uuid.Nil` |
| `extra params ignored` | `{"name":"ignored"}` | nil error; one saga; same payload |

`TestExecuteOperation_UnknownTypeStillWarns` — dispatch an op of type `"not_a_real_op"`; assert nil error and zero sagas created (guards against the new `case` arms swallowing the `default`).

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-map-actions/atlas.com/map-actions && go test ./script/... -v`
Expected: FAIL — both operations fall through to `default` and create no saga.

- [ ] **Step 3: Write minimal implementation**

`executor.go` — add to the switch after `case "drop_message":`:

```go
	case "move_environment":
		return e.executeMoveEnvironment(f, characterId, op)
	case "reset_environment":
		return e.executeResetEnvironment(f, characterId, op)
```

and append the two functions. They are the Task 12 bodies with `rc.Field` → `f`, the initiator strings changed to `"map-action-move-environment"` / `"map-action-reset-environment"`, and the step ids changed to `fmt.Sprintf("move-environment-%s", name)` and `fmt.Sprintf("reset-environment-%d", f.MapId())`. Add the `strings` import (this file currently imports `strconv` but not `strings`).

```go
// executeMoveEnvironment sets the state of one named field object via a saga.
// Parameters: name (required, non-blank), value (required, uint32), kind
// (optional, ENVIRONMENT or OBSTACLE; blank defaults to ENVIRONMENT).
func (e *OperationExecutor) executeMoveEnvironment(f field.Model, characterId uint32, op operation.Model) error {
	params := op.Params()

	name, ok := params["name"]
	if !ok || strings.TrimSpace(name) == "" {
		return fmt.Errorf("move_environment operation missing name parameter")
	}

	valueStr, ok := params["value"]
	if !ok || strings.TrimSpace(valueStr) == "" {
		return fmt.Errorf("move_environment operation missing value parameter")
	}
	value, err := strconv.ParseUint(strings.TrimSpace(valueStr), 10, 32)
	if err != nil {
		return fmt.Errorf("move_environment operation has non-numeric value parameter [%s]: %w", valueStr, err)
	}

	kind, err := field.ParseObjectKind(params["kind"])
	if err != nil {
		return err
	}

	e.l.Debugf("Moving environment object [%s] kind [%s] to state [%d] in map [%d].", name, kind, value, f.MapId())

	s := saga.NewBuilder().
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("map-action-move-environment").
		AddStep(
			fmt.Sprintf("move-environment-%s", name),
			saga.Pending,
			saga.MoveEnvironment,
			saga.MoveEnvironmentPayload{
				WorldId:   f.WorldId(),
				ChannelId: f.ChannelId(),
				MapId:     f.MapId(),
				Instance:  f.Instance(),
				Kind:      kind,
				Name:      name,
				State:     uint32(value),
			},
		).Build()

	return e.sagaP.Create(s)
}

// executeResetEnvironment clears every tracked field object and restores the
// field's objects to their default state. Takes no parameters.
func (e *OperationExecutor) executeResetEnvironment(f field.Model, characterId uint32, op operation.Model) error {
	e.l.Debugf("Resetting environment objects in map [%d].", f.MapId())

	s := saga.NewBuilder().
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("map-action-reset-environment").
		AddStep(
			fmt.Sprintf("reset-environment-%d", f.MapId()),
			saga.Pending,
			saga.ResetEnvironment,
			saga.ResetEnvironmentPayload{
				WorldId:   f.WorldId(),
				ChannelId: f.ChannelId(),
				MapId:     f.MapId(),
				Instance:  f.Instance(),
			},
		).Build()

	return e.sagaP.Create(s)
}
```

`docs/map_script_schema.json` — add `"move_environment"` and `"reset_environment"` to the enum at `:102-110`, and add two `allOf` branches mirroring the reactor schema's (the `move_environment` branch requires `["name", "value"]` and offers optional `kind` with `enum: ["ENVIRONMENT","OBSTACLE"]`; the `reset_environment` branch requires nothing).

`docs/domain.md` — add two rows to the operation table at `:67-75`:

```markdown
| `move_environment` | `MoveEnvironment` | `name`, `value` (uint32); optional: `kind` (`ENVIRONMENT` default, or `OBSTACLE`) |
| `reset_environment` | `ResetEnvironment` | (none) |
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-map-actions/atlas.com/map-actions && go build ./... && go test ./... `
Then, from the worktree root: `python3 -m json.tool services/atlas-map-actions/docs/map_script_schema.json > /dev/null && echo OK`
Expected: tests PASS and `OK`.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-map-actions
git commit -m "feat(atlas-map-actions): add move_environment and reset_environment operations"
```

---

## Task 14: Kafka documentation and final verification

### Files

- `services/atlas-maps/docs/kafka.md` — two rows in the `COMMAND_TOPIC_MAP` consumed table (`:42-48`) and two rows in the `EVENT_TOPIC_MAP_STATUS` produced table (`:85-96`)
- `services/atlas-channel/docs/kafka.md` — two rows in its `EVENT_TOPIC_MAP_STATUS` section (`:155`)
- `services/atlas-saga-orchestrator/docs/kafka.md` — two rows in its `COMMAND_TOPIC_MAP` produced section, if that file enumerates message types (check first with `grep -n "COMMAND_TOPIC_MAP" -A 12 services/atlas-saga-orchestrator/docs/kafka.md`; skip the file if it does not)

**There is no central Kafka topic manifest.** Design.md §3.5 defers this to the plan phase against `task-276-kafka-topic-manifest`; that task has not landed — there is no `docs/tasks/task-276-*` directory and no `*manifest*` file enumerating Kafka topics. Topic message types are documented per-service in `services/<svc>/docs/kafka.md`, and those are the only files to update.

`services/atlas-maps/docs/kafka.md` is already stale in an unrelated way (it documents no `PLAY_JUKEBOX` / `JUKEBOX_START` / `JUKEBOX_END`). **Do not backfill jukebox** — that is outside this task. Add only the four environment rows.

- [ ] **Step 1: Update the documentation**

`services/atlas-maps/docs/kafka.md`, `### COMMAND_TOPIC_MAP` table:

```markdown
| SET_ENVIRONMENT_STATE | SetEnvironmentStateCommandBody | Set the state of one named field object |
| RESET_ENVIRONMENT | ResetEnvironmentCommandBody | Clear all tracked field object state and restore defaults |
```

`services/atlas-maps/docs/kafka.md`, `### EVENT_TOPIC_MAP_STATUS` table:

```markdown
| ENVIRONMENT_STATE_CHANGED | EnvironmentStateChanged | A named field object's state changed |
| ENVIRONMENT_RESET | EnvironmentReset | Field object state was cleared; body carries the cleared objects |
```

Also update that section's lead sentence, which currently reads "Map status events emitted when characters enter or exit maps, when weather effects start or end, and when a map-stay timer is started." — append ", and when field environment object state changes or is reset."

`services/atlas-channel/docs/kafka.md`, `### EVENT_TOPIC_MAP_STATUS`: add the same two event rows in whatever column shape that table uses.

- [ ] **Step 2: Confirm the acceptance criteria**

Run each of these from the worktree root and confirm the stated result:

```bash
# All four writers have an emitting call site outside main.go and the wrappers.
grep -rn "SetObjectStateWriter\|FieldObstacleOnOffWriter\|FieldObstacleOnOffListWriter\|FieldObstacleAllResetWriter" \
  services/atlas-channel --include="*.go" | grep -v "/main.go:" | grep -v "/socket/writer/"
```
Expected: hits in `kafka/consumer/map/consumer.go` (and its test) covering all four constant names.

```bash
# The stub is gone.
grep -rn "This needs a new saga action for environment manipulation" services/
```
Expected: no output (exit 1).

```bash
# Nothing still reads "not yet implemented" for these two operations.
grep -rn "move_environment\|reset_environment" services/atlas-reactor-actions/docs/domain.md services/atlas-map-actions/docs/domain.md
```
Expected: hits that do **not** contain "not yet implemented".

```bash
# libs/atlas-packet is unchanged by the branch.
git diff --stat main...HEAD -- libs/atlas-packet
```
Expected: no output.

```bash
# Seed templates are unchanged by the branch.
git diff --stat main...HEAD -- services/atlas-configurations/seed-data/templates
```
Expected: no output.

If any of these five does not produce the expected result, fix it before proceeding — do not proceed to Step 3.

- [ ] **Step 3: Run the full verification gate**

Run: `tools/verify.sh` (flagless — `--quick` and `--no-docker` skip the bake and `-race` and do not count).
Expected: exit 0.

Dispatch this through the `task-verifier` agent rather than running it inside a large implementer context.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-maps/docs/kafka.md services/atlas-channel/docs/kafka.md
git commit -m "docs: document environment map commands and status events"
```

- [ ] **Step 5: Code review before PR**

Run `superpowers:requesting-code-review`. This change crosses four service boundaries (`atlas-maps` → Kafka → `atlas-channel`, and `atlas-saga-orchestrator` → Kafka → `atlas-maps`), so a green `tools/verify.sh` cannot see a seam defect. Trace `SET_ENVIRONMENT_STATE` and `ENVIRONMENT_RESET` into their consumers by hand and confirm a test asserts the new contract on both sides — in particular that the `Kind` / `Name` / `State` JSON tags and the four wire-string constants are byte-identical across the three copies of the map message types (`atlas-maps`, `atlas-saga-orchestrator`, `atlas-channel`).

# Map Environment Object State & Field Obstacles — Design

Task: `task-278-map-environment-object-state`
Input: `docs/tasks/task-278-map-environment-object-state/prd.md` (v1, approved)
Status: Draft for `/plan-task`

---

## 0. Summary of what the evidence changed

Four PRD assumptions did not survive contact with the client binaries and the tenant
templates. Each is resolved below with its evidence, and each changes the shape of the
implementation. Read §1 before §2 — the packet-selection design is the part the PRD got
wrong.

| PRD item | Verdict | Where |
|---|---|---|
| OQ-1 "is `0` the default? do not assume" | **Resolved: `0` IS the client's own default.** | §1.2 |
| FR-15 "GMS<61 sends one `FieldObstacleOnOff` per obstacle" | **Wrong.** `gms_v48` routes *no* obstacle writer at all; `gms_v61` routes neither `FieldObstacleOnOff` nor `FieldObstacleAllReset`. Replaced by a writer-availability fallback. | §1.3 |
| OQ-2 "is there an instance-teardown hook?" | **Resolved: none exists. One is added** at `CHARACTER_EXIT`-empties-field. | §3.4 |
| OQ-5 "exact REST route shape" | **Resolved:** `instanceId` is a required *path segment*, not a query parameter. | §5 |

OQ-3 (bulk event type) and OQ-4 (`kind` inference) are resolved in §4.3 and §6.

---

## 1. The wire layer — what the client actually does

### 1.1 All three set-state opcodes are one client function

Decompiled from the GMS v95.0 IDB (session `ecc757f4`, the version the repo derives
layouts from):

```
CField::OnSetObjectState          @0x539890  -> DecodeStr(name); Decode4(state); CMapLoadable::SetObjectState(this, name, state)
CField::OnFieldObstacleOnOff      @0x535a80  -> DecodeStr(name); Decode4(state); CMapLoadable::SetObjectState(this, name, state)
CField::OnFieldObstacleOnOffStatus@0x535b00  -> n = Decode4(); loop n { DecodeStr(name); Decode4(state); CMapLoadable::SetObjectState(this, name, state) }
```

`CField::OnSetObjectState` and `CField::OnFieldObstacleOnOff` are byte-for-byte the same
function body. They differ only in opcode. `OnFieldObstacleOnOffStatus` is the same
operation in a count-prefixed loop.

`CMapLoadable::SetObjectState` (@0x6203b0) resolves the name against a single dictionary,
`this->m_mNamedObj`:

```c
v3 = ZMap<char const*, CMapLoadable::CHANGING_OBJECT, ZXString<char>>::GetAt(&this->m_mNamedObj, &sName, 0);
if ( !v3 ) return 0;                                     // unknown name -> no-op
if ( nState != -1 ) {
    a = v3->aState.a;
    if ( !a || a[-1].bRestartMoving <= nState ) return 0; // out-of-range state -> no-op
    ...                                                   // stop current anim, alpha 0 on old state
    v4->nState = v6;
}
... // alpha 255 on new state, play bsSfx, MakeVectorAnimate if bRestartMoving, AnimateObjLayer
```

Three consequences that drive the design:

- **There is no client-side "environment vs obstacle" namespace.** Both kinds live in
  `m_mNamedObj`. `kind` is therefore a *wire-opcode selector on the server*, with no
  observable difference in client behaviour. This is what makes the fallback in §1.3 safe
  rather than a guess.
- **An unknown name is a silent client-side no-op** (`GetAt` returns null → `return 0`).
  PRD FR-3 is confirmed; the server genuinely does not need a WZ name index.
- **An out-of-range state is also a silent no-op.** The server cannot validate the range
  (it has no per-object state count), so out-of-range is accepted at the API and simply
  does nothing on the client. This is stated, not silently assumed.

`nState == -1` (`0xFFFFFFFF`) means "do not change state, just re-animate the current
one." The server never emits it; `state` is a `uint32` on the wire and the API rejects
nothing, so a script author who writes `4294967295` gets a re-animate. Documented, not
special-cased.

### 1.2 OQ-1 resolved: `0` is the client's own reset value

`CField::OnFieldObstacleAllReset`, GMS v95 @0x52c830:

```c
void CField::OnFieldObstacleAllReset(CField *this, CInPacket *iPacket)
{
  pos = this->m_lpObstacle._m_pHead;
  while ( pos ) {
    Next = ZList<ZRef<CMapLoadable::OBSTACLE>>::GetNext(&pos);
    CMapLoadable::SetObjectState(this, Next->p->sName._m_pStr, 0);
  }
}
```

Same on GMS v83 @0x5330b6 (session `754107bf`), structurally identical after decompiler
noise:

```c
v3 = this->m_lpObstacle[3];
while ( v3 ) { ...; CMapLoadable::SetObjectState(this, v5, 0); }
```

So the client's own "reset everything" primitive is literally `SetObjectState(name, 0)`
per object. PRD OQ-1 option **(a)** is correct and is now evidence-backed, not an
assumption.

But the same decompilation shows a second thing the PRD did not anticipate:
`OnFieldObstacleAllReset` walks `m_lpObstacle` — the **obstacle** list — not
`m_mNamedObj`. A named object that is not an obstacle (a PQ gate, a ship, a level object)
is **not** restored by `FieldObstacleAllReset`. Reset therefore cannot be a single
payload-free packet.

**Reset design (§4.5):** send `FieldObstacleAllReset` when the tenant routes it, *and*
send `SetObjectState(name, 0)` for every entry this server tracked. The two overlap
harmlessly on obstacles (setting an already-0 object to 0 re-runs the animation path but
changes no state), and the explicit sweep is the only thing that restores tracked
environment objects.

### 1.3 FR-15 is wrong — the real constraint is writer routing, not a version gate

Per-template writer routing, counted directly out of
`services/atlas-configurations/seed-data/templates/`:

| Template | `SetObjectState` | `FieldObstacleOnOff` | `FieldObstacleOnOffList` | `FieldObstacleAllReset` |
|---|---|---|---|---|
| `gms_48_1`  | ✅ | ❌ | ❌ | ❌ |
| `gms_61_1`  | ✅ | ❌ | ✅ | ❌ |
| `gms_72_1`  | ✅ | ✅ | ✅ | ✅ |
| `gms_79_1`  | ✅ | ✅ | ✅ | ✅ |
| `gms_83_1`  | ✅ | ✅ | ✅ | ✅ |
| `gms_84_1`  | ✅ | ✅ | ✅ | ✅ |
| `gms_87_1`  | ✅ | ✅ | ✅ | ✅ |
| `gms_95_1`  | ✅ | ✅ | ✅ | ✅ |
| `jms_185_1` | ✅ | ✅ | ✅ | ✅ |

FR-15's rule ("on GMS<61 send one `FieldObstacleOnOff` per obstacle instead of a list")
is unimplementable on `gms_v48`, which routes no `FieldObstacleOnOff` at all, and it does
not cover `gms_v61`, which routes the list but neither `FieldObstacleOnOff` nor
`FieldObstacleAllReset`.

`SetObjectState` is the only one of the four routed on **every** template, and §1.1 proves
it is the identical client operation. So:

> **Emission rule.** Ask the tenant's `writer.Producer` for the kind-preferred writer.
> If it returns `writer not found`, fall back to `SetObjectState` with the same
> `(name, state)`. For the bulk case, fall back to one `SetObjectState` per entry.

`writer.ProducerGetter` already returns `errors.New("writer not found")` for an unrouted
name (`libs/atlas-socket/writer/writer.go:28-35`), and `session.Announce` propagates that
error verbatim (`services/atlas-channel/atlas.com/channel/session/processor.go:265-270`).
The fallback is a testable branch on a real error, not a version sniff.

This also means **no version gate and no `MajorAtLeast` idiom appears anywhere in this
task.** The `gms_v48` legacy `NewFieldObstacleLegacy(flag, itemId, name)` constructor is
*not* used: its wire shape is `flag/itemId/name`, an item-triggered variant with no
`state` field, and it is not a drop-in for a `{name, state}` list. It stays untouched.

`libs/atlas-packet` is unchanged, per PRD acceptance criteria.

### 1.4 Writer coverage vs. acceptance criterion 1

PRD acceptance criterion 1 requires an emitting call site for all four writers outside
`main.go` and the writer wrappers. The design keeps all four:

| Writer | Emitting site |
|---|---|
| `SetObjectStateWriter` | `kind == ENVIRONMENT` broadcast; every fallback path; per-object reset sweep |
| `FieldObstacleOnOffWriter` | `kind == OBSTACLE` broadcast |
| `FieldObstacleOnOffListWriter` | enter-replay bulk obstacle announce |
| `FieldObstacleAllResetWriter` | reset broadcast |

The channel-side writer wrappers in
`services/atlas-channel/atlas.com/channel/socket/writer/` already exist for all four
(`set_object_state.go`, `field_obstacle_on_off.go`, `field_obstacle_on_off_list.go`,
`field_obstacle_all_reset.go`) and are used as-is.

---

## 2. Architecture

```
reactor script op  ─┐
map script op      ─┼─> saga step (MoveEnvironment | ResetEnvironment)
REST POST/DELETE   ─┘        │
                             v
              atlas-saga-orchestrator: handleMoveEnvironment / handleResetEnvironment
                             │  map_command.Processor
                             v
                COMMAND_TOPIC_MAP  {SET_ENVIRONMENT_STATE | RESET_ENVIRONMENT}
                             │  key = producer.CreateKey(int(mapId))
                             v
        atlas-maps  map/environment registry  (in-memory, keyed by {tenant, field})
                             │
                             v
              EVENT_TOPIC_MAP_STATUS  {ENVIRONMENT_STATE_CHANGED | ENVIRONMENT_RESET}
                             │
                             v
        atlas-channel map consumer -> ForSessionsInMap -> writer (per §1.3)

        atlas-channel SpawnForSelf ──REST GET──> atlas-maps environment resource
                             └─> announce to the entering session only
```

The shape is deliberately the weather/jukebox shape, which is the only in-repo precedent
for "per-field server-owned soft state that a late joiner must be replayed." Deviating
would buy nothing.

### 2.1 Rejected alternatives

**A. Own the state in `atlas-channel` instead of `atlas-maps`.** Cheaper — no Kafka
round trip, no REST client, the packet emitter and the state live together. Rejected:
`atlas-channel` is per-channel and the field key already spans world/channel/map/instance;
more importantly, `atlas-maps` is the service that owns field-scoped soft state
(`weather`, `jukebox`, `timer`, `character`) and owns the `CHARACTER_ENTER`/`EXIT`
lifecycle this feature needs for teardown (§3.4). Putting it in the channel would fork
that ownership and give the REST/GM surface (a PRD goal) no natural home.

**B. Persist the state to the database.** Rejected: weather, jukebox, map timers, and the
in-map character registry are all in-memory and process-scoped; a moved PQ gate is *less*
durable than a map timer, not more. Persistence would add a migration, a repository, and a
restart-consistency question (a restarted `atlas-maps` would replay stale gate positions to
clients whose `atlas-channel` had also restarted) in exchange for surviving a crash that
already destroys the PQ instance. PRD FR-6 already calls for in-memory; this confirms it.

**C. Drop the `kind` discriminator entirely and always send `SetObjectState`.** §1.1 shows
this is *behaviourally* correct on every version, and it is genuinely tempting: it deletes
an enum, three template-routing concerns, and the fallback branch. Rejected for two
reasons — it would leave `FieldObstacleOnOffWriter`, `FieldObstacleOnOffListWriter`, and
`FieldObstacleAllResetWriter` with no emitting call site, failing PRD acceptance
criterion 1 and leaving three verified codecs permanently dead; and the bulk-replay path
would degrade from one packet to N packets on every map enter for every version, which is
a real cost on a PQ map with a dozen obstacles. The `kind` split is kept as a *transport
optimisation with a proven-equivalent fallback*, which is exactly what §1.3 describes.

**D. A dedicated `COMMAND_TOPIC_ENVIRONMENT` / `EVENT_TOPIC_ENVIRONMENT` topic pair.**
Rejected: `COMMAND_TOPIC_MAP` and `EVENT_TOPIC_MAP_STATUS` are already the field-scoped
command/status pair, already keyed by map id (so ordering per field is already
guaranteed), and already consumed by exactly the two services that need this. New topics
would need manifest entries, new consumer groups, and buy nothing.

---

## 3. `atlas-maps` — `map/environment`

New package `services/atlas-maps/atlas.com/maps/map/environment/`, files
`registry.go`, `processor.go`, `producer.go`, `resource.go`, `rest.go` (+ tests),
mirroring `map/weather/`.

### 3.1 Shared kind constant

`ObjectKind` is needed by `libs/atlas-saga`, `atlas-maps`, and `atlas-channel`. Per the
repository convention ("check `libs/atlas-constants/` before defining a new domain type"),
it goes in `libs/atlas-constants/field/constants.go` alongside the existing `Id` type:

```go
type ObjectKind string

const (
    ObjectKindEnvironment ObjectKind = "ENVIRONMENT"
    ObjectKindObstacle    ObjectKind = "OBSTACLE"
)

func ParseObjectKind(s string) (ObjectKind, error) // "" -> ObjectKindEnvironment; unknown -> error
```

`ParseObjectKind` is the single place the default-to-`ENVIRONMENT` rule (§6, OQ-4) lives,
so the two script executors and the REST handler cannot drift.

### 3.2 Registry

```go
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
    entries map[FieldKey][]ObjectEntry // insertion-ordered
}
```

`FieldKey` is copied from `map/weather/registry.go:11-14` rather than shared — the two
registries are independent singletons and the weather one is not exported. Tenant
isolation is structural (PRD §6).

Operations:

- `Set(key, entry)` — replaces in place when `(Kind, Name)` already exists (preserving its
  original position), otherwise appends. Idempotent per FR-4: the entry is not duplicated,
  and the caller emits the status event unconditionally so a re-set still re-broadcasts.
  Ordering matters because replay order is client-observable (FR-17).
- `Get(key) []ObjectEntry` — returns a **copy** of the slice under `RLock`
  (PRD §8 concurrency). `ObjectEntry` is a value type with no reference fields, so a
  `slices.Clone` is a full deep copy.
- `Delete(key)` — removes the key entirely, so an untracked field occupies no entry
  (PRD §8 memory).

`Set` is keyed on the pair `(Kind, Name)`, not on `Name` alone. §1.1 shows the client
namespace is shared, so the same name under two kinds would be two registry rows driving
the same client object. That is a script-authoring error; the registry stores what it is
told and the last write to reach the client wins. It is not worth a validation branch that
would reject a legal-but-odd script.

### 3.3 Processor

```go
type Processor interface {
    Set(f field.Model, kind field.ObjectKind, name string, state uint32) (ObjectEntry, error)
    Reset(f field.Model) []ObjectEntry
    GetAll(f field.Model) []ObjectEntry
}
```

- `Set` rejects a blank `name` (`400` at REST; a log + no-op on the Kafka path, since a
  Kafka handler has nobody to return an error to). `kind` is already parsed by the caller.
- `Reset` returns the entries it cleared, so the producer can build the per-object reset
  sweep (§1.2) and so the channel knows which environment objects to zero.
- Tenant comes from `tenant.MustFromContext(p.ctx)` in every method — no path reads state
  without a tenant (PRD §8 multi-tenancy).

The REST handlers and the Kafka command handlers both call these three methods. There is
exactly one code path that mutates the registry and exactly one that emits the event, so
"REST behaves exactly as the command path" (PRD §5) is structural rather than a
convention.

### 3.4 OQ-2 resolved: instance teardown

**There is no existing teardown hook.** Confirmed by reading the three field-scoped
registries:

- `map/weather/registry.go` and `map/jukebox/registry.go` expire by wall clock
  (`GetExpired` + a sweeper), never by field lifecycle.
- `map/character/registry.go` `RemoveCharacter` (lines 59-66) shrinks the slice but
  **never deletes the `MapKey`**, so "the map key disappeared" is not a signal that exists
  today.
- `map/timer/registry.go` is keyed by character, not field.

The PRD (FR-7) permits falling back to "cleared on explicit reset only" if this must be
stated. It does not have to be: the hook is producible here.

**Design:** `atlas-maps`'s own character consumer already handles `CHARACTER_EXIT` and
calls `mapcharacter.Processor.ExitAndEmit`
(`services/atlas-maps/atlas.com/maps/kafka/consumer/character/consumer.go:144`). After
that call returns, query `character.NewProcessor(l, ctx).GetCharactersInMap(txId, f)`; if
it is empty, call `environment.NewProcessor(l, ctx).Reset(f)` — registry clear **only**,
with **no** `ENVIRONMENT_RESET` event, because there is nobody left in the field to
receive it and the next entrant will get a clean (empty) replay anyway.

This is intra-service wiring in the consumer layer, not a cross-service reach, so it does
not violate the layering rule.

Two properties worth naming:

- A field that empties transiently (the whole party warps to the next PQ stage map and
  returns) loses its state. For a PQ that is the desired stage-reset behaviour.
- A crashed or restarted `atlas-maps` loses all state (FR-6). Unchanged.

`ExitAll` (character deletion / disconnect,
`consumer.go:194`) removes the character from every map; the same
empty-check is applied there for each field it was removed from. Because `ExitAll` does not
today report which fields it touched, the plan phase should extend
`character.Processor.ExitAll` to return the affected `[]MapKey` rather than re-deriving
them — that is a small, local signature change in the same service.

### 3.5 Kafka contract

`services/atlas-maps/atlas.com/maps/kafka/message/map/command.go`:

```go
CommandTypeSetEnvironmentState = "SET_ENVIRONMENT_STATE"
CommandTypeResetEnvironment    = "RESET_ENVIRONMENT"

type SetEnvironmentStateCommandBody struct {
    Kind  string `json:"kind"`
    Name  string `json:"name"`
    State uint32 `json:"state"`
}

type ResetEnvironmentCommandBody struct{}
```

`Kind` is a plain `string` on the wire (not `field.ObjectKind`) so an unrecognised value
from a future producer deserialises and is rejected by `ParseObjectKind` in the handler,
rather than failing the whole decode.

`services/atlas-maps/atlas.com/maps/kafka/message/map/kafka.go`:

```go
EventTopicMapStatusTypeEnvironmentStateChanged = "ENVIRONMENT_STATE_CHANGED"
EventTopicMapStatusTypeEnvironmentReset        = "ENVIRONMENT_RESET"

type EnvironmentStateChanged struct {
    Kind  string `json:"kind"`
    Name  string `json:"name"`
    State uint32 `json:"state"`
}

type EnvironmentReset struct {
    Cleared []EnvironmentObject `json:"cleared"` // {kind, name} of every entry that was tracked
}

type EnvironmentObject struct {
    Kind string `json:"kind"`
    Name string `json:"name"`
}
```

`EnvironmentReset` is **not** an empty body, contrary to PRD FR-12. §1.2 proves
`FieldObstacleAllReset` restores only obstacles; the channel must be told which
environment objects to zero explicitly, and it has no registry of its own. Carrying the
cleared list is the only way the channel can do that without a second REST call at
broadcast time.

Both are keyed by `producer.CreateKey(int(f.MapId()))` (FR-10), matching every other
map command/event, so all changes for one map are ordered on one partition.

The handlers are registered in `kafka/consumer/map/consumer.go` `InitHandlers` alongside
`handleWeatherStartCommand` and `handlePlayJukeboxCommand`, each guarded by the same
`if c.Type != ...` early return.

Both new command types must be added to the Kafka topic manifest if
`COMMAND_TOPIC_MAP`/`EVENT_TOPIC_MAP_STATUS` message types are enumerated there — the plan
phase resolves this against whatever `task-276-kafka-topic-manifest` landed, by `Grep`,
not by assumption.

---

## 4. `atlas-channel`

### 4.1 REST client

New package `services/atlas-channel/atlas.com/channel/environment/`
(`processor.go`, `requests.go`, `rest.go`, `mock/`), shaped exactly like
`channel/weather/` but returning a slice:

```go
type Processor interface {
    GetAll(f field.Model) ([]RestModel, error)
}
// requests.SliceProvider[RestModel, RestModel](l, ctx)(requestEnvironmentInMap(ctx, f), Extract, nil)()
```

`requests.SliceProvider` already exists (`libs/atlas-rest/requests/provider.go:23`).
The URL constant reuses the same `mapInstanceResource` shape as
`channel/weather/requests.go:12-13`:

```go
mapInstanceResource            = "worlds/%d/channels/%d/maps/%d/instances/%s"
mapInstanceEnvironmentResource = mapInstanceResource + "/environment"
```

### 4.2 Emission helper

One helper owns the §1.3 fallback so no call site can forget it:

```go
// announceObjectState sends one (name, state) to s using the writer preferred for
// kind, falling back to SetObjectState when the tenant routes no writer for that
// opcode. The fallback is behaviourally identical: CField::OnSetObjectState and
// CField::OnFieldObstacleOnOff are the same client function (CMapLoadable::SetObjectState),
// differing only in opcode -- see design.md §1.1. gms_v48 routes no obstacle writer
// at all and gms_v61 routes neither FieldObstacleOnOff nor FieldObstacleAllReset.
func announceObjectState(l, ctx, wp, kind, name, state) model.Operator[session.Model]
```

It resolves the preferred writer name (`OBSTACLE` → `FieldObstacleOnOffWriter`,
`ENVIRONMENT` → `SetObjectStateWriter`), probes `wp(writerName)`, and on error retries
with `SetObjectStateWriter`. Because `SetObjectState` is routed on all nine templates, the
second probe failing is a genuine misconfiguration and is logged at `Error`.

### 4.3 Status-event handlers

Two new arms in
`services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go` `InitHandlers`,
each following `handleStatusEventWeatherStart` (consumer.go:989-1010):

- `handleStatusEventEnvironmentStateChanged` — parses `kind`, then
  `_map.NewProcessor(l, ctx).ForSessionsInMap(f, announceObjectState(..., kind, name, state))`.
- `handleStatusEventEnvironmentReset` — for each session in the field:
  1. `FieldObstacleAllResetWriter` if routed (skipped silently on `gms_v48`/`gms_v61`);
  2. then `announceObjectState(kind, name, 0)` for **every** entry in
     `e.Body.Cleared`, in the order the event carries.

  Step 2 is what actually restores non-obstacle named objects (§1.2), and it is what makes
  reset correct on the two templates that route no `FieldObstacleAllReset`.

Broadcast failures log at `Error` with world/channel/map/instance, matching
consumer.go:1002-1004.

**OQ-3 resolved: no bulk status event.** No `ENVIRONMENT_STATE_LIST` type is added.
Scripts issue one `move_environment` at a time (there is not a single authored
`move_environment` in the repo today — §6), commands are per-object, and the only place a
list is genuinely useful is the enter replay, which is a single-session announce driven by
REST rather than by an event. Adding a bulk event now would be a third code path with no
producer. `FieldObstacleOnOffListWriter` still gets its emitting call site from §4.4.

### 4.4 Enter replay

The replay site is `SpawnForSelf`
(`services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go`), inside a new
`routine.Go` block next to the existing weather / `announceActiveVisuals` /
`announceActiveJukebox` blocks — **not** in `handleStatusEventCharacterEnter`. `SpawnForSelf`
is the function that already owns "send the entering session everything it needs to render
the field," it is called synchronously after `SetField` so ordering is guaranteed, and each
of its blocks already returns silently on fetch failure. PRD FR-16 names the weather replay
at consumer.go:355-365, which is inside `SpawnForSelf`; this is the same site.

New `announceEnvironmentState(l, ctx, wp, f, s)`:

1. `environment.NewProcessor(l, ctx).GetAll(f)`; on error, log at `Error` and return.
   Entry is never blocked (FR-18) — the block is already inside `routine.Go`, so it cannot
   fail the enter path.
2. Empty result → send nothing (FR-19).
3. Partition into obstacles and environment objects, preserving insertion order.
4. Obstacles: if `FieldObstacleOnOffListWriter` is routed, one
   `NewFieldObstacleOnOffList([]ObstacleState{...})`. If not (`gms_v48`), one
   `announceObjectState` per obstacle.
5. Environment objects: one `SetObjectState` each, in insertion order.

This is FR-17's ordering with FR-15's version rule replaced by the §1.3 routing rule.

---

## 5. REST resource (OQ-5 resolved)

Registered in `services/atlas-maps/atlas.com/maps/main.go` next to
`weather.InitResource` / `jukebox.InitResource` (main.go:148-149).

The route follows `map/weather/resource.go:26` exactly. **`instanceId` is a required path
segment, not a query parameter** — this corrects PRD §5:

```
GET    /worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/environment
POST   /worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/environment
DELETE /worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/environment
```

Handlers compose `rest.ParseWorldId` → `ParseChannelId` → `ParseMapId` → `ParseInstanceId`
(all four exist in `services/atlas-maps/atlas.com/maps/rest/handler.go:34-48`) and build
`field.NewBuilder(worldId, channelId, mapId).SetInstance(instanceId).Build()`. `POST`
additionally uses `rest.ParseInput[RestModel]` (`handler.go:24`).

`rest.go`:

```go
type RestModel struct {
    Id    string `json:"-"`
    Kind  string `json:"kind"`
    Name  string `json:"name"`
    State uint32 `json:"state"`
}
func (m RestModel) GetName() string { return "environment-objects" }
// Id is "<KIND>:<name>", per PRD §5.
```

Status codes, per PRD §5:

- `GET` → `200` with a possibly-empty `data` array. **Never `404`.** This is a deliberate
  divergence from `map/weather/resource.go:39-42`, which returns `404` when no weather is
  active: weather is a singleton resource, environment is a collection, and an empty
  collection is `200`. Called out so a reviewer does not read it as an inconsistency.
- `POST` → `202 Accepted`; `400` on blank `name` or unparseable `kind`; `404` only when
  world/channel/map is not routable.
- `DELETE` → `204 No Content`, including for an untracked field.

Both mutating verbs call the same `environment.Processor` methods the Kafka handlers call,
then produce the same status event (§3.3).

Tenant scoping is the standard tenant header middleware plus the `Tenant` component of
`FieldKey`; a cross-tenant read is structurally impossible, and the registry test asserts
it.

---

## 6. Saga actions and script operations

### 6.1 `libs/atlas-saga`

```go
// model.go, beside FieldEffectWeather (model.go:286)
MoveEnvironment  Action = "move_environment"
ResetEnvironment Action = "reset_environment"

// payloads.go, beside FieldEffectWeatherPayload (payloads.go:1300)
type MoveEnvironmentPayload struct {
    WorldId   world.Id         `json:"worldId"`
    ChannelId channel.Id       `json:"channelId"`
    MapId     _map.Id          `json:"mapId"`
    Instance  uuid.UUID        `json:"instance"`
    Kind      field.ObjectKind `json:"kind"`
    Name      string           `json:"name"`
    State     uint32           `json:"state"`
}

type ResetEnvironmentPayload struct {
    WorldId   world.Id   `json:"worldId"`
    ChannelId channel.Id `json:"channelId"`
    MapId     _map.Id    `json:"mapId"`
    Instance  uuid.UUID  `json:"instance"`
}
```

Both get `case` arms in `unmarshal.go` (beside `FieldEffectWeather` at
`unmarshal.go:636-645`).

### 6.2 `atlas-saga-orchestrator`

Four mechanical edits, each with an exact precedent:

- `saga/model.go:252` — re-export both actions; `saga/model.go:394` — re-export both
  payload types; `saga/model.go:1713` — two `case` arms in the payload unmarshaller.
- `saga/event_acceptance.go:324` — add both to the acceptance set.
- `saga/handler.go:189` — two interface methods; `handler.go:1028` — two dispatch arms;
  `handler.go:3715` — `handleMoveEnvironment` / `handleResetEnvironment` bodies modelled on
  `handleFieldEffectWeather`, each type-asserting its payload and returning an error before
  touching Kafka.
- `map_command/processor.go:15-18` — two new interface methods; `map_command/producer.go`
  — `SetEnvironmentStateCommandProvider` / `ResetEnvironmentCommandProvider`, both keyed by
  `producer.CreateKey(int(f.MapId()))`.

Per FR-24, both actions are fire-and-forget: the step completes when the command is
produced, and neither has a compensating action. Reversing a move is the script author's
job (a second `move_environment`, or `reset_environment`).

### 6.3 Script operations

`services/atlas-reactor-actions/.../script/executor.go:278-292` — `executeMoveEnvironment`
loses its `TODO` comment and its bare `return nil`, and builds a saga step in the shape of
`executeSpawnMonster` (executor.go:225-242):

```go
saga.NewBuilder().
    SetSagaType(saga.InventoryTransaction).
    SetInitiatedBy("reactor-action-move-environment").
    AddStep(fmt.Sprintf("move-environment-%s-%s", rc.Classification, name), saga.Pending,
            saga.MoveEnvironment, saga.MoveEnvironmentPayload{...}).Build()
```

Parameters (FR-26): `name` required and non-blank; `value` required and parsed with
`strconv.ParseUint(v, 10, 32)` — a parse failure returns an error, never a silent zero;
`kind` optional, through `field.ParseObjectKind` (blank → `ENVIRONMENT`, unrecognised →
error). Missing-parameter errors follow `executeDropMessage`'s style
(executor.go:311-314).

`reset_environment` is added as a new `case` at executor.go:62, taking no parameters.

`services/atlas-map-actions/.../script/executor.go:36-52` gets both operations with
identical parameter semantics, sharing `field.ParseObjectKind`. Its `ExecuteOperation`
takes `f field.Model` directly rather than a `ReactorContext`, so the payload is built from
`f` instead of `rc.Field`.

Neither `executeKillAllMonsters` nor `executeWeakenAreaBoss` is touched (PRD non-goal).

Docs:

- `services/atlas-reactor-actions/docs/reactor_script_schema.json` — add
  `reset_environment` to the operation-type `enum` (line 107-119) and an `allOf` branch for
  it; extend the existing `move_environment` branch (lines 219-239) with the optional
  `kind` property. `required` stays `["name", "value"]`.
- `services/atlas-reactor-actions/docs/domain.md:108` — replace "(not yet implemented)"
  with the real description; add `reset_environment` (FR-29). Line 109
  (`kill_all_monsters`) keeps its "not yet implemented" — that is a separate task.
- The `atlas-map-actions` schema and domain doc get the same two entries.

### 6.4 OQ-4 resolved: `kind` defaults to `ENVIRONMENT`

A repo-wide grep for `move_environment` / `moveEnvironment` / `MOVE_ENVIRONMENT` returns
seven hits, all of them plumbing:

```
services/atlas-reactor-actions/docs/reactor_script_schema.json:112,220
services/atlas-reactor-actions/atlas.com/reactor/script/executor.go:62,286
services/atlas-reactor-actions/docs/domain.md:108
.claude/commands/convert-reactor.md:38,132
```

**There is not one authored `move_environment` operation in the repository.** No converted
script can be mis-routed by the default, because no converted script uses the operation
yet.

The conversion mapping is also unambiguous: `.claude/commands/convert-reactor.md:38` maps
`rm.getMap().moveEnvironment(name, val)` → `move_environment{name, value}`, and upstream
`moveEnvironment` is the `SetObjectState` path — i.e. exactly `kind == ENVIRONMENT`. So
defaulting to `ENVIRONMENT` is correct for every script the converter will produce, and
`kind: "OBSTACLE"` is an opt-in for hand-authored obstacle content.

Given §1.1 (both opcodes drive the same client function), a mis-specified `kind` is a
transport-efficiency mistake, not a correctness one. No per-map name convention is needed.

---

## 7. Testing

| Unit | Test |
|---|---|
| `environment.Registry` | `Set` appends new / replaces in place preserving order; `Get` returns a copy (mutating the result does not affect the registry); tenant A cannot see tenant B's entries for the same field; `Delete` removes the key |
| `environment.Processor` | blank `name` rejected; `Reset` returns the cleared entries and empties the key; `GetAll` on an untracked field returns an empty slice, not nil-panic |
| `environment` producers | `SetEnvironmentStateCommandProvider` / event providers build the right type and key (`producer_test.go` in `map_command/` is the precedent) |
| `atlas-maps` map consumer | `SET_ENVIRONMENT_STATE` applies + emits; wrong `c.Type` is ignored (mirrors `consumer_test.go:79`); unrecognised `kind` is rejected without mutating |
| `atlas-maps` character consumer | last character exiting a field clears that field's environment entries and emits **no** event; a field that still has characters is untouched |
| `environment` REST | `GET` empty → `200` + empty `data`; `POST` blank name → `400`; `POST` bad kind → `400`; `DELETE` untracked → `204`; cross-tenant `GET` sees nothing |
| `announceObjectState` | `OBSTACLE` with the obstacle writer routed → `FieldObstacleOnOffWriter`; `OBSTACLE` with it unrouted → falls back to `SetObjectStateWriter`; `ENVIRONMENT` → `SetObjectStateWriter`. Driven by a stub `writer.Producer` that returns `errors.New("writer not found")` for the unrouted name — no tenant version fixture needed |
| enter replay | tracked obstacles + environment → one `FieldObstacleOnOffList` then N `SetObjectState` in insertion order; with the list writer unrouted → N+M `SetObjectState`; empty state → zero packets; REST failure → zero packets and the enter still completes |
| reset broadcast | `AllReset` sent when routed, skipped when not; `SetObjectState(name, 0)` sent for every cleared entry either way |
| saga | `MoveEnvironment` / `ResetEnvironment` round-trip through `libs/atlas-saga/unmarshal.go`; both present in `event_acceptance`; both handlers reject a wrong-typed payload before touching Kafka (`TestHandlePlayJukebox_InvalidPayload` is the precedent) |
| reactor executor | `move_environment` with `name`+`value` creates a saga step with the expected payload (replacing today's `return nil`); non-numeric `value` → error; missing `name` → error; `kind: "OBSTACLE"` reaches the payload; omitted `kind` → `ENVIRONMENT` |
| map-actions executor | same four cases |

Test setup uses the project Builder pattern; no `*_testhelpers.go`.

`tools/verify.sh` (flagless) must exit 0 before the branch is called done.

---

## 8. Service impact (revised)

| Service / lib | Change | Delta vs PRD §7 |
|---|---|---|
| `libs/atlas-constants` | `field.ObjectKind` + `ParseObjectKind` in `field/constants.go` | **New** — PRD did not name a home for the shared enum |
| `libs/atlas-saga` | 2 actions, 2 payloads, 2 unmarshal arms | as PRD |
| `atlas-saga-orchestrator` | re-exports, acceptance set, 2 handlers, 2 `map_command` methods + 2 providers | as PRD |
| `atlas-maps` | `map/environment` package; 2 command types + 2 consumer arms; 2 status types; route registration; **empty-field teardown in the character consumer**; `character.Processor.ExitAll` returns affected keys | teardown + `ExitAll` signature are new |
| `atlas-channel` | `environment` REST client; 2 status-event arms; `announceObjectState` fallback helper; enter replay in `SpawnForSelf` | fallback replaces the PRD's GMS<61 rule |
| `atlas-reactor-actions` | implement `executeMoveEnvironment`; add `reset_environment`; schema + `domain.md` | as PRD |
| `atlas-map-actions` | both operations + schema/doc | as PRD |
| `libs/atlas-packet` | **no change** | as PRD |
| seed templates | **no change** — no new opcode is routed; `gms_v48`/`gms_v61` gaps are handled by fallback, not by adding opcodes we have no evidence for | clarification |

---

## 9. Out of scope (restated)

`executeKillAllMonsters`, `executeWeakenAreaBoss`, per-map PQ obstacle content, an
`atlas-party-quests` action executor, new codecs or version gates, and `atlas-ui`. All per
PRD §2.

## 10. Known follow-ups (genuine, not deferrable work)

- **No obstacle/environment name index exists** in `libs/atlas-wz` or `atlas-data`, so the
  server cannot validate a name and a typo is a silent no-op (§1.1). Building that index is
  a real, separately-scoped data task; it is not a prerequisite for this mechanism and the
  PRD explicitly accepts opaque names (FR-3).
- **`gms_v48` and `gms_v61` obstacle opcodes are unrouted, not proven absent.** The fallback
  makes this moot for behaviour. Whether those clients genuinely lack the handlers, or the
  templates are merely incomplete, is a packet-coverage question for the matrix, not for
  this task.

# Task-033 Context — Monster Aggro & Controller Switching

Quick reference for the executing subagent. Companions: `prd.md`, `design.md`,
`data-model.md`, `plan.md`.

---

## Services in scope

| Service | Path |
|---|---|
| atlas-monsters | `services/atlas-monsters/atlas.com/monsters/` |
| atlas-channel | `services/atlas-channel/atlas.com/channel/` |

`go.mod` module names are `atlas-monsters` and `atlas-channel` (the short
names, not full repo paths). Run `go build ./...` and `go test ./...` from
each service directory.

---

## Key existing files (atlas-monsters)

| File | Role |
|---|---|
| `monster/registry.go` | Redis-backed registry. `applyDamageScript` Lua. `storedMonster`/`storedDamageEntry` structs and `to/fromStored`. Also `unmarshalTolerantArray` (handles cjson `{}` for empty arrays). |
| `monster/model.go` | Immutable `Model`. `entry` type for damage entries. `DamageEntries`/`DamageSummary`/`DamageLeader`. `Damage(charId, dmg)` Go method (used by tests / non-Lua paths only). |
| `monster/builder.go` | `Clone(m)` + `ModelBuilder` fluent setters. Includes `AddDamageEntry`. |
| `monster/processor.go` | `Damage(id, characterId, damages, attackType)` (lines 244-337). `StartControl`/`StopControl`. `Damage` already has DPS-leader controller-switch on lines 322-336. |
| `monster/kafka.go` | Event constants + body types. |
| `monster/producer.go` | Provider builders: `startControlStatusEventProvider`, `stopControlStatusEventProvider`, etc. |
| `monster/status_task.go` | `StatusExpirationTask` template for the new aggro decay task. |
| `monster/registry_test.go` | miniredis-based tests; `TestMain` initializes the registry. |
| `monster/processor_test.go` | Defines `newRecordingProcessor` that injects an `emitter` capturing emitted events. Use this for all processor tests. |
| `map/processor.go` | `CharacterIdsInFieldProvider(l)(ctx)(f)` — returns `model.Provider[[]uint32]`. Used for FR-10 field-membership check. |
| `main.go` | Tasks registered around line 83-85. Add `MonsterAggroDecayTask` here. |
| `monster/information/*.go` | `information.GetById(l)(ctx)(monsterId)` returns `Model` with `Boss() bool`. |

## Key existing files (atlas-channel)

| File | Role |
|---|---|
| `kafka/message/monster/kafka.go` | Mirror of monster service event types. Add `EventStatusAggroChanged` + `StatusEventAggroChangedBody`; add `ControllerHasAggro` to `StatusEventStartControlBody`. |
| `kafka/consumer/monster/consumer.go` | `InitHandlers`. `handleStatusEventStartControl` at line 223-247 — line 241 has hardcoded `false` to replace. Pattern for new `handleStatusEventAggroChanged` mirrors `handleStatusEventStartControl`. |
| `kafka/message/monster/kafka_test.go` | Existing JSON encode/decode tests. Use the same pattern for new fixtures. |
| `socket/writer/monster_control.go` | `StartControlMonsterBody(m, aggro bool)` — already supports the active/passive flag. No changes needed. |
| `monster/builder.go` | `NewModelBuilder(uniqueId, field, monsterId).SetControlCharacterId(...).SetX(...).SetY(...).SetStance(...).SetFH(...).SetTeam(...).MustBuild()` — used to construct a `Model` from event data. |
| `monster/processor.go` | `monster.NewProcessor(l, ctx).GetById(uniqueId)` — used by `AGGRO_CHANGED` handler to look up the channel-side monster. |
| `docs/kafka.md` | Update for new `AGGRO_CHANGED` consumer + `START_CONTROL` body field. |

---

## Design decisions (locked, do not re-litigate)

| # | Decision |
|---|---|
| 1 | `applyDamageScript` returns `{wasFirstHit, monster}` envelope. |
| 2 | Decay sweep is hybrid: Go pre-filter on snapshot, conditional Lua write. |
| 3 | FR-10 field-membership uses synchronous `_map.CharacterIdsInFieldProvider`. |
| 4 | Controller switch stays two-step (`StopControl` + `StartControl`) — NOT a single Lua. |
| 5 | `AGGRO_CHANGED` emitted at end of `Damage` iff `firstHitObserved && !controllerSwitched`. |

---

## Data shapes (target after this task lands)

### Redis `storedDamageEntry`
```go
type storedDamageEntry struct {
    CharacterId uint32 `json:"characterId"`
    Damage      uint32 `json:"damage"`     // aggregated total
    LastHitMs   int64  `json:"lastHitMs"`  // unix milliseconds
}
```

### Redis `storedMonster` (delta)
```go
type storedMonster struct {
    // ... existing ...
    ControllerHasAggro bool `json:"controllerHasAggro"`
}
```

### Go `Model` / `entry` (delta)
```go
type entry struct {
    CharacterId uint32
    Damage      uint32
    LastHitMs   int64
}

type Model struct {
    // ... existing ...
    controllerHasAggro bool
}
```

`Model.ControllerHasAggro() bool` getter; `ModelBuilder.SetControllerHasAggro(bool)`.

### `DamageSummary` (delta)
```go
type DamageSummary struct {
    CharacterId   uint32
    Monster       Model
    VisibleDamage uint32
    ActualDamage  int64
    Killed        bool
    WasFirstHit   bool // NEW — set by ApplyDamage script envelope
}
```

### Constants — `monster/aggro.go`
```go
const (
    AggroIdleThresholdMs = int64(10000)        // Cosmic MonsterAggroCoordinator idle threshold
    AggroDecayMultiplier = 0.85                // Cosmic decay multiplier per tick
    AggroDecayFloor      = uint32(1)           // prune-below threshold
    AggroSweepInterval   = 1500 * time.Millisecond
)
```

### `monster/kafka.go` — atlas-monsters (delta)
```go
const EventMonsterStatusAggroChanged = "AGGRO_CHANGED"

type statusEventStartControlBody struct {
    ActorId            uint32 `json:"actorId"`
    X                  int16  `json:"x"`
    Y                  int16  `json:"y"`
    Stance             byte   `json:"stance"`
    FH                 int16  `json:"fh"`
    Team               int8   `json:"team"`
    ControllerHasAggro bool   `json:"controllerHasAggro"`
}

type statusEventAggroChangedBody struct {
    ControllerCharacterId uint32 `json:"controllerCharacterId"`
    ControllerHasAggro    bool   `json:"controllerHasAggro"`
}
```

### `kafka/message/monster/kafka.go` — atlas-channel (delta)
```go
const EventStatusAggroChanged = "AGGRO_CHANGED"

type StatusEventStartControlBody struct {
    ActorId            uint32 `json:"actorId"`
    X                  int16  `json:"x"`
    Y                  int16  `json:"y"`
    Stance             byte   `json:"stance"`
    FH                 int16  `json:"fh"`
    Team               int8   `json:"team"`
    ControllerHasAggro bool   `json:"controllerHasAggro"`
}

type StatusEventAggroChangedBody struct {
    ControllerCharacterId uint32 `json:"controllerCharacterId"`
    ControllerHasAggro    bool   `json:"controllerHasAggro"`
}
```

---

## Lua scripts (target shapes)

### `applyDamageScript` (rewrite of `registry.go:381`)
```lua
local key = KEYS[1]
local charId = tonumber(ARGV[1])
local damage = tonumber(ARGV[2])
local nowMs = tonumber(ARGV[3])
local j = redis.call('GET', key)
if not j then
    return redis.error_reply("monster not found")
end
local m = cjson.decode(j)
local hp = m.hp
local actual = hp - math.max(hp - damage, 0)
m.hp = hp - actual

local entries = m.damageEntries
if type(entries) ~= 'table' or (next(entries) == nil and #entries == 0) then
    entries = {}
end

local found = false
local hadAnyEntry = #entries > 0
for _, e in ipairs(entries) do
    if e.characterId == charId then
        e.damage = e.damage + actual
        e.lastHitMs = nowMs
        found = true
        break
    end
end
if not found then
    table.insert(entries, {
        characterId = charId,
        damage = actual,
        lastHitMs = nowMs
    })
end
m.damageEntries = entries

local hadAggro = m.controllerHasAggro
local wasFirstHit = false
if m.controlCharacterId ~= 0 and not hadAggro then
    m.controllerHasAggro = true
    wasFirstHit = true
end
-- 'wasFirstHit' is also gated on hadAnyEntry == false to honor the spec
-- (first hit on a fresh monster). The flag flip alone covers it because the
-- Lua side only flips false→true on the first damage line that observes a
-- controller present and no prior aggro.

local encoded = cjson.encode({wasFirstHit = wasFirstHit, monster = m})
redis.call('SET', key, cjson.encode(m))
return encoded
```

`Registry.ApplyDamage` signature changes to:
```go
func (r *Registry) ApplyDamage(t tenant.Model, characterId uint32, damage uint32, uniqueId uint32, nowMs int64) (DamageSummary, error)
```

It decodes the envelope and sets `DamageSummary.WasFirstHit`.

### `decayDamageEntriesScript` (new)
```lua
local key = KEYS[1]
local now = tonumber(ARGV[1])
local idleMs = tonumber(ARGV[2])
local mult = tonumber(ARGV[3])
local floorVal = tonumber(ARGV[4])
local j = redis.call('GET', key)
if not j then
    return redis.error_reply("monster not found")
end
local m = cjson.decode(j)

local entries = m.damageEntries
if type(entries) ~= 'table' or (next(entries) == nil and #entries == 0) then
    entries = {}
end

local kept = {}
for _, e in ipairs(entries) do
    if (now - e.lastHitMs) > idleMs then
        e.damage = math.floor(e.damage * mult)
    end
    if e.damage >= floorVal then
        table.insert(kept, e)
    end
end
m.damageEntries = kept

local prevControllerId = m.controlCharacterId
local controllerCleared = false
if #kept == 0 and m.controlCharacterId ~= 0 then
    m.controlCharacterId = 0
    m.controllerHasAggro = false
    controllerCleared = true
elseif #kept == 0 then
    m.controllerHasAggro = false
end

redis.call('SET', key, cjson.encode(m))
return cjson.encode({
    controllerCleared = controllerCleared,
    prevControllerId = prevControllerId,
    monster = m,
})
```

`Registry.DecayDamageEntries` signature:
```go
type DecaySummary struct {
    Monster           Model
    PrevControllerId  uint32
    ControllerCleared bool
}
func (r *Registry) DecayDamageEntries(t tenant.Model, uniqueId uint32, nowMs int64) (DecaySummary, error)
```

---

## Behavior summary (atlas-monsters Damage flow)

1. Reflect check (unchanged).
2. Look up `information.GetById(...).Boss()` and `Revives()` (unchanged).
3. `nowMs := time.Now().UnixMilli()`.
4. For each damage line: `Registry.ApplyDamage(t, charId, d, uniqueId, nowMs)`.
   - Track `firstHitObserved = firstHitObserved || s.WasFirstHit`.
   - If `s.Killed`, break.
5. Emit `DAMAGED` event (unchanged).
6. If killed: emit `KILLED`, RemoveMonster, spawnRevives. Return.
7. **FR-10 + controller switch**: If `characterId != m.ControlCharacterId() && m.DamageLeader() == characterId`:
   - Call `attackerInField(m.Field(), characterId)`. If error or `false`, skip switch.
   - Else: if `m.ControlCharacterId() != 0`, call `StopControl(m)` (FR-9 — no `STOP_CONTROL` for ID 0). Then `StartControl(uniqueId, characterId)`. Set `controllerSwitched = true`.
8. **AGGRO_CHANGED**: If `firstHitObserved && !controllerSwitched`, call `emitAggroChanged(m, m.ControlCharacterId(), true)`.

## Behavior summary (decay sweep)

`MonsterAggroDecayTask.Run()` every 1500ms:

1. `monsters := GetMonsterRegistry().GetMonsters()` (per-tenant map).
2. `bossCache := map[uint32]bool{}` (per-tick).
3. `nowMs := time.Now().UnixMilli()`.
4. For each tenant `ten`, for each monster `m`:
   - `tctx := tenant.WithContext(ctx, ten)`.
   - Boss check via cache → `information.GetById(l)(tctx)(m.MonsterId())`. Skip if boss.
   - Skip if `len(m.DamageEntries()) == 0`.
   - Pre-filter: any entry with `IsAggroIdle(e, nowMs)`? If no, skip.
   - Call `Registry.DecayDamageEntries(ten, m.UniqueId(), nowMs)`.
   - If `summary.ControllerCleared`, emit `STOP_CONTROL` with `summary.PrevControllerId` via `stopControlStatusEventProvider(summary.Monster, summary.PrevControllerId)`.

---

## Test infrastructure

### atlas-monsters
- `monster/registry_test.go` uses miniredis via `TestMain` (`testMiniRedis` global). Use `r.Clear(ctx)` at start of each test.
- `monster/processor_test.go`'s `newRecordingProcessor(t, ten)` returns `(*ProcessorImpl, *[]emittedEvent)`. The processor's `emit` field is a function variable injected for tests. **Important:** the processor calls `producer.ProviderImpl(...)` directly in many places (`StartControl`, `StopControl`, etc., and most `Damage` paths). Convert these call sites to use `p.emit` so tests can intercept. The PRD/design assumes this is in scope.
- For tests that need `information.GetById`, current `processor.Damage` calls it; tests pass against monster IDs that don't have an info entry (returns error, falls back). For new boss-exemption tests in the decay task, use a monster ID that the information cache returns boss=true. Inspect `monster/information/` cache shape if needed; otherwise inject a stub `bossLookupFn` field on the task struct (simplest seam).
- For `_map.CharacterIdsInFieldProvider` injection: today it's called inline. Add a struct field `inFieldFn func(field.Model) ([]uint32, error)` on `ProcessorImpl` defaulting to the real provider, so tests can stub it.

### atlas-channel
- `kafka/message/monster/kafka_test.go` is plain JSON encode/decode round-trip tests. Use the same pattern.
- Consumer tests do not currently exist for the monster package. New tests can use a stubbed `wp writer.Producer` that records announce calls, and a stubbed session lookup. If that's heavy, keep handler tests minimal: confirm the handler does not panic on a typed event and skips wrong tenant/world/channel.

---

## Build / test commands

```bash
# atlas-monsters
cd services/atlas-monsters/atlas.com/monsters
go build ./...
go test ./...

# atlas-channel
cd services/atlas-channel/atlas.com/channel
go build ./...
go test ./...
```

Docker:
```bash
docker build -f services/atlas-monsters/Dockerfile services/atlas-monsters
docker build -f services/atlas-channel/Dockerfile services/atlas-channel
```

(If Dockerfile paths differ, locate via `find services/atlas-monsters -name Dockerfile`.)

---

## Acceptance criteria — checklist mapping

PRD §10:

| AC | Plan task |
|---|---|
| Damage entries aggregated; legacy migration | T1, T4 |
| Non-controller takes DPS lead → STOP_CONTROL + START_CONTROL with `controllerHasAggro: true` | T9, T10 |
| Current controller stays leader → no STOP/START | T9 |
| First damage flips aggro and emits AGGRO_CHANGED (no controller change) | T9, T10 |
| Decay task runs every 1500ms with Cosmic schedule | T7, T12, T13 |
| Non-boss full clear → STOP_CONTROL + `controllerHasAggro: false`; bosses skipped | T7, T12 |
| atlas-channel `START_CONTROL` passes through `controllerHasAggro` | T15, T16 |
| atlas-channel `AGGRO_CHANGED` re-sends `MonsterControlWriter` | T17 |
| Boss retains state until death | T12 |
| Reflect / `DamageSourceHeal` don't write damage entries | (already true; T6 ensures DoT path passes nowMs and Heal path is unchanged — no new test needed beyond a regression note) |
| Existing tests updated for aggregated entries | T4, T5 |
| Docs updated | T14, T18 |
| Builds + tests pass | T19 |

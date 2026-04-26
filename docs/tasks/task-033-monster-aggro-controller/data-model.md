# Data Model — Monster Aggro & Controller Switching

Companion to `prd.md`. Captures the schema deltas in one place so the design and plan phases can review the migration shape without re-reading the full PRD.

## 1. Damage entry

### Current (`registry.go:76-79`)
```go
type storedDamageEntry struct {
    CharacterId uint32 `json:"characterId"`
    Damage      uint32 `json:"damage"`
}
```
Multiple rows per character — one row per damage line written by `applyDamageScript` via `table.insert`. `Model.DamageSummary()` (model.go:130-148) collapses them at read time.

### New
```go
type storedDamageEntry struct {
    CharacterId uint32 `json:"characterId"`
    Damage      uint32 `json:"damage"`     // accumulated total
    LastHitMs   int64  `json:"lastHitMs"`  // unix milliseconds, set by Go caller
}
```
One row per character. `applyDamageScript` upserts: if an entry with `characterId` exists, increment `damage` and overwrite `lastHitMs`; else append.

`Model.DamageSummary()` becomes a passthrough of `Model.DamageEntries()` — they return the same slice.

## 2. Monster model

### Stored (`registry.go:25-48`) — added field
```go
type storedMonster struct {
    // ... existing ...
    ControllerHasAggro bool `json:"controllerHasAggro"`
}
```

### In-memory (`model.go:21-40`) — added field
```go
type Model struct {
    // ... existing ...
    controllerHasAggro bool
}
```
Plus getter `ControllerHasAggro() bool` and builder `SetControllerHasAggro(bool)`.

## 3. Lua script — `applyDamageScript`

### Current (`registry.go:381-397`)
```lua
local m = cjson.decode(j)
local hp = m.hp
local actual = hp - math.max(hp - damage, 0)
m.hp = hp - actual
table.insert(m.damageEntries, {characterId = charId, damage = actual})
local encoded = cjson.encode(m)
redis.call('SET', key, encoded)
return encoded
```

### New (sketch — implementer to finalize during planning phase)
```lua
local m = cjson.decode(j)
local hp = m.hp
local actual = hp - math.max(hp - damage, 0)
m.hp = hp - actual

local found = false
for _, e in ipairs(m.damageEntries) do
    if e.characterId == charId then
        e.damage = e.damage + actual
        e.lastHitMs = nowMs
        found = true
        break
    end
end
if not found then
    table.insert(m.damageEntries, {
        characterId = charId,
        damage = actual,
        lastHitMs = nowMs
    })
end

local hadAggro = m.controllerHasAggro
if m.controlCharacterId ~= 0 and not hadAggro then
    m.controllerHasAggro = true
end

local encoded = cjson.encode(m)
redis.call('SET', key, encoded)
return encoded
```
- `nowMs` is passed in as ARGV[3]. The Go caller computes it via `time.Now().UnixMilli()` — keeping wall-clock generation in Go makes unit tests deterministic.
- The script returns the post-state. The Go caller compares `hadAggro` (decoded from before-state if needed, or inferred by checking the current Model before calling) against the new state to decide whether to emit `AGGRO_CHANGED`. Cleanest approach: the script returns both pre-aggro and post-aggro flags; alternatively, the Go caller reads the monster, computes the delta after the script runs, and emits accordingly. Implementer to choose during plan phase.

## 4. New decay script — `decayDamageEntriesScript`

Applied by the `MonsterAggroDecayTask`. Pseudocode:
```lua
local m = cjson.decode(j)
local now = tonumber(ARGV[1])
local idleMs = tonumber(ARGV[2])
local mult = tonumber(ARGV[3])  -- e.g., 0.85, scaled to int math
local floor = tonumber(ARGV[4])

local kept = {}
for _, e in ipairs(m.damageEntries) do
    if (now - e.lastHitMs) > idleMs then
        e.damage = math.floor(e.damage * mult)
    end
    if e.damage >= floor then
        table.insert(kept, e)
    end
end
m.damageEntries = kept

if #kept == 0 then
    m.controllerHasAggro = false
end

local encoded = cjson.encode(m)
redis.call('SET', key, encoded)
return encoded
```

The Go task wrapper inspects the post-state: if the entry list went from non-empty to empty AND the monster is non-boss, it calls `StopControl` (which clears `controlCharacterId` and emits `STOP_CONTROL`).

Note on float math in Lua: cjson encodes `0.85 * 100` as `85`, but applying `math.floor(damage * 0.85)` in Lua works fine for non-trivial damage values. For very small damage values the floor will round to 0 quickly, which the floor check then prunes — this is the desired behavior.

## 5. Migration of in-flight monsters

Monsters spawned before the upgrade have:
- Multiple `damageEntries` rows per character.
- No `lastHitMs` field on entries.
- No `controllerHasAggro` field on the monster.

`fromStored` is updated to:
1. Group entries by `characterId`, summing `damage`. Use `lastHitMs = 0` for any entry missing the field; if multiple legacy rows for one character exist, the merged entry's `lastHitMs` is `0` (oldest possible — they'll decay quickly, which is correct behavior for "we don't know how fresh this is").
2. Default `ControllerHasAggro` to `false` if missing.

The first `applyDamageScript` write after deploy upgrades the entry to the new format because the upsert path overwrites `lastHitMs`. The first decay sweep prunes any merged-from-legacy entries that were idle (which all of them are, given `lastHitMs = 0`).

No explicit migration job needed. Bounded by monster lifecycle; within a few minutes of deploy, all legacy state is gone.

## 6. Field summary

| Field | Location | Type | Purpose |
|---|---|---|---|
| `lastHitMs` | `storedDamageEntry`, `entry` (Go) | `int64` | Wall-clock ms of last damage application; drives decay |
| `controllerHasAggro` | `storedMonster`, `Model` | `bool` | Two-state flag for client active/passive rendering |
| `AggroIdleThresholdMs` | `monster/aggro.go` const | `int64` | Idle period before decay starts |
| `AggroDecayMultiplier` | `monster/aggro.go` const | `float64` | Multiplicative decay per tick |
| `AggroDecayFloor` | `monster/aggro.go` const | `uint32` | Removal threshold |
| `AggroSweepIntervalMs` | `monster/aggro.go` const | `time.Duration` | Sweep cadence (1500ms) |

# Mob Skill Firing Semantics + HP/MP Recovery — Design

Version: v1
Status: Draft
Created: 2026-04-27
PRD: [`prd.md`](./prd.md)

---

## 1. Summary

This design refines the PRD into concrete code shapes. Five gaps are closed in one feature: aggro-gated picker triggers (gap 1), missed-attack first-cast handling (gap 2), picker prop-fail re-pick scheduling (gap 3), periodic mob HP/MP recovery (gap 4), and REST visibility for picker/aggro state (gap 5).

All five PRD §9 open questions are resolved (see §3 Decisions Log). The work spans atlas-monsters and atlas-data; atlas-channel, atlas-packet, and atlas-constants are untouched.

## 2. Architecture overview

### 2.1 Call graph

```
Damage (existing)
  └─► ApplyDamage (Lua) — also writes lastDamageTakenMs = nowMs (NEW)
  └─► firstHitObserved? || HpPercentage changed?  (FR-3.1, loosened guard)
        └─► RepickAndEmit(RepickReasonDamaged)

Create (existing)
  └─► if ControllerHasAggro: RepickAndEmit(RepickReasonSpawn)  (FR-2.1)
       // always false at spawn → no-op in practice; guard kept for symmetry

PostUseSkill closure (in applyAnimationDelayedEffect)
  └─► if ControllerHasAggro: RepickAndEmit(RepickReasonPostUseSkill)  (FR-2.3)

MonsterSkillPickerSweepTask.Run
  └─► skip if !ControllerHasAggro (FR-2.2), in addition to existing skips
  └─► repickFn → RepickAndEmit(RepickReasonSweep)

pickNextSkill
  └─► track propEligibleSeen across loop  (FR-4.4)
  └─► if sentinel && propEligibleSeen:
        nextRepick = min(nextRepick, nowMs+sweepIntervalMs)  (FR-4.1, Q3:A)

MonsterRecoveryTask.Run  (NEW, every 10s)
  └─► for (tenant, monster):
        ├─► fetch info (per-Run cache)
        ├─► skip if hp==0, hpRecovery==0 && mpRecovery==0, or already-full
        ├─► applyRecoveryScript (Lua, atomic CAS):
        │     reads hp, mp, maxHp, maxMp, lastDamageTakenMs
        │     hp += hpRecovery iff hp<maxHp && idle>10s
        │     mp += mpRecovery iff mp<maxMp
        │     returns updated + hpApplied flag
        └─► if hpApplied: emit damagedStatusEventProvider(..., DamageSourceHeal, damage=0)
              (mirrors processor.go:683)
```

### 2.2 Component boundaries

- `monster.Model` gains one persisted timestamp (`lastDamageTakenMs`); existing immutable-builder discipline applies.
- `information.Model` gains two `uint32` recovery accessors; populated by Extract from the REST shape.
- `MonsterRecoveryTask` is a new file that mirrors the structure of `MonsterAggroDecayTask` and `MonsterSkillPickerSweepTask`. It exposes the same `Run() / SleepTime()` pair and is wired in `main.go`.
- The picker (`pickNextSkill`) gains one local boolean (`propEligibleSeen`) and one `min`. No new exports, no behavioral change for the non-prop-fail paths.
- The picker sweep, the spawn trigger, and the post-UseSkill trigger gain one aggro-check guard each. Implemented at the call sites; no new `RepickReason` (FR-2.5).
- atlas-data's monster reader gains two integer parses (`info/hpRecovery`, `info/mpRecovery`); the REST shape gains two snake_case fields. atlas-monsters' `information/rest.go` mirrors them.

## 3. Decisions log

| ID | PRD ref | Decision | Rationale |
|---|---|---|---|
| D1 | §9.2 | `lastDamageTakenMs` is a direct `int64` field on `monster.Model`, persisted in `storedMonster`. | Hot-path read in the recovery task; explicit field is robust against any future `damageEntries` pruning and avoids per-tick O(N_attackers) work. Plumbing cost is small relative to the rest of the feature. |
| D2 | §9.1 (FR-5.6) | HP-regen emits via the existing `damagedStatusEventProvider(..., DamageSourceHeal, damage=0)` pattern. MP-regen emits nothing. The recovery write itself uses a new `applyRecoveryScript` Lua for atomic CAS against concurrent damage. | The heal-event path is already wired end-to-end (`processor.go:683` `executeHeal`). A new Lua script is cheap (~30 lines) and keeps regen correct under concurrent damage; `UpdateMonster` is a full overwrite and would race. |
| D3 | gap-3 plumbing | Picker prop-fail re-pick scheduling uses `min(nextRepick, nowMs+sweepIntervalMs)` whenever `propEligibleSeen == true`, not only when `nextRepick == 0`. | Strict superset of PRD §FR-4.4; ensures low-prop skills fire under sustained aggro even when an unrelated skill is cooldown-locked. Cost is one extra `min` per loop and matches the intent of "re-roll until aggro decays." |
| D4 | §9.4 | Recovery does not special-case `RemoveAfter > 0`. | The despawn path is authoritative; healing a mob about to vanish is a cosmetic blip at worst. Skipping conflates despawn-timer with regen-policy and adds plumbing for no real gain. |
| D5 | §9.5 | MP regen runs independently of SEAL and other cast-blocking statuses. | Recovery task should be dumb: read hp/mp/recovery, write the result. SEAL is the cast gate, not a resource gate. v83 has no contrary signal. |
| D6 | §9.3 | atlas-data REST tags use snake_case (`hp_recovery`, `mp_recovery`); Go fields use PascalCase (`HpRecovery`, `MpRecovery`). | Confirmed by spot-check against `RemoveAfter`/`remove_after` in both `atlas-data/.../monster/rest.go:17` and `atlas-monsters/.../monster/information/rest.go:15`. |
| D7 | FR-5.9 reading | Boss exclusion is data-driven: boss templates carry `hpRecovery=0,mpRecovery=0` in WZ; the both-zero skip naturally excludes them. The recovery task does NOT inspect `info.Boss()`. | FR-5.9's first clause ("Boss mobs … SHALL be skipped") and second clause ("Zero is the explicit 'no regen' sentinel") read as redundant statements of the same data-driven rule. v83 reference WZ data backs this — bosses with non-zero recovery are vanishingly rare, and the simpler rule avoids special-casing one mob category in code. If a boss template ever ships with `hpRecovery > 0`, that's an intentional designer signal and we should honor it. |

## 4. Detailed component design

### 4.1 `monster.Model` and registry — `lastDamageTakenMs`

**Field placement** (D1): direct field on `monster.Model`, mirrored in `ModelBuilder` and `storedMonster`. Treated as a write-mostly bookkeeping field along with `nextEligibleRepickAtMs`.

**Builder contract:**

```go
// monster/model.go
func (m Model) LastDamageTakenMs() int64 { return m.lastDamageTakenMs }

// monster/builder.go
func (b *ModelBuilder) SetLastDamageTakenMs(v int64) *ModelBuilder {
    b.m.lastDamageTakenMs = v
    return b
}
// Clone copies through; Build returns the immutable Model.
```

**Persistence:**

```go
// monster/registry.go
type storedMonster struct {
    // ... existing fields ...
    LastDamageTakenMs int64 `json:"lastDamageTakenMs,omitempty"`
}
```

`omitempty` is used so existing records without the field deserialize to 0 — interpreted as "no damage taken yet" by the recovery task, which is correct for the only realistic cold-start case (mobs at full HP).

**Damage-side write:** the existing `applyDamageScript` Lua already operates on a JSON-encoded monster blob in Redis. The script decodes the blob, applies damage, re-encodes. The new field is a single additional assignment:

```lua
-- inside applyDamageScript, after damage application
mon.lastDamageTakenMs = nowMs  -- nowMs already passed to the script
```

This is the only damage-path edit. The Go-side `ApplyDamage` continues to return a `DamageSummary`; the new field rides through opaquely.

**Test:** unit test in `registry_test.go` asserts that `ApplyDamage` updates `LastDamageTakenMs()` to the passed `nowMs`.

### 4.2 Picker — `propEligibleSeen` + min-merge re-pick

```go
// monster/picker.go (inside pickNextSkill, after existing guards, before loop)
chosen := Decision{}
var nextRepick int64
propEligibleSeen := false
sweepIntervalMs := MonsterSkillPickerSweepInterval.Milliseconds()

for _, s := range ma.Skills() {
    // ... existing eligibility gates unchanged ...
    // (byte-overflow, AREA_POISON, info-fetch error, cooldown, HP threshold,
    //  MP, reflect/immunity already-active)

    // Reaching here means every gate passed. The only remaining check is the
    // prop roll. Mark propEligibleSeen BEFORE the roll so a fail still counts.
    propEligibleSeen = true

    prop := int(sd.Prop())
    if prop <= 0 {
        continue
    }
    if prop > 100 {
        prop = 100
    }
    if rng.Intn(100) < prop {
        chosen = Decision{SkillId: byte(skillId16), SkillLevel: byte(skillLevel16)}
        break
    }
}

chosen.DecidedAtMs = nowMs

// D3: when sentinel returned and at least one candidate prop-rolled, schedule
// a sweep-cadence re-pick. min-merges with any cooldown-derived nextRepick.
if chosen.SkillId == 0 && propEligibleSeen {
    candidate := nowMs + sweepIntervalMs
    if nextRepick == 0 || candidate < nextRepick {
        nextRepick = candidate
    }
}
chosen.NextEligibleRepickAtMs = nextRepick
return chosen
```

**Notes:**
- `propEligibleSeen` is set before `prop <= 0` is checked, so a 0-prop skill (which falls through with `continue`) doesn't count. This matches the PRD intent: "passed every gate including … but failed only the prop roll."
- The min-merge correctly handles the four sentinel-cause combinations:
  - SEAL (early return at picker.go:122) — bypasses this code entirely; `nextRepick == 0`. ✓ (FR-4.2)
  - Empty skills list (early return at picker.go:118) — same. ✓
  - Info-fetch error (early return at picker.go:114) — same. ✓
  - All-cooldown-gated — `propEligibleSeen == false`, branch skipped, cooldown-derived `nextRepick` survives. ✓
  - All-prop-failed — `propEligibleSeen == true`, `nextRepick = min(cooldownExpiry, nowMs+1500ms)`. ✓

**Tests** (in `picker_test.go`):
1. All candidates pass eligibility, all prop-roll fail → `NextEligibleRepickAtMs == nowMs + 1500`.
2. SEAL → `NextEligibleRepickAtMs == 0`, `propEligibleSeen` irrelevant.
3. Empty skills list → `NextEligibleRepickAtMs == 0`.
4. All skills cooldown-gated → `NextEligibleRepickAtMs == min cooldown expiry`, no min-merge with sweep.
5. Mixed: skill A on 5s cooldown, skill B prop-fails → `NextEligibleRepickAtMs == nowMs + 1500` (sweep wins via min).
6. Mixed: skill A on 500ms cooldown, skill B prop-fails → `NextEligibleRepickAtMs == nowMs + 500` (cooldown wins via min).

### 4.3 Aggro-gated triggers

Three call sites, each gains a single guard. No new types, no new RepickReason.

**Spawn (FR-2.1, in `processor.go` `Create`):**

```go
// existing (approximate):
//   _ = p.RepickAndEmit(uniqueId, RepickReasonSpawn)
// becomes:
if m.ControllerHasAggro() {
    _ = p.RepickAndEmit(uniqueId, RepickReasonSpawn)
}
```

In practice `ControllerHasAggro` is always `false` at spawn (no damage yet); the guard makes the post-condition explicit and protects against any future code path that flips aggro before the first damage event (e.g., if a consumer ever sets aggro on map-entry).

**Sweep (FR-2.2, in `picker_task.go` `Run`):**

```go
// inside the inner loop, alongside the existing nextEligibleRepickAtMs guard:
if !m.ControllerHasAggro() {
    continue
}
```

Placed before the `hasSkillsFn` check to short-circuit the per-template REST cache miss.

**Post-UseSkill (FR-2.3, in the `postExecute` closure of `applyAnimationDelayedEffect`):**

```go
// re-fetch the monster (may have changed during anim delay):
m2, err := GetMonsterRegistry().GetMonster(p.t, m.UniqueId())
if err != nil {
    return
}
if !m2.ControllerHasAggro() {
    return
}
_ = p.RepickAndEmit(m2.UniqueId(), RepickReasonPostUseSkill)
```

The re-fetch is necessary: aggro can decay during the animation delay (1-3 seconds). Reading the closure-captured `m` would test stale aggro state.

**Note:** `RepickReasonDamaged`, `RepickReasonStatusApplied`, `RepickReasonStatusExpired`, and `RepickReasonControlChange` remain un-gated (FR-2.4).

### 4.4 First-cast handling for missed attacks (FR-3.1)

**One-line change at `processor.go:312`:**

```go
// before:
if !killed && last.Monster.HpPercentage() != oldHpPercentage {

// after:
if !killed && (firstHitObserved || last.Monster.HpPercentage() != oldHpPercentage) {
```

`firstHitObserved` is already tracked at lines 283-294 from each `DamageSummary.WasFirstHit`. No new state.

**Tests** (in `processor_test.go`):
1. Damage applied, hits 0 dmg, `WasFirstHit = true` → repick fires.
2. Damage applied, hits 0 dmg, `WasFirstHit = false` → repick does NOT fire.
3. Damage applied, hits >0 dmg, HP percentage changed → repick fires (existing behavior preserved).

### 4.5 `MonsterRecoveryTask` (NEW)

**File:** `services/atlas-monsters/atlas.com/monsters/monster/recovery_task.go`

**Structure** (mirrors `MonsterSkillPickerSweepTask`):

```go
const MonsterRecoveryInterval = 10 * time.Second
const HpRecoveryIdleThresholdMs = 10000  // matches AggroIdleThresholdMs from task-033

type MonsterRecoveryTask struct {
    l        logrus.FieldLogger
    ctx      context.Context
    interval time.Duration
    nowFn    func() int64
    infoFn   func(t tenant.Model, monsterId uint32) (information.Model, error)
    applyFn  func(t tenant.Model, m Model, hpRecovery, mpRecovery uint32, nowMs int64) (Model, bool, bool, error)
    emitFn   func(t tenant.Model, m Model) error
}

func NewMonsterRecoveryTask(l logrus.FieldLogger, ctx context.Context, interval time.Duration) *MonsterRecoveryTask
func (tk *MonsterRecoveryTask) SleepTime() time.Duration { return tk.interval }
func (tk *MonsterRecoveryTask) Run()
```

**Run() body:**

```go
func (tk *MonsterRecoveryTask) Run() {
    monsters := GetMonsterRegistry().GetMonsters()
    nowMs := tk.nowFn()
    infoCache := make(map[uuid.UUID]map[uint32]information.Model)

    for ten, mons := range monsters {
        for _, m := range mons {
            // fast-path skips
            if !m.Alive() {
                continue
            }
            if m.Hp() == m.MaxHp() && m.Mp() == m.MaxMp() {
                continue
            }

            // info lookup with per-Run cache (per-tenant since tenants can
            // theoretically have divergent atlas-data overrides)
            tenantId := ten.Id()
            if infoCache[tenantId] == nil {
                infoCache[tenantId] = make(map[uint32]information.Model)
            }
            info, ok := infoCache[tenantId][m.MonsterId()]
            if !ok {
                fetched, err := tk.infoFn(ten, m.MonsterId())
                if err != nil {
                    tk.l.WithError(err).Debugf(
                        "Recovery: cannot fetch info for monster [%d]; skipping.", m.UniqueId())
                    continue
                }
                info = fetched
                infoCache[tenantId][m.MonsterId()] = info
            }

            hpR := info.HpRecovery()
            mpR := info.MpRecovery()
            if hpR == 0 && mpR == 0 {
                continue
            }

            updated, hpApplied, _, err := tk.applyFn(ten, m, hpR, mpR, nowMs)
            if err != nil {
                tk.l.WithError(err).Debugf(
                    "Recovery: apply failed for monster [%d]; skipping.", m.UniqueId())
                continue
            }
            if hpApplied {
                if err := tk.emitFn(ten, updated); err != nil {
                    tk.l.WithError(err).Debugf(
                        "Recovery: HP-bar emit failed for monster [%d].", updated.UniqueId())
                }
            }
        }
    }
}
```

**Production wiring** (in `NewMonsterRecoveryTask`):

- `infoFn`: `tenant.WithContext(ctx, t)` + `information.GetById(...)`.
- `applyFn`: calls a new `GetMonsterRegistry().ApplyRecovery(t, uniqueId, hpRecovery, mpRecovery, nowMs)`.
- `emitFn`: `tenant.WithContext(ctx, t)` + `producer.ProviderImpl(l)(tctx)(EnvEventTopicMonsterStatus)(damagedStatusEventProvider(updated, updated.UniqueId(), updated.UniqueId(), false, DamageSourceHeal, updated.DamageSummary()))`.

**Note on `damageSummary` for the heal emission:** `executeHeal` at `processor.go:683` passes `healed.DamageSummary()` — the post-heal summary. We do the same: it gives atlas-channel the up-to-date HP for its bar render. The `actorId == observerId == m.UniqueId()` framing matches "self-source" healing (the existing `executeHeal` pattern uses `m.UniqueId()` as both).

### 4.6 `applyRecoveryScript` (Lua)

**Location:** alongside `applyDamageScript` in `monster/registry.go`.

**Inputs:** `monsterKey, hpRecovery, mpRecovery, idleThresholdMs, nowMs`.

**Returns:** `[updatedJsonBlob, hpApplied, mpApplied]` (Lua → Go converts blob via existing `fromStored`).

```lua
-- pseudocode
local raw = redis.call('GET', KEYS[1])
if not raw then return {nil, 0, 0} end
local mon = cjson.decode(raw)

if mon.hp == 0 then return {raw, 0, 0} end

local hpApplied = 0
local mpApplied = 0

local hpRecovery = tonumber(ARGV[1])
local mpRecovery = tonumber(ARGV[2])
local idleThresholdMs = tonumber(ARGV[3])
local nowMs = tonumber(ARGV[4])

if hpRecovery > 0 and mon.hp < mon.maxHp then
    local since = nowMs - (mon.lastDamageTakenMs or 0)
    if since > idleThresholdMs then
        mon.hp = math.min(mon.maxHp, mon.hp + hpRecovery)
        hpApplied = 1
    end
end

if mpRecovery > 0 and mon.mp < mon.maxMp then
    mon.mp = math.min(mon.maxMp, mon.mp + mpRecovery)
    mpApplied = 1
end

if hpApplied == 1 or mpApplied == 1 then
    redis.call('SET', KEYS[1], cjson.encode(mon))
end

return {cjson.encode(mon), hpApplied, mpApplied}
```

**Atomicity:** the SCRIPT LOAD + EVALSHA pattern matches `applyDamageScript`. Concurrent damage and recovery either interleave with last-writer-wins on the blob, or one of them re-runs (Redis serializes per-key script execution). The `mon.hp == 0` early-out enforces "healing a dead mob is forbidden" (FR-8.4 last bullet).

**Note on `lastDamageTakenMs == 0` cold-start:** `nowMs - 0 > 10000` is true for any realistic `nowMs`, so a freshly-loaded mob without the field starts regenerating immediately on the next tick. This matches PRD §8.5: a fresh mob is at full HP anyway, so the regen call is a no-op via the `hp < maxHp` guard.

**Tests** (in `registry_test.go`):
1. Apply with `hpRecovery > 0`, `lastDamageTakenMs` 11s ago → hp increases, returns `hpApplied=true`.
2. Apply with `hpRecovery > 0`, `lastDamageTakenMs` 5s ago → hp unchanged, returns `hpApplied=false`.
3. Apply with `mpRecovery > 0`, `mp < maxMp` → mp increases, returns `mpApplied=true`.
4. Apply with `mp == maxMp` → mp unchanged.
5. Apply clamps at `maxHp` / `maxMp`.
6. Apply on `hp == 0` mob → no-op, returns both false.
7. Apply with both recoveries 0 → no-op (caller filters this; test confirms script handles it safely).

### 4.7 `information.Model` recovery accessors

**Files:**
- `monster/information/model.go`: add `hpRecovery uint32`, `mpRecovery uint32` fields + `HpRecovery() uint32`, `MpRecovery() uint32` getters.
- `monster/information/builder.go`: matching builder setters.
- `monster/information/rest.go`: matching `HpRecovery uint32 \`json:"hp_recovery"\`` and `MpRecovery uint32 \`json:"mp_recovery"\`` on the REST type; Extract populates the Model.

**Tests** (in `information/rest_test.go` if it exists, or alongside existing tests): round-trip a payload with `hp_recovery: 20, mp_recovery: 2` and assert the Model exposes them.

### 4.8 atlas-monsters monsters REST resource (FR-1)

**File:** `monster/rest.go` `RestModel`.

```go
type RestModel struct {
    // ... existing fields ...
    ControllerHasAggro     bool  `json:"controllerHasAggro"`
    NextEligibleRepickAtMs int64 `json:"nextEligibleRepickAtMs,omitempty"`
}
```

**Transform:** populate from `m.ControllerHasAggro()` and `m.NextSkillDecision().nextEligibleRepickAtMs` (using the existing `Decision`-style accessor on `monster.Model`).

**Note:** `controllerHasAggro` does NOT use `omitempty` (PRD §FR-1.4) — `false` is meaningful information, not absence. `nextEligibleRepickAtMs` uses `omitempty` because 0 is the documented sentinel for "no scheduled repick."

**Tests:**
- `rest_test.go`: round-trip a Model with `controllerHasAggro=true`, `nextEligibleRepickAtMs=12345` and assert both serialize.
- Same with `controllerHasAggro=false`, `nextEligibleRepickAtMs=0` — assert `controllerHasAggro` is present and `false`, `nextEligibleRepickAtMs` is omitted.

### 4.9 atlas-data — WZ parsing (§FR-5.1)

**Files:**
- `monster/reader.go` lines ~50-65 (where `RemoveAfter` is read): add two parses:

  ```go
  m.HpRecovery = uint32(node.GetIntegerWithDefault("hpRecovery", 0))
  m.MpRecovery = uint32(node.GetIntegerWithDefault("mpRecovery", 0))
  ```

- `monster/entity.go`: add `HpRecovery uint32`, `MpRecovery uint32` fields to the entity.
- `monster/rest.go`:
  ```go
  type RestModel struct {
      // ... existing fields ...
      HpRecovery uint32 `json:"hp_recovery"`
      MpRecovery uint32 `json:"mp_recovery"`
  }
  ```
  Update Extract to populate from the entity (and the reverse if Transform exists).

**Tests** (`reader_test.go`):
- The existing fixture at lines 32-33 already includes `<int name="hpRecovery" value="10000"/>` and `<int name="mpRecovery" value="50000"/>`. Add an assertion that `rm.HpRecovery == 10000` and `rm.MpRecovery == 50000` post-parse.
- `rest_test.go`: round-trip `hp_recovery` and `mp_recovery` JSON fields.

### 4.10 `main.go` wiring

```go
// services/atlas-monsters/atlas.com/monsters/main.go
// alongside MonsterSkillPickerSweepTask and MonsterAggroDecayTask:
recoveryTask := monster.NewMonsterRecoveryTask(l, ctx, monster.MonsterRecoveryInterval)
go task.Register(l, ctx)(recoveryTask)
```

Match the exact registration pattern used by the other two tasks (verify in implementation).

## 5. Sequence — recovery during sustained engagement

```
t=0s   Player hits mob (first hit, miss).
       ApplyDamage → lastDamageTakenMs = 0; firstHitObserved = true.
       Damage trigger guard now fires (FR-3.1) → RepickAndEmit(Damaged).
       Picker rolls; if any candidate fires, MoveMonsterAck carries skill.

t=2s   Player hits mob (10 dmg).
       lastDamageTakenMs = 2000; HpPercentage drops → repick.

t=10s  Recovery tick #1.
       since = 10000 - 2000 = 8000ms. Not > 10000. HP regen SKIPPED.
       MP regen runs unconditionally if mp < maxMp.

t=12s  Player walks away. No more damage.
t=20s  Recovery tick #2.
       since = 20000 - 2000 = 18000ms > 10000. HP regen APPLIED.
       Emit damaged event with damage=0, DamageSourceHeal.
       Channel updates HP bar.

t=30s+ Aggro decay (task-033) flips ControllerHasAggro=false ~10s after last damage.
       Picker sweep no longer re-evaluates this mob. (FR-2.2)
       Recovery continues as long as mob is alive.
```

## 6. Failure modes

| Failure | Behavior |
|---|---|
| atlas-data unreachable during recovery tick | Per-Run cache miss → log Debug → skip mob for tick. Next tick retries. |
| Concurrent damage during recovery Lua | Redis serializes script execution per key; both writes apply, last writer wins on the blob. Invariants (`hp <= maxHp`, `mp <= maxMp`) preserved by both scripts. |
| Mob killed during recovery Lua | Script's `mon.hp == 0` early-out aborts the heal write. |
| Process restart with `lastDamageTakenMs` field absent in stored blob | Deserialize as 0 → first recovery tick treats as "infinitely idle" → applies HP regen immediately. Mob is at full HP after restart anyway, so the call is a no-op via the `hp < maxHp` guard. |
| atlas-data deployed without `hp_recovery`/`mp_recovery` fields | Deserialize as 0 → recovery task treats as "no regen" → mob is skipped. Safe rollout window. |
| Picker sweep storms (every prop-fail re-pick triggers an emission) | Per PRD §8.1, accepted: ≤1 extra emission per 1.5s per aggro'd low-prop mob. atlas-channel inbox is last-writer-wins. |
| Aggro decays during animation delay between picker-fire and post-UseSkill repick | New aggro guard at the post-UseSkill site (FR-2.3) re-reads aggro and no-ops. |

## 7. Test strategy

**Unit tests** (per acceptance criteria §10.2):

- Picker:
  - All-prop-fail with eligible candidates → `nextEligibleRepickAtMs == nowMs + 1500`.
  - All-prop-fail mixed with cooldown-locked skill where cooldown < 1500ms → cooldown wins.
  - All-prop-fail mixed with cooldown-locked skill where cooldown > 1500ms → sweep wins (D3).
  - SEAL → 0.
  - Empty skills → 0.
  - All cooldown-gated, no prop-eligible → min cooldown expiry, no sweep merge.
- Sweep task:
  - Skip on `!ControllerHasAggro` even with `nextEligibleRepickAtMs` elapsed.
  - Repick when aggro held and elapsed.
- Spawn:
  - `Create` does not call repick (because newly-created mob has `ControllerHasAggro=false`).
- Post-UseSkill:
  - `postExecute` re-reads aggro and no-ops on flip-to-false.
- Damage trigger:
  - `firstHitObserved && HP unchanged` → repick fires.
  - `!firstHitObserved && HP unchanged` → no repick.
- Recovery task:
  - Apply MP when `mp < maxMp`.
  - No-op when `mp == maxMp`.
  - No-op when `mpRecovery == 0`.
  - Apply HP only when idle > 10s.
  - No-op on `hp == 0`.
  - Clamp at `maxHp` / `maxMp`.
  - Both-zero recovery skipped.
  - Boss exclusion is data-driven (not Go-code-driven): boss templates carry `hpRecovery=0,mpRecovery=0` in WZ data, so the both-zero skip naturally excludes them. Test asserts a boss template fixture with both recoveries 0 is skipped. (D7 below clarifies the FR-5.9 reading.)
- REST:
  - Round-trip `controllerHasAggro` (both true and false).
  - Round-trip `nextEligibleRepickAtMs` (non-zero present, zero omitted).
- atlas-data:
  - Round-trip `hp_recovery` and `mp_recovery` in REST.
  - Reader test: existing fixture's `<int name="hpRecovery" value="10000"/>` parses to `HpRecovery == 10000`.

**Build gates** (per §10.3): atlas-monsters and atlas-data both `go build ./... && go test ./...` clean. atlas-packet and atlas-constants sanity-built.

**Integration / manual** (per §10.1): the full PRD acceptance list — spawn-without-aggro, engage-then-cast, prop-fail recovery, MP regen mid-fight, HP regen out-of-combat, HP regen suppression during combat, boss with `hpRecovery=0`, aggro decay, REST visibility, miss flips aggro.

## 8. Service impact summary

| Service | Files modified | Files added |
|---|---|---|
| atlas-monsters | `monster/model.go`, `monster/builder.go`, `monster/registry.go` (storage + applyDamageScript + new ApplyRecovery), `monster/picker.go`, `monster/picker_task.go`, `monster/processor.go` (3 guards + 1 loosened condition), `monster/rest.go`, `monster/information/model.go`, `monster/information/builder.go`, `monster/information/rest.go`, `main.go` | `monster/recovery_task.go` |
| atlas-data | `monster/reader.go`, `monster/entity.go`, `monster/rest.go`, plus matching test files | none |
| atlas-channel, atlas-packet, atlas-constants | none | none |

## 9. Risks & mitigations

- **Risk: recovery task causes atlas-data REST load spike.** Mitigation: per-Run per-tenant cache of `information.Model` keyed by templateId. With tens of monsters per channel and a handful of templates per channel, cache hit rate is high; cold-start tick is the worst case.
- **Risk: applyRecoveryScript drift from applyDamageScript.** Mitigation: unit tests for both scripts in `registry_test.go`; integration test for concurrent damage + recovery on the same mob.
- **Risk: picker prop-fail loop becomes hot under sustained aggro on mobs with many low-prop skills.** Mitigation: per PRD §8.1, accepted at one extra emission per 1.5s per mob; atlas-channel last-writer-wins absorbs.
- **Risk: existing tests reference internal symbols renamed during this work.** Mitigation: the only renames are additive (new builder methods, new fields). No existing public symbols are removed or renamed.
- **Risk: `lastDamageTakenMs` cold-start on a chipped mob loaded from a pre-feature blob causes one cycle of premature regen.** Mitigation: per PRD §8.5, accepted; behavior corrects within 10s.

## 10. Out of scope

Confirmed non-goals from PRD §2:
- Boss multi-phase skill rotations and HP-band scripts (spec-task-4).
- AREA_POISON mist execution (spec-task-3).
- Player HP/MP regen.
- Client-visible aggro indicators.
- Non-damage aggro acquisition paths.
- Configurable per-tenant recovery cadence.

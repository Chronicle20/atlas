# Mob Skill Firing Semantics + HP/MP Recovery — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-04-27

---

## 1. Overview

Task-034 landed the server-side mob skill picker, but post-merge testing surfaced four behavioral and observability gaps that together prevent mob skill casting from matching v83 reference behavior. Mobs cast their first skill immediately on spawn instead of waiting for engagement. Mobs with a single low-prop skill that rolls poorly on first attempt strand themselves in the picker's sentinel state forever. Once a mob spends MP, it stays depleted because no recovery mechanism exists. And the monsters JSON:API resource omits the `controllerHasAggro` field, making it impossible to diagnose any of these issues from outside the service.

This task closes all four gaps in a single coherent feature: it (1) gates the picker's spawn, sweep, and post-`UseSkill` triggers on `controllerHasAggro=true` so mobs only cast after engagement; (2) fixes the picker's prop-fail dead-end by scheduling a sweep re-pick when at least one candidate was eligible but no roll succeeded; (3) extends the existing damage trigger to fire on first-hit even when no HP change occurred (handles missed attacks correctly); (4) adds a periodic mob HP/MP recovery task driven by atlas-data's `info/hpRecovery` and `info/mpRecovery` WZ values, with the v83 damage-idle gate on HP regen; and (5) surfaces `controllerHasAggro` and `nextEligibleRepickAtMs` on the monsters REST resource for diagnostic visibility.

The work spans atlas-monsters (picker, regen task, REST), atlas-data (WZ field exposure), and is invisible to atlas-channel. No client-facing packet shapes change.

## 2. Goals

Primary goals:
- Mobs only cast skills after a player has engaged them (taken damage from them, attacked them, or received aggro through any path). Spawn-then-immediate-cast is eliminated.
- Mobs with low-prop skills eventually fire those skills under sustained aggro (the picker re-rolls until the prop succeeds or aggro decays).
- Mobs regenerate HP and MP over time per WZ data, so prolonged engagements don't strand a mob with depleted MP unable to cast and don't leave a chipped mob at low HP indefinitely.
- The monsters REST resource exposes the fields needed to diagnose picker and aggro state without service-side log spelunking.

Non-goals:
- Boss multi-skill phase rotations and HP-band-driven scripts. Bosses participate in the picker via the same eligibility gates as ordinary mobs; multi-phase logic remains spec-task-4.
- AREA_POISON mist mechanics. The picker's mist exclusion remains in place; spec-task-3 still covers that work.
- Player HP/MP regeneration. Player-side recovery is unrelated and out of scope.
- Client-visible aggro indicators. The v83 client renders aggro implicitly through monster movement; we do not introduce new packets for it.
- Aggro acquisition through paths other than damage. v83 only acquires aggro through damage (including miss damage); we do not add proximity-based or map-entry-based aggro.
- Configurable recovery cadence per tenant. The 10s tick mirrors v83 and is hardcoded.

## 3. User Stories

- As a player, I want monsters to wait until I engage them before they cast skills, so spawn camps don't immediately apply WEAPON_ATTACK_UP buffs to passing mobs that I'm not fighting.
- As a player, I want a monster that I've been fighting for a while to eventually fire its signature skill (not silently fail to roll), so the encounter feels alive.
- As a player, I want to walk away from a chipped monster, return later, and find it healed (matching v83 expectations).
- As a player, I want a monster with skills that consume MP to regenerate enough MP to keep casting through a long fight, instead of depleting permanently after the first or second cast.
- As an operator/developer, I want to GET a monster's REST resource and see its `controllerHasAggro` and `nextEligibleRepickAtMs` fields, so I can diagnose "why isn't this mob casting?" without reading server logs.

## 4. Functional Requirements

### 4.1 REST visibility (gap 1)

- **FR-1.1** The atlas-monsters monsters JSON:API resource SHALL include a `controllerHasAggro` boolean attribute reflecting `monster.Model.ControllerHasAggro()`.
- **FR-1.2** The same resource SHALL include a `nextEligibleRepickAtMs` int64 attribute reflecting `monster.Model.NextSkillDecision().nextEligibleRepickAtMs` (or 0 when no decision is set).
- **FR-1.3** Both fields SHALL be read-only (not accepted on POST/PATCH if such verbs exist on the resource); they reflect server-managed internal state.
- **FR-1.4** The fields SHALL be omitted from JSON output only if their zero value is the natural omission semantic (i.e., `controllerHasAggro=false` is included; `nextEligibleRepickAtMs=0` MAY use `omitempty` since 0 is the documented sentinel for "no scheduled repick").

### 4.2 Aggro-gated picker triggers (gap 2)

- **FR-2.1** `RepickReasonSpawn` SHALL be suppressed when the monster's `ControllerHasAggro()` returns `false` at the moment the trigger would fire. Implementation: in `processor.go` `Create`, the existing `p.RepickAndEmit(uniqueId, RepickReasonSpawn)` call is replaced with a guarded equivalent that no-ops on idle mobs.
- **FR-2.2** The `MonsterSkillPickerSweepTask.Run()` loop SHALL skip monsters whose `ControllerHasAggro()` returns `false` even when their `nextEligibleRepickAtMs` is in the past. Implementation: add the aggro check alongside the existing `nextEligibleRepickAtMs` and `hasSkillsFn` filters.
- **FR-2.3** The post-`UseSkill` repick triggered from `applyAnimationDelayedEffect`'s `postExecute` closure SHALL be suppressed when the monster's `ControllerHasAggro()` returns `false` at the moment the post-execute fires. (A mob can lose aggro during the animation delay; in that case we should not repick.)
- **FR-2.4** `RepickReasonDamaged`, `RepickReasonStatusApplied`, `RepickReasonStatusExpired`, and `RepickReasonControlChange` SHALL fire un-gated regardless of aggro state. Damage acquires aggro; status and control changes are tied to real state transitions where re-evaluating is correct.
- **FR-2.5** No new `RepickReason` constants are introduced. The aggro gate is implemented at the call sites, not as a new reason.

### 4.3 First-cast handling for missed attacks (gap 2 follow-up)

- **FR-3.1** The damage trigger guard at `processor.go:312` SHALL be loosened from `if !killed && last.Monster.HpPercentage() != oldHpPercentage` to `if !killed && (firstHitObserved || last.Monster.HpPercentage() != oldHpPercentage)`. The `firstHitObserved` flag is already tracked at lines 283-294; this change is one-line.
- **FR-3.2** As a consequence of FR-3.1, a missed attack (damage value of 0) that flips `controllerHasAggro` from false to true SHALL fire the picker, allowing the mob to begin casting. Subsequent damaging hits continue to fire the picker via the `HpPercentage` change path.
- **FR-3.3** Misses that do not flip aggro (e.g., second-hit misses on an already-aggro'd mob) SHALL NOT fire the picker via this path. The `firstHitObserved` flag is set only on the first hit in the mob's lifecycle.

### 4.4 Picker prop-fail re-roll scheduling (gap 3)

- **FR-4.1** When `pickNextSkill` exhausts the candidate skill list without selecting any skill (sentinel decision), AND at least one candidate was prop-eligible (passed every gate including HP, MP, cooldown, AREA_POISON exclusion, byte-overflow, reflect/immunity already-active, but failed only the prop roll), the returned `Decision.NextEligibleRepickAtMs` SHALL be set to `nowMs + MonsterSkillPickerSweepInterval.Milliseconds()` (i.e., approximately 1500ms in the future).
- **FR-4.2** When the loop returns sentinel for any reason OTHER than prop failure (SEAL gate fired before the loop, empty skills list, all skills cooldown-gated, info-fetch error), the existing `nextRepick` calculation continues to apply. Specifically: SEAL/empty/info-error → `nextEligibleRepickAtMs=0`; all-cooldown-gated → `nextEligibleRepickAtMs = min cooldown expiry`.
- **FR-4.3** Prop-fail re-rolls SHALL be unbounded while aggro is held. The existing aggro decay (10s idle threshold per task-033's `AggroIdleThresholdMs`) provides the natural termination: once aggro flips to false, FR-2.2 stops the sweep from invoking the picker.
- **FR-4.4** The picker SHALL track whether at least one candidate reached the prop-roll stage during the loop. A new local boolean (e.g., `propEligibleSeen`) SHALL be set to true when a candidate passes every prior gate; if the loop completes with `chosen.SkillId == 0 && propEligibleSeen && nextRepick == 0`, set `nextRepick = nowMs + MonsterSkillPickerSweepInterval.Milliseconds()`.

### 4.5 Mob HP/MP recovery (gap 4)

- **FR-5.1** atlas-data's monsters resource (`services/atlas-data/atlas.com/data/monster/rest.go`) SHALL expose `hp_recovery` and `mp_recovery` fields parsed from the monster's WZ `info/hpRecovery` and `info/mpRecovery` values.
- **FR-5.2** atlas-monsters' `information.Model` (`services/atlas-monsters/atlas.com/monsters/monster/information/model.go`) SHALL gain `HpRecovery() uint32` and `MpRecovery() uint32` accessors. The `information/rest.go` REST shape (consumer side) SHALL gain matching `HpRecovery` and `MpRecovery` JSON fields and Extract-side population.
- **FR-5.3** A new `MonsterRecoveryTask` SHALL run every 10 seconds (`MonsterRecoveryInterval = 10 * time.Second`). Per tick, the task iterates `GetMonsterRegistry().GetMonsters()` for every tenant, fetches the monster's template `information.Model` (cached per-templateId per Run for cost), and applies recovery as follows:
  - **MP recovery:** Always-on while the mob is alive. `newMp = min(maxMp, currentMp + mpRecovery)`. Skip if `mpRecovery == 0` or `currentMp == maxMp`.
  - **HP recovery:** Apply only if the mob is alive AND `time.Now().UnixMilli() - lastDamageTakenMs > AggroIdleThresholdMs` (10000ms). `newHp = min(maxHp, currentHp + hpRecovery)`. Skip if `hpRecovery == 0` or `currentHp == maxHp`.
- **FR-5.4** The recovery task SHALL skip monsters with `controlCharacterId == 0` OR `hp == 0` OR `hp == maxHp && mp == maxMp` (already at full state).
- **FR-5.5** Recovery writes go through the registry's atomic update path (mirroring `Damage`/`SetNextSkillDecision`). The `applyRecoveryScript` Lua script SHALL atomically update `hp` and `mp` on the stored monster, respecting the max bounds.
- **FR-5.6** When HP recovery applies, the task SHALL emit a `MONSTER_DAMAGED` status event (or equivalent existing healing event if one exists; otherwise `damagedStatusEventProvider` with a positive heal value, OR a new `MONSTER_HEALED` event) so atlas-channel can update HP-bar packets to reflect the new HP. **Open question:** confirm whether v83 client rendering relies on a specific packet for HP regen, or whether the HP-bar simply reflects the next damage event. Default behavior: emit a synthetic damage event with damage=0 to trigger the existing HP-bar refresh path used elsewhere (similar to `processor.go:682`'s "Emit a damaged event with 0 damage to trigger HP bar update").
- **FR-5.7** When MP recovery applies, no event emission is required (MP is not currently broadcast in MoveLife or other packets to the controller).
- **FR-5.8** The recovery task SHALL track a `lastDamageTakenMs` per monster on the model, separate from the per-attacker `damageEntries[i].LastHitMs`. The existing damage flow SHALL update this field on every `ApplyDamage` call. Implementation note: derive from `max(damageEntries[i].LastHitMs)` if a separate field would create migration overhead; OR add the field directly to `monster.Model` alongside other timestamps.
- **FR-5.9** Boss mobs (where `info.Boss() == true`) and other mobs with `hpRecovery == 0` and/or `mpRecovery == 0` SHALL be skipped for that recovery type. Zero is the explicit "no regen" sentinel matching v83 WZ semantics.

### 4.6 Cross-cutting

- **FR-6.1** All new code MUST follow the project's immutable-model + builder pattern. Recovery state changes go through `Clone(m).SetHp(...).SetMp(...).Build()` via the registry's atomic update path; the `lastDamageTakenMs` field uses the same builder discipline.
- **FR-6.2** All registry reads in the recovery task MUST be tenant-scoped via `GetMonsters()` returning `map[tenant.Model][]Model`.
- **FR-6.3** All new logging SHALL be at `Debug` level for per-tick activity; `Info` for cadence start (e.g., "Initializing monster recovery task to run every 10000ms") matching `MonsterAggroDecayTask`'s style.
- **FR-6.4** No new metrics are introduced (matches task-034 §FR-31).

## 5. API Surface

### 5.1 atlas-monsters monsters resource (modified)

GET `/api/monsters/{uniqueId}` (or whatever the existing path is — confirm in design phase):

```json
{
  "data": {
    "type": "monsters",
    "id": "1002707",
    "attributes": {
      "worldId": 0,
      "channelId": 1,
      "mapId": 104010001,
      "instance": "00000000-0000-0000-0000-000000000000",
      "monsterId": 4090000,
      "controlCharacterId": 12,
      "controllerHasAggro": true,                       // NEW
      "nextEligibleRepickAtMs": 1730000005000,          // NEW (omitempty when 0)
      "x": 737,
      "y": 335,
      "fh": 44,
      "stance": 4,
      "team": -1,
      "maxHp": 2200,
      "hp": 2200,
      "maxMp": 60,
      "mp": 50,
      "damageEntries": [],
      "statusEffects": []
    }
  }
}
```

### 5.2 atlas-data monsters resource (modified)

GET `/api/monsters/{monsterId}` — adds two fields to the existing payload:

```json
{
  "name": "...",
  "hp": 2200,
  "mp": 60,
  "hp_recovery": 20,        // NEW
  "mp_recovery": 2,         // NEW
  ... (existing fields unchanged) ...
}
```

### 5.3 No new Kafka events or commands

The recovery task emits an existing-shape damaged event (FR-5.6) with damage=0 if a heal-specific event is not already present. No new event topic, body type, or producer is introduced. The picker's `NEXT_SKILL_DECIDED`, the inbox flow, and all other task-034 plumbing remain unchanged.

### 5.4 No client packet changes

`MoveMonsterAck`, `MonsterMovementAck`, and all client-visible monster packets remain unchanged.

## 6. Data Model

### 6.1 atlas-monsters `monster.Model` additions

| Field | Type | Initial | Purpose |
|---|---|---|---|
| `lastDamageTakenMs` | `int64` | `0` | Most recent `ApplyDamage` timestamp; gates HP recovery via the 10s idle window. Zero means "no damage taken yet" → HP regen runs (a fresh-spawned mob at full HP is a no-op anyway). |

The field follows the same persistence pattern as `nextEligibleRepickAtMs` (task-034 deviation): it MUST be persisted to Redis via `storedMonster` so the recovery task sees consistent state across in-memory rebuilds. `omitempty` on the JSON tag.

### 6.2 atlas-monsters `information.Model` additions

| Field | Type | Initial | Purpose |
|---|---|---|---|
| `hpRecovery` | `uint32` | `0` | Per-tick HP regen amount sourced from atlas-data WZ `info/hpRecovery`. |
| `mpRecovery` | `uint32` | `0` | Per-tick MP regen amount sourced from atlas-data WZ `info/mpRecovery`. |

Both follow the existing immutable-model + builder pattern in `information/`. No persistence (the model is reconstructed from atlas-data REST on each `GetById` call).

### 6.3 atlas-data `monster.RestModel` and entity additions

Two fields added to the REST shape and the entity-side parsing pipeline. The WZ source is `Mob.wz/####.img/info/hpRecovery` and `info/mpRecovery`.

### 6.4 No database schema changes

atlas-monsters uses Redis only (no relational DB). The new `lastDamageTakenMs` field rides in the existing JSON-encoded `storedMonster` value with `omitempty`. atlas-data's storage of WZ-derived data lives in its own representation; the design phase will confirm whether the existing entity layer needs a column add or whether the parsing pipeline already loads recovery values into memory.

## 7. Service Impact

### 7.1 atlas-monsters

| File | Change |
|---|---|
| `monster/rest.go` | Add `ControllerHasAggro bool` and `NextEligibleRepickAtMs int64 \`json:"nextEligibleRepickAtMs,omitempty"\`` to `RestModel`; populate in Transform. |
| `monster/model.go` | Add `lastDamageTakenMs int64` field + `LastDamageTakenMs()` getter. |
| `monster/builder.go` | Mirror new field in `ModelBuilder`; `SetLastDamageTakenMs` setter; copy through `Clone` and `Build`. |
| `monster/registry.go` | Extend `storedMonster` with `LastDamageTakenMs int64 \`json:"lastDamageTakenMs,omitempty"\``; update `toStored`/`fromStored`. Update `applyDamageScript` (or add a new touchpoint) to write `lastDamageTakenMs = nowMs` on every damage application. |
| `monster/processor.go` | (FR-2.1) Guard `RepickReasonSpawn` call in `Create` on `m.ControllerHasAggro()`. (FR-3.1) Loosen damage trigger guard to `firstHitObserved || HpPercentage changed`. (FR-2.3) Guard the post-UseSkill `postExecute` repick on aggro. |
| `monster/picker.go` | (FR-4.1, FR-4.4) Track `propEligibleSeen` across the loop; when sentinel returned with `propEligibleSeen && nextRepick == 0`, set `nextRepick = nowMs + sweepIntervalMs`. |
| `monster/picker_task.go` | (FR-2.2) Add aggro check alongside the existing skip filters; sweep no-ops for `!ControllerHasAggro()`. |
| `monster/recovery_task.go` (NEW) | New `MonsterRecoveryTask` running every 10s. Implements FR-5.3 through FR-5.9. |
| `monster/information/model.go`, `information/builder.go`, `information/rest.go` | Add `hpRecovery`/`mpRecovery` fields, accessors, builder methods, REST tags, and Extract-side population. |
| `main.go` | Register `MonsterRecoveryTask` alongside existing tasks. |

### 7.2 atlas-data

| File | Change |
|---|---|
| `monster/rest.go` | Add `HpRecovery uint32 \`json:"hp_recovery"\`` and `MpRecovery uint32 \`json:"mp_recovery"\`` to the monster `RestModel`. Update Extract to populate from the entity. |
| `monster/entity.go` | Add `HpRecovery` and `MpRecovery` fields if the entity layer is the parse target. |
| `monster/reader.go` | Update WZ parsing to read `info/hpRecovery` and `info/mpRecovery`. |
| Tests | Update `rest_test.go` / `reader_test.go` to verify the new fields are parsed and serialized. |

### 7.3 atlas-channel

No changes. The inbox flow continues to consume `NEXT_SKILL_DECIDED` events; aggro-gated triggers reduce event volume but require no consumer changes.

### 7.4 atlas-packet, atlas-constants

No changes.

## 8. Non-Functional Requirements

### 8.1 Performance

- The new `MonsterRecoveryTask` runs every 10s, scanning all in-memory monsters across all tenants. Cost is O(N_monsters × 1 Redis CAS) per tick. With typical fields holding tens of monsters per channel and ~1-3 channels per tenant, expected work per tick is well under 1ms wall-clock.
- The recovery task caches `information.Model` lookups per-templateId per-Run to avoid repeating atlas-data REST calls for identical templates.
- The picker's prop-fail re-pick (FR-4.1) increases sweep activity for aggro'd mobs with one or two low-prop skills. Worst case: a single 33%-prop skill rolling unsuccessfully causes one extra picker run + one extra `NEXT_SKILL_DECIDED` emission per 1.5s during sustained aggro. atlas-channel's inbox absorbs the extra emissions via last-writer-wins.

### 8.2 Multi-tenancy

- All registry reads SHALL go through tenant-scoped accessors (`GetMonstersInMap`, `GetMonsters` returning `map[tenant.Model][]Model`).
- atlas-data REST calls in the recovery task SHALL run with a tenant-enriched context (matching task-034's MT-06 fix in `picker_task.go`'s `hasSkillsFn` closure).
- The new `controllerHasAggro` and `nextEligibleRepickAtMs` REST fields are tenant-scoped via the existing resource-handler tenant gate.

### 8.3 Observability

- `MonsterRecoveryTask`: emit `Info` log at startup with cadence; per-Run `Debug` log summarizing applied recoveries for diagnostic depth.
- Per-monster recovery decisions (apply HP, apply MP, skip due to idle gate) SHALL log at `Debug` level only.
- Picker prop-fail re-pick scheduling SHALL log at `Debug` level (e.g., "Picker: monster [%d] all candidates prop-failed; rescheduling sweep at %d").

### 8.4 Failure Modes

- **atlas-data unreachable:** the recovery task's per-Run fetch of `information.Model` returns an error → mob is skipped for that tick. Implementation MUST NOT crash the tick on a single failed fetch.
- **Stale Redis state during recovery:** the recovery `applyRecoveryScript` is atomic per-monster; concurrent damage and recovery resolve via Redis CAS. Last-writer-wins.
- **Process restart:** `lastDamageTakenMs` is persisted in `storedMonster`; HP regen resumes correctly with the original idle window. `nextEligibleRepickAtMs` was already persisted in task-034.
- **Recovery applied to a dying mob:** the recovery script SHALL re-check `hp > 0` inside the Lua script and abort if the mob has been killed concurrently. Healing a dead mob is forbidden.

### 8.5 Backwards compatibility

- Existing `storedMonster` records without `lastDamageTakenMs` deserialize the field as 0, which means "no damage yet" → HP regen runs immediately on the next tick. This is acceptable for a freshly-loaded mob since it would also be at full HP. For a chipped mob loaded from a record without the field, the mob will start regenerating on first tick — this is a minor behavior shift on cold start that lasts at most one 10s cycle.
- Existing atlas-data responses without `hp_recovery`/`mp_recovery` (e.g., during the rollout window when atlas-data hasn't been redeployed yet) deserialize to 0, which the recovery task interprets as "no regen" — safe.

## 9. Open Questions

1. **HP-regen event emission shape (FR-5.6).** Does the v83 client rendering require a specific packet on HP regen, or is a `damagedStatusEventProvider(...)` with damage=0 sufficient to refresh the HP bar? The design phase should verify by inspecting the existing `processor.go:682` "Emit a damaged event with 0 damage" pattern's downstream consumer. Fallback: emit nothing, accept that the HP bar lags until the next packet — acceptable but worth confirming.
2. **`lastDamageTakenMs` field placement.** Add to `monster.Model` directly (FR-5.8 default) OR derive from `max(damageEntries[i].LastHitMs)` to avoid a new persistent field. Trade-off: deriving avoids a migration but adds runtime work and depends on `damageEntries` not being pruned. The design phase should commit to one approach.
3. **WZ field name casing.** atlas-data's existing field naming convention uses snake_case in REST and camelCase in models. Confirm `hp_recovery` (REST) and `hpRecovery` (model) match the project's WZ-field naming convention. Spot-check against existing recovery-adjacent fields (e.g., is it `remove_after` in REST? Yes, per `information/rest.go:15`).
4. **Should the recovery task respect `RemoveAfter`?** Mobs with `info.RemoveAfter > 0` despawn after a duration. Worth confirming the recovery task doesn't apply to mobs that are about to despawn. Default: no special-casing; let the despawn path delete the mob naturally.
5. **MP regen during status-effect-blocked casting.** A SEALed mob can't cast skills, but should it still regen MP? Default: yes, MP regen is independent of cast capability — when SEAL expires the mob has a fresh MP pool. Confirm in design.

## 10. Acceptance Criteria

### 10.1 Functional verification (manual gameplay)

- [ ] **Spawn-without-aggro:** spawn an Iron Hog (4090000); without engaging it, observe that no WEAPON_ATTACK_UP buff icon appears for at least 60 seconds. Logs SHOULD show no `RepickReasonSpawn` activity reaching the inbox-served path.
- [ ] **Engage-then-cast:** attack the Iron Hog (any hit, including misses). Within ~5 seconds of the first hit, the WEAPON_ATTACK_UP buff icon SHOULD appear (subject to the skill's prop roll succeeding).
- [ ] **Prop-fail recovery:** find a mob with a single low-prop skill (e.g., a mob with only DEFENSE_UP at 30% prop). Engage it. The skill SHOULD fire within ~30 seconds even if it doesn't fire on the first roll. (At 30% prop, ~3-5 rolls expected before success at 1.5s cadence.)
- [ ] **MP regen mid-fight:** observe a mob's MP via the REST endpoint during a sustained engagement. After casting a skill that consumes MP, MP SHOULD increase visibly over subsequent 10s ticks if the mob's `mp_recovery > 0`.
- [ ] **HP regen out-of-combat:** chip a mob to ~50% HP, leave the area for 30+ seconds, return. The mob's HP SHOULD have increased toward `maxHp` (subject to `hp_recovery > 0` and the 10s idle gate).
- [ ] **HP regen suppressed during combat:** continuously damage a mob for 30+ seconds. The mob's HP SHOULD NOT increase between hits during sustained damage.
- [ ] **Boss with hpRecovery=0:** chip a boss to ~50% HP, leave the area. The boss's HP SHALL NOT regen.
- [ ] **Aggro decay flips off picker:** engage a mob, then walk away. After ~10 seconds (`AggroIdleThresholdMs`), `controllerHasAggro` SHALL flip to false (per task-033) and the picker sweep SHALL stop re-evaluating the mob. Logs confirm.
- [ ] **REST visibility:** GET the monsters resource for an active mob. The response SHALL include `controllerHasAggro` and (when set) `nextEligibleRepickAtMs`.
- [ ] **Miss flips aggro:** for a mob whose level forces missed attacks (or via a low-accuracy character), confirm that purely missed attacks still flip `controllerHasAggro=true` and trigger the first cast.

### 10.2 Automated tests

- [ ] Unit test: picker loop returns `nextEligibleRepickAtMs == nowMs + sweepIntervalMs` when all candidates pass eligibility but every prop roll fails.
- [ ] Unit test: picker loop returns `nextEligibleRepickAtMs == 0` when sentinel is returned for a non-prop reason (SEAL, empty skills, info-fetch error).
- [ ] Unit test: damage trigger guard fires on `firstHitObserved=true` even when HP unchanged (miss case).
- [ ] Unit test: damage trigger guard does NOT fire on subsequent hits when neither HP changes nor `firstHitObserved` is true.
- [ ] Unit test: `MonsterSkillPickerSweepTask.Run()` skips a monster with `nextEligibleRepickAtMs > 0 && < nowMs && controllerHasAggro=false`.
- [ ] Unit test: `MonsterSkillPickerSweepTask.Run()` repicks a monster with `nextEligibleRepickAtMs > 0 && < nowMs && controllerHasAggro=true`.
- [ ] Unit test: `Create`'s spawn picker call no-ops when the freshly-created monster has `controllerHasAggro=false` (which is always, immediately post-spawn).
- [ ] Unit test: post-`UseSkill` `postExecute` repick no-ops when the mob has `controllerHasAggro=false` at trigger time (mob lost aggro during anim delay).
- [ ] Unit test: `MonsterRecoveryTask` applies `mpRecovery` to a mob with `currentMp < maxMp`.
- [ ] Unit test: `MonsterRecoveryTask` does NOT apply `mpRecovery` when `mpRecovery == 0`.
- [ ] Unit test: `MonsterRecoveryTask` does NOT apply `mpRecovery` when `currentMp == maxMp`.
- [ ] Unit test: `MonsterRecoveryTask` applies `hpRecovery` only when `nowMs - lastDamageTakenMs > AggroIdleThresholdMs`.
- [ ] Unit test: `MonsterRecoveryTask` does NOT exceed `maxHp`/`maxMp` (clamping).
- [ ] Unit test: `MonsterRecoveryTask` skips dead mobs (`hp == 0`).
- [ ] Unit test: `MonsterRecoveryTask` skips mobs with `hpRecovery=0 && mpRecovery=0`.
- [ ] Unit test: `controllerHasAggro` and `nextEligibleRepickAtMs` round-trip correctly via the monsters REST Transform.
- [ ] Unit test: atlas-data `monster.RestModel` round-trips `hp_recovery` and `mp_recovery`.
- [ ] Integration test (or sanity build): full atlas-monsters + atlas-data `go build ./... && go test ./...` clean.

### 10.3 Build and test gate

- [ ] `cd services/atlas-monsters/atlas.com/monsters && go build ./... && go test ./...` PASS.
- [ ] `cd services/atlas-data/atlas.com/data && go build ./... && go test ./...` PASS.
- [ ] `cd libs/atlas-packet && go build ./... && go test ./...` PASS (no expected changes; sanity check).
- [ ] `cd libs/atlas-constants && go build ./... && go test ./...` PASS (no expected changes; sanity check).
- [ ] Docker smoke build for atlas-monsters and atlas-data succeeds.

### 10.4 Code review gates

- [ ] `superpowers:requesting-code-review` dispatches plan-adherence + backend-guidelines reviewers; both clean (or all flagged issues addressed) before merge.

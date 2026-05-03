# PRD — Mob Basic Attack Handling (Magic / Ranged)

## Status

Investigation complete; design pending.

## Problem

Monsters whose normal attacks are magic or ranged (e.g. Samiho `5100004`, presumably Wraiths, Voodoos, Fire Boars) fire the attack **once** after being engaged, then never attack again until killed. Melee-only mobs (e.g. `6090003` in Fox Ridge) attack continuously as expected. Named SKILLS (mob skill IDs 100–200, e.g. SLOW=126, CURSE=124) work correctly across both — those go through the recently-fixed `atlas-monsters` picker pipeline.

The bug is specifically in the handling of basic ranged/magic attacks (`attack1`, `attack2`, `attack3` from mob WZ data), not skills.

## User-visible behavior

- v83 GMS client engaged with Samiho: Samiho fires its magic attack once on first aggro tick, plays animation, hits player. From that point forward, Samiho moves around but never attacks again.
- Same player engaged with `6090003` (melee-only, same map): mob attacks repeatedly across the full encounter as expected.

## Investigation findings (verified)

Reference investigation: prior conversation thread, supported by Cosmic v83 source comparison.

### Where the relevant code lives

- **atlas-channel mob-move handler:** `services/atlas-channel/atlas.com/channel/socket/handler/monster_movement.go` (opcode `0xBC`)
  - Delegates to `services/atlas-channel/atlas.com/channel/movement/processor.go:109` `Processor.ForMonster`.
  - `ForMonster` only branches on `skillId > 0` (the named skill case). Basic-attack actions (encoded in `nActionAndDir`) are not classified or validated.
- **atlas-monsters skill execution:** `services/atlas-monsters/atlas.com/monsters/monster/processor.go:483` `UseSkill`
  - Validates and decrements MP for skill casts (line 528 MP gate, line 535 `DeductMp`).
  - No equivalent path exists for basic attacks.
- **atlas-data mob template reader:** `services/atlas-data/atlas.com/data/monster/reader.go:174-186` `getAnimationTimes`
  - Parses only animation frame durations (`attack1`, `attack2`, `move`, etc.).
  - Does not parse `attack{1,2,3}/info` subnodes — `conMP`, `attackAfter`, `mpBurn`, `disease`, `deadlyAttack` are silently dropped.
- **atlas-data mob REST model:** `services/atlas-data/atlas.com/data/monster/rest.go:5-43`
  - No `attacks[]` field. Confirmed via live query of mob `7130003` (Dual Beetle): response includes `animation_times`, `skills`, `resistances`, but no per-attack metadata.

### Cosmic v83 reference flow

Cosmic's `MoveLifeHandler.java:108-114` classifies the inbound `rawActivity` byte:

```java
boolean isAttack = inRangeInclusive(rawActivity, 24, 41);
boolean isSkill  = inRangeInclusive(rawActivity, 42, 59);
...
} else {
    int castPos = (rawActivity - 24) / 2;
    int atkStatus = monster.canUseAttack(castPos, isSkill);
    if (atkStatus < 1) {
        rawActivity = -1;
        pOption = 0;
    }
}
```

`Monster.canUseAttack` (`Monster.java:1530-1559`) gates on MP, then calls `usedAttack` (`:1561-1576`):

```java
private void usedAttack(final int attackPos, int mpCon, int cooltime) {
    mp -= mpCon;                  // server-side MP decrement
    usedAttacks.add(attackPos);   // mark attack on cooldown
    Runnable r = () -> mons.clearAttack(attackPos);
    service.registerMobClearSkillAction(mmap.getId(), r, cooltime);
}
```

The MP value is then echoed back to the controlling client in `moveMonsterResponse` (`PacketCreator.java:1626`, equivalent to atlas's `MonsterMovementAck`). Cosmic's response carries a *decremented* MP after a magic-/ranged-attack action.

Atlas's `MonsterMovementAck` (`movement/processor.go:129`) returns `uint16(mo.Mp())` — always the unchanged MP because no decrement ever happens for basic attacks.

### Hypothesis (consistent with all evidence)

The v83 client's mob state machine, after firing a basic attack with `mpCon > 0`, expects the server's ack to reflect a decremented MP. When MP returns unchanged, the client treats the attack as not-yet-finalized and refuses to fire the same `attackPos` again. Since `usedAttacks` is also untracked server-side, no `clearAttack` pulse is ever sent that would otherwise resync.

Melee mobs (`mpCon = 0`, `attackAfter = 0`) are indistinguishable: nothing to decrement, no cooldown to clear, ack-with-unchanged-MP looks identical to "valid melee attack accepted."

This precisely matches the observed asymmetry between Samiho (magic, freezes after one) and `6090003` (melee, attacks continuously).

## Goals

- v83 client correctly identifies a magic-/ranged-attacking mob as "able to attack again" after each basic attack, matching Cosmic's behavior.
- atlas-monsters tracks per-mob basic-attack MP and per-attack-position cooldowns server-side.
- No regression for melee-only mobs (`mpCon = 0`).
- No regression for named-skill use (the picker pipeline is unrelated to this).

## Non-goals

- Server-authoritative attack-type validation (e.g. rejecting if range exceeds map distance, validating projectile trajectory). Cosmic doesn't fully do this either; out of scope.
- Adding `disease`, `mpBurn`, `deadlyAttack` semantics from `attack{1,2}/info`. Worth pulling into the data model alongside but separate gameplay decisions; deferrable.
- Improving stance/animation-timing fidelity. Untouched by this fix.

## Scope (preliminary — refine in design)

The fix spans three services. Confirmed via investigation, **medium total scope**:

1. **atlas-data**
   - Extend `monster/reader.go` `getAnimationTimes` (or sibling) to parse `attack{1,2,3}/info` subnodes.
   - Extract `conMP`, `attackAfter` (cooldown ms), and at minimum the position index. Optional first cut: also `disease`, `deadlyAttack`, `mpBurn` for downstream parity, but the bug-fix MVP only needs `conMP` + `attackAfter`.
   - Extend `monster/rest.go` `RestModel` with an `Attacks []AttackInfo` field. JSON shape suggested:
     ```json
     "attacks": [
       { "pos": 1, "conMP": 0, "attackAfter": 0 },
       { "pos": 2, "conMP": 5, "attackAfter": 1500 }
     ]
     ```

2. **atlas-monsters**
   - Add a `MonsterAttackCooldown` (or similar) in-memory registry keyed by `(tenant, uniqueId, attackPos)` storing earliest-eligible-time, with a TTL sweep to drop expired entries (mirror the existing skill cooldown registry pattern).
   - Add a processor method e.g. `UseBasicAttack(uniqueId, attackPos uint8) (validated bool, currentMp uint16, err error)` that:
     - Loads the mob's `Attacks[]` from atlas-data (cached lookup).
     - Looks up `conMP` and `attackAfter` for `attackPos`.
     - Gates on `m.Mp() >= conMP` AND not on cooldown → returns `validated=false` if either fails.
     - On validation: `DeductMp(conMP)`, register cooldown to expire at `now + attackAfter` ms.
   - Optional: emit a `MONSTER_BASIC_ATTACK_USED` Kafka event for observability / cross-service hooks. Defer if no consumer.

3. **atlas-channel**
   - In `movement/processor.go:109` `ForMonster`, after the existing skill branch, add a basic-attack branch:
     - Detect `nActionAndDir ∈ [24, 41]` (Cosmic's classification).
     - Compute `attackPos = (raw - 24) / 2` (Cosmic logic).
     - Call atlas-monsters' `UseBasicAttack(objectId, attackPos)`.
     - On success: use the returned `currentMp` in `MonsterMovementAck` instead of `mo.Mp()`.
     - On rejection: log and pass through unchanged (matches Cosmic — server can't authoritatively suppress an attack the client already played; the broadcast goes to others as `rawActivity = -1` to prevent visual replay there, but the controller already saw it).

## Risks / open questions for design

- **Cooldown registry shape.** The skill picker has a similar cooldown construct already; reuse vs separate registry. Reusing reduces surface area; separating clarifies semantics. Design should pick.
- **Where `UseBasicAttack` lives.** Direct atlas-monsters processor call (synchronous REST or in-process) vs Kafka command — atlas-channel already has both patterns. The skill path (`UseSkill`) is async via Kafka; consistency argues for the same here. But the controlling client needs an MP value back in the ack synchronously — implies either a sync REST hop OR atlas-monsters echoes the new MP via a follow-up packet rather than the ack. Design should resolve.
- **Atlas-data parse-time cost.** `attack{1,2,3}/info` parsing adds work to mob template loads (cached, so amortised). Confirm no startup regression in `atlas-data`.
- **Backwards compatibility of REST model.** Adding `attacks: []` to mob responses — verify no consumer breaks on the new field. Likely safe given JSON:API conventions but worth grepping.
- **`attackAfter` zero / missing.** Some mobs may have `attack1/info` without `attackAfter`. Default to 0 (no cooldown) — same effect as melee attacks today.

## Success criteria

- Samiho (`5100004`) at Fox Ridge fires its magic attack repeatedly across an encounter (verified by gameplay test).
- `6090003` continues to attack in melee continuously (no regression).
- New atlas-monsters tests cover: MP-gate rejection, cooldown rejection, successful decrement+cooldown registration, melee path bypass.
- New atlas-data tests cover: `attack{1,2}/info` parsing for at least one magic-attacker (Samiho) and one melee-only (Beetle).
- No new "Read a unhandled message with op 0xXX" lines around basic-attack actions.

## References

- Cosmic v83 reference: `~/source/Cosmic/src/main/java/server/life/Monster.java:1467-1576`, `net/server/channel/handlers/MoveLifeHandler.java:80-180`.
- Atlas attack MP gate (skills): `services/atlas-monsters/atlas.com/monsters/monster/processor.go:528-540`.
- Atlas mob-move handler: `services/atlas-channel/atlas.com/channel/movement/processor.go:109-167`.
- Atlas data reader: `services/atlas-data/atlas.com/data/monster/reader.go:174-186`.
- Atlas data REST: `services/atlas-data/atlas.com/data/monster/rest.go:5-43`.
- Recent merged context: PR #365 (mob skill effects on v83 — picker aggro gate, disease BuffGive shape, mist disease duration unit).

# Task 157 — Context

Companion to `plan.md`. Everything an implementer needs that isn't a plan step.

## What this task is

atlas-channel applies the damage-taken packet's value to HP verbatim (`socket/handler/character_damage.go:43`, `ChangeHP(..., -int16(p.Damage()))`) and carries 10 TODOs. This task implements the server-authoritative mitigation pipeline (design.md) for Magic Guard, Power Guard, Meso Guard, Mana Reflection, Achilles, High Defense, Combo Barrier, Magic Shield, the GUARD suppression rule, block-sentinel handling, and anti-cheat clamps — plus the mandatory packet-decoder fix. The battleship TODO stays (task-153).

## Key files

| File | Role |
|---|---|
| `libs/atlas-packet/model/damage_taken_info.go` | serverbound decoder — Task 1 rewrites (conditional extension, mob-branch fix, renames) |
| `services/atlas-channel/atlas.com/channel/socket/handler/character_damage.go` | the handler being reworked (Task 7) |
| `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go:78-120` | `damageInfoEntryDeps`/`processDamageInfoEntry` — the deps-struct pattern this task mirrors |
| `services/atlas-channel/atlas.com/channel/character/buff/` | buff processor; `GetByCharacterId` is REST-per-call; `Model.Changes() []stat.Model{Type() string, Amount() int32}`, `Expired()` |
| `services/atlas-channel/atlas.com/channel/character/skill/processor.go` | `GetByCharacterId`, package-level `GetLevel(skills, id)` helper |
| `services/atlas-channel/atlas.com/channel/data/skill/` | skill data + `GetEffect(skillId, level)`; Task 4 adds the tenant cache; `effect.Model.X() int16`, `Prop() float64` |
| `services/atlas-channel/atlas.com/channel/monster/` | live monster (`GetById` → `Hp()/MaxHp() uint32`); `Damage(f, monsterId, characterId, damages, attackType)` producer |
| `services/atlas-data/atlas.com/data/monster/{reader,rest}.go` | template ingestion — Task 2 adds `fixedDamage`; `Boss` already at rest.go:20 |
| `services/atlas-character/.../kafka/message/character/kafka.go:22,127-132` | `REQUEST_CHANGE_MESO` consumer contract (verified) |
| `services/atlas-channel/.../socket/model/damage_taken_info.go` | DEAD CODE — deleted in Task 7 |
| `libs/atlas-constants/character/temporary_stat.go` | all roster stat types exist (MAGIC_GUARD:15, POWER_GUARD:18, MESO_GUARD:34, MANA_REFLECTION:44, INFINITY:47, COMBO_BARRIER:76, MAGIC_SHIELD:85, GUARD:104) |
| `libs/atlas-constants/skill/constants.go` | all roster skill ids exist except Divine Shield (see blocker) |
| `libs/atlas-packet/test/` | `pt.Variants`, `pt.RoundTrip` (asserts zero unconsumed bytes — the over-read guard) |

## Decisions locked during planning

1. **Extension decode is length-derived** (`r.Available() > 1`), NOT the design's `nX != 0 || blockByte != 0`. IDA-verified on v83 (0x9581a9) and v95 (0x9343c0): the client gates the 14-byte extension on `bKnockback || nX`, and `bBlocked` (block byte) occurs without knockback for blocked mob-skill attacks (`bKnockback = pInfo == 0`), while vehicle-riding blocks set knockback without the block byte. Design §2's stated condition over/under-reads on those edges; FR-3.4 (binary wins) applies.
2. **Mob branch is `attackIdx >= -1`** and positive values are mob skill-attack slot indices. The non-mob branch emits −2 (`flag==0` case) or −3 (obstacle) — never −4 on any verified version. Handler classifies `<= -2` as non-mob (Meso/Power/Mana never apply there; Achilles/Combo/MagicShield/MagicGuard do).
3. **Power Guard version gates (plan-phase IDA discovery, not in design.md):** cap divisor = templateMaxHP/**10** on v83 (line 423) / v87 (line 414) / jms185 (line 476), but /**2** on GMS v95 (line 1109). Template `fixedDamage` is `min()`d pre-BB but **replaces** the reflect on v95/jms. Gates resolved in the orchestrator: `pgCapDivisor`, `pgFixedDamageOverride`.
4. **Divine Shield constant is NOT shipped.** The GUARD-suppression behavior needs no skill id (it keys off the GUARD temporary stat, granted by the buff domain). Verifying the id from v95 WZ is impossible in this environment — checked exhaustively on 2026-07-10: local dumps are v83-era (Cosmic, ms_1172); live atlas-data v95 tenant (`c794c706-aea3-4882-90a6-a3b7ee314f52`) has zero skill documents (2001002 404s); MinIO `atlas-wz` bucket holds only `shared/regions/GMS/versions/{83.1,84.1}/`. When v95 WZ lands, add `PaladinDivineShieldId` to `libs/atlas-constants/skill` from that data — never from memory.
5. **Body Pressure ships nothing here** (design §1 correction): it is a separate serverbound touch-attack packet already handled by `CharacterTouchAttackHandle`. Its TODO is removed with the handler rewrite.
6. **Mana Reflection proc**: client rolls prop; server honors the validated wire signal (reflect echo without isPowerGuard on attackIdx ≥ 0, MANA_REFLECTION buff active) and recomputes the amount (x% from effect, cap maxHp/20). Forged signals are warn-logged and ignored.
7. **Meso Guard** resolves affordability from the fetched character's `Meso()` inside the pure chain (partial guard `100*meso/x`, verified `CalcDamage` v83 0x792FA8); the deduction goes out as `REQUEST_CHANGE_MESO` with `actorType="SKILL"`, `actorId=characterId`, negative amount. The channel `Command` envelope has no TransactionId — same as the shipping `REQUEST_DROP_MESO` path.
8. **Reflect emission**: `monster.Damage` with attackType 0 (PG, physical) / 2 (MR, magic) — the values atlas-monsters' `checkReflect` distinguishes. `EmitDamageReflected` is NOT used (its consumer applies character-side HP loss — the opposite direction).
9. **Skill fetch is conditional**: passives only exist for jobs 112/122/132/2112, so `getSkills` is called only for those (base character fetch stays undecorated).
10. **Buff lookup failure fails open to unmitigated damage** (never leaves a hit unapplied); reflect-target lookup failure drops only the reflect (debug log), keeping character-side mitigation.
11. **fixedDamage rollout**: tenants ingested before the atlas-data change serve `fixed_damage: 0` (cap doesn't bind) until re-ingested. Graceful; no migration.

## Dependencies between tasks

Task 1 (packet) is independent. Task 2 (atlas-data) → Task 3 (channel template client, needs `fixed_damage` in the payload). Tasks 3/4/5 are independent of each other; all feed Task 7. Task 6 (pure math) only needs Task 1's `DamageType` constants. Task 7 needs 1+3+4+5+6. Task 8 is the sweep.

## Verification environment notes

- IDA instances (ida-pro MCP): v83 = port 13342 (`MapleStory_dump.exe`), v87 = 13343, v95 = 13341, jms185 = 13344. `CUserLocal::SetDamaged`: v83 0x9581a9, v87 0x9da6f2, v95 0x9343c0, jms 0xa228f8.
- Test-value grounding (v83 Skill.wz, Cosmic dump): Achilles/High Defense x per-mille 995→850; Magic Guard x percent 11→80; Meso Guard x (cost rate) 90→81; Mana Reflection x 55→140, prop 31+; Combo Barrier x per-mille 916→864.
- Changed modules for CI/bake: `libs/atlas-packet`, `services/atlas-data`, `services/atlas-channel` → `docker buildx bake atlas-channel atlas-data` both required.

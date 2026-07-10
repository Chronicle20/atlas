# Combo Drain (Aran) — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-07-10
---

## 1. Overview

Combo Drain (Aran 2nd-job skill, id `21100005`, introduced with the Aran job in GMS v84) is a self-buff that, while active, restores the caster's HP by a percentage of the damage they deal with attacks. The buff side of the skill already works end-to-end in Atlas: casting the skill flows through the generic skill handler, atlas-data's skill reader produces a `COMBO_DRAIN` temporary-stat statup carrying the skill effect's `x` value (`services/atlas-data/atlas.com/data/skill/reader.go:372-373`), and the buff is applied/rendered like any other. What is missing is the attack-side effect: the damage handler in atlas-channel never checks for the active buff, so the heal never happens. The gap is marked by `// TODO Combo Drain` at `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go:420`.

Reference behavior is Cosmic `AbstractDealDamageHandler.java:421-431`: when `BuffStat.COMBO_DRAIN` is present, the attacker is healed `totalDamage * x / 100`. Note that Cosmic evaluates this inside its per-monster loop against a *running* damage total, which over-heals multi-target attacks (a Cosmic quirk, not retail semantics). Per owner decision, Atlas implements the semantically correct version: one heal per accepted attack, computed from the total damage across all monsters hit.

This is a small, single-service change: read the attacker's active buffs (an API the attack pipeline already uses for projectile attacks), and if a `COMBO_DRAIN` stat is present, emit an HP change for the computed heal.

## 2. Goals

Primary goals:
- An Aran with Combo Drain active recovers HP equal to `x`% of the total damage of each attack they land, where `x` is the buff's `COMBO_DRAIN` statup amount.
- The heal is emitted once per attack (not per monster, not per hit line).
- No behavior change for characters without the buff, and no change to the attack pipeline's existing broadcast/damage/proc ordering.

Non-goals:
- Combo-orb consumption or any Aran combo-counter mechanics (buff-activation side; out of scope).
- Any of the sibling TODOs in the same block (Energy Drain/Vampire/Drain, Mortal Blow, charges, etc.).
- Packet/writer changes — the HP change flows through the existing character stat-update path.
- Replicating Cosmic's per-monster running-total quirk.

## 3. User Stories

- As an Aran player with Combo Drain active, I want my HP to visibly recover as I attack monsters so that the skill functions as described in-game.
- As a player without Combo Drain, I want attack handling to behave exactly as before so that unrelated combat is unaffected.
- As a server operator, I want the heal to be derived from tenant skill data (the buff statup), not hard-coded percentages, so that WZ-driven customization keeps working.

## 4. Functional Requirements

FR-1 — Buff detection
- After an attack is accepted and damage has been applied (at the `character_attack_common.go:420` TODO site), fetch the attacker's active buffs via `buff.NewProcessor(l, ctx).GetByCharacterId(characterId)` (same API used by `character_attack_projectile.go:97`).
- The effect triggers when any active, non-expired buff carries a stat change with `Type() == string(character.TemporaryStatTypeComboDrain)` (atlas-constants `TemporaryStatTypeComboDrain = "COMBO_DRAIN"`).
- If the buff-fetch fails, log a warning and skip the heal; the rest of the attack pipeline must be unaffected (same failure posture as MP Eater, see `character_attack_common.go:348-355`).

FR-2 — Heal computation
- `totalDamage` = sum of all values in `di.Damages()` across every entry of `ai.DamageInfo()` for the attack.
- `healPercent` = the `Amount()` of the matching `COMBO_DRAIN` stat change (this is the skill effect's `x`, already populated by atlas-data; verified WZ values for 21100005: x = 1..5 over levels 1..20).
- `heal = totalDamage * healPercent / 100`, integer arithmetic.
- If `totalDamage == 0` or the computed heal is `<= 0`, do nothing (no zero-amount HP command).
- Clamp the computed heal to `math.MaxInt16` before conversion — `ChangeHP` takes `int16` and a large attack total must not overflow/wrap.

FR-3 — Heal application
- Emit the heal via the existing `character.Processor.ChangeHP(f field.Model, characterId uint32, amount int16)` (`services/atlas-channel/atlas.com/channel/character/processor.go:271`), which produces the Kafka `ChangeHP` command; downstream clamping to max HP is owned by atlas-character.
- The heal applies for any attack type (melee/ranged/magic/energy) — the gate is the buff alone, with no job or attack-type check.
- Ordering: evaluate after per-monster damage processing (so `DamageInfo` is final) and independent of broadcast success, consistent with the other post-damage effects in the handler.

FR-4 — No regression
- Characters without the `COMBO_DRAIN` stat active take the existing code path with no additional Kafka emissions and at most one added buff lookup (see NFR on avoiding duplicate fetches).

## 5. API Surface

No new or modified REST endpoints. No new Kafka topics or message shapes — the feature reuses the existing character `ChangeHP` command and the existing atlas-buffs REST read (`buff/requests.go`). No packet/opcode/template changes.

## 6. Data Model

No schema or entity changes. All required data already exists:
- `libs/atlas-constants/skill/constants.go:3394` — `AranStage2ComboDrainId = Id(21100005)`.
- `libs/atlas-constants/character/temporary_stat.go:75` — `TemporaryStatTypeComboDrain`.
- atlas-data skill reader already emits the `COMBO_DRAIN` statup with amount = effect `x` (`skill/reader.go:372-373`).

## 7. Service Impact

- **atlas-channel** (only service touched): implement the Combo Drain block in `socket/handler/character_attack_common.go`, replacing the line-420 TODO. Reuse the buff processor already imported by the attack pipeline. Add unit tests.
- atlas-character, atlas-buffs, atlas-data: consumed as-is, no changes.

## 8. Non-Functional Requirements

- **Multi-tenancy:** all reads/emits use the request `ctx` (tenant-scoped) as the surrounding handler already does; no tenant-specific literals.
- **Performance:** at most one buff lookup per attack for this feature. If the handler variant already fetched buffs for the same attack (projectile path), prefer reusing that result over a second fetch; do not fetch when a cheap short-circuit shows it cannot apply only if such a short-circuit is already available — otherwise a single `GetByCharacterId` per attack is acceptable (it is already the per-attack cost on the projectile path).
- **Resilience:** buff-service failure degrades to "no heal" with a logged warning; it must never fail the attack handler.
- **Observability:** debug-level log when a Combo Drain heal is emitted (characterId, totalDamage, percent, heal).

## 9. Open Questions

None — scope decisions resolved in the spec interview (2026-07-10): semantically-correct single heal per attack; percent sourced from the buff statup amount; buff-only gate (all attack types); no cap beyond the natural max-HP clamp plus the `int16` conversion clamp.

## 10. Acceptance Criteria

- [ ] With a `COMBO_DRAIN` buff of amount `x` active, an accepted attack dealing total damage `D` (summed over all monsters and hit lines) emits exactly one `ChangeHP` command for `min(D*x/100, math.MaxInt16)` when that value is `> 0`.
- [ ] Multi-target attacks heal on the plain total (no Cosmic running-total over-heal).
- [ ] No `ChangeHP` emission when the buff is absent, expired, when `D*x/100 == 0`, or when the buff lookup errors (warning logged instead).
- [ ] Works for melee, ranged, magic, and energy attack types.
- [ ] Unit tests cover: buff present (single + multi monster), buff absent, zero damage, `int16` clamp boundary, buff-fetch error. Tests use the project Builder pattern (no `*_testhelpers.go`).
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...` clean in atlas-channel; `tools/redis-key-guard.sh` clean; `docker buildx bake atlas-channel` succeeds.
- [ ] The `// TODO Combo Drain` marker is removed; `docs/TODO.md` line item checked off.

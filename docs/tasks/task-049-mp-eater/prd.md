# MP Eater Passive — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-05-03
Parity reference: `~/source/Cosmic` (`MagicDamageHandler.java:85-91`, `StatEffect.java:882-904`, `StatEffect.java:1811-1813`)

---

## 1. Overview

Magicians at 2nd job receive a passive skill called MP Eater (Fire/Poison Wizard `2100000`, Ice/Lightning Wizard `2200000`, Cleric `2300000`). On every magic attack, the skill rolls its `prop` (chance) once per damaged monster; on a successful roll, a fraction of the monster's MaxMP is drained from its current MP and refunded to the caster, accompanied by a "skill special" visual effect on the caster and to onlookers.

Today the three skill IDs exist in `libs/atlas-constants/skill/constants.go` and are assigned to the matching jobs in `libs/atlas-constants/job/constants.go`, but the trigger is unimplemented. The hook point exists as a `// TODO Apply MPEater` comment at `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go:278`, immediately after the per-monster damage application loop.

Heal vs. undead is the user-facing motivator for this work: a Bishop healing a Wraith dispatches the same `processAttack` flow (since Heal is "dual-packet" — damage half goes through `processAttack`, party-heal half through the per-skill handler `skill/handler/heal/heal.go`), so wiring MP Eater into `processAttack` automatically covers the Heal-vs-undead case.

---

## 2. Goals

Primary goals:
- Implement MP Eater for all three variants (FP Wizard, IL Wizard, Cleric/Priest/Bishop) with Cosmic-parity mechanics.
- Integrate at the existing TODO site in `processAttack` so it fires for every magic attack, including Heal vs. undead.
- Drain the monster's MP, refund the caster's MP, and broadcast the "skill special" visual to caster and same-map sessions.
- Provide unit-test coverage of the chance roll and drain math via injected/seeded RNG.

Non-goals:
- Implementing the other passive/attack-side TODOs in the same block (Combo Drain, Pick Pocket, Energy Drain, Vampire, Mortal Blow, Hamstring, Slow, Blind, Paladin charges, etc.) — each is a separate task.
- Restructuring Heal's dual-packet architecture.
- Adding new effect-data fields — the existing `Prop` and `X` fields on `effect.Model` are sufficient (parity confirmed against Cosmic).
- Atlas UI changes.
- Cooldown, per-mob exclusion list, or anti-farm rate limiting beyond what Cosmic does.

---

## 3. User Stories

- As a Bishop healing undead (e.g., Wraiths), I see MP Eater proc and refund a portion of my MP, the same as on any other magic attack.
- As a 2nd-job magician (Fire/Poison Wizard, Ice/Lightning Wizard, Cleric), every magic attack I land has the documented chance per monster to drain MP — no Bishop-special path.
- As a player attacking a boss monster, MP Eater does **not** trigger (Cosmic parity — anti-farm).
- As a player attacking a monster with no MP pool (`MaxMP == 0`) or already drained dry (`current MP == 0`), MP Eater does **not** trigger and does **not** play its visual.
- As a player attacking a monster with a multi-line magic skill (e.g., Ice Strike), MP Eater rolls **once per monster hit**, not once per damage line.

---

## 4. Functional Requirements

### 4.1 Skill ID resolution (caster-side)

- Derive the candidate MP Eater skill id from the caster's job using the Cosmic formula: `(jobId - jobId%10) * 10000`.
  - Bishop (job 232) → `(232 - 2) * 10000 = 2300000` (Cleric MP Eater).
  - Priest (231) → `2300000`. Cleric (230) → `2300000`. ILWizard (220) → `2200000`. FPWizard (210) → `2100000`. Mages 3rd/4th job inherit appropriately.
- Reject candidate ids not present in the skill registry (`libs/atlas-constants/skill/constants.go`). 1st-job Magician (job 200) computes to `2000000`, which does not exist; that path must be a no-op.
- If the caster has skill level `0` for the candidate id (i.e., did not learn it), do not roll, do not visual.
- The caster's skill level determines which `effect.Model` is fetched for the roll (`prop`, `X`).

### 4.2 Trigger conditions

MP Eater is evaluated only when **all** of the following are true:

1. The current attack is processed by `processAttack` (covers all magic attack types, including Heal vs. undead). It is **not** evaluated for melee, ranged, or energy attacks.
2. The caster owns a non-zero level of the resolved MP Eater variant for their job.
3. The damage application for the monster (`mp.Damage`) was attempted (i.e., the per-target loop reached the end of its successful path; reflect-bounced entries are skipped).
4. The target is a monster (not a reactor, NPC, or player).
5. The target is **not a boss**. (Cosmic parity, anti-farm.)
6. The target's `MaxMP > 0` and current `MP > 0`.

If any condition fails, do not roll, do not visual, do not emit.

### 4.3 Chance roll

- Use the `Prop` field of the resolved skill effect (already a `float64` in `[0.0, 1.0]`).
- Cosmic logic: `prop == 1.0 || rand() < prop`. Mirror exactly.
- One roll per **monster** (not per damage line). Multi-line magic skills (e.g., Ice Strike's two lines per target) get exactly one MP Eater roll per affected monster.
- RNG must be injectable for test seeding. The chosen RNG injection mechanism must follow whatever pattern already exists in `processAttack` (e.g., the venom DPT path uses `rand.Float64()` directly; a thin wrapper or test seam is acceptable so long as it does not regress that path).

### 4.4 Drain calculation

- `absorbMp = min(monster.MaxMP * X / 100, monster.currentMP)` where `X` is the skill effect's X value (the documented "absorb percentage" — at Cleric MP Eater max level in v83, X = 10).
- If `absorbMp == 0`, skip — no monster mutation, no caster MP refund, no visual.
- Player MP is increased by `absorbMp` and clamped to the player's effective MaxMP (consistent with the existing `cp.ChangeMP` clamp behavior used by Heal and skill costs).
- Monster MP is decreased by `absorbMp`. Monster MP must not underflow below zero.

### 4.5 Monster MP mutation

- Channel emits a new Kafka command on the monster command topic (`COMMAND_TOPIC_MONSTER`) — proposed name `DRAIN_MP` (final string TBD in design phase) — containing tenant headers, world/channel/map/instance, monster unique id, source character id, source skill id, and absorb amount.
- atlas-monsters consumes the command, applies the decrement to current MP with a non-negative clamp, and persists/emits whatever state-change event it normally emits for HP/MP changes (parity with the existing `Damage` consumer's MP/HP path — exact mechanism TBD in design).
- Boss check **may** live in either atlas-channel (using the monster snapshot already loaded for reflect) or atlas-monsters (defense-in-depth). Final placement TBD in design phase. The visible behavior must be: bosses receive no MP drain, no visual.

### 4.6 Caster MP refund

- Use the existing `character.Processor.ChangeMP(field, characterId, +int16(absorbMp))` path that `processAttack`'s generic cost block and `heal.go` use for HP/MP delta.
- `absorbMp` will fit in `int16` for any realistic monster MaxMP × max X% (X max is 10 on Cleric MP Eater; even a 100k-MaxMP boss would yield 10k, safely within int16). Bosses are excluded by 4.2.5 anyway.

### 4.7 Visual effect

- On successful drain (absorbMp > 0), broadcast a "skill special" effect:
  - Caster: `CharacterSkillSpecialEffectBody(skillId)` (already exists in `libs/atlas-packet/character/effect_body.go`, mode `SKILL_SPECIAL`).
  - Same-map foreign sessions: `CharacterSkillSpecialEffectForeignBody(characterId, skillId)`.
- Use the existing `socketHandler.AnnounceSkillUse`-style broadcast helpers for consistency with `heal.go`.
- The visual must fire only on actual drain (not on a roll-failure, not on `MaxMP == 0` skip, not on boss skip).

### 4.8 Ordering inside `processAttack`

- MP Eater proc runs **after** `mp.Damage(...)` for a given `DamageInfo` and **after** the existing `ApplyStatus` block (matches Cosmic, which calls `applyAttack` then `applyPassive` per target).
- The per-monster `for _, di := range ai.DamageInfo()` loop already collapses to one iteration per monster in this codebase, so the natural placement is inside that loop, immediately after the status-apply block (around line 215 today, with the `// TODO Apply MPEater` comment relocated/removed accordingly).
- Failures in MP Eater (RNG, monster lookup, Kafka emit) must log and continue — they must never abort the rest of the attack pipeline.

### 4.9 Heal-vs-undead interaction

- No special-casing. Heal's damage half routes through `processAttack` like any other magic attack, so MP Eater triggers naturally on Wraiths and other undead that take Heal damage.
- The "is target undead" check is already performed by the existing damage path (or rather, by the client/data layer that produced the `DamageInfo`). MP Eater trusts that — if `mp.Damage` was attempted on a target, the target is a valid magic-attack victim.

---

## 5. API Surface

No new REST endpoints. The change is internal to the channel attack pipeline plus one new monster command.

### 5.1 New Kafka command (atlas-monsters)

- **Topic:** `COMMAND_TOPIC_MONSTER` (existing).
- **Type constant:** new — proposed `CommandTypeDrainMp = "DRAIN_MP"` in `services/atlas-channel/atlas.com/channel/kafka/message/monster/kafka.go` (and matching consumer constant in atlas-monsters).
- **Body fields (proposed):**
  - tenant headers (existing envelope).
  - `WorldId`, `ChannelId`, `MapId`, `Instance` (envelope, existing pattern).
  - `MonsterId` — `uint32`, monster unique id.
  - `CharacterId` — `uint32`, caster.
  - `SkillId` — `uint32`, MP Eater skill id (for downstream observability / event source).
  - `Amount` — `uint32`, absorb amount the channel computed.
- **Consumer behavior (atlas-monsters):**
  - Look up monster by id; ignore if missing.
  - Decrement `Mp` by `Amount`, clamp at zero.
  - Persist via the same registry mutation path used by HP/MP changes from `Damage`.
  - Emit whatever monster MP/state-change event currently fires for damage (e.g., monster status change, `MonsterStatChanged` if one exists). Final event shape TBD in design phase to match existing patterns.

The exact field set, type names, and event emission shape are design-phase decisions. The PRD requires only that the wire format is consistent with the existing monster-command envelope and that the consumer mutation matches the precedent set by `Damage`.

---

## 6. Data Model

No persistent schema changes. All state involved (player MP, monster MP) is already modeled and persisted by their respective services.

In-memory: no new registries, caches, or shared state. The MP Eater check is stateless per-attack and does not introduce per-character or per-mob locks.

---

## 7. Service Impact

### 7.1 atlas-channel (primary)

- `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go`
  - Replace `// TODO Apply MPEater` (line 278) with the actual implementation.
  - Add per-target MP Eater evaluation inside the `for _, di := range ai.DamageInfo()` loop, after the existing `ApplyStatus` block (around line 215).
  - Add a small helper that resolves the MP Eater skill id from job (`(jobId - jobId%10) * 10000`), guarded by registry lookup.
- `services/atlas-channel/atlas.com/channel/monster/processor.go`
  - Add `DrainMp(f, monsterId, characterId, skillId, amount)` that emits the new command via `producer.ProviderImpl(...)`.
- `services/atlas-channel/atlas.com/channel/kafka/message/monster/kafka.go`
  - Add `CommandTypeDrainMp` constant and the corresponding command body struct.
- New broadcaster (or reuse an existing one) for `CharacterSkillSpecialEffectBody` / `CharacterSkillSpecialEffectForeignBody`. The packet writers already exist; only a thin call site is needed.
- Unit tests: new `_test.go` file or extension of the existing attack-handler tests, exercising:
  - Roll success / failure (seeded RNG).
  - Drain math (current-MP cap, MaxMP-percent cap).
  - Boss skip.
  - `MaxMP == 0` and `current MP == 0` skip.
  - 1st-job Magician (no-op).
  - Wrong-job (e.g., Fighter) → no roll.

### 7.2 atlas-monsters

- New consumer handler for `DRAIN_MP` command.
- Registry mutation: decrement `Mp` with non-negative clamp.
- State-change event emission consistent with the existing damage path.
- Unit tests for drain command consumer (clamp at zero, missing monster, etc.).

### 7.3 atlas-character

- No code changes. The existing `character.Processor.ChangeMP` is reused for caster MP refund.

### 7.4 libs/atlas-constants

- No changes — skill IDs, job assignments, and `Prop`/`X` accessors already exist.

### 7.5 libs/atlas-packet

- No changes — `CharacterSkillSpecialEffectBody` and `CharacterSkillSpecialEffectForeignBody` already exist (`character/effect_body.go:122-129`).

### 7.6 atlas-data

- No changes assumed — the existing skill effect endpoint returns `Prop` and `X` for all three MP Eater skills. Validation that real data is present and correct for skill ids `2100000`, `2200000`, `2300000` at all known levels is part of acceptance.

---

## 8. Non-Functional Requirements

### 8.1 Performance
- One additional Kafka emit per successful proc (small fraction of attacks). Roll itself is `rand.Float64()` plus arithmetic, negligible.
- No new synchronous network calls in the hot path beyond the existing monster-command emit pattern.
- No new locks, no new goroutines.

### 8.2 Multi-tenancy
- All emitted commands carry tenant headers via the existing envelope. The drain consumer must scope its monster lookup and mutation to the tenant in the headers, mirroring the `Damage` consumer.

### 8.3 Observability
- Debug log on each proc (skill id, caster, monster, absorb amount, monster current/max MP before drain). Match the verbosity used by `mp.Damage` (`Debugf`).
- Errors (Kafka emit failure, etc.) at `Errorf`, never panic.
- No new metrics required for v1; existing OTel span around `processAttack` remains the parent.

### 8.4 Security / abuse
- No client-trusted fields. The channel computes `absorbMp` server-side from server-side monster MP and skill data. Client cannot inflate the drain.
- Boss exclusion is server-side only.

### 8.5 Determinism / testability
- RNG must be injectable for unit tests (seeded). Dice-roll path must be exercised in both success and failure branches.

---

## 9. Open Questions

1. **Boss check placement** — atlas-channel (cheap, since the monster snapshot is already loaded for reflect handling on melee/ranged but not currently for magic) vs. atlas-monsters (single source of truth, but requires emitting a command that may be dropped). Resolved in design phase.
2. **DrainMp event emission shape** — what state-change event(s) atlas-monsters fires after a drain. Match existing damage-path patterns. Resolved in design phase.
3. **RNG injection mechanism** — does this codebase already have an RNG seam or do we add one? Only the venom-DPT path in `processAttack` uses `rand.Float64()` directly today, so a thin wrapper may be cleanest. Resolved in design phase.
4. **Visual broadcast helper** — reuse one of the existing broadcasters (e.g., `socketHandler.AnnounceSkillUse`) or add a dedicated `AnnounceSkillSpecial` helper. Resolved in design phase.
5. **Other passives in the same block** — separate TODO items (Combo Drain, Energy Drain, Vampire, Pick Pocket, Mortal Blow). Out of scope for this task; track separately so the same RNG/visual scaffolding can be reused.

---

## 10. Acceptance Criteria

- [ ] Bishop heals Wraiths in-game; on roll success, MP visibly refunds and the SKILL_SPECIAL effect plays for the caster and onlookers.
- [ ] FP Wizard / IL Wizard / Cleric / Priest casting any normal magic attack against a non-boss MP-bearing monster sees the same behavior at the documented rate.
- [ ] Boss monster (e.g., Pap, Zakum body, Horntail head, mini-bosses with `Boss == true`) never receives a drain or visual, regardless of roll.
- [ ] Monster with `MaxMP == 0` (e.g., low-level slimes/snails) never receives a drain or visual.
- [ ] Monster already at `MP == 0` never receives a drain or visual.
- [ ] Multi-line magic skill (Ice Strike, etc.) produces at most one MP Eater proc per affected monster per cast.
- [ ] 1st-job Magician (job 200) and any non-magician job casting magic attacks (e.g., Cleric Holy Arrow if used by a Bishop equivalent — n/a; or any non-magician casting an item-derived magic effect) trigger no MP Eater logic.
- [ ] Unit tests cover: roll success, roll failure (seeded RNG), drain-amount math (MaxMP-percent cap, current-MP cap), boss skip, MaxMP-zero skip, current-MP-zero skip, 1st-job no-op, missing-skill-effect no-op, Kafka emit failure logged-and-continued.
- [ ] atlas-monsters DrainMp consumer: clamps at zero, ignores missing monster, mutates registry consistent with existing Damage path. Unit-tested.
- [ ] No regressions in existing Heal, magic-attack, reflect, or status-apply flows. Existing tests still pass.
- [ ] `// TODO Apply MPEater` removed from `character_attack_common.go:278`. `docs/TODO.md:90` (`- [ ] Apply MPEater`) checked off or removed.
- [ ] Builds for atlas-channel and atlas-monsters succeed; tests for both pass.

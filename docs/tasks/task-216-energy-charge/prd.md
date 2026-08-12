# Energy Charge — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-12

---

## 1. Overview

Energy Charge is the defining mechanic of the Pirate melee line: Marauder
(`5110001`) and its Cygnus mirror, Thunder Breaker Stage 2 (`15100004`). Every
monster a qualifying character hits pushes their energy bar up; when the bar
fills, the character enters a visibly *charged* state for a fixed window, gains
a weapon-attack bonus, and unlocks Energy Blast. When the window lapses the bar
empties and accumulation starts over.

Atlas today has none of this. `grep -rE "EnergyCharge|ENERGY_CHARGE"` finds only
declarations, never behaviour: the temporary-stat constant
(`libs/atlas-constants/character/temporary_stat.go:116`), its two-state wire
encoding (`libs/atlas-packet/model/character_temporary_stat.go:1192`), and the
`CharacterAttackEnergy` broadcast writer
(`libs/atlas-packet/character/clientbound/attack.go:19`) — which broadcasts an
*energy-type attack*, an unrelated concern. No accumulation, no charged state,
no reset exists in atlas-channel, atlas-buffs, or atlas-effective-stats. For a
player the bar simply never moves, so a Marauder or Thunder Breaker plays as a
strictly worse melee class with a dead skill in their book.

The mechanic is a near-exact structural twin of Combo Attack orbs, which Atlas
already implements end to end (`character_attack_combo.go`,
`buff.UpdateStatValue` → `UPDATE_STAT_VALUE` with `INCREMENT` + `Cap`). This
task ports Energy Charge onto that same spine: accumulate a capped stat on a
buff in atlas-buffs, broadcast each change through the existing buff-status
consumer, and add the two things Combo does not need — a terminal *charged*
phase with its own timer, and a stat payoff sourced from the skill effect.

## 2. Goals

Primary goals:

- The energy bar fills by 102 per attacked monster, clamped to 10000, for any
  character who owns Energy Charge at level > 0 and is in the Marauder or
  Thunder Breaker Stage 2+ job line.
- On reaching 10000 the character enters the charged state (bar value 15000)
  for the skill effect's duration, then resets to 0 and is broadcast as such.
- Every bar change is reflected to the owning client and to observers in the
  map, so the client renders the bar and the charged aura.
- While charged, the character gains the skill effect's `pad` as weapon attack
  in atlas-effective-stats.
- Energy Blast casts are rejected server-side unless the caster is charged.
- Energy state survives whatever the underlying buff store survives (channel
  change, map change), because it *is* a buff.

Non-goals:

- Cosmic's `calcDmgMax` damage-ceiling multiplier
  (`AbstractDealDamageHandler.java:717-720`). That value exists only to feed
  Cosmic's autoban damage check; Atlas performs no equivalent server-side damage
  validation on these attacks, so there is nothing for the multiplier to scale.
- Consuming the bar on a charged cast. The bar resets on its timer only.
- Any change to Buccaneer Energy Orb (`5121002`), Energy Drain (`5111004` /
  `15111001`), or the `CharacterAttackEnergy` broadcast writer.
- Body Pressure (`// TODO BodyPressure`,
  `character_attack_common.go:1052`). It shares the touch-attack path but is a
  separate skill and a separate task.
- gms_v12 and gms_v48 support. Pirates postdate both; see §7.4.

## 3. User Stories

- As a Marauder, I want my energy bar to fill as I hit monsters so that the
  class plays the way its skill description promises.
- As a Marauder at a full bar, I want the charged aura to appear on my character
  and on other players' screens so that the state is legible to me and my party.
- As a Marauder, I want a measurable payoff while charged — extra weapon attack
  and access to Energy Blast — so that filling the bar is worth doing.
- As a Marauder, I want the bar to drain on its own after the charged window so
  that the cycle repeats without me managing it.
- As a Thunder Breaker, I want all of the above from my own skill line
  (`15100004`) without a separate implementation quirk.
- As a Marauder who changes channel mid-fight, I want my energy state to follow
  me rather than silently resetting.

## 4. Functional Requirements

### FR-1 — Eligibility

- **FR-1.1** A character is *energy-eligible* when they own the Energy Charge
  skill for their line at level > 0:
  - `skill.MarauderEnergyCharge` (5110001) for the adventurer Pirate line, or
  - `skill.ThunderBreakerStage2EnergyCharge` (15100004) for the Cygnus line.
- **FR-1.2** Line selection follows the character's job identity, mirroring
  Cosmic's `isCygnus()` split: `job.Marauder` (511) and above for the
  adventurer line, `job.ThunderBreakerStage2` (1510) and above for Cygnus. A
  character owning neither skill is not eligible and every requirement below is
  a no-op for them.
- **FR-1.3** Neither skill id appears in
  `docs/tasks/task-187-version-aware-id-semantics/audit/divergences.csv`, so
  both are version-stable across the provisioned versions. Comparisons must
  carry the same `// version-stable per task-187 audit` annotation
  `character_attack_combo.go:33-36` uses. The job identities must still be
  resolved through the version-aware resolver where the surrounding code does
  so.

### FR-2 — Accumulation

- **FR-2.1** For each qualifying attack, the bar gains **102 per attacked
  monster** — i.e. `102 × len(ai.DamageInfo())`, matching Cosmic's
  `for (int i = 0; i < attack.numAttacked; i++) chr.handleEnergyChargeGain()`
  (`CloseRangeDamageHandler.java:136-140`).
- **FR-2.2** The bar is clamped to 10000 during accumulation
  (`Character.java:6017-6021`).
- **FR-2.3** Qualifying attack sites:
  - All close-range/melee attacks, including basic attack (`skillId == 0`),
    per `CloseRangeDamageHandler.java:136-140`.
  - The touch-attack path (`character_attack_touch.go`, which builds an
    `AttackTypeEnergy` AttackInfo) — this is the Energy Charge aura's own
    damage and is routed through the same close-range handler in Cosmic.
  - Thunder Breaker Shark Wave (`15111007`) on the ranged path, and **only**
    that skill, per `RangedAttackHandler.java:90-99`.
- **FR-2.4** An attack that hit zero monsters grants nothing.
- **FR-2.5** No gain occurs while the character is already charged (bar at
  15000). Cosmic's `if (energybar < 10000)` guard makes both the increment and
  its broadcast conditional; the 15000 sentinel falls outside it.
- **FR-2.6** Accumulation bookkeeping must never fail an attack. Every error is
  logged and swallowed, matching `comboOrbTryUpdate`'s contract
  (`character_attack_combo.go:158-165`).

### FR-3 — Charged state

- **FR-3.1** When accumulation reaches exactly 10000, the character transitions
  to the charged state, represented by bar value **15000**
  (`Character.java:6030-6031`). 15000 is a sentinel, not a bar reading; nothing
  may treat it as "150% full".
- **FR-3.2** The charged state lasts the Energy Charge effect's duration at the
  character's skill level. Verified from WZ (`Skill.wz/511.img.xml`, node
  `5110001/level/<n>/time`): 31s at level 1, 40s at level 20. Note the WZ node
  is **seconds**; `effect.Model.Duration()` in atlas-channel already returns
  **milliseconds** (`data/skill/effect/model.go:80-85`), and
  `COMMAND_TOPIC_CHARACTER_BUFF`'s `duration` field is milliseconds. No scaling
  may be applied at the emit site — see `tools/buff-duration-guard.sh`.
- **FR-3.3** When the charged window lapses, the bar returns to 0 and the
  character is no longer charged.
- **FR-3.4** After reset, the next qualifying hit begins accumulation again from
  102.

### FR-4 — Client communication

- **FR-4.1** Each accumulation step announces the new bar value to the owning
  client as an `ENERGY_CHARGE` temporary stat. Cosmic sends
  `giveBuff(energybar, 0, [ENERGY_CHARGE])` per gain
  (`Character.java:6024`) — a stat update, not a new buff.
- **FR-4.2** Each accumulation step announces the skill-use effect to the owner
  and to the map (`showOwnBuffEffect(skillId, 2)` /
  `showBuffEffect(id, skillId, 2)`, `Character.java:6025-6026`).
- **FR-4.3** The bar value is mirrored to other characters in the map as a
  foreign buff (`giveForeignPirateBuff`, `Character.java:6027-6028`). Atlas
  already has `BuffGiveForeign`
  (`libs/atlas-packet/character/clientbound/buff_give.go`) and the channel's
  buff-status consumer already fans APPLIED/STAT_UPDATED out to observers
  (`kafka/consumer/buff/consumer.go:197`).
- **FR-4.4** The reset announces the zeroed bar to the owner and a foreign
  cancel of the `ENERGY_CHARGE` stat to the map
  (`cancelForeignFirstDebuff(id, 1<<50)`, `Character.java:6040`). Atlas has
  `BuffCancelForeign` (`buff_cancel.go:64`).
- **FR-4.5** No new opcode, writer, or template routing is introduced. The
  `ENERGY_CHARGE` two-state dynamic CTS entry already exists
  (`character_temporary_stat.go:1192`, `:1130`). Its per-version encoding must
  be confirmed rather than assumed — see AC-9.

### FR-5 — Charged payoff: weapon attack

- **FR-5.1** While charged (bar == 15000), the character gains the Energy Charge
  effect's `pad` value as weapon attack, per
  `Character.java:7676-7680` (`localwatk += ceffect.getWatk()`). Verified from
  WZ: `pad` is 0 at levels 1–3, 11 at levels 4–5, and 15 at level 20.
- **FR-5.2** The bonus is applied in atlas-effective-stats as a
  `weapon_attack` `stat.Bonus` sourced `buff:<skillId>`, alongside the existing
  buff-derived bonuses (`character/initializer.go:174-197`).
- **FR-5.3** `ENERGY_CHARGE` must **not** be mapped as a generic stat bonus.
  The stat's amount is the bar reading (0–15000), not an attack value; feeding
  it to `BonusesForBuffChange` directly would grant a five-digit stat. The
  charged bonus must instead be resolved from the skill effect's `pad` at the
  buff's level, gated on `amount == 15000`.
- **FR-5.4** atlas-data already serves both fields needed here — `pad` via
  `SetWeaponAttack` (`data/skill/reader.go:218`) and `damage` via `SetDamage`
  (`:268`). Only the consuming service's effect model needs the getters plumbed
  through; atlas-channel's `effect.Model` currently exposes neither
  (`data/skill/effect/model.go`).

### FR-6 — Charged-state cast gating

- **FR-6.1** A cast of Energy Blast is rejected unless the caster is charged
  (`ENERGY_CHARGE` value == 15000). This applies to `MarauderEnergyBlast`
  (5111002) and `ThunderBreakerStage2EnergyBlast` (15101005).
- **FR-6.2** Rejection follows the existing Enrage precedent exactly
  (`character_skill_use.go:114-132`): the cast is declined, no buff or effect is
  applied, and the reason is logged at Debug.
- **FR-6.3** The bar is **not** consumed by a successful charged cast. Only the
  FR-3.3 timer resets it.
- **FR-6.4** This is a deliberate divergence from Cosmic, which performs no
  server-side charge check at all. See §9 OQ-1 for the drift risk this
  introduces and the mitigation required before it ships.

### FR-7 — Touch-attack refresh guard

- **FR-7.1** An attack whose skill id *is* Energy Charge must not re-apply the
  Energy Charge effect to the attacker. Cosmic added this explicitly
  (`AbstractDealDamageHandler.java:183-184`: "thanks IxianMace for noticing
  Energy Charge skills refreshing on touch"). Without it the aura's own touch
  damage perpetually refreshes the charged window.
- **FR-7.2** The guard must not suppress the FR-2 *gain* — only effect
  re-application. Cosmic's touch damage still calls
  `handleEnergyChargeGain()`.

## 5. API Surface

No new REST endpoints.

Consumed, unchanged:

- `COMMAND_TOPIC_CHARACTER_BUFF` / `APPLY` — creates the accumulation buff on
  first gain and the charged buff at the transition.
- `COMMAND_TOPIC_CHARACTER_BUFF` / `UPDATE_STAT_VALUE` — `INCREMENT` with
  `Amount: 102 × mobsHit`, `Cap: 10000`, `StatType: "ENERGY_CHARGE"`.
- `EVENT_TOPIC_CHARACTER_BUFF_STATUS` / `APPLIED`, `STAT_UPDATED`, `EXPIRED` —
  drive the FR-4 broadcasts and the FR-3.1 phase transition.
- `GET /characters/{id}/buffs` (atlas-buffs) — read by atlas-effective-stats
  (FR-5) and by the FR-6 cast gate.
- `GET /skills/{id}/effects/{level}` (atlas-data) — supplies `time` (duration)
  and `pad`.

Message-shape changes: none anticipated. If the design phase concludes
atlas-buffs must own the phase transition (§9 OQ-2), a new command type would be
required and this section must be revised.

## 6. Data Model

No new persisted entities. Energy state is a buff row in atlas-buffs keyed by
`(tenant, characterId, sourceId)` where `sourceId` is the Energy Charge skill
id, carrying a single `ENERGY_CHARGE` stat change.

Two phases of the same buff:

| Phase | Trigger | `Duration` | `NoExpiry` | `ENERGY_CHARGE` amount |
|---|---|---|---|---|
| Accumulating | first qualifying hit with no existing buff | 0 | `true` | 102, incrementing to 10000 |
| Charged | amount reaches 10000 | effect duration (ms) | `false` | 15000 |
| (absent) | charged buff expires | — | — | — |

`NoExpiry` is the existing task-167 FR-2 flag; the command consumer already
rejects `NoExpiry` with a non-zero `Duration`
(`kafka/message/buff/kafka.go`). No migration is required — buffs are runtime
state.

## 7. Service Impact

### 7.1 atlas-channel (primary)

- New `character_attack_energy_charge.go` alongside
  `character_attack_combo.go`, holding line resolution, eligibility, the gain
  computation, and a `deps` struct so every branch is unit-testable without a
  live processor — the shape `comboOrbDeps` established
  (`character_attack_combo.go:137-156`).
- A call site in the attack pipeline next to
  `comboOrbTryUpdate(...)` (`character_attack_common.go:981`), covering the
  melee, touch, and Shark-Wave-only ranged paths per FR-2.3.
- The FR-7 touch-refresh guard on the effect-application path.
- The FR-6 Energy Blast cast gate in `character_skill_use.go`, modelled on the
  Enrage gate at `:114-132`.
- Phase-transition handling in the buff-status consumer
  (`kafka/consumer/buff/consumer.go:197`) if the transition is channel-driven
  (§9 OQ-2).
- `data/skill/effect/model.go`: expose the `pad` (weapon attack) getter if the
  channel needs it locally; `damage` is not needed given the §2 non-goal.

### 7.2 atlas-effective-stats

- `character/initializer.go` `fetchBuffBonuses`: special-case the
  `ENERGY_CHARGE` stat per FR-5.3 — resolve `pad` from the skill effect at the
  buff's level and emit a `weapon_attack` bonus, only when the amount is 15000.
  The service already fetches skill data for passives (`fetchPassiveBonuses`),
  so the lookup path exists.

### 7.3 atlas-buffs

- Expected to need no change: `UPDATE_STAT_VALUE`/`INCREMENT`/`Cap`,
  `NoExpiry`, and timed expiry with an `EXPIRED` event all already exist and
  are exercised by Combo Attack and task-167.
- Revisit only if §9 OQ-2 resolves toward buffs-owned phase transition.

### 7.4 Version scope

Derived from the per-version constant tables in `libs/atlas-constants/skill/`:

| Version | Marauder 5110001 | Thunder Breaker 15100004 |
|---|---|---|
| gms_v12 | absent | absent |
| gms_v48 | absent | absent |
| gms_v61 | present | **absent** |
| gms_v72 | present | present |
| gms_v79 | present | present |
| gms_v83 | present | present |
| gms_v84 | present | present |
| gms_v87 | present | present |
| gms_v92 | present | present |
| gms_v95 | present | present |
| jms_v185 | present | present |

gms_v12 and gms_v48 are **n-a**: Pirates postdate both. gms_v61 has the
adventurer line only — the Cygnus branch must degrade to a no-op there rather
than resolving a bogus id.

No socket-config template changes are anticipated (FR-4.5), so
`tools/template-opcode-order-guard.sh` and friends should stay untouched.

## 8. Non-Functional Requirements

- **NFR-1 (hot path)** Accumulation runs on every melee hit of an eligible
  character. It must add **at most one Kafka emit per attack** and **zero
  blocking REST calls** on the attack path — the same budget `comboOrbTryUpdate`
  keeps. Per-mob looping (FR-2.1) must collapse into a single emit of
  `102 × mobsHit`, not N emits.
- **NFR-2 (non-fatal)** No energy bookkeeping failure may reject, delay, or
  error an attack (FR-2.6).
- **NFR-3 (multi-tenancy)** All state is tenant-scoped via the existing buff
  path; no new tenant plumbing. Skill and job ids resolve through the
  version-aware constants surface.
- **NFR-4 (guards)** `tools/buff-duration-guard.sh` must stay clean — the
  charged-buff `duration` is passed through in milliseconds with no
  seconds→ms scaling. `tools/skill-job-id-guard.sh` must stay clean.
- **NFR-5 (observability)** Every swallowed error logs at Error with the
  character id and the skill line, matching the Combo messages
  (`character_attack_combo.go:175, 188, 195`).
- **NFR-6 (testing)** Pure helpers (eligibility, line resolution, gain amount,
  charged predicate, cast gate) are unit-tested through injected deps in the
  style of `character_attack_combo_test.go`. Test setup uses the project Builder
  pattern; no `*_testhelpers.go`.

## 9. Open Questions

- **OQ-1 — Cast-gate drift risk.** FR-6 diverges from Cosmic, which never
  checks the bar server-side. The client maintains its own bar from the CTS
  values we send; if server and client ever disagree (a dropped
  `STAT_UPDATED`, a reconnect before the buff replays), the server will reject a
  cast the client believed was legal, and the player loses the skill with no
  feedback. Design must decide: (a) reject silently as Enrage does, (b) reject
  and re-announce the authoritative bar value so the client resynchronises, or
  (c) gate only when a buff is present at all, tolerating value drift. **(b) is
  the recommendation** — it turns a dead-input bug into a self-healing one.
- **OQ-2 — Who owns the 10000 → 15000 transition?** Channel-driven (react to the
  `STAT_UPDATED` event whose amount is 10000 and emit the charged `APPLY`) keeps
  atlas-buffs generic and mirrors Combo, but adds a round trip and a window in
  which two commands race for the same buff. Buffs-owned (a threshold/promotion
  concept inside atlas-buffs) is atomic but bakes a skill-specific rule into a
  generic service. Design phase decides; §5 and §7.3 change if the latter wins.
- **OQ-3 — Exact charged-skill set for FR-6.** Energy Blast is near-certain:
  WZ shows `5111002` has **no `mpCon`** while its sibling Shockwave `5111006`
  has `mpCon: 18` — consistent with Energy Blast costing energy rather than MP.
  But WZ carries no explicit "requires charge" node, so whether Shockwave,
  Thunder Breaker Spark (`15111006`), or Shark Wave (`15111007`) are also
  client-gated is **unverified**. Resolve in design by reading the client's own
  gate in IDA (v83 IDB) before finalising the FR-6.1 id list. Until then FR-6
  covers Energy Blast only.
- **OQ-4 — Reset broadcast fidelity.** Cosmic sends the owner a `giveBuff` with
  value 0 and the map a foreign *cancel*. Whether Atlas's existing `EXPIRED`
  handling already produces both halves, or whether the zero-value owner packet
  needs an explicit emit, must be confirmed against
  `kafka/consumer/buff/consumer.go` in design.
- **OQ-5 — Task number.** `tools/task-numbers.sh next` returns **213**; nothing
  occupies 213–215 in `docs/tasks/`, `.worktrees/`, or local branches. 216 was
  assigned by explicit instruction. If 213–215 are not in fact reserved
  elsewhere, this number leaves a gap.

## 10. Acceptance Criteria

- **AC-1** A Marauder with Energy Charge level > 0 who hits N monsters with one
  melee attack has their `ENERGY_CHARGE` value increase by exactly `102 × N`,
  clamped at 10000.
- **AC-2** A character without the skill, or outside the Marauder / Thunder
  Breaker Stage 2+ job lines, produces no energy emit on any attack.
- **AC-3** An attack that hits zero monsters produces no energy emit.
- **AC-4** On reaching 10000 the character's `ENERGY_CHARGE` becomes 15000 and
  a timed buff is in place for the effect's duration at their level (31s at
  level 1, 40s at level 20, per WZ `5110001/level/<n>/time`).
- **AC-5** While at 15000, further hits do not increment the bar.
- **AC-6** When the charged buff expires the bar reads 0, the owner is told, and
  observers in the map receive a foreign cancel of the `ENERGY_CHARGE` stat.
- **AC-7** After expiry, the next qualifying hit sets the bar to 102.
- **AC-8** While charged, `GET /characters/{id}/effective-stats` reports weapon
  attack increased by the Energy Charge effect's `pad` at the character's skill
  level (0 at levels 1–3, 11 at levels 4–5, 15 at level 20). It is **not**
  increased by the bar value.
- **AC-9** The `ENERGY_CHARGE` temporary stat encodes correctly on every version
  in §7.4's supported set — verified against the coverage matrix, not asserted.
  Any version whose cell is not already verified is either promoted or
  explicitly recorded as out of scope with a reason.
- **AC-10** A Thunder Breaker Stage 2+ character exhibits AC-1 through AC-8 via
  skill `15100004`, and on gms_v61 (where that skill does not exist) the Cygnus
  branch is a no-op rather than an error.
- **AC-11** An Energy Blast cast at a non-full bar is rejected with no effect
  applied and no session teardown; at a full bar it proceeds, and the bar is
  **not** consumed.
- **AC-12** An Energy Charge touch attack grants energy (FR-7.2) but does not
  re-apply or refresh the Energy Charge effect (FR-7.1).
- **AC-13** No energy-bookkeeping failure path can reject or error an attack;
  covered by a unit test that forces each dep to fail.
- **AC-14** Verification per CLAUDE.md: `go test -race ./...`, `go vet ./...`,
  and `go build ./...` clean in atlas-channel and atlas-effective-stats;
  `tools/lint.sh --check`, `tools/buff-duration-guard.sh`,
  `tools/skill-job-id-guard.sh`, `tools/redis-key-guard.sh`, and
  `tools/goroutine-guard.sh` clean from the repo root. `docker buildx bake` is
  required only if any touched service's `go.mod` changes.

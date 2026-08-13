# Combo Drain (Aran) — Product Requirements Document

Version: v3
Status: Draft
Created: 2026-07-10
Revised: 2026-08-07 (v2) — rebased onto `main`; all code references re-derived
against current `main`; version scope expanded to every supported client
version (§4A, FR-5, §10).
Revised: 2026-08-13 (v3) — merged `main` into the task branch; all code
references re-derived again; §1.2 records the sibling Aran/attack-path features
that landed in the window (task-216 Energy Charge, task-217 Aran Combo Counter)
and what they do — and do not — change for this task.
---

## 1. Overview

Combo Drain (Aran 2nd-job skill, id `21100005`) is a self-buff that, while
active, restores the caster's HP by a percentage of the damage they deal with
attacks. The buff side of the skill already works end-to-end in Atlas: casting
the skill flows through the generic skill handler, atlas-data's skill reader
produces a `COMBO_DRAIN` temporary-stat statup carrying the skill effect's `x`
value (`services/atlas-data/atlas.com/data/skill/reader.go:407-408`), and the
buff is applied/rendered like any other. What is missing is the attack-side
effect: the damage handler in atlas-channel never checks for the active buff,
so the heal never happens. The gap is marked by `// TODO Combo Drain` at
`services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go:1117`.

Reference behavior is Cosmic `AbstractDealDamageHandler.java:421-431`: when
`BuffStat.COMBO_DRAIN` is present, the attacker is healed
`totalDamage * x / 100`. Note that Cosmic evaluates this inside its per-monster
loop against a *running* damage total, which over-heals multi-target attacks (a
Cosmic quirk, not retail semantics). Per owner decision, Atlas implements the
semantically correct version: one heal per accepted attack, computed from the
total damage across all monsters hit.

This is a small, single-service change: read the attacker's active buffs (an
API the attack pipeline already uses for the projectile gate and Pick Pocket),
and if a `COMBO_DRAIN` stat is present, emit an HP change for the computed
heal.

### 1.1 What changed since v1 of this PRD

The task branch sat behind `main` for 254 commits. Several sibling attack-side
features landed in the same handler in that window — drain-family heals
(`drainTryHeal`), Pick Pocket (`pickPocketResolveState`/`pickPocketTryProc`),
Homing Beacon (`beaconTryApply`), Mortal Blow, Sacrifice, and Aran combo-orb
bookkeeping (`comboOrbTryUpdate`). Three consequences for this PRD:

- Every line reference in v1 was stale and has been re-derived.
- `processAttack` now performs **zero to one** buff REST reads per attack
  (Pick Pocket fetches only for whitelisted skills; the projectile planner
  fetches only for ranged attacks that clear its earlier gates). v1's
  performance NFR assumed the projectile path always fetched. Combo Drain needs
  the buff list on *every* attack, so the honest cost statement changed — see
  NFR in §8.
- `buff.NewBuff` gained a seventh parameter (`noExpiry bool`).

## 1.2 What changed since v2 of this PRD (2026-08-13 `main` merge)

Two further Aran/attack-path features landed in `main` and are now merged into
this branch. Neither changes any requirement below, but both touch the same
function and both bear on the one-buff-read NFR, so they are recorded here
explicitly rather than left for the implementer to rediscover:

- **task-217 — Aran Combo Counter** (`character_aran_combo.go`, new). The
  combo *count* now lives in a channel-local mirror (`character/combo`), seeded
  by a `ARAN_COMBO` no-expiry buff and advanced by the client's
  `ARAN_COMBO_COUNTER` packet. `processAttack` calls
  `aranComboRefreshEligibility` on the melee path
  (`character_attack_common.go:1032`) to cache the gate result off the character
  fetch it already paid for. **It performs no buff REST read**, so the one-read
  ceiling in §8 is unaffected. This is a *different* Aran mechanic from Combo
  Drain — the counter is the combo-orb/Combo Ability chain; Combo Drain is a
  self-buff percentage heal — and the two share no state. It does confirm this
  PRD's version posture, though: task-217 resolved its one genuinely
  version-varying value (the 3000 ms vs 5000 ms client idle window) as *tenant
  configuration* (`idleResetMs`), not a compiled major-version branch.
- **task-216 — Energy Charge** (`character_attack_energy_charge.go`, new).
  `processAttack` calls `energyChargeTryUpdate` at
  `character_attack_common.go:1050`; its production deps emit a buff
  `UpdateStatValue` with `CreateIfMissing`, deliberately so the attack path
  needs no buff read. The one REST read in that file
  (`energyReannounceAuthoritative:198`) fires only on a *rejected* Energy Blast
  cast, which returns before reaching the TODO block. Ceiling unaffected.

So the buff-read inventory for `processAttack` is unchanged from v2: the
projectile consumption gate and Pick Pocket, both gate-before-fetch. Line
numbers moved (the TODO block is at `:1117`, was `:991`); all references in
this PRD and in `design.md` / `plan.md` were re-derived against the merge.

## 2. Goals

Primary goals:
- An Aran with Combo Drain active recovers HP equal to `x`% of the total damage
  of each attack they land, where `x` is the buff's `COMBO_DRAIN` statup amount.
- The heal is emitted once per attack (not per monster, not per hit line).
- The behavior is correct on **every supported client version that has the
  skill**, with no version-specific branch in the implementation (§4A, FR-5).
- No behavior change for characters without the buff, and no change to the
  attack pipeline's existing broadcast/damage/proc ordering.

Non-goals:
- Combo-orb consumption or any Aran combo-counter mechanics. Both already
  landed separately: combo orbs as `comboOrbTryUpdate`
  (`character_attack_combo.go:172`), and the combo *counter* as task-217
  (`character_aran_combo.go`, the `ARAN_COMBO_COUNTER` handler plus the
  `character/combo` mirror). Combo Drain shares no state with either — its
  gate is the `COMBO_DRAIN` temporary stat alone (FR-1).
- Any of the sibling TODOs in the same block (Flame Thrower, Snow Charge,
  Hamstring, Slow, Blind, charges, Three Snails, Heavens Hammer, ComboTempest,
  BodyPressure).
- Packet/writer changes — the HP change flows through the existing character
  stat-update path.
- Replicating Cosmic's per-monster running-total quirk.
- Completing the `gms_92_1` or `gms_12_1` socket-config template bring-ups
  (§4A).

## 3. User Stories

- As an Aran player with Combo Drain active, I want my HP to visibly recover as
  I attack monsters so that the skill functions as described in-game.
- As an Aran player on any supported client version that ships the skill
  (GMS v0.79 through v0.95, and JMS v185), I want the same behavior, so the
  server does not silently differ by version.
- As a player without Combo Drain, I want attack handling to behave exactly as
  before so that unrelated combat is unaffected.
- As a server operator, I want the heal to be derived from tenant skill data
  (the buff statup), not hard-coded percentages, so that per-version WZ values
  and WZ-driven customization keep working.

## 4. Functional Requirements

FR-1 — Buff detection
- After an attack is accepted and damage has been applied (at the
  `character_attack_common.go:991` TODO site), evaluate the attacker's active
  buffs, obtained from the per-attack memoized buff loader (§8 NFR, design §3).
- The effect triggers when any active, non-expired buff carries a stat change
  with `Type() == string(character.TemporaryStatTypeComboDrain)`
  (atlas-constants `TemporaryStatTypeComboDrain = "COMBO_DRAIN"`).
- If the buff-fetch fails, log a warning and skip the heal; the rest of the
  attack pipeline must be unaffected (same failure posture as the existing
  projectile gate and Pick Pocket).

FR-2 — Heal computation
- `totalDamage` = sum of all values in `di.Damages()` across every entry of
  `ai.DamageInfo()` for the attack.
- `healPercent` = the `Amount()` of the matching `COMBO_DRAIN` stat change.
  This is the skill effect's `x`, already populated by atlas-data from the
  **tenant's own WZ data**, so per-version differences in the level→`x` curve
  are honored automatically. No `x` value is hard-coded anywhere in this
  feature.
- `heal = totalDamage * healPercent / 100`, integer arithmetic.
- If `totalDamage == 0` or the computed heal is `<= 0`, do nothing (no
  zero-amount HP command).
- Clamp the computed heal to `math.MaxInt16` before conversion — `ChangeHP`
  takes `int16` and a large attack total must not overflow/wrap.

FR-3 — Heal application
- Emit the heal via the existing
  `character.Processor.ChangeHP(f field.Model, characterId uint32, amount int16)`,
  which produces the Kafka `ChangeHP` command; downstream clamping to max HP is
  owned by atlas-character.
- The heal applies for any attack type (melee/ranged/magic/energy) — the gate
  is the buff alone, with no job, skill-ownership, version, or attack-type
  check.
- Ordering: evaluate after per-monster damage processing (so `DamageInfo` is
  final) and independent of broadcast success, consistent with the other
  post-damage effects in the handler.

FR-4 — No regression
- Characters without the `COMBO_DRAIN` stat active take the existing code path
  with no additional Kafka emissions.
- Existing buff consumers in `processAttack` (projectile gate, Pick Pocket)
  keep their exact current semantics, including their failure postures.

FR-5 — Version behavior (new in v2)
- The implementation contains **no version branch, no region check, and no
  `MajorAtLeast` gate**. Availability is a property of the data, not the code:
  a version whose client has no Aran cannot produce a `COMBO_DRAIN` statup, so
  the gate simply never fires there. This matches the precedent set by the
  Aran combo-orb handler, which gates on the character's learned skills rather
  than on version (`character_attack_combo.go:37` `comboSkillIds`, called at
  `:167`).
- No skill-id comparison is introduced, so `tools/skill-job-id-guard.sh` is not
  engaged. (Independently confirmed: wire id `21100005` has no row in
  `docs/tasks/task-187-version-aware-id-semantics/audit/divergences.csv`, so it
  is not a version-divergent id.)
- The `COMBO_DRAIN` temporary-stat bit is allocated unconditionally for every
  supported version — it sits in the pre-SoulStone block of the registry
  (`libs/atlas-packet/model/character_temporary_stat.go:155`), ahead of the
  first version-gated slot at bit 82 — so buff encode/decode needs no change on
  any version.

## 4A. Supported Versions & Applicability

Atlas provisions **eleven** tenant socket-config templates
(`services/atlas-configurations/seed-data/templates/`). The table below states,
per version, whether Combo Drain is in scope, with the evidence for each cell.

| Tenant template | Aran `21100005` present | Attack handlers routed in template | Combo Drain scope |
|---|---|---|---|
| `gms_12_1`  | no  | none                                  | **N/A** — no Aran, no attack routing |
| `gms_48_1`  | no  | melee, ranged, magic (no touch)       | **N/A** — no Aran |
| `gms_61_1`  | no  | melee, ranged, magic, touch           | **N/A** — no Aran |
| `gms_72_1`  | no  | melee, ranged, magic, touch           | **N/A** — no Aran |
| `gms_79_1`  | yes | melee, ranged, magic, touch           | **IN SCOPE** |
| `gms_83_1`  | yes | melee, ranged, magic, touch           | **IN SCOPE** |
| `gms_84_1`  | yes | melee, ranged, magic, touch           | **IN SCOPE** |
| `gms_87_1`  | yes | melee, ranged, magic, touch           | **IN SCOPE** |
| `gms_92_1`  | yes | **none**                              | **BLOCKED** — see §4A.3 |
| `gms_95_1`  | yes | melee, ranged, magic, touch           | **IN SCOPE** |
| `jms_185_1` | yes | melee, ranged, magic, touch           | **IN SCOPE** |

### 4A.1 Evidence

- **Skill availability** — `libs/atlas-constants/gen/wzsnapshot/<key>.json`
  (`skills` list), drained from live atlas-data per tenant; provenance in
  `libs/atlas-constants/gen/wzsnapshot/PROVENANCE.md`. `21100005` appears in
  `gms_79_1`, `gms_83_1`, `gms_84_1`, `gms_87_1`, `gms_92_1`, `gms_95_1`,
  `jms_185_1` and is absent from `gms_12_1`, `gms_48_1`, `gms_61_1`,
  `gms_72_1`. The generated per-version tables agree
  (`libs/atlas-constants/skill/version_<key>_gen.go`), as do the Aran job ids
  (`libs/atlas-constants/job/version_<key>_gen.go`: jobs 2100/2110/2111/2112
  present in exactly the same seven versions).
  - This corrects v1 of this PRD, which asserted the skill arrived "in GMS
    v84". Per the snapshot it is present from `gms_79_1` onward.
- **Attack routing** — handler names `CharacterMeleeAttackHandle`,
  `CharacterRangedAttackHandle`, `CharacterMagicAttackHandle`,
  `CharacterTouchAttackHandle` in
  `services/atlas-configurations/seed-data/templates/template_<key>.json`.
  There is no `CharacterEnergyAttackHandle`: `AttackTypeEnergy` is produced by
  the **touch** handler (`character_attack_touch.go:18`), so "energy attacks"
  in FR-3 means the touch path.
- **Wire coverage** — `docs/packets/audits/status.json`: `CLOSE_RANGE_ATTACK`,
  `RANGED_ATTACK` and `MAGIC_ATTACK` (serverbound) are `verified` on all nine
  matrix columns; `TOUCH_MONSTER_ATTACK` is `n-a` on `gms_v48` and `verified`
  on the other eight; the downstream `STAT_CHANGED` and `GIVE_BUFF` clientbound
  packets are `verified` on all nine. So on every in-scope version the damage
  totals this feature sums, and the stat update it produces, rest on verified
  codecs — no new packet work is required.

### 4A.2 Version columns vs. tenant templates

The packet coverage matrix tracks **nine** versions (`gms_v48`, `gms_v61`,
`gms_v72`, `gms_v79`, `gms_v83`, `gms_v84`, `gms_v87`, `gms_v95`, `jms_v185` —
`docs/packets/PROCESS.md`, `packet-process-facts` block). `gms_92_1` and
`gms_12_1` are provisioned templates without matrix columns. This task requires
no matrix change and declares no coverage manifest, because it adds no codec
and no version gate.

### 4A.3 `gms_92_1` — documented blocker, not a deferral

`template_gms_92_1.json` routes ~49 handlers versus ~131 for `gms_95_1`, and
routes **no** attack handler at all. Aran and `21100005` exist on that client,
so Combo Drain is unreachable there purely because the attack packets are not
routed.

This is not producible inside this task: routing the four attack handlers
requires the `gms_v92` serverbound opcodes for `CLOSE_RANGE_ATTACK`,
`RANGED_ATTACK`, `MAGIC_ATTACK` and `TOUCH_MONSTER_ATTACK`, and `gms_v92` is
not a packet-matrix column — there is no verified opcode source to route
against, and inventing one would violate the project's grounding rule. The same
is true of `gms_12_1` (~24 handlers). Completing those two template bring-ups
is a version-bring-up effort (`/bringup-version`), not a skill feature.

Consequence for this task: Combo Drain is implemented once, version-blind, and
becomes live on `gms_92_1` and `gms_12_1` the moment their attack handlers are
routed — no further Combo Drain work will be needed then.

## 5. API Surface

No new or modified REST endpoints. No new Kafka topics or message shapes — the
feature reuses the existing character `ChangeHP` command and the existing
atlas-buffs REST read (`buff/requests.go`). No packet, opcode, or template
changes on any version.

## 6. Data Model

No schema or entity changes. All required data already exists:
- `libs/atlas-constants/skill/constants.go:3400` —
  `AranStage2ComboDrainId = Id(21100005)` (not needed at runtime; the gate is
  the stat, not the skill).
- `libs/atlas-constants/character/temporary_stat.go:75` —
  `TemporaryStatTypeComboDrain`.
- atlas-data skill reader already emits the `COMBO_DRAIN` statup with
  amount = effect `x` (`skill/reader.go:407-408`), version-blind and sourced
  from the tenant's WZ.

## 7. Service Impact

- **atlas-channel** (only service touched): implement the Combo Drain block in
  `socket/handler/character_attack_common.go`, replacing the line-991 TODO, plus
  a memoized per-attack buff loader shared with the existing buff consumers.
  Add unit tests.
- atlas-character, atlas-buffs, atlas-data, atlas-configurations: consumed
  as-is, no changes. No template edits on any of the eleven versions.

## 8. Non-Functional Requirements

- **Multi-tenancy:** all reads/emits use the request `ctx` (tenant-scoped) as
  the surrounding handler already does; no tenant-specific and no
  version-specific literals.
- **Performance:** at most **one** buff REST read per attack, total, across
  every consumer in `processAttack`. The two consumers are the projectile
  consumption gate and Pick Pocket; the post-damage effects added since v2
  (Aran combo eligibility, Energy Charge) read no buffs at all (§1.2), so the
  inventory is complete. Today melee and magic attacks that are not
  Pick-Pocket-whitelisted perform zero buff reads, so an unshared
  implementation would add a read to the hot path for every such attack; a
  per-attack memoized loader (mirroring the existing `loadEffectiveStats`
  closure) keeps the ceiling at one and lets the ranged/Pick-Pocket paths reuse
  the same snapshot rather than issuing their own. The buff-only gate (FR-3) is
  retained, so a cheap short-circuit that skips the read for non-Aran
  characters is deliberately not taken.
- **Resilience:** buff-service failure degrades to "no heal" with a logged
  warning; it must never fail the attack handler, and must not change the
  existing consumers' degraded behavior.
- **Observability:** debug-level log when a Combo Drain heal is emitted
  (characterId, totalDamage, percent, heal).
- **Version neutrality:** the diff must contain no `MajorVersion`,
  `MajorAtLeast`, `IsRegion`, or version-keyed literal. A reviewer finding one
  should treat it as a defect (FR-5).

## 9. Open Questions

None. Scope decisions were resolved in the spec interview (2026-07-10):
semantically-correct single heal per attack; percent sourced from the buff
statup amount; buff-only gate (all attack types); no cap beyond the natural
max-HP clamp plus the `int16` conversion clamp. The 2026-08-07 revision
resolved two more from evidence rather than opinion: version scope is all
eleven templates with per-version applicability derived from the WZ snapshot
(§4A), and the `gms_92_1`/`gms_12_1` gaps are documented blockers with a named
cause (§4A.3).

## 10. Acceptance Criteria

Behavior:
- [ ] With a `COMBO_DRAIN` buff of amount `x` active, an accepted attack
      dealing total damage `D` (summed over all monsters and hit lines) emits
      exactly one `ChangeHP` command for `min(D*x/100, math.MaxInt16)` when
      that value is `> 0`.
- [ ] Multi-target attacks heal on the plain total (no Cosmic running-total
      over-heal).
- [ ] No `ChangeHP` emission when the buff is absent, expired, when
      `D*x/100 == 0`, or when the buff lookup errors (warning logged instead).
- [ ] Works for melee, ranged, magic, and energy (touch) attack types.

Version coverage:
- [ ] The diff contains no version or region branch: `git diff main` shows no
      `MajorVersion`, `MajorAtLeast`, or `IsRegion` in the added lines.
- [ ] No socket-config template is modified on any of the eleven versions
      (`git diff --stat main -- services/atlas-configurations/seed-data/templates/`
      is empty).
- [ ] The per-version applicability table (§4A) is reproduced in `design.md`
      and each of its cells is backed by a cited artifact, not by recall.
- [ ] `docs/packets/audits/status.json` is unchanged — this task promotes no
      matrix cell and needs none promoted.

Regression & cost:
- [ ] The projectile gate and Pick Pocket retain their exact current semantics,
      including buff-fetch-failure posture; their existing tests pass unmodified
      in substance.
- [ ] `processAttack` issues at most one buff REST read per attack on every
      path (verified by a test that counts loader invocations).

Tests & gates:
- [ ] Unit tests cover: buff present (single + multi monster), buff absent,
      expired buff, zero damage, `int16` clamp boundary, buff-fetch error, and
      one case per `AttackType`. Tests use the project Builder pattern /
      production constructors (no `*_testhelpers.go`) and the current
      seven-argument `buff.NewBuff`.
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...` clean in
      atlas-channel; `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`,
      `tools/skill-job-id-guard.sh`, `tools/buff-duration-guard.sh` and
      `tools/lint.sh --check` clean from the repo root;
      `docker buildx bake atlas-channel` succeeds.
- [ ] The `// TODO Combo Drain` marker is removed; `docs/TODO.md:151` line item
      checked off.

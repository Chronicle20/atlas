# Chakra (Chief Bandit 4211001) — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-12

---

## 1. Overview

Chakra (skill id `4211001`) is the Chief Bandit's self-recovery skill. In an authentic
pre-Big-Bang client the player activates it while badly wounded; the character enters a
stationary recovery animation, takes amplified damage for its duration, and — if the
animation completes uninterrupted — restores HP. It is a deliberately risky heal: the
lower the skill level, the larger the incoming-damage penalty.

In Atlas today the skill does nothing server-side. `grep -rE "Chakra|CHAKRA"` across
`services/` returns exactly one functional hit — `services/atlas-data/atlas.com/data/skill/reader.go:600`,
where `ChiefBanditChakraId` appears in the `isCategory1` decode list. There is no cast
handler in `services/atlas-channel/atlas.com/channel/skill/handler/`, no entry in
`skill/handler/registrations/registrations.go:6-16`, no `CHAKRA` temporary stat in
`libs/atlas-packet/model/character_temporary_stat.go`, and no branch in the damage-taken
mitigation roster (`services/atlas-channel/atlas.com/channel/socket/handler/character_damage_mitigation.go`).
The observable result is the one reported: **the animation plays, MP is spent by the
generic cast path, and no HP is restored.**

This task implements Chakra end to end across all eleven provisioned client versions:
the activation gate, the recovery state, the incoming-damage amplifier, the interruption
rules, and the HP restore itself. Two of its inputs are deliberately *not* settled by
this PRD and are assigned to the design phase as mandatory IDA derivations — the heal
formula and the keydown/prepare-broadcast question (§9, §10.1). Both have a documented
history of being answered from community recollection and getting it wrong, so this PRD
specifies the shape of the answer and forbids guessing the values.

## 2. Goals

Primary goals:

- Restore HP on a successful Chakra cast, using a formula derived from the client binary
  rather than from Cosmic source or community tables.
- Enforce the `currentHP < maxHP × 0.50` activation requirement.
- Model the recovery window as an explicit server-side state with a defined lifetime,
  during which incoming damage is amplified per the skill's level.
- Interrupt the pending recovery on movement, on damage, and on death, cancelling the
  heal without refunding MP.
- Work on every provisioned version (gms 12/48/61/72/79/83/84/87/92/95, jms 185) —
  `4211001` is present in all eleven version tables in `libs/atlas-constants/skill/`.
- Settle, with IDA evidence, whether Chakra is a keydown/prepare-broadcast skill, and
  reconcile the two conflicting records in the repo (see §10.1).

Non-goals:

- Awarding experience for the heal. Chakra is self-only; unlike Cleric Heal there is no
  heal-XP accumulator (FR-8.4).
- Charging MP or applying cooldown inside the Chakra handler. The generic cast path in
  `skill/handler/common.go` `UseSkill` already does both once a handler is registered;
  duplicating it is the task-200 double-charge trap (`skill/handler/registry.go:56-76`).
- Implementing FP Mage Explosion (2111002), the other skill named alongside Chakra in
  `docs/research/missing-features/skills-and-buffs.md` P7.
- Randomised recovery percentage (§9.4) and double MP consumption (§11 of the source
  spec) — both are recorded as UNVERIFIED and are explicitly excluded unless the design
  phase's IDA pass produces evidence for them.
- Meso Guard (4211005), for which Chakra Lv3 is a prerequisite in the skill tree. Out of
  scope; noted only because it explains why Chakra is commonly levelled to 3 and no further.
- Any packet-coverage-matrix work, new opcodes, or codec changes, **unless** §10.1's IDA
  pass shows a prepare/keydown broadcast is required, in which case only the existing
  prepare writer is reused.

## 3. User Stories

- As a Chief Bandit at low HP, I want to cast Chakra and recover HP, so that the skill I
  spent points on has an effect.
- As a Chief Bandit at full HP, I want Chakra to refuse to activate, so that the skill's
  intended emergency-only role is preserved and I do not waste MP.
- As a Chief Bandit mid-recovery, I want a mob hit to interrupt the heal and hurt more
  than usual, so that the skill carries its authentic risk.
- As a Chief Bandit who moves during recovery, I want the heal cancelled, so that the
  skill cannot be used as a free heal while kiting.
- As another player in the map, I want to see the Chakra cast effect, so that party
  members can read what a wounded Chief Bandit is doing.
- As a maintainer, I want the heal formula and the keydown decision recorded with IDA
  evidence in-repo, so that the next person does not re-derive them from a wiki.

## 4. Functional Requirements

### FR-1 — Activation gate

- **FR-1.1** Chakra MUST NOT begin recovery when `currentHP >= maxHP × 0.50`. The
  comparison is strict-less-than on the left: `canActivate = currentHP < (maxHP × 0.50)`.
- **FR-1.2** `maxHP` for the gate MUST be the **effective** max HP from
  `atlas-effective-stats` (gear + buffs), falling back to the character record's base
  MaxHp when the upstream returns zero or an out-of-range value — the same defensive
  narrowing `skill/handler/heal/heal.go` `effectiveMaxHpOrBase` already performs.
- **FR-1.3** The threshold is checked **at activation only**. External healing that
  raises the caster to ≥50% mid-recovery MUST NOT cancel the pending heal. (Source spec
  §12; recorded as the chosen behaviour, to be covered by a test.)
- **FR-1.4** A rejected activation MUST NOT consume MP and MUST NOT apply cooldown. Since
  the generic path in `UseSkill` performs the cost/cooldown steps before invoking the
  registered handler, the gate MUST be evaluated in a position where a rejection still
  suppresses the spend, or the spend MUST be reversed. The design phase MUST state which
  of the two it uses and why, with the ordering shown against `common.go`.
- **FR-1.5** The caster MUST know the skill at level ≥ 1. This is already enforced
  generically; the requirement is listed so the design phase confirms it rather than
  re-implementing it.

### FR-2 — Recovery state

- **FR-2.1** A successful activation MUST place the caster into a server-tracked
  `CHAKRA_RECOVERING` state, scoped per (tenant, character).
- **FR-2.2** The state MUST have a bounded lifetime and MUST self-clear when it expires,
  even if no completion or interruption event arrives. No state may leak on disconnect,
  channel change, or map change.
- **FR-2.3** The nominal recovery duration is **~1500 ms** (source spec §5), but this
  value MUST be confirmed against the client's Chakra action/animation timing in the
  design phase. If IDA/WZ shows a different figure — including a level-dependent or
  version-dependent one — the derived value wins and the PRD is amended.
- **FR-2.4** The state MUST be readable from the damage-taken path (FR-4) and from the
  movement path (FR-5) without a cross-service round trip on every hit.
- **FR-2.5** There is no `CHAKRA` entry in the character-temporary-stat registry
  (`libs/atlas-packet/model/character_temporary_stat.go` — verified absent). The design
  phase MUST determine from IDA whether the client tracks a Chakra recovery stat/flag at
  all. If it does not, the state is purely server-side and MUST NOT be invented as a
  temporary stat, because a fabricated CTS entry would be broadcast to clients that never
  read it.

### FR-3 — HP restore

- **FR-3.1** On uninterrupted completion, the caster's HP MUST increase by the computed
  heal amount.
- **FR-3.2** The heal MUST be clamped so that `newHP = min(effectiveMaxHP, currentHP + healAmount)`.
  Chakra MUST never raise HP above maximum, and MUST never push a value that trips
  `atlas-character`'s `enforceBounds` saturation — the same clamp discipline as
  `heal/formula.go` `appliedPerRecipient`.
- **FR-3.3** The heal MUST be applied via the existing `character.ChangeHP` path so that
  HP-change broadcast and persistence behave identically to every other heal source.
- **FR-3.4** The heal MUST occur at the completion point of the recovery window, not at
  keypress (source spec §5).
- **FR-3.5** A negative or zero computed heal MUST clamp to zero and MUST NOT be applied
  as a damage event.

### FR-4 — Incoming damage during recovery

- **FR-4.1** While `CHAKRA_RECOVERING` is active, incoming damage MUST be multiplied by
  the skill level's damage-taken factor before being applied to HP.
- **FR-4.2** The amplifier MUST be implemented as an entry in the existing server-side
  mitigation roster — `computeMitigation` / `mitigationInput` in
  `services/atlas-channel/atlas.com/channel/socket/handler/character_damage_mitigation.go:21-132`
  — and MUST NOT be applied by trusting a client-supplied amplified value. task-157's IDA
  pass established the governing fact that **the client sends the raw, pre-mitigation
  damage value on every supported version** (`docs/tasks/task-157-damage-taken-mitigation/design.md` §1),
  so the server is already the authority for every multiplier on this path.
- **FR-4.3** Ordering relative to task-157's existing mitigations (Magic Guard, Meso Guard,
  Power Guard, Achilles, Combo Barrier, Magic Shield, the `-1` block sentinel) MUST be
  derived from the client's `CUserLocal::SetDamaged` / `CalcDamage` sequence, not chosen
  for convenience. An amplifier applied before vs. after Achilles produces materially
  different numbers.
- **FR-4.4** Rounding MUST follow the same integer-arithmetic convention `computeMitigation`
  already uses ("integer arithmetic follows the decompiled formulas exactly",
  `character_damage_mitigation.go:129-132`).
- **FR-4.5** The amplifier MUST apply to the interrupting hit itself (§8 of the source
  spec): the player takes the increased damage even though that same hit cancels the heal.

### FR-5 — Interruption

- **FR-5.1** Movement during recovery MUST interrupt Chakra: walking, jumping, climbing,
  and any server-represented forced displacement or externally-induced position change.
- **FR-5.2** Taking damage during recovery MUST interrupt Chakra, in this order: amplify
  (FR-4) → apply damage → cancel pending heal → exit state.
- **FR-5.3** Death during recovery ends the state; normal death processing takes
  precedence and the pending heal MUST NOT fire afterwards. A pending recovery MUST NOT
  resurrect a dead character.
- **FR-5.4** An interrupted cast MUST NOT refund MP.
- **FR-5.5** Map change, channel change, and disconnect MUST clear the state (a specific
  case of FR-2.2).

### FR-6 — Level parameters

- **FR-6.1** MP cost, recovery rate, and damage-taken percentage MUST be read from the
  WZ-derived skill effect served by `atlas-data` for the caster's tenant version — the
  same `effect.Model` every other handler receives. Level tables MUST NOT be hardcoded
  in `atlas-channel`.
- **FR-6.2** The table below is the **provided reference table** for the pre-Big-Bang
  skill. It is recorded here so the design phase can cross-check what `atlas-data`
  actually serves. Where the served WZ values and this table disagree, **the WZ data
  wins** and the discrepancy is documented; per CLAUDE.md ("Verification Over Memory")
  a community table is a cross-check, never a source.

  | Lv | MP | Recovery | Damage Taken | | Lv | MP | Recovery | Damage Taken |
  |---|---|---|---|---|---|---|---|---|
  | 1 | 15 | 9% | 200% | | 16 | 21 | 129% | 145% |
  | 2 | 15 | 18% | 196% | | 17 | 21 | 135% | 142% |
  | 3 | 15 | 27% | 192% | | 18 | 21 | 141% | 139% |
  | 4 | 15 | 36% | 188% | | 19 | 21 | 147% | 136% |
  | 5 | 15 | 45% | 184% | | 20 | 21 | 153% | 133% |
  | 6 | 15 | 54% | 180% | | 21 | 27 | 159% | 130% |
  | 7 | 15 | 62% | 176% | | 22 | 27 | 164% | 128% |
  | 8 | 15 | 70% | 172% | | 23 | 27 | 169% | 126% |
  | 9 | 15 | 78% | 168% | | 24 | 27 | 174% | 124% |
  | 10 | 15 | 86% | 164% | | 25 | 27 | 179% | 122% |
  | 11 | 21 | 94% | 160% | | 26 | 27 | 184% | 120% |
  | 12 | 21 | 101% | 157% | | 27 | 27 | 188% | 118% |
  | 13 | 21 | 108% | 154% | | 28 | 27 | 192% | 116% |
  | 14 | 21 | 115% | 151% | | 29 | 27 | 196% | 114% |
  | 15 | 21 | 122% | 148% | | 30 | 27 | 200% | 112% |

  Master level 30. Several values round non-uniformly, so the table must not be
  regenerated from an interpolating formula.

- **FR-6.3** The design phase MUST identify which WZ effect fields (`x`, `y`, `hp`, `mp`,
  `damage`, …) carry recovery rate and damage-taken, per version, and MUST confirm the
  `effect.Model` accessors already expose them (`X()` and `Y()` exist at
  `services/atlas-channel/atlas.com/channel/data/skill/effect/model.go:177,190`). If a
  needed field is not parsed by `atlas-data`'s skill reader, adding that parse is in scope.
- **FR-6.4** Local `Skill.wz` copies are available on this machine for GMS 48, 61, 62, 72,
  75, 84, 87, 117 and 118, plus the Cosmic reference set — enough to check node `4211001`
  directly for most of the version range without a live cluster. Versions with no local
  copy (notably 79, 83, 92, 95, jms 185) MUST be checked against what `atlas-data` serves
  in the live baseline rather than assumed to match a neighbouring version. Concrete paths
  are environment-specific and deliberately not recorded here.

### FR-7 — Heal formula (design-phase derivation, mandatory)

- **FR-7.1** The heal amount MUST be derived from the client binary via `ida-pro-mcp`.
  This is an explicit instruction, not a preference.
- **FR-7.2** The recovery percentage MUST NOT be interpreted as a percentage of maximum
  HP. `healAmount = maxHP × recoveryRate` is specifically ruled out.
- **FR-7.3** A LUK-based formula MUST NOT be implemented on the strength of community
  recollection or of Cosmic's `StatEffect.calcHPChange` alone. Cosmic may be consulted as
  a cross-check only; if the derived client formula and Cosmic disagree, the client wins
  and the divergence is recorded.
- **FR-7.4** The implementation MUST keep the base-recovery term and the level's recovery
  rate as separate, separately-testable inputs:
  `healAmount = ChakraRecoveryFunction(baseRecovery, recoveryRate)`.
- **FR-7.5** If the design phase's IDA pass cannot establish the base-recovery term with
  confidence on a given version, that MUST be reported as an explicit blocker with the
  addresses and pseudocode examined — not filled in with a plausible guess (CLAUDE.md,
  "Grounding & Honesty").
- **FR-7.6** Randomised recovery (source spec §10) MUST NOT be implemented unless the IDA
  pass corroborates it. Absent evidence, the heal is deterministic for a given
  (level, caster state).

### FR-8 — Cast effects and cost

- **FR-8.1** A successful activation MUST broadcast the cast effect to the caster
  (`CharacterEffect`) and to same-map sessions (`CharacterEffectForeign`), matching the
  pattern in `skill/handler/heal/heal.go` step 8.
- **FR-8.2** MP cost and cooldown MUST come from the generic `UseSkill` block; the Chakra
  handler MUST NOT charge either. Registering a `Handler` is itself the "this handler owns
  the cost" signal described at `skill/handler/registry.go:56-76`.
- **FR-8.3** MP MUST be consumed exactly once per activation. Double consumption
  (activation + recovery completion) is UNVERIFIED and MUST NOT be implemented.
- **FR-8.4** No experience is awarded.

### FR-9 — Version coverage

- **FR-9.1** The handler MUST be registered on the version-blind identity
  `skill2.ChiefBanditChakra`, per the task-187 convention documented at
  `skill/handler/registry.go:26-32`. Raw wire-id comparison is prohibited.
- **FR-9.2** Behaviour MUST be correct on all eleven provisioned versions: gms 12, 48, 61,
  72, 79, 83, 84, 87, 92, 95 and jms 185. `4211001` is present in every one of the
  `libs/atlas-constants/skill/version_*_gen.go` tables (verified).
- **FR-9.3** Where a parameter, formula term, or the recovery duration differs by version,
  the difference MUST be expressed with the `MajorAtLeast` gate idiom, never a raw
  `> N` comparison (the `MajorVersion()>83` off-by-one has already bitten v87 once).
- **FR-9.4** Versions where the design phase's IDA pass cannot reach a conclusion MUST be
  listed explicitly in the design doc with what is missing, rather than silently assumed
  to match v83.

### FR-10 — Repo-record reconciliation

- **FR-10.1** The design phase MUST re-verify in IDA whether Chakra (`4211001`) is a
  keydown/prepare-broadcast skill. Two in-repo records currently conflict:
  - `libs/atlas-constants/skill/model_test.go:33-37` pins Chakra as **not** keydown,
    citing a task-161 IDA verification across v61/v72/v79/v83/v87/v95/jms185, and warns
    that re-adding it "broadcasts a phantom aura and makes `attack_info.go` over-read a
    tKeyDown field the client never sends."
  - `docs/research/missing-features/skills-and-buffs.md` P7 (line 103) states that
    Cosmic's `SkillEffectHandler.java:58-77` broadcasts a prepare packet for Chakra and
    that Atlas is missing it.
  The IDA result is authoritative. Exactly one of the two records is wrong and MUST be
  corrected in this task.
- **FR-10.2** If Chakra **is** keydown, `skill.IsKeyDownSkill`
  (`libs/atlas-constants/skill/model.go:58-75`) gains the id, `model_test.go` moves it
  from the `notKeydown` list to the `keydown` list with the new evidence cited inline, and
  the design doc MUST address the `attack_info.go` over-read the existing comment warns
  about.
- **FR-10.3** If Chakra is **not** keydown, `IsKeyDownSkill` and its pinning test are left
  untouched, and P7 in the research doc is corrected so the stale Cosmic-derived claim
  stops re-seeding tasks.
- **FR-10.4** Either way the outcome MUST be recorded with the IDBs, addresses, and
  versions examined, so this question is not re-opened a third time.

## 5. API Surface

No new REST endpoints and no changes to existing request/response shapes are anticipated.

Existing surfaces consumed:

- `atlas-character` — `ChangeHP` (HP restore and damage application).
- `atlas-effective-stats` — caster INT/LUK/DEX/MaxHP, for both the FR-1 gate and whatever
  stat terms FR-7's derived formula requires.
- `atlas-data` — the WZ-derived skill effect for `4211001` at the caster's tenant version.

If FR-6.3 finds that a required WZ field is unparsed, the change is confined to
`atlas-data`'s skill reader and the `effect` model; the served JSON:API resource gains
fields but no endpoint is added or renamed.

## 6. Data Model

No new persisted entities, no migrations, no schema changes.

The `CHAKRA_RECOVERING` state (FR-2) is transient per-character in-memory state in
`atlas-channel`, keyed by (tenant, character) with a bounded TTL. It follows the existing
singleton-registry convention (`sync.Once` + `sync.RWMutex`) used elsewhere in the
service. If the design phase instead routes it through Redis, the access MUST go through
`libs/atlas-redis` — `tools/redis-key-guard.sh` bans keyed commands on the raw `go-redis`
client outside that library.

Any timer or delayed-completion goroutine MUST be spawned via `routine.Go`
(`tools/goroutine-guard.sh` bans bare `go` statements outside `libs/atlas-routine`).

## 7. Service Impact

| Service / lib | Change |
|---|---|
| `atlas-channel` | New `skill/handler/chakra` package (handler + formula, mirroring the `heal` package layout); blank import added to `skill/handler/registrations/registrations.go`; recovery-state registry; amplifier branch wired into `socket/handler/character_damage_mitigation.go`; interruption hooks on the movement and damage paths. |
| `libs/atlas-constants` | Only if FR-10.2 applies (`IsKeyDownSkill` + its pinning test). Otherwise untouched. |
| `atlas-data` | Only if FR-6.3 finds a required WZ effect field is unparsed by the skill reader. |
| `libs/atlas-packet` | Only if FR-2.5's IDA pass shows a client-side Chakra stat/flag exists that must be encoded. Default expectation: no change. |
| `atlas-character` | No change — consumed via existing `ChangeHP`. |
| `atlas-effective-stats` | No change — consumed via existing stat query. |
| Seed templates / opcodes | No change expected. Any template edit would additionally require the opcode-order, duplicate-binding, and movement-types guards to pass. |

## 8. Non-Functional Requirements

- **Multi-tenancy** — every lookup resolves the tenant from context
  (`tenant.MustFromContext`); the recovery-state registry is tenant-scoped. Version
  resolution goes through `constants.For(region, major, minor)`.
- **Server authority** — no client-supplied value is trusted for the heal amount, the
  damage multiplier, the activation gate, or interruption. Client flags may be used as
  cross-checks only, consistent with task-157's posture.
- **Hot-path cost** — the damage-taken path runs on every hit taken by every character in
  the channel. The Chakra state check MUST be an in-process lookup (FR-2.4); a
  cross-service call per hit is unacceptable.
- **Concurrency** — the state registry is accessed from the cast path, the damage path,
  the movement path, and an expiry timer concurrently, and must be race-free under
  `go test -race`. Note the known trap that `atlas-channel`'s `ForEachInMap` runs in
  parallel over shared state.
- **Observability** — activation, rejection-by-gate, completion, and each interruption
  reason are logged at a level that makes "Chakra did nothing" diagnosable from logs
  alone.
- **No silent stubs** — no `// TODO`, stubbed branch, or 501 in landed commits. A term
  that cannot be derived is an explicit escalation, not a placeholder.

## 9. Open Questions

Resolved by this PRD (recorded so the design phase does not re-litigate):

- Version scope → all eleven provisioned versions (FR-9.2).
- XP → none (FR-8.4).
- MP/cooldown ownership → generic path, handler charges nothing (FR-8.2).
- Cast effect → self + foreign broadcast (FR-8.1).
- HP threshold re-check → activation only (FR-1.3).

Open, and assigned to the design phase:

- **OQ-1 (blocking, FR-7)** — the exact base-recovery term of the heal formula, per
  version, from IDA. The single largest unknown; the feature cannot be correct without it.
- **OQ-2 (blocking, FR-10.1)** — keydown/prepare-broadcast: does the client wind Chakra up
  as a keydown skill? Two in-repo records disagree.
- **OQ-3 (FR-2.3)** — the true recovery-window duration, and whether it varies by level or
  version. 1500 ms is a working figure only.
- **OQ-4 (FR-4.3)** — where the Chakra amplifier sits in the client's mitigation ordering
  relative to task-157's existing roster.
- **OQ-5 (FR-2.5)** — does the client track a Chakra recovery stat/flag, or is the state
  purely server-side?
- **OQ-6 (FR-6.3)** — which WZ effect fields carry recovery rate and damage-taken, and are
  they already parsed?
- **OQ-7 (FR-1.4)** — where the activation gate sits relative to `UseSkill`'s cost block
  so a rejected cast spends nothing.
- **OQ-8 (FR-5.1)** — which concrete movement events on Atlas's movement path constitute
  "movement" for interruption, given that the client reports movement fragments rather
  than discrete verbs.
- **OQ-9** — behaviour when the caster is at exactly 50% HP is defined by FR-1.1's strict
  inequality (no activation). Flagged only because it is the most likely off-by-one.

## 10. Acceptance Criteria

Behavioural:

- [ ] Casting Chakra below 50% HP restores HP; the amount matches the IDA-derived formula
      for the caster's level and version.
- [ ] Casting Chakra at or above 50% HP does not begin recovery, does not consume MP, and
      does not apply cooldown.
- [ ] HP never exceeds effective max HP after a Chakra heal.
- [ ] Damage taken during the recovery window is amplified by the level's damage-taken
      factor, computed server-side from the raw client-reported damage.
- [ ] A hit during recovery applies the amplified damage **and** cancels the pending heal.
- [ ] Movement during recovery cancels the pending heal; MP is not refunded.
- [ ] Death during recovery ends the state and the pending heal does not fire or revive.
- [ ] The recovery state self-clears on expiry, map change, channel change, and
      disconnect; no state leaks.
- [ ] The cast effect is visible to the caster and to other players in the map.
- [ ] MP is consumed exactly once per activation.
- [ ] No XP is awarded.

Evidence and record-keeping:

- [ ] `design.md` records the IDA-derived heal formula with IDB names, function addresses,
      and the versions examined; any version that could not be derived is listed explicitly
      (FR-7.5, FR-9.4).
- [ ] `design.md` records the keydown verdict with evidence, and exactly one of
      `model_test.go` / research-doc P7 is corrected accordingly (FR-10.1–10.4).
- [ ] Any disagreement between the FR-6.2 reference table and the WZ values `atlas-data`
      actually serves is documented, with the WZ values used in the implementation.

Implementation and verification:

- [ ] Handler registered on `skill2.ChiefBanditChakra`; no raw wire-id comparison anywhere
      in the change (`tools/skill-job-id-guard.sh` clean).
- [ ] Unit tests cover: the activation gate at 49%/50%/51%, the formula across the level
      range, the max-HP clamp, the amplifier at Lv1 (2.00×) and Lv30 (1.12×), each
      interruption path, and state expiry.
- [ ] `go test -race ./...` clean in every changed module.
- [ ] `go vet ./...` clean in every changed module.
- [ ] `go build ./...` clean in every changed service.
- [ ] `docker buildx bake atlas-<svc>` clean for every service whose `go.mod` was touched.
- [ ] `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`, and `tools/lint.sh --check`
      clean from the repo root.
- [ ] Code review run (`superpowers:requesting-code-review`) before the PR is opened.

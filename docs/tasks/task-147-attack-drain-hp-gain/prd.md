# Attack-Side Drain HP Gain (Drain / Energy Drain / Vampire) — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-07-09
---

## 1. Overview

Four drain-family attack skills are supposed to heal the attacker by a fraction of the damage they deal: Assassin **Drain** (4101005), Marauder **Energy Drain** (5111004), Thunder Breaker **Energy Drain** (15111001), and Night Walker **Vampire** (14101006). In Atlas today the attack pipeline applies the damage but never grants the heal — the gap is the `// TODO increase HP from Energy Drain, Vampire, or Drain` marker in `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go` (line 409 at time of writing), inside `processAttack` after per-monster damage application.

The reference behavior is Cosmic `AbstractDealDamageHandler.java:314-315` (verified against the local Cosmic checkout): for each damaged monster, the attacker is healed by `totalDamageToMonster × X / 100`, where `X` is the skill effect's per-level `x` value, capped by the monster's max HP and by half the attacker's (buff-inclusive) max HP. The heal is computed and applied **per damaged monster**, not once per attack, so multi-target casts (Vampire hits up to 4 monsters) heal per target with each target's heal capped individually.

Atlas already has every building block: skill id constants in `libs/atlas-constants/skill`, the per-level `x` value via the skill-effect pipeline (`effect.Model.X()`, same path MP Eater uses), monster max HP on the monster snapshot, buffed max HP via the `effective_stats` package (already consumed in this handler for venom), and `character.Processor.ChangeHP` (already used in the same function for skill HP costs). This task wires them together, following the existing MP Eater proc as the structural template.

## 2. Goals

Primary goals:
- Attacks made with any of the four drain-family skills heal the attacker per the Cosmic formula, per damaged monster.
- The heal uses the skill effect's level-dependent `X` percentage from the tenant's skill data (no hard-coded percentages).
- The heal is capped by the monster's max HP and by half the attacker's **effective** (buff-inclusive) max HP.
- Drain-heal failures are logged and swallowed — they never abort or delay the attack pipeline (same policy as MP Eater and venom).

Non-goals:
- Server-side validation that energy is fully charged for Marauder/Thunder Breaker Energy Drain (the client gates cast eligibility; Cosmic performs no server-side energy check in the heal path).
- Any other TODO in the post-attack block (Bandit Steal, Pick Pocket, Sacrifice HP loss, charges, Mortal Blow, etc.).
- Aran **Combo Drain** (2110000/21100005-family) — despite the name it is a buff-driven mechanic with its own TODO line; explicitly out of scope.
- New packets or packet changes — the HP gain rides the existing character stat-change path triggered by `ChangeHP`.
- MP Eater changes.

## 3. User Stories

- As an Assassin using Drain, I want to recover HP proportional to the damage I deal so that the skill functions as its tooltip describes ("absorb some of the damage dished out to the enemy as HP; at most MaxHP/2, and no more than the enemy's MaxHP").
- As a Night Walker using Vampire against multiple monsters, I want each damaged monster to contribute healing so that multi-target drains behave as in the reference client.
- As a Marauder or Thunder Breaker using Energy Drain, I want the lost HP of the monster converted into my HP so that the energy-charge payoff skill is worth casting.
- As a player whose HP gain would exceed the caps, I want the heal clamped (monster max HP, half my max HP) so that drain skills cannot be exploited on high-damage crits against weak monsters.

## 4. Functional Requirements

### FR-1 — Skill set

The heal applies **only** when the attack's skill id (`ai.SkillId()`) is one of:

| Skill | Id | Constant (`libs/atlas-constants/skill`) |
|---|---|---|
| Assassin Drain | 4101005 | `AssassinDrainId` |
| Marauder Energy Drain | 5111004 | `MarauderEnergyDrainId` |
| Thunder Breaker Energy Drain | 15111001 | `ThunderBreakerStage3EnergyDrainId` |
| Night Walker Vampire | 14101006 | `NightWalkerStage2VampireId` |

Constants must be referenced from `libs/atlas-constants/skill` — no numeric literals in the handler (DOM-21).

### FR-2 — Heal formula (per damaged monster)

For each damaged monster in the attack:

```
totalDamage = sum of the damage values in that monster's DamageInfo entry
rawHeal     = floor(totalDamage × X / 100)
heal        = min(monsterMaxHp, rawHeal, effectiveMaxHp / 2)
```

- `X` is `effect.Model.X()` from `skill.Processor.GetEffect(skillId, castLevel)`, where `castLevel` is the character's owned level of the skill (the same `sk.Level()` the handler already resolves for the attack).
- `monsterMaxHp` is the monster snapshot's `MaxHp()` (the same snapshot source the damage pipeline uses).
- `effectiveMaxHp` is the buff-inclusive max HP from the `effective_stats` package (`RestModel.MaxHp`) — **not** base `character.Model.MaxHp()` — so Hyper Body et al. raise the cap, matching Cosmic's `getCurrentMaxHp()`.
- Integer division/truncation follows Cosmic (`(int)` cast after the multiply): floor semantics.
- If `heal <= 0` (zero damage, X=0, effect lookup failure), no HP change is emitted.

### FR-3 — Reference X values (verified)

Verified against local WZ data (`Cosmic/wz/Skill.wz/*.img.xml`); these arrive at runtime through the tenant skill-effect pipeline and must NOT be hard-coded — the table below exists to validate the pipeline during testing:

| Skill | Levels | X range |
|---|---|---|
| 4101005 Drain | 1–30 | 16 → 45 (+1/level) |
| 5111004 Energy Drain | 1–20 | 11 → 20 (+1 every 2 levels) |
| 15111001 Energy Drain | 1–20 | 11 → 20 (+1 every 2 levels) |
| 14101006 Vampire | 1–20 | 4 → 10 |

### FR-4 — Pipeline wiring

- The heal executes in `processAttack`'s per-monster damage loop, after damage application for that monster — the same post-damage hook position as the MP Eater proc (`onDamageApplied` / `processDamageInfoEntry` deps in `character_attack_common.go`).
- A monster killed by the attack still yields the heal (Cosmic heals from damage dealt regardless of survival). If the monster snapshot cannot be fetched, skip the heal for that monster and log at debug.
- Effective stats are fetched **at most once per attack**, lazily, only when the attack skill is drain-family — reuse or mirror the existing lazy `loadVenomStats` pattern. If the effective-stats fetch fails, log and skip the heal (fail-safe: no heal rather than an uncapped heal).
- The skill-effect model (`se`) already fetched for the attack must be reused; do not re-fetch the effect per monster.

### FR-5 — HP application

- Apply via the existing `character.Processor.ChangeHP(field, characterId, amount)` command path (the same call the handler uses for `HPConsume`), with a positive amount.
- `ChangeHP` takes `int16`; the computed heal must be defensively clamped to `int16` range before the cast. (The formula already bounds it to `effectiveMaxHp/2` ≤ 15,000 in the v83-era HP ceiling, but the clamp guards against data anomalies.)
- Downstream clamping to the character's current max HP is owned by the character service's existing HP-change handling; this feature does not add its own current-HP arithmetic.

### FR-6 — Failure isolation and observability

- Every failure (effect lookup, monster snapshot, effective stats, emit) is logged and swallowed; the attack pipeline, broadcast, and projectile consumption proceed regardless.
- A successful drain heal logs at debug with caster id, skill id, monster id, damage total, X, and the final heal amount (mirroring the MP Eater proc log line).

### FR-7 — TODO removal

The `// TODO increase HP from Energy Drain, Vampire, or Drain` line is removed as part of the change. No other TODO lines in the block are touched.

## 5. API Surface

None. No new or modified REST endpoints, no new Kafka topics. The feature uses:

- Existing `CHANGE_HP`-style character command emission via `character.Processor.ChangeHP` (atlas-channel → atlas-character, already in use in this handler).
- Existing skill-effect REST lookup (`skill.Processor.GetEffect`) and effective-stats REST lookup, both already consumed by this handler.

## 6. Data Model

None. No new entities, fields, or migrations. All inputs (skill `x` values) already flow from atlas-data's skill-effect pipeline per tenant.

## 7. Service Impact

- **atlas-channel** — the only changed service. `socket/handler/character_attack_common.go` gains the drain-heal logic (a small pure helper for the cap math plus wiring into the per-monster post-damage hook), with unit tests for the helper.
- **libs/atlas-constants** — read-only dependency; all four skill constants already exist.
- No impact on atlas-character (existing HP-change consumer), atlas-data, atlas-ui, or deploy manifests.

## 8. Non-Functional Requirements

- **Multi-tenancy / versions:** the logic is version-independent server arithmetic; `X` values resolve per tenant from that tenant's skill data. No per-version branching, no opcode/template changes. See §8.1 for the supported-version matrix.

### 8.1 Version support (added post-rebase onto main, 2026-07-25)

This PRD was written 2026-07-09. GMS legacy versions **48.1 / 61.1 / 72.1 / 79.1** landed on main
2026-07-13 (`3d77511d0`, plus `8100ce299` task-178 and `e14765d7d` task-181), so they are in scope
for this task.

**The attack pipeline is wired for nine versions.** `CharacterMeleeAttack` / `CharacterRangedAttack`
/ `CharacterMagicAttack` handlers are present in the seed templates for gms 48, 61, 72, 79, 83, 84,
87, 95 and jms 185. They are **absent** from `template_gms_12_1.json` and `template_gms_92_1.json`,
so the feature is simply unreachable on those two — no work required.

**No per-version branching is required.** `processAttack` gates on skill ownership before anything
else: `character_attack_common.go:283-291` scans `c.Skills()` and destroys the session if the
character does not own the cast skill. A skill that does not exist in a given version's data can
never be owned, so the drain branch is structurally unreachable there. Version applicability is
therefore a *test-and-documentation* matter, not a code-gating one.

`libs/atlas-packet/model/attack_info.go` **is** now heavily version-gated (v61/v72/v79/v84/v95
gates at lines 31-61 and throughout `Decode`), but those gates govern head/trailer *framing* only —
`skillId` and `damageInfo` are plain struct fields (`attack_info.go:73,93`) populated on every
version. Nothing this feature consumes is version-dependent.

Skill availability by job-branch introduction floor (repo source:
`services/atlas-ui/src/lib/jobs/job-advancement-tree.ts:130-145` — Adventurer floor 1, Pirate node
floor 62, Cygnus branch floor 83):

| Version | 4101005 Drain | 5111004 Energy Drain | 15111001 TB Energy Drain | 14101006 Vampire |
|---|---|---|---|---|
| gms_48 | applicable | n-a (Pirate is v62+) | n-a | n-a |
| gms_61 | applicable | n-a (Pirate is v62+) | n-a | n-a |
| gms_72 | applicable | applicable | **unverified** | **unverified** |
| gms_79 | applicable | applicable | **unverified** | **unverified** |
| gms_83 | applicable | applicable | applicable | applicable |
| gms_84 | applicable | applicable | applicable | applicable |
| gms_87 | applicable | applicable | applicable | applicable |
| gms_95 | applicable | applicable | applicable | applicable |
| jms_185 | applicable | applicable | applicable | applicable |
| gms_12, gms_92 | attack handlers not wired — feature unreachable on these templates | | | |

**Unverified cells — honest status.** Whether Knights of Cygnus (and therefore Thunder Breaker /
Night Walker) exist in GMS v72/v79 is **not established** from artifacts available locally. The
repo's Cygnus floor of 83 is explicitly documented as a *display-curation* choice, not a data gate
(`job-advancement-tree.ts:112-116`), and the memory it cites only establishes that KoC exist *at*
v83 — an upper bound, not an introduction point. Two verification attempts were made and both were
inconclusive:

- Local WZ dumps cover only v83 (`Cosmic/wz`) and ms_1172. There is no v72/v79 `Skill.wz` /
  `String.wz` to read.
- An IDB string probe for `Cygnus` / `Noblesse` returned zero hits on v72 and v79 — but the
  **control run returned zero hits on v83 and v87 too**, where Cygnus definitively exists. Those
  names live in WZ, not the binary, so the probe is uninformative. It is recorded here so the test
  is not repeated.

To settle it: query atlas-data for skill `15111001` under a live v72/v79 tenant and see whether that
tenant's ingested skill data carries it. **This does not block implementation** — the ownership gate
makes both outcomes behaviourally identical. It only decides whether the v72/v79 Cygnus cells are
labelled `applicable` or `n-a` in the acceptance matrix.
- **Performance:** no new per-monster REST calls. Effective stats at most one (lazy) fetch per attack; monster snapshots reuse the pipeline's existing fetch path. The heal emit is one Kafka command per damaged monster, same order of magnitude as the existing damage emits.
- **Concurrency:** no new shared state; all computation is per-attack local. Tests must pass `go test -race`.
- **Determinism/testability:** the cap math is a pure function (damage, x, monsterMaxHp, effectiveMaxHp → heal) with table-driven unit tests using the project Builder pattern — no `*_testhelpers.go` files.

## 9. Open Questions

None blocking. One design-phase verification: confirm the tenant skill-effect pipeline actually surfaces `x` for these four skill ids in a live tenant (the MP Eater precedent uses the identical pipeline, so this is expected to hold; the FR-3 table gives the expected values).

## 10. Acceptance Criteria

- [x] Attacking with each of the four skill ids heals the attacker; attacks with any other skill id produce no drain heal (no behavior change).
- [x] Heal per damaged monster equals `min(monsterMaxHp, floor(totalDamage × X / 100), effectiveMaxHp/2)`, with `X` matching the FR-3 table for a spot-checked level of each skill.
- [x] Multi-target Vampire attack heals once per damaged monster, each individually capped.
- [x] A monster killed by the attack still contributes its heal.
- [x] Zero-damage entries, effect-lookup failures, and effective-stats failures produce no heal and never abort the attack (attack broadcast still goes out).
- [x] Heal amount defensively clamped to `int16` before `ChangeHP`.
- [x] Table-driven unit tests cover: X-percentage math, monster-max-HP cap, half-effective-max-HP cap, zero damage, and floor/truncation semantics.
- [x] The TODO line for drain heal is removed; adjacent TODOs untouched.
- [x] No per-version code branching is introduced; the feature relies on the skill-ownership gate for version applicability (§8.1).
- [x] `go test -race ./...`, `go vet ./...`, `go build ./...` clean in atlas-channel; `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`, and `tools/lint.sh --check` clean from the repo root; `docker buildx bake atlas-channel` if `go.mod` is touched (not expected).

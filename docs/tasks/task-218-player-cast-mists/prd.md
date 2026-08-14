# Player-cast mists II: Smokescreen, Flame Gear, Poison Bomb, Recovery Aura — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-12

---

## 1. Overview

Task-200 (#1255, `080472e0b`) built the player-cast mist mechanism and shipped
exactly one skill on it: Fire/Poison Mage Poison Mist (`2111003`). It
generalised the mist Kafka contract with `targetKind` (`CHARACTER` / `MONSTER`)
and `effectKind` (`DISEASE` / `DAMAGE_OVER_TIME`)
(`services/atlas-maps/atlas.com/maps/kafka/message/mist/kafka.go:26-35`), added
a monster-targeting tick path
(`services/atlas-maps/atlas.com/maps/tasks/mist_tick.go`,
`mist_tick_monster_test.go`), plumbed the WZ `lt`/`rb`/`time` fields into the
`atlas-channel` skill effect model, and introduced a version-blind per-skill
attack-cast handler registry keyed on `skill2.Identity`
(`services/atlas-channel/atlas.com/channel/skill/handler/registry.go:56-104`).
Its PRD explicitly listed the other four mist skills as non-goals: *"the
plumbing must not preclude them; nothing registers them."*

This task is that follow-up. It delivers the remaining four player-cast mists:

| Skill | Identity | Wire id | Mist behaviour |
|---|---|---|---|
| Shadower Smokescreen | `ShadowerSmokescreen` | 4221006 | Characters inside take no damage |
| Blaze Wizard Flame Gear | `BlazeWizardStage3FlameGear` | 12111005 | Monsters inside take periodic damage |
| Night Walker Poison Bomb | `NightWalkerStage3PoisonBomb` | 14111006 | Monsters inside take periodic poison damage |
| Evan Recovery Aura | `EvanStage8RecoveryAura` | 22161003 | Party members inside recover HP/MP periodically |

Two of the four (Flame Gear, Poison Bomb) fit task-200's existing
`MONSTER`/`DAMAGE_OVER_TIME` contract and are close to registration-only work.
The other two do not: Smokescreen and Recovery Aura are `CHARACTER`-targeted
with effects that have no matching `effectKind`, and Smokescreen additionally
requires a change to the client-facing `nType` discriminator and a hook in the
character damage-mitigation pipeline. Recovery Aura is further blocked by a
generator defect (§4 FR-0) that leaves every Evan skill unresolvable.

The backlog item that motivated this task quoted Flame Gear as `12101001`. That
is wrong; the repo binds `BlazeWizardStage3FlameGear` to `12111005`
(`libs/atlas-constants/skill/identities_gen.go:449`). This PRD uses the
repo-verified ids throughout.

## 2. Goals

Primary goals:

- Casting each of the four skills spawns a server-side mist anchored at the
  caster's position, sized by the skill effect's `lt`/`rb`, living for the
  skill effect's `time`, visible to every session in the field.
- Monsters standing inside a Flame Gear or Poison Bomb mist take periodic
  damage sourced to the casting character and to that skill's wire id.
- Characters standing inside a Smokescreen mist take no damage from monster
  attacks for as long as they remain inside.
- Party members standing inside a Recovery Aura mist periodically recover HP
  and/or MP.
- Every Evan skill (the `22xxxxxx` range, not just Recovery Aura) resolves from
  wire id to `skill2.Identity` on the versions that bind it, so identity-keyed
  dispatch works for Evan at all.
- The mist `effectKind` set and the `nType` derivation express these behaviours
  explicitly, so neither a future protective mist nor a future recovery mist
  needs another contract change.

Non-goals:

- Any change to Poison Mist (`2111003`) behaviour. Its path shipped in
  task-200 and is verified; this task must not regress it.
- Any change to the monster `AREA_POISON` mist. Its wire path predates
  task-200 and must stay byte-identical, including its `nType == 0`.
- PvP / character-vs-character mist effects.
- Reworking the `atlas-monsters` DoT damage tick, the
  `monster.ResolvePoisonDamage` magnitude formula, or the
  `atlas-buffs` periodic-effect engine.
- New coverage-matrix packet cells. `AffectedAreaCreated` / `AffectedAreaRemoved`
  are already implemented and registered in every seed template as of
  `ae3341511` (#1226, task-165); these skills reuse them unchanged.
- Broadening the wzsnapshot drain beyond what FR-0 requires. FR-0 fixes the
  Evan gap; a general re-drain of all ten snapshots is out of scope.

## 3. User Stories

- As a Shadower, I want Smokescreen to place a smoke cloud that protects me and
  my party from monster damage, so the skill has a defensive effect at all.
- As a Blaze Wizard, I want Flame Gear to leave a burning strip that damages
  monsters standing on it, so the skill contributes to killing them.
- As a Night Walker, I want Poison Bomb to leave a poison cloud that damages
  monsters, so the skill contributes to killing them.
- As an Evan, I want Recovery Aura to heal my party while they stand in it, so
  the skill functions as a support ability.
- As any player in the field, I want to see another player's mist appear and
  disappear, so the map state I see matches the server's.
- As a developer, I want the four skills to differ only in their registered
  descriptor, not in bespoke per-skill service code.

## 4. Functional Requirements

### FR-0 — Prerequisite: Evan skill identities are unresolvable (blocks Recovery Aura)

**Every skill in the `22xxxxxx` range is missing from every per-version
wire→Identity binding table.** Verified:

- `grep -cE "^\t22[0-9]{6}:" libs/atlas-constants/skill/version_gms_95_1_gen.go`
  → `0`. The same holds for v84, v87, v92, and jms_185.
- The only `Evan*` identities bound at v95 are the beginner-range
  `2001xxxx` skills (12 of them: `20010012`, `20011000`–`20011011`).
- `EvanStage8RecoveryAura` exists as an identity
  (`libs/atlas-constants/skill/identities_gen.go:590` = `22161003`,
  `libs/atlas-constants/skill/constants.go:3454`) but is bound to no wire id on
  any version.

Root cause: the per-version tables are generated by
`libs/atlas-constants/gen` from a pinned snapshot in
`libs/atlas-constants/gen/wzsnapshot/<version>.json`, joined with
`gen/semantics/<version>.yaml` (`gen/semantics.go:365-380`). Every snapshot uses
`stable: auto` — auto-bind each snapshot wire id whose value matches a canonical
identity token. But the snapshots were drained by the **jobs-union** method:
`GET /api/data/jobs?page[size]=200`, taking the union of each row's
`attributes.skills` array, because the skills list endpoint returned HTTP 400
(`gen/wzsnapshot/PROVENANCE.md`, drain timestamp 2026-07-30T12:37Z). The Evan
job documents were blank at drain time (the known `Skill.wz/Dragon/` subdirectory
defect), so the union contained zero Evan skills. `atlas-data` itself serves
these skills — they are visible in the web UI for GMS 84/87/92/95 and jms 185 —
so this is a snapshot-capture defect, not missing game data.

- **FR-0.1** The wzsnapshot for every version that provides Evan skills MUST be
  re-derived so the `22xxxxxx` range is present, and
  `gen/wzsnapshot/PROVENANCE.md` MUST record the corrected drain method and
  timestamp.
- **FR-0.2** The regenerated `version_*_gen.go` files MUST bind
  `22161003 → EvanStage8RecoveryAura` on each version whose live `atlas-data`
  serves it. The binding set MUST be derived from live `atlas-data`, not
  hand-authored, and not asserted from this PRD's table.
- **FR-0.3** `go run . -check` in `libs/atlas-constants/gen` MUST exit 0 after
  regeneration (no drift).
- **FR-0.4** Regeneration MUST NOT change any existing wire→Identity binding.
  A diff that alters a previously bound pair is a defect to investigate, not a
  result to accept — task-187's divergence semantics depend on those bindings
  and `tools/skill-job-id-guard.sh` is derived from them.
- **FR-0.5** If FR-0.1's re-drain shows a version does **not** serve `22161003`,
  Recovery Aura is simply not delivered on that version and the PRD's
  availability table (§7) MUST be corrected to match — the drain result wins
  over this document.

### FR-1 — Skill data: per-skill effect fields

- **FR-1.1** For each of the four skills, the `atlas-channel` effect model MUST
  expose non-zero `Duration()` (mist lifetime, ms) and a non-degenerate
  `LT()`/`RB()` rectangle for every level at which the skill is granted, on
  every version that binds it. These are the same fields task-200 plumbed
  (`data/skill/effect/rest.go`, `data/skill/effect/model.go`); no new reader
  work is expected for them.
- **FR-1.2** Recovery Aura's per-tick recovery magnitude MUST come from WZ, not
  from a constant. The design phase MUST identify which WZ node carries it
  (`hp` / `mp` / `x` / `y` are the candidates) by reading the ingested data for
  `22161003` and MUST plumb it through the effect model if it is not already
  exposed. Unit semantics MUST be normalised at the `atlas-data` reader,
  matching the existing `time` treatment (one conversion point,
  per task-190 FR-1.1).
- **FR-1.3** Flame Gear's monster status MUST be derived from the ingested WZ
  for `12111005` during the design phase — read the actual data and pick the
  status it describes. It MUST NOT be assumed to be `POISON` by analogy with
  Poison Mist. If the WZ describes a burn/DoT distinct from `POISON`, the
  design MUST state which `atlas-monsters` status carries it and whether that
  status already exists.
- **FR-1.4** If the ingested WZ for any provisioned version yields a zero or
  absent value for a field a skill requires, that is a data defect to report in
  the design doc — never a reason to hard-code a fallback.

### FR-2 — Mist contract: new effect kinds

The contract currently defines exactly two effect kinds
(`kafka/message/mist/kafka.go:31-35`). Smokescreen and Recovery Aura fit
neither.

- **FR-2.1** `EffectKind` MUST gain a value expressing "protect the characters
  inside from damage" (Smokescreen).
- **FR-2.2** `EffectKind` MUST gain a value expressing "periodically recover
  HP/MP for the party members inside" (Recovery Aura).
- **FR-2.3** The empty-string default MUST continue to mean `DISEASE` with
  `CHARACTER` targeting, so the pre-task-200 `atlas-monsters` `AREA_POISON`
  producer keeps working with no change.
- **FR-2.4** `CreateCommandBody` MUST carry whatever magnitude the new effect
  kinds need (e.g. Recovery Aura's per-tick HP/MP amounts) as explicit fields.
  Overloading `DiseaseValue` for a non-disease effect is prohibited.
- **FR-2.5** An unrecognised `effectKind` or `targetKind` MUST be rejected at
  the `atlas-maps` consumer with a warning naming the value, and MUST NOT
  create a mist. Silently falling back to `DISEASE` would apply the wrong
  effect to the wrong targets.

### FR-3 — Client `nType` discriminator (Smokescreen)

`AffectedAreaTypeFor` derives the client's `AFFECTEDAREA::nType` purely from
owner type (`mist/model.go:143-149`): character-owned → `1`, monster-owned →
`0`. That is insufficient for Smokescreen.

The client reads `nType == 2` as Smoke Screen specifically:
`CAffectedAreaPool::IsSmokeAreaByPoint` (v95 @0x434f40), and v83
`CAffectedAreaPool::Update` (@0x43109f) gates the fade-out animation on
`nType == 2` (`mist/model.go:110-114`). A Smokescreen mist sent as `nType 1`
will not be recognised by the client's smoke lookup and will not fade correctly.

- **FR-3.1** A Smokescreen mist MUST go out with `nType == 2`. A named constant
  MUST be added alongside `AffectedAreaTypeMobSkill` / `AffectedAreaTypeUserSkill`
  with a doc comment citing the client evidence.
- **FR-3.2** The `nType` derivation MUST be extended to account for the mist's
  effect kind, not owner type alone, while preserving both existing outcomes
  exactly: monster-owned → `0`, character-owned non-smoke → `1`.
- **FR-3.3** `nType` MUST remain derived inside `atlas-maps` and MUST NOT be
  added to `COMMAND_TOPIC_MIST`. It is a client wire detail; no producer should
  need the client's value table to create a mist (the rationale already recorded
  at `mist/model.go:139-142`).
- **FR-3.4** Flame Gear, Poison Bomb, and Recovery Aura MUST continue to use
  `nType == 1`. Sending `0` for a character-owned mist causes the caster to be
  billed an uninitialised `nDamage` — the live 1,434,803-damage self-hit
  task-200 diagnosed (`mist/model.go:126-133`). Any change to this derivation
  MUST have a regression test covering all four values.

### FR-4 — Smokescreen: damage protection

`atlas-channel` already runs a server-authoritative mitigation chain over one
damage-taken event (`socket/handler/character_damage.go:165-310`,
`processDamageTaken`), sourcing every mitigation from active buffs via
`extractBuffAmounts` (`character_damage.go:132-163`) — Magic Guard, Power Guard,
Meso Guard, Mana Reflection, Combo Barrier, Magic Shield, Guard, Infinity.

- **FR-4.1** A character standing inside a Smokescreen mist MUST take zero HP
  damage from monster-sourced attacks for as long as they remain inside.
- **FR-4.2** The protection MUST be server-authoritative. It MUST NOT be
  granted on a client claim, matching the existing pipeline's treatment of
  Power Guard and Mana Reflection (`character_damage.go:225-247`), where the
  amount is never taken from the wire.
- **FR-4.3** Protection MUST end when the mist expires or is cancelled, and
  MUST end for a character who leaves the rectangle. A character who walks out
  and is hit MUST take full damage.
- **FR-4.4** The design phase MUST choose between two mechanisms and justify
  the choice:
  - **(a) buff-mediated** — the `atlas-maps` mist tick applies a protective
    character status via `COMMAND_TOPIC_CHARACTER_BUFF`, and
    `extractBuffAmounts` gains a case for it. Fits the existing pipeline with
    no new cross-service query, at the cost of tick-granularity lag on entering
    and leaving the cloud.
  - **(b) positional query** — `processDamageTaken` queries live mist state for
    the character's position. Exact, but introduces a synchronous
    `atlas-channel` → `atlas-maps` dependency on the hot damage path.

  Option (a) is the recommended default: it matches how every other mitigation
  in this pipeline is sourced. Whichever is chosen, FR-4.3's leave-the-cloud
  behaviour MUST hold within one tick interval, and the design MUST state the
  worst-case lag explicitly.
- **FR-4.5** Smokescreen protection MUST compose with the existing mitigation
  chain without corrupting it: reflect (Power Guard / Mana Reflection) and Meso
  Guard amounts MUST NOT be computed from damage that Smokescreen zeroed.
- **FR-4.6** Party scoping MUST be explicit and tested: the design MUST state
  whether the mist protects every character inside or only the caster's party,
  derived from the client/WZ behaviour rather than assumed.

### FR-5 — Recovery Aura: party recovery

- **FR-5.1** Party members inside a Recovery Aura mist MUST periodically
  recover HP and/or MP, at the magnitude and cadence FR-1.2 resolves from WZ.
- **FR-5.2** Recovery MUST be scoped to the caster's party. A non-party
  character standing in the cloud MUST NOT be healed.
- **FR-5.3** Recovery MUST NOT exceed a character's maximum HP/MP, and MUST NOT
  affect a dead character.
- **FR-5.4** The recovery MUST be emitted as a command to the owning service
  rather than written directly, following the existing mist tick's pattern of
  emitting `COMMAND_TOPIC_CHARACTER_BUFF` / `COMMAND_TOPIC_MONSTER` rather than
  mutating state in `atlas-maps` (`tasks/mist_tick.go`).

### FR-6 — Monster-damaging mists: Flame Gear and Poison Bomb

- **FR-6.1** Both MUST create a `MONSTER` / `DAMAGE_OVER_TIME` mist reusing
  task-200's existing tick path with no change to that path's semantics.
- **FR-6.2** Poison Bomb MUST apply `POISON`. Its magnitude is target-derived
  and MUST be sent as `0`, exactly as Poison Mist does — `atlas-monsters`
  resolves it per monster at apply time via `monster.ResolvePoisonDamage` and
  overwrites any value sent (`tasks/mist_tick.go`, `applyStatusCommandProvider`
  doc comment).
- **FR-6.3** Flame Gear MUST apply the status FR-1.3 resolves. If that status
  carries a caster-derived magnitude (unlike `POISON`), the magnitude MUST be
  sent explicitly and MUST come from WZ.
- **FR-6.4** The re-apply cadence MUST remain strictly greater than the DoT
  tick interval `atlas-maps` emits (`monsterDotTickIntervalMs`, currently
  1000ms). Setting them equal makes the eligible damage window exactly zero and
  the mist deals no damage at any tuning — the failure mode documented at
  `skill/handler/poisonmist/poisonmist.go:33-63`. The design MUST state each
  skill's chosen cadence and show the resulting window.

### FR-7 — Cast dispatch and registration

- **FR-7.1** Each skill MUST be registered on the correct registry. The two
  registries are not interchangeable: `RegisterAttackCast` is for skills the
  client delivers on an ATTACK packet; `Register` is for USE_SKILL and doubles
  as the "this handler owns the HP/MP cost" signal that makes `processAttack`
  skip its generic cost block (`skill/handler/registry.go:56-79`). Registering
  an attack-delivered skill on the wrong one both never fires **and** silently
  zeroes its MP cost — the exact defect task-200 hit with `2111003`.
- **FR-7.2** For each of the four skills, the design phase MUST determine which
  packet the client actually delivers it on, from the skill's WZ shape (the
  presence of `damage` / `attackCount` / `mobCount` is the discriminator
  task-200 used) and from the client, and MUST record the evidence. It MUST NOT
  be assumed by analogy with Poison Mist — Smokescreen and Recovery Aura are
  plausibly USE_SKILL skills.
- **FR-7.3** Registration MUST be keyed on `skill2.Identity`, never on a raw
  wire id, so one registration covers every version that binds the skill
  (`registry.go:24-40`).
- **FR-7.4** Each handler MUST pass the **wire** skill id (not the Identity)
  as `SourceSkillId` on the CREATE command. The client compares it against its
  own WZ to select the rendering arm (v83 @0x431d50, v95 @0x437515), so it must
  be the id that version binds (`poisonmist.go` `SourceSkillId` comment).
- **FR-7.5** Each new handler subpackage MUST be added as a blank import to
  `skill/handler/registrations/registrations.go`. A handler package that is not
  imported never runs its `init()` and is silently absent.
- **FR-7.6** Per-skill handler logic that differs only in constants MUST be
  shared, not copy-pasted four times. The design MUST propose the factoring —
  the validation block in `poisonmist.go` (lifetime > 0, lifetime ≥ one tick,
  non-degenerate rectangle, lifetime under ceiling) is common to all five mist
  skills.

### FR-8 — Cast-time validation

- **FR-8.1** Every handler MUST reject a cast whose effect yields a
  non-positive lifetime, a lifetime shorter than one tick interval, or a
  degenerate rectangle (`rb.X <= lt.X || rb.Y <= lt.Y`), emitting nothing and
  logging a warning naming the skill and the reason.
- **FR-8.2** Every handler MUST reject (never truncate) an implausible
  lifetime. The ceiling MUST NOT clamp: the client computes its own
  `tEnd = tStart + 1000 * SKILLLEVELDATA::tTime` from its own WZ (v83
  @0x43200f, v95 @0x437c95), so a server-side clamp desynchronises the client,
  which keeps rendering a mist the server stopped ticking
  (`poisonmist.go:64-75`).
- **FR-8.3** A rejected cast MUST NOT roll back MP or cooldown. There is no
  rollback path, by design (task-200 FR-3.2 / FR-6.5); handlers return nil.
- **FR-8.4** A failure to load the caster's position or to emit the CREATE
  command MUST be logged at error level and MUST NOT fail the cast packet.

## 5. API Surface

No new REST endpoints.

Modified Kafka contract — `COMMAND_TOPIC_MIST` `CREATE`
(`services/atlas-maps/atlas.com/maps/kafka/message/mist/kafka.go`, mirrored in
`services/atlas-channel/atlas.com/channel/kafka/message/mist/kafka.go`):

- New `EffectKind` constants per FR-2.1 and FR-2.2.
- New magnitude field(s) on `CreateCommandBody` per FR-2.4.
- All additions are additive. Existing fields MUST NOT be renamed or retyped.

**Mirror discipline.** The mist contract exists in two Go modules
(`atlas-maps` owns it; `atlas-channel` carries a copy). They are separate
modules, so a json tag changed in one and not the other fails no build and
decodes into a zero-valued body at runtime. Both copies MUST be updated
together. (`tools/trade-contract-mirror-guard.sh` guards the trade contract
this way; the mist contract has no such guard — the design phase SHOULD
consider whether to add one.)

`COMMAND_TOPIC_MONSTER` `APPLY_STATUS` is reused unchanged. It is a **shared
topic**: every registered handler in `atlas-monsters` unmarshals every message
on it, so no key may be added, renamed, or retyped
(`tasks/mist_tick.go`, `applyStatusBody` doc comment).

Modified REST model — the `atlas-data` skill effect model gains whatever field
FR-1.2 requires, following the existing `LT`/`RB` pattern
(`data/skill/effect/rest.go:57-58,75-81`). Absent WZ nodes MUST default to zero
and MUST NOT change the serialized shape for skills that lack them.

## 6. Data Model

No new persisted entities and no database migration. Mists are in-memory,
registry-held, per-field state in `atlas-maps`
(`services/atlas-maps/atlas.com/maps/mist/registry.go`) with a lifetime bounded
by the skill's `time`.

Modified in-memory model — `mist.Mist`
(`services/atlas-maps/atlas.com/maps/mist/model.go:19`):

- New field(s) backing FR-2.4's magnitude, with getters, set via the existing
  builder.
- The new `nType` constant from FR-3.1 and the extended derivation from FR-3.2.

Multi-tenancy is unchanged: mists are scoped by `field.Model` (world, channel,
map, instance) and every command carries the tenant, resolved via
`tenant.MustFromContext(ctx)`.

Regenerated source — `libs/atlas-constants/skill/version_*_gen.go` and
`libs/atlas-constants/gen/wzsnapshot/*.json` per FR-0. These are generated
files; they MUST be regenerated, never hand-edited.

## 7. Service Impact

| Service | Change |
|---|---|
| `libs/atlas-constants` | FR-0: re-derive the Evan range into the affected wzsnapshots and regenerate the per-version binding tables. |
| `atlas-data` | FR-1.2 / FR-1.3: parse and expose whichever WZ fields Recovery Aura and Flame Gear need, if not already exposed. Unit normalisation at the reader. |
| `atlas-channel` | Four new handler subpackages under `skill/handler/`, blank-imported from `registrations`; shared validation helper (FR-7.6); mist contract mirror update; FR-4 Smokescreen mitigation in `socket/handler/character_damage.go`. |
| `atlas-maps` | New effect kinds in the mist contract and model; `nType` derivation extended (FR-3); tick handling for the protection and recovery effect kinds; magnitude fields on the model/builder. |
| `atlas-buffs` | Possibly none. If FR-4.4 option (a) is chosen, the protective status must be a stat `atlas-buffs` can carry and expire; the design must confirm whether the required temporary-stat type already exists. |
| `atlas-monsters` | None expected. Flame Gear's status (FR-1.3) may require a status type that does not yet exist; if so, that is an `atlas-monsters` change the design must call out. |
| Seed templates | None expected. `AffectedAreaCreated` / `AffectedAreaRemoved` writers are already registered in every template (`ae3341511`, #1226). This MUST be re-verified rather than assumed. |

**Availability.** Verified from the per-version binding tables at spec time:

| Skill | Versions binding it |
|---|---|
| Smokescreen (4221006) | all 11 — gms 12, 48, 61, 72, 79, 83, 84, 87, 92, 95; jms 185 |
| Flame Gear (12111005) | gms 72, 79, 83, 84, 87, 92, 95; jms 185 |
| Poison Bomb (14111006) | gms 72, 79, 83, 84, 87, 92, 95; jms 185 |
| Recovery Aura (22161003) | **unknown until FR-0 lands** — expected gms 84, 87, 92, 95 and jms 185 |

A skill is delivered on exactly the versions that bind it. No version-gated
branching in the handler is expected: dispatch resolves the wire id to an
Identity before lookup, so a version that does not bind the id simply never
dispatches.

## 8. Non-Functional Requirements

- **Performance.** Mist ticking is per-field and already bounded. Adding two
  effect kinds MUST NOT change the tick's asymptotic cost: each tick still
  scans the field's occupants once per live mist. The design MUST state the
  chosen tick cadence for each new effect kind.
- **Bounded lifetime.** FR-8.2's reject-don't-clamp ceiling bounds worst-case
  mist lifetime. Mist count per field is bounded by cast rate and cooldown; the
  design MUST state whether an explicit per-field cap is warranted.
- **Concurrency.** `atlas-maps` field iteration is parallel in places
  (`ForEachInMap`), so any per-tick accumulator introduced for the recovery or
  protection effect MUST NOT be shared mutable state across that iteration.
- **Multi-tenancy.** Every command carries the tenant; mist state is
  field-scoped and therefore tenant-scoped. No cross-tenant leakage path is
  introduced.
- **Observability.** Every cast rejection MUST log at warn with the skill, the
  character, and the specific reason. Every successful cast MUST log the skill,
  level, position, rectangle, and lifetime, matching the existing Poison Mist
  handler's log lines.
- **Security.** No mitigation amount, magnitude, or duration may be taken from
  the wire. FR-4.2 is the load-bearing case: a client-claimed damage immunity
  would be a trivially exploitable invulnerability.

## 9. Open Questions

1. **Smokescreen mechanism** — buff-mediated vs positional query (FR-4.4).
   Recommendation is buff-mediated; the design phase decides and justifies.
2. **Smokescreen scope** — everyone inside, or the caster's party only
   (FR-4.6). Must be derived from client/WZ behaviour.
3. **Does a suitable protective temporary-stat type already exist** in
   `atlas-buffs` / `atlas-constants`, or must one be added? Per project
   convention, `libs/atlas-constants` MUST be checked before defining anything
   new (DOM-21).
4. **Flame Gear's status** (FR-1.3) — resolve from WZ. If it needs a status
   `atlas-monsters` does not have, scope grows.
5. **Recovery Aura's magnitude node** (FR-1.2) — which WZ node, and is it
   already exposed on the effect model?
6. **Which packet delivers each cast** (FR-7.2) — attack vs use-skill, per
   skill, with evidence.
7. **Does FR-0's re-drain confirm Recovery Aura on gms 84/87/92/95 and
   jms 185?** The user reports seeing it in the web UI for those versions; the
   drain result is authoritative (FR-0.5).
8. **Should a mist-contract mirror guard be added** (§5), given the trade
   contract needed one for the same two-module divergence hazard?

## 10. Acceptance Criteria

**Prerequisite**

- [ ] `grep -cE "^\t22[0-9]{6}:" libs/atlas-constants/skill/version_gms_95_1_gen.go` returns non-zero, and `22161003` maps to `EvanStage8RecoveryAura` on every version that serves it.
- [ ] `go run . -check` in `libs/atlas-constants/gen` exits 0.
- [ ] No previously existing wire→Identity binding changed (FR-0.4), demonstrated by diff.
- [ ] `gen/wzsnapshot/PROVENANCE.md` records the corrected drain method and timestamp.

**Per skill (all four)**

- [ ] Casting spawns a server-side mist at the caster's position, sized by the effect's `lt`/`rb`, living for the effect's `time`.
- [ ] Every session in the field sees the mist appear on cast and disappear on expiry.
- [ ] Registered on the registry matching the packet the client actually delivers, with the evidence recorded (FR-7.2).
- [ ] Registered by `skill2.Identity`, and blank-imported in `registrations.go`.
- [ ] `SourceSkillId` on the CREATE command is the wire id, not the Identity.
- [ ] Casts with non-positive lifetime, sub-tick lifetime, degenerate rectangle, or implausible lifetime are rejected with a named warning, emit nothing, and roll back neither MP nor cooldown.

**Smokescreen**

- [ ] Goes out with `nType == 2`; regression test covers all four `nType` outcomes (monster → 0, three character-owned cases).
- [ ] A character inside takes zero HP damage from a monster attack.
- [ ] A character who leaves the rectangle takes full damage within the stated worst-case lag.
- [ ] Protection ends on mist expiry and on cancellation.
- [ ] Protection is not grantable by client claim.
- [ ] Reflect and Meso Guard amounts are not computed from zeroed damage.

**Flame Gear / Poison Bomb**

- [ ] Monsters inside take periodic damage sourced to the caster and to the skill's wire id.
- [ ] Poison Bomb sends `POISON` with magnitude 0; the applied magnitude matches `ResolvePoisonDamage`.
- [ ] Flame Gear applies the WZ-derived status, with a WZ-derived magnitude if caster-derived.
- [ ] Re-apply cadence exceeds the emitted DoT tick interval; the resulting non-zero damage window is asserted in a test.

**Recovery Aura**

- [ ] Party members inside periodically recover HP/MP at the WZ-derived magnitude.
- [ ] A non-party character inside is not healed.
- [ ] Recovery never exceeds max HP/MP and never affects a dead character.

**Regression**

- [ ] Poison Mist (`2111003`) behaviour is unchanged; its existing tests pass untouched.
- [ ] The monster `AREA_POISON` mist is unchanged, including `nType == 0`.
- [ ] Both copies of the mist contract are updated identically.
- [ ] `COMMAND_TOPIC_MONSTER` `APPLY_STATUS` gained no key, and no key was renamed or retyped.

**Build & verification** (per CLAUDE.md)

- [ ] `go test -race ./...` clean in every changed module.
- [ ] `go vet ./...` clean in every changed module.
- [ ] `go build ./...` clean in every changed service.
- [ ] `docker buildx bake atlas-<svc>` clean for every service whose `go.mod` was touched.
- [ ] `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`, `tools/buff-duration-guard.sh`, and `tools/skill-job-id-guard.sh` clean.
- [ ] `tools/lint.sh --check` clean.
- [ ] Code review run before PR.

# Pick Pocket Meso Spawn (Chief Bandit 4211003) — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-07-10
---

## 1. Overview

Pick Pocket (Chief Bandit skill 4211003) is a buff that, while active, gives each
damage line of qualifying attacks a chance to spawn a meso drop at the struck
monster's position. In Atlas today the buff half already works: atlas-data maps the
skill's statup to `TemporaryStatTypePickPocket` with amount `X`
(`services/atlas-data/atlas.com/data/skill/reader.go:296`), atlas-buffs stores and
serves it, and the client shows the buff icon. The proc half is absent — the attack
pipeline carries a bare `// TODO apply Pick Pocket`
(`services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go:408`)
— so no mesos ever drop and the skill is functionally dead.

This task implements the proc in atlas-channel's common attack handler, mirroring
the Cosmic reference behavior (`AbstractDealDamageHandler.java:290-313`, verified
against local Cosmic source): per damage line on a whitelisted skill, roll the Pick
Pocket effect's prop; on success spawn a meso drop near the monster via the existing
atlas-drops `SPAWN` Kafka command, which already carries a `Mesos` field.

The precedent for this shape of change is the MP Eater proc in the same file
(`mpEaterTryProc`, character_attack_common.go:206): a self-contained, failure-swallowing
proc evaluated inside the attack pipeline that emits a command to another service.

## 2. Goals

Primary goals:
- Pick Pocket procs per damage line on whitelisted attacks while the buff is active, spawning meso drops the attacker can pick up.
- Behavior matches the Cosmic reference formula and gating (documented deviations aside).
- Proc failures (buff lookup errors, effect lookup errors, emit errors) never abort or delay the surrounding attack pipeline.

Non-goals:
- Meso Explosion (4211006) consuming/destroying spawned mesos (separate TODO at character_attack_common.go:407).
- Bandit Steal, or any other TODO in the same block.
- Changes to buff application, buff rendering, or the atlas-buffs service.
- Drop-time staggering (Cosmic's 100 ms per-drop delay) — see Deviations.
- GM special-casing (Cosmic grants GMs max skill level) — see Deviations.

## 3. User Stories

- As a Chief Bandit/Shadower player, I want mesos to drop from monsters I hit while Pick Pocket is active so that the skill does what its tooltip promises.
- As a Chief Bandit in a party, I want the spawned mesos to be immediately lootable by me so that Pick Pocket feeds Meso Explosion gameplay (future task).
- As a server operator, I want the proc to be tenant-correct and non-blocking so that a drop-service hiccup cannot break attacks.

## 4. Functional Requirements

### 4.1 Gating (all must hold, evaluated once per attack)

1. The attacking character has an active, unexpired buff whose `Changes()` include
   stat type `PICK_POCKET` (`charconst.TemporaryStatTypePickPocket`). Lookup via the
   existing `character/buff` processor (`GetByCharacterId`, REST to atlas-buffs) —
   one lookup per attack, not per damage line.
2. The attack's skill id is in the Cosmic whitelist:

   | Skill | atlas-constants Id |
   |---|---|
   | Basic (regular) attack | `0` (i.e. `ai.SkillId() == 0`) |
   | Double Stab | `skill.RogueDoubleStabId` (4001334) |
   | Savage Blow | `skill.BanditSavageBlowId` (4201005) |
   | Assaulter | `skill.ChiefBanditAssaulterId` (4211002) |
   | Band of Thieves | `skill.ChiefBanditBandOfThievesId` (4211004) |
   | Assassinate | `skill.ShadowerAssassinateId` (4221001) |
   | Taunt | `skill.ShadowerTauntId` (4221003) |
   | Boomerang Step | `skill.ShadowerBoomerangStepId` (4221007) |

   No other skill procs Pick Pocket, regardless of attack type.

### 4.2 Effect parameters

3. `maxmeso` = the PICK_POCKET stat change's `Amount()` from the active buff (this is
   the skill effect's `X` captured at buff application — decision: use the
   buff-captured value, do NOT re-read the character's current skill level at attack
   time).
4. `prop` = the Pick Pocket effect's proc chance, resolved via the skill-effect
   processor (`GetEffect(4211003, level)`) at the buff's captured `Level()`.
   Same prop-roll semantics as MP Eater (`mpEaterShouldProc`): `prop >= 1.0` always
   procs; `prop <= 0` never procs.
5. If `maxmeso <= 0` or the effect lookup fails, skip the proc for this attack
   (log at debug/error, do not abort the attack).

### 4.3 Per-damage-line roll and meso amount

6. For each monster in `ai.DamageInfo()`, for each individual damage line against
   that monster: roll once against `prop`. Independent rolls; multi-line skills
   (e.g. Boomerang Step) can proc multiple times per attack.
7. On success, meso amount = `min(max(damage/20000 × maxmeso, 1), maxmeso)`,
   computed in floating point then truncated, exactly as Cosmic:
   `Math.min((int) Math.max(((double) d / 20000.0) * maxmeso, 1), maxmeso)`.
   A damage line of 0 still yields 1 meso if the roll succeeds (Cosmic behavior).
8. Deviation from Cosmic (deliberate): Cosmic's per-line loop runs the damage value
   through a Java int-overflow dance (`eachd += Integer.MAX_VALUE` then re-correct)
   that nets out to `d − 2` for normal positive damage — an artifact of its
   negative-damage wire encoding. Atlas decodes damage as unsigned; use the actual
   damage value `d`. The ±2 meso difference is immaterial.

### 4.4 Drop spawn

9. Each successful roll emits one atlas-drops `SPAWN` command
   (`COMMAND_TOPIC_DROP`, `CommandTypeSpawn`) with:
   - `Mesos` = computed amount, `ItemId` = 0, `Quantity` = 0
   - `X` = monster.x + uniform random in [−50, 49], `Y` = monster.y
   - `DropType` = 2 (free-for-all, per Cosmic `(byte) 2`)
   - `OwnerId` = attacking character id, `OwnerPartyId` = 0 (FFA type makes party
     ownership moot; design may confirm against atlas-drops handling)
   - `DropperId` = monster's unique id, `DropperX`/`DropperY` = monster position
   - `PlayerDrop` = true (per Cosmic)
   - `Mod` = the value atlas-drops expects for a fresh airborne drop (design phase
     confirms against atlas-drops/atlas-monsters usage; do not invent)
10. All drops for one attack are emitted immediately, in damage-line order.
    Deviation from Cosmic (agreed): no 100 ms per-drop stagger. Acceptable visual
    difference; revisit only if it looks wrong in testing.
11. Monster position comes from the monster snapshot already fetched in the damage
    pipeline (`mp.GetById`); if the snapshot fetch fails, skip that monster's procs.

### 4.5 Failure isolation

12. Every failure path (buff REST error, effect lookup error, monster fetch error,
    Kafka emit error) logs and returns without affecting damage application,
    broadcast, or any other attack side effect — same contract as MP Eater.

## 5. API Surface

No new or modified REST endpoints.

Kafka (existing contract, new producer call site):
- atlas-channel gains a `SPAWN` command producer for `COMMAND_TOPIC_DROP` in its
  `drop` package (today the package only emits `REQUEST_RESERVATION`;
  `services/atlas-channel/atlas.com/channel/kafka/message/drop/kafka.go` already
  declares the `Command`/spawn body shape with `Mesos` and `Mod`).
- atlas-drops consumes `SPAWN` unchanged — `CommandSpawnBody.Mesos` already exists
  (`services/atlas-drops/atlas.com/drops/kafka/message/drop/kafka.go:126`).

## 6. Data Model

No schema or entity changes. No migrations. All required data already exists:
- Buff + PICK_POCKET stat amount: atlas-buffs (queried via existing REST).
- Pick Pocket effect prop: atlas-data skill effect (queried via existing skill processor).
- Drop persistence: atlas-drops in-memory/registry handling of meso drops, unchanged.

## 7. Service Impact

| Service | Change |
|---|---|
| atlas-channel | Primary. New proc logic in `socket/handler/character_attack_common.go` (replacing the line-408 TODO), a `SpawnCommandProvider`/processor method in the `drop` package, whitelist + formula helpers with unit tests. |
| atlas-drops | None expected (SPAWN with Mesos is an existing code path, exercised today by atlas-monsters kill drops). If design review finds meso-only spawns from a character-owned FFA source hit an unhandled edge, fix belongs there. |
| atlas-buffs, atlas-data | None. Read-only consumers of existing endpoints. |

## 8. Non-Functional Requirements

- **Multi-tenancy:** all REST and Kafka interactions carry tenant context exactly as
  the surrounding handler does (`tenant.MustFromContext` via existing processors/producers).
- **Performance:** at most one atlas-buffs REST call and one skill-effect lookup per
  attack (skill-effect lookups are already cached/cheap via atlas-data). No lookups
  when the skill id is not whitelisted — check the whitelist before the REST call.
- **Non-blocking:** proc evaluation and emission must not delay the attack
  broadcast path beyond the synchronous lookups above; emit errors are fire-and-forget.
- **Testability:** prop-roll and meso-amount math live in pure functions (mirroring
  `mpEaterShouldProc`/`mpEaterAbsorbAmount`) with table-driven unit tests, per the
  project Builder-pattern test conventions (no `*_testhelpers.go`).
- **Observability:** debug-level log per proc emission (character, monster, meso
  amount); error-level on swallowed failures.

## 9. Open Questions

- Exact `Mod` byte value for the SPAWN command (fresh airborne drop). Resolve in
  design phase by reading atlas-drops' consumer and atlas-monsters' existing SPAWN
  emissions — do not guess.
- Whether `DropType` 2 (FFA) needs `OwnerPartyId` populated for correct pickup
  semantics in atlas-drops, or whether owner fields are ignored for FFA. Resolve by
  reading atlas-drops reservation logic in design phase.
- Whether atlas-drops treats `Mesos > 0` with `ItemId == 0` cleanly on every code
  path (expiry, pickup, packet writing). Expected yes (monster meso drops use it);
  design phase confirms.

## 10. Acceptance Criteria

- [ ] With Pick Pocket active, a whitelisted attack's damage lines each roll the
      effect prop; successes spawn meso drops at the monster (x ± 50, monster y)
      visible to and lootable by the attacker.
- [ ] Meso amount per drop equals `min(max(d/20000 × X, 1), X)` for damage `d` and
      buff amount `X`, verified by unit tests including d=0, d huge (clamps to X),
      and X≤0 (no proc).
- [ ] Non-whitelisted skills (e.g. Meso Explosion 4211006, ranged/magic attacks)
      never proc, even with the buff active — unit-tested gate.
- [ ] No atlas-buffs REST call is made when the attack skill is not whitelisted.
- [ ] Without the buff, whitelisted attacks emit nothing and add no drop commands.
- [ ] Buff-lookup, effect-lookup, monster-fetch, and emit failures are logged and
      swallowed; the attack still applies damage and broadcasts (unit/integration
      coverage of at least the lookup-failure path).
- [ ] The `// TODO apply Pick Pocket` comment is removed.
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...` clean in atlas-channel;
      `docker buildx bake atlas-channel` succeeds; `tools/redis-key-guard.sh` clean.

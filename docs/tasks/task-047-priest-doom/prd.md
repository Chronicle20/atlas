# Priest Doom (Skill 2311005) — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-05-03
---

## 1. Overview

The Priest job skill **Doom** (id `2311005`, master level 30) is a non-damaging area
magic skill that polymorphs affected monsters into snails for the skill's
duration. In v83 the snail visual and the elemental-resistance normalization
are entirely client-side — they trigger off the `MORPH`/`DOOM` mask bit on the
server-broadcast `MonsterStatSet` packet. The server's responsibilities are
limited to: routing the cast, applying the `DOOM` monster-status entry,
broadcasting the status packet, and removing the status when its duration
expires. Reference behaviour comes from the Cosmic source tree
(`server/StatEffect.java:823-825`, `server/StatEffect.java:1531`,
`server/life/Monster.java:1087-1095`).

A reconnaissance pass (recorded in this task's design phase) found that the
plumbing for this skill is already largely assembled in the Atlas codebase:
the skill id, job grant, monster-status constant, atlas-data effect mapping,
magic-attack handler's empty-damage status-apply branch, the
`MonsterStatus.DOOM` mask bit on the wire, the `STATUS_APPLIED` Kafka event,
and the channel-side `MonsterStatSet` broadcast all exist today. The work
remaining for this task is therefore focused on (a) closing one specific
elemental-immunity gap that prevents Doom from sticking on element-resistant
mobs the way the source material intends, (b) confirming bosses are immune,
(c) producing a minimal end-to-end test that pins the cast → status →
packet path against regressions, and (d) wiring an explicit
ApplyStatus debug log that is searchable in production.

## 2. Goals

Primary goals:

- A Priest with skill `2311005` learned can cast Doom in a magic attack
  packet (no damage entries, monster ids in the affected list) and cause
  every legal target in the LT/RB area to receive the `DOOM` monster status
  for the skill's duration.
- The v83 client renders affected mobs as snails for the duration and
  resumes the original sprite at expiry without the player having to
  rejoin the map. (This is verified by the wire-level evidence: a
  `MonsterStatSet` with the DOOM bit, followed by `MonsterStatReset` with
  the same bit at expiry. Snail rendering itself is client-side.)
- Doom's status-apply path is exempt from the existing
  poison/freeze elemental-immunity gate so that an element-resistant mob
  (e.g., a fire-immune mob) does not silently reject the DOOM status.
- Bosses do not receive the DOOM status. The existing boss immunity
  rejects it; this PRD pins that with a test rather than adding new logic.
- Magic-reflect targets, per the existing handler short-circuit, do not
  receive Doom. This is the chosen behaviour and is pinned by a test.
- A grep-friendly debug log line is emitted at cast time identifying the
  caster, the affected monster ids, and the duration, so production
  diagnoses do not have to reconstruct the chain from the generic
  ApplyStatus log.

Non-goals:

- No other Priest skills (Mystic Door, Holy Symbol, Summon Dragon,
  Dispel) are in scope.
- No refactor of the magic-attack handler beyond what this skill needs.
- No server-side polymorph entity swap. Polymorph-to-snail is a v83
  client-side effect of the `DOOM` mask bit; the server does not change
  the spawned monster id.
- No server-side elemental damage recomputation while a mob is Doomed.
  Atlas-channel does not compute attack damage; the v83 client computes
  damage and sends it. The client interprets DOOM as snail-elemental and
  produces normal-element damage on its own.
- No XP award for the cast itself. Doom does no damage; XP from kills
  flows through the existing damage-attribution path when an unrelated
  attack finishes the mob.
- No new Kafka topic or event type. All wiring uses existing
  `APPLY_STATUS`, `STATUS_APPLIED`, `STATUS_EXPIRED`, and `STATUS_CANCELLED`.
- No work on the Solution test framework (task-042); per direction,
  tests use the existing per-package unit-test pattern that
  `services/atlas-channel/atlas.com/channel/skill/handler/heal/` follows.

## 3. User Stories

- As a Priest player, I want to cast Doom on a group of regular mobs so
  that they visibly turn into snails and become harmless for the duration.
- As a Priest player, I want Doom to land on element-resistant mobs (fire
  imps, ice mobs) so the skill is useful for its intended counter-niche.
- As a Priest player, I want Doom to *not* affect bosses so I do not
  waste MP and a cast on a target that the source material treats as
  immune.
- As a server operator diagnosing a stuck mob report, I want a single
  log line per Doom cast that names the caster, the targets, and the
  duration so I can reconstruct the timeline quickly.
- As a developer adding a future Priest skill or refactoring the magic
  attack path, I want the Doom test suite to flag any regression that
  silently breaks the cast → status → packet flow.

## 4. Functional Requirements

### 4.1 Cast intake

- The magic-attack packet handler
  (`services/atlas-channel/atlas.com/channel/socket/handler/character_attack_magic.go`)
  routes a packet whose `SkillId() == 2311005` through
  `processAttack` (the existing common path). No new dispatch table
  entry is required, and no entry is added to the per-skill
  `skill/handler/registry.go` registry — the empty-damage
  monster-status apply branch in `character_attack_common.go` already
  covers Doom's behaviour.
- The cast verifies the caster owns skill `2311005` and consumes MP and,
  if defined for the skill effect, HP. The existing
  `character_attack_common.go` cost block (lines 113-120) handles this
  because Doom is not registered in the per-skill
  `handler.Lookup` table; the generic path applies.

### 4.2 Target resolution

- For each `DamageInfo` entry in the magic attack packet (Doom carries
  one entry per affected monster id with an empty `Damages()` slice),
  the handler attempts a status apply.
- The reflect short-circuit
  (`character_attack_common.go:172-197`) continues to apply: if a
  magic-reflect mob is in the target list and within reflect range,
  Doom does not stick to that mob. The reflect short-circuit emits the
  reflect damage path and skips the status apply for that entry.
  No special exemption for Doom.

### 4.3 Status apply

- For each non-reflected target, the handler calls
  `monster.ApplyStatus(field, monsterId, characterId, 2311005, skillLevel,
  {"DOOM": 1}, duration)`. The status map and duration come from
  `effect.Model` populated by atlas-data's
  `services/atlas-data/atlas.com/data/skill/reader.go:351-352` mapping.
- The atlas-monsters consumer
  (`services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/consumer.go:91`)
  calls `ProcessorImpl.ApplyStatusEffect`, which:
  - Rejects the apply if the target is a boss
    (`processor.go:1093-1097`). DOOM is not in `isBossAllowedStatus`'s
    allow list, so this rejection is automatic and is pinned by a new
    test.
  - **NEW:** When the inbound status set contains `DOOM`, the
    elemental-immunity gate (`isElementallyImmune`,
    `processor.go:1116-1131`) returns `false, ""` for that effect
    regardless of the monster's resistance table. This lets Doom land
    on fire-/ice-/poison-/lightning-immune mobs.
  - Persists the effect to the registry, emits `STATUS_APPLIED`, and
    triggers a picker re-pick if the effect touches the picker.

### 4.4 Status broadcast

- The atlas-channel monster status consumer
  (`services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer.go`)
  receives `STATUS_APPLIED`, builds a `MonsterStatSet` packet whose
  `MonsterTemporaryStat` mask includes the `TemporaryStatTypeDoom` bit
  and whose payload encodes one stat value (`value=1`,
  `sourceId=2311005`, `sourceLevel=skillLevel`, `expiresAt=now+duration`),
  and broadcasts it to all sessions in the field. No code change is
  required here — `libs/atlas-packet/model/monster.go:108` already wires
  the bit in the mask order.

### 4.5 Status expiry

- The atlas-monsters status task expires the effect after `duration`
  ms and emits `STATUS_EXPIRED`. The atlas-channel consumer translates
  that into a `MonsterStatReset` packet whose mask includes the DOOM
  bit. The v83 client restores the original mob sprite. No code change
  required; the test suite in 4.7 verifies the round-trip.

### 4.6 Cast logging

- A `Debugf` line is emitted from
  `services/atlas-channel/atlas.com/channel/monster/processor.go`
  inside `Processor.ApplyStatus` (or a small wrapper) when the inbound
  `statuses` map contains `DOOM`. The line names the caster id, the
  monster id, the skill id, the skill level, and the duration. The
  generic ApplyStatus debug remains in place — this is an additional,
  Doom-targeted line. Format:
  `Doom: caster=[%d] monster=[%d] skill=[%d] level=[%d] duration=[%d]ms.`

### 4.7 Tests (unit-test pattern, not Solution)

- `services/atlas-monsters/atlas.com/monsters/monster/processor_test.go`
  - **DOOM bypasses elemental immunity:** monster with
    `resistances={"P": "1", "I": "1", "F": "1", "S": "1", "L": "1"}`,
    apply `{"DOOM": 1}` from a player skill, assert apply succeeds and
    `STATUS_APPLIED` is emitted.
  - **DOOM rejected on bosses:** monster with `boss=true`, apply
    `{"DOOM": 1}`, assert `boss immunity` error and no
    `STATUS_APPLIED` event.
  - **DOOM re-apply replaces the existing entry (refresh):**
    re-applying `{"DOOM": 1}` while DOOM is already active replaces the
    prior `StatusEffect` with the new one and emits a second
    `STATUS_APPLIED` event. This is the realized behaviour of
    `Model.AddStatusEffect`
    (`services/atlas-monsters/atlas.com/monsters/monster/builder.go:140-163`),
    which removes any same-type entry before appending the new one for
    every status except VENOM. The test pins this refresh semantics for
    DOOM specifically; the prior assumption that re-apply is a no-op was
    incorrect (see `design.md` §6 risk note).
- `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common_test.go`
  - **Doom magic attack with empty damages applies status:** an
    `AttackInfo` with `SkillId=2311005`, one `DamageInfo` whose
    `Damages()` is empty, runs through `processAttack`, and produces
    one `monster.ApplyStatus` call with `{"DOOM": 1}` and the effect's
    duration. Verified against an existing-style fake monster
    processor.
  - **Doom blocked by reflect:** target has a magical-reflect window
    that contains the caster; `processAttack` does not call
    `ApplyStatus` for that target and does emit the reflect path.
  - **Doom on multi-target spread:** three `DamageInfo` entries, one
    reflect-blocked, two clean — exactly two `ApplyStatus` calls.
- `services/atlas-data/atlas.com/data/skill/reader_test.go` (or the
  equivalent existing test file)
  - **Doom effect maps DOOM=1 with non-zero duration:** load the
    effect for skill `2311005` level 30 and assert
    `MonsterStatus()["DOOM"] == 1` and `Duration() > 0`. If a fixture
    is needed, add a minimal hand-crafted skill XML node rather than
    pulling the full WZ data.

## 5. API Surface

No new HTTP endpoints, no Kafka topic or event type changes, no command
schema changes. All wiring uses:

- `APPLY_STATUS` command on the existing monster command topic.
- `STATUS_APPLIED` and `STATUS_EXPIRED` events on the existing monster
  status event topic.
- `MonsterStatSet` and `MonsterStatReset` socket packets, encoded by
  the existing `MonsterTemporaryStat` mask machinery in
  `libs/atlas-packet/model/monster.go`.

## 6. Data Model

No schema changes. The `DOOM` status entry rides in the existing
`storedStatusEffect.Statuses` map persisted to Redis by
`services/atlas-monsters/atlas.com/monsters/monster/registry.go:85-101`.

## 7. Service Impact

| Service / Library | Change |
|---|---|
| `services/atlas-monsters` | `processor.go` — change `isElementallyImmune` (or its caller in `ApplyStatusEffect`) so that an effect containing `DOOM` returns `false, ""`. Add unit tests covering the bypass and the boss rejection. No interface changes. |
| `services/atlas-channel` | `monster/processor.go` — add a `Doom`-specific Debugf line when the inbound status set contains `DOOM`. `socket/handler/character_attack_common_test.go` — add tests for cast-to-ApplyStatus, reflect blocking, and multi-target spread. No production handler logic changes. |
| `services/atlas-data` | Add a unit test pinning the effect mapping for `2311005` (no production change). |
| `libs/atlas-packet` | None. The DOOM mask bit is already wired. |
| `libs/atlas-constants` | None. `PriestDoomId`, `StatusDoom`, and `TemporaryStatTypeDoom` are present. |
| `services/atlas-configurations` | None. Skill grant and level cap (30) are already in `seed-data/templates/template_gms_83_1.json:3094`. |

## 8. Non-Functional Requirements

- **Performance:** Doom adds at most one ApplyStatus emit per affected
  target (typically <=15 mobs in a single cast). No new poll, no new
  cache entry, no new map iteration beyond what the generic magic
  attack handler already does.
- **Multi-tenancy:** `ApplyStatusEffect` already takes the tenant via
  the processor context; no tenant-scope work is required.
- **Observability:** the new Doom Debugf line is the only added
  emission. No new metric, no new trace span — the generic
  ApplyStatus span and the existing per-monster status counter
  already cover this skill.
- **Security:** the cast verification path
  (`character_attack_common.go:97-100`) already disconnects a session
  whose `Skills()` does not include `2311005`. No new client-trust
  surface is introduced.
- **Backwards compatibility:** because the elemental-immunity bypass
  is gated on the inbound status set containing `DOOM`, no existing
  POISON/FREEZE-bearing skill flow is altered.

## 9. Open Questions

None. All scope/behaviour questions raised in the spec interview were
resolved before generating this document; the answers are folded into
sections 2 and 4. The `task-047-priest-doom/design.md` produced in the
next phase may surface further questions about the implementation
strategy (e.g., whether to add a third arm to `isElementallyImmune` or
to short-circuit before calling it); those are deferred to the design
phase by intent.

## 10. Acceptance Criteria

A reviewer accepts this task as done when, in the worktree branch:

- [ ] Casting Doom in a manual end-to-end (live channel, live monster)
  applies the DOOM mask bit on the wire, the v83 client renders the
  affected mob as a snail, and the original sprite returns at expiry —
  verified by packet capture or by visual inspection.
- [ ] The new `processor_test.go` cases in atlas-monsters pass:
  DOOM bypasses elemental immunity; DOOM is rejected on bosses; DOOM
  no-ops while already active.
- [ ] The new `character_attack_common_test.go` cases in atlas-channel
  pass: empty-damage Doom triggers ApplyStatus, reflect blocks Doom,
  multi-target spread routes correctly.
- [ ] The new atlas-data reader test pins
  `effect.MonsterStatus()["DOOM"] == 1` and `effect.Duration() > 0`
  for skill `2311005` level 30.
- [ ] `go build ./...` and `go test ./...` succeed in atlas-monsters,
  atlas-channel, and atlas-data.
- [ ] The Doom-specific Debugf log line appears in atlas-channel logs
  on a real cast and contains the caster id, monster id, skill id,
  level, and duration.
- [ ] No regression in adjacent skill flows: Heal still pays its MP
  cost exactly once; Cleric Bless and Cure paths are unchanged; the
  generic ApplyStatus debug log continues to appear for non-Doom
  status applies.
- [ ] No new Kafka topic, no new event type, no new HTTP route.

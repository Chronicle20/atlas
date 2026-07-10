# Priest Dispel — Party Debuff Cure — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-07-10
---

## 1. Overview

Priest Dispel (skill `2311001`) is a dual-effect skill: it "cancels out all spell
effects of the enemies within a certain area, along with curing everyone in the
party of all sicknesses" (in-game description). Atlas currently implements only
the mob half — `applyToMobs` in `services/atlas-channel/atlas.com/channel/skill/handler/common.go`
classifies Dispel as a magic-class cancel (`isCrashOrDispel` / `dispelSkillClass`)
and cancels mob buffs with magic-reflect awareness. The party half is missing
entirely: casting Dispel does nothing for debuffed party members.

The infrastructure for the party half already exists on both ends. The client's
affected-party-member bitmap is decoded and exposed as
`info.AffectedPartyMemberBitmap()` (used by the generic party-buff apply at
`common.go:110`), map-wide party member selection exists as
`SelectPartyMembersInMap` (`skill/handler/recipients.go`), and `atlas-buffs`
already consumes a `CANCEL_BY_TYPES` command (`CommandTypeCancelByTypes`,
`kafka/consumer/character/consumer.go:82`) whose processor
(`character/processor.go` `CancelByStatTypes`) cancels matching buffs and emits
the standard `EXPIRED` status events that `atlas-channel` already turns into
client buff-cancel packets. The only missing pieces are a channel-side
`CANCEL_BY_TYPES` producer and a per-skill Dispel handler that wires the two
together.

Reference behavior is Cosmic's `StatEffect.applyTo` → `isDispel() &&
makeChanceResult()` → `Character.dispelDebuffs()`
(`Character.java:2759-2766`), which cures exactly six diseases: CURSE,
DARKNESS, POISON, SEAL, WEAKEN, SLOW. Notably ZOMBIFY, SEDUCE, and CONFUSE are
*not* dispellable (those belong to `purgeDebuffs()`, the cure-all-abnormal
path used by items/GM skills).

## 2. Goals

Primary goals:
- Casting Dispel cures the six dispellable debuff stat types on the caster and
  on party members in the same map selected by the client's affected-member
  bitmap.
- The skill's WZ success chance (`prop`) is honored per recipient, matching
  Cosmic's per-`applyTo` `makeChanceResult()` roll.
- A reusable channel-side `CANCEL_BY_TYPES` buff producer exists for later
  consumers (task-156 SuperGM Heal+Dispel plans to use the same command).

Non-goals:
- SuperGM Heal+Dispel (`9101000`) — task-156.
- Any change to the existing mob-side Dispel path (`applyToMobs`, reflect
  handling, rect verification) — already implemented and out of scope.
- Curing ZOMBIFY / SEDUCE / CONFUSE (cure-all semantics, not Dispel).
- Implementing mob-skill disease *infliction* on characters. No channel-side
  path applying SEAL/POISON/etc. to characters was found in the scan; until
  one exists, live-play Dispel casts will typically find nothing to cancel.
  This task builds the cure half correctly regardless (verifiable by seeding
  a debuff via a direct buff `APPLY` command).
- atlas-buffs changes — `CANCEL_BY_TYPES` support is already complete there.

## 3. User Stories

- As a Priest, I want casting Dispel to remove Seal/Curse/Darkness/Poison/
  Weaken/Slow from me and my nearby party members so that the skill works as
  described.
- As a party member debuffed by a mob skill, I want my Priest's Dispel to cure
  me so that I can resume fighting (attack, cast, move at full speed).
- As an operator, I want the dispel emission visible in logs so that I can
  verify casts, recipients, and prop-roll outcomes when debugging.

## 4. Functional Requirements

### 4.1 Channel-side CANCEL_BY_TYPES producer

- **FR-1.** `atlas-channel`'s mirrored buff message package
  (`kafka/message/buff/kafka.go`) MUST gain `CommandTypeCancelByTypes =
  "CANCEL_BY_TYPES"` and `CancelByTypesCommandBody{ Types []string }`, matching
  the shapes `atlas-buffs` consumes
  (`services/atlas-buffs/atlas.com/buffs/kafka/message/character/kafka.go`).
- **FR-2.** `character/buff` (processor + producer) MUST expose a
  `CancelByTypes` method following the existing `Apply` curried style — field +
  types bound first, returning a per-character emitter — emitting to
  `EnvCommandTopic` (`COMMAND_TOPIC_CHARACTER_BUFF`) keyed like existing buff
  commands.

### 4.2 Dispel skill handler

- **FR-3.** A new `skill/handler/dispel` subpackage MUST register via
  `channelhandler.Register(skill2.PriestDispelId, Apply)` in `init()`, and be
  blank-imported from `skill/handler/registrations/registrations.go` (pattern:
  `heal`, `mysticdoor`). The handler runs from the per-skill dispatcher in
  `UseSkill` (`common.go:117`), after the mob-side `applyToMobs` call — the two
  halves stay independent.
- **FR-4.** Recipient selection MUST be the caster plus
  `SelectPartyMembersInMap(l, ctx, f, casterId, info.AffectedPartyMemberBitmap())`
  — map-wide, no rectangle limit. (The WZ lt/rb rect governs the *mob* half,
  which `applyToMobs` already enforces; the party cure is "everyone in the
  party", per the skill description and owner decision.)
- **FR-5.** The debuff set MUST be exactly:
  `CURSE, DARKNESS, POISON, SEAL, WEAKEN, SLOW`
  (`charconst.TemporaryStatTypeCurse/Darkness/Poison/Seal/Weaken/Slow`,
  `libs/atlas-constants/character/temporary_stat.go:24-38`). Cosmic parity:
  `Character.dispelDebuffs()`. The set is a package-level constant slice, not
  built inline per cast.
- **FR-6.** The skill's success chance MUST be rolled once per recipient
  (caster included) using the effect's `prop`, reusing the existing
  `propRollFunc` seam semantics from `common.go:45` (share or mirror the seam
  so tests can inject deterministic rolls). WZ-verified values (v83 Cosmic
  dump, `Skill.wz/231.img.xml` / 2311001): prop 34 at level 1 rising to 100 at
  level 20. A failed roll skips that recipient's cure; it never fails the cast.
- **FR-7.** One `CANCEL_BY_TYPES` command MUST be emitted per recipient that
  passes the prop roll, carrying the six stat types. Per-recipient emit
  failures are logged and do not abort remaining recipients (pattern: heal's
  per-recipient error handling).
- **FR-8.** The handler MUST emit a per-cast structured debug summary (caster,
  skill level, bitmap, recipients selected, cures emitted, prop-skipped count)
  following the `buildSummaryFields` precedent in `common.go:296`.

### 4.3 Downstream behavior (verification only — no changes)

- **FR-9.** No `atlas-buffs` change: its `CancelByStatTypes`
  (`character/processor.go:103`) already cancels matching buffs and emits
  `EXPIRED` status events per cancelled buff, which `atlas-channel`'s buff
  status consumer (`kafka/consumer/buff/consumer.go:95`) already converts to
  client buff-cancel packets. The task MUST verify this end-to-end path in
  acceptance, not reimplement it.

## 5. API Surface

No REST changes. Kafka only:

- **Producer (new, atlas-channel):** `COMMAND_TOPIC_CHARACTER_BUFF`, type
  `CANCEL_BY_TYPES`, body `{ "types": ["CURSE","DARKNESS","POISON","SEAL","WEAKEN","SLOW"] }`,
  envelope fields (worldId/channelId/mapId/instance/characterId) as per the
  existing `Command[E]` in `kafka/message/buff/kafka.go`.
- **Consumer (existing, atlas-buffs):** already handles this type; no change.

## 6. Data Model

None. No new entities, no persistence. Buff state lives in atlas-buffs'
in-memory registry as today.

## 7. Service Impact

- **`atlas-channel`** (only changed service):
  - `kafka/message/buff/kafka.go` — add `CommandTypeCancelByTypes` + body.
  - `character/buff/producer.go` + `processor.go` (+ mock if present) — add
    `CancelByTypes`.
  - `skill/handler/dispel/` — new handler subpackage + tests.
  - `skill/handler/registrations/registrations.go` — blank import.
- **`atlas-buffs`** — no change (verified: command, processor, event emission
  all present).
- **Coordination:** task-156 (`gm-hide-heal-dispel`, in design) plans the same
  channel-side `CancelByTypes` producer (its FR-8). task-163 builds it;
  whichever lands second rebases and consumes the shared producer. The debuff
  sets differ intentionally (task-156 purges a wider disease set).

## 8. Non-Functional Requirements

- **Multi-tenancy:** all emission via the existing tenant-aware producer
  chain (`tenant.MustFromContext(ctx)` implicit in `producer.ProviderImpl`);
  no tenant-specific behavior — stat-type strings are semantic keys, not
  client wire values (DOM-25 not implicated).
- **Performance:** one Kafka message per cured recipient per cast (≤6 party
  members); negligible.
- **Observability:** per-cast debug summary (FR-8); per-recipient emit errors
  at error level.
- **Testing:** Builder-pattern test setup (project rule — no
  `*_testhelpers.go`); deterministic prop-roll injection via the seam;
  registry/registration test mirroring `registry_test.go` precedent.

## 9. Open Questions

None blocking. Two recorded observations:

1. No mob→character disease infliction path exists yet (scan found no producer
   applying the six debuff stat types to characters). Party Dispel will be a
   correct no-op in live play until that lands; acceptance seeds debuffs via a
   direct buff `APPLY` command.
2. Whether the client sends a non-zero affected-member bitmap for Dispel casts
   on all supported versions is unverified; if a version sends an empty bitmap,
   only the caster is cured on that version. Verify during implementation
   testing; no design change expected either way.

## 10. Acceptance Criteria

- [ ] `skill/handler/dispel` registered for `skill2.PriestDispelId`; blank
      import present in `registrations/registrations.go`.
- [ ] Casting Dispel with a party emits one `CANCEL_BY_TYPES` command per
      prop-passing recipient (caster + bitmap-selected in-map members) with
      exactly the six stat types `CURSE, DARKNESS, POISON, SEAL, WEAKEN, SLOW`.
- [ ] Prop roll is per-recipient and test-injectable; a failed roll skips only
      that recipient.
- [ ] A seeded debuff (buff `APPLY` with a debuff stat type) on a party member
      is cancelled by a Dispel cast and the member's client receives the
      buff-cancel packet via the existing `EXPIRED` event path.
- [ ] Mob-side Dispel behavior is byte-for-byte unchanged (existing
      `applyToMobs` tests still pass, no diffs in that path).
- [ ] No changes under `services/atlas-buffs/`.
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...` clean in
      `atlas-channel`; `docker buildx bake atlas-channel` succeeds;
      `tools/redis-key-guard.sh` clean.
- [ ] Per-cast summary log line present with recipient/cure/prop-skip counts.

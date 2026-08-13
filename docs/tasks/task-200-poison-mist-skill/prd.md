# Poison Mist (Fire/Poison Mage, skill 2111003) — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-07

---

## 1. Overview

Poison Mist (`2111003`, Fire/Poison Mage, job 211) is a placed area-of-effect
skill. Casting it drops a poison cloud at the caster's feet, sized by the
skill's WZ `lt`/`rb` rectangle. The cloud is drawn for every player in the
field, persists for the skill's WZ `time`, and periodically poisons every
monster standing inside it. It is the archetypal "player-cast mist" — the same
mechanism later reused by Smokescreen, Flame Gear, Poison Bomb, and Recovery
Aura.

Atlas already owns most of the parts, but none of them are connected on the
player-cast side. `atlas-maps` has a complete mist subsystem (registry,
lifecycle processor, tick task, `COMMAND_TOPIC_MIST` / `EVENT_TOPIC_MIST`
contract) whose only producer is the monster AREA_POISON path in
`atlas-monsters` (`monster/processor.go:1055` `executeMist`) and whose only
effect is applying a *character* disease (`tasks/mist_tick.go:182-197`).
`atlas-channel` translates mist events into `AffectedAreaCreated` /
`AffectedAreaRemoved` (`kafka/consumer/mist/consumer.go:78-116`), and as of
`ae3341511` (#1226, task-165) every seed template registers both writers.
`atlas-monsters` already models `PLAYER_SKILL`-sourced POISON with periodic
damage (`monster/status_task.go:66`) and `atlas-channel` already has an
`ApplyStatus` producer for it. What is missing is (a) anything in the
skill-cast path that creates a mist, (b) any mist that targets monsters rather
than characters, and (c) the DoT magnitude fields in the skill data.

This task closes those three gaps. The plumbing is built generically — a mist
declares *what it targets* and *what effect it carries* — so the other four
mist skills become registration-only work later, but only `2111003` is wired
up here.

## 2. Goals

Primary goals:

- Casting Poison Mist spawns a server-side mist anchored at the caster's
  position, sized by the skill effect's `lt`/`rb`, living for the skill
  effect's `time`.
- Every session in the field sees the mist appear on cast and disappear on
  expiry, on all ten matrix-tracked client versions.
- Monsters inside the mist's rectangle take periodic poison damage for as long
  as the mist lives, sourced to the casting character and to skill `2111003`.
- The mist Kafka contract and the `atlas-maps` mist model express target kind
  and effect kind explicitly, so a second mist skill needs no contract change.
- `atlas-data` parses and serves the WZ DoT fields (`dot`, `dotInterval`,
  `dotTime`) that every mist and DoT skill needs.

Non-goals:

- Smokescreen (`4221006`), Flame Gear, Poison Bomb, Recovery Aura, or any other
  mist skill. The plumbing must not preclude them; nothing registers them.
- Any change to how the *monster* AREA_POISON mist behaves. Its wire path is
  already correct and already registered; this task must not regress it.
- PvP / character-vs-character mist effects.
- Reworking the `atlas-monsters` poison damage tick itself.
- gms_12. The v12 client does reference skill `2111003` directly
  (`CAffectedAreaPool::OnAffectedAreaCreated` @0x4167cc compares `nSkillID`
  against `130`/`131`/`2111003`), and the identity mapping exists
  (`libs/atlas-constants/skill/version_gms_12_1_gen.go`), but v12 has no third
  job and no coverage-matrix column. The code path is version-blind, so it will
  work if a v12 tenant ever grants the skill; it is not a deliverable.

## 3. User Stories

- As a Fire/Poison Mage, I want casting Poison Mist to place a visible poison
  cloud on the map so that the skill has any observable effect at all.
- As a Fire/Poison Mage, I want monsters standing in my mist to take poison
  damage over time so that the skill contributes to killing them.
- As any player in the field, I want to see another player's mist appear and
  disappear so that the map state I see matches the map state the server has.
- As a server operator, I want mist lifetime and tick cost bounded and
  observable so that a crowded map cannot be degraded by mist spam.
- As a developer adding Smokescreen later, I want to register a skill identity
  and a target/effect descriptor rather than re-plumb four services.

## 4. Functional Requirements

### FR-1 — Skill data: DoT fields (`atlas-data`)

- **FR-1.1** `skill/reader.go` MUST parse the per-level WZ integers `dot`,
  `dotInterval`, and `dotTime` alongside the existing `lt`/`rb`
  (`skill/reader.go:232-236`) and `time` (`skill/reader.go:171`) reads.
- **FR-1.2** Unit semantics MUST be normalised at the reader, matching the
  existing `time` treatment (`skill/reader.go:195-198`, task-054):
  - `dot` is a raw damage-per-tick integer — forwarded unscaled.
  - `dotInterval` is WZ **seconds** — converted to **milliseconds** at the
    reader.
  - `dotTime` is WZ **seconds** — converted to **milliseconds** at the reader.
  The effect model and REST model MUST carry milliseconds; no downstream
  service may re-scale. This mirrors the mobskill contract
  (`mobskill/reader.go`, task-190 FR-1.1) — one conversion point.
- **FR-1.3** Absent nodes MUST default to `0` and MUST NOT change the serialized
  shape for skills that do not have them (fields are omitted or zero-valued;
  existing consumers of the skill effect REST model must not break).
- **FR-1.4** The three fields MUST be exposed on the skill effect REST model and
  hydrated into the `atlas-channel` effect model
  (`data/skill/effect/rest.go`, `data/skill/effect/model.go`), following the
  existing `LT`/`RB` pattern (`rest.go:57-58,75-81`).
- **FR-1.5** For `2111003`, the resulting effect model MUST expose non-zero
  `Duration()` (mist lifetime, ms), `LT()`/`RB()` (rectangle), `Dot()`,
  `DotInterval()` (ms) and `DotTime()` (ms) for every level 1..N at which the
  skill is granted. If the ingested WZ for any provisioned version yields zero
  for `dot`, `dotInterval`, or `dotTime`, that is a data defect to report — not
  a reason to hard-code a fallback.

### FR-2 — Mist contract generalization (`COMMAND_TOPIC_MIST` / `EVENT_TOPIC_MIST`)

The current `CreateCommandBody`
(`services/atlas-maps/atlas.com/maps/kafka/message/mist/kafka.go:32-53`)
hard-codes a character-disease effect via `Disease` / `DiseaseValue` /
`DiseaseDuration`. It MUST be generalized:

- **FR-2.1** `CreateCommandBody` MUST gain a `targetKind` field with values
  `CHARACTER` and `MONSTER`, and an `effectKind` field with values `DISEASE`
  (apply a named status to the target) and `DAMAGE_OVER_TIME` (apply a
  damage-bearing status to the target).
- **FR-2.2** The existing `Disease` / `DiseaseValue` / `DiseaseDuration` /
  `TickIntervalMs` fields MUST be retained and reused as the generic
  status-name / magnitude / per-target-duration / tick-interval triple. Renaming
  is permitted only if every producer and consumer is updated in the same
  change; the wire JSON keys are the contract.
- **FR-2.3** An absent or empty `targetKind` MUST be treated as `CHARACTER` and
  an absent or empty `effectKind` as `DISEASE`, so the existing
  `atlas-monsters` AREA_POISON producer
  (`monster/processor.go:1065-1098` `buildMistCreateBody`) keeps working
  byte-for-byte unchanged until it is updated. Updating it to set the fields
  explicitly is in scope; changing its behavior is not.
- **FR-2.4** `CreatedBody` MUST carry enough for the channel to render the
  packet correctly for a player-cast mist: it already carries `ownerType`,
  `ownerId`, `sourceSkillId`, `sourceSkillLevel`, `type`, origin, `lt`/`rb`,
  and `duration`. It MUST additionally carry the `nType`/`elemAttr`/`skillDelay`
  values resolved per FR-5 rather than the channel hard-coding `0`
  (`consumer.go:87-99` currently passes literal `0` for phase, tStart, and
  relies on the constructor's positional args).
- **FR-2.5** The `Mist` model (`atlas-maps/mist/model.go`) MUST carry
  `targetKind` and `effectKind` with builder setters and getters, following the
  existing immutable model + `Builder` pattern.

### FR-3 — Player-cast mist creation (`atlas-channel`)

- **FR-3.1** A new skill handler subpackage MUST be registered against the
  version-blind identity `skill2.FirePoisonMagicianPoisonMist` via
  `skill/handler.Register` (`skill/handler/registry.go:39`), and blank-imported
  from `skill/handler/registrations`. Dispatch MUST go through the identity, not
  the raw wire id (task-187). The wire id happens to be `2111003` on all eleven
  provisioned versions, which does not license a raw compare.
- **FR-3.2** The handler MUST run after the generic cost / cooldown / buff steps
  in `UseSkill` (`skill/handler/common.go:104`), consistent with the other
  registered handlers. A failed mist creation MUST NOT roll back MP or cooldown.
- **FR-3.3** The mist origin MUST be the caster's current position at cast time,
  loaded through the existing caster-load seam (`loadCasterFunc`,
  `skill/handler/common.go:37`). If the caster position cannot be resolved, the
  handler MUST log and return without emitting — no mist, no error to the
  client.
- **FR-3.4** The handler MUST emit exactly one `CREATE` command on
  `COMMAND_TOPIC_MIST` with:
  - `ownerType` = `CHARACTER`, `ownerId` = the casting character id
  - `targetKind` = `MONSTER`, `effectKind` = `DAMAGE_OVER_TIME`
  - origin = caster x/y; `ltX/ltY/rbX/rbY` = the effect's `LT()`/`RB()`
  - status name = `POISON`, magnitude = the effect's `Dot()`
  - per-target duration = the effect's `DotTime()` (ms)
  - tick interval = the effect's `DotInterval()` (ms)
  - mist lifetime = the effect's `Duration()` (ms)
  - `sourceSkillId` = the **wire** skill id for the tenant's version (the value
    the client will compare against its own WZ), `sourceSkillLevel` = the cast
    level
- **FR-3.5** The mist lifetime MUST come from the skill effect's WZ `time`. The
  60 s `MistDurationCapMs` clamp used by the monster path
  (`atlas-monsters/monster/processor.go:1053`) MUST NOT be applied to
  player-cast mists — the client derives its own `tEnd` from the same WZ data
  (`v83 @0x43200f`: `tEnd = tStart + 1000 * SKILLENTRY::GetLevelData(...)`), so
  a server-side clamp would desynchronise the client's rendering from the
  server's authority. A separate, higher sanity ceiling MAY be applied only to
  reject absurd data (see FR-6.4).
- **FR-3.6** Repeated casts MUST each create a distinct mist. Superseding or
  merging overlapping mists from the same caster is out of scope.

### FR-4 — Monster-targeting mist tick (`atlas-maps`)

- **FR-4.1** `tasks/mist_tick.go` MUST branch on the mist's `targetKind`. The
  existing `CHARACTER` branch (position lookup per character in field, bounding
  box test, `APPLY` disease command on `COMMAND_TOPIC_CHARACTER_BUFF`) MUST be
  preserved unchanged in behavior.
- **FR-4.2** For `targetKind = MONSTER`, each tick MUST resolve the monsters
  whose position falls inside the mist's rectangle and emit one
  `APPLY_STATUS` command per monster on the monster command topic, with body
  shape matching `ApplyStatusCommandBody`
  (`atlas-channel/kafka/message/monster/kafka.go:44-52`): `sourceType` =
  `PLAYER_SKILL`, `sourceCharacterId` = the mist's owner id, `sourceSkillId` /
  `sourceSkillLevel` from the mist, `statuses` = `{"POISON": <magnitude>}`,
  `duration` = the mist's per-target duration, `tickInterval` = the mist's tick
  interval.
- **FR-4.3** Monster resolution MUST use the existing atlas-monsters rectangle
  endpoint (`worlds/{w}/channels/{c}/maps/{m}/instances/{i}/monsters/in-rect`,
  already consumed by `atlas-channel/monster/requests.go:12`). The
  `atlas-maps` monster client (`maps/monster/processor.go`) MUST gain the
  corresponding `GetInMapRect` method. Per-tick rect queries are accepted for
  this task; optimisation is explicitly deferred.
- **FR-4.4** Monster resolution MUST go through an injectable seam, matching the
  existing `posLookup` / `charsInField` seams (`tasks/mist_tick.go:106-133`), so
  the tick is unit-testable without a REST mock.
- **FR-4.5** All commands emitted for one mist in one tick MUST go through a
  single `message.Emit` / `message.Buffer` batch, matching the character branch
  (`mist_tick.go:182-197`).
- **FR-4.6** A failed monster lookup for one mist MUST be logged and MUST NOT
  abort the tick for other mists or other tenants.
- **FR-4.7** Expiry MUST continue to be driven by the existing `Expired()` check
  and `Destroy(..., ReasonExpired)` path (`mist_tick.go:168-172`), which already
  emits `MIST_DESTROYED` and therefore `AffectedAreaRemoved`.
- **FR-4.8** A player-cast mist MUST persist for its full lifetime regardless of
  what the caster does — leaving the map, changing channel, dying, or logging
  out MUST NOT cancel it. This matches the client, which owns no cancel trigger
  of its own.

### FR-5 — Client visibility and packet correctness

- **FR-5.1** `AffectedAreaCreated`'s `nType` value for a player-cast poison mist
  is currently unknown to us. It MUST be derived from the client, not guessed:
  read `CAffectedAreaPool::OnAffectedAreaCreated` and the code that consumes
  `AFFECTEDAREA::nType` (+0x4) in the gms_v95 IDB (PDB-backed) and cross-check
  against gms_v83, and record the finding in the design doc with an address
  citation. The mob-mist arms are already identified as skill ids `130`/`131`
  (`libs/atlas-packet/field/clientbound/affected_area_created.go:57-58`), which
  gives a known-good comparison point.
- **FR-5.2** `dwOwnerId` for a player-cast mist MUST be verified the same way —
  specifically whether the client uses it to exclude the owner from the mist's
  effect or purely for attribution, and whether a character id in that slot
  collides with the monster unique ids the mob path already sends.
- **FR-5.3** `skillDelay` MUST be sent as `0` unless FR-5.1's reading shows
  otherwise. It is a *draw delay* in units of 100 ms, not a lifetime
  (`affected_area_created.go:43-49`); a non-zero value hides the mist.
- **FR-5.4** `nElemAttr` MUST be set from the skill's element attribute if the
  client reads it for player mists; `0` otherwise, with the choice justified by
  the FR-5.1 reading.
- **FR-5.5** No change may be made to the encoding of an already-verified
  matrix cell. `SPAWN_MIST` is ✅ on v48/v61/v72/v79/v83/v84/v87/v95/JMS185 and
  `REMOVE_MIST` is ✅ on all but v92 (`docs/packets/audits/STATUS.md:337,340`).
  If FR-5.1/FR-5.4 change what values are sent, that is a payload change, not a
  layout change, and the fixtures must still pass.
- **FR-5.6** The v92 gaps MUST be closed as part of this task, since v92 is one
  of the ten in-scope versions: `SPAWN_MIST` × v92 is ❌ (opcode `0x140`) and
  `REMOVE_MIST` × v92 is 🟡ᶠ (opcode `0x141`). Both cells MUST promote to ✅ via
  the standard `docs/packets/audits/VERIFYING_A_PACKET.md` flow — byte fixture
  with a `packet-audit:verify` marker, pinned evidence record, regenerated
  matrix. The encoder already models v92 explicitly
  (`affected_area_created.go:67,141`), so this is expected to be a verification
  pass rather than a codec change.
- **FR-5.7** `packet-audit matrix --check`, `fname-doc --check`, and
  `operations --check` MUST all exit 0 on the final branch, with the matrix
  regenerated *after* merging main (`toolSha` reads git HEAD).

### FR-6 — Safety and bounds

- **FR-6.1** A mist whose resolved lifetime is `<= 0` MUST NOT be created; the
  handler logs and returns.
- **FR-6.2** A mist whose resolved tick interval is `<= 0` MUST NOT be created
  (`ShouldTick()` already returns false for a non-positive interval,
  `mist/model.go:178-183`, which would produce an invisible no-op mist).
- **FR-6.3** A mist whose rectangle is degenerate (`rb <= lt` on either axis)
  MUST NOT be created.
- **FR-6.4** A sanity ceiling on player-cast mist lifetime MUST exist to reject
  corrupt data, set high enough not to clamp any legitimate WZ value (the
  design doc fixes the constant against the actual ingested `time` values for
  `2111003` across provisioned versions).
- **FR-6.5** The cast MUST be gated by the skill's existing cost, cooldown, and
  ownership checks in `UseSkill` before the handler runs. The handler MUST NOT
  introduce a second, independent path to mist creation.

## 5. API Surface

No new REST endpoints.

**Modified — `atlas-data` skill effect REST model.** Three additive fields on
the existing skill effect payload:

```json
{
  "dot": 0,
  "dotInterval": 0,
  "dotTime": 0
}
```

`dotInterval` and `dotTime` are milliseconds (FR-1.2). Additive and
zero-defaulted; existing consumers are unaffected.

**New — `atlas-maps` → `atlas-monsters` REST consumption.** `atlas-maps` begins
calling the existing, unchanged endpoint:

```
GET worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/monsters/in-rect
    ?x1={x1}&y1={y1}&x2={x2}&y2={y2}&limit={limit}
```

**Modified — `COMMAND_TOPIC_MIST` `CREATE` body.** Additive:

```json
{
  "targetKind": "MONSTER",
  "effectKind": "DAMAGE_OVER_TIME"
}
```

Both default to the pre-existing behavior (`CHARACTER` / `DISEASE`) when absent
(FR-2.3).

**Modified — `EVENT_TOPIC_MIST` `MIST_CREATED` body.** Additive: the resolved
`nType`, `elemAttr`, and `skillDelay` values (FR-2.4).

**Reused, unchanged — monster command topic `APPLY_STATUS`.** Body shape per
`ApplyStatusCommandBody`
(`atlas-channel/kafka/message/monster/kafka.go:44-52`). `atlas-maps` becomes a
second producer of this command alongside `atlas-channel`.

Error cases are all internal: unresolvable caster position, unresolvable
monsters, or invalid skill data all result in a logged no-op. Nothing new is
surfaced to the client.

## 6. Data Model

No database entities. All mist state is in-memory and tenant-scoped in the
`atlas-maps` mist registry (`mist/registry.go`), which is authoritative for the
channel it runs on and is intentionally not persisted — a channel restart drops
mists, matching the existing monster-mist behavior.

Model changes:

- `atlas-maps/mist.Mist` gains `targetKind string` and `effectKind string`,
  with getters and `Builder` setters, preserving the private-fields +
  getters + `Builder` pattern (`mist/model.go`).
- `atlas-channel/data/skill/effect.Model` gains `dot int32`,
  `dotInterval int32` (ms), `dotTime int32` (ms), with getters, hydrated from
  the REST model.
- `atlas-data` skill effect model gains the same three fields.

Multi-tenancy: unchanged. The mist registry is already keyed by
`tenant.Model` (`registry.AllByTenant`, `mist_tick.go:166`), every emitted
command carries the tenant via the standard producer decorators, and the tick
task already fans out per tenant (`mist_tick.go:149-159`).

No migrations.

## 7. Service Impact

| Service | Change |
|---|---|
| `atlas-data` | Parse `dot` / `dotInterval` / `dotTime` in `skill/reader.go`; normalise seconds→ms; expose on the skill effect model and REST model. |
| `atlas-channel` | New `skill/handler/poisonmist` subpackage registered against `FirePoisonMagicianPoisonMist`; new `COMMAND_TOPIC_MIST` producer; blank import in `skill/handler/registrations`; effect model + REST hydration for the three new fields. |
| `atlas-maps` | `Mist` model gains target/effect kind; `mist.Processor.Create` populates them; `tasks/mist_tick.go` gains the `MONSTER` branch; `maps/monster` client gains `GetInMapRect`; new producer of monster `APPLY_STATUS`. |
| `atlas-monsters` | `buildMistCreateBody` sets `targetKind`/`effectKind` explicitly (behavior unchanged). Consumes `APPLY_STATUS` from a second producer — no code change expected; verify the existing consumer is producer-agnostic. |
| `libs/atlas-packet` | No codec change expected. v92 `SPAWN_MIST` / `REMOVE_MIST` fixtures and evidence records added (FR-5.6). |
| `atlas-configurations` | No template change expected — `AffectedAreaCreated` / `AffectedAreaRemoved` are already registered in all eleven templates as of `ae3341511` (#1226). Verify, do not re-add. |

## 8. Non-Functional Requirements

- **NFR-1 (tick cost).** Each active monster-targeting mist costs one rect query
  per tick. The default tick interval comes from WZ `dotInterval` (typically
  1 s). The design doc MUST state the expected worst-case query rate for a busy
  map and confirm it is within the atlas-monsters rect endpoint's budget. If it
  is not, the mitigation is a longer floor on tick interval — not silently
  dropping mists.
- **NFR-2 (isolation).** A slow or failing atlas-monsters MUST NOT stall the
  mist tick for other tenants or other mists. The existing per-tenant goroutine
  fan-out (`mist_tick.go:149-159`) plus per-mist error containment (FR-4.6)
  satisfies this; the design must not introduce a shared blocking call.
- **NFR-3 (multi-tenancy).** Every emitted command MUST carry tenant context.
  The `atlas-maps` monster rect call MUST be made under the tenant-scoped
  context, as the character position lookup already is
  (`mist_tick.go:163,184`).
- **NFR-4 (observability).** Mist creation, expiry, and per-tick monster-apply
  counts MUST be logged at a level that makes "the mist did nothing" diagnosable
  without a debugger. Creation-rejection reasons (FR-6.1–6.4) MUST be logged
  distinctly.
- **NFR-5 (no regression).** The monster AREA_POISON mist path MUST behave
  identically before and after. Its existing tests
  (`atlas-monsters/monster/processor_test.go:1162+`,
  `atlas-maps/tasks/mist_tick_test.go`) MUST pass unmodified except where the
  new contract fields are explicitly asserted.
- **NFR-6 (security).** The mist rectangle and lifetime MUST be derived from
  server-side skill data and the server's view of the caster's position — never
  from client-supplied values in the skill-use packet.

## 9. Open Questions

- **OQ-1** What `nType` value does the client expect for a player-cast poison
  mist, and does it drive rendering, the owner-exclusion check, or both? To be
  resolved by IDB reading during design (FR-5.1) — no implementation may start
  from a guess.
- **OQ-2** Does the client apply any client-side effect of its own for a mist
  whose `nSkillID` is `2111003` (the v12 client explicitly compares against it
  @0x4167cc), and if so does the server need to suppress or complement it?
- **OQ-3** Are `dot` / `dotInterval` / `dotTime` populated in the ingested WZ for
  `2111003` on every provisioned version, or does one or more version's Skill.wz
  parse produce zeros? This intersects the known
  "v95 skill `common` formula nodes unparsed" defect — v95 in particular must be
  checked before assuming data is available.
- **OQ-4** Should the poison magnitude be the raw WZ `dot`, or scaled by the
  caster's magic attack? This PRD specifies raw `dot` (decision made at spec
  time). If the observed in-game damage is visibly wrong, revisit — but not in
  this task.
- **OQ-5** Does `atlas-monsters`' `APPLY_STATUS` consumer stack or replace a
  POISON effect that is re-applied every tick by the same source? The intended
  behavior is *refresh* (each tick extends the existing effect rather than
  adding a parallel one). Verify against `monster/registry.go` before
  implementation; if it stacks, the tick interval must be reconciled with
  `dotTime`.

## 10. Acceptance Criteria

**Data**

- [ ] `atlas-data` parses `dot`, `dotInterval`, `dotTime`; interval and time are
      converted seconds→ms at the reader, with unit tests covering present,
      absent, and zero nodes.
- [ ] The three fields round-trip through the skill effect REST model into the
      `atlas-channel` effect model, with a test asserting the ms values.
- [ ] `GET /api/data/skills/2111003` on a live baseline returns non-zero
      `dot`, `dotInterval`, `dotTime`, `lt`, `rb`, and duration for each granted
      level — or the discrepancy is documented as a data defect (OQ-3).

**Cast path**

- [ ] `skill/handler.Lookup(skill2.FirePoisonMagicianPoisonMist)` resolves to the
      new handler, and the subpackage is blank-imported from `registrations`.
- [ ] Casting `2111003` emits exactly one `COMMAND_TOPIC_MIST` `CREATE` with
      `ownerType=CHARACTER`, `targetKind=MONSTER`,
      `effectKind=DAMAGE_OVER_TIME`, origin = caster position, bounds from
      `LT`/`RB`, `POISON` magnitude = `dot`, per-target duration = `dotTime`,
      tick interval = `dotInterval`, lifetime = effect duration — asserted by a
      unit test against a recording producer.
- [ ] Each of FR-6.1–6.4's rejection cases emits no command and logs a distinct
      reason, covered by tests.
- [ ] No MP/cooldown rollback path is introduced; the existing cost tests still
      pass.

**Effect path**

- [ ] A `targetKind=MONSTER` mist emits one `APPLY_STATUS` per monster inside
      its rectangle per tick, and none for monsters outside it — asserted by a
      unit test against injected monster positions and a recording producer.
- [ ] A `targetKind=CHARACTER` mist still emits character `APPLY` disease
      commands with unchanged body; existing `mist_tick_test.go` passes.
- [ ] A monster-lookup failure for one mist does not prevent other mists in the
      same tenant from ticking.
- [ ] Mist expiry emits `MIST_DESTROYED` exactly once and stops all ticking.

**Client path**

- [ ] `nType`, `dwOwnerId`, `skillDelay`, and `nElemAttr` for a player-cast mist
      are each justified in the design doc by an IDB address citation
      (FR-5.1–5.4).
- [ ] `SPAWN_MIST` × v92 and `REMOVE_MIST` × v92 both show ✅ in
      `docs/packets/audits/STATUS.md` on the final branch, backed by byte
      fixtures with `packet-audit:verify` markers and pinned evidence records.
- [ ] No previously-✅ mist cell degrades.
- [ ] `packet-audit matrix --check`, `fname-doc --check`, and
      `operations --check` all exit 0, with the matrix regenerated after merging
      main.

**End-to-end**

- [ ] On a live tenant, an FP Mage casting Poison Mist sees the cloud appear at
      their feet, a second character in the same map sees the same cloud, the
      cloud disappears at the WZ-specified time, and monsters standing in it
      lose HP periodically for the mist's duration.
- [ ] A monster AREA_POISON mist still spawns, still poisons characters, and
      still renders — verified on the same tenant.
- [ ] `go build ./...` and `go test ./...` pass in every touched service.

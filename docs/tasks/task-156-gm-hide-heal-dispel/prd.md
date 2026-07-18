# SuperGM Skills: Hide + Heal & Dispel — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-07-10
---

## 1. Overview

Atlas defines the full SuperGM skill line (`SuperGm...Id`, job `910`) as constants
and decodes their WZ effects in `atlas-data`, but none of the SuperGM skills are
actually *executed* in `atlas-channel`. Two of them are the subject of this task:

- **Heal + Dispel** (`SuperGmHealDispelId` = `9101000`) — a GM utility that restores
  HP and MP and purges status ailments (debuffs) from players in the map.
- **Hide** (`SuperGmHideId` = `9101004`) — a GM utility that makes the caster
  invisible and untargetable to other players, toggled on and off by re-casting.

Today both skills are effectively no-ops when cast. `SuperGmHealDispelId` is decoded
as an action/attack effect (it is in `atlas-data`'s `isCategory1` list, which forces
`buff = false`), and no per-skill handler is registered for it, so casting it only
plays the client animation and consumes MP. `SuperGmHideId` has **no statup mapping**
in the WZ reader, so it produces zero temporary stats; the generic buff-apply path in
`skill/handler/common.go` (guarded on `len(e.StatUps()) > 0`) is skipped entirely, and
there is no invisibility plumbing anywhere in the spawn/despawn broadcast paths.

This task implements both skills end to end in `atlas-channel`, following the
established active-skill handler pattern (Cleric Heal `skill/handler/heal`, Mystic Door
`skill/handler/mysticdoor`, and Resurrection from task-111). Handlers register via
`channelhandler.Register(skillId, Apply)` and are blank-imported from
`skill/handler/registrations/registrations.go`. They are dispatched from the generic
`UseSkill` orchestrator, which has already validated skill ownership/level, loaded the
WZ effect, and consumed MP + applied cooldown by the time the handler runs.

## 2. Goals

Primary goals:

- Implement a **Heal + Dispel** handler for `SuperGmHealDispelId` that, for **all
  players in the caster's map**, restores HP and MP from the skill's WZ recovery values
  and cancels all disease debuffs.
- Implement a **Hide** handler for `SuperGmHideId` that toggles the caster's GM
  invisibility: on cast the caster is hidden from other players (suppressed foreign
  spawn + despawn from current viewers) and untargetable; on re-cast the caster becomes
  visible again. Hide state persists across map changes until toggled off.
- Gate both casts to characters whose job is **SuperGM** (`job.SuperGmId` = `910`).
- Broadcast the skill-use animation consistently with the existing handlers.

Non-goals:

- The other seven SuperGM skills (Haste, Holy Symbol, Bless, Resurrection, Dragon Roar,
  Teleport, Hyper Body). Resurrection is already handled by task-111.
- The plain-GM variants (`GmHideId`, `GmHide`) unless they fall out for free.
- GM chat/admin commands (`!hide`, `!heal`, etc.) — this is skill-cast only.
- Any new REST endpoints, database tables, or migrations.
- Proximity-based mob aggro. Aggro in `atlas-monsters` is damage-driven, not
  proximity-driven, so there is no proximity-targeting to suppress; "untargetable by
  mobs" falls out of the caster not being visible/attackable, and no aggro rework is in
  scope.

## 3. User Stories

- As a **SuperGM**, I want to cast Hide so that I become invisible and untargetable to
  players while I observe or moderate a map, without being seen.
- As a **SuperGM**, I want to re-cast Hide to reveal myself again when I'm done.
- As a **SuperGM**, I want my Hide state to survive changing maps so I don't flicker
  into view every time I move.
- As a **SuperGM**, I want to cast Heal + Dispel to restore HP/MP and clear status
  ailments for everyone in a map (e.g. after a wipe or a debuff-heavy event).
- As a **regular player**, I should not see a hidden SuperGM's avatar, pets, or
  movement, and my client should not crash or desync when a SuperGM hides or reveals.
- As a **regular player** in the map, I should benefit from a SuperGM's Heal + Dispel
  (HP/MP restored, ailments cleared) regardless of whether I'm in the caster's party.

## 4. Functional Requirements

### 4.1 Skill dispatch & registration

- **FR-1.** A handler for `skill.SuperGmHealDispelId` (`9101000`) MUST be registered via
  `channelhandler.Register` in a new `skill/handler/healdispel` (or similarly named)
  subpackage, blank-imported from `registrations/registrations.go`.
- **FR-2.** A handler for `skill.SuperGmHideId` (`9101004`) MUST be registered via
  `channelhandler.Register` in a new `skill/handler/hide` subpackage, blank-imported
  from `registrations/registrations.go`.
- **FR-3.** Both handlers MUST be dispatched through the existing `UseSkill` orchestrator
  path (`skill/handler/common.go`) — i.e. they rely on its prior ownership/level
  validation, WZ effect load, and MP/cooldown application, and do not re-implement those.

### 4.2 SuperGM gating

- **FR-4.** Before performing any effect, each handler MUST verify the caster's job is
  SuperGM (`job.IsA(c.JobId(), job.SuperGmId)`). A non-SuperGM caster MUST be rejected:
  no HP/MP change, no dispel, no hide toggle, and no error surfaced to other players.
  The rejection MUST be logged at warn level.
- **FR-4.1.** The gate uses `job.SuperGmId` only (`910`); plain `GmId` is **not**
  sufficient for these two SuperGM skills.

### 4.3 Heal + Dispel behavior

- **FR-5.** Recipients are **all players in the caster's map**, on the same world and
  channel and field instance, including the caster. This requires a new map-wide
  (non-party) recipient selector, since existing selectors are party-bitmap scoped.
- **FR-6.** For each recipient, HP MUST be restored using the skill's WZ recovery values
  (`hp` / `hpR` recovery from the effect), and MP MUST be restored using the WZ mp
  values (`mp` / `mpR`). This requires exposing MP and recovery accessors on the
  channel-side `effect.Model` (see FR-15).
- **FR-6.1.** HP/MP MUST NOT exceed each recipient's effective max; deltas MUST be
  clamped against effective stats the same way the existing Heal handler clamps HP
  (fetch effective MaxHp/MaxMp per recipient; use the ChangeHP/ChangeMP command path).
- **FR-7.** For each recipient, all **disease debuffs** MUST be cancelled. The debuff set
  is exactly the `atlas-buffs` disease set:
  `STUN, POISON, SEAL, DARKNESS, WEAKEN, CURSE, SEDUCE, CONFUSE, UNDEAD, SLOW,
  STOP_PORTION`.
- **FR-8.** The dispel MUST be delivered via a **new channel-side buff producer** that
  emits `CancelByTypes` (or `CancelAll` scoped to the disease set) to `atlas-buffs`,
  which already supports the `CANCEL_BY_TYPES` command and emits the corresponding
  EXPIRED status events. The channel's buff consumer already broadcasts cancel to self
  and foreign sessions; the handler MUST NOT hand-roll the cancel broadcast.
- **FR-9.** Heal + Dispel MUST NOT award experience (unlike Cleric Heal's undead-XP
  path); this is a GM utility, not a combat heal.
- **FR-10.** Per-recipient failures (a failed HP restore, a failed dispel for one player)
  MUST be logged but MUST NOT abort the cast for the other recipients.

### 4.4 Hide behavior

- **FR-11.** Hide is a **toggle**. Casting `SuperGmHideId` while not hidden turns hide
  **on**; casting it while already hidden turns hide **off**. The handler MUST determine
  current hide state to decide the direction.
- **FR-12.** When hide turns **on**:
  - The caster MUST receive the hide temporary stat so the client renders the local
    hidden state. This requires mapping `SuperGmHideId` to a hide statup
    (`DARK_SIGHT`, or a dedicated `SNEAK` stat — see OQ-1) in the WZ reader and/or the
    handler.
  - The caster MUST be **despawned** from every other player currently in the map
    (`CharacterDespawn` to other sessions).
  - Subsequent foreign spawns MUST be **suppressed**: when another player enters the
    map, or when the hidden caster changes maps, the caster MUST NOT be spawned into
    other players' views. This modifies the currently-unconditional spawn paths in
    `kafka/consumer/map/consumer.go` (`enterMap` / `spawnCharacterForSession`) to
    consult hide state.
- **FR-13.** When hide turns **off**:
  - The hide temporary stat MUST be cancelled on the caster.
  - The caster MUST be **spawned** into every other player's view in the current map
    (`CharacterSpawn` to other sessions), restoring normal visibility.
- **FR-14.** Hide state MUST **persist across map changes** while on: a hidden SuperGM
  who warps to a new map stays hidden (not spawned to players there) until they toggle
  hide off. This implies hide state is stored somewhere durable enough to be read by the
  spawn broadcast path on map entry (see OQ-2 for the storage decision — buff/temp-stat
  vs. character flag).
- **FR-14.1.** "Untargetable" is satisfied by the caster not being spawned to other
  players; no additional mob-aggro suppression is required (aggro is damage-driven and a
  hidden, unseen caster cannot be hit). If any spawn path *does* leak the caster, that is
  a bug against FR-12, not a separate aggro feature.

### 4.5 Effect-data projection (supporting)

- **FR-15.** The channel-side `data/skill/effect` model MUST expose the fields these
  handlers need that it does not currently: at minimum `MP()` and the MP/HP **recovery**
  ratio(s) for the Heal component, plus whatever the hide statup needs. The WZ reader in
  `atlas-data` already parses these raw values; this is a projection/accessor gap on the
  channel side, not a new WZ parse.
- **FR-16.** If `SuperGmHealDispelId` remaining in `isCategory1` (which forces
  `buff=false`) prevents the needed recovery fields from being surfaced, the decode path
  MUST be adjusted so the handler can read HP/MP recovery. The handler-driven model
  (explicit per-skill handler) is authoritative for behavior; the category flag only
  governs how the effect is decoded.

### 4.6 Skill-use broadcast

- **FR-17.** Both handlers MUST broadcast the skill-use animation consistently with the
  existing handlers: `AnnounceSkillUse` to the caster's session and
  `AnnounceForeignSkillUse` to other sessions in the map — **except** that a hidden
  caster's Hide/Heal casts MUST NOT reveal the caster's position to players who cannot
  see them (a foreign skill-use animation for an invisible caster would leak presence).
  When the caster is hidden, foreign skill-use broadcast MUST be suppressed. (See OQ-3.)

## 5. API Surface

No new REST endpoints.

New/modified Kafka command surface (channel → atlas-buffs), all on existing topics:

- **`COMMAND_TOPIC_CHARACTER_BUFF` / `CANCEL_BY_TYPES`** — new channel-side producer +
  `character/buff` processor method to purge the disease-debuff set for a character.
  `atlas-buffs` already consumes this command and emits EXPIRED status events; no
  atlas-buffs change is expected for dispel.
- **HP/MP restore** — reuse the existing `ChangeHP` command; add a `ChangeMP` command if
  one does not already exist (verify during design) so the Heal component can restore MP.

Hide visibility is delivered entirely via existing clientbound writers
(`CharacterSpawnWriter` / `CharacterSpawnForeignWriter`, `CharacterDespawnWriter`) and
the buff give/cancel writers; no new packet types are anticipated. Any new packet must
be verified against source per repo packet-audit rules before use.

## 6. Data Model

No new database entities or migrations.

Hide state is runtime-only. The storage decision (OQ-2) is one of:

- **Option A — temporary stat / buff**: represent hide as a `DARK_SIGHT`/`SNEAK` buff in
  `atlas-buffs`. Naturally persists across map changes (buffs are per-character, not
  per-map) and is already carried into `CharacterSpawnBody`'s temporary-stat block. The
  spawn suppression path reads the character's active buffs (already loaded in
  `spawnCharacterForSession`) to decide visibility.
- **Option B — character hide flag**: a dedicated boolean on the character/session state
  in `atlas-character` or the channel character projection, read by the spawn path.

Option A is preferred unless design finds the buff's foreign-visibility semantics
conflict with "suppress the spawn entirely" (a hide buff that itself gets broadcast to
foreigners would be self-defeating). Design MUST resolve this.

## 7. Service Impact

- **`atlas-channel`** (primary):
  - New `skill/handler/healdispel` and `skill/handler/hide` subpackages + registration.
  - New map-wide (non-party) recipient selector in `skill/handler/recipients.go`.
  - New `character/buff` producer + processor method for `CancelByTypes` (dispel).
  - Possibly a new `ChangeMP` producer/command (if absent) for MP restore.
  - New accessors on `data/skill/effect` model (MP, recovery ratios).
  - Modified spawn/despawn broadcast in `kafka/consumer/map/consumer.go` to consult hide
    state (`enterMap`, `spawnCharacterForSession`, and the map-entry self-spawn loop).
  - Modified foreign skill-use broadcast to suppress hidden casters.
- **`atlas-data`**:
  - Add a `SuperGmHideId` statup branch (hide stat) in `skill/reader.go` `getEffect`,
    and surface HP/MP recovery for `SuperGmHealDispelId` (adjust `isCategory1` handling
    per FR-16).
- **`atlas-buffs`**: expected **no change** — it already supports `CANCEL_BY_TYPES` and
  disease-set semantics. Verify during design; if a hide-specific stat needs immunity or
  broadcast rules, that surfaces here.
- **`libs/atlas-constants`**: IDs and `Skill`/`Job` vars already exist
  (`SuperGmHideId`, `SuperGmHealDispelId`, `SuperGmId`). If a new `SNEAK` temporary-stat
  producer/consumer is chosen over `DARK_SIGHT`, the constant already exists
  (`TemporaryStatTypeSneak`) but is currently unused.

## 8. Non-Functional Requirements

- **Multi-tenancy**: all processors resolve tenant from context; commands carry tenant
  headers per the existing consumer pattern. No cross-tenant leakage of hide state or
  heal recipients.
- **Correctness of visibility**: hide suppression MUST be race-safe against concurrent
  map entry — a player entering the map while a SuperGM is hidden MUST NOT momentarily
  see the caster. The suppression check MUST live in the same broadcast path that emits
  the spawn, not a best-effort follow-up despawn.
- **Observability**: cast, toggle direction, recipient count, and per-recipient failures
  logged at appropriate levels (info/debug for success, warn/error for failures and
  rejected non-SuperGM casts).
- **Client stability**: hiding/revealing and map transitions while hidden MUST NOT crash
  or desync other players' clients (no dangling avatar, no double-spawn). Verify with
  byte-level correctness for any spawn/despawn packet touched.
- **No regressions**: existing Cleric Heal, Mystic Door, mounts, and normal
  spawn/despawn for non-hidden characters MUST be unaffected.

## 9. Open Questions

- **OQ-1 (hide stat choice)**: Use the existing `DARK_SIGHT` temporary stat for the GM
  hide buff, or the currently-unused `SNEAK` (`TemporaryStatTypeSneak`) stat? `DARK_SIGHT`
  is what the WZ reader already emits for Rogue Dark Sight; `SNEAK` is semantically closer
  to "invisible to others" but has no existing producer/consumer. Resolve in design,
  verifying against the v83 client's handling of each stat.
- **OQ-2 (hide-state storage)**: buff/temp-stat (Option A) vs. character flag (Option B)
  — see §6. Preference is A; design must confirm the foreign-broadcast semantics work.
- **OQ-3 (hidden-cast broadcast)**: Exact behavior of skill-use animation when the caster
  is hidden — fully suppress the foreign animation (chosen default, FR-17), or the client
  handles a hidden caster's animation gracefully? Verify against source.
- **OQ-4 (self-Heal while hidden)**: When a hidden SuperGM casts Heal+Dispel, other
  players still receive HP/MP/dispel benefits — confirm this does not require spawning or
  otherwise revealing the hidden caster to those recipients.
- **OQ-5 (MP restore command)**: Does a `ChangeMP` command/producer already exist in the
  channel character package, or must one be added alongside the existing `ChangeHP`?

## 10. Acceptance Criteria

- [ ] Casting `SuperGmHealDispelId` as a SuperGM restores HP and MP (from WZ recovery
      values, clamped to effective max) for **every** player in the map, party or not.
- [ ] The same cast cancels all disease debuffs (`STUN, POISON, SEAL, DARKNESS, WEAKEN,
      CURSE, SEDUCE, CONFUSE, UNDEAD, SLOW, STOP_PORTION`) on every player in the map, via
      the new channel-side `CancelByTypes` producer.
- [ ] Casting `SuperGmHideId` as a SuperGM while visible hides the caster: other players
      in the map receive a despawn, and players entering the map afterward do not see the
      caster.
- [ ] Casting `SuperGmHideId` again reveals the caster: other players in the map receive
      a spawn and see the caster normally.
- [ ] A hidden SuperGM who changes maps remains hidden in the new map until toggling off.
- [ ] A non-SuperGM character that somehow has either skill is rejected on cast (no
      effect, warn-logged), verified by test.
- [ ] Heal + Dispel awards no experience.
- [ ] Skill-use animation broadcasts correctly for a visible caster and is suppressed for
      a hidden caster (no position leak).
- [ ] Existing Cleric Heal, Mystic Door, mount buffs, and normal spawn/despawn for
      non-hidden characters are unaffected (regression tests pass).
- [ ] `go test -race ./...` and `go vet ./...` clean in every changed module;
      `go build ./...` clean; `docker buildx bake` succeeds for every service whose
      `go.mod` was touched; `tools/redis-key-guard.sh` clean.
- [ ] Any spawn/despawn/buff packet touched is byte-verified against source per repo
      packet-audit rules.

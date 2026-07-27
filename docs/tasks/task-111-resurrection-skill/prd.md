# Resurrection Skill — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-06-24
---

## 1. Overview

Resurrection is the cleric-line revive skill. The Bishop player skill
(`2321006`) and its two GM tool variants (`9001005` GM, `9101005` SuperGM)
revive dead characters within an area of effect, restoring them to full HP
**in place** — at the exact position where they died, with no map reload, no
town return, and no experience penalty. A "ray of holy light" effect plays on
each revived character.

Today atlas-channel implements no resurrection: a dead character (`Hp() == 0`)
can only return to play by entering a portal, which routes through the
town-respawn flow (`socket/handler/map_change.go:52` → `respawn.Respawn`),
warping them to the map's return map at 50 HP with exp loss. There is no
in-place, full-HP party revive. This task adds it.

The feature is built on the chase-warp primitive landed by task-093 (Mystic
Door, PR #769): `portal.WarpToPosition(f, characterId, mapId, x, y)` warps a
character to an exact coordinate via the v83 `SET_FIELD` "chase" mechanism.
Because a dead character is in the client's death stance, that same `SET_FIELD`
fires the client's native revive routine (`CWvsContext::OnRevive` /
`CUser::OnRevive`, verified in `CStage::OnSetField` @0x776020), closing the
death prompt and standing the avatar up at the chase coordinates. This means
resurrection is, mechanically, a respawn variant that warps the dead member to
their own death position at full HP — reusing proven, multi-version warp
plumbing rather than introducing a new packet.

## 2. Goals

Primary goals:

- Implement an active-skill handler for the three resurrection skill IDs
  (`2321006`, `9001005`, `9101005`) in atlas-channel, registered the same way
  the Heal handler is (`skill/handler/heal`).
- Revive every eligible **dead** target within the skill's WZ range to **full
  HP**, in place at each target's death coordinates, with the holy-light
  skill-use effect broadcast to self and to other players in the map.
- Scope the revive recipients correctly per variant:
  - Bishop (`2321006`): dead **party** members in range.
  - GM / SuperGM (`9001005` / `9101005`): **all** dead players in range,
    party-agnostic (the GM-tool behavior).
- Consume MP and apply cooldown via the existing generic `UseSkill` path,
  driven by per-version WZ effect data.
- Work across all supported tenant versions (GMS v83/v84/v87/v95, JMS v185),
  inheriting task-093's version-aware chase-warp encoding and the existing
  version-aware skill config / atlas-data effect lookup.

Non-goals:

- Post-revive invincibility. The v83 WZ data for `2321006`
  (`Cosmic/wz/Skill.wz/232.img.xml`) contains **no** invincibility property —
  only `hs`, `lt`/`rb`, `cooltime`, `mpCon`. The "~3s invincibility" is a
  later-version (post-remaster) property. Faithful v83 behavior is a plain
  full-HP revive, so no invincibility window is implemented and the
  damage-application path is left unchanged.
- Experience handling or death-penalty interaction (resurrection neither
  restores lost exp nor charges any).
- Implementing any of the damage-mitigation TODOs in
  `socket/handler/character_damage.go` (PowerGuard, MagicGuard, etc.).
- Reviving non-player entities (pets, summons) or self-revive.

## 3. User Stories

- As a **Bishop**, I want to cast Resurrection on my fallen party so that dead
  members in range stand back up at full HP exactly where they died, instead of
  having to respawn in town and run back.
- As a **dead party member**, I want a Bishop's Resurrection to revive me in
  place so that I keep my position in a fight and lose no progress.
- As a **GM**, I want the GM/SuperGM Resurrection to revive every dead player in
  range regardless of party, so that I can recover a whole area after a wipe.
- As **another player in the map**, I want to see the holy-light effect and the
  revived players stand up, so the revive reads correctly on my screen.

## 4. Functional Requirements

### 4.1 Skill dispatch and registration

- FR-1. A new handler package `skill/handler/resurrection/` registers a handler
  (via `channelhandler.Register`) for all three skill IDs: `BishopResurrectionId`
  (`2321006`), `GmResurrectionId` (`9001005`), and `SuperGmResurrectionId`
  (`9101005`). Constants already exist in
  `libs/atlas-constants/skill/constants.go`.
- FR-2. The package is blank-imported in
  `skill/handler/registrations/registrations.go` so the `init()` registrations
  run, matching the Heal pattern.
- FR-3. The handler conforms to the existing `handler.Handler` signature
  (`func(l) func(ctx) func(wp, f field.Model, characterId uint32,
  info packetmodel.SkillUsageInfo, e effect.Model) error`). It is invoked from
  the generic `UseSkill` dispatch (`registry.Lookup` →
  `character_skill_use.go`), which already validates skill ownership/level,
  loads the WZ effect, and consumes MP / applies cooldown before the handler
  runs.

### 4.2 Recipient selection

- FR-4. Resurrection selects only **dead** characters (`Hp() == 0`) within the
  effect's `LT`/`RB` rectangle around the caster. This is the inverse of the
  Heal selector's living-only filter (`skill/handler/recipients.go:162` skips
  `Hp() == 0`).
- FR-5. For the Bishop variant, recipients are restricted to the caster's
  **party** members in range (same channel + map, live session in field), mirror
  of `SelectInRangePartyMembers` with the dead/alive filter inverted.
- FR-6. For the GM and SuperGM variants, recipients are **all** dead players in
  range in the caster's map, regardless of party membership (Cosmic
  `isGmBuff()` semantics, `StatEffect.java:1150,1156`).
- FR-7. The caster is never a recipient (the caster is alive when casting).
- FR-8. Each recipient carries the coordinates used as the revive position
  (see FR-11). The selector captures each recipient's `X`/`Y` the same way the
  Heal `PartyRecipient` does.
- FR-9. If no eligible dead target is found, the cast is a no-op beyond the
  normal MP/cooldown consumption and the self skill-use effect; the handler
  returns without error.

### 4.3 Revive application (per recipient)

- FR-10. Restore the recipient's HP to full **before** issuing the warp, because
  the `SET_FIELD` packet built downstream reads the character's current HP
  (`kafka/consumer/character/consumer.go:253` passes `c.Hp()`). HP is set via
  the existing character HP command path (`SetHP`, absolute, to the recipient's
  max HP). Ordering mirrors the respawn saga (`respawn/processor.go`:
  `set_hp` step precedes the warp step).
- FR-11. Warp the recipient to their **own death position** on the **same map**
  via `portal.NewProcessor(l, ctx).WarpToPosition(field, recipientId,
  currentMapId, deathX, deathY)`. This emits the chase-warp `SET_FIELD`, which
  (because the target is in the death stance) triggers the client's `OnRevive`,
  closing the death prompt and standing the avatar up at `(deathX, deathY)`.
- FR-12. Broadcast the resurrection skill-use effect: the holy-light effect to
  the caster's own session and the foreign skill-use effect to other sessions in
  the map, reusing the existing `AnnounceSkillUse` / `AnnounceForeignSkillUse`
  helpers (`socket/handler/effects.go`) as the Heal handler does. The per-target
  effect (showOwnBuffEffect equivalent) on each revived player is a fidelity
  detail to confirm during implementation against the Heal precedent.
- FR-13. No experience is granted to the caster for resurrecting (unlike Heal's
  undead-XP path); no exp is restored to or deducted from the revived target.

### 4.4 MP, cooldown, and version handling

- FR-14. MP cost (`mpCon`: Bishop 85→45 across levels) and cooldown
  (`cooltime`: Bishop 3420→1800s) are applied by the generic `UseSkill` path
  from the WZ effect — the handler does not re-implement them.
- FR-15. The skill's range (`lt(-400,-350)` / `rb(400,250)` for all three IDs in
  v83) is taken from the WZ effect (`effect.Model.LT()/RB()`), not hard-coded.
- FR-16. The feature must function on every supported tenant version. The
  chase-warp `SET_FIELD` encoding is already version-branched and IDA-cited for
  GMS v83/v84/v87/v95 and JMS v185 in
  `libs/atlas-packet/field/clientbound/warp_to_map.go`. Skill opcodes/effects
  are resolved per version through the existing config / atlas-data lookups.

## 5. API Surface

No new REST endpoints or Kafka topics. The feature is internal to atlas-channel
and reuses existing cross-service messaging:

- **Reused (no change):** `portal.WarpToPosition` →
  `portals.WarpCommand{ ..., UseTargetPosition: true, TargetX, TargetY }` on
  `EnvPortalCommandTopic`; atlas-portals → atlas-maps `CHANGE_MAP`; atlas-maps
  `MAP_CHANGED` status event (carries `UseTargetPosition/TargetX/TargetY`) →
  atlas-channel `warpCharacter` consumer → `SetFieldWriter`
  (`WarpToPositionBody`).
- **Reused (no change):** the character HP command used for `SetHP`
  (full-HP restore).
- **Possibly modified (verification-gated, see §9 / Open Question OQ-2):** a
  `WarpToMap` chase variant that sets the packet's `revive` byte to `1` instead
  of the hard-coded `0` (`libs/atlas-packet/field/clientbound/warp_to_map.go`).
  Only introduced if live testing shows the death-stance gate alone does not
  reliably fire `OnRevive`.

## 6. Data Model

No new persistent entities, tables, or migrations. Resurrection operates on
transient in-flight state (skill cast → HP command → warp command → effect
broadcast). All persistent state (character HP, location) is owned by existing
services (atlas-character, atlas-maps) and mutated through their existing
commands. No `tenant_id`-scoped storage is added.

## 7. Service Impact

- **atlas-channel** (primary): new `skill/handler/resurrection/` package
  (handler + dead-target selector); registration wiring in
  `skill/handler/registrations/registrations.go`. Orchestrates per-recipient
  `SetHP` (full) → `WarpToPosition` (death coords) → effect broadcast. No
  changes to `character_damage.go` (invincibility is out of scope).
- **libs/atlas-packet** (conditional): only if OQ-2 resolves toward an explicit
  `revive` byte — a chase+revive `WarpToMap` variant. Otherwise untouched.
- **atlas-portals / atlas-maps / atlas-character** (no change): consumed via
  existing commands/events (chase warp, HP command). Listed to confirm the
  data flow, not because they change.

## 8. Non-Functional Requirements

- **Multi-tenancy:** all processor calls thread `ctx` (tenant from context) as
  existing handlers do; recipient selection and warps are tenant-scoped via the
  caster's field/channel. No cross-tenant access.
- **Concurrency:** multiple recipients are revived independently; follow the
  Heal handler's iteration/ordering. Each recipient's `SetHP`-before-warp
  ordering must hold per recipient (the warp reads HP).
- **Observability:** debug-log the cast, the resolved recipient set (count +
  ids), and each revive (id + death coords), consistent with existing skill
  handlers. Failures to revive a single recipient are logged and skipped without
  aborting the whole cast.
- **Performance:** range/party selection is bounded by party size (Bishop) or
  in-map player count (GM); no heavy queries beyond the per-recipient character
  loads the Heal selector already performs.
- **Faithfulness:** all gameplay values (range, MP, cooldown, full-HP restore,
  no invincibility) are taken from / validated against v83 WZ data, not general
  MapleStory knowledge.

## 9. Open Questions

These are empirical, settled on the running environment during implementation
(they are verification gates, not blockers to planning):

- **OQ-1 — Dead-player chase warp fires `OnRevive`.** Static analysis
  (`CStage::OnSetField` @0x776020) shows `OnRevive` runs when the target is in
  the death stance, which a dead recipient is. Confirm live that a dead player
  warped via `WarpToPosition` actually closes the death prompt and stands up in
  place.
- **OQ-2 — `revive` byte vs. death-stance gate.** The `WarpToMap` encoder
  hard-codes the `revive` byte to `0`, so revive currently depends on the
  death-stance check. If OQ-1 is unreliable, add a chase+revive variant that
  sets the byte to `1`. Decide based on the live result.
- **OQ-3 — Same-map warp behavior.** `warp.ChangeMap` does not reject a
  same-map destination, but the same-map transition path
  (`TransitionMapAndEmit(dest == oldField)`) should be checked live for any
  despawn/respawn flicker for observers.
- **OQ-4 — Death coordinate source.** Confirm the recipient's tracked `X`/`Y`
  (from the character/location model the selector reads) reflects the actual
  death position closely enough for "in place." If not, source the death
  coordinates from atlas-maps location state.
- **OQ-5 — Per-version revive parity.** OQ-1 is verified for v83; confirm the
  same dead-player chase-warp revive behavior on v87/v95/JMS, and that the
  GM/SuperGM skill IDs and their WZ range exist per version.

## 10. Acceptance Criteria

- [ ] A `skill/handler/resurrection/` package registers handlers for `2321006`,
      `9001005`, and `9101005`, blank-imported in `registrations.go`.
- [ ] Casting Bishop Resurrection (`2321006`) revives every **dead party
      member** within the WZ range to full HP, in place at their death
      coordinates; living members and out-of-range members are unaffected.
- [ ] Casting GM/SuperGM Resurrection (`9001005`/`9101005`) revives **all** dead
      players in range to full HP, in place, regardless of party.
- [ ] Each revived target's own client closes the death prompt and stands up at
      the death position (OQ-1 confirmed live on v83).
- [ ] The holy-light skill-use effect is broadcast to the caster and to other
      players in the map.
- [ ] Each revived target's HP is restored to full **before** the warp packet is
      built (verified via the warp reading full HP).
- [ ] MP is consumed and cooldown applied per the WZ effect; range comes from
      the WZ effect, not hard-coded.
- [ ] No invincibility is applied and `character_damage.go` is unchanged.
- [ ] Casting with no eligible dead target is a clean no-op (MP/cooldown +
      self effect only), no error.
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...` clean for
      atlas-channel (and any changed module); `docker buildx bake` for any
      service whose `go.mod` was touched; `tools/redis-key-guard.sh` clean.
- [ ] The feature is exercised/verified on at least v83; per-version parity
      (OQ-5) is recorded for the other supported versions.

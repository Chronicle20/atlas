# Meso Explosion — Exploded-Meso Destruction + Damage — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-07-10
---

## 1. Overview

Chief Bandit's Meso Explosion (skill 4211006) detonates meso drops lying on the ground, dealing damage to nearby monsters proportional to the exploded meso piles. In Atlas today the skill is server-side inert: the melee attack broadcast already flags the skill (`services/atlas-channel/atlas.com/channel/socket/writer/character_attack_melee.go:19` sets `isMesoExplosion`, and the clientbound writer in `libs/atlas-packet/character/clientbound/attack.go` encodes the meso-explosion body variant), but the serverbound decode never parses the meso-explosion-specific packet layout, the listed meso drops are never destroyed, and the damage is never applied. The gap is marked at `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go:407` (`// TODO destroy Chief Bandit exploded mesos`).

The meso-explosion attack packet differs from a standard melee attack in two ways (behavioral reference: Cosmic `AbstractDealDamageHandler.java:218, 598-626, 950-960`): the per-monster damage-line count is variable (encoded per monster rather than taken from the shared hits nibble), and the packet carries a trailing list of exploded meso drop object ids. Because `AttackInfo.Decode` (`libs/atlas-packet/model/attack_info.go:181`) knows nothing of either difference, a real Meso Explosion attack currently mis-frames the packet tail.

This task makes the skill functional across all supported tenant versions: decode the meso-explosion layout correctly, validate and destroy the listed meso drops (with the existing explode animation route), and apply the client-reported damage through the existing attack pipeline.

## 2. Goals

Primary goals:
- Correctly decode the serverbound meso-explosion attack variant (variable per-monster hit counts + exploded-meso drop id list) for all supported tenant versions.
- Destroy each validated exploded meso drop immediately via the existing drop `CONSUME` route, which already broadcasts `DropDestroyWriter` with `DropDestroyTypeExplode`.
- Apply the client-reported damage lines through the existing `processDamageInfoEntry` pipeline (client-trusted, consistent with all other attack types).
- Reject the whole attack (no damage, no drop destruction) when any listed drop fails validation.
- Byte-fixture the decode per supported version with `packet-audit` verification markers.

Non-goals:
- Pick Pocket (4211003) implementation — separate TODO (`character_attack_common.go:408`). Meso Explosion works against any meso drop regardless of origin, so it does not depend on Pick Pocket.
- Server-side damage recomputation from destroyed meso amounts or skill data (owner decision: client-trusted).
- Staggered/delayed drop removal to mirror Cosmic's ~100 ms-per-drop cadence (owner decision: destroy immediately; the client animation covers presentation).
- Crediting any portion of exploded mesos to the attacker — exploded mesos are purely destroyed.
- Any other attack-effect TODO in the `character_attack_common.go` block (cooldowns, combo orbs, drains, charges, etc.).

## 3. User Stories

- As a Chief Bandit player, I want Meso Explosion to destroy the meso piles I detonate and damage the monsters near them, so the skill actually functions instead of only animating.
- As a player on the same map, I want to see the exploded meso drops disappear with the explode animation and see the attacker's meso-explosion attack animation, so the game state stays visually consistent for everyone.
- As a server operator, I want the server to reject meso-explosion attacks that reference drops that don't exist, aren't mesos, or aren't in the attacker's field, so the skill cannot be used to delete arbitrary drops or desync map state.

## 4. Functional Requirements

### 4.1 Serverbound decode (libs/atlas-packet)

- FR-1: `AttackInfo.Decode` MUST detect the meso-explosion variant (skill id 4211006, `skill.ChiefBanditMesoExplosionId` in `libs/atlas-constants/skill`) and parse the variant layout:
  - FR-1a: Per-monster damage-line count read per monster entry (variable), instead of the shared `hits` nibble from `numAttackedAndDamageMask`.
  - FR-1b: Trailing exploded-meso list: a count followed by per-entry meso drop object ids.
- FR-2: The exact byte order for each supported version (gms_v83, gms_v84, gms_v87, gms_v95, jms_v185) MUST be derived from the corresponding client via IDA during design, following `docs/packets/audits/VERIFYING_A_PACKET.md`. Cosmic is a behavioral reference only; no read order may be assumed from Cosmic or from memory. gms_v92 has no IDB: it follows whatever version-family branch the verified neighbors dictate and is documented as unverified.
- FR-3: Decoded meso drop ids MUST be exposed on `AttackInfo` via an accessor (e.g. `ExplodedMesoDrops() []uint32`), empty for non-meso-explosion attacks.
- FR-4: Decode of all existing attack variants (melee/ranged/magic/energy, all versions) MUST remain byte-identical — covered by existing fixtures continuing to pass.

### 4.2 Validation (atlas-channel)

- FR-5: On receiving a meso-explosion attack, atlas-channel MUST validate every listed drop id against the drops in the attacker's field (existing drop REST client, `services/atlas-channel/atlas.com/channel/drop/`):
  - The drop exists in the attacker's field (world/channel/map/instance).
  - The drop is a meso drop (`Meso() > 0` / meso drop type — exact predicate confirmed at design time from the drop model).
  - The listed count does not exceed the skill's maximum detonatable drops for the character's skill level (from skill data via atlas-data; exact property confirmed at design time from WZ-derived data, not from memory).
- FR-6: If ANY listed entry fails validation, the ENTIRE attack MUST be skipped: no damage applied, no drops destroyed, no attack broadcast; log a warning identifying the character, skill, and failing drop id.
- FR-7: Duplicate drop ids within one attack packet count as a validation failure (each id must be unique).

### 4.3 Drop destruction (atlas-channel → atlas-drops)

- FR-8: For a fully validated attack, atlas-channel MUST emit one drop `CONSUME` command (`CommandTypeConsume`, `EnvCommandTopic`) per listed drop, immediately (no staggering), carrying the attacker's field coordinates in the envelope.
- FR-9: No changes to atlas-drops semantics are required: `Consume` already removes the drop from the registry and emits `CONSUMED`, and atlas-channel's existing drop consumer already announces `DropDestroyWriter` with `DropDestroyTypeExplode` to the field (`services/atlas-channel/atlas.com/channel/kafka/consumer/drop/consumer.go:144`). If design finds `CONSUME` needs a meso-only guard server-side, that guard goes in atlas-drops' `Consume` path without breaking the reactor caller (`services/atlas-reactors`).

### 4.4 Damage application (atlas-channel)

- FR-10: Damage lines decoded from the meso-explosion packet MUST flow through the existing `processDamageInfoEntry` pipeline unchanged (client-trusted), after validation passes.
- FR-11: The existing remote broadcast (melee writer with `isMesoExplosion`) MUST carry the variable per-monster damage counts correctly — the clientbound encode already branches on `isMesoExplosion`; verify it round-trips the decoded variable-hit data.
- FR-12: The `// TODO destroy Chief Bandit exploded mesos` comment at `character_attack_common.go:407` is removed by this task.

## 5. API Surface

No new or modified REST endpoints.

Kafka (all existing message shapes, no schema changes):
- atlas-channel produces `CommandTypeConsume` on `COMMAND_TOPIC_DROP` (existing envelope: transactionId, worldId, channelId, mapId, instance, body.dropId) — new producer call site in atlas-channel.
- atlas-drops produces `StatusEventTypeConsumed` on `EVENT_TOPIC_DROP_STATUS` (existing).

## 6. Data Model

No new entities, fields, or migrations. Drop state remains in the atlas-drops in-memory registry; skill/skill-effect data comes from atlas-data.

## 7. Service Impact

- `libs/atlas-packet` — `AttackInfo` decode extension (meso-explosion variant per version), accessor for exploded drop ids, byte fixtures per supported version with `packet-audit:verify` markers; clientbound attack encode round-trip check for variable hits.
- `services/atlas-channel` — meso-explosion branch in the attack handler: validate listed drops against the field's drops (REST), emit `CONSUME` per drop, reject-whole-attack on failure; remove the TODO.
- `services/atlas-drops` — expected no change (FR-9); at most a meso-only consume guard if design requires it.
- Packet audit artifacts — serverbound attack cells for the affected versions re-verified per `VERIFYING_A_PACKET.md` (evidence records + matrix regeneration).

## 8. Non-Functional Requirements

- Multi-tenancy: decode branches keyed off `tenant.MustFromContext(ctx)` region/major-version, consistent with the existing `attack_info.go` branching; no hard-coded version assumptions outside those branches.
- Grounding: every version-specific read order is IDA-verified; anything unverifiable (gms_v92) is explicitly documented as unverified rather than guessed.
- Performance: validation adds one drops-in-field REST fetch per meso-explosion attack; no per-drop REST calls.
- Observability: warn-level log on attack rejection with character id, skill id, and offending drop id; debug-level log of destroyed drop count on success.
- Safety: a malicious client cannot destroy non-meso drops, drops in other fields/instances, or more drops than the skill level allows, and cannot double-list a drop.

## 9. Open Questions

- Exact per-version byte layout of the meso-explosion variant (variable-hit encoding and meso-list position/width) — resolved via IDA during design (FR-2).
- Exact skill-data property bounding the number of detonatable drops per skill level — resolved from atlas-data/WZ during design (FR-5).
- Whether the meso-drop predicate on the channel drop model is `Meso() > 0`, a drop `Type()` value, or both — resolved from the drop model during design (FR-5).

## 10. Acceptance Criteria

- [ ] `AttackInfo` decodes a meso-explosion attack correctly for gms_v83, gms_v84, gms_v87, gms_v95, and jms_v185, proven by byte fixtures derived from IDA-verified read orders; all pre-existing attack fixtures still pass.
- [ ] In-game (or integration-level) flow: a meso-explosion attack destroys exactly the listed, validated meso drops with the explode animation visible to all sessions in the field, and the monsters take the client-reported damage.
- [ ] An attack listing a non-existent drop, a non-meso drop, a drop from another field/instance, a duplicate id, or more drops than the skill level allows is skipped entirely, with a warning log and no side effects.
- [ ] Exploded mesos yield no meso gain to any character.
- [ ] The TODO at `character_attack_common.go:407` is gone.
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...` clean in every changed module; `tools/redis-key-guard.sh` clean; `docker buildx bake` green for every touched service (`atlas-channel`, plus `atlas-drops` if touched).

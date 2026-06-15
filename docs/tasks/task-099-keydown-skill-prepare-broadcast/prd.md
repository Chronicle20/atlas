# Keydown Skill Prepare/Cancel Broadcast — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-06-15
---

## 1. Overview

Keydown (hold-and-cast) skills — Bowmaster Hurricane, Monster Magnet, Rapid Fire, Piercing Arrow, BigBang, Evan Breaths, etc. — display a looping "cast aura" animation around the caster while the key is held. Today in Atlas, when one player casts such a skill, **other players in the same map see the fired projectiles/attack but NOT the cast aura**. The aura never starts (and, were it to start, never stops) for observers.

Root cause (IDA-verified against the GMS v95 client, cross-checked against the Cosmic reference server): the looping aura on an observer is driven by a dedicated **skill-prepare** packet, not by the attack packet. On keydown the casting client sends a serverbound *prepare* packet (`CUserLocal::DoActiveSkill_Prepare`); a correct server rebroadcasts a clientbound *remote-prepare* (`CUserRemote::OnSkillPrepare`) to other players in the map, which starts and loops the aura. On keyup the client sends a serverbound *cancel* packet (`CUserLocal::SendSkillCancelRequest`); the server rebroadcasts a clientbound *remote-cancel* (`CUserRemote::OnSkillCancel`), which plays the keydown-end animation and clears the aura. atlas-channel currently implements **none** of these four packets (all marked ❌ in `docs/packets/audits/STATUS.md`). The attack/projectile broadcast already works, which is why observers see arrows but no aura.

This task implements the four packets in atlas-channel as a **map-local visual broadcast** — the same pattern as the existing attack broadcast (`processAttack` → `ForOtherSessionsInMap`). It introduces no persistence, no Kafka events, and no cross-service coordination. It explicitly does **not** touch the attack packet's `tKeyDown` field, which IDA confirmed is correct as-is (the v95 remote attack reads `tKeyDown` only for the BigBang trio + Evan magic-charge skills and never for shoot skills like Hurricane).

## 2. Goals

Primary goals:
- Observers in the same map see the looping cast aura **start** when a player begins a keydown skill, and **stop** when they release it (or the keydown otherwise ends).
- Cover **all** keydown skills as classified by `skill.IsKeyDownSkill` (Hurricane, WindArcher Hurricane, Monster Magnet ×3, Rapid Fire, Piercing Arrow, Poison Bomb, BigBang ×3, Evan Ice/Fire Breath, …).
- Support **all configured tenant versions** (GMS v83/v84/v87/v92/v95, JMS v185), with opcodes and packet read/write orders **verified against each version's IDB** — not inferred from the registry CSV or from Cosmic.
- The prepare/cancel packets are promoted from ❌ to verified in the coverage matrix, backed by byte-fixture tests, for every supported version.

Non-goals:
- No change to the attack/projectile packets or their `tKeyDown` handling (confirmed correct; broadening it would corrupt the remote packet).
- No buff, stat, damage, cooldown, or HP/MP-cost behavior (those flow through existing skill-use/attack paths).
- No persistence, Kafka event, or REST surface — purely an in-channel socket broadcast.
- No summon keydown / monster skill prepare.
- Not fixing any other unimplemented skill-effect modes (e.g. the broader EffectSkillUse conditional branches).

## 3. User Stories

- As a player watching another player in my map cast Hurricane, I want to see the looping cast aura around them (not just the arrows), so the game looks correct and I can read what they're doing.
- As a player, when the caster releases the key, I want the aura to disappear promptly, so I don't see a stuck animation.
- As a player on any supported game version (v83–v95, JMS185), I want this to work identically, since the mechanic is the same across versions.
- As an operator, I want the new packets wired into both the seed templates and live tenant configs, so existing tenants get the behavior without a full reprovision.

## 4. Functional Requirements

### 4.1 Inbound — serverbound prepare (keydown)
- FR-1.1 atlas-channel registers a handler for the serverbound skill-prepare op for each supported version (the op fired by `CUserLocal::DoActiveSkill_Prepare`).
- FR-1.2 The handler decodes the packet per the **IDB-verified read order**. Working model from v95 (to be confirmed per version): `skillId` (int32), `level` (int8), `action` (int16: bit15 = move-action flag, low 15 bits = action), `actionSpeed` (int8). The decoder must match each version's actual order; version-conditional fields are allowed (mirror existing attack-info decode).
- FR-1.3 The handler validates that the casting character **owns the skill** (and resolves its level) before broadcasting; an unowned/unknown skill is dropped (logged), not broadcast. Mirrors the ownership check in `processAttack`/`character_skill_use`.
- FR-1.4 The handler broadcasts only when the skill is a keydown skill per `skill.IsKeyDownSkill`. Non-keydown skill ids are dropped (logged at debug), never broadcast.

### 4.2 Outbound — clientbound remote-prepare
- FR-2.1 atlas-channel broadcasts a clientbound remote skill-prepare to **all other sessions in the caster's map** (exclude the caster), via the existing `_map.Processor.ForOtherSessionsInMap` pattern.
- FR-2.2 The clientbound packet is encoded per the **IDB-verified write order** for each version. Working model from v95: `characterId` (int32) + `skillId` (int32) + `level` (int8) + `action` (int16) + `actionSpeed` (int8).
- FR-2.3 Field values are taken from the decoded inbound packet (after ownership validation), not fabricated.

### 4.3 Inbound — serverbound cancel (keyup)
- FR-3.1 atlas-channel registers a handler for the serverbound skill-cancel op for each supported version. **The op must be identified via each version's IDB** (v95 uses a dedicated cancel-skill op `0x068`, distinct from prepare `0x069`; Cosmic instead overloads `CANCEL_BUFF` `0x5C` — do NOT assume Cosmic's choice; verify the real per-version opcode).
- FR-3.2 The handler decodes the cancel packet per the IDB-verified read order (v95: `skillId` int32).
- FR-3.3 The handler validates skill ownership and keydown-classification as in FR-1.3/1.4 before broadcasting.

### 4.4 Outbound — clientbound remote-cancel
- FR-4.1 atlas-channel broadcasts a clientbound remote skill-cancel to all other sessions in the map (exclude caster).
- FR-4.2 Encoded per the IDB-verified write order. Working model from v95 (`CUserRemote::OnSkillCancel`): `characterId` (int32) + `skillId` (int32).
- FR-4.3 **Pairing requirement:** any keydown skill for which a remote-prepare is broadcast MUST also be eligible for a remote-cancel broadcast. The observer does not auto-clear the aura (IDA-verified: remote movement and subsequent attacks do not clear `m_bKeyDown`; the time-based `Update` self-clear is unreliable for channeled skills), so a missing cancel leaves a **stuck aura**.

### 4.5 Termination edge cases
- FR-5.1 If the caster leaves the map, disconnects, or dies while a keydown is active, observers must not be left with a stuck aura. Define and implement the chosen mechanism (e.g., synthesize a remote-cancel broadcast on field-leave/cleanup). The exact trigger set is an open question (§9) to resolve in design.

### 4.6 Version coverage & verification
- FR-6.1 All four packets (serverbound prepare, clientbound prepare, serverbound cancel, clientbound cancel) are implemented for **every supported version**: GMS v83, v84, v87, v92, v95, JMS v185.
- FR-6.2 For each version, the opcode and the exact read/write order are **verified against that version's IDB** (ida-pro instances: v83 :13342, v84 :13337, v87 :13341, v95 :13340, JMS185 :13339; v92 has no IDB — see §9). The registry `fname`/opcode is a starting hint only.
- FR-6.3 Each packet × version cell is promoted in `docs/packets/audits/STATUS.md` from ❌ to verified, backed by a byte-fixture test following the `verify-packet` / `VERIFYING_A_PACKET` playbook.

### 4.7 Tenant config wiring
- FR-7.1 The new handler and writer opcodes are added to the channel handler/writer **seed templates** for each version.
- FR-7.2 The new opcodes are also patched into **live tenant configs** (handlers/writers don't hot-reload from projection — see the known "new opcodes missing from live tenant config" failure mode), so existing tenants get the behavior. Every new `socket.handlers` entry must carry a valid validator (LoggedInValidator) or it is silently dropped.

## 5. API Surface

No REST/JSON:API surface. The surface is **socket packets** within atlas-channel:

| Direction | Logical op | Client function | v95 opcode | v83 opcode | Notes |
|---|---|---|---|---|---|
| serverbound | skill prepare | `CUserLocal::DoActiveSkill_Prepare` | 0x069 | 0x5D (`SKILL_EFFECT` srv) | read: skillId, level, action(int16), actionSpeed |
| clientbound | remote skill prepare | `CUserRemote::OnSkillPrepare` | 0x0D7 | 0x0BE (`SKILL_EFFECT` cli) | write: charId + above |
| serverbound | skill cancel | `CUserLocal::SendSkillCancelRequest` | 0x068 | **TBD via IDB** | read: skillId |
| clientbound | remote skill cancel | `CUserRemote::OnSkillCancel` | nType 217 | 0x0BF (per Cosmic, **verify**) | write: charId + skillId |

All opcodes/read-orders above are the **working model**; the authoritative values come from per-version IDB verification (FR-6.2). Opcodes are resolved per tenant via the existing config-driven handler/writer registry (do not hardcode bytes in handlers; resolve via the tenant `socket.handlers`/writer tables, consistent with the codebase's config-derived opcode pattern).

## 6. Data Model

None. No new entities, columns, or migrations. The prepare/cancel state is transient and lives only in the client; the server is a stateless relay for these packets (with the field-leave cleanup of FR-5.1 being the only server-side state interaction, and that reuses existing session/field lifecycle).

## 7. Service Impact

- **atlas-channel** — new socket handlers (prepare, cancel), new broadcast writers (remote-prepare, remote-cancel), wired into the map-broadcast path. Primary site of change.
- **libs/atlas-packet** — new clientbound packet codecs (remote-prepare, remote-cancel) and serverbound decode for prepare/cancel, version-conditional per the IDB read orders. Byte-fixture tests live here / in the audit harness.
- **libs/atlas-constants/skill** — no change expected; `IsKeyDownSkill` already classifies the target skills (verify it covers every skill the per-version IDBs gate on; extend if a gap is found).
- **Tenant config** (seed templates + live configs) — new handler/writer opcode rows per version.
- **docs/packets** — registry rows confirmed; audit docs + STATUS matrix updated for the four ops × six versions.

No `go.mod` changes anticipated (no new modules), so the docker-bake gate does not apply unless a new lib is added; standard `go test -race`/`vet`/`build` for atlas-channel and libs/atlas-packet apply.

## 8. Non-Functional Requirements

- **Multi-tenancy / versioning:** opcodes and read/write orders are version-scoped and resolved from tenant config; one code path serves all versions via the existing config-driven registry. No version byte literals in handlers.
- **Performance:** broadcast is O(players in map) per keydown/keyup event — identical in shape to the existing attack broadcast, which already runs per attack. Negligible additional load.
- **Correctness over liberal echo:** unlike the Cosmic baseline (which reflects client-supplied fields unchecked), validate skill ownership/level server-side before broadcasting.
- **Observability:** log dropped packets (unowned skill, non-keydown skill, unknown op) at debug/info with character + skill id, consistent with existing handlers.
- **No regression:** the attack/projectile broadcast (arrows) must continue to work unchanged; the attack writer's `isKeydownSkill` narrow list is intentionally left as-is.

## 9. Open Questions

- **OQ-1 (per-version cancel opcode):** the serverbound keyup/cancel opcode must be read from each IDB. v95 = dedicated `0x068`; v83/84/87/jms185 unknown (Cosmic's `CANCEL_BUFF` overload is explicitly NOT to be trusted). Resolve during design/verification.
- **OQ-2 (v92 has no IDB):** there is no v92 client IDB available. How do we verify v92 opcodes/read-order — port from the nearest verified version (v95?) and banner as unverified, or defer v92 wiring until an IDB exists? (Cf. the parked v92 mount-food precedent.)
- **OQ-3 (read-order drift across versions):** v95 reads `action(int16)+actionSpeed`; confirm whether v83/84/87/jms differ (field widths, extra crc/dr blocks, v95+-only fields) as the attack-info decode does.
- **OQ-4 (termination triggers):** which server-side events must synthesize a remote-cancel — map change, disconnect, death, debuff/stun? Minimum viable = field-leave/disconnect; confirm death/stun in design.
- **OQ-5 (MovingShootAttackPrepare):** v95 has a separate `OnMovingShootAttackPrepare` (nType 216 / `MOVING_SHOOT_ATTACK_PREPARE` ~0x0D8) for moving-shoot keydown. Is it required for any in-scope skill (e.g. Rapid Fire / Hurricane while moving), or is the standard prepare sufficient? Determine in design; may expand the packet set.
- **OQ-6 (cancel for all vs. channeled subset):** we broadcast cancel for every prepared skill (FR-4.3). Confirm no client misbehaves when it receives a remote-cancel for a skill whose client-side aura already self-terminated.

## 10. Acceptance Criteria

- [ ] In a live env, with two characters in one map, a Bowmaster casting Hurricane shows the looping cast aura on the observer's client; the aura stops promptly on key release.
- [ ] The same holds for a representative sample across keydown families: a Monster Magnet (warrior), a BigBang (mage), Rapid Fire (corsair), Piercing Arrow (marksman).
- [ ] Serverbound prepare and cancel handlers are registered and decode correctly for v83/v84/v87/v92/v95/jms185 (per-version read order verified against IDB; v92 per OQ-2 resolution).
- [ ] Clientbound remote-prepare and remote-cancel are broadcast to all other map sessions excluding the caster, encoded per IDB-verified write order for each version.
- [ ] Broadcast is gated to `skill.IsKeyDownSkill` and to skills the caster actually owns; non-keydown/unowned skills are dropped (covered by tests).
- [ ] No stuck aura: caster key-release, map change, and disconnect all clear the observer's aura (per OQ-4 resolution).
- [ ] The attack/projectile broadcast is unchanged and still renders arrows (regression check); the attack writer `isKeydownSkill` list is untouched.
- [ ] `docs/packets/audits/STATUS.md` shows the four ops verified for all supported versions, each backed by a byte-fixture test with a `packet-audit:verify` marker.
- [ ] New handler/writer opcodes wired into seed templates AND patched into live tenant configs (every handler entry has a validator).
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...` clean in atlas-channel and libs/atlas-packet.

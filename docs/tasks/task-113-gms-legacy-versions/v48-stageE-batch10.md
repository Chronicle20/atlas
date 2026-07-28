# v48 Stage E — Batch 10 (character sub-family B: attacks + buffs + skills + movement)

Anchor v61 fast-path, IDB port 13337 (GMS_v48_1_DEVM.exe). Every fixture byte traced to a
v48 decompile line; op→body bindings body-verified from the COutPacket send-site (distrust
symbols). Branch `task-113-gms-legacy-versions`.

## Per-cell outcomes

| Cell | Dir | Op | v48 sender/handler | Outcome |
|---|---|---|---|---|
| CharacterAttackMeleeRequest (CLOSE_RANGE_ATTACK) | sb | 36 | sub_6A0528 @0x6a0528 | ✅ v48-gated |
| CharacterAttackRangedRequest (RANGED_ATTACK) | sb | 37 | sub_6A228C @0x6a228c | ✅ v48-gated |
| CharacterAttackMagicRequest (MAGIC_ATTACK) | sb | 38 | sub_6A3AC7 @0x6a3ac7 | ✅ v48-gated |
| Move (MOVE_PLAYER) | sb | 33 | sub_6E9923 @0x6e9923 (COutPacket 33 @0x6e9ac1) | ✅ = v61 |
| CharacterSkillChange (UPDATE_SKILLS) | cb | 30 | OnChangeSkillRecordResult @0x71a02c | ✅ = v61 |
| CharacterSkillPrepare (SKILL_EFFECT) | sb | 72 | sub_6ADD4C (COutPacket 72 @0x6ae20e) | ✅ = v61 |
| BuffCancelRequest (CANCEL_BUFF, skill-buff) | sb | 71 | SendSkillCancelRequest @0x6afcba (COutPacket 71) | ✅ = v61 |
| GIVE_BUFF (OnTemporaryStatSet) | cb | 28 | OnTemporaryStatSet @0x71af4b | ⛔ remaining — see below |
| CANCEL_BUFF (OnTemporaryStatReset) | cb | 29 | OnTemporaryStatReset @0x71b054 | ⛔ remaining — see below |

n-a (already dispositioned op=-1, out of my hands): remote-move/attack broadcasts
(CUserRemote::OnMove/OnAttack), ENERGY/TOUCH sb, GIVE/CANCEL_FOREIGN_BUFF, SUMMON_ATTACK cb,
SPECIAL_MOVE/SKILL_MACRO (tier2). SUMMON_ATTACK sb / DISTRIBUTE_SP already verified.

## Codec gate added

`libs/atlas-packet/model/attack_info.go` — `legacyGmsNoRangedBulletCoords(t)` (`GMS && <61`):
the v48 ranged sender (sub_6A228C) ends the trailer at characterX/Y @0x6a3965/0x6a3979 then
SendPacket @0x6a3988 — NO bulletX/bulletY. The v61 shoot sender (0x7a67e9) also ends at
characterX/Y (the v61 fixture's bulletX/Y is a pre-existing over-write, left unchanged per
anchor rule). Gate omits the 4-byte bullet trailer for v48 only; v61+/JMS unchanged.

All three attacks: per-mob DamageInfo carries NO trailing anti-hack CRC on v48 (loop returns
straight to the next mob, no Encode4). This is already handled by the existing
`DamageInfo` mob-CRC gate `>= 61` (v48 excluded). Head is byte-identical to v61
(v48 < 72 → no skill-data CRCs; v48 < 79 → 1-byte action).

## Op→body bindings verified (COutPacket opcode at send-site, not symbol)

- op36 melee sub_6A0528 COutPacket(36)@0x6a1711 · op37 ranged sub_6A228C COutPacket(37)@0x6a36bd
- op38 magic sub_6A3AC7 COutPacket(38)@0x6a4af8 · op33 move sub_6E9923 COutPacket(33)@0x6e9ac1
- op72 skill-prep sub_6ADD4C COutPacket(72)@0x6ae20e · op71 SendSkillCancelRequest COutPacket(71)@0x6afcf0
- op30 UPDATE_SKILLS OnChangeSkillRecordResult: Decode1 excl + Decode2 count + 3×Decode4/skill + Decode1 sn (no expiration <83)

## Remaining (genuine substantial divergence — attempted the unblock, not producible in-budget)

**GIVE_BUFF (op28) + CANCEL_BUFF cb (op29)** use a **64-bit (8-byte) CharacterTemporaryStat
mask**, not the 16-byte (128-bit) mask the shared codec emits for all currently-verified
versions. Evidence (decompiled, not assumed):
- `CWvsContext::OnTemporaryStatReset` @0x71b054 → `CInPacket::DecodeBuffer(&v8, 8u)` @0x71b06e (8 bytes).
- `CWvsContext::OnTemporaryStatSet` @0x71af4b → CTS decode helper `sub_5CA524`, whose first op
  is `DecodeBuffer(&v150, 8u)` @0x5ca539 and whose bit tests span bits 0..~46 (64-bit mask),
  each set stat = Decode2(value)+Decode4(reason)+Decode2(duration).
- v61+ read `DecodeBuffer(16)` (per the committed v61/v72/v79 BuffCancel fixtures).

Producing these two cells correctly requires a v48-specific 64-bit mask codec path AND a full
v48 CTS stat-bit table (bits 0-46, distinct from the v83-derived Atlas enum) added to the shared
`CharacterTemporaryStat` model — a feature-sized change that touches the model 8 verified
versions depend on. Not a v61 mirror; flagged with exact evidence rather than fabricating a
16-byte-mask fixture that contradicts the decompile. **EXACT REMAINING = these 2 cells only.**

## Gates / verification

- go test -race (model, character/serverbound, character/clientbound): PASS
- go vet (same three packages): clean; packet-audit tool `go build ./...`: ok
- `packet-audit matrix --check`: exit 0; problem-grep (orphan|dangling|stale|drift|unresolv|malformed): 0
- v48 conflicts: 0 (unchanged)
- Existing version verified counts NOT dropped: v61 208, v72 216, v79 228, v83 367, v84 345,
  v87 379, v95 399, jms 362 (all match bar). v48 verified 113 → 120 (+7).
- STATUS.md diff touches only the 7 target ops; status.json deletions are the 4 attack/skill
  writer sub-struct rows now correctly consumed by their op rows (linkage fix). No
  AuthSuccess/ChatMulti/ReactorHit/SUMMARY/MonsterCarnival drift.

## Tooling changes

- `tools/packet-audit/cmd/run.go`: added `candidatesFromFName` cases for the 5 unnamed v48
  senders (sub_6A0528/6A228C/6A3AC7/6ADD4C/6E9923 → the shared per-op wrappers), mirroring the
  existing v48 pet/summon/mob sub_XXXX pattern.
- Serverbound audit reports regenerated (root command → temp → copied); report IDAName set to
  the v48 primary `sub_XXXX` fname so the op links (matches the pet PetMovementRequest=sub_6E5BD6
  precedent — the named twin is a different version's symbol).

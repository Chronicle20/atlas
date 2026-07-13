# v48 Stage E — Batch 11 (character sub-family C)

Anchor v61, IDB port 13337 (GMS_v48_1_DEVM.exe). Branch `task-113-gms-legacy-versions`.

## Scope resolution
In-scope ❌/🟡 tier-1 character cells (after excluding the reserved heavies
CHAR_INFO/CHARLIST/VIEW_ALL_CHAR/ADD_NEW_CHAR_ENTRY/DISTRIBUTE_AP and the
buff-mask GIVE_BUFF/CANCEL_BUFF): 9 cells.

## Per-cell outcome

| Cell | Dir | v48 op | Result | Notes |
|---|---|---|---|---|
| CANCEL_CHAIR (ChairFixed) | sb | 0x22 | ✅ =v61 | body-verified sub_69DF44 (=HandleXKeyDown): COutPacket(34)+Encode2(0xFFFF) |
| USE_CHAIR (ChairPortable) | sb | 0x23 | ✅ =v61 | sub_712894: COutPacket(35)+Encode4(itemId) |
| FACE_EXPRESSION (ExpressionRequest) | sb | 0x2A | ✅ =v61 | SendEmotionChange @0x71d251: COutPacket(42)+Encode4(emote), no duration/byItemOption (<87) |
| CANCEL_ITEM_EFFECT (ItemCancel) | sb | 0x39 | ✅ =v61 | sub_70DD49: throttle+COutPacket(57)+Encode4(sourceId) |
| CHANGE_KEYMAP (KeyMapChange) | sb | 0x6E | ✅ =v61 | SaveFuncKeyMap @0x4e5fae: COutPacket(110)+Encode4(0)+Encode4(count)+per-key{Encode4(keyIdx)+EncodeBuffer(type[1]+action[4])} via sub_49C937 |
| SET_TAMING_MOB_INFO (CharacterSetTamingMobInfo) | cb | 0x28 | ✅ =v61 | client read sub_72032B: Decode4×4 (charId,level,exp,tiredness)+Decode1(levelUp) |
| REMOVE_PLAYER_FROM_MAP (CharacterDespawn) | cb | 0x65 | ✅ =v61 | client read sub_6B2976 (OnUserLeaveField): Decode4(charId) only |
| SPAWN_PLAYER (CharacterSpawn) | cb | 0x64 | **REMAINING** | v48-specific codec divergence — see below |
| SHOW_STATUS_INFO (OnMessage family) | cb | 0x21 | **REMAINING** | 19-arm dispatcher family, 0 arms fixtured for v48 — dedicated batch |

## Gates added
No new codec gates were needed — all 7 promoted cells are v48==v61 legacy
shape and use the existing version gates (expression `>87`, spawn/others
unchanged). The 7 promotions were achieved by:
- Body-verifying each send-site / client-read in the v48 IDB (distrust IDB
  symbol names — verified by COutPacket/Decode body).
- Setting the registry `fname` for the five `sub_XXXX` cells to the canonical
  demangled name (uniform with v72+ siblings) so `report.IDAName == registry.fname`
  and each op links to its codec + audit report. Original `sub_XXXX` kept in
  `fname_alts`. (CANCEL_CHAIR→CUserLocal::HandleXKeyDown, USE_CHAIR→
  CWvsContext::SendSitOnPortableChairRequest, CANCEL_ITEM_EFFECT→
  CWvsContext::SendStatChangeItemCancelRequest, SET_TAMING_MOB_INFO→
  CWvsContext::OnSetTamingMobInfo, REMOVE_PLAYER_FROM_MAP→CUserPool::OnUserLeaveField).
  FACE_EXPRESSION/CHANGE_KEYMAP already carried demangled fnames.
- Adding the v48 byte-fixture + `packet-audit:verify version=gms_v48` marker +
  TIER1-FIXTURE evidence pin (real send-site addr + decompile hash + `verifies:`)
  per cell.

## Senders named
sub_69DF44→CUserLocal::HandleXKeyDown; sub_712894→SendSitOnPortableChairRequest;
sub_70DD49→SendStatChangeItemCancelRequest; sub_72032B→OnSetTamingMobInfo;
sub_6B2976→OnUserLeaveField (all via registry fname, evidence pins the real sub addr).

## RELOG false-pass check
Not a RELOG-family batch. All fixtures assert real multi-field wire bytes
(chairs Encode2/Encode4, keymap multi-entry loop, taming 17-byte body,
despawn 4-byte) traced to decompile lines — no trivially-passing empty fixture.

## Arms n-a'd
None.

## REMAINING (exact) — 2 cells, both dedicated-batch-sized

### SPAWN_PLAYER (character/clientbound/CharacterSpawn), v48 op 100 (0x64)
Client read = `sub_6B277B` (OnUserEnterField, `Decode4(charId)` then `sub_6BBC17`
= CUserRemote::Init body). Compared field-by-field against the **v79-verified
legacy path** (`sub_8D589E`, port 13340). v48 diverges from the current
`MajorVersion()<83` legacy branch in `spawn.go`:
1. **No jobId short.** v79 reads `Decode2(jobId)` between CTS-foreign and
   AvatarLook (@0x8d597a → this+13312); v48 goes straight from `sub_5CBA1F`
   (CTS foreign) to `AvatarLook::Decode` (@0x6bbcea) — no jobId field.
2. **Different CTS-foreign decoder / mask.** v48 `sub_5CBA1F` (8-byte
   `DecodeBuffer` mask + bit-gated Decode1/2/4) vs v79
   `SecondaryStat::DecodeForRemote`. The foreign temp-stat mask layout must be
   re-verified against `model.CharacterTemporaryStat.EncodeForeign` for v48.
3. **Tail: 6 bool-blocks, not 7.** After the mount triple, v48 reads
   miniroom/adboard/couple/friend/marriage/final-effect (6 `Decode1`-gated
   blocks, @0x6bbed5…0x6bc25c). v79 has an **extra count+loop block**
   (@0x8d5f2b: `Decode1`; if>0 `Decode4(count)` then `count× Decode4`) between
   marriage-ring and final-effect that v48 lacks. v48 also reads no trailing
   team byte (already handled by `!legacy`).
4. AvatarLook internals (`sub_49E1E0`: gender/skin/face + one discarded byte +
   two 0xFF-terminated equip loops + 2×Decode4) still need a field-level match
   against `model.Avatar` before fixturing.

Fix requires a **v48-specific (`<61`) gate** on the spawn codec (drop jobId,
v48 CTS-foreign mask, 6-block tail) on a path **shared with the verified
v61/v72/v79 anchors** — a codec-divergence change, not a fast-path fixture.
Deferred to avoid regressing the v61 anchor (208) / v79 (228) under budget.

### SHOW_STATUS_INFO (CWvsContext::OnMessage family), v48 op 33 (0x21)
Dispatcher — grades **worst-of-siblings** across **19** `StatusMessage*`
report writers (CashItemExpire, IncreaseExp/Fame/Meso/GuildPoint,
DropPickUp*/DropLoss*, System/Update/Complete/Forfeit-QuestRecord,
GiveBuff, GeneralItemExpire, …). **0** of the 19 arms currently have a v48
fixture/marker/evidence. The op cannot flip until every arm is byte-verified
(arms 0-9 operation table = v61, per registry note). This is a full
status-message family batch on its own.

## Commits
1. `777eee03b2` — verify serverbound chair/expression/item-cancel/keymap (5 cells)
2. `4d893b596f` — verify clientbound SET_TAMING_MOB_INFO + REMOVE_PLAYER_FROM_MAP (2 cells)

Each staged explicitly (registry + test files + evidence + STATUS.md/status.json);
no `git add -A`; no out-of-scope report-regen drift.

## Bars
- go test (serverbound + clientbound): PASS. go vet: clean.
- `matrix --check`: exit 0. problem-grep (orphan|dangling|stale|drift|unresolv|malformed): 0.
- Conflicts: 0 total, 0 v48 (unchanged).
- Regression — verified counts held: v61 208 / v72 216 / v79 228 / v83 367 /
  v84 345 / v87 379 / v95 399 / jms 362. v48 120 → **127** (+7).
- Branch after each commit: `task-113-gms-legacy-versions`.

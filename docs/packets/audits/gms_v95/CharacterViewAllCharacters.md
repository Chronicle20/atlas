# CharacterViewAllCharacters (← `CLogin::OnViewAllCharResult#CharacterViewAllCharacters`)

- **IDA:** 0x5de435
- **Atlas file:** `libs/atlas-packet/character/clientbound/view_all.go`
- **Variant:** GMS/v95
- **Branch depth:** 2
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `code byte (case 0 = NORMAL)` | ✅ |  |
| 1 | byte | byte `nWorldID (world for this batch)` | ✅ |  |
| 2 | byte | byte `nCount (character entries in this world)` | ✅ |  |
| 3 | int32 | int32 `GW_CharacterStat::dwCharacterID` | ✅ |  |
| 4 | bytes | bytes `GW_CharacterStat::sCharacterName (padded 13 bytes)` | ✅ |  |
| 5 | byte | byte `GW_CharacterStat::nGender` | ✅ |  |
| 6 | byte | byte `GW_CharacterStat::nSkin` | ✅ |  |
| 7 | int32 | int32 `GW_CharacterStat::nFace` | ✅ |  |
| 8 | int32 | int32 `GW_CharacterStat::nHair` | ✅ |  |
| 9 | int64 | int64 `GW_CharacterStat::petLockerSN (DecodeBuffer 24 bytes = 3 × int64)` | ✅ |  |
| 10 | byte | byte `GW_CharacterStat::nLevel` | ✅ |  |
| 11 | int16 | int16 `GW_CharacterStat::nJob` | ✅ |  |
| 12 | int16 | int16 `GW_CharacterStat::nSTR` | ✅ |  |
| 13 | int16 | int16 `GW_CharacterStat::nDEX` | ✅ |  |
| 14 | int16 | int16 `GW_CharacterStat::nINT` | ✅ |  |
| 15 | int16 | int16 `GW_CharacterStat::nLUK` | ✅ |  |
| 16 | int32 | int32 `GW_CharacterStat::nHP (v95 widened from int16)` | ✅ |  |
| 17 | int32 | int32 `GW_CharacterStat::nMHP (v95 widened from int16)` | ✅ |  |
| 18 | int32 | int32 `GW_CharacterStat::nMP (v95 widened from int16)` | ✅ |  |
| 19 | int32 | int32 `GW_CharacterStat::nMMP (v95 widened from int16)` | ✅ |  |
| 20 | int16 | int16 `GW_CharacterStat::nAP` | ✅ |  |
| 21 | int16 | int16 `GW_CharacterStat::nSP (common-job branch)` | ✅ |  |
| 22 | int32 | int32 `GW_CharacterStat::nEXP` | ✅ |  |
| 23 | int16 | int16 `GW_CharacterStat::nPOP (fame)` | ✅ |  |
| 24 | int32 | int32 `GW_CharacterStat::nTempEXP (gachaponExperience)` | ✅ |  |
| 25 | int32 | int32 `GW_CharacterStat::dwPosMap (mapId)` | ✅ |  |
| 26 | byte | byte `GW_CharacterStat::nPortal (spawnPoint)` | ✅ |  |
| 27 | int32 | int32 `GW_CharacterStat::nPlaytime` | ✅ |  |
| 28 | int16 | int16 `GW_CharacterStat::nSubJob` | ✅ |  |
| 29 | byte | byte `AvatarLook::nGender (duplicate)` | ✅ |  |
| 30 | byte | byte `AvatarLook::nSkin (duplicate)` | ✅ |  |
| 31 | int32 | int32 `AvatarLook::nFace (duplicate)` | ✅ |  |
| 32 | byte | byte `AvatarLook::hairBase/mega flag` | ✅ |  |
| 33 | int32 | int32 `AvatarLook::anHairEquip[0] (hair)` | ✅ |  |
| 34 | byte | byte `AvatarLook::equipment slot (WriteKeyValue byte)` | ✅ |  |
| 35 | int32 | int32 `AvatarLook::equipment itemId (WriteKeyValue int32)` | ✅ |  |
| 36 | byte | byte `AvatarLook::equipment-loop terminator (0xFF)` | ✅ |  |
| 37 | byte | byte `AvatarLook::masked-equip slot` | ✅ |  |
| 38 | int32 | int32 `AvatarLook::masked-equip itemId` | ✅ |  |
| 39 | byte | byte `AvatarLook::masked-equipment-loop terminator (0xFF)` | ✅ |  |
| 40 | int32 | int32 `AvatarLook::nWeaponStickerID` | ✅ |  |
| 41 | int32 | int32 `AvatarLook::anPetID[0]` | ✅ |  |
| 42 | int32 | int32 `AvatarLook::anPetID[1]` | ✅ |  |
| 43 | int32 | int32 `AvatarLook::anPetID[2]` | ✅ |  |
| 44 | byte | byte `rankEnabled / hasRank byte (Decode1(v3) in v95 loop)` | ✅ |  |
| 45 | byte | bytes `rank buffer 16 bytes: worldRank + worldRankGap + jobRank + jobRankGap` | ❌ | width mismatch |
| 46 | int32 | byte `m_bLoginOpt (PIC handling — GMS v95 guard: >v87)` | ❌ | width mismatch |
| 47 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 48 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 49 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 50 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

---

ack: tool-limitation false positive — DecodeBuf granularity mismatch + analyzer linearization offset.

Two independent causes:

1. **DecodeBuf vs 4 × int32**: IDA uses `DecodeBuffer(v3, &rank, 0x10u)` (bulk 16-byte read) for
   the four rank fields (worldRank, worldRankGap, jobRank, jobRankGap). The diff tool maps `DecodeBuf`
   to `bytes` (single entry = 16 bytes), while atlas `CharacterListEntry.Encode` emits four individual
   `WriteInt` calls (4 × int32 = 16 bytes). The type system sees `byte | bytes` at row 45 and
   misaligns every subsequent field.

2. **Linearization offset**: Because the 4 rank int32s (atlas) vs 1 DecodeBuf (IDA) differ in field
   count (4 vs 1), the trailing `WriteByte(1)` PIC byte and the IDA `Decode1 m_bLoginOpt` byte
   fall at different logical positions in the flat comparison sequence, producing phantom ❌s on
   rows 46–50.

On the wire, both encodings are identical: 16 bytes for rank data followed by 1 byte for PIC.
No atlas encoder change is required. Resolution would require the diff tool to expand `DecodeBuf(N)`
into N/width individual entries — deferred to a Phase 3 analyzer enhancement.


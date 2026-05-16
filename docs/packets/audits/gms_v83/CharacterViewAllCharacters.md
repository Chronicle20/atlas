# CharacterViewAllCharacters (← `CLogin::OnViewAllCharResult#CharacterViewAllCharacters`)

- **IDA:** 0x5facca
- **Atlas file:** `libs/atlas-packet/character/clientbound/view_all.go`
- **Variant:** GMS/v83
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
| 9 | int64 | int64 `GW_CharacterStat::petLockerSN (DecodeBuffer 24; atlas for-loop body collapses to 1 static entry)` | ✅ |  |
| 10 | byte | byte `GW_CharacterStat::nLevel` | ✅ |  |
| 11 | int16 | int16 `GW_CharacterStat::nJob` | ✅ |  |
| 12 | int16 | int16 `GW_CharacterStat::nSTR` | ✅ |  |
| 13 | int16 | int16 `GW_CharacterStat::nDEX` | ✅ |  |
| 14 | int16 | int16 `GW_CharacterStat::nINT` | ✅ |  |
| 15 | int16 | int16 `GW_CharacterStat::nLUK` | ✅ |  |
| 16 | int16 | int16 `GW_CharacterStat::nHP (v83 int16)` | ✅ |  |
| 17 | int16 | int16 `GW_CharacterStat::nMHP (v83 int16)` | ✅ |  |
| 18 | int16 | int16 `GW_CharacterStat::nMP (v83 int16)` | ✅ |  |
| 19 | int16 | int16 `GW_CharacterStat::nMMP (v83 int16)` | ✅ |  |
| 20 | int16 | int16 `GW_CharacterStat::nAP` | ✅ |  |
| 21 | int16 | int16 `GW_CharacterStat::nSP (common-job branch)` | ✅ |  |
| 22 | int32 | int32 `GW_CharacterStat::nEXP` | ✅ |  |
| 23 | int16 | int16 `GW_CharacterStat::nPOP (fame)` | ✅ |  |
| 24 | int32 | int32 `GW_CharacterStat::nTempEXP (gachaponExperience)` | ✅ |  |
| 25 | int32 | int32 `GW_CharacterStat::dwPosMap (mapId)` | ✅ |  |
| 26 | byte | byte `GW_CharacterStat::nPortal (spawnPoint)` | ✅ |  |
| 27 | int32 | int32 `GW_CharacterStat::nPlaytime` | ✅ |  |
| 28 | byte | byte `AvatarLook::nGender (duplicate)` | ✅ |  |
| 29 | byte | byte `AvatarLook::nSkin (duplicate)` | ✅ |  |
| 30 | int32 | int32 `AvatarLook::nFace (duplicate)` | ✅ |  |
| 31 | byte | byte `AvatarLook::hairBase/mega flag` | ✅ |  |
| 32 | int32 | int32 `AvatarLook::anHairEquip[0] (hair)` | ✅ |  |
| 33 | byte | byte `AvatarLook::equipment slot (WriteKeyValue byte)` | ✅ |  |
| 34 | int32 | int32 `AvatarLook::equipment itemId (WriteKeyValue int32)` | ✅ |  |
| 35 | byte | byte `AvatarLook::equipment-loop terminator (0xFF)` | ✅ |  |
| 36 | byte | byte `AvatarLook::masked-equip slot` | ✅ |  |
| 37 | int32 | int32 `AvatarLook::masked-equip itemId` | ✅ |  |
| 38 | byte | byte `AvatarLook::masked-equipment-loop terminator (0xFF)` | ✅ |  |
| 39 | int32 | int32 `AvatarLook::nWeaponStickerID` | ✅ |  |
| 40 | int32 | int32 `AvatarLook::anPetID[0]` | ✅ |  |
| 41 | int32 | int32 `AvatarLook::anPetID[1]` | ✅ |  |
| 42 | int32 | int32 `AvatarLook::anPetID[2]` | ✅ |  |
| 43 | byte | byte `rankEnabled / hasRank byte` | ✅ |  |
| 44 | byte | bytes `rank buffer 16 bytes: worldRank + worldRankGap + jobRank + jobRankGap` | ❌ | width mismatch |
| 45 | int32 | byte `m_bLoginOpt (PIC handling — v83 included)` | ❌ | width mismatch |
| 46 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 47 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 48 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |


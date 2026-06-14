# CharacterList (← `CLogin::OnSelectWorldResult`)

- **IDA:** 0x5f9891
- **Atlas file:** `libs/atlas-packet/character/clientbound/list.go`
- **Variant:** GMS/v83
- **Branch depth:** 2
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `resultCode` | ✅ |  |
| 1 | byte | byte `nCount (character entries)` | ✅ |  |
| 2 | int32 | int32 `GW_CharacterStat::dwCharacterID (loop body)` | ✅ |  |
| 3 | bytes | bytes `GW_CharacterStat::sCharacterName (padded 13 bytes)` | ✅ |  |
| 4 | byte | byte `GW_CharacterStat::nGender` | ✅ |  |
| 5 | byte | byte `GW_CharacterStat::nSkin` | ✅ |  |
| 6 | int32 | int32 `GW_CharacterStat::nFace` | ✅ |  |
| 7 | int32 | int32 `GW_CharacterStat::nHair` | ✅ |  |
| 8 | int64 | int64 `GW_CharacterStat::petLockerSN (DecodeBuffer 24; atlas for-loop body collapses to 1 static entry)` | ✅ |  |
| 9 | byte | byte `GW_CharacterStat::nLevel` | ✅ |  |
| 10 | int16 | int16 `GW_CharacterStat::nJob` | ✅ |  |
| 11 | int16 | int16 `GW_CharacterStat::nSTR` | ✅ |  |
| 12 | int16 | int16 `GW_CharacterStat::nDEX` | ✅ |  |
| 13 | int16 | int16 `GW_CharacterStat::nINT` | ✅ |  |
| 14 | int16 | int16 `GW_CharacterStat::nLUK` | ✅ |  |
| 15 | int16 | int16 `GW_CharacterStat::nHP (v83 int16, v95 widened to int32)` | ✅ |  |
| 16 | int16 | int16 `GW_CharacterStat::nMHP (v83 int16)` | ✅ |  |
| 17 | int16 | int16 `GW_CharacterStat::nMP (v83 int16)` | ✅ |  |
| 18 | int16 | int16 `GW_CharacterStat::nMMP (v83 int16)` | ✅ |  |
| 19 | int16 | int16 `GW_CharacterStat::nAP` | ✅ |  |
| 20 | int16 | int16 `GW_CharacterStat::nSP (common-job branch)` | ✅ |  |
| 21 | int32 | int32 `GW_CharacterStat::nEXP` | ✅ |  |
| 22 | int16 | int16 `GW_CharacterStat::nPOP (fame)` | ✅ |  |
| 23 | int32 | int32 `GW_CharacterStat::nTempEXP (gachaponExperience)` | ✅ |  |
| 24 | int32 | int32 `GW_CharacterStat::dwPosMap (mapId)` | ✅ |  |
| 25 | byte | byte `GW_CharacterStat::nPortal (spawnPoint)` | ✅ |  |
| 26 | int32 | int32 `GW_CharacterStat::nPlaytime` | ✅ |  |
| 27 | byte | byte `AvatarLook::nGender (duplicate)` | ✅ |  |
| 28 | byte | byte `AvatarLook::nSkin (duplicate)` | ✅ |  |
| 29 | int32 | int32 `AvatarLook::nFace (duplicate)` | ✅ |  |
| 30 | byte | byte `AvatarLook::hairBase/mega flag` | ✅ |  |
| 31 | int32 | int32 `AvatarLook::anHairEquip[0] (hair)` | ✅ |  |
| 32 | byte | byte `AvatarLook::equipment slot (WriteKeyValue byte)` | ✅ |  |
| 33 | int32 | int32 `AvatarLook::equipment itemId (WriteKeyValue int32)` | ✅ |  |
| 34 | byte | byte `AvatarLook::equipment-loop terminator (0xFF)` | ✅ |  |
| 35 | byte | byte `AvatarLook::masked-equip slot` | ✅ |  |
| 36 | int32 | int32 `AvatarLook::masked-equip itemId` | ✅ |  |
| 37 | byte | byte `AvatarLook::masked-equipment-loop terminator (0xFF)` | ✅ |  |
| 38 | int32 | int32 `AvatarLook::nWeaponStickerID` | ✅ |  |
| 39 | int32 | int32 `AvatarLook::anPetID[0]` | ✅ |  |
| 40 | int32 | int32 `AvatarLook::anPetID[1]` | ✅ |  |
| 41 | byte | int32 `AvatarLook::anPetID[2]` | ❌ | width mismatch |
| 42 | byte | byte `viewAll/onFamily byte` | ✅ |  |
| 43 | int32 | byte `rankEnabled / hasRank byte` | ❌ | width mismatch |
| 44 | int32 | int32 `worldRank` | ✅ |  |
| 45 | int32 | int32 `worldRankMove` | ✅ |  |
| 46 | int32 | int32 `jobRank` | ✅ |  |
| 47 | byte | int32 `jobRankMove` | ❌ | width mismatch |
| 48 | int32 | byte `m_bLoginOpt (hasPic)` | ❌ | width mismatch |
| 49 | byte | int32 `m_nSlotCount` | ❌ | atlas: short — missing trailing field |
| 50 | byte | int32 `m_nBuyCharCount` | ❌ | atlas: short — missing trailing field |


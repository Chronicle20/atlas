# CharacterList (← `CLogin::OnSelectWorldResult`)

- **IDA:** 0x5dda00
- **Atlas file:** `libs/atlas-packet/character/clientbound/list.go`
- **Variant:** GMS/v95
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
| 8 | int64 | int64 `GW_CharacterStat::petLockerSN (DecodeBuffer 24 bytes = 3 × int64; atlas for-loop body collapses to 1 static entry)` | ✅ |  |
| 9 | byte | byte `GW_CharacterStat::nLevel` | ✅ |  |
| 10 | int16 | int16 `GW_CharacterStat::nJob` | ✅ |  |
| 11 | int16 | int16 `GW_CharacterStat::nSTR` | ✅ |  |
| 12 | int16 | int16 `GW_CharacterStat::nDEX` | ✅ |  |
| 13 | int16 | int16 `GW_CharacterStat::nINT` | ✅ |  |
| 14 | int16 | int16 `GW_CharacterStat::nLUK` | ✅ |  |
| 15 | int32 | int32 `GW_CharacterStat::nHP (v95 widened from int16)` | ✅ |  |
| 16 | int32 | int32 `GW_CharacterStat::nMHP (v95 widened from int16)` | ✅ |  |
| 17 | int32 | int32 `GW_CharacterStat::nMP (v95 widened from int16)` | ✅ |  |
| 18 | int32 | int32 `GW_CharacterStat::nMMP (v95 widened from int16)` | ✅ |  |
| 19 | int16 | int16 `GW_CharacterStat::nAP` | ✅ |  |
| 20 | int16 | int16 `GW_CharacterStat::nSP (common-job branch)` | ✅ |  |
| 21 | int32 | int32 `GW_CharacterStat::nEXP` | ✅ |  |
| 22 | int16 | int16 `GW_CharacterStat::nPOP (fame)` | ✅ |  |
| 23 | int32 | int32 `GW_CharacterStat::nTempEXP (gachaponExperience)` | ✅ |  |
| 24 | int32 | int32 `GW_CharacterStat::dwPosMap (mapId)` | ✅ |  |
| 25 | byte | byte `GW_CharacterStat::nPortal (spawnPoint)` | ✅ |  |
| 26 | int32 | int32 `GW_CharacterStat::nPlaytime` | ✅ |  |
| 27 | int16 | int16 `GW_CharacterStat::nSubJob` | ✅ |  |
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
| 42 | byte | int32 `AvatarLook::anPetID[2]` | ❌ | width mismatch |
| 43 | byte | byte `viewAll/onFamily byte` | ✅ |  |
| 44 | int32 | byte `rankEnabled / hasRank byte` | ❌ | width mismatch |
| 45 | int32 | int32 `worldRank` | ✅ |  |
| 46 | int32 | int32 `worldRankMove` | ✅ |  |
| 47 | int32 | int32 `jobRank` | ✅ |  |
| 48 | byte | int32 `jobRankMove` | ❌ | width mismatch |
| 49 | int32 | byte `m_bLoginOpt (hasPic)` | ❌ | width mismatch |
| 50 | int32 | int32 `m_nSlotCount` | ✅ |  |
| 51 | byte | int32 `m_nBuyCharCount` | ❌ | atlas: short — missing trailing field |


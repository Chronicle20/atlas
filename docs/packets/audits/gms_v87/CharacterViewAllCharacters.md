# CharacterViewAllCharacters (← `CLogin::OnViewAllCharResult#CharacterViewAllCharacters`)

- **IDA:** 0x6328eb
- **Atlas file:** `../../libs/atlas-packet/character/clientbound/view_all.go`
- **Variant:** GMS/v87
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
| 16 | int16 | int16 `GW_CharacterStat::nHP (v87 still int16)` | ✅ |  |
| 17 | int16 | int16 `GW_CharacterStat::nMHP (v87 still int16)` | ✅ |  |
| 18 | int16 | int16 `GW_CharacterStat::nMP (v87 still int16)` | ✅ |  |
| 19 | int16 | int16 `GW_CharacterStat::nMMP (v87 still int16)` | ✅ |  |
| 20 | int16 | int16 `GW_CharacterStat::nAP` | ✅ |  |
| 21 | int16 | int16 `GW_CharacterStat::nSP (common-job branch)` | ✅ |  |
| 22 | int32 | int32 `GW_CharacterStat::nEXP` | ✅ |  |
| 23 | int16 | int16 `GW_CharacterStat::nPOP (fame)` | ✅ |  |
| 24 | int32 | int32 `GW_CharacterStat::nTempEXP` | ✅ |  |
| 25 | int32 | int32 `GW_CharacterStat::dwPosMap (mapId)` | ✅ |  |
| 26 | byte | byte `GW_CharacterStat::nPortal (spawnPoint)` | ✅ |  |
| 27 | int32 | int32 `GW_CharacterStat::nPlaytime` | ✅ |  |
| 28 | int16 | int16 `GW_CharacterStat::nSubJob (present in v87)` | ✅ |  |
| 29 | byte | byte `AvatarLook::nGender (duplicate)` | ✅ |  |
| 30 | byte | byte `AvatarLook::nSkin (duplicate)` | ✅ |  |
| 31 | int32 | int32 `AvatarLook::nFace (duplicate)` | ✅ |  |
| 32 | byte | byte `AvatarLook::hairBase/mega flag` | ✅ |  |
| 33 | int32 | int32 `AvatarLook::anHairEquip[0] (hair)` | ✅ |  |
| 34 | byte | byte `AvatarLook::equipment slot` | ✅ |  |
| 35 | int32 | int32 `AvatarLook::equipment itemId` | ✅ |  |
| 36 | byte | byte `AvatarLook::equipment-loop terminator (0xFF)` | ✅ |  |
| 37 | byte | byte `AvatarLook::masked-equip slot` | ✅ |  |
| 38 | int32 | int32 `AvatarLook::masked-equip itemId` | ✅ |  |
| 39 | byte | byte `AvatarLook::masked-equipment-loop terminator (0xFF)` | ✅ |  |
| 40 | int32 | int32 `AvatarLook::nWeaponStickerID` | ✅ |  |
| 41 | int32 | int32 `AvatarLook::anPetID[0]` | ✅ |  |
| 42 | int32 | int32 `AvatarLook::anPetID[1]` | ✅ |  |
| 43 | byte | int32 `AvatarLook::anPetID[2]` | ❌ | width mismatch |
| 44 | byte | byte `rankEnabled / hasRank byte` | ✅ |  |
| 45 | int32 | bytes `rank buffer 16 bytes: worldRank + worldRankGap + jobRank + jobRankGap — final per-entry field; NO m_bLoginOpt at end of case 0 in v87 (added >v87 in v95; gate MajorVersion()>87 in atlas)` | ✅ |  |
| 46 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 47 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 48 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |


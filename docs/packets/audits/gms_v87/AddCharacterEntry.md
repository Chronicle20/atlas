# AddCharacterEntry (← `CLogin::OnCreateNewCharacterResult`)

- **IDA:** 0x631b13
- **Atlas file:** `libs/atlas-packet/character/clientbound/add_entry.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `result code: 0=success, 10=limit, 26=notice, 30=cannotUse` | ✅ |  |
| 1 | int32 | int32 `GW_CharacterStat::dwCharacterID (success path)` | ✅ |  |
| 2 | bytes | bytes `GW_CharacterStat::sCharacterName (padded 13 bytes)` | ✅ |  |
| 3 | byte | byte `GW_CharacterStat::nGender` | ✅ |  |
| 4 | byte | byte `GW_CharacterStat::nSkin` | ✅ |  |
| 5 | int32 | int32 `GW_CharacterStat::nFace` | ✅ |  |
| 6 | int32 | int32 `GW_CharacterStat::nHair` | ✅ |  |
| 7 | int64 | int64 `GW_CharacterStat::petLockerSN (DecodeBuffer 24 bytes)` | ✅ |  |
| 8 | byte | byte `GW_CharacterStat::nLevel` | ✅ |  |
| 9 | int16 | int16 `GW_CharacterStat::nJob` | ✅ |  |
| 10 | int16 | int16 `GW_CharacterStat::nSTR` | ✅ |  |
| 11 | int16 | int16 `GW_CharacterStat::nDEX` | ✅ |  |
| 12 | int16 | int16 `GW_CharacterStat::nINT` | ✅ |  |
| 13 | int16 | int16 `GW_CharacterStat::nLUK` | ✅ |  |
| 14 | int16 | int16 `GW_CharacterStat::nHP (v87 int16; widened to int32 in v95)` | ✅ |  |
| 15 | int16 | int16 `GW_CharacterStat::nMHP (v87 int16)` | ✅ |  |
| 16 | int16 | int16 `GW_CharacterStat::nMP (v87 int16)` | ✅ |  |
| 17 | int16 | int16 `GW_CharacterStat::nMMP (v87 int16)` | ✅ |  |
| 18 | int16 | int16 `GW_CharacterStat::nAP` | ✅ |  |
| 19 | int16 | int16 `GW_CharacterStat::nSP (common-job branch)` | ✅ |  |
| 20 | int32 | int32 `GW_CharacterStat::nEXP` | ✅ |  |
| 21 | int16 | int16 `GW_CharacterStat::nPOP (fame)` | ✅ |  |
| 22 | int32 | int32 `GW_CharacterStat::nTempEXP` | ✅ |  |
| 23 | int32 | int32 `GW_CharacterStat::dwPosMap (mapId)` | ✅ |  |
| 24 | byte | byte `GW_CharacterStat::nPortal (spawnPoint)` | ✅ |  |
| 25 | int32 | int32 `GW_CharacterStat::nPlaytime` | ✅ |  |
| 26 | int16 | int16 `GW_CharacterStat::nSubJob (present in v87)` | ✅ |  |
| 27 | byte | byte `AvatarLook::nGender (duplicate)` | ✅ |  |
| 28 | byte | byte `AvatarLook::nSkin (duplicate)` | ✅ |  |
| 29 | int32 | int32 `AvatarLook::nFace (duplicate)` | ✅ |  |
| 30 | byte | byte `AvatarLook::hairBase/mega flag` | ✅ |  |
| 31 | int32 | int32 `AvatarLook::anHairEquip[0] (hair)` | ✅ |  |
| 32 | byte | byte `AvatarLook::equipment slot` | ✅ |  |
| 33 | int32 | int32 `AvatarLook::equipment itemId` | ✅ |  |
| 34 | byte | byte `AvatarLook::equipment-loop terminator (0xFF)` | ✅ |  |
| 35 | byte | byte `AvatarLook::masked-equip slot` | ✅ |  |
| 36 | int32 | int32 `AvatarLook::masked-equip itemId` | ✅ |  |
| 37 | byte | byte `AvatarLook::masked-equipment-loop terminator (0xFF)` | ✅ |  |
| 38 | int32 | int32 `AvatarLook::nWeaponStickerID` | ✅ |  |
| 39 | int32 | int32 `AvatarLook::anPetID[0]` | ✅ |  |
| 40 | int32 | int32 `AvatarLook::anPetID[1]` | ✅ |  |
| 41 | int32 | int32 `AvatarLook::anPetID[2]` | ✅ |  |
| 42 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 43 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 44 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 45 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 46 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 47 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |


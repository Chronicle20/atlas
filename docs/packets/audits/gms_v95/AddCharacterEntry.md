# AddCharacterEntry (← `CLogin::OnCreateNewCharacterResult`)

- **IDA:** 0x5dab90
- **Atlas file:** `../../libs/atlas-packet/character/clientbound/add_entry.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `v3/v5 result code: 0=success, 10=limit, 26=notice, 30=cannotUse` | ✅ |  |
| 1 | int32 | int32 `GW_CharacterStat::dwCharacterID (success path)` | ✅ |  |
| 2 | bytes | bytes `GW_CharacterStat::sCharacterName (padded 13 bytes)` | ✅ |  |
| 3 | byte | byte `GW_CharacterStat::nGender` | ✅ |  |
| 4 | byte | byte `GW_CharacterStat::nSkin` | ✅ |  |
| 5 | int32 | int32 `GW_CharacterStat::nFace` | ✅ |  |
| 6 | int32 | int32 `GW_CharacterStat::nHair` | ✅ |  |
| 7 | int64 | int64 `GW_CharacterStat::petLockerSN (DecodeBuffer 24 bytes = 3 × int64)` | ✅ |  |
| 8 | byte | byte `GW_CharacterStat::nLevel` | ✅ |  |
| 9 | int16 | int16 `GW_CharacterStat::nJob` | ✅ |  |
| 10 | int16 | int16 `GW_CharacterStat::nSTR` | ✅ |  |
| 11 | int16 | int16 `GW_CharacterStat::nDEX` | ✅ |  |
| 12 | int16 | int16 `GW_CharacterStat::nINT` | ✅ |  |
| 13 | int16 | int16 `GW_CharacterStat::nLUK` | ✅ |  |
| 14 | int32 | int32 `GW_CharacterStat::nHP (v95 widened from int16)` | ✅ |  |
| 15 | int32 | int32 `GW_CharacterStat::nMHP (v95 widened from int16)` | ✅ |  |
| 16 | int32 | int32 `GW_CharacterStat::nMP (v95 widened from int16)` | ✅ |  |
| 17 | int32 | int32 `GW_CharacterStat::nMMP (v95 widened from int16)` | ✅ |  |
| 18 | int16 | int16 `GW_CharacterStat::nAP` | ✅ |  |
| 19 | int16 | int16 `GW_CharacterStat::nSP (common-job branch)` | ✅ |  |
| 20 | int32 | int32 `GW_CharacterStat::nEXP` | ✅ |  |
| 21 | int16 | int16 `GW_CharacterStat::nPOP (fame)` | ✅ |  |
| 22 | int32 | int32 `GW_CharacterStat::nTempEXP (gachaponExperience)` | ✅ |  |
| 23 | int32 | int32 `GW_CharacterStat::dwPosMap (mapId)` | ✅ |  |
| 24 | byte | byte `GW_CharacterStat::nPortal (spawnPoint)` | ✅ |  |
| 25 | int32 | int32 `GW_CharacterStat::nPlaytime` | ✅ |  |
| 26 | int16 | int16 `GW_CharacterStat::nSubJob` | ✅ |  |
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
| 42 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 43 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 44 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 45 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 46 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |


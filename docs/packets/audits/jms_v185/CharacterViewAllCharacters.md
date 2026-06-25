# CharacterViewAllCharacters (← `CLogin::OnViewAllCharResult#CharacterViewAllCharacters`)

- **IDA:** 0x6709e4
- **Atlas file:** `libs/atlas-packet/character/clientbound/view_all.go`
- **Variant:** JMS/v185
- **Branch depth:** 2
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `code byte (case 0 = NORMAL)` | ✅ |  |
| 1 | byte | byte `nWorldID (world for this batch)` | ✅ |  |
| 2 | byte | byte `nCount (character entries in this world)` | ✅ |  |
| 3 | int32 | int32 `GW_CharacterStat::dwCharacterID` | ✅ |  |
| 4 | bytes | bytes `GW_CharacterStat::sCharacterName (13 bytes)` | ✅ |  |
| 5 | byte | byte `GW_CharacterStat::nGender` | ✅ |  |
| 6 | byte | byte `GW_CharacterStat::nSkin` | ✅ |  |
| 7 | int32 | int32 `GW_CharacterStat::nFace` | ✅ |  |
| 8 | int32 | int32 `GW_CharacterStat::nHair` | ✅ |  |
| 9 | int64 | bytes `GW_CharacterStat::aliPetLockerSN (24 bytes)` | ✅ |  |
| 10 | byte | byte `GW_CharacterStat::nLevel` | ✅ |  |
| 11 | int16 | int16 `GW_CharacterStat::nJob` | ✅ |  |
| 12 | int16 | int16 `GW_CharacterStat::nSTR` | ✅ |  |
| 13 | int16 | int16 `GW_CharacterStat::nDEX` | ✅ |  |
| 14 | int16 | int16 `GW_CharacterStat::nINT` | ✅ |  |
| 15 | int16 | int16 `GW_CharacterStat::nLUK` | ✅ |  |
| 16 | int16 | int16 `GW_CharacterStat::nHP (int16)` | ✅ |  |
| 17 | int16 | int16 `GW_CharacterStat::nMHP (int16)` | ✅ |  |
| 18 | int16 | int16 `GW_CharacterStat::nMP (int16)` | ✅ |  |
| 19 | int16 | int16 `GW_CharacterStat::nMMP (int16)` | ✅ |  |
| 20 | int16 | int16 `GW_CharacterStat::extra MMP/SP-CS (common-job else-branch)` | ✅ |  |
| 21 | int16 | int32 `GW_CharacterStat::nAP (jms widened to int32)` | ❌ | width mismatch |
| 22 | int32 | int16 `GW_CharacterStat::nSP` | ❌ | width mismatch |
| 23 | int16 | int32 `GW_CharacterStat::nEXP` | ❌ | width mismatch |
| 24 | int32 | int32 `GW_CharacterStat::extendSP/nPOP block (int32)` | ✅ |  |
| 25 | int32 | byte `GW_CharacterStat::extendSP tail byte` | ❌ | width mismatch |
| 26 | byte | int16 `GW_CharacterStat::posMap HIWORD` | ❌ | width mismatch |
| 27 | int16 | bytes `GW_CharacterStat::nPortal block (8 bytes)` | ✅ |  |
| 28 | int64 | int32 `GW_CharacterStat::nPlaytime` | ❌ | width mismatch |
| 29 | int32 | int32 `GW_CharacterStat::subAvatar/dwCharacterID[1]` | ✅ |  |
| 30 | int32 | int32 `GW_CharacterStat::dwPosMap[1]` | ✅ |  |
| 31 | int32 | byte `AvatarLook::nGender` | ❌ | width mismatch |
| 32 | byte | byte `AvatarLook::nSkin` | ✅ |  |
| 33 | byte | int32 `AvatarLook::nFace` | ❌ | width mismatch |
| 34 | int32 | byte `AvatarLook::hair flag byte` | ❌ | width mismatch |
| 35 | byte | int32 `AvatarLook::anHairEquip[0] (hair)` | ❌ | width mismatch |
| 36 | int32 | byte `AvatarLook::equipment slot (WriteKeyValue byte)` | ❌ | width mismatch |
| 37 | byte | int32 `AvatarLook::equipment itemId (WriteKeyValue int32)` | ❌ | width mismatch |
| 38 | int32 | byte `AvatarLook::equipment-loop terminator (0xFF)` | ❌ | width mismatch |
| 39 | byte | byte `AvatarLook::masked-equip slot` | ✅ |  |
| 40 | byte | int32 `AvatarLook::masked-equip itemId` | ❌ | width mismatch |
| 41 | int32 | byte `AvatarLook::masked-equipment-loop terminator (0xFF)` | ❌ | width mismatch |
| 42 | byte | int32 `AvatarLook::nWeaponStickerID` | ❌ | width mismatch |
| 43 | int32 | bytes `AvatarLook::anPetID (12 bytes)` | ✅ |  |
| 44 | int32 | byte `rankEnabled / hasRank byte` | ❌ | width mismatch |
| 45 | int32 | bytes `rank buffer 16 bytes (worldRank+gap+jobRank+gap)` | ✅ |  |
| 46 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 47 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |
| 48 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 49 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 50 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 51 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |


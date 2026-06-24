# AddCharacterEntry (← `CLogin::OnCreateNewCharacterResult`)

- **IDA:** 0x66ffa8
- **Atlas file:** `libs/atlas-packet/character/clientbound/add_entry.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `v3/v5 result code: 0=success, 10=limit, -24=JMS notice, 30=cannotUse` | ✅ |  |
| 1 | int32 | int32 `dwCharacterID` | ✅ |  |
| 2 | bytes | bytes `sCharacterName (DecodeBuffer 13 bytes)` | ✅ |  |
| 3 | byte | byte `nGender` | ✅ |  |
| 4 | byte | byte `nSkin` | ✅ |  |
| 5 | int32 | int32 `nFace` | ✅ |  |
| 6 | int32 | int32 `nHair` | ✅ |  |
| 7 | int64 | bytes `aliPetLockerSN (DecodeBuffer 24 bytes = 3 x int64)` | ✅ |  |
| 8 | byte | byte `nLevel` | ✅ |  |
| 9 | int16 | int16 `nJob` | ✅ |  |
| 10 | int16 | int16 `nSTR` | ✅ |  |
| 11 | int16 | int16 `nDEX` | ✅ |  |
| 12 | int16 | int16 `nINT` | ✅ |  |
| 13 | int16 | int16 `nLUK` | ✅ |  |
| 14 | int16 | int16 `nHP (int16)` | ✅ |  |
| 15 | int16 | int16 `nMHP (int16)` | ✅ |  |
| 16 | int16 | int16 `nMP (int16)` | ✅ |  |
| 17 | int16 | int16 `nMMP (int16)` | ✅ |  |
| 18 | int16 | int16 `nAP` | ✅ |  |
| 19 | int16 | int16 `nSP (common-job branch; extended-SP jobs read a variable array instead)` | ✅ |  |
| 20 | int32 | int32 `nEXP` | ✅ |  |
| 21 | int16 | int16 `nPOP (fame)` | ✅ |  |
| 22 | int32 | int32 `nTempEXP (gachaponExperience)` | ✅ |  |
| 23 | int32 | int32 `dwPosMap (mapId)` | ✅ |  |
| 24 | byte | byte `nPortal (spawnPoint)` | ✅ |  |
| 25 | int16 | int16 `nSubJob` | ✅ |  |
| 26 | int64 | bytes `jms extra 8-byte field (DecodeBuffer 8)` | ✅ |  |
| 27 | int32 | int32 `nPlaytime` | ✅ |  |
| 28 | int32 | int32 `jms extra int32` | ✅ |  |
| 29 | int32 | int32 `jms extra int32` | ✅ |  |
| 30 | byte | byte `nGender` | ✅ |  |
| 31 | byte | byte `nSkin` | ✅ |  |
| 32 | int32 | int32 `nFace` | ✅ |  |
| 33 | byte | byte `hairBase/mega flag` | ✅ |  |
| 34 | int32 | int32 `anHairEquip[0] (hair)` | ✅ |  |
| 35 | byte | byte `equipment slot (loop entry; 0xFF terminates)` | ✅ |  |
| 36 | int32 | int32 `equipment itemId` | ✅ |  |
| 37 | byte | byte `equipment-loop terminator (0xFF)` | ✅ |  |
| 38 | byte | byte `masked-equip slot (loop entry; 0xFF terminates)` | ✅ |  |
| 39 | int32 | int32 `masked-equip itemId` | ✅ |  |
| 40 | byte | byte `masked-equipment-loop terminator (0xFF)` | ✅ |  |
| 41 | int32 | int32 `nWeaponStickerID` | ✅ |  |
| 42 | int32 | int32 `anPetID[0] (DecodeBuffer 12 = 3 x int32)` | ✅ |  |
| 43 | int32 | int32 `anPetID[1]` | ✅ |  |
| 44 | byte | int32 `anPetID[2]` | ❌ | width mismatch |
| 45 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 46 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 47 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 48 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 49 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |


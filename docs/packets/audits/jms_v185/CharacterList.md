# CharacterList (← `CLogin::OnSelectWorldResult`)

- **IDA:** 0x66f3d8
- **Atlas file:** `libs/atlas-packet/character/clientbound/list.go`
- **Variant:** JMS/v185
- **Branch depth:** 2
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `resultCode` | ✅ |  |
| 1 | string | int32 `dwCharacterID` | ❌ | width mismatch |
| 2 | byte | bytes `sCharacterName (DecodeBuffer 13 bytes)` | ✅ |  |
| 3 | int32 | byte `nGender` | ❌ | width mismatch |
| 4 | bytes | byte `nSkin` | ✅ |  |
| 5 | byte | int32 `nFace` | ❌ | width mismatch |
| 6 | byte | int32 `nHair` | ❌ | width mismatch |
| 7 | int32 | bytes `aliPetLockerSN (DecodeBuffer 24 bytes = 3 x int64)` | ✅ |  |
| 8 | int32 | byte `nLevel` | ❌ | width mismatch |
| 9 | int64 | int16 `nJob` | ❌ | width mismatch |
| 10 | byte | int16 `nSTR` | ❌ | width mismatch |
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
| 21 | int32 | int16 `nPOP (fame)` | ❌ | width mismatch |
| 22 | int16 | int32 `nTempEXP (gachaponExperience)` | ❌ | width mismatch |
| 23 | int32 | int32 `dwPosMap (mapId)` | ✅ |  |
| 24 | int32 | byte `nPortal (spawnPoint)` | ❌ | width mismatch |
| 25 | byte | int16 `nSubJob` | ❌ | width mismatch |
| 26 | int16 | bytes `jms extra 8-byte field (DecodeBuffer 8)` | ✅ |  |
| 27 | int64 | int32 `nPlaytime` | ❌ | width mismatch |
| 28 | int32 | int32 `jms extra int32` | ✅ |  |
| 29 | int32 | int32 `jms extra int32` | ✅ |  |
| 30 | int32 | byte `nGender` | ❌ | width mismatch |
| 31 | byte | byte `nSkin` | ✅ |  |
| 32 | byte | int32 `nFace` | ❌ | width mismatch |
| 33 | int32 | byte `hairBase/mega flag` | ❌ | width mismatch |
| 34 | byte | int32 `anHairEquip[0] (hair)` | ❌ | width mismatch |
| 35 | int32 | byte `equipment slot (loop entry; 0xFF terminates)` | ❌ | width mismatch |
| 36 | byte | int32 `equipment itemId` | ❌ | width mismatch |
| 37 | int32 | byte `equipment-loop terminator (0xFF)` | ❌ | width mismatch |
| 38 | byte | byte `masked-equip slot (loop entry; 0xFF terminates)` | ✅ |  |
| 39 | byte | int32 `masked-equip itemId` | ❌ | width mismatch |
| 40 | int32 | byte `masked-equipment-loop terminator (0xFF)` | ❌ | width mismatch |
| 41 | byte | int32 `nWeaponStickerID` | ❌ | width mismatch |
| 42 | int32 | int32 `anPetID[0] (DecodeBuffer 12 = 3 x int32)` | ✅ |  |
| 43 | int32 | int32 `anPetID[1]` | ✅ |  |
| 44 | int32 | int32 `anPetID[2]` | ✅ |  |
| 45 | byte | byte `onFamily byte (per-entry)` | ✅ |  |
| 46 | byte | byte `rankEnabled / hasRank byte (per-entry)` | ✅ |  |
| 47 | int32 | int32 `worldRank (per-entry)` | ✅ |  |
| 48 | int32 | int32 `worldRankMove (per-entry)` | ✅ |  |
| 49 | int32 | int32 `jobRank (per-entry)` | ✅ |  |
| 50 | int32 | int32 `jobRankMove (per-entry)` | ✅ |  |
| 51 | byte | byte `m_bLoginOpt` | ✅ |  |
| 52 | byte | byte `m_bQuerySSNOnCreateNewCharacter (JMS extra field)` | ✅ |  |
| 53 | int32 | int32 `m_nSlotCount` | ✅ |  |
| 54 | int32 | int32 `m_nBuyCharCount` | ✅ |  |


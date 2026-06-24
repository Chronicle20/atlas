# CharacterSpawn (← `CUserPool::OnUserEnterField`)

- **IDA:** 0xa43ddd
- **Atlas file:** `libs/atlas-packet/character/clientbound/spawn.go`
- **Variant:** JMS/v185
- **Branch depth:** 2
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `` | ✅ |  |
| 1 | byte | byte `this[12276] (level/portal byte)` | ✅ |  |
| 2 | string | string `remote user name` | ✅ |  |
| 3 | string | string `second string (guild/community)` | ✅ |  |
| 4 | int16 | int16 `this[5004]` | ✅ |  |
| 5 | byte | byte `this[5006]` | ✅ |  |
| 6 | int16 | int16 `this[5008]` | ✅ |  |
| 7 | byte | byte `this[5010]` | ✅ |  |
| 8 | bytes | bytes `stat mask (DecodeBuffer 16 = UINT128)` | ✅ |  |
| 9 | int16 | byte `stat (mask bit)` | ❌ | width mismatch |
| 10 | byte | byte `stat (mask bit)` | ✅ |  |
| 11 | byte | int32 `stat (mask bit)` | ❌ | width mismatch |
| 12 | int32 | int32 `stat (mask bit)` | ✅ |  |
| 13 | byte | int32 `stat (mask bit)` | ❌ | width mismatch |
| 14 | int32 | int32 `stat (mask bit)` | ✅ |  |
| 15 | byte | int32 `stat (mask bit)` | ❌ | width mismatch |
| 16 | int32 | int32 `stat (mask bit)` | ✅ |  |
| 17 | int16 | int16 `stat (mask bit)` | ✅ |  |
| 18 | int32 | int32 `stat (mask bit)` | ✅ |  |
| 19 | byte | int16 `stat (mask bit)` | ❌ | width mismatch |
| 20 | int32 | int16 `stat (mask bit)` | ❌ | width mismatch |
| 21 | int32 | int32 `stat (mask bit)` | ✅ |  |
| 22 | int32 | int32 `stat (mask bit)` | ✅ |  |
| 23 | int32 | int32 `stat (mask bit)` | ✅ |  |
| 24 | int32 | int32 `stat (mask bit)` | ✅ |  |
| 25 | int32 | int32 `stat (mask bit)` | ✅ |  |
| 26 | int32 | int32 `stat (mask bit)` | ✅ |  |
| 27 | int32 | int32 `stat (mask bit)` | ✅ |  |
| 28 | int32 | int32 `stat (mask bit)` | ✅ |  |
| 29 | int32 | int32 `stat (mask bit)` | ✅ |  |
| 30 | int32 | int32 `stat (mask bit)` | ✅ |  |
| 31 | byte | int32 `stat (mask bit)` | ❌ | width mismatch |
| 32 | int32 | int32 `stat (mask bit)` | ✅ |  |
| 33 | string | int32 `stat (mask bit)` | ❌ | width mismatch |
| 34 | int64 | int32 `stat (mask bit)` | ❌ | width mismatch |
| 35 | int32 | int32 `stat (mask bit)` | ✅ |  |
| 36 | int32 | int32 `stat (mask bit)` | ✅ |  |
| 37 | byte | byte `trailing nDefenseAtt (unconditional)` | ✅ |  |
| 38 | byte | byte `trailing nDefenseState (unconditional)` | ✅ |  |
| 39 | int32 | unresolved `7x vtable-dispatched per-stat-set conditional decode (hand-trace)` | 🚫 | IDA read-order unresolved: 7x vtable-dispatched per-stat-set conditional decode (hand-trace) |
| 40 | int32 | int16 `this[16560]` | ❌ | width mismatch |
| 41 | int32 | byte `nGender` | ❌ | width mismatch |
| 42 | byte | byte `nSkin` | ✅ |  |
| 43 | int32 | int32 `nFace` | ✅ |  |
| 44 | byte | byte `hairBase/mega flag` | ✅ |  |
| 45 | byte | int32 `anHairEquip[0] (hair)` | ❌ | width mismatch |
| 46 | byte | byte `equipment slot (loop entry; 0xFF terminates)` | ✅ |  |
| 47 | byte | int32 `equipment itemId` | ❌ | atlas: short — missing trailing field |
| 48 | byte | byte `equipment-loop terminator (0xFF)` | ❌ | atlas: short — missing trailing field |
| 49 | byte | byte `masked-equip slot (loop entry; 0xFF terminates)` | ❌ | atlas: short — missing trailing field |
| 50 | byte | int32 `masked-equip itemId` | ❌ | atlas: short — missing trailing field |
| 51 | byte | byte `masked-equipment-loop terminator (0xFF)` | ❌ | atlas: short — missing trailing field |
| 52 | byte | int32 `nWeaponStickerID` | ❌ | atlas: short — missing trailing field |
| 53 | byte | int32 `anPetID[0] (DecodeBuffer 12 = 3 x int32)` | ❌ | atlas: short — missing trailing field |
| 54 | byte | int32 `anPetID[1]` | ❌ | atlas: short — missing trailing field |
| 55 | byte | int32 `anPetID[2]` | ❌ | atlas: short — missing trailing field |
| 56 | byte | int32 `this[9400]` | ❌ | atlas: short — missing trailing field |
| 57 | byte | int32 `this[9404]` | ❌ | atlas: short — missing trailing field |
| 58 | byte | int32 `nChocoCount (carry item effect)` | ❌ | atlas: short — missing trailing field |
| 59 | byte | int32 `nActiveEffectItemID` | ❌ | atlas: short — missing trailing field |
| 60 | byte | int32 `this[16516] (active portable chair)` | ❌ | atlas: short — missing trailing field |
| 61 | byte | int16 `x (position)` | ❌ | atlas: short — missing trailing field |
| 62 | byte | int16 `y (position)` | ❌ | atlas: short — missing trailing field |
| 63 | byte | byte `this[1112] (move action/stance)` | ❌ | atlas: short — missing trailing field |
| 64 | byte | int16 `foothold (GetFoothold)` | ❌ | atlas: short — missing trailing field |
| 65 | byte | byte `pet-loop terminator (while Decode1: per-pet CPet::Init body, Unresolved)` | ❌ | atlas: short — missing trailing field |
| 66 | byte | unresolved `per-pet CPet::Init body (variable; hand-trace)` | ❌ | atlas: short — missing trailing field |
| 67 | byte | int32 `this[12144]` | ❌ | atlas: short — missing trailing field |
| 68 | byte | int32 `this[12148]` | ❌ | atlas: short — missing trailing field |
| 69 | byte | int32 `this[12152]` | ❌ | atlas: short — missing trailing field |
| 70 | byte | byte `mini-room/shop active flag (this[9252])` | ❌ | atlas: short — missing trailing field |
| 71 | byte | int32 `mini-room sn (this[9256])` | ❌ | atlas: short — missing trailing field |
| 72 | byte | string `mini-room title (this[9260])` | ❌ | atlas: short — missing trailing field |
| 73 | byte | byte `this[9264]` | ❌ | atlas: short — missing trailing field |
| 74 | byte | byte `this[9268]` | ❌ | atlas: short — missing trailing field |
| 75 | byte | byte `this[9276]` | ❌ | atlas: short — missing trailing field |
| 76 | byte | byte `this[9272]` | ❌ | atlas: short — missing trailing field |
| 77 | byte | byte `this[9280]` | ❌ | atlas: short — missing trailing field |
| 78 | byte | byte `AD board active flag (this[16552])` | ❌ | atlas: short — missing trailing field |
| 79 | byte | string `AD board text` | ❌ | atlas: short — missing trailing field |
| 80 | byte | byte `couple-ring active flag` | ❌ | atlas: short — missing trailing field |
| 81 | byte | int32 `couple-ring count` | ❌ | atlas: short — missing trailing field |
| 82 | byte | bytes `couple-ring item (16 bytes per entry)` | ❌ | atlas: short — missing trailing field |
| 83 | byte | int32 `couple-ring itemId (per entry)` | ❌ | atlas: short — missing trailing field |
| 84 | byte | byte `friendship-ring active flag` | ❌ | atlas: short — missing trailing field |
| 85 | byte | int32 `friendship-ring count` | ❌ | atlas: short — missing trailing field |
| 86 | byte | bytes `friendship-ring item (16 bytes per entry)` | ❌ | atlas: short — missing trailing field |
| 87 | byte | int32 `friendship-ring itemId (per entry)` | ❌ | atlas: short — missing trailing field |
| 88 | byte | byte `marriage-record active flag` | ❌ | atlas: short — missing trailing field |
| 89 | byte | int32 `this[9416] (marriage)` | ❌ | atlas: short — missing trailing field |
| 90 | byte | int32 `this[9420] (marriage)` | ❌ | atlas: short — missing trailing field |
| 91 | byte | int32 `this[9424] (marriage)` | ❌ | atlas: short — missing trailing field |
| 92 | byte | byte `dragon/effect (1320006) active flag` | ❌ | atlas: short — missing trailing field |
| 93 | byte | byte `final effect active flag (this[16564])` | ❌ | atlas: short — missing trailing field |
| 94 | byte | unresolved `packet var passed to unresolved/indirect call; hand-trace` | ❌ | atlas: short — missing trailing field |


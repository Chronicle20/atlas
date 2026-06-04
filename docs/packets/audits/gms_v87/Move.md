# Move (← `CVecCtrlUser::EndUpdateActive`)

- **IDA:** 0xa5c937
- **Atlas file:** `../../libs/atlas-packet/character/serverbound/move.go`
- **Variant:** GMS/v87
- **Branch depth:** 2
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dr0 (~drInfo[0]) — present in v87, same as v95` | ✅ |  |
| 1 | int32 | int32 `dr1 (~drInfo[1])` | ✅ |  |
| 2 | byte | byte `fieldKey` | ✅ |  |
| 3 | int32 | int32 `dr2 (~drInfo[2])` | ✅ |  |
| 4 | int32 | int32 `dr3 (~drInfo[3])` | ✅ |  |
| 5 | int32 | int32 `crc (field CRC — get_field()+2084)` | ✅ |  |
| 6 | int32 | int32 `dwKey (random seed for CRC32)` | ✅ |  |
| 7 | int32 | int32 `crc32 (CRC32 of bDetect using dwKey)` | ✅ |  |
| 8 | int16 | bytes `movement: CMovePath::Flush — tool cannot linearize loop` | ✅ |  |
| 9 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 10 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 11 | byte | byte `` | ❌ | atlas: extra — client never reads this field |


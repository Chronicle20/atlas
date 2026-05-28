# Move (← `CVecCtrlUser::EndUpdateActive`)

- **IDA:** 0x9cb992
- **Atlas file:** `../../libs/atlas-packet/character/serverbound/move.go`
- **Variant:** GMS/v83
- **Branch depth:** 2
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `fieldKey (Encode1; NO dr0/dr1/dr2/dr3/dwKey/crc32 in v83)` | ✅ |  |
| 1 | int32 | int32 `crc (field CRC for anti-cheat; GMS>28 guard still applies in v83)` | ✅ |  |
| 2 | int16 | bytes `movement: CMovePath::Flush — encoded movement path; tool cannot linearize loop — ack:tool-limitation` | ❌ | width mismatch |
| 3 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 4 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 5 | byte | byte `` | ❌ | atlas: extra — client never reads this field |


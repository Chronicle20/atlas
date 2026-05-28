# MonsterControl (← `CMobPool::OnMobChangeController`)

- **IDA:** 0x6f8b84
- **Atlas file:** `../../libs/atlas-packet/monster/clientbound/control.go`
- **Variant:** JMS/v185
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `controlMode` | ✅ |  |
| 1 | int32 | int32 `dwMobID` | ✅ |  |
| 2 | byte | byte `aggro byte — atlas hardcodes 5` | ✅ |  |
| 3 | int32 | int32 `dwTemplateID via sub_6F75D6 — atlas monsterId` | ✅ |  |
| 4 | int32 | bytes `MonsterModel body` | ❌ | width mismatch |
| 5 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 6 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 7 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 8 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 9 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 10 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 11 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 12 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 13 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 14 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 15 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 16 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 17 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 18 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 19 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 20 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 21 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 22 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 23 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 24 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 25 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 26 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 27 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 28 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 29 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 30 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |


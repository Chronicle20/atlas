# MonsterControl (← `CMobPool::OnMobChangeController`)

- **IDA:** 0x658d10
- **Atlas file:** `libs/atlas-packet/monster/clientbound/control.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `controlMode (v4/v5 — 0 = remote, non-zero = local-controlled)` | ✅ |  |
| 1 | int32 | int32 `moveRandSeed.s1 — conditional: only if controlMode && CClientOptMan::GetOpt(2)` | ✅ |  |
| 2 | byte | int32 `moveRandSeed.s2 — conditional with seed.s1` | ❌ | width mismatch |
| 3 | int32 | int32 `moveRandSeed.s3 — conditional with seed.s1` | ✅ |  |
| 4 | int32 | int32 `dwMobID (uniqueId / v7)` | ✅ |  |
| 5 | int32 | byte `aggro byte — conditional: only if controlMode != 0` | ❌ | width mismatch |
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


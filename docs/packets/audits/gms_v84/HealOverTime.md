# HealOverTime (← `CWvsContext::SendStatChangeRequest`)

- **IDA:** 0xa69c4d
- **Atlas file:** `libs/atlas-packet/character/serverbound/heal_over_time.go`
- **Variant:** GMS/v84
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `` | ✅ |  |
| 1 | int32 | int32 `` | ✅ |  |
| 2 | int16 | int16 `` | ✅ |  |
| 3 | int16 | int16 `` | ✅ |  |
| 4 | byte | byte `` | ✅ |  |


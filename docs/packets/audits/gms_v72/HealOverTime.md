# HealOverTime (← `CWvsContext::SendStatChangeRequest`)

- **IDA:** 0x9179c6
- **Atlas file:** `libs/atlas-packet/character/serverbound/heal_over_time.go`
- **Variant:** GMS/v72
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `` | ✅ |  |
| 1 | int16 | int16 `` | ✅ |  |
| 2 | int16 | int16 `` | ✅ |  |
| 3 | byte | byte `` | ✅ |  |


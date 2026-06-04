# FieldClock (← `CField::OnClock`)

- **IDA:** 0x5361bd
- **Atlas file:** `../../libs/atlas-packet/field/clientbound/clock.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `clockType (mode discriminator; switch arms 0/1/2/3)` | ✅ |  |
| 1 | int32 | byte `flag1 (EventTimerClock enable byte; representative arm = case 3)` | ❌ | width mismatch |
| 2 | int32 | int32 `seconds` | ✅ |  |
| 3 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 4 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 5 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 6 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |


# FieldClock (← `CField::OnClock`)

- **IDA:** 0x55DA5F
- **Atlas file:** `../../libs/atlas-packet/field/clientbound/clock.go`
- **Variant:** GMS/v87
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `clockType (mode discriminator, v4 @0x55da76)` | ✅ |  |
| 1 | int32 | byte `flag1 (EventTimerClock enable byte; representative arm = case 3 @0x55dac2)` | ❌ | width mismatch |
| 2 | int32 | int32 `seconds (@0x55dae6)` | ✅ |  |
| 3 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 4 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 5 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 6 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |


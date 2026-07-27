# FieldClock (← `CField::OnClock`)

- **IDA:** 0x51a522
- **Atlas file:** `libs/atlas-packet/field/clientbound/clock.go`
- **Variant:** GMS/v72
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `clockType @0x51a539` | ✅ |  |
| 1 | int32 | int32 `seconds (EventClock type 0) @0x51a732` | ✅ |  |
| 2 | byte | byte `hour (TownClock type 1) @0x51a703` | ✅ |  |
| 3 | byte | byte `minute (TownClock type 1) @0x51a710` | ✅ |  |
| 4 | byte | byte `second (TownClock type 1) @0x51a712` | ✅ |  |
| 5 | byte | int32 `seconds (TimerClock type 2) @0x51a6e0` | ❌ | width mismatch |
| 6 | int32 | byte `flag (EventTimerClock type 3) @0x51a59e` | ❌ | width mismatch |
| 7 | byte | int32 `seconds (EventTimerClock type 3) @0x51a62b` | ❌ | width mismatch |
| 8 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 9 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |


# HealOverTime (← `CWvsContext::SendStatChangeRequest`)

- **IDA:** 
- **Atlas file:** `../../libs/atlas-packet/character/serverbound/heal_over_time.go`
- **Variant:** GMS/v84
- **Branch depth:** 2
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | unresolved `function not found in IDB` | 🚫 | IDA read-order unresolved: function not found in IDB |
| 1 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 2 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 3 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 4 | byte | byte `` | ❌ | atlas: extra — client never reads this field |


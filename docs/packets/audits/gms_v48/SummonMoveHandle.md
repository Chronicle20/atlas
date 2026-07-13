# SummonMoveHandle (← `CVecCtrlSummoned::EndUpdateActive`)

- **IDA:** 
- **Atlas file:** `libs/atlas-packet/summon/serverbound/move.go`
- **Variant:** GMS/v48
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | unresolved `function not found in IDB` | 🚫 | IDA read-order unresolved: function not found in IDB |
| 1 | bytes | byte `` | ❌ | atlas: extra — client never reads this field |


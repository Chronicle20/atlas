# SummonMove (← `CSummonedPool::OnMove`)

- **IDA:** 
- **Atlas file:** `libs/atlas-packet/summon/clientbound/move.go`
- **Variant:** GMS/v61
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | unresolved `function not found in IDB` | 🚫 | IDA read-order unresolved: function not found in IDB |
| 1 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 2 | bytes | byte `` | ❌ | atlas: extra — client never reads this field |


# Open (← `CUserLocal::OnOpenUI`)

- **IDA:** 0x8ebf80
- **Atlas file:** `libs/atlas-packet/rps/clientbound/operation.go`
- **Variant:** GMS/v92
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ✅ |  |
| 1 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |


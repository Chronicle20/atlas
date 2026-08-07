# StartError (← `CClientSocket::OnConnect#StartError`)

- **IDA:** 
- **Atlas file:** `libs/atlas-packet/socket/serverbound/start_error.go`
- **Variant:** GMS/v92
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | unresolved `dispatcher-family arm not harvested for gms_v92 (see notes)` | 🚫 | IDA read-order unresolved: dispatcher-family arm not harvested for gms_v92 (see notes) |
| 1 | bytes | byte `` | ❌ | atlas: extra — client never reads this field |


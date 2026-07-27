# StartError (← `CClientSocket::OnConnect#StartError`)

- **IDA:** 
- **Atlas file:** `libs/atlas-packet/socket/serverbound/start_error.go`
- **Variant:** GMS/v48
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | unresolved `function not found in IDB` | 🚫 | IDA read-order unresolved: function not found in IDB |
| 1 | bytes | byte `` | ❌ | atlas: extra — client never reads this field |


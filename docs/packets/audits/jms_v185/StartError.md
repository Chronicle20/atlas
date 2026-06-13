# StartError (← `CClientSocket::OnConnect#StartError`)

- **IDA:** 0x4b0066
- **Atlas file:** `libs/atlas-packet/socket/serverbound/start_error.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int16 `length uint16 (byte count of exception log data)` | ✅ |  |
| 1 | bytes | bytes `bytes variable-length exception log data` | ✅ |  |


# StartError (← `CClientSocket::OnConnect#StartError`)

- **IDA:** 0x494ed1
- **Atlas file:** `../../libs/atlas-packet/socket/serverbound/start_error.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int16 `length uint16 (byte count of exception log data) — v83 OnConnect obfuscated; layout inferred from v95` | ✅ |  |
| 1 | bytes | bytes `bytes variable-length exception log data` | ✅ |  |


# ServerStatus (← `CLogin::OnCheckUserLimitResult`)

- **IDA:** 0x5d2250
- **Atlas file:** `libs/atlas-packet/login/clientbound/server_status.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int16 `world status (2 bytes — v95 reads as 2 × Decode1; atlas writes WriteShort, wire-equivalent)` | ✅ |  |


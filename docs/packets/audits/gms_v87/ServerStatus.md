# ServerStatus (← `CLogin::OnCheckUserLimitResult`)

- **IDA:** 0x630af9
- **Atlas file:** `../../libs/atlas-packet/login/clientbound/server_status.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int16 `world status (2 bytes — same as v95)` | ✅ |  |


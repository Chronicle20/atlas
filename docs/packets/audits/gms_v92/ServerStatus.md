# ServerStatus (← `CLogin::OnCheckUserLimitResult`)

- **IDA:** 0x5c8790
- **Atlas file:** `libs/atlas-packet/login/clientbound/server_status.go`
- **Variant:** GMS/v92
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | byte `` | ❌ | width mismatch |
| 1 | byte | byte `` | ❌ | atlas: short — missing trailing field |


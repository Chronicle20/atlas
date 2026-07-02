# ServerStatus (← `CLogin::OnCheckUserLimitResult`)

- **IDA:** 0x56660e
- **Atlas file:** `libs/atlas-packet/login/clientbound/server_status.go`
- **Variant:** GMS/v61
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int16 `world status (2 bytes; v61 sub_56660E@0x56660e reads as 2 x Decode1 @0x566623/@0x566626, atlas writes WriteShort -- wire-equivalent)` | ✅ |  |


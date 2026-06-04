# ConfirmManage (← `CWvsContext::OnEntrustedShopCheckResult#ConfirmManage`)

- **IDA:** 0x9ffcb0
- **Atlas file:** `../../libs/atlas-packet/merchant/clientbound/operation.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (case 0x11 = CONFIRM_MANAGE)` | ✅ |  |
| 1 | int32 | int32 `shopId / dwCharacterID (v24)` | ✅ |  |
| 2 | int16 | int16 `position / slot index (v25)` | ✅ |  |
| 3 | int64 | int64 `liCashItemSN — 8-byte serial number (DecodeBuffer 8, stored as _LARGE_INTEGER)` | ✅ |  |


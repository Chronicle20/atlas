# ConfirmManage (← `CWvsContext::OnEntrustedShopCheckResult#ConfirmManage`)

- **IDA:** 0xb0ee59
- **Atlas file:** `libs/atlas-packet/merchant/clientbound/operation.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (case 0x11 = CONFIRM_MANAGE)` | ✅ |  |
| 1 | int32 | int32 `shopId / dwCharacterID` | ✅ |  |
| 2 | int16 | int16 `position / slot index` | ✅ |  |
| 3 | int64 | int64 `liCashItemSN (DecodeBuffer 8)` | ✅ |  |


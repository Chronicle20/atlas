# RemoteShopWarp (← `CWvsContext::OnEntrustedShopCheckResult#RemoteShopWarp`)

- **IDA:** 0xabf9ea
- **Atlas file:** `../../libs/atlas-packet/merchant/clientbound/operation.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (case 0x10 = REMOTE_SHOP_WARP)` | ✅ |  |
| 1 | int32 | int32 `shopId (v23)` | ✅ |  |
| 2 | byte | byte `channelId (v14 — 0xFE/0xFD/0xFF = error; otherwise shows YesNo warp dialog)` | ✅ |  |


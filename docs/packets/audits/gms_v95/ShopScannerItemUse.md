# ShopScannerItemUse (← `CWvsContext::SendShopScannerItemUseRequest`)

- **IDA:** 0x9e10e0
- **Atlas file:** `libs/atlas-packet/merchant/serverbound/shop_scanner_item_use.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int16 `source (nPOS inventory slot). COutPacket(0x5A); gated nItemID/10000==231` | ✅ |  |
| 1 | int32 | int32 `itemId (nItemID)` | ✅ |  |
| 2 | int32 | int32 `searchItemId (nItemID arg) — appended by CUIShopScanner::SendScanPacket` | ✅ |  |
| 3 | byte | byte `descending (bDescendingOrder) — SendScanPacket` | ✅ |  |
| 4 | int32 | int32 `updateTime (get_update_time; trailing, NO leading updateTime any version) — SendScanPacket` | ✅ |  |


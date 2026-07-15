# ShopScannerHotList (← `CWvsContext::OnShopScannerResult#HotList`)

- **IDA:** 0xa076c0
- **Atlas file:** `libs/atlas-packet/merchant/clientbound/shop_scanner_result.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (=7; #HotList). switch(Decode1()-6)==1` | ✅ |  |
| 1 | byte | byte `count (byte; hot-list length)` | ✅ |  |
| 2 | int32 | int32 `itemId (m_anHotList entry)` | ✅ |  |


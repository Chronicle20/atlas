# ShopScannerResult (← `CWvsContext::OnShopScannerResult#Result`)

- **IDA:** 0x849800
- **Atlas file:** `libs/atlas-packet/merchant/clientbound/shop_scanner_result.go`
- **Variant:** GMS/v61
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (=6; #Result). switch(Decode1()-6)` | ✅ |  |
| 1 | int32 | int32 `itemId (nItemID)` | ✅ |  |
| 2 | int32 | int32 `nCount (result record count)` | ✅ |  |
| 3 | int32 | string `ownerName (sCharacterName)` | ❌ | width mismatch |
| 4 | string | int32 `mapId (dwFieldID)` | ❌ | width mismatch |
| 5 | int32 | string `title (sTitle)` | ❌ | width mismatch |
| 6 | string | int32 `bundles (nNumber)` | ❌ | width mismatch |
| 7 | int32 | int32 `bundleSize (nSet)` | ✅ |  |
| 8 | int32 | int32 `price (nPrice)` | ✅ |  |
| 9 | int32 | int32 `ownerId (dwMiniRoomSN)` | ✅ |  |
| 10 | int32 | byte `channelId (nChannelID)` | ❌ | width mismatch |
| 11 | byte | byte `inventoryType (nTI)` | ✅ |  |
| 12 | byte | bytes `GW_ItemSlotBase::Decode(&pItem) = model.Asset; only when nTI==1` | ✅ |  |
| 13 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |


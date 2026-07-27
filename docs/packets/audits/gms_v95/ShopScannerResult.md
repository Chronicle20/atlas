# ShopScannerResult (← `CWvsContext::OnShopScannerResult#Result`)

- **IDA:** 0xa076c0
- **Atlas file:** `libs/atlas-packet/merchant/clientbound/shop_scanner_result.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** 🔍

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (=6; #Result). switch(Decode1()-6)` | ✅ |  |
| 1 | int32 | int32 `nNpcShopPrice (>0 inserts synthetic npc-shop row; Atlas always 0)` | ✅ |  |
| 2 | int32 | int32 `itemId (nItemID)` | ✅ |  |
| 3 | int32 | int32 `nCount (result record count)` | ✅ |  |
| 4 | string | string `ownerName (sCharacterName)` | ✅ |  |
| 5 | int32 | int32 `mapId (dwFieldID)` | ✅ |  |
| 6 | string | string `title (sTitle)` | ✅ |  |
| 7 | int32 | int32 `bundles (nNumber)` | ✅ |  |
| 8 | int32 | int32 `bundleSize (nSet)` | ✅ |  |
| 9 | int32 | int32 `price (nPrice)` | ✅ |  |
| 10 | int32 | int32 `ownerId (dwMiniRoomSN)` | ✅ |  |
| 11 | byte | byte `channelId (nChannelID)` | ✅ |  |
| 12 | byte | byte `inventoryType (nTI)` | ✅ |  |
| 13 | byte | bytes `GW_ItemSlotBase::Decode(&pItem) = model.Asset; only when nTI==1` | 🔍 | sub-struct: asset — see _substruct/ |


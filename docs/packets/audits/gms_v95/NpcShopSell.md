# NpcShopSell (← `CShopDlg::SendSellRequest`)

- **IDA:** 0x6e7260
- **Atlas file:** `../../libs/atlas-packet/npc/serverbound/shop_sell.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int16 `slot (inventory position nPOS)` | ✅ |  |
| 1 | int32 | int32 `itemId (nItemID)` | ✅ |  |
| 2 | int16 | int16 `quantity` | ✅ |  |


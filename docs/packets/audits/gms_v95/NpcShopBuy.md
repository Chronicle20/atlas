# NpcShopBuy (← `CShopDlg::SendBuyRequest`)

- **IDA:** 0x6e9bb0
- **Atlas file:** `../../libs/atlas-packet/npc/serverbound/shop_buy.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int16 `slot (commodity index in shop)` | ✅ |  |
| 1 | int32 | int32 `itemId` | ✅ |  |
| 2 | int16 | int16 `quantity (nCount)` | ✅ |  |
| 3 | int32 | int32 `discountPrice` | ✅ |  |


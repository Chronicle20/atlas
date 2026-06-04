# NpcShopSell (← `CShopDlg::SendSellRequest`)

- **IDA:** 0x756a04
- **Atlas file:** `../../libs/atlas-packet/npc/serverbound/shop_sell.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int16 `slot / nPOS (a2)` | ✅ |  |
| 1 | int32 | int32 `itemId (v30)` | ✅ |  |
| 2 | int16 | int16 `quantity (v32)` | ✅ |  |


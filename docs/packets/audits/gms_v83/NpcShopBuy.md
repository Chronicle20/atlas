# NpcShopBuy (← `CShopDlg::SendBuyRequest`)

- **IDA:** 0x7561c1
- **Atlas file:** `../../libs/atlas-packet/npc/serverbound/shop_buy.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int16 `slot (commodity index in shop; v57)` | ✅ |  |
| 1 | int32 | int32 `itemId (v58)` | ✅ |  |
| 2 | int16 | int16 `quantity (a2)` | ✅ |  |
| 3 | int32 | int32 `discountPrice / unit meso price (v8[6])` | ✅ |  |


Ack: world-audit Phase 3 v83 (12b npc) on 2026-05-28

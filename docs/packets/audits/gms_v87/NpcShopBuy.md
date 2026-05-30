# NpcShopBuy (← `CShopDlg::SendBuyRequest`)

- **IDA:** 0x7a1d49
- **Atlas file:** `../../libs/atlas-packet/npc/serverbound/shop_buy.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int16 `shop slot index (v46)` | ✅ |  |
| 1 | int32 | int32 `itemId (TSecType GetData)` | ✅ |  |
| 2 | int16 | int16 `quantity (v68)` | ✅ |  |
| 3 | int32 | int32 `discountPrice / unit price (v48)` | ✅ |  |


Ack: world-audit Phase 3 v87 cross-version on 2026-05-28

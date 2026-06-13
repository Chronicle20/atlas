# NpcShopBuy (← `CShopDlg::SendBuyRequest`)

- **IDA:** 0x7561c1
- **Atlas file:** `libs/atlas-packet/npc/serverbound/shop_buy.go`
- **Variant:** GMS/v83
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `leading sub-action byte (task-081 off-by-one remediation 2026-06-10)` | ✅ |  |
| 1 | int16 | int16 `slot (commodity index in shop; v57)` | ✅ |  |
| 2 | int32 | int32 `itemId (v58)` | ✅ |  |
| 3 | int16 | int16 `quantity (a2)` | ✅ |  |
| 4 | int32 | int32 `discountPrice / unit meso price (v8[6])` | ✅ |  |


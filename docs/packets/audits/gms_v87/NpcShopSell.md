# NpcShopSell (← `CShopDlg::SendSellRequest`)

- **IDA:** 0x7a256b
- **Atlas file:** `libs/atlas-packet/npc/serverbound/shop_sell.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `leading sub-action byte (task-081 off-by-one remediation 2026-06-10)` | ✅ |  |
| 1 | int16 | int16 `inventory position (v28)` | ✅ |  |
| 2 | int32 | int32 `itemId (v27)` | ✅ |  |
| 3 | int16 | int16 `quantity (v29)` | ✅ |  |


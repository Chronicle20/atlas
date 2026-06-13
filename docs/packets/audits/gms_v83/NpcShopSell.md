# NpcShopSell (← `CShopDlg::SendSellRequest`)

- **IDA:** 0x756a04
- **Atlas file:** `libs/atlas-packet/npc/serverbound/shop_sell.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `leading sub-action byte (task-081 off-by-one remediation 2026-06-10)` | ✅ |  |
| 1 | int16 | int16 `slot / nPOS (a2)` | ✅ |  |
| 2 | int32 | int32 `itemId (v30)` | ✅ |  |
| 3 | int16 | int16 `quantity (v32)` | ✅ |  |


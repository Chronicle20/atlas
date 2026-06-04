# NpcShopBuy (← `CShopDlg::SendBuyRequest`)

- **IDA:** 0x7ca2c9
- **Atlas file:** `../../libs/atlas-packet/npc/serverbound/shop_buy.go`
- **Variant:** JMS/v185
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | byte `op = 0 (buy @0x7caaf3)` | ❌ | width mismatch |
| 1 | int32 | int16 `slot / commodity index (@0x7cab10)` | ❌ | width mismatch |
| 2 | int16 | int32 `itemId (@0x7cab20)` | ❌ | width mismatch |
| 3 | byte | int16 `quantity (@0x7cab2b)` | ❌ | atlas: short — missing trailing field |


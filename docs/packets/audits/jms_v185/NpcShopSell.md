# NpcShopSell (← `CShopDlg::SendSellRequest`)

- **IDA:** 0x7cacab
- **Atlas file:** `../../libs/atlas-packet/npc/serverbound/shop_sell.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | byte `op = 1 (sell @0x7cae68)` | ❌ | width mismatch |
| 1 | int32 | int16 `nPOS / slot (@0x7cae73)` | ❌ | width mismatch |
| 2 | int16 | int32 `nItemID (@0x7cae7e)` | ❌ | width mismatch |
| 3 | byte | int16 `quantity (@0x7cae89)` | ❌ | atlas: short — missing trailing field |


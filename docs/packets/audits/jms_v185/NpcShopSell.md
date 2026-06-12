# NpcShopSell (← `CShopDlg::SendSellRequest`)

- **IDA:** 0x7cacab
- **Atlas file:** `../../libs/atlas-packet/npc/serverbound/shop_sell.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `op = 1 (sell @0x7cae68)` | ✅ |  |
| 1 | int16 | int16 `nPOS / slot (@0x7cae73)` | ✅ |  |
| 2 | int32 | int32 `nItemID (@0x7cae7e)` | ✅ |  |
| 3 | int16 | int16 `quantity (@0x7cae89)` | ✅ |  |


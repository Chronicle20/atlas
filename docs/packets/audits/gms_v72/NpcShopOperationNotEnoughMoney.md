# NpcShopOperationNotEnoughMoney (← `CShopDlg::OnPacket#NotEnoughMoney`)

- **IDA:** 0x6a912b
- **Atlas file:** `libs/atlas-packet/npc/clientbound/shop_operation.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (NOT_ENOUGH_MONEY mode-only notice arm: v72 mode 2)` | ✅ |  |


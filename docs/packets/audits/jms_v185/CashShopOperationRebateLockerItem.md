# CashShopOperationRebateLockerItem (← `CCashShop::OnRebateLockerItem`)

- **IDA:** 0x47c059
- **Atlas file:** `../../libs/atlas-packet/cash/serverbound/shop_operation_rebate_locker_item.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | string `spw (secondary password). op-byte 0x1B (NOT GMS 0x1C). JMS has SPW string (atlas else-branch has int birthday)` | ❌ | atlas: short — missing trailing field |
| 1 | byte | bytes `8-byte locker SN (Src). matches atlas unk long` | ❌ | atlas: short — missing trailing field |


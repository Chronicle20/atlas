# CashShopOperationRebateLockerItem (← `CCashShop::OnRebateLockerItem`)

- **IDA:** 0x485840
- **Atlas file:** `../../libs/atlas-packet/cash/serverbound/shop_operation_rebate_locker_item.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | string `sSPW secondary-password string (atlas models leading int birthday - MISMATCH)` | ❌ | atlas: short — missing trailing field |
| 1 | byte | bytes `v5 8 bytes locker SN (atlas reads uint64 unk)` | ❌ | atlas: short — missing trailing field |


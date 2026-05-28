# CashShopOperationRebateLockerItem (← `CCashShop::OnRebateLockerItem`)

- **IDA:** 0x46bde1
- **Atlas file:** `../../libs/atlas-packet/cash/serverbound/shop_operation_rebate_locker_item.go`
- **Variant:** GMS/v83
- **Branch depth:** 2
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `ask_SPW() int (a2). NOTE: v83 is a 4-byte int; v95 sends EncodeStr sSPW` | ✅ |  |
| 1 | int64 | bytes `8-byte locker serial (v4)` | ❌ | width mismatch |


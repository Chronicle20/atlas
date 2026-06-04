# CashShopOperationRebateLockerItem (← `CCashShop::OnRebateLockerItem`)

- **IDA:** 0x46bde1
- **Atlas file:** `../../libs/atlas-packet/cash/serverbound/shop_operation_rebate_locker_item.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | int32 `ask_SPW() int (a2). NOTE: v83 is a 4-byte int; v95 sends EncodeStr sSPW` | ❌ | atlas: short — missing trailing field |
| 1 | byte | bytes `8-byte locker serial (v4)` | ❌ | atlas: short — missing trailing field |


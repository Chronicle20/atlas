# CashShopOperationRebateLockerItem (← `CCashShop::OnRebateLockerItem`)

- **IDA:** 0x46bde1
- **Atlas file:** `libs/atlas-packet/cash/serverbound/shop_operation_rebate_locker_item.go`
- **Variant:** GMS/v83
- **Branch depth:** 3
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `leading sub-action byte (task-081 off-by-one remediation 2026-06-10)` | ❌ | width mismatch |
| 1 | int64 | int32 `ask_SPW() int (a2). NOTE: v83 is a 4-byte int; v95 sends EncodeStr sSPW` | ❌ | width mismatch |
| 2 | byte | bytes `8-byte locker serial (v4)` | ❌ | atlas: short — missing trailing field |


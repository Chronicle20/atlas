# ShopOperationRebateLockerItem (← `CCashShop::OnRebateLockerItem`)

- **IDA:** 0x485840
- **Atlas file:** `../../libs/atlas-packet/cash/serverbound/shop_operation_rebate_locker_item.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | string `sSPW secondary-password string (atlas models leading int birthday - MISMATCH)` | ❌ | width mismatch |
| 1 | int64 | bytes `v5 8 bytes locker SN (atlas reads uint64 unk)` | ❌ | width mismatch |


> defer: version-gated — leading field is an SPW string in v95 (atlas models int birthday). See _pending.md "Cash serverbound SPW-string vs birthday-int divergence".

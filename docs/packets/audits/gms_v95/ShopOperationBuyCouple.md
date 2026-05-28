# ShopOperationBuyCouple (← `CCashShop::OnBuyCouple`)

- **IDA:** 0x490d80
- **Atlas file:** `../../libs/atlas-packet/cash/serverbound/shop_operation_buy_couple.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | string `sSPW secondary-password string (atlas models leading int birthday - MISMATCH)` | ❌ | width mismatch |
| 1 | int32 | int32 `dwOption (option)` | ✅ |  |
| 2 | int32 | int32 `nCommSN (serialNumber)` | ✅ |  |
| 3 | string | string `sGiveTo (name)` | ✅ |  |
| 4 | string | string `sText (message)` | ✅ |  |


> defer: version-gated — leading field is an SPW string in v95 (atlas models int birthday); SPW is per-region/version. See _pending.md "Cash serverbound SPW-string vs birthday-int divergence".

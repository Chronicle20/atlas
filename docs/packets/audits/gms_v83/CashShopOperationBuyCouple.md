# CashShopOperationBuyCouple (← `CCashShop::OnBuyCouple`)

- **IDA:** 0x46ffe7
- **Atlas file:** `../../libs/atlas-packet/cash/serverbound/shop_operation_buy_couple.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | int32 `ask_SPW() int (v31). NOTE: v83 is a 4-byte int; v95 sends EncodeStr sSPW` | ❌ | atlas: short — missing trailing field |
| 1 | byte | int32 `option (v37)` | ❌ | atlas: short — missing trailing field |
| 2 | byte | int32 `serialNumber (arg0)` | ❌ | atlas: short — missing trailing field |
| 3 | byte | string `name (a2)` | ❌ | atlas: short — missing trailing field |
| 4 | byte | string `message (a3)` | ❌ | atlas: short — missing trailing field |


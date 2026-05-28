# CashShopOperationBuyCouple (← `CCashShop::OnBuyCouple`)

- **IDA:** 0x47a820
- **Atlas file:** `../../libs/atlas-packet/cash/serverbound/shop_operation_buy_couple.go`
- **Variant:** GMS/v87
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `leading ask_SPW int (v31 = sub_A37DDD). v87 is a 4-byte INT (line 106), NOT EncodeStr sSPW. The SPW string is v95-only; >=95 gate CORRECT.` | ✅ |  |
| 1 | int32 | int32 `option (v37)` | ✅ |  |
| 2 | int32 | int32 `serialNumber (arg0)` | ✅ |  |
| 3 | string | string `name (v35)` | ✅ |  |
| 4 | string | string `message (v33)` | ✅ |  |


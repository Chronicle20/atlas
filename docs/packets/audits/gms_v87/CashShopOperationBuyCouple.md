# CashShopOperationBuyCouple (← `CCashShop::OnBuyCouple`)

- **IDA:** 0x47a820
- **Atlas file:** `libs/atlas-packet/cash/serverbound/shop_operation_buy_couple.go`
- **Variant:** GMS/v87
- **Branch depth:** 3
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `leading sub-action byte (task-081 off-by-one remediation 2026-06-10)` | ❌ | width mismatch |
| 1 | int32 | int32 `leading ask_SPW int (v31 = sub_A37DDD). v87 is a 4-byte INT (line 106), NOT EncodeStr sSPW. The SPW string is v95-only; >=95 gate CORRECT.` | ✅ |  |
| 2 | int32 | int32 `option (v37)` | ✅ |  |
| 3 | string | int32 `serialNumber (arg0)` | ❌ | width mismatch |
| 4 | string | string `name (v35)` | ✅ |  |
| 5 | byte | string `message (v33)` | ❌ | atlas: short — missing trailing field |


# CashShopOperationBuyFriendship (← `CCashShop::OnBuyFriendship`)

- **IDA:** 0x47b293
- **Atlas file:** `../../libs/atlas-packet/cash/serverbound/shop_operation_buy_friendship.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | int32 `leading ask_SPW int (v31 = sub_A37DDD). v87 is a 4-byte INT (line 104), NOT EncodeStr sSPW. The SPW string is v95-only; >=95 gate CORRECT.` | ❌ | atlas: short — missing trailing field |
| 1 | byte | int32 `option (v37)` | ❌ | atlas: short — missing trailing field |
| 2 | byte | int32 `serialNumber (arg0)` | ❌ | atlas: short — missing trailing field |
| 3 | byte | string `name (v35)` | ❌ | atlas: short — missing trailing field |
| 4 | byte | string `message (v33)` | ❌ | atlas: short — missing trailing field |


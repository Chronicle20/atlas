# CashShopOperationBuyFriendship (← `CCashShop::OnBuyFriendship`)

- **IDA:** 0x470a5a
- **Atlas file:** `libs/atlas-packet/cash/serverbound/shop_operation_buy_friendship.go`
- **Variant:** GMS/v83
- **Branch depth:** 3
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `leading sub-action byte (task-081 off-by-one remediation 2026-06-10)` | ❌ | width mismatch |
| 1 | int32 | int32 `ask_SPW() int (v31). NOTE: v83 is a 4-byte int; v95 sends EncodeStr sSPW` | ✅ |  |
| 2 | int32 | int32 `option (v37)` | ✅ |  |
| 3 | string | int32 `serialNumber (arg0)` | ❌ | width mismatch |
| 4 | string | string `name (a2)` | ✅ |  |
| 5 | byte | string `message (a3)` | ❌ | atlas: short — missing trailing field |


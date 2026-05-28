# ShopOperationBuyFriendship (← `CCashShop::OnBuyFriendship`)

- **IDA:** 0x470a5a
- **Atlas file:** `../../libs/atlas-packet/cash/serverbound/shop_operation_buy_friendship.go`
- **Variant:** GMS/v83
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `ask_SPW() int (v31). NOTE: v83 is a 4-byte int; v95 sends EncodeStr sSPW` | ✅ |  |
| 1 | int32 | int32 `option (v37)` | ✅ |  |
| 2 | int32 | int32 `serialNumber (arg0)` | ✅ |  |
| 3 | string | string `name (a2)` | ✅ |  |
| 4 | string | string `message (a3)` | ✅ |  |


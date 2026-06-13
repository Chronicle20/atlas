# InteractionOperationMerchantRemoveFromBlackList (← `CEntrustedShopDlg::DeleteBlackList`)

- **IDA:** 0x54bbf9
- **Atlas file:** `libs/atlas-packet/interaction/serverbound/operation_merchant_remove_from_black_list.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | byte `` | ❌ | width mismatch |
| 1 | byte | string `` | ❌ | atlas: short — missing trailing field |


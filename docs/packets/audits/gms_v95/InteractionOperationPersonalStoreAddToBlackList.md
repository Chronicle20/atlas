# InteractionOperationPersonalStoreAddToBlackList (← `CPersonalShopDlg::OnClickBanButton`)

- **IDA:** 0x69b1c0
- **Atlas file:** `../../libs/atlas-packet/interaction/serverbound/operation_personal_store_add_to_black_list.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `slot (nSlot byte)` | ✅ |  |
| 1 | string | string `name (user id)` | ✅ |  |


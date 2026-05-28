# OperationPersonalStoreSetBlackList (← `CPersonalShopDlg::DeliverBlackList`)

- **IDA:** 0x6fdeda
- **Atlas file:** `../../libs/atlas-packet/interaction/serverbound/operation_personal_store_set_black_list.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int16 `count (config blacklist size)` | ✅ |  |
| 1 | string | string `per-entry character name; repeated count times (string[], NOT byte[])` | ✅ |  |


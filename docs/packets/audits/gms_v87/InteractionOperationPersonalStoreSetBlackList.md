# InteractionOperationPersonalStoreSetBlackList (← `CPersonalShopDlg::DeliverBlackList`)

- **IDA:** 0x74146f
- **Atlas file:** `../../libs/atlas-packet/interaction/serverbound/operation_personal_store_set_black_list.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int16 `count (config blacklist size, v2)` | ✅ |  |
| 1 | string | string `per-entry character name; repeated count times (string[], NOT byte[]) — same as v83/v95. Unconditional fix holds.` | ✅ |  |


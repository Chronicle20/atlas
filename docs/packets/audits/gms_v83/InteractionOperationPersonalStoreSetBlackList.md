# InteractionOperationPersonalStoreSetBlackList (← `CPersonalShopDlg::DeliverBlackList`)

- **IDA:** 0x6fdeda
- **Atlas file:** `libs/atlas-packet/interaction/serverbound/operation_personal_store_set_black_list.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | byte `leading sub-action byte (task-081 off-by-one remediation 2026-06-10)` | ❌ | width mismatch |
| 1 | string | int16 `count (config blacklist size)` | ❌ | width mismatch |
| 2 | byte | string `per-entry character name; repeated count times (string[], NOT byte[])` | ❌ | atlas: short — missing trailing field |


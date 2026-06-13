# InteractionOperationPersonalStoreSetBlackList (← `CPersonalShopDlg::DeliverBlackList`)

- **IDA:** 0x71a1f8
- **Atlas file:** `libs/atlas-packet/interaction/serverbound/operation_personal_store_set_black_list.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | byte `` | ❌ | width mismatch |
| 1 | string | int16 `` | ❌ | width mismatch |
| 2 | byte | string `` | ❌ | atlas: short — missing trailing field |


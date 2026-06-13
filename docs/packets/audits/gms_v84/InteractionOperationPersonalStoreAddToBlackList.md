# InteractionOperationPersonalStoreAddToBlackList (← `CPersonalShopDlg::OnClickBanButton`)

- **IDA:** 0x71a2ac
- **Atlas file:** `libs/atlas-packet/interaction/serverbound/operation_personal_store_add_to_black_list.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ✅ |  |
| 1 | string | byte `` | ❌ | width mismatch |
| 2 | byte | string `` | ❌ | atlas: short — missing trailing field |


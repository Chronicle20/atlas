# InteractionOperationPersonalStoreSetBlackList (← `CPersonalShopDlg::DeliverBlackList`)

- **IDA:** 0x763021
- **Atlas file:** `libs/atlas-packet/interaction/serverbound/operation_personal_store_set_black_list.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | byte `leading sub-action byte (task-081 off-by-one remediation 2026-06-10)` | ❌ | width mismatch |
| 1 | string | int16 `count (m_asBlackList size). op-byte 0x1B consumed by dispatcher` | ❌ | width mismatch |
| 2 | byte | string `per-entry name loop (string[]). JMS string[] — atlas unconditional []string fix is correct` | ❌ | atlas: short — missing trailing field |


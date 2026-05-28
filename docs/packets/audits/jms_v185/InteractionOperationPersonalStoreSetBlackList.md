# InteractionOperationPersonalStoreSetBlackList (← `CPersonalShopDlg::DeliverBlackList`)

- **IDA:** 0x763021
- **Atlas file:** `../../libs/atlas-packet/interaction/serverbound/operation_personal_store_set_black_list.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int16 `count (m_asBlackList size). op-byte 0x1B consumed by dispatcher` | ✅ |  |
| 1 | string | string `per-entry name loop (string[]). JMS string[] — atlas unconditional []string fix is correct` | ✅ |  |


# RpsOperationSelect (← `CRPSGameDlg::SendSelection`)

- **IDA:** 0x6ca280
- **Atlas file:** `libs/atlas-packet/rps/serverbound/operation_select.go`
- **Variant:** GMS/v92
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ✅ |  |
| 1 | byte | byte `` | ❌ | atlas: short — missing trailing field |


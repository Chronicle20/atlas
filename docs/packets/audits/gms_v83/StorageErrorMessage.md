# StorageErrorMessage (← `CTrunkDlg::OnPacket#ErrorMessage`)

- **IDA:** 0x7c8a4c
- **Atlas file:** `libs/atlas-packet/storage/clientbound/error.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (23)` | ✅ |  |
| 1 | byte | byte `enabled flag (if true -> read message)` | ✅ |  |
| 2 | string | string `message` | ✅ |  |


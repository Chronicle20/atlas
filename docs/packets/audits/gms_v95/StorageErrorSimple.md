# StorageErrorSimple (← `CTrunkDlg::OnPacket#ErrorSimple`)

- **IDA:** 0x76a990
- **Atlas file:** `../../libs/atlas-packet/storage/clientbound/error.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (error code; client maps to StringPool message)` | ✅ |  |


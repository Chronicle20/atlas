# StorageErrorOneOfAKind (← `CTrunkDlg::OnPacket#ErrorOneOfAKind`)

- **IDA:** 0x76a990
- **Atlas file:** `libs/atlas-packet/storage/clientbound/error_modes.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (ONE_OF_A_KIND = 12; error code -> StringPool message, no further reads)` | ✅ |  |

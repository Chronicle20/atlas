# StorageOperationMeso (← `CTrunkDlg::SendGetMoneyRequest`)

- **IDA:** 0x7688e0
- **Atlas file:** `../../libs/atlas-packet/storage/serverbound/operation_meso.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `amount (signed; +withdraw / -deposit; mode byte 7 written by dispatcher)` | ✅ |  |


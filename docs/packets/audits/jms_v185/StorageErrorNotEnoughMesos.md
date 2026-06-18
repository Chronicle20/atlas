# StorageErrorNotEnoughMesos (← `CTrunkDlg::OnPacket#ErrorNotEnoughMesos`)

- **IDA:** 0x84e5a1
- **Atlas file:** `libs/atlas-packet/storage/clientbound/error_modes.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (NOT_ENOUGH_MESOS = 10; error code -> StringPool message, no further reads)` | ✅ |  |

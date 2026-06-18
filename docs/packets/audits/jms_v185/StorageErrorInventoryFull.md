# StorageErrorInventoryFull (← `CTrunkDlg::OnPacket#ErrorInventoryFull`)

- **IDA:** 0x84e5a1
- **Atlas file:** `libs/atlas-packet/storage/clientbound/error_modes.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (INVENTORY_FULL = 9; error code -> StringPool message, no further reads)` | ✅ |  |

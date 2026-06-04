# StorageUpdateMeso (← `CTrunkDlg::OnPacket#UpdateMeso`)

- **IDA:** 0x76a990
- **Atlas file:** `../../libs/atlas-packet/storage/clientbound/error.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (19 = UPDATE_MESO; dispatcher case 0x13 -> SetGetItems)` | ✅ |  |
| 1 | byte | byte `slotCount (m_nSlotCount)` | ✅ |  |
| 2 | int64 | int64 `tab-flag bitmask (atlas writes 2 = currency-only; 8 bytes)` | ✅ |  |
| 3 | int32 | int32 `meso (m_nMoney; read because flag&2)` | ✅ |  |


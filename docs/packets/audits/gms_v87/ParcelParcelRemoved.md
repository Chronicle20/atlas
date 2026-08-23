# ParcelParcelRemoved (← `CParcelDlg::OnPacket#ParcelRemoved`)

- **IDA:** 0x7349fe
- **Atlas file:** `libs/atlas-packet/parcel/clientbound/parcel.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (23/0x17 PARCEL_REMOVED; dispatch byte)` | ✅ |  |
| 1 | int32 | int32 `parcelId` | ✅ |  |
| 2 | byte | byte `kind (3 = deleted; else = claimed)` | ✅ |  |


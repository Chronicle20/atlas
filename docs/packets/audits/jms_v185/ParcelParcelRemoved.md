# ParcelParcelRemoved (← `CParcelDlg::OnPacket#ParcelRemoved`)

- **IDA:** 0x755d43
- **Atlas file:** `libs/atlas-packet/parcel/clientbound/parcel.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (24 PARCEL_REMOVED; dispatch byte, jms_v185 shift from v83's mode 23)` | ✅ |  |
| 1 | int32 | int32 `parcelId` | ✅ |  |
| 2 | byte | byte `kind (3 = deleted; else = claimed)` | ✅ |  |


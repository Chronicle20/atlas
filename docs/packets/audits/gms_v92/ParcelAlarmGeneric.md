# ParcelAlarmGeneric (← `CParcelDlg::OnPacket#AlarmGeneric`)

- **IDA:** 0x6895fe
- **Atlas file:** `libs/atlas-packet/parcel/clientbound/parcel.go`
- **Variant:** GMS/v92
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (27/0x1B ALARM_GENERIC; dispatch byte)` | ✅ |  |
| 1 | byte | byte `hasItem` | ✅ |  |


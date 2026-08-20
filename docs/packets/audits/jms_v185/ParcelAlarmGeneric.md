# ParcelAlarmGeneric (← `CParcelDlg::OnPacket#AlarmGeneric`)

- **IDA:** 0x755b4b
- **Atlas file:** `libs/atlas-packet/parcel/clientbound/parcel.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (28 ALARM_GENERIC; dispatch byte, jms_v185 shift from v83's mode 27)` | ✅ |  |
| 1 | byte | byte `hasItem` | ✅ |  |


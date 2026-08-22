# ParcelAlarmNamed (← `CParcelDlg::OnPacket#AlarmNamed`)

- **IDA:** 0x755bf5
- **Atlas file:** `libs/atlas-packet/parcel/clientbound/parcel.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (26 ALARM_NAMED; dispatch byte, jms_v185 shift from v83's mode 25)` | ✅ |  |
| 1 | string | string `senderName` | ✅ |  |
| 2 | byte | byte `hasItem` | ✅ |  |


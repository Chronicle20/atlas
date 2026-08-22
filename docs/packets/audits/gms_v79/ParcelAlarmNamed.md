# ParcelAlarmNamed (← `CParcelDlg::OnPacket#AlarmNamed`)

- **IDA:** 0x6838cf
- **Atlas file:** `libs/atlas-packet/parcel/clientbound/parcel.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (25/0x19 ALARM_NAMED; dispatch byte)` | ✅ |  |
| 1 | string | string `senderName` | ✅ |  |
| 2 | byte | byte `hasItem` | ✅ |  |


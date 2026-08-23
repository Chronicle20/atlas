# ParcelOpenQuick (← `CParcelDlg::OnPacket#OpenQuick`)

- **IDA:** 0x755bcf
- **Atlas file:** `libs/atlas-packet/parcel/clientbound/parcel.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (27 OPEN_QUICK; dispatch byte, no body, jms_v185 shift from v83's mode 26)` | ✅ |  |


# ParcelSuccessfullySent (← `CParcelDlg::OnPacket#SuccessfullySent`)

- **IDA:** 0x755ad9
- **Atlas file:** `libs/atlas-packet/parcel/clientbound/parcel.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (19 SUCCESSFULLY_SENT; dispatch byte, notice-only, no body; also gates the CloseParcelDlg side effect, jms_v185 shift from v83's mode 18)` | ✅ |  |


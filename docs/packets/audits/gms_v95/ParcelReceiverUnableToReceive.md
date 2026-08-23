# ParcelReceiverUnableToReceive (← `CParcelDlg::OnPacket#ReceiverUnableToReceive`)

- **IDA:** 0x68f081
- **Atlas file:** `libs/atlas-packet/parcel/clientbound/parcel.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (15 RECEIVER_UNABLE_TO_RECEIVE; dispatch byte, notice-only, no body)` | ✅ |  |


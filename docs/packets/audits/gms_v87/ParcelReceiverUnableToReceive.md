# ParcelReceiverUnableToReceive (← `CParcelDlg::OnPacket#ReceiverUnableToReceive`)

- **IDA:** 0x734c30
- **Atlas file:** `libs/atlas-packet/parcel/clientbound/parcel.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (15/0x0F RECEIVER_UNABLE_TO_RECEIVE; dispatch byte, notice-only, no body)` | ✅ |  |


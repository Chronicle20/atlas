# ParcelReceiverUnableToReceive (← `CParcelDlg::OnPacket#ReceiverUnableToReceive`)

- **IDA:** 0x65fe62
- **Atlas file:** `libs/atlas-packet/parcel/clientbound/parcel.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (15/0x0F ReceiverUnableToReceive; dispatch byte, notice-only, no body)` | ✅ |  |


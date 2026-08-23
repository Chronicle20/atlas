# ParcelReceiverStorageFull (← `CParcelDlg::OnPacket#ReceiverStorageFull`)

- **IDA:** 0x65fe78
- **Atlas file:** `libs/atlas-packet/parcel/clientbound/parcel.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (14/0x0E ReceiverStorageFull; dispatch byte, notice-only, no body)` | ✅ |  |


# ParcelReceiverStorageFull (← `CParcelDlg::OnPacket#ReceiverStorageFull`)

- **IDA:** 0x680868
- **Atlas file:** `libs/atlas-packet/parcel/clientbound/parcel.go`
- **Variant:** GMS/v92
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (14 RECEIVER_STORAGE_FULL; dispatch byte, notice-only, no body)` | ✅ |  |


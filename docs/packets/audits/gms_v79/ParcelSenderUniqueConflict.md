# ParcelSenderUniqueConflict (← `CParcelDlg::OnPacket#SenderUniqueConflict`)

- **IDA:** 0x683cc4
- **Atlas file:** `libs/atlas-packet/parcel/clientbound/parcel.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (16/0x10 SenderUniqueConflict; dispatch byte, notice-only, no body)` | ✅ |  |


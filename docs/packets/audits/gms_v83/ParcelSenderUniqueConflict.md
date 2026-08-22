# ParcelSenderUniqueConflict (← `CParcelDlg::OnPacket#SenderUniqueConflict`)

- **IDA:** 0x6f5c99
- **Atlas file:** `libs/atlas-packet/parcel/clientbound/parcel.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (16/0x10 SENDER_UNIQUE_CONFLICT; dispatch byte, notice-only, no body)` | ✅ |  |


# ParcelSuccessfullySent (← `CParcelDlg::OnPacket#SuccessfullySent`)

- **IDA:** 0x683d29
- **Atlas file:** `libs/atlas-packet/parcel/clientbound/parcel.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (18/0x12 SuccessfullySent; dispatch byte, notice-only, no body)` | ✅ |  |


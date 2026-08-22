# ParcelSuccessfullySent (← `CParcelDlg::OnPacket#SuccessfullySent`)

- **IDA:** 0x6896e1
- **Atlas file:** `libs/atlas-packet/parcel/clientbound/parcel.go`
- **Variant:** GMS/v92
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (18/0x12 SUCCESSFULLY_SENT; dispatch byte, notice-only, no body; also gates the CloseParcelDlg side effect)` | ✅ |  |


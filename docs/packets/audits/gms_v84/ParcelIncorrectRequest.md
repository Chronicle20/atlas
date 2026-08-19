# ParcelIncorrectRequest (← `CParcelDlg::OnPacket#IncorrectRequest`)

- **IDA:** 0x70e122
- **Atlas file:** `libs/atlas-packet/parcel/clientbound/parcel.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (11/0x0B INCORRECT_REQUEST; dispatch byte, notice-only, no body)` | ✅ |  |


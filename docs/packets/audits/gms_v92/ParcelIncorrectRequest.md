# ParcelIncorrectRequest (← `CParcelDlg::OnPacket#IncorrectRequest`)

- **IDA:** 0x68081b
- **Atlas file:** `libs/atlas-packet/parcel/clientbound/parcel.go`
- **Variant:** GMS/v92
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (11 INCORRECT_REQUEST; dispatch byte, notice-only, no body)` | ✅ |  |


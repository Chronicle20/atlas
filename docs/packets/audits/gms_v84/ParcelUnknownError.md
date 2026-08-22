# ParcelUnknownError (← `CParcelDlg::OnPacket#UnknownError`)

- **IDA:** 0x70e19e
- **Atlas file:** `libs/atlas-packet/parcel/clientbound/parcel.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (19/0x13 UNKNOWN_ERROR; dispatch byte, notice-only, no body)` | ✅ |  |


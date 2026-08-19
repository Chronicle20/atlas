# ParcelUnknownError (← `CParcelDlg::OnPacket#UnknownError`)

- **IDA:** 0x6808c5
- **Atlas file:** `libs/atlas-packet/parcel/clientbound/parcel.go`
- **Variant:** GMS/v92
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (19 UNKNOWN_ERROR; dispatch byte, notice-only, no body)` | ✅ |  |


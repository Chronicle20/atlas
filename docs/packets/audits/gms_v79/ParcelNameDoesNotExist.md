# ParcelNameDoesNotExist (← `CParcelDlg::OnPacket#NameDoesNotExist`)

- **IDA:** 0x683c82
- **Atlas file:** `libs/atlas-packet/parcel/clientbound/parcel.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (12/0x0C NameDoesNotExist; dispatch byte, notice-only, no body)` | ✅ |  |


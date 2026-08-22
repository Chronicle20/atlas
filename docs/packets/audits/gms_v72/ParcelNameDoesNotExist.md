# ParcelNameDoesNotExist (← `CParcelDlg::OnPacket#NameDoesNotExist`)

- **IDA:** 0x65fea4
- **Atlas file:** `libs/atlas-packet/parcel/clientbound/parcel.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (12/0x0C NameDoesNotExist; dispatch byte, notice-only, no body)` | ✅ |  |


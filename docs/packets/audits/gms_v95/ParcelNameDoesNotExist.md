# ParcelNameDoesNotExist (← `CParcelDlg::OnPacket#NameDoesNotExist`)

- **IDA:** 0x68f034
- **Atlas file:** `libs/atlas-packet/parcel/clientbound/parcel.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (12 NAME_DOES_NOT_EXIST; dispatch byte, notice-only, no body)` | ✅ |  |


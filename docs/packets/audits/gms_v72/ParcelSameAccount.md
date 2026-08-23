# ParcelSameAccount (← `CParcelDlg::OnPacket#SameAccount`)

- **IDA:** 0x65fe8e
- **Atlas file:** `libs/atlas-packet/parcel/clientbound/parcel.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (13/0x0D SameAccount; dispatch byte, notice-only, no body)` | ✅ |  |


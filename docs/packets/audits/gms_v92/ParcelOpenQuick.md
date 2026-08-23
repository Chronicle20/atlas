# ParcelOpenQuick (← `CParcelDlg::OnPacket#OpenQuick`)

- **IDA:** 0x689235
- **Atlas file:** `libs/atlas-packet/parcel/clientbound/parcel.go`
- **Variant:** GMS/v92
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (26/0x1A OPEN_QUICK; dispatch byte, no body)` | ✅ |  |


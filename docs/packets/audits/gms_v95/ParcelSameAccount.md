# ParcelSameAccount (← `CParcelDlg::OnPacket#SameAccount`)

- **IDA:** 0x68f04e
- **Atlas file:** `libs/atlas-packet/parcel/clientbound/parcel.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (13 SAME_ACCOUNT; dispatch byte, notice-only, no body)` | ✅ |  |


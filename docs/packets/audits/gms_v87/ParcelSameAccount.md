# ParcelSameAccount (← `CParcelDlg::OnPacket#SameAccount`)

- **IDA:** 0x734c5c
- **Atlas file:** `libs/atlas-packet/parcel/clientbound/parcel.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (13/0x0D SAME_ACCOUNT; dispatch byte, notice-only, no body)` | ✅ |  |


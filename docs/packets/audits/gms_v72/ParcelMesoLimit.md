# ParcelMesoLimit (← `CParcelDlg::OnPacket#MesoLimit`)

- **IDA:** 0x65ff5e
- **Atlas file:** `libs/atlas-packet/parcel/clientbound/parcel.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (17/0x11 MesoLimit; dispatch byte, notice-only, no body)` | ✅ |  |


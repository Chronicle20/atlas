# ParcelMesoLimit (← `CParcelDlg::OnPacket#MesoLimit`)

- **IDA:** 0x6f5d11
- **Atlas file:** `libs/atlas-packet/parcel/clientbound/parcel.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (17/0x11 MESO_LIMIT; dispatch byte, notice-only, no body)` | ✅ |  |


# ParcelMesoLimit (← `CParcelDlg::OnPacket#MesoLimit`)

- **IDA:** 0x6807e7
- **Atlas file:** `libs/atlas-packet/parcel/clientbound/parcel.go`
- **Variant:** GMS/v92
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (17 MESO_LIMIT; dispatch byte, notice-only, no body)` | ✅ |  |


# ParcelMesoLimit (← `CParcelDlg::OnPacket#MesoLimit`)

- **IDA:** 0x70e1c6
- **Atlas file:** `libs/atlas-packet/parcel/clientbound/parcel.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (17/0x11 MESO_LIMIT; dispatch byte, notice-only, no body)` | ✅ |  |


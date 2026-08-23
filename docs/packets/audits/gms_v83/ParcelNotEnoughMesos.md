# ParcelNotEnoughMesos (← `CParcelDlg::OnPacket#NotEnoughMesos`)

- **IDA:** 0x6f5c83
- **Atlas file:** `libs/atlas-packet/parcel/clientbound/parcel.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (10/0x0A NOT_ENOUGH_MESOS; dispatch byte, notice-only, no body)` | ✅ |  |


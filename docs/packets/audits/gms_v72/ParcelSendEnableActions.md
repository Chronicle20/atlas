# ParcelSendEnableActions (← `CParcelDlg::OnPacket#SendEnableActions`)

- **IDA:** 0x65fe59
- **Atlas file:** `libs/atlas-packet/parcel/clientbound/parcel.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (9/0x09 SendEnableActions; dispatch byte, notice-only, no body)` | ✅ |  |


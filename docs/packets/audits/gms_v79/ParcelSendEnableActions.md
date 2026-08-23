# ParcelSendEnableActions (← `CParcelDlg::OnPacket#SendEnableActions`)

- **IDA:** 0x683c28
- **Atlas file:** `libs/atlas-packet/parcel/clientbound/parcel.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (9/0x09 SendEnableActions; dispatch byte, notice-only, no body)` | ✅ |  |


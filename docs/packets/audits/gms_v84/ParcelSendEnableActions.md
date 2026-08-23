# ParcelSendEnableActions (← `CParcelDlg::OnPacket#SendEnableActions`)

- **IDA:** 0x70e0a6
- **Atlas file:** `libs/atlas-packet/parcel/clientbound/parcel.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (9 SEND_ENABLE_ACTIONS; dispatch byte, no notice text, no body)` | ✅ |  |


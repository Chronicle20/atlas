# ParcelSendEnableActions (← `CParcelDlg::OnPacket#SendEnableActions`)

- **IDA:** 0x68efe0
- **Atlas file:** `libs/atlas-packet/parcel/clientbound/parcel.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (9 SEND_ENABLE_ACTIONS; dispatch byte, no notice text, no body)` | ✅ |  |


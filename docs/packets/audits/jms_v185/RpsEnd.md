# RpsEnd (← `CRPSGameDlg::OnPacket#END`)

- **IDA:** 0x7ae489
- **Atlas file:** `libs/atlas-packet/rps/clientbound/operation.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte only (case 13 = CLOSE; CWnd::Destroy @0x7ae489, no further wire reads)` | ✅ |  |


# RpsEnd (← `CRPSGameDlg::OnPacket#END`)

- **IDA:** 0x6c1e08
- **Atlas file:** `libs/atlas-packet/rps/clientbound/operation.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte only (case 13 = CLOSE; CWnd::Destroy @0x6c1e08, no further wire reads)` | ✅ |  |


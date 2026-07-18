# RpsStartSelect (← `CRPSGameDlg::OnPacket#START_SELECT`)

- **IDA:** 0x76200d
- **Atlas file:** `libs/atlas-packet/rps/clientbound/operation.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte only (case 9 = START_SELECT; enable R/P/S buttons + arm selection timer, no further wire reads)` | ✅ |  |


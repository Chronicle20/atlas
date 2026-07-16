# CashVegaScroll (← `CUIVega::OnVegaResult`)

- **IDA:** 0x7bf7b0
- **Atlas file:** `libs/atlas-packet/cash/clientbound/vega_scroll.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `vega result mode byte; single Decode1 routed to START/RESULT/notice arms (gms_v95 CUIVega::OnPacket(0x1AD), task-130 Task 4 IDA-verified 0x7bf7b0)` | ✅ |  |


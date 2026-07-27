# CashVegaScroll (← `CUIVega::OnVegaResult`)

- **IDA:** 0x82d8d5
- **Atlas file:** `libs/atlas-packet/cash/clientbound/vega_scroll.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `vega result mode byte; single Decode1 routed to START/RESULT/notice arms (gms_v83 CUIVega::OnPacket(0x166), task-130 Task 4 IDA-verified 0x82d8d5)` | ✅ |  |


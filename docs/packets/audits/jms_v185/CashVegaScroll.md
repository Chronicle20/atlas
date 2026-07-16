# CashVegaScroll (← `CUIVega::OnVegaResult`)

- **IDA:** 0x8b89ad
- **Atlas file:** `libs/atlas-packet/cash/clientbound/vega_scroll.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `vega result mode byte; single Decode1 routed to START/RESULT/notice arms (jms_v185 CUIVega::OnPacket(0x183), task-130 Task 4 IDA-verified 0x8b89ad)` | ✅ |  |


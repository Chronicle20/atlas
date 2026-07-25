# CashVegaScroll (← `CUIVega::OnVegaResult`)

- **IDA:** 0x858d7e
- **Atlas file:** `libs/atlas-packet/cash/clientbound/vega_scroll.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `vega result mode byte; single Decode1 routed to START/RESULT/notice arms (gms_v84 CUIVega::OnPacket(0x170), task-130 Task 4b IDA-verified 0x858d7e)` | ✅ |  |

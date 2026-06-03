# Open (← `CUserLocal::OnOpenUI`)

- **IDA:** 0x9e32c0
- **Atlas file:** `libs/atlas-packet/ui/clientbound/open.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `nUIID (window mode byte; dispatched to CWvsContext::UI_Open)` | ✅ |  |


## Manual analysis

v87 vs v95/v83: gate confirmed ✅. `CUserLocal::OnOpenUI` @ 0x9e32c0: reads Decode1(nUIID). Atlas matches.

Ack: misc-audit Phase 3 v87 on 2026-06-03

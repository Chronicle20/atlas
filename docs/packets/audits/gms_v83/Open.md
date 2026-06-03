# Open (← `CUserLocal::OnOpenUI`)

- **IDA:** 0x9600f0
- **Atlas file:** `../../libs/atlas-packet/ui/clientbound/open.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `nUIID (window mode byte; dispatched to CWvsContext::UI_Open)` | ✅ |  |


## Manual analysis

**v83 IDA:** `CUserLocal::OnOpenUI` @ 0x9600f0 — Decode1(nUIID). Matches v95 exactly.

**Gate:** None needed — version-agnostic. Gate confirmed correct (✅).


Ack: misc-audit Phase 3 v83 on 2026-06-03

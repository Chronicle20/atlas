# Disable (← `CUserLocal::OnSetStandAloneMode`)

- **IDA:** 0x95ffa2
- **Atlas file:** `../../libs/atlas-packet/ui/clientbound/disable.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `bStandAlone (enable flag; stored to CWvsContext+0x70)` | ✅ |  |


## Manual analysis

**v83 IDA:** `CUserLocal::OnSetStandAloneMode` @ 0x95ffa2 — Decode1(bStandAlone) only. Matches v95 exactly.

**Gate:** None needed — version-agnostic. Gate confirmed correct (✅).


Ack: misc-audit Phase 3 v83 on 2026-06-03

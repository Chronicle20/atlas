# Disable (← `CUserLocal::OnSetStandAloneMode`)

- **IDA:** 0x9e3172
- **Atlas file:** `libs/atlas-packet/ui/clientbound/disable.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `bStandAlone (enable flag)` | ✅ |  |


## Manual analysis

v87 vs v95/v83: gate confirmed ✅. `CUserLocal::OnSetStandAloneMode` @ 0x9e3172: reads Decode1(bStandAlone). Atlas matches.

Ack: misc-audit Phase 3 v87 on 2026-06-03

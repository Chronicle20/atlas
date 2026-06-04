# ScriptProgress (← `CWvsContext::OnScriptProgressMessage`)

- **IDA:** 0xaa9a6d
- **Atlas file:** `libs/atlas-packet/quest/clientbound/script_progress.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `message (quest script progress string)` | ✅ |  |


## Manual analysis

v87 vs v95/v83: gate confirmed ✅. `OnScriptProgressMessage` @ 0xaa9a6d: reads DecodeStr(message). Atlas matches.

Ack: misc-audit Phase 3 v87 on 2026-06-03

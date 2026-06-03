# ScriptProgress (← `CWvsContext::OnScriptProgressMessage`)

- **IDA:** 0xa13f20
- **Atlas file:** `../../libs/atlas-packet/quest/clientbound/script_progress.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `message (quest script progress string)` | ✅ |  |


## Manual analysis

**v83 IDA:** `CWvsContext::OnScriptProgressMessage` @ 0xa13f20 — DecodeStr(message) only. Matches v95 exactly.

**Gate:** None needed — version-agnostic. Gate confirmed correct (✅).


Ack: misc-audit Phase 3 v83 on 2026-06-03

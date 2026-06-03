# Action (← `CWvsContext::ResignQuest#Action`)

- **IDA:** 0xabeb10
- **Atlas file:** `libs/atlas-packet/quest/serverbound/action.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `action type byte (Encode1 literal 3u @0xabebdd)` | ✅ |  |
| 1 | int16 | int16 `questId uint16` | ✅ |  |


## Manual analysis

v87 vs v95/v83: gate confirmed ✅. `CWvsContext::ResignQuest` @ 0xabeb10: Encode1(literal 3u action type) + Encode2(questId). Atlas matches.

Ack: misc-audit Phase 3 v87 on 2026-06-03

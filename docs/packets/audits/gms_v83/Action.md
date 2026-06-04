# Action (← `CWvsContext::ResignQuest#Action`)

- **IDA:** 0xa26ea7
- **Atlas file:** `../../libs/atlas-packet/quest/serverbound/action.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `action type byte (literal 3u @0xa26f70)` | ✅ |  |
| 1 | int16 | int16 `questId uint16 @0xa26f79` | ✅ |  |


## Manual analysis

**v83 IDA:** `CWvsContext::ResignQuest` @ 0xa26ea7 — Encode1(literal 3u), Encode2(questId). Matches v95 exactly.

**Gate:** None needed — packet is version-agnostic. Gate confirmed correct (✅).


Ack: misc-audit Phase 3 v83 on 2026-06-03

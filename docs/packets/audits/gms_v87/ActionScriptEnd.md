# ActionScriptEnd (← `CQuest::StartQuest#ActionScriptEnd`)

- **IDA:** 0x75bf04
- **Atlas file:** `libs/atlas-packet/quest/serverbound/action_script_end.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `npcId uint32 (Encode4 *(this+16) @0x75c047)` | ✅ |  |
| 1 | int16 | int16 `x int16 (Encode2 v58)` | ✅ |  |
| 2 | int16 | int16 `y int16 (Encode2 v59)` | ✅ |  |


## Manual analysis

v87 vs v95/v83: gate confirmed ✅. `CQuest::StartQuest` @ 0x75bf04 (Encode1 action 5): Encode2(questId) + Encode4(npcId) + Encode2(x) + Encode2(y). Atlas matches.

Ack: misc-audit Phase 3 v87 on 2026-06-03

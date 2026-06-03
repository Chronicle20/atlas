# ActionScriptEnd (← `CQuest::StartQuest#ActionScriptEnd`)

- **IDA:** 0x716fe1
- **Atlas file:** `../../libs/atlas-packet/quest/serverbound/action_script_end.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `npcId uint32 (*(this+16) @0x717503)` | ✅ |  |
| 1 | int16 | int16 `x int16 @0x717524` | ✅ |  |
| 2 | int16 | int16 `y int16 @0x71752f` | ✅ |  |


## Manual analysis

**v83 IDA:** `CQuest::StartQuest` @ 0x716fe1, action=5 branch — Encode1(5)+Encode2(questId)+Encode4(npcId)+Encode2(x)+Encode2(y). Sub-struct fields (npcId, x, y) match v95.

**Gate:** None needed — version-agnostic. Gate confirmed correct (✅).


Ack: misc-audit Phase 3 v83 on 2026-06-03

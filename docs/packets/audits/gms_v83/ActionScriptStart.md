# ActionScriptStart (← `CQuest::StartQuest#ActionScriptStart`)

- **IDA:** 0x716fe1
- **Atlas file:** `../../libs/atlas-packet/quest/serverbound/action_script_start.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `npcId uint32 (*(this+16) @0x7170ae)` | ✅ |  |
| 1 | int16 | int16 `x int16 (ptUserPos @0x7170b9)` | ✅ |  |
| 2 | int16 | int16 `y int16 @0x7170c4` | ✅ |  |


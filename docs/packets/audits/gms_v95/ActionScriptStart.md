# ActionScriptStart (← `CQuest::StartQuest#ActionScriptStart`)

- **IDA:** 0x6b40a0
- **Atlas file:** `../../libs/atlas-packet/quest/serverbound/action_script_start.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `npcId uint32` | ✅ |  |
| 1 | int16 | int16 `x int16` | ✅ |  |
| 2 | int16 | int16 `y int16` | ✅ |  |


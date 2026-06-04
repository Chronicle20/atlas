# ActionScriptEnd (← `CQuest::StartQuest#ActionScriptEnd`)

- **IDA:** 0x77d065
- **Atlas file:** `../../libs/atlas-packet/quest/serverbound/action_script_end.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `npcId uint32` | ✅ |  |
| 1 | int16 | int16 `x int16 (bLoopback, conditional on !IsAutoAlertQuest)` | ✅ |  |
| 2 | int16 | int16 `y int16 (conditional on !IsAutoAlertQuest)` | ✅ |  |


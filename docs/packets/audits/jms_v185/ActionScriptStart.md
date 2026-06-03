# ActionScriptStart (← `CQuest::StartQuest#ActionScriptStart`)

- **IDA:** 0x77d065
- **Atlas file:** `libs/atlas-packet/quest/serverbound/action_script_start.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `npcId uint32` | ✅ |  |
| 1 | int16 | int16 `x int16 (bLoopback, conditional on !IsAutoAlertQuest)` | ✅ |  |
| 2 | int16 | int16 `y int16 (v59[0], conditional on !IsAutoAlertQuest)` | ✅ |  |


## Manual analysis

JMS v185 `CQuest::StartQuest` (@ 0x77d065) action=4 branch: sub-struct matches GMS v95 — `Encode4(npcId) + Encode2(x) + Encode2(y)` conditional on `!IsAutoAlertQuest`. No nItemPos field in JMS for this action.

**JMS vs GMS: gate confirmed ✅.** No gate change needed.

Ack: misc-audit Phase 3 JMS185 on 2026-06-03

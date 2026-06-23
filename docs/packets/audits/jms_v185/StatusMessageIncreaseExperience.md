# StatusMessageIncreaseExperience (← `CWvsContext::OnMessage#IncreaseExperience`)

- **IDA:** 0xb08a97
- **Atlas file:** `libs/atlas-packet/character/clientbound/status_message.go`
- **Variant:** JMS/v185
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `outer mode (INCREASE_EXPERIENCE)` | ✅ |  |
| 1 | byte | byte `white bool` | ✅ |  |
| 2 | int32 | int32 `amount` | ✅ |  |
| 3 | byte | byte `inChat bool` | ✅ |  |
| 4 | int32 | int32 `monsterBookBonus` | ✅ |  |
| 5 | byte | byte `mobEventBonusPercentage` | ✅ |  |
| 6 | byte | byte `partyBonusPercentage` | ✅ |  |
| 7 | int32 | int32 `weddingBonusEXP` | ✅ |  |
| 8 | byte | byte `playTimeHour (only if mobEventBonusPercentage>0)` | ✅ |  |
| 9 | byte | byte `questBonusRate (only if inChat)` | ✅ |  |
| 10 | byte | byte `questBonusRemainCount (only if inChat && questBonusRate>0)` | ✅ |  |
| 11 | byte | byte `partyBonusEventRate` | ✅ |  |
| 12 | int32 | int32 `partyBonusExp` | ✅ |  |
| 13 | int32 | int32 `itemBonusEXP` | ✅ |  |
| 14 | int32 | int32 `premiumIPExp` | ✅ |  |
| 15 | int32 | int32 `rainbowWeekEventEXP` | ✅ |  |


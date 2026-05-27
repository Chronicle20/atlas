# CharacterExpression (← `CUser::OnEmotion`)

- **IDA:** 0x9f636b
- **Atlas file:** `libs/atlas-packet/character/clientbound/expression.go`
- **Variant:** JMS/v185
- **Branch depth:** 3
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `characterId (consumed by CUserPool::OnUserRemotePacket dispatcher)` | ✅ |  |
| 1 | int32 | int32 `nEmotion (expression/emote ID)` | ✅ |  |
| 2 | int32 | int32 `tDuration (display duration in ms — JMS has duration but NOT byItemOption)` | ✅ |  |


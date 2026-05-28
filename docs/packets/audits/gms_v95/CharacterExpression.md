# CharacterExpression (← `CUser::OnEmotion`)

- **IDA:** 0x8e0150
- **Atlas file:** `../../libs/atlas-packet/character/clientbound/expression.go`
- **Variant:** GMS/v95
- **Branch depth:** 3
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `characterId (consumed by CUserPool::OnUserRemotePacket dispatcher, case 219 = 0xDB)` | ✅ |  |
| 1 | int32 | int32 `nEmotion (expression/emote ID)` | ✅ |  |
| 2 | int32 | int32 `nEmotionDuration (duration in ms)` | ✅ |  |
| 3 | byte | byte `m_bEmotionByItemOption (item-option emotion flag)` | ✅ |  |


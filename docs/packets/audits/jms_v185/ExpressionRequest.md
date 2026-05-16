# ExpressionRequest (← `CWvsContext::SendEmotionChange`)

- **IDA:** 0xb0b8be
- **Atlas file:** `libs/atlas-packet/character/serverbound/expression.go`
- **Variant:** JMS/v185
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `characterId (local user's charId from CUserLocal — NOT emotionId)` | ✅ |  |


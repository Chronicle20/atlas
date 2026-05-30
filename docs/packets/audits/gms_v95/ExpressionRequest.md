# ExpressionRequest (← `CWvsContext::SendEmotionChange`)

- **IDA:** 0x9f9320
- **Atlas file:** `libs/atlas-packet/character/serverbound/expression.go`
- **Variant:** GMS/v95
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `nEmotion (emotion/expression ID; validated <= 0x17)` | ✅ |  |
| 1 | int32 | int32 `nDuration (display duration in ms)` | ✅ |  |
| 2 | byte | byte `bByItemOption (1 = triggered by item option, 0 = normal)` | ✅ |  |


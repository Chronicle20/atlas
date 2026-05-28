# ExpressionRequest (← `CWvsContext::SendEmotionChange`)

- **IDA:** 0xabbfbb
- **Atlas file:** `../../libs/atlas-packet/character/serverbound/expression.go`
- **Variant:** GMS/v87
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `nEmotion (emotion/expression ID; validated <= 0x16 in v87) — only field. duration and byItemOption absent in v87; added in v95. Gate GMS>83\|\|JMS is wrong for v87.` | ✅ |  |


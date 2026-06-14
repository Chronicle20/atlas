# ExpressionRequest (← `CWvsContext::SendEmotionChange`)

- **IDA:** 0xa24470
- **Atlas file:** `libs/atlas-packet/character/serverbound/expression.go`
- **Variant:** GMS/v83
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `nEmotion (emotion/expression ID; validated <= 0x16 in v83 vs 0x17 in v95) — NOTE: v83 does NOT encode nDuration or bByItemOption` | ✅ |  |


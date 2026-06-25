# CharacterExpression (← `CUser::OnEmotion`)

- **IDA:** 0x9724f9
- **Atlas file:** `libs/atlas-packet/character/clientbound/expression.go`
- **Variant:** GMS/v83
- **Branch depth:** 3
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `characterId — read by CUserPool::OnUserRemotePacket dispatcher` | ✅ |  |
| 1 | int32 | int32 `nEmotion (expression/emote ID) — only field; no duration, no byItemOption in v83` | ✅ |  |


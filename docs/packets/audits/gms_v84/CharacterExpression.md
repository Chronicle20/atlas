# CharacterExpression (← `CUser::OnEmotion`)

- **IDA:** 0x9b2518
- **Atlas file:** `libs/atlas-packet/character/clientbound/expression.go`
- **Variant:** GMS/v84
- **Branch depth:** 3
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `characterId — read by CUserPool::OnUserRemotePacket dispatcher @0x9b2522` | ✅ |  |
| 1 | int32 | int32 `nEmotion (expression/emote ID) @0x9b2598 — only field; no duration, no byItemOption in v84` | ✅ |  |


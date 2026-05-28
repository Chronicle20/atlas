# ChatGeneralChat (← `CUser::OnChat`)

- **IDA:** 0x9f5c74
- **Atlas file:** `libs/atlas-packet/chat/clientbound/general.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `characterId — consumed by CUserPool dispatcher before OnChat` | ✅ |  |
| 1 | byte | byte `isGM flag (mode byte: 0=normal, 1=NPC-name)` | ✅ |  |
| 2 | string | string `msg text` | ✅ |  |
| 3 | byte | byte `bOnlyBalloon flag` | ✅ |  |


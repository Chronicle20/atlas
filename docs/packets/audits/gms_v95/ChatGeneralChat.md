# ChatGeneralChat (← `CUser::OnChat`)

- **IDA:** 0x8e86c0
- **Atlas file:** `../../libs/atlas-packet/chat/clientbound/general.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `isGM / chat type byte (0=normal player, 1=npc-name; consumed as gm flag in atlas)` | ❌ | width mismatch |
| 1 | byte | string `chat message text` | ❌ | width mismatch |
| 2 | string | byte `bOnlyBalloon (show flag: 0=show in chat log, 1=balloon only)` | ❌ | width mismatch |
| 3 | byte | byte `` | ❌ | atlas: extra — client never reads this field |


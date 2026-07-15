# StatusMessageCompleteQuestRecord (← `CWvsContext::OnMessage#CompleteQuestRecord`)

- **IDA:** 0x843bd8
- **Atlas file:** `libs/atlas-packet/character/clientbound/status_message.go`
- **Variant:** GMS/v61
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | int16 `questId @0x919627` | ❌ | width mismatch |
| 1 | int16 | byte `subtype 2 @0x919638` | ❌ | width mismatch |
| 2 | byte | bytes `completedAt FILETIME int64 (8) @0x919659` | ✅ |  |
| 3 | int64 | byte `` | ✅ | absorbed by trailing opaque buffer |


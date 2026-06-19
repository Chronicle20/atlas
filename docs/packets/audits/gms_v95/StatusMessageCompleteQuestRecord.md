# StatusMessageCompleteQuestRecord (← `CWvsContext::OnMessage#CompleteQuestRecord`)

- **IDA:** 0xa03920
- **Atlas file:** `libs/atlas-packet/character/clientbound/status_message.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `outer mode (1 = quest record)` | ✅ |  |
| 1 | int16 | int16 `questId` | ✅ |  |
| 2 | byte | byte `inner disc byte = 2 (complete)` | ✅ |  |
| 3 | int64 | int64 `completedAt FILETIME (8-byte int64)` | ✅ |  |


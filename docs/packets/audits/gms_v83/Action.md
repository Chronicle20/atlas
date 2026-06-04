# Action (← `CWvsContext::ResignQuest#Action`)

- **IDA:** 0xa26ea7
- **Atlas file:** `../../libs/atlas-packet/quest/serverbound/action.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `action type byte (literal 3u @0xa26f70)` | ✅ |  |
| 1 | int16 | int16 `questId uint16 @0xa26f79` | ✅ |  |


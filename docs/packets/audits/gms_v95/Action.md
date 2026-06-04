# Action (← `CWvsContext::ResignQuest#Action`)

- **IDA:** 0x9f3cf0
- **Atlas file:** `../../libs/atlas-packet/quest/serverbound/action.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `action type byte` | ✅ |  |
| 1 | int16 | int16 `questId uint16` | ✅ |  |


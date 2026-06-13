# Action (← `CWvsContext::ResignQuest#Action`)

- **IDA:** 0xa7265e
- **Atlas file:** `libs/atlas-packet/quest/serverbound/action.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 1 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |


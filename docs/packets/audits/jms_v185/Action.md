# Action (← `CWvsContext::ResignQuest#Action`)

- **IDA:** 0xb0e6e9
- **Atlas file:** `libs/atlas-packet/quest/serverbound/action.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `action type byte (literal 3u for resign; opcode 0x66)` | ✅ |  |
| 1 | int16 | int16 `questId uint16` | ✅ |  |


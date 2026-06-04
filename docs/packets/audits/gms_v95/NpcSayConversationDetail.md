# NpcSayConversationDetail (← `CScriptMan::OnSay#Say`)

- **IDA:** 0x6dc110
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/conversation.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `message text` | ✅ |  |
| 1 | byte | byte `bPrev (Previous button)` | ✅ |  |
| 2 | byte | byte `bNext (Next button)` | ✅ |  |


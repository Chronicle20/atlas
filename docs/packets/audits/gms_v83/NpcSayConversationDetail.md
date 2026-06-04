# NpcSayConversationDetail (← `CScriptMan::OnSay#Say`)

- **IDA:** 0x7467ab
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/conversation.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `message text` | ✅ |  |
| 1 | byte | byte `bPrev / v26 (Previous button)` | ✅ |  |
| 2 | byte | byte `bNext / v7 (Next button)` | ✅ |  |


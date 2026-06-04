# NpcSayImageConversationDetail (← `CScriptMan::OnSayImage#SayImage`)

- **IDA:** 0x961275
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/conversation.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `image count (v9)` | ✅ |  |
| 1 | string | string `image path -- loop body (count iterations; analyzer flattens)` | ✅ |  |


# NpcSayImageConversationDetail (← `CScriptMan::OnSayImage#SayImage`)

- **IDA:** 0x6dc310
- **Atlas file:** `libs/atlas-packet/npc/clientbound/conversation.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `image count (nCount)` | ✅ |  |
| 1 | string | string `image path -- loop body (count iterations; analyzer flattens)` | ✅ |  |


Ack: world-audit sub-phase 2f on 2026-05-28

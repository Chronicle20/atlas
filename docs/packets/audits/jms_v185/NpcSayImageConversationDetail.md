# NpcSayImageConversationDetail (← `CScriptMan::OnSayImage#SayImage`)

- **IDA:** 0x7b7496
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/conversation.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `image count (nCount @0x7b7524)` | ✅ |  |
| 1 | string | string `image path -- loop body (count iterations; analyzer flattens @0x7b753a)` | ✅ |  |


Ack: world-audit Phase 3 JMS185 npc domain on 2026-05-28

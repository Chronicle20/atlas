# NpcAskMemberShopAvatarConversationDetail (← `CScriptMan::OnAskMembershopAvatar#AskMemberShopAvatar`)

- **IDA:** 0x74730b
- **Atlas file:** `libs/atlas-packet/npc/clientbound/conversation.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `message text (v25)` | ✅ |  |
| 1 | byte | byte `candidate count (v5)` | ✅ |  |
| 2 | int32 | int32 `candidate style id -- loop body (count iterations; analyzer flattens)` | ✅ |  |


Ack: world-audit Phase 3 v83 on 2026-05-28

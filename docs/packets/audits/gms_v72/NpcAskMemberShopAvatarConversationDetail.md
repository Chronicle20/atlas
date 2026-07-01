# NpcAskMemberShopAvatarConversationDetail (← `CScriptMan::OnAskMembershopAvatar#AskMemberShopAvatar`)

- **IDA:** 
- **Atlas file:** `libs/atlas-packet/npc/clientbound/conversation.go`
- **Variant:** GMS/v72
- **Branch depth:** 2
- **Verdict:** ⚠️

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | unresolved `function not found in IDB` | 🚫 | IDA read-order unresolved: function not found in IDB |
| 1 | byte | byte `` | ⚠️ | atlas: trailing padding byte — client stops reading (harmless over-write) |


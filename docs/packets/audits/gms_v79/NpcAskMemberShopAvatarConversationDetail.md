# NpcAskMemberShopAvatarConversationDetail (← `CScriptMan::OnAskMembershopAvatar#AskMemberShopAvatar`)

- **IDA:** 0x6c8bc8
- **Atlas file:** `libs/atlas-packet/npc/clientbound/conversation.go`
- **Variant:** GMS/v79
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `message text` | ✅ |  |
| 1 | byte | byte `count (cash-avatar SN entries; 0 for the legacy range, Atlas drives no candidates)` | ✅ |  |


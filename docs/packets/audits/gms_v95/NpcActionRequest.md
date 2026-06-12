# NpcActionRequest (← `CNpc::GenerateMovePath`)

- **IDA:** 0x671590
- **Atlas file:** `../../libs/atlas-packet/npc/serverbound/action.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `npcId (m_dwNpcId; atlas ActionRequest.objectId)` | ✅ |  |
| 1 | byte | byte `nAction (atlas unk)` | ✅ |  |
| 2 | byte | byte `nChatIdx (atlas unk2)` | ✅ |  |
| 3 | bytes | bytes `movement body (CMovePath::Flush) -- gated on m_pTemplate->bMove; atlas optional WriteByteArray(movement)` | ✅ |  |


# InteractionOperationInvite (← `CField::SendInviteTradingRoomMsg`)

- **IDA:** 0x52e9e0
- **Atlas file:** `libs/atlas-packet/interaction/serverbound/operation_invite.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `targetCharacterId (m_dwCharacterId)` | ✅ |  |


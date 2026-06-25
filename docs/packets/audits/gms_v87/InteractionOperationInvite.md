# InteractionOperationInvite (← `CField::SendInviteTradingRoomMsg`)

- **IDA:** 0x556cfe
- **Atlas file:** `libs/atlas-packet/interaction/serverbound/operation_invite.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `targetCharacterId (m_dwCharacterId)` | ✅ |  |


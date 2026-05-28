# PartyOperationInvite (← `CField::SendJoinPartyMsg`)

- **IDA:** 0x56cce9
- **Atlas file:** `../../libs/atlas-packet/party/serverbound/operation_invite.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | byte `sub-op = 4 (INVITE)` | ❌ | width mismatch |
| 1 | byte | string `target character name` | ❌ | atlas: short — missing trailing field |


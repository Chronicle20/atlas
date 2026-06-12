# PartyOperationInvite (← `CField::SendJoinPartyMsg`)

- **IDA:** 0x534310
- **Atlas file:** `../../libs/atlas-packet/party/serverbound/operation_invite.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `op=4 (INVITE_PARTY_MEMBER)` | ✅ |  |
| 1 | string | string `v22 = target character name to invite` | ✅ |  |


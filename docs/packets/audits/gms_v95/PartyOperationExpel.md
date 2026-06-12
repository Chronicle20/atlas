# PartyOperationExpel (← `CField::SendKickPartyMsg`)

- **IDA:** 0x530140
- **Atlas file:** `../../libs/atlas-packet/party/serverbound/operation_expel.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `op=5 (EXPEL_PARTY_MEMBER)` | ✅ |  |
| 1 | int32 | int32 `PartyMemberByName = targetCharacterId` | ✅ |  |


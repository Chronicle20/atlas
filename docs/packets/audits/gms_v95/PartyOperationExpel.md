# PartyOperationExpel (← `CField::SendKickPartyMsg`)

- **IDA:** 0x530140
- **Atlas file:** `../../libs/atlas-packet/party/serverbound/operation_expel.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `op=5 (EXPEL_PARTY_MEMBER)` | ❌ | width mismatch |
| 1 | byte | int32 `PartyMemberByName = targetCharacterId` | ❌ | atlas: short — missing trailing field |


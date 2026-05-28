# PartyOperationJoin (← `CField::SendJoinPartyMsg#OperationJoin`)

- **IDA:** 0x534310
- **Atlas file:** `../../libs/atlas-packet/party/serverbound/operation_join.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `partyId — the party to join` | ✅ |  |


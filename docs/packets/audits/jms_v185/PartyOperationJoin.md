# PartyOperationJoin (← `CField::SendJoinPartyMsg#OperationJoin`)

- **IDA:** 0x56cce9
- **Atlas file:** `../../libs/atlas-packet/party/serverbound/operation_join.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `sub-op = 0 (JOIN_RESPONSE)` | ✅ |  |
| 1 | int32 | int32 `partyId` | ✅ |  |
| 2 | byte | byte `accepted flag (1=accept, 0=decline)` | ❌ | atlas: short — missing trailing field |


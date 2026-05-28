# PartyOperationJoin (← `CField::SendJoinPartyMsg#OperationJoin`)

- **IDA:** 0x56cce9
- **Atlas file:** `libs/atlas-packet/party/serverbound/operation_join.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `sub-op = 0 (JOIN_RESPONSE)` | ❌ | width mismatch |
| 1 | byte | int32 `partyId` | ❌ | atlas: short — missing trailing field |
| 2 | byte | byte `accepted flag (1=accept, 0=decline)` | ❌ | atlas: short — missing trailing field |


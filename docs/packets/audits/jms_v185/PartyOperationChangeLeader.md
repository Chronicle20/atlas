# PartyOperationChangeLeader (← `CField::SendChangePartyBossMsg`)

- **IDA:** 0x56d0cc
- **Atlas file:** `../../libs/atlas-packet/party/serverbound/operation_change_leader.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `sub-op = 5 (CHANGE_LEADER)` | ❌ | width mismatch |
| 1 | byte | string `target character name` | ❌ | atlas: short — missing trailing field |


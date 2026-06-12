# PartyOperationChangeLeader (← `CField::SendChangePartyBossMsg`)

- **IDA:** 0x56d0cc
- **Atlas file:** `../../libs/atlas-packet/party/serverbound/operation_change_leader.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `sub-op=6` | ✅ |  |
| 1 | int32 | int32 `charId` | ✅ |  |


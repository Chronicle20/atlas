# PartyOperationChangeLeader (← `CField::SendChangePartyBossMsg`)

- **IDA:** 0x530370
- **Atlas file:** `../../libs/atlas-packet/party/serverbound/operation_change_leader.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `op=6 (CHANGE_PARTY_BOSS)` | ❌ | width mismatch |
| 1 | byte | int32 `PartyMemberByName = targetCharacterId` | ❌ | atlas: short — missing trailing field |


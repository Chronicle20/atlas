# PartyChangeLeader (← `CWvsContext::OnPartyResult#ChangeLeader`)

- **IDA:** 0xa1169e
- **Atlas file:** `../../libs/atlas-packet/party/clientbound/change_leader.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | int32 `v66 = newLeaderId — stored as dwPartyBossCharacterID` | ❌ | width mismatch |
| 1 | int32 | byte `v67 = disconnected flag (0=normal transfer, 1=due to disconnect)` | ❌ | width mismatch |
| 2 | byte | byte `` | ❌ | atlas: extra — client never reads this field |


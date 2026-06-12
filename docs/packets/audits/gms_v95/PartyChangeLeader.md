# PartyChangeLeader (← `CWvsContext::OnPartyResult#ChangeLeader`)

- **IDA:** 0xa1169e
- **Atlas file:** `../../libs/atlas-packet/party/clientbound/change_leader.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode` | ✅ |  |
| 1 | int32 | int32 `targetCharacterId` | ✅ |  |
| 2 | byte | byte `disconnected` | ✅ |  |


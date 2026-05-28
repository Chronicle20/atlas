# PartyJoin (← `CWvsContext::OnPartyResult#Join`)

- **IDA:** 0xa3e31c
- **Atlas file:** `../../libs/atlas-packet/party/clientbound/join.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (15)` | ✅ |  |
| 1 | int32 | int32 `partyId` | ✅ |  |
| 2 | string | string `joinedMemberName` | ✅ |  |
| 3 | byte | int32 `PARTYDATA ids[6] × 6` | ❌ | atlas: short — missing trailing field |
| 4 | byte | byte `PARTYDATA names[6] (padded 13 × 6)` | ❌ | atlas: short — missing trailing field |
| 5 | byte | int32 `PARTYDATA jobs/levels/channels × 6 each` | ❌ | atlas: short — missing trailing field |
| 6 | byte | int32 `PARTYDATA leaderId` | ❌ | atlas: short — missing trailing field |
| 7 | byte | int32 `PARTYDATA maps[6]` | ❌ | atlas: short — missing trailing field |
| 8 | byte | int32 `PARTYDATA portals[6] × 4 ints (v83 no skillId, no PQ)` | ❌ | atlas: short — missing trailing field |


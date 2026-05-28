# PartyDisband (← `CWvsContext::OnPartyResult#Disband`)

- **IDA:** 0xa3e31c
- **Atlas file:** `libs/atlas-packet/party/clientbound/disband.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (12)` | ✅ |  |
| 1 | int32 | int32 `partyId` | ✅ |  |
| 2 | int32 | int32 `targetId` | ✅ |  |
| 3 | byte | byte `forced (1=expel, 0=disband/leave)` | ✅ |  |
| 4 | int32 | string `targetName` | ❌ | width mismatch |
| 5 | byte | int32 `PARTYDATA ids[6] × 6` | ❌ | atlas: short — missing trailing field |
| 6 | byte | byte `PARTYDATA names[6] × 6 (padded 13)` | ❌ | atlas: short — missing trailing field |
| 7 | byte | int32 `PARTYDATA jobs[6] × 6` | ❌ | atlas: short — missing trailing field |
| 8 | byte | int32 `PARTYDATA levels[6] × 6` | ❌ | atlas: short — missing trailing field |
| 9 | byte | int32 `PARTYDATA channels[6] × 6` | ❌ | atlas: short — missing trailing field |
| 10 | byte | int32 `PARTYDATA leaderId` | ❌ | atlas: short — missing trailing field |
| 11 | byte | int32 `PARTYDATA maps[6] × 6` | ❌ | atlas: short — missing trailing field |
| 12 | byte | int32 `PARTYDATA portals[6] × 4 ints (v83: no m_nSKillID, no PQ fields)` | ❌ | atlas: short — missing trailing field |


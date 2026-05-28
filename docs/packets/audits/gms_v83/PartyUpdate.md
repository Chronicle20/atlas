# PartyUpdate (← `CWvsContext::OnPartyResult#Update`)

- **IDA:** 0xa3e31c
- **Atlas file:** `libs/atlas-packet/party/clientbound/update.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (7 or 34)` | ✅ |  |
| 1 | int32 | int32 `partyId` | ✅ |  |
| 2 | byte | int32 `PARTYDATA ids[0..5]` | ❌ | atlas: short — missing trailing field |
| 3 | byte | string `PARTYDATA names[0..5] (padded 13 each)` | ❌ | atlas: short — missing trailing field |
| 4 | byte | int32 `PARTYDATA jobs[0..5]` | ❌ | atlas: short — missing trailing field |
| 5 | byte | int32 `PARTYDATA levels[0..5]` | ❌ | atlas: short — missing trailing field |
| 6 | byte | int32 `PARTYDATA channels[0..5]` | ❌ | atlas: short — missing trailing field |
| 7 | byte | int32 `PARTYDATA leaderId` | ❌ | atlas: short — missing trailing field |
| 8 | byte | int32 `PARTYDATA maps[0..5]` | ❌ | atlas: short — missing trailing field |
| 9 | byte | int32 `PARTYDATA portals[6] × 4 ints: townId+fieldId+ptX+ptY (v83 no m_nSKillID, no PQ arrays)` | ❌ | atlas: short — missing trailing field |


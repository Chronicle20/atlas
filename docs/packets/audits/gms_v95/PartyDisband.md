# PartyDisband (← `CWvsContext::OnPartyResult#Disband`)

- **IDA:** 0xa11085
- **Atlas file:** `libs/atlas-packet/party/clientbound/disband.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | int32 `m_nPartyID = partyId (stored, zeroed after)` | ❌ | width mismatch |
| 1 | int32 | int32 `v37 = targetId` | ✅ |  |
| 2 | int32 | byte `isForced flag — 0=leave/disband, 1=expelled` | ❌ | width mismatch |
| 3 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 4 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |


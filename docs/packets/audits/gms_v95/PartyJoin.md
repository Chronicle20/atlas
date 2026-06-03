# PartyJoin (← `CWvsContext::OnPartyResult#Join`)

- **IDA:** 0xa11405
- **Atlas file:** `libs/atlas-packet/party/clientbound/join.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | int32 `m_nPartyID = partyId` | ❌ | width mismatch |
| 1 | int32 | string `sCharacterName = joining member name` | ❌ | width mismatch |
| 2 | string | bytes `PARTYDATA::Decode(0x17A=378 bytes) — includes PARTYMEMBER+adwFieldID+aTownPortal+aPQReward fields` | ❌ | width mismatch |


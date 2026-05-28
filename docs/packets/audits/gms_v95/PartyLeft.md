# PartyLeft (← `CWvsContext::OnPartyResult#Left`)

- **IDA:** 0xa11085
- **Atlas file:** `../../libs/atlas-packet/party/clientbound/left.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | int32 `m_nPartyID = partyId` | ❌ | width mismatch |
| 1 | int32 | int32 `v37 = targetId` | ✅ |  |
| 2 | int32 | byte `isForced branch discriminator (1 = member leaving with PARTYDATA)` | ❌ | width mismatch |
| 3 | byte | byte `v38 = forced flag (0=voluntary leave, 1=expelled by leader)` | ✅ |  |
| 4 | byte | string `sCharacterName = leaving member name` | ❌ | width mismatch |
| 5 | string | bytes `PARTYDATA::Decode(0x17A=378 bytes) — full party state after member left` | ❌ | width mismatch |


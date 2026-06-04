# PartyUpdate (← `CWvsContext::OnPartyResult#Update`)

- **IDA:** 0xa10ea1
- **Atlas file:** `../../libs/atlas-packet/party/clientbound/update.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | int32 `m_nPartyID = partyId` | ❌ | width mismatch |
| 1 | int32 | bytes `PARTYDATA::Decode(0x17A=378 bytes) — full PARTYDATA struct including PQ reward fields` | ✅ |  |


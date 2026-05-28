# PartyCreated (← `CWvsContext::OnPartyResult#Created`)

- **IDA:** 0xa10efc
- **Atlas file:** `../../libs/atlas-packet/party/clientbound/created.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | int32 `partyId — stored as m_nPartyID` | ❌ | width mismatch |
| 1 | int32 | int16 `townPortalFromId (map4 in atlas Created)` | ❌ | width mismatch |
| 2 | int32 | int16 `townPortalToId (map4 in atlas Created)` | ❌ | width mismatch |
| 3 | int32 | int16 `door x (short)` | ❌ | width mismatch |
| 4 | int16 | int16 `door y (short)` | ✅ |  |
| 5 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |


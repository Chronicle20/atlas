# PartyLeft (← `CWvsContext::OnPartyResult#Left`)

- **IDA:** 0xad697a
- **Atlas file:** `../../libs/atlas-packet/party/clientbound/left.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (12)` | ✅ |  |
| 1 | int32 | int32 `partyLeaderId` | ✅ |  |
| 2 | int32 | int32 `expelledCharId` | ✅ |  |
| 3 | byte | byte `discharge flag` | ✅ |  |
| 4 | byte | bytes `PARTYDATA (298 bytes in v87)` | ❌ | width mismatch |
| 5 | string | byte `` | ❌ | atlas: extra — client never reads this field |


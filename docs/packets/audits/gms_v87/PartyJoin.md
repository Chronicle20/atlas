# PartyJoin (← `CWvsContext::OnPartyResult#Join`)

- **IDA:** 0xad697a
- **Atlas file:** `../../libs/atlas-packet/party/clientbound/join.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (15)` | ✅ |  |
| 1 | int32 | int32 `partyLeaderId` | ✅ |  |
| 2 | string | string `inviteeName` | ✅ |  |
| 3 | byte | bytes `PARTYDATA (298 bytes in v87 — same as v83; v95+ is 378)` | ❌ | atlas: short — missing trailing field |


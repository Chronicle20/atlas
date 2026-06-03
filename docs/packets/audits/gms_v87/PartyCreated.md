# PartyCreated (← `CWvsContext::OnPartyResult#Created`)

- **IDA:** 0xad697a
- **Atlas file:** `../../libs/atlas-packet/party/clientbound/created.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (4)` | ✅ |  |
| 1 | int32 | int32 `partyId (v151)` | ✅ |  |
| 2 | int32 | string `inviterName (v152)` | ❌ | width mismatch |
| 3 | int32 | int32 `originatorJobId (v146) — present in v87 (NOT v83)` | ✅ |  |
| 4 | int16 | int32 `originatorLevel (v141) — present in v87 (NOT v83)` | ❌ | width mismatch |
| 5 | int16 | byte `autoJoinFlag (*v142)` | ❌ | width mismatch |


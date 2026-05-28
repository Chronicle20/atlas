# PartyLeft (← `CWvsContext::OnPartyResult#Left`)

- **IDA:** 0xa3e31c
- **Atlas file:** `libs/atlas-packet/party/clientbound/left.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (12)` | ✅ |  |
| 1 | int32 | int32 `partyId` | ✅ |  |
| 2 | int32 | int32 `targetId` | ✅ |  |
| 3 | byte | byte `forced (1=expel/forced, 0=leave/disband)` | ✅ |  |
| 4 | byte | string `targetName (via ZXString assign)` | ❌ | width mismatch |
| 5 | string | int32 `PARTYDATA (298 bytes in v83)` | ❌ | width mismatch |


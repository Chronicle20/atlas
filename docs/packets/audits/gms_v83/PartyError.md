# PartyError (← `CWvsContext::OnPartyResult#Error`)

- **IDA:** 0xa3e31c
- **Atlas file:** `../../libs/atlas-packet/party/clientbound/error.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (error code)` | ✅ |  |
| 1 | string | byte `` | ❌ | atlas: extra — client never reads this field |


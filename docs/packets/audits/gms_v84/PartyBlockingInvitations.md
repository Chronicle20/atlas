# PartyBlockingInvitations (← `CWvsContext::OnPartyResult#BlockingInvitations`)

- **IDA:** 0xa89cf3
- **Atlas file:** `libs/atlas-packet/party/clientbound/error.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (21)` | ✅ |  |
| 1 | string | string `name` | ✅ |  |


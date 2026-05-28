# PartyInviteReject (← `CWvsContext::OnPartyResult#InviteReject`)

- **IDA:** 0xa3e31c
- **Atlas file:** `libs/atlas-packet/party/serverbound/invite_reject.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (21/22/23)` | ✅ |  |
| 1 | string | string `targetName (v152)` | ✅ |  |


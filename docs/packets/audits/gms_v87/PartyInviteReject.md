# PartyInviteReject (← `CWvsContext::OnPartyResult#InviteReject`)

- **IDA:** 0xad697a
- **Atlas file:** `../../libs/atlas-packet/party/serverbound/invite_reject.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (22)` | ✅ |  |
| 1 | string | byte `` | ❌ | atlas: extra — client never reads this field |


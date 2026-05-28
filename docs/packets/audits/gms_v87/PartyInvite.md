# PartyInvite (← `CWvsContext::OnPartyResult#Invite`)

- **IDA:** 0xad697a
- **Atlas file:** `../../libs/atlas-packet/party/clientbound/invite.go`
- **Variant:** GMS/v87
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (4)` | ✅ |  |
| 1 | int32 | int32 `partyId` | ✅ |  |
| 2 | string | string `inviterName` | ✅ |  |
| 3 | int32 | int32 `originatorJobId — present in v87` | ✅ |  |
| 4 | int32 | int32 `originatorLevel — present in v87` | ✅ |  |
| 5 | byte | byte `autoJoinFlag` | ✅ |  |


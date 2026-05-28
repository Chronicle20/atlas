# BuddyInvite (← `CWvsContext::OnFriendResult#Invite`)

- **IDA:** 0xb2a873
- **Atlas file:** `libs/atlas-packet/buddy/clientbound/invite.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** 🔍

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte = 9 (Invite)` | ✅ |  |
| 1 | int32 | int32 `dwFriendID (inviter's character id)` | ✅ |  |
| 2 | string | string `inviter name` | ✅ |  |
| 3 | byte | int32 `jobId (inviter)` | 🔍 | sub-struct: b — see _substruct/ |
| 4 | byte | int32 `level (inviter)` | ❌ | width mismatch |


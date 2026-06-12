# BuddyInvite (← `CWvsContext::OnFriendResult#Invite`)

- **IDA:** 0xb2a873
- **Atlas file:** `../../libs/atlas-packet/buddy/clientbound/invite.go`
- **Variant:** JMS/v185
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte = 9 (Invite)` | ✅ |  |
| 1 | int32 | int32 `dwFriendID (inviter's character id)` | ✅ |  |
| 2 | string | string `inviter name` | ✅ |  |
| 3 | int32 | int32 `jobId (inviter)` | ✅ |  |
| 4 | int32 | int32 `level (inviter)` | ✅ |  |
| 5 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 6 | bytes | byte `` | ❌ | atlas: extra — client never reads this field |
| 7 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 8 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 9 | bytes | byte `` | ❌ | atlas: extra — client never reads this field |
| 10 | byte | byte `` | ❌ | atlas: extra — client never reads this field |


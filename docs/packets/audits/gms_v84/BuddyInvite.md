# BuddyInvite (← `CWvsContext::OnFriendResult#Invite`)

- **IDA:** 0xa8ada2
- **Atlas file:** `libs/atlas-packet/buddy/clientbound/invite.go`
- **Variant:** GMS/v84
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ✅ |  |
| 1 | int32 | int32 `` | ✅ |  |
| 2 | string | string `` | ✅ |  |
| 3 | int32 | int32 `` | ✅ |  |
| 4 | int32 | int32 `` | ✅ |  |
| 5 | int32 | byte `` | ❌ | width mismatch |
| 6 | bytes | byte `` | ✅ |  |
| 7 | byte | string `` | ❌ | width mismatch |
| 8 | int32 | int32 `` | ✅ |  |
| 9 | bytes | byte `` | ✅ |  |
| 10 | byte | int32 `` | ❌ | width mismatch |
| 11 | byte | byte `` | ❌ | atlas: short — missing trailing field |


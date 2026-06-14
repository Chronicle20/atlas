# BuddyChannelChange (← `CWvsContext::OnFriendResult#ChannelChange`)

- **IDA:** 0xa8ada2
- **Atlas file:** `libs/atlas-packet/buddy/clientbound/channel_change.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ✅ |  |
| 1 | int32 | int32 `` | ✅ |  |
| 2 | byte | string `` | ❌ | width mismatch |
| 3 | int32 | int32 `` | ✅ |  |
| 4 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 5 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 6 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 7 | byte | string `` | ❌ | atlas: short — missing trailing field |
| 8 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 9 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 10 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 11 | byte | byte `` | ❌ | atlas: short — missing trailing field |


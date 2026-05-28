# BuddyError (← `CWvsContext::OnFriendResult#Error`)

- **IDA:** 0xa3f2e8
- **Atlas file:** `libs/atlas-packet/buddy/clientbound/error.go`
- **Variant:** GMS/v83
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (error code)` | ✅ |  |
| 1 | byte | byte `` | ❌ | atlas: extra — client never reads this field |


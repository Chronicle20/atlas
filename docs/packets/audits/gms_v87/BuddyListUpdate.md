# BuddyListUpdate (← `CWvsContext::OnFriendResult#ListUpdate`)

- **IDA:** 0xad7ae5
- **Atlas file:** `libs/atlas-packet/buddy/clientbound/list_update.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (7 / 10 / 18)` | ✅ |  |
| 1 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 2 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 3 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |


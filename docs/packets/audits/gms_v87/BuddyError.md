# BuddyError (← `CWvsContext::OnFriendResult#Error`)

- **IDA:** 0xad7ae5
- **Atlas file:** `../../libs/atlas-packet/buddy/clientbound/error.go`
- **Variant:** GMS/v87
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte` | ✅ |  |
| 1 | byte | byte `hasMsg flag` | ✅ |  |
| 2 | byte | string `error message string (only if hasMsg != 0)` | ❌ | atlas: short — missing trailing field |


# BuddyUnknownError (← `CWvsContext::OnFriendResult#UnknownError`)

- **IDA:** 0xa8ada2
- **Atlas file:** `libs/atlas-packet/buddy/clientbound/error.go`
- **Variant:** GMS/v84
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (16)` | ✅ |  |
| 1 | byte | byte `trailing extra byte (CInPacket::Decode1)` | ✅ |  |


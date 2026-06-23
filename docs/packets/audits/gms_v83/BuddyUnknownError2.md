# BuddyUnknownError2 (← `CWvsContext::OnFriendResult#UnknownError2`)

- **IDA:** 0xa3f2e8
- **Atlas file:** `libs/atlas-packet/buddy/clientbound/error.go`
- **Variant:** GMS/v83
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (17)` | ✅ |  |
| 1 | byte | byte `trailing extra byte (CInPacket::Decode1)` | ✅ |  |


# BuddyUnknownError4 (← `CWvsContext::OnFriendResult#UnknownError4`)

- **IDA:** 0xad7ae5
- **Atlas file:** `libs/atlas-packet/buddy/clientbound/error.go`
- **Variant:** GMS/v87
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (22)` | ✅ |  |
| 1 | byte | byte `trailing extra byte (CInPacket::Decode1)` | ✅ |  |


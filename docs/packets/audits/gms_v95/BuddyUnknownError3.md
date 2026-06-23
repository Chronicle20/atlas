# BuddyUnknownError3 (← `CWvsContext::OnFriendResult#UnknownError3`)

- **IDA:** 0xa12630
- **Atlas file:** `libs/atlas-packet/buddy/clientbound/error.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (19)` | ✅ |  |
| 1 | byte | byte `trailing extra byte (CInPacket::Decode1)` | ✅ |  |


# BuddyError (← `CWvsContext::OnFriendResult#Error`)

- **IDA:** 0xb2a873
- **Atlas file:** `../../libs/atlas-packet/buddy/clientbound/error.go`
- **Variant:** JMS/v185
- **Branch depth:** 1
- **Verdict:** ⚠️

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte = 11–22 range (error codes)` | ✅ |  |
| 1 | byte | byte `` | ⚠️ | atlas: trailing padding byte — client stops reading (harmless over-write) |


# BuddyChannelChange (← `CWvsContext::OnFriendResult#ChannelChange`)

- **IDA:** 0xb2a873
- **Atlas file:** `libs/atlas-packet/buddy/clientbound/channel_change.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte = 0x14 (20 — ChannelChange)` | ✅ |  |
| 1 | int32 | int32 `friendId` | ✅ |  |
| 2 | byte | byte `channelId` | ✅ |  |
| 3 | int32 | int32 `mapId` | ✅ |  |


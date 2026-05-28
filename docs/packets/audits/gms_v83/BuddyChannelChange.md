# BuddyChannelChange (← `CWvsContext::OnFriendResult#ChannelChange`)

- **IDA:** 0xa3f2e8
- **Atlas file:** `../../libs/atlas-packet/buddy/clientbound/channel_change.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (0x14)` | ✅ |  |
| 1 | int32 | int32 `characterId` | ✅ |  |
| 2 | byte | byte `channelId` | ✅ |  |
| 3 | int32 | int32 `mapId` | ✅ |  |


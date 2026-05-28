# BuddyChannelChange (← `CWvsContext::OnFriendResult#ChannelChange`)

- **IDA:** 0xa12630
- **Atlas file:** `../../libs/atlas-packet/buddy/clientbound/channel_change.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (=0x14/20, CHANNEL_CHANGE) — dispatcher switch byte consumed by OnFriendResult` | ✅ |  |
| 1 | int32 | int32 `dwFriendID — characterId of the buddy whose channel changed` | ✅ |  |
| 2 | byte | byte `inShop — 0=not in cash shop, 1=in cash shop (m_aInShop[Index])` | ✅ |  |
| 3 | int32 | int32 `nChannelID — new channel id (stored in m_aFriend[Index].nChannelID; -1=offline)` | ✅ |  |


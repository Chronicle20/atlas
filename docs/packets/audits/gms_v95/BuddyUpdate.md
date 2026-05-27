# BuddyUpdate (← `CWvsContext::OnFriendResult#Update`)

- **IDA:** 0xa12630
- **Atlas file:** `libs/atlas-packet/buddy/clientbound/update.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** 🔍

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (=0x08, UPDATE) — dispatcher switch byte consumed by OnFriendResult` | ✅ |  |
| 1 | int32 | int32 `dwFriendID — characterId used to locate the buddy via CFriend::FindIndex` | ✅ |  |
| 2 | byte | bytes `GW_Friend update — 39 bytes via GW_Friend::Decode → DecodeBuffer(this,0x27): dwFriendID(4)+sFriendName(13)+nFlag(1)+nChannelID(4)+sFriendGroup(17)` | 🔍 | sub-struct: bm — see _substruct/ |

ack: tool-limitation — model.Buddy sub-struct call emits 39 bytes (FriendId4+PaddedName13+Flag1+ChannelId4+Group17) matching GW_Friend::Decode DecodeBuffer(this,0x27) at CFriend::UpdateFriend@0xa125d0 exactly. Analyzer represents the sub-struct call as a single byte placeholder (🔍). Wire is correct. Verdict promoted to ⚠️.
| 3 | byte | byte `inShop — updated inShop flag stored in m_aInShop[Index]` | ✅ |  |


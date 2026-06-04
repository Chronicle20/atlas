# BuddyInvite (← `CWvsContext::OnFriendResult#Invite`)

- **IDA:** 0xa12630
- **Atlas file:** `../../libs/atlas-packet/buddy/clientbound/invite.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** 🔍

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (=0x09, INVITE) — dispatcher switch byte consumed by OnFriendResult` | ✅ |  |
| 1 | int32 | int32 `dwFriendID — originatorId (characterId of the player sending the friend request)` | ✅ |  |
| 2 | string | string `originatorName — display name of the requester (ZXString, variable length)` | ✅ |  |
| 3 | int32 | int32 `v25 — extra 4-byte field decoded before GW_Friend::Insert (purpose unknown; possibly job/level)` | ✅ |  |
| 4 | int32 | int32 `v26 — extra 4-byte field decoded before GW_Friend::Insert (purpose unknown; possibly job/level)` | ✅ |  |
| 5 | byte | bytes `GW_Friend entry (39 bytes via CFriend::Insert → GW_Friend::Decode → DecodeBuffer(this,0x27)): dwFriendID(4)+sFriendName(13)+nFlag(1)+nChannelID(4)+sFriendGroup(17)` | 🔍 | sub-struct: b — see _substruct/ |
| 6 | byte | byte `inShop — after GW_Friend::Insert, CFriend::Insert reads Decode1 for m_aInShop[new_entry]` | ✅ |  |


# BuddyInvite (← `CWvsContext::OnFriendResult#Invite`)

- **IDA:** 0xad7ae5
- **Atlas file:** `../../libs/atlas-packet/buddy/clientbound/invite.go`
- **Variant:** GMS/v87
- **Branch depth:** 1
- **Verdict:** 🔍

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (9)` | ✅ |  |
| 1 | int32 | int32 `characterId (v21)` | ✅ |  |
| 2 | string | string `inviterName` | ✅ |  |
| 3 | int32 | byte `buddy list count` | ❌ | width mismatch |
| 4 | int32 | int32 `buddy[i].characterId (loop)` | ✅ |  |
| 5 | byte | byte `buddy[i].channelId` | 🔍 | sub-struct: b — see _substruct/ |
| 6 | byte | int32 `buddy[i].mapId` | ❌ | width mismatch |


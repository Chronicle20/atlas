# BuddyInvite (← `CWvsContext::OnFriendResult#Invite`)

- **IDA:** 0xa3f2e8
- **Atlas file:** `../../libs/atlas-packet/buddy/clientbound/invite.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** 🔍

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (9)` | ✅ |  |
| 1 | int32 | int32 `characterId (v23)` | ✅ |  |
| 2 | string | string `inviterName` | ✅ |  |
| 3 | byte | byte `buddy list count (sub_A40028)` | 🔍 | sub-struct: b — see _substruct/ |
| 4 | byte | int32 `buddy[i].characterId (loop)` | ❌ | width mismatch |
| 5 | byte | byte `buddy[i].channelId` | ❌ | atlas: short — missing trailing field |
| 6 | byte | int32 `buddy[i].mapId` | ❌ | atlas: short — missing trailing field |


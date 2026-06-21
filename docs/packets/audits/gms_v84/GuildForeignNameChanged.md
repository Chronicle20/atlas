# GuildForeignNameChanged (← `CUserRemote::OnGuildNameChanged`)

- **IDA:** 0x0
- **Atlas file:** `libs/atlas-packet/guild/clientbound/name_changed_foreign.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `characterId (dispatcher-prefix pattern — consumed by CUserPool::OnUserRemotePacket Decode4 before OnGuildNameChanged)` | ✅ |  |
| 1 | string | string `newGuildName` | ✅ |  |


# GuildForeignNameChanged (← `CUserRemote::OnGuildNameChanged`)

- **IDA:** 0xa5763e
- **Atlas file:** `../../libs/atlas-packet/guild/clientbound/name_changed_foreign.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `characterId — consumed by CUserPool dispatcher before OnGuildNameChanged` | ✅ |  |
| 1 | string | string `new guild name` | ✅ |  |


# GuildForeignNameChanged (← `CUserRemote::OnGuildNameChanged`)

- **IDA:** 0x7cc166
- **Atlas file:** `libs/atlas-packet/guild/clientbound/name_changed_foreign.go`
- **Variant:** GMS/v61
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | string `` | ❌ | width mismatch |
| 1 | string | byte `` | ❌ | atlas: extra — client never reads this field |


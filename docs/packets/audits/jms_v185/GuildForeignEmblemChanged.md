# GuildForeignEmblemChanged (← `CUserRemote::OnGuildMarkChanged`)

- **IDA:** 0xa57689
- **Atlas file:** `libs/atlas-packet/guild/clientbound/emblem_changed_foreign.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `characterId — consumed by dispatcher` | ✅ |  |
| 1 | int16 | int16 `logoId` | ✅ |  |
| 2 | byte | byte `logoColor` | ✅ |  |
| 3 | int16 | int16 `backgroundId` | ✅ |  |
| 4 | byte | byte `backgroundColor` | ✅ |  |


# GuildForeignEmblemChanged (← `CUserRemote::OnGuildMarkChanged`)

- **IDA:** 0x7cc1b1
- **Atlas file:** `libs/atlas-packet/guild/clientbound/emblem_changed_foreign.go`
- **Variant:** GMS/v61
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int16 `` | ❌ | width mismatch |
| 1 | int16 | byte `` | ❌ | width mismatch |
| 2 | byte | int16 `` | ❌ | width mismatch |
| 3 | int16 | byte `` | ❌ | width mismatch |
| 4 | byte | byte `` | ⚠️ | atlas: trailing padding byte — client stops reading (harmless over-write) |


# GuildWithdraw (← `CField::SendWithdrawGuildMsg`)

- **IDA:** 0x4c5cc4
- **Atlas file:** `libs/atlas-packet/guild/serverbound/operation_withdraw.go`
- **Variant:** GMS/v48
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `GUILD_OPERATION mode = 7 (WITHDRAW) @0x4c5dab` | ✅ |  |
| 1 | int32 | int32 `character id @0x4c5db9` | ✅ |  |
| 2 | string | string `character name @0x4c5dd8` | ✅ |  |


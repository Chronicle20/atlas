# GuildSetEmblem (← `CField::SendSetGuildMarkMsg`)

- **IDA:** 0x56e325
- **Atlas file:** `libs/atlas-packet/guild/serverbound/operation_set_emblem.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | byte `sub-op = 9 (SET_EMBLEM)` | ❌ | width mismatch |
| 1 | byte | int16 `logoId (word)` | ❌ | width mismatch |
| 2 | int16 | byte `logoColor (byte)` | ❌ | width mismatch |
| 3 | byte | int16 `backgroundId (word)` | ❌ | width mismatch |
| 4 | byte | byte `backgroundColor (byte)` | ❌ | atlas: short — missing trailing field |


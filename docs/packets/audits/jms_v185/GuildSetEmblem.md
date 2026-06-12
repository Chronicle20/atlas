# GuildSetEmblem (← `CField::SendSetGuildMarkMsg`)

- **IDA:** 0x56e325
- **Atlas file:** `../../libs/atlas-packet/guild/serverbound/operation_set_emblem.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `sub-op = 9 (SET_EMBLEM)` | ✅ |  |
| 1 | int16 | int16 `logoId (word)` | ✅ |  |
| 2 | byte | byte `logoColor (byte)` | ✅ |  |
| 3 | int16 | int16 `backgroundId (word)` | ✅ |  |
| 4 | byte | byte `backgroundColor (byte)` | ✅ |  |


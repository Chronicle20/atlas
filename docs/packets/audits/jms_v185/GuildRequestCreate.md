# GuildRequestCreate (← `CField::InputGuildName`)

- **IDA:** 0x56d98c
- **Atlas file:** `../../libs/atlas-packet/guild/serverbound/operation_request_create.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `sub-op = 1 (CREATE request with name)` | ✅ |  |
| 1 | string | string `desired guild name` | ✅ |  |


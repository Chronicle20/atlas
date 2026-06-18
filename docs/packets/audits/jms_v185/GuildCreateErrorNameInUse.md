# GuildCreateErrorNameInUse (← `CWvsContext::OnGuildResult#CreateErrorNameInUse`)

- **IDA:** 0xb22518
- **Atlas file:** `libs/atlas-packet/guild/clientbound/operation.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (THE_NAME_IS_ALREADY_IN_USE)` | ✅ |  |


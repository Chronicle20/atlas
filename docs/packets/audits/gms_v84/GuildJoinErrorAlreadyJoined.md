# GuildJoinErrorAlreadyJoined (← `CWvsContext::OnGuildResult#JoinErrorAlreadyJoined`)

- **IDA:** 0xa82e2b
- **Atlas file:** `libs/atlas-packet/guild/clientbound/operation.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (ALREADY_JOINED)` | ✅ |  |


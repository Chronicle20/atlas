# GuildInviteErrorAnotherInvite (← `CWvsContext::OnGuildResult#InviteErrorAnotherInvite`)

- **IDA:** 0xacf7d3
- **Atlas file:** `libs/atlas-packet/guild/clientbound/operation.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (IS_TAKING_CARE_OF_ANOTHER_INVITATION)` | ✅ |  |
| 1 | string | string `targetName` | ✅ |  |


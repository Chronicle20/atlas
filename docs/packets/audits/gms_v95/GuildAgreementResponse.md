# GuildAgreementResponse (← `CField::SendCreateGuildAgreeMsg`)

- **IDA:** 0x52d780
- **Atlas file:** `libs/atlas-packet/guild/serverbound/operation_agreement_response.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `guildId (from CWvsContext)` | ✅ |  |
| 1 | byte | byte `bAgree bool (0=no, 1=yes)` | ✅ |  |


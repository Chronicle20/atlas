# GuildAgreementResponse (← `CField::SendCreateGuildAgreeMsg`)

- **IDA:** 0x56da47
- **Atlas file:** `../../libs/atlas-packet/guild/serverbound/operation_agreement_response.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `partyId (guild-creation party)` | ✅ |  |
| 1 | byte | byte `agreed flag (1=yes, 0=no)` | ✅ |  |


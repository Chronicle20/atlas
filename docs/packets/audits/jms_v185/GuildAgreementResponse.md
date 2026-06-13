# GuildAgreementResponse (← `CField::SendCreateGuildAgreeMsg`)

- **IDA:** 0x56da47
- **Atlas file:** `libs/atlas-packet/guild/serverbound/operation_agreement_response.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `leading sub-action byte (task-081 off-by-one remediation 2026-06-10)` | ✅ |  |
| 1 | int32 | int32 `partyId (guild-creation party)` | ✅ |  |
| 2 | byte | byte `agreed flag (1=yes, 0=no)` | ✅ |  |


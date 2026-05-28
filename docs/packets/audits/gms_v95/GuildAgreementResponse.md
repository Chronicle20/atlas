# GuildAgreementResponse (← `CWvsContext::OnGuildResult#AgreementResponse`)

- **IDA:** 0x0
- **Atlas file:** `libs/atlas-packet/guild/serverbound/operation_agreement_response.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `unk (partyId or similar)` | ✅ |  |
| 1 | byte | byte `agreed bool` | ✅ |  |


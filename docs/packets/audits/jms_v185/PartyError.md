# PartyError (← `CWvsContext::OnPartyResult#Error`)

- **IDA:** 0xb297e7
- **Atlas file:** `../../libs/atlas-packet/party/clientbound/error.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (error code)` | ✅ |  |
| 1 | string | string `name` | ✅ |  |


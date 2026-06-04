# PartyError (← `CWvsContext::OnPartyResult#Error`)

- **IDA:** 0xa10e9e
- **Atlas file:** `../../libs/atlas-packet/party/clientbound/error.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 1 | string | byte `` | ❌ | atlas: extra — client never reads this field |


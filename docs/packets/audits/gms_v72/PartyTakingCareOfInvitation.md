# PartyTakingCareOfInvitation (← `CWvsContext::OnPartyResult#TakingCareOfInvitation`)

- **IDA:** 
- **Atlas file:** `libs/atlas-packet/party/clientbound/error.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | unresolved `function not found in IDB` | 🚫 | IDA read-order unresolved: function not found in IDB |
| 1 | string | byte `` | ❌ | atlas: extra — client never reads this field |


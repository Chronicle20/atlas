# PartyCreated (← `CWvsContext::OnPartyResult#Created`)

- **IDA:** 
- **Atlas file:** `libs/atlas-packet/party/clientbound/created.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | unresolved `function not found in IDB` | 🚫 | IDA read-order unresolved: function not found in IDB |
| 1 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 2 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 3 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 4 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 5 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |


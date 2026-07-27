# ReceiveResponse (← `CWvsContext::OnGivePopularityResult#ReceiveResponse`)

- **IDA:** 
- **Atlas file:** `libs/atlas-packet/fame/clientbound/response.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | unresolved `function not found in IDB` | 🚫 | IDA read-order unresolved: function not found in IDB |
| 1 | string | byte `` | ❌ | atlas: extra — client never reads this field |
| 2 | byte | byte `` | ❌ | atlas: extra — client never reads this field |


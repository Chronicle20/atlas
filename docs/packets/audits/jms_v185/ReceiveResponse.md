# ReceiveResponse (← `CWvsContext::OnGivePopularityResult#ReceiveResponse`)

- **IDA:** 0xb094aa
- **Atlas file:** `libs/atlas-packet/fame/clientbound/response.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (case 5 = RECEIVE)` | ✅ |  |
| 1 | string | string `fromName (character who gave fame)` | ✅ |  |
| 2 | byte | byte `bInc (1=fame, 0=defame)` | ✅ |  |


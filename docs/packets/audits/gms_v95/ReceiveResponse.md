# ReceiveResponse (← `CWvsContext::OnGivePopularityResult#ReceiveResponse`)

- **IDA:** 0x9fea60
- **Atlas file:** `../../libs/atlas-packet/fame/clientbound/response.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (switch dispatch; case 5 = RECEIVE)` | ✅ |  |
| 1 | string | string `fromName (character who gave fame)` | ✅ |  |
| 2 | byte | byte `bInc (1=fame, 0=defame)` | ✅ |  |


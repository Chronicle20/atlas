# ServerListRecommendations (← `CLogin::OnRecommendWorldMessage`)

- **IDA:** 0x66e6f1
- **Atlas file:** `libs/atlas-packet/login/clientbound/server_list_recommendations.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ✅ |  |
| 1 | int32 | int32 `` | ✅ |  |
| 2 | string | string `` | ✅ |  |


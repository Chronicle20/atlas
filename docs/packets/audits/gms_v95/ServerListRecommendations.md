# ServerListRecommendations (← `CLogin::OnRecommendWorldMessage`)

- **IDA:** 0x5d7280
- **Atlas file:** `../../libs/atlas-packet/login/clientbound/server_list_recommendations.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ⚠️

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `nCount` | ✅ |  |
| 1 | byte | int32 `nWorldID (loop body)` | ⚠️ | loop body — atlas emits zero iterations (count==0) |
| 2 | byte | string `sMessage (loop body)` | ⚠️ | loop body — atlas emits zero iterations (count==0) |


# ServerListRecommendations (← `CLogin::OnRecommendWorldMessage`)

- **IDA:** 0x5d7280
- **Atlas file:** `../../libs/atlas-packet/login/clientbound/server_list_recommendations.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** 🔍

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `nCount` | ✅ |  |
| 1 | byte | int32 `nWorldID (loop body)` | 🔍 | loop body — see follow-up scan |
| 2 | byte | string `sMessage (loop body)` | ❌ | atlas: short — missing trailing field |


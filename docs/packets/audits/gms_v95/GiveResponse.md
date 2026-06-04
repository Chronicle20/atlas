# GiveResponse (← `CWvsContext::OnGivePopularityResult#GiveResponse`)

- **IDA:** 0x9fea60
- **Atlas file:** `../../libs/atlas-packet/fame/clientbound/response.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (switch dispatch; case 0 = GIVE)` | ✅ |  |
| 1 | string | string `toName (recipient of the fame)` | ✅ |  |
| 2 | byte | byte `bInc (1=fame, 0=defame)` | ✅ |  |
| 3 | int32 | int32 `nPOP (new total fame as int32)` | ✅ |  |


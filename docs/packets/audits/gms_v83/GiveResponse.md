# GiveResponse (← `CWvsContext::OnGivePopularityResult#GiveResponse`)

- **IDA:** 0xa223dc
- **Atlas file:** `../../libs/atlas-packet/fame/clientbound/response.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (switch dispatch; case 0 = GIVE — else-branch in v83 subtraction chain)` | ✅ |  |
| 1 | string | string `fromName / toName (recipient of fame)` | ✅ |  |
| 2 | byte | byte `bInc (1=fame, 0=defame)` | ✅ |  |
| 3 | int32 | int32 `nPOP (new total fame as int32; passed to CUIUserInfo::NotifyGivePopResult @0xa22663)` | ✅ |  |


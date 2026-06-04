# GiveResponse (← `CWvsContext::OnGivePopularityResult#GiveResponse`)

- **IDA:** 0xb094aa
- **Atlas file:** `../../libs/atlas-packet/fame/clientbound/response.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (case 0 = GIVE)` | ✅ |  |
| 1 | string | string `toName (recipient)` | ✅ |  |
| 2 | byte | byte `bInc (1=fame, 0=defame)` | ✅ |  |
| 3 | int32 | int32 `nPOP (new total fame — CUIUserInfo::NotifyGivePopResult uses Decode4)` | ✅ |  |


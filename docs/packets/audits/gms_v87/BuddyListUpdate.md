# BuddyListUpdate (← `CWvsContext::OnFriendResult#ListUpdate`)

- **IDA:** 0xad7ae5
- **Atlas file:** `../../libs/atlas-packet/buddy/clientbound/list_update.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode` | ✅ |  |
| 1 | byte | byte `count` | ✅ |  |
| 2 | bytes | bytes `GW_Friend ×count` | ✅ |  |
| 3 | int32 | bytes `inShop flags ×count` | ✅ |  |


# BuddyUpdate (← `CWvsContext::OnFriendResult#Update`)

- **IDA:** 0xad7ae5
- **Atlas file:** `../../libs/atlas-packet/buddy/clientbound/update.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode` | ✅ |  |
| 1 | int32 | int32 `characterId` | ✅ |  |
| 2 | bytes | bytes `GW_Friend block` | ✅ |  |
| 3 | byte | byte `inShop` | ✅ |  |


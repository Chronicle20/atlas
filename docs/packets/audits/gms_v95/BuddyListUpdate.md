# BuddyListUpdate (← `CWvsContext::OnFriendResult#ListUpdate`)

- **IDA:** 0xa12630
- **Atlas file:** `../../libs/atlas-packet/buddy/clientbound/list_update.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (=0x07/0x0A/0x12, LIST_UPDATE variants) — dispatcher switch byte consumed by OnFriendResult` | ✅ |  |
| 1 | byte | byte `count — number of buddy entries (v4); used for ZArray realloc` | ✅ |  |
| 2 | bytes | bytes `buddy entries — count × 39 bytes (GW_Friend array) decoded as DecodeBuffer(m_aFriend.a, 39*count)` | ✅ |  |
| 3 | int32 | bytes `inShop flags — count × 4 bytes (int array) decoded as DecodeBuffer(m_aInShop.a, 4*count)` | ✅ |  |


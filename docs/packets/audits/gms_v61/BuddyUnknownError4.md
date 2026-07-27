# BuddyUnknownError4 (← `CWvsContext::OnFriendResult#UnknownError4`)

- **IDA:** 
- **Atlas file:** `libs/atlas-packet/buddy/clientbound/error.go`
- **Variant:** GMS/v61
- **Branch depth:** 1
- **Verdict:** ⚠️

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | unresolved `function not found in IDB` | 🚫 | IDA read-order unresolved: function not found in IDB |
| 1 | byte | byte `` | ⚠️ | atlas: trailing padding byte — client stops reading (harmless over-write) |


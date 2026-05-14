# ServerListEnd (← `CLogin::OnWorldInformation#ServerListEnd`)

- **IDA:** 0x5f95b7
- **Atlas file:** `../../libs/atlas-packet/login/clientbound/server_list_end.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `nWorldID terminator (0xFF dispatch — end of world list)` | ✅ |  |


# ServerListEnd (← `CLogin::OnWorldInformation#ServerListEnd`)

- **IDA:** 0x56663f
- **Atlas file:** `libs/atlas-packet/login/clientbound/server_list_end.go`
- **Variant:** GMS/v61
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `worldId/terminator @0x566660 (0xFF = end of world list)` | ✅ |  |


# SelectWorld (← `CLogin::OnLatestConnectedWorld`)

- **IDA:** 0x5d2200
- **Atlas file:** `libs/atlas-packet/login/clientbound/select_world.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `latestConnectedWorldID` | ✅ |  |


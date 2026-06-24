# ServerListEnd (← `CLogin::OnWorldInformation#ServerListEnd`)

- **IDA:** 0x66f107
- **Atlas file:** `libs/atlas-packet/login/clientbound/server_list_end.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `0xFF end-of-list sentinel (OnWorldInformation nWorldID<0 branch); atlas writes WriteByte(0xFF)` | ✅ |  |


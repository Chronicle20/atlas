# Request (← `CLogin::SendCheckPasswordPacket`)

- **IDA:** 0x62dfb4
- **Atlas file:** `libs/atlas-packet/login/serverbound/request.go`
- **Variant:** GMS/v87
- **Branch depth:** 2
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `name` | ✅ |  |
| 1 | string | string `password` | ✅ |  |
| 2 | bytes | bytes `machineId (16 bytes)` | ✅ |  |
| 3 | int32 | int32 `gameRoomClient` | ✅ |  |
| 4 | byte | byte `gameStartMode` | ✅ |  |
| 5 | byte | byte `unknown1 (literal 0)` | ✅ |  |
| 6 | byte | byte `unknown2 (literal 0) — present in v87; gate MajorVersion()>=95 in atlas is wrong for v87` | ❌ | atlas: short — missing trailing field |
| 7 | byte | int32 `PartnerCode (GetPartnerCode()) — v87-only extra int32 after the three Encode1s; absent in v83 and v95` | ❌ | atlas: short — missing trailing field |


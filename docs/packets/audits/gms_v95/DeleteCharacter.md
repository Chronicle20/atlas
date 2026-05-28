# DeleteCharacter (← `CLogin::SendDeleteCharPacket`)

- **IDA:** 0x5d53a0
- **Atlas file:** `../../libs/atlas-packet/character/serverbound/delete.go`
- **Variant:** GMS/v95
- **Branch depth:** 3
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `sSPW (PIC/soft-keyboard password; v95 always uses PIC path, m_bLoginOpt==1)` | ✅ |  |
| 1 | int32 | int32 `characterStat.dwCharacterID` | ✅ |  |


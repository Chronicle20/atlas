# ChalkboardUse (← `CUser::OnADBoard`)

- **IDA:** 0x9b1d1e
- **Atlas file:** `libs/atlas-packet/character/clientbound/chalkboard.go`
- **Variant:** GMS/v87
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `characterId (consumed by CUserPool::OnUserCommonPacket)` | ✅ |  |
| 1 | byte | byte `active (bool: 1=show chalkboard, 0=clear)` | ✅ |  |
| 2 | string | string `message text (only if active != 0)` | ✅ |  |


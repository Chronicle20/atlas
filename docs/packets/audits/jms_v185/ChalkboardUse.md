# ChalkboardUse (← `CUser::OnADBoard`)

- **IDA:** 0x9f6199
- **Atlas file:** `libs/atlas-packet/character/clientbound/chalkboard.go`
- **Variant:** JMS/v185
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `characterId (consumed by dispatcher before calling this function)` | ✅ |  |
| 1 | byte | byte `active (bool: 1=show chalkboard, 0=clear)` | ✅ |  |
| 2 | string | string `message text (only if active != 0)` | ✅ |  |


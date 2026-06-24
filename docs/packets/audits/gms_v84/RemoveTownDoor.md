# RemoveTownDoor (← `CWvsContext::OnTownPortal`)

- **IDA:** 0xa6dbb8
- **Atlas file:** `libs/atlas-packet/door/clientbound/remove_town.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `townId (SpawnPortal); MapId.NONE (999999999) for RemoveTownDoor` | ✅ |  |
| 1 | int32 | int32 `targetId; MapId.NONE (999999999) for RemoveTownDoor` | ✅ |  |
| 2 | byte | int16 `x (door position)` | ❌ | atlas: short — missing trailing field |
| 3 | byte | int16 `y (door position)` | ❌ | atlas: short — missing trailing field |


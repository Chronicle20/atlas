# SpawnDoor (← `CTownPortalPool::OnTownPortalCreated`)

- **IDA:** 0x6f96a2
- **Atlas file:** `libs/atlas-packet/door/clientbound/spawn.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ✅ |  |
| 1 | int32 | int32 `` | ✅ |  |
| 2 | int16 | int16 `` | ✅ |  |
| 3 | int16 | int16 `` | ✅ |  |
| 4 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 5 | byte | int16 `` | ❌ | atlas: short — missing trailing field |


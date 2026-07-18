# Use (← `CWvsContext::SendMapTransferItemUseRequest`)

- **IDA:** 0x9560c6
- **Atlas file:** `libs/atlas-packet/teleportrock/serverbound/use.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int16 `nPOS (inventory slot)` | ✅ |  |
| 1 | int32 | int32 `nItemID` | ✅ |  |
| 2 | byte | int32 `updateTime (trailing, only if the target dialog was confirmed)` | ❌ | width mismatch |
| 3 | string | byte `` | ❌ | atlas: extra — client never reads this field |
| 4 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 5 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |


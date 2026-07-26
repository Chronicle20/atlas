# PetItemUse (← `CWvsContext::SendStatChangeItemUseRequestByPetQ`)

- **IDA:** 0x70dc8d
- **Atlas file:** `libs/atlas-packet/pet/serverbound/item_use.go`
- **Variant:** GMS/v48
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int64 | byte `` | ❌ | width mismatch |
| 1 | byte | int32 `` | ❌ | width mismatch |
| 2 | int32 | int16 `` | ❌ | width mismatch |
| 3 | int16 | int32 `` | ❌ | width mismatch |
| 4 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |


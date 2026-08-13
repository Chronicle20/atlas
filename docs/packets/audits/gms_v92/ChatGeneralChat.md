# ChatGeneralChat (← `CUser::OnChat`)

- **IDA:** 0x8d12b0
- **Atlas file:** `libs/atlas-packet/chat/clientbound/general.go`
- **Variant:** GMS/v92
- **Branch depth:** 0
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `` | ❌ | width mismatch |
| 1 | byte | string `` | ❌ | width mismatch |
| 2 | string | byte `` | ❌ | width mismatch |
| 3 | byte | string `` | ❌ | width mismatch |


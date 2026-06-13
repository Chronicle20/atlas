# BuffCancel (← `CWvsContext::OnTemporaryStatReset`)

- **IDA:** 0xa6bb24
- **Atlas file:** `libs/atlas-packet/character/clientbound/buff_cancel.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | bytes `` | ✅ |  |
| 1 | byte | byte `` | ❌ | atlas: short — missing trailing field |


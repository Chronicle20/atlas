# BuffCancel (← `CWvsContext::OnTemporaryStatReset`)

- **IDA:** 0x9f2ab0
- **Atlas file:** `../../libs/atlas-packet/character/clientbound/buff_cancel.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | bytes `uFlagTemp: 16-byte UINT128 stat mask (DecodeBuffer 0x10)` | ✅ |  |
| 1 | byte | byte `nChangedStatPoint (only present when IsMovementAffectingStat(mask) is true)` | ❌ | atlas: short — missing trailing field |


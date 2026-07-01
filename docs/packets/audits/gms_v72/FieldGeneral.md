# FieldGeneral (← `CField::SendChatMsg`)

- **IDA:** 0x50b7dc
- **Atlas file:** `libs/atlas-packet/field/serverbound/general.go`
- **Variant:** GMS/v72
- **Branch depth:** 2
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | string `message @0x50b898` | ❌ | width mismatch |
| 1 | string | byte `bOnlyBalloon @0x50b8a3` | ❌ | width mismatch |
| 2 | byte | byte `` | ⚠️ | atlas: trailing padding byte — client stops reading (harmless over-write) |


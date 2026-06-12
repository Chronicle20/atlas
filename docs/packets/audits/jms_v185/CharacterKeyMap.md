# CharacterKeyMap (← `CFuncKeyMappedMan::OnInit`)

- **IDA:** 0x5e79aa
- **Atlas file:** `../../libs/atlas-packet/character/clientbound/keymap.go`
- **Variant:** JMS/v185
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `resetToDefault flag` | ✅ |  |
| 1 | byte | byte `FUNCKEY_MAPPED::nType (loop body)` | ✅ |  |
| 2 | byte | int32 `FUNCKEY_MAPPED::nID (loop body)` | ❌ | width mismatch |
| 3 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |


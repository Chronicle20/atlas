# CharacterSkillMacroHandle (← `CMacroSysMan::FlushToSvr`)

- **IDA:** 0x602ed0
- **Atlas file:** `libs/atlas-packet/character/serverbound/skill_macro.go`
- **Variant:** GMS/v92
- **Branch depth:** 0
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ✅ |  |
| 1 | string | string `` | ✅ |  |
| 2 | byte | byte `` | ✅ |  |
| 3 | int32 | int32 `task-226: hand-added -- Hex-Rays emits this as an inlined manual buffer write (direct pointer store + ZArray growth), not a textual COutPacket::Encode4(...) call, so the regex parser cannot see it; verified by decompile against the 3x uint32 skill-id fields (matches the DecodeStr/Decode1/Decode4 shape of every sibling version's SINGLEMACRO::Encode/Decode).` | ✅ |  |
| 4 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 5 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |


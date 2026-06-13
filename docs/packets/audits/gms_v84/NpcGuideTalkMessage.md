# NpcGuideTalkMessage (← `CUserLocal::OnTutorMsg#Message`)

- **IDA:** 0x99f28c
- **Atlas file:** `libs/atlas-packet/npc/clientbound/guide_talk.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ✅ |  |
| 1 | string | int32 `` | ❌ | width mismatch |
| 2 | int32 | int32 `` | ✅ |  |
| 3 | int32 | string `` | ❌ | width mismatch |
| 4 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 5 | byte | int32 `` | ❌ | atlas: short — missing trailing field |


# DropPickUp (← `sub_8316B8`)

- **IDA:** 0x8316b8
- **Atlas file:** `libs/atlas-packet/drop/serverbound/pick_up.go`
- **Variant:** GMS/v61
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `Encode1 fieldKey=*((_BYTE*)get_field()+248) @0x831721` | ✅ |  |
| 1 | int32 | int32 `Encode4 updateTime @0x83172f` | ✅ |  |
| 2 | int16 | int16 `Encode2 x=*a2 @0x831740` | ✅ |  |
| 3 | int16 | int16 `Encode2 y=a2[2] @0x83174f` | ✅ |  |
| 4 | int32 | int32 `Encode4 dropId=a3 @0x83175a` | ✅ |  |
| 5 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |


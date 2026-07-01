# FieldMultiChat (← `CField::OnGroupMessage`)

- **IDA:** 0x51626c
- **Atlas file:** `libs/atlas-packet/field/clientbound/multi.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode @0x516284` | ✅ |  |
| 1 | string | string `from @0x5162c1` | ✅ |  |
| 2 | string | string `message @0x516306` | ✅ |  |


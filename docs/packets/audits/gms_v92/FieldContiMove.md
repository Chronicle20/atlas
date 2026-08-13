# FieldContiMove (← `CField_ContiMove::OnContiMove`)

- **IDA:** 0x546fa0
- **Atlas file:** `libs/atlas-packet/field/clientbound/conti_move.go`
- **Variant:** GMS/v92
- **Branch depth:** 1
- **Verdict:** ⚠️

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ✅ |  |
| 1 | byte | byte `` | ⚠️ | atlas: trailing padding byte — client stops reading (harmless over-write) |


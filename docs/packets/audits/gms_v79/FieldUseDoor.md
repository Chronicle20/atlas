# FieldUseDoor (← `CField::TryEnterTownPortal#UseDoor`)

- **IDA:** 0x522946
- **Atlas file:** `libs/atlas-packet/field/serverbound/use_door.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `portalFieldId @0x522b1b` | ✅ |  |
| 1 | byte | byte `flag=1 @0x522b24` | ✅ |  |


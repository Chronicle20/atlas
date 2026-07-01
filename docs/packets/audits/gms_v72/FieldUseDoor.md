# FieldUseDoor (← `CField::TryEnterTownPortal#UseDoor`)

- **IDA:** 0x51b86c
- **Atlas file:** `libs/atlas-packet/field/serverbound/use_door.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `portalFieldId @0x51ba41` | ✅ |  |
| 1 | byte | byte `flag=1 @0x51ba4a` | ✅ |  |


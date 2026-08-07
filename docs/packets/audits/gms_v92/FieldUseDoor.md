# FieldUseDoor (← `CField::TryEnterTownPortal#UseDoor`)

- **IDA:** 
- **Atlas file:** `libs/atlas-packet/field/serverbound/use_door.go`
- **Variant:** GMS/v92
- **Branch depth:** 0
- **Verdict:** ⚠️

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | unresolved `dispatcher-family arm not harvested for gms_v92 (see notes)` | 🚫 | IDA read-order unresolved: dispatcher-family arm not harvested for gms_v92 (see notes) |
| 1 | byte | byte `` | ⚠️ | atlas: trailing padding byte — client stops reading (harmless over-write) |


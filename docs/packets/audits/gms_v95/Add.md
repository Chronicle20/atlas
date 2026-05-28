# Add (← `CWvsContext::OnInventoryOperation#Add`)

- **IDA:** 0xa08a70
- **Atlas file:** `../../libs/atlas-packet/inventory/clientbound/change.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** 🔍

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `!silent (exclRequestSent flag, line 85)` | ✅ |  |
| 1 | byte | byte `count (line 144)` | ✅ |  |
| 2 | byte | byte `mode 0 = Add (line 150)` | ✅ |  |
| 3 | byte | byte `inventoryType (line 151)` | ✅ |  |
| 4 | int16 | int16 `slot (line 152)` | ✅ |  |
| 5 | byte | bytes `asset GW_ItemSlotBase::Decode (case 0 line 158) — sub-struct, tool-opaque` | 🔍 | sub-struct: asset — see _substruct/ |


> ack: tool limitation (asset `GW_ItemSlotBase::Decode` sub-struct flattened to
> one opaque row). NOT a wire bug — the dispatcher case 0 reads
> mode/invType/slot then the item via `GW_ItemSlotBase::Decode`, which is exactly
> what `Add.Encode` emits (`mode 0, type, slot, asset.Encode`). The asset body is
> the shared `model.Asset` sub-struct, audited independently. See
> `docs/packets/ida-exports/_pending.md` → "Add clientbound — asset sub-struct
> tool limitation (inventory)".

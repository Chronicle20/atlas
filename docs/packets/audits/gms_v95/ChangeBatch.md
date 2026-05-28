# ChangeBatch (← `CWvsContext::OnInventoryOperation#ChangeBatch`)

- **IDA:** 0xa08a70
- **Atlas file:** `../../libs/atlas-packet/inventory/clientbound/change_batch.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `!silent (exclRequestSent flag, line 85)` | ✅ |  |
| 1 | byte | byte `count (line 144)` | ✅ |  |
| 2 | bytes | bytes `per-entry loop: Decode1 mode, Decode1 invType, Decode2 slot, + mode-specific body (line 148-411); trailing Decode1 addMov if any equip-slot move/remove (line 315)` | ✅ |  |
| 3 | byte | byte `` | ❌ | atlas: extra — client never reads this field |


> ack: tool limitation (variable-length per-entry loop collapsed to one buffer
> op; the conditional trailing `addMov` byte (row 3) cannot be aligned against
> it). NOT a wire bug — the dispatcher reads one trailing addMov byte after the
> loop (line 315) iff any entry is an equip-slot move/remove (`invType==1` with a
> negative slot), which is exactly the condition `ChangeBatch.Encode` uses to emit
> its single trailing `WriteInt8(addMov)`. See
> `docs/packets/ida-exports/_pending.md` → "ChangeBatch clientbound — loop +
> conditional trailing addMov tool limitation (inventory)".

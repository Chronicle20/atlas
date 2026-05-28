# Phase 0 Survey — Foreign-Encoder Methods (task-067)

## Non-`Encode`/`Write` foreign-encoder methods in libs/atlas-packet/

| File:Line | Receiver | Method | Return shape | In task-067 scope? |
|---|---|---|---|---|
| `cash/clientbound/shop_inventory.go:25` | `CashInventoryItem` | `EncodeBytes` | flat `[]byte` | Yes |
| `inventory/change_entry.go:73` | `AddEntry` | `EncodeEntry` | `func(options map[string]interface{}) []byte` (closure) | Yes |
| `inventory/change_entry.go:103` | `QuantityUpdateEntry` | `EncodeEntry` | `func(options map[string]interface{}) []byte` (closure) | Yes |
| `inventory/change_entry.go:141` | `MoveEntry` | `EncodeEntry` | `func(options map[string]interface{}) []byte` (closure) | Yes |
| `inventory/change_entry.go:173` | `RemoveEntry` | `EncodeEntry` | `func(options map[string]interface{}) []byte` (closure) | Yes |
| `model/character_temporary_stat.go:575` | `*CharacterTemporaryStat` | `EncodeForeign` | `func(options map[string]interface{}) []byte` (closure) | **No** — task-028 character scope, do not re-audit |

No unexpected methods were found beyond the design-time prediction. The survey matches the predicted set exactly.

## Auto-discovered (no action)

- `model.Asset.Encode` (asset.go:164) — inventory item slot sub-struct; used by inventory/clientbound/change.go, cash/clientbound/shop_item_moved.go, and storage panels.
- All top-level packet structs (cash/interaction/inventory/storage × clientbound/serverbound) expose `Encode`; registry pass-2 picks them up. (Encode count from Step 2: 94.)

## Decision (design §8)

- If only `CashInventoryItem.EncodeBytes` was found: **Option A1** (recognise `EncodeBytes` only).
- If `EncodeEntry` was also found: **Option A2** (recognise both `EncodeBytes` and `EncodeEntry`). EncodeEntry has the same closure shape as `Encode`, so the existing `findReturnClosure` path handles it.
- If a method with semantics divergent from both is found: **Option B** (per-call ack in `_pending.md`).

Outcome: **Option A2** — both `CashInventoryItem.EncodeBytes` (flat `[]byte`) and the four `*.EncodeEntry` methods (closure shape identical to `Encode`) were found in task-067 scope. The registry extension must recognise both method names:
1. `EncodeBytes` — flat `[]byte` return; needs its own branch in the type-switch (not a closure, so `findReturnClosure` does not apply).
2. `EncodeEntry` — closure `func(map[string]interface{}) []byte`; the existing `findReturnClosure` path applies unchanged once the method name is added to the switch.

`EncodeForeign` (`model/character_temporary_stat.go:575`) is closure-shaped but belongs to task-028 (character scope) and is excluded from this audit.

No entries need to be added to `docs/packets/ida-exports/_pending.md`; all discovered methods were predicted by design §8.

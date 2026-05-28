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

## Cash constructor ↔ struct map (design §4.4)

| Constructor (shop_operation_body.go) | Target struct | Defined in |
|---|---|---|
| `CashShopCashGiftsBody()` | `CashShopGifts` | `shop_inventory.go` |
| `CashShopInventoryCapacityIncreaseSuccessBody(inventoryType, capacity)` | `InventoryCapacitySuccess` | `shop_operation_result.go` |
| `CashShopInventoryCapacityIncreaseFailedBody(message)` | `InventoryCapacityFailed` | `shop_operation_result.go` |
| `CashShopWishListBody(update, sns)` | `WishList` | `shop_operation_result.go` |
| `CashShopCashInventoryBody(items, storageSlots, characterSlots)` | `CashShopInventory` | `shop_inventory.go` |
| `CashShopCashInventoryPurchaseSuccessBody(item)` | `CashShopPurchaseSuccess` | `shop_inventory.go` |
| `CashShopCashItemMovedToInventoryBody(slot, asset)` | `CashItemMovedToInventory` | `shop_item_moved.go` |
| `CashShopCashItemMovedToCashInventoryBody(item)` | `CashItemMovedToCashInventory` | `shop_item_moved.go` |

**Discrepancy vs design prediction:** The design predicted constructor names with a `New` prefix (e.g. `NewCashShopWishListBody`). The actual public body factories have no `New` prefix — they are bare names like `CashShopWishListBody`. The internal `New*` constructors (e.g. `NewWishList`, `NewCashShopInventory`) live in the target-struct files and are called from within the body factories; the design prediction conflated these two layers. The count is the same (8 constructors → 8 target structs across 3 files); only the naming convention differed.

Implication for Phase 1d: `shop_operation_body.go` gets ONE row in SUMMARY.md
(verdict ⚠️ "router; per-shape rows recorded under target structs above").
The cash wire-shape denominator stays at ~36; Phase 1d does not duplicate
constructor rows.

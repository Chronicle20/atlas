# Completeness critic — task-240-cash-shop-stub-operations

Verdict: 2 findings (1 CHANGED-BUT-UNCLAIMED, 6 CLAIMED-BUT-UNVERIFIED cells rolled into 2 rows).

Base: `d9ec287b8` (also confirmed via `git merge-base origin/main HEAD`). Head: `3bc7ebd21`.

## CHANGED-BUT-UNCLAIMED

| kind | file / packet | evidence | recommendation |
|---|---|---|---|
| codec | `libs/atlas-packet/character/data.go` (`InventoryData.Timestamp` → `EquipSlotExtExpire`) | `git diff d9ec287b8...3bc7ebd21 -- libs/atlas-packet/character/data.go`: field renamed and re-documented ("Equip-slot-extension expiry FILETIME" replacing "Inventory-update FILETIME"), read/write sites in `encodeInventory`/`decodeInventory` changed from `m.Inventory.Timestamp` to `m.Inventory.EquipSlotExtExpire`. This struct (`character.CharacterData`/`InventoryData`) is embedded in **two ops the manifest never declares**: `field/clientbound/set_field.go:21` (`characterData charpkt.CharacterData`, op `SET_FIELD`) and `field/clientbound/set_itc.go:83` (op `SET_ITC`), both currently mostly `verified` in `status.json` (`SET_FIELD`: v61–v87/v95/jms verified, v92 incomplete; `SET_ITC`: same pattern). The manifest's `ops:` list is only `SET_CASH_SHOP` / `CASHSHOP_OPERATION`; the `out_of_scope:` list has no packet path covering `character/` or `field/clientbound/{SetField,SetItc}` — the closest entry, "equip-slot extension domain … plan Tasks 21-23", is prose about the atlas-character/channel equip-slot domain, not a packet path per the coverage-manifest schema. | Add `character` (or the specific `CharacterData`/`InventoryData` sub-struct) to `out_of_scope` with the justification already implicit in the diff — same bytes on the wire, comment/field-name-only reinterpretation, no behavior change to `SET_FIELD`/`SET_ITC` — or explicitly add those two ops to `ops:` if the critic's assumption of "no wire change" needs its own verification pass. |

No unclaimed **gate** changes: the only `MajorVersion|MajorAtLeast|IsRegion|Region()` diff hit (`ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)`) is inside the new, claimed `libs/atlas-packet/cash/serverbound/shop_operation_buy_other_package_test.go` (confirmed via `git diff ... | grep -B30 ... | grep 'diff --git'`) — it is test-harness plumbing for the claimed `BuyOtherPackage` codec, not a version-gate move.

No unclaimed **matrix** changes: `git diff d9ec287b8...3bc7ebd21 -- docs/packets/audits/status.json` (57 added lines, 1 changed line) shows exactly one new row added — `cash/serverbound/CashShopOperationBuyOtherPackage` (all cells `incomplete` except `gms_v95: verified`) — plus the `exportHashes.gms_v95` hash bump from the hand-spliced `ida-exports/gms_v95.json` entry. This matches the manifest's declared claim #2 exactly (new codec, does not promote other rows). No other row's `cells[...].state` changed.

## CLAIMED-BUT-UNVERIFIED

`claimedOps` = `{SET_CASH_SHOP, CASHSHOP_OPERATION} × {gms_v48, gms_v61, gms_v72, gms_v79, gms_v83, gms_v84, gms_v87, gms_v92, gms_v95, jms_v185}`. Reading the final (HEAD) `status.json` cells for the 3 rows those ops resolve to:

| op | packet | version | actual state | recommendation |
|---|---|---|---|---|
| SET_CASH_SHOP | (row has no `packet` field) | gms_v92 | `incomplete` ("no audit report") | Manifest's own comment (line 54) calls this "the lone ❌ in that row, pre-existing and out of this plan's scope" — but the manifest schema has no per-version carve-out inside `ops × versions`, and `gms_v92` is not removed from `versions:` nor listed in `out_of_scope`. Either drop `gms_v92` from the claimed `versions:` list (if the schema allows per-op version scoping) or add an explicit `out_of_scope` note naming this cell so a future critic doesn't have to infer it from prose. |
| CASHSHOP_OPERATION | cash/serverbound/CashShopOperationGetPurchaseRecord | gms_v48 | `incomplete` ("tier-1 without fixture; verdict 🚫") | Not mentioned anywhere in the manifest's disclaimers (those only cover `gms_v92` for the `SET_CASH_SHOP` row). Verify the cell via `/verify-packet`, or narrow the claimed `versions:` list, or add an explicit `out_of_scope`/notes entry. |
| CASHSHOP_OPERATION | cash/serverbound/CashShopOperationGetPurchaseRecord | gms_v61 | `incomplete` ("no audit report") | Same as above. |
| CASHSHOP_OPERATION | cash/serverbound/CashShopOperationGetPurchaseRecord | gms_v92 | `incomplete` ("tier-1 without fixture; verdict 🚫") | Same as above. |
| CASHSHOP_OPERATION | cash/clientbound/CashBuyFailed | gms_v48 | `n-a` | Legitimately inapplicable state, consistent with an `n-a` disposition — but the manifest's `fields`/notes never call this cell out as intentionally n-a, so a reader can't distinguish "known n-a" from "accidentally uncovered." Add a one-line note. |
| CASHSHOP_OPERATION | cash/clientbound/CashBuyFailed | gms_v92 | `incomplete` ("tier-1 without fixture; verdict ⚠️") | Same recommendation as the GetPurchaseRecord gms_v92 row above. |

All six of the above pre-date this branch — `git diff` over `status.json` shows no state transitions for these rows (only the new `BuyOtherPackage` row was added) — so this is not new breakage from this task, but the manifest's full `ops × versions` cross-product claim is broader than what the branch actually closed, and the reader has no way to tell "acknowledged pre-existing gap" from "silently unclaimed" for 5 of the 6 cells (only the `SET_CASH_SHOP`/`gms_v92` cell has a prose disclaimer).

## Manifest presence

`docs/tasks/task-240-cash-shop-stub-operations/coverage-manifest.yaml` is present and well-formed (`ops`, `versions`, `fields`, `out_of_scope` sections all populated) — not a missing-manifest case.

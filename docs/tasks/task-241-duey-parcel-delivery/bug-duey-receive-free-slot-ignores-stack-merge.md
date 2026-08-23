# Bug — DUEY_ACTION RECEIVE's free-slot check ignores stack merging

- **Task**: task-241-duey-parcel-delivery
- **Branch**: task-241-duey-parcel-delivery
- **Environment**: `atlas-pr-1434`, tenant `a049bb75-1ccc-4cb8-ac6a-bd604dfbbe5b`, GMS 83.1
- **Found by**: inspection while diagnosing
  `bug-duey-receive-unique-conflict-on-stackables.md`. The user was told about it and asked
  for it to be fixed on this branch. **Not player-reported and not reproduced live.**

## Sibling of the unique-conflict bug

Same handler, same shape of defect: a pre-flight gate that ignores stackability. Fix that one
first (`bug-duey-receive-unique-conflict-on-stackables.md`) — it edits the same function and
the same `dueyReceiveDeps` struct. This file assumes that fix has landed.

## Observed

`receiveParcel` (`services/atlas-channel/.../socket/handler/duey_action_receive.go:162`)
rejects with `RECV_NO_FREE_SLOTS` (0x15) on a pure slot count:

```go
if uint32(len(cp.Assets())) >= cp.Capacity() {
    reject(parcelcb.ParcelRecvNoFreeSlotsBody())
```

Two independent defects in that one line:

1. **No merge consideration.** A recipient whose USE compartment is full but who holds a
   half-full stack of the parcel's item is told they have no room, though the quantity would
   merge. Item 2000004's `slotMax` is 100 (confirmed live from atlas-data in this
   environment).
2. **Equipped items are miscounted.** `len(cp.Assets())` counts every asset, and equipped
   items live in negative slots that never consume inventory space. atlas-inventory's own
   `freeSlots` explicitly skips `a.Slot() < 1`
   (`atlas-inventory/.../compartment/processor_accommodation.go:22-34`); this check does not,
   so it over-counts occupancy and can reject a compartment that has room.

## Expected

The parcel is claimable whenever atlas-inventory would actually accept the grant: a free slot
exists, **or** the item is a mergeable stackable whose quantity fully fits an existing stack.

## Root cause

The check reimplements — badly — a decision atlas-inventory already owns.
`compartment.CanAccommodate` / `accommodatesOne`
(`processor_accommodation.go:48-101`) is exactly this verdict, including the equipped-slot
exclusion, the full-merge arithmetic against `GetSlotMax`, and the guards that keep equipment,
bullets and throwing stars from merging. It is exposed at
`POST /characters/{characterId}/inventory/accommodation`
(`compartment/resource.go:27`). atlas-channel simply has no client for it.

## Fix

Call the endpoint instead of counting slots. Do not re-derive the rule in atlas-channel — the
duplicated version is what produced this bug.

| File | Change |
|---|---|
| `services/atlas-channel/atlas.com/channel/compartment/rest.go` | Add the accommodation input/output JSON:API rest models. **In `rest.go`, not a new `accommodation.go`** — task-131's backend audit flagged exactly that split as a FILE-rule violation in the atlas-consumables copy (`docs/tasks/task-131-random-reward-items/audit-backend.json:36`). |
| `services/atlas-channel/atlas.com/channel/compartment/requests.go` | Add `accommodationResource = "characters/%d/inventory/accommodation"` and the `requests.PostRequest` builder, in `requests.go` for the same reason (audit-backend.json:42). |
| `services/atlas-channel/atlas.com/channel/compartment/processor.go` | Add a `CanAccommodate(characterId uint32, templateId uint32, quantity uint32) (bool, error)` processor method over that request. |
| `services/atlas-channel/atlas.com/channel/socket/handler/duey_action_receive.go` | Replace the `len(cp.Assets()) >= cp.Capacity()` check with a `canAccommodate func(characterId uint32, templateId uint32, quantity uint32) (bool, error)` dep; reject with `ParcelRecvNoFreeSlotsBody()` when it returns false. |

Model the client on `services/atlas-consumables/atlas.com/consumables/inventory/` (requests.go
+ its accommodation rest models), which already calls this endpoint — but place the files per
the FILE-rule note above rather than copying its layout.

The ingress already routes it: `^/api/characters/[^/]+/inventory(/.*)?$` → atlas-inventory
(verified against the live `atlas-ingress-routes` ConfigMap in `atlas-pr-1434`). No route or
overlay change.

Note the parcel's stored `itemType` is **not** an input here: `accommodatesOne` derives the
compartment from the template id itself. Keep the existing `getCompartment` dep — the
unique-conflict check still needs the compartment.

Lookup-failure posture: an error from the accommodation call is a refusal, not a permissive
default — reject with `ParcelIncorrectRequestBody()` and log at error level, matching the
posture the unique-conflict fix uses for its atlas-data lookup.

Preserve the existing pre-flight order (accommodation → unique-conflict → parcel state) so the
result arms a player sees do not reorder.

Tests:

- `duey_action_receive_test.go` — `canAccommodate` false announces `RECV_NO_FREE_SLOTS` and
  starts no saga; true proceeds to the unique check; an error announces `INCORRECT_REQUEST`.
- `compartment` package — the request builds the expected path and the output model decodes
  the `accommodated` field.

## Not yet answered

- Not reproduced live before fixing (found by inspection). A live confirmation needs a
  recipient with a full compartment and a partial stack of the parcel's item; worth doing once
  the branch redeploys, but the unit tests are the primary evidence here.

## Resolution

Fixed by **`a7adb9d9c`** — "fix(atlas-channel): defer DUEY_ACTION RECEIVE's slot check to
atlas-inventory". The hand-rolled slot count was replaced by a client for
`POST /characters/{id}/inventory/accommodation`, so both defects (no merge consideration,
equipped items miscounted) are gone with the duplicated logic. Rest models and request builders
were placed in `rest.go` / `requests.go` per the FILE-rule note.

- **Gate**: `tools/verify.sh --quick --base bfa713a5c` → PASS (exit 0, 91 modules). `--quick`
  skips the docker bake, so this is not the pre-PR gate.
- **Review**: `atlas-reviewer`, verdict `APPROVED`, 0 blocking / 0 non-blocking — see
  `reviews/review-duey-preflight-gates.md`. It confirmed the request path matches what
  atlas-inventory serves and the rest models match `accommodation_rest.go` field-for-field.
- **Live re-test**: still outstanding, as this file anticipated — needs a recipient with a full
  compartment and a partial stack of the parcel's item. Unit tests are the current evidence.

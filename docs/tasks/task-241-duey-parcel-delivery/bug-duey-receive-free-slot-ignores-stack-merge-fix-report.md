# Fix report — DUEY_ACTION RECEIVE's free-slot check ignores stack merging

## What I implemented

Replaced `receiveParcel`'s reimplemented free-slot check
(`uint32(len(cp.Assets())) >= cp.Capacity()`) with a call to atlas-inventory's
`CanAccommodate` endpoint, exactly per the brief's `## Fix` inventory:

- `services/atlas-channel/atlas.com/channel/compartment/rest.go` — added
  `accommodationInputRestModel`/`accommodationItemRestModel` (POST body) and
  `accommodationOutputRestModel`/`accommodationResultRestModel` (response),
  modeled on atlas-consumables' `inventory` package but with the single-item
  shape the brief's processor signature calls for. No new file — placed in
  `rest.go` per the FILE-rule note.
- `services/atlas-channel/atlas.com/channel/compartment/requests.go` — added
  `accommodationResource = "characters/%d/inventory/accommodation"` and
  `requestCheckAccommodation`, which builds the `requests.PostRequest` with a
  one-item `Items` slice. Placed in `requests.go` per the FILE-rule note (not
  a new `accommodation.go`).
- `services/atlas-channel/atlas.com/channel/compartment/processor.go` — added
  `CanAccommodate(characterId uint32, templateId uint32, quantity uint32) (bool, error)`
  to the `Processor` interface and `ProcessorImpl`, decoding the response's
  `Accommodated` field via `requests.Provider`.
- `services/atlas-channel/atlas.com/channel/socket/handler/duey_action_receive.go`
  — added a `canAccommodate func(characterId uint32, templateId uint32, quantity uint32) (bool, error)`
  field to `dueyReceiveDeps`, wired it to `compartment.NewProcessor(l,
  ctx).CanAccommodate` in `handleDueyActionReceive`, and replaced the
  slot-count check in `receiveParcel` with a call to `deps.canAccommodate`
  using `*itemId` and `uint32(p.Quantity())`. An error rejects with
  `ParcelIncorrectRequestBody()` and logs at error level (matching the
  unique-conflict fix's posture for its atlas-data lookup); `false` rejects
  with `ParcelRecvNoFreeSlotsBody()` as before.
- Kept `getCompartment` — the unique-conflict check downstream still needs
  the compartment to call `FindFirstByItemId`.
- Preserved pre-flight order: accommodation check → unique-conflict check →
  parcel-state check (unchanged from the already-landed unique-conflict fix).

The parcel's stored `itemType` was not passed as an input to the
accommodation call — `CanAccommodate`/`accommodatesOne` derives the
compartment from the template id itself, per the brief's note.

## Tests

- `services/atlas-channel/atlas.com/channel/socket/handler/duey_action_receive_test.go`
  — replaced the `compFul` fixture field with `accommodated *bool` /
  `accommodatedErr error` on the `dueyReceiveDeps.canAccommodate` stub
  (defaults to `true` when unset, matching a normally-accommodatable
  request). Updated the "no free slot" case to set `accommodated:
  boolPtr(false)` and added a new "canAccommodate error" case asserting
  `ParcelOperationIncorrectRequest` with no saga started, mirroring the
  existing "getItemOnly error" case's posture. All pre-existing cases (happy
  path, meso-only, unique conflict, stackable merge, not-receivable,
  not-addressed-to-me, already-received, getParcel error) pass unchanged.
- `services/atlas-channel/atlas.com/channel/compartment/requests_accommodation_test.go`
  (new) — `TestCanAccommodate` spins up an `httptest.Server`, asserts the
  request is a POST to a path ending `/characters/1/inventory/accommodation`,
  and asserts the decoded `accommodated: true` response is returned.
  `TestCanAccommodateFalse` does the same for a `false` response.

### Commands and output

```
cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./...
```

Full `go test ./...` output showed no `FAIL`/`panic` lines (grepped and
confirmed empty); all packages report `ok` or `[no test files]`. Targeted run:

```
cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/... ./compartment/...
ok   atlas-channel/socket/handler   0.668s
ok   atlas-channel/compartment      0.007s
```

`gofmt -l` on every touched file printed nothing (all clean); `go vet
./socket/handler/... ./compartment/...` printed nothing.

## Files changed

- `services/atlas-channel/atlas.com/channel/compartment/rest.go`
- `services/atlas-channel/atlas.com/channel/compartment/requests.go`
- `services/atlas-channel/atlas.com/channel/compartment/processor.go`
- `services/atlas-channel/atlas.com/channel/compartment/requests_accommodation_test.go` (new)
- `services/atlas-channel/atlas.com/channel/socket/handler/duey_action_receive.go`
- `services/atlas-channel/atlas.com/channel/socket/handler/duey_action_receive_test.go`
- `docs/tasks/task-241-duey-parcel-delivery/bug-duey-receive-free-slot-ignores-stack-merge.md` (the brief itself, was untracked in the worktree; committed alongside the fix)

Commit: `a7adb9d9c` — "fix(atlas-channel): defer DUEY_ACTION RECEIVE's slot check to atlas-inventory"

## Self-review

- Confirmed no `compartment/mock` package exists in atlas-channel that would
  need a `CanAccommodate` stub added — `go build ./...` succeeded cleanly,
  which would have failed if any other type implemented the `Processor`
  interface without the new method.
- Confirmed the ingress route (`^/api/characters/[^/]+/inventory(/.*)?$` →
  atlas-inventory) already covers `/accommodation` per the brief — no
  route/overlay change made.
- Confirmed atlas-inventory's actual route
  (`services/atlas-inventory/atlas.com/inventory/compartment/resource.go:27`)
  and its `AccommodationInputRestModel`/`AccommodationOutputRestModel`
  (`accommodation_rest.go`) field names (`items`, `itemId`, `quantity`,
  `accommodated`, `results`) match what the new channel-side rest models
  send/expect.
- Did not touch `getCompartment` or the unique-conflict logic — out of this
  brief's scope, already fixed by the prerequisite commit.

## Issues or concerns

None. The brief's "Not yet answered" note (no live reproduction) still
applies — this fix has unit-test coverage only, per the brief's own
acknowledgment that live confirmation is a follow-up once the branch
redeploys.

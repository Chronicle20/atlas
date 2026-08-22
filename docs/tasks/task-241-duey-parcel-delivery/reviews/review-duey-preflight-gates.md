# Review — DUEY_ACTION RECEIVE pre-flight gates (unique-conflict + free-slot)

Range: `bfa713a5c..a7adb9d9c`
Commits: `5dabf1e31` (gate unique-conflict on `only`), `a7adb9d9c` (defer slot check to atlas-inventory accommodation)
Requirements: `bug-duey-receive-unique-conflict-on-stackables.md`, `bug-duey-receive-free-slot-ignores-stack-merge.md`

## Scope

Reviewed the full diff (`git diff --stat bfa713a5c..a7adb9d9c`, 22 files) plus, for
cross-service-seam verification, the atlas-inventory files the diff's brief calls by contract
(`services/atlas-inventory/atlas.com/inventory/compartment/resource.go`,
`accommodation_rest.go`) even though those files are unchanged in this range. No other files
were read.

`scope_confirmed`: the range matches the two briefs exactly — one commit per bug, both touching
`duey_action_receive.go` and `dueyReceiveDeps` as the briefs anticipated. No scope mismatch.

## Findings

### 1. `only` flag reaches every atlas-data category, correct JSON tag — PASS

All five WZ item categories now surface `info/only`:

- `equipment/reader.go:115` `Only: info.GetBool("only", false)` → `equipment/rest.go:45` `Only bool \`json:"only"\``
- `cash/reader.go:88` `m.Only = i.GetBool("only", false)` → `cash/rest.go:71` `Only bool \`json:"only"\``
- `etc/reader.go:48` `m.Only = i.GetBool("only", false)` → `etc/rest.go:14` `Only bool \`json:"only"\``
- `setup/reader.go:48` `m.Only = i.GetBool("only", false)` → `setup/rest.go:14` `Only bool \`json:"only"\``
- `consumable/reader.go:56` / `consumable/rest.go:54` — pre-existing, unchanged.

`atlas-channel/data/tradeability/rest.go` carries `Only bool \`json:"only"\`` on all five
`*RestModel`s (`EquipmentRestModel`, `ConsumableRestModel`, `SetupRestModel`, `EtcRestModel`,
`CashRestModel`), each wired through `fields() (bool, int32, bool)` into the shared `extract`
(`rest.go:118-122`), landing in `Model.only` with a `Only() bool` accessor
(`rest.go:19-26`). JSON tags match atlas-data's wire field name (`only`) in every case.

Every atlas-data reader has a matching test asserting both the true case (fixture sets
`only=1`) and the false-default case (fixture omits the node) —
`cash/reader_test.go:637-676`, `equipment/reader_test.go:609-634`,
`etc/reader_test.go:94-119`, `setup/reader_test.go:114-139`. Verified `cash/reader_test.go`'s
`5240001` fixture (imgdir at line 55) genuinely omits the `only` node, so the default-false
assertion is not vacuous.

`data/tradeability/processor_test.go` was extended (not a new file) to assert `Only()` for the
`extract` unit test and, live-round-trip, for two of the five compartments (consumable,
equipment) plus a `TestByIdProvider_OnlyFalse` covering the false-default decode. This satisfies
the brief's "at least the consumable and one non-consumable resource" bar.

### 2. Accommodation request/response wire contract — PASS

atlas-channel (`compartment/requests.go:14`) posts to
`characters/%d/inventory/accommodation`; atlas-inventory
(`compartment/resource.go:27`) routes exactly
`/characters/{characterId}/inventory/accommodation` to `handleCheckAccommodation`. Path match
confirmed.

Field-for-field comparison, atlas-channel `compartment/rest.go` vs atlas-inventory
`compartment/accommodation_rest.go`:

| | atlas-channel (unexported) | atlas-inventory (exported) | Match |
|---|---|---|---|
| input type name | `inventoryAccommodations` | `inventoryAccommodations` | yes |
| input `Items[].ItemId` | `json:"itemId"` | `json:"itemId"` | yes |
| input `Items[].Quantity` | `json:"quantity"` | `json:"quantity"` | yes |
| output type name | `inventoryAccommodations` | `inventoryAccommodations` | yes |
| output `Accommodated` | `json:"accommodated"` | `json:"accommodated"` | yes |
| output `Results[].ItemId/Quantity/Accommodated` | matching tags | matching tags | yes |

JSON:API envelope: `requests.PostRequest` → `MakePostRequest` → `createOrUpdate`
(`libs/atlas-rest/requests/post.go:26`) calls `jsonapi.Marshal(input)`, which requires
`jsonapi.MarshalIdentifier`; `accommodationInputRestModel` implements `GetName`/`GetID`/
`SetID`/`SetToOneReferenceID`/`SetToManyReferenceIDs` (`compartment/rest.go:132-140`), so the
outbound body is a genuine JSON:API document, not a bare struct. atlas-inventory's handler is
registered via `rest.RegisterInputHandler[AccommodationInputRestModel]`, the same decode path
every other JSON:API POST in this service uses — symmetric with the encode side.

`requestCheckAccommodation` (`compartment/requests.go:29-38`) sends exactly one item per call
(`Items: []accommodationItemRestModel{{ItemId: templateId, Quantity: quantity}}`), matching
`CanAccommodate`'s single-item signature (`characterId, templateId, quantity`) — consistent
with how `receiveParcel` calls it (one parcel, one item).

`compartment/requests_accommodation_test.go` round-trips this against an `httptest.Server`
serving a real JSON:API payload (`{"data":{"type":"inventoryAccommodations",...}}`) and asserts
both the POST path and the decoded `Accommodated` bool for true and false cases — this is not
just a unit test against a hand-built Go struct, it exercises the actual JSON:API decode path.

**FILE-rule placement (free-slot brief's explicit call-out)**: confirmed no
`accommodation.go` exists — `find services/atlas-channel/.../compartment -iname
"*accommodation*"` returns only `requests_accommodation_test.go`. The input/output rest models
live in `rest.go` (per audit-backend.json:36) and the request builder + resource constant live
in `requests.go` (per audit-backend.json:42), as the brief required.

### 3. Both new lookups refuse on error, never permissive — PASS

`duey_action_receive.go`:

- `canAccommodate` error (line ~187-191): logs at Error, `reject(parcelcb.ParcelIncorrectRequestBody())`, returns — never proceeds and never treats a lookup failure as "accommodated=true".
- `getItemOnly` error (line ~199-203): logs at Error, `reject(parcelcb.ParcelIncorrectRequestBody())`, returns — never falls back to the old template-equality behaviour and never treats a lookup failure as "not unique".

Both match `tradeability.Processor.Get`'s own documented contract
(`data/tradeability/processor.go:17-18`, `:51-67`): an error is a refusal, distinguished only
for logging (404 → Warn "no entry", non-404 → Error "lookup failed"), but both arms return the
error up to the caller, which — per the two call sites above — always rejects. Confirmed by
tests: `duey_action_receive_test.go` "getItemOnly error" and "canAccommodate error" cases both
assert `RECV_INCORRECT_REQUEST` and `sagaLen: 0`.

### 4. Pre-flight ordering preserved — PASS

`receiveParcel` (`duey_action_receive.go:163-215`): `getCompartment` → `canAccommodate` (free
slots, reject `RECV_NO_FREE_SLOTS`) → `FindFirstByItemId` + `getItemOnly` (unique conflict,
reject `RECV_UNIQUE_CONFLICT`) → `ReceivableAt` check (parcel state, reject
`RECV_INCORRECT_REQUEST`). This is accommodation → unique-conflict → parcel-state, exactly the
order both briefs specify ("Preserve the existing pre-flight order (accommodation →
unique-conflict → parcel state) so the result arms a player sees do not reorder").

### 5. Behavioural assertion, not just reading — PASS

`duey_action_receive_test.go`'s `TestDueyActionReceive` table includes:

- `"unique conflict"`: `compDup: true, itemOnly: true` → asserts
  `ParcelOperationRecvUniqueConflict` announced, implicitly `sagaLen: 0` (default zero value,
  checked against `tc.want.sagaLen`).
- `"stackable already held merges, no rejection"`: `compDup: true, itemOnly: false` → asserts
  `sagaLen: 1`, `invType: inventory.TypeValueEquip`, and (via the `tc.want.reason == ""` branch)
  zero announcements. This is the reported-regression scenario and a real behavioural
  assertion, not a read-and-trust: the `getItemOnly` stub used in this case returns
  `tc.itemOnly` (false), and the test would fail if `receiveParcel` still gated on
  `FindFirstByItemId` alone.

Both cases run through the real `receiveParcel` function with stub deps, not a hand-simulated
substitute — this satisfies "asserted by a test, not just by reading."

### 6. Mechanical fallout — PASS

`character_cash_item_use_karma_test.go`'s five `tradeability.NewModel(...)` call sites were
updated for the new three-arg constructor (`NewModel(tradeBlock, tradeAvailable, only)`), all
passing `false` for `only` — inert for those karma-arm assertions, correctly mechanical.

### Build/test verification

- `go build ./...` in `atlas-channel` and `atlas-data`: clean.
- `go test ./compartment/... ./data/tradeability/... ./socket/handler/...` (atlas-channel):
  pass.
- `go test ./cash/... ./equipment/... ./etc/... ./setup/...` (atlas-data): pass.

## Not evaluable

- atlas-inventory's `CanAccommodate`/`accommodatesOne` internal correctness
  (`processor_accommodation.go`, e.g. the equipped-slot exclusion and merge arithmetic) is
  outside this diff's surface — atlas-channel is explicitly instructed by the brief not to
  re-derive it, only to call it, and this range does not touch atlas-inventory. Not reviewed
  here; would need its own unit if atlas-inventory's logic is in question.
- The free-slot brief's live-reproduction note ("Not reproduced live before fixing... worth
  doing once the branch redeploys") is still open per the brief's own text; this is a
  process/deployment follow-up, not a defect in the diff.

## Verdict rationale

Both fixes trace correctly end-to-end: the WZ `only` flag is read consistently across all five
atlas-data categories with matching JSON tags and reaches atlas-channel's tradeability model;
the accommodation client's path, JSON:API envelope, and field names match atlas-inventory's
server contract exactly; both new lookups fail closed; the pre-flight order is unchanged; and
the exact regression scenario (stackable already held, `only == false`) plus its counter-case
(`only == true`) are both asserted by tests that exercise the real `receiveParcel` function, not
just described. No blocking defects found within scope.

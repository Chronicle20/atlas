# Task 19 review — atlas-maker upstream REST clients

Commit range: `fd7a3c6c1..2451a0c51` (single commit `2451a0c51`).

## Scope

Reviewed the six new client packages (`data/itemmake`, `data/equipment`, `character`,
`skill`, `compartment`, `quest`) under `services/atlas-maker/atlas.com/maker`, plus the
carried-over `crystalband` test. For each client, read the exact upstream `resource.go`
route the client's `requests.go` targets — not the brief's pagination table, which is
proven wrong on `data/itemmake` — to independently verify pagination handling, wire-shape
fidelity, and service token/route correctness. Ran `go build ./...`, `go test ./... -count=1`,
and `go vet ./...` from the module root; all clean, matching the implementer's report.

## Deviation 1 — itemmake pagination (brief's table was wrong)

Verified directly:

- `services/atlas-data/atlas.com/data/itemmake/resource.go`: `GET /data/item-makes`
  (list) calls `paginate.ParseParams`/`MarshalPaginatedResponse` — paginated.
  `GET /data/item-makes/{itemId}` is a single-item unpaginated read, matching
  `itemmake.GetById`'s plain `requests.GetRequest`.
- `services/atlas-maker/.../data/itemmake/processor.go:35`: `GetAll()` uses
  `allItemMakesUrl` + `requests.DrainProvider[RestModel, Model](...)( url, 250, Extract,
  model.Filters[Model]())()`, the exact idiom of
  `services/atlas-channel/.../data/quest/processor.go:49` (bare URL, `DrainProvider`, page
  size 250).
- `processor_test.go:130` `TestGetAllDrainsBeyondOnePage` stubs two pages (50+10 = 60
  recipes) and asserts recipe id 60 (page 2) is present — a real drain proof, not a
  one-page stub dressed up.

Confirmed the deviation is correct and complete. Per-client pagination sweep (checked
each against its real upstream `resource.go`, not the brief's table):

| Client | Upstream route consumed | Paginated? | Client handling | Verdict |
|---|---|---|---|---|
| `data/equipment.GetById` | `GET /data/equipment/{id}` | No (only the sibling `/slots` route is) | single `GetRequest` | correct |
| `data/itemmake.GetAll` | `GET /data/item-makes` | Yes | `DrainProvider`, page 250, two-page test | correct |
| `data/itemmake.GetById` | `GET /data/item-makes/{id}` | No | single `GetRequest` | correct |
| `character.GetById` | `GET /characters/{id}` | No (list routes are, this one isn't) | single `GetRequest` | correct |
| `skill.GetByCharacterId` | `GET /characters/{id}/skills` | Yes | `DrainProvider`, two-page test (`skill:1007` on page 2 recovered) | correct |
| `compartment.GetByType` | `GET /characters/{id}/inventory/compartments?type=N` | No | single `GetRequest` | correct |
| `compartment.CanAccommodate` | `POST /characters/{id}/inventory/accommodation` | N/A (POST) | single `PostRequest` | correct |
| `quest.GetByCharacterId` | `GET /characters/{id}/quests` | Yes | `DrainProvider`, two-page test (`quest:300` on page 2 recovered) | correct |

No client consumes a paginated upstream with a single-shot request. Deviation 1 confirmed
correct, matching the parent's independent finding — **not** an unauthorised deviation.

## Deviation 2 — `compartment.Model.QuantityOf`

`services/atlas-maker/.../compartment/model.go:53-61`:

```go
func (m Model) QuantityOf(itemId item.Id) uint32 {
    var total uint32
    for _, a := range m.assets {
        if a.TemplateId() == itemId {
            total += a.Quantity()
        }
    }
    return total
}
```

This sums **across all matching asset stacks**, not just the first — it does not have the
bug shape design §5 warns about. It is a plain accessor method on `Model` (already the
brief's required `GetByType` return type), adds no new interface method, and is exercised
implicitly wherever `Model.Assets()` is decoded correctly (the `processor_test.go` asset
decode test covers the underlying data it sums over). Given Tasks 20-23 genuinely need a
per-template on-hand total from a compartment snapshot and this is a ~9-line correct
convenience rather than a redesign, it is justified, not speculative YAGNI. **Ruling:
accepted, not blocking.**

## Contract fidelity for Tasks 20-23 — all verified against the brief's exact signatures

- `itemmake.GetAll() ([]Model, error)` / `GetById(item.Id) (Model, error)` — match.
- `equipment.GetById(item.Id) (Model, error)`, `Model.ReqLevel() uint32` — match
  (`data/equipment/model.go`).
- `character.GetById(uint32) (Model, error)`, `Model.Level() byte`, `Model.Meso() uint32`
  — match (`character/model.go`, `character/rest.go` field types `byte`/`uint32` mirror
  atlas-character's own `RestModel` exactly).
- `skill.GetByCharacterId(uint32) ([]Model, error)` — match.
- `compartment.GetByType(uint32, inventory.Type) (Model, error)` and
  `CanAccommodate(uint32, []AccommodationItem) (bool, error)` — match exactly.
- `quest.GetByCharacterId(uint32) ([]Model, error)` — match.

## Service tokens / routes

All six confirmed against the brief's table: `data/equipment`→`DATA`, `data/itemmake`→`DATA`,
`character`→`CHARACTERS`, `skill`→`SKILLS`, `compartment`→`INVENTORY`, `quest`→`QUESTS`
(`grep` sweep of every `requests.go`'s `RootUrlFor` call). Compartment resource path uses
`inventory.TypeValueEquip/Use/ETC` constants via the `inventory.Type` parameter, never a
literal 1/2/4 (`compartment/requests.go:11-13`, `ByType` format string parameterized on
`inventoryType`).

## Multi-tenancy

Every processor threads `ctx` into `requests.Provider`/`requests.DrainProvider`
(`libs/atlas-rest/requests/provider.go`, `paged.go`), the same shared tenant-propagation
path every reference client in the repo uses. No client bypasses it or hardcodes a tenant.

## `libs/atlas-constants` reuse

- `item.Id` used for all item-template ids (equipment, itemmake, compartment
  asset/accommodation types).
- `inventory.Type` used for compartment type, never a raw int.
- `itemmake`'s `reqItem`/`reqEquip`/`reqQuest.state` are `uint32`, matching atlas-data's
  own `RestModel` field types exactly (`services/atlas-data/.../itemmake/rest.go:14-15,42`)
  — atlas-data's own recipe schema uses raw `uint32` here, not `item.Id`/a quest-state
  enum, so this is upstream fidelity, not a missed-constant defect.
- `quest.Model.State` is `byte`, matching atlas-quest's own `quest.State` wire type
  (`services/atlas-quest/.../quest/state.go`), per the report's explicit citation.

## Test quality

Every package has `mock/processor.go` (func-field mock, `var _ Processor = (*ProcessorMock)(nil)`)
and a real `processor_test.go` using `httptest.NewServer` + `t.Setenv(..._SERVICE_URL, ...)`,
asserting decoded fields, a 404→`requests.ErrNotFound` path, and (for the two paginated
clients plus itemmake) a genuine two-page drain proof that inspects a specific late-page
entry rather than just a total count. No `*_testhelpers.go` files; inline `testLogger()`
helper per file, consistent with the existing `reagent`/`crystalband` convention already in
this module.

One gap, non-blocking: the brief's Step 1 asked for "a `CanAccommodate` test ... plus a
per-item `Results` assertion." `compartment/processor_test.go:86`
(`TestCanAccommodateRoundTripsMultipleItems`) asserts the multi-item **request** round-trips
and that the overall `Accommodated == false` is surfaced, but does not assert on the
**response**'s per-item `Results` fields — because `Processor.CanAccommodate`'s signature
(fixed by the brief itself, `(bool, error)`) discards `Results` before it ever reaches a
caller. There is no way to assert per-item results through the public API without adding a
method the brief doesn't ask for. Not blocking — the signature is correct per the brief and
the important behavior (`Accommodated == false` surfacing) is proven — but it means the
`accommodationResultRestModel`'s field-level decode correctness (e.g. a wire-shape typo in
`Results[i].Quantity`) is currently unverified by any test.

## Task 18 addendum

`crystalband/processor_test.go`: `TestCrystalForLevelNotFoundAcrossTenants` added,
mirroring `reagent`'s scoped/not-found pair — seeds a band under tenant A only, reads under
tenant B, asserts `ErrNotFound`. `TestCrystalForLevelIsTenantScoped` and all other existing
tests in the file are byte-identical to before (confirmed by reading the full file; only
one new test function was appended).

## `resp.Body.Close()` / errcheck

No new test file makes a raw `http.Get`/`http.Client.Do` call; all HTTP traffic goes
through the shared `requests` package. The only `.Close()` calls in new test files are
`httptest.Server.Close()` (no error return), so `errcheck`'s `resp.Body.Close()` rule does
not apply here. `golangci-lint`/`errcheck` binaries were not available in this environment
to run directly; verified by source inspection instead (`grep` swept for `http.Get`/`.Do(`
across all six packages' test files — zero matches).

## Build/test verification (independently re-run)

```
go build ./...   # clean
go test ./... -count=1   # all packages ok, no skips
go vet ./...     # clean
```

## Not evaluable

- `golangci-lint`/`errcheck` could not be run directly in this environment (binary not
  present); covered instead by source-level sweep above.
- Tasks 20-23's actual consumption of these clients is out of this task's scope and not
  reviewed here.

## Verdict rationale

No blocking findings. Both deviations are justified and correctly implemented. Contract
signatures, service tokens, multi-tenancy propagation, `atlas-constants` reuse, and the
Task 18 addendum are all verified against source, not asserted. One non-blocking test gap
noted (per-item `Results` field decode is unexercised, though the signature that makes this
so was itself specified by the brief).

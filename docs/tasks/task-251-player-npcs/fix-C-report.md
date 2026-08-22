# Fix round C report — atlas-channel builder + atlas-query-aggregator playernpc/

## Summary

Fixed the three blocking findings assigned to this round from
`audit-backend-consumers.md`, and closed the atlas-channel half of the
Task 18 equipment-decode gap. DOM-31 (atlas-tenants) was left untouched per
the controller's rejection. `services/atlas-player-npcs/` was not touched.

## 1. DOM-01 — `playernpc.Builder.Build()` now returns `(Model, error)`

`services/atlas-channel/atlas.com/channel/playernpc/builder.go`:

- Added `ErrMissingId`, `ErrMissingCharacterId`, `ErrIncoherentPosition` sentinels.
- `Build()` now validates:
  - `id != uuid.Nil` — every deployed player NPC is addressed by this id.
  - `characterId != 0` — a player NPC always mirrors a real character.
  - `rx0 <= rx1` — rx0/rx1 bound the patrol range the NPC walks between; a
    reversed range is not a valid position, and it's the only "position
    coherent" invariant the brief's wording plausibly maps to.
  - Deliberately did not add vacuous checks on world/map/gender/skin/etc.
    (uint types with a legitimate 0 value, or values whose validity depends
    on WZ data this package doesn't have).
- Added `MustBuild()` matching the pattern in `note/builder.go`,
  `world/builder.go`, and 18 other siblings.

Call-site sweep: `grep -rn "playernpc.NewBuilder\|playernpc\.Builder"` across
`services/atlas-channel/` found no call sites outside the package itself —
`rest.go`'s `Extract` builds `Model{...}` directly (bypassing the Builder
entirely) and no test used `.Build()` on this builder before this round.
`go build ./...` for the whole module confirms nothing else needed updating.

## 2. FILE-05 — `EligibilityModel` moved to `model.go`

- New `services/atlas-query-aggregator/atlas.com/query-aggregator/playernpc/model.go`:
  `EligibilityModel`, `NewEligibilityModel`, `NewUnavailableEligibility`
  (moved verbatim, including doc comments).
- `processor.go` loses them; only `Processor`/`ProcessorImpl`/`GetEligibility`
  remain.

## 3. EXT-02 — new `rest_test.go` for query-aggregator's `playernpc/`

The package had no test file. `getBaseRequest` was hardwired to
`requests.RootUrlFor(ctx, "PLAYER_NPCS")` with no seam for `httptest`, so I
added the same `baseURLProvider` var + `SetBaseURLForTest` helper that
`atlas-channel/playernpc/requests.go` and `location/requests.go` already use
(`services/atlas-query-aggregator/atlas.com/query-aggregator/playernpc/requests.go`).

New `rest_test.go`:

- `TestGetEligibility_HappyPath` — real `httptest.NewServer`, asserts the
  eligibility GET carries `characterId`/`mapId`/`worldId` on the query
  string and that the plain (non-JSON:API) JSON body decodes into
  `EligibilityModel` correctly (`Eligible()`/`Reason()`).
- `TestGetEligibility_InfrastructureError` — a 500 response surfaces as a
  non-nil error from `GetEligibility` (not silently swallowed into a false
  eligible), then asserts the fail-closed
  `NewUnavailableEligibility("player npc eligibility unavailable")` a caller
  builds on that error (mirroring `validation/context.go`'s
  `GetPlayerNpcEligibility`) produces an ineligible model with that reason.

This exercises the real HTTP/JSON decode path end-to-end, not just an
interface-level fake.

## 4. Task 18 equipment-decode gap — atlas-channel half

Added `TestRestModel_EquipmentRoundTrip` to
`services/atlas-channel/atlas.com/channel/playernpc/rest_test.go`: builds a
`RestModel` with a non-empty `Equipment` slice, round-trips it through real
`jsonapi.Marshal` → `jsonapi.Unmarshal` (not a struct literal), asserts the
decoded `EquipmentRestModel` slice matches slot/itemId order, then runs it
through `Extract` and asserts the domain `EquipmentModel` slice matches too.

**Result: the decode is not broken.** The test passes cleanly against the
existing `RestModel`/`EquipmentRestModel`/`Extract` code — no bug found, no
broken shape asserted.

## Testing

```
cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./...
```
All packages pass, including `playernpc` (0.024s) and the rest of the
module (pre-existing cached passes unaffected).

```
cd services/atlas-query-aggregator/atlas.com/query-aggregator && go build ./... && go test ./...
```
All packages pass, including the new `playernpc` tests (0.016s–0.017s) and
`validation`/`validation/mock` (which consume `playernpc.NewUnavailableEligibility`
unchanged).

`gofmt -l` on every touched file: no output (all formatted).

## Files changed

- `services/atlas-channel/atlas.com/channel/playernpc/builder.go` —
  `Build() (Model, error)` + `MustBuild()`, three validation sentinels.
- `services/atlas-channel/atlas.com/channel/playernpc/rest_test.go` — added
  `TestRestModel_EquipmentRoundTrip`.
- `services/atlas-query-aggregator/atlas.com/query-aggregator/playernpc/model.go`
  — NEW, `EligibilityModel` + constructors.
- `services/atlas-query-aggregator/atlas.com/query-aggregator/playernpc/processor.go`
  — loses the domain types, keeps `Processor`/`ProcessorImpl`.
- `services/atlas-query-aggregator/atlas.com/query-aggregator/playernpc/requests.go`
  — added `baseURLProvider`/`SetBaseURLForTest` seam (needed for EXT-02's
  `httptest` test; not in the brief's Files list but required to satisfy it).
- `services/atlas-query-aggregator/atlas.com/query-aggregator/playernpc/rest_test.go`
  — NEW, `TestGetEligibility_HappyPath` + `TestGetEligibility_InfrastructureError`.

## Self-review

- Completeness: all three blocking findings fixed; the equipment-decode gap
  closed with a real bug-vs-no-bug determination (no bug).
- Quality: sentinel errors are descriptive; the position-coherence check is
  a real invariant (patrol range), not a vacuous one. Test names/comments
  explain what they exercise and why.
- Discipline: did not touch `services/atlas-player-npcs/` or
  `services/atlas-tenants/`. Did not weaken any existing test. The
  `requests.go` change was the minimal seam needed, copied verbatim in
  spirit from an existing sibling pattern rather than invented.
- Testing: both new test files exercise the real HTTP/JSON(:API) decode
  path via `httptest`, not mocks.

## Concerns

None. Both modules build and test clean in isolation.

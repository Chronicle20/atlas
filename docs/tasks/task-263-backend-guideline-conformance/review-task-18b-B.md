# Review: Task 18b-B — hand-write Transform for 5 forgotten packages

Commit reviewed: `89c396334` ("feat(task-263): add Transform for five DOM-04 forgotten packages")
Method: `git show`/`git diff 89c396334~1 89c396334` only — no working-tree reads (concurrent
implementers share this worktree). Module-local `go build ./... && go test ./...` (scoped to the
touched packages) run to confirm green, per Contract 2 (module-local only, no `tools/verify.sh`).

## Verdict

APPROVED

## 1. Near-miss self-report — independently verified

Implementer reports it initially used `Write` on two pre-existing `rest_test.go` files
(`consumables/cash`, `maps/data/map/info`), overwriting them, then caught it via `git diff` and
restored from `HEAD` before re-appending.

Verified directly with `git diff 89c396334~1 89c396334 -- <file>` for both:

- `services/atlas-consumables/atlas.com/consumables/cash/rest_test.go` — diff is `+63 -0`, entirely
  additive. All three pre-existing tests (`TestRestModelSpecRoundTripsMorphKeys`,
  `TestSpecTypeWireValues`, `TestGetSpecAbsentKey`) are absent from the diff hunk (i.e. untouched,
  byte-identical) and only `TestTransformRoundTrip` is a new addition at the end of the file.
- `services/atlas-maps/atlas.com/maps/data/map/info/rest_test.go` — diff is `+21 -0`, applied as a
  hunk appended after the existing `TestRestModel_ImplementsJSONApiResource` (visible in the diff
  context line `@@ -28,3 +28,24 @@ func TestRestModel_ImplementsJSONApiResource`). Nothing before
  that point is touched.

No pre-existing test function was dropped, renamed, or reworded in either file. Claim confirmed.

## 2. Field parity — per package

- **`channel/mts/listing`** (`rest.go`): all 40 `RestModel` fields assigned from the correspondingly
  named `Model` field; `WorldId` correctly narrows `world.Id` → `byte` (inverse of `Extract`'s
  `world.Id(r.WorldId)`). No field skipped, no cross-wired neighbour.
- **`channel/mts/wish`** (`rest.go`): all `Model`-backed fields (`Id`, `WorldId`, `Serial`,
  `CharacterId`, `ItemId`, `ListingSerial`, `Price`, `Count`, `ExpiresAt`) assigned correctly.
  `RestModel.CreatedAt` has no `Model` counterpart — confirmed (`model.go` carries no `createdAt`
  field, and pre-existing `Extract` never reads `RestModel.CreatedAt` either) — left at zero value
  and explicitly asserted by the test rather than silently dropped.
- **`consumables/cash`** (`rest.go`): `Id`, `SlotMax`, `Spec`, `PetSkills`, `PetSkillAdd` all
  assigned from the matching `Model` field.
- **`cashshop/character`** (`rest.go`): all 29 `RestModel` fields assigned from identically named
  `Model` fields. `Model.equipment`/`Model.inventory` correctly omitted — `RestModel` never carries
  them (confirmed against `RestModel` struct definition). Note (non-blocking, pre-existing, not
  introduced by this commit): `Extract` in the same file never populates `Model.spawnPoint` from
  `RestModel.SpawnPoint` — `Transform` itself is correct (`SpawnPoint: m.spawnPoint`), but because
  `Extract` already zeroes `spawnPoint` on the way in, the round-trip test can never exercise a
  non-zero `SpawnPoint` through this path. This is an existing `Extract` bug, untouched by the diff
  (confirmed via `git diff 89c396334~1 89c396334` — only the `Transform` function is new), so it is
  out of scope for this task but worth flagging for whoever next touches `Extract` in this package.
- **`maps/data/map/info`** (`rest.go`): all 3 fields (`Id`, `TimeLimit`, `ForcedReturnMapId`)
  assigned correctly; trivial full parity.

No dropped field, no wrong-neighbour assignment found in any of the five `Transform` functions.

## 3. `*time.Time` packages — nil/non-nil coherence and no aliasing

- **`mts/listing`** (`EndsAt`): nil `m.endsAt` → nil propagated; non-nil is copied into a freshly
  allocated pointer (`v := *m.endsAt; endsAt = &v`), not aliased. Test
  (`rest_test.go`, `TestTransformRoundTrip`) has both `nil EndsAt` and `non-nil EndsAt` subtests; the
  non-nil subtest asserts `rm2.EndsAt != rm.EndsAt` (pointer identity) in addition to value equality
  via `.Equal`. Judgment (nil = fixed-price sale, no expiry) is coherent with the domain comment.
- **`mts/wish`** (`ExpiresAt`): identical pattern — copy-not-alias, both nil and non-nil subtests,
  pointer-identity assertion in the non-nil case. Judgment (nil = never expires) is stated and
  consistent.

Both packages pass the aliasing check: mutating a non-nil `rm2.EndsAt`/`ExpiresAt` value cannot
reach the source `Model`, and the test proves it structurally (distinct pointer) rather than by
assertion alone.

## 4. Collection copying — `consumables/cash`

`Transform` allocates `spec := make(map[SpecType]int32, len(m.spec))` and copies key/value pairs by
hand (`rest.go`), and `petSkills := make([]string, len(m.petSkills)); copy(petSkills, m.petSkills)`
for the slice. The test goes further than the brief requires: it mutates the returned `rm2.Spec` and
`rm2.PetSkills` in place and asserts the *Model's* own accessors (`m.GetSpec`, `m.PetSkills()`) are
unaffected — a real aliasing check, not just a shape check. Confirmed no aliasing.

## 5. `RestModel.Id` mapped, and asserted in every test

- `mts/listing`: `Id: m.id`; both subtests assert `rm2.Id`.
- `mts/wish`: `Id: m.id`; both subtests assert `rm2.Id`.
- `consumables/cash`: `Id: m.id`; test asserts `rm2.Id`.
- `cashshop/character`: `Id: m.id`; test asserts `rm2.Id`.
- `maps/data/map/info`: `Id: m.id`; test asserts `require.Equal(t, m.Id(), rm.Id)`.

No package leaves `Id` at zero; every test asserts it. Confirmed against the diffs directly
(section 2 above) and the test bodies (section 3/4 quotes and the two remaining files below).

## 6. Test rigor

- `mts/listing`, `mts/wish`: full `reflect.DeepEqual(m, m2)` on a fully-populated, non-zero fixture
  (every field given a distinct non-zero/non-default value in `base`), run twice (once per
  nil/non-nil subtest) — not a spot-check, not an all-zero fixture.
- `consumables/cash`: `reflect.DeepEqual(m, m2)` on a non-zero fixture, plus the explicit aliasing
  mutation checks (section 4).
- `cashshop/character`: fixture in `TestTransformRoundTrip` gives every field a distinct non-zero
  value (`Level: 200`, `Ap: 5`, `X: 100`, `Y: 200`, etc. — checked the full literal in
  `rest_test.go`), then does `reflect.DeepEqual(m, m2)` over the double round trip
  (`rm → Extract → m → Transform → rm2 → Extract → m2`). This is equivalent in strength to a direct
  `rm`-vs-`rm2` field comparison for any field `Extract` and `Transform` treat symmetrically, and the
  one field where that equivalence breaks down (`SpawnPoint`, section 2 note) is a pre-existing
  `Extract` defect, not something introduced or hidden by this test.
- `maps/data/map/info`: 3-field struct, all three set to distinct non-zero values in the fixture,
  `require.Equal(t, m, m2)` — no blind spot possible at this field count.

No package's test would pass against a half-empty `RestModel` or an all-zero fixture.

## 7. Design decision D1 — direct field access

Grepped each new `Transform` function body for `m.<ExportedGetter>()` call patterns on the `Model`
parameter; all matches found were in the pre-existing `Extract` functions (parameter named `m`/`rm`
of type `RestModel`, e.g. `cashshop/character`'s `Extract(m RestModel)` — a different function,
different parameter type, untouched by this diff) — not in `Transform`. Every `Transform` reads
`Model`'s unexported fields directly (`m.id`, `m.worldId`, `m.spec`, etc.). No new accessor was
minted on any `Model`/`Builder` (confirmed via the diffs — `model.go` files are unmodified by this
commit, `git diff --stat` shows only `rest.go`/`rest_test.go`/`handwork-notes.md` touched). D1
satisfied.

## Verification run (module-local, this review only)

```
cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./mts/...        # all ok
cd services/atlas-consumables/atlas.com/consumables && go build ./... && go test ./cash/...  # ok
cd services/atlas-cashshop/atlas.com/cashshop && go build ./... && go test ./character/... # ok
cd services/atlas-maps/atlas.com/maps && go build ./... && go test ./data/map/info/...    # ok
```
All four green.

## Not evaluable

None. The unit is self-contained (five packages, `Transform`/test pairs), and every claim in the
implementer's report was independently checked against the commit diff or a targeted test run.

## Non-blocking notes

- `cashshop/character`'s pre-existing `Extract` does not populate `Model.spawnPoint` from
  `RestModel.SpawnPoint` (section 2). Not introduced by this commit, not in scope to fix here, but
  worth a follow-up ticket since it silently caps the round-trip test's coverage of that one field.

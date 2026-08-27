# Task 15 review — atlas-doors

**Range reviewed:** `38810baa9..75c61ac` (the unit commit is `75c61ac`, "feat(atlas-doors): add
Transform inverses for map data with round-trip tests"; the range also contains `8bfc2e007`,
a docs-only commit recording other batches' gates — not part of this unit and not evaluated here).

**Files touched by the unit commit (`75c61ac`):**
- `services/atlas-doors/atlas.com/doors/data/map/rest.go` (+32)
- `services/atlas-doors/atlas.com/doors/data/map/rest_test.go` (+69)

## 1. Field-by-field inventory vs Transform

**`Portal` struct** (`data/map/model.go:9-16`): `id`, `name`, `portalType`, `x`, `y`, `targetMapId`.

`TransformPortal` (`rest.go:65-74`) maps all six: `Id: strconv.Itoa(int(p.id))`, `Name: p.name`,
`Type: p.portalType`, `X: p.x`, `Y: p.y`, `TargetMapId: p.targetMapId`. PASS — exact inverse of
`ExtractPortal` (`rest.go:45-62`), including the `strconv` round trip for `id`/`Id`.

**`PortalRestModel` struct** (`rest.go:13-20`): `Id`, `Name`, `Type`, `X`, `Y`, `TargetMapId`. All
six appear in `TransformPortal`'s return. PASS.

**`Model` struct** (`data/map/model.go:43-50`): `id`, `returnMapId`, `forcedReturnMapId`, `town`,
`fieldLimit`, `portals`.

`Transform` (`rest.go:170-187`) maps all six: `Id: m.id`, `ReturnMapId: m.returnMapId`,
`ForcedReturnMapId: m.forcedReturnMapId`, `Town: m.town`, `FieldLimit: m.fieldLimit`,
`Portals: portals` (built via a per-element `TransformPortal` loop that propagates errors). PASS —
exact inverse of `Extract` (`rest.go:151-167`).

**`RestModel` struct** (`rest.go:78-85`): `Id`, `ReturnMapId`, `ForcedReturnMapId`, `Town`,
`FieldLimit`, `Portals`. All six appear in `Transform`'s return. PASS.

No field was found on any of the four types that is absent from its Transform. This was verified
directly against this package's own `model.go`/`rest.go` (not assumed equivalent to another
service).

## 2. Hardcoded-value accessors

No accessor method on `Model` or `Portal` returns a hardcoded/constant value (`model.go:18-40`,
`52-74` are all plain field getters). No such case to exclude from Transform/Extract. N/A — no
finding.

## 3. Live mutation test (reviewer-run, independent of implementer's own Step 4a)

Ran a distinct mutation from the implementer's (implementer dropped `TargetMapId`; I mutated
`Name`):

```
sed -i 's/Name:        p.name,/Name:        "MUTATED",/' data/map/rest.go
go test ./data/map/... -run TestTransformRoundTrip -v
```

Result: both subtests failed with a field-level diff (`name:MUTATED` vs `name:tp` for `Portal`,
propagated into `Model` via the portal slice). Reverted with `git checkout -- data/map/rest.go`;
`git diff --exit-code -- data/map/rest.go` returned 0 (clean). PASS.

The implementer's own report (`.superpowers/sdd/plan/task-15-report-atlas-doors.md`) documents a
second, independent mutation (dropping `TargetMapId`) with the same outcome — RED evidence
(`undefined: TransformPortal`/`undefined: Transform`) is also pasted and matches expectations.

## 4. Fixture non-defaults

`rest_test.go`'s `Portal` subtest fixture: `id:7, name:"tp", portalType:6, x:-100, y:200,
targetMapId:999` — all non-zero, and `x`/`y` chosen with a sign difference (`-100` vs `200`) that
would catch an X/Y swap. `Model` subtest fixture: `id:104000000, returnMapId:104000001,
forcedReturnMapId:999999999, town:true, fieldLimit:42`, plus two distinct portals with distinct
ids/names/coordinates/targetMapIds — sufficient to catch a portal-index or field swap. PASS.

`ExtractPortal`'s `rm.Id == ""` branch (defaulting to `id: 0`) is not exercised as a
default-vs-mutation distinction here, but this is not a defaulting behavior in `Extract` proper —
it is jsonapi's empty-ID convention for the zero-value construction path, and the round-trip test
correctly bypasses it by always supplying a real, parseable id. No finding.

## 5. Docs / exemption note

`75c61ac` touches only `rest.go` and `rest_test.go` — confirmed via `git show 75c61ac --stat`. No
`docs/` file is touched by the unit commit, and no exemption note was added (correct: B1 packages
have a `RestModel`, so no `handwork-notes.md` entry is required per `task-15-common.md`). PASS.

(The broader range `38810baa9..75c61ac` also contains `8bfc2e007`, a docs-only commit recording
other batches' gate results — this is not part of the atlas-doors unit and is out of scope for this
review.)

## 6. Extract inventory completeness

```
grep -rn '^func Extract\|^func Transform' services/atlas-doors/atlas.com/doors/data/map/
```

```
rest.go:45:func ExtractPortal(rm PortalRestModel) (Portal, error) {
rest.go:65:func TransformPortal(p Portal) (PortalRestModel, error) {
rest.go:151:func Extract(rm RestModel) (Model, error) {
rest.go:170:func Transform(m Model) (RestModel, error) {
```

Exactly two `Extract*` functions, each with a paired `Transform*` immediately following it. No
`Extract*` left without an inverse. Matches the brief's table exactly (`rest.go:45` /
`rest.go:139` — line numbers shifted slightly post-edit as expected, but the pairing is intact).
PASS.

## Gate evidence (from implementer report, spot-checked)

- RED: `go test ./data/map/... -run TestTransformRoundTrip -v` → `undefined: TransformPortal`,
  `undefined: Transform` (build failure), as required before implementation.
- GREEN: both subtests pass after implementation.
- `tools/lint.sh --check --fmt --go services/atlas-doors/atlas.com/doors` → OK (per report; not
  independently re-run by this review, gate evidence is not re-verified per review scope).

## Not evaluable

None. All items in the reviewer's checklist and the task's requirements were directly verifiable
within this unit's diff and the two source files it depends on (`model.go`, `rest.go`).

## Verdict rationale

Every field of `Model`, `RestModel`, `Portal`, `PortalRestModel` is present in its Transform.
Naming matches FR-2 exactly (`Extract`→`Transform`, `ExtractPortal`→`TransformPortal`). The
round-trip test fixture uses distinct, non-zero values including a sign-differentiated X/Y pair.
An independent reviewer-run mutation reproduces a field-level failure and reverts clean. No docs
files were touched by the unit commit. No `Extract*` is missing a `Transform*` inverse.

No blocking or non-blocking findings.

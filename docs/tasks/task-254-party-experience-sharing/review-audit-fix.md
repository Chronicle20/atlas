# Review — task-254 backend-audit fix round

**Commit under review:** `b53d4bc25` (parent `5acf3f5ef`)
**Source of findings:** `docs/tasks/task-254-party-experience-sharing/audit.md`
**Implementer's report:** `.superpowers/sdd/plan/audit-fix-report.md`

## Scope

`git diff --stat 5acf3f5ef b53d4bc25` touches exactly 5 files:

- `services/atlas-monster-death/atlas.com/monster/monster/processor.go`
- `services/atlas-monster-death/atlas.com/monster/party/rest.go`
- `services/atlas-monster-death/atlas.com/monster/party/rest_test.go`
- `services/atlas-monster-death/atlas.com/monster/monster/information/rest.go`
- `services/atlas-monster-death/atlas.com/monster/monster/information/rest_test.go`

This matches the audit's 4 blocking findings (DOM-28 x2 sites, EXT-01 x2
packages, EXT-02 x2 packages) with no extra files. Scope confirmed — no
mismatch.

## Finding 1 — DOM-28 (`monster/processor.go`)

Rule (audit-checklist.md:121): fallible enrichment/decorator/fallback paths
must degrade loudly via `degrade.Observe(...)`, never a bare
`if err != nil { return fallback }`.

**Diff** (`monster/processor.go:277-285` and `:352-356`):

```go
p.l.WithError(err).Warnf("Unable to locate party for character [%d]; treating as solo.", characterId)
degrade.Observe(p.l, "monster_death.party.lookup", characterId, err)
solos = append(solos, p.soloInputFor(characterId))
continue
```

```go
p.l.WithError(err).Errorf("Unable to locate character [%d] for distributing experience from monster death.", characterId)
degrade.Observe(p.l, "monster_death.character.solo_level", characterId, err)
return SoloInput{CharacterId: characterId, Level: 0}
```

- `degrade.Observe(l logrus.FieldLogger, component string, entityId uint32, err error)` signature (`libs/atlas-rest/degrade/degrade.go:25`) matches both call sites exactly.
- Control flow and fallback values are byte-for-byte unchanged: `p.soloInputFor(characterId)` is still called with the same argument, and `SoloInput{CharacterId: characterId, Level: 0}` is still returned. Only two lines were inserted between the existing log call and the existing fallback statement; nothing was reordered, no branch condition changed, no return value altered.
- New import `github.com/Chronicle20/atlas/libs/atlas-rest/degrade` is genuinely used at both call sites — not dead.
- The added comment explicitly (and correctly) notes `party.ErrEmptySlice` (the ordinary no-party case) is deliberately left as a known/accepted ambiguity, not silently expanded scope.

**PASS** — genuinely resolved, no regression to fallback semantics.

## Finding 2 — EXT-01, `party/rest.go`

Rule (audit-checklist.md:158): target REST model implements
`SetToOneReferenceID` and `SetToManyReferenceIDs`, even as no-ops.

Full file read (`services/atlas-monster-death/atlas.com/monster/party/rest.go`):

- `RestModel.SetToManyReferenceIDs` (lines 75-96) — the pre-existing **real** implementation that parses the `members` to-many relationship into `r.Members` — is untouched, byte-identical to the parent commit. Confirmed by diff: the only change to this function's neighborhood is the new `SetToOneReferenceID` no-op inserted immediately above it (lines 69-73), not a replacement.
- `RestModel.SetToOneReferenceID` (line 73): new no-op, `func (r *RestModel) SetToOneReferenceID(_, _ string) error { return nil }`.
- `MemberRestModel.SetToOneReferenceID` and `MemberRestModel.SetToManyReferenceIDs` (lines 182-183): new no-ops, added where neither existed before.

**PASS** — both setters present on both types; the real `members` handler is preserved intact, not a smuggled-in no-op regression.

## Finding 3 — EXT-01, `monster/information/rest.go`

Diff adds (lines 58-63):

```go
func (r *RestModel) SetToOneReferenceID(_, _ string) error            { return nil }
func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }
```

`RestModel` had no relationship handling before this diff (per the audit); both methods are new. **PASS.**

## Finding 4 — EXT-02, httptest-backed tests

### `monster/information/rest_test.go` — PASS, test is honest

`TestRequestById_RoundTrip` stands up a real `httptest.Server`, serves a
`monsters` JSON:API document with a `relationships.drops` block, drives it
through the real `requestById(...)` decode path, and asserts a populated
struct (`Id`, `Name`, `Hp`, `Experience`, `Level` all checked against
non-zero expected values).

Verified test honesty directly: I checked out `b53d4bc25` into a scratch
worktree, replaced `monster/information/rest.go` with the **parent**
(pre-fix) version while keeping the new test, and re-ran it:

```
rest_test.go:105: requestById: struct *information.RestModel does not implement UnmarshalToManyRelations
--- FAIL: TestRequestById_RoundTrip (0.03s)
```

The test genuinely fails without the fix. **PASS.**

### `party/rest_test.go` — test does not pin the claimed fix (finding)

`TestRequestByMemberId_RoundTrip` stands up an `httptest.Server` serving a
`parties` JSON:API document with a `relationships.members` **to-many**
block and two `included` member resources, drives it through
`requestByMemberId(...)`, and asserts populated `Id`/`LeaderId`/`Members`
(name/level) — not vacuous on its face.

However, I ran the same honesty check as above: checked out `b53d4bc25`
into a scratch worktree, replaced `party/rest.go` with the **parent**
(pre-fix) version — i.e. with `RestModel.SetToOneReferenceID` and both new
`MemberRestModel` methods all removed, leaving only the pre-existing real
`RestModel.SetToManyReferenceIDs` — and re-ran the new test unmodified:

```
=== RUN   TestRequestByMemberId_RoundTrip
--- PASS: TestRequestByMemberId_RoundTrip (0.03s)
```

**The test passes identically with or without the EXT-01 fix for
`party/rest.go`.** Root cause: api2go's `UnmarshalToOneRelations` /
`UnmarshalToManyRelations` interfaces are required per relationship
*kind actually present in the payload*
(`jsonapi/unmarshal.go:242-270` in `github.com/jtumidanski/api2go@v1.0.4`
— confirmed by reading the vendored source), not "any struct with any
relationships block anywhere needs both." The party fixture only carries a
to-many relationship (`members`), which `RestModel.SetToManyReferenceIDs`
already handled *before* this fix commit. Because the fixture never
exercises a to-one relationship, the newly-added
`RestModel.SetToOneReferenceID` and both new `MemberRestModel` no-ops are
never on the path this test walks — the test would have passed just as
well against the pre-fix `party/rest.go`.

This is exactly the "a test that passes either way is a finding, not
coverage" pattern: the production fix (Finding 2, EXT-01 for party) is
real and correct on its own merits (confirmed above by direct file
inspection), but the implementer's report's claim that "the EXT-02 tests
exercise the real decode path... proving you got it right" overstates what
this particular test proves for the party package — it proves the
pre-existing to-many path still decodes correctly, not that the new
no-op setters are load-bearing. A fixture that included even a trivial
to-one relationship (e.g. a `leader` singular relationship, if the real
party service ever emits one) would have been necessary to actually pin
finding 2.

**Disposition:** blocking — not because the production code is wrong (it
isn't), but because the review's job is to confirm EXT-02 is *genuinely*
resolved for party, and the test as written does not discriminate between
"fix present" and "fix absent," so it cannot be relied on to catch a future
regression that drops `RestModel.SetToOneReferenceID` again.

## Other checks

- **Unrelated production behavior changes:** none found. All five files'
  diffs are scoped exactly to the four findings; no drive-by changes.
- **New imports:** `degrade` in `processor.go` (used, needed).
  `context`, `net/http`, `net/http/httptest`, `strings`, `uuid`, `logrus`,
  `tenant` in both `rest_test.go` files — all used by the new tests, none
  dead.
- **Build/test:** `go build ./...` and `go test ./party/... ./monster/...`
  in `services/atlas-monster-death/atlas.com/monster` both pass clean
  (module-local only, no `tools/verify.sh` run per instructions).

## Not evaluable

None — all four findings' fix sites and their test coverage were directly
inspected and, where claims of "genuinely resolved" needed independent
verification (both EXT-02 tests), verified by running the tests against
the pre-fix file version in a disposable scratch worktree (removed after
use; no mutation left behind in the reviewed tree).

## Summary

| Finding | Status |
|---|---|
| DOM-28 (`monster/processor.go`) | Resolved — control flow/fallback values unchanged, `degrade.Observe` correctly wired at both sites |
| EXT-01 (`party/rest.go`) | Resolved — both setters present on `RestModel` and `MemberRestModel`, real `members` handler preserved |
| EXT-01 (`monster/information/rest.go`) | Resolved — both no-op setters added |
| EXT-02 (`monster/information/rest_test.go`) | Resolved — test verified to fail without the fix |
| EXT-02 (`party/rest_test.go`) | **Not genuinely resolved** — test passes identically with or without the fix; fixture never exercises a to-one relationship, so it doesn't pin `RestModel.SetToOneReferenceID`/`MemberRestModel`'s new setters |

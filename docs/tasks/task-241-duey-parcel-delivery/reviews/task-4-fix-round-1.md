# Task 4 — scoped re-review of fix round 1

Range: `cae523317..6879aa7e0` (2 commits: `7449ce973` Task 3 error-mapping fix,
`6879aa7e0` Task 4 fix commit). Scope: the two open blocking findings from
`docs/tasks/task-241-duey-parcel-delivery/reviews/task-4.md`, plus new
breakage introduced in the fix diff itself.

## Finding 1 — `filter[worldId]` silently defaulted to `world.Id(0)` — ADDRESSED

`services/atlas-parcel/atlas.com/parcel/parcel/resource.go` (`handleGetParcels`,
recipientId branch): the default-to-`world.Id(0)` fallback is gone. When
`filter[worldId]` is empty, the handler now returns 400
("filter[worldId] is required") before ever calling `p.GetForRecipient`. No
other code path assigns a default `worldId` in this branch — `worldId` is
only set from the parsed query value.

`resource_test.go` adds `list by recipient missing worldId`, asserting 400 for
`GET /parcels?filter[recipientId]=100&filter[status]=pending` (no
`filter[worldId]`). Verified the test is genuine, not decorative: reverted
`resource.go` (and, to keep the module self-consistent, `processor.go`) to the
pre-fix (`cae523317`) content and re-ran this subtest — it fails 400≠200
(actual 200) against the old code. Restored the fix-round files afterward;
working tree diff against `6879aa7e0` is empty.

The existing `tenant isolation` subtest was also updated to add
`&filter[worldId]=0` to its URL (`resource_test.go:243`), matching the
prescribed fix and keeping that subtest passing under the now-required
parameter, consistent with `resource_test.go:120`'s "list by recipient" shape.

## Finding 2 — wrong error sentinel in `handleGetParcel` — ADDRESSED

`resource.go:154-155`: `errors.Is(err, gorm.ErrRecordNotFound)` replaced with
`errors.Is(err, ErrNotFound)`. `gorm` import is still needed only for the
`*gorm.DB` parameter type elsewhere in the file (`InitResource`), not
reintroduced for error comparison.

Confirmed the seam: `parcel/provider.go`'s `ById` uses `db...First(&e)`,
which raw-returns `gorm.ErrRecordNotFound`. `processor.go:65-73`'s `GetById`
(landed in Task 3's `7449ce973`) maps that to the domain `ErrNotFound` before
returning to the resource layer — so the resource layer's check must, and now
does, match the domain sentinel, not the raw gorm one.

Verified the regression is real: checked out `resource.go` at `7449ce973`
(post-Task-3, pre-Task-4-fix) with the current `GetById` — reproduced
`TestParcelResource/get_by_id_missing` failing 500 instead of 404. This
matches the report's claim that this test "was failing on the branch"
between the two commits. Restored fix-round files afterward; working tree
clean against `6879aa7e0`.

`Processor.resolve` (`processor.go:170`, part of the bundled `7449ce973`
commit) already returned the domain `ErrNotFound` directly (not via a gorm
mapping) prior to this fix round — so the sentinel mismatch was confined to
`GetById`'s call path, and the fix closes it without introducing a second,
looser check that would mask a future regression (no `errors.Is(err,
gorm.ErrRecordNotFound) || errors.Is(err, ErrNotFound)`-style OR was added —
`gorm` is decoupled entirely from the resource layer's error handling now).

## New breakage in the fix diff — none found

- `go build ./...` and `go test ./...` (module-local,
  `services/atlas-parcel/atlas.com/parcel`) both pass cleanly against the
  fix-round HEAD.
- The diff also carries `7449ce973`'s `HasInFlight` world-scoping and
  CAS-write changes (`administrator.go`, `provider.go`,
  `ReceivableByRecipientAnyWorld`) — these are Task 3's own fix-round content,
  already reviewed under Task 3's own review artifact per the given task
  sequencing context, not new findings introduced by the Task 4 fix commit.
  Reviewed only for build/test integrity as a dependency of the resource
  layer; no defect found (`UpdateStatusIfPending` CAS test coverage in
  `administrator_test.go` exercises both the affected-row and
  zero-rows-affected paths).
- No unused imports, no dead code, no widened error matching introduced.

## Verified: the single deliberately-failing test is now green

`TestParcelResource/get_by_id_missing` — the one test the task description
says was pinned failing between the two commits — passes at `6879aa7e0`
(confirmed via `go test ./parcel/... -run TestParcelResource -v`, all 8
subtests PASS). Full module suite (`go build ./... && go test ./...`) is
green.

## Verdict

Both blocking findings ADDRESSED with genuine, verified regression coverage.
No new breakage introduced by the fix diff.

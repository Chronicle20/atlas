# Review: DOM-20 test-conversion commit (task-294)

Commit under review: `a7f11c189` "test(maps): convert one-time-spawn subtests
to table-driven form (DOM-20)", range `e001bc7c4..a7f11c189`.

Brief: `.superpowers/sdd/plan/fix-audit-dom20-brief.md`
Report: `.superpowers/sdd/plan/fix-audit-dom20-report.md`

## Scope

`git diff --stat e001bc7c4..a7f11c189` touches exactly three files, all test
files, matching the brief and report exactly:

```
services/atlas-maps/atlas.com/maps/map/monster/processor_test.go | 217 +++---
services/atlas-maps/atlas.com/maps/map/monster/registry_test.go  | 842 ++++++++-----------
services/atlas-maps/atlas.com/maps/map/processor_test.go         | 234 +++---
3 files changed, 617 insertions(+), 676 deletions(-)
```

No non-test source file was touched. Scope matches the brief.

## Primary check: no assertion weakened, dropped, or made less specific

For each of the 6 converted tests I diffed the pre-image (`e001bc7c4`) against
the post-image (`a7f11c189`) hunk by hunk and traced every original
`t.Errorf`/`t.Fatalf` check to its post-conversion location.

1. **`TestRegistryKeys_AreV2AndDistinct`**
   (`services/atlas-maps/atlas.com/maps/map/monster/registry_test.go:333`) —
   genuine data table, 3 rows (recurring/oneTime/meta). Per-row suffix,
   `:maps:spawn:`, and tenant-id checks all preserved verbatim inside the
   per-row `t.Run`. The pairwise-distinctness check (needs all three values
   at once) is correctly hoisted out of the loop into a `seen` map populated
   per row, then asserted once after the loop — same three pairwise
   comparisons as the original, same failure message shape. PASS.

2. **`TestClaimOneTimeSpawnPoints`**
   (`registry_test.go:444`) — 4-row table with a per-case `assert` closure.
   Verified: `wantLen` check on first claim preserved for all 4 original
   subtests; the "armed field" case's claimed-ID-set, Template, onetimeFired
   timestamp window, and one-time-hash HLEN checks are all present verbatim
   inside its closure; the "recurring-only" case's `Count()==4` check is
   present, and its NextSpawnAt-preservation check is preserved via the new
   `checkNextSpawnAtPreserved` gate — capture-before/assert-after ordering
   relative to the claim call is unchanged from the original (capture before
   `before := time.Now()`, assert after `tc.assert`); the "mixed field"
   case's `claimed[0].Id == 2` check is present; the "unseeded field" case's
   len-0 checks are present via the shared skeleton. The unconditional
   second-claim-returns-0 check that all four original subtests shared is
   still unconditional in the table loop. PASS.

3. **`TestRearmOneTime`** (`registry_test.go:667`) — table of named scenarios
   with a per-case `run` closure (option 2, per brief). All four scenarios'
   call sequences and assertions (`fired field re-arms once`,
   `never-fired field returns false`, `re-armed field fires a fresh full
   batch`, `re-arm leaves the recurring hash untouched`) are copied verbatim
   into their closures with identical error messages and identical read
   order (e.g. the "re-arm leaves the recurring hash untouched" case still
   captures `beforeNextSpawnAt` before the claim/rearm calls). PASS.

4. **`TestRearmOneTime_IsPerFieldKey`** (`registry_test.go:887`) — genuine
   2-row table (`fieldA`/`fieldB`/`labelA`/`labelB`) for the "per channel"
   and "per instance" cases. Both cases' full call sequence (init both
   fields, claim both, rearm A, reclaim A expecting 10, reclaim B expecting
   0) is preserved in the shared loop body with only field/label
   substitution. PASS.

5. **`TestSpawnMonsters_OneTimeBatch`**
   (`services/atlas-maps/atlas.com/maps/map/monster/processor_test.go:1177`)
   — genuine 4-row table. `checkXPositions` gates the X-position/MonsterId
   check that only the original "solo entrant" subtest ran; the anonymous
   `secondPass` struct field carries the two-pass cases' reset/re-seed state
   and `wantSecondCreated`, nil for the two single-pass cases (verified via
   early `return` after the first-pass assertions when `secondPass == nil`).
   All four original subtests' first-pass-count checks, the X/MonsterId
   check, and the second-pass-count checks are present with identical
   expected values (10/8/10/10 first pass, X-check only for solo, second
   pass 0/0 for the two two-pass cases). PASS.

6. **`TestProcessorImpl_Exit_RearmsAndDestroysOnEmpty`**
   (`services/atlas-maps/atlas.com/maps/map/processor_test.go:726`) — 4-row
   table with a per-case `seed` closure. `wantDestroyMsgs` and
   `wantClaimedAfter`/`seededOneTime` gate the destroy-message
   unmarshal/type/mapId checks and the post-Exit re-arm-claim check exactly
   where the original subtests had them (present for the two fired-field
   cases, absent for "never-fired" and "unseeded"). The CHARACTER_EXIT
   message-count-1 check is unconditional in all four original subtests and
   remains unconditional in the shared loop body. PASS.

No case found where a table row's assertion is a lowest-common-denominator
weakening of what its pre-image subtest checked.

### Test-count integrity

Top-level `func Test` count and names, before vs. after, per file (via
`git show e001bc7c4:<file> | grep '^func Test'` vs. `grep '^func Test'
<file>`):

- `registry_test.go`: 14 → 14, identical name list.
- `map/monster/processor_test.go`: 25 → 25, identical name list.
- `map/processor_test.go`: 15 → 15, identical name list.

No test was silently deleted or renamed.

## Secondary check: the 12 non-conversions

Read the report's "Not converted" section and independently confirmed, for
each listed test, that the pre-image function body contains zero `t.Run`
calls (i.e. it was already a single top-level scenario in the pre-image, not
a multi-subtest function the implementer declined to tabulate):

```
TestInitializeForMap_IsIdempotent                        t.Run count: 0
TestFlushTenant_ClearsAllThreeHashes                      t.Run count: 0
TestClaimOneTimeSpawnPoints_ConcurrentFiresExactlyOnce     t.Run count: 0
TestRearmOneTime_ConcurrentTrueExactlyOnce                 t.Run count: 0
TestFlushTenant_ReArmsDisarmedField                        t.Run count: 0
TestSpawnMonsters_OneTimeIgnoresCharacterCount             t.Run count: 0
TestSpawnMonsters_MixedMapUsesRecurringDenominator         t.Run count: 0
TestSpawnMonsters_HiddenPointNeverSpawns                   t.Run count: 0
TestSpawnMonsters_RecurringOnlyRegression                  t.Run count: 0
TestSpawnMonsters_ZeroSpawnPointsLogsBothCounts            t.Run count: 0
TestProcessorImpl_Exit_RearmIsPerFieldKey                  t.Run count: 0
TestProcessorImpl_Exit_LogsRearm                           t.Run count: 0
```

All 12 were, in the pre-image, already single-scenario functions with no
sibling `t.Run` cases — consistent with the report's stated reasoning
(concurrency-race, log-hook, or one-flow-pins-one-requirement tests). The
report's argument that the 5 `TestSpawnMonsters_*` singletons each pin a
qualitatively different requirement/assertion kind (getMonsterMax vs. Count
vs. CountOneTime vs. log-hook) rather than differing only in
input/expected-value shape is consistent with what a `grep`-level read of
each function shows (each asserts a distinct mechanism, not a parallel
input/output pair). None of the 12 look like a dodged genuine table; the
non-conversions are technically sound.

## Verification (module-local, as specified by the brief)

From `services/atlas-maps/atlas.com/maps`:

```
go build ./... && go vet ./... && gofmt -l .
```
— clean, no output from `gofmt -l .`.

```
go test ./... -count=1
```
— all packages `ok`, no failures (full output captured during review).

```
go test ./map/... -count=2
```
— `ok` for `map`, `map/character`, `map/jukebox`, `map/monster`,
`map/timer` across both runs; no flakes observed.

`tools/verify.sh` was intentionally not run per the brief's instruction (a
repo-wide flagless run was already in progress separately).

## Findings

None blocking. No non-blocking findings either — the conversion is faithful
assertion-for-assertion, no test was dropped, and the 12 non-conversions are
each genuinely single-scenario in the pre-image.

## Not evaluable

None. The full review surface (the 3-file diff plus the pre-image of the
same 3 files) was available and reviewed in full; no dependency outside this
diff was needed to judge correctness of a test-only conversion.

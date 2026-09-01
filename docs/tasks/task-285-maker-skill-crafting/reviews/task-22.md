# Review — Task 22: `atlas-maker` weighted reward draw

Range: `ea58e0781..a8864d7` (commit `a8864d710012e57624dbf023c93e0aa1d7bbf010`)
Files: `services/atlas-maker/atlas.com/maker/craft/draw.go`,
`services/atlas-maker/atlas.com/maker/craft/draw_test.go`

## Scope confirmed

Diff stat matches the brief's file list exactly: `craft/draw.go` (71 lines,
new) and `craft/draw_test.go` (123 lines, new), no other files touched.
Reviewed both in full plus the reference implementation
(`services/atlas-reward-pools/atlas.com/reward-pools/reward/processor.go:225-276`)
and `recipe.Reward`'s definition (`recipe/model.go:16-20`).

## 1. `crypto/rand` only — PASS

`grep -rn "math/rand" services/atlas-maker` (repo root) returns exactly one
hit: the doc comment at `craft/draw.go:47`, text only, not an import. `draw.go`
imports `crypto/rand` (`draw.go:5`) and `math/big` (`draw.go:7`). No test-only
`math/rand` import exists in `draw_test.go` — its imports are `atlas-maker/recipe`,
`testing`, and the constants lib (`draw_test.go:3-8`); tests inject deterministic
`roll` values directly into the pure `selectWeightedIndex` rather than stubbing
the RNG.

Both draw sites propagate the `rand.Int` error genuinely:
- `draw.go:58-61` (weighted path) — `err != nil` returns `recipe.Reward{}, err`.
- `draw.go:66-69` (fallback path) — same pattern.
Neither swallows the error or defaults to index 0.

Bound passed to `rand.Int` is `total` (`draw.go:58`, `int64(total)`), which is
`totalWeight(rewards)` — matches the reference (`processor.go:262`) and is
correct: `rand.Int` returns `[0, max)`, so `roll ∈ [0, total)`, matching
`selectWeightedIndex`'s precondition documented at `draw.go:26-27`.

## 2. Distribution correctness — PASS

Hand-checked `selectWeightedIndex` (`draw.go:32-44`) against pool `[70, 25, 5]`
(cumulative `[70, 95, 100]`):
- roll `0` → cumulative after i=0 is 70, `0 < 70` → index 0. Correct (first
  entry, lower bound).
- roll `69` → `69 < 70` → index 0. Correct (first entry, upper bound).
- roll `70` → `70 < 70` false, cumulative after i=1 is 95, `70 < 95` → index 1.
  Correct (second entry, lower bound — the classic off-by-one point, handled
  correctly because `cumulative` is checked strictly-less-than *after* adding
  the current weight).
- roll `94` → index 1. roll `95` → index 2. roll `99` → index 2. All correct
  by the same reasoning.
- roll `100` (== total, out of the documented `[0,total)` domain) falls through
  the loop with no `roll < cumulative` ever true and returns `len(pool)-1 = 2`
  via the documented defensive fallback (`draw.go:40-43`). This exercises the
  unreachable-in-practice branch deliberately and correctly.

Zero-weight entries: a reward with `Prob = 0` contributes `cumulative += 0`,
i.e. a zero-width range, so no roll value can satisfy `roll < cumulative` for
the first time at that index unless a prior entry already claimed that value —
confirmed by `TestSelectWeightedIndexZeroWeightEntriesAreNeverSelected`
(`draw_test.go:67-74`), which sweeps all 100 rolls against pool `[50, 0, 50]`
and asserts index 1 is never returned.

Bound passed to `rand.Int` for the weighted path is `total`
(`draw.go:58`), not `total-1` or `len(pool)` — confirmed above.

## 3. Zero-weight pool fallback — PASS

`draw.go:65-70` mirrors `processor.go:269-276` exactly: when
`totalWeight(rewards) == 0`, draws `n := rand.Int(rand.Reader,
big.NewInt(int64(len(rewards))))` and returns `rewards[n.Int64()]`. Reachable
(the `if total := totalWeight(rewards); total > 0` gate at `draw.go:57` falls
through to it) and tested by `TestDrawAllZeroWeightsSelectsUniformly`
(`draw_test.go:105-123`), which uses pool `[0, 0, 0]` and asserts a result is
returned without error and is a pool member.

## 4. No in-place mutation of the input slice — PASS (blocking class, verified clean)

`totalWeight` (`draw.go:18-24`), `selectWeightedIndex` (`draw.go:32-44`), and
`Draw` (`draw.go:52-71`) only `range` over `pool`/`rewards` and index into them
for reads (`rewards[selectWeightedIndex(...)]`, `rewards[n.Int64()]`). No
`sort.Slice`, no `append` to the parameter, no element assignment
(`rewards[i] = ...`) anywhere in the file — confirmed by direct read of
`draw.go` and a targeted grep for `sort`/mutating index-assignment patterns,
both empty. The doc comment at `draw.go:50-51` states the invariant
explicitly. Safe against the recipe cache handing out its own backing slice.

## 5. No REST/response surface — PASS

`git diff --stat` for the commit touches only `craft/draw.go` and
`craft/draw_test.go`; no `rest.go` or transform file is part of this diff.
`go test ./...` from the module root confirms `atlas-maker/rest` has "no test
files" and was untouched — `Draw`'s signature (`[]recipe.Reward) (recipe.Reward,
error)`) has no marshaling annotations and nothing in this package references
an HTTP handler.

## 6. Test quality — PASS

`TestSelectWeightedIndexAcrossCumulativeRanges` (`draw_test.go:22-47`) injects
exact rolls at every cumulative boundary from the brief's table, including the
three lower-bound cases the brief calls out as the classic off-by-one trap.
Flipping `roll < cumulative` to `roll <= cumulative` in `selectWeightedIndex`
would move the boundary rolls (69, 94→ now caught by 70 too, etc.) enough to
fail this test (e.g., roll `70` would then still read as index 1 correctly by
luck at this exact boundary, but roll `94` would move to index 1 either way —
however, the *lower*-bound assertions, e.g. roll `70` expected index `1`, would
break under `roll <= cumulative` starting from index 0's check since `70 <= 70`
is true, wrongly returning index 0). This is a real off-by-one detector, not a
tautology.

`TestSelectWeightedIndexZeroWeightEntriesAreNeverSelected` sweeps the full
`[0,100)` roll space and would fail immediately if a zero-weight entry could
ever be selected.

`TestDrawReturnsAnEntryFromThePool` / `TestDrawAllZeroWeightsSelectsUniformly`
correctly avoid asserting a distribution (would be flaky against real
`crypto/rand`), limiting themselves to membership checks per the brief's
explicit instruction (`task-22-brief.md:66`).

`TestDrawEmptyPoolReturnsError` confirms `ErrEmptyRewardPool` is returned
(distinguishable, not a panic or zero-value) — `draw.go:53-55`.

## Field-naming / overflow check

`recipe.Reward.Prob` (`recipe/model.go:19`, `uint32`) is used as the weight
throughout `draw.go`, correctly diverging from the reference's `poolItem.Weight`
naming per the brief's note (brief and report both call this out; confirmed by
reading `recipe/model.go`).

Overflow: `totalWeight` sums `uint32` values into a `uint32` accumulator
(`draw.go:18-24`) with no widening. This is not a hardened defense against
overflow, but I do not consider it reachable in practice: real reward pools
are seeded from `recipe/processor.go:106`'s port of atlas-data item-make WZ
fields, and the only committed sample data
(`recipe/processor_test.go:52-54`, weights `70/25/5`) uses small
percentage-like values. To overflow a `uint32` accumulator (wrap past
4,294,967,295) a single recipe's reward pool would need either a handful of
absurdly large individual `Prob` values or an enormous number of entries — no
WZ item-make table remotely approaches that. Stating explicitly per the
task's ask: overflow is not reachable given the real data shape this module
consumes, though the code has no explicit guard if a bogus/malicious data
source ever fed a `Prob` near `uint32` max on multiple entries.

## Build/test re-run (independent)

```
cd services/atlas-maker/atlas.com/maker && go build ./... && go test ./... -count=1
```
All packages `ok`, including `ok atlas-maker/craft 0.008s`. Matches the
implementer's report.

## Not evaluable

- Task 23 (the consumer of `Draw`) does not exist yet in this range, so I
  cannot confirm the recipe cache actually hands `Draw` its live backing
  slice rather than a copy — Task 20's contract is taken on the report's word,
  consistent with `recipe/model.go`'s doc comments ("never sorted") but not
  independently re-verified here since `recipe/*.go` is outside this unit's
  diff.

## Verdict rationale

No requirement from the brief was dropped. The distribution math is correct
at every boundary I hand-checked, the zero-weight fallback is present and
tested, no mutation of the input slice occurs, nothing reaches a REST surface,
and `math/rand` does not appear anywhere including in tests. Tests are
boundary-exact, not tautological. Build and tests independently re-run clean.

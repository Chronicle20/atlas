# Review: Task 2 — `splitMeso` pure split function

Range reviewed: `53b43c59a..47dd62e02` (single commit `47dd62e02`, `feat(atlas-drops): add pure meso split function`).

## Scope

Two new files only, confirmed via `git diff --stat`:
- `services/atlas-drops/atlas.com/drops/drop/split.go` (+51)
- `services/atlas-drops/atlas.com/drops/drop/split_test.go` (+196)

No other files touched. No `go.mod`/`go.sum` changes (checked `git diff --stat` scoped to those globs — empty). Matches the brief's declared file list exactly. `scope_confirmed`: diff matches the unit brief with no drift.

## Requirement-by-requirement

1. **Signature** — `func splitMeso(f field.Model, meso uint32, pickerId uint32, members []party.MemberModel) []Recipient` at `drop/split.go:25`. Matches brief verbatim.
2. **`Recipient` type** — `drop/split.go:11-15`, fields `CharacterId uint32; Amount uint32; Picker bool`. Matches.
3. **Picker always a recipient** — `ids := []uint32{pickerId}` seeds the id list unconditionally (`split.go:26-27`), independent of whether `pickerId` appears in `members`, is online, or has a stale field. Verified by three distinct subtests: `no party` (nil members), `offline member excluded`/`picker included despite offline own record` (`split_test.go:150-156`), `picker included despite stale field on own record` (`split_test.go:158-166`) — all pin `Picker: true, Amount` computed over the full recipient count, not just eligible members.
4. **Non-picker eligibility (online AND all 4 dims match)** — `split.go:31-38`: `if !m.Online() { continue }` then compares `WorldId/ChannelId/MapId/Instance`, all four required (logical OR of `!=` → any mismatch excludes). Exercised by 5 exclusion subtests (`offline`, `different world/channel/map/instance excluded`) each isolating exactly one dimension — good discrimination, not just "some field differs."
5. **Sort ascending by character id, exactly one Picker** — `sort.Slice` at `split.go:41`; `sorted by character id` subtest (picker id 25 in the middle of 20/30) confirms sort is real, not accidental input order. `TestSplitMeso_ExactlyOnePicker` (`split_test.go:180-193`) independently counts `Picker: true` == 1 over a 3-member all-eligible case.
6. **Integer division, remainder not suppressed here** — `share := meso / uint32(len(ids))` (`split.go:44`), no zero-filtering logic present. `remainder discarded` and `meso less than recipient count` (share=0, all three recipients present with `Amount: 0`) both confirm no suppression happens in this unit — correctly deferred to Task 3. `TestSplitMeso_RemainderIsDiscarded` sums to 99, not 100, independently pinning the discard.
7. **Duplicate ids collapsed** — `seen` map at `split.go:27,29-30,39` skips a member id already present (including a second occurrence of a non-picker id). Subtest `duplicate member ids collapsed` (two identical `onField(20)` entries) confirms exactly one `{20,...}` recipient results.

## Test-table fidelity vs. brief

Compared the brief's table (task-2-brief.md lines 67-81, 14 rows) against `split_test.go` case-by-case:

| brief row | present in test file | expected values match |
|---|---|---|
| no party | yes (`split_test.go:33-38`) | yes |
| empty member list | yes (`:39-44`) | yes |
| party of one is the picker | yes (`:45-50`) | yes |
| party of three all eligible | yes (`:51-56`) | yes |
| offline member excluded | yes (`:57-62`) | yes |
| different world excluded | yes (`:63-72`) | yes |
| different channel excluded | yes (`:73-82`) | yes |
| different map excluded | yes (`:83-92`) | yes |
| different instance excluded | yes (`:93-102`) | yes |
| duplicate member ids collapsed | yes (`:103-109`) | yes |
| remainder discarded | yes (`:110-116`) | yes |
| meso less than recipient count | yes (`:117-123`) | yes |
| picker included despite offline own record | yes (`:124-130`) | yes |
| picker included despite stale field on own record | yes (`:131-139`) | yes |
| sorted by character id | yes (`:140-146`) | yes |

All 14 rows reproduced case-for-case, in the brief's order, with identical inputs/expected outputs. Both standalone assertions (`TestSplitMeso_ExactlyOnePicker`, `TestSplitMeso_RemainderIsDiscarded`) present and match the brief's specified inputs (100/10/3-eligible) and expected values (n=1, total=99). No missing or altered subtest — no spec gap here.

Ran (read-only confirmation, not a substitute for the implementer's own run):
```
go build ./... && go test ./drop/... -run TestSplitMeso -v
```
All 14 table subtests + both standalone tests PASS. `gofmt -l drop/split.go drop/split_test.go` produces no output (clean).

## Correctness of the change itself

- Value receivers confirmed on `party.MemberModel` (`party/model.go:29-39`: `Id()`, `Field()`, `Online()` all `func (m MemberModel) ...`) — no mutation risk, consistent with immutable-model convention.
- `field.Model.Instance()` returns `uuid.UUID` ([16]byte, comparable) — `!=` comparison at `split.go:36` is valid and correct, not comparing pointers or requiring a `.Equals` method.
- No panic paths: `len(ids) >= 1` always (picker always seeded), so `meso / uint32(len(ids))` never divides by zero.
- `members == nil` iterates zero times in the `for _, m := range members` loop (Go range-over-nil-slice is a no-op) — the "no party" degrade path is genuinely just the general path with an empty loop, not a special-cased branch, matching the code comment's claim.

## Repo conventions

- No new domain type/alias/constant introduced; reuses `field.Model`, `world.Id`, `channel.Id`, `_map.Id`, `uuid.UUID` from `libs/atlas-constants` and `google/uuid` — consistent with existing usage in the module.
- No cross-service import: `atlas-drops/party` is same-module. The field-match predicate is re-expressed inline rather than imported from atlas-channel's `party.MemberInMap`, with the comment at `split.go` file scope explaining why (service-boundary discipline) — correctly not imported across services.
- No `*_testhelpers.go` file — `member`/`onField` are closures local to each test function (duplicated across `TestSplitMeso`, `TestSplitMeso_ExactlyOnePicker`, `TestSplitMeso_RemainderIsDiscarded` rather than factored into a shared non-test file), which is the brief-sanctioned pattern.
- Comment on `splitMeso` (`split.go:17-24`) describes only in-repo derived behavior (integer division, remainder discarded, picker inclusion rule) — no external/Cosmic citation.
- No `go.mod` change.
- Builder pattern used for both `field.Model` and `party.MemberModel` test construction — no raw struct literals bypassing builders.

## Deviation noted by the implementer

The report flags that Steps 1-2 (write failing test, confirm RED with `splitMeso`/`Recipient` undefined) were not independently observed — both files were authored together since the brief supplied `split.go`'s complete text verbatim, leaving no design decision to derive via a literal RED step. This is a process deviation from the brief's step ordering, not a functional defect: the code delivered matches the brief's Step 3 block verbatim (confirmed by diff), and the test suite as written does fail without `split.go` (removing `Recipient`/`splitMeso` breaks compilation of `split_test.go`, so the tests are not vacuously true — package-level compile failure is a stronger constraint than a mere assertion failure). Non-blocking; recorded as a task-quality note, not a spec-compliance gap.

## Not evaluable

- Task 3's suppression-of-zero-share behavior and the wiring of `splitMeso` into the actual drop-processing handler are out of this unit's surface (not yet landed) — correctly deferred per the brief, not evaluated here.
- Whether `party.MemberModel`/`party.NewMemberBuilder` themselves are correct is Task 1's surface; this review only confirmed their signatures are used as declared (read-only reliance), consistent with the review's stated scope.

## Verdict rationale

Every brief requirement is implemented and pinned by a test that would fail without it (compile-time dependency + per-case assertions). All 14 table rows and both standalone assertions reproduced exactly. No repo-convention violation found. The only note is the implementer's own disclosed process deviation (RED step not independently observed), which does not affect correctness of the delivered artifact.

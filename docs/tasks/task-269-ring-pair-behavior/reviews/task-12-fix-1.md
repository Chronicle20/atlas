# Review: Task 12 fix round 1 — invalidate the ring cache on session destroy (FR-4)

Range: `efd355942..b6f255f71` (1 commit, 2 files, +132/-0).
Module: `services/atlas-channel/atlas.com/channel`.

## Summary

This commit closes the one blocking finding from `task-12.md`: the ring
pair cache was never dropped on session end, an unbounded per-character
memory leak. The fix adds `clearRingsOnDestroy` next to its three siblings
in `session/processor.go`, wires it into `Destroy`, and adds a genuine
white-box test in a new file. I independently fault-injected a no-op body
and confirmed the positive test goes red for exactly the claimed reason,
then restored the file and confirmed `git status`/`git diff` are clean. No
collateral changes to `ring/`, `kafka/`, or `main.go` were found. The
deviation from the brief's named test file location is accepted per the
task instructions (Ruling 34) and verified below to match its stated
justification.

## Priority 1 — does the cache entry actually go away, for the right tenant?

**Verified — real cache-state assertion, not "helper was called."**
`session/ring_hook_test.go:69-88` (`TestClearRingsOnDestroy_NonZeroCharacter_ClearsState`)
seeds the real cache via a real `ring.NewProcessor(...).Populate` call
against an `httptest.Server`, confirms the entry is present via
`GetRingRecords`, calls `clearRingsOnDestroy`, and asserts
`GetRingRecords` now returns zero `Couple` entries. No mock, no spy.

**Fault injection performed independently:** I replaced
`clearRingsOnDestroy`'s body with `_ = ring.NewProcessor(l, ctx)` (a
no-op that keeps the import used, exactly the implementer's reported RED
step) and reran:

```
--- FAIL: TestClearRingsOnDestroy_NonZeroCharacter_ClearsState (0.02s)
    ring_hook_test.go:86: after clearRingsOnDestroy: want no cached ring
    records, got {Couple:[{PairCharacterId:200 PairCharacterName:Partner
    OwnSN:1111 PairSN:2222}] Friend:[] Marriage:[]}
--- PASS: TestClearRingsOnDestroy_ZeroCharacter_NoOp (0.00s)
```

This matches the implementer's report verbatim. Restored the file
(`cp /tmp/processor.go.bak session/processor.go`), reran
`go test ./session/... -run TestClearRingsOnDestroy -v` — both green — and
confirmed `git status --porcelain` shows only unrelated untracked
review-artifact paths, no diff on any tracked file.

**Tenant threading — verified, not defaulted.** `clearRingsOnDestroy`
(`session/processor.go:495-499`) takes `p.ctx` from `Destroy`'s call site
(`session/processor.go:430`, `p.ctx` is the `ProcessorImpl`'s own field, not
a fresh/default context) and passes it straight into
`ring.NewProcessor(l, ctx).Invalidate(characterId)`. `Processor.Invalidate`
(`ring/processor.go:116-119`) derives the tenant exclusively via
`tenant.MustFromContext(p.ctx)` — no default path exists; a context without
a tenant would panic, not silently invalidate the zero-tenant. The two new
tests each construct a fresh, unique tenant via `tenant.Create(uuid.New(),
...)` (`ring_hook_test.go:29-36`, mirroring `position_hook_test.go`'s
pattern) specifically so cross-tenant leakage would be visible, matching
the class of bug the original Task 12 review caught by fault-injecting a
dropped tenant id. I did not re-fault-inject the tenant path here since
`Invalidate`/`Populate`/`GetRingRecords` are unchanged code already covered
by that fault injection in the Task 12 review; this commit only adds a new
caller of the same, already-tenant-safe `Invalidate` method.

## Priority 2 — scope discipline (Ruling 32): Destroy only, no map-transfer clear

`clearRingsOnDestroy` is wired into `Destroy` only
(`session/processor.go:430`), the same funnel point (logout, disconnect,
timeout, channel change) as the three siblings, immediately above their
matching comments. The helper's own doc comment
(`session/processor.go:485-494`) explicitly states the map-transfer
exclusion and why: "an intra-channel map transfer must NOT drop the entry
... doing so per map change would defeat Task 12's Populate call site ...
and reintroduce a per-map-change refetch."

Confirmed by diff scope, not just comment text: `git diff --stat
efd355942..b6f255f71 -- kafka/ ring/ main.go` is empty — no map-transfer
consumer (`kafka/consumer/map/consumer.go`), no `ring/` package file, and no
`main.go` was touched by this commit. The map-transfer path this fix was
warned against touching is untouched.

## Priority 3 — no collateral damage

- `git diff --stat efd355942..b6f255f71` touches exactly
  `session/processor.go` (+24/-0) and the new
  `session/ring_hook_test.go` (+108/-0). No other file in the commit.
- `main.go`'s `ring.EvictTenant(tid)` registration, the deleted
  `ring/cache.go` `init()`, `Processor.Invalidate`'s existing coverage, the
  `cd.Rings` writer-site test (`socket/writer/character_data_test.go`), and
  the I/O-free contract on `GetRingSet`/`GetRingRecords` are all outside
  this diff's file list — confirmed untouched.
- **I/O-free contract check on what `clearRingsOnDestroy` calls:**
  `Processor.Invalidate` (`ring/processor.go:116-119`) is
  `tenant.MustFromContext` + `getRingCache().invalidate(t.Id(),
  characterId)` — a pure in-memory map delete, no REST call, no
  `upstreamFn` reference. `Populate` (`ring/processor.go:101-113`) remains
  the only method in the package that calls `upstreamFn`. Verified by
  reading the full `ring/processor.go` body; `upstreamFn` appears exactly
  once, inside `Populate`.
- `go build ./...` and `go test ./session/... ./ring/...` both clean
  (reproduced independently, not just trusted from the report).

## Priority 4 — the accepted deviation (Ruling 34), verified not re-litigated

New file `session/ring_hook_test.go` is `package session` (white-box),
matching `battleship_hook_test.go`/`aran_combo_hook_test.go`/
`position_hook_test.go`. Structurally it follows
`position_hook_test.go`'s exact shape: a per-test-file `*TestTenant`
helper constructing a fresh tenant (`ringHookTestTenant` mirrors
`positionHookTestTenant`), a positive "ClearsState" test and a negative
"ZeroCharacter_NoOp" test with matching naming convention
(`TestClear<X>OnDestroy_NonZeroCharacter_ClearsState` /
`TestClear<X>OnDestroy_ZeroCharacter_NoOp`), and a top-of-file comment
explaining why this state needed direct seeding rather than a mock seam.
No coverage was left in `processor_test.go` (confirmed: `git diff
efd355942..b6f255f71 -- session/processor_test.go` is empty) — no
duplicate or conflicting test exists.

## Doc-comment / FR-4 citation check

`clearRingsOnDestroy`'s doc comment (`session/processor.go:485-494`) cites
"PRD FR-4" explicitly and states WHY the state cannot outlive the session
("The cached ring pair halves cannot outlive the character's presence on
the channel: logout, disconnect, timeout, and channel change all funnel
here"), matching the style of `clearBattleshipOnDestroy` ("FR-5.1"),
`clearAranComboOnDestroy` ("task-217 design.md §3.4"), and
`clearLastPositionOnDestroy` (task-250 design.md, inherited from its
sibling's comment). The `Destroy` call-site comment
(`session/processor.go:422-428`) additionally cites the PRD path
(`docs/tasks/task-269-ring-pair-behavior/prd.md:93-95`), one better than
some siblings.

## Non-blocking

None found beyond what the brief already named as acceptable
(`kafka/consumer/session/consumer.go:217`'s untested `Populate` wiring,
which this commit correctly leaves untouched per its own instructions).

## Not evaluable

None. The full diff surface (2 files, both read in full) and its one
direct dependency (`ring.Processor.Invalidate`/`GetRingRecords`, already
verified in the Task 12 review and re-read here) were both within scope
and evaluable.

## Verification performed

- `git diff --stat efd355942..b6f255f71` and full hunk read of both files.
- `go build ./...` — clean.
- `go test ./session/... ./ring/...` — clean.
- Fault-injected `clearRingsOnDestroy` to a no-op body (keeping the `ring`
  import used); reran `go test ./session/... -run TestClearRingsOnDestroy
  -v`; confirmed the positive test fails with the exact message reported
  by the implementer, the negative test still passes (expected, since it
  asserts non-interference with an unrelated character, which a no-op also
  satisfies).
- Restored `session/processor.go` from a pre-mutation copy; reran the same
  test command green; confirmed `git status --porcelain` shows only
  pre-existing untracked review-artifact paths, no diff on any tracked
  file.
- Confirmed `git diff --stat efd355942..b6f255f71 -- kafka/ ring/ main.go`
  is empty (no map-transfer or Task-12-surface collateral change).
- Confirmed `git diff efd355942..b6f255f71 -- session/processor_test.go`
  is empty (no duplicate coverage left in the brief's originally-named
  file).

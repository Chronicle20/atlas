# Review — Task 7: `atlas-monsters` HP-threshold detonation and `SelfDestruct` processor API

Commit range: `7526324eb..c5a1da3d0` (single commit `c5a1da3d0`)
Brief: `.superpowers/sdd/plan/task-7-brief.md`
Report: `.superpowers/sdd/plan/task-7-report.md`

## Scope

`git diff --stat 7526324eb..c5a1da3d0`:

```
docs/tasks/task-253-self-destructing-mobs/design.md          |  18 +-
services/.../monster/model.go                                |   7 +
services/.../monster/processor.go                             | 108 +++++-
services/.../monster/self_destruct_registry_test.go            |  44 +++
services/.../monster/self_destruct_test.go                     | 369 ++++++++
```

Matches the brief's file list plus one disclosed out-of-brief fix
(`model.go` — `DamageLeader` panic). No files outside `atlas-monsters` /
`docs/tasks/task-253-self-destructing-mobs/design.md` were touched. Scope
confirmed.

## Findings

### 1. API shape for tasks 8/9/10 — PASS

- `type SelfDestructTrigger string` with `TriggerThreshold`, `TriggerTimer`,
  `TriggerContact` — `processor.go:88-94` — exact match to brief.
- `Processor.SelfDestruct(uniqueId uint32, characterId uint32, trigger SelfDestructTrigger)`
  added to both the `Processor` interface (`processor.go:68`) and
  `*ProcessorImpl` (`processor.go:1821`) — exact signature match. Any of
  task 8 (DoT), task 9 (timer sweep), or task 10 (SELF_DESTRUCT command
  handler) can call this directly with an appropriate trigger constant and a
  `characterId` of `0` when there is no attributable actor.
- `selfDestructFrom(m Model, characterId uint32, action byte, trigger SelfDestructTrigger)`
  present at `processor.go:1841` matching the brief's signature, used both by
  `damageCore`'s threshold check and by `SelfDestruct`.
- `monsterInformation(monsterId uint32) (information.Model, error)` present
  at `processor.go:107`, and `Kill` was refactored (not just left alone) to
  route through the same seam (`processor.go:1811`), removing the duplicate
  inline `testInformationLookup` branch — matches Step 3 exactly.
- FR-2.4 (DoT ticks trip the threshold) is not this task's job — the PRD
  requires DoT ticks to route through the shared `damageCore` threshold
  check, which task 7 already wires; task 8 does not need a new
  `SelfDestructTrigger` value for that path. Confirmed no 4th trigger
  constant was expected here (design.md's "four trigger paths" =
  threshold/timer/contact/DoT-via-threshold, not four distinct constants).

### 2. Spec table — "ordinary kill wins" — PASS

`damageCore` (`processor.go:604-676`):

```go
if killed {
    p.finalizeKill(last.Monster, last.CharacterId, isBoss, revives, DeathTypeFadeOut)
    return
}
if !killed && sd.OnHpThreshold() && int64(last.Monster.Hp()) <= int64(sd.Hp()) {
    p.selfDestructFrom(last.Monster, last.CharacterId, sd.Action(), TriggerThreshold)
    return
}
```

The ordinary-kill branch is structurally first and returns before the
threshold check ever runs — an attack that reaches 0 HP can never also take
the self-destruct branch, regardless of where the threshold sits relative to
0. `DeathTypeFadeOut == 1` (`kafka.go:64`), matching the spec table's
"ordinary kill wins → DeathType 1, fade-out, ordinary path" row. Verified by
running the full 8-row table test live (see §5) — the "ordinary kill wins"
and both "no block" regression rows pass with `DeathType == 1`.

### 3. Carry-forward item 1 — concurrent-kill `-race` test — PASS

`self_destruct_registry_test.go:98-141`,
`TestRegistrySelfDestructConcurrentCallersExactlyOneWins`: 50 real
goroutines (`go func() { ... }()`, `wg.Add(50)`/`wg.Wait()`) each call
`r.SelfDestruct(ten, m.UniqueId())` concurrently on one monster, tallying
`Killed`/error counts with `atomic.AddInt64`, and asserts `errCount == 0` and
`killedCount == 1`.

Ran it directly:

```
$ go test ./monster/ -race -run 'TestRegistrySelfDestructConcurrentCallersExactlyOneWins' -v
=== RUN   TestRegistrySelfDestructConcurrentCallersExactlyOneWins
--- PASS: TestRegistrySelfDestructConcurrentCallersExactlyOneWins (0.58s)
PASS
```

Committed, genuinely concurrent, passes under `-race`.

### 4. Carry-forward item 2 — design.md D2 rationale-only rewrite — PASS

Diff of D2 (`design.md:288-306`) replaces only the rolling-deploy rationale
paragraph. Confirmed by reading the full D2 section
(`design.md:262-306`): the DECISION code block
(`present := rm.SelfDestruction.Hp > -1 || rm.SelfDestruction.RemoveAfter > -1`),
"Alternative A — derive presence from the all-zero struct" (rejected, still
present, unedited), and "Alternative B" are all byte-identical to before this
commit. The new paragraph corrects an arithmetic error (`0 > -1` is true, not
false) and narrows the actual blast radius (`OnTimer()` stays false at
`hp==0`; only `OnHpThreshold()` coincides with the ordinary kill point) — it
does not adopt Alternative A anywhere. No pattern-match (`action != 0 || ...`)
logic exists anywhere in this diff or in `information/model.go`.

### 5. Presence predicate unchanged everywhere — PASS

`information/model.go:42-48` (not touched by this diff — confirmed via
`git diff` file list, which does not include it) still reads:

```go
func (s SelfDestruction) OnHpThreshold() bool { return s.present && s.hp > -1 }
func (s SelfDestruction) OnTimer() bool       { return s.present && s.hp <= -1 }
```

`Present()` (`present` field) is set only from `NewSelfDestruction`
(task 4), not touched here. `damageCore` and `SelfDestruct` both call
`sd.Present()` / `sd.OnHpThreshold()` through the `information.Model`
returned by `monsterInformation` — no inline reimplementation of the
predicate anywhere in `processor.go`.

### 6. No new `packet-audit gate-lint` site — PASS

Ran `go run ./tools/packet-audit gate-lint --check`: 38 pre-existing sites
reported, all under `libs/atlas-packet/{character,note,model}/...`. None in
`services/atlas-monsters` or in any file this commit touches. This task's
diff contains no `MajorVersion()` comparisons at all.

### 7. Out-of-brief `DamageLeader` panic fix — PASS, correctly scoped

`model.go:225-243`:

```go
func (m Model) DamageLeader() uint32 {
	index := -1
	for i, x := range m.damageEntries { ... }
	if index == -1 {
		return 0
	}
	return m.damageEntries[index].CharacterId
}
```

- Genuine pre-existing bug: `DamageLeader()` on a `Model` with zero
  `damageEntries` indexed `m.damageEntries[-1]` and panicked. The report's
  TDD trace (`self_destruct_test.go` `TestSelfDestructRejects/valid_target`,
  panic at `model.go:237` before the fix) is consistent with what a git-blame
  of the pre-change function would produce, and I independently confirmed
  the fixed code is what's now in the tree.
- `0` is the correct sentinel: `selfDestructFrom` already treats
  `killerId == 0` as "credit nobody" (comment at `processor.go:1858` — FR-6.3,
  "atlas-monster-death tolerates ActorId 0"), and `Kill`'s existing MaxUint32
  damage-core path always has at least one damage entry by the time
  `DamageLeader` is reachable there, so this is a strictly additive case, not
  a behavior change for any existing caller.
- Verified both existing call sites: `processor.go:696`
  (`last.Monster.DamageLeader() == characterId`, only reachable after at
  least one `ApplyDamage` in the same `damageCore` invocation, so
  `damageEntries` is never empty there) and `processor.go:1873`
  (`selfDestructFrom`'s own new call, the one that needed the fix).
  Neither existing caller's behavior changes.
- `TestSelfDestructNoDamageEntriesReportsNoKiller` exercises exactly this
  path and passes (`ActorId == 0`), confirming the fix works end-to-end
  through the new `SelfDestruct` API, not just at the unit level.

Judgment: correct, minimal, and safely scoped — not a scope violation given
it was discovered by and is required for this task's own mandated test
coverage (FR-6.3 damage-leader fallback on an untouched mob).

## Verification run (independent of the report)

```
$ cd services/atlas-monsters/atlas.com/monsters
$ go build ./...                                                    # clean
$ go test ./monster/ -run 'SelfDestruct|Threshold' -v                # all PASS (13 subtests)
$ go test ./monster/ -race -run 'TestRegistrySelfDestructConcurrentCallersExactlyOneWins' -v  # PASS
$ go vet ./...                                                       # clean
$ gofmt -l .                                                          # clean
$ go run ./tools/packet-audit gate-lint --check                      # 38 pre-existing sites, none new
```

## Not evaluable

- Whether task 8 (DoT), task 9 (timer sweep), and task 10 (SELF_DESTRUCT
  command) actually call `Processor.SelfDestruct` correctly is out of this
  unit's surface — those tasks have not landed yet on this branch as of
  `c5a1da3d0`. This review only confirms the API shape they will consume.
- Full-module `go test ./...` (all packages, not just `monster`) was not
  re-run here; the report's log for it (`Tested` section) was taken as
  reported since it is a mechanical build-gate concern already covered by
  the targeted `-run` invocations and `go vet`/`gofmt` above, and repo-wide
  verification is a separate gate (`tools/verify.sh`), not this reviewer's job.

## Verdict

APPROVED. All four dispatch-mandated items were independently verified, not
just claimed. The spec table's "ordinary kill wins" case is structurally
enforced, not incidental. The disclosed out-of-brief `DamageLeader` fix is
correct, minimally scoped, and required by the task's own mandated test
coverage.

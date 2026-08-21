# Review: Task 5 — `Registry.SelfDestruct` atomic transition

**Commit range:** `29bf5bc65..4c4a4c3f4` (single commit `4c4a4c3f4`)
**Brief:** `.superpowers/sdd/plan/task-5-brief.md`
**Report:** `.superpowers/sdd/plan/task-5-report.md`

## Scope

`git diff --stat 29bf5bc65..4c4a4c3f4`:

```
services/atlas-monsters/atlas.com/monsters/monster/registry.go                | 33 ++++++++
services/atlas-monsters/atlas.com/monsters/monster/self_destruct_registry_test.go | 95 ++++++++++++++++++++++
2 files changed, 128 insertions(+)
```

Matches the brief's file list exactly (`registry.go` new method after `ApplyDamage`, new test file). Scope confirmed — no unrelated files touched.

**Worktree hazard noted, not attributable to this unit:** at the time of review the working tree had *uncommitted* WIP touching `monster/producer.go`, `monster/processor.go`, `monster/processor_catch.go`, `monster/kafka.go` (a `deathType byte` parameter added to `destroyedStatusEventProvider`/`killedStatusEventProvider` without updating both call sites), which made `go build ./...` fail at HEAD as found. This is unrelated to task 5's diff — confirmed by `git stash -u` isolating task 5's commit alone, at which point `go build ./...` and `go test ./monster/ -run 'TestRegistrySelfDestruct' -v -race` both passed cleanly. The stash was popped afterward and the tree left exactly as found (`git status --short` shows the same four modified files, no task-5 files touched). Flagging this so it isn't silently absorbed by a later reviewer's "build is red" observation — it predates and is orthogonal to task 5.

## Findings

### 1. PASS — `SelfDestruct` matches the brief's interface and doc contract exactly

`registry.go:492-524` (new). Signature, doc comment, and body match the brief's Step 3 code block verbatim. `Killed` is derived from `cur.Hp > 0` captured *before* zeroing, inside the `r.reg.Update` closure — this is the correct CAS shape (see finding 2).

### 2. PASS — the CAS/locking shape is a genuine optimistic-lock transaction, not a check-then-set race

`r.reg.Update` (used by both `ApplyDamage` and `SelfDestruct`) is `TenantRegistry[K,V].Update` in `libs/atlas-redis/tenant_registry.go:145-191`. It wraps the read-modify-write in a Redis `WATCH`/`MULTI`/`EXEC` transaction (`r.client.Watch(ctx, txFn, rk)`) and retries up to `updateMaxRetries` times on `goredis.TxFailedErr` (`tenant_registry.go:180-190`), re-reading current state and re-invoking `fn` on each retry. This primitive already has a real-contention proof in the library (`TestTenantRegistry_Update_RetriesOnContention`, `libs/atlas-redis/tenant_registry_test.go:292-333`, which uses an interloper write between `Get` and `EXEC` to force a genuine `TxFailedErr` and asserts the retry re-observes the interloper's write). `SelfDestruct` is not introducing a new locking primitive — it reuses the same primitive `ApplyDamage` already relies on, per the brief's "pattern to copy" instruction.

The closure correctness: `transitioned` is declared outside the closure and reassigned on every invocation (including retries). Because `Update` only returns after a transaction actually commits, the value of `transitioned` visible after `Update` returns is always the one set by the *committing* invocation, which observed the true current state. Two concurrent `SelfDestruct` calls on the same key: whichever transaction commits first sees `Hp > 0` → `Killed=true`; the loser's `EXEC` fails, it retries, re-reads `Hp == 0` → `Killed=false`. This is exactly the exactly-once shape the brief requires — verified independently (see finding 3).

### 3. FINDING (non-blocking) — the committed test suite proves idempotency under sequential re-invocation, not exactly-once under actual concurrent contention

The three committed tests (`self_destruct_registry_test.go`) are all sequential: `TestRegistrySelfDestructTransitionsOnce` calls `SelfDestruct` twice in series on the same goroutine, `TestRegistrySelfDestructLeavesDamageEntries` and `TestRegistrySelfDestructUnknownMonster` each call it once. None spawn goroutines or exercise the `WATCH`/`TxFailedErr` retry path this method's exactly-once guarantee actually depends on. This matches the brief's Step 1 spec verbatim — the brief itself only specified sequential tests — so this is not an implementer deviation from the brief, but it is a real gap given the review directive that tasks 7-10 depend on the concurrency guarantee.

To close the gap I ran an ad-hoc (uncommitted, not part of this review's edits) probe: 50 goroutines calling `r.SelfDestruct(ten, m.UniqueId())` concurrently against the same monster, `-race`, 5 repetitions. Result: exactly 1 `Killed==true` out of 50 on every run, no race detected. This confirms the implementation is correct under real contention — the finding is a coverage gap in the committed suite, not a defect in the code. Recommend a follow-up (or amendment to this task) adding a goroutine-based `-race` test asserting `sum(Killed) == 1` across N concurrent callers, so the exactly-once guarantee that tasks 7-10 build on is pinned by a test in this package rather than only provable by inspection + a reviewer's ad-hoc probe.

### 4. PASS — damage entries and damage leader are left untouched by a detonation

`SelfDestruct`'s closure only mutates `cur.Hp`; it never touches `cur.DamageEntries`. `fromStored` (`registry.go:172-207`) aggregates `DamageEntries` unmodified into the returned `Model`. `TestRegistrySelfDestructLeavesDamageEntries` exercises this against a real `ApplyDamage`-populated entry and asserts both the single entry and `DamageLeader() == 777` survive — this is a genuine assertion of the "detonation is not damage" requirement (design D3), and independently verified by reading `fromStored`.

### 5. PASS — error handling matches the existing `ApplyDamage`/registry convention

Any error from `r.reg.Update` (not-found or otherwise) is collapsed to the package sentinel `errMonsterNotFound` (`registry.go:513`), identical to the established pattern at `registry.go:477` (`ApplyDamage`) and elsewhere in the file (`registry.go:326,349,589,777,840`). `TestRegistrySelfDestructUnknownMonster` asserts a non-nil error and `Killed == false` without panicking — verified to actually pass against the built code (see Verification below), not just trusted from the implementer's report.

### 6. PASS — `DamageSummary` zero-value fields match the brief

`CharacterId` and `VisibleDamage` are left at their Go zero values (0) in the returned `DamageSummary{Monster: m, Killed: transitioned}` literal — matches the brief's explicit "`CharacterId` is 0 and `VisibleDamage` is 0 — a detonation is not damage" requirement. Confirmed by reading `DamageSummary`'s field list (`model.go:36-43`) — no additional required fields are silently defaulted incorrectly.

### 7. PASS — no raw-comparison / presence-predicate pattern issues in scope

This method operates purely on `cur.Hp`; it does not touch `information.SelfDestruction` or introduce an `action != 0` pattern-match. Consistent with the standing ruling that the presence predicate belongs to task 4, not this registry method. No packet-audit gate-lint sites are touched by this diff.

## Verification performed

- `git diff --stat` and full hunk read of both changed files (128 lines total — read whole, appropriately, given the size).
- Read `TenantRegistry.Update`'s WATCH/MULTI/EXEC + retry implementation and its own contention-proof unit test, since `SelfDestruct`'s correctness is entirely dependent on that contract.
- Isolated task 5's commit from unrelated worktree WIP via `git stash -u` / `git stash pop` (tree verified unchanged afterward) and confirmed `go build ./...` and `go test ./monster/ -run 'TestRegistrySelfDestruct' -v -race` pass cleanly at `4c4a4c3f4`.
- Wrote and ran (not committed) a 50-goroutine, `-race`, 5x-repeated concurrent `SelfDestruct` probe against a single monster; result: exactly one `Killed==true` every run. Probe file removed after use; `git status` confirmed no residue.

## Not evaluable

None — the unit's full review surface (the two changed files, plus the `TenantRegistry.Update` contract the correctness genuinely depends on) was reviewed and independently verified.

## Disposition

APPROVED_WITH_FINDINGS. The implementation is correct and matches the brief; the one finding (non-blocking) is that the committed test suite does not itself contain a concurrent/goroutine test proving the exactly-once guarantee, even though the guarantee holds (verified independently). Given tasks 7-10 explicitly depend on this guarantee, recommend closing the gap with a `-race` goroutine test before/alongside those tasks land, rather than leaving the concurrency proof living only in this review's ad-hoc probe.

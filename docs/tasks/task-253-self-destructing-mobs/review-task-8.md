# Review — Task 8: DoT ticks trip the self-destruct threshold

Commit range: `c5a1da3d0..64047571a` (single commit `64047571a`)
Brief: `.superpowers/sdd/plan/task-8-brief.md`
Report: `.superpowers/sdd/plan/task-8-report.md`

## Scope

`git diff --stat c5a1da3d0..64047571a`:

```
services/atlas-monsters/atlas.com/monsters/monster/processor.go        |  9 ++++++++-
services/atlas-monsters/atlas.com/monsters/monster/self_destruct_dot_test.go | 148 +++++++++++++++++++++
services/atlas-monsters/atlas.com/monsters/monster/status_task.go      | 12 ++++++++++++
3 files changed, 168 insertions(+), 1 deletion(-)
```

Matches the brief's declared file set (`status_task.go`, new test file) plus one
undeclared-but-disclosed file, `processor.go`, which the report explains and
which this review adjudicates below. No scope drift beyond that.

## Requirement-by-requirement

### Step 3 threshold check (`status_task.go:101-116`)

```go
if ma, ierr := resolveMonsterInformation(t.l, ctx, m.MonsterId()); ierr == nil {
    sd := ma.SelfDestruction()
    if sd.OnHpThreshold() && int64(ds.Monster.Hp()) <= int64(sd.Hp()) {
        NewProcessor(t.l, ctx).SelfDestruct(m.UniqueId(), se.SourceCharacterId(), TriggerThreshold)
    }
}
```

Placed after the "Emit damaged event" line, exactly where the brief specifies.
`SourceCharacterId()`/`TriggerThreshold` match the brief. `int64(...)  <=
int64(...)` mirrors the pre-existing pattern at `processor.go:680`
(damageCore's own threshold check) — not a new raw-comparison site.

**PASS.**

### Deviation: `resolveMonsterInformation` seam instead of the brief's literal `information.NewProcessor(...).GetById` call

This is the item the task explicitly calls out for adjudication. Verified on
the merits, not on the implementer's say-so:

- `processor.go:98-112` (diff): `ProcessorImpl.monsterInformation` is
  refactored from an inline body into a one-line delegate to the new free
  function `resolveMonsterInformation(l, ctx, monsterId)`. The extracted body
  is verbatim — `if testInformationLookup != nil { return
  testInformationLookup(monsterId) }; return information.NewProcessor(l,
  ctx).GetById(monsterId)` — with `p.l`/`p.ctx` replaced by parameters `l`/
  `ctx` passed in by the caller. For `ProcessorImpl.monsterInformation` those
  parameters are literally `p.l`/`p.ctx`, so every existing caller
  (`damageCore`, `Kill`, `SelfDestruct`, `selfDestructFrom`) is unaffected —
  confirmed by reading the diff hunk directly, not inferring from the report.
- **Production code path** (`testInformationLookup == nil`, i.e. always
  outside a unit test): `resolveMonsterInformation` calls
  `information.NewProcessor(l, ctx).GetById(monsterId)` — the exact call the
  brief's snippet specifies, with the exact same `ctx`. No caching behaviour
  is bypassed: `information.Processor.GetById` (`monster/information/processor.go:34-40`)
  gates on its own package-level `dataCachePtr`, which this refactor does not
  touch or route around.
- **Tenant propagation**: `status_task.go:114` passes the same `ctx` used
  everywhere else in `processDoTTick` (the parameter, which traces back to
  `tctx := tenant.WithContext(t.ctx, ten)` built in `processMonsterEffects`
  at `status_task.go:37` and threaded through `processDoTTick`'s `ctx`
  parameter at `status_task.go:65`). `resolveMonsterInformation` does not
  drop or replace this context; it is passed straight through to
  `information.NewProcessor(ctx, ...)`. Tenant scoping is intact.
- **Test-seam concern is real**: `information.NewProcessor(t.l, ctx).GetById`
  called directly, as the brief's literal snippet shows, has no test-only
  override — with `dataCachePtr == nil` (the test default) it falls through
  to `upstreamAndExtract`, a real HTTP call. The brief's own Step-1 test
  plan requires no live REST dependency (uses `testInformationLookup` per
  `processor_test.go:1594-1600`, cited by the brief itself as the pattern to
  copy). The brief's literal Step-3 snippet is therefore inconsistent with
  its own Step-1 test plan; the implementer's rationale for deviating is
  correct, not merely plausible.
- Net effect: identical production behaviour to the brief's literal call,
  with an added test seam that the brief's own test plan required anyway.
  Not a regression on caching or multi-tenancy.

**Verdict on the deviation: justified, not blocking.**

### Emitter (SelfDestruct → producer) — no seam, and correctly left alone

The report notes `NewProcessor(t.l, ctx).SelfDestruct(...)` always emits via
the real `producer.ProviderImpl`, consistent with every other package-level
`NewProcessor` call in this file (e.g. `RepickAndEmit` three lines above at
`status_task.go:47`). Verified: `status_task.go:47` uses the same
unseamed pattern for an existing, previously-reviewed call. Tests instead
swap the producer manager (`producertest.InstallCapturing`/`InstallNoop`),
which is the standing convention (`monster/registry_test.go`'s `TestMain`
already installs the no-op floor). Consistent, not a new pattern invented for
this task.

### Double-fire / exactly-once on the DoT path

Traced into Task 7's machinery, not reimplemented here:

- `ProcessorImpl.SelfDestruct` (`processor.go:1832`) re-checks `m.Alive()`
  and re-derives `sd.Present()` before calling `selfDestructFrom` — it does
  not trust the caller.
- `selfDestructFrom` → `GetMonsterRegistry().SelfDestruct` (`registry.go:503`)
  does an atomic CAS via `r.reg.Update`: `transitioned = cur.Hp > 0; cur.Hp =
  0`. `DamageSummary.Killed` is only `true` on the transitioning call. A mob
  with both poison and venom active could call `processDoTTick` twice in one
  `Run()` cycle, each independently evaluating the threshold and calling
  `SelfDestruct` — but the second call's CAS finds `Hp == 0` already,
  `transitioned == false`, `s.Killed == false`, and `selfDestructFrom`
  returns before `finalizeKill` — so at most one `EventMonsterStatusKilled`
  is emitted regardless of how many ticks trip the threshold in the same
  cycle. **Task 8 does not need to (and does not) reimplement this — it only
  needs to call the Task 7 entry point, which it does.**

### Ordinary-kill-wins ordering

Same CAS in `registry.go:503`: if an ordinary kill (attack, Mortal Blow)
already dropped the mob's HP to 0 before the DoT-triggered `SelfDestruct`
call lands, `transitioned == false` and the self-destruct is silently
dropped — the ordinary kill's own `finalizeKill` call already ran. This is
unmodified Task 7 code; Task 8 correctly relies on it rather than adding its
own ordering logic.

### Kill-prevention cap untouched

`status_task.go:82-90` (the `totalDamage >= current.Hp() { totalDamage =
current.Hp() - 1 }` cap) is above the new threshold-check block and
unmodified by this diff. `TestDoTTickCannotReachZeroHp` and
`TestDoTTickThresholdMobStillCannotBeReducedToZero` both exercise it
directly and pass, including the case where the cap clamps to HP 1 which
still crosses a nonzero threshold (correct per FR-2.4: the cap prevents 0 HP,
not detonation).

### Test honesty

Ran the four new tests directly in this review (not trusting the report's
transcript alone):

```
go test ./monster/ -run 'TestDoTTick' -v
--- PASS: TestDoTTickCrossingThresholdDetonates (0.03s)
--- PASS: TestDoTTickNotCrossingThresholdDoesNotDetonate (0.00s)
--- PASS: TestDoTTickCannotReachZeroHp (0.00s)
--- PASS: TestDoTTickThresholdMobStillCannotBeReducedToZero (0.00s)
PASS
```

`go build ./...` is clean.

The report's RED evidence (`git stash push -- status_task.go processor.go`,
rerun, two of four tests fail with "monster must be absent from the registry
after detonation") is consistent with the diff: without the threshold-check
block, no code path removes the monster from the registry or emits KILLED
for a threshold crossing, so `TestDoTTickCrossingThresholdDetonates` and
`TestDoTTickThresholdMobStillCannotBeReducedToZero` would fail exactly as
described, while the two non-crossing/no-block tests would pass unchanged.
Assertions are against registry state (`GetMonster` presence/HP) and decoded
kafka event bodies via a capturing producer (`killedEvents` decodes
`EnvEventTopicMonsterStatus` messages and filters on `EventMonsterStatusKilled`)
— not against a mock's call count. This is a genuine threshold-crossing test,
not a tautology.

### Test helper reuse

`newTestTenant`, `testField`, `boomerMonsterId` are pre-existing helpers from
`self_destruct_test.go` (Task 7's test file), not newly invented for this
task — no new `*_testhelpers.go` file was created, consistent with repo
convention.

## Findings

None blocking. No non-blocking notes beyond what's already covered above.

## Not evaluable

None — the full review surface (the diff plus the Task 7 `SelfDestruct`/
`selfDestructFrom`/registry CAS contract it depends on) was read and traced
by hand.

## Disposition

APPROVED. The deviation from the brief's literal Step-3 snippet is justified
on its merits: identical production call, identical caching gate, identical
tenant-scoped context, with the test seam the brief's own Step-1 plan
required. Double-fire and kill-ordering are correctly delegated to Task 7's
already-reviewed CAS/epilogue rather than reimplemented. Tests assert
observable state and event bodies, and RED/GREEN evidence is consistent with
the diff.

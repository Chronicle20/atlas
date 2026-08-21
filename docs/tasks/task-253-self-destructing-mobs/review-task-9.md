# Review — Task 9: timer-driven self-destruction (atlas-monsters)

Commit range: `64047571a..9babf80fe` (single commit `9babf80fe`)
Brief: `.superpowers/sdd/plan/task-9-brief.md`
Report: `.superpowers/sdd/plan/task-9-report.md`

## Scope

`git diff --stat 64047571a..9babf80fe`:

```
services/atlas-monsters/atlas.com/monsters/main.go                        |   2 +
services/atlas-monsters/atlas.com/monsters/monster/processor.go           |  17 +-
services/atlas-monsters/atlas.com/monsters/monster/registry_test.go       |   1 +
.../monster/self_destruct_timer_registry.go                               | 113 +++
.../monsters/monster/self_destruct_timer_task.go                          |  51 ++
.../monsters/monster/self_destruct_timer_test.go                          | 326 +++
6 files changed, 509 insertions(+), 1 deletion(-)
```

Matches the brief's file list exactly (registry, task, tests, `processor.go` arm/unregister x2, `registry_test.go` TestMain, `main.go` init + sweep registration). No files outside the stated surface were touched.

## Checklist findings

### 1. Detonation routes through Task 7's `Processor.SelfDestruct`/`TriggerTimer` — PASS

`self_destruct_timer_task.go:34-47` — `processEntry` calls `NewProcessor(t.l, tctx).SelfDestruct(uniqueId, 0, TriggerTimer)` with no reimplemented kill logic. `SelfDestruct` (`processor.go:1847-1868`) does the alive check, information/`SelfDestruction` re-derivation, and delegates to `selfDestructFrom` → `GetMonsterRegistry().SelfDestruct` (the exactly-once transition guard: `!s.Killed` drops a lost race) → `finalizeKill`. No second detonation path was added.

### 2. Unregister happens on both ordinary death and `Destroy` — PASS

- `finalizeKill` (`processor.go:602`, immediately after `GetDropTimerRegistry().Unregister(...)`): `GetSelfDestructTimerRegistry().Unregister(p.ctx, p.t, m.UniqueId())`. `finalizeKill` is documented as "the single kill epilogue... every death... runs exactly this sequence" (processor.go:596-600), so ordinary damage kills, Mortal Blow, and all self-destruct triggers all pass through it.
- `Destroy` (`processor.go:1404-1406`): unregister is the first statement, before `GetDropTimerRegistry().Unregister`, `GetAttackCooldownRegistry().ClearCooldowns`, and `GetMonsterRegistry().RemoveMonster`.
- `DestroyAll` (`processor.go:1902-1903`) → `destroyInTenant` (`processor.go:1703-1714`) iterates every mob and calls `p.Destroy` per unique id, so bulk teardown (`Teardown`, used at service shutdown) also unregisters every armed timer — satisfies FR-3.6, and `TestDestroyAllLeavesNoTimers` exercises this exact path (two tenants, `DestroyAll`, assert both tenants' filtered `entriesForTenant` empty).

Traced both call sites; no leaked-entry path found.

### 3. Only `selfDestruction` mobs with no HP threshold arm the timer — PASS

`information/model.go:44,48`:
```go
func (s SelfDestruction) OnHpThreshold() bool { return s.present && s.hp > -1 }
func (s SelfDestruction) OnTimer() bool { return s.present && s.hp <= -1 }
```
These are mutually exclusive on `hp`. `Create`'s arm block (`processor.go:311-319`) gates on `sd.OnTimer()`, so a threshold mob (`hp > -1`, e.g. Boomer `{1,-1,1800}`) never arms a timer, and a mob with both a threshold and a `removeAfter` (`{3,5,1}`, `hp==1`) still routes to `OnHpThreshold()`, not `OnTimer()`. `TestCreateDoesNotArmTimerForOtherMobs` table-drives exactly these cases (`self_destruct_timer_test.go:67-128`) and passed under a real run (`go test ./monster/... -run Timer -v`, confirmed locally).

### 4. Registry pattern fidelity — PASS

`self_destruct_timer_registry.go` mirrors the brief's literal code block (which itself mirrors `drop_timer_registry.go`): tenant-scoped `atlasredis.TenantRegistry[uint32, storedSelfDestructTimer]`, `sync.Once`-gated `Init`, `Register`/`Unregister`/`GetAll` with the `fromStoredSelfDestructTimer` round-trip (tenant reconstruction via `uuid.Parse` + `tenant.Create`, `fireAt` round-tripped through `UnixMilli`/`UnixMilli`). `GetAll` correctly uses the cross-tenant sibling (`GetAllAcrossTenants`) since the sweep task has no single tenant to scope to — same shape as `DropTimerRegistry`.

Registered in:
- `registry_test.go:38` (`TestMain`), immediately after `InitDropTimerRegistry(rc)`, before `InitPuppetRegistry(rc)` — matches the brief.
- `main.go` — `InitSelfDestructTimerRegistry(rc)` added after `InitDropTimerRegistry(rc)` (line ~65), and `tasks.Register(l, ctx)(monster.NewSelfDestructTimerTask(l, ctx, time.Second))` added in `registerSweepTasks` immediately before the `NewMonsterAggroDecayTask` line — both confirmed via diff.

### 5. Tests assert real registry/event state — PASS

Read the full `self_destruct_timer_test.go`. Every assertion touches real state, not mock call counts:
- `TestCreateArmsTimerForTimerMob` / `TestCreateDoesNotArmTimerForOtherMobs` read `GetSelfDestructTimerRegistry().GetAll(ctx)` (filtered per-tenant via `entriesForTenant`) and assert on `Action()`/`MonsterId()`/`FireAt()`.
- `TestSelfDestructTimerTaskFiresOnElapsedEntry` uses `producertest.InstallCapturing()` and asserts on a real captured `EventMonsterStatusKilled` body (`DeathType`, `ActorId`), plus `GetMonster` absence and empty `GetAll` — a genuine end-to-end assertion, not a call-count.
- `TestKillUnregistersTimer` drives a real kill via `p.damageCore(m, 55, ...)` and asserts the registry is empty afterward (FR-3.3).
- `TestDestroyUnregistersTimer`/`TestDestroyAllLeavesNoTimers` call the real `Destroy`/`DestroyAll` and assert on registry state.

Ran the new tests directly to confirm (not just trusting the report):
```
cd services/atlas-monsters/atlas.com/monsters && go build ./... && go vet ./...   # clean
go test ./monster/... -run 'Timer' -v   # all 9 new + 2 pre-existing DropTimer* PASS
```
Output matches the report's pasted evidence (including the pre-existing, unrelated `unsupported protocol scheme` warnings from `Create`'s controller-candidate lookup — not introduced by this change, and `Create` still returns successfully).

RED evidence in the report (compile failure `undefined: GetSelfDestructTimerRegistry` before the three implementation files existed) is credible test-honesty evidence for a new-file addition; nothing in this diff can be evaluated for "does this specific assertion fail without the change" more strongly than that without reverting the diff, which is outside a read-only review's tooling. No sign of a tautological or mock-count-only test.

## Disclosed deviation: `Create`'s information lookup moved to `p.monsterInformation`

`processor.go:249` — `ma, err := p.monsterInformation(input.MonsterId)`, replacing a direct `information.NewProcessor(p.l, p.ctx).GetById(input.MonsterId)` call.

`p.monsterInformation` (`processor.go:103-105`) delegates to `resolveMonsterInformation(p.l, p.ctx, monsterId)` (`processor.go:107-115`):
```go
func resolveMonsterInformation(l logrus.FieldLogger, ctx context.Context, monsterId uint32) (information.Model, error) {
	if testInformationLookup != nil {
		return testInformationLookup(monsterId)
	}
	return information.NewProcessor(l, ctx).GetById(monsterId)
}
```

Applying the same test as Task 8's accepted deviation:
- Same constructor in production: when `testInformationLookup` is nil, the fallback is `information.NewProcessor(l, ctx).GetById(monsterId)` — identical to the code path being replaced, with the identical `p.l`/`p.ctx` arguments (`resolveMonsterInformation` is called with exactly `p.l, p.ctx`).
- Same tenant-scoped ctx: `p.ctx` is threaded unchanged; no context substitution.
- No `dataCachePtr` bypass: the seam does not touch `information.Processor`'s internal cache at all — it either forwards to the real `information.Processor.GetById` (which owns its own caching internally) or, in tests, calls the injected hook. Neither path reaches around `information.Processor`'s cache gate from outside.

The reasoning holds unchanged from Task 8: production behavior is byte-identical, and the deviation is disclosed both in the diff (comment-free but self-evident from the one-line change) and prominently in the report. It is also independently necessary — without it, `Create`'s new arm-block (which depends on `ma.SelfDestruction()`) is untestable per the brief's own Step 1 spec, which was verified by re-running the tests above.

## Other observations (non-blocking)

- The report's self-review says `TestDestroyAllLeavesNoTimers` uses `r.Clear(ctx)` "which clears across all tenants" to avoid cross-test pollution — the actual test body only calls `GetMonsterRegistry().Clear(context.Background())` (monster registry, not the self-destruct timer registry) and relies on `entriesForTenant`'s per-tenant filter plus fresh per-test tenant UUIDs (`newTestTenant`) for isolation. The report's prose is slightly imprecise but the test itself is correct and isolated; this is a documentation nit, not a code defect.
- `processEntry`'s belt-and-suspenders `!m.Alive()` check (`self_destruct_timer_task.go:40`) before calling `SelfDestruct` is redundant with `SelfDestruct`'s own alive check (`processor.go:1853`), but this exactly matches the brief's literal code block and is not a correctness issue — it lets the sweep avoid constructing a `Processor` for an already-dead/missing mob and unregister eagerly.

## Not evaluable

None. The full review surface (registry, task, processor wiring, main.go, TestMain, and the new test file) was read and exercised; no file the diff depends on for correctness was left unchecked.

## Verdict rationale

All five checklist items PASS with cited evidence; the disclosed deviation is sound under the same reasoning already accepted for Task 8. Build, vet, and the new test suite were independently re-run and match the report. No blocking findings.

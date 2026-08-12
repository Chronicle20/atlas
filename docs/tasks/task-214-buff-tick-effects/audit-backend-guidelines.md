# Backend Audit — atlas-buffs (task-214-buff-tick-effects)

- **Service Path:** services/atlas-buffs/atlas.com/buffs
- **Scope:** Diff ef4855e32..1e57f3436, confined to services/atlas-buffs/atlas.com/buffs/**
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-08-12
- **Build:** PASS
- **Tests:** all packages pass (`go test ./... -count=1` — periodic 0.004s, character 0.493s, tasks 0.040s, berserk 0.366s, all others green)
- **Overall:** NEEDS-WORK

## Build & Test Results

```
$ go build ./...
(clean, no output)

$ go test ./... -count=1
ok  atlas-buffs/berserk                    0.366s
ok  atlas-buffs/buff                        0.006s
ok  atlas-buffs/buff/stat                   0.003s
ok  atlas-buffs/character                   0.493s
ok  atlas-buffs/external/character           0.031s
ok  atlas-buffs/external/dataskill           0.031s
ok  atlas-buffs/external/effectivestats      0.030s
ok  atlas-buffs/external/skills              0.030s
ok  atlas-buffs/kafka/consumer/characterstatus 0.061s
ok  atlas-buffs/kafka/consumer/skillstatus     0.058s
ok  atlas-buffs/kafka/message/character        0.008s
ok  atlas-buffs/periodic                       0.004s
ok  atlas-buffs/tasks                          0.040s
```

`tools/goroutine-guard.sh` from repo root: exit 0 (no bare `go` statements in changed packages; DOM-26 clean).

## Structural note (pre-existing, not scored)

atlas-buffs has **no `entity.go`, `builder.go`, `administrator.go`, or `provider.go` anywhere in the service** (`find services/atlas-buffs -name entity.go -o -name builder.go -o -name administrator.go -o -name provider.go` → no matches). The service is entirely Redis-`TenantRegistry`-backed, not GORM. DOM-01/02/03/16/11 (builder/ToEntity/Make/administrator/provider-laziness), which assume the GORM layer described in `file-responsibilities.md`, are **structurally N/A** for this service — this predates task-214 and none of the changed files introduce a GORM entity. Not re-litigated below.

## Domain / Package Checklist Results

### `periodic` (new support package — data table, no model.go per DOM's DB sense, but does hold an immutable domain value type)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| Immutable-model convention | `Effect` has unexported fields + accessor methods | PASS | `periodic/model.go:37-60` — `statType`, `interval`, `resource`, `direction`, `floor` are all unexported; `StatType()`, `Interval()`, `Resource()`, `Direction()`, `Floor()` are the only means of reading them. No setter exists. |
| DOM-21 | Types/constants sourced from `libs/atlas-constants` | PASS | `periodic/table.go:26-48` keys the `effects` map by `character.TemporaryStatType` (imported from `github.com/Chronicle20/atlas/libs/atlas-constants/character`, `periodic/model.go:12`). Verified the three constants are real, not invented: `libs/atlas-constants/character/temporary_stat.go:24` (`TemporaryStatTypePoison`), `:29` (`TemporaryStatTypeDragonBlood`), `:40` (`TemporaryStatTypeRecovery`). `Lookup(statType string)` (`periodic/table.go:54-57`) is the single conversion point from the pre-existing `stat.Model.Type() string` (`buff/stat/model.go:33`, unchanged) to the typed constant — grep across the changed non-test files (`character/*.go`, `tasks/*.go`, `periodic/*.go`) for the raw literals `"POISON"`, `"DRAGON_BLOOD"`, `"RECOVERY"` found only one hit, in the pre-existing, explicitly out-of-scope `character/immunity.go:8`. |
| FILE-01/02/06 (support-package file placement) | No package-named catch-all; each file single-purpose | PASS | `periodic/model.go` holds only the `Effect`/`Resource`/`Direction` types and accessors; `periodic/table.go` holds only the static `effects` table and `Lookup`. Neither file is named `periodic.go` and neither bundles ≥2 unrelated responsibilities (Processor/RestModel/requests) — this package has none of those since it does no I/O. |
| Test quality | Table-driven, no `*_testhelpers.go` | PASS | `periodic/table_test.go:16-39` (`TestLookupRows`) and `:43-45` (`TestLookupRowCount`, which fails the suite if a row is added without a matching case) are table-driven; no helper file was added. |

### `character` (existing domain package — has `model.go`; changed: `processor.go`, `registry.go`, new `periodic.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06 | Processor constructor takes `logrus.FieldLogger` | PASS | `character/processor.go:42` — `func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor`. |
| FILE-01 | New `Processor` method (`ProcessPeriodicTicks`) lives in `processor.go` | PASS | `character/processor.go:298-362` (`func (p *ProcessorImpl) ProcessPeriodicTicks() error`) and its private helper `hpFor` (`:368-380`) are both in `processor.go`, not a topic file. |
| Int16 arithmetic — sign/overflow | `amount := int16(eff.Direction()) * int16(magnitude)` | PASS (verified, not just eyeballed) | `magnitude` is clamped to `maxTickMagnitude = int32(32767)` at `character/processor.go:284,321-323` before the earlier non-positive check (`:318-320`) — so `magnitude` is always in `[1, 32767]` when this line runs. `int16(magnitude)` is therefore always a safe positive int16 conversion; `eff.Direction()` is `int8(±1)`; the product range is `[-32767, 32767]`, inside int16's `[-32768, 32767]`. No overflow. |
| Int16 arithmetic — floor clamp | `amount = -int16(hp - 1)` at `character/processor.go:348` | PASS (verified — this is the item the task flagged as needing careful sign/overflow proof) | `hp` is `uint16` (`external/character/rest.go:10`, `Hp uint16`). The floor branch (`:338-350`) only executes when `int32(hp)+int32(amount) < 1` (`:347`), and `amount` is bounded to `amount >= -32767` (proven above, since only the `Drain` direction reaches this branch and `eff.Floor()==true` implies a negative `amount`). Substituting the tightest bound: the branch can only be reached when `hp < 1 - amount <= 1 + 32767 = 32768`, i.e. `hp <= 32767`. Combined with the preceding `hp <= 1 { continue }` guard (`:343-346`), `hp` is proven to be in `[2, 32767]` whenever `hp - 1` is computed, so `hp - 1 ∈ [1, 32766]` — always representable as a positive `int16` with no truncation/sign-flip. **This line is safe as written**; the safety is *not* self-evident from the line alone (a naive read suggests `hp` up to 65535 could overflow), so it is recorded here as a proven pass, not an assumption. |
| DOM-25 (no client wire-byte literals) | N/A | N/A | This tick path emits internal command bodies (`CHANGE_HP`), not client-facing dispatcher codes. |
| DOM-26 | Goroutines via `routine.Go` | PASS | `character/processor.go:262,392` — both `ExpireBuffs` and `ProcessPeriodicTicks` package-level fan-out functions use `routine.Go(l, ctx, func(_ context.Context) {...})`, no bare `go`. |
| DOM-28 (no silent degradation) | **FAIL** | `character/processor.go:368-380` (`hpFor`). This is a fallible enrichment: it fetches remote character HP (`p.getCharacterHp`, itself backed by `extchar.RequestById` — a cross-service REST call, `character/processor.go:48-54`) and, on failure, degrades by skipping the floor-sensitive tick rather than propagating the error (`ProcessPeriodicTicks` continues the loop at `:341-342` when `hpFor` returns `ok=false`). Per `patterns-resilience.md`'s "No silent degradation" policy (mandatory for new code, enforced by DOM-28), a degrading fallback must both **log Warn** and **increment `atlas_enrichment_degraded_total`** via `degrade.Observe`. `hpFor` does log Warn (`:374`, `p.l.WithError(err).Warnf(...)`) but never calls `degrade.Observe` and never increments any metric — the degradation is invisible to the `atlas_enrichment_degraded_total{component}` dashboard the guideline exists to populate. This is new code introduced by this branch (not a pre-existing file this branch left untouched), so it is graded against the guideline directly, not exempted by berserk's `getMaxHp`/`getCharacter` doing the same uninstrumented thing (`berserk/processor.go:208-210`, pre-existing and out of this diff's scope — noted, not double-counted). |
| Error handling — `GetPeriodicEntries` | Redis read error → silent `nil` | **WARN (non-blocking)** | `character/periodic.go:55-60`: `vals, err := r.characters.GetAllValues(ctx, t); if err != nil { return nil }` — on a Redis read failure the entire tenant's periodic-tick pass silently no-ops for that 1s cycle, with **zero log line**, not even a Warn. This is weaker than the pre-existing poison-path precedent it mirrors in *shape*, but the pre-existing precedent's history is not itself a pass — see the "prevalence is not compliance" rule. Judged non-blocking here because: (a) the ticker self-heals on the next pass (no data is dropped, only delayed by ≤1 driving interval), (b) the design doc (§3.5, accepted out-of-scope) already documents this class of throttle-window imprecision as accepted for task-214. Recommend adding a `p.l` Warn log at minimum in a follow-up so a sustained Redis outage is observable, but not gating this PR on it. |
| Error handling — `UpdatePeriodicTick`/`ClearPeriodicTick` | Discard write errors with `_ =` | **PASS (defensible, not a finding)** | `character/periodic.go:112-121`. This is a self-healing throttle store: a failed `PutWithTTL` means the next pass re-reads no throttle entry and ticks again (one extra tick, bounded by the row's own interval); a failed `Remove` on cancel leaves a stale throttle entry that the documented `PeriodicTickTTL = 5 * time.Minute` (`:21`) bounds to "stale for ≤5 min" per the file's own doc comment (`:15-20`). Both failure modes are explicitly reasoned about in the source comments, not silently assumed benign, and match the task's stated framing of "defensible for a ticker path." |
| Concurrency — fan-out shared state | Per-tenant `routine.Go` fan-out | PASS | `character/processor.go:385-400` (`ProcessPeriodicTicks` package func). Each goroutine gets its own `NewProcessor(l, tctx)` with `tctx := tenant.WithContext(ctx, t)` (`:393`) — no processor state is shared across tenants; the only cross-goroutine shared object is `GetRegistry()`, which is a package singleton already required to be tenant-safe by its Redis key scheme (`TenantRegistry.entityKey` namespaces every key by tenant, `libs/atlas-redis/tenant_registry.go:40-42`). `hpCache` (`character/processor.go:301`) is a `map[uint32]hpLookup` local to one `ProcessPeriodicTicks()` call/goroutine — not shared across the fan-out. No new data race introduced. |
| Redis key construction / TTL | `periodicTicks` `TenantRegistry[TickKey, time.Time]` | PASS | `character/registry.go:36-38`: `keyFn: func(k TickKey) string { return strconv.FormatUint(uint64(k.CharacterId), 10) + ":" + k.StatType }` — mirrors the pre-existing `statKey` composite-key convention (`character/registry.go:56-58`) already used for `sourceId:statType`. `UpdatePeriodicTick` uses `PutWithTTL(ctx, t, key, at, PeriodicTickTTL)` (`character/periodic.go:114`) with the documented 5-minute TTL bound (`:21`); `library`'s `PutWithTTL` correctly threads the ttl into `client.Set(..., ttl)` (`libs/atlas-redis/tenant_registry.go:117-124`). No unbounded key growth risk: every throttle key is either explicitly cleared (`ClearPeriodicTick`/`ClearPeriodicTicksFor`) or expires within 5 minutes of its last refresh. |

### `tasks` (support package — `tasks/periodic.go`, replaces `tasks/poison.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-06 | No package-named catch-all | PASS | `tasks/periodic.go` holds only `PeriodicTick`/`NewPeriodicTick`/`Run`/`SleepTime` — a single-purpose ticker-task type, same shape as the deleted `tasks/poison.go` and the untouched `tasks/expiration.go`/`tasks/berserk.go` siblings. |
| Dead-code check | Old poison path fully removed | PASS | `grep -rn "PoisonTick\|poisonTicks\|ProcessPoisonTicks"` across the whole service after the diff returns no matches; `tasks/poison.go` and `tasks/poison_test.go` are deleted, `main.go:74` now wires `tasks.NewPeriodicTick(l, 1000)`, and the `berserk/processor.go:265` doc comment was correctly updated (`ProcessPoisonTicks` → `ProcessPeriodicTicks`) rather than left stale. |
| DOM-24 | Kafka producer stubbed in tests | PASS | `tasks/periodic_test.go:29-36` (`TestPeriodicTick_Run_DoesNotPanicWithNoTenants`) calls `pt.Run()` → `character.ProcessPeriodicTicks`, but registers zero tenants via a fresh `miniredis` instance, so `GetTenants` returns an empty slice and the per-tenant `routine.Go` fan-out never fires — no emit path is exercised, so no producer stub is required for this specific test. |

## Test-infrastructure checklist (`character/testmain_test.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-24 | Shared `producertest.Capture`, no per-test `t.Cleanup(producer.ResetInstance)` | PASS | `character/testmain_test.go:18-21`: `TestMain` installs `emitted = producertest.InstallCapturing()` once, package-wide. Every test in `periodic_processor_test.go` that reads `emitted` calls `emitted.Reset()` first (e.g. `:78,103,125,142,182,198,215,233,249,270,287,335,372`) rather than reinstalling or `t.Cleanup`-reverting the singleton. Grepped the whole package for `t.Cleanup(producer.ResetInstance)` and `producertest.InstallNoop` — no matches; only the one `InstallCapturing()` call in `TestMain`. |
| Test parallelism / state-leak risk | No `t.Parallel()` in the changed test files | PASS | `grep -rn "t.Parallel" character/*_test.go tasks/*_test.go periodic/*_test.go` — no matches. The package-level shared `emitted` capture is safe under the current fully-serial test execution; flag if a future change adds `t.Parallel()` to this package, since `emitted.Reset()` would then race across subtests. |
| Test quality | Same-package struct literal setup, no `*_testhelpers.go` | PASS | `character/periodic_processor_test.go:24-36` (`tickProcessor`) builds `&ProcessorImpl{...}` directly with a frozen clock and stubbed HP closure — same-file, same-package, not a separate helper file. Table-driven: `TestPeriodicTickDragonBloodFloorsAtOne` (`:167-193`) and `TestPeriodicTickClearedOnRemoval` (`:306-366`) both use the `tests := []struct{...}{}` + `t.Run` shape (DOM-20). |

## Security Review

Not applicable — atlas-buffs is not an auth/token-handling service; SEC-01..04 skipped.

## Summary

### Blocking (must fix)
- **DOM-28** — `character/processor.go:368-380` (`hpFor`): the Dragon-Blood HP-floor fetch degrades on failure (logs Warn, skips the tick) but never calls `degrade.Observe` / increments `atlas_enrichment_degraded_total`, so a sustained upstream `CHARACTERS` outage that silently disables the HP floor for every Dragon-Blood tick is invisible on the `atlas_enrichment_degraded_total{component}` dashboard the guideline exists to populate. Fix: wrap the fetch with `model.ErrDecorator` + `degrade.Observe(p.l, "atlas-buffs.character.hp", characterId, err)` (or call `degrade.Observe` directly at the existing Warn call site if the decorator shape doesn't fit this non-`Model`-returning helper).

### Non-Blocking (should fix)
- `character/periodic.go:55-60` (`GetPeriodicEntries`): a Redis `GetAllValues` failure returns `nil` with zero logging — add at minimum a `p.l`-equivalent Warn log (the `Registry` has no logger today; consider threading one through, or logging from the `ProcessorImpl` caller when the entries slice comes back empty in a way that's indistinguishable from "no periodic buffs exist"). Self-healing on the next pass, so not gating.

### Passed / verified (worth recording, not just "looked fine")
- DOM-21: `periodic` table sourced from real `libs/atlas-constants/character` constants, single conversion point (`periodic.Lookup`), no literal duplication in changed files.
- Int16 sign/overflow arithmetic in the tick pass (`character/processor.go:284,321-323,336,347-348`): proved safe by chaining the `maxTickMagnitude` clamp through the floor-branch guard condition — `hp` is provably `<= 32767` whenever `hp - 1` is cast to `int16`.
- DOM-24 / DOM-26: shared `producertest.Capture` install and `routine.Go` usage both clean.
- Redis key/TTL handling in the new `periodicTicks` registry.

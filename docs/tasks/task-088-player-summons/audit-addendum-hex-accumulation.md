# Backend Audit — atlas-buffs + atlas-summons (commit b87046430)

- **Scope:** single commit `b87046430` vs parent `197abfd3` (task-088 addendum "Hex of the Beholder buff accumulation")
- **Modules:** `services/atlas-buffs/atlas.com/buffs`, `services/atlas-summons/atlas.com/summons`
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-06-14
- **Build:** PASS (both modules)
- **Vet:** PASS (both modules)
- **Tests:** PASS (both modules, including `-race`)
- **Overall:** PASS

## Build & Test Results

atlas-buffs (`services/atlas-buffs/atlas.com/buffs`):
- `go build ./...` clean
- `go vet ./...` clean
- `go test ./... -count=1` — all OK (`atlas-buffs/character` 0.333s, no Kafka-producer hang)
- `go test -race ./... -count=1` — clean (`atlas-buffs/tasks` 1.038s)

atlas-summons (`services/atlas-summons/atlas.com/summons`):
- `go build ./...` clean
- `go vet ./...` clean
- `go test ./... -count=1` — all OK (`atlas-summons/summon` 0.071s)
- `go test -race ./summon/... ./buff/...` — clean (`summon` 1.153s)

Objective gate PASSED — proceeded to mechanical checks.

## Focus-Area Findings

### 1. `map[int32]buff.Model` -> `map[string]buff.Model` key-type change — blast radius

PASS. Migration is complete; zero `map[int32]buff` references remain anywhere in
the module (grep returns nothing). Every site updated:
- `character/model.go:21` field, `:28-30` defensive-copy `Buffs()`, `:52`/`:66` Marshal/Unmarshal aux struct.
- `character/registry.go` all five map allocations migrated (`:77`, `:171`/`not`, `:197`/`not`, `:218`, `:263`/`keep`).
- Production value-iterators are key-agnostic and unaffected:
  - `character/resource.go:39` `for _, bs := range cm.Buffs()` — iterates values, ignores key.
  - `character/immunity.go:23` `for _, b := range m.buffs` — iterates values, ignores key.
- No other production caller of `.Buffs()` exists; remaining `.Buffs()[...]` callers are all in `*_test.go` and were updated to `srcKey(...)`/`statKey(...)`.

Behavioral note (NOT a defect): in accumulate mode all per-stat entries keep the
same `SourceId()`, so source-scoped ops (`Cancel`, `CancelByStatTypes`,
`GetExpired`) iterate by value and still operate correctly across the composite
keys — `registry.go:163`, `:190`, `:244`. Verified by `TestRegistry_Apply_Accumulate_DistinctStatsCoexist` (source-wide Cancel removes both per-stat entries, `registry_test.go:447-450`).

### 2. Cross-service contract mirror (atlas-buffs vs atlas-summons `ApplyCommandBody`)

PASS — byte-identical field set, names, types, json tags.
- atlas-buffs: `kafka/message/character/kafka.go:30-43`, new field `Accumulate bool \`json:"accumulate,omitempty"\`` at `:42`.
- atlas-summons: `buff/producer.go:46-58`, new field `Accumulate bool \`json:"accumulate,omitempty"\`` at `:57`.
- `StatChange` identical on both sides (`type string`, `amount int32`).
- `omitempty` on both: a producer that never sets it omits the key; the consumer's `c.Body.Accumulate` (plain bool) then defaults to `false`, preserving exact replace-by-sourceId semantics for every existing producer (`kafka/consumer/character/consumer.go:56`).
- Contract is exercised end-to-end in test: `beholder_task_test.go:142-166` json-unmarshals the actually-emitted payload into `buffmsg.Command[buffmsg.ApplyCommandBody]` and asserts `Accumulate=true`, exactly one stat, correct `SourceId`.

### 3. Per-stat APPLIED emission in processor.go

PASS. `character/processor.go:54-63`: `Apply` now loops over the slice returned by
`Registry.Apply` and emits one `appliedStatusEventProvider` per stored buff.
- Default mode: `Registry.Apply` returns exactly one whole-source buff (`registry.go:95-101`), so exactly one APPLIED — unchanged from prior behavior.
- Accumulate mode: returns one single-stat buff per change (`registry.go:86-94`); the loop emits one APPLIED per stat, each carrying that stat's own `SourceId()/Level()/Duration()/Changes()/CreatedAt()/ExpiresAt()` so the channel sets and later expires each icon independently.
- Loop short-circuits on first `buf.Put` error (`processor.go:60-62`).
- Disease-immunity early return preserved ahead of emit (`processor.go:44-47`).

### 4. Multi-tenancy / context handling

PASS.
- `Registry.Apply` resolves tenant via `tenant.MustFromContext(ctx)` (`registry.go:69`) and propagates `ctx` to `r.characters.Get/Put` and `r.tenants.Add`.
- Consumer passes the per-tenant `ctx` straight through to `NewProcessor(l, ctx).Apply(...)` (`consumer.go:46,56`); processor stores `ctx` and forwards to the registry (`processor.go`).
- atlas-summons producer side builds a per-tenant context (`beholder_task.go:69` `tenant.WithContext(t.ctx, ten)`) before `sweepBuff`; the documented tenant-context requirement at `beholder_task.go:26-30` is honored.

### 5. Immutable-model / builder conventions

PASS (no regression). `Model` keeps private fields + getter + defensive copy in
`Buffs()` (`model.go:27-33`). The in-place `m.buffs[...] = b` mutation followed by
`r.characters.Put` is the pre-existing registry persistence pattern; this commit
only changed key type and added the accumulate branch, not the mutation strategy.
`buff.Model` continues to be constructed via `buff.NewBuff(...)` (`registry.go:88`, `:96`); error from `NewBuff` is checked in both branches.

### 6. DOM-21 — shared atlas-constants types

PASS / N/A. The commit introduces no new domain type, alias, enum, or numeric
constant. The two new helpers `srcKey`/`statKey` (`registry.go:44-55`) build
plain string map keys (not a domain type). Stat keys remain the pre-existing
`stat.Model.Type() string` representation, unchanged by this commit; no
atlas-constants equivalent is being shadowed by new code. (`git diff` of non-test
files shows only local `var` declarations, no `type`/`const`/`iota`.)

## Domain / Sub-Domain Checklist (applicable subset)

These are existing Kafka/registry packages (no `model.go` domain scaffolding with
ToEntity/REST resource for the changed surface beyond the unchanged `resource.go`
GET handler), so most DOM-* REST checks are N/A to the diff. Applicable items:

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06 | Processor accepts FieldLogger | PASS | `character/processor.go` `ProcessorImpl` holds `logrus.FieldLogger` (constructor unchanged); consumer passes `l logrus.FieldLogger` (`consumer.go:46`) |
| DOM-09 | Emission errors handled | PASS | `processor.go:60-62` checks `buf.Put` error; `registry.go:88-90,96-98,104-106` check all errors |
| DOM-20 | Table-/case-driven, substantive tests | PASS | `registry_test.go:424-512` accumulate coexist/refresh/per-stat-expiry + default-replace regression; `beholder_task_test.go` multi-pulse pool coverage |
| DOM-21 | No duplication of atlas-constants types | PASS | No new types/consts introduced (see Focus 6) |
| DOM-24 | Kafka producer stubbed in tests that emit | PASS | atlas-buffs: `character/testmain_test.go:10-13` `producertest.InstallNoop()`, no `ResetInstance` cleanup; tests ran 0.333s (no 42s hang). atlas-summons: `beholder_task_test.go:114,209,249` inject capturing `emit` + deterministic `pick`, never reach `producer.ProviderImpl` |

DOM-01..05, DOM-07/08/10..19, DOM-22/23, SUB-*, EXT-*, SCAFFOLD-*, SEC-*:
N/A to this commit (no new service, no new REST POST/PATCH handler, no go.mod
change, no new external HTTP client, no auth/token handling, no Dockerfile/k8s/
topic-config change).

## Summary

### Blocking (must fix)
- None.

### Non-Blocking (should fix)
- None.

**Overall: PASS.** Build/vet/test (incl. `-race`) clean on both modules. The key-type
migration is complete with no surviving `int32`-keyed references; the cross-service
`ApplyCommandBody` mirror is byte-identical with a matching `accumulate,omitempty`
field and is verified by an end-to-end payload round-trip test; per-stat APPLIED
emission is correct in both modes; tenant context is propagated correctly; and the
emit paths are properly stubbed in tests.

---

# Plan Adherence Audit — addendum-hex-accumulation (commit b87046430)

**Plan:** `docs/tasks/task-088-player-summons/addendum-hex-accumulation-plan.md`
**Design:** `docs/tasks/task-088-player-summons/addendum-hex-accumulation-design.md`
**Audit Date:** 2026-06-14
**Branch:** task-088-player-summons
**Base:** commit b87046430 vs parent
**Auditor:** plan-adherence-reviewer

## Executive Summary

All three implementation phases (1–3) of the addendum plan were faithfully
implemented in commit b87046430, with file:line evidence for every task item.
Builds, vet, `go test`, and `tools/redis-key-guard.sh` are clean on both
`atlas-buffs` and `atlas-summons`. The plan's "no atlas-channel change" claim is
confirmed — the commit touches zero channel files. One minor deviation: the
Phase 1 *Verify* step asked for a dedicated JSON round-trip test in **each**
module asserting `accumulate` omitted⇒absent and a cross-mirror byte-identity
assertion; that explicit contract test was not added. The wire shape is instead
verified indirectly (summons `beholder_task_test` unmarshals the produced payload
and asserts `Accumulate:true`), and the `omitempty` tag is correct by inspection.
Non-blocking. **Recommendation: READY_TO_MERGE.**

## Task Completion

| # | Task | Status | Evidence |
|---|------|--------|----------|
| 0.1 | Enumerate `.Buffs()` callers | DONE | Only non-test caller `character/resource.go:39` iterates values (`for _, bs := range cm.Buffs()`), key-type-agnostic; unaffected by `int32`→`string` |
| 0.2 | Confirm only two APPLY producers; others default false | DONE | Other producers exist (atlas-channel, atlas-consumables, atlas-saga-orchestrator `ApplyCommandBody`/buff producers) but none set `accumulate`; correctly left unchanged per plan |
| 1.1 | Add `Accumulate` to atlas-buffs `ApplyCommandBody` | DONE | `kafka/message/character/kafka.go:42` `Accumulate bool \`json:"accumulate,omitempty"\`` |
| 1.2 | Thread `Accumulate` through consumer `handleApply` | DONE | `kafka/consumer/character/consumer.go:56` passes `c.Body.Accumulate` to `Processor.Apply` |
| 1.3 | Add identical field to atlas-summons mirror; thread through provider; update mirror comment | DONE | `summons/buff/producer.go:57` field; `:69,:79` `applyProvider` gains `accumulate bool`; `:90,:91` `ApplyProvider` threads it; comment updated `:45` (`...kafka.go:30-43`) |
| 1.V | JSON round-trip test (each module): omitted⇒absent, true⇒present, cross-mirror identical | PARTIAL | No dedicated contract test in either module. Wire shape verified indirectly: `summon/beholder_task_test.go:142-159` unmarshals the produced payload into `Command[ApplyCommandBody]` and asserts `Accumulate==true`. No "omitted⇒absent" assertion, no cross-module serialize/deserialize test. `omitempty` correct by inspection. |
| 2.1 | `model.go` map `int32`→`string`; getter + Marshal/Unmarshal | DONE | `character/model.go:23` field; `:26` `Buffs() map[string]`; `:52,:66` JSON structs |
| 2.2 | `registry.go` `srcKey`/`statKey` helpers; map-key type change in value-iterating methods | DONE | `registry.go:46` `srcKey`, `:52` `statKey`; `make(map[string]…)` at `:77,:162,:188,:218,:243`; `Cancel`/`GetExpired`/`CancelAll`/`CancelByStatTypes` logic unchanged |
| 2.3 | `registry.Apply(..., accumulate bool)` — branch; return per-stat list | DONE | `registry.go:64` signature returns `([]buff.Model, error)`; `:85-103` accumulate branch stores per `statKey(sourceId,c.Type())`, default branch `srcKey` |
| 2.4 | `processor.Apply(..., accumulate bool)` — emit one APPLIED per stat; disease short-circuit kept | DONE | `processor.go:43` signature; `:55-63` loops `applied` emitting one event per buff; `:44-47` immunity short-circuit retained |
| 2.5 | Update `Processor` interface + mocks | DONE | `processor.go:19` interface updated. No mock file exists for this processor (consumer/tests call concrete impl); all `Apply` call sites updated (consumer + tests pass explicit `false`) |
| 2.T | Tests: accumulate coexist/refresh/per-stat-expiry; Cancel removes all; regression default overwrites | DONE | `registry_test.go:441` DistinctStatsCoexist; `:467` SameStatRefreshes; `:490` PerStatExpiry; `:509` DefaultReplacesWholeSource (regression). All pass |
| 3.1 | Injectable `pick func(n int) int` field on `BeholderTask` (default `rand.Intn`) | DONE | `summon/beholder_task.go:34` field; `:49` default `pick: rand.Intn` |
| 3.2 | `sweepBuff` emits ONE random statup with accumulate=true; empty-pool guard kept | DONE | `beholder_task.go:125-131` pool guard + `c := pool[t.pick(len(pool))]`; `:132` single-element `changes`; emits `ApplyProvider(..., changes, true)`; SKILL pulse + timer advance unchanged |
| 3.3 | producer passes accumulate=true | DONE | wired via 1.3; call at `beholder_task.go:132` passes `true` |
| 3.T | Tests: fixed-pick one APPLY w/ one change + Accumulate:true; multi-pulse union coverage; SourceId stays 1320009 | DONE | `beholder_task_test.go:98` single-pulse (`:158` Accumulate, `:162` one change, `:155` SourceId 1320009); `:232` TestBeholderSweepBuffAccumulatesAcrossPulses union coverage; heal+SKILL assertions retained `:168-172` |
| — | atlas-channel: NO change | CONFIRMED | `git show --name-only b87046430` lists zero `atlas-channel` files |

**Completion Rate:** 16/16 task items implemented (1 — the Phase 1 contract test — only partially as specified).
**Skipped without approval:** 0
**Partial implementations:** 1 (Phase 1 Verify — dedicated JSON contract test)

## Partial / Deferred Tasks

- **Phase 1 Verify (JSON round-trip contract test):** The plan asked for a
  stat-level test in *each* module asserting `accumulate` omitted⇒absent in JSON,
  `true`⇒present, and identical across mirrors. No such dedicated test exists in
  `atlas-buffs` (no `kafka/message/character/*_test.go`) and none in `atlas-summons`
  beyond the indirect assertion in `beholder_task_test.go:142-159`. Impact: low —
  the `json:"accumulate,omitempty"` tag is identical in both mirrors by inspection
  (`atlas-buffs kafka.go:42`, `atlas-summons producer.go:57`) and the produced
  payload is decoded and checked for `Accumulate==true` end-to-end on the summons
  side. The "omitted⇒absent" wire-bytes claim is unverified by an automated test.
  Note: the backend-reviewer section above describes this as "verified by an
  end-to-end payload round-trip test," which is accurate only for the single-module
  summons round-trip, not the cross-mirror byte-identity the plan specified.

## Build & Test Results

Run from the worktree module paths.

| Service | Build | Vet | Tests | redis-key-guard |
|---------|-------|-----|-------|-----------------|
| atlas-buffs | PASS | PASS | PASS (`character` 30 tests incl. 4 new accumulate/regression) | PASS (clean, exit 0) |
| atlas-summons | PASS | PASS | PASS (`summon` incl. 3 beholder tests) | PASS (clean, exit 0) |

Note: `tools/redis-key-guard.sh` exits 0 and prints no FAIL when run normally;
an apparent FAIL only appears when forcing `GOWORK=off`, which breaks module
resolution for unrelated services and is an invocation artifact, not a real
violation. No buffs/summons `.go:` violations exist. No `go.mod` was touched, so
`docker buildx bake` is not required for this change (consistent with the commit
message).

## Overall Assessment

- **Plan Adherence:** MOSTLY_COMPLETE (all functional tasks DONE; one Verify-step
  test partially fulfilled)
- **Recommendation:** READY_TO_MERGE

## Action Items

1. (Optional, non-blocking) Add an explicit `ApplyCommandBody` JSON table test —
   `accumulate` omitted⇒key absent, `true`⇒present — and ideally a cross-mirror
   serialize(summons)/deserialize(buffs) assertion, to lock the Phase 1 contract
   against future drift as the plan's Verify step intended.

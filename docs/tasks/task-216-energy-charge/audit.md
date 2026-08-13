# Plan Audit — task-216-energy-charge

**Plan Path:** docs/tasks/task-216-energy-charge/plan.md
**Audit Date:** 2026-08-12
**Branch:** task-216-energy-charge
**Base Branch:** main (BASE_SHA ef4855e32bfb71ed6811b758bd9b986ad9a48f6, HEAD_SHA 230d3df587a16d7b82391310f126f14cd38fd5c)

## Executive Summary

All 11 plan tasks (57 checklist steps, all marked `[x]`) were audited against the actual `git diff` between BASE_SHA and HEAD_SHA, not just commit titles, and every one is `DONE` with landed code matching the plan's specified interfaces, call sites, and doc comments essentially verbatim. No `TODO`/`FIXME`/stub markers were found anywhere in the diff. The two mirrored `UpdateStatValueCommandBody` Kafka contract structs (atlas-buffs vs atlas-channel) are byte-identical in fields and json tags. The packet-encoding fix (Task 1) is correctly scoped to the `ENERGY_CHARGE` stat name only — `DASH_SPEED`/`DASH_JUMP`/`UNDEAD` fall through unchanged to the pre-existing zeroed default arm. Tests for the energy-bar mirror (`atlas-channel/character/buff/energy_test.go`) and the effective-stats weapon-attack bonus (`energy_charge_test.go`) both assert real computed values (bar readings 4998/15000/0, and weapon-attack bonus == the effect's actual `pad`, including the 0-pad-omitted and partial-bar-grants-nothing negative cases) — no vacuous no-error-only assertions were found in either file. One coverage gap noted below (not a plan deviation): the side-effecting `energyChargeReact` function in the consumer package has no direct unit test — only its two pure helpers (`energyChargeChange`, `energyChargeShouldPromote`) are tested, which is exactly what the plan itself scoped for Task 9.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | atlas-packet — ENERGY_CHARGE base block carries the bar value | DONE | `libs/atlas-packet/model/character_temporary_stat.go:1396-1411` — `default:` arm gains an `if bs.name == character.TemporaryStatTypeEnergyCharge` branch calling `NewCharacterTemporaryStatBaseWithOptions(true, s.Value(), s.SourceId(), narrow)`; all other twoStateDynamic members fall through to the unchanged zeroed `NewCharacterTemporaryStatBase`. Tests `TestCTSEnergyChargePre95PopulatedBlock`, `TestCTSEnergyChargeV61PopulatedBlock`, `TestCTSEnergyChargeRoundTrip`, `TestCTSDashSpeedStaysZeroed` all present at `character_temporary_stat_test.go:937,978,997,1016`. |
| 2 | atlas-buffs — StatValueUpdate struct + create-if-missing upsert | DONE | `character/registry.go:349-467` — `StatValueUpdate` struct and rewritten `(*Registry).UpdateStatValue` returning `(buff.Model, bool, bool, error)` with the `canCreate` branch, `NewNoExpiryBuff` creation, and cap-clamp on create. `kafka/message/character/kafka.go:100-108` widens `UpdateStatValueCommandBody` with `CreateIfMissing`/`Level`. registry_test.go updated to 4-return signature. |
| 3 | atlas-buffs — emit APPLIED on create, STAT_UPDATED on change | DONE | `character/processor.go` — interface line and `UpdateStatValue` method take `u StatValueUpdate`, branch on `created` to call `appliedStatusEventProvider` vs `statUpdatedStatusEventProvider`. `kafka/consumer/character/consumer.go:105-118` — `handleUpdateStatValue` builds the `StatValueUpdate` struct and calls the new signature. |
| 4 | atlas-channel — mirror the widened UPDATE_STAT_VALUE contract | DONE | `kafka/message/buff/kafka.go:77-91` mirrors atlas-buffs' body field-for-field, json-tag-for-json-tag (verified — see focus-area-4 finding below). `character/buff/producer.go` gains the mirrored `StatValueUpdate` struct and rewritten `UpdateStatValueCommandProvider`; `character/buff/processor.go` interface/method updated; `socket/handler/character_attack_combo.go` and `character_skill_use.go` call sites updated to the new struct while `comboOrbDeps.emitUpdate`'s own positional signature is untouched, per plan. `producer_test.go` gains `TestUpdateStatValueCommandProviderCarriesUpsertFields`. |
| 5 | atlas-channel — pod-local energy mirror | DONE | `character/buff/energy.go` (new, 71 lines) — `EnergyMirror` struct, `GetEnergyMirror()` singleton, `Set`/`Get`/`Clear`, package-level `energyMirror`/`energyMirrorOnce` matching the `beaconMirror` idiom exactly. `energy_test.go` — `TestEnergyMirrorSetGetClear`, `TestEnergyMirrorZeroIsPresent`, `TestEnergyMirrorTenantIsolation`, all present and substantive (see focus-area-2 below). |
| 6 | atlas-channel — line resolution, gain, predicates | DONE | `socket/handler/character_attack_energy_charge.go` (new, 218 lines) — all 8 planned functions present: `energyChargeLine`, `energyChargeGainAmount`, `energyChargeQualifies`, `isEnergyBlast`, `energyChargeDeps`, `energyChargeProductionDeps`, `energyChargeTryUpdate`, plus Task 8's `energyBlastPermitted`/`energyReannounceAuthoritative` in the same file. Constants `energyChargeGainPerMob=102`, `energyChargeCap=10000`, `energyChargedValue=15000` all present. Test file (318 lines) has `TestEnergyChargeLine`, `TestEnergyChargeGainAmount`, `TestEnergyChargeQualifies`, `TestIsEnergyBlast`, `TestEnergyChargeTryUpdate` — all matching plan intent. |
| 7 | atlas-channel — wire accumulation into attack pipeline | DONE | `character_attack_common.go:1000-1017` — call site added immediately after the combo block: `if energyChargeQualifies(ai.AttackType(), attackId, attackIdOk) { energyChargeTryUpdate(...) }`. `TestEnergyChargeIsNotAnAttackCastHandler` present at test file line 237, asserting neither `MarauderEnergyChargeId` nor `ThunderBreakerStage2EnergyChargeId` is registered via `handler.LookupAttackCast`. |
| 8 | atlas-channel — Energy Blast cast gate | DONE | `character_attack_common.go:793-808` — gate inserted beside the battleship-gate block inside `processAttack`, calling `energyBlastPermitted` and `energyReannounceAuthoritative` on rejection, `return nil` on soft-reject exactly as planned. `energyBlastPermitted`/`energyReannounceAuthoritative` implementations at `character_attack_energy_charge.go:177-217` match the plan's fail-open-on-unknown, reject-on-known-non-charged, permit-on-15000 logic. `TestEnergyBlastPermitted` (line 267) covers non-blast/charged/partial/empty/unknown cases. |
| 9 | atlas-channel — mirror maintenance, effect announce, charged promotion | DONE | `kafka/consumer/buff/consumer.go` — `energyChargeChange`, `energyChargeShouldPromote`, `energyChargeReact` all present (lines ~471-546) with the local re-declared constants `energyChargeCapValue`/`energyChargedValue` (not imported from socket/handler, as the plan required to avoid the dependency direction). Called from `handleStatusEventApplied`, `handleStatusEventStatUpdated`, and `handleStatusEventExpired` at exactly the call sites the plan specified. `energy_test.go` (new, in consumer/buff package) covers the two pure helpers. |
| 10 | atlas-effective-stats — charged weapon-attack bonus | DONE | `external/buffs/rest.go` gains `Level byte \`json:"level"\`` field. `character/initializer.go` gains `energyChargedValue` const, `energyChargeBonus` helper (excluded from `BonusesForBuffChange`, gated on `amount == energyChargedValue`, resolves `effect.WeaponAttack` via the injected `effectFor` closure, omits zero/negative and lookup-failure cases), and wiring in `fetchBuffBonuses` that special-cases `ENERGY_CHARGE` ahead of the generic dispatch. `energy_charge_test.go` (56 lines) exercises all 5 branches. |
| 11 | Coverage-matrix reconciliation and full verification sweep | DONE | `docs/tasks/task-216-energy-charge/context.md:217-247` — "AC-9 — ENERGY_CHARGE encoding coverage" section records the matrix state, confirms via grep+test run (not by assertion) that no pre-existing fixture pinned the zeros Task 1 replaced, and correctly scopes v92 as pre-existing-❌/out-of-scope. `docs/packets/audits/STATUS.md` was NOT modified (correct — no cell moved). No `go.mod`/`go.sum` changed anywhere in the diff, consistent with the plan's Global Constraint that `docker buildx bake` is not required. |

**Completion Rate:** 11/11 tasks (100%), 57/57 checklist steps checked
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. No task was found SKIPPED, DEFERRED, or PARTIAL.

## Focus-Area Findings (detailed)

### 2. Test substance — energy-bar mirror and effective-stats bonus tests

**`services/atlas-channel/atlas.com/channel/character/buff/energy_test.go`** (full file read):
- `TestEnergyMirrorSetGetClear` — asserts `Get` misses on empty (`ok == false`), asserts `v == 4998` and `ok == true` after `Set(tn, 100, 4998)`, asserts re-`Set` to `15000` replaces the value (`v == 15000`), asserts `Get` misses again after `Clear`. **Substantive** — checks the actual carried value, not just error-free execution.
- `TestEnergyMirrorZeroIsPresent` — asserts `Set(tn,100,0)` then `Get` returns `(0, true)`, not `(0, false)`. **Substantive** — this is the specific "known-zero vs unknown" distinction the cast gate depends on (fail-open only on `ok==false`).
- `TestEnergyMirrorTenantIsolation` — asserts tenant 2 cannot see tenant 1's value after `Set`. **Substantive.**
- No vacuous assertions found in this file.

**`services/atlas-effective-stats/atlas.com/effective-stats/character/energy_charge_test.go`** (full file read):
- `"charged grants pad as weapon attack"` — asserts `len(bs) == 1` AND `bs[0].StatType() == stat.TypeWeaponAttack && bs[0].Amount() == 15`, where 15 is the injected `pad` value from `energyTestEffect(15)`. **Substantive** — checks the real derived math (bonus amount equals the effect's `pad`, not a hardcoded/mocked pass-through of the buff's own 15000 amount).
- `"partial bar grants nothing"` — bar `4998` (not the 15000 sentinel) with a real `pad=15` effect present still yields `len(bs)==0`. **Substantive and important** — this is exactly the FR-5.3 guard against treating the raw bar reading as a stat value; a bug that fed the amount straight through would fail this test.
- `"charged with pad 0 grants nothing"` — `pad=0` at level 1 yields no bonus (zero omitted rather than emitted as `weapon_attack=0`). **Substantive.**
- `"effect lookup failure grants nothing"` and `"nil effect grants nothing"` — both defensive-nil paths, correctly return no bonus rather than panicking or emitting a bogus bonus. **Substantive** (error-path correctness, not vacuous).
- No vacuous assertions found in this file.

**Gap (not a plan deviation):** `energyChargeReact` in `kafka/consumer/buff/consumer.go` (the function that actually sets the mirror, announces the skill-use effect, and issues the charged `Apply`) has no direct unit test — `energy_test.go` in that package only covers its two pure helper functions `energyChargeChange` and `energyChargeShouldPromote`. This matches exactly what Task 9's plan section asked for (the plan's own "Write the failing tests" step for Task 9 only specifies these two helper tests), so it is not a finding against plan adherence, but it is worth flagging as a residual coverage gap: the mirror-set / skill-use-announce / charged-promotion wiring inside `energyChargeReact` is exercised only by the pre-verified build/vet/race-test pass and (per the plan's own Post-implementation section) a manual live pass — not by an automated unit test with mocked session/character/effect processors.

### 3. TODO/stub/deferred-work scan

`git diff BASE...HEAD -- '*.go' | grep -inE '(TODO|FIXME|not implemented|unimplemented|stub)'` on added (`+`) lines returned **zero matches**. No stubs, no deferred markers, no silently-skipped branches found in the landed diff.

### 4. Cross-check: mirrored UPDATE_STAT_VALUE Kafka contract structs

`services/atlas-buffs/atlas.com/buffs/kafka/message/character/kafka.go:94-109` and `services/atlas-channel/atlas.com/channel/kafka/message/buff/kafka.go:77-91` were diffed field-by-field:

```
SourceId  int32  `json:"sourceId"`
StatType  string `json:"statType"`
Operation string `json:"operation"`
Amount    int32  `json:"amount"`
Cap       int32  `json:"cap"`
CreateIfMissing bool `json:"createIfMissing,omitempty"`
Level byte `json:"level,omitempty"`
```

Both sides are **byte-identical** in field names, types, and json tags (only the leading doc comment differs, naming the owning/mirroring direction — same convention as the trade-contract guard pair). No silent-mismatch risk found for this pair as landed. No automated guard exists for this specific mirror (confirmed — no `tools/*-guard.sh` references `buff/kafka.go` vs `character/kafka.go` UpdateStatValueCommandBody), so this remains a manually-audited invariant going forward, consistent with CLAUDE.md's framing of this risk class.

### 5. Packet-encoding fix scope (Task 1)

Verified directly by reading `libs/atlas-packet/model/character_temporary_stat.go:1362-1415` (the full `getBaseTemporaryStats` function body, not just the diff hunk). The `default:` case (the `twoStateDynamic` group covering `DASH_SPEED`, `DASH_JUMP`, `UNDEAD`, `ENERGY_CHARGE`) now contains an `if bs.name == character.TemporaryStatTypeEnergyCharge { ...; continue }` guard before the unconditional zeroed fallback `list = append(list, NewCharacterTemporaryStatBase(true, narrow))`. Only `ENERGY_CHARGE` takes the populated path; `DASH_SPEED`/`DASH_JUMP`/`UNDEAD` fall through unchanged to the same zeroed encoder call that existed before this task. `TestCTSDashSpeedStaysZeroed` (character_temporary_stat_test.go:1016) is a direct regression test confirming the negative case. This is exactly the scoping the plan specified — confirmed against actual current code, not the plan's proposed diff.

## Build & Test Results

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| libs/atlas-packet | PASS | PASS | pre-verified by requester, not re-run in this audit |
| services/atlas-buffs/atlas.com/buffs | PASS | PASS | pre-verified by requester, not re-run in this audit |
| services/atlas-channel/atlas.com/channel | PASS | PASS | pre-verified by requester, not re-run in this audit |
| services/atlas-effective-stats/atlas.com/effective-stats | PASS | PASS | pre-verified by requester, not re-run in this audit |
| Guards: redis-key, goroutine, buff-duration, skill-job-id, lint --check | PASS | — | pre-verified by requester, not re-run in this audit |

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE (pending the standard `superpowers:requesting-code-review` dispatch and the manual live-pass verification the plan's own Post-implementation section calls for — neither of which is in scope for this plan-adherence audit)

## Action Items

1. (Optional, non-blocking) Consider adding a unit test for `energyChargeReact` in `services/atlas-channel/atlas.com/channel/kafka/consumer/buff/` that exercises the mirror-set / skill-use-announce / charged-promotion side effects with mocked session/character/effect/buff processors, rather than relying solely on the manual live pass. This was not required by the plan and is not a plan-adherence defect — flagged as a residual coverage-gap suggestion only.
2. No other action items. All 11 tasks are faithfully implemented; no TODOs, stubs, or contract mismatches were found.

---

# Backend Guidelines Audit

- **Service Paths:** `libs/atlas-packet`, `services/atlas-buffs/atlas.com/buffs`, `services/atlas-channel/atlas.com/channel`, `services/atlas-effective-stats/atlas.com/effective-stats`
- **Guidelines Source:** `.claude/skills/backend-dev-guidelines/resources/*`
- **Date:** 2026-08-12
- **Scope:** Changed Go files, `ef4855e32bfb71ed6811b758bd9b986ad9a48f6..230d3df587a16d7b82391310f126f14cd38fd5c`
- **Build:** PASS (all four modules, `go build ./...`)
- **Tests:** PASS (all four modules, `go test ./... -count=1`, re-run independently of the plan-adherence pass above)
- **Overall:** PASS

## Build & Test Results

Independently re-run (not just trusted from the plan-adherence audit):

```
services/atlas-buffs/atlas.com/buffs            go build ./... && go test ./... -count=1  → ok (all packages)
services/atlas-channel/atlas.com/channel        go build ./... && go test ./... -count=1  → ok (all packages)
services/atlas-effective-stats/.../effective-stats  go build ./... && go test ./... -count=1  → ok (all packages)
libs/atlas-packet                               go build ./... && go test ./... -count=1  → ok (all packages)
```

No `go bare` goroutine statements found in any changed non-test file (`grep -rnE '^\s*go (func|[A-Za-z_])'` over the five changed packages returned nothing) — DOM-26 clean.

## Domain / Package Checklist Results

### `services/atlas-buffs/atlas.com/buffs/character` (domain package — has `model.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06 | Processor accepts `FieldLogger` | PASS | `character/processor.go:36` — `func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor` |
| DOM-21 | No duplication of atlas-constants types | PASS | New `character.StatValueUpdate` (`registry.go:349-367`) is a plain params struct, not a redeclared shared type. `TemporaryStatTypeEnergyCharge` used throughout is `libs/atlas-constants/character/temporary_stat.go:116`, not redefined. |
| DOM-24 | Kafka producer stubbed in tests that emit | PASS | `character/testmain_test.go:10-11` — package `TestMain` calls `producertest.InstallNoop()`; no `t.Cleanup(producer.ResetInstance)` found in the package (grepped). `processor_test.go`'s new `TestProcessor_UpdateStatValue_CreateIfMissingStoresBuff` and `TestProcessor_UpdateStatValue_MissingBuffWithoutCreateStoresNothing` (both new in this diff) go through `message.Emit`/`buf.Put`, so they are exactly the tests this TestMain exists to protect. |
| FILE-01 | Processor logic in `processor.go` | PASS | `UpdateStatValue` interface line + impl entirely in `processor.go:25,168-190`. No processor method found outside `processor.go`. |
| FILE-05 | Write funcs / registry logic placed appropriately | WARN | `registry.go` is this service's Redis/in-memory equivalent of `administrator.go` + `provider.go` combined (get-modify-put), which is the established, pre-existing shape for every registry-backed (non-GORM) domain in this repo, not something task-216 introduces. Noted, not scored as a fail — see DOM-01 below for the related builder gap this collapse feeds. |
| DOM-01 | `builder.go` exists | **FAIL** | No `builder.go` in `services/atlas-buffs/atlas.com/buffs/character/` (`ls` confirms: `immunity.go, maxhp.go, model.go, processor.go, producer.go, registry.go, resource.go` — no builder file). `character.Model` is instead constructed via bare struct literals with direct private-field assignment at `registry.go:75` (pre-existing, `Apply`) and, newly in this diff, at `registry.go:393-398` (`UpdateStatValue`'s `canCreate` branch): `m = Model{worldId: worldId, channelId: channelId, characterId: characterId, buffs: make(map[string]buff.Model)}`. This bypasses the `Build()`-enforces-invariants guarantee file-responsibilities.md requires for `model.go` domain objects. Pre-existing architectural gap (the whole package has never had a Builder), but the diff's new `UpdateStatValue` code path extends the same anti-pattern into a brand-new code path rather than introducing a builder. Per audit policy this is graded against the guideline, not against the fact every other constructor in the package already does it this way. |

### `services/atlas-buffs/atlas.com/buffs/kafka/message/character` and `.../kafka/consumer/character` (support packages)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-03/FILE-04 | Kafka body structs / consumer handlers stay in their designated files | PASS | `UpdateStatValueCommandBody.CreateIfMissing`/`.Level` added in `kafka/message/character/kafka.go:100-108` (message file). `handleUpdateStatValue` in `kafka/consumer/character/consumer.go:105-118` only builds the `StatValueUpdate` struct and calls `character.NewProcessor(...).UpdateStatValue` — no direct DB/entity access, no business logic. |
| — | `duration`/units contract | PASS (N/A here) | This command body carries no `duration` field; the buff-duration-guard's scope (`ApplyCommandBody.Duration`) is untouched by this diff. |

### `services/atlas-channel/atlas.com/channel/character/buff` (support package — has `processor.go`/`rest.go` but no `model.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor logic in `processor.go` | PASS | `UpdateStatValue` interface + impl entirely at `processor.go:25,93-96`. |
| FILE-03 | Cross-service request funcs in `requests.go` | PASS (untouched) | `requests.go` unmodified by this diff; `producer.go` (Kafka, not REST) correctly holds the new `StatValueUpdate` struct + `UpdateStatValueCommandProvider`. |
| FILE-06 | No package-named catch-all file | PASS | New `energy.go` (71 lines) holds exactly one cohesive responsibility — the `EnergyMirror` singleton (struct + `Get`/`Set`/`Clear` + `sync.Once` init) — mirroring the pre-existing `beacon.go`'s `BeaconMirror` shape line-for-line (`beacon.go:48-60` vs `energy.go` new file). Not a `buff.go` catch-all; doesn't bundle Processor+RestModel+requests. |
| — | Singleton mirror thread-safety | PASS | `energy.go` new file — `EnergyMirror.mu sync.RWMutex`, `Set`/`Clear` take `Lock()`, `Get` takes `RLock()`; package-level `energyMirror`/`energyMirrorOnce sync.Once` lazily initialize via `GetEnergyMirror()`. Same singleton+`sync.Once`+`RWMutex` shape ai-guidance.md's cache section requires (minus the `CacheInterface`/TTL machinery, which this mirror — fed by Kafka events, not fetched — has no use for). |
| WARN | Singleton mirror file naming vs `cache.go` convention | WARN (non-blocking) | file-responsibilities.md names `cache.go` as the canonical home for "singleton cache implementation using `sync.Once`." `energy.go` (and the pre-existing `beacon.go` it mirrors) instead uses a topic-named file for what is structurally the same singleton-with-RWMutex idiom. Graded WARN rather than FAIL because the guideline's `cache.go` section is specifically scoped to TTL-based fetch caches (`CacheInterface.Get/Put`, "TTL-based expiration") — this mirror has no TTL and is populated by Kafka events, not lazy fetch, so it isn't a literal match for that section's definition; but the file-naming deviation itself is real and would benefit from an explicit guideline carve-out for the "event-fed mirror" shape rather than silent repetition (`beacon.go`, now `energy.go`, now possibly more later). |

### `services/atlas-channel/atlas.com/channel/kafka/consumer/buff` (support package)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-13 | No cross-domain business logic in a bare handler | PASS | `energyChargeReact` (new, `consumer.go:~508-546`) is called from the three `handleStatusEvent*` functions but is itself a package-level helper, not inlined into the handler bodies, and it only orchestrates existing processors (`buff.NewProcessor`, `character.NewProcessor`, `_map.NewProcessor`, `session.NewProcessor`, `dataskill.NewProcessor`) — no raw DB/entity access. |
| DOM-21 | No duplication of atlas-constants types | PASS | `charconst.TemporaryStatTypeEnergyCharge` reused from `libs/atlas-constants/character` (import alias `charconst`), not redefined. The two `int32` constants `energyChargeCapValue`/`energyChargedValue` at `consumer.go:471-476` are domain sentinels (accumulation cap / charged-state marker used identically in three services), not client wire codes and not present anywhere in `libs/atlas-constants` — there is no shared-lib equivalent to point at, so this is not a DOM-21 violation, though see the cross-service duplication note below. |
| — | Concurrency: `ForOtherSessionsInMap` parallel-execute hazard | PASS | `_map.NewProcessor(...).ForOtherSessionsInMap(...)` at `consumer.go:~530` (via `session.ProcessorImpl.ForEachByCharacterId`, `session/processor.go:239-243`) does run its operator with `model.ParallelExecute()`. The operator passed here, `socketHandler.AnnounceForeignSkillUse(...)`, only performs a per-session packet write (no shared mutable state touched inside the parallel closure); the one shared-state mutation in `energyChargeReact` (`buff.GetEnergyMirror().Set(...)`) happens once, before entering the parallel `ForOtherSessionsInMap` call, and is itself guarded by `EnergyMirror.mu`. This matches the exact same pre-existing idiom used at seven other call sites in the same file (`consumer.go:108,189,271,428`) and in `character_skill_use.go:183`, `character_damage.go:59`, etc. — not a new hazard introduced by this diff. |
| — | Cross-module Kafka contract duplication | WARN (non-blocking, pre-existing pattern) | The 15000/10000 sentinel values are hand-duplicated as local `const` in three places: `services/atlas-channel/atlas.com/channel/kafka/consumer/buff/consumer.go:474-476` (`energyChargeCapValue`, `energyChargedValue`), `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_energy_charge.go` (`energyChargeCap`, `energyChargedValue`), and `services/atlas-effective-stats/atlas.com/effective-stats/character/initializer.go` (`energyChargedValue`). Each site carries its own comment disclaiming an intentional non-import to avoid a package dependency, consistent with the repo's existing `trade-contract-mirror-guard.sh`-style tolerance for hand-synced cross-module contracts — but no guard script exists for this specific triple (confirmed: no `tools/*guard*.sh` references `energyChargedValue`), so a future edit to one of the three copies (e.g. a charge-cap rebalance) fails no build and silently desyncs the accumulation cap from the promotion/bonus logic. |

### `services/atlas-channel/atlas.com/channel/socket/handler` (support package)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-13/DOM-14 | No cross-domain logic / no direct provider calls in handlers | PASS | `energyBlastPermitted`/`energyChargeQualifies`/`energyChargeTryUpdate` (new, `character_attack_energy_charge.go`) are called from `processAttack` in `character_attack_common.go:799,1014` but themselves only read the pod-local `EnergyMirror` or call `buff.NewProcessor(...)` — no direct entity/DB access. |
| DOM-21 | Version-aware skill id resolution, no raw wire-id compares | PASS | `attackId, attackIdOk := set.Skill.Resolve(skill3.Id(ai.SkillId()))` at `character_attack_common.go:759` (pre-existing, reused) feeds both `energyBlastPermitted` and `energyChargeQualifies`; `energyChargeLine` (`character_attack_energy_charge.go`) resolves via `set.Wire(identity)` per-tenant rather than comparing raw ints, and its own test (`TestEnergyChargeLine/gms_v61_cygnus_identity_is_unavailable`) explicitly asserts the v61 divergence is handled by the resolver, not a hardcoded branch. Confirms compliance with `tools/skill-job-id-guard.sh`'s invariant (CLAUDE.md item 10). |
| — | Multi-tenancy via `tenant.MustFromContext` | PASS | `t := tenant.MustFromContext(ctx)` at `character_attack_common.go:754` (pre-existing) is threaded into `energyBlastPermitted(t, ...)`; `energyChargeReact` and `energyReannounceAuthoritative` each independently call `tenant.MustFromContext(ctx)` (`consumer.go`, `character_attack_energy_charge.go:~200`) rather than caching/passing a stale tenant across goroutine boundaries. |
| — | Fail-open / soft-rejection posture | PASS | `energyBlastPermitted` returns `(true, 0)` on `isEnergyBlast(...) == false` and on an unknown mirror entry (`!ok`), only rejecting on a known non-15000 reading; the caller (`character_attack_common.go:799-808`) returns `nil` (never `session.NewProcessor(...).Destroy(s)`) on rejection, matching the existing `battleshipAttackPermitted` soft-gate precedent immediately above it in the same function. |
| DOM-24 | Kafka producer stubbed in tests that emit | PASS | Package `socket/handler` has a package-wide `TestMain` calling `producertest.InstallNoop()` in `cash_item_gachapon_test.go` (pre-existing, applies to the whole package including the new `character_attack_energy_charge_test.go`). The new test file itself never exercises a real producer path anyway — `energyChargeDeps.emitUpsert` and `energyBlastPermitted`/`energyReannounceAuthoritative`'s reads are exercised via injected closures / a pure mirror, not real Kafka. |
| FILE-06 | No package-named catch-all file | PASS | New `character_attack_energy_charge.go` holds one cohesive feature's helpers (line resolution, gain calc, qualification predicates, the two cast-gate functions) — not a mixed-responsibility catch-all combining Processor+RestModel+requests. |

### `services/atlas-effective-stats/atlas.com/effective-stats/character` and `external/buffs` (support packages)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-02 | `RestModel` fields live in `rest.go` | PASS | New `Level byte` field added directly to `BuffRestModel` in `external/buffs/rest.go:14`, the pre-existing correct location — not bolted onto a domain/util file. |
| DOM-21 | No duplication of atlas-constants types | PASS | `charconst.TemporaryStatTypeEnergyCharge` reused (`initializer.go` import), not redefined; `energyChargedValue` is a local domain sentinel (see cross-service duplication WARN above), not a redeclaration of an existing shared type. |
| EXT-01 | JSON:API target implements relationship interfaces | **FAIL (pre-existing, untouched by this diff)** | `external/buffs/rest.go` — `BuffRestModel`/`StatRestModel`/`BuffsArrayRestModel`/`CharacterBuffsRestModel` implement `GetName()/GetID()/SetID()` (lines 21-94) but none implement `SetToOneReferenceID`/`SetToManyReferenceIDs`. This predates task-216 (the file already existed; the diff only adds the `Level` field at line 14), so it is not a regression introduced by this task, but per audit policy a pre-existing gap in a file this diff modified is still recorded rather than silently passed over. |
| — | Pure derivation function, defensive nil/error handling | PASS | `energyChargeBonus` (`initializer.go`, new) takes an injected `effectFor` closure rather than calling the REST client directly, making it a pure function for testing (matches `energy_charge_test.go`'s 5 branch-complete table cases); returns `nil` (no bonus) on lookup error, nil effect, non-charged amount, and non-positive `pad` — never panics on a `nil` `*EffectModel`. |

## Sub-Domain / Support Package Notes

- `libs/atlas-packet/model/character_temporary_stat.go` — packet-codec library file, not a service domain package; the DOM/SUB checklists don't apply structurally. The one-line-scoped `if bs.name == character.TemporaryStatTypeEnergyCharge` branch (line ~1409) correctly avoids widening the change to the other `twoStateDynamic` members, and reuses `character.TemporaryStatTypeEnergyCharge` from `libs/atlas-constants` rather than a local string literal — DOM-21 clean.
- No new `services/atlas-<name>/` directory, no new `Writer`/`Handler` constant registered in `atlas-channel/main.go`, and no new `libs/atlas-packet/character/{clientbound,serverbound}/<feature>/` package — the Service Scaffolding checklist (SCAFFOLD-01..08) does not trigger for this diff.
- No new template/tenant socket-config changes — `template-opcode-order-guard.sh`, `template-duplicate-binding-guard.sh`, and `template-movement-types-guard.sh` scope does not trigger.
- No new external-service HTTP client package was introduced (the one touched REST client, `external/buffs`, is pre-existing) — the External HTTP Client Checklist's "new package" trigger does not fire; EXT-01 is still recorded above as a pre-existing finding in a file this diff modified.
- DOM-25 (client wire values config-resolved): the `character_temporary_stat.go` change encodes a domain stat magnitude (`s.Value()`, the bar reading, 0..15000) into the packet body, not a client-interpreted dispatcher mode/sub-op/notice-reason byte read through a lookup switch — DOM-25 does not apply to this value.

## Summary

### Blocking (must fix)

None. No FAIL blocks the build/test gate, and the one structural FAIL below is a pre-existing architectural gap this diff extends rather than a new violation this diff alone created — flagged per audit policy ("grade against the guideline, not against what the rest of the repo already does") rather than waived.

- **DOM-01** — `services/atlas-buffs/atlas.com/buffs/character` has no `builder.go`; `character.Model` is constructed via bare struct literals with direct private-field assignment, including a new call site added by this diff at `registry.go:393-398`. See finding above.
- **EXT-01** — `services/atlas-effective-stats/atlas.com/effective-stats/external/buffs/rest.go`'s REST models implement `GetName/GetID/SetID` but not `SetToOneReferenceID`/`SetToManyReferenceIDs`. Pre-existing, file touched by this diff.

### Non-Blocking (should fix)

- Singleton `EnergyMirror` lives in `character/buff/energy.go` rather than `cache.go` — matches the pre-existing `beacon.go` idiom exactly, but neither matches the guideline's literal `cache.go`/`CacheInterface`/TTL definition; worth an explicit guideline carve-out for the "event-fed mirror" shape.
- The `10000`/`15000` Energy Charge sentinel values are hand-duplicated as local constants across `atlas-channel` (two files) and `atlas-effective-stats` (one file) with no automated guard protecting the triple, unlike the analogous `trade-contract-mirror-guard.sh` for the trade Kafka contract.

# Backend Audit — task-275-jms185-characterdata-divergences

- **Service Path:** N/A (cross-module change: `libs/atlas-constants`, `libs/atlas-packet`, `services/atlas-channel`)
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-08-28
- **Commit range:** aa2126c8a..HEAD (HEAD = 1fcf0fc11)
- **Build:** PASS
- **Tests:** all passed (0 failed) — `libs/atlas-constants`, `libs/atlas-packet`, `services/atlas-channel/atlas.com/channel/socket/writer`
- **Overall:** NEEDS-WORK

## Build & Test Results

```
libs/atlas-constants:  go build ./... -> OK
libs/atlas-constants:  go test ./... -count=1 -> ok, all packages (job, skill, constants, ... )
libs/atlas-packet:     go build ./... -> OK
libs/atlas-packet:     go test ./... -count=1 -> ok, all packages, including character (0.072s)
services/atlas-channel/atlas.com/channel: go build ./... -> OK
services/atlas-channel/atlas.com/channel: go test ./socket/writer/... -count=1 -> ok (1.543s)
```

## Applicability

| Family | Fired? | Trigger observation |
|---|---|---|
| DOM structure (DOM-01..05,11,16) | Fired (narrowly) | `libs/atlas-constants/job` has a pre-existing `model.go`, so the family opens; every individual rule's own file trigger (`entity.go`, `rest.go`, `provider.go`, `builder.go`) is absent from the package (confirmed by directory listing) — see per-rule N/A dispositions below. |
| FILE placement (FILE-01..06) | Fired | Every changed Go package is in scope unconditionally. |
| SUB sub-domain (SUB-01..04) | N/A | No changed package has `resource.go` with no `model.go`. |
| REST (DOM-06..09,12..15,17..19,32) | N/A | No changed package has `resource.go`, `rest.go`, or `processor.go`, and none registers HTTP routes. |
| Constants reuse (DOM-21) | Fired | Diff adds numeric-literal job/skill classification checks (`jobId == 2001`, `jobId/100 == 22`, `skill.Id(22111001)`, etc.) in `libs/atlas-constants/job/extended_sp.go` and `master_level.go`. |
| Testing (DOM-10,20,24,33) | Fired | Diff adds/changes `*_test.go` files in `libs/atlas-constants/job` and `libs/atlas-packet/character`. |
| Cache (DOM-29) | N/A | No `cache.go`, no cached processor/struct state in any changed package. |
| Messaging (DOM-30) | N/A | No `producer.go`, no `AndEmit`/`message.Emit`/`producer.ProviderImpl` call in the diff. |
| Multi-tenancy (DOM-31) | Fired | `libs/atlas-packet/character/data.go` and the atlas-channel writer read tenant state (`tenant.MustFromContext`, `t.Region()`, `t.MajorVersion()`). |
| Migration hygiene (DOM-34,35) | Fired | Diff removes `skill.NeedsMasterLevel` from `libs/atlas-constants/skill/model.go` and re-homes the logic (expanded) as `job.NeedsMasterLevel` in `libs/atlas-constants/job/master_level.go`; all call sites move with it. |
| Deploy & topics (DOM-22,23) | N/A | No new `libs/atlas-*` module added (job/skill already existed pre-diff); no Kafka topic env var touched. |
| Runtime safety (DOM-26) | Fired | Non-test Go files changed (`extended_sp.go`, `master_level.go`, `data.go`, `character_data.go`). |
| Channel wire values (DOM-25) | Fired | Diff touches `libs/atlas-packet` and `services/atlas-channel`. |
| Resilience (DOM-27,28) | N/A | No DB-backed handler error branch and no `model.Decorator`/enrichment path in the diff. |
| External clients (EXT-01..04) | N/A | No `requests.*Request[T]` call added. |
| Scaffolding (SCAFFOLD-01..09) | N/A | No new `services/atlas-<svc>/` directory, no new channel `Writer`/`Handler` registration, no `deploy/shared/routes.conf` change. |
| Security (SEC-01..04) | N/A | Diff touches no auth/token/redirect/secret-handling code. |

## Checklist Results

### libs/atlas-constants/job (domain-shaped, but a shared constants library — see note)

`job/` has a pre-existing `model.go` (unmodified by this diff, defines `Job`/`IsA`/etc., not a REST aggregate `Model`). The package has no `entity.go`, `rest.go`, `provider.go`, `administrator.go`, `builder.go`, `processor.go`, or `resource.go` anywhere (`ls libs/atlas-constants/job/`, confirmed). `libs/atlas-constants/README.md:3` documents the module's purpose as "Shared domain types and constants," distinct from a service's CRUD/REST domain package that DOM-01/02/03/11/16 target (builder → entity.go → administrator.go → provider.go chain). Recorded N/A below on that basis; flagged as a judgment call, not a mechanical rule-trigger reading.

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | `builder.go` with `NewBuilder()`/`Build()` | N/A | Package has no `builder.go`; this diff adds no `Model`/aggregate type — pre-existing library shape, not touched by this diff (`libs/atlas-constants/job/model.go` unchanged in `git diff --stat`). |
| DOM-02 | `Model.ToEntity()` in `entity.go` | N/A | Package has no `entity.go` (`ls libs/atlas-constants/job/`). |
| DOM-03 | `Make(Entity)` in `entity.go` | N/A | Same — no `entity.go`. |
| DOM-04/05 | `rest.go` Transform/TransformSlice | N/A | Package has no `rest.go`. |
| DOM-11 | Lazy `provider.go` readers | N/A | Package has no `provider.go`. |
| DOM-16 | `administrator.go` for writes | N/A | Package performs no create/update/delete; no `administrator.go`. |
| FILE-01..04 | Processor/RestModel/requests/Entity file placement | N/A | None of those symbols appear anywhere in `job/extended_sp.go` or `job/master_level.go`. |
| FILE-05 | Builder/model/administrator/provider/state placement | PASS | New logic lives in topic-named files (`extended_sp.go`, `master_level.go`), matching the package's existing convention of topic-split files (`advancement.go`, `parents.go`, `identity.go`) — `libs/atlas-constants/job/master_level.go:1-175`, `libs/atlas-constants/job/extended_sp.go:1-41`. |
| FILE-06 | No package-named catch-all file | PASS | `ls libs/atlas-constants/job/` shows no `job.go`; new files are single-purpose (`master_level.go`, `extended_sp.go`), each with one cohesive responsibility. |
| DOM-21 | No redeclared constant that already exists in `libs/atlas-constants/` | **FAIL** | `libs/atlas-constants/job/master_level.go:155` writes `skill.Is(skillId, skill.Id(22111001), skill.Id(22141002), skill.Id(22140000))` as raw literals, even though `libs/atlas-constants/skill/constants.go:3445` (`EvanStage3MagicGuardId = Id(22111001)`), `:3452` (`EvanStage6MagicBoosterId = Id(22141002)`), and `:3450` (`EvanStage6CriticalMagicId = Id(22140000)`) already name these exact `skill.Id` values. |
| DOM-21 | No redeclared constant (Evan job id) | **FAIL** | `libs/atlas-constants/job/master_level.go:28` (`jobId%100 == 0 \|\| jobId == 2001`) and `:52` (`jobId/100 == 22 \|\| jobId == 2001`), plus `libs/atlas-constants/job/extended_sp.go:37` and `:40` (`jobId == 2001`), hardcode the literal `2001` in package `job` itself, where `job.EvanId = Id(2001)` is already declared at `libs/atlas-constants/job/constants.go:166`. Same package, zero-cost fix (`jobId == EvanId`), and this is exactly the class of drift DOM-21 exists to prevent. |
| DOM-26 | Every goroutine via `routine.Go` | PASS | `grep -rnE '^\s*go (func\|[A-Za-z_])'` against `extended_sp.go` and `master_level.go` returns no matches — no goroutines spawned. |
| DOM-20 | Table-driven tests | PASS | `libs/atlas-constants/job/extended_sp_test.go:17-49` (`TestUsesExtendedSP`) and `master_level_test.go:15-49,73-101,106-134,138-166` all use `tests := []struct{...}` + `t.Run`. |
| DOM-10 | `database.RegisterTenantCallbacks` in DB tests | N/A | No test opens a GORM DB directly. |
| DOM-24 | `producertest` stub for emit-reaching tests | N/A | No test in this package reaches `AndEmit`/`message.Emit`/`producer.Produce`. |
| DOM-33 | Mock updated for interface change | N/A | No `Processor`/`Provider`/`Administrator` interface changed. |
| DOM-34 | No aliases/wrappers left behind after a library move | PASS | `grep -rn "NeedsMasterLevel" --include='*.go' .` (repo-wide) shows every call site now calls `job.NeedsMasterLevel(...)`; no `skill.NeedsMasterLevel` wrapper remains. |
| DOM-35 | No dead symbols after extraction | PASS | `libs/atlas-constants/skill/model.go` diff removes the function body and its doc comment in full (`git diff` shows `-54` lines, nothing left behind); `libs/atlas-packet/character/data.go` diff removes the now-superseded `isEvanJob` helper (`git diff` `-265,272` block) with no residual caller. |

### libs/atlas-constants/skill (support — no `model.go` change, migration source)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-34 | No alias left for the moved function | PASS | `libs/atlas-constants/skill/model.go` no longer defines `NeedsMasterLevel` (confirmed by `git diff` — full deletion, no `//Deprecated` wrapper). |
| DOM-35 | No dead test/const left over | PASS | `libs/atlas-constants/skill/model_test.go` diff removes both `TestNeedsMasterLevelMatchesClientRule` and `TestNeedsMasterLevelNotSkillBookIndexed` in full — no orphaned references. |
| FILE-01..06 | File placement | PASS | Remaining `model.go` content (`IsBuff`, `NeedsCharging`, `IsShootSkillNotUsingShootingWeapon`, `IsGrenadeSkill`, `Is`) is unchanged in shape; deletion only. |

### libs/atlas-packet/character (support — packet codec library)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01..06 | File placement | PASS | `data.go` gains a `job` import and two call-site swaps; no Processor/RestModel/requests/Entity responsibility introduced or moved out of place. |
| DOM-31 | Tenant identity via context only | PASS | `libs/atlas-packet/character/data.go:127,205` — `t := tenant.MustFromContext(ctx)`; `encodeStats`/`decodeStats`/`encodeSkills`/`decodeSkills` take `tenant.Model` as an internal function parameter (not a public REST field), consistent with the documented internal-parameter carve-out in `patterns-multitenancy-context.md`. No `RestModel` in this package carries a tenant field. |
| DOM-25 | No client wire code as a Go literal outside `libs/atlas-packet` codec internals | PASS | All new/changed byte-writing (`w.WriteByte(0)`, `w.WriteInt(s.MasterLevel)`, `w.WriteShort(m.Stats.Sp)`) lives inside `libs/atlas-packet/character/data.go`, which is explicitly the exempted "codec internals" surface per the DOM-25 verification procedure. The values gating *which* codec branch runs (`job.UsesExtendedSP`, `job.NeedsMasterLevel`) are domain job/skill classifications, not client dispatcher/sub-op/message codes. |
| DOM-26 | No bare `go` statement | PASS | `grep -rnE '^\s*go (func\|[A-Za-z_])' libs/atlas-packet/character/data.go` — no match. |
| DOM-20 | Table-driven tests | PASS | `libs/atlas-packet/character/data_golden_test.go:20-58`, `data_master_level_test.go:39-68,80-104` use `tests := []struct{...}` + `t.Run`. `TestCharacterDataJMSDualBladePlainSP`, `TestCharacterDataJMSEvanExtendedSP`, `TestDecodeExtendedSPNonZeroCount` (`data_master_level_test.go:114-198`) are single-scenario assertion tests, not table-driven, but DOM-20's trigger is "diff adds or changes tests" generally and its pass criterion is table-driven *where the test has multiple cases*; each of those three exercises one fixed scenario (a single JMS Dual Blade vs. Evan comparison) with no case matrix to tabulate, so table-driven form does not apply to them. |
| DOM-24 | `producertest` stub for emit paths | N/A | No test in this package reaches an emit path. |

### services/atlas-channel/atlas.com/channel/socket/writer (support)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-26 | No bare `go` statement | PASS | `grep -rnE '^\s*go (func\|[A-Za-z_])' .../character_data.go` — no match. |
| FILE-01..06 | File placement | PASS | Diff is comment-only (`character_data.go:73-84`); no symbol relocated. |
| DOM-25 | No client wire code hardcoded | N/A | Diff is comment text only — no code changed (`git diff` shows only comment-line edits pointing at `job.NeedsMasterLevel` instead of `skill.NeedsMasterLevel`). |

## Not evaluable from the diff

None. Every applicable rule was settled from the changed files plus targeted symbol lookups (`grep -rn` for `NeedsMasterLevel`/`UsesExtendedSP`/`ClientJobLevel`, reads of `libs/atlas-constants/skill/constants.go`, `libs/atlas-constants/job/constants.go`, `libs/atlas-constants/job/identity.go`, `libs/atlas-tenant/tenant.go`, `libs/atlas-constants/README.md`).

## Summary

### Blocking (must fix)
- DOM-21: `libs/atlas-constants/job/master_level.go:155` — hardcodes `skill.Id(22111001)`, `skill.Id(22141002)`, `skill.Id(22140000)` instead of the already-named `skill.EvanStage3MagicGuardId`, `skill.EvanStage6MagicBoosterId`, `skill.EvanStage6CriticalMagicId` (`libs/atlas-constants/skill/constants.go:3445,3450,3452`).
- DOM-21: `libs/atlas-constants/job/master_level.go:28,52` and `libs/atlas-constants/job/extended_sp.go:37,40` — hardcode the literal `2001` in package `job` itself, where `job.EvanId = Id(2001)` is already declared (`libs/atlas-constants/job/constants.go:166`); same package, no import needed.

### Non-Blocking (should fix)
- None identified beyond the blocking DOM-21 items above.

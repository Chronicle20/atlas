# Backend Audit — atlas-monsters (task-279-carnival-mob-skills)

- **Service Path:** services/atlas-monsters, libs/atlas-constants
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-08-28
- **Build:** PASS
- **Tests:** all passed (0 failed)
- **Overall:** PASS

## Build & Test Results

```
libs/atlas-constants: go build ./... -> clean; go test ./monster/... -count=1 -> ok (0.003s)
services/atlas-monsters/atlas.com/monsters: go build ./... -> clean
go test ./monster/... -count=1 ->
  ok  atlas-monsters/monster                26.232s
  ok  atlas-monsters/monster/consumable     0.028s
  ok  atlas-monsters/monster/drop           0.037s
  ok  atlas-monsters/monster/information    15.382s
  ok  atlas-monsters/monster/mobskill       0.008s
```

Isolated re-run of only the new tests (`TestExecuteStatBuff_Carnival*`, `TestCarnival*`,
`TestUseSkill_SealSkill*`, `TestUseSkill_Skill157*`) confirms no unstubbed-Kafka-retry stall
(`go test ./monster/... -run '...' -v -count=1` -> 0.919s total for that package, all PASS).

## Applicability

| Family | Fired? | Trigger observation |
|---|---|---|
| DOM structure | Yes | changed package `monster` has `model.go` (pre-existing, unchanged by diff) |
| FILE placement | Yes | every changed Go package runs FILE-* unconditionally |
| SUB sub-domain | N/A | `monster` package has `model.go` — not a sub-domain package |
| REST | Yes | `monster` package has `resource.go`/`rest.go`/`processor.go` (processor.go changed) |
| Constants reuse (DOM-21) | Yes | `libs/atlas-constants/monster/skill.go` adds a numeric-literal classification (skill types 150-157) and a new named const (`SkillCategoryCarnivalBuf`) |
| Testing (DOM-10/20/24/33) | Yes | diff adds/changes `_test.go` files; new tests reach `ApplyStatusEffect` -> `producer.ProviderImpl` transitively |
| Cache (DOM-29) | N/A | no `cache.go` touched, no cached processor state added |
| Messaging (DOM-30) | N/A | diff adds no new `AndEmit`/`message.Emit`/direct `producer.ProviderImpl` call site; the only emission reached (`ApplyStatusEffect` at processor.go:1549) is pre-existing, unmodified by this diff |
| Multi-tenancy (DOM-31) | N/A | diff does not touch `rest.go`, does not read/pass tenant or trace state as a new field/param |
| Migration hygiene (DOM-34/35) | N/A | diff does not move/extract symbols between a service and a `libs/atlas-*` module — it *adds* new constants directly in `libs/atlas-constants` |
| Deploy & topics (DOM-22/23) | N/A | diff adds no `libs/atlas-*` module, no Kafka topic env var |
| Runtime safety (DOM-26) | Yes (family) / N/A (rule) | non-test Go files changed (processor.go, picker.go, skill.go), but no bare `go` statement added — `git diff ... | grep -E '^\+\s*go (func|[A-Za-z_])'` returns nothing |
| Channel wire values (DOM-25) | N/A | diff does not touch `services/atlas-channel` or `libs/atlas-packet`; no domain service emits a new client-interpreted byte |
| Resilience (DOM-27/28) | N/A | diff changes no handler writing `http.StatusInternalServerError`; no `model.Decorator`/enrichment path changed |
| External clients (EXT-*) | N/A | diff calls no `requests.RootUrl`/`requests.*Request[T]` |
| Scaffolding (SCAFFOLD-*) | N/A | diff adds no `services/atlas-<svc>/` directory, no channel Writer/Handler, no routes.conf change |
| Security (SEC-*) | N/A | atlas-monsters does not handle auth/tokens/redirects/secrets in this diff |
| Foundational: patterns-provider.md | N/A | diff defines/composes no providers |
| Foundational: patterns-functional.md | N/A | diff defines no curried constructors/decorators/combinators |

## Checklist Results

### libs/atlas-constants/monster (support package — no `resource.go`/`model.go`; pure constants/lookup functions)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-05 | Enums in `state.go` | N/A | `skill.go` is the pre-existing home for all `SkillType*`/`SkillCategory*` consts; not a new file, and this file (not `state.go`) is the established convention for this const family — no `model.go`/`builder.go`/`administrator.go`/`provider.go` exist in this package for the rule's other clauses to apply against |
| FILE-06 | No catch-all file bundling ≥2 responsibilities | PASS | `libs/atlas-constants/monster/skill.go` holds only skill-type constants and pure lookup/classification functions (`SkillCategory`, `SkillTypeToStatusName`, `IsAoeSkill`, `SkillNameToId`, `ReflectKindForSkill`) — a single-purpose classification file, not a Processor/RestModel/requests bundle |
| DOM-21 | No redeclaration of a type/const that already exists in `libs/atlas-constants` | PASS | The new `SkillTypeCarnivalPAD..SkillTypeCarnivalSealSkill` (150-157) and `SkillCategoryCarnivalBuf` constants are declared **in** `libs/atlas-constants/monster/skill.go:54-61,13` — this diff *is* the canonical declaration, not a downstream redeclaration. `grep -rn "= 150\|= 151\|...\|= 157" services/atlas-monsters/atlas.com/monsters/monster/*.go` (excluding `_test.go`) finds no service-side hardcoding of these skill-type values (only unrelated `MonsterSkillPickerSweepInterval`/`AggroSweepInterval` matches) |
| DOM-20 | Tests are table-driven | PASS | `libs/atlas-constants/monster/skill_test.go:34-199` — `TestSkillTypeToStatusName_Carnival`, `TestIsAoeSkill_CarnivalAndRegressions`, `TestSkillNameToId_Carnival`, `TestSkillCategory_Carnival` are all `cases := []struct{...}{}` + range-loop table tests |

### services/atlas-monsters/atlas.com/monsters/monster (domain package — has `model.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06 | Processor constructor takes `logrus.FieldLogger`, never `*logrus.Logger` | PASS | `services/atlas-monsters/atlas.com/monsters/monster/processor.go:142` — `func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor` (unchanged by this diff, already compliant) |
| DOM-26 | Every goroutine spawned via `routine.Go` | PASS | no bare `go` statement added in the diff — `git diff bda6566f3..23b2f2b6a -- services/atlas-monsters libs/atlas-constants \| grep -E '^\+\s*go (func\|[A-Za-z_])'` returns no matches |
| DOM-30 | Writes emit through `AndEmit`+`message.Buffer`, not a direct `producer.ProviderImpl` call from the success path | N/A for this diff | The only emission the new `CARNIVAL_BUFF` dispatch path reaches, `ApplyStatusEffect` at `processor.go:1549` (`_ = producer.ProviderImpl(p.l)(p.ctx)(EnvEventTopicMonsterStatus)(...)`), is pre-existing code not touched by this diff — the diff only adds a new `case monster2.SkillCategoryCarnivalBuf` arm that routes into the already-existing, already-reviewed `executeStatBuff` → `ApplyStatusEffect` path (`processor.go:949-950`, `:1039-1040`). No new emission call site was introduced |
| DOM-24 | Test packages reaching an emit path install `producertest` or inject a no-op producer | PASS | Package-wide `TestMain` at `services/atlas-monsters/atlas.com/monsters/monster/registry_test.go:27-28` calls `producertest.InstallNoop()`, covering all tests in the `monster` package including the three new test files, whose tests reach `executeStatBuff` → `ApplyStatusEffect` → `producer.ProviderImpl` transitively (2 hops, within the 3-hop trigger). Confirmed empirically: isolated re-run of only the new tests completes in 0.919s with no visible 10-retry/100ms→10s backoff stall |
| DOM-20 | Tests are table-driven | PASS | `carnival_skill_test.go:27-41` (`carnivalCase`/`carnivalCases` table), `picker_test.go` new `TestSkillSuppressingStatus` (table `tests := []struct{...}`) |
| DOM-10 | Test DB setup calls `database.RegisterTenantCallbacks` | N/A | none of the new/changed tests open a GORM DB directly — `grep -n "gorm.Open\|database.RegisterTenantCallbacks"` across `carnival_skill_test.go`, `carnival_value_test.go`, `seal_skill_test.go`, `picker_test.go` returns nothing |
| DOM-33 | Interface change updates every mock | N/A | `type Processor interface` (`processor.go:35`) is unchanged by this diff — no method added/removed/re-signed |
| FILE-01 | Processor interface/constructor/`ProcessorImpl` methods live in `processor.go` (or a `processor_<group>.go` split) | PASS | all new logic (`UseSkill`/`UseSkillGM` seal-gate replacement, `CARNIVAL_BUFF` dispatch-case additions, `testMobSkillLookup` seam in `UseSkillGM`) lands inside `services/atlas-monsters/atlas.com/monsters/monster/processor.go` |
| FILE-06 | No catch-all file bundling ≥2 responsibilities | PASS | `picker.go` (where `skillSuppressingStatus` was added) is a single-purpose decision/picker file (`Decision`, `RepickReason`, `pickNextSkill`) — not a Processor+RestModel+requests bundle; this diff's additions do not change that file's responsibility scope |

## Not evaluable from the diff

- DOM-11/DOM-16/DOM-04/DOM-05 (provider/administrator/rest.go structural checks): `provider.go`, `administrator.go`, and `rest.go` in the `monster` package were not touched by this diff; grading their compliance would require reading the full, unchanged files outside the diff's review surface — not attempted.
- DOM-07/08/09/12-15/17-19/32 (resource.go/handler-layer REST checks): `resource.go` was not part of the changed-file list and no hunk in it appears in the diff; these rules' trigger files are untouched, so evaluating them would require surveying a file outside scope.

## Summary

### Blocking (must fix)
- none

### Non-Blocking (should fix)
- none

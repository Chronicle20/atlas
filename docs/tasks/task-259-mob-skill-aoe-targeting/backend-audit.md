# Backend Audit — atlas-monsters (task-259-mob-skill-aoe-targeting)

- **Service Path:** services/atlas-monsters/atlas.com/monsters
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-08-21
- **Range:** 33e4cc7..HEAD
- **Build:** PASS
- **Tests:** all packages `ok` (no failures)
- **Overall:** NEEDS-WORK

## Build & Test Results

```
$ go build ./...      → exit 0, no output
$ go test ./... -count=1
ok  	atlas-monsters	2.173s
ok  	atlas-monsters/character/buff	0.149s
ok  	atlas-monsters/character/hidden	0.115s
ok  	atlas-monsters/character/position	0.111s
ok  	atlas-monsters/kafka/consumer/buff	0.364s
ok  	atlas-monsters/kafka/consumer/data	0.071s
ok  	atlas-monsters/kafka/consumer/monster	0.055s
ok  	atlas-monsters/map	0.131s
ok  	atlas-monsters/monster	22.578s
ok  	atlas-monsters/monster/consumable	0.105s
ok  	atlas-monsters/monster/drop	0.163s
ok  	atlas-monsters/monster/information	15.670s
ok  	atlas-monsters/world	0.193s
```

## Applicability

| Family | Fired? | Trigger observation |
|---|---|---|
| FILE (FILE-01..06) | Fired | Every changed package audited: `character/position` (new), `monster`, `monster/information`, `monster/mobskill` |
| RUNTIME (DOM-26) | Fired | Non-test Go files changed (`monster/disease_targets.go`, `monster/processor.go`) |
| DOM structure (DOM-01..05, 11, 16) | Fired | `character/position/rest.go` added (new package); `monster/rest.go`/`monster/model.go` present in a changed package but not themselves touched by this diff |
| REST (DOM-06..09, 12..15, 17..19, 32) | Fired | `character/position/processor.go`, `character/position/rest.go` added; `monster/processor.go` changed |
| Constants reuse (DOM-21) | Fired | `monster/disease_targets.go` classifies `skillId` against `monster2.SkillTypeSeduce` |
| Testing (DOM-10, 20, 24, 33) | Fired | Diff adds/changes `_test.go` files in `character/position` and `monster`; reaches emit paths via `p.emit` |
| Cache (DOM-29) | N/A | No changed package has `cache.go`; no new cached state introduced (`positionFn`/`inFieldFn` are per-call seams, not caches) |
| Messaging (DOM-30) | Fired | `monster/processor.go` calls `p.emit` → `producer.ProviderImpl` (via `emitter` field) |
| Multi-tenancy (DOM-31) | Fired | `character/position/rest.go` added; `monster/processor.go` reads tenant/context state |
| Migration hygiene (DOM-34, 35) | N/A | No symbols moved between service and `libs/atlas-*` |
| Deploy & topics (DOM-22, 23) | N/A | No `libs/atlas-*` module added; no Kafka topic env var added/renamed |
| Channel wire values (DOM-25) | N/A | Diff does not touch `services/atlas-channel` or `libs/atlas-packet`; no new client-interpreted byte introduced |
| Resilience (DOM-27, 28) | N/A | `db_surface=false` per `tools/task-facts.sh`; no `model.Decorator` changed; monster processor holds no DB-backed handlers |
| External clients (EXT-01..04) | Fired | `character/position/requests.go` calls `requests.GetRequest[RestModel]` against atlas-character |
| Scaffolding (SCAFFOLD-01..09) | N/A | No `services/atlas-<svc>/` directory added |
| Security (SEC-01..04) | N/A | atlas-monsters does not handle authentication, tokens, redirects, or secrets |
| Foundational: patterns-provider.md | N/A | No provider composed/defined in the diff |
| Foundational: patterns-functional.md | N/A | No curried constructor/decorator/combinator defined in the diff |

## Checklist Results

### character/position (support — REST client package, no `model.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor iface/ctor/methods in `processor.go` | PASS | `character/position/processor.go:11,18,24,31` — all in `processor.go` |
| FILE-02 | RestModel/Transform/JSON:API methods in `rest.go` | PASS | `character/position/rest.go:9,16,21,26` |
| FILE-03 | Cross-service request functions in `requests.go` | PASS | `character/position/requests.go:20-26` (`requestById`) |
| FILE-04 | entity struct/Migration/TableName in `entity.go` | N/A | Package has no `entity.go`; no entity symbols present |
| FILE-05 | Builder/domain Model/writes/readers/enums placed correctly | N/A | Package declares none of these symbols (pure read-only client) |
| FILE-06 | No `<pkgname>.go` bundling ≥2 responsibilities | PASS | `position.go` does not exist; responsibilities split across `processor.go`/`rest.go`/`requests.go` |
| DOM-06 | Processor ctor takes `logrus.FieldLogger` | PASS | `character/position/processor.go:24` |
| DOM-18 | RestModel implements `GetName/GetID/SetID` | PASS | `character/position/rest.go:16,21,26` |
| DOM-19 | Request models flat, no nested Data/Type/Attributes | N/A | Package declares no request/create/update models |
| DOM-31 | Tenant/trace travel via context only | PASS | `character/position/requests.go:18,21` — `baseURLProvider(ctx)`; `RestModel` (`rest.go:9-13`) carries no tenant field |
| EXT-01 | Target RestModel implements SetToOne/SetToMany ReferenceID(s) | PASS | `character/position/rest.go:38-39` (no-ops) |
| EXT-02 | httptest-backed test with representative fixture, asserts populated struct | PASS | `character/position/processor_test.go:21-32,42-64` — JSON:API fixture with `data.attributes.x/y`, asserts `gotX`/`gotY` |
| EXT-03 | Only genuine 404 maps to "not found"; other errors bubble | PASS | `character/position/processor.go:32-35` propagates `err` unchanged from `requestById`; `character/position/processor_test.go:66-80` asserts `requests.ErrNotFound` only for the 404 case |
| EXT-04 | URL composed via `requests.RootUrl`(-family), not hardcoded DNS | PASS | `character/position/requests.go:18` — `requests.RootUrlFor(ctx, "CHARACTERS")`, the context-aware variant of `RootUrl` defined in the same `libs/atlas-rest/requests` package (`libs/atlas-rest/requests/url.go:20,34`) |
| DOM-20 | Tests are table-driven (`tests := []struct{}` + `t.Run`) | **FAIL** | `character/position/processor_test.go:42-80` — two standalone `Test...` functions (`TestProcessor_GetPosition_ProjectsCoordinates`, `TestProcessor_GetPosition_PropagatesNotFound`), no `tests := []struct{...}` table, no `t.Run` |
| DOM-24 | Test package reaching emit path stubs the producer | N/A | Package makes no Kafka emit calls (read-only HTTP client) |
| DOM-33 | Mock updated alongside interface change | PASS | `character/position/mock/processor.go:7-18` implements the new `position.Processor` in the same diff |

### monster (domain — `model.go` present)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | ProcessorImpl methods in `processor.go` or `processor_<group>.go` | **FAIL (Important)** | `monster/disease_targets.go:67` (`func (p *ProcessorImpl) resolvePositions`) and `:104` (`func (p *ProcessorImpl) getDiseaseTargets`) — bare topic-named file, not `processor.go` and not named `processor_<group>.go` (cf. the package's own established split `monster/processor_catch.go`, which does follow the convention) |
| FILE-06 | No file bundling ≥2 responsibilities | PASS | `monster/disease_targets.go` carries only one FILE-0105 category (ProcessorImpl methods); the pure `selectDiseaseTargets` selector is not one of the enumerated categories |
| DOM-04 | `Transform(Model) (RestModel, error)` in `rest.go` | PASS | `monster/rest.go:64` — pre-existing, unchanged by this diff |
| DOM-06 | Processor ctor takes `logrus.FieldLogger` | PASS | `monster/processor.go:107` |
| DOM-21 | No redeclaration of a shared constant | PASS | `monster/disease_targets.go:43` reuses `monster2.SkillTypeSeduce`, declared once in `libs/atlas-constants/monster/skill.go:40` |
| DOM-26 | Every goroutine via `routine.Go(l, ctx, fn)` | PASS | `monster/disease_targets.go:73` — `routine.Go(p.l, p.ctx, func(_ context.Context) {...})`, matching signature `libs/atlas-routine/routine.go:15`; `tools/goroutine-guard.sh` exit 0 |
| DOM-30 | DB-writing operation emits via `AndEmit`+`message.Buffer`, not a bare producer call | N/A | `monster` package holds no DB-backed write path (`grep` for `gorm`/`db.Create`/`database.Connect` in `monster/*.go` — zero matches); state lives in the in-memory `registry.go`. Documented exception in `patterns-kafka.md` §"Operations over non-DB state" applies — direct `p.emit` (wrapping `producer.ProviderImpl`) at `monster/processor.go:1245,1274,1285` is not a finding |
| DOM-31 | Tenant/trace travel via context only | PASS | `monster/processor.go:110` (`t: tenant.MustFromContext(ctx)`); `disease_targets.go`'s `resolvePositions`/`getDiseaseTargets` take no tenant parameter |
| DOM-20 | Tests are table-driven | **FAIL** (×2 files) | `monster/disease_targets_shell_test.go` — 8 standalone `Test...` functions (`:54,71,88,107,126,145,170,190`), no `tests := []struct{...}` table; `monster/disease_callers_test.go` — 4 standalone `Test...` functions (`:16,49,82,120`), no table. (`monster/disease_targets_test.go` IS table-driven — `:14-128` `tests := []struct{...}` + `t.Run` at `:131` — PASS for that file.) |
| DOM-24 | Test package reaching emit path stubs the producer | PASS | `monster/disease_callers_test.go:17,50,89,127` use `newRecordingProcessor` (`monster/processor_test.go:39`), which injects a per-test no-op `emitter` into `p.emit` — the documented per-test-injection form of DOM-24 |
| DOM-33 | Mock updated alongside interface change | N/A | `monster.Processor` (exported interface, `monster/processor.go:32-73`) unchanged by this diff — `executeDebuff`/`executeBanish`/`executeDispel`/`getDiseaseTargets`/`resolvePositions` are all unexported and not interface members |

### monster/information, monster/mobskill (domain — `model.go` present; only `builder.go` touched)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | `builder.go` has `NewBuilder()`, fluent setters, validating `Build()` | PASS | `monster/mobskill/builder.go:72-75` (`SetCount`), `:95-111` (`Build()`); `monster/information/builder.go:61-64` (`SetBanish`), `:67-85` (`Build()`) — pure additive setters on the pre-existing builder, fluent chain preserved |
| FILE-05 | Builder in `builder.go` | PASS | Both new setters live in the package's existing `builder.go`, no new file added |

## Not evaluable from the diff

- DOM-01/02/03/11/16 for `monster` package's `model.go`/`entity.go`(absent)/`provider.go`(absent)/`administrator.go`(absent): these files were not touched by this diff and reviewing their compliance would require surveying pre-existing package structure beyond the changed files. `monster` has no `entity.go`, `provider.go`, or `administrator.go` at all (`ls` confirms absence), so DOM-02/03/11/16 do not have a file to evaluate against within this diff's surface.
- DOM-05 (`TransformSlice` used by list handlers, no inline loops in `resource.go`) for `monster` package: `monster/resource.go` was not changed by this diff; would require reading the unchanged `resource.go` to determine whether it already used `TransformSlice` before this change.

## Summary

### Blocking (must fix)
- FILE-01: `monster/disease_targets.go:67,104` — `resolvePositions` and `getDiseaseTargets` are `*ProcessorImpl` methods living in a bare topic-named file instead of `processor.go` or a `processor_<group>.go` split (e.g. `processor_disease.go`), diverging from the package's own established convention (`monster/processor_catch.go`).
- DOM-20: `character/position/processor_test.go:42-80` — two new tests are standalone functions, not the required `tests := []struct{...}` + `t.Run` table-driven pattern.
- DOM-20: `monster/disease_targets_shell_test.go:54-226` — 8 new tests are standalone functions, not table-driven.
- DOM-20: `monster/disease_callers_test.go:16-149` — 4 new tests are standalone functions, not table-driven.

### Non-Blocking (should fix)
- none

Non-blocking WARN items: none identified. All findings above are FAIL-level per their rule's explicit pass criteria.

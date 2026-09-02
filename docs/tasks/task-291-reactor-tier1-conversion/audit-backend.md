# Backend Audit — task-291-reactor-tier1-conversion

- **Service Path:** `services/atlas-reactor-actions` (script package) + `tools/reactor-seed-gen`, `tools/reactor-seed-lint` (new build-time tool modules)
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-09-02
- **Build:** PASS
- **Tests:** all packages pass, 0 failed
- **Overall:** NEEDS-WORK

## Build & Test Results

```
cd tools/reactor-seed-gen && GOWORK=off go build ./...      -> clean, no output
cd tools/reactor-seed-lint && GOWORK=off go build ./...     -> clean, no output
cd services/atlas-reactor-actions/atlas.com/reactor && go build ./... -> clean, no output

cd tools/reactor-seed-gen && GOWORK=off go test ./... -count=1
  ok  github.com/Chronicle20/atlas/tools/reactor-seed-gen  0.024s

cd tools/reactor-seed-lint && GOWORK=off go test ./... -count=1
  ok  github.com/Chronicle20/atlas/tools/reactor-seed-lint  12.787s

cd services/atlas-reactor-actions/atlas.com/reactor && go test ./... -count=1
  ok  atlas-reactor-actions                0.025s
  ok  atlas-reactor-actions/script          0.065s
  (remaining packages: no test files)
```

`gofmt -l` on all changed files: empty (no formatting violations).
`go vet ./...` in all three modules: clean.
`tools/goroutine-guard.sh`: exit 0 (`93 module(s), 8 parallel`, no bare `go` statement findings).

Corpus consistency spot-checks (both were run, not sampled — they cover the
full 1,749-file corpus in one pass):
- `go run ./tools/reactor-seed-gen -check` against the committed
  `deploy/seed` tree: `reactor-seed-gen: 159 reactors x 11 directories = 1749
  files match` — the generator reproduces every committed byte exactly.
- `go run ./tools/reactor-seed-lint deploy/seed
  services/atlas-reactor-actions/docs/reactor_script_schema.json`: exit 0,
  no output — schema conformance, envelope well-formedness, non-empty
  description and cross-tenant byte identity all hold across the full corpus.
- `grep -rl '"minMeso"\|"maxMeso"\|"mesoRange"' deploy/seed/`: 0 matches —
  confirms no seed file still emits the legacy keys `executor.go` just
  stopped reading, so the fallback removal (below) is not a live regression
  against the data on disk.

## Applicability

| Family | Fired? | Trigger observation |
|---|---|---|
| DOM structure (DOM-01..05,11,16) | N/A | `script` package's `model.go`/`entity.go`/`rest.go`/`provider.go` are not among the changed files (`executor.go`, `subdomain.go` only); each rule's own file-existence trigger did not fire on the diff |
| FILE placement (FILE-01..06) | Fired | Every changed package (`script`, `tools/reactor-seed-gen`, `tools/reactor-seed-lint`) is in scope unconditionally |
| SUB sub-domain (SUB-01..04) | N/A | `script` package has `model.go` — SUB's own trigger ("resource.go but no model.go") never fires for this package |
| REST (DOM-06..09,12..15,17..19,32) | N/A | `script` has `resource.go`/`rest.go`/`processor.go` but none were touched by this diff; each rule's file-existence trigger did not fire on the changed files |
| Constants reuse (DOM-21) | Fired (checked, clean) | `doc.go` declares `scriptDoc`/`ruleDoc`/`condDoc`/`opDoc`; grepped `libs/atlas-constants/` for an equivalent classification — none found |
| Testing (DOM-10,20,24,33) | Fired | Diff touches four `_test.go` files |
| Cache (DOM-29) | N/A | No `cache.go`, no cached state in the diff |
| Messaging (DOM-30) | N/A | No `producer.go`; `executor.go`'s `sagaP.Create` path is pre-existing and untouched by this diff |
| Multi-tenancy (DOM-31) | N/A | `rest.go` untouched; the touched `subdomain.go` hunk (touch-rule loop) and `executor.go` hunk (meso param parsing) neither read nor pass tenant/trace state |
| Migration hygiene (DOM-34,35) | N/A | Diff does not move symbols between a service and `libs/atlas-*` |
| Deploy & topics (DOM-22,23) | N/A | `go.work` adds two `tools/` modules, not a `libs/atlas-*` module; no topic env var added or renamed |
| Runtime safety (DOM-26) | Fired (checked, clean) | Non-test Go files changed; `tools/goroutine-guard.sh` exit 0 |
| Channel wire values (DOM-25) | N/A | Diff touches neither `services/atlas-channel` nor `libs/atlas-packet`; no client-interpreted byte in any emitted event |
| Resilience (DOM-27,28) | N/A | No DB-backed handler or `model.Decorator`/enrichment path in the diff |
| External clients (EXT-01..04) | Not evaluable from diff (see below) | `script` package calls `requests.RootUrlFor`/`requests.GetRequest[T]` (`executor.go:494-499`, `evaluator.go:176-182`), but both call sites are pre-existing and untouched by this diff's hunks |
| Scaffolding (SCAFFOLD-01..09) | N/A | No new `services/atlas-<svc>/` directory, no new channel writer/handler, no `routes.conf` change |
| Security (SEC-01..04) | N/A | Neither module handles auth, tokens, redirects, or secrets |
| Foundational: patterns-provider.md | N/A | Diff defines no providers |
| Foundational: patterns-functional.md | N/A | Diff defines no curried constructors/decorators/combinators |

## Checklist Results

### `script` (domain package — has `model.go`; only `executor.go` and `subdomain.go` are in the changed-file surface)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01..05 | Domain file responsibilities | N/A | Neither `executor.go` nor `subdomain.go` is `processor.go`/`rest.go`/`requests.go`/`entity.go`/`builder.go`/`model.go`/`administrator.go`/`provider.go`/`state.go`; `executor.go` is operation-execution logic and `subdomain.go` is the `seeder.Subdomain` adapter — neither responsibility class applies to either file |
| FILE-06 | No package-named catch-all carrying ≥2 responsibilities | PASS | `services/atlas-reactor-actions/atlas.com/reactor/script/executor.go` (operation execution only) and `subdomain.go` (seeder adapter only) each carry a single, distinct responsibility — neither collapses two of the FILE-01..05 categories |
| DOM-20 | Table-driven tests | FAIL | `services/atlas-reactor-actions/atlas.com/reactor/script/executor_test.go:185` (`TestExecuteSprayItems_SetsDropType`) and `:241` (`TestExecuteSprayItems_NoParams`) are two separate, non-table Test functions that exercise the same entry point (`ExecuteOperation` with `spray_items`) varying only the input params map and the assertions on the resulting payload — the pattern the guideline requires be consolidated via `tests := []struct{...}` + `t.Run`, as the adjacent `TestExecuteDropItems_MesoParams` in the same file (lines 54-183) already correctly does |
| DOM-20 | Table-driven tests | PASS | `services/atlas-reactor-actions/atlas.com/reactor/script/subdomain_test.go:23-92` (`TestReactorSubdomainBuildTouchRules`) uses `tests := []struct{...}` + `t.Run` |
| DOM-10 | Test DB setup calls `RegisterTenantCallbacks` | N/A | Neither `executor_test.go` nor `subdomain_test.go` opens a GORM DB directly — `subdomain_test.go` calls `ReactorSubdomain{}.Decode`/`Build` in-process; no `gorm.Open` in either file |
| DOM-24 | Emit-path producer stub | N/A | `executor_test.go` injects `fakeSagaProcessor` (`executor_test.go:19-30`) in place of `reactorsaga.Processor`; no test reaches `AndEmit`/`message.Emit`/`producer.Produce`, directly or transitively |
| DOM-33 | Interface change updates mocks | N/A | No `Processor`/`Provider`/`Administrator` interface method was added, removed, or re-signed by this diff — `ReactorSubdomain.Build`'s signature is unchanged, only its body grew a loop |

### `tools/reactor-seed-gen` (support package — build-time tool, no `model.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01..06 | File responsibilities | N/A | No `processor.go`, `rest.go`, `requests.go`, `entity.go`, `builder.go`/`model.go`/`administrator.go`/`provider.go`/`state.go` exists or is implied anywhere in the module; `main.go`, `parse.go`, `convert.go`, `describe.go`, `emit.go`, `doc.go` each own one pipeline stage per `doc.go:1-5`'s own description, and none carries ≥2 of the FILE-01..05 domain responsibilities |
| DOM-20 | Table-driven tests | PASS | `convert_test.go:9` (`TestConvertBody`) and `:320` (`TestConvertBody_NegativeCases`), `parse_test.go:9` (`TestParseInventory`), `describe_test.go:10` (`TestDescribe`) all use `tests`/`cases := []struct{...}` + `t.Run` |
| DOM-20 | Table-driven tests | PASS (single-scenario, not an enumerable-case violation) | `emit_test.go`'s six Test functions (`TestEmit_GoldenBytes:11`, `TestEmit_BareDropOmitsParams:50`, `TestEmit_EmptyRulesAreArraysNotNull:78`, `TestEmit_ConditionOmitsEmptyStep:102`, `TestEmit_TrailingNewline:140`, `TestFanOut_WritesElevenIdenticalCopies:162`) and `describe_test.go`'s `TestDescribe_MissingOverrideAborts:88`, `TestDescribe_NoBoilerplateLeaks:98`, `TestDescribe_OverridesAreGrounded:174` each assert a distinct structural invariant over a single fixture, not a repeated scenario varying only its input — there is no duplicated near-identical body across these functions to consolidate |
| DOM-10 | Test DB setup calls `RegisterTenantCallbacks` | N/A | No test in this module opens a GORM DB |
| DOM-24 | Emit-path producer stub | N/A | Module has no Kafka dependency at all |
| DOM-33 | Interface change updates mocks | N/A | No `Processor`/`Provider`/`Administrator` interface exists in this module |

### `tools/reactor-seed-lint` (support package — build-time tool, no `model.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01..06 | File responsibilities | N/A | No domain-construct file exists; `main.go` (walk/validate orchestration), `schema.go` (schema compile/validate), `identity.go` (cross-tenant digest compare) each own one concern |
| DOM-20 | Table-driven tests | FAIL | `main_test.go:22-125` defines ten separate Test functions (`TestLint_GoodCorpusExitsZero`, `TestLint_LegacyKeysExitNonZero`, `TestLint_TypeMismatchExitsNonZero`, `TestLint_IDMismatchExitsNonZero`, `TestLint_MissingRequiredExitsNonZero`, `TestLint_MissingDescriptionExitsNonZero`, `TestLint_DivergentCopiesExitNonZero`, `TestLint_MissingCopyExitsNonZero`, `TestLint_EmptyRootExitsNonZero`, `TestLint_NonexistentRootExitsNonZero`) that all perform the identical sequence — `buildLint(t)`, `exec.Command(exe, <fixture dir>, schemaPath)`, assert exit code, optionally assert an output substring — varying only the fixture path and the expected substring. This is exactly the repeated-scenario shape `tests := []struct{...}` + `t.Run` exists to collapse, and none of the ten functions uses it |
| DOM-10 | Test DB setup calls `RegisterTenantCallbacks` | N/A | No test opens a GORM DB |
| DOM-24 | Emit-path producer stub | N/A | Module has no Kafka dependency |
| DOM-33 | Interface change updates mocks | N/A | No `Processor`/`Provider`/`Administrator` interface exists in this module |

## Security Review

Not applicable — SEC-* trigger did not fire (neither the `script` package changes nor either tool module handle authentication, authorization, tokens, redirects, or secrets).

## Not evaluable from the diff

- EXT-01..04: `script` package calls `requests.RootUrlFor`/`requests.GetRequest[T]` against the party-quests service (`executor.go:493-499`, and the near-duplicate in `evaluator.go:176-182`), which would trigger the External clients family. Both call sites are pre-existing and outside this diff's changed hunks (the diff only touches `executor.go` lines 96-133, the `executeDropItems` meso-parsing block). Verifying EXT-01..04 against them would mean auditing code this branch did not write or modify — out of the review surface. Would need: confirmation this branch is not the origin of that call (already established via `git blame`, showing commits from Feb/Aug 2026, well before this branch), and, if a full audit were wanted anyway, a read of `pqInstanceRestModel`'s JSON:API compliance and an httptest-backed fixture check in `evaluator_test.go`, neither of which changed here.

## Summary

### Blocking (must fix)
- DOM-20: `services/atlas-reactor-actions/atlas.com/reactor/script/executor_test.go:185,241` — `TestExecuteSprayItems_SetsDropType` and `TestExecuteSprayItems_NoParams` duplicate the same exercise-and-assert shape across two non-table Test functions; consolidate into a table alongside (or as an extra case of) the existing `TestExecuteDropItems_MesoParams` table.
- DOM-20: `tools/reactor-seed-lint/main_test.go:22-125` — ten Test functions repeat the identical build/exec/assert-exit-code shape varying only fixture path and expected substring; consolidate into `tests := []struct{...}` + `t.Run`.

### Non-Blocking (should fix)
- None identified.

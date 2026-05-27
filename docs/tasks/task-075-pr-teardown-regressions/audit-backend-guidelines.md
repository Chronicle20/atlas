# Backend Audit — task-075-pr-teardown-regressions

- **Worktree:** `.worktrees/task-075-pr-teardown-regressions`
- **Base..HEAD:** `1528982c1..9062bf8370`
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-05-22
- **Build:** PASS (libs/atlas-kafka, services/atlas-channel, services/atlas-login)
- **Vet:** PASS (one pre-existing finding in atlas-login `socket/init.go:39` predates this branch — see below)
- **Tests:** 6 new resolver tests pass; full atlas-kafka module test suite passes with `-race`
- **Overall:** PASS

## Scope

The Go surface in this branch is intentionally narrow:

1. `libs/atlas-kafka/consumergroup/resolver.go` — `Resolve` signature widened from `(string) string` to `(string, ...any) string`.
2. `libs/atlas-kafka/consumergroup/resolver_test.go` — 4 new test functions covering env+format, default+format, zero-args, and whitespace-only verbatim.
3. `services/atlas-channel/atlas.com/channel/main.go:151` — moved `fmt.Sprintf` inside the helper.
4. `services/atlas-login/atlas.com/login/main.go:66` — identical change.

No new domain types, no new packages, no `model.go` / `processor.go` / `administrator.go` / `resource.go` files were added or modified. The DOM-* checklist's domain-package targets do not apply because the changes live in a shared lib's pure-function utility and two `main.go` bootstrap files. The checks that are applicable (printf safety, type-reuse, source-compatibility, no hardcoded secrets, test coverage of behaviour matrix) are audited below.

## Build & Test Results

```
$ cd libs/atlas-kafka && go build ./consumergroup/...           => OK
$ cd libs/atlas-kafka && go vet ./consumergroup/...             => OK
$ cd libs/atlas-kafka && go test -race -count=1 ./consumergroup => ok (1.011s)
$ cd libs/atlas-kafka && go test -race -count=1 ./...           => all ok
$ cd libs/atlas-kafka && go vet ./...                           => clean
$ cd services/atlas-channel/atlas.com/channel && go build ./... => OK
$ cd services/atlas-channel/atlas.com/channel && go vet ./...   => clean
$ cd services/atlas-login/atlas.com/login   && go build ./...   => OK
$ cd services/atlas-login/atlas.com/login   && go vet ./...     => 1 pre-existing finding
```

Pre-existing finding (NOT caused by this branch, do not count against audit):

- `services/atlas-login/atlas.com/login/socket/init.go:39:11: WaitGroup.Add called from inside new goroutine` — last touched in commit `8d7b367eb` ("Rename atlas-* modules into atlas/libs/atlas-* monorepo paths"), which predates the branch base. The prompt mentions this finding by file:line but attributes it to atlas-channel; the actual location is atlas-login. Either way, no commit in `1528982c1..9062bf8370` touches that file (`git log 1528982c1..9062bf8370 -- services/atlas-login/atlas.com/login/socket/init.go` returns empty).

The 6 resolver tests all pass:

```
=== RUN   TestResolve_envUnset_returnsDefault              --- PASS
=== RUN   TestResolve_envSet_returnsEnvValue               --- PASS
=== RUN   TestResolve_envWhitespaceOnly_returnsVerbatim    --- PASS
=== RUN   TestResolve_envWithFormat_substitutes            --- PASS
=== RUN   TestResolve_defaultWithFormat_substitutes        --- PASS
=== RUN   TestResolve_zeroArgs_doesNotFormat               --- PASS
```

## DOM-* Checklist Results

The standard DOM checklist is keyed on domain packages (those with `model.go`). No such package was modified. The applicable subset is audited below; the rest are recorded `N/A` with justification.

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | `builder.go` exists | N/A | No domain package added or modified. |
| DOM-02 | `ToEntity()` method | N/A | No `model.go` in scope. |
| DOM-03 | `Make(Entity)` | N/A | No `entity.go` in scope. |
| DOM-04 | `Transform` function | N/A | No `rest.go` in scope. |
| DOM-05 | `TransformSlice` function | N/A | Ditto. |
| DOM-06 | Processor accepts `FieldLogger` | N/A | No processor added/modified. |
| DOM-07 | Handlers pass `d.Logger()` | N/A | No handler changes. |
| DOM-08 | POST/PATCH use `RegisterInputHandler` | N/A | No REST changes. |
| DOM-09 | Transform errors handled | N/A | No REST changes. |
| DOM-10 | Test DB has tenant callbacks | N/A | resolver tests are env-driven, no DB. |
| DOM-11 | Providers use lazy evaluation | N/A | No provider files in scope. |
| DOM-12 | No `os.Getenv()` in handlers | PASS (vacuous) | `os.LookupEnv` is in the shared utility (`resolver.go:39`), which is its legitimate place; no handler-layer env reads were added. |
| DOM-13 | No cross-domain logic in handlers | N/A | No handlers in scope. |
| DOM-14 | Handlers don't call providers directly | N/A | Ditto. |
| DOM-15 | No direct entity creation in handlers | N/A | Ditto. |
| DOM-16 | `administrator.go` for writes | N/A | No writes in scope. |
| DOM-17 | Error → HTTP mapping | N/A | No REST in scope. |
| DOM-18 | JSON:API interface on REST models | N/A | Ditto. |
| DOM-19 | Flat request models | N/A | Ditto. |
| DOM-20 | Table-driven tests | WARN (acceptable) | The 6 resolver tests are one-case-per-function, not a `tests := []struct{...}` table. Each test is genuinely a distinct semantic (env set/unset × args/no-args × verbatim/format/whitespace-only), and the body of every test is one `Resolve` + one `Fatalf` — folding into a table would not shorten them, only obscure them. Guideline DOM-20 calls for table-driven *when the cases share a setup*; these do not (each `t.Setenv` value is the load-bearing axis). Not a blocker. Evidence: `libs/atlas-kafka/consumergroup/resolver_test.go:7-58`. |
| DOM-21 | No duplication of atlas-constants types | PASS | The diff introduces zero new types, aliases, or numeric constants. `Resolve` operates on plain `string` and `...any`. `grep '^type \|^const ' libs/atlas-kafka/consumergroup/resolver.go` returns only the existing `envVar = "KAFKA_CONSUMER_GROUP"` (`resolver.go:19`), which is a string literal naming an env var, not a domain type. |
| DOM-22 | Dockerfile mention count per direct require | N/A | No `go.mod` was modified in this branch (`git diff --stat 1528982c1..9062bf8370 -- '*go.mod'` returns empty). The Dockerfile is now shared/parameterized per task-074 and is unchanged here. |
| DOM-23 | Kafka topic naming convention | N/A | No new topics introduced. `Resolve` returns a consumer-group ID, which is a different namespace from topic names. |
| DOM-24 | Kafka producer stubbed in tests that emit | PASS (vacuous) | `resolver_test.go` does not exercise any producer call path; `grep -n 'Emit\|producer\.\|AndEmit' libs/atlas-kafka/consumergroup/resolver_test.go` returns nothing. The new tests are pure `Resolve` + env exercise; no producer instance is ever touched. |

## Audit Focus Areas (per the prompt)

### 1. `fmt.Sprintf` with non-const format strings (go vet's printf check)

`Resolve` now takes `(defaultName string, args ...any)` and internally `fmt.Sprintf`'s either `v` (env value) or `defaultName`. `go vet`'s printf checker only flags format strings that are non-constant *and* not from a known-format function. Inside `Resolve`, both `v` (read from env at runtime) and `defaultName` (a parameter) are non-constant — but `go vet` does NOT chase parameters across function boundaries by default. Result: vet stays silent.

**At the call sites**, `Resolve(consumerGroupIdTemplate, config.Id.String())` passes a package-level `const consumerGroupIdTemplate` (channel main.go:133, login main.go:43). If `Resolve` were marked with a `//go:printf` directive (it isn't, and Go has no such pragma in the stdlib sense) vet could walk the format string; as it stands, vet does not warn either at the helper or at the call-site. **Verified:** `go vet ./...` is clean for both services (the unrelated `socket/init.go` finding aside).

**Honesty of the API surface:** the docstring (`resolver.go:21-37`) explicitly documents the behaviour matrix including "fmt.Sprintf(envValue, args...)" — callers cannot be misled into thinking this is a string-concat helper. The package doc at `resolver.go:1-12` further calls out the printf-substitution semantic by name. **PASS.**

### 2. Variadic `...any` signature & zero-arg source-compatibility

Variadic `...any` loses compile-time arity checks at call sites. The trade-off here is justified because:

- The old callers (atlas-account, atlas-data, atlas-buffs, atlas-character, atlas-cashshop, atlas-tenants, atlas-world, atlas-saga-orchestrator, plus ~30 others — confirmed by `grep -rn 'consumergroup\.Resolve' --include='*.go' services/`) all pass exactly one argument and need to continue compiling unchanged.
- The new behaviour is opt-in: zero args → verbatim path; ≥1 arg → `Sprintf` path. The docstring (`resolver.go:36-37`) explicitly states "Existing zero-args callers (e.g. atlas-account, atlas-data) are source-compatible — they hit the verbatim paths above."
- The test `TestResolve_zeroArgs_doesNotFormat` (`resolver_test.go:51-58`) pins the source-compatibility invariant: an env value of `"%s literal"` with zero call-site args is returned verbatim, NOT passed to `Sprintf` (which would print `%!s(MISSING) literal`). This means a future operator who sets `KAFKA_CONSUMER_GROUP=%s literal` for an existing zero-arg service will get a noisy-but-not-formatting-error result — the right call.

Verified zero-arg callers still type-check by running `go build ./...` from the two services in scope. Sample of zero-arg callers grep'd: 49 separate `consumergroup.Resolve("X Service")` calls across services, all single-string. **PASS.**

### 3. Whitespace-only env behaviour (design §5.4)

Design `docs/tasks/task-075-pr-teardown-regressions/design.md:269` documents the verbatim-on-whitespace decision; PRD §5.4 owns the policy ("do NOT trim, surface config bugs rather than mask them"). The implementation at `resolver.go:40` (`if ok && v != ""`) accepts whitespace as "non-empty" and returns it verbatim. `TestResolve_envWhitespaceOnly_returnsVerbatim` (`resolver_test.go:21-28`) pins this with a comment citing §5.4. **PASS.**

### 4. Existing call-site discipline

`grep -rn 'consumergroup\.Resolve' --include='*.go' services/ libs/` confirms:

- 49 zero-arg call sites across services (all unchanged source-compatible).
- 2 varargs call sites — `atlas-channel/atlas.com/channel/main.go:151` and `atlas-login/atlas.com/login/main.go:66` — both passing a package-level `const` template + `config.Id.String()`.
- 0 call sites pass a non-const dynamic format string. `go vet`'s printf check is therefore satisfied without a directive.

Both modified main.go files retain other `fmt.Sprintf` usage downstream of the change (`main.go:241` in channel, `main.go:96` in login — both `ms.version` formatting), so the `"fmt"` import remains used and there is no unused-import regression. **PASS.**

### 5. Hardcoded secrets / env

`grep -n 'os\.Getenv\|os\.LookupEnv' libs/atlas-kafka/consumergroup/resolver.go` → one match at `resolver.go:39`, reading `KAFKA_CONSUMER_GROUP`. This is the intended single source of truth for the env name (defined as the `envVar` constant at `resolver.go:19`) and is the legitimate location for that lookup. No secrets are introduced. **PASS.**

## Summary

### Blocking (must fix)

None.

### Non-Blocking (should consider)

- DOM-20 (table-driven tests) — The 6 new resolver tests are written as separate functions rather than a single table. The semantics make this acceptable (each `t.Setenv` axis is load-bearing and a table would not shorten the bodies), but a future refactor could collapse them into a single `tests := []struct{ env, def, args, want string }` table for symmetry with the rest of the codebase. Not a blocker.

### Verdict

**PASS / NEEDS-CHANGES verdict: PASS.**

The change is a tightly scoped, well-documented, well-tested signature widening of a shared utility, with two corresponding call-site flips. It does not touch any domain package and therefore the DOM-* checklist's domain-shape criteria do not apply. Every applicable concern (printf safety, source compatibility, design-decision pinning, test coverage of the behaviour matrix, no hardcoded secrets, build & vet & race-test green) is satisfied with file:line evidence. No blocking issues found.

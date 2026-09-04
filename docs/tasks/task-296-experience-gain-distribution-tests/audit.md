# Backend Audit — atlas-channel (task-296)

- **Service Path:** services/atlas-channel/atlas.com/channel
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-09-04
- **Commit range:** 25ba8d1cb..HEAD (c4fc10ab7)
- **Build:** PASS
- **Tests:** 177 packages passed (0 failed) module-wide; `kafka/consumer/character` — 5 top-level tests / all subtests PASS, 0 FAIL
- **Overall:** PASS

## Build & Test Results

```
$ go build ./...          # exit 0, no output
$ go test ./... -count=1  # all 177 exercised packages report "ok", 0 "FAIL"
$ go test ./kafka/consumer/character/... -count=1 -v
--- PASS: TestClearAranComboOnMapChange_ClearsState (0.00s)
--- PASS: TestClearAranComboOnMapChange_UnknownCharacter_NoOp (0.00s)
--- PASS: TestSnapshotHandlers (0.00s)             # 6 sub-tests, all PASS
--- PASS: TestBuildIncreaseExperienceConfig (0.00s) # 18 sub-tests, all PASS
--- PASS: TestExperienceDistributionTypeExhaustiveness (0.00s)
PASS
ok  	atlas-channel/kafka/consumer/character	...
```

## Behavioral-equivalence check (switch conversion)

`bdfb6c239` converts the `if d.ExperienceType == X { ... } else if ... ` chain
in `announceExperienceGain` (formerly inline, now extracted as
`buildIncreaseExperienceConfig`) into a `switch d.ExperienceType { case X: ... }`.

Diffed the pre-refactor body
(`git show 25ba8d1cb:services/atlas-channel/atlas.com/channel/kafka/consumer/character/consumer.go`,
lines 366–405) against the current
`services/atlas-channel/atlas.com/channel/kafka/consumer/character/consumer.go:369-423`
case-by-case: every branch's condition, field assignment, and body ordering is
identical; the chain compared only `d.ExperienceType == <const>` (mutually
exclusive, single-variable equality), which is semantics-preserving under a
`switch` on the same variable. No default/fallthrough was introduced. The
unmapped-type behavior (silently drop, e.g. `"DEATH"`) is unchanged — pinned by
`consumer_test.go:377-384` (`UnknownType_DeathIgnored`).

**No behavioral divergence found.**

## Applicability

| Family | Fired? | Trigger observation |
|---|---|---|
| DOM structure (DOM-01..05,11,16) | No | Neither `kafka/consumer/character` nor `kafka/message/character` has `model.go`, `entity.go`, `rest.go`, or `provider.go` |
| FILE placement (FILE-01..06) | Yes (mandatory) | Runs on every changed package; see per-package tables below |
| SUB (SUB-01..04) | No | Neither changed package has `resource.go` |
| REST (DOM-06..09,12..15,17..19,32) | No | No `resource.go`/`rest.go`/`processor.go`, no route registration in the diff |
| Constants reuse (DOM-21) | No | Diff adds doc comments to existing consts and a `[]string` registry var (`AllExperienceDistributionTypes`) — no new `type`, named `const` block, or numeric-literal classification |
| Testing (DOM-10,20,24,33) | Yes | `consumer_test.go` is new/changed |
| Cache (DOM-29) | No | No `cache.go`; no processor/struct holds cached state in the diff |
| Messaging (DOM-30) | No | `grep -n "AndEmit\|message.Emit\|producer.Produce\|producer.ProviderImpl"` across the 3 changed files returns no match |
| Multi-tenancy (DOM-31) | Yes | Test code passes tenant state (`consumer_test.go:64` `tenant.WithContext`) |
| Migration hygiene (DOM-34,35) | No | Diff moves no symbols between service and `libs/atlas-*` |
| Deploy & topics (DOM-22,23) | No | No `libs/atlas-*` module added, no Kafka topic env var added/renamed |
| Runtime safety (DOM-26) | Yes (mandatory) | Non-test Go files changed (`consumer.go`, `kafka.go`) |
| Channel wire values (DOM-25) | Yes (family) | Diff is entirely inside `services/atlas-channel` |
| Resilience (DOM-27,28) | No | No DB-backed handler, no `model.Decorator`/enrichment path touched |
| External clients (EXT-01..04) | No | No `requests.*Request[T]` call added |
| Scaffolding (SCAFFOLD-01..09) | No | No new service dir, no new Writer/Handler registration, `routes.conf` untouched |
| Security (SEC-01..04) | No | No auth/token/redirect/secret handling touched |
| patterns-provider.md (foundational) | No | No provider defined/composed in the diff |
| patterns-functional.md (foundational) | Yes | `buildIncreaseExperienceConfig` and the pre-existing curried `announceExperienceGain` are functional-style constructs the diff touches |

## Checklist Results

### kafka/consumer/character (support package)

Files present: `consumer.go`, `consumer_test.go`, `aran_combo_hook_test.go`, `channel_change.go`. No `model.go`, `entity.go`, `rest.go`, `provider.go`, `processor.go`, `resource.go`.

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor interface/constructor/methods in `processor.go` | N/A | `grep -n "type Processor interface\|type ProcessorImpl\|func NewProcessor("` in the package returns no match — nothing to place |
| FILE-02 | RestModel/Transform/Extract/JSON:API methods in `rest.go` | N/A | No `RestModel`/`Transform`/`Extract` symbol in the package |
| FILE-03 | Cross-service request functions in `requests.go` | N/A | No `requests.RootUrl(`/`requests.GetRequest[`/`requests.PostRequest[` in the package |
| FILE-04 | Entity/`Migration`/`TableName` in `entity.go` | N/A | No `entity struct`/`Migration(`/`TableName()` in the package |
| FILE-05 | Builder/Model/writes/readers placed per file table | N/A | No `Builder`, domain `Model`, `Create*`/`Update*`/`Delete*` write, or `database.Query`/`SliceQuery` reader in the package |
| FILE-06 | No `<pkgname>.go` bundling ≥2 responsibilities | N/A | No responsibility from FILE-01..05 is present anywhere in the package, so there is nothing to bundle; `consumer.go` holds only Kafka `message.Handler` functions, which are not one of the five graded responsibilities |
| DOM-10 | Test DB setup calls `database.RegisterTenantCallbacks` | N/A | `consumer_test.go` opens no GORM DB — `newTestServer`/`seedSnapshotCore` use `server.NewProcessor` and an in-memory `snapshot.GetRegistry()`, no `gorm.Open` |
| DOM-20 | Tests are table-driven | PASS | `consumer_test.go:56` `TestSnapshotHandlers` uses `tests := []struct{...}` + `t.Run`; `consumer_test.go:215-396` `distributionMappingCases` + `TestBuildIncreaseExperienceConfig` uses the same table-driven shape |
| DOM-24 | Emit-reaching test packages install `producertest`/no-op producer | N/A | Direct check: no `AndEmit`/`message.Emit`/`producer.Produce` in `consumer_test.go`. Transitive check: every handler exercised by the tests (`handleSnapshotStatChanged` `consumer.go:596`, `handleSnapshotLevelChanged` `consumer.go:609`, `handleSnapshotExperienceChanged` `consumer.go:622`, `handleSnapshotMapChanged` `consumer.go:640`, and `buildIncreaseExperienceConfig` `consumer.go:369`) takes its `writer.Producer` parameter as `_` (discarded) or no producer parameter at all, and its body only touches `snapshot.GetRegistry()` or pure struct construction — no path reaches `producer.ProviderImpl` or `message.Emit` |
| DOM-33 | Interface change updates every mock | N/A | Diff adds/removes/re-signs no method on a `Processor`/`Provider`/`Administrator` interface |
| DOM-26 | Every goroutine via `routine.Go` | PASS | `grep -nE '^\s*go (func\|[A-Za-z_])' consumer.go` — no match; no goroutine spawned in the diff |
| DOM-25 | Client-interpreted wire values resolved from a table, not literals | N/A | The changed mapping (`buildIncreaseExperienceConfig`, `consumer.go:369-423`) sets `Amount`/percentage/hour/bonus payload fields from pre-existing semantic `ExperienceDistributionType*` string keys (`kafka/message/character/kafka.go:134-147`) — these are display quantities, not dispatcher-mode/sub-op/notice-fail-reason classification codes selected from a version-dependent lookup table (the category `patterns-multitenancy...`/`anti-patterns.md:136-165` actually governs); no new or changed literal client wire code is introduced by this diff, and the downstream call `charpkt.CharacterStatusMessageOperationIncreaseExperienceBody(...)` (`consumer.go:426-430`) is unchanged from before the refactor |
| DOM-31 | Tenant/trace identifiers travel in context only | PASS | `consumer_test.go:64` `ctx := tenant.WithContext(context.Background(), tm)`; tenant reaches production code only via `tenant.MustFromContext(ctx)` (`consumer.go:604`, `617`, `630`, `648`); no `rest.go`/`resource.go` exists in the package to carry a tenant on a public surface |

### kafka/message/character (support package)

Files present: `kafka.go`, `channel_change.go` (untouched by this diff). No `model.go`, `entity.go`, `rest.go`, `provider.go`, `processor.go`, `resource.go`.

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01..05 | (as above) | N/A | `kafka.go` holds only `const`/`var`/struct type declarations for Kafka event bodies — no Processor/RestModel/requests/Entity/Builder/Model/write/read symbol present |
| FILE-06 | No catch-all bundling ≥2 responsibilities | N/A | Same — nothing to bundle |
| DOM-21 | No redeclaration of a shared `libs/atlas-constants` type/const | N/A | Diff adds doc comments to the existing `ExperienceDistributionType*` consts and one new `[]string` var (`AllExperienceDistributionTypes`, `kafka.go:160-176`) — no new `type X`, named `const` block, or numeric-literal classification is declared |
| DOM-26 | Every goroutine via `routine.Go` | PASS | `grep -nE '^\s*go (func\|[A-Za-z_])' kafka.go` — no match |
| DOM-25 | Client-interpreted wire values | N/A | Same disposition as above — `AllExperienceDistributionTypes` is a documentation/exhaustiveness registry of pre-existing semantic keys, not a wire-code table |

## Security Review

N/A — SEC-* family did not fire; the diff handles no authentication, authorization, tokens, redirects, or secrets.

## Not evaluable from the diff

- None. Both changed non-test files and the new test file were read in full, and every rule whose trigger fired was settled by the diff plus targeted greps within the two changed packages (`kafka/consumer/character`, `kafka/message/character`). No cross-package or cross-service lookup was required to dispose of any item.

## Summary

### Blocking (must fix)
- None.

### Non-Blocking (should fix)
- None.

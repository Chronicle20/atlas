# Backend Audit — task-227 world-transfer fixes (62b0b733c..HEAD)

- **Service Path:** services/atlas-character, services/atlas-channel, libs/atlas-packet, services/atlas-configurations
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-08-17
- **Build:** PASS (all 4 modules)
- **Tests:** all passed (see below), 0 failed
- **Overall:** NEEDS-WORK

## Build & Test Results

```
services/atlas-character/atlas.com/character: go build ./... -> exit 0
services/atlas-character/atlas.com/character: go test ./... -count=1 -> ok (pending_change 198.716s, all other packages ok)
services/atlas-channel/atlas.com/channel:     go build ./... -> exit 0
services/atlas-channel/atlas.com/channel:     go test ./... -count=1 -> ok (all packages, including pendingchange, socket/handler)
libs/atlas-packet:                            go build ./... -> exit 0
libs/atlas-packet:                            go test ./... -count=1 -> ok (all packages)
services/atlas-configurations/atlas.com/configurations: go build ./... -> exit 0
services/atlas-configurations/atlas.com/configurations: go test ./... -count=1 -> ok (templates 0.453s, all others ok)
```

## Applicability

| Family | Fired? | Trigger observation |
|---|---|---|
| DOM structure (DOM-01..05,11,16) | Yes | `pending_change/resource.go`, `pending_change/rest.go`, `pendingchange/rest.go` all present/changed |
| FILE placement (FILE-01..06) | Yes | Every changed Go package audited |
| SUB sub-domain (SUB-01..04) | N/A | No changed package has `resource.go` without `model.go` — `pending_change` has `model.go`; `pendingchange` and `socket/handler` have no `resource.go` at all |
| REST (DOM-06..09,12..15,17..19,32) | Yes | `pending_change` has `resource.go`, `processor.go`; `pendingchange` has `processor.go` |
| Constants reuse (DOM-21) | No new declaration | Diff does not declare a new type/const-block/numeric-classification that shadows an `atlas-constants` equivalent — reuses `world.Id` throughout, and character/account ids continue as bare `uint32` matching the pre-existing service convention (not a new declaration) |
| Testing (DOM-10,20,24,33) | Yes | Diff touches many `_test.go` files |
| Cache (DOM-29) | N/A | No `cache.go`, no cached state added |
| Messaging (DOM-30) | N/A | No `producer.go` touched; no `AndEmit`/`message.Emit` in new code |
| Multi-tenancy (DOM-31) | Yes | `rest.go` present in both `pending_change` and `pendingchange` |
| Migration hygiene (DOM-34,35) | N/A | No symbol moved between service and `libs/atlas-*` |
| Deploy & topics (DOM-22,23) | N/A | No new `libs/atlas-*` module, no new/renamed Kafka topic env var |
| Runtime safety (DOM-26) | Yes | Non-test Go files changed; no bare `go` statements found |
| Channel wire values (DOM-25) | Yes | Diff touches `atlas-channel` and `libs/atlas-packet` |
| Resilience (DOM-27,28) | Yes (27) / N/A (28) | `atlas-character` is DB-backed and the new handler writes 500 via `server.WriteErrorResponse`; no `model.Decorator`/enrichment path changed |
| External clients (EXT-01..04) | Yes | `pending_change/requests.go`, `pendingchange/requests.go` call other atlas services |
| Scaffolding (SCAFFOLD-01..09) | N/A | No new `services/atlas-<svc>/` dir; no new channel `Writer`/`Handler` registered (existing writer's `errors` table gains alias entries only); `deploy/shared/routes.conf` untouched |
| Security (SEC-01..04) | N/A | No token/JWT/redirect/system-secret handling touched — the WORLD_TRANSFER credential check (PIC/birthday) is unchanged pre-existing gameplay logic, not this diff's session-auth substrate |

## Checklist Results

### atlas-character `pending_change` (domain — has `model.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor in `processor.go`/`processor_<group>.go` | PASS | `processor.go` carries the interface/constructor; `processor_eligibility.go` is the idiomatic `processor_<group>.go` split (all `evaluateTransfer*`/`checkXxx`/gate methods) |
| FILE-02 | RestModel/Transform/JSON:API in `rest.go` | PASS | `EligibilityRestModel` and its `GetName`/`GetID`/`SetID` live in `rest.go:110-131` (unchanged by this diff, reused by the new handler) |
| FILE-03 | Cross-service requests in `requests.go` | PASS | `requests.go` carries `inFamily`, `requestFamilyTree`, and every other gate client |
| FILE-06 | No catch-all file | PASS | Package file list has no `pending_change.go` bundling ≥2 responsibilities |
| DOM-04 | `Transform(Model)(RestModel,error)` in `rest.go` | PASS | Pre-existing `Transform` for `RestModel` still present (unchanged); `EligibilityRestModel` is a read-only projection with no back-and-forth Model, mirroring the pre-existing `.../transfer-eligibility` route this diff extends |
| DOM-06 | Processor ctor takes `logrus.FieldLogger` | PASS | `resource.go:116` — `NewProcessor(d.Logger(), d.Context(), d.DB())` |
| DOM-07 | Handler passes `d.Logger()` | PASS | `resource.go:116` |
| DOM-09 | Every `Transform` call checked | N/A | New handler does not call `Transform` — it round-trips `EligibilityRestModel` directly (no domain `Model`) |
| DOM-12 | No `os.Getenv` in handler | PASS | `resource.go` new handler has no `os.Getenv` |
| DOM-13 | No cross-domain orchestration in handler | PASS | `handleGetTransferEligibilityIndependent` (resource.go:113-134) only calls the processor and marshals the result |
| DOM-14 | Handler calls processor, not provider | PASS | `resource.go:116-117` |
| DOM-15 | No db writes in handler | PASS | No `db.Create`/`Save`/`Delete` in the new handler |
| DOM-17 | Domain errors map to correct status | PASS | `statusForError` (resource.go:310-328, unchanged by this diff) still used via the shared error path |
| DOM-27 | 503 via `server.WriteErrorResponse` on transient DB error | PASS | `resource.go:120` — new handler's error branch uses `server.WriteErrorResponse`, same as every other handler in the file |
| DOM-31 | Tenant/trace never a REST field/param | PASS | No `tenant.` reference in `resource.go`; new route's only path param is `characterId` |
| DOM-32 | Routes via `server.RegisterHandler`/`RegisterInputHandler[T]` | PASS | `resource.go:44` — new route registered through the same `registerGet` (`rest.RegisterHandler`) helper as every other GET route in the file; no bare `http.HandlerFunc` |
| EXT-01 | Target REST model has `SetToOneReferenceID`/`SetToManyReferenceIDs` | FAIL | `pending_change/rest.go:241-247` — `familyMemberRestModel` (the model backing the changed `inFamily` function, `requests.go`) has only `GetName`/`GetID`/`SetID`, no `SetToOneReferenceID`/`SetToManyReferenceIDs`. Pre-existing gap, but `inFamily`'s logic was substantively rewritten in this diff and its correctness depends on this model's unmarshal contract |
| EXT-03 | Only genuine 404 -> "not found" | PASS | `requests.go` (post-diff) — `inFamily` maps only `errors.Is(err, requests.ErrNotFound)` to `false, nil`; every other error (transport/decode/5xx) propagates as `false, err` |
| EXT-04 | URL via `requests.RootUrl` | PASS | `requests.go` — `familyBaseUrl()`/equivalent compose via `requests.RootUrl(...)` (unchanged pattern) |
| DOM-20 | Table-driven tests | PASS (mixed) | `requests_test.go:107-171` `TestInFamily` and `processor_eligibility_test.go:166-264`/`328-415` use `tests := []struct{...}` + `t.Run`; single-scenario tests (`TestEligibilityGate1WorldSame` etc.) have nothing to tabulate |
| DOM-10 | Test DB calls `RegisterTenantCallbacks` | Not evaluable from the diff | `newProcessorTestDB(t)` helper is not in the diff; would need to read its definition (likely in an unchanged `_test.go` in the same package) to confirm |
| DOM-24 | Producer stub on emit-reaching tests | N/A | No test added/changed in this diff reaches `AndEmit`/`message.Emit`/`producer.Produce` |
| DOM-33 | Mocks updated for interface change | PASS (by absence) | `grep` for `pending_change.Processor` mock implementations in `services/atlas-character/` returns nothing — the interface has no mock to update |

### atlas-channel `pendingchange` (support — no `model.go`, no `resource.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor in `processor.go` | PASS | `processor.go:30-50` (interface + `NewProcessor` + `ProcessorImpl` methods, including the new `CheckTransferEligibilityIndependent`) |
| FILE-02 | RestModel/JSON:API methods in `rest.go` | PASS | `rest.go:99-116` — new `EligibilityRestModel` + `GetName`/`GetID`/`SetID` |
| FILE-03 | Request functions in `requests.go` | PASS | `requests.go:87-93` — new `requestTransferEligibilityIndependentUrl`/`requestTransferEligibilityIndependent` |
| FILE-06 | No catch-all file | PASS | No `pendingchange.go` |
| DOM-04 | `func Transform(` in `rest.go` | FAIL | `rest.go` has no `func Transform(` at all — the package has no domain `Model`, so `identityRestModel`/`identityEligibilityRestModel` (the closest equivalent) live in `processor.go:63-71`, not `rest.go`, and are not named `Transform`. This gap pre-dates this diff's range (package created at 6de1ae5b7) but `rest.go` is a changed file in this diff (the new `EligibilityRestModel` was appended to it), so the rule's literal grep still fails against the file's current state |
| DOM-05 | `TransformSlice` used by list handlers | FAIL | Same as above — no `func TransformSlice(` in `rest.go`; `GetByCharacterId` (processor.go:56-58) uses `requests.SliceProvider` + `identityRestModel` instead |
| DOM-18 | JSON:API interface (`GetName`/`GetID`/`SetID`) | PASS | `rest.go:105-116` (`EligibilityRestModel`); pre-existing `RestModel`/`CreateInputRestModel`/`CancelInputRestModel` likewise (rest.go:25-91, unchanged by this diff) |
| DOM-19 | Flat request models | PASS | `CreateInputRestModel`/`CancelInputRestModel` (rest.go, unchanged) are flat structs, no nested `Data`/`Attributes` |
| DOM-31 | Tenant/trace never a REST field | PASS | No `tenant.` reference anywhere in the package's `.go` files |
| EXT-01 | Target REST model has `SetToOneReferenceID`/`SetToManyReferenceIDs` | FAIL | `rest.go:99-116` — the new `EligibilityRestModel` has only `GetName`/`GetID`/`SetID`; no `SetToOneReferenceID`/`SetToManyReferenceIDs`. Same gap pre-exists on the unchanged `RestModel`/`CreateInputRestModel`/`CancelInputRestModel` (rest.go:11-91) — not introduced by this diff, but present in the same changed file |
| EXT-02 | httptest-backed integration test with representative fixture | FAIL | `grep -rl CheckTransferEligibilityIndependent services/atlas-channel/` finds only `processor.go` and the caller `cash_shop_check_transfer_world_possible.go` — zero test files reference it. The new client method has no test at all (not `httptest`, not any other form) in this package; the only exercise of the seam is via the swappable `checkPossibleTransferEligibilityIndependentFunc` package var in `socket/handler`, which bypasses the real unmarshal path entirely |
| EXT-03 | Only genuine 404 -> "not found" | PASS (trivially) | `processor.go:76-82` — `CheckTransferEligibilityIndependent` does not collapse any error into a domain "not found"; every error propagates verbatim |
| EXT-04 | URL via `requests.RootUrl` | PASS | `requests.go:58-60` — `getBaseRequest()` returns `requests.RootUrl("CHARACTERS")` |
| DOM-20 | Table-driven tests | PASS | `processor_test.go:27-142` (`TestRequestNameChange`), `146-248` (`TestCancelPendingChange`) use `tests := []struct{...}` + `t.Run` |
| DOM-24 | Producer stub on emit-reaching tests | N/A | No test in this package reaches `AndEmit`/`message.Emit`/`producer.Produce` |
| DOM-33 | Mocks updated for interface change | PASS (by absence) | No mock of `pendingchange.Processor` exists anywhere under `services/atlas-channel/` |

### atlas-channel `socket/handler` (support — no `model.go`, no `resource.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-06 | No catch-all file | PASS | `cash_shop_check_transfer_world_possible.go`/`cash_shop_operation.go` are op-specific handler files, not a package-named catch-all |
| DOM-25 | Client-interpreted bytes resolved from tenant writer-options table | PASS | `cash_shop_operation.go:402-419` — new `transferFailureReasonConfigured` resolves via `writer.TenantWriterOptions(...)` + `atlaspacket.CodeConfigured(opts, "errors", reason)`, no Go literal byte; `check_transfer_world_possible_result.go:225-227,292-294,331-335` — `atlas_packet.WithResolvedCode("operations", ...)` |
| DOM-26 | No bare `go` statement | PASS | `grep -nE '^\s*go '` over both changed files returns nothing |
| DOM-13/14 | No cross-domain orchestration / provider calls bypassing processor | PASS | `CashShopCheckTransferWorldPossibleHandleFunc` (cash_shop_check_transfer_world_possible.go:97-185) calls only the swappable seam funcs (`checkPossible*Func`), each itself delegating to a `NewProcessor(...)` call |
| DOM-20 | Table-driven tests | PASS | `cash_shop_check_transfer_world_possible_test.go` uses `t.Run` subtests for every multi-case scenario (`TestWorldTransferCheckNeverEmitsStorageWarning`, `TestTransferWorldNameListIsIndexedByWorldId`, `TestTransferWorldPossibleWorldListFailureRefusesRatherThanCrashing`); single-scenario tests have nothing to tabulate |
| DOM-24 | Producer stub on emit-reaching tests | N/A | These handlers write via `session.Announce`/`writer.Producer` (socket writer), not the Kafka `AndEmit`/`message.Emit` path; no test reaches it |

### libs/atlas-packet `cash/clientbound` (support)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-25 | Wire bytes resolved from tenant table | PASS | `check_transfer_world_possible_result.go:225-227,292-294,331-335` — `atlas_packet.WithResolvedCode`, no literal bytes |
| DOM-20 | Table-driven tests | PASS | `check_transfer_world_possible_result_test.go` — `TestCheckTransferWorldPossibleResultReasonMapping` asserts the reason-arm map as data (unchanged shape, `required` slice extended by 1 line) |

### atlas-configurations `templates` (support, seed-data test package)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-20 | Table-driven tests | FAIL | `world_transfer_alias_test.go` — none of `TestWorldTransferReasonAliasesResolveToAnchorCode` (lines 64-102), `TestCannotTransferToNewWorldHasNoAlias` (108-129), `TestWorldTransferUnmappableTemplatesCarryNoAliases` (134-154) uses the `tests := []struct{...}` + `t.Run` shape, even though each iterates multiple cases (every catalog entry × every alias group, or a fixed list of two named templates) where a failing case gives no per-case isolation or `t.Run` name in `go test -v` output |
| FILE-06 | No catch-all file | N/A | New file is a single-purpose test file, not a catch-all production file |

## Security Review

Not applicable — no `SEC-*` trigger fired. This diff does not touch JWT parsing, revocation/logout, redirect targets, or secret handling. The WORLD_TRANSFER "second password" (PIC/birthday) comparison (`transferWorldCredentialMatches`, `cash_shop_check_transfer_world_possible.go:235-243`) is unchanged pre-existing gameplay validation, not the service's session-auth substrate.

## Not evaluable from the diff

- DOM-10: `newProcessorTestDB(t)` (used throughout `processor_eligibility_test.go`) is not itself part of this diff's changed files; would need to read its definition in an unchanged sibling `_test.go` in `pending_change` to confirm it calls `database.RegisterTenantCallbacks(l, db)`.

## Summary

### Blocking (must fix)
- EXT-02: `services/atlas-channel/atlas.com/channel/pendingchange/processor.go:76-82` (`CheckTransferEligibilityIndependent`) has zero test coverage — no `httptest`-backed test serves a representative JSON:API fixture and asserts a populated `EligibilityRestModel`; the only exercise of the seam bypasses unmarshal entirely via a swappable func var in `socket/handler`.
- EXT-01: `services/atlas-channel/atlas.com/channel/pendingchange/rest.go:99-116` — the new `EligibilityRestModel` implements only `GetName`/`GetID`/`SetID`, not `SetToOneReferenceID`/`SetToManyReferenceIDs`; api2go errors on any response carrying a `relationships` block.
- DOM-04/DOM-05: `services/atlas-channel/atlas.com/channel/pendingchange/rest.go` has no `func Transform(`/`func TransformSlice(` — the package's identity-transform equivalents (`identityRestModel`/`identityEligibilityRestModel`) live in `processor.go` under different names, so the mechanical rule as written fails against this file's current state.
- EXT-01: `services/atlas-character/atlas.com/character/pending_change/rest.go:241-247` — `familyMemberRestModel`, the model backing this diff's rewritten `inFamily` function, lacks `SetToOneReferenceID`/`SetToManyReferenceIDs`.
- DOM-20: `services/atlas-configurations/atlas.com/configurations/templates/world_transfer_alias_test.go` — three multi-case tests do not use the `tests := []struct{...}` + `t.Run` table-driven shape the guideline requires.

### Non-Blocking (should fix)
- None beyond the items above; all other applicable checks passed with cited evidence.

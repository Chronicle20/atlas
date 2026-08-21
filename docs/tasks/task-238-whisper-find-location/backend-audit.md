# Backend Audit — task-238-whisper-find-location

- **Service Path:** services/atlas-channel, services/atlas-maps (+ libs/atlas-constants, libs/atlas-packet)
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-08-18
- **Build:** PASS
- **Tests:** PASS (all packages green; see note on isolated `GOCACHE`)
- **Overall:** NEEDS-WORK

## Build & Test Results

`go build ./... && go test ./... -count=1` run module-local, per changed module:

- `services/atlas-channel/atlas.com/channel`: build PASS, `go test ./...` all packages `ok`.
- `services/atlas-maps/atlas.com/maps`: build PASS, `go test ./...` all packages `ok`. Two packages ran anomalously long: `atlas-maps/kafka/consumer/cashshop` 98.985s and `atlas-maps/kafka/consumer/character` 59.521s (all other packages <3s). This runtime is corroborating evidence for the DOM-24 finding below (unstubbed Kafka producer retry/backoff), not a build/test failure.
- `libs/atlas-constants`: build PASS, tests PASS.
- `libs/atlas-packet`: build/test initially reported `[build failed]` / `internal/saferio not in std` / cache-link errors under the shared `GOCACHE`. This is environmental — a concurrent `tools/verify.sh` run against the same working tree was writing to the same Go build cache. Re-run with an isolated `GOCACHE=/tmp/claude-1000/gocache-audit` (read-only w.r.t. the repo; no working-tree mutation) built and tested clean: all packages `ok`, including `field/clientbound` (the changed package). Recorded as PASS on that basis.

No test failures anywhere. The overall status is NEEDS-WORK, not FAIL, because the objective build/test gate passed; NEEDS-WORK is driven entirely by the FAIL checklist items below.

## Applicability

| Family | Fired? | Trigger observation |
|---|---|---|
| DOM structure (DOM-01..05,11,16) | Yes | `model.go`/`entity.go`/`rest.go`/`provider.go` present in `atlas-maps/character/location` (changed); `atlas-channel/character`, `atlas-channel/session` have `model.go` (changed) |
| FILE placement (FILE-01..06) | Yes | Runs unconditionally on every changed Go package |
| SUB sub-domain (SUB-01..04) | No | No changed package has `resource.go` without `model.go` — `atlas-maps/character/location` has both; `atlas-channel/maps/location` and `atlas-channel/socket/handler` have neither |
| REST (DOM-06..09,12..19,32) | Yes | `atlas-maps/character/location` has `resource.go`, `rest.go`, `processor.go` (package changed) |
| Constants reuse (DOM-21) | Yes | New type `character.PresenceState` in `libs/atlas-constants/character/presence.go` |
| Testing (DOM-10,20,24,33) | Yes | Diff touches many `_test.go` files; new Kafka-consumer tests reach `AndEmit`-backed emit paths |
| Cache (DOM-29) | No | No changed package has `cache.go` or cached processor state |
| Messaging (DOM-30) | No | No new `AndEmit`/`message.Emit`/`producer.ProviderImpl` call site added by this diff — existing emit sites (`EnterAndEmit`/`ExitAndEmit`/`TransitionChannelAndEmit`) are unchanged, only newly *exercised* by tests (covered under DOM-24, not DOM-30) |
| Multi-tenancy (DOM-31) | Yes | `atlas-maps/character/location/rest.go` changed; `atlas-channel` location `Get`/`GetField` open a tenant-scoped REST client context |
| Migration hygiene (DOM-34,35) | No | Diff adds only new symbols; nothing moved/extracted between a service and `libs/atlas-*` |
| Deploy & topics (DOM-22,23) | No | No new `libs/atlas-*` module added; no Kafka topic env var added or renamed — the cash-shop and character status topics are reused unchanged |
| Runtime safety (DOM-26) | Yes | Non-test Go files changed; no bare `go` statements found (`git diff` grep, zero hits) |
| Channel wire values (DOM-25) | Yes | Diff touches `services/atlas-channel` and `libs/atlas-packet` |
| Resilience (DOM-27,28) | Partial | `atlas-maps/character/location` is DB-backed but `resource.go` itself is unchanged (DOM-27 confirmed pre-existing PASS by targeted grep); DOM-28's decorator/enrichment shape does not match `findDecision` — see table |
| External clients (EXT-01..04) | Yes | `atlas-channel/maps/location/requests.go` calls `requests.GetRequest[RestModel]` against atlas-maps |
| Scaffolding (SCAFFOLD-01..09) | No | No new service directory; no new atlas-channel `Writer`/`Handler` registered (`fieldcb.WhisperWriter` is pre-existing); `deploy/shared/routes.conf` untouched |
| Security (SEC-01..04) | No | Diff touches no auth/token/redirect/secret handling code |

## Checklist Results

### atlas-channel/socket/handler (support package — `character_chat_whisper.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01..06 | File placement | PASS | File carries only /find decision logic and dispatch; no `Processor`/`RestModel`/`requests` bundling — `services/atlas-channel/atlas.com/channel/socket/handler/character_chat_whisper.go:44-223` |
| DOM-25 | Client wire values resolved from a table, not literals | PASS | `resultMode` 0x09/0x48 and packet body construction are unchanged pre-existing code (confirmed against `git show 6fb58bdc5:.../character_chat_whisper.go:44-48`); no new client-interpreted literal introduced by this diff |
| DOM-26 | No bare `go` statements | PASS | `git diff 6fb58bdc5..HEAD -- '*.go' \| grep -E '^\+.*\bgo (func\|[A-Za-z_])'` — zero hits |
| DOM-20 | Table-driven tests | PASS | `character_chat_whisper_test.go:147-154` (`findArms` table), `:236-247` (GM concealment table), `:397-420` (FR-7 table), all driven with `t.Run` |
| DOM-13/14/15 | No cross-domain orchestration / provider calls / writes in handlers | N/A | Trigger is "package has `resource.go`" — this package has no `resource.go` |

### atlas-channel/maps/location (support/REST-client package — `requests.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-02 | `RestModel`/`Transform`/JSON:API methods live in `rest.go` | **FAIL (Important)** | `RestModel` type and its `GetName()`/`GetID()`/`SetID()`/`SetToOneReferenceID`/`SetToManyReferenceIDs` methods are defined in `requests.go`, not `rest.go` — `services/atlas-channel/atlas.com/channel/maps/location/requests.go:33-56`. FILE-02's explicit FAIL list names `requests.go` by name. Pre-existing before this branch (confirmed via `git show 6fb58bdc5`), but this diff adds a new field to the same misplaced struct and doubles down by adding a second responsibility to the same file (see FILE-05) — task-102's "wallet.go" precedent is exactly this shape and is not exempted by prevalence. |
| FILE-05 | Domain `Model` lives in `model.go` | **FAIL (Important)** | A new domain `Model` struct with `CharacterId()/WorldId()/ChannelId()/MapId()/Instance()/State()` accessors and its `Get()` constructor are added to `requests.go`, not `model.go` — `services/atlas-channel/atlas.com/channel/maps/location/requests.go:88-133`. This package has no `model.go` at all (`ls` confirms only `requests.go`, `requests_test.go`, `resolve.go`). |
| FILE-06 | No catch-all file bundling ≥2 responsibilities | Related finding, not literally triggered | The literal rule text names a "package-named catch-all file" (e.g. `location.go`); `requests.go` is not package-named, so FILE-06's own wording does not fire on the filename. The substance of the finding is fully captured by FILE-02 + FILE-05 above — `requests.go` now carries three of the FILE-01..06 responsibilities (REST model, cross-service requests, and domain Model) in one file. |
| EXT-01 | Target RestModel implements `SetToOneReferenceID`/`SetToManyReferenceIDs` | PASS | `requests.go:55-56` (no-op stubs) |
| EXT-02 | `httptest`-backed integration test with populated domain struct | PASS | `requests_test.go` `TestGet_DecodesState`, `TestGet_DecodesInField`, `TestGet_AbsentStateIsOffline`, `TestGet_UnrecognisedStateIsOffline` — all serve a JSON:API fixture via `httptest.NewServer` and assert populated `Model` fields |
| EXT-03 | Only genuine 404 maps to "not found"; 5xx/transport bubble up | PASS | `requests.go:117-124` (`Get`) maps `requests.ErrNotFound` → `ErrNotFound`, else returns the raw error; `requests_test.go` `TestGet_NotFoundIsErrNotFound` / `TestGet_InfrastructureErrorIsNotErrNotFound` assert the distinction |
| EXT-04 | Service URL via `requests.RootUrl`/`RootUrlFor`, no hardcoded DNS | PASS | `requests.go:58-60` `requests.RootUrlFor(ctx, "MAPS")` (pre-existing, unchanged) |
| DOM-31 | Tenant/trace never on a REST model/request body/query param | PASS | `RestModel`/`Model` carry only world/channel/map/instance/state — no tenant or trace field added — `requests.go:33-40, 91-98` |

### atlas-maps/character/location (domain package)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-04 | `entity`, `Migration`, `TableName` in `entity.go` | PASS | `entity.go:15-32` |
| DOM-16 | Writes live in `administrator.go` | PASS | `upsertLocation` and new `setLocationState` both in `administrator.go:1-60` (approx.); `processor.go`'s `SetState`/`SetStateIfOnline` call through them |
| DOM-06 | Processor constructor takes `logrus.FieldLogger` | PASS | `processor.go:34` `func NewProcessor(l logrus.FieldLogger, ...)` (pre-existing, unchanged by this diff) |
| DOM-07 | Handlers pass `d.Logger()` into `NewProcessor` | PASS | `resource.go:52,104` `NewProcessor(d.Logger(), d.Context(), db)` (unchanged; confirmed by targeted grep since `resource.go` itself is not in this diff but the package is in scope) |
| DOM-27 | DB-backed handler 500s use `WriteErrorResponse` | PASS | `resource.go:56,111,118` all call `server.WriteErrorResponse(d.Logger())(w)(err)` (unchanged; `resource.go` not touched by this diff) |
| DOM-33 | Interface change updates every mock | N/A | `location.Processor` gained `SetState`/`SetStateIfOnline`; no mock implementation of `location.Processor` exists anywhere in the service (`find ... -path "*location*mock*"` — no results) |
| DOM-10 | Test DB bootstrap calls `database.RegisterTenantCallbacks(l, db)` | **FAIL (Important)** | `processor_test.go:102-112` `newTestDB` calls `gorm.Open(sqlite.Open(":memory:"), ...)` directly with no `database.RegisterTenantCallbacks(l, db)` call. Pre-existing helper (unchanged body, confirmed via `git show 6fb58bdc5`), but three new tests added by this diff (`TestSetState_TransitionsWithoutDisturbingPosition`, `TestSet_PreservesState`, `TestSetStateIfOnline_DoesNotResurrectOfflineRow`, `processor_test.go:193-...`) newly exercise it, and `resource_test.go`'s two new tests (`TestTransform_CarriesState`, `TestRestModel_StateJSONKey`) reuse the same package-level gap. Per the Mindset rule against excusing a pre-existing pattern by prevalence, this is graded as a finding against the package as it stands in the diff. |
| DOM-28 | Fallible enrichment paths degrade loudly via `model.ErrDecorator`+`degrade.Observe` | N/A | No `model.Decorator[...]` implementation changed in this package; the new `SetState`/`SetStateIfOnline` writes are administrator-layer writes, not a remote-fetch enrichment/fallback path |

### atlas-maps/kafka/consumer/cashshop (support package)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-10 | Test DB bootstrap calls `database.RegisterTenantCallbacks` | **FAIL (Important)** | `consumer_test.go:23-33` — brand-new file (`git show 6fb58bdc5:.../cashshop/consumer_test.go` → path does not exist at merge-base). `newTestDB` opens `gorm.Open(sqlite.Open(":memory:"), ...)` with no `database.RegisterTenantCallbacks(l, db)` call. |
| DOM-24 | Test packages reaching an emit path install `producertest` (or per-test no-op injection) | **FAIL (Important)** | `consumer_test.go` `TestEnterHandler_SetsInCashShop` / `TestEnterHandler_IsIdempotent` / `TestExitHandler_SetsInField` / `TestExitHandler_DoesNotResurrectOfflineCharacter` / `TestEnterHandler_DoesNotResurrectOfflineCharacter` all call `handleStatusEventEnterFunc(db)(...)` / `handleStatusEventExitFunc(db)(...)`, which (`consumer.go:55-56,75-76`) construct `_map.NewProcessor(l, ctx, producer.ProviderImpl(l)(ctx), nil)` and call `p.ExitAndEmit(...)` / `p.EnterAndEmit(...)` — `map/processor.go:99-116` shows these route through `message.Emit(p.p)`, i.e. a live, unstubbed Kafka producer. `grep -rn "TestMain\|producertest\|InstallNoop\|WithProducer"` across the `cashshop` consumer package returns zero hits. Corroborated by the package's anomalous 98.985s test runtime (vs. <1s for sibling packages with no emit path) — consistent with the documented ~42s per-emit retry/backoff cost. |

### atlas-maps/kafka/consumer/character (support package)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-24 | Same rule as above | **FAIL (Important)** | New tests `TestLoginHandler_SetsInField`, `TestLogoutHandler_SetsOfflineAndPreservesPosition`, `TestChannelChangedHandler_SetsInFieldOnNewChannel` (`consumer_test.go:123-...`, all additions per diff) call `handleStatusEventLoginFunc`/`handleStatusEventLogoutFunc`/`handleStatusEventChannelChangedFunc`, each of which (`consumer.go:101-102,143-144,155-156`) builds `_map.NewProcessor(l, ctx, producer.ProviderImpl(l)(ctx), db)` and calls `EnterAndEmit`/`ExitAndEmit`/`TransitionChannelAndEmit`. No `producertest`/`TestMain`/`WithProducer` anywhere in this package (same grep, zero hits). Corroborated by the package's 59.521s runtime. |
| DOM-30 | Writes emit via `AndEmit`+`message.Buffer`, not a bare `producer.ProviderImpl` call from the success path | PASS | `EnterAndEmit`/`ExitAndEmit`/`TransitionChannelAndEmit` route through `message.Emit(p.p)(...)` — `map/processor.go:99-145` (unchanged, pre-existing pattern; this diff adds no new direct-producer call site) |

### libs/atlas-constants/character (support package — `presence.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-21 | No redeclaration of a type/const that already exists elsewhere | PASS | `PresenceState` is a genuinely new shared type, consumed identically by both `atlas-maps` (`entity.go:26`, `model.go`, `rest.go:12`) and `atlas-channel` (`requests.go:13,97`) rather than being redeclared per-service |
| DOM-20 | Table-driven tests | PASS | `presence_test.go:8-21` and `:24-39` both use `cases := []struct{...}` + loop/`t.Run` |

### libs/atlas-packet/field/clientbound (codec package — `whisper.go`, unchanged; `whisper_test.go` new)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-25 | Client wire literals only inside `libs/atlas-packet` codec internals | PASS (by exemption) | All the mode/findMode byte literals in `whisper_test.go` (e.g. `0x09`, `0x48`, `findMode`) are inside `libs/atlas-packet` — the documented exemption in the DOM-25 pass criteria ("No client wire code appears as a Go literal outside `libs/atlas-packet` codec internals") applies |
| DOM-20 | Table-driven / version-fixture tests | PASS | Round-trip tests loop over `pt.Variants` with `t.Run`; per-version byte-pin tests (`TestWhisperByteOutputV61/V48/V92`, etc.) follow the documented per-version byte-fixture playbook, which DOM-20 explicitly defers to |

## Security Review

Not applicable — SEC-01..04's trigger ("service handles authentication, authorization, tokens, redirects, or secrets") did not fire; this diff touches whisper/location/presence-state logic only.

## Not evaluable from the diff

- Packet-audit byte-fixture accuracy against the IDA disassembly cited in `whisper_test.go` (e.g. `@0x53e2a0`, `@0x4eabd7`) — verifying those addresses requires IDA/binary access outside this review's surface; the packet-audit process (`docs/packets/PROCESS.md`) is the system of record for that claim, not this checklist.
- Whether the GM-level claim in `character/model.go:61-70`'s comment ("GM levels above 1 exist in this repo — see `libs/atlas-saga/validation.go` and atlas-query-aggregator's character model") is accurate — those two locations are outside the changed-package surface for this diff and the claim is documentation, not an enforceable rule.

## Summary

### Blocking (must fix)
- FILE-02: `services/atlas-channel/atlas.com/channel/maps/location/requests.go:33-56` — `RestModel` and its JSON:API methods must move to `rest.go`.
- FILE-05: `services/atlas-channel/atlas.com/channel/maps/location/requests.go:88-139` — the new domain `Model` (and its `Get`/`NewModelForTest` constructors) must move to `model.go`.
- DOM-10: `services/atlas-maps/atlas.com/maps/character/location/processor_test.go:102-112` and `services/atlas-maps/atlas.com/maps/kafka/consumer/cashshop/consumer_test.go:23-33` — GORM test bootstraps must call `database.RegisterTenantCallbacks(l, db)`.
- DOM-24: `services/atlas-maps/atlas.com/maps/kafka/consumer/cashshop/consumer_test.go` and `services/atlas-maps/atlas.com/maps/kafka/consumer/character/consumer_test.go` — new tests reach live, unstubbed Kafka emit paths (`producer.ProviderImpl` via `_map.NewProcessor(...).EnterAndEmit`/`ExitAndEmit`/`TransitionChannelAndEmit`) with no `producertest.InstallNoop()`/`WithProducer` stub anywhere in either package; corroborated by 98.985s and 59.521s package test runtimes.

### Non-Blocking (should fix)
- FILE-06 (informational): the substance of a `wallet.go`-style collapse is present in `atlas-channel/maps/location/requests.go` even though the literal filename rule doesn't fire on `requests.go`; fixing FILE-02/FILE-05 resolves it.

### Informational (not a finding — controller ruling)
- `character/model.go:61-70` widens `Gm()` from `gm == 1` to `gm > 0` in `atlas-channel` only; seven other services still carry `gm == 1`. Per controller ruling this is a known, pre-existing, explicitly-scoped-out inconsistency and is not raised as blocking here.

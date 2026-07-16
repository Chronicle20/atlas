# Backend Audit — atlas-saga-orchestrator

- **Service Path:** services/atlas-saga-orchestrator/atlas.com/saga-orchestrator
- **Scope:** task-102 modified packages — `saga`, `mts`, `cashshop`, `kafka/message/{saga,mts/custody,cashshop}`, `kafka/consumer/{mts/custody,cashshop}`, `main.go`
- **Guidelines Source:** backend-dev-guidelines skill (file-responsibilities, patterns-saga, patterns-deploy, cross-service-implementation, testing-guide)
- **Date:** 2026-07-10
- **Build:** PASS
- **Vet:** PASS
- **Tests:** PASS (all packages `ok`; `saga` 0.439s, `-count=1`)
- **Overall:** NEEDS-WORK

## Build & Test Results

```
go build ./...   -> clean (exit 0)
go vet ./...     -> clean (exit 0)
go test ./... -count=1 -> all ok
  atlas-saga-orchestrator/saga                          ok  0.439s
  atlas-saga-orchestrator/kafka/consumer/cashshop       ok  0.013s
  atlas-saga-orchestrator/kafka/consumer/mts/custody    [no test files]
  atlas-saga-orchestrator/mts                           [no test files]
  ... (all remaining packages ok / no test files)
```

The objective build/vet/test gate PASSES. Overall NEEDS-WORK is driven by two Important
findings below, not by the gate.

---

## Counts

- Critical: 0
- Important: 2
- Minor: 3

### Important (one-liners)
- **saga/handler.go:468 `WithMtsProcessor` drops 11 of 29 HandlerImpl fields** — reconstructs the struct listing only 18 fields (nils out `reactorDropP, portalBlockingP, questP, storageP, buffP, transportP, savedLocationP, gachaponP, partyQuestP, reactorP, mapCommandP`). Latent field-drop bug. Currently unreachable (method is never called; only the Compensator's `WithMtsProcessor` is used, and that one was correctly refactored to `copy()`), but it is a landmine that mirrors the pre-existing buggy `WithCashshopProcessor`/`WithSystemMessageProcessor`. The new code should have used the same `copy()`-clone pattern the compensator adopted.
- **mts/requests.go:25-74 holds a JSON:API RestModel (`HoldingRestModel`) instead of a `rest.go`** — file-responsibilities.md assigns the `RestModel` (`GetName`/`GetID`/`SetID` + relationship stubs) to `rest.go`, and states `requests.go` is "always paired with a `rest.go` in the same package." The `mts` package has no `rest.go`. The sibling `cashshop` package does this correctly (`cashshop/rest.go` for models, `cashshop/requests.go` for the client fn). File-organization violation (not downgraded to Minor per the audit rule).

### Minor (one-liners)
- **kafka/consumer/mts/custody** has no test files — the four step-completion handlers (`handleAcceptedEvent/Released/Moved/ErrorEvent`) are exercised only transitively; the underlying `AcceptEvent`/`StepCompleted` paths are covered in the `saga` package, but the consumer arms themselves are untested.
- **mts** package has no test files of its own — the `mts.RequestHoldings` unmarshal path IS covered via `saga/mts_expansion_test.go` (httptest), which satisfies EXT-02, but the dispatch processor/providers have no direct package-local tests.
- **deploy env-suffix drift (observation, shared infra):** `deploy/k8s/overlays/main/kustomization.yaml` lists `COMMAND_TOPIC_WALLET`/`EVENT_TOPIC_WALLET_STATUS` for per-env suffixing but omits `COMMAND_TOPIC_MTS_CUSTODY`/`EVENT_TOPIC_MTS_CUSTODY_STATUS` (present only in the `pr` overlay). Functionally consistent in `main` (both producer and consumer read the same base value), so not a correctness break, but the env-isolation suffixing is asymmetric vs the `pr` overlay.

---

## Per-Package Results

### `mts` (NEW — support/dispatch + REST client)

Files: `processor.go` (173), `producer.go` (141), `requests.go` (84). No `rest.go`, no `model.go` → support package (dispatch + external client). DOM domain checklist N/A.

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE (rest.go) | RestModel belongs in rest.go | **FAIL** | `HoldingRestModel` (GetName/GetID/SetID at mts/requests.go:60-74) lives in requests.go; no mts/rest.go exists |
| FILE (processor) | dispatch logic in processor.go | PASS | mts/processor.go:87-173 — Buffer + AndEmit wrappers only |
| FILE (producer) | kafka providers in producer.go | PASS | mts/producer.go:15-141 — `*Provider` funcs returning `model.Provider[[]kafka.Message]` |
| Proc ctor | `NewProcessor(l FieldLogger, ctx)` | PASS | mts/processor.go:94; `l logrus.FieldLogger` (not `*logrus.Logger`) |
| EXT-01 | JSON:API relationship stubs | PASS | mts/requests.go:72-74 `SetToOneReferenceID`/`SetToManyReferenceIDs` present |
| EXT-02 | httptest integration test exercises unmarshal | PASS | saga/mts_expansion_test.go:141-206 serves a `holdings` JSON:API doc and asserts the snapshot (str 9, watk 11, slots 7, flags 4) round-trips through `RequestHoldings` |
| EXT-03 | 404 not conflated with other errors | PASS | mts/requests.go:79-84 returns `requests.GetRequest` error verbatim; no false not-found mapping |
| EXT-04 | URL via `RootUrl(DOMAIN)` | PASS | mts/requests.go:19 `requests.RootUrl("MTS")` |

### `cashshop` (MODIFIED — dispatch + REST client)

Files: `processor.go`, `producer.go`, `requests.go`, `rest.go`. All file responsibilities correct.

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE (rest.go) | RestModels in rest.go | PASS | cashshop/rest.go:12,47 `AssetRestModel`/`CompartmentRestModel` |
| FILE (requests) | client fn only in requests.go | PASS | cashshop/requests.go:21 `RequestCompartment` only |
| Topic correctness | wallet adjust → correct topic | PASS | cashshop/processor.go:53 `AwardCurrency` now emits to `EnvCommandTopicWallet` (COMMAND_TOPIC_WALLET) — fixes the NX-debit timeout; Accept/Release correctly use compartment topic (processor.go:65,77) |
| EXT-01 | relationship stubs | PASS | cashshop/rest.go:101,105 |
| EXT-04 | `RootUrl` | PASS | cashshop/requests.go:17 `RootUrl("CASHSHOP")` |

### `kafka/message/mts/custody` (NEW — wire contract mirror)

| Check | Status | Evidence |
|-------|--------|----------|
| Topic constant naming | PASS | custody/kafka.go:17 `COMMAND_TOPIC_MTS_CUSTODY`, :131 `EVENT_TOPIC_MTS_CUSTODY_STATUS` (uppercase, no dotted/versioned suffix) |
| Command + StatusEvent envelopes carry TransactionId | PASS | custody/kafka.go:35 (Command), :143 (StatusEvent) |
| Error status arm defined | PASS | custody/kafka.go:137 `StatusEventTypeError` + :171 `StatusEventErrorBody` |
| Own-copy documented (cannot import atlas-mts) | PASS | custody/kafka.go:9-14 documents the mirror precedent |

### `kafka/message/cashshop` (MODIFIED)

| Check | Status | Evidence |
|-------|--------|----------|
| Wallet error event added for fast-fail | PASS | cashshop/kafka.go:16 `StatusEventTypeError`, :45 `StatusEventErrorBody{TransactionId,Reason}` |

### `kafka/message/saga` (MODIFIED)

| Check | Status | Evidence |
|-------|--------|----------|
| MTS failure kind discriminators on Failed body | PASS | saga/kafka.go:30-34 `MtsFailureKind{Buy,List,TakeHome}`, `StatusEventFailedBody.MtsKind` (omitempty) |

### `kafka/consumer/mts/custody` (NEW — saga step-completion consumer)

| Saga-Pattern Check | Status | Evidence |
|--------------------|--------|----------|
| Consumer file exists for target service | PASS | kafka/consumer/mts/custody/consumer.go |
| `InitConsumers` subscribes to status topic | PASS | consumer.go:22-28 → `EnvStatusEventTopic` |
| `InitHandlers` registers all arms | PASS | consumer.go:30-48 registers Accepted/Released/Moved/Error |
| Success path → `StepCompleted(true)` | PASS | consumer.go:64 (Accepted), :81 (Released), :99 (Moved) |
| Failure path → `StepCompleted(false)` | PASS | consumer.go:116 (Error) |
| Terminal-race guard before completion | PASS | each handler gates on `p.AcceptEvent(txId, EventKind…)` (consumer.go:55,72,89,107) |
| Registered in main.go | PASS | main.go:95 `InitConsumers`, main.go:116 `InitHandlers` |

### `kafka/consumer/cashshop` (MODIFIED)

| Check | Status | Evidence |
|-------|--------|----------|
| Failure path added (fast-fail vs timeout) | PASS | consumer.go:73-95 `handleWalletErrorEvent` → `StepCompleted(false)` |
| Both arms registered | PASS | consumer.go:31-36 (Updated + Error) |
| DOM-24 producer stubbed in tests | PASS | kafka/consumer/cashshop/testmain_test.go `producertest.InstallNoop()`; no `t.Cleanup(ResetInstance)` |
| Both success + failure tested | PASS | consumer_test.go:43 (completes), :74 (fails MTS award), :102 (nil-tx ignored) |

### `saga` (MODIFIED — orchestrator core)

Saga Patterns checklist — every MTS async action added:

| Action | Handler (dispatch) | Acceptance events | Completion consumer | Compensation | Status |
|--------|--------------------|--------------------|--------------------|--------------|--------|
| `TransferToMts` (composite) | expanded → release_from_character + accept_to_mts_listing | `{}` (expanded) | via child steps | reverse-walk re-grants item | PASS (processor.go:1166,1519) |
| `WithdrawFromMts` (composite) | expanded → release_from_mts_holding + accept_to_character | `{}` (expanded) | via child steps | reverse-walk RestoreMtsHolding | PASS (processor.go:1168,1623) |
| `MtsSettlePurchase` (composite) | expanded → award_currency×2 + move | `{}` (expanded) | via child steps | reverse-walk currency reversal | PASS (processor.go:1170,1717) |
| `AcceptToMtsListing` (atomic) | handler.go:1962 | `{MtsCustodyAccepted, MtsCustodyError}` | mts/custody consumer | reverse-walk (no listing row on fail) + late RemoveMtsListing | PASS |
| `ReleaseFromMtsHolding` (atomic) | handler.go:2018 | `{MtsCustodyReleased, MtsCustodyError}` | mts/custody consumer | late RestoreMtsHolding | PASS |
| `MtsMoveListingToHolding` (atomic) | handler.go:2038 | `{MtsCustodyMoved, MtsCustodyError}` | mts/custody consumer | late RestoreListingFromHolding | PASS |
| `MtsBidEscrow` (atomic) | handler.go:2060 (reuses wallet AwardCurrency) | `{CashShopWalletUpdated, CashShopWalletError}` | cashshop wallet consumer | AwardCurrency reversal | PASS |

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| GetHandler wiring | all atomic MTS actions routed | PASS | handler.go:831-841 |
| No premature StepCompleted on async | PASS | handler.go MTS handlers return nil without marking complete (handler.go:2011,2032,2054,2072) |
| Acceptance + outcome tables | PASS | event_acceptance.go:163-169 (acceptance), :293-301 (outcomes: Moved/Accepted/Released Success, Error Failure) |
| UnmarshalJSON cases for all MTS payloads | PASS | model.go:1315-1356 (all 7 MTS payloads) |
| Reverse-walk compensation (dupe-safety) | PASS | compensator.go `compensateMtsOperation` + `DispatchMtsOperationRollbacks` (currency-negate, re-grant, RestoreMtsHolding) |
| Timeout triggers MTS rollback | PASS | timer.go:122-124 `DispatchMtsOperationRollbacks` for MtsOperation |
| Late-compensation inverses registered | PASS | compensator.go `lateCompensableActions` adds the 3 custody actions; `dispatchLateInverse` cases for each |
| Failure notify carries characterId + kind | PASS | producer.go `EmitSagaFailed`→`EmitMtsSagaFailed`/`extractMtsFailureTarget` (buy=buyer, list=seller, take-home=recipient) |
| Take-home completion notice guarded | PASS | producer.go `extractMtsTakeHomeResults` fires only from the single terminal COMPLETED emit |
| DOM-21 no shared-type duplication | PASS | model.go aliases `sharedsaga.*` MTS actions/payloads; payloads defined in libs/atlas-saga/payloads.go:560+ |
| DOM-24 producer stubbed | PASS | saga/testmain_test.go `producertest.InstallNoop()`; no cleanup-revert |
| Failure-path test coverage | PASS | mts_dupe_safety_test.go (3), mts_failure_notify_test.go (5), compensator_test.go:482-547 (late-comp ×3), mts_integration_test.go:168 |
| WithMtsProcessor builder | **FAIL** | handler.go:468-489 sets 18/29 fields, nils 11 processors; dead (unused) but latent bug — compensator’s equivalent (compensator.go:168) correctly uses `copy()` |

### `main.go` (MODIFIED)

| Check | Status | Evidence |
|-------|--------|----------|
| mts/custody consumer registered | PASS | main.go:13 import, :95 InitConsumers, :116 InitHandlers (Fatal on error) |
| No os.Getenv in handlers | PASS | env reads confined to main.go bootstrap (SAGA_* config) |

### Deploy (DOM-22/23)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-22 | Dockerfile lib blocks | N/A/PASS | No per-service Dockerfile; service uses shared repo-root Dockerfile; go.mod unchanged (no new shared lib) |
| DOM-23 | topics in env-configmap, KEY:"KEY" | PASS | env-configmap.yaml:53,77,131,152 all four topics `KEY: "KEY"` |
| DOM-23 | no literal override in manifest; envFrom configmap | PASS | atlas-saga-orchestrator.yaml:21-23 `envFrom: configMapRef: atlas-env`; no per-topic `- name:` literal |

---

## Summary

### Blocking (must fix)
_None — build, vet, and tests all pass; no Critical findings._

### Should fix (Important)
- **saga/handler.go:468** — `WithMtsProcessor` field-drop (18/29 fields). Refactor to the `copy()`-clone pattern the compensator uses (compensator.go:82), or remove the unused method. Fix the sibling `WithCashshopProcessor`/`WithSystemMessageProcessor` while there.
- **mts/requests.go:25-74** — move `HoldingRestModel` into a new `mts/rest.go`, leaving `requests.go` with `getBaseRequest` + `RequestHoldings` only (mirror the `cashshop` package layout).

### Nice to have (Minor)
- Add a package-local test for `kafka/consumer/mts/custody` covering the four step-completion arms.
- Add per-package tests for `mts` dispatch/providers.
- Align `deploy/k8s/overlays/main/kustomization.yaml` MTS-custody topic env-suffixing with the `pr` overlay.

## Final resolution (post-audit fixes)

Both Important were in task-102 code and are FIXED:
- **handler.go WithMtsProcessor field-drop — FIXED.** Replaced the field-by-field struct rebuild (which dropped 11 of 29 HandlerImpl fields to nil) with a shallow struct clone (`c := *h; c.mtsP = mtsP`), the pattern the compensator uses — a HandlerImpl field added later can never be silently dropped.
- **mts/requests.go RestModel misplacement (FILE-02) — FIXED.** `HoldingRestModel` + its JSON:API methods moved to a dedicated `mts/rest.go`; `requests.go` keeps only the client funcs (matches the sibling cashshop package).
Minors (test coverage on mts/custody + mts packages) deferred.

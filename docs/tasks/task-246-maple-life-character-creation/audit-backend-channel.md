# Backend Audit — atlas-channel & libs (task-246)

- **Scope:** `services/atlas-channel/` (17 files), `libs/atlas-packet/` (8 files), `libs/atlas-saga/` (3 files)
- **Range:** `24a33a2e6..77b302206`
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-08-21
- **Build:** PASS
- **Tests:** all packages `ok` (no FAIL) — see Build & Test Results
- **Overall:** NEEDS-WORK

## Build & Test Results

```
cd services/atlas-channel/atlas.com/channel && go build ./...      # exit 0, no output
cd services/atlas-channel/atlas.com/channel && go test ./... -count=1
  ok  atlas-channel/character/factory      0.036s
  ok  atlas-channel/kafka/consumer/seed    0.035s
  ok  atlas-channel/maplelife              0.025s
  ok  atlas-channel/socket/handler         2.210s
  (all other packages: ok or "no test files"; zero FAIL lines)

cd libs/atlas-packet && go build ./... && go test ./... -count=1
  ok  github.com/Chronicle20/atlas/libs/atlas-packet/cash/serverbound        0.115s
  ok  github.com/Chronicle20/atlas/libs/atlas-packet/maplelife/clientbound   0.026s
  ok  github.com/Chronicle20/atlas/libs/atlas-packet/maplelife/serverbound   0.028s

cd libs/atlas-saga && go build ./... && go test ./... -count=1
  ok  github.com/Chronicle20/atlas/libs/atlas-saga   0.005s
```

Per instructions, `tools/verify.sh` was not re-run (branch-end flagless run already passed).

## Applicability

| Family | Fired? | Trigger observation |
|---|---|---|
| DOM structure (DOM-01..05,11,16) | Fired, narrowly | `services/atlas-channel/atlas.com/channel/saga/model.go` and `libs/atlas-saga/model.go` are the only changed files literally named `model.go`. Both are type-alias/const re-export shims, not domain-object definitions — see per-rule disposition below. |
| FILE placement (FILE-01..06) | Fired | Every changed Go package runs this family unconditionally. |
| SUB sub-domain (SUB-01..04) | N/A | No changed package has `resource.go` without `model.go` — `git diff --name-only` shows no `resource.go` anywhere in scope. |
| REST (DOM-06..09,12..15,17..19,32) | Fired, narrowly | `character/factory/rest.go` and `processor.go` exist; no `resource.go` and no route registration in scope, so only DOM-06 (processor logger type) has a firing trigger. |
| Constants reuse (DOM-21) | Fired | New `saga.Type` constant `MapleLifeUse`; new `maplelife.Phase` type; new writer/handler constants. Checked against `libs/atlas-constants` and sibling packages for redeclaration. |
| Testing (DOM-10,20,24,33) | Fired | Diff adds/changes many `_test.go` files. |
| Cache (DOM-29) | Fired | `maplelife/registry.go` holds package-level cached/tracked state (`Registry.pending`) behind a `sync.Once` accessor. |
| Messaging (DOM-30) | Fired, narrowly | `kafka/consumer/seed/consumer.go` calls `saga.NewProcessor(...).Create(s)`, an existing (unchanged) abstraction; no direct `producer.ProviderImpl`/`AndEmit` call is added by this diff. |
| Multi-tenancy (DOM-31) | Fired | `character/factory/rest.go` exists; tenant travels via `libs/atlas-rest` header decorators, never a field. |
| Migration hygiene (DOM-34,35) | N/A | No symbol move/extraction between a service and a `libs/atlas-*` module in this diff — new code only. |
| Deploy & topics (DOM-22,23) | N/A | No new `libs/atlas-*` module added (both `atlas-packet`/`atlas-saga` already exist); `EnvEventTopicStatus = "EVENT_TOPIC_SEED_STATUS"` in `kafka/message/seed/kafka.go` is a **new topic env var reference**, but the topic itself and its configmap/overlay wiring belong to the cross-cutting seed-status topic already used by atlas-character-factory/atlas-login (out of this diff's changed-file set — no configmap/overlay files are in scope). See "Not evaluable from the diff." |
| Runtime safety (DOM-26) | Fired | Every non-test Go file in scope is a trigger; no bare `go` statement was added anywhere in the diff (`git diff` grep for `go func`/`go \w` in additions found none). |
| Channel wire values (DOM-25) | Fired | Diff touches `services/atlas-channel` and `libs/atlas-packet` directly. |
| Resilience (DOM-27,28) | N/A | No DB-backed handler (`http.StatusInternalServerError` + `database.Connect`) and no `model.Decorator`/enrichment fallback changed in scope. |
| External clients (EXT-01..04) | Fired | `character/factory/processor.go` and `requests.go` call `requests.PostRequest[T]` against `CHARACTER_FACTORY`. |
| Scaffolding (SCAFFOLD-01..09) | Fired, narrowly (07 only) | Diff registers new channel writers (`MapleLifeResultWriter`, `MapleLifeErrorWriter`) and a new handler (`MapleLifeCheckNameHandle`) in `main.go`. SCAFFOLD-01..06,08,09 do not fire (no new service directory). SCAFFOLD-07 requires reading `services/atlas-configurations/seed-data/templates/*` — owned by the sibling reviewer; see "Not evaluable from the diff." |
| Security (SEC-01..04) | N/A | atlas-channel's changed surface here is packet dispatch and a service-to-service REST call, not auth/token/redirect/secret handling. |

## Checklist Results

### `services/atlas-channel/atlas.com/channel/character/factory` (support package — no `model.go`; has `processor.go`, `requests.go`, `rest.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor interface/constructor/impl in `processor.go` | PASS | `character/factory/processor.go:17-24` (interface), `:33-39` (constructor), `:41-67` (impl methods) — all in one file. |
| FILE-02 | RestModel/Transform/JSON:API methods in `rest.go` | PASS | `character/factory/rest.go` — all `RestModel`, `MapleLifeCreateRestModel`, `CreateCharacterResponse` and their `GetName`/`GetID`/`SetID` live here; none in `processor.go` or `requests.go`. |
| FILE-03 | Cross-service request functions in `requests.go` | PASS | `character/factory/requests.go:23,60` — `requestCreate`/`requestCreateMapleLife` and `getBaseRequest`. |
| FILE-06 | No catch-all file carrying ≥2 responsibilities | PASS | Three single-purpose files (`processor.go`, `requests.go`, `rest.go`); no `factory.go` package-named file exists. |
| DOM-06 | Processor constructor takes `logrus.FieldLogger` | PASS | `character/factory/processor.go:33` — `func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor`. |
| DOM-21 | No redeclaration of an existing shared const/type | PASS | `item.ClassificationCharacterCreation` reused from `libs/atlas-constants/item/constants.go:116`, not redeclared; `character.NameReason*` reused from `character/name_validity_requests.go:35-38`, not redeclared. |
| DOM-31 | Tenant travels in context only | PASS | `character/factory/rest.go` — neither `RestModel` nor `MapleLifeCreateRestModel` carries a tenant field; `getBaseRequest` (`requests.go:19`) passes only `ctx` to `requests.RootUrlFor`, which threads tenant via `TenantHeaderDecorator` (`libs/atlas-rest/requests/decorated.go:11`). |
| EXT-01 | Target REST model implements `SetToOneReferenceID`/`SetToManyReferenceIDs` (even as no-ops) | **FAIL** | `character/factory/rest.go:84-96` — `CreateCharacterResponse` (the type unmarshaled by `requests.PostRequest[CreateCharacterResponse]` in both `requests.go:23` and `requests.go:60`, per `libs/atlas-rest/requests/response.go:9`'s `jsonapi.Unmarshal`) defines only `GetName`/`GetID`/`SetID` — no `SetToOneReferenceID`/`SetToManyReferenceIDs`. If the factory's response ever carries a `relationships` block, `jsonapi.Unmarshal` errors here. `RestModel` (`rest.go:11-51`) and `MapleLifeCreateRestModel` (`rest.go:60-80`) are request bodies only (Marshal side), so EXT-01 does not require the methods on them — but `CreateCharacterResponse` is unmarshal-side and is missing them. |
| EXT-02 | `httptest`-backed integration test with a representative JSON:API fixture, asserting a populated struct | PASS | `character/factory/processor_test.go:64-98` (`TestSeedCharacterSendsSessionSuppliedIds`) and `:228-260` (`TestCreateMapleLifePostsTheChosenValues`) serve `{"data":{"type":"characters","id":"tx-...","attributes":{"transactionId":"tx-..."}}}` fixtures and assert the returned `txId` is populated. |
| EXT-03 | Only genuine 404s map to "not found"; transport/decode/5xx bubble up | PASS | `character/factory/processor.go:41-48,60-67` passes the raw `err` from `requests.PostRequest` straight through with no reclassification; `libs/atlas-rest/requests/post.go:78-79` is the only 404→`ErrNotFound` mapping point and is untouched by this diff. The caller (`socket/handler/maple_life_create.go:237-247`) distinguishes only `ErrConflict`/`ErrBadRequest`, defaulting every other error (including any 5xx/transport failure) to the generic `UNKNOWN_ERROR` arm — it never maps a non-404 failure onto a "not found"/name-taken outcome. |
| EXT-04 | Service URL via `requests.RootUrl(<DOMAIN>)`, not hardcoded DNS | PASS | `character/factory/requests.go:19-21` — `requests.RootUrlFor(ctx, "CHARACTER_FACTORY")` (the context-aware sibling of `requests.RootUrl`, defined in the same `libs/atlas-rest/requests/url.go:20,34`). |

### `services/atlas-channel/atlas.com/channel/maplelife` (support package — no `model.go`, no `resource.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-06 | No catch-all file carrying ≥2 of the FILE-01..05 responsibilities | PASS | `registry.go` defines an in-memory `Phase`/`Entry`/`Registry` presentation-state tracker — none of Processor, RestModel, cross-service requests, DB entity, or Builder/domain-Model/administrator/provider are present; this is the "genuine single-purpose utility" the rule explicitly allows. |
| DOM-29 | Cache is an application-scoped singleton reached through an accessor, never per-instance state | PASS | `maplelife/registry.go:76-83` — `GetRegistry()` uses `sync.Once`, matching the `cache.go` singleton shape even though the file is named `registry.go`; no processor holds this state per-instance. |

### `services/atlas-channel/atlas.com/channel/kafka/consumer/seed` and `kafka/message/seed` (support packages)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-06 | No catch-all file | PASS | `kafka/message/seed/kafka.go` is DTO-only (`StatusEvent`, `CreatedStatusEventBody`, `FailedStatusEventBody`, topic/type consts); `kafka/consumer/seed/consumer.go` is init/handler wiring — single-purpose files, no domain-object/REST/request collapse. |
| DOM-24 | Test packages reaching an emit path install the shared `producertest` stub or a no-op producer | N/A | `kafka/consumer/seed/consumer_test.go:125-129` substitutes the `destroyCashItemFunc` package-var seam itself, so tests never reach `saga.Processor.Create`'s real `AndEmit`/Kafka path — the trigger ("reaches an emit path... directly or transitively") does not fire for these tests. |
| DOM-30 | DB writes emit through `AndEmit` + `message.Buffer`, not a direct `producer.ProviderImpl` call | N/A | `kafka/consumer/seed/consumer.go` performs no database write and calls no `producer.ProviderImpl`/`AndEmit` itself; it calls the pre-existing, unchanged `saga.NewProcessor(l, ctx).Create(s)` (`services/atlas-channel/atlas.com/channel/saga/processor.go:34`, not in this diff), which owns its own emission pattern outside this review's scope. |

### `services/atlas-channel/atlas.com/channel/socket/handler` (support package — no `model.go`, no `resource.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06 | Processor constructor takes `logrus.FieldLogger` | PASS | `socket/handler/maple_life_create.go:30,42,54` and `maple_life_check_name.go:26` — every `NewProcessor(l, ctx)` call passes the handler's own `logrus.FieldLogger` parameter, never `logrus.StandardLogger()`. |
| DOM-25 | Client-interpreted wire bytes resolved from tenant writer-options table, never a Go literal; classification routing, not slot-type | PASS | `socket/handler/character_cash_item_use.go:800` — dispatch is `if category == item.ClassificationCharacterCreation` (`category := item.GetClassification(itemId)` at `:783`), never a `CashSlotItemType`/slot comparison, matching the design constraint (comment at `:792-799` documents the cash-slot-type collision this avoids). Outbound arms use `mlcb.MapleLifeErrorBody(mlcb.MapleLifeErrorUnknownError \| MapleLifeErrorNameTakenAtSubmit \| MapleLifeErrorSuccess)` (`maple_life_create.go:113,157,205`; `kafka/consumer/seed/consumer.go:174,213`), which resolves through `atlas_packet.WithResolvedCode("operations", key, ...)` (`libs/atlas-packet/maplelife/clientbound/error.go:129-134`) — no raw byte literal is written by channel code. |
| DOM-25 (version gating) | `MajorAtLeast` idiom, not raw `>` | PASS | `socket/handler/character_cash_item_use.go:1128` — `mapleLifeSupported` uses `t.IsRegion("GMS") && t.MajorAtLeast(83) && t.MajorVersion() != 84`, not a raw `>` comparison. |
| DOM-13 | No cross-domain orchestration in handlers (belongs in processor layer) | WARN | `socket/handler/maple_life_create.go:102-231` (`handleMapleLifeCreate`) directly orchestrates account/character/name-validity/factory calls across four domains (`account`, `character`, `factory`) inline in the handler body, rather than delegating to a processor. This matches an established package-var-seam idiom already used elsewhere in this same file (`character_cash_item_use.go`'s existing arms), so it is consistent with the surrounding code's pattern — flagged as non-blocking since `resource.go` (the rule's literal trigger, "package has `resource.go`") is absent here; this is a socket dispatcher, not a REST resource handler, so DOM-13's literal trigger does not fire. Recorded as WARN rather than N/A because the underlying concern (orchestration logic embedded in a dispatch function) is real, even though the rule's REST-specific trigger does not. |

### `libs/atlas-packet/cash/serverbound`, `libs/atlas-packet/maplelife/*` (packet-codec packages, no `model.go`/`resource.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-06 | No catch-all file | PASS | Each file (`item_use_maple_life.go`, `error.go`, `result.go`, `check_name.go`) defines exactly one wire struct + its `Encode`/`Decode`/`Operation`/`String` — single-purpose codec files. |
| DOM-25 | No client-interpreted byte hardcoded as a Go literal | PASS | `maplelife/clientbound/error.go:129-134` (`MapleLifeErrorBody`) and `maplelife/clientbound/result.go:150-154` (`MapleLifeResultBody`) both resolve `nType`/`nResult` via `atlas_packet.WithResolvedCode("operations", key, ...)` — no literal wire byte in the codec. |
| DOM-21 | No redeclaration of an existing constant | PASS | `atlas_packet.WithResolvedCode` (pre-existing, `libs/atlas-packet/resolve.go:41`) is reused, not reimplemented. |

### `services/atlas-channel/atlas.com/channel/saga` and `libs/atlas-saga` (`model.go` changed — additive const only)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01/02/03/04/05/11/16 | Domain-structure rules keyed on `model.go`/`entity.go`/`rest.go`/`provider.go` | N/A | Both changed `model.go` files are pure re-export shims: `services/atlas-channel/atlas.com/channel/saga/model.go:9` ("// Re-export types from atlas-saga shared library") and `libs/atlas-saga/model.go` define only type aliases and const re-exports, never a locally-defined domain `Model` struct with private fields/accessors (the definition `file-responsibilities.md` gives for `model.go`). Neither package has `entity.go`, `rest.go`, `builder.go`, or `provider.go` (`ls services/atlas-channel/atlas.com/channel/saga/` → `model.go`, `processor.go`, `producer.go` only). Judgment call, recorded explicitly: the literal filename triggers the DOM family, but the rules' own triggers (a genuine domain `Model`, or co-located `entity.go`/`rest.go`/`provider.go`) do not fire against a content-free re-export shim, so DOM-01/02/03/04/05/11/16 are disposed N/A rather than graded against a file that was never attempting to be a domain package. This shim shape predates this diff (24a33a2e6) and this diff only adds one constant line to it. |
| DOM-21 | `saga.MapleLifeUse` const not a redeclaration | PASS | `libs/atlas-saga/model.go` (new `MapleLifeUse Type = "maple_life_use"`) and `services/atlas-channel/atlas.com/channel/saga/model.go:84` (`MapleLifeUse = sharedsaga.MapleLifeUse`) — the channel-side constant is a re-export of the single shared-library definition, not a second declaration of the string literal. |
| — | AP/SP payload addition is strictly additive | PASS | `libs/atlas-saga/payloads.go` — `AP uint16 \`json:"ap,omitempty"\`` and `SP string \`json:"sp,omitempty"\`` appended to `CharacterCreatePayload`; `libs/atlas-saga/payloads_test.go:47-83` (`TestCharacterCreatePayloadCarriesApAndSp`) asserts both round-trip when set and are absent from the marshaled JSON when zero-valued. |
| DOM-20 | Table-driven tests | PASS | `libs/atlas-saga/payloads_test.go` uses direct assertion style consistent with the file's existing tests (`TestDestroyAssetFromSlotPayloadTemplateIdRoundTrip`, pre-existing); the new test follows the same non-table shape as its sibling in the same file — file predates a table-driven convention for this specific file and the new test is consistent with what's already there, not a new deviation. |

## Not evaluable from the diff

- **DOM-22/23 (topic env var registration)** — `kafka/message/seed/kafka.go:3` references `EVENT_TOPIC_SEED_STATUS`, but confirming it is declared as `KEY: "KEY"` in the base env-configmap and re-listed in both overlays' `atlas-env` generator requires reading `deploy/` manifests, which are outside the changed-file set for this review (none of `deploy/**` appears in `git diff --name-only 24a33a2e6..77b302206 -- services/atlas-channel libs/atlas-packet libs/atlas-saga`). If this topic is genuinely new (rather than an existing one atlas-character-factory/atlas-login already emit to), its configmap/overlay wiring needs separate verification.
- **SCAFFOLD-07 (writer/handler seeded in tenant templates)** — verifying `MapleLifeResultWriter`, `MapleLifeErrorWriter`, and `MapleLifeCheckNameHandle` appear in `services/atlas-configurations/seed-data/templates/template_gms_{83,87,92,95}_*.json` requires reading `services/atlas-configurations`, which this task's brief explicitly assigns to the sibling reviewer to avoid scope collision. Not evaluated here.
- **`saga.Processor.Create`'s own DOM-30 compliance** — `kafka/consumer/seed/consumer.go` calls into `services/atlas-channel/atlas.com/channel/saga/processor.go:34`, an unchanged file not in this diff's scope; whether that pre-existing `Create` method itself follows the `AndEmit`+`message.Buffer` pattern was not re-verified here (out of scope, not touched by task-246).

## Summary

### Blocking (must fix)
- EXT-01: `services/atlas-channel/atlas.com/channel/character/factory/rest.go:84-96` — `CreateCharacterResponse` (the type both `requestCreate` and `requestCreateMapleLife` unmarshal responses into) is missing `SetToOneReferenceID`/`SetToManyReferenceIDs`. Add both as no-ops, per the established pattern in e.g. `data/equipment/rest.go:37-41`.

### Non-Blocking (should fix)
- DOM-13 (WARN): `services/atlas-channel/atlas.com/channel/socket/handler/maple_life_create.go:102-231` inlines account/character/name-validity/factory cross-domain orchestration directly in the socket handler rather than a processor layer. Consistent with the surrounding file's existing idiom (package-var seams), so not blocking, but worth a follow-up if this handler grows further.

### Judgment calls flagged for visibility
- DOM-01/02/03/04/05/11/16 disposed N/A against `saga/model.go` (both copies) despite the literal `model.go` filename trigger firing, because neither file defines a domain `Model` — see the table above for the full reasoning. A reviewer who disagrees with this reading should treat it as 7 additional FAILs rather than accept the N/A silently.

# Backend Audit — task-252-jukebox-cash-item

- **Scope:** All Go changes on branch `task-252-jukebox-cash-item`, range `d17404dbc..a3f2519e9`
- **Services touched:** atlas-channel, atlas-maps, atlas-saga-orchestrator
- **Libs touched:** libs/atlas-saga, libs/atlas-packet (codec only, no non-generated logic change)
- **Tools touched:** tools/packet-audit (`internal/matrix/build.go`, `cmd/run.go`)
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-08-21
- **Build:** PASS (atlas-channel, atlas-maps, atlas-saga-orchestrator, libs/atlas-saga, libs/atlas-packet, tools/packet-audit — `go build ./...` in each module, zero errors)
- **Tests:** atlas-channel 142 packages `ok`, atlas-maps 25 packages `ok`, atlas-saga-orchestrator 41 packages `ok`, libs/atlas-saga 1 package `ok`, libs/atlas-packet/tools/packet-audit `ok`/no test files — zero `FAIL` lines across any module
- **Overall:** NEEDS-WORK

## Note on this re-run

A prior instance of this audit was killed by a session limit before writing
any artifact. Its two live findings were re-derived independently in this
pass, not assumed:

- **EXT-01** is fixed. `services/atlas-channel/atlas.com/channel/jukebox/rest.go:22-34` defines `SetToOneReferenceID` and `SetToManyReferenceIDs` as no-ops with an explanatory comment citing EXT-01. Confirmed by direct read of the current file at tip `a3f2519e9`. The stale `docs/tasks/task-252-jukebox-cash-item/audit.md`/`backend-audit.md` predate this fix (evaluated at `6360d7538`) and are superseded by this document.
- **EXT-04**: `jukebox/requests.go:16-18` uses `requests.RootUrlFor(ctx, "MAPS")`, not the literal `requests.RootUrl(<DOMAIN>)` the checklist text shows. This is **not** a deviation. `requests.RootUrlFor` (`libs/atlas-rest/requests/url.go:34-65`) is the environment-aware successor to `RootUrl` introduced by task-232 (sparse ephemeral environments) — byte-identical to `RootUrl` when no per-request environment is set (`url.go:41`, "legacy: byte-identical to RootUrl"). Every other `requests.go` in atlas-channel (63 files checked, e.g. `character/requests.go:20-21`, `weather/requests.go`, `mount/requests.go`) already uses `RootUrlFor(ctx, "DOMAIN")` — confirmed by `grep -rln "RootUrlFor" services/atlas-channel/atlas.com/channel/`. `docs/tasks/task-232-sparse-ephemeral-environments/service-wiring-recipe.md:166-205` documents the repo-wide `RootUrl(` → `RootUrlFor(ctx, ...)` migration as the target pattern. `jukebox/requests.go` was authored directly against the already-migrated convention; EXT-04's checklist wording predates that migration but its substance (domain-parameterized resolver, no hardcoded DNS) is met. **PASS.**

## Build & Test Results

```
services/atlas-channel/atlas.com/channel:        go build ./...  -> no output (pass)
                                                  go test ./... -count=1 -> 142 "ok" lines, 0 FAIL
services/atlas-maps/atlas.com/maps:              go build ./...  -> no output (pass)
                                                  go test ./... -count=1 -> 25 "ok" lines, 0 FAIL
services/atlas-saga-orchestrator/.../saga-orchestrator: go build ./... -> no output (pass)
                                                  go test ./... -count=1 -> 41 "ok" lines, 0 FAIL
libs/atlas-saga:                                 go build ./... && go test ./... -count=1 -> 1 "ok" line, 0 FAIL
libs/atlas-packet:                                go build ./... && go test ./... -count=1 -> ok, 0 FAIL
tools/packet-audit:                               go build ./... && go test ./... -count=1 -> ok, 0 FAIL
tools/goroutine-guard.sh:                         exit 0 (89 modules)
-race spot check (kafka/consumer/map, jukebox tests, x3): ok, 1.2s
```

## Applicability

| Family | Fired? | Trigger observation |
|---|---|---|
| DOM structure (DOM-01..05,11,16) | Fired | `atlas-maps/map/jukebox` has `rest.go`; `atlas-channel/jukebox` has `rest.go`; neither has `model.go`/`entity.go`/`provider.go` |
| FILE placement (FILE-01..06) | Fired | Every changed Go package — no exemptions |
| SUB sub-domain (SUB-01..04) | Fired | `atlas-maps/map/jukebox` has `resource.go`, no `model.go` |
| REST (DOM-06..09,12..19,32) | Fired | `atlas-maps/map/jukebox` has `resource.go`/`rest.go`; `atlas-channel/jukebox` has `processor.go` |
| Constants reuse (DOM-21) | Fired | New types/consts: `CashSlotItemTypeSongPlayer`, `saga.PlayJukebox`/`PlayJukeboxPayload`, `mapKafka.CommandTypePlayJukebox`, jukebox event/command bodies |
| Testing (DOM-10,20,24,33) | Fired | Diff adds many `_test.go` files; `map_command.Processor` interface gains a method |
| Cache (DOM-29) | Fired | `atlas-maps/map/jukebox/registry.go` holds a `sync.Once` singleton with cached state |
| Messaging (DOM-30) | Fired | `producer.go`, `producer.ProviderImpl` call sites added in atlas-maps jukebox/tasks and saga-orchestrator map_command |
| Multi-tenancy (DOM-31) | Fired | Both jukebox packages have `rest.go`; changed code reads tenant state (`tenant.MustFromContext`) |
| Migration hygiene (DOM-34,35) | N/A | No symbol moved between a service and a `libs/atlas-*` module in this diff — `libs/atlas-saga` additions are new symbols, not migrations |
| Deploy & topics (DOM-22,23) | N/A | No new `libs/atlas-*` module added; no new/renamed Kafka topic env var — `PlayJukebox`/jukebox events ride the existing `EnvCommandTopicMap`/`EnvEventTopicMapStatus` topics |
| Runtime safety (DOM-26) | Fired | Non-test Go files changed across all three services; `tools/goroutine-guard.sh` exit 0 |
| Channel wire values (DOM-25) | Fired | Diff touches `services/atlas-channel` and `libs/atlas-packet` |
| Resilience (DOM-27,28) | N/A | No changed handler writes `http.StatusInternalServerError`; no `model.Decorator`/enrichment path touched |
| External clients (EXT-01..04) | Fired | `atlas-channel/jukebox` calls atlas-maps via `requests.GetRequest[RestModel]` |
| Scaffolding (SCAFFOLD-01..09) | N/A | No new `services/atlas-<svc>/` directory; no new channel `Writer`/`Handler` registration (jukebox writer `PlayJukeboxWriter` pre-exists from task-096/100, only invoked here); `deploy/shared/routes.conf` untouched |
| Security (SEC-01..04) | N/A | No token/auth/redirect/secret surface touched |
| patterns-provider.md (foundational) | N/A | No provider composition defined/changed in this diff |
| patterns-functional.md (foundational) | N/A | No curried-constructor/decorator/model-combinator pattern introduced |

## Checklist Results

### services/atlas-channel/atlas.com/channel/jukebox (support — REST client, no model.go/resource.go)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor in processor.go | PASS | `jukebox/processor.go:12-25` — interface, constructor (`FieldLogger`), `ProcessorImpl` all in `processor.go` |
| FILE-02 | RestModel/Transform/Extract/JSON:API methods in rest.go | **FAIL (Important)** | `RestModel`, `GetID`/`GetName`/`SetID` are in `jukebox/rest.go:3-20` but `func Extract(m RestModel) (RestModel, error)` is in `jukebox/processor.go:31-33`, not `rest.go`. Every other atlas-channel client package places `Extract` in `rest.go` — confirmed for 20+ siblings (`character/rest.go:89`, `messenger/rest.go:106`, `chalkboard/rest.go:27`, `buddylist/rest.go:39`, `chair/rest.go:28`, `channel/rest.go:50`, `door/rest.go:52`, `guild/rest.go:45`, `inventory/rest.go:117`, `kite/rest.go:62`, `macro/rest.go:46`, `merchant/rest.go:116`, `minigame/rest.go:37`, `drop/rest.go:54`, `mount/rest.go:21`, `note/rest.go:56`, `party/rest.go:113`, `pet/rest.go:49`, `account/rest.go:80`, `incubator/rest.go:38`). Graded against the file table (file-responsibilities.md `rest.go`: "Implement `Transform`/`Extract`..."), not against prevalence — this is a genuine misplacement. |
| FILE-03 | Cross-service requests in requests.go | PASS | `requests.RootUrlFor`, `requests.GetRequest[RestModel]`, `getBaseRequest` all in `jukebox/requests.go:16-25`; none in `rest.go`/`processor.go` |
| FILE-04 | entity.go conventions | N/A | No `entity.go` — package is a REST client with no DB-backed entity |
| FILE-05 | Builder/Model/administrator/provider/state placement | N/A | No `builder.go`, `model.go`, `administrator.go`, `provider.go`, or `state.go` content exists in this package to misplace |
| FILE-06 | No catch-all file | PASS | Files: `processor.go`, `requests.go`, `rest.go`, `mock/processor.go` — each single-purpose except the FILE-02 misplacement above, which is a placement finding, not a catch-all-file finding |
| DOM-06 | Processor ctor takes `logrus.FieldLogger` | PASS | `jukebox/processor.go:21` — `func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor` |
| DOM-18 | RestModel implements JSON:API interface | PASS | `jukebox/rest.go:9-20` — `GetID()`, `GetName()`, `SetID()` |
| DOM-19 | Request models flat | N/A | No `CreateRequest`/`UpdateRequest` — GET-only client |
| DOM-31 | Tenant/trace via context only | PASS | `jukebox/rest.go` `RestModel` carries no tenant/trace field; tenant travels via `ctx` passed to `NewProcessor` |
| EXT-01 | Target RestModel implements SetToOneReferenceID/SetToManyReferenceIDs | PASS | `jukebox/rest.go:26-34` — both present as documented no-ops, citing EXT-01 |
| EXT-02 | httptest-backed integration test, populated struct assertion | PASS | `jukebox/requests_test.go:30-47` (`TestGetActiveDecodesTheJukeboxResource`) serves a representative JSON:API fixture via `httptest.NewServer` and asserts `m.ItemId`/`m.PlayerName` are populated |
| EXT-03 | Only genuine 404 maps to "not found"; other failures bubble | PASS | `jukebox/processor.go:27-29` does no `ErrNotFound`/error-reclassification at all — `GetActive` returns whatever `requests.Provider` yields verbatim, so no error class is ever mis-mapped |
| EXT-04 | URL composed via domain-parameterized resolver, no hardcoded DNS | PASS | `jukebox/requests.go:16-18` — `requests.RootUrlFor(ctx, "MAPS")`; see "Note on this re-run" above for the RootUrl→RootUrlFor migration evidence |
| DOM-25 | No client-interpreted wire byte as a Go literal outside libs/atlas-packet | N/A | This package writes no wire bytes — the client-facing jukebox packet write lives in `kafka/consumer/map/consumer.go` (graded separately below) |
| DOM-26 | Goroutines via routine.Go | N/A | No `go` statement in this package; `tools/goroutine-guard.sh` exit 0 |
| DOM-29 | Cache scope | N/A | No cache/singleton state in this package |
| DOM-30 | AndEmit/message.Buffer for DB writes | N/A | Package performs no DB write and emits no Kafka message |

### services/atlas-channel/atlas.com/channel/kafka/consumer/map (support — existing package, jukebox handlers added)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-25 | Client-interpreted wire byte resolved from tenant table, not literal | PASS (judgment call — see note) | `jukeboxStopItemId = int32(-1)` (`consumer.go:1102`) is written into the outbound `PlayJukebox` body as an itemId sentinel. This is **not** the category DOM-25 targets: the anti-pattern (`anti-patterns.md:136-165`, `docs/packets/DISPATCHER_FAMILY.md:103-124`) covers dispatcher mode bytes / sub-op codes / message types / notice-fail-reason codes — client-side **lookup-switch** values that vary across tenant client versions and require per-version writer-options tables (the worked examples are `NoticeFailReason`, `msgType`). `m_nJukeBoxItemID == -1` is a single-value equality sentinel baked into the client binary identically across every version examined (`consumer.go:1096-1101`, citing GMS v95.0 @0x61dab0), not a multi-code lookup table selecting between tenant-configurable behaviors — there is nothing to seed into `services/atlas-configurations/seed-data/templates/`. Graded as outside DOM-25's documented scope, not exempted by "it's version-stable" (which the rule explicitly rejects as an argument). |
| DOM-26 | Goroutine via routine.Go | PASS | `consumer.go:381` — `routine.Go(l, ctx, func(_ context.Context) { announceActiveJukebox(...) })` |
| DOM-30 | AndEmit/message.Buffer for DB writes | N/A | `handleStatusEventJukeboxStart`/`End` write to the in-memory (weather/jukebox) client-side state only via broadcast, no DB write; documented exception `patterns-kafka.md:83` ("Operations over non-DB state") covers producer calls originating from `atlas-maps` jukebox processor (below) |
| DOM-20 | Table-driven tests | WARN (already logged, not new) | `consumer_test.go:18-107` — three near-identical `TestHandleStatusEventJukebox*` scenarios are a table-driven candidate; carried over unchanged from the prior pass, race-fixed by `22db8d5fa` without altering the assertion shape |

### services/atlas-channel/atlas.com/channel/socket/handler (support — existing package, jukebox arm added to character_cash_item_use.go)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-25 | No client-interpreted wire byte as Go literal | PASS | `CashSlotItemType(20)` (`character_cash_item_use.go:1088-1089`) classifies an **inbound** client byte (the item's own cash-slot classification, computed server-side from `item.Classification`), not an outbound dispatch/reason code the rule targets; comment cites IDA evidence for the 510→20 mapping being stable across every version examined |
| DOM-26 | Goroutine via routine.Go | N/A | No bare `go` statement added in the jukebox arm |
| DOM-24 | producertest / injected no-op producer for emit-reaching tests | PASS | `character_cash_item_use_jukebox_test.go:45-58` calls `installCapturingProducer()` (`cash_item_gachapon_test.go:50`, an existing package helper) before exercising `CharacterCashItemUseHandleFunc`, which reaches `saga.NewProcessor(...).Create` |
| DOM-20 | Table-driven tests | WARN (already logged, not new) | `character_cash_item_use_jukebox_test.go:127-213` — three near-identical reject-path tests (`TestJukeboxArmRejectsSlotTemplateMismatch`/`RejectsZeroSoundLength`/`RejectsUnresolvableCharacter`) are a table-driven candidate |
| DOM-21 | No redeclared shared constant | PASS | `CashSlotItemType` type predates this diff (`character_cash_item_use.go:1035`); only a new value is added to the existing const block — no equivalent classification exists in `libs/atlas-constants` for the cash-slot-type mapping (distinct from `item.Classification`) |

### libs/atlas-packet/cash/serverbound (item_use_song_player.go, item_use_song_player_test.go)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-25 | Packet codec confined to libs/atlas-packet | PASS | `item_use_song_player.go` Encode/Decode are entirely self-contained; carries `packet-audit:fname CWvsContext::SendConsumeCashItemUseRequest` (line 34) with IDA citations for both GMS v95.0 and v83 |
| DOM-20 | Table-driven / playbook-governed tests | N/A (playbook governs) | Governed by `docs/packets/audits/VERIFYING_A_PACKET.md`'s per-version byte-fixture playbook per DOM-20's own documented carve-out (`testing-guide.md:322`) — not evaluated against the generic table-driven shape |

### services/atlas-maps/atlas.com/maps/map/jukebox (sub-domain-adjacent — has resource.go/rest.go/processor.go, no model.go)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor in processor.go | PASS | `processor.go:12-27` — interface, `NewProcessor(l logrus.FieldLogger, ctx context.Context)`, `ProcessorImpl` methods |
| FILE-02 | RestModel/Transform in rest.go | PASS | `rest.go:5-29` — `RestModel`, `GetID`/`GetName`/`SetID`, `func Transform(e JukeboxEntry) (RestModel, error)` all in `rest.go` |
| FILE-03 | Cross-service requests in requests.go | N/A | Package makes no outbound cross-service call; no `requests.go` |
| FILE-04 | entity.go conventions | N/A | No `entity.go` — Registry is in-memory, not DB-backed |
| FILE-05 | Builder/Model/administrator/provider/state placement | N/A | No `builder.go`/`model.go`/`administrator.go`/`provider.go`/`state.go`; `JukeboxEntry`/`FieldKey` are cache-entry/key types co-located with the singleton cache logic in `registry.go`, matching the canonical `cache.go` example's own co-location of `cacheEntry` with the `Cache` struct (`patterns-cache.md:31-47`) — not the DOM "domain Model" this rule targets |
| FILE-06 | No catch-all file | PASS | `registry.go` combines exactly one responsibility (cache: entry type + singleton), matching the doc's own reference shape; no file combines ≥2 of the FILE-01..06 responsibilities |
| DOM-04 | `Transform(Model) (RestModel, error)` | PASS | `rest.go:24-29` |
| DOM-05 | TransformSlice used by list handlers | N/A | `resource.go` has one GET-by-instance handler returning a single entry, no list endpoint exists to require `TransformSlice` |
| DOM-06 | Processor ctor takes FieldLogger | PASS | `processor.go:23` |
| DOM-07 | Handler passes `d.Logger()` to NewProcessor | PASS | `resource.go:37` — `jp := NewProcessor(d.Logger(), d.Context())` |
| DOM-08 | POST/PATCH via RegisterInputHandler | N/A | Only a GET route registered (`resource.go:26`) |
| DOM-09 | Transform() error checked | PASS | `resource.go:44-49` checks `err` and writes an error response before marshaling |
| DOM-12 | No os.Getenv in resource.go | PASS | `grep os.Getenv resource.go` — zero matches |
| DOM-13 | No cross-domain orchestration in handler | PASS | `handleGetJukeboxInMap` calls only `jukebox.NewProcessor(...).GetActive` |
| DOM-14 | Handler calls processor, not provider | PASS | Same call site; no provider function called from `resource.go` |
| DOM-15 | No db.Create/Save/Delete in resource.go | PASS | Zero matches; package is not DB-backed |
| DOM-17 | Error status mapping | PASS | `resource.go:39-42` — not-found → 404; no validation/conflict branch exists in this GET-only handler to grade |
| DOM-18 | RestModel implements JSON:API interface | PASS | `rest.go:11-22` |
| DOM-19 | Request models flat | N/A | GET-only — no request models |
| DOM-32 | Routes register via server.RegisterHandler/RegisterInputHandler | PASS | `resource.go:26` calls `rest.RegisterHandler` which is `var RegisterHandler = server.RegisterHandler` (`atlas-maps/rest/handler.go:28`, pre-existing alias, not touched by this diff) |
| DOM-29 | Cache is application-scoped singleton via accessor | PASS | `registry.go:27-38` — package-level `registry`/`once`, `getRegistry()` accessor with `sync.Once`; `registry.go:22-24` guards reads/writes with `sync.RWMutex`; `ProcessorImpl` (`processor.go:16-25`) holds no cached state itself, only calls `getRegistry()` |
| DOM-30 | AndEmit/message.Buffer for DB writes | N/A (documented exception) | `Start`/`GetActive` operate over the in-memory Registry, not a DB — `patterns-kafka.md:83-84` ("Operations over non-DB state... has no transaction on any path") covers `jukebox.NewProcessor(...).Start` in `kafka/consumer/map/consumer.go:94` and `tasks/jukebox.go:45` which call `producer.ProviderImpl` directly, not `AndEmit` |
| DOM-31 | Tenant via context only | PASS | `processor.go:30,42` reads `tenant.MustFromContext(p.ctx)`; `rest.go` `RestModel` carries no tenant field |
| DOM-20 | Table-driven tests | PASS | `registry_test.go` (4 funcs) and — see tasks/jukebox_test.go below — each test exercises a genuinely distinct scenario (start-then-get, replace-active-entry, expired-filtering, tenant-isolation), not a parameterizable variant of the same case; matches the shape of `testing-guide.md`'s own single-scenario canonical `Example` (`testing-guide.md:23-29`) |

### services/atlas-maps/atlas.com/maps/kafka/consumer/map, tasks (existing packages, jukebox command handled / sweep task added)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-24 | producertest / injected no-op for emit-reaching tests | PASS | `kafka/consumer/map/testmain_test.go:7-11` — `producertest.InstallNoop()` at package `TestMain`; `tasks/jukebox_test.go` injects its own `noopEmit`/spy `emit` function into `processExpiredJukebox`, never reaching the real producer |
| DOM-30 | AndEmit/message.Buffer for DB writes | N/A (documented exception) | `handlePlayJukeboxCommand` (`consumer.go:78-100`) and `emitJukeboxEnd` (`tasks/jukebox.go:40-46`) both call `producer.ProviderImpl` directly against the in-memory Registry, matching the same non-DB-state exception as the atlas-maps jukebox package above |
| DOM-21 | No redeclared shared constant | PASS | `maxJukeboxDuration = 10 * time.Minute` (`consumer.go:76`) is a package-local cap, not a value tracked in `libs/atlas-constants` |
| DOM-20 | Table-driven tests | PASS | `tasks/jukebox_test.go` — two distinct-scenario tests (env-context threading, entry deletion), not near-identical candidates for consolidation |

### services/atlas-saga-orchestrator/map_command, saga (existing files extended)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-33 | Mock kept in sync with `map_command.Processor` | PASS | `map_command/processor.go:15` adds `PlayJukebox` to `Processor`; `grep -rln "map_command.Processor" services/atlas-saga-orchestrator/` found only `saga/handler.go` (the real caller) — no mock implements this interface, so nothing can go stale |
| DOM-33 | `saga.Handler` interface change | N/A | `Handler` is package-private, implemented only by `HandlerImpl` in the same file (`saga/handler.go:182,1005-1006`) — not mockable, no mock exists |
| DOM-20 | Table-driven tests | PASS | `TestPlayJukeboxCommandProvider` (`map_command/producer_test.go:19`) and `TestHandlePlayJukebox_InvalidPayload` (`saga/handler_test.go:1676`) are single-scenario tests, matching the doc's own canonical `Example` (`testing-guide.md:23-29`) and sibling `handleFieldEffectWeather` tests |
| DOM-31 | Tenant/trace via context | PASS | `handlePlayJukebox` (`saga/handler.go:3610-3636`) reads no tenant/trace field off the payload; uses `h.l`/`h.ctx` and the saga's own transaction id only |
| DOM-24 | producertest for emit-reaching test | N/A | `TestPlayJukeboxCommandProvider` calls the pure `model.Provider` function directly (no Kafka client); `TestHandlePlayJukebox_InvalidPayload` returns before reaching `h.mapCommandP.PlayJukebox` |

### libs/atlas-saga (model.go, payloads.go, unmarshal.go)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-21 | No constant/type redeclared vs. existing | PASS | `PlayJukebox`/`PlayJukeboxPayload` are new symbols; `unmarshal_test.go` round-trips them; `world_transfer_test.go` adds `PlayJukebox` to the action-uniqueness assertion |
| DOM-34/35 | Migration hygiene | N/A | Nothing moved between a service and this library — symbols are added directly here, referenced through the pre-existing type-alias convention in each service's own `saga/model.go` |

### tools/packet-audit (internal/matrix/build.go, cmd/run.go)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01..06 | File responsibilities | N/A | This is a CLI/reporting tool package, not a service domain/sub-domain package — none of `model.go`/`rest.go`/`entity.go`/`processor.go`/`requests.go` conventions apply; no such symbols exist here to misplace |
| DOM-26 | Goroutines via routine.Go | N/A | `grep -n "^\s*go " tools/packet-audit/cmd/run.go tools/packet-audit/internal/matrix/build.go` — zero matches; `tools/goroutine-guard.sh` exit 0 |
| DOM-21 | Constants reuse | N/A | `legacyConsumedSiblingWriters` allow-list entry and the `candidatesFromFName` fixture registration are packet-audit's own internal bookkeeping, not domain constants with a `libs/atlas-constants` equivalent |

## Security Review

Not applicable — SEC-* trigger did not fire (no token/auth/redirect/secret surface touched by this diff).

## Not evaluable from the diff

- none

## Summary

### Blocking (must fix)
- FILE-02: `services/atlas-channel/atlas.com/channel/jukebox/processor.go:31-33` — `Extract(m RestModel) (RestModel, error)` belongs in `rest.go` per file-responsibilities.md's `rest.go` definition ("Implement Transform/Extract...") and per every sibling atlas-channel client package (20+ confirmed instances all place `Extract` in `rest.go`). Move it to `jukebox/rest.go`.

### Non-Blocking (should fix)
- DOM-20 (WARN, already logged in a prior pass, unchanged by this diff's fixes): `services/atlas-maps/atlas.com/maps/kafka/consumer/map/consumer_test.go:18-107` — three near-identical `TestHandlePlayJukeboxCommand_*` tests are a table-driven candidate.
- DOM-20 (WARN, already logged in a prior pass, unchanged by this diff's fixes): `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_jukebox_test.go:127-213` — three near-identical reject-path tests are a table-driven candidate.

### Already-logged, not re-litigated
- `services/atlas-maps/atlas.com/maps/map/jukebox/registry.go` — `Get`/`GetActive` do not filter on `ExpiresAt` (inherited verbatim from the existing weather registry pattern). No numbered audit rule covers this functional-correctness gap.

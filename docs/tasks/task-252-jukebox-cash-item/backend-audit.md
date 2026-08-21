# Backend Audit — task-252-jukebox-cash-item

- **Service Path:** services/atlas-maps, services/atlas-channel, services/atlas-saga-orchestrator, libs/atlas-saga, libs/atlas-packet
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-08-21
- **Build:** PASS (all 5 modules)
- **Tests:** All packages `ok` or `[no test files]` across all 5 modules; 0 failures
- **Overall:** NEEDS-WORK

## Build & Test Results

```
services/atlas-maps/atlas.com/maps            go build ./...  -> exit 0
services/atlas-channel/atlas.com/channel      go build ./...  -> exit 0
services/atlas-saga-orchestrator/.../saga-orchestrator  go build ./...  -> exit 0
libs/atlas-saga                               go build ./...  -> exit 0
libs/atlas-packet                             go build ./...  -> exit 0

services/atlas-maps/atlas.com/maps            go test ./... -count=1  -> all ok
services/atlas-channel/atlas.com/channel      go test ./... -count=1  -> all ok
services/atlas-saga-orchestrator/.../saga-orchestrator  go test ./... -count=1  -> all ok
libs/atlas-saga                               go test ./... -count=1  -> ok
libs/atlas-packet                             go test ./... -count=1  -> all ok
```

## Applicability

| Family | Fired? | Evidence |
|---|---|---|
| FILE (01-06) | Fired | Every changed Go package audited (unconditional). |
| DOM structure (01-05,11,16) | Fired (narrowed) | `atlas-maps/map/jukebox` has `rest.go` (fires DOM-04/05 by their own `Applies when`); no changed package has `model.go`/`entity.go`/`provider.go`, so DOM-01/02/03/11/16 are N/A on their own triggers. |
| REST (06-09,12-15,17-19,32) | Fired | `atlas-maps/map/jukebox` has `resource.go`+`rest.go`+`processor.go` and registers a route. `atlas-channel/jukebox` has `rest.go`+`processor.go` (no HTTP routes registered there). |
| Constants reuse (DOM-21) | Fired | Diff declares new types/consts (`JukeboxEntry`, `PlayJukeboxPayload`, `CashSlotItemTypeSongPlayer`, `jukeboxStopItemId`, event/command type strings). |
| Testing (DOM-10,20,24,33) | Fired | Diff touches many `_test.go` files, adds a method to two `Processor` interfaces, and touches emit paths. |
| Cache (DOM-29) | Fired | `atlas-maps/map/jukebox/registry.go` is a package-level singleton (`sync.Once`/`sync.RWMutex`) holding cached state. |
| Messaging (DOM-30) | Fired | `atlas-maps/map/jukebox/producer.go` + direct `producer.ProviderImpl(...)` call sites in `tasks/jukebox.go` and `kafka/consumer/map/consumer.go`. |
| Multi-tenancy (DOM-31) | Fired | Packages with `rest.go` (`atlas-maps/map/jukebox`, `atlas-channel/jukebox`); `tenant.MustFromContext` used in `processor.go`. |
| Migration hygiene (DOM-34,35) | N/A | No symbol moved from a service into a `libs/atlas-*` module or vice versa — `PlayJukebox`/`PlayJukeboxPayload` are new symbols added directly to `libs/atlas-saga`, then aliased into both services via the pre-existing, repo-wide alias convention (not a migration). |
| Deploy & topics (DOM-22,23) | Opened, both N/A | No new `libs/atlas-*` module (`atlas-saga`, `atlas-packet` already existed); no new/renamed topic env var — `EnvCommandTopicMap` and `EnvEventTopicMapStatus` are reused, not added. |
| Runtime safety (DOM-26) | Fired | Every non-test changed `.go` file swept for bare `go` statements — none found; new goroutine in `consumer.go` uses `routine.Go(l, ctx, ...)`. |
| Channel wire values (DOM-25) | Fired | Diff touches `services/atlas-channel` and `libs/atlas-packet` (`PlayJukeboxWriter`, `ItemUseSongPlayer`). |
| Resilience (DOM-27,28) | N/A | No changed handler writes `http.StatusInternalServerError`; no `model.Decorator`/enrichment path touched. Not in the family list supplied for this diff. |
| External clients (EXT-01..04) | Fired | `atlas-channel/jukebox` calls atlas-maps via `requests.GetRequest[RestModel]`. |
| Scaffolding (SCAFFOLD-01..09) | N/A | No `services/atlas-<svc>/` directory added; no `Writer`/`Handler` constant registered in `services/atlas-channel/atlas.com/channel/main.go` (that file is untouched in this diff); `deploy/shared/routes.conf` untouched. |
| Security (SEC-01..04) | N/A | No token/auth/redirect/secret surface touched. |

## Checklist Results

### atlas-maps/map/jukebox (domain-shaped support package — cache-backed, no `model.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor in processor.go | PASS | `map/jukebox/processor.go:13-45` — interface, constructor, `ProcessorImpl` methods all in one file. |
| FILE-02 | RestModel/Transform in rest.go | PASS | `map/jukebox/rest.go:5-30`. |
| FILE-04 | entity.go responsibilities | N/A | No `entity.go` — package has no DB-backed persistence (in-memory registry). |
| FILE-05/06 | No catch-all file bundling ≥2 responsibilities | PASS | `registry.go` holds only the cache-shaped singleton (`JukeboxEntry`, `FieldKey`, `Registry`) — this matches the `cache.go` file-responsibility shape (singleton, `sync.Once`, `sync.RWMutex`), not the `model.go` shape (no builder, no accessors, no persisted `Model`). No file mixes Processor/RestModel/requests/entity. |
| DOM-04 | `Transform(Model)(RestModel,error)` in rest.go | PASS | `map/jukebox/rest.go:24`. |
| DOM-05 | `TransformSlice` + list handlers | N/A | No list handler exists in `resource.go` (only a single-entry "get active" GET) — nothing to bulk-transform. |
| DOM-06 | Processor ctor takes `logrus.FieldLogger` | PASS | `processor.go:23`. |
| DOM-07 | Handler passes `d.Logger()` | PASS | `resource.go:37`. |
| DOM-08 | POST/PATCH via `RegisterInputHandler[T]` | N/A | Package registers a GET route only (`resource.go:26`). |
| DOM-09 | Every `Transform(` call checked | PASS | `resource.go:44-49` checks `err`. |
| DOM-12 | No `os.Getenv` in resource.go | PASS | Zero matches. |
| DOM-13 | No cross-domain orchestration in handler | PASS | Handler calls only `jp.GetActive(f)` (`resource.go:38`). |
| DOM-14 | Handler calls processor only, not provider | PASS | No provider call sites in `resource.go`. |
| DOM-15 | No `db.Create/Save/Delete` in handler | PASS | Zero matches (no gorm usage in this package). |
| DOM-17 | not-found -> 404 | PASS | `resource.go:39-42`. |
| DOM-18 | RestModel implements JSON:API interface | PASS | `rest.go:11-22`. |
| DOM-19 | Request models flat | N/A | No request models (GET only). |
| DOM-29 | Cache is a scoped singleton, not per-processor state | PASS | `registry.go:27-38` — package-level `registry`/`sync.Once`; `ProcessorImpl` (`processor.go:18-25`) holds no cache field, calls `getRegistry()` each time. |
| DOM-30 | DB write emits via AndEmit; documented exception for non-DB state | PASS (documented exception) | No DB write anywhere in this package (in-memory registry only); direct `producer.ProviderImpl(...)` calls at `kafka/consumer/map/consumer.go:96` and `tasks/jukebox.go:45` are on the success path of operations over non-DB registry state — the exact exception in `patterns-kafka.md` (`services/atlas-chairs` precedent). |
| DOM-31 | Tenant travels via context only | PASS | `processor.go:30,42` use `tenant.MustFromContext(p.ctx)`; `RestModel` carries no tenant field. |
| DOM-32 | Routes via `server.RegisterHandler`/`RegisterInputHandler[T]` | PASS | `resource.go:26` uses `rest.RegisterHandler` which is `= server.RegisterHandler` (`services/atlas-maps/atlas.com/maps/rest/handler.go:28`, pre-existing alias). No hand-rolled error writer; uses `server.WriteErrorResponse` (`resource.go:47`). |
| DOM-20 | Table-driven tests | PASS | `registry_test.go` and `tasks/jukebox_test.go` each cover distinct behaviors (not repeated near-identical variants), matching the doc's own single-scenario example (`testing-guide.md:22-27`). |
| DOM-20 | Table-driven tests | WARN | `kafka/consumer/map/consumer_test.go:18-107` — three near-identical `TestHandlePlayJukeboxCommand_*` tests differ only in `DurationMs`/`Type` and the expected outcome; a textbook table-driven candidate left un-tabled. |
| DOM-24 | Emit-reaching test package stubs the producer | PASS | `kafka/consumer/map/testmain_test.go:10-13` calls `producertest.InstallNoop()` in `TestMain`; `TestHandlePlayJukeboxCommand_*` reaches `producer.ProviderImpl` transitively through `handlePlayJukeboxCommand`. |

### atlas-channel/jukebox (EXT client package, no `model.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01/02/03 | processor.go/rest.go/requests.go split | PASS | `jukebox/processor.go`, `jukebox/rest.go`, `jukebox/requests.go` each single-purpose. |
| FILE-06 | No catch-all file | PASS | No file bundles ≥2 responsibilities. |
| DOM-06 | Processor ctor takes `logrus.FieldLogger` | PASS | `processor.go:21`. |
| DOM-18 | RestModel implements JSON:API interface | PASS | `rest.go:9-19`. |
| DOM-31 | Tenant via context only | PASS | No tenant field on `RestModel`; tenant travels via `ctx` into `requests.RootUrlFor`/`requests.Provider`. |
| DOM-33 | Mock kept in sync with interface | PASS | `jukebox/mock/processor.go:9-19` implements the newly-added `Processor.GetActive` with matching signature and nil-check default, in the same diff. |
| EXT-01 | Target RestModel implements `SetToOneReferenceID`/`SetToManyReferenceIDs` | **FAIL** (Important) | `services/atlas-channel/atlas.com/channel/jukebox/rest.go:1-20` — `RestModel` defines only `GetID`/`GetName`/`SetID`; no `SetToOneReferenceID`/`SetToManyReferenceIDs`, even as no-ops. Per `libs/atlas-rest/CLAUDE.md:24-25` and `cross-service-implementation.md`, api2go errors on any response carrying a `relationships` block without these methods. |
| EXT-02 | httptest-backed integration test with populated-struct assertion | PASS | `jukebox/requests_test.go:30-47` (`TestGetActiveDecodesTheJukeboxResource`) — `httptest.NewServer` serves a representative JSON:API fixture and asserts `m.ItemId`/`m.PlayerName`. |
| EXT-03 | Only genuine 404s map to "not found"; other errors bubble | PASS | No custom error-mapping code exists in `processor.go`/`requests.go` — the raw error from `requests.Provider`/`requests.GetRequest` is returned unchanged in all cases, so nothing re-labels a transport/5xx failure as "not found". |
| EXT-04 | URL via `requests.RootUrl`/`RootUrlFor` | PASS | `jukebox/requests.go:17` — `requests.RootUrlFor(ctx, "MAPS")`. |
| DOM-20 | Table-driven tests | PASS | `requests_test.go` — three tests cover distinct concerns (decode, path shape, 404 handling), not repeated variants. |

### services/atlas-channel/kafka/consumer/map (support package, existing file extended)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-26 | Goroutines via `routine.Go` | PASS | `consumer.go:381-383` — `routine.Go(l, ctx, func(_ context.Context) { announceActiveJukebox(...) })`. |
| DOM-25 | Client wire values resolved, not literal | PASS | `jukeboxStopItemId = int32(-1)` (`consumer.go:1098`) is a documented, IDA-verified item-id sentinel (not a dispatcher/sub-op/fail-reason code that flows through a lookup switch); `fieldcb.PlayJukeboxWriter`/`fieldcb.NewPlayJukebox` encode/decode logic lives entirely in `libs/atlas-packet/field/clientbound/play_jukebox.go`. |
| DOM-31 | Tenant via context | PASS | `handleStatusEventJukeboxStart`/`End` use `tenant.MustFromContext(ctx)` (`consumer.go:1101,1121`), no tenant field on the event body beyond the pre-existing `StatusEvent[E]` envelope. |

### services/atlas-channel/socket/handler (character_cash_item_use.go — extended, not new)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-25 | No client wire code as Go literal outside `libs/atlas-packet` | PASS | `CashSlotItemTypeSongPlayer = CashSlotItemType(20)` (`character_cash_item_use.go:1092`) is documented and IDA-cited, following the exact pattern of every sibling `CashSlotItemType*` constant in the same file (pre-existing convention, not introduced ad hoc by this diff). |
| DOM-20 | Table-driven tests | PASS | `TestJukeboxArmSuccessCreatesTwoStepSaga` — distinct, deep single-scenario assertion, matches sibling arm tests in the same file. |
| DOM-20 | Table-driven tests | WARN | `character_cash_item_use_jukebox_test.go:127-213` — `TestJukeboxArmRejectsSlotTemplateMismatch`, `TestJukeboxArmRejectsZeroSoundLength`, `TestJukeboxArmRejectsUnresolvableCharacter` share an almost identical "assert nothing emitted" body and differ only in the rejection trigger; a reasonable table-driven candidate left un-tabled. |

### libs/atlas-packet/field/clientbound (play_jukebox.go — pre-existing, unchanged by this diff) / libs/atlas-packet/cash/serverbound (new)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-25 | Packet codec confined to `libs/atlas-packet` | PASS | `item_use_song_player.go` Encode/Decode live entirely inside the codec file; carries `packet-audit:fname` annotation (line 13) and IDA citations. |
| DOM-20 | Table-driven / playbook-governed tests | PASS | `item_use_song_player_test.go:45-70` (`TestItemUseSongPlayerWireOrder`) uses `tests := []struct{...}` + `t.Run`; the two round-trip tests iterate `test.Variants` with `t.Run`, governed by the packet-fixture playbook (`docs/packets/audits/VERIFYING_A_PACKET.md`) per DOM-20's own documented carve-out. |

### services/atlas-saga-orchestrator/map_command, saga (existing files extended)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-33 | Mock kept in sync with `map_command.Processor` | PASS | `map_command/processor.go:15-16` adds `PlayJukebox` to `Processor`; grep for `type.*Mock struct` under `saga-orchestrator/` found no mock implementing `map_command.Processor` — nothing to go stale. |
| DOM-33 | `saga.Handler` interface change | N/A | `Handler` is an unexported interface whose only implementation is `HandlerImpl` in the same file (`handler.go:182,1005-1006`) — not externally mockable, no mock exists. |
| DOM-20 | Table-driven tests | PASS | `TestPlayJukeboxCommandProvider` (`map_command/producer_test.go:19`) and `TestHandlePlayJukebox_InvalidPayload` (`saga/handler_test.go:1676`) are single-scenario tests matching sibling `handleFieldEffectWeather`/`FieldEffectWeather` tests and the doc's own canonical example. |
| DOM-31 | Tenant/trace via context | PASS | `handlePlayJukebox` reads no tenant/trace field off the payload; uses `h.ctx`/`h.l` only (`handler.go:3610-3636`). |

### libs/atlas-saga (model.go, payloads.go, unmarshal.go)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-21 | No constant/type redeclared vs. existing | PASS | `PlayJukebox`/`PlayJukeboxPayload` are new symbols; `unmarshal_test.go:1384-1401` round-trips them; `world_transfer_test.go:127` adds `PlayJukebox` to the uniqueness assertion. |
| DOM-34/35 | Migration hygiene | N/A | Nothing moved between a service and this library — the symbols are added directly here and referenced through the pre-existing repo-wide type-alias convention in each service's own `saga/model.go` (not a migration this diff performs). |

## Security Review

Not applicable — SEC-* trigger did not fire (no token/auth/redirect/secret surface touched by this diff).

## Not evaluable from the diff

- DOM-17 (validation -> 400 / conflict -> 409) for `atlas-maps/map/jukebox`: the only error branch in `resource.go` is not-found -> 404; no validation or conflict path exists in this handler to grade, and the packages that would exercise 400/409 (e.g. `field.NewBuilder` validation) were not read beyond what the diff itself shows — the change surface has no such branch to evaluate.
- DOM-27/28 (resilience): the given family list for this diff does not include Resilience, and the changed handlers do not write `http.StatusInternalServerError`; whether `atlas-maps` as a whole is "DB-backed" in the sense DOM-27's trigger means was not independently re-derived beyond what the diff shows (no DB code touched here).
- Whether the atlas-maps jukebox `Registry.GetActive`/`Get` ExpiresAt-filtering gap has any live operational impact beyond the already-logged deferred item was not further investigated — out of scope per the task's own note.

## Summary

### Blocking (must fix)
- EXT-01: `services/atlas-channel/atlas.com/channel/jukebox/rest.go` — `RestModel` is missing `SetToOneReferenceID`/`SetToManyReferenceIDs` (even as no-ops); api2go will error on any atlas-maps jukebox response that carries a `relationships` block.

### Non-Blocking (should fix)
- DOM-20 (WARN): `services/atlas-maps/atlas.com/maps/kafka/consumer/map/consumer_test.go:18-107` — three near-identical `TestHandlePlayJukeboxCommand_*` tests are a table-driven candidate.
- DOM-20 (WARN): `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_jukebox_test.go:127-213` — three near-identical reject-path tests are a table-driven candidate.

### Already-logged, not re-litigated
- `services/atlas-maps/atlas.com/maps/map/jukebox/registry.go` — `Get`/`GetActive` do not filter on `ExpiresAt` (inherited verbatim from the existing weather registry pattern). No numbered audit rule covers this functional-correctness gap; recorded here only because the task instructions asked it be reported if a rule genuinely covers it — none does.

# Backend Audit — task-145-player-reports (whole workstream)

- **Audit Scope:** hand-written Go changes in range `9ecf9c9759a9db059fb40f497e7c6a04e43f06fd..6a9e91e561dbd7138274e636ec9fa545f304cd2f` (76 commits; bulk is generated packet-audit data, excluded). In scope: `libs/atlas-packet/report/`, `libs/atlas-packet/field/serverbound/sue_character.go`, `libs/atlas-redis` (keyed sorted set), `services/atlas-ban/atlas.com/ban/{report,chat,character,kafka/consumer/report,kafka/message/report,main.go,rest/handler.go}`, `services/atlas-messages/atlas.com/messages/{chat,message,rest/handler.go,main.go}`, `services/atlas-channel/atlas.com/channel/{socket/handler,socket/writer,kafka/consumer/report,kafka/message/report,report,main.go}`, `tools/packet-audit/` (tooling, not a microservice — DOM/FILE checklist N/A, verified build/vet/test only).
- **Guidelines Source:** `.claude/skills/backend-dev-guidelines/resources/*.md` (DOM-\*/SUB-\*/EXT-\*/FILE-\* checklists)
- **Date:** 2026-08-05
- **Build:** PASS (`atlas-ban`, `atlas-messages`, `atlas-channel`, `libs/atlas-packet`, `tools/packet-audit`)
- **Tests:** all packages `ok` under `go test -race ./... -count=1` in every changed module (0 failed)
- **Vet / Guards:** `go vet ./...` clean in all 5 modules; `tools/goroutine-guard.sh` exit 0; `tools/redis-key-guard.sh` exit 0
- **Overall:** NEEDS-WORK (7 FAIL findings, all Important; see Summary)

## Build & Test Results

```
$ cd services/atlas-ban/atlas.com/ban && go build ./... && go vet ./...          # clean
$ cd services/atlas-messages/atlas.com/messages && go build ./... && go vet ./...  # clean
$ cd services/atlas-channel/atlas.com/channel && go build ./... && go vet ./...    # clean
$ cd libs/atlas-packet && go build ./... && go vet ./...                           # clean
$ cd tools/packet-audit && go build ./... && go vet ./...                          # clean

$ go test -race ./... -count=1   (per module above)
atlas-ban:        ok x11 packages (report, chat, kafka/consumer/report, history, ban, ...), 0 failed
atlas-messages:    ok x14 packages (chat, message, map, pet, skill, saga, ...), 0 failed
atlas-channel:     ok x30+ packages (report, socket/handler, socket/writer, session, ...), 0 failed
libs/atlas-packet:  ok x40+ packages incl. report/clientbound, report/serverbound, 0 failed
tools/packet-audit: ok x13 packages incl. cmd, idasrc, matrix, 0 failed

$ tools/goroutine-guard.sh   # exit 0
$ tools/redis-key-guard.sh   # exit 0
```

## Domain Checklist Results

### `atlas-ban/report` (domain package — has `model.go`, `entity.go`, `builder.go`, `administrator.go`, `provider.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | `builder.go` exists | PASS | `report/builder.go:26` `NewBuilder`, fluent setters, `Build()` validates `Kind`/`Status` (lines 51-56) |
| DOM-02 | `ToEntity()` method | **FAIL** | Grepped `report/*.go` for `func (m Model) ToEntity()` / `func (m *Model) ToEntity()` — zero matches. `file-responsibilities.md:14` documents `entity.go` as providing both `Make(Entity) (Model, error)` **and** `Model.ToEntity()`; only `Make` exists (`report/administrator.go:57`). Concrete gap: `report/administrator.go:10-42`'s `create()` builds an `Entity{}` literal by hand from 10 primitive args instead of `model.ToEntity()`, so any future write path that already holds a `Model` (e.g. a bulk re-import, a future `Update` beyond status) has no supported conversion and will either duplicate the literal or reach into private fields. |
| DOM-03 | `Make(Entity)` function | PASS | `report/administrator.go:57-77`, returns `(Model, error)` |
| DOM-04 | `Transform` function | PASS | `report/rest.go:42-58` |
| DOM-05 | `TransformSlice` function | **FAIL** | Grepped `report/rest.go` for `func TransformSlice(` — absent. `handleGetReports` (`report/resource.go:54`) instead inlines `model.SliceMap(Transform)(model.FixedProvider(reports))(model.ParallelMap())()` directly in the handler. `ai-guidance.md:211` and `patterns-rest-jsonapi.md:262-274` document `TransformSlice([]Model) ([]RestModel, error)` as the required list-transform function so list handlers don't hand-roll the composition. |
| DOM-06 | Processor accepts `FieldLogger` | PASS | `report/processor.go:44` `func NewProcessor(l logrus.FieldLogger, ...)` |
| DOM-07 | Handlers pass `d.Logger()` | PASS | `report/resource.go:44,46,70,100` all `NewProcessor(d.Logger(), d.Context(), d.DB())` |
| DOM-08 | POST/PATCH use `RegisterInputHandler` | PASS | `report/resource.go:22,27` — PATCH `/reports/{reportId}` registered via `rest.RegisterInputHandler[RestModel]`. No POST endpoint exists on this resource (creation is Kafka-command-driven only), so no POST case to check. |
| DOM-09 | Transform errors handled | PASS | `report/resource.go:77-82` (`GetById`), `115-120` (`UpdateStatus`) both check `err != nil` after `Transform(m)`; the slice path (`resource.go:54-59`) checks the error returned by `model.SliceMap(...)()`. |
| DOM-10 | Test DB has tenant callbacks | PASS | `report/administrator_test.go:22` `database.RegisterTenantCallbacks(l, db)`; also `kafka/consumer/report/consumer_test.go` (no report.Entity DB use, N/A) and `processor_test.go` share `setupTestDatabase` (same helper). |
| DOM-11 | Providers use lazy evaluation | PASS | `report/provider.go:11-42` matches the **documented canonical pattern** in `patterns-provider.md:17-29` verbatim (string-based `db.Where("id = ?", id).First(...)` inside the closure, `model.ErrorProvider`/`model.FixedProvider`). This is NOT the DOM-11 anti-pattern despite superficial resemblance — the guideline's own worked example is this exact shape (see note below on `database.Query`/`SliceQuery` alternative, which is an equally-valid but not mandatory alternative used elsewhere in the repo). |
| DOM-12 | No `os.Getenv()` in handlers | PASS | Zero matches in `report/resource.go` |
| DOM-13 | No cross-domain logic in handlers | PASS | `resource.go` handlers call only `NewProcessor(...).<Method>()`; cross-domain orchestration (character/chat resolution) lives in `report/processor.go:82-153`. |
| DOM-14 | Handlers don't call providers directly | PASS | Confirmed by reading all of `resource.go` |
| DOM-15 | No direct entity creation in handlers | PASS | Zero `db.Create`/`db.Save`/`db.Delete` in `resource.go` |
| DOM-16 | `administrator.go` exists | PASS | `report/administrator.go` — `create()` (10-42), `updateStatus()` (44-55) |
| DOM-17 | Domain error → HTTP status mapping | **FAIL** | `handleGetReportById` (`report/resource.go:70-75`) maps **every** non-nil error from `GetById` to `404 Not Found`, including a transient DB error (connection failure, timeout) that `entityById`'s `model.ErrorProvider[Entity](err)` (`provider.go:16`) would surface unfiltered. Only `handleUpdateReportStatus` (`resource.go:101-113`) correctly distinguishes `ErrInvalidStatus`→400, `gorm.ErrRecordNotFound`→404, else→500. Failure scenario: a momentary Postgres pool exhaustion on `GET /reports/{id}` returns `404`, telling the GM client "report doesn't exist" when the report is actually present and the DB is merely unavailable — actively misleading, not merely imprecise. |
| DOM-18 | JSON:API interface on REST models | PASS | `report/rest.go:25-40` `GetName()`/`GetID()`/`SetID()` |
| DOM-19 | Request models flat structure | PASS | PATCH input is `RestModel` itself (`resource.go:22`), already flat, no nested Data/Type/Attributes |
| DOM-20 | Table-driven tests | PASS | Extensive `t.Run(...)` subtests (`rest_test.go:132,161,180,198,225,241,257,279,296`); table-driven `cases := []struct{...}` pattern also used in the sibling `atlas-channel/kafka/consumer/report/consumer_test.go:31-45` |
| DOM-27 | Transient DB errors → 503, not bare 500 | **FAIL** | `report/resource.go:50,57,111,118` all use bare `w.WriteHeader(http.StatusInternalServerError)`. This is a real, available, in-service convention: `atlas-ban/main.go:57-63` **already registers** `server.RegisterTransientErrorClassifier(...)` composing `database.IsTransientConnectionError` + `database.CountTransient`, and sibling packages in the **same service** consistently call `server.WriteErrorResponse(d.Logger())(w)(err)` instead — `ban/resource.go` (10 call sites, e.g. lines 49,56,113,120,143,160,178,205) and `history/resource.go` (6 call sites, lines 49,56,75,82,104,111). `report/resource.go` is new code in this PR and does not follow the established local pattern. Failure scenario: pool exhaustion during `GET /reports` or `PATCH /reports/{id}` surfaces as a bare 500 instead of 503, so a caller/load-balancer that retries on 503 will not retry, and the `atlas_pool_exhausted_total`-class transient-error metric (`database.CountTransient`) is never incremented for these code paths despite the classifier being registered process-wide. |

**`report` package multi-tenancy check (patterns-multitenancy-context.md):** PASS. `provider.go` and `administrator.go` never construct an explicit `tenant_id = ?` WHERE clause; all DB access goes through `p.db.WithContext(p.ctx)` (`processor.go:155,169,182,186,190,194`), relying on the GORM callbacks registered by `database.RegisterTenantCallbacks`. Verified by dedicated isolation tests: `administrator_test.go:131` `TestEntityByIdFiltersByTenant`, `administrator_test.go:157` `TestEntitiesByTenantIsolatedPerTenant`.

**Kafka business-rejection semantics (explicit ask):** PASS, traced through source. `report/processor.go:78-80`'s `fail(code)` closure returns `buf.Put(...)`. `kafka/message/Buffer.Put` (`kafka/message/*.go:24-33`) returns `nil` on success (only a JSON-marshal failure inside the message provider would make it non-nil). `CreateFromCommand`'s business-rejection branches (`processor.go:85,108,111,158`) all `return fail(code)`, i.e. return the `nil` from a successful `Put`, not a propagated error. This matters because `kafka/message.Emit(p)(f)` (`kafka/message/*.go:45-59`) aborts and skips ALL buffered messages if `f(b)` returns non-nil — confirmed by reading `Emit`'s implementation, lines 48-51. So a genuine business rejection (accused not found, description too long, etc.) correctly still emits the buffered `ERROR` status event; only an actual internal `Put`/marshal failure would (correctly) abort emission entirely.

**Truncation rune-safety (explicit ask):** PASS. `report/processor.go:197-206` `truncateRunes` truncates by `[]rune` count; `processor.go:208-225` `truncateBytesAtRuneBoundary` walks byte-by-byte via `utf8.DecodeRuneInString` and only accepts a cut point that lands exactly on a rune boundary. Both are exercised by dedicated tests: `processor_test.go:289` `TestCreateFromCommandDescriptionTruncationIsRuneSafe`, `processor_test.go:316` `TestCreateFromCommandChatLogTruncationIsByteCapRuneSafe`, plus boundary tests at lines 354 and 385.

**Best-effort chat-transcript degradation (explicit ask):** PASS, scoped correctly. `report/processor.go:138-153` wraps only `p.chatP.RecentInvolving(...)` in the best-effort branch (`terr != nil` → `Warnf` + `nil` transcript). No other error class is caught by this branch — the preceding `reporter`/`accused` resolution (lines 82-112) and the following `create(...)` DB write (line 155) each have their own explicit, non-swallowing error handling that returns `fail(...)`. The degradation is narrowly scoped to exactly the one documented failure mode (messages-service unreachable), not a catch-all.

### `atlas-ban/character` (support / REST-client package — no `model.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor in `processor.go` | PASS | `character/processor.go:20-41` |
| FILE-02 | RestModel + Extract + JSON:API methods in `rest.go` | PASS | `character/rest.go:5-29` |
| FILE-03 | Request funcs in `requests.go` | PASS | `character/requests.go:15-25` |
| FILE-05 | Domain `Model` in `model.go` | Minor | `type Model struct` lives in `character/processor.go:12-15`, not a dedicated `model.go`. Low severity: this is a 2-field, getter-only struct used purely as the processor's return type in a thin REST-client package; the codebase's other thin client packages (`chat`, both new in this same PR) follow the identical layout, and no `model.go` file exists for either. Not the FILE-06 "catch-all" anti-pattern (processor.go holds exactly one responsibility — Processor — plus its own trivial return type, not Processor+RestModel+requests). |
| FILE-06 | No package-named catch-all file | PASS | No `character.go`; 3 files, each single-responsibility |
| EXT-01 | JSON:API relationship interfaces | **FAIL** | `character/rest.go` has `GetName`/`GetID`/`SetID` (lines 10-25) but **no** `SetToOneReferenceID`/`SetToManyReferenceIDs`. Grep confirms zero matches. This is directly comparable to the near-identical `atlas-fame/atlas.com/fame/character/rest.go`, which fetches the same `characters/{id}` resource and correctly implements the full stub set (`GetReferences`, `GetReferencedIDs`, `GetReferencedStructs`, `SetToOneReferenceID`, `SetToManyReferenceIDs`, `SetReferencedStructs` — all present). `libs/atlas-rest/CLAUDE.md:23-36` documents this as required "even when you don't care about relationships" because `requests.GetRequest[T]` fails the whole unmarshal if the response carries a `relationships` block and the target type lacks these methods — the exact task-037 bug class the doc cites as having recurred twice. |
| EXT-02 | httptest-backed integration test | **FAIL** | No test file exists for `character` at all (`ls character/` shows only `processor.go`, `requests.go`, `rest.go` — zero `_test.go`). No httptest coverage of the `GetRequest[RestModel]` → `Extract` pipeline. |
| EXT-03 | 404 vs. other-failure distinction | Minor / Note | `character/processor.go` never calls `errors.Is(err, requests.ErrNotFound)` itself — it passes the raw client error straight through (`processor.go:35,40`). This is not lossy by itself (the caller still gets the real error to classify), and `report/processor.go:106` **does** perform the classification for the business-critical "accused" lookup. However, the "reporter" lookup (`report/processor.go:82-86`) maps **any** `GetById` error — including a genuine 404 — to `ErrorCodeInternal`, not distinguishing a true not-found from a transport failure. Low severity because a missing reporter (the currently-connected player) is itself an anomalous condition either way, but it means a `character` service redeploy hiccup and an actually-deleted-reporter race would be indistinguishable from the reporter's error code alone. |
| EXT-04 | `RootUrl(domain)`, not hardcoded | PASS | `character/requests.go:16` `requests.RootUrl("CHARACTERS")` |

### `atlas-ban/chat` (support / REST-client package — no `model.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01/02/03/06 | File placement | PASS | Processor in `chat/processor.go:26-43`, RestModel+Extract in `chat/rest.go`, request func in `chat/requests.go:19-25`, no catch-all file |
| FILE-05 | Domain `Model` in `model.go` | Minor | Same as `character` above — `Model` struct in `chat/processor.go:12-18`, not a dedicated `model.go`. |
| EXT-01 | JSON:API relationship interfaces | **FAIL** | `chat/rest.go:4-24` has no `SetToOneReferenceID`/`SetToManyReferenceIDs`. Same task-037-class exposure as `character` above — if `atlas-messages`'s `chat-messages` resource response ever carries a `relationships` block, `requests.GetRequest[[]RestModel]` fails the unmarshal for the entire report-transcript fetch, which `report/processor.go:139-141` would then classify as the best-effort "messages unreachable" path — silently hiding a decode bug as a service-outage. |
| EXT-02 | httptest-backed integration test | **FAIL** | `chat/rest_test.go` exists but contains only `TestExtract` (`rest_test.go:5-22`), which unit-tests the `Extract(RestModel) (Model, error)` function directly against a hand-built `RestModel` literal — it never exercises `requests.GetRequest[[]RestModel]`'s actual JSON unmarshal via an `httptest.NewServer`. This is precisely the anti-pattern `libs/atlas-rest/CLAUDE.md:32` calls out: "The `FakeClient` mocks... bypass the unmarshal path and won't catch this" — testing `Extract` alone gives the same false confidence. |
| EXT-03 | 404 vs. other-failure distinction | Minor / Note | Same pattern as `character`: `chat/processor.go` passes errors through unmodified (no internal classification); `report/processor.go:141` correctly treats any `RecentInvolving` error as best-effort-degrade regardless of cause, which is the documented, intended behavior for this specific call site (see "Best-effort chat-transcript degradation" above) — so this is not a defect here, just noted for consistency with `character`. |
| EXT-04 | `RootUrl(domain)`, not hardcoded | PASS | `chat/requests.go:16` `requests.RootUrl("MESSAGES")` |

### `atlas-messages/chat` (domain package — has `model.go`; Redis-backed, not GORM)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01/02 | Processor / RestModel placement | PASS | `chat/processor.go:14-32` (Processor), `chat/resource.go:25-45` (RestModel + JSON:API methods — co-located with the handler in a single-endpoint package; acceptable, no second responsibility is smashed in) |
| DOM-04/05 | Transform / TransformSlice | Partial PASS | `chat/resource.go:47-56` defines `Transform(index int, line Line) RestModel` (not the standard `Transform(m Model) (RestModel, error)` signature, but this package has no `Model` type — `Line` is the domain type — and no separate `TransformSlice`; the handler loops manually at `resource.go:97-100`). Given this is a thin, single-endpoint, Redis-backed read package (not a full CRUD domain), this is judged acceptable rather than a DOM-05-style finding — noting it here for completeness rather than failing it, since the shape genuinely differs from the GORM-domain template DOM-05 targets. |
| DOM-12 | No `os.Getenv()` in handlers | PASS | `resource.go` has none; `os.Getenv` usage is confined to `config.go:24` (config loading, not a handler) |
| SEC (explicit ask) | Exposure documented accurately, not broadened | PASS | `chat/resource.go:17-24` and `messages/main.go` `Server` doc comment both state plainly: routed through ingress, reachable **without authentication**, exposes **including whispers**, accepted per `scope-amendment.md` Amendment 2, and explicitly instruct "Do not describe this endpoint as server-to-server only." Verified against `docs/tasks/task-145-player-reports/scope-amendment.md:111-121` — the code comments match the accepted-risk text precisely, no understatement. Verified `deploy/shared/routes.conf:337-340` routes `/api/chat/history` with the exact same flat, unauthenticated pattern as the pre-existing `/api/reports` (342-345), `/api/bans` (332-335), `/api/history` (417-420) routes — no broader exposure than the existing convention. Tenant scoping confirmed: `chat/processor.go:30-31` derives `t` from `tenant.MustFromContext(ctx)` and every registry call passes it through (`registry.go:33,45`), so the endpoint cannot leak cross-tenant chat. |
| DOM-25 | Config-resolved wire values | N/A | This package carries no client wire codes (plain JSON REST resource) |

### `atlas-messages/message` (existing domain package, `capture_test.go` newly added)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-24 | Kafka producer stubbed in tests that emit | **FAIL** | `message/capture_test.go` (282 lines, entirely new in this PR) defines a service-local `stubWriter` (`capture_test.go:107-113`, a hand-rolled `producer.Writer` that discards messages) and `setupStubProducer` (`capture_test.go:118-125`) instead of using the shared `producertest.InstallNoop()` (`libs/atlas-kafka/producer/producertest/producertest.go:33-38`, which does the identical thing and is the documented single source of truth). Worse, `capture_test.go:124` calls `t.Cleanup(producer.ResetInstance)` — the explicitly-banned pattern per DOM-24(e), since it reverts the process-wide producer singleton to its unstubbed default after each test, undermining the very purpose of the install. This is exercised for real: `TestHandleGeneralCapturesLine` (line 153), `TestHandlePetDoesNotCaptureLine` (183), `TestIssuePinkTextDoesNotCaptureLine` (209), and `TestSlashCommandShortCircuitsBeforeCapture` (258) all call `setupStubProducer(t)` then invoke `p.HandleGeneral`/`HandlePet`/`IssuePinkText`, which internally call `producer.ProviderImpl(...)(...)(...)(...)` for real (`message/processor.go` — confirmed via diff, e.g. the post-capture `producer.ProviderImpl(p.l)(p.ctx)(message2.EnvEventTopicChat)(...)` call immediately following each new `captureLine` call). Failure scenario: because each test's `t.Cleanup` fires at test end and reverts the singleton, any other test elsewhere in the `message` package binary that emits without its own producer stub, executing after one of these four tests in `go test`'s (non-deterministic within a package, but same-binary) run order, would hit the real 10-retry/100ms→10s/~42s backoff described in `producertest.go`'s doc comment instead of failing fast — this is exactly the class of hazard `producertest.InstallNoop()` (installed once via `TestMain`) exists to close off permanently for the whole package. |

### `atlas-channel` report feature (`socket/handler`, `socket/writer`, `kafka/consumer/report`, `report/`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-25 | Config-resolved wire values | PASS | `socket/writer/claim_result.go:28,34` and `socket/writer/sue_character_result.go:21` both resolve their mode byte via `atlas_packet.WithResolvedCode("operations", string(key), func(code byte) packet.Encoder {...})` against closed, typed constant key sets (`ClaimResultCode`, `SueResultCode`). Verified the per-writer `operations` tables exist and are correctly, distinctly scoped in `services/atlas-configurations/seed-data/templates/template_gms_92_1.json:664-681` (`ClaimResult`) and `:701-714` (`SueCharacterResult`) — confirmed no key-string collision between the two tables despite `UNABLE_TO_LOCATE`-shaped keys appearing at different opcodes for different writers. `libs/atlas-packet/report/clientbound/claim_result.go` and `serverbound/claim_request.go` correctly keep the raw `byte mode`/no version-resolution logic at the codec layer — resolution happens one layer up in `socket/writer`, matching the documented pattern. Both `ClaimAvailableTimeBody` and `ClaimSvrStatusChangedBody` (`socket/writer/claim_available_time.go`, `claim_status_changed.go`) carry no client-interpreted mode byte (hour/bool data only) — correctly un-resolved. |
| DOM-25 | Version gate idiom | PASS (verified, not re-litigated per instructions) | `libs/atlas-packet/field/serverbound/sue_character.go:86,100` both use the inline `t.MajorVersion() >= 92` form, matching the documented, intentional exception at lines 67-79 of the same file (guard-DSL only parses `*ast.BinaryExpr`, `MajorAtLeast(92)` would silently degrade to always-true). No other site in the report/sue feature reintroduces a bare `MajorAtLeast` call — grepped `report/clientbound`, `report/serverbound` for `MajorAtLeast`/`MajorVersion()`: zero hits (these codecs have no version branching at all, confirmed unversioned per `claim_request.go:27-30`'s comment). |
| DOM-25 | Seed template coverage | PASS | Only `template_gms_92_1.json` carries the `ClaimResult`/`SueCharacterResult`/`ClaimRequest`/`SueCharacter` entries — correct, since the feature is version-gated at `>= 92` (serverbound `SueCharacter`) and the report feature as a whole targets v92+ per `packet-findings.md`. No stale partial registration in earlier-version templates. |
| DOM-24 | Kafka producer stubbed in tests that emit | N/A (no unstubbed emit path exercised) | `report/producer_test.go` calls `sueCommandProvider(...)()`/`claimCommandProvider(...)()` directly — pure `model.Provider[[]kafka.Message]` functions that never touch `producer.ProviderImpl`. `kafka/consumer/report/consumer_test.go` exercises `handleStatusEvent`, which is a **consumer** (translates a Kafka event into a socket write via `reportAnnouncer`/`session.Announce`) with no Kafka-producing code path at all — most tests swap `reportAnnouncer` for a recording stub (`withRecordingAnnouncer`, `consumer_test.go:100-115`); the two tests that exercise the real `reportAnnouncer` (lines 416, 471) only reach `session.Announce`/a `net.Conn` write, never a Kafka producer. Correctly out of DOM-24's scope. |
| Multi-tenancy | PASS | `kafka/consumer/report/consumer.go:85` gates on `sc.IsWorld(tenant.MustFromContext(ctx), e.WorldId)`; `socket/handler/claim_request.go` and `sue_character.go` operate entirely on the per-session `s session.Model`, no tenant leakage risk. |
| Goroutines | PASS | `tools/goroutine-guard.sh` exit 0 covers this package; the one bare `go func(){...}()` at `kafka/consumer/report/consumer_test.go:497` is inside a `_test.go` file, which the guard (and DOM-26's own "excluding `_test.go` files" scoping) explicitly exempts. |

### `libs/atlas-packet/report/*` and `field/serverbound/sue_character.go`

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| Codec correctness (mode/byte handling) | PASS | Reviewed `report/clientbound/claim_result.go`, `report/clientbound/claim_available_time.go` (not shown, referenced), `report/serverbound/claim_request.go` — all keep raw byte fields, encode/decode symmetric, no config resolution baked into the codec layer (correctly left to the writer layer per DOM-25). |
| Tests | PASS | `go test -race` clean for both `report/clientbound` and `report/serverbound`; `field/serverbound/sue_character_test.go:102` explicitly regression-tests the `>=92` boundary against the previously-incorrect `>=95` gate. |

### `tools/packet-audit/*` (CLI tooling — not a microservice; DOM/FILE/SUB/EXT checklists N/A)

Build, vet, and `go test -race` all clean across `cmd`, `internal/idasrc` (new `mcphttp.go` `BinaryInfoProvider`/`GetBinaryInfo`, `internal/idasrc/mcphttp_test.go`), `internal/matrix`, `internal/discover`. No service-scaffolding, REST, Kafka, or GORM code paths — outside the scope of every checklist above. Reviewed for gross issues only; none found (`GetBinaryInfo` is well-documented, has a dedicated test file, and fails loudly on transport/decode errors per its own doc comment at `mcphttp.go:20-22`).

## Sub-Domain Checklist Results

No sub-domain (action-event, `resource.go`-without-`model.go`) packages were introduced or changed in this diff — `character` and `chat` (both atlas-ban and atlas-messages) are REST-client/support packages, evaluated above under the File Responsibilities and External HTTP Client checklists instead.

## External HTTP Client Checklist Summary

| Package | EXT-01 (relationship stubs) | EXT-02 (httptest integration test) | EXT-03 (404 vs. other) | EXT-04 (RootUrl) |
|---|---|---|---|---|
| `atlas-ban/character` | **FAIL** | **FAIL** | Minor/Note | PASS |
| `atlas-ban/chat` | **FAIL** | **FAIL** | Minor/Note (not a defect at the call site) | PASS |

## Security Review (SEC — explicit ask: chat-history endpoint exposure)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| Exposure accurately documented | PASS | See `atlas-messages/chat` row above |
| Exposure not broadened beyond accepted scope | PASS | Same flat, unauthenticated `routes.conf` pattern as pre-existing `/api/reports`, `/api/bans`, `/api/history` routes; no new auth bypass introduced |
| Tenant isolation preserved | PASS | `chat.Processor`/`Registry` scope every Redis key by `tenant.Model` |

Not re-litigating the accept-risk decision itself, per instructions — Amendment 2 in `scope-amendment.md` is the authority and this audit found no code that contradicts or understates it.

## Summary

### Blocking (must fix) — all Important severity per file-responsibilities.md / DOM checklist weighting; none downgraded for prevalence

- **DOM-02**: `atlas-ban/report` — `entity.go` has `Make(Entity)` but no `Model.ToEntity()`, contra `file-responsibilities.md:14`.
- **DOM-05**: `atlas-ban/report/rest.go` — no `TransformSlice` function; `handleGetReports` inlines the `model.SliceMap(Transform)` composition in the handler instead.
- **DOM-17 / DOM-27**: `atlas-ban/report/resource.go` — `handleGetReportById` (lines 70-75) collapses every error (including transient DB failures) into 404; `handleGetReports`/`handleUpdateReportStatus`/`handleGetReportById` use bare `w.WriteHeader(http.StatusInternalServerError)` (lines 50, 57, 111, 118) instead of the already-registered `server.WriteErrorResponse(d.Logger())(w)(err)` pattern used consistently by sibling `ban/resource.go` and `history/resource.go` in the same service.
- **EXT-01**: `atlas-ban/character/rest.go` and `atlas-ban/chat/rest.go` — `RestModel`s lack `SetToOneReferenceID`/`SetToManyReferenceIDs`, unlike the near-identical `atlas-fame/character/rest.go`. Risk: a `relationships` block in either upstream response breaks the entire unmarshal (task-037 bug class).
- **EXT-02**: `atlas-ban/character` (no test file at all) and `atlas-ban/chat` (`rest_test.go` only unit-tests `Extract`, never exercises the real unmarshal via `httptest.NewServer`) — no integration test would catch the EXT-01 gap above.
- **DOM-24**: `atlas-messages/message/capture_test.go` — service-local `stubWriter`/`setupStubProducer` instead of the shared `producertest.InstallNoop()`, plus the explicitly-banned `t.Cleanup(producer.ResetInstance)` (line 124) that reverts the stub after each test.

### Non-Blocking (should fix)

- **FILE-05 (Minor)**: `atlas-ban/character` and `atlas-ban/chat` define their thin `Model` struct inside `processor.go` rather than a dedicated `model.go`.
- **EXT-03 (Minor/Note)**: neither `character` nor `chat` performs `errors.Is(err, requests.ErrNotFound)` classification internally; `report/processor.go:82-86`'s reporter lookup collapses a genuine 404 into `ErrorCodeInternal` (the accused lookup at line 106 correctly distinguishes it, so this is asymmetric rather than uniformly lossy).

### Verified PASS (explicit asks from the audit brief, all confirmed by reading source, not assumed)

- DOM-25 config-resolved wire values: `claim_result.go` / `sue_character_result.go` writers, verified against actual per-writer `operations` tables in `template_gms_92_1.json` with no key collisions.
- `sue_character.go`'s inline `>= 92` version-gate idiom: confirmed intentional, documented, and not duplicated incorrectly elsewhere in the report/sue feature.
- Kafka business-rejection semantics: traced `buf.Put`/`Emit` source to confirm a rejection buffers the ERROR event and returns nil, never skipping emission.
- Chat-transcript best-effort degradation: narrowly scoped to the one documented failure mode, doesn't swallow other error classes.
- Truncation: both description (rune-count) and chat-log (byte-cap, rune-boundary-safe) truncation helpers verified correct and independently tested.
- Multi-tenancy: `report` package uses `db.WithContext(ctx)` exclusively, no explicit `tenant_id` filters, verified with dedicated isolation tests.
- Goroutines: `tools/goroutine-guard.sh` exit 0 across the entire diff.
- SEC: `/api/chat/history` exposure accurately documented and not broadened beyond the accepted scope-amendment risk.

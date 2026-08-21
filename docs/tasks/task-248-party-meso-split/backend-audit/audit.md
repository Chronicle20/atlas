# Backend Audit — task-248-party-meso-split

- **Service Path:** services/atlas-drops, services/atlas-character (changed packages only)
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-08-21
- **Build:** PASS
- **Tests:** All passed (atlas-drops: `go test ./... -count=1` all `ok`; atlas-character: `go test ./... -count=1` all `ok`), 0 failed
- **Overall:** NEEDS-WORK

## Build & Test Results

```
$ cd services/atlas-drops/atlas.com/drops && go build ./...        # exit 0, no output
$ cd services/atlas-drops/atlas.com/drops && go test ./... -count=1
ok  	atlas-drops	0.027s
ok  	atlas-drops/data/foothold	0.020s
ok  	atlas-drops/drop	0.662s
ok  	atlas-drops/map	0.059s
ok  	atlas-drops/party	0.028s
[...no-test-file packages omitted...]

$ cd services/atlas-character/atlas.com/character && go build ./...   # exit 0, no output
$ cd services/atlas-character/atlas.com/character && go test ./... -count=1   # exit 0, all ok
  (targeted re-run) ok atlas-character/character  7.340s
  (targeted re-run) ok atlas-character/kafka/consumer/drop  0.029s
  (full-module background run) exit code 0
```

## Applicability

| Family | Fired? | Trigger observation |
|---|---|---|
| DOM structure (DOM-01..05,11,16) | Yes | `party/model.go` (model.go), `party/rest.go`, `drop/processor.go` (processor.go, not this family but see REST) |
| FILE placement (FILE-01..06) | Yes | Every changed package audited unconditionally |
| SUB (SUB-01..04) | No | No changed package has `resource.go` without `model.go` |
| REST (DOM-06..09,12..15,17..19,32) | Yes | `party/rest.go`, `party/processor.go`, `drop/processor.go` all present |
| Constants reuse (DOM-21) | Yes | New `Recipient` struct (drop/split.go), new `actorTypeDrop` const (character/processor.go) |
| Testing (DOM-10,20,24,33) | Yes | `_test.go` files changed in every touched package; `drop.Processor` and `character.Processor` interfaces changed |
| Cache (DOM-29) | No | No `cache.go`, no cached processor state introduced |
| Messaging (DOM-30) | Yes | `drop/processor.go` emits via `AndEmit`/`message.Buffer`; `character/processor.go` emits via outbox |
| Multi-tenancy (DOM-31) | Yes | `party/rest.go` exists; `character/processor.go` reads tenant state |
| Migration hygiene (DOM-34,35) | No | No symbol moved between service and `libs/atlas-*` |
| Deploy & topics (DOM-22,23) | No | No new `libs/atlas-*` module; `MESO_AWARDED` is a new event *type* string within the pre-existing `EVENT_TOPIC_DROP_STATUS` topic, not a new topic env var |
| Runtime safety (DOM-26) | Yes | Every non-test changed Go file scanned; no bare `go` statement found |
| Channel wire values (DOM-25) | No | Diff does not touch `services/atlas-channel` or `libs/atlas-packet`; `MESO_AWARDED`'s `Amount`/`Picker`/`CharacterId` fields are consumed server-side only (atlas-character), never interpreted as a client dispatcher byte |
| Resilience (DOM-27,28) | Yes | `drop/processor.go`'s `resolveMembers` is an enrichment/fallback path that fetches remote data (atlas-parties) and degrades on error — DOM-28 fires |
| External clients (EXT-01..04) | Yes | `party/processor.go` calls `requests.GetRequest[[]RestModel]` for atlas-parties |
| Scaffolding (SCAFFOLD-01..09) | No | No new `services/atlas-<svc>/` directory, no new channel Writer/Handler, no `routes.conf` change |
| Security (SEC-01..04) | No | Neither service handles auth/tokens/redirects/secrets in the changed code |

## Checklist Results

### party (domain package — has `model.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | `builder.go` exists with `NewBuilder()`, fluent setters, validating `Build()` | FAIL | No `builder.go` file exists in `party/`. `NewBuilder()`/`NewMemberBuilder()` and the fluent setters are defined inline in `party/model.go:41-83` instead. |
| DOM-04 | `Transform(Model) (RestModel, error)` in `rest.go` | N/A | `party` is a pure external-client (consumer) package for atlas-parties — it never serves its own `parties` REST endpoint. `cross-service-implementation.md`'s "REST Client Pattern" (lines 465-533) documents this exact shape (`Extract`, no `Transform`) as the correct pattern for a consumer-only `rest.go`; that document is the EXT family's detail document and governs this file. `party/rest.go` correctly defines `Extract`/`ExtractMember` (rest.go:102-123) instead. |
| DOM-05 | `TransformSlice` used by list handlers | N/A | Same reasoning as DOM-04 — no list handler exists in this consumer-only package (no `resource.go`). |
| DOM-06 | Processor constructor takes `logrus.FieldLogger` | PASS | `party/processor.go:21` — `NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor`. |
| DOM-11 | Providers lazy via `database.Query`/`SliceQuery` | N/A | No `provider.go` in `party/`. |
| DOM-18 | REST models implement JSON:API interface | PASS | `party/rest.go:20-35` (`RestModel`), `:134-150` (`MemberRestModel`) implement `GetName`/`GetID`/`SetID`. |
| DOM-19 | Request models flat, no nested Data/Type/Attributes | PASS | `MemberRestModel` (rest.go:125-132) is flat. |
| DOM-31 | Tenant/trace never a REST-model field | PASS | Neither `RestModel` nor `MemberRestModel` carries a tenant/trace field (rest.go:15-18, 125-132). |
| FILE-01 | Processor in `processor.go` | PASS | `party/processor.go` holds `Processor`, `NewProcessor`, `ProcessorImpl` methods. |
| FILE-02 | RestModel/Transform-Extract/JSON:API methods in `rest.go` | PASS | All in `party/rest.go`. |
| FILE-03 | Cross-service request functions in `requests.go` | PASS | `party/requests.go:15-25`. |
| FILE-05 | Builder in `builder.go`, Model in `model.go` | FAIL | Same defect as DOM-01: `modelBuilder`/`memberBuilder` (party/model.go:41-83) live in `model.go`, not a separate `builder.go`. Important — structural file-responsibility violation, not down-rated for being a small package. |
| FILE-06 | No package-named catch-all carrying ≥2 responsibilities | PASS | `model.go` is not a package-named file; its two responsibilities (Model + Builder) are the specific FILE-05 finding above, tracked there rather than double-counted here. |
| EXT-01 | Target RestModel implements `SetToOneReferenceID` and `SetToManyReferenceIDs` | FAIL | `party/rest.go` defines `SetToManyReferenceIDs` (rest.go:67-78) but has **no** `SetToOneReferenceID` method on `RestModel` or `MemberRestModel`. `libs/atlas-rest/CLAUDE.md` states this stub is required "even when you don't care about the relationship payload," and gives the exact boilerplate atlas-drops omitted here. |
| EXT-02 | httptest-backed integration test with JSON:API fixture, asserts populated struct | FAIL | `party/rest_test.go` contains only `TestExtract`/`TestExtract_NoMembers`, which hand-construct a `RestModel` in Go and call `Extract` directly — no `httptest.Server`, no JSON:API-wire fixture exercising the actual `jsonapi.Unmarshal`/relationship-decode path that EXT-01's stub exists to satisfy. `libs/atlas-rest/CLAUDE.md`'s own remediation advice ("Add an httptest-backed integration test... FakeClient mocks... won't catch this") is exactly what's missing. |
| EXT-03 | Only genuine 404 → not-found; other errors bubble | PASS (no violation introduced) | `party/processor.go:30-33` (`GetByMemberId`) does no custom error reclassification at all — it passes the shared `requests.SliceProvider`/`model.FirstProvider` result straight through, introducing no bespoke 404-vs-other-error logic to get wrong. |
| EXT-04 | Service URL via `requests.RootUrl`/`RootUrlFor` | PASS | `party/requests.go:16` — `requests.RootUrlFor(ctx, "PARTIES")`. |
| DOM-20 | Table-driven tests (`tests := []struct{}` + `t.Run`) | FAIL | `party/rest_test.go` has two independent `Test*` functions (`TestExtract`, `TestExtract_NoMembers`) instead of a `tests := []struct{...}` table with `t.Run` subtests, despite the two cases sharing an obvious tabulable shape (input `RestModel` → expected `Model`). |
| DOM-24 | producertest stub for emit-reaching tests | N/A | `party/rest_test.go` never reaches an emit path. |

### drop (domain package — has `model.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06 | Processor constructor takes `logrus.FieldLogger` | PASS | `drop/processor.go:79` — `NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor`. |
| DOM-21 | No redeclaration of an existing `libs/atlas-constants` type | PASS | `Recipient` (drop/split.go:11-15) is a novel per-drop-award concept with no `libs/atlas-constants/` equivalent (checked item/inventory/weapon/world/channel/job/skill/monster-id categories — none apply). |
| DOM-26 | Goroutines via `routine.Go` | PASS | `grep -rn "\bgo "` across all changed non-test files in scope returns no bare `go` statement (only false-positive comment-text matches). |
| DOM-28 | Enrichment/fallback paths degrade loudly (`model.ErrDecorator` + `degrade.Observe`) | **FAIL** | `drop/processor.go:195-202` (`resolveMembers`) fetches remote data from atlas-parties (`p.pp.GetByMemberId`) and, on error, silently returns `nil` after only `p.l.WithError(err).Errorf(...)` — no `degrade.Observe` call, no `atlas_enrichment_degraded_total` increment, and `Errorf` (not `Warnf`). `libs/atlas-rest/degrade` exists in this repo (`libs/atlas-rest/degrade/degrade.go`) and is used by zero call sites in atlas-drops or atlas-character. `patterns-resilience.md`'s DOM-28 verification text is explicit: "A bare `if err != nil { return m }` that drops fetched data with no log and no metric is a finding regardless of justification." This is exactly that shape. Important severity — this is the mechanism behind the "atlas-parties outage degrades to a full-amount award" design note, so the degradation is invisible in `atlas_enrichment_degraded_total` today. |
| DOM-30 | DB-write + emit stay atomic via `AndEmit`+`message.Buffer` | PASS | `Reserve` appends `mesoAwardedEventStatusProvider` puts into the same `msgBuf` already used for the RESERVED event (drop/processor.go:175-186), inside the function later wrapped by `ReserveAndEmit`'s `message.Emit(producerProvider)(...)` (drop/processor.go:209-214) — same atomic buffer as the reservation write. |
| DOM-33 | Interface change updates every mock | N/A | `drop.Processor` gained `With(opts ...ProcessorOption) Processor` (drop/processor.go:21-22). `drop/mock/processor.go` does not implement the full `Processor` interface (missing `Consume`/`ConsumeAndEmit` already, pre-existing) and carries no `var _ drop.Processor = (*ProcessorMock)(nil)` assertion, so it does not fall under "every mock implementing the changed interface." `go build ./...` (drops module) confirms this compiles standalone. |
| FILE-01 | Processor in `processor.go` | PASS | `drop/processor.go` — interface, constructor, all `ProcessorImpl` methods. |
| FILE-05 / FILE-06 | No new violation | PASS | `split.go` is a focused, single-responsibility helper file (pure function + `Recipient` type), not a package-named catch-all. |
| DOM-20 | Table-driven tests | PASS (split_test.go) / **FAIL** (processor_test.go, new tests) | `drop/split_test.go:25-160` uses `tests := []struct{...}` + `t.Run` — compliant. `drop/processor_test.go`'s six new meso-split tests (`TestProcessor_Reserve_MesoDrop_SplitsAmongCoLocatedPartyMembers`, `..._ExcludesMembersNotCoLocated`, `..._ItemDrop_MakesNoPartyLookup`, `..._PartyLookupError_AwardsFullAmountToPicker`, `..._FailedReservation_EmitsNoAwards`, `..._ZeroShareSuppressesNonPickersOnly` — processor_test.go:1012-1264) are six independent `Test*` functions, not a table, despite sharing an obviously tabulable shape (party-roster fixture in, expected `[]Recipient`-shaped awards out). |
| DOM-24 | producertest stub for emit-reaching tests | N/A | None of `drop/processor_test.go`'s tests call an `*AndEmit` method or otherwise reach `producer.ProviderImpl`/`message.Emit`; every test drives the buffer-based (`Spawn`/`Reserve`/etc.) form directly. `grep -n "AndEmit\|producertest\|TestMain"` on the file returns nothing. |

### drop/mock (support package)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Mock lives with its own responsibility, not miscategorized | PASS | `drop/mock/processor.go` holds only the `ProcessorMock` struct and its methods. |
| DOM-33 | See drop package row above | N/A | Same disposition — `ProcessorMock` was left un-updated deliberately (context.md), matches the "no assertion, no full-interface implementation, no in-repo caller requiring `drop.Processor` satisfaction" test. |

### atlas-character `kafka/message/drop` (support package — shared event schema)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-21 | No redeclared constants | PASS | `MesoAwardedStatusEventBody`/`StatusEventTypeMesoAwarded` (kafka.go:77-84, 54-55) are new event-schema types mirroring atlas-drops' own event, not a `libs/atlas-constants` domain type. |
| DOM-23 | Topic env vars follow convention | N/A | No new topic env var added — `MESO_AWARDED` is a new `Type` string value inside the existing `EVENT_TOPIC_DROP_STATUS` topic (kafka.go:52-55), which is unchanged. |
| FILE placement | Command/Event/Body structs colocated correctly | PASS | All in `kafka/message/drop/kafka.go`, consistent with the rest of the file's existing shape. |

### atlas-character `kafka/consumer/drop` (support package — has neither `model.go` nor `resource.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-26 | Goroutines via `routine.Go` | PASS | No `go` statement added (`consumer.go` diff only changes handler wiring/body). |
| DOM-20 | Table-driven tests | FAIL | `consumer_test.go` has a single `Test*` function with no `tests := []struct{}` table. Only one scenario exists in this file (the type-guard short-circuit), so a table adds no coverage value here, but the rule's pass criterion is mechanical and the file does not meet it. Flagged for completeness, not weighted as equivalent severity to the multi-case violations above. |
| DOM-24 | producertest stub for emit-reaching tests | N/A | `TestHandleMesoAwarded_IgnoresNonMesoAwardedEvents` supplies a `RESERVED`-typed event so `handleMesoAwarded`'s type guard returns before reaching `character.NewProcessor(...).AwardPickedUpMeso` — no emit path is exercised. |

### atlas-character `character` (domain package — has `model.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06 | Processor constructor takes `logrus.FieldLogger` | PASS | `character/processor.go:172` — `NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor` (pre-existing, unchanged signature). |
| DOM-21 | No redeclared constants | PASS | `actorTypeDrop = "DROP"` (character/processor.go:931) is a free-form wire `ActorType` string value; no `SYSTEM`/`CHARACTER`/`ITEM` equivalent exists anywhere as a shared constant in `libs/atlas-constants/` (`find ... -iname "*actor*"` returns nothing), and the sibling values at other call sites are themselves bare string literals (e.g. `"SYSTEM"` in `meso_outbox_test.go:29`), so there is no existing declaration this redeclares. |
| DOM-30 | DB write + emit stay atomic | PASS | `AwardPickedUpMeso` (character/processor.go:941-981) wraps the meso credit in `database.ExecuteTransaction` and buffers `MESO_CHANGED`/`STAT_CHANGED` via `message.Emit(outbox.EmitProvider(...))` inside that same transaction (processor.go:944-969), consistent with the existing outbox pattern used by `RequestChangeMeso` right above it. |
| DOM-33 | Interface change updates every mock | N/A | `character.Processor` replaced `AttemptMesoPickUp` with `AwardPickedUpMeso` (processor.go:130); `grep -rln "type ProcessorMock struct"` under the character module and a search for any mock implementing `character.Processor` found none — no mock exists to update. |
| DOM-28 | Enrichment/fallback path degrades loudly | N/A (not this pattern) | The `drop.NewProcessor(...).RequestPickUp(...)` best-effort call inside `AwardPickedUpMeso` (processor.go:975-979) is a fire-and-forget *command* dispatch on the completion path, not a data-enrichment fetch that populates a `Model` field — DOM-28's trigger ("enrichment/fallback path... fetches remote data" to enrich a returned model) does not match this shape. The credit-fetch inside the transaction (`p.WithTransaction(tx).GetById()`) is a local DB read within the same transaction, not a remote-service enrichment either. |
| DOM-20 | Table-driven tests | FAIL | `character/meso_award_test.go` defines six independent `Test*` functions (`TestAwardPickedUpMeso_CreditsAndEmitsMesoChangedAndStatChanged`, `..._PickerCompletesThePickUp`, `..._NonPickerDoesNotCompleteThePickUp`, `..._ZeroAmountRunsNoTransactionButCompletesThePickUp`, `..._OverflowSkipsTheCreditButStillCompletesThePickUp`, `..._AmountAboveInt32IsRejected` — meso_award_test.go:54-216), sharing an obviously tabulable shape (meso amount / picker flag in → credited-meso / event-count / pick-up-command-count out), but implemented as separate functions rather than a `tests := []struct{}` + `t.Run` table. |
| DOM-10 | Test DB setup calls `database.RegisterTenantCallbacks` | PASS | `meso_award_test.go` reuses the existing `outboxTestDb(t)` → `testDatabase(t)` helper, which calls `database.RegisterTenantCallbacks(l, db)` at `character/processor_test.go:43` (pre-existing helper, not part of this diff, confirmed still in force). |
| DOM-24 | producertest stub for emit-reaching tests | PASS | `meso_award_test.go:55-56` — `capture := producertest.InstallCapturing(); t.Cleanup(producertest.InstallNoop)` in every test function; no `t.Cleanup(producer.ResetInstance)` used. |

## Security Review

Not applicable — the SEC-* family did not fire. Neither atlas-drops nor atlas-character's changed code in this diff handles authentication, authorization, tokens, redirects, or secrets.

## Not evaluable from the diff

- None. Every checklist item whose trigger fired was settled directly from the changed files, the linked pattern documents, and one targeted grep each for existing mocks / `libs/atlas-constants` equivalents / `degrade` package usage — no item required reading outside the declared scope.

## Summary

### Blocking (must fix)

- DOM-01 / FILE-05 — `services/atlas-drops/atlas.com/drops/party/model.go:41-83`: `modelBuilder`/`memberBuilder` and their fluent setters live inline in `model.go` instead of a separate `builder.go`.
- EXT-01 — `services/atlas-drops/atlas.com/drops/party/rest.go`: `RestModel` (and `MemberRestModel`) implement `SetToManyReferenceIDs` but never define `SetToOneReferenceID`, even as a no-op stub required by `libs/atlas-rest/CLAUDE.md`.
- EXT-02 — `services/atlas-drops/atlas.com/drops/party/rest_test.go`: no httptest-backed integration test serves a JSON:API fixture through the real unmarshal/relationship-decode path; only hand-constructed-`RestModel` unit tests of `Extract` exist.
- DOM-28 — `services/atlas-drops/atlas.com/drops/drop/processor.go:195-202` (`resolveMembers`): the atlas-parties enrichment fallback silently returns `nil` on error with only an `Errorf` log — no `degrade.Observe` call, no `atlas_enrichment_degraded_total` increment.
- DOM-20 — `services/atlas-drops/atlas.com/drops/party/rest_test.go`, `services/atlas-drops/atlas.com/drops/drop/processor_test.go` (new meso-split tests, lines 1012-1264), `services/atlas-character/atlas.com/character/character/meso_award_test.go`, `services/atlas-character/atlas.com/character/kafka/consumer/drop/consumer_test.go`: new/changed tests use independent `Test*` functions instead of the required `tests := []struct{...}` + `t.Run` table-driven pattern.

### Non-Blocking (should fix)

- None beyond the DOM-20 instances already listed as blocking (the checklist gives DOM-20 no lesser severity tier, so all instances are listed above rather than split here).

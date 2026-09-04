# Backend Audit — task-284-npc-ask-text-state

- **Service Path:** services/atlas-channel, services/atlas-configurations, services/atlas-npc-conversations, services/atlas-saga-orchestrator, libs/atlas-packet
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-08-29
- **Range:** `9cd1ec5af..e6a1540cb`
- **Build:** PASS
- **Tests:** all packages `ok` or `[no test files]` (no failures)
- **Overall:** NEEDS-WORK

## Build & Test Results

```
$ cd services/atlas-channel/atlas.com/channel && go build ./...        -> exit 0, no output
$ cd services/atlas-configurations/atlas.com/configurations && go build ./... -> exit 0, no output
$ cd services/atlas-npc-conversations/atlas.com/npc && go build ./...  -> exit 0, no output
$ cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go build ./... -> exit 0, no output

$ go test ./... -count=1  (all four modules): every package reports `ok` or
  `[no test files]`. No `FAIL` lines in any module's output.
```

## Applicability

| Family | Fired? | Trigger observation |
|---|---|---|
| FILE placement | Yes | Every changed Go package audited (unconditional) |
| DOM structure | Yes | `model.go` changed in `conversation` (askText additions); no new `entity.go`/`provider.go` in scope |
| SUB sub-domain | No | No changed package has `resource.go` without `model.go` |
| REST (DOM-06..09,12..15,17..19,32) | Yes | `conversation/processor.go`, `conversation/rest.go`, `conversation/quest/rest.go` changed |
| Constants reuse (DOM-21) | Yes | New const blocks: `AskTextType`, `CommandTypeText` |
| Testing (DOM-10,20,24,33) | Yes | Diff touches many `_test.go` files; `Continue`/`ContinueConversation`/`Processor` interfaces re-signed |
| Cache (DOM-29) | No | No `cache.go`, no cached-state struct touched |
| Messaging (DOM-30) | Yes | `npc/producer.go` (atlas-npc-conversations) changed, adds `textConversationProvider` |
| Multi-tenancy (DOM-31) | Yes | `rest.go` files changed in `conversation` and `conversation/quest` |
| Migration hygiene (DOM-34/35) | No | No symbol moved between service and `libs/atlas-*` |
| Deploy & topics (DOM-22/23) | No | No new `libs/atlas-*` module; no new/renamed Kafka topic env var — existing `EnvConversationCommandTopic` reused |
| Runtime safety (DOM-26) | Yes | Non-test Go files changed; grepped for bare `go ` — none added |
| Channel wire values (DOM-25) | Yes | `services/atlas-channel` and `libs/atlas-packet` touched |
| Resilience (DOM-27/28) | No | No `model.Decorator` / enrichment path changed; no handler touched (no `resource.go` in scope) |
| External clients (EXT-01..04) | Yes | New `conversation/quest/progress` package calls atlas-quest via `requests.RootUrlFor` + `requests.DrainProvider[RestModel, Model]` |
| Scaffolding (SCAFFOLD-01..09) | No | No new service directory; no new channel `Writer`/`Handler` constant registered (existing `NpcConversationWriter` / `NPCContinueConversationHandle` reused); `routes.conf` untouched |
| Security (SEC-01..04) | No | None of the touched services handle auth/tokens/secrets in this diff |
| patterns-provider.md (foundational) | No | No new provider composition pattern introduced |
| patterns-functional.md (foundational) | No | No new curried constructor/combinator pattern introduced beyond existing precedent (`announceTextConversation` mirrors sibling functions exactly) |

## Checklist Results

### conversation (domain package, atlas-npc-conversations)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | `builder.go` has `NewBuilder()`, fluent setters, validating `Build()` | PASS | `services/atlas-npc-conversations/atlas.com/npc/conversation/builder.go:1221` `NewAskTextBuilder()`; `builder.go:1233-1265` `Build()` validates text/maxLength/minLength/contextKey/nextState |
| DOM-02 | `Model.ToEntity()` in `entity.go` | N/A | Package has `model.go` but no `entity.go` (conversation state is registry-backed, not GORM-persisted) |
| DOM-03 | `Make(Entity)` in `entity.go` | N/A | Same — no `entity.go` |
| DOM-06 | Processor constructor takes `logrus.FieldLogger` | PASS | `services/atlas-npc-conversations/atlas.com/npc/conversation/processor.go:115` `func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor` |
| DOM-09 | Every `Transform(` call site checks its error | N/A | No `resource.go` in the changed `conversation` package |
| DOM-11 | Providers lazy via `database.Query`/`SliceQuery` | N/A | Package has no `provider.go` |
| DOM-18 | REST models implement `GetName()`/`GetID()`/`SetID()` | PASS | `RestStateModel` (pre-existing) unaffected; new `RestAskTextModel`/`RestAskTextMatchModel` are nested attribute structs, not top-level JSON:API resources, so they carry no `GetName`/`GetID` — consistent with the sibling `RestAskNumberModel` |
| DOM-19 | Request models are flat | PASS | `services/atlas-npc-conversations/atlas.com/npc/conversation/rest.go:194-203` `RestAskTextModel`/`RestAskTextMatchModel` — flat, no nested `Data`/`Attributes` |
| DOM-21 | No redeclaration of an existing shared constant | PASS | `AskTextType` (`model.go:47`), `CommandTypeText` (`kafka/message/npc/kafka.go:25`) — grepped `libs/atlas-constants/` for `ASK_TEXT`/`AskText`, no match |
| DOM-24 | Test packages reaching an emit path install `producertest` or inject a no-op | N/A | New tests (`processor_asktext_test.go`, `processor_asktext_reprompt_test.go`) swap the entire downstream sender via `SetNpcSenderProcessorFactory` (`processor.go:97-110`) before any `AndEmit`/`producer.ProviderImpl` call is reached — no live producer path exercised |
| DOM-26 | Every goroutine via `routine.Go` | PASS | `git diff -- '*.go' \| grep '^\+.*\bgo '` — no bare `go` statement added in this diff |
| DOM-30 | DB-writing operation emits via `AndEmit`+`message.Buffer` | PASS (documented exception) | `services/atlas-npc-conversations/atlas.com/npc/npc/processor.go:146` `SendText` calls `producer.ProviderImpl` directly, matching the "operations over non-DB state" exception — no DB write occurs on this path (conversation state lives in the Redis-backed registry, not GORM) |
| DOM-31 | Tenant/trace never in a REST model or request body | PASS | `RestAskTextModel`/`RestAskTextMatchModel` (`rest.go:194-203`) carry no tenant/trace field |
| DOM-33 | Interface change updates every mock in the same diff | PASS | `Processor.Continue` re-signed (`processor.go:48`); `services/atlas-npc-conversations/atlas.com/npc/conversation/mock/processor.go:20,54-60` updated with matching signature in the same diff |

### conversation/quest (domain-adjacent REST layer, atlas-npc-conversations)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-04 | `Transform(Model) (RestModel, error)` in `rest.go` | PASS | `services/atlas-npc-conversations/atlas.com/npc/conversation/quest/rest.go:410-433` `TransformAskText` mirrors the top-level `conversation` package |
| DOM-19 | Flat request models | PASS | `quest/rest.go:166-183` `RestAskTextModel`/`RestAskTextMatchModel` flat |

### conversation/quest/progress (new support package, atlas-npc-conversations)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor interface/constructor/methods live in `processor.go` | PASS | `services/atlas-npc-conversations/atlas.com/npc/conversation/quest/progress/processor.go:17-30,37-55` |
| FILE-02 | `RestModel`, `Extract`, JSON:API methods live in `rest.go` | FAIL | `services/atlas-npc-conversations/atlas.com/npc/conversation/quest/progress/rest.go:7-36` (`RestModel`, `GetName`/`GetID`/`SetID`, `Extract` at line 58) — technically satisfied for the REST responsibility, but the same file also carries the domain `Model` (see FILE-05/06) |
| FILE-03 | Cross-service request functions live in `requests.go` | PASS | `services/atlas-npc-conversations/atlas.com/npc/conversation/quest/progress/requests.go:10-24` |
| FILE-04 | Entity/Migration/TableName in `entity.go` | N/A | No persistence — this package is a pure atlas-quest REST client, no `entity.go` expected |
| FILE-05 | Domain `Model` struct lives in `model.go` | FAIL (Important) | `services/atlas-npc-conversations/atlas.com/npc/conversation/quest/progress/rest.go:42-55` defines `type Model struct { infoNumber uint32; progress string }` with accessor methods — this is the domain model and belongs in a `model.go`, not `rest.go`. There is no `model.go` in this package at all. |
| FILE-06 | No file carrying ≥2 of the FILE-01..05 responsibilities | FAIL (Important) | `rest.go` bundles the REST responsibility (`RestModel`/`Extract`/JSON:API methods, lines 7-36) together with the domain-model responsibility (`Model` struct + accessors, lines 42-55) in the same file — two of the FILE-* responsibilities in one file. Prevalence elsewhere in the service does not exempt this (Mindset rule); this is a new package created in this diff and is directly gradable. |
| EXT-01 | Target REST model implements `SetToOneReferenceID`/`SetToManyReferenceIDs` | FAIL | `services/atlas-npc-conversations/atlas.com/npc/conversation/quest/progress/rest.go:7-36` — `RestModel` has no `SetToOneReferenceID` or `SetToManyReferenceIDs` method, not even as a no-op |
| EXT-02 | `httptest`-backed integration test asserts a populated domain struct from a representative JSON:API fixture | FAIL | No `httptest.NewServer` anywhere under `conversation/quest/progress/`. `rest_test.go` only exercises `Extract`/`SetID` on hand-built `RestModel` values (`rest_test.go:9-40`) — it never round-trips a real JSON:API response through `requests.DrainProvider`. The sibling `atlas-npc-conversations/atlas.com/npc/map` package, which also uses `requests.DrainProvider` for a paginated GET, does carry this integration test (`map/processor_drain_test.go`) — this new package should have matched that precedent and did not. |
| EXT-03 | Only genuine 404s map to domain "not found" | PASS | `services/atlas-npc-conversations/atlas.com/npc/conversation/quest/progress/processor.go:47-48` `if errors.Is(err, requests.ErrNotFound) { return nil, ErrNotFound }` — all other errors bubble with their original error (line 50-51) |
| EXT-04 | URL composed via `requests.RootUrl`/`RootUrlFor`, not hardcoded DNS | PASS | `services/atlas-npc-conversations/atlas.com/npc/conversation/quest/progress/requests.go:10-11` `requests.RootUrlFor(ctx, "QUEST")` |

### operation_executor (conversation package, atlas-npc-conversations)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-13/14 | Handler-layer orchestration / processor-only calls | N/A | No `resource.go` in scope; `executeLocalOperation` (`operation_executor.go:806-885`) calls `e.questProgressP.GetByCharacterAndQuest` (a processor method), never a bare provider or DB call, for `local:get_quest_progress` |
| DOM-20 | Table-driven tests | PASS | `services/atlas-npc-conversations/atlas.com/npc/conversation/operation_executor_test.go:963` `tests := []struct{...}` + `t.Run` for `TestGetQuestProgressOperation` |

### npc (channel-command sender, atlas-npc-conversations `atlas.com/npc/npc`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-33 | Interface change updates every mock | N/A | `Processor.SendText` added (`npc/processor.go:38`); `find ... -iname '*mock*'` under this package returns nothing — no mock exists to update |
| DOM-30 | DB write + emit atomicity | PASS (documented exception) | `npc/processor.go:146` `SendText` → `producer.ProviderImpl` directly; no DB write on this path (matches "operations over non-DB state" exception, same as sibling `SendNumber`/`SendStyle`) |

### atlas-channel: npc (command producer)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-33 | Interface change updates every mock | N/A | `Processor.ContinueConversation` re-signed (`services/atlas-channel/atlas.com/channel/npc/processor.go:13`); no mock exists under `atlas-channel/atlas.com/channel/npc` |

### atlas-channel: kafka/consumer/npc/conversation

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-25 | Client-interpreted bytes resolved from a tenant table, not Go literals | PASS | `services/atlas-channel/atlas.com/channel/kafka/consumer/npc/conversation/consumer.go:237-238` `case "TEXT": return npcpkt.NpcConversationMessageTypeAskText` — a semantic key (pre-existing const in `libs/atlas-packet`), resolved to the wire byte via the tenant opcode template's `messageType` map (`services/atlas-configurations/seed-data/templates/template_gms_92_1.json` new `"ASK_TEXT": 3` entry), never a literal byte in Go |
| DOM-20 | Table-driven tests | PASS | `services/atlas-channel/atlas.com/channel/kafka/consumer/npc/conversation/consumer_test.go:23` `tests := []struct{...}` + `t.Run` |
| DOM-24 | Emit-path stubbing | N/A | `TestAnnounceTextConversation` calls `newTextConversation` directly, never `announceTextConversation`/`session.Announce`, so no writer.Producer path is reached |

### atlas-channel: socket/handler (NPCContinueConversationHandleFunc)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-20 | Table-driven tests | PASS | `services/atlas-channel/atlas.com/channel/socket/handler/npc_continue_conversation_test.go` (new file, 180 lines) — table-driven per grep of `tests :=` |
| (pre-existing, noted, not a new finding) | `_ = npcProcessorFunc(...).ContinueConversation(...)` discards the error | Out of scope | `services/atlas-channel/atlas.com/channel/socket/handler/npc_continue_conversation.go:80,94,101` — pre-existing discard pattern, not introduced or worsened by this diff |

### SCAFFOLD-07 (seed-data templates)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| SCAFFOLD-07 | New writer/handler seeded in every targeted tenant template | N/A | No *new* `Writer`/`Handler` constant is registered by this diff — `NpcConversationWriter` and `NPCContinueConversationHandle` both pre-date this branch. The three template edits (`template_gms_87_1.json`, `template_gms_92_1.json`, `template_jms_185_1.json`) add the `ASK_TEXT` entry to an existing `messageType` options table and (gms_92 only) fill a pre-existing handler-binding gap — governed by DOM-25, not SCAFFOLD-07 |

## Security Review

Not applicable — none of the touched services (`atlas-channel`, `atlas-configurations`, `atlas-npc-conversations`, `atlas-saga-orchestrator`) handle authentication, authorization, tokens, redirects, or secrets in this diff. SEC-01..04 triggers did not fire.

## Not evaluable from the diff

- Whether `libs/atlas-packet/npc/clientbound/conversation.go`'s `AskTextConversationDetail.Encode` correctly version-gates gms_v84 vs gms_v92 byte layout — the codec itself was not touched in this diff (only its test file changed), and it predates `9cd1ec5af`; verifying it would require reading packet-audit history outside this diff's range.
- Whether `RestStateModel`'s `GetName()`/`GetID()`/`SetID()` (DOM-18) remain correct after the `AskText` field addition — the methods themselves are in unchanged lines of `rest.go`; the diff only adds a new optional field to the struct, which cannot break an existing method body, so this was not independently re-verified beyond confirming the field addition is additive (`omitempty`).

## Summary

### Blocking (must fix)
- FILE-05 / FILE-06: `services/atlas-npc-conversations/atlas.com/npc/conversation/quest/progress/rest.go:42-55` — the domain `Model` struct (and its accessors) is defined in `rest.go` instead of a `model.go`; `rest.go` thereby carries two FILE-* responsibilities (REST transform + domain model) in one file.
- EXT-01: `services/atlas-npc-conversations/atlas.com/npc/conversation/quest/progress/rest.go:7-36` — `RestModel` has no `SetToOneReferenceID`/`SetToManyReferenceIDs`, even as no-ops.
- EXT-02: `services/atlas-npc-conversations/atlas.com/npc/conversation/quest/progress/` — no `httptest`-backed integration test asserts a populated `Model` from a representative atlas-quest JSON:API fixture; the sibling `map/processor_drain_test.go` establishes this precedent in the same service and was not followed.

### Non-Blocking (should fix)
- None identified beyond the blocking items above.

### Known, already-reviewed items (not new findings, reported per instruction)
- `services/atlas-npc-conversations/atlas.com/npc/kafka/consumer/npc/consumer.go:102` discards `Continue`'s error (`_ = ...`). Pre-existing, consumer-wide, explicitly out of scope for this branch.
- `services/atlas-npc-conversations/atlas.com/npc/conversation/processor.go:399-412` — the `AskNumberType` arm of `Continue` returns a bare error on a min/max violation with no re-prompt, the same defect the `AskTextType` arm (lines 432-460 of this diff) just fixed. Deliberately left for a separate follow-up per task instructions.

# Backend Audit — task-247-extra-expression-items

- **Service Path:** services/atlas-channel, services/atlas-expressions, libs/atlas-constants, libs/atlas-packet
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-08-21
- **Commit range:** 24a33a2e6..HEAD
- **Build:** PASS
- **Tests:** all packages `ok` (0 failed), across atlas-channel, atlas-expressions, atlas-constants, atlas-packet
- **Overall:** PASS

## Build & Test Results

```
services/atlas-channel/atlas.com/channel:      go build ./...  -> exit 0 (no output)
services/atlas-channel/atlas.com/channel:      go test ./... -count=1 -> all `ok`, no FAIL
services/atlas-expressions/atlas.com/expressions: go build ./... -> exit 0 (no output)
services/atlas-expressions/atlas.com/expressions: go test ./... -count=1 -> all `ok`, no FAIL
libs/atlas-constants:                          go build ./... -> exit 0; go test ./... -> all `ok`
libs/atlas-packet:                             go build ./... -> exit 0; go test ./... -> all `ok`
```

## Applicability

| Family | Fired? | Trigger observation |
|---|---|---|
| DOM structure (DOM-01,02,03,04,05,11,16) | Fired | `services/atlas-expressions/atlas.com/expressions/expression/` has `model.go` (unchanged by diff) |
| FILE placement (FILE-01..06) | Fired | Every changed Go package runs this family unconditionally |
| SUB sub-domain (SUB-01..04) | N/A | No changed package has `resource.go` without `model.go` |
| REST (DOM-06..09,12..15,17..19,32) | Fired (partially) | Changed packages have `processor.go` (`character/expression/processor.go`, `expression/processor.go`); no `resource.go`/`rest.go` in scope, no route registration |
| Constants reuse (DOM-21) | Fired | `libs/atlas-constants/item/expression.go` declares new consts/functions and a numeric-literal item-id classification (`ClassificationExpression*10000+…`) |
| Testing (DOM-10,20,24,33) | Fired | Diff touches multiple `_test.go` files; `expression.Processor` interface (atlas-expressions) re-signed |
| Cache (DOM-29) | N/A | No `cache.go` in any changed package; no cached state introduced |
| Messaging (DOM-30) | Fired | Both `character/expression/producer.go` (atlas-channel) and `expression/producer.go` (atlas-expressions) emit Kafka messages |
| Multi-tenancy (DOM-31) | Fired | New ownership-check code (`character_expression.go`) reads tenant/trace state via `ctx` |
| Migration hygiene (DOM-34,35) | N/A | No diff hunk moves/extracts symbols between a service and a `libs/atlas-*` module — `item/expression.go` is a wholly new file, not a relocation |
| Deploy & topics (DOM-22,23) | N/A | No new `libs/atlas-*` module added; no Kafka topic env var added or renamed — `COMMAND_TOPIC_EXPRESSION`/`EVENT_TOPIC_EXPRESSION` pre-exist and are unchanged, only new struct fields were added |
| Runtime safety (DOM-26) | Fired | Non-test Go files changed; `tools/goroutine-guard.sh` run, exit 0 |
| Channel wire values (DOM-25) | Fired | Diff touches `services/atlas-channel` and `libs/atlas-packet` | 
| Resilience (DOM-27,28) | N/A | No DB-backed handler error branches changed; no `model.Decorator`/enrichment path changed |
| External clients (EXT-01..04) | N/A | No new `requests.RootUrl`/`requests.*Request[T]` call site added in the diff (the ownership check calls the pre-existing, unmodified `character.NewProcessor(...).GetById(...)`) |
| Scaffolding (SCAFFOLD-01..09) | N/A | No new `services/atlas-<svc>/`, no new channel `Writer`/`Handler`, no `deploy/shared/routes.conf` change |
| Security (SEC-01..04) | N/A | Neither service handles authentication, tokens, or secrets |
| patterns-provider.md | N/A | No new provider defined/composed in the diff |
| patterns-functional.md | N/A | No curried constructor/decorator/combinator defined in the diff |

## Checklist Results

### `services/atlas-channel/atlas.com/channel/character/expression` (support — no `model.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor lives in `processor.go` | PASS | `character/expression/processor.go:14-38` — `Processor` interface, `ProcessorImpl`, `Change` all in one file, correctly named |
| FILE-05 | Producer/writer/reader placement | PASS | `character/expression/producer.go` holds only `SetCommandProvider` |
| FILE-06 | No catch-all file mixing ≥2 responsibilities | PASS | `processor.go` and `producer.go` each hold a single responsibility |
| DOM-06 | Processor constructor takes `logrus.FieldLogger` | PASS | `character/expression/processor.go:25` `func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor` |
| DOM-30 | DB-write operations emit via `AndEmit`+`message.Buffer` | PASS (documented exception) | `character/expression/processor.go:35-38` emits via a direct `producer.ProviderImpl(...)` call; this processor performs no DB write on any path (state lives only in the outbound Kafka command), matching the "operations over non-DB state" exception in `patterns-kafka.md` |
| DOM-33 | Interface change updates every mock | PASS (no mock exists) | `grep` for a mock of `atlas-channel`'s `expression.Processor` found none — nothing to update |

### `services/atlas-channel/atlas.com/channel/kafka/message/expression` (support)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-06 | No catch-all responsibility collapse | PASS | `kafka.go` holds only the `Command`/`Event` DTOs and their topic-env consts — the established message-package shape used throughout `atlas-channel/kafka/message/*` |
| DOM-23 | Topic env vars follow convention, no literal redeclare | N/A | `EnvExpressionCommand`/`EnvExpressionEvent` constants unchanged (`kafka.go:12,28`); only new struct fields (`Duration`, `ByItemOption`) were added — no topic added or renamed |

### `services/atlas-channel/atlas.com/channel/kafka/consumer/expression` (support)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-25 | Client-interpreted wire bytes resolved from a tenant table, not literals | N/A | `consumer.go:63` passes `e.Duration`/`e.ByItemOption` straight through as data fields, not a dispatcher mode/sub-op/fail-reason code selected from a lookup switch — outside DOM-25's scope |
| — | int32→uint32 narrowing correctness | PASS | `consumer.go:57-63` — `uint32(e.Duration)` is a documented, deliberate bit-pattern-preserving cast (`e.Duration` is `int32`, always `-1` for a v95-originated emote per the cited IDA addresses); the comment at lines 57-62 states the rationale and the resulting client-visible bytes (`FF FF FF FF`), matching `libs/atlas-packet/character/clientbound/expression_test.go`'s `TestCharacterExpressionByteOutputV95NegativeDuration` fixture |

### `services/atlas-channel/atlas.com/channel/socket/handler` (support — no `model.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-06 | No catch-all responsibility collapse | PASS | `character_expression.go` holds only the handler function and its test seam; no Processor/RestModel/Entity mixed in |
| DOM-13/14/15 (handlers: no cross-domain orchestration / no direct provider calls / no direct DB writes) | N/A | Rule triggers on "package has `resource.go`" — `socket/handler` has no `resource.go` (it is a socket dispatcher package, not a REST resource package) |
| DOM-31 | Tenant/trace travel in context only | PASS | `character_expression.go:22` `expressionItemOwnedFunc` takes `ctx context.Context` and `character_expression.go:23` calls `character.NewProcessor(l, ctx)` — tenant is resolved from `ctx` inside the pre-existing, unmodified `character` processor, never passed as an explicit field/param |
| DOM-26 | Every goroutine via `routine.Go` | PASS | `tools/goroutine-guard.sh` exit 0 (repo-wide, includes this package) |
| — | Ownership-gate fail-closed on lookup error | PASS | `character_expression.go` (test `TestCharacterExpressionHandleFunc_Gate/"lookup error fails closed"` in `character_expression_test.go:91-98`) asserts 0 commands emitted when `expressionItemOwnedFunc` returns an error |
| — | Emote bound enforced before ownership lookup | PASS | Handler drops `emote > item.MaxEmoteId` before ever calling `expressionItemOwnedFunc` — asserted by `TestCharacterExpressionHandleFunc_Gate/"out of range never reaches the lookup"` (emote 24, `expectCalls: 0`) |

### `libs/atlas-constants/item` (support library package)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-06 | No catch-all responsibility collapse | PASS | `expression.go` adds only consts and pure helper functions, consistent with sibling files `death_protection.go`/`vegas_spell.go` in the same package |
| DOM-21 | No redeclaration of an existing shared type/const | PASS | `expression.go:36` reuses `ClassificationExpression` from `constants.go:96` (`Classification(516)`) rather than redeclaring a `516` literal; `grep -rn -i "emote\|expression" libs/atlas-constants` outside `item/expression.go` found no prior emote-id or item-mapping helper to duplicate |

### `services/atlas-expressions/atlas.com/expressions/expression` (domain — has `model.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | `builder.go` has `NewBuilder()` + fluent `Build()` | Out of diff scope | `builder.go` is unmodified by this diff (not in `git diff --stat`); its constructor is pre-existing `NewModelBuilder`, not introduced or touched by task-247 — not graded against this change |
| DOM-02/03 | `Model.ToEntity()` / `Make(Entity)` in `entity.go` | N/A | No `entity.go` in this package |
| DOM-04/05 | `Transform`/`TransformSlice` in `rest.go` | N/A | No `rest.go` in this package |
| DOM-06 | Processor constructor takes `logrus.FieldLogger` | PASS | `expression/processor.go:37` `func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor` |
| DOM-10 | Test DB setup calls `RegisterTenantCallbacks` | N/A | No GORM DB opened directly in the changed test files (`processor_test.go` uses `message.NewBuffer()`/miniredis-backed registry, not a SQL DB) |
| DOM-11 | Providers lazy via `database.Query` | N/A | No `provider.go` in this package |
| DOM-16 | `administrator.go` holds writes | N/A | No `administrator.go`; state lives in the in-memory `registry.go` (unchanged by this diff), not a DB |
| DOM-20 | Tests are table-driven | PASS | `processor_test.go:211-263` (`TestProcessor_Change_PropagatesDurationAndByItemOption`, `TestRevertExpressionEmitsZeroDurationAndFalseByItemOption`) are simple single-case tests, acceptable per DOM-20 (table-driven applies where multiple cases exist); `character_expression_test.go` (atlas-channel) uses `tests := []struct{...}` + `t.Run` at line 40 |
| DOM-24 | Test packages reaching an emit path stub the producer | PASS | `expression/testmain_test.go:1-11` installs `producertest.InstallNoop()` in `TestMain` for the whole package |
| DOM-30 | DB writes emit via `AndEmit`+`message.Buffer` | PASS | `expression/processor.go:46-58` (`Change`) writes to `GetRegistry()` (in-memory, not DB) and buffers its event via `mb.Put(...)`; `ChangeAndEmit` (`processor.go:63-79`) wraps `Change` in `message.EmitWithResult(...)`, the `AndEmit` shape |
| DOM-33 | Interface change updates every mock | PASS | `expression/mock/processor.go:13-25` — both `ChangeFunc` and `ChangeAndEmitFunc` (and the methods invoking them) updated with the new `duration int32, byItemOption bool` parameters, matching `processor.go`'s re-signed `Change`/`ChangeAndEmit` |

### `services/atlas-expressions/atlas.com/expressions/kafka/consumer/expression` (support)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| — | Command→ChangeAndEmit field-for-field passthrough | PASS | `consumer.go:41` `processor.ChangeAndEmit(c.TransactionId, c.CharacterId, f, c.Expression, c.Duration, c.ByItemOption)` — all five new/changed `Command` fields threaded through, none dropped |

### `libs/atlas-packet/character/clientbound` (packet codec library)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| — | Version-gated encode/decode symmetry preserved | PASS | `expression.go:54-71` (`Encode`) and `expression.go:73-88` (`Decode`) both gate the new `byItemOption` field identically on `t.Region()=="GMS" && t.MajorVersion()>87`; round-trip test `TestCharacterExpressionRoundTrip` (unchanged) plus new `TestCharacterExpressionByteOutputV95ByItemOption`/`...NegativeDuration` (`expression_test.go:54-86`) assert the exact wire bytes |
| — | Downstream call sites updated for new constructor arity | PASS | `v61_test.go:724`, `v72_test.go:506`, `v79_test.go:520` all updated to pass the new `false` `byItemOption` arg, keeping pre-v95 fixtures unchanged in value |

## Not evaluable from the diff

none — every applicable rule was settled from the changed files plus the targeted lookups above (character's `GetById`/`InventoryDecorator`/`FindFirstByItemId` symbols were located and confirmed pre-existing and unmodified, which was sufficient to dispose of DOM-31/EXT-01..04 without reading further into the character package's HTTP client internals).

## Additional Observation (not a checklist rule — flagged for awareness)

The two Kafka struct shapes on either side of the event seam are asymmetric:
`services/atlas-expressions/atlas.com/expressions/kafka/message/expression/kafka.go:17-27`
gives `StatusEvent` a `TransactionId uuid.UUID` field (populated at
`services/atlas-expressions/atlas.com/expressions/expression/producer.go` via
the `transactionId` passed into `expressionEventProvider`), but the
channel-side consumer's mirror struct,
`services/atlas-channel/atlas.com/channel/kafka/message/expression/kafka.go:31-40`
(`Event`), has no `TransactionId` field at all. JSON unmarshal silently drops
the field rather than erroring, so this is not a functional break — nothing in
`services/atlas-channel/atlas.com/channel/kafka/consumer/expression/consumer.go`
reads a transaction id from the event — but it is a genuine drift between the
two sides of the same wire contract, and every sibling `Command`/`Event` pair
this diff touches otherwise keeps `TransactionId` symmetric. No numbered
checklist rule governs message-struct field symmetry, so this is not graded as
a FAIL, but it is worth a deliberate call: either the channel side should carry
`TransactionId` too (for tracing/correlation on the outbound announce) or the
omission should be a documented, intentional choice.

## Summary

### Blocking (must fix)
- none

### Non-Blocking (should fix)
- none (see "Additional Observation" above for a drift worth a deliberate decision, not gated as a rule violation)

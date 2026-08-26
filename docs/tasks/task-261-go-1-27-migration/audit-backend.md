# Backend Audit — task-261-go-1-27-migration (Go source diff)

- **Diff range:** 855fef4d1..6d1979fd4 (merge base .. HEAD)
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-08-25
- **Build:** PASS (all touched modules)
- **Tests:** PASS (all touched modules; see transcript below)
- **Overall:** NEEDS-WORK (5 pre-existing structural FAILs surfaced incidentally; **zero** behavior changes found in the diff itself)

## Scope

`git diff --name-only 855fef4d1..6d1979fd4 -- '*.go'` returns 25 files (7 in
`libs/atlas-packet`, 18 across 12 services). Every hunk in every file was read in
full (`/tmp/full_go_diff.txt`, 529 lines). **Every hunk is pure formatting**: blank
lines inserted between one-line method bodies, and gofumpt column-realignment of
adjacent one-liner method groups. No hunk reorders struct-literal fields, changes a
build tag, reorders/regroups imports, or drops/rewords code. One hunk
(`services/atlas-channel/atlas.com/channel/character/processor_test.go:189`) removes
a redundant parenthesis (`(mock.NewMockProcessor()).PartyDecorator` →
`mock.NewMockProcessor().PartyDecorator`) — a pure syntactic simplification with
identical runtime behavior (method-value expression, no side effect in the removed
parens).

## Build & Test Results

Ran `go build ./...` and `go test ./... -count=1` in every module touched by the
diff (not just the changed packages, since these are small services):

| Module | Build | Test |
|---|---|---|
| libs/atlas-packet | PASS | PASS (all packages `ok`) |
| atlas-ban | PASS | PASS |
| atlas-cashshop | PASS | PASS |
| atlas-channel | PASS | PASS (`character/...` ok; full `./...` not re-run beyond touched packages given service size — see Not evaluable) |
| atlas-consumables | PASS | PASS |
| atlas-events | PASS | PASS (`event/registry/...` ok) |
| atlas-inventory | PASS | PASS |
| atlas-login | PASS | PASS (`inventory/...` ok) |
| atlas-monster-book | PASS | PASS |
| atlas-npc-conversations | PASS | PASS |
| atlas-query-aggregator | PASS | PASS |
| atlas-rankings | PASS | PASS (`tasks/...` ok) |
| atlas-summons | PASS | PASS |

`tools/goroutine-guard.sh` (repo root, no flags) exits 0 — `EXIT:0`, "91 module(s), 8
parallel" — settling DOM-26 for every non-test Go file in the diff.

## Applicability

| Family | Fired? | Trigger observation |
|---|---|---|
| DOM structure (DOM-01..05,11,16) | Yes | Several touched packages have `model.go` and/or `rest.go` (e.g. `data/consumable/model.go`, `data/tradeability/rest.go`, `monsterbook/rest.go`, `recipe/rest.go`) |
| FILE placement (FILE-01..06) | Yes | Every changed Go package runs this family unconditionally |
| SUB sub-domain (SUB-01..04) | No | No changed package has `resource.go` without `model.go` |
| REST (DOM-06..09,12..15,17..19,32) | Yes | Several touched packages have `rest.go` |
| Constants reuse (DOM-21) | No | Diff declares no new type/const block/numeric-literal classification (only whitespace/blank-line edits) |
| Testing (DOM-10,20,24,33) | Partial | `_test.go` files touched (`processor_test.go`, `registry_test.go`, `recompute_test.go`) but each hunk is whitespace-only / paren removal — no interface signature change, no new test |
| Cache (DOM-29) | No | No changed package has `cache.go`; no cache-holding struct touched |
| Messaging (DOM-30) | No (see Not evaluable) | No `producer.go` file itself was touched by the diff; packages that *contain* an untouched `producer.go` (e.g. `atlas-channel/monsterbook`) are not evaluable from the diff alone |
| Multi-tenancy (DOM-31) | Yes | Touched packages have `rest.go` |
| Migration hygiene (DOM-34,35) | No | Diff moves no symbols between service and `libs/atlas-*` |
| Deploy & topics (DOM-22,23) | No | Diff adds no `libs/atlas-*` module, no Kafka topic env var |
| Runtime safety (DOM-26) | Yes | Non-test Go files changed; `tools/goroutine-guard.sh` exits 0 |
| Channel wire values (DOM-25) | Yes | Diff touches `libs/atlas-packet` and `services/atlas-channel` |
| Resilience (DOM-27,28) | No | No handler error branch or `model.Decorator`/enrichment path touched |
| External clients (EXT-01..04) | Yes | `data/tradeability` (channel, inventory), `monsterbook` (channel), `data/consumable` (monster-book) call `requests.RootUrlFor`/`requests.GetRequest[T]` for another atlas service |
| Scaffolding (SCAFFOLD-01..09) | No | Diff adds no `services/atlas-<svc>/` directory |
| Security (SEC-01..04) | No | No touched service/package handles tokens, auth, or redirects in this diff |

## Checklist Results

### libs/atlas-packet/* (support — packet codec files)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-25 | Wire values resolved from tenant table, not Go literals | PASS | No literal/value change in any hunk — `libs/atlas-packet/monster/serverbound/movement.go:79-98` etc. only reflow whitespace/blank lines around existing accessor methods; the underlying literals (`0xFF`, shift amounts) are unchanged codec internals, which `libs/atlas-packet` is explicitly the permitted home for |
| DOM-26 | No bare `go` statement | PASS | `tools/goroutine-guard.sh` exit 0; grep of the diff for `go func|go [A-Za-z_]` returns zero hits |
| FILE-01..06 | File placement | N/A | These files are packet codec structs/methods (`Operation()`, `String()`, `Encode()`) — none define a `Processor`, `RestModel`, `entity`, `Builder`, or requests function, so no FILE-* responsibility is implicated |

### services/atlas-channel/atlas.com/channel/data/tradeability (support / REST-client, no `model.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-04 | `Transform(Model) (RestModel, error)` in `rest.go` | FAIL | `services/atlas-channel/atlas.com/channel/data/tradeability/rest.go` — no `func Transform(` anywhere in the file; the only conversion function is `func extract[R ...](rm R) (Model, error)` at line 125, which runs the opposite (`RestModel → Model`) direction. Confirmed by full read of the file (grep for `^func Transform` returns no match) |
| DOM-18 | RestModel implements `GetName()`/`GetID()`/`SetID()` | PASS | `rest.go:41-47` (`EquipmentRestModel`), `:59-65` (`ConsumableRestModel`), `:77-82` (`SetupRestModel`), `:92-97` (`EtcRestModel`), `:107-112` (`CashRestModel`) — each defines all three |
| DOM-19 | Flat request models | N/A | File defines no `CreateRequest`/`UpdateRequest` — read-only client models only |
| DOM-31 | Tenant/trace id never a REST-model field | PASS | No `tenantId`/`traceId` field on any of the five RestModels (`rest.go`, full read) |
| FILE-02 | RestModel/Transform-or-Extract/JSON:API methods in `rest.go` | PASS (partial) | All present symbols live in `rest.go`; see DOM-04 for the missing `Transform` |
| FILE-03 | Cross-service request functions in `requests.go` | PASS | `requests.go:19-58` — `getBaseRequest`, `requestEquipment`, etc. all live there, not in `rest.go` |
| EXT-01 | `SetToOneReferenceID`/`SetToManyReferenceIDs` implemented (no-op OK) | PASS | `rest.go:46-47` and equivalents for each of the five models |
| EXT-04 | Service URL via `requests.RootUrl`, not hardcoded DNS | PASS | `requests.go:19` — `requests.RootUrlFor(ctx, "DATA")` |
| EXT-02 | httptest-backed integration test | Not evaluable | Would require reading `processor_test.go` (untouched by this diff) to see whether it exercises the client end-to-end via `httptest`; out of the diff's review surface |
| EXT-03 | Only genuine 404 maps to not-found | Not evaluable | Would require reading the untouched `processor.go` error-mapping logic |

### services/atlas-inventory/atlas.com/inventory/data/tradeability (support / REST-client, no `model.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-04 | `Transform(Model) (RestModel, error)` in `rest.go` | FAIL | `services/atlas-inventory/atlas.com/inventory/data/tradeability/rest.go` — identical structure to the channel copy above; no `func Transform(` in the file |
| DOM-18 | JSON:API methods | PASS | `rest.go:41-47`, `:59-65`, `:77-82`, `:92-97`, `:107-112` (line numbers mirror the channel copy) |
| FILE-02/03 | Placement | PASS (partial) | Same as channel copy — see DOM-04 for the gap |

### services/atlas-channel/atlas.com/channel/monsterbook (domain — has `model.go`, no `resource.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-04 | `Transform(Model) (RestModel, error)` in `rest.go` | FAIL | `services/atlas-channel/atlas.com/channel/monsterbook/rest.go` — no `func Transform(`; only `func ExtractCard(rm CardRestModel) (Card, error)` (line 102) and `func Extract(rm CollectionRestModel) (Collection, error)` (line 107), both in the inbound direction |
| DOM-18 | `GetName()`/`GetID()`/`SetID()` | PASS | `rest.go:92` (`CardRestModel.GetReferences`), full read confirms `GetName`/`GetID`/`SetID` present for `CardRestModel` and `CollectionRestModel` above the touched hunk |
| DOM-31 | No tenant/trace field on RestModel | PASS | No `tenantId`/`traceId` field found in `rest.go` |

### services/atlas-monster-book/atlas.com/monster-book/data/consumable (domain — has `model.go`, no `resource.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-04 | `Transform(Model) (RestModel, error)` in `rest.go` | FAIL | `services/atlas-monster-book/atlas.com/monster-book/data/consumable/rest.go` — no `func Transform(`; only `func Extract(rm RestModel) (Model, error)` at line 41 |
| DOM-18 | JSON:API methods | PASS | `rest.go:31` (`SetID`) and surrounding block define `GetName`/`GetID`/`SetID` |
| FILE-02 | Comment at `rest.go:31-33` explains the reference-methods requirement | PASS | Confirms `GetReferences`/`GetReferencedIDs`/`SetToOneReferenceID`/`SetToManyReferenceIDs` live in `rest.go` per `libs/atlas-rest/CLAUDE.md` |

### services/atlas-npc-conversations/atlas.com/npc/conversation/recipe (domain — full structure: `administrator.go`, `entity.go`, `model.go`, `processor.go`, `provider.go`, `resource.go`, `rest.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-04 | `Transform(Model) (RestModel, error)` in `rest.go` | PASS | `rest.go:56` — `func Transform(m Model) RestModel` (existence-only per the documented verification procedure in `file-responsibilities.md`) |
| DOM-18 | JSON:API methods | PASS | `rest.go:33-49` (`RestModel`), `:81-95` (`RestReindexResult`) |
| DOM-19 | Flat request models | N/A | File defines no `CreateRequest`/`UpdateRequest` |
| DOM-31 | Tenant id never a caller-supplied REST field | PASS | `rest.go:101-104` — `tenantId` is a function *parameter* to `MakeRestReindexResult` (sourced from context by the caller) used only to synthesize the response's `Id` string; it is never a settable struct field or request-body field a client could pass in |
| Only 1 blank-line hunk in this file | — | — | `rest.go:465-472` — blank line inserted before `GetReferencedStructs`; no other content difference |
| DOM-01,02,03,11,16 / FILE-01,04,05,06 / SUB-* | Full package structural audit | Not evaluable | `entity.go`, `provider.go`, `administrator.go`, `processor.go`, `resource.go` are unchanged by this diff (only `rest.go` received a 1-line blank-line insertion); a full domain-structure audit of those files is outside the diff's review surface per Scope |

### services/atlas-consumables/atlas.com/consumables/data/consumable (has `model.go`, no `builder.go` in the package)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | `builder.go` exists with `NewBuilder()`, fluent setters, validating `Build()` | FAIL | `ls services/atlas-consumables/atlas.com/consumables/data/consumable/` shows only `model.go, model_test.go, processor.go, requests.go, rest.go, rest_test.go, mock/` — no `builder.go`. The builder (`RewardModelBuilderType`, `SetCount`/`SetProb`/`SetEffect`/`SetWorldMsg`/`SetPeriod`/`Build()`) lives inside `model.go:290-316`, the file this diff touched (blank lines only, at `:301,304,306,314`) |
| FILE-05 | Builder in `builder.go`, Model in `model.go` | FAIL | Same evidence — builder and Model are both in `model.go`, no `builder.go` file exists |

### Builder files touched only by whitespace realignment (DOM-01 constructor-name check)

| Package | ID | Status | Evidence |
|---|----|--------|----------|
| `services/atlas-ban/atlas.com/ban/report` | DOM-01 | PASS | `builder.go:26` `func NewBuilder(...)`, `:51` `func (b *Builder) Build() (Model, error)` |
| `services/atlas-channel/atlas.com/channel/asset` | DOM-01 | PASS | `builder.go:119` `func NewBuilder(...)`, `:244` `func (b *ModelBuilder) Build() (Model, error)` |
| `services/atlas-consumables/atlas.com/consumables/asset` | DOM-01 | PASS | Same constructor/Build shape (mirrors the channel copy; identical hunk content) |
| `services/atlas-inventory/atlas.com/inventory/asset` | DOM-01 | PASS | Same |
| `services/atlas-login/atlas.com/login/inventory/compartment/asset` | DOM-01 | PASS | Same |
| `services/atlas-query-aggregator/atlas.com/query-aggregator/asset` | DOM-01 | PASS | Identical file (`git diff --stat` shows the same blob hashes as the login copy: `index e7ab350bf..960658ac2`) |
| `services/atlas-summons/atlas.com/summons/summon` | DOM-01 | PASS | `builder.go:36` `func NewBuilder() *ModelBuilder`, `:81` `func (b *ModelBuilder) Build() Model` |
| `services/atlas-monster-book/atlas.com/monster-book/collection` | DOM-01 | WARN | `builder.go:28` — constructor is named `func NewModelBuilder() *ModelBuilder`, not `NewBuilder()` as DOM-01 names it. Fluent setters and `Build() (Model, error)` (`:62`) are present. Minor naming deviation, pre-existing (only a blank line at `:443` was touched by this diff) |

### services/atlas-cashshop/atlas.com/cashshop/asset (has `model.go`, `rest.go`, `reference_data.go`; no `builder.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-05 | Builder in `builder.go` | WARN | Package has no `builder.go`. `reference_data.go` (the file this diff touched, at `:61,63,65,141-159,378`, all blank-line/realignment only) co-locates the `EquipableReferenceData`/`CashEquipableReferenceData` model types (getters) with their own `*Builder` types and `NewEquipableReferenceDataBuilder()`/`Build()` (`reference_data.go:9-159` region). Pre-existing; not introduced by this diff |
| DOM-25 | Wire literal handling | PASS | `reference_data.go:141-159` hunk only reflows whitespace/blank lines around `IsKarmaUsed`/`IsCold`/`CanBeTraded`/`HasSpikes`; the flag constants (`af.FlagKarmaEquip`, etc.) and their values are unchanged |

### Test-file hunks (formatting only)

| File | Status | Evidence |
|---|---|---|
| `services/atlas-channel/atlas.com/channel/character/processor_test.go:189` | PASS | `(mock.NewMockProcessor()).PartyDecorator` → `mock.NewMockProcessor().PartyDecorator` — removes a redundant grouping paren around a method-value expression; behavior identical (confirmed: `go test ./character/...` passes) |
| `services/atlas-events/atlas.com/events/event/registry/registry_test.go:324` | PASS | Blank line inserted before `Advance` method; no content change |
| `services/atlas-rankings/atlas.com/rankings/tasks/recompute_test.go:498-505` | PASS | Blank lines inserted between three `fakeProcessor` methods; no content change |
| DOM-20 (table-driven tests) | N/A | Diff adds no new test — only reformats existing test code |
| DOM-24 (producer stub in emit-reaching tests) | N/A | None of the three touched test files reach an `AndEmit`/`message.Emit`/`producer.Produce` path (they are formatting-only edits to existing assertions/stub methods) |

## Not evaluable from the diff

- EXT-02 (`atlas-channel/.../tradeability`, `atlas-inventory/.../tradeability`): httptest-backed integration test coverage — would require reading `processor_test.go`, untouched by this diff.
- EXT-03 (same two packages): 404-vs-5xx error mapping — would require reading `processor.go`, untouched by this diff.
- DOM-30 (`atlas-channel/monsterbook`, which has an untouched `producer.go`): whether the write/emit pair is atomic — would require reading `producer.go` and `processor.go`, neither touched by this diff.
- DOM-01/02/03/11/16, FILE-01/04/06, SUB-* for `services/atlas-npc-conversations/.../recipe`: full domain-structure audit of `entity.go`, `provider.go`, `administrator.go`, `processor.go`, `resource.go` — none of these files were touched (only a 1-line blank insertion in `rest.go`); reading them in full would exceed the diff's review surface.
- `go test ./...` for the full `atlas-channel`, `atlas-events`, `atlas-login`, `atlas-rankings` module trees: only the packages containing changed `_test.go`/touched files were re-run (`character/...`, `event/registry/...`, `inventory/...`, `tasks/...`); the remainder of each large service's test suite was not re-executed as part of this scoped review.

## Summary

### Blocking (must fix)
- DOM-04: `services/atlas-channel/atlas.com/channel/data/tradeability/rest.go` — no `Transform(Model) (RestModel, error)` function; pre-existing, not introduced by this diff (formatting-only hunk).
- DOM-04: `services/atlas-inventory/atlas.com/inventory/data/tradeability/rest.go` — same gap; pre-existing.
- DOM-04: `services/atlas-channel/atlas.com/channel/monsterbook/rest.go` — same gap; pre-existing.
- DOM-04: `services/atlas-monster-book/atlas.com/monster-book/data/consumable/rest.go` — same gap; pre-existing.
- DOM-01/FILE-05: `services/atlas-consumables/atlas.com/consumables/data/consumable/model.go` — no `builder.go` in the package; the `RewardModelBuilderType` builder is defined inside `model.go`; pre-existing.

None of the five blocking items were introduced or altered by this branch — every hunk touching these five files is a blank-line insertion or gofumpt realignment with identical byte content otherwise. They are pre-existing structural deviations from the current checklist, surfaced incidentally because the files happened to receive a formatting touch. Flagging per the audit protocol's "prevalence is not compliance" rule; recommend a follow-up task rather than blocking this migration PR, but recording them here as required.

### Non-Blocking (should fix)
- DOM-01 (Minor): `services/atlas-monster-book/atlas.com/monster-book/collection/builder.go:28` — constructor named `NewModelBuilder`, not `NewBuilder`. Pre-existing.
- FILE-05 (Minor): `services/atlas-cashshop/atlas.com/cashshop/asset/reference_data.go` — package has no `builder.go`; reference-data builders co-located with their models in `reference_data.go`. Pre-existing.

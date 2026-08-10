# Backend Guidelines Audit — task-205-player-trade

- **Scope:** Go files changed on branch `task-205-player-trade` vs merge-base `1e0a321b808a1cf70e3638a6433408209ac744a9`. Headline: new service `services/atlas-trades/atlas.com/trades`, plus `libs/atlas-packet`, `libs/atlas-saga`, `libs/atlas-constants`, and touched packages in atlas-channel, atlas-inventory, atlas-tenants, atlas-data, atlas-saga-orchestrator, atlas-mini-games, atlas-consumables.
- **Guidelines Source:** `.claude/skills/backend-dev-guidelines/resources/*`
- **Date:** 2026-08-10
- **Build/Tests:** Not re-run per task instructions — reported clean by the invoking session (all 12 changed modules, 9 repo guards, `tools/lint.sh --check`, `docker buildx bake` for all 9 touched services). This audit performed Phase 2/3 mechanical + file-responsibilities review only.
- **Overall:** NEEDS-WORK

## Domain Checklist Results — `atlas-trades/trade` (domain: has `model.go`, `builder.go`, `resource.go`, `rest.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | builder.go exists | PASS | `trade/builder.go:14` `NewBuilder`, room construction with invariants |
| FILE-01 | Processor logic in `processor.go` (or `processor_<group>.go`) | **FAIL (Important)** | `trade/processor.go:145` declares the `Processor` interface including `Confirm`, `Attest`, `ExpireAttestation`, `SettlementSucceeded`, `SettlementFailed`, `ReconcileSettlements` (processor.go:198,206,210,220,222,229 per the interface doc block), but every one of those methods is *implemented* in `trade/settlement.go` (e.g. `func (p *ProcessorImpl) Confirm` at `trade/settlement.go:176`, `Attest` at `:277`, `ExpireAttestation` at `:325`, `SettlementSucceeded` at `:820`, `SettlementFailed` at `:829`, `ReconcileSettlements` at `:1033`). `settlement.go` is a bare topic-name file, not `processor.go` or the idiomatic `processor_settlement.go` split the guideline explicitly allows. Same file also holds unrelated helpers (`attestationTimers`, `GetAttestationTimers`, `Reconcile` package function) — i.e. it mixes a second responsibility (attestation-timer state) into the same catch-all file. |
| DOM-06/07 | FieldLogger + `d.Logger()` | PASS | `trade/processor.go:265` `NewProcessor(l logrus.FieldLogger, ...)`; `trade/resource.go:152,184` pass `d.Logger()` |
| DOM-09 | Transform errors handled | PASS | no bare `_, _ :=`/`_ =` Transform calls found in `trade/resource.go` |
| DOM-12 | No os.Getenv in handlers | PASS | zero matches in `trade/resource.go` |
| DOM-15 | No direct entity creation in handlers | PASS | zero `db.Create`/`db.Save`/`db.Delete` in `trade/resource.go` |
| DOM-11 | Providers use lazy evaluation | N/A | `trade` package has no `entity.go`/DB provider (in-memory registry by design) |
| DOM-25 | Client wire values config-resolved | **FAIL (Important, cross-repo)** | See "DOM-25 — TRADE_MESO_LIMIT" finding below. |

## Domain Checklist Results — `atlas-trades/ledger` (domain: has `model.go`, `entity.go`, `builder.go`, `administrator.go`, `provider.go`, `resource.go`, `rest.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | builder.go exists | PASS | `ledger/builder.go` present with `NewBuilder`/`Build` |
| DOM-02 | `ToEntity()` method on Model | **FAIL (Important)** | No `Model.ToEntity()` exists anywhere in the package. `ledger/administrator.go:52` instead defines a free function `func toEntity(tenantId uuid.UUID, m Model) Entry` — wrong shape (function, not method) AND wrong file (`administrator.go`, not `entity.go`). |
| DOM-03 | `Make(Entity)` function in `entity.go` | **FAIL (Important)** | `Make` exists but lives in `ledger/model.go:146`, not `ledger/entity.go` as file-responsibilities.md and DOM-03 require. |
| FILE-04 | Entity + Migration + TableName in `entity.go` | PARTIAL — `Migration`/`TableName` correctly in `entity.go` (`ledger/entity.go:52,74,88,91`), but `Make`/`toEntity` conversion logic is scattered into `model.go` and `administrator.go` respectively instead of `entity.go`. |
| DOM-11 | Providers use lazy evaluation | **FAIL (Important)** | `ledger/provider.go:33-42` `entryByIdProvider` runs `First(&e)` **eagerly** inside the "provider" constructor call and then wraps the already-fetched row in `model.FixedProvider(e)` — the exact anti-pattern DOM-11 bans ("eager execution wrapped in FixedProvider" instead of `database.Query[T]`). Same pattern repeats at `ledger/provider.go:48-57` (`entryByTransactionIdProvider`) and `ledger/provider.go:84-93` (`entriesByCharacterProvider`). None of these defer the query to the point the returned closure is invoked. |
| DOM-06/07 | FieldLogger + `d.Logger()` | PASS | `ledger/processor.go:47`; `ledger/resource.go:127,153` |
| DOM-09/17/27 | Transform errors handled; error→status mapping; transient errors via `WriteErrorResponse` | PASS | `ledger/resource.go:130,137,154-161,167` — 404 via `gorm.ErrRecordNotFound`, else `server.WriteErrorResponse` |
| DOM-18 | JSON:API interface on RestModel | PASS | not independently re-verified beyond rest.go presence; no contrary evidence found |

## Domain Checklist Results — `atlas-trades/settlement` (domain: has `model.go`, `entity.go`, `builder.go`, `administrator.go`, `provider.go`; no REST resource by design — internal-only)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-02 | `ToEntity()` method | **FAIL (Important)** | No `Model.ToEntity()`; `settlement/administrator.go:21` defines free function `func toEntity(t tenant.Model, m Model) Entry` in the wrong file. |
| DOM-03 | `Make(Entity)` in `entity.go` | **FAIL (Important)** | `Make` is in `settlement/model.go:134`, not `settlement/entity.go`. |
| DOM-11 | Providers use lazy evaluation | **FAIL (Important)** | `settlement/provider.go:30-38` `byTransactionIdProvider` eagerly runs `First(&e)` then wraps the result in `model.FixedProvider(e)` — same anti-pattern as ledger. |

## Domain Checklist Results — `atlas-trades/configuration` (domain: has `model.go`; REST-client config cache, no DB)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | builder.go exists | **FAIL (Important)** | No `builder.go` in the package (`ls configuration/` has no builder file). `Model` is constructed via `DefaultConfig()` + `With*` copy-setters (`configuration/model.go:61-103`) instead of the documented `NewBuilder()`/`Build()` pattern with invariant validation at construction time. |
| — | Singleton cache pattern | Minor / non-blocking | `configuration/registry.go` implements exactly the file-responsibilities.md `cache.go` shape (`sync.Once` singleton, `sync.RWMutex`, per-key cache) but is named `registry.go`/`GetRegistry()` rather than `cache.go`/`GetCache()`. It correctly avoids the "cache in processor constructor" anti-pattern (ai-guidance.md Core Rule 8), so this is a naming/placement deviation, not a functional bug. |

## File Responsibilities Checklist — support/data packages (`data/character`, `data/inventory`, `data/item`, `data/location`, `data/map`, `data/saga`, `compartment`, `saga`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01/02/03 | Processor/RestModel/requests in correct files | PASS (all 8 packages) | e.g. `data/character/processor.go:17,31,40`; `data/character/rest.go:11`; `data/character/requests.go:15-20` — verified for all of `data/character`, `data/inventory`, `data/item`, `data/location`, `data/map`, `data/saga`, `compartment`, `saga`; no catch-all `<pkg>.go` file in any of them |
| DOM-21 | Reuse atlas-constants | PASS | `inventory.Type` reused directly (`data/inventory/model.go:52`, `trade/model.go:65`, `trade/restriction.go:60-65`); `_map.FieldLimitNoMiniRoom`/`_map.NoMiniRoom` reused via wrapper, not redeclared (`services/atlas-trades/atlas.com/trades/data/map/field_limit.go:3,20` wrapping `libs/atlas-constants/map/field_limit.go:35,44`) |
| EXT-01 | JSON:API relationship no-op stubs | PASS | `SetToOneReferenceID`/`SetToManyReferenceIDs` present on every external `RestModel`/`AssetRestModel`/`EquipmentRestModel`/etc. in `data/character/rest.go:40,44`, `data/inventory/rest.go:40,42,90,92`, `data/item/rest.go:33,35,53,55,75,77,97,99,119,121`, `data/location/rest.go:46,48`, `data/map/rest.go:39,43`, `data/saga/rest.go:80,82` |
| EXT-02 | httptest-backed integration test | **FAIL (Important)** — all 6 external-HTTP-client packages | No package under `data/character`, `data/inventory`, `data/item`, `data/location`, `data/map`, `data/saga` contains an `httptest.NewServer`-backed test. `grep -rln "httptest.NewServer"` across the whole `atlas-trades` module returns nothing (only `trade/resource_test.go` and `ledger/resource_test.go` use `httptest.NewRequest`/`NewRecorder` to test this service's *own* handlers, not to fixture the upstream response). Existing `rest_test.go` files (`data/inventory/rest_test.go:35-129`, `data/saga/rest_test.go:10`) only unit-test `Extract`/getters against hand-built structs, never round-trip an actual JSON:API fixture (with a `relationships` block) through the real `GetRequest[T]` unmarshal path. Per EXT-02, hand-built `Extract` tests do not satisfy the requirement — they bypass unmarshal entirely. |
| EXT-04 | RootUrl-based service URLs | PASS | `data/character/requests.go:16`, `data/inventory/requests.go:17`, `data/item/requests.go:19`, `data/location/requests.go:15`, `data/map/requests.go:15`, `data/saga/requests.go:21` — all use `requests.RootUrl(...)` |

## Cross-Repo Finding — DOM-25: `TRADE_MESO_LIMIT` unconditionally sent to tenants whose config table lacks it

**FAIL (Blocking).**

1. `libs/atlas-packet/interaction/clientbound/interaction_body.go:297` resolves the `TRADE_MESO_LIMIT` mode byte via `atlas_packet.WithResolvedCode("operations", CharacterInteractionModeTradeMesoLimit, ...)` — the **mandatory** resolver variant per DOM-25/anti-patterns.md ("`WithResolvedCode(...)` for mandatory tables; a soft resolver with bare-arm fallback for optional ones").
2. `services/atlas-configurations/seed-data/templates/template_jms_185_1.json` — the clientbound `"CharacterInteraction"` writer's `"operations"` table (around line 4151-4178) contains `TRADE_PUT_ITEM`, `TRADE_ADD_MESO`, `TRADE_CONFIRM` but has **no `TRADE_MESO_LIMIT` key at all**. Every other in-scope template (`template_gms_48_1.json` … `template_gms_95_1.json`) has all four keys (verified: `grep -o "TRADE_PUT_ITEM\|TRADE_ADD_MESO\|TRADE_CONFIRM\|TRADE_MESO_LIMIT" template_*.json | sort | uniq -c`).
3. `services/atlas-channel/atlas.com/channel/kafka/consumer/trade/consumer.go:367-373` (`handleMesoRefusedEvent`) **unconditionally** calls `interactioncb.CharacterInteractionTradeMesoLimitBody()` for every tenant on a meso-limit rejection — it does not gate on whether the tenant's template configured the arm, unlike the `CashTradingRoomDlg` case the design doc calls out (design.md §4.2/§11.2: "the cash room has no mode-21 arm at all").
4. `libs/atlas-packet/resolve.go:29-45` (`ResolveCode`): when a key is missing from the options map it logs `Errorf("Code [%s] not configured...Defaulting to 99 which will likely cause a client crash.")` and returns byte `99`.

Net effect: any JMS (`jms_v185`) tenant whose trade partner sends an out-of-range meso amount will have the server encode and send a `TRADE_MESO_LIMIT` packet with wire mode `99` to the client — the exact "likely client crash" scenario DOM-25 exists to prevent. This is also a design-doc self-contradiction: `plan.md:5330` documents "TRADE_MESO_LIMIT (omit on versions whose dispatcher has no mode-21 arm)" but the emitting consumer code was never updated to omit the call for those versions/tenants; the resolver was also left as the mandatory `WithResolvedCode` variant instead of the soft-fallback pattern the guideline requires for genuinely optional tables.

## DOM-24 — Kafka producer stubbing in tests

**PASS**, atlas-trades: the only `message.Emit`/`AndEmit` path exercised by trades' own tests is `trade/processor.go:311` `(p *ProcessorImpl) emit`, which routes through `message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))` — the transactional outbox (`libs/atlas-outbox/provider.go:21`), which writes to a DB table inside the SQLite test transaction rather than calling a real Kafka producer. No `producertest` stub is required for this path, and none of `compartment/`, `saga/` (atlas-trades) has direct tests of their `AndEmit` functions that would need one.

**PASS**, cross-repo: `atlas-saga-orchestrator/saga/testmain_test.go:7-11` installs `producertest.InstallNoop()`; `atlas-inventory/compartment/processor_test.go:38,79-84` installs it in `TestMain` and correctly restores it (not `t.Cleanup(producer.ResetInstance)`) after `installCapturingProducer` swaps it out (`compartment/processor_test.go:74-76`).

## Summary

### Blocking (must fix)
- **DOM-25**: `TRADE_MESO_LIMIT` clientbound packet resolves to wire byte 99 (client-crash sentinel) for `jms_v185` tenants because the seed template's writer options omit the key while `services/atlas-channel/.../kafka/consumer/trade/consumer.go:372` sends it unconditionally, and `libs/atlas-packet/interaction/clientbound/interaction_body.go:297` uses the mandatory `WithResolvedCode` resolver instead of a soft fallback for this documented-optional arm.

### Non-Blocking / Important (should fix before merge)
- FILE-01: `trade/settlement.go` holds 6 `Processor` interface method implementations (`Confirm`, `Attest`, `ExpireAttestation`, `SettlementSucceeded`, `SettlementFailed`, `ReconcileSettlements`) plus unrelated attestation-timer state, in a bare-topic-named file instead of `processor.go`/`processor_settlement.go`.
- DOM-02/DOM-03 (ledger): no `Model.ToEntity()`; `toEntity` is a free function in `ledger/administrator.go:52`; `Make` is in `ledger/model.go:146` instead of `ledger/entity.go`.
- DOM-02/DOM-03 (settlement): same pattern — `toEntity` free function in `settlement/administrator.go:21`; `Make` in `settlement/model.go:134` instead of `settlement/entity.go`.
- DOM-11 (ledger + settlement): `ledger/provider.go:33-42,48-57,84-93` and `settlement/provider.go:30-38` eagerly execute GORM queries and wrap the already-fetched result in `model.FixedProvider`, instead of deferring via `database.Query[T]`/`database.SliceQuery[T]`.
- DOM-01 (configuration): no `builder.go`/`NewBuilder()`/`Build()` for `configuration.Model`; constructed via `DefaultConfig()` + `With*` copy-setters instead.
- EXT-02: none of `data/character`, `data/inventory`, `data/item`, `data/location`, `data/map`, `data/saga` has an `httptest.NewServer`-backed integration test exercising the real unmarshal path against a JSON:API fixture with a `relationships` block; existing `rest_test.go` files only unit-test `Extract` against hand-built structs.

### Minor
- `configuration/registry.go` implements the exact `cache.go`/`GetCache()` singleton shape described in file-responsibilities.md but is named/placed as `registry.go`/`GetRegistry()`.

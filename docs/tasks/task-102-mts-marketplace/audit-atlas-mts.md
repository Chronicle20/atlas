# Backend Audit — atlas-mts (full-service)

- **Service Path:** services/atlas-mts/atlas.com/mts
- **Guidelines Source:** backend-dev-guidelines skill (DOM-*/SUB-*/EXT-*/SEC-*)
- **Date:** 2026-07-10
- **Scope:** FULL SERVICE — every package audited, not just changed surface. Supersedes the earlier changed-surface pass.
- **Build:** PASS (`go build ./...` exit 0)
- **Vet:** PASS (`go vet ./...` exit 0)
- **Tests:** PASS — all packages green, `go test ./... -count=1` exit 0 (no package exceeds 0.18s, so no unstubbed 42s Kafka-emit stalls)
- **Overall:** NEEDS-WORK (build+tests green; 2 Important latent defects; several Minor deviations)

## Build & Test Results (verbatim summary)

```
ok  atlas-mts/bid 0.021s          ok  atlas-mts/listing 0.177s
ok  atlas-mts/configuration 0.005s ok  atlas-mts/serial 0.033s
ok  atlas-mts/holding 0.066s      ok  atlas-mts/task 0.027s
ok  atlas-mts/kafka/consumer/custody 0.079s  ok atlas-mts/testsupport 0.115s
ok  atlas-mts/kafka/consumer/mts 0.069s      ok atlas-mts/transaction 0.018s
ok  atlas-mts/wallet 0.027s       ok  atlas-mts/wish 0.072s
? (no test files): kafka/message/*, kafka/producer/*, kafka/consumer, logger, rest, saga, test, main
```

---

## Findings by severity

### Critical
None newly found in scope.

Out-of-scope caveat (explicitly excluded per audit brief — pre-existing platform bug, fixed elsewhere in task-119): the entire money path's "in ONE local DB transaction / can never half-complete" guarantees — `listing.transitionToSellerHolding` (Cancel/Expire), `listing.SettleMove`, `listing.PlaceBid`, `listing.Accept`, `holding.Release`/`RestoreHolding`, and the `serial.Next` co-commit — all wrap in `database.ExecuteTransaction`, which is the known no-op primitive (`bug_execute_transaction_noop`). Until task-119 lands, those atomicity protections are documented but not actually in force. Noted, not raised as a new Critical.

### Important
1. **bid/administrator.go:83 — struct-based WHERE on the auction escrow state machine (documented anti-pattern).**
   `UpdateState` builds its predicate as `Where(&entity{Id: parseId(id), State: string(from)})`. This is the exact anti-pattern the guidelines call out (anti-patterns.md:24 "Using struct-based WHERE after removing TenantId — GORM skips zero-value fields"). If `parseId(id)` returns `uuid.Nil` (malformed id), GORM elides the `Id` predicate and the UPDATE degrades to a tenant-wide `state=from → state=to` transition of **every** bid in that state — an escrow-state corruption on the money path. Every sibling administrator does this safely: `listing.UpdateState` (administrator.go:152-168), `holding.SoftDelete`/`Restore` (administrator.go:121-148), and `wish` all guard `uuid.Nil` explicitly and use a map-keyed WHERE. bid is the sole unguarded, struct-condition case. Current production callers (`heldBidFor` results in listing/processor.go, guarded by `!= uuid.Nil`) never pass a nil id, so it is **latent**, but it is a footgun on the auction bid/escrow state transitions. Fix: mirror `listing.UpdateState` (nil guard + `Where(map[string]interface{}{"id":..., "state":...})`).

2. **holding/processor.go:33 — take-home saga per-step timeout (1s) contradicts the codebase's own documented incident.**
   `takeHomeSagaPerStepTimeout = 1 * time.Second`. The sibling list/buy flows in listing/processor.go:143-151 document exactly why 1s/step is wrong: "an observed MTS buy had a single wallet-credit step take ~11s, tripping the old 1s/step budget and firing compensation while the step was still in flight" — and they moved to 15s/step. The WithdrawFromMts saga (2 expanded steps) here gets `base 10s + 2×1s = 12s` total; under the same stressed-broker condition a legitimate take-home can time out and be compensated (the released holding is re-created, the player's take-home spuriously "fails"). Not money-loss (compensation restores the holding), but a reliability regression of the exact `bug_preset_creation_saga_flat_timeout` family the rest of the service already fixed. Fix: scale the per-step budget to 15s (or justify the divergence in-code).

### Minor / informational
- **DOM-02 / DOM-03 deviation (all 5 domains):** no package defines `func (m Model) ToEntity()` or `func Make(Entity)`. Entity↔model mapping is centralized in the documented alternative `modelFromEntity(e entity) (Model, error)` (provider.go) plus explicit entity assembly in the `CreateXxx` administrators. file-responsibilities.md sanctions `modelFromEntity` in its provider.go section, so the mapping goal is met — but the DOM-02/03 named-convention is not followed. Consistent across bid/holding/listing/transaction/wish; non-blocking.
- **DOM-05 deviation:** no `TransformSlice` function anywhere; list handlers use `model.SliceMap(Transform)(model.FixedProvider(ms))(model.ParallelMap())()` inline (listing/resource.go:257, wish/resource.go:62/89, transaction/resource.go:44). Functionally equivalent (no raw loop), but not the named function DOM-05 expects.
- **saga/model.go:10-39 — type-alias re-exports** of atlas-saga types (`type Saga = sharedsaga.Saga`, etc.), which anti-patterns.md discourages ("Type aliases for library migrations"). This mirrors the accepted atlas-character-factory convention and is for local saga-construction ergonomics (not a migration leaving dead aliases), so low risk.
- **configuration/registry.go — no TTL / invalidation on the per-tenant config cache** (loads once per tenant per process lifetime). A live MTS-config change requires a pod restart to take effect. cache.md recommends TTL-based expiration. Degrades gracefully to `DefaultConfig()` on a fetch miss, so not a correctness risk, only a staleness one.
- **listing/processor.go:387 and listing/custody.go:419 — `uuid.MustParse(...)`** on the listing-id string (panics on malformed input). Unreachable in practice (a malformed id fails the earlier `GetById` lookup before reaching these lines, and REST/Kafka ids are validated upstream), but a raw panic primitive inside a processor/consumer path is a latent crash-on-bad-input; prefer `uuid.Parse` with an error return.
- **Shallow builder invariants:** every `Build()` validates only `tenantId != uuid.Nil`; business invariants (non-empty saleType/state, positive prices) are enforced in the List/Buy/PlaceBid processor flows instead. Acceptable per the layering, but the builder's "invariant enforcement" is thin.

---

## Per-package coverage & DOM checklist

### Domain packages (have `model.go`) — bid, holding, listing, transaction, wish

| ID | Check | bid | holding | listing | transaction | wish |
|----|-------|-----|---------|---------|-------------|------|
| DOM-01 | builder.go + Build() validation | PASS builder.go:68 | PASS builder.go:200 | PASS builder.go:299 | PASS builder.go:75 | PASS builder.go:91 |
| DOM-02 | `ToEntity()` | DEV (uses modelFromEntity) | DEV | DEV | DEV | DEV |
| DOM-03 | `Make(Entity)` | DEV (modelFromEntity+CreateBid) | DEV | DEV | DEV | DEV |
| DOM-04 | `Transform` in rest.go | N/A (no REST) | PASS rest.go:83 | PASS rest.go:123 | PASS rest.go:33 | PASS rest.go:37 |
| DOM-05 | `TransformSlice` | N/A | DEV (SliceMap inline) | DEV | DEV | DEV |
| DOM-06 | Processor takes FieldLogger | PASS processor.go:29 | PASS processor.go:81 | PASS processor.go:287 | PASS processor.go:26 | PASS |
| DOM-07 | handlers pass d.Logger() | N/A | PASS | PASS | PASS | PASS |
| DOM-08 | POST/PATCH RegisterInputHandler | N/A (no POST) | PASS resource.go:31/35 | PASS resource.go:34/38 | N/A (read-only) | PASS resource.go:30/34 |
| DOM-09 | Transform errors handled | N/A | PASS | PASS (resource.go:257,310) | PASS | PASS |
| DOM-10 | test DB tenant callbacks | PASS test/database.go:28 | PASS | PASS | PASS | PASS |
| DOM-11 | providers lazy (Query/SliceQuery) | PASS provider.go | PASS | PASS provider.go:12-14 | PASS | PASS |
| DOM-12 | no os.Getenv in handlers | PASS | PASS | PASS | PASS | PASS |
| DOM-13 | no cross-domain logic in handlers | PASS | PASS | PASS | PASS | PASS |
| DOM-14 | handlers call processors not providers | N/A | PASS | PASS | PASS | PASS |
| DOM-15 | no db.Create/Save/Delete in resource.go | PASS | PASS | PASS | PASS | PASS |
| DOM-16 | administrator.go for writes | PASS | PASS | PASS | PASS | PASS |
| DOM-17 | domain error → HTTP status | N/A | PASS | PASS (400/403/404/409/500) | PASS | PASS (400/404/500) |
| DOM-18 | JSON:API GetName/GetID/SetID | N/A | PASS | PASS rest.go:72-83 | PASS | PASS |
| DOM-19 | flat request models | N/A | PASS (TakeHomeRestModel) | PASS (CreateListingRestModel) | N/A | PASS |
| DOM-20 | table-driven tests | PASS | PASS | PASS | PASS | PASS |

Service-wide DOM checks (apply across all packages):
- **DOM-21 (atlas-constants reuse): PASS** — no locally-redefined id/type; uses `world.Id`, `inventory.TypeFromItemId`, `item.Id` (listing/custody.go:109, serial/serial.go, transaction, holding). No item-id classification or inventory-type enum reinvented.
- **DOM-22 (Dockerfile lib blocks): PASS** — repo uses the shared root `Dockerfile` (ARG SERVICE) model, not a per-service Dockerfile. atlas-mts is enumerated in docker-bake.hcl:73 and .github/config/services.json:305; every direct-require lib (constants/database/kafka/model/rest/saga/service/tenant/tracing) is a pre-existing shared lib present in the root Dockerfile — atlas-saga confirmed at Dockerfile lines 44/73/93 and go.work:16.
- **DOM-23 (Kafka topic naming/config): PASS** — topics `COMMAND_TOPIC_MTS`, `COMMAND_TOPIC_MTS_CUSTODY`, `EVENT_TOPIC_MTS_STATUS`, `EVENT_TOPIC_MTS_CUSTODY_STATUS`, `COMMAND_TOPIC_SAGA` all resolved via `topic.EnvProvider`, all present in deploy/k8s/base/env-configmap.yaml with `KEY:"KEY"` shape (lines 52,53,131,132,68), and deploy/k8s/base/atlas-mts.yaml consumes via `envFrom: configMapRef: atlas-env` with NO literal topic override. UPPER_SNAKE, no dotted-lowercase, no versioned suffixes.
- **DOM-24 (Kafka producer stubbed in emit tests): PASS (substance)** — no test package stalls (all <0.18s), so no unstubbed 42s emit path. Emit paths are stubbed by injection rather than the shared `producertest` package: listing/holding processors expose `WithSagaEmitter(...)` and tests inject a capturing stub; consumer handlers take a `pf providerFn` parameter and tests pass a recording/no-op `producer.Provider` (kafka/consumer/custody/consumer_test.go `recordingProducer`). NOTE: the service does not use `producertest.InstallNoop()` — the shared-package purity DOM-24(d) prefers — but the acceptable "per-test injection of a no-op producer.Provider" arm is satisfied.

### Sub-domain / support packages

| Package | Purpose | Notes |
|---------|---------|-------|
| wallet/ | EXT cross-service client (reads atlas-cashshop wallet) + read-only REST passthrough | EXT-01 PASS (flat RestModel, no relationships block — comment wallet.go:27-28); EXT-02 PASS (httptest.NewServer in wallet_test.go:16,40); EXT-03 PASS (`errors.Is(err, requests.ErrNotFound)` wallet.go:116); EXT-04 PASS (`requests.RootUrl("CASHSHOP")` wallet.go:52). Resource GET uses `rest.RegisterHandler`; no direct entity writes. |
| configuration/ | Per-tenant MTS config singleton registry + atlas-tenants REST fetch | Singleton via sync.Once (registry.go:31); EXT client `requests.RootUrl("TENANTS")` (requests.go:16); graceful DefaultConfig fallback. Minor: no cache TTL. |
| saga/ | atlas-saga type re-exports + saga emitter Processor + command producer | Producer uses `producer.ProviderImpl(l)(ctx)` with span+tenant decorators (producer.go:14-15 via kafka/producer). Minor: type-alias re-exports. |
| serial/ | Per-(tenant,world) monotonic ITC counter | Correct: seed-on-conflict + atomic `next_serial + 1` under row lock, all map-keyed WHERE (world 0 safe). Must run inside caller tx — documented. |
| task/ | DB-driven auction-expiration sweep ticker | Cross-tenant via `database.WithoutTenantFilter`; tenant taken from each listing row; batch-capped (500) with deferred-tail logging; SettleAuction/Expire share the atomic transitions. Clean. |
| kafka/message/{mts,custody,saga} | topic constants + command/event envelopes | Semantic `ReasonKey` failure keys (DOM-25 compliant — no client wire bytes emitted); distinct JSON tag `reasonKey` to avoid the shared-topic string/number unmarshal collision. |
| kafka/producer/{,mts,custody} | context-decorated producers | `ProviderImpl` builds SpanHeaderDecorator + TenantHeaderDecorator per ctx (producer.go). |
| kafka/consumer/{mts,custody} | command consumers, saga step acks, failure notices | Thin handlers delegate business logic to processors; own only acks/emits; atomic `msg.Emit(buf)`; ERROR ack on failure drives saga compensation; per-context producer for correct headers. Layer separation clean. |
| rest/ | service-local RegisterHandler/RegisterInputHandler + path parsers | Wraps `server.ParseTenant` + `jsonapi.Unmarshal`; automatic tenant/span; no manual JSON envelope in domain resources. Equivalent to server.* helpers. |
| testsupport/ | env-gated (`MTS_TEST_ROUTES_ENABLED`) e2e test routes | Never routed through ingress; mirrors real channel command providers. main.go:105 gates registration. |
| logger/, test/, main.go | app wiring | main.go clean: `database.Connect(SetMigrations(...))` (no RegisterTenantCallbacks in main — correct), consumer registration, REST routes, teardown, sweep ticker. |

### Security (SEC-*)
atlas-mts is not an auth/token service (no JWT parsing, no OAuth callback/redirect). SEC-01/02/03 N/A. SEC-04 (no hardcoded secrets): PASS — DB creds via `db-credentials` secretKeyRef (atlas-mts.yaml:33-42); no keys/passwords in source. Input validation: REST handlers validate path/query params (ParseWorldId/CharacterId/AccountId/ListingId, explicit uuid.Parse guards on delete paths e.g. wish/resource.go:146), server-authoritative price-floor / active-cap / auction-duration / owner-checks enforced in the List/Cancel/Buy/Bid flows (never trusted from the wire body). No SQL injection surface (all queries parameterized via GORM `?`/map-keyed WHERE).

---

## Summary

### Blocking (must fix)
None — build and tests are green with zero hard failures.

### Should fix (Important)
- bid/administrator.go:83 — replace struct-based `Where(&entity{...})` with a `uuid.Nil`-guarded map-keyed WHERE (mirror listing.UpdateState); the current form can degrade a bad id into a tenant-wide escrow-state rewrite.
- holding/processor.go:33 — raise `takeHomeSagaPerStepTimeout` from 1s to 15s (or justify), matching the list/buy fix for the same documented broker-stress compensation-in-flight incident.

### Non-blocking (Minor)
- DOM-02/03 (ToEntity/Make absent — modelFromEntity used instead), DOM-05 (TransformSlice absent — SliceMap inline), saga type-alias re-exports, configuration registry lacks TTL, `uuid.MustParse` panic primitives (listing/processor.go:387, listing/custody.go:419), shallow builder invariants.

## Final resolution (full-service audit fixes)

- **Important #1 (bid UpdateState struct-condition WHERE) — FIXED.** `bid/administrator.go` `UpdateState` now guards `uuid.Nil` and uses a map-keyed WHERE (`{"id": bid, "state": from}`), matching the listing/holding/wish siblings — a malformed id errors instead of degrading to a tenant-wide escrow-state rewrite. Locked by `TestAdministratorUpdateStateRejectsMalformedId` (asserts error + zero rows touched + both seeded bids stay Held).
- **Important #2 (take-home flat 1s/step saga timeout) — FIXED.** `holding/processor.go` `takeHomeSagaPerStepTimeout` raised 1s → 15s to match the list/buy flows (bug_preset_creation_saga_flat_timeout family); take-home no longer spuriously compensates under broker stress.
- **Minors — DEFERRED (convention-consistent / low-risk / documented).** ToEntity/Make naming, no TransformSlice, saga type-alias re-exports, configuration cache TTL, two unreachable `uuid.MustParse` primitives, shallow builder invariants.
- **ExecuteTransaction no-op (out of scope)** — pre-existing platform bug (task-119); the money-path atomicity guarantees depend on it landing.

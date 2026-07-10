# Plan Audit — task-102-mts-marketplace

**Plans audited:** `plan.md` (main), `plan-e2e-testing.md`, `design-wanted-offers.md`
**Audit Date:** 2026-07-10
**Branch:** task-102-mts-marketplace
**Base (merge-base):** 6c6f52abfcdb9915257c9dfd68a27ddaf99dfd06
**HEAD:** e3696ccf22f37e33017309492ff2604fd5c5b2b5 (178 commits, 616 files)

## Executive Summary

The planned scope is present and faithfully implemented across all three plans. Every
phase of `plan.md` (0–10), every task of `plan-e2e-testing.md` (1–8), and the
`design-wanted-offers.md` offer/accept loop landed with source evidence. All affected
modules build and test green: atlas-mts, atlas-saga-orchestrator, atlas-channel,
atlas-tenants (build), and atlas-ui (MTS tests 12/12). No `// TODO`, stubbed handler,
501, or saga action lacking its completion/compensation consumer was found in the new
MTS code. The branch went well beyond the plan (ITC serial domain, transaction/history,
wallet passthrough, cart/wanted channel packages, live-testing fixes) — expected extra
work, not a defect.

## Task Completion — plan.md

| Phase / Task | Status | Evidence |
|---|---|---|
| 0.1–0.3 Serverbound packet verification + design notes | DONE | STATUS.md L663 ENTER_MTS, L775 ITC_STATUS_CHARGE, L776 ITC_QUERY_CASH_REQUEST, L777 ITC_OPERATION, L450/452 MTS_OPERATION2/MTS_OPERATION — all ✅ across gms_83/84/87/95/jms (MTS_OPERATION* ⬜ for jms per §9.4). design.md L67–81 records §9.1 (escrow-default) + §9.4 jms surface |
| 1.1 Module scaffold + registration | DONE | go.work:61, .github/config/services.json:305, docker-bake.hcl:73, logger/init.go, go.mod |
| 1.2 Test harness (no testhelpers) | DONE | test/database.go, test/tenant.go, test/processor.go |
| 1.3–1.5 Listing model/builder/entity/provider/administrator/processor | DONE | listing/{model,builder,entity,provider,administrator,processor}.go + tests |
| 1.6 Holding, Bid, Wish domains | DONE | holding/, bid/, wish/ full model→processor sets |
| 1.7 REST reads + wish CRUD | DONE | listing/resource.go, holding/resource.go, wish/resource.go, rest/handler.go |
| 1.8 Config registry | DONE | configuration/{registry,model,requests}.go + registry_test.go |
| 1.9 main.go + k8s manifest | DONE | main.go, deploy/k8s/base/atlas-mts.yaml |
| 2.1 Saga Action constants + payloads | DONE | libs/atlas-saga/model.go:22,130–133 (MtsOperation type, 7 actions), payloads + unmarshal_test.go |
| 2.2 atlas-mts custody consumer + events | DONE | kafka/message/custody/kafka.go, kafka/consumer/custody/consumer.go (+ dupe_safety_test.go) |
| 2.3 Orchestrator handlers/expansion/acceptance/compensation | DONE | saga/handler.go:831–842, event_acceptance.go:61–64,163–170,297–300, compensator.go:204+, mts/processor.go, mts_expansion_test.go, mts_integration_test.go |
| 3.1 COMMAND_TOPIC_MTS consumer + status producer | DONE | kafka/message/mts/kafka.go, kafka/consumer/mts/consumer.go, kafka/producer/mts/producer.go; main.go:73 |
| 4.1 List flow (TransferToMts + fee + floor/cap/duration) | DONE | listing/list_flow_test.go, listing/processor.go, saga/builder.go |
| 4.2 Cancel (local active→holding) | DONE | listing/cancel_test.go, cancel_resource_test.go |
| 4.3 Take-home (WithdrawFromMts idempotent) | DONE | holding/resource.go:20 route, take_home_flow_test.go |
| 4.4 Expiration ticker | DONE | task/periodic.go + periodic_test.go; main.go:85 |
| 4.5 Channel ENTER_MTS + ITC list/cancel/take-home arms | DONE | socket/handler/mts_entry.go, itc_operation.go (all arms) |
| 5.1 Buy/buy-now (MtsSettlePurchase) | DONE | listing/buy_flow_test.go, processor.go |
| 5.2 Dupe-safety suite | DONE | listing/dupe_safety_test.go, kafka/consumer/custody/dupe_safety_test.go, saga/mts_dupe_safety_test.go |
| 5.3 Channel buy arm + wallet query | DONE | itc_operation.go BUY arm, itc_query_cash_request.go, itc_status_charge.go, wallet/resource.go |
| 6.1 Bid escrow + outbid release + settle-at-expiry | DONE | listing/auction_bid_flow_test.go, task/periodic.go settle branch |
| 6.2 Channel bid arm (+ conditional push) | DONE | itc_operation.go PLACE_BID/BUY_AUCTION_IMM; live-push correctly omitted (escrow default) per design.md L298–305 |
| 7.1 Wish CRUD + buy-from-wish + channel arms | DONE | wish/ domain, itc_operation.go SET/DELETE/VIEW/BUY/CANCEL_WISH arms |
| 8.1 mts-configs JSONB resource | DONE | configuration/{rest,processor,resource,kafka,provider,seed,mock}.go, rest/handler.go ParseMtsConfigId |
| 8.2 Per-version socket + operations tables (5 templates) | DONE | all 5 templates: EnterMtsHandle/ItcStatusChargeHandle/ItcQueryCashRequestHandle/ItcOperationHandle each ×1, validators (LoggedInValidator + NoOpValidator on bodiless), per-version operations tables, MtsOperation/MtsOperation2 writers (MtsOperation2 absent for jms per §9.4) |
| 8.3 Rollout note | DONE | rollout-checklist.md |
| 9.1 mts-config service client + Zod | DONE | services/api/mts-config.service.ts, lib/schemas/mts-config.schema.ts + tests |
| 9.2 Tenant config page | DONE | pages/TenantsMtsConfigPage.tsx, tenants-mts-config-form.tsx |
| 9.3 Read-only listings browser | DONE | pages/MarketplacePage.tsx, services/api/mts-listings.service.ts |
| 10.1–10.2 Verification + review | DONE | audit-backend.md, audit-frontend.md present; build/test green (below) |

## Task Completion — plan-e2e-testing.md

| Task | Status | Evidence |
|---|---|---|
| 1 listing.BackdateEndsAt | DONE | listing/administrator.go, administrator_test.go |
| 2 testsupport buy/bid command providers | DONE | testsupport/producer.go + producer_test.go |
| 3 Seed endpoint | DONE | testsupport/rest.go, resource.go, resource_test.go |
| 4 Expire + sweep endpoints | DONE | testsupport/resource.go handlers |
| 5 Simulated purchase + bid endpoints | DONE | testsupport/resource.go, simulate_test.go |
| 6 Env-gated main.go registration | DONE | main.go:105–107 (MTS_TEST_ROUTES_ENABLED gate) |
| 7 E2E playbook doc | DONE | e2e-test-playbook.md |
| 8 Full-module verification | DONE | tests green |

## Task Completion — design-wanted-offers.md

| Item | Status | Evidence |
|---|---|---|
| SaleTypeOffer + offer_wish_serial/owner_id columns | DONE | listing/entity.go:88–89, model builder, provider.go:61–63 (OfferWishSerial/ExcludeOffers filters) |
| TransferToMts offer (fee 0) + AcceptToMtsListing carries SaleType/offer fields | DONE | libs/atlas-saga payloads, orchestrator mts/processor.go |
| Channel SALE_CURRENT_ITEM→offer, VIEW_WISH→offers-by-wish, BUY_WISH→Buy-as-poster | DONE | itc_operation.go buildCreateListingFromSaleCurrentItem, resolveWantedPrice; mts/wanted/, mts/wish/ |
| Public browse excludes offers; Not-Yet-Sold includes | DONE | provider.go ExcludeOffers flag |

## Build & Test Results

| Module | Build | Tests | Notes |
|---|---|---|---|
| atlas-mts | PASS | PASS | `go build ./...` exit 0; `go test ./...` all packages ok |
| atlas-saga-orchestrator | PASS | PASS | saga + party_quest + mocks ok |
| atlas-channel | PASS | PASS | mts/... + socket/handler/... ok |
| atlas-tenants | PASS | (not run) | `go build ./...` exit 0 |
| atlas-ui | n/a | PASS | mts-config + mts-listings service tests 12/12 (node v22) |

## Stub / TODO scan

No MTS-related stubs. The only `TODO` markers in changed non-test source are:
- `socket/handler/cash_shop_entry.go` (5) — pre-existing copied idiom (event/mini-dungeon gates), not MTS.
- `channel/main.go:275` — pre-existing session-lookup note, not MTS.
- `tools/packet-audit/cmd/run.go:2498` — an explanatory comment, not a stub.

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

None required. Optional (non-blocking): mts-configs seed JSON is loaded at runtime from
the mounted `/configurations/mts-configs` volume (same pattern as routes/vessels), so no
in-repo seed file exists — confirm the deploy overlay mounts a default `mts-configs`
document before enabling for existing tenants (already covered by rollout-checklist.md).

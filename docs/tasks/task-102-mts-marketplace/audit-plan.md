# Plan Audit — task-102-mts-marketplace

**Plan Path:** docs/tasks/task-102-mts-marketplace/plan.md
**Audit Date:** 2026-06-18
**Branch:** task-102-mts-marketplace
**Base Branch:** main (BASE eed47d480 → HEAD 87dfa758a, 58 commits)

## Executive Summary

The plan was implemented faithfully and end-to-end. All ten phases are present with
file:line and test evidence; the hard Phase-0 packet gate is fully promoted (all four
serverbound MTS packets ✅ across all five versions, 22 byte fixtures per version, one
per ITC arm — no enumeration-only false pass). Every changed Go module builds, vets,
and tests `-race` clean (atlas-mts, atlas-saga, atlas-saga-orchestrator, atlas-channel,
atlas-tenants); `packet-audit matrix --check` exits 0; atlas-ui MTS tests pass 11/11.
No `// TODO`, stub handler, 501, or silently-unrouted command was found in landed code.

Two genuine (small) gaps, neither a silent stub:
1. **Task 5.3 Step 3** — the planned `GET /characters/{characterId}/mts/wallet` REST
   passthrough was **not built**. The in-game wallet query is fully functional via the
   socket path (`ITC_QUERY_CASH_REQUEST` → cashshop wallet processor → `MTS_OPERATION2`),
   so no behavior is missing, but the specific REST route the plan named does not exist.
2. **Task 4.5 Step 1** — `ENTER_MTS` (`EnterMtsHandleFunc`) implements the level gate and
   wallet announce, but the plan's "leave channel/map, mark entered, announce initial
   browse page + active listings + holding" is a clearly-marked SEAM
   (`mts_entry.go:66-73`) deferred to the browse/listing/holding arm wiring. Partial, and
   explicitly documented rather than silently skipped.

All six listed accepted deviations are real and grounded with code comments. The
redis-key-guard FAIL is a pre-existing baseline (identical FAIL on `main`); atlas-mts
uses no redis at all.

## Task Completion

| Phase / Task | Status | Evidence / Notes |
|---|---|---|
| **0.1–0.2** Serverbound packet verification (HARD GATE) | DONE | STATUS.md: ENTER_MTS / ITC_STATUS_CHARGE / ITC_QUERY_CASH_REQUEST / ITC_OPERATION all ✅ for v83/v84/v87/v95/jms. 22 serverbound MTS byte fixtures per version under `docs/packets/evidence/<v>/` (18 ITC arms + 3 standalone + tab-search). Codecs in `libs/atlas-packet/field/serverbound/{enter_mts,itc_operation,itc_query_cash_request,itc_status_charge}.go` + `_test.go`. |
| **0 (v84 opcode corrections)** | DONE | Commits `3cbb993a6` (ITC_STATUS_CHARGE 0xFB→0x102), `0b6a28595` (ITC_QUERY 0xFC→0x103), `1651a9472` (ITC_OPERATION 0xFD→0x104); registry-grounded against IDA (csv-carryover-stale fix), evidence YAMLs present. |
| **0.3** §9.1 / §9.4 decision note | DONE | `design.md:539-584` "Phase 0 decision note": §9.1 = NO server-push → escrow-at-expiry; §9.4 = jms has serverbound opcodes, clientbound ⬜. Commit `9524669ab`. |
| **1.1** Module scaffold + registration | DONE | `services/atlas-mts/.../go.mod:1` (`module atlas-mts`); `go.work:60`; `.github/config/services.json:297`; `docker-bake.hcl:72`. |
| **1.2** Test harness (sqlite, tenant ctx) | DONE | `test/database.go`, `test/tenant.go`, `test/processor.go` (no `*_testhelpers.go`). |
| **1.3–1.5** Listing model/builder/entity/provider/administrator/processor | DONE | `listing/model.go:11-26` enums; `builder.go:280-283` validates `tenantId!=Nil`; `entity.go:42-51,85,94` surrogate UUID PK + (tenant_id,id) unique + 3 design indexes + explicit equip columns + TableName "listings"; `administrator.go:149` conditional `WHERE state=from` returning RowsAffected; `processor.go:215` `NewProcessor(l,ctx,db)`. |
| **1.6** Holding / Bid / Wish domains | DONE | Holding soft-delete `holding/entity.go:77`,`administrator.go:120`; Bid `entity.go:33-34` + `model.go:13-15` (held/released/won); Wish `entity.go:28-29` (characterId+itemId). Each has Migration. |
| **1.7** REST reads + wish CRUD | DONE | `listing/rest.go:62` GetName "listings"; `resource.go:33-36` GET browse + detail; `holding/resource.go`, `wish/resource.go` GET + wish POST/DELETE; `rest/handler.go` parsers. |
| **1.8** Config registry | DONE | `configuration/registry.go:19-23,61-83` lazy RWMutex double-check, default-on-miss; `model.go:133-145` defaults 5000/0.10/10/10/24/168/110/16/1; `requests.go:24` fetches `mts-configs`. |
| **1.9** main.go + k8s manifest | DONE | `main.go:25,60-65,90-92`; `deploy/k8s/base/atlas-mts.yaml` + `kustomization.yaml:43`. |
| **2.1** Saga actions + payloads | DONE | `libs/atlas-saga/model.go:22,130-136` (MtsOperation type + 7 actions); `payloads.go:563+` structs; `unmarshal.go:366-403` registered. |
| **2.2** Custody consumer + events | DONE | `kafka/consumer/custody/consumer.go:82,156,166` idempotent accept/release; `message/custody/kafka.go:13,114` topic consts. |
| **2.3** Orchestrator dispatch/expansion/acceptance/compensation | DONE | `mts/processor.go:70-76`; `saga/processor.go:1421` (TransferToMts→[release,accept]), `:1489` (Withdraw), `:1583-1619` (Settle debit-first); `handler.go:831-842`; `event_acceptance.go:60-63,166-168`; `compensator.go:1215-1274`. Timeouts explicit base+perStep*N (`listing/processor.go:473,580,725,841`). |
| **3.1** COMMAND_TOPIC_MTS consumer + EVENT_TOPIC_MTS_STATUS | DONE | `kafka/message/mts/kafka.go:13,168`; `consumer/mts/consumer.go:137,346,383`; `main.go:67-77` registers. |
| **4.1** List flow (TransferToMts + fee + floor/cap/duration) | DONE | `listing/processor.go:397-465` (floor 110 reject, maxActive reject, 24–168h reject, preallocate id, AwardMesos(-fee), TransferToMts); POST route `resource.go:34`; `list_flow_test.go`. |
| **4.2** Cancel (race-safe local transition) | DONE | `processor.go:285-378` single-tx active→seller-holding conditional; DELETE seller-only `resource.go:36,91`; `cancel_test.go`. |
| **4.3** Take-home (WithdrawFromMts, idempotent) | DONE | `holding/processor.go:121-164`; POST `/characters/{id}/mts/holding/{hid}/take-home`; `take_home_flow_test.go`. |
| **4.4** Expiration ticker (DB-driven) | DONE | `task/periodic.go:32-180` ticker+stopCh+wg, WithoutTenantFilter enumerate, bounded `sweepBatchLimit=500`, logged; `main.go:81-83`; `periodic_test.go`. |
| **4.5** Channel ENTER_MTS + list/cancel/take-home arms | PARTIAL | ITC_OPERATION dispatcher + list/cancel/take-home arms DONE (`itc_operation.go:301-433`, byte fixtures). ENTER_MTS (`mts_entry.go`) does level-gate + wallet announce only; browse/listings/holding announce + leave-channel/mark-entered is a documented SEAM (`:66-73`). |
| **5.1** Buy / buy-now (MtsSettlePurchase) | DONE | `listing/processor.go:542-575` markedUp + prepaid reject + debit-first + commission sink; `buy_flow_test.go`. |
| **5.2** Dupe-safety suite | DONE | `listing/dupe_safety_test.go` + `kafka/consumer/custody/dupe_safety_test.go` (replay, cancel-vs-settle, take-home replay); orchestrator `saga/mts_integration_test.go` reverse-walk. |
| **5.3** Channel buy arm + wallet query | PARTIAL | Buy/buy-now arms `itc_operation.go:338-355`; `ITC_QUERY_CASH_REQUEST`→2-bucket→MTS_OPERATION2 `itc_query_cash_request.go:21-38`; bodiless `ITC_STATUS_CHARGE` NoOpValidator `itc_status_charge.go`. **Gap:** planned `GET /characters/{id}/mts/wallet` REST passthrough not built (socket path is functional). |
| **6.1** Bid escrow + outbid release + settle-at-expiry | DONE | `listing/processor.go:642-847` (floor, escrow marked-up, CAS row-lock, outbid release, settle credits seller + custody→winner, no-bids→seller); ticker settle branch `task/periodic.go:134-167`; `auction_bid_flow_test.go`. |
| **6.2** Channel bid arm; NO live push | DONE | PLACE_BID + BUY_AUCTION_IMM arms `itc_operation.go:346-365` + fixtures. Confirmed no live-outbid-push path (per §9.1) — zero "outbid" matches in atlas-channel. |
| **7.1** Wish CRUD + buy-from-wish + channel arms | DONE | `wish/processor.go` + `resource.go` CRUD; channel zzim/wish arms `itc_operation.go:366-430`; buy-from-wish routes resolved serial into shared `Buy()` (`:419,:428`) — DONE as routing decision (no separate path needed). |
| **8.1** mts-configs JSONB resource | DONE | `configuration/{rest,processor,resource,kafka,provider,seed}.go`, `rest/handler.go:48`, `mock/processor.go:366-490` — all touch-points present; routes incl. `/seed`. |
| **8.2** Per-version socket + operations seeds (5 templates) | DONE | All 4 handlers in all 5 templates with correct per-version opcodes + validator on every entry (NoOpValidator only on ITC_STATUS_CHARGE); identical 17-entry operations table per version; MTS_OPERATION/MTS_OPERATION2 writers in 4 gms versions (jms clientbound absent per design). Crash-risk area clean. |
| **8.3** Rollout checklist | DONE | `docs/tasks/task-102-mts-marketplace/rollout-checklist.md`. |
| **9.1** mts-config UI client + Zod | DONE | `services/api/mts-config.service.ts`, `lib/schemas/mts-config.schema.ts`, `__tests__/mts-config.service.test.ts`. |
| **9.2** Tenant config page | DONE | `pages/TenantsMtsConfigPage.tsx` + `tenants-mts-config-form.tsx` (react-hook-form+Zod); routed `App.tsx:126`. |
| **9.3** Read-only listings browser | DONE | `pages/MarketplacePage.tsx` + `services/api/mts-listings.service.ts`; routed `App.tsx:98`. |
| **10.1** Verification gates | DONE (with caveats) | See Build & Test Results. |

**Completion Rate:** ~30/32 plan units fully DONE; 2 PARTIAL.
**Skipped without approval:** 0
**Partial implementations:** 2 (Task 4.5 ENTER_MTS announce-seam; Task 5.3 wallet REST route)

## Skipped / Deferred Tasks

- **Task 5.3 Step 3 — `GET /characters/{characterId}/mts/wallet` REST passthrough (missing).**
  The plan explicitly lists this route. It does not exist in `holding/resource.go` or any
  atlas-mts resource (`atlas-mts/wallet/wallet.go` is a REST *client* of cashshop, not a
  server route). Impact: **low/none for gameplay** — the in-game wallet query is satisfied
  entirely on the socket path. Impact is only that an external/UI consumer cannot read the
  MTS wallet via atlas-mts REST. This was producible (a thin passthrough handler) and was
  not built; flagging as a real, if minor, plan-adherence gap.

- **Task 4.5 Step 1 — ENTER_MTS initial-state announce (partial / documented seam).**
  `EnterMtsHandleFunc` (`mts_entry.go`) gates min level and announces the wallet
  (`MTS_OPERATION2`), but does not announce the initial browse page / the character's
  active listings / their holding, nor perform a leave-channel/mark-entered migration.
  These are explicitly marked as a SEAM (`mts_entry.go:66-73`) pending the channel-side
  atlas-mts REST client + REST→MtsItem mapping. Impact: **medium** — a player entering MTS
  sees their wallet but not the marketplace contents until that wiring lands. It is honestly
  documented (not a silent stub), but it is functional scope the plan placed in this task.
  (Note: the plan's "mirror cash_shop_entry leave channel/map" phrasing — `cash_shop_entry.go`
  has no Leave/Migrate call either, so that sub-claim was loosely specified.)

## Build & Test Results

| Module / Area | Build | Vet | Tests (-race) | Notes |
|---|---|---|---|---|
| atlas-mts | PASS | PASS | PASS | All domain + kafka + serial + task + flow suites green |
| libs/atlas-saga | PASS | PASS | PASS | unmarshal/validation green |
| atlas-saga-orchestrator | PASS | PASS | PASS | incl. saga (expansion/compensation/integration) |
| atlas-channel (mts) | PASS | PASS | PASS | mts, mts/listing, mts/wish |
| atlas-tenants | PASS | PASS | PASS | configuration + tenant |
| atlas-ui (MTS) | n/a | n/a | PASS (11/11) | mts-config + mts-listings service tests; TenantsPage flake is unrelated/isolated-green |
| packet-audit `matrix --check` | — | — | PASS (exit 0) | run from worktree root (must NOT pass `GOFLAGS=-mod=mod`; from `tools/packet-audit` cwd it false-reports stale due to relative template/export paths) |
| tools/redis-key-guard.sh | — | — | **FAIL (pre-existing)** | Identical FAIL on `main` base branch; offenders are atlas-merchant/atlas-world/atlas-party-quests etc. atlas-mts uses **no redis** (verified: zero go-redis imports). Not introduced by task-102. |

Docker `buildx bake` was not executed in this audit environment; only atlas-mts's `go.mod`
was added and the design adds **no new shared lib** (no Dockerfile COPY edits required), so
the bake risk surface is minimal. Recommend the executor confirm
`docker buildx bake atlas-mts atlas-saga-orchestrator atlas-channel atlas-tenants` per Task 10.1
Step 4 before merge if not already done.

## Accepted Deviations — verified real and grounded

| Deviation | Grounded at |
|---|---|
| nITCSN ↔ UUID persistent per-world serial scheme | `serial/serial.go` + `listing/entity.go` (4th unique index tenant,world,serial); commit `45eca605d` |
| BUY_ZZIM / BUY_WISH surface as BuyItemDone (cosmetic) | `itc_operation.go:418,427`; `kafka/consumer/mts/consumer.go:183-193` |
| SALE_CURRENT_ITEM carries no price → floor-rejected | `itc_operation.go:184-205` (ListValue:0 + comment) |
| RegisterWishEntry wire price/duration/desc logged-then-dropped | `itc_operation.go:400-407`; wish model stores itemId only (`wish/model.go`) |
| Auction-winner settle surfaces as BuyItemDone (not SuccessBidInfoResult) | `kafka/consumer/mts/consumer.go:183-193` (cosmetic notice follow-up) |
| jms clientbound MTS results version-absent (⬜) | STATUS.md MTS_OPERATION/MTS_OPERATION2 jms ⬜; design §9.4 |

## Overall Assessment

- **Plan Adherence:** MOSTLY_COMPLETE (Phase 0 gate fully honored; 30/32 units DONE; 2 PARTIAL,
  both honestly documented, neither a silent stub or fabricated value)
- **Recommendation:** NEEDS_REVIEW — resolve or explicitly accept the two PARTIAL items below;
  otherwise the branch is in strong shape and the safety-critical core (custody/settlement/
  dupe-safety + Phase-0 packet gate) is complete and tested.

## Action Items

1. **Decide on Task 5.3 `GET /characters/{characterId}/mts/wallet`** — either add the thin REST
   passthrough the plan specified (producible now) or strike it from the plan with a note that
   the socket path supersedes it. Currently silently absent.
2. **Decide on Task 4.5 ENTER_MTS announce-seam** — either land the browse/active-listings/holding
   announce (the clientbound MtsResult* codecs already exist; the missing piece is the channel-side
   atlas-mts REST client + REST→MtsItem mapping) or split it into a tracked follow-up task. As-is, a
   player entering MTS sees only their wallet, not marketplace contents.
3. **Confirm `docker buildx bake atlas-mts atlas-saga-orchestrator atlas-channel atlas-tenants`** green
   from the worktree root (Task 10.1 Step 4) before opening the PR — not run in this audit.
4. **(Informational) redis-key-guard** baseline FAIL is pre-existing and unrelated to task-102; no
   action required for this branch, but note it does not gate clean here.

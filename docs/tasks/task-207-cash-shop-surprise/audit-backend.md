# Backend Audit — task-207-cash-shop-surprise

- **Scope:** Go changes on branch `task-207-cash-shop-surprise`, `1e0a321b8` → `92fddbb61`
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-08-10
- **Build/Test/Vet/Bake/Guards:** Not re-run — verified clean by the requester prior to this audit (per task instructions). This audit is guideline-conformance only.
- **Overall:** NEEDS-WORK (2 Important findings; 1 Minor)

## Modules / packages audited

- `libs/atlas-packet` — `cash/serverbound/item_gachapon_button.go`, `cash/clientbound/item_gachapon_result.go` (new codecs)
- `libs/atlas-rest` — `requests.ErrConflict` sentinel (additive)
- `services/atlas-cashshop/atlas.com/cashshop/surprise` (new domain: `processor.go`, `capacity.go`, `opening/*`) — support/orchestration package, no `model.go`
- `services/atlas-cashshop/atlas.com/cashshop/rewardpool` (new REST-client package) — support package, no `model.go`
- `services/atlas-cashshop/atlas.com/cashshop/configuration` (registry additions), `kafka/message|producer|consumer/cashshop` (additions)
- `services/atlas-channel/atlas.com/channel/cashshop`, `socket/handler/cash_item_gachapon.go`, `kafka/consumer/cashshop`, `kafka/message/cashshop`
- `services/atlas-reward-pools/atlas.com/reward-pools/{gachapon,item,reward}` — `cash-surprise` kind, `commodity_id`, flat-weight selection
- `services/atlas-configurations` — cashshop/surprise RestModels (templates + tenants)

## Findings

### FILE-05 (Important) — `rewardpool` domain `Model` defined in `processor.go`, not `model.go`

`services/atlas-cashshop/atlas.com/cashshop/rewardpool/processor.go:22-30` defines the package's
domain `Model` struct (with `ItemId()`/`Quantity()`/`CommodityId()` accessors) inline in
`processor.go`, alongside the `Processor` interface/impl. File-responsibilities.md assigns
"immutable domain objects with private fields and accessor methods" to `model.go` explicitly.

This is not a "no model.go needed" support package by the letter of the guideline — the package
already has a genuine private-field/accessor domain type, it is simply filed in the wrong place.
The sibling packages in the *same service* that play the identical "thin REST-client, no DB
entity" role do it correctly: `services/atlas-cashshop/atlas.com/cashshop/character/` has a
dedicated `model.go` (11KB) alongside `processor.go`/`rest.go`/`requests.go`, and
`services/atlas-cashshop/atlas.com/cashshop/cashshop/commodity/` (touched by this same branch —
`commodity/model.go` +7 lines) also keeps `Model` in its own `model.go`. `rewardpool` is the one
new package in this diff that collapses the two responsibilities into `processor.go`.

Fix: move `Model`, its accessors, `ErrPoolMissing`/`ErrPoolEmpty`, and `classifySelectError` are
fine to stay (error/business-logic), but the `Model` type belongs in a new `rewardpool/model.go`.

### EXT-01 (Important) — `rewardpool.RewardRestModel` missing JSON:API relationship-interface stubs

`services/atlas-cashshop/atlas.com/cashshop/rewardpool/rest.go:7-22` defines `RewardRestModel`,
decoded via `requests.PostRequest[RewardRestModel]` (`rewardpool/requests.go:21`), which routes
through `unmarshalResponse` → `jsonapi.Unmarshal` (`libs/atlas-rest/requests/response.go:9`).
Per `libs/atlas-rest/CLAUDE.md`, every JSON:API target struct decoded this way must implement
`SetToOneReferenceID` and `SetToManyReferenceIDs`, even as no-ops, because api2go errors on any
response carrying a `relationships` block and the failure surfaces later as a misleading generic
error (task-037, hit twice previously). `RewardRestModel` implements only `GetName`/`GetID`/
`SetID` — the two relationship stubs are absent.

This is currently *safe in practice* only because the atlas-reward-pools server-side
`reward.RestModel` (`services/atlas-reward-pools/atlas.com/reward-pools/reward/rest.go`) never
emits a `relationships` block (no `GetReferences`/`GetReferencedIDs` implemented there either) —
but the guideline requires the client-side stub unconditionally, precisely so a future change on
the server side (e.g. adding a related-resource link) doesn't reintroduce the exact bug task-037
already paid for twice. Compare `services/atlas-cashshop/atlas.com/cashshop/character/rest.go:76-82`,
which already carries the stubs for the same reason and is the correct template to copy.

### Minor — nested nested-transaction call is redundant but not a defect

`services/atlas-cashshop/atlas.com/cashshop/surprise/processor.go:207` (`Open`'s internal
`database.ExecuteTransaction(p.db.WithContext(p.ctx), ...)`) runs one call site: `OpenAndEmit`
(processor.go:73) already wraps the call in its own `database.ExecuteTransaction` and passes the
resulting `tx` into a fresh `NewProcessor(..., tx)`, so `p.db` inside `Open` is already the outer
transaction. `database.ExecuteTransaction` (`libs/atlas-database/transaction.go:9-14`) detects
this via `isTransaction(db)` and short-circuits to `fn(db)` rather than opening a real nested
transaction/savepoint, so there is no functional bug — but the inner `database.ExecuteTransaction`
call is dead code from a transaction-semantics standpoint (it can never take the `db.Transaction`
branch given the only caller). Not scored as a checklist FAIL; noted because the task specifically
flagged transaction discipline as a risk area — the discipline itself is correct (verified below).

## Checklist Results

### `surprise` (support/orchestration package — no `model.go`, no `resource.go`; File Responsibilities Checklist + transaction/multi-tenancy focus)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor in `processor.go` | PASS | `surprise/processor.go:43-70` (`Processor` interface + `ProcessorImpl` + `NewProcessor`) |
| FILE-06 | No catch-all file | PASS | `capacity.go` holds one pure function (`HasRoomForSwap`), not a collapsed responsibility set |
| Multi-tenancy | `tenant.MustFromContext` used | PASS | `surprise/processor.go:63` (`t: tenant.MustFromContext(ctx)`) |
| Multi-tenancy | DB write path tenant-scoped | PASS | `opening.Insert` receives `p.t.Id()` explicitly (create-path convention) — `surprise/processor.go:208`, `opening/entity.go:26` (`TenantId` in ledger PK) |
| Transaction discipline | Writes inside `OpenAndEmit`'s tx use `tx`, not `p.db` | PASS | `surprise/processor.go:226` rebuilds `astP := asset.NewProcessor(p.l, p.ctx, tx)` before any asset write (`UpdateQuantity`/`Release`/`Create` at lines 230-241); `opening.Insert(tx, ...)` at line 208 also takes `tx` directly |
| Kafka topic guard | `COMMAND_TOPIC_CASH_SHOP` handler self-guards | PASS | `kafka/consumer/cashshop/consumer.go:146` (`if c.Type != cashshop.CommandTypeOpenSurprise { return }`) |
| DOM-24 | Kafka producer stubbed in tests | PASS | `surprise/processor_test.go:31-34` `TestMain` calls `producertest.InstallNoop()`, no `t.Cleanup(producer.ResetInstance)` found |
| DOM-10 | Test DB has tenant callbacks | PASS | via shared `databasetest.NewInMemoryTenantDB` helper (`libs/atlas-database/databasetest/testdb.go:39` calls `database.RegisterTenantCallbacks`) — used at `surprise/processor_test.go:54` |
| EXT-02 | httptest-backed integration test for new client (character/commodity/rewardpool calls) | PASS | `surprise/processor_test.go:80-118` (`startCharacterServer`, `startCommodityServer`, `startRewardPoolServer` all use `httptest.NewServer`) |
| EXT-03 | rewardpool client distinguishes 404 vs 409 vs other | PASS | `rewardpool/processor.go:58-66` (`classifySelectError`: `ErrNotFound`→`ErrPoolMissing`, `ErrConflict`→`ErrPoolEmpty`, else passthrough) |
| SEC (ownership) | Compartment lookup keyed by `accountId`, not cross-checked against `characterId` — safe only if `accountId` is session-derived at the edge | PASS | `services/atlas-channel/atlas.com/channel/socket/handler/cash_item_gachapon.go:26` passes `s.AccountId()` (session-derived) into `cashshop.NewProcessor(...).OpenSurprise`; the wire packet (`cash/serverbound/item_gachapon_button.go:27-30`) carries only `cashId`, no accountId/characterId field — confirms the client cannot spoof the account |

### `opening` (support package — ledger; File Responsibilities Checklist)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-04 | Entity + `Migration` + `TableName` in `entity.go` | PASS | `opening/entity.go:10-33` |
| FILE-05 | Write ops in `administrator.go` | PASS | `opening/administrator.go:26-42` (`Insert`) |
| FILE-06 | No catch-all | PASS | `duplicate.go` is single-purpose (`isDuplicateKeyError`) |
| Idempotency correctness | PK-constraint based, not read-then-write | PASS | `opening/administrator.go:18-25` comment + `entity.go:24-25` (`TenantId`+`TransactionId` composite PK) |

### `rewardpool` (support/REST-client package; File Responsibilities + External HTTP Client Checklist)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor in `processor.go` | PASS | `rewardpool/processor.go:32-53` |
| FILE-02 | RestModel + JSON:API methods in `rest.go` | PASS (partial — see EXT-01) | `rewardpool/rest.go:7-22` (`GetName`/`GetID`/`SetID` present) |
| FILE-03 | Request funcs in `requests.go` | PASS | `rewardpool/requests.go:19-22` |
| FILE-05 | Domain `Model` in `model.go` | **FAIL** | `rewardpool/processor.go:22-30` — see Findings above |
| EXT-01 | Relationship-interface stubs on target struct | **FAIL** | `rewardpool/rest.go` — missing `SetToOneReferenceID`/`SetToManyReferenceIDs`; see Findings above |
| EXT-02 | httptest-backed test | PASS | `surprise/processor_test.go:100-118` (`startRewardPoolServer` + `rewardPoolOK`/`rewardPoolStatus` fixtures) exercises the rewardpool client end-to-end |
| EXT-03 | 404 vs other errors distinguished | PASS | `rewardpool/processor.go:58-66` |
| EXT-04 | No hardcoded service URL | PASS | `rewardpool/requests.go:9-11` (`requests.RootUrl("GACHAPONS")`), consistent with the pre-existing sibling client `services/atlas-channel/atlas.com/channel/incubator/requests.go:10` |

### `libs/atlas-packet` — `cash/serverbound/item_gachapon_button.go`, `cash/clientbound/item_gachapon_result.go` (DOM-25 focus)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-25 | SUCCESS/FAILED mode bytes config-resolved, not literals | PASS | `cash/clientbound/item_gachapon_result.go:143-155` (`CashItemGachaponSuccessBody`/`CashItemGachaponFailedBody` both use `atlas_packet.WithResolvedCode("operations", ..., ...)`) |
| DOM-25 | Table seeded in every version template that carries the feature | PASS | Verified all 6 templates that define the `CashItemGachaponResult` writer (`gms_83/84/87/92/95`, `jms_185`) carry `operations.SUCCESS`/`operations.FAILED` matching the code-comment opcodes exactly (e.g. v83: SUCCESS=229/0xE5, FAILED=228/0xE4); v79/v48/v61/v72/v12 correctly omit the writer (documented n/a: no result handler / no `CUICashItemGachapon` on those versions) |
| DOM-25 | Handler opcode (`CashItemGachaponHandle`) present in every template that has the button | PASS | `handlers[]` opCode present and version-matches the code comment in v79 (0x9F) through jms_185 (0xA7); absent (correctly) on v12/v48/v61/v72 |
| DOM-21 | No reinvented job-type classification | PASS | `surprise/processor.go:129-134` uses `job.GetType(c.JobId())` / `job.TypeExplorer` / `job.TypeCygnus` from `libs/atlas-constants/job` rather than a service-local `jobId/1000` literal |
| Wire correctness | Client-fed byte (`cashId`) is the only field carried on `CashItemGachaponButton`; no server-trusted account/character data on the wire | PASS | `cash/serverbound/item_gachapon_button.go:23-30` |

### `libs/atlas-rest` — `ErrConflict` sentinel

| Check | Status | Evidence |
|-------|--------|----------|
| Additive-only change to shared lib | PASS | `libs/atlas-rest/requests/get.go` unmodified; `libs/atlas-rest/requests/post.go:16,81-83` adds one new sentinel var and one new `if statusCode == http.StatusConflict` branch — no existing behavior altered, one new consumer (`rewardpool/processor.go:62`) |
| Naming/placement consistent with existing sentinels | PASS | `requests/get.go:16-25` — `ErrConflict` declared alongside `ErrBadRequest`/`ErrNotFound` in the same `var (...)` block, documented the same way |

### `services/atlas-reward-pools` — `cash-surprise` kind, `commodity_id`, flat-weight selection

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| Immutability | `item.Model`/`gachapon.Model` private fields + builder validation | PASS | `item/builder.go:81-101` (`Build()` enforces `ErrCommodityIdRequired` when `kind == gachapon.KindCashSurprise && commodityId == 0`); `gachapon/builder.go:72-92` (`isValidKind` closed-union check) |
| Closed-union kind handling | New kind wired into both `isValidKind` and `usesFlatWeights` | PASS | `gachapon/builder.go:15-19,97-99`; `reward/processor.go:40-42` |
| REST error mapping | 409 (empty pool) vs 404 (missing pool) vs 500 | PASS | `reward/resource.go:37-51` |
| DOM-15 | No direct entity creation in handler | PASS | `reward/resource.go:31-67` calls `NewProcessor(...).SelectReward(...)`, no `db.Create/Save/Delete` |

## Summary

### Blocking (must fix)
- FILE-05: `rewardpool/processor.go:22-30` — move the domain `Model` type and its accessors into a new `rewardpool/model.go`.
- EXT-01: `rewardpool/rest.go` — add no-op `SetToOneReferenceID`/`SetToManyReferenceIDs` on `RewardRestModel`, matching `services/atlas-cashshop/atlas.com/cashshop/character/rest.go:76-82`.

### Non-Blocking (should fix)
- `surprise/processor.go:207` — the inner `database.ExecuteTransaction` call is dead-code-in-practice given the single caller already supplies a `tx`; consider calling `Open`'s step-5 body directly against `p.db.WithContext(p.ctx)` (still safe under `isTransaction` short-circuit either way, but the redundant wrapper obscures that this function has exactly one caller and is never invoked outside a transaction).

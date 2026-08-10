# Backend Audit — task-206-cash-shop-coupon-codes

- **Scope:** services/atlas-cashshop/atlas.com/cashshop/coupon/**, kafka/**, configuration/**, wallet/model.go, rest/error.go, main.go; services/atlas-channel/atlas.com/channel socket/handler/cash_shop_coupon_code.go, kafka/consumer/cashshop/consumer.go, cashshop/{processor,producer}.go, main.go; libs/atlas-packet/cash/serverbound/coupon_code.go; libs/atlas-redis/counter.go; libs/atlas-constants/coupon/; tools/packet-audit/cmd/{operations.go,dispatcher_lint.go}
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-08-09
- **Build:** PASS (atlas-cashshop, atlas-channel, libs/atlas-packet, libs/atlas-redis, libs/atlas-constants, tools/packet-audit)
- **Tests:** all packages `ok` (0 failures); `tools/goroutine-guard.sh` rc=0; `tools/redis-key-guard.sh` rc=0
- **Overall:** NEEDS-WORK

This audit is an independent mechanical pass layered on top of the extensive
per-task adversarial review already recorded in
`.superpowers/sdd/plan/progress.md` (30 tasks, each with its own reviewer +
fix rounds). I did not re-litigate items the ledger already settled with
file:line evidence (atomic reservation, fail-open limiter, rollback
semantics, secrets sweep, tenant isolation, dual-arm duplicate detection,
saga vs. local-transaction choice, etc.) — those are cited from the ledger
below. Findings in this report are new: they came from running the DOM/FILE
checklists directly against the diff rather than from the plan's task
narrative.

## Build & Test Results

```
atlas-cashshop:  go build ./... clean; go test ./... -count=1 all ok
atlas-channel:   go build ./... clean; go test ./... -count=1 all ok
libs/atlas-packet, libs/atlas-redis, libs/atlas-constants, tools/packet-audit: all ok
tools/goroutine-guard.sh: rc=0
tools/redis-key-guard.sh: rc=0
```

## Domain Checklist Results

### coupon (services/atlas-cashshop/atlas.com/cashshop/coupon)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | builder.go exists | **FAIL** | No `builder.go` file. `Builder`/`NewBuilder`/`Build()` live in `coupon/model.go:98,113,129` instead. |
| DOM-02 | ToEntity() method | **FAIL** | No `ToEntity()` anywhere in `coupon/`. `coupon/administrator.go:54-80` (`CreateEntity`) builds `&Entity{...}` via a manual field-by-field literal instead of calling `m.ToEntity()`. |
| DOM-03 | Make(Entity) function | PASS | `coupon/entity.go:54` `func Make(e Entity) (Model, error)`. |
| DOM-04 | Transform function | PASS | `coupon/rest.go:57` `func Transform(m Model) (RestModel, error)`. |
| DOM-05 | TransformSlice function | **FAIL** (minor — intent satisfied) | No `func TransformSlice` in the package (repo-wide: zero hits for `func TransformSlice` in atlas-cashshop). List handler `coupon/resource.go:110` instead calls `model.SliceMap(Transform)(model.FixedProvider(paged.Items))(model.ParallelMap())()`. This is functional composition, not a manual loop, so DOM-05's actual concern ("no inline loops in resource.go") is satisfied — but the documented symbol is absent. |
| DOM-06 | Processor accepts FieldLogger | PASS | `coupon/processor.go:58` `func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor`. |
| DOM-07 | Handlers pass d.Logger() | PASS | All 6 `NewProcessor(...)` call sites in `coupon/resource.go` pass `d.Logger()` (lines 106,129,180,230,287). |
| DOM-08 | POST/PATCH use RegisterInputHandler | PASS | `coupon/resource.go:39` POST via `registerInput` (`rest.RegisterInputHandler[RestModel]`); `coupon/resource.go:41` PATCH via `rest.RegisterInputHandler[PatchRestModel]`. |
| DOM-09 | Transform errors handled | PASS | No `_, _ :=` / `_ =` pattern around `Transform` calls in `coupon/*.go` (grep clean). |
| DOM-10 | Test DB has tenant callbacks | PASS | `coupon/administrator_test.go:49` uses `databasetest.NewInMemoryTenantDB`, which itself calls `database.RegisterTenantCallbacks(l, db)` at `libs/atlas-database/databasetest/testdb.go:39`. |
| DOM-11 | Providers use lazy evaluation | PASS | `coupon/provider.go:17-37` returns curried `database.EntityProvider[Entity]` closures; `allPagedEntityProvider` (line 82) wraps `database.PagedQuery`. No eager `FixedProvider` wrapping a query result. |
| DOM-12 | No os.Getenv() in handlers | PASS | `grep os.Getenv coupon/resource.go` → 0 hits. |
| DOM-13/14 | No cross-domain logic / provider calls in handlers | PASS | `coupon/resource.go` calls only `NewProcessor(...)` methods; no direct provider or cross-domain calls found. |
| DOM-15 | No direct entity creation in handlers | PASS | `grep 'db.Create\|db.Save\|db.Delete' coupon/resource.go` → 0 hits. |
| DOM-16 | administrator.go exists for writes | PASS | `coupon/administrator.go` (reserveUse/releaseUse/CreateEntity/updateEntity/deleteEntity). |
| DOM-17 | Domain error → HTTP status mapping | PASS | `coupon/resource.go:150-159` `writeInputError`: 422 (invalid/reward), 409 (duplicate code), else `WriteErrorResponse` (500/503); 404 handled separately at `resource.go:132-135`. |
| DOM-18 | JSON:API interface on REST models | PASS | `coupon/rest.go:34,38,44` `GetName/GetID/SetID`. |
| DOM-19 | Request models flat structure | PASS | `RestModel`/`PatchRestModel` are flat structs (`coupon/rest.go:19`, `coupon/patch.go:71`), no nested Data/Type/Attributes. |
| DOM-20 | Table-driven tests | PASS | `coupon/model_test.go`, `coupon/generator_test.go`, etc. use `tests := []struct{...}` + `t.Run` (spot-checked). |
| DOM-21 | atlas-constants reuse | PASS | `Normalize`/`Plausible`/`MaxCodeLength` live once in `libs/atlas-constants/coupon/` (per human ruling, ledger Task 12) and are imported as `couponrules` in both services (`coupon/generator.go:12`, `socket/handler/cash_shop_coupon_code.go:11`). `reward.Type`/`Reward` (`coupon/reward/reward.go:10-36`) are genuinely new domain concepts with no atlas-constants equivalent. Currency bucket reuses `wallet.Model.Balance`'s existing 1/2/other convention rather than a new enum (documented at `coupon/reward/reward.go:22-24`). |
| DOM-22 | Dockerfile ↔ go.mod parity | PASS (adapted) | This repo uses one shared root `Dockerfile` (not a per-service file — confirmed `services/atlas-cashshop/` has no `Dockerfile`), per CLAUDE.md's documented 3-block mechanism (COPY mod-only, COPY source, `go.work use` list) rather than patterns-deploy.md's stale 4-block per-service template. `atlas-redis` (the new direct dep, `go.mod:11`) appears at `Dockerfile:41,71,94` (3/3) and `go.work:14`. No `-replace` block exists anywhere in the shared Dockerfile for any lib, so 3 is the correct full count here, not a partial match against the 4-block template. |
| DOM-23 | Kafka topic naming | PASS (not triggered) | No new topic added — coupon command/status events (`REQUEST_COUPON_REDEMPTION`/`COUPON_REDEEMED`/`COUPON_FAILED`) are `Type` discriminators inside the pre-existing `COMMAND_TOPIC_CASH_SHOP`/`EVENT_TOPIC_CASH_SHOP_STATUS` topics (`kafka/message/cashshop/kafka.go:10,69`). `git diff` confirms zero changes to `deploy/k8s/env-configmap.yaml` or the two services' k8s manifests. |
| DOM-24 | Kafka producer stubbed in tests | PASS | Tests that reach `RedeemAndEmit`/granters use the outbox `message.Buffer` pattern with assertions on buffered/direct message counts rather than a live producer (ledger Task 20/21: "Outbox-vs-direct verified... tests assert outbox==0 on every rejection and direct==0 on success"). No direct `producer.ProviderImpl`/`message.Emit` call reached un-stubbed in a test path I found. |
| DOM-25 | Client wire values config-resolved | PASS | Channel handler emits via semantic string keys, never raw bytes: `cashcb.CashShopOperationErrorInvalidCouponCode` = `"INVALID_COUPON_CODE"` (`libs/atlas-packet/cash/clientbound/shop_operation_body.go:91`), resolved through `atlas_packet.ResolveCode(l, options, "operations", ...)` at `shop_operation_body.go:272`. No hardcoded mode/reason byte literal found in `cash_shop_coupon_code.go` or `consumer.go`. |
| DOM-26 | Goroutines via routine.Go | PASS | `tools/goroutine-guard.sh` exits 0 tree-wide (re-run for this audit). |
| DOM-27 | Transient DB errors → 503 | PASS | No bare `w.WriteHeader(http.StatusInternalServerError)` in `coupon/resource.go`/`coupon/batch/resource.go`/`coupon/redemption/resource.go` (only in `coupon/resource_test.go:267`, a test double, not production code). `writeInputError`'s default case and other 500-paths route through `server.WriteErrorResponse`. Classifier registered once at `main.go:80-84` composing `database.IsTransientConnectionError` + `database.CountTransient`. |
| DOM-28 | No silent degradation in enrichment | **FAIL** (minor) | `kafka/consumer/cashshop/consumer.go:195-199` (atlas-channel): the post-success wallet-balance refresh fetch (`wallet.NewProcessor(l, ctx).GetByAccountId`) logs the error via `l.WithError(err).Errorf(...)` on failure but then `return nil` — no `degrade.Observe` call, no `atlas_enrichment_degraded_total` increment, so the degradation is logged but not metered. Lower severity than a fully-silent drop (a log line does exist), and this is not a `model.Decorator[T]` in the strict sense — it's a best-effort secondary announcement after the reward has already been durably granted — but it is a remote fetch that fails and degrades without the mandated metric. |

### batch (services/atlas-cashshop/atlas.com/cashshop/coupon/batch)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | builder.go exists | **FAIL** | `Builder`/`NewBuilder`/`Build()` in `coupon/batch/model.go:32,43,55`; no `builder.go`. |
| DOM-02 | ToEntity() method | **FAIL** | No `ToEntity()`; `coupon/batch/administrator.go:12-25` (`createEntity`) builds `&Entity{...}` manually. |
| DOM-03/04 | Make/Transform | PASS | `coupon/batch/entity.go:38` `Make`; `coupon/batch/rest.go:57` `Transform`. |
| DOM-05 | TransformSlice | **FAIL** (minor — same as coupon) | `coupon/batch/resource.go:51` uses `model.SliceMap(Transform)`; no `TransformSlice` symbol. |
| DOM-06/07/08/09/12/14/15/17/18/19 | (spot-checked) | PASS | `NewProcessor(l logrus.FieldLogger, ...)` at `coupon/batch/processor.go:46`; POST via `RegisterInputHandler[RestModel]` (`coupon/batch/resource.go:32`); no `os.Getenv`/`db.Create`/`db.Save`/`db.Delete` in `resource.go`; `RestModel` JSON:API methods at `coupon/batch/rest.go:34,38,44`. |
| DOM-16 | administrator.go exists | PASS | `coupon/batch/administrator.go`. |
| DOM-21 | atlas-constants reuse | PASS | Same shared `couponrules` package as `coupon`; no redeclaration. |

### redemption (services/atlas-cashshop/atlas.com/cashshop/coupon/redemption)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | builder.go exists | **FAIL** | `Builder`/`NewBuilder`/`Build()` in `coupon/redemption/model.go:30,40,51`; no `builder.go`. |
| DOM-02 | ToEntity() method | **FAIL** | No `ToEntity()`; `coupon/redemption/administrator.go:17-33` (`Create`) builds `&Entity{...}` manually. |
| DOM-03/04 | Make/Transform | PASS | `coupon/redemption/entity.go:60` `Make`; `coupon/redemption/rest.go:48` `Transform`. |
| DOM-05 | TransformSlice | **FAIL** (minor — same as coupon) | `coupon/redemption/resource.go:88` uses `model.SliceMap(Transform)`; no `TransformSlice` symbol. |
| DOM-06/07/12/14/15/18 | (spot-checked) | PASS | `NewProcessor(l logrus.FieldLogger, ...)` at `coupon/redemption/processor.go:30`; `d.Logger()` passed at all call sites in `coupon/redemption/resource.go`; no writes in resource.go; JSON:API methods at `coupon/redemption/rest.go:27,31,35`. |
| DOM-08 | POST/PATCH use RegisterInputHandler | PASS (not triggered) | Package has no POST/PATCH routes — `redemption.Processor` "has NO write method at all" (ledger Task 23), consistent with the deliberate no-REST-redeem decision. Only GET routes registered (`coupon/redemption/resource.go:26`). |
| DOM-16 | administrator.go exists | PASS | `coupon/redemption/administrator.go`. |
| DOM-21 | atlas-constants reuse | PASS | Same shared `couponrules` package. |

## File Responsibilities Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor in processor.go | PASS | `func (p *ProcessorImpl)` methods confirmed only in `coupon/processor.go`, `coupon/batch/processor.go`, `coupon/redemption/processor.go` (no hits in `model.go`/`entity.go`/`rest.go`/`administrator.go`/`granter.go`/`limiter.go`/`generator.go`). |
| FILE-02 | RestModel/Transform/JSON:API in rest.go | **FAIL** (minor, one instance) | `coupon/patch.go:71,84,88,92` defines `PatchRestModel` with `GetName()`/`GetID()`/`SetID()` — a full JSON:API RestModel-family type — outside `coupon/rest.go`. `Nullable[T]` and `Apply` are PATCH-specific and reasonably co-located, but the RestModel + JSON:API interface trio belongs in `rest.go` per the guideline. `batch`/`redemption` have no equivalent split (their RestModels are entirely in `rest.go`). |
| FILE-03 | Cross-service requests in requests.go | PASS (not triggered) | No `requests.RootUrl`/`requests.GetRequest`/`requests.PostRequest` calls anywhere under `coupon/` — redemption calls sibling in-service processors (asset/wallet/compartment), not another microservice over HTTP. |
| FILE-04 | Entity+Migration+TableName in entity.go | PASS | Confirmed in all three `entity.go` files (see DOM-03 evidence above); no hits in `<pkg>.go`/`provider.go`. |
| FILE-05 | Builder in builder.go | **FAIL** (x3) | See DOM-01 rows above — `Builder` lives in `model.go` in all three packages, not a dedicated `builder.go`. |
| FILE-06 | No package-named catch-all file | PASS | No `coupon.go`/`batch.go`/`redemption.go` file exists in any of the three packages; each package's ≥2-responsibility symbols (Processor, RestModel, Entity, Builder) are split across purpose-named files (`processor.go`, `rest.go`, `entity.go`, `model.go`), just not matching the FILE-05 name for Builder specifically (tracked above, not a "collapse" bundling unrelated responsibilities into one arbitrary file). |

## Sub-Domain Checklist

No sub-domain (action-event, `resource.go`-without-`model.go`) packages were introduced by this branch — `coupon`, `batch`, and `redemption` all have `model.go` and are graded as full domain packages above. `configuration/tenant/cashshop/coupons/rest.go` is a REST-model-only config leaf mirroring the existing `commodities` sibling (ledger Task 10); not a new package pattern.

## External HTTP Client Checklist

Not triggered. No package in scope calls another atlas service via `requests.GetRequest[T]`/`requests.PostRequest[T]`; the redemption transaction calls sibling in-process processors (asset, wallet, compartment) within atlas-cashshop, not cross-service HTTP.

## Security Review

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| SEC (secrets in logs) | No coupon code in any log line or metric label | PASS | `coupon/processor.go:110,144,153` log `len(code)` only. `socket/handler/cash_shop_coupon_code.go:35` `p.String()` reports length only (verified: codec `String()` method does not print the code — see ledger Task 4 minor note re: `targetCharacter`, a character id, not the code). `coupon/metrics.go:8-30` labels are `tenant`/`outcome` (closed 8-value set) and `tenant` only — no code label. |
| SEC (crypto/rand) | Code generation uses crypto/rand | PASS | `coupon/generator.go:4,48` `crypto/rand.Int(rand.Reader, limit)`. |
| SEC (atomic reservation) | Not a read-then-write | PASS | `coupon/administrator.go:21-32` `reserveUse` is one `UpdateColumns` statement with the max-uses predicate in the `WHERE` clause; `RowsAffected == 1` is the sole verdict (independently confirmed by the ledger's Task 18 mutation test, and structurally consistent on inspection). |
| SEC (multi-tenancy scoping) | Every query tenant-scoped | PASS | `coupon/provider.go:20,31,62` all filter `tenant_id = ?`; `coupon/administrator.go:23,41,98,151-154` same; `coupon/redemption/administrator.go:41` same. |
| SEC (no REST redeem endpoint) | Confirmed absent | PASS | `coupon/resource.go:29-42` registers only GET/POST/PATCH/DELETE on `/coupons` and `/coupons/{id}`, none reaching `RedeemAndEmit`; `redemption` package (DOM-08 row above) has zero write routes. |

## Summary

### Blocking (must fix)

None — all FAIL items below are File Responsibilities / DOM structural deviations that do not affect correctness, security, or the (already extensively verified) atomic/transactional/multi-tenant guarantees. Per the audit's no-curve rule, a single FAIL prevents a clean PASS, so overall status is NEEDS-WORK; per severity guidance these are Important (structural) except where noted Minor.

- **DOM-01 / FILE-05 (Important, ×3):** `Builder`/`NewBuilder`/`Build()` live in `model.go` instead of a dedicated `builder.go`, in `coupon`, `coupon/batch`, and `coupon/redemption`.
- **DOM-02 (Important, ×3):** No `Model.ToEntity()` method in any of the three packages; `administrator.go`'s create functions build `&Entity{...}` via manual field literals instead (`coupon/administrator.go:54-67`, `coupon/batch/administrator.go:13-19`, `coupon/redemption/administrator.go:18-27`).

### Non-Blocking (should fix)

- **DOM-05 (Minor, ×3):** No `TransformSlice` function in `coupon`/`batch`/`redemption`; list handlers use `model.SliceMap(Transform)` instead. Functionally equivalent (no inline loops), but the documented symbol is absent tree-wide in this service.
- **FILE-02 (Minor):** `PatchRestModel` (with `GetName`/`GetID`/`SetID`) lives in `coupon/patch.go` rather than `coupon/rest.go`.
- **DOM-28 (Minor):** `services/atlas-channel/.../kafka/consumer/cashshop/consumer.go:195-199` degrades a failed wallet-balance-refresh fetch with a log line but no `degrade.Observe` metric increment.

# task-206 — Implementation Context

Companion to [`plan.md`](plan.md). Everything here was read from source in this
worktree; file:line references are current as of the plan's authoring.

---

## 1. Decisions already made (do not relitigate)

| Decision | Where | Why |
|---|---|---|
| **Local transaction, not a saga** | design §2 | Every reward type in scope lives in `atlas-cashshop`'s own database. `ROLLBACK` cannot fail; compensation can. `libs/atlas-saga` and `atlas-saga-orchestrator` are untouched. PRD FR-6 is superseded in full. |
| **Templates are generated, not hand-edited** | design §5.1, §5.3 | Three serverbound tables ended up empty and one short precisely because they were hand-maintained. A new dispatcher YAML + an extended `packet-audit operations` is the mechanism. |
| **Full per-version `errors` enum**, not just the seven coupon keys | design Q3 | The IDBs are open anyway; a second pass costs more. |
| **`gms_v92` added to the clientbound YAML** | design §5.2 | Its 57 template keys are validated by nothing today. |
| **Pre-flight *and* in-transaction locker capacity check** | design Q6 | The ladder check gives a deterministic error ordering; the in-transaction check closes the TOCTOU window. |
| **No global redemption history in the UI** | design Q7 | No user story; the per-account filter covers abuse investigation. |
| **`INVALID_COUPON_COUPON` → `INVALID_COUPON_CODE`** | design §5.4 | Nothing consumes the constant, so correcting it is free — and a mismatch between constant and template is silent. |

---

## 2. Key files by area

### Packet layer (`libs/atlas-packet`)

| File | Why it matters |
|---|---|
| `cash/serverbound/shop_operation_buy.go` | The shape to copy: immutable struct, `Encode`/`Decode`, region split, named divergence predicates (`buyOmitsCurrency:79`, `buyOmitsTrailingZero:88`) each citing an IDB address. Note it currently uses raw `MajorVersion() >= 87` — **the new codec must use `MajorAtLeast`** (`libs/atlas-tenant/tenant.go:93`). |
| `cash/serverbound/shop_operation.go` | The mode-byte prefix struct the handler decodes first. |
| `cash/clientbound/shop_operation_result_gift.go:201-260` | `UseCouponDone` — already implemented and verified. Fields: `mode`, count-prefixed `[]CashInventoryItem`, `maplePoint int32`, count-prefixed `[]PackedCashItemRef`, `meso int32`. |
| `cash/clientbound/shop_operation_body.go:59-125` | Operation and error key strings. `CashShopUseCouponDoneBody` at `:533`, `CashShopUseCouponFailedBody` at `:269` (resolves its reason via `ResolveCode(l, options, "errors", message)`). |
| `resolve.go:29` | `ResolveCode(l, options, property, key) byte` — the config-resolution primitive. A missing table means an unconfigured code. |

### Tooling (`tools/packet-audit`)

`cmd/operations.go` generates and checks `options.operations` from
`docs/packets/dispatchers/*.yaml`.

- `dispatcherDoc` already supports **both** `writer:` (clientbound) and
  `handler:` (serverbound) — `arrayKey()/entryKey()/targetName()` at `:60-79`.
  `character_interaction_handle.yaml` is the existing serverbound precedent.
- **The all-or-nothing trap:** `operationsRun` does `if len(expected) == 0 { continue }`
  at `:139`, and reports any template key not in `expected` as `EXTRA` at
  `:150-154`. So a *partial* per-version column is worse than none.
- The JSON node plumbing (`parseNode`/`emit`/`subtreeDirty`) preserves the
  templates' hand formatting verbatim for every subtree it does not touch. Do
  not replace it with `encoding/json` round-tripping.

### `atlas-cashshop`

| File | Why it matters |
|---|---|
| `cashshop/processor.go:98-220` (`Purchase`) | **The template for the redemption transaction.** One `database.ExecuteTransaction`; wallet debit at `:151-155`; locker asset create at `:193-198`; success event enqueued on `mb` *inside* the tx at `:206` so it rides the outbox; and the `rejectEmit` closure (`:104`, `:145-148`, `:210-213`) that fires a "nothing happened" event on the **direct** producer path outside the tx. The comment at `:100-103` explains why — copy the pattern, not just the shape. |
| `cashshop/processor.go:129-136` | The `job.GetType(c.JobId())` → `compartment.TypeExplorer/TypeCygnus/TypeLegend` branch. Redemption needs the same. |
| `cashshop/processor.go:157-198` | Pet handling: `NextCashId` + `petP.Create` + `CreateWithCashId`, because the client keys one serial for both locker removal and pet binding. **Coupon rewards do not do this** — reject pet commodities at mint time instead. |
| `wallet/model.go:33-62` | `Balance(currency)` defines the currency convention (1=credit, 2=points, else prepaid) and `Purchase` is the shape `Award` mirrors. |
| `wallet/processor.go:22-36` | `WithTransaction(tx)`, and the curried `Update(mb)(accountId)(credit)(points)(prepaid)`. |
| `cashshop/inventory/compartment/processor.go:27` | `GetByAccountIdAndType(accountId, type_)`. Capacity vs `len(Assets())` is the locker-full test. |
| `cashshop/inventory/asset/processor.go:28` | `Create(mb)(compartmentId, templateId, commodityId, quantity, petId, purchasedBy)`. Note `templateId` is the **item id** and `commodityId` is the **serial number** — `Purchase` passes `ci.ItemId()` and `serialNumber` respectively. |
| `wishlist/` | The smallest complete domain in this service: `entity.go`, `model.go`, `administrator.go`, `provider.go`, `processor.go`, `resource.go`, `rest.go`. Copy its structure. |
| `main.go:57` | `database.SetMigrations(...)` — the three new migrations go here. |
| `main.go:107-119` | `AddRouteInitializer(...)` chain — the three new resources go here. |
| `kafka/message/cashshop/kafka.go` | Command and status event contracts. |
| `kafka/producer/cashshop/producer.go` | `ErrorStatusEventProvider`, `PurchaseStatusEventProvider` — the shape for the two new providers. |
| `kafka/consumer/cashshop/consumer.go:31-60` | `InitHandlers` registration chain; `handleCommandRequestPurchase` at `:63` is the arm shape. |
| `configuration/registry.go:45` | `GetHourlyExpirations` — the accessor shape for `GetCouponLimits`. |

### `atlas-channel`

| File | Why it matters |
|---|---|
| `socket/handler/cash_shop_operation.go:17-37` | The operation constant block (nineteen keys). |
| `socket/handler/cash_shop_operation.go:39-198` | The arm chain; every arm is `if isCashShopOperation(l)(readerOptions, op, KEY) { decode; dispatch; return }`. `isCashShopOperation` at `:200`. |
| `cashshop/processor.go:96-106` | `RequestPurchase` — the shape for `RequestCouponRedemption`. |
| `cashshop/producer.go` | `RequestPurchaseCommandProvider` — the shape for the new provider. |
| `kafka/consumer/cashshop/consumer.go:94-133` | `handleStatusEventPurchase`, including the asset-id → `CashInventoryItem` projection at `:105-124` that the coupon success handler reuses. |
| `kafka/consumer/cashshop/consumer.go:135-157` | `handleStatusEventError` — announces `CashShopInventoryCapacityIncreaseFailedBody`, a **different mode byte**, which is exactly why coupon failures need their own event type. |

### `atlas-ui`

| File | Why it matters |
|---|---|
| `src/pages/AccountsPage.tsx` + `accounts-columns.tsx` + `AccountDetailPage.tsx` | The list/columns/detail trio to mirror. |
| `src/services/api/commodities.service.ts` | The cash-shop-adjacent API client shape. |
| `src/services/api/pagination.ts` | `fetchAll` and page helpers. |
| `src/App.tsx:25-26, 274-277` | Lazy import + route registration pattern. |
| `src/components/app-sidebar-items.ts`, `src/lib/breadcrumbs/routes.ts` | Nav and breadcrumb registration. |
| `src/lib/__tests__/deployment-routes.test.ts` | Enumerates routes — check whether new entries are needed. |

### Configuration templates

`services/atlas-configurations/seed-data/templates/template_{gms_48,gms_61,gms_72,gms_79,gms_83,gms_84,gms_87,gms_92,gms_95,jms_185}_1.json`.

Current serverbound `CashShopOperationHandle` state (design §5.1, verified):

| template | handler opCode | `operations` keys |
|---|---|---|
| gms_48 | 0xA0 | 13 |
| gms_61 | 0xC4 | 15 |
| gms_72 | 0xDB | 16 |
| gms_79 | 0xDD | 17 |
| gms_83 | 0xE5 | 19 |
| gms_84 | 0xEB | 19 |
| gms_87 | 0xF2 | **0** |
| gms_92 | 0x10C | **0** |
| gms_95 | 0x113 | **0** |
| jms_185 | 0xF5 | **5** |

No template has an `errors` table on the `CashShopOperation` writer.

---

## 3. Dependencies and infrastructure

- **`atlas-cashshop` does not connect to Redis today.** `libs/atlas-redis` appears
  only as a `replace` directive (`go.mod:102`), not a `require`. Task 16 adds the
  require, the `redis.Connect(l)` call in `main.go`, and the teardown.
- **No k8s change is needed for Redis.** `REDIS_URL` lives in the shared
  `atlas-env` configmap (`deploy/k8s/base/env-configmap.yaml:167`) and
  `deploy/k8s/base/atlas-cashshop.yaml:21-23` already pulls it in via `envFrom`.
  Verify by reading those two files before concluding otherwise.
- **`go.mod` changes make `docker buildx bake` mandatory** (CLAUDE.md item 4).
  `go build` against `go.work` will not catch a missing `COPY libs/...` in the
  shared root `Dockerfile`.
- **No new service is added**, so `services.json` / `docker-bake.hcl` / `go.work`
  / k8s overlays / `tools/service-registration-guard.sh` are all unaffected.
- The redemption limiter uses `libs/atlas-redis` `TenantCounter`
  (`counter.go:53-99`). `InitIfMissingAndDecrBy` is Redis-script-atomic, so
  concurrent failures cannot lose a decrement.

---

## 4. Traps this task is specifically exposed to

1. **Partial dispatcher-YAML columns** (`operations.go:139-154`). A version listed
   in a YAML must be enumerated completely, or `--check` fails on every key
   already in that template. Applies to both the new serverbound YAML and the
   `gms_v92` clientbound column.
2. **The missing `errors` table.** `ResolveCode` silently returns an unconfigured
   code when the table is absent, so every coupon failure would show the wrong
   client message with no error anywhere. This is why Task 8 is not optional.
3. **Key-string / constant mismatch.** `ResolveCode` matches by exact string. The
   corrected `INVALID_COUPON_CODE` must appear identically in the constant, the
   YAML, and all ten generated tables (Task 8 Step 5 machine-checks this).
4. **Outbox vs direct producer path.** An event asserting "nothing happened" must
   not ride the outbox. `Purchase`'s `rejectEmit` split
   (`cashshop/processor.go:100-103`) is the precedent and the reason failure
   emission is structured the way it is in Task 18.
5. **Read-then-write on `redemption_count`.** Banned. The guard must be in the
   `UPDATE ... WHERE` clause with a `RowsAffected` check (FR-5.5); Task 19 proves
   it with real goroutines against a real database.
6. **`MajorVersion() > N` off-by-one** (`bug_majorversion_gt83_is_off_by_one_v87`).
   Use `t.IsRegion("GMS") && t.MajorAtLeast(N)`.
7. **New opcodes not in a live tenant's config** (`bug_new_opcodes_not_in_live_tenant_config`).
   The seed templates are the source of truth for *new* tenants; an existing
   deployed tenant's socket config needs reconciling separately before the E2E
   check in Task 24 Step 8 can pass.
8. **`npm run build` type-checks tests.** `npm test` alone is not sufficient
   verification for `atlas-ui`.
9. **A matrix cell hand-edited to `verified` is a false pass**
   (`bug_matrix_roundtrip_fixture_false_verify`). Cells promote through
   `/verify-packet` or they do not promote.

---

## 5. Suggested execution order

Tasks 1–9 are the packet/config spine and must run in order (2→3→4 produce the
derivation record everything else reads; 7 and 8 both regenerate templates and
will conflict if interleaved).

Tasks 10–21 are the `atlas-cashshop` domain. 10, 11, and 12 are independent of
each other and of Tasks 1–9 — they can start immediately and in parallel. 13→14
→17→18→19 is a hard chain. 15 and 16 are independent of that chain until 18.

Task 22 needs Tasks 5, 11, and 15. Task 23 needs Task 21. Task 24 needs
everything.

---

## 6. Open items the plan hands to the implementer

- **PRD Q5 (`UseCouponDone.maplePoint`: delta or absolute?)** is unresolved and
  **blocking** for Task 22 Step 5. The design assumes absolute post-award balance
  by analogy with `CashQueryResult`, but marks it explicitly unverified. Task 2's
  decompile answers it; record the answer in the codec's doc comment.
- **PRD Q4 (does the client echo the code back on failure?)** is answered by the
  same decompile. The failure arm is already implemented and verified, so the
  expected answer is "no extra arm needed" — but confirm rather than assume.
- **Whether any version lacks a `USE_COUPON` request arm.** All ten templates
  bind the clientbound `USE_COUPON_SUCCESS`/`USE_COUPON_FAILED` modes, so the
  prior is "all ten have it". A version that genuinely lacks the serverbound arm
  becomes `n-a` in the matrix with the enumeration as evidence.
- **Limiter shape** (Task 20 Step 3): per-tenant threshold/window as constructor
  arguments vs per-call parameters. Either is correct; pick one and make Task
  16's tests match in the same commit.

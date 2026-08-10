# task-206 — Implementation Context

Companion to [`plan.md`](plan.md). Read this first if you are picking the task up cold.

Artifacts, in the order they were produced: [`prd.md`](prd.md) → [`design.md`](design.md) → [`derivation.md`](derivation.md) → `plan.md`. **Where they disagree, the later one wins**, and `derivation.md` beats both of the first two on any wire value.

---

## 1. The one thing that reframes the task

`USE_COUPON` is **not** a mode arm of the serverbound `CASHSHOP_OPERATION` dispatcher. The PRD and the design both assume it is; they are wrong.

Two facts, read out of the gms_v83 IDB and re-confirmed on v84/v87/v92/v95 (`derivation.md`, "Structural finding that reframes the whole task"):

1. There is no `CCashShop::OnCashItemRequest` function. The client has no single cash-shop request builder — each UI action constructs its own `COutPacket(&pkt, <CASHSHOP_OPERATION opcode>)` and writes its own leading `Encode1(mode)`. The "mode switch" is a server-side construct.
2. The coupon submission is a **separate opcode** — the registry's `COUPON_CODE`, built by `CCashShop::OnStatusCoupon` — with **no mode byte at all**. The body starts immediately with strings.

Consequences the plan is built around:

- No `USE_COUPON` key is ever added to `CashShopOperationHandle.options.operations`.
- The codec is a standalone `cash/serverbound/CouponCode`, not a `ShopOperation*` sub-arm.
- The channel gets a standalone handler registered by name in `main.go`, not a branch in `cash_shop_operation.go`.
- The templates need a whole new **handler entry** (opCode + validator + handler + services), which is unscoped work the PRD never anticipated — and which is why Task 6 has to fix `packet-audit`'s `addEntry` first.

The clientbound half is unaffected: `USE_COUPON_SUCCESS` / `USE_COUPON_FAILED` are writer modes and already exist in all ten templates.

## 2. Prior-phase questions that are now answered

| Question | Answer | Evidence |
|---|---|---|
| PRD Q1 — saga or local transaction? | **Local transaction.** No saga, no `libs/atlas-saga` change, no orchestrator change. | design §2 (user decision) |
| PRD Q2 — which versions? | `gms_v83`, `v84`, `v87`, `v92`, `v95`, `jms_v185`. The four legacy versions have no `COUPON_CODE` op and are already `n-a` in the matrix; Task 3 evidences that. | registry + `status.json` |
| PRD Q4 — does the client echo the code back on failure? | **No.** `OnCashItemResUseCouponFailed` @ `0x47a7db` reads exactly one `Decode1` and nothing else. No extra arm. | `derivation.md`, "Blocking answer 2" |
| PRD Q5 — is `maplePoint` a delta or a balance? | **DELTA.** Skipped entirely when zero (`if (v68)`) and rendered inside "You have received … using the coupon". The design's "absolute post-award balance" comment is wrong; Task 16 corrects it. | `derivation.md`, "Blocking answer 1" |
| PRD Q6 — capacity-check timing? | **Both** — a pre-flight ladder check for deterministic error ordering, and an in-transaction re-check in the cash-item granter to close the TOCTOU window. | design §5/Q6 |
| PRD Q7 — global redemption history in the UI? | **No.** Per-code and per-account only. | design §1 |
| Design Q3 — how much of the error enum? | The **full** per-version enum, tool-generated. | design §5.3 |

## 3. What is already on the branch

| Commit | Content | Status |
|---|---|---|
| `2d4aab2b7` | PRD | done |
| `e954909fb` | design | done, with the FR-2/FR-4 corrections above |
| `46129a110` | the **first** plan + context | **superseded** — that plan was written on the false "USE_COUPON is a mode arm" premise |
| `15f811528` | `packet-audit operations` generates and checks `options.errors` | done, reviewed clean; plan Task 8 supplies its data |
| `a36149bf0` | `derivation.md` — five GMS versions | done, but see §4 for what is missing |
| `4e43b5631` | WZ/StringPool cross-check | done; promoted 2 of 53 v83 error rows to `verified` |

Nothing in `libs/atlas-packet`, `services/atlas-cashshop`, `services/atlas-channel` or `services/atlas-ui` has been touched yet.

## 4. What `derivation.md` does and does not contain

**Complete:** gms_v83 / v84 / v87 / v92 / v95 — coupon request body, `COUPON_CODE` opcode, full `NoticeFailReason` error enum. Serverbound `CashShopOperationHandle` arm tables for v83 (19 keys), v84 (19 + one unnamed), v87 (19 + one unnamed).

**Missing, and each has a task:**

| Gap | Task |
|---|---|
| `jms_v185` — everything (body, opcode confirmation, error enum) | 2 |
| `gms_v48`/`v61`/`v72`/`v79` — coupon-send applicability verdict | 3 |
| `gms_v48`/`v61`/`v72`/`v79` (+ jms) — error enums | 9 |
| `gms_v92` clientbound arm table (57 keys, currently validated by nothing) | 9 |
| `gms_v92` serverbound arm attribution (19 of 25 modes read, none attributed to a key — the IDB has no names on the sender functions) | 29 |
| `gms_v95` serverbound — 3 unresolved sites (`0x483b82`, `0x4901aa`, `0x490ad8`) | 29 |

**Evidence-confidence caveat.** Roughly 50 of the 53 error rows per version are marked `aligned`: the **byte** is read from the jump table and is real, but the **key name** is ordinal alignment of the case list against the declared order of the `CashShopOperationError*` constants, pinned by three anchors per version. Two v83 rows were promoted to `verified (cross-decompile)` by the WZ pass. Encoding an `aligned` byte is safe — the client renders *a* notice — but the specific English wording is unconfirmed. Do not restate an `aligned` row as verified without new evidence. Task 30's live end-to-end test is the cheapest chance to convert several of them.

## 5. Key wire values

```
COUPON_CODE opcode          body
gms_v83   230 / 0xE6        str targetCharacter · str code · str extra (guarded on field1 non-empty)
gms_v84   236 / 0xEC        same          ← registry says 230; that is a BUG, Task 1 fixes it
gms_v87   243 / 0xF3        same
gms_v92   269 / 0x10D       str targetCharacter · str code            (no third string)
gms_v95   276 / 0x114       same
jms_v185  246 / 0xF6 ?      unverified — Task 2
```

Error-enum scales, all relative to the v83 baseline: v84 = **+9**, v87 = **+15** (plus one new reason at 249), v92 and v95 = **−162** (1-based; five new reasons at 63–67, two keys out of domain). Each offset is proved by exact set-equality of the jump table's default-case set plus a call-site anchor, not assumed.

**Reserved reason bytes — never send as a generic failure**, they change client state rather than showing a notice: v83 `0xA2`/`0xA4` (kick out of the cash shop) and `0xB1` (wrong-coupon-number notice then disconnect); v84 `171`/`173`/`186`; v92/v95 `0`/`2`/`15`.

## 6. Files that carry the load

| Concern | File |
|---|---|
| Purchase path to copy | `services/atlas-cashshop/atlas.com/cashshop/cashshop/processor.go:90-220` — `PurchaseAndEmit` / `Purchase`. Note the `rejectEmit` closure at line 100-103: an event asserting "nothing happened" goes on the **direct** producer path, not the outbox. |
| Wallet | `.../wallet/model.go` (`Balance`/`Purchase`; `Award` is Task 11), `.../wallet/{entity,provider,administrator,resource,rest}.go` — the canonical domain shape to copy |
| Status-event consumer | `services/atlas-channel/atlas.com/channel/kafka/consumer/cashshop/consumer.go` — `handleStatusEventPurchase` at :92 is the model, incl. the asset-id → `CashInventoryItem` projection at :105-124 |
| Standalone handler | `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_check_wallet.go` + its registration at `main.go:926` |
| Codec shape | `libs/atlas-packet/cash/serverbound/shop_operation_gift.go` (version gating), `check_wallet.go` (standalone op), `shop_operation_buy_name_change_test.go` (fixture test) |
| Clientbound coupon arms | `libs/atlas-packet/cash/clientbound/shop_operation_result_gift.go:201` (`UseCouponDone`), `shop_operation_result_failed.go:161` (`UseCouponFailed`), `shop_operation_body.go:80-135` (54 error constants), `:269` (`CashShopUseCouponFailedBody`) |
| Table generation | `tools/packet-audit/cmd/operations.go` — `dispatcherDoc` :43, `tables()` :66, the all-or-nothing short-circuit :142, `addEntry` :504 |
| Handler dispatch | `libs/atlas-opcodes/producer.go:58-87` (`BuildHandlerMap`) and `config.go:36-46` (`appliesToService`) |
| Code resolution | `libs/atlas-packet/resolve.go:29-56` (`ResolveCode`; a miss logs and returns 99) |
| Redis | `libs/atlas-redis/counter.go` (`TenantCounter`), `connection.go` (`Connect` reads `REDIS_URL`) |
| Tenant config | `services/atlas-cashshop/atlas.com/cashshop/configuration/registry.go`, `.../configuration/tenant/cashshop/rest.go`; the seed block lives in each `template_*.json` under top-level `cashShop` |
| UI patterns | `services/atlas-ui/src/pages/AccountsPage.tsx` + `accounts-columns.tsx` + `AccountDetailPage.tsx`; `services/api/mts-config.service.ts` for the JSON:API envelope |

## 7. Traps this plan is specifically defending against

1. **A generated handler entry with no `validator` never routes.** `BuildHandlerMap` looks the name up and `continue`s on a miss with only a `Warnf`. `addEntry` does not emit one today — Task 6.
2. **A generated entry appended to the end breaks the opcode-order guard.** Same task.
3. **The `operations`/`errors` all-or-nothing rule.** `expectedTable` reports every template key absent from the YAML as `EXTRA`, but only once a version has ≥1 declared key. A *partial* per-version declaration is therefore worse than none. Declare a version completely or not at all.
4. **`gms_v92` is invisible to `packet-audit operations --check` today** — its 57 clientbound keys are validated by nothing, and the check reports OK for exactly that reason.
5. **A read-then-write on `redemption_count` is a race.** Task 21 Step 3 requires proving the race test actually fails when the conditional `UPDATE` is removed.
6. **Every handler on `COMMAND_TOPIC_CASH_SHOP` receives every message.** The `c.Type != …` guard is load-bearing; the generic unmarshal succeeds with a zero body for other command types.
7. **`UNKNOWN_ERROR` is the jump table's default case**, not a byte. It is deliberately absent from the `errors` table; `ResolveCode` misses, returns 99, and 99 is itself unlisted, so the client shows the default notice. Do not "fix" this by adding a key.
8. **Coupon codes are secrets.** Never log one, never label a metric with one. `CouponCode.String()` logs the length.
9. **`tools/lint.sh --check` false-fails without nvm**, and contends on a lock across worktrees. Neither is a lint failure.
10. **`docker buildx bake` is mandatory** for `atlas-cashshop` — Task 17 adds `libs/atlas-redis` to its `go.mod`, and only the bake catches a missing `COPY libs/...` in the shared root `Dockerfile`.

## 8. Verification quick reference

```bash
# per changed module
go build ./... && go vet ./... && go test -race ./...

# repo root
tools/redis-key-guard.sh; tools/goroutine-guard.sh; tools/lint.sh --check
tools/template-opcode-order-guard.sh; tools/template-duplicate-binding-guard.sh
tools/template-movement-types-guard.sh; tools/service-registration-guard.sh

go run ./tools/packet-audit matrix
go run ./tools/packet-audit operations --check
go run ./tools/packet-audit fname-doc --check

docker buildx bake atlas-cashshop atlas-channel atlas-configurations

cd services/atlas-ui && npm run build && npm test
```

`atlas-saga-orchestrator` is **not** in the bake list — design §2 removed it from the blast radius.

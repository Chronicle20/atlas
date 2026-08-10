# Cash Shop Coupon-Code Redemption Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A player types a promo code into the Cash Shop Coupon tab and receives NX / Maple Points / cash-locker items, with an admin surface to mint and audit codes.

**Architecture:** The client sends the standalone serverbound `COUPON_CODE` opcode (NOT a `CASHSHOP_OPERATION` mode arm). `atlas-channel` decodes it, normalizes the code, and publishes `REQUEST_COUPON_REDEMPTION` on `COMMAND_TOPIC_CASH_SHOP`. `atlas-cashshop` owns a new `coupon` domain and performs redemption in ONE local `database.ExecuteTransaction` (no saga — design §2), emitting `COUPON_REDEEMED` / `COUPON_FAILED` on `EVENT_TOPIC_CASH_SHOP_STATUS`. The channel's status consumer announces `USE_COUPON_SUCCESS` / `USE_COUPON_FAILED`.

**Tech Stack:** Go 1.24 microservices, GORM + Postgres, Kafka (`libs/atlas-kafka` + outbox), JSON:API via api2go, `libs/atlas-packet` codecs, `libs/atlas-redis`, `tools/packet-audit`, React 19 + TanStack Query + shadcn/ui for `atlas-ui`.

---

## Global Constraints

Every task's requirements implicitly include this section.

- **`USE_COUPON` is NOT a `CashShopOperationHandle` mode arm.** The request is the standalone serverbound op `COUPON_CODE`, fname `CCashShop::OnStatusCoupon`, with **no mode byte**. PRD FR-2.1/FR-4.1 and design §5.1/§6 are wrong on this point; `derivation.md` "Structural finding that reframes the whole task" is authoritative. Never add a `USE_COUPON` key to `options.operations`.
- **In-scope versions for the coupon request:** `gms_v83`, `gms_v84`, `gms_v87`, `gms_v92`, `gms_v95`, `jms_v185`. `gms_v48`/`v61`/`v72`/`v79` carry **no** `COUPON_CODE` op in `docs/packets/registry/` and are already `n-a` in `docs/packets/audits/status.json`; Task 3 turns that registry-absence into positive evidence.
- **Derived wire values come only from `derivation.md`.** A value absent from that file may not appear in a codec, a dispatcher YAML, or a tenant template. If a task needs a value that is not there, derive it and record it there first.
- `UseCouponDone.maplePoint` is a **DELTA** (the amount this coupon awarded), not an absolute balance — `derivation.md` "Blocking answer 1". `design.md:160`'s `// absolute post-award balance` comment is wrong and is corrected in Task 15.
- The client does **not** echo the code back on failure — `UseCouponFailed` reads exactly one byte (`derivation.md` "Blocking answer 2"). No extra arm.
- Version gating uses `t.IsRegion("GMS") && t.MajorAtLeast(N)`. Raw `t.MajorVersion() > N` is banned (`bug_majorversion_gt83_is_off_by_one_v87`).
- Client wire values are **config-resolved**, never hard-coded in a handler or processor (DOM-25).
- No new service: `services.json` / `docker-bake.hcl` / `go.work` / k8s overlays are untouched, and `tools/service-registration-guard.sh` is unaffected. `atlas-cashshop` already receives `REDIS_URL` via the shared `atlas-env` configMap (`deploy/k8s/base/atlas-cashshop.yaml:21-23`, `deploy/k8s/base/env-configmap.yaml:167`) — no manifest change for the rate limiter.
- `libs/atlas-saga` and `atlas-saga-orchestrator` are **not touched** (design §2, §13.1).
- Domain code follows the project shape: immutable model (private fields + getters + Builder), `Interface` + `Impl` processor via `NewProcessor(l, ctx, db)`, GORM entity + `Migration`, `Make(e Entity) (Model, error)`, JSON:API resource via `AddRouteInitializer`. Test setup uses Builders — no `*_testhelpers.go`.
- Every coupon table carries `tenant_id`; every query scopes through `tenant.MustFromContext(ctx)`. No endpoint accepts a tenant id in a body.
- No `// TODO`, stubbed handler, or 501 in any landed commit.
- Commit after every task. Never commit to `main`; all work is on branch `task-206-cash-shop-coupon-codes` in `.worktrees/task-206-cash-shop-coupon-codes`.

### Already landed on this branch (do not redo)

| Commit | What |
|---|---|
| `15f81152` | `packet-audit operations` generates and checks `options.errors` alongside `options.operations` (`dispatcherDoc.Errors`, `tables()`, per-table DRIFT/MISSING/EXTRA). |
| `a36149bf` | `derivation.md` — gms_v83/v84/v87/v92/v95 coupon body, error enums, serverbound arm tables. |
| `4e43b563` | WZ/StringPool cross-check of the v83 error enum (2 rows promoted to `verified (cross-decompile)`). |

### Evidence-confidence caveat (carry into every template edit)

Most `errors` rows in `derivation.md` are marked `aligned` — the **byte** is read from the jump table (real), but the **English wording** behind each key is ordinal inference pinned by three anchors. Encoding an `aligned` byte is safe: the client renders *a* notice for it. Do not upgrade an `aligned` row to `verified` in any document without new decompile evidence.

---

## File Structure

**New files**

| Path | Responsibility |
|---|---|
| `libs/atlas-packet/cash/serverbound/coupon_code.go` | `CouponCode` immutable codec for the standalone `COUPON_CODE` op. |
| `libs/atlas-packet/cash/serverbound/coupon_code_test.go` | Round-trip fixture test + `packet-audit:verify` markers. |
| `docs/packets/dispatchers/cash_shop_coupon_code.yaml` | Handler doc that wires `CashShopCouponCodeHandle` into all in-scope templates (opcode only — the op has no mode table). |
| `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_coupon_code.go` | Channel handler arm. |
| `services/atlas-channel/atlas.com/channel/cashshop/coupon/normalize.go` | Shared normalization + plausibility rules (channel copy). |
| `services/atlas-cashshop/atlas.com/cashshop/coupon/normalize.go` | Same rules (service copy — the service must not trust its input). |
| `services/atlas-cashshop/atlas.com/cashshop/coupon/reward.go` | `Reward` discriminated value + jsonb marshalling. |
| `services/atlas-cashshop/atlas.com/cashshop/coupon/model.go` | Immutable `Model` + `Builder`. |
| `services/atlas-cashshop/atlas.com/cashshop/coupon/entity.go` | `Entity`, `Migration`, `Make`. |
| `services/atlas-cashshop/atlas.com/cashshop/coupon/provider.go` | Read providers. |
| `services/atlas-cashshop/atlas.com/cashshop/coupon/administrator.go` | Write operations incl. the conditional counter bump. |
| `services/atlas-cashshop/atlas.com/cashshop/coupon/granter.go` | `rewardGranter` interface + currency/cash-item implementations. |
| `services/atlas-cashshop/atlas.com/cashshop/coupon/processor.go` | `Interface` + `Impl`, the redemption transaction. |
| `services/atlas-cashshop/atlas.com/cashshop/coupon/limiter.go` | Per-account failed-attempt rate limiter. |
| `services/atlas-cashshop/atlas.com/cashshop/coupon/resource.go` / `rest.go` | JSON:API surface. |
| `services/atlas-cashshop/atlas.com/cashshop/coupon/batch/` | `coupon_batches` entity/model/provider/administrator/rest. |
| `services/atlas-cashshop/atlas.com/cashshop/coupon/redemption/` | `coupon_redemptions` entity/model/provider/administrator/rest. |
| `libs/atlas-redis/counter.go` (extend) | `TenantCounter.IncrWithTTL`. |
| `services/atlas-ui/src/services/api/coupons.service.ts` | API client. |
| `services/atlas-ui/src/lib/hooks/api/useCoupons.ts` | React Query hooks. |
| `services/atlas-ui/src/pages/CouponsPage.tsx`, `coupons-columns.tsx`, `CouponDetailPage.tsx` | Admin UI trio. |

**Modified files**

| Path | Change |
|---|---|
| `docs/packets/registry/gms_v84.yaml` | `COUPON_CODE` opcode `230` → `236`. |
| `docs/packets/registry/{gms_v83,gms_v84,gms_v87,gms_v92,gms_v95,jms_v185}.yaml` | Add `packet: cash/serverbound/CouponCode` to the `COUPON_CODE` entry. |
| `libs/atlas-packet/cash/clientbound/shop_operation_body.go:91` | `INVALID_COUPON_COUPON` → `INVALID_COUPON_CODE`. |
| `tools/packet-audit/cmd/operations.go` | `addEntry` emits `validator` + `services` and inserts in sorted opCode position. |
| `docs/packets/dispatchers/cash_shop_operation.yaml` | Add the `errors:` table; add the `gms_v92` column. |
| `services/atlas-configurations/seed-data/templates/template_*.json` | Generated `options.errors`; new `CashShopCouponCodeHandle` handler entry; `cashShop.coupons` config block. |
| `services/atlas-cashshop/atlas.com/cashshop/main.go` | Register three migrations, three resources, redis client. |
| `services/atlas-cashshop/atlas.com/cashshop/wallet/model.go` | Add `Award`. |
| `services/atlas-cashshop/atlas.com/cashshop/kafka/message/cashshop/kafka.go` | New command + status event contracts. |
| `services/atlas-cashshop/atlas.com/cashshop/kafka/consumer/cashshop/consumer.go` | `REQUEST_COUPON_REDEMPTION` arm. |
| `services/atlas-cashshop/atlas.com/cashshop/configuration/tenant/cashshop/rest.go` | `Coupons` config section. |
| `services/atlas-channel/atlas.com/channel/kafka/message/cashshop/`, `.../consumer/cashshop/consumer.go`, `.../cashshop/processor.go`, `.../cashshop/producer.go`, `main.go` | Mirror contracts, request path, response arms, handler registration. |
| `services/atlas-ui/src/App.tsx`, `src/components/app-sidebar-items.ts` | Route + nav. |
| `docs/tasks/task-206-cash-shop-coupon-codes/derivation.md` | Extended by Tasks 2, 3, 27. |
| `docs/tasks/task-206-cash-shop-coupon-codes/design.md` | maplePoint delta correction (Task 15). |

---

## Task Dependency Order

```
Phase 1  T1 registry ──┐
         T2 jms deriv ─┼──→ T4 codec ──→ T5 key-string fix
         T3 legacy n-a ┘         │
                                 ├──→ T6 addEntry ──→ T7 coupon handler YAML ──→ T24 channel handler
Phase 3                          └──→ T8 errors table ──────────────────────────→ T25 channel response
         T9 legacy/jms errors + v92 clientbound   (extends T8's YAML)
         T10 rate-limit config ──→ T17 limiter

Phase 4  T11 wallet.Award ──┐
         T12 normalize ─────┤
         T13 Reward ──→ T14 coupon entity ──→ T15 batch+redemption ──→ T18 providers/reservation
                                                                             │
                            └──────────────→ T19 granters ──→ T20 transaction ──→ T21 race tests
                                                                    │
                                                                    ├──→ T22 command consumer
                                                                    └──→ T23 admin REST

Phase 6  T23 ──→ T26 UI client ──→ T27 UI pages

Phase 7  T28 serverbound ops YAML (v83/84/87 + BUY_NORMAL fix)
              └──→ T29 v92/v95 completion
                        └──→ T30 verification gate  (needs every prior task)
```

**Parallelizable:** T1/T2/T3 are independent of each other. T10–T15 touch disjoint files and can proceed while Phase 1–3 run. T28/T29 are independent of the whole feature path and can run at any point after T6.

---

## Phase 1 — Registry corrections and derivation completion

### Task 1: Correct the `gms_v84` `COUPON_CODE` opcode and declare the codec path

The matrix promotes an `op`-kind cell from a byte-fixture only when the registry entry declares the atlas struct: `gradeOpCell` falls back to `ref.Packet` when there is no audit report (`tools/packet-audit/internal/matrix/grade.go:127-132`). Without a `packet:` line the Task 4 fixtures cannot promote any cell.

**Files:**
- Modify: `docs/packets/registry/gms_v84.yaml:3982-3987`
- Modify: `docs/packets/registry/gms_v83.yaml:3228-3232`
- Modify: `docs/packets/registry/gms_v87.yaml:3423-3427`
- Modify: `docs/packets/registry/gms_v92.yaml` (the `COUPON_CODE` entry)
- Modify: `docs/packets/registry/gms_v95.yaml:3802-3806`
- Modify: `docs/packets/registry/jms_v185.yaml:3373-3377`

**Interfaces:**
- Produces: registry op `COUPON_CODE` with `packet: cash/serverbound/CouponCode` on six versions, and `gms_v84` opcode `236`. Task 4 relies on both.

- [ ] **Step 1: Fix the v84 opcode**

`derivation.md` §gms_v84 "Registry bug": the only real coupon send is `COutPacket::COutPacket(&pkt, 236)` @ `0x473c84` in `CCashShop::OnStatusCoupon` @ `0x473bde`. The `68 E6 00 00 00` hit at `0x473fd1` is a dialog pixel coordinate, not a packet.

In `docs/packets/registry/gms_v84.yaml`, change the `COUPON_CODE` entry:

```yaml
- op: COUPON_CODE
  direction: serverbound
  opcode: 236
  fname: CCashShop::OnStatusCoupon
  packet: cash/serverbound/CouponCode
  provenance: manual
  note: >-
    task-206 corrected 230 -> 236. csv-import seeded 230 from the v83 column;
    the v84 client builds the coupon packet with COutPacket(&pkt, 236) at
    0x473c84 inside CCashShop::OnStatusCoupon @ 0x473bde. The 68 E6 00 00 00
    byte hit at 0x473fd1 is a CreateDlg pixel coordinate in sub_473F02, not a
    packet ctor. 236 = CASHSHOP_OPERATION(235) + 1, matching every other
    version's +1 relation. Evidence:
    docs/tasks/task-206-cash-shop-coupon-codes/derivation.md.
```

- [ ] **Step 2: Add `packet:` to the other five registries**

Add exactly one line, `  packet: cash/serverbound/CouponCode`, immediately after the `fname:` line of the `COUPON_CODE` entry in each of `gms_v83.yaml`, `gms_v87.yaml`, `gms_v92.yaml`, `gms_v95.yaml`, `jms_v185.yaml`. Do not change their opcodes — `derivation.md` confirms 230 / 243 / 269 / 276 and jms 246 is unverified but untouched by this task (Task 2 verifies it).

- [ ] **Step 3: Regenerate the matrix and confirm the opcode moved**

Run:
```bash
go run ./tools/packet-audit matrix
git diff --stat docs/packets/audits/status.json docs/packets/audits/STATUS.md
python3 -c "
import json
r=[x for x in json.load(open('docs/packets/audits/status.json'))['rows'] if x.get('op')=='COUPON_CODE'][0]
print({k:v.get('opcode') for k,v in r['cells'].items()})
print({k:v.get('state') for k,v in r['cells'].items()})
"
```
Expected: `gms_v84` opcode is `236` (was `230`); `gms_v48`/`v61`/`v72`/`v79` remain `n-a`; the six in-scope cells remain `incomplete` (no fixture yet).

- [ ] **Step 4: Run the registry-consistency checks**

Run: `go run ./tools/packet-audit fname-doc --check && go run ./tools/packet-audit operations --check`
Expected: both exit 0.

- [ ] **Step 5: Commit**

```bash
git add docs/packets/registry docs/packets/audits/status.json docs/packets/audits/STATUS.md
git commit -m "fix(packet-registry): correct gms_v84 COUPON_CODE opcode to 236 and declare the codec path"
```

---

### Task 2: Derive the `jms_v185` coupon body and error enum

`derivation.md` covers five GMS versions. `jms_v185` carries `COUPON_CODE` opcode 246 in the registry (`provenance: csv-import`, never IDA-confirmed) and its `CashShopOperation` writer needs an `errors` table like every other version. This task is a hard prerequisite for Task 4's jms branch and Task 8's jms column.

**Files:**
- Modify: `docs/tasks/task-206-cash-shop-coupon-codes/derivation.md` (new `## jms_v185` section before "Cross-version summary")

**Interfaces:**
- Produces: the jms `COUPON_CODE` opcode (confirmed or corrected), its `EncodeStr` count (2 or 3), and its `NoticeFailReason` reason-byte table. Tasks 4 and 8 consume both.

- [ ] **Step 1: Resolve the jms IDB session**

Use `mcp__ida-pro__idb_list` and match the binary by **NAME** (a JMS v185 client). `select_instance(port)` is dead — pass the resolved session id as the `database` parameter on every subsequent call. Record the binary name and session id in the derivation table at the top of `derivation.md`.

- [ ] **Step 2: Locate `CCashShop::OnStatusCoupon` and read the send**

```
func_query(database=<session>, name_regex="CCashShop::OnStatusCoupon")
decompile(database=<session>, address=<addr>)
```
Read off, and record with addresses:
- the `COutPacket::COutPacket(&pkt, <N>)` literal — this is the real opcode; compare to the registry's 246 and, if it differs, correct `docs/packets/registry/jms_v185.yaml` in this task with the same `note:` discipline Task 1 used for v84;
- every `COutPacket::EncodeStr` call site in the send path, in order, and whether any is guarded by an `if (field1 && *field1)`-style branch;
- confirm there is **no** `Encode1` mode byte before the first string.

If `func_query` returns nothing for the name, fall back to a byte search for the opcode push (`find_bytes` for `68 <op> 00 00 00` and `6A <op>`) restricted to the `CCashShop` address region, exactly as the GMS passes did, and name the function you find (`rename`) before recording it.

- [ ] **Step 3: Derive the error enum**

Find the failure-reason sink the same way the GMS passes did: decompile `CCashShop::OnCashItemResUseCouponFailed`, read the single `Decode1`, and follow the call it passes the byte to — that function is `NoticeFailReason`. Record:
- the jump-table bias (`add eax, <imm>` or `dec eax`), the `cmp eax, <N>` bound, IDA's "switch N cases" annotation, and the default-case set;
- whether the default-case set maps one-for-one onto the v83 set under a constant offset (that is the structural proof the GMS passes used — v84 = v83+9, v87 = v83+15, v92/v95 = v83−162);
- a call-site anchor: `CCashShop::OnStatusCoupon`'s cash-shop-disabled gate calls `NoticeFailReason(this, X)`; v83's X was 195, so `X − 195` must equal the derived offset.

- [ ] **Step 4: Write the section**

Append a `## jms_v185` section to `derivation.md` mirroring the existing per-version sections exactly: bullet header (opcodes, dispatcher address, failure sink address), a `### COUPON_CODE request body` table with one row per `EncodeStr` and an address in every Evidence cell, a `### errors enum` table with one row per Atlas key, and a **Row-count self-check** line. Mark each error row `verified` or `aligned` per the same rule the GMS sections use. If any key is out of the switch domain, record it as absent with the reason, exactly as the v92 section does for the two `*_WHEN_UNDER_SEVEN` keys.

Also add the jms row to the two tables in `## Cross-version summary of the two values the codec needs`.

- [ ] **Step 5: Verify no value was invented**

Re-read the new section and confirm every byte has an address or a stated structural derivation. A row with neither is a plan failure — delete it and mark the key absent instead.

- [ ] **Step 6: Commit**

```bash
git add docs/tasks/task-206-cash-shop-coupon-codes/derivation.md docs/packets/registry/jms_v185.yaml
git commit -m "docs(task-206): derive the jms_v185 coupon body and cash-shop error enum"
```

---

### Task 3: Turn the legacy `n-a` verdict into positive evidence

`gms_v48`/`v61`/`v72`/`v79` have no `COUPON_CODE` op in their registries, so the matrix already reports `n-a`. Registry-absence is not evidence of client-absence (`bug_matrix_redx_unverified_shared_codec` — an absent/❌ cell often means "nobody looked"). All four templates *do* bind the clientbound `USE_COUPON_SUCCESS`/`USE_COUPON_FAILED` writer modes (54/57, 61/64, 69/72, 81/84), so the receive half exists. This task settles the send half so the `n-a` cells survive the n-a consistency gate.

**Files:**
- Modify: `docs/tasks/task-206-cash-shop-coupon-codes/derivation.md` (new `## Legacy versions — COUPON_CODE applicability` section)
- Modify: `docs/packets/registry/gms_v{48,61,72,79}.yaml` (only if a send is found)

**Interfaces:**
- Produces: a per-legacy-version verdict, `absent` or `present + opcode`. Task 4 adds a version to the codec only for a `present` verdict; Task 7 wires a template handler entry only for a `present` verdict.

- [ ] **Step 1: For each of the four IDBs, look for a coupon send**

Resolve each session from `idb_list` by binary name. For each version:

1. `func_query(database=<session>, name_regex="OnStatusCoupon")` — a named hit settles it immediately.
2. If unnamed, find the version's `CASHSHOP_OPERATION` opcode from `docs/packets/registry/gms_v<N>.yaml` (v48 `0xA0`, v61 `0xC4`, v72 `0xDB`, v79 `0xDD` per the template handler opCodes) and byte-search for the **next** opcode's push — every derived GMS version puts `COUPON_CODE` at `CASHSHOP_OPERATION + 1`. Search `68 <op+1> 00 00 00` and `6A <op+1>`.
3. Cross-check by locating the coupon input dialog: search for the literal `"Please enter the coupon code."` (`find` / `search_text`) and follow its xref. Every derived version rejects an empty code with that exact literal immediately before the send, so an xref to it inside a `CCashShop` method is the strongest positive signal, and its total absence is the strongest negative one.

- [ ] **Step 2: Record the verdict per version**

Append to `derivation.md`:

```markdown
## Legacy versions — COUPON_CODE applicability

The four legacy templates bind the CLIENTBOUND coupon arms
(USE_COUPON_SUCCESS/USE_COUPON_FAILED at 54/57, 61/64, 69/72, 81/84), so the
receive half exists on all four. This section settles the SEND half, which is
what the registry and the coverage matrix key `n-a` on.

| version | IDB binary | session | OnStatusCoupon | "Please enter the coupon code." | opcode+1 push | verdict |
|---|---|---|---|---|---|---|
| gms_v48 | … | … | … | … | … | … |
| gms_v61 | … | … | … | … | … | … |
| gms_v72 | … | … | … | … | … | … |
| gms_v79 | … | … | … | … | … | … |
```

Fill every cell with an address or an explicit "not found (searched: <what>)". A blank cell is a plan failure.

- [ ] **Step 3: Act on a `present` verdict**

If any version *does* send a coupon packet, add its `COUPON_CODE` op to that registry with `opcode`, `fname`, `packet: cash/serverbound/CouponCode`, `provenance: manual`, and a `note:` citing the derivation address — and derive its request body (`EncodeStr` order) in the same section, because Task 4 will need it. Do not leave a discovered op unwired: that is exactly the deferral the project bans.

- [ ] **Step 4: Confirm the matrix still agrees**

Run:
```bash
go run ./tools/packet-audit matrix
go run ./tools/packet-audit fname-doc --check
```
Expected: versions with an `absent` verdict stay `n-a`; any version promoted in Step 3 flips to `incomplete` with its new opcode.

- [ ] **Step 5: Commit**

```bash
git add docs/tasks/task-206-cash-shop-coupon-codes/derivation.md docs/packets/registry docs/packets/audits
git commit -m "docs(task-206): evidence the legacy COUPON_CODE applicability verdict"
```

---

## Phase 2 — Packet layer

### Task 4: `CouponCode` serverbound codec

**Files:**
- Create: `libs/atlas-packet/cash/serverbound/coupon_code.go`
- Test: `libs/atlas-packet/cash/serverbound/coupon_code_test.go`

**Interfaces:**
- Consumes: Task 1's `packet: cash/serverbound/CouponCode` registry declaration; Task 2's jms body shape; Task 3's legacy verdict.
- Produces: `serverbound.CashShopCouponCodeHandle` (const `"CashShopCouponCodeHandle"`), `serverbound.CouponCode` with `TargetCharacter() string`, `Code() string`, `Extra() string`, `Operation() string`, `String() string`, `Encode(l, ctx)(options) []byte`, `Decode(l, ctx)(r, options)`. Tasks 7 and 22 consume the handle const; Task 22 consumes `Code()`.

**Wire shape (from `derivation.md` "Cross-version summary"):**

| version | opcode | body |
|---|---|---|
| gms_v83 | 230 / `0xE6` | `str targetCharacter` · `str code` · `str extra` *(third emitted only when `targetCharacter` is non-empty; dead on the plain-redeem path)* |
| gms_v84 | 236 / `0xEC` | same |
| gms_v87 | 243 / `0xF3` | same |
| gms_v92 | 269 / `0x10D` | `str targetCharacter` · `str code` — no third string, no guard |
| gms_v95 | 276 / `0x114` | same as v92 |
| jms_v185 | per Task 2 | per Task 2 |

`targetCharacter` is v95's `sCharacterID`, the second out-parameter of the coupon dialog (`CCouponUseSelectDlg::Confirm(&sCouponID, &sCharacterID)` @ `0x487f5a`). It is the *target* character for the reward, empty on the plain self-redeem path this task implements. Whether it carries a character name or a numeric id as text is **unverified** — say so in the doc comment and do not consume it in Task 22.

- [ ] **Step 1: Write the failing round-trip test**

Create `libs/atlas-packet/cash/serverbound/coupon_code_test.go`. Fill each `ida=` with the `CCashShop::OnStatusCoupon` address `derivation.md` records for that version (v83 `0x4710e8`, v84 `0x473bde`, v87 `0x47b9d4`, v92 `0x484430`, v95 `0x487ee0`, jms per Task 2):

```go
package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=cash/serverbound/CouponCode version=gms_v83 ida=0x4710e8
// packet-audit:verify packet=cash/serverbound/CouponCode version=gms_v84 ida=0x473bde
// packet-audit:verify packet=cash/serverbound/CouponCode version=gms_v87 ida=0x47b9d4
// packet-audit:verify packet=cash/serverbound/CouponCode version=gms_v92 ida=0x484430
// packet-audit:verify packet=cash/serverbound/CouponCode version=gms_v95 ida=0x487ee0
func TestCouponCodeRoundTripSelfRedeem(t *testing.T) {
	// The plain-redeem path the client actually takes: targetCharacter empty,
	// so the v83/v84/v87 conditional third string is never emitted and every
	// version is on the wire as [str ""][str code].
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := CouponCode{targetCharacter: "", code: "MAPLE2026"}
			output := CouponCode{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.TargetCharacter() != "" {
				t.Errorf("targetCharacter: got %q, want empty", output.TargetCharacter())
			}
			if output.Code() != input.Code() {
				t.Errorf("code: got %q, want %q", output.Code(), input.Code())
			}
			if output.Extra() != "" {
				t.Errorf("extra: got %q, want empty", output.Extra())
			}
		})
	}
}

func TestCouponCodeRoundTripTargetedRedeem(t *testing.T) {
	// targetCharacter non-empty: v83/v84/v87 add the guarded third string;
	// v92/v95 have no third string at all, so extra decodes back empty there.
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := CouponCode{targetCharacter: "Sidekick", code: "MAPLE2026", extra: "EXTRA"}
			output := CouponCode{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.TargetCharacter() != input.TargetCharacter() {
				t.Errorf("targetCharacter: got %q, want %q", output.TargetCharacter(), input.TargetCharacter())
			}
			if output.Code() != input.Code() {
				t.Errorf("code: got %q, want %q", output.Code(), input.Code())
			}
			want := input.Extra()
			if !hasTrailingCouponString(pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)) {
				want = ""
			}
			if output.Extra() != want {
				t.Errorf("extra: got %q, want %q for %s", output.Extra(), want, v.Name)
			}
		})
	}
}
```

- [ ] **Step 2: Run it and watch it fail**

Run: `cd libs/atlas-packet && go test ./cash/serverbound/ -run TestCouponCode -v`
Expected: FAIL — `undefined: CouponCode`, `undefined: hasTrailingCouponString`.

- [ ] **Step 3: Write the codec**

Create `libs/atlas-packet/cash/serverbound/coupon_code.go`:

```go
package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const CashShopCouponCodeHandle = "CashShopCouponCodeHandle"

// CouponCode - the standalone serverbound COUPON_CODE op. This is NOT a
// CASHSHOP_OPERATION mode arm: the client has no single cash-shop request
// builder, and the coupon submission gets its own opcode with NO leading mode
// byte (gms_v83 0xE6, v84 0xEC, v87 0xF3, v92 0x10D, v95 0x114). The body
// begins immediately with strings.
//
// Field 1 is v95's sCharacterID, the second out-parameter of
// CCouponUseSelectDlg::Confirm(&sCouponID, &sCharacterID) @ 0x487f5a — the
// character the reward is applied to. On the plain self-redeem path it is a
// zero-length string on every version (on v83 the dialog never writes it at
// all). Whether a populated value carries a character NAME or a numeric id as
// text is UNVERIFIED; nothing in Atlas consumes it today.
//
// Field 3 exists only on v83/v84/v87 and only when field 1 is non-empty
// (`if (field1 && *field1)` guard); v92 and v95 dropped it entirely. Its
// purpose is unknown / unverified.
//
// Derivation: docs/tasks/task-206-cash-shop-coupon-codes/derivation.md.
// packet-audit:fname CCashShop::OnStatusCoupon
type CouponCode struct {
	targetCharacter string
	code            string
	extra           string
}

func (m CouponCode) TargetCharacter() string { return m.targetCharacter }
func (m CouponCode) Code() string            { return m.code }
func (m CouponCode) Extra() string           { return m.extra }

func (m CouponCode) Operation() string {
	return CashShopCouponCodeHandle
}

func (m CouponCode) String() string {
	// The code is a secret; log its length, never its value.
	return fmt.Sprintf("targetCharacter [%s], code length [%d]", m.targetCharacter, len(m.code))
}

// hasTrailingCouponString reports whether this version's send path contains the
// third, conditionally-emitted EncodeStr. Derived per version: present on
// gms_v83/v84/v87, absent on gms_v92 and gms_v95 (2 EncodeStr sites each).
func hasTrailingCouponString(ctx context.Context) bool {
	t := tenant.MustFromContext(ctx)
	return t.IsRegion("GMS") && !t.MajorAtLeast(92)
}

func (m CouponCode) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(m.targetCharacter)
		w.WriteAsciiString(m.code)
		if hasTrailingCouponString(ctx) && m.targetCharacter != "" {
			w.WriteAsciiString(m.extra)
		}
		return w.Bytes()
	}
}

func (m *CouponCode) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.targetCharacter = r.ReadAsciiString()
		m.code = r.ReadAsciiString()
		if hasTrailingCouponString(ctx) && m.targetCharacter != "" {
			m.extra = r.ReadAsciiString()
		}
	}
}
```

**jms branch:** `hasTrailingCouponString` returns `false` for JMS because `IsRegion("GMS")` is false. That is correct **only if** Task 2 recorded two `EncodeStr` sites for jms_v185. If Task 2 recorded three with the same `if (field1 && *field1)` guard, change the predicate to `(t.IsRegion("GMS") && !t.MajorAtLeast(92)) || t.Region() == "JMS"` and update the doc comment to say so. Read `derivation.md`'s jms section before leaving this step.

**Legacy branch:** if Task 3's verdict was `absent` for all four, `pt.Variants` still runs them; they take the two-string path, which is unreachable in production because no template routes the opcode there. If Task 3 found a send on a legacy version, extend the predicate from that version's recorded `EncodeStr` count and add its `packet-audit:verify` marker line.

- [ ] **Step 4: Run the tests**

Run: `cd libs/atlas-packet && go test ./cash/serverbound/ -run TestCouponCode -v`
Expected: PASS for every variant.

- [ ] **Step 5: Promote the matrix cells**

Run:
```bash
go run ./tools/packet-audit matrix
python3 -c "
import json
r=[x for x in json.load(open('docs/packets/audits/status.json'))['rows'] if x.get('op')=='COUPON_CODE'][0]
print({k:v.get('state') for k,v in r['cells'].items()})
"
```
Expected: the six in-scope cells are no longer `incomplete`. A cell that does not move is a **failure to report, not to narrate around** — re-read `docs/packets/audits/VERIFYING_A_PACKET.md`, confirm the marker's `packet=` string exactly equals the registry's `packet:` value, and confirm the evidence record for that version exists and is fresh under `docs/packets/audits/evidence/<version>/`. Pin any missing evidence record before proceeding.

- [ ] **Step 6: Full module gates**

Run: `cd libs/atlas-packet && go test -race ./... && go vet ./...`
Expected: clean.

- [ ] **Step 7: Commit**

```bash
git add libs/atlas-packet/cash/serverbound/coupon_code.go libs/atlas-packet/cash/serverbound/coupon_code_test.go docs/packets/audits
git commit -m "feat(atlas-packet): add the serverbound COUPON_CODE codec across all applicable versions"
```

---

### Task 5: Correct the `INVALID_COUPON_COUPON` key string

`shop_operation_body.go:91` declares `CashShopOperationErrorInvalidCouponCode = "INVALID_COUPON_COUPON"` — a typo. Nothing consumes it yet, so correcting it now is free; after Task 8 writes it into ten templates it would not be. A mismatch between the constant and the templates' `errors` tables is silent: `ResolveCode` misses, logs, and returns 99 (`libs/atlas-packet/resolve.go:42-46`).

**Files:**
- Modify: `libs/atlas-packet/cash/clientbound/shop_operation_body.go:91`

**Interfaces:**
- Produces: `CashShopOperationErrorInvalidCouponCode == "INVALID_COUPON_CODE"`. Task 8's YAML key and Task 18's error mapping both use this exact string.

- [ ] **Step 1: Confirm nothing already depends on the typo'd string**

Run: `grep -rn "INVALID_COUPON_COUPON" --include='*.go' --include='*.json' --include='*.yaml' --include='*.ts' --include='*.tsx' .`
Expected: exactly one hit, the constant declaration itself. If a template or test also carries it, change those in this task too.

- [ ] **Step 2: Make the change**

In `libs/atlas-packet/cash/clientbound/shop_operation_body.go`:

```go
	CashShopOperationErrorInvalidCouponCode                 = "INVALID_COUPON_CODE"
```

- [ ] **Step 3: Build and test**

Run: `cd libs/atlas-packet && go build ./... && go test -race ./...`
Expected: clean.

- [ ] **Step 4: Commit**

```bash
git add libs/atlas-packet/cash/clientbound/shop_operation_body.go
git commit -m "fix(atlas-packet): correct the INVALID_COUPON_CODE error key string"
```

---

## Phase 3 — Configuration and templates

### Task 6: `packet-audit` `addEntry` must emit a validator and insert in sorted position

`addEntry` (`tools/packet-audit/cmd/operations.go:504-529`) creates a new template entry with exactly three keys — `opCode`, `handler`/`writer`, `options` — and **appends** it to the end of the array. Both are fatal for Task 7:

1. **No `validator`.** `BuildHandlerMap` looks the validator name up in `validatorMap` and `continue`s on a miss with only a `Warnf` (`libs/atlas-opcodes/producer.go:65-69`) — the handler is silently dropped and the opcode never routes. This is the known `bug_socket_handler_missing_validator_silently_dropped` failure mode.
2. **Append, not sorted insert.** `tools/template-opcode-order-guard.sh` fails on any entry whose opCode is lower than its predecessor's, and the coupon opcodes (`0xE6`, `0xEC`, `0xF3`, `0x10D`, `0x114`, jms) all sort well before each template's last handler.

`services` is left optional: `appliesToService` treats an empty list as "all services" (`libs/atlas-opcodes/config.go:36-39`), so omitting it is safe — but every hand-written handler entry carries `"services": ["channel"]`, so the generator should too when the YAML declares it.

**Files:**
- Modify: `tools/packet-audit/cmd/operations.go` (`dispatcherDoc`, `addEntry`)
- Test: `tools/packet-audit/cmd/operations_test.go`

**Interfaces:**
- Produces: `dispatcherDoc.Validator string` (yaml `validator`) and `dispatcherDoc.Services []string` (yaml `services`); `addEntry` emits `opCode`, `validator` (when declared), `handler`/`writer`, `options` (when any table is non-empty), `services` (when declared), inserted at the ascending-opCode position. Task 7's YAML sets both fields.

- [ ] **Step 1: Write the failing tests**

Append to `tools/packet-audit/cmd/operations_test.go`. Match the existing tests' construction of a temp dispatchers dir + templates dir; reuse whatever helper they already use to build a minimal template (read the file first and follow it rather than inventing a second harness).

```go
// A generated handler entry MUST carry a validator: BuildHandlerMap drops a
// handler whose validator name is not in the validator map, with only a
// Warnf, so a validator-less entry silently never routes.
func TestAddEntryEmitsValidatorAndServices(t *testing.T) {
	dir := t.TempDir()
	tplDir := writeMinimalTemplates(t, dir) // existing helper
	writeDispatcherDoc(t, dir, `
handler: CashShopCouponCodeHandle
validator: LoggedInValidator
services: [channel]
fname: CCashShop::OnStatusCoupon
op: COUPON_CODE
opcodes:
  gms_v83: "0xE6"
`)
	if code := operationsRun(operationsOpts{DispatchersDir: dir, TemplatesDir: tplDir}, io.Discard, io.Discard); code != 0 {
		t.Fatalf("generate exit = %d, want 0", code)
	}
	e := handlerEntry(t, filepath.Join(tplDir, "template_gms_83_1.json"), "CashShopCouponCodeHandle")
	if e["validator"] != "LoggedInValidator" {
		t.Errorf("validator = %v, want LoggedInValidator", e["validator"])
	}
	if got := e["services"]; !reflect.DeepEqual(got, []interface{}{"channel"}) {
		t.Errorf("services = %v, want [channel]", got)
	}
	if e["opCode"] != "0xE6" {
		t.Errorf("opCode = %v, want 0xE6", e["opCode"])
	}
	if _, ok := e["options"]; ok {
		t.Errorf("options should be omitted when the doc declares no tables, got %v", e["options"])
	}
}

// A generated entry must land at its ascending-opCode position so
// tools/template-opcode-order-guard.sh stays green.
func TestAddEntryInsertsInSortedPosition(t *testing.T) {
	dir := t.TempDir()
	tplDir := writeMinimalTemplates(t, dir) // must contain handlers 0x28, 0xE5, 0xFF
	writeDispatcherDoc(t, dir, `
handler: CashShopCouponCodeHandle
validator: LoggedInValidator
services: [channel]
op: COUPON_CODE
opcodes:
  gms_v83: "0xE6"
`)
	if code := operationsRun(operationsOpts{DispatchersDir: dir, TemplatesDir: tplDir}, io.Discard, io.Discard); code != 0 {
		t.Fatalf("generate exit = %d, want 0", code)
	}
	codes := handlerOpCodes(t, filepath.Join(tplDir, "template_gms_83_1.json"))
	for i := 1; i < len(codes); i++ {
		if codes[i] < codes[i-1] {
			t.Fatalf("handlers not ascending at %d: 0x%X after 0x%X (%v)", i, codes[i], codes[i-1], codes)
		}
	}
	if !slices.Contains(codes, 0xE6) {
		t.Fatalf("0xE6 not inserted: %v", codes)
	}
}
```

Add the two small helpers alongside if the file has no equivalent:

```go
func handlerEntry(t *testing.T, path, name string) map[string]interface{} {
	t.Helper()
	var d struct {
		Socket struct {
			Handlers []map[string]interface{} `json:"handlers"`
		} `json:"socket"`
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &d); err != nil {
		t.Fatal(err)
	}
	for _, e := range d.Socket.Handlers {
		if e["handler"] == name {
			return e
		}
	}
	t.Fatalf("handler %q not found in %s", name, path)
	return nil
}

func handlerOpCodes(t *testing.T, path string) []int {
	t.Helper()
	var d struct {
		Socket struct {
			Handlers []struct {
				OpCode string `json:"opCode"`
			} `json:"handlers"`
		} `json:"socket"`
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &d); err != nil {
		t.Fatal(err)
	}
	var out []int
	for _, e := range d.Socket.Handlers {
		n, err := strconv.ParseInt(e.OpCode, 0, 32)
		if err != nil {
			t.Fatalf("bad opCode %q", e.OpCode)
		}
		out = append(out, int(n))
	}
	return out
}
```

- [ ] **Step 2: Run and watch them fail**

Run: `cd tools/packet-audit && go test ./cmd/ -run 'TestAddEntry' -v`
Expected: FAIL — `validator` absent, and `0xE6` sitting after `0xFF`.

- [ ] **Step 3: Extend `dispatcherDoc`**

In `tools/packet-audit/cmd/operations.go`, add two fields next to `Opcodes`:

```go
	// Validator is the socket-config validator name for a generated HANDLER
	// entry. BuildHandlerMap (libs/atlas-opcodes/producer.go) looks this name
	// up and silently `continue`s past the handler when it is missing or
	// unknown, so a generated entry without it never routes. Required for any
	// doc that declares `opcodes` for a handler.
	Validator string `yaml:"validator"`
	// Services restricts the generated entry to specific services. Empty means
	// "all services" (libs/atlas-opcodes/config.go appliesToService), which is
	// safe but unconventional — every hand-written entry names its service.
	Services []string `yaml:"services"`
```

- [ ] **Step 4: Rewrite `addEntry`**

Replace the body of `addEntry` (`operations.go:504-529`) with:

```go
func addEntry(root *node, doc dispatcherDoc, version string, opcode string) bool {
	arr := arrayNode(root, doc.arrayKey())
	if arr == nil {
		return false
	}
	entryKey := doc.entryKey()

	w := &node{kind: 'o', dirty: true, obj: map[string]*node{}}
	put := func(k string, v interface{}) {
		b, _ := json.Marshal(v)
		w.keys = append(w.keys, k)
		w.obj[k] = &node{kind: 's', raw: b}
	}
	// Key order mirrors the hand-written entries: opCode, validator, handler,
	// fname, options, services.
	put("opCode", opcode)
	if doc.Validator != "" {
		put("validator", doc.Validator)
	}
	put(entryKey, doc.targetName())
	if doc.FName != "" {
		put("fname", doc.FName)
	}

	opts := &node{kind: 'o', obj: map[string]*node{}, dirty: true}
	for _, tb := range doc.tables() {
		expected := expectedFor(tb.entries, version)
		if len(expected) == 0 {
			continue
		}
		opts.keys = append(opts.keys, tb.name)
		opts.obj[tb.name] = buildTableNode(tb.entries, expected)
	}
	if len(opts.keys) > 0 {
		w.keys = append(w.keys, "options")
		w.obj["options"] = opts
	}
	if len(doc.Services) > 0 {
		sb, _ := json.Marshal(doc.Services)
		w.keys = append(w.keys, "services")
		w.obj["services"] = &node{kind: 'a', raw: sb, dirty: true}
	}

	insertSorted(arr, w, opcode)
	arr.dirty = true
	return true
}

// insertSorted places e at the first index whose opCode is greater than e's,
// keeping the array in the strictly-ascending order that
// tools/template-opcode-order-guard.sh enforces. An entry with an unparseable
// opCode is treated as "not greater" so a pre-existing malformed row cannot
// push the insertion point earlier than it belongs.
func insertSorted(arr *node, e *node, opcode string) {
	want, err := strconv.ParseInt(opcode, 0, 32)
	if err != nil {
		arr.arr = append(arr.arr, e)
		return
	}
	idx := len(arr.arr)
	for i, cur := range arr.arr {
		if cur.kind != 'o' {
			continue
		}
		oc, ok := cur.obj["opCode"]
		if !ok || oc.kind != 's' {
			continue
		}
		var s string
		if json.Unmarshal(oc.raw, &s) != nil {
			continue
		}
		got, cerr := strconv.ParseInt(s, 0, 32)
		if cerr != nil {
			continue
		}
		if got > want {
			idx = i
			break
		}
	}
	arr.arr = append(arr.arr, nil)
	copy(arr.arr[idx+1:], arr.arr[idx:])
	arr.arr[idx] = e
}
```

Confirm `strconv` and `encoding/json` are already imported in this file (they are — `operations.go:5,12`).

- [ ] **Step 5: Run the tests**

Run: `cd tools/packet-audit && go test ./... -v`
Expected: PASS, including the pre-existing `operations` tests — the new key ordering must not break them. If an existing test asserts the exact three-key shape, update that assertion to the new order and say so in the commit body.

- [ ] **Step 6: Confirm nothing regenerated unexpectedly**

Run: `go run ./tools/packet-audit operations --check && git status --short services/atlas-configurations/seed-data/templates`
Expected: `operations check OK`, no modified templates (no YAML declares a handler with `opcodes` yet).

- [ ] **Step 7: Commit**

```bash
git add tools/packet-audit/cmd/operations.go tools/packet-audit/cmd/operations_test.go
git commit -m "feat(packet-audit): generated socket entries carry a validator and sort by opCode"
```

---

### Task 7: Route `COUPON_CODE` in every applicable template

The op has **no mode table**, so this dispatcher doc exists purely to add the handler entry (opCode + validator + handler + fname + services) to each template. `packet-audit operations` supports exactly this: when a template lacks the entry and the YAML supplies the version's opcode, generate adds it and `--check` reports it missing (`operations.go:150-162`).

**Files:**
- Create: `docs/packets/dispatchers/cash_shop_coupon_code.yaml`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_{83,84,87,92,95}_1.json`, `template_jms_185_1.json` (generated)

**Interfaces:**
- Consumes: Task 4's `CashShopCouponCodeHandle` const, Task 6's `validator`/`services`/sorted insert, Task 2's jms opcode, Task 3's legacy verdict.
- Produces: a routed `CashShopCouponCodeHandle` in each applicable template. Task 22 registers the Go handler under the same name.

- [ ] **Step 1: Write the dispatcher doc**

Create `docs/packets/dispatchers/cash_shop_coupon_code.yaml`. Use the jms opcode Task 2 recorded (the registry's `246` if confirmed) and add any legacy version Task 3 found:

```yaml
# CashShopCouponCodeHandle — serverbound COUPON_CODE (CCashShop::OnStatusCoupon).
#
# SOURCE OF TRUTH for this handler's tenant-template registration.
# `packet-audit operations` adds the entry to any template missing it and
# `--check` fails when one is absent.
#
# This op has NO mode table. It is NOT an arm of CASHSHOP_OPERATION: the client
# has no single cash-shop request builder, and the coupon send is its own
# opcode whose body starts immediately with strings, with no leading Encode1.
# Consequently there is no `operations:` or `errors:` section here — only the
# per-version opcode. Adding a USE_COUPON key to CashShopOperationHandle would
# be wrong; see the "Structural finding" section of
# docs/tasks/task-206-cash-shop-coupon-codes/derivation.md.
#
# gms_v48/v61/v72/v79 are omitted: those clients have no coupon send (evidence
# in derivation.md "Legacy versions — COUPON_CODE applicability"), which is why
# their registries carry no COUPON_CODE op and the matrix reports them n-a.

handler: CashShopCouponCodeHandle
validator: LoggedInValidator
services: [channel]
fname: CCashShop::OnStatusCoupon
op: COUPON_CODE
direction: serverbound
opcodes:
  gms_v83: "0xE6"
  gms_v84: "0xEC"
  gms_v87: "0xF3"
  gms_v92: "0x10D"
  gms_v95: "0x114"
  jms_v185: "0xF6"
```

`0xF6` is the registry's `246` for jms. **Replace it with whatever Task 2 actually derived** — and if Task 2 corrected the registry, this value must match the corrected one.

- [ ] **Step 2: Generate and inspect the diff**

Run:
```bash
go run ./tools/packet-audit operations
git diff services/atlas-configurations/seed-data/templates
```
Expected: exactly six templates changed, each gaining one handler entry shaped like:

```json
    {
      "opCode": "0xE6",
      "validator": "LoggedInValidator",
      "handler": "CashShopCouponCodeHandle",
      "fname": "CCashShop::OnStatusCoupon",
      "services": [
        "channel"
      ]
    },
```

No `options` key (the doc declares no tables). No other template touched.

- [ ] **Step 3: Confirm the opcode is not already bound**

Run:
```bash
python3 - <<'PY'
import glob, json, os
want = {"template_gms_83_1.json":0xE6,"template_gms_84_1.json":0xEC,"template_gms_87_1.json":0xF3,
        "template_gms_92_1.json":0x10D,"template_gms_95_1.json":0x114,"template_jms_185_1.json":0xF6}
for p in sorted(glob.glob("services/atlas-configurations/seed-data/templates/template_*.json")):
    b=os.path.basename(p)
    if b not in want: continue
    hs=json.load(open(p))["socket"]["handlers"]
    hits=[h for h in hs if int(h["opCode"],16)==want[b]]
    print(b, [(h["opCode"],h.get("handler"),h.get("validator")) for h in hits])
PY
```
Expected: exactly one hit per template, `CashShopCouponCodeHandle` with `LoggedInValidator`. **Two hits means the opcode was already bound to something else** — stop, and resolve which handler owns it before continuing; a duplicate `(name, opCode)` pair is what `tools/template-duplicate-binding-guard.sh` bans and a duplicate opcode is last-write-wins in the dispatch map.

- [ ] **Step 4: Run the template guards**

Run:
```bash
tools/template-opcode-order-guard.sh
tools/template-duplicate-binding-guard.sh
tools/template-movement-types-guard.sh
go run ./tools/packet-audit operations --check
```
Expected: all four exit 0.

- [ ] **Step 5: Commit**

```bash
git add docs/packets/dispatchers/cash_shop_coupon_code.yaml services/atlas-configurations/seed-data/templates
git commit -m "feat(templates): route the serverbound COUPON_CODE handler in every applicable version"
```

---

### Task 8: Generate the `errors` table into every template

No template has an `errors` table today, so every `*_FAILED` cash-shop arm — coupon, buy, gift, storage — resolves its reason byte to 99 (`libs/atlas-packet/resolve.go:30-34`). Task 1's landed `packet-audit` change already generates and checks `options.errors`; this task supplies the data.

Because no template carries the table yet, there are no pre-existing keys to be reported `EXTRA`. Declaring a version's enum is therefore additive and safe — but declare each version **completely** from its `derivation.md` table, not just the seven coupon keys, so a later partial addition never trips the `expectedTable` all-or-nothing rule.

**Files:**
- Modify: `docs/packets/dispatchers/cash_shop_operation.yaml` (new `errors:` section)
- Modify: `services/atlas-configurations/seed-data/templates/template_*.json` (generated)

**Interfaces:**
- Consumes: Task 5's corrected `INVALID_COUPON_CODE` string; the per-version tables in `derivation.md`.
- Produces: `options.errors` on the `CashShopOperation` writer. Task 23's failure announcements resolve through it.

- [ ] **Step 1: Append the `errors:` section to `cash_shop_operation.yaml`**

Add after the existing `operations:` list. Every byte comes from `derivation.md`; the four columns are v83 (baseline), v84 (= v83 + 9), v87 (= v83 + 15), v92 and v95 (= v83 − 162, identical to each other), plus jms from Task 2 and the four legacy columns from Task 9 — **omit any column you do not yet have**, since an absent version is simply skipped (`len(expected) == 0 → continue`).

```yaml
# errors — the reason byte every *_FAILED arm carries, resolved by
# ResolveCode(l, options, "errors", key). Read from each version's
# CCashShop::NoticeFailReason jump table; no template carried this table before
# task-206, so every failure resolved to 99.
#
# CONFIDENCE. The BYTES are read from the jump table and are real. Most KEY
# NAMES are ordinal alignment of the case list against the declared order of
# the CashShopOperationError* constants, pinned by three anchors per version
# (see derivation.md). Rows marked `verified` there have direct evidence;
# rows marked `aligned` do not. Encoding an aligned byte is safe — the client
# renders a notice for it — but do not restate an aligned row as verified.
#
# RESERVED, DO NOT SEND as a generic failure: v83 0xA2/0xA4 and v84 171/173
# kick the player out of the cash shop; v83 0xB1, v84 186, v92/v95 15 show the
# wrong-coupon-number notice and then disconnect the cash shop. None of them
# is mapped to an Atlas key here.
#
# gms_v92 and gms_v95 renumber the whole enum to a 1-based scale and drop
# CANNOT_BE_PURCHASED_WHEN_UNDER_SEVEN / CANNOT_BE_RECEIVED_WHEN_UNDER_SEVEN
# out of the switch domain entirely — those two keys omit both columns.
errors:
  - key: REQUEST_TIMED_OUT
    modes: { gms_v83: 163, gms_v84: 172, gms_v87: 178, gms_v92: 1, gms_v95: 1 }
  - key: NOT_ENOUGH_CASH
    modes: { gms_v83: 165, gms_v84: 174, gms_v87: 180, gms_v92: 3, gms_v95: 3 }
  - key: CANNOT_GIFT_WHEN_UNDERAGE
    modes: { gms_v83: 166, gms_v84: 175, gms_v87: 181, gms_v92: 4, gms_v95: 4 }
  - key: EXCEEDED_GIFT_LIMIT
    modes: { gms_v83: 167, gms_v84: 176, gms_v87: 182, gms_v92: 5, gms_v95: 5 }
  - key: CANNOT_GIFT_TO_OWN_ACCOUNT
    modes: { gms_v83: 168, gms_v84: 177, gms_v87: 183, gms_v92: 6, gms_v95: 6 }
  - key: INCORRECT_NAME
    modes: { gms_v83: 169, gms_v84: 178, gms_v87: 184, gms_v92: 7, gms_v95: 7 }
  - key: CANNOT_GIFT_GENDER_RESTRICTION
    modes: { gms_v83: 170, gms_v84: 179, gms_v87: 185, gms_v92: 8, gms_v95: 8 }
  - key: CANNOT_GIFT_RECIPIENT_INVENTORY_FULL
    modes: { gms_v83: 171, gms_v84: 180, gms_v87: 186, gms_v92: 9, gms_v95: 9 }
  - key: EXCEEDED_CASH_ITEM_LIMIT
    modes: { gms_v83: 172, gms_v84: 181, gms_v87: 187, gms_v92: 10, gms_v95: 10 }
  - key: INCORRECT_NAME_OR_GENDER_RESTRICTION
    modes: { gms_v83: 173, gms_v84: 182, gms_v87: 188, gms_v92: 11, gms_v95: 11 }
  - key: INVALID_COUPON_CODE
    modes: { gms_v83: 176, gms_v84: 185, gms_v87: 191, gms_v92: 14, gms_v95: 14 }
  - key: COUPON_EXPIRED
    modes: { gms_v83: 178, gms_v84: 187, gms_v87: 193, gms_v92: 16, gms_v95: 16 }
  - key: COUPON_ALREADY_USED
    modes: { gms_v83: 179, gms_v84: 188, gms_v87: 194, gms_v92: 17, gms_v95: 17 }
  - key: COUPON_INTERNET_CAFE_RESTRICTION
    modes: { gms_v83: 180, gms_v84: 189, gms_v87: 195, gms_v92: 18, gms_v95: 18 }
  - key: INTERNET_CAFE_COUPON_ALREADY_USED
    modes: { gms_v83: 181, gms_v84: 190, gms_v87: 196, gms_v92: 19, gms_v95: 19 }
  - key: INTERNET_CAFE_COUPON_EXPIRED
    modes: { gms_v83: 182, gms_v84: 191, gms_v87: 197, gms_v92: 20, gms_v95: 20 }
  - key: COUPON_NOT_REGISTERED
    modes: { gms_v83: 183, gms_v84: 192, gms_v87: 198, gms_v92: 21, gms_v95: 21 }
  - key: COUPON_GENDER_RESTRICTION
    modes: { gms_v83: 184, gms_v84: 193, gms_v87: 199, gms_v92: 22, gms_v95: 22 }
  - key: COUPON_CANNOT_BE_GIFTED
    modes: { gms_v83: 185, gms_v84: 194, gms_v87: 200, gms_v92: 23, gms_v95: 23 }
  - key: COUPON_ONLY_FOR_MAPLE_STORY
    modes: { gms_v83: 186, gms_v84: 195, gms_v87: 201, gms_v92: 24, gms_v95: 24 }
  - key: INVENTORY_FULL
    modes: { gms_v83: 187, gms_v84: 196, gms_v87: 202, gms_v92: 25, gms_v95: 25 }
  - key: NOT_AVAILABLE_FOR_PURCHASE
    modes: { gms_v83: 188, gms_v84: 197, gms_v87: 203, gms_v92: 26, gms_v95: 26 }
  - key: CANNOT_GIFT_INVALID_NAME_OR_GENDER
    modes: { gms_v83: 189, gms_v84: 198, gms_v87: 204, gms_v92: 27, gms_v95: 27 }
  - key: CHECK_NAME_OF_RECEIVER
    modes: { gms_v83: 190, gms_v84: 199, gms_v87: 205, gms_v92: 28, gms_v95: 28 }
  - key: NOT_AVAILABLE_FOR_PURCHASE_AT_HOUR
    modes: { gms_v83: 191, gms_v84: 200, gms_v87: 206, gms_v92: 29, gms_v95: 29 }
  - key: OUT_OF_STOCK
    modes: { gms_v83: 192, gms_v84: 201, gms_v87: 207, gms_v92: 30, gms_v95: 30 }
  - key: EXCEEDED_SPENDING_LIMIT
    modes: { gms_v83: 193, gms_v84: 202, gms_v87: 208, gms_v92: 31, gms_v95: 31 }
  - key: NOT_ENOUGH_MESOS
    modes: { gms_v83: 194, gms_v84: 203, gms_v87: 209, gms_v92: 32, gms_v95: 32 }
  - key: CASH_SHOP_NOT_AVAILABLE_DURING_BETA
    modes: { gms_v83: 195, gms_v84: 204, gms_v87: 210, gms_v92: 33, gms_v95: 33 }
  - key: INVALID_BIRTHDAY
    modes: { gms_v83: 196, gms_v84: 205, gms_v87: 211, gms_v92: 34, gms_v95: 34 }
  - key: ONLY_AVAILABLE_TO_USERS_BUYING
    modes: { gms_v83: 199, gms_v84: 208, gms_v87: 214, gms_v92: 37, gms_v95: 37 }
  - key: ALREADY_APPLIED
    modes: { gms_v83: 200, gms_v84: 209, gms_v87: 215, gms_v92: 38, gms_v95: 38 }
  - key: DAILY_PURCHASE_LIMIT
    modes: { gms_v83: 205, gms_v84: 214, gms_v87: 220, gms_v92: 43, gms_v95: 43 }
  - key: COUPON_USAGE_LIMIT
    modes: { gms_v83: 208, gms_v84: 217, gms_v87: 223, gms_v92: 46, gms_v95: 46 }
  - key: COUPON_SYSTEM_AVAILABLE_SOON
    modes: { gms_v83: 210, gms_v84: 219, gms_v87: 225, gms_v92: 48, gms_v95: 48 }
  - key: FIFTEEN_DAY_LIMIT
    modes: { gms_v83: 211, gms_v84: 220, gms_v87: 226, gms_v92: 49, gms_v95: 49 }
  - key: NOT_ENOUGH_GIFT_TOKENS
    modes: { gms_v83: 212, gms_v84: 221, gms_v87: 227, gms_v92: 50, gms_v95: 50 }
  - key: CANNOT_SEND_TECHNICAL_DIFFICULTIES
    modes: { gms_v83: 213, gms_v84: 222, gms_v87: 228, gms_v92: 51, gms_v95: 51 }
  - key: CANNOT_GIFT_ACCOUNT_AGE
    modes: { gms_v83: 214, gms_v84: 223, gms_v87: 229, gms_v92: 52, gms_v95: 52 }
  - key: CANNOT_GIFT_PREVIOUS_INFRACTIONS
    modes: { gms_v83: 215, gms_v84: 224, gms_v87: 230, gms_v92: 53, gms_v95: 53 }
  - key: CANNOT_GIFT_AT_THIS_TIME
    modes: { gms_v83: 216, gms_v84: 225, gms_v87: 231, gms_v92: 54, gms_v95: 54 }
  - key: CANNOT_GIFT_LIMIT
    modes: { gms_v83: 217, gms_v84: 226, gms_v87: 232, gms_v92: 55, gms_v95: 55 }
  - key: CANNOT_GIFT_TECHNICAL_DIFFICULTIES
    modes: { gms_v83: 218, gms_v84: 227, gms_v87: 233, gms_v92: 56, gms_v95: 56 }
  - key: CANNOT_TRANSFER_UNDER_LEVEL_TWENTY
    modes: { gms_v83: 219, gms_v84: 228, gms_v87: 234, gms_v92: 57, gms_v95: 57 }
  - key: CANNOT_TRANSFER_TO_SAME_WORLD
    modes: { gms_v83: 220, gms_v84: 229, gms_v87: 235, gms_v92: 58, gms_v95: 58 }
  - key: CANNOT_TRANSFER_TO_NEW_WORLD
    modes: { gms_v83: 221, gms_v84: 230, gms_v87: 236, gms_v92: 59, gms_v95: 59 }
  - key: CANNOT_TRANSFER_OUT
    modes: { gms_v83: 222, gms_v84: 231, gms_v87: 237, gms_v92: 60, gms_v95: 60 }
  - key: CANNOT_TRANSFER_NO_EMPTY_SLOTS
    modes: { gms_v83: 223, gms_v84: 232, gms_v87: 238, gms_v92: 61, gms_v95: 61 }
  - key: EVENT_ENDED_OR_CANT_BE_FREELY_TESTED
    modes: { gms_v83: 224, gms_v84: 233, gms_v87: 239, gms_v92: 62, gms_v95: 62 }
  - key: CANNOT_BE_PURCHASED_WITH_MAPLE_POINTS
    modes: { gms_v83: 230, gms_v84: 239, gms_v87: 245, gms_v92: 68, gms_v95: 68 }
  - key: PLEASE_TRY_AGAIN
    modes: { gms_v83: 231, gms_v84: 240, gms_v87: 246, gms_v92: 69, gms_v95: 69 }
  # gms_v92/gms_v95 cap their switch at 69, so these two have no case there.
  - key: CANNOT_BE_PURCHASED_WHEN_UNDER_SEVEN
    modes: { gms_v83: 232, gms_v84: 241, gms_v87: 247 }
  - key: CANNOT_BE_RECEIVED_WHEN_UNDER_SEVEN
    modes: { gms_v83: 233, gms_v84: 242, gms_v87: 248 }
```

`UNKNOWN_ERROR` is deliberately **absent**: it is the jump table's default case on every version, i.e. "any byte not listed above". Mapping it to a specific byte would be wrong. Task 18 must therefore never send `UNKNOWN_ERROR` through `ResolveCode` expecting a configured value — see Task 23 Step 3.

Add jms_v185 to every row from Task 2's table, and the four legacy columns from Task 9.

- [ ] **Step 2: Cross-check the YAML against `derivation.md` mechanically**

Run:
```bash
python3 - <<'PY'
import re, yaml
doc = yaml.safe_load(open("docs/packets/dispatchers/cash_shop_operation.yaml"))
rows = {e["key"]: e["modes"] for e in doc.get("errors", [])}
txt = open("docs/tasks/task-206-cash-shop-coupon-codes/derivation.md").read()
# v83 rows are "| KEY | 0xNN | ..." ; v84/v87/v92/v95 are "| KEY | 0xNN (DDD) | ..."
bad = 0
for key, modes in rows.items():
    for ver, val in modes.items():
        pats = [r"\|\s*%s\s*\|\s*0x%02X\s*\|" % (re.escape(key), val),
                r"\|\s*%s\s*\|\s*0x%02X \(%d\)\s*\|" % (re.escape(key), val, val),
                r"\|\s*%s\s*\|\s*\*\*%d\*\*\s*\|" % (re.escape(key), val)]
        if not any(re.search(p, txt) for p in pats):
            print("NO EVIDENCE ROW: %s %s = %d (0x%02X)" % (ver, key, val, val)); bad += 1
print("checked %d keys; %d unevidenced" % (len(rows), bad))
PY
```
Expected: `0 unevidenced`. Any hit means the YAML carries a byte `derivation.md` does not — fix the YAML, never the record. (The check is per-value, not per-version, because the record's v84/v87/v92/v95 tables are single-column; a value that appears for the right key in *some* section is the signal that it was derived rather than invented. Spot-check three hits per version by eye against the correct section.)

- [ ] **Step 3: Generate and inspect**

Run:
```bash
go run ./tools/packet-audit operations
git diff --stat services/atlas-configurations/seed-data/templates
```
Expected: the five GMS templates with a declared column (plus jms/legacy if you added them) each gain an `options.errors` object on the `CashShopOperation` writer. Templates with no declared column are untouched.

Spot-check one:
```bash
python3 -c "
import json
d=json.load(open('services/atlas-configurations/seed-data/templates/template_gms_83_1.json'))
w=[x for x in d['socket']['writers'] if x.get('writer')=='CashShopOperation'][0]
e=w['options']['errors']
print(len(e), e['INVALID_COUPON_CODE'], e['COUPON_EXPIRED'], e['COUPON_ALREADY_USED'], e['COUPON_USAGE_LIMIT'], e['COUPON_NOT_REGISTERED'], e['INVENTORY_FULL'])
print('UNKNOWN_ERROR' in e)
"
```
Expected: `52 176 178 179 208 183 187` and `False`.

- [ ] **Step 4: Guards**

Run:
```bash
go run ./tools/packet-audit operations --check
tools/template-opcode-order-guard.sh
tools/template-duplicate-binding-guard.sh
```
Expected: all exit 0.

- [ ] **Step 5: Commit**

```bash
git add docs/packets/dispatchers/cash_shop_operation.yaml services/atlas-configurations/seed-data/templates
git commit -m "feat(templates): supply the CashShopOperation errors table from the IDA-derived enums"
```

---

### Task 9: Derive the legacy and jms error enums and close the `gms_v92` clientbound hole

Two independent gaps that both live in `cash_shop_operation.yaml`:

1. **Legacy `errors`.** `gms_v48`/`v61`/`v72`/`v79` (and `jms_v185`, if Task 2 did not already cover it) have no `errors` column, so their `BUY_FAILED` / `GIFT_FAILED` arms still resolve to 99. These versions are out of the coupon feature's scope but are in the design's Q3 scope ("the full per-version error enum, all ten versions").
2. **`gms_v92` clientbound `operations`.** `cash_shop_operation.yaml` has no `gms_v92` key anywhere, while `template_gms_92_1.json`'s `CashShopOperation` writer carries 57 `operations` keys. Because of the `len(expected) == 0 → continue` short-circuit, those 57 keys are validated by **nothing** — `--check` reports OK today only because v92 is invisible to it.

**Files:**
- Modify: `docs/tasks/task-206-cash-shop-coupon-codes/derivation.md`
- Modify: `docs/packets/dispatchers/cash_shop_operation.yaml`
- Modify: `services/atlas-configurations/seed-data/templates/template_*.json` (generated)

**Interfaces:**
- Produces: `errors` columns for the remaining versions, and a complete `gms_v92` column in the clientbound `operations` list.

- [ ] **Step 1: Derive each remaining version's error enum**

For `gms_v48`, `gms_v61`, `gms_v72`, `gms_v79` (and `jms_v185` if outstanding), repeat the mechanical procedure the five GMS passes used, per version:

1. Decompile `CCashShop::OnCashItemResUseCouponFailed` (or any `*Failed` arm) and follow its single `Decode1` to the reason sink — that function is `NoticeFailReason`. Name it with `rename` if the IDB has not.
2. Read the jump-table header: the bias (`add eax, <imm>` / `dec eax`), the `cmp eax, <N>` bound, IDA's "switch N cases" annotation, and the default-case set.
3. Test the constant-offset hypothesis: `v83_default_set + offset == this_version_default_set` as an exact set equality, with matching case count and range. That equality is the structural proof; without it, do **not** offset-derive — read each case body's StringPool id individually instead.
4. Anchor it: `CCashShop::OnStatusCoupon`'s (or the cash-shop-disabled gate's) `NoticeFailReason(this, X)` argument must satisfy `X − 195 == offset`.

Record each version as its own `### errors enum` subsection under a `## gms_v48` / `## gms_v61` / … heading, in the same table shape, with a **Row-count self-check** line. Where a version's switch domain is smaller than v83's, mark the out-of-domain keys absent with the reason — exactly as the v92 section does.

- [ ] **Step 2: Add those columns to the YAML**

Extend every `errors:` row in `cash_shop_operation.yaml` with the newly derived version keys. Re-run the Step-2 cross-check script from Task 8 and require `0 unevidenced`.

- [ ] **Step 3: Establish the `gms_v92` clientbound arm table**

`derivation.md` §gms_v92 "clientbound template diff" records that an IDA-derived v92 column **already exists** in `docs/tasks/task-183-cashshop-result-family/arm-catalog.md` (that table covers all nine tenant versions). Do not re-decompile all 57 arms:

1. Port the arm-catalog's v92 column into the `modes:` map of each `operations:` entry in `cash_shop_operation.yaml`.
2. Spot re-derive **at least five** arms directly from `CCashShop::OnCashItemResult` @ `0x495300` in the v92 IDB — including `USE_COUPON_SUCCESS` and `USE_COUPON_FAILED`, whose template values are 101 and 104 — and record the addresses in a new `### gms_v92 clientbound arm table` subsection of `derivation.md`. A spot-check that disagrees with the catalog invalidates the port: fall back to a full enumeration.
3. Diff the ported column against `template_gms_92_1.json`'s existing 57 keys and record the differences in that same subsection.

- [ ] **Step 4: Treat any v92 disagreement as a template bug**

Where the derived column and the template disagree, the **template** is wrong — its 57 keys were never validated against anything. Let `packet-audit operations` regenerate them, and record each changed key (old → new) in the derivation subsection so the wire change is auditable.

- [ ] **Step 5: Generate, verify, guard**

Run:
```bash
go run ./tools/packet-audit operations
git diff services/atlas-configurations/seed-data/templates
go run ./tools/packet-audit operations --check
tools/template-opcode-order-guard.sh
tools/template-duplicate-binding-guard.sh
go run ./tools/packet-audit matrix
```
Expected: `--check` and both guards exit 0. Review the v92 `operations` diff key by key against Step 3's recorded table before accepting it.

- [ ] **Step 6: Commit**

```bash
git add docs/tasks/task-206-cash-shop-coupon-codes/derivation.md docs/packets/dispatchers/cash_shop_operation.yaml services/atlas-configurations/seed-data/templates docs/packets/audits
git commit -m "feat(templates): complete the cash-shop errors enums and close the gms_v92 validation hole"
```

---

### Task 10: Tenant configuration for the coupon rate limiter

PRD §8 requires the brute-force threshold and window to be tenant configuration, not constants (DOM-25). The `cashShop` config block already exists in every template (`cashShop.commodities.hourlyExpirations`) and is served to `atlas-cashshop` through `configuration.GetTenantConfig`.

Per `feedback_config_table_all_versions`, the new block goes in **all eleven** templates including `template_gms_12_1.json` — a config table present on some versions and missing on others is the drift this project keeps paying for.

**Files:**
- Modify: `services/atlas-configurations/seed-data/templates/template_*.json` (all 11, hand-edited — this block is not tool-generated)
- Modify: `services/atlas-cashshop/atlas.com/cashshop/configuration/tenant/cashshop/rest.go`
- Create: `services/atlas-cashshop/atlas.com/cashshop/configuration/tenant/cashshop/coupons/rest.go`
- Modify: `services/atlas-cashshop/atlas.com/cashshop/configuration/registry.go`
- Test: `services/atlas-cashshop/atlas.com/cashshop/configuration/registry_test.go`

**Interfaces:**
- Produces: `configuration.GetCouponRateLimit(l, ctx, tenantId) (attempts uint32, window time.Duration)`. Task 16's limiter is its only consumer.

- [ ] **Step 1: Write the failing test**

Create/extend `services/atlas-cashshop/atlas.com/cashshop/configuration/registry_test.go`:

```go
func TestGetCouponRateLimitDefaults(t *testing.T) {
	// A tenant whose config omits the coupons block gets the documented
	// defaults, resolved HERE — never a magic number at the call site.
	l, _ := test.NewNullLogger()
	attempts, window := couponRateLimitFrom(tenant.RestModel{})
	if attempts != DefaultCouponAttempts {
		t.Errorf("attempts = %d, want %d", attempts, DefaultCouponAttempts)
	}
	if window != DefaultCouponWindow {
		t.Errorf("window = %v, want %v", window, DefaultCouponWindow)
	}
	_ = l
}

func TestGetCouponRateLimitFromConfig(t *testing.T) {
	cfg := tenant.RestModel{}
	cfg.CashShop.Coupons.RateLimit.Attempts = 3
	cfg.CashShop.Coupons.RateLimit.WindowSeconds = 900
	attempts, window := couponRateLimitFrom(cfg)
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
	if window != 15*time.Minute {
		t.Errorf("window = %v, want 15m", window)
	}
}

func TestGetCouponRateLimitRejectsZero(t *testing.T) {
	// A zero threshold would lock every account out of the coupon tab
	// permanently; a zero window would make the counter never expire.
	cfg := tenant.RestModel{}
	cfg.CashShop.Coupons.RateLimit.Attempts = 0
	cfg.CashShop.Coupons.RateLimit.WindowSeconds = 0
	attempts, window := couponRateLimitFrom(cfg)
	if attempts != DefaultCouponAttempts || window != DefaultCouponWindow {
		t.Errorf("zero config must fall back to defaults, got %d / %v", attempts, window)
	}
}
```

- [ ] **Step 2: Run and watch it fail**

Run: `cd services/atlas-cashshop && go test ./atlas.com/cashshop/configuration/... -v`
Expected: FAIL — undefined `couponRateLimitFrom`, `DefaultCouponAttempts`, `DefaultCouponWindow`, and no `Coupons` field.

- [ ] **Step 3: Add the config model**

Create `services/atlas-cashshop/atlas.com/cashshop/configuration/tenant/cashshop/coupons/rest.go`:

```go
package coupons

type RestModel struct {
	RateLimit RateLimitRestModel `json:"rateLimit"`
}

// RateLimitRestModel bounds coupon brute-forcing. Attempts is the number of
// FAILED redemption attempts one account may make inside WindowSeconds before
// further attempts short-circuit; WindowSeconds is the counter's TTL.
type RateLimitRestModel struct {
	Attempts      uint32 `json:"attempts"`
	WindowSeconds uint32 `json:"windowSeconds"`
}
```

Extend `services/atlas-cashshop/atlas.com/cashshop/configuration/tenant/cashshop/rest.go`:

```go
package cashshop

import (
	"atlas-cashshop/configuration/tenant/cashshop/commodities"
	"atlas-cashshop/configuration/tenant/cashshop/coupons"
)

type RestModel struct {
	Commodities commodities.RestModel `json:"commodities"`
	Coupons     coupons.RestModel     `json:"coupons"`
}
```

- [ ] **Step 4: Add the resolver**

Append to `services/atlas-cashshop/atlas.com/cashshop/configuration/registry.go`:

```go
// Documented defaults for the coupon rate limiter, applied when a tenant has
// not configured one. Resolved here alongside the other tenant defaults so no
// call site ever carries a magic number (DOM-25).
const (
	DefaultCouponAttempts = 10
	DefaultCouponWindow   = time.Hour
)

// GetCouponRateLimit returns the number of failed coupon attempts an account
// may make per window, and the window itself.
func GetCouponRateLimit(l logrus.FieldLogger, ctx context.Context, tenantId uuid.UUID) (uint32, time.Duration) {
	cfg, _ := GetTenantConfig(l, ctx, tenantId)
	return couponRateLimitFrom(cfg)
}

func couponRateLimitFrom(cfg tenant.RestModel) (uint32, time.Duration) {
	rl := cfg.CashShop.Coupons.RateLimit
	attempts := uint32(DefaultCouponAttempts)
	window := DefaultCouponWindow
	// Zero is "unset", not "zero allowed": a 0 threshold would lock every
	// account out of the coupon tab and a 0 window would make the Redis
	// counter immortal.
	if rl.Attempts > 0 {
		attempts = rl.Attempts
	}
	if rl.WindowSeconds > 0 {
		window = time.Duration(rl.WindowSeconds) * time.Second
	}
	return attempts, window
}
```

Add `"time"` to the imports.

- [ ] **Step 5: Run the tests**

Run: `cd services/atlas-cashshop && go test ./atlas.com/cashshop/configuration/... -v`
Expected: PASS.

- [ ] **Step 6: Seed the block in all eleven templates**

Add to each `template_*.json`'s top-level `cashShop` object, alongside `commodities`:

```json
    "coupons": {
      "rateLimit": {
        "attempts": 10,
        "windowSeconds": 3600
      }
    }
```

Per-file Edit, not a shell patch loop; preserve each file's existing line endings and indentation.

Verify:
```bash
python3 -c "
import glob, json, os
for p in sorted(glob.glob('services/atlas-configurations/seed-data/templates/template_*.json')):
    d=json.load(open(p))
    print(os.path.basename(p), d.get('cashShop',{}).get('coupons'))
"
```
Expected: all eleven print the same block; none prints `None`.

- [ ] **Step 7: Guards and commit**

Run: `tools/template-opcode-order-guard.sh && go run ./tools/packet-audit operations --check`
Expected: exit 0 (this block is outside `socket`, so neither is affected — run them to prove it).

```bash
git add services/atlas-configurations/seed-data/templates services/atlas-cashshop/atlas.com/cashshop/configuration
git commit -m "feat(cashshop): tenant-configured coupon rate limit with resolved defaults"
```

---

## Phase 4 — Coupon domain in `atlas-cashshop`

### Task 11: `wallet.Model.Award`

`wallet.Model` has `Purchase(currency, amount)` (a debit) but no credit operation. Coupon currency rewards need the symmetric one, with a **saturating** add — the balances are `uint32`, and a malformed reward that wraps a balance to near-zero is a silent, unrecoverable data loss.

**Files:**
- Modify: `services/atlas-cashshop/atlas.com/cashshop/wallet/model.go`
- Test: `services/atlas-cashshop/atlas.com/cashshop/wallet/model_test.go`

**Interfaces:**
- Produces: `func (m Model) Award(currency uint32, amount uint32) Model`. Task 18's `currencyGranter` is its only caller. Currency ids follow the existing `Balance` convention: `1` = credit (NX), `2` = Maple Points, anything else = prepaid.

- [ ] **Step 1: Write the failing tests**

Append to `services/atlas-cashshop/atlas.com/cashshop/wallet/model_test.go`:

```go
func TestAwardCreditsEachCurrency(t *testing.T) {
	base := Model{credit: 100, points: 200, prepaid: 300}
	for _, c := range []struct {
		name     string
		currency uint32
		credit   uint32
		points   uint32
		prepaid  uint32
	}{
		{"credit", 1, 150, 200, 300},
		{"points", 2, 100, 250, 300},
		{"prepaid", 3, 100, 200, 350},
		{"prepaid fallback for unknown currency", 99, 100, 200, 350},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := base.Award(c.currency, 50)
			if got.Credit() != c.credit || got.Points() != c.points || got.Prepaid() != c.prepaid {
				t.Errorf("Award(%d, 50) = (%d,%d,%d), want (%d,%d,%d)",
					c.currency, got.Credit(), got.Points(), got.Prepaid(), c.credit, c.points, c.prepaid)
			}
		})
	}
}

func TestAwardSaturatesInsteadOfWrapping(t *testing.T) {
	// A uint32 wrap would turn a huge reward into a near-zero balance and
	// silently destroy the account's existing funds.
	base := Model{credit: math.MaxUint32 - 5, points: math.MaxUint32, prepaid: 0}
	got := base.Award(1, 100)
	if got.Credit() != math.MaxUint32 {
		t.Errorf("credit = %d, want saturate at %d", got.Credit(), uint32(math.MaxUint32))
	}
	got = base.Award(2, 1)
	if got.Points() != math.MaxUint32 {
		t.Errorf("points = %d, want saturate at %d", got.Points(), uint32(math.MaxUint32))
	}
}

func TestAwardDoesNotMutateReceiver(t *testing.T) {
	base := Model{credit: 10}
	_ = base.Award(1, 5)
	if base.Credit() != 10 {
		t.Errorf("receiver mutated: credit = %d, want 10", base.Credit())
	}
}
```

Add `"math"` to the test file's imports.

- [ ] **Step 2: Run and watch it fail**

Run: `cd services/atlas-cashshop && go test ./atlas.com/cashshop/wallet/ -run TestAward -v`
Expected: FAIL — `base.Award undefined`.

- [ ] **Step 3: Implement**

Append to `services/atlas-cashshop/atlas.com/cashshop/wallet/model.go`:

```go
// Award credits amount to the given currency and returns a new Model. The add
// SATURATES at math.MaxUint32: balances are uint32, and wrapping a large
// reward around would silently destroy the account's existing funds rather
// than merely over-crediting. Currency ids follow the Balance convention:
// 1 = credit (NX), 2 = Maple Points, anything else = prepaid.
func (m Model) Award(currency uint32, amount uint32) Model {
	add := func(v uint32) uint32 {
		if v > math.MaxUint32-amount {
			return math.MaxUint32
		}
		return v + amount
	}

	newCredit := m.credit
	newPoints := m.points
	newPrepaid := m.prepaid
	if currency == 1 {
		newCredit = add(newCredit)
	} else if currency == 2 {
		newPoints = add(newPoints)
	} else {
		newPrepaid = add(newPrepaid)
	}

	return Model{
		id:        m.id,
		accountId: m.accountId,
		credit:    newCredit,
		points:    newPoints,
		prepaid:   newPrepaid,
	}
}
```

Add `"math"` to the file's imports.

- [ ] **Step 4: Run the tests**

Run: `cd services/atlas-cashshop && go test -race ./atlas.com/cashshop/wallet/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-cashshop/atlas.com/cashshop/wallet
git commit -m "feat(cashshop): add a saturating wallet.Model.Award"
```

---

### Task 12: Coupon code normalization rules

Codes are stored and looked up **normalized** — trimmed and uppercased — so the `(tenant_id, code)` unique index *is* the case-insensitive guarantee (PRD FR-5.3, design §7.1). The channel normalizes so the wire value and the stored value have one shape; the service normalizes again because it must not trust an input a future caller could also produce.

The two copies are deliberate (the services share no library today), so they must be kept byte-identical and each must carry a test proving the same table of cases.

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/cashshop/coupon/normalize.go`
- Test: `services/atlas-channel/atlas.com/channel/cashshop/coupon/normalize_test.go`
- Create: `services/atlas-cashshop/atlas.com/cashshop/coupon/normalize.go`
- Test: `services/atlas-cashshop/atlas.com/cashshop/coupon/normalize_test.go`

**Interfaces:**
- Produces, in **both** packages: `const MaxCodeLength = 32`, `func Normalize(code string) string`, `func Plausible(code string) bool`. Task 19 (service) and Task 23 (channel) are the consumers.

- [ ] **Step 1: Write the shared test table**

Create `services/atlas-cashshop/atlas.com/cashshop/coupon/normalize_test.go` (and, in Step 4, the byte-identical channel copy with only the `package` line differing):

```go
package coupon

import "testing"

func TestNormalize(t *testing.T) {
	for _, c := range []struct{ name, in, want string }{
		{"uppercases", "maple2026", "MAPLE2026"},
		{"trims both ends", "  MAPLE2026  ", "MAPLE2026"},
		{"trims tabs and newlines", "\t MAPLE2026 \r\n", "MAPLE2026"},
		{"leaves an already-normal code alone", "MAPLE2026", "MAPLE2026"},
		{"does not strip interior spaces", "MAPLE 2026", "MAPLE 2026"},
		{"empty stays empty", "   ", ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := Normalize(c.in); got != c.want {
				t.Errorf("Normalize(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestPlausible(t *testing.T) {
	for _, c := range []struct {
		name string
		in   string
		want bool
	}{
		{"normal code", "MAPLE2026", true},
		{"single character", "A", true},
		{"exactly the column limit", "12345678901234567890123456789012", true},
		{"empty", "", false},
		{"one over the column limit", "123456789012345678901234567890123", false},
		{"un-normalized input is not plausible", " maple ", false},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := Plausible(c.in); got != c.want {
				t.Errorf("Plausible(%q) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run and watch it fail**

Run: `cd services/atlas-cashshop && go test ./atlas.com/cashshop/coupon/ -v`
Expected: FAIL — undefined `Normalize`, `Plausible`.

- [ ] **Step 3: Implement the service copy**

Create `services/atlas-cashshop/atlas.com/cashshop/coupon/normalize.go`:

```go
package coupon

import "strings"

// MaxCodeLength matches the coupons.code column width. A submission longer
// than this cannot match any stored code, so it is rejected without a query.
const MaxCodeLength = 32

// Normalize is the ONE canonical form a coupon code takes: surrounding
// whitespace trimmed, then uppercased. Codes are STORED normalized, so the
// unique index on (tenant_id, code) is what makes lookups case-insensitive —
// there is no functional index over a raw value.
//
// Interior whitespace is deliberately preserved: it is part of the code.
//
// This function is duplicated verbatim in
// services/atlas-channel/atlas.com/channel/cashshop/coupon/normalize.go. The
// two services share no library; if you change one, change both, and keep
// both test tables identical.
func Normalize(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

// Plausible reports whether an ALREADY-NORMALIZED code could possibly match a
// stored coupon. It is a cheap local gate — the first line of brute-force
// defence — not a validity check.
func Plausible(code string) bool {
	if code == "" || len(code) > MaxCodeLength {
		return false
	}
	return code == Normalize(code)
}
```

- [ ] **Step 4: Mirror into the channel**

Create `services/atlas-channel/atlas.com/channel/cashshop/coupon/normalize.go` and `normalize_test.go` as byte-identical copies of the two service files, changing nothing but keeping the same `package coupon` clause. Confirm:

```bash
diff <(tail -n +1 services/atlas-cashshop/atlas.com/cashshop/coupon/normalize.go) \
     <(tail -n +1 services/atlas-channel/atlas.com/channel/cashshop/coupon/normalize.go) \
  && echo IDENTICAL
```
Expected: `IDENTICAL` (adjust only the cross-reference comment's path so each file points at the *other* one).

- [ ] **Step 5: Run both test suites**

Run:
```bash
cd services/atlas-cashshop && go test -race ./atlas.com/cashshop/coupon/ -v && cd -
cd services/atlas-channel && go test -race ./atlas.com/channel/cashshop/coupon/ -v && cd -
```
Expected: PASS in both.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-cashshop/atlas.com/cashshop/coupon services/atlas-channel/atlas.com/channel/cashshop/coupon
git commit -m "feat(coupon): canonical code normalization and plausibility rules"
```

---

### Task 13: `Reward` model

Rewards are stored as `jsonb` on the coupon row — always read and written as a whole bundle, never queried by reward attribute (PRD §6, design §7.1). Two reward types are in scope; mesos and regular-inventory items are explicit non-goals (PRD §2).

**Files:**
- Create: `services/atlas-cashshop/atlas.com/cashshop/coupon/reward.go`
- Test: `services/atlas-cashshop/atlas.com/cashshop/coupon/reward_test.go`

**Interfaces:**
- Produces: `RewardType` (`RewardTypeCurrency` = `"CURRENCY"`, `RewardTypeCashItem` = `"CASH_ITEM"`), `Reward` with `Type()`, `Currency()`, `Amount()`, `SerialNumber()`, `Quantity()`, `Validate() error`; constructors `NewCurrencyReward(currency, amount uint32) Reward` and `NewCashItemReward(serialNumber, quantity uint32) Reward`; `Rewards []Reward` implementing `driver.Valuer` / `sql.Scanner`. Tasks 14, 18 and 22 consume these.

- [ ] **Step 1: Write the failing tests**

Create `services/atlas-cashshop/atlas.com/cashshop/coupon/reward_test.go`:

```go
package coupon

import (
	"encoding/json"
	"testing"
)

func TestRewardRoundTripsThroughJSON(t *testing.T) {
	in := Rewards{
		NewCurrencyReward(2, 10000),
		NewCashItemReward(50200000, 1),
	}
	v, err := in.Value()
	if err != nil {
		t.Fatalf("Value: %v", err)
	}
	var out Rewards
	if err := out.Scan(v); err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("len = %d, want 2", len(out))
	}
	if out[0].Type() != RewardTypeCurrency || out[0].Currency() != 2 || out[0].Amount() != 10000 {
		t.Errorf("currency reward = %+v", out[0])
	}
	if out[1].Type() != RewardTypeCashItem || out[1].SerialNumber() != 50200000 || out[1].Quantity() != 1 {
		t.Errorf("cash item reward = %+v", out[1])
	}
}

func TestRewardsScanNil(t *testing.T) {
	var out Rewards
	if err := out.Scan(nil); err != nil {
		t.Fatalf("Scan(nil): %v", err)
	}
	if len(out) != 0 {
		t.Errorf("len = %d, want 0", len(out))
	}
}

func TestRewardJSONShapeMatchesTheRESTContract(t *testing.T) {
	b, err := json.Marshal(NewCurrencyReward(1, 5))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"type":"CURRENCY","currency":1,"amount":5}` {
		t.Errorf("currency JSON = %s", b)
	}
	b, err = json.Marshal(NewCashItemReward(50200000, 2))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"type":"CASH_ITEM","serialNumber":50200000,"quantity":2}` {
		t.Errorf("cash item JSON = %s", b)
	}
}

func TestRewardValidate(t *testing.T) {
	for _, c := range []struct {
		name    string
		reward  Reward
		wantErr bool
	}{
		{"valid currency", NewCurrencyReward(1, 100), false},
		{"valid cash item", NewCashItemReward(50200000, 1), false},
		{"zero currency amount", NewCurrencyReward(1, 0), true},
		{"zero serial number", NewCashItemReward(0, 1), true},
		{"zero quantity", NewCashItemReward(50200000, 0), true},
		{"unknown type", Reward{rewardType: "MESO", amount: 1}, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			if err := c.reward.Validate(); (err != nil) != c.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, c.wantErr)
			}
		})
	}
}
```

- [ ] **Step 2: Run and watch it fail**

Run: `cd services/atlas-cashshop && go test ./atlas.com/cashshop/coupon/ -run 'TestReward' -v`
Expected: FAIL — undefined symbols.

- [ ] **Step 3: Implement**

Create `services/atlas-cashshop/atlas.com/cashshop/coupon/reward.go`:

```go
package coupon

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
)

type RewardType string

const (
	RewardTypeCurrency RewardType = "CURRENCY"
	RewardTypeCashItem RewardType = "CASH_ITEM"
)

var ErrInvalidReward = errors.New("invalid reward")

// Reward is a discriminated value. Only the fields belonging to its Type are
// meaningful; the others are zero and are omitted from JSON.
//
// Currency ids reuse the existing wallet.Model.Balance convention (1 = credit
// / NX, 2 = Maple Points, anything else = prepaid) rather than introducing a
// second enum for the same thing (DOM-21).
//
// Mesos and regular-inventory items are explicit non-goals (PRD §2); adding
// either means adding a RewardType here AND a granter in granter.go, and — for
// a reward owned by another service — is the point at which the local
// redemption transaction has to become a saga (design §2.1).
type Reward struct {
	rewardType   RewardType
	currency     uint32
	amount       uint32
	serialNumber uint32
	quantity     uint32
}

func NewCurrencyReward(currency uint32, amount uint32) Reward {
	return Reward{rewardType: RewardTypeCurrency, currency: currency, amount: amount}
}

func NewCashItemReward(serialNumber uint32, quantity uint32) Reward {
	return Reward{rewardType: RewardTypeCashItem, serialNumber: serialNumber, quantity: quantity}
}

func (r Reward) Type() RewardType    { return r.rewardType }
func (r Reward) Currency() uint32    { return r.currency }
func (r Reward) Amount() uint32      { return r.amount }
func (r Reward) SerialNumber() uint32 { return r.serialNumber }
func (r Reward) Quantity() uint32    { return r.quantity }

func (r Reward) Validate() error {
	switch r.rewardType {
	case RewardTypeCurrency:
		if r.amount == 0 {
			return fmt.Errorf("%w: currency reward amount must be positive", ErrInvalidReward)
		}
		return nil
	case RewardTypeCashItem:
		if r.serialNumber == 0 {
			return fmt.Errorf("%w: cash item reward needs a serial number", ErrInvalidReward)
		}
		if r.quantity == 0 {
			return fmt.Errorf("%w: cash item reward quantity must be positive", ErrInvalidReward)
		}
		return nil
	default:
		return fmt.Errorf("%w: unknown reward type %q", ErrInvalidReward, r.rewardType)
	}
}

// rewardJSON is the on-disk and on-the-wire shape. It is deliberately the same
// document in the jsonb column and in the REST attribute, so an admin editing
// a bundle sees exactly what is stored.
type rewardJSON struct {
	Type         RewardType `json:"type"`
	Currency     uint32     `json:"currency,omitempty"`
	Amount       uint32     `json:"amount,omitempty"`
	SerialNumber uint32     `json:"serialNumber,omitempty"`
	Quantity     uint32     `json:"quantity,omitempty"`
}

func (r Reward) MarshalJSON() ([]byte, error) {
	return json.Marshal(rewardJSON{
		Type:         r.rewardType,
		Currency:     r.currency,
		Amount:       r.amount,
		SerialNumber: r.serialNumber,
		Quantity:     r.quantity,
	})
}

func (r *Reward) UnmarshalJSON(b []byte) error {
	var j rewardJSON
	if err := json.Unmarshal(b, &j); err != nil {
		return err
	}
	r.rewardType = j.Type
	r.currency = j.Currency
	r.amount = j.Amount
	r.serialNumber = j.SerialNumber
	r.quantity = j.Quantity
	return nil
}

// Rewards is the whole bundle, persisted as one jsonb document.
type Rewards []Reward

func (rs Rewards) Value() (driver.Value, error) {
	if rs == nil {
		return []byte("[]"), nil
	}
	return json.Marshal([]Reward(rs))
}

func (rs *Rewards) Scan(src interface{}) error {
	if src == nil {
		*rs = nil
		return nil
	}
	var b []byte
	switch v := src.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return fmt.Errorf("coupon: cannot scan %T into Rewards", src)
	}
	if len(b) == 0 {
		*rs = nil
		return nil
	}
	return json.Unmarshal(b, (*[]Reward)(rs))
}

func (rs Rewards) Validate() error {
	if len(rs) == 0 {
		return fmt.Errorf("%w: a coupon must grant at least one reward", ErrInvalidReward)
	}
	for i, r := range rs {
		if err := r.Validate(); err != nil {
			return fmt.Errorf("reward %d: %w", i, err)
		}
	}
	return nil
}

// CashItemCount is the number of locker slots this bundle needs.
func (rs Rewards) CashItemCount() int {
	n := 0
	for _, r := range rs {
		if r.Type() == RewardTypeCashItem {
			n++
		}
	}
	return n
}
```

- [ ] **Step 4: Run the tests**

Run: `cd services/atlas-cashshop && go test -race ./atlas.com/cashshop/coupon/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-cashshop/atlas.com/cashshop/coupon/reward.go services/atlas-cashshop/atlas.com/cashshop/coupon/reward_test.go
git commit -m "feat(coupon): reward bundle model persisted as jsonb"
```

---

### Task 14: Coupon entity, model, and migration

**Files:**
- Create: `services/atlas-cashshop/atlas.com/cashshop/coupon/entity.go`
- Create: `services/atlas-cashshop/atlas.com/cashshop/coupon/model.go`
- Test: `services/atlas-cashshop/atlas.com/cashshop/coupon/model_test.go`
- Modify: `services/atlas-cashshop/atlas.com/cashshop/main.go:57`

**Interfaces:**
- Consumes: `Rewards` from Task 13.
- Produces: `coupon.Entity` (table `coupons`), `coupon.Migration`, `coupon.Model` with `Id() uuid.UUID`, `BatchId() uuid.UUID`, `Code() string`, `Description() string`, `Active() bool`, `StartsAt() *time.Time`, `ExpiresAt() *time.Time`, `MaxUses() *uint32`, `RedemptionCount() uint32`, `Rewards() Rewards`, plus `NewBuilder(code string) *Builder` with `SetBatchId`/`SetDescription`/`SetActive`/`SetStartsAt`/`SetExpiresAt`/`SetMaxUses`/`SetRedemptionCount`/`SetRewards`/`Build()`, and `Make(e Entity) (Model, error)`. Also `func (m Model) RedeemableAt(now time.Time) error` returning the first ladder failure. Tasks 15, 19 and 22 consume these.

- [ ] **Step 1: Write the failing tests**

Create `services/atlas-cashshop/atlas.com/cashshop/coupon/model_test.go`:

```go
package coupon

import (
	"errors"
	"testing"
	"time"
)

func ptrTime(t time.Time) *time.Time { return &t }
func ptrU32(v uint32) *uint32        { return &v }

func TestRedeemableAtLadderOrder(t *testing.T) {
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	for _, c := range []struct {
		name    string
		build   func() Model
		wantKey string
	}{
		{
			"active, open window, uses left",
			func() Model {
				m, _ := NewBuilder("OK").SetRewards(Rewards{NewCurrencyReward(1, 1)}).Build()
				return m
			},
			"",
		},
		{
			"inactive reports NOT_REGISTERED",
			func() Model {
				m, _ := NewBuilder("OFF").SetActive(false).SetRewards(Rewards{NewCurrencyReward(1, 1)}).Build()
				return m
			},
			ErrorKeyNotRegistered,
		},
		{
			"before startsAt reports NOT_REGISTERED",
			func() Model {
				m, _ := NewBuilder("EARLY").SetStartsAt(ptrTime(future)).SetRewards(Rewards{NewCurrencyReward(1, 1)}).Build()
				return m
			},
			ErrorKeyNotRegistered,
		},
		{
			"after expiresAt reports EXPIRED",
			func() Model {
				m, _ := NewBuilder("OLD").SetExpiresAt(ptrTime(past)).SetRewards(Rewards{NewCurrencyReward(1, 1)}).Build()
				return m
			},
			ErrorKeyExpired,
		},
		{
			"exhausted reports USAGE_LIMIT",
			func() Model {
				m, _ := NewBuilder("USED").SetMaxUses(ptrU32(1)).SetRedemptionCount(1).SetRewards(Rewards{NewCurrencyReward(1, 1)}).Build()
				return m
			},
			ErrorKeyUsageLimit,
		},
		{
			"inactive AND expired reports NOT_REGISTERED — inactive wins",
			func() Model {
				m, _ := NewBuilder("BOTH").SetActive(false).SetExpiresAt(ptrTime(past)).SetRewards(Rewards{NewCurrencyReward(1, 1)}).Build()
				return m
			},
			ErrorKeyNotRegistered,
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			err := c.build().RedeemableAt(now)
			if c.wantKey == "" {
				if err != nil {
					t.Fatalf("want redeemable, got %v", err)
				}
				return
			}
			var re *RedemptionError
			if !errors.As(err, &re) {
				t.Fatalf("want a *RedemptionError, got %v", err)
			}
			if re.Key() != c.wantKey {
				t.Errorf("key = %q, want %q", re.Key(), c.wantKey)
			}
		})
	}
}

func TestBuilderDefaultsAndNormalization(t *testing.T) {
	m, err := NewBuilder("  maple2026 ").SetRewards(Rewards{NewCurrencyReward(1, 1)}).Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if m.Code() != "MAPLE2026" {
		t.Errorf("code = %q, want MAPLE2026 (the builder normalizes)", m.Code())
	}
	if !m.Active() {
		t.Error("active should default true")
	}
	if m.MaxUses() != nil {
		t.Error("maxUses should default nil (unlimited)")
	}
	if m.RedemptionCount() != 0 {
		t.Error("redemptionCount should default 0")
	}
}

func TestBuilderRejectsAnEmptyOrInvalidCoupon(t *testing.T) {
	if _, err := NewBuilder("   ").SetRewards(Rewards{NewCurrencyReward(1, 1)}).Build(); err == nil {
		t.Error("want an error for an empty code")
	}
	if _, err := NewBuilder("OK").Build(); err == nil {
		t.Error("want an error for a coupon with no rewards")
	}
	now := time.Now()
	if _, err := NewBuilder("OK").
		SetRewards(Rewards{NewCurrencyReward(1, 1)}).
		SetStartsAt(ptrTime(now.Add(time.Hour))).
		SetExpiresAt(ptrTime(now)).
		Build(); err == nil {
		t.Error("want an error when expiresAt <= startsAt")
	}
}
```

- [ ] **Step 2: Run and watch it fail**

Run: `cd services/atlas-cashshop && go test ./atlas.com/cashshop/coupon/ -run 'TestRedeemable|TestBuilder' -v`
Expected: FAIL — undefined `NewBuilder`, `RedemptionError`, `ErrorKey*`.

- [ ] **Step 3: Write the entity**

Create `services/atlas-cashshop/atlas.com/cashshop/coupon/entity.go`:

```go
package coupon

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}

// Entity is the coupons table. Code is stored NORMALIZED (trimmed, uppercased),
// so uniqueIndex on (tenant_id, code) IS the case-insensitive uniqueness
// guarantee — do not add a functional index over a raw value.
//
// Rewards is jsonb because the bundle is always read and written whole and is
// never queried by reward attribute.
type Entity struct {
	Id              uuid.UUID  `gorm:"primaryKey;type:uuid"`
	TenantId        uuid.UUID  `gorm:"not null;index;uniqueIndex:idx_coupons_tenant_code;index:idx_coupons_tenant_batch,priority:1"`
	BatchId         *uuid.UUID `gorm:"type:uuid;index:idx_coupons_tenant_batch,priority:2"`
	Code            string     `gorm:"not null;type:varchar(32);uniqueIndex:idx_coupons_tenant_code"`
	Description     string     `gorm:"type:text"`
	Active          bool       `gorm:"not null;default:true"`
	StartsAt        *time.Time
	ExpiresAt       *time.Time
	MaxUses         *uint32
	RedemptionCount uint32  `gorm:"not null;default:0"`
	Rewards         Rewards `gorm:"not null;type:jsonb"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (e Entity) TableName() string {
	return "coupons"
}

func (e *Entity) BeforeCreate(_ *gorm.DB) (err error) {
	if e.Id == uuid.Nil {
		e.Id = uuid.New()
	}
	return
}

func Make(e Entity) (Model, error) {
	batchId := uuid.Nil
	if e.BatchId != nil {
		batchId = *e.BatchId
	}
	return Model{
		id:              e.Id,
		batchId:         batchId,
		code:            e.Code,
		description:     e.Description,
		active:          e.Active,
		startsAt:        e.StartsAt,
		expiresAt:       e.ExpiresAt,
		maxUses:         e.MaxUses,
		redemptionCount: e.RedemptionCount,
		rewards:         e.Rewards,
		createdAt:       e.CreatedAt,
		updatedAt:       e.UpdatedAt,
	}, nil
}
```

- [ ] **Step 4: Write the model, builder, and error type**

Create `services/atlas-cashshop/atlas.com/cashshop/coupon/model.go`:

```go
package coupon

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// The seven client-facing outcome keys. Each is a key in the CashShopOperation
// writer's `errors` table (see docs/packets/dispatchers/cash_shop_operation.yaml),
// EXCEPT ErrorKeyUnknown, which is the jump table's default case on every
// version and is therefore deliberately NOT configured — see Task 23.
const (
	ErrorKeyInvalidCode   = "INVALID_COUPON_CODE"
	ErrorKeyNotRegistered = "COUPON_NOT_REGISTERED"
	ErrorKeyExpired       = "COUPON_EXPIRED"
	ErrorKeyAlreadyUsed   = "COUPON_ALREADY_USED"
	ErrorKeyUsageLimit    = "COUPON_USAGE_LIMIT"
	ErrorKeyInventoryFull = "INVENTORY_FULL"
	ErrorKeyUnknown       = "UNKNOWN_ERROR"
)

// RedemptionError carries the client-facing key, so the mapping from outcome
// to wire key lives once — at the ladder — and no transport layer re-derives it.
type RedemptionError struct {
	key    string
	detail string
}

func NewRedemptionError(key string, detail string) *RedemptionError {
	return &RedemptionError{key: key, detail: detail}
}

func (e *RedemptionError) Key() string { return e.key }

func (e *RedemptionError) Error() string {
	return fmt.Sprintf("coupon redemption rejected [%s]: %s", e.key, e.detail)
}

type Model struct {
	id              uuid.UUID
	batchId         uuid.UUID
	code            string
	description     string
	active          bool
	startsAt        *time.Time
	expiresAt       *time.Time
	maxUses         *uint32
	redemptionCount uint32
	rewards         Rewards
	createdAt       time.Time
	updatedAt       time.Time
}

func (m Model) Id() uuid.UUID           { return m.id }
func (m Model) BatchId() uuid.UUID      { return m.batchId }
func (m Model) Code() string            { return m.code }
func (m Model) Description() string     { return m.description }
func (m Model) Active() bool            { return m.active }
func (m Model) StartsAt() *time.Time    { return m.startsAt }
func (m Model) ExpiresAt() *time.Time   { return m.expiresAt }
func (m Model) MaxUses() *uint32        { return m.maxUses }
func (m Model) RedemptionCount() uint32 { return m.redemptionCount }
func (m Model) Rewards() Rewards        { return m.rewards }
func (m Model) CreatedAt() time.Time    { return m.createdAt }
func (m Model) UpdatedAt() time.Time    { return m.updatedAt }

// RedeemableAt runs steps 2-4 and 6 of the FR-5.4 validation ladder — the
// checks answerable from this row alone. FIRST FAILURE WINS, in this exact
// order, so the error a player sees is deterministic. Steps 1 (existence),
// 5 (per-account prior redemption) and 7 (locker capacity) need a query and
// live in the processor.
//
// The usage-limit check here is a FRIENDLY-ERROR FAST PATH only. The real
// enforcement is the conditional UPDATE in administrator.go — a read-then-write
// on redemptionCount is a race and is banned.
func (m Model) RedeemableAt(now time.Time) error {
	if !m.active {
		return NewRedemptionError(ErrorKeyNotRegistered, "coupon is inactive")
	}
	if m.startsAt != nil && now.Before(*m.startsAt) {
		return NewRedemptionError(ErrorKeyNotRegistered, "coupon has not started")
	}
	if m.expiresAt != nil && now.After(*m.expiresAt) {
		return NewRedemptionError(ErrorKeyExpired, "coupon has expired")
	}
	if m.maxUses != nil && m.redemptionCount >= *m.maxUses {
		return NewRedemptionError(ErrorKeyUsageLimit, "coupon has no uses remaining")
	}
	return nil
}

type Builder struct {
	id              uuid.UUID
	batchId         uuid.UUID
	code            string
	description     string
	active          bool
	startsAt        *time.Time
	expiresAt       *time.Time
	maxUses         *uint32
	redemptionCount uint32
	rewards         Rewards
}

// NewBuilder normalizes the code immediately, so a Model can never hold an
// un-normalized code and no caller has to remember to normalize.
func NewBuilder(code string) *Builder {
	return &Builder{code: Normalize(code), active: true}
}

func (b *Builder) SetId(id uuid.UUID) *Builder                { b.id = id; return b }
func (b *Builder) SetBatchId(id uuid.UUID) *Builder           { b.batchId = id; return b }
func (b *Builder) SetDescription(d string) *Builder           { b.description = d; return b }
func (b *Builder) SetActive(a bool) *Builder                  { b.active = a; return b }
func (b *Builder) SetStartsAt(t *time.Time) *Builder          { b.startsAt = t; return b }
func (b *Builder) SetExpiresAt(t *time.Time) *Builder         { b.expiresAt = t; return b }
func (b *Builder) SetMaxUses(n *uint32) *Builder              { b.maxUses = n; return b }
func (b *Builder) SetRedemptionCount(n uint32) *Builder       { b.redemptionCount = n; return b }
func (b *Builder) SetRewards(r Rewards) *Builder              { b.rewards = r; return b }

var ErrInvalidCoupon = errors.New("invalid coupon")

func (b *Builder) Build() (Model, error) {
	if !Plausible(b.code) {
		return Model{}, fmt.Errorf("%w: code must be 1-%d characters after normalization", ErrInvalidCoupon, MaxCodeLength)
	}
	if err := b.rewards.Validate(); err != nil {
		return Model{}, fmt.Errorf("%w: %s", ErrInvalidCoupon, err)
	}
	if b.startsAt != nil && b.expiresAt != nil && !b.expiresAt.After(*b.startsAt) {
		return Model{}, fmt.Errorf("%w: expiresAt must be after startsAt", ErrInvalidCoupon)
	}
	if b.maxUses != nil && *b.maxUses < b.redemptionCount {
		return Model{}, fmt.Errorf("%w: maxUses (%d) is below the current redemption count (%d)", ErrInvalidCoupon, *b.maxUses, b.redemptionCount)
	}
	return Model{
		id:              b.id,
		batchId:         b.batchId,
		code:            b.code,
		description:     b.description,
		active:          b.active,
		startsAt:        b.startsAt,
		expiresAt:       b.expiresAt,
		maxUses:         b.maxUses,
		redemptionCount: b.redemptionCount,
		rewards:         b.rewards,
	}, nil
}
```

- [ ] **Step 5: Register the migration**

In `services/atlas-cashshop/atlas.com/cashshop/main.go:57`, add `coupon.Migration` to the list:

```go
	db := database.Connect(l, database.SetMigrations(wallet.Migration, wishlist.Migration, compartment.Migration, asset.Migration, coupon.Migration, outboxlib.Migration))
```

Add the `"atlas-cashshop/coupon"` import. Task 15 extends this same line with the batch and redemption migrations.

- [ ] **Step 6: Run the tests and build**

Run: `cd services/atlas-cashshop && go test -race ./atlas.com/cashshop/coupon/ -v && go build ./...`
Expected: PASS, clean build.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-cashshop/atlas.com/cashshop/coupon services/atlas-cashshop/atlas.com/cashshop/main.go
git commit -m "feat(coupon): coupon entity, immutable model, and validation ladder"
```

---

### Task 15: Batch and redemption entities

`coupon_redemptions` carries the database-level one-time-per-account rule — the unique index on `(tenant_id, coupon_id, account_id)`. The FR-5.4 ladder check is a friendly-error fast path; **this index is the enforcement**, and it is what resolves the same-account race (design §2.2).

**Files:**
- Create: `services/atlas-cashshop/atlas.com/cashshop/coupon/batch/entity.go`, `model.go`
- Create: `services/atlas-cashshop/atlas.com/cashshop/coupon/redemption/entity.go`, `model.go`
- Test: `services/atlas-cashshop/atlas.com/cashshop/coupon/redemption/model_test.go`
- Modify: `services/atlas-cashshop/atlas.com/cashshop/main.go:57`

**Interfaces:**
- Produces: `batch.Entity` (table `coupon_batches`), `batch.Migration`, `batch.Model` (`Id()`, `Description()`, `RequestedCount() uint32`, `GeneratedCount() uint32`, `CreatedAt()`), `batch.Make`; `redemption.Entity` (table `coupon_redemptions`), `redemption.Migration`, `redemption.Model` (`Id()`, `CouponId() uuid.UUID`, `AccountId() uint32`, `CharacterId() uint32`, `TransactionId() uuid.UUID`, `RewardsGranted() coupon.Rewards`, `RedeemedAt() time.Time`), `redemption.Make`, and `redemption.IsUniqueViolation(err error) bool`. Tasks 19 and 22 consume these.

- [ ] **Step 1: Write the failing unique-violation test**

Create `services/atlas-cashshop/atlas.com/cashshop/coupon/redemption/model_test.go`:

```go
package redemption

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestIsUniqueViolation(t *testing.T) {
	// 23505 is Postgres' unique_violation. The redemption insert relies on it
	// to resolve the same-account race into exactly one COUPON_ALREADY_USED,
	// so misclassifying it would turn a lost race into UNKNOWN_ERROR.
	for _, c := range []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"unrelated", errors.New("boom"), false},
		{"unique violation", &pgconn.PgError{Code: "23505"}, true},
		{"wrapped unique violation", errors.Join(errors.New("insert failed"), &pgconn.PgError{Code: "23505"}), true},
		{"foreign key violation", &pgconn.PgError{Code: "23503"}, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := IsUniqueViolation(c.err); got != c.want {
				t.Errorf("IsUniqueViolation(%v) = %v, want %v", c.err, got, c.want)
			}
		})
	}
}
```

Confirm the pgx driver import path against `services/atlas-cashshop/atlas.com/cashshop/go.mod` before writing it; if the service uses a different Postgres driver, use that driver's error type and code accessor instead — the assertion is on SQLSTATE `23505`, not on a particular package.

- [ ] **Step 2: Run and watch it fail**

Run: `cd services/atlas-cashshop && go test ./atlas.com/cashshop/coupon/redemption/ -v`
Expected: FAIL — undefined `IsUniqueViolation`.

- [ ] **Step 3: Write the redemption entity and model**

Create `services/atlas-cashshop/atlas.com/cashshop/coupon/redemption/entity.go`:

```go
package redemption

import (
	"atlas-cashshop/coupon"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}

// Entity is the coupon_redemptions table — one row per SUCCESSFUL redemption.
//
// The uniqueIndex on (tenant_id, coupon_id, account_id) is the DATABASE-LEVEL
// one-time-per-account rule, not a convenience: it is what makes two concurrent
// redemptions by the same account resolve to exactly one success and one
// COUPON_ALREADY_USED. The ladder check in coupon.Model is only a friendly-error
// fast path.
//
// RewardsGranted is a SNAPSHOT, not a reference, so later edits to the coupon's
// bundle never rewrite history.
type Entity struct {
	Id             uuid.UUID      `gorm:"primaryKey;type:uuid"`
	TenantId       uuid.UUID      `gorm:"not null;uniqueIndex:idx_redemptions_tenant_coupon_account,priority:1;index:idx_redemptions_tenant_account,priority:1"`
	CouponId       uuid.UUID      `gorm:"not null;type:uuid;index;uniqueIndex:idx_redemptions_tenant_coupon_account,priority:2"`
	AccountId      uint32         `gorm:"not null;uniqueIndex:idx_redemptions_tenant_coupon_account,priority:3;index:idx_redemptions_tenant_account,priority:2"`
	CharacterId    uint32         `gorm:"not null"`
	TransactionId  uuid.UUID      `gorm:"not null;type:uuid"`
	RewardsGranted coupon.Rewards `gorm:"not null;type:jsonb"`
	RedeemedAt     time.Time      `gorm:"not null"`
}

func (e Entity) TableName() string {
	return "coupon_redemptions"
}

func (e *Entity) BeforeCreate(_ *gorm.DB) (err error) {
	if e.Id == uuid.Nil {
		e.Id = uuid.New()
	}
	return
}

// IsUniqueViolation reports whether err is a Postgres unique_violation (23505).
// The redemption insert treats that specific code as "this account already
// redeemed this coupon" — the race loser — and every other error as internal.
func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

func Make(e Entity) (Model, error) {
	return Model{
		id:             e.Id,
		couponId:       e.CouponId,
		accountId:      e.AccountId,
		characterId:    e.CharacterId,
		transactionId:  e.TransactionId,
		rewardsGranted: e.RewardsGranted,
		redeemedAt:     e.RedeemedAt,
	}, nil
}
```

Create `services/atlas-cashshop/atlas.com/cashshop/coupon/redemption/model.go` with the immutable `Model`, its getters, and a `NewBuilder(couponId uuid.UUID, accountId uint32, characterId uint32)` exposing `SetTransactionId`, `SetRewardsGranted`, `SetRedeemedAt`, `Build() (Model, error)`. `Build` rejects a zero `couponId`, a zero `accountId`, or an empty `rewardsGranted`.

- [ ] **Step 4: Write the batch entity and model**

Create `services/atlas-cashshop/atlas.com/cashshop/coupon/batch/entity.go`:

```go
package batch

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}

// Entity is the coupon_batches table — the grouping created by one bulk
// generation. GeneratedCount always equals RequestedCount on success: the
// generator RETRIES a code collision rather than skipping it (design §8), so a
// short batch is a bug, not an expected outcome.
type Entity struct {
	Id             uuid.UUID `gorm:"primaryKey;type:uuid"`
	TenantId       uuid.UUID `gorm:"not null;index"`
	Description    string    `gorm:"type:text"`
	RequestedCount uint32    `gorm:"not null"`
	GeneratedCount uint32    `gorm:"not null"`
	CreatedAt      time.Time
}

func (e Entity) TableName() string {
	return "coupon_batches"
}

func (e *Entity) BeforeCreate(_ *gorm.DB) (err error) {
	if e.Id == uuid.Nil {
		e.Id = uuid.New()
	}
	return
}

func Make(e Entity) (Model, error) {
	return Model{
		id:             e.Id,
		description:    e.Description,
		requestedCount: e.RequestedCount,
		generatedCount: e.GeneratedCount,
		createdAt:      e.CreatedAt,
	}, nil
}
```

Create `services/atlas-cashshop/atlas.com/cashshop/coupon/batch/model.go` with the immutable `Model` + getters + `NewBuilder`.

- [ ] **Step 5: Register both migrations**

Extend `services/atlas-cashshop/atlas.com/cashshop/main.go:57`:

```go
	db := database.Connect(l, database.SetMigrations(wallet.Migration, wishlist.Migration, compartment.Migration, asset.Migration, coupon.Migration, batch.Migration, redemption.Migration, outboxlib.Migration))
```

Order matters only in that `coupon.Migration` precedes the other two, which reference `coupons.id` conceptually (there is no FK constraint — the project's other tables do not declare them either).

- [ ] **Step 6: Verify the migration on a real database**

`AutoMigrate` silently skips an index it cannot build, so assert the indexes exist rather than assuming:

```bash
cd services/atlas-cashshop && go test -race ./atlas.com/cashshop/coupon/... && go build ./...
```

Then, against a scratch Postgres (the same one the concurrency tests in Task 20 use), run the service once and check:

```sql
SELECT indexname, indexdef FROM pg_indexes
WHERE tablename IN ('coupons','coupon_batches','coupon_redemptions')
ORDER BY tablename, indexname;
```
Expected to include a UNIQUE index on `coupons (tenant_id, code)` and a UNIQUE index on `coupon_redemptions (tenant_id, coupon_id, account_id)`. **A non-unique index here silently disables the one-time-per-account rule** — if either is missing or not unique, fix the tag before continuing; Task 20's race test depends on both.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-cashshop/atlas.com/cashshop/coupon services/atlas-cashshop/atlas.com/cashshop/main.go
git commit -m "feat(coupon): batch and redemption entities with the one-time-per-account unique index"
```

---

### Task 16: Kafka contracts for coupon redemption

The request/response shape copies the existing purchase flow: channel → `COMMAND_TOPIC_CASH_SHOP` → `atlas-cashshop` → `EVENT_TOPIC_CASH_SHOP_STATUS` → the channel's status consumer.

The contracts are **duplicated** in both services (that is the existing pattern — `services/atlas-cashshop/.../kafka/message/cashshop/kafka.go` and `services/atlas-channel/.../kafka/message/cashshop/`). Keep the JSON tags byte-identical; a tag typo is a silent no-op at runtime.

**Files:**
- Modify: `services/atlas-cashshop/atlas.com/cashshop/kafka/message/cashshop/kafka.go`
- Modify: `services/atlas-channel/atlas.com/channel/kafka/message/cashshop/kafka.go`
- Modify: `docs/tasks/task-206-cash-shop-coupon-codes/design.md:157-161`
- Test: `services/atlas-cashshop/atlas.com/cashshop/kafka/message/cashshop/kafka_test.go`

**Interfaces:**
- Produces, in both packages: `CommandTypeRequestCouponRedemption = "REQUEST_COUPON_REDEMPTION"`, `RequestCouponRedemptionCommandBody{Code string}`, `StatusEventTypeCouponRedeemed = "COUPON_REDEEMED"`, `CouponRedeemedBody{CompartmentId uuid.UUID, AssetIds []uint32, MaplePoints uint32, Credit uint32}`, `StatusEventTypeCouponFailed = "COUPON_FAILED"`, `CouponFailedBody{Error string}`. Tasks 19, 21, 23 and 24 consume them.

- [ ] **Step 1: Correct the design's maplePoint claim first**

`design.md:160` reads `MaplePoints   uint32    \`json:"maplePoints"\`   // absolute post-award balance`. That is **wrong**: `derivation.md` "Blocking answer 1" proves the field is the amount awarded by this coupon — the client skips it entirely when zero (`if (v68)`) and renders it inside "You have received … using the coupon" alongside `mesos`. Edit `design.md` so the comment reads:

```go
    MaplePoints   uint32    `json:"maplePoints"`   // DELTA — the Maple Points this coupon awarded, NOT a balance
```

and add a line under the block:

> **Correction (task-206 derivation).** PRD Q5 is answered: `UseCouponDone.maplePoint` is a delta, not an absolute post-award balance. `CCashShop::OnCashItemResUseCouponDone` @ `0x479d8a` reads it at `0x479efb` and uses it only at `0x47a0b6`-`0x47a0df` to format `SP_587_D_MAPLEPOINTS` inside the `SP_585_YOU_HAVE_RECEIVED` sentence; the balance itself is refreshed separately by `CCashShop::OnQueryCashResult` @ `0x478f81`.

- [ ] **Step 2: Write the failing contract test**

Create `services/atlas-cashshop/atlas.com/cashshop/kafka/message/cashshop/kafka_test.go`:

```go
package cashshop

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

// The channel decodes these bodies from a duplicated struct definition, so a
// JSON tag change on one side is a silent field drop on the other. Pin the
// wire shape.
func TestCouponCommandWireShape(t *testing.T) {
	b, err := json.Marshal(Command[RequestCouponRedemptionCommandBody]{
		CharacterId: 7,
		Type:        CommandTypeRequestCouponRedemption,
		Body:        RequestCouponRedemptionCommandBody{Code: "MAPLE2026"},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"characterId":7,"type":"REQUEST_COUPON_REDEMPTION","body":{"code":"MAPLE2026"}}`
	if string(b) != want {
		t.Errorf("got  %s\nwant %s", b, want)
	}
}

func TestCouponRedeemedWireShape(t *testing.T) {
	id := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	b, err := json.Marshal(StatusEvent[CouponRedeemedBody]{
		CharacterId: 7,
		Type:        StatusEventTypeCouponRedeemed,
		Body:        CouponRedeemedBody{CompartmentId: id, AssetIds: []uint32{9}, MaplePoints: 500, Credit: 250},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"characterId":7,"type":"COUPON_REDEEMED","body":{"compartmentId":"11111111-2222-3333-4444-555555555555","assetIds":[9],"maplePoints":500,"credit":250}}`
	if string(b) != want {
		t.Errorf("got  %s\nwant %s", b, want)
	}
}

func TestCouponFailedWireShape(t *testing.T) {
	b, err := json.Marshal(StatusEvent[CouponFailedBody]{
		CharacterId: 7,
		Type:        StatusEventTypeCouponFailed,
		Body:        CouponFailedBody{Error: "COUPON_EXPIRED"},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"characterId":7,"type":"COUPON_FAILED","body":{"error":"COUPON_EXPIRED"}}`
	if string(b) != want {
		t.Errorf("got  %s\nwant %s", b, want)
	}
}
```

- [ ] **Step 3: Run and watch it fail**

Run: `cd services/atlas-cashshop && go test ./atlas.com/cashshop/kafka/message/cashshop/ -v`
Expected: FAIL — undefined types.

- [ ] **Step 4: Add the contracts**

In `services/atlas-cashshop/atlas.com/cashshop/kafka/message/cashshop/kafka.go`, add `CommandTypeRequestCouponRedemption = "REQUEST_COUPON_REDEMPTION"` to the command const block and:

```go
// RequestCouponRedemptionCommandBody carries only the code: the channel has
// already normalized it (trimmed + uppercased), and the owning ACCOUNT is
// resolved service-side from Command.CharacterId, because the packet arrives
// on a character session while wallets are account-scoped.
//
// The v83..v95 clients also send a leading target-character string, but
// targeted redemption (gift coupons) is out of scope (PRD §2) and the field is
// deliberately not carried here.
type RequestCouponRedemptionCommandBody struct {
	Code string `json:"code"`
}
```

Add to the status const block:

```go
	StatusEventTypeCouponRedeemed = "COUPON_REDEEMED"
	StatusEventTypeCouponFailed   = "COUPON_FAILED"
```

and the bodies:

```go
// CouponRedeemedBody describes one successful redemption.
//
// MaplePoints and Credit are DELTAS — the amounts this coupon awarded — not
// balances. UseCouponDone.maplePoint is rendered by the client inside a
// "You have received ... using the coupon" sentence and is skipped entirely
// when zero; the balance is refreshed separately by CashQueryResult. See
// docs/tasks/task-206-cash-shop-coupon-codes/derivation.md, "Blocking answer 1".
//
// AssetIds rather than fully-built CashInventoryItem records: the channel
// already owns the asset-id -> CashInventoryItem projection (its purchase
// handler at kafka/consumer/cashshop/consumer.go:105-124), and duplicating it
// here would put packet concerns in atlas-cashshop.
type CouponRedeemedBody struct {
	CompartmentId uuid.UUID `json:"compartmentId"`
	AssetIds      []uint32  `json:"assetIds"`
	MaplePoints   uint32    `json:"maplePoints"`
	Credit        uint32    `json:"credit"`
}

// CouponFailedBody carries one of the coupon.ErrorKey* strings.
//
// This is a DISTINCT event type rather than a reuse of StatusEventTypeError,
// because the existing ERROR handler announces
// CashShopInventoryCapacityIncreaseFailedBody — a different mode byte. A
// coupon failure must go out on the USE_COUPON_FAILED arm, so folding it into
// ERROR would force the channel to guess which failure arm an error belongs to.
type CouponFailedBody struct {
	Error string `json:"error"`
}
```

- [ ] **Step 5: Mirror into the channel**

Add the identical const values and struct definitions to `services/atlas-channel/atlas.com/channel/kafka/message/cashshop/kafka.go`. Copy the doc comments too — the reasoning is what stops the next person from "simplifying" `COUPON_FAILED` into `ERROR`.

Verify the two definitions agree:
```bash
diff <(grep -A 8 'type CouponRedeemedBody' services/atlas-cashshop/atlas.com/cashshop/kafka/message/cashshop/kafka.go) \
     <(grep -A 8 'type CouponRedeemedBody' services/atlas-channel/atlas.com/channel/kafka/message/cashshop/kafka.go)
```
Expected: no differences.

- [ ] **Step 6: Test and build both modules**

Run:
```bash
cd services/atlas-cashshop && go test -race ./atlas.com/cashshop/kafka/... && go build ./... && cd -
cd services/atlas-channel && go build ./... && cd -
```
Expected: PASS, clean.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-cashshop/atlas.com/cashshop/kafka services/atlas-channel/atlas.com/channel/kafka/message/cashshop docs/tasks/task-206-cash-shop-coupon-codes/design.md
git commit -m "feat(cashshop): coupon redemption command and status-event contracts"
```

---

### Task 17: Redis counter increment and the failed-attempt rate limiter

Brute-forcing short codes is the obvious attack (PRD §8). Failed attempts are counted per account; past the tenant-configured threshold within the window, further attempts short-circuit to `INVALID_COUPON_CODE` **with no database lookup**.

Returning `INVALID_COUPON_CODE` rather than a distinct "rate limited" key is deliberate: a distinct key would leak that the attacker had found a *real* code and merely run out of attempts.

`libs/atlas-redis`'s `TenantCounter` has `Set`, `DecrByIfExists`, `InitIfMissingAndDecrBy` and `Remove` — no increment. `tools/redis-key-guard.sh` bans keyed Redis commands on the raw `go-redis` client outside `libs/atlas-redis`, so the increment must be added to the library, not written at the call site.

**Files:**
- Modify: `libs/atlas-redis/counter.go`
- Test: `libs/atlas-redis/counter_test.go`
- Create: `services/atlas-cashshop/atlas.com/cashshop/coupon/limiter.go`
- Test: `services/atlas-cashshop/atlas.com/cashshop/coupon/limiter_test.go`
- Modify: `services/atlas-cashshop/atlas.com/cashshop/main.go`

**Interfaces:**
- Produces: `func (c *TenantCounter) IncrWithTTL(ctx, t tenant.Model, key string, ttl time.Duration) (int64, error)`; `coupon.InitLimiter(client *goredis.Client)`, `coupon.Limiter` with `Allowed(ctx, t, accountId) (bool, error)`, `RecordFailure(ctx, t, accountId) error`, `Reset(ctx, t, accountId) error`, and `NewLimiter(attempts uint32, window time.Duration) Limiter`. Task 19 consumes the limiter.

- [ ] **Step 1: Write the failing library test**

Append to `libs/atlas-redis/counter_test.go`, following the harness the existing counter tests use (miniredis or a real client — read the file and match it):

```go
func TestTenantCounterIncrWithTTL(t *testing.T) {
	client, tm := newCounterTestClient(t) // existing helper
	c := NewTenantCounter(client, "test-incr")
	ctx := context.Background()

	// The FIRST increment must create the key AND set the TTL. A plain INCR
	// leaves a fresh key with no expiry, which makes a rate-limit window
	// permanent and locks the account out forever.
	v, err := c.IncrWithTTL(ctx, tm, "acct-1", time.Minute)
	if err != nil {
		t.Fatalf("first incr: %v", err)
	}
	if v != 1 {
		t.Errorf("first incr = %d, want 1", v)
	}
	ttl, err := client.TTL(ctx, c.entityKey(tm, "acct-1")).Result()
	if err != nil {
		t.Fatal(err)
	}
	if ttl <= 0 {
		t.Fatalf("TTL = %v, want a positive expiry on the first increment", ttl)
	}

	v, err = c.IncrWithTTL(ctx, tm, "acct-1", time.Minute)
	if err != nil {
		t.Fatalf("second incr: %v", err)
	}
	if v != 2 {
		t.Errorf("second incr = %d, want 2", v)
	}
}

func TestTenantCounterIncrWithTTLDoesNotSlideTheWindow(t *testing.T) {
	// A sliding window lets an attacker keep the key alive forever by
	// attempting just often enough. The TTL is set ONCE, on creation.
	client, tm := newCounterTestClient(t)
	c := NewTenantCounter(client, "test-incr-slide")
	ctx := context.Background()

	if _, err := c.IncrWithTTL(ctx, tm, "acct-2", 10*time.Second); err != nil {
		t.Fatal(err)
	}
	first, _ := client.PTTL(ctx, c.entityKey(tm, "acct-2")).Result()
	if _, err := c.IncrWithTTL(ctx, tm, "acct-2", 10*time.Hour); err != nil {
		t.Fatal(err)
	}
	second, _ := client.PTTL(ctx, c.entityKey(tm, "acct-2")).Result()
	if second > first {
		t.Errorf("TTL grew from %v to %v — the window slid", first, second)
	}
}

func TestTenantCounterIsTenantScoped(t *testing.T) {
	client, tm := newCounterTestClient(t)
	other := otherTenant(t) // a second tenant.Model
	c := NewTenantCounter(client, "test-incr-tenant")
	ctx := context.Background()

	if _, err := c.IncrWithTTL(ctx, tm, "acct-3", time.Minute); err != nil {
		t.Fatal(err)
	}
	v, err := c.IncrWithTTL(ctx, other, "acct-3", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if v != 1 {
		t.Errorf("other tenant's counter = %d, want 1 — counters must not share a key", v)
	}
}
```

- [ ] **Step 2: Run and watch it fail**

Run: `cd libs/atlas-redis && go test ./... -run TestTenantCounterIncr -v`
Expected: FAIL — `c.IncrWithTTL undefined`.

- [ ] **Step 3: Add the library method**

Append to `libs/atlas-redis/counter.go`:

```go
// incrWithTTLScript increments a counter and sets its TTL only when the key is
// being created. A plain INCR leaves a new key with no expiry, which makes a
// rate-limit window permanent; refreshing the TTL on EVERY increment instead
// turns a fixed window into a sliding one an attacker can keep alive forever.
// Setting it exactly once, at creation, is the fixed-window semantics callers
// expect.
var incrWithTTLScript = goredis.NewScript(`
local v = redis.call("incr", KEYS[1])
if v == 1 then
	redis.call("pexpire", KEYS[1], ARGV[1])
end
return v`)

// IncrWithTTL increments the counter by one and returns the new value. The TTL
// is applied only when the counter is created, giving a fixed window that
// starts at the first increment.
func (c *TenantCounter) IncrWithTTL(ctx context.Context, t tenant.Model, key string, ttl time.Duration) (int64, error) {
	v, err := incrWithTTLScript.Run(ctx, c.client, []string{c.entityKey(t, key)}, ttl.Milliseconds()).Int64()
	if err != nil {
		return 0, fmt.Errorf("redis incr-with-ttl: %w", err)
	}
	return v, nil
}
```

- [ ] **Step 4: Run the library tests**

Run: `cd libs/atlas-redis && go test -race ./... && go vet ./...`
Expected: clean.

- [ ] **Step 5: Write the failing limiter test**

Create `services/atlas-cashshop/atlas.com/cashshop/coupon/limiter_test.go`:

```go
package coupon

import (
	"context"
	"testing"
	"time"
)

func TestLimiterAllowsUntilTheThreshold(t *testing.T) {
	ctx, tm := limiterTestContext(t) // spins a test redis + tenant, calls InitLimiter
	l := NewLimiter(3, time.Minute)

	for i := 0; i < 3; i++ {
		ok, err := l.Allowed(ctx, tm, 42)
		if err != nil {
			t.Fatalf("attempt %d: %v", i, err)
		}
		if !ok {
			t.Fatalf("attempt %d blocked, want allowed", i)
		}
		if err := l.RecordFailure(ctx, tm, 42); err != nil {
			t.Fatal(err)
		}
	}
	ok, err := l.Allowed(ctx, tm, 42)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("attempt 4 allowed, want blocked after 3 failures")
	}
}

func TestLimiterOnlyCountsFailures(t *testing.T) {
	// A successful redemption must not consume the account's budget.
	ctx, tm := limiterTestContext(t)
	l := NewLimiter(2, time.Minute)
	for i := 0; i < 5; i++ {
		if ok, _ := l.Allowed(ctx, tm, 43); !ok {
			t.Fatalf("attempt %d blocked without any recorded failure", i)
		}
	}
}

func TestLimiterResetClearsTheCounter() {}

func TestLimiterIsPerAccount(t *testing.T) {
	ctx, tm := limiterTestContext(t)
	l := NewLimiter(1, time.Minute)
	if err := l.RecordFailure(ctx, tm, 44); err != nil {
		t.Fatal(err)
	}
	if ok, _ := l.Allowed(ctx, tm, 45); !ok {
		t.Error("account 45 blocked by account 44's failure")
	}
}

func TestLimiterFailsOpenWhenRedisIsDown(t *testing.T) {
	// Redis being unreachable must not make every coupon un-redeemable. The
	// limiter is a brute-force brake, not an authorization gate: degrade to
	// allowing the attempt and let the ladder decide.
	ctx, tm := limiterBrokenRedisContext(t)
	l := NewLimiter(1, time.Minute)
	ok, err := l.Allowed(ctx, tm, 46)
	if !ok {
		t.Errorf("Allowed = false with redis down, want fail-open (err was %v)", err)
	}
}
```

Replace the empty `TestLimiterResetClearsTheCounter` stub with a real body: record failures up to the threshold, call `Reset`, assert `Allowed` is true again. Write `limiterTestContext` and `limiterBrokenRedisContext` helpers in the test file using whatever redis test harness `libs/atlas-redis/counter_test.go` already uses (`limiterBrokenRedisContext` points the client at a closed port).

- [ ] **Step 6: Implement the limiter**

Create `services/atlas-cashshop/atlas.com/cashshop/coupon/limiter.go`:

```go
package coupon

import (
	"context"
	"strconv"
	"sync"
	"time"

	goredis "github.com/redis/go-redis/v9"

	redis "github.com/Chronicle20/atlas/libs/atlas-redis"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const limiterNamespace = "coupon-attempts"

var (
	limiterOnce  sync.Once
	limiterStore *redis.TenantCounter
)

// InitLimiter wires the shared Redis client. Called once from main.go, like
// the other registry initializers in this project.
func InitLimiter(client *goredis.Client) {
	limiterOnce.Do(func() {
		limiterStore = redis.NewTenantCounter(client, limiterNamespace)
	})
}

// Limiter brakes coupon brute-forcing by counting an account's FAILED attempts
// inside a fixed window.
//
// It FAILS OPEN: when Redis is unreachable, Allowed returns true. This is a
// brute-force brake, not an authorization gate — a Redis outage must not make
// every coupon un-redeemable for every player.
type Limiter struct {
	attempts uint32
	window   time.Duration
}

func NewLimiter(attempts uint32, window time.Duration) Limiter {
	return Limiter{attempts: attempts, window: window}
}

func limiterKey(accountId uint32) string {
	return strconv.FormatUint(uint64(accountId), 10)
}

// Allowed reports whether this account may make another attempt. It never
// increments; only RecordFailure does.
func (l Limiter) Allowed(ctx context.Context, t tenant.Model, accountId uint32) (bool, error) {
	if limiterStore == nil {
		return true, nil
	}
	// Increment by zero is not available, so read the current value by
	// incrementing and immediately compensating would be a race. Instead the
	// counter is only ever incremented on failure, and Allowed compares the
	// stored value: use a zero-delta read via the counter's own accessor.
	n, err := limiterStore.Get(ctx, t, limiterKey(accountId))
	if err != nil {
		return true, err
	}
	return n < int64(l.attempts), nil
}

// RecordFailure counts one failed attempt against the account's window.
func (l Limiter) RecordFailure(ctx context.Context, t tenant.Model, accountId uint32) error {
	if limiterStore == nil {
		return nil
	}
	_, err := limiterStore.IncrWithTTL(ctx, t, limiterKey(accountId), l.window)
	return err
}

// Reset clears the account's counter after a successful redemption, so a
// player who mistyped a few times before getting it right is not left one
// failure away from a block.
func (l Limiter) Reset(ctx context.Context, t tenant.Model, accountId uint32) error {
	if limiterStore == nil {
		return nil
	}
	return limiterStore.Remove(ctx, t, limiterKey(accountId))
}
```

`Allowed` needs a read accessor `TenantCounter` does not have. Add it to `libs/atlas-redis/counter.go` in Step 3 alongside `IncrWithTTL`, with its own test in Step 1:

```go
// Get returns the counter's current value, or 0 when the key is absent.
func (c *TenantCounter) Get(ctx context.Context, t tenant.Model, key string) (int64, error) {
	v, err := c.client.Get(ctx, c.entityKey(t, key)).Int64()
	if errors.Is(err, goredis.Nil) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("redis get counter: %w", err)
	}
	return v, nil
}
```

Add `"errors"` to that file's imports.

- [ ] **Step 7: Wire the client in `main.go`**

In `services/atlas-cashshop/atlas.com/cashshop/main.go`, after the database connection, add:

```go
	rc := redis.Connect(l)
	coupon.InitLimiter(rc)
```

`atlas-cashshop` already receives `REDIS_URL` from the shared `atlas-env` configMap, so no manifest change is needed. Add the `redis "github.com/Chronicle20/atlas/libs/atlas-redis"` import and confirm `libs/atlas-redis` is in the service's `go.mod` require block (the `replace` directive is already there at line 102 — add the `require` entry if `go mod tidy` does not).

- [ ] **Step 8: Run everything**

Run:
```bash
cd libs/atlas-redis && go test -race ./... && cd -
cd services/atlas-cashshop && go mod tidy && go test -race ./atlas.com/cashshop/coupon/ -v && go build ./... && cd -
tools/redis-key-guard.sh
```
Expected: all pass; the guard exits 0 (no raw keyed command outside the library).

- [ ] **Step 9: Commit**

```bash
git add libs/atlas-redis services/atlas-cashshop
git commit -m "feat(coupon): per-account failed-attempt rate limiter backed by a fixed-window redis counter"
```

---

### Task 18: Providers and the atomic reservation

The two race-critical writes live here. Neither may be a read-then-write.

**Files:**
- Create: `services/atlas-cashshop/atlas.com/cashshop/coupon/provider.go`
- Create: `services/atlas-cashshop/atlas.com/cashshop/coupon/administrator.go`
- Create: `services/atlas-cashshop/atlas.com/cashshop/coupon/redemption/provider.go`, `administrator.go`
- Create: `services/atlas-cashshop/atlas.com/cashshop/coupon/batch/provider.go`, `administrator.go`
- Test: `services/atlas-cashshop/atlas.com/cashshop/coupon/administrator_test.go`

**Interfaces:**
- Produces, in package `coupon`: `byCodeEntityProvider(t tenant.Model, code string) database.EntityProvider[Entity]`, `byIdEntityProvider(t tenant.Model, id uuid.UUID) database.EntityProvider[Entity]`, `allEntityProvider(t tenant.Model, f Filters) database.EntityProvider[[]Entity]`, the `Filters` struct, `updateEntity`, `deleteEntity`, the sentinel `ErrHasRedemptions`, and critically `reserveUse(db *gorm.DB, t tenant.Model, id uuid.UUID) (bool, error)` / `releaseUse(db *gorm.DB, t tenant.Model, id uuid.UUID) error`.
- Produces, **exported** because package `coupon/batch` calls it across a package boundary: `coupon.CreateEntity(db *gorm.DB, t tenant.Model, m Model) (Model, error)`.
- Produces, in package `coupon/redemption`, **exported** because package `coupon` calls them: `redemption.Create(db *gorm.DB, t tenant.Model, m Model) (Model, error)` — returning the driver error unwrapped enough for `IsUniqueViolation` to classify it — and `redemption.CountByCouponAndAccount(db *gorm.DB, t tenant.Model, couponId uuid.UUID, accountId uint32) (int64, error)`, plus `byCouponIdProvider`, `byAccountIdProvider`, `countByCouponIdProvider` for the REST reads.
- Produces, in package `coupon/batch`: `createEntity`, `byIdEntityProvider`, `allEntityProvider`.
- Tasks 20 and 23 consume all of them.

- [ ] **Step 1: Write the failing reservation test**

Create `services/atlas-cashshop/atlas.com/cashshop/coupon/administrator_test.go`. This test needs a **real Postgres** — `RowsAffected` on a conditional `UPDATE` is precisely what an in-memory fake would get wrong. Use the same database harness the rest of `atlas-cashshop`'s DB-touching tests use; if there is none, add one that reads a `TEST_DB_DSN` env var and skips with `t.Skip` when it is unset, and say so in the commit body.

```go
func TestReserveUseRespectsMaxUses(t *testing.T) {
	db, tm := newCouponTestDB(t)
	id := seedCoupon(t, db, tm, NewBuilder("LIMITED").SetMaxUses(ptrU32(2)).SetRewards(Rewards{NewCurrencyReward(1, 1)}))

	for i := 1; i <= 2; i++ {
		ok, err := reserveUse(db, tm, id)
		if err != nil {
			t.Fatalf("reserve %d: %v", i, err)
		}
		if !ok {
			t.Fatalf("reserve %d = false, want true", i)
		}
	}
	ok, err := reserveUse(db, tm, id)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("third reserve succeeded, want refusal at maxUses")
	}
	if got := loadCount(t, db, id); got != 2 {
		t.Errorf("redemptionCount = %d, want 2", got)
	}
}

func TestReserveUseUnlimitedWhenMaxUsesIsNull(t *testing.T) {
	db, tm := newCouponTestDB(t)
	id := seedCoupon(t, db, tm, NewBuilder("UNLIMITED").SetRewards(Rewards{NewCurrencyReward(1, 1)}))
	for i := 0; i < 5; i++ {
		if ok, err := reserveUse(db, tm, id); err != nil || !ok {
			t.Fatalf("reserve %d: ok=%v err=%v", i, ok, err)
		}
	}
	if got := loadCount(t, db, id); got != 5 {
		t.Errorf("redemptionCount = %d, want 5", got)
	}
}

func TestReserveUseIsTenantScoped(t *testing.T) {
	db, tm := newCouponTestDB(t)
	other := otherTenantModel(t)
	id := seedCoupon(t, db, tm, NewBuilder("SCOPED").SetMaxUses(ptrU32(1)).SetRewards(Rewards{NewCurrencyReward(1, 1)}))
	ok, err := reserveUse(db, other, id)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("another tenant reserved this coupon")
	}
}

func TestReleaseUseDecrementsWithoutGoingNegative(t *testing.T) {
	db, tm := newCouponTestDB(t)
	id := seedCoupon(t, db, tm, NewBuilder("REL").SetRewards(Rewards{NewCurrencyReward(1, 1)}))
	if _, err := reserveUse(db, tm, id); err != nil {
		t.Fatal(err)
	}
	if err := releaseUse(db, tm, id); err != nil {
		t.Fatal(err)
	}
	if got := loadCount(t, db, id); got != 0 {
		t.Errorf("redemptionCount = %d, want 0", got)
	}
	// A stray release must not underflow the unsigned column.
	if err := releaseUse(db, tm, id); err != nil {
		t.Fatal(err)
	}
	if got := loadCount(t, db, id); got != 0 {
		t.Errorf("redemptionCount = %d after a stray release, want 0", got)
	}
}
```

- [ ] **Step 2: Run and watch it fail**

Run: `cd services/atlas-cashshop && go test ./atlas.com/cashshop/coupon/ -run TestReserveUse -v`
Expected: FAIL — undefined `reserveUse`.

- [ ] **Step 3: Write the administrator**

Create `services/atlas-cashshop/atlas.com/cashshop/coupon/administrator.go`. The reservation is the whole point of the file:

```go
package coupon

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// reserveUse claims one use of a coupon ATOMICALLY and reports whether the
// claim succeeded.
//
// The WHERE clause carries the max-uses predicate, so the check and the
// increment are one statement and RowsAffected is the verdict. A
// read-then-write here is a race: two concurrent redemptions of a
// max_uses = 1 coupon would both read 0, both write 1, and both succeed.
// This is FR-5.5 and it is explicitly banned in review.
func reserveUse(db *gorm.DB, t tenant.Model, id uuid.UUID) (bool, error) {
	res := db.Model(&Entity{}).
		Where("id = ? AND tenant_id = ? AND (max_uses IS NULL OR redemption_count < max_uses)", id, t.Id()).
		UpdateColumns(map[string]interface{}{
			"redemption_count": gorm.Expr("redemption_count + 1"),
			"updated_at":       time.Now(),
		})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}

// releaseUse gives a claimed use back. It is the compensation for a redemption
// that failed AFTER the reservation but is only reachable when the surrounding
// transaction did NOT roll back — inside one ExecuteTransaction the ROLLBACK
// already undoes the increment. It exists for the out-of-transaction paths and
// is guarded against underflow because redemption_count is unsigned.
func releaseUse(db *gorm.DB, t tenant.Model, id uuid.UUID) error {
	return db.Model(&Entity{}).
		Where("id = ? AND tenant_id = ? AND redemption_count > 0", id, t.Id()).
		UpdateColumns(map[string]interface{}{
			"redemption_count": gorm.Expr("redemption_count - 1"),
			"updated_at":       time.Now(),
		}).Error
}
```

Add `CreateEntity` (exported — `coupon/batch` calls it across the package boundary during bulk generation), `updateEntity` and `deleteEntity` in the same file, following `wallet/administrator.go`'s shape: build the `Entity` from the `Model`, always set `TenantId` from `t.Id()`, and scope every `Where` by `tenant_id`. `deleteEntity` must refuse when redemptions exist — return a sentinel `ErrHasRedemptions` that Task 23 maps to `409`.

`CreateEntity` must return the driver error **unwrapped enough for `redemption.IsUniqueViolation` to classify it**: bulk generation depends on recognizing a `(tenant_id, code)` collision as a retryable duplicate rather than an internal error. Use `%w` if you wrap at all.

- [ ] **Step 4: Write the providers**

Create `services/atlas-cashshop/atlas.com/cashshop/coupon/provider.go` following `wallet/provider.go`'s `database.EntityProvider[Entity]` shape. Every provider scopes on `tenant_id`:

```go
// byCodeEntityProvider looks a coupon up by its NORMALIZED code. Callers must
// pass Normalize(code); the column stores the normalized form and the unique
// index on (tenant_id, code) is what makes the lookup case-insensitive.
func byCodeEntityProvider(t tenant.Model, code string) database.EntityProvider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		var result Entity
		err := db.Where("tenant_id = ? AND code = ?", t.Id(), code).First(&result).Error
		if err != nil {
			return model.ErrorProvider[Entity](err)
		}
		return model.FixedProvider[Entity](result)
	}
}
```

plus `byIdEntityProvider`, and `allEntityProvider(t, f Filters)` where `Filters` carries the optional `Code *string`, `Active *bool`, `BatchId *uuid.UUID`, `ExpiresBefore *time.Time`, `ExpiresAfter *time.Time` from PRD §5 and applies each only when set.

Create the `redemption` and `batch` providers/administrators the same way. `redemption.createEntity` returns the raw driver error unchanged so `IsUniqueViolation` can classify it — do **not** wrap it in a way that hides the `*pgconn.PgError`; use `%w` if you wrap at all.

- [ ] **Step 5: Run the tests**

Run: `cd services/atlas-cashshop && go test -race ./atlas.com/cashshop/coupon/... -v`
Expected: PASS (or SKIP with a clear message if `TEST_DB_DSN` is unset — but run it at least once against a real database before moving on; the whole task is about `RowsAffected`).

- [ ] **Step 6: Commit**

```bash
git add services/atlas-cashshop/atlas.com/cashshop/coupon
git commit -m "feat(coupon): providers and the atomic max-uses reservation"
```

---

### Task 19: Reward granters

Reward application is dispatched by reward type behind one small interface (design §2.1). This is not abstraction for its own sake: it keeps the reward loop readable and gives the future out-of-service reward type one obvious insertion point — the place where a saga would be introduced — instead of a rewrite.

**Files:**
- Create: `services/atlas-cashshop/atlas.com/cashshop/coupon/granter.go`
- Test: `services/atlas-cashshop/atlas.com/cashshop/coupon/granter_test.go`

**Interfaces:**
- Consumes: `wallet.Model.Award` (Task 11), `Reward` (Task 13), `asset.Processor.Create` (`compartmentId, templateId, commodityId, quantity, petId, purchasedBy`), `compartment.Processor.GetByAccountIdAndType`.
- Produces: `type redemptionContext struct` (`accountId`, `characterId`, `compartmentId uuid.UUID`), `type grantedReward struct` (`assetId uint32`, `maplePoints uint32`, `credit uint32`), `type rewardGranter interface { Grant(mb *message.Buffer) func(tx *gorm.DB, rc redemptionContext, r Reward) (grantedReward, error) }`, and `func granterFor(l logrus.FieldLogger, ctx context.Context, r Reward) (rewardGranter, error)`. Task 20 consumes them.

- [ ] **Step 1: Write the failing tests**

Create `services/atlas-cashshop/atlas.com/cashshop/coupon/granter_test.go`:

```go
func TestGranterForDispatchesByType(t *testing.T) {
	l, ctx := testLoggerAndContext(t)
	if _, err := granterFor(l, ctx, NewCurrencyReward(1, 5)); err != nil {
		t.Errorf("currency: %v", err)
	}
	if _, err := granterFor(l, ctx, NewCashItemReward(50200000, 1)); err != nil {
		t.Errorf("cash item: %v", err)
	}
	// An unknown type must be a hard error, not a silent no-op: a coupon that
	// claims to grant something and grants nothing is worse than one that fails.
	if _, err := granterFor(l, ctx, Reward{rewardType: "MESO", amount: 1}); err == nil {
		t.Error("want an error for an unknown reward type")
	}
}

func TestCurrencyGranterCreditsTheWalletAndReportsTheDelta(t *testing.T) {
	db, tm := newCouponTestDB(t)
	seedWallet(t, db, tm, 1001, 100, 200, 300) // credit, points, prepaid
	l, gctx := testLoggerAndContext(t)
	g, _ := granterFor(l, gctx, NewCurrencyReward(2, 500))
	mb := message.NewBuffer()

	got, err := g.Grant(mb)(db, redemptionContext{accountId: 1001}, NewCurrencyReward(2, 500))
	if err != nil {
		t.Fatalf("Grant: %v", err)
	}
	// The event body carries the DELTA the coupon awarded, not the balance.
	if got.maplePoints != 500 {
		t.Errorf("maplePoints delta = %d, want 500", got.maplePoints)
	}
	if got.credit != 0 {
		t.Errorf("credit delta = %d, want 0", got.credit)
	}
	if p := loadWalletPoints(t, db, 1001); p != 700 {
		t.Errorf("stored points = %d, want 700", p)
	}
}

func TestCashItemGranterRechecksCapacityInsideTheTransaction(t *testing.T) {
	// Q6: the ladder's pre-flight capacity check gives a deterministic error
	// ordering; THIS re-check closes the TOCTOU window between the ladder and
	// the grant.
	db, tm := newCouponTestDB(t)
	cid := seedFullCompartment(t, db, tm, 1002) // capacity == len(assets)
	l, gctx := testLoggerAndContext(t)
	g, _ := granterFor(l, gctx, NewCashItemReward(50200000, 1))
	mb := message.NewBuffer()

	_, err := g.Grant(mb)(db, redemptionContext{accountId: 1002, compartmentId: cid}, NewCashItemReward(50200000, 1))
	var re *RedemptionError
	if !errors.As(err, &re) || re.Key() != ErrorKeyInventoryFull {
		t.Fatalf("err = %v, want a RedemptionError with key %s", err, ErrorKeyInventoryFull)
	}
}

func TestCashItemGranterReturnsTheAssetId(t *testing.T) {
	db, tm := newCouponTestDB(t)
	cid := seedEmptyCompartment(t, db, tm, 1003)
	l, gctx := testLoggerAndContext(t)
	g, _ := granterFor(l, gctx, NewCashItemReward(50200000, 1))
	mb := message.NewBuffer()

	got, err := g.Grant(mb)(db, redemptionContext{accountId: 1003, compartmentId: cid}, NewCashItemReward(50200000, 1))
	if err != nil {
		t.Fatalf("Grant: %v", err)
	}
	if got.assetId == 0 {
		t.Error("assetId = 0; the channel needs it to build the CashInventoryItem")
	}
}
```

- [ ] **Step 2: Run and watch it fail**

Run: `cd services/atlas-cashshop && go test ./atlas.com/cashshop/coupon/ -run 'TestGranter|TestCurrencyGranter|TestCashItemGranter' -v`
Expected: FAIL — undefined symbols.

- [ ] **Step 3: Implement**

Create `services/atlas-cashshop/atlas.com/cashshop/coupon/granter.go`:

```go
package coupon

import (
	"atlas-cashshop/cashshop/inventory/asset"
	"atlas-cashshop/cashshop/inventory/compartment"
	"atlas-cashshop/kafka/message"
	"atlas-cashshop/wallet"
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// redemptionContext is what a granter needs about the redeeming player,
// resolved once by the processor before the reward loop.
type redemptionContext struct {
	accountId     uint32
	characterId   uint32
	compartmentId uuid.UUID
}

// grantedReward is one granter's contribution to the success event. The
// currency fields are DELTAS — the amounts this coupon awarded — because
// UseCouponDone renders maplePoint inside "You have received ... using the
// coupon" and skips it when zero (derivation.md, "Blocking answer 1").
type grantedReward struct {
	assetId     uint32
	maplePoints uint32
	credit      uint32
}

// rewardGranter applies one reward INSIDE the redemption transaction. Every
// implementation today writes only to atlas-cashshop's own tables, which is
// exactly why redemption is a single local transaction rather than a saga
// (design §2). When a reward type owned by ANOTHER service is added, that
// granter is the single place a saga gets introduced.
type rewardGranter interface {
	Grant(mb *message.Buffer) func(tx *gorm.DB, rc redemptionContext, r Reward) (grantedReward, error)
}

func granterFor(l logrus.FieldLogger, ctx context.Context, r Reward) (rewardGranter, error) {
	switch r.Type() {
	case RewardTypeCurrency:
		return currencyGranter{l: l, ctx: ctx}, nil
	case RewardTypeCashItem:
		return cashItemGranter{l: l, ctx: ctx}, nil
	default:
		// Never silently skip: a coupon that claims to grant something and
		// grants nothing is worse than one that fails loudly.
		return nil, fmt.Errorf("%w: no granter for reward type %q", ErrInvalidReward, r.Type())
	}
}

type currencyGranter struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func (g currencyGranter) Grant(mb *message.Buffer) func(tx *gorm.DB, rc redemptionContext, r Reward) (grantedReward, error) {
	return func(tx *gorm.DB, rc redemptionContext, r Reward) (grantedReward, error) {
		wp := wallet.NewProcessor(g.l, g.ctx, tx)
		w, err := wp.WithTransaction(tx).GetByAccountId(rc.accountId)
		if err != nil {
			return grantedReward{}, err
		}
		w = w.Award(r.Currency(), r.Amount())
		if _, err = wp.WithTransaction(tx).Update(mb)(rc.accountId)(w.Credit())(w.Points())(w.Prepaid()); err != nil {
			return grantedReward{}, err
		}
		out := grantedReward{}
		switch r.Currency() {
		case 1:
			out.credit = r.Amount()
		case 2:
			out.maplePoints = r.Amount()
		}
		// Prepaid has no field in UseCouponDone; it is credited to the wallet
		// and shows up on the next CashQueryResult, which is what the client
		// refreshes balances from anyway.
		return out, nil
	}
}

type cashItemGranter struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func (g cashItemGranter) Grant(mb *message.Buffer) func(tx *gorm.DB, rc redemptionContext, r Reward) (grantedReward, error) {
	return func(tx *gorm.DB, rc redemptionContext, r Reward) (grantedReward, error) {
		// Re-read capacity INSIDE the transaction. The processor's pre-flight
		// ladder check gives a deterministic error ordering; this closes the
		// window between that check and this write (design Q6).
		cp := compartment.NewProcessor(g.l, g.ctx, tx)
		ccm, err := cp.GetById(rc.compartmentId)
		if err != nil {
			return grantedReward{}, err
		}
		if ccm.Capacity() <= uint32(len(ccm.Assets())) {
			return grantedReward{}, NewRedemptionError(ErrorKeyInventoryFull, "cash locker filled up during redemption")
		}

		am, err := asset.NewProcessor(g.l, g.ctx, tx).
			Create(mb)(rc.compartmentId, 0, r.SerialNumber(), r.Quantity(), 0, rc.characterId)
		if err != nil {
			return grantedReward{}, err
		}
		return grantedReward{assetId: am.Id()}, nil
	}
}
```

**Check the `asset.Create` argument order against the real signature before writing it** — `Create(mb)(compartmentId, templateId, commodityId, quantity, petId, purchasedBy)`. A coupon reward names the commodity by `serialNumber`, which is the `commodityId` position, and `asset.Create` resolves the template id from the commodity when `commodityId != 0` (see `asset/processor.go:84-90`, the `period`/commodity lookup). Confirm that passing `templateId = 0` with a non-zero `commodityId` produces the right asset; if it does not, resolve the commodity first with `commodity.Processor.GetById(serialNumber)` and pass both explicitly. Fix the test's expectations to match whichever is correct — do not guess.

Also confirm `compartment.Processor` exposes `GetById`; if it only exposes `GetByAccountIdAndType`, use that with the account id and the compartment type the processor already resolved, and carry the type on `redemptionContext`.

- [ ] **Step 4: Run the tests**

Run: `cd services/atlas-cashshop && go test -race ./atlas.com/cashshop/coupon/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-cashshop/atlas.com/cashshop/coupon
git commit -m "feat(coupon): reward granters for wallet currency and cash-locker items"
```

---

### Task 20: The redemption transaction

One `database.ExecuteTransaction` wrapping `message.Emit(outbox.EmitProvider(...))`, exactly as `PurchaseAndEmit` does (`cashshop/processor.go:90-96`). This is design §2's decision made concrete: `ROLLBACK` is the atomicity mechanism, and compensation — which can itself fail — is not needed.

The success event rides the **outbox** because it asserts a committed state change. Every failure event goes on the **direct producer path**, outside the transaction — the same distinction `Purchase` draws with its `rejectEmit` closure (`processor.go:100-103`): an event asserting "nothing happened" must not ride an outbox that implies a commit.

**Files:**
- Create: `services/atlas-cashshop/atlas.com/cashshop/coupon/processor.go`
- Create: `services/atlas-cashshop/atlas.com/cashshop/kafka/producer/coupon/producer.go`
- Test: `services/atlas-cashshop/atlas.com/cashshop/coupon/processor_test.go`

**Interfaces:**
- Consumes: everything from Tasks 11–19.
- Produces: `coupon.NewProcessor(l, ctx, db) Processor`, where `Processor` exposes `RedeemAndEmit(characterId uint32, code string) error` plus the CRUD methods `GetById`, `GetByCode`, `GetAll(f Filters)`, `Create`, `Update`, `Delete`. The in-transaction worker `redeem(mb *message.Buffer) func(tx *gorm.DB, req redeemRequest) (grantedTotals, error)` and the `redeemRequest` / `grantedTotals` structs are unexported — `RedeemAndEmit` is the only entry point.
- Producers: `CouponRedeemedStatusEventProvider(characterId uint32, compartmentId uuid.UUID, assetIds []uint32, maplePoints uint32, credit uint32)`, `CouponFailedStatusEventProvider(characterId uint32, errorKey string)`.
- Task 22 calls `RedeemAndEmit`; Task 23 calls the CRUD methods.

- [ ] **Step 1: Write the failing outcome tests**

Create `services/atlas-cashshop/atlas.com/cashshop/coupon/processor_test.go`. One test per FR-5.4 outcome, asserting the key the player would see:

```go
func TestRedeemLadderOutcomes(t *testing.T) {
	now := time.Now()
	for _, c := range []struct {
		name    string
		seed    func(t *testing.T, db *gorm.DB, tm tenant.Model) string // returns the code to submit
		wantKey string
	}{
		{
			"no such code",
			func(t *testing.T, db *gorm.DB, tm tenant.Model) string { return "NOSUCHCODE" },
			ErrorKeyInvalidCode,
		},
		{
			"inactive code",
			func(t *testing.T, db *gorm.DB, tm tenant.Model) string {
				seedCoupon(t, db, tm, NewBuilder("OFF").SetActive(false).SetRewards(Rewards{NewCurrencyReward(1, 1)}))
				return "OFF"
			},
			ErrorKeyNotRegistered,
		},
		{
			"not started yet",
			func(t *testing.T, db *gorm.DB, tm tenant.Model) string {
				seedCoupon(t, db, tm, NewBuilder("EARLY").SetStartsAt(ptrTime(now.Add(time.Hour))).SetRewards(Rewards{NewCurrencyReward(1, 1)}))
				return "EARLY"
			},
			ErrorKeyNotRegistered,
		},
		{
			"expired",
			func(t *testing.T, db *gorm.DB, tm tenant.Model) string {
				seedCoupon(t, db, tm, NewBuilder("OLD").SetExpiresAt(ptrTime(now.Add(-time.Hour))).SetRewards(Rewards{NewCurrencyReward(1, 1)}))
				return "OLD"
			},
			ErrorKeyExpired,
		},
		{
			"already redeemed by this account",
			func(t *testing.T, db *gorm.DB, tm tenant.Model) string {
				id := seedCoupon(t, db, tm, NewBuilder("TWICE").SetRewards(Rewards{NewCurrencyReward(1, 1)}))
				seedRedemption(t, db, tm, id, testAccountId)
				return "TWICE"
			},
			ErrorKeyAlreadyUsed,
		},
		{
			"global uses exhausted",
			func(t *testing.T, db *gorm.DB, tm tenant.Model) string {
				seedCoupon(t, db, tm, NewBuilder("GONE").SetMaxUses(ptrU32(1)).SetRedemptionCount(1).SetRewards(Rewards{NewCurrencyReward(1, 1)}))
				return "GONE"
			},
			ErrorKeyUsageLimit,
		},
		{
			"locker has no room for the item reward",
			func(t *testing.T, db *gorm.DB, tm tenant.Model) string {
				seedFullCompartmentForTestCharacter(t, db, tm)
				seedCoupon(t, db, tm, NewBuilder("ITEM").SetRewards(Rewards{NewCashItemReward(50200000, 1)}))
				return "ITEM"
			},
			ErrorKeyInventoryFull,
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			db, tm, ctx := newProcessorTestEnv(t)
			code := c.seed(t, db, tm)
			events := captureDirectEvents(t) // intercepts the direct producer path

			err := NewProcessor(testLogger(t), ctx, db).RedeemAndEmit(testCharacterId, code)
			if err == nil {
				t.Fatal("want a rejection")
			}
			if got := events.lastCouponFailure(); got != c.wantKey {
				t.Errorf("emitted key = %q, want %q", got, c.wantKey)
			}
			assertNoRedemptionRow(t, db, tm)
			assertWalletUnchanged(t, db, tm)
		})
	}
}

func TestRedeemSuccessGrantsAndEmits(t *testing.T) {
	db, tm, ctx := newProcessorTestEnv(t)
	seedWalletForTestAccount(t, db, tm, 0, 0, 0)
	seedEmptyCompartmentForTestCharacter(t, db, tm)
	seedCoupon(t, db, tm, NewBuilder("WIN").SetRewards(Rewards{
		NewCurrencyReward(2, 1500),
		NewCashItemReward(50200000, 1),
	}))

	if err := NewProcessor(testLogger(t), ctx, db).RedeemAndEmit(testCharacterId, "  win  "); err != nil {
		t.Fatalf("RedeemAndEmit: %v", err)
	}

	// Normalization: a lowercase, padded submission matched the stored code.
	r := loadOnlyRedemption(t, db, tm)
	if r.AccountId() != testAccountId {
		t.Errorf("redemption accountId = %d, want %d", r.AccountId(), testAccountId)
	}
	if len(r.RewardsGranted()) != 2 {
		t.Errorf("rewardsGranted = %d entries, want 2 (a snapshot of what was granted)", len(r.RewardsGranted()))
	}
	if p := loadWalletPoints(t, db, testAccountId); p != 1500 {
		t.Errorf("points = %d, want 1500", p)
	}
	if n := countAssets(t, db, tm); n != 1 {
		t.Errorf("locker assets = %d, want 1", n)
	}
	// The success event rides the OUTBOX, not the direct path.
	e := loadOnlyOutboxCouponEvent(t, db)
	if e.Body.MaplePoints != 1500 {
		t.Errorf("maplePoints = %d, want the 1500 DELTA this coupon awarded", e.Body.MaplePoints)
	}
	if len(e.Body.AssetIds) != 1 {
		t.Errorf("assetIds = %v, want one", e.Body.AssetIds)
	}
}

func TestRedeemRateLimitedShortCircuits(t *testing.T) {
	db, tm, ctx := newProcessorTestEnv(t)
	seedCoupon(t, db, tm, NewBuilder("REAL").SetRewards(Rewards{NewCurrencyReward(1, 1)}))
	exhaustLimiter(t, ctx, tm, testAccountId)
	events := captureDirectEvents(t)
	queries := countQueriesFrom(t, db)

	_ = NewProcessor(testLogger(t), ctx, db).RedeemAndEmit(testCharacterId, "REAL")

	// The whole point of the limiter is that a blocked attempt does NOT hit
	// the coupons table.
	if queries.couponSelects() != 0 {
		t.Errorf("coupon lookups = %d, want 0 for a rate-limited attempt", queries.couponSelects())
	}
	// It reports INVALID_COUPON_CODE, never a distinct "rate limited" key,
	// so a blocked attacker cannot tell a real code from a fake one.
	if got := events.lastCouponFailure(); got != ErrorKeyInvalidCode {
		t.Errorf("emitted key = %q, want %q", got, ErrorKeyInvalidCode)
	}
}

func TestRedeemSuccessResetsTheLimiter(t *testing.T) {
	db, tm, ctx := newProcessorTestEnv(t)
	seedWalletForTestAccount(t, db, tm, 0, 0, 0)
	seedCoupon(t, db, tm, NewBuilder("OK").SetRewards(Rewards{NewCurrencyReward(1, 1)}))
	recordFailures(t, ctx, tm, testAccountId, 3)

	if err := NewProcessor(testLogger(t), ctx, db).RedeemAndEmit(testCharacterId, "OK"); err != nil {
		t.Fatal(err)
	}
	if n := limiterCount(t, ctx, tm, testAccountId); n != 0 {
		t.Errorf("limiter count = %d after a success, want 0", n)
	}
}
```

- [ ] **Step 2: Run and watch it fail**

Run: `cd services/atlas-cashshop && go test ./atlas.com/cashshop/coupon/ -run TestRedeem -v`
Expected: FAIL — undefined `NewProcessor`.

- [ ] **Step 3: Write the producers**

Create `services/atlas-cashshop/atlas.com/cashshop/kafka/producer/coupon/producer.go` following `kafka/producer/cashshop/`'s shape:

```go
func CouponRedeemedStatusEventProvider(characterId uint32, compartmentId uuid.UUID, assetIds []uint32, maplePoints uint32, credit uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &cashshop.StatusEvent[cashshop.CouponRedeemedBody]{
		CharacterId: characterId,
		Type:        cashshop.StatusEventTypeCouponRedeemed,
		Body: cashshop.CouponRedeemedBody{
			CompartmentId: compartmentId,
			AssetIds:      assetIds,
			MaplePoints:   maplePoints,
			Credit:        credit,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func CouponFailedStatusEventProvider(characterId uint32, errorKey string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &cashshop.StatusEvent[cashshop.CouponFailedBody]{
		CharacterId: characterId,
		Type:        cashshop.StatusEventTypeCouponFailed,
		Body:        cashshop.CouponFailedBody{Error: errorKey},
	}
	return producer.SingleMessageProvider(key, value)
}
```

- [ ] **Step 4: Write the processor**

Create `services/atlas-cashshop/atlas.com/cashshop/coupon/processor.go`:

```go
package coupon

import (
	"atlas-cashshop/cashshop/inventory/compartment"
	"atlas-cashshop/character"
	"atlas-cashshop/configuration"
	"atlas-cashshop/coupon/redemption"
	"atlas-cashshop/kafka/message"
	kafkacashshop "atlas-cashshop/kafka/message/cashshop"
	couponproducer "atlas-cashshop/kafka/producer/coupon"
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type Processor interface {
	RedeemAndEmit(characterId uint32, code string) error
	GetById(id uuid.UUID) (Model, error)
	GetByCode(code string) (Model, error)
	GetAll(f Filters) ([]Model, error)
	Create(m Model) (Model, error)
	Update(id uuid.UUID, m Model) (Model, error)
	Delete(id uuid.UUID) error
}

type ProcessorImpl struct {
	l    logrus.FieldLogger
	ctx  context.Context
	db   *gorm.DB
	t    tenant.Model
	chaP character.Processor
	cicP compartment.Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{
		l:    l,
		ctx:  ctx,
		db:   db,
		t:    tenant.MustFromContext(ctx),
		chaP: character.NewProcessor(l, ctx),
		cicP: compartment.NewProcessor(l, ctx, db),
	}
}

var _ Processor = (*ProcessorImpl)(nil)

// RedeemAndEmit runs one redemption attempt end to end.
//
// Success rides the OUTBOX (it asserts a committed state change). Every
// failure goes on the DIRECT producer path outside the transaction: an event
// asserting "nothing happened" must not ride an outbox that implies a commit
// (the same distinction Purchase draws at cashshop/processor.go:100-103).
func (p *ProcessorImpl) RedeemAndEmit(characterId uint32, code string) error {
	code = Normalize(code)

	// Resolve the owning ACCOUNT: the packet arrives on a character session,
	// but wallets and the one-time-per-account rule are account-scoped.
	c, err := p.chaP.GetById(p.chaP.InventoryDecorator)(characterId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to resolve character [%d] for coupon redemption.", characterId)
		return p.rejectDirect(characterId, ErrorKeyUnknown)
	}
	accountId := c.AccountId()

	attempts, window := configuration.GetCouponRateLimit(p.l, p.ctx, p.t.Id())
	limiter := NewLimiter(attempts, window)
	allowed, lerr := limiter.Allowed(p.ctx, p.t, accountId)
	if lerr != nil {
		// Fail open — a Redis outage must not make every coupon un-redeemable.
		p.l.WithError(lerr).Warnf("Coupon rate limiter unavailable for account [%d]; allowing the attempt.", accountId)
	}
	if !allowed {
		p.l.Infof("Coupon attempt from account [%d] character [%d] blocked by the rate limiter.", accountId, characterId)
		// INVALID_COUPON_CODE, not a distinct key: a "rate limited" reply
		// would tell an attacker they had found a real code.
		return p.rejectDirect(characterId, ErrorKeyInvalidCode)
	}

	if !Plausible(code) {
		_ = limiter.RecordFailure(p.ctx, p.t, accountId)
		return p.rejectDirect(characterId, ErrorKeyInvalidCode)
	}

	compartmentType := lockerTypeFor(c.JobId())
	transactionId := uuid.New()

	var rejectKey string
	var totals grantedTotals
	txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(mb *message.Buffer) error {
			t, rerr := p.redeem(mb)(tx, redeemRequest{
				characterId:     characterId,
				accountId:       accountId,
				code:            code,
				compartmentType: compartmentType,
				transactionId:   transactionId,
			})
			if rerr != nil {
				var re *RedemptionError
				if errors.As(rerr, &re) {
					rejectKey = re.Key()
				} else {
					rejectKey = ErrorKeyUnknown
				}
				// Returning the error rolls the transaction back, which undoes
				// the reservation, the redemption row and every grant. There is
				// nothing to compensate.
				return rerr
			}
			totals = t
			return nil
		})
	})

	if rejectKey != "" {
		p.l.Infof("Coupon redemption rejected for account [%d] character [%d] code length [%d] transaction [%s]: %s",
			accountId, characterId, len(code), transactionId, rejectKey)
		_ = limiter.RecordFailure(p.ctx, p.t, accountId)
		return p.rejectDirect(characterId, rejectKey)
	}
	if txErr != nil {
		p.l.WithError(txErr).Errorf("Coupon redemption failed for account [%d] character [%d] transaction [%s].", accountId, characterId, transactionId)
		_ = limiter.RecordFailure(p.ctx, p.t, accountId)
		return p.rejectDirect(characterId, ErrorKeyUnknown)
	}

	p.l.Infof("Coupon redeemed by account [%d] character [%d] code [%s] transaction [%s].", accountId, characterId, code, transactionId)
	// A player who mistyped before getting it right should not be left one
	// failure away from a block.
	_ = limiter.Reset(p.ctx, p.t, accountId)
	return nil
}

func (p *ProcessorImpl) rejectDirect(characterId uint32, key string) error {
	return producer.ProviderImpl(p.l)(p.ctx)(kafkacashshop.EnvEventTopicStatus)(
		couponproducer.CouponFailedStatusEventProvider(characterId, key))
}

type redeemRequest struct {
	characterId     uint32
	accountId       uint32
	code            string
	compartmentType compartment.CompartmentType
	transactionId   uuid.UUID
}

type grantedTotals struct {
	compartmentId uuid.UUID
	assetIds      []uint32
	maplePoints   uint32
	credit        uint32
}

// redeem is the FR-5.4 ladder plus the grants, all inside one transaction.
func (p *ProcessorImpl) redeem(mb *message.Buffer) func(tx *gorm.DB, req redeemRequest) (grantedTotals, error) {
	return func(tx *gorm.DB, req redeemRequest) (grantedTotals, error) {
		var out grantedTotals

		// 1. code exists
		e, err := byCodeEntityProvider(p.t, req.code)(tx)()
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return out, NewRedemptionError(ErrorKeyInvalidCode, "no such code")
			}
			return out, err
		}
		m, err := Make(e)
		if err != nil {
			return out, err
		}

		// 2-4, 6. active / window / uses-remaining fast path
		if err := m.RedeemableAt(time.Now()); err != nil {
			return out, err
		}

		// 5. this account has no prior redemption
		prior, err := redemption.CountByCouponAndAccount(tx, p.t, m.Id(), req.accountId)
		if err != nil {
			return out, err
		}
		if prior > 0 {
			return out, NewRedemptionError(ErrorKeyAlreadyUsed, "account already redeemed this coupon")
		}

		// 7. locker capacity, pre-flight. The granter re-checks inside the
		// transaction (Q6); this check exists so the ERROR ORDERING is
		// deterministic — a full locker reports INVENTORY_FULL rather than
		// whichever grant happened to run first.
		ccm, err := p.cicP.GetByAccountIdAndType(req.accountId, req.compartmentType)
		if err != nil {
			return out, err
		}
		need := uint32(m.Rewards().CashItemCount())
		if need > 0 && ccm.Capacity() < uint32(len(ccm.Assets()))+need {
			return out, NewRedemptionError(ErrorKeyInventoryFull, "cash locker has no free slot")
		}
		out.compartmentId = ccm.Id()

		// Atomic reservation (FR-5.5). RowsAffected is the verdict.
		reserved, err := reserveUse(tx, p.t, m.Id())
		if err != nil {
			return out, err
		}
		if !reserved {
			return out, NewRedemptionError(ErrorKeyUsageLimit, "coupon has no uses remaining")
		}

		// The redemption row. A unique violation on
		// (tenant_id, coupon_id, account_id) is the same-account race loser,
		// not a redundant check.
		rm, err := redemption.NewBuilder(m.Id(), req.accountId, req.characterId).
			SetTransactionId(req.transactionId).
			SetRewardsGranted(m.Rewards()).
			SetRedeemedAt(time.Now()).
			Build()
		if err != nil {
			return out, err
		}
		if _, err = redemption.Create(tx, p.t, rm); err != nil {
			if redemption.IsUniqueViolation(err) {
				return out, NewRedemptionError(ErrorKeyAlreadyUsed, "account already redeemed this coupon")
			}
			return out, err
		}

		// Grants.
		rc := redemptionContext{accountId: req.accountId, characterId: req.characterId, compartmentId: ccm.Id()}
		for _, r := range m.Rewards() {
			g, gerr := granterFor(p.l, p.ctx, r)
			if gerr != nil {
				return out, gerr
			}
			got, gerr := g.Grant(mb)(tx, rc, r)
			if gerr != nil {
				return out, gerr
			}
			if got.assetId != 0 {
				out.assetIds = append(out.assetIds, got.assetId)
			}
			out.maplePoints += got.maplePoints
			out.credit += got.credit
		}

		// Committed with the state change, delivered by the outbox.
		return out, mb.Put(kafkacashshop.EnvEventTopicStatus,
			couponproducer.CouponRedeemedStatusEventProvider(req.characterId, out.compartmentId, out.assetIds, out.maplePoints, out.credit))
	}
}

// lockerTypeFor mirrors the branch Purchase uses at
// cashshop/processor.go:130-136.
func lockerTypeFor(jobId uint16) compartment.CompartmentType {
	switch job.GetType(jobId) {
	case job.TypeExplorer:
		return compartment.TypeExplorer
	case job.TypeCygnus:
		return compartment.TypeCygnus
	default:
		return compartment.TypeLegend
	}
}
```

Implement the CRUD methods (`GetById`, `GetByCode`, `GetAll`, `Create`, `Update`, `Delete`) over the Task 18 providers/administrators; Task 23 is their only caller. `Delete` returns `ErrHasRedemptions` when a redemption exists.

Check `job.GetType`'s parameter type against `libs/atlas-constants/job` and match it (`Purchase` passes `c.JobId()` directly). Add `redemption.CountByCouponAndAccount` and `redemption.Create` to Task 18's redemption administrator/provider if they are not there yet.

- [ ] **Step 5: Run the tests**

Run: `cd services/atlas-cashshop && go test -race ./atlas.com/cashshop/coupon/... -v`
Expected: PASS.

- [ ] **Step 6: Add the observability counters**

PRD §8 requires counters for attempts by outcome and successful redemptions. Follow the existing idiom in `services/atlas-channel/atlas.com/channel/monster/metrics.go` — `promauto.NewCounterVec` in a package-level `metrics.go`, labelled by tenant.

Create `services/atlas-cashshop/atlas.com/cashshop/coupon/metrics.go`:

```go
package coupon

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// attemptsTotal counts every redemption attempt that got past the handler,
	// labelled by the client-facing outcome key ("SUCCESS" or one of the
	// coupon.ErrorKey* values). The outcome label is a CLOSED set of at most
	// eight values, so it cannot explode cardinality — never label by code.
	attemptsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "atlas_cashshop_coupon_attempts_total",
			Help: "Coupon redemption attempts, by tenant and outcome.",
		},
		[]string{"tenant", "outcome"},
	)

	// rateLimitedTotal counts attempts short-circuited by the limiter. These
	// are reported to the player as INVALID_COUPON_CODE, so they are
	// indistinguishable from a genuine miss in attemptsTotal — this counter is
	// the only way to see brute-force pressure.
	rateLimitedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "atlas_cashshop_coupon_rate_limited_total",
			Help: "Coupon attempts blocked by the per-account rate limiter, by tenant.",
		},
		[]string{"tenant"},
	)
)

const outcomeSuccess = "SUCCESS"
```

**Never label a metric by the coupon code** — it is both a secret and unbounded cardinality.

Wire the increments into `RedeemAndEmit`: `rateLimitedTotal.WithLabelValues(p.t.Id().String()).Inc()` on the limiter branch, `attemptsTotal.WithLabelValues(p.t.Id().String(), rejectKey).Inc()` on each rejection path, `attemptsTotal.WithLabelValues(p.t.Id().String(), outcomeSuccess).Inc()` on success. A rate-limited attempt increments both (`INVALID_COUPON_CODE` and `rate_limited`), which is what makes the two series comparable.

There is no "compensated redemptions" counter: design §2 replaced compensation with a transaction rollback, so PRD §8's third counter has no event to count. Rollbacks show up as the non-`SUCCESS` outcomes already counted.

- [ ] **Step 7: Verify the guards**

Run: `tools/goroutine-guard.sh && tools/redis-key-guard.sh`
Expected: exit 0.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-cashshop/atlas.com/cashshop
git commit -m "feat(coupon): atomic redemption in a single local transaction"
```

---

### Task 21: Concurrency and rollback tests

These are the tests that actually matter here: they are the proof of design §2's decision and of FR-5.5/FR-5.6. They need **real goroutines against a real Postgres** — a mocked counter cannot exercise `RowsAffected` or a unique index.

**Files:**
- Test: `services/atlas-cashshop/atlas.com/cashshop/coupon/concurrency_test.go`

**Interfaces:**
- Consumes: Task 20's `RedeemAndEmit`.
- Produces: nothing — this task adds only tests.

- [ ] **Step 1: Write the tests**

Create `services/atlas-cashshop/atlas.com/cashshop/coupon/concurrency_test.go`:

```go
package coupon

import (
	"sync"
	"testing"
)

// FR-5.5: two concurrent redemptions of a max_uses = 1 coupon must yield
// exactly one success and one COUPON_USAGE_LIMIT. This fails if reserveUse is
// ever rewritten as a read-then-write.
func TestConcurrentRedemptionsOfASingleUseCoupon(t *testing.T) {
	db, tm, ctx := newProcessorTestEnv(t)
	seedWalletForTestAccount(t, db, tm, 0, 0, 0)
	seedWalletForAccount(t, db, tm, secondAccountId, 0, 0, 0)
	seedCoupon(t, db, tm, NewBuilder("ONESHOT").SetMaxUses(ptrU32(1)).SetRewards(Rewards{NewCurrencyReward(2, 100)}))
	events := captureDirectEvents(t)

	var wg sync.WaitGroup
	errs := make([]error, 2)
	chars := [2]uint32{testCharacterId, secondCharacterId} // DIFFERENT accounts
	start := make(chan struct{})
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			errs[i] = NewProcessor(testLogger(t), ctx, db).RedeemAndEmit(chars[i], "ONESHOT")
		}(i)
	}
	close(start)
	wg.Wait()

	successes := 0
	for _, e := range errs {
		if e == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("successes = %d, want exactly 1", successes)
	}
	if got := events.couponFailureKeys(); len(got) != 1 || got[0] != ErrorKeyUsageLimit {
		t.Errorf("failure keys = %v, want exactly [%s]", got, ErrorKeyUsageLimit)
	}
	if n := loadCountByCode(t, db, tm, "ONESHOT"); n != 1 {
		t.Errorf("redemptionCount = %d, want 1 — the loser must not have incremented it", n)
	}
	if n := countRedemptions(t, db, tm); n != 1 {
		t.Errorf("redemption rows = %d, want 1", n)
	}
}

// FR-5.6: two concurrent redemptions of the same code by the SAME account must
// yield exactly one success and one COUPON_ALREADY_USED, resolved by the
// unique index on (tenant_id, coupon_id, account_id).
func TestConcurrentRedemptionsBySameAccount(t *testing.T) {
	db, tm, ctx := newProcessorTestEnv(t)
	seedWalletForTestAccount(t, db, tm, 0, 0, 0)
	seedCoupon(t, db, tm, NewBuilder("SAMEACCT").SetRewards(Rewards{NewCurrencyReward(2, 100)}))
	events := captureDirectEvents(t)

	var wg sync.WaitGroup
	errs := make([]error, 2)
	start := make(chan struct{})
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			errs[i] = NewProcessor(testLogger(t), ctx, db).RedeemAndEmit(testCharacterId, "SAMEACCT")
		}(i)
	}
	close(start)
	wg.Wait()

	successes := 0
	for _, e := range errs {
		if e == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("successes = %d, want exactly 1", successes)
	}
	if got := events.couponFailureKeys(); len(got) != 1 || got[0] != ErrorKeyAlreadyUsed {
		t.Errorf("failure keys = %v, want exactly [%s]", got, ErrorKeyAlreadyUsed)
	}
	if p := loadWalletPoints(t, db, testAccountId); p != 100 {
		t.Errorf("points = %d, want 100 — the reward must be granted exactly once", p)
	}
	if n := countRedemptions(t, db, tm); n != 1 {
		t.Errorf("redemption rows = %d, want 1", n)
	}
}

// The single test that proves the §2 local-transaction decision: a failure in
// the item-grant step must leave NOTHING behind. With a saga this would be a
// compensation assertion; with a transaction it is a rollback assertion.
func TestGrantFailureRollsEverythingBack(t *testing.T) {
	db, tm, ctx := newProcessorTestEnv(t)
	seedWalletForTestAccount(t, db, tm, 0, 500, 0)
	// The currency reward is granted FIRST and the cash-item reward then fails,
	// so a partial commit would be visible as a credited wallet.
	seedCompartmentThatRejectsCreates(t, db, tm)
	seedCoupon(t, db, tm, NewBuilder("HALF").SetMaxUses(ptrU32(1)).SetRewards(Rewards{
		NewCurrencyReward(2, 1000),
		NewCashItemReward(50200000, 1),
	}))

	if err := NewProcessor(testLogger(t), ctx, db).RedeemAndEmit(testCharacterId, "HALF"); err == nil {
		t.Fatal("want a failure")
	}

	if p := loadWalletPoints(t, db, testAccountId); p != 500 {
		t.Errorf("points = %d, want the original 500 — the wallet must be untouched", p)
	}
	if n := countAssets(t, db, tm); n != 0 {
		t.Errorf("locker assets = %d, want 0", n)
	}
	if n := loadCountByCode(t, db, tm, "HALF"); n != 0 {
		t.Errorf("redemptionCount = %d, want 0 — the reservation must be rolled back", n)
	}
	if n := countRedemptions(t, db, tm); n != 0 {
		t.Errorf("redemption rows = %d, want 0", n)
	}
	// And the code must be redeemable again.
	seedEmptyCompartmentForTestCharacter(t, db, tm)
	if err := NewProcessor(testLogger(t), ctx, db).RedeemAndEmit(testCharacterId, "HALF"); err != nil {
		t.Errorf("retry after rollback: %v — the code must still be redeemable", err)
	}
}

// Codes differing only in case or surrounding whitespace resolve to the same
// coupon, and redeeming one blocks the other for that account.
func TestCaseAndWhitespaceVariantsAreTheSameCoupon(t *testing.T) {
	db, tm, ctx := newProcessorTestEnv(t)
	seedWalletForTestAccount(t, db, tm, 0, 0, 0)
	seedCoupon(t, db, tm, NewBuilder("MAPLE2026").SetRewards(Rewards{NewCurrencyReward(2, 10)}))
	events := captureDirectEvents(t)

	if err := NewProcessor(testLogger(t), ctx, db).RedeemAndEmit(testCharacterId, " maple2026 "); err != nil {
		t.Fatalf("first: %v", err)
	}
	if err := NewProcessor(testLogger(t), ctx, db).RedeemAndEmit(testCharacterId, "MaPlE2026"); err == nil {
		t.Fatal("second variant should have been rejected as already used")
	}
	if got := events.lastCouponFailure(); got != ErrorKeyAlreadyUsed {
		t.Errorf("key = %q, want %q", got, ErrorKeyAlreadyUsed)
	}
}
```

Write `seedCompartmentThatRejectsCreates` by filling the compartment to capacity — that makes the cash-item granter's in-transaction re-check fail, which is a real failure path rather than an injected fault.

- [ ] **Step 2: Run them**

Run: `cd services/atlas-cashshop && go test -race -count=5 ./atlas.com/cashshop/coupon/ -run 'TestConcurrent|TestGrantFailure|TestCaseAndWhitespace' -v`
Expected: PASS on every one of the five runs. `-count=5` is not optional — a race test that passes once has proved nothing.

- [ ] **Step 3: Confirm the tests fail when the invariant is removed**

Temporarily rewrite `reserveUse` as a read-then-write:

```go
	var e Entity
	if err := db.Where("id = ? AND tenant_id = ?", id, t.Id()).First(&e).Error; err != nil { return false, err }
	if e.MaxUses != nil && e.RedemptionCount >= *e.MaxUses { return false, nil }
	e.RedemptionCount++
	return true, db.Save(&e).Error
```

Run `go test -race -count=10 ./atlas.com/cashshop/coupon/ -run TestConcurrentRedemptionsOfASingleUseCoupon`.
Expected: **FAIL** on at least one run. If it passes ten times, the test is not actually racing — the two goroutines are probably serializing on a shared `*gorm.DB` or the `start` channel is not doing its job. Fix the test, then revert `reserveUse`.

- [ ] **Step 4: Revert and re-run**

Restore the conditional `UPDATE`, re-run Step 2, and confirm PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-cashshop/atlas.com/cashshop/coupon/concurrency_test.go
git commit -m "test(coupon): prove the max-uses and per-account races and the rollback guarantee"
```

---

### Task 22: Consume `REQUEST_COUPON_REDEMPTION` in `atlas-cashshop`

**Files:**
- Modify: `services/atlas-cashshop/atlas.com/cashshop/kafka/consumer/cashshop/consumer.go`

**Interfaces:**
- Consumes: Task 16's command contract, Task 20's `RedeemAndEmit`.
- Produces: a registered handler for `CommandTypeRequestCouponRedemption`. Task 24's channel producer is what feeds it.

- [ ] **Step 1: Add the handler**

Following the seven existing `handleCommandRequest*` functions in the same file:

```go
func handleCommandRequestCouponRedemption(db *gorm.DB) message.Handler[cashshop.Command[cashshop.RequestCouponRedemptionCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c cashshop.Command[cashshop.RequestCouponRedemptionCommandBody]) {
		if c.Type != cashshop.CommandTypeRequestCouponRedemption {
			return
		}
		// RedeemAndEmit owns the whole outcome, including emitting COUPON_FAILED
		// on the direct producer path, so a returned error is already reported
		// to the player and only needs logging here.
		if err := coupon.NewProcessor(l, ctx, db).RedeemAndEmit(c.CharacterId, c.Body.Code); err != nil {
			l.WithError(err).Debugf("Coupon redemption for character [%d] did not succeed.", c.CharacterId)
		}
	}
}
```

Register it in `InitHandlers` alongside the others, exactly as the seven siblings are registered (same `rf(t, message.AdaptHandler(message.PersistentConfig(...)))` shape — read lines 32-62 and match).

**Every handler on this topic receives every message** (`bug_monster_command_topic_shared_handler_unmarshal_collision`): the `c.Type != …` guard is what stops this handler from acting on a `REQUEST_PURCHASE`. Do not remove it, and do not assume the generic unmarshal into `RequestCouponRedemptionCommandBody` will fail for other command types — it will succeed with a zero-valued body.

- [ ] **Step 2: Build and test**

Run: `cd services/atlas-cashshop && go build ./... && go test -race ./... && go vet ./...`
Expected: clean.

- [ ] **Step 3: Commit**

```bash
git add services/atlas-cashshop/atlas.com/cashshop/kafka/consumer/cashshop/consumer.go
git commit -m "feat(cashshop): consume REQUEST_COUPON_REDEMPTION"
```

---

### Task 23: Admin REST surface

Exactly PRD §5 — three api2go resources, no redemption endpoint. **A REST redeem endpoint would be an unauthenticated reward faucet**; the packet path is the only trigger.

**Files:**
- Create: `services/atlas-cashshop/atlas.com/cashshop/coupon/rest.go`, `resource.go`
- Create: `services/atlas-cashshop/atlas.com/cashshop/coupon/batch/rest.go`, `resource.go`, `processor.go`
- Create: `services/atlas-cashshop/atlas.com/cashshop/coupon/redemption/rest.go`, `resource.go`, `processor.go`
- Create: `services/atlas-cashshop/atlas.com/cashshop/coupon/generator.go`
- Test: `services/atlas-cashshop/atlas.com/cashshop/coupon/generator_test.go`, `coupon/rest_test.go`
- Modify: `services/atlas-cashshop/atlas.com/cashshop/main.go:112-116`

**Interfaces:**
- Produces: `coupon.RestModel` (`GetName() == "coupons"`), `batch.RestModel` (`"coupon-batches"`), `redemption.RestModel` (`"coupon-redemptions"`), `coupon.InitResource`, `batch.InitResource`, `redemption.InitResource`, and `coupon.GenerateCode(length int) (string, error)`. Task 25's UI client consumes the JSON shapes.

**Endpoints (PRD §5):**

| Method | Path | Notes |
|---|---|---|
| `GET` | `/coupons` | filters `code`, `active`, `batchId`, `expiresBefore`, `expiresAfter`; paginated per `docs/rest-pagination.md` |
| `GET` | `/coupons/{id}` | |
| `POST` | `/coupons` | `code` optional — blank means generate |
| `PATCH` | `/coupons/{id}` | `active`, `startsAt`, `expiresAt`, `maxUses`, `description`, `rewards[]` |
| `DELETE` | `/coupons/{id}` | `409` when redemptions exist |
| `POST` | `/coupon-batches` | bulk generate |
| `GET` | `/coupon-batches`, `/coupon-batches/{id}` | includes generated-vs-redeemed counts |
| `GET` | `/coupons/{id}/redemptions` | |
| `GET` | `/coupon-redemptions?filter[accountId]=` | |

**Error mapping (PRD §5):** duplicate normalized code on create → `409`; reward references an unknown commodity serial → `422`; `expiresAt` ≤ `startsAt` → `422`; `maxUses` < current `redemptionCount` on PATCH → `422`; delete with redemptions → `409`.

- [ ] **Step 1: Write the failing generator test**

Create `services/atlas-cashshop/atlas.com/cashshop/coupon/generator_test.go`:

```go
package coupon

import (
	"strings"
	"testing"
)

func TestGenerateCodeUsesTheUnambiguousAlphabet(t *testing.T) {
	// No O/0, no I/1/L — these codes get read off a screen and typed by hand.
	banned := "O0I1L"
	for i := 0; i < 500; i++ {
		c, err := GenerateCode(10)
		if err != nil {
			t.Fatal(err)
		}
		if len(c) != 10 {
			t.Fatalf("len = %d, want 10", len(c))
		}
		if strings.ContainsAny(c, banned) {
			t.Fatalf("generated %q contains an ambiguous character", c)
		}
		if c != Normalize(c) {
			t.Fatalf("generated %q is not already normalized", c)
		}
	}
}

func TestGenerateCodeIsNotObviouslyPredictable(t *testing.T) {
	// Codes are secrets; math/rand would make them guessable. This is a smoke
	// test for "not a fixed sequence", not a statistical test.
	seen := map[string]bool{}
	for i := 0; i < 200; i++ {
		c, err := GenerateCode(12)
		if err != nil {
			t.Fatal(err)
		}
		if seen[c] {
			t.Fatalf("duplicate %q within 200 draws at length 12", c)
		}
		seen[c] = true
	}
}
```

- [ ] **Step 2: Run and watch it fail, then implement**

Run: `cd services/atlas-cashshop && go test ./atlas.com/cashshop/coupon/ -run TestGenerateCode -v` → FAIL.

Create `services/atlas-cashshop/atlas.com/cashshop/coupon/generator.go`:

```go
package coupon

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// codeAlphabet omits O/0 and I/1/L: these codes are read off a screen and
// typed by hand, and an ambiguous character turns a valid code into a support
// ticket.
const codeAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"

// GenerateCode draws from crypto/rand. Coupon codes are SECRETS — a code drawn
// from math/rand is guessable from a handful of observed codes.
func GenerateCode(length int) (string, error) {
	if length <= 0 || length > MaxCodeLength {
		return "", fmt.Errorf("%w: code length must be 1-%d", ErrInvalidCoupon, MaxCodeLength)
	}
	out := make([]byte, length)
	max := big.NewInt(int64(len(codeAlphabet)))
	for i := range out {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		out[i] = codeAlphabet[n.Int64()]
	}
	return string(out), nil
}
```

Run the test again → PASS.

- [ ] **Step 3: Write the REST models**

Create `services/atlas-cashshop/atlas.com/cashshop/coupon/rest.go` following `wallet/rest.go`'s shape:

```go
type RestModel struct {
	Id              uuid.UUID  `json:"-"`
	BatchId         *uuid.UUID `json:"batchId,omitempty"`
	Code            string     `json:"code"`
	Description     string     `json:"description,omitempty"`
	Active          bool       `json:"active"`
	StartsAt        *time.Time `json:"startsAt,omitempty"`
	ExpiresAt       *time.Time `json:"expiresAt,omitempty"`
	MaxUses         *uint32    `json:"maxUses,omitempty"`
	RedemptionCount uint32     `json:"redemptionCount"`
	Rewards         Rewards    `json:"rewards"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

func (r RestModel) GetName() string { return "coupons" }
```

with `GetID`/`SetID`/`Transform`/`Extract`, exactly as `wallet/rest.go` does. `Rewards` marshals through the Task 13 shape, so the REST attribute and the jsonb column are the same document.

Do the same for `batch.RestModel` (`"coupon-batches"`, with `RequestedCount`, `GeneratedCount`, `RedeemedCount`, and on POST the input-only `Count`, `Prefix`, `Length`, `StartsAt`, `ExpiresAt`, `Rewards`, `Description`, plus a `Codes []string` on the response) and `redemption.RestModel` (`"coupon-redemptions"`).

- [ ] **Step 4: Write the resources**

Create the three `resource.go` files following `wallet/resource.go`: `InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer`, `rest.RegisterHandler(l)(si)` for reads and `rest.RegisterInputHandler[RestModel](l)(si)` for writes, with the paths in the table above. Register all three in `main.go`:

```go
		AddRouteInitializer(coupon.InitResource(GetServer())(db)).
		AddRouteInitializer(batch.InitResource(GetServer())(db)).
		AddRouteInitializer(redemption.InitResource(GetServer())(db)).
```

- [ ] **Step 5: Implement bulk generation**

In `batch/processor.go`, one `database.ExecuteTransaction` that inserts the batch row and then `count` coupons, each with `MaxUses = 1` and the shared reward bundle:

```go
// generateOne inserts a single generated coupon, RETRYING on a code collision
// rather than skipping it, so GeneratedCount always equals RequestedCount and
// the caller's "500 codes" really is 500. The unique index on
// (tenant_id, code) is the collision detector — there is no pre-check, which
// would be a race.
const maxCollisionRetries = 10

func generateOne(tx *gorm.DB, t tenant.Model, prefix string, length int, tmpl coupon.Model) (coupon.Model, error) {
	for attempt := 0; attempt < maxCollisionRetries; attempt++ {
		suffix, err := coupon.GenerateCode(length)
		if err != nil {
			return coupon.Model{}, err
		}
		m, err := coupon.NewBuilder(prefix + suffix).
			SetBatchId(tmpl.BatchId()).
			SetDescription(tmpl.Description()).
			SetStartsAt(tmpl.StartsAt()).
			SetExpiresAt(tmpl.ExpiresAt()).
			SetMaxUses(ptrU32(1)).
			SetRewards(tmpl.Rewards()).
			Build()
		if err != nil {
			return coupon.Model{}, err
		}
		created, err := coupon.CreateEntity(tx, t, m)
		if err == nil {
			return created, nil
		}
		if !redemption.IsUniqueViolation(err) {
			return coupon.Model{}, err
		}
	}
	return coupon.Model{}, fmt.Errorf("unable to generate a unique coupon code after %d attempts; increase the length", maxCollisionRetries)
}
```

A collision-retry exhaustion aborts the **whole** transaction: a batch that silently produced 497 of 500 codes is worse than one that failed.

Note the `prefix + suffix` total must satisfy `Plausible` (≤ 32); reject a too-long combination at input validation with `422`.

- [ ] **Step 6: Write the REST behaviour tests**

Create `services/atlas-cashshop/atlas.com/cashshop/coupon/rest_test.go` covering, with real HTTP requests through the registered router:

- `POST /coupons` with an explicit code creates it; the same code again (and a case/whitespace variant of it) returns `409`.
- `POST /coupons` with a blank code generates one.
- `POST /coupons` with `expiresAt <= startsAt` returns `422`.
- `PATCH /coupons/{id}` with `maxUses` below the current `redemptionCount` returns `422`.
- `DELETE /coupons/{id}` succeeds with no redemptions, returns `409` once one exists.
- `POST /coupon-batches` with `count: 500` returns 500 **unique** codes and a batch whose `generatedCount` is 500.
- `GET /coupons?filter[active]=false` returns only inactive coupons.
- `GET /coupons/{id}/redemptions` and `GET /coupon-redemptions?filter[accountId]=` each return only that scope.
- Every list endpoint honours the pagination contract in `docs/rest-pagination.md` (read it and assert the same envelope the other paginated resources produce).
- A request carrying a *different* tenant's context cannot read or mutate this tenant's coupons.

- [ ] **Step 7: Run everything and commit**

Run: `cd services/atlas-cashshop && go test -race ./... && go vet ./... && go build ./...`
Expected: clean.

```bash
git add services/atlas-cashshop
git commit -m "feat(coupon): admin REST surface for coupons, batches, and redemptions"
```

---

## Phase 5 — `atlas-channel`

### Task 24: Channel handler for the coupon request

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_coupon_code.go`
- Modify: `services/atlas-channel/atlas.com/channel/cashshop/processor.go`
- Modify: `services/atlas-channel/atlas.com/channel/cashshop/producer.go`
- Modify: `services/atlas-channel/atlas.com/channel/main.go:926` (nearby)
- Test: `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_coupon_code_test.go`

**Interfaces:**
- Consumes: Task 4's `cashsb.CouponCode` and `cashsb.CashShopCouponCodeHandle`, Task 12's `coupon.Normalize`/`Plausible`, Task 16's command contract, Task 7's template routing.
- Produces: `handler.CashShopCouponCodeHandleFunc`, `cashshop.Processor.RequestCouponRedemption(characterId uint32, code string) error`, `cashshop.RequestCouponRedemptionCommandProvider`.

**This is a standalone handler, not an arm.** It is registered by name in `main.go` exactly like `CashShopCheckWalletHandleFunc` — do **not** add a branch to `cash_shop_operation.go`.

- [ ] **Step 1: Write the failing handler test**

Create `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_coupon_code_test.go`. Follow the existing handler tests in this directory for the session/reader harness:

```go
// An empty or over-long code is answered locally with INVALID_COUPON_CODE and
// never reaches Kafka — FR-4.3, and the first line of brute-force defence.
func TestCouponCodeHandlerShortCircuitsAnImplausibleCode(t *testing.T) {
	for _, c := range []struct{ name, code string }{
		{"empty", ""},
		{"whitespace only", "    "},
		{"over the column limit", strings.Repeat("A", 33)},
	} {
		t.Run(c.name, func(t *testing.T) {
			env := newCouponHandlerEnv(t)
			env.handle(couponPacket(t, env.ctx, "", c.code))
			if env.commandsPublished() != 0 {
				t.Errorf("published %d commands, want 0", env.commandsPublished())
			}
			if got := env.lastAnnouncedErrorKey(); got != cashcb.CashShopOperationErrorInvalidCouponCode {
				t.Errorf("announced %q, want %q", got, cashcb.CashShopOperationErrorInvalidCouponCode)
			}
		})
	}
}

// A plausible code is normalized ONCE, here, so the value on the wire to
// atlas-cashshop and the value in the database have the same shape.
func TestCouponCodeHandlerNormalizesBeforePublishing(t *testing.T) {
	env := newCouponHandlerEnv(t)
	env.handle(couponPacket(t, env.ctx, "", "  maple2026 "))
	if got := env.lastPublishedCode(); got != "MAPLE2026" {
		t.Errorf("published %q, want MAPLE2026", got)
	}
	if env.announcements() != 0 {
		t.Errorf("announced %d packets, want 0 — the reply comes from the status event", env.announcements())
	}
}

// The code is a secret: it must not appear in the logs.
func TestCouponCodeHandlerDoesNotLogTheCode(t *testing.T) {
	env := newCouponHandlerEnv(t)
	env.handle(couponPacket(t, env.ctx, "", "SECRETCODE"))
	if strings.Contains(env.logOutput(), "SECRETCODE") {
		t.Error("the coupon code leaked into the logs")
	}
}
```

- [ ] **Step 2: Run and watch it fail**

Run: `cd services/atlas-channel && go test ./atlas.com/channel/socket/handler/ -run TestCouponCodeHandler -v`
Expected: FAIL.

- [ ] **Step 3: Write the handler**

Create `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_coupon_code.go`:

```go
package handler

import (
	"atlas-channel/cashshop"
	"atlas-channel/cashshop/coupon"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/sirupsen/logrus"

	cashcb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/clientbound"
	cashsb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// CashShopCouponCodeHandleFunc handles the standalone serverbound COUPON_CODE
// op. This is NOT an arm of CashShopOperationHandle: the coupon submission has
// its own opcode and no mode byte (derivation.md, "Structural finding").
func CashShopCouponCodeHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := cashsb.CouponCode{}
		p.Decode(l, ctx)(r, readerOptions)
		// p.String() logs the code's LENGTH, never its value — codes are secrets.
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		// Normalize once, here, so the value sent to atlas-cashshop and the
		// value stored in the database have the same shape. The service
		// normalizes again defensively.
		code := coupon.Normalize(p.Code())
		if !coupon.Plausible(code) {
			// FR-4.3: empty or longer than the column can hold cannot match
			// anything, so answer locally with no round trip.
			err := session.Announce(l)(ctx)(wp)(cashcb.CashShopOperationWriter)(
				cashcb.CashShopUseCouponFailedBody(cashcb.CashShopOperationErrorInvalidCouponCode))(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to announce coupon rejection to character [%d].", s.CharacterId())
			}
			return
		}

		// TargetCharacter (p.TargetCharacter()) is deliberately ignored:
		// targeted / gift coupons are out of scope (PRD §2), and the plain
		// redeem path always sends it empty.
		if err := cashshop.NewProcessor(l, ctx).RequestCouponRedemption(s.CharacterId(), code); err != nil {
			l.WithError(err).Errorf("Unable to request coupon redemption for character [%d].", s.CharacterId())
		}
	}
}
```

- [ ] **Step 4: Add the request path**

In `services/atlas-channel/atlas.com/channel/cashshop/processor.go`, add `RequestCouponRedemption(characterId uint32, code string) error` to the `Processor` interface and:

```go
func (p *ProcessorImpl) RequestCouponRedemption(characterId uint32, code string) error {
	// Log the length, not the code.
	p.l.Debugf("Character [%d] submitting a coupon code of length [%d].", characterId, len(code))
	return producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvCommandTopic)(RequestCouponRedemptionCommandProvider(characterId, code))
}
```

In `producer.go`, following `RequestPurchaseCommandProvider`:

```go
func RequestCouponRedemptionCommandProvider(characterId uint32, code string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &cashshop.Command[cashshop.RequestCouponRedemptionCommandBody]{
		CharacterId: characterId,
		Type:        cashshop.CommandTypeRequestCouponRedemption,
		Body:        cashshop.RequestCouponRedemptionCommandBody{Code: code},
	}
	return producer.SingleMessageProvider(key, value)
}
```

- [ ] **Step 5: Register the handler**

In `services/atlas-channel/atlas.com/channel/main.go`, beside line 926:

```go
	handlerMap[cashsb.CashShopCouponCodeHandle] = handler.CashShopCouponCodeHandleFunc
```

The name **must** be byte-identical to the `handler` value Task 7 generated into the templates (`CashShopCouponCodeHandle`) — a mismatch produces only a `Warnf` and the opcode never routes.

- [ ] **Step 6: Prove the registration lines up**

Run:
```bash
python3 -c "
import glob, json, os
for p in sorted(glob.glob('services/atlas-configurations/seed-data/templates/template_*.json')):
    hs=json.load(open(p))['socket']['handlers']
    for h in hs:
        if h.get('handler')=='CashShopCouponCodeHandle':
            print(os.path.basename(p), h['opCode'], h.get('validator'), h.get('services'))
"
grep -n "CashShopCouponCodeHandle" services/atlas-channel/atlas.com/channel/main.go libs/atlas-packet/cash/serverbound/coupon_code.go
```
Expected: six templates each with a validator of `LoggedInValidator`, and the same literal string in `main.go` and the codec.

- [ ] **Step 7: Test, build, commit**

Run: `cd services/atlas-channel && go test -race ./... && go vet ./... && go build ./...`
Expected: clean.

```bash
git add services/atlas-channel
git commit -m "feat(channel): handle the serverbound COUPON_CODE request"
```

---

### Task 25: Channel response wiring

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/cashshop/consumer.go`
- Test: `services/atlas-channel/atlas.com/channel/kafka/consumer/cashshop/consumer_test.go`

**Interfaces:**
- Consumes: Task 16's status contracts; `cashpkt.CashShopUseCouponDoneBody` / `CashShopUseCouponFailedBody`; Task 8's `errors` table.
- Produces: two registered handlers, `handleStatusEventCouponRedeemed` and `handleStatusEventCouponFailed`.

- [ ] **Step 1: Write the failing tests**

Create/extend `services/atlas-channel/atlas.com/channel/kafka/consumer/cashshop/consumer_test.go`:

```go
func TestCouponRedeemedAnnouncesSuccessAndRefreshesTheWallet(t *testing.T) {
	env := newConsumerEnv(t)
	env.seedAsset(compartmentId, assetId)

	handleStatusEventCouponRedeemed(env.sc, env.wp)(env.logger, env.ctx, cashshop2.StatusEvent[cashshop2.CouponRedeemedBody]{
		CharacterId: 7,
		Type:        cashshop2.StatusEventTypeCouponRedeemed,
		Body: cashshop2.CouponRedeemedBody{
			CompartmentId: compartmentId,
			AssetIds:      []uint32{assetId},
			MaplePoints:   1500,
		},
	})

	// USE_COUPON_SUCCESS first, then CASH_QUERY_RESULT so the open Cash Shop
	// window shows the new balance without a relog.
	if got := env.announcedWriters(); !reflect.DeepEqual(got, []string{cashpkt.CashShopOperationWriter, cashpkt.CashQueryResultWriter}) {
		t.Errorf("announced %v", got)
	}
	body := env.decodeUseCouponDone(t)
	if body.MaplePoint() != 1500 {
		t.Errorf("maplePoint = %d, want the 1500 DELTA the event carried", body.MaplePoint())
	}
	if len(body.Items()) != 1 {
		t.Errorf("items = %d, want 1", len(body.Items()))
	}
	if body.Meso() != 0 {
		t.Errorf("meso = %d, want 0 — meso rewards are out of scope", body.Meso())
	}
}

func TestCouponFailedAnnouncesOnTheCouponArm(t *testing.T) {
	env := newConsumerEnv(t)
	handleStatusEventCouponFailed(env.sc, env.wp)(env.logger, env.ctx, cashshop2.StatusEvent[cashshop2.CouponFailedBody]{
		CharacterId: 7,
		Type:        cashshop2.StatusEventTypeCouponFailed,
		Body:        cashshop2.CouponFailedBody{Error: "COUPON_EXPIRED"},
	})
	// The USE_COUPON_FAILED mode, NOT the capacity-increase mode the generic
	// ERROR handler uses.
	if got := env.lastAnnouncedMode(); got != env.modeFor("USE_COUPON_FAILED") {
		t.Errorf("mode = %d, want the USE_COUPON_FAILED mode %d", got, env.modeFor("USE_COUPON_FAILED"))
	}
	if got := env.lastAnnouncedReasonByte(); got != env.errorByteFor("COUPON_EXPIRED") {
		t.Errorf("reason = %d, want %d — did the template errors table generate?", got, env.errorByteFor("COUPON_EXPIRED"))
	}
}

// UNKNOWN_ERROR is the jump table's DEFAULT case on every version, so it is
// deliberately not a key in the errors table. ResolveCode misses and returns
// 99, which is itself an unlisted byte and therefore renders the default
// notice — the intended outcome. Pin it so nobody "fixes" it by adding a
// bogus UNKNOWN_ERROR key to the templates.
func TestCouponFailedUnknownErrorFallsThroughToTheDefaultNotice(t *testing.T) {
	env := newConsumerEnv(t)
	handleStatusEventCouponFailed(env.sc, env.wp)(env.logger, env.ctx, cashshop2.StatusEvent[cashshop2.CouponFailedBody]{
		CharacterId: 7,
		Type:        cashshop2.StatusEventTypeCouponFailed,
		Body:        cashshop2.CouponFailedBody{Error: "UNKNOWN_ERROR"},
	})
	if got := env.lastAnnouncedReasonByte(); got != 99 {
		t.Errorf("reason = %d, want 99 (unconfigured -> default notice)", got)
	}
	if !env.reservedReasonBytes().excludes(99) {
		t.Fatal("99 collides with a reserved reason byte on some version — pick an explicit mapped key instead")
	}
}
```

The last assertion matters: `derivation.md` lists reserved bytes that change client state rather than showing a notice (v83 `0xA2`/`0xA4`/`0xB1`, v84 `171`/`173`/`186`, v92/v95 `0`/`2`/`15`). `99` is `0x63` — outside every version's switch domain on v83/v84/v87 and, on v92/v95, above the 69-case cap. Assert that explicitly rather than assuming it.

- [ ] **Step 2: Run and watch it fail**

Run: `cd services/atlas-channel && go test ./atlas.com/channel/kafka/consumer/cashshop/ -v`
Expected: FAIL.

- [ ] **Step 3: Implement the handlers**

Add to `services/atlas-channel/atlas.com/channel/kafka/consumer/cashshop/consumer.go`, modelled on `handleStatusEventPurchase`:

```go
func handleStatusEventCouponRedeemed(sc server.Model, wp writer.Producer) message.Handler[cashshop2.StatusEvent[cashshop2.CouponRedeemedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e cashshop2.StatusEvent[cashshop2.CouponRedeemedBody]) {
		if e.Type != cashshop2.StatusEventTypeCouponRedeemed {
			return
		}
		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, func(s session.Model) error {
			ap := asset.NewProcessor(l, ctx)
			items := make([]cashpkt.CashInventoryItem, 0, len(e.Body.AssetIds))
			for _, id := range e.Body.AssetIds {
				a, err := ap.GetById(s.AccountId(), e.Body.CompartmentId, id)
				if err != nil {
					l.WithError(err).Errorf("Unable to retrieve coupon asset [%d] for character [%d].", id, e.CharacterId)
					return err
				}
				items = append(items, cashpkt.CashInventoryItem{
					CashId:      a.Item().CashId(),
					AccountId:   s.AccountId(),
					CharacterId: e.CharacterId,
					TemplateId:  a.Item().TemplateId(),
					CommodityId: a.CommodityId(),
					Quantity:    int16(a.Item().Quantity()),
					GiftFrom:    "",
					Expiration:  packetmodel.MsTime(a.Expiration()),
				})
			}

			// maplePoint is the DELTA this coupon awarded — the client renders
			// it inside "You have received ... using the coupon" and skips it
			// when zero. meso is 0: meso rewards are out of scope (PRD §2).
			err := session.Announce(l)(ctx)(wp)(cashpkt.CashShopOperationWriter)(
				cashpkt.CashShopUseCouponDoneBody(items, int32(e.Body.MaplePoints), nil, 0))(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to announce coupon success to character [%d].", e.CharacterId)
				return err
			}

			// Refresh the balances so the OPEN Cash Shop window updates without
			// a relog: the client reads balances from CashQueryResult, never
			// from the coupon arm.
			w, err := wallet.NewProcessor(l, ctx).GetByAccountId(s.AccountId())
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve cash shop wallet for character [%d].", s.CharacterId())
				return nil
			}
			if err = session.Announce(l)(ctx)(wp)(cashpkt.CashQueryResultWriter)(
				cashpkt.NewCashQueryResult(w.Credit(), w.Points(), w.Prepaid()).Encode)(s); err != nil {
				l.WithError(err).Errorf("Unable to announce cash shop wallet to character [%d].", s.CharacterId())
			}
			return nil
		})
	}
}

func handleStatusEventCouponFailed(sc server.Model, wp writer.Producer) message.Handler[cashshop2.StatusEvent[cashshop2.CouponFailedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e cashshop2.StatusEvent[cashshop2.CouponFailedBody]) {
		if e.Type != cashshop2.StatusEventTypeCouponFailed {
			return
		}
		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}
		// The USE_COUPON_FAILED arm specifically — the generic ERROR handler
		// announces the capacity-increase arm, which is a different mode byte.
		op := session.Announce(l)(ctx)(wp)(cashpkt.CashShopOperationWriter)(cashpkt.CashShopUseCouponFailedBody(e.Body.Error))
		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, op)
	}
}
```

Register both in `InitHandlers` alongside the three existing handlers.

Check `CashShopUseCouponDoneBody`'s real signature in `libs/atlas-packet/cash/clientbound/shop_operation_body.go` before writing the call — `UseCouponDone` takes `(mode, items, maplePoint int32, refs []PackedCashItemRef, meso int32)` and the `*Body` wrapper resolves the mode itself. Match whatever the wrapper actually accepts; if no `CashShopUseCouponDoneBody` wrapper exists yet, add one in the same shape as `CashShopCashInventoryPurchaseSuccessBody`.

- [ ] **Step 4: Test, build, commit**

Run: `cd services/atlas-channel && go test -race ./... && go vet ./... && go build ./...`
Expected: clean.

```bash
git add services/atlas-channel libs/atlas-packet
git commit -m "feat(channel): announce coupon redemption success and failure"
```

---

## Phase 6 — Admin UI

### Task 26: Coupons API client and hooks

**Files:**
- Create: `services/atlas-ui/src/services/api/coupons.service.ts`
- Create: `services/atlas-ui/src/lib/hooks/api/useCoupons.ts`
- Test: `services/atlas-ui/src/services/api/__tests__/coupons.service.test.ts`
- Modify: `services/atlas-ui/src/services/api/index.ts`

**Interfaces:**
- Consumes: Task 23's JSON:API resources.
- Produces: `couponsService` (`list`, `getOne`, `create`, `update`, `remove`, `listRedemptions`, `listBatches`, `getBatch`, `generateBatch`), the `Coupon` / `CouponReward` / `CouponBatch` / `CouponRedemption` types, and the `useCoupons` / `useCoupon` / `useCouponRedemptions` / `useCreateCoupon` / `useUpdateCoupon` / `useDeleteCoupon` / `useGenerateCouponBatch` hooks plus a `couponKeys` query-key factory. Task 27 consumes all of them.

- [ ] **Step 1: Write the failing service test**

Create `services/atlas-ui/src/services/api/__tests__/coupons.service.test.ts`, matching the harness `accounts.service.test.ts` uses. Cover:

- `create` sends a **JSON:API envelope** — `{data:{type:"coupons", attributes:{…}}}`. A bare body 400s (`bug_ui_jsonapi_envelope_required_for_input_handlers`); assert the exact request body.
- `update` sends `{data:{type:"coupons", id, attributes:{…}}}` via `PATCH`.
- `generateBatch` posts to `/api/coupon-batches` with `{data:{type:"coupon-batches", attributes:{count, prefix, length, startsAt, expiresAt, rewards, description}}}` and returns the generated `codes` array.
- `list` passes `filter[active]`, `filter[code]`, `filter[batchId]` through as query params.
- `remove` surfaces a `409` as a typed error the UI can distinguish from a generic failure.

- [ ] **Step 2: Run and watch it fail**

Run: `cd services/atlas-ui && npm test -- coupons.service`
Expected: FAIL — module not found. (`nvm use 22` first if node is not already 22 — `reference_atlas_ui_npm_nvm_and_lint_baseline`.)

- [ ] **Step 3: Write the service**

Create `services/atlas-ui/src/services/api/coupons.service.ts` modelled on `mts-config.service.ts` and `accounts.service.ts`:

```ts
import { api } from "@/lib/api/client";
import type { ServiceOptions } from "@/lib/api/query-params";

export const COUPON_RESOURCE_TYPE = "coupons";
export const COUPON_BATCH_RESOURCE_TYPE = "coupon-batches";
export const COUPON_REDEMPTION_RESOURCE_TYPE = "coupon-redemptions";

/** Mirrors the Go coupon.Reward jsonb/REST document exactly. */
export type CouponReward =
  | { type: "CURRENCY"; currency: 1 | 2 | 3; amount: number }
  | { type: "CASH_ITEM"; serialNumber: number; quantity: number };

export interface CouponAttributes {
  batchId?: string;
  code: string;
  description?: string;
  active: boolean;
  startsAt?: string;
  expiresAt?: string;
  maxUses?: number;
  redemptionCount: number;
  rewards: CouponReward[];
  createdAt: string;
  updatedAt: string;
}

export interface Coupon {
  id: string;
  attributes: CouponAttributes;
}
```

…plus `CouponBatch`, `CouponRedemption`, the filter type, and the `couponsService` object. Every write wraps its body in the JSON:API envelope.

- [ ] **Step 4: Write the hooks**

Create `services/atlas-ui/src/lib/hooks/api/useCoupons.ts` following `useAccounts.ts`: a `couponKeys` factory, `useQuery`-based readers, and `useMutation`-based writers that invalidate `couponKeys.lists()` (and, for delete/update, `couponKeys.detail(id)`) on success.

- [ ] **Step 5: Run the tests and the build**

Run: `cd services/atlas-ui && npm test -- coupons && npm run build`
Expected: both pass. `npm run build` type-checks the tests too, so it is not redundant (`reference_atlas_ui_build_typechecks_tests`).

- [ ] **Step 6: Commit**

```bash
git add services/atlas-ui/src/services/api services/atlas-ui/src/lib/hooks/api/useCoupons.ts
git commit -m "feat(atlas-ui): coupons API client and query hooks"
```

---

### Task 27: Coupons admin pages

**Files:**
- Create: `services/atlas-ui/src/pages/CouponsPage.tsx`, `coupons-columns.tsx`, `CouponDetailPage.tsx`
- Create: `services/atlas-ui/src/components/features/coupons/CreateCouponDialog.tsx`, `GenerateCouponBatchDialog.tsx`, `RewardRowsField.tsx`
- Test: `services/atlas-ui/src/pages/__tests__/CouponsPage.test.tsx`, `.../CouponDetailPage.test.tsx`
- Modify: `services/atlas-ui/src/App.tsx`, `services/atlas-ui/src/components/app-sidebar-items.ts`

**Interfaces:**
- Consumes: Task 26's hooks and types.
- Produces: the routes `/coupons` and `/coupons/:couponId`, and a sidebar entry.

- [ ] **Step 1: Write the failing page tests**

Create `services/atlas-ui/src/pages/__tests__/CouponsPage.test.tsx`, following `AccountsPage.test.tsx`'s render/mocking harness. Cover:

- the list renders code, status, reward summary, uses (`redemptionCount / maxUses`, with `∞` when `maxUses` is null), and window;
- the active filter narrows the list;
- the create form rejects an invalid reward row before submitting (Zod discriminated union) — assert no mutation fired;
- the create form's `code` field is optional, and a blank submission sends no `code` attribute;
- the bulk-generate dialog offers a CSV download built from the response;
- **delete is disabled when `redemptionCount > 0`**, matching the server's `409`;
- the active toggle issues a `PATCH` with only `active` changed.

And `CouponDetailPage.test.tsx`: the detail page renders the redemption history for that code.

- [ ] **Step 2: Run and watch them fail**

Run: `cd services/atlas-ui && npm test -- CouponsPage CouponDetailPage`
Expected: FAIL — modules not found.

- [ ] **Step 3: Build the pages**

Follow the existing list/columns/detail trio (`AccountsPage.tsx` + `accounts-columns.tsx` + `AccountDetailPage.tsx`) exactly: TanStack Query for fetching, shadcn/ui components, Tailwind, the shared data-table.

The reward editor is a field array switching on `type`, validated by a **Zod discriminated union** so an invalid combination cannot reach the API:

```ts
const rewardSchema = z.discriminatedUnion("type", [
  z.object({
    type: z.literal("CURRENCY"),
    currency: z.union([z.literal(1), z.literal(2), z.literal(3)]),
    amount: z.coerce.number().int().positive(),
  }),
  z.object({
    type: z.literal("CASH_ITEM"),
    serialNumber: z.coerce.number().int().positive(),
    quantity: z.coerce.number().int().positive(),
  }),
]);

const couponSchema = z
  .object({
    // Blank means "generate one" — the server picks the code.
    code: z.string().trim().max(32).optional(),
    description: z.string().optional(),
    active: z.boolean().default(true),
    startsAt: z.string().optional(),
    expiresAt: z.string().optional(),
    maxUses: z.coerce.number().int().positive().optional(),
    rewards: z.array(rewardSchema).min(1, "A coupon must grant at least one reward"),
  })
  .refine(
    (v) => !v.startsAt || !v.expiresAt || new Date(v.expiresAt) > new Date(v.startsAt),
    { path: ["expiresAt"], message: "Expiry must be after activation" },
  );
```

The bulk-generate dialog builds its CSV client-side from the response's `codes` array — no extra endpoint.

Per design Q7 there is **no** global redemption list: history is per-code (the detail page) and per-account (`filter[accountId]`) only.

- [ ] **Step 4: Register the route and nav**

In `App.tsx`, add the lazy import beside the others and the routes:

```tsx
const CouponsPage = lazyWithReload(() =>
  import("@/pages/CouponsPage").then((m) => ({ default: m.CouponsPage })),
);
const CouponDetailPage = lazyWithReload(() =>
  import("@/pages/CouponDetailPage").then((m) => ({ default: m.CouponDetailPage })),
);
```
```tsx
                    <Route path="/coupons" element={<CouponsPage />} />
                    <Route path="/coupons/:couponId" element={<CouponDetailPage />} />
```

Add the sidebar entry in `services/atlas-ui/src/components/app-sidebar-items.ts` next to the other commerce entries.

- [ ] **Step 5: Run the tests, build, and lint**

Run:
```bash
cd services/atlas-ui && npm test && npm run build
```
Expected: both pass. Then from the repo root: `tools/lint.sh` (fix mode) followed by `tools/lint.sh --check`.

`tools/lint.sh --check` false-fails without nvm (`bug_lint_check_false_fails_without_nvm`) — make sure node 22 is active first, and if another worktree is running golangci-lint, wait rather than concluding the check is broken.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-ui
git commit -m "feat(atlas-ui): coupons admin pages with bulk generation and redemption history"
```

---

## Phase 7 — Adjacent template debt and the verification gate

The next two tasks are **not** required for coupon redemption to work — `USE_COUPON` is not a `CashShopOperationHandle` arm, so the serverbound `operations` tables are orthogonal. They are in scope because PRD FR-2.2 and design §5.1 asked for them, because `derivation.md` already contains most of the answer, and because the derivation surfaced a live client-facing bug (`BUY_NORMAL`).

### Task 28: Serverbound `CashShopOperationHandle` operations, and the `BUY_NORMAL` fix

`derivation.md` records **complete** serverbound arm tables for `gms_v83` (19 keys), `gms_v84` (19 keys + one unnamed mode 72) and `gms_v87` (19 keys + one unnamed mode 74). `gms_v92` is explicitly incomplete and `gms_v95` has three unresolved sites — Task 29 finishes those. Declare only the complete columns here: a version with no declared key is skipped entirely (`len(expected) == 0 → continue`), so partial coverage of the *file* is safe while partial coverage of a *version* is not.

**The `BUY_NORMAL` bug.** `template_gms_83_1.json` and `template_gms_84_1.json` both declare `"BUY_NORMAL": 20`. The clients push `20h` = **32** (`CCashShop::OnBuyNormal` ctor @ `0x46e79e` / `0x471227`). It is a hex-digits-read-as-decimal transcription error, and it means every v83/v84 "buy normal" request currently dispatches to whatever handler owns mode 20 — or to nothing.

**Files:**
- Create: `docs/packets/dispatchers/cash_shop_operation_handle.yaml`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_{83,84,87}_1.json` (generated)

- [ ] **Step 1: Write the dispatcher doc**

```yaml
# CashShopOperationHandle — serverbound CASHSHOP_OPERATION mode table.
#
# SOURCE OF TRUTH for this handler's options.operations map.
#
# The client has NO single cash-shop request builder: each UI action builds its
# own COutPacket(&pkt, <CASHSHOP_OPERATION opcode>) and writes its own leading
# Encode1(mode). The "mode switch" is a SERVER-side construct. Each version's
# arm set below was therefore enumerated by an exhaustive byte search for the
# opcode push (68 <op> 00 00 00 / 6A <op>) across the CCashShop region, so the
# table is complete BY CONSTRUCTION rather than by name search. Evidence and
# per-arm addresses: docs/tasks/task-206-cash-shop-coupon-codes/derivation.md.
#
# NOT LISTED HERE: USE_COUPON. The coupon submission is a SEPARATE opcode
# (registry op COUPON_CODE, CCashShop::OnStatusCoupon) with no mode byte at
# all. See docs/packets/dispatchers/cash_shop_coupon_code.yaml.
#
# ALL-OR-NOTHING: expectedTable reports every template key absent from this
# file as EXTRA, but only once a version has at least one declared key. A
# version listed here must therefore be enumerated COMPLETELY. gms_v92 and
# gms_v95 are deliberately absent until their enumerations are finished
# (derivation.md marks v92 INCOMPLETE and three v95 sites unresolved); the four
# legacy versions and jms_v185 have not been enumerated at all.
#
# Unnamed arms are omitted rather than guessed: gms_v84 mode 72 and gms_v87
# mode 74 are real sends whose Atlas key is unknown (derivation.md). Omitting
# them is safe — they are absent from the templates too — but it does mean this
# file is not yet a complete description of the CLIENT, only of the arms Atlas
# handles.

handler: CashShopOperationHandle
validator: LoggedInValidator
services: [channel]
fname: CCashShop::RequestCashPurchaseRecord
op: CASHSHOP_OPERATION
direction: serverbound
operations:
  - key: BUY
    modes: { gms_v83: 3, gms_v84: 3, gms_v87: 3 }
  - key: GIFT
    modes: { gms_v83: 4, gms_v84: 4, gms_v87: 4 }
  - key: SET_WISHLIST
    modes: { gms_v83: 5, gms_v84: 5, gms_v87: 5 }
  - key: INCREASE_INVENTORY
    modes: { gms_v83: 6, gms_v84: 6, gms_v87: 6 }
  - key: INCREASE_STORAGE
    modes: { gms_v83: 7, gms_v84: 7, gms_v87: 7 }
  - key: INCREASE_CHARACTER_SLOT
    modes: { gms_v83: 8, gms_v84: 8, gms_v87: 8 }
  - key: ENABLE_EQUIP_SLOT
    modes: { gms_v83: 9, gms_v84: 9, gms_v87: 9 }
  - key: MOVE_FROM_CASH_INVENTORY
    modes: { gms_v83: 13, gms_v84: 13, gms_v87: 13 }
  - key: MOVE_TO_CASH_INVENTORY
    modes: { gms_v83: 14, gms_v84: 14, gms_v87: 14 }
  - key: REBATE_LOCKER_ITEM
    modes: { gms_v83: 26, gms_v84: 26, gms_v87: 26 }
  - key: BUY_COUPLE
    modes: { gms_v83: 29, gms_v84: 29, gms_v87: 29 }
  - key: BUY_PACKAGE
    modes: { gms_v83: 30, gms_v84: 30, gms_v87: 30 }
  - key: BUY_OTHER_PACKAGE
    modes: { gms_v83: 31, gms_v84: 31, gms_v87: 31 }
  # 32 == 0x20. The v83/v84 templates carried 20 — a hex-read-as-decimal
  # transcription error. CCashShop::OnBuyNormal pushes 20h at 0x46e7ab (v83),
  # 0x471234 (v84), 0x478ec3 (v87).
  - key: BUY_NORMAL
    modes: { gms_v83: 32, gms_v84: 32, gms_v87: 32 }
  - key: APPLY_WISHLIST
    modes: { gms_v83: 33, gms_v84: 33, gms_v87: 33 }
  - key: BUY_FRIENDSHIP
    modes: { gms_v83: 35, gms_v84: 35, gms_v87: 35 }
  # v87 shifts the last three by +2.
  - key: GET_PURCHASE_RECORD
    modes: { gms_v83: 40, gms_v84: 40, gms_v87: 42 }
  - key: BUY_NAME_CHANGE
    modes: { gms_v83: 46, gms_v84: 46, gms_v87: 48 }
  - key: BUY_WORLD_TRANSFER
    modes: { gms_v83: 49, gms_v84: 49, gms_v87: 51 }
```

- [ ] **Step 2: Cross-check against `derivation.md`**

Reuse Task 8 Step 2's script against the `operations` list instead of `errors`, and require `0 unevidenced`. Then confirm the v83 and v84 key **sets** match the templates' existing 19 keys exactly:

```bash
python3 - <<'PY'
import json, yaml
doc = yaml.safe_load(open("docs/packets/dispatchers/cash_shop_operation_handle.yaml"))
for ver, path in (("gms_v83","template_gms_83_1.json"), ("gms_v84","template_gms_84_1.json")):
    want = {e["key"] for e in doc["operations"] if ver in e["modes"]}
    d = json.load(open("services/atlas-configurations/seed-data/templates/"+path))
    h = [x for x in d["socket"]["handlers"] if x.get("handler")=="CashShopOperationHandle"][0]
    got = set(h.get("options",{}).get("operations",{}))
    print(ver, "missing from yaml:", sorted(got-want), "| new in yaml:", sorted(want-got))
PY
```
Expected: both lists empty for v83 and v84. **A non-empty "missing from yaml" would make `--check` report those keys EXTRA** — add them (with derivation evidence) before generating.

- [ ] **Step 3: Generate and review the diff**

Run:
```bash
go run ./tools/packet-audit operations
git diff services/atlas-configurations/seed-data/templates
```
Expected: v83 and v84 change **only** `BUY_NORMAL` 20 → 32; v87 gains all 19 keys where it had none. No other template touched.

- [ ] **Step 4: Guards and matrix**

Run:
```bash
go run ./tools/packet-audit operations --check
tools/template-opcode-order-guard.sh
tools/template-duplicate-binding-guard.sh
go run ./tools/packet-audit matrix
```
Expected: all exit 0.

- [ ] **Step 5: Commit**

```bash
git add docs/packets/dispatchers/cash_shop_operation_handle.yaml services/atlas-configurations/seed-data/templates docs/packets/audits
git commit -m "fix(templates): tool-govern the serverbound cash-shop arm table and correct BUY_NORMAL to 32"
```

---

### Task 29: Finish the `gms_v92` and `gms_v95` serverbound arm attribution

`derivation.md` names exactly what is outstanding, so this is bounded work, not open-ended research:

- **gms_v92** — 25 candidate send sites, mode bytes read at 19, but the IDB has **no names on the request-builder functions**, so a mode cannot be attributed to a key without decompiling each sender. Four sites (`0x4805a2`, `0x48b7af`, `0x48c56e`, `0x48cead`) push their mode through a register and were not resolved. `template_gms_92_1.json` declares **zero** serverbound operations.
- **gms_v95** — 25 candidate sites, 22 read, 19 with a mode byte. Three remain: `0x483b82` (`CCashShop::OnRemoveWish`), `0x4901aa` (`CCashShop::OnGiftMateInfoResult`), `0x490ad8` (`CCashShop::OnGiftPackage`), each pushing its mode through a register. By analogy with v83/v87 they are `SET_WISHLIST`, `GIFT` and `BUY_OTHER_PACKAGE` — **that is an inference, not a decompile.** `template_gms_95_1.json` also declares zero.

**Files:**
- Modify: `docs/tasks/task-206-cash-shop-coupon-codes/derivation.md`
- Modify: `docs/packets/dispatchers/cash_shop_operation_handle.yaml`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_{92,95}_1.json` (generated)

- [ ] **Step 1: Resolve the three v95 sites**

Decompile each of `CCashShop::OnRemoveWish` (`0x483b82`), `CCashShop::OnGiftMateInfoResult` (`0x4901aa`) and `CCashShop::OnGiftPackage` (`0x490ad8`) in session for `GMS_v95.0_U_DEVM.exe.i64` and follow the register that reaches `Encode1`. Record the actual byte and its address. Do **not** write the inferred value if the decompile disagrees — v95 already shifted `ENABLE_EQUIP_SLOT` 9→10, `MOVE_FROM_CASH_INVENTORY` 13→14, `REBATE_LOCKER_ITEM` 26→28 and others, so analogy is exactly what would be wrong here.

Update the v95 table's **Row-count self-check** to `25 candidate sites, 25 read → N rows`, and remove the "not certified complete" note only once it is.

- [ ] **Step 2: Attribute the v92 arms**

Decompile and `rename` the 25 v92 sender functions in session for `GMS_v92_1_DEVM.exe.i64`, including the four register-pushed sites. For each, record the function name, the ctor address, the mode byte and its address, and the matching Atlas key. Where a sender has no Atlas key (v84's mode 72 / v87's mode 74 family, v95's mode 76 `OnCashGachaponCopy`), record it as **key unknown / unverified** and omit it from the YAML rather than inventing a name.

Replace the "Serverbound `operations` — INCOMPLETE (explicit gap)" subsection with a proper key → mode table and a Row-count self-check.

- [ ] **Step 3: Extend the YAML**

Add the `gms_v92` and `gms_v95` keys to every `operations` row in `cash_shop_operation_handle.yaml`. Each version must be enumerated **completely** — that is the all-or-nothing rule, and both templates are currently empty, so nothing will be reported EXTRA, but a half-declared version becomes a trap the moment anyone adds a key by hand.

Re-run Task 28 Step 2's cross-check script and require `0 unevidenced`.

- [ ] **Step 4: Generate, guard, commit**

Run:
```bash
go run ./tools/packet-audit operations
git diff services/atlas-configurations/seed-data/templates
go run ./tools/packet-audit operations --check
tools/template-opcode-order-guard.sh
tools/template-duplicate-binding-guard.sh
go run ./tools/packet-audit matrix
```
Expected: `template_gms_92_1.json` and `template_gms_95_1.json` each gain their full arm table; all checks exit 0.

```bash
git add docs/tasks/task-206-cash-shop-coupon-codes/derivation.md docs/packets/dispatchers/cash_shop_operation_handle.yaml services/atlas-configurations/seed-data/templates docs/packets/audits
git commit -m "feat(templates): complete the gms_v92 and gms_v95 serverbound cash-shop arm tables"
```

---

### Task 30: Full verification gate

Nothing here is optional, and none of it may be reported from memory: run each command and read its output.

**Files:** none — this task only verifies and, if needed, fixes.

- [ ] **Step 1: Go module gates**

Run, from the worktree root, for each changed module:

```bash
for m in libs/atlas-packet libs/atlas-redis libs/atlas-tenant tools/packet-audit \
         services/atlas-cashshop services/atlas-channel; do
  echo "=== $m ==="
  ( cd "$m" && go build ./... && go vet ./... && go test -race ./... ) || echo "FAILED: $m"
done
```
Expected: no `FAILED` lines. `libs/atlas-tenant` is in the list only because Task 4 depends on `MajorAtLeast`; drop it if nothing under it changed.

- [ ] **Step 2: Repo-root guards**

```bash
tools/redis-key-guard.sh
tools/goroutine-guard.sh
tools/skill-job-id-guard.sh
tools/buff-duration-guard.sh
tools/template-opcode-order-guard.sh
tools/template-duplicate-binding-guard.sh
tools/template-movement-types-guard.sh
tools/service-registration-guard.sh
```
Expected: every one exits 0. `service-registration-guard.sh` is included to *prove* no service registration changed, not because one did.

- [ ] **Step 3: Lint**

```bash
tools/lint.sh            # fix mode — rewrites files
git diff --stat          # review what it rewrote
tools/lint.sh --check
```
Expected: `--check` exits 0. Ensure node 22 is active first, and if a sibling worktree is holding the golangci-lint lock, wait for it — a lock-contention failure is not a lint failure.

- [ ] **Step 4: Packet-audit gates**

```bash
go run ./tools/packet-audit matrix
go run ./tools/packet-audit operations --check
go run ./tools/packet-audit fname-doc --check
go run ./tools/packet-audit dispatcher-lint
git diff --stat docs/packets/audits
```
Expected: all exit 0, and `matrix` produces no diff (it was regenerated and committed in the tasks that changed it — a diff here means a commit was missed).

Confirm the coverage row actually moved:
```bash
python3 -c "
import json
r=[x for x in json.load(open('docs/packets/audits/status.json'))['rows'] if x.get('op')=='COUPON_CODE'][0]
for k,v in r['cells'].items(): print(k, v.get('state'), v.get('opcode'))
"
```
Expected: `gms_v48`/`v61`/`v72`/`v79` = `n-a` (unless Task 3 promoted one); the six in-scope cells verified, with `gms_v84` at opcode `236`.

- [ ] **Step 5: Docker bake for every service whose `go.mod` was touched**

```bash
docker buildx bake atlas-cashshop atlas-channel atlas-configurations
```
Expected: success. This is **mandatory** — `go build` against the workspace `go.work` cannot catch a missing `COPY libs/...` line in the shared root `Dockerfile`, and `libs/atlas-redis` is a new dependency of `atlas-cashshop`. If the bake fails on a missing lib, add the two `COPY` lines (mod-only block and source block) to the repo-root `Dockerfile` and re-bake.

`atlas-saga-orchestrator` is deliberately **not** in this list — design §2 removed it from the blast radius.

- [ ] **Step 6: `atlas-ui`**

```bash
cd services/atlas-ui && npm run build && npm test && cd -
```
Expected: both pass. `npm run build` type-checks the tests, so a green `npm test` alone is not sufficient.

- [ ] **Step 7: End-to-end on a live v83 tenant**

Deploy the branch to an ephemeral environment and, through the actual client:

1. Mint a code via the UI with a Maple Points reward and a cash-item reward.
2. Open the Cash Shop, enter the code in the Coupon tab → both rewards land, the balance and locker update **without a relog**.
3. Enter the same code again → "already used".
4. Enter a garbage code → "invalid code".
5. Enter an expired code → "expired".
6. Enter a code whose global uses are exhausted → "usage limit".

Record the outcome of each of the six in `docs/tasks/task-206-cash-shop-coupon-codes/verification.md`, with the pod logs backing the two that are easy to get wrong (the delta-vs-balance rendering in step 2 and the arm selection in steps 3-6). If a message renders as the generic default notice rather than the specific text, that is an `aligned` error row whose ordinal alignment was wrong — record the observed byte and the observed text and correct `derivation.md` and the YAML from the observation. **That is the single most valuable evidence this task can produce**, because it converts an `aligned` row into a `verified` one.

- [ ] **Step 8: Code review before the PR**

Run `superpowers:requesting-code-review`. It dispatches `plan-adherence-reviewer`, `backend-guidelines-reviewer` (Go changed) and `frontend-guidelines-reviewer` (TS changed) in parallel; pin them to Sonnet, not Fable. Each writes to `docs/tasks/task-206-cash-shop-coupon-codes/audit.md`. Ensure they run **inside this worktree** and that `git status` is clean afterwards.

Also dispatch `packet-completeness-critic` — this branch touches codecs, gates and the matrix, which is exactly the CHANGED-BUT-UNCLAIMED / CLAIMED-BUT-UNVERIFIED failure mode it exists to catch. Write `docs/tasks/task-206-cash-shop-coupon-codes/coverage-manifest.yaml` first, declaring the op × version cells this task claims.

- [ ] **Step 9: Prove no previously-verified cell changed wire behaviour**

PRD §10 requires this explicitly, and it is easy to violate by accident — Task 28 changes `BUY_NORMAL` on two versions and Task 9 may regenerate up to 57 `gms_v92` clientbound keys.

```bash
git diff main...HEAD -- libs/atlas-packet | grep -E '^[-+]' | grep -vE '^[-+]{3}' | grep -E 'Write|Read|Encode|Decode' 
git diff main...HEAD -- services/atlas-configurations/seed-data/templates
```

Every wire-affecting change must be one of these three, and nothing else:

1. the **new** `CouponCode` codec and the new `CashShopCouponCodeHandle` template entries (new behaviour, no prior cell);
2. the **new** `options.errors` tables (the prior behaviour was `ResolveCode` returning 99 — a broken cell, not a verified one);
3. the deliberate, evidenced corrections: `INVALID_COUPON_COUPON` → `INVALID_COUPON_CODE` (Task 5, unconsumed constant), `gms_v84` `COUPON_CODE` 230 → 236 (Task 1), `BUY_NORMAL` 20 → 32 on v83/v84 (Task 28), and any `gms_v92` clientbound key Task 9 corrected.

List each item in category 3 in `verification.md` with its `derivation.md` evidence address. **Anything outside these three categories is a regression** — investigate it before opening the PR.

- [ ] **Step 10: Commit the verification record**

```bash
git add docs/tasks/task-206-cash-shop-coupon-codes
git commit -m "docs(task-206): record the verification gate results"
```

---

## Self-Review: spec coverage

| Spec item | Task(s) | Note |
|---|---|---|
| PRD FR-1.1–1.4 serverbound codec + fixtures | 4 | |
| PRD FR-1.2 per-version derivation | 2, 3 (+ landed `a36149bf`) | |
| PRD FR-1.5 registry + matrix promotion | 1, 4 | Design §13.2 corrected "new registry op" → the op already exists; this task adds `packet:` and promotes the cells. |
| PRD FR-2.1 `USE_COUPON` in `operations` | — | **Superseded.** `derivation.md`'s structural finding proves the coupon request is a standalone op with no mode byte. Task 7 routes it as its own handler instead. |
| PRD FR-2.2 backfill short/empty `operations` tables | 28, 29 | |
| PRD FR-2.3 config-resolved mode byte | 7, 24 | No mode byte exists; the *opcode* is config-resolved through the template. |
| PRD FR-2.4 template guards | 7, 8, 28, 29, 30 | |
| PRD FR-3.1 success arm | 25 | |
| PRD FR-3.2 failure arm | 25 | |
| PRD FR-3.3 `errors` table on all templates | 8, 9 | |
| PRD FR-3.4 seven outcome→key mappings | 14 (`ErrorKey*`), 8 (bytes), 20 (ladder) | |
| PRD FR-3.5 key-string correction | 5 | |
| PRD FR-4.1–4.4 channel arm | 24 | FR-4.1's "arm in `cash_shop_operation.go`" is superseded — it is a standalone handler. |
| PRD FR-5.1–5.3 domain shape, migrations, normalized codes | 12, 13, 14, 15 | |
| PRD FR-5.4 validation ladder | 14 (`RedeemableAt`), 20 (full ladder) | |
| PRD FR-5.5 atomic counter | 18, 21 | |
| PRD FR-5.6 redemption uniqueness | 15, 21 | |
| PRD FR-6 saga orchestration | — | **Superseded in full** by design §2 (local transaction). Recorded in Global Constraints. |
| PRD FR-7.1–7.4 admin REST | 23 | |
| PRD FR-8.1–8.6 admin UI | 26, 27 | |
| PRD §6 data model | 14, 15 | |
| PRD §8 multi-tenancy | Constraints; asserted in 21, 23 | |
| PRD §8 concurrency | 21 | |
| PRD §8 security (no redeem endpoint) | 23 | |
| PRD §8 rate limiting | 10, 17, 20 | |
| PRD §8 observability | 20 (logging + counters) | "Compensated redemptions" has no event under the local-transaction design; noted in Task 20 Step 6. |
| PRD §8 performance | 18 (single indexed point read) | |
| PRD §8 correctness gates | 30 | |
| Design §2 local transaction | 20; proved by 21 | |
| Design §2.1 `rewardGranter` seam | 19 | |
| Design §2.2 dedup by unique index | 15, 21 | |
| Design §3.1 Kafka contracts | 16 | |
| Design §5.1 serverbound tables tool-governed | 28, 29 | |
| Design §5.2 `gms_v92` clientbound hole | 9 | |
| Design §5.3 full error enum, tool-generated | 8, 9 | `packet-audit` support landed in `15f81152`. |
| Design §5.4 typo correction | 5 | |
| Design §7.5 rate limiter | 10, 17 | |
| Design §10 testing | 4, 12, 13, 14, 17, 18, 19, 20, 21, 23, 26, 27 | |
| Design §11 verification gates | 30 | |
| Design Q4 (no echo) / Q5 (delta) | Resolved in `derivation.md`; applied in 16, 25 | |
| Design Q6 pre-flight + in-transaction re-check | 19, 20 | |
| Design Q7 no global redemption list | 27 | |

**Known scope boundaries, stated rather than hidden:**

- `gms_v48`/`v61`/`v72`/`v79` are `n-a` for the coupon request. Task 3 makes that verdict evidence-backed; if it finds a send, Task 3 Step 3 wires it rather than deferring.
- `gms_v84` mode 72 and `gms_v87` mode 74 are real client sends with **no known Atlas key**. They are recorded in `derivation.md` and deliberately omitted from the YAML rather than guessed. Naming them requires pairing each with its clientbound result arm in `docs/tasks/task-183-cashshop-result-family/arm-catalog.md` — out of scope here, and recorded as unknown, not as absent.
- Five `gms_v92`/`gms_v95` reason bytes (63–67) and one `gms_v87` reason byte (249) have no Atlas constant. Same treatment.
- Most `errors` key **names** are ordinal alignment, not decompiled text (see the evidence-confidence caveat in Global Constraints). Task 30 Step 7 is the cheapest opportunity to convert some of them into observed evidence.

---

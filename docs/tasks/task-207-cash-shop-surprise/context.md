# Cash Shop Surprise — Implementation Context

Companion to [`plan.md`](plan.md). Read this first if you are picking the task up cold.

Task: `task-207-cash-shop-surprise` · Worktree: `.worktrees/task-207-cash-shop-surprise` · Branch: `task-207-cash-shop-surprise`

---

## 1. What this feature is

Item `5222000` ("Cash Shop Surprise") is a loot box that lives in the **cash locker**, not the character inventory. Double-clicking it there consumes one and grants a randomly-rolled **cash** item into the same locker. The in-game text is explicit: *"you'll only be able to open the box in the cash inventory."*

Atlas had no implementation — before this branch, `5222000` appeared nowhere in `services/` or `libs/`, only in the backlog at `docs/research/missing-features/economy-and-trade.md:29`. That file is **not tracked in git** — not at the branch point, not at HEAD, not in the main checkout either (it has sat untracked in the working tree since 2026-08-07, a local research artifact rather than a committed doc). Task 20 therefore deliberately did **not** update it as part of this branch: there is nothing to mark delivered in version control, and copying an untracked scratch file into the branch would misrepresent it as a tracked deliverable. The delivered state is documented instead in this file and in `services/atlas-cashshop/docs/domain.md` § Surprise.

---

## 2. Decisions already made (do not relitigate)

| Question | Answer | Where |
|---|---|---|
| Reuse `GachaponOpenDone` on v84+? | **No.** That arm calls `CUICashGachapon` — the Cash Gachapon UI, an explicit PRD non-goal. The Surprise box uses the standalone opcode on every version. | design §0, §1.4 |
| Failure vehicle? | The **native FAILED mode** of the same standalone opcode. Not `BuyFailed` — that raises the cash-shop *purchase* notice, a second wrong dialog on top of the gachapon UI. | design §2.3 |
| Error codes on the wire? | **None exist.** The FAILED arm reads nothing after the mode byte. Reasons are logged server-side only. | design §1.3, §2.3 |
| New `box_template_id` column? | **No.** A `cash-surprise` pool's slug **is** the box template id, exactly as an incubator pool's slug is the egg item id. | design §0, §4.1 |
| New roll-by-template endpoint? | **No.** `POST /gachapons/5222000/rewards/select` already exists and already 404s on a missing pool. | design §4.1 |
| Tier weights or flat? | **Flat**, same branch as incubator. Resolves PRD Q2. | design §4.1 |
| Saga or in-service transaction? | **Single in-service transaction**, ordering roll → (consume + grant). The roll mutates nothing, so a failed grant loses nothing. Resolves PRD Q3. | design §2.2, §4.2 |
| v79? | **Route serverbound only** (user decision, 2026-08-09). Overrides the design's `n-a` recommendation. The grant is silent — the item lands in the locker and is visible after reopening the Cash Shop. The v79 clientbound cell stays `n-a` with proof. | user answer; design §5 |
| jms `0xA7` arm-catalog note? | **Superseded, not a correction.** Task 3's image-wide RE pass (beyond design §1.5's UI-segment-bounded search) found opcode `0xA7` on jms_v185 has **two** real senders: `CCashShop::SendChangeMaplePoint` @ `0x4851be` and `CUICashItemGachapon::OnButtonClicked` @ `0xa6e309`, both emitting an 8-byte serial. task-183's original `arm-catalog.md` attribution to `SendChangeMaplePoint` was correct all along — design §1.5's "very likely wrong" was itself wrong, an artifact of the narrower search. Per the user's ruling, the registry now carries two op rows at 167 for this opcode; routing relies on server-side validation to reject a maple-point serial arriving on the Surprise-box handler (and vice versa). | user ruling, 2026-08-09; design §1.5 (superseded) |

PRD Q1 (cash-only) stands. Q4, Q5 dissolved. Q6, Q7 answered by the RE pass.

---

## 3. The per-version table (the single most important artifact)

Every value IDA-verified in `design.md` §1.1. Nothing here came from general MapleStory knowledge.

| Version | IDB session | `CUICashItemGachapon` | Send opcode | Result opcode | SUCCESS mode | FAILED mode | Implement? |
|---|---|---|---|---|---|---|---|
| gms_v48 | `93cc947e` | **absent** | — | `0x101` → 1-byte flag stub | — | — | `n-a` both |
| gms_v61 | `415bf585` | **absent** | — | `0x100` → 1-byte flag stub | — | — | `n-a` both |
| gms_v72 | `c8acae95` | **absent** | — | `0x124` → 1-byte flag stub | — | — | `n-a` both |
| gms_v79 | `1438cecd` | present | `0x9F` | **none** | — | — | serverbound only |
| gms_v83 | `41f13e0d` | present | `0xA1` | `0x14D` | `0xE5` (229) | `0xE4` (228) | both |
| gms_v84 | `5881cf84` | present | `0xA5` | `0x154` (+alias `0x155`) | `0xEE` (238) | `0xED` (237) | both |
| gms_v87 | `d51ecbd3` | present | `0xA9` | `0x15E` (+alias `0x15F`) | `0xF4` (244) | `0xF3` (243) | both |
| gms_v92 | `acdfccff` | present | `0xB6` | `0x180` (+alias `0x181`) | `0xBE` (190) | `0xBD` (189) | both |
| gms_v95 | `79906a1e` | present | `0xB9` | `0x188` (+alias `0x189`) | `0xC1` (193) | `0xC0` (192) | both |
| jms_v185 | `b6864e54` | present | `0xA7` | `0x16D` | `0xEB` (235) | `0xEA` (234) | both |

Resolve IDB sessions from `idb_list` **by binary name** and pass the session as the `database` parameter. `select_instance(port)` is dead.

**jms_v185 `0xA7` is shared by two distinct senders**, confirmed by Task 3's image-wide RE pass (superseding design §1.5's narrower UI-segment-bounded search): `CUICashItemGachapon::OnButtonClicked` @ `0xa6e309` (this feature, `CASH_ITEM_GACHAPON_BUTTON` registry row, ✅) and `CCashShop::SendChangeMaplePoint` @ `0x4851be` (`CASHSHOP_SURPRISE` registry row, ❌ — a distinct, unrelated feature). Both emit an 8-byte serial on the wire; the two are indistinguishable by shape alone. The registry carries both as separate op rows bound to the same numeric opcode (167) on jms_v185 by the user's ruling; the Surprise-box handler and any future maple-point handler must reject a serial that resolves to the wrong domain via server-side validation, since the opcode alone cannot disambiguate.

WZ cross-check (`design.md` §1.6): `GET /api/data/cash/items/5222000` returns 404 on v48/v61/v72 and 200 from v79 onward. The WZ axis and the binary axis agree exactly. The record is `{"slotMax":0,"spec":{}}` — **no WZ spec node**, so the drop table is entirely server-owned.

---

## 4. Wire shapes

**Serverbound** — identical on every version that has it:

```
COutPacket(<send opcode>)
EncodeBuffer(&m_liItemSN, 8)     // little-endian int64 cash SN
```

Guarded client-side by `if (m_nState < 1)`. Only v79 also calls `CWvsContext::SetExclRequestSent`, so on every version in scope the send does **not** arm the excl-request gate — no `EnableActions` is owed.

**Clientbound SUCCESS** — 74 bytes:

```
mode:1  sn:int64  remain:int32  newItem:55  itemId:int32  count:1  jackpot:1
```

`newItem` is the 55-byte `GW_CashItemInfo` blob, **unconditional** — unlike `GachaponOpenDone` there is no `isCashItem` gate. It is byte-identical to `CashInventoryItem.EncodeBytes` (8+4+4+4+4+2+13+8+4+4 = 55). The trailing `itemId`/`count`/`jackpot` are read by the **UI object**, not `CCashShop`; on GMS they are simply left in the buffer when no dialog is open. The server always writes them.

`sn` is the **box's** serial, not the reward's — the client matches it against `m_aCashItemInfo[i].liSN` to find the row to decrement, and removes that row when `remain` is 0.

**Clientbound FAILED** — 1 byte, the mode. The client calls `StringPool::GetString(<fixed id>)` and `CUtilDlg::Notice`, and does **not** re-enable the dialog's Open button. That is native behaviour; the player closes and reopens the dialog. We replicate it rather than inventing a recovery packet.

---

## 5. Key files

**Packet layer**
- `libs/atlas-packet/cash/serverbound/check_wallet.go` — the codec shell to copy
- `libs/atlas-packet/cash/clientbound/shop_inventory.go:16-60` — `CashInventoryItem`, `EncodeBytes`, `decodeCashInventoryItemSkipPadding`
- `libs/atlas-packet/cash/clientbound/vega_scroll.go:168-197` — the DOM-25 `WithResolvedCode` body-provider idiom
- `libs/atlas-packet/resolve.go:15` — `WithResolvedCode`; `ResolveCode` returns a loud **99** on any lookup miss
- `libs/atlas-packet/cash/clientbound/vega_scroll_test.go` — the per-version `operations`-map fixture pattern

**atlas-reward-pools**
- `gachapon/builder.go:9-22,77` — the closed `Kind` union and its validation
- `reward/processor.go:43-99` — `SelectReward`; the `KindIncubator` branch at `:52` is what `cash-surprise` joins
- `reward/processor.go:202-220` — `selectWeightedIndex`, already extracted as a pure function for deterministic testing
- `reward/processor.go:77` — the anonymous empty-pool error that becomes `ErrEmptyPool`
- `item/entity.go` — `Weight uint32 \`gorm:"not null;default:0"\`` is the exact precedent for the `CommodityId` column
- `item/builder.go:53-70` — `Build()` requires a valid tier; weighted kinds pass `"common"` as a placeholder

**atlas-cashshop**
- `cashshop/processor.go:88-207` — `PurchaseAndEmit` / `Purchase`: the transaction + outbox shape to mirror, the Explorer/Cygnus/Legend three-way at `:126-133`, and the `rejectEmit` pattern for firing a no-state-change rejection on the **direct** producer path
- `cashshop/inventory/asset/processor.go:79-121` — `Create`, which already derives `expiration` from the commodity's `period`
- `cashshop/inventory/asset/processor.go:200-231` — `UpdateQuantity` and `Release`; both run on the processor's `db`, so the processor **must** be rebuilt against `tx` inside the transaction closure
- `cashshop/inventory/compartment/model.go:19-50` — `Model.Assets()` is decorated; `Capacity()`; `DefaultCapacity = 55` at `processor.go:21`
- `cashshop/commodity/model.go` — has `ItemId()`, `Count()`, `Period()` but **no** `Id()`; the plan adds one for logging
- `configuration/registry.go:21-51` — the memoized tenant-config cache; `GetTenantConfig` swallows fetch failures and returns a zero `RestModel`, so defaults belong in the accessor

**atlas-channel**
- `socket/handler/cash_shop_check_wallet.go` — the handler shell to copy
- `cashshop/processor.go:96-100` — `RequestPurchase`, the command-producer shape
- `kafka/consumer/cashshop/consumer.go:93-131` — `handleStatusEventPurchase`, including the asset-read-back that builds the `CashInventoryItem` blob
- `main.go:~625` — `produceWriters()`, the writer-name list; `main.go:~925` — the `handlerMap` assignments. **Both** are hand-maintained.
- `incubator/requests.go:26-33` — the `POST /gachapons/{id}/rewards/select` client, nil body (jsonapi.Marshal panics otherwise)

**atlas-ui**
- `src/components/features/reward-pools/PoolFormDialog.tsx` — per-kind form split; `:154` `id: String(values.eggItemId)` is the "slug is the item id" precedent
- `src/components/features/reward-pools/PoolItemDialog.tsx:52` — `const weighted = kind === "incubator"`
- `src/components/features/reward-pools/KindBadge.tsx` — a binary ternary with **no default branch**; a third kind would silently render as "Gachapon"

**Packet audit**
- `docs/packets/audits/VERIFYING_A_PACKET.md` — the single-cell playbook; §"Is this cell `n-a`?" is the evidentiary bar for the seven `n-a` cells
- `docs/packets/audits/STATUS.md:409,486,723` — the three rows in play
- `docs/packets/feature-na-evidence.yaml`, `docs/packets/feature-families.yaml` — required for the v79 family-inconsistent `n-a`

---

## 6. Traps this task is walking through

Each of these has bitten this repo before.

1. **A handler with an empty `validator` is silently dropped** at template load. `LoggedInValidator` for every cash-shop handler.
2. **A writer without `fname`** is rejected by the seed loader.
3. **Template entries must be at sorted `opCode` position** — `tools/template-opcode-order-guard.sh` enforces strictly ascending order. Never append next to a semantically-related entry.
4. **New opcodes in the seed template but not in a live tenant's socket config** → the handler compiles, registers, and never fires. See §8.
5. **A shared Kafka command topic fans every message to every handler.** Without a `c.Type != …` guard, another command's body unmarshals into `OpenSurpriseCommandBody` and produces garbage.
6. **A processor built on `p.db` writes outside the transaction.** Rebuild against `tx` inside the closure, as `PurchaseAndEmit` does.
7. **A rejection that commits no state must fire on the direct producer path**, not the outbox — otherwise it leaks in as though it were part of a committed transaction. `Purchase`'s `rejectEmit` is the precedent.
8. **`tools/lint.sh --check` false-fails without nvm.** `nvm use 22` first. Cross-worktree golangci-lint lock contention is also a thing — if it hangs, check whether another worktree is linting.
9. **`docker buildx bake` is mandatory** for any service whose `go.mod` was touched. `go build` against `go.work` will not catch a missing `COPY libs/...` in the shared Dockerfile.
10. **The mode byte differs on all six implemented versions.** Hard-coding one silently mis-dispatches on the other five. `ResolveCode`'s 99 sentinel is the guard; the plan's `TestCashItemGachaponModeIsNotHardCoded` pins it.
11. **`n-a` needs positive proof**, held to the same bar as a positive verification. A failed name search is absence-of-evidence. The v79 clientbound cell is family-inconsistent (`n-a` while its serverbound sibling is verified) and **requires** a `feature-na-evidence.yaml` entry with non-empty evidence text.

---

## 7. Deliberate deviations from the PRD

Recorded here so a reviewer does not read them as misses.

| PRD | Deviation | Reason |
|---|---|---|
| FR-4.2 "via the existing compartment `Accept` path" | Uses `asset.Create` instead | `compartment.Accept` is the **saga-facing** inbound path and emits `ACCEPTED` events for a saga to correlate. `asset.Create` is the in-service path `Purchase` uses and produces the identical flattened row. FR-4.2's *intent* — a fully-populated flattened cash asset — is met; its named mechanism is not the right one here. |
| FR-6.1/6.2/6.3 `BuyFailed` + distinct numeric error codes | Native FAILED mode, no error code | The wire has no error-code field. `BuyFailed` targets the purchase notice, a wrong second dialog. |
| FR-3.5/3.6 + §6 new column + new endpoint | Neither | The pool slug already is the box template id; the select endpoint already exists. |
| FR-5.1 reuse `GachaponOpenDone` | Standalone opcode on every version | Different UI class, different feature. |
| FR-5.5 v79 out of scope | v79 serverbound **is** routed | User decision. The design's evidence stands; the risk trade-off was the user's to make. |
| FR-4.5 recursion guard | None in code | Honoured by configuration, called out in UI copy and pool docs. A pool that awards a Surprise box creates an endless box; the implementation logs a WARN and does not block it. |

---

## 8. Rollout

**Live tenant socket configuration must be reseeded or PATCHed** with the new handler and writer entries before this feature can be exercised on a running environment. Adding them to the seed templates is necessary but not sufficient: a live tenant's socket config is a stored copy, and an opcode absent from it means the handler is present in code and never fires at runtime. This is a deployment step, not a code change, and it applies to every tenant on v79/v83/v84/v87/v92/v95/jms_v185.

**`GACHAPONS_SERVICE_URL`** (or whatever `requests.RootUrl("GACHAPONS")` reads) must be set for `atlas-cashshop` in the k8s base **and both kustomize overlays**. A value present in base but missing from an overlay is a known silent failure, and a hard-coded base namespace breaks in overlays.

**Reward pools must be configured per tenant** before any box can be opened. An unconfigured tenant gets `POOL_MISSING` on every open — which surfaces to the player as the bare FAILED notice, indistinguishable from any other failure. Seed at least one `cash-surprise` pool with id `5222000`.

---

## 9. Verification results

Recorded by Task 20 (the verification gauntlet). A check that could not be run is recorded as **not run**, never as passed. Pre-existence of any failure is established against the **branch point `1e0a321b8`**, never the branch tip.

Note on Task 20 Step 6's third bullet: `docs/research/missing-features/economy-and-trade.md` is **untracked, local-only** and is deliberately NOT added to this branch (human ruling). The plan brief's instruction to annotate it there does not apply.

| Check | Result | Notes |
|---|---|---|
| `go test -race ./...` — all seven changed Go modules (libs/atlas-packet, libs/atlas-rest, atlas-cashshop, atlas-channel, atlas-reward-pools, atlas-configurations, tools/packet-audit) | **PASS** | Fresh (`-count=1`) runs, every module exit 0, no FAIL/panic |
| `go vet ./...` — same seven modules | **PASS** | exit 0, no output |
| `go build ./...` — same seven modules | **PASS** | exit 0 |
| `docker buildx bake atlas-cashshop atlas-channel atlas-reward-pools atlas-configurations` | **PASS** | exit 0, all four images built. No `go.mod`/`go.sum`/`go.work` changed on this branch (verified with `git diff 1e0a321b8..HEAD --stat -- '**/go.mod' '**/go.sum' go.work go.work.sum` → empty), so CLAUDE.md item 4's mandatory-bake *trigger* does not fire; the four changed Go services were baked anyway per this task's Step 4 |
| `tools/redis-key-guard.sh` | **PASS** | exit 0 |
| `tools/goroutine-guard.sh` | **PASS** | exit 0 |
| `tools/service-registration-guard.sh` | **not run** | Not required: `services.json`, `deploy/k8s`, `docker-bake.hcl`, `go.work`, `tools/db-bootstrap.sh` unchanged vs `1e0a321b8` |
| `tools/lint.sh` (fix mode) then `tools/lint.sh --check` | **PASS (after fix)** | `nvm use 22` first, per the known false-fail trap. Fix mode **did rewrite three files** — `services/atlas-channel/.../socket/handler/cash_item_gachapon_test.go`, `services/atlas-reward-pools/.../item/resource_test.go`, `services/atlas-reward-pools/.../reward/resource_test.go`, all `defer resp.Body.Close()` → `defer func() { _ = resp.Body.Close() }()` (errcheck). Those three test files were **added by this branch** (Tasks 5, 6, 14), so the errcheck failures were **branch regressions, not pre-existing** — an earlier agent called them pre-existing based on a branch-tip run, which is not how pre-existence is established (branch point is `1e0a321b8`). The rewrites are kept and committed. `--check` then exits 0 across every Go module and atlas-ui (5 pre-existing atlas-ui ESLint *warnings*, 0 errors) |
| `tools/template-opcode-order-guard.sh` | **PASS** | exit 0, 22 template arrays checked |
| `tools/template-duplicate-binding-guard.sh` | **PASS** | exit 0, 22 template arrays checked |
| `tools/skill-job-id-guard.sh` | **PASS** | exit 0, 14 divergent consts checked |
| `tools/buff-duration-guard.sh` | **PASS** | exit 0 |
| `tools/template-movement-types-guard.sh` | **PASS** | exit 0, 54 move handlers across 11 templates |
| `go run ./tools/packet-audit matrix --check` (from repo root) | **PASS** | exit 0 |
| `services/atlas-ui`: `npx vitest run` | **PASS** | 243 test files, 1994 tests, all passed |
| `services/atlas-ui`: `npm run build` | **PASS** | exit 0, clean (pre-existing chunk-size warning only) |
| v83 live: open with stack of 3 | **not run** | Requires a live tenant whose socket config carries the new opcodes (context.md §8); not available in this environment |
| v83 live: open the last box (row removed) | **not run** | Same blocker |
| v84+ live: standalone opcode path | **not run** | Same blocker |
| Full locker → error, box intact | **not run** | Same blocker |
| Empty pool → error, box intact | **not run** | Same blocker |
| Forged asset id rejected, no state change | **not run** | Same blocker |
| Reward carries correct commodityId/templateId/quantity/expiration | **not run** | Same blocker |
| Two tenants, same box id, roll from their own pool only | **not run** | Same blocker |
| `gachapon` + `incubator` pools behaviourally unchanged | **not run** (live); unit-level **PASS** | `services/atlas-reward-pools/atlas.com/reward-pools/reward` tests exercise all three kinds and pass under `go test -race`; no live-tenant regression check was run |

Live-tenant rows above are blocked on the rollout step recorded in §8: socket configs must be reseeded/PATCHed with the new handler and writer entries before `OPEN_SURPRISE` (or its serverbound trigger) can fire on a running environment.

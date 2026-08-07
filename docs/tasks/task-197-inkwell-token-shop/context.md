# Inkwell Token Shop — Implementation Context

Companion to [`plan.md`](./plan.md). Read this first if you are picking the task up cold — it is the map of what exists, what was verified, and what will bite you.

Source documents: [`prd.md`](./prd.md) (requirements, FR-* ids) · [`design.md`](./design.md) (decisions D1-D10, risks R1-R7).

---

## 1. What this task actually fixes

Inkwell (NPC `9000069`) sells items for Perfect Pitch (`4310000`) instead of mesos. Today his shop opens **empty** in-game and the Web UI labels all nine commodities "Free". Two independent defects:

- **Data.** Every one of the eleven seeded `shop-9000069.json` files has `mesoPrice: 0, tokenPrice: 0, tokenTemplateId: 0` on all nine commodities. The v83 client decodes each commodity but only inserts it into the buy list under `if (v99 || v100)` (`CShopDlg::SetShopDlg` @ `0x7529ad`, where `v99` = mesoPrice, `v100` = tokenPrice) — nine decoded, nine dropped. The UI's "Free" is a faithful render of the same zeros, not a separate bug.
- **Code.** `shops/processor.go:484` is a landed stub returning `GENERIC_ERROR_WITH_REASON("not implemented")`. `Buy()` implements only the rechargeable and meso paths. Fixing the data alone would make the items visible but unbuyable.

A third, smaller ask rides along: the Web UI's Add/Edit Commodity dialogs get item-search pickers for the two template-id fields.

---

## 2. Repository facts verified for this plan

Everything below was read from source in this worktree, not recalled.

### Seed data

- **All eleven `shop-9000069.json` files are byte-identical** — one md5, `a4d28ed497c213bbc6f24294276b86d3`. So one canonical replacement serves all six survivors.
- Format: 2-space indent, LF, trailing newline, keys alphabetically ordered inside each commodity object (`discountRate, levelLimit, mesoPrice, period, templateId, tokenPrice, tokenTemplateId`).
- The current commodity array is the **exact reverse** of the intended display order.
- The catalog is directory-driven — `ShopSubdomain.Path()` = `"npc-shops/shops"`, `EntityIDPattern()` = `^shop-(\d+)\.json$` (`shops/subdomain.go:29-33`), rooted at `./deploy/seed` (`seed/groups.go:18`). **No manifest, index or count to update** — deleting a file removes the shop.
- Array order is the on-screen order end to end: `ShopList.Encode` iterates the slice (`libs/atlas-packet/npc/clientbound/shop_list.go:50`) → `ShopSubdomain.Build` iterates the JSON array (`shops/subdomain.go:73`) → the client appends in decode order. There is no `sortOrder` column and none is being added.

### `services/atlas-npc-shops` (Go module `atlas-npc`)

Module root: `services/atlas-npc-shops/atlas.com/npc`. **Baseline verified green** before planning — `go test ./...` passes in every package.

| Thing | Where | Shape |
|---|---|---|
| `Buy()` | `shops/processor.go:385-486` | Three ordered branches: rechargeable → `MesoPrice() > 0` → stub |
| Free-slot pattern to mirror | `shops/processor.go:462-467` | `CompartmentByType(it).NextFreeSlot()` before any mutation |
| Slot-based destroy | `compartment/processor.go:35-39` | `RequestDestroyItem(mb)(characterId, inventoryType, slot, quantity)` |
| Destroy command body | `compartment/producer.go:32-44` | `{Slot int16, Quantity uint32}` per message, key `producer.CreateKey(int(characterId))` on `COMMAND_TOPIC_COMPARTMENT` |
| Error providers | `shops/producer.go:34` / `:59` | `errorEventProvider(characterId, errorMsg)` / `reasonErrorEventProvider(…, reason)` |
| Error constants | `kafka/message/shops/kafka.go:47-60` | `ErrorNeedMoreItems = "NEED_MORE_ITEMS"`, `ErrorInventoryFull`, `ErrorGenericError`, … |
| Buffer | `kafka/message/message.go` | `NewBuffer()`, `Put(topic, provider)`, `GetAll() map[string][]kafka.Message`; `Emit` publishes after the closure returns |
| Message serialization | `libs/atlas-kafka/producer/message.go:39-53` | `SingleMessageProvider` → `json.Marshal(value)` into `kafka.Message.Value`, so tests unmarshal JSON |
| Commodity getters | `commodities/model.go` | `TemplateId()`, `MesoPrice()`, `TokenTemplateId()`, `TokenPrice()`, `UnitPrice()`, `SlotMax()` — all `uint32` except `DiscountRate() byte`, `UnitPrice() float64` |

**Builders available (use these; do not write `*_testhelpers.go`):**

- `commodities.NewBuilder()` — `SetId(uuid)` and `SetTemplateId(…)` are **required**; `Build()` returns `(Model, error)`.
- `compartment.NewBuilder(id uuid.UUID, characterId uint32, it inventory.Type, capacity uint32)` → `.SetAssets([]asset.Model)` / `.AddAsset(…)` → `.Build()`.
- `inventory.NewBuilder(characterId)` → `.SetCompartment(compartment.Model)` (keys by the compartment's own type) → `.Build()`.
- `character.NewModelBuilder()` — **no arguments**; `.SetId(…)`, `.SetInventory(…)`, `.SetMeso(…)`, `.Build()`. (`design.md` D7 sketched `character.NewBuilder(...)`, which does not exist.)
- `asset.Model` has **no builder** — construct via the exported production path `asset.Extract(asset.BaseRestModel{Slot, TemplateId, Quantity})`, which returns `(Model, error)`.

**Asset quantity gotcha:** `asset.Model.Quantity()` returns the stored value only when `HasQuantity()` is true, which requires `IsStackable()` — i.e. USE / SETUP / ETC (`asset/model.go:127-140`). Perfect Pitch `4310000` is ETC, so it holds. Equip-id fixtures would silently report quantity 1. The plan's `etcAsset` helper asserts the quantity survived `Extract`, so a future fixture using a non-stackable id fails loudly instead of silently.

**Test package note:** the existing `shops/*_test.go` files are all `package shops_test` (external). The new `token_test.go` must be `package shops` (internal) to reach the unexported `planTokenSpend`, `tokenDraw`, and the `ProcessorImpl` fields `l` / `compP`. Both packages coexist in one directory legally.

**Import-alias note:** inside `package shops`, `processor.go` imports both `"atlas-npc/inventory"` as `inventory2` and `atlas-constants/inventory` as `inventory`. The new test file must use the same aliasing. It also imports `"atlas-npc/kafka/message/shops"` as `shops` — legal even though the file's own package is `shops`, because a package's name is not a binding inside its own files.

### `services/atlas-ui`

| Thing | Where | Note |
|---|---|---|
| Search combobox (add-mode) | `characters/templates/ItemSearchCombobox.tsx` | The `settled {term, page}` comment at :38-49 documents a real prior regression — preserve it verbatim |
| Its test | `characters/templates/__tests__/ItemSearchCombobox.test.tsx` | **The regression harness. Never edit it.** Mocks `@/services/api/items.service` + `@/context/tenant-context` |
| Value-bearing precedent | `characters/templates/MapPicker.tsx` | `value`/`onChange`, current-value label, manual-id row, `unresolved` hint |
| Pool config | `characters/templates/poolSearchConfig.ts` | Exactly **two** importers today: `ItemSearchCombobox.tsx:15`, `presets/EquipmentSection.tsx:13` (type-only) |
| Move destination | `src/lib/items/` | Already exists — holds `taxonomy.ts` |
| Name resolution | `lib/hooks/api/useItemStrings.ts` | `useItemName(itemId: string)`, cache key `itemStringKeys.byId`. **`enabled: !!itemId`** — so `String(0)` = `"0"` would fire a real request; gate with `value > 0 ? String(value) : ""` |
| Dialog | `npc/NpcShopCommodityDialog.tsx` | One `FIELDS` array of 7 numeric inputs; `disabled` special case for `templateId` in edit mode (:79) |
| Dialog's only consumer | `npc/NpcShopCard.tsx:393-408` | Uses conditional spread `{...(editing && { initial: editing.attributes })}` — the `exactOptionalPropertyTypes` idiom |
| Price rendering | `npc/NpcShopCommodityWidget.tsx:88-100` | `formatPrice` gates on `tokenPrice > 0 && tokenTemplateId > 0`, else "Free" |

**Config facts:** `tsconfig.app.json` sets `strict: true` **and `exactOptionalPropertyTypes: true`** (line 27). Vitest picks up `src/**/*.test.{ts,tsx}` with `environment: "jsdom"` and `setupFiles: ["./src/test/setup.ts"]`. `npm run build` is `tsc -b && vite build` — **it is what type-checks the test files**, so `npm run test` alone is not sufficient verification.

---

## 3. Key decisions and why

| Decision | Rationale |
|---|---|
| Delete the five pre-v83 seed files rather than reprice or zero them | A token-priced shop is structurally unrepresentable pre-v83 — the v79 client's `CShopDlg::SetShopDlg` @ `0x6d3459` reads only `mesoPrice` and guards on `if (v98)` alone. Zero-priced rows are today's bug; meso-priced rows would invent a shop that never existed. (D1) |
| Whole-file rewrite of the six survivors | The array is the exact reverse of the target order, so every element moves anyway. Six deterministic writes beat a per-field patch loop across six files, and CLAUDE.md prefers per-file Write/Edit over shell patch loops. (D2) |
| Seed `tokenTemplateId` on all six versions, including JMS | The client only receives it on GMS ≥ 95 (`shop_list.go:58`), but the **server** reads it to decide what to consume and the UI gates its price line on it. A row with `tokenPrice > 0, tokenTemplateId == 0` is invalid data, not a version-appropriate omission. (design §1.1 C-2) |
| Pure `planTokenSpend` + thin `buyWithTokens` | `Buy()` is untestable as-is (HTTP character fetch, no seam). Moving the character to a parameter buys the five required behavioural test cases for free, without widening the public struct with `Fn` seam fields. (D5, D7) |
| One `RequestDestroyItem` per drawn slot, ascending | No aggregated destroy exists — the compartment command contract is slot-based. Ordering is safe: all commands share the `characterId` key on one topic, and `message.Emit` publishes the buffer as a unit. **This resolves PRD open question 2.** (D5) |
| Free-slot probe before any destroy | Correctness, not UX: tokens must never be consumed for an item that cannot be received. Mirrors the meso path exactly. (D6) |
| `uint64` overflow guard on `tokenPrice × quantity` | `quantity` arrives from the wire. Wrapping would produce a small cost for a large purchase — i.e. granting items without charging, which the PRD's NFR forbids. New code gets the check; the meso path's identical shape is not being fixed here. (D6) |
| Extract a headless hook + presentational list, don't duplicate | A standalone copy would fork ~90 lines including the documented `settled {term,page}` regression trap. A `mode` prop on the existing component would force a discriminated-union props type onto a working, tested component. **This resolves PRD open question 1.** (D8) |
| Move `poolSearchConfig.ts` to `src/lib/items/` | Otherwise the items feature would depend on the characters feature. Two import statements change; contents do not. (D8) |
| Edit-mode `Template ID` renders as text, not a disabled picker | A disabled control that opens nothing is worse than a label. Preserves today's non-editable semantics while making the row legible. (D10) |

---

## 4. Traps

1. **`useItemName(String(0))` fires a request for item id 0.** `"0"` is truthy and the hook's `enabled` is `!!itemId`. Always gate: `useItemName(value > 0 ? String(value) : "")`.
2. **`exactOptionalPropertyTypes: true`.** Passing `prop={undefined}` is a type error where the prop is declared `prop?: T`. Use conditional spread — `{...(allowClear ? { leadingRow: … } : {})}`.
3. **`ItemSearchCombobox.test.tsx` is the only proof the extraction is behaviour-preserving.** If it fails, fix the extraction, never the test. Task 9 Step 6 re-verifies it has zero diff against `main`.
4. **`npm run test` alone does not type-check test files.** `npm run build` does. Run both.
5. **`character.NewBuilder(...)` does not exist** — it's `character.NewModelBuilder()` with no args. `design.md` D7's sketch is wrong on this point.
6. **`docs/TODO.md:270` cites a stale line number** (`shops/processor.go:430`). Find both entries by grepping `TokenItem`, not by line number.
7. **Do not touch `services/atlas-channel`.** Its `commodities.Model.Quantity()` hardcoded-`0` TODO is an explicit PRD non-goal and was verified harmless: the v83 client uses the field only to *suppress* the quantity prompt (`sub_4284BE(itemId) || v8[14] > 1` in `CShopDlg::SendBuyRequest`), so `0` lands stackables on the prompt branch bounded by `slotMax` — correct for every commodity Atlas can express today.
8. **`grep -rln '9000069' deploy/seed` will hit non-shop catalogs** (npc definitions, spawns, conversations). Leave them alone — adding or removing the NPC from those is an explicit non-goal.
9. **Re-seeding is destructive.** `runSubdomain` (`libs/atlas-seeder/seed.go`) does `DeleteAllForTenant` then `BulkCreate`, and `ShopSubdomain.DeleteAllForTenant` (`shops/subdomain.go:34-49`) unscoped-deletes **all** commodities and **all** shops for the tenant. This must appear in the PR description, not just here.

---

## 5. Dependency order

Three independent workstreams; within each, tasks are strictly ordered.

```
A (seed):    Task 1 ──▶ Task 2
B (backend): Task 3 ──▶ Task 4
C (frontend):Task 5 ──▶ Task 6 ──▶ Task 7 ──▶ Task 8
                                              └──▶ Task 9 (needs all of A, B, C)
```

A, B and C touch disjoint files and can be executed in any interleaving. Task 9 is the joint gate and must run last.

---

## 6. Verification gates (CLAUDE.md, applied to this change)

| Gate | Required? | Why |
|---|---|---|
| `go test -race ./...` in `services/atlas-npc-shops/atlas.com/npc` | **Yes** | Go code changed |
| `go vet ./...`, `go build ./...` (same dir) | **Yes** | |
| `tools/redis-key-guard.sh` | **Yes** | Runs alongside `go vet` |
| `tools/goroutine-guard.sh` | **Yes** | Runs alongside `go vet`; this change spawns no goroutines, so it should be trivially clean |
| `tools/lint.sh --check` from `<root>` | **Yes** | Both Go and TS changed. Needs nvm 22 for the UI half |
| atlas-ui `npm run build` **and** `npm run test` | **Yes** | Build is what type-checks tests |
| `docker buildx bake atlas-npc-shops` | **No** — unless a `go.mod` gets touched | No new dependency, no new shared lib. Task 9 Step 2 re-checks and escalates if a `go.mod` appears in the diff |
| `tools/service-registration-guard.sh` | **No** | No service added; `services.json`, `deploy/k8s`, `docker-bake.hcl`, `go.work`, `tools/db-bootstrap.sh` all untouched |
| `tools/template-opcode-order-guard.sh` | **No** | No tenant socket-config template changed |
| `tools/skill-job-id-guard.sh` | **No** | No job/skill id comparisons added |
| Code review before PR | **Yes** | `superpowers:requesting-code-review` → plan-adherence + backend-guidelines + frontend-guidelines |

---

## 7. Grounding citations used

Every client-behaviour claim in the plan traces to a decompiled address. Reproduced here so implementation and review do not have to re-derive them; anything new must be cited the same way or marked unverified.

| Claim | Evidence |
|---|---|
| Zero-priced commodities are dropped by the v83 client | `CShopDlg::SetShopDlg` @ `0x7529ad` — insertion guarded by `if ( v99 \|\| v100 )`, `v99` = mesoPrice, `v100` = tokenPrice |
| Pre-v83 has no `tokenPrice` on the wire | v79 IDB `CShopDlg::SetShopDlg` @ `0x6d3459` — reads only `v98 = Decode4` (mesoPrice), guards on `if (v98)` |
| The buy packet's final field is the **meso** price | `CShopDlg::SendBuyRequest` @ `0x7566f4` — ends `COutPacket::Encode4(&v66, v8[6])`, `v8[6]` = `ITEM+24` = mesoPrice |
| The v83 client hardcodes `4310000` for its local pre-check only | `TSecType<long>::GetData(v70 + 12, 0) == &loc_41C3F0`, `0x41C3F0` = `4310000` |
| The client's refusal string is "you need N more" | SP_5433 "YOU NEED %d MORE %s" — hence `NEED_MORE_ITEMS` over `NOT_ENOUGH_MONEY` |
| The v83 client always sends quantity 1 on the token branch | Its `a2` stays at the `*a2 = 1` initialisation; only the meso branch prompts |
| `quantity` on the wire only suppresses the bundle prompt | `sub_4284BE(itemId) \|\| v8[14] > 1` in `CShopDlg::SendBuyRequest`; field read at `ITEM+56` |
| Field version gates | `libs/atlas-packet/npc/clientbound/shop_list.go:52-68` — `discountRate` on `GMS && Major>=87`; `tokenTemplateId` on `GMS && Major>=95`; `tokenPrice`/`period`/`levelLimit` on `!(GMS && Major<83)` |
| The channel writer applies no version logic | `services/atlas-channel/atlas.com/channel/socket/writer/npc_shop.go:19-46` copies all fields unconditionally; the gates live in the codec |

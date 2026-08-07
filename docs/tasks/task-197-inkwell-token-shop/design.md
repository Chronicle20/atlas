# Inkwell Token Shop — Design

Version: v1
Status: Proposed
Created: 2026-08-06
PRD: [`prd.md`](./prd.md)

---

## 1. Scope Recap

Three independent workstreams, joined only by the feature they unblock:

| # | Workstream | Module | Coupling |
|---|---|---|---|
| A | Seed data for NPC `9000069` | `deploy/seed/**` | none (data only) |
| B | Token-purchase branch in `Buy()` | `services/atlas-npc-shops` | none — no API/schema change |
| C | Item-search pickers in the commodity dialogs | `services/atlas-ui` | none — payload-invariant |

They can be built and reviewed in any order. B is the only one that changes runtime
behaviour on the wire-adjacent path; A is the only one that must be re-applied to a live
tenant after merge.

### 1.1 Grounding corrections to the PRD

Two claims in the PRD do not survive contact with the source. Neither changes the plan,
but the design records the corrected facts so implementation is not built on them.

**C-1 — the writer file is `npc_shop.go`, not `shop_list.go`.** The PRD's
"`shop_list.go:55-68`" refers to two different files. The atlas-channel writer that maps
`commodities.Model` → packet struct is
`services/atlas-channel/atlas.com/channel/socket/writer/npc_shop.go:19-46`; it copies
`TokenTemplateId`/`TokenPrice` unconditionally, with no version logic at all. The version
gates live one layer down, in the codec:
`libs/atlas-packet/npc/clientbound/shop_list.go:52-68`.

**C-2 — JMS does *not* receive `tokenTemplateId` on the wire.** The PRD states "the JMS
branch writes the full field set including `tokenTemplateId`". The actual gates in
`shop_list.go` are:

| Field | Gate (`shop_list.go`) | GMS 83/84/87/92 | GMS 95 | JMS 185 |
|---|---|---|---|---|
| `discountRate` | `Region=="GMS" && Major>=87` | 87/92 only | ✅ | ❌ |
| `tokenTemplateId` | `Region=="GMS" && Major>=95` | ❌ | ✅ | ❌ |
| `tokenPrice`, `period`, `levelLimit` | `!(Region=="GMS" && Major<83)` | ✅ | ✅ | ✅ |

So on GMS 83–92 and on JMS the client never learns *which* item the token is — it is
either hardcoded client-side (GMS v83 hardcodes `4310000` at `0x41C3F0`, per the PRD's
`CShopDlg::SetShopDlg` citation) or unverified (JMS 185). This **strengthens** FR-1.2
rather than weakening it: `tokenTemplateId` is seeded for all six versions because the
*server* reads it (FR-2.3) and the UI renders it, not because the client needs it. No
change to `libs/atlas-packet` follows from this, consistent with the PRD non-goal.

**Unverified, deliberately not acted on:** whether the JMS 185 client renders a token
price at all given it receives `tokenPrice` but no `tokenTemplateId`. The JMS gate shape
is inherited from the GMS v83 branch and no JMS IDB evidence was consulted for this task.
Seeding JMS identically is the low-risk choice — a JMS client that ignores the field is
unaffected, and the server-side purchase path works regardless.

---

## 2. Workstream A — Seed Data

### D1 — Delete rather than neutralise the five pre-v83 files

**Decision.** Delete `deploy/seed/gms/{12,48,61,72,79}_1/npc-shops/shops/shop-9000069.json`
outright.

**Alternatives considered.**

1. *Keep the files, price them in mesos.* Rejected — it invents a shop that never existed
   in those clients and creates a second maintenance surface. Inkwell's inventory is
   token-priced by design; a meso Inkwell is a different feature.
2. *Keep the files with zero prices.* Rejected — that is exactly today's bug: nine
   commodities decoded and dropped by `if (v98)` at `CShopDlg::SetShopDlg` @ `0x6d3459`
   (v79 IDB). An entry that can never render is dead data that reads as intentional.
3. *Delete.* Chosen. The seed catalog is directory-driven — `ShopSubdomain.Path()` returns
   `"npc-shops/shops"` and `EntityIDPattern()` is `^shop-(\d+)\.json$`
   (`shops/subdomain.go:29-33`), with the filesystem source rooted at `./deploy/seed`
   (`seed/groups.go:18`). There is no manifest, index, or count to update; removing the
   file removes the shop.

**Consequence for v48/v61.** `shop_list.go:74-90` shows v48 and v61 have materially
different commodity encodings (no trailing `slotMax` on v48; no token/period/levelLimit
below v83). Deleting is also the only option that keeps those columns honest — there is no
correct token-priced payload to emit.

### D2 — Rewrite the six remaining files as whole documents in display order

**Decision.** Replace each of the six `shop-9000069.json` files with a fully rewritten
commodity array in position order 1→9 per the FR-1.3 table, `tokenTemplateId: 4310000` on
every row, `mesoPrice`/`discountRate`/`period`/`levelLimit` left at `0`. All six files
(GMS 83/84/87/92/95, JMS 185) receive byte-identical `commodities` arrays.

**Why whole-file rewrite over surgical edits.** The current array is the exact reverse of
the intended order, so every element moves anyway; a per-field patch loop across six files
is more error-prone than six deterministic writes (and CLAUDE.md prefers per-file
Write/Edit over shell patch loops). The files are ~90 lines each and fully specified by
the FR-1.3 table.

**Ordering is load-bearing, not cosmetic.** `ShopList.Encode` iterates `m.commodities` in
slice order (`shop_list.go:50`); `ShopSubdomain.Build` iterates `jm.Commodities` in JSON
array order (`shops/subdomain.go:73`); the client appends to its buy list in decode order.
The JSON array order is therefore the on-screen order, end to end. There is no `sortOrder`
column and none is being added.

**Grounding note.** The nine template ids and their token prices come from the PRD's
FR-1.3 table, which is the approved specification for this task. Implementation must
reproduce that table verbatim — it must not be "corrected" against any external source.
The nine ids are exactly the nine already present in the file, so a diff of the id set
before/after must be empty; that invariant is checkable and belongs in the plan.

### D3 — Live-tenant repair is out-of-band, by re-seed

Per the PRD's operating decision, no migration and no PATCH. The design's only
contribution is the operator warning, which must survive into the PR description:
`libs/atlas-seeder/seed.go` `runSubdomain` performs `DeleteAllForTenant` then `BulkCreate`,
and `ShopSubdomain.DeleteAllForTenant` (`shops/subdomain.go:34-49`) issues
`db.Unscoped().Where("tenant_id = ?", …).Delete(...)` for **all** commodities and **all**
shops of the tenant. Re-seeding `npc-shops` is a destructive full replace of every shop on
that tenant, not a merge of the one file that changed.

---

## 3. Workstream B — Token Purchasing

### 3.1 Where the branch goes

`Buy()` (`shops/processor.go:385-486`) currently resolves shop → commodity → character,
then runs three ordered branches:

```
1. item.IsRechargeable(itemTemplateId)   → rechargeable path (meso)
2. cm.MesoPrice() > 0                    → meso path
3. (fallthrough)                         → TODO stub
```

The token branch replaces branch 3. Branches 1 and 2 are untouched, per FR-2.1 and the
"meso and rechargeable paths behaviourally unchanged" acceptance criterion.

### D4 — Keep the rechargeable branch first, and document the resulting hole

`item.IsRechargeable` is `IsThrowingStar || IsBullet` (`libs/atlas-constants/item/constants.go:187`),
i.e. `207xxxx` and `233xxxx`. A commodity that is *both* rechargeable *and* token-priced
would be captured by branch 1, compute `totalCost` from `unitPrice × slotMax`, charge
**mesos**, and — if that product is zero — emit `GENERIC_ERROR` with "no price configured".
It would never reach the token branch.

**Decision.** Leave the ordering as-is. Inkwell sells none of these: the nine ids are
`2022503, 2000004, 2022514, 2000005, 3010116, 1122017, 2049000, 2049100, 1003016` — no
`207`/`233` prefix among them, so the hole is unreachable for this task's data. Reordering
the branches would change behaviour on the rechargeable path, which the PRD forbids.

**Recorded as a known limitation** (§7) rather than fixed: a token-priced star/bullet
vendor is not expressible today. Fixing it means teaching branch 1 about tokens, which is
a strictly larger feature (slot-quantity semantics × token cost) and not in scope.

### D5 — Cost and consumption modelled as a pure planner + a thin emitter

**Decision.** Split the token branch into

```go
// pure, package-level, no logger / no buffer / no I/O
func planTokenSpend(as []asset.Model, tokenTemplateId uint32, cost uint32) ([]tokenDraw, uint32)

type tokenDraw struct {
    slot     int16
    quantity uint32
}
```

`planTokenSpend` sorts the matching assets by ascending slot, accumulates until `cost` is
met, and returns the draws plus the total available. The caller compares
`available < cost` for the `NEED_MORE_ITEMS` decision, then replays the draws through
`p.compP.RequestDestroyItem(mb)`.

**Alternatives.**

1. *`FindFirstByItemId` + single destroy* (`compartment/model.go:76`). Rejected outright —
   FR-2.4.1. Perfect Pitch stacks at `slotMax`; a 100-token purchase can straddle stacks,
   and a single-slot destroy would either under-charge or emit a destroy for more than the
   slot holds. This is the specific defect the PRD calls out.
2. *One aggregated "destroy N of template T" command.* Rejected — no such affordance
   exists. `compartment.Processor.RequestDestroyItem` is slot-based
   (`compartment/processor.go:35-39`) and `RequestDestroyAssetCommandProvider`
   (`compartment/producer.go:32-44`) encodes `{Slot, Quantity}` per message. Inventing a
   template-based destroy command means changing the compartment service's command
   contract — far outside this task.
3. *Inline loop inside `Buy()`.* Rejected on testability grounds (D7): `Buy()` cannot be
   invoked in a unit test without stubbing the character REST client, whereas a pure
   planner is table-testable at zero cost and covers the multi-slot, insufficient, and
   `quantity > 1` acceptance criteria directly.

**This resolves PRD open question 2.** One `RequestDestroyItem` call per drawn slot, in
ascending slot order. Ordering and grouping are safe: every compartment command is keyed
`producer.CreateKey(int(characterId))` on the single `EnvCommandTopic`
(`compartment/producer.go:33`), so all destroys and the subsequent create land on one
partition in emission order. `message.Emit` publishes the whole buffer after the closure
returns (`kafka/message/message.go:44-58`), so the batch is all-or-nothing at the
publish boundary.

### D6 — Guard order inside the token branch

```
a. cm.TokenPrice() == 0                     → GENERIC_ERROR   (misconfigured: no price at all)
b. cm.TokenTemplateId() == 0                → GENERIC_ERROR   (misconfigured: price, no currency)
c. inventory.TypeFromItemId(token)  !ok     → GENERIC_ERROR   (unresolvable currency)
d. inventory.TypeFromItemId(purchased) !ok  → GENERIC_ERROR   (unresolvable destination)
e. cost = TokenPrice() * quantity; overflow → GENERIC_ERROR
f. available < cost                         → NEED_MORE_ITEMS (nothing consumed)
g. destination NextFreeSlot() err           → INVENTORY_FULL  (nothing consumed)
h. emit destroys (ascending slot), then create
```

Rationale for the two orderings that matter:

- **(f) before (g)** — the balance check is the cheaper and more informative refusal, and
  it is what the client's own local pre-check tests. Either order is safe (neither
  consumes), so this is a UX ordering, not a correctness one.
- **(g) before (h)** — this is the correctness one, and it mirrors the meso path exactly
  (`processor.go:462-467`): the free-slot probe must precede any consumption so tokens are
  never destroyed for an item that cannot be received (PRD NFR "no unpriced purchases",
  inverted).

`discountPrice` is not referenced anywhere in the branch (FR-2.2). The acceptance criterion
"provably does not influence the amount charged" is satisfied structurally — the parameter
is simply absent from the token code path — and asserted by a test that varies
`discountPrice` across calls and asserts identical destroy quantities.

**Overflow (e).** `cost := cm.TokenPrice() * quantity` is `uint32 × uint32`. `quantity`
arrives from the wire. A hostile or buggy value can wrap and produce a small cost for a
large purchase. The meso path has the same shape and is not being fixed here, but the
token path is new code and gets the check: compute in `uint64` and reject if the product
exceeds `math.MaxUint32`. Cheap, and it closes a "grants an item without charging"
avenue that the PRD's NFR explicitly forbids.

### D7 — Testability: extract the branch behind a character-model seam

**The problem.** `ProcessorImpl.charP` / `compP` are unexported and set in `NewProcessor`
(`processor.go:69-96`). `Buy()` calls `p.charP.GetById(p.charP.InventoryDecorator)(…)`,
which is an HTTP request (`character/processor.go:44-47`). There is no existing seam for
it, and none of the seven existing `*_test.go` files in the service exercises `Buy()` at
all. The PRD requires five behavioural test cases against this path.

**Decision.** Extract the branch as a method taking the already-resolved character:

```go
func (p *ProcessorImpl) buyWithTokens(mb *message.Buffer) func(c character.Model, cm commodities.Model, itemTemplateId uint32, quantity uint32) error
```

`Buy()` calls it after its existing character fetch. Tests construct a `ProcessorImpl`
directly (in-package `package shops` test file), supplying a real
`compartment.NewProcessor()` — which is a pure `mb.Put` wrapper with no dependencies — and
a real `message.NewBuffer()`, then assert on `buf.GetAll()`. No HTTP, no database, no
mocks, no test-only constructor.

**Alternatives.**

1. *Exported `Fn` seam fields* (`BuyFn`, or `CharP`/`CompP` fields), following the existing
   `GetByNpcIdFn` / `GetAllShopsFn` / `RechargeableConsumablesDecoratorFn` precedent
   (`processor.go:74-76`). Rejected as the primary mechanism: it widens the public struct
   surface permanently to buy one test, and the same coverage is available for free by
   moving the branch behind a parameter. (The precedent exists because those methods have
   no such natural seam.)
2. *`httptest` stub for the character service.* Rejected — heavier, slower, and it tests
   the REST client rather than the purchase logic.
3. *Interface-ify and hand-roll mocks.* Rejected — `character.Processor` and
   `compartment.Processor` are already interfaces, but mocking `compP` buys nothing when
   the real one is a three-line `mb.Put`, and mocking `charP` is unnecessary once the
   character model is a parameter.

**Test fixtures without a builder.** `asset.Model` has no `Builder` (the package is
`model.go` + `rest.go` only). Rather than add one, tests construct assets through the
exported production path `asset.Extract(asset.BaseRestModel{Slot: …, TemplateId: …,
Quantity: …})` (`asset/rest.go:111`), then fold them into
`compartment.NewBuilder(...).SetAssets(...)` → `inventory.NewBuilder(...).SetCompartment(...)`
→ `character.NewBuilder(...)...Build().SetInventory(inv)`. This uses the project's Builder
pattern where builders exist and an existing exported constructor where one does not — no
`*_testhelpers.go`, per CLAUDE.md.

*(If, during implementation, `asset.Extract` proves awkward for a `Quantity` to survive —
`Model.Quantity()` returns the stored value only when `HasQuantity()`, which for ETC ids
is true via `IsStackable()` (`asset/model.go:127-140`) — the fallback is to add a proper
`asset.NewBuilder`/`Clone` pair matching the package convention. Perfect Pitch `4310000`
is ETC, so the primary path is expected to hold.)*

### 3.2 Error taxonomy

| Condition | Emitted | Consumes? |
|---|---|---|
| `TokenPrice()==0 && MesoPrice()==0` | `GENERIC_ERROR` | no |
| `TokenPrice()>0 && TokenTemplateId()==0` | `GENERIC_ERROR` | no |
| currency or destination id unresolvable to an inventory type | `GENERIC_ERROR` | no |
| `TokenPrice() * quantity` overflows `uint32` | `GENERIC_ERROR` | no |
| held tokens `< cost` | `NEED_MORE_ITEMS` | no |
| destination compartment full | `INVENTORY_FULL` | no |
| success | *(no error event)* | yes |

`NEED_MORE_ITEMS` over `NOT_ENOUGH_MONEY` per FR-2.4.2 — it matches the client's own
SP_5433 "YOU NEED %d MORE %s" refusal, and `NOT_ENOUGH_MONEY` is the meso-path signal.

Logging follows the surrounding convention: `Errorf` on each refusal including
`characterId`, `itemTemplateId`, and slot where meaningful; `Debugf` on success naming the
token cost and the token template id.

---

## 4. Workstream C — Web UI Item Pickers

### 4.1 What already exists

| Piece | File | Shape |
|---|---|---|
| Search popover, add-mode | `characters/templates/ItemSearchCombobox.tsx` | action control — `onAdd(id)`, holds no value |
| Value-bearing picker precedent | `characters/templates/MapPicker.tsx` | `value` / `onChange`, resolves current value to a label, manual-id escape hatch, `unresolved` hint |
| Pool filter config | `characters/templates/poolSearchConfig.ts` | `POOL_SEARCH_CONFIGS.items` = `{}` (all compartments) |
| Search call | `services/api/items.service.ts` `searchItems` | `/api/data/item-strings` + filters, paged |
| Single-id name | `lib/hooks/api/useItemStrings.ts` `useItemName` | same endpoint, shared `itemStringKeys.byId` cache key |

`MapPicker` is the closer structural precedent for what FR-3 asks for; `ItemSearchCombobox`
owns the item-search behaviour that must be reused. The new component is the intersection.

### D8 — Extract the search internals into a headless hook; two thin shells consume it

**Decision (resolves PRD open question 1).** Extract, don't duplicate, and extract the
*logic*, not the markup:

```
src/lib/items/poolSearchConfig.ts          ← moved from characters/templates/ (2 importers)
src/components/features/items/item-search/
    useItemSearch.ts                       ← debounce, settled {term,page}, useQuery,
                                             client-side subcategory filter, manualId,
                                             hasMore, loadMore  (headless)
    ItemSearchResults.tsx                  ← the <ul role="listbox"> body: rows, icon,
                                             "Use id N", loading / error / empty states
    ItemPicker.tsx                         ← NEW value-bearing field (value/onChange)
characters/templates/ItemSearchCombobox.tsx ← keeps its props, becomes trigger + popover
                                              around the two extracted pieces
```

`ItemSearchCombobox`'s public props (`poolKey`, `existingIds`, `onAdd`, `triggerLabel`,
`debounceMs`) and its rendered DOM are unchanged, so
`__tests__/ItemSearchCombobox.test.tsx` passes untouched — that file is the regression
harness for the extraction and must not be edited.

**Alternatives.**

1. *Standalone `ItemPicker` modeled on `MapPicker`, duplicating the list body.* Simplest
   diff, but forks ~90 lines of subtle behaviour: the `settled {term, page}` atomicity
   comment in `ItemSearchCombobox.tsx:38-49` documents a real prior regression (page
   advancing independently of the term), and a copy would re-open that trap in a second
   place. Rejected.
2. *Add a `mode: "add" | "value"` prop to `ItemSearchCombobox`.* One component, two
   behaviours, `onAdd` and `onChange`/`value` mutually exclusive in the type — the props
   interface becomes a discriminated union to express "these three props only exist in
   that mode". Rejected: it makes the existing, tested component more complex to serve a
   new caller.
3. *Extract hook + list (chosen).* Each unit answers the three isolation questions
   cleanly — `useItemSearch` does query state and owns no DOM; `ItemSearchResults` renders
   rows and owns no fetching; the two shells own only their trigger and their value
   semantics.

**Why `poolSearchConfig.ts` moves.** It is currently imported by
`characters/templates/ItemSearchCombobox.tsx` and `characters/presets/EquipmentSection.tsx`
(type-only). Once `features/items/` consumes it too, leaving it under
`characters/templates/` would make the items feature depend on the characters feature.
`src/lib/items/` already exists (`@/lib/items/taxonomy`, imported by `items.service.ts`),
so that is its natural home. Two import statements change; the file's contents do not.

**Where `ItemPicker` lives.** `features/items/item-search/`, not `features/npc/` — it is a
generic control and the npc dialog is its first, not its only, consumer.

### D9 — `ItemPicker` semantics

```ts
interface ItemPickerProps {
  value: number;                // 0 = unset
  onChange: (id: number) => void;
  poolKey?: SearchPoolKey;      // default "items" (all compartments)
  placeholder?: string;         // label rendered when value === 0
  allowClear?: boolean;         // renders a "None" row that calls onChange(0)
  disabled?: boolean;
  debounceMs?: number;          // test hook, mirrors the two existing components
}
```

Modeled directly on `MapPicker`'s trigger: a full-width outline `Button` whose label is
the resolved item, opening the same popover.

- **Label resolution** uses the existing `useItemName(String(value))` hook — *not* a
  bespoke `useQuery` on `itemsService.getItemName`. Both hit `/api/data/item-strings/{id}`,
  but the hook shares `itemStringKeys.byId`, so a name already fetched by the commodity
  grid, the item browser, or a previous dialog open is served from cache with no request.
  The PRD names `itemsService.getItemName`; this is the same endpoint behind the
  project's existing caching seam, and is a strict improvement over calling the service
  directly.
- **Label states** (FR-3.5), following `MapPicker.tsx:38-43`: `value === 0` → the
  `placeholder`; loading → `Item {value}`; resolved → `{name} · {value}`; error →
  `Item {value}` plus the muted "couldn't resolve" hint. Never a blocking spinner, never a
  submit-blocking state.
- **`allowClear`** exists solely to keep FR-3.4's "`tokenTemplateId: 0` must remain
  expressible" true. Typing `0` into the search box also works — `manualId` accepts it —
  but an explicit "None" row is the discoverable path.
- **`existingIds`** is not a prop. It is an add-to-a-pool concept; a single-valued field
  has nothing to exclude.

### D10 — `NpcShopCommodityDialog` restructure

Today the dialog maps one `FIELDS` array of seven numeric inputs, with a `disabled` special
case for `templateId` in edit mode (`NpcShopCommodityDialog.tsx:74-101`). The two template
fields break out; the array shrinks to the five numeric fields
(`mesoPrice`, `discountRate`, `tokenPrice`, `period`, `levelLimit`) and keeps its loop.

```
mode="create":  Template ID        → <ItemPicker value={form.templateId} onChange=… />
                Token Template ID  → <ItemPicker … allowClear placeholder="None" />
mode="edit":    Template ID        → read-only resolved name (useItemName), no picker
                Token Template ID  → <ItemPicker … allowClear placeholder="None" />
```

Edit-mode `Template ID` renders as text, not a disabled picker — a disabled control that
opens nothing is worse than a label. This preserves the existing `disabled` semantics
(FR-3.3) while making the row legible, and it means the edit dialog cannot repoint a
commodity at a different item, exactly as today.

**Payload invariance (FR-3.4)** falls out of the structure: `form` remains
`CommodityAttributes` with numeric ids, the reset-on-open logic
(`NpcShopCommodityDialog.tsx:53-58`) is untouched, and `onSubmit(form)` is unchanged. The
pickers only write numbers into the same state the inputs wrote. This is asserted by test,
not just by inspection — see §5.

**Layout.** The existing `grid-cols-4` label/control rows are kept; `ItemPicker` occupies
`col-span-3` like the inputs it replaces, so the dialog's shape does not change.

---

## 5. Testing Strategy

### Backend (`services/atlas-npc-shops`)

| Level | Target | Cases |
|---|---|---|
| Pure | `planTokenSpend` | exact single slot; cost spanning 2 and 3 slots; cost > total held (returns short); zero-quantity slots skipped; non-matching template ids ignored; ascending-slot draw order regardless of input order |
| Branch | `buyWithTokens` via a directly-constructed `ProcessorImpl` + real `message.Buffer` | sufficient tokens → N destroys + 1 create, correct slots/quantities, no error event; insufficient → exactly one `NEED_MORE_ITEMS` message and **zero** compartment commands; destination full → `INVENTORY_FULL`, zero compartment commands; `tokenPrice>0 && tokenTemplateId==0` → `GENERIC_ERROR`; `quantity=3` → cost multiplied; `discountPrice` varied → identical destroy quantities |
| Regression | existing `processor_test.go` suite | unchanged, must still pass (meso/rechargeable parity) |

Assertions read `buf.GetAll()` (`kafka/message/message.go:34`) and inspect the messages on
`EnvCommandTopic` / `EnvStatusEventTopic`. The "consumes nothing" criteria are asserted as
*absence* of any message on the compartment topic — the strongest available form of "no
partial purchase".

### Frontend (`services/atlas-ui`)

| File | Asserts |
|---|---|
| `features/items/item-search/__tests__/ItemPicker.test.tsx` (new) | renders placeholder at `value=0`; renders resolved name once `useItemName` settles; falls back to `Item {id}` on lookup error; picking a row calls `onChange(id)` and closes; "Use id N" commits a raw id; "None" calls `onChange(0)` |
| `features/npc/__tests__/NpcShopCommodityDialog.test.tsx` (new) | create mode submits the **exact** `CommodityAttributes` object for a picker-chosen id, byte-comparable to what the numeric input produced; edit mode renders the template name as text with no picker for `templateId`; edit mode's token picker is interactive; `tokenTemplateId: 0` round-trips |
| `characters/templates/__tests__/ItemSearchCombobox.test.tsx` (existing) | **unmodified** — the extraction's regression harness |

Mocking follows the established local convention: `vi.mock` on
`@/services/api/items.service` and `@/context/tenant-context`, per
`ItemSearchCombobox.test.tsx:12-24`, plus `@/lib/hooks/api/useItemStrings` for name
resolution, per the `useItemNames` mocking pattern in
`AppearanceBrowserDialog.test.tsx:25-27`.

### Seed data

No unit test — the files are data. The plan's verification step is a mechanical diff
check: for each of the six files, the multiset of `templateId` values before and after is
identical, the array order matches the FR-1.3 table, and every row has
`tokenTemplateId == 4310000`. Plus `git status` showing exactly five deletions and six
modifications under `deploy/seed`, and nothing else.

---

## 6. Data Flow — Successful Token Purchase

```
client  ──BUY(slot, itemTemplateId, quantity, discountPrice=0)──▶ atlas-channel
atlas-channel ──command──▶ atlas-npc-shops  Buy(mb)(characterId)(…)
    registry lookup            → shopId
    cp.GetByNpcId(shopId)      → commodity row (authoritative prices)
    charP.GetById(+inventory)  → character + compartments
    ── branch: !rechargeable, MesoPrice()==0, TokenPrice()>0 ──▶ buyWithTokens
        inventory.TypeFromItemId(4310000)      → ETC
        planTokenSpend(etc.Assets(), 4310000, tokenPrice*quantity)
                                               → [{slot 3, 60}, {slot 7, 40}], available=115
        dest = TypeFromItemId(itemTemplateId); dest.NextFreeSlot()  → ok
        mb.Put(COMPARTMENT_COMMAND, destroy ETC slot 3 × 60)
        mb.Put(COMPARTMENT_COMMAND, destroy ETC slot 7 × 40)
        mb.Put(COMPARTMENT_COMMAND, create  itemTemplateId × quantity)
message.Emit ──▶ all three published, same key (characterId), same partition, in order
```

Failure at any guard emits exactly one message on `EVENT_TOPIC_NPC_SHOP_STATUS` and
nothing on the compartment topic.

---

## 7. Risks & Known Limitations

| # | Item | Disposition |
|---|---|---|
| R1 | Token-priced rechargeable (star/bullet) commodities are captured by the rechargeable branch and mispriced/refused (D4) | Documented, unreachable for Inkwell's nine ids. Not fixed — fixing it changes the rechargeable path, which the PRD forbids. |
| R2 | The free-slot pre-check is conservative when the token compartment *is* the destination compartment: fully draining a token stack frees a slot the pre-check did not count, so a full compartment can refuse a purchase that would have fit | Accepted — identical to the meso path's behaviour. Unreachable for Inkwell (token is ETC; the nine items are equip/use/setup). |
| R3 | The free-slot check does not consider merging into an existing partial stack of the purchased item | Pre-existing on the meso path; parity preserved, not extended. |
| R4 | Seed correction does not repair the live tenant; the re-seed is a destructive full replace of the tenant's shops | Explicit operating decision (PRD §6). Must appear in the PR description, not only in this doc. |
| R5 | JMS 185 token rendering is unverified (§1.1 C-2) | Seeded identically; server-side purchase is version-independent, so the worst case is a cosmetic gap on JMS. |
| R6 | The UI extraction touches a component with a documented prior regression (`settled {term,page}` atomicity) | Mitigated by the untouched existing test file acting as the regression harness. |
| R7 | `docs/TODO.md:270` cites a stale line number (`shops/processor.go:430`) | Both entries are deleted (FR-2.6), so the staleness resolves itself. Grep for `TokenItem`/`Token Item` rather than trusting the PRD's line numbers, which will have shifted. |

---

## 8. Verification Plan

Run from the worktree root:

1. `go test -race ./...` in `services/atlas-npc-shops` — clean.
2. `go vet ./...` in `services/atlas-npc-shops` — clean.
3. `go build ./...` in `services/atlas-npc-shops` — clean.
4. `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh` — clean.
5. `tools/lint.sh --check` — clean (needs nvm 22 on PATH for the atlas-ui half).
6. atlas-ui: `npm run test` **and** `npm run build` — the build is what type-checks the
   test files, so vitest alone is not sufficient verification.
7. `docker buildx bake atlas-npc-shops` — **not required**: no `go.mod` in this service is
   being touched (no new dependency, no new shared lib). If the implementation does end up
   editing `services/atlas-npc-shops/atlas.com/npc/go.mod`, the bake becomes mandatory.
8. Seed diff check per §5.
9. `grep -rn "TokenItem\|not implemented" services/atlas-npc-shops docs/TODO.md` — no hits
   in `Buy()` or `TODO.md`.
10. Code review (`superpowers:requesting-code-review` → plan-adherence +
    backend-guidelines + frontend-guidelines) before opening the PR.

Manual post-merge, on the live tenant: re-seed `npc-shops`, confirm nine items in FR-1.3
order with Perfect Pitch prices, and confirm a purchase debits the correct token count.

---

## 9. Resolved Open Questions

**PRD Q1 — picker extraction shape.** Extract a headless `useItemSearch` hook plus a
presentational `ItemSearchResults`; `ItemSearchCombobox` and the new `ItemPicker` become
thin shells over both. `ItemSearchCombobox`'s props, DOM, and test file are unchanged.
`poolSearchConfig.ts` moves to `src/lib/items/` so the items feature does not depend on
the characters feature. (D8)

**PRD Q2 — multi-slot consumption granularity.** One `RequestDestroyItem` per drawn slot,
ascending slot order. No batching affordance exists — the compartment command contract is
slot-based (`compartment/producer.go:32-44`) — and none is being invented. Ordering is
safe because all commands share the `characterId` key on one topic, and `message.Emit`
publishes the buffer as a unit. (D5)

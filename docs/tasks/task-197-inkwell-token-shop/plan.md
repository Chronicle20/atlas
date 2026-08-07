# Inkwell Token Shop Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make NPC `9000069` (Inkwell) a working Perfect Pitch token vendor — correct the seed data, implement generic token purchasing in `atlas-npc-shops`, and give the Web UI commodity dialogs item-search pickers.

**Architecture:** Three independent workstreams joined only by the feature they unblock. (A) Seed data — delete the five pre-v83 `shop-9000069.json` files, rewrite the six remaining ones in display order with token prices. (B) Backend — replace the `// TODO: implement TokenItem purchasing.` stub in `shops/processor.go` `Buy()` with a token branch built as a pure `planTokenSpend` planner plus a thin `buyWithTokens` emitter, so multi-slot consumption is table-testable without HTTP or a database. (C) Frontend — extract the item-search logic from `ItemSearchCombobox` into a headless `useItemSearch` hook and a presentational `ItemSearchResults`, add a value-bearing `ItemPicker` over both, and consume it from `NpcShopCommodityDialog` without changing any request payload.

**Tech Stack:** Go 1.24 (module `atlas-npc` at `services/atlas-npc-shops/atlas.com/npc`), Kafka via `libs/atlas-kafka` + the in-package `kafka/message.Buffer`, GORM (untouched — no schema change), JSON seed files under `deploy/seed/`, React 19 + TypeScript + TanStack React Query + shadcn/ui + Vitest in `services/atlas-ui`.

## Global Constraints

- **Source of truth for the commodity table is `prd.md` FR-1.3.** Reproduce it verbatim. Do not "correct" any template id or token price against WZ data, an external wiki, or memory.
- **The nine template ids are exactly the nine already seeded.** No additions, no removals. The multiset of `templateId` values before and after each file rewrite must be identical.
- **No change to `libs/atlas-packet`.** The v83 wire encoding is already correct and verified.
- **No change to `services/atlas-channel`.** The `commodities.Model.Quantity()` hardcoded-`0` TODO is an explicit non-goal (PRD §2) and stays.
- **No schema change, no migration.** `commodities.Entity` already carries `TokenTemplateId` and `TokenPrice`.
- **No change to the meso or rechargeable branches of `Buy()`.** They keep their current order and behaviour; the existing test suite must still pass unmodified.
- **The client-supplied `discountPrice` must not influence the token cost.** Charge `cm.TokenPrice() * quantity` from the commodity row.
- **The token item is `cm.TokenTemplateId()`, never a hardcoded `4310000`.** The constant appears only in seed JSON and test fixtures.
- **Nothing is consumed on a refusal.** Every failing guard returns before any `RequestDestroyItem`.
- **`services/atlas-ui/tsconfig.app.json` sets `strict: true` and `exactOptionalPropertyTypes: true`.** Never pass an explicit `undefined` to an optional prop — use conditional spread (`{...(x ? { prop: x } : {})}`), the pattern already used at `NpcShopCard.tsx:406`.
- **`services/atlas-ui/src/components/features/characters/templates/__tests__/ItemSearchCombobox.test.tsx` must not be edited.** It is the regression harness for the Workstream C extraction.
- **atlas-ui verification requires `npm run build`, not just `npm run test`** — `tsc -b` is what type-checks the test files. Both must be run. `npm` needs nvm 22 on PATH.
- **No literal home/absolute paths** (`/home/<user>/...`) in any committed file.
- **Every command below runs from the task worktree**, `.worktrees/task-197-inkwell-token-shop/`, referred to as `<root>`. Never `cd` to the main checkout.

## Deviations from `design.md` (deliberate, with reasons)

These are the only places this plan departs from the design doc. Each is a correction or a refinement found while reading the actual source; implementers should follow the plan, and reviewers should not flag them as drift.

1. **`planTokenSpend` returns `uint64` available, not `uint32`.** Design D5 wrote `([]tokenDraw, uint32)`. Summing quantities across slots in `uint32` can itself overflow; returning `uint64` makes the `available < cost` comparison exact with no saturation logic. Signature: `func planTokenSpend(as []asset.Model, tokenTemplateId uint32, cost uint32) ([]tokenDraw, uint64)`.
2. **The character builder is `character.NewModelBuilder()`, not `character.NewBuilder(...)`.** Design D7's test-fixture sketch names a constructor that does not exist. The real one is at `character/model.go:323` and takes no arguments.
3. **No test varies `discountPrice`.** Design §D6 proposed one. `buyWithTokens` has no `discountPrice` parameter at all, so the acceptance criterion "provably does not influence the amount charged" is satisfied *structurally* — a test that varies an argument the function cannot receive would be theatre. Task 9 verifies it with a grep instead.
4. **`NpcShopCommodityDialog` keeps ONE `FIELDS` array with a `kind` discriminator** rather than splitting into a five-entry numeric array plus two hoisted rows (design D10). Splitting would reorder the visible form fields (`templateId, tokenTemplateId, mesoPrice, …` instead of today's `templateId, mesoPrice, discountRate, tokenTemplateId, …`). Field order is user-facing and neither the PRD nor the design sanctions changing it.
5. **`ItemPicker` gains an `id?: string` prop** not listed in design D9. The dialog associates each control with a `<Label htmlFor=…>`; without an `id` on the trigger button that association breaks.
6. **`useItemName` is called with `""` when the value is `0`.** `useItemName(String(0))` would be `useItemName("0")`, which is truthy and would fire a request for item id 0. Gate it: `useItemName(value > 0 ? String(value) : "")`.

---

## File Structure

**Workstream A — seed data (`deploy/seed/`)**

| Path | Change |
|---|---|
| `deploy/seed/gms/{12,48,61,72,79}_1/npc-shops/shops/shop-9000069.json` | Delete (5 files) |
| `deploy/seed/gms/{83,84,87,92,95}_1/npc-shops/shops/shop-9000069.json` | Rewrite (5 files) |
| `deploy/seed/jms/185_1/npc-shops/shops/shop-9000069.json` | Rewrite (1 file) |

All eleven files are currently byte-identical (verified: single md5 `a4d28ed497c213bbc6f24294276b86d3`). The six survivors receive one byte-identical replacement.

**Workstream B — `services/atlas-npc-shops/atlas.com/npc/`**

| Path | Responsibility |
|---|---|
| `shops/token.go` (new) | `tokenDraw`, `planTokenSpend` (pure), `(*ProcessorImpl).buyWithTokens` |
| `shops/token_test.go` (new, `package shops`) | Table tests for `planTokenSpend`; buffer-assertion tests for `buyWithTokens` |
| `shops/processor.go` (modify, ~line 484) | Replace the stub with a call to `buyWithTokens` |
| `docs/TODO.md` (modify, lines 15 and 270) | Delete the two satisfied entries |

**Workstream C — `services/atlas-ui/src/`**

| Path | Responsibility |
|---|---|
| `lib/items/poolSearchConfig.ts` (moved) | Pool→filter config; moved out of `components/features/characters/templates/` so the items feature does not depend on the characters feature |
| `components/features/items/item-search/useItemSearch.ts` (new) | Headless: debounce, atomic `{term,page}`, `useQuery`, client-side subcategory filter, `manualId`, `hasMore`/`loadMore`. Owns no DOM. |
| `components/features/items/item-search/ItemSearchResults.tsx` (new) | The `<ul role="listbox">`: rows, icon, "Use id N", loading/error/empty. Owns no fetching. |
| `components/features/items/item-search/ItemPicker.tsx` (new) | Value-bearing field: `value`/`onChange`, resolved-name trigger, optional "None" row |
| `components/features/items/item-search/__tests__/ItemPicker.test.tsx` (new) | ItemPicker behaviour |
| `components/features/characters/templates/ItemSearchCombobox.tsx` (modify) | Becomes a thin trigger+popover shell; **props and rendered DOM unchanged** |
| `components/features/characters/presets/EquipmentSection.tsx` (modify, line 13) | Import path only |
| `components/features/npc/NpcShopCommodityDialog.tsx` (modify) | Two template-id fields become pickers |
| `components/features/npc/__tests__/NpcShopCommodityDialog.test.tsx` (new) | Payload invariance, edit-mode read-only name, `tokenTemplateId: 0` round-trip |

---

## Task 1: Delete the five pre-v83 Inkwell shop seed files

Inkwell does not exist pre-v83 and a token-priced shop is structurally unrepresentable there — the v79 client's `CShopDlg::SetShopDlg` @ `0x6d3459` reads only `mesoPrice` and guards insertion with `if (v98)` alone. The seed catalog is directory-driven (`shops/subdomain.go:29-33` — `Path()` = `"npc-shops/shops"`, `EntityIDPattern()` = `^shop-(\d+)\.json$`, filesystem root `./deploy/seed` at `seed/groups.go:18`), so there is no manifest, index, or count to update. Deleting the file removes the shop.

**Files:**
- Delete: `deploy/seed/gms/12_1/npc-shops/shops/shop-9000069.json`
- Delete: `deploy/seed/gms/48_1/npc-shops/shops/shop-9000069.json`
- Delete: `deploy/seed/gms/61_1/npc-shops/shops/shop-9000069.json`
- Delete: `deploy/seed/gms/72_1/npc-shops/shops/shop-9000069.json`
- Delete: `deploy/seed/gms/79_1/npc-shops/shops/shop-9000069.json`

**Interfaces:**
- Consumes: nothing.
- Produces: nothing consumed by later tasks. Task 9 re-checks the resulting `git status` shape.

- [ ] **Step 1: Confirm exactly eleven files exist before touching anything**

Run from `<root>`:

```bash
ls deploy/seed/*/*/npc-shops/shops/shop-9000069.json | wc -l
```

Expected: `11`

- [ ] **Step 2: Delete the five pre-v83 files**

Run from `<root>`:

```bash
git rm deploy/seed/gms/12_1/npc-shops/shops/shop-9000069.json \
       deploy/seed/gms/48_1/npc-shops/shops/shop-9000069.json \
       deploy/seed/gms/61_1/npc-shops/shops/shop-9000069.json \
       deploy/seed/gms/72_1/npc-shops/shops/shop-9000069.json \
       deploy/seed/gms/79_1/npc-shops/shops/shop-9000069.json
```

- [ ] **Step 3: Verify six remain, and see what else mentions NPC 9000069**

Run from `<root>`:

```bash
ls deploy/seed/*/*/npc-shops/shops/shop-9000069.json
grep -rln '9000069' deploy/seed | sort
```

Expected from the first command — exactly these six lines:

```
deploy/seed/gms/83_1/npc-shops/shops/shop-9000069.json
deploy/seed/gms/84_1/npc-shops/shops/shop-9000069.json
deploy/seed/gms/87_1/npc-shops/shops/shop-9000069.json
deploy/seed/gms/92_1/npc-shops/shops/shop-9000069.json
deploy/seed/gms/95_1/npc-shops/shops/shop-9000069.json
deploy/seed/jms/185_1/npc-shops/shops/shop-9000069.json
```

Expected from the second: the six paths above, plus any NPC-definition / spawn / conversation seed files that happen to mention `9000069`. **Do not modify those** — adding or removing NPC `9000069` from non-shop catalogs is an explicit PRD non-goal. Just note what they are.

- [ ] **Step 4: Verify the working tree shows five deletions and nothing else**

Run from `<root>`:

```bash
git status --porcelain
```

Expected: exactly five lines, each starting with `D ` and naming one of the five deleted paths.

- [ ] **Step 5: Commit**

```bash
git commit -m "fix(task-197): remove Inkwell shop seed from pre-v83 versions

Inkwell (NPC 9000069) does not exist before GMS v83, and the pre-v83
commodity wire format has no tokenPrice field at all, so a token-priced
shop is structurally unrepresentable there. The seeded rows could never
render."
```

---

## Task 2: Rewrite the six remaining shop seed files in display order with token prices

The commodity array order *is* the wire order *is* the client's display order: `ShopList.Encode` iterates `m.commodities` in slice order (`libs/atlas-packet/npc/clientbound/shop_list.go:50`), `ShopSubdomain.Build` iterates `jm.Commodities` in JSON array order (`shops/subdomain.go:73`), and the client appends to its buy list in decode order. The current array is the exact reverse of the intended display positions, so every element moves — a whole-file rewrite is both simpler and less error-prone than a patch loop.

Zero-priced commodities are silently dropped by the client: `CShopDlg::SetShopDlg` @ `0x7529ad` (v83 IDB) inserts a decoded commodity into the buy list only under `if ( v99 || v100 )`, where `v99` is `mesoPrice` and `v100` is `tokenPrice`. That is the "empty shop" symptom.

`tokenTemplateId` is seeded on all six versions even though it only reaches the client on GMS ≥ 95 (`shop_list.go:58`), because the **server** reads it to decide what to consume (Task 4) and the Web UI gates its price line on it.

**Files:**
- Modify (full rewrite): `deploy/seed/gms/83_1/npc-shops/shops/shop-9000069.json`
- Modify (full rewrite): `deploy/seed/gms/84_1/npc-shops/shops/shop-9000069.json`
- Modify (full rewrite): `deploy/seed/gms/87_1/npc-shops/shops/shop-9000069.json`
- Modify (full rewrite): `deploy/seed/gms/92_1/npc-shops/shops/shop-9000069.json`
- Modify (full rewrite): `deploy/seed/gms/95_1/npc-shops/shops/shop-9000069.json`
- Modify (full rewrite): `deploy/seed/jms/185_1/npc-shops/shops/shop-9000069.json`

**Interfaces:**
- Consumes: nothing.
- Produces: the seeded `tokenTemplateId: 4310000` that Task 4's server-side consumption reads at runtime. No compile-time coupling.

- [ ] **Step 1: Record the "before" template-id multiset**

Run from `<root>`:

```bash
grep -o '"templateId": [0-9]*' deploy/seed/gms/83_1/npc-shops/shops/shop-9000069.json | sort > /tmp/inkwell-before.txt
cat /tmp/inkwell-before.txt
```

Expected (nine lines, sorted):

```
"templateId": 1003016
"templateId": 1122017
"templateId": 2000004
"templateId": 2000005
"templateId": 2022503
"templateId": 2022514
"templateId": 2049000
"templateId": 2049100
"templateId": 3010116
```

- [ ] **Step 2: Write the canonical file content to all six paths**

Write this **exact** content — 2-space indent, LF line endings, alphabetically-ordered keys within each commodity object, trailing newline — to each of the six paths. Use the Write tool once per file (six separate writes). Do not use a shell patch loop.

The array is in display position order 1→9 per PRD FR-1.3. `mesoPrice`, `discountRate`, `period` and `levelLimit` stay at `0`.

```json
{
  "data": {
    "attributes": {
      "commodities": [
        {
          "discountRate": 0,
          "levelLimit": 0,
          "mesoPrice": 0,
          "period": 0,
          "templateId": 2022503,
          "tokenPrice": 5,
          "tokenTemplateId": 4310000
        },
        {
          "discountRate": 0,
          "levelLimit": 0,
          "mesoPrice": 0,
          "period": 0,
          "templateId": 2000004,
          "tokenPrice": 5,
          "tokenTemplateId": 4310000
        },
        {
          "discountRate": 0,
          "levelLimit": 0,
          "mesoPrice": 0,
          "period": 0,
          "templateId": 2022514,
          "tokenPrice": 10,
          "tokenTemplateId": 4310000
        },
        {
          "discountRate": 0,
          "levelLimit": 0,
          "mesoPrice": 0,
          "period": 0,
          "templateId": 2000005,
          "tokenPrice": 10,
          "tokenTemplateId": 4310000
        },
        {
          "discountRate": 0,
          "levelLimit": 0,
          "mesoPrice": 0,
          "period": 0,
          "templateId": 3010116,
          "tokenPrice": 25,
          "tokenTemplateId": 4310000
        },
        {
          "discountRate": 0,
          "levelLimit": 0,
          "mesoPrice": 0,
          "period": 0,
          "templateId": 1122017,
          "tokenPrice": 30,
          "tokenTemplateId": 4310000
        },
        {
          "discountRate": 0,
          "levelLimit": 0,
          "mesoPrice": 0,
          "period": 0,
          "templateId": 2049000,
          "tokenPrice": 45,
          "tokenTemplateId": 4310000
        },
        {
          "discountRate": 0,
          "levelLimit": 0,
          "mesoPrice": 0,
          "period": 0,
          "templateId": 2049100,
          "tokenPrice": 70,
          "tokenTemplateId": 4310000
        },
        {
          "discountRate": 0,
          "levelLimit": 0,
          "mesoPrice": 0,
          "period": 0,
          "templateId": 1003016,
          "tokenPrice": 100,
          "tokenTemplateId": 4310000
        }
      ],
      "npcId": 9000069,
      "recharger": false
    },
    "id": "9000069",
    "type": "npc-shop"
  }
}
```

- [ ] **Step 3: Verify all six files are byte-identical to each other**

Run from `<root>`:

```bash
md5sum deploy/seed/gms/83_1/npc-shops/shops/shop-9000069.json \
       deploy/seed/gms/84_1/npc-shops/shops/shop-9000069.json \
       deploy/seed/gms/87_1/npc-shops/shops/shop-9000069.json \
       deploy/seed/gms/92_1/npc-shops/shops/shop-9000069.json \
       deploy/seed/gms/95_1/npc-shops/shops/shop-9000069.json \
       deploy/seed/jms/185_1/npc-shops/shops/shop-9000069.json | awk '{print $1}' | sort -u | wc -l
```

Expected: `1` (one distinct checksum across all six).

- [ ] **Step 4: Verify the template-id set is unchanged and every row is token-priced**

Run from `<root>`:

```bash
grep -o '"templateId": [0-9]*' deploy/seed/gms/83_1/npc-shops/shops/shop-9000069.json | sort > /tmp/inkwell-after.txt
diff /tmp/inkwell-before.txt /tmp/inkwell-after.txt && echo "TEMPLATE-ID SET UNCHANGED"
grep -c '"tokenTemplateId": 4310000' deploy/seed/gms/83_1/npc-shops/shops/shop-9000069.json
grep -c '"tokenTemplateId": 0' deploy/seed/gms/83_1/npc-shops/shops/shop-9000069.json || true
```

Expected: `TEMPLATE-ID SET UNCHANGED`, then `9`, then `0` (the last `grep -c` prints `0` and exits 1; the `|| true` keeps the shell happy).

- [ ] **Step 5: Verify the wire/display order matches FR-1.3 exactly**

Run from `<root>`:

```bash
grep -o '"templateId": [0-9]*\|"tokenPrice": [0-9]*' deploy/seed/gms/83_1/npc-shops/shops/shop-9000069.json | paste - -
```

Expected, in this order (JSON key order puts `templateId` before `tokenPrice` within each object, so each line pairs one commodity):

```
"templateId": 2022503	"tokenPrice": 5
"templateId": 2000004	"tokenPrice": 5
"templateId": 2022514	"tokenPrice": 10
"templateId": 2000005	"tokenPrice": 10
"templateId": 3010116	"tokenPrice": 25
"templateId": 1122017	"tokenPrice": 30
"templateId": 2049000	"tokenPrice": 45
"templateId": 2049100	"tokenPrice": 70
"templateId": 1003016	"tokenPrice": 100
```

- [ ] **Step 6: Verify every file is valid JSON**

Run from `<root>`:

```bash
for f in deploy/seed/gms/83_1 deploy/seed/gms/84_1 deploy/seed/gms/87_1 deploy/seed/gms/92_1 deploy/seed/gms/95_1 deploy/seed/jms/185_1; do
  python3 -m json.tool "$f/npc-shops/shops/shop-9000069.json" > /dev/null && echo "OK $f"
done
```

Expected: six `OK <path>` lines.

- [ ] **Step 7: Verify nothing outside these six files changed**

Run from `<root>`:

```bash
git status --porcelain
```

Expected: exactly six lines, each starting with ` M` and naming one of the six paths (the five deletions from Task 1 are already committed).

- [ ] **Step 8: Commit**

```bash
git add deploy/seed/gms/83_1/npc-shops/shops/shop-9000069.json \
        deploy/seed/gms/84_1/npc-shops/shops/shop-9000069.json \
        deploy/seed/gms/87_1/npc-shops/shops/shop-9000069.json \
        deploy/seed/gms/92_1/npc-shops/shops/shop-9000069.json \
        deploy/seed/gms/95_1/npc-shops/shops/shop-9000069.json \
        deploy/seed/jms/185_1/npc-shops/shops/shop-9000069.json
git commit -m "fix(task-197): price Inkwell commodities in Perfect Pitch, in display order

All nine commodities were seeded with mesoPrice/tokenPrice/tokenTemplateId
of 0, which the client silently drops (CShopDlg::SetShopDlg @ 0x7529ad
inserts only under 'if (v99 || v100)'), producing an empty shop dialog.
Set tokenTemplateId to 4310000 (Perfect Pitch) and tokenPrice per the PRD
FR-1.3 table, and reorder the array 1->9 since JSON array order is the
client's display order end to end."
```

---

## Task 3: Add the pure `planTokenSpend` planner

The token cost can straddle multiple stacks — Perfect Pitch stacks at `slotMax`, so a 100-token purchase may need several slots. `compartment.Model.FindFirstByItemId` (`compartment/model.go:76`) returns one asset and is therefore insufficient. There is no aggregated "destroy N of template T" command: `compartment.Processor.RequestDestroyItem` is slot-based (`compartment/processor.go:35-39`) and `RequestDestroyAssetCommandProvider` encodes `{Slot, Quantity}` per message (`compartment/producer.go:32-44`).

This task builds only the pure planner — no logger, no buffer, no I/O — so the multi-slot, insufficient-balance and quantity-multiplier cases are table-testable at zero cost.

**Files:**
- Create: `services/atlas-npc-shops/atlas.com/npc/shops/token.go`
- Test: `services/atlas-npc-shops/atlas.com/npc/shops/token_test.go`

**Interfaces:**
- Consumes: `asset.Model` (`atlas-npc/asset`) — `Slot() int16`, `TemplateId() uint32`, `Quantity() uint32`.
- Produces, for Task 4:
  - `type tokenDraw struct { slot int16; quantity uint32 }`
  - `func planTokenSpend(as []asset.Model, tokenTemplateId uint32, cost uint32) ([]tokenDraw, uint64)` — returns the draws (ascending slot order, accumulating until `cost` is met) and the **total available** across all matching slots. Callers must compare `available < uint64(cost)` themselves; when short, the returned draws are a partial plan and must not be executed.

- [ ] **Step 1: Write the failing test scaffold**

Create `services/atlas-npc-shops/atlas.com/npc/shops/token_test.go`. Note the package: `package shops` (in-package), because `planTokenSpend` and `tokenDraw` are unexported. The existing `shops/*_test.go` files use `package shops_test`; both may coexist in one directory.

```go
package shops

import (
	"testing"

	"atlas-npc/asset"
)

const perfectPitch = uint32(4310000)

// etcAsset builds an asset.Model through the exported production path.
// asset.Model has no Builder (the package is model.go + rest.go only), and
// CLAUDE.md forbids *_testhelpers.go with test-only constructors.
// Model.Quantity() returns the stored value only when HasQuantity() is true,
// which holds for ETC ids like 4310000 via IsStackable() (asset/model.go:127-140).
func etcAsset(t *testing.T, slot int16, templateId uint32, quantity uint32) asset.Model {
	t.Helper()
	a, err := asset.Extract(asset.BaseRestModel{
		Slot:       slot,
		TemplateId: templateId,
		Quantity:   quantity,
	})
	if err != nil {
		t.Fatalf("failed to build asset: %v", err)
	}
	if a.Quantity() != quantity {
		t.Fatalf("asset quantity did not survive Extract: got %d want %d", a.Quantity(), quantity)
	}
	return a
}

func TestPlanTokenSpend(t *testing.T) {
	draws, available := planTokenSpend([]asset.Model{etcAsset(t, 3, perfectPitch, 60)}, perfectPitch, 60)
	if available != 60 || len(draws) != 1 {
		t.Fatalf("got draws %v available %d", draws, available)
	}
}
```

That minimal body exists only to prove the file compiles against the not-yet-written symbols. Replace it wholesale in Step 3.

- [ ] **Step 2: Run the test to verify it fails**

Run from `<root>/services/atlas-npc-shops/atlas.com/npc`:

```bash
go test ./shops/ -run TestPlanTokenSpend -v
```

Expected: FAIL — a compile error naming `undefined: planTokenSpend`.

- [ ] **Step 3: Write the real table test**

Replace the whole `TestPlanTokenSpend` function (keep the imports, `perfectPitch`, and `etcAsset` helper from Step 1):

```go
func TestPlanTokenSpend(t *testing.T) {
	tests := []struct {
		name          string
		assets        func(t *testing.T) []asset.Model
		cost          uint32
		wantDraws     []tokenDraw
		wantAvailable uint64
	}{
		{
			name: "exact single slot",
			assets: func(t *testing.T) []asset.Model {
				return []asset.Model{etcAsset(t, 3, perfectPitch, 60)}
			},
			cost:          60,
			wantDraws:     []tokenDraw{{slot: 3, quantity: 60}},
			wantAvailable: 60,
		},
		{
			name: "cost spans two slots and the second is drawn partially",
			assets: func(t *testing.T) []asset.Model {
				return []asset.Model{
					etcAsset(t, 3, perfectPitch, 60),
					etcAsset(t, 7, perfectPitch, 55),
				}
			},
			cost:          100,
			wantDraws:     []tokenDraw{{slot: 3, quantity: 60}, {slot: 7, quantity: 40}},
			wantAvailable: 115,
		},
		{
			name: "cost spans three slots",
			assets: func(t *testing.T) []asset.Model {
				return []asset.Model{
					etcAsset(t, 1, perfectPitch, 40),
					etcAsset(t, 2, perfectPitch, 40),
					etcAsset(t, 3, perfectPitch, 40),
				}
			},
			cost: 100,
			wantDraws: []tokenDraw{
				{slot: 1, quantity: 40},
				{slot: 2, quantity: 40},
				{slot: 3, quantity: 20},
			},
			wantAvailable: 120,
		},
		{
			name: "cost exceeds total held returns a short plan and the true total",
			assets: func(t *testing.T) []asset.Model {
				return []asset.Model{
					etcAsset(t, 2, perfectPitch, 10),
					etcAsset(t, 5, perfectPitch, 15),
				}
			},
			cost:          100,
			wantDraws:     []tokenDraw{{slot: 2, quantity: 10}, {slot: 5, quantity: 15}},
			wantAvailable: 25,
		},
		{
			name: "zero-quantity slots are skipped",
			assets: func(t *testing.T) []asset.Model {
				return []asset.Model{
					etcAsset(t, 1, perfectPitch, 0),
					etcAsset(t, 4, perfectPitch, 30),
				}
			},
			cost:          20,
			wantDraws:     []tokenDraw{{slot: 4, quantity: 20}},
			wantAvailable: 30,
		},
		{
			name: "non-matching template ids are ignored",
			assets: func(t *testing.T) []asset.Model {
				return []asset.Model{
					etcAsset(t, 1, 4000000, 999),
					etcAsset(t, 2, perfectPitch, 12),
				}
			},
			cost:          12,
			wantDraws:     []tokenDraw{{slot: 2, quantity: 12}},
			wantAvailable: 12,
		},
		{
			name: "draws are ascending by slot regardless of input order",
			assets: func(t *testing.T) []asset.Model {
				return []asset.Model{
					etcAsset(t, 9, perfectPitch, 50),
					etcAsset(t, 2, perfectPitch, 50),
					etcAsset(t, 5, perfectPitch, 50),
				}
			},
			cost: 110,
			wantDraws: []tokenDraw{
				{slot: 2, quantity: 50},
				{slot: 5, quantity: 50},
				{slot: 9, quantity: 10},
			},
			wantAvailable: 150,
		},
		{
			name: "zero cost draws nothing but still reports what is held",
			assets: func(t *testing.T) []asset.Model {
				return []asset.Model{etcAsset(t, 1, perfectPitch, 7)}
			},
			cost:          0,
			wantDraws:     []tokenDraw{},
			wantAvailable: 7,
		},
		{
			name: "empty compartment",
			assets: func(t *testing.T) []asset.Model {
				return []asset.Model{}
			},
			cost:          5,
			wantDraws:     []tokenDraw{},
			wantAvailable: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			draws, available := planTokenSpend(tt.assets(t), perfectPitch, tt.cost)

			if available != tt.wantAvailable {
				t.Errorf("available: got %d want %d", available, tt.wantAvailable)
			}
			if len(draws) != len(tt.wantDraws) {
				t.Fatalf("draws: got %d entries %v, want %d entries %v",
					len(draws), draws, len(tt.wantDraws), tt.wantDraws)
			}
			for i := range draws {
				if draws[i] != tt.wantDraws[i] {
					t.Errorf("draws[%d]: got %+v want %+v", i, draws[i], tt.wantDraws[i])
				}
			}
		})
	}
}
```

- [ ] **Step 4: Run the test to verify it still fails for the right reason**

Run from `<root>/services/atlas-npc-shops/atlas.com/npc`:

```bash
go test ./shops/ -run TestPlanTokenSpend -v
```

Expected: FAIL — still `undefined: tokenDraw` / `undefined: planTokenSpend`.

- [ ] **Step 5: Write the implementation**

Create `services/atlas-npc-shops/atlas.com/npc/shops/token.go`:

```go
package shops

import (
	"sort"

	"atlas-npc/asset"
)

// tokenDraw is one slot-scoped withdrawal from a token stack. The compartment
// command contract is slot-based (compartment/producer.go:32-44), so a spend
// that straddles stacks becomes one draw — and one DESTROY command — per slot.
type tokenDraw struct {
	slot     int16
	quantity uint32
}

// planTokenSpend computes how to withdraw cost units of tokenTemplateId from
// as, drawing from the lowest slot first, and reports the total quantity held
// across every matching slot.
//
// The returned plan is only valid to execute when available >= cost; when the
// character is short, the draws describe everything they hold and the caller
// must refuse instead of executing them. available is uint64 because summing
// uint32 stack quantities can itself overflow uint32.
func planTokenSpend(as []asset.Model, tokenTemplateId uint32, cost uint32) ([]tokenDraw, uint64) {
	matching := make([]asset.Model, 0, len(as))
	for _, a := range as {
		if a.TemplateId() != tokenTemplateId || a.Quantity() == 0 {
			continue
		}
		matching = append(matching, a)
	}
	sort.Slice(matching, func(i, j int) bool {
		return matching[i].Slot() < matching[j].Slot()
	})

	var available uint64
	for _, a := range matching {
		available += uint64(a.Quantity())
	}

	draws := make([]tokenDraw, 0, len(matching))
	remaining := cost
	for _, a := range matching {
		if remaining == 0 {
			break
		}
		take := a.Quantity()
		if take > remaining {
			take = remaining
		}
		draws = append(draws, tokenDraw{slot: a.Slot(), quantity: take})
		remaining -= take
	}
	return draws, available
}
```

- [ ] **Step 6: Run the test to verify it passes**

Run from `<root>/services/atlas-npc-shops/atlas.com/npc`:

```bash
go test ./shops/ -run TestPlanTokenSpend -v
```

Expected: PASS — all nine subtests.

- [ ] **Step 7: Run the full service suite and vet**

Run from `<root>/services/atlas-npc-shops/atlas.com/npc`:

```bash
go test -race ./...
go vet ./...
```

Expected: all packages `ok` or `[no test files]`; `go vet` silent.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-npc-shops/atlas.com/npc/shops/token.go \
        services/atlas-npc-shops/atlas.com/npc/shops/token_test.go
git commit -m "feat(task-197): add planTokenSpend multi-slot token withdrawal planner"
```

---

## Task 4: Implement the token branch in `Buy()` and close the TODOs

`Buy()` (`shops/processor.go:385-486`) resolves shop → commodity → character, then runs three ordered branches: rechargeable (meso), `MesoPrice() > 0` (meso), and a fallthrough that is currently the stub. This task replaces branch 3. Branches 1 and 2 are untouched.

Guard order inside the new branch is load-bearing: the free-slot probe must precede any consumption so tokens are never destroyed for an item that cannot be received — mirroring the meso path at `processor.go:462-467`.

`buyWithTokens` takes the already-resolved `character.Model` so it is testable. `Buy()` cannot be unit-tested as-is: `p.charP.GetById(p.charP.InventoryDecorator)(…)` is an HTTP request (`character/processor.go:44-47`) with no existing seam, and none of the service's seven `*_test.go` files exercises `Buy()`.

`buyWithTokens` deliberately has **no `discountPrice` parameter**. The v83 client sends the *meso* price in the buy packet's final field (`CShopDlg::SendBuyRequest` @ `0x7566f4` ends with `COutPacket::Encode4(&v66, v8[6])` where `v8[6]` is `ITEM+24` = mesoPrice), which is `0` for a token item. FR-2.2's "provably does not influence the amount charged" is satisfied by the parameter's structural absence.

**Files:**
- Modify: `services/atlas-npc-shops/atlas.com/npc/shops/token.go` (append `buyWithTokens`)
- Modify: `services/atlas-npc-shops/atlas.com/npc/shops/processor.go` (replace the stub, ~line 484)
- Modify: `services/atlas-npc-shops/atlas.com/npc/shops/token_test.go` (append `buyWithTokens` tests)
- Modify: `docs/TODO.md` (delete lines 15 and 270)

**Interfaces:**
- Consumes: `planTokenSpend`, `tokenDraw` (Task 3); `errorEventProvider(characterId uint32, errorMsg string)` (`shops/producer.go:34`); `p.compP.RequestDestroyItem(mb)(characterId uint32, inventoryType inventory.Type, slot int16, quantity uint32) error` and `p.compP.RequestCreateItem(mb)(characterId uint32, templateId uint32, quantity uint32) error` (`compartment/processor.go:14-16`).
- Produces:
  - `func (p *ProcessorImpl) buyWithTokens(mb *message.Buffer) func(c character.Model, cm commodities.Model, itemTemplateId uint32, quantity uint32) error`

- [ ] **Step 1: Write the failing tests**

Append to `services/atlas-npc-shops/atlas.com/npc/shops/token_test.go`. Extend the file's import block to:

```go
import (
	"encoding/json"
	"io"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"

	"atlas-npc/asset"
	"atlas-npc/character"
	"atlas-npc/commodities"
	"atlas-npc/compartment"
	inventory2 "atlas-npc/inventory"
	"atlas-npc/kafka/message"
	compartmentMessage "atlas-npc/kafka/message/compartment"
	"atlas-npc/kafka/message/shops"
)
```

`inventory2` aliases the service's own `atlas-npc/inventory` package while `inventory` is the shared `atlas-constants` one — the same aliasing `processor.go:13,29` uses. The file's own package is `shops`, which does not conflict with importing `atlas-npc/kafka/message/shops` as `shops` (a package's own name is not a binding inside its files — `processor.go` already relies on this).

Then append:

```go
const (
	testCharacterId = uint32(1234)
	// 2022503 is a USE item, so the destination compartment is TypeValueUse.
	testItemId = uint32(2022503)
)

func testProcessor(t *testing.T) *ProcessorImpl {
	t.Helper()
	l := logrus.New()
	l.SetOutput(io.Discard)
	return &ProcessorImpl{
		l:     l,
		compP: compartment.NewProcessor(),
	}
}

// testCharacter builds a character holding etcAssets in its ETC compartment
// and useAssets in a USE compartment of the given capacity.
func testCharacter(t *testing.T, etcAssets []asset.Model, useCapacity uint32, useAssets []asset.Model) character.Model {
	t.Helper()
	etcComp := compartment.NewBuilder(uuid.New(), testCharacterId, inventory.TypeValueETC, 24).
		SetAssets(etcAssets).
		Build()
	useComp := compartment.NewBuilder(uuid.New(), testCharacterId, inventory.TypeValueUse, useCapacity).
		SetAssets(useAssets).
		Build()
	inv := inventory2.NewBuilder(testCharacterId).
		SetCompartment(etcComp).
		SetCompartment(useComp).
		Build()
	return character.NewModelBuilder().
		SetId(testCharacterId).
		SetInventory(inv).
		Build()
}

func testCommodity(t *testing.T, tokenTemplateId uint32, tokenPrice uint32) commodities.Model {
	t.Helper()
	cm, err := commodities.NewBuilder().
		SetId(uuid.New()).
		SetNpcId(9000069).
		SetTemplateId(testItemId).
		SetTokenTemplateId(tokenTemplateId).
		SetTokenPrice(tokenPrice).
		Build()
	if err != nil {
		t.Fatalf("failed to build commodity: %v", err)
	}
	return cm
}

func decodeDestroy(t *testing.T, raw []byte) compartmentMessage.Command[compartmentMessage.DestroyCommandBody] {
	t.Helper()
	var c compartmentMessage.Command[compartmentMessage.DestroyCommandBody]
	if err := json.Unmarshal(raw, &c); err != nil {
		t.Fatalf("failed to decode destroy command: %v", err)
	}
	return c
}

func decodeCreate(t *testing.T, raw []byte) compartmentMessage.Command[compartmentMessage.CreateAssetCommandBody] {
	t.Helper()
	var c compartmentMessage.Command[compartmentMessage.CreateAssetCommandBody]
	if err := json.Unmarshal(raw, &c); err != nil {
		t.Fatalf("failed to decode create command: %v", err)
	}
	return c
}

func decodeStatusError(t *testing.T, raw []byte) shops.StatusEvent[shops.StatusEventErrorBody] {
	t.Helper()
	var e shops.StatusEvent[shops.StatusEventErrorBody]
	if err := json.Unmarshal(raw, &e); err != nil {
		t.Fatalf("failed to decode status event: %v", err)
	}
	return e
}

func TestBuyWithTokensSufficientBalanceSpansSlots(t *testing.T) {
	p := testProcessor(t)
	buf := message.NewBuffer()
	c := testCharacter(t,
		[]asset.Model{
			etcAsset(t, 7, perfectPitch, 55),
			etcAsset(t, 3, perfectPitch, 60),
		},
		24, nil)
	cm := testCommodity(t, perfectPitch, 100)

	if err := p.buyWithTokens(buf)(c, cm, testItemId, 1); err != nil {
		t.Fatalf("buyWithTokens returned an error: %v", err)
	}

	all := buf.GetAll()
	if got := len(all[shops.EnvStatusEventTopic]); got != 0 {
		t.Errorf("expected no status events on success, got %d", got)
	}

	cmds := all[compartmentMessage.EnvCommandTopic]
	if len(cmds) != 3 {
		t.Fatalf("expected 2 destroys + 1 create, got %d commands", len(cmds))
	}

	d0 := decodeDestroy(t, cmds[0].Value)
	if d0.Type != compartmentMessage.CommandDestroy ||
		d0.CharacterId != testCharacterId ||
		d0.InventoryType != byte(inventory.TypeValueETC) ||
		d0.Body.Slot != 3 || d0.Body.Quantity != 60 {
		t.Errorf("destroy[0]: got %+v", d0)
	}

	d1 := decodeDestroy(t, cmds[1].Value)
	if d1.Body.Slot != 7 || d1.Body.Quantity != 40 {
		t.Errorf("destroy[1]: got slot %d quantity %d, want slot 7 quantity 40",
			d1.Body.Slot, d1.Body.Quantity)
	}

	cr := decodeCreate(t, cmds[2].Value)
	if cr.Type != compartmentMessage.CommandCreateAsset ||
		cr.InventoryType != byte(inventory.TypeValueUse) ||
		cr.Body.TemplateId != testItemId || cr.Body.Quantity != 1 {
		t.Errorf("create: got %+v", cr)
	}
}

func TestBuyWithTokensQuantityMultipliesCost(t *testing.T) {
	p := testProcessor(t)
	buf := message.NewBuffer()
	c := testCharacter(t, []asset.Model{etcAsset(t, 1, perfectPitch, 100)}, 24, nil)
	cm := testCommodity(t, perfectPitch, 5)

	if err := p.buyWithTokens(buf)(c, cm, testItemId, 3); err != nil {
		t.Fatalf("buyWithTokens returned an error: %v", err)
	}

	cmds := buf.GetAll()[compartmentMessage.EnvCommandTopic]
	if len(cmds) != 2 {
		t.Fatalf("expected 1 destroy + 1 create, got %d commands", len(cmds))
	}
	if d := decodeDestroy(t, cmds[0].Value); d.Body.Quantity != 15 {
		t.Errorf("destroy quantity: got %d want 15 (tokenPrice 5 x quantity 3)", d.Body.Quantity)
	}
	if cr := decodeCreate(t, cmds[1].Value); cr.Body.Quantity != 3 {
		t.Errorf("create quantity: got %d want 3", cr.Body.Quantity)
	}
}

func TestBuyWithTokensRefusals(t *testing.T) {
	noAssets := func(t *testing.T) []asset.Model { return nil }
	plentyOfTokens := func(t *testing.T) []asset.Model {
		return []asset.Model{etcAsset(t, 1, perfectPitch, 500)}
	}

	tests := []struct {
		name            string
		etcAssets       func(t *testing.T) []asset.Model
		useCapacity     uint32
		useAssets       func(t *testing.T) []asset.Model
		tokenTemplateId uint32
		tokenPrice      uint32
		quantity        uint32
		wantError       string
	}{
		{
			name:            "insufficient tokens",
			etcAssets:       func(t *testing.T) []asset.Model { return []asset.Model{etcAsset(t, 1, perfectPitch, 4)} },
			useCapacity:     24,
			useAssets:       noAssets,
			tokenTemplateId: perfectPitch,
			tokenPrice:      5,
			quantity:        1,
			wantError:       shops.ErrorNeedMoreItems,
		},
		{
			name:            "holds none of the token item at all",
			etcAssets:       noAssets,
			useCapacity:     24,
			useAssets:       noAssets,
			tokenTemplateId: perfectPitch,
			tokenPrice:      5,
			quantity:        1,
			wantError:       shops.ErrorNeedMoreItems,
		},
		{
			name:        "destination compartment is full",
			etcAssets:   plentyOfTokens,
			useCapacity: 1,
			useAssets: func(t *testing.T) []asset.Model {
				return []asset.Model{etcAsset(t, 1, testItemId, 1)}
			},
			tokenTemplateId: perfectPitch,
			tokenPrice:      5,
			quantity:        1,
			wantError:       shops.ErrorInventoryFull,
		},
		{
			name:            "token price with no token item configured",
			etcAssets:       plentyOfTokens,
			useCapacity:     24,
			useAssets:       noAssets,
			tokenTemplateId: 0,
			tokenPrice:      5,
			quantity:        1,
			wantError:       shops.ErrorGenericError,
		},
		{
			name:            "no price configured at all",
			etcAssets:       plentyOfTokens,
			useCapacity:     24,
			useAssets:       noAssets,
			tokenTemplateId: perfectPitch,
			tokenPrice:      0,
			quantity:        1,
			wantError:       shops.ErrorGenericError,
		},
		{
			name:            "zero quantity",
			etcAssets:       plentyOfTokens,
			useCapacity:     24,
			useAssets:       noAssets,
			tokenTemplateId: perfectPitch,
			tokenPrice:      5,
			quantity:        0,
			wantError:       shops.ErrorGenericError,
		},
		{
			name:            "cost overflows uint32",
			etcAssets:       plentyOfTokens,
			useCapacity:     24,
			useAssets:       noAssets,
			tokenTemplateId: perfectPitch,
			tokenPrice:      100000,
			quantity:        100000,
			wantError:       shops.ErrorGenericError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := testProcessor(t)
			buf := message.NewBuffer()
			c := testCharacter(t, tt.etcAssets(t), tt.useCapacity, tt.useAssets(t))
			cm := testCommodity(t, tt.tokenTemplateId, tt.tokenPrice)

			if err := p.buyWithTokens(buf)(c, cm, testItemId, tt.quantity); err != nil {
				t.Fatalf("buyWithTokens returned an error: %v", err)
			}

			all := buf.GetAll()

			// The strongest available form of "no partial purchase": nothing
			// at all was published on the compartment topic.
			if got := len(all[compartmentMessage.EnvCommandTopic]); got != 0 {
				t.Errorf("expected zero compartment commands on refusal, got %d", got)
			}

			events := all[shops.EnvStatusEventTopic]
			if len(events) != 1 {
				t.Fatalf("expected exactly one status event, got %d", len(events))
			}
			ev := decodeStatusError(t, events[0].Value)
			if ev.Type != shops.StatusEventTypeError {
				t.Errorf("status event type: got %q want %q", ev.Type, shops.StatusEventTypeError)
			}
			if ev.CharacterId != testCharacterId {
				t.Errorf("status event characterId: got %d want %d", ev.CharacterId, testCharacterId)
			}
			if ev.Body.Error != tt.wantError {
				t.Errorf("status event error: got %q want %q", ev.Body.Error, tt.wantError)
			}
			if ev.Body.Reason != "" {
				t.Errorf("expected no reason on a typed refusal, got %q", ev.Body.Reason)
			}
		})
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run from `<root>/services/atlas-npc-shops/atlas.com/npc`:

```bash
go test ./shops/ -run 'TestBuyWithTokens' -v
```

Expected: FAIL — a compile error naming `p.buyWithTokens undefined`.

- [ ] **Step 3: Implement `buyWithTokens`**

Append to `services/atlas-npc-shops/atlas.com/npc/shops/token.go`, and extend its import block to:

```go
import (
	"math"
	"sort"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"

	"atlas-npc/asset"
	"atlas-npc/character"
	"atlas-npc/commodities"
	"atlas-npc/kafka/message"
	"atlas-npc/kafka/message/shops"
)
```

```go
// buyWithTokens executes the token-priced purchase path: the commodity is paid
// for with cm.TokenPrice() units of cm.TokenTemplateId() rather than mesos.
//
// It deliberately takes no discountPrice: the v83 client sends the *meso*
// price in the buy packet's final field (CShopDlg::SendBuyRequest @ 0x7566f4,
// COutPacket::Encode4(&v66, v8[6]) where v8[6] is ITEM+24 = mesoPrice), which
// is 0 for a token item. The commodity row is the only pricing authority.
//
// The token item is resolved from the row, never hardcoded — the v83 client
// hardcodes 4310000 for its own local pre-check (0x41C3F0), but the server
// stays version- and vendor-agnostic.
//
// Guard order matters: the free-slot probe precedes any consumption so tokens
// are never destroyed for an item that cannot be received, mirroring the meso
// path at processor.go:462-467.
func (p *ProcessorImpl) buyWithTokens(mb *message.Buffer) func(c character.Model, cm commodities.Model, itemTemplateId uint32, quantity uint32) error {
	return func(c character.Model, cm commodities.Model, itemTemplateId uint32, quantity uint32) error {
		characterId := c.Id()

		if cm.TokenPrice() == 0 {
			p.l.Errorf("Character [%d] is attempting to buy item [%d] but no price is configured.", characterId, itemTemplateId)
			return mb.Put(shops.EnvStatusEventTopic, errorEventProvider(characterId, shops.ErrorGenericError))
		}
		if cm.TokenTemplateId() == 0 {
			p.l.Errorf("Character [%d] is attempting to buy item [%d] but it has a token price with no token item configured.", characterId, itemTemplateId)
			return mb.Put(shops.EnvStatusEventTopic, errorEventProvider(characterId, shops.ErrorGenericError))
		}

		tokenType, ok := inventory.TypeFromItemId(item.Id(cm.TokenTemplateId()))
		if !ok {
			p.l.Errorf("Character [%d] is attempting to buy item [%d] but token item [%d] is not a valid item.", characterId, itemTemplateId, cm.TokenTemplateId())
			return mb.Put(shops.EnvStatusEventTopic, errorEventProvider(characterId, shops.ErrorGenericError))
		}
		destinationType, ok := inventory.TypeFromItemId(item.Id(itemTemplateId))
		if !ok {
			p.l.Errorf("Character [%d] is attempting to buy item [%d] but it is not a valid item.", characterId, itemTemplateId)
			return mb.Put(shops.EnvStatusEventTopic, errorEventProvider(characterId, shops.ErrorGenericError))
		}

		// quantity arrives from the wire; uint32 x uint32 can wrap and produce
		// a small cost for a large purchase, which would grant items without
		// charging for them.
		total := uint64(cm.TokenPrice()) * uint64(quantity)
		if total == 0 || total > uint64(math.MaxUint32) {
			p.l.Errorf("Character [%d] is attempting to buy [%d] of item [%d] at token price [%d], which is not a valid cost.", characterId, quantity, itemTemplateId, cm.TokenPrice())
			return mb.Put(shops.EnvStatusEventTopic, errorEventProvider(characterId, shops.ErrorGenericError))
		}
		cost := uint32(total)

		draws, available := planTokenSpend(c.Inventory().CompartmentByType(tokenType).Assets(), cm.TokenTemplateId(), cost)
		if available < uint64(cost) {
			p.l.Errorf("Character [%d] is attempting to buy item [%d] but holds [%d] of token item [%d] and needs [%d].", characterId, itemTemplateId, available, cm.TokenTemplateId(), cost)
			return mb.Put(shops.EnvStatusEventTopic, errorEventProvider(characterId, shops.ErrorNeedMoreItems))
		}

		if _, err := c.Inventory().CompartmentByType(destinationType).NextFreeSlot(); err != nil {
			p.l.WithError(err).Errorf("Cannot locate free slot for character [%d].", characterId)
			return mb.Put(shops.EnvStatusEventTopic, errorEventProvider(characterId, shops.ErrorInventoryFull))
		}

		for _, d := range draws {
			if err := p.compP.RequestDestroyItem(mb)(characterId, tokenType, d.slot, d.quantity); err != nil {
				return err
			}
		}
		if err := p.compP.RequestCreateItem(mb)(characterId, itemTemplateId, quantity); err != nil {
			return err
		}

		p.l.Debugf("Character [%d] bought [%d] of item [%d] for [%d] of token item [%d].", characterId, quantity, itemTemplateId, cost, cm.TokenTemplateId())
		return nil
	}
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run from `<root>/services/atlas-npc-shops/atlas.com/npc`:

```bash
go test ./shops/ -run 'TestBuyWithTokens|TestPlanTokenSpend' -v
```

Expected: PASS — `TestBuyWithTokensSufficientBalanceSpansSlots`, `TestBuyWithTokensQuantityMultipliesCost`, all seven `TestBuyWithTokensRefusals` subtests, and all nine `TestPlanTokenSpend` subtests.

- [ ] **Step 5: Wire the branch into `Buy()`**

In `services/atlas-npc-shops/atlas.com/npc/shops/processor.go`, replace these two lines (currently at ~484, the last statement inside the innermost closure of `Buy`):

```go
			// TODO: implement TokenItem purchasing.
			return mb.Put(shops.EnvStatusEventTopic, reasonErrorEventProvider(characterId, shops.ErrorGenericErrorWithReason, "not implemented"))
```

with:

```go
			return p.buyWithTokens(mb)(c, cm, itemTemplateId, quantity)
```

Leave everything above it — the rechargeable branch and the `cm.MesoPrice() > 0` branch — exactly as it is.

- [ ] **Step 6: Verify the stub is gone and the build is clean**

Run from `<root>/services/atlas-npc-shops/atlas.com/npc`:

```bash
go build ./...
go vet ./...
grep -n 'TokenItem\|not implemented' shops/processor.go || echo "STUB GONE"
```

Expected: `go build` and `go vet` silent, then `STUB GONE`.

If a linter later reports `reasonErrorEventProvider` as unused, check for other callers first:

```bash
grep -rn 'reasonErrorEventProvider' .
```

Do not delete it as a side effect of this task — removing a producer is outside this task's scope, and Go does not fail the build on an unused package-level function.

- [ ] **Step 7: Run the full suite with the race detector**

Run from `<root>/services/atlas-npc-shops/atlas.com/npc`:

```bash
go test -race ./...
```

Expected: every package `ok` or `[no test files]`. In particular `atlas-npc/shops` must pass — the pre-existing suite (`processor_test.go`, `cache_test.go`, `registry_test.go`, `rest_test.go`, `provider_tenant_test.go`, `processor_partition_test.go`, `resource_paginate_test.go`) proves the meso and rechargeable paths are unchanged. If any pre-existing test now fails, the branch wiring is wrong — fix it rather than editing the test.

- [ ] **Step 8: Remove the two satisfied TODO entries**

In `<root>/docs/TODO.md`, delete these two lines. Locate them by content, not by line number — the numbers shift as the file is edited, and the second entry's `shops/processor.go:430` reference was already stale before this task.

Under "### High Priority (Feature Incomplete)" (currently line 15):

```
- [ ] **TokenItem Purchasing** - Returns "not implemented" error in NPC shops
```

Under "### NPC Shops Service" (currently line 270):

```
- [ ] **Implement TokenItem purchasing** (`shops/processor.go:430`)
```

Delete only those two lines. If deleting the second one leaves the `### NPC Shops Service` heading with no bullets under it, remove that now-empty heading too, keeping the surrounding blank-line spacing consistent with its neighbouring sections.

- [ ] **Step 9: Verify the TODO entries are gone**

Run from `<root>`:

```bash
grep -n 'TokenItem' docs/TODO.md || echo "TODO ENTRIES REMOVED"
```

Expected: `TODO ENTRIES REMOVED`.

- [ ] **Step 10: Commit**

```bash
git add services/atlas-npc-shops/atlas.com/npc/shops/token.go \
        services/atlas-npc-shops/atlas.com/npc/shops/token_test.go \
        services/atlas-npc-shops/atlas.com/npc/shops/processor.go \
        docs/TODO.md
git commit -m "feat(task-197): implement token-item purchasing in NPC shops

Replaces the 'not implemented' stub in Buy() with a generic token path
driven by the commodity row's tokenTemplateId, so any token vendor works
without code changes. Consumption spans multiple stacks in ascending slot
order; the free-slot probe precedes any destroy so tokens are never spent
on an item that cannot be received. The client-supplied discountPrice is
structurally absent from the path."
```

---

## Task 5: Move `poolSearchConfig.ts` into `src/lib/items/`

`poolSearchConfig.ts` currently lives under `components/features/characters/templates/`. Once `features/items/` consumes it (Task 6), leaving it there would make the items feature depend on the characters feature. `src/lib/items/` already exists (it holds `taxonomy.ts`, imported by `items.service.ts`, `types/models/item.ts` and `pages/ItemsPage.tsx`), so that is its natural home. The file's contents do not change — only its location and two import statements.

**Files:**
- Move: `services/atlas-ui/src/components/features/characters/templates/poolSearchConfig.ts` → `services/atlas-ui/src/lib/items/poolSearchConfig.ts`
- Modify: `services/atlas-ui/src/components/features/characters/templates/ItemSearchCombobox.tsx:15`
- Modify: `services/atlas-ui/src/components/features/characters/presets/EquipmentSection.tsx:13`

**Interfaces:**
- Consumes: nothing.
- Produces, for Tasks 6 and 7: `@/lib/items/poolSearchConfig` exporting `POOL_SEARCH_CONFIGS: Record<SearchPoolKey, PoolSearchConfig>`, `type SearchPoolKey = "tops" | "bottoms" | "shoes" | "weapons" | "items"`, `interface PoolSearchConfig { compartment?: "equipment"; subcategory?: string; clientSubcategories?: ReadonlySet<string> }`, and `WEAPON_SUBCATEGORIES`.

- [ ] **Step 1: Confirm there are exactly two importers**

Run from `<root>/services/atlas-ui`:

```bash
grep -rn 'poolSearchConfig' src
```

Expected: exactly two lines —
```
src/components/features/characters/templates/ItemSearchCombobox.tsx:15:import { POOL_SEARCH_CONFIGS, type SearchPoolKey } from "./poolSearchConfig";
src/components/features/characters/presets/EquipmentSection.tsx:13:import type { SearchPoolKey } from "../templates/poolSearchConfig";
```

If a third importer appears, update it the same way in Step 3.

- [ ] **Step 2: Move the file**

Run from `<root>`:

```bash
git mv services/atlas-ui/src/components/features/characters/templates/poolSearchConfig.ts \
       services/atlas-ui/src/lib/items/poolSearchConfig.ts
```

- [ ] **Step 3: Update both import statements**

In `services/atlas-ui/src/components/features/characters/templates/ItemSearchCombobox.tsx`, replace:

```ts
import { POOL_SEARCH_CONFIGS, type SearchPoolKey } from "./poolSearchConfig";
```

with:

```ts
import {
  POOL_SEARCH_CONFIGS,
  type SearchPoolKey,
} from "@/lib/items/poolSearchConfig";
```

In `services/atlas-ui/src/components/features/characters/presets/EquipmentSection.tsx`, replace:

```ts
import type { SearchPoolKey } from "../templates/poolSearchConfig";
```

with:

```ts
import type { SearchPoolKey } from "@/lib/items/poolSearchConfig";
```

- [ ] **Step 4: Verify no stale references remain**

Run from `<root>/services/atlas-ui`:

```bash
grep -rn 'templates/poolSearchConfig\|"./poolSearchConfig"' src || echo "NO STALE IMPORTS"
```

Expected: `NO STALE IMPORTS`.

- [ ] **Step 5: Type-check and test**

Run from `<root>/services/atlas-ui` (needs nvm 22 on PATH):

```bash
npm run build
npm run test
```

Expected: `build` completes with no `tsc` errors; `vitest run` reports all suites passing, including the untouched `ItemSearchCombobox.test.tsx`.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-ui/src/lib/items/poolSearchConfig.ts \
        services/atlas-ui/src/components/features/characters/templates/ItemSearchCombobox.tsx \
        services/atlas-ui/src/components/features/characters/presets/EquipmentSection.tsx
git commit -m "refactor(task-197): move poolSearchConfig to src/lib/items

Prepares for the items feature consuming it, without making the items
feature depend on the characters feature."
```

---

## Task 6: Extract `useItemSearch` and `ItemSearchResults` from `ItemSearchCombobox`

`ItemSearchCombobox` is an *action* control — it fires `onAdd(id)` and holds no value. The commodity dialog needs a *value-bearing* field (Task 7). Rather than duplicate ~90 lines of subtle behaviour, extract the logic.

The `settled {term, page}` atomicity comment at `ItemSearchCombobox.tsx:38-49` documents a real prior regression: a synchronous page reset on every keystroke could pair the OLD settled term with a NEW page number and fire an un-debounced query. That coupling must survive the extraction verbatim — it is the single riskiest part of this task.

`ItemSearchCombobox`'s public props (`poolKey`, `existingIds`, `onAdd`, `triggerLabel`, `debounceMs`) and its rendered DOM must not change, so `__tests__/ItemSearchCombobox.test.tsx` passes untouched. **That file is the regression harness — do not edit it.**

**Files:**
- Create: `services/atlas-ui/src/components/features/items/item-search/useItemSearch.ts`
- Create: `services/atlas-ui/src/components/features/items/item-search/ItemSearchResults.tsx`
- Modify: `services/atlas-ui/src/components/features/characters/templates/ItemSearchCombobox.tsx`
- Test (existing, unmodified): `services/atlas-ui/src/components/features/characters/templates/__tests__/ItemSearchCombobox.test.tsx`

**Interfaces:**
- Consumes: `@/lib/items/poolSearchConfig` (Task 5); `itemsService.searchItems(filters: ItemSearchFilters): Promise<ItemSearchPage>` and `type ItemSearchFilters` from `@/services/api/items.service`; `type ItemSearchResult` from `@/types/models/item`; `useTenant` from `@/context/tenant-context`; `getAssetIconUrl` from `@/lib/utils/asset-url`.
- Produces, for Task 7:
  - `useItemSearch({ poolKey, open, debounceMs }: UseItemSearchOptions): UseItemSearchResult` where `UseItemSearchResult` is `{ search: string; setSearch: (v: string) => void; rows: ItemSearchResult[]; manualId: number | undefined; hasMore: boolean; loadMore: () => void; isLoading: boolean; isError: boolean; settledTerm: string; reset: () => void }`
  - `<ItemSearchResults rows manualId isLoading isError settledTerm onPick disabledIds? leadingRow? />` — renders the `<ul role="listbox">` and nothing else.

- [ ] **Step 1: Create the headless hook**

Create `services/atlas-ui/src/components/features/items/item-search/useItemSearch.ts`:

```ts
import { useEffect, useMemo, useState } from "react";
import { useQuery, keepPreviousData } from "@tanstack/react-query";
import { itemsService } from "@/services/api/items.service";
import type { ItemSearchFilters } from "@/services/api/items.service";
import type { ItemSearchResult } from "@/types/models/item";
import { useTenant } from "@/context/tenant-context";
import {
  POOL_SEARCH_CONFIGS,
  type SearchPoolKey,
} from "@/lib/items/poolSearchConfig";

const PAGE_SIZE = 50;

export interface UseItemSearchOptions {
  poolKey: SearchPoolKey;
  /** Queries only fire while the consumer's popover is open. */
  open: boolean;
  /** Test hook: pass 0 to disable debouncing. */
  debounceMs?: number;
}

export interface UseItemSearchResult {
  search: string;
  setSearch: (value: string) => void;
  rows: ItemSearchResult[];
  /** The search box parsed as a raw id, for the "Use id N" escape hatch. */
  manualId: number | undefined;
  hasMore: boolean;
  loadMore: () => void;
  isLoading: boolean;
  isError: boolean;
  /** The debounced term the current results belong to. */
  settledTerm: string;
  reset: () => void;
}

export function useItemSearch({
  poolKey,
  open,
  debounceMs = 300,
}: UseItemSearchOptions): UseItemSearchResult {
  const { activeTenant } = useTenant();
  const [search, setSearch] = useState("");
  // The settled query term and its page are held TOGETHER so they can only
  // ever change atomically — the page must never move independently of the
  // term it belongs to. `term` updates only from the debounce timer's
  // callback (async — not a synchronous setState-in-effect, so this stays
  // clean under react-hooks/set-state-in-effect); "Load more" advances
  // `page` via a functional update that leaves `term` untouched. Raw
  // keystrokes update only `search` (below), never `settled` directly —
  // that decoupling is exactly what caused the prior regression: a
  // synchronous page reset on every keystroke could pair the OLD settled
  // term with a NEW page number and fire an un-debounced query.
  const [settled, setSettled] = useState({ term: "", page: 1 });

  useEffect(() => {
    const handle = setTimeout(() => {
      setSettled({ term: search, page: 1 });
    }, debounceMs);
    return () => clearTimeout(handle);
  }, [search, debounceMs]);

  const cfg = POOL_SEARCH_CONFIGS[poolKey];

  const filters: ItemSearchFilters = {
    pageNumber: settled.page,
    pageSize: PAGE_SIZE,
    ...(settled.term ? { q: settled.term } : {}),
    ...(cfg.compartment ? { compartment: cfg.compartment } : {}),
    ...(cfg.subcategory ? { subcategory: cfg.subcategory } : {}),
  };

  const query = useQuery({
    queryKey: ["item-search", poolKey, settled.term, settled.page],
    queryFn: () => itemsService.searchItems(filters),
    enabled: open && !!activeTenant && settled.term.trim().length > 0,
    placeholderData: keepPreviousData,
    staleTime: 10 * 60 * 1000,
  });

  const rows = useMemo(() => {
    const items = query.data?.items ?? [];
    return cfg.clientSubcategories
      ? items.filter((r) => cfg.clientSubcategories!.has(r.subcategory))
      : items;
  }, [query.data, cfg.clientSubcategories]);

  const manualId = /^\d+$/.test(search.trim())
    ? Number(search.trim())
    : undefined;

  return {
    search,
    setSearch,
    rows,
    manualId,
    hasMore: (query.data?.lastPage ?? 1) > settled.page,
    loadMore: () => setSettled((s) => ({ ...s, page: s.page + 1 })),
    isLoading: query.isLoading,
    isError: query.isError,
    settledTerm: settled.term,
    reset: () => setSearch(""),
  };
}
```

- [ ] **Step 2: Create the presentational result list**

Create `services/atlas-ui/src/components/features/items/item-search/ItemSearchResults.tsx`. The row markup, class names, ARIA attributes and conditional-state ordering are copied verbatim from `ItemSearchCombobox` so the existing DOM assertions keep passing:

```tsx
import type { ReactNode } from "react";
import type { ItemSearchResult } from "@/types/models/item";
import { getAssetIconUrl } from "@/lib/utils/asset-url";
import { useTenant } from "@/context/tenant-context";

interface ItemSearchResultsProps {
  rows: ItemSearchResult[];
  manualId: number | undefined;
  isLoading: boolean;
  isError: boolean;
  settledTerm: string;
  onPick: (id: number) => void;
  /** Ids already in the caller's pool — rendered disabled with an "Added" tag. */
  disabledIds?: number[];
  /** Rendered as the first <li>. Used for ItemPicker's "None" option. */
  leadingRow?: ReactNode;
}

export function ItemSearchResults({
  rows,
  manualId,
  isLoading,
  isError,
  settledTerm,
  onPick,
  disabledIds = [],
  leadingRow,
}: ItemSearchResultsProps) {
  const { activeTenant } = useTenant();

  return (
    <ul role="listbox" className="mt-2 max-h-64 space-y-0.5 overflow-y-auto">
      {leadingRow}
      {rows.map((row) => {
        const id = Number(row.id);
        const inPool = disabledIds.includes(id);
        return (
          <li
            key={row.id}
            role="option"
            aria-selected={false}
            aria-disabled={inPool}
            tabIndex={inPool ? -1 : 0}
            onClick={() => !inPool && onPick(id)}
            onKeyDown={(e) => {
              if ((e.key === "Enter" || e.key === " ") && !inPool) {
                e.preventDefault();
                onPick(id);
              }
            }}
            className={
              inPool
                ? "flex cursor-not-allowed items-center gap-2 rounded px-2 py-1 opacity-50"
                : "flex cursor-pointer items-center gap-2 rounded px-2 py-1 hover:bg-accent focus-visible:bg-accent"
            }
          >
            {activeTenant && (
              <img
                src={getAssetIconUrl(
                  activeTenant.id,
                  activeTenant.attributes.region,
                  activeTenant.attributes.majorVersion,
                  activeTenant.attributes.minorVersion,
                  "item",
                  id,
                )}
                alt=""
                width={24}
                height={24}
                loading="lazy"
                className="[image-rendering:pixelated]"
                onError={(e) => {
                  (e.target as HTMLImageElement).style.visibility = "hidden";
                }}
              />
            )}
            <span className="flex-1 truncate text-sm">{row.name}</span>
            <span className="font-mono text-xs text-muted-foreground">
              {row.id}
            </span>
            {inPool && (
              <span className="text-xs text-muted-foreground">Added</span>
            )}
          </li>
        );
      })}
      {manualId !== undefined && (
        <li
          role="option"
          aria-selected={false}
          tabIndex={0}
          onClick={() => onPick(manualId)}
          onKeyDown={(e) => {
            if (e.key === "Enter" || e.key === " ") {
              e.preventDefault();
              onPick(manualId);
            }
          }}
          className="cursor-pointer rounded px-2 py-1 text-sm hover:bg-accent focus-visible:bg-accent"
        >
          Use id {manualId}
        </li>
      )}
      {isLoading && settledTerm && (
        <li className="px-2 py-1 text-sm text-muted-foreground">Searching…</li>
      )}
      {isError && settledTerm && (
        <li className="px-2 py-1 text-sm text-warning-foreground">
          Search failed — enter an id manually
        </li>
      )}
      {!isLoading &&
        !isError &&
        settledTerm &&
        rows.length === 0 &&
        manualId === undefined && (
          <li className="px-2 py-1 text-sm text-muted-foreground">
            No matches.
          </li>
        )}
    </ul>
  );
}
```

- [ ] **Step 3: Rewrite `ItemSearchCombobox` as a shell over both**

Replace the entire contents of `services/atlas-ui/src/components/features/characters/templates/ItemSearchCombobox.tsx` with:

```tsx
import { useState } from "react";
import { Plus } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import type { SearchPoolKey } from "@/lib/items/poolSearchConfig";
import { useItemSearch } from "@/components/features/items/item-search/useItemSearch";
import { ItemSearchResults } from "@/components/features/items/item-search/ItemSearchResults";

interface ItemSearchComboboxProps {
  poolKey: SearchPoolKey;
  existingIds: number[];
  onAdd: (id: number) => void;
  triggerLabel?: string;
  /** Test hook: pass 0 to disable debouncing. */
  debounceMs?: number;
}

export function ItemSearchCombobox({
  poolKey,
  existingIds,
  onAdd,
  triggerLabel = "Add",
  debounceMs = 300,
}: ItemSearchComboboxProps) {
  const [open, setOpen] = useState(false);
  const search = useItemSearch({ poolKey, open, debounceMs });

  const handleAdd = (id: number) => {
    if (existingIds.includes(id)) return;
    onAdd(id);
    setOpen(false);
    search.reset();
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button type="button" variant="outline" size="sm">
          <Plus className="size-4" /> {triggerLabel}
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-80 p-2" align="start">
        <Input
          autoFocus
          value={search.search}
          onChange={(e) => search.setSearch(e.target.value)}
          placeholder="Search by name or enter an id…"
        />
        <ItemSearchResults
          rows={search.rows}
          manualId={search.manualId}
          isLoading={search.isLoading}
          isError={search.isError}
          settledTerm={search.settledTerm}
          disabledIds={existingIds}
          onPick={handleAdd}
        />
        {search.hasMore && (
          <Button
            type="button"
            variant="ghost"
            size="sm"
            className="mt-1 w-full"
            onClick={search.loadMore}
          >
            Load more
          </Button>
        )}
      </PopoverContent>
    </Popover>
  );
}
```

- [ ] **Step 4: Run the regression harness**

Run from `<root>/services/atlas-ui`:

```bash
npm run test -- src/components/features/characters/templates/__tests__/ItemSearchCombobox.test.tsx
```

Expected: PASS, with the test file unmodified. If any assertion fails, the extraction changed observable behaviour — fix the extraction, never the test.

- [ ] **Step 5: Confirm the harness really is untouched**

Run from `<root>`:

```bash
git diff --stat -- services/atlas-ui/src/components/features/characters/templates/__tests__/ItemSearchCombobox.test.tsx
```

Expected: no output (zero changes to that file).

- [ ] **Step 6: Full type-check and test run**

Run from `<root>/services/atlas-ui`:

```bash
npm run build
npm run test
```

Expected: no `tsc` errors; every suite passes.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-ui/src/components/features/items/item-search/useItemSearch.ts \
        services/atlas-ui/src/components/features/items/item-search/ItemSearchResults.tsx \
        services/atlas-ui/src/components/features/characters/templates/ItemSearchCombobox.tsx
git commit -m "refactor(task-197): extract useItemSearch and ItemSearchResults

ItemSearchCombobox becomes a thin trigger+popover shell over a headless
search hook and a presentational result list, so a value-bearing picker
can reuse the same behaviour instead of forking it. Props and rendered
DOM are unchanged; the existing test file is the regression harness and
is untouched."
```

---

## Task 7: Add the value-bearing `ItemPicker`

`ItemPicker` is the intersection of `ItemSearchCombobox` (owns the item-search behaviour) and `MapPicker` (the value-bearing precedent: `value`/`onChange`, resolves the current value to a label, manual-id escape hatch, `unresolved` hint). It lives in `features/items/item-search/`, not `features/npc/`, because it is a generic control and the commodity dialog is its first consumer, not its only one.

Label resolution uses the existing `useItemName` hook (`lib/hooks/api/useItemStrings.ts`) rather than calling `itemsService.getItemName` directly: both hit `/api/data/item-strings/{id}`, but the hook shares the `itemStringKeys.byId` cache key, so a name already fetched by the commodity grid or a previous dialog open is served from cache with no request.

**Files:**
- Create: `services/atlas-ui/src/components/features/items/item-search/ItemPicker.tsx`
- Test: `services/atlas-ui/src/components/features/items/item-search/__tests__/ItemPicker.test.tsx`

**Interfaces:**
- Consumes: `useItemSearch`, `ItemSearchResults` (Task 6); `useItemName(itemId: string): UseQueryResult<string, Error>` from `@/lib/hooks/api/useItemStrings`; `type SearchPoolKey` from `@/lib/items/poolSearchConfig`.
- Produces, for Task 8:
  - `<ItemPicker value={number} onChange={(id: number) => void} poolKey? placeholder? allowClear? disabled? debounceMs? id? />`
  - `value === 0` means unset. `allowClear` renders a "None" row that calls `onChange(0)`.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-ui/src/components/features/items/item-search/__tests__/ItemPicker.test.tsx`:

```tsx
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

const searchItemsMock = vi.fn();
vi.mock("@/services/api/items.service", () => ({
  itemsService: { searchItems: (...a: unknown[]) => searchItemsMock(...a) },
}));

const useItemNameMock = vi.fn();
vi.mock("@/lib/hooks/api/useItemStrings", () => ({
  useItemName: (...a: unknown[]) => useItemNameMock(...a),
}));

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({
    activeTenant: {
      id: "t1",
      attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
    },
  }),
}));

import { ItemPicker } from "../ItemPicker";

function renderPicker(
  props: Partial<React.ComponentProps<typeof ItemPicker>> = {},
) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  const onChange = props.onChange ?? vi.fn();
  const utils = render(
    <QueryClientProvider client={client}>
      <ItemPicker value={0} debounceMs={0} {...props} onChange={onChange} />
    </QueryClientProvider>,
  );
  return { ...utils, onChange };
}

const page = (items: unknown[]) => ({
  items,
  total: items.length,
  pageNumber: 1,
  pageSize: 50,
  lastPage: 1,
});

beforeEach(() => {
  searchItemsMock.mockReset();
  useItemNameMock.mockReset();
  useItemNameMock.mockReturnValue({ data: undefined, isError: false });
});

describe("ItemPicker", () => {
  it("renders the placeholder when unset and does not look up item 0", () => {
    renderPicker({ value: 0, placeholder: "None" });

    expect(screen.getByRole("button", { name: "None" })).toBeInTheDocument();
    expect(useItemNameMock).toHaveBeenCalledWith("");
  });

  it("renders the resolved name and id once the lookup settles", () => {
    useItemNameMock.mockReturnValue({ data: "Perfect Pitch", isError: false });
    renderPicker({ value: 4310000 });

    expect(
      screen.getByRole("button", { name: "Perfect Pitch · 4310000" }),
    ).toBeInTheDocument();
  });

  it("falls back to the raw id while loading", () => {
    useItemNameMock.mockReturnValue({ data: undefined, isError: false });
    renderPicker({ value: 4310000 });

    expect(
      screen.getByRole("button", { name: "Item 4310000" }),
    ).toBeInTheDocument();
    expect(screen.queryByText(/couldn't resolve/i)).not.toBeInTheDocument();
  });

  it("falls back to the raw id and hints when the lookup fails", () => {
    useItemNameMock.mockReturnValue({ data: undefined, isError: true });
    renderPicker({ value: 4310000 });

    expect(
      screen.getByRole("button", { name: "Item 4310000" }),
    ).toBeInTheDocument();
    expect(screen.getByText(/couldn't resolve/i)).toBeInTheDocument();
  });

  it("picking a row calls onChange with its id and closes the popover", async () => {
    const user = userEvent.setup();
    searchItemsMock.mockResolvedValue(
      page([{ id: "2022503", name: "Red Potion", subcategory: "potion" }]),
    );
    const { onChange } = renderPicker({ value: 0, placeholder: "None" });

    await user.click(screen.getByRole("button", { name: "None" }));
    await user.type(screen.getByPlaceholderText(/search by name/i), "red");

    await user.click(await screen.findByText("Red Potion"));

    expect(onChange).toHaveBeenCalledWith(2022503);
    await waitFor(() =>
      expect(screen.queryByRole("listbox")).not.toBeInTheDocument(),
    );
  });

  it("commits a raw id through the Use id escape hatch", async () => {
    const user = userEvent.setup();
    searchItemsMock.mockResolvedValue(page([]));
    const { onChange } = renderPicker({ value: 0, placeholder: "None" });

    await user.click(screen.getByRole("button", { name: "None" }));
    await user.type(screen.getByPlaceholderText(/search by name/i), "4310000");

    await user.click(await screen.findByText("Use id 4310000"));

    expect(onChange).toHaveBeenCalledWith(4310000);
  });

  it("renders a None row only when allowClear is set, and it clears to 0", async () => {
    const user = userEvent.setup();
    searchItemsMock.mockResolvedValue(page([]));
    useItemNameMock.mockReturnValue({ data: "Perfect Pitch", isError: false });

    const withClear = renderPicker({ value: 4310000, allowClear: true });
    await user.click(
      screen.getByRole("button", { name: "Perfect Pitch · 4310000" }),
    );
    await user.click(screen.getByText("None"));
    expect(withClear.onChange).toHaveBeenCalledWith(0);
    withClear.unmount();

    const withoutClear = renderPicker({ value: 4310000 });
    await user.click(
      screen.getByRole("button", { name: "Perfect Pitch · 4310000" }),
    );
    expect(screen.queryByText("None")).not.toBeInTheDocument();
    withoutClear.unmount();
  });
});
```

- [ ] **Step 2: Run the test to verify it fails**

Run from `<root>/services/atlas-ui`:

```bash
npm run test -- src/components/features/items/item-search/__tests__/ItemPicker.test.tsx
```

Expected: FAIL — module not found, `../ItemPicker` does not exist.

- [ ] **Step 3: Write `ItemPicker`**

Create `services/atlas-ui/src/components/features/items/item-search/ItemPicker.tsx`:

```tsx
import { useState } from "react";
import { TriangleAlert } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { useItemName } from "@/lib/hooks/api/useItemStrings";
import type { SearchPoolKey } from "@/lib/items/poolSearchConfig";
import { useItemSearch } from "./useItemSearch";
import { ItemSearchResults } from "./ItemSearchResults";

interface ItemPickerProps {
  /** 0 means unset. */
  value: number;
  onChange: (id: number) => void;
  /** Defaults to the unfiltered all-compartment pool. */
  poolKey?: SearchPoolKey;
  /** Trigger label rendered when value is 0. */
  placeholder?: string;
  /** Renders a "None" row that clears the value back to 0. */
  allowClear?: boolean;
  disabled?: boolean;
  /** Test hook: pass 0 to disable debouncing. */
  debounceMs?: number;
  /** Applied to the trigger button so a <Label htmlFor> can address it. */
  id?: string;
}

export function ItemPicker({
  value,
  onChange,
  poolKey = "items",
  placeholder = "Select an item…",
  allowClear = false,
  disabled = false,
  debounceMs = 300,
  id,
}: ItemPickerProps) {
  const [open, setOpen] = useState(false);
  const search = useItemSearch({ poolKey, open, debounceMs });
  // Guard on value > 0: String(0) is "0", which is truthy and would fire a
  // lookup for item id 0.
  const current = useItemName(value > 0 ? String(value) : "");

  const label =
    value === 0
      ? placeholder
      : current.data
        ? `${current.data} · ${value}`
        : `Item ${value}`;
  // atlas-data coverage varies by version: unresolvable is a hint, not an error.
  const unresolved = value > 0 && !current.data && current.isError;

  const pick = (nextId: number) => {
    onChange(nextId);
    setOpen(false);
    search.reset();
  };

  return (
    <div className="space-y-1">
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <Button
            type="button"
            variant="outline"
            className="w-full justify-start font-normal"
            disabled={disabled}
            {...(id ? { id } : {})}
          >
            {label}
          </Button>
        </PopoverTrigger>
        <PopoverContent className="w-96 p-2" align="start">
          <Input
            autoFocus
            value={search.search}
            onChange={(e) => search.setSearch(e.target.value)}
            placeholder="Search by name or enter an id…"
          />
          <ItemSearchResults
            rows={search.rows}
            manualId={search.manualId}
            isLoading={search.isLoading}
            isError={search.isError}
            settledTerm={search.settledTerm}
            onPick={pick}
            {...(allowClear
              ? {
                  leadingRow: (
                    <li
                      role="option"
                      aria-selected={false}
                      tabIndex={0}
                      onClick={() => pick(0)}
                      onKeyDown={(e) => {
                        if (e.key === "Enter" || e.key === " ") {
                          e.preventDefault();
                          pick(0);
                        }
                      }}
                      className="cursor-pointer rounded px-2 py-1 text-sm hover:bg-accent focus-visible:bg-accent"
                    >
                      None
                    </li>
                  ),
                }
              : {})}
          />
          {search.hasMore && (
            <Button
              type="button"
              variant="ghost"
              size="sm"
              className="mt-1 w-full"
              onClick={search.loadMore}
            >
              Load more
            </Button>
          )}
        </PopoverContent>
      </Popover>
      {unresolved && (
        <p className="flex items-center gap-1 text-xs text-warning-foreground">
          <TriangleAlert className="size-3" />
          couldn&apos;t resolve this item&apos;s name for this version
        </p>
      )}
    </div>
  );
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run from `<root>/services/atlas-ui`:

```bash
npm run test -- src/components/features/items/item-search/__tests__/ItemPicker.test.tsx
```

Expected: PASS — all seven cases.

- [ ] **Step 5: Full type-check and test run**

Run from `<root>/services/atlas-ui`:

```bash
npm run build
npm run test
```

Expected: no `tsc` errors; every suite passes, including the untouched `ItemSearchCombobox.test.tsx`.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-ui/src/components/features/items/item-search/ItemPicker.tsx \
        services/atlas-ui/src/components/features/items/item-search/__tests__/ItemPicker.test.tsx
git commit -m "feat(task-197): add value-bearing ItemPicker over the item search"
```

---

## Task 8: Use `ItemPicker` in `NpcShopCommodityDialog`

Today the dialog maps one `FIELDS` array of seven numeric `<Input>`s, with a `disabled` special case for `templateId` in edit mode (`NpcShopCommodityDialog.tsx:74-101`). The two template-id fields become item controls; the other five stay numeric and keep their loop.

Edit-mode `Template ID` renders as **text**, not a disabled picker — a disabled control that opens nothing is worse than a label. This preserves the existing `disabled` semantics (the edit dialog still cannot repoint a commodity at a different item) while making the row legible.

Payload invariance is the hard requirement: `form` stays `CommodityAttributes` with numeric ids, the reset-on-open logic at `NpcShopCommodityDialog.tsx:53-58` is untouched, and `onSubmit(form)` is unchanged. The pickers only write numbers into the same state the inputs wrote.

**Files:**
- Modify: `services/atlas-ui/src/components/features/npc/NpcShopCommodityDialog.tsx`
- Test: `services/atlas-ui/src/components/features/npc/__tests__/NpcShopCommodityDialog.test.tsx`

**Interfaces:**
- Consumes: `<ItemPicker … />` (Task 7); `useItemName` from `@/lib/hooks/api/useItemStrings`; `type CommodityAttributes` from `@/types/models/npc` (`templateId, mesoPrice, discountRate, tokenTemplateId, tokenPrice, period, levelLimit`, all `number`).
- Produces: nothing consumed by later tasks. `NpcShopCommodityDialog`'s own props (`open`, `onOpenChange`, `mode`, `initial`, `onSubmit`) are unchanged, so `NpcShopCard.tsx:393-408` needs no edit.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-ui/src/components/features/npc/__tests__/NpcShopCommodityDialog.test.tsx`:

```tsx
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

const searchItemsMock = vi.fn();
vi.mock("@/services/api/items.service", () => ({
  itemsService: { searchItems: (...a: unknown[]) => searchItemsMock(...a) },
}));

const useItemNameMock = vi.fn();
vi.mock("@/lib/hooks/api/useItemStrings", () => ({
  useItemName: (...a: unknown[]) => useItemNameMock(...a),
}));

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({
    activeTenant: {
      id: "t1",
      attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
    },
  }),
}));

import { NpcShopCommodityDialog } from "../NpcShopCommodityDialog";
import type { CommodityAttributes } from "@/types/models/npc";

const EXISTING: CommodityAttributes = {
  templateId: 2022503,
  mesoPrice: 0,
  discountRate: 0,
  tokenTemplateId: 4310000,
  tokenPrice: 5,
  period: 0,
  levelLimit: 0,
};

function renderDialog(
  props: Partial<React.ComponentProps<typeof NpcShopCommodityDialog>> = {},
) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  const onSubmit = props.onSubmit ?? vi.fn();
  const utils = render(
    <QueryClientProvider client={client}>
      <NpcShopCommodityDialog
        open
        onOpenChange={vi.fn()}
        mode="create"
        {...props}
        onSubmit={onSubmit}
      />
    </QueryClientProvider>,
  );
  return { ...utils, onSubmit };
}

const page = (items: unknown[]) => ({
  items,
  total: items.length,
  pageNumber: 1,
  pageSize: 50,
  lastPage: 1,
});

beforeEach(() => {
  searchItemsMock.mockReset();
  searchItemsMock.mockResolvedValue(page([]));
  useItemNameMock.mockReset();
  useItemNameMock.mockReturnValue({ data: undefined, isError: false });
});

describe("NpcShopCommodityDialog", () => {
  it("create mode submits the exact CommodityAttributes shape a picker-chosen id produces", async () => {
    const user = userEvent.setup();
    const { onSubmit } = renderDialog({ mode: "create" });

    // Choose the item via the Template ID picker's raw-id escape hatch.
    await user.click(screen.getByRole("button", { name: "Select an item…" }));
    await user.type(screen.getByPlaceholderText(/search by name/i), "2022503");
    await user.click(await screen.findByText("Use id 2022503"));

    // Choose the token item the same way.
    await user.click(screen.getByRole("button", { name: "None" }));
    await user.type(screen.getByPlaceholderText(/search by name/i), "4310000");
    await user.click(await screen.findByText("Use id 4310000"));

    const tokenPrice = screen.getByLabelText("Token Price");
    await user.clear(tokenPrice);
    await user.type(tokenPrice, "5");

    await user.click(screen.getByRole("button", { name: "Create" }));

    await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1));
    expect(onSubmit).toHaveBeenCalledWith({
      templateId: 2022503,
      mesoPrice: 0,
      discountRate: 0,
      tokenTemplateId: 4310000,
      tokenPrice: 5,
      period: 0,
      levelLimit: 0,
    });
  });

  it("edit mode renders the template id as read-only text, not a picker", () => {
    useItemNameMock.mockReturnValue({ data: "Red Potion", isError: false });
    renderDialog({ mode: "edit", initial: EXISTING });

    expect(screen.getByText("Red Potion · 2022503")).toBeInTheDocument();
    // The only remaining item-picker trigger is the token one.
    expect(
      screen.queryByRole("button", { name: /Red Potion · 2022503/ }),
    ).not.toBeInTheDocument();
  });

  it("edit mode's token picker is interactive and clears to 0", async () => {
    const user = userEvent.setup();
    useItemNameMock.mockReturnValue({ data: "Perfect Pitch", isError: false });
    const { onSubmit } = renderDialog({ mode: "edit", initial: EXISTING });

    await user.click(
      screen.getByRole("button", { name: "Perfect Pitch · 4310000" }),
    );
    await user.click(screen.getByText("None"));
    await user.click(screen.getByRole("button", { name: "Update" }));

    await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1));
    expect(onSubmit).toHaveBeenCalledWith({ ...EXISTING, tokenTemplateId: 0 });
  });

  it("submits an unedited commodity byte-for-byte unchanged", async () => {
    const user = userEvent.setup();
    const { onSubmit } = renderDialog({ mode: "edit", initial: EXISTING });

    await user.click(screen.getByRole("button", { name: "Update" }));

    await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1));
    expect(onSubmit).toHaveBeenCalledWith(EXISTING);
  });
});
```

Note on the second test: in edit mode the `Template ID` row is a `<span>`, so `getByText` finds it while `getByRole("button", …)` does not — that assertion pair is what proves the picker is gone.

- [ ] **Step 2: Run the test to verify it fails**

Run from `<root>/services/atlas-ui`:

```bash
npm run test -- src/components/features/npc/__tests__/NpcShopCommodityDialog.test.tsx
```

Expected: FAIL — the dialog still renders numeric `<Input>`s, so there is no `Select an item…` button and the first test cannot find it.

- [ ] **Step 3: Rewrite the dialog**

Replace the entire contents of `services/atlas-ui/src/components/features/npc/NpcShopCommodityDialog.tsx` with:

```tsx
import { useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ItemPicker } from "@/components/features/items/item-search/ItemPicker";
import { useItemName } from "@/lib/hooks/api/useItemStrings";
import type { CommodityAttributes } from "@/types/models/npc";

const EMPTY: CommodityAttributes = {
  templateId: 0,
  mesoPrice: 0,
  discountRate: 0,
  tokenTemplateId: 0,
  tokenPrice: 0,
  period: 0,
  levelLimit: 0,
};

interface NpcShopCommodityDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  mode: "create" | "edit";
  initial?: CommodityAttributes;
  onSubmit: (attrs: CommodityAttributes) => Promise<void> | void;
}

// Field order is preserved from the all-numeric version; `kind` decides which
// control renders, not where the row sits.
const FIELDS: Array<{
  key: keyof CommodityAttributes;
  label: string;
  kind: "item" | "number";
}> = [
  { key: "templateId", label: "Template ID", kind: "item" },
  { key: "mesoPrice", label: "Meso Price", kind: "number" },
  { key: "discountRate", label: "Discount Rate", kind: "number" },
  { key: "tokenTemplateId", label: "Token Template ID", kind: "item" },
  { key: "tokenPrice", label: "Token Price", kind: "number" },
  { key: "period", label: "Period", kind: "number" },
  { key: "levelLimit", label: "Level Limit", kind: "number" },
];

/** Read-only rendering of an item id, used for the non-editable edit-mode
 *  Template ID row. Falls back to the raw id while loading or on failure. */
function ResolvedItemName({ value, id }: { value: number; id: string }) {
  const current = useItemName(value > 0 ? String(value) : "");
  return (
    <span id={id} className="block truncate text-sm">
      {current.data ? `${current.data} · ${value}` : `Item ${value}`}
    </span>
  );
}

export function NpcShopCommodityDialog({
  open,
  onOpenChange,
  mode,
  initial,
  onSubmit,
}: NpcShopCommodityDialogProps) {
  const [form, setForm] = useState<CommodityAttributes>(initial ?? EMPTY);
  const [submitting, setSubmitting] = useState(false);

  // Reset the form when the dialog opens (or the target commodity changes
  // while open). Adjusted during render instead of in an effect.
  const [prevSync, setPrevSync] = useState({ open, initial });
  if (open !== prevSync.open || initial !== prevSync.initial) {
    setPrevSync({ open, initial });
    if (open) setForm(initial ?? EMPTY);
  }

  const handleSubmit = async () => {
    setSubmitting(true);
    try {
      await onSubmit(form);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-xl">
        <DialogHeader>
          <DialogTitle>
            {mode === "create" ? "Add Commodity" : "Edit Commodity"}
          </DialogTitle>
        </DialogHeader>
        <div className="grid gap-4 py-4">
          {FIELDS.map(({ key, label, kind }) => {
            const controlId = `commodity-${key}`;
            return (
              <div key={key} className="grid grid-cols-4 items-center gap-4">
                <Label htmlFor={controlId} className="text-right">
                  {label}
                </Label>
                {kind === "number" ? (
                  <Input
                    id={controlId}
                    name={key}
                    type="number"
                    value={form[key]}
                    onChange={(e) =>
                      setForm((prev) => ({
                        ...prev,
                        [key]: Number(e.target.value),
                      }))
                    }
                    className="col-span-3"
                  />
                ) : (
                  <div className="col-span-3">
                    {mode === "edit" && key === "templateId" ? (
                      <ResolvedItemName
                        value={form.templateId}
                        id={controlId}
                      />
                    ) : (
                      <ItemPicker
                        id={controlId}
                        value={form[key]}
                        onChange={(next) =>
                          setForm((prev) => ({ ...prev, [key]: next }))
                        }
                        {...(key === "tokenTemplateId"
                          ? { allowClear: true, placeholder: "None" }
                          : {})}
                      />
                    )}
                  </div>
                )}
              </div>
            );
          })}
        </div>
        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={submitting}
          >
            Cancel
          </Button>
          <Button onClick={handleSubmit} disabled={submitting}>
            {mode === "create" ? "Create" : "Update"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run from `<root>/services/atlas-ui`:

```bash
npm run test -- src/components/features/npc/__tests__/NpcShopCommodityDialog.test.tsx
```

Expected: PASS — all four cases.

- [ ] **Step 5: Verify the dialog's consumer needed no change**

Run from `<root>`:

```bash
git diff --stat -- services/atlas-ui/src/components/features/npc/NpcShopCard.tsx
```

Expected: no output. `NpcShopCommodityDialog`'s props are unchanged, so `NpcShopCard.tsx:393-408` still compiles as-is. If it did need a change, the props changed — revisit Step 3.

- [ ] **Step 6: Full type-check and test run**

Run from `<root>/services/atlas-ui`:

```bash
npm run build
npm run test
```

Expected: no `tsc` errors; every suite passes.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-ui/src/components/features/npc/NpcShopCommodityDialog.tsx \
        services/atlas-ui/src/components/features/npc/__tests__/NpcShopCommodityDialog.test.tsx
git commit -m "feat(task-197): search-pick items in the shop commodity dialog

Template ID and Token Template ID become item search pickers in Add mode;
Edit mode renders Template ID as the resolved item name and keeps it
non-editable. Request payloads are unchanged — the pickers only write the
same numbers the inputs wrote."
```

---

## Task 9: Full verification sweep and code review

Everything below runs from `<root>` unless stated otherwise. This task changes no source; if any gate fails, fix it in the workstream it belongs to and re-run this whole task.

**Files:** none created or modified (unless a gate turns up a fix).

**Interfaces:**
- Consumes: the complete branch from Tasks 1-8.
- Produces: `docs/tasks/task-197-inkwell-token-shop/audit.md` (written by the review agents).

- [ ] **Step 1: Go module gates**

Run from `<root>/services/atlas-npc-shops/atlas.com/npc`:

```bash
go test -race ./...
go vet ./...
go build ./...
```

Expected: every package `ok` or `[no test files]`; `go vet` and `go build` silent.

- [ ] **Step 2: Confirm `docker buildx bake` is not required**

Run from `<root>`:

```bash
git diff --name-only main...HEAD -- '*/go.mod' '*/go.sum'
```

Expected: no output. This task adds no dependency and no shared lib, so the bake step is not required (design §8.7). **If any `go.mod` shows up, run `docker buildx bake atlas-npc-shops` from `<root>` and require it to succeed before proceeding** — `go build` against `go.work` will not catch a missing `COPY libs/...` line in the shared Dockerfile.

- [ ] **Step 3: Repo guards**

Run from `<root>`:

```bash
tools/redis-key-guard.sh
tools/goroutine-guard.sh
```

Expected: both exit 0 with no findings. `tools/service-registration-guard.sh` is not required — no service was added and none of `services.json`, `deploy/k8s`, `docker-bake.hcl`, `go.work` or `tools/db-bootstrap.sh` changed. `tools/template-opcode-order-guard.sh` is not required — no tenant socket-config template changed.

- [ ] **Step 4: Lint and format**

Run from `<root>` (needs nvm 22 on PATH for the atlas-ui half):

```bash
tools/lint.sh --check
```

Expected: clean. If it reports formatting differences, run `tools/lint.sh` (no flags) to rewrite files in place, re-run `--check`, and amend the affected commit.

- [ ] **Step 5: atlas-ui gates**

Run from `<root>/services/atlas-ui`:

```bash
npm run build
npm run test
```

Expected: `tsc -b` and `vite build` succeed; every vitest suite passes. Both are required — `npm run build` is what type-checks the test files, so vitest alone is not sufficient verification.

- [ ] **Step 6: Verify the regression harness was never edited**

Run from `<root>`:

```bash
git diff --stat main...HEAD -- services/atlas-ui/src/components/features/characters/templates/__tests__/ItemSearchCombobox.test.tsx
```

Expected: no output.

- [ ] **Step 7: Verify the stub and TODOs are gone**

Run from `<root>`:

```bash
grep -rn 'TokenItem\|not implemented' services/atlas-npc-shops docs/TODO.md || echo "NO STUB, NO TODO"
```

Expected: `NO STUB, NO TODO`.

- [ ] **Step 8: Verify `discountPrice` never reaches the token path**

Run from `<root>`:

```bash
grep -n 'discountPrice' services/atlas-npc-shops/atlas.com/npc/shops/token.go || echo "TOKEN PATH HAS NO discountPrice"
```

Expected: `TOKEN PATH HAS NO discountPrice`. This is the structural proof for the PRD acceptance criterion "the client-supplied `discountPrice` provably does not influence the amount charged" — the parameter is absent from the code path entirely.

- [ ] **Step 9: Verify `4310000` is not hardcoded in service logic**

Run from `<root>`:

```bash
grep -rn '4310000' services/atlas-npc-shops --include='*.go'
```

Expected: hits only in `shops/token_test.go` (the `perfectPitch` test fixture). Any hit in non-test Go source violates FR-2.3 — the token item must come from `cm.TokenTemplateId()`.

- [ ] **Step 10: Verify the seed diff shape**

Run from `<root>`:

```bash
git diff --name-status main...HEAD -- deploy/seed
```

Expected: exactly eleven lines — five `D` (the pre-v83 deletions) and six `M` (the rewrites), and nothing else under `deploy/seed`.

- [ ] **Step 11: Verify the whole-branch diff shape**

Run from `<root>`:

```bash
git status --porcelain
git diff --name-status main...HEAD
```

Expected: `git status` empty (everything committed). The `git diff` list should contain only: the eleven `deploy/seed` entries, `docs/TODO.md`, the `docs/tasks/task-197-inkwell-token-shop/*.md` files, four files under `services/atlas-npc-shops`, and the eight files under `services/atlas-ui`. Anything else is unintended scope — investigate before proceeding.

- [ ] **Step 12: Run code review**

Invoke `superpowers:requesting-code-review`. Both Go and TypeScript changed, so it should dispatch `plan-adherence-reviewer`, `backend-guidelines-reviewer` and `frontend-guidelines-reviewer` in parallel. Each writes to `docs/tasks/task-197-inkwell-token-shop/audit.md`.

Ensure every dispatched agent operates inside `<root>` and writes nothing into the main checkout. After the run, confirm the tree is clean apart from `audit.md`:

```bash
git status --porcelain
```

- [ ] **Step 13: Address review findings and commit the audit**

Fix anything the reviewers flag as a real defect, re-running the relevant gates from Steps 1-11. Then:

```bash
git add docs/tasks/task-197-inkwell-token-shop/audit.md
git commit -m "docs(task-197): code review audit"
```

- [ ] **Step 14: Record the post-merge operator warning for the PR description**

The PR description **must** carry this warning — it is a risk the design flagged (R4) and it does not belong only in a doc:

> **Post-merge action required.** Seed files are the catalog, not the live rows. Correcting these files does not repair an already-seeded tenant; the live tenant must be re-seeded. Be aware that `libs/atlas-seeder/seed.go` `runSubdomain` performs `DeleteAllForTenant` followed by `BulkCreate`, and `ShopSubdomain.DeleteAllForTenant` (`shops/subdomain.go:34-49`) issues `db.Unscoped().Where("tenant_id = ?", …).Delete(...)` for **all** commodities and **all** shops of the tenant. Re-seeding `npc-shops` is therefore a **destructive full replace of every shop on that tenant**, not a merge of the one file that changed — any hand-edits made through the Web UI will be lost.
>
> After re-seeding, verify manually: Inkwell's shop lists nine items in the FR-1.3 order with Perfect Pitch prices, and a purchase debits the correct number of Perfect Pitch.

Do not open the PR until Steps 1-13 have all passed.

---

## Known Limitations Carried Forward (do not "fix" in this task)

These are recorded in `design.md` §7 and are deliberately out of scope. A reviewer should not treat them as defects of this change.

- **R1 — token-priced rechargeables are unreachable.** `item.IsRechargeable` (`libs/atlas-constants/item/constants.go:187`) is `IsThrowingStar || IsBullet` (`207xxxx`, `233xxxx`). A commodity that is both rechargeable and token-priced is captured by branch 1, priced in mesos, and refused. Inkwell sells none of these — the nine ids carry no `207`/`233` prefix — and reordering the branches would change the rechargeable path, which the PRD forbids.
- **R2 — the free-slot pre-check is conservative when the token compartment is also the destination.** Fully draining a token stack frees a slot the pre-check did not count. Identical to the meso path's behaviour; unreachable for Inkwell (token is ETC, the nine items are equip/use/setup).
- **R3 — the free-slot check does not consider merging into an existing partial stack** of the purchased item. Pre-existing on the meso path; parity preserved, not extended.
- **R5 — JMS 185 token rendering is unverified.** `tokenTemplateId` is gated to `Region=="GMS" && Major>=95` (`libs/atlas-packet/npc/clientbound/shop_list.go:58`), so JMS clients never receive it. Seeded identically anyway because the server reads it; the worst case is a cosmetic gap on JMS.
- **Per-commodity bundle quantity** stays a TODO in `services/atlas-channel` (`npc/shops/commodities/model.go` `Quantity()` returns a hardcoded `0`). PRD §2 non-goal — verified harmless: the v83 client reads the field at `ITEM+56` and uses it only in `sub_4284BE(itemId) || v8[14] > 1` (`CShopDlg::SendBuyRequest`) to *suppress* the quantity prompt, so `0` lands stackables on the prompt branch bounded by `slotMax`, which is correct for every commodity Atlas can currently express.

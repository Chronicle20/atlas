# Reward Pools UI — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the bolted-together "Gachapons" admin pages with one kind-adaptive "Reward Pools" surface (list + detail + full CRUD incl. the global pool), backed by four small backend additions to `atlas-reward-pools`.

**Architecture:** Phase 1 adds the missing write endpoints and fixes the kind-blind prize pool in `atlas-reward-pools` (Go/GORM). Phase 2 builds the frontend foundation (types, service, hooks, chance util, zod schemas). Phase 3 builds shared CRUD dialog components, then the list and detail pages. Phase 4 swaps routing/nav and deletes the old Gachapons modules. Design: `docs/tasks/task-128-item-tag-seal-incubator/design-reward-pools-ui.md`.

**Tech Stack:** Go + GORM + api2go JSON:API (atlas-reward-pools); atlas-ui = Vite + React Router v7 + TanStack Query 5 + react-hook-form + zod + shadcn/ui + Vitest.

## Global Constraints

- atlas-ui: run under **nvm 22** (`source ~/.nvm/nvm.sh; nvm use 22` before `npm`); `npm run build` type-checks `*.test.ts(x)`; gate on `npm run build` + `npm run test` + no new lint errors.
- atlas-ui conventions: **named exports** on pages; `@/` alias; compose `api` primitives from `@/lib/api/client`; React Query hooks under `src/lib/hooks/api/`; zod schemas under `src/lib/schemas/`; services under `src/services/api/`; tests in colocated `__tests__/` dirs, Vitest style (`vi.*`, never `jest.*`).
- **REST identity is unchanged**: base path stays `/api/gachapons` (+ `/api/global-items`); JSON:API resource types are `"gachapons"`, `"gachapon-items"`, `"global-gachapon-items"` (each `RestModel.GetName()`). Only UI-facing naming says "Reward Pools".
- Writes to `RegisterInputHandler` endpoints MUST use the JSON:API envelope `{data: {type, attributes}}` (`{data: {id, type, attributes}}` when an id is carried) — a bare attributes body 400s.
- `kind` is immutable after pool creation; a pool's `id` doubles as the egg item id for incubator pools.
- Go: `go build ./... && go vet ./... && go test -race ./...` clean in `services/atlas-reward-pools/atlas.com/reward-pools`; `docker buildx bake atlas-reward-pools` from the worktree root; `tools/redis-key-guard.sh` + `tools/goroutine-guard.sh` clean.
- Commit after every task on the `task-128-item-tag-seal-incubator` branch.

---

# Phase 1 — Backend: atlas-reward-pools write endpoints + kind-aware prize pool

All paths below are under `services/atlas-reward-pools/atlas.com/reward-pools/`.

### Task 1: Pool-item PATCH endpoint

**Files:**
- Modify: `item/administrator.go` (add `UpdateItem`)
- Modify: `item/processor.go` (add `Update` to interface + impl)
- Modify: `item/resource.go` (add PATCH route + handler)
- Test: `item/update_test.go` (create)

**Interfaces:**
- Consumes: `test.CreateItemProcessor(t)` → `(item.Processor, *gorm.DB, func())`; `test.TestTenantId`; existing `item.NewBuilder(tenantId uuid.UUID, id uint32)` builder.
- Produces: `PATCH /gachapons/{gachaponId}/items/{itemId}` → 204; `Processor.Update(id uint32, itemId uint32, quantity uint32, tier string, weight uint32) error`.

- [ ] **Step 1: Write the failing test**

`item/update_test.go`:

```go
package item_test

import (
	"atlas-reward-pools/item"
	"atlas-reward-pools/test"
	"testing"
)

func TestUpdateItem(t *testing.T) {
	processor, db, cleanup := test.CreateItemProcessor(t)
	defer cleanup()

	m, err := item.NewBuilder(test.TestTenantId, 0).
		SetGachaponId("4170001").
		SetItemId(2000000).
		SetQuantity(1).
		SetWeight(50).
		Build()
	if err != nil {
		t.Fatalf("Failed to build item model: %v", err)
	}
	if err = item.CreateItem(db, m); err != nil {
		t.Fatalf("Failed to create item: %v", err)
	}

	created, err := processor.GetByGachaponId("4170001")()
	if err != nil || len(created) != 1 {
		t.Fatalf("Expected 1 item, got %d (err %v)", len(created), err)
	}
	id := created[0].Id()

	if err = processor.Update(id, 2000001, 3, "", 75); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	after, err := processor.GetByGachaponId("4170001")()
	if err != nil || len(after) != 1 {
		t.Fatalf("Expected 1 item after update, got %d (err %v)", len(after), err)
	}
	got := after[0]
	if got.ItemId() != 2000001 || got.Quantity() != 3 || got.Weight() != 75 || got.Tier() != "" {
		t.Errorf("Update not applied: itemId=%d qty=%d weight=%d tier=%q",
			got.ItemId(), got.Quantity(), got.Weight(), got.Tier())
	}
	if got.GachaponId() != "4170001" {
		t.Errorf("Update must not re-parent: gachaponId=%q", got.GachaponId())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-reward-pools/atlas.com/reward-pools && go test ./item/ -run TestUpdateItem -v`
Expected: FAIL — `processor.Update undefined (type item.Processor has no field or method Update)`.

- [ ] **Step 3: Implement**

`item/administrator.go` — append (mirrors `gachapon/administrator.go:41` `UpdateGachapon`'s `Updates`-map form, which writes zero values):

```go
func UpdateItem(db *gorm.DB, id uint32, itemId uint32, quantity uint32, tier string, weight uint32) error {
	return db.Model(&entity{}).
		Where(&entity{ID: id}).
		Updates(map[string]interface{}{
			"item_id":  itemId,
			"quantity": quantity,
			"tier":     tier,
			"weight":   weight,
		}).Error
}
```

`item/processor.go` — add to the `Processor` interface (after `Create`):

```go
	// Update rewrites an item's itemId/quantity/tier/weight in place. The
	// owning gachapon is never re-parented.
	Update(id uint32, itemId uint32, quantity uint32, tier string, weight uint32) error
```

and the impl (after `Create`):

```go
func (p *ProcessorImpl) Update(id uint32, itemId uint32, quantity uint32, tier string, weight uint32) error {
	return UpdateItem(p.db.WithContext(p.ctx), id, itemId, quantity, tier, weight)
}
```

`item/resource.go` — add the route after the POST line:

```go
			r.HandleFunc("/{itemId}", registerInput("update_gachapon_item", handleUpdateItem)).Methods(http.MethodPatch)
```

and the handler (mirrors `handleUpdateGachapon`'s shape; `rest.ParseItemId` already exists at `rest/handler.go:109`):

```go
func handleUpdateItem(d *rest.HandlerDependency, c *rest.HandlerContext, rm RestModel) http.HandlerFunc {
	return rest.ParseGachaponId(d.Logger(), func(gachaponId string) http.HandlerFunc {
		return rest.ParseItemId(d.Logger(), func(itemId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				err := NewProcessor(d.Logger(), d.Context(), d.DB()).Update(itemId, rm.ItemId, rm.Quantity, rm.Tier, rm.Weight)
				if err != nil {
					d.Logger().WithError(err).Errorf("Updating item [%d] for gachapon [%s].", itemId, gachaponId)
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}
				w.WriteHeader(http.StatusNoContent)
			}
		})
	})
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./item/ -run TestUpdateItem -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add services/atlas-reward-pools/atlas.com/reward-pools/item/
git commit -m "feat(reward-pools): PATCH endpoint for pool items"
```

### Task 2: Global-item PATCH endpoint

**Files:**
- Modify: `global/administrator.go` (add `UpdateItem`)
- Modify: `global/processor.go` (add `Update` to interface + impl)
- Modify: `global/resource.go` (add PATCH route + handler)
- Test: `global/update_test.go` (create)

**Interfaces:**
- Consumes: `test.CreateGlobalProcessor(t)` → `(global.Processor, *gorm.DB, func())`; `global.NewBuilder(tenantId, id)` with `SetItemId/SetQuantity/SetTier`.
- Produces: `PATCH /global-items/{itemId}` → 204; `Processor.Update(id uint32, itemId uint32, quantity uint32, tier string) error` (global items have **no weight** — always 0 in the roll, `reward/processor.go` `getMergedPool`).

- [ ] **Step 1: Write the failing test**

`global/update_test.go`:

```go
package global_test

import (
	"atlas-reward-pools/global"
	"atlas-reward-pools/test"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func TestUpdateGlobalItem(t *testing.T) {
	processor, db, cleanup := test.CreateGlobalProcessor(t)
	defer cleanup()

	m, err := global.NewBuilder(test.TestTenantId, 0).
		SetItemId(2000000).
		SetQuantity(1).
		SetTier("common").
		Build()
	if err != nil {
		t.Fatalf("Failed to build global item model: %v", err)
	}
	if err = global.CreateItem(db, m); err != nil {
		t.Fatalf("Failed to create global item: %v", err)
	}

	paged, err := processor.GetAll(model.Page{Number: 1, Size: 10})()
	if err != nil || len(paged.Items) != 1 {
		t.Fatalf("Expected 1 global item, got %d (err %v)", len(paged.Items), err)
	}
	id := paged.Items[0].Id()

	if err = processor.Update(id, 2000001, 5, "rare"); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	after, err := processor.GetAll(model.Page{Number: 1, Size: 10})()
	if err != nil || len(after.Items) != 1 {
		t.Fatalf("Expected 1 global item after update, got %d (err %v)", len(after.Items), err)
	}
	got := after.Items[0]
	if got.ItemId() != 2000001 || got.Quantity() != 5 || got.Tier() != "rare" {
		t.Errorf("Update not applied: itemId=%d qty=%d tier=%q", got.ItemId(), got.Quantity(), got.Tier())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./global/ -run TestUpdateGlobalItem -v`
Expected: FAIL — `processor.Update undefined`.

- [ ] **Step 3: Implement**

`global/administrator.go` — append:

```go
func UpdateItem(db *gorm.DB, id uint32, itemId uint32, quantity uint32, tier string) error {
	return db.Model(&entity{}).
		Where(&entity{ID: id}).
		Updates(map[string]interface{}{
			"item_id":  itemId,
			"quantity": quantity,
			"tier":     tier,
		}).Error
}
```

`global/processor.go` — interface method + impl (same shape as Task 1):

```go
	Update(id uint32, itemId uint32, quantity uint32, tier string) error
```

```go
func (p *ProcessorImpl) Update(id uint32, itemId uint32, quantity uint32, tier string) error {
	return UpdateItem(p.db.WithContext(p.ctx), id, itemId, quantity, tier)
}
```

`global/resource.go` — route after the POST line:

```go
			r.HandleFunc("/{itemId}", registerInput("update_global_item", handleUpdateGlobalItem)).Methods(http.MethodPatch)
```

handler (the URL var is parsed inline exactly like `handleDeleteGlobalItem` — this subrouter has no gachaponId):

```go
func handleUpdateGlobalItem(d *rest.HandlerDependency, c *rest.HandlerContext, rm RestModel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		itemIdStr := mux.Vars(r)["itemId"]
		itemId, err := strconv.ParseUint(itemIdStr, 10, 32)
		if err != nil {
			d.Logger().WithError(err).Errorf("Unable to parse itemId.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		err = NewProcessor(d.Logger(), d.Context(), d.DB()).Update(uint32(itemId), rm.ItemId, rm.Quantity, rm.Tier)
		if err != nil {
			d.Logger().WithError(err).Errorf("Updating global item [%d].", itemId)
			server.WriteErrorResponse(d.Logger())(w)(err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./global/ -run TestUpdateGlobalItem -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add services/atlas-reward-pools/atlas.com/reward-pools/global/
git commit -m "feat(reward-pools): PATCH endpoint for global pool items"
```

### Task 3: Pool PATCH covers npcIds

**Files:**
- Modify: `gachapon/administrator.go:41-50` (`UpdateGachapon` gains npcIds)
- Modify: `gachapon/processor.go:50-52` (`Update` signature)
- Modify: `gachapon/resource.go:122-131` (`handleUpdateGachapon` passes `rm.NpcIds`)
- Test: `gachapon/update_test.go` (create)

**Interfaces:**
- Consumes: `test.CreateGachaponProcessor(t)`; `gachapon.NewBuilder(tenantId uuid.UUID, id string)`.
- Produces: `Processor.Update(id string, name string, npcIds []uint32, commonWeight uint32, uncommonWeight uint32, rareWeight uint32) error`. The only existing caller is `resource.go:125` — no other call sites (verified by grep). `kind` remains un-updatable.

- [ ] **Step 1: Write the failing test**

`gachapon/update_test.go`:

```go
package gachapon_test

import (
	"atlas-reward-pools/gachapon"
	"atlas-reward-pools/test"
	"testing"
)

func TestUpdateGachaponNpcIds(t *testing.T) {
	processor, db, cleanup := test.CreateGachaponProcessor(t)
	defer cleanup()

	m, err := gachapon.NewBuilder(test.TestTenantId, "henesys").
		SetName("Henesys Gachapon").
		SetNpcIds([]uint32{9100100}).
		SetCommonWeight(70).
		SetUncommonWeight(25).
		SetRareWeight(5).
		Build()
	if err != nil {
		t.Fatalf("Failed to build gachapon model: %v", err)
	}
	if err = gachapon.CreateGachapon(db, m); err != nil {
		t.Fatalf("Failed to create gachapon: %v", err)
	}

	if err = processor.Update("henesys", "Henesys Gachapon", []uint32{9100100, 9100109}, 60, 30, 10); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, err := processor.GetById("henesys")
	if err != nil {
		t.Fatalf("GetById failed: %v", err)
	}
	if len(got.NpcIds()) != 2 || got.NpcIds()[0] != 9100100 || got.NpcIds()[1] != 9100109 {
		t.Errorf("npcIds not updated: %v", got.NpcIds())
	}
	if got.CommonWeight() != 60 {
		t.Errorf("weights not updated: common=%d", got.CommonWeight())
	}
	if got.Kind() != "gachapon" {
		t.Errorf("kind must be untouched: %q", got.Kind())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./gachapon/ -run TestUpdateGachaponNpcIds -v`
Expected: FAIL — `too many arguments in call to processor.Update`.

- [ ] **Step 3: Implement**

`gachapon/administrator.go` — replace `UpdateGachapon` (npcIds conversion copies `CreateGachapon`'s loop):

```go
func UpdateGachapon(db *gorm.DB, id string, name string, npcIds []uint32, commonWeight uint32, uncommonWeight uint32, rareWeight uint32) error {
	dbNpcIds := make(int64Array, len(npcIds))
	for i, nid := range npcIds {
		dbNpcIds[i] = int64(nid)
	}
	return db.Model(&entity{}).
		Where(&entity{ID: id}).
		Updates(map[string]interface{}{
			"name":            name,
			"npc_ids":         dbNpcIds,
			"common_weight":   commonWeight,
			"uncommon_weight": uncommonWeight,
			"rare_weight":     rareWeight,
		}).Error
}
```

`gachapon/processor.go` — update the interface's `Update` line and the impl:

```go
func (p *ProcessorImpl) Update(id string, name string, npcIds []uint32, commonWeight uint32, uncommonWeight uint32, rareWeight uint32) error {
	return UpdateGachapon(p.db.WithContext(p.ctx), id, name, npcIds, commonWeight, uncommonWeight, rareWeight)
}
```

`gachapon/resource.go:125`:

```go
			err := NewProcessor(d.Logger(), d.Context(), d.DB()).Update(gachaponId, rm.Name, rm.NpcIds, rm.CommonWeight, rm.UncommonWeight, rm.RareWeight)
```

- [ ] **Step 4: Run test + the package suite**

Run: `go test ./gachapon/ -v`
Expected: PASS (including pre-existing tests).

- [ ] **Step 5: Commit**

```bash
git add services/atlas-reward-pools/atlas.com/reward-pools/gachapon/
git commit -m "feat(reward-pools): pool PATCH updates npcIds"
```

### Task 4: Kind-aware prize pool + weight on the wire

**Files:**
- Modify: `reward/builder.go`, `reward/model.go` (add `weight`)
- Modify: `reward/rest.go` (add `Weight` json field)
- Modify: `reward/processor.go` (`GetPrizePool` branches on kind; thread `pi.Weight`)
- Test: `reward/prize_pool_test.go` (create)

**Interfaces:**
- Consumes: `gachapon.KindIncubator` const (already used by `SelectReward`); `item.Processor.GetByGachaponId`.
- Produces: `reward.Model.Weight() uint32`; `GET /gachapons/{id}/prize-pool` returns weighted rows (tier `""`) for incubator pools and includes `"weight"` for all rows. Additive for the existing atlas-channel consumer (`channel/incubator/requests.go` decodes `rewards/select`, unaffected).

- [ ] **Step 1: Write the failing test**

`reward/prize_pool_test.go`:

```go
package reward_test

import (
	"atlas-reward-pools/gachapon"
	"atlas-reward-pools/item"
	"atlas-reward-pools/test"
	"testing"
)

func TestGetPrizePoolIncubator(t *testing.T) {
	processor, db, cleanup := test.CreateRewardProcessor(t)
	defer cleanup()

	g, err := gachapon.NewBuilder(test.TestTenantId, "4170001").
		SetName("Pigmy Egg (Victoria)").
		SetNpcIds([]uint32{1012004}).
		SetKind(gachapon.KindIncubator).
		Build()
	if err != nil {
		t.Fatalf("Failed to build incubator pool: %v", err)
	}
	if err = gachapon.CreateGachapon(db, g); err != nil {
		t.Fatalf("Failed to create incubator pool: %v", err)
	}

	for _, spec := range []struct {
		itemId uint32
		weight uint32
	}{{2000000, 50}, {1302000, 5}} {
		m, err := item.NewBuilder(test.TestTenantId, 0).
			SetGachaponId("4170001").
			SetItemId(spec.itemId).
			SetQuantity(1).
			SetWeight(spec.weight).
			Build()
		if err != nil {
			t.Fatalf("Failed to build item: %v", err)
		}
		if err = item.CreateItem(db, m); err != nil {
			t.Fatalf("Failed to create item: %v", err)
		}
	}

	pool, err := processor.GetPrizePool("4170001", "")
	if err != nil {
		t.Fatalf("GetPrizePool failed: %v", err)
	}
	if len(pool) != 2 {
		t.Fatalf("Incubator prize pool must return the weighted items, got %d", len(pool))
	}
	weights := map[uint32]uint32{}
	for _, m := range pool {
		if m.Tier() != "" {
			t.Errorf("Incubator rows carry no tier, got %q", m.Tier())
		}
		weights[m.ItemId()] = m.Weight()
	}
	if weights[2000000] != 50 || weights[1302000] != 5 {
		t.Errorf("Weights not threaded: %v", weights)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./reward/ -run TestGetPrizePoolIncubator -v`
Expected: FAIL — `m.Weight undefined` (compile) or, once compiling, `got 0` rows.

- [ ] **Step 3: Implement**

`reward/model.go` — add the field + getter:

```go
type Model struct {
	itemId     uint32
	quantity   uint32
	tier       string
	weight     uint32
	gachaponId string
}

func (m Model) Weight() uint32 {
	return m.weight
}
```

`reward/builder.go` — add `weight uint32` to the struct, a setter, and thread it in `Build()`:

```go
func (b *Builder) SetWeight(weight uint32) *Builder {
	b.weight = weight
	return b
}
```

`reward/rest.go` — add to `RestModel` (after `Tier`) and to `Transform`:

```go
	Weight     uint32 `json:"weight"`
```

```go
		Weight:     m.Weight(),
```

`reward/processor.go` — replace `GetPrizePool` (branch mirrors `SelectReward`'s `KindIncubator` branch; the gachapon path also threads `pi.Weight` so a weighted gachapon tier renders truthfully):

```go
func (p *ProcessorImpl) GetPrizePool(gachaponId string, tier string) ([]Model, error) {
	g, err := gachapon.NewProcessor(p.l, p.ctx, p.db).GetById(gachaponId)
	if err != nil {
		return nil, err
	}

	if g.Kind() == gachapon.KindIncubator {
		// Incubator pools roll the whole machine weighted by item.Weight —
		// they have no tiers and never merge the global pool (SelectReward).
		machineItems, err := item.NewProcessor(p.l, p.ctx, p.db).GetByGachaponId(gachaponId)()
		if err != nil {
			return nil, err
		}
		var results []Model
		for _, mi := range machineItems {
			results = append(results, NewBuilder(gachaponId).
				SetItemId(mi.ItemId()).
				SetQuantity(mi.Quantity()).
				SetWeight(mi.Weight()).
				Build())
		}
		return results, nil
	}

	tiers := []string{"common", "uncommon", "rare"}
	if tier != "" {
		tiers = []string{tier}
	}

	var results []Model
	for _, t := range tiers {
		pool, err := p.getMergedPool(gachaponId, t)
		if err != nil {
			return nil, err
		}
		for _, pi := range pool {
			results = append(results, NewBuilder(gachaponId).
				SetItemId(pi.ItemId).
				SetQuantity(pi.Quantity).
				SetTier(t).
				SetWeight(pi.Weight).
				Build())
		}
	}
	return results, nil
}
```

- [ ] **Step 4: Run the full module gate**

```bash
cd services/atlas-reward-pools/atlas.com/reward-pools
go build ./... && go vet ./... && go test -race ./...
cd "$(git rev-parse --show-toplevel)"
docker buildx bake atlas-reward-pools
tools/redis-key-guard.sh && tools/goroutine-guard.sh
```
Expected: all clean. (`docker buildx bake` is mandatory — CLAUDE.md Build & Verification.)

- [ ] **Step 5: Commit**

```bash
git add services/atlas-reward-pools/atlas.com/reward-pools/reward/
git commit -m "fix(reward-pools): kind-aware prize pool; weight on the wire"
```

---

# Phase 2 — Frontend foundation

All paths below are under `services/atlas-ui/src/`. Before any `npm` command: `source ~/.nvm/nvm.sh && nvm use 22` in `services/atlas-ui`.

### Task 5: Reward-pool types + service

**Files:**
- Create: `types/models/reward-pool.ts`, `types/models/reward-pool-item.ts`, `types/models/global-reward-item.ts`
- Create: `services/api/reward-pools.service.ts`
- Modify: `types/models/index.ts` (export the three new modules; keep the old gachapon exports until Task 12)
- Test: `services/api/__tests__/reward-pools.service.test.ts`

**Interfaces:**
- Produces (consumed by every later task):

```ts
// types/models/reward-pool.ts
export type RewardPoolKind = "gachapon" | "incubator";
export interface RewardPoolAttributes {
  name: string;
  kind: RewardPoolKind;
  npcIds: number[];
  commonWeight: number;
  uncommonWeight: number;
  rareWeight: number;
}
export interface RewardPoolData { id: string; type: string; attributes: RewardPoolAttributes; }

// types/models/reward-pool-item.ts  (mirrors item/rest.go; id = numeric record id as string)
export interface RewardPoolItemAttributes {
  gachaponId: string;
  itemId: number;
  quantity: number;
  tier: string;      // "" on incubator items
  weight: number;    // 0 on classic gachapon items
}
export interface RewardPoolItemData { id: string; type: string; attributes: RewardPoolItemAttributes; }

// types/models/global-reward-item.ts (mirrors global/rest.go — no weight)
export interface GlobalRewardItemAttributes { itemId: number; quantity: number; tier: string; }
export interface GlobalRewardItemData { id: string; type: string; attributes: GlobalRewardItemAttributes; }
```

- `rewardPoolsService` methods (exact):
  - `getAllPools(options?) → Promise<RewardPoolData[]>` — `fetchAll` on `/api/gachapons` (drain-all; the collection is small, tabs need exact counts)
  - `getPoolById(id) → Promise<RewardPoolData>`
  - `createPool(id: string | undefined, attrs: RewardPoolAttributes) → Promise<void>` — envelope `{data: {…(id ? {id} : {}), type: "gachapons", attributes: attrs}}` (client-supplied id is honored by `handleCreateGachapon` — `resource.go:93` `NewBuilder(t.Id(), rm.Id)`; incubator creation passes the egg item id)
  - `updatePool(id, attrs) → Promise<void>` — PATCH `/api/gachapons/{id}`, envelope with id
  - `removePool(id) → Promise<void>`
  - `getItems(poolId) → Promise<RewardPoolItemData[]>` — `fetchAll` on `/api/gachapons/{poolId}/items`
  - `createItem(poolId, attrs: Omit<RewardPoolItemAttributes, "gachaponId">) → Promise<void>` — type `"gachapon-items"`
  - `updateItem(poolId, itemRecordId, attrs) → Promise<void>` — PATCH `/api/gachapons/{poolId}/items/{itemRecordId}`
  - `removeItem(poolId, itemRecordId) → Promise<void>`
  - `getGlobalItems() → Promise<GlobalRewardItemData[]>` — `fetchAll` on `/api/global-items`
  - `createGlobalItem(attrs: GlobalRewardItemAttributes)`, `updateGlobalItem(itemRecordId, attrs)`, `removeGlobalItem(itemRecordId)` — type `"global-gachapon-items"`

- [ ] **Step 1: Write the failing test**

`services/api/__tests__/reward-pools.service.test.ts`:

```ts
import { describe, it, expect, vi, beforeEach } from "vitest";
import { rewardPoolsService } from "../reward-pools.service";
import { api } from "@/lib/api/client";
import { fetchAll } from "@/services/api/pagination";

vi.mock("@/lib/api/client", () => ({
  api: { getOne: vi.fn(), post: vi.fn(), patch: vi.fn(), delete: vi.fn() },
}));
vi.mock("@/services/api/pagination", () => ({
  fetchAll: vi.fn().mockResolvedValue([]),
}));

describe("rewardPoolsService", () => {
  beforeEach(() => vi.clearAllMocks());

  it("getAllPools drains /api/gachapons", async () => {
    await rewardPoolsService.getAllPools();
    expect(fetchAll).toHaveBeenCalledWith("/api/gachapons", undefined, undefined);
  });

  it("createPool posts an id-carrying JSON:API envelope for incubators", async () => {
    await rewardPoolsService.createPool("4170001", {
      name: "Pigmy Egg (Victoria)", kind: "incubator", npcIds: [1012004],
      commonWeight: 0, uncommonWeight: 0, rareWeight: 0,
    });
    expect(api.post).toHaveBeenCalledWith("/api/gachapons", {
      data: {
        id: "4170001",
        type: "gachapons",
        attributes: expect.objectContaining({ kind: "incubator" }),
      },
    });
  });

  it("createPool omits id when not supplied", async () => {
    await rewardPoolsService.createPool(undefined, {
      name: "Henesys", kind: "gachapon", npcIds: [9100100],
      commonWeight: 70, uncommonWeight: 25, rareWeight: 5,
    });
    const body = (api.post as ReturnType<typeof vi.fn>).mock.calls[0][1];
    expect(body.data.id).toBeUndefined();
  });

  it("updatePool PATCHes with envelope", async () => {
    const attrs = { name: "Henesys", kind: "gachapon" as const, npcIds: [9100100], commonWeight: 60, uncommonWeight: 30, rareWeight: 10 };
    await rewardPoolsService.updatePool("henesys", attrs);
    expect(api.patch).toHaveBeenCalledWith("/api/gachapons/henesys", {
      data: { id: "henesys", type: "gachapons", attributes: attrs },
    });
  });

  it("item CRUD targets the nested collection", async () => {
    await rewardPoolsService.getItems("4170001");
    expect(fetchAll).toHaveBeenCalledWith("/api/gachapons/4170001/items");

    await rewardPoolsService.createItem("4170001", { itemId: 2000000, quantity: 1, tier: "", weight: 50 });
    expect(api.post).toHaveBeenCalledWith("/api/gachapons/4170001/items", {
      data: { type: "gachapon-items", attributes: { itemId: 2000000, quantity: 1, tier: "", weight: 50 } },
    });

    await rewardPoolsService.updateItem("4170001", "12", { itemId: 2000001, quantity: 2, tier: "", weight: 75 });
    expect(api.patch).toHaveBeenCalledWith("/api/gachapons/4170001/items/12", {
      data: { id: "12", type: "gachapon-items", attributes: { itemId: 2000001, quantity: 2, tier: "", weight: 75 } },
    });

    await rewardPoolsService.removeItem("4170001", "12");
    expect(api.delete).toHaveBeenCalledWith("/api/gachapons/4170001/items/12");
  });

  it("global item CRUD targets /api/global-items", async () => {
    await rewardPoolsService.createGlobalItem({ itemId: 2000000, quantity: 1, tier: "common" });
    expect(api.post).toHaveBeenCalledWith("/api/global-items", {
      data: { type: "global-gachapon-items", attributes: { itemId: 2000000, quantity: 1, tier: "common" } },
    });
    await rewardPoolsService.removeGlobalItem("3");
    expect(api.delete).toHaveBeenCalledWith("/api/global-items/3");
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npm run test -- reward-pools.service`
Expected: FAIL — cannot resolve `../reward-pools.service`.

- [ ] **Step 3: Implement**

Create the three type files exactly as in **Interfaces** above, then `services/api/reward-pools.service.ts`:

```ts
import { api } from "@/lib/api/client";
import { buildQueryString, type QueryOptions } from "@/lib/api/query-params";
import { fetchAll } from "@/services/api/pagination";
import type { RewardPoolData, RewardPoolAttributes } from "@/types/models/reward-pool";
import type { RewardPoolItemData, RewardPoolItemAttributes } from "@/types/models/reward-pool-item";
import type { GlobalRewardItemData, GlobalRewardItemAttributes } from "@/types/models/global-reward-item";

const BASE_PATH = "/api/gachapons"; // REST identity deliberately unchanged (design §5)
const GLOBAL_PATH = "/api/global-items";

export const REWARD_POOL_TYPE = "gachapons";
export const REWARD_POOL_ITEM_TYPE = "gachapon-items";
export const GLOBAL_REWARD_ITEM_TYPE = "global-gachapon-items";

export const rewardPoolsService = {
  /** Drain the whole pool collection (small: machines + ten eggs) so kind tabs count exactly. */
  async getAllPools(options?: QueryOptions): Promise<RewardPoolData[]> {
    return fetchAll<RewardPoolData>(`${BASE_PATH}${buildQueryString(options)}`, undefined, options);
  },

  async getPoolById(id: string): Promise<RewardPoolData> {
    return api.getOne<RewardPoolData>(`${BASE_PATH}/${id}`);
  },

  /** id is client-supplied for incubator pools (the egg item id); omitted for classic gachapons. */
  async createPool(id: string | undefined, attributes: RewardPoolAttributes): Promise<void> {
    await api.post(BASE_PATH, { data: { ...(id !== undefined ? { id } : {}), type: REWARD_POOL_TYPE, attributes } });
  },

  async updatePool(id: string, attributes: RewardPoolAttributes): Promise<void> {
    await api.patch(`${BASE_PATH}/${id}`, { data: { id, type: REWARD_POOL_TYPE, attributes } });
  },

  async removePool(id: string): Promise<void> {
    await api.delete(`${BASE_PATH}/${id}`);
  },

  async getItems(poolId: string): Promise<RewardPoolItemData[]> {
    return fetchAll<RewardPoolItemData>(`${BASE_PATH}/${poolId}/items`);
  },

  async createItem(poolId: string, attributes: Omit<RewardPoolItemAttributes, "gachaponId">): Promise<void> {
    await api.post(`${BASE_PATH}/${poolId}/items`, { data: { type: REWARD_POOL_ITEM_TYPE, attributes } });
  },

  async updateItem(poolId: string, itemRecordId: string, attributes: Omit<RewardPoolItemAttributes, "gachaponId">): Promise<void> {
    await api.patch(`${BASE_PATH}/${poolId}/items/${itemRecordId}`, {
      data: { id: itemRecordId, type: REWARD_POOL_ITEM_TYPE, attributes },
    });
  },

  async removeItem(poolId: string, itemRecordId: string): Promise<void> {
    await api.delete(`${BASE_PATH}/${poolId}/items/${itemRecordId}`);
  },

  async getGlobalItems(): Promise<GlobalRewardItemData[]> {
    return fetchAll<GlobalRewardItemData>(GLOBAL_PATH);
  },

  async createGlobalItem(attributes: GlobalRewardItemAttributes): Promise<void> {
    await api.post(GLOBAL_PATH, { data: { type: GLOBAL_REWARD_ITEM_TYPE, attributes } });
  },

  async updateGlobalItem(itemRecordId: string, attributes: GlobalRewardItemAttributes): Promise<void> {
    await api.patch(`${GLOBAL_PATH}/${itemRecordId}`, {
      data: { id: itemRecordId, type: GLOBAL_REWARD_ITEM_TYPE, attributes },
    });
  },

  async removeGlobalItem(itemRecordId: string): Promise<void> {
    await api.delete(`${GLOBAL_PATH}/${itemRecordId}`);
  },
};
```

Add to `types/models/index.ts`:

```ts
export * from './reward-pool';
export * from './reward-pool-item';
export * from './global-reward-item';
```

- [ ] **Step 4: Run test to verify it passes**

Run: `npm run test -- reward-pools.service`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/types/models/ services/atlas-ui/src/services/api/
git commit -m "feat(ui): reward-pools types + service"
```

### Task 6: React Query hooks

**Files:**
- Create: `lib/hooks/api/useRewardPools.ts`
- Modify: `lib/hooks/api/index.ts` (export)
- Test: `lib/hooks/api/__tests__/useRewardPools.test.tsx`

**Interfaces:**
- Consumes: `rewardPoolsService` (Task 5); `useTenant` from `@/context/tenant-context`.
- Produces:

```ts
export const rewardPoolKeys = {
  all: ['reward-pools'] as const,
  lists: () => [...rewardPoolKeys.all, 'list'] as const,
  list: () => [...rewardPoolKeys.lists()] as const,
  details: () => [...rewardPoolKeys.all, 'detail'] as const,
  detail: (id: string) => [...rewardPoolKeys.details(), id] as const,
  items: (poolId: string) => [...rewardPoolKeys.all, 'items', poolId] as const,
  globalItems: () => [...rewardPoolKeys.all, 'global-items'] as const,
};
// queries
useRewardPools(): UseQueryResult<RewardPoolData[], Error>
useRewardPool(id: string): UseQueryResult<RewardPoolData, Error>
useRewardPoolItems(poolId: string): UseQueryResult<RewardPoolItemData[], Error>
useGlobalRewardItems(): UseQueryResult<GlobalRewardItemData[], Error>
// mutations (all invalidate the affected keys onSettled)
useCreateRewardPool()   // vars: { id?: string; attributes: RewardPoolAttributes }        → invalidates lists()
useUpdateRewardPool()   // vars: { id: string; attributes: RewardPoolAttributes }          → invalidates lists() + detail(id)
useDeleteRewardPool()   // vars: { id: string }                                            → invalidates lists()
useCreatePoolItem()     // vars: { poolId: string; attributes: Omit<RewardPoolItemAttributes,"gachaponId"> } → invalidates items(poolId)
useUpdatePoolItem()     // vars: { poolId: string; itemRecordId: string; attributes: … }   → invalidates items(poolId)
useDeletePoolItem()     // vars: { poolId: string; itemRecordId: string }                  → invalidates items(poolId)
useCreateGlobalItem()   // vars: { attributes: GlobalRewardItemAttributes }                → invalidates globalItems()
useUpdateGlobalItem()   // vars: { itemRecordId: string; attributes: GlobalRewardItemAttributes } → invalidates globalItems()
useDeleteGlobalItem()   // vars: { itemRecordId: string }                                  → invalidates globalItems()
```

- [ ] **Step 1: Write the failing test**

`lib/hooks/api/__tests__/useRewardPools.test.tsx`:

```tsx
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { useRewardPools, useCreatePoolItem, rewardPoolKeys } from "../useRewardPools";
import { rewardPoolsService } from "@/services/api/reward-pools.service";

vi.mock("@/services/api/reward-pools.service", () => ({
  rewardPoolsService: {
    getAllPools: vi.fn().mockResolvedValue([]),
    createItem: vi.fn().mockResolvedValue(undefined),
  },
}));
vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({ activeTenant: { id: "t1", attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 } } }),
}));

function wrapper(qc: QueryClient) {
  return ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={qc}>{children}</QueryClientProvider>
  );
}

describe("useRewardPools", () => {
  beforeEach(() => vi.clearAllMocks());

  it("fetches the drained pool list", async () => {
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const { result } = renderHook(() => useRewardPools(), { wrapper: wrapper(qc) });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(rewardPoolsService.getAllPools).toHaveBeenCalled();
  });

  it("useCreatePoolItem invalidates the pool's items key", async () => {
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const spy = vi.spyOn(qc, "invalidateQueries");
    const { result } = renderHook(() => useCreatePoolItem(), { wrapper: wrapper(qc) });
    result.current.mutate({ poolId: "4170001", attributes: { itemId: 2000000, quantity: 1, tier: "", weight: 50 } });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(spy).toHaveBeenCalledWith({ queryKey: rewardPoolKeys.items("4170001") });
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npm run test -- useRewardPools`
Expected: FAIL — cannot resolve `../useRewardPools`.

- [ ] **Step 3: Implement**

`lib/hooks/api/useRewardPools.ts` — queries follow `useGachapons.ts`'s pattern (`enabled: !!activeTenant`, `gcTime: 10 * 60 * 1000`, `useCache: false` is not needed since the service has no cache option); mutations follow the `useNpcs.ts` mutation pattern:

```ts
import { useMutation, useQuery, useQueryClient, type UseMutationResult, type UseQueryResult } from '@tanstack/react-query';
import { rewardPoolsService } from '@/services/api/reward-pools.service';
import { useTenant } from '@/context/tenant-context';
import type { RewardPoolData, RewardPoolAttributes } from '@/types/models/reward-pool';
import type { RewardPoolItemData, RewardPoolItemAttributes } from '@/types/models/reward-pool-item';
import type { GlobalRewardItemData, GlobalRewardItemAttributes } from '@/types/models/global-reward-item';

export const rewardPoolKeys = {
  all: ['reward-pools'] as const,
  lists: () => [...rewardPoolKeys.all, 'list'] as const,
  list: () => [...rewardPoolKeys.lists()] as const,
  details: () => [...rewardPoolKeys.all, 'detail'] as const,
  detail: (id: string) => [...rewardPoolKeys.details(), id] as const,
  items: (poolId: string) => [...rewardPoolKeys.all, 'items', poolId] as const,
  globalItems: () => [...rewardPoolKeys.all, 'global-items'] as const,
};

export function useRewardPools(): UseQueryResult<RewardPoolData[], Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: rewardPoolKeys.list(),
    queryFn: () => rewardPoolsService.getAllPools(),
    enabled: !!activeTenant,
    gcTime: 10 * 60 * 1000,
  });
}

export function useRewardPool(id: string): UseQueryResult<RewardPoolData, Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: rewardPoolKeys.detail(id),
    queryFn: () => rewardPoolsService.getPoolById(id),
    enabled: !!activeTenant && !!id,
    gcTime: 10 * 60 * 1000,
  });
}

export function useRewardPoolItems(poolId: string): UseQueryResult<RewardPoolItemData[], Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: rewardPoolKeys.items(poolId),
    queryFn: () => rewardPoolsService.getItems(poolId),
    enabled: !!activeTenant && !!poolId,
    gcTime: 10 * 60 * 1000,
  });
}

export function useGlobalRewardItems(): UseQueryResult<GlobalRewardItemData[], Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: rewardPoolKeys.globalItems(),
    queryFn: () => rewardPoolsService.getGlobalItems(),
    enabled: !!activeTenant,
    gcTime: 10 * 60 * 1000,
  });
}

export function useCreateRewardPool(): UseMutationResult<void, Error, { id?: string; attributes: RewardPoolAttributes }> {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, attributes }) => rewardPoolsService.createPool(id, attributes),
    onSettled: () => qc.invalidateQueries({ queryKey: rewardPoolKeys.lists() }),
  });
}

export function useUpdateRewardPool(): UseMutationResult<void, Error, { id: string; attributes: RewardPoolAttributes }> {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, attributes }) => rewardPoolsService.updatePool(id, attributes),
    onSettled: (_d, _e, { id }) => {
      qc.invalidateQueries({ queryKey: rewardPoolKeys.lists() });
      qc.invalidateQueries({ queryKey: rewardPoolKeys.detail(id) });
    },
  });
}

export function useDeleteRewardPool(): UseMutationResult<void, Error, { id: string }> {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id }) => rewardPoolsService.removePool(id),
    onSettled: () => qc.invalidateQueries({ queryKey: rewardPoolKeys.lists() }),
  });
}

export function useCreatePoolItem(): UseMutationResult<void, Error, { poolId: string; attributes: Omit<RewardPoolItemAttributes, 'gachaponId'> }> {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ poolId, attributes }) => rewardPoolsService.createItem(poolId, attributes),
    onSettled: (_d, _e, { poolId }) => qc.invalidateQueries({ queryKey: rewardPoolKeys.items(poolId) }),
  });
}

export function useUpdatePoolItem(): UseMutationResult<void, Error, { poolId: string; itemRecordId: string; attributes: Omit<RewardPoolItemAttributes, 'gachaponId'> }> {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ poolId, itemRecordId, attributes }) => rewardPoolsService.updateItem(poolId, itemRecordId, attributes),
    onSettled: (_d, _e, { poolId }) => qc.invalidateQueries({ queryKey: rewardPoolKeys.items(poolId) }),
  });
}

export function useDeletePoolItem(): UseMutationResult<void, Error, { poolId: string; itemRecordId: string }> {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ poolId, itemRecordId }) => rewardPoolsService.removeItem(poolId, itemRecordId),
    onSettled: (_d, _e, { poolId }) => qc.invalidateQueries({ queryKey: rewardPoolKeys.items(poolId) }),
  });
}

export function useCreateGlobalItem(): UseMutationResult<void, Error, { attributes: GlobalRewardItemAttributes }> {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ attributes }) => rewardPoolsService.createGlobalItem(attributes),
    onSettled: () => qc.invalidateQueries({ queryKey: rewardPoolKeys.globalItems() }),
  });
}

export function useUpdateGlobalItem(): UseMutationResult<void, Error, { itemRecordId: string; attributes: GlobalRewardItemAttributes }> {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ itemRecordId, attributes }) => rewardPoolsService.updateGlobalItem(itemRecordId, attributes),
    onSettled: () => qc.invalidateQueries({ queryKey: rewardPoolKeys.globalItems() }),
  });
}

export function useDeleteGlobalItem(): UseMutationResult<void, Error, { itemRecordId: string }> {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ itemRecordId }) => rewardPoolsService.removeGlobalItem(itemRecordId),
    onSettled: () => qc.invalidateQueries({ queryKey: rewardPoolKeys.globalItems() }),
  });
}
```

Export from `lib/hooks/api/index.ts` alongside the existing exports:

```ts
export * from './useRewardPools';
```

- [ ] **Step 4: Run test to verify it passes**

Run: `npm run test -- useRewardPools`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/lib/hooks/api/
git commit -m "feat(ui): reward-pools React Query hooks"
```

### Task 7: Chance-computation util

**Files:**
- Create: `lib/utils/reward-pool-chance.ts`
- Test: `lib/utils/__tests__/reward-pool-chance.test.ts`

**Interfaces:**
- Produces (consumed by Task 11's detail page):

```ts
export interface ChanceRow { key: string; chance: number; excluded: boolean; }
/** Incubator pools: weight / Σweight. Empty or zero-total pool → all 0. */
export function incubatorChances(items: { id: string; weight: number }[]): Map<string, number>;
/**
 * Gachapon pools — mirrors reward/processor.go selectTier + selectItem exactly:
 * tierChance = tierWeight / (common+uncommon+rare); within the tier's merged
 * pool (machine items + global items, global always weight 0): if Σweight > 0
 * → weight-proportional (zero-weight rows get chance 0 AND excluded=true);
 * else uniform 1/N.
 */
export function gachaponChances(
  tierWeights: { common: number; uncommon: number; rare: number },
  rows: { key: string; tier: "common" | "uncommon" | "rare"; weight: number }[],
): Map<string, ChanceRow>;
/** True when a tier mixes weighted and zero-weight rows — the excluded-rows footgun. */
export function tierHasMixedWeights(rows: { tier: string; weight: number }[], tier: string): boolean;
```

- [ ] **Step 1: Write the failing test**

`lib/utils/__tests__/reward-pool-chance.test.ts`:

```ts
import { describe, it, expect } from "vitest";
import { incubatorChances, gachaponChances, tierHasMixedWeights } from "../reward-pool-chance";

describe("incubatorChances", () => {
  it("divides weight by total", () => {
    const m = incubatorChances([{ id: "a", weight: 75 }, { id: "b", weight: 25 }]);
    expect(m.get("a")).toBeCloseTo(0.75);
    expect(m.get("b")).toBeCloseTo(0.25);
  });
  it("zero-total pool yields all zeros", () => {
    const m = incubatorChances([{ id: "a", weight: 0 }]);
    expect(m.get("a")).toBe(0);
  });
});

describe("gachaponChances", () => {
  const tw = { common: 70, uncommon: 25, rare: 5 };

  it("uniform within an unweighted tier, scaled by tier chance", () => {
    const m = gachaponChances(tw, [
      { key: "a", tier: "common", weight: 0 },
      { key: "b", tier: "common", weight: 0 },
    ]);
    expect(m.get("a")!.chance).toBeCloseTo(0.7 / 2);
    expect(m.get("a")!.excluded).toBe(false);
  });

  it("weight-proportional when any row in the tier is weighted; zero-weight rows excluded", () => {
    const m = gachaponChances(tw, [
      { key: "w", tier: "rare", weight: 10 },
      { key: "z", tier: "rare", weight: 0 },
    ]);
    expect(m.get("w")!.chance).toBeCloseTo(0.05);
    expect(m.get("z")!.chance).toBe(0);
    expect(m.get("z")!.excluded).toBe(true);
  });

  it("zero tier-weight sum yields zeros", () => {
    const m = gachaponChances({ common: 0, uncommon: 0, rare: 0 }, [{ key: "a", tier: "common", weight: 0 }]);
    expect(m.get("a")!.chance).toBe(0);
  });
});

describe("tierHasMixedWeights", () => {
  it("detects a mixed tier", () => {
    const rows = [
      { tier: "rare", weight: 10 },
      { tier: "rare", weight: 0 },
      { tier: "common", weight: 0 },
    ];
    expect(tierHasMixedWeights(rows, "rare")).toBe(true);
    expect(tierHasMixedWeights(rows, "common")).toBe(false);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npm run test -- reward-pool-chance`
Expected: FAIL — cannot resolve `../reward-pool-chance`.

- [ ] **Step 3: Implement**

`lib/utils/reward-pool-chance.ts`:

```ts
// Mirrors services/atlas-reward-pools reward/processor.go exactly:
// selectTier (weighted over the three tier weights) then selectItem
// (weight-proportional when the merged pool's Σweight > 0, else uniform).
// Global items always enter the merged pool with weight 0.

export interface ChanceRow {
  key: string;
  chance: number;
  /** true when weighted rows exist in the tier and this zero-weight row can never win */
  excluded: boolean;
}

export function incubatorChances(items: { id: string; weight: number }[]): Map<string, number> {
  const total = items.reduce((s, i) => s + i.weight, 0);
  return new Map(items.map((i) => [i.id, total > 0 ? i.weight / total : 0]));
}

export function gachaponChances(
  tierWeights: { common: number; uncommon: number; rare: number },
  rows: { key: string; tier: 'common' | 'uncommon' | 'rare'; weight: number }[],
): Map<string, ChanceRow> {
  const tierTotal = tierWeights.common + tierWeights.uncommon + tierWeights.rare;
  const result = new Map<string, ChanceRow>();
  for (const tier of ['common', 'uncommon', 'rare'] as const) {
    const tierRows = rows.filter((r) => r.tier === tier);
    if (tierRows.length === 0) continue;
    const tierChance = tierTotal > 0 ? tierWeights[tier] / tierTotal : 0;
    const weightSum = tierRows.reduce((s, r) => s + r.weight, 0);
    for (const r of tierRows) {
      const within = weightSum > 0 ? r.weight / weightSum : 1 / tierRows.length;
      result.set(r.key, {
        key: r.key,
        chance: tierChance * within,
        excluded: weightSum > 0 && r.weight === 0,
      });
    }
  }
  return result;
}

export function tierHasMixedWeights(rows: { tier: string; weight: number }[], tier: string): boolean {
  const tierRows = rows.filter((r) => r.tier === tier);
  return tierRows.some((r) => r.weight > 0) && tierRows.some((r) => r.weight === 0);
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `npm run test -- reward-pool-chance`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/lib/utils/
git commit -m "feat(ui): reward-pool chance math util"
```

### Task 8: Zod schemas

**Files:**
- Create: `lib/schemas/reward-pools.schema.ts`
- Test: `lib/schemas/__tests__/reward-pools.schema.test.ts`

**Interfaces:**
- Produces (consumed by Task 9's dialogs):

```ts
export const gachaponPoolSchema;    // { name: string≥1; npcIds: number[] of int>0; commonWeight,uncommonWeight,rareWeight: int≥0; refine Σ>0 }
export const incubatorPoolSchema;   // { eggItemId: int>0 (create only — becomes the pool id); name: string≥1; successNpcId: int>0 }
export const tierItemSchema;        // { itemId: int>0; quantity: int>0; tier: enum(common|uncommon|rare) }
export const weightItemSchema;      // { itemId: int>0; quantity: int>0; weight: int>0 }
export type GachaponPoolFormData = z.infer<typeof gachaponPoolSchema>;
export type IncubatorPoolFormData = z.infer<typeof incubatorPoolSchema>;
export type TierItemFormData = z.infer<typeof tierItemSchema>;
export type WeightItemFormData = z.infer<typeof weightItemSchema>;
```

- [ ] **Step 1: Write the failing test**

`lib/schemas/__tests__/reward-pools.schema.test.ts`:

```ts
import { describe, it, expect } from "vitest";
import { gachaponPoolSchema, incubatorPoolSchema, tierItemSchema, weightItemSchema } from "../reward-pools.schema";

describe("gachaponPoolSchema", () => {
  it("accepts a valid pool", () => {
    expect(gachaponPoolSchema.safeParse({ name: "Henesys", npcIds: [9100100], commonWeight: 70, uncommonWeight: 25, rareWeight: 5 }).success).toBe(true);
  });
  it("rejects an all-zero tier-weight sum", () => {
    expect(gachaponPoolSchema.safeParse({ name: "X", npcIds: [], commonWeight: 0, uncommonWeight: 0, rareWeight: 0 }).success).toBe(false);
  });
});

describe("incubatorPoolSchema", () => {
  it("requires a positive egg item id", () => {
    expect(incubatorPoolSchema.safeParse({ eggItemId: 4170001, name: "Pigmy Egg (Victoria)", successNpcId: 1012004 }).success).toBe(true);
    expect(incubatorPoolSchema.safeParse({ eggItemId: 0, name: "X", successNpcId: 1 }).success).toBe(false);
  });
});

describe("item schemas", () => {
  it("tierItemSchema enforces the tier enum", () => {
    expect(tierItemSchema.safeParse({ itemId: 2000000, quantity: 1, tier: "common" }).success).toBe(true);
    expect(tierItemSchema.safeParse({ itemId: 2000000, quantity: 1, tier: "epic" }).success).toBe(false);
  });
  it("weightItemSchema requires weight ≥ 1", () => {
    expect(weightItemSchema.safeParse({ itemId: 2000000, quantity: 1, weight: 50 }).success).toBe(true);
    expect(weightItemSchema.safeParse({ itemId: 2000000, quantity: 1, weight: 0 }).success).toBe(false);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npm run test -- reward-pools.schema`
Expected: FAIL — cannot resolve.

- [ ] **Step 3: Implement**

`lib/schemas/reward-pools.schema.ts`:

```ts
import { z } from "zod";

export const gachaponPoolSchema = z
  .object({
    name: z.string().min(1, "Name is required"),
    npcIds: z.array(z.number().int().positive()),
    commonWeight: z.number().int().min(0),
    uncommonWeight: z.number().int().min(0),
    rareWeight: z.number().int().min(0),
  })
  .refine((v) => v.commonWeight + v.uncommonWeight + v.rareWeight > 0, {
    message: "Tier weights must sum to more than zero",
    path: ["commonWeight"],
  });
export type GachaponPoolFormData = z.infer<typeof gachaponPoolSchema>;

export const incubatorPoolSchema = z.object({
  eggItemId: z.number().int().positive("Egg item id is required"),
  name: z.string().min(1, "Name is required"),
  successNpcId: z.number().int().positive("Success NPC id is required"),
});
export type IncubatorPoolFormData = z.infer<typeof incubatorPoolSchema>;

export const tierItemSchema = z.object({
  itemId: z.number().int().positive("Item id is required"),
  quantity: z.number().int().positive(),
  tier: z.enum(["common", "uncommon", "rare"]),
});
export type TierItemFormData = z.infer<typeof tierItemSchema>;

export const weightItemSchema = z.object({
  itemId: z.number().int().positive("Item id is required"),
  quantity: z.number().int().positive(),
  weight: z.number().int().positive("Weight must be at least 1"),
});
export type WeightItemFormData = z.infer<typeof weightItemSchema>;
```

- [ ] **Step 4: Run test to verify it passes**

Run: `npm run test -- reward-pools.schema`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/lib/schemas/
git commit -m "feat(ui): reward-pools zod schemas"
```

---

# Phase 3 — Components and pages

New components live in `components/features/reward-pools/`. Pattern sources: the retired incubator form (`git show 5da125b76~1:services/atlas-ui/src/pages/tenants-incubator-rewards-form.tsx`) for dialog-CRUD mechanics; `ItemNameCell` (`components/item-name-cell.tsx`, props `{ itemId: string; tenant: Tenant | null }`) for item-name resolution; `getAssetIconUrl(activeTenant.id, activeTenant.attributes.region, activeTenant.attributes.majorVersion, activeTenant.attributes.minorVersion, "item", numericId)` for icons (see `ItemHeader.tsx:28-35`).

### Task 9: Shared CRUD dialogs

**Files:**
- Create: `components/features/reward-pools/PoolFormDialog.tsx`
- Create: `components/features/reward-pools/PoolItemDialog.tsx`
- Test: `components/features/reward-pools/__tests__/PoolFormDialog.test.tsx`
- Test: `components/features/reward-pools/__tests__/PoolItemDialog.test.tsx`

**Interfaces:**
- Consumes: schemas (Task 8), hooks (Task 6), shadcn `Dialog`, `react-hook-form` + `zodResolver`, `sonner` toast, `createErrorFromUnknown` from `@/lib/api/errors`.
- Produces:

```tsx
// Create or edit a pool. mode="create" shows the kind selector (radio) first;
// mode="edit" locks kind + id and prefills from `pool`.
export function PoolFormDialog(props: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  mode: "create" | "edit";
  pool?: RewardPoolData;           // required when mode="edit"
}): JSX.Element;

// Add or edit one pool item. kind picks the schema: "gachapon" → tierItemSchema
// (tier select), "incubator" → weightItemSchema (weight input),
// "global" → tierItemSchema posted via the global-item mutations.
export function PoolItemDialog(props: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  kind: "gachapon" | "incubator" | "global";
  poolId?: string;                 // required unless kind="global"
  item?: RewardPoolItemData | GlobalRewardItemData;  // present → edit mode
}): JSX.Element;
```

- [ ] **Step 1: Write the failing tests**

`components/features/reward-pools/__tests__/PoolItemDialog.test.tsx`:

```tsx
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { PoolItemDialog } from "../PoolItemDialog";
import { rewardPoolsService } from "@/services/api/reward-pools.service";

vi.mock("@/services/api/reward-pools.service", () => ({
  rewardPoolsService: {
    createItem: vi.fn().mockResolvedValue(undefined),
    updateItem: vi.fn().mockResolvedValue(undefined),
    createGlobalItem: vi.fn().mockResolvedValue(undefined),
  },
}));
vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({ activeTenant: { id: "t1", attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 } } }),
}));

function renderDialog(props: Partial<Parameters<typeof PoolItemDialog>[0]> = {}) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <PoolItemDialog open onOpenChange={() => {}} kind="incubator" poolId="4170001" {...props} />
    </QueryClientProvider>,
  );
}

describe("PoolItemDialog", () => {
  beforeEach(() => vi.clearAllMocks());

  it("incubator mode shows a Weight field and no Tier select", () => {
    renderDialog();
    expect(screen.getByLabelText(/weight/i)).toBeInTheDocument();
    expect(screen.queryByLabelText(/tier/i)).not.toBeInTheDocument();
  });

  it("gachapon mode shows a Tier select and no Weight field", () => {
    renderDialog({ kind: "gachapon" });
    expect(screen.getByLabelText(/tier/i)).toBeInTheDocument();
    expect(screen.queryByLabelText(/weight/i)).not.toBeInTheDocument();
  });

  it("submits an incubator item with tier '' and the entered weight", async () => {
    const user = userEvent.setup();
    renderDialog();
    await user.type(screen.getByLabelText(/item id/i), "2000000");
    await user.type(screen.getByLabelText(/quantity/i), "1");
    await user.type(screen.getByLabelText(/weight/i), "50");
    await user.click(screen.getByRole("button", { name: /save|add/i }));
    await waitFor(() =>
      expect(rewardPoolsService.createItem).toHaveBeenCalledWith("4170001", {
        itemId: 2000000, quantity: 1, tier: "", weight: 50,
      }),
    );
  });

  it("rejects weight 0 before calling the service", async () => {
    const user = userEvent.setup();
    renderDialog();
    await user.type(screen.getByLabelText(/item id/i), "2000000");
    await user.type(screen.getByLabelText(/quantity/i), "1");
    await user.type(screen.getByLabelText(/weight/i), "0");
    await user.click(screen.getByRole("button", { name: /save|add/i }));
    await waitFor(() => expect(screen.getByText(/weight must be at least 1/i)).toBeInTheDocument());
    expect(rewardPoolsService.createItem).not.toHaveBeenCalled();
  });
});
```

`components/features/reward-pools/__tests__/PoolFormDialog.test.tsx`:

```tsx
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { PoolFormDialog } from "../PoolFormDialog";
import { rewardPoolsService } from "@/services/api/reward-pools.service";

vi.mock("@/services/api/reward-pools.service", () => ({
  rewardPoolsService: {
    createPool: vi.fn().mockResolvedValue(undefined),
    updatePool: vi.fn().mockResolvedValue(undefined),
  },
}));
vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({ activeTenant: { id: "t1", attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 } } }),
}));

function renderDialog(props: Partial<Parameters<typeof PoolFormDialog>[0]> = {}) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <PoolFormDialog open onOpenChange={() => {}} mode="create" {...props} />
    </QueryClientProvider>,
  );
}

describe("PoolFormDialog", () => {
  beforeEach(() => vi.clearAllMocks());

  it("create mode: choosing Incubator swaps tier-weight fields for egg fields", async () => {
    const user = userEvent.setup();
    renderDialog();
    await user.click(screen.getByRole("radio", { name: /incubator/i }));
    expect(screen.getByLabelText(/egg item id/i)).toBeInTheDocument();
    expect(screen.queryByLabelText(/common weight/i)).not.toBeInTheDocument();
  });

  it("creates an incubator pool with the egg id as the pool id and zero tier weights", async () => {
    const user = userEvent.setup();
    renderDialog();
    await user.click(screen.getByRole("radio", { name: /incubator/i }));
    await user.type(screen.getByLabelText(/egg item id/i), "4170001");
    await user.type(screen.getByLabelText(/name/i), "Pigmy Egg (Victoria)");
    await user.type(screen.getByLabelText(/success npc/i), "1012004");
    await user.click(screen.getByRole("button", { name: /create/i }));
    await waitFor(() =>
      expect(rewardPoolsService.createPool).toHaveBeenCalledWith("4170001", {
        name: "Pigmy Egg (Victoria)",
        kind: "incubator",
        npcIds: [1012004],
        commonWeight: 0, uncommonWeight: 0, rareWeight: 0,
      }),
    );
  });

  it("edit mode locks kind and prefills", () => {
    renderDialog({
      mode: "edit",
      pool: { id: "henesys", type: "gachapons", attributes: { name: "Henesys", kind: "gachapon", npcIds: [9100100], commonWeight: 70, uncommonWeight: 25, rareWeight: 5 } },
    });
    expect(screen.queryByRole("radio")).not.toBeInTheDocument();
    expect(screen.getByLabelText(/name/i)).toHaveValue("Henesys");
  });
});
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `npm run test -- reward-pools/`
Expected: FAIL — components unresolved.

- [ ] **Step 3: Implement `PoolItemDialog`**

`components/features/reward-pools/PoolItemDialog.tsx`:

```tsx
import { useEffect } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { toast } from "sonner";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { createErrorFromUnknown } from "@/lib/api/errors";
import { tierItemSchema, weightItemSchema, type TierItemFormData, type WeightItemFormData } from "@/lib/schemas/reward-pools.schema";
import { useCreatePoolItem, useUpdatePoolItem, useCreateGlobalItem, useUpdateGlobalItem } from "@/lib/hooks/api/useRewardPools";
import type { RewardPoolItemData } from "@/types/models/reward-pool-item";
import type { GlobalRewardItemData } from "@/types/models/global-reward-item";

interface PoolItemDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  kind: "gachapon" | "incubator" | "global";
  poolId?: string;
  item?: RewardPoolItemData | GlobalRewardItemData;
}

export function PoolItemDialog({ open, onOpenChange, kind, poolId, item }: PoolItemDialogProps) {
  const isEdit = !!item;
  const weighted = kind === "incubator";
  const schema = weighted ? weightItemSchema : tierItemSchema;

  const form = useForm<TierItemFormData | WeightItemFormData>({
    resolver: zodResolver(schema),
    defaultValues: item
      ? {
          itemId: item.attributes.itemId,
          quantity: item.attributes.quantity,
          ...(weighted
            ? { weight: (item as RewardPoolItemData).attributes.weight }
            : { tier: (item.attributes.tier || "common") as "common" | "uncommon" | "rare" }),
        }
      : { itemId: undefined, quantity: 1, ...(weighted ? { weight: undefined } : { tier: "common" as const }) },
  });
  useEffect(() => {
    if (open) form.reset(form.formState.defaultValues as never);
  }, [open]); // eslint-disable-line react-hooks/exhaustive-deps

  const createItem = useCreatePoolItem();
  const updateItem = useUpdatePoolItem();
  const createGlobal = useCreateGlobalItem();
  const updateGlobal = useUpdateGlobalItem();
  const pending = createItem.isPending || updateItem.isPending || createGlobal.isPending || updateGlobal.isPending;

  const onSubmit = form.handleSubmit(async (values) => {
    try {
      if (kind === "global") {
        const attrs = { itemId: values.itemId, quantity: values.quantity, tier: (values as TierItemFormData).tier };
        if (isEdit) await updateGlobal.mutateAsync({ itemRecordId: item!.id, attributes: attrs });
        else await createGlobal.mutateAsync({ attributes: attrs });
      } else {
        const attrs = weighted
          ? { itemId: values.itemId, quantity: values.quantity, tier: "", weight: (values as WeightItemFormData).weight }
          : { itemId: values.itemId, quantity: values.quantity, tier: (values as TierItemFormData).tier, weight: 0 };
        if (isEdit) await updateItem.mutateAsync({ poolId: poolId!, itemRecordId: item!.id, attributes: attrs });
        else await createItem.mutateAsync({ poolId: poolId!, attributes: attrs });
      }
      toast.success(isEdit ? "Item updated" : "Item added");
      onOpenChange(false);
    } catch (e) {
      toast.error(createErrorFromUnknown(e).message);
    }
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{isEdit ? "Edit Item" : "Add Item"}</DialogTitle>
        </DialogHeader>
        <form onSubmit={onSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="pi-itemId">Item Id</Label>
            <Input id="pi-itemId" type="number" {...form.register("itemId", { valueAsNumber: true })} />
            {form.formState.errors.itemId && <p className="text-sm text-destructive">{form.formState.errors.itemId.message}</p>}
          </div>
          <div className="space-y-2">
            <Label htmlFor="pi-quantity">Quantity</Label>
            <Input id="pi-quantity" type="number" {...form.register("quantity", { valueAsNumber: true })} />
            {form.formState.errors.quantity && <p className="text-sm text-destructive">{form.formState.errors.quantity.message}</p>}
          </div>
          {weighted ? (
            <div className="space-y-2">
              <Label htmlFor="pi-weight">Weight</Label>
              <Input id="pi-weight" type="number" {...form.register("weight" as const, { valueAsNumber: true })} />
              {"weight" in form.formState.errors && form.formState.errors.weight && (
                <p className="text-sm text-destructive">{form.formState.errors.weight.message}</p>
              )}
            </div>
          ) : (
            <div className="space-y-2">
              <Label htmlFor="pi-tier">Tier</Label>
              <Select
                value={form.watch("tier" as const)}
                onValueChange={(v) => form.setValue("tier" as const, v as "common" | "uncommon" | "rare")}
              >
                <SelectTrigger id="pi-tier" aria-label="Tier"><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="common">common</SelectItem>
                  <SelectItem value="uncommon">uncommon</SelectItem>
                  <SelectItem value="rare">rare</SelectItem>
                </SelectContent>
              </Select>
            </div>
          )}
          <DialogFooter>
            <Button type="submit" disabled={pending}>{isEdit ? "Save" : "Add"}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
```

(If the shadcn `Select` interplays badly with `getByLabelText` in jsdom, a native `<select aria-label="Tier">` styled via the `cn` helper is acceptable — the retired incubator form used plain inputs for the same reason.)

- [ ] **Step 4: Implement `PoolFormDialog`**

`components/features/reward-pools/PoolFormDialog.tsx`:

```tsx
import { useState, useEffect } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { toast } from "sonner";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { createErrorFromUnknown } from "@/lib/api/errors";
import {
  gachaponPoolSchema, incubatorPoolSchema,
  type GachaponPoolFormData, type IncubatorPoolFormData,
} from "@/lib/schemas/reward-pools.schema";
import { useCreateRewardPool, useUpdateRewardPool } from "@/lib/hooks/api/useRewardPools";
import type { RewardPoolData, RewardPoolKind } from "@/types/models/reward-pool";

interface PoolFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  mode: "create" | "edit";
  pool?: RewardPoolData;
}

export function PoolFormDialog({ open, onOpenChange, mode, pool }: PoolFormDialogProps) {
  const isEdit = mode === "edit";
  const [kind, setKind] = useState<RewardPoolKind>(pool?.attributes.kind ?? "gachapon");
  useEffect(() => {
    if (open) setKind(pool?.attributes.kind ?? "gachapon");
  }, [open, pool]);

  const createPool = useCreateRewardPool();
  const updatePool = useUpdateRewardPool();
  const pending = createPool.isPending || updatePool.isPending;

  const gachaponForm = useForm<GachaponPoolFormData>({
    resolver: zodResolver(gachaponPoolSchema),
    defaultValues: pool && pool.attributes.kind === "gachapon"
      ? { name: pool.attributes.name, npcIds: pool.attributes.npcIds, commonWeight: pool.attributes.commonWeight, uncommonWeight: pool.attributes.uncommonWeight, rareWeight: pool.attributes.rareWeight }
      : { name: "", npcIds: [], commonWeight: 70, uncommonWeight: 25, rareWeight: 5 },
  });
  const incubatorForm = useForm<IncubatorPoolFormData>({
    resolver: zodResolver(incubatorPoolSchema),
    defaultValues: pool && pool.attributes.kind === "incubator"
      ? { eggItemId: Number(pool.id), name: pool.attributes.name, successNpcId: pool.attributes.npcIds[0] ?? 0 }
      : { eggItemId: undefined, name: "", successNpcId: undefined },
  });

  // npcIds is edited as a comma-separated string for gachapons
  const [npcIdsText, setNpcIdsText] = useState((pool?.attributes.npcIds ?? []).join(", "));
  useEffect(() => {
    if (open) setNpcIdsText((pool?.attributes.npcIds ?? []).join(", "));
  }, [open, pool]);

  const submitGachapon = gachaponForm.handleSubmit(async (values) => {
    const npcIds = npcIdsText.split(",").map((s) => Number(s.trim())).filter((n) => Number.isInteger(n) && n > 0);
    const attributes = { name: values.name, kind: "gachapon" as const, npcIds, commonWeight: values.commonWeight, uncommonWeight: values.uncommonWeight, rareWeight: values.rareWeight };
    try {
      if (isEdit) await updatePool.mutateAsync({ id: pool!.id, attributes });
      else await createPool.mutateAsync({ attributes });
      toast.success(isEdit ? "Pool updated" : "Pool created");
      onOpenChange(false);
    } catch (e) {
      toast.error(createErrorFromUnknown(e).message);
    }
  });

  const submitIncubator = incubatorForm.handleSubmit(async (values) => {
    const attributes = { name: values.name, kind: "incubator" as const, npcIds: [values.successNpcId], commonWeight: 0, uncommonWeight: 0, rareWeight: 0 };
    try {
      if (isEdit) await updatePool.mutateAsync({ id: pool!.id, attributes });
      else await createPool.mutateAsync({ id: String(values.eggItemId), attributes });
      toast.success(isEdit ? "Pool updated" : "Pool created");
      onOpenChange(false);
    } catch (e) {
      toast.error(createErrorFromUnknown(e).message);
    }
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{isEdit ? "Edit Pool" : "New Pool"}</DialogTitle>
        </DialogHeader>

        {!isEdit && (
          <RadioGroup value={kind} onValueChange={(v) => setKind(v as RewardPoolKind)} className="flex gap-6">
            <div className="flex items-center gap-2">
              <RadioGroupItem value="gachapon" id="kind-gachapon" />
              <Label htmlFor="kind-gachapon">Gachapon</Label>
            </div>
            <div className="flex items-center gap-2">
              <RadioGroupItem value="incubator" id="kind-incubator" />
              <Label htmlFor="kind-incubator">Incubator</Label>
            </div>
          </RadioGroup>
        )}

        {kind === "gachapon" ? (
          <form onSubmit={submitGachapon} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="pf-name">Name</Label>
              <Input id="pf-name" {...gachaponForm.register("name")} />
              {gachaponForm.formState.errors.name && <p className="text-sm text-destructive">{gachaponForm.formState.errors.name.message}</p>}
            </div>
            <div className="space-y-2">
              <Label htmlFor="pf-npcs">NPC Ids (comma-separated)</Label>
              <Input id="pf-npcs" value={npcIdsText} onChange={(e) => setNpcIdsText(e.target.value)} />
            </div>
            <div className="grid grid-cols-3 gap-3">
              <div className="space-y-2">
                <Label htmlFor="pf-cw">Common Weight</Label>
                <Input id="pf-cw" type="number" {...gachaponForm.register("commonWeight", { valueAsNumber: true })} />
              </div>
              <div className="space-y-2">
                <Label htmlFor="pf-uw">Uncommon Weight</Label>
                <Input id="pf-uw" type="number" {...gachaponForm.register("uncommonWeight", { valueAsNumber: true })} />
              </div>
              <div className="space-y-2">
                <Label htmlFor="pf-rw">Rare Weight</Label>
                <Input id="pf-rw" type="number" {...gachaponForm.register("rareWeight", { valueAsNumber: true })} />
              </div>
            </div>
            {gachaponForm.formState.errors.commonWeight && (
              <p className="text-sm text-destructive">{gachaponForm.formState.errors.commonWeight.message}</p>
            )}
            <DialogFooter>
              <Button type="submit" disabled={pending}>{isEdit ? "Save" : "Create"}</Button>
            </DialogFooter>
          </form>
        ) : (
          <form onSubmit={submitIncubator} className="space-y-4">
            {!isEdit && (
              <div className="space-y-2">
                <Label htmlFor="pf-egg">Egg Item Id</Label>
                <Input id="pf-egg" type="number" {...incubatorForm.register("eggItemId", { valueAsNumber: true })} />
                {incubatorForm.formState.errors.eggItemId && <p className="text-sm text-destructive">{incubatorForm.formState.errors.eggItemId.message}</p>}
                <p className="text-xs text-muted-foreground">The egg item id becomes the pool id (e.g. 4170001).</p>
              </div>
            )}
            <div className="space-y-2">
              <Label htmlFor="pf-iname">Name</Label>
              <Input id="pf-iname" {...incubatorForm.register("name")} />
              {incubatorForm.formState.errors.name && <p className="text-sm text-destructive">{incubatorForm.formState.errors.name.message}</p>}
            </div>
            <div className="space-y-2">
              <Label htmlFor="pf-snpc">Success NPC Id</Label>
              <Input id="pf-snpc" type="number" {...incubatorForm.register("successNpcId", { valueAsNumber: true })} />
              {incubatorForm.formState.errors.successNpcId && <p className="text-sm text-destructive">{incubatorForm.formState.errors.successNpcId.message}</p>}
            </div>
            <DialogFooter>
              <Button type="submit" disabled={pending}>{isEdit ? "Save" : "Create"}</Button>
            </DialogFooter>
          </form>
        )}
      </DialogContent>
    </Dialog>
  );
}
```

(If `components/ui/radio-group.tsx` does not exist in the shadcn set, add it via the standard shadcn radio-group source — check `ls src/components/ui/` first.)

- [ ] **Step 5: Run tests to verify they pass**

Run: `npm run test -- reward-pools/`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add services/atlas-ui/src/components/features/reward-pools/
git commit -m "feat(ui): reward-pool CRUD dialogs"
```

### Task 10: RewardPoolsPage (list + tabs + global pool)

**Files:**
- Create: `pages/RewardPoolsPage.tsx`
- Create: `pages/reward-pools-columns.tsx`
- Test: `pages/__tests__/RewardPoolsPage.test.tsx`

**Interfaces:**
- Consumes: `useRewardPools`, `useGlobalRewardItems`, `useDeleteGlobalItem` (Task 6); `PoolFormDialog`, `PoolItemDialog` (Task 9); `DataTableWrapper`, `PageLoader`, `useGridRefresh`; `ItemNameCell`; shadcn `Tabs`.
- Produces: `export function RewardPoolsPage()` — routed at `/reward-pools` in Task 12. Columns module exports `poolColumns: ColumnDef<RewardPoolData>[]`.

- [ ] **Step 1: Write the failing test**

`pages/__tests__/RewardPoolsPage.test.tsx`:

```tsx
import { describe, it, expect, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { RewardPoolsPage } from "../RewardPoolsPage";

const pools = [
  { id: "henesys", type: "gachapons", attributes: { name: "Henesys", kind: "gachapon", npcIds: [9100100], commonWeight: 70, uncommonWeight: 25, rareWeight: 5 } },
  { id: "4170001", type: "gachapons", attributes: { name: "Pigmy Egg (Victoria)", kind: "incubator", npcIds: [1012004], commonWeight: 0, uncommonWeight: 0, rareWeight: 0 } },
];
vi.mock("@/services/api/reward-pools.service", () => ({
  rewardPoolsService: {
    getAllPools: vi.fn().mockResolvedValue(pools),
    getGlobalItems: vi.fn().mockResolvedValue([
      { id: "1", type: "global-gachapon-items", attributes: { itemId: 2000000, quantity: 1, tier: "common" } },
    ]),
  },
}));
vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({ activeTenant: { id: "t1", attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 } } }),
}));
vi.mock("@/components/item-name-cell", () => ({
  ItemNameCell: ({ itemId }: { itemId: string }) => <span>item-{itemId}</span>,
}));
vi.mock("@/lib/hooks/api/useItemStrings", () => ({
  // Egg-name resolution falls back to the pool's seeded name when undefined —
  // the assertions below rely on that fallback.
  useItemName: () => ({ data: undefined }),
}));

function renderPage() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <MemoryRouter>
      <QueryClientProvider client={qc}>
        <RewardPoolsPage />
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

describe("RewardPoolsPage", () => {
  it("shows both pools on the All tab with kind badges", async () => {
    renderPage();
    await waitFor(() => expect(screen.getByText("Henesys")).toBeInTheDocument());
    expect(screen.getByText("Pigmy Egg (Victoria)")).toBeInTheDocument();
    expect(screen.getAllByText(/gachapon/i).length).toBeGreaterThan(0);
    expect(screen.getAllByText(/incubator/i).length).toBeGreaterThan(0);
  });

  it("Incubators tab filters out gachapon pools", async () => {
    const user = userEvent.setup();
    renderPage();
    await waitFor(() => expect(screen.getByText("Henesys")).toBeInTheDocument());
    await user.click(screen.getByRole("tab", { name: /incubators/i }));
    expect(screen.queryByText("Henesys")).not.toBeInTheDocument();
    expect(screen.getByText("Pigmy Egg (Victoria)")).toBeInTheDocument();
  });

  it("Global Pool tab lists global items", async () => {
    const user = userEvent.setup();
    renderPage();
    await waitFor(() => expect(screen.getByText("Henesys")).toBeInTheDocument());
    await user.click(screen.getByRole("tab", { name: /global pool/i }));
    await waitFor(() => expect(screen.getByText("item-2000000")).toBeInTheDocument());
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npm run test -- RewardPoolsPage`
Expected: FAIL — page unresolved.

- [ ] **Step 3: Implement columns**

`pages/reward-pools-columns.tsx`:

```tsx
import { type ColumnDef } from "@tanstack/react-table";
import { Link } from "react-router-dom";
import { Badge } from "@/components/ui/badge";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { useTenant } from "@/context/tenant-context";
import { useItemName } from "@/lib/hooks/api/useItemStrings";
import { getAssetIconUrl } from "@/lib/utils/asset-url";
import type { RewardPoolData } from "@/types/models/reward-pool";

/**
 * Incubator pools are identified by their egg item id (= pool id): show the
 * egg's item icon + resolved item name, falling back to the seeded pool name.
 */
function PoolNameCell({ pool }: { pool: RewardPoolData }) {
  const { activeTenant } = useTenant();
  const isIncubator = pool.attributes.kind === "incubator";
  const { data: eggName } = useItemName(isIncubator ? pool.id : "");
  const iconUrl =
    isIncubator && activeTenant
      ? getAssetIconUrl(activeTenant.id, activeTenant.attributes.region, activeTenant.attributes.majorVersion, activeTenant.attributes.minorVersion, "item", parseInt(pool.id))
      : null;
  return (
    <Link to={`/reward-pools/${pool.id}`} className="hover:underline">
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>
            <span className="inline-flex items-center gap-2 font-medium">
              {iconUrl && <img src={iconUrl} alt="" width={20} height={20} loading="lazy" />}
              {isIncubator ? (eggName ?? pool.attributes.name) : pool.attributes.name}
            </span>
          </TooltipTrigger>
          <TooltipContent copyable>
            <p>{pool.id}</p>
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
    </Link>
  );
}

export const poolColumns: ColumnDef<RewardPoolData>[] = [
  {
    accessorKey: "attributes.name",
    header: "Name",
    cell: ({ row }) => <PoolNameCell pool={row.original} />,
  },
  {
    accessorKey: "attributes.kind",
    header: "Kind",
    cell: ({ row }) =>
      row.original.attributes.kind === "incubator" ? (
        <Badge className="bg-amber-500/15 text-amber-600 dark:text-amber-400 border-transparent">Incubator</Badge>
      ) : (
        <Badge variant="secondary">Gachapon</Badge>
      ),
  },
  {
    id: "details",
    header: "Details",
    cell: ({ row }) => {
      const a = row.original.attributes;
      if (a.kind === "incubator") {
        return <span className="text-muted-foreground font-mono text-sm">egg {row.original.id}</span>;
      }
      return (
        <span className="text-muted-foreground text-sm">
          C/U/R {a.commonWeight}·{a.uncommonWeight}·{a.rareWeight} — {a.npcIds.length} NPC{a.npcIds.length === 1 ? "" : "s"}
        </span>
      );
    },
  },
];
```

- [ ] **Step 4: Implement the page**

`pages/RewardPoolsPage.tsx`:

```tsx
import { useMemo, useState } from "react";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog";
import { DataTableWrapper } from "@/components/common/DataTableWrapper";
import { PageLoader } from "@/components/common/PageLoader";
import { useGridRefresh } from "@/lib/hooks/useGridRefresh";
import { useTenant } from "@/context/tenant-context";
import { useRewardPools, useGlobalRewardItems, useDeleteGlobalItem } from "@/lib/hooks/api/useRewardPools";
import { poolColumns } from "./reward-pools-columns";
import { PoolFormDialog } from "@/components/features/reward-pools/PoolFormDialog";
import { PoolItemDialog } from "@/components/features/reward-pools/PoolItemDialog";
import { ItemNameCell } from "@/components/item-name-cell";
import { toast } from "sonner";
import { createErrorFromUnknown } from "@/lib/api/errors";
import type { GlobalRewardItemData } from "@/types/models/global-reward-item";

export function RewardPoolsPage() {
  const { activeTenant } = useTenant();
  const poolsQuery = useRewardPools();
  const globalQuery = useGlobalRewardItems();
  const { isRefreshing, onRefresh } = useGridRefresh([poolsQuery]);
  const deleteGlobal = useDeleteGlobalItem();

  const [createOpen, setCreateOpen] = useState(false);
  const [globalDialog, setGlobalDialog] = useState<{ open: boolean; item?: GlobalRewardItemData }>({ open: false });
  const [globalDelete, setGlobalDelete] = useState<GlobalRewardItemData | null>(null);

  const pools = useMemo(() => poolsQuery.data ?? [], [poolsQuery.data]);
  const gachapons = useMemo(() => pools.filter((p) => p.attributes.kind === "gachapon"), [pools]);
  const incubators = useMemo(() => pools.filter((p) => p.attributes.kind === "incubator"), [pools]);
  const globalItems = globalQuery.data ?? [];
  const error = poolsQuery.error?.message ?? null;

  if (poolsQuery.isLoading) return <PageLoader />;

  const poolTable = (data: typeof pools, emptyTitle: string, emptyDescription: string) => (
    <DataTableWrapper
      columns={poolColumns}
      data={data}
      error={error}
      onRefresh={onRefresh}
      isRefreshing={isRefreshing}
      emptyState={{ title: emptyTitle, description: emptyDescription }}
    />
  );

  return (
    <div className="flex flex-col flex-1 space-y-6 p-10 pb-16">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold tracking-tight">Reward Pools</h2>
        <Button onClick={() => setCreateOpen(true)}>New Pool</Button>
      </div>

      <Tabs defaultValue="all">
        <TabsList>
          <TabsTrigger value="all">All ({pools.length})</TabsTrigger>
          <TabsTrigger value="gachapons">Gachapons ({gachapons.length})</TabsTrigger>
          <TabsTrigger value="incubators">Incubators ({incubators.length})</TabsTrigger>
          <TabsTrigger value="global">Global Pool ({globalItems.length})</TabsTrigger>
        </TabsList>

        <TabsContent value="all" className="mt-4">
          {poolTable(pools, "No reward pools found", "Seed defaults from Setup, or create a pool.")}
        </TabsContent>
        <TabsContent value="gachapons" className="mt-4">
          {poolTable(gachapons, "No gachapon pools", "Seed defaults from Setup, or create one.")}
        </TabsContent>
        <TabsContent value="incubators" className="mt-4">
          {poolTable(incubators, "No incubator pools", "Seed defaults from Setup, or create one.")}
        </TabsContent>

        <TabsContent value="global" className="mt-4 space-y-4">
          <p className="text-sm text-muted-foreground">
            Global items merge into every gachapon machine's roll for their tier. They never apply to incubator pools.
          </p>
          <div className="flex justify-end">
            <Button onClick={() => setGlobalDialog({ open: true })}>Add Item</Button>
          </div>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Item</TableHead>
                <TableHead>Quantity</TableHead>
                <TableHead>Tier</TableHead>
                <TableHead className="w-24" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {globalItems.map((gi) => (
                <TableRow key={gi.id}>
                  <TableCell><ItemNameCell itemId={String(gi.attributes.itemId)} tenant={activeTenant} /></TableCell>
                  <TableCell>{gi.attributes.quantity}</TableCell>
                  <TableCell><Badge variant="outline">{gi.attributes.tier}</Badge></TableCell>
                  <TableCell className="space-x-2 text-right">
                    <Button variant="ghost" size="sm" onClick={() => setGlobalDialog({ open: true, item: gi })}>Edit</Button>
                    <Button variant="ghost" size="sm" onClick={() => setGlobalDelete(gi)}>Delete</Button>
                  </TableCell>
                </TableRow>
              ))}
              {globalItems.length === 0 && (
                <TableRow><TableCell colSpan={4} className="text-muted-foreground">No global items.</TableCell></TableRow>
              )}
            </TableBody>
          </Table>
        </TabsContent>
      </Tabs>

      <PoolFormDialog open={createOpen} onOpenChange={setCreateOpen} mode="create" />
      <PoolItemDialog
        open={globalDialog.open}
        onOpenChange={(open) => setGlobalDialog((s) => ({ ...s, open }))}
        kind="global"
        item={globalDialog.item}
      />
      <AlertDialog open={!!globalDelete} onOpenChange={(open) => !open && setGlobalDelete(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete global item?</AlertDialogTitle>
            <AlertDialogDescription>
              Item {globalDelete?.attributes.itemId} will stop appearing in every gachapon's {globalDelete?.attributes.tier} rolls.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={async () => {
                try {
                  await deleteGlobal.mutateAsync({ itemRecordId: globalDelete!.id });
                  toast.success("Global item deleted");
                } catch (e) {
                  toast.error(createErrorFromUnknown(e).message);
                } finally {
                  setGlobalDelete(null);
                }
              }}
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `npm run test -- RewardPoolsPage`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add services/atlas-ui/src/pages/RewardPoolsPage.tsx services/atlas-ui/src/pages/reward-pools-columns.tsx services/atlas-ui/src/pages/__tests__/RewardPoolsPage.test.tsx
git commit -m "feat(ui): Reward Pools list page with kind tabs + global pool management"
```

### Task 11: RewardPoolDetailPage (kind-adaptive detail + item CRUD)

**Files:**
- Create: `pages/RewardPoolDetailPage.tsx`
- Create: `components/features/reward-pools/PoolItemsTable.tsx`
- Test: `pages/__tests__/RewardPoolDetailPage.test.tsx`

**Interfaces:**
- Consumes: `useRewardPool`, `useRewardPoolItems`, `useGlobalRewardItems`, `useDeletePoolItem`, `useDeleteRewardPool` (Task 6); `incubatorChances`, `gachaponChances`, `tierHasMixedWeights` (Task 7); `PoolFormDialog`, `PoolItemDialog` (Task 9); `ItemNameCell`, `useItemName(itemId: string)`, `useNPC(tenant, npcId)`, `getAssetIconUrl`.
- Produces: `export function RewardPoolDetailPage()` — routed at `/reward-pools/:id` in Task 12. `PoolItemsTable` props:

```tsx
export function PoolItemsTable(props: {
  kind: "gachapon" | "incubator";
  poolId: string;
  tierWeights: { common: number; uncommon: number; rare: number };  // ignored for incubator
  items: RewardPoolItemData[];
  globalItems: GlobalRewardItemData[];  // pass [] for incubator
  onEdit: (item: RewardPoolItemData) => void;
  onDelete: (item: RewardPoolItemData) => void;
}): JSX.Element;
```

- [ ] **Step 1: Write the failing test**

`pages/__tests__/RewardPoolDetailPage.test.tsx`:

```tsx
import { describe, it, expect, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { RewardPoolDetailPage } from "../RewardPoolDetailPage";

const henesys = { id: "henesys", type: "gachapons", attributes: { name: "Henesys", kind: "gachapon", npcIds: [9100100], commonWeight: 70, uncommonWeight: 25, rareWeight: 5 } };
const egg = { id: "4170001", type: "gachapons", attributes: { name: "Pigmy Egg (Victoria)", kind: "incubator", npcIds: [1012004], commonWeight: 0, uncommonWeight: 0, rareWeight: 0 } };

const mocks = vi.hoisted(() => ({
  getPoolById: vi.fn(),
  getItems: vi.fn(),
  getGlobalItems: vi.fn().mockResolvedValue([]),
}));
vi.mock("@/services/api/reward-pools.service", () => ({ rewardPoolsService: mocks }));
vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({ activeTenant: { id: "t1", attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 } } }),
}));
vi.mock("@/components/item-name-cell", () => ({
  ItemNameCell: ({ itemId }: { itemId: string }) => <span>item-{itemId}</span>,
}));
vi.mock("@/lib/hooks/api/useItemStrings", () => ({
  useItemName: () => ({ data: "Pigmy Egg" }),
}));
vi.mock("@/lib/hooks/api/useNpcs", () => ({
  useNPC: () => ({ data: { attributes: { name: "Pigmy & Etran" } } }),
}));

function renderAt(id: string) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <MemoryRouter initialEntries={[`/reward-pools/${id}`]}>
      <QueryClientProvider client={qc}>
        <Routes>
          <Route path="/reward-pools/:id" element={<RewardPoolDetailPage />} />
        </Routes>
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

describe("RewardPoolDetailPage", () => {
  it("gachapon: shows tier weights card and tier-grouped pool with global rows badged", async () => {
    mocks.getPoolById.mockResolvedValue(henesys);
    mocks.getItems.mockResolvedValue([
      { id: "1", type: "gachapon-items", attributes: { gachaponId: "henesys", itemId: 2000000, quantity: 1, tier: "common", weight: 0 } },
    ]);
    mocks.getGlobalItems.mockResolvedValue([
      { id: "9", type: "global-gachapon-items", attributes: { itemId: 2000001, quantity: 1, tier: "common" } },
    ]);
    renderAt("henesys");
    await waitFor(() => expect(screen.getByText("Henesys")).toBeInTheDocument());
    expect(screen.getByText(/tier weights/i)).toBeInTheDocument();
    expect(screen.getByText("item-2000001")).toBeInTheDocument();
    expect(screen.getByText(/global/i)).toBeInTheDocument();
    // two common rows, uniform within a 70% tier → 35.00% each
    expect(screen.getAllByText("35.00%").length).toBe(2);
  });

  it("incubator: shows egg card, weight column, weight-based chance; no tier weights card", async () => {
    mocks.getPoolById.mockResolvedValue(egg);
    mocks.getItems.mockResolvedValue([
      { id: "1", type: "gachapon-items", attributes: { gachaponId: "4170001", itemId: 2000000, quantity: 1, tier: "", weight: 75 } },
      { id: "2", type: "gachapon-items", attributes: { gachaponId: "4170001", itemId: 1302000, quantity: 1, tier: "", weight: 25 } },
    ]);
    mocks.getGlobalItems.mockResolvedValue([]);
    renderAt("4170001");
    await waitFor(() => expect(screen.getByText("Pigmy Egg (Victoria)")).toBeInTheDocument());
    expect(screen.queryByText(/tier weights/i)).not.toBeInTheDocument();
    expect(screen.getByText(/success npc/i)).toBeInTheDocument();
    expect(screen.getByText("75.00%")).toBeInTheDocument();
    expect(screen.getByText("25.00%")).toBeInTheDocument();
  });

  it("warns when a gachapon tier mixes weighted and zero-weight rows", async () => {
    mocks.getPoolById.mockResolvedValue(henesys);
    mocks.getItems.mockResolvedValue([
      { id: "1", type: "gachapon-items", attributes: { gachaponId: "henesys", itemId: 2000000, quantity: 1, tier: "rare", weight: 10 } },
      { id: "2", type: "gachapon-items", attributes: { gachaponId: "henesys", itemId: 2000001, quantity: 1, tier: "rare", weight: 0 } },
    ]);
    mocks.getGlobalItems.mockResolvedValue([]);
    renderAt("henesys");
    await waitFor(() => expect(screen.getByText("Henesys")).toBeInTheDocument());
    expect(screen.getByText(/exclude/i)).toBeInTheDocument();
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npm run test -- RewardPoolDetailPage`
Expected: FAIL — page unresolved.

- [ ] **Step 3: Implement `PoolItemsTable`**

`components/features/reward-pools/PoolItemsTable.tsx`:

```tsx
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { ItemNameCell } from "@/components/item-name-cell";
import { useTenant } from "@/context/tenant-context";
import { gachaponChances, incubatorChances, tierHasMixedWeights } from "@/lib/utils/reward-pool-chance";
import type { RewardPoolItemData } from "@/types/models/reward-pool-item";
import type { GlobalRewardItemData } from "@/types/models/global-reward-item";

const TIERS = ["common", "uncommon", "rare"] as const;

function pct(v: number): string {
  return `${(v * 100).toFixed(2)}%`;
}

interface PoolItemsTableProps {
  kind: "gachapon" | "incubator";
  poolId: string;
  tierWeights: { common: number; uncommon: number; rare: number };
  items: RewardPoolItemData[];
  globalItems: GlobalRewardItemData[];
  onEdit: (item: RewardPoolItemData) => void;
  onDelete: (item: RewardPoolItemData) => void;
}

export function PoolItemsTable({ kind, tierWeights, items, globalItems, onEdit, onDelete }: PoolItemsTableProps) {
  const { activeTenant } = useTenant();

  if (kind === "incubator") {
    const chances = incubatorChances(items.map((i) => ({ id: i.id, weight: i.attributes.weight })));
    return (
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Item</TableHead>
            <TableHead>Quantity</TableHead>
            <TableHead>Weight</TableHead>
            <TableHead>Chance</TableHead>
            <TableHead className="w-24" />
          </TableRow>
        </TableHeader>
        <TableBody>
          {items.map((it) => (
            <TableRow key={it.id}>
              <TableCell><ItemNameCell itemId={String(it.attributes.itemId)} tenant={activeTenant} /></TableCell>
              <TableCell>{it.attributes.quantity}</TableCell>
              <TableCell>{it.attributes.weight}</TableCell>
              <TableCell>{pct(chances.get(it.id) ?? 0)}</TableCell>
              <TableCell className="space-x-2 text-right">
                <Button variant="ghost" size="sm" onClick={() => onEdit(it)}>Edit</Button>
                <Button variant="ghost" size="sm" onClick={() => onDelete(it)}>Delete</Button>
              </TableCell>
            </TableRow>
          ))}
          {items.length === 0 && (
            <TableRow><TableCell colSpan={5} className="text-muted-foreground">No items in this pool.</TableCell></TableRow>
          )}
        </TableBody>
      </Table>
    );
  }

  // Gachapon: merged rows (machine + global) grouped by tier, chances via the
  // exact selectTier×selectItem model. Global rows are read-only here.
  const machineRows = items.map((it) => ({
    key: `m-${it.id}`,
    tier: it.attributes.tier as (typeof TIERS)[number],
    weight: it.attributes.weight,
    itemId: it.attributes.itemId,
    quantity: it.attributes.quantity,
    source: "machine" as const,
    item: it,
  }));
  const globalRows = globalItems.map((gi) => ({
    key: `g-${gi.id}`,
    tier: gi.attributes.tier as (typeof TIERS)[number],
    weight: 0, // global items always roll with weight 0 (reward/processor.go getMergedPool)
    itemId: gi.attributes.itemId,
    quantity: gi.attributes.quantity,
    source: "global" as const,
    item: undefined,
  }));
  const rows = [...machineRows, ...globalRows];
  const chances = gachaponChances(tierWeights, rows.map(({ key, tier, weight }) => ({ key, tier, weight })));

  return (
    <div className="space-y-6">
      {TIERS.map((tier) => {
        const tierRows = rows.filter((r) => r.tier === tier);
        if (tierRows.length === 0) return null;
        const mixed = tierHasMixedWeights(rows, tier);
        return (
          <div key={tier} className="space-y-2">
            <div className="flex items-center gap-2">
              <Badge variant={tier === "rare" ? "destructive" : tier === "uncommon" ? "secondary" : "outline"}>{tier}</Badge>
              <span className="text-sm text-muted-foreground">{tierRows.length} item{tierRows.length === 1 ? "" : "s"}</span>
            </div>
            {mixed && (
              <Alert variant="destructive">
                <AlertDescription>
                  This tier mixes weighted and unweighted items — the weighted roll excludes every zero-weight item
                  (including all global items) from this tier.
                </AlertDescription>
              </Alert>
            )}
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Item</TableHead>
                  <TableHead>Quantity</TableHead>
                  <TableHead>Weight</TableHead>
                  <TableHead>Chance</TableHead>
                  <TableHead className="w-24" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {tierRows.map((r) => {
                  const c = chances.get(r.key);
                  return (
                    <TableRow key={r.key} className={c?.excluded ? "opacity-60" : undefined}>
                      <TableCell className="space-x-2">
                        <ItemNameCell itemId={String(r.itemId)} tenant={activeTenant} />
                        {r.source === "global" && <Badge variant="outline">Global</Badge>}
                      </TableCell>
                      <TableCell>{r.quantity}</TableCell>
                      <TableCell>{r.weight > 0 ? r.weight : "—"}</TableCell>
                      <TableCell>
                        <TooltipProvider>
                          <Tooltip>
                            <TooltipTrigger asChild><span>{pct(c?.chance ?? 0)}</span></TooltipTrigger>
                            <TooltipContent>
                              <p>tier chance × within-tier share (mirrors the server roll)</p>
                            </TooltipContent>
                          </Tooltip>
                        </TooltipProvider>
                      </TableCell>
                      <TableCell className="space-x-2 text-right">
                        {r.source === "machine" ? (
                          <>
                            <Button variant="ghost" size="sm" onClick={() => onEdit(r.item!)}>Edit</Button>
                            <Button variant="ghost" size="sm" onClick={() => onDelete(r.item!)}>Delete</Button>
                          </>
                        ) : (
                          <span className="text-xs text-muted-foreground">Global Pool tab</span>
                        )}
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          </div>
        );
      })}
      {rows.length === 0 && <p className="text-sm text-muted-foreground">No items in this pool.</p>}
    </div>
  );
}
```

(If `components/ui/alert.tsx` lacks `AlertDescription`, use its actual export shape — it exists, see `ls src/components/ui/alert.tsx`.)

- [ ] **Step 4: Implement the page**

`pages/RewardPoolDetailPage.tsx`:

```tsx
import { useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog";
import { PageLoader } from "@/components/common/PageLoader";
import { ErrorDisplay } from "@/components/common/ErrorDisplay";
import { useTenant } from "@/context/tenant-context";
import { useRewardPool, useRewardPoolItems, useGlobalRewardItems, useDeletePoolItem, useDeleteRewardPool } from "@/lib/hooks/api/useRewardPools";
import { useItemName } from "@/lib/hooks/api/useItemStrings";
import { useNPC } from "@/lib/hooks/api/useNpcs";
import { getAssetIconUrl } from "@/lib/utils/asset-url";
import { createErrorFromUnknown } from "@/lib/api/errors";
import { PoolFormDialog } from "@/components/features/reward-pools/PoolFormDialog";
import { PoolItemDialog } from "@/components/features/reward-pools/PoolItemDialog";
import { PoolItemsTable } from "@/components/features/reward-pools/PoolItemsTable";
import type { RewardPoolItemData } from "@/types/models/reward-pool-item";

function NpcChip({ npcId }: { npcId: number }) {
  const { activeTenant } = useTenant();
  const { data: npc } = useNPC(activeTenant!, npcId);
  return (
    <Link to={`/npcs/${npcId}`} className="hover:underline">
      <Badge variant="secondary">{npc?.attributes?.name ?? npcId}</Badge>
    </Link>
  );
}

export function RewardPoolDetailPage() {
  const params = useParams();
  const id = params.id as string;
  const navigate = useNavigate();
  const { activeTenant } = useTenant();

  const { data: pool, isLoading, error, refetch } = useRewardPool(id);
  const itemsQuery = useRewardPoolItems(id);
  const isIncubator = pool?.attributes.kind === "incubator";
  const globalQuery = useGlobalRewardItems();
  const deleteItem = useDeletePoolItem();
  const deletePool = useDeleteRewardPool();

  const { data: eggName } = useItemName(isIncubator ? id : "");

  const [editPoolOpen, setEditPoolOpen] = useState(false);
  const [itemDialog, setItemDialog] = useState<{ open: boolean; item?: RewardPoolItemData }>({ open: false });
  const [itemDelete, setItemDelete] = useState<RewardPoolItemData | null>(null);
  const [poolDeleteOpen, setPoolDeleteOpen] = useState(false);

  if (isLoading) return <PageLoader />;
  if (error || !pool) {
    return (
      <div className="p-10">
        <ErrorDisplay error={error ?? "Reward pool not found"} retry={() => refetch()} />
      </div>
    );
  }

  const attrs = pool.attributes;
  const items = itemsQuery.data ?? [];
  const globalItems = isIncubator ? [] : (globalQuery.data ?? []);
  const totalWeight = items.reduce((s, i) => s + i.attributes.weight, 0);
  const tierTotal = attrs.commonWeight + attrs.uncommonWeight + attrs.rareWeight;
  const eggIconUrl =
    isIncubator && activeTenant
      ? getAssetIconUrl(activeTenant.id, activeTenant.attributes.region, activeTenant.attributes.majorVersion, activeTenant.attributes.minorVersion, "item", parseInt(id))
      : null;

  return (
    <div className="flex flex-col flex-1 space-y-6 p-10 pb-16 overflow-y-auto">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          {eggIconUrl && <img src={eggIconUrl} alt="" width={32} height={32} />}
          <h2 className="text-2xl font-bold tracking-tight">{attrs.name}</h2>
          {isIncubator ? (
            <Badge className="bg-amber-500/15 text-amber-600 dark:text-amber-400 border-transparent">Incubator</Badge>
          ) : (
            <Badge variant="secondary">Gachapon</Badge>
          )}
          <span className="text-muted-foreground font-mono">#{pool.id}</span>
        </div>
        <Button variant="outline" onClick={() => setEditPoolOpen(true)}>Edit Pool</Button>
      </div>

      {isIncubator ? (
        <Card>
          <CardHeader><CardTitle className="text-sm font-medium">Egg</CardTitle></CardHeader>
          <CardContent className="space-y-2 text-sm">
            <div className="flex justify-between">
              <span className="text-muted-foreground">Egg item</span>
              <Link to={`/items/${pool.id}`} className="hover:underline">{eggName ?? pool.id}</Link>
            </div>
            <div className="flex justify-between items-center">
              <span className="text-muted-foreground">Success NPC</span>
              {attrs.npcIds.length > 0 ? <NpcChip npcId={attrs.npcIds[0]} /> : <span className="text-muted-foreground">none</span>}
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">Total weight</span>
              <span>{totalWeight}</span>
            </div>
          </CardContent>
        </Card>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <Card>
            <CardHeader><CardTitle className="text-sm font-medium">Tier Weights</CardTitle></CardHeader>
            <CardContent className="space-y-2 text-sm">
              {([["Common", attrs.commonWeight], ["Uncommon", attrs.uncommonWeight], ["Rare", attrs.rareWeight]] as const).map(([label, w]) => (
                <div key={label} className="flex justify-between">
                  <span className="text-muted-foreground">{label}</span>
                  <span>
                    {w}
                    <span className="text-muted-foreground ml-2">({tierTotal > 0 ? ((w / tierTotal) * 100).toFixed(1) : "0.0"}%)</span>
                  </span>
                </div>
              ))}
            </CardContent>
          </Card>
          <Card>
            <CardHeader><CardTitle className="text-sm font-medium">NPCs</CardTitle></CardHeader>
            <CardContent className="text-sm">
              {attrs.npcIds.length > 0 ? (
                <div className="flex flex-wrap gap-2">
                  {attrs.npcIds.map((npcId) => <NpcChip key={npcId} npcId={npcId} />)}
                </div>
              ) : (
                <span className="text-muted-foreground">No NPCs assigned</span>
              )}
            </CardContent>
          </Card>
        </div>
      )}

      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle className="text-sm font-medium">Pool Items ({items.length})</CardTitle>
          <Button size="sm" onClick={() => setItemDialog({ open: true })}>Add Item</Button>
        </CardHeader>
        <CardContent>
          {itemsQuery.isLoading ? (
            <p className="text-sm text-muted-foreground">Loading pool items...</p>
          ) : (
            <PoolItemsTable
              kind={isIncubator ? "incubator" : "gachapon"}
              poolId={id}
              tierWeights={{ common: attrs.commonWeight, uncommon: attrs.uncommonWeight, rare: attrs.rareWeight }}
              items={items}
              globalItems={globalItems}
              onEdit={(item) => setItemDialog({ open: true, item })}
              onDelete={setItemDelete}
            />
          )}
        </CardContent>
      </Card>

      <Card className="border-destructive/40">
        <CardHeader><CardTitle className="text-sm font-medium">Danger Zone</CardTitle></CardHeader>
        <CardContent>
          <Button variant="destructive" onClick={() => setPoolDeleteOpen(true)}>Delete Pool</Button>
        </CardContent>
      </Card>

      <PoolFormDialog open={editPoolOpen} onOpenChange={setEditPoolOpen} mode="edit" pool={pool} />
      <PoolItemDialog
        open={itemDialog.open}
        onOpenChange={(open) => setItemDialog((s) => ({ ...s, open }))}
        kind={isIncubator ? "incubator" : "gachapon"}
        poolId={id}
        item={itemDialog.item}
      />

      <AlertDialog open={!!itemDelete} onOpenChange={(open) => !open && setItemDelete(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete item?</AlertDialogTitle>
            <AlertDialogDescription>Item {itemDelete?.attributes.itemId} will be removed from this pool.</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={async () => {
                try {
                  await deleteItem.mutateAsync({ poolId: id, itemRecordId: itemDelete!.id });
                  toast.success("Item deleted");
                } catch (e) {
                  toast.error(createErrorFromUnknown(e).message);
                } finally {
                  setItemDelete(null);
                }
              }}
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog open={poolDeleteOpen} onOpenChange={setPoolDeleteOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete this pool?</AlertDialogTitle>
            <AlertDialogDescription>
              "{attrs.name}" and its reward assignments will be removed. This cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={async () => {
                try {
                  await deletePool.mutateAsync({ id });
                  toast.success("Pool deleted");
                  navigate("/reward-pools");
                } catch (e) {
                  toast.error(createErrorFromUnknown(e).message);
                }
              }}
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `npm run test -- RewardPoolDetailPage`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add services/atlas-ui/src/pages/RewardPoolDetailPage.tsx services/atlas-ui/src/components/features/reward-pools/PoolItemsTable.tsx services/atlas-ui/src/pages/__tests__/RewardPoolDetailPage.test.tsx
git commit -m "feat(ui): kind-adaptive Reward Pool detail page with item CRUD"
```

---

# Phase 4 — Wiring swap and cleanup

### Task 12: Routes, nav, breadcrumbs, Setup label; delete the old Gachapons modules

**Files:**
- Modify: `App.tsx` (lazy imports at :23-24, routes at :86-87)
- Modify: `components/app-sidebar.tsx:61`
- Modify: `lib/breadcrumbs/routes.ts:193-198` + the `GACHAPONS` constant at :488
- Modify: `pages/SetupPage.tsx:186` (label only)
- Delete: `pages/GachaponsPage.tsx`, `pages/GachaponDetailPage.tsx`, `pages/gachapons-columns.tsx`, `pages/__tests__/GachaponsPage.test.tsx`, `services/api/gachapons.service.ts`, `lib/hooks/api/useGachapons.ts`, `types/models/gachapon.ts`, `types/models/gachapon-reward.ts`
- Modify: `types/models/index.ts`, `lib/hooks/api/index.ts`, `services/api/index.ts` (drop the deleted modules' exports)
- Test: `pages/__tests__/reward-pools-redirect.test.tsx` (create)

**Interfaces:**
- Consumes: `RewardPoolsPage` (Task 10), `RewardPoolDetailPage` (Task 11); `Navigate` from `react-router-dom` (first use in App.tsx — add to the existing import).
- Produces: `/reward-pools`, `/reward-pools/:id` routes; permanent redirects from `/gachapons`(`/:id`).

- [ ] **Step 1: Write the failing redirect test**

`pages/__tests__/reward-pools-redirect.test.tsx` — exercises the same route elements App.tsx will use, without mounting the whole shell:

```tsx
import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Navigate, Route, Routes, useParams } from "react-router-dom";

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({ activeTenant: null }),
}));

function GachaponDetailRedirect() {
  const { id } = useParams();
  return <Navigate to={`/reward-pools/${id}`} replace />;
}

function renderAt(path: string) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route path="/reward-pools" element={<div>reward pools list</div>} />
        <Route path="/reward-pools/:id" element={<div>reward pool detail</div>} />
        <Route path="/gachapons" element={<Navigate to="/reward-pools" replace />} />
        <Route path="/gachapons/:id" element={<GachaponDetailRedirect />} />
      </Routes>
    </MemoryRouter>,
  );
}

describe("gachapons → reward-pools redirects", () => {
  it("redirects the list route", () => {
    renderAt("/gachapons");
    expect(screen.getByText("reward pools list")).toBeInTheDocument();
  });
  it("redirects a deep link preserving the id", () => {
    renderAt("/gachapons/4170001");
    expect(screen.getByText("reward pool detail")).toBeInTheDocument();
  });
});
```

- [ ] **Step 2: Run test — it passes standalone; now do the swap**

Run: `npm run test -- reward-pools-redirect` → PASS (it pins the redirect shape App.tsx must use).

`App.tsx` — replace lines 23-24:

```tsx
const RewardPoolsPage = lazy(() => import("@/pages/RewardPoolsPage").then(m => ({ default: m.RewardPoolsPage })));
const RewardPoolDetailPage = lazy(() => import("@/pages/RewardPoolDetailPage").then(m => ({ default: m.RewardPoolDetailPage })));
```

replace lines 86-87 (inside the AppShell route group) and add `Navigate` + `useParams` to the react-router-dom import + a small redirect component near the top of the file:

```tsx
                    <Route path="/reward-pools" element={<RewardPoolsPage />} />
                    <Route path="/reward-pools/:id" element={<RewardPoolDetailPage />} />
                    <Route path="/gachapons" element={<Navigate to="/reward-pools" replace />} />
                    <Route path="/gachapons/:id" element={<GachaponRedirect />} />
```

```tsx
function GachaponRedirect() {
  const { id } = useParams();
  return <Navigate to={`/reward-pools/${id}`} replace />;
}
```

`app-sidebar.tsx:61`:

```tsx
            { title: "Reward Pools", url: "/reward-pools" },
```

`lib/breadcrumbs/routes.ts` — replace the Gachapon block:

```ts
  // Reward pool routes
  {
    pattern: '/reward-pools',
    label: 'Reward Pools',
    parent: '/',
  },
  {
    pattern: '/reward-pools/[id]',
    label: 'Pool',
    parent: '/reward-pools',
  },
```

and the constant at :488: `GACHAPONS: '/gachapons',` → `REWARD_POOLS: '/reward-pools',` (update its references — grep `ROUTES.GACHAPONS` / `GACHAPONS`).

`pages/SetupPage.tsx:186`: `label: "Gachapons",` → `label: "Reward Pools",` (the seed group/hooks are untouched — the backend group name is still `gachapons`).

Delete the old modules and drop their exports from `types/models/index.ts`, `lib/hooks/api/index.ts`, `services/api/index.ts`:

```bash
git rm services/atlas-ui/src/pages/GachaponsPage.tsx services/atlas-ui/src/pages/GachaponDetailPage.tsx services/atlas-ui/src/pages/gachapons-columns.tsx services/atlas-ui/src/pages/__tests__/GachaponsPage.test.tsx services/atlas-ui/src/services/api/gachapons.service.ts services/atlas-ui/src/lib/hooks/api/useGachapons.ts services/atlas-ui/src/types/models/gachapon.ts services/atlas-ui/src/types/models/gachapon-reward.ts
```

Then `grep -rn "gachapons.service\|useGachapons\|GachaponData\|GachaponRewardData\|GachaponsPage\|GachaponDetailPage" services/atlas-ui/src/` — every remaining hit must be fixed (expected hits: only the index files being edited in this step; `SetupPage` imports `useSeedGachapons`/`useGachaponsSeedStatus` from `useSeed.ts`, which are seed hooks, not the deleted module — leave them).

- [ ] **Step 3: Full frontend gate**

```bash
cd services/atlas-ui
source ~/.nvm/nvm.sh && nvm use 22
npm run build && npm run test && npm run lint
```
Expected: build + tests clean; `lint` introduces no NEW errors over the pre-existing baseline (compare against `git stash`-free main if unsure — the lint baseline is pre-broken, gate on no-new-errors only).

- [ ] **Step 4: Repo guards (frontend swap touched no Go, but run before PR update anyway)**

```bash
cd "$(git rev-parse --show-toplevel)"
tools/redis-key-guard.sh && tools/goroutine-guard.sh
```
Expected: clean.

- [ ] **Step 5: Commit**

```bash
git add -A services/atlas-ui/src
git commit -m "feat(ui): swap Gachapons surface for Reward Pools; redirect old routes"
```

---

# Final verification (before updating PR #909)

- [ ] Backend: `cd services/atlas-reward-pools/atlas.com/reward-pools && go build ./... && go vet ./... && go test -race ./...` — clean.
- [ ] `docker buildx bake atlas-reward-pools` from the worktree root — clean.
- [ ] Frontend: `npm run build && npm run test` under nvm 22 — clean; no new lint errors.
- [ ] `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh` — clean. (`service-registration-guard.sh` only if deploy/config files changed — this plan touches none.)
- [ ] Manual smoke via the dev proxy (`npm run dev` against a live ingress): `/gachapons` redirects; incubator pool `4170001` shows its weighted items (the pre-fix behavior showed an empty pool); item add/edit/delete round-trips.
- [ ] Code review (CLAUDE.md): dispatch `backend-guidelines-reviewer` (Go changed) + `frontend-guidelines-reviewer` (TS changed) + `plan-adherence-reviewer` via `superpowers:requesting-code-review` — findings to `docs/tasks/task-128-item-tag-seal-incubator/audit-reward-pools-ui-*.md`. Pin reviewer subagents to Sonnet (project model policy).

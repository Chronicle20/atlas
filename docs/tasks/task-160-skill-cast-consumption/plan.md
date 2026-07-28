# Skill Cast Consumption Fidelity (itemConNo) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

> **Scope reconciliation (2026-07-27):** This plan was descoped after `task-158`
> (PR #1003, "Shadow Stars") landed on main and independently implemented the
> Shadow Stars half of the original task-160 PRD — the cast-time `bulletConsume`
> cost, the SHADOW_CLAW star-id rewrite, and the claw attack-path skip
> (original FR-2, FR-3, and Tasks 1/5/6/7). Those are DONE on main; see
> `common.go` `resolveShadowStarsCast`, `shadow_stars.go`,
> `character_attack_projectile.go` `projectileConsumptionSkipped`, and
> `SkillUsageInfo.SpiritJavelinItemId()`. The only genuinely-remaining task-160
> work is **FR-1: generic `itemConNo` quantity plumbing** — a skill declaring
> `itemConNo > 1` (e.g. Echo of Hero consuming 2× Magic Rock on a later-version
> tenant) still consumes exactly one item, because `RequestItemConsume`
> hardcodes `1`. This plan covers FR-1 only.

**Goal:** Skill casts in atlas-channel consume the WZ-declared item quantity (`itemConNo`) drawn from the lowest-index slot that alone holds ≥ that amount, instead of a hardcoded `1`.

**Architecture:** All behavior changes live in atlas-channel: a slot-selection helper in `compartment`, a `quantity` parameter on the `consumable.Processor.RequestItemConsume` interface method (and its `ProcessorImpl`), and the `itemCon` block of `UseSkill` in `skill/handler`. The existing Kafka `REQUEST_ITEM_CONSUME` contract already carries `Quantity int16` and atlas-consumables already honors it — no schema, no atlas-data, and no atlas-consumables changes. FR-1 is entirely WZ-data-driven (`ItemConsumeAmount()` resolves from per-tenant atlas-data), so **no version branches and no legacy-version handling are required** — `itemConNo` is a server-side skill-effect attribute, not a wire field.

**Tech Stack:** Go (workspace `go.work` monorepo), logrus, segmentio/kafka-go message envelopes, project Builder-pattern test setup, package-level var-func test seams.

## Global Constraints

- No Kafka schema changes: `RequestItemConsumeBody{Source, ItemId, Quantity int16}` is used as-is; `RequestItemConsumeCommandProvider` already takes `quantity int16` (`consumable/producer.go:16`).
- No atlas-data changes; no atlas-consumables behavior changes.
- The consume amount is WZ-data-driven (`effect.Model.ItemConsumeAmount()`, backed by atlas-data's `itemConNo` read at `atlas-data/skill/reader.go:218`), never keyed to a skill id or tenant version.
- Single-slot draw only: choose the lowest-index slot that alone holds ≥ the required amount. If no single slot qualifies, warn + skip + let the cast proceed (unchanged defense-in-depth stance — an `itemCon` shortfall does not block the cast).
- An absent/zero `itemConNo` (reader default `0`, and some later dumps carry a string `"0"`) means **one** item, never zero — floor `< 1` to `1`, defensively at BOTH the processor layer and the cast-path layer.
- Builder-pattern test setup; no `*_testhelpers.go` files with test-only constructors (CLAUDE.md).
- Test seams are package-level `var` funcs restored via `t.Cleanup` (established `common.go` convention: `loadCasterFunc`, `loadCasterInventoryFunc`, `rectQueryFunc`, `propRollFunc`).
- No literal home/absolute paths in committed files.
- Optional atlas-consumables pinning test is NOT built: that service's quantity pass-through is a one-line data flow and has no Kafka/compartment seam infra; the wire value is pinned by the atlas-channel producer test in Task 2. Recorded here so plan-adherence review doesn't flag it as silently skipped.
- Commit after every task. All commands below run from the worktree root unless a `cd` is shown.

## Module map (for `cd` targets)

| Module | Path |
|---|---|
| atlas-channel | `services/atlas-channel/atlas.com/channel` |

Only atlas-channel is touched. (The original plan's `libs/atlas-constants` move and `libs/atlas-packet` getter were part of the Shadow Stars scope now shipped by task-158.)

---

### Task 1: `compartment.FindFirstByItemIdWithQuantity`

Slot-selection helper for FR-1: lowest-index slot holding ≥ N of the item. The existing `FindFirstByItemId` (`compartment/model.go:58`) relies on the backing slice's incidental order and ignores quantity; this helper sorts by slot ascending first (convention set by `resolvePlan`, `character_attack_projectile.go`) and requires the single slot to hold enough.

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/compartment/model.go` (add method after `FindFirstByItemId`, add `"sort"` import)
- Test: `services/atlas-channel/atlas.com/channel/compartment/model_test.go` (append — file already exists, package `compartment`)

**Interfaces:**
- Consumes: `asset.Model` (`TemplateId() uint32`, `Slot() int16`, `Quantity() uint32`), `asset.NewModelBuilder(id uint32, compartmentId uuid.UUID, templateId uint32)` builder, `compartment.NewBuilder(id, characterId, invType, capacity)`.
- Produces: `func (m Model) FindFirstByItemIdWithQuantity(templateId uint32, quantity int16) (*asset.Model, bool)` — Task 3 calls it from `skill/handler` via `c.Inventory().CompartmentByType(invType)`.

- [ ] **Step 1: Write the failing tests**

Append to `services/atlas-channel/atlas.com/channel/compartment/model_test.go` (package `compartment`; reuse the file's existing import block — `asset`, `testing`, `uuid`, `inventory`, `item` — no new imports needed):

```go
func qtyAsset(t *testing.T, slot int16, templateId uint32, qty uint32) asset.Model {
	t.Helper()
	a, err := asset.NewModelBuilder(uint32(slot), uuid.New(), templateId).SetSlot(slot).SetQuantity(qty).Build()
	if err != nil {
		t.Fatalf("asset build: %v", err)
	}
	return a
}

func qtyCompartment(t *testing.T, assets ...asset.Model) Model {
	t.Helper()
	b := NewBuilder(uuid.New(), 1, inventory.TypeValueUse, 96)
	for _, a := range assets {
		b.AddAsset(a)
	}
	m, err := b.Build()
	if err != nil {
		t.Fatalf("compartment build: %v", err)
	}
	return m
}

func TestFindFirstByItemIdWithQuantity_LowestSlotWinsUnsortedInput(t *testing.T) {
	// Assets deliberately out of slot order; both qualify — slot 2 must win.
	m := qtyCompartment(t, qtyAsset(t, 5, 4006000, 10), qtyAsset(t, 2, 4006000, 3))
	a, found := m.FindFirstByItemIdWithQuantity(4006000, 2)
	if !found || a.Slot() != 2 {
		t.Fatalf("got (slot=%v, found=%v), want (slot=2, found=true)", a, found)
	}
}

func TestFindFirstByItemIdWithQuantity_SkipsShortSlots(t *testing.T) {
	// Slot 1 is short (1 < 2); slot 3 qualifies.
	m := qtyCompartment(t, qtyAsset(t, 1, 4006000, 1), qtyAsset(t, 3, 4006000, 2))
	a, found := m.FindFirstByItemIdWithQuantity(4006000, 2)
	if !found || a.Slot() != 3 {
		t.Fatalf("got (slot=%v, found=%v), want (slot=3, found=true)", a, found)
	}
}

func TestFindFirstByItemIdWithQuantity_ExactBoundary(t *testing.T) {
	m := qtyCompartment(t, qtyAsset(t, 1, 2070000, 200))
	a, found := m.FindFirstByItemIdWithQuantity(2070000, 200)
	if !found || a.Slot() != 1 {
		t.Fatalf("got (slot=%v, found=%v), want (slot=1, found=true)", a, found)
	}
}

func TestFindFirstByItemIdWithQuantity_NoSlotQualifies(t *testing.T) {
	// Aggregate 300 across two slots, but no single slot holds 200.
	m := qtyCompartment(t, qtyAsset(t, 1, 2070000, 150), qtyAsset(t, 2, 2070000, 150))
	if _, found := m.FindFirstByItemIdWithQuantity(2070000, 200); found {
		t.Fatal("expected not found: no single slot holds 200")
	}
}

func TestFindFirstByItemIdWithQuantity_ItemAbsent(t *testing.T) {
	m := qtyCompartment(t, qtyAsset(t, 1, 4006000, 10))
	if _, found := m.FindFirstByItemIdWithQuantity(4006001, 1); found {
		t.Fatal("expected not found: template id absent")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./compartment/ -run TestFindFirstByItemIdWithQuantity -v`
Expected: FAIL to compile with `m.FindFirstByItemIdWithQuantity undefined`

- [ ] **Step 3: Write the implementation**

In `services/atlas-channel/atlas.com/channel/compartment/model.go`, add `"sort"` to the imports (stdlib block) and add after `FindFirstByItemId` (line 58):

```go
// FindFirstByItemIdWithQuantity returns the matching asset in the
// lowest-index slot whose quantity is at least `quantity`. Candidates are
// sorted by slot ascending before scanning, so the result is deterministic
// regardless of the backing slice's order (unlike FindFirstByItemId). A slot
// holding less than `quantity` is skipped — single-slot draw only.
func (m Model) FindFirstByItemIdWithQuantity(templateId uint32, quantity int16) (*asset.Model, bool) {
	matching := make([]asset.Model, 0, len(m.Assets()))
	for _, a := range m.Assets() {
		if a.TemplateId() == templateId {
			matching = append(matching, a)
		}
	}
	sort.Slice(matching, func(i, j int) bool { return matching[i].Slot() < matching[j].Slot() })
	for _, a := range matching {
		if int64(a.Quantity()) >= int64(quantity) {
			a := a
			return &a, true
		}
	}
	return nil, false
}
```

(`asset.Quantity()` is `uint32`, the parameter is `int16` — compare via `int64` to avoid conversion truncation.)

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go test -race ./compartment/ -v`
Expected: PASS (new tests plus existing compartment tests)

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/compartment/model.go services/atlas-channel/atlas.com/channel/compartment/model_test.go
git commit -m "feat(channel): compartment lowest-slot-with-quantity lookup (task-160 FR-1)"
```

---

### Task 2: `RequestItemConsume` gains a `quantity` parameter

Signature change on the `consumable.Processor` interface AND `ProcessorImpl` (a signature change, not a sibling method, so the compiler enforces the call-site sweep). The hardcoded `1` at `consumable/processor.go:43` disappears; all existing call sites pass a literal `1` (behavior unchanged — `common.go`'s consume becomes the real amount in Task 3).

> **Changed since the original plan:** `consumable.Processor` is now an
> interface with a `ProcessorImpl` (task-116 Gen3 processor convergence), so
> the signature change touches BOTH `processor.go:17` (interface) and
> `processor.go:41` (impl). Main also added an **eighth** call site,
> `shopscanner/processor.go:79` (the legacy owl-shop-scanner feature), which the
> original 7-site list did not have.

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/consumable/processor.go` (interface method line 17; impl line 41)
- Modify: `services/atlas-channel/atlas.com/channel/skill/handler/common.go:110` (literal `1` for now; Task 3 makes it the effect amount)
- Modify: `services/atlas-channel/atlas.com/channel/shopscanner/processor.go:79`
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/pet_food.go:23`
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/pet_item_use.go:23`
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:66`
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_item_use.go:23,32,50`
- Test: `services/atlas-channel/atlas.com/channel/consumable/producer_test.go` (create)

**Interfaces:**
- Consumes: existing `RequestItemConsumeCommandProvider(f, characterId, source, itemId, quantity int16)` (already takes quantity).
- Produces: `RequestItemConsume(f field.Model, characterId character.Id, itemId item.Id, source slot.Position, quantity int16, updateTime uint32) error` on the interface and impl — quantity inserted BEFORE updateTime. Task 3 passes the real amount through it.

- [ ] **Step 1: Write the producer test (regression pin)**

Create `services/atlas-channel/atlas.com/channel/consumable/producer_test.go` (pins that the emitted command carries the given quantity — the itemCon path now sends values > 1):

```go
package consumable

import (
	"encoding/json"
	"testing"

	consumablemsg "atlas-channel/kafka/message/consumable"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// TestRequestItemConsumeCommandProvider_CarriesQuantity pins that the
// REQUEST_ITEM_CONSUME command body carries the caller's quantity on the
// wire — the skill-cast itemCon path now sends values > 1 (FR-1).
func TestRequestItemConsumeCommandProvider_CarriesQuantity(t *testing.T) {
	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()
	cid := character.Id(42)

	provider := RequestItemConsumeCommandProvider(f, cid, slot.Position(7), item.Id(4006000), int16(2))
	msgs, err := provider()
	if err != nil {
		t.Fatalf("provider error: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}

	var cmd consumablemsg.Command[consumablemsg.RequestItemConsumeBody]
	if err := json.Unmarshal(msgs[0].Value, &cmd); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if cmd.Body.Source != slot.Position(7) {
		t.Errorf("source: got %d, want 7", cmd.Body.Source)
	}
	if cmd.Body.ItemId != item.Id(4006000) {
		t.Errorf("itemId: got %d, want 4006000", cmd.Body.ItemId)
	}
	if cmd.Body.Quantity != 2 {
		t.Errorf("quantity: got %d, want 2", cmd.Body.Quantity)
	}
}
```

Note: match the actual `consumablemsg` envelope/field names (`Command[...]`, `RequestItemConsumeBody`, `Type`/`CharacterId` if present). Verify against `kafka/message/consumable/kafka.go` before asserting extra fields; keep the assertions to fields the type actually exposes.

- [ ] **Step 2: Run the test — it should PASS already**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./consumable/ -run TestRequestItemConsumeCommandProvider -v`
Expected: PASS — the provider already takes quantity; this pins the wire contract before the processor change. (Regression pin, not red-green.)

- [ ] **Step 3: Change the interface and impl signature**

In `services/atlas-channel/atlas.com/channel/consumable/processor.go`:

1. Interface (line 17-ish), change the method to:

```go
	RequestItemConsume(f field.Model, characterId character.Id, itemId item.Id, source slot.Position, quantity int16, updateTime uint32) error
```

2. Impl (line 41), replace with:

```go
func (p *ProcessorImpl) RequestItemConsume(f field.Model, characterId character.Id, itemId item.Id, source slot.Position, quantity int16, updateTime uint32) error {
	// Defense in depth for an absent/"0"-string itemConNo (FR-1): an absent
	// amount means one item, never zero.
	if quantity < 1 {
		quantity = 1
	}
	p.l.Debugf("Character [%d] using item [%d] from slot [%d]. quantity [%d], updateTime [%d]", characterId, itemId, source, quantity, updateTime)
	return producer.ProviderImpl(p.l)(p.ctx)(consumable2.EnvCommandTopic)(RequestItemConsumeCommandProvider(f, characterId, source, itemId, quantity))
}
```

- [ ] **Step 4: Sweep the call sites (compiler-enforced)**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./...`
Expected: FAIL listing every call site — exactly these eight (seven files):

| File | Line(s) | Change |
|---|---|---|
| `skill/handler/common.go` | 110 | insert `1` before the trailing `0`: `..., slot.Position(a.Slot()), 1, 0)` (Task 3 replaces the `1` with the effect amount) |
| `shopscanner/processor.go` | 79 | `..., source, 1, updateTime)` |
| `socket/handler/pet_food.go` | 23 | `..., slot.Position(p.Source()), 1, p.UpdateTime())` |
| `socket/handler/pet_item_use.go` | 23 | `..., slot.Position(p.Source()), 1, p.UpdateTime())` |
| `socket/handler/character_cash_item_use.go` | 66 | `..., source, 1, updateTime)` |
| `socket/handler/character_item_use.go` | 23, 32, 50 | `..., slot.Position(p.Source()), 1, p.UpdateTime())` |

Insert the literal `1` as the new `quantity` argument before the final `updateTime` argument. Do not touch any other argument. (The line numbers are pre-sweep; if the compiler reports a different line after another edit, trust the compiler's list — the set of files/sites is authoritative.)

- [ ] **Step 5: Verify the module compiles and tests pass**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test -race ./consumable/... ./socket/handler/... ./skill/handler/... ./shopscanner/...`
Expected: build clean, all tests PASS

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/consumable/processor.go services/atlas-channel/atlas.com/channel/consumable/producer_test.go services/atlas-channel/atlas.com/channel/skill/handler/common.go services/atlas-channel/atlas.com/channel/shopscanner/processor.go services/atlas-channel/atlas.com/channel/socket/handler/pet_food.go services/atlas-channel/atlas.com/channel/socket/handler/pet_item_use.go services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go services/atlas-channel/atlas.com/channel/socket/handler/character_item_use.go
git commit -m "feat(channel): RequestItemConsume carries an explicit quantity (task-160 FR-1)"
```

---

### Task 3: `UseSkill` itemCon path — real amount + lowest-qualifying-slot + seams

Wire `ItemConsumeAmount()` through the cast path (FR-1) and introduce two test seams so the block is exercisable offline: `loadCasterWithInventoryFunc` (the block currently loads inline via `cp.GetById(cp.InventoryDecorator)`, untestable) and `requestItemConsumeFunc`. Follows the same seam idiom task-158 used for `loadCasterInventoryFunc`.

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/skill/handler/common.go` (two seams near `loadCasterFunc`; rewrite the itemCon block at lines 105-118)
- Test: `services/atlas-channel/atlas.com/channel/skill/handler/common_consume_test.go` (create)

**Interfaces:**
- Consumes: Task 1's `FindFirstByItemIdWithQuantity`; Task 2's `RequestItemConsume(..., quantity int16, updateTime uint32)`; existing `character.Processor.InventoryDecorator`, `character.Model.Inventory().CompartmentByType(invType)`, `inventoryconst.TypeFromItemId`, `effect.Model.ItemConsumeAmount()`.
- Produces:
  - `var loadCasterWithInventoryFunc func(cp character.Processor, characterId uint32) (character.Model, error)`
  - `var requestItemConsumeFunc func(p consumable.Processor, f field.Model, characterId charcon.Id, itemId itemconst.Id, source slot.Position, quantity int16, updateTime uint32) error`

- [ ] **Step 1: Write the failing tests**

Create `services/atlas-channel/atlas.com/channel/skill/handler/common_consume_test.go`:

```go
package handler

import (
	"context"
	"errors"
	"testing"

	"atlas-channel/asset"
	"atlas-channel/character"
	"atlas-channel/compartment"
	"atlas-channel/consumable"
	"atlas-channel/data/skill/effect"
	"atlas-channel/inventory"

	charcon "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	inventoryconst "github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	itemconst "github.com/Chronicle20/atlas/libs/atlas-constants/item"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// testConsumeSkillId is an arbitrary id NOT equal to NightLordShadowStarsId and
// with no mount/registry/dispatcher classification, so UseSkill exercises only
// the itemCon path (the Shadow Stars pre-flight and the mob/mount/dispatcher
// paths are all inert for it).
const testConsumeSkillId = uint32(99999999)

// magicRockItemId is Magic Rock (a real itemCon for several casts, e.g. Echo of
// Hero). classification 400 -> ETC compartment via inventoryconst.TypeFromItemId.
const magicRockItemId = uint32(4006000)

type consumeRecorder struct {
	called   bool
	itemId   itemconst.Id
	source   slot.Position
	quantity int16
}

// overrideConsumeSeams replaces the two itemCon seams with recorders and
// restores them via t.Cleanup. loadCasterWithInventoryFunc returns the given
// caster (or loadErr).
func overrideConsumeSeams(t *testing.T, c character.Model, loadErr error) *consumeRecorder {
	t.Helper()
	rec := &consumeRecorder{}

	prevLoad := loadCasterWithInventoryFunc
	loadCasterWithInventoryFunc = func(_ character.Processor, _ uint32) (character.Model, error) {
		return c, loadErr
	}
	prevConsume := requestItemConsumeFunc
	requestItemConsumeFunc = func(_ consumable.Processor, _ field.Model, _ charcon.Id, itemId itemconst.Id, source slot.Position, quantity int16, _ uint32) error {
		rec.called = true
		rec.itemId = itemId
		rec.source = source
		rec.quantity = quantity
		return nil
	}
	t.Cleanup(func() {
		loadCasterWithInventoryFunc = prevLoad
		requestItemConsumeFunc = prevConsume
	})
	return rec
}

func testField() field.Model {
	return field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()
}

func consumeAsset(t *testing.T, slotIdx int16, templateId uint32, qty uint32) asset.Model {
	t.Helper()
	a, err := asset.NewModelBuilder(uint32(slotIdx), uuid.New(), templateId).SetSlot(slotIdx).SetQuantity(qty).Build()
	if err != nil {
		t.Fatalf("asset build: %v", err)
	}
	return a
}

// casterWithCompartment builds a character whose compartment for the given
// inventory type holds the provided assets.
func casterWithCompartment(t *testing.T, it inventoryconst.Type, assets ...asset.Model) character.Model {
	t.Helper()
	cb := compartment.NewBuilder(uuid.New(), 1, it, 96)
	for _, a := range assets {
		cb.AddAsset(a)
	}
	cm, err := cb.Build()
	if err != nil {
		t.Fatalf("compartment build: %v", err)
	}
	inv := inventory.NewBuilder(1).SetCompartment(cm).MustBuild()
	return character.NewModelBuilder().SetId(1).SetInventory(inv).MustBuild()
}

func consumeEffect(t *testing.T, itemConsume uint32, amount uint32) effect.Model {
	t.Helper()
	e, err := effect.Extract(effect.RestModel{
		ItemConsume:       itemConsume,
		ItemConsumeNumber: amount,
	})
	if err != nil {
		t.Fatalf("effect extract: %v", err)
	}
	return e
}

func useSkillInfo() packetmodel.SkillUsageInfo {
	return packetmodel.NewSkillUsageInfoBuilder().SetSkillId(testConsumeSkillId).SetSkillLevel(1).Build()
}

func runUseSkill(t *testing.T, e effect.Model) {
	t.Helper()
	if err := UseSkill(logrus.New())(context.Background())(nil, testField(), 1, useSkillInfo(), e); err != nil {
		t.Fatalf("UseSkill: %v", err)
	}
}

func TestUseSkill_ItemConsumeAmountPlumbed(t *testing.T) {
	it, _ := inventoryconst.TypeFromItemId(itemconst.Id(magicRockItemId))
	// Slot 1 is short (1 < 2); slot 2 qualifies -> quantity 2 from slot 2.
	c := casterWithCompartment(t, it, consumeAsset(t, 1, magicRockItemId, 1), consumeAsset(t, 2, magicRockItemId, 3))
	rec := overrideConsumeSeams(t, c, nil)

	runUseSkill(t, consumeEffect(t, magicRockItemId, 2))

	if !rec.called {
		t.Fatal("expected a consume request")
	}
	if rec.quantity != 2 {
		t.Errorf("quantity: got %d, want 2", rec.quantity)
	}
	if rec.source != slot.Position(2) {
		t.Errorf("source slot: got %d, want 2", rec.source)
	}
	if rec.itemId != itemconst.Id(magicRockItemId) {
		t.Errorf("itemId: got %d, want %d", rec.itemId, magicRockItemId)
	}
}

func TestUseSkill_ItemConsumeAmountZeroFloorsToOne(t *testing.T) {
	it, _ := inventoryconst.TypeFromItemId(itemconst.Id(magicRockItemId))
	c := casterWithCompartment(t, it, consumeAsset(t, 1, magicRockItemId, 5))
	rec := overrideConsumeSeams(t, c, nil)

	runUseSkill(t, consumeEffect(t, magicRockItemId, 0))

	if !rec.called || rec.quantity != 1 {
		t.Fatalf("got (called=%v, quantity=%d), want (true, 1)", rec.called, rec.quantity)
	}
}

func TestUseSkill_ItemConShortfallSkipsButCastProceeds(t *testing.T) {
	it, _ := inventoryconst.TypeFromItemId(itemconst.Id(magicRockItemId))
	// Two slots of 1 each; no single slot holds 2 -> skip, warn, no error (cast proceeds).
	c := casterWithCompartment(t, it, consumeAsset(t, 1, magicRockItemId, 1), consumeAsset(t, 2, magicRockItemId, 1))
	rec := overrideConsumeSeams(t, c, nil)

	runUseSkill(t, consumeEffect(t, magicRockItemId, 2)) // no t.Fatalf => UseSkill returned nil

	if rec.called {
		t.Error("expected NO consume request on shortfall")
	}
}

func TestUseSkill_ItemConCasterLoadFailureCastProceeds(t *testing.T) {
	rec := overrideConsumeSeams(t, character.Model{}, errors.New("boom"))

	runUseSkill(t, consumeEffect(t, magicRockItemId, 2)) // no t.Fatalf => UseSkill returned nil

	if rec.called {
		t.Error("expected NO consume request on load failure")
	}
}
```

Note for the implementer:
- `inventory.NewBuilder(1).SetCompartment(...)` keys by the compartment's own type (`inventory/builder.go` — `b.compartments[m.Type()] = m`), so it stores under whichever type `TypeFromItemId(4006000)` resolves to; the test builds the compartment with that same resolved type, so lookup and storage agree.
- Verify the exact builder/API names against the current tree before running: `effect.RestModel` field name for `itemConNo` (the reader's setter is `SetItemConsumeNumber`, so the RestModel field is likely `ItemConsumeNumber`), `inventory.NewBuilder(...).SetCompartment/MustBuild`, `character.NewModelBuilder().SetId/SetInventory/MustBuild`, and `compartment.NewBuilder(...).Build()` return shape. Adjust the helpers if any differ — the assertions are the contract, the builders are mechanics.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./skill/handler/ -run TestUseSkill_ItemCon -v`
Expected: FAIL to compile with `undefined: loadCasterWithInventoryFunc` (and `requestItemConsumeFunc`)

- [ ] **Step 3: Implement the seams and rewrite the itemCon block**

In `services/atlas-channel/atlas.com/channel/skill/handler/common.go`:

1. After the existing `loadCasterFunc` var (around line 32), add the two seams:

```go
// loadCasterWithInventoryFunc is the inventory-decorated caster-load seam the
// generic itemCon consume path uses (it needs an arbitrary compartment, not
// just the USE compartment loadCasterInventoryFunc returns). Tests replace it.
var loadCasterWithInventoryFunc = func(cp character.Processor, characterId uint32) (character.Model, error) {
	return cp.GetById(cp.InventoryDecorator)(characterId)
}

// requestItemConsumeFunc is the consumable-request seam tests can replace.
// Production delegates to consumable.Processor.RequestItemConsume.
var requestItemConsumeFunc = func(p consumable.Processor, f field.Model, characterId charcon.Id, itemId itemconst.Id, source slot.Position, quantity int16, updateTime uint32) error {
	return p.RequestItemConsume(f, characterId, itemId, source, quantity, updateTime)
}
```

2. Replace the itemCon block (currently lines 105-118) with:

```go
			if itemId := e.ItemConsume(); itemId > 0 {
				cp := character.NewProcessor(l, ctx)
				if c, cErr := loadCasterWithInventoryFunc(cp, characterId); cErr == nil {
					if invType, typeOk := inventoryconst.TypeFromItemId(itemconst.Id(itemId)); typeOk {
						amount := int16(e.ItemConsumeAmount())
						if amount < 1 {
							// Absent itemConNo (reader default 0) means one item (FR-1).
							amount = 1
						}
						if a, found := c.Inventory().CompartmentByType(invType).FindFirstByItemIdWithQuantity(itemId, amount); found {
							_ = requestItemConsumeFunc(consumable.NewProcessor(l, ctx), f, charcon.Id(characterId), itemconst.Id(itemId), slot.Position(a.Slot()), amount, 0)
						} else {
							l.Warnf("Character [%d] cast skill [%d] requiring [%d]x item [%d] but no single slot holds enough; cast permitted (defense-in-depth gate only).", characterId, info.SkillId(), amount, itemId)
						}
					}
				} else {
					l.WithError(cErr).Warnf("Character [%d] cast skill [%d] requiring item [%d] but failed to load inventory; cast permitted.", characterId, info.SkillId(), itemId)
				}
			}
```

(This replaces the Task-2 interim `..., slot.Position(a.Slot()), 1, 0)` line with the real amount and the quantity-aware slot lookup.)

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go test -race ./skill/handler/... -v`
Expected: PASS — the four new tests plus every pre-existing `skill/handler` test (Shadow Stars, mount, apply-to-mobs, registry, recipients).

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/skill/handler/common.go services/atlas-channel/atlas.com/channel/skill/handler/common_consume_test.go
git commit -m "feat(channel): UseSkill consumes the WZ itemConNo amount from a qualifying slot (task-160 FR-1)"
```

---

### Task 4: Full verification sweep

Per CLAUDE.md Build & Verification. Only atlas-channel has code changes; no `go.mod`/Dockerfile/`go.work` change, so atlas-channel is the single bake target. Note the guard set grew since the original plan (lint, goroutine-guard).

**Files:** none (verification only; fix-and-recommit anything that fails).

- [ ] **Step 1: Test, vet, build atlas-channel**

```bash
cd services/atlas-channel/atlas.com/channel && go test -race ./... && go vet ./... && go build ./...
```

Expected: all clean. `-race` matters — the seam vars are package-level and tests mutate them; a race flag means a missing `t.Cleanup` restore or an errant `t.Parallel()` on a seam-mutating test (do NOT add `t.Parallel()` to those).

- [ ] **Step 2: Bake the service image**

From the worktree root:

```bash
docker buildx bake atlas-channel
```

Expected: builds clean. (No new lib was added, but the bake is mandatory per CLAUDE.md.)

- [ ] **Step 3: Repo-root guards**

From the worktree root:

```bash
tools/redis-key-guard.sh
tools/goroutine-guard.sh
tools/lint.sh --check
```

Expected: all clean. If `tools/lint.sh --check` reports formatting diffs on the touched files, run `tools/lint.sh` (no flags) to fix in place, then re-commit. (`service-registration-guard.sh` and `template-opcode-order-guard.sh` are N/A — no services.json/k8s/docker-bake/go.work or template changes.)

- [ ] **Step 4: Acceptance-criteria walk**

Confirm each FR-1 criterion maps to a passing test or a compile-time fact:

| Criterion | Evidence |
|---|---|
| quantity N emitted for `ItemConsumeAmount() == N` (N>1) | `TestUseSkill_ItemConsumeAmountPlumbed`, `TestRequestItemConsumeCommandProvider_CarriesQuantity` |
| `N == 0` emits quantity 1 (floor) | `TestUseSkill_ItemConsumeAmountZeroFloorsToOne` + processor `< 1` guard |
| lowest qualifying slot chosen; shortfall warns + proceeds | `TestFindFirstByItemIdWithQuantity_*`, `TestUseSkill_ItemConShortfallSkipsButCastProceeds` |
| pre-existing call sites still quantity 1 | Task 2 compile sweep + existing item-use/pet/cash/shopscanner tests passing |
| module + bake + guard checks | Steps 1-3 above |

- [ ] **Step 5: Commit any verification fixes**

```bash
git add -A && git commit -m "fix(channel): verification sweep fixes (task-160)"
```

If nothing changed, no commit — the branch is ready for the code-review phase (`superpowers:requesting-code-review` before any PR, per CLAUDE.md).

---

## Superseded scope (implemented by task-158, PR #1003 — do NOT re-implement)

Recorded so plan-adherence review does not flag these original-PRD items as missing:

| Original task-160 item | Where it lives on main |
|---|---|
| FR-2 cast-time `bulletConsume` (Shadow Stars) + SHADOW_CLAW star-id rewrite | `skill/handler/common.go` `resolveShadowStarsCast`/`emitStarConsume`, `skill/handler/shadow_stars.go` |
| FR-3 claw attack-path SHADOW_CLAW skip | `socket/handler/character_attack_projectile.go` `projectileConsumptionSkipped` |
| Task 5 `SkillUsageInfo.SpiritJavelinItemId()` getter | `libs/atlas-packet/model/skill_usage_info.go:62` |
| Task 1 weapon→projectile mapping in atlas-constants | not needed — task-158 solved FR-2 without extracting `requiredClassification`; it remains local in `character_attack_projectile.go` |

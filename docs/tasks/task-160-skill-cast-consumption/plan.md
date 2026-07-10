# Skill Cast Consumption Fidelity Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Skill casts in atlas-channel consume the WZ-declared item quantity (`itemConNo`), Shadow Stars pays its 200-star `bulletConsume` cost at cast time with the SHADOW_CLAW buff amount encoding the consumed star, and claw attacks under SHADOW_CLAW consume zero projectiles.

**Architecture:** All behavior changes live in atlas-channel's `skill/handler` (cast path), `consumable` (quantity plumbing), `compartment` (slot-selection helper), and `socket/handler` (attack-path skip), plus one getter in `libs/atlas-packet` and one moved pure function in `libs/atlas-constants/item`. The existing Kafka `REQUEST_ITEM_CONSUME` contract already carries `Quantity int16` and atlas-consumables already honors it — no schema or downstream changes.

**Tech Stack:** Go (workspace `go.work` monorepo), logrus, segmentio/kafka-go message envelopes, project Builder-pattern test setup, package-level var-func test seams.

## Global Constraints

- No Kafka schema changes: `RequestItemConsumeBody{Source, ItemId, Quantity int16}` is used as-is (design §4.1).
- No atlas-data changes; no atlas-consumables behavior changes (design §1, §3).
- Consume triggers are WZ-data-driven (`ItemConsumeAmount()`, `BulletConsume()`), never keyed to a skill id or tenant version (PRD NFR §8).
- `shadowClawStarEncodingBase = 2069999` is a named constant with IDA citations (v83 0x949C4C, v87 0x9C4A50, v95 0x907461, jms185 0xA0A2F4), NOT tenant config (design §2.1, §5.2 option C rejected).
- Single-slot draws only for both `itemCon` and `bulletConsume` (design §5.3 documents the deliberate divergence from the client's aggregate gate).
- `itemCon` shortfall: warn + skip + cast proceeds (unchanged stance). `bulletConsume` shortfall: warn + reject cast with zero side effects (FR-2.3).
- Builder-pattern test setup; no `*_testhelpers.go` files with test-only constructors (CLAUDE.md).
- Test seams are package-level `var` funcs restored via `t.Cleanup` (established `common.go` convention: `loadCasterFunc`, `propRollFunc`).
- No literal home/absolute paths in committed files.
- Optional atlas-consumables pinning test (PRD §7 "Optionally") is NOT built: that service's quantity pass-through is a one-line data flow (`processor.go` reserve with received quantity), its `processor_test.go` has no Kafka/compartment seam infrastructure, and the wire value is pinned by the atlas-channel producer test in Task 3. Recorded here so plan-adherence review doesn't flag it as silently skipped.
- Commit after every task. All commands below run from the worktree root unless a `cd` is shown.

## Module map (for `cd` targets)

| Module | Path |
|---|---|
| atlas-channel | `services/atlas-channel/atlas.com/channel` |
| atlas-constants | `libs/atlas-constants` |
| atlas-packet | `libs/atlas-packet` |

---

### Task 1: Move weapon→projectile mapping to `libs/atlas-constants/item`

The mapping currently lives as `requiredClassification` in `socket/handler` (`character_attack_projectile.go:211`). `skill/handler` needs it too but cannot import `socket/handler` (import cycle: `socket/handler` → `skill/handler`). It is a pure WeaponType→Classification fact — exactly what atlas-constants is for (DOM-21, design §5.4).

**Files:**
- Modify: `libs/atlas-constants/item/constants.go` (add function after `GetWeaponType`)
- Test: `libs/atlas-constants/item/constants_test.go` (add test)
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_projectile.go:90,209-222` (delete `requiredClassification`, call the lib)
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_projectile_test.go:33-56` (delete `TestRequiredClassification` — moved to the lib)

**Interfaces:**
- Consumes: existing `item.WeaponType`, `item.Classification` constants.
- Produces: `func ProjectileClassificationForWeapon(w WeaponType) (Classification, bool)` in package `github.com/Chronicle20/atlas/libs/atlas-constants/item`. Task 6 calls it from `skill/handler`; the projectile handler calls it from `socket/handler`.

- [ ] **Step 1: Write the failing test**

Append to `libs/atlas-constants/item/constants_test.go`:

```go
func TestProjectileClassificationForWeapon(t *testing.T) {
	cases := []struct {
		name   string
		w      WeaponType
		wantC  Classification
		wantOk bool
	}{
		{"bow", WeaponTypeBow, ClassificationConsumableArrow, true},
		{"crossbow", WeaponTypeCrossbow, ClassificationConsumableArrow, true},
		{"claw", WeaponTypeClaw, ClassificationConsumableThrowingStar, true},
		{"gun", WeaponTypeGun, ClassificationBullet, true},
		{"sword non-ranged", WeaponTypeOneHandedSword, 0, false},
		{"none", WeaponTypeNone, 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, ok := ProjectileClassificationForWeapon(tc.w)
			if ok != tc.wantOk || c != tc.wantC {
				t.Fatalf("got (%d, %v), want (%d, %v)", c, ok, tc.wantC, tc.wantOk)
			}
		})
	}
}
```

Note: this file is in package `item` (internal test package) — match the existing package declaration at the top of `constants_test.go`; if it is `package item_test`, prefix the identifiers with `item.` instead.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-constants && go test ./item/ -run TestProjectileClassificationForWeapon -v`
Expected: FAIL to compile with `undefined: ProjectileClassificationForWeapon`

- [ ] **Step 3: Write the implementation**

In `libs/atlas-constants/item/constants.go`, immediately after `GetWeaponType`:

```go
// ProjectileClassificationForWeapon returns the projectile classification
// consumed by the given ranged weapon type. The second return is false for
// non-ranged weapons.
func ProjectileClassificationForWeapon(w WeaponType) (Classification, bool) {
	switch w {
	case WeaponTypeBow, WeaponTypeCrossbow:
		return ClassificationConsumableArrow, true
	case WeaponTypeClaw:
		return ClassificationConsumableThrowingStar, true
	case WeaponTypeGun:
		return ClassificationBullet, true
	default:
		return 0, false
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd libs/atlas-constants && go test ./item/ -run TestProjectileClassificationForWeapon -v`
Expected: PASS

- [ ] **Step 5: Refactor the projectile handler to use it**

In `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_projectile.go`:

1. Delete the whole `requiredClassification` function (lines 209–222, including its doc comment).
2. Change line 90 from:

```go
	classification, rangedWeapon := requiredClassification(weaponType)
```

to:

```go
	classification, rangedWeapon := item.ProjectileClassificationForWeapon(weaponType)
```

(The `item` import — `github.com/Chronicle20/atlas/libs/atlas-constants/item` — is already present.)

3. In `character_attack_projectile_test.go`, delete `TestRequiredClassification` (lines 33–56). Its coverage moved to the lib in Step 1.

- [ ] **Step 6: Run both modules' tests**

Run: `cd services/atlas-channel/atlas.com/channel && go test -race ./socket/handler/... && cd ../../../../libs/atlas-constants && go test -race ./item/`
Expected: PASS (all)

- [ ] **Step 7: Commit**

```bash
git add libs/atlas-constants/item/constants.go libs/atlas-constants/item/constants_test.go services/atlas-channel/atlas.com/channel/socket/handler/character_attack_projectile.go services/atlas-channel/atlas.com/channel/socket/handler/character_attack_projectile_test.go
git commit -m "refactor(constants): move weapon->projectile classification to atlas-constants (task-160)"
```

---

### Task 2: `compartment.FindFirstByItemIdWithQuantity`

Slot-selection helper for FR-1.3: lowest-index slot holding ≥ N of the item. The existing `FindFirstByItemId` relies on incidental asset order; this helper sorts by slot ascending first (convention set by `resolvePlan`, `character_attack_projectile.go`).

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/compartment/model.go` (add method after `FindFirstByItemId`, add `sort` import)
- Test: `services/atlas-channel/atlas.com/channel/compartment/model_test.go` (create)

**Interfaces:**
- Consumes: `asset.Model` (`TemplateId() uint32`, `Slot() int16`, `Quantity() uint32`), `asset.NewModelBuilder(id uint32, compartmentId uuid.UUID, templateId uint32)` builder.
- Produces: `func (m Model) FindFirstByItemIdWithQuantity(templateId uint32, quantity int16) (*asset.Model, bool)` — Task 4 calls it from `skill/handler` via `c.Inventory().CompartmentByType(invType)`.

- [ ] **Step 1: Write the failing tests**

Create `services/atlas-channel/atlas.com/channel/compartment/model_test.go`:

```go
package compartment

import (
	"testing"

	"atlas-channel/asset"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/google/uuid"
)

func qtyAsset(slot int16, templateId uint32, qty uint32) asset.Model {
	return asset.NewModelBuilder(1, uuid.New(), templateId).
		SetSlot(slot).
		SetQuantity(qty).
		MustBuild()
}

func compartmentWith(assets ...asset.Model) Model {
	b := NewBuilder(uuid.New(), 1, inventory.TypeValueUse, 96)
	for _, a := range assets {
		b.AddAsset(a)
	}
	return b.MustBuild()
}

func TestFindFirstByItemIdWithQuantity_LowestSlotWinsUnsortedInput(t *testing.T) {
	// Assets deliberately out of slot order; both qualify — slot 2 must win.
	m := compartmentWith(
		qtyAsset(5, 4006000, 10),
		qtyAsset(2, 4006000, 3),
	)
	a, found := m.FindFirstByItemIdWithQuantity(4006000, 2)
	if !found || a.Slot() != 2 {
		t.Fatalf("got (slot=%v, found=%v), want (slot=2, found=true)", a, found)
	}
}

func TestFindFirstByItemIdWithQuantity_SkipsShortSlots(t *testing.T) {
	// Slot 1 is short (1 < 2); slot 3 qualifies.
	m := compartmentWith(
		qtyAsset(1, 4006000, 1),
		qtyAsset(3, 4006000, 2),
	)
	a, found := m.FindFirstByItemIdWithQuantity(4006000, 2)
	if !found || a.Slot() != 3 {
		t.Fatalf("got (slot=%v, found=%v), want (slot=3, found=true)", a, found)
	}
}

func TestFindFirstByItemIdWithQuantity_ExactBoundary(t *testing.T) {
	m := compartmentWith(qtyAsset(1, 2070000, 200))
	a, found := m.FindFirstByItemIdWithQuantity(2070000, 200)
	if !found || a.Slot() != 1 {
		t.Fatalf("got (slot=%v, found=%v), want (slot=1, found=true)", a, found)
	}
}

func TestFindFirstByItemIdWithQuantity_NoSlotQualifies(t *testing.T) {
	m := compartmentWith(
		qtyAsset(1, 2070000, 150),
		qtyAsset(2, 2070000, 150),
	)
	if _, found := m.FindFirstByItemIdWithQuantity(2070000, 200); found {
		t.Fatal("expected not found: no single slot holds 200")
	}
}

func TestFindFirstByItemIdWithQuantity_ItemAbsent(t *testing.T) {
	m := compartmentWith(qtyAsset(1, 4006000, 10))
	if _, found := m.FindFirstByItemIdWithQuantity(4006001, 1); found {
		t.Fatal("expected not found: template id absent")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./compartment/ -run TestFindFirstByItemIdWithQuantity -v`
Expected: FAIL to compile with `m.FindFirstByItemIdWithQuantity undefined`

- [ ] **Step 3: Write the implementation**

In `services/atlas-channel/atlas.com/channel/compartment/model.go`, add `"sort"` to the imports and add after `FindFirstByItemId`:

```go
// FindFirstByItemIdWithQuantity returns the matching asset in the
// lowest-index slot whose quantity is at least `quantity`. Candidates are
// sorted by slot ascending before scanning, so the result is deterministic
// regardless of the backing slice's order (unlike FindFirstByItemId).
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
git commit -m "feat(channel): compartment lowest-slot-with-quantity lookup (task-160)"
```

---

### Task 3: `RequestItemConsume` gains a `quantity` parameter

Signature change (design §5.1 option A — chosen over a sibling method so the compiler enforces the call-site sweep, PRD acceptance criterion 3). The hardcoded `1` at `consumable/processor.go:30` disappears; all seven existing call sites pass a literal `1` (behavior unchanged — `common.go`'s literal `1` becomes the real amount in Task 4).

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/consumable/processor.go:28-31`
- Modify: `services/atlas-channel/atlas.com/channel/skill/handler/common.go:84`
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/pet_food.go:22`
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/pet_item_use.go:22`
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:51`
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_item_use.go:22,31,49`
- Test: `services/atlas-channel/atlas.com/channel/consumable/producer_test.go` (create)

**Interfaces:**
- Consumes: existing `RequestItemConsumeCommandProvider(f field.Model, characterId character.Id, source slot.Position, itemId item.Id, quantity int16)` (already takes quantity).
- Produces: `func (p *Processor) RequestItemConsume(f field.Model, characterId character.Id, itemId item.Id, source slot.Position, quantity int16, updateTime uint32) error` — quantity inserted BEFORE updateTime. Tasks 4 and 6 pass real quantities through this.

- [ ] **Step 1: Write the failing producer test**

Create `services/atlas-channel/atlas.com/channel/consumable/producer_test.go` (pins FR-1.4: the emitted command carries the given quantity; modeled on `food/producer_test.go`):

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
// wire (FR-1.4) — the skill-cast path now sends values > 1.
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
	if cmd.CharacterId != cid {
		t.Errorf("characterId: got %d, want %d", cmd.CharacterId, cid)
	}
	if cmd.Type != consumablemsg.CommandRequestItemConsume {
		t.Errorf("type: got %q, want %q", cmd.Type, consumablemsg.CommandRequestItemConsume)
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

- [ ] **Step 2: Run the test — it should PASS already**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./consumable/ -run TestRequestItemConsumeCommandProvider -v`
Expected: PASS — the provider already takes quantity; this test pins the wire contract before the processor change. (This step is a regression pin, not red-green.)

- [ ] **Step 3: Change the processor signature**

In `services/atlas-channel/atlas.com/channel/consumable/processor.go`, replace `RequestItemConsume`:

```go
func (p *Processor) RequestItemConsume(f field.Model, characterId character.Id, itemId item.Id, source slot.Position, quantity int16, updateTime uint32) error {
	// Defense in depth for the WZ "0"-string itemConNo default (FR-1.1):
	// an absent amount means one item, never zero.
	if quantity < 1 {
		quantity = 1
	}
	p.l.Debugf("Character [%d] using item [%d] from slot [%d]. quantity [%d], updateTime [%d]", characterId, itemId, source, quantity, updateTime)
	return producer.ProviderImpl(p.l)(p.ctx)(consumable2.EnvCommandTopic)(RequestItemConsumeCommandProvider(f, characterId, source, itemId, quantity))
}
```

- [ ] **Step 4: Sweep the call sites (compiler-enforced)**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./...`
Expected: FAIL listing every call site — exactly these seven:

| File | Line(s) | Change |
|---|---|---|
| `skill/handler/common.go` | 84 | `..., slot.Position(a.Slot()), 1, 0)` (literal 1 for now; Task 4 makes it the effect amount) |
| `socket/handler/pet_food.go` | 22 | `..., slot.Position(p.Source()), 1, p.UpdateTime())` |
| `socket/handler/pet_item_use.go` | 22 | `..., slot.Position(p.Source()), 1, p.UpdateTime())` |
| `socket/handler/character_cash_item_use.go` | 51 | `..., source, 1, updateTime)` |
| `socket/handler/character_item_use.go` | 22, 31, 49 | `..., slot.Position(p.Source()), 1, p.UpdateTime())` |

In each, insert the literal `1` as the new `quantity` argument before the final `updateTime` argument. Do not touch any other argument.

- [ ] **Step 5: Verify the module compiles and tests pass**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test -race ./consumable/... ./socket/handler/... ./skill/handler/...`
Expected: build clean, all tests PASS

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/consumable/processor.go services/atlas-channel/atlas.com/channel/consumable/producer_test.go services/atlas-channel/atlas.com/channel/skill/handler/common.go services/atlas-channel/atlas.com/channel/socket/handler/pet_food.go services/atlas-channel/atlas.com/channel/socket/handler/pet_item_use.go services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go services/atlas-channel/atlas.com/channel/socket/handler/character_item_use.go
git commit -m "feat(channel): RequestItemConsume carries an explicit quantity (task-160 FR-1.2)"
```

---

### Task 4: `UseSkill` itemCon path — real amount + lowest-qualifying-slot + seams

Wire `ItemConsumeAmount()` through the cast path (FR-1.1, FR-1.3) and introduce the three test seams the design names (§6): `loadCasterWithInventoryFunc`, `requestItemConsumeFunc`, `applyBuffStatupsFunc`. Also hoist `statups := e.StatUps()` into a local so Task 6 can rewrite it.

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/skill/handler/common.go` (seams at top; itemCon block; buff-apply block)
- Test: `services/atlas-channel/atlas.com/channel/skill/handler/common_consume_test.go` (create)

**Interfaces:**
- Consumes: Task 2's `FindFirstByItemIdWithQuantity(templateId uint32, quantity int16) (*asset.Model, bool)`; Task 3's `RequestItemConsume(..., quantity int16, updateTime uint32)`.
- Produces (used again by Task 6):
  - `var loadCasterWithInventoryFunc func(cp character.Processor, characterId uint32) (character.Model, error)`
  - `var requestItemConsumeFunc func(p *consumable.Processor, f field.Model, characterId charcon.Id, itemId itemconst.Id, source slot.Position, quantity int16, updateTime uint32) error`
  - `var applyBuffStatupsFunc func(l logrus.FieldLogger, ctx context.Context, f field.Model, characterId uint32, info packetmodel.SkillUsageInfo, duration int32, statups []statup.Model)`
  - `UseSkill` holds a local `statups := e.StatUps()` that the buff block consumes.

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
	"atlas-channel/data/skill/effect/statup"
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

// testConsumeSkillId is an arbitrary id with no mount/registry/party/mob
// classification so UseSkill exercises only the consume + buff paths.
const testConsumeSkillId = uint32(99999999)

// magicRockItemId is Magic Rock (WZ itemCon for several casts); classification
// 400 -> ETC compartment via inventoryconst.TypeFromItemId.
const magicRockItemId = uint32(4006000)

type consumeRecorder struct {
	called     bool
	itemId     itemconst.Id
	source     slot.Position
	quantity   int16
	buffCalled bool
	buffStatups []statup.Model
	buffDuration int32
}

// overrideCastSeams replaces the three UseSkill seams with recorders and
// restores them via t.Cleanup. The caster returned by the load seam is the
// provided model.
func overrideCastSeams(t *testing.T, c character.Model, loadErr error) *consumeRecorder {
	t.Helper()
	rec := &consumeRecorder{}

	prevLoad := loadCasterWithInventoryFunc
	loadCasterWithInventoryFunc = func(_ character.Processor, _ uint32) (character.Model, error) {
		return c, loadErr
	}
	prevConsume := requestItemConsumeFunc
	requestItemConsumeFunc = func(_ *consumable.Processor, _ field.Model, _ charcon.Id, itemId itemconst.Id, source slot.Position, quantity int16, _ uint32) error {
		rec.called = true
		rec.itemId = itemId
		rec.source = source
		rec.quantity = quantity
		return nil
	}
	prevBuff := applyBuffStatupsFunc
	applyBuffStatupsFunc = func(_ logrus.FieldLogger, _ context.Context, _ field.Model, _ uint32, _ packetmodel.SkillUsageInfo, duration int32, statups []statup.Model) {
		rec.buffCalled = true
		rec.buffDuration = duration
		rec.buffStatups = statups
	}
	t.Cleanup(func() {
		loadCasterWithInventoryFunc = prevLoad
		requestItemConsumeFunc = prevConsume
		applyBuffStatupsFunc = prevBuff
	})
	return rec
}

func testField() field.Model {
	return field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()
}

func consumeAsset(slotIdx int16, templateId uint32, qty uint32) asset.Model {
	return asset.NewModelBuilder(1, uuid.New(), templateId).
		SetSlot(slotIdx).
		SetQuantity(qty).
		MustBuild()
}

// casterWithCompartment builds a character whose compartment for the given
// inventory type holds the provided assets.
func casterWithCompartment(t *testing.T, it inventoryconst.Type, assets ...asset.Model) character.Model {
	t.Helper()
	cb := compartment.NewBuilder(uuid.New(), 1, it, 96)
	for _, a := range assets {
		cb.AddAsset(a)
	}
	inv := inventory.NewBuilder(1).SetCompartment(cb.MustBuild()).MustBuild()
	return character.NewModelBuilder().SetId(1).SetInventory(inv).MustBuild()
}

func consumeEffect(t *testing.T, itemConsume uint32, amount uint32, duration int32, statups []statup.RestModel) effect.Model {
	t.Helper()
	e, err := effect.Extract(effect.RestModel{
		ItemConsume:       itemConsume,
		ItemConsumeAmount: amount,
		Duration:          duration,
		Statups:           statups,
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
	l := logrus.New()
	if err := UseSkill(l)(context.Background())(nil, testField(), 1, useSkillInfo(), e); err != nil {
		t.Fatalf("UseSkill: %v", err)
	}
}

func TestUseSkill_ItemConsumeAmountPlumbed(t *testing.T) {
	it, _ := inventoryconst.TypeFromItemId(itemconst.Id(magicRockItemId))
	// Slot 1 is short (1 < 2); slot 2 qualifies -> quantity 2 from slot 2.
	c := casterWithCompartment(t, it,
		consumeAsset(1, magicRockItemId, 1),
		consumeAsset(2, magicRockItemId, 3),
	)
	rec := overrideCastSeams(t, c, nil)

	runUseSkill(t, consumeEffect(t, magicRockItemId, 2, 0, nil))

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
	c := casterWithCompartment(t, it, consumeAsset(1, magicRockItemId, 5))
	rec := overrideCastSeams(t, c, nil)

	runUseSkill(t, consumeEffect(t, magicRockItemId, 0, 0, nil))

	if !rec.called || rec.quantity != 1 {
		t.Fatalf("got (called=%v, quantity=%d), want (true, 1)", rec.called, rec.quantity)
	}
}

func TestUseSkill_ItemConShortfallSkipsButCastProceeds(t *testing.T) {
	it, _ := inventoryconst.TypeFromItemId(itemconst.Id(magicRockItemId))
	// Two slots of 1 each; no single slot holds 2 -> skip, warn, cast proceeds.
	c := casterWithCompartment(t, it,
		consumeAsset(1, magicRockItemId, 1),
		consumeAsset(2, magicRockItemId, 1),
	)
	rec := overrideCastSeams(t, c, nil)

	runUseSkill(t, consumeEffect(t, magicRockItemId, 2, 60000, []statup.RestModel{{Type: "WATK", Amount: 10}}))

	if rec.called {
		t.Error("expected NO consume request on shortfall")
	}
	if !rec.buffCalled {
		t.Error("expected the buff to still apply (cast proceeds, FR-1.3)")
	}
}

func TestUseSkill_ItemConCasterLoadFailureCastProceeds(t *testing.T) {
	rec := overrideCastSeams(t, character.Model{}, errors.New("boom"))

	runUseSkill(t, consumeEffect(t, magicRockItemId, 2, 60000, []statup.RestModel{{Type: "WATK", Amount: 10}}))

	if rec.called {
		t.Error("expected NO consume request on load failure")
	}
	if !rec.buffCalled {
		t.Error("expected the buff to still apply (cast permitted)")
	}
}
```

Note for the implementer: `inventory.NewBuilder(1).SetCompartment(...)` keys by the compartment's own type (`inventory/builder.go:54` — `b.compartments[m.Type()] = m`), so it stores correctly whichever inventory type `TypeFromItemId(4006000)` resolves to; the test builds the compartment with that same resolved type, so lookup and storage always agree.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./skill/handler/ -run TestUseSkill_ItemCon -v`
Expected: FAIL to compile with `undefined: loadCasterWithInventoryFunc` (and the other seams)

- [ ] **Step 3: Implement seams and the itemCon block**

In `services/atlas-channel/atlas.com/channel/skill/handler/common.go`:

1. Add `"atlas-channel/data/skill/effect/statup"` to the imports.

2. After the existing `loadCasterFunc` var (line ~33), add the three seams:

```go
// loadCasterWithInventoryFunc is the inventory-decorated caster-load seam
// tests can replace. The consume paths need Inventory() (and, for the
// bullet-consume gate, Equipment()) populated, which the decorator provides.
var loadCasterWithInventoryFunc = func(cp character.Processor, characterId uint32) (character.Model, error) {
	return cp.GetById(cp.InventoryDecorator)(characterId)
}

// requestItemConsumeFunc is the consumable-request seam tests can replace.
// Production delegates to consumable.Processor.RequestItemConsume.
var requestItemConsumeFunc = func(p *consumable.Processor, f field.Model, characterId charcon.Id, itemId itemconst.Id, source slot.Position, quantity int16, updateTime uint32) error {
	return p.RequestItemConsume(f, characterId, itemId, source, quantity, updateTime)
}

// applyBuffStatupsFunc is the buff-apply seam tests can replace. It wraps
// both the self-apply and the party fan-out so a single override captures
// the statups a cast actually applied.
var applyBuffStatupsFunc = func(l logrus.FieldLogger, ctx context.Context, f field.Model, characterId uint32, info packetmodel.SkillUsageInfo, duration int32, statups []statup.Model) {
	applyBuffFunc := buff.NewProcessor(l, ctx).Apply(f, characterId, int32(info.SkillId()), info.SkillLevel(), duration, statups)
	_ = applyBuffFunc(characterId)
	_ = applyToParty(l)(ctx)(f, characterId, info.AffectedPartyMemberBitmap())(applyBuffFunc)
}
```

3. At the top of `UseSkill`'s innermost function (before the `e.HPConsume()` block), add:

```go
			statups := e.StatUps()
```

4. Replace the itemCon block (currently lines 79–92) with:

```go
			if itemId := e.ItemConsume(); itemId > 0 {
				cp := character.NewProcessor(l, ctx)
				if c, cErr := loadCasterWithInventoryFunc(cp, characterId); cErr == nil {
					if invType, typeOk := inventoryconst.TypeFromItemId(itemconst.Id(itemId)); typeOk {
						amount := int16(e.ItemConsumeAmount())
						if amount < 1 {
							// Absent itemConNo (reader default 0) means one item (FR-1.1).
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

5. Replace the buff-apply block (currently lines 107–111) with:

```go
			if e.Duration() > 0 && len(statups) > 0 {
				applyBuffStatupsFunc(l, ctx, f, characterId, info, e.Duration(), statups)
			}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go test -race ./skill/handler/... -v`
Expected: PASS — the four new tests plus every pre-existing `skill/handler` test (mount, apply-to-mobs, registry, recipients).

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/skill/handler/common.go services/atlas-channel/atlas.com/channel/skill/handler/common_consume_test.go
git commit -m "feat(channel): UseSkill consumes the WZ itemConNo amount from a qualifying slot (task-160 FR-1)"
```

---

### Task 5: `SpiritJavelinItemId()` getter on `SkillUsageInfo`

The decoder already reads the field for skill 4121006 (`skill_usage_info.go:32`) and the builder already sets it; only the getter is missing (design §4.5).

**Files:**
- Modify: `libs/atlas-packet/model/skill_usage_info.go` (add getter alongside the other getters, after `Delay()`)
- Test: `libs/atlas-packet/model/skill_usage_info_test.go` (add test)

**Interfaces:**
- Produces: `func (m *SkillUsageInfo) SpiritJavelinItemId() uint32` — Task 6 uses it as the slot-selection hint.

- [ ] **Step 1: Write the failing test**

Append to `libs/atlas-packet/model/skill_usage_info_test.go` (match the file's existing package declaration and import style):

```go
func TestSkillUsageInfo_SpiritJavelinItemIdGetter(t *testing.T) {
	info := NewSkillUsageInfoBuilder().SetSpiritJavelinItemId(2070006).Build()
	if got := info.SpiritJavelinItemId(); got != 2070006 {
		t.Fatalf("SpiritJavelinItemId: got %d, want 2070006", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-packet && go test ./model/ -run TestSkillUsageInfo_SpiritJavelinItemIdGetter -v`
Expected: FAIL to compile with `info.SpiritJavelinItemId undefined`

- [ ] **Step 3: Add the getter**

In `libs/atlas-packet/model/skill_usage_info.go`, after `Delay()`:

```go
// SpiritJavelinItemId returns the star item id the client chose for a
// Shadow Stars (SpiritJavelin) cast; 0 for every other skill (the decoder
// only reads the field for skill.NightLordShadowStarsId).
func (m *SkillUsageInfo) SpiritJavelinItemId() uint32 {
	return m.spiritJavelinItemId
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd libs/atlas-packet && go test -race ./model/ -run TestSkillUsageInfo -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-packet/model/skill_usage_info.go libs/atlas-packet/model/skill_usage_info_test.go
git commit -m "feat(packet): expose SpiritJavelinItemId on SkillUsageInfo (task-160)"
```

---

### Task 6: Cast-time `bulletConsume` gate (`bullet_consume.go`)

The core FR-2 feature: when `e.BulletConsume() > 0`, settle the star cost before anything else in `UseSkill` — find a qualifying USE-compartment slot (hint-preferred), request the consume, and rewrite the SHADOW_CLAW statup amount to `consumedStarItemId − 2069999` (design §2.1 — without this the client resolves bullet item 2069999, finds nothing, and refuses to attack). No qualifying slot → reject the whole cast with zero side effects (FR-2.3).

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/skill/handler/bullet_consume.go`
- Modify: `services/atlas-channel/atlas.com/channel/skill/handler/common.go` (wire the gate at the top of `UseSkill`)
- Test: `services/atlas-channel/atlas.com/channel/skill/handler/bullet_consume_test.go` (create)

**Interfaces:**
- Consumes: Task 1's `itemconst.ProjectileClassificationForWeapon`, Task 4's `loadCasterWithInventoryFunc` / `requestItemConsumeFunc` / `applyBuffStatupsFunc` seams and `statups` local, Task 5's `info.SpiritJavelinItemId()`, `equipment.Model.Get("weapon")` (slot value `slot.Model{Position, Equipable *asset.Model}`), `charconst.TemporaryStatTypeShadowClaw` (`"SHADOW_CLAW"`).
- Produces: `func consumeCastBullets(l logrus.FieldLogger, ctx context.Context, f field.Model, characterId uint32, info packetmodel.SkillUsageInfo, e effect.Model) ([]statup.Model, bool)` and `func rewriteShadowClawAmount(statups []statup.Model, amount int32) []statup.Model` (both unexported, same package).

- [ ] **Step 1: Write the failing tests**

Create `services/atlas-channel/atlas.com/channel/skill/handler/bullet_consume_test.go`. It reuses `overrideCastSeams`, `testField`, `consumeAsset`, and `runUseSkill` from `common_consume_test.go` (same package):

```go
package handler

import (
	"context"
	"errors"
	"testing"

	"atlas-channel/asset"
	"atlas-channel/character"
	"atlas-channel/compartment"
	"atlas-channel/data/skill/effect"
	"atlas-channel/data/skill/effect/statup"
	"atlas-channel/equipment"
	eqslot "atlas-channel/equipment/slot"
	"atlas-channel/inventory"

	charconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	inventoryconst "github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const (
	// Claw weapon: (1472000/10000)%100 == 47 -> WeaponTypeClaw.
	clawWeaponItemId = uint32(1472000)
	// Bow weapon: (1452000/10000)%100 == 45 -> WeaponTypeBow.
	bowWeaponItemId = uint32(1452000)
	// Two throwing-star templates (classification 207).
	starSubiItemId  = uint32(2070000)
	starOtherItemId = uint32(2070001)
	// An arrow (classification 206) — invalid as a claw hint.
	arrowItemId = uint32(2060000)
)

// bulletCaster builds a character with the given weapon equipped and the
// given assets in the USE compartment.
func bulletCaster(t *testing.T, weaponItemId uint32, stars ...asset.Model) character.Model {
	t.Helper()
	weapon := asset.NewModelBuilder(99, uuid.New(), weaponItemId).SetSlot(-11).MustBuild()
	em := equipment.NewModel()
	em.Set("weapon", eqslot.Model{Position: slot.Position(-11), Equipable: &weapon})

	cb := compartment.NewBuilder(uuid.New(), 1, inventoryconst.TypeValueUse, 96)
	for _, a := range stars {
		cb.AddAsset(a)
	}
	inv := inventory.NewBuilder(1).SetConsumable(cb.MustBuild()).MustBuild()
	return character.NewModelBuilder().SetId(1).SetEquipment(em).SetInventory(inv).MustBuild()
}

// shadowStarsEffect builds an effect with bulletConsume=200, a duration, and
// a zero-amount SHADOW_CLAW statup (what atlas-data emits today).
func shadowStarsEffect(t *testing.T) effect.Model {
	t.Helper()
	e, err := effect.Extract(effect.RestModel{
		BulletConsume: 200,
		Duration:      100000,
		Statups: []statup.RestModel{
			{Type: string(charconst.TemporaryStatTypeShadowClaw), Amount: 0},
		},
	})
	if err != nil {
		t.Fatalf("effect extract: %v", err)
	}
	return e
}

func bulletInfo(hintItemId uint32) packetmodel.SkillUsageInfo {
	return packetmodel.NewSkillUsageInfoBuilder().
		SetSkillId(testConsumeSkillId).
		SetSkillLevel(1).
		SetSpiritJavelinItemId(hintItemId).
		Build()
}

func shadowClawAmount(t *testing.T, statups []statup.Model) int32 {
	t.Helper()
	for _, su := range statups {
		if su.Mask() == string(charconst.TemporaryStatTypeShadowClaw) {
			return su.Amount()
		}
	}
	t.Fatal("no SHADOW_CLAW statup found")
	return 0
}

func runUseSkillWithInfo(t *testing.T, info packetmodel.SkillUsageInfo, e effect.Model) {
	t.Helper()
	if err := UseSkill(logrus.New())(context.Background())(nil, testField(), 1, info, e); err != nil {
		t.Fatalf("UseSkill: %v", err)
	}
}

func TestUseSkill_BulletConsume_HintHonored(t *testing.T) {
	// Both templates qualify; the hint pins template 2070001 in slot 2.
	c := bulletCaster(t, clawWeaponItemId,
		consumeAsset(1, starSubiItemId, 250),
		consumeAsset(2, starOtherItemId, 250),
	)
	rec := overrideCastSeams(t, c, nil)

	runUseSkillWithInfo(t, bulletInfo(starOtherItemId), shadowStarsEffect(t))

	if !rec.called {
		t.Fatal("expected a consume request")
	}
	if rec.quantity != 200 {
		t.Errorf("quantity: got %d, want 200", rec.quantity)
	}
	if uint32(rec.itemId) != starOtherItemId || rec.source != slot.Position(2) {
		t.Errorf("draw: got (item=%d, slot=%d), want (%d, 2)", rec.itemId, rec.source, starOtherItemId)
	}
	if !rec.buffCalled {
		t.Fatal("expected the buff to apply")
	}
	if got := shadowClawAmount(t, rec.buffStatups); got != int32(starOtherItemId)-2069999 {
		t.Errorf("SHADOW_CLAW amount: got %d, want %d", got, int32(starOtherItemId)-2069999)
	}
}

func TestUseSkill_BulletConsume_NoHintLowestQualifyingSlot(t *testing.T) {
	// No hint: slot 1 is short (150 < 200); slot 3 qualifies.
	c := bulletCaster(t, clawWeaponItemId,
		consumeAsset(1, starSubiItemId, 150),
		consumeAsset(3, starOtherItemId, 200),
	)
	rec := overrideCastSeams(t, c, nil)

	runUseSkillWithInfo(t, bulletInfo(0), shadowStarsEffect(t))

	if !rec.called || rec.source != slot.Position(3) {
		t.Fatalf("got (called=%v, slot=%d), want (true, 3)", rec.called, rec.source)
	}
	if got := shadowClawAmount(t, rec.buffStatups); got != int32(starOtherItemId)-2069999 {
		t.Errorf("SHADOW_CLAW amount: got %d, want %d", got, int32(starOtherItemId)-2069999)
	}
}

func TestUseSkill_BulletConsume_InvalidHintFallsBackToScan(t *testing.T) {
	// Hint is an arrow (classification 206 != 207) -> forgery guard ignores it.
	c := bulletCaster(t, clawWeaponItemId, consumeAsset(1, starSubiItemId, 200))
	rec := overrideCastSeams(t, c, nil)

	runUseSkillWithInfo(t, bulletInfo(arrowItemId), shadowStarsEffect(t))

	if !rec.called || uint32(rec.itemId) != starSubiItemId {
		t.Fatalf("got (called=%v, item=%d), want (true, %d)", rec.called, rec.itemId, starSubiItemId)
	}
}

func TestUseSkill_BulletConsume_NoQualifyingSlotRejectsCast(t *testing.T) {
	// 150+150 across two slots: aggregate is enough but no single slot is
	// (design §5.3 single-slot rule) -> zero side effects.
	c := bulletCaster(t, clawWeaponItemId,
		consumeAsset(1, starSubiItemId, 150),
		consumeAsset(2, starSubiItemId, 150),
	)
	rec := overrideCastSeams(t, c, nil)

	runUseSkillWithInfo(t, bulletInfo(0), shadowStarsEffect(t))

	if rec.called {
		t.Error("expected NO consume request (FR-2.3)")
	}
	if rec.buffCalled {
		t.Error("expected NO buff apply (FR-2.3)")
	}
}

func TestUseSkill_BulletConsume_NonRangedWeaponRejectsCast(t *testing.T) {
	// A claw-gated cast arriving with a non-ranged weapon implies desync or
	// forgery -> reject. (Sword: (1302000/10000)%100 == 30 -> one-handed sword.)
	c := bulletCaster(t, uint32(1302000), consumeAsset(1, starSubiItemId, 200))
	rec := overrideCastSeams(t, c, nil)

	runUseSkillWithInfo(t, bulletInfo(0), shadowStarsEffect(t))

	if rec.called || rec.buffCalled {
		t.Fatalf("expected zero side effects, got (consume=%v, buff=%v)", rec.called, rec.buffCalled)
	}
}

func TestUseSkill_BulletConsume_CasterLoadFailureRejectsCast(t *testing.T) {
	rec := overrideCastSeams(t, character.Model{}, errors.New("boom"))

	runUseSkillWithInfo(t, bulletInfo(0), shadowStarsEffect(t))

	if rec.called || rec.buffCalled {
		t.Fatalf("expected zero side effects, got (consume=%v, buff=%v)", rec.called, rec.buffCalled)
	}
}

func TestRewriteShadowClawAmount(t *testing.T) {
	in := []statup.Model{
		statup.NewModel("WATK", 10),
		statup.NewModel(string(charconst.TemporaryStatTypeShadowClaw), 0),
	}
	out := rewriteShadowClawAmount(in, 7)
	if len(out) != 2 {
		t.Fatalf("len: got %d, want 2", len(out))
	}
	if out[0].Mask() != "WATK" || out[0].Amount() != 10 {
		t.Errorf("non-SHADOW_CLAW entry must pass through unchanged: %+v", out[0])
	}
	if out[1].Amount() != 7 {
		t.Errorf("SHADOW_CLAW amount: got %d, want 7", out[1].Amount())
	}
	// Input slice must not be mutated (copy semantics).
	if in[1].Amount() != 0 {
		t.Errorf("input mutated: %d", in[1].Amount())
	}
}
```

Notes for the implementer:
- `overrideCastSeams`, `testField`, `consumeAsset`, and `testConsumeSkillId` come from Task 4's `common_consume_test.go` (same package) — do not redefine them.
- `equipment.Model.Set` has a pointer receiver — call it on an addressable `em := equipment.NewModel()` local as shown.
- `"WATK"` here is only an opaque non-SHADOW_CLAW mask for pass-through assertions; the buff pipeline treats masks as opaque strings.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./skill/handler/ -run 'TestUseSkill_BulletConsume|TestRewriteShadowClawAmount' -v`
Expected: FAIL to compile with `undefined: rewriteShadowClawAmount` (and the gate not rejecting yet)

- [ ] **Step 3: Write `bullet_consume.go`**

Create `services/atlas-channel/atlas.com/channel/skill/handler/bullet_consume.go`:

```go
package handler

import (
	"atlas-channel/asset"
	"atlas-channel/character"
	"atlas-channel/consumable"
	"atlas-channel/data/skill/effect"
	"atlas-channel/data/skill/effect/statup"
	"context"
	"sort"

	charconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	itemconst "github.com/Chronicle20/atlas/libs/atlas-constants/item"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/sirupsen/logrus"
)

// shadowClawStarEncodingBase converts a throwing-star item id to the int16
// value the client expects in the SHADOW_CLAW temporary stat: the client
// computes bulletItemId = value + 2069999 in CUserLocal::GetProperBulletPosition
// (IDA: v83 0x949C4C, v87 0x9C4A50, v95 0x907461, jms185 0xA0A2F4 — all
// +0x1F95EF). Amount 0 resolves to nonexistent item 2069999 and the client
// refuses to attack, so the cast path MUST rewrite it (design §2.1).
const shadowClawStarEncodingBase = 2069999

// consumeCastBullets settles a bulletConsume cast cost (FR-2): it picks a
// single USE-compartment slot holding at least e.BulletConsume() projectiles
// matching the equipped weapon (preferring the client's SpiritJavelin item-id
// hint), requests their consumption, and returns the effect's statups with
// the SHADOW_CLAW amount re-encoded for the consumed star. Returns
// (nil, false) when the cast must be rejected with zero side effects
// (FR-2.3): caster load failure, non-ranged weapon, or no qualifying slot.
func consumeCastBullets(l logrus.FieldLogger, ctx context.Context, f field.Model, characterId uint32, info packetmodel.SkillUsageInfo, e effect.Model) ([]statup.Model, bool) {
	required := int64(e.BulletConsume())

	cp := character.NewProcessor(l, ctx)
	c, err := loadCasterWithInventoryFunc(cp, characterId)
	if err != nil {
		l.WithError(err).Warnf("Character [%d] cast skill [%d] costing [%d] projectiles but caster load failed; cast rejected.", characterId, info.SkillId(), required)
		return nil, false
	}

	s, ok := c.Equipment().Get("weapon")
	if !ok || s.Equipable == nil {
		l.Warnf("Character [%d] cast skill [%d] costing [%d] projectiles with no weapon equipped; cast rejected.", characterId, info.SkillId(), required)
		return nil, false
	}
	weaponType := itemconst.GetWeaponType(itemconst.Id(s.Equipable.TemplateId()))
	classification, ranged := itemconst.ProjectileClassificationForWeapon(weaponType)
	if !ranged {
		// The client's own cast gate requires a claw; reaching here implies
		// desync or forgery (design §4.3 step 3).
		l.Warnf("Character [%d] cast skill [%d] costing [%d] projectiles with non-ranged weapon [%d]; cast rejected.", characterId, info.SkillId(), required, s.Equipable.TemplateId())
		return nil, false
	}

	matching := make([]asset.Model, 0)
	for _, a := range c.Inventory().Consumable().Assets() {
		if itemconst.GetClassification(itemconst.Id(a.TemplateId())) == classification && a.Quantity() > 0 {
			matching = append(matching, a)
		}
	}
	sort.Slice(matching, func(i, j int) bool { return matching[i].Slot() < matching[j].Slot() })

	// Hint-preferred selection (§2.2): a classification-valid SpiritJavelin
	// hint restricts candidates to the client-chosen star; an invalid hint
	// is ignored (forgery guard) and the generic scan applies.
	if hint := info.SpiritJavelinItemId(); hint != 0 && itemconst.GetClassification(itemconst.Id(hint)) == classification {
		restricted := make([]asset.Model, 0, len(matching))
		for _, a := range matching {
			if a.TemplateId() == hint {
				restricted = append(restricted, a)
			}
		}
		if len(restricted) > 0 {
			matching = restricted
		}
	}

	var chosen *asset.Model
	for _, a := range matching {
		if int64(a.Quantity()) >= required {
			a := a
			chosen = &a
			break
		}
	}
	if chosen == nil {
		l.Warnf("Character [%d] cast skill [%d] costing [%d] projectiles of classification [%d] but no single slot holds enough; cast rejected (FR-2.3).", characterId, info.SkillId(), required, classification)
		return nil, false
	}

	_ = requestItemConsumeFunc(consumable.NewProcessor(l, ctx), f, charconst.Id(characterId), itemconst.Id(chosen.TemplateId()), slot.Position(chosen.Slot()), int16(e.BulletConsume()), 0)

	return rewriteShadowClawAmount(e.StatUps(), int32(chosen.TemplateId())-shadowClawStarEncodingBase), true
}

// rewriteShadowClawAmount returns a copy of statups with the SHADOW_CLAW
// entry's amount replaced (the runtime-synthesized-amount precedent is the
// mount path's MONSTER_RIDING, mount.go). Entries without a SHADOW_CLAW mask
// pass through unchanged; if none is present the copy equals the input —
// appending one here would wrongly grant free-throw semantics to a
// hypothetical non-Shadow-Stars bulletConsume buff.
func rewriteShadowClawAmount(statups []statup.Model, amount int32) []statup.Model {
	out := make([]statup.Model, 0, len(statups))
	for _, su := range statups {
		if su.Mask() == string(charconst.TemporaryStatTypeShadowClaw) {
			out = append(out, statup.NewModel(su.Mask(), amount))
		} else {
			out = append(out, su)
		}
	}
	return out
}
```

- [ ] **Step 4: Wire the gate into `UseSkill`**

In `common.go`, change the line added in Task 4 from:

```go
			statups := e.StatUps()
```

to (still before the `e.HPConsume()` block — a rejected cast must not consume HP/MP, design §4.3 step 5):

```go
			statups := e.StatUps()
			if e.BulletConsume() > 0 {
				var ok bool
				if statups, ok = consumeCastBullets(l, ctx, f, characterId, info, e); !ok {
					return nil
				}
			}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go test -race ./skill/handler/... -v`
Expected: PASS — all seven new tests plus Task 4's tests plus pre-existing ones. (Task 4's itemCon tests use effects with `BulletConsume: 0`, so the gate is inert there.)

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/skill/handler/bullet_consume.go services/atlas-channel/atlas.com/channel/skill/handler/bullet_consume_test.go services/atlas-channel/atlas.com/channel/skill/handler/common.go
git commit -m "feat(channel): cast-time bulletConsume with SHADOW_CLAW star encoding (task-160 FR-2)"
```

---

### Task 7: Projectile-attack SHADOW_CLAW skip (FR-3)

While SpiritJavelin is active the client requires only `quantity > 0` of the encoded star and consumes nothing per throw (design §2.1). Mirror that: claw attacks under SHADOW_CLAW skip consumption entirely, placed before `computeCount` so Shadow Partner's ×2 cannot resurrect a consume (FR-3.1).

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_projectile.go` (insert skip after the Soul Arrow skip)
- Test: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_projectile_test.go` (add `Plan()`-level tests)

**Interfaces:**
- Consumes: existing `hasBuff(buffs []buff.Model, statType ts.TemporaryStatType) bool`, `ts.TemporaryStatTypeShadowClaw`, `buff.Processor` interface (`ByCharacterIdProvider`, `GetByCharacterId`, `Apply`, `Cancel`).
- Produces: no new symbols — behavior change inside `ProjectileProcessorImpl.Plan`.

- [ ] **Step 1: Write the failing tests**

Append to `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_projectile_test.go` (extend the existing import block as needed with: `context`, stdlib `errors`, `atlas-channel/character`, `atlas-channel/compartment`, `statup "atlas-channel/data/skill/effect/statup"`, `atlas-channel/equipment`, `eqslot "atlas-channel/equipment/slot"`, `atlas-channel/inventory`, `invslot "github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"`, `inventoryconst "github.com/Chronicle20/atlas/libs/atlas-constants/inventory"`, `model2 "github.com/Chronicle20/atlas/libs/atlas-model/model"`, `"github.com/Chronicle20/atlas/libs/atlas-constants/field"`, `packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"`, `"github.com/sirupsen/logrus"`):

```go
// stubBuffProcessor satisfies buff.Processor for Plan() tests; only
// GetByCharacterId is exercised.
type stubBuffProcessor struct {
	buffs []buff.Model
	err   error
}

func (s *stubBuffProcessor) ByCharacterIdProvider(_ uint32) model2.Provider[[]buff.Model] {
	return func() ([]buff.Model, error) { return s.buffs, s.err }
}
func (s *stubBuffProcessor) GetByCharacterId(_ uint32) ([]buff.Model, error) {
	return s.buffs, s.err
}
func (s *stubBuffProcessor) Apply(_ field.Model, _ uint32, _ int32, _ byte, _ int32, _ []statup.Model) model2.Operator[uint32] {
	return func(_ uint32) error { return nil }
}
func (s *stubBuffProcessor) Cancel(_ field.Model, _ uint32, _ int32) error { return nil }

// planCharacter builds a character with the given weapon equipped and the
// given USE-compartment assets, for driving Plan() directly.
func planCharacter(t *testing.T, weaponItemId uint32, assets ...asset.Model) character.Model {
	t.Helper()
	weapon := asset.NewModelBuilder(99, uuid.New(), weaponItemId).SetSlot(-11).MustBuild()
	em := equipment.NewModel()
	em.Set("weapon", eqslot.Model{Position: invslot.Position(-11), Equipable: &weapon})

	cb := compartment.NewBuilder(uuid.New(), 1, inventoryconst.TypeValueUse, 96)
	for _, a := range assets {
		cb.AddAsset(a)
	}
	inv := inventory.NewBuilder(1).SetConsumable(cb.MustBuild()).MustBuild()
	return character.NewModelBuilder().SetId(1).SetEquipment(em).SetInventory(inv).MustBuild()
}

func planProcessor(bp buff.Processor) *ProjectileProcessorImpl {
	return &ProjectileProcessorImpl{l: logrus.New(), ctx: context.Background(), bp: bp}
}

const (
	planClawItemId = uint32(1472000) // (1472000/10000)%100 == 47 -> claw
	planBowItemId  = uint32(1452000) // (1452000/10000)%100 == 45 -> bow
)

func rangedAttack() packetmodel.AttackInfo {
	// Skill id 0 (normal attack) is in no skip list; AttackTypeRanged drives
	// the projectile path. ProperBulletPosition defaults to 0 (no hint).
	return *packetmodel.NewAttackInfo(packetmodel.AttackTypeRanged)
}

func TestPlan_ClawWithShadowClawSkipsConsumption(t *testing.T) {
	c := planCharacter(t, planClawItemId, makeAsset(1, throwingStarSubi, 100))
	bp := &stubBuffProcessor{buffs: []buff.Model{buffWithStat(ts.TemporaryStatTypeShadowClaw)}}

	plan, ok := planProcessor(bp).Plan(c, rangedAttack(), effect.Model{})
	if ok || plan != nil {
		t.Fatalf("got (plan=%v, ok=%v), want (nil, false): SHADOW_CLAW skip (FR-3.1)", plan, ok)
	}
}

func TestPlan_ClawWithShadowClawAndShadowPartnerStillSkips(t *testing.T) {
	// The skip precedes computeCount, so Shadow Partner's x2 cannot
	// resurrect a consume (FR-3.1).
	c := planCharacter(t, planClawItemId, makeAsset(1, throwingStarSubi, 100))
	bp := &stubBuffProcessor{buffs: []buff.Model{
		buffWithStat(ts.TemporaryStatTypeShadowClaw),
		buffWithStat(ts.TemporaryStatTypeShadowPartner),
	}}

	plan, ok := planProcessor(bp).Plan(c, rangedAttack(), effect.Model{})
	if ok || plan != nil {
		t.Fatalf("got (plan=%v, ok=%v), want (nil, false)", plan, ok)
	}
}

func TestPlan_ClawWithoutShadowClawConsumes(t *testing.T) {
	c := planCharacter(t, planClawItemId, makeAsset(1, throwingStarSubi, 100))
	bp := &stubBuffProcessor{}

	plan, ok := planProcessor(bp).Plan(c, rangedAttack(), effect.Model{})
	if !ok || plan == nil || len(plan.Draws) != 1 || plan.Draws[0].Quantity != 1 {
		t.Fatalf("got (plan=%+v, ok=%v), want a single 1-star draw", plan, ok)
	}
}

func TestPlan_BowIgnoresShadowClaw(t *testing.T) {
	// The skip is claw-only; a bow user with a stray SHADOW_CLAW buff still
	// consumes arrows.
	c := planCharacter(t, planBowItemId, makeAsset(1, arrowForBow, 100))
	bp := &stubBuffProcessor{buffs: []buff.Model{buffWithStat(ts.TemporaryStatTypeShadowClaw)}}

	plan, ok := planProcessor(bp).Plan(c, rangedAttack(), effect.Model{})
	if !ok || plan == nil || len(plan.Draws) != 1 {
		t.Fatalf("got (plan=%+v, ok=%v), want a single arrow draw", plan, ok)
	}
}

func TestPlan_BowWithSoulArrowStillSkips(t *testing.T) {
	// Soul Arrow regression (AC-7).
	c := planCharacter(t, planBowItemId, makeAsset(1, arrowForBow, 100))
	bp := &stubBuffProcessor{buffs: []buff.Model{buffWithStat(ts.TemporaryStatTypeSoulArrow)}}

	plan, ok := planProcessor(bp).Plan(c, rangedAttack(), effect.Model{})
	if ok || plan != nil {
		t.Fatalf("got (plan=%v, ok=%v), want (nil, false): Soul Arrow skip", plan, ok)
	}
}

func TestPlan_BuffLookupFailureFallsBackToConsuming(t *testing.T) {
	// FR-3.2: a buff-lookup failure is treated as "no buffs" -> over-consume
	// rather than break the attack path.
	c := planCharacter(t, planClawItemId, makeAsset(1, throwingStarSubi, 100))
	bp := &stubBuffProcessor{err: errors.New("boom")}

	plan, ok := planProcessor(bp).Plan(c, rangedAttack(), effect.Model{})
	if !ok || plan == nil || len(plan.Draws) != 1 {
		t.Fatalf("got (plan=%+v, ok=%v), want a single draw despite lookup failure", plan, ok)
	}
}
```

Notes for the implementer:
- `buff.Model`, `buffWithStat`, `makeAsset`, `throwingStarSubi`, `arrowForBow`, `ts` are already imported/defined in this test file.
- The `stubBuffProcessor` method set must match `buff.Processor` (`character/buff/processor.go:16-21`) exactly — `Apply`'s statups parameter is `[]statup.Model` from `atlas-channel/data/skill/effect/statup` as shown, NOT the `buff/stat` package's `stat.Model`.
- `ProjectileProcessorImpl.cpp` stays nil — `Plan()` never touches it (only `Emit` does).

- [ ] **Step 2: Run tests to verify the new ones fail**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run TestPlan_ -v`
Expected: `TestPlan_ClawWithShadowClawSkipsConsumption` and `TestPlan_ClawWithShadowClawAndShadowPartnerStillSkips` FAIL (plan produced, not skipped); the other four PASS (they pin current behavior).

- [ ] **Step 3: Implement the skip**

In `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_projectile.go`, immediately after the Soul Arrow skip block (after line 111), insert:

```go
	if weaponType == item.WeaponTypeClaw && hasBuff(buffs, ts.TemporaryStatTypeShadowClaw) {
		// Shadow Stars: the 200-star cost was paid at cast time; throws are
		// free while the buff lasts (client consumes nothing per throw when
		// SpiritJavelin is active). Placed before computeCount so Shadow
		// Partner's x2 cannot resurrect a consume (FR-3.1).
		p.l.WithField("characterId", c.Id()).WithField("skillId", ai.SkillId()).
			Debugf("Skipping projectile consumption: Shadow Stars active.")
		return nil, false
	}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go test -race ./socket/handler/... -v`
Expected: PASS — all six new `TestPlan_*` tests plus every pre-existing `socket/handler` test.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/character_attack_projectile.go services/atlas-channel/atlas.com/channel/socket/handler/character_attack_projectile_test.go
git commit -m "feat(channel): claw attacks under SHADOW_CLAW consume no projectiles (task-160 FR-3)"
```

---

### Task 8: Full verification sweep

Per CLAUDE.md Build & Verification, in every changed module. No `go.mod` was touched and no new lib was added, so no Dockerfile/`go.work` changes; `atlas-channel` is the only service with code changes, so it is the bake target.

**Files:** none (verification only; fix-and-recommit anything that fails).

- [ ] **Step 1: Test, vet, build every changed module**

```bash
cd services/atlas-channel/atlas.com/channel && go test -race ./... && go vet ./... && go build ./...
cd ../../../../libs/atlas-constants && go test -race ./... && go vet ./... && go build ./...
cd ../atlas-packet && go test -race ./... && go vet ./... && go build ./...
```

Expected: all clean. `-race` matters here — the seam vars are package-level and the tests mutate them; if the race detector flags anything, the tests are missing `t.Cleanup` restores or run in parallel (do NOT add `t.Parallel()` to seam-mutating tests).

- [ ] **Step 2: Bake the service image**

From the worktree root:

```bash
docker buildx bake atlas-channel
```

Expected: builds clean. This is the only check that catches a missing `COPY libs/...` line in the shared Dockerfile (both libs touched here are long-established, but the bake is mandatory regardless).

- [ ] **Step 3: Redis key guard**

From the worktree root (no global `GOWORK=off` prefix — known false-FAIL footgun):

```bash
tools/redis-key-guard.sh
```

Expected: clean.

- [ ] **Step 4: Acceptance-criteria walk**

Re-read PRD §10 and confirm each criterion maps to a passing test or a compile-time fact:

| Criterion | Evidence |
|---|---|
| quantity N / floor to 1 emitted | `TestUseSkill_ItemConsumeAmountPlumbed`, `TestUseSkill_ItemConsumeAmountZeroFloorsToOne`, `TestRequestItemConsumeCommandProvider_CarriesQuantity` |
| lowest qualifying slot; shortfall warns + proceeds | `TestFindFirstByItemIdWithQuantity_*`, `TestUseSkill_ItemConShortfallSkipsButCastProceeds` |
| pre-existing call sites still quantity 1 | Task 3 compile sweep + existing item-use/pet tests passing |
| Shadow Stars qualifying stack: 200 consumed + buff w/ encoded amount | `TestUseSkill_BulletConsume_HintHonored`, `_NoHintLowestQualifyingSlot` |
| no single stack ≥ 200: nothing consumed, no buff, warn | `TestUseSkill_BulletConsume_NoQualifyingSlotRejectsCast` |
| claw attack under SHADOW_CLAW consumes zero (incl. Shadow Partner) | `TestPlan_ClawWithShadowClaw*` |
| Soul Arrow / pet food / item-use / cash-item regressions | `TestPlan_BowWithSoulArrowStillSkips` + pre-existing suites green |
| module + bake + redis-guard checks | Steps 1–3 above |
| §9.1 resolved and implemented | design §2.1 (IDA-verified) + `shadowClawStarEncodingBase` rewrite tests |

- [ ] **Step 5: Commit any verification fixes**

If Steps 1–3 required fixes, commit them:

```bash
git add -A && git commit -m "fix(channel): verification sweep fixes (task-160)"
```

If nothing changed, no commit — the branch is ready for the code-review phase (`superpowers:requesting-code-review` before any PR, per CLAUDE.md).

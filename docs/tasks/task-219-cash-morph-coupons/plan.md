# Cash Transformation (Morph) Coupons — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

Task: `task-219-cash-morph-coupons`
PRD: [`prd.md`](./prd.md) · Design: [`design.md`](./design.md) · Context: [`context.md`](./context.md)
Created: 2026-08-12

**Goal:** Make a `Cash/0530` transformation coupon transform the character for its WZ `time`, heal its WZ `hp`, and decrement one coupon from the **Cash** compartment — closing the three plumbing gaps (atlas-data spec parse, atlas-consumables consume branch, atlas-channel handler arm) plus one serverbound sub-body codec.

**Architecture:** Data rides existing rails end to end. `atlas-channel` gains one classification-gated arm that decodes an empty sub-body and forwards the existing `REQUEST_ITEM_CONSUME` Kafka command. `atlas-consumables` gains a `ConsumeMorphCoupon` `ItemConsumer` that reads cash data, commits the Cash-compartment reservation, then issues an HP change and a `TemporaryStatTypeMorph` statup through the existing `atlas-buffs` pipeline. `atlas-data` gains two additive spec parses so `morph`/`hp` survive ingest. No new REST endpoints, Kafka topics, message bodies, DDL, or seed-template edits.

**Tech Stack:** Go 1.x, Go workspace (`go.work`), logrus, `atlas-model` provider/group combinators, `atlas-socket` request/response codecs, JSON:API via api2go, Kafka via `atlas-kafka`, testify (`assert`) in some packages and plain `t.Fatalf` in others — match the file you are editing.

---

## Global Constraints

These apply to **every** task below. A task that violates one is not done.

- **No region or major-version literal** in any of the three services' new code. The only version-dependent code touched is the pre-existing `cashsb.UpdateTimeFirst(t)` split. (PRD §8, FR-1.3)
- **No raw `530` numeric literal** outside `libs/atlas-constants`. Use `item.ClassificationTransformationCoupon` (channel) / `item2.ClassificationTransformationCoupon` (consumables). (PRD §10)
- **Buff duration is MILLISECONDS, unscaled.** The WZ `time` spec value (600000) is passed through with no `*1000` and no `/1000`. `tools/buff-duration-guard.sh` enforces this. (FR-3.6)
- **Cash compartment on every path.** `ConsumeMorphCoupon` uses `inventory2.TypeValueCash` for `ConsumeItem` *and* for every `ConsumeError`. Never `TypeValueUse`. (FR-3.3, FR-3.4, design §1.4)
- **No `session.EnableActions`.** The non-silent `INVENTORY_OPERATION` packet produced by the consume commit already clears the client's exclusive-request lock (design §1.2, IDA-verified). Failure paths send nothing, matching every neighbouring arm. (FR-4.3)
- **No seed-template edits.** `CharacterCashItemUseHandle` is already registered in ten of eleven templates. The four template guards are not implicated.
- **No `// TODO`, stubs, or 501s** in landed commits (CLAUDE.md).
- **No absolute home paths** (`/home/<user>/…`) in any committed file (CLAUDE.md).
- **Test setup uses the project Builder pattern or exported `Extract`.** No `*_testhelpers.go`, no test-only constructors. (CLAUDE.md)
- Run `tools/lint.sh` (fix mode, no flags) from the repo root before each commit, so formatting never shows up as review noise.

## Corrections to the PRD / design — read before starting

Both were checked against source during planning. Three claims need adjusting; the plan below already reflects the corrected versions.

1. **The cash-slot type-byte collision is CROSS-version, not same-version.** PRD FR-1.3 says "Pre-95, `ClassificationGachaponCoupon` also maps to 40". That is wrong. Read from `character_cash_item_use.go:896-901`, `:924-929`, `:937-942`:

   | Classification | GMS ≥ 95 | otherwise |
   |---|---|---|
   | 522 gachapon coupon | **40** | 39 |
   | 530 transformation coupon | 41 | **40** |
   | 538 pet evolution | 42 | **41** |

   So `40` means *transformation* pre-95 and *gachapon* on ≥95; `41` means *pet evolution* pre-95 and *transformation* on ≥95. Two different tenants, same byte. The classification gate is still exactly the right fix — and the collision is *worse* than the PRD described, because a type-byte-keyed arm would silently swap meaning across a version bump. Task 7's tests pin the corrected matrix (a v95 gachapon and a v83 pet-evolution must both miss the arm), which is a strictly stronger assertion than the PRD's bullet.

2. **No `it ==` arm in the handler uses 39, 40 or 41** (verified by grep: the only `it ==` comparisons are the named enum constants plus `viciousHammerCashSlotItemType(t)`, which returns 74/75). Placing the new arm in the classification-first block is therefore safe today; the placement is about organising principle, per design §3.2.

3. **`ConsumeMorphCoupon` cannot be unit-tested with the package mocks as the design sketched it.** Design §3.5's snippet constructs its five collaborators inline via `NewProcessor(l, ctx)`, and no test in `services/atlas-consumables/atlas.com/consumables/consumable/` injects a mock into any consumer — every existing test in that package exercises a *pure* function (`computeEffectPlan`, `collectCureTypes`, `usesStandardConsumer`, `selectMorph`, `buildScrollChanges`). But mocks satisfying all five interfaces already exist and are unused (`cash/mock`, `map/character/mock`, `character/mock`, `character/buff/mock`, `compartment/mock`). Task 5 therefore splits the consumer into a testable core taking an explicit deps struct plus a thin exported wrapper that binds the real processors — no package-level mutable seam, no test-only constructor, and PRD acceptance criteria FR-3.3/FR-3.4/FR-3.5/FR-3.6/FR-3.8 all become real assertions instead of prose.

**WZ fixture values re-verified during planning** (not carried from the design): `Item.wz/Cash/0530.img.xml` in two independent local corpora contains exactly `05300000`/`05300001`/`05300002`, each with `spec/hp = 50`, `spec/time = 600000`, `spec/morph = 1`/`2`/`3`, and **no** `morphRandom` node. Every fixture below uses these values.

---

## File Structure

| File | Responsibility | Task |
|---|---|---|
| `libs/atlas-packet/cash/serverbound/item_use_morph_coupon.go` (create) | The case-40/41 sub-body: trailing `updateTime` on GMS ≤ v84, nothing on GMS ≥ v87 / JMS | 1 |
| `libs/atlas-packet/cash/serverbound/item_use_morph_coupon_test.go` (create) | Round-trip both variants across every tenant variant | 1 |
| `services/atlas-data/atlas.com/data/cash/rest.go` (modify) | Two new `SpecType` constants | 2 |
| `services/atlas-data/atlas.com/data/cash/reader.go` (modify) | Parse `spec/morph`, `spec/hp` omit-when-zero | 2 |
| `services/atlas-data/atlas.com/data/cash/reader_test.go` (modify) | 0530 fixture (morph 1/2/3, hp 50, time 600000) + 0521 additive-only regression | 2 |
| `services/atlas-consumables/atlas.com/consumables/cash/rest.go` (modify) | Mirror three `SpecType` constants (`morph`, `hp`, `time`) | 3 |
| `services/atlas-consumables/atlas.com/consumables/cash/rest_test.go` (create) | JSON round-trip of the three keys through `RestModel` → `Extract` → `Model.GetSpec` | 3 |
| `services/atlas-consumables/atlas.com/consumables/consumable/morph_coupon.go` (create) | `morphCouponPlan`, `computeMorphCouponPlan`, `routesToMorphCoupon`, `morphCouponDeps`, `consumeMorphCoupon`, `ConsumeMorphCoupon` | 4, 5 |
| `services/atlas-consumables/atlas.com/consumables/consumable/morph_coupon_test.go` (create) | Planner table test, routing predicate test, consumer tests with the existing mocks | 4, 5, 6 |
| `services/atlas-consumables/atlas.com/consumables/consumable/processor.go` (modify) | One routing branch in `RequestItemConsume` | 6 |
| `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go` (modify) | One classification-gated arm + one test seam | 7 |
| `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_test.go` (modify) | Arm-entered / arm-not-entered tests over the corrected collision matrix | 7 |
| `docs/research/missing-features/items-and-consumables.md` (modify) | Retire "Wholly missing #7"; record the gms_12 no-op and the operational re-ingest follow-up | 8 |

A new file (`morph_coupon.go`) rather than appending to `processor.go`: that file is already 71 KB, and the package precedent for a self-contained item family is exactly this (`morph.go`, `skill_book.go`, `vega.go`, `reward.go`, `processor_catch.go`).

Tasks 1, 2, 3 are mutually independent. Task 4 → 5 → 6 are sequential. Task 7 depends on Task 1. Task 8 is last.

---

### Task 1: The `ItemUseMorphCoupon` sub-body codec

**Files:**
- Create: `libs/atlas-packet/cash/serverbound/item_use_morph_coupon.go`
- Test: `libs/atlas-packet/cash/serverbound/item_use_morph_coupon_test.go`

**Interfaces:**
- Consumes: `cashsb.UpdateTimeFirst(t tenant.Model) bool` from `item_use.go:21-23` — `(GMS && major >= 87) || JMS`.
- Produces: `NewItemUseMorphCoupon(updateTimeFirst bool) *ItemUseMorphCoupon`; methods `UpdateTime() uint32`, `Operation() string`, `String() string`, `Encode(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte`, `Decode(logrus.FieldLogger, context.Context) func(*request.Reader, map[string]interface{})`. Task 7 calls `NewItemUseMorphCoupon` and `UpdateTime`.

- [x] **Step 1: Write the failing round-trip test**

Create `libs/atlas-packet/cash/serverbound/item_use_morph_coupon_test.go`. This mirrors `item_use_pet_consumable_test.go` exactly — the same wire shape, verified independently for this arm.

```go
package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestItemUseMorphCouponUpdateTimeFirstRoundTrip pins FR-1.2's leading-updateTime
// half: when the common ItemUse header already carried updateTime (GMS >= v87,
// JMS), the case-41 sub-body reads NOTHING. Zero bytes on the wire.
func TestItemUseMorphCouponUpdateTimeFirstRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ItemUseMorphCoupon{updateTimeFirst: true}
			output := *NewItemUseMorphCoupon(true)
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.UpdateTime() != 0 {
				t.Errorf("updateTime: got %v, want 0 (nothing may be consumed when updateTimeFirst)", output.UpdateTime())
			}
		})
	}
}

// TestItemUseMorphCouponNoUpdateTimeFirstRoundTrip pins FR-1.2's trailing half:
// on GMS <= v84 the case-40 sub-body consumes exactly one trailing int32.
func TestItemUseMorphCouponNoUpdateTimeFirstRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ItemUseMorphCoupon{updateTime: 600000, updateTimeFirst: false}
			output := *NewItemUseMorphCoupon(false)
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.UpdateTime() != input.UpdateTime() {
				t.Errorf("updateTime: got %v, want %v", output.UpdateTime(), input.UpdateTime())
			}
		})
	}
}

// TestItemUseMorphCouponEncodedLength pins the byte count directly, so a future
// field added to the struct cannot pass the round-trip by symmetry alone.
func TestItemUseMorphCouponEncodedLength(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	if got := len(pt.Encode(t, ctx, ItemUseMorphCoupon{updateTime: 42, updateTimeFirst: false}.Encode)); got != 4 {
		t.Errorf("trailing-updateTime encoded length = %d, want 4", got)
	}
	if got := len(pt.Encode(t, ctx, ItemUseMorphCoupon{updateTime: 42, updateTimeFirst: true}.Encode)); got != 0 {
		t.Errorf("leading-updateTime encoded length = %d, want 0", got)
	}
}
```

Note: `pt.Encode(t, ctx, encodeFn)` is the helper in `libs/atlas-packet/test/` that runs an `Encode` closure under a tenant context with a null logger and returns the bytes. Confirm its exact name and signature by reading that package before writing this test; if it differs, adapt the call (the assertion is what matters) or drop the third test and keep the two round-trips.

- [x] **Step 2: Run the test to verify it fails**

```bash
cd libs/atlas-packet && go test ./cash/serverbound/ -run ItemUseMorphCoupon -v
```
Expected: FAIL — `undefined: ItemUseMorphCoupon`, `undefined: NewItemUseMorphCoupon`.

- [x] **Step 3: Write the codec**

Create `libs/atlas-packet/cash/serverbound/item_use_morph_coupon.go`:

```go
package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

// packet-audit:fname CWvsContext::SendConsumeCashItemUseRequest
//
// ItemUseMorphCoupon is the USE_CASH_ITEM sub-body for a transformation
// (morph) coupon, item classification 530 — cash-slot type 40 on GMS < 95 and
// 41 on GMS >= 95 (see GetCashSlotItemType in atlas-channel).
//
// The sub-body is EMPTY. IDA-verified on GMS v83 (MapleStory_dump.exe,
// CWvsContext::SendConsumeCashItemUseRequest @0xa0a63f): the case-40 arm spans
// 0xa0caf0-0xa0cb37 and contains no Encode* call at all. It runs three
// client-side predicates — the first, sub_A0ECCD @0xa0eccd, is literally
// `itemId / 10000 == 530` — then calls play_item_sound(nItemID, 0x29) @0xa0cb30
// and jumps to the shared send tail. The tail is what appends the trailing
// Encode4(get_update_time()) @0xa0ea5c on the versions that trail it.
//
// So the only thing this codec carries is that trailing updateTime, and only on
// the versions where the common ItemUse header did not already read it: GMS
// <= v84 trail, GMS v87+ and JMS lead (cashsb.UpdateTimeFirst). Byte-identical
// in behaviour to ItemUsePetConsumable, kept as its own type because the
// package convention is one struct per client arm and a future divergence on
// either arm must not force an unpick of the other.
type ItemUseMorphCoupon struct {
	updateTime      uint32
	updateTimeFirst bool
}

func NewItemUseMorphCoupon(updateTimeFirst bool) *ItemUseMorphCoupon {
	return &ItemUseMorphCoupon{updateTimeFirst: updateTimeFirst}
}

func (m ItemUseMorphCoupon) UpdateTime() uint32 { return m.updateTime }

func (m ItemUseMorphCoupon) Operation() string { return "ItemUseMorphCoupon" }

func (m ItemUseMorphCoupon) String() string {
	return fmt.Sprintf("updateTime [%d]", m.updateTime)
}

func (m ItemUseMorphCoupon) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		if !m.updateTimeFirst {
			w.WriteInt(m.updateTime)
		}
		return w.Bytes()
	}
}

func (m *ItemUseMorphCoupon) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		if !m.updateTimeFirst {
			m.updateTime = r.ReadUint32()
		}
	}
}
```

- [x] **Step 4: Run the test to verify it passes**

```bash
cd libs/atlas-packet && go test -race ./cash/serverbound/ -run ItemUseMorphCoupon -v
cd libs/atlas-packet && go vet ./... && go build ./...
```
Expected: PASS for every `pt.Variants` entry; vet and build clean.

- [x] **Step 5: Commit**

```bash
cd "$(git rev-parse --show-toplevel)" && tools/lint.sh
git add libs/atlas-packet/cash/serverbound/item_use_morph_coupon.go libs/atlas-packet/cash/serverbound/item_use_morph_coupon_test.go
git commit -m "feat(task-219): add ItemUseMorphCoupon serverbound sub-body codec"
```

---

### Task 2: `atlas-data` parses `spec/morph` and `spec/hp`

**Files:**
- Modify: `services/atlas-data/atlas.com/data/cash/rest.go:25` (add two constants after `SpecTypeTime`)
- Modify: `services/atlas-data/atlas.com/data/cash/reader.go:136-138` (add two parses after the `time` parse)
- Test: `services/atlas-data/atlas.com/data/cash/reader_test.go` (append)

**Interfaces:**
- Consumes: `xml.Node.GetIntegerWithDefault(name string, def int32) int32`; `Read(l) func(model.Provider[xml.Node]) model.Provider[[]RestModel]`; the test helpers already in `reader_test.go` — `Identity[M any]`, `model.CollectToMap[RestModel, string, RestModel](rms, RestModel.GetID, Identity)`, and `test.NewNullLogger()` from `logrus/hooks/test`.
- Produces: `cash.SpecTypeMorph = SpecType("morph")`, `cash.SpecTypeHp = SpecType("hp")`, both keys populated in `RestModel.Spec` when non-zero. Task 3 mirrors these string values verbatim.

- [x] **Step 1: Write the failing tests**

Append to `services/atlas-data/atlas.com/data/cash/reader_test.go`. The XML fixture is the real `Item.wz/Cash/0530.img.xml` shape, trimmed of canvas nodes (verified in two local corpora during planning: three items, `hp` 50, `time` 600000, `morph` 1/2/3, no `morphRandom`).

```go
// testMorphCouponXML mirrors Item.wz/Cash/0530.img.xml (transformation coupons,
// classification 530), trimmed of canvas nodes. Values verified against two
// independent local WZ corpora: every item carries spec/hp 50 and
// spec/time 600000, with spec/morph 1, 2, 3 respectively, and no morphRandom.
const testMorphCouponXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="0530.img">
  <imgdir name="05300000">
    <imgdir name="info">
      <int name="cash" value="1"/>
      <int name="price" value="100"/>
      <int name="slotMax" value="200"/>
      <int name="tradeBlock" value="1"/>
    </imgdir>
    <imgdir name="spec">
      <int name="hp" value="50"/>
      <int name="time" value="600000"/>
      <int name="morph" value="1"/>
    </imgdir>
  </imgdir>
  <imgdir name="05300001">
    <imgdir name="info">
      <int name="cash" value="1"/>
      <int name="price" value="100"/>
      <int name="slotMax" value="200"/>
      <int name="tradeBlock" value="1"/>
    </imgdir>
    <imgdir name="spec">
      <int name="hp" value="50"/>
      <int name="time" value="600000"/>
      <int name="morph" value="2"/>
    </imgdir>
  </imgdir>
  <imgdir name="05300002">
    <imgdir name="info">
      <int name="cash" value="1"/>
      <int name="price" value="100"/>
      <int name="slotMax" value="200"/>
      <int name="tradeBlock" value="1"/>
    </imgdir>
    <imgdir name="spec">
      <int name="hp" value="50"/>
      <int name="time" value="600000"/>
      <int name="morph" value="3"/>
    </imgdir>
  </imgdir>
</imgdir>`

// TestReaderMorphCoupons pins FR-2.3: all three 0530 items surface morph, hp and
// time. Before this task the reader dropped morph and hp entirely, so the coupon
// was inert no matter what the downstream services did.
func TestReaderMorphCoupons(t *testing.T) {
	l, _ := test.NewNullLogger()

	rms := Read(l)(xml.FromByteArrayProvider([]byte(testMorphCouponXML)))
	rmm, err := model.CollectToMap[RestModel, string, RestModel](rms, RestModel.GetID, Identity)()
	if err != nil {
		t.Fatal(err)
	}
	if len(rmm) != 3 {
		t.Fatalf("len(rmm) = %d, want 3", len(rmm))
	}

	for id, wantMorph := range map[int]int32{5300000: 1, 5300001: 2, 5300002: 3} {
		rm, ok := rmm[strconv.Itoa(id)]
		if !ok {
			t.Fatalf("rmm[%d] does not exist", id)
		}
		if rm.SlotMax != 200 {
			t.Errorf("[%d] SlotMax = %d, want 200", id, rm.SlotMax)
		}
		if !rm.TradeBlock {
			t.Errorf("[%d] TradeBlock = false, want true", id)
		}
		morph, ok := rm.Spec[SpecTypeMorph]
		if !ok {
			t.Fatalf("[%d] Spec[SpecTypeMorph] does not exist", id)
		}
		if morph != wantMorph {
			t.Errorf("[%d] Spec[SpecTypeMorph] = %d, want %d", id, morph, wantMorph)
		}
		hp, ok := rm.Spec[SpecTypeHp]
		if !ok {
			t.Fatalf("[%d] Spec[SpecTypeHp] does not exist", id)
		}
		if hp != 50 {
			t.Errorf("[%d] Spec[SpecTypeHp] = %d, want 50", id, hp)
		}
		specTime, ok := rm.Spec[SpecTypeTime]
		if !ok {
			t.Fatalf("[%d] Spec[SpecTypeTime] does not exist", id)
		}
		// 600000 is the raw WZ value in MILLISECONDS. atlas-buffs' duration
		// contract is milliseconds, so nothing on this path may rescale it.
		if specTime != 600000 {
			t.Errorf("[%d] Spec[SpecTypeTime] = %d, want 600000", id, specTime)
		}
	}
}

// TestReaderMorphHpAdditiveOnly pins FR-2.4: the two new keys are omit-when-zero,
// so a non-0530 cash item's parse output gains nothing. 5211000 is a 0521 EXP
// coupon, already covered end-to-end by TestReaderExpCoupons; this asserts the
// only thing that could have regressed — spurious keys.
func TestReaderMorphHpAdditiveOnly(t *testing.T) {
	l, _ := test.NewNullLogger()

	rms := Read(l)(xml.FromByteArrayProvider([]byte(testExpCouponXML)))
	rmm, err := model.CollectToMap[RestModel, string, RestModel](rms, RestModel.GetID, Identity)()
	if err != nil {
		t.Fatal(err)
	}

	for id := range rmm {
		rm := rmm[id]
		if v, ok := rm.Spec[SpecTypeMorph]; ok {
			t.Errorf("[%s] Spec[SpecTypeMorph] = %d, want absent on a 0521 EXP coupon", id, v)
		}
		if v, ok := rm.Spec[SpecTypeHp]; ok {
			t.Errorf("[%s] Spec[SpecTypeHp] = %d, want absent on a 0521 EXP coupon", id, v)
		}
	}
}
```

- [x] **Step 2: Run the tests to verify they fail**

```bash
cd services/atlas-data/atlas.com/data && go test ./cash/ -run 'TestReaderMorphCoupons|TestReaderMorphHpAdditiveOnly' -v
```
Expected: FAIL — `undefined: SpecTypeMorph`, `undefined: SpecTypeHp`.

- [x] **Step 3: Add the two constants**

In `services/atlas-data/atlas.com/data/cash/rest.go`, replace the `SpecTypeTime` line inside the `const` block with:

```go
	SpecTypeTime = SpecType("time") // Duration from spec node; raw WZ units (0530 morph coupons: milliseconds)
	// Transformation-coupon properties (0530.img): the Morph.wz creature id and
	// the flat HP heal. Both are omit-when-zero in the reader, so downstream
	// "absent or zero" collapses to a single `ok && val > 0` test.
	SpecTypeMorph = SpecType("morph")
	SpecTypeHp    = SpecType("hp")
```

The `SpecTypeTime` comment is corrected in the same edit: it currently reads "Duration in minutes from spec node", which is wrong for 0530 (600000 = ten minutes in milliseconds) and would mislead the next reader into scaling it. The reader passes the raw WZ value through either way, so this is a comment-only fix with no behaviour change.

- [x] **Step 4: Add the two parses**

In `services/atlas-data/atlas.com/data/cash/reader.go`, immediately after the existing `time` parse (currently lines 136-138), inside the same `if err == nil && s != nil` block:

```go
			// Transformation coupons (0530.img): the Morph.wz creature id and the
			// flat HP heal. Omit-when-zero, matching expR/drpR/time above.
			if morph := s.GetIntegerWithDefault(string(SpecTypeMorph), 0); morph != 0 {
				m.Spec[SpecTypeMorph] = morph
			}
			if hp := s.GetIntegerWithDefault(string(SpecTypeHp), 0); hp != 0 {
				m.Spec[SpecTypeHp] = hp
			}
```

- [x] **Step 5: Run the tests to verify they pass**

```bash
cd services/atlas-data/atlas.com/data && go test -race ./cash/ -v
cd services/atlas-data/atlas.com/data && go vet ./... && go build ./...
```
Expected: the two new tests PASS; every pre-existing cash test (`TestReader`, `TestReaderExpCoupons`, `TestReaderDropCoupons`, `TestReaderPetSkills`, `TestReaderProtectTime`, the resource tests) still PASSES.

- [x] **Step 6: Commit**

```bash
cd "$(git rev-parse --show-toplevel)" && tools/lint.sh
git add services/atlas-data/atlas.com/data/cash/rest.go services/atlas-data/atlas.com/data/cash/reader.go services/atlas-data/atlas.com/data/cash/reader_test.go
git commit -m "feat(task-219): parse cash spec/morph and spec/hp in atlas-data"
```

---

### Task 3: `atlas-consumables` cash REST model mirrors the spec keys

**Files:**
- Modify: `services/atlas-consumables/atlas.com/consumables/cash/rest.go:9-21` (add three constants)
- Test: `services/atlas-consumables/atlas.com/consumables/cash/rest_test.go` (create)

**Interfaces:**
- Consumes: the string values fixed in Task 2 — `"morph"`, `"hp"`, `"time"`. These are wire-format keys in the JSON `spec` object; a typo here silently yields a zero-valued spec at runtime, which is exactly the failure mode this task's test exists to prevent.
- Produces: `cash.SpecTypeMorph`, `cash.SpecTypeHp`, `cash.SpecTypeTime`; readable via the existing `cash.Model.GetSpec(SpecType) (int32, bool)`. Tasks 4 and 5 consume all three.

**Why three, not two:** PRD FR-3.1 names only `morph` and `hp`, but `atlas-consumables`' cash `SpecType` set has no `SpecTypeTime` at all (unlike atlas-data's), and FR-3.6 requires reading the duration from `time`. Adding `SpecTypeTime` is a strict superset of FR-3.1, called out here so it is not read as scope creep. The atlas-data-only keys `rate`, `expR`, `drpR` are deliberately **not** mirrored — nothing on this path consumes them.

- [x] **Step 1: Write the failing test**

Create `services/atlas-consumables/atlas.com/consumables/cash/rest_test.go`:

```go
package cash

import (
	"encoding/json"
	"testing"
)

// TestRestModelSpecRoundTripsMorphKeys pins FR-3.1: the three spec keys a
// transformation coupon needs survive the atlas-data REST hop. The literal JSON
// below is the shape atlas-data emits for 5300000 (PRD §5); if a constant here
// drifts from atlas-data's string value the spec silently reads as zero at
// runtime and the coupon becomes an inert consume, so this test asserts the
// wire strings, not just the Go identifiers.
func TestRestModelSpecRoundTripsMorphKeys(t *testing.T) {
	const body = `{"slotMax":200,"spec":{"morph":1,"hp":50,"time":600000}}`

	var rm RestModel
	if err := json.Unmarshal([]byte(body), &rm); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}

	for _, tc := range []struct {
		spec SpecType
		want int32
	}{
		{SpecTypeMorph, 1},
		{SpecTypeHp, 50},
		{SpecTypeTime, 600000},
	} {
		got, ok := m.GetSpec(tc.spec)
		if !ok {
			t.Fatalf("GetSpec(%q) missing", tc.spec)
		}
		if got != tc.want {
			t.Errorf("GetSpec(%q) = %d, want %d", tc.spec, got, tc.want)
		}
	}

	if m.slotMax != 200 {
		t.Errorf("slotMax = %d, want 200", m.slotMax)
	}
}

// TestSpecTypeWireValues pins the exact JSON keys against atlas-data's
// (services/atlas-data/atlas.com/data/cash/rest.go). These two SpecType sets
// live in separate Go modules, so a rename in one and not the other fails no
// build — it decodes into a zero-valued spec, silently.
func TestSpecTypeWireValues(t *testing.T) {
	for _, tc := range []struct {
		spec SpecType
		want string
	}{
		{SpecTypeMorph, "morph"},
		{SpecTypeHp, "hp"},
		{SpecTypeTime, "time"},
	} {
		if string(tc.spec) != tc.want {
			t.Errorf("SpecType = %q, want %q", tc.spec, tc.want)
		}
	}
}

// TestGetSpecAbsentKey pins the negative half FR-3.7 depends on: an absent key
// reports ok=false rather than a zero value indistinguishable from a real zero.
func TestGetSpecAbsentKey(t *testing.T) {
	m, err := Extract(RestModel{Spec: map[SpecType]int32{SpecTypeTime: 600000}})
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if v, ok := m.GetSpec(SpecTypeMorph); ok {
		t.Errorf("GetSpec(morph) = (%d, true), want ok=false when the key is absent", v)
	}
}
```

- [x] **Step 2: Run the test to verify it fails**

```bash
cd services/atlas-consumables/atlas.com/consumables && go test ./cash/ -v
```
Expected: FAIL — `undefined: SpecTypeMorph`, `undefined: SpecTypeHp`, `undefined: SpecTypeTime`.

- [x] **Step 3: Add the three constants**

In `services/atlas-consumables/atlas.com/consumables/cash/rest.go`, inside the existing `const` block, after `SpecTypeIndexNine`:

```go
	// Transformation-coupon properties (0530.img), mirroring atlas-data's
	// cash SpecType set (services/atlas-data/atlas.com/data/cash/rest.go).
	// `time` is the buff duration in MILLISECONDS, the unit atlas-buffs
	// expects — nothing on this path may rescale it.
	SpecTypeMorph = SpecType("morph")
	SpecTypeHp    = SpecType("hp")
	SpecTypeTime  = SpecType("time")
```

- [x] **Step 4: Run the test to verify it passes**

```bash
cd services/atlas-consumables/atlas.com/consumables && go test -race ./cash/ -v
```
Expected: PASS (all three tests).

- [x] **Step 5: Commit**

```bash
cd "$(git rev-parse --show-toplevel)" && tools/lint.sh
git add services/atlas-consumables/atlas.com/consumables/cash/rest.go services/atlas-consumables/atlas.com/consumables/cash/rest_test.go
git commit -m "feat(task-219): mirror cash morph/hp/time spec keys in atlas-consumables"
```

---

### Task 4: The pure planner and the routing predicate

**Files:**
- Create: `services/atlas-consumables/atlas.com/consumables/consumable/morph_coupon.go`
- Test: `services/atlas-consumables/atlas.com/consumables/consumable/morph_coupon_test.go`

**Interfaces:**
- Consumes: `cash.Model.GetSpec(cash.SpecType) (int32, bool)`, `cash.SpecTypeMorph`/`SpecTypeHp`/`SpecTypeTime` (Task 3); `cash.Extract(cash.RestModel) (cash.Model, error)` for test fixtures; `stat.Model{Type ts.TemporaryStatType; Amount int32}`; `ts.TemporaryStatTypeMorph`; `item2.GetClassification(item2.Id) item2.Classification`; `item2.ClassificationTransformationCoupon`.
- Produces:
  - `type morphCouponPlan struct { hp int16; statups []stat.Model; duration int32 }`
  - `func computeMorphCouponPlan(ci cash.Model) morphCouponPlan`
  - `func routesToMorphCoupon(itemId item2.Id) bool`

  Task 5 calls `computeMorphCouponPlan`; Task 6 calls `routesToMorphCoupon`.

- [x] **Step 1: Write the failing tests**

Create `services/atlas-consumables/atlas.com/consumables/consumable/morph_coupon_test.go`:

```go
package consumable

import (
	"testing"

	"atlas-consumables/cash"
	"atlas-consumables/character/buff/stat"

	ts "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	item2 "github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

// extractCash builds a cash.Model through the package's own exported Extract,
// so no test-only constructor is introduced (CLAUDE.md test-helper rule).
func extractCash(t *testing.T, spec map[cash.SpecType]int32) cash.Model {
	t.Helper()
	m, err := cash.Extract(cash.RestModel{Id: 5300000, SlotMax: 200, Spec: spec})
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	return m
}

// TestComputeMorphCouponPlan covers FR-3.5 and every FR-3.7 permutation. The
// full-spec row uses the real WZ values for 5300000 (morph 1, hp 50,
// time 600000 ms), verified against two local Item.wz/Cash/0530.img.xml copies.
func TestComputeMorphCouponPlan(t *testing.T) {
	tests := []struct {
		name         string
		spec         map[cash.SpecType]int32
		wantHp       int16
		wantMorph    int32 // 0 = expect no morph statup
		wantDuration int32
	}{
		{
			name:         "full spec (5300000)",
			spec:         map[cash.SpecType]int32{cash.SpecTypeMorph: 1, cash.SpecTypeHp: 50, cash.SpecTypeTime: 600000},
			wantHp:       50,
			wantMorph:    1,
			wantDuration: 600000,
		},
		{
			name:         "morph 3 (5300002)",
			spec:         map[cash.SpecType]int32{cash.SpecTypeMorph: 3, cash.SpecTypeHp: 50, cash.SpecTypeTime: 600000},
			wantHp:       50,
			wantMorph:    3,
			wantDuration: 600000,
		},
		{
			name:         "morph absent: heals, does not morph",
			spec:         map[cash.SpecType]int32{cash.SpecTypeHp: 50, cash.SpecTypeTime: 600000},
			wantHp:       50,
			wantMorph:    0,
			wantDuration: 600000,
		},
		{
			name:         "morph zero: heals, does not morph",
			spec:         map[cash.SpecType]int32{cash.SpecTypeMorph: 0, cash.SpecTypeHp: 50, cash.SpecTypeTime: 600000},
			wantHp:       50,
			wantMorph:    0,
			wantDuration: 600000,
		},
		{
			name:         "hp absent: morphs, does not heal",
			spec:         map[cash.SpecType]int32{cash.SpecTypeMorph: 2, cash.SpecTypeTime: 600000},
			wantHp:       0,
			wantMorph:    2,
			wantDuration: 600000,
		},
		{
			name:         "hp zero: morphs, does not heal",
			spec:         map[cash.SpecType]int32{cash.SpecTypeMorph: 2, cash.SpecTypeHp: 0, cash.SpecTypeTime: 600000},
			wantHp:       0,
			wantMorph:    2,
			wantDuration: 600000,
		},
		{
			name:         "both absent: does nothing (stale ingest)",
			spec:         map[cash.SpecType]int32{cash.SpecTypeTime: 600000},
			wantHp:       0,
			wantMorph:    0,
			wantDuration: 600000,
		},
		{
			name:         "empty spec: does nothing, duration zero",
			spec:         map[cash.SpecType]int32{},
			wantHp:       0,
			wantMorph:    0,
			wantDuration: 0,
		},
		{
			name:         "time absent: morph applied with zero duration",
			spec:         map[cash.SpecType]int32{cash.SpecTypeMorph: 1, cash.SpecTypeHp: 50},
			wantHp:       50,
			wantMorph:    1,
			wantDuration: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			plan := computeMorphCouponPlan(extractCash(t, tc.spec))

			if plan.hp != tc.wantHp {
				t.Errorf("hp = %d, want %d", plan.hp, tc.wantHp)
			}
			// FR-3.6: the WZ `time` value is passed through unscaled — atlas-buffs
			// expects milliseconds. Any *1000 or /1000 fails here.
			if plan.duration != tc.wantDuration {
				t.Errorf("duration = %d, want %d (raw WZ ms, unscaled)", plan.duration, tc.wantDuration)
			}
			if tc.wantMorph == 0 {
				if len(plan.statups) != 0 {
					t.Fatalf("statups = %+v, want none", plan.statups)
				}
				return
			}
			if len(plan.statups) != 1 {
				t.Fatalf("len(statups) = %d, want 1", len(plan.statups))
			}
			want := stat.Model{Type: ts.TemporaryStatTypeMorph, Amount: tc.wantMorph}
			if plan.statups[0] != want {
				t.Errorf("statups[0] = %+v, want %+v", plan.statups[0], want)
			}
		})
	}
}

// TestRoutesToMorphCoupon pins FR-1.3 / FR-3.2's gate: selection is by item
// classification, never by the cash-slot type byte. The negatives are the
// classifications whose type bytes collide with 530's across versions
// (gachapon 522 -> 40 on GMS>=95; pet evolution 538 -> 41 on GMS<95) plus the
// use-tab transformation potion (221), which must keep routing to
// ConsumeStandard.
func TestRoutesToMorphCoupon(t *testing.T) {
	for _, id := range []item2.Id{5300000, 5300001, 5300002} {
		if item2.GetClassification(id) != item2.ClassificationTransformationCoupon {
			t.Fatalf("fixture invalid: GetClassification(%d) = %d, want 530", id, item2.GetClassification(id))
		}
		if !routesToMorphCoupon(id) {
			t.Errorf("routesToMorphCoupon(%d) = false, want true", id)
		}
	}
	for _, id := range []item2.Id{
		5220000, // 522 gachapon coupon  -> cash-slot type 40 on GMS >= 95
		5380000, // 538 pet evolution    -> cash-slot type 41 on GMS <  95
		5211000, // 521 EXP coupon
		2210000, // 221 use-tab transformation potion -> ConsumeStandard
		2000000, // 200 HP potion
	} {
		if routesToMorphCoupon(id) {
			t.Errorf("routesToMorphCoupon(%d) = true, want false (classification %d)", id, item2.GetClassification(id))
		}
	}
}

// TestMorphCouponNotStandardConsumer pins FR-3.2's negative half: ConsumeStandard
// hard-codes inventory2.TypeValueUse and fetches from the *consumable* data
// resource, where 5300000 does not exist. A 530 item must never reach it.
func TestMorphCouponNotStandardConsumer(t *testing.T) {
	for _, id := range []item2.Id{5300000, 5300001, 5300002} {
		if usesStandardConsumer(id) {
			t.Errorf("usesStandardConsumer(%d) = true, want false", id)
		}
	}
}
```

- [x] **Step 2: Run the tests to verify they fail**

```bash
cd services/atlas-consumables/atlas.com/consumables && go test ./consumable/ -run 'MorphCoupon' -v
```
Expected: FAIL — `undefined: computeMorphCouponPlan`, `undefined: routesToMorphCoupon`.

- [x] **Step 3: Write the planner and predicate**

Create `services/atlas-consumables/atlas.com/consumables/consumable/morph_coupon.go`:

```go
package consumable

import (
	"atlas-consumables/cash"
	"atlas-consumables/character/buff/stat"

	ts "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	item2 "github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

// morphCouponPlan is the pure result of interpreting a transformation coupon's
// specs: everything the consumer will do, decided before any side effect.
// Mirrors effectPlan's rationale (processor.go) — keeping the decision pure is
// what makes the morph/hp permutations pinnable by plain unit tests.
type morphCouponPlan struct {
	hp       int16        // 0 = no HP change
	statups  []stat.Model // at most one: MORPH
	duration int32        // raw WZ `time` spec in MILLISECONDS, unscaled
}

// computeMorphCouponPlan interprets a 0530 cash item's specs with no side
// effects.
//
// morph and hp are independent (FR-3.7): a coupon whose morph is absent or zero
// still heals and still consumes, and vice versa. A coupon with neither — the
// shape served by a tenant whose cash WZ was ingested before this feature landed
// — consumes and does nothing, which is the honest outcome for absent data.
//
// morphRandom is deliberately not consulted: no item in Item.wz/Cash/0530.img.xml
// carries one in any inspected corpus, so the weighted selector in morph.go stays
// unwired on this path.
func computeMorphCouponPlan(ci cash.Model) morphCouponPlan {
	plan := morphCouponPlan{statups: make([]stat.Model, 0, 1)}

	if val, ok := ci.GetSpec(cash.SpecTypeHp); ok && val > 0 {
		plan.hp = int16(val)
	}
	if val, ok := ci.GetSpec(cash.SpecTypeMorph); ok && val > 0 {
		plan.statups = append(plan.statups, stat.Model{Type: ts.TemporaryStatTypeMorph, Amount: val})
	}
	if val, ok := ci.GetSpec(cash.SpecTypeTime); ok && val > 0 {
		// Milliseconds, passed through as-is. atlas-buffs computes expiry as
		// now + duration*time.Millisecond (ApplyCommandBody.Duration), so any
		// seconds<->milliseconds scaling here makes the morph expire ~1000x
		// early or late. tools/buff-duration-guard.sh enforces this.
		plan.duration = val
	}

	return plan
}

// routesToMorphCoupon reports whether itemId is a transformation coupon and
// therefore routes to ConsumeMorphCoupon.
//
// The gate is item CLASSIFICATION, never the cash-slot type byte. Those bytes
// collide across client versions: atlas-channel's GetCashSlotItemType maps
// classification 530 to type 41 on GMS >= 95 and 40 otherwise, while gachapon
// coupons (522) take 40 on GMS >= 95 and pet evolution (538) takes 41 on
// GMS < 95. A type-keyed gate would silently change meaning at a version bump.
func routesToMorphCoupon(itemId item2.Id) bool {
	return item2.GetClassification(itemId) == item2.ClassificationTransformationCoupon
}
```

- [x] **Step 4: Run the tests to verify they pass**

```bash
cd services/atlas-consumables/atlas.com/consumables && go test -race ./consumable/ -run 'MorphCoupon' -v
```
Expected: PASS — every table row, both predicate tests.

- [x] **Step 5: Commit**

```bash
cd "$(git rev-parse --show-toplevel)" && tools/lint.sh
git add services/atlas-consumables/atlas.com/consumables/consumable/morph_coupon.go services/atlas-consumables/atlas.com/consumables/consumable/morph_coupon_test.go
git commit -m "feat(task-219): add pure morph-coupon effect planner and routing predicate"
```

---

### Task 5: The `ConsumeMorphCoupon` consumer

**Files:**
- Modify: `services/atlas-consumables/atlas.com/consumables/consumable/morph_coupon.go` (append)
- Modify: `services/atlas-consumables/atlas.com/consumables/consumable/morph_coupon_test.go` (append)

**Interfaces:**
- Consumes: `computeMorphCouponPlan` (Task 4); `ItemConsumer func(l logrus.FieldLogger) func(ctx context.Context) error` (`processor.go:66`); `NewProcessor(l, ctx).ConsumeError(characterId uint32, transactionId uuid.UUID, inventoryType inventory2.Type, slot int16, err error) error` (`processor.go:373`); the five collaborator interfaces —
  - `cash.Processor`: `GetById(itemId uint32) (cash.Model, error)`
  - `character2.Processor` (`atlas-consumables/map/character`): `GetMap(characterId uint32) (field.Model, error)`
  - `compartment.Processor`: `ConsumeItem(characterId uint32, inventoryType inventory2.Type, transactionId uuid.UUID, slot int16) error`
  - `character.Processor`: `ChangeHP(f field.Model, characterId uint32, amount int16) error`
  - `buff.Processor`: `Apply(f field.Model, fromId uint32, sourceId int32, level byte, duration int32, statups []stat.Model) model.Operator[uint32]`

  and the matching mocks `cash/mock`, `map/character/mock`, `compartment/mock`, `character/mock`, `character/buff/mock` — each a `ProcessorMock` with `<Method>Func` fields.
- Produces: `func ConsumeMorphCoupon(transactionId uuid.UUID, characterId uint32, slot int16, itemId item2.Id) ItemConsumer` — the exact argument order Task 6's routing branch uses, matching `ConsumeCashPetFood`'s.

**Design note (correction 3 above):** the consumer is split in two. `consumeMorphCoupon` takes a `morphCouponDeps` struct and holds all the logic; the exported `ConsumeMorphCoupon` binds the real processors and calls it. `morphCouponDeps` is production code used by the production wrapper — not a test hook — and it is what makes FR-3.3/FR-3.4/FR-3.6/FR-3.8 real assertions rather than prose. No package-level mutable seam is introduced.

- [x] **Step 1: Write the failing tests**

Append to `services/atlas-consumables/atlas.com/consumables/consumable/morph_coupon_test.go`. Add these imports to the existing import block:

```go
	"context"
	"errors"

	buffmock "atlas-consumables/character/buff/mock"
	charmock "atlas-consumables/character/mock"
	cashmock "atlas-consumables/cash/mock"
	compmock "atlas-consumables/compartment/mock"
	mapcharmock "atlas-consumables/map/character/mock"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	fieldc "github.com/Chronicle20/atlas/libs/atlas-constants/field"
	inventory2 "github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	_map2 "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)
```

Then the tests:

```go
// consumeItemCall records one compartment.ConsumeItem invocation so the
// compartment-type contract (FR-3.4) can be asserted, not assumed.
type consumeItemCall struct {
	characterId   uint32
	inventoryType inventory2.Type
	slot          int16
}

type applyCall struct {
	fromId   uint32
	sourceId int32
	level    byte
	duration int32
	statups  []stat.Model
}

type changeHPCall struct {
	characterId uint32
	amount      int16
}

// morphCouponHarness wires the five package mocks into a morphCouponDeps and
// captures every outbound call. onError records the ConsumeError route without
// needing a live Kafka broker.
type morphCouponHarness struct {
	deps         morphCouponDeps
	consumeItems []consumeItemCall
	applies      []applyCall
	hpChanges    []changeHPCall
	errors       []error
}

func newMorphCouponHarness(t *testing.T, ci cash.Model, cashErr error) *morphCouponHarness {
	t.Helper()
	f := fieldc.NewBuilder(world.Id(0), channel.Id(0), _map2.Id(100000000)).Build()
	h := &morphCouponHarness{}
	h.deps = morphCouponDeps{
		cash: &cashmock.ProcessorMock{
			GetByIdFunc: func(uint32) (cash.Model, error) { return ci, cashErr },
		},
		fields: &mapcharmock.ProcessorMock{
			GetMapFunc: func(uint32) (fieldc.Model, error) { return f, nil },
		},
		compartment: &compmock.ProcessorMock{
			ConsumeItemFunc: func(characterId uint32, it inventory2.Type, _ uuid.UUID, slot int16) error {
				h.consumeItems = append(h.consumeItems, consumeItemCall{characterId, it, slot})
				return nil
			},
		},
		character: &charmock.ProcessorMock{
			ChangeHPFunc: func(_ fieldc.Model, characterId uint32, amount int16) error {
				h.hpChanges = append(h.hpChanges, changeHPCall{characterId, amount})
				return nil
			},
		},
		buff: &buffmock.ProcessorMock{
			ApplyFunc: func(_ fieldc.Model, fromId uint32, sourceId int32, level byte, duration int32, statups []stat.Model) model.Operator[uint32] {
				h.applies = append(h.applies, applyCall{fromId, sourceId, level, duration, statups})
				return func(uint32) error { return nil }
			},
		},
		onError: func(err error) error {
			h.errors = append(h.errors, err)
			return err
		},
	}
	return h
}

func fullMorphSpec() map[cash.SpecType]int32 {
	return map[cash.SpecType]int32{cash.SpecTypeMorph: 1, cash.SpecTypeHp: 50, cash.SpecTypeTime: 600000}
}

// TestConsumeMorphCouponSuccess pins FR-3.4, FR-3.5 and FR-3.6 in one pass:
// the coupon leaves the CASH compartment, the HP heal is issued, and the morph
// statup carries source = -itemId, level 0, and the raw WZ duration in ms.
func TestConsumeMorphCouponSuccess(t *testing.T) {
	const characterId = uint32(555)
	const slot = int16(3)
	const itemId = item2.Id(5300000)

	h := newMorphCouponHarness(t, extractCash(t, fullMorphSpec()), nil)

	if err := consumeMorphCoupon(logrus.New(), context.Background(), h.deps, uuid.New(), characterId, slot, itemId); err != nil {
		t.Fatalf("consumeMorphCoupon: %v", err)
	}

	if len(h.errors) != 0 {
		t.Fatalf("errors = %v, want none", h.errors)
	}
	if len(h.consumeItems) != 1 {
		t.Fatalf("ConsumeItem call count = %d, want 1", len(h.consumeItems))
	}
	// FR-3.4: the CASH compartment, not the Use compartment. ConsumeStandard's
	// hard-coded TypeValueUse is exactly why this consumer exists.
	if got := h.consumeItems[0].inventoryType; got != inventory2.TypeValueCash {
		t.Errorf("ConsumeItem inventoryType = %d, want TypeValueCash (%d)", got, inventory2.TypeValueCash)
	}
	if h.consumeItems[0].characterId != characterId || h.consumeItems[0].slot != slot {
		t.Errorf("ConsumeItem = %+v, want characterId %d slot %d", h.consumeItems[0], characterId, slot)
	}

	if len(h.hpChanges) != 1 || h.hpChanges[0].amount != 50 {
		t.Fatalf("hpChanges = %+v, want one call of amount 50", h.hpChanges)
	}

	if len(h.applies) != 1 {
		t.Fatalf("Apply call count = %d, want 1", len(h.applies))
	}
	a := h.applies[0]
	if a.sourceId != -int32(itemId) {
		t.Errorf("Apply sourceId = %d, want %d (-itemId)", a.sourceId, -int32(itemId))
	}
	if a.level != 0 {
		t.Errorf("Apply level = %d, want 0", a.level)
	}
	// FR-3.6: 600000 is the WZ value in milliseconds, unscaled.
	if a.duration != 600000 {
		t.Errorf("Apply duration = %d, want 600000 (raw WZ ms)", a.duration)
	}
	want := stat.Model{Type: ts.TemporaryStatTypeMorph, Amount: 1}
	if len(a.statups) != 1 || a.statups[0] != want {
		t.Errorf("Apply statups = %+v, want [%+v]", a.statups, want)
	}
	if a.fromId != characterId {
		t.Errorf("Apply fromId = %d, want %d", a.fromId, characterId)
	}
}

// TestConsumeMorphCouponCashFetchFailureKeepsCoupon pins FR-3.3: every fallible
// read happens BEFORE the commit, so a data failure releases the reservation via
// ConsumeError and the player keeps the paid item.
func TestConsumeMorphCouponCashFetchFailureKeepsCoupon(t *testing.T) {
	wantErr := errors.New("cash-items 404")
	h := newMorphCouponHarness(t, cash.Model{}, wantErr)

	err := consumeMorphCoupon(logrus.New(), context.Background(), h.deps, uuid.New(), 555, 3, 5300000)
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}
	if len(h.errors) != 1 {
		t.Fatalf("onError call count = %d, want 1", len(h.errors))
	}
	if len(h.consumeItems) != 0 {
		t.Errorf("ConsumeItem call count = %d, want 0 — the coupon must stay in the inventory", len(h.consumeItems))
	}
	if len(h.applies) != 0 || len(h.hpChanges) != 0 {
		t.Errorf("effects applied on a failed read: applies %+v, hp %+v", h.applies, h.hpChanges)
	}
}

// TestConsumeMorphCouponConsumeFailureAppliesNoEffects: a failed commit must not
// morph or heal — the reservation is released and nothing else happens.
func TestConsumeMorphCouponConsumeFailureAppliesNoEffects(t *testing.T) {
	wantErr := errors.New("reservation expired")
	h := newMorphCouponHarness(t, extractCash(t, fullMorphSpec()), nil)
	h.deps.compartment = &compmock.ProcessorMock{
		ConsumeItemFunc: func(uint32, inventory2.Type, uuid.UUID, int16) error { return wantErr },
	}

	err := consumeMorphCoupon(logrus.New(), context.Background(), h.deps, uuid.New(), 555, 3, 5300000)
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}
	if len(h.applies) != 0 || len(h.hpChanges) != 0 {
		t.Errorf("effects applied after a failed commit: applies %+v, hp %+v", h.applies, h.hpChanges)
	}
}

// TestConsumeMorphCouponZeroSpecs pins FR-3.7 through the consumer, not just the
// planner: each half is independently skippable and neither blocks the consume.
func TestConsumeMorphCouponZeroSpecs(t *testing.T) {
	tests := []struct {
		name        string
		spec        map[cash.SpecType]int32
		wantApplies int
		wantHp      int
	}{
		{"morph absent", map[cash.SpecType]int32{cash.SpecTypeHp: 50, cash.SpecTypeTime: 600000}, 0, 1},
		{"hp absent", map[cash.SpecType]int32{cash.SpecTypeMorph: 1, cash.SpecTypeTime: 600000}, 1, 0},
		{"both absent (stale ingest)", map[cash.SpecType]int32{cash.SpecTypeTime: 600000}, 0, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newMorphCouponHarness(t, extractCash(t, tc.spec), nil)
			if err := consumeMorphCoupon(logrus.New(), context.Background(), h.deps, uuid.New(), 555, 3, 5300000); err != nil {
				t.Fatalf("consumeMorphCoupon: %v", err)
			}
			// The coupon is consumed in every row — an empty spec is not an error.
			if len(h.consumeItems) != 1 {
				t.Errorf("ConsumeItem call count = %d, want 1", len(h.consumeItems))
			}
			if len(h.applies) != tc.wantApplies {
				t.Errorf("Apply call count = %d, want %d", len(h.applies), tc.wantApplies)
			}
			if len(h.hpChanges) != tc.wantHp {
				t.Errorf("ChangeHP call count = %d, want %d", len(h.hpChanges), tc.wantHp)
			}
			if len(h.errors) != 0 {
				t.Errorf("errors = %v, want none — an absent spec is not a failure", h.errors)
			}
		})
	}
}

// TestConsumeMorphCouponReuseWhileMorphedApplies pins FR-3.8: using a second
// coupon while already transformed issues a second Apply unconditionally. There
// is no "already morphed" rejection branch — replace-and-restart is the default
// overwrite behaviour of the atlas-buffs apply path. This test asserts the
// ABSENCE of a rejection, so adding one would fail here.
func TestConsumeMorphCouponReuseWhileMorphedApplies(t *testing.T) {
	h := newMorphCouponHarness(t, extractCash(t, fullMorphSpec()), nil)
	l := logrus.New()

	for i := 0; i < 2; i++ {
		if err := consumeMorphCoupon(l, context.Background(), h.deps, uuid.New(), 555, 3, 5300000); err != nil {
			t.Fatalf("use %d: %v", i+1, err)
		}
	}

	if len(h.applies) != 2 {
		t.Fatalf("Apply call count = %d, want 2 (second use replaces the morph and restarts the timer)", len(h.applies))
	}
	if h.applies[0] != h.applies[1] {
		t.Errorf("second Apply differs from the first: %+v vs %+v", h.applies[0], h.applies[1])
	}
	if len(h.consumeItems) != 2 {
		t.Errorf("ConsumeItem call count = %d, want 2", len(h.consumeItems))
	}
}

// TestConsumeMorphCouponBindsRealProcessors: the exported wrapper must build a
// complete deps set. A nil collaborator would panic at runtime on the first
// coupon use, which no other test in this file would catch.
func TestConsumeMorphCouponBindsRealProcessors(t *testing.T) {
	if ConsumeMorphCoupon(uuid.New(), 555, 3, 5300000) == nil {
		t.Fatal("ConsumeMorphCoupon returned a nil ItemConsumer")
	}
}
```

If a mock import path or `ProcessorMock` field name differs from the above, read the mock package and adapt the wiring — the assertions are what matter. All five were confirmed to exist during planning with these exact `<Method>Func` field names.

- [x] **Step 2: Run the tests to verify they fail**

```bash
cd services/atlas-consumables/atlas.com/consumables && go test ./consumable/ -run 'ConsumeMorphCoupon' -v
```
Expected: FAIL — `undefined: morphCouponDeps`, `undefined: consumeMorphCoupon`, `undefined: ConsumeMorphCoupon`.

- [x] **Step 3: Write the consumer**

Append to `services/atlas-consumables/atlas.com/consumables/consumable/morph_coupon.go`, and extend its import block with:

```go
	"context"

	"atlas-consumables/character"
	"atlas-consumables/character/buff"
	"atlas-consumables/compartment"
	character2 "atlas-consumables/map/character"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	inventory2 "github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)
```

```go
// morphCouponDeps is the collaborator set consumeMorphCoupon needs.
//
// Split out from the exported consumer so the ordering contract — every fallible
// read BEFORE the commit (FR-3.3), the CASH compartment on every path (FR-3.4),
// the unscaled millisecond duration (FR-3.6) — is pinnable with the package's
// existing mocks. ConsumeMorphCoupon below binds the real processors.
type morphCouponDeps struct {
	cash        cash.Processor
	fields      character2.Processor
	compartment compartment.Processor
	character   character.Processor
	buff        buff.Processor
	// onError releases the reservation and notifies the client. Bound to
	// ProcessorImpl.ConsumeError with the CASH compartment already applied.
	onError func(err error) error
}

// consumeMorphCoupon applies a transformation coupon: read, commit, then heal
// and morph.
//
// Ordering is deliberate and mirrors ConsumeStandard. Both reads are fallible
// and both happen before ConsumeItem, so a data failure returns the coupon to
// the player (FR-3.3) — losing a paid cash item to a no-op is the failure this
// ordering exists to prevent. Effect failures AFTER the commit are logged and
// not rolled back, matching the ApplyItemEffects convention.
func consumeMorphCoupon(l logrus.FieldLogger, ctx context.Context, d morphCouponDeps, transactionId uuid.UUID, characterId uint32, slot int16, itemId item2.Id) error {
	pg, _ := model.NewGroup(ctx)
	ff := model.Submit(pg, func() (field.Model, error) { return d.fields.GetMap(characterId) })
	fi := model.Submit(pg, func() (cash.Model, error) { return d.cash.GetById(uint32(itemId)) })
	if err := pg.Wait(); err != nil {
		return d.onError(err)
	}
	f, ci := ff.Get(), fi.Get()

	plan := computeMorphCouponPlan(ci)

	if err := d.compartment.ConsumeItem(characterId, inventory2.TypeValueCash, transactionId, slot); err != nil {
		return d.onError(err)
	}

	if plan.hp > 0 {
		if err := d.character.ChangeHP(f, characterId, plan.hp); err != nil {
			l.WithError(err).Errorf("Character [%d] consumed transformation coupon [%d] but the HP heal of [%d] failed.", characterId, itemId, plan.hp)
		}
	}
	if len(plan.statups) > 0 {
		// Re-use while already morphed is intentionally unguarded: the second
		// apply replaces the active morph and restarts the timer, which is the
		// default overwrite behaviour of the atlas-buffs apply path (FR-3.8).
		if err := d.buff.Apply(f, characterId, -int32(itemId), byte(0), plan.duration, plan.statups)(characterId); err != nil {
			l.WithError(err).Errorf("Character [%d] consumed transformation coupon [%d] but the morph buff apply failed.", characterId, itemId)
		}
	}

	if plan.hp == 0 && len(plan.statups) == 0 {
		l.Warnf("Character [%d] consumed transformation coupon [%d] but its cash data carries neither a morph nor an hp spec; the tenant's cash WZ likely predates the spec/morph and spec/hp parse.", characterId, itemId)
	}
	return nil
}

// ConsumeMorphCoupon commits a reserved transformation coupon (classification
// 530) from the CASH compartment and applies its morph and HP heal.
//
// It cannot reuse ConsumeStandard: that consumer hard-codes
// inventory2.TypeValueUse and fetches from the *consumable* data resource, where
// cash items do not exist.
func ConsumeMorphCoupon(transactionId uuid.UUID, characterId uint32, slot int16, itemId item2.Id) ItemConsumer {
	return func(l logrus.FieldLogger) func(ctx context.Context) error {
		return func(ctx context.Context) error {
			p := NewProcessor(l, ctx)
			d := morphCouponDeps{
				cash:        cash.NewProcessor(l, ctx),
				fields:      character2.NewProcessor(l, ctx),
				compartment: compartment.NewProcessor(l, ctx),
				character:   character.NewProcessor(l, ctx),
				buff:        buff.NewProcessor(l, ctx),
				onError: func(err error) error {
					return p.ConsumeError(characterId, transactionId, inventory2.TypeValueCash, slot, err)
				},
			}
			return consumeMorphCoupon(l, ctx, d, transactionId, characterId, slot, itemId)
		}
	}
}
```

Add `"github.com/Chronicle20/atlas/libs/atlas-constants/field"` to the import block for the `field.Model` in the `model.Submit` closure.

- [x] **Step 4: Run the tests to verify they pass**

```bash
cd services/atlas-consumables/atlas.com/consumables && go test -race ./consumable/ -run 'MorphCoupon' -v
cd services/atlas-consumables/atlas.com/consumables && go vet ./...
```
Expected: PASS for all seven `ConsumeMorphCoupon*` tests plus the Task 4 tests; vet clean.

- [x] **Step 5: Verify the duration guard**

```bash
cd "$(git rev-parse --show-toplevel)" && tools/buff-duration-guard.sh
```
Expected: exit 0. This is the guard that exists because the millisecond contract has been flipped three times in prose — do not skip it, and never silence it with `//buffdurationguard:allow`.

- [x] **Step 6: Commit**

```bash
cd "$(git rev-parse --show-toplevel)" && tools/lint.sh
git add services/atlas-consumables/atlas.com/consumables/consumable/morph_coupon.go services/atlas-consumables/atlas.com/consumables/consumable/morph_coupon_test.go
git commit -m "feat(task-219): add ConsumeMorphCoupon cash-compartment consumer"
```

---

### Task 6: Route classification 530 in `RequestItemConsume`

**Files:**
- Modify: `services/atlas-consumables/atlas.com/consumables/consumable/processor.go:284-287` (insert one branch into the classification chain)
- Test: `services/atlas-consumables/atlas.com/consumables/consumable/morph_coupon_test.go` (already written in Task 4 — `TestRoutesToMorphCoupon`, `TestMorphCouponNotStandardConsumer`)

**Interfaces:**
- Consumes: `routesToMorphCoupon` (Task 4), `ConsumeMorphCoupon` (Task 5).
- Produces: nothing new. `inventory2.TypeFromItemId(5300000)` already resolves to Cash (`processor.go:268`), so the reservation opened before this branch is on the right compartment.

**Placement matters.** The branch must go *before* the reward-table fallback at `processor.go:288` (`else if ci, derr := p.cdp.GetById(...)`). That fallback queries the *consumable* data resource; for a cash item the lookup fails, so it would fall through to `ConsumeBare` — which consumes the coupon and applies nothing. Putting the new branch ahead of it is what prevents a silent no-op consume. Insert it directly after the `ClassificationConsumableMonsterCard` branch.

- [x] **Step 1: Confirm the routing tests currently pass on the predicate but the router is unwired**

```bash
cd services/atlas-consumables/atlas.com/consumables && go test ./consumable/ -run 'TestRoutesToMorphCoupon|TestMorphCouponNotStandardConsumer' -v
grep -n "routesToMorphCoupon" consumable/processor.go
```
Expected: both tests PASS (they exercise the predicate from Task 4); the grep prints nothing — the router does not use it yet.

- [x] **Step 2: Add the routing branch**

In `services/atlas-consumables/atlas.com/consumables/consumable/processor.go`, insert immediately after the `ClassificationConsumableMonsterCard` branch and before the `else if ci, derr := p.cdp.GetById(...)` reward-table fallback:

```go
	} else if routesToMorphCoupon(itemId) {
		// Transformation (morph) coupon, classification 530, CASH compartment.
		// Must precede the reward-table fallback below: that fallback queries the
		// *consumable* data resource, which has no cash items, so a 530 item would
		// fall through to ConsumeBare and be destroyed with no effect applied.
		// Deliberately NOT in usesStandardConsumer — ConsumeStandard hard-codes
		// inventory2.TypeValueUse and fetches from the same consumable resource.
		itemConsumer = ConsumeMorphCoupon(transactionId, characterId, slot, itemId)
```

- [x] **Step 3: Add a test pinning the branch order**

Append to `morph_coupon_test.go`:

```go
// TestMorphCouponRoutedBeforeRewardFallback guards the branch ORDER in
// RequestItemConsume. The reward-table fallback queries the consumable data
// resource, which has no cash items, so a 530 item reaching it falls through to
// ConsumeBare — the coupon is destroyed and nothing is applied. A source check
// is the honest test here: the routing chain is a private if/else inside a
// method that opens a Kafka reservation, so it has no seam to observe.
func TestMorphCouponRoutedBeforeRewardFallback(t *testing.T) {
	src, err := os.ReadFile("processor.go")
	if err != nil {
		t.Fatalf("read processor.go: %v", err)
	}
	morphAt := bytes.Index(src, []byte("routesToMorphCoupon(itemId)"))
	if morphAt < 0 {
		t.Fatal("RequestItemConsume has no routesToMorphCoupon branch")
	}
	fallbackAt := bytes.Index(src, []byte("validateRewardTable(ci.Rewards())"))
	if fallbackAt < 0 {
		t.Fatal("could not locate the reward-table fallback branch")
	}
	if morphAt > fallbackAt {
		t.Error("the morph-coupon branch must precede the reward-table fallback, or 530 items fall through to ConsumeBare")
	}
}
```

Add `"bytes"` and `"os"` to the test file's import block.

- [x] **Step 4: Run the full package suite**

```bash
cd services/atlas-consumables/atlas.com/consumables && go test -race ./... 2>&1 | tail -30
cd services/atlas-consumables/atlas.com/consumables && go vet ./... && go build ./...
```
Expected: all PASS, vet and build clean. Pay attention to `TestUsesStandardConsumer` — it must still pass unchanged, confirming 530 was not added there.

- [x] **Step 5: Commit**

```bash
cd "$(git rev-parse --show-toplevel)" && tools/lint.sh
git add services/atlas-consumables/atlas.com/consumables/consumable/processor.go services/atlas-consumables/atlas.com/consumables/consumable/morph_coupon_test.go
git commit -m "feat(task-219): route transformation coupons to ConsumeMorphCoupon"
```

---

### Task 7: The `atlas-channel` handler arm

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go` — insert the arm at line 640 (after the megaphone/avatar-megaphone block closes at 639, before the terminal warn at 641); add the test seam near `cashItemInSlotFunc` (~line 684); add one import
- Test: `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_test.go` (append)

**Interfaces:**
- Consumes: `cashsb.NewItemUseMorphCoupon(updateTimeFirst bool)` and `UpdateTime()` (Task 1); `cashsb.UpdateTimeFirst(t)`; the in-scope locals `category`, `updateTimeFirst`, `updateTime`, `itemId`, `source`, `s`; `consumable.NewProcessor(l, ctx).RequestItemConsume(f field.Model, characterId character.Id, itemId item.Id, source slot.Position, quantity int16, updateTime uint32) error` (`atlas-channel/consumable/processor.go:43`); the existing test helpers `installCashItemInSlotSeam`, `newCashItemUseTestSession`, `cashItemUsePrefix`, `mustTenant`.
- Produces: `var requestItemConsumeFunc` — a package-level test seam, following the `cashItemInSlotFunc` / `useRockFunc` precedent already in this package.

- [x] **Step 1: Write the failing tests**

Append to `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_test.go`:

```go
// morphCouponCall records one forwarded consume request.
type morphCouponCall struct {
	itemId     item.Id
	source     slot.Position
	quantity   int16
	updateTime uint32
}

// installRequestItemConsumeSeam swaps requestItemConsumeFunc for the test
// (precedent: installUseRockSeam, installCashItemInSlotSeam) so no Kafka broker
// is needed.
func installRequestItemConsumeSeam(t *testing.T) (*[]morphCouponCall, func()) {
	t.Helper()
	calls := make([]morphCouponCall, 0)
	orig := requestItemConsumeFunc
	requestItemConsumeFunc = func(_ logrus.FieldLogger, _ context.Context, _ field.Model, _ character.Id, itemId item.Id, source slot.Position, quantity int16, updateTime uint32) error {
		calls = append(calls, morphCouponCall{itemId, source, quantity, updateTime})
		return nil
	}
	return &calls, func() { requestItemConsumeFunc = orig }
}

const cashMorphSlot = int16(3)

// TestCharacterCashItemUseHandleFunc_MorphCouponInvokesConsume: a v83 5300000
// request reaches the new arm and forwards the consume command with the trailing
// updateTime decoded (FR-4.1, FR-1.2). Before this task it fell through to the
// terminal warn and nothing happened at all.
func TestCharacterCashItemUseHandleFunc_MorphCouponInvokesConsume(t *testing.T) {
	const itemId = uint32(5300000)
	if item.GetClassification(item.Id(itemId)) != item.ClassificationTransformationCoupon {
		t.Fatalf("fixture invalid: GetClassification(%d) = %d, want 530", itemId, item.GetClassification(item.Id(itemId)))
	}
	restoreSlot := installCashItemInSlotSeam(t, cashMorphSlot, itemId)
	defer restoreSlot()
	calls, restoreConsume := installRequestItemConsumeSeam(t)
	defer restoreConsume()

	s, ctx, cleanup := newCashItemUseTestSession(t, 555)
	defer cleanup()

	// v83 trails updateTime in the sub-body (cashsb.UpdateTimeFirst false).
	raw := append(cashItemUsePrefix(cashMorphSlot, itemId),
		0x2A, 0x00, 0x00, 0x00, // updateTime = 42
	)
	req := request.Request(raw)
	reader := request.NewRequestReader(&req, 0)

	CharacterCashItemUseHandleFunc(logrus.New(), ctx, nil)(s, &reader, map[string]interface{}{})

	if len(*calls) != 1 {
		t.Fatalf("RequestItemConsume call count = %d, want 1", len(*calls))
	}
	c := (*calls)[0]
	if c.itemId != item.Id(itemId) {
		t.Errorf("itemId = %d, want %d", c.itemId, itemId)
	}
	if c.source != slot.Position(cashMorphSlot) {
		t.Errorf("source = %d, want %d", c.source, cashMorphSlot)
	}
	if c.quantity != 1 {
		t.Errorf("quantity = %d, want 1", c.quantity)
	}
	if c.updateTime != 42 {
		t.Errorf("updateTime = %d, want 42 (decoded from the trailing sub-body int32)", c.updateTime)
	}
}

// TestCharacterCashItemUseHandleFunc_MorphCouponV95NoSubBody: on GMS v95 the
// common ItemUse header already carried updateTime, so the sub-body reads
// nothing and the header value is forwarded (FR-1.2's leading half).
func TestCharacterCashItemUseHandleFunc_MorphCouponV95NoSubBody(t *testing.T) {
	const itemId = uint32(5300000)
	restoreSlot := installCashItemInSlotSeam(t, cashMorphSlot, itemId)
	defer restoreSlot()
	calls, restoreConsume := installRequestItemConsumeSeam(t)
	defer restoreConsume()

	ten := mustTenant(t, "GMS", 95, 1)
	ctx := tenant.WithContext(context.Background(), ten)
	sessionId := uuid.New()
	sess := session.NewSession(sessionId, ten, 0, nil)
	session.AddSessionToRegistry(ten.Id(), sess)
	defer session.ClearRegistryForTenant(ten.Id())
	sp := session.NewProcessor(logrus.New(), ctx)
	sp.SetCharacterId(sessionId, 555)
	s := sp.SetField(sessionId, field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100000000)).Build())

	// Leading updateTime, then the ItemUse prefix. No trailing bytes.
	raw := append([]byte{0x2A, 0x00, 0x00, 0x00}, cashItemUsePrefix(cashMorphSlot, itemId)...)
	req := request.Request(raw)
	reader := request.NewRequestReader(&req, 0)

	CharacterCashItemUseHandleFunc(logrus.New(), ctx, nil)(s, &reader, map[string]interface{}{})

	if len(*calls) != 1 {
		t.Fatalf("RequestItemConsume call count = %d, want 1", len(*calls))
	}
	if (*calls)[0].updateTime != 42 {
		t.Errorf("updateTime = %d, want 42 (from the leading header, not a sub-body read)", (*calls)[0].updateTime)
	}
}

// TestCharacterCashItemUseHandleFunc_MorphCouponTypeByteCollisions pins FR-1.3.
//
// The cash-slot type bytes collide ACROSS versions, not within one tenant
// (GetCashSlotItemType, character_cash_item_use.go): classification 522
// gachapon -> 40 on GMS >= 95; classification 530 transformation -> 40 on
// GMS < 95 and 41 on GMS >= 95; classification 538 pet evolution -> 41 on
// GMS < 95. So byte 40 means "transformation" on v83 and "gachapon" on v95,
// and byte 41 means "pet evolution" on v83 and "transformation" on v95. A
// type-byte-keyed arm would silently swap meaning at a version bump; the arm
// gates on classification instead, and neither collider may enter it.
func TestCharacterCashItemUseHandleFunc_MorphCouponTypeByteCollisions(t *testing.T) {
	tests := []struct {
		name     string
		region   string
		major    uint16
		itemId   uint32
		wantType CashSlotItemType
	}{
		{"v95 gachapon coupon shares byte 40 with v83 transformation", "GMS", 95, 5220000, CashSlotItemType(40)},
		{"v83 pet evolution shares byte 41 with v95 transformation", "GMS", 83, 5380000, CashSlotItemType(41)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ten := mustTenant(t, tc.region, tc.major, 1)
			ctx := tenant.WithContext(context.Background(), ten)
			// Confirm the collision is real under this tenant, so the test is
			// exercising the disambiguation rather than a vacuous case.
			if got := GetCashSlotItemType(ten)(item.Id(tc.itemId)); got != tc.wantType {
				t.Fatalf("fixture invalid: GetCashSlotItemType(%d) = %d, want %d", tc.itemId, got, tc.wantType)
			}
			if item.GetClassification(item.Id(tc.itemId)) == item.ClassificationTransformationCoupon {
				t.Fatalf("fixture invalid: %d is classification 530", tc.itemId)
			}

			restoreSlot := installCashItemInSlotSeam(t, cashMorphSlot, tc.itemId)
			defer restoreSlot()
			calls, restoreConsume := installRequestItemConsumeSeam(t)
			defer restoreConsume()

			sessionId := uuid.New()
			sess := session.NewSession(sessionId, ten, 0, nil)
			session.AddSessionToRegistry(ten.Id(), sess)
			defer session.ClearRegistryForTenant(ten.Id())
			sp := session.NewProcessor(logrus.New(), ctx)
			sp.SetCharacterId(sessionId, 555)
			s := sp.SetField(sessionId, field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100000000)).Build())

			raw := append(cashItemUsePrefix(cashMorphSlot, tc.itemId), 0x2A, 0x00, 0x00, 0x00)
			if cashsb.UpdateTimeFirst(ten) {
				raw = append([]byte{0x2A, 0x00, 0x00, 0x00}, cashItemUsePrefix(cashMorphSlot, tc.itemId)...)
			}
			req := request.Request(raw)
			reader := request.NewRequestReader(&req, 0)

			CharacterCashItemUseHandleFunc(logrus.New(), ctx, nil)(s, &reader, map[string]interface{}{})

			if len(*calls) != 0 {
				t.Errorf("RequestItemConsume call count = %d, want 0 — classification %d must not enter the morph-coupon arm", len(*calls), item.GetClassification(item.Id(tc.itemId)))
			}
		})
	}
}

// TestCharacterCashItemUseHandleFunc_MorphCouponMismatchedSlotNotInvoked: the
// arm inherits the ownership check by position (FR-4.2), and a mismatch must
// send nothing — no consume, and no unlock, exactly as every neighbouring arm.
func TestCharacterCashItemUseHandleFunc_MorphCouponMismatchedSlotNotInvoked(t *testing.T) {
	const itemId = uint32(5300000)
	// The seam reports a different template id for this slot.
	restoreSlot := installCashItemInSlotSeam(t, cashMorphSlot, 5300002)
	defer restoreSlot()
	calls, restoreConsume := installRequestItemConsumeSeam(t)
	defer restoreConsume()

	s, ctx, cleanup := newCashItemUseTestSession(t, 555)
	defer cleanup()

	raw := append(cashItemUsePrefix(cashMorphSlot, itemId), 0x2A, 0x00, 0x00, 0x00)
	req := request.Request(raw)
	reader := request.NewRequestReader(&req, 0)

	CharacterCashItemUseHandleFunc(logrus.New(), ctx, nil)(s, &reader, map[string]interface{}{})

	if len(*calls) != 0 {
		t.Errorf("RequestItemConsume call count = %d, want 0 on a template-id mismatch", len(*calls))
	}
}
```

Add to the test file's import block whatever is not already present: `"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"`, `cashsb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/serverbound"`. `context`, `uuid`, `logrus`, `channel`, `field`, `item`, `_map`, `world`, `request`, `tenant` and `session` are already imported there.

- [x] **Step 2: Run the tests to verify they fail**

```bash
cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run 'MorphCoupon' -v
```
Expected: FAIL — `undefined: requestItemConsumeFunc`. (After the seam exists but before the arm, the invocation tests fail on a call count of 0 while the two negative tests pass — that is the correct intermediate state.)

- [x] **Step 3: Add the test seam**

In `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go`, immediately after the `cashItemInSlotFunc` block (~line 692):

```go
// requestItemConsumeFunc is a test seam over the atlas-consumables consume
// command emit (package-var injection precedent: cashItemInSlotFunc above,
// useRockFunc in teleport_rock_use.go). Handler tests must not require a live
// Kafka broker to assert which arm a request reached.
var requestItemConsumeFunc = func(l logrus.FieldLogger, ctx context.Context, f field.Model, characterId character.Id, itemId item.Id, source slot.Position, quantity int16, updateTime uint32) error {
	return consumable.NewProcessor(l, ctx).RequestItemConsume(f, characterId, itemId, source, quantity, updateTime)
}
```

Add `"github.com/Chronicle20/atlas/libs/atlas-constants/field"` to the import block. The alias `field` is free — the packet-side imports are already aliased `fieldpkt` and `fieldcb`.

- [x] **Step 4: Add the arm**

In the same file, insert at line 640 — after the megaphone/avatar-megaphone block's closing `}` (line 639) and before the terminal `l.Warnf` (line 641):

```go
		// Transformation (morph) coupons, classification 530. Gated on
		// CLASSIFICATION, never on the cash-slot type byte `it`: those bytes
		// collide across versions (GetCashSlotItemType maps 530 -> 41 on
		// GMS >= 95 and 40 otherwise, while 522 gachapon takes 40 on GMS >= 95
		// and 538 pet evolution takes 41 on GMS < 95), so a type-keyed arm
		// would change meaning at a version bump.
		//
		// The sub-body is empty apart from the trailing updateTime on the
		// versions that trail it (IDA-verified: the case-40 arm of
		// CWvsContext::SendConsumeCashItemUseRequest @0xa0caf0-0xa0cb37 on GMS
		// v83 contains no Encode* call).
		//
		// No EnableActions: the effect does not warp, and the non-silent
		// INVENTORY_OPERATION emitted by the consume commit already clears the
		// client's exclusive-request lock — CWvsContext::OnInventoryOperation
		// @0xa1ead9 clears the same dword pair OnGameStageChanged does, gated
		// on the packet's leading bOnExclRequest byte, which
		// inventory/clientbound/change_batch.go writes as !silent.
		if category == item.ClassificationTransformationCoupon {
			sp := cashsb.NewItemUseMorphCoupon(updateTimeFirst)
			sp.Decode(l, ctx)(r, readerOptions)
			if !updateTimeFirst {
				updateTime = sp.UpdateTime()
			}
			_ = requestItemConsumeFunc(l, ctx, s.Field(), character.Id(s.CharacterId()), itemId, source, 1, updateTime)
			return
		}

```

- [x] **Step 5: Run the tests to verify they pass**

```bash
cd services/atlas-channel/atlas.com/channel && go test -race ./socket/handler/ -run 'MorphCoupon' -v
cd services/atlas-channel/atlas.com/channel && go test -race ./socket/... 2>&1 | tail -20
cd services/atlas-channel/atlas.com/channel && go vet ./... && go build ./...
```
Expected: all five new tests PASS; every pre-existing handler test still PASSES (in particular the four teleport-rock/megaphone arm tests, which share the seam helpers); vet and build clean.

- [x] **Step 6: Confirm the terminal warn no longer fires for 530**

```bash
cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run 'MorphCouponInvokesConsume' -v 2>&1 | grep -i "attempting to use cash item" || echo "OK: terminal warn not reached for classification 530"
```
Expected: `OK: …`. If the warn text appears, the arm was placed after the warn or the classification gate is wrong.

- [x] **Step 7: Commit**

```bash
cd "$(git rev-parse --show-toplevel)" && tools/lint.sh
git add services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_test.go
git commit -m "feat(task-219): route classification 530 to the consume path in atlas-channel"
```

---

### Task 8: Full verification sweep and documentation

**Files:**
- Modify: `docs/research/missing-features/items-and-consumables.md:38` (retire "Wholly missing #7")

**Interfaces:**
- Consumes: everything above. Produces no code.

- [x] **Step 1: Run every module's tests, vet and build**

```bash
cd "$(git rev-parse --show-toplevel)"
for m in libs/atlas-packet services/atlas-data/atlas.com/data services/atlas-consumables/atlas.com/consumables services/atlas-channel/atlas.com/channel; do
  echo "===== $m ====="
  ( cd "$m" && go test -race ./... 2>&1 | tail -15 && go vet ./... && go build ./... ) || echo "FAILED: $m"
done
```
Expected: no `FAILED:` line, no vet or build diagnostics, every package `ok` or `no test files`.

- [x] **Step 2: Run the repo-root guards**

```bash
cd "$(git rev-parse --show-toplevel)"
tools/redis-key-guard.sh    && echo "redis-key-guard OK"
tools/goroutine-guard.sh    && echo "goroutine-guard OK"
tools/buff-duration-guard.sh && echo "buff-duration-guard OK"
tools/lint.sh --check       && echo "lint OK"
```
Expected: all four print their OK line. `tools/lint.sh --check` needs nvm on PATH for the atlas-ui half; if it false-fails on the frontend with no TS files changed, run `nvm use 22` first and re-run — do not declare it clean without seeing exit 0.

The four template guards and `service-registration-guard.sh` are **not** required: no seed template, `services.json`, `deploy/k8s`, `docker-bake.hcl`, `go.work` or `tools/db-bootstrap.sh` file changed. Confirm that claim rather than asserting it:

```bash
git diff --stat main...HEAD -- services/atlas-configurations/seed-data/templates/ .github/config/services.json deploy/k8s docker-bake.hcl go.work tools/db-bootstrap.sh
```
Expected: empty output.

- [x] **Step 3: Confirm no `go.mod` was touched, so no bake is required**

```bash
git diff --name-only main...HEAD -- '*go.mod' '*go.sum'
```
Expected: empty. If anything prints, `docker buildx bake atlas-<svc>` from the worktree root becomes **mandatory** for each affected service (CLAUDE.md item 4).

- [x] **Step 4: Audit the diff for the global constraints**

```bash
cd "$(git rev-parse --show-toplevel)"
# No region/major-version literal in the new code.
git diff main...HEAD -- services libs | grep -nE '^\+' | grep -E 'MajorVersion\(\)|Region\(\) ==' || echo "OK: no version literal"
# No raw 530 classification literal outside atlas-constants.
git diff main...HEAD -- services libs | grep -nE '^\+' | grep -E '\b530\b' || echo "OK: no raw 530"
# No seconds<->milliseconds scaling near the duration.
git diff main...HEAD | grep -nE '^\+.*(duration|Duration).*(\*|/) *1000' || echo "OK: no duration scaling"
# No EnableActions added.
git diff main...HEAD | grep -nE '^\+.*EnableActions' || echo "OK: no EnableActions"
# No TODOs.
git diff main...HEAD | grep -nE '^\+.*(TODO|FIXME|XXX|HACK)' || echo "OK: no TODO markers"
```
Expected: five `OK:` lines. `5300000`/`5300002` inside *test* fixtures are item ids, not the classification, and are fine — the `530` grep should only surface those; read each hit rather than accepting the count.

- [x] **Step 5: Update the backlog doc**

Read `docs/research/missing-features/items-and-consumables.md` around line 38, then edit the "Wholly missing #7 / Transformation-morph coupons" row to record: implemented in `task-219`; the effect fires wherever the tenant's cash WZ carries `Cash/0530` items with `spec/morph`; **inert on `gms_12`**, whose `template_gms_12_1.json` does not register `CharacterCashItemUseHandle` at all (a pre-existing gap for the entire cash-item-use family, not a regression from this task); and **inert for any tenant whose cash WZ was ingested before this change**, because the reader change materialises `morph`/`hp` only for newly ingested data.

Match the file's existing row/table format — read it before writing, don't impose a new shape.

- [x] **Step 6: File the operational re-ingest follow-up**

The last PRD acceptance criterion asks for a follow-up item covering the operational re-ingest plus a live `GET /cash-items/5300000` check per tenant. Add it to the repo's tracking doc — locate it first, per CLAUDE.md ("use Glob or Grep to find the file rather than assuming a path"):

```bash
cd "$(git rev-parse --show-toplevel)" && ls docs/TODO.md docs/tasks/README.md 2>/dev/null; grep -rln "follow-up" docs/*.md | head
```

Record: re-ingest cash WZ for every provisioned tenant, then verify `GET /data/{tenantId}/cash-items/5300000` returns `spec` containing `morph: 1`, `hp: 50`, `time: 600000`. Until that runs, a coupon use consumes the item and applies nothing (the "both absent" row of the error table) — note this explicitly so the first bug report against it is self-answering.

- [x] **Step 7: Verify the worktree is the right one and commit**

```bash
cd "$(git rev-parse --show-toplevel)"
git rev-parse --show-toplevel   # must end with /.worktrees/task-219-cash-morph-coupons
git branch --show-current       # must be task-219-cash-morph-coupons
git status --short
git add docs/
git commit -m "docs(task-219): retire morph-coupon backlog entry; file re-ingest follow-up"
```

- [x] **Step 8: Walk the PRD §10 acceptance criteria and tick each one**

Open `prd.md` §10 and check off each box against real evidence — the test name that pins it, or the command output that proves it. The exclusive-request-lock criterion (FR-4.3) is satisfied by design §1.2's IDA evidence plus the "no EnableActions" grep in Step 4; the type-byte-collision criterion is satisfied by `TestCharacterCashItemUseHandleFunc_MorphCouponTypeByteCollisions` **as corrected** — record the PRD's factual error about pre-95 gachapon in the same edit rather than ticking a box whose premise is wrong. Commit the ticked PRD.

- [x] **Step 9: Code review before PR**

Run the code-review step — do not skip it even though the plan looks complete (CLAUDE.md). Go files changed in three services, no TypeScript, so `superpowers:requesting-code-review` should dispatch `plan-adherence-reviewer` and `backend-guidelines-reviewer` (not the frontend one). Pin the reviewer subagents to Sonnet. Findings land in `docs/tasks/task-219-cash-morph-coupons/audit.md`. Ensure the subagents run **inside this worktree** and the tree is clean afterwards.

---

## Self-Review

**Spec coverage.** Every PRD functional requirement maps to a task: FR-1.1 (no new layout — the existing `ItemUse` header is reused unchanged, Task 7); FR-1.2 → Task 1; FR-1.3 → Task 4 (`routesToMorphCoupon`) + Task 7 (arm gate and collision tests); FR-2.1/2.2 → Task 2 Steps 3-4; FR-2.3/2.4 → Task 2 Step 1; FR-3.1 → Task 3; FR-3.2 → Task 6; FR-3.3/3.4 → Task 5; FR-3.5/3.6 → Tasks 4 and 5; FR-3.7 → Task 4 table + Task 5 `TestConsumeMorphCouponZeroSpecs`; FR-3.8 → Task 5 `TestConsumeMorphCouponReuseWhileMorphedApplies`; FR-4.1 → Task 7 Step 4; FR-4.2 → Task 7 `MorphCouponMismatchedSlotNotInvoked`; FR-4.3 → design §1.2 (resolved: no unlock emitted) + Task 8 Step 4 grep. Design §3.1-§3.6 all land in Tasks 1, 2, 3, 4, 5. Design §7's four limitations are recorded in Task 8 Steps 5-6, except §7.3 (`ConsumeCashPetFood`'s `TypeValueUse` inconsistency), which stays untouched by explicit design decision — flagged again below.

**Placeholders.** None: every code step carries the literal code, every test step the literal test, every run step the literal command and expected outcome. Two steps hedge on a detail the implementer must read first — `pt.Encode`'s exact signature (Task 1 Step 1) and the mock field names (Task 5 Step 1) — with the assertion stated so an adaptation cannot lose the intent. Both were confirmed to exist during planning; the hedge is against a signature drift, not unknown content.

**Type consistency.** `computeMorphCouponPlan(cash.Model) morphCouponPlan` with fields `hp int16` / `statups []stat.Model` / `duration int32` is used identically in Tasks 4 and 5. `ConsumeMorphCoupon(transactionId uuid.UUID, characterId uint32, slot int16, itemId item2.Id) ItemConsumer` — the arg order in Task 5's definition matches Task 6's call and `ConsumeCashPetFood`'s convention. `consumeMorphCoupon(l, ctx, d, transactionId, characterId, slot, itemId)` is spelled the same in the definition and in all six test call sites (note: `transactionId` precedes `characterId`, unlike the exported wrapper's caller-facing order — deliberate, matching the wrapper's internal call). `routesToMorphCoupon(item2.Id) bool` matches its use in Task 6. `NewItemUseMorphCoupon(bool) *ItemUseMorphCoupon` / `UpdateTime() uint32` match between Tasks 1 and 7. `SpecTypeMorph`/`SpecTypeHp` carry identical string values in atlas-data (Task 2) and atlas-consumables (Task 3), pinned by `TestSpecTypeWireValues`.

**Two things a reviewer should push back on if they disagree:**

1. `morphCouponDeps` is a new pattern in the `consumable` package — no other consumer there takes injected collaborators. It is the smallest change that makes four PRD acceptance criteria into real assertions, but it is a pattern introduction, and a reviewer who prefers matching the package's existing shape exactly would have to accept those four criteria being unpinned. Correction 3 above states the trade-off.
2. `docs/research/missing-features/items-and-consumables.md` and the follow-up tracking doc are edited without their current contents being read during planning. Task 8 Steps 5-6 instruct reading first and matching the existing format; if the row shape or the tracking-doc location differs from what those steps assume, follow the file, not the plan.

**Known limitation carried forward, not fixed:** `ConsumeCashPetFood` (`processor.go:591-599`) passes `inventory2.TypeValueUse` on three paths for what is unambiguously a Cash-compartment item, while its first `ConsumeError` at `:576` correctly passes `TypeValueCash`. That looks like a live defect in a neighbouring consumer. This task does not touch it — different item family, different acceptance criteria, and no evidence was gathered on its runtime impact — but it should not be lost. It is recorded in design §7.3 and worth its own task with its own evidence.

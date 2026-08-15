# Scissors of Karma Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the Scissors of Karma end-to-end — the `USE_CASH_ITEM` karma sub-body, a server-gated handler arm, the WZ eligibility props, an in-place asset flag mutation through the saga machinery, and a one-shot trade/merchant override that is consumed on transfer.

**Architecture:** The client sends `USE_CASH_ITEM` with a karma sub-body (`int32 targetInventoryType`, `int32 targetSlot`). atlas-channel decodes it, resolves the cash-slot type from the tenant version, applies nine refusal gates (five structural, three mirroring `CUIKarmaDlg::PutItem`, one server-only), and on success creates a two-step saga: `DestroyAsset` (the scissors) then `ApplyAssetKarma` (the target). atlas-saga-orchestrator dispatches an `APPLY_KARMA` Kafka command to atlas-inventory, which re-asserts every gate (the owning service is the authority), sets the slot-class-correct karma bit in the existing `flag uint16` column, and emits the existing `UPDATED` status event. atlas-trades and atlas-merchant treat a karma-marked asset as tradeable despite `FlagUntradeable`/`FlagMergeUntradeable`/WZ `tradeBlock`, and clear the bit **in the transfer snapshot** at the two sites that build the receiving asset — making consumption atomic with the transfer by construction.

**Tech Stack:** Go 1.24 microservices; GORM/Postgres; Kafka (`libs/atlas-kafka` `message.Buffer` + outbox); JSON:API REST via api2go; `libs/atlas-packet` codecs; `libs/atlas-saga` shared saga contract; `libs/atlas-constants` shared domain types.

**Spec:** [`design.md`](design.md) (PRD: [`prd.md`](prd.md); IDA derivation record: [`ida-findings.md`](ida-findings.md))

## Global Constraints

- **Worktree.** All work happens in `/.worktrees/task-223-karma-scissors` on branch `task-223-karma-scissors`. Never edit the main checkout.
- **The karma bit is slot-class dependent.** `0x10` (`asset.FlagKarmaEquip`) on an equip, `0x02` (`asset.FlagKarmaUse`) on a bundle. Verified in the gms_v83 IDB (`GW_ItemSlotEquip::IsPossibleTradingItem @0x4E956E`, `GW_ItemSlotBundle::IsPossibleTradingItem @0x4E9B6A`) and gms_v95 (`@0x4F6130`, `@0x4F67A0`). **Constant values MUST NOT change** — they match the client. No call site may name a bit directly; every site goes through `KarmaFlagFor`.
- **`0x02` is `FlagSpikes` on an equip.** Karma mutations are targeted `SetFlag`/`ClearFlag` of the resolved bit only, never an assignment. This is the single sharpest failure mode in the task.
- **Pets are refused.** `KarmaFlagFor` returns `(0, false)` for pet-classification templates; every caller must handle the refusal. The pet karma bit is `0x01`, which aliases `FlagLock` in Atlas's shared `flag` column.
- **WZ property spellings (resolved in design §1/OQ-1, do not re-derive):**
  - target's applicable karma type: `info/tradeAvailable`, an **int** (StringPool id `3234`, pinned to `BUNDLEITEM+0x14` via neighbours `only@0xC`, `tradeBlock@0x10`, `notSale@0x18`).
  - scissors' own karma type: `info/karma`, an **int** (StringPool id `5595`).
  - Both parse with `GetIntegerWithDefault("<name>", 0)` — **integers, not bools**. The existing `TradeBlock bool`/`GetBool` shape must NOT be copied.
- **Eligibility predicate (design §3.2), identical on every version, no version gate:**
  `eligible := targetKarma != 0 && (scissorsKarma == 0 || targetKarma == scissorsKarma)`
- **Client wire values are config-resolved (DOM-25).** Cash-slot type comes from `karmaScissorsCashSlotItemType(t tenant.Model)`, never a bare constant compare. `GetCashSlotItemType`'s 552 return values (`64` on GMS ≥ 95, `63` otherwise) MUST NOT change.
- **Every refusal logs at warn** naming character id, scissors template id, target inventory type, target slot, resolved target template id, and the failing rule — and **unlocks the client** with an empty `StatChanged` carrying `ExclRequestSent`. No scissors consumed, no state mutated.
- **Verification gate (CLAUDE.md), run before claiming done:** `go test -race ./...`, `go vet ./...`, `go build ./...` clean in every changed module; `docker buildx bake atlas-<svc>` for every service whose `go.mod` was touched; `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`, `tools/lint.sh --check` clean from the repo root.
- **Commit style:** conventional commits scoped `task-223`, e.g. `feat(task-223): ...`.

---

## File Structure

| File | Responsibility |
|---|---|
| `libs/atlas-constants/asset/flag.go` (modify) | Document the slot-class context of each karma bit |
| `libs/atlas-constants/asset/karma.go` (create) | `KarmaFlagFor` (slot-class → bit) and `KarmaEligible` (the §3.2 predicate) |
| `libs/atlas-constants/item/constants.go` (modify) | `ClassificationKarmaScissors` |
| `libs/atlas-packet/cash/serverbound/item_use_karma_scissors.go` (create) | The karma sub-body codec |
| `services/atlas-data/.../{equipment,consumable,setup,etc,cash}/{reader,rest}.go` (modify) | Parse + expose `tradeAvailable`; `karma` on cash |
| `services/atlas-inventory/.../data/tradeability/` (create) | Per-compartment atlas-data client for `tradeAvailable` + `tradeBlock` |
| `services/atlas-inventory/.../asset/processor.go` (modify) | `ApplyKarma` / `ClearKarma` in-place mutators |
| `services/atlas-inventory/.../compartment/processor.go` (modify) | `ApplyAssetKarma` slot-addressed wrapper + gate re-assertion |
| `services/atlas-inventory/.../kafka/{message,consumer}/compartment/` (modify) | `APPLY_KARMA` command + handler |
| `libs/atlas-saga/{model,payloads,unmarshal}.go` (modify) | `KarmaScissorsUse` type, `ApplyAssetKarma` action + payload |
| `services/atlas-saga-orchestrator/.../{compartment,saga}/` (modify) | Command mirror, producer, dispatch, acceptance, compensation, settlement snapshot clear |
| `services/atlas-channel/.../data/{cash,tradeability}/` (modify/create) | Scissors `karma` field; target eligibility client |
| `services/atlas-channel/.../socket/handler/character_cash_item_use.go` (modify) | Type resolver + the karma arm |
| `services/atlas-trades/.../trade/restriction.go` + `data/item/` (modify) | Karma override of flags and `tradeBlock` |
| `services/atlas-merchant/.../shop/{validation,processor}.go` (modify) | Listing override + buy-path snapshot clear |
| `docs/tasks/task-223-karma-scissors/coverage-manifest.yaml` (create) | Packet coverage declaration |

---

## Task 1: The karma bit resolver

**Files:**
- Create: `libs/atlas-constants/asset/karma.go`
- Create: `libs/atlas-constants/asset/karma_test.go`
- Modify: `libs/atlas-constants/asset/flag.go:1-17`

**Interfaces:**
- Consumes: nothing.
- Produces:
  - `asset.KarmaFlagFor(templateId uint32) (Flag, bool)` — `(FlagKarmaEquip, true)` for equips, `(FlagKarmaUse, true)` for bundles, `(0, false)` for pets.
  - `asset.KarmaEligible(scissorsKarma int32, targetKarma int32) bool` — the design §3.2 predicate.

- [ ] **Step 1: Write the failing test**

Create `libs/atlas-constants/asset/karma_test.go`:

```go
package asset

import "testing"

func TestKarmaFlagFor(t *testing.T) {
	testCases := []struct {
		name       string
		templateId uint32
		wantFlag   Flag
		wantOk     bool
	}{
		{"equip zakum helmet", 1002357, FlagKarmaEquip, true},
		{"equip weapon", 1302000, FlagKarmaEquip, true},
		{"consumable mastery book", 2280000, FlagKarmaUse, true},
		{"setup chair", 3010000, FlagKarmaUse, true},
		{"etc material", 4000000, FlagKarmaUse, true},
		{"cash scissors", 5520000, FlagKarmaUse, true},
		{"pet low", 5000000, 0, false},
		{"pet high", 5009999, 0, false},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotFlag, gotOk := KarmaFlagFor(tc.templateId)
			if gotFlag != tc.wantFlag || gotOk != tc.wantOk {
				t.Fatalf("KarmaFlagFor(%d) = (%#x, %v), want (%#x, %v)", tc.templateId, gotFlag, gotOk, tc.wantFlag, tc.wantOk)
			}
		})
	}
}

// TestKarmaFlagForEquipIsNotSpikes is the FR-4.5 guard: the equip karma bit
// must never be 0x02, which is FlagSpikes on an equip.
func TestKarmaFlagForEquipIsNotSpikes(t *testing.T) {
	f, ok := KarmaFlagFor(1002357)
	if !ok {
		t.Fatal("KarmaFlagFor refused an equip")
	}
	if f == FlagSpikes {
		t.Fatalf("equip karma bit resolved to FlagSpikes (%#x); marking karma would render spikes", f)
	}
}

func TestKarmaEligible(t *testing.T) {
	testCases := []struct {
		name          string
		scissorsKarma int32
		targetKarma   int32
		want          bool
	}{
		{"v83 model: target not karma-applicable", 0, 0, false},
		{"v83 model: target karma-applicable", 0, 1, true},
		{"v95 model: types match", 1, 1, true},
		{"v95 model: types differ", 1, 2, false},
		{"v95 model: types differ reversed", 2, 1, false},
		{"v95 model: ordinary item, scissors typed", 1, 0, false},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := KarmaEligible(tc.scissorsKarma, tc.targetKarma); got != tc.want {
				t.Fatalf("KarmaEligible(%d, %d) = %v, want %v", tc.scissorsKarma, tc.targetKarma, got, tc.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-constants && go test ./asset/... -run 'TestKarma' -v`
Expected: FAIL — `undefined: KarmaFlagFor`, `undefined: KarmaEligible`.

- [ ] **Step 3: Write the implementation**

Create `libs/atlas-constants/asset/karma.go`:

```go
package asset

import "github.com/Chronicle20/atlas/libs/atlas-constants/item"

// KarmaFlagFor returns the karma-mark bit for an asset's slot class, plus
// whether karma applies to that class at all.
//
// The class is derived from the template id exactly as the client derives it:
// CItemInfo::GetAppliableKarmaType (gms_v95 @0x5C09F0) branches on
// nItemID / 1000000 == 1 to choose EQUIPITEM over BUNDLEITEM, and the two
// GW_ItemSlot subclasses read different bits out of nAttribute:
//
//	GW_ItemSlotEquip::IsPossibleTradingItem  gms_v83 @0x4E956E / gms_v95 @0x4F6130 -> nAttribute & 0x10
//	GW_ItemSlotBundle::IsPossibleTradingItem gms_v83 @0x4E9B6A / gms_v95 @0x4F67A0 -> nAttribute & 0x02
//
// Pets are DELIBERATELY refused rather than resolved. The client's
// GW_ItemSlotPet::IsPossibleTradingItem (gms_v83 @0x4EA01E) reads 0x01, which
// it can afford because GW_ItemSlotPet::IsProtectedItem (@0x4EA012) hard-returns
// 0 — the two meanings never coexist on one client object. Atlas has no such
// guarantee: FlagLock is written against the same shared `flag` column by the
// Sealing Lock arm, so a pet karma mark would read back as a lock. Returning
// (0, false) forces every caller to handle the case rather than silently
// writing 0x01. See design.md OQ-5.
func KarmaFlagFor(templateId uint32) (Flag, bool) {
	if item.GetClassification(item.Id(templateId)) == item.ClassificationPet {
		return 0, false
	}
	if templateId/1000000 == 1 {
		return FlagKarmaEquip, true
	}
	return FlagKarmaUse, true
}

// KarmaEligible reports whether a Scissors of Karma carrying scissorsKarma
// (WZ info/karma on the scissors, 0 when absent) may be applied to a target
// carrying targetKarma (WZ info/tradeAvailable on the target, 0 when absent).
//
// One predicate covers both client models (design §3.2-3.3):
//
//   - gms_v83 asks "is tradeAvailable non-zero?" (CItemInfo::IsAppliableKarmaItem
//     @0x5D4E8F). Its scissors carry no `karma` node, so scissorsKarma is 0, the
//     second clause is vacuous, and this reduces to exactly that test.
//   - gms_v87 (CUIKarmaDlg::PutItem @0x895261) and gms_v95 (@0x7D7BA0) ask
//     "does GetAppliableKarmaType(target) equal m_nKarmaType?". Their scissors
//     carry a `karma` node, so this is that equality plus one extra condition.
//
// The extra condition — targetKarma != 0 — closes a real client hole: a v95-era
// tenant whose WZ omits `karma` on the scissors makes the client compare
// 0 != tradeAvailable and thereby accept every ordinary item. The server must
// not. Being a strict subset of both client rules, the worst case on any
// un-decompiled version is a logged server refusal where the client would have
// allowed.
func KarmaEligible(scissorsKarma int32, targetKarma int32) bool {
	if targetKarma == 0 {
		return false
	}
	return scissorsKarma == 0 || targetKarma == scissorsKarma
}
```

- [ ] **Step 4: Document the bits in `flag.go`**

Replace the `const` block header in `libs/atlas-constants/asset/flag.go` so the file reads:

```go
package asset

type Flag uint16

// The client reads these bits out of GW_ItemSlotBase::nAttribute, and TWO of
// them are SLOT-CLASS DEPENDENT — the same value means different things on an
// equip, a bundle and a pet:
//
//	bit    equip                      bundle                     pet
//	0x01   lock (IsProtectedItem)     lock                       karma (IsProtectedItem hard-returns 0)
//	0x02   spikes                     KARMA MARK                 --
//	0x10   KARMA MARK                 --                         --
//
// gms_v83: GW_ItemSlotEquip::IsProtectedItem @0x4E9506, ::IsPossibleTradingItem
// @0x4E956E; GW_ItemSlotBundle::IsProtectedItem @0x4E9B4F,
// ::IsPossibleTradingItem @0x4E9B6A; GW_ItemSlotPet::IsProtectedItem @0x4EA012,
// ::IsPossibleTradingItem @0x4EA01E.
// gms_v95: equip @0x4F60B0 / @0x4F6130; bundle @0x4F6780 / @0x4F67A0.
//
// The names below read BACKWARDS from that table and are load-bearing in seven
// services, so they are documented rather than renamed: FlagKarmaUse (0x02) is
// the BUNDLE karma bit and FlagKarmaEquip (0x10) is the EQUIP karma bit. Never
// pick one by hand — call KarmaFlagFor (karma.go), which resolves the bit from
// the template id's slot class and refuses pets.
//
// The values are the client's and MUST NOT change.
const (
	FlagLock             Flag = 0x01
	FlagSpikes           Flag = 0x02
	FlagKarmaUse         Flag = 0x02
	FlagCold             Flag = 0x04
	FlagUntradeable      Flag = 0x08
	FlagKarmaEquip       Flag = 0x10
	FlagSandbox          Flag = 0x40
	FlagPetCome          Flag = 0x80
	FlagAccountSharing   Flag = 0x100
	FlagMergeUntradeable Flag = 0x200
)
```

Leave `HasFlag` / `SetFlag` / `ClearFlag` below unchanged.

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd libs/atlas-constants && go test -race ./... && go vet ./...`
Expected: PASS, vet clean.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-constants/asset/
git commit -m "feat(task-223): slot-class karma bit resolver and eligibility predicate"
```

---

## Task 2: Classification constant and the `ItemUseKarmaScissors` codec

**Files:**
- Modify: `libs/atlas-constants/item/constants.go:108` (insert in the 5xx block, sorted)
- Create: `libs/atlas-packet/cash/serverbound/item_use_karma_scissors.go`
- Create: `libs/atlas-packet/cash/serverbound/item_use_karma_scissors_test.go`

**Interfaces:**
- Consumes: `cashsb.UpdateTimeFirst(t tenant.Model) bool` (existing, `item_use.go:21-23`).
- Produces:
  - `item.ClassificationKarmaScissors = item.Classification(552)`
  - `serverbound.NewItemUseKarmaScissors(updateTimeFirst bool) *ItemUseKarmaScissors`
  - `(ItemUseKarmaScissors).InventoryType() int32`, `.Slot() int32`, `.UpdateTime() uint32`, `.Operation() string`
  - `.Encode(l, ctx) func(map[string]interface{}) []byte`, `(*ItemUseKarmaScissors).Decode(l, ctx) func(*request.Reader, map[string]interface{})`

- [ ] **Step 1: Write the failing test**

Create `libs/atlas-packet/cash/serverbound/item_use_karma_scissors_test.go`:

```go
package serverbound

import (
	"bytes"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestItemUseKarmaScissorsRoundTrip(t *testing.T) {
	for _, first := range []bool{true, false} {
		for _, v := range pt.Variants {
			t.Run(v.Name, func(t *testing.T) {
				ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
				input := ItemUseKarmaScissors{inventoryType: 1, slot: 3, updateTime: 1000, updateTimeFirst: first}
				output := *NewItemUseKarmaScissors(first)
				pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
				if output.InventoryType() != input.InventoryType() {
					t.Errorf("inventoryType = %d, want %d", output.InventoryType(), input.InventoryType())
				}
				if output.Slot() != input.Slot() {
					t.Errorf("slot = %d, want %d", output.Slot(), input.Slot())
				}
				if !first && output.UpdateTime() != input.UpdateTime() {
					t.Errorf("updateTime = %d, want %d", output.UpdateTime(), input.UpdateTime())
				}
			})
		}
	}
}

// v83 golden bytes (CUIKarmaDlg::_SendConsumeCashItemUseRequest @0x830FB5, which
// TRAILS update_time): int nTargetTI (1) + int nTargetPOS (3) + int update_time (1000).
func TestItemUseKarmaScissorsV83Bytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	m := ItemUseKarmaScissors{inventoryType: 1, slot: 3, updateTime: 1000, updateTimeFirst: false}
	got := m.Encode(l, pt.CreateContext("GMS", 83, 1))(nil)
	want := []byte{0x01, 0x00, 0x00, 0x00, 0x03, 0x00, 0x00, 0x00, 0xE8, 0x03, 0x00, 0x00}
	if !bytes.Equal(got, want) {
		t.Fatalf("got % X, want % X", got, want)
	}
}

// v95 golden bytes (@0x7D7EF0, which LEADS update_time in the common ItemUse
// header): the sub-body is int nTargetTI (1) + int nTargetPOS (3) and nothing else.
func TestItemUseKarmaScissorsV95Bytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	m := ItemUseKarmaScissors{inventoryType: 1, slot: 3, updateTime: 1000, updateTimeFirst: true}
	got := m.Encode(l, pt.CreateContext("GMS", 95, 0))(nil)
	want := []byte{0x01, 0x00, 0x00, 0x00, 0x03, 0x00, 0x00, 0x00}
	if !bytes.Equal(got, want) {
		t.Fatalf("got % X, want % X", got, want)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-packet && go test ./cash/serverbound/... -run 'KarmaScissors' -v`
Expected: FAIL — `undefined: ItemUseKarmaScissors`.

- [ ] **Step 3: Write the codec**

Create `libs/atlas-packet/cash/serverbound/item_use_karma_scissors.go`:

```go
package serverbound

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

// ItemUseKarmaScissors is the Scissors of Karma (classification 552) sub-body of
// the cash ItemUse packet, sent by CUIKarmaDlg::_SendConsumeCashItemUseRequest.
//
// Derived per version, both ends of the supported range:
//
//	gms_v83 @0x830FB5, opcode 0x4F:
//	  Encode2(m_nPOS) Encode4(m_nItemID) Encode4(m_nTargetTI) Encode4(m_nTargetPOS) Encode4(get_update_time())
//	gms_v95 @0x7D7EF0, opcode 0x55:
//	  Encode4(get_update_time()) Encode2(m_nPOS) Encode4(m_nItemID) Encode4(m_nTargetTI) Encode4(m_nTargetPOS)
//
// The leading Encode2+Encode4 pair is the common ItemUse header (item_use.go);
// the update_time position difference is the existing UpdateTimeFirst gate. What
// remains here is nTargetTI + nTargetPOS, byte-identical in shape to ItemUseSeal.
//
// This is a DISCRETE struct rather than an alias of ItemUseSeal, per the
// discrete-struct-per-mode rule in docs/packets/DISPATCHER_FAMILY.md. It is
// emphatically NOT ItemUseTargetSlot (a bare int16), which is the Item Tag /
// expiration-extender shape and carries no target inventory type.
type ItemUseKarmaScissors struct {
	inventoryType   int32
	slot            int32
	updateTime      uint32
	updateTimeFirst bool
}

func NewItemUseKarmaScissors(updateTimeFirst bool) *ItemUseKarmaScissors {
	return &ItemUseKarmaScissors{updateTimeFirst: updateTimeFirst}
}

func (m ItemUseKarmaScissors) InventoryType() int32 { return m.inventoryType }
func (m ItemUseKarmaScissors) Slot() int32          { return m.slot }
func (m ItemUseKarmaScissors) UpdateTime() uint32   { return m.updateTime }
func (m ItemUseKarmaScissors) Operation() string    { return "ItemUseKarmaScissors" }

func (m ItemUseKarmaScissors) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt32(m.inventoryType)
		w.WriteInt32(m.slot)
		if !m.updateTimeFirst {
			w.WriteInt(m.updateTime)
		}
		return w.Bytes()
	}
}

func (m *ItemUseKarmaScissors) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.inventoryType = r.ReadInt32()
		m.slot = r.ReadInt32()
		if !m.updateTimeFirst {
			m.updateTime = r.ReadUint32()
		}
	}
}
```

- [ ] **Step 4: Add the classification constant**

In `libs/atlas-constants/item/constants.go`, insert into the 5xx block in sorted position — between `ClassificationRemoteStore = Classification(547)` and `ClassificationViciousHammer = Classification(557)`:

```go
	ClassificationKarmaScissors            = Classification(552)
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd libs/atlas-packet && go test -race ./cash/serverbound/... && go vet ./...`
Then: `cd libs/atlas-constants && go build ./... && go vet ./...`
Expected: PASS, both clean.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-packet/cash/serverbound/item_use_karma_scissors.go libs/atlas-packet/cash/serverbound/item_use_karma_scissors_test.go libs/atlas-constants/item/constants.go
git commit -m "feat(task-223): ItemUseKarmaScissors sub-body codec and classification 552 constant"
```

---

## Task 3: atlas-data parses `tradeAvailable` (five readers) and `karma` (cash)

**Files:**
- Modify: `services/atlas-data/atlas.com/data/equipment/reader.go:114`, `equipment/rest.go:44`
- Modify: `services/atlas-data/atlas.com/data/consumable/reader.go:49`, `consumable/rest.go:46`
- Modify: `services/atlas-data/atlas.com/data/setup/reader.go:47`, `setup/rest.go:13`
- Modify: `services/atlas-data/atlas.com/data/etc/reader.go:47`, `etc/rest.go:13`
- Modify: `services/atlas-data/atlas.com/data/cash/reader.go:84`, `cash/rest.go:53`
- Create: `services/atlas-data/atlas.com/data/equipment/reader_karma_test.go`
- Create: `services/atlas-data/atlas.com/data/consumable/reader_karma_test.go`
- Create: `services/atlas-data/atlas.com/data/setup/reader_karma_test.go`
- Create: `services/atlas-data/atlas.com/data/etc/reader_karma_test.go`
- Create: `services/atlas-data/atlas.com/data/cash/reader_karma_test.go`

**Interfaces:**
- Consumes: `xml.Node.GetIntegerWithDefault(name string, def int32) int32` (existing).
- Produces: JSON field `tradeAvailable` (int32) on all five atlas-data item resources; JSON field `karma` (int32) on the cash resource. Both default to `0` when the WZ node is absent.

> **Why new `*_karma_test.go` files rather than edits to the existing `reader_test.go`:** each existing file carries one giant `testXML` const and asserts an exact item count (`len(rmm) != 55`). A self-contained fixture per reader avoids perturbing those counts.

- [ ] **Step 1: Write the failing test for the equipment reader**

Create `services/atlas-data/atlas.com/data/equipment/reader_karma_test.go`. Model the fixture on the real v83 corpus: `Character.wz/Cap/01002357.img` (Zakum Helmet) carries `only`, `tradeBlock` and `tradeAvailable` side by side under `info`.

```go
package equipment

import (
	"atlas-data/xml"
	"strconv"
	"testing"

	"github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// karmaTestXML mirrors the shipped v83 layout verbatim: 01002357 is the Zakum
// Helmet (Character.wz/Cap/01002357.img), which carries
// <int name="tradeAvailable" value="1"/> beside tradeBlock under info.
const karmaTestXML = `
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="01002357.img">
  <imgdir name="info">
    <int name="only" value="1"/>
    <int name="tradeBlock" value="1"/>
    <int name="tradeAvailable" value="1"/>
  </imgdir>
</imgdir>
`

// karmaAbsentTestXML is the same node with no tradeAvailable child.
const karmaAbsentTestXML = `
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="01002358.img">
  <imgdir name="info">
    <int name="tradeBlock" value="1"/>
  </imgdir>
</imgdir>
`

func readOne(t *testing.T, doc string, id int) RestModel {
	t.Helper()
	l, _ := test.NewNullLogger()
	rms := Read(l)(xml.FromByteArrayProvider([]byte(doc)))
	rmm, err := model.CollectToMap[RestModel, string, RestModel](rms, RestModel.GetID, Identity)()
	if err != nil {
		t.Fatal(err)
	}
	rm, ok := rmm[strconv.Itoa(id)]
	if !ok {
		t.Fatalf("rmm[%d] does not exist", id)
	}
	return rm
}

// TestReaderTradeAvailablePresent is the FR-3.5 real-item case: the Zakum Helmet.
func TestReaderTradeAvailablePresent(t *testing.T) {
	rm := readOne(t, karmaTestXML, 1002357)
	if rm.TradeAvailable != 1 {
		t.Fatalf("TradeAvailable = %d, want 1", rm.TradeAvailable)
	}
}

func TestReaderTradeAvailableAbsentDefaultsToZero(t *testing.T) {
	rm := readOne(t, karmaAbsentTestXML, 1002358)
	if rm.TradeAvailable != 0 {
		t.Fatalf("TradeAvailable = %d, want 0", rm.TradeAvailable)
	}
}
```

> **If `Read`'s signature or the surrounding helper names in this package differ** (each atlas-data reader has its own shape — check the package's existing `reader_test.go` for the exact `Read(l)(provider)` call and the `Identity` helper), adapt the fixture and the `readOne` helper to match that package's existing test idiom. The three assertions — present, absent-defaults-to-zero, real id — are what matters.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-data/atlas.com/data && go test ./equipment/... -run 'TradeAvailable' -v`
Expected: FAIL — `rm.TradeAvailable undefined`.

- [ ] **Step 3: Add the field and the parse to the equipment reader**

In `services/atlas-data/atlas.com/data/equipment/rest.go`, beside `TradeBlock bool \`json:"tradeBlock"\`` (line 44):

```go
	// TradeAvailable is WZ info/tradeAvailable — the item's APPLICABLE KARMA
	// TYPE, read by CItemInfo::GetAppliableKarmaType (gms_v95 @0x5C09F0) off
	// BUNDLEITEM/EQUIPITEM+0x14. It is an INT, not a bool: the v87+ client tests
	// it for EQUALITY against the scissors' own info/karma type, so two scissors
	// variants gate two different target sets. Absent => 0 => not karma-applicable.
	TradeAvailable int32 `json:"tradeAvailable"`
```

In `services/atlas-data/atlas.com/data/equipment/reader.go`, in the struct literal beside `TradeBlock: info.GetBool("tradeBlock", false),` (line 114):

```go
			TradeAvailable: info.GetIntegerWithDefault("tradeAvailable", 0),
```

- [ ] **Step 4: Run the equipment test to verify it passes**

Run: `cd services/atlas-data/atlas.com/data && go test ./equipment/... -run 'TradeAvailable' -v`
Expected: PASS.

- [ ] **Step 5: Repeat for consumable, setup and etc**

For each of `consumable`, `setup`, `etc`:

- In `<pkg>/rest.go`, beside the existing `TradeBlock bool` field, add the same `TradeAvailable int32 \`json:"tradeAvailable"\`` field with the same doc comment (abbreviate to one line after the first: `// TradeAvailable is WZ info/tradeAvailable; see equipment/rest.go for the derivation.`).
- In `<pkg>/reader.go`, beside the existing `m.TradeBlock = i.GetBool("tradeBlock", false)` line, add:

```go
			m.TradeAvailable = i.GetIntegerWithDefault("tradeAvailable", 0)
```

- Create `<pkg>/reader_karma_test.go` mirroring Step 1, using an id in that package's range (`consumable`: `2280000` — the v83 corpus's other `tradeAvailable` carrier is `Item.wz/Consume/0228.img`, the mastery books; `setup`: `3010000`; `etc`: `4000000`) and that package's `Read`/`Identity` idiom. Each file keeps both cases: present → the fixture's value, absent → `0`.

- [ ] **Step 6: Do the cash reader, which also carries the scissors' own `karma`**

In `services/atlas-data/atlas.com/data/cash/rest.go`, beside `TradeBlock` (line 53):

```go
	// TradeAvailable is WZ info/tradeAvailable; see equipment/rest.go.
	TradeAvailable int32 `json:"tradeAvailable"`
	// Karma is WZ info/karma — the SCISSORS' OWN karma type, read by
	// CItemInfo::RegisterKarmaScissorsItem (gms_v95 @0x5A1120) into
	// KARMASCISSORSITEM.nKarmaType. Parsed for every cash item and left 0 for
	// non-scissors: absence already yields 0, so no classification filter is
	// needed. The v83 corpus carries no `karma` node at all, which is why the
	// eligibility predicate treats 0 as "untyped scissors" (design §3.2).
	Karma int32 `json:"karma"`
```

In `services/atlas-data/atlas.com/data/cash/reader.go`, beside line 84:

```go
			m.TradeAvailable = i.GetIntegerWithDefault("tradeAvailable", 0)
			m.Karma = i.GetIntegerWithDefault("karma", 0)
```

Create `services/atlas-data/atlas.com/data/cash/reader_karma_test.go` with three cases — a fixture for `05520000` carrying `<int name="karma" value="1"/>` asserting `Karma == 1`; a fixture for `05520000` carrying only `<int name="cash" value="1"/>` (the actual shipped v83 node) asserting `Karma == 0`; and a `tradeAvailable` present/absent pair as in Step 1.

- [ ] **Step 7: Run the full atlas-data suite**

Run: `cd services/atlas-data/atlas.com/data && go test -race ./... && go vet ./... && go build ./...`
Expected: all PASS. In particular the pre-existing `reader_test.go` item-count assertions must be unchanged.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-data/atlas.com/data/
git commit -m "feat(task-223): parse WZ info/tradeAvailable across five item readers and info/karma on cash"
```

---

## Task 4: Fix the `KarmaUsed` / `SetKarmaUsed` round-trip in seven services

**Files:**
- Modify: `services/atlas-inventory/atlas.com/inventory/asset/model.go:84`, `asset/builder.go:147-153`
- Modify: `services/atlas-channel/atlas.com/channel/asset/model.go:92`, `asset/builder.go:180-186`
- Modify: `services/atlas-login/atlas.com/login/inventory/compartment/asset/model.go:82`, `.../builder.go:144-150`
- Modify: `services/atlas-consumables/atlas.com/consumables/asset/model.go:82`, `asset/builder.go:144-150`
- Modify: `services/atlas-storage/atlas.com/storage/asset/model.go:82`, `asset/builder.go:142-148`
- Modify: `services/atlas-query-aggregator/atlas.com/query-aggregator/asset/model.go:82`, `asset/builder.go:144-150`
- Modify: `services/atlas-cashshop/atlas.com/cashshop/asset/reference_data.go:54,373`
- Test: one `asset/karma_roundtrip_test.go` per service (seven files)

**Interfaces:**
- Consumes: `asset.KarmaFlagFor` (Task 1).
- Produces: no signature changes. `SetKarmaUsed(true)` followed by `KarmaUsed()` returns `true` for both an equip and a bundle in all seven services.

**Background.** `SetKarmaUsed(true)` writes `FlagKarmaEquip` (0x10) while `KarmaUsed()` reads `FlagKarmaUse` (0x02): a set never reads back, for **any** asset. Six of the seven models carry the template id already (`model.go:58-59` in each), so both sides route through `KarmaFlagFor`. `atlas-cashshop` is the exception: `EquipableReferenceData` and `CashEquipableReferenceData` carry no template id, but are equip-class *by type*, so the correct bit is unconditionally `FlagKarmaEquip` — their setters are already right and only the getters change.

- [ ] **Step 1: Write the failing test for atlas-inventory**

Create `services/atlas-inventory/atlas.com/inventory/asset/karma_roundtrip_test.go`:

```go
package asset

import (
	"testing"

	af "github.com/Chronicle20/atlas/libs/atlas-constants/asset"
)

// TestKarmaUsedRoundTrip is the FR-4.4 regression guard. Before task-223,
// SetKarmaUsed wrote FlagKarmaEquip (0x10) while KarmaUsed read FlagKarmaUse
// (0x02), so a set NEVER read back for any asset.
func TestKarmaUsedRoundTrip(t *testing.T) {
	testCases := []struct {
		name       string
		templateId uint32
	}{
		{"equip", 1002357},
		{"bundle", 2280000},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := NewBuilder(1, 1, tc.templateId, 1).SetKarmaUsed(true).Build()
			if !m.KarmaUsed() {
				t.Fatalf("KarmaUsed() = false after SetKarmaUsed(true) for template %d", tc.templateId)
			}
			cleared := Clone(m).SetKarmaUsed(false).Build()
			if cleared.KarmaUsed() {
				t.Fatalf("KarmaUsed() = true after SetKarmaUsed(false) for template %d", tc.templateId)
			}
		})
	}
}

// TestKarmaUsedLeavesSpikesAlone is the FR-4.5 guard: 0x02 is FlagSpikes on an
// EQUIP, so a karma mark on an equip must not render spikes, and clearing karma
// on an equip must not silently clear a genuine spikes flag.
func TestKarmaUsedLeavesSpikesAlone(t *testing.T) {
	spiked := NewBuilder(1, 1, 1002357, 1).AddFlag(af.FlagSpikes).SetKarmaUsed(true).Build()
	if !spiked.Spikes() {
		t.Fatal("Spikes() = false after karma-marking a spiked equip")
	}
	if !spiked.KarmaUsed() {
		t.Fatal("KarmaUsed() = false on a spiked equip")
	}

	plain := NewBuilder(1, 1, 1002357, 1).SetKarmaUsed(true).Build()
	if plain.Spikes() {
		t.Fatal("Spikes() = true after karma-marking an unspiked equip; the wrong bit was written")
	}
}
```

> Adjust `NewBuilder`'s argument list to the package's actual constructor signature — check `asset/builder.go`. The three assertions are what matters, not the construction ceremony.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-inventory/atlas.com/inventory && go test ./asset/... -run 'TestKarmaUsed' -v`
Expected: FAIL — `KarmaUsed() = false after SetKarmaUsed(true)` for both cases.

- [ ] **Step 3: Fix the getter and the setter in atlas-inventory**

`services/atlas-inventory/atlas.com/inventory/asset/model.go:84` becomes:

```go
// KarmaUsed reports whether this asset carries the one-free-trade karma mark.
// The bit is SLOT-CLASS DEPENDENT (0x10 equip / 0x02 bundle), so it is resolved
// from the template id rather than named — see libs/atlas-constants/asset.KarmaFlagFor.
func (m Model) KarmaUsed() bool {
	f, ok := af.KarmaFlagFor(m.templateId)
	if !ok {
		return false
	}
	return af.HasFlag(m.flag, f)
}
```

`services/atlas-inventory/atlas.com/inventory/asset/builder.go:147` becomes:

```go
// SetKarmaUsed sets or clears the slot-class-correct karma bit, touching NO
// other bit. On an equip the bundle karma bit (0x02) is FlagSpikes, so a
// hand-picked constant here would render spikes on every karma'd equip.
func (b *ModelBuilder) SetKarmaUsed(v bool) *ModelBuilder {
	f, ok := af.KarmaFlagFor(b.templateId)
	if !ok {
		return b
	}
	if v {
		b.flag = af.SetFlag(b.flag, f)
	} else {
		b.flag = af.ClearFlag(b.flag, f)
	}
	return b
}
```

> If the builder's template-id field is named differently (`b.templateId` is the expected name — verify against the struct), use the actual field.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-inventory/atlas.com/inventory && go test -race ./asset/...`
Expected: PASS.

- [ ] **Step 5: Repeat identically for five more services**

For `atlas-channel`, `atlas-login`, `atlas-consumables`, `atlas-storage`, `atlas-query-aggregator`: apply the same getter and setter replacements at the file:line pairs listed under **Files** above, and copy `karma_roundtrip_test.go` into each service's `asset` package (adjusting only the package clause and the builder constructor call).

Run per service: `cd services/<svc>/atlas.com/<name> && go test -race ./asset/...`

- [ ] **Step 6: Fix atlas-cashshop's two getters**

`services/atlas-cashshop/atlas.com/cashshop/asset/reference_data.go:54`:

```go
// IsKarmaUsed reads the EQUIP karma bit (0x10) unconditionally. Unlike the six
// asset models, EquipableReferenceData carries no template id — it is the
// equip-shaped reference block hanging off an asset that holds the id — but it
// is equip-class BY TYPE, so the bit is fixed rather than resolved. The setter
// (:261) already writes FlagKarmaEquip; before task-223 this getter read
// FlagKarmaUse (0x02, the BUNDLE bit) and the pair never round-tripped.
func (e EquipableReferenceData) IsKarmaUsed() bool { return af.HasFlag(e.flag, af.FlagKarmaEquip) }
```

`services/atlas-cashshop/atlas.com/cashshop/asset/reference_data.go:373` — the same change with the same comment abbreviated to `// See EquipableReferenceData.IsKarmaUsed: equip-class by type, so the bit is fixed.`:

```go
func (e CashEquipableReferenceData) IsKarmaUsed() bool { return af.HasFlag(e.flag, af.FlagKarmaEquip) }
```

Create `services/atlas-cashshop/atlas.com/cashshop/asset/karma_roundtrip_test.go` with an equip-only round trip for both types (build with `SetKarmaUsed(true)`, assert `IsKarmaUsed()`; build with `SetKarmaUsed(false)`, assert `!IsKarmaUsed()`), matching each builder's existing constructor idiom.

- [ ] **Step 7: Verify all seven services**

Run, for each of the seven: `cd services/<svc>/atlas.com/<name> && go test -race ./... && go vet ./... && go build ./...`
Expected: all clean.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-inventory services/atlas-channel services/atlas-login services/atlas-consumables services/atlas-storage services/atlas-query-aggregator services/atlas-cashshop
git commit -m "fix(task-223): karma flag getter/setter round-trip across seven services"
```

---

## Task 5: The `ApplyAssetKarma` saga contract

**Files:**
- Modify: `libs/atlas-saga/model.go:39-45` (Type block), `model.go:219-222` (Action block)
- Modify: `libs/atlas-saga/payloads.go:1088-1100` (beside `ApplyAssetLockPayload`)
- Modify: `libs/atlas-saga/unmarshal.go:582-586` (beside the `ApplyAssetLock` case)
- Test: `libs/atlas-saga/karma_payload_test.go` (create)

**Interfaces:**
- Consumes: nothing.
- Produces:
  - `saga.KarmaScissorsUse Type = "karma_scissors_use"`
  - `saga.ApplyAssetKarma Action = "apply_asset_karma"`
  - `saga.ApplyAssetKarmaPayload{CharacterId uint32; InventoryType byte; Slot int16; ScissorsKarma int32}`

- [ ] **Step 1: Write the failing test**

Create `libs/atlas-saga/karma_payload_test.go`:

```go
package saga

import (
	"encoding/json"
	"testing"
)

// TestApplyAssetKarmaPayloadUnmarshal proves the action's arm exists in
// Step.UnmarshalJSON: without it the payload decodes to nil and the orchestrator
// fails the step with "invalid payload".
func TestApplyAssetKarmaPayloadUnmarshal(t *testing.T) {
	raw := []byte(`{
		"stepId": "apply_asset_karma",
		"status": "pending",
		"action": "apply_asset_karma",
		"payload": {"characterId": 42, "inventoryType": 1, "slot": 3, "scissorsKarma": 2}
	}`)

	var st Step[any]
	if err := json.Unmarshal(raw, &st); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	p, ok := st.Payload.(ApplyAssetKarmaPayload)
	if !ok {
		t.Fatalf("payload type = %T, want ApplyAssetKarmaPayload", st.Payload)
	}
	if p.CharacterId != 42 || p.InventoryType != 1 || p.Slot != 3 {
		t.Fatalf("payload = %+v, want {42 1 3 ...}", p)
	}
	// ScissorsKarma is what lets atlas-inventory re-run the EQUALITY half of the
	// eligibility predicate. Dropping it would silently weaken the v87+ model to
	// the v83 non-zero model at the owning service.
	if p.ScissorsKarma != 2 {
		t.Fatalf("ScissorsKarma = %d, want 2", p.ScissorsKarma)
	}
}
```

> Match the `Step[any]` field access idiom used by the package's existing unmarshal tests (`st.Payload` vs `st.Payload()`); check `libs/atlas-saga/unmarshal.go` and any sibling test.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-saga && go test ./... -run 'ApplyAssetKarma' -v`
Expected: FAIL — `undefined: ApplyAssetKarmaPayload`.

- [ ] **Step 3: Add the type, action, payload and unmarshal arm**

In `libs/atlas-saga/model.go`, in the saga `Type` block beside `SealingLockUse` (line 40):

```go
	KarmaScissorsUse    Type = "karma_scissors_use"
```

In the same file's `Action` block, beside `ApplyAssetLock` (line 220):

```go
	ApplyAssetKarma Action = "apply_asset_karma"
```

In `libs/atlas-saga/payloads.go`, beside `ApplyAssetLockPayload` (line 1088):

```go
// ApplyAssetKarmaPayload represents the payload required to apply (or, on the
// compensation path, clear) the one-free-trade karma mark on an asset in a
// specific inventory slot.
//
// There is no Clear field here and no near-duplicate ClearAssetKarma action:
// the saga surface stays one entry wide, and compensation dispatches the
// inventory command directly with its own clear discriminator (see
// atlas-saga-orchestrator's DispatchCashItemUseRollbacks). Keeping the
// acceptance table one entry wide is the point.
type ApplyAssetKarmaPayload struct {
	CharacterId   uint32 `json:"characterId"`   // CharacterId associated with the action
	InventoryType byte   `json:"inventoryType"` // Type of inventory (1=equip, 2=use, 3=setup, 4=etc, 5=cash)
	Slot          int16  `json:"slot"`          // Slot of the asset to mark (must be >= 0; equipped items are refused upstream)
	// ScissorsKarma is the SCISSORS' OWN WZ info/karma type, carried so the
	// owning service can re-run the EQUALITY half of the eligibility predicate
	// without knowing which scissors were used. 0 means untyped scissors (the
	// gms_v83 model), under which the predicate reduces to "is the target
	// karma-applicable at all". Omitting it would silently weaken the v87+
	// equality model to the v83 non-zero model at atlas-inventory, which is the
	// authority — so it must travel with the action.
	ScissorsKarma int32 `json:"scissorsKarma"`
}
```

In `libs/atlas-saga/unmarshal.go`, beside the `ApplyAssetLock` case (line 582):

```go
	case ApplyAssetKarma:
		var payload ApplyAssetKarmaPayload
		if err := json.Unmarshal(aux.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.Action, err)
		}
		s.Payload = any(payload).(T)
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd libs/atlas-saga && go test -race ./... && go vet ./...`
Expected: PASS, clean.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-saga/
git commit -m "feat(task-223): ApplyAssetKarma saga action, payload and karma_scissors_use type"
```

---

## Task 6: atlas-inventory's per-compartment tradeability client

**Files:**
- Create: `services/atlas-inventory/atlas.com/inventory/data/tradeability/rest.go`
- Create: `services/atlas-inventory/atlas.com/inventory/data/tradeability/requests.go`
- Create: `services/atlas-inventory/atlas.com/inventory/data/tradeability/processor.go`
- Create: `services/atlas-inventory/atlas.com/inventory/data/tradeability/mock/processor.go`
- Create: `services/atlas-inventory/atlas.com/inventory/data/tradeability/processor_test.go`

**Interfaces:**
- Consumes: atlas-data's five item resources (`statistics`, `consumables`, `setups`, `etcs`, `cash_items`) and their new `tradeAvailable` field (Task 3).
- Produces:
  - `tradeability.Model` with `TradeAvailable() int32` and `TradeBlock() bool`
  - `tradeability.Processor` interface with `Get(inventoryType inventory.Type, templateId item.Id) (Model, error)` and `ByIdProvider(inventoryType inventory.Type, templateId item.Id) model.Provider[Model]`
  - `tradeability/mock.ProcessorMock` with a `GetFunc` field

**Why a new package:** the karma gates need `tradeAvailable` **and** `tradeBlock` for *any* of the five compartments. atlas-inventory's existing `data/{consumable,equipment,etc,setup}` clients cover four compartments, carry neither field, and have no cash client. This package is modelled directly on `services/atlas-trades/atlas.com/trades/data/item/` — five wire models dispatched on the compartment — which is the established shape for exactly this problem.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-inventory/atlas.com/inventory/data/tradeability/processor_test.go`:

```go
package tradeability

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
)

// TestExtractCarriesBothFields proves the shared Extract does not silently drop
// one of the two fields the karma gates need.
func TestExtractCarriesBothFields(t *testing.T) {
	m, err := extract(EquipmentRestModel{Id: 1002357, TradeBlock: true, TradeAvailable: 1})
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if !m.TradeBlock() {
		t.Fatal("TradeBlock() = false, want true")
	}
	if m.TradeAvailable() != 1 {
		t.Fatalf("TradeAvailable() = %d, want 1", m.TradeAvailable())
	}
}

// TestByIdProviderRejectsUnknownCompartment: an unknown compartment must be an
// error the caller refuses on, never a zero-valued permissive default.
func TestByIdProviderRejectsUnknownCompartment(t *testing.T) {
	p := &ProcessorImpl{}
	if _, err := p.ByIdProvider(inventory.Type(99), 1002357)(); err == nil {
		t.Fatal("ByIdProvider(99) returned no error; an unknown compartment must refuse")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-inventory/atlas.com/inventory && go test ./data/tradeability/... -v`
Expected: FAIL — package does not exist.

- [ ] **Step 3: Write the package**

`rest.go` — five wire models plus the shared extract:

```go
// Package tradeability is atlas-inventory's read client for the two atlas-data
// item properties the karma gates need: WZ info/tradeBlock (is the item
// untradeable by data?) and WZ info/tradeAvailable (the item's applicable karma
// type). atlas-data exposes them on FIVE separate resources, one per inventory
// compartment, at five paths with five JSON:API type names — there is no
// unified item resource — so this reader carries one RestModel per resource and
// dispatches on the compartment the asset came from. Modelled on
// services/atlas-trades/atlas.com/trades/data/item.
package tradeability

import (
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

type Model struct {
	tradeBlock     bool
	tradeAvailable int32
}

func (m Model) TradeBlock() bool      { return m.tradeBlock }
func (m Model) TradeAvailable() int32 { return m.tradeAvailable }

type EquipmentRestModel struct {
	Id             item.Id `json:"-"`
	TradeBlock     bool    `json:"tradeBlock"`
	TradeAvailable int32   `json:"tradeAvailable"`
}

func (r EquipmentRestModel) GetName() string                              { return "statistics" }
func (r EquipmentRestModel) GetID() string                                { return strconv.FormatUint(uint64(r.Id), 10) }
func (r *EquipmentRestModel) SetID(s string) error                        { return setItemId(&r.Id, s) }
func (r *EquipmentRestModel) SetToOneReferenceID(_ string, _ string) error { return nil }
func (r *EquipmentRestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }
func (r EquipmentRestModel) fields() (bool, int32)                        { return r.TradeBlock, r.TradeAvailable }

type ConsumableRestModel struct {
	Id             item.Id `json:"-"`
	TradeBlock     bool    `json:"tradeBlock"`
	TradeAvailable int32   `json:"tradeAvailable"`
}

func (r ConsumableRestModel) GetName() string                              { return "consumables" }
func (r ConsumableRestModel) GetID() string                                { return strconv.FormatUint(uint64(r.Id), 10) }
func (r *ConsumableRestModel) SetID(s string) error                        { return setItemId(&r.Id, s) }
func (r *ConsumableRestModel) SetToOneReferenceID(_ string, _ string) error { return nil }
func (r *ConsumableRestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }
func (r ConsumableRestModel) fields() (bool, int32)                        { return r.TradeBlock, r.TradeAvailable }

type SetupRestModel struct {
	Id             item.Id `json:"-"`
	TradeBlock     bool    `json:"tradeBlock"`
	TradeAvailable int32   `json:"tradeAvailable"`
}

func (r SetupRestModel) GetName() string                              { return "setups" }
func (r SetupRestModel) GetID() string                                { return strconv.FormatUint(uint64(r.Id), 10) }
func (r *SetupRestModel) SetID(s string) error                        { return setItemId(&r.Id, s) }
func (r *SetupRestModel) SetToOneReferenceID(_ string, _ string) error { return nil }
func (r *SetupRestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }
func (r SetupRestModel) fields() (bool, int32)                        { return r.TradeBlock, r.TradeAvailable }

type EtcRestModel struct {
	Id             item.Id `json:"-"`
	TradeBlock     bool    `json:"tradeBlock"`
	TradeAvailable int32   `json:"tradeAvailable"`
}

func (r EtcRestModel) GetName() string                              { return "etcs" }
func (r EtcRestModel) GetID() string                                { return strconv.FormatUint(uint64(r.Id), 10) }
func (r *EtcRestModel) SetID(s string) error                        { return setItemId(&r.Id, s) }
func (r *EtcRestModel) SetToOneReferenceID(_ string, _ string) error { return nil }
func (r *EtcRestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }
func (r EtcRestModel) fields() (bool, int32)                        { return r.TradeBlock, r.TradeAvailable }

type CashRestModel struct {
	Id             item.Id `json:"-"`
	TradeBlock     bool    `json:"tradeBlock"`
	TradeAvailable int32   `json:"tradeAvailable"`
}

func (r CashRestModel) GetName() string                              { return "cash_items" }
func (r CashRestModel) GetID() string                                { return strconv.FormatUint(uint64(r.Id), 10) }
func (r *CashRestModel) SetID(s string) error                        { return setItemId(&r.Id, s) }
func (r *CashRestModel) SetToOneReferenceID(_ string, _ string) error { return nil }
func (r *CashRestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }
func (r CashRestModel) fields() (bool, int32)                        { return r.TradeBlock, r.TradeAvailable }

func setItemId(target *item.Id, strId string) error {
	id, err := strconv.ParseUint(strId, 10, 32)
	if err != nil {
		return err
	}
	*target = item.Id(id)
	return nil
}

// extract is the shared Extract for all five wire models.
func extract[R interface{ fields() (bool, int32) }](rm R) (Model, error) {
	tb, ta := rm.fields()
	return Model{tradeBlock: tb, tradeAvailable: ta}, nil
}
```

`requests.go`:

```go
package tradeability

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

func getBaseRequest() string { return requests.RootUrl("DATA") }

func requestEquipment(id item.Id) requests.Request[EquipmentRestModel] {
	return requests.GetRequest[EquipmentRestModel](fmt.Sprintf(getBaseRequest()+"data/equipment/%d", id))
}

func requestConsumable(id item.Id) requests.Request[ConsumableRestModel] {
	return requests.GetRequest[ConsumableRestModel](fmt.Sprintf(getBaseRequest()+"data/consumables/%d", id))
}

func requestSetup(id item.Id) requests.Request[SetupRestModel] {
	return requests.GetRequest[SetupRestModel](fmt.Sprintf(getBaseRequest()+"data/setups/%d", id))
}

func requestEtc(id item.Id) requests.Request[EtcRestModel] {
	return requests.GetRequest[EtcRestModel](fmt.Sprintf(getBaseRequest()+"data/etcs/%d", id))
}

func requestCash(id item.Id) requests.Request[CashRestModel] {
	return requests.GetRequest[CashRestModel](fmt.Sprintf(getBaseRequest()+"data/cash/items/%d", id))
}
```

> Confirm the five path templates against `services/atlas-trades/atlas.com/trades/data/item/requests.go` (that file already resolves all five correctly) and copy them verbatim if they differ from the above.

`processor.go`:

```go
package tradeability

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// Processor answers the two WZ questions the karma gates ask of a target item.
// An error means the LOOKUP FAILED; every caller must treat that as a refusal,
// never as a permissive default.
type Processor interface {
	ByIdProvider(inventoryType inventory.Type, templateId item.Id) model.Provider[Model]
	Get(inventoryType inventory.Type, templateId item.Id) (Model, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) ByIdProvider(inventoryType inventory.Type, templateId item.Id) model.Provider[Model] {
	switch inventoryType {
	case inventory.TypeValueEquip:
		return requests.Provider[EquipmentRestModel, Model](p.l, p.ctx)(requestEquipment(templateId), extract)
	case inventory.TypeValueUse:
		return requests.Provider[ConsumableRestModel, Model](p.l, p.ctx)(requestConsumable(templateId), extract)
	case inventory.TypeValueSetup:
		return requests.Provider[SetupRestModel, Model](p.l, p.ctx)(requestSetup(templateId), extract)
	case inventory.TypeValueETC:
		return requests.Provider[EtcRestModel, Model](p.l, p.ctx)(requestEtc(templateId), extract)
	case inventory.TypeValueCash:
		return requests.Provider[CashRestModel, Model](p.l, p.ctx)(requestCash(templateId), extract)
	default:
		return model.ErrorProvider[Model](fmt.Errorf("tradeability: no atlas-data resource for inventory type [%d]", inventoryType))
	}
}

func (p *ProcessorImpl) Get(inventoryType inventory.Type, templateId item.Id) (Model, error) {
	return p.ByIdProvider(inventoryType, templateId)()
}
```

`mock/processor.go`:

```go
package mock

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"

	"atlas-inventory/data/tradeability"
)

type ProcessorMock struct {
	ByIdProviderFunc func(inventoryType inventory.Type, templateId item.Id) model.Provider[tradeability.Model]
	GetFunc          func(inventoryType inventory.Type, templateId item.Id) (tradeability.Model, error)
}

var _ tradeability.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) ByIdProvider(inventoryType inventory.Type, templateId item.Id) model.Provider[tradeability.Model] {
	if m.ByIdProviderFunc != nil {
		return m.ByIdProviderFunc(inventoryType, templateId)
	}
	return model.FixedProvider(tradeability.Model{})
}

func (m *ProcessorMock) Get(inventoryType inventory.Type, templateId item.Id) (tradeability.Model, error) {
	if m.GetFunc != nil {
		return m.GetFunc(inventoryType, templateId)
	}
	return tradeability.Model{}, nil
}
```

> The mock needs a way to build a non-zero `tradeability.Model` from outside the package. Add an exported test constructor to `rest.go` — `func NewModel(tradeBlock bool, tradeAvailable int32) Model { return Model{tradeBlock: tradeBlock, tradeAvailable: tradeAvailable} }` — and use it from the compartment tests in Task 7.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-inventory/atlas.com/inventory && go test -race ./data/tradeability/... -v && go build ./...`
Expected: PASS, build clean.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-inventory/atlas.com/inventory/data/tradeability/
git commit -m "feat(task-223): atlas-inventory per-compartment tradeAvailable/tradeBlock client"
```

---

## Task 7: atlas-inventory applies (and clears) the karma mark

**Files:**
- Modify: `services/atlas-inventory/atlas.com/inventory/asset/processor.go:53-54` (interface), after `:342` (impl)
- Modify: `services/atlas-inventory/atlas.com/inventory/asset/mock/processor.go:36-37,213-232`
- Modify: `services/atlas-inventory/atlas.com/inventory/compartment/processor.go:98-99` (interface), after `:1077` (impl)
- Modify: `services/atlas-inventory/atlas.com/inventory/compartment/mock/processor.go:55-56,365-380`
- Test: `services/atlas-inventory/atlas.com/inventory/compartment/karma_processor_test.go` (create)

**Interfaces:**
- Consumes: `asset.KarmaFlagFor`, `asset.KarmaEligible` (Task 1); `tradeability.Processor` (Task 6); existing `updateFlag`-style persistence and `UpdatedEventStatusProvider`.
- Produces:
  - `asset.Processor.ApplyKarma(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32) func(a Model, scissorsKarma int32, d tradeability.Model) error`
  - `asset.Processor.ClearKarma(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32) func(a Model) error`
  - `compartment.Processor.ApplyAssetKarma(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, slot int16, scissorsKarma int32, clear bool) error`
  - `compartment.Processor.ApplyAssetKarmaAndEmit(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, slot int16, scissorsKarma int32, clear bool) error`

- [ ] **Step 1: Write the failing test**

Create `services/atlas-inventory/atlas.com/inventory/compartment/karma_processor_test.go`. Model the fixture setup on the existing `TestApplyAssetLock` (`compartment/processor_test.go:1150-1190`) — same DB bootstrap, same asset creation, same `mb` construction.

```go
package compartment

import (
	"testing"

	"github.com/google/uuid"

	af "github.com/Chronicle20/atlas/libs/atlas-constants/asset"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
)

// TestApplyAssetKarmaMarksAnUntradeableEquip is the happy path: an untradeable,
// karma-applicable equip gains the EQUIP karma bit (0x10) and nothing else.
func TestApplyAssetKarmaMarksAnUntradeableEquip(t *testing.T) {
	// ... bootstrap as TestApplyAssetLock does, with:
	//   templateId 1002357 (Zakum Helmet), flag = uint16(af.FlagUntradeable), slot 3
	//   tradeability mock: Get -> NewModel(true /*tradeBlock*/, 1 /*tradeAvailable*/)
	//   scissorsKarma = 0 (v83-era untyped scissors)
	if err := cp.ApplyAssetKarma(mb)(uuid.New(), characterId, inventory.TypeValueEquip, slot, 0, false); err != nil {
		t.Fatalf("ApplyAssetKarma returned unexpected error: %v", err)
	}
	a := reloadAsset(t)
	if !af.HasFlag(a.Flag(), af.FlagKarmaEquip) {
		t.Fatal("expected the EQUIP karma bit (0x10) to be set")
	}
	if af.HasFlag(a.Flag(), af.FlagKarmaUse) && !af.HasFlag(a.Flag(), af.FlagSpikes) {
		t.Fatal("the BUNDLE karma bit (0x02 = FlagSpikes on an equip) was written; wrong bit")
	}
	if !af.HasFlag(a.Flag(), af.FlagUntradeable) {
		t.Fatal("FlagUntradeable was disturbed")
	}
}

// TestApplyAssetKarmaRefusesIneligibleTarget is the FR-6.4 re-assertion: the
// channel's gates are advisory across a service boundary.
func TestApplyAssetKarmaRefusesIneligibleTarget(t *testing.T) {
	// tradeability mock: Get -> NewModel(true, 0)  // tradeAvailable == 0
	if err := cp.ApplyAssetKarma(mb)(uuid.New(), characterId, inventory.TypeValueEquip, slot, 0, false); err == nil {
		t.Fatal("expected ApplyAssetKarma to refuse a target with tradeAvailable == 0")
	}
	if af.HasFlag(reloadAsset(t).Flag(), af.FlagKarmaEquip) {
		t.Fatal("a refused apply still mutated the flag")
	}
}

// TestApplyAssetKarmaRefusesAlreadyMarked is FR-6.7: a redelivered command must
// not silently consume a second scissors against an already-marked item.
func TestApplyAssetKarmaRefusesAlreadyMarked(t *testing.T) {
	// asset flag = uint16(af.FlagUntradeable | af.FlagKarmaEquip)
	if err := cp.ApplyAssetKarma(mb)(uuid.New(), characterId, inventory.TypeValueEquip, slot, 0, false); err == nil {
		t.Fatal("expected ApplyAssetKarma to refuse an already-marked asset")
	}
}

// TestApplyAssetKarmaRefusesLockedTarget mirrors client gate 1
// (GW_ItemSlotEquip::IsProtectedItem, gms_v83 @0x4E9506).
func TestApplyAssetKarmaRefusesLockedTarget(t *testing.T) {
	// asset flag = uint16(af.FlagUntradeable | af.FlagLock)
	if err := cp.ApplyAssetKarma(mb)(uuid.New(), characterId, inventory.TypeValueEquip, slot, 0, false); err == nil {
		t.Fatal("expected ApplyAssetKarma to refuse a Sealing-Lock'd asset")
	}
}

// TestApplyAssetKarmaRefusesAlreadyTradeableTarget is gate 4: karma exists to
// unlock an UNTRADEABLE item; marking a tradeable one consumes the scissors for
// nothing.
func TestApplyAssetKarmaRefusesAlreadyTradeableTarget(t *testing.T) {
	// asset flag = 0; tradeability mock: Get -> NewModel(false, 1)
	if err := cp.ApplyAssetKarma(mb)(uuid.New(), characterId, inventory.TypeValueEquip, slot, 0, false); err == nil {
		t.Fatal("expected ApplyAssetKarma to refuse an already-tradeable asset")
	}
}

// TestApplyAssetKarmaRefusesPet is OQ-5: the pet karma bit aliases FlagLock.
func TestApplyAssetKarmaRefusesPet(t *testing.T) {
	// templateId 5000000, in the CASH compartment
	if err := cp.ApplyAssetKarma(mb)(uuid.New(), characterId, inventory.TypeValueCash, slot, 0, false); err == nil {
		t.Fatal("expected ApplyAssetKarma to refuse a pet-class target")
	}
}

// TestApplyAssetKarmaRefusesUnreadableItemData: a failed lookup is a refusal,
// never a permissive default.
func TestApplyAssetKarmaRefusesUnreadableItemData(t *testing.T) {
	// tradeability mock: Get -> (Model{}, errors.New("boom"))
	if err := cp.ApplyAssetKarma(mb)(uuid.New(), characterId, inventory.TypeValueEquip, slot, 0, false); err == nil {
		t.Fatal("expected ApplyAssetKarma to refuse when item data is unreadable")
	}
}

// TestApplyAssetKarmaClearRemovesTheBit is the compensation path.
func TestApplyAssetKarmaClearRemovesTheBit(t *testing.T) {
	// asset flag = uint16(af.FlagUntradeable | af.FlagKarmaEquip)
	if err := cp.ApplyAssetKarma(mb)(uuid.New(), characterId, inventory.TypeValueEquip, slot, 0, true); err != nil {
		t.Fatalf("clear returned unexpected error: %v", err)
	}
	a := reloadAsset(t)
	if af.HasFlag(a.Flag(), af.FlagKarmaEquip) {
		t.Fatal("expected the karma bit to be cleared")
	}
	if !af.HasFlag(a.Flag(), af.FlagUntradeable) {
		t.Fatal("clearing karma disturbed FlagUntradeable")
	}
}
```

> Fill in the elided bootstrap from `TestApplyAssetLock`. `reloadAsset` is a local helper that re-reads the asset by slot through the same processor. `clear = true` skips the gates entirely — a compensation must always be able to undo.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-inventory/atlas.com/inventory && go test ./compartment/... -run 'ApplyAssetKarma' -v`
Expected: FAIL — `cp.ApplyAssetKarma undefined`.

- [ ] **Step 3: Add the asset-layer mutators**

In `services/atlas-inventory/atlas.com/inventory/asset/processor.go`, add to the `Processor` interface beside `ApplyLock`/`ClearLock` (lines 53-54):

```go
	ApplyKarma(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32) func(a Model, scissorsKarma int32, d tradeability.Model) error
	ClearKarma(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32) func(a Model) error
```

Add the implementations after `ClearLock` (after line 357), modelled on `ApplyLock`:

```go
// ApplyKarma sets the slot-class-correct karma mark on an asset in place,
// emitting the existing UPDATED status event. atlas-inventory owns the asset, so
// it re-asserts every gate the channel arm applied — the channel's checks are
// advisory across a service boundary and a crafted command reaches this consumer
// regardless (FR-6.4).
//
// Gates, in the client's own order (CUIKarmaDlg::PutItem, gms_v95 @0x7D7BA0):
//
//	0c pet class      -> KarmaFlagFor reports no bit; the pet karma bit is 0x01,
//	                     which is FlagLock in Atlas's shared flag column.
//	1  FlagLock       -> GW_ItemSlot*::IsProtectedItem
//	2  eligibility    -> KarmaEligible(scissorsKarma, d.TradeAvailable())
//	3  already marked -> GW_ItemSlot*::IsPossibleTradingItem. THIS IS THE
//	                     IDEMPOTENCY GUARANTEE (FR-6.7): a redelivered command
//	                     finds the bit set and refuses rather than letting a
//	                     second scissors be consumed against a marked item.
//	4  already tradeable -> server-only; karma exists to unlock an UNTRADEABLE
//	                     item, and "untradeable" here is the same pair of
//	                     conditions atlas-trades enforces.
func (p *ProcessorImpl) ApplyKarma(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32) func(a Model, scissorsKarma int32, d tradeability.Model) error {
	return func(transactionId uuid.UUID, characterId uint32) func(a Model, scissorsKarma int32, d tradeability.Model) error {
		return func(a Model, scissorsKarma int32, d tradeability.Model) error {
			f, ok := af.KarmaFlagFor(a.TemplateId())
			if !ok {
				return errors.New("karma does not apply to a pet-class asset")
			}
			if a.Locked() {
				return errors.New("asset is protected by a sealing lock")
			}
			if !af.KarmaEligible(scissorsKarma, d.TradeAvailable()) {
				return errors.New("asset is not applicable to this scissors' karma type")
			}
			if af.HasFlag(a.Flag(), f) {
				return errors.New("asset is already karma-marked")
			}
			if !af.HasFlag(a.Flag(), af.FlagUntradeable) && !af.HasFlag(a.Flag(), af.FlagMergeUntradeable) && !d.TradeBlock() {
				return errors.New("asset is already tradeable")
			}
			updated := Clone(a).AddFlag(f).Build()
			if err := updateFlag(p.db.WithContext(p.ctx), a.Id(), updated.Flag()); err != nil {
				return err
			}
			return mb.Put(asset.EnvEventTopicStatus, UpdatedEventStatusProvider(transactionId, characterId, updated))
		}
	}
}

// ClearKarma removes the karma mark in place. It runs NO gates: it is the
// compensation path for a saga that failed after the mark was applied (FR-6.6),
// and a compensation that can be refused is not a compensation. Clearing an
// already-clear bit is a no-op write plus one UPDATED event.
func (p *ProcessorImpl) ClearKarma(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32) func(a Model) error {
	return func(transactionId uuid.UUID, characterId uint32) func(a Model) error {
		return func(a Model) error {
			f, ok := af.KarmaFlagFor(a.TemplateId())
			if !ok {
				return nil
			}
			updated := Clone(a).RemoveFlag(f).Build()
			if err := updateFlag(p.db.WithContext(p.ctx), a.Id(), updated.Flag()); err != nil {
				return err
			}
			return mb.Put(asset.EnvEventTopicStatus, UpdatedEventStatusProvider(transactionId, characterId, updated))
		}
	}
}
```

> `ApplyLock` persists via `updateFlagAndExpiration`. If no flag-only `updateFlag` helper exists in this package, add one beside it:
> ```go
> func updateFlag(db *gorm.DB, id uint32, flag uint16) error {
> 	return db.Model(&Entity{}).Where("id = ?", id).Update("flag", flag).Error
> }
> ```
> matching `updateFlagAndExpiration`'s exact idiom (entity type, key column, error handling).

- [ ] **Step 4: Add the compartment-layer wrapper**

In `services/atlas-inventory/atlas.com/inventory/compartment/processor.go`, add to the `Processor` interface beside `ApplyAssetLock` (lines 98-99):

```go
	ApplyAssetKarmaAndEmit(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, slot int16, scissorsKarma int32, clear bool) error
	ApplyAssetKarma(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, slot int16, scissorsKarma int32, clear bool) error
```

Add the implementations after `ApplyAssetLock` (after line 1077), modelled on it exactly:

```go
func (p *ProcessorImpl) ApplyAssetKarmaAndEmit(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, slot int16, scissorsKarma int32, clear bool) error {
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			return p.WithTransaction(tx).ApplyAssetKarma(buf)(transactionId, characterId, inventoryType, slot, scissorsKarma, clear)
		})
	})
}

// ApplyAssetKarma resolves the asset addressed by (inventoryType, slot) and
// applies — or, when clear is set, removes — its karma mark. The clear path is
// the saga compensator's; it runs no gates and reads no item data.
func (p *ProcessorImpl) ApplyAssetKarma(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, slot int16, scissorsKarma int32, clear bool) error {
	return func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, slot int16, scissorsKarma int32, clear bool) error {
		p.l.Debugf("Character [%d] attempting to apply karma (clear [%t]) to asset in inventory [%d] slot [%d].", characterId, clear, inventoryType, slot)
		invLock := LockRegistry().Get(characterId, inventoryType)
		invLock.Lock()
		defer invLock.Unlock()

		c, err := p.GetByCharacterAndType(characterId)(inventoryType)
		if err != nil {
			p.l.WithError(err).Errorf("Character [%d] unable to apply karma to asset in inventory [%d] slot [%d].", characterId, inventoryType, slot)
			return err
		}
		a, err := p.assetProcessor.WithTransaction(p.db).GetBySlot(c.Id(), slot)
		if err != nil {
			p.l.WithError(err).Errorf("Character [%d] unable to apply karma to asset in inventory [%d] slot [%d].", characterId, inventoryType, slot)
			return err
		}

		if clear {
			if err := p.assetProcessor.WithTransaction(p.db).ClearKarma(mb)(transactionId, characterId)(a); err != nil {
				p.l.WithError(err).Errorf("Character [%d] unable to clear karma on asset in inventory [%d] slot [%d].", characterId, inventoryType, slot)
				return err
			}
			p.l.Debugf("Character [%d] cleared karma on asset [%d] in inventory [%d] slot [%d].", characterId, a.Id(), inventoryType, slot)
			return nil
		}

		// Gates 2 and 4 need item data. An unreadable lookup is a REFUSAL, never
		// a permissive default — the same contract atlas-trades holds itself to
		// for tradeBlock.
		d, err := p.tradeabilityProcessor.Get(inventoryType, item.Id(a.TemplateId()))
		if err != nil {
			p.l.WithError(err).Errorf("Character [%d] unable to read item data for template [%d]; refusing the karma mark rather than assuming eligibility.", characterId, a.TemplateId())
			return err
		}

		if err := p.assetProcessor.WithTransaction(p.db).ApplyKarma(mb)(transactionId, characterId)(a, scissorsKarma, d); err != nil {
			p.l.WithError(err).Errorf("Character [%d] unable to apply karma to asset in inventory [%d] slot [%d].", characterId, inventoryType, slot)
			return err
		}
		p.l.Debugf("Character [%d] applied karma to asset [%d] in inventory [%d] slot [%d].", characterId, a.Id(), inventoryType, slot)
		return nil
	}
}
```

Add a `tradeabilityProcessor tradeability.Processor` field to `ProcessorImpl`, initialise it in `NewProcessor` with `tradeability.NewProcessor(l, ctx)`, carry it through `WithTransaction`/`copy`-style helpers exactly as the existing `assetProcessor` field is carried, and add a `WithTradeabilityProcessor(p tradeability.Processor) Processor` seam so the tests can inject the mock — following whatever injection idiom the package already uses for `assetProcessor`.

- [ ] **Step 5: Extend both mocks**

In `asset/mock/processor.go` add `ApplyKarmaFunc` / `ClearKarmaFunc` fields and their delegating methods, matching the `ApplyLockFunc` / `ClearLockFunc` shape at lines 36-37 and 213-232.

In `compartment/mock/processor.go` add `ApplyAssetKarmaAndEmitFunc` / `ApplyAssetKarmaFunc` fields and methods, matching lines 55-56 and 365-380.

- [ ] **Step 6: Run tests to verify they pass**

Run: `cd services/atlas-inventory/atlas.com/inventory && go test -race ./... && go vet ./... && go build ./...`
Expected: all PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-inventory/atlas.com/inventory/asset services/atlas-inventory/atlas.com/inventory/compartment
git commit -m "feat(task-223): ApplyKarma/ClearKarma asset and compartment processors with re-asserted gates"
```

---

## Task 8: The `APPLY_KARMA` Kafka command

**Files:**
- Modify: `services/atlas-inventory/atlas.com/inventory/kafka/message/compartment/kafka.go:35` (command name), `:186` (body)
- Modify: `services/atlas-inventory/atlas.com/inventory/kafka/consumer/compartment/consumer.go:96` (registration), after `:403` (handler)
- Test: `services/atlas-inventory/atlas.com/inventory/kafka/consumer/compartment/karma_consumer_test.go` (create)

**Interfaces:**
- Consumes: `compartment.Processor.ApplyAssetKarmaAndEmit` (Task 7).
- Produces:
  - `compartment.CommandApplyKarma = "APPLY_KARMA"`
  - `compartment.ApplyKarmaCommandBody{Slot int16; ScissorsKarma int32; Clear bool}`

- [ ] **Step 1: Write the failing test**

Create `services/atlas-inventory/atlas.com/inventory/kafka/consumer/compartment/karma_consumer_test.go`:

```go
package compartment

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"

	compartment2 "atlas-inventory/kafka/message/compartment"
)

// TestHandleApplyKarmaCommandIgnoresOtherTypes: every handler registered on the
// shared COMMAND_TOPIC_COMPARTMENT sees every command on that topic, so the type
// guard is load-bearing — without it an APPLY_LOCK body unmarshals into an
// ApplyKarmaCommandBody and mutates the wrong thing.
func TestHandleApplyKarmaCommandIgnoresOtherTypes(t *testing.T) {
	l, _ := test.NewNullLogger()
	called := false
	// ... construct a compartment mock whose ApplyAssetKarmaAndEmitFunc sets called = true
	handleApplyKarmaCommand(nil)(l, context.Background(), compartment2.Command[compartment2.ApplyKarmaCommandBody]{
		TransactionId: uuid.New(),
		CharacterId:   1,
		InventoryType: 1,
		Type:          compartment2.CommandApplyLock,
		Body:          compartment2.ApplyKarmaCommandBody{Slot: 3},
	})
	if called {
		t.Fatal("handleApplyKarmaCommand acted on an APPLY_LOCK command")
	}
}
```

> If the package's existing consumer tests use a different seam for injecting the processor (the handler takes `*gorm.DB` and constructs its own), follow that package's established test idiom instead — or, if there is no seam, assert only the type-guard early return, which needs no processor at all. Do not add a new injection seam solely for this test.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-inventory/atlas.com/inventory && go test ./kafka/consumer/compartment/... -run 'ApplyKarma' -v`
Expected: FAIL — `undefined: handleApplyKarmaCommand`.

- [ ] **Step 3: Add the command contract**

In `services/atlas-inventory/atlas.com/inventory/kafka/message/compartment/kafka.go`, in the command-name block beside `CommandApplyLock` (line 35):

```go
	CommandApplyKarma        = "APPLY_KARMA"
```

Beside `ApplyLockCommandBody` (line 186):

```go
// ApplyKarmaCommandBody applies (or, when Clear is set, removes) the
// one-free-trade karma mark on the asset at Slot.
//
// ScissorsKarma is the SCISSORS' OWN WZ info/karma type, forwarded from the
// channel arm so atlas-inventory can re-run the equality half of the eligibility
// predicate without knowing which scissors were used. 0 means "untyped scissors"
// (the v83-era model), under which the predicate reduces to "is the target
// karma-applicable at all".
//
// Clear is the compensation discriminator. It exists here rather than as a
// second saga action so the saga's action and event-acceptance tables stay one
// entry wide (libs/atlas-saga ApplyAssetKarmaPayload).
type ApplyKarmaCommandBody struct {
	Slot          int16 `json:"slot"`
	ScissorsKarma int32 `json:"scissorsKarma"`
	Clear         bool  `json:"clear"`
}
```

- [ ] **Step 4: Add the handler and register it**

In `services/atlas-inventory/atlas.com/inventory/kafka/consumer/compartment/consumer.go`, after `handleApplyLockCommand` (after line 403):

```go
func handleApplyKarmaCommand(db *gorm.DB) message.Handler[compartment2.Command[compartment2.ApplyKarmaCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c compartment2.Command[compartment2.ApplyKarmaCommandBody]) {
		if c.Type != compartment2.CommandApplyKarma {
			return
		}

		l.Debugf("Received APPLY_KARMA command for character [%d], slot [%d], clear [%t].",
			c.CharacterId, c.Body.Slot, c.Body.Clear)

		err := compartment.NewProcessor(l, ctx, db).ApplyAssetKarmaAndEmit(
			c.TransactionId,
			c.CharacterId,
			inventory.Type(c.InventoryType),
			c.Body.Slot,
			c.Body.ScissorsKarma,
			c.Body.Clear,
		)
		if err != nil {
			l.WithError(err).Errorf("Failed to apply karma to asset in slot [%d] for character [%d].", c.Body.Slot, c.CharacterId)
		}
	}
}
```

Register it beside the `handleApplyLockCommand` registration (line 96), matching that block's exact error handling:

```go
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleApplyKarmaCommand(db)))); err != nil {
				// ... same error handling as the neighbouring registrations
			}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd services/atlas-inventory/atlas.com/inventory && go test -race ./... && go vet ./... && go build ./...`
Expected: all PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-inventory/atlas.com/inventory/kafka/
git commit -m "feat(task-223): APPLY_KARMA compartment command and consumer"
```

---

## Task 9: atlas-saga-orchestrator dispatch, acceptance and compensation

**Files:**
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/compartment/kafka.go:32,161` (mirror the contract)
- Modify: `.../compartment/producer.go:126-140`, `.../compartment/processor.go:44,130`
- Modify: `.../saga/model.go:222,327,1610`
- Modify: `.../saga/handler.go:949-950`, after `:1150`
- Modify: `.../saga/event_acceptance.go:126`
- Modify: `.../saga/compensator.go:277` (saga-type arm), inside `DispatchCashItemUseRollbacks`
- Test: `.../saga/karma_compensation_test.go` (create)

**Interfaces:**
- Consumes: `sharedsaga.ApplyAssetKarma`, `sharedsaga.ApplyAssetKarmaPayload`, `sharedsaga.KarmaScissorsUse` (Task 5); atlas-inventory's `APPLY_KARMA` contract (Task 8).
- Produces:
  - `compartment.Processor.RequestApplyKarma(transactionId uuid.UUID, characterId uint32, inventoryType byte, slot int16, scissorsKarma int32, clear bool) error`
  - `saga.ApplyAssetKarma`, `saga.ApplyAssetKarmaPayload` aliases
  - `HandlerImpl.handleApplyAssetKarma`

- [ ] **Step 1: Write the failing test**

Create `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/karma_compensation_test.go`:

```go
package saga

import (
	"testing"

	sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"
)

// TestApplyAssetKarmaIsInTheAcceptanceTable: a missing entry default-denies in
// StepAcceptsEvent, so the step would never complete and the saga would time out.
func TestApplyAssetKarmaIsInTheAcceptanceTable(t *testing.T) {
	kinds, ok := acceptanceTable[sharedsaga.ApplyAssetKarma]
	if !ok {
		t.Fatal("acceptanceTable has no entry for ApplyAssetKarma")
	}
	if len(kinds) != 1 || kinds[0] != EventKindAssetUpdated {
		t.Fatalf("acceptanceTable[ApplyAssetKarma] = %v, want [EventKindAssetUpdated]", kinds)
	}
}

// TestKarmaScissorsUseTakesTheCashItemUseReverseWalk: without the saga-type arm
// a failed karma saga only compensates the failing step, leaving the already
// destroyed scissors unrefunded.
func TestKarmaScissorsUseTakesTheCashItemUseReverseWalk(t *testing.T) {
	// Build a KarmaScissorsUse saga whose consume_scissors DestroyAsset step is
	// Completed and whose apply_asset_karma step is Failed; run
	// CompensateFailedStep against a compartment mock, and assert a
	// RequestCreateItem for the scissors template was dispatched.
}

// TestKarmaRollbackClearsTheMark is FR-6.6: a saga failing AFTER the mark is
// applied must not leave a free trade behind.
func TestKarmaRollbackClearsTheMark(t *testing.T) {
	// Build a KarmaScissorsUse saga whose apply_asset_karma step is Completed,
	// run DispatchCashItemUseRollbacks, and assert RequestApplyKarma was called
	// with clear == true for the payload's (characterId, inventoryType, slot).
}
```

> Fill the two elided bodies from the existing `meso_sack_compensation_test.go` / `point_reset_compensation_test.go`, which already build a saga, inject a compartment mock, and assert on dispatched commands. Use the same construction and assertion helpers.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test ./saga/... -run 'Karma' -v`
Expected: FAIL — no acceptance-table entry.

- [ ] **Step 3: Mirror the Kafka contract**

In `.../kafka/message/compartment/kafka.go`, beside `CommandApplyLock = "APPLY_LOCK"` (line 32):

```go
	CommandApplyKarma         = "APPLY_KARMA"
```

Beside `ApplyLockCommandBody` (line 161), add `ApplyKarmaCommandBody` **byte-identically to atlas-inventory's** (Task 8, Step 3) — same field names, same json tags, same order. The two live in separate Go modules; a field renamed in one and not the other decodes into a zero-valued body at runtime and fails no build.

- [ ] **Step 4: Add the producer and the processor method**

In `.../compartment/producer.go`, after `RequestApplyLockCommandProvider`:

```go
func RequestApplyKarmaCommandProvider(transactionId uuid.UUID, characterId uint32, inventoryType byte, slot int16, scissorsKarma int32, clear bool) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &compartment.Command[compartment.ApplyKarmaCommandBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		InventoryType: inventoryType,
		Type:          compartment.CommandApplyKarma,
		Body:          compartment.ApplyKarmaCommandBody{Slot: slot, ScissorsKarma: scissorsKarma, Clear: clear},
	}
	return producer.SingleMessageProvider(key, value)
}
```

In `.../compartment/processor.go`, add to the interface beside `RequestApplyLock` (line 44):

```go
	RequestApplyKarma(transactionId uuid.UUID, characterId uint32, inventoryType byte, slot int16, scissorsKarma int32, clear bool) error
```

and the implementation beside line 130:

```go
func (p *ProcessorImpl) RequestApplyKarma(transactionId uuid.UUID, characterId uint32, inventoryType byte, slot int16, scissorsKarma int32, clear bool) error {
	return producer.ProviderImpl(p.l)(p.ctx)(compartment.EnvCommandTopic)(RequestApplyKarmaCommandProvider(transactionId, characterId, inventoryType, slot, scissorsKarma, clear))
}
```

Add the matching `RequestApplyKarmaFunc` field and method to the package's `mock` processor.

- [ ] **Step 5: Wire the saga action**

In `.../saga/model.go`, beside line 222: `ApplyAssetKarma = sharedsaga.ApplyAssetKarma`; beside line 327: `ApplyAssetKarmaPayload = sharedsaga.ApplyAssetKarmaPayload`; and beside the `case ApplyAssetLock:` payload-decode arm (line 1610) an identical `case ApplyAssetKarma:` arm decoding `ApplyAssetKarmaPayload`. Also add `KarmaScissorsUse = sharedsaga.KarmaScissorsUse` beside the other saga-type aliases.

In `.../saga/handler.go`, beside lines 949-950:

```go
	case ApplyAssetKarma:
		return h.handleApplyAssetKarma, true
```

and after `handleApplyAssetLock` (after line 1150):

```go
// handleApplyAssetKarma handles the ApplyAssetKarma action.
//
// The scissors' own karma type rides on the payload and is forwarded onto the
// Kafka command body, because atlas-inventory — the owning service and the
// authority — re-runs the eligibility predicate, and the EQUALITY half of that
// predicate is meaningless without it. Forwarding 0 would silently degrade the
// v87+ equality model to the v83 non-zero model.
func (h *HandlerImpl) handleApplyAssetKarma(s Saga, st Step[any]) error {
	payload, ok := st.Payload().(ApplyAssetKarmaPayload)
	if !ok {
		return errors.New("invalid payload")
	}
	err := h.compP.RequestApplyKarma(s.TransactionId(), payload.CharacterId, payload.InventoryType, payload.Slot, payload.ScissorsKarma, false)
	if err != nil {
		h.logActionError(s, st, err, "Unable to apply asset karma.")
		return err
	}
	return nil
}
```

In `.../saga/event_acceptance.go`, beside line 126:

```go
	sharedsaga.ApplyAssetKarma:      {EventKindAssetUpdated},
```

- [ ] **Step 6: Wire compensation**

In `.../saga/compensator.go`, extend the cash-item-use saga-type arm (line 277) to include the new type:

```go
	if s.SagaType() == ItemTagUse || s.SagaType() == SealingLockUse || s.SagaType() == IncubatorUse || s.SagaType() == KarmaScissorsUse {
		return c.compensateCashItemUse(s, failedStep)
	}
```

Inside `DispatchCashItemUseRollbacks`'s reverse-walk switch, add the `ApplyAssetKarma` inverse beside the `DestroyAsset` case:

```go
		case ApplyAssetKarma:
			// Inverse of a completed mark: clear it. A saga that failed after the
			// mark was applied must not leave a free trade behind (FR-6.6).
			if payload, ok := step.Payload().(ApplyAssetKarmaPayload); ok {
				if err := c.compP.RequestApplyKarma(s.TransactionId(), payload.CharacterId, payload.InventoryType, payload.Slot, payload.ScissorsKarma, true); err != nil {
					c.l.WithError(err).Errorf("Unable to clear the karma mark for character [%d] in inventory [%d] slot [%d] during compensation.", payload.CharacterId, payload.InventoryType, payload.Slot)
				}
			}
```

Extend the function's `Inverses:` doc comment with a matching bullet: `- ApplyAssetKarma (target marked one-trade-enabled) → RequestApplyKarma(clear=true).`

- [ ] **Step 7: Run tests to verify they pass**

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test -race ./... && go vet ./... && go build ./...`
Expected: all PASS — including the pre-existing `event_acceptance_test.go` coverage test and `unmarshal_completeness_test.go`, both of which will fail loudly if the new action is missing an entry.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-saga-orchestrator/ libs/atlas-saga/
git commit -m "feat(task-223): orchestrator dispatch, acceptance and reverse-walk compensation for ApplyAssetKarma"
```

---

## Task 10: atlas-channel data clients

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/data/cash/rest.go:7-16`
- Create: `services/atlas-channel/atlas.com/channel/data/tradeability/` (rest.go, requests.go, processor.go, mock/processor.go, processor_test.go)

**Interfaces:**
- Consumes: atlas-data's new `karma` and `tradeAvailable` fields (Task 3).
- Produces:
  - `cash.RestModel.Karma int32` (`json:"karma"`)
  - `tradeability.Processor` with `Get(inventoryType inventory.Type, templateId item.Id) (Model, error)`, `Model.TradeAvailable() int32`, `Model.TradeBlock() bool`, and `tradeability.NewModel(tradeBlock bool, tradeAvailable int32) Model`

- [ ] **Step 1: Add the scissors' karma field to the cash client**

In `services/atlas-channel/atlas.com/channel/data/cash/rest.go`, in `RestModel`:

```go
	// Karma is the SCISSORS' OWN WZ info/karma type (atlas-data cash/rest.go).
	// Absent or 0 means untyped scissors, under which the eligibility predicate
	// reduces to "is the target karma-applicable at all" — the gms_v83 model.
	Karma int32 `json:"karma"`
```

- [ ] **Step 2: Create the tradeability client**

Create `services/atlas-channel/atlas.com/channel/data/tradeability/` as an exact port of the atlas-inventory package from Task 6 — same five wire models, same `extract`, same `requests.go` paths, same `Processor` interface, same mock, same `NewModel` constructor. Change only the package's module-local import path (`atlas-channel/data/tradeability` in the mock).

Copy `processor_test.go` from Task 6 unchanged apart from the package clause.

- [ ] **Step 3: Run tests**

Run: `cd services/atlas-channel/atlas.com/channel && go test -race ./data/... && go build ./...`
Expected: PASS, build clean.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/data/
git commit -m "feat(task-223): channel-side karma and tradeability data clients"
```

---

## Task 11: The version-scoped cash-slot-type resolver

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:1103-1109` (use the constant), after `:765` (add the resolver)
- Test: `services/atlas-channel/atlas.com/channel/socket/handler/karma_slot_type_test.go` (create)

**Interfaces:**
- Consumes: `item.ClassificationKarmaScissors` (Task 2); the existing `CashSlotItemTypeSealTimed`/`SealTimedV95` constants.
- Produces: `karmaScissorsCashSlotItemType(t tenant.Model) CashSlotItemType`, `CashSlotItemTypeKarmaScissors = CashSlotItemType(63)`, `CashSlotItemTypeKarmaScissorsV95 = CashSlotItemType(64)`.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-channel/atlas.com/channel/socket/handler/karma_slot_type_test.go`:

```go
package handler

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

// TestKarmaAndSealResolversAreDisjoint is the FR-2.4 regression guard, and it is
// not ceremony: pre-95, CashSlotItemTypeSealTimed is 64 and so is the GMS >= 95
// karma type. The two arms are disjoint today ONLY because the seal arm
// recomputes itself to 65 at GMS >= 95. A version-scoped resolver on each side
// makes the disjointness structural; this test is what keeps it that way.
func TestKarmaAndSealResolversAreDisjoint(t *testing.T) {
	for _, v := range testTenantVariants(t) {
		t.Run(v.name, func(t *testing.T) {
			karma := karmaScissorsCashSlotItemType(v.tenant)
			seal := CashSlotItemTypeSealTimed
			if v.tenant.Region() == "GMS" && v.tenant.MajorVersion() >= 95 {
				seal = CashSlotItemTypeSealTimedV95
			}
			if karma == seal {
				t.Fatalf("karma and seal cash-slot types collide at %s: both %d", v.name, karma)
			}
		})
	}
}

// TestGetCashSlotItemTypeFor552Unchanged: rewriting the bare `category == 552`
// branch to use the named constant must not change a single returned value.
func TestGetCashSlotItemTypeFor552Unchanged(t *testing.T) {
	for _, v := range testTenantVariants(t) {
		t.Run(v.name, func(t *testing.T) {
			want := CashSlotItemType(63)
			if v.tenant.Region() == "GMS" && v.tenant.MajorVersion() >= 95 {
				want = CashSlotItemType(64)
			}
			if got := GetCashSlotItemType(v.tenant)(item.Id(5520000)); got != want {
				t.Fatalf("GetCashSlotItemType(5520000) = %d, want %d", got, want)
			}
			if got := GetCashSlotItemType(v.tenant)(item.Id(5520001)); got != want {
				t.Fatalf("GetCashSlotItemType(5520001) = %d, want %d", got, want)
			}
		})
	}
}
```

> `testTenantVariants` is a small local helper returning one `{name string; tenant tenant.Model}` per configured tenant version. Build it from whatever version list this package's existing tests already enumerate (check the handler package's other `_test.go` files, and `libs/atlas-packet/test`'s `Variants` for the canonical set: gms v48/61/72/79/83/84/87/92/95 and jms v185). If the handler package has no such helper, write one in this file covering that full set.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/... -run 'Karma' -v`
Expected: FAIL — `undefined: karmaScissorsCashSlotItemType`.

- [ ] **Step 3: Add the constants and the resolver**

In `character_cash_item_use.go`, in the `CashSlotItemType` const block beside `CashSlotItemTypeSealTimedV95` (line 697):

```go
	CashSlotItemTypeKarmaScissors    = CashSlotItemType(63) // GMS < 95, and JMS
	CashSlotItemTypeKarmaScissorsV95 = CashSlotItemType(64) // GMS >= 95
```

After `viciousHammerCashSlotItemType` (after line 765):

```go
// karmaScissorsCashSlotItemType returns the version-scoped CashSlotItemType for
// the Scissors of Karma (classification 552).
//
// A bare constant compare is FORBIDDEN here: pre-95, CashSlotItemTypeSealTimed
// is also 64. The karma and seal arms are disjoint at runtime today only because
// the seal arm recomputes itself to 65 on GMS >= 95 (:261-265) — a coincidence
// that a version-scoped resolver on both sides turns into a structural property.
// karma_slot_type_test.go asserts the two never collide on any configured
// version.
func karmaScissorsCashSlotItemType(t tenant.Model) CashSlotItemType {
	if t.Region() == "GMS" && t.MajorVersion() >= 95 {
		return CashSlotItemTypeKarmaScissorsV95
	}
	return CashSlotItemTypeKarmaScissors
}
```

Replace the bare `if category == 552 { ... }` branch (lines 1103-1109) with:

```go
		if category == item.ClassificationKarmaScissors {
			return karmaScissorsCashSlotItemType(t)
		}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go test -race ./socket/handler/... && go vet ./...`
Expected: PASS. The returned values are unchanged (63 / 64), which `TestGetCashSlotItemTypeFor552Unchanged` proves.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/
git commit -m "feat(task-223): version-scoped karma-scissors cash-slot-type resolver"
```

---

## Task 12: The atlas-channel handler arm

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go` — insert the arm after the seal arm's closing brace (currently ends at line 332, immediately before `if it == CashSlotItemTypeIncubator {`)
- Test: `services/atlas-channel/atlas.com/channel/socket/handler/karma_arm_test.go` (create)

**Interfaces:**
- Consumes: `cashsb.NewItemUseKarmaScissors` (Task 2); `karmaScissorsCashSlotItemType` (Task 11); `cashData.Processor.GetById` with the new `Karma` field and the new `tradeability.Processor` (Task 10); `asset.KarmaFlagFor`/`KarmaEligible` (Task 1); `saga.KarmaScissorsUse`/`ApplyAssetKarma`/`ApplyAssetKarmaPayload` (Task 5, incl. `ScissorsKarma`).
- Produces: the karma arm. No new exported symbols.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-channel/atlas.com/channel/socket/handler/karma_arm_test.go`. Assert, one case per gate, that **no saga is created** and that the client is unlocked. Use the package's existing handler-test idiom (the seal/item-tag arms already have tests — follow their seam for stubbing `cashItemInSlotFunc`, the character processor, the saga processor and `session.Announce`).

```go
package handler

import "testing"

// One case per refusal gate. Each asserts: no saga created, no state mutated,
// exactly one unlock announced, and a warn logged naming the failing rule.
func TestKarmaArmRefusals(t *testing.T) {
	testCases := []struct {
		name string
		// ... per-case fixture knobs
	}{
		{name: "0a scissors not in the claimed cash slot"},
		{name: "0b unknown target inventory type"},
		{name: "0c pet-class target"},
		{name: "0d negative target slot (equipped)"},
		{name: "0e empty target slot"},
		{name: "1 target is sealing-locked"},
		{name: "2 target tradeAvailable is 0"},
		{name: "2 target tradeAvailable differs from the scissors' karma type"},
		{name: "3 target is already karma-marked"},
		{name: "4 target is already tradeable"},
		{name: "unreadable scissors cash data"},
		{name: "unreadable target item data"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// ... arrange, invoke CharacterCashItemUseHandleFunc with a karma
			// sub-body, assert sagaCreated == false and unlockCount == 1
		})
	}
}

// TestKarmaArmSuccessCreatesTwoStepSaga: consume first, mark second, so a failed
// mark compensates by restoring the scissors rather than leaving a free trade.
func TestKarmaArmSuccessCreatesTwoStepSaga(t *testing.T) {
	// ... assert saga.SagaType == saga.KarmaScissorsUse, len(Steps) == 2,
	// Steps[0].Action == saga.DestroyAsset with the scissors templateId and
	// Quantity 1, Steps[1].Action == saga.ApplyAssetKarma with the target's
	// inventoryType, slot and the scissors' ScissorsKarma.
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/... -run 'TestKarmaArm' -v`
Expected: FAIL — no arm exists; the use falls through to the terminal `not implemented` warn.

- [ ] **Step 3: Write the arm**

Insert after the seal arm's closing `}` (before `if it == CashSlotItemTypeIncubator {`):

```go
		if it == karmaScissorsCashSlotItemType(t) {
			sp := cashsb.NewItemUseKarmaScissors(updateTimeFirst)
			sp.Decode(l, ctx)(r, readerOptions)
			invTypeRaw := sp.InventoryType()
			targetSlot := int16(sp.Slot())

			// The client takes an exclusive-request lock before sending
			// (gms_v83 @0x830FB5 gates on CanSendExclRequest(500, 0) and then
			// sets the lock), so EVERY outcome must unlock — a refusal that
			// returns silently wedges the client until the next unlocking
			// packet. The success path's non-silent INVENTORY_OPERATION, driven
			// by the UPDATED event, clears the lock on its own; only the
			// refusals need this.
			refuse := func(format string, args ...interface{}) {
				l.Warnf(format, args...)
				_ = session.Announce(l)(ctx)(wp)(statpkt.StatChangedWriter)(statpkt.NewStatChanged(make([]statpkt.Update, 0), true).Encode)(s)
			}

			// Gate 0b: the raw inventory-type int off the wire must be one of the
			// five known compartments. inventory.Type is a signed int8, so an
			// out-of-range value would otherwise address a nonexistent
			// compartment rather than fail.
			invType, ok := knownInventoryType(invTypeRaw)
			if !ok {
				refuse("Character [%d] attempted to use karma scissors [%d] against unknown inventory type [%d] slot [%d].", s.CharacterId(), itemId, invTypeRaw, targetSlot)
				return
			}
			// Gate 0d: a negative slot is an equipped item.
			if targetSlot < 0 {
				refuse("Character [%d] attempted to use karma scissors [%d] on equipped slot [%d] of inventory [%d].", s.CharacterId(), itemId, targetSlot, invType)
				return
			}
			// Gate 0e: the slot must be occupied.
			target, err := character2.NewProcessor(l, ctx).GetItemInSlot(s.CharacterId(), invType, targetSlot)()
			if err != nil {
				refuse("Character [%d] attempted to use karma scissors [%d] on empty slot [%d] of inventory [%d].", s.CharacterId(), itemId, targetSlot, invType)
				return
			}
			// Gate 0c: pets carry karma on bit 0x01, which is FlagLock in
			// Atlas's shared flag column. See libs/atlas-constants/asset.KarmaFlagFor.
			karmaBit, ok := af.KarmaFlagFor(target.TemplateId())
			if !ok {
				refuse("Character [%d] attempted to use karma scissors [%d] on pet-class item [%d] in inventory [%d] slot [%d]; pets are not karma targets.", s.CharacterId(), itemId, target.TemplateId(), invType, targetSlot)
				return
			}
			// Gate 1: CUIKarmaDlg::PutItem's first refusal — IsProtectedItem.
			if target.Locked() {
				refuse("Character [%d] attempted to use karma scissors [%d] on sealing-locked item [%d] in inventory [%d] slot [%d].", s.CharacterId(), itemId, target.TemplateId(), invType, targetSlot)
				return
			}
			// Gate 2: the eligibility predicate. The scissors' own karma type
			// comes from ITS data, the target's from the target's — no literal
			// karma type appears anywhere, which is why 5520001 works the moment
			// a tenant's WZ carries it and is unusable when it does not.
			cd, err := cashData.NewProcessor(l, ctx).GetById(uint32(itemId))
			if err != nil {
				refuse("Character [%d] used karma scissors [%d] but its cash item data could not be read; refusing rather than assuming an untyped scissors.", s.CharacterId(), itemId)
				return
			}
			td, err := tradeability.NewProcessor(l, ctx).Get(invType, item.Id(target.TemplateId()))
			if err != nil {
				refuse("Character [%d] used karma scissors [%d] on item [%d] whose item data could not be read; refusing rather than assuming eligibility.", s.CharacterId(), itemId, target.TemplateId())
				return
			}
			if !af.KarmaEligible(cd.Karma, td.TradeAvailable()) {
				refuse("Character [%d] attempted to use karma scissors [%d] (karma type [%d]) on ineligible item [%d] (tradeAvailable [%d]) in inventory [%d] slot [%d].", s.CharacterId(), itemId, cd.Karma, target.TemplateId(), td.TradeAvailable(), invType, targetSlot)
				return
			}
			// Gate 3: IsPossibleTradingItem — the mark is already set.
			if af.HasFlag(target.Flag(), karmaBit) {
				refuse("Character [%d] attempted to use karma scissors [%d] on already-marked item [%d] in inventory [%d] slot [%d].", s.CharacterId(), itemId, target.TemplateId(), invType, targetSlot)
				return
			}
			// Gate 4: server-only. Karma exists to unlock an UNTRADEABLE item;
			// marking a tradeable one is a no-op that still consumes the
			// scissors. "Untradeable" is the same pair of conditions
			// atlas-trades enforces, so this gate and the trade-side override
			// are two readings of one definition and cannot disagree.
			if !af.HasFlag(target.Flag(), af.FlagUntradeable) && !af.HasFlag(target.Flag(), af.FlagMergeUntradeable) && !td.TradeBlock() {
				refuse("Character [%d] attempted to use karma scissors [%d] on already-tradeable item [%d] in inventory [%d] slot [%d].", s.CharacterId(), itemId, target.TemplateId(), invType, targetSlot)
				return
			}

			// Consume first, mark second: a failure to apply the mark then
			// compensates by restoring the scissors rather than leaving a free
			// trade behind.
			transactionId := uuid.New()
			now := time.Now()
			_ = saga.NewProcessor(l, ctx).Create(saga.Saga{
				TransactionId: transactionId,
				SagaType:      saga.KarmaScissorsUse,
				InitiatedBy:   "CASH_ITEM_USE",
				Steps: []saga.Step{
					{
						StepId: "consume_karma_scissors",
						Status: saga.Pending,
						Action: saga.DestroyAsset,
						Payload: saga.DestroyAssetPayload{
							CharacterId: s.CharacterId(),
							TemplateId:  uint32(itemId),
							Quantity:    1,
						},
						CreatedAt: now,
						UpdatedAt: now,
					},
					{
						StepId: "apply_asset_karma",
						Status: saga.Pending,
						Action: saga.ApplyAssetKarma,
						Payload: saga.ApplyAssetKarmaPayload{
							CharacterId:   s.CharacterId(),
							InventoryType: byte(invType),
							Slot:          targetSlot,
							ScissorsKarma: cd.Karma,
						},
						CreatedAt: now,
						UpdatedAt: now,
					},
				},
			})
			return
		}
```

Add the `knownInventoryType` helper beside `karmaScissorsCashSlotItemType`:

```go
// knownInventoryType decodes the raw inventory-type int off the wire into a
// shared inventory.Type, reporting false for anything that is not one of the
// five compartments. inventory.Type is a SIGNED int8, so an out-of-range value
// would silently address a nonexistent compartment if merely converted — a
// crafted packet must be a refusal, not a panic or a wrong-compartment read.
// Mirrors atlas-trades' stageableInventoryType.
func knownInventoryType(raw int32) (inventory.Type, bool) {
	if raw < 0 || raw > math.MaxInt8 {
		return 0, false
	}
	t := inventory.Type(raw)
	for _, known := range inventory.Types {
		if t == known {
			return t, true
		}
	}
	return 0, false
}
```

Add the imports the arm needs: `af "github.com/Chronicle20/atlas/libs/atlas-constants/asset"` and `"atlas-channel/data/tradeability"`. `math`, `time`, `uuid`, `statpkt`, `cashData`, `character2`, `saga`, `session`, `item` and `inventory` are already imported.

> **Gate 0a is already enforced above the arm**, at lines 55-59: `cashItemInSlotFunc` verifies the claimed CASH slot really holds the claimed scissors template and returns early on mismatch. Do not duplicate it inside the arm; do include a `refuse`-shaped test case for it (Step 1's case "0a"), asserting the existing early return.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go test -race ./... && go vet ./... && go build ./...`
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/
git commit -m "feat(task-223): karma scissors handler arm with nine server-side gates"
```

---

## Task 13: atlas-trades honours the mark

**Files:**
- Modify: `services/atlas-trades/atlas.com/trades/trade/restriction.go:25-37` (assetView), `:78-93` (checkRestrictions)
- Modify: `services/atlas-trades/atlas.com/trades/trade/processor.go:928` (the call site)
- Test: `services/atlas-trades/atlas.com/trades/trade/restriction_test.go` (extend)

> `services/atlas-trades/.../data/item/` is deliberately NOT touched. The stage-time gate reads the karma bit off the asset's own flags; it never needs `tradeAvailable`, which decides *whether the scissors may be applied*, not *whether a marked item may be traded*. Adding an unused field to five wire models would be surface for nothing.

**Interfaces:**
- Consumes: `asset.KarmaFlagFor` (Task 1).
- Produces: `assetView` gains `TemplateId uint32`. `checkRestrictions`'s two tradeability refusals become conditional on the karma bit.

- [ ] **Step 1: Write the failing test**

Append to `services/atlas-trades/atlas.com/trades/trade/restriction_test.go`:

```go
// TestKarmaMarkOverridesUntradeableFlag: karma exists to let an untradeable item
// through exactly once.
func TestKarmaMarkOverridesUntradeableFlag(t *testing.T) {
	flags := uint16(asset.FlagUntradeable) | uint16(asset.FlagKarmaEquip)
	err := checkRestrictions(assetView{Flags: flags, TemplateId: 1002357}, itemDataView{}, byte(inventory.TypeValueEquip))
	if err != nil {
		t.Fatalf("checkRestrictions refused a karma-marked untradeable equip: %v", err)
	}
}

// TestKarmaMarkOverridesTradeBlock is the one that decides whether the feature
// works at all: untradeable items derive their untradeability MOSTLY from the WZ
// tradeBlock prop, so a karma mark that only defeated the flag check would still
// be refused.
func TestKarmaMarkOverridesTradeBlock(t *testing.T) {
	flags := uint16(asset.FlagKarmaEquip)
	err := checkRestrictions(assetView{Flags: flags, TemplateId: 1002357}, itemDataView{TradeBlock: true}, byte(inventory.TypeValueEquip))
	if err != nil {
		t.Fatalf("checkRestrictions refused a karma-marked tradeBlock'd equip: %v", err)
	}
}

// TestKarmaMarkUsesTheSlotClassBit: the BUNDLE bit (0x02) on an equip is
// FlagSpikes and must NOT read as a karma mark.
func TestKarmaMarkUsesTheSlotClassBit(t *testing.T) {
	spikedEquip := uint16(asset.FlagUntradeable) | uint16(asset.FlagSpikes)
	if err := checkRestrictions(assetView{Flags: spikedEquip, TemplateId: 1002357}, itemDataView{}, byte(inventory.TypeValueEquip)); err != errUntradeableFlag {
		t.Fatalf("a SPIKED untradeable equip was treated as karma-marked: %v", err)
	}
	markedBundle := uint16(asset.FlagUntradeable) | uint16(asset.FlagKarmaUse)
	if err := checkRestrictions(assetView{Flags: markedBundle, TemplateId: 2280000}, itemDataView{}, byte(inventory.TypeValueUse)); err != nil {
		t.Fatalf("checkRestrictions refused a karma-marked untradeable bundle: %v", err)
	}
}

// TestKarmaMarkDoesNotWeakenTheOtherRules is FR-7.2.
func TestKarmaMarkDoesNotWeakenTheOtherRules(t *testing.T) {
	flags := uint16(asset.FlagKarmaEquip)
	if err := checkRestrictions(assetView{Flags: flags, TemplateId: 1002357, SourceSlot: -11}, itemDataView{}, byte(inventory.TypeValueEquip)); err != errEquipped {
		t.Fatalf("a karma mark rescued an EQUIPPED item: %v", err)
	}
	if err := checkRestrictions(assetView{Flags: flags, TemplateId: 1002357}, itemDataView{Unreadable: true}, byte(inventory.TypeValueEquip)); err != errItemDataUnknown {
		t.Fatalf("a karma mark rescued an UNREADABLE item lookup: %v", err)
	}
	if err := checkRestrictions(assetView{Flags: flags, TemplateId: 1002357}, itemDataView{}, byte(99)); err != errUnknownInventory {
		t.Fatalf("a karma mark rescued an UNKNOWN compartment: %v", err)
	}
}

// TestUnmarkedUntradeableStillRefuses: the override must not become the rule.
func TestUnmarkedUntradeableStillRefuses(t *testing.T) {
	if err := checkRestrictions(assetView{Flags: uint16(asset.FlagUntradeable), TemplateId: 1002357}, itemDataView{}, byte(inventory.TypeValueEquip)); err != errUntradeableFlag {
		t.Fatalf("an UNMARKED untradeable equip was allowed: %v", err)
	}
	if err := checkRestrictions(assetView{TemplateId: 1002357}, itemDataView{TradeBlock: true}, byte(inventory.TypeValueEquip)); err != errTradeBlock {
		t.Fatalf("an UNMARKED tradeBlock'd equip was allowed: %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-trades/atlas.com/trades && go test ./trade/... -run 'Karma' -v`
Expected: FAIL — `unknown field TemplateId in struct literal`.

- [ ] **Step 3: Add `TemplateId` and the override**

In `restriction.go`, add to `assetView`:

```go
	// TemplateId is needed to resolve the KARMA BIT, which is slot-class
	// dependent: 0x10 on an equip, 0x02 on a bundle — and 0x02 on an equip is
	// FlagSpikes. See libs/atlas-constants/asset.KarmaFlagFor.
	TemplateId uint32
```

Replace the two tradeability refusals in `checkRestrictions` with:

```go
	// A karma mark (Scissors of Karma, task-223) buys exactly one transfer, and
	// it must defeat BOTH tradeability rules or it defeats nothing useful:
	// untradeable items derive their untradeability mostly from the WZ
	// tradeBlock prop, not from the flag. The mark is CONSUMED by the transfer
	// — it is masked off the settlement snapshot at the moment the receiving
	// asset is built (atlas-saga-orchestrator's trade-settlement expansion), so
	// the item arrives untradeable for its new owner. An UNWOUND (cancelled)
	// trade replays the same snapshot unmasked, so a staged-then-unstaged item
	// keeps its mark.
	//
	// The other three rules are untouched: unknown compartment and equipped slot
	// are checked above, and errItemDataUnknown stays ABOVE the tradeBlock check
	// so an unreadable lookup is never rescued by a mark.
	karmaMarked := false
	if f, ok := asset.KarmaFlagFor(a.TemplateId); ok {
		karmaMarked = asset.HasFlag(a.Flags, f)
	}
	if !karmaMarked && (asset.HasFlag(a.Flags, asset.FlagUntradeable) || asset.HasFlag(a.Flags, asset.FlagMergeUntradeable)) {
		return errUntradeableFlag
	}
	if d.Unreadable {
		return errItemDataUnknown
	}
	if !karmaMarked && d.TradeBlock {
		return errTradeBlock
	}
	return nil
```

- [ ] **Step 4: Populate `TemplateId` at the call site**

In `services/atlas-trades/atlas.com/trades/trade/processor.go:928`:

```go
	if err = checkRestrictions(assetView{Flags: a.Flag(), SourceSlot: a.Slot(), TemplateId: a.TemplateId()}, p.itemData(it, a.TemplateId()), inventoryType); err != nil {
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd services/atlas-trades/atlas.com/trades && go test -race ./... && go vet ./... && go build ./...`
Expected: all PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-trades/atlas.com/trades/
git commit -m "feat(task-223): karma mark overrides the untradeable flag and WZ tradeBlock at stage time"
```

---

## Task 14: The mark is consumed by a completed trade

**Files:**
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/processor.go:1597-1598` (settlement expansion only — **not** the unwind expansion at `:1682-1683`)
- Test: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/karma_consumption_test.go` (create)

**Interfaces:**
- Consumes: `asset.KarmaFlagFor` (Task 1); the existing `assetDataFromSnapshot(AssetSnapshot) asset2.AssetData`.
- Produces: `clearKarmaFromSnapshot(s AssetSnapshot) AssetSnapshot`.

**Why here and not "after the trade completes":** both transfer paths already move an asset by destroying it on one side and re-materialising it on the other from a snapshot carrying `Flag uint16`. Masking the bit off **in the snapshot** makes the clear and the transfer the same write — FR-7.5's atomicity becomes structural rather than something to coordinate. It also gives FR-7.6 for free: unwind replays the *same* snapshot unmasked, so a cancelled trade keeps the mark.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/karma_consumption_test.go`:

```go
package saga

import (
	"testing"

	af "github.com/Chronicle20/atlas/libs/atlas-constants/asset"
)

func TestClearKarmaFromSnapshotEquip(t *testing.T) {
	in := AssetSnapshot{TemplateId: 1002357, Flag: uint16(af.FlagUntradeable) | uint16(af.FlagKarmaEquip)}
	out := clearKarmaFromSnapshot(in)
	if af.HasFlag(out.Flag, af.FlagKarmaEquip) {
		t.Fatal("the equip karma bit survived the transfer")
	}
	if !af.HasFlag(out.Flag, af.FlagUntradeable) {
		t.Fatal("clearing karma disturbed FlagUntradeable; the item must arrive UNTRADEABLE")
	}
}

func TestClearKarmaFromSnapshotBundle(t *testing.T) {
	in := AssetSnapshot{TemplateId: 2280000, Flag: uint16(af.FlagUntradeable) | uint16(af.FlagKarmaUse)}
	out := clearKarmaFromSnapshot(in)
	if af.HasFlag(out.Flag, af.FlagKarmaUse) {
		t.Fatal("the bundle karma bit survived the transfer")
	}
}

// TestClearKarmaFromSnapshotLeavesSpikesAlone: 0x02 on an EQUIP is FlagSpikes,
// and a traded spiked equip must arrive spiked.
func TestClearKarmaFromSnapshotLeavesSpikesAlone(t *testing.T) {
	in := AssetSnapshot{TemplateId: 1002357, Flag: uint16(af.FlagSpikes)}
	out := clearKarmaFromSnapshot(in)
	if !af.HasFlag(out.Flag, af.FlagSpikes) {
		t.Fatal("a spiked equip lost its spikes in transfer")
	}
}

// TestClearKarmaFromSnapshotSkipsPets: KarmaFlagFor reports no bit for a pet,
// and the pet bit is 0x01 = FlagLock. A pet passing through a trade must be
// untouched.
func TestClearKarmaFromSnapshotSkipsPets(t *testing.T) {
	in := AssetSnapshot{TemplateId: 5000000, Flag: uint16(af.FlagLock)}
	out := clearKarmaFromSnapshot(in)
	if out.Flag != in.Flag {
		t.Fatalf("a pet's flag changed in transfer: %#x -> %#x", in.Flag, out.Flag)
	}
}

// TestSettlementClearsTheMarkAndUnwindDoesNot is FR-7.4 + FR-7.6 together.
func TestSettlementClearsTheMarkAndUnwindDoesNot(t *testing.T) {
	// Build a TradeSettlementPayload whose one item's Snapshot carries
	// FlagKarmaEquip; expand it; assert the AcceptToCharacter step's AssetData
	// Flag has the bit CLEAR.
	//
	// Then build the equivalent TradeUnwindPayload; expand it; assert the
	// AcceptToCharacter step's AssetData Flag has the bit SET.
	//
	// Reuse the fixtures in trade_expansion_test.go / trade_escrow_expansion_test.go.
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test ./saga/... -run 'ClearKarma' -v`
Expected: FAIL — `undefined: clearKarmaFromSnapshot`.

- [ ] **Step 3: Add the mask and apply it to the settlement expansion only**

Beside `assetDataFromSnapshot` in `processor.go` (near line 1723):

```go
// clearKarmaFromSnapshot masks the one-free-trade karma mark off a snapshot at
// the moment a TRANSFER OF OWNERSHIP re-materialises the asset for its new owner
// (task-223 FR-7.4). The grant is "1 time of trading has been enabled" — the
// unit consumed is a transfer, so the item must arrive untradeable.
//
// Doing it HERE, in the snapshot, rather than as a follow-up mutation on the
// delivered item, is what makes FR-7.5 structural: the clear and the transfer
// are the same write, and there is no window in which a delivered item still
// carries a free trade.
//
// It is applied to the SETTLEMENT expansion only. The UNWIND expansion replays
// the same snapshot unmasked, so a staged-then-cancelled trade keeps its mark
// (FR-7.6) with no extra code and no risk of the two paths diverging.
//
// A pet is skipped entirely: KarmaFlagFor reports no bit for the pet class, and
// the client's pet karma bit is 0x01, which is FlagLock everywhere else.
func clearKarmaFromSnapshot(s AssetSnapshot) AssetSnapshot {
	f, ok := af.KarmaFlagFor(s.TemplateId)
	if !ok {
		return s
	}
	s.Flag = af.ClearFlag(s.Flag, f)
	return s
}
```

In the **settlement** expansion (line 1598), change:

```go
					AssetData:     assetDataFromSnapshot(clearKarmaFromSnapshot(it.Snapshot)),
```

Leave the **unwind** expansion at line 1683 exactly as it is, and add a one-line comment there:

```go
				// NOT clearKarmaFromSnapshot: an unwound trade is not a transfer
				// of ownership, so a staged-then-cancelled item keeps its mark
				// (task-223 FR-7.6).
				AssetData:     assetDataFromSnapshot(ui.Item.Snapshot),
```

Add the `af "github.com/Chronicle20/atlas/libs/atlas-constants/asset"` import if not already present.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test -race ./... && go vet ./... && go build ./...`
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/
git commit -m "feat(task-223): a completed trade consumes the karma mark; an unwind preserves it"
```

---

## Task 15: atlas-merchant lists and consumes the mark

**Files:**
- Modify: `services/atlas-merchant/atlas.com/merchant/shop/validation.go:122-137`
- Modify: `services/atlas-merchant/atlas.com/merchant/shop/processor.go:1038` (buy path only)
- Test: `services/atlas-merchant/atlas.com/merchant/shop/karma_test.go` (create)

**Interfaces:**
- Consumes: `asset.KarmaFlagFor` (Task 1).
- Produces: `IsListableItem` accepts a karma-marked asset; `clearKarmaFromAssetData(itemId uint32, ad asset2.AssetData) asset2.AssetData`.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-merchant/atlas.com/merchant/shop/karma_test.go`:

```go
package shop

import (
	"testing"

	af "github.com/Chronicle20/atlas/libs/atlas-constants/asset"
)

func TestIsListableItemAcceptsKarmaMarkedEquip(t *testing.T) {
	flags := uint16(af.FlagUntradeable) | uint16(af.FlagKarmaEquip)
	if err := IsListableItem(1002357, flags); err != nil {
		t.Fatalf("IsListableItem refused a karma-marked untradeable equip: %v", err)
	}
}

func TestIsListableItemAcceptsKarmaMarkedBundle(t *testing.T) {
	flags := uint16(af.FlagUntradeable) | uint16(af.FlagKarmaUse)
	if err := IsListableItem(2280000, flags); err != nil {
		t.Fatalf("IsListableItem refused a karma-marked untradeable bundle: %v", err)
	}
}

// A SPIKED untradeable equip carries 0x02, the BUNDLE karma bit, and must still
// be refused.
func TestIsListableItemStillRefusesSpikedUntradeableEquip(t *testing.T) {
	flags := uint16(af.FlagUntradeable) | uint16(af.FlagSpikes)
	if err := IsListableItem(1002357, flags); err != ErrUntradeableItem {
		t.Fatalf("IsListableItem = %v, want ErrUntradeableItem", err)
	}
}

func TestIsListableItemStillRefusesUnmarkedUntradeable(t *testing.T) {
	if err := IsListableItem(1002357, uint16(af.FlagUntradeable)); err != ErrUntradeableItem {
		t.Fatalf("IsListableItem = %v, want ErrUntradeableItem", err)
	}
}

// ErrPetItem and ErrCashItem are untouched by the karma override.
func TestIsListableItemStillRefusesPetsAndCash(t *testing.T) {
	if err := IsListableItem(5000000, uint16(af.FlagKarmaUse)); err != ErrPetItem {
		t.Fatalf("IsListableItem(pet) = %v, want ErrPetItem", err)
	}
	if err := IsListableItem(5520000, uint16(af.FlagKarmaUse)); err != ErrCashItem {
		t.Fatalf("IsListableItem(cash) = %v, want ErrCashItem", err)
	}
}

func TestClearKarmaFromAssetData(t *testing.T) {
	// equip: the bit is cleared, FlagUntradeable survives
	// bundle: the bit is cleared
	// spiked equip: FlagSpikes survives
	// pet: the flag is untouched
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-merchant/atlas.com/merchant && go test ./shop/... -run 'Karma' -v`
Expected: FAIL — `IsListableItem` refuses the karma-marked cases; `clearKarmaFromAssetData` undefined.

- [ ] **Step 3: Add the listing override and the sale-time clear**

In `validation.go`, replace the untradeable refusal:

```go
	// A karma mark (Scissors of Karma, task-223) buys exactly one transfer, and a
	// hired-merchant SALE is one — so a marked item lists, and the mark is
	// consumed when the buyer's asset is built (see the buy path in
	// processor.go). Listing-only semantics would let a player launder one mark
	// into unlimited transfers by re-listing. ErrPetItem and ErrCashItem above
	// are untouched: they are not tradeability rules.
	//
	// The bit is slot-class dependent — 0x02 on an EQUIP is FlagSpikes, so a
	// spiked untradeable equip must still be refused. KarmaFlagFor is the only
	// thing that may pick it.
	karmaMarked := false
	if f, ok := asset.KarmaFlagFor(itemId); ok {
		karmaMarked = asset.HasFlag(flag, f)
	}
	if !karmaMarked && asset.HasFlag(flag, asset.FlagUntradeable) {
		return ErrUntradeableItem
	}
	return nil
```

Add beside `IsListableItem`:

```go
// clearKarmaFromAssetData masks the karma mark off the snapshot used to build
// the BUYER's asset. Same rationale as the trade path (atlas-saga-orchestrator's
// clearKarmaFromSnapshot): the clear and the transfer are the same write, so
// there is no window in which the delivered item still carries a free trade.
//
// Applied ONLY where ownership changes hands. The three "return the item to its
// owner" paths — shop closure, listing removal, Frederick retrieval — pass the
// snapshot through untouched, exactly as a cancelled trade does.
func clearKarmaFromAssetData(itemId uint32, ad asset2.AssetData) asset2.AssetData {
	f, ok := asset.KarmaFlagFor(itemId)
	if !ok {
		return ad
	}
	ad.Flag = asset.ClearFlag(ad.Flag, f)
	return ad
}
```

In `processor.go:1038`, the buy path:

```go
		// Grant items to buyer. A completed sale is a transfer of ownership, so
		// it CONSUMES any karma mark (task-223 FR-7.4).
		ad := clearKarmaFromAssetData(result.ItemId, result.ItemSnapshot.WithQuantity(uint32(result.BundleSize)*uint32(result.BundlesPurchased)))
```

Leave `acceptItemToBuffer` (line 1602) and the Frederick path (line 1345) untouched — both return the item to its original owner.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-merchant/atlas.com/merchant && go test -race ./... && go vet ./... && go build ./...`
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-merchant/atlas.com/merchant/
git commit -m "feat(task-223): hired-merchant listing honours the karma mark; a completed sale consumes it"
```

---

## Task 16: Coverage manifest, template check, and the full verification gate

**Files:**
- Create: `docs/tasks/task-223-karma-scissors/coverage-manifest.yaml`

**Interfaces:**
- Consumes: everything above.
- Produces: the declared packet coverage for this task.

- [ ] **Step 1: Confirm `USE_CASH_ITEM` is bound in every template**

Run:

```bash
grep -rl "CharacterCashItemUseHandle" services/atlas-configurations/seed-data/templates/
```

Expected: every template file. **If any template lacks the binding**, add it following `docs/packets/TEMPLATE_CONVENTIONS.md` (sorted `opCode` insertion, never appended beside a semantically-related entry) and re-run `tools/template-opcode-order-guard.sh` and `tools/template-duplicate-binding-guard.sh`. If all templates have it, no template change is needed (FR-8.4) — record that in the manifest.

- [ ] **Step 2: Write the coverage manifest**

Read the schema in `docs/packets/PROCESS.md` first, then create `docs/tasks/task-223-karma-scissors/coverage-manifest.yaml` declaring:

- the op this task touches: `cash/serverbound/USE_CASH_ITEM`, karma sub-body arm;
- the versions covered: every configured tenant version that binds `USE_CASH_ITEM` (enumerate them explicitly from `docs/packets/audits/STATUS.md:588`);
- an explicit note that this task adds a **sub-body arm behind an existing opcode** and changes **no already-verified column's wire layout** (FR-8.3);
- the three columns that row shows as unverified, named, so the critic can tell "not covered" from "silently dropped".

Follow the schema's field names exactly; do not invent keys.

- [ ] **Step 3: Run the packet completeness critic**

Dispatch the `packet-completeness-critic` agent against `task-223-karma-scissors`. It writes `docs/tasks/task-223-karma-scissors/completeness-critic.md`. Resolve every **CHANGED-BUT-UNCLAIMED** finding (either declare it in the manifest or revert the change) before proceeding.

- [ ] **Step 4: Run the full verification gate**

From the worktree root:

```bash
# 1-3: per-module test / vet / build
for m in libs/atlas-constants libs/atlas-packet libs/atlas-saga \
         services/atlas-data/atlas.com/data \
         services/atlas-inventory/atlas.com/inventory \
         services/atlas-channel/atlas.com/channel \
         services/atlas-saga-orchestrator/atlas.com/saga-orchestrator \
         services/atlas-trades/atlas.com/trades \
         services/atlas-merchant/atlas.com/merchant \
         services/atlas-login/atlas.com/login \
         services/atlas-cashshop/atlas.com/cashshop \
         services/atlas-consumables/atlas.com/consumables \
         services/atlas-storage/atlas.com/storage \
         services/atlas-query-aggregator/atlas.com/query-aggregator; do
  echo "=== $m ==="
  ( cd "$m" && go test -race ./... && go vet ./... && go build ./... ) || exit 1
done

# 5-8: repo-root guards
tools/redis-key-guard.sh
tools/goroutine-guard.sh
tools/lint.sh --check
```

Expected: every command exits 0. Run `tools/lint.sh` (no flags) first to fix formatting in place, then re-run `--check`.

> `tools/lint.sh --check` false-fails without nvm on PATH; source nvm first if it reports a missing node.

- [ ] **Step 5: Bake every service whose `go.mod` was touched**

No `go.mod` should have changed — every new package lives inside an existing module and every dependency is already required. **Verify that**, and bake only if it is false:

```bash
git diff --name-only main... | grep 'go\.mod$'
```

If the list is non-empty, run `docker buildx bake atlas-<svc>` from the worktree root for each affected service. `go build` against the workspace will NOT catch a missing `COPY libs/...` line in the shared Dockerfile — only the bake will.

- [ ] **Step 6: Commit**

```bash
git add docs/tasks/task-223-karma-scissors/
git commit -m "docs(task-223): packet coverage manifest and completeness-critic findings"
```

- [ ] **Step 7: Code review before PR**

Invoke `superpowers:requesting-code-review`. It dispatches `plan-adherence-reviewer` and `backend-guidelines-reviewer` (no frontend files changed). Both write to `docs/tasks/task-223-karma-scissors/audit.md`. Resolve findings before opening the PR.

---

## Acceptance Criteria Checklist

Cross-reference for the reviewer. Each maps to the task that satisfies it.

- [ ] `ItemUseKarmaScissors` round-trips at both `updateTime` positions with v83 and v95 golden bytes — Task 2
- [ ] `ClassificationKarmaScissors` defined; `GetCashSlotItemType`'s 552 branch uses it and returns unchanged values — Tasks 2, 11
- [ ] Karma and seal cash-slot-type resolvers differ on every configured version — Task 11
- [ ] `tradeAvailable` parses for all five categories; `karma` parses on cash; both are integers; present / absent-default-0 / Zakum Helmet `01002357` → 1 covered — Task 3
- [ ] `SetKarmaUsed(true)` → `KarmaUsed() == true` for an equip and a bundle in all seven services (equip-only in atlas-cashshop) — Task 4
- [ ] Karma-marking an equip leaves `FlagSpikes` exactly as it was — Tasks 1, 4, 14, 15
- [ ] The v83 bit split is confirmed and the symbols named — **done in the design phase** (design §1/OQ-2); Task 1 records the addresses in `flag.go`
- [ ] An eligible untradeable item is marked, exactly one scissors is consumed, and the mark reaches the client without relog — Tasks 7, 8, 12
- [ ] Each gate (0a–0e, 1–4) refuses server-side with no scissors consumed, no mutation, a distinct warn, and the client unlocked — Task 12
- [ ] A crafted packet naming a target the scissors' karma type does not cover is refused at both layers — Tasks 7, 12
- [ ] A karma-marked asset stages despite `FlagUntradeable` **and** despite `tradeBlock`; unmarked still refuses; unreadable item data still refuses — Task 13
- [ ] A karma-marked asset lists in a hired merchant shop — Task 15
- [ ] After a completed trade or merchant sale the item is untradeable for the receiver — Tasks 14, 15
- [ ] A cancelled trade leaves the mark intact — Task 14
- [ ] A saga failing after the mark compensates it away — Task 9
- [ ] `5520000` works wherever `USE_CASH_ITEM` binds; `5520001` works wherever its data exists — asserted data-driven, no version literal — Tasks 3, 12
- [ ] Pet-class targets are refused at both the channel and inventory layers with a distinct reason — Tasks 1, 7, 12
- [ ] No already-verified `USE_CASH_ITEM` matrix column regresses; a coverage manifest records the columns covered — Task 16
- [ ] Sealing Lock, Item Tag and expiration-extender arms are behaviorally unchanged — Tasks 11, 12 (the resolver-disjointness test is the guard)
- [ ] Full build & verification gate clean — Task 16

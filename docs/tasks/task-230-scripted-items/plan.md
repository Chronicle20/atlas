# Scripted Items Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the `SCRIPTED_ITEM` and `NPC_ITEM_USE_REQUEST` serverbound routes end to end — codecs across every version that has them, an item-keyed conversation family in `atlas-npc-conversations`, a conversation-first saga that consumes the item only once the dialogue opens, and reference content proving the path.

**Architecture:** Two new immutable codecs in `libs/atlas-packet/inventory/serverbound/` are routed by nine tenant socket templates into two new `atlas-channel` handlers. The `243` handler creates a two-step saga — `start_item_conversation` (non-self-completing, awaits a new `EVENT_TOPIC_NPC_CONVERSATION_STATUS` topic) then `destroy_asset_from_slot` — so an unauthored item costs the player nothing. The `239`/`545` handler reuses the existing NPC conversation and `open_npc_shop` machinery. A parser fix in `atlas-data` makes `spec/npc` resolve for the first time.

**Tech Stack:** Go 1.x (multi-module workspace via `go.work`), GORM + Postgres, Kafka (`segmentio/kafka-go` via `libs/atlas-kafka`), Redis (`libs/atlas-redis`), JSON:API via api2go, `libs/atlas-socket` for wire codecs.

**Spec:** [`design.md`](design.md) (PRD at [`prd.md`](prd.md); evidence at [`version-evidence.md`](version-evidence.md) and [`item-inventory.md`](item-inventory.md); grounding notes at [`context.md`](context.md))

## Global Constraints

- **Worktree.** All work happens in `.worktrees/task-230-scripted-items/` on branch `task-230-scripted-items`. Never edit the main repo.
- **Two ops, two bodies.** `ScriptedItem` = `updateTime uint32` → `source int16` → `itemId uint32`. `NpcItemUse` = `source int16` → `itemId uint32`, **no `updateTime`**. Never add `updateTime` to `NpcItemUse` by pattern-matching its sibling.
- **No version gates.** Both bodies are byte-identical on every version that has the op (design §1). A `MajorAtLeast` gate with no divergence to express is noise and must not be added.
- **Opcodes are config-resolved (DOM-25).** Never hard-code an opcode in Go. The values below are for template/registry files only.
  - `SCRIPTED_ITEM`: v72 `0x04D`, v79 `0x04C`, v83 `0x04E`, v84 `0x04E`, v87 `0x051`, v92 `0x055`, v95 `0x054`, jms `0x046`. Absent on v12/v48/v61.
  - `NPC_ITEM_USE_REQUEST`: v61 `0x066`, v72 `0x06E`, v79 `0x06D`, v83 `0x06F`, v84 `0x06F`, v87 `0x072`, v92 `0x07A`, v95 `0x07B`, jms `0x06A`. Absent on v12/v48.
- **Item ranges.** `243xxxx` = scripted item (`ClassificationConsumableScriptedItem`). `239xxxx` = remote NPC (`ClassificationConsumableRemoteNpc`). `545xxxx` = `ClassificationRemoteMerchant` (already exists). Item `3994225` is **out of scope** and must be rejected loudly on v95.
- **Excl-request contract.** Every rejection path calls `session.EnableActions`. The success path does **not** — the `destroy_asset_from_slot` inventory delta clears `m_bExclRequestSent` client-side. The compensation path **does**, because that destroy never happened. An outcome that warps must never be unlocked.
- **No `// TODO`, no stubs, no 501s** in any landed commit.
- **No literal home/absolute paths** in committed files — repo-relative only.
- **Preserve line endings**; do not normalize CRLF→LF as a side effect.
- **Verify, don't invent.** Every wire byte traces to a decompile line. Every item id traces to [`item-inventory.md`](item-inventory.md).

---

### Task 1: `atlas-data` reads `npc` and `runOnPickup` from the `spec` node

The `0243` family authors all three fields under `spec`; the reader reads two of them from `info`, so `Npc` is `0` for every scripted item today. `0239` genuinely authors under `info`, so this is spec-first-with-`info`-fallback, not a move.

**Files:**
- Modify: `services/atlas-data/atlas.com/data/consumable/reader.go:75-76` (remove), `:151-163` (add)
- Test: `services/atlas-data/atlas.com/data/consumable/reader_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces: `RestModel.Npc uint32`, `RestModel.Script string`, `RestModel.RunOnPickup bool` now carry correct values for `243xxxx` items. All three fields already exist on `RestModel` (`consumable/rest.go:74-76`); only the values change.

- [ ] **Step 1: Read the current reader to find the two anchors**

Run: `sed -n '30,40p;70,80p;145,165p' services/atlas-data/atlas.com/data/consumable/reader.go`

You are looking for three things:
- `i, err := cxml.ChildByName("info")` — the `info` node binding (~line 36)
- `m.Npc = uint32(i.GetIntegerWithDefault("npc", 0))` and `m.RunOnPickup = i.GetBool("runOnPickup", false)` (~lines 75-76)
- the `if err == nil && s != nil {` block guarding the `spec` node, containing `m.ConsumeOnPickup = s.GetBool("consumeOnPickup", false)` and `m.Script = s.GetString("script", "")`

- [ ] **Step 2: Write the failing test**

Create `services/atlas-data/atlas.com/data/consumable/reader_test.go`. Check first whether the file already exists (`ls services/atlas-data/atlas.com/data/consumable/`); if it does, append the tests to it instead of creating it.

The test needs an XML fixture in the shape the reader parses. Read an existing reader test elsewhere in `atlas-data` to copy the fixture-construction idiom — `grep -rln "ChildByName" services/atlas-data/atlas.com/data/*/[a-z]*_test.go` will find one. The behaviour to assert:

```go
// Verified against Item.wz/Consume/0243.img.xml: all 23 items in the 0243
// family author npc/script/runOnPickup under spec, and ZERO of them carry
// info/npc. 02430010 (openTreasure) is the only one with spec/runOnPickup=1.
// Contrast 02390001, which genuinely authors npc under info — hence the
// fallback rather than a move. See docs/tasks/task-230-scripted-items/item-inventory.md.
func TestConsumableReader_SpecNpcTakesPrecedence(t *testing.T) {
	// item authored spec-side only (the 0243 shape)
	m := readOne(t, `
		<imgdir name="02430010">
		  <imgdir name="info">
		    <int name="tradeBlock" value="1"/>
		    <int name="slotMax" value="100"/>
		  </imgdir>
		  <imgdir name="spec">
		    <string name="script" value="openTreasure"/>
		    <int name="npc" value="2040030"/>
		    <int name="runOnPickup" value="1"/>
		  </imgdir>
		</imgdir>`)

	if m.Npc != 2040030 {
		t.Errorf("Npc: got %d, want 2040030", m.Npc)
	}
	if m.Script != "openTreasure" {
		t.Errorf("Script: got %q, want %q", m.Script, "openTreasure")
	}
	if !m.RunOnPickup {
		t.Error("RunOnPickup: got false, want true")
	}
}

func TestConsumableReader_InfoNpcFallback(t *testing.T) {
	// item authored info-side only (the 0239 shape, verified on 02390001)
	m := readOne(t, `
		<imgdir name="02390001">
		  <imgdir name="info">
		    <int name="npc" value="9090002"/>
		    <int name="slotMax" value="100"/>
		  </imgdir>
		</imgdir>`)

	if m.Npc != 9090002 {
		t.Errorf("Npc: got %d, want 9090002", m.Npc)
	}
	if m.RunOnPickup {
		t.Error("RunOnPickup: got true, want false")
	}
}

func TestConsumableReader_NoNpcAnywhere(t *testing.T) {
	m := readOne(t, `
		<imgdir name="02000000">
		  <imgdir name="info">
		    <int name="slotMax" value="100"/>
		  </imgdir>
		  <imgdir name="spec">
		    <int name="hp" value="50"/>
		  </imgdir>
		</imgdir>`)

	if m.Npc != 0 {
		t.Errorf("Npc: got %d, want 0", m.Npc)
	}
}
```

Write the `readOne(t, xml string) RestModel` helper to match whatever fixture idiom the sibling reader test uses. Do **not** create a `*_testhelpers.go` file — the project bans them; put the helper in the test file itself.

- [ ] **Step 3: Run the test to verify it fails**

Run: `cd services/atlas-data/atlas.com/data && go test ./consumable/ -run 'TestConsumableReader_' -v`
Expected: `TestConsumableReader_SpecNpcTakesPrecedence` FAILs with `Npc: got 0, want 2040030`. The other two PASS already (they are regression guards).

- [ ] **Step 4: Delete the two `info`-side reads**

In `reader.go`, remove these two lines (~75-76):

```go
			m.Npc = uint32(i.GetIntegerWithDefault("npc", 0))
			m.RunOnPickup = i.GetBool("runOnPickup", false)
```

- [ ] **Step 5: Add the spec-first reads inside the `spec` guard**

Inside the `if err == nil && s != nil {` block, immediately after the `m.ConsumeOnPickup` line, add:

```go
				// npc and runOnPickup are authored under spec for the 0243
				// scripted-item family (verified: all 23 items in
				// Item.wz/Consume/0243.img.xml carry spec/npc, ZERO carry
				// info/npc) but under info for the 0239 remote-NPC family
				// (verified on 02390001). Read spec first, fall back to info,
				// so both families resolve. Same defect class as
				// consumeOnPickup above.
				m.Npc = uint32(s.GetIntegerWithDefault("npc", i.GetIntegerWithDefault("npc", 0)))
				m.RunOnPickup = s.GetBool("runOnPickup", i.GetBool("runOnPickup", false))
```

Now handle the case where the `spec` node is **absent**: the `0239` fallback must still fire. Immediately **before** the `spec` guard block, seed the defaults from `info`:

```go
			// Defaults from info; the spec block below overrides when the item
			// authors these fields spec-side (the 0243 family).
			m.Npc = uint32(i.GetIntegerWithDefault("npc", 0))
			m.RunOnPickup = i.GetBool("runOnPickup", false)
```

- [ ] **Step 6: Run the tests to verify they pass**

Run: `cd services/atlas-data/atlas.com/data && go test ./consumable/ -run 'TestConsumableReader_' -v`
Expected: all three PASS.

- [ ] **Step 7: Run the full module test suite**

Run: `cd services/atlas-data/atlas.com/data && go test -race ./... && go vet ./...`
Expected: clean.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-data/atlas.com/data/consumable/reader.go services/atlas-data/atlas.com/data/consumable/reader_test.go
git commit -m "fix(atlas-data): read consumable npc/runOnPickup from spec node with info fallback

The 0243 scripted-item family authors npc, script, and runOnPickup under the
item's spec node; the reader read npc and runOnPickup from info, so Npc
resolved to 0 for all 23 items in the family. The 0239 remote-NPC family
genuinely authors npc under info, so this is spec-first with an info fallback
rather than a move. Same defect class as the already-fixed consumeOnPickup.

Re-ingest is required for the values to appear; a parser fix alone leaves
every tenant at npc = 0."
```

---

### Task 2: Item classifications for the `239` and `243` families

DOM-21 requires handlers classify via `item.GetClassification` rather than open-coding `/10000`.

**Files:**
- Modify: `libs/atlas-constants/item/constants.go` (the `Consumable*` block, ~lines 32-46)
- Test: `libs/atlas-constants/item/constants_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces: `item.ClassificationConsumableRemoteNpc` (= `Classification(239)`) and `item.ClassificationConsumableScriptedItem` (= `Classification(243)`), consumed by Tasks 15 and 16.

- [ ] **Step 1: Read the surrounding block to find the insertion points**

Run: `sed -n '30,50p' libs/atlas-constants/item/constants.go`

You will see the `Consumable*` block ending at `ClassificationConsumableMonsterCard = Classification(238)` (line 46). Note the block is sorted ascending by numeric value and the `=` signs are gofmt-aligned within the const group — adding a longer identifier will re-align neighbours, which is expected and correct.

- [ ] **Step 2: Write the failing test**

Append to `libs/atlas-constants/item/constants_test.go` (create it if absent):

```go
// The two item families task-230 routes. 243xxxx is the scripted-item family
// (Consume/0243.img — 23 items, each with spec/script + spec/npc); 239xxxx is
// the remote-NPC family (verified names "Athena Pierce's Marble",
// "Traveling Tommy's Ticket"), which resolves an NPC from info/npc and opens
// that NPC's own shop or conversation.
func TestScriptedItemAndRemoteNpcClassifications(t *testing.T) {
	if got := GetClassification(Id(2430008)); got != ClassificationConsumableScriptedItem {
		t.Errorf("2430008: got %d, want %d", got, ClassificationConsumableScriptedItem)
	}
	if got := GetClassification(Id(2390001)); got != ClassificationConsumableRemoteNpc {
		t.Errorf("2390001: got %d, want %d", got, ClassificationConsumableRemoteNpc)
	}
	if ClassificationConsumableScriptedItem == ClassificationConsumableRemoteNpc {
		t.Fatal("the two classifications must be distinct")
	}
}
```

Check the exact spelling of the id type and helper first: `grep -n "func GetClassification" libs/atlas-constants/item/constants.go`. Adjust `Id(...)` to whatever that function accepts.

- [ ] **Step 3: Run the test to verify it fails**

Run: `cd libs/atlas-constants && go test ./item/ -run TestScriptedItemAndRemoteNpcClassifications -v`
Expected: FAIL — `undefined: ClassificationConsumableScriptedItem`.

- [ ] **Step 4: Add the two constants at their sorted positions**

In `libs/atlas-constants/item/constants.go`, in the `Consumable*` block, after `ClassificationConsumableMonsterCard = Classification(238)`:

```go
	// 239xxxx — remote-NPC summons. The item names an NPC in its info/npc node
	// and opens that NPC's own shop or conversation from anywhere
	// (CWvsContext::SendSelectNpcItemUseRequest). Distinct from 243: the item
	// does not carry its own dialogue.
	ClassificationConsumableRemoteNpc = Classification(239)
	// 243xxxx — scripted items. The item carries its own dialogue, keyed by
	// item id, rendered with the avatar named in its spec/npc node
	// (CWvsContext::SendScriptRunItemRequest, gated client-side on
	// itemId/10000 == 243).
	ClassificationConsumableScriptedItem = Classification(243)
```

- [ ] **Step 5: Run the test to verify it passes**

Run: `cd libs/atlas-constants && go test ./item/ -run TestScriptedItemAndRemoteNpcClassifications -v`
Expected: PASS.

- [ ] **Step 6: Run the module suite and gofmt**

Run: `cd libs/atlas-constants && go test -race ./... && go vet ./... && gofmt -l .`
Expected: tests and vet clean; `gofmt -l` prints nothing. If it prints `item/constants.go`, run `gofmt -w item/constants.go` (the const-block alignment shifted) and re-run.

- [ ] **Step 7: Commit**

```bash
git add libs/atlas-constants/item/constants.go libs/atlas-constants/item/constants_test.go
git commit -m "feat(atlas-constants): add remote-npc (239) and scripted-item (243) classifications

Both families are routed by task-230. DOM-21 requires handlers classify via
GetClassification rather than open-coding itemId/10000."
```

---

### Task 3: `ScriptedItem` serverbound codec

**Files:**
- Create: `libs/atlas-packet/inventory/serverbound/scripted_item.go`
- Create: `libs/atlas-packet/inventory/serverbound/scripted_item_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces: package `serverbound` (import path `github.com/Chronicle20/atlas/libs/atlas-packet/inventory/serverbound`) gains:
  - `const ScriptedItemHandle = "ScriptedItemHandle"` — the template `handler` key (Task 6) and the `handlerMap` key (Task 15).
  - `type ScriptedItem struct` with methods `UpdateTime() uint32`, `Source() int16`, `ItemId() uint32`, `Operation() string`, `String() string`, `Encode(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte`, `Decode(logrus.FieldLogger, context.Context) func(*request.Reader, map[string]interface{})` (pointer receiver on `Decode`).
  - `func NewScriptedItem(updateTime uint32, source int16, itemId uint32) ScriptedItem`
  - Matrix/evidence path: `inventory/serverbound/InventoryScriptedItem`.

- [ ] **Step 1: Read the structural model**

Run: `cat libs/atlas-packet/inventory/serverbound/lottery_item_use.go libs/atlas-packet/inventory/serverbound/lottery_item_use_test.go`

That file is the exact idiom to follow: package-level `Handle` const, `// packet-audit:fname` marker in the doc comment, private fields, value-receiver getters, `Operation()`, `String()`, value-receiver `Encode`, pointer-receiver `Decode`.

- [ ] **Step 2: Write the failing test**

Create `libs/atlas-packet/inventory/serverbound/scripted_item_test.go`:

```go
package serverbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// Byte round-trip over the invariant serverbound body. The client body is
// byte-identical on every version that carries the opcode — a full sweep of all
// ten IDBs found no divergence (task-230 design §1.1), so no version gating is
// required or permitted.
//
//	Encode4(get_update_time())   // uint32 update time
//	Encode2(nPOS)                // int16  source inventory slot
//	Encode4(nItemID)             // int32  item template id
//
// Gated client-side on nItemID / 10000 == 243 under CanSendExclRequest(500, 0).
// v83+ additionally guards on CWvsContext::IsAbleToConsume, which v72/v79 lack;
// that is a client-side convenience check the server must not rely on.
// v95 alone also whitelists nItemID == 3994225 (an Install/Setup item) — out of
// scope per design D-3 and rejected server-side.
//
// The op is ABSENT from gms_v12, gms_v48, and gms_v61 (design §1.1 absence
// evidence: dense Send*ItemUseRequest export sets with no SendScriptRunItemRequest).
//
// packet-audit:verify markers are added per cell in the verification task; this
// round-trip alone is NOT a verification.
func TestScriptedItemRoundTrip(t *testing.T) {
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ScriptedItem{updateTime: 0x1A2B3C4D, source: 7, itemId: 2430008}
			output := ScriptedItem{}
			test.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.UpdateTime() != input.UpdateTime() {
				t.Errorf("updateTime: got %v, want %v", output.UpdateTime(), input.UpdateTime())
			}
			if output.Source() != input.Source() {
				t.Errorf("source: got %v, want %v", output.Source(), input.Source())
			}
			if output.ItemId() != input.ItemId() {
				t.Errorf("itemId: got %v, want %v", output.ItemId(), input.ItemId())
			}
		})
	}
}

// The field ORDER is the defect this guards. The sibling NpcItemUse codec has
// no leading updateTime; reading these two files side by side and copying the
// wrong prologue misaligns every subsequent field. Assert the exact bytes.
func TestScriptedItemWireLayout(t *testing.T) {
	ctx := test.CreateContext("GMS", 83, 1)
	m := ScriptedItem{updateTime: 0x01020304, source: 0x0506, itemId: 0x0708090A}
	got := m.Encode(test.Logger(), ctx)(nil)
	want := []byte{
		0x04, 0x03, 0x02, 0x01, // updateTime, little-endian uint32
		0x06, 0x05, //             source, little-endian int16
		0x0A, 0x09, 0x08, 0x07, //  itemId, little-endian uint32
	}
	if len(got) != len(want) {
		t.Fatalf("length: got %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("byte %d: got 0x%02X, want 0x%02X (full: %v)", i, got[i], want[i], got)
		}
	}
}
```

Check the logger helper's name first: `grep -n "func Logger\|func CreateContext\|func RoundTrip" libs/atlas-packet/test/*.go`. If there is no `test.Logger()`, use whatever the sibling tests use (commonly a `logrus.New()` or a test package helper) — read `lottery_item_use_test.go` and any test in the package that calls `Encode` directly.

- [ ] **Step 3: Run the test to verify it fails**

Run: `cd libs/atlas-packet && go test ./inventory/serverbound/ -run TestScriptedItem -v`
Expected: FAIL — `undefined: ScriptedItem`.

- [ ] **Step 4: Write the codec**

Create `libs/atlas-packet/inventory/serverbound/scripted_item.go`:

```go
package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const ScriptedItemHandle = "ScriptedItemHandle"

// ScriptedItem - the 243xxxx scripted-item use request. The item carries its
// own dialogue, keyed by item id and rendered with the avatar named in the
// item's WZ spec/npc node.
//
// Body is invariant across every version that carries the opcode (v72 through
// jms_v185) — a full sweep of all ten IDBs found no divergence, so there is no
// version gating here deliberately:
//
//	Encode4(get_update_time())   // uint32 update time
//	Encode2(nPOS)                // int16  source inventory slot
//	Encode4(nItemID)             // int32  item template id
//
// Contrast the sibling NpcItemUse in this package, which has NO leading
// updateTime. Client-side gate: nItemID / 10000 == 243 under
// CanSendExclRequest(500, 0). v83+ additionally calls
// CWvsContext::IsAbleToConsume; v72/v79 do not, so the server performs its own
// ownership and quantity validation on every version.
//
// Absent from gms_v12, gms_v48, gms_v61.
//
// packet-audit:fname CWvsContext::SendScriptRunItemRequest
type ScriptedItem struct {
	updateTime uint32
	source     int16
	itemId     uint32
}

func NewScriptedItem(updateTime uint32, source int16, itemId uint32) ScriptedItem {
	return ScriptedItem{updateTime: updateTime, source: source, itemId: itemId}
}

func (m ScriptedItem) UpdateTime() uint32 { return m.updateTime }
func (m ScriptedItem) Source() int16      { return m.source }
func (m ScriptedItem) ItemId() uint32     { return m.itemId }

func (m ScriptedItem) Operation() string {
	return ScriptedItemHandle
}

func (m ScriptedItem) String() string {
	return fmt.Sprintf("updateTime [%d], source [%d], itemId [%d]", m.updateTime, m.source, m.itemId)
}

func (m ScriptedItem) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.updateTime)
		w.WriteInt16(m.source)
		w.WriteInt(m.itemId)
		return w.Bytes()
	}
}

func (m *ScriptedItem) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.updateTime = r.ReadUint32()
		m.source = r.ReadInt16()
		m.itemId = r.ReadUint32()
	}
}
```

- [ ] **Step 5: Run the tests to verify they pass**

Run: `cd libs/atlas-packet && go test ./inventory/serverbound/ -run TestScriptedItem -v`
Expected: both PASS across every variant.

If `TestScriptedItemWireLayout` fails on byte order, check `response.Writer`'s `WriteInt` width — confirm with `grep -n "func (w \*Writer) WriteInt\b" -A 6 libs/atlas-socket/response/*.go` that it writes 4 little-endian bytes. Fix the expectation to match the verified writer, not the other way round.

- [ ] **Step 6: Run the module suite**

Run: `cd libs/atlas-packet && go test -race ./... && go vet ./...`
Expected: clean.

- [ ] **Step 7: Commit**

```bash
git add libs/atlas-packet/inventory/serverbound/scripted_item.go libs/atlas-packet/inventory/serverbound/scripted_item_test.go
git commit -m "feat(atlas-packet): add ScriptedItem serverbound codec

CWvsContext::SendScriptRunItemRequest, the 243xxxx scripted-item use route.
Body updateTime(uint32) + slot(int16) + itemId(uint32), byte-identical on every
version that carries the opcode (v72..jms_v185, verified by a full ten-IDB
sweep), so no version gating. Absent on v12/v48/v61."
```

---

### Task 4: `NpcItemUse` serverbound codec

**Files:**
- Create: `libs/atlas-packet/inventory/serverbound/npc_item_use.go`
- Create: `libs/atlas-packet/inventory/serverbound/npc_item_use_test.go`

**Interfaces:**
- Consumes: nothing (deliberately independent of Task 3 — no shared helper, so a mistake in one cannot propagate).
- Produces: package `serverbound` gains:
  - `const NpcItemUseHandle = "NpcItemUseHandle"`
  - `type NpcItemUse struct` with `Source() int16`, `ItemId() uint32`, `Operation() string`, `String() string`, `Encode(...)`, `Decode(...)` (pointer receiver).
  - `func NewNpcItemUse(source int16, itemId uint32) NpcItemUse`
  - Matrix/evidence path: `inventory/serverbound/InventoryNpcItemUse`.

- [ ] **Step 1: Write the failing test**

Create `libs/atlas-packet/inventory/serverbound/npc_item_use_test.go`:

```go
package serverbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// Byte round-trip over the invariant serverbound body. Identical on all nine
// versions that carry the opcode (v61 through jms_v185), so no version gating.
//
//	Encode2(nPOS)                // int16  source inventory slot
//	Encode4(nItemID)             // int32  item template id
//
// THERE IS NO updateTime. The sibling ScriptedItem codec in this package leads
// with one; copying its prologue here misaligns every subsequent read. This is
// the single most likely defect in this pair.
//
// Client gate on every version:
//
//	(nItemID / 10000 == 545 || nItemID / 10000 == 239) && CanSendExclRequest(200, 0)
//
// plus two refusal arms that emit a chat message and send nothing (field flag
// bit 18 set; a CUniqueModeless dialog already open).
//
// ABSENT from gms_v12 and gms_v48 — confirmed by instruction scan for
// `cmp ,545` / `cmp ,239`, not by a missing symbol.
func TestNpcItemUseRoundTrip(t *testing.T) {
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NpcItemUse{source: 3, itemId: 2390001}
			output := NpcItemUse{}
			test.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Source() != input.Source() {
				t.Errorf("source: got %v, want %v", output.Source(), input.Source())
			}
			if output.ItemId() != input.ItemId() {
				t.Errorf("itemId: got %v, want %v", output.ItemId(), input.ItemId())
			}
		})
	}
}

// Guards the no-updateTime invariant explicitly: the encoded frame must be
// exactly 6 bytes. A stray leading updateTime makes it 10.
func TestNpcItemUseWireLayoutHasNoUpdateTime(t *testing.T) {
	ctx := test.CreateContext("GMS", 83, 1)
	m := NpcItemUse{source: 0x0102, itemId: 0x03040506}
	got := m.Encode(test.Logger(), ctx)(nil)
	want := []byte{
		0x02, 0x01, //             source, little-endian int16
		0x06, 0x05, 0x04, 0x03, // itemId, little-endian uint32
	}
	if len(got) != 6 {
		t.Fatalf("frame length: got %d, want 6 — a leading updateTime would make it 10 (%v)", len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("byte %d: got 0x%02X, want 0x%02X (full: %v)", i, got[i], want[i], got)
		}
	}
}
```

Use the same logger helper you resolved in Task 3 Step 2.

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd libs/atlas-packet && go test ./inventory/serverbound/ -run TestNpcItemUse -v`
Expected: FAIL — `undefined: NpcItemUse`.

- [ ] **Step 3: Write the codec**

Create `libs/atlas-packet/inventory/serverbound/npc_item_use.go`:

```go
package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const NpcItemUseHandle = "NpcItemUseHandle"

// NpcItemUse - the remote-NPC item use request, covering the 239xxxx
// (remote-NPC summon) and 545xxxx (remote merchant) families. The item names an
// NPC and opens that NPC's own shop or conversation from anywhere; unlike
// ScriptedItem the item carries no dialogue of its own.
//
// Body is invariant across all nine versions that carry the opcode (v61 through
// jms_v185):
//
//	Encode2(nPOS)                // int16  source inventory slot
//	Encode4(nItemID)             // int32  item template id
//
// THERE IS NO LEADING updateTime — contrast ScriptedItem in this same package,
// which has one. Client gate:
// (nItemID/10000 == 545 || nItemID/10000 == 239) && CanSendExclRequest(200, 0),
// note the 200 ms window against ScriptedItem's 500 ms. Two client-side refusal
// arms send nothing at all: field flag bit 18 (0x40000) set, and a
// CUniqueModeless dialog already open.
//
// Absent from gms_v12 and gms_v48.
//
// packet-audit:fname CWvsContext::SendSelectNpcItemUseRequest
type NpcItemUse struct {
	source int16
	itemId uint32
}

func NewNpcItemUse(source int16, itemId uint32) NpcItemUse {
	return NpcItemUse{source: source, itemId: itemId}
}

func (m NpcItemUse) Source() int16  { return m.source }
func (m NpcItemUse) ItemId() uint32 { return m.itemId }

func (m NpcItemUse) Operation() string {
	return NpcItemUseHandle
}

func (m NpcItemUse) String() string {
	return fmt.Sprintf("source [%d], itemId [%d]", m.source, m.itemId)
}

func (m NpcItemUse) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt16(m.source)
		w.WriteInt(m.itemId)
		return w.Bytes()
	}
}

func (m *NpcItemUse) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.source = r.ReadInt16()
		m.itemId = r.ReadUint32()
	}
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `cd libs/atlas-packet && go test ./inventory/serverbound/ -run TestNpcItemUse -v`
Expected: both PASS.

- [ ] **Step 5: Run the module suite**

Run: `cd libs/atlas-packet && go test -race ./... && go vet ./...`
Expected: clean.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-packet/inventory/serverbound/npc_item_use.go libs/atlas-packet/inventory/serverbound/npc_item_use_test.go
git commit -m "feat(atlas-packet): add NpcItemUse serverbound codec

CWvsContext::SendSelectNpcItemUseRequest, covering the 239xxxx remote-NPC and
545xxxx remote-merchant families. Body slot(int16) + itemId(int32) with NO
leading updateTime — the sibling ScriptedItem codec has one, and copying its
prologue here would misalign every field. Invariant across v61..jms_v185."
```

---

### Task 5: Registry entries and `packet-audit` fname linkage

Five registry entries are missing, and neither op's fname is keyed to a codec in the audit tool. Without the fname cases, all 17 verification attempts in Task 18 fail to resolve a codec and no cell promotes.

**Files:**
- Modify: `docs/packets/registry/gms_v61.yaml` (add `NPC_ITEM_USE_REQUEST` @ 102)
- Modify: `docs/packets/registry/gms_v72.yaml` (add `SCRIPTED_ITEM` @ 77, `NPC_ITEM_USE_REQUEST` @ 110)
- Modify: `docs/packets/registry/gms_v79.yaml` (add `SCRIPTED_ITEM` @ 76, `NPC_ITEM_USE_REQUEST` @ 109)
- Modify: `tools/packet-audit/cmd/run.go` (the `candidatesFromFName` switch, near the `SendLotteryItemUseRequest` case at ~`:2205`)

**Interfaces:**
- Consumes: the struct names from Tasks 3 and 4 — `ScriptedItem` and `NpcItemUse`, both `pkg: "inventory"`, `dir: serverbound`.
- Produces: `candidatesFromFName("CWvsContext::SendScriptRunItemRequest")` → `[]candidate{{name: "ScriptedItem", dir: csvpkg.DirServerbound, pkg: "inventory"}}`; same shape for `SendSelectNpcItemUseRequest` → `NpcItemUse`. Consumed by Task 18.

- [ ] **Step 1: Read an existing entry to copy the exact YAML shape**

Run: `sed -n '2243,2250p;2459,2466p' docs/packets/registry/gms_v83.yaml`

Expected shape (note `opcode` is **decimal**, and entries are sorted ascending by opcode within the file):

```yaml
- op: SCRIPTED_ITEM
  direction: serverbound
  opcode: 78
  fname: CWvsContext::SendScriptRunItemRequest
  provenance: csv-import
```

- [ ] **Step 2: Find the sorted insertion points in each of the three registries**

Run:

```bash
for v in 61 72 79; do
  echo "=== gms_v$v ==="
  grep -n "^  opcode: \(7[5-9]\|10[0-9]\|11[0-2]\)$" docs/packets/registry/gms_v$v.yaml
done
```

For each target opcode (v61: 102; v72: 77 and 110; v79: 76 and 109), locate the entry with the next-highest opcode and insert immediately **before** it. Read ~6 lines of context around each hit to confirm you are at an entry boundary (a line starting `- op:`), not mid-entry.

- [ ] **Step 3: Add the five entries**

`docs/packets/registry/gms_v61.yaml` — at the sorted position for opcode 102:

```yaml
- op: NPC_ITEM_USE_REQUEST
  direction: serverbound
  opcode: 102
  fname: CWvsContext::SendSelectNpcItemUseRequest
  provenance: ida-decompile
```

`docs/packets/registry/gms_v72.yaml` — two entries, each at its sorted position:

```yaml
- op: SCRIPTED_ITEM
  direction: serverbound
  opcode: 77
  fname: CWvsContext::SendScriptRunItemRequest
  provenance: ida-decompile
```

```yaml
- op: NPC_ITEM_USE_REQUEST
  direction: serverbound
  opcode: 110
  fname: CWvsContext::SendSelectNpcItemUseRequest
  provenance: ida-decompile
```

`docs/packets/registry/gms_v79.yaml` — two entries, each at its sorted position:

```yaml
- op: SCRIPTED_ITEM
  direction: serverbound
  opcode: 76
  fname: CWvsContext::SendScriptRunItemRequest
  provenance: ida-decompile
```

```yaml
- op: NPC_ITEM_USE_REQUEST
  direction: serverbound
  opcode: 109
  fname: CWvsContext::SendSelectNpcItemUseRequest
  provenance: ida-decompile
```

Before writing `provenance: ida-decompile`, confirm that value is one the tooling accepts:
`grep -rho "provenance: .*" docs/packets/registry/*.yaml | sort -u`. If `ida-decompile` is not in the set, use whichever existing value denotes a fresh decompile; if the only value present is `csv-import`, use that and note the addresses in the commit message instead.

- [ ] **Step 4: Add the two `candidatesFromFName` cases**

In `tools/packet-audit/cmd/run.go`, immediately after the `SendLotteryItemUseRequest` case (~`:2205`):

```go
	// Scripted items (task-230). Struct ScriptedItem in inventory/serverbound;
	// body updateTime(uint32) + slot(int16) + itemId(int32). Opcode exists v72
	// through jms_v185; v12/v48/v61 lack the sender entirely.
	case "CWvsContext::SendScriptRunItemRequest":
		return []candidate{{name: "ScriptedItem", dir: csvpkg.DirServerbound, pkg: "inventory"}}
	// Remote-NPC item use (task-230), covering the 239xxxx and 545xxxx
	// families. Struct NpcItemUse in inventory/serverbound; body slot(int16) +
	// itemId(int32) with NO leading updateTime. Opcode exists v61 through
	// jms_v185; v12/v48 lack it.
	case "CWvsContext::SendSelectNpcItemUseRequest":
		return []candidate{{name: "NpcItemUse", dir: csvpkg.DirServerbound, pkg: "inventory"}}
```

- [ ] **Step 5: Verify the tool still builds and its own tests pass**

Run: `cd tools/packet-audit && go build ./... && go test ./...`
Expected: clean. Note `cmd/dispatcher_lint_test.go:17` defines a test-local `candidatesFromFName` — if that test mirrors the production switch, it may need the same two cases; the compiler will not tell you, so read it: `sed -n '1,40p' tools/packet-audit/cmd/dispatcher_lint_test.go`. Add the cases there too if it is a mirror.

- [ ] **Step 6: Verify the registries still parse and the fname resolves**

Run:

```bash
go run ./tools/packet-audit fname-doc --check
go run ./tools/packet-audit matrix --check
```

Expected: `fname-doc --check` exits 0. `matrix --check` will now report the five new registry ops as **unimplemented cells** (the codecs exist but no evidence is pinned yet) — that is expected at this point and is resolved in Task 18. If it fails on anything else (a 🟥 conflict, an orphan, a parse error), fix that before continuing.

Note: do **not** regenerate STATUS.md/status.json here. The matrix `toolSha` reads git HEAD, so regeneration is the branch's final commit (Task 18).

- [ ] **Step 7: Commit**

```bash
git add docs/packets/registry/gms_v61.yaml docs/packets/registry/gms_v72.yaml docs/packets/registry/gms_v79.yaml tools/packet-audit/cmd/run.go
git commit -m "feat(packets): register SCRIPTED_ITEM/NPC_ITEM_USE_REQUEST on legacy versions

The matrix recorded n-a for SCRIPTED_ITEM on v72/v79 and for
NPC_ITEM_USE_REQUEST on v61/v72/v79, but all five senders exist in their
binaries:

  SCRIPTED_ITEM         v72 0x9044d8 0x04D   v79 0x955840 0x04C
  NPC_ITEM_USE_REQUEST  v61 0x83778d 0x066   v72 0x90a5ac 0x06E   v79 0x95b96c 0x06D

Also keys both fnames to their codecs in candidatesFromFName, without which no
cell can be verified."
```

---

### Task 6: Bind both handlers in all nine seed templates

Seventeen entries across nine templates. Every target opcode was checked and is free — no entry displaces an existing binding.

**Files:**
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_61_1.json` (1 entry)
- Modify: `template_gms_72_1.json`, `template_gms_79_1.json`, `template_gms_83_1.json`, `template_gms_84_1.json`, `template_gms_87_1.json`, `template_gms_92_1.json`, `template_gms_95_1.json`, `template_jms_185_1.json` (2 entries each)

**Interfaces:**
- Consumes: `ScriptedItemHandle` and `NpcItemUseHandle` string constants from Tasks 3 and 4.
- Produces: the socket dispatcher routes each version's opcode to those handler names. Consumed at runtime by Tasks 15 and 16.

- [ ] **Step 1: Confirm every target slot is still free**

Run:

```bash
python3 - <<'PY'
import json
T = 'services/atlas-configurations/seed-data/templates/'
targets = {
 'template_gms_61_1.json':  [('NpcItemUseHandle',0x66)],
 'template_gms_72_1.json':  [('ScriptedItemHandle',0x4D),('NpcItemUseHandle',0x6E)],
 'template_gms_79_1.json':  [('ScriptedItemHandle',0x4C),('NpcItemUseHandle',0x6D)],
 'template_gms_83_1.json':  [('ScriptedItemHandle',0x4E),('NpcItemUseHandle',0x6F)],
 'template_gms_84_1.json':  [('ScriptedItemHandle',0x4E),('NpcItemUseHandle',0x6F)],
 'template_gms_87_1.json':  [('ScriptedItemHandle',0x51),('NpcItemUseHandle',0x72)],
 'template_gms_92_1.json':  [('ScriptedItemHandle',0x55),('NpcItemUseHandle',0x7A)],
 'template_gms_95_1.json':  [('ScriptedItemHandle',0x54),('NpcItemUseHandle',0x7B)],
 'template_jms_185_1.json': [('ScriptedItemHandle',0x46),('NpcItemUseHandle',0x6A)],
}
for f, ts in targets.items():
    hs = json.load(open(T+f))['socket']['handlers']
    used = {int(h['opCode'],16): h['handler'] for h in hs}
    for name, oc in ts:
        print(f, name, hex(oc), ('OCCUPIED by '+used[oc]) if oc in used else 'FREE')
PY
```

Expected: all 17 lines say `FREE`. If any says `OCCUPIED`, **stop** and report — the design's opcode is wrong or the template drifted.

- [ ] **Step 2: Read the entry shape you are copying**

Run: `python3 -c "import json;print(json.dumps([h for h in json.load(open('services/atlas-configurations/seed-data/templates/template_gms_83_1.json'))['socket']['handlers'] if h['handler']=='ShopScannerItemUseHandle'],indent=1))"`

Expected:

```json
[{
  "opCode": "0x53",
  "validator": "LoggedInValidator",
  "handler": "ShopScannerItemUseHandle",
  "fname": "CWvsContext::SendShopScannerItemUseRequest",
  "services": ["channel"]
}]
```

`opCode` is a hex **string**. `validator` is mandatory — a handler with a missing validator is silently dropped at dispatch. `services` is `["channel"]` for both new handlers.

- [ ] **Step 3: Insert the entries by hand, one file at a time**

Edit each template with the `Edit` tool, inserting each new object at its **sorted position** in the `socket.handlers` array — immediately before the first entry whose numeric `opCode` exceeds the new one. Never append next to a semantically-related entry; `tools/template-opcode-order-guard.sh` enforces strictly ascending order and will reject it.

The 17 entries, by file:

| File | Entry |
|---|---|
| `template_gms_61_1.json` | `{"opCode": "0x66", "validator": "LoggedInValidator", "handler": "NpcItemUseHandle", "fname": "CWvsContext::SendSelectNpcItemUseRequest", "services": ["channel"]}` |
| `template_gms_72_1.json` | `"0x4D"` → `ScriptedItemHandle` / `CWvsContext::SendScriptRunItemRequest`; `"0x6E"` → `NpcItemUseHandle` / `CWvsContext::SendSelectNpcItemUseRequest` |
| `template_gms_79_1.json` | `"0x4C"` → `ScriptedItemHandle`; `"0x6D"` → `NpcItemUseHandle` |
| `template_gms_83_1.json` | `"0x4E"` → `ScriptedItemHandle`; `"0x6F"` → `NpcItemUseHandle` |
| `template_gms_84_1.json` | `"0x4E"` → `ScriptedItemHandle`; `"0x6F"` → `NpcItemUseHandle` |
| `template_gms_87_1.json` | `"0x51"` → `ScriptedItemHandle`; `"0x72"` → `NpcItemUseHandle` |
| `template_gms_92_1.json` | `"0x55"` → `ScriptedItemHandle`; `"0x7A"` → `NpcItemUseHandle` |
| `template_gms_95_1.json` | `"0x54"` → `ScriptedItemHandle`; `"0x7B"` → `NpcItemUseHandle` |
| `template_jms_185_1.json` | `"0x46"` → `ScriptedItemHandle`; `"0x6A"` → `NpcItemUseHandle` |

Every entry uses `"validator": "LoggedInValidator"` and `"services": ["channel"]`. The `fname` is `CWvsContext::SendScriptRunItemRequest` for `ScriptedItemHandle` and `CWvsContext::SendSelectNpcItemUseRequest` for `NpcItemUseHandle`, on every version.

Do **not** write a shell loop to patch these — per the project's editing conventions, use per-file `Edit` calls. Match the surrounding file's indentation exactly.

- [ ] **Step 4: Verify the count and ordering programmatically**

Run:

```bash
python3 - <<'PY'
import json, sys
T = 'services/atlas-configurations/seed-data/templates/'
files = ['template_gms_61_1.json','template_gms_72_1.json','template_gms_79_1.json',
         'template_gms_83_1.json','template_gms_84_1.json','template_gms_87_1.json',
         'template_gms_92_1.json','template_gms_95_1.json','template_jms_185_1.json']
total = 0
bad = False
for f in files:
    hs = json.load(open(T+f))['socket']['handlers']
    ocs = [int(h['opCode'],16) for h in hs]
    if ocs != sorted(ocs) or len(set(ocs)) != len(ocs):
        print('ORDER/DUP FAIL', f); bad = True
    n = sum(1 for h in hs if h['handler'] in ('ScriptedItemHandle','NpcItemUseHandle'))
    missing = [h['handler'] for h in hs
               if h['handler'] in ('ScriptedItemHandle','NpcItemUseHandle') and not h.get('validator')]
    if missing:
        print('MISSING VALIDATOR', f, missing); bad = True
    print(f, 'new entries:', n)
    total += n
print('TOTAL', total, '(want 17)')
sys.exit(1 if bad or total != 17 else 0)
PY
```

Expected: `TOTAL 17`, no `FAIL`/`MISSING` lines, exit 0.

- [ ] **Step 5: Run the template guards**

Run:

```bash
tools/template-opcode-order-guard.sh
tools/template-duplicate-binding-guard.sh
tools/template-movement-types-guard.sh
```

Expected: all three exit 0. The movement-types guard is unrelated to this change but any template edit trips its scope, so it must pass.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-configurations/seed-data/templates/
git commit -m "feat(templates): route SCRIPTED_ITEM and NPC_ITEM_USE_REQUEST in nine templates

17 handler bindings: ScriptedItemHandle on v72..jms_v185, NpcItemUseHandle on
v61..jms_v185. Every target opcode was verified free before insertion, and each
entry sits at its sorted position per the opcode-order guard. All carry
LoggedInValidator — a handler with a missing validator is silently dropped at
dispatch.

Live tenants do not inherit seed-template edits; already-provisioned tenants
need these entries PATCHed into their live socket configuration."
```

---

### Task 7: The `conversation/item/` family in `atlas-npc-conversations`

Packaged like `conversation/quest/` (own table, own REST resource, own seeder subdomain, own `MigrateTable`) but **shaped** like `conversation/npc/` (single state machine, `FindState` directly on the model). An item has one entry point, so the quest family's dual start/end pair would leave `endStateMachine` permanently `nil`.

**Files:**
- Create: `services/atlas-npc-conversations/atlas.com/npc/conversation/item/model.go`
- Create: `.../conversation/item/entity.go`
- Create: `.../conversation/item/rest.go`
- Create: `.../conversation/item/provider.go`
- Create: `.../conversation/item/administrator.go`
- Create: `.../conversation/item/processor.go`
- Create: `.../conversation/item/resource.go`
- Create: `.../conversation/item/subdomain.go`
- Create: `.../conversation/item/groups.go`
- Create: `.../conversation/item/model_test.go`, `entity_test.go`, `processor_test.go`, `subdomain_test.go`
- Modify: `services/atlas-npc-conversations/atlas.com/npc/main.go:60` (migrations), `:100-110` (route initializers)
- Modify: `tools/catalog-lint/subdomains.go` (add the `npc-conversations/items` rule)

**Interfaces:**
- Consumes: `conversation.StateModel`, `conversation.StateContainer` (`conversation/model.go:12-17`).
- Produces, in package `item` (import `atlas-npc-conversations/conversation/item`):
  - `type Model struct` — unexported fields `id uuid.UUID`, `itemId uint32`, `npcId uint32`, `scriptName string`, `startState string`, `states []conversation.StateModel`, `createdAt`, `updatedAt time.Time`. Methods `Id() uuid.UUID`, `ItemId() uint32`, `NpcId() uint32`, `ScriptName() string`, `StartState() string`, `States() []conversation.StateModel`, `CreatedAt() time.Time`, `UpdatedAt() time.Time`, `FindState(stateId string) (conversation.StateModel, error)`.
  - `func NewBuilder() *Builder` with `SetId`, `SetItemId`, `SetNpcId`, `SetScriptName`, `SetStartState`, `SetStates`, `AddState`, `SetCreatedAt`, `SetUpdatedAt`, and `Build() (Model, error)`.
  - `type Entity struct` → table `item_conversations`; `func Make(Entity) (Model, error)`, `func ToEntity(Model, uuid.UUID) (Entity, error)`, `func MigrateTable(*gorm.DB) error`.
  - `const Resource = "item-conversations"`; `type RestModel struct` with `Transform(Model) (RestModel, error)` and `Extract(RestModel) (Model, error)`.
  - `type Processor interface` with `Create(Model) (Model, error)`, `Update(uuid.UUID, Model) (Model, error)`, `Delete(uuid.UUID) error`, `ByIdProvider(uuid.UUID) model.Provider[Model]`, `ByItemIdProvider(uint32) model.Provider[Model]`, `AllProvider(model.Page) model.Provider[model.Paged[Model]]`, `DeleteAllForTenant() (int64, error)`, `Count() (int64, *time.Time, error)`; constructor `func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor`.
  - `func InitResource(jsonapi.ServerInformation) func(*gorm.DB) server.RouteInitializer` and `func InitSeedResource(jsonapi.ServerInformation) func(*gorm.DB) server.RouteInitializer`.
  - `type ItemConversationSubdomain struct{}` implementing `seeder.Subdomain[RestModel, Model]`.
  - `ByItemIdProvider` and `Model.FindState` are consumed by Task 8; `MigrateTable`, `InitResource`, `InitSeedResource` by `main.go`.

- [ ] **Step 1: Read all nine files you are mirroring**

Run:

```bash
cd services/atlas-npc-conversations/atlas.com/npc/conversation
cat npc/model.go            # the SHAPE to copy (single state machine, FindState)
cat quest/entity.go quest/provider.go quest/administrator.go quest/subdomain.go quest/groups.go
sed -n '1,80p' quest/rest.go
sed -n '1,60p' quest/resource.go
sed -n '1,80p' quest/processor.go
```

The item family is `quest`'s packaging with `npc`'s model shape. Note `quest/entity.go` stores the serialised `RestModel` in a single `data jsonb` column — do the same; do not invent a normalised state table.

- [ ] **Step 2: Write the failing model test**

Create `conversation/item/model_test.go`:

```go
package item

import (
	"testing"

	"atlas-npc-conversations/conversation"
)

func TestBuilderRequiresItemIdStartStateAndStates(t *testing.T) {
	state, err := conversation.NewStateModelBuilder().SetId("intro").Build()
	if err != nil {
		t.Fatalf("building fixture state: %v", err)
	}

	if _, err := NewBuilder().SetStartState("intro").AddState(state).Build(); err == nil {
		t.Error("expected error when itemId is unset")
	}
	if _, err := NewBuilder().SetItemId(2430008).AddState(state).Build(); err == nil {
		t.Error("expected error when startState is unset")
	}
	if _, err := NewBuilder().SetItemId(2430008).SetStartState("intro").Build(); err == nil {
		t.Error("expected error when states is empty")
	}

	m, err := NewBuilder().
		SetItemId(2430008).
		SetNpcId(2084002).
		SetScriptName("compassUse").
		SetStartState("intro").
		AddState(state).
		Build()
	if err != nil {
		t.Fatalf("valid build: %v", err)
	}
	if m.ItemId() != 2430008 || m.NpcId() != 2084002 || m.ScriptName() != "compassUse" {
		t.Errorf("round-trip: got item %d npc %d script %q", m.ItemId(), m.NpcId(), m.ScriptName())
	}
}

// FindState is the conversation.StateContainer contract — it is what lets the
// existing ProcessState/Continue/End machinery drive an item conversation with
// no changes.
func TestModelImplementsStateContainer(t *testing.T) {
	var _ conversation.StateContainer = Model{}

	state, err := conversation.NewStateModelBuilder().SetId("intro").Build()
	if err != nil {
		t.Fatalf("building fixture state: %v", err)
	}
	m, err := NewBuilder().SetItemId(2430013).SetStartState("intro").AddState(state).Build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if got, err := m.FindState("intro"); err != nil || got.Id() != "intro" {
		t.Errorf("FindState(intro): got %q err %v", got.Id(), err)
	}
	if _, err := m.FindState("nope"); err == nil {
		t.Error("FindState on an unknown state must error")
	}
}
```

Resolve `conversation.NewStateModelBuilder()`'s real name and required setters first:
`grep -n "func NewStateModelBuilder\|StateModelBuilder) Set\|func (b \*StateModelBuilder) Build" services/atlas-npc-conversations/atlas.com/npc/conversation/model.go | head -20`. Adjust the fixture construction to whatever that builder actually requires.

- [ ] **Step 3: Run it to verify it fails**

Run: `cd services/atlas-npc-conversations/atlas.com/npc && go test ./conversation/item/ -v`
Expected: FAIL — the package does not exist.

- [ ] **Step 4: Write `model.go`**

Create `conversation/item/model.go` following `conversation/npc/model.go` field-for-field, adding `itemId`, `npcId`, and `scriptName` in place of `npcId` alone:

```go
package item

import (
	"atlas-npc-conversations/conversation"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Model represents a conversation attached to a scripted item (the 243xxxx
// family). Resolution is by item id; npcId names only the avatar the dialogue
// renders with, and scriptName records the WZ spec/script value for authoring
// traceability — neither is a lookup key.
//
// The shape mirrors conversation/npc (a single state machine) rather than
// conversation/quest (a start/end pair): a scripted item has exactly one entry
// point, so a second machine would be permanently nil.
type Model struct {
	id         uuid.UUID
	itemId     uint32
	npcId      uint32
	scriptName string
	startState string
	states     []conversation.StateModel
	createdAt  time.Time
	updatedAt  time.Time
}

func (m Model) Id() uuid.UUID                        { return m.id }
func (m Model) ItemId() uint32                       { return m.itemId }
func (m Model) NpcId() uint32                        { return m.npcId }
func (m Model) ScriptName() string                   { return m.scriptName }
func (m Model) StartState() string                   { return m.startState }
func (m Model) States() []conversation.StateModel    { return m.states }
func (m Model) CreatedAt() time.Time                 { return m.createdAt }
func (m Model) UpdatedAt() time.Time                 { return m.updatedAt }

// FindState implements conversation.StateContainer.
func (m Model) FindState(stateId string) (conversation.StateModel, error) {
	for _, state := range m.states {
		if state.Id() == stateId {
			return state, nil
		}
	}
	return conversation.StateModel{}, errors.New("state not found")
}

type Builder struct {
	id         uuid.UUID
	itemId     uint32
	npcId      uint32
	scriptName string
	startState string
	states     []conversation.StateModel
	createdAt  time.Time
	updatedAt  time.Time
}

func NewBuilder() *Builder {
	return &Builder{
		id:        uuid.Nil,
		states:    make([]conversation.StateModel, 0),
		createdAt: time.Now(),
		updatedAt: time.Now(),
	}
}

func (b *Builder) SetId(id uuid.UUID) *Builder                       { b.id = id; return b }
func (b *Builder) SetItemId(itemId uint32) *Builder                  { b.itemId = itemId; return b }
func (b *Builder) SetNpcId(npcId uint32) *Builder                    { b.npcId = npcId; return b }
func (b *Builder) SetScriptName(name string) *Builder                { b.scriptName = name; return b }
func (b *Builder) SetStartState(startState string) *Builder          { b.startState = startState; return b }
func (b *Builder) SetStates(s []conversation.StateModel) *Builder    { b.states = s; return b }
func (b *Builder) AddState(s conversation.StateModel) *Builder       { b.states = append(b.states, s); return b }
func (b *Builder) SetCreatedAt(t time.Time) *Builder                 { b.createdAt = t; return b }
func (b *Builder) SetUpdatedAt(t time.Time) *Builder                 { b.updatedAt = t; return b }

func (b *Builder) Build() (Model, error) {
	if b.itemId == 0 {
		return Model{}, errors.New("itemId is required")
	}
	if b.startState == "" {
		return Model{}, errors.New("startState is required")
	}
	if len(b.states) == 0 {
		return Model{}, errors.New("at least one state is required")
	}
	return Model{
		id:         b.id,
		itemId:     b.itemId,
		npcId:      b.npcId,
		scriptName: b.scriptName,
		startState: b.startState,
		states:     b.states,
		createdAt:  b.createdAt,
		updatedAt:  b.updatedAt,
	}, nil
}
```

- [ ] **Step 5: Run the model test to verify it passes**

Run: `cd services/atlas-npc-conversations/atlas.com/npc && go test ./conversation/item/ -run 'TestBuilder|TestModelImplements' -v`
Expected: PASS. (`gofmt` will re-align the one-line method bodies; run `gofmt -w conversation/item/model.go`.)

- [ ] **Step 6: Write `rest.go`**

Mirror `quest/rest.go`, but with a single state machine inline rather than a nested `RestStateMachineModel` pair. Read `quest/rest.go` in full first — it contains the `RestStateModel` conversion helpers the item family reuses verbatim (states are the same vocabulary; no new `StateType` is introduced).

```go
package item

import (
	"atlas-npc-conversations/conversation"
	"fmt"

	"github.com/google/uuid"
	"github.com/jtumidanski/api2go/jsonapi"
)

const (
	Resource = "item-conversations"
)

// RestModel is the JSON:API representation of an item conversation. It is also
// the serialised form stored in the entity's `data` jsonb column, exactly as
// quest conversations are stored.
type RestModel struct {
	Id         uuid.UUID `json:"-"`
	ItemId     uint32    `json:"itemId"`
	NpcId      uint32    `json:"npcId,omitempty"`
	ScriptName string    `json:"scriptName,omitempty"`
	StartState string    `json:"startState"`
	States     []RestStateModel `json:"states"`
}

func (r RestModel) GetName() string { return Resource }
func (r RestModel) GetID() string   { return r.Id.String() }

func (r *RestModel) SetID(idStr string) error {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return fmt.Errorf("invalid conversation ID: %w", err)
	}
	r.Id = id
	return nil
}

func (r RestModel) GetReferences() []jsonapi.Reference               { return []jsonapi.Reference{} }
func (r RestModel) GetReferencedIDs() []jsonapi.ReferenceID          { return []jsonapi.ReferenceID{} }
func (r RestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier { return []jsonapi.MarshalIdentifier{} }

func Transform(m Model) (RestModel, error) { /* … */ }
func Extract(rm RestModel) (Model, error)  { /* … */ }
```

**Do not leave `/* … */` in the file.** `RestStateModel` and the two conversion bodies come from `quest/rest.go` — the state-level types there are shared and must be reused, not redefined. Read that file and decide, based on what it actually declares:

- If `quest/rest.go` declares `RestStateModel` (or equivalent) at package scope in `quest`, **move** the state-level rest types into the shared `conversation` package and have both `quest` and `item` reference them. Prefer a straightforward move over re-exporting aliases (project code-patterns rule).
- If the state-level rest types already live in `conversation`, just reference them.

Then write `Transform`/`Extract` as the direct field-by-field mapping between `Model` and `RestModel`, delegating state conversion to whichever helper `quest` uses.

- [ ] **Step 7: Write `entity.go`**

```go
package item

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Entity is one authored item conversation. Unique on (tenant_id, item_id):
// an item has at most one dialogue.
type Entity struct {
	ID         uuid.UUID      `gorm:"primaryKey;column:id;type:uuid"`
	TenantID   uuid.UUID      `gorm:"column:tenant_id;type:uuid;not null;uniqueIndex:idx_item_conversations_tenant_item,priority:1"`
	ItemID     uint32         `gorm:"column:item_id;not null;uniqueIndex:idx_item_conversations_tenant_item,priority:2"`
	NpcID      uint32         `gorm:"column:npc_id;index"`
	ScriptName string         `gorm:"column:script_name"`
	Data       string         `gorm:"column:data;type:jsonb;not null"`
	CreatedAt  time.Time      `gorm:"column:created_at;not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt  time.Time      `gorm:"column:updated_at;not null;default:CURRENT_TIMESTAMP"`
	DeletedAt  gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

func (Entity) TableName() string { return "item_conversations" }

func Make(e Entity) (Model, error) {
	var data RestModel
	if err := json.Unmarshal([]byte(e.Data), &data); err != nil {
		return Model{}, err
	}
	data.Id = e.ID
	return Extract(data)
}

func ToEntity(m Model, tenantId uuid.UUID) (Entity, error) {
	rm, err := Transform(m)
	if err != nil {
		return Entity{}, err
	}
	jsonData, err := json.Marshal(rm)
	if err != nil {
		return Entity{}, err
	}
	id := m.Id()
	if id == uuid.Nil {
		id = uuid.New()
	}
	return Entity{
		ID:         id,
		TenantID:   tenantId,
		ItemID:     m.ItemId(),
		NpcID:      m.NpcId(),
		ScriptName: m.ScriptName(),
		Data:       string(jsonData),
		CreatedAt:  m.CreatedAt(),
		UpdatedAt:  m.UpdatedAt(),
	}, nil
}

func MigrateTable(db *gorm.DB) error { return db.AutoMigrate(&Entity{}) }
```

Note this uses `uniqueIndex` where `quest/entity.go` uses a plain `index` — the design requires uniqueness on `(tenant_id, item_id)`.

- [ ] **Step 8: Write `provider.go` and `administrator.go`**

`provider.go` mirrors `quest/provider.go` exactly, with `getByItemIdProvider(itemId uint32)` in place of `getByQuestIdProvider`:

```go
func getByItemIdProvider(itemId uint32) func(db *gorm.DB) func() (Entity, error) {
	return func(db *gorm.DB) func() (Entity, error) {
		return func() (Entity, error) {
			var entity Entity
			result := db.Where("item_id = ?", itemId).First(&entity)
			return entity, result.Error
		}
	}
}
```

plus `getByIdProvider(id uuid.UUID)` and `getAllPagedProvider(page model.Page)` copied verbatim from `quest/provider.go` (only the entity type differs, and it is the package-local `Entity`).

`administrator.go` mirrors `quest/administrator.go`: `createItemConversation`, `updateItemConversation`, `deleteItemConversation`, `deleteAllItemConversations`. In `updateItemConversation`'s `Updates(map[string]interface{}{…})`, the columns are `"item_id"`, `"npc_id"`, `"script_name"`, `"data"`, `"updated_at"`.

- [ ] **Step 9: Write `processor.go`**

Mirror `quest/processor.go` minus `GetStateMachineForCharacter` (there is no per-character state for an item) and with `ByItemIdProvider` in place of `ByQuestIdProvider`. Keep `Count()` and its `parseDBTime` helper — the seeder's status endpoint needs them. Name the helper `parseDBTime` locally in this package; it is not exported from `quest`.

- [ ] **Step 10: Write `resource.go` and `groups.go`**

`resource.go` mirrors `quest/resource.go` with routes under `/items/conversations`:

```go
router.HandleFunc("/items/conversations", registerHandler("get_all_item_conversations", GetAllConversationsHandler)).Methods(http.MethodGet)
router.HandleFunc("/items/conversations/{conversationId}", registerHandler("get_item_conversation", GetConversationHandler)).Methods(http.MethodGet)
router.HandleFunc("/items/conversations", registerInputHandler("create_item_conversation", CreateConversationHandler)).Methods(http.MethodPost)
router.HandleFunc("/items/conversations/{conversationId}", registerInputHandler("update_item_conversation", UpdateConversationHandler)).Methods(http.MethodPatch)
router.HandleFunc("/items/conversations/{conversationId}", registerHandler("delete_item_conversation", DeleteConversationHandler)).Methods(http.MethodDelete)
```

In `CreateConversationHandler` and `UpdateConversationHandler`, reject an item id outside `2430000`–`2439999` with `400`, per PRD §5. Write that check explicitly:

```go
	if input.ItemId < 2430000 || input.ItemId > 2439999 {
		d.Logger().Errorf("Item conversation for item [%d] is outside the scripted-item range 2430000-2439999.", input.ItemId)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
```

Match the surrounding handler's error-writing idiom — read how `quest/resource.go` writes a 4xx before copying this in.

`groups.go` mirrors `quest/groups.go`:

```go
func InitSeedResource(_ jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			src := seeder.NewFilesystemCatalogSource("SEED_CATALOG_ROOT", "./deploy/seed")
			seeder.RegisterRoutes(router, db, l, src, seeder.Group{
				Name:      "npc-conversations:items",
				URLPrefix: "/items/conversations",
				Subdomains: []seeder.SubdomainAny{
					seeder.AdaptSubdomain[RestModel, Model](ItemConversationSubdomain{}),
				},
			})
		}
	}
}
```

- [ ] **Step 11: Write `subdomain.go`**

Mirror `quest/subdomain.go` exactly, changing four values and the tenant-id helper's name:

```go
var _ seeder.Subdomain[RestModel, Model] = ItemConversationSubdomain{}

type ItemConversationSubdomain struct{}

func (ItemConversationSubdomain) Name() string { return "item.conversation" }
func (ItemConversationSubdomain) Path() string { return "npc-conversations/items" }
func (ItemConversationSubdomain) Type() string { return "item-conversation" }
func (ItemConversationSubdomain) EntityIDPattern() *regexp.Regexp {
	return regexp.MustCompile(`^item-(\d+)\.json$`)
}
```

with `DeleteAllForTenant`, `Decode`, `Build`, `BulkCreate`, and `Count` copied from `quest/subdomain.go` (substituting `item-conversations:` in the error strings and `extractItemTenantId` for `extractQuestTenantId`).

- [ ] **Step 12: Register in `main.go`**

In `services/atlas-npc-conversations/atlas.com/npc/main.go`:

Import: `"atlas-npc-conversations/conversation/item"`.

Add to the `database.SetMigrations(...)` list (~`:60`), after `quest.MigrateTable`:

```go
		item.MigrateTable,
```

Add to the route initializers (~`:105`), after the two `quest.` lines:

```go
		AddRouteInitializer(item.InitResource(GetServer())(db)).
		AddRouteInitializer(item.InitSeedResource(GetServer())(db)).
```

- [ ] **Step 13: Register the catalog-lint rule**

In `tools/catalog-lint/subdomains.go`, in the `rules` slice, immediately after the `npc-conversations/quests` line:

```go
	{path: "npc-conversations/items", typ: "item-conversation", pattern: regexp.MustCompile(`^item-(\d+)\.json$`)},
```

- [ ] **Step 14: Write the entity and subdomain tests**

Create `conversation/item/entity_test.go`:

```go
package item

import (
	"testing"

	"atlas-npc-conversations/conversation"
	"github.com/google/uuid"
)

func TestEntityRoundTrip(t *testing.T) {
	state, err := conversation.NewStateModelBuilder().SetId("intro").Build()
	if err != nil {
		t.Fatalf("fixture state: %v", err)
	}
	in, err := NewBuilder().
		SetItemId(2430008).
		SetNpcId(2084002).
		SetScriptName("compassUse").
		SetStartState("intro").
		AddState(state).
		Build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	tenantId := uuid.New()
	e, err := ToEntity(in, tenantId)
	if err != nil {
		t.Fatalf("ToEntity: %v", err)
	}
	if e.TableName() != "item_conversations" {
		t.Errorf("table: got %q", e.TableName())
	}
	if e.ItemID != 2430008 || e.NpcID != 2084002 || e.ScriptName != "compassUse" {
		t.Errorf("columns: item %d npc %d script %q", e.ItemID, e.NpcID, e.ScriptName)
	}
	if e.TenantID != tenantId {
		t.Errorf("tenant: got %s want %s", e.TenantID, tenantId)
	}

	out, err := Make(e)
	if err != nil {
		t.Fatalf("Make: %v", err)
	}
	if out.ItemId() != in.ItemId() || out.NpcId() != in.NpcId() || out.ScriptName() != in.ScriptName() {
		t.Errorf("round-trip mismatch: %+v", out)
	}
	if out.StartState() != "intro" || len(out.States()) != 1 {
		t.Errorf("state machine lost in round-trip: start %q states %d", out.StartState(), len(out.States()))
	}
}
```

Create `conversation/item/subdomain_test.go`:

```go
package item

import "testing"

// The subdomain's four values are mirrored by hand in
// tools/catalog-lint/subdomains.go. If this test changes, that file must too.
func TestSubdomainIdentity(t *testing.T) {
	s := ItemConversationSubdomain{}
	if s.Name() != "item.conversation" {
		t.Errorf("Name: got %q", s.Name())
	}
	if s.Path() != "npc-conversations/items" {
		t.Errorf("Path: got %q", s.Path())
	}
	if s.Type() != "item-conversation" {
		t.Errorf("Type: got %q", s.Type())
	}
	for _, ok := range []string{"item-2430008.json", "item-2430013.json"} {
		if !s.EntityIDPattern().MatchString(ok) {
			t.Errorf("pattern should match %q", ok)
		}
	}
	for _, bad := range []string{"quest-1021.json", "item-abc.json", "2430008.json"} {
		if s.EntityIDPattern().MatchString(bad) {
			t.Errorf("pattern should not match %q", bad)
		}
	}
}
```

- [ ] **Step 15: Run the tests to verify they pass**

Run: `cd services/atlas-npc-conversations/atlas.com/npc && go build ./... && go test -race ./conversation/item/ -v`
Expected: all PASS.

- [ ] **Step 16: Run the module suite and catalog-lint**

Run:

```bash
cd services/atlas-npc-conversations/atlas.com/npc && go test -race ./... && go vet ./...
cd - && cd tools/catalog-lint && go build ./... && go test ./...
```

Expected: clean.

- [ ] **Step 17: Commit**

```bash
git add services/atlas-npc-conversations/atlas.com/npc/conversation/item/ \
        services/atlas-npc-conversations/atlas.com/npc/main.go \
        services/atlas-npc-conversations/atlas.com/npc/conversation/ \
        tools/catalog-lint/subdomains.go
git commit -m "feat(npc-conversations): add item-keyed conversation family

Packaged like conversation/quest (own item_conversations table, own
item-conversations REST resource, own seeder subdomain) but shaped like
conversation/npc (a single state machine). A scripted item has exactly one
entry point, so the quest family's dual start/end pair would leave
endStateMachine permanently nil.

Unique on (tenant_id, item_id). No new StateType is introduced — the existing
state vocabulary is reused unchanged, so ProcessState/Continue/End drive item
conversations with no modification."
```

---

### Task 8: `ItemConversationType`, `StartItem`, and redelivery idempotency

Kafka is at-least-once. `Processor.Start` returns `"another conversation exists"` when a context is already live (`processor.go:104-108`), so a redelivered start command would emit `START_ERROR` for a conversation the handler itself already opened — failing a saga that had already succeeded.

**Files:**
- Modify: `services/atlas-npc-conversations/atlas.com/npc/conversation/model.go:2213-2218` (add `ItemConversationType`), `:2223-2232` + `~:2287` (context field + getter), `:2300-2380` (builder)
- Modify: `.../conversation/model_json.go:495-542` (marshal/unmarshal the new field)
- Modify: `.../conversation/processor.go:22-34` (interface), add `StartItem` after `StartQuest` (~`:210`)
- Test: `.../conversation/processor_item_test.go`, `.../conversation/model_json_test.go`

**Interfaces:**
- Consumes: `item.Model` from Task 7 (satisfies `conversation.StateContainer`).
- Produces:
  - `conversation.ItemConversationType ConversationType = "item"`
  - `ConversationContext.OriginTransactionId() *uuid.UUID` and `ConversationContextBuilder.SetOriginTransactionId(uuid.UUID) *ConversationContextBuilder`
  - `Processor.StartItem(f field.Model, itemId uint32, npcId uint32, characterId uint32, accountId uint32, scriptName string, originTransactionId uuid.UUID, stateMachine StateContainer) error`
  - Sentinel errors `ErrConversationInProgress` and `ErrAlreadyStartedByThisTransaction`, consumed by Task 9's consumer to pick the status event.

- [ ] **Step 1: Write the failing JSON round-trip test**

`ConversationContext` is persisted in Redis via `atlas.TenantRegistry`, so a field that is not in `MarshalJSON`/`UnmarshalJSON` silently vanishes on the next read — and the idempotency check would then never fire.

Append to `conversation/model_json_test.go`:

```go
// originTransactionId must survive the Redis round-trip. The registry stores
// ConversationContext as JSON; a field missing from the marshal pair is
// silently dropped, and the redelivery guard in StartItem would never fire.
func TestConversationContextOriginTransactionIdSurvivesJSON(t *testing.T) {
	txn := uuid.New()
	in := NewConversationContextBuilder().
		SetCharacterId(42).
		SetNpcId(9010000).
		SetCurrentState("intro").
		SetConversationType(ItemConversationType).
		SetSourceId(2430013).
		SetOriginTransactionId(txn).
		Build()

	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out ConversationContext
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.OriginTransactionId() == nil {
		t.Fatal("originTransactionId lost in JSON round-trip")
	}
	if *out.OriginTransactionId() != txn {
		t.Errorf("originTransactionId: got %s want %s", *out.OriginTransactionId(), txn)
	}
	if out.ConversationType() != ItemConversationType {
		t.Errorf("conversationType: got %q want %q", out.ConversationType(), ItemConversationType)
	}
	if out.SourceId() != 2430013 {
		t.Errorf("sourceId: got %d want 2430013", out.SourceId())
	}
}
```

Add whatever imports the file needs (`encoding/json`, `testing`, `github.com/google/uuid`) — check the file's existing import block first.

- [ ] **Step 2: Run it to verify it fails**

Run: `cd services/atlas-npc-conversations/atlas.com/npc && go test ./conversation/ -run TestConversationContextOriginTransactionId -v`
Expected: FAIL — `undefined: ItemConversationType` and `SetOriginTransactionId`.

- [ ] **Step 3: Add the conversation type and the context field**

In `conversation/model.go`, extend the `ConversationType` block (`:2216-2219`):

```go
const (
	NpcConversationType   ConversationType = "npc"
	QuestConversationType ConversationType = "quest"
	// ItemConversationType is a scripted item's own dialogue (the 243xxxx
	// family). SourceId carries the item id; NpcId carries only the avatar the
	// dialogue renders with.
	ItemConversationType ConversationType = "item"
)
```

Add the field to `ConversationContext` (after `sourceId`):

```go
	// originTransactionId is the saga transaction that started this
	// conversation, when one did. It exists so a redelivered start command can
	// be recognised as its own: Kafka is at-least-once, and re-emitting
	// START_ERROR for a conversation this very transaction already opened would
	// fail a saga that had already succeeded.
	//
	// Deliberately NOT folded into the registry's sagaIndex, which is keyed by
	// sagas a conversation INITIATED — the opposite direction.
	originTransactionId *uuid.UUID
```

Add the getter beside `SourceId()`:

```go
// OriginTransactionId returns the saga transaction that started this
// conversation, or nil for a conversation not started by a saga.
func (c ConversationContext) OriginTransactionId() *uuid.UUID {
	return c.originTransactionId
}
```

Add the same field to `ConversationContextBuilder`, the setter:

```go
// SetOriginTransactionId records the saga transaction that started this
// conversation.
func (b *ConversationContextBuilder) SetOriginTransactionId(id uuid.UUID) *ConversationContextBuilder {
	b.originTransactionId = &id
	return b
}
```

and carry it through `Build()`:

```go
		originTransactionId: b.originTransactionId,
```

- [ ] **Step 4: Add the field to the JSON pair**

In `conversation/model_json.go`, add to **both** the marshal anonymous struct (`:508-517`) and the unmarshal one (`:522-530`):

```go
		OriginTransactionId *uuid.UUID `json:"originTransactionId,omitempty"`
```

In `MarshalJSON`, append `c.originTransactionId` to the positional composite literal at `:517` — it is a positional literal, so the field must go in the same position in both the struct definition and the value list. In `UnmarshalJSON`, add:

```go
	c.originTransactionId = aux.OriginTransactionId
```

- [ ] **Step 5: Run the JSON test to verify it passes**

Run: `cd services/atlas-npc-conversations/atlas.com/npc && go test ./conversation/ -run TestConversationContextOriginTransactionId -v`
Expected: PASS.

- [ ] **Step 6: Write the failing `StartItem` idempotency test**

Create `conversation/processor_item_test.go`. Read `conversation/processor_test.go` and `processor_state_transition_test.go` first for how the registry is stood up in tests (there is a Redis-backed `InitRegistry`; the existing tests show the seam — use exactly what they use, do not invent a fake).

```go
// A redelivered start command carrying the SAME transaction id as the live
// context must be treated as already-succeeded, not as a conflict. Kafka is
// at-least-once, and emitting START_ERROR here would fail a saga that had
// already opened its dialogue.
func TestStartItemRedeliveryOfSameTransactionIsNotAConflict(t *testing.T) {
	// … stand up registry + processor per the sibling tests …
	txn := uuid.New()

	if err := p.StartItem(f, 2430013, 9010000, characterId, accountId, "item_2430013", txn, sm); err != nil {
		t.Fatalf("first start: %v", err)
	}
	err := p.StartItem(f, 2430013, 9010000, characterId, accountId, "item_2430013", txn, sm)
	if !errors.Is(err, ErrAlreadyStartedByThisTransaction) {
		t.Fatalf("redelivery: got %v, want ErrAlreadyStartedByThisTransaction", err)
	}
}

// A DIFFERENT transaction against a live context is a genuine conflict.
func TestStartItemDifferentTransactionIsAConflict(t *testing.T) {
	// … same setup …
	if err := p.StartItem(f, 2430013, 9010000, characterId, accountId, "item_2430013", uuid.New(), sm); err != nil {
		t.Fatalf("first start: %v", err)
	}
	err := p.StartItem(f, 2430008, 2084002, characterId, accountId, "compassUse", uuid.New(), sm2)
	if !errors.Is(err, ErrConversationInProgress) {
		t.Fatalf("conflict: got %v, want ErrConversationInProgress", err)
	}
}
```

Fill in the setup from the sibling tests — do not leave the `…` comments in the committed file.

- [ ] **Step 7: Run it to verify it fails**

Run: `cd services/atlas-npc-conversations/atlas.com/npc && go test ./conversation/ -run TestStartItem -v`
Expected: FAIL — `p.StartItem undefined`.

- [ ] **Step 8: Add the sentinel errors and `StartItem`**

In `conversation/processor.go`, near the top of the file:

```go
// ErrConversationInProgress means a DIFFERENT conversation is already live for
// this character. The caller maps it to START_ERROR reason
// "conversation_in_progress".
var ErrConversationInProgress = errors.New("another conversation exists")

// ErrAlreadyStartedByThisTransaction means the live conversation was started by
// this very saga transaction — a Kafka redelivery, not a conflict. The caller
// re-emits STARTED rather than START_ERROR.
var ErrAlreadyStartedByThisTransaction = errors.New("conversation already started by this transaction")
```

Add to the `Processor` interface, after `StartQuest`:

```go
	// StartItem starts a scripted item's own conversation (the 243xxxx family).
	// Resolution is by item id; npcId selects only the avatar the dialogue
	// renders with. originTransactionId is the saga transaction awaiting the
	// STARTED/START_ERROR status event; it is stamped into the context so a
	// redelivered command is recognised as its own.
	StartItem(f field.Model, itemId uint32, npcId uint32, characterId uint32, accountId uint32, scriptName string, originTransactionId uuid.UUID, stateMachine StateContainer) error
```

Implement it after `StartQuest` (`~:210`), following that method's shape exactly and differing only in the guard and the context values:

```go
func (p *ProcessorImpl) StartItem(f field.Model, itemId uint32, npcId uint32, characterId uint32, accountId uint32, scriptName string, originTransactionId uuid.UUID, stateMachine StateContainer) error {
	p.l.Debugf("Starting item [%d] conversation with NPC [%d] for character [%d] in map [%d].", itemId, npcId, characterId, f.MapId())

	// Redelivery guard. A live context stamped with this same transaction id is
	// this command's own earlier delivery, not a conflict — Kafka is
	// at-least-once and START_ERROR here would fail a saga that succeeded.
	if prev, err := GetRegistry().GetPreviousContext(p.ctx, characterId); err == nil {
		if prev.OriginTransactionId() != nil && *prev.OriginTransactionId() == originTransactionId {
			p.l.Debugf("Item [%d] conversation for character [%d] was already started by transaction [%s]; treating redelivery as success.", itemId, characterId, originTransactionId.String())
			return ErrAlreadyStartedByThisTransaction
		}
		p.l.Debugf("Previous conversation for character [%d] exists, avoiding starting item [%d] conversation.", characterId, itemId)
		return ErrConversationInProgress
	}

	startStateId := stateMachine.StartState()

	builder := NewConversationContextBuilder().
		SetField(f).
		SetCharacterId(characterId).
		SetNpcId(npcId).
		SetCurrentState(startStateId).
		SetConversation(stateMachine).
		SetConversationType(ItemConversationType).
		SetSourceId(itemId).
		SetOriginTransactionId(originTransactionId)

	builder.AddContextValue("itemId", strconv.FormatUint(uint64(itemId), 10))
	if scriptName != "" {
		builder.AddContextValue("scriptName", scriptName)
	}
	if f.WorldId() > 0 {
		builder.AddContextValue("worldId", strconv.Itoa(int(f.WorldId())))
	}
	if f.ChannelId() > 0 {
		builder.AddContextValue("channelId", strconv.Itoa(int(f.ChannelId())))
	}
	if accountId > 0 {
		builder.AddContextValue("accountId", strconv.Itoa(int(accountId)))
	}

	ctx := builder.Build()
	GetRegistry().SetContext(p.ctx, ctx.CharacterId(), ctx)

	cont := true
	var err error
	for cont {
		ctx, err = GetRegistry().GetPreviousContext(p.ctx, characterId)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to retrieve conversation context for [%d].", characterId)
			return errors.New("conversation context not found")
		}
		cont, err = p.ProcessState(ctx)
		if err != nil {
			p.l.WithError(err).Errorf("Failed to process state [%s] for character [%d] and item [%d]", startStateId, characterId, itemId)
			return err
		}
	}
	return nil
}
```

Add `"github.com/google/uuid"` to the file's imports if it is not already there.

- [ ] **Step 9: Update the processor mock**

Run: `ls services/atlas-npc-conversations/atlas.com/npc/conversation/mock/`

Add `StartItem` to the mock with the same signature and a `StartItemFunc` field, following exactly how `StartQuest` is mocked there. Without this the package will not compile.

- [ ] **Step 10: Run the tests to verify they pass**

Run: `cd services/atlas-npc-conversations/atlas.com/npc && go test -race ./conversation/... -v 2>&1 | tail -40`
Expected: all PASS, including the pre-existing suite.

- [ ] **Step 11: Run the module suite**

Run: `cd services/atlas-npc-conversations/atlas.com/npc && go build ./... && go test -race ./... && go vet ./...`
Expected: clean.

- [ ] **Step 12: Commit**

```bash
git add services/atlas-npc-conversations/atlas.com/npc/conversation/
git commit -m "feat(npc-conversations): add StartItem with redelivery-safe transaction stamping

Adds ItemConversationType and Processor.StartItem, structurally identical to
StartQuest. The conversation context now carries originTransactionId, persisted
through the Redis registry's JSON pair, so a redelivered start command carrying
the same transaction id returns ErrAlreadyStartedByThisTransaction rather than
the conflict error — re-emitting START_ERROR there would fail a saga that had
already opened its dialogue.

Kept separate from the registry's sagaIndex, which is keyed by sagas a
conversation initiated — the opposite direction."
```

---

### Task 9: The npc-conversation Kafka contract — start command + new status topic

`atlas-npc-conversations` today produces **no** status topic; it only consumes `EVENT_TOPIC_SAGA_STATUS` for sagas a conversation initiates. The awaited-step saga needs the opposite direction.

**Files:**
- Modify: `services/atlas-npc-conversations/atlas.com/npc/kafka/message/npc/kafka.go` (owner: `TransactionId` on `Command`, new command type + body, new status topic)
- Create: `.../npc/kafka/producer/conversation_status.go` (or extend an existing producer file — check `ls services/atlas-npc-conversations/atlas.com/npc/kafka/` first)
- Modify: `.../npc/kafka/consumer/npc/consumer.go` (new `START_ITEM_CONVERSATION` handler; status emission on the existing `START_CONVERSATION` handler)
- Test: `.../npc/kafka/consumer/npc/consumer_test.go`

**Interfaces:**
- Consumes: `conversation.Processor.StartItem`, `ErrConversationInProgress`, `ErrAlreadyStartedByThisTransaction` (Task 8); `item.NewProcessor(...).ByItemIdProvider` (Task 7).
- Produces, in package `npc` (`atlas-npc-conversations/kafka/message/npc`):
  - `Command[E].TransactionId uuid.UUID \`json:"transactionId,omitempty"\``
  - `CommandTypeStartItemConversation = "START_ITEM_CONVERSATION"`
  - `type CommandItemConversationStartBody struct { WorldId; ChannelId; MapId; Instance; AccountId; ItemId; Slot }`
  - `EnvStatusEventTopic = "EVENT_TOPIC_NPC_CONVERSATION_STATUS"`, `StatusEventTypeStarted = "STARTED"`, `StatusEventTypeStartError = "START_ERROR"`
  - reason constants `StartErrorNoConversationAuthored = "NO_CONVERSATION_AUTHORED"`, `StartErrorConversationInProgress = "CONVERSATION_IN_PROGRESS"`, `StartErrorInternal = "INTERNAL_ERROR"`
  - `type StatusEvent[E any] struct { TransactionId uuid.UUID; CharacterId uint32; Type string; Body E }`
  - `type StatusEventStartedBody struct { NpcTemplateId uint32; SourceId uint32 }`
  - `type StatusEventStartErrorBody struct { NpcTemplateId uint32; SourceId uint32; Reason string }`
- This file is mirrored into `atlas-saga-orchestrator` by Task 12. Any change here must be replayed there.

- [ ] **Step 1: Read the contract you are modelling on**

Run:

```bash
sed -n '15,30p;50,115p' services/atlas-npc-shops/atlas.com/npc/kafka/message/shops/kafka.go
sed -n '1,50p' services/atlas-npc-conversations/atlas.com/npc/kafka/message/npc/kafka.go
ls services/atlas-npc-conversations/atlas.com/npc/kafka/
```

The npc-shop contract is the shape: `TransactionId uuid.UUID` on both `Command` and `StatusEvent`, a split `ENTER_ERROR`/`ERROR` pair, and reason constants grouped under a comment naming which body carries them.

- [ ] **Step 2: Extend the contract file**

In `services/atlas-npc-conversations/atlas.com/npc/kafka/message/npc/kafka.go`:

Add to the first `const` block:

```go
	// CommandTypeStartItemConversation opens a scripted item's own dialogue
	// (the 243xxxx family). Unlike START_CONVERSATION the conversation is keyed
	// by item id, not by NPC — NpcId carries only the avatar it renders with.
	CommandTypeStartItemConversation = "START_ITEM_CONVERSATION"
```

Add `TransactionId` to `Command[E]`:

```go
type Command[E any] struct {
	// TransactionId correlates this command with the saga step that issued it.
	// uuid.Nil means "not saga-driven" — the ordinary NPC-talk path — and the
	// handler emits no status event for it. Non-nil means a saga step is
	// awaiting STARTED or START_ERROR on EnvStatusEventTopic.
	TransactionId uuid.UUID `json:"transactionId,omitempty"`
	NpcId         uint32    `json:"npcId"`
	CharacterId   uint32    `json:"characterId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}
```

Add the new body type after `CommandConversationStartBody`:

```go
// CommandItemConversationStartBody starts a scripted item's dialogue. Slot is
// carried so the destroy step's payload and this command describe the same
// asset; the conversation itself does not consume.
type CommandItemConversationStartBody struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
	AccountId uint32     `json:"accountId"`
	ItemId    uint32     `json:"itemId"`
	Slot      int16      `json:"slot"`
}
```

Add the status topic block at the end of the file:

```go
const (
	// EnvStatusEventTopic reports the outcome of a saga-driven conversation
	// start. atlas-npc-conversations produced no status topic before task-230;
	// it only consumed EVENT_TOPIC_SAGA_STATUS for sagas a conversation
	// initiates. The awaited-step saga needs the opposite direction.
	EnvStatusEventTopic = "EVENT_TOPIC_NPC_CONVERSATION_STATUS"

	StatusEventTypeStarted = "STARTED"

	// StatusEventTypeStartError is deliberately a distinct type rather than a
	// generic ERROR, for the same reason npc-shops splits ENTER_ERROR from
	// ERROR: a generic error type is rendered differently by the channel and
	// would be ambiguous here.
	StatusEventTypeStartError = "START_ERROR"

	// Reasons carried by StatusEventStartErrorBody. The reason is what makes a
	// Loki trace of a content gap distinguishable from a real fault without
	// reading code.
	StartErrorNoConversationAuthored = "NO_CONVERSATION_AUTHORED"
	StartErrorConversationInProgress = "CONVERSATION_IN_PROGRESS"
	StartErrorInternal               = "INTERNAL_ERROR"
)

// StatusEvent reports a conversation start outcome back to the saga that asked
// for it.
type StatusEvent[E any] struct {
	// TransactionId echoes the originating command's id so a saga can accept
	// only its own event.
	TransactionId uuid.UUID `json:"transactionId"`
	CharacterId   uint32    `json:"characterId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type StatusEventStartedBody struct {
	NpcTemplateId uint32 `json:"npcTemplateId"`
	// SourceId is the item id for an item conversation, the NPC template id for
	// an NPC conversation — mirroring ConversationContext.SourceId.
	SourceId uint32 `json:"sourceId"`
}

type StatusEventStartErrorBody struct {
	NpcTemplateId uint32 `json:"npcTemplateId"`
	SourceId      uint32 `json:"sourceId"`
	Reason        string `json:"reason"`
}
```

- [ ] **Step 3: Write the status producer**

Check for an existing producer package: `ls services/atlas-npc-conversations/atlas.com/npc/kafka/`. If there is a `producer/` directory, add the file there; otherwise create `kafka/producer/conversation_status.go`. Model it on `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/producer.go:371-383` (key by character id, `producer.SingleMessageProvider`):

```go
package producer

import (
	npc2 "atlas-npc-conversations/kafka/message/npc"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// StartedStatusProvider reports that a saga-driven conversation opened. Keyed by
// character id, matching every other producer on the conversation topics.
func StartedStatusProvider(transactionId uuid.UUID, characterId uint32, npcTemplateId uint32, sourceId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &npc2.StatusEvent[npc2.StatusEventStartedBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		Type:          npc2.StatusEventTypeStarted,
		Body: npc2.StatusEventStartedBody{
			NpcTemplateId: npcTemplateId,
			SourceId:      sourceId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// StartErrorStatusProvider reports that a saga-driven conversation did not open.
// The awaiting step fails, which fails the saga, which means the following
// destroy step never runs — the player keeps the item.
func StartErrorStatusProvider(transactionId uuid.UUID, characterId uint32, npcTemplateId uint32, sourceId uint32, reason string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &npc2.StatusEvent[npc2.StatusEventStartErrorBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		Type:          npc2.StatusEventTypeStartError,
		Body: npc2.StatusEventStartErrorBody{
			NpcTemplateId: npcTemplateId,
			SourceId:      sourceId,
			Reason:        reason,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
```

Match the package name and import alias style of whatever producer files already exist in the service.

- [ ] **Step 4: Write the failing consumer test**

Read `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/consumer/npcshop/consumer_test.go` for the test idiom (how the emit seam is observed without a broker), then create/extend `services/atlas-npc-conversations/atlas.com/npc/kafka/consumer/npc/consumer_test.go`:

```go
// captured is what the emit seam recorded, so the tests can assert on emissions
// without a broker.
type captured struct {
	count  int
	evType string
	reason string
}

// install replaces the emit seam for one test and restores it after. It decodes
// whichever StatusEvent body arrived so a single helper serves both types.
func install(t *testing.T, c *captured) {
	t.Helper()
	orig := emitConversationStatus
	emitConversationStatus = func(_ logrus.FieldLogger, _ context.Context, p model.Provider[[]kafka.Message]) error {
		msgs, err := p()
		if err != nil {
			return err
		}
		for _, m := range msgs {
			var probe struct {
				Type string `json:"type"`
				Body struct {
					Reason string `json:"reason"`
				} `json:"body"`
			}
			if err := json.Unmarshal(m.Value, &probe); err != nil {
				return err
			}
			c.count++
			c.evType = probe.Type
			c.reason = probe.Body.Reason
		}
		return nil
	}
	t.Cleanup(func() { emitConversationStatus = orig })
}

// A saga-driven start (non-nil transactionId) emits exactly one STARTED.
func TestStartItemConversation_EmitsStartedOnSuccess(t *testing.T) {
	var c captured
	install(t, &c)
	db := seedItemConversation(t, 2430013, 9010000, "item_2430013")

	handleStartItemConversationCommand(db)(testLogger(t), testCtx(t), npc2.Command[npc2.CommandItemConversationStartBody]{
		TransactionId: uuid.New(),
		NpcId:         9010000,
		CharacterId:   1234,
		Type:          npc2.CommandTypeStartItemConversation,
		Body:          npc2.CommandItemConversationStartBody{ItemId: 2430013, Slot: 5, AccountId: 77},
	})

	if c.count != 1 || c.evType != npc2.StatusEventTypeStarted {
		t.Fatalf("emitted %d event(s) of type %q, want 1 STARTED", c.count, c.evType)
	}
}

// A content gap is not a fault. START_ERROR fails the awaiting step, the
// following destroy never runs, and the player keeps the item.
func TestStartItemConversation_EmitsNoConversationAuthoredWhenUnauthored(t *testing.T) {
	var c captured
	install(t, &c)
	db := emptyDB(t) // no conversation authored for 2430013

	handleStartItemConversationCommand(db)(testLogger(t), testCtx(t), npc2.Command[npc2.CommandItemConversationStartBody]{
		TransactionId: uuid.New(),
		NpcId:         9010000,
		CharacterId:   1234,
		Type:          npc2.CommandTypeStartItemConversation,
		Body:          npc2.CommandItemConversationStartBody{ItemId: 2430013, Slot: 5},
	})

	if c.count != 1 || c.evType != npc2.StatusEventTypeStartError {
		t.Fatalf("emitted %d event(s) of type %q, want 1 START_ERROR", c.count, c.evType)
	}
	if c.reason != npc2.StartErrorNoConversationAuthored {
		t.Errorf("reason: got %q, want %q", c.reason, npc2.StartErrorNoConversationAuthored)
	}
}

// A DIFFERENT transaction against a live context is a genuine conflict.
func TestStartItemConversation_EmitsConversationInProgressOnConflict(t *testing.T) {
	var c captured
	db := seedItemConversation(t, 2430013, 9010000, "item_2430013")
	cmd := npc2.Command[npc2.CommandItemConversationStartBody]{
		TransactionId: uuid.New(),
		NpcId:         9010000,
		CharacterId:   1234,
		Type:          npc2.CommandTypeStartItemConversation,
		Body:          npc2.CommandItemConversationStartBody{ItemId: 2430013, Slot: 5},
	}
	handleStartItemConversationCommand(db)(testLogger(t), testCtx(t), cmd) // occupies the character

	install(t, &c) // only observe the SECOND command
	second := cmd
	second.TransactionId = uuid.New()
	handleStartItemConversationCommand(db)(testLogger(t), testCtx(t), second)

	if c.count != 1 || c.evType != npc2.StatusEventTypeStartError {
		t.Fatalf("emitted %d event(s) of type %q, want 1 START_ERROR", c.count, c.evType)
	}
	if c.reason != npc2.StartErrorConversationInProgress {
		t.Errorf("reason: got %q, want %q", c.reason, npc2.StartErrorConversationInProgress)
	}
}

// Redelivery: the SAME transaction id against its own live context re-emits
// STARTED, never START_ERROR. Emitting an error here would fail a saga that had
// already succeeded — Kafka is at-least-once, so this is the realistic case.
func TestStartItemConversation_RedeliveryReemitsStarted(t *testing.T) {
	var c captured
	db := seedItemConversation(t, 2430013, 9010000, "item_2430013")
	cmd := npc2.Command[npc2.CommandItemConversationStartBody]{
		TransactionId: uuid.New(),
		NpcId:         9010000,
		CharacterId:   1234,
		Type:          npc2.CommandTypeStartItemConversation,
		Body:          npc2.CommandItemConversationStartBody{ItemId: 2430013, Slot: 5},
	}
	handleStartItemConversationCommand(db)(testLogger(t), testCtx(t), cmd)

	install(t, &c) // only observe the redelivery
	handleStartItemConversationCommand(db)(testLogger(t), testCtx(t), cmd)

	if c.count != 1 || c.evType != npc2.StatusEventTypeStarted {
		t.Fatalf("redelivery emitted %d event(s) of type %q, want 1 STARTED", c.count, c.evType)
	}
}

// The ordinary NPC-talk path is unchanged: no transaction, no status event.
func TestStartConversation_NilTransactionEmitsNoStatus(t *testing.T) {
	var c captured
	install(t, &c)
	db := emptyDB(t)

	handleStartConversationCommand(db)(testLogger(t), testCtx(t), npc2.Command[npc2.CommandConversationStartBody]{
		TransactionId: uuid.Nil,
		NpcId:         1002000,
		CharacterId:   1234,
		Type:          npc2.CommandTypeStartConversation,
		Body:          npc2.CommandConversationStartBody{},
	})

	if c.count != 0 {
		t.Fatalf("non-saga start emitted %d event(s), want 0", c.count)
	}
}
```

`testLogger`, `testCtx` (a tenant-bearing context), `emptyDB`, and `seedItemConversation(t, itemId, npcId, scriptName) *gorm.DB` must follow whatever the service's existing tests already use to stand up a tenant context, a GORM handle, and the Redis-backed conversation registry — read `services/atlas-npc-conversations/atlas.com/npc/conversation/processor_test.go` and any `testmain_test.go` in the service and reuse those helpers verbatim rather than inventing new ones. Do **not** create a `*_testhelpers.go` file; the project bans them.

Introduce the package-level emit seam these tests replace, following how `EmitNpcShopExit` is indirected in `saga/producer.go`:

```go
// emitConversationStatus is indirected through a package var so consumer tests
// can observe emissions without a broker (same seam shape as EmitNpcShopExit).
var emitConversationStatus = func(l logrus.FieldLogger, ctx context.Context, p model.Provider[[]kafka.Message]) error {
	return producer.ProviderImpl(l)(ctx)(npc2.EnvStatusEventTopic)(p)
}
```

- [ ] **Step 5: Run the tests to verify they fail**

Run: `cd services/atlas-npc-conversations/atlas.com/npc && go test ./kafka/consumer/npc/ -v`
Expected: FAIL — the handler does not exist.

- [ ] **Step 6: Add the `START_ITEM_CONVERSATION` handler**

In `kafka/consumer/npc/consumer.go`, register a fourth handler in `InitHandlers`:

```go
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStartItemConversationCommand(db)))); err != nil {
			return err
		}
```

and implement it:

```go
func handleStartItemConversationCommand(db *gorm.DB) message.Handler[npc2.Command[npc2.CommandItemConversationStartBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c npc2.Command[npc2.CommandItemConversationStartBody]) {
		if c.Type != npc2.CommandTypeStartItemConversation {
			return
		}

		fields := logrus.Fields{
			"transaction_id":  c.TransactionId.String(),
			"character_id":    c.CharacterId,
			"item_id":         c.Body.ItemId,
			"npc_template_id": c.NpcId,
		}

		// The item's own dialogue, keyed by item id. A missing conversation is a
		// content gap, not a fault: START_ERROR fails the awaiting step, the
		// following destroy never runs, and the player keeps the item.
		m, err := item.NewProcessor(l, ctx, db).ByItemIdProvider(c.Body.ItemId)()
		if err != nil {
			l.WithError(err).WithFields(fields).Warn("No conversation authored for scripted item; not consuming.")
			emitStartError(l, ctx, c, npc2.StartErrorNoConversationAuthored)
			return
		}

		f := field.NewBuilder(c.Body.WorldId, c.Body.ChannelId, c.Body.MapId).SetInstance(c.Body.Instance).Build()

		err = conversation.NewProcessor(l, ctx, db).StartItem(f, c.Body.ItemId, c.NpcId, c.CharacterId, c.Body.AccountId, m.ScriptName(), c.TransactionId, m)
		switch {
		case err == nil:
			l.WithFields(fields).Info("Scripted item conversation started.")
			emitStarted(l, ctx, c)
		case errors.Is(err, conversation.ErrAlreadyStartedByThisTransaction):
			// Kafka redelivery of this very command. The dialogue is already
			// open; re-emit success so the awaiting step completes.
			l.WithFields(fields).Debug("Redelivered start command; re-emitting STARTED.")
			emitStarted(l, ctx, c)
		case errors.Is(err, conversation.ErrConversationInProgress):
			l.WithFields(fields).Warn("Character is already in a conversation; not consuming.")
			emitStartError(l, ctx, c, npc2.StartErrorConversationInProgress)
		default:
			l.WithError(err).WithFields(fields).Error("Unable to start scripted item conversation.")
			emitStartError(l, ctx, c, npc2.StartErrorInternal)
		}
	}
}

// emitStarted / emitStartError are no-ops for a non-saga command (uuid.Nil),
// which is how the ordinary NPC-talk path stays unchanged.
func emitStarted[E any](l logrus.FieldLogger, ctx context.Context, c npc2.Command[E]) {
	if c.TransactionId == uuid.Nil {
		return
	}
	_ = emitConversationStatus(l, ctx, producer2.StartedStatusProvider(c.TransactionId, c.CharacterId, c.NpcId, sourceIdOf(c)))
}
```

`sourceIdOf` cannot be generic over an arbitrary body — write **two** concrete emit pairs instead, one for `CommandItemConversationStartBody` (source = `c.Body.ItemId`) and one for `CommandConversationStartBody` (source = `c.NpcId`). Duplicated four-line functions are correct here; a generic accessor over unrelated body types is not.

Verify `field.NewBuilder(...).SetInstance(...)` is the real API before using it: `grep -n "func NewBuilder\|func (b \*Builder) SetInstance" libs/atlas-constants/field/*.go`.

- [ ] **Step 7: Emit status from the existing `START_CONVERSATION` handler**

`handleStartConversationCommand` currently discards its error (`_ = ...Start(...)`). The `239` saga path (Task 16) reuses this command type with a non-nil transaction id, so it must now report:

```go
func handleStartConversationCommand(db *gorm.DB) message.Handler[npc2.Command[npc2.CommandConversationStartBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c npc2.Command[npc2.CommandConversationStartBody]) {
		if c.Type != npc2.CommandTypeStartConversation {
			return
		}
		f := field.NewBuilder(c.Body.WorldId, c.Body.ChannelId, c.Body.MapId).SetInstance(c.Body.Instance).Build()
		err := conversation.NewProcessor(l, ctx, db).Start(f, c.NpcId, c.CharacterId, c.Body.AccountId)

		// uuid.Nil = the ordinary NPC-talk path, which has no saga awaiting it
		// and must stay exactly as it was. Only a saga-driven start reports.
		if c.TransactionId == uuid.Nil {
			return
		}
		if err != nil {
			l.WithError(err).WithFields(logrus.Fields{
				"transaction_id":  c.TransactionId.String(),
				"character_id":    c.CharacterId,
				"npc_template_id": c.NpcId,
			}).Warn("Unable to start saga-driven NPC conversation.")
			emitNpcStartError(l, ctx, c, npc2.StartErrorInternal)
			return
		}
		emitNpcStarted(l, ctx, c)
	}
}
```

Note this changes the existing handler's field-model construction to carry `Instance`; confirm `CommandConversationStartBody` already has `Instance` (it does — `kafka.go`) and that the previous code's omission was not deliberate by checking `git log -1 -p --  services/atlas-npc-conversations/atlas.com/npc/kafka/consumer/npc/consumer.go | head -60`. If dropping the instance was deliberate, leave that line as it was and only add the status emission.

- [ ] **Step 8: Run the tests to verify they pass**

Run: `cd services/atlas-npc-conversations/atlas.com/npc && go test -race ./kafka/... -v 2>&1 | tail -30`
Expected: all PASS.

- [ ] **Step 9: Run the module suite**

Run: `cd services/atlas-npc-conversations/atlas.com/npc && go build ./... && go test -race ./... && go vet ./...`
Expected: clean.

- [ ] **Step 10: Commit**

```bash
git add services/atlas-npc-conversations/atlas.com/npc/kafka/
git commit -m "feat(npc-conversations): add START_ITEM_CONVERSATION and a conversation status topic

Adds TransactionId to the npc Command envelope and a new
EVENT_TOPIC_NPC_CONVERSATION_STATUS carrying STARTED / START_ERROR, so a saga
step can await the outcome of a conversation start instead of assuming it.
START_ERROR carries a reason (NO_CONVERSATION_AUTHORED /
CONVERSATION_IN_PROGRESS / INTERNAL_ERROR) so a Loki trace distinguishes a
content gap from a real fault without reading code.

uuid.Nil means not-saga-driven and emits nothing, leaving the ordinary NPC-talk
path unchanged — the same discipline npc-shops uses for its ENTER command."
```

---

### Task 10: Register the new Kafka topic in k8s

An unsuffixed topic name silently falls back and cross-talks between environments.

**Files:**
- Modify: `deploy/k8s/base/env-configmap.yaml` (beside `EVENT_TOPIC_NPC_SHOP_STATUS`, `:147`)
- Modify: `deploy/k8s/overlays/pr/kustomization.yaml` (`:308` region)
- Modify: `deploy/k8s/overlays/main/kustomization.yaml` (`:184` region)

**Interfaces:**
- Consumes: `EnvStatusEventTopic = "EVENT_TOPIC_NPC_CONVERSATION_STATUS"` (Task 9).
- Produces: the env var resolves in every environment, for both the producer (`atlas-npc-conversations`) and the consumer (`atlas-saga-orchestrator`, Task 13).

- [ ] **Step 1: Read all three existing registrations for the sibling topic**

Run:

```bash
grep -n "EVENT_TOPIC_NPC_SHOP_STATUS" deploy/k8s/base/env-configmap.yaml \
  deploy/k8s/overlays/pr/kustomization.yaml deploy/k8s/overlays/main/kustomization.yaml
```

Expected:

```
deploy/k8s/base/env-configmap.yaml:147:  EVENT_TOPIC_NPC_SHOP_STATUS: "EVENT_TOPIC_NPC_SHOP_STATUS"
deploy/k8s/overlays/pr/kustomization.yaml:308:      - EVENT_TOPIC_NPC_SHOP_STATUS=EVENT_TOPIC_NPC_SHOP_STATUS-PLACEHOLDER_ATLAS_ENV
deploy/k8s/overlays/main/kustomization.yaml:184:      - EVENT_TOPIC_NPC_SHOP_STATUS=EVENT_TOPIC_NPC_SHOP_STATUS-main
```

- [ ] **Step 2: Add the three lines**

`deploy/k8s/base/env-configmap.yaml` — at the alphabetically/structurally matching position beside the sibling:

```yaml
  EVENT_TOPIC_NPC_CONVERSATION_STATUS: "EVENT_TOPIC_NPC_CONVERSATION_STATUS"
```

`deploy/k8s/overlays/pr/kustomization.yaml`:

```yaml
      - EVENT_TOPIC_NPC_CONVERSATION_STATUS=EVENT_TOPIC_NPC_CONVERSATION_STATUS-PLACEHOLDER_ATLAS_ENV
```

`deploy/k8s/overlays/main/kustomization.yaml`:

```yaml
      - EVENT_TOPIC_NPC_CONVERSATION_STATUS=EVENT_TOPIC_NPC_CONVERSATION_STATUS-main
```

Read ~10 lines of context around each insertion point first. If the overlay blocks are per-service (a `configMapGenerator` per deployment) rather than one global list, add the entry to **every** block that already contains `EVENT_TOPIC_NPC_SHOP_STATUS`, plus any `atlas-npc-conversations` block — both the producer and the consumer need it resolved. A `behavior: replace` generator drops keys not restated, so check for that marker before assuming a merge.

- [ ] **Step 3: Verify the count matches the sibling exactly**

Run:

```bash
for t in EVENT_TOPIC_NPC_SHOP_STATUS EVENT_TOPIC_NPC_CONVERSATION_STATUS; do
  echo "$t: $(grep -rc "$t" deploy/k8s/base/env-configmap.yaml deploy/k8s/overlays/pr/kustomization.yaml deploy/k8s/overlays/main/kustomization.yaml | tr '\n' ' ')"
done
```

Expected: identical counts per file for both topics. A mismatch means a block was missed.

- [ ] **Step 4: Verify kustomize still builds**

Run: `kustomize build deploy/k8s/overlays/main >/dev/null && kustomize build deploy/k8s/overlays/pr >/dev/null && echo OK`
Expected: `OK`. If `kustomize` is not installed, use `kubectl kustomize deploy/k8s/overlays/main >/dev/null`. If neither is available, verify the YAML parses instead: `python3 -c "import yaml,sys; [yaml.safe_load(open(f)) for f in sys.argv[1:]]" deploy/k8s/base/env-configmap.yaml deploy/k8s/overlays/pr/kustomization.yaml deploy/k8s/overlays/main/kustomization.yaml && echo PARSE-OK` — and say in the commit which check you ran.

- [ ] **Step 5: Commit**

```bash
git add deploy/k8s/
git commit -m "chore(deploy): register EVENT_TOPIC_NPC_CONVERSATION_STATUS

Base configmap plus both overlays, mirroring EVENT_TOPIC_NPC_SHOP_STATUS. The
env-suffixed overlay values matter: an unsuffixed topic name silently falls back
and cross-talks between environments."
```

---

### Task 11: Saga actions and payloads in `libs/atlas-saga`

**Files:**
- Modify: `libs/atlas-saga/model.go` (~`:192` action block; ~`:46-49` saga types)
- Modify: `libs/atlas-saga/payloads.go` (after `OpenNpcShopPayload`, ~`:514`)
- Modify: `libs/atlas-saga/unmarshal.go` (~`:319`, beside the `OpenNpcShop` arm)
- Test: `libs/atlas-saga/unmarshal_test.go` (mirror the block at `:1252`)

**Interfaces:**
- Consumes: nothing.
- Produces, in package `saga` (`github.com/Chronicle20/atlas/libs/atlas-saga`):
  - `StartItemConversation Action = "start_item_conversation"`
  - `StartNpcConversation Action = "start_npc_conversation"`
  - `ScriptedItemUse Type = "scripted_item_use"`
  - `RemoteNpcUse Type = "remote_npc_use"`
  - `type StartItemConversationPayload struct { CharacterId, AccountId, ItemId, NpcTemplateId uint32; Slot int16; WorldId world.Id; ChannelId channel.Id; MapId _map.Id; Instance uuid.UUID }`
  - `type StartNpcConversationPayload struct { CharacterId, AccountId, NpcTemplateId uint32; WorldId world.Id; ChannelId channel.Id; MapId _map.Id; Instance uuid.UUID }`
  - Both decode arms in `UnmarshalStep` (or whatever `unmarshal.go`'s entry point is named).
  - Consumed by Tasks 12, 13, 15, 16.

- [ ] **Step 1: Read the three anchors**

Run:

```bash
sed -n '44,50p;190,194p' libs/atlas-saga/model.go
sed -n '500,516p' libs/atlas-saga/payloads.go
sed -n '312,328p' libs/atlas-saga/unmarshal.go
sed -n '1240,1268p' libs/atlas-saga/unmarshal_test.go
```

- [ ] **Step 2: Write the failing unmarshal test**

Append to `libs/atlas-saga/unmarshal_test.go`, mirroring the `OpenNpcShopPayload` block at `:1252`:

```go
func TestUnmarshalStartItemConversationPayload(t *testing.T) {
	raw := []byte(`{
		"stepId": "start_item_conversation",
		"status": "pending",
		"action": "start_item_conversation",
		"payload": {
			"characterId": 1234,
			"accountId": 77,
			"itemId": 2430008,
			"npcTemplateId": 2084002,
			"slot": 5,
			"worldId": 0,
			"channelId": 1,
			"mapId": 100000000,
			"instance": "00000000-0000-0000-0000-000000000000"
		}
	}`)

	s := unmarshalOneStep(t, raw)

	p, ok := s.Payload.(StartItemConversationPayload)
	if !ok {
		t.Fatalf("payload type = %T, want StartItemConversationPayload", s.Payload)
	}
	if p.CharacterId != 1234 || p.ItemId != 2430008 || p.NpcTemplateId != 2084002 || p.Slot != 5 || p.AccountId != 77 {
		t.Errorf("payload round-trip: %+v", p)
	}
}

func TestUnmarshalStartNpcConversationPayload(t *testing.T) {
	raw := []byte(`{
		"stepId": "start_npc_conversation",
		"status": "pending",
		"action": "start_npc_conversation",
		"payload": {
			"characterId": 1234,
			"accountId": 77,
			"npcTemplateId": 9090002,
			"worldId": 0,
			"channelId": 1,
			"mapId": 100000000,
			"instance": "00000000-0000-0000-0000-000000000000"
		}
	}`)

	s := unmarshalOneStep(t, raw)

	p, ok := s.Payload.(StartNpcConversationPayload)
	if !ok {
		t.Fatalf("payload type = %T, want StartNpcConversationPayload", s.Payload)
	}
	if p.CharacterId != 1234 || p.NpcTemplateId != 9090002 || p.AccountId != 77 {
		t.Errorf("payload round-trip: %+v", p)
	}
}
```

`unmarshalOneStep` stands in for whatever the file's existing tests use to drive one step through the unmarshaller — read the block at `:1240-1268` and use exactly that idiom (it may unmarshal a whole `Saga` and index `Steps[0]`). Do not add a new helper if one already exists.

- [ ] **Step 3: Run it to verify it fails**

Run: `cd libs/atlas-saga && go test ./... -run 'TestUnmarshalStart(Item|Npc)Conversation' -v`
Expected: FAIL — `undefined: StartItemConversationPayload`.

- [ ] **Step 4: Add the two actions and two saga types**

In `libs/atlas-saga/model.go`, after the `// NPC shop actions` block:

```go
	// NPC conversation actions. Both are deliberately NOT self-completing: the
	// step stays Pending until EVENT_TOPIC_NPC_CONVERSATION_STATUS reports
	// STARTED or START_ERROR, which is what lets a following destroy step
	// consume the item only once the dialogue actually opened.
	//
	// Two discrete actions rather than one with a mode discriminator: the
	// orchestrator's handler dispatch is per-action, and a discriminator inside
	// the payload would move branching somewhere the compensator and
	// event-acceptance tables cannot see it.
	StartItemConversation Action = "start_item_conversation"
	StartNpcConversation  Action = "start_npc_conversation"
```

In the `Type` block, after `RemoteMerchant`:

```go
	// ScriptedItemUse is the classification-243 flow: open the item's own
	// dialogue, then consume the item — in that order, so an unauthored item
	// costs the player nothing.
	ScriptedItemUse Type = "scripted_item_use"

	// RemoteNpcUse is the classification-239 flow: open the named NPC's shop or
	// conversation from anywhere, then consume the item.
	RemoteNpcUse Type = "remote_npc_use"
```

- [ ] **Step 5: Add the two payloads**

In `libs/atlas-saga/payloads.go`, after `OpenNpcShopPayload`:

```go
// StartItemConversationPayload opens a scripted item's own dialogue (the
// 243xxxx family). Like OpenNpcShop this step is NOT self-completing: it waits
// for EVENT_TOPIC_NPC_CONVERSATION_STATUS to report STARTED or START_ERROR,
// which is what lets the following destroy step consume the item only once the
// dialogue actually opened.
//
// The ordering matters more here than for a shop: an item with no authored
// conversation must survive, and with conversation-first that falls out of the
// ordering instead of needing a rollback.
type StartItemConversationPayload struct {
	CharacterId   uint32     `json:"characterId"`   // CharacterId the dialogue opens for
	AccountId     uint32     `json:"accountId"`     // AccountId, carried into the conversation context
	ItemId        uint32     `json:"itemId"`        // Scripted item template id; the conversation lookup key
	NpcTemplateId uint32     `json:"npcTemplateId"` // Avatar the dialogue renders with (WZ spec/npc)
	Slot          int16      `json:"slot"`          // Source inventory slot, so this step and the destroy step describe one asset
	WorldId       world.Id   `json:"worldId"`
	ChannelId     channel.Id `json:"channelId"`
	MapId         _map.Id    `json:"mapId"`
	Instance      uuid.UUID  `json:"instance"`
}

// StartNpcConversationPayload opens an NPC's own conversation from anywhere
// (the 239xxxx family, conversation branch — the shop branch uses
// OpenNpcShopPayload). Also NOT self-completing, for the same reason.
type StartNpcConversationPayload struct {
	CharacterId   uint32     `json:"characterId"`
	AccountId     uint32     `json:"accountId"`
	NpcTemplateId uint32     `json:"npcTemplateId"`
	WorldId       world.Id   `json:"worldId"`
	ChannelId     channel.Id `json:"channelId"`
	MapId         _map.Id    `json:"mapId"`
	Instance      uuid.UUID  `json:"instance"`
}
```

Confirm the import aliases already present in `payloads.go` (`world`, `channel`, `_map`, `uuid`) — they are used by `OpenNpcShopPayload` and its neighbours, so no new imports should be needed.

- [ ] **Step 6: Add the two unmarshal arms**

In `libs/atlas-saga/unmarshal.go`, beside the `OpenNpcShop` arm at `:319`, following that arm's exact error-handling shape:

```go
	case StartItemConversation:
		var payload StartItemConversationPayload
		// … same decode + assign as the OpenNpcShop arm …
	case StartNpcConversation:
		var payload StartNpcConversationPayload
		// … same decode + assign …
```

Read `:312-328` and reproduce the arm body literally (it is typically a `json.Unmarshal(rawPayload, &payload)` with an error return and `step.Payload = payload`). Do not leave the `…` comments in the file.

- [ ] **Step 7: Run the tests to verify they pass**

Run: `cd libs/atlas-saga && go test -race ./... && go vet ./...`
Expected: clean, both new tests PASS.

- [ ] **Step 8: Commit**

```bash
git add libs/atlas-saga/
git commit -m "feat(atlas-saga): add start_item_conversation and start_npc_conversation actions

Both are non-self-completing, mirroring open_npc_shop: the step stays Pending
until the conversation status topic reports STARTED or START_ERROR, so a
following destroy step consumes the item only once the dialogue opened. An item
with no authored conversation therefore costs the player nothing, without
needing a rollback path.

Two discrete actions rather than one with a mode discriminator — the
orchestrator dispatches per action, and a discriminator inside the payload would
hide branching from the compensator and event-acceptance tables."
```

---

### Task 12: Mirror the conversation contract into the orchestrator, with a guard

The two copies live in separate Go modules; nothing in the compiler links them. A field name or json tag changed in one and not the other fails no build — it decodes into a zero-valued body at runtime, silently. This is the failure class `CLAUDE.md` items 13/14/15 each exist for.

**Files:**
- Create: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/npc/kafka.go` (new package — the orchestrator has `npcshop/` but no `npc/`)
- Create: `tools/npc-conversation-contract-mirror-guard.sh`
- Modify: `CLAUDE.md` (add gate 16)
- Modify: the CI workflow that runs the sibling guards

**Interfaces:**
- Consumes: the owner file from Task 9.
- Produces: package `npc` in the orchestrator (import `atlas-saga-orchestrator/kafka/message/npc`) exposing the identical `Command`, `StatusEvent`, topic and type constants. Consumed by Task 13.

- [ ] **Step 1: Create the mirror**

Copy the owner file verbatim and replace only the leading doc comment:

```bash
mkdir -p services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/npc
cp services/atlas-npc-conversations/atlas.com/npc/kafka/message/npc/kafka.go \
   services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/npc/kafka.go
```

Then prepend, **above** the `package` clause, a doc comment naming the mirror direction (this is the only permitted difference — the guard compares from `package` onward):

```go
// Mirror of services/atlas-npc-conversations/atlas.com/npc/kafka/message/npc/kafka.go.
// atlas-npc-conversations OWNS this contract; this copy exists because the two
// services are separate Go modules and nothing links them at compile time.
// Edit the owner, then replay the change here —
// tools/npc-conversation-contract-mirror-guard.sh enforces it.
```

If the owner file has its own leading comment, replace it here rather than stacking two.

- [ ] **Step 2: Verify the mirror compiles in its module**

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go build ./kafka/message/npc/`
Expected: clean. If it fails on an import (`world`, `channel`, `_map`, `stat`), the orchestrator's `go.mod` may not require `libs/atlas-constants` — check `grep -n atlas-constants go.mod` and, if missing, verify the workspace resolves it via `go.work` before adding a require.

- [ ] **Step 3: Write the guard**

Create `tools/npc-conversation-contract-mirror-guard.sh`, modelled on `tools/npc-shop-contract-mirror-guard.sh` (read it first — the `body()` awk function and `check_pair` shape are the parts to reuse):

```bash
#!/usr/bin/env bash
# npc-conversation-contract-mirror-guard.sh — enforces that the
# COMMAND_TOPIC_NPC / EVENT_TOPIC_NPC_CONVERSATION_STATUS contract is identical
# in its two copies.
#
# atlas-npc-conversations owns the contract; atlas-saga-orchestrator carries a
# mirror because the two services live in separate Go modules and nothing in the
# compiler links them. A field name or json tag changed in one copy and not the
# other does not fail any build — it decodes into a zero-valued body at runtime,
# silently: a conversation start with no item id, no avatar, or no
# transactionId, so the awaiting saga step never completes. task-230.
#
# The files are compared from their `package` clause onward: the only permitted
# difference is the leading doc comment, which names the mirror direction.
#
# Run from the repo root; drift → non-zero exit.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OWNER="$ROOT/services/atlas-npc-conversations/atlas.com/npc/kafka/message/npc/kafka.go"
SAGA_MIRROR="$ROOT/services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/npc/kafka.go"

rc=0
for f in "$OWNER" "$SAGA_MIRROR"; do
    if [ ! -f "$f" ]; then
        echo "npc-conversation-contract-mirror-guard: FAIL — missing contract file: ${f#"$ROOT"/}"
        rc=1
    fi
done
[ "$rc" -ne 0 ] && exit "$rc"

body() { awk '/^package /{p=1} p' "$1"; }

if ! diff -u <(body "$OWNER") <(body "$SAGA_MIRROR"); then
    echo "npc-conversation-contract-mirror-guard: FAIL — atlas-saga-orchestrator mirror drift (diff above)."
    exit 1
fi

echo "npc-conversation-contract-mirror-guard: OK — both copies identical."
```

Then: `chmod +x tools/npc-conversation-contract-mirror-guard.sh`

- [ ] **Step 4: Prove the guard actually catches drift**

A guard that never fails is not a guard. Run:

```bash
tools/npc-conversation-contract-mirror-guard.sh                      # expect OK, exit 0
sed -i.bak 's/"transactionId,omitempty"/"txnId,omitempty"/' \
  services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/npc/kafka.go
tools/npc-conversation-contract-mirror-guard.sh; echo "exit=$?"      # expect FAIL, exit 1
mv services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/npc/kafka.go.bak \
   services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/npc/kafka.go
tools/npc-conversation-contract-mirror-guard.sh                      # expect OK again
```

Expected: OK / `exit=1` / OK. Confirm no `.bak` file is left behind: `ls services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/npc/`.

- [ ] **Step 5: Wire the guard into CI and `CLAUDE.md`**

Find where the sibling guards run:

```bash
grep -rn "npc-shop-contract-mirror-guard" .github/ tools/ CLAUDE.md
```

Add `tools/npc-conversation-contract-mirror-guard.sh` alongside every hit in `.github/`. In `CLAUDE.md`, append a gate 16 after item 15, in the same voice:

```markdown
16. **`tools/npc-conversation-contract-mirror-guard.sh` clean from the repo root**
    whenever either copy of the npc-conversation Kafka contract changed.
    atlas-npc-conversations owns `kafka/message/npc/kafka.go`;
    atlas-saga-orchestrator carries a mirror in a separate Go module, so a field
    name or json tag changed in one and not the other fails no build — it
    decodes into a zero-valued body at runtime, silently: a conversation start
    with no item id, no avatar, or no transactionId, so the awaiting saga step
    never completes (task-230). The guard diffs the two files from their
    `package` clause onward; only the leading doc comment, which names the
    mirror direction, may differ.
```

- [ ] **Step 6: Commit**

```bash
git add services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/npc/ \
        tools/npc-conversation-contract-mirror-guard.sh CLAUDE.md .github/
git commit -m "feat(saga-orchestrator): mirror the npc-conversation contract, guarded

The orchestrator needs the COMMAND_TOPIC_NPC and
EVENT_TOPIC_NPC_CONVERSATION_STATUS types to dispatch and await conversation
starts, but it is a separate Go module from the owner. Adds the mirror plus
tools/npc-conversation-contract-mirror-guard.sh, following the three existing
contract-mirror guards. Drift-detection was verified by deliberately renaming a
json tag and confirming the guard exits 1."
```

---

### Task 13: Orchestrator wiring for both conversation actions

`OpenNpcShop` occupies six touch points in the orchestrator plus a status consumer. Both new actions follow it exactly.

**Files:**
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/model.go` (`:285` aliases, `:1333` unmarshal arm)
- Modify: `.../saga/handler.go` (`:178` interface, `:854` dispatch, `:1195-1222` impls)
- Modify: `.../saga/producer.go` (after `:398`)
- Modify: `.../saga/event_acceptance.go` (`:112`, `:177`, `:424`)
- Modify: `.../saga/character_extractor.go` (`:65`)
- Modify: `.../saga/compensator.go` (`:1508` reverse-walk switch)
- Create: `.../kafka/consumer/npcconversation/consumer.go`
- Modify: `.../main.go` (register the consumer + handlers)
- Test: `.../saga/producer_test.go`, `.../saga/character_extractor_test.go`, `.../saga/conversation_compensation_test.go`, `.../kafka/consumer/npcconversation/consumer_test.go`

**Interfaces:**
- Consumes: `StartItemConversation`, `StartNpcConversation`, both payload types (Task 11); the mirrored `npc` message package (Task 12).
- Produces:
  - `saga.StartItemConversationPayload` / `saga.StartNpcConversationPayload` aliases
  - `EventKindNpcConversationStarted EventKind = "npcconversation.started"`, `EventKindNpcConversationStartError EventKind = "npcconversation.start_error"`
  - `NpcConversationStartItemCommandProvider(transactionId uuid.UUID, payload StartItemConversationPayload) model.Provider[[]kafka.Message]`
  - `NpcConversationStartNpcCommandProvider(transactionId uuid.UUID, payload StartNpcConversationPayload) model.Provider[[]kafka.Message]`
  - `NpcConversationEndCommandProvider(transactionId uuid.UUID, characterId uint32, npcTemplateId uint32) model.Provider[[]kafka.Message]`
  - `EmitNpcConversationEnd(l, ctx, transactionId, characterId, npcTemplateId) error` — a package var, so compensator tests can observe it without a broker

- [ ] **Step 1: Read every `OpenNpcShop` touch point**

Run:

```bash
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator
grep -n "OpenNpcShop" saga/*.go
sed -n '1195,1225p' saga/handler.go
sed -n '365,405p' saga/producer.go
sed -n '108,118p;172,182p;418,430p' saga/event_acceptance.go
sed -n '1505,1522p' saga/compensator.go
cat kafka/consumer/npcshop/consumer.go
```

- [ ] **Step 2: Write the failing producer and extractor tests**

Append to `saga/producer_test.go`, mirroring the `NpcShopEnterCommandProvider` test at `:284`:

```go
func TestNpcConversationStartItemCommandProvider(t *testing.T) {
	txn := uuid.New()
	msgs, err := NpcConversationStartItemCommandProvider(txn, StartItemConversationPayload{
		CharacterId:   1234,
		AccountId:     77,
		ItemId:        2430008,
		NpcTemplateId: 2084002,
		Slot:          5,
		ChannelId:     1,
		MapId:         100000000,
	})()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("messages: got %d, want 1", len(msgs))
	}

	var c npc.Command[npc.CommandItemConversationStartBody]
	if err := json.Unmarshal(msgs[0].Value, &c); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if c.Type != npc.CommandTypeStartItemConversation {
		t.Errorf("type: got %q", c.Type)
	}
	if c.TransactionId != txn {
		t.Errorf("transactionId: got %s want %s", c.TransactionId, txn)
	}
	if c.NpcId != 2084002 {
		t.Errorf("npcId (the avatar): got %d want 2084002", c.NpcId)
	}
	if c.Body.ItemId != 2430008 || c.Body.Slot != 5 || c.Body.AccountId != 77 {
		t.Errorf("body: %+v", c.Body)
	}
}
```

Append to `saga/character_extractor_test.go`, mirroring `:23`:

```go
func TestExtractCharacterIdFromConversationPayloads(t *testing.T) {
	item := NewStep[any]("s1", Pending, StartItemConversation, StartItemConversationPayload{CharacterId: 4242, ItemId: 2430008})
	if got := ExtractCharacterId(item); got != 4242 {
		t.Errorf("item: got %d want 4242", got)
	}
	npcStep := NewStep[any]("s2", Pending, StartNpcConversation, StartNpcConversationPayload{CharacterId: 4243, NpcTemplateId: 9090002})
	if got := ExtractCharacterId(npcStep); got != 4243 {
		t.Errorf("npc: got %d want 4243", got)
	}
}
```

Use the real extractor entry-point name — read `saga/character_extractor_test.go:23` and copy how it calls the function.

- [ ] **Step 3: Run them to verify they fail**

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test ./saga/ -run 'TestNpcConversationStartItem|TestExtractCharacterIdFromConversation' -v`
Expected: FAIL — undefined identifiers.

- [ ] **Step 4: Add the aliases and unmarshal arm in `saga/model.go`**

At `:285`, beside `OpenNpcShopPayload`:

```go
	StartItemConversationPayload = sharedsaga.StartItemConversationPayload
	StartNpcConversationPayload  = sharedsaga.StartNpcConversationPayload
```

In the const re-export block, beside `OpenNpcShop`:

```go
	StartItemConversation = sharedsaga.StartItemConversation
	StartNpcConversation  = sharedsaga.StartNpcConversation
	ScriptedItemUse       = sharedsaga.ScriptedItemUse
	RemoteNpcUse          = sharedsaga.RemoteNpcUse
```

At `:1333`, beside the `OpenNpcShop` unmarshal arm, add the two arms in exactly that arm's shape.

- [ ] **Step 5: Add the producers**

In `saga/producer.go`, after `NpcShopExitCommandProvider` and its `EmitNpcShopExit` seam:

```go
// NpcConversationStartItemCommandProvider builds the COMMAND_TOPIC_NPC
// START_ITEM_CONVERSATION command for a start_item_conversation step. Keyed by
// character id, matching every other producer on this topic.
func NpcConversationStartItemCommandProvider(transactionId uuid.UUID, payload StartItemConversationPayload) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(payload.CharacterId))
	value := &npc.Command[npc.CommandItemConversationStartBody]{
		TransactionId: transactionId,
		NpcId:         payload.NpcTemplateId,
		CharacterId:   payload.CharacterId,
		Type:          npc.CommandTypeStartItemConversation,
		Body: npc.CommandItemConversationStartBody{
			WorldId:   payload.WorldId,
			ChannelId: payload.ChannelId,
			MapId:     payload.MapId,
			Instance:  payload.Instance,
			AccountId: payload.AccountId,
			ItemId:    payload.ItemId,
			Slot:      payload.Slot,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// NpcConversationStartNpcCommandProvider builds the COMMAND_TOPIC_NPC
// START_CONVERSATION command for a start_npc_conversation step. It reuses the
// existing command type — the transactionId is what makes it saga-driven, and a
// uuid.Nil one is the ordinary NPC-talk path that emits no status.
func NpcConversationStartNpcCommandProvider(transactionId uuid.UUID, payload StartNpcConversationPayload) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(payload.CharacterId))
	value := &npc.Command[npc.CommandConversationStartBody]{
		TransactionId: transactionId,
		NpcId:         payload.NpcTemplateId,
		CharacterId:   payload.CharacterId,
		Type:          npc.CommandTypeStartConversation,
		Body: npc.CommandConversationStartBody{
			WorldId:   payload.WorldId,
			ChannelId: payload.ChannelId,
			MapId:     payload.MapId,
			Instance:  payload.Instance,
			AccountId: payload.AccountId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// NpcConversationEndCommandProvider builds the END_CONVERSATION command used to
// compensate a conversation-start step whose saga later failed, so a player is
// never left standing in a dialogue for an item they still hold.
func NpcConversationEndCommandProvider(transactionId uuid.UUID, characterId uint32, npcTemplateId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &npc.Command[npc.CommandConversationEndBody]{
		TransactionId: transactionId,
		NpcId:         npcTemplateId,
		CharacterId:   characterId,
		Type:          npc.CommandTypeEndConversation,
		Body:          npc.CommandConversationEndBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

// EmitNpcConversationEnd emits the END_CONVERSATION compensation command.
// Indirected through a package var so compensator tests can observe it without
// a broker (same seam shape as EmitNpcShopExit).
var EmitNpcConversationEnd = func(l logrus.FieldLogger, ctx context.Context, transactionId uuid.UUID, characterId uint32, npcTemplateId uint32) error {
	return producer.ProviderImpl(l)(ctx)(npc.EnvCommandTopic)(NpcConversationEndCommandProvider(transactionId, characterId, npcTemplateId))
}
```

Read how `EmitNpcShopExit` is declared just below `:398` and match its exact signature style and topic-provider call.

- [ ] **Step 6: Add the handler arms**

In `saga/handler.go`: two interface methods beside `:178`, two dispatch cases beside `:854`, and two implementations after `handleOpenNpcShop`:

```go
// handleStartItemConversation handles the StartItemConversation action.
//
// Deliberately NOT self-completing (contrast handleShowStorage): the step stays
// Pending until the conversation status consumer reports STARTED or
// START_ERROR. That is the whole point of the scripted-item saga — the
// following destroy_asset_from_slot step must not run unless the dialogue
// actually opened, so an item with no authored conversation survives.
func (h *HandlerImpl) handleStartItemConversation(s Saga, st Step[any]) error {
	payload, ok := st.Payload().(StartItemConversationPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	err := producer.ProviderImpl(h.l)(h.ctx)(npc.EnvCommandTopic)(NpcConversationStartItemCommandProvider(s.TransactionId(), payload))
	if err != nil {
		h.logActionError(s, st, err, "Unable to emit item conversation start command.")
		return err
	}

	h.l.WithFields(logrus.Fields{
		"transaction_id":  s.TransactionId().String(),
		"character_id":    payload.CharacterId,
		"item_id":         payload.ItemId,
		"npc_template_id": payload.NpcTemplateId,
	}).Debug("Dispatched item conversation START; awaiting STARTED/START_ERROR.")

	return nil
}
```

and the `StartNpcConversation` twin, identical but for the payload type, the provider, and the log fields (`npc_template_id` only, no `item_id`).

- [ ] **Step 7: Add the event-acceptance entries**

In `saga/event_acceptance.go`, beside the `NpcShop` entries:

```go
	EventKindNpcConversationStarted    EventKind = "npcconversation.started"
	EventKindNpcConversationStartError EventKind = "npcconversation.start_error"
```

in the action→kinds map (`:177`):

```go
	sharedsaga.StartItemConversation: {EventKindNpcConversationStarted, EventKindNpcConversationStartError},
	sharedsaga.StartNpcConversation:  {EventKindNpcConversationStarted, EventKindNpcConversationStartError},
```

and in the kind→outcome map (`:424`):

```go
	EventKindNpcConversationStarted:    OutcomeSuccess,
	EventKindNpcConversationStartError: OutcomeFailure,
```

- [ ] **Step 8: Add the character-extractor arms**

In `saga/character_extractor.go`, beside `case OpenNpcShopPayload` (`:65`):

```go
	case StartItemConversationPayload:
		return p.CharacterId
	case StartNpcConversationPayload:
		return p.CharacterId
```

- [ ] **Step 9: Add the compensator arms**

In `saga/compensator.go`, in the reverse-walk switch beside `case OpenNpcShop` (`:1508`):

```go
		case StartItemConversation:
			// The inverse of "opened a dialogue" is "close it". Because the
			// destroy is the LAST step, the only path that reaches here is
			// "conversation opened, destroy failed" — rare, and its
			// compensation is a UI teardown rather than an item restore. That
			// asymmetry is the point of the conversation-first ordering.
			if payload, ok := step.Payload().(StartItemConversationPayload); ok {
				if err := EmitNpcConversationEnd(c.l, c.ctx, s.TransactionId(), payload.CharacterId, payload.NpcTemplateId); err != nil {
					c.l.WithError(err).WithFields(logrus.Fields{
						"transaction_id": s.TransactionId().String(),
						"step_id":        step.StepId(),
						"character_id":   payload.CharacterId,
						"item_id":        payload.ItemId,
					}).Error("Reverse-walk: StartItemConversation -> END_CONVERSATION dispatch failed; continuing chain.")
				}
			}
		case StartNpcConversation:
			if payload, ok := step.Payload().(StartNpcConversationPayload); ok {
				if err := EmitNpcConversationEnd(c.l, c.ctx, s.TransactionId(), payload.CharacterId, payload.NpcTemplateId); err != nil {
					c.l.WithError(err).WithFields(logrus.Fields{
						"transaction_id": s.TransactionId().String(),
						"step_id":        step.StepId(),
						"character_id":   payload.CharacterId,
					}).Error("Reverse-walk: StartNpcConversation -> END_CONVERSATION dispatch failed; continuing chain.")
				}
			}
```

Confirm this switch is reached for the `ScriptedItemUse` / `RemoteNpcUse` saga types. Read how the compensator selects a reverse-walk function per saga type (`grep -n "RemoteMerchant\|DispatchRemoteMerchantRollbacks\|func (c \*CompensatorImpl) Compensate" saga/compensator.go | head -20`) and register the two new saga types to the same reverse-walk path `RemoteMerchant` uses. If the dispatch is a `switch s.SagaType()`, add both new types to that arm.

- [ ] **Step 10: Write the compensation test**

Create `saga/conversation_compensation_test.go`, mirroring `saga/remote_merchant_compensation_test.go` (read it first — it shows how `EmitNpcShopExit` is stubbed):

```go
// Conversation-first ordering means the only compensable path is "dialogue
// opened, destroy failed". Compensation is a UI teardown, not an item restore.
func TestScriptedItemUseCompensationEndsTheConversation(t *testing.T) {
	var gotCharacter, gotNpc uint32
	var called int
	orig := EmitNpcConversationEnd
	EmitNpcConversationEnd = func(_ logrus.FieldLogger, _ context.Context, _ uuid.UUID, characterId uint32, npcTemplateId uint32) error {
		called++
		gotCharacter, gotNpc = characterId, npcTemplateId
		return nil
	}
	t.Cleanup(func() { EmitNpcConversationEnd = orig })

	s := NewBuilder(). // … match the builder idiom in remote_merchant_compensation_test.go …
		AddStep("start_item_conversation", Completed, StartItemConversation, StartItemConversationPayload{
			CharacterId:   1234,
			ItemId:        2430008,
			NpcTemplateId: 2084002,
		}).
		AddStep("consume_scripted_item", Failed, DestroyAssetFromSlot, DestroyAssetFromSlotPayload{
			CharacterId: 1234,
			Slot:        5,
		}).
		Build()

	// … invoke the same compensator entry point remote_merchant_compensation_test.go uses …

	if called != 1 {
		t.Fatalf("END_CONVERSATION emitted %d times, want 1", called)
	}
	if gotCharacter != 1234 || gotNpc != 2084002 {
		t.Errorf("emitted for character %d npc %d", gotCharacter, gotNpc)
	}
}
```

Fill in the builder and invocation from the sibling test — do not commit the `…` comments.

- [ ] **Step 11: Write the status consumer**

Create `kafka/consumer/npcconversation/consumer.go`, mirroring `kafka/consumer/npcshop/consumer.go` file-for-file:

```go
package npcconversation

import (
	consumer2 "atlas-saga-orchestrator/kafka/consumer"
	npc "atlas-saga-orchestrator/kafka/message/npc"
	"atlas-saga-orchestrator/saga"
	"context"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("npc_conversation_status_event")(npc.EnvStatusEventTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		t, _ = topic.EnvProvider(l)(npc.EnvStatusEventTopic)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStartedEvent))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStartErrorEvent))); err != nil {
			return err
		}
		return nil
	}
}

// handleStartedEvent completes a pending conversation-start step. The ordinary
// NPC-talk path emits nothing at all (uuid.Nil commands are not reported), so
// every event reaching this handler should match a saga — but the nil check and
// AcceptEvent both decline any that do not.
//
// A redelivered STARTED for an already-completed step is declined by
// AcceptEvent, which is the idempotency guarantee the at-least-once topic needs.
func handleStartedEvent(l logrus.FieldLogger, ctx context.Context, e npc.StatusEvent[npc.StatusEventStartedBody]) {
	if e.Type != npc.StatusEventTypeStarted {
		return
	}
	if e.TransactionId == uuid.Nil {
		return
	}
	p := saga.NewProcessor(l, ctx)
	if _, ok := p.AcceptEvent(e.TransactionId, saga.EventKindNpcConversationStarted); !ok {
		return
	}

	l.WithFields(logrus.Fields{
		"transaction_id":  e.TransactionId.String(),
		"character_id":    e.CharacterId,
		"npc_template_id": e.Body.NpcTemplateId,
		"source_id":       e.Body.SourceId,
	}).Debug("Conversation started; completing conversation-start step.")

	_ = p.StepCompleted(e.TransactionId, true)
}

// handleStartErrorEvent fails the step, which fails the saga, which means the
// following destroy_asset_from_slot never runs — the player keeps the item.
// reason distinguishes a content gap (NO_CONVERSATION_AUTHORED) from a real
// fault without reading code.
func handleStartErrorEvent(l logrus.FieldLogger, ctx context.Context, e npc.StatusEvent[npc.StatusEventStartErrorBody]) {
	if e.Type != npc.StatusEventTypeStartError {
		return
	}
	if e.TransactionId == uuid.Nil {
		return
	}
	p := saga.NewProcessor(l, ctx)
	if _, ok := p.AcceptEvent(e.TransactionId, saga.EventKindNpcConversationStartError); !ok {
		return
	}

	l.WithFields(logrus.Fields{
		"transaction_id":  e.TransactionId.String(),
		"character_id":    e.CharacterId,
		"npc_template_id": e.Body.NpcTemplateId,
		"source_id":       e.Body.SourceId,
		"reason":          e.Body.Reason,
	}).Warn("Conversation start failed; failing conversation-start step. Item not consumed.")

	_ = p.StepCompleted(e.TransactionId, false)
}
```

Also create `kafka/consumer/npcconversation/consumer_test.go` and `testmain_test.go` by mirroring the `npcshop` pair — the tests assert that a `uuid.Nil` event is declined and that a matching event completes/fails the step.

- [ ] **Step 12: Register the consumer in `main.go`**

Run: `grep -n "npcshop" services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/main.go`

Add the parallel `npcconversation.InitConsumers(l)(cmf)(consumerGroupId)` and `npcconversation.InitHandlers(l)(consumer.GetManager().RegisterHandler)` lines beside every `npcshop` line, with the same error handling.

- [ ] **Step 13: Run the tests to verify they pass**

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go build ./... && go test -race ./... && go vet ./...`
Expected: clean.

- [ ] **Step 14: Run the mirror guard**

Run: `tools/npc-conversation-contract-mirror-guard.sh`
Expected: `OK`. If it fails, you edited the mirror instead of the owner — revert the mirror and replay from the owner.

- [ ] **Step 15: Commit**

```bash
git add services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/
git commit -m "feat(saga-orchestrator): wire start_item_conversation and start_npc_conversation

Six touch points each, following open_npc_shop exactly: payload aliases and
unmarshal arms, non-self-completing handler arms, command producers, event
acceptance (npcconversation.started / .start_error), character extraction, and
reverse-walk compensation via END_CONVERSATION. Adds the
EVENT_TOPIC_NPC_CONVERSATION_STATUS consumer that completes or fails the
awaiting step.

Because the destroy is the LAST step, the only path reaching compensation is
'dialogue opened, destroy failed' — its compensation is a UI teardown rather
than an item restore, which is the point of the conversation-first ordering."
```

---

### Task 14: Surface `Npc`, `Script`, and `RunOnPickup` on the channel's consumable model

The channel's consumable `RestModel` carries only `spec`, so the handlers cannot read the avatar. `atlas-data` already serves all three fields.

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/data/consumable/rest.go`
- Modify: `services/atlas-channel/atlas.com/channel/data/consumable/model.go`
- Modify: `services/atlas-channel/atlas.com/channel/saga/model.go` (payload + saga-type re-exports)
- Test: `services/atlas-channel/atlas.com/channel/data/consumable/processor_test.go`

**Interfaces:**
- Consumes: `atlas-data`'s consumable REST fields `npc`, `script`, `runOnPickup` (Task 1 makes their values correct); `saga.StartItemConversationPayload`, `saga.StartNpcConversationPayload`, `saga.ScriptedItemUse`, `saga.RemoteNpcUse` (Task 11).
- Produces:
  - `consumable.Model` gains `Npc() uint32`, `Script() string`, `RunOnPickup() bool`
  - `atlas-channel/saga` re-exports `StartItemConversationPayload`, `StartNpcConversationPayload`, `ScriptedItemUse`, `RemoteNpcUse`
  - Consumed by Tasks 15 and 16.

- [ ] **Step 1: Write the failing test**

Read `services/atlas-channel/atlas.com/channel/data/consumable/processor_test.go` for the existing fixture idiom, then append:

```go
// The 243 handler resolves the dialogue's avatar from the consumable's npc
// field. Until task-230 the channel's RestModel carried only `spec`, so the
// value never reached the handler even after atlas-data started parsing it
// correctly.
func TestConsumableExtractCarriesNpcScriptAndRunOnPickup(t *testing.T) {
	rm := RestModel{
		Id:          2430008,
		Spec:        map[SpecType]int32{},
		Npc:         2084002,
		Script:      "compassUse",
		RunOnPickup: false,
	}
	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if m.Npc() != 2084002 {
		t.Errorf("Npc: got %d want 2084002", m.Npc())
	}
	if m.Script() != "compassUse" {
		t.Errorf("Script: got %q want %q", m.Script(), "compassUse")
	}
	if m.RunOnPickup() {
		t.Error("RunOnPickup: got true want false")
	}
}

// The json tags must match what atlas-data serves
// (services/atlas-data/atlas.com/data/consumable/rest.go:74-76). A mismatch
// decodes to zero silently and looks exactly like a content gap.
func TestConsumableRestModelJsonTags(t *testing.T) {
	var rm RestModel
	if err := json.Unmarshal([]byte(`{"npc":2084002,"script":"compassUse","runOnPickup":true,"spec":{}}`), &rm); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if rm.Npc != 2084002 || rm.Script != "compassUse" || !rm.RunOnPickup {
		t.Errorf("decoded: %+v", rm)
	}
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./data/consumable/ -v`
Expected: FAIL — `unknown field Npc in struct literal`.

- [ ] **Step 3: Extend `rest.go` and `model.go`**

`rest.go`:

```go
type RestModel struct {
	Id   uint32             `json:"-"`
	Spec map[SpecType]int32 `json:"spec"`
	// Npc is the NPC template a scripted item's dialogue renders with (the
	// 243xxxx family, WZ spec/npc) or the NPC a remote-NPC item summons (the
	// 239xxxx family, WZ info/npc). Tags must match atlas-data's
	// consumable/rest.go exactly — a mismatch decodes to zero silently and is
	// indistinguishable from a content gap.
	Npc uint32 `json:"npc"`
	// Script is the WZ spec/script value. Recorded for authoring traceability
	// only; conversations are keyed by item id, never by script name.
	Script      string `json:"script"`
	RunOnPickup bool   `json:"runOnPickup"`
}
```

and in `Extract`:

```go
func Extract(rm RestModel) (Model, error) {
	return Model{
		id:          rm.Id,
		spec:        rm.Spec,
		npc:         rm.Npc,
		script:      rm.Script,
		runOnPickup: rm.RunOnPickup,
	}, nil
}
```

`model.go`:

```go
type Model struct {
	id          uint32
	spec        map[SpecType]int32
	npc         uint32
	script      string
	runOnPickup bool
}

func (m Model) Npc() uint32       { return m.npc }
func (m Model) Script() string    { return m.script }
func (m Model) RunOnPickup() bool { return m.runOnPickup }
```

- [ ] **Step 4: Add the saga re-exports**

In `services/atlas-channel/atlas.com/channel/saga/model.go`, in the payload alias block beside `OpenNpcShopPayload` (`:54`):

```go
	// NPC conversation payload types
	StartItemConversationPayload = sharedsaga.StartItemConversationPayload
	StartNpcConversationPayload  = sharedsaga.StartNpcConversationPayload
```

and in the const block beside `RemoteMerchant` (`:74`):

```go
	ScriptedItemUse = sharedsaga.ScriptedItemUse
	RemoteNpcUse    = sharedsaga.RemoteNpcUse
```

Check whether the file also re-exports action constants (`grep -n "OpenNpcShop\b" services/atlas-channel/atlas.com/channel/saga/model.go`); if it does, add `StartItemConversation` and `StartNpcConversation` there too — the handlers in Tasks 15/16 reference them as `saga.StartItemConversation`.

- [ ] **Step 5: Run the tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test -race ./data/consumable/ ./saga/ -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/data/consumable/ services/atlas-channel/atlas.com/channel/saga/model.go
git commit -m "feat(channel): surface consumable npc/script/runOnPickup and conversation saga types

The channel's consumable RestModel carried only spec, so the scripted-item
handler could not resolve the avatar its dialogue renders with even though
atlas-data serves the field. Adds the three fields with json tags matching
atlas-data exactly, plus the conversation payload and saga-type re-exports the
new handlers need."
```

---

### Task 15: `ScriptedItemHandleFunc` — the `243` route

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/scripted_item.go`
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/scripted_item_test.go`
- Modify: `services/atlas-channel/atlas.com/channel/main.go` (~`:966`)

**Interfaces:**
- Consumes: `invsb.ScriptedItem` + `invsb.ScriptedItemHandle` (Task 3); `item.ClassificationConsumableScriptedItem` (Task 2); `consumable.Model.Npc()` (Task 14); `saga.StartItemConversationPayload`, `saga.ScriptedItemUse`, `saga.StartItemConversation` (Tasks 11/14).
- Produces: `handler.ScriptedItemHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{})`, registered under `invsb.ScriptedItemHandle`. Package var `scriptedItemSagaCreateFunc` as the test seam.

- [ ] **Step 1: Read the two structural models**

Run:

```bash
cd services/atlas-channel/atlas.com/channel
cat socket/handler/shop_scanner_item_use.go
sed -n '60,175p' socket/handler/character_cash_item_use_remote_merchant.go
cat socket/handler/character_cash_item_use_remote_merchant_test.go | head -60
```

The first gives the decode→classify→validate-slot skeleton; the second gives the saga construction, the `enableActions` closure, the structured-log fields, and the `remoteMerchantSagaCreateFunc` test seam.

- [ ] **Step 2: Write the failing tests**

Create `socket/handler/scripted_item_test.go`, mirroring the remote-merchant handler test's setup:

```go
// Every rejection path must unlock the client and consume nothing. The success
// path must NOT unlock: the destroy step's inventory delta is what clears the
// client's m_bExclRequestSent, and an explicit unlock as well would
// double-resolve the lock.
func TestScriptedItem_RejectsNonScriptedClassification(t *testing.T) {
	// itemId 2000000 is a 200-class consumable, not 243. Impossible from a
	// legitimate client — the sender is gated on itemId/10000 == 243.
	// Expect: no saga created, EnableActions called once.
}

// v95 alone whitelists 3994225 (Evolving Ring Upgrade Potion, an Install/Setup
// item). Supporting it needs setup/reader.go spec parsing plus a second
// inventory type on the destroy step, so it is a documented gap — but the
// rejection must NAME it, so a play-test report is self-explaining.
func TestScriptedItem_RejectsItem3994225ByName(t *testing.T) {
	// Expect: no saga, EnableActions called once, and the warn log mentions
	// 3994225 explicitly.
}

func TestScriptedItem_RejectsSlotTemplateMismatch(t *testing.T) {
	// GetItemInSlot returns a different template than the packet claims.
	// Expect: no saga, EnableActions called once.
}

// Until the atlas-data re-ingest lands, every tenant is in exactly this state.
// The log line must say so rather than presenting as a mysterious content gap.
func TestScriptedItem_RejectsNpcZeroAndNamesReingest(t *testing.T) {
	// consumable data returns Npc == 0.
	// Expect: no saga, EnableActions called once, warn mentions re-ingest.
}

// The happy path builds exactly two steps in this order. Conversation FIRST:
// an item with no authored conversation gets START_ERROR, the destroy never
// runs, and the player keeps the item — no rollback required.
func TestScriptedItem_CreatesConversationThenDestroySaga(t *testing.T) {
	// Expect: one saga, SagaType == saga.ScriptedItemUse, two steps:
	//   [0] StartItemConversation with ItemId/NpcTemplateId/Slot/AccountId set
	//   [1] DestroyAssetFromSlot, InventoryType Use, Quantity 1, TemplateId set
	// and EnableActions NOT called.
}

func TestScriptedItem_UnlocksWhenSagaCreationFails(t *testing.T) {
	// scriptedItemSagaCreateFunc returns an error.
	// Expect: EnableActions called once.
}
```

Write the real bodies. Stub `scriptedItemSagaCreateFunc` and the consumable-data lookup through package vars the same way `remoteMerchantSagaCreateFunc` and `cashItemDataFunc` are stubbed in the remote-merchant test, and count `EnableActions` calls through whatever seam that test already uses for the writer producer.

Resolve the numeric value of the USE inventory type before asserting on it: `grep -n "TypeValueUse" libs/atlas-constants/inventory/*.go`. Assert against the constant, not a literal.

- [ ] **Step 3: Run them to verify they fail**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run TestScriptedItem -v`
Expected: FAIL — `undefined: ScriptedItemHandleFunc`.

- [ ] **Step 4: Write the handler**

Create `socket/handler/scripted_item.go`:

```go
package handler

import (
	character2 "atlas-channel/character"
	consumabledata "atlas-channel/data/consumable"
	"atlas-channel/saga"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	invsb "github.com/Chronicle20/atlas/libs/atlas-packet/inventory/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// scriptedItemSagaCreateFunc is a test seam for saga creation.
var scriptedItemSagaCreateFunc = func(l logrus.FieldLogger, ctx context.Context, s saga.Saga) error {
	return saga.NewProcessor(l, ctx).Create(s)
}

// scriptedItemDataFunc is a test seam for the consumable data lookup.
var scriptedItemDataFunc = func(l logrus.FieldLogger, ctx context.Context, itemId uint32) (consumabledata.Model, error) {
	return consumabledata.NewProcessor(l, ctx).GetById(itemId)
}

// evolvingRingUpgradePotionId is the one item outside the 243 family that a v95
// client will send this op for (CWvsContext::SendScriptRunItemRequest @0x9de7a0
// gates on nItemID/10000 == 243 || nItemID == 3994225). It is an Install/Setup
// item and is a documented out-of-scope gap: supporting it needs spec parsing in
// atlas-data's setup reader — which parses no spec node at all today — plus a
// second inventory type on the destroy step.
const evolvingRingUpgradePotionId = 3994225

// ScriptedItemHandleFunc handles CWvsContext::SendScriptRunItemRequest — the
// 243xxxx scripted-item route. The item carries its own dialogue, keyed by item
// id and rendered with the avatar named in its WZ spec/npc node.
//
// Ordering is conversation-first: the saga opens the dialogue and only then
// consumes. An item with no authored conversation therefore survives via
// START_ERROR rather than needing a rollback, and there is no pre-flight round
// trip and no TOCTOU window.
//
// Excl-request contract: every rejection path unlocks explicitly; the success
// path does not, because the destroy step's inventory delta is what clears the
// client's m_bExclRequestSent. An explicit unlock as well would double-resolve
// the lock.
func ScriptedItemHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := invsb.ScriptedItem{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		enableActions := func() {
			_ = session.EnableActions(l)(ctx)(wp)(s)
		}

		itemId := item.Id(p.ItemId())

		if uint32(itemId) == evolvingRingUpgradePotionId {
			l.Warnf("Character [%d] used item [%d] (Evolving Ring Upgrade Potion). v95's client whitelists this id alongside the 243 family, but it is an Install/Setup item and is a known out-of-scope gap (task-230 design D-3): atlas-data's setup reader parses no spec node, so its script and npc are unavailable. Not consuming.", s.CharacterId(), itemId)
			enableActions()
			return
		}

		if item.GetClassification(itemId) != item.ClassificationConsumableScriptedItem {
			l.Warnf("Character [%d] attempted scripted item use with non-scripted item [%d]. The client gates this op on itemId/10000 == 243, so this is impossible from a legitimate client. Not consuming.", s.CharacterId(), itemId)
			enableActions()
			return
		}

		a, err := character2.NewProcessor(l, ctx).GetItemInSlot(s.CharacterId(), inventory.TypeValueUse, p.Source())()
		if err != nil || item.Id(a.TemplateId()) != itemId {
			l.Warnf("Character [%d] attempted to use scripted item [%d] in slot [%d], but item not found or mismatched. Not consuming.", s.CharacterId(), itemId, p.Source())
			enableActions()
			return
		}

		cd, err := scriptedItemDataFunc(l, ctx, uint32(itemId))
		if err != nil {
			l.WithError(err).Errorf("Character [%d] scripted item [%d]: unable to read consumable data.", s.CharacterId(), itemId)
			enableActions()
			return
		}
		if cd.Npc() == 0 {
			l.Warnf("Character [%d] scripted item [%d] resolves to npc 0; no avatar to render the dialogue with. Every 0243 item authors npc under its spec node — if atlas-data has not been re-ingested since that parser fix, this is expected and re-ingest is the fix. Not consuming.", s.CharacterId(), itemId)
			enableActions()
			return
		}

		f := s.Field()
		now := time.Now()
		transactionId := uuid.New()

		// Conversation FIRST, destroy SECOND. The two dominant failure modes —
		// no conversation authored, and character already in a conversation —
		// both fail step 1, so step 2 never runs and the item is intact. The
		// PRD's consume-first ordering would have needed a rollback for both.
		sg := saga.Saga{
			TransactionId: transactionId,
			SagaType:      saga.ScriptedItemUse,
			InitiatedBy:   "SCRIPTED_ITEM",
			Steps: []saga.Step{
				{
					StepId: "start_item_conversation",
					Status: saga.Pending,
					Action: saga.StartItemConversation,
					Payload: saga.StartItemConversationPayload{
						CharacterId:   s.CharacterId(),
						AccountId:     s.AccountId(),
						ItemId:        uint32(itemId),
						NpcTemplateId: cd.Npc(),
						Slot:          int16(p.Source()),
						WorldId:       f.WorldId(),
						ChannelId:     f.ChannelId(),
						MapId:         f.MapId(),
						Instance:      f.Instance(),
					},
					CreatedAt: now,
					UpdatedAt: now,
				},
				{
					StepId: "consume_scripted_item",
					Status: saga.Pending,
					Action: saga.DestroyAssetFromSlot,
					Payload: saga.DestroyAssetFromSlotPayload{
						CharacterId:   s.CharacterId(),
						InventoryType: byte(inventory.TypeValueUse),
						Slot:          int16(p.Source()),
						Quantity:      1,
						ShowEffect:    false,
						TemplateId:    uint32(itemId),
					},
					CreatedAt: now,
					UpdatedAt: now,
				},
			},
		}

		if err := scriptedItemSagaCreateFunc(l, ctx, sg); err != nil {
			l.WithError(err).Errorf("Character [%d] scripted item [%d]: unable to create saga.", s.CharacterId(), itemId)
			enableActions()
			return
		}

		l.WithFields(logrus.Fields{
			"character_id":    s.CharacterId(),
			"item_id":         uint32(itemId),
			"slot":            p.Source(),
			"npc_template_id": cd.Npc(),
			"script_name":     cd.Script(),
			"transaction_id":  transactionId.String(),
		}).Info("Scripted item conversation requested.")
	}
}
```

Two things to verify against real code before this compiles:

- `DestroyAssetFromSlotPayload.InventoryType`'s type — the remote-merchant handler passes a bare `5` with a `// cash` comment. Read `grep -n "InventoryType" libs/atlas-saga/payloads.go` and use the correct type; if it is `byte`, the conversion above is right, otherwise drop it.
- `field.Model` accessors — `f.Instance()` may not exist. Check `grep -n "func (m Model)" libs/atlas-constants/field/model.go` and use whatever the instance accessor is actually called (or omit `Instance` if the field model does not carry one).

- [ ] **Step 5: Register the handler**

In `services/atlas-channel/atlas.com/channel/main.go`, after `:966`:

```go
	handlerMap[invsb.ScriptedItemHandle] = handler.ScriptedItemHandleFunc
```

Check whether `main.go` already imports `libs/atlas-packet/inventory/serverbound` and under which alias (`grep -n "atlas-packet/inventory/serverbound" services/atlas-channel/atlas.com/channel/main.go`); reuse that alias rather than adding a second import of the same package.

- [ ] **Step 6: Run the tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test -race ./socket/handler/ -run TestScriptedItem -v`
Expected: all PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/scripted_item.go \
        services/atlas-channel/atlas.com/channel/socket/handler/scripted_item_test.go \
        services/atlas-channel/atlas.com/channel/main.go
git commit -m "feat(channel): handle SCRIPTED_ITEM (243 family)

Validates classification, slot/template ownership, and a non-zero avatar, then
creates a two-step saga: start_item_conversation, then
destroy_asset_from_slot. Conversation-first means an unauthored item fails at
step 1 and survives, with no rollback path and no TOCTOU window.

Every rejection unlocks the client explicitly; the success path does not,
because the destroy step's inventory delta is what clears m_bExclRequestSent.
Item 3994225 (v95's extra whitelist entry) is rejected by name so a play-test
report explains itself."
```

---

### Task 16: `NpcItemUseHandleFunc` — the `239` / `545` route

`239` resolves an NPC and takes exactly the dispatch `npc_start_conversation.go:31-42` already performs: probe for a shop, open it if one exists, otherwise start the NPC conversation. `545` always takes the shop path.

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/npc_item_use.go`
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/npc_item_use_test.go`
- Modify: `services/atlas-channel/atlas.com/channel/main.go`

**Interfaces:**
- Consumes: `invsb.NpcItemUse` + `invsb.NpcItemUseHandle` (Task 4); `item.ClassificationConsumableRemoteNpc`, `item.ClassificationRemoteMerchant` (Task 2 + existing); `consumable.Model.Npc()` (Task 14); `shops.NewProcessor(l, ctx).GetShop` (existing); `saga.StartNpcConversationPayload`, `saga.OpenNpcShopPayload`, `saga.RemoteNpcUse`, `saga.RemoteMerchant` (Tasks 11/14).
- Produces: `handler.NpcItemUseHandleFunc(...)` with the same signature as `ScriptedItemHandleFunc`, registered under `invsb.NpcItemUseHandle`. Package var `npcItemUseSagaCreateFunc` as the test seam.

- [ ] **Step 1: Read the shop-probe dispatch you are reusing**

Run: `sed -n '28,45p' services/atlas-channel/atlas.com/channel/socket/handler/npc_start_conversation.go`

The probe is `sp.GetShop(n.Template())`; `err == nil` means a shop exists. Reuse that exact predicate — do not invent a different "has shop" check.

- [ ] **Step 2: Write the failing tests**

Create `socket/handler/npc_item_use_test.go`:

```go
// 239 with a shop: open_npc_shop then destroy, from the USE inventory.
func TestNpcItemUse_RemoteNpcWithShopOpensShop(t *testing.T) {}

// 239 without a shop: start_npc_conversation then destroy.
func TestNpcItemUse_RemoteNpcWithoutShopStartsConversation(t *testing.T) {}

// 545 always takes the shop path, from the CASH inventory.
func TestNpcItemUse_RemoteMerchantOpensShopFromCashInventory(t *testing.T) {}

// An unhandled classification is impossible from a legitimate client — the
// sender gates on itemId/10000 in {239, 545}.
func TestNpcItemUse_RejectsOtherClassifications(t *testing.T) {}

func TestNpcItemUse_RejectsSlotTemplateMismatch(t *testing.T) {}

func TestNpcItemUse_RejectsNpcZero(t *testing.T) {}

func TestNpcItemUse_UnlocksWhenSagaCreationFails(t *testing.T) {}

// Both routes must stay live on v72-v95: CDraggableItem::OnDoubleClicked decides
// whether a 545 item goes out as CASH_ITEM_USE or NPC_ITEM_USE_REQUEST, so
// neither handler may assume it is the only path.
func TestNpcItemUse_RemoteMerchantDoesNotDependOnRemoteMerchantEnabled(t *testing.T) {
	// A v61 tenant — for which remoteMerchantEnabled() is deliberately false,
	// because 545 sits in that version's CASH_ITEM_USE dispatcher default arm —
	// must still open the shop through THIS route.
}
```

Write real bodies for all eight. The last one is the v61 behavioural gain the design calls out; it must be asserted, not assumed.

- [ ] **Step 3: Run them to verify they fail**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run TestNpcItemUse -v`
Expected: FAIL — `undefined: NpcItemUseHandleFunc`.

- [ ] **Step 4: Write the handler**

Create `socket/handler/npc_item_use.go`. Structure:

```go
// NpcItemUseHandleFunc handles CWvsContext::SendSelectNpcItemUseRequest, which
// covers two item families that share one opcode:
//
//   239xxxx — remote-NPC summons. The item names an NPC in its info/npc node;
//             open that NPC's shop if it has one, otherwise its conversation.
//   545xxxx — remote merchant. Always a shop.
//
// Dispatch is classification-first, never slot-type-first, for the reason
// character_cash_item_use.go already documents: type bytes collide.
//
// Interaction with the CASH_ITEM_USE route (task-221): on v72-v95 both opcodes
// exist and CDraggableItem::OnDoubleClicked decides which one a 545 item goes
// out as. The server accepts BOTH; neither handler may assume it is the only
// path. On v61 this is the ONLY route for 545 — remoteMerchantEnabled() is
// correctly false there because 545 sits in that version's CASH_ITEM_USE
// dispatcher default arm — so this handler must not consult that predicate.
func NpcItemUseHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := invsb.NpcItemUse{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		enableActions := func() { _ = session.EnableActions(l)(ctx)(wp)(s) }

		itemId := item.Id(p.ItemId())

		switch item.GetClassification(itemId) {
		case item.ClassificationConsumableRemoteNpc:
			handleRemoteNpcItemUse(l, ctx, s, p, itemId, enableActions)
		case item.ClassificationRemoteMerchant:
			handleRemoteMerchantItemUse(l, ctx, s, p, itemId, enableActions)
		default:
			l.Warnf("Character [%d] attempted npc item use with item [%d] of an unhandled classification. The client gates this op on itemId/10000 in {239, 545}, so this is impossible from a legitimate client. Not consuming.", s.CharacterId(), itemId)
			enableActions()
		}
	}
}
```

`handleRemoteNpcItemUse`:
1. `character2.NewProcessor(l, ctx).GetItemInSlot(s.CharacterId(), inventory.TypeValueUse, p.Source())()`; reject on error or template mismatch.
2. `cd, err := scriptedItemDataFunc(l, ctx, uint32(itemId))` — reuse the Task 15 seam; reject on error or `cd.Npc() == 0`.
3. `_, shopErr := shops.NewProcessor(l, ctx).GetShop(cd.Npc())`.
4. If `shopErr == nil`, build a `saga.RemoteNpcUse` saga with steps `open_npc_shop` (`saga.OpenNpcShopPayload{CharacterId, WorldId, ChannelId, NpcTemplateId: cd.Npc()}`) then `consume_remote_npc_item` (`DestroyAssetFromSlot`, `InventoryType` = USE, `Quantity: 1`, `TemplateId`).
5. Otherwise the first step is `start_npc_conversation` with `saga.StartNpcConversationPayload{CharacterId, AccountId, NpcTemplateId: cd.Npc(), WorldId, ChannelId, MapId, Instance}`, same second step.
6. On saga-create failure: log, `enableActions()`, return.
7. On success: `l.WithFields(...).Info(...)` with `character_id`, `item_id`, `slot`, `npc_template_id`, `route` (`"shop"` or `"conversation"`), `transaction_id`. No unlock.

`handleRemoteMerchantItemUse`:
1. Validate the slot in the **cash** inventory, matching how `character_cash_item_use.go` validates ownership for the cash route — read `sed -n '45,70p' services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go` and reuse that lookup rather than the USE-inventory one.
2. Resolve the NPC from cash item data via the same `cashItemDataFunc` seam the remote-merchant handler uses; reject on error or `ci.Npc == 0`.
3. Build a `saga.RemoteMerchant` saga: `open_npc_shop` then `consume_remote_merchant_item` (`DestroyAssetFromSlot`, `InventoryType` = the cash value the existing handler passes, `Quantity: 1`, `TemplateId`).
4. Populate the `remotemerchant.GetRegistry().Put(...)` entry **before** creating the saga, exactly as `character_cash_item_use_remote_merchant.go:113-117` does with its "Registry first: a very fast ENTERED must not arrive before the entry that tells the shop consumer to unlock this client" comment — and `ClearCharacter` on saga-create failure. Without this the shop consumer will not know to unlock.
5. Same logging and unlock discipline.

Do **not** call `remoteMerchantEnabled(t)` anywhere in this file.

- [ ] **Step 5: Register the handler**

In `services/atlas-channel/atlas.com/channel/main.go`, beside the Task 15 line:

```go
	handlerMap[invsb.NpcItemUseHandle] = handler.NpcItemUseHandleFunc
```

- [ ] **Step 6: Run the tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test -race ./socket/handler/ -run 'TestNpcItemUse|TestScriptedItem' -v`
Expected: all PASS.

- [ ] **Step 7: Run the full channel suite**

Run: `cd services/atlas-channel/atlas.com/channel && go test -race ./... && go vet ./...`
Expected: clean. In particular the pre-existing `character_cash_item_use_remote_merchant_test.go` must still pass — this task adds a second route to the same outcome and must not have changed the first.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/npc_item_use.go \
        services/atlas-channel/atlas.com/channel/socket/handler/npc_item_use_test.go \
        services/atlas-channel/atlas.com/channel/main.go
git commit -m "feat(channel): handle NPC_ITEM_USE_REQUEST (239 and 545 families)

239 resolves its NPC from info/npc and takes the same shop-or-conversation
dispatch NPCStartConversationHandleFunc already performs; 545 always takes the
shop path and reuses task-221's open_npc_shop action and remote-merchant
registry entry.

Both CASH_ITEM_USE and NPC_ITEM_USE_REQUEST reach a 545 item on v72-v95 —
CDraggableItem::OnDoubleClicked picks — so neither handler assumes it is the
only path. On v61 this is the only route, which incidentally brings the remote
merchant to that version; the handler therefore does not consult
remoteMerchantEnabled."
```

---

### Task 17: Reference conversation content

Two items, chosen for distinct avatars and no side effects, seeded across all eight versions that carry `SCRIPTED_ITEM`.

**Files:**
- Create: `deploy/seed/gms/{72_1,79_1,83_1,84_1,87_1,92_1,95_1}/npc-conversations/items/item-2430008.json` and `item-2430013.json`
- Create: `deploy/seed/jms/185_1/npc-conversations/items/item-2430008.json` and `item-2430013.json`

**Interfaces:**
- Consumes: the `item.RestModel` JSON:API shape and the `^item-(\d+)\.json$` filename pattern (Task 7).
- Produces: 16 seed files, loaded by the `item.conversation` subdomain at `npc-conversations/items`.

- [ ] **Step 1: Confirm the two items against the recorded inventory**

Run: `grep -n "2430008\|2430013" docs/tasks/task-230-scripted-items/item-inventory.md`

Expected:

```
| `2430008` | Golden Compass | `compassUse` | `2084002` | — |
| `2430013` | Peng Peng Popsicle | `item_2430013` | `9010000` | — |
```

Distinct avatars (`2084002` vs `9010000`), so a play-test proves the avatar is carried per-item rather than defaulted. Neither carries `runOnPickup`.

Do **not** substitute other item ids. `2430010` (`openTreasure`) is deliberately excluded — it is the only `runOnPickup` item, and that is a *pickup* trigger this task does not implement; leaving it unseeded keeps the distinction observable now that Task 1 makes the flag visible for the first time.

- [ ] **Step 2: Read the seed envelope and the state vocabulary**

Run:

```bash
head -50 "$(ls deploy/seed/gms/83_1/npc-conversations/npc/*.json | head -1)"
```

The envelope is `{"data": {"attributes": { … }}}`. The two dialogue node types you need are `dialogue` (with `dialogueType`, `text`, `choices`) and `listSelection` (with `title`, `choices`). A `choices` entry with `"nextState": null` terminates the conversation.

**No new `StateType` may be introduced.** These conversations use exactly the vocabulary that already exists.

- [ ] **Step 3: Write the two conversation files**

`deploy/seed/gms/83_1/npc-conversations/items/item-2430013.json` — a plain two-node dialogue:

```json
{
  "data": {
    "attributes": {
      "itemId": 2430013,
      "npcId": 9010000,
      "scriptName": "item_2430013",
      "startState": "intro",
      "states": [
        {
          "id": "intro",
          "type": "dialogue",
          "dialogue": {
            "dialogueType": "sendNext",
            "text": "A Peng Peng Popsicle! It's so cold it makes your teeth ache just looking at it.",
            "choices": [
              { "nextState": "outro", "text": "Next" },
              { "nextState": null, "text": "Exit" }
            ]
          }
        },
        {
          "id": "outro",
          "type": "dialogue",
          "dialogue": {
            "dialogueType": "sendOk",
            "text": "You finish it in three bites and immediately regret how fast that went.",
            "choices": [
              { "nextState": null, "text": "OK" }
            ]
          }
        }
      ]
    }
  }
}
```

`deploy/seed/gms/83_1/npc-conversations/items/item-2430008.json` — a branching dialogue on a different avatar:

```json
{
  "data": {
    "attributes": {
      "itemId": 2430008,
      "npcId": 2084002,
      "scriptName": "compassUse",
      "startState": "intro",
      "states": [
        {
          "id": "intro",
          "type": "listSelection",
          "listSelection": {
            "title": "The Golden Compass needle spins lazily, then settles. What would you like to do?",
            "choices": [
              { "nextState": "readNeedle", "text": "Read the needle." },
              { "nextState": "shakeIt", "text": "Give it a shake." },
              { "nextState": null, "text": "Put it away." }
            ]
          }
        },
        {
          "id": "readNeedle",
          "type": "dialogue",
          "dialogue": {
            "dialogueType": "sendOk",
            "text": "North. Reliably, stubbornly north. The compass is very pleased with itself.",
            "choices": [
              { "nextState": null, "text": "OK" }
            ]
          }
        },
        {
          "id": "shakeIt",
          "type": "dialogue",
          "dialogue": {
            "dialogueType": "sendOk",
            "text": "The needle wobbles, spins twice, and points north again. It was never going to be anything else.",
            "choices": [
              { "nextState": null, "text": "OK" }
            ]
          }
        }
      ]
    }
  }
}
```

Neither conversation warps, grants items, or touches quest state — those are the three selection criteria, and a warping outcome in particular would test the excl-request unlock contract and the dispatch path at once, making any failure ambiguous.

**Validate the attribute names against `item.RestModel`** (Task 7) before writing all sixteen files: the json tags are `itemId`, `npcId`, `scriptName`, `startState`, `states`. If the state-level shape differs from the `npc-*.json` files you read in Step 2 (e.g. the field is `dialogue` vs something else), match the existing NPC conversation files — they are the working reference.

**These are authored fresh, not ported.** Atlas conversations are declarative JSON state machines, and the WZ `script` value is traceability metadata, not a lookup key. Of the 23 scripts, only `killarmush` and `removethorns` exist in the local Cosmic tree, and neither of these two is among them — **no claim is made about what `compassUse` or `item_2430013` originally did.** A later content pass can re-author the state machine with no schema or code change.

- [ ] **Step 4: Replicate to the other seven version directories**

```bash
for v in gms/72_1 gms/79_1 gms/84_1 gms/87_1 gms/92_1 gms/95_1 jms/185_1; do
  mkdir -p "deploy/seed/$v/npc-conversations/items"
  cp deploy/seed/gms/83_1/npc-conversations/items/item-2430008.json \
     deploy/seed/gms/83_1/npc-conversations/items/item-2430013.json \
     "deploy/seed/$v/npc-conversations/items/"
done
ls deploy/seed/*/*/npc-conversations/items/ | grep -c json
```

Expected: 16 (2 files × 8 version dirs). `gms/12_1`, `gms/48_1`, and `gms/61_1` get **nothing** — none of them carry `SCRIPTED_ITEM`.

- [ ] **Step 5: Validate the JSON and run catalog-lint**

Run:

```bash
for f in deploy/seed/*/*/npc-conversations/items/*.json; do python3 -m json.tool "$f" >/dev/null || echo "BAD: $f"; done
cd tools/catalog-lint && go run . ; echo "exit=$?"
```

Expected: no `BAD:` lines; catalog-lint exits 0. If catalog-lint reports an unknown subdomain path, the Task 7 Step 13 rule was not added.

Do **not** hand-edit any `CATALOG_REVISION` file — it holds a commit sha bumped by the image-overlay automation, and catalog-lint only requires it to be non-empty.

- [ ] **Step 6: Commit**

```bash
git add deploy/seed/
git commit -m "feat(seed): reference scripted-item conversations for 2430008 and 2430013

Two items with distinct avatars (2084002 and 9010000), so a play-test proves the
avatar is carried per-item rather than defaulted. Neither warps, grants items,
nor touches quest state — a warping outcome would exercise the excl-request
unlock contract and the dispatch path at once and make any failure ambiguous.

Seeded for the eight versions that carry SCRIPTED_ITEM. Authored fresh, not
ported: the WZ script value is traceability metadata, not a lookup key, and no
claim is made about what these scripts originally did. 2430010 (openTreasure)
is deliberately unseeded — it is the only runOnPickup item, a pickup trigger
this task does not implement."
```

---

### Task 18: Coverage manifest, matrix promotion, and the 17-cell verification fan-out

The matrix is the deliverable, not a side effect. Seventeen cells must promote: `SCRIPTED_ITEM` × 8 versions, `NPC_ITEM_USE_REQUEST` × 9 versions.

**Files:**
- Create: `docs/tasks/task-230-scripted-items/coverage-manifest.yaml`
- Modify: `docs/packets/audits/support/gms_v61.md`, `gms_v72.md`, `gms_v79.md` (`n-a` → real coverage state — regenerated, not hand-edited)
- Create: `docs/packets/evidence/<version>/inventory.serverbound.*.yaml` (17 records, written by the verifier)
- Create: `docs/packets/audits/<version>/InventoryScriptedItem.{json,md}`, `InventoryNpcItemUse.{json,md}`
- Modify: `libs/atlas-packet/inventory/serverbound/scripted_item_test.go`, `npc_item_use_test.go` (`packet-audit:verify` markers)
- Modify: `docs/packets/audits/STATUS.md`, `docs/packets/audits/status.json` (regenerated)

**Interfaces:**
- Consumes: the codecs (Tasks 3/4), the registry entries and fname linkage (Task 5), the template bindings (Task 6).
- Produces: a matrix with no `❌` in any in-scope column and no residual incorrect `⬜`.

- [ ] **Step 1: Write the coverage manifest**

Create `docs/tasks/task-230-scripted-items/coverage-manifest.yaml`:

```yaml
# coverage-manifest
ops:
  - SCRIPTED_ITEM
  - NPC_ITEM_USE_REQUEST
versions:
  - gms_v61
  - gms_v72
  - gms_v79
  - gms_v83
  - gms_v84
  - gms_v87
  - gms_v92
  - gms_v95
  - jms_v185
fields:
  - "inventory/serverbound/InventoryScriptedItem: no version gates — body identical on all eight versions that carry the op (full ten-IDB sweep)"
  - "inventory/serverbound/InventoryNpcItemUse: no version gates — body identical on all nine versions; no leading updateTime on any"
out_of_scope:
  # SCRIPTED_ITEM is absent from gms_v61 (and v12/v48); only NPC_ITEM_USE_REQUEST
  # is claimed on that column. Both ops stay n-a on gms_v48.
  - inventory/serverbound/InventoryLotteryItemUse
  - inventory/serverbound/InventoryItemUse
```

The `out_of_scope` entries are a claim that touching those neighbouring codecs is intentional and needs no verification. **Only list them if the diff actually touches them** — if `gofmt` did not move those files, delete both lines rather than pre-emptively silencing the critic.

- [ ] **Step 2: Generate the audit reports**

Per `VERIFYING_A_PACKET.md` §9, a serverbound cell needs three agreeing artifacts and a routed template. The reports are generated deterministically from the committed export:

```bash
for v in gms_v61 gms_v72 gms_v79 gms_v83 gms_v84 gms_v87 gms_v92 gms_v95 jms_v185; do
  case "$v" in
    jms_v185) tmpl=template_jms_185_1.json; exp=gms_jms_185.json ;;
    *)        tmpl="template_${v/gms_v/gms_}_1.json"; exp="$v.json" ;;
  esac
  echo "=== $v ($tmpl / $exp)"
done
```

Confirm the actual export filenames first (`ls docs/packets/ida-exports/`) — the mapping above is a guess at the naming and must be corrected against reality before running the generator. Then, per version:

```bash
go run ./tools/packet-audit \
  -csv-clientbound "docs/packets/MapleStory Ops - ClientBound.csv" \
  -csv-serverbound "docs/packets/MapleStory Ops - ServerBound.csv" \
  -template services/atlas-configurations/seed-data/templates/<tmpl> \
  -ida-source docs/packets/ida-exports/<export> \
  -output /tmp/rpt
cp /tmp/rpt/<version>/InventoryScriptedItem.{json,md} docs/packets/audits/<version>/    # where applicable
cp /tmp/rpt/<version>/InventoryNpcItemUse.{json,md}   docs/packets/audits/<version>/
```

Copy **only** the two reports per version — never bulk-copy the output directory, which would overwrite unrelated reports.

If report generation fails with `delegate to COutPacket: not in export`, that is the known harvest artifact: strip that one call from the export entry (it is the packet constructor, not a wire read). **Never re-run `packet-audit export` and overwrite a committed export** — it is not idempotent and drifts ~150 unrelated function keys.

- [ ] **Step 3: Verify the cells one at a time**

For each of the 17 cells, drive the single-cell procedure via the `/verify-packet` command (or the `packet-verifier` agent), which follows [`docs/packets/audits/VERIFYING_A_PACKET.md`](../../packets/audits/VERIFYING_A_PACKET.md). The cells:

| Op | Versions |
|---|---|
| `SCRIPTED_ITEM` | `gms_v72`, `gms_v79`, `gms_v83`, `gms_v84`, `gms_v87`, `gms_v92`, `gms_v95`, `jms_v185` |
| `NPC_ITEM_USE_REQUEST` | `gms_v61`, `gms_v72`, `gms_v79`, `gms_v83`, `gms_v84`, `gms_v87`, `gms_v92`, `gms_v95`, `jms_v185` |

Batch the fan-out **per IDB** — `ida-pro-mcp` sessions are resolved by binary name and one agent at a time per database. Pin verify subagents to Sonnet or Haiku, not an expensive model.

The addresses each cell's evidence must cite are already derived and recorded in design §1.3 — use that table, do not re-derive:

```
SCRIPTED_ITEM              NPC_ITEM_USE_REQUEST
gms_v61  —                 gms_v61  0x83778d  0x066
gms_v72  0x9044d8  0x04D   gms_v72  0x90a5ac  0x06E
gms_v79  0x955840  0x04C   gms_v79  0x95b96c  0x06D
gms_v83  0xa09b26  0x04E   gms_v83  0xa10075  0x06F
gms_v84  0xa53f08  0x04E   gms_v84  0xa5a4b2  0x06F
gms_v87  0xa9f3d2  0x051   gms_v87  0xaa5a85  0x072
gms_v92  0x9b3da0  0x055   gms_v92  0x9aff40  0x07A
gms_v95  0x9de7a0  0x054   gms_v95  0x9da430  0x07B
jms_v185 0xaee7ce  0x046   jms_v185 0xaf43ee  0x06A
```

Note `0xa9f3d2` (v87 `SCRIPTED_ITEM`) and `0xaa5a85` (v87 `NPC_ITEM_USE_REQUEST`) are adjacent and easily transposed.

Each verified cell adds a `// packet-audit:verify packet=inventory/serverbound/Inventory<Struct> version=<v> ida=<addr>` marker to the codec's test file and pins an evidence record. **A round-trip Encode→Decode fixture is not a verification** — the fixture bytes must be hand-computed from the decompiled read order, with the decompile line cited per field.

A cell that does not promote is a failure report, not a prose claim. If a cell cannot be promoted, stop and report it rather than marking it done.

- [ ] **Step 4: Regenerate the support docs**

The three legacy support docs currently record `n-a` for these ops. They are **generated**, not hand-maintained — find the generator and run it:

```bash
go run ./tools/packet-audit --help 2>&1 | head -40
grep -rn "audits/support" tools/packet-audit/ | head -5
```

Regenerate `docs/packets/audits/support/gms_v61.md`, `gms_v72.md`, `gms_v79.md` and confirm the rows changed from `n-a` to a real coverage state:

```bash
grep -n "SCRIPTED_ITEM\|NPC_ITEM_USE_REQUEST" docs/packets/audits/support/gms_v61.md docs/packets/audits/support/gms_v72.md docs/packets/audits/support/gms_v79.md
```

Expected: `gms_v61` shows `SCRIPTED_ITEM … n-a` (correct — the op is genuinely absent) and `NPC_ITEM_USE_REQUEST … 0x066 verified`; `gms_v72` and `gms_v79` show both ops verified with their opcodes.

`gms_v92` has no support doc in the directory. Do not create one — that gap predates this task and is out of scope.

- [ ] **Step 5: Regenerate the matrix — as the last content commit**

The matrix `toolSha` reads git HEAD, so regeneration must come after every other change on the branch, not mid-branch.

```bash
go run ./tools/packet-audit matrix
go run ./tools/packet-audit matrix --check ; echo "exit=$?"
go run ./tools/packet-audit fname-doc --check ; echo "exit=$?"
go run ./tools/packet-audit operations --check ; echo "exit=$?"
go run ./tools/packet-audit dispatcher-lint ; echo "exit=$?"
```

Expected: every command exits 0. `matrix --check` fails on any 🟥 conflict, orphan, dangling evidence, stale-drift finding, or a stale committed `STATUS.md`/`status.json`.

- [ ] **Step 6: Confirm the two rows**

Run: `grep -n "SCRIPTED_ITEM\|NPC_ITEM_USE_REQUEST" docs/packets/audits/STATUS.md`

Expected, in the two rows:

- `SCRIPTED_ITEM` — `⬜` for v48 and v61 (genuinely absent), `✅` for v72, v79, v83, v84, v87, v92, v95, jms. **No `❌` anywhere.**
- `NPC_ITEM_USE_REQUEST` — `⬜` for v48 only, `✅` for v61 through jms. **No `❌` anywhere.**

If either row still shows `⬜` for v72/v79 (or v61 on the second row), the matrix correction did not land and the task is not done.

- [ ] **Step 7: Commit**

```bash
git add docs/tasks/task-230-scripted-items/coverage-manifest.yaml \
        docs/packets/evidence/ docs/packets/audits/ \
        libs/atlas-packet/inventory/serverbound/
git commit -m "feat(packets): verify SCRIPTED_ITEM x8 and NPC_ITEM_USE_REQUEST x9

Promotes 17 matrix cells with byte fixtures traced to decompiled read order,
pinned evidence, and generated audit reports. Corrects five cells previously
recorded n-a: SCRIPTED_ITEM on v72/v79 and NPC_ITEM_USE_REQUEST on v61/v72/v79
— all five senders exist in their binaries.

v48 stays n-a for both ops on the dense-naming absence evidence; v61 stays n-a
for SCRIPTED_ITEM for the same reason. v12 is not a matrix column, so FR-1.1 is
discharged by the recorded evidence with no matrix edit."
```

---

### Task 19: Full verification gates and pre-PR review

**Files:** none created. This task runs the gates and fixes what they find.

**Interfaces:**
- Consumes: everything.
- Produces: `docs/tasks/task-230-scripted-items/audit.md` and `completeness-critic.md`.

- [ ] **Step 1: Confirm you are in the right worktree on the right branch**

Run:

```bash
git rev-parse --show-toplevel   # must end with /.worktrees/task-230-scripted-items
git branch --show-current       # must be task-230-scripted-items
git status --short
```

If either is wrong, STOP and report BLOCKED.

- [ ] **Step 2: Per-module test, vet, and build**

Run, in each changed module:

```bash
for m in libs/atlas-packet libs/atlas-saga libs/atlas-constants \
         services/atlas-data/atlas.com/data \
         services/atlas-channel/atlas.com/channel \
         services/atlas-npc-conversations/atlas.com/npc \
         services/atlas-saga-orchestrator/atlas.com/saga-orchestrator \
         tools/packet-audit tools/catalog-lint; do
  echo "=== $m"
  (cd "$m" && go build ./... && go vet ./... && go test -race ./...) || echo "FAIL: $m"
done
```

Expected: no `FAIL:` lines.

- [ ] **Step 3: Repo-root guards**

Run:

```bash
tools/redis-key-guard.sh
tools/goroutine-guard.sh
tools/template-opcode-order-guard.sh
tools/template-duplicate-binding-guard.sh
tools/template-movement-types-guard.sh
tools/trade-contract-mirror-guard.sh
tools/mist-contract-mirror-guard.sh
tools/npc-shop-contract-mirror-guard.sh
tools/npc-conversation-contract-mirror-guard.sh
tools/skill-job-id-guard.sh
tools/buff-duration-guard.sh
```

Expected: all exit 0.

- [ ] **Step 4: Lint**

Run: `tools/lint.sh` (fix mode) then `tools/lint.sh --check`
Expected: `--check` exits 0.

`--check` false-fails without nvm active and contends on a cross-worktree golangci-lint lock. If it fails on something outside this branch's diff, confirm nvm is loaded and no sibling worktree is mid-lint before investigating the finding.

- [ ] **Step 5: Docker bake every service whose `go.mod` moved**

Run: `git diff --name-only main...HEAD -- '**/go.mod' '**/go.sum' go.work`

For each service that appears, run `docker buildx bake atlas-<svc>` **from the worktree root**. `go build` against the workspace will not catch a missing `COPY libs/...` in the shared Dockerfile — only bake will. Expected candidates: `atlas-channel`, `atlas-npc-conversations`, `atlas-saga-orchestrator`, `atlas-data`.

If no `go.mod` moved, say so explicitly rather than silently skipping the step.

Expected: every bake succeeds.

- [ ] **Step 6: Packet-audit gates**

Run:

```bash
go run ./tools/packet-audit matrix --check
go run ./tools/packet-audit fname-doc --check
go run ./tools/packet-audit operations --check
go run ./tools/packet-audit dispatcher-lint
```

Expected: all exit 0. If the matrix went stale because a later commit changed HEAD, regenerate (`go run ./tools/packet-audit matrix`) and amend the Task 18 commit rather than adding a second matrix commit.

- [ ] **Step 7: Code review**

Invoke `superpowers:requesting-code-review`. It dispatches `plan-adherence-reviewer` and `backend-guidelines-reviewer` (no TS changed, so no frontend reviewer). Additionally dispatch `packet-completeness-critic` against `docs/tasks/task-230-scripted-items/coverage-manifest.yaml`.

Pin every review subagent to Sonnet or Haiku — not an expensive model.

Ensure each subagent operates inside this worktree and writes to `docs/tasks/task-230-scripted-items/`, never into the main repo. After the run: `git -C . status --short` and confirm nothing landed outside this tree.

Read `audit.md` and `completeness-critic.md` and fix every finding. The two the critic exists to catch:

- **CHANGED-BUT-UNCLAIMED** — a codec or gate moved that the manifest never declared. Either add it to `ops` and verify it, or justify it under `out_of_scope`.
- **CLAIMED-BUT-UNVERIFIED** — a manifest `op × version` with no verified cell. Verify it or remove the claim.

- [ ] **Step 8: Manual play-test**

Design §11 requires v83 **and** one legacy column (v72 or v79 — both are newly claimed for both ops). Before testing, confirm two deployment steps landed, because both are silent failures:

1. **`atlas-data` re-ingest.** `GET /api/data/consumables/2430008` must return `npc: 2084002` and `script: "compassUse"`. A non-zero `npc` is the proof. Without re-ingest the parser fix from Task 1 changes nothing and the feature is dead.
2. **Live tenant socket config PATCHed** with both new handler entries. Seed-template edits are not inherited by already-provisioned tenants; verify the handlers appear in the *live* config, not just the template, and that each carries a validator.

Then cover:

- [ ] A scripted item with an authored conversation opens the dialogue with the **correct avatar** and consumes exactly one. Test both `2430013` (npc `9010000`) and `2430008` (npc `2084002`) — two different avatars is the point.
- [ ] A scripted item with **no** authored conversation logs a warn, consumes nothing, and leaves the client responsive.
- [ ] A slot/template mismatch is rejected and consumes nothing.
- [ ] On v95, item `3994225` is rejected with the log line naming it explicitly.
- [ ] A redelivered command does not double-consume.
- [ ] A `239` item opens its NPC's shop or conversation and consumes one.
- [ ] **v61 remote merchant opens via the new route** — a real behavioural gain that must be observed, not assumed.

Play-test primarily on v83 and one legacy column. Treat any v87 anomaly as suspect until reproduced elsewhere: that version's movement decode already spews `Code 254` at ~2k/min independently of this work, and it is a standing confound for any v87 report.

Report actual outcomes. If something fails, say so with the output.

- [ ] **Step 9: Final state check**

Run:

```bash
git status --short          # clean
git log --oneline main..HEAD
```

Expected: a clean tree and one commit per task. Only then is the branch ready for `superpowers:finishing-a-development-branch`.

---

## Self-Review Notes

Recorded so a reader can see what was checked rather than trusting that it was.

**Spec coverage.** Every design section maps to a task: §1 evidence → Tasks 5/18; §2 (the two features are distinct) → the whole task split, and the "no updateTime" guard test in Task 4; §3.1 → Task 1; §3.2 → Task 2; §3.3 (item `3994225`) → Task 15 Step 4 and its dedicated test; §4.1 → Tasks 3/4; §4.2 → Tasks 5/6/18; §5.1 → Tasks 7/8; §5.2 (`239` reuses what exists) → Task 16; §6.1 (ordering) → Tasks 11/15; §6.2 (status topic) → Tasks 9/10/12; §6.3 (actions/payloads/wiring) → Tasks 11/13; §6.4 (compensation) → Task 13; §6.5 (idempotency) → Task 8; §7.1 → Task 15; §7.2 → Task 16; §7.3 (excl-request) → Global Constraints plus the per-handler tests; §8 (content) → Task 17; §10 (risks) → Task 19 Step 8; §11 (verification) → Tasks 18/19. PRD FR-1.1 needs no matrix action (v12 is not a column) and is discharged by the recorded evidence — Task 18's commit message states this.

**Two decisions the design deferred, settled here.** Codec package = `libs/atlas-packet/inventory/serverbound/` (Task 3, rationale in `context.md` §2). The `239` start command reuses the existing `START_CONVERSATION` type with a transactionId rather than adding a second command type (Task 9, rationale in `context.md` §3.1); the saga *actions* stay discrete as the design requires.

**Two design statements corrected against the tree.** The REST path is `/items/conversations`, not `/item-conversations` — the house pattern is `/quests/conversations` with `quest-conversations` as the JSON:API type, so the design's resource-type name survives and the path follows the existing convention (`context.md` §4.3). And the consumable fields must be added to `atlas-channel/data/consumable`, not `atlas-consumables`/`atlas-inventory` as PRD §7 supposed — that is the model the handler actually reads through (Task 14).

**Type consistency.** `ScriptedItemHandle`/`NpcItemUseHandle` are used identically in Tasks 3/4 (definition), 6 (template `handler` value), and 15/16 (`handlerMap` key). `StartItemConversationPayload`'s field set is defined once in Task 11 and consumed unchanged in Tasks 13 and 15. `EventKindNpcConversationStarted`/`…StartError` appear only in Task 13. `ErrConversationInProgress`/`ErrAlreadyStartedByThisTransaction` are defined in Task 8 and consumed in Task 9. `emitConversationStatus` (Task 9) and `EmitNpcConversationEnd` (Task 13) are distinct seams in different services.

**Where the plan tells the implementer to check rather than assert.** Several call sites could not be confirmed at plan time and are marked with an explicit "verify this before using it" step: `test.Logger()`'s real name, `conversation.NewStateModelBuilder`'s required setters, `field.Model`'s instance accessor, `DestroyAssetFromSlotPayload.InventoryType`'s type, the registry `provenance` vocabulary, the IDA export filenames, and whether `dispatcher_lint_test.go`'s `candidatesFromFName` is a mirror needing the same two cases. These are genuine unknowns, not placeholders — each names the exact command that resolves it.


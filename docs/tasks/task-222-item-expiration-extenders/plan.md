# Item-Expiration Extenders (Magical Sandglass) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make dragging a Magical Sandglass (item classification 550) onto a time-limited equipped item extend that item's expiration by the sandglass's WZ `addTime`, bounded by `now + maxDays`, consuming the sandglass exactly once and atomically.

**Architecture:** A new arm inside the existing `CASH_ITEM_USE` dispatcher in atlas-channel decodes a bare `int16` equip position, resolves the sandglass's `addTime`/`maxDays` from atlas-data and the target's `notExtend` flag from atlas-data's equipment resource, evaluates seven eligibility gates plus the extension formula in a pure function, and creates a two-step `ExpirationExtenderUse` saga (destroy the sandglass, then extend the target). atlas-saga-orchestrator dispatches the second step as a new `EXTEND_EXPIRATION` compartment command; atlas-inventory independently re-derives the cap from the sandglass's cash data, writes the expiration without touching flags, and emits the existing asset `UPDATED` event, which atlas-channel already turns into an `INVENTORY_OPERATION` that refreshes the client tooltip.

**Tech Stack:** Go 1.24 workspace (`go.work`), GORM + Postgres (atlas-inventory), Kafka via `message.Buffer`/`producer.Provider`, JSON:API over `libs/atlas-rest`, immutable models with Builder pattern, `libs/atlas-packet` codecs, `libs/atlas-saga` shared saga contract.

## Global Constraints

These apply to **every** task below.

- **Design decisions already signed off** (design §7): **D1 = reject** over-cap uses (never clamp-and-consume); **D2 = carry `ExtenderTemplateId`** on the payload and command body so atlas-inventory re-derives the cap independently; **D3 = rename** the reused codec to `ItemUseTargetSlot`.
- **No bare wire literals.** The `CashSlotItemType` for classification 550 is 61 on GMS < 95 and 62 on GMS >= 95. It MUST only ever appear inside `expirationExtenderCashSlotItemType(t)` and the existing `GetCashSlotItemType` classifier (CLAUDE.md DOM-25).
- **Never reuse `asset.ApplyLock`** for this feature. It unconditionally adds `FlagLock` and it rejects exactly the asset shape this feature targets (`services/atlas-inventory/atlas.com/inventory/asset/processor.go:332`, `errors.New("asset has a non-lock expiration")`).
- **Set-to-absolute, never increment.** Kafka delivery in this cluster is at-least-once; a duration-shaped command would stack on redelivery.
- **No `// TODO`, no stubs, no 501s** in landed commits (CLAUDE.md).
- **Repo-relative paths only** in committed files — never a literal home or absolute path.
- **Preserve line endings**; do not normalize CRLF to LF as a side effect.
- Per-task verification runs `go test -race ./...` and `go vet ./...` in the module touched. The full cross-module sweep (`docker buildx bake`, `tools/lint.sh --check`, and the guards) runs once in Task 15.
- **All commands below are run from the worktree root** — the directory containing this task's `docs/tasks/task-222-item-expiration-extenders/`. Paths shown after `cd` are relative to it.

## File Structure

| File | Responsibility | Task |
|---|---|---|
| `services/atlas-data/atlas.com/data/cash/reader.go` | parse `info/addTime`, `info/maxDays` | 1 |
| `services/atlas-data/atlas.com/data/cash/rest.go` | expose `addTime`, `maxDays` | 1 |
| `services/atlas-data/atlas.com/data/equipment/reader.go` | parse `info/notExtend` | 2 |
| `services/atlas-data/atlas.com/data/equipment/rest.go` | expose `notExtend` | 2 |
| `libs/atlas-constants/item/constants.go` | `ClassificationExpirationExtender` (550) | 3 |
| `libs/atlas-packet/cash/serverbound/item_use_target_slot.go` | the shared `int16`-slot sub-body codec (renamed) | 4 |
| `libs/atlas-saga/model.go`, `payloads.go`, `unmarshal.go` | saga type, action, payload, decode arm | 5 |
| `services/atlas-inventory/.../kafka/message/compartment/kafka.go` | `EXTEND_EXPIRATION` command contract | 6 |
| `services/atlas-inventory/.../data/cash/` | atlas-data cash client (for cap re-derivation) | 6 |
| `services/atlas-inventory/.../asset/processor.go` | `ExtendExpiration` — expiration-only, flag-preserving | 7 |
| `services/atlas-inventory/.../compartment/processor.go` | `ExtendAssetExpiration` + server-side cap re-validation | 8 |
| `services/atlas-inventory/.../kafka/consumer/compartment/consumer.go` | consume `EXTEND_EXPIRATION` | 9 |
| `services/atlas-saga-orchestrator/.../kafka/message/compartment/kafka.go` | mirrored command contract | 10 |
| `services/atlas-saga-orchestrator/.../compartment/producer.go`, `processor.go` | `RequestExtendExpiration` | 10 |
| `services/atlas-saga-orchestrator/.../saga/model.go`, `handler.go`, `event_acceptance.go` | aliases, step handler, acceptance entry | 11 |
| `services/atlas-saga-orchestrator/.../saga/timer.go`, `compensator.go` | timeout + compensation registration | 12 |
| `services/atlas-channel/.../data/cash/rest.go` | mirror `addTime`/`maxDays` | 13 |
| `services/atlas-channel/.../data/equipment/{rest,model}.go` | mirror `notExtend` | 13 |
| `services/atlas-channel/.../socket/handler/character_cash_item_use_expiration_extender.go` | version resolver + pure gate/formula evaluation | 14 |
| `services/atlas-channel/.../socket/handler/character_cash_item_use.go` | the dispatcher arm | 14 |

---

### Task 1: atlas-data parses `addTime` and `maxDays`

**Files:**
- Modify: `services/atlas-data/atlas.com/data/cash/reader.go:78`
- Modify: `services/atlas-data/atlas.com/data/cash/rest.go:38-49`
- Test: `services/atlas-data/atlas.com/data/cash/reader_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces: `cash.RestModel.AddTime uint32` (json `addTime,omitempty`) and `cash.RestModel.MaxDays uint32` (json `maxDays,omitempty`). Task 13 mirrors these on the channel model; Task 6 mirrors them on the atlas-inventory client.

- [ ] **Step 1: Write the failing test**

Append to `services/atlas-data/atlas.com/data/cash/reader_test.go`. The five ids and values are the v83 extract from `Item.wz/Cash/0550.img.xml` recorded in prd.md §4; `05500009` is the absent-field default case (it exists only in this fixture, not in v83 WZ).

```go
const testSandglassXML = `<?xml version="1.0" encoding="UTF-8"?>
<imgdir name="0550.img">
  <imgdir name="05500000">
    <imgdir name="info">
      <int name="cash" value="1"/>
      <int name="slotMax" value="1"/>
      <int name="addTime" value="86400"/>
      <int name="maxDays" value="30"/>
    </imgdir>
  </imgdir>
  <imgdir name="05500001">
    <imgdir name="info">
      <int name="cash" value="1"/>
      <int name="slotMax" value="1"/>
      <int name="addTime" value="604800"/>
      <int name="maxDays" value="30"/>
    </imgdir>
  </imgdir>
  <imgdir name="05500002">
    <imgdir name="info">
      <int name="cash" value="1"/>
      <int name="slotMax" value="1"/>
      <int name="addTime" value="1728000"/>
      <int name="maxDays" value="30"/>
    </imgdir>
  </imgdir>
  <imgdir name="05500005">
    <imgdir name="info">
      <int name="cash" value="1"/>
      <int name="slotMax" value="1"/>
      <int name="addTime" value="4320000"/>
      <int name="maxDays" value="30"/>
    </imgdir>
  </imgdir>
  <imgdir name="05500006">
    <imgdir name="info">
      <int name="cash" value="1"/>
      <int name="slotMax" value="1"/>
      <int name="addTime" value="8553600"/>
      <int name="maxDays" value="30"/>
    </imgdir>
  </imgdir>
  <imgdir name="05500009">
    <imgdir name="info">
      <int name="cash" value="1"/>
      <int name="slotMax" value="1"/>
    </imgdir>
  </imgdir>
</imgdir>`

func TestReaderSandglassAddTimeAndMaxDays(t *testing.T) {
	l, _ := test.NewNullLogger()
	rms := Read(l)(xml.FromByteArrayProvider([]byte(testSandglassXML)))
	rmm, err := model.CollectToMap[RestModel, string, RestModel](rms, RestModel.GetID, Identity)()
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		id      int
		addTime uint32
		maxDays uint32
	}{
		{5500000, 86400, 30},
		{5500001, 604800, 30},
		{5500002, 1728000, 30},
		{5500005, 4320000, 30},
		{5500006, 8553600, 30},
		{5500009, 0, 0}, // both fields absent -> default 0
	}
	for _, c := range cases {
		rm := rmm[strconv.Itoa(c.id)]
		if rm.AddTime != c.addTime {
			t.Errorf("AddTime(%d) = %d, want %d", c.id, rm.AddTime, c.addTime)
		}
		if rm.MaxDays != c.maxDays {
			t.Errorf("MaxDays(%d) = %d, want %d", c.id, rm.MaxDays, c.maxDays)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-data/atlas.com/data && go test ./cash/ -run TestReaderSandglassAddTimeAndMaxDays -v
```

Expected: FAIL — compile error, `rm.AddTime undefined (type RestModel has no field or method AddTime)`.

- [ ] **Step 3: Add the two REST fields**

In `services/atlas-data/atlas.com/data/cash/rest.go`, inside `type RestModel struct`, add after the `ProtectTime` line:

```go
	// AddTime is info/addTime in SECONDS — the expiration grant of an
	// item-expiration extender (Magical Sandglass, classification 550). The
	// client multiplies it by 10^7 into FILETIME 100ns units
	// (CDraggableItem::ModifyEquipItem, gms_v83 @0x4F4BB7), which is what
	// fixes the unit as seconds.
	AddTime uint32 `json:"addTime,omitempty"`
	// MaxDays is info/maxDays in DAYS — the ceiling, anchored to now, past
	// which an extender may not push a target's expiration.
	MaxDays uint32 `json:"maxDays,omitempty"`
```

- [ ] **Step 4: Parse the two fields**

In `services/atlas-data/atlas.com/data/cash/reader.go`, immediately after the existing `m.ProtectTime = ...` line:

```go
			m.AddTime = uint32(i.GetIntegerWithDefault("addTime", 0))
			m.MaxDays = uint32(i.GetIntegerWithDefault("maxDays", 0))
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
cd services/atlas-data/atlas.com/data && go test -race ./cash/... && go vet ./...
```

Expected: PASS, vet clean.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-data/atlas.com/data/cash/
git commit -m "feat(task-222): parse cash addTime/maxDays in atlas-data"
```

---

### Task 2: atlas-data parses equipment `notExtend`

The client reads `info/notExtend` off the **target equip** and refuses the extension when it is set (`CItemInfo::IsNotExtendItem`, reached from `CDraggableItem::ModifyEquipItem`; the property name comes from StringPool entry `SP_5109_NOTEXTEND`, gms_v83 `sub_5D586A` @`0x5D586A`). Nothing in the tree parses it today.

**Files:**
- Modify: `services/atlas-data/atlas.com/data/equipment/reader.go:112-114`
- Modify: `services/atlas-data/atlas.com/data/equipment/rest.go:43-44`
- Test: `services/atlas-data/atlas.com/data/equipment/reader_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces: `equipment.RestModel.NotExtend bool` (json `notExtend`). Task 13 mirrors it on the channel model.

- [ ] **Step 1: Write the failing test**

Append to `services/atlas-data/atlas.com/data/equipment/reader_test.go`:

```go
const testNotExtendXML = `
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="01402046.img">
  <imgdir name="info">
    <int name="reqLevel" value="30"/>
    <int name="notExtend" value="1"/>
  </imgdir>
</imgdir>
`

const testNotExtendFalseXML = `
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="01402047.img">
  <imgdir name="info">
    <int name="reqLevel" value="30"/>
    <int name="notExtend" value="0"/>
  </imgdir>
</imgdir>
`

const testNotExtendAbsentXML = `
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="01402048.img">
  <imgdir name="info">
    <int name="reqLevel" value="30"/>
  </imgdir>
</imgdir>
`

func TestReaderNotExtend(t *testing.T) {
	l, _ := test.NewNullLogger()
	cases := []struct {
		name string
		xml  string
		want bool
	}{
		{"present true", testNotExtendXML, true},
		{"present false", testNotExtendFalseXML, false},
		{"absent defaults false", testNotExtendAbsentXML, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rm, err := Read(l)(xml.FromByteArrayProvider([]byte(c.xml)))()
			if err != nil {
				t.Fatal(err)
			}
			if rm.NotExtend != c.want {
				t.Errorf("NotExtend = %v, want %v", rm.NotExtend, c.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-data/atlas.com/data && go test ./equipment/ -run TestReaderNotExtend -v
```

Expected: FAIL — compile error, `rm.NotExtend undefined`.

- [ ] **Step 3: Add the REST field**

In `services/atlas-data/atlas.com/data/equipment/rest.go`, in `type RestModel struct`, immediately after the `TradeBlock bool \`json:"tradeBlock"\`` line:

```go
	// NotExtend is info/notExtend — when set, an item-expiration extender
	// (Magical Sandglass) may not be applied to this equip. The client
	// enforces it via CItemInfo::IsNotExtendItem; the server re-checks so a
	// crafted request cannot bypass it.
	NotExtend      bool            `json:"notExtend"`
```

- [ ] **Step 4: Parse the field**

In `services/atlas-data/atlas.com/data/equipment/reader.go`, inside the struct literal, immediately after the `TradeBlock: info.GetBool("tradeBlock", false),` line:

```go
			NotExtend:      info.GetBool("notExtend", false),
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
cd services/atlas-data/atlas.com/data && go test -race ./equipment/... && go vet ./...
```

Expected: PASS, vet clean.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-data/atlas.com/data/equipment/
git commit -m "feat(task-222): parse equipment notExtend in atlas-data"
```

---

### Task 3: `ClassificationExpirationExtender` in atlas-constants

`character_cash_item_use.go:1034` currently reads `if category == 550`, a bare literal. Naming the classification satisfies DOM-21 and gives Task 14 a symbol to gate on.

**Files:**
- Modify: `libs/atlas-constants/item/constants.go:108` (between `ClassificationRemoteStore` = 547 and `ClassificationViciousHammer` = 557)
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:1034`
- Test: `libs/atlas-constants/item/constants_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces: `item.ClassificationExpirationExtender` — an `item.Classification` whose value is 550. Task 14 compares `item.GetClassification(itemId)` against it.

- [ ] **Step 1: Write the failing test**

Append to `libs/atlas-constants/item/constants_test.go` (create the file with `package item` and `import "testing"` if it does not exist):

```go
func TestClassificationExpirationExtender(t *testing.T) {
	if ClassificationExpirationExtender != Classification(550) {
		t.Fatalf("ClassificationExpirationExtender = %d, want 550", ClassificationExpirationExtender)
	}
	if got := GetClassification(Id(5500001)); got != ClassificationExpirationExtender {
		t.Fatalf("GetClassification(5500001) = %d, want %d", got, ClassificationExpirationExtender)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd libs/atlas-constants && go test ./item/ -run TestClassificationExpirationExtender -v
```

Expected: FAIL — `undefined: ClassificationExpirationExtender`.

- [ ] **Step 3: Define the constant**

In `libs/atlas-constants/item/constants.go`, between the `ClassificationRemoteStore` and `ClassificationViciousHammer` lines (the block is in ascending numeric order):

```go
	ClassificationExpirationExtender       = Classification(550)
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd libs/atlas-constants && go test -race ./item/ -run TestClassificationExpirationExtender -v
```

Expected: PASS.

- [ ] **Step 5: Replace the bare literal in the classifier**

In `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go`, change line 1034 from `if category == 550 {` to:

```go
		if category == item.ClassificationExpirationExtender {
```

Leave the branch body unchanged — it already returns 62 on GMS >= 95 and 61 otherwise, which is the IDA-verified mapping (design §1.1).

- [ ] **Step 6: Run tests and vet**

```bash
cd libs/atlas-constants && go test -race ./... && go vet ./...
cd ../../services/atlas-channel/atlas.com/channel && go build ./... && go test -race ./socket/handler/ && go vet ./...
```

Expected: all PASS.

- [ ] **Step 7: Commit**

```bash
git add libs/atlas-constants/item/ services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go
git commit -m "feat(task-222): name item classification 550 ExpirationExtender"
```

---

### Task 4: Rename the shared sub-body codec to `ItemUseTargetSlot`

Design §1.2: the sandglass sub-body is one `Encode2` of the target equip position — the client literally shares one jump-table arm between case 25 (Item Tag) and case 61/62 (sandglass), IDA-labelled *"jumptable 00A0A6E6 cases 25,61"* at gms_v83 `0xA0CAE0`; the v95 PDB types the argument as `unsigned __int16 nEPOS`. It is byte-identical to the existing `ItemUseItemTag`, so the type is reused. Signed-off decision D3 renames it to describe the layout rather than one of its two uses.

The rename is self-contained: the only non-test reference is `character_cash_item_use.go:203`, and no socket-config template or packet registry names the type (verified by grep over `services/atlas-configurations/seed-data/` and `docs/packets/`).

**Files:**
- Rename: `libs/atlas-packet/cash/serverbound/item_use_item_tag.go` → `libs/atlas-packet/cash/serverbound/item_use_target_slot.go`
- Rename: `libs/atlas-packet/cash/serverbound/item_use_item_tag_test.go` → `libs/atlas-packet/cash/serverbound/item_use_target_slot_test.go`
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:203`

**Interfaces:**
- Consumes: nothing.
- Produces: `serverbound.ItemUseTargetSlot` with constructor `NewItemUseTargetSlot(updateTimeFirst bool) *ItemUseTargetSlot`, getters `Slot() int16` / `UpdateTime() uint32` / `Operation() string`, and `Encode`/`Decode` with the existing signatures. Task 14 calls `cashsb.NewItemUseTargetSlot(updateTimeFirst)` and reads `sp.Slot()`.

- [ ] **Step 1: Rename both files with git**

```bash
git mv libs/atlas-packet/cash/serverbound/item_use_item_tag.go libs/atlas-packet/cash/serverbound/item_use_target_slot.go
git mv libs/atlas-packet/cash/serverbound/item_use_item_tag_test.go libs/atlas-packet/cash/serverbound/item_use_target_slot_test.go
```

- [ ] **Step 2: Rewrite the codec file**

Replace the entire contents of `libs/atlas-packet/cash/serverbound/item_use_target_slot.go` with:

```go
package serverbound

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

// ItemUseTargetSlot is the bare-target-slot sub-body of the cash ItemUse
// packet: one int16 equip position (negative when the target is equipped),
// followed by the trailing updateTime on the two builds that carry it there.
//
// The client shares ONE dispatch arm between the Item Tag case (25) and the
// item-expiration-extender / Magical Sandglass case (61 on GMS < 95, 62 on
// GMS >= 95) — gms_v83 CWvsContext::SendConsumeCashItemUseRequest jump-table
// target @0xA0CAE0, which IDA labels "jumptable 00A0A6E6 cases 25,61" and
// which performs exactly one COutPacket::Encode2. The gms_v95 PDB types the
// encoded argument as `unsigned __int16 nEPOS`. The type is therefore named
// for its layout, not for either caller.
type ItemUseTargetSlot struct {
	slot            int16
	updateTime      uint32
	updateTimeFirst bool
}

func NewItemUseTargetSlot(updateTimeFirst bool) *ItemUseTargetSlot {
	return &ItemUseTargetSlot{updateTimeFirst: updateTimeFirst}
}

func (m ItemUseTargetSlot) Slot() int16        { return m.slot }
func (m ItemUseTargetSlot) UpdateTime() uint32 { return m.updateTime }
func (m ItemUseTargetSlot) Operation() string  { return "ItemUseTargetSlot" }

func (m ItemUseTargetSlot) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt16(m.slot)
		if !m.updateTimeFirst {
			w.WriteInt(m.updateTime)
		}
		return w.Bytes()
	}
}

func (m *ItemUseTargetSlot) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.slot = r.ReadInt16()
		if !m.updateTimeFirst {
			m.updateTime = r.ReadUint32()
		}
	}
}
```

- [ ] **Step 3: Rewrite the test file, adding the equipped-slot case**

Replace the entire contents of `libs/atlas-packet/cash/serverbound/item_use_target_slot_test.go` with:

```go
package serverbound

import (
	"bytes"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestItemUseTargetSlotRoundTrip(t *testing.T) {
	// Negative slots are the equipped positions, which is the ONLY shape the
	// sandglass arm ever sees: the client resolves the drop point with
	// CDraggableItem::ModifyEquipItem and negates it.
	for _, s := range []int16{-1, -8, 3} {
		for _, first := range []bool{true, false} {
			for _, v := range pt.Variants {
				t.Run(v.Name, func(t *testing.T) {
					ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
					input := ItemUseTargetSlot{slot: s, updateTime: 1000, updateTimeFirst: first}
					output := *NewItemUseTargetSlot(first)
					pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
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
}

// v83 golden bytes: short slot (-1 = FF FF) + trailing int updateTime (1000 = E8 03 00 00)
func TestItemUseTargetSlotV83Bytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	m := ItemUseTargetSlot{slot: -1, updateTime: 1000, updateTimeFirst: false}
	got := m.Encode(l, pt.CreateContext("GMS", 83, 1))(nil)
	want := []byte{0xFF, 0xFF, 0xE8, 0x03, 0x00, 0x00}
	if !bytes.Equal(got, want) {
		t.Fatalf("got % X, want % X", got, want)
	}
}

// An equipped weapon position (-11) encodes as F5 FF little-endian.
func TestItemUseTargetSlotEquippedSlotBytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	m := ItemUseTargetSlot{slot: -11, updateTimeFirst: true}
	got := m.Encode(l, pt.CreateContext("GMS", 83, 1))(nil)
	want := []byte{0xF5, 0xFF}
	if !bytes.Equal(got, want) {
		t.Fatalf("got % X, want % X", got, want)
	}
}
```

- [ ] **Step 4: Update the one non-test call site**

In `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:203`, change:

```go
			sp := cashsb.NewItemUseItemTag(updateTimeFirst)
```

to:

```go
			sp := cashsb.NewItemUseTargetSlot(updateTimeFirst)
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
cd libs/atlas-packet && go test -race ./cash/... && go vet ./...
cd ../../services/atlas-channel/atlas.com/channel && go build ./... && go vet ./...
```

Then from the worktree root, confirm no stale reference survives:

```bash
grep -rn "ItemUseItemTag" --include="*.go" . && echo "STALE REFERENCE" || echo "clean"
```

Expected: tests PASS, build clean, the grep prints `clean`.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-packet/cash/serverbound/ services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go
git commit -m "refactor(task-222): rename ItemUseItemTag to ItemUseTargetSlot"
```

---

### Task 5: `libs/atlas-saga` — new saga type, action, and payload

**Files:**
- Modify: `libs/atlas-saga/model.go:41` (Type block) and `libs/atlas-saga/model.go:220` (Action block)
- Modify: `libs/atlas-saga/payloads.go:1102` (after `ApplyAssetLockPayload`)
- Modify: `libs/atlas-saga/unmarshal.go:587` (after the `ApplyAssetLock` case)
- Test: `libs/atlas-saga/unmarshal_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces:
  - `saga.ExpirationExtenderUse Type = "expiration_extender_use"`
  - `saga.ExtendAssetExpiration Action = "extend_asset_expiration"`
  - `saga.ExtendAssetExpirationPayload{CharacterId uint32; InventoryType byte; Slot int16; Expiration time.Time; ExtenderTemplateId uint32}` with json tags `characterId`, `inventoryType`, `slot`, `expiration`, `extenderTemplateId`.

  Tasks 11, 12, and 14 all reference these exact names.

- [ ] **Step 1: Write the failing test**

Append to `libs/atlas-saga/unmarshal_test.go` (match the surrounding style of `TestUnmarshalApplyAssetLockStep` at `:799`):

```go
func TestUnmarshalExtendAssetExpirationStep(t *testing.T) {
	data := []byte(`{"stepId":"extend_asset_expiration","status":"pending","action":"extend_asset_expiration","payload":{"characterId":12345,"inventoryType":1,"slot":-11,"expiration":"2026-09-12T00:00:00Z","extenderTemplateId":5500001}}`)
	var s Step[any]
	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	p, ok := s.Payload.(ExtendAssetExpirationPayload)
	if !ok {
		t.Fatalf("payload type = %T, want ExtendAssetExpirationPayload", s.Payload)
	}
	if p.CharacterId != 12345 {
		t.Errorf("CharacterId = %d, want 12345", p.CharacterId)
	}
	if p.InventoryType != 1 {
		t.Errorf("InventoryType = %d, want 1", p.InventoryType)
	}
	if p.Slot != -11 {
		t.Errorf("Slot = %d, want -11", p.Slot)
	}
	if p.ExtenderTemplateId != 5500001 {
		t.Errorf("ExtenderTemplateId = %d, want 5500001", p.ExtenderTemplateId)
	}
	want := time.Date(2026, 9, 12, 0, 0, 0, 0, time.UTC)
	if !p.Expiration.Equal(want) {
		t.Errorf("Expiration = %v, want %v", p.Expiration, want)
	}
}

func TestExpirationExtenderUseSagaTypeValue(t *testing.T) {
	if ExpirationExtenderUse != Type("expiration_extender_use") {
		t.Fatalf("ExpirationExtenderUse = %q, want %q", ExpirationExtenderUse, "expiration_extender_use")
	}
}
```

If `unmarshal_test.go` does not already import `time`, add it to the import block.

- [ ] **Step 2: Run test to verify it fails**

```bash
cd libs/atlas-saga && go test ./... -run "TestUnmarshalExtendAssetExpirationStep|TestExpirationExtenderUseSagaTypeValue" -v
```

Expected: FAIL — `undefined: ExtendAssetExpirationPayload`, `undefined: ExpirationExtenderUse`.

- [ ] **Step 3: Add the saga type**

In `libs/atlas-saga/model.go`, in the `Type` const block, immediately after the `IncubatorUse` line:

```go
	ExpirationExtenderUse Type = "expiration_extender_use"
```

- [ ] **Step 4: Add the action**

In `libs/atlas-saga/model.go`, in the `Action` const block, in the "Item tag / sealing lock / incubator actions" group, immediately after the `ApplyAssetLock` line:

```go
	// ExtendAssetExpiration pushes a time-limited asset's expiration out. It
	// is deliberately NOT ApplyAssetLock: that action stamps FlagLock and
	// rejects an unlocked asset carrying a non-zero expiration, which is
	// exactly this action's only valid target.
	ExtendAssetExpiration Action = "extend_asset_expiration"
```

- [ ] **Step 5: Add the payload**

In `libs/atlas-saga/payloads.go`, immediately after the `ApplyAssetLockPayload` struct:

```go
// ExtendAssetExpirationPayload represents the payload required to extend the expiration of a time-limited asset in a specific inventory slot.
//
// Expiration is ABSOLUTE, never a duration: Kafka delivery here is
// at-least-once, and a duration-shaped payload would stack a second extension
// on redelivery. ExtenderTemplateId names the item-expiration extender being
// consumed so atlas-inventory can independently re-derive the maxDays cap
// rather than trusting the channel-computed timestamp.
type ExtendAssetExpirationPayload struct {
	CharacterId        uint32    `json:"characterId"`        // CharacterId associated with the action
	InventoryType      byte      `json:"inventoryType"`      // Type of inventory (1=equip, 2=use, 3=setup, 4=etc, 5=cash)
	Slot               int16     `json:"slot"`               // Slot of the asset to extend (negative for equipped slots, positive for inventory slots)
	Expiration         time.Time `json:"expiration"`         // Absolute expiration to set on the asset
	ExtenderTemplateId uint32    `json:"extenderTemplateId"` // Template id of the extender being consumed, for server-side cap re-derivation
}
```

- [ ] **Step 6: Add the unmarshal arm**

In `libs/atlas-saga/unmarshal.go`, immediately after the `case ApplyAssetLock:` block:

```go
	case ExtendAssetExpiration:
		var payload ExtendAssetExpirationPayload
		if err := json.Unmarshal(aux.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.Action, err)
		}
		s.Payload = any(payload).(T)
```

- [ ] **Step 7: Run tests to verify they pass**

```bash
cd libs/atlas-saga && go test -race ./... && go vet ./...
```

Expected: PASS, vet clean.

- [ ] **Step 8: Commit**

```bash
git add libs/atlas-saga/
git commit -m "feat(task-222): add ExpirationExtenderUse saga type and extend-expiration action"
```

---

### Task 6: atlas-inventory — `EXTEND_EXPIRATION` contract and cash data client

Two independent additions that the next two tasks both need: the Kafka command contract, and a client for `GET /data/cash/items/{id}` so the compartment processor can re-derive `maxDays` (signed-off decision D2).

**Files:**
- Modify: `services/atlas-inventory/atlas.com/inventory/kafka/message/compartment/kafka.go:35` and `:190`
- Create: `services/atlas-inventory/atlas.com/inventory/data/cash/requests.go`
- Create: `services/atlas-inventory/atlas.com/inventory/data/cash/rest.go`
- Create: `services/atlas-inventory/atlas.com/inventory/data/cash/model.go`
- Create: `services/atlas-inventory/atlas.com/inventory/data/cash/processor.go`
- Create: `services/atlas-inventory/atlas.com/inventory/data/cash/mock/processor.go`
- Test: `services/atlas-inventory/atlas.com/inventory/data/cash/processor_test.go`

**Interfaces:**
- Consumes: `cash.RestModel.AddTime` / `.MaxDays` json field names from Task 1.
- Produces:
  - `compartment.CommandExtendExpiration = "EXTEND_EXPIRATION"`
  - `compartment.ExtendExpirationCommandBody{Slot int16; Expiration time.Time; ExtenderTemplateId uint32}` with json tags `slot`, `expiration`, `extenderTemplateId`
  - `cash.Processor` interface with `GetById(itemId uint32) (Model, error)`; `cash.Model` with `Id() uint32`, `AddTime() uint32`, `MaxDays() uint32`
  - `cash.NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor`
  - `cash.NewModelBuilder(id uint32)` with `SetAddTime` / `SetMaxDays` / `Build`
  - `mock.ProcessorMock` with field `GetByIdFunc func(itemId uint32) (cash.Model, error)`

- [ ] **Step 1: Write the failing test**

Create `services/atlas-inventory/atlas.com/inventory/data/cash/processor_test.go`, modelled on `data/consumable/processor_drain_test.go`:

```go
package cash

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func TestGetByIdReadsAddTimeAndMaxDays(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"type":"cash_items","id":"5500001","attributes":{"addTime":604800,"maxDays":30}}}`))
	}))
	defer srv.Close()
	t.Setenv("DATA_SERVICE_URL", srv.URL+"/")

	l, _ := test.NewNullLogger()
	te, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), te)

	m, err := NewProcessor(l, ctx).GetById(5500001)
	if err != nil {
		t.Fatal(err)
	}
	if m.AddTime() != 604800 {
		t.Errorf("AddTime = %d, want 604800", m.AddTime())
	}
	if m.MaxDays() != 30 {
		t.Errorf("MaxDays = %d, want 30", m.MaxDays())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-inventory/atlas.com/inventory && go test ./data/cash/ -v
```

Expected: FAIL — the package does not exist.

- [ ] **Step 3: Create the REST model**

`services/atlas-inventory/atlas.com/inventory/data/cash/rest.go`:

```go
package cash

import (
	"strconv"
)

// RestModel is the subset of atlas-data's cash resource this service needs:
// the item-expiration-extender grant and ceiling, used to re-derive the cap
// server-side rather than trusting the channel-computed expiration.
type RestModel struct {
	Id      uint32 `json:"-"`
	AddTime uint32 `json:"addTime,omitempty"`
	MaxDays uint32 `json:"maxDays,omitempty"`
}

func (r RestModel) GetName() string {
	return "cash_items"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(idStr string) error {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

// SetToOneReferenceID / SetToManyReferenceIDs are required by api2go's
// unmarshal whenever the upstream response carries a relationships block.
func (r *RestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{id: rm.Id, addTime: rm.AddTime, maxDays: rm.MaxDays}, nil
}
```

- [ ] **Step 4: Create the domain model and its builder**

`services/atlas-inventory/atlas.com/inventory/data/cash/model.go`:

```go
package cash

// Model is a cash item template's expiration-extender attributes, as resolved
// from atlas-data.
type Model struct {
	id      uint32
	addTime uint32
	maxDays uint32
}

func (m Model) Id() uint32 { return m.id }

// AddTime is the expiration grant in SECONDS.
func (m Model) AddTime() uint32 { return m.addTime }

// MaxDays is the ceiling in DAYS, anchored to now.
func (m Model) MaxDays() uint32 { return m.maxDays }

// ModelBuilder constructs a Model. It exists so callers outside this package
// (notably test doubles) can build one without exported fields, per the
// project's Builder convention.
type ModelBuilder struct {
	id      uint32
	addTime uint32
	maxDays uint32
}

func NewModelBuilder(id uint32) *ModelBuilder {
	return &ModelBuilder{id: id}
}

func (b *ModelBuilder) SetAddTime(v uint32) *ModelBuilder { b.addTime = v; return b }
func (b *ModelBuilder) SetMaxDays(v uint32) *ModelBuilder { b.maxDays = v; return b }

func (b *ModelBuilder) Build() Model {
	return Model{id: b.id, addTime: b.addTime, maxDays: b.maxDays}
}
```

- [ ] **Step 5: Create the request and processor**

`services/atlas-inventory/atlas.com/inventory/data/cash/requests.go`:

```go
package cash

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	cashItemResource = "data/cash/items/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

func requestById(itemId uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+cashItemResource, itemId))
}
```

`services/atlas-inventory/atlas.com/inventory/data/cash/processor.go`:

```go
package cash

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

type Processor interface {
	GetById(itemId uint32) (Model, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) GetById(itemId uint32) (Model, error) {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(itemId), Extract)()
}
```

- [ ] **Step 6: Create the mock**

`services/atlas-inventory/atlas.com/inventory/data/cash/mock/processor.go`:

```go
package mock

import (
	"atlas-inventory/data/cash"
)

// ProcessorMock is a mock implementation of cash.Processor.
type ProcessorMock struct {
	GetByIdFunc func(itemId uint32) (cash.Model, error)
}

var _ cash.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) GetById(itemId uint32) (cash.Model, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(itemId)
	}
	return cash.Model{}, nil
}
```

- [ ] **Step 7: Add the Kafka command contract**

In `services/atlas-inventory/atlas.com/inventory/kafka/message/compartment/kafka.go`, in the command-type const block immediately after `CommandApplyLock`:

```go
	CommandExtendExpiration  = "EXTEND_EXPIRATION"
```

and immediately after the `ApplyLockCommandBody` struct:

```go
// ExtendExpirationCommandBody extends a time-limited asset's expiration
// WITHOUT touching its flags. Expiration is absolute, not a duration, so a
// redelivered command is a no-op rather than a second extension.
// ExtenderTemplateId names the consumed item-expiration extender so this
// service can re-derive the maxDays cap itself — the channel is not a trust
// boundary.
type ExtendExpirationCommandBody struct {
	Slot               int16     `json:"slot"`
	Expiration         time.Time `json:"expiration"`
	ExtenderTemplateId uint32    `json:"extenderTemplateId"`
}
```

- [ ] **Step 8: Run tests to verify they pass**

```bash
cd services/atlas-inventory/atlas.com/inventory && go test -race ./data/cash/... ./kafka/... && go build ./... && go vet ./...
```

Expected: PASS, build and vet clean.

- [ ] **Step 9: Commit**

```bash
git add services/atlas-inventory/atlas.com/inventory/data/cash/ services/atlas-inventory/atlas.com/inventory/kafka/message/compartment/kafka.go
git commit -m "feat(task-222): add EXTEND_EXPIRATION contract and cash data client to atlas-inventory"
```

---

### Task 7: atlas-inventory asset-level `ExtendExpiration`

The flag-preservation invariant lives here. `ApplyLock` (`asset/processor.go:329`) unconditionally adds `FlagLock` and rejects an unlocked asset with a non-zero expiration — the exact shape this feature targets — so this is a sibling method, not a reuse.

**Files:**
- Modify: `services/atlas-inventory/atlas.com/inventory/asset/processor.go` (interface at `:20-45`, implementation after `ClearLock`)
- Modify: `services/atlas-inventory/atlas.com/inventory/asset/mock/processor.go`
- Test: `services/atlas-inventory/atlas.com/inventory/asset/processor_test.go`

**Interfaces:**
- Consumes: `asset.Clone(m).SetExpiration(t).Build()` (`asset/builder.go:11,106`), `updateFlagAndExpiration(db, id, flag, expiration)` (`asset/administrator.go:65`), `UpdatedEventStatusProvider(transactionId, characterId, m)`.
- Produces: `asset.Processor.ExtendExpiration(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32) func(a Model, expiration time.Time) error` — the same curried shape as `ApplyLock`. Task 8 calls it.

**Semantics:**
- reject when `a.Locked()` — a lock window is not an item time limit
- reject when `a.Expiration().IsZero()` — a permanent item has nothing to extend
- reject when `expiration.Before(a.Expiration())` — never walk an expiration backwards
- when `expiration.Equal(a.Expiration())` — redelivery: skip the DB write but STILL emit `UPDATED`, so the saga step completes instead of timing out
- otherwise write the expiration through `updateFlagAndExpiration` with `a.Flag()` passed through **unchanged**, and emit `UPDATED`

- [ ] **Step 1: Write the failing test**

Append to `services/atlas-inventory/atlas.com/inventory/asset/processor_test.go`. Read that file and `change_template_test.go` in the same package first and reuse whatever harness they already use for a `*gorm.DB`, a tenant context, and a seeded asset — do not introduce a parallel one. The names `testDatabase` / `testContext` / `seedAsset` below are placeholders for whatever those helpers are actually called; substitute them.

```go
func TestExtendExpirationPreservesFlags(t *testing.T) {
	db := testDatabase(t)
	l, _ := test.NewNullLogger()
	ctx := testContext(t)

	base := time.Now().UTC().Add(120 * time.Hour).Truncate(time.Second)
	// An unlocked, time-limited equip carrying an unrelated flag bit.
	a := seedAsset(t, db, func(b *ModelBuilder) {
		// AddFlag, not SetFlag: SetFlag takes a raw uint16 while the
		// constants are typed af.Flag (builder.go:111 vs :174).
		b.SetExpiration(base).AddFlag(af.FlagUntradeable)
	})

	mb := message.NewBuffer()
	want := base.Add(168 * time.Hour)
	err := NewProcessor(l, ctx, db).ExtendExpiration(mb)(uuid.New(), 12345)(a, want)
	if err != nil {
		t.Fatalf("ExtendExpiration: %v", err)
	}

	got, err := NewProcessor(l, ctx, db).GetById(a.Id())
	if err != nil {
		t.Fatal(err)
	}
	if !got.Expiration().Equal(want) {
		t.Errorf("Expiration = %v, want %v", got.Expiration(), want)
	}
	if got.Flag() != a.Flag() {
		t.Errorf("Flag = %d, want %d (unchanged)", got.Flag(), a.Flag())
	}
	if got.Locked() {
		t.Error("FlagLock was set; ExtendExpiration must never touch flags")
	}
}

func TestExtendExpirationRejectsLockedAndPermanent(t *testing.T) {
	db := testDatabase(t)
	l, _ := test.NewNullLogger()
	ctx := testContext(t)
	p := NewProcessor(l, ctx, db)
	future := time.Now().UTC().Add(240 * time.Hour)

	locked := seedAsset(t, db, func(b *ModelBuilder) {
		b.SetExpiration(time.Now().UTC().Add(48 * time.Hour)).AddFlag(af.FlagLock)
	})
	if err := p.ExtendExpiration(message.NewBuffer())(uuid.New(), 12345)(locked, future); err == nil {
		t.Error("expected rejection for a locked asset")
	}

	permanent := seedAsset(t, db, func(b *ModelBuilder) {})
	if err := p.ExtendExpiration(message.NewBuffer())(uuid.New(), 12345)(permanent, future); err == nil {
		t.Error("expected rejection for a permanent asset")
	}
}

func TestExtendExpirationRedeliveryIsIdempotent(t *testing.T) {
	db := testDatabase(t)
	l, _ := test.NewNullLogger()
	ctx := testContext(t)
	p := NewProcessor(l, ctx, db)

	base := time.Now().UTC().Add(120 * time.Hour).Truncate(time.Second)
	a := seedAsset(t, db, func(b *ModelBuilder) { b.SetExpiration(base) })
	want := base.Add(168 * time.Hour)

	if err := p.ExtendExpiration(message.NewBuffer())(uuid.New(), 12345)(a, want); err != nil {
		t.Fatal(err)
	}
	extended, err := p.GetById(a.Id())
	if err != nil {
		t.Fatal(err)
	}
	// Replay the same absolute value against the already-extended asset.
	mb := message.NewBuffer()
	if err := p.ExtendExpiration(mb)(uuid.New(), 12345)(extended, want); err != nil {
		t.Fatalf("redelivery must succeed, got %v", err)
	}
	again, err := p.GetById(a.Id())
	if err != nil {
		t.Fatal(err)
	}
	if !again.Expiration().Equal(want) {
		t.Errorf("Expiration = %v, want %v (redelivery must not stack)", again.Expiration(), want)
	}
	if len(mb.GetAll()) == 0 {
		t.Error("redelivery must still emit UPDATED so the saga step completes")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-inventory/atlas.com/inventory && go test ./asset/ -run TestExtendExpiration -v
```

Expected: FAIL — `p.ExtendExpiration undefined`.

- [ ] **Step 3: Add the method to the interface**

In `services/atlas-inventory/atlas.com/inventory/asset/processor.go`, in `type Processor interface`, immediately after the `ApplyLock` line:

```go
	ExtendExpiration(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32) func(a Model, expiration time.Time) error
```

- [ ] **Step 4: Implement it**

In the same file, immediately after the `ClearLock` implementation:

```go
// ExtendExpiration sets the expiration on a genuinely time-limited asset in
// place WITHOUT touching its flags, emitting the existing UPDATED status
// event. It is the deliberate mirror of ApplyLock, never a reuse of it:
// ApplyLock unconditionally adds FlagLock and rejects an unlocked asset
// carrying a non-zero expiration, which is exactly the shape this method
// exists to mutate.
//
// The expiration is absolute. A redelivered command carrying the value the
// asset already holds skips the write but still emits UPDATED, so the saga
// step completes rather than timing out — that, plus set-to-absolute, is what
// keeps at-least-once delivery from stacking a second extension.
func (p *ProcessorImpl) ExtendExpiration(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32) func(a Model, expiration time.Time) error {
	return func(transactionId uuid.UUID, characterId uint32) func(a Model, expiration time.Time) error {
		return func(a Model, expiration time.Time) error {
			if a.Locked() {
				return errors.New("asset expiration is a lock window, not a time limit")
			}
			if a.Expiration().IsZero() {
				return errors.New("asset is permanent and has no expiration to extend")
			}
			if expiration.Before(a.Expiration()) {
				return errors.New("expiration must not move backwards")
			}
			if expiration.Equal(a.Expiration()) {
				return mb.Put(asset.EnvEventTopicStatus, UpdatedEventStatusProvider(transactionId, characterId, a))
			}
			updated := Clone(a).SetExpiration(expiration).Build()
			if err := updateFlagAndExpiration(p.db.WithContext(p.ctx), a.Id(), a.Flag(), expiration); err != nil {
				return err
			}
			return mb.Put(asset.EnvEventTopicStatus, UpdatedEventStatusProvider(transactionId, characterId, updated))
		}
	}
}
```

- [ ] **Step 5: Add the mock method**

In `services/atlas-inventory/atlas.com/inventory/asset/mock/processor.go`, add a field beside the existing `ApplyLockFunc`:

```go
	ExtendExpirationFunc func(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32) func(a asset.Model, expiration time.Time) error
```

and the method, matching the file's existing mock style and its import alias for the asset package:

```go
// ExtendExpiration is a mock implementation of the asset.Processor.ExtendExpiration method
func (m *ProcessorMock) ExtendExpiration(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32) func(a asset.Model, expiration time.Time) error {
	if m.ExtendExpirationFunc != nil {
		return m.ExtendExpirationFunc(mb)
	}
	return func(_ uuid.UUID, _ uint32) func(asset.Model, time.Time) error {
		return func(asset.Model, time.Time) error { return nil }
	}
}
```

- [ ] **Step 6: Run tests to verify they pass**

```bash
cd services/atlas-inventory/atlas.com/inventory && go test -race ./asset/... && go build ./... && go vet ./...
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-inventory/atlas.com/inventory/asset/
git commit -m "feat(task-222): add flag-preserving asset ExtendExpiration"
```

---

### Task 8: atlas-inventory compartment `ExtendAssetExpiration` with server-side cap re-validation

This is the trust boundary (PRD FR-4.3, design D2). The channel computes the expiration; this service re-derives `cap = now + maxDays*24h` from the extender's own cash data and clamps the incoming value to it. A forged command carrying an expiration years out is bounded here, not trusted.

**Files:**
- Modify: `services/atlas-inventory/atlas.com/inventory/compartment/processor.go` (struct and `NewProcessor` near the top, the `Processor` interface at `:37-60`, and the implementation after `ApplyAssetLock` at `:1052-1076`)
- Modify: `services/atlas-inventory/atlas.com/inventory/compartment/mock/processor.go`
- Test: `services/atlas-inventory/atlas.com/inventory/compartment/processor_test.go`

**Interfaces:**
- Consumes: `asset.Processor.ExtendExpiration` (Task 7), `cash.Processor.GetById` / `cash.Model.MaxDays()` / `cash.NewModelBuilder` (Task 6).
- Produces:
  - `compartment.Processor.ExtendAssetExpiration(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, slot int16, expiration time.Time, extenderTemplateId uint32) error`
  - `compartment.Processor.ExtendAssetExpirationAndEmit(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, slot int16, expiration time.Time, extenderTemplateId uint32) error`
  - `compartment.Processor.WithCashProcessor(cp cash.Processor) *ProcessorImpl` — the test seam, mirroring the existing `WithAssetProcessor` at `:142`.

  Task 9 calls `ExtendAssetExpirationAndEmit`.

- [ ] **Step 1: Write the failing test**

Append to `services/atlas-inventory/atlas.com/inventory/compartment/processor_test.go`, using the file's existing database/context harness. `testDatabase` / `testContext` / `seedEquipCompartmentWithAsset` are placeholders — substitute the file's real helpers, and if no compartment-with-asset seeder exists, add one alongside the existing seeding helpers in that file.

```go
func TestExtendAssetExpirationClampsToServerDerivedCap(t *testing.T) {
	db := testDatabase(t)
	l, _ := test.NewNullLogger()
	ctx := testContext(t)

	// A 5-day-remaining equip, and a forged command asking for 10 years.
	base := time.Now().UTC().Add(120 * time.Hour).Truncate(time.Second)
	c, a := seedEquipCompartmentWithAsset(t, db, base)

	cp := &cashmock.ProcessorMock{
		GetByIdFunc: func(itemId uint32) (cash.Model, error) {
			return cash.NewModelBuilder(itemId).SetAddTime(604800).SetMaxDays(30).Build(), nil
		},
	}
	forged := time.Now().UTC().Add(10 * 365 * 24 * time.Hour)

	err := NewProcessor(l, ctx, db).WithCashProcessor(cp).
		ExtendAssetExpirationAndEmit(uuid.New(), c.CharacterId(), inventory.TypeValueEquip, a.Slot(), forged, 5500001)
	if err != nil {
		t.Fatalf("ExtendAssetExpirationAndEmit: %v", err)
	}

	got, err := asset.NewProcessor(l, ctx, db).GetById(a.Id())
	if err != nil {
		t.Fatal(err)
	}
	ceiling := time.Now().UTC().Add(30 * 24 * time.Hour)
	if got.Expiration().After(ceiling.Add(time.Minute)) {
		t.Errorf("Expiration = %v, want clamped to about %v", got.Expiration(), ceiling)
	}
	if !got.Expiration().After(base) {
		t.Errorf("Expiration = %v, want later than the original %v", got.Expiration(), base)
	}
}

func TestExtendAssetExpirationHonorsInBoundsRequest(t *testing.T) {
	db := testDatabase(t)
	l, _ := test.NewNullLogger()
	ctx := testContext(t)

	base := time.Now().UTC().Add(120 * time.Hour).Truncate(time.Second)
	c, a := seedEquipCompartmentWithAsset(t, db, base)

	cp := &cashmock.ProcessorMock{
		GetByIdFunc: func(itemId uint32) (cash.Model, error) {
			return cash.NewModelBuilder(itemId).SetAddTime(604800).SetMaxDays(30).Build(), nil
		},
	}
	want := base.Add(168 * time.Hour) // +7d, well inside the 30d cap

	err := NewProcessor(l, ctx, db).WithCashProcessor(cp).
		ExtendAssetExpirationAndEmit(uuid.New(), c.CharacterId(), inventory.TypeValueEquip, a.Slot(), want, 5500001)
	if err != nil {
		t.Fatal(err)
	}
	got, err := asset.NewProcessor(l, ctx, db).GetById(a.Id())
	if err != nil {
		t.Fatal(err)
	}
	if !got.Expiration().Equal(want) {
		t.Errorf("Expiration = %v, want %v", got.Expiration(), want)
	}
}

func TestExtendAssetExpirationRejectsZeroMaxDays(t *testing.T) {
	db := testDatabase(t)
	l, _ := test.NewNullLogger()
	ctx := testContext(t)

	base := time.Now().UTC().Add(120 * time.Hour).Truncate(time.Second)
	c, a := seedEquipCompartmentWithAsset(t, db, base)

	cp := &cashmock.ProcessorMock{
		GetByIdFunc: func(itemId uint32) (cash.Model, error) {
			return cash.NewModelBuilder(itemId).SetAddTime(604800).SetMaxDays(0).Build(), nil
		},
	}
	err := NewProcessor(l, ctx, db).WithCashProcessor(cp).
		ExtendAssetExpirationAndEmit(uuid.New(), c.CharacterId(), inventory.TypeValueEquip, a.Slot(), base.Add(time.Hour), 5500001)
	if err == nil {
		t.Fatal("expected rejection when the extender has no maxDays ceiling")
	}
	got, gerr := asset.NewProcessor(l, ctx, db).GetById(a.Id())
	if gerr != nil {
		t.Fatal(gerr)
	}
	if !got.Expiration().Equal(base) {
		t.Errorf("Expiration = %v, want unchanged %v", got.Expiration(), base)
	}
}
```

Import `"atlas-inventory/data/cash"` and `cashmock "atlas-inventory/data/cash/mock"`.

- [ ] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-inventory/atlas.com/inventory && go test ./compartment/ -run TestExtendAssetExpiration -v
```

Expected: FAIL — `WithCashProcessor undefined`.

- [ ] **Step 3: Add the cash processor to the struct and constructor**

In `services/atlas-inventory/atlas.com/inventory/compartment/processor.go`, add the field to `type ProcessorImpl struct`:

```go
	cashProcessor      cash.Processor
```

initialise it in `NewProcessor`:

```go
		cashProcessor:      cash.NewProcessor(l, ctx),
```

and add `"atlas-inventory/data/cash"` to the import block.

- [ ] **Step 4: Add the interface entries and the seam**

In `type Processor interface`, immediately after the `WithAssetProcessor` line:

```go
	WithCashProcessor(cp cash.Processor) *ProcessorImpl
```

and beside the `ApplyAssetLock` / `ApplyAssetLockAndEmit` entries:

```go
	ExtendAssetExpirationAndEmit(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, slot int16, expiration time.Time, extenderTemplateId uint32) error
	ExtendAssetExpiration(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, slot int16, expiration time.Time, extenderTemplateId uint32) error
```

Implement the seam beside `WithAssetProcessor` at `:142` — read that method first and copy its exact shape (it returns a shallow copy with one field replaced, not a mutation of the receiver):

```go
func (p *ProcessorImpl) WithCashProcessor(cp cash.Processor) *ProcessorImpl {
	p2 := *p
	p2.cashProcessor = cp
	return &p2
}
```

- [ ] **Step 5: Implement the processor methods**

Immediately after `ApplyAssetLock` in the same file:

```go
func (p *ProcessorImpl) ExtendAssetExpirationAndEmit(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, slot int16, expiration time.Time, extenderTemplateId uint32) error {
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			return p.WithTransaction(tx).ExtendAssetExpiration(buf)(transactionId, characterId, inventoryType, slot, expiration, extenderTemplateId)
		})
	})
}

// ExtendAssetExpiration extends a time-limited asset's expiration, clamping
// the requested value to a cap this service re-derives itself.
//
// The channel computes the expiration, but the channel is NOT a trust
// boundary: a forged COMMAND_TOPIC_COMPARTMENT message could otherwise set an
// arbitrary expiration. The cap is re-derived here from the consumed
// extender's own cash data (maxDays), anchored to now — the same anchor the
// client uses (CDraggableItem::ModifyEquipItem compares against
// GetCorrectTime() + maxDays).
func (p *ProcessorImpl) ExtendAssetExpiration(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, slot int16, expiration time.Time, extenderTemplateId uint32) error {
	return func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, slot int16, expiration time.Time, extenderTemplateId uint32) error {
		p.l.Debugf("Character [%d] attempting to extend expiration of asset in inventory [%d] slot [%d] with extender [%d].", characterId, inventoryType, slot, extenderTemplateId)

		cd, err := p.cashProcessor.GetById(extenderTemplateId)
		if err != nil {
			p.l.WithError(err).Errorf("Character [%d] unable to resolve extender [%d] cash data; refusing to extend expiration.", characterId, extenderTemplateId)
			return err
		}
		if cd.MaxDays() == 0 {
			p.l.Errorf("Character [%d] extender [%d] has no maxDays ceiling; refusing to extend expiration.", characterId, extenderTemplateId)
			return errors.New("extender has no maxDays ceiling")
		}
		serverCap := time.Now().Add(time.Duration(cd.MaxDays()) * 24 * time.Hour)
		if expiration.After(serverCap) {
			p.l.Warnf("Character [%d] requested expiration [%s] beyond the server-derived cap [%s] for extender [%d]; clamping.", characterId, expiration, serverCap, extenderTemplateId)
			expiration = serverCap
		}

		invLock := LockRegistry().Get(characterId, inventoryType)
		invLock.Lock()
		defer invLock.Unlock()

		c, err := p.GetByCharacterAndType(characterId)(inventoryType)
		if err != nil {
			p.l.WithError(err).Errorf("Character [%d] unable to extend expiration of asset in inventory [%d] slot [%d].", characterId, inventoryType, slot)
			return err
		}
		a, err := p.assetProcessor.WithTransaction(p.db).GetBySlot(c.Id(), slot)
		if err != nil {
			p.l.WithError(err).Errorf("Character [%d] unable to extend expiration of asset in inventory [%d] slot [%d].", characterId, inventoryType, slot)
			return err
		}
		if err := p.assetProcessor.WithTransaction(p.db).ExtendExpiration(mb)(transactionId, characterId)(a, expiration); err != nil {
			p.l.WithError(err).Errorf("Character [%d] unable to extend expiration of asset in inventory [%d] slot [%d].", characterId, inventoryType, slot)
			return err
		}
		p.l.Debugf("Character [%d] extended expiration of asset [%d] in inventory [%d] slot [%d] to [%s].", characterId, a.Id(), inventoryType, slot, expiration)
		return nil
	}
}
```

Add `"errors"` to the import block if it is not already there.

- [ ] **Step 6: Add the mock methods**

In `services/atlas-inventory/atlas.com/inventory/compartment/mock/processor.go`, add three fields to the `ProcessorMock` struct — beside `WithAssetProcessorFunc` (`:20`) and `ApplyAssetLockFunc` (`:56`) respectively:

```go
	WithCashProcessorFunc             func(cp cash.Processor) *compartment.ProcessorImpl
```

```go
	ExtendAssetExpirationAndEmitFunc  func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, slot int16, expiration time.Time, extenderTemplateId uint32) error
	ExtendAssetExpirationFunc         func(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, slot int16, expiration time.Time, extenderTemplateId uint32) error
```

and the three methods, following the file's nil-check-then-zero-value shape:

```go
func (m *ProcessorMock) WithCashProcessor(cp cash.Processor) *compartment.ProcessorImpl {
	if m.WithCashProcessorFunc != nil {
		return m.WithCashProcessorFunc(cp)
	}
	return nil
}

func (m *ProcessorMock) ExtendAssetExpirationAndEmit(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, slot int16, expiration time.Time, extenderTemplateId uint32) error {
	if m.ExtendAssetExpirationAndEmitFunc != nil {
		return m.ExtendAssetExpirationAndEmitFunc(transactionId, characterId, inventoryType, slot, expiration, extenderTemplateId)
	}
	return nil
}

func (m *ProcessorMock) ExtendAssetExpiration(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, slot int16, expiration time.Time, extenderTemplateId uint32) error {
	if m.ExtendAssetExpirationFunc != nil {
		return m.ExtendAssetExpirationFunc(mb)
	}
	return func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, slot int16, expiration time.Time, extenderTemplateId uint32) error {
		return nil
	}
}
```

Add `"atlas-inventory/data/cash"` to the mock file's import block.

- [ ] **Step 7: Run tests to verify they pass**

```bash
cd services/atlas-inventory/atlas.com/inventory && go test -race ./compartment/... ./asset/... && go build ./... && go vet ./...
```

Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-inventory/atlas.com/inventory/compartment/
git commit -m "feat(task-222): add compartment ExtendAssetExpiration with server-side cap re-validation"
```

---

### Task 9: atlas-inventory consumes `EXTEND_EXPIRATION`

**Files:**
- Modify: `services/atlas-inventory/atlas.com/inventory/kafka/consumer/compartment/consumer.go:96` (registration) and after `handleApplyLockCommand` at `:384-404`
- Test: `services/atlas-inventory/atlas.com/inventory/kafka/consumer/compartment/consumer_test.go` (create if absent)

**Interfaces:**
- Consumes: `compartment2.CommandExtendExpiration`, `compartment2.ExtendExpirationCommandBody` (Task 6); `compartment.Processor.ExtendAssetExpirationAndEmit` (Task 8).
- Produces: nothing consumed by later tasks.

- [ ] **Step 1: Write the failing test**

Create or append to `services/atlas-inventory/atlas.com/inventory/kafka/consumer/compartment/consumer_test.go`. Substitute the package's real tenant-context helper for `testContext`; if none exists, build the context inline with `tenant.Create` + `tenant.WithContext` as the other consumer tests in this service do.

```go
func TestHandleExtendExpirationCommandIgnoresOtherTypes(t *testing.T) {
	l, _ := test.NewNullLogger()
	ctx := testContext(t)
	// A command of the wrong type must be a no-op: every handler on this
	// shared topic sees every message, so the type guard is what keeps an
	// APPLY_LOCK from being processed as an extension.
	c := compartment2.Command[compartment2.ExtendExpirationCommandBody]{
		TransactionId: uuid.New(),
		CharacterId:   12345,
		InventoryType: byte(inventory.TypeValueEquip),
		Type:          compartment2.CommandApplyLock,
		Body:          compartment2.ExtendExpirationCommandBody{Slot: -11},
	}
	// A nil *gorm.DB would panic if the handler proceeded past the guard.
	handleExtendExpirationCommand(nil)(l, ctx, c)
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-inventory/atlas.com/inventory && go test ./kafka/consumer/compartment/ -run TestHandleExtendExpirationCommand -v
```

Expected: FAIL — `undefined: handleExtendExpirationCommand`.

- [ ] **Step 3: Write the handler**

In `services/atlas-inventory/atlas.com/inventory/kafka/consumer/compartment/consumer.go`, immediately after `handleApplyLockCommand`:

```go
func handleExtendExpirationCommand(db *gorm.DB) message.Handler[compartment2.Command[compartment2.ExtendExpirationCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c compartment2.Command[compartment2.ExtendExpirationCommandBody]) {
		if c.Type != compartment2.CommandExtendExpiration {
			return
		}

		l.Debugf("Received EXTEND_EXPIRATION command for character [%d], slot [%d], extender [%d].",
			c.CharacterId, c.Body.Slot, c.Body.ExtenderTemplateId)

		err := compartment.NewProcessor(l, ctx, db).ExtendAssetExpirationAndEmit(
			c.TransactionId,
			c.CharacterId,
			inventory.Type(c.InventoryType),
			c.Body.Slot,
			c.Body.Expiration,
			c.Body.ExtenderTemplateId,
		)
		if err != nil {
			l.WithError(err).Errorf("Failed to extend expiration of asset in slot [%d] for character [%d].", c.Body.Slot, c.CharacterId)
		}
	}
}
```

- [ ] **Step 4: Register it**

In the same file, immediately after the `handleApplyLockCommand` registration at `:96`:

```go
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleExtendExpirationCommand(db)))); err != nil {
				return err
			}
```

Read the surrounding registration lines first and match their exact error-handling shape — the block at `:96` is the template.

- [ ] **Step 5: Run tests to verify they pass**

```bash
cd services/atlas-inventory/atlas.com/inventory && go test -race ./... && go build ./... && go vet ./...
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-inventory/atlas.com/inventory/kafka/consumer/compartment/
git commit -m "feat(task-222): consume EXTEND_EXPIRATION in atlas-inventory"
```

---

### Task 10: atlas-saga-orchestrator — mirror the command contract and add the producer

atlas-saga-orchestrator carries its own copy of the compartment command contract (`kafka/message/compartment/kafka.go`), in a separate Go module from atlas-inventory's. A field name or json tag added to one and not the other fails no build — it decodes into a zero-valued body at runtime. The two copies must stay byte-compatible.

**Files:**
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/compartment/kafka.go:32` and `:164`
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/compartment/producer.go` (after `RequestApplyLockCommandProvider` at `:126`)
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/compartment/processor.go` (interface at `:44`, impl at `:130`)
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/compartment/mock/processor.go`
- Test: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/compartment/kafka_test.go`

**Interfaces:**
- Consumes: the atlas-inventory json tags from Task 6 — `slot`, `expiration`, `extenderTemplateId`, command type string `EXTEND_EXPIRATION`.
- Produces: `compartment.Processor.RequestExtendExpiration(transactionId uuid.UUID, characterId uint32, inventoryType byte, slot int16, expiration time.Time, extenderTemplateId uint32) error`. Task 11's step handler calls it.

- [ ] **Step 1: Write the failing test**

Append to `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/compartment/kafka_test.go`:

```go
func TestExtendExpirationCommandBodyWireShape(t *testing.T) {
	// This body is mirrored in atlas-inventory
	// (kafka/message/compartment/kafka.go). The two live in separate Go
	// modules, so a tag changed on one side and not the other decodes into a
	// zero-valued body at runtime rather than failing a build. Pin the tags.
	body := ExtendExpirationCommandBody{
		Slot:               -11,
		Expiration:         time.Date(2026, 9, 12, 0, 0, 0, 0, time.UTC),
		ExtenderTemplateId: 5500001,
	}
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"slot":-11,"expiration":"2026-09-12T00:00:00Z","extenderTemplateId":5500001}`
	if string(b) != want {
		t.Fatalf("marshalled = %s, want %s", b, want)
	}
	if CommandExtendExpiration != "EXTEND_EXPIRATION" {
		t.Fatalf("CommandExtendExpiration = %q, want EXTEND_EXPIRATION", CommandExtendExpiration)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test ./kafka/message/compartment/ -run TestExtendExpirationCommandBodyWireShape -v
```

Expected: FAIL — `undefined: ExtendExpirationCommandBody`.

- [ ] **Step 3: Mirror the contract**

In `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/compartment/kafka.go`, add to the command-type const block immediately after `CommandApplyLock`:

```go
	CommandExtendExpiration   = "EXTEND_EXPIRATION"
```

and immediately after the `ApplyLockCommandBody` struct:

```go
// ExtendExpirationCommandBody MIRRORS
// services/atlas-inventory/atlas.com/inventory/kafka/message/compartment/kafka.go.
// The two live in separate Go modules; keep the field names and json tags
// identical or the body decodes zero-valued at runtime with no build error.
type ExtendExpirationCommandBody struct {
	Slot               int16     `json:"slot"`
	Expiration         time.Time `json:"expiration"`
	ExtenderTemplateId uint32    `json:"extenderTemplateId"`
}
```

- [ ] **Step 4: Add the producer**

In `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/compartment/producer.go`, immediately after `RequestApplyLockCommandProvider`:

```go
func RequestExtendExpirationCommandProvider(transactionId uuid.UUID, characterId uint32, inventoryType byte, slot int16, expiration time.Time, extenderTemplateId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &compartment.Command[compartment.ExtendExpirationCommandBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		InventoryType: inventoryType,
		Type:          compartment.CommandExtendExpiration,
		Body: compartment.ExtendExpirationCommandBody{
			Slot:               slot,
			Expiration:         expiration,
			ExtenderTemplateId: extenderTemplateId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
```

- [ ] **Step 5: Add the processor method**

In `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/compartment/processor.go`, add to the `Processor` interface immediately after `RequestApplyLock`:

```go
	RequestExtendExpiration(transactionId uuid.UUID, characterId uint32, inventoryType byte, slot int16, expiration time.Time, extenderTemplateId uint32) error
```

and the implementation immediately after `RequestApplyLock`:

```go
func (p *ProcessorImpl) RequestExtendExpiration(transactionId uuid.UUID, characterId uint32, inventoryType byte, slot int16, expiration time.Time, extenderTemplateId uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(compartment.EnvCommandTopic)(RequestExtendExpirationCommandProvider(transactionId, characterId, inventoryType, slot, expiration, extenderTemplateId))
}
```

- [ ] **Step 6: Add the mock method**

In `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/compartment/mock/processor.go`, add beside `RequestApplyLockFunc`:

```go
	RequestExtendExpirationFunc    func(transactionId uuid.UUID, characterId uint32, inventoryType byte, slot int16, expiration time.Time, extenderTemplateId uint32) error
```

```go
// RequestExtendExpiration is a mock implementation of the compartment.Processor.RequestExtendExpiration method
func (m *ProcessorMock) RequestExtendExpiration(transactionId uuid.UUID, characterId uint32, inventoryType byte, slot int16, expiration time.Time, extenderTemplateId uint32) error {
	if m.RequestExtendExpirationFunc != nil {
		return m.RequestExtendExpirationFunc(transactionId, characterId, inventoryType, slot, expiration, extenderTemplateId)
	}
	return nil
}
```

- [ ] **Step 7: Run tests to verify they pass**

```bash
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test -race ./kafka/... ./compartment/... && go build ./... && go vet ./...
```

Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/compartment/ services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/compartment/
git commit -m "feat(task-222): mirror EXTEND_EXPIRATION contract and producer in saga-orchestrator"
```

---

### Task 11: atlas-saga-orchestrator — aliases, step handler, acceptance entry

**Files:**
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/model.go:44` (Type aliases), `:221` (Action aliases), `:326` (payload aliases), `:1609` (local unmarshal switch)
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go:950` (dispatch) and after `handleApplyAssetLock` at `:1138-1150`
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/event_acceptance.go:126`
- Test: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler_test.go`

**Interfaces:**
- Consumes: `sharedsaga.ExtendAssetExpiration`, `sharedsaga.ExtendAssetExpirationPayload`, `sharedsaga.ExpirationExtenderUse` (Task 5); `compartment.Processor.RequestExtendExpiration` (Task 10).
- Produces: local aliases `ExpirationExtenderUse`, `ExtendAssetExpiration`, `ExtendAssetExpirationPayload`. Task 12 uses `ExpirationExtenderUse`.

Note: `event_acceptance_test.go` carries a coverage test that fails when an action is missing from `acceptanceTable`, so Step 5 is not optional bookkeeping — it is load-bearing.

- [ ] **Step 1: Write the failing test**

Append to `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler_test.go`. Read the existing tests at `:1371` first and construct `HandlerImpl`, the saga, and the step exactly the way they do — if the file builds a handler through a constructor rather than a struct literal, follow that.

```go
func TestHandleExtendAssetExpirationIssuesCommand(t *testing.T) {
	var gotSlot int16
	var gotExpiration time.Time
	var gotExtender uint32
	var gotInvType byte
	cp := &compmock.ProcessorMock{
		RequestExtendExpirationFunc: func(_ uuid.UUID, _ uint32, inventoryType byte, slot int16, expiration time.Time, extenderTemplateId uint32) error {
			gotInvType = inventoryType
			gotSlot = slot
			gotExpiration = expiration
			gotExtender = extenderTemplateId
			return nil
		},
	}

	want := time.Date(2026, 9, 12, 0, 0, 0, 0, time.UTC)
	payload := ExtendAssetExpirationPayload{
		CharacterId:        12345,
		InventoryType:      1,
		Slot:               -11,
		Expiration:         want,
		ExtenderTemplateId: 5500001,
	}
	s := NewSagaBuilder().SetSagaType(ExpirationExtenderUse).Build()
	st := NewStepBuilder().SetAction(ExtendAssetExpiration).SetPayload(any(payload)).Build()

	h := &HandlerImpl{compP: cp}
	if err := h.handleExtendAssetExpiration(s, st); err != nil {
		t.Fatalf("handleExtendAssetExpiration: %v", err)
	}
	if gotInvType != 1 {
		t.Errorf("inventoryType = %d, want 1", gotInvType)
	}
	if gotSlot != -11 {
		t.Errorf("slot = %d, want -11", gotSlot)
	}
	if !gotExpiration.Equal(want) {
		t.Errorf("expiration = %v, want %v", gotExpiration, want)
	}
	if gotExtender != 5500001 {
		t.Errorf("extenderTemplateId = %d, want 5500001", gotExtender)
	}
}

func TestExtendAssetExpirationIsAcceptanceMapped(t *testing.T) {
	kinds, ok := acceptanceTable[sharedsaga.ExtendAssetExpiration]
	if !ok {
		t.Fatal("ExtendAssetExpiration missing from acceptanceTable; the step would default-deny every event and time out")
	}
	if len(kinds) != 1 || kinds[0] != EventKindAssetUpdated {
		t.Fatalf("acceptance kinds = %v, want [%v]", kinds, EventKindAssetUpdated)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test ./saga/ -run "TestHandleExtendAssetExpiration|TestExtendAssetExpirationIsAcceptanceMapped" -v
```

Expected: FAIL — `undefined: ExtendAssetExpirationPayload`.

- [ ] **Step 3: Add the aliases**

In `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/model.go`:

- after the `IncubatorUse` Type alias (`:44`):

```go
	ExpirationExtenderUse = sharedsaga.ExpirationExtenderUse
```

- after the `ApplyAssetLock` Action alias (`:221`):

```go
	ExtendAssetExpiration = sharedsaga.ExtendAssetExpiration
```

- after the `ApplyAssetLockPayload` type alias (`:326`):

```go
	ExtendAssetExpirationPayload        = sharedsaga.ExtendAssetExpirationPayload
```

- in the local unmarshal switch, after the `case ApplyAssetLock:` block (`:1609`). Read `:1603-1615` first and match the surrounding arms' error text and assignment form exactly:

```go
	case ExtendAssetExpiration:
		var payload ExtendAssetExpirationPayload
		if err := json.Unmarshal(aux.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.Action, err)
		}
		s.Payload = any(payload).(T)
```

- [ ] **Step 4: Add the step handler and dispatch entry**

In `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go`, add to the action switch immediately after the `case ApplyAssetLock:` arm (`:949-950`):

```go
	case ExtendAssetExpiration:
		return h.handleExtendAssetExpiration, true
```

and the handler immediately after `handleApplyAssetLock`:

```go
// handleExtendAssetExpiration handles the ExtendAssetExpiration action
func (h *HandlerImpl) handleExtendAssetExpiration(s Saga, st Step[any]) error {
	payload, ok := st.Payload().(ExtendAssetExpirationPayload)
	if !ok {
		return errors.New("invalid payload")
	}
	err := h.compP.RequestExtendExpiration(s.TransactionId(), payload.CharacterId, payload.InventoryType, payload.Slot, payload.Expiration, payload.ExtenderTemplateId)
	if err != nil {
		h.logActionError(s, st, err, "Unable to extend asset expiration.")
		return err
	}
	return nil
}
```

- [ ] **Step 5: Add the acceptance-table entry**

In `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/event_acceptance.go`, in the "Asset actions" group immediately after the `sharedsaga.ApplyAssetLock` line:

```go
	sharedsaga.ExtendAssetExpiration: {EventKindAssetUpdated},
```

A missing entry default-denies every event in `StepAcceptsEvent`, so the step would never complete and the saga would sit until timeout.

- [ ] **Step 6: Run tests to verify they pass**

```bash
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test -race ./saga/... && go build ./... && go vet ./...
```

Expected: PASS — including the pre-existing acceptance-coverage test in `event_acceptance_test.go`.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/
git commit -m "feat(task-222): handle ExtendAssetExpiration step in saga-orchestrator"
```

---

### Task 12: atlas-saga-orchestrator — timer and compensator registration

Design §9 names this the single highest-risk omission in the change: a saga type absent from `timer.go`'s lists gets no timeout and no reverse walk, and one absent from `compensator.go:267` never refunds the consumed sandglass. There are **four** distinct registration sites.

`DispatchCashItemUseRollbacks` needs no change — its `DestroyAsset` arm already re-creates the consumed item by `TemplateId`, which is exactly the sandglass refund.

**Files:**
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/timer.go:176` (`reverseWalkSagaTypes`), `:205` (`allSagaTypes`), `:237` (`dispatchTimeoutRollbacks` switch)
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/compensator.go:263-267`
- Test: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/compensator_test.go`

**Interfaces:**
- Consumes: `ExpirationExtenderUse` (Task 11).
- Produces: nothing consumed by later tasks.

- [ ] **Step 1: Write the failing test**

Append to `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/compensator_test.go`. Read the existing cash-item-use compensation test at `:213` and construct the compensator, saga, and steps exactly the way it does — replace `NewCompensatorWith(cp)` below with whatever seam that test already uses to inject a mock compartment processor.

```go
func TestExpirationExtenderUseRefundsConsumedExtender(t *testing.T) {
	var createdTemplate uint32
	var createdQty uint32
	cp := &compmock.ProcessorMock{
		RequestCreateItemFunc: func(_ uuid.UUID, _ uint32, templateId uint32, quantity uint32, _ time.Time) error {
			createdTemplate = templateId
			createdQty = quantity
			return nil
		},
	}

	consume := NewStepBuilder().
		SetStepId("consume_expiration_extender").
		SetAction(DestroyAsset).
		SetStatus(Completed).
		SetPayload(any(DestroyAssetPayload{CharacterId: 12345, TemplateId: 5500001, Quantity: 1})).
		Build()
	extend := NewStepBuilder().
		SetStepId("extend_asset_expiration").
		SetAction(ExtendAssetExpiration).
		SetStatus(Failed).
		SetPayload(any(ExtendAssetExpirationPayload{CharacterId: 12345, InventoryType: 1, Slot: -11, ExtenderTemplateId: 5500001})).
		Build()
	s := NewSagaBuilder().
		SetSagaType(ExpirationExtenderUse).
		SetSteps([]Step[any]{consume, extend}).
		Build()

	NewCompensatorWith(cp).DispatchCashItemUseRollbacks(s)

	if createdTemplate != 5500001 {
		t.Errorf("refunded template = %d, want 5500001", createdTemplate)
	}
	if createdQty != 1 {
		t.Errorf("refunded quantity = %d, want 1", createdQty)
	}
}

func TestExpirationExtenderUseIsTimerClassified(t *testing.T) {
	inReverse := false
	for _, ty := range reverseWalkSagaTypes {
		if ty == ExpirationExtenderUse {
			inReverse = true
		}
	}
	if !inReverse {
		t.Error("ExpirationExtenderUse missing from reverseWalkSagaTypes: a timed-out saga would leave the extender consumed with no extension")
	}
	inAll := false
	for _, ty := range allSagaTypes {
		if ty == ExpirationExtenderUse {
			inAll = true
		}
	}
	if !inAll {
		t.Error("ExpirationExtenderUse missing from allSagaTypes")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test ./saga/ -run "TestExpirationExtenderUse" -v
```

Expected: FAIL — `TestExpirationExtenderUseIsTimerClassified` reports both missing lists. The pre-existing `TestEverySagaTypeIsClassified` may also fail once the type is added to only one list, which is the intended safety net.

- [ ] **Step 3: Register in `reverseWalkSagaTypes`**

In `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/timer.go`, in `reverseWalkSagaTypes`, immediately after `IncubatorUse`:

```go
	ExpirationExtenderUse,
```

- [ ] **Step 4: Register in `allSagaTypes`**

In the same file, in `allSagaTypes`, extend the line containing `PetEvolution, ItemTagUse, SealingLockUse, IncubatorUse, PointReset,` so it reads:

```go
	PetEvolution, ItemTagUse, SealingLockUse, IncubatorUse, ExpirationExtenderUse, PointReset,
```

- [ ] **Step 5: Register in the timeout dispatch switch**

In the same file, change the `case ItemTagUse, SealingLockUse, IncubatorUse:` arm (`:237`) to:

```go
	case ItemTagUse, SealingLockUse, IncubatorUse, ExpirationExtenderUse:
		// The four share one compensator, exactly as CompensateFailedStep
		// routes them.
		c.DispatchCashItemUseRollbacks(s)
```

- [ ] **Step 6: Register in the compensator branch**

In `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/compensator.go:267`, change:

```go
	if s.SagaType() == ItemTagUse || s.SagaType() == SealingLockUse || s.SagaType() == IncubatorUse {
```

to:

```go
	if s.SagaType() == ItemTagUse || s.SagaType() == SealingLockUse || s.SagaType() == IncubatorUse || s.SagaType() == ExpirationExtenderUse {
```

Also update the comment immediately above it (`:263-266`) to name `expiration_extender_use` alongside the other three.

- [ ] **Step 7: Run tests to verify they pass**

```bash
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test -race ./... && go vet ./...
```

Expected: PASS — including `TestEverySagaTypeIsClassified`.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/
git commit -m "feat(task-222): register ExpirationExtenderUse in timer and compensator tables"
```

---

### Task 13: atlas-channel mirrors the two data models

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/data/cash/rest.go`
- Modify: `services/atlas-channel/atlas.com/channel/data/equipment/rest.go`
- Modify: `services/atlas-channel/atlas.com/channel/data/equipment/model.go`
- Test: `services/atlas-channel/atlas.com/channel/data/equipment/processor_test.go`

**Interfaces:**
- Consumes: json tags `addTime`, `maxDays` (Task 1) and `notExtend` (Task 2).
- Produces: `cash.RestModel.AddTime uint32` / `.MaxDays uint32`; `equipment.Model.NotExtend() bool`. Task 14 reads both.

- [ ] **Step 1: Write the failing test**

Append to `services/atlas-channel/atlas.com/channel/data/equipment/processor_test.go`, matching that file's existing httptest harness and its way of building a tenant context (`testContext` below is a placeholder for whatever it already uses):

```go
func TestGetByIdReadsNotExtend(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"type":"statistics","id":"1402046","attributes":{"notExtend":true}}}`))
	}))
	defer srv.Close()
	t.Setenv("DATA_SERVICE_URL", srv.URL+"/")

	l, _ := test.NewNullLogger()
	m, err := NewProcessor(l, testContext(t)).GetById(1402046)
	if err != nil {
		t.Fatal(err)
	}
	if !m.NotExtend() {
		t.Error("NotExtend = false, want true")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-channel/atlas.com/channel && go test ./data/equipment/ -run TestGetByIdReadsNotExtend -v
```

Expected: FAIL — `m.NotExtend undefined`.

- [ ] **Step 3: Mirror `notExtend` on the equipment model**

In `services/atlas-channel/atlas.com/channel/data/equipment/rest.go`, add the field to `RestModel`:

```go
	NotExtend    bool     `json:"notExtend"`
```

and thread it through `Extract`:

```go
func Extract(rm RestModel) (Model, error) {
	return Model{
		id:           rm.Id,
		petAbilities: rm.PetAbilities,
		notExtend:    rm.NotExtend,
	}, nil
}
```

In `services/atlas-channel/atlas.com/channel/data/equipment/model.go`, add the field and getter:

```go
type Model struct {
	id           uint32
	petAbilities []string
	notExtend    bool
}
```

```go
// NotExtend reports whether info/notExtend is set on the equip template — a
// WZ blacklist that forbids applying an item-expiration extender (Magical
// Sandglass) to it. The client enforces it via CItemInfo::IsNotExtendItem;
// the server re-checks so a crafted request cannot bypass it.
func (m Model) NotExtend() bool {
	return m.notExtend
}
```

- [ ] **Step 4: Mirror `addTime` / `maxDays` on the cash model**

In `services/atlas-channel/atlas.com/channel/data/cash/rest.go`, add to `RestModel` after `ProtectTime`:

```go
	// AddTime is the expiration grant in SECONDS; MaxDays is the ceiling in
	// DAYS, anchored to now. Mirrors atlas-data's cash resource.
	AddTime         uint32 `json:"addTime"`
	MaxDays         uint32 `json:"maxDays"`
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
cd services/atlas-channel/atlas.com/channel && go test -race ./data/... && go build ./... && go vet ./...
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/data/
git commit -m "feat(task-222): mirror addTime/maxDays and notExtend on atlas-channel data models"
```

---

### Task 14: atlas-channel — the `CASH_ITEM_USE` arm

The gates and the formula go in a **pure** function so they are table-testable without inventing five package-var seams; the handler arm stays thin. This mirrors the file-splitting precedent of `character_cash_item_use_point_reset.go`.

**Version resolver.** The value must stay version-scoped, and not for style. At GMS >= 95, plain `61` is the megaphone arm (`ClassificationMegaphones`, `otherCategory == 7`, `character_cash_item_use.go:829`); at GMS < 95, plain `62` is classification 551 (`:1041`). A bare literal on either side would route another family's packet into this arm.

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_expiration_extender.go`
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_expiration_extender_test.go`
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go` (const block at `:658-673`; new arm after the seal arm ends at `:331`)
- Modify: `services/atlas-channel/atlas.com/channel/saga/model.go` (aliases)

**Interfaces:**
- Consumes: `item.ClassificationExpirationExtender` (Task 3); `cashsb.NewItemUseTargetSlot` (Task 4); `saga.ExpirationExtenderUse` / `saga.ExtendAssetExpiration` / `saga.ExtendAssetExpirationPayload` (Task 5); `cashData.RestModel.AddTime`/`.MaxDays` and `equipment.Model.NotExtend()` (Task 13); `asset.Model.Expiration()/.Locked()/.CashId()/.TemplateId()`.
- Produces: nothing consumed by later tasks.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_expiration_extender_test.go`:

```go
package handler

import (
	"testing"
	"time"

	"github.com/google/uuid"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func testExtenderTenant(t *testing.T, region string, major uint16, minor uint16) tenant.Model {
	t.Helper()
	te, err := tenant.Create(uuid.New(), region, major, minor)
	if err != nil {
		t.Fatal(err)
	}
	return te
}

func TestExpirationExtenderCashSlotItemTypeIsVersionScoped(t *testing.T) {
	// 62 at GMS >= 95, 61 below — IDA-verified: gms_v83
	// get_cashslot_item_type @0x48645B case 550 -> 61; gms_v95 @0x488C70
	// case 550 -> 62. It must never be a bare literal: 61 is the
	// otherCategory==7 megaphone arm at GMS >= 95, and 62 is classification
	// 551 below it.
	cases := []struct {
		region string
		major  uint16
		want   CashSlotItemType
	}{
		{"GMS", 72, CashSlotItemTypeExpirationExtender},
		{"GMS", 83, CashSlotItemTypeExpirationExtender},
		{"GMS", 87, CashSlotItemTypeExpirationExtender},
		{"GMS", 95, CashSlotItemTypeExpirationExtenderV95},
		{"JMS", 185, CashSlotItemTypeExpirationExtender},
	}
	for _, c := range cases {
		te := testExtenderTenant(t, c.region, c.major, 1)
		if got := expirationExtenderCashSlotItemType(te); got != c.want {
			t.Errorf("%s v%d: got %d, want %d", c.region, c.major, got, c.want)
		}
	}
}

func TestExpirationExtenderResolverAgreesWithClassifier(t *testing.T) {
	// The arm matches on the resolver, but dispatch computes the type through
	// GetCashSlotItemType. If the two ever disagree the arm is unreachable.
	for _, major := range []uint16{72, 79, 83, 84, 87, 92, 95} {
		te := testExtenderTenant(t, "GMS", major, 1)
		classified := GetCashSlotItemType(te)(5500001)
		resolved := expirationExtenderCashSlotItemType(te)
		if classified != resolved {
			t.Errorf("GMS v%d: classifier gave %d, resolver gave %d", major, classified, resolved)
		}
	}
}

func TestEvaluateExpirationExtension(t *testing.T) {
	now := time.Date(2026, 8, 13, 0, 0, 0, 0, time.UTC)
	day := 24 * time.Hour

	cases := []struct {
		name        string
		expiration  time.Time
		locked      bool
		cashId      int64
		notExtend   bool
		addTime     uint32
		maxDays     uint32
		wantReject  string
		wantNewTime time.Time
	}{
		{
			name:        "under cap accepts",
			expiration:  now.Add(5 * day),
			addTime:     604800, // +7d
			maxDays:     30,
			wantNewTime: now.Add(12 * day),
		},
		{
			name:        "exactly at cap accepts",
			expiration:  now.Add(23 * day),
			addTime:     604800, // +7d -> exactly now+30d
			maxDays:     30,
			wantNewTime: now.Add(30 * day),
		},
		{
			name:       "over cap rejects without consuming",
			expiration: now.Add(25 * day),
			addTime:    604800, // +7d -> now+32d, past the ceiling
			maxDays:    30,
			wantReject: "over cap",
		},
		{
			name:       "already past cap rejects",
			expiration: now.Add(40 * day),
			addTime:    604800,
			maxDays:    30,
			wantReject: "over cap",
		},
		{
			name:       "99-day extender against a 30-day ceiling always rejects",
			expiration: now.Add(1 * day),
			addTime:    8553600, // 99d
			maxDays:    30,
			wantReject: "over cap",
		},
		{
			name:       "zero maxDays rejects",
			expiration: now.Add(5 * day),
			addTime:    604800,
			maxDays:    0,
			wantReject: "no ceiling",
		},
		{
			name:       "permanent target rejects",
			expiration: time.Time{},
			addTime:    604800,
			maxDays:    30,
			wantReject: "permanent",
		},
		{
			name:       "cash equip rejects",
			expiration: now.Add(5 * day),
			cashId:     987654321,
			addTime:    604800,
			maxDays:    30,
			wantReject: "cash",
		},
		{
			name:       "locked target rejects",
			expiration: now.Add(5 * day),
			locked:     true,
			addTime:    604800,
			maxDays:    30,
			wantReject: "lock",
		},
		{
			name:       "notExtend target rejects",
			expiration: now.Add(5 * day),
			notExtend:  true,
			addTime:    604800,
			maxDays:    30,
			wantReject: "notExtend",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := evaluateExpirationExtension(now, extensionTarget{
				Expiration: c.expiration,
				Locked:     c.locked,
				CashId:     c.cashId,
				NotExtend:  c.notExtend,
			}, c.addTime, c.maxDays)

			if c.wantReject != "" {
				if got.Reason == "" {
					t.Fatalf("expected rejection (%s), got acceptance with %v", c.wantReject, got.Expiration)
				}
				return
			}
			if got.Reason != "" {
				t.Fatalf("expected acceptance, got rejection: %s", got.Reason)
			}
			if !got.Expiration.Equal(c.wantNewTime) {
				t.Errorf("Expiration = %v, want %v", got.Expiration, c.wantNewTime)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run "TestExpirationExtender|TestEvaluateExpirationExtension" -v
```

Expected: FAIL — `undefined: expirationExtenderCashSlotItemType`, `undefined: evaluateExpirationExtension`.

- [ ] **Step 3: Add the two type constants**

In `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go`, in the `CashSlotItemType` const block beside `CashSlotItemTypeViciousHammer` / `CashSlotItemTypeViciousHammerV95`:

```go
	CashSlotItemTypeExpirationExtender    = CashSlotItemType(61) // GMS < 95, JMS
	CashSlotItemTypeExpirationExtenderV95 = CashSlotItemType(62) // GMS >= 95
```

- [ ] **Step 4: Create the resolver and the pure evaluator**

Create `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_expiration_extender.go`:

```go
package handler

import (
	"time"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// expirationExtenderCashSlotItemType returns the version-scoped
// CashSlotItemType for item classification 550 (item-expiration extenders,
// the Magical Sandglass family).
//
// This MUST remain version-scoped rather than a single constant. Plain 61 is
// also the otherCategory==7 megaphone arm on GMS >= 95, and plain 62 is
// classification 551 on GMS < 95 — a bare literal on either side would route
// another family's sub-body into this arm and mis-decode it.
//
// IDA-verified: gms_v72 get_cashslot_item_type @0x49FB33, gms_v79 @0x47EC3E,
// gms_v83 @0x48645B and gms_v87 @0x473D96 all map case 550 -> 61; gms_v95
// @0x488C70 maps it to 62. gms_v48 and gms_v61 have no arm for the family at
// all (their SendConsumeCashItemUseRequest switches cover types 12-47 and
// 12-52 respectively), so the arm is simply unreachable there.
func expirationExtenderCashSlotItemType(t tenant.Model) CashSlotItemType {
	if t.Region() == "GMS" && t.MajorVersion() >= 95 {
		return CashSlotItemTypeExpirationExtenderV95
	}
	return CashSlotItemTypeExpirationExtender
}

// extensionTarget is the subset of the target equip's state the eligibility
// gates need, lifted out of asset.Model and the equipment data so the
// decision is a pure function.
type extensionTarget struct {
	Expiration time.Time
	Locked     bool
	CashId     int64
	NotExtend  bool
}

// extensionOutcome is the result of evaluating an extender use. A non-empty
// Reason means the use is rejected: nothing is consumed and nothing is
// mutated.
type extensionOutcome struct {
	Expiration time.Time
	Reason     string
}

// evaluateExpirationExtension applies the client's own gates and formula.
//
// Formula (CDraggableItem::ModifyEquipItem, gms_v83 @0x4F4BB7):
//
//	cap      = now + maxDays*24h        // anchored to NOW, not to the target
//	proposed = target.Expiration + addTime seconds
//	accept iff proposed <= cap
//
// An over-cap use is REJECTED, never clamped-and-consumed: the client shows
// "You cannot extend the effective date beyond %d days" and sends nothing, so
// clamping would only ever run for a forged packet — and burning a player's
// extender for a partial grant the client refused is a visible loss.
func evaluateExpirationExtension(now time.Time, target extensionTarget, addTime uint32, maxDays uint32) extensionOutcome {
	if target.Expiration.IsZero() {
		return extensionOutcome{Reason: "target is permanent and has no time limit to extend"}
	}
	if target.CashId != 0 {
		return extensionOutcome{Reason: "target is a cash equip"}
	}
	if target.Locked {
		return extensionOutcome{Reason: "target expiration is a sealing-lock window, not a time limit"}
	}
	if target.NotExtend {
		return extensionOutcome{Reason: "target is flagged notExtend"}
	}
	if maxDays == 0 {
		return extensionOutcome{Reason: "extender has no maxDays ceiling"}
	}
	ceiling := now.Add(time.Duration(maxDays) * 24 * time.Hour)
	proposed := target.Expiration.Add(time.Duration(addTime) * time.Second)
	if proposed.After(ceiling) {
		return extensionOutcome{Reason: "extension would push the expiration past the maxDays ceiling"}
	}
	return extensionOutcome{Expiration: proposed}
}
```

- [ ] **Step 5: Run the pure tests to verify they pass**

```bash
cd services/atlas-channel/atlas.com/channel && go test -race ./socket/handler/ -run "TestExpirationExtender|TestEvaluateExpirationExtension" -v
```

Expected: PASS, all sub-cases.

- [ ] **Step 6: Add the saga aliases**

In `services/atlas-channel/atlas.com/channel/saga/model.go`, add each of these to the alias block that already holds its neighbour (`Type`, `Action`, payload types respectively):

```go
	ExpirationExtenderUse = sharedsaga.ExpirationExtenderUse
```

```go
	ExtendAssetExpiration = sharedsaga.ExtendAssetExpiration
```

```go
	ExtendAssetExpirationPayload = sharedsaga.ExtendAssetExpirationPayload
```

- [ ] **Step 7: Wire the handler arm**

In `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go`, immediately after the sealing-lock arm's closing `}` (currently line 331, right before `if it == CashSlotItemTypeIncubator {`):

```go
		if it == expirationExtenderCashSlotItemType(t) {
			// Sub-body: a bare int16 equip position, shared verbatim with the
			// Item Tag arm (the client uses one jump-table target for both --
			// gms_v83 SendConsumeCashItemUseRequest @0xA0CAE0, "cases 25,61").
			// No inventory type is on the wire: the client hard-codes EQUIP
			// (CharacterData::GetItem(charData, 1, -hitTestResult)), so the
			// compartment is EQUIP unconditionally and the slot is negative.
			sp := cashsb.NewItemUseTargetSlot(updateTimeFirst)
			sp.Decode(l, ctx)(r, readerOptions)
			targetSlot := sp.Slot()

			target, err := character2.NewProcessor(l, ctx).GetItemInSlot(s.CharacterId(), inventory.TypeValueEquip, targetSlot)()
			if err != nil {
				l.Warnf("Character [%d] attempted to use expiration extender [%d] on empty equip slot [%d].", s.CharacterId(), itemId, targetSlot)
				return
			}

			cd, err := cashData.NewProcessor(l, ctx).GetById(uint32(itemId))
			if err != nil {
				l.WithError(err).Warnf("Character [%d] unable to resolve cash item data for expiration extender [%d].", s.CharacterId(), itemId)
				return
			}

			ed, err := equipmentData.NewProcessor(l, ctx).GetById(target.TemplateId())
			if err != nil {
				l.WithError(err).Warnf("Character [%d] unable to resolve equipment data for extender target [%d] in slot [%d].", s.CharacterId(), target.TemplateId(), targetSlot)
				return
			}

			outcome := evaluateExpirationExtension(time.Now(), extensionTarget{
				Expiration: target.Expiration(),
				Locked:     target.Locked(),
				CashId:     target.CashId(),
				NotExtend:  ed.NotExtend(),
			}, cd.AddTime, cd.MaxDays)
			if outcome.Reason != "" {
				l.Warnf("Character [%d] expiration extender [%d] rejected on equip slot [%d] target [%d]: %s.", s.CharacterId(), itemId, targetSlot, target.TemplateId(), outcome.Reason)
				return
			}

			// No EnableActions: this arm mutates inventory without warping,
			// matching the sealing-lock and kite arms.
			transactionId := uuid.New()
			now := time.Now()
			_ = saga.NewProcessor(l, ctx).Create(saga.Saga{
				TransactionId: transactionId,
				SagaType:      saga.ExpirationExtenderUse,
				InitiatedBy:   "CASH_ITEM_USE",
				Steps: []saga.Step{
					{
						StepId: "consume_expiration_extender",
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
						StepId: "extend_asset_expiration",
						Status: saga.Pending,
						Action: saga.ExtendAssetExpiration,
						Payload: saga.ExtendAssetExpirationPayload{
							CharacterId:        s.CharacterId(),
							InventoryType:      byte(inventory.TypeValueEquip),
							Slot:               targetSlot,
							Expiration:         outcome.Expiration,
							ExtenderTemplateId: uint32(itemId),
						},
						CreatedAt: now,
						UpdatedAt: now,
					},
				},
			})
			return
		}
```

Add `equipmentData "atlas-channel/data/equipment"` to the file's import block (the file already aliases `cashData "atlas-channel/data/cash"` at `:7`).

- [ ] **Step 8: Run tests to verify they pass**

```bash
cd services/atlas-channel/atlas.com/channel && go build ./... && go test -race ./... && go vet ./...
```

Expected: build clean, all tests PASS, vet clean.

- [ ] **Step 9: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/ services/atlas-channel/atlas.com/channel/saga/model.go
git commit -m "feat(task-222): add expiration-extender arm to CASH_ITEM_USE"
```

---

### Task 15: Version confirmation and the full verification sweep

Two pieces of due diligence, then the CLAUDE.md build gates. Every command runs from the worktree root.

**GMS v84 / v92.** Design §1.1 confirmed `case 550 -> 61` on v72, v79, v83, and v87, and `-> 62` on v95, but `get_cashslot_item_type` is unnamed in the v84 and v92 IDBs. This task settles them by the same jump-table method used for v48/v61. The risk is bounded: `GetCashSlotItemType`'s classification-550 branch already carried this mapping before this task, and Task 14's resolver mirrors that branch's version condition exactly, so the two cannot disagree — this confirms a pre-existing mapping rather than gating new code.

**Templates.** Design §6 expects no socket-config template edits (no new opcode; `CASH_ITEM_USE` is an existing registered handler). Confirm rather than assume.

**Files:**
- Modify: `docs/tasks/task-222-item-expiration-extenders/design.md` (§1.1 and §6 — record the v84/v92 findings)

- [ ] **Step 1: Confirm the v84 and v92 mapping in IDA**

Resolve the session from `idb_list` by binary **name** and pass it as the `database` parameter — port-based selection is dead. Locate `get_cashslot_item_type` by its `switch (nItemID / 10000)` shape, or settle it from the dispatch switch bound in `CWvsContext::SendConsumeCashItemUseRequest` the way design §1.1 settled v48/v61 (`add eax, -12` / `cmp eax, <range>` bounds the covered type range).

Record for each of v84 and v92: the IDB name, the function address, and the `case 550` result. If either turns out **not** to be 61, stop and report — `GetCashSlotItemType`'s existing 550 branch would then be wrong for that version and needs a version arm, which is a change to pre-existing behaviour and needs sign-off.

- [ ] **Step 2: Update the design's evidence table**

Replace the two `assumed yes / 61 (unverified)` rows in design.md §6 and the "Not verified: GMS v84 and GMS v92" paragraph in §1.1 with the actual findings and their addresses. If a version could not be settled, say so plainly — do not record a guess as verified.

- [ ] **Step 3: Confirm no template gap**

```bash
grep -l "CharacterCashItemUseHandle" services/atlas-configurations/seed-data/templates/*.json
```

Expected: every in-scope GMS/JMS template that has the family (v72 and up, plus JMS) appears. v48 and v61 have no arm for the family, so their absence is expected and is not a gap. If an in-scope version is missing the registration, record it in design.md §6 as a finding; any template edit must then satisfy `tools/template-opcode-order-guard.sh` and `tools/template-duplicate-binding-guard.sh`.

- [ ] **Step 4: Per-module test and vet sweep**

```bash
for m in libs/atlas-constants libs/atlas-packet libs/atlas-saga \
         services/atlas-data/atlas.com/data \
         services/atlas-channel/atlas.com/channel \
         services/atlas-inventory/atlas.com/inventory \
         services/atlas-saga-orchestrator/atlas.com/saga-orchestrator; do
  echo "=== $m ==="
  ( cd "$m" && go build ./... && go test -race ./... && go vet ./... ) || echo "FAILED: $m"
done
```

Expected: every module builds, tests PASS, vet clean; no `FAILED:` line.

- [ ] **Step 5: Repo guards**

```bash
tools/redis-key-guard.sh
tools/goroutine-guard.sh
tools/skill-job-id-guard.sh
tools/buff-duration-guard.sh
tools/lint.sh --check
```

Expected: all exit 0. If `tools/lint.sh --check` reports formatting drift, run `tools/lint.sh` (no flags) to fix in place, re-run `--check`, and commit the result. `lint.sh --check` false-fails without nvm on PATH — if it errors before running, load nvm first.

The template guards are only required if Step 3 produced a template edit.

- [ ] **Step 6: Docker bake for every service whose `go.mod` was touched**

Adding `atlas-inventory/data/cash` does not move a `go.mod`, but confirm rather than assume:

```bash
git diff --name-only main...HEAD -- '*go.mod' '*go.sum'
```

Then bake the four changed services regardless — the shared root `Dockerfile` is parameterized by `ARG SERVICE` and a missing `COPY libs/...` line is invisible to `go build` against the workspace:

```bash
docker buildx bake atlas-data atlas-channel atlas-inventory atlas-saga-orchestrator
```

If the `git diff` named any other service, bake its target too. Expected: every target builds.

- [ ] **Step 7: Live verification**

Not inferrable, and required by the PRD's acceptance criteria. On a v83 tenant: equip a time-limited item, drag a 7-day Magical Sandglass (5500001) onto it, and confirm from the client that the tooltip shows the new expiration **without a relog** and that exactly one sandglass was consumed. Then confirm a rejection path leaves the sandglass in place — drag it onto a permanent equip and check the channel log for the warn line naming the reason.

Record the result. If the live check cannot be run in this environment, say so explicitly in the PR rather than marking the criterion met.

- [ ] **Step 8: Commit the evidence updates**

```bash
git add docs/tasks/task-222-item-expiration-extenders/design.md
git commit -m "docs(task-222): record v84/v92 slot-item-type confirmation and template audit"
```

- [ ] **Step 9: Code review before the PR**

Run `superpowers:requesting-code-review`. It dispatches `plan-adherence-reviewer` and `backend-guidelines-reviewer` (no frontend files changed). Pin the reviewer subagents to a cheaper model per the project's cost preference. Findings land in `docs/tasks/task-222-item-expiration-extenders/audit.md`. Do not open the PR before this step completes.

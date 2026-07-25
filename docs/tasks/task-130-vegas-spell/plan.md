# Vega's Spell Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement Vega's Spell 10/60 (cash items 5610000/5610001) end-to-end: the serverbound sub-body decode in atlas-channel, a `REQUEST_VEGA_SCROLL` command in atlas-consumables that reserves the vega (CASH) and scroll (USE) items via chained single-item reservations and applies the scroll at the boosted rate (10%→30%, 60%→90%), and the new `VegaScroll` clientbound writer with config-resolved, outcome-keyed mode bytes.

**Architecture:** Design §3 decisions — a distinct `REQUEST_VEGA_SCROLL` command on the existing `COMMAND_TOPIC_CONSUMABLE` (never touching `RequestScrollBody`); chained single-item reservations CASH-first (never batched — the inventory batch path only processes the first entry, design §2.8); a shared `applyScrollCore` extracted from `ConsumeScroll` with `successProb` as a parameter; immediate start+result packet emission (no server-side delay, owner decision); outcome-keyed operations table because v95's START byte carries the outcome (design §2.3).

**Tech Stack:** Go, Kafka (atlas-kafka), atlas-packet codecs with `WithResolvedCode` operations resolution, IDA-verified byte fixtures via tools/packet-audit.

## Global Constraints

- Spec: `docs/tasks/task-130-vegas-spell/design.md` (authoritative); PRD at `prd.md` in the same folder. Design section references (§N.N) below point there.
- Item ids: `VegasSpell10 = 5610000`, `VegasSpell60 = 5610001`. Rate policy (server-side, NOT WZ data): required 10 → boosted 30; required 60 → boosted 90. Exact match on the scroll's natural `SuccessRate()` only.
- `whiteScroll=false` and `legendarySpirit=false` everywhere on the vega path.
- Serverbound sub-body (IDA-verified v83 `sub_82CBE2`, v95 `CUIVega::OnButtonClicked` 0x7bf4a0 — design §2.1): **six int32s** — equipTab(=1), equipSlot, scrollTab(=2), scrollSlot, flag(=1, read and ignored), updateTime. The trailing updateTime is present on EVERY version, independent of the prefix's `updateTimeFirst` convention.
- `VEGA_SCROLL` clientbound opcodes: gms_v83 **0x166 verified**, gms_v95 **0x1AD verified**; gms_v87 0x17B and jms_v185 0x183 are csv-import (verify in Task 4); gms_v84's 0x166 is **suspect** (stale csv carryover, design §2.5 — wire only after re-verification); gms_v92 **parked** (no IDB, no registry file, no USE_CASH_ITEM handler — design §2.6).
- Operations table values (design §2.3): v83 `START_SUCCESS=64, START_FAILURE=64, RESULT_SUCCESS=65, RESULT_FAILURE=67, INVALID=66`; v95 `START_SUCCESS=68, START_FAILURE=73, RESULT_SUCCESS=69, RESULT_FAILURE=71, INVALID=66` — the v95 START success/fail pairing (68 vs 73) is pinned in Task 4 via string-pool templates 5417/5418 and may swap.
- Mode bytes are ALWAYS config-resolved via `atlas_packet.WithResolvedCode("operations", <fixed named-constant key>, ...)`. Never hard-code a mode byte; never pass a variable as the key (owner-established uniformity policy).
- Reservations: one single-item `RequestReserve` per compartment, chained CASH → USE → consume. NEVER batch two reserves into one call (design §2.8: `RequestReserve` inventory-side returns after the first entry).
- The pre-existing `successRoll <= int32(successProb)` comparator is deliberately inherited unchanged (PRD non-goal: no scroll-math changes).
- IDA discipline: `mcp__ida-pro__list_instances` and match the **binary name** before any read (the loaded set rotates). An unresolvable fname/opcode is **STOP AND REPORT BLOCKED** — never substitute, guess, or fake evidence. Follow `docs/packets/audits/VERIFYING_A_PACKET.md` exactly for every fixture.
- Task-126 coordination (design §4.2, §4.8): task-126 touches the same `candidatesFromFName` fname case, the shared USE_CASH_ITEM serverbound audit, and adds `CharacterCashItemUseHandle` to the gms_87/95/jms templates. Whichever task lands second **splices** into the existing artifact (append candidate / append handler entry / splice audit arm) — never overwrite, never double-add.
- Test setup uses the project Builder pattern (`asset.NewBuilder`, `compartment.NewBuilder`, `character.NewModelBuilder`, `consumable3.RestModel` + `Extract`); no `*_testhelpers.go` files.
- Verification gates (CLAUDE.md): `go test -race ./...`, `go vet ./...`, `go build ./...` per changed module; `docker buildx bake` for changed services (shared libs change → `all-go-services`); `tools/redis-key-guard.sh`; `go run ./tools/packet-audit matrix --check`; `tools/template-symbol-check.sh` per touched template.
- Commit after every task. Run all commands from the worktree root (`.worktrees/task-130-vegas-spell`) unless a step says otherwise.

---

### Task 1: Vega item constants in libs/atlas-constants

**Files:**
- Create: `libs/atlas-constants/item/vegas_spell.go`
- Create: `libs/atlas-constants/item/vegas_spell_test.go`

**Interfaces:**
- Consumes: existing `item.Id`, `item.Classification`, `item.Is` (`constants.go:226`), `item.GetClassification`.
- Produces (used by Tasks 5, 7, 10): `item.VegasSpell10`, `item.VegasSpell60`, `item.ClassificationVegasSpell`, `func IsVegasSpell(id Id) bool`.

DOM-21 note: re-checked at plan time — no vega ids/classifier exist anywhere in `libs/atlas-constants` (grep for `VegasSpell|5610000` is empty). The file-per-topic layout follows `death_protection.go`.

- [ ] **Step 1: Write the failing test**

Create `libs/atlas-constants/item/vegas_spell_test.go`:

```go
package item_test

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

func TestIsVegasSpell(t *testing.T) {
	cases := []struct {
		name string
		id   item.Id
		want bool
	}{
		{"Vega's Spell 10", item.VegasSpell10, true},
		{"Vega's Spell 60", item.VegasSpell60, true},
		{"adjacent cash id", item.Id(5610002), false},
		{"chaos scroll", item.ChaosScrollSixtyPercent, false},
		{"zero", item.Id(0), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := item.IsVegasSpell(tc.id); got != tc.want {
				t.Errorf("IsVegasSpell(%d) = %t, want %t", tc.id, got, tc.want)
			}
		})
	}
}

func TestVegasSpellClassification(t *testing.T) {
	if got := item.GetClassification(item.VegasSpell10); got != item.ClassificationVegasSpell {
		t.Errorf("GetClassification(VegasSpell10) = %d, want %d", got, item.ClassificationVegasSpell)
	}
	if got := item.GetClassification(item.VegasSpell60); got != item.ClassificationVegasSpell {
		t.Errorf("GetClassification(VegasSpell60) = %d, want %d", got, item.ClassificationVegasSpell)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-constants && go test ./item/ -run 'Vegas' -v`
Expected: FAIL — `undefined: item.VegasSpell10` (compile error).

- [ ] **Step 3: Write the implementation**

Create `libs/atlas-constants/item/vegas_spell.go`:

```go
package item

// Vega's Spell cash consumables (Item.wz/Cash/0561.img.xml). Using one
// consumes the vega item together with an upgrade scroll whose natural
// success rate matches the variant exactly, applying the scroll at a boosted
// rate. The rate pairing itself (10→30, 60→90) is server policy and lives in
// atlas-consumables — these are wire/domain identities only.
const (
	// VegasSpell10 boosts a 10% scroll to 30%.
	VegasSpell10 = Id(5610000)

	// VegasSpell60 boosts a 60% scroll to 90%.
	VegasSpell60 = Id(5610001)

	// ClassificationVegasSpell is the cash-compartment classification
	// (item id / 10000) for Vega's Spell items.
	ClassificationVegasSpell = Classification(561)
)

// IsVegasSpell returns true if the item is a Vega's Spell variant.
func IsVegasSpell(id Id) bool {
	return Is(id, VegasSpell10, VegasSpell60)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd libs/atlas-constants && go test -race ./item/ -run 'Vegas' -v`
Expected: PASS (both tests).

- [ ] **Step 5: Full module gate + commit**

```bash
cd libs/atlas-constants && go test -race ./... && go vet ./... && cd ../..
git add libs/atlas-constants/item/vegas_spell.go libs/atlas-constants/item/vegas_spell_test.go
git commit -m "feat(constants): Vega's Spell item ids, classification, classifier (task-130)"
```

---

### Task 2: `ItemUseVegaScroll` serverbound codec

**Files:**
- Create: `libs/atlas-packet/cash/serverbound/item_use_vega_scroll.go`
- Create: `libs/atlas-packet/cash/serverbound/item_use_vega_scroll_test.go`

**Interfaces:**
- Consumes: `atlas-socket/request.Reader`, `atlas-socket/response.Writer` (same imports as `item_use_field_effect.go`).
- Produces (used by Tasks 4, 10): `type ItemUseVegaScroll` with `NewItemUseVegaScroll(equipTab uint32, equipSlot int32, scrollTab uint32, scrollSlot int32, flag uint32, updateTime uint32) *ItemUseVegaScroll`, getters `EquipTab() uint32`, `EquipSlot() int32`, `ScrollTab() uint32`, `ScrollSlot() int32`, `Flag() uint32`, `UpdateTime() uint32`, plus `Operation()`, `String()`, `Encode`, `Decode`.

Unlike `ItemUsePointReset`/`ItemUseFieldEffect` there is **no `updateTimeFirst` constructor flag**: the trailing updateTime is present on every version (design §2.1); the prefix-side gate is already handled by the outer `ItemUse` codec.

- [ ] **Step 1: Write the failing tests**

Create `libs/atlas-packet/cash/serverbound/item_use_vega_scroll_test.go`:

```go
package serverbound

import (
	"encoding/hex"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

func TestItemUseVegaScrollRoundTrip(t *testing.T) {
	input := NewItemUseVegaScroll(1, 5, 2, 7, 1, 305419896)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := ItemUseVegaScroll{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}

// TestItemUseVegaScrollBytes locks the six-int32 little-endian wire shape:
// equipTab(1) equipSlot(5) scrollTab(2) scrollSlot(7) flag(1)
// updateTime(0x12345678) — 24 bytes, version-independent (no gate in codec).
func TestItemUseVegaScrollBytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := NewItemUseVegaScroll(1, 5, 2, 7, 1, 0x12345678)
	want := "01000000" + "05000000" + "02000000" + "07000000" + "01000000" + "78563412"
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			got := hex.EncodeToString(input.Encode(l, ctx)(nil))
			if got != want {
				t.Errorf("bytes: got %s, want %s", got, want)
			}
		})
	}
}
```

Note: if `pt.RoundTrip`'s signature differs (compare with `item_use_field_effect_test.go` in the same package), match that file's call shape exactly — the fixture strings above are the load-bearing part.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd libs/atlas-packet && go test ./cash/serverbound/ -run 'ItemUseVegaScroll' -v`
Expected: FAIL — `undefined: NewItemUseVegaScroll` (compile error).

- [ ] **Step 3: Write the implementation**

Create `libs/atlas-packet/cash/serverbound/item_use_vega_scroll.go`:

```go
package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// ItemUseVegaScroll — the category-561 (Vega's Spell) sub-body of the
// USE_CASH_ITEM request. The packet is assembled and sent by the CUIVega
// dialog, not by CWvsContext::SendConsumeCashItemUseRequest directly (v83
// sub_82CBE2 LABEL_28; v95 CUIVega::OnButtonClicked 0x7bf4a0). The trailing
// updateTime is present on EVERY version regardless of the prefix's
// updateTimeFirst convention — v95 carries updateTime in the prefix AND here
// (both IDA-verified, task-130 design §2.1) — so Decode reads all six int32s
// unconditionally.
//
// packet-audit:fname CWvsContext::SendConsumeCashItemUseRequest#VegaScroll
type ItemUseVegaScroll struct {
	equipTab   uint32 // inventory tab index of the equip target; always 1 (equip inventory)
	equipSlot  int32  // positive = equip inventory; sign is passed through to the service
	scrollTab  uint32 // inventory tab index of the scroll; always 2 (use inventory)
	scrollSlot int32
	flag       uint32 // constant 1 on v83+v95 (v95 IDB names it m_nWhiteScrollUse but always writes 1); read and ignored
	updateTime uint32
}

func NewItemUseVegaScroll(equipTab uint32, equipSlot int32, scrollTab uint32, scrollSlot int32, flag uint32, updateTime uint32) *ItemUseVegaScroll {
	return &ItemUseVegaScroll{
		equipTab:   equipTab,
		equipSlot:  equipSlot,
		scrollTab:  scrollTab,
		scrollSlot: scrollSlot,
		flag:       flag,
		updateTime: updateTime,
	}
}

func (m ItemUseVegaScroll) EquipTab() uint32   { return m.equipTab }
func (m ItemUseVegaScroll) EquipSlot() int32   { return m.equipSlot }
func (m ItemUseVegaScroll) ScrollTab() uint32  { return m.scrollTab }
func (m ItemUseVegaScroll) ScrollSlot() int32  { return m.scrollSlot }
func (m ItemUseVegaScroll) Flag() uint32       { return m.flag }
func (m ItemUseVegaScroll) UpdateTime() uint32 { return m.updateTime }

func (m ItemUseVegaScroll) Operation() string { return "ItemUseVegaScroll" }

func (m ItemUseVegaScroll) String() string {
	return fmt.Sprintf("equipTab [%d] equipSlot [%d] scrollTab [%d] scrollSlot [%d] flag [%d] updateTime [%d]",
		m.equipTab, m.equipSlot, m.scrollTab, m.scrollSlot, m.flag, m.updateTime)
}

func (m ItemUseVegaScroll) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.equipTab)
		w.WriteInt(uint32(m.equipSlot))
		w.WriteInt(m.scrollTab)
		w.WriteInt(uint32(m.scrollSlot))
		w.WriteInt(m.flag)
		w.WriteInt(m.updateTime)
		return w.Bytes()
	}
}

func (m *ItemUseVegaScroll) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.equipTab = r.ReadUint32()
		m.equipSlot = int32(r.ReadUint32())
		m.scrollTab = r.ReadUint32()
		m.scrollSlot = int32(r.ReadUint32())
		m.flag = r.ReadUint32()
		m.updateTime = r.ReadUint32()
	}
}
```

(If the `response.Writer`/`request.Reader` in this package expose dedicated signed helpers — check how sibling codecs write int32 values — prefer those over the `uint32` casts, keeping the identical wire bytes.)

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd libs/atlas-packet && go test -race ./cash/serverbound/ -run 'ItemUseVegaScroll' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-packet/cash/serverbound/item_use_vega_scroll.go libs/atlas-packet/cash/serverbound/item_use_vega_scroll_test.go
git commit -m "feat(packet): ItemUseVegaScroll serverbound sub-body codec (task-130)"
```

---

### Task 3: `VegaScroll` clientbound writer with outcome-keyed operations

**Files:**
- Create: `libs/atlas-packet/cash/clientbound/vega_scroll.go`
- Create: `libs/atlas-packet/cash/clientbound/vega_scroll_test.go`

**Interfaces:**
- Consumes: `atlas_packet.WithResolvedCode` (`libs/atlas-packet/resolve.go:13`), `atlas-socket/packet.Encoder`.
- Produces (used by Tasks 4, 11, 12):
  - `const VegaScrollWriter = "VegaScroll"`
  - operations keys: `VegaScrollModeStartSuccess = "START_SUCCESS"`, `VegaScrollModeStartFailure = "START_FAILURE"`, `VegaScrollModeResultSuccess = "RESULT_SUCCESS"`, `VegaScrollModeResultFailure = "RESULT_FAILURE"`, `VegaScrollModeInvalid = "INVALID"`
  - structs `VegaScrollStart`, `VegaScrollResult`, `VegaScrollInvalid` (each one mode byte; constructors `NewVegaScrollStart(mode byte)`, `NewVegaScrollResult(mode byte)`, `NewVegaScrollInvalid(mode byte)`)
  - body funcs `VegaScrollStartBody(success bool)`, `VegaScrollResultBody(success bool)`, `VegaScrollInvalidBody()` each returning `func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte`

Rationale (design §2.3/§3.3): the mode values shifted +4 between v83 and v95 AND v95 selects its result popup from the START byte, so "the mode" is a per-version function of the outcome. Callers pass the outcome; every `WithResolvedCode` call site fixes its key as a named constant (uniformity policy — same shape as `storage/operation_body.go` and `party/clientbound/operation_body.go`).

- [ ] **Step 1: Write the failing tests**

Create `libs/atlas-packet/cash/clientbound/vega_scroll_test.go`:

```go
package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// vegaOpsV83 mirrors the gms_83 tenant template operations table (task-130
// design §2.3, IDA-verified CUIVega::OnVegaResult 0x82d8d5). Template JSON
// numbers decode as float64.
func vegaOpsV83() map[string]interface{} {
	return map[string]interface{}{
		"operations": map[string]interface{}{
			"START_SUCCESS":  float64(64),
			"START_FAILURE":  float64(64),
			"RESULT_SUCCESS": float64(65),
			"RESULT_FAILURE": float64(67),
			"INVALID":        float64(66),
		},
	}
}

func TestVegaScrollBodyResolution(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 83, 1)
	cases := []struct {
		name string
		body func() []byte
		want byte
	}{
		{"start success", func() []byte { return VegaScrollStartBody(true)(l, ctx)(vegaOpsV83()) }, 0x40},
		{"start failure", func() []byte { return VegaScrollStartBody(false)(l, ctx)(vegaOpsV83()) }, 0x40},
		{"result success", func() []byte { return VegaScrollResultBody(true)(l, ctx)(vegaOpsV83()) }, 0x41},
		{"result failure", func() []byte { return VegaScrollResultBody(false)(l, ctx)(vegaOpsV83()) }, 0x43},
		{"invalid", func() []byte { return VegaScrollInvalidBody()(l, ctx)(vegaOpsV83()) }, 0x42},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.body()
			if len(got) != 1 || got[0] != tc.want {
				t.Errorf("body bytes: got %v, want [%#x]", got, tc.want)
			}
		})
	}
}

// A missing operations table must fall back to 99 (ResolveCode contract) —
// this is the misconfigured-tenant canary, not a supported path.
func TestVegaScrollBodyMissingOperations(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 83, 1)
	got := VegaScrollInvalidBody()(l, ctx)(map[string]interface{}{})
	if len(got) != 1 || got[0] != 99 {
		t.Errorf("missing-operations fallback: got %v, want [99]", got)
	}
}

func TestVegaScrollRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewVegaScrollStart(0x40)
			output := VegaScrollStart{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd libs/atlas-packet && go test ./cash/clientbound/ -run 'VegaScroll' -v`
Expected: FAIL — `undefined: VegaScrollStartBody` (compile error).

- [ ] **Step 3: Write the implementation**

Create `libs/atlas-packet/cash/clientbound/vega_scroll.go`:

```go
package clientbound

import (
	"context"
	"fmt"

	atlas_packet "github.com/Chronicle20/atlas/libs/atlas-packet"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// VegaScroll — CUIVega::OnVegaResult (v83 0x82d8d5 via OnPacket 0x82d8bf
// opcode 0x166; v95 0x7bf7b0 via OnPacket 0x7c0680 opcode 0x1AD). Body is a
// single mode byte. The accepted values are version-shifted (+4 from v83 to
// v95) AND v95 selects its result popup from the START byte while v83 renders
// EffectSuccess/EffectFail from the RESULT byte — so the operations keys are
// outcome-keyed and the byte is resolved from the tenant operations table at
// encode time (task-130 design §2.2–§2.3). On v83 both START keys collapse to
// 0x40 harmlessly. Any unconfigured key resolves to 99, which both clients
// route to the safe "This item cannot be used." notice arm (no crash arm
// exists in either version).
const VegaScrollWriter = "VegaScroll"

const (
	VegaScrollModeStartSuccess  = "START_SUCCESS"
	VegaScrollModeStartFailure  = "START_FAILURE"
	VegaScrollModeResultSuccess = "RESULT_SUCCESS"
	VegaScrollModeResultFailure = "RESULT_FAILURE"
	VegaScrollModeInvalid       = "INVALID"
)

// VegaScrollStart — the start-animation arm (twinkle sound + gauge).
//
// packet-audit:fname CUIVega::OnVegaResult#Start
type VegaScrollStart struct {
	mode byte
}

func NewVegaScrollStart(mode byte) VegaScrollStart { return VegaScrollStart{mode: mode} }

func (m VegaScrollStart) Mode() byte        { return m.mode }
func (m VegaScrollStart) Operation() string { return VegaScrollWriter }
func (m VegaScrollStart) String() string    { return fmt.Sprintf("vega scroll start mode [%d]", m.mode) }

func (m VegaScrollStart) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *VegaScrollStart) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// VegaScrollResult — the latched pass/fail arm, displayed after the
// animation completes on the client's own clock.
//
// packet-audit:fname CUIVega::OnVegaResult#Result
type VegaScrollResult struct {
	mode byte
}

func NewVegaScrollResult(mode byte) VegaScrollResult { return VegaScrollResult{mode: mode} }

func (m VegaScrollResult) Mode() byte        { return m.mode }
func (m VegaScrollResult) Operation() string { return VegaScrollWriter }
func (m VegaScrollResult) String() string    { return fmt.Sprintf("vega scroll result mode [%d]", m.mode) }

func (m VegaScrollResult) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *VegaScrollResult) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// VegaScrollInvalid — the else-arm: the client shows "This item cannot be
// used." and closes the dialog. REQUIRED on rejection (not optional): after
// sending the request the client sets m_bRequestSent and disables the dialog;
// a rejection that sent nothing would leave it wedged (design §2.3).
//
// packet-audit:fname CUIVega::OnVegaResult#Invalid
type VegaScrollInvalid struct {
	mode byte
}

func NewVegaScrollInvalid(mode byte) VegaScrollInvalid { return VegaScrollInvalid{mode: mode} }

func (m VegaScrollInvalid) Mode() byte        { return m.mode }
func (m VegaScrollInvalid) Operation() string { return VegaScrollWriter }
func (m VegaScrollInvalid) String() string    { return fmt.Sprintf("vega scroll invalid mode [%d]", m.mode) }

func (m VegaScrollInvalid) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *VegaScrollInvalid) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// VegaScrollStartBody resolves the outcome-keyed START mode from the tenant
// operations table. The server resolves the outcome before sending (immediate
// resolution, no 3s timer), so the start byte can be outcome-selected; v83
// collapses both keys to the same byte.
func VegaScrollStartBody(success bool) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	if success {
		return atlas_packet.WithResolvedCode("operations", VegaScrollModeStartSuccess, func(mode byte) packet.Encoder {
			return NewVegaScrollStart(mode)
		})
	}
	return atlas_packet.WithResolvedCode("operations", VegaScrollModeStartFailure, func(mode byte) packet.Encoder {
		return NewVegaScrollStart(mode)
	})
}

func VegaScrollResultBody(success bool) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	if success {
		return atlas_packet.WithResolvedCode("operations", VegaScrollModeResultSuccess, func(mode byte) packet.Encoder {
			return NewVegaScrollResult(mode)
		})
	}
	return atlas_packet.WithResolvedCode("operations", VegaScrollModeResultFailure, func(mode byte) packet.Encoder {
		return NewVegaScrollResult(mode)
	})
}

func VegaScrollInvalidBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", VegaScrollModeInvalid, func(mode byte) packet.Encoder {
		return NewVegaScrollInvalid(mode)
	})
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd libs/atlas-packet && go test -race ./cash/clientbound/ -run 'VegaScroll' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-packet/cash/clientbound/vega_scroll.go libs/atlas-packet/cash/clientbound/vega_scroll_test.go
git commit -m "feat(packet): VegaScroll clientbound writer with outcome-keyed operations (task-130)"
```

---

### Task 4: IDA verification campaign — fixtures, evidence, registry, matrix

**Files:**
- Modify: `tools/packet-audit/cmd/run.go` (`candidatesFromFName` — splice, see below)
- Modify: `libs/atlas-packet/cash/serverbound/item_use_vega_scroll_test.go` (per-version verify markers; codec ONLY if IDA contradicts §2.1)
- Modify: `libs/atlas-packet/cash/clientbound/vega_scroll_test.go` (per-version exact-byte fixtures + verify markers)
- Create: `docs/packets/audits/<version>/VegaScroll.{md,json}` per wired version; serverbound evidence spliced into the shared USE_CASH_ITEM audit artifacts
- Modify: `docs/packets/registry/gms_v83.yaml`, `gms_v95.yaml` (+ `gms_v87.yaml`/`gms_v84.yaml`/`jms_v185.yaml` as verified) — VEGA_SCROLL row promotion per the playbook
- Modify: `docs/packets/audits/STATUS.md` / `status.json` (regenerated by the tool)

**Interfaces:**
- Consumes: Task 2 codec, Task 3 writer; `docs/packets/audits/VERIFYING_A_PACKET.md` (the governing playbook — follow it exactly, including its evidence/REPORT/registry steps); ida-pro-mcp instances or checked-in exports under `docs/packets/ida-exports/`.
- Produces: verified per-version fixtures and the pinned v95 operations values that Task 12's templates copy verbatim.

**Rules for this task (non-negotiable):**
- Before any IDA read: `mcp__ida-pro__list_instances` and match the **binary name** (as of plan time the loaded set has v83-dump on 13342 and v95 on 13341; NO v84/v87/jms). For a version with no live IDB and no usable export: **STOP and report BLOCKED** for that version — never substitute an fname, guess an opcode, or fake evidence. Absence of wiring is the safe failure; a wrong opcode can crash the client (design §2.5).
- gms_v92: **parked** (design §2.6). No fixture, no registry row, no template entry. Documented in Task 12's deployment note.
- Task-126 coordination: if task-126's `candidatesFromFName` case for `CWvsContext::SendConsumeCashItemUseRequest` already exists, **append** the vega candidate to its list; if the shared USE_CASH_ITEM serverbound audit REPORT already exists, **splice** the vega arm in (export splice rule) — never overwrite.

- [ ] **Step 1: Splice the `candidatesFromFName` cases**

In `tools/packet-audit/cmd/run.go`, in the serverbound CWvsContext-senders block (near `CWvsContext::SendUpgradeItemUseRequest`, ~line 1831): if a case for `"CWvsContext::SendConsumeCashItemUseRequest", "CItemSpeakerDlg::_SendConsumeCashItemUseRequest"` already exists (task-126), append `{name: "ItemUseVegaScroll", dir: csvpkg.DirServerbound, pkg: "cash"}` to its returned slice; otherwise add the case returning just the vega candidate. In the clientbound section, add:

```go
	case "CUIVega::OnVegaResult":
		return []candidate{{name: "VegaScroll", dir: csvpkg.DirClientbound, pkg: "cash"}}
```

(Match the exact `candidate` literal shape used by neighboring cases.)

Run: `cd tools/packet-audit && go build ./... && go test ./...`
Expected: clean.

- [ ] **Step 2: Verify gms_v83 (serverbound + clientbound)** — follow `docs/packets/audits/VERIFYING_A_PACKET.md` end-to-end. Serverbound: decompile the v83 CUIVega send path (`sub_82CBE2`, dialog construction via the case-68 arm of `0xa0a63f`) to confirm the §2.1 read order; add the marker `// packet-audit:verify packet=cash/serverbound/ItemUseVegaScroll version=gms_v83 ida=0x82cbe2` above `TestItemUseVegaScrollBytes`; produce/splice the serverbound evidence + REPORT under the shared USE_CASH_ITEM fname. Clientbound: decompile `CUIVega::OnVegaResult` 0x82d8d5 + `OnPacket` 0x82d8bf (opcode 0x166); add per-mode exact-byte fixtures to `vega_scroll_test.go` (they exist from Task 3 — add the marker `// packet-audit:verify packet=cash/clientbound/VegaScroll version=gms_v83 ida=0x82d8d5`); produce `docs/packets/audits/gms_v83/VegaScroll.{md,json}`; promote the registry row per the playbook.

- [ ] **Step 3: Verify gms_v95** — same procedure: serverbound `CUIVega::OnButtonClicked` 0x7bf4a0 (marker ida=0x7bf4a0); clientbound `CUIVega::OnVegaResult` 0x7bf7b0 / `OnPacket` 0x7c0680 (opcode 0x1AD). **Pin the START pairing**: read v95 string-pool templates 5417 (popup type 1, start 0x44) and 5418 (popup type 2, start 0x49) via `CUIVega::Draw` 0x7c1dd0 to determine which popup is the success one; record the pinned `START_SUCCESS`/`START_FAILURE` values in the audit REPORT and add a v95 fixture test to `vega_scroll_test.go` asserting the resolved bytes against a v95 operations map (mirroring `vegaOpsV83`, e.g. `vegaOpsV95`). If the pairing is the reverse of the Global Constraints hypothesis, the fixture + Task 12 template values swap 68/73 — nothing else changes.

- [ ] **Step 4: Verify gms_v87** — registry says opcode 0x17B (csv-import). If a v87 IDB/export is available: confirm the serverbound sub-body, the VEGA_SCROLL opcode, and the mode values (they are version-dependent — design §2.2); fixtures + markers + REPORT + registry promotion as above; record the v87 operations values for Task 12. If not available: **report BLOCKED for v87** and continue; v87 gets no template entry in Task 12.

- [ ] **Step 5: Verify gms_v84** — the registry's 0x166 is a suspected stale csv carryover in the +2→+10 shifted region (design §2.5). Verify against a v84 IDB/export (or `discover-ops` per the playbook) before trusting ANY v84 value. Verified → fixtures + markers + REPORT + registry correction/promotion + record operations values. Unavailable → **BLOCKED for v84**, no template entry.

- [ ] **Step 6: Verify jms_v185** — registry 0x183 (csv-import). Same procedure; note the jms audit dir is `docs/packets/audits/jms_v185` — pass `--audit-dir` explicitly to triage/decompose subcommands (default name mismatch silently reports 0/0/0/0). Unavailable → **BLOCKED for jms**, no template entry.

- [ ] **Step 7: Regenerate matrix + gate**

```bash
cd libs/atlas-packet && go test -race ./cash/... && cd ../..
go run ./tools/packet-audit matrix --check
```
Expected: tests PASS; `matrix --check` exit 0 with VegaScroll/ItemUseVegaScroll cells promoted for each verified version.

- [ ] **Step 8: Commit**

```bash
git add tools/packet-audit/cmd/run.go libs/atlas-packet/cash/ docs/packets/
git commit -m "feat(packet): IDA-verify VegaScroll + ItemUseVegaScroll fixtures, evidence, registry (task-130)"
```

---

### Task 5: atlas-consumables — Kafka contract, event producer, rate policy

**Files:**
- Modify: `services/atlas-consumables/atlas.com/consumables/kafka/message/consumable/kafka.go`
- Modify: `services/atlas-consumables/atlas.com/consumables/consumable/producer.go`
- Create: `services/atlas-consumables/atlas.com/consumables/consumable/vega.go`
- Create: `services/atlas-consumables/atlas.com/consumables/consumable/vega_test.go`

**Interfaces:**
- Consumes: existing `Command[E]`/`Event[E]` envelopes, `slot.Position`, `item.Id`, `producer.SingleMessageProvider`.
- Produces (used by Tasks 7, 8; mirrored by Task 9):
  - `CommandRequestVegaScroll = "REQUEST_VEGA_SCROLL"`, `type RequestVegaScrollBody`
  - `EventTypeVegaScroll = "VEGA_SCROLL"`, `type VegaScrollBody { Success, Cursed bool }`
  - `ErrorTypeVegaInvalid = "VEGA_INVALID"`
  - `VegaScrollEventProvider(characterId character.Id) func(success bool, cursed bool) model.Provider[[]kafka.Message]`
  - `func vegaRates(id item2.Id) (required uint32, boosted uint32, ok bool)`

- [ ] **Step 1: Write the failing test for the rate policy**

Create `services/atlas-consumables/atlas.com/consumables/consumable/vega_test.go`:

```go
package consumable

import (
	"testing"

	item2 "github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

func TestVegaRates(t *testing.T) {
	cases := []struct {
		name         string
		id           item2.Id
		wantRequired uint32
		wantBoosted  uint32
		wantOk       bool
	}{
		{"Vega's Spell 10", item2.VegasSpell10, 10, 30, true},
		{"Vega's Spell 60", item2.VegasSpell60, 60, 90, true},
		{"non-vega cash item", item2.Id(5610002), 0, 0, false},
		{"scroll id", item2.ChaosScrollSixtyPercent, 0, 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			required, boosted, ok := vegaRates(tc.id)
			if required != tc.wantRequired || boosted != tc.wantBoosted || ok != tc.wantOk {
				t.Errorf("vegaRates(%d) = (%d, %d, %t), want (%d, %d, %t)",
					tc.id, required, boosted, ok, tc.wantRequired, tc.wantBoosted, tc.wantOk)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-consumables/atlas.com/consumables && go test ./consumable/ -run TestVegaRates -v`
Expected: FAIL — `undefined: vegaRates` (compile error).

- [ ] **Step 3: Write the implementation**

Create `services/atlas-consumables/atlas.com/consumables/consumable/vega.go`:

```go
package consumable

import (
	item2 "github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

// vegaRates returns the natural scroll success rate a Vega's Spell requires
// (exact match only) and the boosted rate it applies. This is server policy
// (PRD FR-4.1), not WZ data — the Item.wz entries for 0561 carry only info
// nodes. Non-vega ids return ok=false.
func vegaRates(id item2.Id) (required uint32, boosted uint32, ok bool) {
	switch id {
	case item2.VegasSpell10:
		return 10, 30, true
	case item2.VegasSpell60:
		return 60, 90, true
	}
	return 0, 0, false
}
```

In `kafka/message/consumable/kafka.go`, extend the command const block (after `CommandRequestScroll`):

```go
	CommandRequestVegaScroll = "REQUEST_VEGA_SCROLL"
```

after `RequestScrollBody` add:

```go
// RequestVegaScrollBody asks the service to apply the scroll at ScrollSlot to
// the equip at EquipSlot at the vega-boosted rate, consuming the vega cash
// item at VegaSlot together with the scroll. EquipSlot sign convention:
// positive = equip inventory (the vega dialog's targets), negative = equipped.
type RequestVegaScrollBody struct {
	VegaSlot   slot.Position `json:"vegaSlot"`   // cash compartment
	VegaItemId item.Id       `json:"vegaItemId"` // re-validated against slot contents
	ScrollSlot slot.Position `json:"scrollSlot"` // use compartment
	EquipSlot  slot.Position `json:"equipSlot"`
}
```

extend the event const block (after `EventTypeScroll`) and error types:

```go
	EventTypeVegaScroll = "VEGA_SCROLL"
```

```go
	ErrorTypeVegaInvalid = "VEGA_INVALID"
```

after `ScrollBody` add:

```go
// VegaScrollBody carries the resolved vega scroll outcome. Distinct from
// ScrollBody so the channel can emit the CUIVega dialog packets instead of
// the plain map broadcast; whiteScroll/legendarySpirit are always false on
// the vega path and therefore not carried.
type VegaScrollBody struct {
	Success bool `json:"success"`
	Cursed  bool `json:"cursed"`
}
```

In `consumable/producer.go`, after `ScrollEventProvider`:

```go
func VegaScrollEventProvider(characterId character.Id) func(success bool, cursed bool) model.Provider[[]kafka.Message] {
	return func(success bool, cursed bool) model.Provider[[]kafka.Message] {
		key := producer.CreateKey(int(characterId))
		value := &consumable.Event[consumable.VegaScrollBody]{
			CharacterId: characterId,
			Type:        consumable.EventTypeVegaScroll,
			Body: consumable.VegaScrollBody{
				Success: success,
				Cursed:  cursed,
			},
		}
		return producer.SingleMessageProvider(key, value)
	}
}
```

- [ ] **Step 4: Run tests + build to verify**

Run: `cd services/atlas-consumables/atlas.com/consumables && go test -race ./consumable/ -run TestVegaRates -v && go build ./...`
Expected: PASS; clean build.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-consumables/atlas.com/consumables/kafka/message/consumable/kafka.go services/atlas-consumables/atlas.com/consumables/consumable/producer.go services/atlas-consumables/atlas.com/consumables/consumable/vega.go services/atlas-consumables/atlas.com/consumables/consumable/vega_test.go
git commit -m "feat(consumables): REQUEST_VEGA_SCROLL contract, VEGA_SCROLL event, vega rate policy (task-130)"
```

---

### Task 6: atlas-consumables — extract the shared scroll-application core

**Files:**
- Modify: `services/atlas-consumables/atlas.com/consumables/consumable/processor.go` (`ConsumeScroll`, lines ~606-738)
- Modify: `services/atlas-consumables/atlas.com/consumables/consumable/processor_test.go` (new `buildScrollChanges` tests)

**Interfaces:**
- Consumes: existing `applyChaos`, `equipable.Change` constructors, `consumable3.Model` getters, `asset.Model`.
- Produces (used by Task 7):
  - `type scrollOutcome struct { success bool; cursed bool }`
  - `func buildScrollChanges(ci consumable3.Model, equip asset.Model, scrollId item2.Id, isSuccess bool, whiteScroll bool) ([]equipable.Change, error)`
  - `func applyScrollCore(l logrus.FieldLogger, ctx context.Context, transactionId uuid.UUID, characterId uint32, ci consumable3.Model, scrollItem *asset.Model, equip *asset.Model, successProb uint32, whiteScroll bool) (scrollOutcome, error)`

**Behavioral contract:** `ConsumeScroll` stays observably identical — same roll order (success roll, then chaos rolls on chaos-success, then curse roll on failure), same log lines, same error wrapping via `p.ConsumeError`, same consume/curse/emit sequence. This refactor also **deletes the `// TODO consume vega scroll` at processor.go:641** (the successProb line moves into the caller as `ci.SuccessRate()`).

- [ ] **Step 1: Write the failing tests**

Append to `processor_test.go` (reusing the existing helpers `createTestEquipableAsset`, `createTestScrollAsset`, and the `consumable3.RestModel` + `Extract` construction pattern from `makeCureModel`):

```go
func makeScrollModel(t *testing.T, success uint32, cursed uint32, incSTR uint32) consumable3.Model {
	t.Helper()
	rm := consumable3.RestModel{Success: success, Cursed: cursed, IncreaseSTR: incSTR}
	m, err := consumable3.Extract(rm)
	if err != nil {
		t.Fatalf("extract failed: %v", err)
	}
	return m
}

func TestBuildScrollChanges_RegularSuccess(t *testing.T) {
	ci := makeScrollModel(t, 60, 0, 5)
	equip := createTestEquipableAsset(1302000, 7, 0)
	changes, err := buildScrollChanges(ci, equip, item2.Id(2043001), true, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 15 stat adds + AddSlots(-1) + AddLevel(1) = 17 changes.
	if len(changes) != 17 {
		t.Errorf("expected 17 changes for regular success, got %d", len(changes))
	}
}

func TestBuildScrollChanges_RegularFailureConsumesSlot(t *testing.T) {
	ci := makeScrollModel(t, 60, 0, 5)
	equip := createTestEquipableAsset(1302000, 7, 0)
	changes, err := buildScrollChanges(ci, equip, item2.Id(2043001), false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(changes) != 1 {
		t.Errorf("expected 1 change (slot decrement) on plain failure, got %d", len(changes))
	}
}

func TestBuildScrollChanges_FailureWithWhiteScrollPreservesSlot(t *testing.T) {
	ci := makeScrollModel(t, 60, 0, 5)
	equip := createTestEquipableAsset(1302000, 7, 0)
	changes, err := buildScrollChanges(ci, equip, item2.Id(2043001), false, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(changes) != 0 {
		t.Errorf("expected 0 changes on white-scroll failure, got %d", len(changes))
	}
}

func TestBuildScrollChanges_SpikeSuccess(t *testing.T) {
	ci := makeScrollModel(t, 10, 0, 0)
	equip := createTestEquipableAsset(1072000, 5, 0)
	changes, err := buildScrollChanges(ci, equip, item2.ScrollForSpikesOnShoesTenPercent, true, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(changes) != 1 {
		t.Errorf("expected 1 change (SetSpike) for spike success, got %d", len(changes))
	}
}

func TestBuildScrollChanges_SpikeFailureNoSlotLoss(t *testing.T) {
	ci := makeScrollModel(t, 10, 0, 0)
	equip := createTestEquipableAsset(1072000, 5, 0)
	changes, err := buildScrollChanges(ci, equip, item2.ScrollForSpikesOnShoesTenPercent, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(changes) != 0 {
		t.Errorf("expected 0 changes for spike failure, got %d", len(changes))
	}
}

func TestBuildScrollChanges_CleanSlateSuccessAddsSlot(t *testing.T) {
	ci := makeScrollModel(t, 1, 0, 0)
	equip := createTestEquipableAsset(1302000, 0, 2)
	changes, err := buildScrollChanges(ci, equip, item2.CleanSlateOnePercent, true, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(changes) != 1 {
		t.Errorf("expected 1 change (AddSlots) for clean slate success, got %d", len(changes))
	}
}

func TestBuildScrollChanges_ChaosSuccess(t *testing.T) {
	ci := makeScrollModel(t, 60, 0, 0)
	equip := asset.NewBuilder(uuid.New(), 1302000).
		SetId(1).
		SetSlots(7).
		SetStrength(10).
		SetDexterity(10).
		Build()
	changes, err := buildScrollChanges(ci, equip, item2.ChaosScrollSixtyPercent, true, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 2 chaos stat changes + AddSlots(-1) + AddLevel(1) = 4.
	if len(changes) != 4 {
		t.Errorf("expected 4 changes for chaos success on 2 non-zero stats, got %d", len(changes))
	}
}
```

(Add the missing imports the file doesn't already have; it already imports `asset`, `consumable3`, `uuid`, and `item2` aliases — mirror the existing import block.)

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-consumables/atlas.com/consumables && go test ./consumable/ -run TestBuildScrollChanges -v`
Expected: FAIL — `undefined: buildScrollChanges` (compile error).

- [ ] **Step 3: Extract the core**

In `processor.go`, add above `ConsumeScroll`:

```go
type scrollOutcome struct {
	success bool
	cursed  bool
}

// buildScrollChanges assembles the equip change-set for a scroll application
// outcome. Extracted verbatim from ConsumeScroll (task-130) so the vega path
// shares it; behavior is locked by the TestBuildScrollChanges_* table.
func buildScrollChanges(ci consumable3.Model, equip asset.Model, scrollId item2.Id, isSuccess bool, whiteScroll bool) ([]equipable.Change, error) {
	changes := make([]equipable.Change, 0)
	if isSuccess {
		if item2.IsScrollSpikes(scrollId) {
			changes = append(changes, equipable.SetSpike())
		} else if item2.IsScrollColdProtection(scrollId) {
			changes = append(changes, equipable.SetCold())
		} else if item2.IsScrollCleanSlate(scrollId) {
			changes = append(changes, equipable.AddSlots(1))
		} else if item2.IsChaosScroll(scrollId) {
			ccs, err := applyChaos(equip)
			if err != nil {
				return nil, err
			}
			changes = append(changes, ccs...)
			changes = append(changes,
				equipable.AddSlots(-1),
				equipable.AddLevel(1))
		} else {
			changes = append(changes,
				equipable.AddStrength(int16(ci.StrengthIncrease())),
				equipable.AddDexterity(int16(ci.DexterityIncrease())),
				equipable.AddIntelligence(int16(ci.IntelligenceIncrease())),
				equipable.AddLuck(int16(ci.LuckIncrease())),
				equipable.AddHp(int16(ci.MaxHPIncrease())),
				equipable.AddMp(int16(ci.MaxMPIncrease())),
				equipable.AddWeaponAttack(int16(ci.WeaponAttackIncrease())),
				equipable.AddMagicAttack(int16(ci.MagicAttackIncrease())),
				equipable.AddWeaponDefense(int16(ci.WeaponDefenseIncrease())),
				equipable.AddMagicDefense(int16(ci.MagicDefenseIncrease())),
				equipable.AddAccuracy(int16(ci.AccuracyIncrease())),
				equipable.AddAvoidability(int16(ci.AvoidabilityIncrease())),
				equipable.AddHands(int16(ci.HandsIncrease())),
				equipable.AddSpeed(int16(ci.SpeedIncrease())),
				equipable.AddJump(int16(ci.JumpIncrease())),
				equipable.AddSlots(-1),
				equipable.AddLevel(1))
		}
	} else {
		if !item2.IsScrollSpikes(scrollId) && !item2.IsScrollColdProtection(scrollId) && !item2.IsScrollCleanSlate(scrollId) && !whiteScroll {
			changes = append(changes, equipable.AddSlots(-1))
		}
	}
	return changes, nil
}

// applyScrollCore is the shared middle of ConsumeScroll and ConsumeVegaScroll:
// the success roll at successProb, change-set assembly, the curse roll, and
// ChangeStat. Reservation commits, curse destruction, and result emission stay
// with the caller (they differ between the normal and vega paths). successProb
// is the roll threshold — the scroll's natural rate on the normal path, the
// vega-boosted rate on the vega path. The pre-existing `roll <= prob`
// comparator is deliberately inherited (PRD non-goal: no scroll-math changes).
func applyScrollCore(l logrus.FieldLogger, ctx context.Context, transactionId uuid.UUID, characterId uint32, ci consumable3.Model, scrollItem *asset.Model, equip *asset.Model, successProb uint32, whiteScroll bool) (scrollOutcome, error) {
	ep := equipable.NewProcessor(l, ctx)

	successRoll := rand.Int31n(100)
	isSuccess := successRoll <= int32(successProb)

	passFail := "failed"
	if isSuccess {
		passFail = "passed"
	}
	l.Debugf("Character [%d] has [%s] scroll [%d]. Rolled [%d]. Needed [%d].", characterId, passFail, scrollItem.TemplateId(), successRoll, successProb)

	changes, err := buildScrollChanges(ci, *equip, item2.Id(scrollItem.TemplateId()), isSuccess, whiteScroll)
	if err != nil {
		return scrollOutcome{}, err
	}

	isCursed := false
	if !isSuccess && rand.Int31n(100) <= int32(ci.CursedRate()) {
		l.Debugf("Character [%d] item has been cursed.", characterId)
		isCursed = true
	}

	if len(changes) > 0 {
		l.Debugf("Applying [%d] changes to character [%d] item [%d].", len(changes), characterId, equip.TemplateId())
		if err := ep.ChangeStat(characterId, transactionId, *equip, changes...); err != nil {
			return scrollOutcome{}, err
		}
	}
	return scrollOutcome{success: isSuccess, cursed: isCursed}, nil
}
```

Then rewrite the middle of `ConsumeScroll`: keep everything through the `ValidateScrollUse` check and the `ci` lookup unchanged, **delete the `// TODO consume vega scroll` comment and lines 642-710** (successProb/roll/changes/ChangeStat), replacing them with:

```go
			outcome, err := applyScrollCore(l, ctx, transactionId, characterId, ci, scrollItem, sm.Equipable, ci.SuccessRate(), whiteScroll)
			if err != nil {
				return p.ConsumeError(characterId, transactionId, inventory2.TypeValueUse, scrollItem.Slot(), err)
			}
```

and update the tail to use `outcome.success` / `outcome.cursed` in place of `isSuccess` / `isCursed` (consume scroll → consume white scroll → curse destroy → `PassScroll`/`FailScroll` emission — sequence unchanged).

Note: the original ConsumeScroll curse-roll sits inside the failure branch of the change assembly; `applyScrollCore` performs it after `buildScrollChanges`. The observable behavior is identical — the failure branch of `buildScrollChanges` makes no `rand` calls, so the rand sequence (success roll → chaos rolls if chaos-success → curse roll if failure) is preserved exactly.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-consumables/atlas.com/consumables && go test -race ./consumable/ -v`
Expected: PASS — the new TestBuildScrollChanges_* AND every pre-existing test.

- [ ] **Step 5: Verify the TODO is gone and commit**

```bash
grep -rn "consume vega scroll" services/atlas-consumables && echo "FAIL: TODO still present" || echo OK
git add services/atlas-consumables/atlas.com/consumables/consumable/processor.go services/atlas-consumables/atlas.com/consumables/consumable/processor_test.go
git commit -m "refactor(consumables): extract applyScrollCore/buildScrollChanges from ConsumeScroll (task-130)"
```
Expected: grep finds nothing (prints OK).

---

### Task 7: atlas-consumables — vega request, chained reservations, consume

**Files:**
- Modify: `services/atlas-consumables/atlas.com/consumables/consumable/vega.go`
- Modify: `services/atlas-consumables/atlas.com/consumables/consumable/vega_test.go`

**Interfaces:**
- Consumes: Task 1 (`item2.IsVegasSpell`), Task 5 (`vegaRates`, `VegaScrollEventProvider`, `ErrorTypeVegaInvalid`), Task 6 (`applyScrollCore`), existing `once.ReservationValidator` (`kafka/once/compartment/once.go:12`), `compartment.Consume` (`compartment/processor.go:37`), `compartment.Reserves`, `cpp.RequestReserve/ConsumeItem/DestroyItem/CancelItemReservation`, `p.ValidateScrollUse`, `consumer.GetManager().RegisterHandler`, `message.OneTimeConfig`.
- Produces (used by Task 8):
  - `func (p *Processor) RequestVegaScroll(characterId uint32, vegaSlot int16, vegaItemId item2.Id, scrollSlot int16, equipSlot int16) error`
  - `func ConsumeVegaScroll(transactionId uuid.UUID, characterId uint32, vegaItem *asset.Model, scrollItem *asset.Model, equipSlot int16, boostedProb uint32) ItemConsumer`
  - internal: `resolveVegaEquip`, `ReserveVegaScrollStage`, `vegaReservation`, `(p *Processor) VegaScrollError`

**Chain shape (design §3.2):** register once-listener B (txn + scroll templateId → `ConsumeVegaScroll`), register once-listener A (txn + vega templateId → issue the USE scroll reservation), then kick off with the CASH vega reservation. Item-id keying is collision-free (561xxxx vs 20xxxxx). Reservation-side failures emit no event; the earlier reservation TTL-expires in ~30s (design §2.9) — accepted, same envelope as the existing scroll flow.

- [ ] **Step 1: Write the failing tests**

Append to `vega_test.go`:

```go
func TestResolveVegaEquip_PositiveSlotFromEquipInventory(t *testing.T) {
	equip := asset.NewBuilder(uuid.New(), 1302000).SetId(1).SetSlot(3).SetSlots(7).Build()
	comp := compartment.NewBuilder(uuid.New(), 1, inventory2.TypeValueEquip, 96).AddAsset(equip).Build()
	inv := inventory.NewBuilder(1).SetEquipable(comp).Build()
	c := character.NewModelBuilder().SetId(1).SetInventory(inv).SetEquipment(equipment.NewModel()).Build()

	got, err := resolveVegaEquip(c, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TemplateId() != 1302000 {
		t.Errorf("resolved template: got %d, want 1302000", got.TemplateId())
	}
}

func TestResolveVegaEquip_PositiveSlotMissing(t *testing.T) {
	comp := compartment.NewBuilder(uuid.New(), 1, inventory2.TypeValueEquip, 96).Build()
	inv := inventory.NewBuilder(1).SetEquipable(comp).Build()
	c := character.NewModelBuilder().SetId(1).SetInventory(inv).SetEquipment(equipment.NewModel()).Build()

	if _, err := resolveVegaEquip(c, 3); err == nil {
		t.Error("expected error for empty slot, got nil")
	}
}

func TestResolveVegaEquip_NegativeSlotFromEquipped(t *testing.T) {
	weapon := asset.NewBuilder(uuid.New(), 1302000).SetId(1).SetSlots(7).Build()
	eq := equipment.NewModel()
	s, err := slot.GetSlotByPosition(slot.Position(-11)) // weapon slot
	if err != nil {
		t.Fatalf("slot lookup: %v", err)
	}
	sm, _ := eq.Get(s.Type)
	sm.Equipable = &weapon
	eq.Set(s.Type, sm)
	c := character.NewModelBuilder().SetId(1).SetEquipment(eq).Build()

	got, err := resolveVegaEquip(c, -11)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TemplateId() != 1302000 {
		t.Errorf("resolved template: got %d, want 1302000", got.TemplateId())
	}
}

func TestResolveVegaEquip_NegativeSlotEmpty(t *testing.T) {
	c := character.NewModelBuilder().SetId(1).SetEquipment(equipment.NewModel()).Build()
	if _, err := resolveVegaEquip(c, -11); err == nil {
		t.Error("expected error for empty equipped slot, got nil")
	}
}
```

Imports to add in `vega_test.go`: `atlas-consumables/asset`, `atlas-consumables/compartment`, `atlas-consumables/inventory`, `atlas-consumables/equipment`, `atlas-consumables/character`, `github.com/Chronicle20/atlas/libs/atlas-constants/inventory` (as `inventory2`), `github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot`, `github.com/google/uuid`. Adjust builder-method spellings to the actual APIs (`asset.ModelBuilder.SetSlot` — verify against `asset/builder.go`; the weapon slot position for `slot.GetSlotByPosition` must be a valid position from `libs/atlas-constants/inventory/slot` — pick one from `slot.Slots`).

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-consumables/atlas.com/consumables && go test ./consumable/ -run TestResolveVegaEquip -v`
Expected: FAIL — `undefined: resolveVegaEquip` (compile error).

- [ ] **Step 3: Write the implementation**

Append to `vega.go` (add the imports it needs — mirror `processor.go`'s aliases):

```go
// vegaReservation identifies one reservation the vega chain may need to roll
// back: compartment type + slot.
type vegaReservation struct {
	inventoryType inventory2.Type
	slot          int16
}

// VegaScrollError cancels any reservations already made on the vega path and
// emits the VEGA_INVALID error event. The channel answers with the VEGA
// INVALID packet + enable-actions; the client shows its own "This item cannot
// be used." notice and closes the dialog (required — after sending, the
// client is excl-request-blocked and a silent rejection would wedge it).
func (p *Processor) VegaScrollError(characterId uint32, transactionId uuid.UUID, reservations []vegaReservation, err error) error {
	p.l.Debugf("Character [%d] unable to vega scroll due to error: [%v]", characterId, err)
	for _, r := range reservations {
		if cErr := p.cpp.CancelItemReservation(characterId, r.inventoryType, transactionId, r.slot); cErr != nil {
			p.l.WithError(cErr).Errorf("Unable to cancel item reservation at inventory [%d] slot [%d] for character [%d] as part of transaction [%s].", r.inventoryType, r.slot, characterId, transactionId)
		}
	}
	if cErr := producer.ProviderImpl(p.l)(p.ctx)(consumable.EnvEventTopic)(ErrorEventProvider(ts.Id(characterId), consumable.ErrorTypeVegaInvalid)); cErr != nil {
		p.l.WithError(cErr).Errorf("Unable to issue vega error event. Character [%d] likely going to be stuck.", characterId)
	}
	return err
}

// resolveVegaEquip resolves the vega dialog's equip target. The dialog's
// equips come from the equip INVENTORY (positive slots — the drop handler
// stores the drag-source position; design §2.7), while the classic scroll
// path only addresses EQUIPPED items (negative positions). Negative slots are
// honored defensively with the classic resolution.
func resolveVegaEquip(c character.Model, equipSlot int16) (*asset.Model, error) {
	if equipSlot < 0 {
		s, err := slot.GetSlotByPosition(slot.Position(equipSlot))
		if err != nil {
			return nil, errors.New("failed to locate equipment being scrolled")
		}
		sm, ok := c.Equipment().Get(s.Type)
		if !ok || sm.Equipable == nil {
			return nil, errors.New("failed to locate equipment being scrolled")
		}
		return sm.Equipable, nil
	}
	a, ok := c.Inventory().Equipable().FindBySlot(equipSlot)
	if !ok {
		return nil, errors.New("failed to locate equipment being scrolled")
	}
	return a, nil
}

// RequestVegaScroll validates everything up front (FR-2: every rejection here
// consumes nothing), then starts the chained single-item reservation flow
// (design §3.2): CASH vega first; its RESERVED confirmation triggers the USE
// scroll reservation; the scroll's confirmation triggers ConsumeVegaScroll.
// NEVER batch the two reserves — the inventory-side batch path only processes
// the first entry (design §2.8).
func (p *Processor) RequestVegaScroll(characterId uint32, vegaSlot int16, vegaItemId item2.Id, scrollSlot int16, equipSlot int16) error {
	cp := character.NewProcessor(p.l, p.ctx)
	cpp := compartment.NewProcessor(p.l, p.ctx)
	transactionId := uuid.New()

	required, boosted, ok := vegaRates(vegaItemId)
	if !ok || !item2.IsVegasSpell(vegaItemId) {
		return p.VegaScrollError(characterId, transactionId, nil, errors.New("not a vega scroll item"))
	}

	c, err := cp.GetById(cp.InventoryDecorator)(characterId)
	if err != nil {
		return p.VegaScrollError(characterId, transactionId, nil, err)
	}

	vegaItem, ok := c.Inventory().Cash().FindBySlot(vegaSlot)
	if !ok || item2.Id(vegaItem.TemplateId()) != vegaItemId {
		return p.VegaScrollError(characterId, transactionId, nil, errors.New("vega item not found"))
	}

	scrollItem, ok := c.Inventory().Consumable().FindBySlot(scrollSlot)
	if !ok {
		return p.VegaScrollError(characterId, transactionId, nil, errors.New("scroll item not found"))
	}

	ci, err := p.cdp.GetById(scrollItem.TemplateId())
	if err != nil {
		return p.VegaScrollError(characterId, transactionId, nil, err)
	}
	if ci.SuccessRate() != required {
		p.l.Debugf("Character [%d] vega [%d] rejected: scroll [%d] rate [%d] does not match required [%d].", characterId, vegaItemId, scrollItem.TemplateId(), ci.SuccessRate(), required)
		return p.VegaScrollError(characterId, transactionId, nil, errors.New("scroll rate mismatch"))
	}

	equip, err := resolveVegaEquip(c, equipSlot)
	if err != nil {
		return p.VegaScrollError(characterId, transactionId, nil, err)
	}
	if !p.ValidateScrollUse(*scrollItem, *equip) {
		return p.VegaScrollError(characterId, transactionId, nil, errors.New("failed slot validation"))
	}

	p.l.Debugf("Character [%d] using vega [%d]: scroll [%d] (slot [%d], rate [%d] boosted to [%d]) onto equip slot [%d] (transaction [%s]).",
		characterId, vegaItemId, scrollItem.TemplateId(), scrollSlot, required, boosted, equipSlot, transactionId.String())

	t, _ := topic.EnvProvider(p.l)(compartment2.EnvEventTopicStatus)()
	scrollValidator := once.ReservationValidator(transactionId, scrollItem.TemplateId())
	scrollHandler := compartment.Consume(ConsumeVegaScroll(transactionId, characterId, vegaItem, scrollItem, equipSlot, boosted))
	if _, err = consumer.GetManager().RegisterHandler(t, message.AdaptHandler(message.OneTimeConfig(scrollValidator, scrollHandler))); err != nil {
		return p.VegaScrollError(characterId, transactionId, nil, err)
	}
	vegaValidator := once.ReservationValidator(transactionId, vegaItem.TemplateId())
	vegaHandler := compartment.Consume(ReserveVegaScrollStage(transactionId, characterId, vegaItem, scrollItem))
	if _, err = consumer.GetManager().RegisterHandler(t, message.AdaptHandler(message.OneTimeConfig(vegaValidator, vegaHandler))); err != nil {
		return p.VegaScrollError(characterId, transactionId, nil, err)
	}

	err = cpp.RequestReserve(transactionId, characterId, inventory2.TypeValueCash, []compartment.Reserves{{
		Slot:     vegaSlot,
		ItemId:   vegaItem.TemplateId(),
		Quantity: 1,
	}})
	if err != nil {
		return p.VegaScrollError(characterId, transactionId, nil, err)
	}
	return nil
}

// ReserveVegaScrollStage fires when the vega CASH reservation confirms; it
// issues the second (USE scroll) reservation of the chain. A synchronous
// producer failure cancels the vega reservation. An asynchronous inventory-
// side rejection emits nothing — the vega reservation TTL-expires (~30s) and
// the player keeps everything (design §2.9).
func ReserveVegaScrollStage(transactionId uuid.UUID, characterId uint32, vegaItem *asset.Model, scrollItem *asset.Model) ItemConsumer {
	return func(l logrus.FieldLogger) func(ctx context.Context) error {
		return func(ctx context.Context) error {
			p := NewProcessor(l, ctx)
			cpp := compartment.NewProcessor(l, ctx)
			l.Debugf("Character [%d] vega reservation confirmed (transaction [%s]); reserving scroll in slot [%d].", characterId, transactionId.String(), scrollItem.Slot())
			err := cpp.RequestReserve(transactionId, characterId, inventory2.TypeValueUse, []compartment.Reserves{{
				Slot:     scrollItem.Slot(),
				ItemId:   scrollItem.TemplateId(),
				Quantity: 1,
			}})
			if err != nil {
				return p.VegaScrollError(characterId, transactionId, []vegaReservation{{inventory2.TypeValueCash, vegaItem.Slot()}}, err)
			}
			return nil
		}
	}
}

// ConsumeVegaScroll fires when both reservations are confirmed: re-validates
// (state may have moved between request and confirmation), applies the scroll
// at the boosted rate via the shared core (whiteScroll=false), commits both
// reservations, handles curse destruction, and emits the VEGA_SCROLL event.
func ConsumeVegaScroll(transactionId uuid.UUID, characterId uint32, vegaItem *asset.Model, scrollItem *asset.Model, equipSlot int16, boostedProb uint32) ItemConsumer {
	return func(l logrus.FieldLogger) func(ctx context.Context) error {
		return func(ctx context.Context) error {
			p := NewProcessor(l, ctx)
			cp := character.NewProcessor(l, ctx)
			cpp := compartment.NewProcessor(l, ctx)
			both := []vegaReservation{
				{inventory2.TypeValueUse, scrollItem.Slot()},
				{inventory2.TypeValueCash, vegaItem.Slot()},
			}

			l.Debugf("Character [%d] has reserved vega [%d] and scroll [%d]. Applying scroll at boosted rate [%d] (transaction [%s]).", characterId, vegaItem.TemplateId(), scrollItem.TemplateId(), boostedProb, transactionId.String())
			c, err := cp.GetById(cp.InventoryDecorator)(characterId)
			if err != nil {
				return p.VegaScrollError(characterId, transactionId, both, err)
			}

			required, _, _ := vegaRates(item2.Id(vegaItem.TemplateId()))
			ci, err := p.cdp.GetById(scrollItem.TemplateId())
			if err != nil {
				return p.VegaScrollError(characterId, transactionId, both, err)
			}
			if ci.SuccessRate() != required {
				return p.VegaScrollError(characterId, transactionId, both, errors.New("scroll rate mismatch"))
			}
			equip, err := resolveVegaEquip(c, equipSlot)
			if err != nil {
				return p.VegaScrollError(characterId, transactionId, both, err)
			}
			if !p.ValidateScrollUse(*scrollItem, *equip) {
				return p.VegaScrollError(characterId, transactionId, both, errors.New("failed slot validation"))
			}

			// whiteScroll=false and legendarySpirit=false throughout (FR-4.2).
			outcome, err := applyScrollCore(l, ctx, transactionId, characterId, ci, scrollItem, equip, boostedProb, false)
			if err != nil {
				return p.VegaScrollError(characterId, transactionId, both, err)
			}

			if err = cpp.ConsumeItem(characterId, inventory2.TypeValueUse, transactionId, scrollItem.Slot()); err != nil {
				l.WithError(err).Errorf("Unable to consume item [%d] for character [%d] used during scrolling.", scrollItem.TemplateId(), characterId)
			}
			if err = cpp.ConsumeItem(characterId, inventory2.TypeValueCash, transactionId, vegaItem.Slot()); err != nil {
				l.WithError(err).Errorf("Unable to consume item [%d] for character [%d] used during scrolling.", vegaItem.TemplateId(), characterId)
			}
			if outcome.cursed {
				if err = cpp.DestroyItem(characterId, inventory2.TypeValueEquip, equipSlot); err != nil {
					l.WithError(err).Errorf("Unable to destroy item in slot [%d] for character [%d] during scrolling.", equipSlot, characterId)
				}
			}
			return producer.ProviderImpl(l)(ctx)(consumable.EnvEventTopic)(VegaScrollEventProvider(ts.Id(characterId))(outcome.success, outcome.cursed))
		}
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-consumables/atlas.com/consumables && go test -race ./consumable/ -v && go build ./... && go vet ./...`
Expected: PASS; clean build and vet.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-consumables/atlas.com/consumables/consumable/vega.go services/atlas-consumables/atlas.com/consumables/consumable/vega_test.go
git commit -m "feat(consumables): vega scroll request, chained reservations, boosted consume (task-130)"
```

---

### Task 8: atlas-consumables — consumer arm for REQUEST_VEGA_SCROLL

**Files:**
- Modify: `services/atlas-consumables/atlas.com/consumables/kafka/consumer/consumable/consumer.go`

**Interfaces:**
- Consumes: Task 5 (`CommandRequestVegaScroll`, `RequestVegaScrollBody`), Task 7 (`RequestVegaScroll`).
- Produces: the service-side entry point for the channel's command (Task 9's producer).

- [ ] **Step 1: Add the handler + registration**

In `InitHandlers`, after the `handleRequestScroll` registration:

```go
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleRequestVegaScroll))); err != nil {
			return err
		}
```

At the bottom of the file, after `handleRequestScroll`:

```go
func handleRequestVegaScroll(l logrus.FieldLogger, ctx context.Context, c consumable2.Command[consumable2.RequestVegaScrollBody]) {
	if c.Type != consumable2.CommandRequestVegaScroll {
		return
	}
	err := consumable.NewProcessor(l, ctx).RequestVegaScroll(uint32(c.CharacterId), int16(c.Body.VegaSlot), c.Body.VegaItemId, int16(c.Body.ScrollSlot), int16(c.Body.EquipSlot))
	if err != nil {
		l.WithError(err).Errorf("Character [%d] unable to vega scroll with item in slot [%d] as expected.", c.CharacterId, c.Body.VegaSlot)
	}
}
```

- [ ] **Step 2: Build + full service test gate**

Run: `cd services/atlas-consumables/atlas.com/consumables && go build ./... && go test -race ./... && go vet ./...`
Expected: clean.

- [ ] **Step 3: Commit**

```bash
git add services/atlas-consumables/atlas.com/consumables/kafka/consumer/consumable/consumer.go
git commit -m "feat(consumables): consume REQUEST_VEGA_SCROLL commands (task-130)"
```

---

### Task 9: atlas-channel — command mirror, producer, processor method

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/message/consumable/kafka.go`
- Modify: `services/atlas-channel/atlas.com/channel/consumable/producer.go`
- Modify: `services/atlas-channel/atlas.com/channel/consumable/processor.go`

**Interfaces:**
- Consumes: existing channel-local `Command[E]`/`Event[E]` envelopes and `producer.ProviderImpl` wrapper.
- Produces (used by Tasks 10, 11):
  - mirror constants/bodies: `CommandRequestVegaScroll`, `RequestVegaScrollBody`, `EventTypeVegaScroll`, `VegaScrollBody`, `ErrorTypeVegaInvalid` (field-for-field identical JSON to Task 5's — the mirrors must serialize compatibly)
  - `RequestVegaScrollCommandProvider(f field.Model, characterId character.Id, vegaSlot slot.Position, vegaItemId item.Id, scrollSlot slot.Position, equipSlot slot.Position) model.Provider[[]kafka.Message]`
  - `func (p *Processor) RequestVegaScrollUse(f field.Model, characterId character.Id, vegaItemId item.Id, vegaSlot slot.Position, scrollSlot slot.Position, equipSlot slot.Position) error`

- [ ] **Step 1: Extend the message mirror**

In `kafka/message/consumable/kafka.go` add, mirroring Task 5 exactly (same JSON tags):

```go
	CommandRequestVegaScroll = "REQUEST_VEGA_SCROLL"
```

```go
type RequestVegaScrollBody struct {
	VegaSlot   slot.Position `json:"vegaSlot"`
	VegaItemId item.Id       `json:"vegaItemId"`
	ScrollSlot slot.Position `json:"scrollSlot"`
	EquipSlot  slot.Position `json:"equipSlot"`
}
```

```go
	EventTypeVegaScroll = "VEGA_SCROLL"
```

```go
	ErrorTypeVegaInvalid = "VEGA_INVALID"
```

```go
type VegaScrollBody struct {
	Success bool `json:"success"`
	Cursed  bool `json:"cursed"`
}
```

- [ ] **Step 2: Add the producer + processor method**

In `consumable/producer.go`, after `RequestScrollCommandProvider`:

```go
func RequestVegaScrollCommandProvider(f field.Model, characterId character.Id, vegaSlot slot.Position, vegaItemId item.Id, scrollSlot slot.Position, equipSlot slot.Position) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &consumable.Command[consumable.RequestVegaScrollBody]{
		WorldId:     f.WorldId(),
		ChannelId:   f.ChannelId(),
		MapId:       f.MapId(),
		Instance:    f.Instance(),
		CharacterId: characterId,
		Type:        consumable.CommandRequestVegaScroll,
		Body: consumable.RequestVegaScrollBody{
			VegaSlot:   vegaSlot,
			VegaItemId: vegaItemId,
			ScrollSlot: scrollSlot,
			EquipSlot:  equipSlot,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
```

In `consumable/processor.go`, after `RequestScrollUse`:

```go
func (p *Processor) RequestVegaScrollUse(f field.Model, characterId character.Id, vegaItemId item.Id, vegaSlot slot.Position, scrollSlot slot.Position, equipSlot slot.Position) error {
	p.l.Debugf("Character [%d] attempting vega scroll [%d] from cash slot [%d]: scroll slot [%d] onto equip slot [%d].", characterId, vegaItemId, vegaSlot, scrollSlot, equipSlot)
	return producer.ProviderImpl(p.l)(p.ctx)(consumable2.EnvCommandTopic)(RequestVegaScrollCommandProvider(f, characterId, vegaSlot, vegaItemId, scrollSlot, equipSlot))
}
```

- [ ] **Step 3: Build gate + commit**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go vet ./...`
Expected: clean.

```bash
git add services/atlas-channel/atlas.com/channel/kafka/message/consumable/kafka.go services/atlas-channel/atlas.com/channel/consumable/producer.go services/atlas-channel/atlas.com/channel/consumable/processor.go
git commit -m "feat(channel): REQUEST_VEGA_SCROLL command mirror and producer (task-130)"
```

---

### Task 10: atlas-channel — vega dispatch arm in the cash-item-use handler

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go`
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_test.go`

**Interfaces:**
- Consumes: Task 1 (`item.IsVegasSpell`, `item.ClassificationVegasSpell`), Task 2 (`cashsb.ItemUseVegaScroll`), Task 9 (`RequestVegaScrollUse`).
- Produces: the serverbound entry point. Named constants `CashSlotItemTypeVegasSpellPre95 = CashSlotItemType(68)` and `CashSlotItemTypeVegasSpell95 = CashSlotItemType(71)`.

These CashSlotItemType values are server-internal dispatch bookkeeping mirroring the client's `get_consume_cash_item_type` — they never appear on the wire, so JMS falling into the pre-95 else-branch (68) is correct by construction (design §4.4).

- [ ] **Step 1: Write the failing test**

Create `character_cash_item_use_test.go` (pure classification test — no session/socket machinery; construct tenants the same way the packet tests do, or with the tenant lib directly):

```go
package handler

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func mustTenant(t *testing.T, region string, major uint16, minor uint16) tenant.Model {
	t.Helper()
	m, err := tenant.Create(uuid.New(), region, major, minor)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return m
}

func TestGetCashSlotItemTypeVegasSpell(t *testing.T) {
	pre95 := mustTenant(t, "GMS", 83, 1)
	v95 := mustTenant(t, "GMS", 95, 1)
	jms := mustTenant(t, "JMS", 185, 1)

	cases := []struct {
		name string
		tn   tenant.Model
		id   item.Id
		want CashSlotItemType
	}{
		{"v83 vega 10", pre95, item.VegasSpell10, CashSlotItemTypeVegasSpellPre95},
		{"v83 vega 60", pre95, item.VegasSpell60, CashSlotItemTypeVegasSpellPre95},
		{"v95 vega 10", v95, item.VegasSpell10, CashSlotItemTypeVegasSpell95},
		{"v95 vega 60", v95, item.VegasSpell60, CashSlotItemTypeVegasSpell95},
		{"jms vega 10", jms, item.VegasSpell10, CashSlotItemTypeVegasSpellPre95},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := GetCashSlotItemType(tc.tn)(tc.id); got != tc.want {
				t.Errorf("GetCashSlotItemType(%d) = %d, want %d", tc.id, got, tc.want)
			}
		})
	}
}
```

(Adjust `tenant.Create`'s exact signature to the lib — check `libs/atlas-tenant`; other handler tests in this package that build tenants show the working call shape. Add the `uuid` import.)

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run TestGetCashSlotItemTypeVegasSpell -v`
Expected: FAIL — `undefined: CashSlotItemTypeVegasSpellPre95` (compile error).

- [ ] **Step 3: Implement the arm**

In `character_cash_item_use.go`:

1. Add the constants to the const block (lines 116-120):

```go
	CashSlotItemTypeVegasSpellPre95 = CashSlotItemType(68)
	CashSlotItemTypeVegasSpell95    = CashSlotItemType(71)
```

2. In `GetCashSlotItemType`, replace the raw-literal branch (line 476) `if category == 561 {` with:

```go
		if category == item.ClassificationVegasSpell {
			if t.Region() == "GMS" && t.MajorVersion() >= 95 {
				return CashSlotItemTypeVegasSpell95
			}
			return CashSlotItemTypeVegasSpellPre95
		}
```

3. Change the handler signature's discarded writer producer (line 25) from `_ writer.Producer` to `wp writer.Producer`.

4. Add the dispatch arm after the FieldEffect arm (line 106), before the fall-through warn:

```go
		if it == CashSlotItemTypeVegasSpellPre95 || it == CashSlotItemTypeVegasSpell95 {
			sp := cashsb.ItemUseVegaScroll{}
			sp.Decode(l, ctx)(r, readerOptions)
			l.Debugf("[%s] read vega sub-body [%s]", p.Operation(), sp.String())
			enableActions := func() {
				_ = session.Announce(l)(ctx)(wp)(statpkt.StatChangedWriter)(statpkt.NewStatChanged(make([]statpkt.Update, 0), true).Encode)(s)
			}
			if !item.IsVegasSpell(itemId) {
				l.Warnf("Character [%d] attempted vega scroll with non-vega category-561 item [%d]. Rejecting.", s.CharacterId(), itemId)
				enableActions()
				return
			}
			if sp.EquipTab() != 1 || sp.ScrollTab() != 2 {
				l.Warnf("Character [%d] vega scroll with unexpected tab markers equip [%d] scroll [%d]. Impossible from a legit client. Rejecting.", s.CharacterId(), sp.EquipTab(), sp.ScrollTab())
				enableActions()
				return
			}
			_ = consumable.NewProcessor(l, ctx).RequestVegaScrollUse(s.Field(), character.Id(s.CharacterId()), itemId, source, slot.Position(sp.ScrollSlot()), slot.Position(sp.EquipSlot()))
			return
		}
```

Add the `statpkt "github.com/Chronicle20/atlas/libs/atlas-packet/stat/clientbound"` import (alias per `character_skill_use.go:134`). Leave the `// TODO for v83 there is a trailing updateTime.` comment at line 108 in place — it belongs to the remaining un-migrated arms (task-126 owns the same note; do not double-remove).

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go test -race ./socket/handler/ -run TestGetCashSlotItemTypeVegasSpell -v && go build ./...`
Expected: PASS; clean build.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_test.go
git commit -m "feat(channel): vega spell dispatch arm in cash-item-use handler (task-130)"
```

---

### Task 11: atlas-channel — VEGA_SCROLL event consumer + writer registration

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/consumable/consumer.go`
- Modify: `services/atlas-channel/atlas.com/channel/main.go` (`produceWriters()`, ~line 609)

**Interfaces:**
- Consumes: Task 3 (`VegaScrollWriter`, `VegaScrollStartBody`, `VegaScrollResultBody`, `VegaScrollInvalidBody` from `cash/clientbound`), Task 9 mirrors (`EventTypeVegaScroll`, `VegaScrollBody`, `ErrorTypeVegaInvalid`), existing `charpkt.NewItemUpgrade`, `statpkt.NewStatChanged`, `_map.ForSessionsInMap`.
- Produces: the client-facing feedback path (FR-5): to the user's session — VEGA start(outcome), VEGA result(outcome) back-to-back (both clients latch the result and animate on their own clock, design §2.2/§2.3) — then the map-broadcast `ItemUpgrade` with `legendarySpirit=false, whiteScroll=false`, then enable-actions. Inventory-modify packets arrive via the existing compartment/asset event consumers.

- [ ] **Step 1: Add the VEGA_SCROLL handler**

In `kafka/consumer/consumable/consumer.go`, register it in `InitHandlers` after the scroll handler registration (same pattern):

```go
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleVegaScrollConsumableEvent(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
```

Add the handler at the bottom (new import: `cashpkt "github.com/Chronicle20/atlas/libs/atlas-packet/cash/clientbound"`):

```go
func handleVegaScrollConsumableEvent(sc server.Model, wp writer.Producer) message.Handler[consumable2.Event[consumable2.VegaScrollBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e consumable2.Event[consumable2.VegaScrollBody]) {
		if e.Type != consumable2.EventTypeVegaScroll {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(uint32(e.CharacterId), func(s session.Model) error {
			// Start + result back-to-back: the result is resolved immediately
			// server-side (owner decision); the client animates on its own
			// clock and latches the result byte.
			if err := session.Announce(l)(ctx)(wp)(cashpkt.VegaScrollWriter)(cashpkt.VegaScrollStartBody(e.Body.Success))(s); err != nil {
				return err
			}
			if err := session.Announce(l)(ctx)(wp)(cashpkt.VegaScrollWriter)(cashpkt.VegaScrollResultBody(e.Body.Success))(s); err != nil {
				return err
			}
			if err := _map.NewProcessor(l, ctx).ForSessionsInMap(s.Field(), session.Announce(l)(ctx)(wp)(charpkt.CharacterItemUpgradeWriter)(charpkt.NewItemUpgrade(uint32(e.CharacterId), e.Body.Success, e.Body.Cursed, false, false).Encode)); err != nil {
				return err
			}
			return session.Announce(l)(ctx)(wp)(statpkt.StatChangedWriter)(statpkt.NewStatChanged(make([]statpkt.Update, 0), true).Encode)(s)
		})
		if err != nil {
			l.WithError(err).Errorf("Unable to process vega scroll event for character [%d].", e.CharacterId)
		}
	}
}
```

(If `session.Announce(...)(...)` is not directly applicable to a `session.Model` as `(s)` — confirm against `character_skill_use.go:134`, which uses exactly that shape.)

- [ ] **Step 2: Extend the error handler**

In `handleErrorConsumableEvent`, after the `ErrorTypePetCannotConsume` branch:

```go
		if e.Body.Error == consumable2.ErrorTypeVegaInvalid {
			// INVALID (0x42 on both verified versions) closes the dialog with
			// the client's own "This item cannot be used." notice — required,
			// since the dialog is excl-request-blocked after sending (design
			// §2.3/§4.7); then enable-actions.
			err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(uint32(e.CharacterId), func(s session.Model) error {
				if err := session.Announce(l)(ctx)(wp)(cashpkt.VegaScrollWriter)(cashpkt.VegaScrollInvalidBody())(s); err != nil {
					return err
				}
				return session.Announce(l)(ctx)(wp)(statpkt.StatChangedWriter)(statpkt.NewStatChanged(make([]statpkt.Update, 0), true).Encode)(s)
			})
			if err != nil {
				l.WithError(err).Errorf("Unable to process error event for character [%d].", e.CharacterId)
			}
			return
		}
```

- [ ] **Step 3: Register the writer**

In `main.go` `produceWriters()`, add alongside the other cash writers (~line 616-618):

```go
		cashcb.VegaScrollWriter,
```

- [ ] **Step 4: Build + test gate**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test -race ./... && go vet ./...`
Expected: clean.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/consumer/consumable/consumer.go services/atlas-channel/atlas.com/channel/main.go
git commit -m "feat(channel): VEGA_SCROLL event consumer, invalid arm, writer registration (task-130)"
```

---

### Task 12: Seed templates, live-config rollout doc, parked versions

**Files:**
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_83_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_95_1.json`
- Modify (conditional on Task 4 verification): `template_gms_87_1.json`, `template_gms_84_1.json`, `template_jms_185_1.json`
- Create: `docs/tasks/task-130-vegas-spell/deployment.md`

**Interfaces:**
- Consumes: Task 4's pinned per-version opcodes and operations values; Task 3's writer name and keys.
- Produces: tenant wiring for new tenants + the rollout doc for live ones.

**Rules:**
- A version gets a `VegaScroll` writer entry ONLY if Task 4 verified its opcode and modes. Unverified = omitted (absence is the safe failure; a wrong opcode can crash the client — design §2.5). gms_v92: no entries, parked.
- Handler entries: `CharacterCashItemUseHandle` must exist for the feature to work. It is present in gms_83 (opCode 0x4F) and gms_84 today; task-126 adds it for gms_87 (0x52), gms_95 (0x55), jms_185 (0x47). **Skip-if-already-present:** add each handler entry only if the template doesn't already have it (check first — task-126 may have landed). Every handler entry carries `"validator": "LoggedInValidator"` (a validator-less entry is silently dropped — known gotcha).

- [ ] **Step 1: gms_83 writer entry**

In `template_gms_83_1.json` `socket.writers`, add (values from design §2.3, IDA-verified):

```json
{"opCode": "0x166", "writer": "VegaScroll", "options": {"operations": {"START_SUCCESS": 64, "START_FAILURE": 64, "RESULT_SUCCESS": 65, "RESULT_FAILURE": 67, "INVALID": 66}}}
```

- [ ] **Step 2: gms_95 writer entry + handler**

In `template_gms_95_1.json` `socket.writers`, add — **using Task 4's pinned START pairing** (hypothesis shown; swap 68/73 if Task 4 pinned the reverse):

```json
{"opCode": "0x1AD", "writer": "VegaScroll", "options": {"operations": {"START_SUCCESS": 68, "START_FAILURE": 73, "RESULT_SUCCESS": 69, "RESULT_FAILURE": 71, "INVALID": 66}}}
```

If `socket.handlers` has no `CharacterCashItemUseHandle` entry (task-126 not landed yet), add:

```json
{"opCode": "0x55", "validator": "LoggedInValidator", "handler": "CharacterCashItemUseHandle"}
```

- [ ] **Step 3: gms_87 / gms_84 / jms_185 — only as verified by Task 4**

For each version Task 4 verified: add the writer entry with that version's verified opcode + operations values, and (for gms_87 0x52 / jms_185 0x47) the skip-if-present handler entry. For each version Task 4 reported BLOCKED: add nothing; list it in `deployment.md` as parked-pending-IDB (the item no-ops via the fall-through warn — same posture as the v92 mount-food precedent).

- [ ] **Step 4: Write the rollout doc**

Create `docs/tasks/task-130-vegas-spell/deployment.md`:

```markdown
# task-130 Vega's Spell — Rollout Notes

## Live tenants (seed templates only affect NEW tenants)

For every live tenant on a wired version, PATCH the tenant socket config:
1. Add the `VegaScroll` writer entry (opcode + operations per version — copy
   from the matching seed template, which is the source of truth post-merge).
2. Ensure the `CharacterCashItemUseHandle` handler entry exists with
   `LoggedInValidator` (gms_87/95/jms tenants created before task-126/130).
3. Restart the tenant's atlas-channel pods — handlers/writers do NOT
   hot-reload from config changes (known gotcha).

Symptom of a missed patch: using a Vega's Spell logs the handler fall-through
warn (missing handler) or "Property [operations] missing ... defaulting to 99"
(missing writer options); the item no-ops or the dialog shows "This item
cannot be used."

## Parked versions

- gms_v92: no IDB, no registry file, no USE_CASH_ITEM handler — the whole
  cash-item-use path is inert there (design §2.6). CSV hint for a future IDB:
  VEGA_SCROLL 0x1A0 (UNVERIFIED).
- <list any version Task 4 reported BLOCKED, with what evidence is missing>
```

(Replace the final placeholder line with the actual Task 4 outcomes; delete it if all five versions verified.)

- [ ] **Step 5: Template gates + commit**

```bash
for tpl in services/atlas-configurations/seed-data/templates/template_gms_83_1.json \
           services/atlas-configurations/seed-data/templates/template_gms_95_1.json; do
  python3 -m json.tool "$tpl" > /dev/null && tools/template-symbol-check.sh "$tpl"
done
```
(Also run for any 87/84/jms template touched in Step 3.)
Expected: valid JSON; no DANGLING symbols (the `VegaScroll` literal exists from Task 3).

```bash
git add services/atlas-configurations/seed-data/templates/ docs/tasks/task-130-vegas-spell/deployment.md
git commit -m "feat(config): VegaScroll writer + operations in tenant seed templates (task-130)"
```

---

### Task 13: Final verification gates

**Files:** none (verification only; fix-and-rebuild as needed).

- [ ] **Step 1: Per-module gates**

```bash
(cd libs/atlas-constants && go test -race ./... && go vet ./... && go build ./...)
(cd libs/atlas-packet && go test -race ./... && go vet ./... && go build ./...)
(cd services/atlas-consumables/atlas.com/consumables && go test -race ./... && go vet ./... && go build ./...)
(cd services/atlas-channel/atlas.com/channel && go test -race ./... && go vet ./... && go build ./...)
(cd tools/packet-audit && go test ./... && go build ./...)
```
Expected: all clean.

- [ ] **Step 2: Packet tooling gates**

```bash
go run ./tools/packet-audit matrix --check
go run ./tools/packet-audit dispatcher-lint
```
Expected: exit 0 for both (VegaScroll is not a dispatcher family; dispatcher-lint must simply not regress).

- [ ] **Step 3: Redis key guard**

Run from the repo root (never with a global `GOWORK=off` prefix): `tools/redis-key-guard.sh`
Expected: clean.

- [ ] **Step 4: Docker bakes**

Both shared libs (atlas-packet, atlas-constants) changed, so bake everything:

```bash
docker buildx bake all-go-services
```
Expected: every image builds. (Minimum on iteration: `docker buildx bake atlas-channel atlas-consumables atlas-configurations`.)

- [ ] **Step 5: Acceptance-criteria sweep**

Walk PRD §10 and check each criterion against the code (cite file:line). In particular:
- `grep -rn "consume vega scroll" services/` → nothing.
- `grep -n "vega" services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_entry.go` → the line-29 TODO is still present (explicit non-goal).
- Roll threshold visible in logs as 30/90 on the vega path (`applyScrollCore` "Rolled/Needed" line).

- [ ] **Step 6: Code review, then commit any fixes**

Run `superpowers:requesting-code-review` (mandatory before PR — CLAUDE.md). Address findings, re-run the touched gates, commit.

---

## Self-Review Notes (performed at plan time)

- **Spec coverage:** design §4.1→Task 1; §2.1/§4.2→Tasks 2+4; §2.2-2.3/§3.3/§4.3→Tasks 3+4; §2.4-2.6/§6→Tasks 4+12; §3.1/§4.5→Tasks 5+9; §4.6 (core extraction + TODO deletion)→Task 6; §2.7/§3.2/§4.6→Task 7; consumer arm→Task 8; §4.4→Task 10; §4.7→Task 11; §4.8 + rollout→Task 12; §8 verification→Task 13. Design §2.8/§2.9 pre-existing findings are owner-flagged in design.md §9, not plan tasks (explicitly out of scope per PRD non-goals).
- **Placeholder scan:** the one intentional template is deployment.md's "<list any version Task 4 reported BLOCKED>" — instructions to replace it with actual outcomes are inline. No TBDs elsewhere; every code step has full code.
- **Type consistency:** `vegaRates` returns `(uint32, uint32, bool)` (Tasks 5/7); `ConsumeVegaScroll(txn, characterId, vegaItem, scrollItem, equipSlot, boostedProb)` matches its Task 7 call site; `RequestVegaScroll(characterId, vegaSlot, vegaItemId, scrollSlot, equipSlot)` matches Task 8's handler; `RequestVegaScrollUse(f, characterId, vegaItemId, vegaSlot, scrollSlot, equipSlot)` matches Task 10's call; mirror JSON tags in Task 9 match Task 5 field-for-field; `VegaScrollStartBody(success bool)` shape is identical across Tasks 3/11; template writer name `"VegaScroll"` (Task 12) equals `VegaScrollWriter` (Task 3).
- **Flagged verify-before-use sites** (signatures read from source at plan time but marked for re-check in their steps): `pt.RoundTrip` call shape (Task 2/3), `tenant.Create` signature (Task 10), `asset.ModelBuilder.SetSlot` spelling and a valid negative slot position (Task 7), `candidate` literal shape in run.go (Task 4).

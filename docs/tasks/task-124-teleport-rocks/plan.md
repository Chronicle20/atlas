# Teleport Rocks (Regular + VIP) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement teleport rocks end-to-end: persisted 5-slot regular + 10-slot VIP saved-map lists in atlas-character, the two serverbound ops and the `MAP_TRANSFER_RESULT` writer in atlas-channel, real lists in the character-data codec, the warp/consume saga, seed-template wiring for all six versions, and full byte-fixture packet verification.

**Architecture:** atlas-character owns a new `teleport_rock` GORM domain (slot-per-row, `saved_location` package shape) mutated via Kafka commands and read via REST; atlas-channel decodes the client ops, validates the use-flow (fieldLimit/list/continent/session checks), projects status events into `MAP_TRANSFER_RESULT`, and launches a `WarpToRandomPortal` [+ `DestroyAsset`] saga on success. All wire artifacts live in `libs/atlas-packet/teleportrock` with config-resolved mode bytes.

**Tech Stack:** Go, GORM (sqlite in tests), Kafka (atlas-kafka), JSON:API (api2go), atlas-socket request/response, packet-audit tooling, IDA-derived layouts (design §1).

## Global Constraints

- Spec: `docs/tasks/task-124-teleport-rocks/design.md` (all layouts/mode values there are IDA-verified; do NOT invent or alter byte layouts).
- Work only in this worktree (`.worktrees/task-124-teleport-rocks`, branch `task-124-teleport-rocks`). Verify branch after every commit.
- Every socket handler entry in seed templates MUST carry `"validator": "LoggedInValidator"` — a validator-less entry is silently dropped.
- `MAP_TRANSFER_RESULT` mode bytes are NEVER hard-coded in Go: always `atlas_packet.WithResolvedCode("operations", KEY, ...)`. The operations table ships in the same commit as the writer row.
- Immutable models: private fields + getters + Builder. Processors: `Interface` + `Impl`, `NewProcessor(l, ctx, db)`, pure `Method(mb)` vs `MethodAndEmit`. No `*_testhelpers.go` files.
- Tenancy: every row keyed by `tenant_id`; tenant via `tenant.MustFromContext(ctx)`.
- Codec version gate for the VIP block: `(t.Region() == "GMS" && t.MajorVersion() > 28) || t.Region() == "JMS"` — preserve exactly.
- Item semantics: regular rock = item 2320000 (consumed 1 on success only); cash rocks 5040000/5040001 (regular list), 5041000 (VIP list) — never destroyed. VIP-list selector: `itemId/1000 == 5041`.
- Empty wire slots = `_map.EmptyMapId` (999999999). Lists on the wire are a contiguous prefix + padding.
- No `// TODO`, no stubs, no deferred bounded work.
- Verification gates before claiming done: `go test -race ./...`, `go vet ./...`, `go build ./...` in every changed module; `docker buildx bake atlas-character atlas-channel atlas-saga-orchestrator atlas-login` from the worktree root; `tools/redis-key-guard.sh` clean; `packet-audit matrix --check` and `operations --check` exit 0.
- Code review (plan-adherence + backend-guidelines) before PR.

**Module roots** (run `go` commands from these directories):
- `libs/atlas-saga`, `libs/atlas-constants`, `libs/atlas-packet`
- `services/atlas-character/atlas.com/character`
- `services/atlas-channel/atlas.com/channel`

---

### Task 1: Shared constants — saga type, saga re-exports, field-limit bit

**Files:**
- Modify: `libs/atlas-saga/model.go` (Type consts, ~line 22)
- Modify: `services/atlas-channel/atlas.com/channel/saga/model.go` (re-export blocks)
- Modify: `libs/atlas-constants/map/field_limit.go`

**Interfaces:**
- Consumes: nothing.
- Produces: `sharedsaga.TeleportRockUse` (`saga.Type` = `"teleport_rock_use"`); channel-side aliases `saga.TeleportRockUse`, `saga.WarpToRandomPortal` (Action), `saga.WarpToRandomPortalPayload`; `_map.FieldLimitNoTeleportItem uint32 = 0x40`. Used by Tasks 17–19.

- [ ] **Step 1: Add the saga type constant**

In `libs/atlas-saga/model.go`, in the `Type` const block after `FieldEffectUse Type = "field_effect_use"` (line 22), add:

```go
	TeleportRockUse      Type = "teleport_rock_use"
```

- [ ] **Step 2: Re-export in the channel saga package**

In `services/atlas-channel/atlas.com/channel/saga/model.go`:

In the type-alias block (after `WarpToPortalPayload = sharedsaga.WarpToPortalPayload`), add:

```go
	WarpToRandomPortalPayload    = sharedsaga.WarpToRandomPortalPayload
```

In the const block, after `FieldEffectUse = sharedsaga.FieldEffectUse` add:

```go
	TeleportRockUse      = sharedsaga.TeleportRockUse
```

and after `WarpToPortal = sharedsaga.WarpToPortal` add:

```go
	WarpToRandomPortal   = sharedsaga.WarpToRandomPortal
```

- [ ] **Step 3: Add the field-limit constant**

In `libs/atlas-constants/map/field_limit.go`, after `FieldLimitNoMysticDoor` (line 9), add:

```go
	// FieldLimitNoTeleportItem prevents teleport-rock item usage in the map
	// (client RunMapTransferItem checks fieldLimit & 0x40; design task-124 §1 Q2)
	FieldLimitNoTeleportItem uint32 = 0x40
```

- [ ] **Step 4: Compile all three modules**

Run:
```bash
(cd libs/atlas-saga && go build ./... && go vet ./...)
(cd libs/atlas-constants && go build ./... && go vet ./...)
(cd services/atlas-channel/atlas.com/channel && go build ./...)
```
Expected: all clean, no output.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-saga/model.go services/atlas-channel/atlas.com/channel/saga/model.go libs/atlas-constants/map/field_limit.go
git commit -m "feat(task-124): teleport-rock saga type and field-limit constant"
```

---

### Task 2: atlas-packet — shared `Target` payload codec

**Files:**
- Create: `libs/atlas-packet/teleportrock/target.go`
- Test: `libs/atlas-packet/teleportrock/target_test.go`

**Interfaces:**
- Consumes: `request.Reader` (`ReadBool/ReadAsciiString/ReadUint32/Available`), `response.Writer` (`WriteBool/WriteAsciiString/WriteInt`).
- Produces: `teleportrock.Target` with `ByName() bool`, `TargetName() string`, `TargetMap() uint32`, `Valid() bool`, `NewTargetByMap(mapId uint32) Target`, `NewTargetByName(name string) Target`, `(t *Target) Decode(l)(r *request.Reader)`, `(t Target) Encode(w *response.Writer)`. Reused by Tasks 3 and 5.

This models the `CWvsContext::RunMapTransferItem` payload (design §1 Q1): `byte bByName`, then string name (byName=1) or int mapId (byName=0). The client may omit the payload entirely (dialog closed with no selection); a trailing 4-byte `updateTime` always follows the payload in both wrapping ops, so the decoder budgets 4 bytes and treats anything short as invalid — never a panic.

- [ ] **Step 1: Write the failing test**

```go
package teleportrock

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

func decodeTarget(t *testing.T, b []byte) Target {
	t.Helper()
	l, _ := testlog.NewNullLogger()
	r := request.NewReader(&b, 0)
	out := Target{}
	out.Decode(l)(&r)
	return out
}

func TestTargetByMapRoundTrip(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	in := NewTargetByMap(100000000)
	w := response.NewWriter(l)
	in.Encode(w)
	// byName=0, mapId=100000000 LE, plus the trailing updateTime the wrapping op appends
	want := []byte{0x00, 0x00, 0xE1, 0xF5, 0x05}
	got := w.Bytes()
	if len(got) != len(want) {
		t.Fatalf("encoded length: got %d want %d (% x)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("byte %d: got %x want %x", i, got[i], want[i])
		}
	}
	out := decodeTarget(t, append(got, 0x00, 0x00, 0x00, 0x00)) // + trailing updateTime budget
	if !out.Valid() || out.ByName() || out.TargetMap() != 100000000 {
		t.Fatalf("decode: %+v", out)
	}
}

func TestTargetByNameRoundTrip(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	in := NewTargetByName("Adele")
	w := response.NewWriter(l)
	in.Encode(w)
	out := decodeTarget(t, append(w.Bytes(), 0x00, 0x00, 0x00, 0x00))
	if !out.Valid() || !out.ByName() || out.TargetName() != "Adele" {
		t.Fatalf("decode: %+v", out)
	}
}

// The client sends no target payload at all when the dialog resolves with
// neither a name nor a valid map (design §1 Q1 caveat). Only the trailing
// updateTime remains — decode must flag invalid, not read past the buffer.
func TestTargetAbsentPayloadIsInvalid(t *testing.T) {
	out := decodeTarget(t, []byte{0x12, 0x34, 0x56, 0x78}) // 4 bytes = updateTime only
	if out.Valid() {
		t.Fatalf("absent payload must decode as invalid")
	}
}

func TestTargetByMapTruncatedIsInvalid(t *testing.T) {
	// byName=0 but only the 4 trailing updateTime bytes remain — no map id.
	out := decodeTarget(t, []byte{0x00, 0x12, 0x34, 0x56, 0x78})
	if out.Valid() {
		t.Fatalf("byName=0 without a map id must decode as invalid")
	}
}

func TestTargetEmptyNameIsInvalid(t *testing.T) {
	// byName=1, zero-length string, trailing updateTime.
	out := decodeTarget(t, []byte{0x01, 0x00, 0x00, 0x12, 0x34, 0x56, 0x78})
	if out.Valid() {
		t.Fatalf("empty target name must decode as invalid")
	}
}
```

Note: check `request.NewReader`'s exact constructor signature in `libs/atlas-socket/request/reader.go` before writing (other tests in `libs/atlas-packet` construct readers the same way — copy the idiom from an existing serverbound `*_test.go`, e.g. `libs/atlas-packet/door/serverbound/enter_test.go`). Adjust the helper accordingly; the assertions stand.

- [ ] **Step 2: Run test to verify it fails**

Run: `(cd libs/atlas-packet && go test ./teleportrock/... )`
Expected: FAIL — package does not exist / `Target` undefined.

- [ ] **Step 3: Implement `target.go`**

```go
package teleportrock

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// Target models the CWvsContext::RunMapTransferItem request payload shared by
// USE_TELEPORT_ROCK and the cash-item-use teleport-rock branch (design §1 Q1):
//
//	byte bByName
//	  1 -> string targetName (length-prefixed ASCII)
//	  0 -> int dwTargetField (only encoded when a map was actually selected)
//
// The client omits the payload entirely when the dialog resolves with neither
// a name nor a valid map. A trailing 4-byte updateTime always follows the
// payload in both wrapping ops, so Decode budgets those 4 bytes and marks the
// target invalid (never panics) when the remainder is too short.
type Target struct {
	byName     bool
	targetName string
	targetMap  uint32
	valid      bool
}

func NewTargetByMap(mapId uint32) Target {
	return Target{byName: false, targetMap: mapId, valid: true}
}

func NewTargetByName(name string) Target {
	return Target{byName: true, targetName: name, valid: true}
}

func (t Target) ByName() bool        { return t.byName }
func (t Target) TargetName() string  { return t.targetName }
func (t Target) TargetMap() uint32   { return t.targetMap }
func (t Target) Valid() bool         { return t.valid }

func (t Target) String() string {
	if t.byName {
		return fmt.Sprintf("Target{byName=true name=%s valid=%v}", t.targetName, t.valid)
	}
	return fmt.Sprintf("Target{byName=false map=%d valid=%v}", t.targetMap, t.valid)
}

// trailingUpdateTimeBytes is the 4-byte updateTime both wrapping ops append
// after the target payload.
const trailingUpdateTimeBytes = 4

func (t *Target) Decode(_ logrus.FieldLogger) func(r *request.Reader) {
	return func(r *request.Reader) {
		t.valid = false
		if r.Available() <= trailingUpdateTimeBytes {
			return // payload omitted entirely
		}
		t.byName = r.ReadBool()
		if t.byName {
			if r.Available() < 2+trailingUpdateTimeBytes {
				return // not even a string length prefix before the updateTime
			}
			t.targetName = r.ReadAsciiString()
			t.valid = len(t.targetName) > 0
			return
		}
		if r.Available() < 4+trailingUpdateTimeBytes {
			return // byName=0 but no map id was encoded (no selection)
		}
		t.targetMap = r.ReadUint32()
		t.valid = true
	}
}

func (t Target) Encode(w *response.Writer) {
	w.WriteBool(t.byName)
	if t.byName {
		w.WriteAsciiString(t.targetName)
		return
	}
	w.WriteInt(t.targetMap)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `(cd libs/atlas-packet && go test ./teleportrock/... )`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-packet/teleportrock/
git commit -m "feat(task-124): shared RunMapTransferItem target payload codec"
```

---

### Task 3: atlas-packet — serverbound `Use` (USE_TELEPORT_ROCK)

**Files:**
- Create: `libs/atlas-packet/teleportrock/serverbound/use.go`
- Test: `libs/atlas-packet/teleportrock/serverbound/use_test.go`

**Interfaces:**
- Consumes: `teleportrock.Target` (Task 2).
- Produces: `serverbound.Use` with `Slot() int16`, `ItemId() uint32`, `Target() teleportrock.Target`, `UpdateTime() uint32`, `Valid() bool`, `Operation() = "TeleportRockUseHandle"` (const `TeleportRockUseHandle`); `NewUse(slot int16, itemId uint32, target teleportrock.Target, updateTime uint32) Use`. Used by Task 18.

Wire layout (design §1 Q1, `CWvsContext::SendMapTransferItemUseRequest`, v83 `0xA0A3BB` op 0x54 / v95 `0x9E6020` op 0x5B):

```
short  nPOS          // USE-inventory slot
int    nItemID       // client guard: nItemID/10000 == 232 on this op
<Target payload>
int    updateTime    // trailing on BOTH versions (no leading updateTime even on v95)
```

- [ ] **Step 1: Write the failing test**

```go
package serverbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/teleportrock"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// Layout is version-invariant: short slot, int itemId, target payload,
// trailing int updateTime (design §1 Q1 — no leading updateTime even on v95).
//
// packet-audit:verify packet=teleportrock/serverbound/Use version=gms_v83 ida=0xA0A3BB
// packet-audit:verify packet=teleportrock/serverbound/Use version=gms_v95 ida=0x9E6020
func TestUseByMapDecode(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 83, 1)
	b := []byte{
		0x02, 0x00, // slot = 2
		0x80, 0x66, 0x23, 0x00, // itemId = 2320000
		0x00,                   // byName = 0
		0x00, 0xE1, 0xF5, 0x05, // mapId = 100000000
		0x2A, 0x00, 0x00, 0x00, // updateTime = 42
	}
	r := request.NewReader(&b, 0)
	p := Use{}
	p.Decode(l, ctx)(&r, nil)
	if !p.Valid() {
		t.Fatalf("expected valid decode")
	}
	if p.Slot() != 2 || p.ItemId() != 2320000 || p.UpdateTime() != 42 {
		t.Fatalf("fields: %+v", p)
	}
	if p.Target().ByName() || p.Target().TargetMap() != 100000000 {
		t.Fatalf("target: %+v", p.Target())
	}
}

func TestUseByNameDecode(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 95, 1)
	b := []byte{
		0x01, 0x00, // slot = 1
		0x40, 0xEA, 0x4C, 0x00, // itemId = 5040000
		0x01,       // byName = 1
		0x05, 0x00, // name length = 5
		'A', 'd', 'e', 'l', 'e',
		0x00, 0x00, 0x00, 0x00, // updateTime = 0
	}
	r := request.NewReader(&b, 0)
	p := Use{}
	p.Decode(l, ctx)(&r, nil)
	if !p.Valid() || !p.Target().ByName() || p.Target().TargetName() != "Adele" {
		t.Fatalf("decode: %+v target %+v", p, p.Target())
	}
}

// Client sent the packet with no target payload (dialog closed without a
// selection) — must decode as invalid, never panic.
func TestUseAbsentTargetIsInvalid(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 83, 1)
	b := []byte{
		0x02, 0x00,
		0x80, 0x66, 0x23, 0x00,
		0x2A, 0x00, 0x00, 0x00, // only updateTime remains
	}
	r := request.NewReader(&b, 0)
	p := Use{}
	p.Decode(l, ctx)(&r, nil)
	if p.Valid() {
		t.Fatalf("absent target payload must be invalid")
	}
}

func TestUseRoundTrip(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			in := NewUse(3, 5041000, teleportrock.NewTargetByMap(220000000), 7)
			enc := in.Encode(l, ctx)(nil)
			r := request.NewReader(&enc, 0)
			out := Use{}
			out.Decode(l, ctx)(&r, nil)
			if !out.Valid() || out.Slot() != 3 || out.ItemId() != 5041000 ||
				out.Target().TargetMap() != 220000000 || out.UpdateTime() != 7 {
				t.Fatalf("round trip: %+v", out)
			}
		})
	}
}
```

(Adopt the reader-construction idiom from Task 2 / existing serverbound tests if `request.NewReader(&b, 0)` differs. Item id hex: 2320000 = `0x236680` LE `80 66 23 00`; 5040000 = `0x4CEA40` LE `40 EA 4C 00`.)

- [ ] **Step 2: Run test to verify it fails**

Run: `(cd libs/atlas-packet && go test ./teleportrock/serverbound/...)`
Expected: FAIL — `Use` undefined.

- [ ] **Step 3: Implement `use.go`**

```go
package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-packet/teleportrock"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const TeleportRockUseHandle = "TeleportRockUseHandle"

// Use - CWvsContext::SendMapTransferItemUseRequest (USE_TELEPORT_ROCK).
// Layout (design task-124 §1 Q1, version-invariant):
//
//	short nPOS       // USE-inventory slot of the rock
//	int   nItemID    // client-side guard: nItemID/10000 == 232 on this op
//	<RunMapTransferItem target payload — teleportrock.Target>
//	int   updateTime // trailing on all versions (no leading updateTime, even v95)
//
// Valid() is false when the client omitted/truncated the target payload; the
// handler warn-drops such requests (no result packet — the request was
// malformed by the client's own rules).
type Use struct {
	slot       int16
	itemId     uint32
	target     teleportrock.Target
	updateTime uint32
}

func NewUse(slot int16, itemId uint32, target teleportrock.Target, updateTime uint32) Use {
	return Use{slot: slot, itemId: itemId, target: target, updateTime: updateTime}
}

func (m Use) Slot() int16                   { return m.slot }
func (m Use) ItemId() uint32                { return m.itemId }
func (m Use) Target() teleportrock.Target   { return m.target }
func (m Use) UpdateTime() uint32            { return m.updateTime }
func (m Use) Valid() bool                   { return m.target.Valid() }
func (m Use) Operation() string             { return TeleportRockUseHandle }

func (m Use) String() string {
	return fmt.Sprintf("Use{slot=%d itemId=%d target=%s updateTime=%d}", m.slot, m.itemId, m.target.String(), m.updateTime)
}

func (m Use) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt16(m.slot)
		w.WriteInt(m.itemId)
		m.target.Encode(w)
		w.WriteInt(m.updateTime)
		return w.Bytes()
	}
}

func (m *Use) Decode(l logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.slot = r.ReadInt16()
		m.itemId = r.ReadUint32()
		m.target.Decode(l)(r)
		if r.Available() >= 4 {
			m.updateTime = r.ReadUint32()
		}
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `(cd libs/atlas-packet && go test ./teleportrock/...)`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-packet/teleportrock/serverbound/
git commit -m "feat(task-124): USE_TELEPORT_ROCK serverbound decoder"
```

---

### Task 4: atlas-packet — serverbound `AddMap` (TROCK_ADD_MAP)

**Files:**
- Create: `libs/atlas-packet/teleportrock/serverbound/add_map.go`
- Test: `libs/atlas-packet/teleportrock/serverbound/add_map_test.go`

**Interfaces:**
- Consumes: nothing new.
- Produces: `serverbound.AddMap` with `Register() bool`, `Vip() bool`, `MapId() uint32`, `Operation() = "TeleportRockAddMapHandle"` (const `TeleportRockAddMapHandle`); `NewAddMap(register bool, vip bool, mapId uint32) AddMap`. Used by Task 15.

Wire layout (design §1 Q1, `CWvsContext::SendMapTransferRequest`, v83 `0xA261BC` op 0x66 / v95 `0x9F3B90` op 0x72):

```
byte nType                  // 1 = register, 0 = delete
byte bCanTransferContinent  // list selector: 0 = regular (5), 1 = VIP (10)
if nType == 0:
  int dwTargetField         // map to delete
```

**Critical:** on register the client sends NO map id — the server derives the map from session state (design corrects PRD FR-5).

- [ ] **Step 1: Write the failing test**

```go
package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// Layout is version-invariant (design §1 Q1): byte nType, byte
// bCanTransferContinent, then int dwTargetField ONLY when nType==0 (delete).
// On register the client sends no map id — the server uses session state.
//
// packet-audit:verify packet=teleportrock/serverbound/AddMap version=gms_v83 ida=0xA261BC
// packet-audit:verify packet=teleportrock/serverbound/AddMap version=gms_v95 ida=0x9F3B90
func TestAddMapRegisterDecode(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 83, 1)
	b := []byte{0x01, 0x01} // register, VIP list — nothing else on the wire
	r := request.NewReader(&b, 0)
	p := AddMap{}
	p.Decode(l, ctx)(&r, nil)
	if !p.Register() || !p.Vip() || p.MapId() != 0 {
		t.Fatalf("decode: %+v", p)
	}
}

func TestAddMapDeleteDecode(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 95, 1)
	b := []byte{
		0x00, 0x00, // delete, regular list
		0x00, 0xE1, 0xF5, 0x05, // mapId = 100000000
	}
	r := request.NewReader(&b, 0)
	p := AddMap{}
	p.Decode(l, ctx)(&r, nil)
	if p.Register() || p.Vip() || p.MapId() != 100000000 {
		t.Fatalf("decode: %+v", p)
	}
}

func TestAddMapRoundTrip(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			for _, in := range []AddMap{
				NewAddMap(true, false, 0),
				NewAddMap(false, true, 220000000),
			} {
				enc := in.Encode(l, ctx)(nil)
				r := request.NewReader(&enc, 0)
				out := AddMap{}
				out.Decode(l, ctx)(&r, nil)
				if out.Register() != in.Register() || out.Vip() != in.Vip() || out.MapId() != in.MapId() {
					t.Fatalf("round trip: in=%+v out=%+v", in, out)
				}
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `(cd libs/atlas-packet && go test ./teleportrock/serverbound/...)`
Expected: FAIL — `AddMap` undefined.

- [ ] **Step 3: Implement `add_map.go`**

```go
package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const TeleportRockAddMapHandle = "TeleportRockAddMapHandle"

// AddMap - CWvsContext::SendMapTransferRequest (TROCK_ADD_MAP).
// Layout (design task-124 §1 Q1, version-invariant):
//
//	byte nType                 // 1 = register, 0 = delete
//	byte bCanTransferContinent // 0 = regular list (5), 1 = VIP list (10)
//	if nType == 0: int dwTargetField
//
// On register the client sends NO map id: the current map comes from
// server-side session state, never from the packet.
type AddMap struct {
	register bool
	vip      bool
	mapId    uint32
}

func NewAddMap(register bool, vip bool, mapId uint32) AddMap {
	return AddMap{register: register, vip: vip, mapId: mapId}
}

func (m AddMap) Register() bool    { return m.register }
func (m AddMap) Vip() bool         { return m.vip }
func (m AddMap) MapId() uint32     { return m.mapId }
func (m AddMap) Operation() string { return TeleportRockAddMapHandle }

func (m AddMap) String() string {
	return fmt.Sprintf("AddMap{register=%v vip=%v mapId=%d}", m.register, m.vip, m.mapId)
}

func (m AddMap) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteBool(m.register)
		w.WriteBool(m.vip)
		if !m.register {
			w.WriteInt(m.mapId)
		}
		return w.Bytes()
	}
}

func (m *AddMap) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.register = r.ReadBool()
		m.vip = r.ReadBool()
		if !m.register && r.Available() >= 4 {
			m.mapId = r.ReadUint32()
		}
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `(cd libs/atlas-packet && go test ./teleportrock/...)`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-packet/teleportrock/serverbound/add_map.go libs/atlas-packet/teleportrock/serverbound/add_map_test.go
git commit -m "feat(task-124): TROCK_ADD_MAP serverbound decoder"
```

---

### Task 5: atlas-packet — cash serverbound teleport-rock sub-payload

**Files:**
- Create: `libs/atlas-packet/cash/serverbound/item_use_teleport_rock.go`
- Test: `libs/atlas-packet/cash/serverbound/item_use_teleport_rock_test.go`

**Interfaces:**
- Consumes: `teleportrock.Target` (Task 2).
- Produces: `serverbound.ItemUseTeleportRock` with `Target() teleportrock.Target`, `UpdateTime() uint32`; constructor `NewItemUseTeleportRock(updateTimeFirst bool) *ItemUseTeleportRock`. Used by Task 19.

Cash rocks ride `CWvsContext::SendConsumeCashItemUseRequest` (v83 op 0x4F). The common prefix (leading updateTime on v95+, `short nEPOS`, `int nItemID`) is already decoded by `ItemUse` in the handler; this sub-payload is the remainder: the shared `Target` payload, then a trailing `int updateTime` present on all versions (v83 tail `0xA0EA53`; v95 case `0x9EE059` — design §1 Q1). Mirror the constructor shape of `NewItemUsePetConsumable` (`item_use_pet_consumable.go`) so the handler call-site idiom matches; the `updateTimeFirst` flag is retained for signature parity but the trailing read is unconditional (guarded by `Available()`).

- [ ] **Step 1: Write the failing test**

```go
package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// Sub-payload of SendConsumeCashItemUseRequest for teleport rocks (design §1
// Q1): shared RunMapTransferItem target payload + trailing int updateTime on
// ALL versions (v83 tail 0xA0EA53, v95 case 0x9EE059).
//
// packet-audit:verify packet=cash/serverbound/ItemUseTeleportRock version=gms_v83 ida=0xA0EA53
// packet-audit:verify packet=cash/serverbound/ItemUseTeleportRock version=gms_v95 ida=0x9EE059
func TestItemUseTeleportRockByMap(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 83, 1)
	b := []byte{
		0x00,                   // byName = 0
		0x00, 0xE1, 0xF5, 0x05, // mapId = 100000000
		0x2A, 0x00, 0x00, 0x00, // trailing updateTime = 42
	}
	r := request.NewReader(&b, 0)
	p := NewItemUseTeleportRock(false)
	p.Decode(l, ctx)(&r, nil)
	if !p.Target().Valid() || p.Target().TargetMap() != 100000000 || p.UpdateTime() != 42 {
		t.Fatalf("decode: target=%+v updateTime=%d", p.Target(), p.UpdateTime())
	}
}

func TestItemUseTeleportRockByName(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 95, 1)
	b := []byte{
		0x01,       // byName = 1
		0x05, 0x00, // name length
		'A', 'd', 'e', 'l', 'e',
		0x00, 0x00, 0x00, 0x00,
	}
	r := request.NewReader(&b, 0)
	p := NewItemUseTeleportRock(true)
	p.Decode(l, ctx)(&r, nil)
	if !p.Target().Valid() || p.Target().TargetName() != "Adele" {
		t.Fatalf("decode: %+v", p.Target())
	}
}

func TestItemUseTeleportRockAbsentTarget(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 83, 1)
	b := []byte{0x2A, 0x00, 0x00, 0x00} // updateTime only
	r := request.NewReader(&b, 0)
	p := NewItemUseTeleportRock(false)
	p.Decode(l, ctx)(&r, nil)
	if p.Target().Valid() {
		t.Fatalf("absent target payload must be invalid")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `(cd libs/atlas-packet && go test ./cash/serverbound/ -run TestItemUseTeleportRock)`
Expected: FAIL — `ItemUseTeleportRock` undefined.

- [ ] **Step 3: Implement `item_use_teleport_rock.go`**

```go
package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-packet/teleportrock"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// ItemUseTeleportRock - the teleport-rock branch of
// CWvsContext::SendConsumeCashItemUseRequest, after the common ItemUse prefix:
// shared RunMapTransferItem target payload, then trailing int updateTime on
// all versions (design task-124 §1 Q1).
type ItemUseTeleportRock struct {
	updateTimeFirst bool
	target          teleportrock.Target
	updateTime      uint32
}

func NewItemUseTeleportRock(updateTimeFirst bool) *ItemUseTeleportRock {
	return &ItemUseTeleportRock{updateTimeFirst: updateTimeFirst}
}

func (m ItemUseTeleportRock) Target() teleportrock.Target { return m.target }
func (m ItemUseTeleportRock) UpdateTime() uint32          { return m.updateTime }

func (m ItemUseTeleportRock) String() string {
	return fmt.Sprintf("ItemUseTeleportRock{target=%s updateTime=%d}", m.target.String(), m.updateTime)
}

func (m ItemUseTeleportRock) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		m.target.Encode(w)
		w.WriteInt(m.updateTime)
		return w.Bytes()
	}
}

func (m *ItemUseTeleportRock) Decode(l logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.target.Decode(l)(r)
		if r.Available() >= 4 {
			m.updateTime = r.ReadUint32()
		}
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `(cd libs/atlas-packet && go test ./cash/serverbound/ -run TestItemUseTeleportRock)`
Expected: PASS. Also run the full package: `(cd libs/atlas-packet && go test ./cash/...)` — PASS.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-packet/cash/serverbound/item_use_teleport_rock.go libs/atlas-packet/cash/serverbound/item_use_teleport_rock_test.go
git commit -m "feat(task-124): cash item-use teleport-rock sub-payload decoder"
```

---

### Task 6: atlas-packet — clientbound `MapTransferResult` (writer + bodies)

**Files:**
- Create: `libs/atlas-packet/teleportrock/clientbound/result.go`
- Create: `libs/atlas-packet/teleportrock/result_body.go`
- Test: `libs/atlas-packet/teleportrock/clientbound/result_test.go`
- Test: `libs/atlas-packet/teleportrock/result_body_test.go`

**Interfaces:**
- Consumes: `atlas_packet.WithResolvedCode` (`libs/atlas-packet/resolve.go:13`), `_map.EmptyMapId`.
- Produces:
  - `clientbound.MapTransferResultWriter = "MapTransferResult"` (writer name const),
  - `clientbound.NewMapTransferList(mode byte, vip bool, maps []_map.Id) MapTransferList`,
  - `clientbound.NewMapTransferError(mode byte, vip bool) MapTransferError`,
  - root-package mode-key consts `MapTransferModeDeleteList = "DELETE_LIST"`, `MapTransferModeRegisterList = "REGISTER_LIST"`, `MapTransferModeCannotGo = "CANNOT_GO"`, `MapTransferModeUnableToLocate = "UNABLE_TO_LOCATE"`, `MapTransferModeUnableToLocate2 = "UNABLE_TO_LOCATE_2"`, `MapTransferModeCannotGoContinent = "CANNOT_GO_CONTINENT"`, `MapTransferModeCurrentMap = "CURRENT_MAP"`, `MapTransferModeMapNotAvailable = "MAP_NOT_AVAILABLE"`, `MapTransferModeMapleIslandLevel7 = "MAPLE_ISLAND_LEVEL7"`,
  - `teleportrock.MapTransferResultListBody(key string, vip bool, maps []_map.Id) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte`,
  - `teleportrock.MapTransferResultErrorBody(key string, vip bool) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte`.
  Used by Tasks 16–17 and the seed templates (Task 20).

Wire (design §1 Q4, identical v83 `0xA25268` ↔ v95 `0x9F9F90`): `byte mode`, `byte targetList` (0 = regular, 1 = VIP; always present), then for modes 2/3 exactly 5 (regular) or 10 (VIP) × `int mapId` padded with `EmptyMapId`.

- [ ] **Step 1: Write the failing clientbound test**

```go
package clientbound

import (
	"bytes"
	"testing"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// Wire (design §1 Q4, identical v83 0xA25268 / v95 0x9F9F90): byte mode, byte
// targetList (0=regular 1=VIP), then for list modes 5 or 10 x int mapId padded
// with EmptyMapId (999999999 = FF C9 9A 3B LE).
//
// packet-audit:verify packet=teleportrock/clientbound/MapTransferResult version=gms_v83 ida=0xA25268
// packet-audit:verify packet=teleportrock/clientbound/MapTransferResult version=gms_v95 ida=0x9F9F90
func TestMapTransferListRegularGolden(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 83, 1)
	m := NewMapTransferList(3, false, []_map.Id{100000000, 220000000})
	got := m.Encode(l, ctx)(nil)
	want := []byte{
		0x03,                   // mode = REGISTER_LIST
		0x00,                   // targetList = regular
		0x00, 0xE1, 0xF5, 0x05, // 100000000
		0x00, 0xEF, 0x1C, 0x0D, // 220000000
		0xFF, 0xC9, 0x9A, 0x3B, // EmptyMapId
		0xFF, 0xC9, 0x9A, 0x3B,
		0xFF, 0xC9, 0x9A, 0x3B,
	}
	if !bytes.Equal(got, want) {
		t.Errorf("golden mismatch\n got: % x\nwant: % x", got, want)
	}
}

func TestMapTransferListVipPadsToTen(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 95, 1)
	m := NewMapTransferList(2, true, []_map.Id{100000000})
	got := m.Encode(l, ctx)(nil)
	if len(got) != 2+10*4 {
		t.Fatalf("VIP list body must be 42 bytes, got %d", len(got))
	}
	if got[0] != 0x02 || got[1] != 0x01 {
		t.Fatalf("header: % x", got[:2])
	}
	// slots 1..9 must be EmptyMapId
	for i := 0; i < 9; i++ {
		off := 2 + 4 + i*4
		if !bytes.Equal(got[off:off+4], []byte{0xFF, 0xC9, 0x9A, 0x3B}) {
			t.Fatalf("slot %d not padded: % x", i+1, got[off:off+4])
		}
	}
}

func TestMapTransferErrorGolden(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 83, 1)
	m := NewMapTransferError(5, false)
	got := m.Encode(l, ctx)(nil)
	want := []byte{0x05, 0x00}
	if !bytes.Equal(got, want) {
		t.Errorf("golden mismatch\n got: % x\nwant: % x", got, want)
	}
}

func TestMapTransferResultCrossVersionStable(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	m := NewMapTransferList(3, false, []_map.Id{100000000})
	base := m.Encode(l, pt.CreateContext("GMS", 83, 1))(nil)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			got := m.Encode(l, pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion))(nil)
			if !bytes.Equal(got, base) {
				t.Errorf("%s differs from v83\n got: % x\nv83: % x", v.Name, got, base)
			}
		})
	}
}
```

- [ ] **Step 2: Write the failing root-body test**

```go
package teleportrock

import (
	"bytes"
	"context"
	"testing"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

func testOperations() map[string]interface{} {
	return map[string]interface{}{
		"operations": map[string]interface{}{
			MapTransferModeDeleteList:        "0x02",
			MapTransferModeRegisterList:      "0x03",
			MapTransferModeCannotGo:          "0x05",
			MapTransferModeUnableToLocate:    "0x06",
			MapTransferModeUnableToLocate2:   "0x07",
			MapTransferModeCannotGoContinent: "0x08",
			MapTransferModeCurrentMap:        "0x09",
			MapTransferModeMapNotAvailable:   "0x0A",
			MapTransferModeMapleIslandLevel7: "0x0B",
		},
	}
}

// The mode byte is config-resolved via WithResolvedCode("operations", key) —
// never hard-coded (known crash class when the table is missing).
func TestMapTransferResultListBodyResolvesMode(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	got := MapTransferResultListBody(MapTransferModeRegisterList, false, []_map.Id{100000000})(l, context.Background())(testOperations())
	if got[0] != 0x03 || got[1] != 0x00 {
		t.Fatalf("header: % x", got[:2])
	}
	if len(got) != 2+5*4 {
		t.Fatalf("regular list body must be 22 bytes, got %d", len(got))
	}
}

func TestMapTransferResultErrorBodyResolvesMode(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	got := MapTransferResultErrorBody(MapTransferModeCannotGoContinent, true)(l, context.Background())(testOperations())
	want := []byte{0x08, 0x01}
	if !bytes.Equal(got, want) {
		t.Fatalf("got % x want % x", got, want)
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `(cd libs/atlas-packet && go test ./teleportrock/...)`
Expected: FAIL — `NewMapTransferList`, `MapTransferResultListBody` undefined.

- [ ] **Step 4: Implement `clientbound/result.go`**

```go
package clientbound

import (
	"context"
	"fmt"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const MapTransferResultWriter = "MapTransferResult"

// MapTransferList is the list-refresh form of MAP_TRANSFER_RESULT (modes 2/3):
// byte mode, byte targetList (0=regular 1=VIP), then exactly 5 (regular) or 10
// (VIP) x int mapId padded with EmptyMapId. The client reloads
// adwMapTransfer[5] / adwMapTransferEx[10] from this packet (design §1 Q4;
// identical v83 0xA25268 / v95 0x9F9F90).
type MapTransferList struct {
	mode byte
	vip  bool
	maps []_map.Id
}

func NewMapTransferList(mode byte, vip bool, maps []_map.Id) MapTransferList {
	return MapTransferList{mode: mode, vip: vip, maps: maps}
}

func (m MapTransferList) Operation() string { return MapTransferResultWriter }
func (m MapTransferList) String() string {
	return fmt.Sprintf("MapTransferList{mode=%d vip=%v maps=%v}", m.mode, m.vip, m.maps)
}

func (m MapTransferList) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteBool(m.vip)
		count := 5
		if m.vip {
			count = 10
		}
		for i := 0; i < count; i++ {
			v := _map.EmptyMapId
			if i < len(m.maps) {
				v = m.maps[i]
			}
			w.WriteInt(uint32(v))
		}
		return w.Bytes()
	}
}

// MapTransferError is the error form of MAP_TRANSFER_RESULT (modes 5-11):
// byte mode, byte targetList — no list payload.
type MapTransferError struct {
	mode byte
	vip  bool
}

func NewMapTransferError(mode byte, vip bool) MapTransferError {
	return MapTransferError{mode: mode, vip: vip}
}

func (m MapTransferError) Operation() string { return MapTransferResultWriter }
func (m MapTransferError) String() string {
	return fmt.Sprintf("MapTransferError{mode=%d vip=%v}", m.mode, m.vip)
}

func (m MapTransferError) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteBool(m.vip)
		return w.Bytes()
	}
}
```

- [ ] **Step 5: Implement `result_body.go`**

```go
package teleportrock

import (
	"context"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	atlas_packet "github.com/Chronicle20/atlas/libs/atlas-packet"
	"github.com/Chronicle20/atlas/libs/atlas-packet/teleportrock/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

// MAP_TRANSFER_RESULT mode keys, resolved per-version from the tenant
// template's "operations" table (design §7). Never hard-code the byte values.
const (
	MapTransferModeDeleteList        = "DELETE_LIST"
	MapTransferModeRegisterList      = "REGISTER_LIST"
	MapTransferModeCannotGo          = "CANNOT_GO"
	MapTransferModeUnableToLocate    = "UNABLE_TO_LOCATE"
	MapTransferModeUnableToLocate2   = "UNABLE_TO_LOCATE_2"
	MapTransferModeCannotGoContinent = "CANNOT_GO_CONTINENT"
	MapTransferModeCurrentMap        = "CURRENT_MAP"
	MapTransferModeMapNotAvailable   = "MAP_NOT_AVAILABLE"
	MapTransferModeMapleIslandLevel7 = "MAPLE_ISLAND_LEVEL7"
)

// MapTransferResultListBody emits the list-refresh form (REGISTER_LIST /
// DELETE_LIST): the full post-mutation list for the affected list, padded to
// 5/10 with EmptyMapId. The client only updates its UI from this packet.
func MapTransferResultListBody(key string, vip bool, maps []_map.Id) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", key, func(mode byte) packet.Encoder {
		return clientbound.NewMapTransferList(mode, vip, maps)
	})
}

// MapTransferResultErrorBody emits an error mode (CANNOT_GO, UNABLE_TO_LOCATE,
// CANNOT_GO_CONTINENT, CURRENT_MAP, MAP_NOT_AVAILABLE, ...).
func MapTransferResultErrorBody(key string, vip bool) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", key, func(mode byte) packet.Encoder {
		return clientbound.NewMapTransferError(mode, vip)
	})
}
```

(If `packet.Encoder` is not satisfied by the value types, check how `messenger/operation_body.go:24-28` returns its encoders and mirror exactly.)

- [ ] **Step 6: Run tests to verify they pass**

Run: `(cd libs/atlas-packet && go test ./teleportrock/...)`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add libs/atlas-packet/teleportrock/
git commit -m "feat(task-124): MAP_TRANSFER_RESULT writer bodies with config-resolved modes"
```

---

### Task 7: atlas-packet — `CharacterData` real teleport lists

**Files:**
- Modify: `libs/atlas-packet/character/data.go` (struct fields + `encodeTeleports`/`decodeTeleports`, lines 700–720)
- Test: `libs/atlas-packet/character/data_test.go` (append)

**Interfaces:**
- Consumes: existing `CharacterData` struct + `pt.RoundTrip`.
- Produces: exported fields `CharacterData.TeleportMaps []_map.Id` (regular, ≤5) and `CharacterData.VipTeleportMaps []_map.Id` (≤10). Task 14 populates them.

- [ ] **Step 1: Write the failing test (append to `data_test.go`)**

```go
// FR-15: the teleport region carries real lists; empty slots still encode
// EmptyMapId; the VIP block keeps its (GMS>28)||JMS gate. Decode strips
// padding so round-trip is stable on the canonical (unpadded) form.
func TestCharacterDataTeleportRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := CharacterData{
				Stats: CharacterStats{
					Id: 1000, Name: "TestChar", SkinColor: 1,
					Face: 20000, Hair: 30000, Level: 50, JobId: 312,
					MapId: 100000000,
				},
				Inventory: InventoryData{
					EquipCapacity: 24, UseCapacity: 24, SetupCapacity: 24,
					EtcCapacity: 24, CashCapacity: 24,
					Timestamp: 94354848000000000,
				},
				TeleportMaps:    []_map.Id{100000000, 220000000},
				VipTeleportMaps: []_map.Id{104040000},
			}
			output := CharacterData{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if len(output.TeleportMaps) != 2 || output.TeleportMaps[0] != 100000000 || output.TeleportMaps[1] != 220000000 {
				t.Errorf("teleportMaps: got %v", output.TeleportMaps)
			}
			vipExpected := (v.Region == "GMS" && v.MajorVersion > 28) || v.Region == "JMS"
			if vipExpected {
				if len(output.VipTeleportMaps) != 1 || output.VipTeleportMaps[0] != 104040000 {
					t.Errorf("vipTeleportMaps: got %v", output.VipTeleportMaps)
				}
			} else if len(output.VipTeleportMaps) != 0 {
				t.Errorf("vip block must be absent for %s: got %v", v.Name, output.VipTeleportMaps)
			}
		})
	}
}
```

Add `_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"` to the test imports if absent. Check the `pt.Variants` struct field names (`Region`, `MajorVersion`) against `libs/atlas-packet/test` before writing the gate expression.

- [ ] **Step 2: Run test to verify it fails**

Run: `(cd libs/atlas-packet && go test ./character/ -run TestCharacterDataTeleportRoundTrip)`
Expected: FAIL — `TeleportMaps` undefined.

- [ ] **Step 3: Implement**

Add to the `CharacterData` struct (near the other exported list fields):

```go
	// TeleportMaps / VipTeleportMaps are the saved teleport-rock lists
	// (regular: 5 slots, VIP: 10 slots). Encoding pads with EmptyMapId;
	// decoding strips the padding.
	TeleportMaps    []_map.Id
	VipTeleportMaps []_map.Id
```

Replace `encodeTeleports`/`decodeTeleports` (data.go:700-720):

```go
func (m *CharacterData) encodeTeleports(w *response.Writer, t tenant.Model) {
	for i := 0; i < 5; i++ {
		v := _map.EmptyMapId
		if i < len(m.TeleportMaps) {
			v = m.TeleportMaps[i]
		}
		w.WriteInt(uint32(v))
	}
	if (t.Region() == "GMS" && t.MajorVersion() > 28) || t.Region() == "JMS" {
		for i := 0; i < 10; i++ {
			v := _map.EmptyMapId
			if i < len(m.VipTeleportMaps) {
				v = m.VipTeleportMaps[i]
			}
			w.WriteInt(uint32(v))
		}
	}
}

func (m *CharacterData) decodeTeleports(r *request.Reader, t tenant.Model) {
	for i := 0; i < 5; i++ {
		v := _map.Id(r.ReadUint32())
		if v != _map.EmptyMapId {
			m.TeleportMaps = append(m.TeleportMaps, v)
		}
	}
	if (t.Region() == "GMS" && t.MajorVersion() > 28) || t.Region() == "JMS" {
		for i := 0; i < 10; i++ {
			v := _map.Id(r.ReadUint32())
			if v != _map.EmptyMapId {
				m.VipTeleportMaps = append(m.VipTeleportMaps, v)
			}
		}
	}
}
```

- [ ] **Step 4: Run the full package test suite**

Run: `(cd libs/atlas-packet && go test ./character/...)`
Expected: PASS — the new test and all pre-existing round-trips (nothing pinned the all-empty teleport region: verified during design).

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-packet/character/data.go libs/atlas-packet/character/data_test.go
git commit -m "feat(task-124): CharacterData carries real teleport-rock lists"
```

---

### Task 8: atlas-character — `teleport_rock` domain foundation

**Files:**
- Create: `services/atlas-character/atlas.com/character/teleport_rock/entity.go`
- Create: `services/atlas-character/atlas.com/character/teleport_rock/model.go`
- Create: `services/atlas-character/atlas.com/character/teleport_rock/builder.go`
- Create: `services/atlas-character/atlas.com/character/teleport_rock/administrator.go`
- Create: `services/atlas-character/atlas.com/character/teleport_rock/provider.go`
- Modify: `services/atlas-character/atlas.com/character/main.go` (line 68: add migration)
- Test: `services/atlas-character/atlas.com/character/teleport_rock/administrator_test.go`

**Interfaces:**
- Consumes: `saved_location` package shape (`entity.go:13-24`), `database.RegisterTenantCallbacks` test harness (`character/processor_test.go:20-38`).
- Produces (used by Tasks 9–12):
  - consts `ListTypeRegular = "regular"`, `ListTypeVip = "vip"`, `RegularCapacity = 5`, `VipCapacity = 10`; helpers `ListType(vip bool) string`, `Capacity(vip bool) int`, `EligibleForRegistration(mapId _map.Id) bool`.
  - `Model` with `CharacterId() uint32`, `Regular() []_map.Id`, `Vip() []_map.Id`, `List(vip bool) []_map.Id`, `Contains(vip bool, mapId _map.Id) bool`; `NewBuilder()` with `SetCharacterId/SetRegular/SetVip/Build`.
  - `Migration(db *gorm.DB) error`.
  - package-private administrator: `getByCharacterId(db, tenantId, characterId) ([]entity, error)` ordered by `list_type, slot`; `replaceList(db, tenantId, characterId, listType string, maps []_map.Id) error`; `deleteByCharacterId(db, tenantId, characterId) error`.
  - `DeleteForCharacter(db *gorm.DB, tenantId uuid.UUID, characterId uint32) error` (exported — called from `character.Delete`'s transaction in Task 12).

- [ ] **Step 1: Write `entity.go`**

```go
package teleport_rock

import (
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&entity{})
}

type entity struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;not null"`
	TenantId    uuid.UUID `gorm:"not null;uniqueIndex:idx_trock_lookup,priority:1"`
	CharacterId uint32    `gorm:"not null;uniqueIndex:idx_trock_lookup,priority:2"`
	ListType    string    `gorm:"not null;uniqueIndex:idx_trock_lookup,priority:3"` // "regular" | "vip"
	Slot        int       `gorm:"not null;uniqueIndex:idx_trock_lookup,priority:4"` // 0-based position
	MapId       _map.Id   `gorm:"not null"`
}

func (e entity) TableName() string {
	return "teleport_rock_maps"
}
```

- [ ] **Step 2: Write `model.go` and `builder.go`**

`model.go`:

```go
package teleport_rock

import (
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

const (
	ListTypeRegular = "regular"
	ListTypeVip     = "vip"
	RegularCapacity = 5
	VipCapacity     = 10
)

// ListType maps the wire-level VIP flag to the persisted list discriminator.
func ListType(vip bool) string {
	if vip {
		return ListTypeVip
	}
	return ListTypeRegular
}

// Capacity is enforced in the processor, not the schema (PRD §6).
func Capacity(vip bool) int {
	if vip {
		return VipCapacity
	}
	return RegularCapacity
}

// EligibleForRegistration is the client's numeric save rule (design §1 Q2):
// a map may be saved iff mapId/100000000 != 0 && (mapId/1000000)%100 != 9.
// This bars all sub-9-digit maps (Maple Island, Masteria, GM maps) and every
// x09xxxxxxx event block. It is NOT a fieldLimit check.
func EligibleForRegistration(mapId _map.Id) bool {
	return uint32(mapId)/100000000 != 0 && (uint32(mapId)/1000000)%100 != 9
}

// Model holds both saved-map lists for one character (unpadded, ordered).
type Model struct {
	characterId uint32
	regular     []_map.Id
	vip         []_map.Id
}

func (m Model) CharacterId() uint32 { return m.characterId }
func (m Model) Regular() []_map.Id  { return m.regular }
func (m Model) Vip() []_map.Id      { return m.vip }

func (m Model) List(vip bool) []_map.Id {
	if vip {
		return m.vip
	}
	return m.regular
}

func (m Model) Contains(vip bool, mapId _map.Id) bool {
	for _, v := range m.List(vip) {
		if v == mapId {
			return true
		}
	}
	return false
}
```

`builder.go`:

```go
package teleport_rock

import (
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

type modelBuilder struct {
	characterId uint32
	regular     []_map.Id
	vip         []_map.Id
}

func NewBuilder() *modelBuilder {
	return &modelBuilder{}
}

func (b *modelBuilder) SetCharacterId(characterId uint32) *modelBuilder {
	b.characterId = characterId
	return b
}

func (b *modelBuilder) SetRegular(maps []_map.Id) *modelBuilder {
	b.regular = maps
	return b
}

func (b *modelBuilder) SetVip(maps []_map.Id) *modelBuilder {
	b.vip = maps
	return b
}

func (b *modelBuilder) Build() Model {
	return Model{
		characterId: b.characterId,
		regular:     b.regular,
		vip:         b.vip,
	}
}
```

- [ ] **Step 3: Write the failing administrator test**

`administrator_test.go` (package `teleport_rock` — internal test, the administrator is unexported; use the same sqlite harness as `character/processor_test.go:20-38`):

```go
package teleport_rock

import (
	"testing"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func testDatabase(t *testing.T) *gorm.DB {
	l, _ := test.NewNullLogger()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	database.RegisterTenantCallbacks(l, db)
	if err := Migration(db); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}
	return db
}

func TestReplaceListAndGet(t *testing.T) {
	db := testDatabase(t)
	tenantId := uuid.New()

	if err := replaceList(db, tenantId, 42, ListTypeRegular, []_map.Id{100000000, 220000000}); err != nil {
		t.Fatalf("replaceList: %v", err)
	}
	if err := replaceList(db, tenantId, 42, ListTypeVip, []_map.Id{104040000}); err != nil {
		t.Fatalf("replaceList vip: %v", err)
	}

	es, err := getByCharacterId(db, tenantId, 42)
	if err != nil {
		t.Fatalf("getByCharacterId: %v", err)
	}
	if len(es) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(es))
	}

	// Replace compacts: overwriting the regular list removes stale rows.
	if err := replaceList(db, tenantId, 42, ListTypeRegular, []_map.Id{220000000}); err != nil {
		t.Fatalf("replaceList overwrite: %v", err)
	}
	es, _ = getByCharacterId(db, tenantId, 42)
	regular := 0
	for _, e := range es {
		if e.ListType == ListTypeRegular {
			if e.Slot != 0 || e.MapId != 220000000 {
				t.Fatalf("expected compacted slot 0 map 220000000, got slot %d map %d", e.Slot, e.MapId)
			}
			regular++
		}
	}
	if regular != 1 {
		t.Fatalf("expected 1 regular row, got %d", regular)
	}
}

func TestTenantIsolation(t *testing.T) {
	db := testDatabase(t)
	a, b := uuid.New(), uuid.New()
	_ = replaceList(db, a, 42, ListTypeRegular, []_map.Id{100000000})
	es, err := getByCharacterId(db, b, 42)
	if err != nil {
		t.Fatalf("getByCharacterId: %v", err)
	}
	if len(es) != 0 {
		t.Fatalf("tenant b must see no rows, got %d", len(es))
	}
}

func TestDeleteForCharacter(t *testing.T) {
	db := testDatabase(t)
	tenantId := uuid.New()
	_ = replaceList(db, tenantId, 42, ListTypeRegular, []_map.Id{100000000})
	_ = replaceList(db, tenantId, 42, ListTypeVip, []_map.Id{104040000})
	if err := DeleteForCharacter(db, tenantId, 42); err != nil {
		t.Fatalf("DeleteForCharacter: %v", err)
	}
	es, _ := getByCharacterId(db, tenantId, 42)
	if len(es) != 0 {
		t.Fatalf("expected 0 rows after delete, got %d", len(es))
	}
}
```

- [ ] **Step 4: Run test to verify it fails**

Run: `(cd services/atlas-character/atlas.com/character && go test ./teleport_rock/...)`
Expected: FAIL — `replaceList` undefined.

- [ ] **Step 5: Implement `administrator.go` and `provider.go`**

`administrator.go`:

```go
package teleport_rock

import (
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func getByCharacterId(db *gorm.DB, tenantId uuid.UUID, characterId uint32) ([]entity, error) {
	var es []entity
	err := db.Where("tenant_id = ? AND character_id = ?", tenantId, characterId).
		Order("list_type, slot").
		Find(&es).Error
	return es, err
}

// replaceList rewrites one list wholesale: delete all rows for the
// (character, listType) pair, re-insert the new list with contiguous 0-based
// slots. Lists are at most 10 rows, so full rewrite keeps compaction trivial
// and slot uniqueness conflict-free (design §3).
func replaceList(db *gorm.DB, tenantId uuid.UUID, characterId uint32, listType string, maps []_map.Id) error {
	err := db.Where("tenant_id = ? AND character_id = ? AND list_type = ?", tenantId, characterId, listType).
		Delete(&entity{}).Error
	if err != nil {
		return err
	}
	for i, m := range maps {
		e := &entity{
			ID:          uuid.New(),
			TenantId:    tenantId,
			CharacterId: characterId,
			ListType:    listType,
			Slot:        i,
			MapId:       m,
		}
		if err := db.Create(e).Error; err != nil {
			return err
		}
	}
	return nil
}

func deleteByCharacterId(db *gorm.DB, tenantId uuid.UUID, characterId uint32) error {
	return db.Where("tenant_id = ? AND character_id = ?", tenantId, characterId).
		Delete(&entity{}).Error
}

// DeleteForCharacter removes both lists for a character. Called from
// character.Delete's transaction (FR-8 lifecycle cleanup).
func DeleteForCharacter(db *gorm.DB, tenantId uuid.UUID, characterId uint32) error {
	return deleteByCharacterId(db, tenantId, characterId)
}
```

`provider.go`:

```go
package teleport_rock

import (
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

func modelFromEntities(characterId uint32, es []entity) Model {
	var regular, vip []_map.Id
	for _, e := range es {
		if e.ListType == ListTypeVip {
			vip = append(vip, e.MapId)
		} else {
			regular = append(regular, e.MapId)
		}
	}
	return NewBuilder().
		SetCharacterId(characterId).
		SetRegular(regular).
		SetVip(vip).
		Build()
}
```

- [ ] **Step 6: Run test to verify it passes**

Run: `(cd services/atlas-character/atlas.com/character && go test ./teleport_rock/...)`
Expected: PASS.

- [ ] **Step 7: Register the migration**

In `services/atlas-character/atlas.com/character/main.go` line 68, add `teleport_rock.Migration`:

```go
	db := database.Connect(l, database.SetMigrations(character.Migration, history.Migration, saved_location.Migration, teleport_rock.Migration))
```

Add import `"atlas-character/teleport_rock"`.

Run: `(cd services/atlas-character/atlas.com/character && go build ./...)` — clean.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-character/atlas.com/character/teleport_rock/ services/atlas-character/atlas.com/character/main.go
git commit -m "feat(task-124): teleport_rock domain entity, model, administrator"
```

---

### Task 9: atlas-character — processor with validations + status events

**Files:**
- Create: `services/atlas-character/atlas.com/character/kafka/message/teleportrock/kafka.go`
- Create: `services/atlas-character/atlas.com/character/teleport_rock/producer.go`
- Create: `services/atlas-character/atlas.com/character/teleport_rock/processor.go`
- Test: `services/atlas-character/atlas.com/character/teleport_rock/processor_test.go`

**Interfaces:**
- Consumes: Task 8 administrator/model; `atlas-character/kafka/message` (`message.Buffer`, `message.Emit`), `atlas-character/kafka/producer` (`producer.ProviderImpl`), `tenant.MustFromContext`.
- Produces (used by Tasks 10–12 and mirrored channel-side in Task 13):
  - message consts: `EnvCommandTopic = "COMMAND_TOPIC_TELEPORT_ROCK"`, `CommandAddMap = "ADD_MAP"`, `CommandRemoveMap = "REMOVE_MAP"`, `EnvEventTopicStatus = "EVENT_TOPIC_TELEPORT_ROCK_STATUS"`, `StatusEventTypeListUpdated = "LIST_UPDATED"`, `StatusEventTypeError = "ERROR"`, `ErrorReasonListFull = "LIST_FULL"`, `ErrorReasonDuplicate = "DUPLICATE"`, `ErrorReasonMapNotAllowed = "MAP_NOT_ALLOWED"`, `ErrorReasonNotFound = "NOT_FOUND"`.
  - envelopes `Command[E]` / `StatusEvent[E]` `{TransactionId uuid.UUID; WorldId world.Id; CharacterId uint32; Type string; Body E}`; bodies `AddMapCommandBody{MapId _map.Id; Vip bool}`, `RemoveMapCommandBody{MapId _map.Id; Vip bool}`, `ListUpdatedStatusBody{Vip bool; Registered bool; Maps []_map.Id}`, `ErrorStatusBody{Vip bool; Reason string}`.
  - `Processor` interface: `GetByCharacterId(characterId uint32) (Model, error)`; `AddMap(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) error` + `AddMapAndEmit(...)`; `RemoveMap(mb) func(...) error` + `RemoveMapAndEmit(...)`; `NewProcessor(l, ctx, db) Processor`.

- [ ] **Step 1: Write `kafka/message/teleportrock/kafka.go`**

```go
package teleportrock

import (
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	EnvCommandTopic  = "COMMAND_TOPIC_TELEPORT_ROCK"
	CommandAddMap    = "ADD_MAP"
	CommandRemoveMap = "REMOVE_MAP"

	EnvEventTopicStatus        = "EVENT_TOPIC_TELEPORT_ROCK_STATUS"
	StatusEventTypeListUpdated = "LIST_UPDATED"
	StatusEventTypeError       = "ERROR"

	ErrorReasonListFull      = "LIST_FULL"
	ErrorReasonDuplicate     = "DUPLICATE"
	ErrorReasonMapNotAllowed = "MAP_NOT_ALLOWED"
	ErrorReasonNotFound      = "NOT_FOUND"
)

type Command[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	WorldId       world.Id  `json:"worldId"`
	CharacterId   uint32    `json:"characterId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type AddMapCommandBody struct {
	MapId _map.Id `json:"mapId"`
	Vip   bool    `json:"vip"`
}

type RemoveMapCommandBody struct {
	MapId _map.Id `json:"mapId"`
	Vip   bool    `json:"vip"`
}

type StatusEvent[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	WorldId       world.Id  `json:"worldId"`
	CharacterId   uint32    `json:"characterId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

// ListUpdatedStatusBody carries the authoritative post-mutation list for the
// affected list only (unpadded). Registered picks REGISTER_LIST vs DELETE_LIST
// on projection (design §4.2).
type ListUpdatedStatusBody struct {
	Vip        bool      `json:"vip"`
	Registered bool      `json:"registered"`
	Maps       []_map.Id `json:"maps"`
}

type ErrorStatusBody struct {
	Vip    bool   `json:"vip"`
	Reason string `json:"reason"`
}
```

- [ ] **Step 2: Write `teleport_rock/producer.go`**

```go
package teleport_rock

import (
	teleportrock2 "atlas-character/kafka/message/teleportrock"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func listUpdatedEventProvider(transactionId uuid.UUID, worldId world.Id, characterId uint32, vip bool, registered bool, maps []_map.Id) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &teleportrock2.StatusEvent[teleportrock2.ListUpdatedStatusBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		CharacterId:   characterId,
		Type:          teleportrock2.StatusEventTypeListUpdated,
		Body: teleportrock2.ListUpdatedStatusBody{
			Vip:        vip,
			Registered: registered,
			Maps:       maps,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func errorEventProvider(transactionId uuid.UUID, worldId world.Id, characterId uint32, vip bool, reason string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &teleportrock2.StatusEvent[teleportrock2.ErrorStatusBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		CharacterId:   characterId,
		Type:          teleportrock2.StatusEventTypeError,
		Body: teleportrock2.ErrorStatusBody{
			Vip:    vip,
			Reason: reason,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
```

(Match the import path/idiom of `character/producer.go` — if that file imports the service-local `atlas-character/kafka/producer` wrapper instead of the lib, mirror it exactly.)

- [ ] **Step 3: Write the failing processor test**

`processor_test.go` (package `teleport_rock_test`; reuses the internal-package `testDatabase` helper? No — internal helper is in package `teleport_rock`, so keep this test in package `teleport_rock` too and extend the same file conventions):

Write in package `teleport_rock`:

```go
package teleport_rock

import (
	"context"
	"testing"

	"atlas-character/kafka/message"
	teleportrock2 "atlas-character/kafka/message/teleportrock"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
)

func testContext(t *testing.T) context.Context {
	m, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return tenant.WithContext(context.Background(), m)
}

// bufferedTypes extracts the status-event types buffered on a topic so tests
// can assert LIST_UPDATED vs ERROR without a live producer. Inspect the
// message.Buffer API (atlas-character/kafka/message) — GetAll()/messages per
// topic — and decode the value JSON's "type" field.
func addMap(t *testing.T, p Processor, mb *message.Buffer, characterId uint32, mapId _map.Id, vip bool) {
	t.Helper()
	if err := p.AddMap(mb)(uuid.New(), 0, characterId, mapId, vip); err != nil {
		t.Fatalf("AddMap: %v", err)
	}
}

func TestAddMapPersistsAndBuffersListUpdated(t *testing.T) {
	db := testDatabase(t)
	ctx := testContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	mb := message.NewBuffer()
	addMap(t, p, mb, 42, 100000000, false)

	m, err := p.GetByCharacterId(42)
	if err != nil {
		t.Fatalf("GetByCharacterId: %v", err)
	}
	if len(m.Regular()) != 1 || m.Regular()[0] != 100000000 {
		t.Fatalf("regular list: %v", m.Regular())
	}
	assertBuffered(t, mb, teleportrock2.StatusEventTypeListUpdated, "")
}

func TestAddMapRejectsIneligible(t *testing.T) {
	db := testDatabase(t)
	ctx := testContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	for _, mapId := range []_map.Id{4000000, 909000000} { // sub-9-digit; x09 block
		mb := message.NewBuffer()
		addMap(t, p, mb, 42, mapId, false)
		assertBuffered(t, mb, teleportrock2.StatusEventTypeError, teleportrock2.ErrorReasonMapNotAllowed)
	}
	m, _ := p.GetByCharacterId(42)
	if len(m.Regular()) != 0 {
		t.Fatalf("nothing should persist: %v", m.Regular())
	}
}

func TestAddMapRejectsDuplicate(t *testing.T) {
	db := testDatabase(t)
	ctx := testContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	addMap(t, p, message.NewBuffer(), 42, 100000000, false)
	mb := message.NewBuffer()
	addMap(t, p, mb, 42, 100000000, false)
	assertBuffered(t, mb, teleportrock2.StatusEventTypeError, teleportrock2.ErrorReasonDuplicate)
}

func TestAddMapRejectsWhenFull(t *testing.T) {
	db := testDatabase(t)
	ctx := testContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	maps := []_map.Id{100000000, 101000000, 102000000, 103000000, 104000000}
	for _, m := range maps {
		addMap(t, p, message.NewBuffer(), 42, m, false)
	}
	mb := message.NewBuffer()
	addMap(t, p, mb, 42, 105000000, false)
	assertBuffered(t, mb, teleportrock2.StatusEventTypeError, teleportrock2.ErrorReasonListFull)

	// VIP list is independent: same character can still add there (10 cap).
	mb = message.NewBuffer()
	addMap(t, p, mb, 42, 105000000, true)
	assertBuffered(t, mb, teleportrock2.StatusEventTypeListUpdated, "")
}

func TestRemoveMapCompactsAndBuffersListUpdated(t *testing.T) {
	db := testDatabase(t)
	ctx := testContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	for _, m := range []_map.Id{100000000, 101000000, 102000000} {
		addMap(t, p, message.NewBuffer(), 42, m, false)
	}
	mb := message.NewBuffer()
	if err := p.RemoveMap(mb)(uuid.New(), 0, 42, 101000000, false); err != nil {
		t.Fatalf("RemoveMap: %v", err)
	}
	m, _ := p.GetByCharacterId(42)
	if len(m.Regular()) != 2 || m.Regular()[0] != 100000000 || m.Regular()[1] != 102000000 {
		t.Fatalf("compaction failed: %v", m.Regular())
	}
	assertBuffered(t, mb, teleportrock2.StatusEventTypeListUpdated, "")
}

func TestRemoveMapNotFound(t *testing.T) {
	db := testDatabase(t)
	ctx := testContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	mb := message.NewBuffer()
	if err := p.RemoveMap(mb)(uuid.New(), 0, 42, 100000000, false); err != nil {
		t.Fatalf("RemoveMap: %v", err)
	}
	assertBuffered(t, mb, teleportrock2.StatusEventTypeError, teleportrock2.ErrorReasonNotFound)
}
```

Also write the `assertBuffered(t *testing.T, mb *message.Buffer, wantType string, wantReason string)` helper in the same file: read the buffered messages for `teleportrock2.EnvEventTopicStatus` from the buffer (inspect `atlas-character/kafka/message/message.go` for the accessor — e.g. `mb.GetAll()` map keyed by topic env), `json.Unmarshal` each `kafka.Message.Value` into `StatusEvent[json.RawMessage]`, assert exactly one event of `wantType`; when `wantReason != ""` unmarshal the body into `ErrorStatusBody` and assert `Reason == wantReason`. If the Buffer API differs, adapt the helper only — the assertions stand.

- [ ] **Step 4: Run test to verify it fails**

Run: `(cd services/atlas-character/atlas.com/character && go test ./teleport_rock/...)`
Expected: FAIL — `NewProcessor` undefined.

- [ ] **Step 5: Implement `processor.go`**

```go
package teleport_rock

import (
	"atlas-character/kafka/message"
	teleportrock2 "atlas-character/kafka/message/teleportrock"
	"atlas-character/kafka/producer"
	"context"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	GetByCharacterId(characterId uint32) (Model, error)
	AddMap(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) error
	AddMapAndEmit(transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) error
	RemoveMap(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) error
	RemoveMapAndEmit(transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
	t   tenant.Model
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		db:  db,
		t:   tenant.MustFromContext(ctx),
	}
}

func (p *ProcessorImpl) GetByCharacterId(characterId uint32) (Model, error) {
	es, err := getByCharacterId(p.db.WithContext(p.ctx), p.t.Id(), characterId)
	if err != nil {
		return Model{}, err
	}
	return modelFromEntities(characterId, es), nil
}

func (p *ProcessorImpl) AddMapAndEmit(transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) error {
	return message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(buf *message.Buffer) error {
		return p.AddMap(buf)(transactionId, worldId, characterId, mapId, vip)
	})
}

// AddMap registers the character's current map (server-derived, design §1 Q1)
// on the selected list. Validation failures buffer an ERROR status event and
// mutate nothing (FR-7: the client updates its UI only from the result packet).
func (p *ProcessorImpl) AddMap(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) error {
	return func(transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) error {
		txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
			es, err := getByCharacterId(tx, p.t.Id(), characterId)
			if err != nil {
				return err
			}
			m := modelFromEntities(characterId, es)
			list := m.List(vip)

			if !EligibleForRegistration(mapId) {
				p.l.Warnf("Character [%d] attempted to register ineligible map [%d].", characterId, mapId)
				return mb.Put(teleportrock2.EnvEventTopicStatus, errorEventProvider(transactionId, worldId, characterId, vip, teleportrock2.ErrorReasonMapNotAllowed))
			}
			if len(list) >= Capacity(vip) {
				p.l.Warnf("Character [%d] attempted to register map [%d] on a full list (vip=%v).", characterId, mapId, vip)
				return mb.Put(teleportrock2.EnvEventTopicStatus, errorEventProvider(transactionId, worldId, characterId, vip, teleportrock2.ErrorReasonListFull))
			}
			if m.Contains(vip, mapId) {
				p.l.Warnf("Character [%d] attempted to register duplicate map [%d] (vip=%v).", characterId, mapId, vip)
				return mb.Put(teleportrock2.EnvEventTopicStatus, errorEventProvider(transactionId, worldId, characterId, vip, teleportrock2.ErrorReasonDuplicate))
			}

			newList := append(append([]_map.Id{}, list...), mapId)
			if err := replaceList(tx, p.t.Id(), characterId, ListType(vip), newList); err != nil {
				return err
			}
			p.l.Debugf("Registered map [%d] for character [%d] (vip=%v, %d entries).", mapId, characterId, vip, len(newList))
			return mb.Put(teleportrock2.EnvEventTopicStatus, listUpdatedEventProvider(transactionId, worldId, characterId, vip, true, newList))
		})
		if txErr != nil {
			p.l.WithError(txErr).Errorf("Unable to register map [%d] for character [%d].", mapId, characterId)
			return txErr
		}
		return nil
	}
}

func (p *ProcessorImpl) RemoveMapAndEmit(transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) error {
	return message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(buf *message.Buffer) error {
		return p.RemoveMap(buf)(transactionId, worldId, characterId, mapId, vip)
	})
}

// RemoveMap deletes a map from the selected list and compacts the remaining
// slots to a contiguous prefix (design §3).
func (p *ProcessorImpl) RemoveMap(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) error {
	return func(transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) error {
		txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
			es, err := getByCharacterId(tx, p.t.Id(), characterId)
			if err != nil {
				return err
			}
			m := modelFromEntities(characterId, es)
			if !m.Contains(vip, mapId) {
				p.l.Warnf("Character [%d] attempted to remove absent map [%d] (vip=%v).", characterId, mapId, vip)
				return mb.Put(teleportrock2.EnvEventTopicStatus, errorEventProvider(transactionId, worldId, characterId, vip, teleportrock2.ErrorReasonNotFound))
			}

			newList := make([]_map.Id, 0, len(m.List(vip)))
			for _, v := range m.List(vip) {
				if v != mapId {
					newList = append(newList, v)
				}
			}
			if err := replaceList(tx, p.t.Id(), characterId, ListType(vip), newList); err != nil {
				return err
			}
			p.l.Debugf("Removed map [%d] for character [%d] (vip=%v, %d entries).", mapId, characterId, vip, len(newList))
			return mb.Put(teleportrock2.EnvEventTopicStatus, listUpdatedEventProvider(transactionId, worldId, characterId, vip, false, newList))
		})
		if txErr != nil {
			p.l.WithError(txErr).Errorf("Unable to remove map [%d] for character [%d].", mapId, characterId)
			return txErr
		}
		return nil
	}
}
```

(Verify the producer wrapper import: `character/processor.go` imports `atlas-character/kafka/producer` — use the same.)

- [ ] **Step 6: Run test to verify it passes**

Run: `(cd services/atlas-character/atlas.com/character && go test ./teleport_rock/...)`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-character/atlas.com/character/teleport_rock/ services/atlas-character/atlas.com/character/kafka/message/teleportrock/
git commit -m "feat(task-124): teleport_rock processor with validations and status events"
```

---

### Task 10: atlas-character — command consumer + main wiring

**Files:**
- Create: `services/atlas-character/atlas.com/character/kafka/consumer/teleportrock/consumer.go`
- Modify: `services/atlas-character/atlas.com/character/main.go` (consumer wiring, lines 71–88)
- Test: `services/atlas-character/atlas.com/character/kafka/consumer/teleportrock/consumer_test.go`

**Interfaces:**
- Consumes: Task 9 processor + message package; `atlas-character/kafka/consumer` (`consumer2.NewConfig`) — same shape as `kafka/consumer/character/consumer.go:20-35`.
- Produces: `teleportrock.InitConsumers(l)(cmf)(consumerGroupId)`, `teleportrock.InitHandlers(l)(db)(rf) error`.

- [ ] **Step 1: Write `consumer.go`**

```go
package teleportrock

import (
	consumer2 "atlas-character/kafka/consumer"
	teleportrock2 "atlas-character/kafka/message/teleportrock"
	"atlas-character/teleport_rock"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("teleport_rock_command")(teleportrock2.EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(rf func(topic string, handler handler.Handler) (string, error)) error {
			var t string
			t, _ = topic.EnvProvider(l)(teleportrock2.EnvCommandTopic)()
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleAddMap(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleRemoveMap(db)))); err != nil {
				return err
			}
			return nil
		}
	}
}

func handleAddMap(db *gorm.DB) message.Handler[teleportrock2.Command[teleportrock2.AddMapCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c teleportrock2.Command[teleportrock2.AddMapCommandBody]) {
		if c.Type != teleportrock2.CommandAddMap {
			return
		}
		_ = teleport_rock.NewProcessor(l, ctx, db).AddMapAndEmit(c.TransactionId, c.WorldId, c.CharacterId, c.Body.MapId, c.Body.Vip)
	}
}

func handleRemoveMap(db *gorm.DB) message.Handler[teleportrock2.Command[teleportrock2.RemoveMapCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c teleportrock2.Command[teleportrock2.RemoveMapCommandBody]) {
		if c.Type != teleportrock2.CommandRemoveMap {
			return
		}
		_ = teleport_rock.NewProcessor(l, ctx, db).RemoveMapAndEmit(c.TransactionId, c.WorldId, c.CharacterId, c.Body.MapId, c.Body.Vip)
	}
}
```

- [ ] **Step 2: Write the type-guard test**

`consumer_test.go` — assert the handlers ignore mismatched command types (the standard guard test; sqlite DB, no Kafka needed):

```go
package teleportrock

import (
	teleportrock2 "atlas-character/kafka/message/teleportrock"
	"atlas-character/teleport_rock"
	"context"
	"testing"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func testDB(t *testing.T) *gorm.DB {
	l, _ := test.NewNullLogger()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	database.RegisterTenantCallbacks(l, db)
	if err := teleport_rock.Migration(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

// A command with the wrong Type must be a no-op (each handler receives every
// message on the topic).
func TestHandleAddMapIgnoresWrongType(t *testing.T) {
	db := testDB(t)
	l, _ := test.NewNullLogger()
	tm, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), tm)

	handleAddMap(db)(l, ctx, teleportrock2.Command[teleportrock2.AddMapCommandBody]{
		Type: teleportrock2.CommandRemoveMap, // wrong type for this handler
		Body: teleportrock2.AddMapCommandBody{MapId: 100000000, Vip: false},
	})

	m, err := teleport_rock.NewProcessor(l, ctx, db).GetByCharacterId(0)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(m.Regular()) != 0 {
		t.Fatalf("wrong-type command must not mutate: %v", m.Regular())
	}
}
```

Note: `AddMapAndEmit` inside the correct-type path requires a live Kafka producer; do NOT test the happy path here (it is covered at the processor layer in Task 9). If `message.Emit`'s producer would panic in tests, the wrong-type guard test above never reaches it.

- [ ] **Step 3: Run tests**

Run: `(cd services/atlas-character/atlas.com/character && go test ./kafka/consumer/teleportrock/...)`
Expected: FAIL first (missing consumer.go), then PASS after Step 1. (Order steps 1–2 as test-first if practical; the handler bodies are thin adapters so file-then-test is acceptable here.)

- [ ] **Step 4: Wire into `main.go`**

In `services/atlas-character/atlas.com/character/main.go`, add import:

```go
	teleportrock2 "atlas-character/kafka/consumer/teleportrock"
```

Inside the `service.GetMode() == service.Mixed` block, after `drop.InitConsumers(...)` (line 75) add:

```go
		teleportrock2.InitConsumers(l)(cmf)(consumerGroupId)
```

and after the `drop.InitHandlers` guard (lines 85–87) add:

```go
		if err := teleportrock2.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil {
			l.WithError(err).Fatal("Unable to register kafka handlers.")
		}
```

Run: `(cd services/atlas-character/atlas.com/character && go build ./...)` — clean.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-character/atlas.com/character/kafka/consumer/teleportrock/ services/atlas-character/atlas.com/character/main.go
git commit -m "feat(task-124): teleport-rock command consumer"
```

---

### Task 11: atlas-character — REST resource

**Files:**
- Create: `services/atlas-character/atlas.com/character/teleport_rock/rest.go`
- Create: `services/atlas-character/atlas.com/character/teleport_rock/resource.go`
- Modify: `services/atlas-character/atlas.com/character/main.go` (route init, line ~100)
- Test: `services/atlas-character/atlas.com/character/teleport_rock/rest_test.go`

**Interfaces:**
- Consumes: Task 8/9 model + processor; `atlas-character/rest` handler plumbing (`saved_location/resource.go` shape).
- Produces: `GET /characters/{characterId}/teleport-rock-maps` returning `RestModel{Id string; Regular []_map.Id; Vip []_map.Id}`, `GetName() = "teleport-rock-maps"`; `Transform(m Model) (RestModel, error)`; `InitResource(si)(db)`. Channel-side Task 13 consumes the wire shape.

- [ ] **Step 1: Write the failing transform test**

```go
package teleport_rock

import (
	"testing"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

func TestTransform(t *testing.T) {
	m := NewBuilder().
		SetCharacterId(42).
		SetRegular([]_map.Id{100000000}).
		SetVip([]_map.Id{104040000, 220000000}).
		Build()
	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if rm.GetName() != "teleport-rock-maps" {
		t.Fatalf("resource name: %s", rm.GetName())
	}
	if rm.GetID() != "42" {
		t.Fatalf("id: %s", rm.GetID())
	}
	if len(rm.Regular) != 1 || len(rm.Vip) != 2 {
		t.Fatalf("lists: %+v", rm)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `(cd services/atlas-character/atlas.com/character && go test ./teleport_rock/ -run TestTransform)`
Expected: FAIL — `Transform` undefined.

- [ ] **Step 3: Implement `rest.go`**

```go
package teleport_rock

import (
	"strconv"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

// RestModel is the read-side JSON:API resource: both lists, unpadded (wire
// padding to EmptyMapId is the packet codec's job, not the API's — PRD §5).
type RestModel struct {
	Id      string    `json:"-"`
	Regular []_map.Id `json:"regular"`
	Vip     []_map.Id `json:"vip"`
}

func (r RestModel) GetName() string {
	return "teleport-rock-maps"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:      strconv.FormatUint(uint64(m.CharacterId()), 10),
		Regular: m.Regular(),
		Vip:     m.Vip(),
	}, nil
}

func Extract(rm RestModel) (Model, error) {
	characterId, err := strconv.ParseUint(rm.Id, 10, 32)
	if err != nil {
		characterId = 0
	}
	return NewBuilder().
		SetCharacterId(uint32(characterId)).
		SetRegular(rm.Regular).
		SetVip(rm.Vip).
		Build(), nil
}
```

- [ ] **Step 4: Implement `resource.go`**

Mirror `saved_location/resource.go:16-26,40-67` (GET-only):

```go
package teleport_rock

import (
	"atlas-character/rest"
	"net/http"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(db)(si)
			r := router.PathPrefix("/characters/{characterId}/teleport-rock-maps").Subrouter()
			r.HandleFunc("", registerGet("get_teleport_rock_maps", handleGetTeleportRockMaps)).Methods(http.MethodGet)
		}
	}
}

func handleGetTeleportRockMaps(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			m, err := NewProcessor(d.Logger(), d.Context(), d.DB()).GetByCharacterId(characterId)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			res, err := model.Map(Transform)(model.FixedProvider(m))()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
		}
	})
}
```

(Empty lists are a valid 200 — `GetByCharacterId` returns an empty Model, never `gorm.ErrRecordNotFound`, so no 404 branch. FR-8: new characters start empty.)

- [ ] **Step 5: Register the route in `main.go`**

After `.AddRouteInitializer(saved_location.InitResource(GetServer())(db)).` (line 100) add:

```go
		AddRouteInitializer(teleport_rock.InitResource(GetServer())(db)).
```

- [ ] **Step 6: Run tests and build**

Run:
```bash
(cd services/atlas-character/atlas.com/character && go test ./teleport_rock/... && go build ./...)
```
Expected: PASS / clean.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-character/atlas.com/character/teleport_rock/ services/atlas-character/atlas.com/character/main.go
git commit -m "feat(task-124): teleport-rock-maps REST resource"
```

---

### Task 12: atlas-character — character-deletion cleanup + mock

**Files:**
- Modify: `services/atlas-character/atlas.com/character/character/processor.go` (Delete, lines 304–325)
- Create: `services/atlas-character/atlas.com/character/teleport_rock/mock/processor.go`

**Interfaces:**
- Consumes: `teleport_rock.DeleteForCharacter` (Task 8), Task 9 `Processor` interface.
- Produces: character delete removes `teleport_rock_maps` rows in the same transaction; `mock.ProcessorMock` implementing `teleport_rock.Processor` (func-field convention per `services/atlas-fame/atlas.com/fame/character/mock/processor.go`).

- [ ] **Step 1: Add the cleanup to `character.Delete`**

In `character/processor.go` `Delete` (line 304), inside the transaction after `delete(tx, characterId)` succeeds (line 312–315) and before the `mb.Put`, add:

```go
			if err = teleport_rock.DeleteForCharacter(tx, tenant.MustFromContext(p.ctx).Id(), characterId); err != nil {
				return err
			}
```

Add import `"atlas-character/teleport_rock"`. If `ProcessorImpl` already carries a tenant field (check the struct), use it instead of re-parsing the context. atlas-character has no other sub-domain cleanup today (verified design gap) — this is deliberately the first; do NOT also delete saved_locations here (out of scope).

- [ ] **Step 2: Extend the administrator test to cover the lifecycle**

The direct `DeleteForCharacter` unit test already exists (Task 8). Add one test in `character/` only if an existing character-delete test file covers sub-resources; otherwise rely on the Task 8 test — do not build a new cross-domain harness for this.

Run: `(cd services/atlas-character/atlas.com/character && go test ./...)`
Expected: PASS (all packages).

- [ ] **Step 3: Write `teleport_rock/mock/processor.go`**

```go
package mock

import (
	"atlas-character/kafka/message"
	"atlas-character/teleport_rock"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

// ProcessorMock is the func-field mock for teleport_rock.Processor (fame/notes
// convention — atlas-character's first standard mock).
type ProcessorMock struct {
	GetByCharacterIdFunc func(characterId uint32) (teleport_rock.Model, error)
	AddMapFunc           func(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) error
	AddMapAndEmitFunc    func(transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) error
	RemoveMapFunc        func(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) error
	RemoveMapAndEmitFunc func(transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) error
}

func (m *ProcessorMock) GetByCharacterId(characterId uint32) (teleport_rock.Model, error) {
	if m.GetByCharacterIdFunc != nil {
		return m.GetByCharacterIdFunc(characterId)
	}
	return teleport_rock.Model{}, nil
}

func (m *ProcessorMock) AddMap(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) error {
	if m.AddMapFunc != nil {
		return m.AddMapFunc(mb)
	}
	return func(uuid.UUID, world.Id, uint32, _map.Id, bool) error { return nil }
}

func (m *ProcessorMock) AddMapAndEmit(transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) error {
	if m.AddMapAndEmitFunc != nil {
		return m.AddMapAndEmitFunc(transactionId, worldId, characterId, mapId, vip)
	}
	return nil
}

func (m *ProcessorMock) RemoveMap(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) error {
	if m.RemoveMapFunc != nil {
		return m.RemoveMapFunc(mb)
	}
	return func(uuid.UUID, world.Id, uint32, _map.Id, bool) error { return nil }
}

func (m *ProcessorMock) RemoveMapAndEmit(transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) error {
	if m.RemoveMapAndEmitFunc != nil {
		return m.RemoveMapAndEmitFunc(transactionId, worldId, characterId, mapId, vip)
	}
	return nil
}

var _ teleport_rock.Processor = (*ProcessorMock)(nil)
```

- [ ] **Step 4: Verify + commit**

```bash
(cd services/atlas-character/atlas.com/character && go test -race ./... && go vet ./... && go build ./...)
git add services/atlas-character/atlas.com/character/character/processor.go services/atlas-character/atlas.com/character/teleport_rock/mock/
git commit -m "feat(task-124): teleport-rock cleanup on character delete + mock processor"
```

---

### Task 13: atlas-channel — message package + character/teleportrock read+command package

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/kafka/message/teleportrock/kafka.go`
- Create: `services/atlas-channel/atlas.com/channel/character/teleportrock/model.go`
- Create: `services/atlas-channel/atlas.com/channel/character/teleportrock/rest.go`
- Create: `services/atlas-channel/atlas.com/channel/character/teleportrock/requests.go`
- Create: `services/atlas-channel/atlas.com/channel/character/teleportrock/producer.go`
- Create: `services/atlas-channel/atlas.com/channel/character/teleportrock/processor.go`
- Test: `services/atlas-channel/atlas.com/channel/character/teleportrock/model_test.go`

**Interfaces:**
- Consumes: atlas-character REST wire shape (Task 11) and Kafka contracts (Task 9); `character/key` package shape (`requests.go:9-20`, `processor.go`); `atlas-channel/kafka/producer`.
- Produces (used by Tasks 14–19):
  - `kafka/message/teleportrock`: byte-for-byte mirror of the atlas-character message package from Task 9 Step 1 — same consts, `Command[E]`, `StatusEvent[E]`, bodies. Copy that file verbatim (package docs may differ).
  - `character/teleportrock`: `Model` (`Regular()/Vip()/List(vip)/Contains(vip, mapId)` over `[]_map.Id`), `Processor` interface with `GetByCharacterId(characterId uint32) (Model, error)`, `RequestAddMap(f field.Model, characterId uint32, vip bool) error`, `RequestRemoveMap(worldId world.Id, characterId uint32, mapId _map.Id, vip bool) error`; `NewProcessor(l, ctx) Processor`.

- [ ] **Step 1: Create the channel message package**

Copy Task 9 Step 1's `kafka.go` verbatim into `services/atlas-channel/atlas.com/channel/kafka/message/teleportrock/kafka.go` (same package name `teleportrock`; the two services deliberately share the wire contract).

- [ ] **Step 2: Write the model + failing test**

`model.go`:

```go
package teleportrock

import (
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

// Model is the channel-side read model of both saved-map lists (unpadded).
type Model struct {
	regular []_map.Id
	vip     []_map.Id
}

func NewModel(regular []_map.Id, vip []_map.Id) Model {
	return Model{regular: regular, vip: vip}
}

func (m Model) Regular() []_map.Id { return m.regular }
func (m Model) Vip() []_map.Id     { return m.vip }

func (m Model) List(vip bool) []_map.Id {
	if vip {
		return m.vip
	}
	return m.regular
}

func (m Model) Contains(vip bool, mapId _map.Id) bool {
	for _, v := range m.List(vip) {
		if v == mapId {
			return true
		}
	}
	return false
}
```

`model_test.go`:

```go
package teleportrock

import (
	"testing"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

func TestModelContains(t *testing.T) {
	m := NewModel([]_map.Id{100000000}, []_map.Id{104040000})
	if !m.Contains(false, 100000000) || m.Contains(false, 104040000) {
		t.Fatalf("regular membership wrong")
	}
	if !m.Contains(true, 104040000) || m.Contains(true, 100000000) {
		t.Fatalf("vip membership wrong")
	}
}
```

- [ ] **Step 3: Write `rest.go` + `requests.go`**

`rest.go`:

```go
package teleportrock

import (
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

type RestModel struct {
	Id      string    `json:"-"`
	Regular []_map.Id `json:"regular"`
	Vip     []_map.Id `json:"vip"`
}

func (r RestModel) GetName() string {
	return "teleport-rock-maps"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

func Extract(rm RestModel) (Model, error) {
	return NewModel(rm.Regular, rm.Vip), nil
}
```

`requests.go` (mirror `character/key/requests.go:9-20`, root URL `CHARACTERS` like `character/requests.go:17`):

```go
package teleportrock

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource = "characters/%d/teleport-rock-maps"
)

func getBaseRequest() string {
	return requests.RootUrl("CHARACTERS")
}

func requestByCharacterId(characterId uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+Resource, characterId))
}
```

- [ ] **Step 4: Write `producer.go` + `processor.go`**

`producer.go`:

```go
package teleportrock

import (
	teleportrock2 "atlas-channel/kafka/message/teleportrock"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func addMapCommandProvider(transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &teleportrock2.Command[teleportrock2.AddMapCommandBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		CharacterId:   characterId,
		Type:          teleportrock2.CommandAddMap,
		Body:          teleportrock2.AddMapCommandBody{MapId: mapId, Vip: vip},
	}
	return producer.SingleMessageProvider(key, value)
}

func removeMapCommandProvider(transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &teleportrock2.Command[teleportrock2.RemoveMapCommandBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		CharacterId:   characterId,
		Type:          teleportrock2.CommandRemoveMap,
		Body:          teleportrock2.RemoveMapCommandBody{MapId: mapId, Vip: vip},
	}
	return producer.SingleMessageProvider(key, value)
}
```

`processor.go`:

```go
package teleportrock

import (
	teleportrock2 "atlas-channel/kafka/message/teleportrock"
	"atlas-channel/kafka/producer"
	"context"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	GetByCharacterId(characterId uint32) (Model, error)
	RequestAddMap(f field.Model, characterId uint32, vip bool) error
	RequestRemoveMap(worldId world.Id, characterId uint32, mapId _map.Id, vip bool) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

func (p *ProcessorImpl) GetByCharacterId(characterId uint32) (Model, error) {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestByCharacterId(characterId), Extract)()
}

// RequestAddMap registers the character's CURRENT map (server-derived from
// session state — the client sends no map id on register, design §1 Q1).
func (p *ProcessorImpl) RequestAddMap(f field.Model, characterId uint32, vip bool) error {
	return producer.ProviderImpl(p.l)(p.ctx)(teleportrock2.EnvCommandTopic)(addMapCommandProvider(uuid.New(), f.WorldId(), characterId, f.MapId(), vip))
}

func (p *ProcessorImpl) RequestRemoveMap(worldId world.Id, characterId uint32, mapId _map.Id, vip bool) error {
	return producer.ProviderImpl(p.l)(p.ctx)(teleportrock2.EnvCommandTopic)(removeMapCommandProvider(uuid.New(), worldId, characterId, mapId, vip))
}
```

(`requests.Provider[RestModel, Model](p.l, p.ctx)(request, Extract)()` is the established single-resource idiom — see `services/atlas-channel/atlas.com/channel/merchant/processor.go:33`.)

- [ ] **Step 5: Test + build**

Run: `(cd services/atlas-channel/atlas.com/channel && go test ./character/teleportrock/... && go build ./...)`
Expected: PASS / clean.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/message/teleportrock/ services/atlas-channel/atlas.com/channel/character/teleportrock/
git commit -m "feat(task-124): channel-side teleport-rock read model and command producers"
```

---

### Task 14: atlas-channel — thread real lists into the character-data writer

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/writer/character_data.go:16`
- Modify: `services/atlas-channel/atlas.com/channel/socket/writer/set_field.go:28-35`
- Modify: `services/atlas-channel/atlas.com/channel/socket/writer/cash_shop_open.go:16-23`
- Modify: `services/atlas-channel/atlas.com/channel/socket/writer/character_data_test.go:26` (and any other `BuildCharacterData` call sites — grep first)

**Interfaces:**
- Consumes: `charpkt.CharacterData.TeleportMaps/VipTeleportMaps` (Task 7), `character/teleportrock.NewProcessor(l, ctx).GetByCharacterId` (Task 13).
- Produces: `BuildCharacterData(c character.Model, bl buddylist.Model, mapId _map.Id, trm teleportrock.Model) charpkt.CharacterData` — FR-16.

Note: `npm`-style build caveat does not apply, but the Go analog does — `character_data_test.go` compiles with the package, so the signature change and the test call-site fix land in the SAME commit.

- [ ] **Step 1: Update `BuildCharacterData`**

In `character_data.go`, change the signature and add the field population:

```go
import (
	// ... existing imports ...
	"atlas-channel/character/teleportrock"
)

func BuildCharacterData(c character.Model, bl buddylist.Model, mapId _map.Id, trm teleportrock.Model) charpkt.CharacterData {
```

and after the `Meso:` initialization block (line 45) add:

```go
	// Saved teleport-rock lists (FR-15/16). Codec pads to 5/10 with EmptyMapId.
	cd.TeleportMaps = trm.Regular()
	cd.VipTeleportMaps = trm.Vip()
```

- [ ] **Step 2: Update the two body call sites (fetch, fail-open)**

`set_field.go` `SetFieldBody` (line 28):

```go
func SetFieldBody(channelId channel.Id, c character.Model, bl buddylist.Model) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			trm, err := teleportrock.NewProcessor(l, ctx).GetByCharacterId(c.Id())
			if err != nil {
				// Fail-open: a missing list must never block login (design §4.4).
				l.WithError(err).Warnf("Unable to fetch teleport-rock maps for character [%d]; sending empty lists.", c.Id())
				trm = teleportrock.Model{}
			}
			cd := BuildCharacterData(c, bl, location.ResolveMapId(l, ctx, c.Id()), trm)
			return fieldcb.NewSetField(channelId, cd).Encode(l, ctx)(options)
		}
	}
}
```

Add import `"atlas-channel/character/teleportrock"`. Apply the identical fetch + fail-open pattern in `cash_shop_open.go` `CashShopOpenBody` (line 16).

- [ ] **Step 3: Fix remaining call sites**

Run: `grep -rn "BuildCharacterData" services/atlas-channel/atlas.com/channel/` — update every caller. Known: `character_data_test.go:26` becomes:

```go
	cd := BuildCharacterData(c, buddylist.Model{}, _map.Id(0), teleportrock.Model{})
```

- [ ] **Step 4: Add a writer test for the threading**

Append to `character_data_test.go`:

```go
func TestBuildCharacterData_TeleportMaps(t *testing.T) {
	c := character.Model{} // zero-value character is enough for field threading
	trm := teleportrock.NewModel([]_map.Id{100000000}, []_map.Id{104040000, 220000000})
	cd := BuildCharacterData(c, buddylist.Model{}, _map.Id(0), trm)
	if len(cd.TeleportMaps) != 1 || cd.TeleportMaps[0] != 100000000 {
		t.Fatalf("teleport maps: %v", cd.TeleportMaps)
	}
	if len(cd.VipTeleportMaps) != 2 {
		t.Fatalf("vip maps: %v", cd.VipTeleportMaps)
	}
}
```

(If `character.Model{}` cannot be constructed bare in the existing test file, reuse whatever builder `TestBuildCharacterData_MonsterBook` uses at line 14–26.)

- [ ] **Step 5: Test + build + commit**

```bash
(cd services/atlas-channel/atlas.com/channel && go test ./socket/writer/... && go build ./...)
git add services/atlas-channel/atlas.com/channel/socket/writer/
git commit -m "feat(task-124): character-data writer threads real teleport-rock lists"
```

---

### Task 15: atlas-channel — TROCK_ADD_MAP handler

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/teleport_rock_add_map.go`
- Modify: `services/atlas-channel/atlas.com/channel/main.go` (handlerMap, after line 801)
- Test: `services/atlas-channel/atlas.com/channel/socket/handler/teleport_rock_add_map_test.go`

**Interfaces:**
- Consumes: `trsb.AddMap` decoder (Task 4), `character/teleportrock.Processor` (Task 13), `session.Model` (`s.Field()`, `s.CharacterId()`).
- Produces: `TeleportRockAddMapHandleFunc(l, ctx, wp)` registered under `trsb.TeleportRockAddMapHandle`.

- [ ] **Step 1: Write the handler**

```go
package handler

import (
	"atlas-channel/character/teleportrock"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	trsb "github.com/Chronicle20/atlas/libs/atlas-packet/teleportrock/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

// teleportRockRequestsFunc allows tests to capture the emitted commands.
var teleportRockRequestsFunc = func(l logrus.FieldLogger, ctx context.Context) teleportrock.Processor {
	return teleportrock.NewProcessor(l, ctx)
}

// TeleportRockAddMapHandleFunc handles TROCK_ADD_MAP
// (CWvsContext::SendMapTransferRequest). Register carries no map id — the
// current map comes from server-side session state (design §1 Q1). All client
// feedback rides the status-event consumer (fire-and-forget here).
func TeleportRockAddMapHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := trsb.AddMap{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		proc := teleportRockRequestsFunc(l, ctx)
		if p.Register() {
			if err := proc.RequestAddMap(s.Field(), s.CharacterId(), p.Vip()); err != nil {
				l.WithError(err).Errorf("Unable to request map registration for character [%d].", s.CharacterId())
			}
			return
		}
		if err := proc.RequestRemoveMap(s.Field().WorldId(), s.CharacterId(), _map.Id(p.MapId()), p.Vip()); err != nil {
			l.WithError(err).Errorf("Unable to request map removal for character [%d].", s.CharacterId())
		}
	}
}
```

- [ ] **Step 2: Write the handler test**

Use a fake `teleportrock.Processor` injected via `teleportRockRequestsFunc` (the package-var injection precedent is `mystic_door_enter.go:25-51`). Build a fake implementing the three methods that records calls; feed the handler a decoded register packet (`[]byte{0x01, 0x01}`) and a delete packet, and assert:
- register → `RequestAddMap` called once with `mapId == s.Field().MapId()` and `vip == true` (map id came from the session, not the wire);
- delete → `RequestRemoveMap` called with the decoded map id.

Constructing a `session.Model` with a field: reuse the idiom from an existing handler test in this package (grep `session.Model{}` / builder usage under `socket/handler/*_test.go`, e.g. the mystic-door tests). If no handler test constructs a session, use `session.NewBuilder()...` per `session/model.go`; adapt but keep the two assertions.

- [ ] **Step 3: Register in `main.go`**

Add import `trsb "github.com/Chronicle20/atlas/libs/atlas-packet/teleportrock/serverbound"` and after line 801 (`handlerMap[doorsb.EnterDoorHandle] = ...`):

```go
	handlerMap[trsb.TeleportRockAddMapHandle] = handler.TeleportRockAddMapHandleFunc
```

- [ ] **Step 4: Test + build + commit**

```bash
(cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/... && go build ./...)
git add services/atlas-channel/atlas.com/channel/socket/handler/teleport_rock_add_map.go services/atlas-channel/atlas.com/channel/socket/handler/teleport_rock_add_map_test.go services/atlas-channel/atlas.com/channel/main.go
git commit -m "feat(task-124): TROCK_ADD_MAP handler"
```

---

### Task 16: atlas-channel — status consumer + writer registration

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/kafka/consumer/teleportrock/consumer.go`
- Modify: `services/atlas-channel/atlas.com/channel/main.go` (InitConsumers ~line 198, InitHandlers ~line 437, produceWriters ~line 788)
- Test: `services/atlas-channel/atlas.com/channel/kafka/consumer/teleportrock/consumer_test.go`

**Interfaces:**
- Consumes: channel message package (Task 13), `trpkt` result bodies (Task 6), `session.Announce` / `session.NewProcessor(...).IfPresentByCharacterId` (messenger consumer shape, `kafka/consumer/messenger/consumer.go:27-107`), `server.Model` world/channel filtering.
- Produces: `teleportrock.InitConsumers` / `teleportrock.InitHandlers` wired in main; `MapTransferResult` writer name registered in `produceWriters()`.

- [ ] **Step 1: Write `consumer.go`**

```go
package teleportrock

import (
	consumer2 "atlas-channel/kafka/consumer"
	teleportrock2 "atlas-channel/kafka/message/teleportrock"
	"atlas-channel/listener"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	trpkt "github.com/Chronicle20/atlas/libs/atlas-packet/teleportrock"
	trcb "github.com/Chronicle20/atlas/libs/atlas-packet/teleportrock/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("teleport_rock_status_event")(teleportrock2.EnvEventTopicStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
				var t string
				var handles []listener.HandlerHandle
				t, _ = topic.EnvProvider(l)(teleportrock2.EnvEventTopicStatus)()
				id, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleListUpdated(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleError(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				return handles, nil
			}
		}
	}
}

// handleListUpdated projects LIST_UPDATED into the list-refresh
// MAP_TRANSFER_RESULT (mode REGISTER_LIST on add, DELETE_LIST on remove). The
// client only updates its UI from this packet (FR-7).
func handleListUpdated(sc server.Model, wp writer.Producer) message.Handler[teleportrock2.StatusEvent[teleportrock2.ListUpdatedStatusBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e teleportrock2.StatusEvent[teleportrock2.ListUpdatedStatusBody]) {
		if e.Type != teleportrock2.StatusEventTypeListUpdated {
			return
		}
		if !sc.IsWorld(tenant.MustFromContext(ctx), e.WorldId) {
			return
		}
		key := trpkt.MapTransferModeDeleteList
		if e.Body.Registered {
			key = trpkt.MapTransferModeRegisterList
		}
		body := trpkt.MapTransferResultListBody(key, e.Body.Vip, e.Body.Maps)
		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, session.Announce(l)(ctx)(wp)(trcb.MapTransferResultWriter)(body))
		if err != nil {
			l.WithError(err).Errorf("Unable to announce teleport-rock list update to character [%d].", e.CharacterId)
		}
	}
}

// handleError maps rejection reasons to the faithful client strings (design
// §4.2): LIST_FULL/DUPLICATE/MAP_NOT_ALLOWED -> MAP_NOT_AVAILABLE (the client
// prechecks full/duplicate itself; these fire only for bypassed clients);
// NOT_FOUND -> CANNOT_GO.
func handleError(sc server.Model, wp writer.Producer) message.Handler[teleportrock2.StatusEvent[teleportrock2.ErrorStatusBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e teleportrock2.StatusEvent[teleportrock2.ErrorStatusBody]) {
		if e.Type != teleportrock2.StatusEventTypeError {
			return
		}
		if !sc.IsWorld(tenant.MustFromContext(ctx), e.WorldId) {
			return
		}
		key := errorReasonToModeKey(e.Body.Reason)
		body := trpkt.MapTransferResultErrorBody(key, e.Body.Vip)
		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, session.Announce(l)(ctx)(wp)(trcb.MapTransferResultWriter)(body))
		if err != nil {
			l.WithError(err).Errorf("Unable to announce teleport-rock error to character [%d].", e.CharacterId)
		}
	}
}

func errorReasonToModeKey(reason string) string {
	switch reason {
	case teleportrock2.ErrorReasonNotFound:
		return trpkt.MapTransferModeCannotGo
	default: // LIST_FULL, DUPLICATE, MAP_NOT_ALLOWED
		return trpkt.MapTransferModeMapNotAvailable
	}
}
```

- [ ] **Step 2: Test the reason→mode mapping**

`consumer_test.go`:

```go
package teleportrock

import (
	teleportrock2 "atlas-channel/kafka/message/teleportrock"
	"testing"

	trpkt "github.com/Chronicle20/atlas/libs/atlas-packet/teleportrock"
)

func TestErrorReasonToModeKey(t *testing.T) {
	cases := map[string]string{
		teleportrock2.ErrorReasonListFull:      trpkt.MapTransferModeMapNotAvailable,
		teleportrock2.ErrorReasonDuplicate:     trpkt.MapTransferModeMapNotAvailable,
		teleportrock2.ErrorReasonMapNotAllowed: trpkt.MapTransferModeMapNotAvailable,
		teleportrock2.ErrorReasonNotFound:      trpkt.MapTransferModeCannotGo,
	}
	for reason, want := range cases {
		if got := errorReasonToModeKey(reason); got != want {
			t.Errorf("%s: got %s want %s", reason, got, want)
		}
	}
}
```

- [ ] **Step 3: Wire in `main.go`**

Import `teleportrockConsumer "atlas-channel/kafka/consumer/teleportrock"`. Add after the messenger-area InitConsumers block (~line 198):

```go
	teleportrockConsumer.InitConsumers(l)(cmf)(consumerGroupId)
```

Add in the InitHandlers register chain (~line 437, next to the other `register(...)` calls):

```go
		if err := register(teleportrockConsumer.InitHandlers(fl)(sc)(wp)(rh)); err != nil {
			return err
		}
```

(Match the exact `register` idiom of the surrounding lines 416–437.)

In `produceWriters()` (~line 788, next to `doorcb.RemoveDoorWriter`), add import `trcb "github.com/Chronicle20/atlas/libs/atlas-packet/teleportrock/clientbound"` and:

```go
		trcb.MapTransferResultWriter,
```

- [ ] **Step 4: Test + build + commit**

```bash
(cd services/atlas-channel/atlas.com/channel && go test ./kafka/consumer/teleportrock/... && go build ./...)
git add services/atlas-channel/atlas.com/channel/kafka/consumer/teleportrock/ services/atlas-channel/atlas.com/channel/main.go
git commit -m "feat(task-124): teleport-rock status consumer projecting MAP_TRANSFER_RESULT"
```

---

### Task 17: atlas-channel — use-flow (validate → saga)

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/teleportrock/use.go`
- Test: `services/atlas-channel/atlas.com/channel/teleportrock/use_test.go`

**Interfaces:**
- Consumes: `character/teleportrock.Model` (Task 13), `data/map` processor (`GetById(mapId) (Model, error)`, `Model.FieldLimit() uint32`), `character` processor (`GetByName`), `session` processor (`GetByCharacterId(ch)(id) (session.Model, error)`), `saga` (Task 1 re-exports), `trpkt` bodies (Task 6), `_map.FieldLimitNoTeleportItem|FieldLimitNoMysticDoor` (Task 1).
- Produces: `UseRock(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, itemId item.Id, target trpkt.Target)` — the single entry point both handlers call (Tasks 18–19).

**Validation chain** (design §4.3, in order; first failure announces the mode and stops — nothing is consumed, FR-1):

| # | Check | Failure mode key |
|---|---|---|
| 1 | source field: `fieldLimit & (0x40\|0x02) != 0` or map lookup error | `CANNOT_GO` |
| 2a | by-map: target in the rock's list (`useVipList := itemId/1000 == 5041`) | `CANNOT_GO` |
| 2b | by-name: character lookup or same-channel session lookup fails | `UNABLE_TO_LOCATE` |
| 3 | target map == current map | `CURRENT_MAP` |
| 4 | target field: `fieldLimit & (0x40\|0x02) != 0` or lookup error | `CANNOT_GO` |
| 5 | non-VIP rock (`itemId/1000 != 5041`): `continent(src) != continent(dst)` where `continent(m) = m/100000000` | `CANNOT_GO_CONTINENT` |

Success: saga `TeleportRockUse` with step 1 `WarpToRandomPortal{CharacterId, FieldId: targetField.Id()}` and, ONLY for the regular rock (`itemId/10000 == 232`), step 2 `DestroyAsset{CharacterId, TemplateId: itemId, Quantity: 1, RemoveAll: false}`. Warp-before-destroy ordering is FR-2. No success `MAP_TRANSFER_RESULT` is sent (design: SetField is the success signal).

- [ ] **Step 1: Write the failing table test**

```go
package teleportrock

import (
	"context"
	"testing"

	"atlas-channel/saga"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	trpkt "github.com/Chronicle20/atlas/libs/atlas-packet/teleportrock"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// Fixture (installed by installFixture, restored via t.Cleanup): character 42.
// Regular list contains 102000000; VIP list contains 220000000 (different
// continent, 2xx). fieldLimit: 103000000 -> 0x40 (rock-banned), else 0.
// characterByNameFunc: "Buddy" -> id 77 (session on map 102000000), else error.

func TestUseRockRejections(t *testing.T) {
	l, _ := testlog.NewNullLogger()

	type tc struct {
		name     string
		itemId   item.Id
		srcMap   _map.Id
		target   trpkt.Target
		wantMode string
	}
	cases := []tc{
		{"source field barred", 2320000, 103000000, trpkt.NewTargetByMap(102000000), trpkt.MapTransferModeCannotGo},
		{"target not in list", 2320000, 100000000, trpkt.NewTargetByMap(105000000), trpkt.MapTransferModeCannotGo},
		{"target is current map", 2320000, 102000000, trpkt.NewTargetByMap(102000000), trpkt.MapTransferModeCurrentMap},
		{"target field barred", 2320000, 100000000, trpkt.NewTargetByMap(103000000), trpkt.MapTransferModeCannotGo},
		{"continent mismatch regular", 5040000, 100000000, trpkt.NewTargetByMap(220000000), trpkt.MapTransferModeCannotGoContinent},
		{"player not found", 2320000, 100000000, trpkt.NewTargetByName("Ghost"), trpkt.MapTransferModeUnableToLocate},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var announced string
			var sagaCreated *saga.Saga
			installFixture(t, c.srcMap, &announced, &sagaCreated)

			UseRock(l, context.Background(), nil)(testSession(t, 42, c.srcMap), c.itemId, c.target)

			if announced != c.wantMode {
				t.Errorf("announced mode: got %q want %q", announced, c.wantMode)
			}
			if sagaCreated != nil {
				t.Errorf("failed validation must not create a saga (FR-1)")
			}
		})
	}
}

func TestUseRockSuccessRegularConsumes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	var announced string
	var sagaCreated *saga.Saga
	installFixture(t, 100000000, &announced, &sagaCreated)

	UseRock(l, context.Background(), nil)(testSession(t, 42, 100000000), 2320000, trpkt.NewTargetByMap(102000000))

	if announced != "" {
		t.Fatalf("success must not announce an error, got %q", announced)
	}
	if sagaCreated == nil {
		t.Fatalf("expected a saga")
	}
	if sagaCreated.SagaType != saga.TeleportRockUse {
		t.Errorf("saga type: %v", sagaCreated.SagaType)
	}
	if len(sagaCreated.Steps) != 2 {
		t.Fatalf("regular rock: warp + destroy, got %d steps", len(sagaCreated.Steps))
	}
	if sagaCreated.Steps[0].Action != saga.WarpToRandomPortal || sagaCreated.Steps[1].Action != saga.DestroyAsset {
		t.Errorf("step order must be warp-then-destroy (FR-2): %v, %v", sagaCreated.Steps[0].Action, sagaCreated.Steps[1].Action)
	}
}

func TestUseRockSuccessCashDoesNotConsume(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	var announced string
	var sagaCreated *saga.Saga
	installFixture(t, 100000000, &announced, &sagaCreated)

	// 5041000 uses the VIP list and skips the continent check.
	UseRock(l, context.Background(), nil)(testSession(t, 42, 100000000), 5041000, trpkt.NewTargetByMap(220000000))

	if sagaCreated == nil {
		t.Fatalf("expected a saga")
	}
	if len(sagaCreated.Steps) != 1 {
		t.Fatalf("cash rock: warp only, got %d steps", len(sagaCreated.Steps))
	}
}

func TestUseRockByNameSuccess(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	var announced string
	var sagaCreated *saga.Saga
	installFixture(t, 100000000, &announced, &sagaCreated)

	// "Buddy" resolves to a session on map 102000000 (fixture).
	UseRock(l, context.Background(), nil)(testSession(t, 42, 100000000), 2320000, trpkt.NewTargetByName("Buddy"))

	if announced != "" || sagaCreated == nil {
		t.Fatalf("expected warp saga, announced=%q", announced)
	}
}
```

Write the two helpers in the test file:
- `installFixture(t, srcMap, &announced, &sagaCreated)` — overrides the package vars (with `t.Cleanup` restore): `listsFunc` returns `chartrock.NewModel([]_map.Id{102000000}, []_map.Id{220000000})`; `mapLimitFunc` returns `0x40` for map 103000000 else `0`; `characterByNameFunc` resolves `"Buddy"` → id 77, anything else → error; `sessionByCharacterIdFunc` resolves id 77 → a session on map 102000000, else error; `createSagaFunc` captures into `sagaCreated`; `announceErrorFunc` captures the mode key into `announced`.
- `testSession(t, characterId, mapId)` — builds a `session.Model` for world 0 / channel 0 / the given map (reuse the session-construction idiom found in Task 15's test).

- [ ] **Step 2: Run test to verify it fails**

Run: `(cd services/atlas-channel/atlas.com/channel && go test ./teleportrock/...)`
Expected: FAIL — package/`UseRock` undefined.

- [ ] **Step 3: Implement `use.go`**

```go
package teleportrock

import (
	chartrock "atlas-channel/character/teleportrock"
	character2 "atlas-channel/character"
	datamap "atlas-channel/data/map"
	"atlas-channel/saga"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	trpkt "github.com/Chronicle20/atlas/libs/atlas-packet/teleportrock"
	trcb "github.com/Chronicle20/atlas/libs/atlas-packet/teleportrock/clientbound"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// rockUseBarMask are the fieldLimit bits that bar teleport-rock use on a map
// (client checks 0x40 and 0x02 on the source; the server also applies them to
// the target — design §1 Q2).
const rockUseBarMask = _map.FieldLimitNoTeleportItem | _map.FieldLimitNoMysticDoor

// Injection points for table tests (package-var precedent:
// socket/handler/mystic_door_enter.go:25-51).
var listsFunc = func(l logrus.FieldLogger, ctx context.Context, characterId uint32) (chartrock.Model, error) {
	return chartrock.NewProcessor(l, ctx).GetByCharacterId(characterId)
}

var mapLimitFunc = func(l logrus.FieldLogger, ctx context.Context, mapId _map.Id) (uint32, error) {
	m, err := datamap.NewProcessor(l, ctx).GetById(mapId)
	if err != nil {
		return 0, err
	}
	return m.FieldLimit(), nil
}

var characterByNameFunc = func(l logrus.FieldLogger, ctx context.Context, name string) (uint32, error) {
	c, err := character2.NewProcessor(l, ctx).GetByName(name)
	if err != nil {
		return 0, err
	}
	return c.Id(), nil
}

var sessionByCharacterIdFunc = func(l logrus.FieldLogger, ctx context.Context, s session.Model, characterId uint32) (field.Model, error) {
	target, err := session.NewProcessor(l, ctx).GetByCharacterId(s.Field().Channel())(characterId)
	if err != nil {
		return field.Model{}, err
	}
	return target.Field(), nil
}

var createSagaFunc = func(l logrus.FieldLogger, ctx context.Context, s saga.Saga) error {
	return saga.NewProcessor(l, ctx).Create(s)
}

var announceErrorFunc = func(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, s session.Model, key string, vip bool) {
	err := session.Announce(l)(ctx)(wp)(trcb.MapTransferResultWriter)(trpkt.MapTransferResultErrorBody(key, vip))(s)
	if err != nil {
		l.WithError(err).Errorf("Unable to announce teleport-rock rejection to character [%d].", s.CharacterId())
	}
}

func continent(mapId _map.Id) uint32 {
	return uint32(mapId) / 100000000
}

// UseRock validates and executes a teleport-rock warp for both entry ops
// (USE_TELEPORT_ROCK and the cash-item-use branch). The caller has already
// verified the item exists in the claimed slot. Validation failures announce
// the faithful MAP_TRANSFER_RESULT mode and consume nothing (FR-1); success
// launches a warp[-then-consume] saga (FR-2, design §4.3).
func UseRock(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, itemId item.Id, target trpkt.Target) {
	return func(s session.Model, itemId item.Id, target trpkt.Target) {
		// 5041xxx is the only VIP-list rock; 2320000/5040000/5040001 use the
		// regular list (client: bCanTransferContinent = nItemID/1000 != 5040,
		// evaluated only for 504x — design §1 Q5).
		useVipList := uint32(itemId)/1000 == 5041

		fail := func(key string) {
			announceErrorFunc(l, ctx, wp, s, key, useVipList)
		}

		// 1. Source field bar.
		srcLimit, err := mapLimitFunc(l, ctx, s.Field().MapId())
		if err != nil || srcLimit&rockUseBarMask != 0 {
			l.Debugf("Teleport rock: source map [%d] barred (limit=0x%x err=%v) for character [%d].", s.Field().MapId(), srcLimit, err, s.CharacterId())
			fail(trpkt.MapTransferModeCannotGo)
			return
		}

		// 2. Resolve the target map.
		var targetMapId _map.Id
		if target.ByName() {
			targetId, err := characterByNameFunc(l, ctx, target.TargetName())
			if err != nil {
				l.Debugf("Teleport rock: target [%s] not found for character [%d].", target.TargetName(), s.CharacterId())
				fail(trpkt.MapTransferModeUnableToLocate)
				return
			}
			tf, err := sessionByCharacterIdFunc(l, ctx, s, targetId)
			if err != nil {
				// Offline, other channel, or cash shop: same rejection (design §1 Q6).
				l.Debugf("Teleport rock: target [%s] (id %d) has no session on this channel.", target.TargetName(), targetId)
				fail(trpkt.MapTransferModeUnableToLocate)
				return
			}
			targetMapId = tf.MapId()
		} else {
			targetMapId = _map.Id(target.TargetMap())
			lists, err := listsFunc(l, ctx, s.CharacterId())
			if err != nil || !lists.Contains(useVipList, targetMapId) {
				l.Debugf("Teleport rock: map [%d] not in list (vip=%v err=%v) for character [%d].", targetMapId, useVipList, err, s.CharacterId())
				fail(trpkt.MapTransferModeCannotGo)
				return
			}
		}

		// 3. Same map.
		if targetMapId == s.Field().MapId() {
			fail(trpkt.MapTransferModeCurrentMap)
			return
		}

		// 4. Target field bar (server-side policy half of Q2).
		dstLimit, err := mapLimitFunc(l, ctx, targetMapId)
		if err != nil || dstLimit&rockUseBarMask != 0 {
			l.Debugf("Teleport rock: target map [%d] barred (limit=0x%x err=%v) for character [%d].", targetMapId, dstLimit, err, s.CharacterId())
			fail(trpkt.MapTransferModeCannotGo)
			return
		}

		// 5. Continent restriction for non-VIP rocks (server policy, design §1 Q3).
		if !useVipList && continent(s.Field().MapId()) != continent(targetMapId) {
			fail(trpkt.MapTransferModeCannotGoContinent)
			return
		}

		// Success: warp via random spawn portal; consume only the regular rock,
		// and only after the warp (FR-2).
		targetField := field.NewBuilder(s.Field().WorldId(), s.Field().ChannelId(), targetMapId).Build()
		now := time.Now()
		steps := []saga.Step{
			{
				StepId: "warp_to_target",
				Status: saga.Pending,
				Action: saga.WarpToRandomPortal,
				Payload: saga.WarpToRandomPortalPayload{
					CharacterId: s.CharacterId(),
					FieldId:     targetField.Id(),
				},
				CreatedAt: now,
				UpdatedAt: now,
			},
		}
		if uint32(itemId)/10000 == 232 {
			steps = append(steps, saga.Step{
				StepId: "consume_rock",
				Status: saga.Pending,
				Action: saga.DestroyAsset,
				Payload: saga.DestroyAssetPayload{
					CharacterId: s.CharacterId(),
					TemplateId:  uint32(itemId),
					Quantity:    1,
					RemoveAll:   false,
				},
				CreatedAt: now,
				UpdatedAt: now,
			})
		}
		err = createSagaFunc(l, ctx, saga.Saga{
			TransactionId: uuid.New(),
			SagaType:      saga.TeleportRockUse,
			InitiatedBy:   "TELEPORT_ROCK_USE",
			Steps:         steps,
		})
		if err != nil {
			l.WithError(err).Errorf("Unable to create teleport-rock saga for character [%d].", s.CharacterId())
		}
	}
}
```

(Verify `saga.Saga` field names against `libs/atlas-saga/model.go:160-170` — `TransactionId/SagaType/InitiatedBy/Steps` per the cash handler at `character_cash_item_use.go:99-104`.)

- [ ] **Step 4: Run test to verify it passes**

Run: `(cd services/atlas-channel/atlas.com/channel && go test ./teleportrock/...)`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/teleportrock/
git commit -m "feat(task-124): teleport-rock use-flow validation and warp saga"
```

---

### Task 18: atlas-channel — USE_TELEPORT_ROCK handler

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/teleport_rock_use.go`
- Modify: `services/atlas-channel/atlas.com/channel/main.go` (handlerMap)
- Test: `services/atlas-channel/atlas.com/channel/socket/handler/teleport_rock_use_test.go`

**Interfaces:**
- Consumes: `trsb.Use` (Task 3), `teleportrock.UseRock` (Task 17), channel `character` processor `GetItemInSlot`.
- Produces: `TeleportRockUseHandleFunc` registered under `trsb.TeleportRockUseHandle`.

- [ ] **Step 1: Write the handler**

```go
package handler

import (
	character2 "atlas-channel/character"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"atlas-channel/teleportrock"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	trsb "github.com/Chronicle20/atlas/libs/atlas-packet/teleportrock/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

// itemInSlotFunc is a test seam for the inventory ownership check (package-var
// injection precedent: mystic_door_enter.go:25-51). Returns the template id of
// the USE-inventory item in the slot. GetItemInSlot returns an asset whose
// TemplateId() is compared — see character_cash_item_use.go:37.
var itemInSlotFunc = func(l logrus.FieldLogger, ctx context.Context, characterId uint32, slot int16) (uint32, error) {
	a, err := character2.NewProcessor(l, ctx).GetItemInSlot(characterId, inventory.TypeValueUse, slot)()
	if err != nil {
		return 0, err
	}
	return uint32(a.TemplateId()), nil
}

// useRockFunc is a test seam over the shared use-flow — handler tests in this
// package cannot reach the unexported seams inside atlas-channel/teleportrock,
// so they capture the invocation here instead. Shared with the cash branch
// (Task 19).
var useRockFunc = func(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, itemId item.Id, target trpkt.Target) {
	return teleportrock.UseRock(l, ctx, wp)
}

// TeleportRockUseHandleFunc handles USE_TELEPORT_ROCK
// (CWvsContext::SendMapTransferItemUseRequest). Only the regular USE rock
// (232xxxx) arrives on this op — cash rocks ride CASH_ITEM_USE (design §1 Q1).
func TeleportRockUseHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := trsb.Use{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		if !p.Valid() {
			// Client omitted the target payload (dialog closed without a
			// selection) — malformed by the client's own rules; drop silently.
			l.Warnf("Character [%d] sent USE_TELEPORT_ROCK without a target payload.", s.CharacterId())
			return
		}

		// Mirror the client guard: this op only carries 232xxxx.
		if p.ItemId()/10000 != 232 {
			l.Warnf("Character [%d] sent USE_TELEPORT_ROCK with non-rock item [%d].", s.CharacterId(), p.ItemId())
			return
		}

		// Verify the claimed slot actually holds the claimed item.
		templateId, err := itemInSlotFunc(l, ctx, s.CharacterId(), p.Slot())
		if err != nil || templateId != p.ItemId() {
			l.Warnf("Character [%d] attempted to use rock [%d] in slot [%d], but item not found or mismatched.", s.CharacterId(), p.ItemId(), p.Slot())
			return
		}

		useRockFunc(l, ctx, wp)(s, item.Id(p.ItemId()), p.Target())
	}
}
```

Add import `trpkt "github.com/Chronicle20/atlas/libs/atlas-packet/teleportrock"` for the seam's `Target` parameter type.

- [ ] **Step 2: Write the handler test**

Override both seams (with `t.Cleanup` restore): `itemInSlotFunc` returns 2320000 for slot 2 / errors otherwise; `useRockFunc` records `(itemId, target)` into captured variables. Cases:
1. Valid by-map request bytes (Task 3's `TestUseByMapDecode` payload) with matching item → `useRockFunc` invoked with itemId 2320000 and targetMap 100000000.
2. Slot mismatch (itemInSlotFunc returns 2320001) → not invoked.
3. Absent target payload bytes → not invoked.
4. Non-232 item id on this op → not invoked.

- [ ] **Step 3: Register in `main.go`**

After the Task 15 line:

```go
	handlerMap[trsb.TeleportRockUseHandle] = handler.TeleportRockUseHandleFunc
```

- [ ] **Step 4: Test + build + commit**

```bash
(cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/... && go build ./...)
git add services/atlas-channel/atlas.com/channel/socket/handler/teleport_rock_use.go services/atlas-channel/atlas.com/channel/socket/handler/teleport_rock_use_test.go services/atlas-channel/atlas.com/channel/main.go
git commit -m "feat(task-124): USE_TELEPORT_ROCK handler"
```

---

### Task 19: atlas-channel — cash item-use type-12 branch

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go` (lines 25, 60-106, 116-120)
- Test: `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_test.go` (create or extend)

**Interfaces:**
- Consumes: `cashsb.NewItemUseTeleportRock` (Task 5), `teleportrock.UseRock` (Task 17), `item.ClassificationTeleportRock` (= 504, `libs/atlas-constants/item/constants.go:73`).
- Produces: cash rocks (504x, classification 504) route into the use-flow; megaphone enum-12 aliases (`GetCashSlotItemType` maps some megaphones to 12 at line 179) keep falling through to warn-and-drop.

- [ ] **Step 1: Name the enum value and un-discard the writer producer**

In `character_cash_item_use.go`:

- Line 25: change `_ writer.Producer` to `wp writer.Producer`.
- In the const block (lines 116–120) add:

```go
	CashSlotItemTypeTeleportRock  = CashSlotItemType(12)
```

- [ ] **Step 2: Add the branch (before the fall-through warn at line 110)**

After the `CashSlotItemTypeFieldEffect` block:

```go
		if it == CashSlotItemTypeTeleportRock {
			// Enum 12 is shared: teleport rocks (classification 504) AND some
			// megaphones alias here (GetCashSlotItemType line ~179). Only the
			// rocks are implemented; megaphones keep the warn-and-drop path.
			if item.GetClassification(itemId) == item.ClassificationTeleportRock {
				sp := cashsb.NewItemUseTeleportRock(updateTimeFirst)
				sp.Decode(l, ctx)(r, readerOptions)
				if !sp.Target().Valid() {
					l.Warnf("Character [%d] sent cash teleport-rock use without a target payload.", s.CharacterId())
					return
				}
				useRockFunc(l, ctx, wp)(s, itemId, sp.Target())
				return
			}
		}
```

Add import `"atlas-channel/teleportrock"` only if not already pulled in by Task 18's seam file (`useRockFunc` lives in `teleport_rock_use.go`, same package — reuse it, do not redefine).

- [ ] **Step 3: Write the disambiguation test**

Override `useRockFunc` to record `(itemId, target)`. Cases:
1. itemId 5040000 (classification 504) with a valid by-map payload → `useRockFunc` invoked with itemId 5040000.
2. itemId 5041000 → invoked with itemId 5041000 (list selection is Task 17's concern, already table-tested).
3. A megaphone id that maps to enum 12 (pick one where `GetClassification(itemId) == item.ClassificationMegaphones` and `(itemId%10000)/1000 == 1`, e.g. 5071000 — VERIFY against `GetCashSlotItemType` line ~176-181 before pinning the constant) → `useRockFunc` NOT invoked; warn-and-drop reached.
4. Rock payload with absent target → not invoked.

Feed the handler a synthetic reader: common prefix bytes (v83: `short source`, `int itemId`) + rock payload. The handler also calls `GetItemInSlot` (line 37) — the existing cash-inventory check needs a seam too: refactor line 37 onto a package var in the same style (e.g. `cashItemInSlotFunc`, returning the asset template id for `inventory.TypeValueCash`) in the same commit, and have the test return the matching item id.

- [ ] **Step 4: Test + build + commit**

```bash
(cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/... && go build ./...)
git add services/atlas-channel/atlas.com/channel/socket/handler/
git commit -m "feat(task-124): route cash teleport rocks into the use-flow"
```

---

### Task 20: seed templates — handlers, writer, operations (all six versions)

**Files:**
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_83_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_84_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_87_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_92_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_95_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_jms_185_1.json`

**Interfaces:**
- Consumes: handler names `TeleportRockUseHandle` / `TeleportRockAddMapHandle` (Tasks 3–4), writer name `MapTransferResult` + mode keys (Task 6).
- Produces: live socket wiring at tenant-creation time (FR-17/18).

Opcodes (design §8; sources: registry YAMLs + CSV lineage; v83/v95 serverbound IDA-confirmed):

| Version | USE_TELEPORT_ROCK | TROCK_ADD_MAP | MAP_TRANSFER_RESULT |
|---|---|---|---|
| gms_83 | 0x54 | 0x66 | 0x2A |
| gms_84 | 0x54 | 0x66 | 0x2A |
| gms_87 | 0x57 | 0x69 | 0x2A |
| gms_92 | 0x5B | 0x71 | 0x2B |
| gms_95 | 0x5B | 0x72 | 0x29 |
| jms_185 | 0x4C | 0x61 | 0x27 |

- [ ] **Step 1: Add two handler rows per template** (in `socket.handlers`, keep neighbors' opcode ordering):

For gms_83 (adjust `opCode` per the table for the others):

```json
{"opCode": "0x54", "validator": "LoggedInValidator", "handler": "TeleportRockUseHandle"},
{"opCode": "0x66", "validator": "LoggedInValidator", "handler": "TeleportRockAddMapHandle"}
```

Every row MUST have the validator — validator-less entries are silently dropped (known failure class).

- [ ] **Step 2: Add the writer row with the full nine-key operations table per template** (in `socket.writers`; mode values identical across versions per design §1 Q4 — v84/v87/jms re-confirmed during Task 22):

For gms_83 (adjust `opCode` per the table):

```json
{"opCode": "0x2A", "writer": "MapTransferResult", "options": {"operations": {
  "DELETE_LIST": "0x02",
  "REGISTER_LIST": "0x03",
  "CANNOT_GO": "0x05",
  "UNABLE_TO_LOCATE": "0x06",
  "UNABLE_TO_LOCATE_2": "0x07",
  "CANNOT_GO_CONTINENT": "0x08",
  "CURRENT_MAP": "0x09",
  "MAP_NOT_AVAILABLE": "0x0A",
  "MAPLE_ISLAND_LEVEL7": "0x0B"
}}}
```

- [ ] **Step 3: Validate the JSON and check for opcode collisions**

```bash
for f in services/atlas-configurations/seed-data/templates/template_{gms_83,gms_84,gms_87,gms_92,gms_95,jms_185}_1.json; do
  python3 -c "
import json,sys,collections
t=json.load(open('$f'))
h=[x['opCode'].lower() for x in t['socket']['handlers']]
w=[x['opCode'].lower() for x in t['socket']['writers']]
dh=[k for k,v in collections.Counter(h).items() if v>1]
dw=[k for k,v in collections.Counter(w).items() if v>1]
assert not dh and not dw, ('$f', dh, dw)
print('$f OK')
"
done
```

Expected: six `OK` lines. A duplicate means the opcode is already taken in that template — STOP and re-verify against the registry YAML (`docs/packets/registry/<version>.yaml`) before proceeding; do not guess a replacement.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-configurations/seed-data/templates/
git commit -m "feat(task-124): teleport-rock socket wiring in all seed templates"
```

---

### Task 21: deploy manifests — the two new Kafka topics

**Files:**
- Modify: `deploy/k8s/base/env-configmap.yaml`
- Modify: `deploy/k8s/overlays/pr/kustomization.yaml`
- Modify: `deploy/k8s/overlays/main/kustomization.yaml`
- Modify: `deploy/compose/.env.example`

**Interfaces:**
- Consumes: env names from Task 9 (`COMMAND_TOPIC_TELEPORT_ROCK`, `EVENT_TOPIC_TELEPORT_ROCK_STATUS`).
- Produces: topic env vars resolvable by atlas-character and atlas-channel at runtime (design §6).

- [ ] **Step 1: Add both topics to each file, following the exact `COMMAND_TOPIC_CHARACTER` pattern in place** (base configmap line ~19; pr overlay line ~114 with the `-PLACEHOLDER_ATLAS_ENV` suffix; main overlay line ~58 with the `-main` suffix; compose env line ~28):

`deploy/k8s/base/env-configmap.yaml`:
```yaml
  COMMAND_TOPIC_TELEPORT_ROCK: "COMMAND_TOPIC_TELEPORT_ROCK"
  EVENT_TOPIC_TELEPORT_ROCK_STATUS: "EVENT_TOPIC_TELEPORT_ROCK_STATUS"
```

`deploy/k8s/overlays/pr/kustomization.yaml`:
```yaml
      - COMMAND_TOPIC_TELEPORT_ROCK=COMMAND_TOPIC_TELEPORT_ROCK-PLACEHOLDER_ATLAS_ENV
      - EVENT_TOPIC_TELEPORT_ROCK_STATUS=EVENT_TOPIC_TELEPORT_ROCK_STATUS-PLACEHOLDER_ATLAS_ENV
```

`deploy/k8s/overlays/main/kustomization.yaml`:
```yaml
      - COMMAND_TOPIC_TELEPORT_ROCK=COMMAND_TOPIC_TELEPORT_ROCK-main
      - EVENT_TOPIC_TELEPORT_ROCK_STATUS=EVENT_TOPIC_TELEPORT_ROCK_STATUS-main
```

`deploy/compose/.env.example`:
```
COMMAND_TOPIC_TELEPORT_ROCK=COMMAND_TOPIC_TELEPORT_ROCK
EVENT_TOPIC_TELEPORT_ROCK_STATUS=EVENT_TOPIC_TELEPORT_ROCK_STATUS
```

Before editing, grep each file for how the most recently added topic (e.g. a door/mount topic) is expressed and match it exactly — if the overlays use a different mechanism than shown above, follow the file, not this plan.

- [ ] **Step 2: Validate kustomize builds**

```bash
kubectl kustomize deploy/k8s/overlays/main > /dev/null && echo main-ok
kubectl kustomize deploy/k8s/overlays/pr > /dev/null && echo pr-ok
```

Expected: `main-ok`, `pr-ok`.

- [ ] **Step 3: Commit**

```bash
git add deploy/
git commit -m "feat(task-124): teleport-rock Kafka topic env vars"
```

---

### Task 22: packet verification campaign (FR-20/21)

**Files:**
- Modify: byte-fixture tests from Tasks 3, 4, 6 (add per-version `packet-audit:verify` markers)
- Create: evidence YAMLs under `docs/packets/evidence/<version>/`
- Modify: `docs/packets/audits/STATUS.md` (regenerated)

**Process:** one `packet-verifier` agent dispatch per op × version, per `docs/packets/audits/VERIFYING_A_PACKET.md`. Serialize dispatches per IDB (shared IDA instance).

Cells (op × version), with the design's pinned anchors:

| Op | v83 | v95 | v84 / v87 / jms |
|---|---|---|---|
| `USE_TELEPORT_ROCK` (`teleportrock/serverbound/Use`) | `0xA0A3BB` | `0x9E6020` | fnames absent from checked-in exports — needs IDB |
| `TROCK_ADD_MAP` (`teleportrock/serverbound/AddMap`) | `0xA261BC` | `0x9F3B90` | needs IDB |
| `MAP_TRANSFER_RESULT` (`teleportrock/clientbound/MapTransferResult`) | `0xA25268` | `0x9F9F90` | needs IDB |

- [ ] **Step 1: Verify the v83 cells** — dispatch `packet-verifier` for each of the three ops × gms_v83. The v83 IDB is `MapleStory_dump.exe`; run `list_instances` first and match by binary NAME (the loaded set rotates). Each pass produces: marker line in the fixture test, evidence YAML, STATUS.md regen, committed together.
- [ ] **Step 2: Verify the v95 cells** — same, against `GMS_v95.0_U_DEVM.exe`.
- [ ] **Step 3: Verify v84, v87, jms cells** — these IDBs are NOT currently loaded and the three fnames are absent from the checked-in ida-exports (verified at design time). Run `list_instances`; if an IDB is loaded, verify normally. **If an IDB is unavailable or an fname does not resolve: STOP and ask the user** — never substitute an fname, auto-re-export, or fake evidence. Record any stopped cell explicitly in the task summary.
- [ ] **Step 4: gms_92** — no IDB exists; its cells remain at the matrix's unverified designation (template-lineage values). This is the sole sanctioned exception (PRD FR-20). No action beyond confirming STATUS.md renders it unverified.
- [ ] **Step 5: Mode-arm coverage** — MAP_TRANSFER_RESULT is a single-writer mode packet (`dispatcher-lint` not applicable), but per the no-mode-byte-only-verification rule every emitted mode arm needs a body fixture: list-refresh with BOTH `targetList` flags (Task 6 covers regular+VIP) plus at least one error mode (Task 6 covers modes 5/8). Confirm the fixtures satisfy the packet-verifier's evidence format; extend if the verifier requires per-mode goldens.
- [ ] **Step 6: Gate checks**

```bash
packet-audit matrix --check
packet-audit operations --check
```

(Invoke via the repo's documented wrapper if `packet-audit` is not on PATH — see `docs/packets/audits/VERIFYING_A_PACKET.md` for the exact command form.) Expected: both exit 0.

- [ ] **Step 7: Commit** (the verifier passes commit per-cell; this step is only for any residual regen)

```bash
git add docs/packets/ libs/atlas-packet/
git commit -m "verify(task-124): teleport-rock packet fixtures and evidence"
```

---

### Task 23: final verification gates

- [ ] **Step 1: Full test/vet/build sweep in every changed module**

```bash
(cd libs/atlas-saga && go test -race ./... && go vet ./... && go build ./...)
(cd libs/atlas-constants && go test -race ./... && go vet ./... && go build ./...)
(cd libs/atlas-packet && go test -race ./... && go vet ./... && go build ./...)
(cd services/atlas-character/atlas.com/character && go test -race ./... && go vet ./... && go build ./...)
(cd services/atlas-channel/atlas.com/channel && go test -race ./... && go vet ./... && go build ./...)
```

Expected: all PASS/clean.

- [ ] **Step 2: Docker bakes (mandatory — go.work will not catch Dockerfile COPY gaps)**

```bash
docker buildx bake atlas-character atlas-channel atlas-saga-orchestrator atlas-login
```

atlas-saga-orchestrator and atlas-login are included because `libs/atlas-saga` / `libs/atlas-packet` changes ripple into their images. No new lib was added, so no Dockerfile `COPY` edits are expected — but the bake proves it. Expected: all targets build.

- [ ] **Step 3: Redis key guard**

```bash
tools/redis-key-guard.sh
```

Expected: clean (no new Redis usage in this task).

- [ ] **Step 4: Code review before PR**

Invoke `superpowers:requesting-code-review` (dispatches `plan-adherence-reviewer` + `backend-guidelines-reviewer`; no TS changed). Findings go to `docs/tasks/task-124-teleport-rocks/audit.md`. Address findings before opening the PR.

- [ ] **Step 5: Commit any residual fixes**

```bash
git status --short   # should be clean, or commit review fixes with a fix(task-124) message
```

---

## Deploy / Rollout (post-merge, FR-19)

Seed templates apply only at tenant creation — **live tenants do not re-seed**. After the images deploy:

1. For EACH live tenant on gms_83/84/87/92/95/jms: PATCH the tenant's socket configuration (via atlas-configurations REST, same procedure as prior opcode rollouts — task-112 operations-table backfill is the precedent) to add:
   - the two handler rows (per-version opcodes from Task 20's table, each with `LoggedInValidator`),
   - the `MapTransferResult` writer row **including the full nine-key `operations` map** (a missing key → `ResolveCode` returns 99 → client crash).
2. Restart atlas-channel pods (handlers/writers do not hot-reload; the config projection does not re-register socket handlers).
3. Ensure the two Kafka topics exist in the environment (auto-create or provisioning per the environment's Kafka policy; the env vars land via Task 21).
4. Smoke test on a v83 tenant: add map → list refresh appears; remove map → refresh; use regular rock to saved map → warp + rock consumed; failed warp (barred map) → error string, rock NOT consumed; VIP rock warp → not consumed.

Symptom of a missed patch: `unhandled message op 0x54/0x66` at info level in atlas-channel logs (known failure class — memory `bug_new_opcodes_not_in_live_tenant_config`).

## Execution notes

- Task order is the dependency order; Tasks 2–7 (libs) and 8–12 (atlas-character) can interleave, but 13+ require both.
- Where the plan says "check/verify X against file:line", that is a load-bearing instruction: the repo idiom wins over the plan snippet when they disagree (imports, generic helper signatures, test-reader construction). Do not silently invent an alternative — read the cited file.
- Task 22 requires live IDA instances; it is the only task with an external dependency. A stopped cell there is a stop-and-ask, not a silent skip.


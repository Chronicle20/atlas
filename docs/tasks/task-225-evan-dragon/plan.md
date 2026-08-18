# Evan Dragon Entity Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Give every Evan character at a dragon-bearing growth stage a dragon entity that renders and moves on every client in the field, owned by a new `atlas-dragons` service, with all 24 dragon packet matrix cells promoted to verified.

**Architecture:** Four new codecs in `libs/atlas-packet/dragon` carry the wire. A new database-free, Redis-backed `atlas-dragons` service owns dragon lifecycle keyed by `(tenant, ownerCharacterId)` and reacts to `EVENT_TOPIC_CHARACTER_STATUS`; it emits `EVENT_TOPIC_DRAGON_STATUS`. `atlas-channel` decodes the serverbound move into a `COMMAND_TOPIC_DRAGON` command, consumes the status events into map-wide broadcasts, and replays existing dragons to an entering session. Six of eleven seed templates route the four opcodes.

**Tech Stack:** Go 1.25.5, `libs/atlas-redis`, `libs/atlas-kafka` (segmentio), api2go JSON:API, `libs/atlas-constants`, `libs/atlas-socket`, miniredis for registry tests.

**Spec:** [`design.md`](./design.md) (PRD at [`prd.md`](./prd.md), implementation context at [`context.md`](./context.md))

## Global Constraints

- Dragon coordinates are **`int32`** end to end — model, stored value, event body, codec. `SPAWN_DRAGON` encodes 4-byte coords, unlike every other entity in the protocol (design §2.2).
- `SPAWN_DRAGON` carries a **discarded `uint16`** between `stance` and `jobId`. Write `0`; never omit it.
- Serverbound `MOVE_DRAGON` has **no leading identity field** — the body is the `CMovePath` blob and nothing else (design §2.5). The sending session is the identity.
- **No version gating.** All four layouts are uniform across v83/v84/v87/v92/v95/JMS185 (design §2.6). No `MajorAtLeast` call belongs in these codecs; a raw `> N` comparison is prohibited outright.
- The dragon-bearing job set is the identities `job.EvanStage1`…`job.EvanStage10`. The Evan beginner (`2001`) is excluded. Resolve through `constants.For(region, major, minor).Job.Resolve` — never a raw numeric range on wire ids.
- Every registry key, REST call, and Kafka message is tenant-scoped via `tenant.MustFromContext(ctx)`.
- All keyed Redis access goes through `libs/atlas-redis`; all goroutines through `routine.Go`.
- Models are immutable: private fields, getters, Builder. No `*_testhelpers.go`.
- Opcodes are resolved from tenant socket configuration, never hard-coded in Go.
- Six templates change: `template_gms_83_1.json`, `_84_`, `_87_`, `_92_`, `_95_`, `template_jms_185_1.json`. `template_gms_12_1.json`, `_48_`, `_61_`, `_72_`, `_79_` are untouched.
- `atlas-dragons` has **no database**: not in `tools/db-bootstrap.sh`, no `DB_NAME` patch, no `ATLAS_DB_NAMES` entry, no migration.

---

### Task 1: Dragon packet codecs

**Files:**
- Create: `libs/atlas-packet/dragon/clientbound/spawn.go`
- Create: `libs/atlas-packet/dragon/clientbound/move.go`
- Create: `libs/atlas-packet/dragon/clientbound/remove.go`
- Create: `libs/atlas-packet/dragon/serverbound/move.go`
- Test: `libs/atlas-packet/dragon/clientbound/spawn_test.go`, `move_test.go`, `remove_test.go`
- Test: `libs/atlas-packet/dragon/serverbound/move_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces: `clientbound.NewDragonSpawn(ownerCharacterId uint32, x, y int32, stance byte, jobId uint16) DragonSpawn`; `clientbound.NewDragonMove(ownerCharacterId uint32, rawMovement []byte) DragonMove`; `clientbound.NewDragonRemove(ownerCharacterId uint32) DragonRemove`; the writer-name constants `clientbound.DragonSpawnWriter = "DragonSpawn"`, `DragonMoveWriter = "DragonMove"`, `DragonRemoveWriter = "DragonRemove"`; the handler-name constant `serverbound.DragonMoveHandle = "DragonMoveHandle"`; `serverbound.Move` with `RawMovement() []byte`, `StartX() int16`, `StartY() int16`.

- [ ] **Step 1: Write the failing spawn test**

Create `libs/atlas-packet/dragon/clientbound/spawn_test.go`. Use the shared
harness at `libs/atlas-packet/test` — `test.CreateContext(region, major, minor)`
builds a tenant context and `test.Encode` / `test.RoundTrip` run the closures
with a null logger. `test.RoundTrip` additionally fails if the decoder leaves
unconsumed bytes, which is exactly the assertion that catches a forgotten
discarded field.

```go
package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// dragonSpawnBody is the SPAWN_DRAGON wire per CDragon::OnCreated (GMS v95.0
// @0x50dc90): Decode4 x, Decode4 y, Decode1 moveAction, Decode2 <discarded>,
// Decode2 jobCode. The leading owner character id is consumed upstream by
// CUserPool::OnUserCommonPacket (@0x94cdb0) before the family dispatch.
//
//	int  ownerCharacterId = 4242
//	int  x                = 100    (FOUR bytes, not two)
//	int  y                = -200   (FOUR bytes, not two)
//	byte stance           = 3
//	short <discarded>     = 0      (client reads and throws away)
//	short jobId           = 2214
var dragonSpawnBody = []byte{
	0x92, 0x10, 0x00, 0x00,
	0x64, 0x00, 0x00, 0x00,
	0x38, 0xFF, 0xFF, 0xFF,
	0x03,
	0x00, 0x00,
	0xA6, 0x08,
}

func TestDragonSpawnBytes(t *testing.T) {
	in := NewDragonSpawn(4242, 100, -200, 3, 2214)
	ctx := test.CreateContext("GMS", 95, 1)
	got := test.Encode(t, ctx, in.Encode, nil)
	if !bytes.Equal(got, dragonSpawnBody) {
		t.Fatalf("spawn bytes = % X, want % X", got, dragonSpawnBody)
	}
}

// The layout is uniform across all six applicable versions (verified in both
// client size classes: 0x330 = v83/v87/JMS185, 0x464 = v92/v95). If any column
// ever diverges, this table is where it shows up first.
func TestDragonSpawnBytesIdenticalAcrossVersions(t *testing.T) {
	versions := []struct {
		region string
		major  uint16
	}{
		{"GMS", 83}, {"GMS", 84}, {"GMS", 87},
		{"GMS", 92}, {"GMS", 95}, {"JMS", 185},
	}
	in := NewDragonSpawn(4242, 100, -200, 3, 2214)
	for _, v := range versions {
		got := test.Encode(t, test.CreateContext(v.region, v.major, 1), in.Encode, nil)
		if !bytes.Equal(got, dragonSpawnBody) {
			t.Errorf("%s v%d: bytes = % X, want % X", v.region, v.major, got, dragonSpawnBody)
		}
	}
}

// RoundTrip fails on unconsumed trailing bytes, so this also proves the decoder
// reads the discarded short.
func TestDragonSpawnRoundTrip(t *testing.T) {
	in := NewDragonSpawn(4242, 100, -200, 3, 2214)
	var out DragonSpawn
	test.RoundTrip(t, test.CreateContext("GMS", 95, 1), in.Encode, out.Decode, nil)
}

func TestDragonSpawnDecodeRecoversEveryField(t *testing.T) {
	ctx := test.CreateContext("GMS", 95, 1)
	var out DragonSpawn
	test.RoundTrip(t, ctx, NewDragonSpawn(4242, 100, -200, 3, 2214).Encode, out.Decode, nil)
	// RoundTrip decodes into out via the pointer receiver.
	if out.OwnerCharacterId() != 4242 || out.X() != 100 || out.Y() != -200 ||
		out.Stance() != 3 || out.JobId() != 2214 {
		t.Fatalf("round-trip mismatch: %+v", out)
	}
}
```

If `test.RoundTrip`'s signature does not let `out` be populated through the
method value `out.Decode` (Go binds the receiver at method-value creation, so it
does — `out` is addressable here), keep the assertion; otherwise decode manually
with `request.NewRequestReader` exactly as `test.RoundTrip` does internally.

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd libs/atlas-packet && go test ./dragon/... -run TestDragonSpawn -v`
Expected: FAIL — package `dragon/clientbound` does not exist.

- [ ] **Step 3: Implement `spawn.go`**

```go
package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const DragonSpawnWriter = "DragonSpawn"

// DragonSpawn is the server -> client SPAWN_DRAGON packet.
//
// Wire: int ownerCharacterId, int x, int y, byte stance, short <discarded>,
// short jobId.
//
// The leading ownerCharacterId is consumed by CUserPool::OnUserCommonPacket
// (GMS v95.0 @0x94cdb0), which resolves the CUser and only then routes the
// dragon family. The dragon has no wire identity of its own.
//
// TWO TRAPS, both load-bearing:
//   - x and y are FOUR bytes each (Decode4), not the 2-byte coordinates used by
//     every other entity in the protocol.
//   - the short between stance and jobId is read by the client and thrown away
//     (the Decode2 return value is never assigned). It must still be written or
//     jobId is read from the wrong offset.
//
// Layout is identical across v83/v84/v87/v92/v95/JMS185 — verified in two
// distinct client size classes (0x330: v83/v87/JMS185, 0x464: v92/v95). No
// version gate.
//
// packet-audit:fname CDragon::OnCreated
type DragonSpawn struct {
	ownerCharacterId uint32
	x                int32
	y                int32
	stance           byte
	jobId            uint16
}

func NewDragonSpawn(ownerCharacterId uint32, x int32, y int32, stance byte, jobId uint16) DragonSpawn {
	return DragonSpawn{ownerCharacterId: ownerCharacterId, x: x, y: y, stance: stance, jobId: jobId}
}

func (m DragonSpawn) OwnerCharacterId() uint32 { return m.ownerCharacterId }
func (m DragonSpawn) X() int32                 { return m.x }
func (m DragonSpawn) Y() int32                 { return m.y }
func (m DragonSpawn) Stance() byte             { return m.stance }
func (m DragonSpawn) JobId() uint16            { return m.jobId }
func (m DragonSpawn) Operation() string        { return DragonSpawnWriter }
func (m DragonSpawn) String() string {
	return fmt.Sprintf("ownerCharacterId [%d], x [%d], y [%d], stance [%d], jobId [%d]", m.ownerCharacterId, m.x, m.y, m.stance, m.jobId)
}

func (m DragonSpawn) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.ownerCharacterId)
		w.WriteInt32(m.x)
		w.WriteInt32(m.y)
		w.WriteByte(m.stance)
		w.WriteShort(0) // client decodes and discards; omitting it misaligns jobId
		w.WriteShort(m.jobId)
		return w.Bytes()
	}
}

func (m *DragonSpawn) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.ownerCharacterId = r.ReadUint32()
		m.x = r.ReadInt32()
		m.y = r.ReadInt32()
		m.stance = r.ReadByte()
		_ = r.ReadUint16() // discarded by the client (see Encode)
		m.jobId = r.ReadUint16()
	}
}
```

- [ ] **Step 4: Run the spawn test**

Run: `cd libs/atlas-packet && go test ./dragon/... -run TestDragonSpawn -v`
Expected: PASS.

- [ ] **Step 5: Write the failing clientbound-move and remove tests**

Create `libs/atlas-packet/dragon/clientbound/move_test.go`:

```go
package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// MOVE_DRAGON is int ownerCharacterId + the raw CMovePath blob. The blob already
// begins with the start position, so it must NOT be written separately.
func TestDragonMoveIsOwnerIdPlusOpaqueBlob(t *testing.T) {
	blob := []byte{0x0A, 0x00, 0x14, 0x00, 0x01, 0xFF}
	got := test.Encode(t, test.CreateContext("GMS", 95, 1), NewDragonMove(4242, blob).Encode, nil)

	want := append([]byte{0x92, 0x10, 0x00, 0x00}, blob...)
	if !bytes.Equal(got, want) {
		t.Fatalf("move bytes = % X, want % X", got, want)
	}
}

func TestDragonMoveRoundTrip(t *testing.T) {
	var out DragonMove
	test.RoundTrip(t, test.CreateContext("GMS", 95, 1),
		NewDragonMove(4242, []byte{0x0A, 0x00, 0x14, 0x00, 0x01, 0xFF}).Encode, out.Decode, nil)
	if out.OwnerCharacterId() != 4242 || len(out.RawMovement()) != 6 {
		t.Fatalf("round-trip mismatch: %+v", out)
	}
}
```

Create `libs/atlas-packet/dragon/clientbound/remove_test.go`:

```go
package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// REMOVE_DRAGON has no body: the only field is the owner character id, and the
// client has no handler arm for the opcode at all.
func TestDragonRemoveIsOwnerIdOnly(t *testing.T) {
	got := test.Encode(t, test.CreateContext("GMS", 95, 1), NewDragonRemove(4242).Encode, nil)
	if !bytes.Equal(got, []byte{0x92, 0x10, 0x00, 0x00}) {
		t.Fatalf("remove bytes = % X, want 92 10 00 00", got)
	}
}

func TestDragonRemoveRoundTrip(t *testing.T) {
	var out DragonRemove
	test.RoundTrip(t, test.CreateContext("GMS", 95, 1), NewDragonRemove(4242).Encode, out.Decode, nil)
	if out.OwnerCharacterId() != 4242 {
		t.Fatalf("round-trip mismatch: %+v", out)
	}
}
```

- [ ] **Step 6: Run them to verify they fail**

Run: `cd libs/atlas-packet && go test ./dragon/... -v`
Expected: FAIL — `NewDragonMove` / `NewDragonRemove` undefined.

- [ ] **Step 7: Implement `move.go` and `remove.go`**

`libs/atlas-packet/dragon/clientbound/move.go`:

```go
package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const DragonMoveWriter = "DragonMove"

// DragonMove is the server -> client MOVE_DRAGON packet: int ownerCharacterId
// (consumed upstream by CUserPool::OnUserCommonPacket) followed by the raw
// CMovePath blob, rebroadcast byte-faithfully.
//
// CDragon::OnMove (GMS v95.0 @0x50ad30) is a single call:
// CMovePath::OnMovePacket(&m_pvc[142], iPacket, 0). The whole body is the blob.
// The blob already begins with the start position (CMovePath::Encode writes
// Encode2 startX, Encode2 startY first), so the start position must NOT be
// written separately — doing so makes the observing client's CMovePath::Decode
// read 4 bytes off and throw.
//
// Layout is identical across all six applicable versions. No version gate.
//
// packet-audit:fname CDragon::OnMove
type DragonMove struct {
	ownerCharacterId uint32
	rawMovement      []byte
}

func NewDragonMove(ownerCharacterId uint32, rawMovement []byte) DragonMove {
	return DragonMove{ownerCharacterId: ownerCharacterId, rawMovement: rawMovement}
}

func (m DragonMove) OwnerCharacterId() uint32 { return m.ownerCharacterId }
func (m DragonMove) RawMovement() []byte      { return m.rawMovement }
func (m DragonMove) Operation() string        { return DragonMoveWriter }
func (m DragonMove) String() string {
	return fmt.Sprintf("ownerCharacterId [%d], rawMovement [%d bytes]", m.ownerCharacterId, len(m.rawMovement))
}

func (m DragonMove) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.ownerCharacterId)
		w.WriteByteArray(m.rawMovement) // CMovePath blob — begins with start x,y
		return w.Bytes()
	}
}

func (m *DragonMove) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.ownerCharacterId = r.ReadUint32()
		m.rawMovement = r.ReadBytes(r.Available())
	}
}
```

`libs/atlas-packet/dragon/clientbound/remove.go`:

```go
package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const DragonRemoveWriter = "DragonRemove"

// DragonRemove is the server -> client REMOVE_DRAGON packet. The body is EMPTY:
// the only field is the owner character id, and even that is consumed upstream
// by CUserPool::OnUserCommonPacket before the family dispatch.
//
// THE CLIENT HAS NO HANDLER ARM FOR THIS OPCODE. CUser::OnDragonPacket
// (GMS v95.0 @0x8e5c00) switches on nType: 206 -> spawn, 207 -> move, and
// nothing else. The pool routes 206..208 into it and 208 falls through to no
// code. The same shape holds in v83 (@0x93908f), v84, v87 (@0x9b3880),
// v92 (@0x8ce880) and JMS185 (@0x9f822f).
//
// An xref sweep on ZRef<CDragon>::_ReleaseRaw (v95 @0x8decb0) returns four
// callers: the ZRef destructor, ZRef::operator=, OnDragonPacket's respawn path,
// and CUser::~CUser. The ONLY client-side dragon teardown is destroying the
// CUser — i.e. the owner leaving the field.
//
// So: sending this packet is correct and harmless, but it is NOT the mechanism
// that removes the dragon. Do not "fix" the apparently-missing body.
//
// packet-audit:fname CUser::OnDragonPacket
type DragonRemove struct {
	ownerCharacterId uint32
}

func NewDragonRemove(ownerCharacterId uint32) DragonRemove {
	return DragonRemove{ownerCharacterId: ownerCharacterId}
}

func (m DragonRemove) OwnerCharacterId() uint32 { return m.ownerCharacterId }
func (m DragonRemove) Operation() string        { return DragonRemoveWriter }
func (m DragonRemove) String() string {
	return fmt.Sprintf("ownerCharacterId [%d]", m.ownerCharacterId)
}

func (m DragonRemove) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.ownerCharacterId)
		return w.Bytes()
	}
}

func (m *DragonRemove) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.ownerCharacterId = r.ReadUint32()
	}
}
```

- [ ] **Step 8: Run the clientbound tests**

Run: `cd libs/atlas-packet && go test ./dragon/... -v`
Expected: PASS (4 tests).

- [ ] **Step 9: Write the failing serverbound-move test**

Create `libs/atlas-packet/dragon/serverbound/move_test.go`:

```go
package serverbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// CVecCtrlDragon::EndUpdateActive (GMS v95.0 @0x996570, v83 @0x9b7b9c) writes
// COutPacket(op) then CMovePath::Flush(...) and NOTHING else. There is no
// leading identity field — unlike CVecCtrlSummoned::EndUpdateActive, which
// writes Encode4 summonId first.
func TestServerboundMoveHasNoLeadingIdentityField(t *testing.T) {
	ctx := test.CreateContext("GMS", 95, 1)
	l, _ := testlog.NewNullLogger()

	// startX=100 (0x64 0x00), startY=-200 (0x38 0xFF), then payload
	blob := []byte{0x64, 0x00, 0x38, 0xFF, 0x01, 0x00, 0x07}
	req := request.Request(blob)
	reader := request.NewRequestReader(&req, 0)

	var m Move
	m.Decode(l, ctx)(&reader, nil)

	if !bytes.Equal(m.RawMovement(), blob) {
		t.Fatalf("rawMovement must be the WHOLE body, got % X", m.RawMovement())
	}
	if m.StartX() != 100 || m.StartY() != -200 {
		t.Fatalf("start position = %d,%d, want 100,-200", m.StartX(), m.StartY())
	}

	got := test.Encode(t, ctx, m.Encode, nil)
	if !bytes.Equal(got, blob) {
		t.Fatalf("encode must be byte-faithful, got % X, want % X", got, blob)
	}
}

// The layout is uniform across all six applicable versions.
func TestServerboundMoveIdenticalAcrossVersions(t *testing.T) {
	blob := []byte{0x64, 0x00, 0x38, 0xFF, 0x01, 0x00, 0x07}
	l, _ := testlog.NewNullLogger()
	versions := []struct {
		region string
		major  uint16
	}{
		{"GMS", 83}, {"GMS", 84}, {"GMS", 87},
		{"GMS", 92}, {"GMS", 95}, {"JMS", 185},
	}
	for _, v := range versions {
		ctx := test.CreateContext(v.region, v.major, 1)
		req := request.Request(blob)
		reader := request.NewRequestReader(&req, 0)
		var m Move
		m.Decode(l, ctx)(&reader, nil)
		if !bytes.Equal(m.RawMovement(), blob) || m.StartX() != 100 || m.StartY() != -200 {
			t.Errorf("%s v%d: decode diverged: %+v", v.region, v.major, m)
		}
	}
}
```

- [ ] **Step 10: Run it to verify it fails**

Run: `cd libs/atlas-packet && go test ./dragon/serverbound/... -v`
Expected: FAIL — package does not exist.

- [ ] **Step 11: Implement `serverbound/move.go`**

```go
package serverbound

import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const DragonMoveHandle = "DragonMoveHandle"

// Move is the client -> server MOVE_DRAGON packet, decoded from the real client
// SEND site CVecCtrlDragon::EndUpdateActive (GMS v95.0 @0x996570, v83
// @0x9b7b9c). The send is exactly:
//
//	COutPacket(op)
//	CMovePath::Flush(...)   ; the opaque movement blob
//
// THERE IS NO LEADING IDENTITY FIELD. Every sibling move packet in this
// codebase has one — CVecCtrlSummoned::EndUpdateActive writes Encode4 summonId
// before the blob — so its absence here looks like a bug and is not. The dragon
// is 1:1 with its owning CUser, so the server resolves it entirely from the
// sending session's character id. A consequence worth naming: "naming a dragon
// the submitter does not own" is unrepresentable on the wire.
//
// The CMovePath blob is not trivially parseable without a full move-path codec,
// so the whole body is treated as opaque and rebroadcast byte-faithfully.
// startX/startY are lifted from its first 4 bytes (CMovePath::Encode leads with
// Encode2 startX, Encode2 startY) only to seed the persisted position.
//
// Layout is identical across all six applicable versions. No version gate.
//
// packet-audit:fname CVecCtrlDragon::EndUpdateActive
type Move struct {
	startX      int16
	startY      int16
	rawMovement []byte
}

func (m Move) StartX() int16       { return m.startX }
func (m Move) StartY() int16       { return m.startY }
func (m Move) RawMovement() []byte { return m.rawMovement }
func (m Move) Operation() string   { return DragonMoveHandle }
func (m Move) String() string {
	return fmt.Sprintf("startX [%d], startY [%d], rawMovement [%d bytes]", m.startX, m.startY, len(m.rawMovement))
}

func (m Move) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByteArray(m.rawMovement)
		return w.Bytes()
	}
}

func (m *Move) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.rawMovement = r.ReadBytes(r.Available())
		if len(m.rawMovement) >= 4 {
			m.startX = int16(binary.LittleEndian.Uint16(m.rawMovement[0:2]))
			m.startY = int16(binary.LittleEndian.Uint16(m.rawMovement[2:4]))
		}
	}
}
```

- [ ] **Step 12: Run the whole package**

Run: `cd libs/atlas-packet && go test -race ./dragon/... -v && go vet ./dragon/...`
Expected: PASS, vet clean.

- [ ] **Step 13: Commit**

```bash
git add libs/atlas-packet/dragon
git commit -m "feat(task-225): dragon packet codecs (spawn/move/remove clientbound, move serverbound)"
```

---

### Task 2: `atlas-dragons` module scaffold

**Files:**
- Create: `services/atlas-dragons/atlas.com/dragons/go.mod`
- Create: `services/atlas-dragons/atlas.com/dragons/main.go`
- Create: `services/atlas-dragons/atlas.com/dragons/rest/handler.go`
- Create: `services/atlas-dragons/atlas.com/dragons/kafka/consumer/consumer.go`
- Modify: `go.work`

**Interfaces:**
- Consumes: nothing.
- Produces: module `atlas-dragons`; `rest.ParseCharacterId`, `rest.ParseWorldId`, `rest.ParseChannelId`, `rest.ParseMapId`, `rest.ParseInstanceId`, `rest.RegisterHandler`, `rest.HandlerDependency`, `rest.HandlerContext`; `consumer.NewConfig(l)(name)(token)(groupId) consumer.Config`.

- [ ] **Step 1: Create the module and register it in the workspace**

```bash
mkdir -p services/atlas-dragons/atlas.com/dragons/{dragon,character,rest,kafka/consumer/{dragon,character},world}
```

Create `services/atlas-dragons/atlas.com/dragons/go.mod`:

```
module atlas-dragons

go 1.25.5

require (
	github.com/Chronicle20/atlas/libs/atlas-constants v0.0.0
	github.com/Chronicle20/atlas/libs/atlas-kafka v0.0.0
	github.com/Chronicle20/atlas/libs/atlas-model v0.0.0
	github.com/Chronicle20/atlas/libs/atlas-redis v0.0.0
	github.com/Chronicle20/atlas/libs/atlas-rest v0.0.0
	github.com/Chronicle20/atlas/libs/atlas-service v0.0.0-00010101000000-000000000000
	github.com/Chronicle20/atlas/libs/atlas-tenant v0.0.0
	github.com/alicebob/miniredis/v2 v2.38.0
	github.com/google/uuid v1.6.0
	github.com/gorilla/mux v1.8.1
	github.com/jtumidanski/api2go v1.0.4
	github.com/redis/go-redis/v9 v9.22.0
	github.com/segmentio/kafka-go v0.4.51
	github.com/sirupsen/logrus v1.9.4
	github.com/stretchr/testify v1.11.1
)
```

Do NOT hand-write the `replace` directives or the indirect block — add the
`use` line to `go.work` first, then let tooling fill the rest:

Add to `go.work` in the `use()` block, alphabetically between
`./services/atlas-drop-information/...` and `./services/atlas-effective-stats/...`
(check the surrounding lines and insert at the correct sorted position):

```
	./services/atlas-dragons/atlas.com/dragons
```

Then run `go work sync` from the repo root and `go mod tidy` inside the module
once there is source to tidy against (after Step 3).

- [ ] **Step 2: Create the REST helper package**

`services/atlas-dragons/atlas.com/dragons/rest/handler.go`:

```go
package rest

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

type HandlerDependency = server.HandlerDependency

type HandlerContext = server.HandlerContext

type GetHandler = server.GetHandler

var RegisterHandler = server.RegisterHandler

func ParseCharacterId(l logrus.FieldLogger, next func(uint32) http.HandlerFunc) http.HandlerFunc {
	return server.ParseIntId[uint32](l, "characterId", next)
}

func ParseMapId(l logrus.FieldLogger, next func(_map.Id) http.HandlerFunc) http.HandlerFunc {
	return server.ParseIntId[_map.Id](l, "mapId", next)
}

func ParseChannelId(l logrus.FieldLogger, next func(channel.Id) http.HandlerFunc) http.HandlerFunc {
	return server.ParseIntId[channel.Id](l, "channelId", next)
}

func ParseWorldId(l logrus.FieldLogger, next func(world.Id) http.HandlerFunc) http.HandlerFunc {
	return server.ParseIntId[world.Id](l, "worldId", next)
}

func ParseInstanceId(l logrus.FieldLogger, next func(uuid.UUID) http.HandlerFunc) http.HandlerFunc {
	return server.ParseUUIDId(l, "instanceId", next)
}

```

`RegisterHandler` is `server.RegisterHandler`, whose signature takes the
`jsonapi.ServerInformation` the resources pass through — that is what keeps the
`jsonapi` import used. If `go build` reports it unused, drop the import.

- [ ] **Step 3: Create the Kafka consumer-config helper**

`services/atlas-dragons/atlas.com/dragons/kafka/consumer/consumer.go` — copy
`services/atlas-summons/atlas.com/summons/kafka/consumer/consumer.go` verbatim
(it has no service-specific content):

```go
package consumer

import (
	"os"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

func NewConfig(l logrus.FieldLogger) func(name string) func(token string) func(groupId string) consumer.Config {
	return func(name string) func(token string) func(groupId string) consumer.Config {
		return func(token string) func(groupId string) consumer.Config {
			t, _ := topic.EnvProvider(l)(token)()
			return func(groupId string) consumer.Config {
				return consumer.NewConfig(LookupBrokers(), name, t, groupId)
			}
		}
	}
}

func LookupBrokers() []string {
	return []string{os.Getenv("BOOTSTRAP_SERVERS")}
}
```

- [ ] **Step 4: Create a minimal compiling `main.go`**

`services/atlas-dragons/atlas.com/dragons/main.go` — the REST/consumer wiring is
filled in by Tasks 8 and 9; this version must build and serve `/readyz`:

```go
package main

import (
	"os"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	consumergroup "github.com/Chronicle20/atlas/libs/atlas-kafka/consumergroup"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	service "github.com/Chronicle20/atlas/libs/atlas-service"
)

const serviceName = "atlas-dragons"

var consumerGroupId = consumergroup.Resolve("Dragon Registry Service")

type Server struct {
	baseUrl string
	prefix  string
}

func (s Server) GetBaseURL() string { return s.baseUrl }
func (s Server) GetPrefix() string  { return s.prefix }

func GetServer() Server {
	return Server{baseUrl: "", prefix: "/api/"}
}

func main() {
	rt := service.Bootstrap(serviceName)
	l := rt.Logger()

	rc := atlas.Connect(l)
	_ = rc

	rt.TeardownFunc(func() { _ = producer.GetManager().Close(l) })

	server.New(l).
		WithContext(rt.Context()).
		WithWaitGroup(rt.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(server.MountHandler("/debug/consumers", consumer.GetManager().DebugHandler())).
		AddRouteInitializer(server.MountReadiness("/readyz", rt.Ready)).
		Run()

	rt.Wait()
}
```

- [ ] **Step 5: Build**

Run:
```bash
go work sync
cd services/atlas-dragons/atlas.com/dragons && go mod tidy && go build ./... && go vet ./...
```
Expected: clean build. `go mod tidy` fills the indirect requirements and the
`replace` directives resolve through `go.work`.

- [ ] **Step 6: Commit**

```bash
git add go.work go.work.sum services/atlas-dragons
git commit -m "feat(task-225): atlas-dragons module scaffold"
```

---

### Task 3: Dragon model, builder, and the `HasDragon` predicate

**Files:**
- Create: `services/atlas-dragons/atlas.com/dragons/dragon/model.go`
- Create: `services/atlas-dragons/atlas.com/dragons/dragon/builder.go`
- Test: `services/atlas-dragons/atlas.com/dragons/dragon/model_test.go`

**Interfaces:**
- Consumes: `field.Model`, `job.Id`, `tenant.Model`.
- Produces: `dragon.Model` with getters `OwnerCharacterId() uint32`, `Field() field.Model`, `X() int32`, `Y() int32`, `Stance() byte`, `JobId() job.Id`; `dragon.NewBuilder(ownerCharacterId uint32) *Builder` with `SetField`, `SetX`, `SetY`, `SetStance`, `SetJobId`, `Build()`; `dragon.Clone(m Model) *Builder`; `dragon.HasDragon(t tenant.Model, wireJobId job.Id) bool`.

- [ ] **Step 1: Write the failing predicate test**

`services/atlas-dragons/atlas.com/dragons/dragon/model_test.go`:

```go
package dragon

import (
	"testing"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/constants"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func mustTenant(t *testing.T, region string, major, minor uint16) tenant.Model {
	t.Helper()
	ten, err := tenant.Create(uuid.New(), region, major, minor)
	if err != nil {
		t.Fatal(err)
	}
	return ten
}

func TestHasDragonCoversEveryEvanGrowthStage(t *testing.T) {
	ten := mustTenant(t, "GMS", 95, 1)
	stages := []job.Id{2200, 2210, 2211, 2212, 2213, 2214, 2215, 2216, 2217, 2218}
	for _, id := range stages {
		if !HasDragon(ten, id) {
			t.Errorf("job %d must be dragon-bearing", id)
		}
	}
}

func TestHasDragonExcludesEvanBeginnerAndOtherJobs(t *testing.T) {
	ten := mustTenant(t, "GMS", 95, 1)
	for _, id := range []job.Id{2001, 2000, 2100, 2112, 100, 0, 910} {
		if HasDragon(ten, id) {
			t.Errorf("job %d must NOT be dragon-bearing", id)
		}
	}
}

// v83 has no Evan entry in its job table, so Resolve fails and the predicate
// returns false with no version special-case anywhere in the lifecycle code.
func TestHasDragonIsFalseForEveryJobOnV83(t *testing.T) {
	ten := mustTenant(t, "GMS", 83, 1)
	for _, id := range []job.Id{2200, 2214, 2218} {
		if HasDragon(ten, id) {
			t.Errorf("v83 tenant must have no dragon-bearing job, got %d", id)
		}
	}
}

func TestHasDragonOnJms185(t *testing.T) {
	ten := mustTenant(t, "JMS", 185, 1)
	// JMS185 either binds the Evan stages or does not; assert the predicate is
	// consistent with the version table rather than assuming an answer.
	got := HasDragon(ten, 2214)
	want := jms185BindsEvan(t)
	if got != want {
		t.Fatalf("HasDragon(JMS185, 2214) = %v, version table says %v", got, want)
	}
}
```

For `jms185BindsEvan`, resolve directly through the same constants API in the
test so the assertion is derived, not assumed:

```go
func jms185BindsEvan(t *testing.T) bool {
	t.Helper()
	id, ok := constants.For("JMS", 185, 1).Job.Resolve(job.Id(2214))
	return ok && id == job.EvanStage6
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `cd services/atlas-dragons/atlas.com/dragons && go test ./dragon/... -v`
Expected: FAIL — `HasDragon` undefined.

- [ ] **Step 3: Implement `model.go`**

```go
package dragon

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/constants"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// Model is one Evan's dragon. It is 1:1 with its owning character: the client
// addresses all three clientbound dragon ops by owner character id
// (CUserPool::OnUserCommonPacket consumes the id before the family dispatch),
// so the owner IS the identity and there is no separate dragon id.
//
// x/y are int32 because SPAWN_DRAGON encodes 4-byte coordinates — unlike every
// other entity in the protocol. Keeping the wide type end to end stops a
// narrowing conversion from ever entering the pipeline.
type Model struct {
	ownerCharacterId uint32
	fld              field.Model
	x                int32
	y                int32
	stance           byte
	jobId            job.Id
}

func (m Model) OwnerCharacterId() uint32 { return m.ownerCharacterId }
func (m Model) Field() field.Model       { return m.fld }
func (m Model) X() int32                 { return m.x }
func (m Model) Y() int32                 { return m.y }
func (m Model) Stance() byte             { return m.stance }
func (m Model) JobId() job.Id            { return m.jobId }

// Move returns a copy at the new position/stance.
func (m Model) Move(x int32, y int32, stance byte) Model {
	return Clone(m).SetX(x).SetY(y).SetStance(stance).Build()
}

// HasDragon reports whether wireJobId resolves, on this tenant's client version,
// to an Evan growth stage (EvanStage1..EvanStage10). The Evan beginner (2001) is
// excluded: CDragon is created at the first growth stage.
//
// Expressed through the version-aware resolver rather than a numeric range on
// wire ids, which buys three things a `2200 <= id <= 2218` check would not:
//
//  1. v83 falls out for free — the v83 job table has no Evan entry, so Resolve
//     fails and no lifecycle path needs a version special-case.
//  2. tools/skill-job-id-guard.sh compliance — the comparison is over resolved
//     job.Identity values, never over a banned wire constant.
//  3. a future version that remaps 22xx cannot silently break it.
//
// The identity block 2200..2218 is exclusively Evan
// (libs/atlas-constants/job/identities_gen.go:83-92), so the closed range over
// Identity values is exact.
func HasDragon(t tenant.Model, wireJobId job.Id) bool {
	id, ok := constants.For(t.Region(), t.MajorVersion(), t.MinorVersion()).Job.Resolve(wireJobId)
	if !ok {
		return false
	}
	return id >= job.EvanStage1 && id <= job.EvanStage10
}
```

- [ ] **Step 4: Implement `builder.go`**

```go
package dragon

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
)

type Builder struct {
	ownerCharacterId uint32
	fld              field.Model
	x                int32
	y                int32
	stance           byte
	jobId            job.Id
}

// NewBuilder starts a dragon for ownerCharacterId. The owner is required at
// construction because it is the model's identity, not an optional attribute.
func NewBuilder(ownerCharacterId uint32) *Builder {
	return &Builder{ownerCharacterId: ownerCharacterId}
}

func Clone(m Model) *Builder {
	return &Builder{
		ownerCharacterId: m.ownerCharacterId,
		fld:              m.fld,
		x:                m.x,
		y:                m.y,
		stance:           m.stance,
		jobId:            m.jobId,
	}
}

func (b *Builder) SetField(f field.Model) *Builder { b.fld = f; return b }
func (b *Builder) SetX(x int32) *Builder           { b.x = x; return b }
func (b *Builder) SetY(y int32) *Builder           { b.y = y; return b }
func (b *Builder) SetStance(s byte) *Builder       { b.stance = s; return b }
func (b *Builder) SetJobId(id job.Id) *Builder     { b.jobId = id; return b }

func (b *Builder) Build() Model {
	return Model{
		ownerCharacterId: b.ownerCharacterId,
		fld:              b.fld,
		x:                b.x,
		y:                b.y,
		stance:           b.stance,
		jobId:            b.jobId,
	}
}
```

- [ ] **Step 5: Run the tests**

Run: `cd services/atlas-dragons/atlas.com/dragons && go test -race ./dragon/... -v`
Expected: PASS (4 tests).

- [ ] **Step 6: Run the job-id guard**

Run: `tools/skill-job-id-guard.sh` from the repo root.
Expected: exit 0 — the predicate compares `job.Identity` values, not banned wire constants.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-dragons/atlas.com/dragons/dragon
git commit -m "feat(task-225): dragon model, builder, and version-aware HasDragon predicate"
```

---

### Task 4: Redis-backed dragon registry

**Files:**
- Create: `services/atlas-dragons/atlas.com/dragons/dragon/registry.go`
- Test: `services/atlas-dragons/atlas.com/dragons/dragon/registry_test.go`

**Interfaces:**
- Consumes: `dragon.Model`, `dragon.NewBuilder` (Task 3).
- Produces: `dragon.InitRegistry(rc *goredis.Client)`, `dragon.GetRegistry() *Registry`, and on `*Registry`: `Put(ctx, t, m) error`, `Get(ctx, t, characterId uint32) (Model, error)`, `GetInField(ctx, t, f field.Model) ([]Model, error)`, `Exists(ctx, t, characterId uint32) (bool, error)`, `Update(ctx, t, characterId uint32, fn func(Model) Model) (Model, error)`, `Remove(ctx, t, characterId uint32) (bool, error)`.

- [ ] **Step 1: Write the failing registry test**

`services/atlas-dragons/atlas.com/dragons/dragon/registry_test.go`:

```go
package dragon

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func newTestRegistry(t *testing.T) (*Registry, tenant.Model, context.Context) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mr.Close)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	reg := newRegistry(rc)
	ten, err := tenant.Create(uuid.New(), "GMS", 95, 1)
	if err != nil {
		t.Fatal(err)
	}
	return reg, ten, tenant.WithContext(context.Background(), ten)
}

func testField() field.Model {
	return field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).SetInstance(uuid.Nil).Build()
}

func TestRegistryRoundTripPreservesWideCoordinates(t *testing.T) {
	reg, ten, ctx := newTestRegistry(t)
	m := NewBuilder(4242).SetField(testField()).SetX(70000).SetY(-70000).SetStance(3).SetJobId(2214).Build()

	if err := reg.Put(ctx, ten, m); err != nil {
		t.Fatal(err)
	}
	got, err := reg.Get(ctx, ten, 4242)
	if err != nil {
		t.Fatal(err)
	}
	if got.X() != 70000 || got.Y() != -70000 {
		t.Fatalf("coordinates must survive as int32, got %d,%d", got.X(), got.Y())
	}
	if got.JobId() != 2214 || got.Stance() != 3 || got.Field().MapId() != 100000000 {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
}

func TestRegistryIndexesByField(t *testing.T) {
	reg, ten, ctx := newTestRegistry(t)
	f := testField()
	if err := reg.Put(ctx, ten, NewBuilder(1).SetField(f).Build()); err != nil {
		t.Fatal(err)
	}
	if err := reg.Put(ctx, ten, NewBuilder(2).SetField(f).Build()); err != nil {
		t.Fatal(err)
	}
	ms, err := reg.GetInField(ctx, ten, f)
	if err != nil || len(ms) != 2 {
		t.Fatalf("field index miss: %v %+v", err, ms)
	}
}

// The owner character id IS the primary key, so a second Put for the same
// character overwrites rather than creating a second entity (FR-1.1).
func TestRegistryOneDragonPerCharacter(t *testing.T) {
	reg, ten, ctx := newTestRegistry(t)
	f := testField()
	_ = reg.Put(ctx, ten, NewBuilder(7).SetField(f).SetX(1).Build())
	_ = reg.Put(ctx, ten, NewBuilder(7).SetField(f).SetX(2).Build())

	ms, err := reg.GetInField(ctx, ten, f)
	if err != nil || len(ms) != 1 || ms[0].X() != 2 {
		t.Fatalf("expected exactly one dragon at x=2, got %v %+v", err, ms)
	}
}

func TestRegistryRemoveReportsWhetherItExisted(t *testing.T) {
	reg, ten, ctx := newTestRegistry(t)
	_ = reg.Put(ctx, ten, NewBuilder(9).SetField(testField()).Build())

	existed, err := reg.Remove(ctx, ten, 9)
	if err != nil || !existed {
		t.Fatalf("first remove must report existed=true, got %v %v", existed, err)
	}
	existed, err = reg.Remove(ctx, ten, 9)
	if err != nil || existed {
		t.Fatalf("second remove must report existed=false and no error, got %v %v", existed, err)
	}
	ms, _ := reg.GetInField(ctx, ten, testField())
	if len(ms) != 0 {
		t.Fatalf("field index must be cleaned up, got %+v", ms)
	}
}

func TestRegistryIsTenantIsolated(t *testing.T) {
	reg, tenA, ctxA := newTestRegistry(t)
	tenB, err := tenant.Create(uuid.New(), "GMS", 95, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctxB := tenant.WithContext(context.Background(), tenB)
	f := testField()

	_ = reg.Put(ctxA, tenA, NewBuilder(5).SetField(f).Build())

	ms, err := reg.GetInField(ctxB, tenB, f)
	if err != nil || len(ms) != 0 {
		t.Fatalf("tenant B must not see tenant A's dragon: %v %+v", err, ms)
	}
	if _, err := reg.Get(ctxB, tenB, 5); err == nil {
		t.Fatal("tenant B must not fetch tenant A's dragon by character id")
	}
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `cd services/atlas-dragons/atlas.com/dragons && go test ./dragon/... -run TestRegistry -v`
Expected: FAIL — `newRegistry` undefined.

- [ ] **Step 3: Implement `registry.go`**

```go
package dragon

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	atlasredis "github.com/Chronicle20/atlas/libs/atlas-redis"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// storedDragon is the JSON-serializable representation stored in Redis. The key
// carries the tenant id, but the tenant fields are carried in the value too so
// fromStored can rebuild a tenant for sweep-style iteration, mirroring
// storedSummon in atlas-summons.
//
// X/Y are int32, not the int16 atlas-summons uses: SPAWN_DRAGON encodes 4-byte
// coordinates (CDragon::OnCreated Decode4 x, Decode4 y).
type storedDragon struct {
	TenantId           string `json:"tenantId"`
	TenantRegion       string `json:"tenantRegion"`
	TenantMajorVersion uint16 `json:"tenantMajorVersion"`
	TenantMinorVersion uint16 `json:"tenantMinorVersion"`
	OwnerCharacterId   uint32 `json:"ownerCharacterId"`
	WorldId            byte   `json:"worldId"`
	ChannelId          byte   `json:"channelId"`
	MapId              uint32 `json:"mapId"`
	Instance           string `json:"instance"`
	X                  int32  `json:"x"`
	Y                  int32  `json:"y"`
	Stance             byte   `json:"stance"`
	JobId              uint16 `json:"jobId"`
}

func toStored(t tenant.Model, m Model) storedDragon {
	f := m.Field()
	return storedDragon{
		TenantId:           t.Id().String(),
		TenantRegion:       t.Region(),
		TenantMajorVersion: t.MajorVersion(),
		TenantMinorVersion: t.MinorVersion(),
		OwnerCharacterId:   m.OwnerCharacterId(),
		WorldId:            byte(f.WorldId()),
		ChannelId:          byte(f.ChannelId()),
		MapId:              uint32(f.MapId()),
		Instance:           f.Instance().String(),
		X:                  m.X(),
		Y:                  m.Y(),
		Stance:             m.Stance(),
		JobId:              uint16(m.JobId()),
	}
}

func fromStored(s storedDragon) (tenant.Model, Model, error) {
	tenantId, err := uuid.Parse(s.TenantId)
	if err != nil {
		return tenant.Model{}, Model{}, err
	}
	t, err := tenant.Create(tenantId, s.TenantRegion, s.TenantMajorVersion, s.TenantMinorVersion)
	if err != nil {
		return tenant.Model{}, Model{}, err
	}
	inst, perr := uuid.Parse(s.Instance)
	if perr != nil {
		inst = uuid.Nil
	}
	f := field.NewBuilder(world.Id(s.WorldId), channel.Id(s.ChannelId), _map.Id(s.MapId)).
		SetInstance(inst).Build()
	m := NewBuilder(s.OwnerCharacterId).
		SetField(f).
		SetX(s.X).
		SetY(s.Y).
		SetStance(s.Stance).
		SetJobId(job.Id(s.JobId)).
		Build()
	return t, m, nil
}

// Registry is the authority for "which dragons exist and where". There is no id
// allocator and no owner index: the owner character id is the primary key, which
// makes "at most one dragon per character" a property of the key space rather
// than an invariant to enforce.
type Registry struct {
	reg      *atlasredis.Registry[string, storedDragon]
	fieldIdx *atlasredis.KeyedSet[string]
}

var (
	registry *Registry
	once     sync.Once
)

func newRegistry(rc *goredis.Client) *Registry {
	return &Registry{
		reg:      atlasredis.NewRegistry[string, storedDragon](rc, "dragon", func(s string) string { return s }),
		fieldIdx: atlasredis.NewKeyedSet[string](rc, "dragon-map", func(s string) string { return s }),
	}
}

func InitRegistry(rc *goredis.Client) { once.Do(func() { registry = newRegistry(rc) }) }
func GetRegistry() *Registry          { return registry }

func storeSuffix(t tenant.Model, characterId uint32) string {
	return fmt.Sprintf("%s:%d", t.Id().String(), characterId)
}

func fieldSuffix(t tenant.Model, f field.Model) string {
	return fmt.Sprintf("%s:%d:%d:%d:%s", t.Id().String(), f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String())
}

func (r *Registry) Put(ctx context.Context, t tenant.Model, m Model) error {
	// Remove any stale field-index membership first: a dragon that changed field
	// without a Remove would otherwise appear in both fields' indexes.
	if prev, err := r.Get(ctx, t, m.OwnerCharacterId()); err == nil {
		if prev.Field() != m.Field() {
			_ = r.fieldIdx.Remove(ctx, fieldSuffix(t, prev.Field()), member(m.OwnerCharacterId()))
		}
	}
	if err := r.reg.Put(ctx, storeSuffix(t, m.OwnerCharacterId()), toStored(t, m)); err != nil {
		return err
	}
	return r.fieldIdx.Add(ctx, fieldSuffix(t, m.Field()), member(m.OwnerCharacterId()))
}

func member(characterId uint32) string { return fmt.Sprintf("%d", characterId) }

func (r *Registry) Get(ctx context.Context, t tenant.Model, characterId uint32) (Model, error) {
	s, err := r.reg.Get(ctx, storeSuffix(t, characterId))
	if err != nil {
		return Model{}, err
	}
	_, m, derr := fromStored(s)
	if derr != nil {
		return Model{}, derr
	}
	return m, nil
}

func (r *Registry) Exists(ctx context.Context, t tenant.Model, characterId uint32) (bool, error) {
	return r.reg.Exists(ctx, storeSuffix(t, characterId))
}

func (r *Registry) GetInField(ctx context.Context, t tenant.Model, f field.Model) ([]Model, error) {
	members, err := r.fieldIdx.Members(ctx, fieldSuffix(t, f))
	if err != nil {
		return nil, err
	}
	out := make([]Model, 0, len(members))
	for _, mem := range members {
		var characterId uint32
		if _, err := fmt.Sscanf(mem, "%d", &characterId); err != nil {
			continue
		}
		m, err := r.Get(ctx, t, characterId)
		if err != nil {
			continue // stale index entry
		}
		out = append(out, m)
	}
	return out, nil
}

func (r *Registry) Update(ctx context.Context, t tenant.Model, characterId uint32, fn func(Model) Model) (Model, error) {
	s, err := r.reg.Update(ctx, storeSuffix(t, characterId), func(cur storedDragon) storedDragon {
		_, m, derr := fromStored(cur)
		if derr != nil {
			return cur
		}
		return toStored(t, fn(m))
	})
	if err != nil {
		return Model{}, err
	}
	_, m, derr := fromStored(s)
	if derr != nil {
		return Model{}, derr
	}
	return m, nil
}

// Remove deletes the dragon and reports whether one existed. The bool is what
// makes destroy idempotent at the processor level: no dragon means no DESTROYED
// event, not an error.
func (r *Registry) Remove(ctx context.Context, t tenant.Model, characterId uint32) (bool, error) {
	m, err := r.Get(ctx, t, characterId)
	if err == nil {
		_ = r.fieldIdx.Remove(ctx, fieldSuffix(t, m.Field()), member(characterId))
	}
	return r.reg.RemoveExisting(ctx, storeSuffix(t, characterId))
}
```

If `field.Model` is not comparable with `!=` (it embeds a uuid and scalar ids, so
it should be), compare the four components explicitly instead:
`prev.Field().WorldId() != m.Field().WorldId() || ...`.

- [ ] **Step 4: Run the registry tests**

Run: `cd services/atlas-dragons/atlas.com/dragons && go test -race ./dragon/... -v`
Expected: PASS (all model + registry tests).

- [ ] **Step 5: Run the Redis guard**

Run: `tools/redis-key-guard.sh` from the repo root.
Expected: exit 0 — every keyed command goes through `libs/atlas-redis`.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-dragons/atlas.com/dragons/dragon
git commit -m "feat(task-225): Redis-backed dragon registry keyed by owner character id"
```

---

### Task 5: Character REST client

**Files:**
- Create: `services/atlas-dragons/atlas.com/dragons/character/model.go`
- Create: `services/atlas-dragons/atlas.com/dragons/character/rest.go`
- Create: `services/atlas-dragons/atlas.com/dragons/character/requests.go`
- Create: `services/atlas-dragons/atlas.com/dragons/character/processor.go`

**Interfaces:**
- Consumes: nothing from earlier tasks.
- Produces: `character.Model` with `Id() uint32`, `JobId() job.Id`, `X() int16`, `Y() int16`, `Stance() byte`; `character.Processor` interface with `GetById(characterId uint32) (Model, error)`; `character.NewProcessor(l, ctx) Processor`.

- [ ] **Step 1: Write `model.go` and `rest.go`**

`character/model.go`:

```go
package character

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
)

// Model is the slice of a character atlas-dragons needs: the job (to decide
// whether a dragon exists at all) and the position (to seed the dragon's spawn
// coordinates). Nothing else is fetched.
type Model struct {
	id     uint32
	jobId  job.Id
	x      int16
	y      int16
	stance byte
}

func (m Model) Id() uint32     { return m.id }
func (m Model) JobId() job.Id  { return m.jobId }
func (m Model) X() int16       { return m.x }
func (m Model) Y() int16       { return m.y }
func (m Model) Stance() byte   { return m.stance }
```

`character/rest.go`:

```go
package character

import (
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
)

type RestModel struct {
	Id     uint32 `json:"-"`
	JobId  job.Id `json:"jobId"`
	X      int16  `json:"x"`
	Y      int16  `json:"y"`
	Stance byte   `json:"stance"`
}

func (r RestModel) GetName() string { return "characters" }

func (r RestModel) GetID() string { return strconv.Itoa(int(r.Id)) }

func (r *RestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

func Extract(m RestModel) (Model, error) {
	return Model{id: m.Id, jobId: m.JobId, x: m.X, y: m.Y, stance: m.Stance}, nil
}
```

- [ ] **Step 2: Write `requests.go` and `processor.go`**

`character/requests.go`:

```go
package character

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource = "characters"
	// Sparse fieldset: atlas-dragons needs only the job and position. Fetching
	// the full character (inventory, skills, buffs) on every field entry would
	// be wasteful on a path that runs for every logging-in character.
	ById = Resource + "/%d?fields[characters]=jobId,x,y,stance"
)

func getBaseRequest() string {
	return requests.RootUrl("CHARACTERS")
}

func requestById(id uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+ById, id))
}
```

`character/processor.go`:

```go
package character

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type Processor interface {
	GetById(characterId uint32) (Model, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx, t: tenant.MustFromContext(ctx)}
}

var _ Processor = (*ProcessorImpl)(nil)

// GetById returns requests.ErrNotFound when the character no longer exists.
// Callers must treat that as "gone", not as a fetch failure.
func (p *ProcessorImpl) GetById(characterId uint32) (Model, error) {
	return model.Map(model.Decorate([]model.Decorator[Model]{}))(
		requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(characterId), Extract))()
}
```

If the `model.Map(model.Decorate(...))` wrapper is redundant with no decorators,
simplify to `requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(characterId), Extract)()`
— match whichever form compiles cleanly against the current `atlas-rest` API, as
`services/atlas-pets/atlas.com/pets/character/processor.go:45-49` does.

- [ ] **Step 3: Build**

Run: `cd services/atlas-dragons/atlas.com/dragons && go build ./... && go vet ./...`
Expected: clean.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-dragons/atlas.com/dragons/character
git commit -m "feat(task-225): atlas-dragons character REST client"
```

---

### Task 6: Dragon Kafka contracts and event producer

**Files:**
- Create: `services/atlas-dragons/atlas.com/dragons/dragon/kafka.go`
- Create: `services/atlas-dragons/atlas.com/dragons/dragon/producer.go`
- Create: `services/atlas-dragons/atlas.com/dragons/kafka/consumer/dragon/kafka.go`
- Create: `services/atlas-dragons/atlas.com/dragons/kafka/consumer/character/kafka.go`
- Test: `services/atlas-dragons/atlas.com/dragons/dragon/producer_test.go`

**Interfaces:**
- Consumes: `dragon.Model` (Task 3).
- Produces: `dragon.EnvEventTopicDragonStatus = "EVENT_TOPIC_DRAGON_STATUS"`; `dragon.EventDragonStatusCreated/Moved/Destroyed`; `dragon.StatusEvent[E]` envelope with fields `WorldId, ChannelId, MapId, Instance, OwnerCharacterId, Type, Body`; bodies `StatusEventCreatedBody{X int32, Y int32, Stance byte, JobId uint16}`, `StatusEventMovedBody{RawMovement []byte}`, `StatusEventDestroyedBody{}`; providers `createdEventProvider(m Model)`, `movedEventProvider(m Model, rawMovement []byte)`, `destroyedEventProvider(m Model)`. In `kafka/consumer/dragon`: `EnvCommandTopic = "COMMAND_TOPIC_DRAGON"`, `CommandTypeCreate/Destroy/Move`, `Command[E]` envelope, `CreateCommandBody{CharacterId uint32}`, `DestroyCommandBody{CharacterId uint32}`, `MoveCommandBody{CharacterId uint32, StartX int16, StartY int16, Stance byte, RawMovement []byte}`. In `kafka/consumer/character`: `EnvEventTopicCharacterStatus`, the five status-type constants, `StatusEvent[E]` and the five bodies.

- [ ] **Step 1: Write `dragon/kafka.go`**

```go
package dragon

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

const EnvEventTopicDragonStatus = "EVENT_TOPIC_DRAGON_STATUS"

const (
	EventDragonStatusCreated   = "CREATED"
	EventDragonStatusMoved     = "MOVED"
	EventDragonStatusDestroyed = "DESTROYED"
)

// StatusEvent is the dragon-status event envelope. It is exported because
// atlas-channel consumes it across a MODULE boundary — the channel-side mirror
// at services/atlas-channel/.../kafka/message/dragon/kafka.go must keep every
// json tag byte-for-byte identical. A tag renamed in one and not the other
// fails no build and decodes into a zero-valued body at runtime.
//
// The dragon has no id of its own; OwnerCharacterId is the identity (the client
// addresses all three clientbound ops by owner character id).
type StatusEvent[E any] struct {
	WorldId          world.Id   `json:"worldId"`
	ChannelId        channel.Id `json:"channelId"`
	MapId            _map.Id    `json:"mapId"`
	Instance         uuid.UUID  `json:"instance"`
	OwnerCharacterId uint32     `json:"ownerCharacterId"`
	Type             string     `json:"type"`
	Body             E          `json:"body"`
}

// StatusEventCreatedBody carries the dragon's spawn frame. X/Y are int32 because
// SPAWN_DRAGON encodes 4-byte coordinates.
type StatusEventCreatedBody struct {
	X      int32  `json:"x"`
	Y      int32  `json:"y"`
	Stance byte   `json:"stance"`
	JobId  uint16 `json:"jobId"`
}

// StatusEventMovedBody carries the raw CMovePath blob and no coordinates: the
// blob is what other clients render. The stored position exists only so a
// late-entering viewer gets a sane first frame.
type StatusEventMovedBody struct {
	RawMovement []byte `json:"rawMovement"`
}

// StatusEventDestroyedBody is empty: REMOVE_DRAGON carries only the owner id,
// which lives on the envelope.
type StatusEventDestroyedBody struct{}
```

- [ ] **Step 2: Write `dragon/producer.go`**

```go
package dragon

import (
	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func createdEventProvider(m Model) model.Provider[[]kafka.Message] {
	f := m.Field()
	key := producer.CreateKey(int(f.MapId()))
	value := StatusEvent[StatusEventCreatedBody]{
		WorldId: f.WorldId(), ChannelId: f.ChannelId(),
		MapId: f.MapId(), Instance: f.Instance(),
		OwnerCharacterId: m.OwnerCharacterId(),
		Type:             EventDragonStatusCreated,
		Body: StatusEventCreatedBody{
			X: m.X(), Y: m.Y(), Stance: m.Stance(), JobId: uint16(m.JobId()),
		},
	}
	return producer.SingleMessageProvider(key, &value)
}

func movedEventProvider(m Model, rawMovement []byte) model.Provider[[]kafka.Message] {
	f := m.Field()
	key := producer.CreateKey(int(f.MapId()))
	value := StatusEvent[StatusEventMovedBody]{
		WorldId: f.WorldId(), ChannelId: f.ChannelId(),
		MapId: f.MapId(), Instance: f.Instance(),
		OwnerCharacterId: m.OwnerCharacterId(),
		Type:             EventDragonStatusMoved,
		Body:             StatusEventMovedBody{RawMovement: rawMovement},
	}
	return producer.SingleMessageProvider(key, &value)
}

func destroyedEventProvider(m Model) model.Provider[[]kafka.Message] {
	f := m.Field()
	key := producer.CreateKey(int(f.MapId()))
	value := StatusEvent[StatusEventDestroyedBody]{
		WorldId: f.WorldId(), ChannelId: f.ChannelId(),
		MapId: f.MapId(), Instance: f.Instance(),
		OwnerCharacterId: m.OwnerCharacterId(),
		Type:             EventDragonStatusDestroyed,
		Body:             StatusEventDestroyedBody{},
	}
	return producer.SingleMessageProvider(key, &value)
}
```

- [ ] **Step 3: Write the command contract**

`kafka/consumer/dragon/kafka.go`:

```go
package dragon

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// EnvCommandTopic is the COMMAND_TOPIC_DRAGON env var (channel -> dragons).
// The channel-side mirror at
// services/atlas-channel/.../kafka/message/dragon/kafka.go must keep every json
// tag byte-for-byte identical to these definitions.
const EnvCommandTopic = "COMMAND_TOPIC_DRAGON"

const (
	CommandTypeCreate  = "CREATE"
	CommandTypeDestroy = "DESTROY"
	CommandTypeMove    = "MOVE"
)

type Command[E any] struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
	Type      string     `json:"type"`
	Body      E          `json:"body"`
}

type CreateCommandBody struct {
	CharacterId uint32 `json:"characterId"`
}

type DestroyCommandBody struct {
	CharacterId uint32 `json:"characterId"`
}

// MoveCommandBody carries the raw CMovePath blob plus the start position lifted
// from its first four bytes. The serverbound packet has no identity field, so
// CharacterId is the SENDING SESSION's character id, filled in channel-side.
type MoveCommandBody struct {
	CharacterId uint32 `json:"characterId"`
	StartX      int16  `json:"startX"`
	StartY      int16  `json:"startY"`
	Stance      byte   `json:"stance"`
	RawMovement []byte `json:"rawMovement"`
}
```

- [ ] **Step 4: Write the character-status contract**

`kafka/consumer/character/kafka.go`:

```go
package character

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

const (
	EnvEventTopicCharacterStatus  = "EVENT_TOPIC_CHARACTER_STATUS"
	StatusEventTypeLogin          = "LOGIN"
	StatusEventTypeLogout         = "LOGOUT"
	StatusEventTypeMapChanged     = "MAP_CHANGED"
	StatusEventTypeChannelChanged = "CHANNEL_CHANGED"
	StatusEventTypeJobChanged     = "JOB_CHANGED"
)

// StatusEvent mirrors the atlas-character EVENT_TOPIC_CHARACTER_STATUS envelope
// (services/atlas-character/.../kafka/message/character/kafka.go). Bodies are
// decoded faithfully to avoid Kafka parse errors even where a field is unused.
type StatusEvent[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	WorldId       world.Id  `json:"worldId"`
	CharacterId   uint32    `json:"characterId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type LoginBody struct {
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
}

type LogoutBody struct {
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
}

type MapChangedBody struct {
	ChannelId      channel.Id `json:"channelId"`
	OldMapId       _map.Id    `json:"oldMapId"`
	OldInstance    uuid.UUID  `json:"oldInstance"`
	TargetMapId    _map.Id    `json:"targetMapId"`
	TargetInstance uuid.UUID  `json:"targetInstance"`
	TargetPortalId uint32     `json:"targetPortalId"`
}

type ChannelChangedBody struct {
	ChannelId    channel.Id `json:"channelId"`
	OldChannelId channel.Id `json:"oldChannelId"`
	MapId        _map.Id    `json:"mapId"`
	Instance     uuid.UUID  `json:"instance"`
}

// JobChangedBody is the only status body carrying the job id, and it carries no
// map id — so a job change into the dragon-bearing range must resolve the
// character's field from the character service, not from the event.
type JobChangedBody struct {
	ChannelId channel.Id `json:"channelId"`
	JobId     job.Id     `json:"jobId"`
}
```

- [ ] **Step 5: Write the producer test**

`services/atlas-dragons/atlas.com/dragons/dragon/producer_test.go`:

```go
package dragon

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

func TestCreatedEventCarriesWideCoordinatesAndOwner(t *testing.T) {
	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).SetInstance(uuid.Nil).Build()
	m := NewBuilder(4242).SetField(f).SetX(70000).SetY(-70000).SetStance(3).SetJobId(2214).Build()

	msgs, err := createdEventProvider(m)()
	if err != nil || len(msgs) != 1 {
		t.Fatalf("provider failed: %v %d", err, len(msgs))
	}

	var e StatusEvent[StatusEventCreatedBody]
	if err := json.Unmarshal(msgs[0].Value, &e); err != nil {
		t.Fatal(err)
	}
	if e.Type != EventDragonStatusCreated || e.OwnerCharacterId != 4242 {
		t.Fatalf("envelope mismatch: %+v", e)
	}
	if e.Body.X != 70000 || e.Body.Y != -70000 || e.Body.JobId != 2214 {
		t.Fatalf("body mismatch: %+v", e.Body)
	}
}

func TestMovedEventCarriesRawBlobOnly(t *testing.T) {
	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).SetInstance(uuid.Nil).Build()
	m := NewBuilder(4242).SetField(f).Build()

	msgs, err := movedEventProvider(m, []byte{1, 2, 3})()
	if err != nil || len(msgs) != 1 {
		t.Fatalf("provider failed: %v %d", err, len(msgs))
	}
	var e StatusEvent[StatusEventMovedBody]
	if err := json.Unmarshal(msgs[0].Value, &e); err != nil {
		t.Fatal(err)
	}
	if e.Type != EventDragonStatusMoved || len(e.Body.RawMovement) != 3 {
		t.Fatalf("moved event mismatch: %+v", e)
	}
}
```

- [ ] **Step 6: Run**

Run: `cd services/atlas-dragons/atlas.com/dragons && go test -race ./... -v`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-dragons/atlas.com/dragons
git commit -m "feat(task-225): dragon Kafka contracts and status-event producer"
```

---

### Task 7: Dragon processor — create, destroy, move

**Files:**
- Create: `services/atlas-dragons/atlas.com/dragons/dragon/processor.go`
- Test: `services/atlas-dragons/atlas.com/dragons/dragon/processor_test.go`

**Interfaces:**
- Consumes: `dragon.Registry` (Task 4), `dragon.HasDragon` (Task 3), `character.Processor` (Task 5), the event providers (Task 6).
- Produces: `dragon.Processor` interface — `GetByCharacterId(characterId uint32) (Model, error)`, `GetInField(f field.Model) ([]Model, error)`, `Create(f field.Model, characterId uint32) error`, `Destroy(characterId uint32) error`, `Move(characterId uint32, startX, startY int16, stance byte, rawMovement []byte) error`; `dragon.NewProcessor(l, ctx) Processor`.

- [ ] **Step 1: Write the failing processor test**

`services/atlas-dragons/atlas.com/dragons/dragon/processor_test.go`:

```go
package dragon

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"atlas-dragons/character"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// stubCharacters returns a fixed character, or ErrNotFound when notFound is set.
type stubCharacters struct {
	m        character.Model
	notFound bool
	calls    int
}

func (s *stubCharacters) GetById(characterId uint32) (character.Model, error) {
	s.calls++
	if s.notFound {
		return character.Model{}, requests.ErrNotFound
	}
	return s.m, nil
}

type capturedEvent struct {
	topic string
}

func newTestProcessor(t *testing.T, cs *stubCharacters) (*ProcessorImpl, tenant.Model, context.Context, *[]capturedEvent) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mr.Close)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	registry = newRegistry(rc) // package-level singleton used by the processor

	ten, err := tenant.Create(uuid.New(), "GMS", 95, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)

	var emitted []capturedEvent
	p := &ProcessorImpl{
		l: logrus.New(), ctx: ctx, t: ten,
		characters: cs,
		emit: func(topic string, provider model.Provider[[]kafka.Message]) error {
			if _, err := provider(); err != nil {
				return err
			}
			emitted = append(emitted, capturedEvent{topic: topic})
			return nil
		},
	}
	return p, ten, ctx, &emitted
}

// buildCharacter constructs the stub's return value through the character
// Builder added in Step 2 — no test-only constructor, no *_testhelpers.go.
func buildCharacter(t *testing.T, id uint32, jobId job.Id, x, y int16) character.Model {
	t.Helper()
	return character.NewBuilder(id).SetJobId(jobId).SetX(x).SetY(y).Build()
}

func TestCreateIsANoOpForANonDragonJob(t *testing.T) {
	cs := &stubCharacters{m: buildCharacter(t, 1, 2001, 10, 20)}
	p, ten, ctx, emitted := newTestProcessor(t, cs)

	if err := p.Create(testField(), 1); err != nil {
		t.Fatal(err)
	}
	if len(*emitted) != 0 {
		t.Fatalf("Evan beginner must not spawn a dragon, emitted %v", *emitted)
	}
	if ok, _ := GetRegistry().Exists(ctx, ten, 1); ok {
		t.Fatal("no dragon must be stored for a non-dragon job")
	}
}

func TestCreateStoresAndEmitsOnceThenIsIdempotent(t *testing.T) {
	cs := &stubCharacters{m: buildCharacter(t, 42, 2214, 100, -200)}
	p, ten, ctx, emitted := newTestProcessor(t, cs)

	if err := p.Create(testField(), 42); err != nil {
		t.Fatal(err)
	}
	if err := p.Create(testField(), 42); err != nil {
		t.Fatal(err)
	}

	if len(*emitted) != 1 {
		t.Fatalf("a redelivered CREATE must emit exactly one CREATED, got %d", len(*emitted))
	}
	m, err := GetRegistry().Get(ctx, ten, 42)
	if err != nil || m.X() != 100 || m.Y() != -200 || m.JobId() != 2214 {
		t.Fatalf("stored dragon mismatch: %v %+v", err, m)
	}
}

func TestCreateIsANoOpWhenTheCharacterIsGone(t *testing.T) {
	cs := &stubCharacters{notFound: true}
	p, _, _, emitted := newTestProcessor(t, cs)

	if err := p.Create(testField(), 99); err != nil {
		t.Fatalf("a 404 means the character is gone, not a fetch failure: %v", err)
	}
	if len(*emitted) != 0 {
		t.Fatalf("no event for a missing character, got %v", *emitted)
	}
}

func TestDestroyEmitsOnceAndIsANoOpForAnAbsentDragon(t *testing.T) {
	cs := &stubCharacters{m: buildCharacter(t, 42, 2214, 0, 0)}
	p, _, _, emitted := newTestProcessor(t, cs)
	_ = p.Create(testField(), 42)

	if err := p.Destroy(42); err != nil {
		t.Fatal(err)
	}
	if err := p.Destroy(42); err != nil {
		t.Fatalf("destroying an absent dragon must be a no-op, got %v", err)
	}
	// 1 CREATED + 1 DESTROYED
	if len(*emitted) != 2 {
		t.Fatalf("expected exactly one DESTROYED, total emitted %d", len(*emitted))
	}
}

func TestMoveWithoutADragonIsRejectedAndCreatesNothing(t *testing.T) {
	cs := &stubCharacters{m: buildCharacter(t, 42, 2214, 0, 0)}
	p, ten, ctx, emitted := newTestProcessor(t, cs)

	if err := p.Move(42, 5, 6, 1, []byte{1, 2, 3, 4}); err != nil {
		t.Fatalf("a move for a dragonless character must be a logged no-op, got %v", err)
	}
	if len(*emitted) != 0 {
		t.Fatalf("no MOVED event without a dragon, got %v", *emitted)
	}
	if ok, _ := GetRegistry().Exists(ctx, ten, 42); ok {
		t.Fatal("Move must not create a dragon as a side effect")
	}
}

func TestMoveUpdatesPositionAndEmits(t *testing.T) {
	cs := &stubCharacters{m: buildCharacter(t, 42, 2214, 0, 0)}
	p, ten, ctx, emitted := newTestProcessor(t, cs)
	_ = p.Create(testField(), 42)

	if err := p.Move(42, 111, -222, 4, []byte{9, 9}); err != nil {
		t.Fatal(err)
	}
	m, err := GetRegistry().Get(ctx, ten, 42)
	if err != nil || m.X() != 111 || m.Y() != -222 || m.Stance() != 4 {
		t.Fatalf("position not updated: %v %+v", err, m)
	}
	if len(*emitted) != 2 {
		t.Fatalf("expected CREATED + MOVED, got %d", len(*emitted))
	}
}
```

`buildCharacter` above depends on a Builder that `character.Model` does not have
yet — Step 2 adds it. The test never reaches into `character`'s private fields.

- [ ] **Step 2: Add the character Builder**

Append to `services/atlas-dragons/atlas.com/dragons/character/model.go`:

```go
type Builder struct {
	id     uint32
	jobId  job.Id
	x      int16
	y      int16
	stance byte
}

func NewBuilder(id uint32) *Builder { return &Builder{id: id} }

func (b *Builder) SetJobId(id job.Id) *Builder { b.jobId = id; return b }
func (b *Builder) SetX(x int16) *Builder       { b.x = x; return b }
func (b *Builder) SetY(y int16) *Builder       { b.y = y; return b }
func (b *Builder) SetStance(s byte) *Builder   { b.stance = s; return b }

func (b *Builder) Build() Model {
	return Model{id: b.id, jobId: b.jobId, x: b.x, y: b.y, stance: b.stance}
}
```

Then rewrite `character.Extract` to go through the Builder:

```go
func Extract(m RestModel) (Model, error) {
	return NewBuilder(m.Id).SetJobId(m.JobId).SetX(m.X).SetY(m.Y).SetStance(m.Stance).Build(), nil
}
```

- [ ] **Step 3: Run the tests to verify they fail**

Run: `cd services/atlas-dragons/atlas.com/dragons && go test ./dragon/... -run 'TestCreate|TestDestroy|TestMove' -v`
Expected: FAIL — `ProcessorImpl` undefined.

- [ ] **Step 4: Implement `processor.go`**

```go
package dragon

import (
	"atlas-dragons/character"
	"context"
	"errors"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type Processor interface {
	GetByCharacterId(characterId uint32) (Model, error)
	GetInField(f field.Model) ([]Model, error)
	Create(f field.Model, characterId uint32) error
	Destroy(characterId uint32) error
	Move(characterId uint32, startX int16, startY int16, stance byte, rawMovement []byte) error
}

// emitter publishes a kafka message provider to a topic. ProcessorImpl uses this
// indirection so tests can observe event emissions without a live Kafka, exactly
// as atlas-summons' summon.ProcessorImpl does.
type emitter func(topic string, provider model.Provider[[]kafka.Message]) error

// characterSource resolves the owner's job and position. The default is the
// character REST processor; tests substitute a stub.
type characterSource interface {
	GetById(characterId uint32) (character.Model, error)
}

type ProcessorImpl struct {
	l          logrus.FieldLogger
	ctx        context.Context
	t          tenant.Model
	emit       emitter
	characters characterSource
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l: l, ctx: ctx, t: tenant.MustFromContext(ctx),
		emit: func(topic string, provider model.Provider[[]kafka.Message]) error {
			return producer.ProviderImpl(l)(ctx)(topic)(provider)
		},
		characters: character.NewProcessor(l, ctx),
	}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) GetByCharacterId(characterId uint32) (Model, error) {
	return GetRegistry().Get(p.ctx, p.t, characterId)
}

func (p *ProcessorImpl) GetInField(f field.Model) ([]Model, error) {
	return GetRegistry().GetInField(p.ctx, p.t, f)
}

// Create spawns the dragon for characterId in field f, if that character's job
// is dragon-bearing. This is the ONE place the job gate is enforced, so no
// caller has to remember it.
//
// CREATED is emitted only on the absent->present transition. Kafka is
// at-least-once: a redelivered CREATE would otherwise produce a second map-wide
// SPAWN_DRAGON. The client's own release-then-recreate in
// CUser::OnDragonPacket's spawn arm is a second line of defence, not the first.
func (p *ProcessorImpl) Create(f field.Model, characterId uint32) error {
	c, err := p.characters.GetById(characterId)
	if err != nil {
		if errors.Is(err, requests.ErrNotFound) {
			// The character is gone. Normal for a race against logout; not a
			// fetch failure and not retryable.
			p.l.Warnf("Character [%d] not found while creating dragon; skipping.", characterId)
			return nil
		}
		return err
	}

	if !HasDragon(p.t, c.JobId()) {
		return nil
	}

	existed, err := GetRegistry().Exists(p.ctx, p.t, characterId)
	if err != nil {
		return err
	}

	m := NewBuilder(characterId).
		SetField(f).
		SetX(int32(c.X())).
		SetY(int32(c.Y())).
		SetStance(c.Stance()).
		SetJobId(c.JobId()).
		Build()
	if err := GetRegistry().Put(p.ctx, p.t, m); err != nil {
		return err
	}

	if existed {
		p.l.Debugf("Dragon for character [%d] already present in map [%d]; state refreshed, no CREATED emitted.", characterId, f.MapId())
		return nil
	}
	p.l.Debugf("Created dragon for character [%d] in map [%d] at [%d,%d].", characterId, f.MapId(), m.X(), m.Y())
	return p.emit(EnvEventTopicDragonStatus, createdEventProvider(m))
}

// Destroy removes the dragon and emits DESTROYED, only if one existed.
// Destroying an absent dragon is a silent no-op returning nil (FR-1.6).
//
// Note what DESTROYED does and does not accomplish: the client has no handler
// arm for REMOVE_DRAGON, so the packet is discarded. The dragon disappears from
// other players' screens because the owner's CUser is destroyed when they leave
// the field. See the doc comment on clientbound.DragonRemove.
func (p *ProcessorImpl) Destroy(characterId uint32) error {
	m, err := GetRegistry().Get(p.ctx, p.t, characterId)
	if err != nil {
		return nil // no dragon; nothing to do
	}
	existed, err := GetRegistry().Remove(p.ctx, p.t, characterId)
	if err != nil {
		return err
	}
	if !existed {
		return nil
	}
	p.l.Debugf("Destroyed dragon for character [%d] in map [%d].", characterId, m.Field().MapId())
	return p.emit(EnvEventTopicDragonStatus, destroyedEventProvider(m))
}

// Move updates the dragon's position/stance and relays the raw movement blob.
// It never creates a dragon as a side effect: a move from a character with no
// dragon is dropped with a warning and no event (FR-4.4).
//
// Since the serverbound packet carries no identity field, "does this sender own
// the named dragon" is unrepresentable — the only check left is "does this
// sender have a dragon at all", which is this lookup.
func (p *ProcessorImpl) Move(characterId uint32, startX int16, startY int16, stance byte, rawMovement []byte) error {
	if _, err := GetRegistry().Get(p.ctx, p.t, characterId); err != nil {
		p.l.Warnf("Move for character [%d] with no dragon; dropped.", characterId)
		return nil
	}
	m, err := GetRegistry().Update(p.ctx, p.t, characterId, func(cur Model) Model {
		return cur.Move(int32(startX), int32(startY), stance)
	})
	if err != nil {
		return err
	}
	return p.emit(EnvEventTopicDragonStatus, movedEventProvider(m, rawMovement))
}
```

- [ ] **Step 5: Run the tests**

Run: `cd services/atlas-dragons/atlas.com/dragons && go test -race ./... -v`
Expected: PASS (all model, registry, producer, processor tests).

- [ ] **Step 6: Commit**

```bash
git add services/atlas-dragons/atlas.com/dragons
git commit -m "feat(task-225): dragon processor with idempotent create/destroy and move relay"
```

---

### Task 8: `atlas-dragons` REST surface

**Files:**
- Create: `services/atlas-dragons/atlas.com/dragons/dragon/resource.go`
- Create: `services/atlas-dragons/atlas.com/dragons/dragon/rest.go`
- Create: `services/atlas-dragons/atlas.com/dragons/world/resource.go`
- Modify: `services/atlas-dragons/atlas.com/dragons/main.go`

**Interfaces:**
- Consumes: `dragon.Processor` (Task 7).
- Produces: `dragon.RestModel` with `GetName() == "dragons"`; `dragon.Transform(m Model) (RestModel, error)`; `dragon.InitResource(si) server.RouteInitializer` serving `GET /dragons/{characterId}`; `world.InitResource(si) server.RouteInitializer` serving `GET /worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/dragons`.

- [ ] **Step 1: Write `resource.go`**

```go
package dragon

import (
	"strconv"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// RestModel is the JSON:API representation. The resource id is the OWNER
// character id: the dragon has no identity of its own.
type RestModel struct {
	Id               string     `json:"-"`
	OwnerCharacterId uint32     `json:"ownerCharacterId"`
	X                int32      `json:"x"`
	Y                int32      `json:"y"`
	Stance           byte       `json:"stance"`
	JobId            uint16     `json:"jobId"`
	WorldId          world.Id   `json:"worldId"`
	ChannelId        channel.Id `json:"channelId"`
	MapId            _map.Id    `json:"mapId"`
	Instance         uuid.UUID  `json:"instance"`
}

func (m RestModel) GetID() string { return m.Id }

func (m *RestModel) SetID(idStr string) error {
	m.Id = idStr
	return nil
}

func (m RestModel) GetName() string { return "dragons" }

func Transform(m Model) (RestModel, error) {
	f := m.Field()
	return RestModel{
		Id:               strconv.Itoa(int(m.OwnerCharacterId())),
		OwnerCharacterId: m.OwnerCharacterId(),
		X:                m.X(),
		Y:                m.Y(),
		Stance:           m.Stance(),
		JobId:            uint16(m.JobId()),
		WorldId:          f.WorldId(),
		ChannelId:        f.ChannelId(),
		MapId:            f.MapId(),
		Instance:         f.Instance(),
	}, nil
}

func Extract(m RestModel) (Model, error) {
	f := field.NewBuilder(m.WorldId, m.ChannelId, m.MapId).SetInstance(m.Instance).Build()
	return NewBuilder(m.OwnerCharacterId).
		SetField(f).
		SetX(m.X).
		SetY(m.Y).
		SetStance(m.Stance).
		SetJobId(job.Id(m.JobId)).
		Build(), nil
}
```

`Extract` is unused inside `atlas-dragons` itself but keeps the transform pair
symmetrical; if the linter flags it as dead code, drop `Extract` and its `field`
and `job` imports — `atlas-channel` declares its own (Task 10).

- [ ] **Step 2: Write `rest.go`**

```go
package dragon

import (
	"atlas-dragons/rest"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

const getDragon = "get_dragon"

func InitResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		r := router.PathPrefix("/dragons").Subrouter()
		r.HandleFunc("/{characterId}", rest.RegisterHandler(l)(si)(getDragon, handleGetDragonByCharacterId)).Methods(http.MethodGet)
	}
}

// handleGetDragonByCharacterId returns 404 for a character with no dragon.
// THAT IS THE NORMAL ANSWER for the overwhelming majority of characters — every
// non-Evan in the game. Consumers must treat requests.ErrNotFound as "no
// dragon" and continue; a consumer that logs it as a fetch failure emits one
// error line per non-Evan character.
func handleGetDragonByCharacterId(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			p := NewProcessor(d.Logger(), d.Context())
			m, err := p.GetByCharacterId(characterId)
			if err != nil {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			res, err := model.Map(Transform)(model.FixedProvider(m))()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(r.URL.Query())(res)
		}
	})
}
```

- [ ] **Step 3: Write `world/resource.go`**

Copy the shape of `services/atlas-summons/atlas.com/summons/world/resource.go`
exactly, substituting dragons. The list is field-scoped and paginated so
`atlas-channel` can drain it with `requests.DrainProvider`:

```go
package world

import (
	"atlas-dragons/dragon"
	"atlas-dragons/rest"
	"net/http"
	"sort"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate"
)

const getDragonsInMap = "get_dragons_in_map"

func InitResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		r := router.PathPrefix("/worlds").Subrouter()
		r.HandleFunc("/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/dragons",
			rest.RegisterHandler(l)(si)(getDragonsInMap, handleGetDragonsInMap)).Methods(http.MethodGet)
	}
}

func handleGetDragonsInMap(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseWorldId(d.Logger(), func(worldId world.Id) http.HandlerFunc {
		return rest.ParseChannelId(d.Logger(), func(channelId channel.Id) http.HandlerFunc {
			return rest.ParseMapId(d.Logger(), func(mapId _map.Id) http.HandlerFunc {
				return rest.ParseInstanceId(d.Logger(), func(instance uuid.UUID) http.HandlerFunc {
					return func(w http.ResponseWriter, r *http.Request) {
						page, err := paginate.ParseParams(r.URL.Query(), paginate.MaxPageSize, paginate.MaxPageSize)
						if err != nil {
							server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
							return
						}

						f := field.NewBuilder(worldId, channelId, mapId).SetInstance(instance).Build()
						ms, err := dragon.NewProcessor(d.Logger(), d.Context()).GetInField(f)
						if err != nil {
							d.Logger().WithError(err).Errorf("Unable to retrieve dragons in field.")
							w.WriteHeader(http.StatusInternalServerError)
							return
						}

						sorted := make([]dragon.Model, len(ms))
						copy(sorted, ms)
						sort.Slice(sorted, func(i, j int) bool {
							return sorted[i].OwnerCharacterId() < sorted[j].OwnerCharacterId()
						})
						paged := paginate.Slice(sorted, page)

						res, err := model.SliceMap(dragon.Transform)(model.FixedProvider(paged.Items))(model.ParallelMap())()
						if err != nil {
							d.Logger().WithError(err).Errorf("Creating REST models.")
							w.WriteHeader(http.StatusInternalServerError)
							return
						}
						server.MarshalResponse[[]dragon.RestModel](d.Logger())(w)(c.ServerInformation())(r.URL.Query())(res)
					}
				})
			})
		})
	})
}
```

Match the exact tail of `atlas-summons`' `handleGetSummonsInMap` (the
`MarshalResponse` call and any pagination-link decoration it applies) — read it
and copy, do not improvise.

- [ ] **Step 4: Wire the resources and the registry into `main.go`**

In `services/atlas-dragons/atlas.com/dragons/main.go`, replace `_ = rc` with
`dragon.InitRegistry(rc)` and add the two route initializers:

```go
	rc := atlas.Connect(l)
	dragon.InitRegistry(rc)
```

```go
		AddRouteInitializer(dragon.InitResource(GetServer())).
		AddRouteInitializer(world.InitResource(GetServer())).
```

with imports `"atlas-dragons/dragon"` and `"atlas-dragons/world"`.

- [ ] **Step 5: Build and test**

Run: `cd services/atlas-dragons/atlas.com/dragons && go build ./... && go vet ./... && go test -race ./...`
Expected: clean.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-dragons/atlas.com/dragons
git commit -m "feat(task-225): atlas-dragons JSON:API REST surface"
```

---

### Task 9: `atlas-dragons` Kafka consumers

**Files:**
- Create: `services/atlas-dragons/atlas.com/dragons/kafka/consumer/dragon/consumer.go`
- Create: `services/atlas-dragons/atlas.com/dragons/kafka/consumer/character/consumer.go`
- Modify: `services/atlas-dragons/atlas.com/dragons/main.go`

**Interfaces:**
- Consumes: `dragon.Processor` (Task 7), the contracts from Task 6.
- Produces: `dragoncmd.InitConsumers(l)(cmf)(groupId)`, `dragoncmd.InitHandlers(l)(rf) error`, `characterevt.InitConsumers(l)(cmf)(groupId)`, `characterevt.InitHandlers(l)(rf) error`.

**Lifecycle-event ownership (a refinement of design §5.1, with reason):**
`atlas-dragons` handles the four character-status events whose bodies carry a
field — `LOGIN`, `LOGOUT`, `MAP_CHANGED`, `CHANNEL_CHANGED` — plus the *destroy*
half of `JOB_CHANGED`, which needs no field because the registry already holds
one.

`JOB_CHANGED` carries only `channelId` and `jobId` (verified:
`services/atlas-character/.../kafka/message/character/kafka.go:295-298`), and
`GET /characters/{id}` does not return a map id (`character/rest.go:41-43` —
`MapId` is create-time input only). `atlas-dragons` therefore cannot resolve the
field to *create* on a job change; `atlas-channel` can, because it holds the live
session and `s.Field()`. So the split is:

| Direction | Owner | Why |
|---|---|---|
| job change **out of** range → destroy | `atlas-dragons` (`handleJobChanged`, below) | needs no field — the registry has it |
| job change **into** range → create | `atlas-channel` emits `COMMAND_TOPIC_DRAGON` `CREATE` (Task 12, Step 5) | needs the field, which only the session has |

`atlas-channel` needs no job predicate of its own: it emits `CREATE`
unconditionally on any `JOB_CHANGED` for a session it owns, and
`Processor.Create`'s internal `HasDragon` gate (Task 7) decides. The predicate
stays defined in exactly one place.

- [ ] **Step 1: Write the command consumer**

`kafka/consumer/dragon/consumer.go`:

```go
package dragon

import (
	consumer2 "atlas-dragons/kafka/consumer"
	dragonstate "atlas-dragons/dragon"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("dragon_command")(EnvCommandTopic)(consumerGroupId),
				consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		t, _ := topic.EnvProvider(l)(EnvCommandTopic)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCreateCommand))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleDestroyCommand))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleMoveCommand))); err != nil {
			return err
		}
		return nil
	}
}

func handleCreateCommand(l logrus.FieldLogger, ctx context.Context, c Command[CreateCommandBody]) {
	if c.Type != CommandTypeCreate {
		return
	}
	f := field.NewBuilder(c.WorldId, c.ChannelId, c.MapId).SetInstance(c.Instance).Build()
	if err := dragonstate.NewProcessor(l, ctx).Create(f, c.Body.CharacterId); err != nil {
		l.WithError(err).Errorf("Failed to create dragon for character [%d].", c.Body.CharacterId)
	}
}

func handleDestroyCommand(l logrus.FieldLogger, ctx context.Context, c Command[DestroyCommandBody]) {
	if c.Type != CommandTypeDestroy {
		return
	}
	if err := dragonstate.NewProcessor(l, ctx).Destroy(c.Body.CharacterId); err != nil {
		l.WithError(err).Errorf("Failed to destroy dragon for character [%d].", c.Body.CharacterId)
	}
}

func handleMoveCommand(l logrus.FieldLogger, ctx context.Context, c Command[MoveCommandBody]) {
	if c.Type != CommandTypeMove {
		return
	}
	if err := dragonstate.NewProcessor(l, ctx).Move(c.Body.CharacterId, c.Body.StartX, c.Body.StartY, c.Body.Stance, c.Body.RawMovement); err != nil {
		l.WithError(err).Errorf("Failed to move dragon for character [%d].", c.Body.CharacterId)
	}
}
```

- [ ] **Step 2: Write the character-status consumer**

`kafka/consumer/character/consumer.go`:

```go
package character

import (
	dragonstate "atlas-dragons/dragon"
	consumer2 "atlas-dragons/kafka/consumer"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("character_status_event")(EnvEventTopicCharacterStatus)(consumerGroupId),
				consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		t, _ := topic.EnvProvider(l)(EnvEventTopicCharacterStatus)()
		for _, h := range []handler.Handler{
			message.AdaptHandler(message.PersistentConfig(handleLogin)),
			message.AdaptHandler(message.PersistentConfig(handleLogout)),
			message.AdaptHandler(message.PersistentConfig(handleMapChanged)),
			message.AdaptHandler(message.PersistentConfig(handleChannelChanged)),
			message.AdaptHandler(message.PersistentConfig(handleJobChanged)),
		} {
			if _, err := rf(t, h); err != nil {
				return err
			}
		}
		return nil
	}
}

// handleJobChanged owns the DESTROY half of a job change: a character that left
// the dragon-bearing range loses its dragon. It needs no field — the registry
// already holds one — which is why this half lives here while the CREATE half
// lives channel-side (see the ownership table above).
//
// A stage-up within the range (2210 -> 2211) resolves to HasDragon == true and
// is left alone; the channel-side CREATE refreshes the stored jobId.
//
// What the player sees: the dragon stops moving and stops generating traffic
// immediately, but the already-rendered dragon persists on every client in the
// field until the owner next leaves it. The client has no REMOVE_DRAGON handler
// arm; the only client-side teardown is destroying the owner's CUser. This is a
// client limitation, expected, and not a bug to chase.
func handleJobChanged(l logrus.FieldLogger, ctx context.Context, e StatusEvent[JobChangedBody]) {
	if e.Type != StatusEventTypeJobChanged {
		return
	}
	if dragonstate.HasDragon(tenant.MustFromContext(ctx), e.Body.JobId) {
		return
	}
	if err := dragonstate.NewProcessor(l, ctx).Destroy(e.CharacterId); err != nil {
		l.WithError(err).Errorf("Failed to destroy dragon on job change for character [%d].", e.CharacterId)
	}
}

// handleLogin creates the dragon in the field the character logged into. Create
// is job-gated internally, so a non-Evan login is a cheap no-op.
func handleLogin(l logrus.FieldLogger, ctx context.Context, e StatusEvent[LoginBody]) {
	if e.Type != StatusEventTypeLogin {
		return
	}
	f := field.NewBuilder(e.WorldId, e.Body.ChannelId, e.Body.MapId).SetInstance(e.Body.Instance).Build()
	if err := dragonstate.NewProcessor(l, ctx).Create(f, e.CharacterId); err != nil {
		l.WithError(err).Errorf("Failed to create dragon on login for character [%d].", e.CharacterId)
	}
}

// handleLogout destroys the dragon: no dragon may outlive its owner's presence
// in a field (FR-1.5).
func handleLogout(l logrus.FieldLogger, ctx context.Context, e StatusEvent[LogoutBody]) {
	if e.Type != StatusEventTypeLogout {
		return
	}
	if err := dragonstate.NewProcessor(l, ctx).Destroy(e.CharacterId); err != nil {
		l.WithError(err).Errorf("Failed to destroy dragon on logout for character [%d].", e.CharacterId)
	}
}

// handleMapChanged destroys the dragon in the old field and recreates it in the
// new one. Destroy-then-create rather than a field update, so the old map gets
// its DESTROYED broadcast and the new map gets a SPAWN_DRAGON — exactly one of
// each, no orphan.
func handleMapChanged(l logrus.FieldLogger, ctx context.Context, e StatusEvent[MapChangedBody]) {
	if e.Type != StatusEventTypeMapChanged {
		return
	}
	p := dragonstate.NewProcessor(l, ctx)
	if err := p.Destroy(e.CharacterId); err != nil {
		l.WithError(err).Errorf("Failed to destroy dragon on map change for character [%d].", e.CharacterId)
	}
	f := field.NewBuilder(e.WorldId, e.Body.ChannelId, e.Body.TargetMapId).SetInstance(e.Body.TargetInstance).Build()
	if err := p.Create(f, e.CharacterId); err != nil {
		l.WithError(err).Errorf("Failed to create dragon on map change for character [%d].", e.CharacterId)
	}
}

// handleChannelChanged mirrors handleMapChanged across channels.
func handleChannelChanged(l logrus.FieldLogger, ctx context.Context, e StatusEvent[ChannelChangedBody]) {
	if e.Type != StatusEventTypeChannelChanged {
		return
	}
	p := dragonstate.NewProcessor(l, ctx)
	if err := p.Destroy(e.CharacterId); err != nil {
		l.WithError(err).Errorf("Failed to destroy dragon on channel change for character [%d].", e.CharacterId)
	}
	f := field.NewBuilder(e.WorldId, e.Body.ChannelId, e.Body.MapId).SetInstance(e.Body.Instance).Build()
	if err := p.Create(f, e.CharacterId); err != nil {
		l.WithError(err).Errorf("Failed to create dragon on channel change for character [%d].", e.CharacterId)
	}
}
```

- [ ] **Step 3: Wire both consumers into `main.go`**

Add after the registry init:

```go
	cmf := consumer.GetManager().AddConsumer(l, rt.Context(), rt.WaitGroup())
	dragoncmd.InitConsumers(l)(cmf)(consumerGroupId)
	characterevt.InitConsumers(l)(cmf)(consumerGroupId)
	if err := dragoncmd.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register dragon command handlers.")
	}
	if err := characterevt.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register character status handlers.")
	}
```

with imports:

```go
	characterevt "atlas-dragons/kafka/consumer/character"
	dragoncmd "atlas-dragons/kafka/consumer/dragon"
```

- [ ] **Step 4: Build and test**

Run: `cd services/atlas-dragons/atlas.com/dragons && go build ./... && go vet ./... && go test -race ./...`
Expected: clean.

- [ ] **Step 5: Run the goroutine guard**

Run: `tools/goroutine-guard.sh` from the repo root.
Expected: exit 0 — `atlas-dragons` spawns no bare goroutines.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-dragons/atlas.com/dragons
git commit -m "feat(task-225): atlas-dragons Kafka consumers for dragon commands and character lifecycle"
```

---

### Task 10: `atlas-channel` dragon contract mirror and client

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/kafka/message/dragon/kafka.go`
- Create: `services/atlas-channel/atlas.com/channel/dragon/model.go`
- Create: `services/atlas-channel/atlas.com/channel/dragon/requests.go`
- Create: `services/atlas-channel/atlas.com/channel/dragon/processor.go`
- Create: `services/atlas-channel/atlas.com/channel/dragon/producer.go`
- Test: `services/atlas-channel/atlas.com/channel/kafka/message/dragon/kafka_test.go`

**Interfaces:**
- Consumes: the `atlas-dragons` contracts (Task 6) — as a hand-mirrored copy across a module boundary.
- Produces: `dragonmsg.EnvCommandTopic`, `CommandTypeCreate/Destroy/Move`, `Command[E]`, `CreateCommandBody`, `DestroyCommandBody`, `MoveCommandBody`; `dragonmsg.EnvEventTopicDragonStatus`, `EventDragonStatusCreated/Moved/Destroyed`, `StatusEvent[E]`, `StatusEventCreatedBody`, `StatusEventMovedBody`, `StatusEventDestroyedBody`; `dragon.Model` (channel-side) with `OwnerCharacterId()`, `X() int32`, `Y() int32`, `Stance() byte`, `JobId() uint16`; `dragon.Processor` with `InMapModelProvider(f field.Model) model.Provider[[]Model]`, `ForEachInMap(f field.Model, o model.Operator[Model]) error`, `Create(f field.Model, characterId uint32) error`, `Destroy(f field.Model, characterId uint32) error`, `Move(f field.Model, characterId uint32, startX, startY int16, stance byte, rawMovement []byte) error`.

- [ ] **Step 1: Write the failing contract-mirror test**

The two contract copies live in separate Go modules. A json tag renamed in one
and not the other fails no build and decodes into a zero-valued body at runtime.
This test is the only thing standing between that and a silent production bug.

`services/atlas-channel/atlas.com/channel/kafka/message/dragon/kafka_test.go`:

```go
package dragon

import (
	"encoding/json"
	"testing"
)

// The producer-side JSON is pinned literally here rather than imported from
// atlas-dragons: the two contracts live in separate modules, so importing one
// into the other's test would defeat the purpose. If atlas-dragons changes a
// json tag, this test fails and the mirror gets updated deliberately.
//
// Producer definitions: services/atlas-dragons/atlas.com/dragons/dragon/kafka.go
// and .../kafka/consumer/dragon/kafka.go.
func TestCreatedEventMirrorsProducerTags(t *testing.T) {
	raw := []byte(`{
		"worldId": 0, "channelId": 1, "mapId": 100000000,
		"instance": "00000000-0000-0000-0000-000000000000",
		"ownerCharacterId": 4242, "type": "CREATED",
		"body": {"x": 70000, "y": -70000, "stance": 3, "jobId": 2214}
	}`)

	var e StatusEvent[StatusEventCreatedBody]
	if err := json.Unmarshal(raw, &e); err != nil {
		t.Fatal(err)
	}
	if e.OwnerCharacterId != 4242 || e.Type != EventDragonStatusCreated ||
		e.WorldId != 0 || e.ChannelId != 1 || e.MapId != 100000000 {
		t.Fatalf("envelope fields did not survive: %+v", e)
	}
	if e.Body.X != 70000 || e.Body.Y != -70000 || e.Body.Stance != 3 || e.Body.JobId != 2214 {
		t.Fatalf("created body did not survive: %+v", e.Body)
	}
}

func TestMovedEventMirrorsProducerTags(t *testing.T) {
	raw := []byte(`{
		"worldId": 0, "channelId": 1, "mapId": 100000000,
		"instance": "00000000-0000-0000-0000-000000000000",
		"ownerCharacterId": 4242, "type": "MOVED",
		"body": {"rawMovement": "AQID"}
	}`)

	var e StatusEvent[StatusEventMovedBody]
	if err := json.Unmarshal(raw, &e); err != nil {
		t.Fatal(err)
	}
	if e.Type != EventDragonStatusMoved || len(e.Body.RawMovement) != 3 {
		t.Fatalf("moved body did not survive: %+v", e)
	}
}

func TestDestroyedEventMirrorsProducerTags(t *testing.T) {
	raw := []byte(`{
		"worldId": 0, "channelId": 1, "mapId": 100000000,
		"instance": "00000000-0000-0000-0000-000000000000",
		"ownerCharacterId": 4242, "type": "DESTROYED", "body": {}
	}`)

	var e StatusEvent[StatusEventDestroyedBody]
	if err := json.Unmarshal(raw, &e); err != nil {
		t.Fatal(err)
	}
	if e.Type != EventDragonStatusDestroyed || e.OwnerCharacterId != 4242 {
		t.Fatalf("destroyed envelope did not survive: %+v", e)
	}
}

// The command direction is channel -> dragons, so this asserts what the
// CONSUMER in atlas-dragons will see when it decodes what we produce.
func TestMoveCommandMirrorsConsumerTags(t *testing.T) {
	c := Command[MoveCommandBody]{
		WorldId: 0, ChannelId: 1, MapId: 100000000,
		Type: CommandTypeMove,
		Body: MoveCommandBody{CharacterId: 4242, StartX: 100, StartY: -200, Stance: 3, RawMovement: []byte{1, 2, 3}},
	}
	b, err := json.Marshal(&c)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	body, ok := m["body"].(map[string]interface{})
	if !ok {
		t.Fatalf("no body object: %s", b)
	}
	for _, k := range []string{"characterId", "startX", "startY", "stance", "rawMovement"} {
		if _, present := body[k]; !present {
			t.Errorf("MoveCommandBody must serialize a %q key (atlas-dragons decodes it)", k)
		}
	}
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./kafka/message/dragon/... -v`
Expected: FAIL — package does not exist.

- [ ] **Step 3: Write the mirror**

`services/atlas-channel/atlas.com/channel/kafka/message/dragon/kafka.go` — the
bodies and envelopes are byte-identical in json tags to the definitions listed in
Task 6. Copy them, changing only the doc comment to name the mirror direction:

```go
package dragon

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// This file MIRRORS the atlas-dragons contract across a module boundary.
// Command direction: channel -> dragons (producer here, consumer there).
// Event direction: dragons -> channel (producer there, consumer here).
//
// Authoritative definitions:
//   services/atlas-dragons/atlas.com/dragons/kafka/consumer/dragon/kafka.go
//   services/atlas-dragons/atlas.com/dragons/dragon/kafka.go
//
// Every json tag MUST stay byte-for-byte identical. The two files are in
// separate Go modules, so a tag changed in one and not the other fails no build
// — it decodes into a zero-valued body at runtime, silently. kafka_test.go
// pins the wire shape from literal JSON so that divergence fails a test instead.

const EnvCommandTopic = "COMMAND_TOPIC_DRAGON"

const (
	CommandTypeCreate  = "CREATE"
	CommandTypeDestroy = "DESTROY"
	CommandTypeMove    = "MOVE"
)

type Command[E any] struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
	Type      string     `json:"type"`
	Body      E          `json:"body"`
}

type CreateCommandBody struct {
	CharacterId uint32 `json:"characterId"`
}

type DestroyCommandBody struct {
	CharacterId uint32 `json:"characterId"`
}

type MoveCommandBody struct {
	CharacterId uint32 `json:"characterId"`
	StartX      int16  `json:"startX"`
	StartY      int16  `json:"startY"`
	Stance      byte   `json:"stance"`
	RawMovement []byte `json:"rawMovement"`
}

const EnvEventTopicDragonStatus = "EVENT_TOPIC_DRAGON_STATUS"

const (
	EventDragonStatusCreated   = "CREATED"
	EventDragonStatusMoved     = "MOVED"
	EventDragonStatusDestroyed = "DESTROYED"
)

type StatusEvent[E any] struct {
	WorldId          world.Id   `json:"worldId"`
	ChannelId        channel.Id `json:"channelId"`
	MapId            _map.Id    `json:"mapId"`
	Instance         uuid.UUID  `json:"instance"`
	OwnerCharacterId uint32     `json:"ownerCharacterId"`
	Type             string     `json:"type"`
	Body             E          `json:"body"`
}

// X/Y are int32: SPAWN_DRAGON encodes 4-byte coordinates.
type StatusEventCreatedBody struct {
	X      int32  `json:"x"`
	Y      int32  `json:"y"`
	Stance byte   `json:"stance"`
	JobId  uint16 `json:"jobId"`
}

type StatusEventMovedBody struct {
	RawMovement []byte `json:"rawMovement"`
}

type StatusEventDestroyedBody struct{}
```

- [ ] **Step 4: Run the mirror tests**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./kafka/message/dragon/... -v`
Expected: PASS (4 tests).

- [ ] **Step 5: Write the channel-side dragon client**

`services/atlas-channel/atlas.com/channel/dragon/model.go`:

```go
package dragon

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// Model is the channel-side view of a dragon fetched from atlas-dragons.
// X/Y are int32 — SPAWN_DRAGON encodes 4-byte coordinates.
type Model struct {
	ownerCharacterId uint32
	x                int32
	y                int32
	stance           byte
	jobId            uint16
}

func (m Model) OwnerCharacterId() uint32 { return m.ownerCharacterId }
func (m Model) X() int32                 { return m.x }
func (m Model) Y() int32                 { return m.y }
func (m Model) Stance() byte             { return m.stance }
func (m Model) JobId() uint16            { return m.jobId }

type RestModel struct {
	Id               string     `json:"-"`
	OwnerCharacterId uint32     `json:"ownerCharacterId"`
	X                int32      `json:"x"`
	Y                int32      `json:"y"`
	Stance           byte       `json:"stance"`
	JobId            uint16     `json:"jobId"`
	WorldId          world.Id   `json:"worldId"`
	ChannelId        channel.Id `json:"channelId"`
	MapId            _map.Id    `json:"mapId"`
	Instance         uuid.UUID  `json:"instance"`
}

func (r RestModel) GetName() string { return "dragons" }

func (r RestModel) GetID() string { return r.Id }

func (r *RestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}

func Extract(r RestModel) (Model, error) {
	return Model{
		ownerCharacterId: r.OwnerCharacterId,
		x:                r.X,
		y:                r.Y,
		stance:           r.Stance,
		jobId:            r.JobId,
	}, nil
}
```

`services/atlas-channel/atlas.com/channel/dragon/requests.go`:

```go
package dragon

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const dragonsInMapResource = "worlds/%d/channels/%d/maps/%d/instances/%s/dragons"

func getBaseRequest() string {
	return requests.RootUrl("DRAGONS")
}

// inMapUrl returns the list URL for the dragons currently in one map instance.
// It is a bare URL (not a requests.Request) because the list is paginated
// server-side and consumed via requests.DrainProvider, which appends its own
// page[number]/page[size] params.
func inMapUrl(f field.Model) string {
	return fmt.Sprintf(getBaseRequest()+dragonsInMapResource, f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String())
}
```

`services/atlas-channel/atlas.com/channel/dragon/processor.go`:

```go
package dragon

import (
	dragonmsg "atlas-channel/kafka/message/dragon"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

type Processor interface {
	InMapModelProvider(f field.Model) model.Provider[[]Model]
	ForEachInMap(f field.Model, o model.Operator[Model]) error
	Create(f field.Model, characterId uint32) error
	Destroy(f field.Model, characterId uint32) error
	Move(f field.Model, characterId uint32, startX int16, startY int16, stance byte, rawMovement []byte) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

var _ Processor = (*ProcessorImpl)(nil)

// InMapModelProvider drains every page of the field-scoped dragon list. A
// truncated list means some existing dragons silently fail to replay to an
// entering character.
func (p *ProcessorImpl) InMapModelProvider(f field.Model) model.Provider[[]Model] {
	return requests.DrainProvider[RestModel, Model](p.l, p.ctx)(inMapUrl(f), 250, Extract, model.Filters[Model]())
}

func (p *ProcessorImpl) ForEachInMap(f field.Model, o model.Operator[Model]) error {
	return model.ForEachSlice(p.InMapModelProvider(f), o, model.ParallelExecute())
}

func (p *ProcessorImpl) Create(f field.Model, characterId uint32) error {
	p.l.Debugf("Requesting dragon create for character [%d] in map [%d].", characterId, f.MapId())
	return producer.ProviderImpl(p.l)(p.ctx)(dragonmsg.EnvCommandTopic)(CreateCommandProvider(f, characterId))
}

func (p *ProcessorImpl) Destroy(f field.Model, characterId uint32) error {
	p.l.Debugf("Requesting dragon destroy for character [%d].", characterId)
	return producer.ProviderImpl(p.l)(p.ctx)(dragonmsg.EnvCommandTopic)(DestroyCommandProvider(f, characterId))
}

func (p *ProcessorImpl) Move(f field.Model, characterId uint32, startX int16, startY int16, stance byte, rawMovement []byte) error {
	return producer.ProviderImpl(p.l)(p.ctx)(dragonmsg.EnvCommandTopic)(MoveCommandProvider(f, characterId, startX, startY, stance, rawMovement))
}
```

`services/atlas-channel/atlas.com/channel/dragon/producer.go`:

```go
package dragon

import (
	dragonmsg "atlas-channel/kafka/message/dragon"

	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func CreateCommandProvider(f field.Model, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &dragonmsg.Command[dragonmsg.CreateCommandBody]{
		WorldId: f.WorldId(), ChannelId: f.ChannelId(), MapId: f.MapId(), Instance: f.Instance(),
		Type: dragonmsg.CommandTypeCreate,
		Body: dragonmsg.CreateCommandBody{CharacterId: characterId},
	}
	return producer.SingleMessageProvider(key, value)
}

func DestroyCommandProvider(f field.Model, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &dragonmsg.Command[dragonmsg.DestroyCommandBody]{
		WorldId: f.WorldId(), ChannelId: f.ChannelId(), MapId: f.MapId(), Instance: f.Instance(),
		Type: dragonmsg.CommandTypeDestroy,
		Body: dragonmsg.DestroyCommandBody{CharacterId: characterId},
	}
	return producer.SingleMessageProvider(key, value)
}

// MoveCommandProvider keys on the OWNER character id, not the map: dragon moves
// for one character must stay ordered relative to each other.
func MoveCommandProvider(f field.Model, characterId uint32, startX int16, startY int16, stance byte, rawMovement []byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &dragonmsg.Command[dragonmsg.MoveCommandBody]{
		WorldId: f.WorldId(), ChannelId: f.ChannelId(), MapId: f.MapId(), Instance: f.Instance(),
		Type: dragonmsg.CommandTypeMove,
		Body: dragonmsg.MoveCommandBody{
			CharacterId: characterId, StartX: startX, StartY: startY, Stance: stance, RawMovement: rawMovement,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
```

- [ ] **Step 6: Build and test**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go vet ./... && go test -race ./kafka/message/dragon/... ./dragon/...`
Expected: clean.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/message/dragon services/atlas-channel/atlas.com/channel/dragon
git commit -m "feat(task-225): atlas-channel dragon contract mirror and REST/Kafka client"
```

---

### Task 11: `atlas-channel` writers and the serverbound move handler

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/socket/writer/dragon.go`
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/dragon_move.go`
- Modify: `services/atlas-channel/atlas.com/channel/main.go` (writer-name list ~`:653`, `handlerMap` ~`:858`)

**Interfaces:**
- Consumes: the codecs (Task 1), `dragon.Processor` (Task 10).
- Produces: `writer.DragonSpawnBody(ownerCharacterId uint32, x, y int32, stance byte, jobId uint16) packet.Encode`; `writer.DragonMoveBody(ownerCharacterId uint32, rawMovement []byte) packet.Encode`; `writer.DragonRemoveBody(ownerCharacterId uint32) packet.Encode`; `handler.DragonMoveHandleFunc(l, ctx, wp) func(s session.Model, r *request.Reader, readerOptions map[string]interface{})`.

- [ ] **Step 1: Write `socket/writer/dragon.go`**

```go
package writer

import (
	dragoncb "github.com/Chronicle20/atlas/libs/atlas-packet/dragon/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

// DragonSpawnBody builds the SPAWN_DRAGON packet for an Evan's dragon,
// broadcast to every session in the owner's map INCLUDING the owner.
// x/y are int32: this packet encodes 4-byte coordinates.
func DragonSpawnBody(ownerCharacterId uint32, x int32, y int32, stance byte, jobId uint16) packet.Encode {
	return dragoncb.NewDragonSpawn(ownerCharacterId, x, y, stance, jobId).Encode
}

// DragonMoveBody builds the MOVE_DRAGON packet, rebroadcasting the raw CMovePath
// blob byte-faithfully to OTHER sessions. The blob already carries the start
// position, so it is not written separately.
func DragonMoveBody(ownerCharacterId uint32, rawMovement []byte) packet.Encode {
	return dragoncb.NewDragonMove(ownerCharacterId, rawMovement).Encode
}

// DragonRemoveBody builds the REMOVE_DRAGON packet. The client has no handler
// arm for this opcode and discards it — the dragon disappears because the
// owner's CUser is destroyed when they leave the field. Sending it is correct
// and harmless, but it is not the removal mechanism. See dragoncb.DragonRemove.
func DragonRemoveBody(ownerCharacterId uint32) packet.Encode {
	return dragoncb.NewDragonRemove(ownerCharacterId).Encode
}
```

- [ ] **Step 2: Write `socket/handler/dragon_move.go`**

```go
package handler

import (
	dragoncmd "atlas-channel/dragon"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-packet/dragon/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// DragonMoveHandleFunc decodes an inbound MOVE_DRAGON packet and emits a
// COMMAND_TOPIC_DRAGON MOVE command keyed on the SENDING SESSION's character id.
//
// The packet carries no identity field at all (CVecCtrlDragon::EndUpdateActive
// writes only the CMovePath blob), so the session IS the identity: there is no
// id to reconcile and no cross-character spoofing surface. atlas-dragons drops
// the command if the sender has no dragon; it never creates one as a side
// effect.
func DragonMoveHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := serverbound.Move{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		_ = dragoncmd.NewProcessor(l, ctx).Move(s.Field(), s.CharacterId(), p.StartX(), p.StartY(), 0, p.RawMovement())
	}
}
```

- [ ] **Step 3: Register the writers and the handler in `main.go`**

Add to the writer-name list immediately after `summoncb.SummonSkillWriter` (~line 653):

```go
		dragoncb.DragonSpawnWriter,
		dragoncb.DragonMoveWriter,
		dragoncb.DragonRemoveWriter,
```

Add to `handlerMap` immediately after `handlerMap[summonsb.SummonDamageHandle] = ...` (~line 858):

```go
	handlerMap[dragonsb.DragonMoveHandle] = handler.DragonMoveHandleFunc
```

with imports:

```go
	dragoncb "github.com/Chronicle20/atlas/libs/atlas-packet/dragon/clientbound"
	dragonsb "github.com/Chronicle20/atlas/libs/atlas-packet/dragon/serverbound"
```

- [ ] **Step 4: Build**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go vet ./...`
Expected: clean.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel
git commit -m "feat(task-225): atlas-channel dragon writers and MOVE_DRAGON handler"
```

---

### Task 12: `atlas-channel` dragon status consumer and the JOB_CHANGED create

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/kafka/consumer/dragon/consumer.go`
- Test: `services/atlas-channel/atlas.com/channel/kafka/consumer/dragon/consumer_test.go`
- Modify: `services/atlas-channel/atlas.com/channel/kafka/message/character/kafka.go` (add `JobChangedStatusEventBody`)
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/character/consumer.go` (add the JOB_CHANGED handler + its `InitHandlers` registration)
- Modify: `services/atlas-channel/atlas.com/channel/main.go` (`InitConsumers` ~`:224`, `register(...InitHandlers...)` ~`:459`)

**Interfaces:**
- Consumes: `dragonmsg.StatusEvent[...]` (Task 10), `writer.Dragon*Body` (Task 11), `dragoncmd.Processor` (Task 10).
- Produces: `dragonConsumer.InitConsumers(l)(cmf)(groupId)`; `dragonConsumer.InitHandlers(l)(sc)(wp)(rf) ([]listener.HandlerHandle, error)`; a `handleStatusEventJobChanged(sc, wp)` handler in the character consumer.

- [ ] **Step 1: Write the failing fan-out tests**

The three broadcast handlers differ only in *who receives what*, which is exactly
the part a reader gets wrong. Assert it directly.

`services/atlas-channel/atlas.com/channel/kafka/consumer/dragon/consumer_test.go`:

```go
package dragon

import (
	dragonmsg "atlas-channel/kafka/message/dragon"
	"testing"
)

// recipientPolicy is the single fact each handler encodes: CREATED and
// DESTROYED go map-wide including the owner; MOVED excludes the owner because
// their client already rendered the motion locally and re-sending double-applies
// it (the same reasoning as the summon move relay).
func TestRecipientPolicyPerEventType(t *testing.T) {
	cases := []struct {
		eventType     string
		excludesOwner bool
	}{
		{dragonmsg.EventDragonStatusCreated, false},
		{dragonmsg.EventDragonStatusMoved, true},
		{dragonmsg.EventDragonStatusDestroyed, false},
	}
	for _, c := range cases {
		if got := excludesOwner(c.eventType); got != c.excludesOwner {
			t.Errorf("%s: excludesOwner = %v, want %v", c.eventType, got, c.excludesOwner)
		}
	}
}

func TestUnknownEventTypeExcludesNobodyAndIsNotBroadcast(t *testing.T) {
	if handles("SOMETHING_ELSE") {
		t.Fatal("an unrecognised dragon event type must not be broadcast")
	}
}
```

- [ ] **Step 2: Run to verify failure**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./kafka/consumer/dragon/... -v`
Expected: FAIL — package does not exist.

- [ ] **Step 3: Write the consumer**

`services/atlas-channel/atlas.com/channel/kafka/consumer/dragon/consumer.go`:

```go
package dragon

import (
	consumer2 "atlas-channel/kafka/consumer"
	dragonmsg "atlas-channel/kafka/message/dragon"
	"atlas-channel/listener"
	_map "atlas-channel/map"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	model2 "github.com/Chronicle20/atlas/libs/atlas-model/model"
	dragonpkt "github.com/Chronicle20/atlas/libs/atlas-packet/dragon/clientbound"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// excludesOwner reports whether the owner is excluded from this event's
// broadcast. MOVED excludes them: the owner's client already rendered the motion
// locally, so re-sending double-applies it — the same reasoning as the summon
// move relay. CREATED and DESTROYED go map-wide including the owner, because the
// owner has not rendered either locally.
func excludesOwner(eventType string) bool {
	return eventType == dragonmsg.EventDragonStatusMoved
}

// handles reports whether eventType is one this consumer broadcasts.
func handles(eventType string) bool {
	switch eventType {
	case dragonmsg.EventDragonStatusCreated,
		dragonmsg.EventDragonStatusMoved,
		dragonmsg.EventDragonStatusDestroyed:
		return true
	}
	return false
}

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model2.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model2.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("dragon_status_event")(dragonmsg.EnvEventTopicDragonStatus)(consumerGroupId),
				consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser),
				consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
				var handles []listener.HandlerHandle
				t, _ := topic.EnvProvider(l)(dragonmsg.EnvEventTopicDragonStatus)()

				id, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventCreated(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})

				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventMoved(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})

				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventDestroyed(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				return handles, nil
			}
		}
	}
}

func handleStatusEventCreated(sc server.Model, wp writer.Producer) message.Handler[dragonmsg.StatusEvent[dragonmsg.StatusEventCreatedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e dragonmsg.StatusEvent[dragonmsg.StatusEventCreatedBody]) {
		if e.Type != dragonmsg.EventDragonStatusCreated {
			return
		}
		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}
		// Map-wide INCLUDING the owner (FR-3.1): the owner has not rendered its
		// own dragon locally.
		err := _map.NewProcessor(l, ctx).ForSessionsInMap(sc.Field(e.MapId, e.Instance),
			session.Announce(l)(ctx)(wp)(dragonpkt.DragonSpawnWriter)(
				writer.DragonSpawnBody(e.OwnerCharacterId, e.Body.X, e.Body.Y, e.Body.Stance, e.Body.JobId)))
		if err != nil {
			l.WithError(err).Errorf("Unable to spawn dragon for character [%d] in map [%d].", e.OwnerCharacterId, e.MapId)
		}
	}
}

func handleStatusEventMoved(sc server.Model, wp writer.Producer) message.Handler[dragonmsg.StatusEvent[dragonmsg.StatusEventMovedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e dragonmsg.StatusEvent[dragonmsg.StatusEventMovedBody]) {
		if e.Type != dragonmsg.EventDragonStatusMoved {
			return
		}
		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}
		// OTHER sessions only (FR-4.3): the owner's client already rendered the
		// motion locally, so re-sending would double-apply it.
		err := _map.NewProcessor(l, ctx).ForOtherSessionsInMap(sc.Field(e.MapId, e.Instance), e.OwnerCharacterId,
			session.Announce(l)(ctx)(wp)(dragonpkt.DragonMoveWriter)(
				writer.DragonMoveBody(e.OwnerCharacterId, e.Body.RawMovement)))
		if err != nil {
			l.WithError(err).Errorf("Unable to move dragon for character [%d] in map [%d].", e.OwnerCharacterId, e.MapId)
		}
	}
}

func handleStatusEventDestroyed(sc server.Model, wp writer.Producer) message.Handler[dragonmsg.StatusEvent[dragonmsg.StatusEventDestroyedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e dragonmsg.StatusEvent[dragonmsg.StatusEventDestroyedBody]) {
		if e.Type != dragonmsg.EventDragonStatusDestroyed {
			return
		}
		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}
		// Map-wide (FR-3.3). Note the client discards REMOVE_DRAGON — it has no
		// handler arm for the opcode. The dragon actually disappears because the
		// owner's CUser is destroyed when they leave the field, which the
		// character-removal path already does. This broadcast is correct to send
		// and is not the mechanism.
		err := _map.NewProcessor(l, ctx).ForSessionsInMap(sc.Field(e.MapId, e.Instance),
			session.Announce(l)(ctx)(wp)(dragonpkt.DragonRemoveWriter)(
				writer.DragonRemoveBody(e.OwnerCharacterId)))
		if err != nil {
			l.WithError(err).Errorf("Unable to remove dragon for character [%d] in map [%d].", e.OwnerCharacterId, e.MapId)
		}
	}
}
```

- [ ] **Step 4: Run the fan-out tests**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./kafka/consumer/dragon/... -v`
Expected: PASS.

- [ ] **Step 5: Add the JOB_CHANGED create**

Add to `services/atlas-channel/atlas.com/channel/kafka/message/character/kafka.go`,
next to the other status bodies:

```go
// JobChangedStatusEventBody mirrors atlas-character's
// JobChangedStatusEventBody. It carries NO map id — which is why the dragon
// create for a job change has to run channel-side, where the session's field is
// available (see task-225 plan Task 9).
type JobChangedStatusEventBody struct {
	ChannelId channel.Id `json:"channelId"`
	JobId     job.Id     `json:"jobId"`
}
```

Add the `job` import if absent.

Add to `services/atlas-channel/atlas.com/channel/kafka/consumer/character/consumer.go`:

```go
// handleStatusEventJobChanged asks atlas-dragons to (re)create the dragon for a
// character whose job just changed, using the live session's field — the event
// carries no map id and GET /characters/{id} does not return one.
//
// It emits CREATE unconditionally and does NOT test the job itself: the
// dragon-bearing predicate lives in exactly one place (atlas-dragons'
// Processor.Create), and duplicating it channel-side would be a second copy to
// drift. A non-Evan job change is a cheap no-op there.
//
// The DESTROY direction is owned by atlas-dragons, which needs no field.
func handleStatusEventJobChanged(sc server.Model, wp writer.Producer) message.Handler[character2.StatusEvent[character2.JobChangedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, event character2.StatusEvent[character2.JobChangedStatusEventBody]) {
		if event.Type != character2.StatusEventTypeJobChanged {
			return
		}
		if !sc.Is(tenant.MustFromContext(ctx), event.WorldId, event.Body.ChannelId) {
			return
		}
		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(event.CharacterId, func(s session.Model) error {
			return dragoncmd.NewProcessor(l, ctx).Create(s.Field(), event.CharacterId)
		})
	}
}
```

with the import `dragoncmd "atlas-channel/dragon"`, and register it in that
file's `InitHandlers` following the existing pattern:

```go
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventJobChanged(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
```

- [ ] **Step 6: Wire the dragon consumer into `main.go`**

Next to `summonConsumer.InitConsumers(l)(cmf)(consumerGroupId)` (~line 224):

```go
	dragonConsumer.InitConsumers(l)(cmf)(consumerGroupId)
```

Next to `register(summonConsumer.InitHandlers(fl)(sc)(wp)(rh))` (~line 459):

```go
		if err := register(dragonConsumer.InitHandlers(fl)(sc)(wp)(rh)); err != nil {
			return err
		}
```

Match the exact error-handling shape of the surrounding `register(...)` calls —
read lines 450–470 and copy. Add the import
`dragonConsumer "atlas-channel/kafka/consumer/dragon"`.

- [ ] **Step 7: Build and test**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go vet ./... && go test -race ./kafka/... ./dragon/... ./socket/...`
Expected: clean.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-channel/atlas.com/channel
git commit -m "feat(task-225): atlas-channel dragon status broadcasts and job-change create"
```

---

### Task 13: Replay existing dragons to an entering session

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go` (`SpawnForSelf` ~`:236-240`; new operator next to `spawnSummonForSession` ~`:430-445`)

**Interfaces:**
- Consumes: `dragoncmd.Processor.ForEachInMap` (Task 10), `writer.DragonSpawnBody` (Task 11).
- Produces: `spawnDragonForSession(l)(ctx)(wp)(s session.Model) model.Operator[dragoncmd.Model]`.

- [ ] **Step 1: Add the per-session spawn operator**

Immediately after `spawnSummonForSession` in
`services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go`:

```go
// spawnDragonForSession sends a SPAWN_DRAGON for an existing dragon to the
// entering session s (FR-3.2) — a player entering a map must see an Evan's
// dragon immediately, not only after that Evan next moves.
//
// The owner's own dragon reaches them via the CREATED event broadcast, which is
// map-wide and includes the owner, so this operator does not need to special-case
// self: on a fresh entry the entering character's dragon is (re)created by
// atlas-dragons and broadcast, and re-sending here would be a duplicate. Callers
// therefore skip m.OwnerCharacterId() == s.CharacterId().
func spawnDragonForSession(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(s session.Model) model.Operator[dragoncmd.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(s session.Model) model.Operator[dragoncmd.Model] {
		return func(wp writer.Producer) func(s session.Model) model.Operator[dragoncmd.Model] {
			return func(s session.Model) model.Operator[dragoncmd.Model] {
				return func(m dragoncmd.Model) error {
					if m.OwnerCharacterId() == s.CharacterId() {
						return nil
					}
					return session.Announce(l)(ctx)(wp)(dragonpkt.DragonSpawnWriter)(
						writer.DragonSpawnBody(m.OwnerCharacterId(), m.X(), m.Y(), m.Stance(), m.JobId()))(s)
				}
			}
		}
	}
}
```

with imports `dragoncmd "atlas-channel/dragon"` and
`dragonpkt "github.com/Chronicle20/atlas/libs/atlas-packet/dragon/clientbound"`.

- [ ] **Step 2: Fan out from `SpawnForSelf`**

Immediately after the summon fan-out block in `SpawnForSelf`:

```go
		routine.Go(l, ctx, func(_ context.Context) {
			if err := dragoncmd.NewProcessor(l, ctx).ForEachInMap(f, spawnDragonForSession(l)(ctx)(wp)(s)); err != nil {
				l.WithError(err).Debugf("SpawnForSelf: unable to spawn dragons for character [%d].", s.CharacterId())
			}
		})
```

This is one field-scoped query per entry, not one per character — the entering
path must not become an N+1. A query failure is logged at Debug and skipped: it
degrades the entering player's view of other Evans and must not abort the rest of
`SpawnForSelf`. `routine.Go` (never a bare `go`) keeps
`tools/goroutine-guard.sh` clean.

- [ ] **Step 3: Build and vet**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go vet ./... && go test -race ./...`
Expected: clean.

- [ ] **Step 4: Run the goroutine guard**

Run: `tools/goroutine-guard.sh` from the repo root.
Expected: exit 0.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go
git commit -m "feat(task-225): replay existing dragons to a character entering the field"
```

---

### Task 14: Seed template routing

**Files:**
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_83_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_84_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_87_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_92_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_95_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_jms_185_1.json`
- Modify: `tools/template-movement-types-guard.sh` (`MOVE_HANDLERS`, line ~37)

**Interfaces:**
- Consumes: the writer/handler name constants from Task 1 and Task 11.
- Produces: three writer bindings and one handler binding per template, in six templates.

**The opcodes.** Every value below was cross-checked against
`docs/packets/registry/{gms_v83,gms_v84,gms_v87,gms_v92,gms_v95,jms_v185}.yaml`
and a collision scan against the existing arrays came back clean.

Writers (clientbound):

| Template | `DragonSpawn` | `DragonMove` | `DragonRemove` |
|---|---|---|---|
| `template_gms_83_1.json` | `0xB5` | `0xB6` | `0xB7` |
| `template_gms_84_1.json` | `0xB9` | `0xBA` | `0xBB` |
| `template_gms_87_1.json` | `0xC2` | `0xC3` | `0xC4` |
| `template_gms_92_1.json` | `0xD1` | `0xD2` | `0xD3` |
| `template_gms_95_1.json` | `0xCE` | `0xCF` | `0xD0` |
| `template_jms_185_1.json` | `0xBB` | `0xBC` | `0xBD` |

Handler (serverbound `DragonMoveHandle`):

| Template | opCode |
|---|---|
| `template_gms_83_1.json` | `0xB5` |
| `template_gms_84_1.json` | `0xBA` |
| `template_gms_87_1.json` | `0xC1` |
| `template_gms_92_1.json` | `0xD3` |
| `template_gms_95_1.json` | `0xD6` |
| `template_jms_185_1.json` | `0xB9` |

`template_gms_12_1.json`, `_48_`, `_61_`, `_72_`, `_79_` are **not touched** —
those columns have no dragon opcode assignment (`⬜` n-a in the matrix).

Note the opcode strings are written **unpadded** (`0xB5`, not `0x0B5`), matching
the surrounding entries. A padded duplicate of an existing binding is exactly
what `tools/template-duplicate-binding-guard.sh` bans.

- [ ] **Step 1: Add the three writer entries to each of the six templates**

Each entry has this shape — the `fname` values are the four client functions the
codecs were derived from:

```json
{ "opCode": "0xB5", "writer": "DragonSpawn",  "fname": "CDragon::OnCreated",     "services": ["channel"] },
{ "opCode": "0xB6", "writer": "DragonMove",   "fname": "CDragon::OnMove",        "services": ["channel"] },
{ "opCode": "0xB7", "writer": "DragonRemove", "fname": "CUser::OnDragonPacket",  "services": ["channel"] }
```

(the `opCode` values above are the v83 row; use each template's row from the
table). Insert each at its **sorted `opCode` position** in the `writers` array —
never appended next to a semantically related neighbour.
`tools/template-opcode-order-guard.sh` enforces strictly ascending order.

Every writer must carry an `fname`; a writer without one is rejected by the seed
loader.

- [ ] **Step 2: Add the handler entry to each of the six templates**

```json
{
  "opCode": "0xB5",
  "validator": "LoggedInValidator",
  "handler": "DragonMoveHandle",
  "fname": "CVecCtrlDragon::EndUpdateActive",
  "options": { "types": [ ... ] },
  "services": ["channel"]
}
```

Two non-negotiables:

1. **`validator` must be `"LoggedInValidator"`.** A handler with a missing
   validator is silently dropped at load time — no error, the feature just does
   not work.
2. **`options.types` must be byte-identical to the `SummonMoveHandle` entry in
   the SAME template.** Copy that template's array verbatim; do not copy across
   templates. The arrays differ per version — 23 entries in v83, 24 in v84, 25 in
   v87, 37 in v92 and v95, 33 in JMS185 — and
   `tools/template-movement-types-guard.sh` requires every move handler within one
   template to carry the identical array.

The codec treats the movement blob as opaque, so the table is currently unread —
exactly as it is for `SummonMoveHandle` today. It is carried anyway so the six
templates stay uniform and any future drift is caught mechanically rather than
discovered as a `"Code [N] not configured for use in movement"` spew.

Insert at the sorted `opCode` position in the `handlers` array.

- [ ] **Step 3: Register `DragonMoveHandle` with the movement guard**

In `tools/template-movement-types-guard.sh`, add to `MOVE_HANDLERS` (line ~37),
keeping the set alphabetical:

```python
MOVE_HANDLERS = {
    "CharacterMoveHandle",
    "DragonMoveHandle",
    "MonsterMovementHandle",
    "NPCActionHandle",
    "PetMovementHandle",
    "SummonMoveHandle",
}
```

- [ ] **Step 4: Run all three template guards**

Run from the repo root:

```bash
tools/template-opcode-order-guard.sh
tools/template-duplicate-binding-guard.sh
tools/template-movement-types-guard.sh
```

Expected: all exit 0. The movement guard must report `DragonMoveHandle` among
the handlers it checked — if its checked-count did not rise by six, the entries
are not being seen and the run is a false pass.

- [ ] **Step 5: Confirm the five untouched templates really are untouched**

Run: `git diff --stat services/atlas-configurations/seed-data/templates/`
Expected: exactly six files changed. `template_gms_12_1.json`, `_48_`, `_61_`,
`_72_`, `_79_` must not appear.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-configurations/seed-data/templates tools/template-movement-types-guard.sh
git commit -m "feat(task-225): route the four dragon opcodes in six seed templates"
```

---

### Task 15: `atlas-dragons` service registration

**Files:**
- Modify: `.github/config/services.json`
- Modify: `docker-bake.hcl`
- Create: `deploy/k8s/base/atlas-dragons.yaml`
- Modify: `deploy/k8s/base/kustomization.yaml`
- Modify: `deploy/k8s/base/env-configmap.yaml`
- Modify: `deploy/k8s/overlays/main/kustomization.yaml`
- Modify: `deploy/k8s/overlays/main/patches/atlas-env-env.yaml`
- Modify: `deploy/k8s/overlays/pr/kustomization.yaml`
- Regenerate: `deploy/k8s/overlays/pr/patches/consumer-group-env.yaml`, `dev/cluster-infra-coordination/atlas-pr-cleanup-env.example.yaml`
- Modify: `deploy/shared/routes.conf`
- Regenerate: `deploy/k8s/base/routes.conf.template.generated`

**Interfaces:**
- Consumes: the built `atlas-dragons` module (Task 2 onward).
- Produces: a deployable, pullable, routable service.

`atlas-dragons` has **no database**. Skip every DB row of
`docs/adding-a-new-service.md`: no `DB_NAME` patch (§3.1), no `ATLAS_DB_NAMES`
entry (§4.1), no `tools/db-bootstrap.sh` line (§6.2), no manual
`atlas-dragons-main` database, and no regeneration of
`deploy/k8s/overlays/pr/patches/db-name-suffix.yaml`.

- [ ] **Step 1: Build & CI registration**

Add to `.github/config/services.json` `services[]`, alphabetically (between
`atlas-drop-information` and `atlas-effective-stats` — verify the neighbours):

```json
    {
      "name": "atlas-dragons",
      "type": "go-service",
      "path": "services/atlas-dragons",
      "module_path": "services/atlas-dragons/atlas.com/dragons",
      "docker_image": "ghcr.io/chronicle20/atlas-dragons/atlas-dragons",
      "docker_context": "."
    },
```

Add `"atlas-dragons",` to the `go_services` list in `docker-bake.hcl` at its
sorted position. This list is **hand-synced** with services.json — adding to one
does not add to the other.

(`go.work` already gained its line in Task 2. The repo-root `Dockerfile` needs no
change: it is parameterized by `ARG SERVICE`, and this task introduces no new
shared lib.)

- [ ] **Step 2: Kubernetes base**

Create `deploy/k8s/base/atlas-dragons.yaml` — modelled on
`deploy/k8s/base/atlas-summons.yaml`, which is the other Redis-backed,
database-free service. No `namespace:` (overlays set it); the container `name:`
is the short name `dragons`, which the overlay patches match on:

```yaml
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: atlas-dragons
spec:
  replicas: 2
  selector:
    matchLabels:
      app: atlas-dragons
  template:
    metadata:
      labels:
        app: atlas-dragons
    spec:
      containers:
      - name: dragons
        image: ghcr.io/chronicle20/atlas-dragons/atlas-dragons:latest
        ports:
        - containerPort: 8080
        envFrom:
        - configMapRef:
            name: atlas-env
        env:
        - name: LOG_LEVEL
          value: "info"
---
apiVersion: v1
kind: Service
metadata:
  name: atlas-dragons
spec:
  selector:
    app: atlas-dragons
  ports:
  - protocol: TCP
    port: 8080
```

Add `atlas-dragons.yaml` to `resources:` in `deploy/k8s/base/kustomization.yaml`.

Add the two new topic vars to `deploy/k8s/base/env-configmap.yaml` at their
sorted positions, as identity values:

```yaml
  COMMAND_TOPIC_DRAGON: "COMMAND_TOPIC_DRAGON"
  EVENT_TOPIC_DRAGON_STATUS: "EVENT_TOPIC_DRAGON_STATUS"
```

Also confirm the service resolves `REDIS_URL` and `CHARACTERS` (the REST root
used by `requests.RootUrl("CHARACTERS")`) and that `DRAGONS` is added wherever
sibling `*_SERVICE_URL`-style roots are enumerated — `atlas-channel` calls
`requests.RootUrl("DRAGONS")`. Grep `env-configmap.yaml` for `SUMMONS` and add
the matching `DRAGONS` entry in the same shape; a hard-coded base-namespace URL
here breaks in overlays.

- [ ] **Step 3: Main overlay**

In `deploy/k8s/overlays/main/patches/atlas-env-env.yaml`, add a patch document
setting `ATLAS_ENV: "main"` for `atlas-dragons`, container `dragons` — copy the
`atlas-summons` document at line ~759 and change the two names.

In `deploy/k8s/overlays/main/kustomization.yaml`:
- add to `images:` — this is the trap that leaves a service on `:latest` forever,
  because the bump workflow only rewrites entries that already exist:
  ```yaml
    - name: ghcr.io/chronicle20/atlas-dragons/atlas-dragons
      newTag: main-<current fleet sha>
  ```
  Use the tag the rest of the fleet is on and confirm it exists
  (`docker manifest inspect`). Never leave `:latest`.
- add both topic literals to the `configMapGenerator`, at their sorted positions:
  ```
      - COMMAND_TOPIC_DRAGON=COMMAND_TOPIC_DRAGON-main
      - EVENT_TOPIC_DRAGON_STATUS=EVENT_TOPIC_DRAGON_STATUS-main
  ```
  The generator uses `behavior: replace`: any base key not re-listed here is
  **absent** on main, and the topic silently falls back to the unsuffixed name.

Do NOT add `KAFKA_CONSUMER_GROUP` on main — it is intentionally omitted there.

- [ ] **Step 4: PR overlay**

In `deploy/k8s/overlays/pr/kustomization.yaml`:
- add the `images:` entry in the same shape as Step 3.
- regenerate the topic literal block:
  `deploy/k8s/overlays/pr/scripts/gen-topic-config.sh`, and paste its output into
  the `atlas-env` generator. Do not hand-edit individual literals.

Regenerate the two generator-owned files:

```bash
deploy/k8s/overlays/pr/scripts/gen-consumer-group-patch.sh
deploy/k8s/overlays/pr/scripts/gen-cleanup-env.sh
```

The first derives `KAFKA_CONSUMER_GROUP` from the `consumerGroupId` literal in
`main.go` (`"Dragon Registry Service"`). The second rewrites
`dev/cluster-infra-coordination/atlas-pr-cleanup-env.example.yaml`;
`pr-validation.yml` regenerates it too and **hard-fails the PR** when the
committed copy is stale.

`gen-db-name-suffix.sh` is not run — `atlas-dragons` has no database.

- [ ] **Step 5: Ingress**

Add two location blocks to `deploy/shared/routes.conf` at their alphabetical
positions, mirroring the summons pair at lines ~476 and ~501:

```nginx
location ~ ^/api/worlds/[^/]+/channels/[^/]+/maps/[^/]+/instances/[^/]+/dragons(/.*)?$ {
  set $u "atlas-dragons:8080";
  proxy_pass http://$u$request_uri;
}
```

```nginx
location ~ ^/api/dragons(/.*)?$ {
  set $u "atlas-dragons:8080";
  proxy_pass http://$u$request_uri;
}
```

Then regenerate and commit both files:

```bash
tools/gen-routes.sh
```

- [ ] **Step 6: Run the registration guard**

Run: `tools/service-registration-guard.sh` from the repo root.
Expected: exit 0. If it reports a missing list, fix that list — do not silence it.

- [ ] **Step 7: Build the image**

Run: `docker buildx bake atlas-dragons` from the worktree root.
Expected: success. This is mandatory, not optional: `go build` against the
workspace will not catch a missing `COPY libs/...` in the shared Dockerfile —
only the bake will.

- [ ] **Step 8: Commit**

```bash
git add .github/config/services.json docker-bake.hcl deploy dev/cluster-infra-coordination
git commit -m "feat(task-225): register atlas-dragons across CI, k8s, overlays, and ingress"
```

**After the first CI publish (manual, out-of-repo, do not skip):** the GHCR
package `atlas-dragons/atlas-dragons` is created **private**. Every other atlas
package is public and the cluster pulls anonymously, so the pod sits in
`ImagePullBackOff` with a `401` while CI reports green. Flip it to Public
(GitHub → Packages → package settings → Change visibility), then delete the stuck
pod. Verify — `200` means public:

```bash
curl -s -o /dev/null -w '%{http_code}\n' \
  'https://ghcr.io/token?scope=repository%3Achronicle20%2Fatlas-dragons%2Fatlas-dragons%3Apull&service=ghcr.io'
```

---

### Task 16: Promote the 24 coverage-matrix cells

**Files:**
- Modify: `docs/packets/feature-families.yaml`
- Modify: `docs/packets/registry/{gms_v83,gms_v84,gms_v87,gms_v92,gms_v95,jms_v185}.yaml` (provenance, per cell verified)
- Create: evidence records under `docs/packets/audits/<version>/` (one per cell)
- Modify: `libs/atlas-packet/dragon/**/*_test.go` (one `packet-audit:verify` fixture per cell)
- Modify: `docs/packets/audits/status.json`, `docs/packets/audits/STATUS.md` (regenerated)

**Interfaces:**
- Consumes: the codecs from Task 1.
- Produces: 24 cells at `✅`.

**Do not restate the verification procedure here.** The canonical playbook is
[`docs/packets/audits/VERIFYING_A_PACKET.md`](../../packets/audits/VERIFYING_A_PACKET.md),
driven by the `/verify-packet` command and the `packet-verifier` agent (one agent
per packet × version, batched per IDB). Each cell's three artifacts — the byte
fixture carrying a `packet-audit:verify` marker, the pinned evidence record, and
the regenerated matrix — are committed together.

Two things about this particular set that the playbook cannot know:

- All four registry entries currently carry `provenance: csv-import` in every
  column except the four v84 rows: no real decompile has ever backed these
  layouts. Symbol names must be distrusted; the read order comes from the client.
- `REMOVE_DRAGON`'s cells verify a 4-byte body **and the routed-but-unhandled
  dispatch**. The evidence record must state the absence of the handler arm
  explicitly, or the cell will be re-opened later as an incomplete verification.
  The v84 registry note at `docs/packets/registry/gms_v84.yaml:1254` already
  records this finding for v84 and v95 and is the model to follow.

- [ ] **Step 1: Declare the `dragon` feature family**

Add to `docs/packets/feature-families.yaml` under `families:`, at its sorted
position:

```yaml
  dragon:
    - SPAWN_DRAGON
    - MOVE_DRAGON
    - REMOVE_DRAGON
```

The family rule is: if any member is `verified` on a version, no member may be
`n-a` on that version without positive absence evidence. All four ops are `⬜`
uniformly across v48/v61/v72/v79, so no inconsistency arises and **no
`feature-na-evidence.yaml` entry is needed**. Confirm that with
`packet-audit matrix --check` after the family is declared and before any cell is
promoted — if it complains at that point, the family entry is wrong, not the
evidence.

- [ ] **Step 2: Verify the six clientbound `SPAWN_DRAGON` cells**

Run `/verify-packet dragon/clientbound/SPAWN_DRAGON <version>` for each of
`gms_v83`, `gms_v84`, `gms_v87`, `gms_v92`, `gms_v95`, `jms_v185`.

Batch by IDB to avoid thrashing the IDA session. Resolve the session from
`idb_list` **by binary name** and pass it as the `database` parameter —
`select_instance(port)` is dead.

Expected per cell: the matrix cell promotes to `✅`. A cell that does not promote
is a failure report, not a prose claim.

- [ ] **Step 3: Verify the six clientbound `MOVE_DRAGON` cells**

Run `/verify-packet dragon/clientbound/MOVE_DRAGON <version>` for the same six.

- [ ] **Step 4: Verify the six clientbound `REMOVE_DRAGON` cells**

Run `/verify-packet dragon/clientbound/REMOVE_DRAGON <version>` for the same six.
Each evidence record must state the routed-but-unhandled dispatch (see above).

- [ ] **Step 5: Verify the six serverbound `MOVE_DRAGON` cells**

Run `/verify-packet dragon/serverbound/MOVE_DRAGON <version>` for the same six.
The serverbound verification derives the read order from the client's **send**
site, not a receive handler.

- [ ] **Step 6: Check the matrix**

Run: `packet-audit matrix --check`
Expected: exit 0, 24 dragon cells `✅`.

Then confirm nothing else moved:

```bash
git diff docs/packets/audits/status.json
```

Expected: only dragon rows changed. **No previously-`✅` cell of any other op may
have regressed** — if one did, stop and find out why before continuing.

- [ ] **Step 7: Run the full packet test suite**

Run: `cd libs/atlas-packet && go test -race ./...`
Expected: PASS, including every new `packet-audit:verify` fixture.

- [ ] **Step 8: Commit**

Each cell's artifacts were committed with its own verification. Commit the family
declaration and any residual matrix regeneration:

```bash
git add docs/packets
git commit -m "docs(task-225): declare the dragon feature family and promote 24 matrix cells"
```

---

### Task 17: Full verification sweep and code review

**Files:**
- No production files. This task produces evidence, and fixes whatever the
  evidence exposes.

**Interfaces:**
- Consumes: everything.
- Produces: a branch that is genuinely ready for a PR.

- [ ] **Step 1: Per-module build, vet, and race tests**

Run each, from the worktree root:

```bash
(cd libs/atlas-packet && go build ./... && go vet ./... && go test -race ./...)
(cd services/atlas-dragons/atlas.com/dragons && go build ./... && go vet ./... && go test -race ./...)
(cd services/atlas-channel/atlas.com/channel && go build ./... && go vet ./... && go test -race ./...)
```

Expected: all clean. If a test fails, fix it — do not report "verified" with a
known failure.

- [ ] **Step 2: Every guard**

```bash
tools/redis-key-guard.sh
tools/goroutine-guard.sh
tools/skill-job-id-guard.sh
tools/service-registration-guard.sh
tools/template-opcode-order-guard.sh
tools/template-duplicate-binding-guard.sh
tools/template-movement-types-guard.sh
tools/lint.sh --check
```

Expected: all exit 0.

`tools/lint.sh --check` false-fails without nvm on PATH and contends on a
golangci-lint lock across worktrees — if it fails, confirm the failure is real
before chasing it. Run `tools/lint.sh` (no flags) first to fix formatting in
place, then re-run `--check`.

- [ ] **Step 3: Image build**

```bash
docker buildx bake atlas-dragons
```

Expected: success. Mandatory because `atlas-dragons` has a new `go.mod`.

- [ ] **Step 4: Reconcile the live tenant socket configuration**

Updating a seed template does **not** update a provisioned tenant. An opcode
present only in the template is silently dropped at runtime and the feature does
nothing — with a completely clean server log. Reconcile every live tenant socket
config to the updated templates before any behavioural check.

Verify the reconciliation landed by confirming the four dragon bindings are
present in the live tenant configuration, not just in the template file.

- [ ] **Step 5: Live behavioural verification**

On a tenant at **v84 or above** — a v83 tenant cannot produce an Evan at all, so
v83 is packet-verified with no behaviour by design (D-3):

- [ ] An Evan at a dragon-bearing job sees its dragon on login.
- [ ] A second player entering the map sees the existing Evan's dragon
      immediately, before that Evan moves.
- [ ] Dragon movement by the owner is visible to the second player.
- [ ] The owner receives no echo of its own `MOVE_DRAGON`.
- [ ] Changing maps produces exactly one `SPAWN_DRAGON` in the new map and no
      orphan in the old. (The old map's visual cleanup is the character-removal
      path — `REMOVE_DRAGON` is emitted but the client discards it. Check
      "packet emitted", not "dragon removed by that packet".)
- [ ] Logging out emits `REMOVE_DRAGON` to the remaining players in the map.
- [ ] A non-Evan character in the same map has no dragon and generates no dragon
      traffic.
- [ ] An Evan beginner (job 2001) has no dragon.

Use Loki with a `service_name` selector — an `app=` selector silently returns
zero rows.

Record the actual observed values. "I think it worked" is not a result.

- [ ] **Step 6: Code review**

Run `superpowers:requesting-code-review`. It dispatches `plan-adherence-reviewer`
and `backend-guidelines-reviewer` (Go files changed; no atlas-ui TS changed, so
the frontend reviewer is not needed). Findings go to
`docs/tasks/task-225-evan-dragon/audit.md`.

Pin the review subagents to Sonnet — review workflows use the cheaper model.

Ensure the reviewers operate inside this worktree, and confirm the tree is clean
after they run.

- [ ] **Step 7: Address the findings and re-verify**

Fix what the review surfaces on this branch — do not defer it to a follow-up
task. Re-run Steps 1–3 after any change.

- [ ] **Step 8: Commit**

```bash
git add docs/tasks/task-225-evan-dragon/audit.md
git commit -m "docs(task-225): code review findings"
```

---

## Self-review notes

Deviations from `design.md`, each deliberate and justified in place:

1. **REST list endpoint shape** — design §5.5 sketched
   `GET /dragons?filter[worldId]=…`; the plan uses the established project shape
   `GET /worlds/{w}/channels/{c}/maps/{m}/instances/{i}/dragons`
   (Task 8), matching `atlas-summons` and dropping straight into
   `requests.DrainProvider`. Same capability, existing convention. Both nginx
   location blocks are registered in Task 15.
2. **`JOB_CHANGED` ownership is split** — design §5.1 put all five lifecycle
   events in `atlas-dragons`. `JOB_CHANGED` carries no map id and
   `GET /characters/{id}` does not return one, so the create half must run
   channel-side where the session's field exists. The destroy half stays in
   `atlas-dragons` (it needs no field). Reasoned in Task 9; the job predicate
   still exists in exactly one place.
3. **`AndEmit` variants** — design §5.4 listed `Create` "+ AndEmit variant". The
   plan uses the `emitter` injection the real `atlas-summons` processor uses
   (`summon/processor.go:44-47`), which serves the same testability purpose
   without a parallel method set.
4. **Opcode strings are unpadded** (`0xB5`, not design §7.3's `0x0B5`), matching
   the surrounding template entries and avoiding the padded-duplicate class
   `tools/template-duplicate-binding-guard.sh` bans.

PRD requirement coverage: FR-1 → Tasks 7, 9, 12; FR-2 → Tasks 3, 4; FR-3 →
Tasks 12, 13; FR-4 → Tasks 11, 7; FR-5 → Task 1; FR-6 → Task 14; FR-7 → Task 16;
§7 service registration → Task 15; §8 non-functionals → enforced by the guards in
Task 17 and the Global Constraints above.

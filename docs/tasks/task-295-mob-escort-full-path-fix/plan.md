# MobEscortFullPath Wire Model Correction — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Correct the `MOB_ESCORT_FULL_PATH` clientbound codec to the IDA-derived wire layout (drop the phantom `mode`, add `oldDestX`/`oldDestY`, rename the mis-derived tail fields), then promote the `gms_v92` coverage cell to ✅.

**Architecture:** Four sequential tasks. Task 1 rewrites the codec and its byte fixture in `libs/atlas-packet` (test-first). Task 2 updates the single `atlas-channel` consumer to the new signature — the repo does not compile between Task 1 and Task 2, so they run back to back. Task 3 adds the missing `gms_v92` seed-template writer row. Task 4 re-pins the three evidence records with the tool and regenerates the coverage matrix, then runs the packet CI gate set.

**Tech Stack:** Go 1.27, `libs/atlas-socket` response/request codecs, `tools/packet-audit` (evidence/matrix tooling), JSON seed templates in `atlas-configurations`.

**Spec:** [`design.md`](design.md) (PRD: [`prd.md`](prd.md))

## Global Constraints

- **No version gate.** The layout is byte-identical across `gms_v92`, `gms_v95`, `jms_v185`. Adding a `MajorAtLeast` gate is a defect (design §2.2, FR-10).
- **Field names come from the client's own members** (design §3): `oldDestX`/`oldDestY` ← `CVecCtrlMob::m_Old_Dest.dp.x/.y`; `currentDestIndex` ← `m_nCurrentDestIndex`; `attr` ← `EscortDest::nAttr`; waypoint `stopDuration` ← `EscortDest::nStopDuration`; `hasStopDuration`/`stopDuration`/`stopIndefinitely` ← `m_nStopDuration` / `m_bEscortStop`. No placeholder names on exported accessors.
- **`mode` is deleted, not renamed** — it never corresponded to a wire field (design §2.4).
- **Waypoint count is derived** from `len(waypoints)` on encode and is never stored on the struct (FR-8).
- **Struct stays immutable**: unexported fields, value-receiver accessors, `Decode` on the pointer receiver only.
- **No emitter is added.** `MobEscortFullPathBody` remains an intentionally unwired seam (PRD §2 non-goal).
- **No shim / no deprecated overload** for the old `MobEscortFullPathBody` signature.
- **Exactly one matrix cell may move**: `MOB_ESCORT_FULL_PATH` × `gms_v92`, ❌ → ✅, plus that column's summary totals. Any other cell movement is a blocker to investigate, not to absorb (FR-19).
- IDA addresses, used verbatim in markers, doc comments and evidence pins: `gms_v92` `0x6374c0`, `gms_v95` `0x643d90`, `jms_v185` `0x6efa01`; function name `CMob::OnEscortFullPath` in all three.
- Evidence records are written by `go run ./tools/packet-audit evidence pin` only — never hand-edited.
- The `gms_v92` registry row already exists (`docs/packets/registry/gms_v92.yaml:1581-1585`, opcode 296 = `0x128`). FR-16 needs no edit; do not add a duplicate row.

---

### Task 1: Correct the codec and its byte fixture

Rewrite `MobEscortFullPath` / `MobEscortWaypoint` to the derived layout and re-derive the golden fixture. Test-first: the new fixture is written against the corrected constructor signature, so it fails to compile until the codec is rewritten — that compile failure IS the red state for this task.

Module root for all commands: `libs/atlas-packet`.

### Files

- `libs/atlas-packet/monster/clientbound/mob_escort_full_path_test.go` — rewrite `TestMobEscortFullPath`: new golden bytes, new comments, third `packet-audit:verify` marker
- `libs/atlas-packet/monster/clientbound/mob_escort_full_path.go` — rewrite both structs, constructors, accessors, `String`, `Encode`, `Decode`, and both doc comments
- `libs/atlas-packet/test/context.go` — read-only; `pt.Variants` (line 18) and `pt.CreateContext` (line 43)
- `libs/atlas-packet/test/roundtrip.go` — read-only; `pt.RoundTrip` (line 23)

Patterns to copy: `libs/atlas-packet/monster/clientbound/catch_monster_test.go:10-14` (multi-version `packet-audit:verify` marker block above the test func), `libs/atlas-packet/monster/clientbound/catch_monster_test.go:42-48` (the `pt.Variants` round-trip loop).

**Interfaces:**

- Consumes: nothing from earlier tasks.
- Produces, for Task 2 and Task 4:

```go
func NewMobEscortWaypoint(x int32, y int32, attr int32, stopDuration int32) MobEscortWaypoint
func (w MobEscortWaypoint) X() int32
func (w MobEscortWaypoint) Y() int32
func (w MobEscortWaypoint) Attr() int32
func (w MobEscortWaypoint) StopDuration() int32

func NewMobEscortFullPath(
    oldDestX int32,
    oldDestY int32,
    waypoints []MobEscortWaypoint,
    currentDestIndex int32,
    hasStopDuration bool,
    stopDuration int32,
    stopIndefinitely bool,
) MobEscortFullPath
func (m MobEscortFullPath) OldDestX() int32
func (m MobEscortFullPath) OldDestY() int32
func (m MobEscortFullPath) Waypoints() []MobEscortWaypoint
func (m MobEscortFullPath) CurrentDestIndex() int32
func (m MobEscortFullPath) HasStopDuration() bool
func (m MobEscortFullPath) StopDuration() int32
func (m MobEscortFullPath) StopIndefinitely() bool
```

`const MobEscortFullPathWriter = "MobEscortFullPath"` and `func (m MobEscortFullPath) Operation() string` are unchanged.

- [ ] **Step 1: Rewrite the failing test**

Replace the whole body of `libs/atlas-packet/monster/clientbound/mob_escort_full_path_test.go` (keep the `package clientbound` clause and the `bytes` / `testing` / `pt` imports). Three markers, one golden, the existing `pt.Variants` loop preserved verbatim.

Marker block (exact, replaces the two current markers at lines 13-14, and the stale v87/v95 prose comment at lines 10-12):

```go
// MOB_ESCORT_FULL_PATH is routed at gms_v92 (0x128), gms_v95 (0x130) and
// jms_v185 (0x110). Absent in v83/v84/v87 (no escort family). The wire layout is
// byte-identical across all three routed versions — no version gate.
// packet-audit:verify packet=monster/clientbound/MonsterMobEscortFullPath version=gms_v92 ida=0x6374c0
// packet-audit:verify packet=monster/clientbound/MonsterMobEscortFullPath version=gms_v95 ida=0x643d90
// packet-audit:verify packet=monster/clientbound/MonsterMobEscortFullPath version=jms_v185 ida=0x6efa01
func TestMobEscortFullPath(t *testing.T) {
```

Golden input — two waypoints, `attr=1` and `attr=2`, so the conditional per-waypoint `stopDuration` is exercised (the current golden never encodes it):

```go
	// Two waypoints: wp0 attr=1 (no per-waypoint stopDuration), wp1 attr=2 (one
	// follows). Escort stop-duration present; no indefinite stop.
	input := NewMobEscortFullPath(
		0x0000000A, // oldDestX
		0x00000014, // oldDestY
		[]MobEscortWaypoint{
			NewMobEscortWaypoint(0x00000064, 0x000000C8, 1, 0),
			NewMobEscortWaypoint(0x000000FA, 0x0000012C, 2, 0x000002BC),
		},
		0x00000001, // currentDestIndex
		true,       // hasStopDuration
		0x00000320, // stopDuration
		false,      // stopIndefinitely
	)

	// Golden bytes. CMob::OnEscortFullPath — v92 @0x6374c0, v95 @0x643d90,
	// jms @0x6efa01; identical read order in all three:
	//   Decode4 -> count (ZArray<CVecCtrlMob::EscortDest>::_Alloc bound)
	//   Decode4 -> m_Old_Dest.dp.x
	//   Decode4 -> m_Old_Dest.dp.y
	//   per waypoint: Decode4 x, Decode4 y, Decode4 nAttr
	//                 (nAttr==2 -> +Decode4 nStopDuration)
	//   Decode4 -> m_nCurrentDestIndex
	//   Decode1 -> hasStopDuration; if set Decode4 -> m_nStopDuration
	//   Decode1 -> stopIndefinitely (m_bEscortStop, no auto-resume)
	got := input.Encode(nil, pt.CreateContext("GMS", 95, 1))(nil)
	want := []byte{
		0x02, 0x00, 0x00, 0x00, // count = 2
		0x0A, 0x00, 0x00, 0x00, // oldDestX = 10
		0x14, 0x00, 0x00, 0x00, // oldDestY = 20
		0x64, 0x00, 0x00, 0x00, // wp0.x = 100
		0xC8, 0x00, 0x00, 0x00, // wp0.y = 200
		0x01, 0x00, 0x00, 0x00, // wp0.attr = 1
		0xFA, 0x00, 0x00, 0x00, // wp1.x = 250
		0x2C, 0x01, 0x00, 0x00, // wp1.y = 300
		0x02, 0x00, 0x00, 0x00, // wp1.attr = 2
		0xBC, 0x02, 0x00, 0x00, // wp1.stopDuration = 700 (present: attr == 2)
		0x01, 0x00, 0x00, 0x00, // currentDestIndex = 1
		0x01,                   // hasStopDuration = true
		0x20, 0x03, 0x00, 0x00, // stopDuration = 800
		0x00, // stopIndefinitely = false
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("MobEscortFullPath layout mismatch\n got % x\nwant % x", got, want)
	}
```

Then keep the existing round-trip block exactly as it stands today (lines 55-60), unchanged:

```go
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			pt.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
```

Golden length check: 11 × `int32` (44 bytes) + `0x01` + `int32` (4) + `0x00` = **50 bytes**.

- [ ] **Step 2: Run the test to verify it fails**

Run from `libs/atlas-packet`: `go test ./monster/clientbound/ -run TestMobEscortFullPath`

Expected: FAIL — compile error, `NewMobEscortFullPath` called with 7 arguments (has 6) and `NewMobEscortWaypoint` field-name mismatch. That is the red state.

- [ ] **Step 3: Rewrite the waypoint type**

In `mob_escort_full_path.go`, replace lines 15-37 (the `MobEscortWaypoint` doc comment, struct, constructor and accessors):

```go
// MobEscortWaypoint is one entry of the MOB_ESCORT_FULL_PATH waypoint array —
// a CVecCtrlMob::EscortDest (20 bytes) in the client.
//
// Byte layout (per loop iteration in CMob::OnEscortFullPath):
//   - x            : int32 — Decode4; EscortDest::dp.x
//   - y            : int32 — Decode4; EscortDest::dp.y
//   - attr         : int32 — Decode4; EscortDest::nAttr. attr==1 zeroes the entry's
//     ZMass; otherwise the client fills it from
//     CWvsPhysicalSpace2D::GetFootholdUnderneath(x, y).
//   - stopDuration : int32 — Decode4; EscortDest::nStopDuration. Present ONLY when
//     attr == 2.
type MobEscortWaypoint struct {
	x            int32
	y            int32
	attr         int32
	stopDuration int32
}

func NewMobEscortWaypoint(x int32, y int32, attr int32, stopDuration int32) MobEscortWaypoint {
	return MobEscortWaypoint{x: x, y: y, attr: attr, stopDuration: stopDuration}
}

func (w MobEscortWaypoint) X() int32            { return w.x }
func (w MobEscortWaypoint) Y() int32            { return w.y }
func (w MobEscortWaypoint) Attr() int32         { return w.attr }
func (w MobEscortWaypoint) StopDuration() int32 { return w.stopDuration }
```

- [ ] **Step 4: Rewrite the packet doc comment, struct, constructor and accessors**

Replace lines 39-82 (doc comment through `String`). Keep the `// packet-audit:fname CMob::OnEscortFullPath` line — `packet-audit fname-doc --check` requires it.

```go
// MobEscortFullPath is the clientbound MOB_ESCORT_FULL_PATH packet
// (CMob::OnEscortFullPath): the server hands the client the full waypoint path for
// an escort mob. The handler writes into the mob's CVecCtrlMob (1960 bytes).
//
// Byte layout (IDA-derived; identical in all three routed versions):
//   - count            : int32 — Decode4; waypoint count, the
//     ZArray<CVecCtrlMob::EscortDest>::_Alloc bound. Derived from len(waypoints);
//     not a struct field.
//   - oldDestX         : int32 — Decode4; CVecCtrlMob::m_Old_Dest.dp.x
//   - oldDestY         : int32 — Decode4; CVecCtrlMob::m_Old_Dest.dp.y
//     (m_Old_Dest.nAttr is zeroed and m_Old_Dest.ZMass comes from
//     CMob::GetZMass(); neither is on the wire.)
//   - waypoints        : count × {x, y, attr int32, [attr==2: stopDuration int32]}
//   - currentDestIndex : int32 — Decode4; CVecCtrlMob::m_nCurrentDestIndex, the
//     index into the waypoint array the escort tick reads.
//   - hasStopDuration  : bool  — Decode1                       } stopDuration
//   - stopDuration     : int32 — Decode4, only when hasStopDuration; the client
//     sets m_nStopDuration = value + get_update_time() and m_bEscortStop = 1 — a
//     timed stop that auto-resumes.
//   - stopIndefinitely : bool  — Decode1; sets m_nStopDuration = 0 and
//     m_bEscortStop = 1 — the mob never auto-resumes.
//
// IDA basis: CMob::OnEscortFullPath — v92 @0x6374c0, v95 @0x643d90,
// jms @0x6efa01. A 2-waypoint path with no attr==2 entry is 9×Decode4
// (count + oldDestX + oldDestY + 2×(x,y,attr) + currentDestIndex), then
// Decode1 [+Decode4] + Decode1. v92 + v95 + jms only — the escort family is
// absent in v83/v84/v87.
//
// packet-audit:fname CMob::OnEscortFullPath
type MobEscortFullPath struct {
	oldDestX         int32
	oldDestY         int32
	waypoints        []MobEscortWaypoint
	currentDestIndex int32
	hasStopDuration  bool
	stopDuration     int32
	stopIndefinitely bool
}

func NewMobEscortFullPath(oldDestX int32, oldDestY int32, waypoints []MobEscortWaypoint, currentDestIndex int32, hasStopDuration bool, stopDuration int32, stopIndefinitely bool) MobEscortFullPath {
	return MobEscortFullPath{
		oldDestX:         oldDestX,
		oldDestY:         oldDestY,
		waypoints:        waypoints,
		currentDestIndex: currentDestIndex,
		hasStopDuration:  hasStopDuration,
		stopDuration:     stopDuration,
		stopIndefinitely: stopIndefinitely,
	}
}

func (m MobEscortFullPath) OldDestX() int32                 { return m.oldDestX }
func (m MobEscortFullPath) OldDestY() int32                 { return m.oldDestY }
func (m MobEscortFullPath) Waypoints() []MobEscortWaypoint  { return m.waypoints }
func (m MobEscortFullPath) CurrentDestIndex() int32         { return m.currentDestIndex }
func (m MobEscortFullPath) HasStopDuration() bool           { return m.hasStopDuration }
func (m MobEscortFullPath) StopDuration() int32             { return m.stopDuration }
func (m MobEscortFullPath) StopIndefinitely() bool          { return m.stopIndefinitely }
func (m MobEscortFullPath) Operation() string               { return MobEscortFullPathWriter }
func (m MobEscortFullPath) String() string {
	return fmt.Sprintf("oldDest [%d, %d], waypoints [%d], currentDestIndex [%d], hasStopDuration [%t], stopDuration [%d], stopIndefinitely [%t]",
		m.oldDestX, m.oldDestY, len(m.waypoints), m.currentDestIndex, m.hasStopDuration, m.stopDuration, m.stopIndefinitely)
}
```

Run `gofmt -w` on the file afterwards — the accessor block alignment above is illustrative, `gofmt` owns it.

- [ ] **Step 5: Rewrite Encode and Decode**

Replace the bodies (lines 84-129). The count is written from `len`, read into a local, and never stored.

```go
func (m MobEscortFullPath) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt32(int32(len(m.waypoints)))
		w.WriteInt32(m.oldDestX)
		w.WriteInt32(m.oldDestY)
		for _, wp := range m.waypoints {
			w.WriteInt32(wp.x)
			w.WriteInt32(wp.y)
			w.WriteInt32(wp.attr)
			if wp.attr == 2 {
				w.WriteInt32(wp.stopDuration)
			}
		}
		w.WriteInt32(m.currentDestIndex)
		w.WriteBool(m.hasStopDuration)
		if m.hasStopDuration {
			w.WriteInt32(m.stopDuration)
		}
		w.WriteBool(m.stopIndefinitely)
		return w.Bytes()
	}
}

func (m *MobEscortFullPath) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		count := r.ReadInt32()
		m.oldDestX = r.ReadInt32()
		m.oldDestY = r.ReadInt32()
		m.waypoints = make([]MobEscortWaypoint, 0, count)
		for i := int32(0); i < count; i++ {
			var wp MobEscortWaypoint
			wp.x = r.ReadInt32()
			wp.y = r.ReadInt32()
			wp.attr = r.ReadInt32()
			if wp.attr == 2 {
				wp.stopDuration = r.ReadInt32()
			}
			m.waypoints = append(m.waypoints, wp)
		}
		m.currentDestIndex = r.ReadInt32()
		m.hasStopDuration = r.ReadBool()
		if m.hasStopDuration {
			m.stopDuration = r.ReadInt32()
		}
		m.stopIndefinitely = r.ReadBool()
	}
}
```

- [ ] **Step 6: Run the test to verify it passes**

Run from `libs/atlas-packet`:

```bash
gofmt -l ./monster/clientbound/
go test ./monster/clientbound/ -run TestMobEscortFullPath -v
```

Expected: `gofmt -l` prints nothing; the test PASSes including every `pt.Variants` subtest.

- [ ] **Step 7: Run the package suite**

Run from `libs/atlas-packet`: `go test ./...`

Expected: PASS. No other package references `MobEscortFullPath`.

- [ ] **Step 8: Commit**

```bash
git add libs/atlas-packet/monster/clientbound/mob_escort_full_path.go libs/atlas-packet/monster/clientbound/mob_escort_full_path_test.go
git commit -m "fix(packet): correct MobEscortFullPath wire model to the IDA-derived layout"
```

Note: `services/atlas-channel` does not compile after this commit. Task 2 restores it; do not run a repo-wide build between the two.

---

### Task 2: Update the atlas-channel writer to the corrected signature

`MobEscortFullPathBody` is the only non-test consumer of the constructor and has no callers of its own. Update the parameter list to match Task 1 and refresh the doc comment. No shim, no deprecated overload (design §8).

Module root for all commands: `services/atlas-channel/atlas.com/channel`.

### Files

- `services/atlas-channel/atlas.com/channel/socket/writer/mob_escort_full_path.go` — the whole file (21 lines); signature + doc comment
- `libs/atlas-packet/monster/clientbound/mob_escort_full_path.go` — read-only; the constructor Task 1 produced

Patterns to copy: the file's own existing shape (`packet.Encode` closure returning `monsterpkt.New…(…).Encode(l, ctx)(options)`) — only the parameter list and doc comment change.

**Interfaces:**

- Consumes: `monsterpkt.NewMobEscortFullPath` and `monsterpkt.MobEscortWaypoint` from Task 1.
- Produces:

```go
func MobEscortFullPathBody(oldDestX int32, oldDestY int32, waypoints []monsterpkt.MobEscortWaypoint, currentDestIndex int32, hasStopDuration bool, stopDuration int32, stopIndefinitely bool) packet.Encode
```

- [ ] **Step 1: Confirm the symbol has no other callers**

```bash
grep -rn "MobEscortFullPathBody" --include=*.go .
```

Expected: exactly one hit — the definition at `services/atlas-channel/atlas.com/channel/socket/writer/mob_escort_full_path.go:15`. If any call site appears, stop and report it: the plan assumes none, and a caller means the argument order must be re-checked at that site too. (`main.go:758` references the `MobEscortFullPathWriter` *string constant*, which is unchanged — that is not a hit for this grep.)

- [ ] **Step 2: Rewrite the doc comment and signature**

Replace lines 12-20:

```go
// MobEscortFullPathBody encodes the clientbound MOB_ESCORT_FULL_PATH packet
// (CMob::OnEscortFullPath), which delivers an escort mob's full waypoint path:
// the previous destination point (oldDestX/oldDestY → CVecCtrlMob::m_Old_Dest.dp),
// the waypoint array (the count is derived from len(waypoints)), the current
// destination index, and the two escort-stop flags — a timed stop
// (hasStopDuration + stopDuration) and an indefinite one (stopIndefinitely).
// Routed at gms_v92 (0x128), gms_v95 (0x130) and jms_v185 (0x110). No emitter
// wires this writer yet; it is an intentional seam.
func MobEscortFullPathBody(oldDestX int32, oldDestY int32, waypoints []monsterpkt.MobEscortWaypoint, currentDestIndex int32, hasStopDuration bool, stopDuration int32, stopIndefinitely bool) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			return monsterpkt.NewMobEscortFullPath(oldDestX, oldDestY, waypoints, currentDestIndex, hasStopDuration, stopDuration, stopIndefinitely).Encode(l, ctx)(options)
		}
	}
}
```

- [ ] **Step 3: Build and vet the service**

Run from `services/atlas-channel/atlas.com/channel`:

```bash
gofmt -l ./socket/writer/
go build ./...
go vet ./...
```

Expected: `gofmt -l` prints nothing; build and vet clean.

- [ ] **Step 4: Run the service test suite**

Run from `services/atlas-channel/atlas.com/channel`: `go test ./...`

Expected: PASS (or unchanged pre-existing state — no test references this writer).

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/writer/mob_escort_full_path.go
git commit -m "fix(channel): update MobEscortFullPathBody to the corrected escort field list"
```

---

### Task 3: Route MOB_ESCORT_FULL_PATH in the gms_v92 seed template

The `gms_v92` registry already carries the op at opcode 296 (`0x128`); the seed template does not route it. Add the writers-array row so the cell can promote instead of turning into a template-wiring-gap conflict (design §7, rule at `tools/packet-audit/cmd/matrix_test.go:126-197`).

### Files

- `services/atlas-configurations/seed-data/templates/template_gms_92_1.json` — insert one object into `socket.writers`
- `services/atlas-configurations/seed-data/templates/template_gms_95_1.json` — read-only; the row to mirror (`0x130`, writers index 164)
- `services/atlas-configurations/seed-data/templates/template_jms_185_1.json` — read-only; the row to mirror (`0x110`, line 4453-4460)
- `docs/packets/registry/gms_v92.yaml` — read-only; the existing op row at line 1581-1585

Patterns to copy: `services/atlas-configurations/seed-data/templates/template_jms_185_1.json:4453-4460` (exact object shape and 8-space key indentation).

- [ ] **Step 1: Insert the writer row**

`socket.writers` in `template_gms_92_1.json` is sorted ascending by `opCode`. `0x128` goes between the `0x124` entry (`CatchMonsterWithItem`, ends at line 3806) and the `0x12F` entry (`SpawnNPC`, its opening brace at line 3807). Insert after line 3806's `},`:

```json
      {
        "opCode": "0x128",
        "writer": "MobEscortFullPath",
        "fname": "CMob::OnEscortFullPath",
        "services": [
          "channel"
        ]
      },
```

- [ ] **Step 2: Verify the JSON parses and the row landed in order**

```bash
python3 -c "
import json
ws = json.load(open('services/atlas-configurations/seed-data/templates/template_gms_92_1.json'))['socket']['writers']
codes = [int(w['opCode'], 16) for w in ws]
print('count', len(ws), 'sorted', codes == sorted(codes))
print([ (w['opCode'], w['writer']) for w in ws if 0x124 <= int(w['opCode'],16) <= 0x12F ])
"
```

Expected: `count 153 sorted True`, and the printed slice contains `('0x128', 'MobEscortFullPath')` between `0x124` and `0x12F`.

- [ ] **Step 3: Confirm the diff is exactly one added object**

```bash
git diff --stat services/atlas-configurations/seed-data/templates/template_gms_92_1.json
```

Expected: `1 file changed, 8 insertions(+)`, no deletions. A reformat of the whole file is a failure — revert and re-do the insert by hand.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-configurations/seed-data/templates/template_gms_92_1.json
git commit -m "feat(configurations): route MOB_ESCORT_FULL_PATH at 0x128 in the gms_v92 template"
```

---

### Task 4: Re-pin evidence and promote the gms_v92 matrix cell

Three evidence records written by the tool, then a matrix regeneration whose diff must show exactly one cell moving.

Module root: repo root (the `packet-audit` module is invoked as `go run ./tools/packet-audit` from there).

### Files

- `docs/packets/evidence/gms_v92/monster.clientbound.MonsterMobEscortFullPath.yaml` — new file, written by `evidence pin`
- `docs/packets/evidence/gms_v95/monster.clientbound.MonsterMobEscortFullPath.yaml` — re-pinned by the tool
- `docs/packets/evidence/jms_v185/monster.clientbound.MonsterMobEscortFullPath.yaml` — re-pinned by the tool
- `docs/packets/audits/STATUS.md` — regenerated by `packet-audit matrix`
- `docs/packets/audits/status.json` — regenerated by `packet-audit matrix`
- `tools/packet-audit/cmd/evidence.go` — read-only; the flag set (`--packet`, `--version`, `--ida`, `--category`, `--verifies`) at lines 23-36
- `docs/packets/ida-exports/gms_v92.json` — read-only; supplies the address and `decompile_sha256`

Do not hand-edit any of these six files.

- [ ] **Step 1: Pin all three evidence records**

`--verifies` must be passed on every invocation including the two re-pins: `cmd/evidence.go:62` rebuilds the record from flags, so omitting it silently drops the `verifies:` link.

```bash
for v in gms_v92 gms_v95 jms_v185; do
  go run ./tools/packet-audit evidence pin \
    --packet monster/clientbound/MonsterMobEscortFullPath \
    --version "$v" \
    --ida "CMob::OnEscortFullPath" \
    --category TIER1-FIXTURE \
    --verifies libs/atlas-packet/monster/clientbound/mob_escort_full_path_test.go#TestMobEscortFullPath
done
```

Expected: three `pinned docs/packets/evidence/<version>/monster.clientbound.MonsterMobEscortFullPath.yaml` lines, exit 0 each.

- [ ] **Step 2: Check the records and the expected no-op re-pins**

```bash
cat docs/packets/evidence/gms_v92/monster.clientbound.MonsterMobEscortFullPath.yaml
git diff --stat docs/packets/evidence/
```

Expected: the new `gms_v92` record carries `address: "0x6374c0"`, `category: TIER1-FIXTURE`, and the `verifies:` entry. The `gms_v95` and `jms_v185` records are **unchanged** (`git diff --stat` lists only the untracked-file-free delta, i.e. no modification to those two). That is correct, not a skipped step: `decompile_sha256` is computed from the checked-in export, which this task does not change, so the v95 hash stays `fc79aaa5…` and jms stays `fb9623c9…` (design §6).

- [ ] **Step 3: Regenerate the coverage matrix**

```bash
go run ./tools/packet-audit matrix
```

Expected: exit 0.

- [ ] **Step 4: Assert exactly one cell moved**

```bash
git diff docs/packets/audits/STATUS.md
```

Expected: the `MOB_ESCORT_FULL_PATH` row (currently `STATUS.md:457`) changes its `gms_v92` verdict from ❌ to ✅, plus the `gms_v92` summary-row totals. **Nothing else.** Any other row moving is a blocker — stop and report it rather than committing it (Global Constraints, FR-19).

```bash
python3 -c "
import json
row = next(r for r in json.load(open('docs/packets/audits/status.json'))['rows']
           if r.get('op') == 'MOB_ESCORT_FULL_PATH')
print(row['cells']['gms_v92'])
"
```

Expected: `{'state': 'verified', 'opcode': 296}` — the `incomplete` state and its `tier-1 without fixture; verdict 🔍` note are gone.

- [ ] **Step 5: Run the packet CI gate set**

These are the gates `.github/workflows/packet-matrix.yml` runs; `tools/verify.sh` does not cover them.

```bash
cd tools/packet-audit && go test ./... ; cd ../..
go run ./tools/packet-audit fname-doc --check
go run ./tools/packet-audit operations --check
go run ./tools/packet-audit dispatcher-lint
go run ./tools/packet-audit doc-freshness --check
go run ./tools/packet-audit gate-check --check
go run ./tools/packet-audit matrix --check
```

Expected: every command exits 0. `matrix --check` exiting non-zero after a fresh `matrix` run means either a conflict cell or an uncommitted regeneration — read its stderr, do not re-run blindly.

- [ ] **Step 6: Commit**

```bash
git add docs/packets/evidence/gms_v92/monster.clientbound.MonsterMobEscortFullPath.yaml \
        docs/packets/evidence/gms_v95/monster.clientbound.MonsterMobEscortFullPath.yaml \
        docs/packets/evidence/jms_v185/monster.clientbound.MonsterMobEscortFullPath.yaml \
        docs/packets/audits/STATUS.md docs/packets/audits/status.json
git commit -m "chore(packets): pin MobEscortFullPath evidence and promote the gms_v92 cell"
git status --short
```

Expected: `git status --short` is empty.

- [ ] **Step 7: Full verification gate**

```bash
tools/verify.sh
```

Expected: exit 0. Flagless — `--quick`/`--no-docker` do not count as the gate (CLAUDE.md, "Done means verified").

---

## Self-Review

**Spec coverage (design.md → task):** §2 derivation is already complete and is consumed as the layout in Task 1 Steps 3-5 and the doc comments. §3 naming → Task 1 Steps 3-4 + Global Constraints. §4 codec → Task 1. §5 fixture → Task 1 Step 1. §6 evidence → Task 4 Steps 1-2. §7 routing/promotion → Task 3 + Task 4 Steps 3-4. §8 consumer → Task 2. §9 work order → the task order (its steps 1-3 unit is Tasks 1-2, its 4-6 unit is Tasks 3-4). §10 risks → Task 4 Step 4 (matrix drift), Global Constraints (renames), Task 4 Step 2 (unchanged hash), Task 3 before Task 4 (ordering). §11 out-of-scope → Global Constraints (no emitter, no gate).

**PRD FR coverage:** FR-1..FR-5 discharged by design §2 (evidence already gathered; no re-derivation task). FR-6..FR-9 Task 1 Steps 3-5. FR-10 Global Constraints. FR-11, FR-12, FR-15 Task 1 Step 1. FR-13, FR-14 Task 4 Steps 1-2. FR-16 already satisfied by the existing registry row (Global Constraints; no task). FR-17 Task 3. FR-18 Task 4 Step 3. FR-19 Task 4 Step 4. FR-20, FR-21 Task 2.

**Type consistency:** `oldDestX`/`oldDestY`/`currentDestIndex`/`hasStopDuration`/`stopDuration`/`stopIndefinitely` and waypoint `attr`/`stopDuration` are spelled identically in Task 1's struct, Task 1's constructor, Task 1's fixture, and Task 2's writer. The `MobEscortFullPathWriter` constant and `Operation()` are unchanged, so `main.go:758` and the template `"writer": "MobEscortFullPath"` value keep matching.

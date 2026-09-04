# Map Back-Effects Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship the `SET_BACK_EFFECT` / `CLEAR_BACK_EFFECT` clientbound packets end to end — codecs, coverage-matrix promotion, `atlas-maps` state, `atlas-channel` broadcast + map-entry replay, saga actions, and a GM chat command.

**Architecture:** Two new clientbound codecs in `libs/atlas-packet/field/clientbound/`. `atlas-maps` owns per-field back-effect state in a process-local, tenant-keyed registry (`map[FieldKey][]BackEffectEntry` — N entries per field, one per `pageId`, no expiry) and serves it over REST. Commands arrive on `COMMAND_TOPIC_MAP`; status events go out on `EVENT_TOPIC_MAP_STATUS`; `atlas-channel` broadcasts the packet and replays active entries to an entering session. Triggers are a saga action pair and a GM chat command. Every layer is modelled file-for-file on the existing **jukebox** feature.

**Tech Stack:** Go 1.27, `libs/atlas-packet` (response.Writer / request.Reader), `libs/atlas-kafka`, `libs/atlas-rest` (JSON:API via api2go), `libs/atlas-tenant`, `libs/atlas-constants/field`, `tools/packet-audit`, testify-free stdlib `testing`.

**Spec:** `docs/tasks/task-281-map-back-effects/design.md` (PRD: `docs/tasks/task-281-map-back-effects/prd.md`)

## Global Constraints

- **Derived wire layout (design §1.1), 10 bytes, in this order:** `nEffect` byte → `nFieldID` int32 → `nPageID` byte → `tDuration` int32.
- **`CLEAR_BACK_EFFECT` has an empty body** (design §1.2). Opcode only.
- `nEffect` is a two-value enum: `0` = show (fade alpha to 255), `1` = hide (fade alpha to 0). Any other value makes the client handler return without acting.
- `tDuration` is a **fade length in ms**, not a lifetime. It never becomes an expiry.
- Writer constants: `SetBackEffectWriter = "SetBackEffect"`, `ClearBackEffectWriter = "ClearBackEffect"`.
- Packet ids (matrix / evidence / audit): `field/clientbound/FieldSetBackEffect`, `field/clientbound/FieldClearBackEffect`.
- Client fnames: `CMapLoadable::OnSetBackEffect`, `CMapLoadable::OnClearBackEffect`.
- Kafka constants: `CommandTypeSetBackEffect = "SET_BACK_EFFECT"`, `CommandTypeClearBackEffect = "CLEAR_BACK_EFFECT"`, `EventTopicMapStatusTypeBackEffectSet = "BACK_EFFECT_SET"`, `EventTopicMapStatusTypeBackEffectClear = "BACK_EFFECT_CLEAR"`.
- Saga actions: `SetBackEffect Action = "set_back_effect"`, `ClearBackEffect Action = "clear_back_effect"`.
- **No `MajorAtLeast` gate in the first cut** (design §2.3). Add one only if Task 1/2 finds a real per-version delta. `MajorAtLeast` is a method on `tenant.Model` (`libs/atlas-tenant/tenant.go:93`), used as the compound idiom `t.IsRegion("GMS") && t.MajorAtLeast(N)` — see `libs/atlas-packet/field/clientbound/set_field.go:46`.
- **Never invent an IDA address, opcode, or byte value.** Addresses come from Task 1/2's `structures/<version>.md`; opcodes come from `docs/packets/registry/<version>.yaml`.
- `SET_MAP_OBJECT_VISIBLE` is **out of scope** (design §2.4). Record it as a follow-up; do not implement it.
- No teardown/reaper for the registry (design §3.4). Accepted and documented.

### Per-version opcode table (from `docs/packets/audits/STATUS.md:210,231`)

| Op | v48 | v61 | v72 | v79 | v83 | v84 | v87 | v92 | v95 | jms185 |
|---|---|---|---|---|---|---|---|---|---|---|
| SET_BACK_EFFECT | ⬜ | ⬜ | 0x075 | 0x079 | 0x080 | 0x083 | 0x088 | 0x08F | 0x090 | 0x07E |
| CLEAR_BACK_EFFECT | ⬜ | ⬜ | ⬜ | ⬜ | 0x082 | 0x085 | 0x08A | 0x091 | 0x092 | 0x080 |

### Module roots (the `go build` / `go test` cwd per task)

| Module | Root |
|---|---|
| `libs/atlas-packet` | `libs/atlas-packet` |
| `libs/atlas-saga` | `libs/atlas-saga` |
| `atlas-maps` | `services/atlas-maps/atlas.com/maps` |
| `atlas-channel` | `services/atlas-channel/atlas.com/channel` |
| `atlas-saga-orchestrator` | `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator` |
| `atlas-messages` | `services/atlas-messages/atlas.com/messages` |
| `packet-audit` | repo root (`go run ./tools/packet-audit ...`) |

---

## Task 1: Derive the GMS wire layout on v79, v83, v87, v92

Design §1.1 proved the four-field layout identical on v72, v84 and v95. This task proves (or disproves) it on the four remaining GMS versions that carry the op. It writes **no Go code** — its output is the evidence every later task quotes.

### Files

- `docs/tasks/task-281-map-back-effects/structures/gms_v72.md` — new file; transcribe the design §1.1 v72 row (`OnSetBackEffect` @ `0x5f5b4f`, decode `sub_54C265` @ `0x54c265`) into the standard record
- `docs/tasks/task-281-map-back-effects/structures/gms_v79.md` — new file
- `docs/tasks/task-281-map-back-effects/structures/gms_v83.md` — new file
- `docs/tasks/task-281-map-back-effects/structures/gms_v84.md` — new file; transcribe design §1.1 (`0x659e3c`, `sub_597B59` @ `0x597b59`) plus `OnClearBackEffect` `0x65a241` thunk → `0x659d08` from `docs/packets/registry/gms_v84.yaml:905`
- `docs/tasks/task-281-map-back-effects/structures/gms_v87.md` — new file
- `docs/tasks/task-281-map-back-effects/structures/gms_v92.md` — new file
- `docs/packets/registry/gms_v79.yaml` — read-only; line 857 records the `SET_BACK_EFFECT` opcode/address lead to confirm (`gms_v84.yaml` lines 882/898 are the v84 leads)
- `docs/packets/IMPLEMENTING_A_PACKET.md` — read-only; §0–4 is the procedure this task follows

Patterns to copy: `docs/packets/registry/gms_v84.yaml:889` (the v84 note is the exact level of detail a structures record must reach: router address, case number, callee address, body shape, cross-version body match).

Module root: none — documentation only.

- [ ] **Step 1: Read the procedure**

Read `docs/packets/IMPLEMENTING_A_PACKET.md` §0–4. Step 0 ("is this already implemented / is it a shared-codec wrapper?") applies: confirm no existing codec in `libs/atlas-packet/field/clientbound/` already covers a 4-field `byte,int32,byte,int32` shape routed from `CMapLoadable::OnPacket`. Record the outcome in each structures file.

- [ ] **Step 2: Decompile `CMapLoadable::OnPacket` per version**

For each of gms_v79, gms_v83, gms_v87, gms_v92, using `ida-pro-mcp` against that version's IDB (or the checked-in export under `docs/packets/ida-exports/`):

1. Locate `CMapLoadable::OnPacket` and read the switch/if-chain.
2. Record which case number routes to `OnSetBackEffect` and which to `OnClearBackEffect`. Cross-check the case number against the opcode in the per-version opcode table above. **A mismatch is a finding, not a rounding error** — record it and stop for a controller ruling.
3. Decompile `OnSetBackEffect` and follow it into its decode callee.
4. Decompile `OnClearBackEffect` and confirm `iPacket` is untouched (v95 shape: a thunk to `ReloadBack`).

- [ ] **Step 3: Write one structures record per version**

Each file uses this exact skeleton. Fill every field from the decompile; leave nothing as a guess.

```markdown
# <version> — SET_BACK_EFFECT / CLEAR_BACK_EFFECT

IDB session: <session id>
Export: <path under docs/packets/ida-exports/, if used>

## Router

`CMapLoadable::OnPacket` @ <addr>
- case <n> -> `CMapLoadable::OnSetBackEffect` @ <addr>
- case <n> -> `CMapLoadable::OnClearBackEffect` @ <addr>   (or: ABSENT — no arm)

## SET_BACK_EFFECT read order

Decode callee: <symbol> @ <addr>

| # | Read | Width | Field |
|---|---|---|---|
| 1 | Decode1 | byte  | nEffect |
| 2 | Decode4 | int32 | nFieldID |
| 3 | Decode1 | byte  | nPageID |
| 4 | Decode4 | int32 | tDuration |

Branch shape: nEffect==0 -> RelMove(alpha, 255, ...); nEffect==1 -> RelMove(alpha, 0, ...); other -> return.

Verdict vs the gms_v95 reference (design §1.1): IDENTICAL | DIVERGES: <what>

## CLEAR_BACK_EFFECT

Handler @ <addr>. Packet reads: <none | list>.
Verdict vs the gms_v95 reference (design §1.2): IDENTICAL | DIVERGES: <what>
```

For `gms_v72.md` and `gms_v84.md`, transcribe the already-derived facts from design §1.1/§1.2 and the `gms_v84.yaml:889,905` registry notes rather than re-decompiling. Note the source explicitly (`derived in design §1.1`).

- [ ] **Step 4: Report divergence, if any**

If any version diverges from the four-field order or the empty clear, write the divergence into the structures file **and stop for a controller ruling** before Task 3 writes a codec. A divergence becomes a `MajorAtLeast` gate (design §2.3), and the gate's exact form is not this task's call.

- [ ] **Step 5: Commit**

```bash
git add docs/tasks/task-281-map-back-effects/structures/
git commit -m "docs(task-281): derive SET/CLEAR_BACK_EFFECT layout on gms v72/79/83/84/87/92"
```

---

## Task 2: Derive jms_v185, and prove the six ⬜ cells absent

Two jobs that share one procedure — reading a `CMapLoadable::OnPacket` switch — so they share a task.

### Files

- `docs/tasks/task-281-map-back-effects/structures/gms_v95.md` — new file; transcribe design §1.1/§1.2 (`OnSetBackEffect` @ `0x612850`, `Field::BackEffect::Decode` @ `0x565500`, `OnClearBackEffect` @ `0x61f230`, `ReloadBack` @ `0x61f0c0`, router @ `0x61fd80` cases 144/145/146)
- `docs/tasks/task-281-map-back-effects/structures/jms_v185.md` — new file
- `docs/tasks/task-281-map-back-effects/structures/gms_v48.md` — new file; absence proof for both ops
- `docs/tasks/task-281-map-back-effects/structures/gms_v61.md` — new file; absence proof for both ops
- `docs/tasks/task-281-map-back-effects/structures/absence-clear-v72-v79.md` — new file; absence proof for `CLEAR_BACK_EFFECT` on gms_v72 and gms_v79
- `docs/tasks/task-113-gms-legacy-versions/v72-packet-delta.md:179` — read-only; a *lead* about v72's clear arm, not evidence

Patterns to copy: `docs/packets/feature-na-evidence.yaml:9-20` (the `USE_TELEPORT_ROCK × gms_v48` entry is the standard for what counts as positive absence evidence: the opcode slot resolves to a *different* named op, plus a binary-wide search that found nothing).

Module root: none — documentation only.

- [ ] **Step 1: jms_v185 derivation**

Same procedure as Task 1 Step 2, against the jms_v185 IDB. Registry leads: `docs/packets/registry/jms_v185.yaml:623` (`SET_BACK_EFFECT` opcode 126) and `:633` (`CLEAR_BACK_EFFECT` opcode 128). Write `structures/jms_v185.md` with the Task 1 Step 3 skeleton.

JMS has diverged from GMS elsewhere. If the read order differs, that is a **finding**: record it and stop for a controller ruling before Task 3.

- [ ] **Step 2: Absence proofs for gms_v48 and gms_v61**

The checked-in exports record `CMapLoadable::OnSetBackEffect` as `unresolved: true` ("function not found in IDB"). That is a lead, not evidence of absence. For each of gms_v48 and gms_v61:

1. Locate `CMapLoadable::OnPacket` (or the equivalent field-loadable router) and enumerate **every** switch arm actually present.
2. For the opcode slot that `SET_BACK_EFFECT` / `CLEAR_BACK_EFFECT` would occupy, record which op *does* own it.
3. Run a binary-wide search for the layer-alpha tween shape (`ZMap<long, ZRef<ZList<IWzGr2DLayer>>>::GetAt` on the field's back-layer map, followed by `RelMove` on alpha) and for `ReloadBack`. Record the hit count.

Write both proofs into `structures/gms_v48.md` and `structures/gms_v61.md` using this skeleton:

```markdown
# <version> — SET_BACK_EFFECT / CLEAR_BACK_EFFECT: ABSENT

IDB session: <session id>

## Router arms present

`CMapLoadable::OnPacket` @ <addr> — arms: <case n -> fname, ...>

## Positive absence evidence

- Opcode slot <n> on this version is owned by <op> (`<fname>` @ <addr>), not a back-effect handler.
- Binary-wide search for the back-layer alpha tween (ZMap GetAt on m_mlLayerBack + RelMove on alpha): <n> hits.
- Binary-wide search for `ReloadBack` / the whole-field back reload: <n> hits.

Conclusion: <op> does not exist on <version>. Cell is VERSION-ABSENT.
```

- [ ] **Step 3: Absence proof for `CLEAR_BACK_EFFECT` on gms_v72 and gms_v79**

Both versions resolve `OnSetBackEffect` (registry `gms_v72.yaml:836`, `gms_v79.yaml:857`) but carry no clear entry. Read each router's arms directly and record, in `structures/absence-clear-v72-v79.md`, using the Step 2 skeleton once per version: the arm immediately after `SET_BACK_EFFECT`'s (v72 case 117, v79 case 121) is `SET_MAP_OBJECT_VISIBLE` (v72 case 118, v79 case 122) — record what the *next* arm after that is, and whether any arm anywhere routes to a `ReloadBack`-shaped handler.

- [ ] **Step 4: Commit**

```bash
git add docs/tasks/task-281-map-back-effects/structures/
git commit -m "docs(task-281): jms_v185 layout + VERSION-ABSENT proofs for the six blank cells"
```

---

## Task 3: The two clientbound codecs

### Files

- `libs/atlas-packet/field/clientbound/set_back_effect.go` — new file
- `libs/atlas-packet/field/clientbound/set_back_effect_test.go` — new file
- `libs/atlas-packet/field/clientbound/clear_back_effect.go` — new file
- `libs/atlas-packet/field/clientbound/clear_back_effect_test.go` — new file
- `libs/atlas-packet/field/clientbound/field_obstacle_on_off.go` — read-only; the multi-field shape to copy
- `libs/atlas-packet/field/clientbound/field_obstacle_all_reset.go` — read-only; the empty-body shape to copy
- `docs/tasks/task-281-map-back-effects/structures/` — read-only; the source of every `ida=` address in the verify markers

Patterns to copy: `libs/atlas-packet/field/clientbound/field_obstacle_on_off.go:1-49` (struct + `New` + accessors + `Operation`/`String`/`Encode`/`Decode` + `packet-audit:fname`), `libs/atlas-packet/field/clientbound/field_obstacle_all_reset.go:1-39` (empty body), `libs/atlas-packet/field/clientbound/field_obstacle_on_off_test.go:1-34` (golden + `test.Variants` round trip).

Module root: `libs/atlas-packet`.

Test helpers available (`libs/atlas-packet/test/`): `test.CreateContext(region string, major, minor uint16) context.Context`, `test.Encode(t, ctx, encFn, options)`, `test.RoundTrip(t, ctx, encFn, decFn, options)`, `test.Variants` (the per-version variant table the round-trip loop ranges over).

- [ ] **Step 1: Write the failing tests**

`set_back_effect_test.go` — two functions.

`TestSetBackEffectGolden`: builds `NewSetBackEffect(BackEffectShow, 100000000, 1, 1000)` under `test.CreateContext("GMS", 83, 1)` and asserts the exact 10 bytes.

| field | value | wire bytes (LE) |
|---|---|---|
| `effect` | `BackEffectShow` (0) | `0x00` |
| `fieldId` | `100000000` (`0x05F5E100`) | `0x00 0xE1 0xF5 0x05` |
| `pageId` | `1` | `0x01` |
| `duration` | `1000` (`0x000003E8`) | `0xE8 0x03 0x00 0x00` |

Full expected slice: `[]byte{0x00, 0x00, 0xE1, 0xF5, 0x05, 0x01, 0xE8, 0x03, 0x00, 0x00}`.

`TestSetBackEffectRoundTrip`: ranges `test.Variants`, `t.Run(v.Name, ...)`, `test.RoundTrip` with the same input. Copy the loop body verbatim from `field_obstacle_on_off_test.go:26-33`.

`clear_back_effect_test.go` — `TestClearBackEffectGolden` asserts `len(actual) == 0` for `NewClearBackEffect()`; `TestClearBackEffectRoundTrip` is the same `test.Variants` loop. Copy both from `field_obstacle_all_reset_test.go`.

**`packet-audit:verify` markers.** Each test file carries one `// packet-audit:verify packet=<id> version=<key> ida=<addr>` line per cell it covers, immediately above the golden test function — the block form at `field_obstacle_on_off_test.go:10-14`.

- `set_back_effect_test.go`: 8 markers, `packet=field/clientbound/FieldSetBackEffect`, versions `gms_v72 gms_v79 gms_v83 gms_v84 gms_v87 gms_v92 gms_v95 jms_v185`.
- `clear_back_effect_test.go`: 6 markers, `packet=field/clientbound/FieldClearBackEffect`, versions `gms_v83 gms_v84 gms_v87 gms_v92 gms_v95 jms_v185`.

The `ida=` value for each marker is the `OnSetBackEffect` / `OnClearBackEffect` address recorded in `docs/tasks/task-281-map-back-effects/structures/<version>.md` by Tasks 1–2. Three are already known from design §1.1/§1.2 and may be written directly: `gms_v72 ida=0x5f5b4f`, `gms_v84 ida=0x659e3c`, `gms_v95 ida=0x612850` (set); `gms_v95 ida=0x61f230` (clear). **Do not invent the others** — read them out of the structures files.

- [ ] **Step 2: Run the tests to verify they fail**

Run from `libs/atlas-packet`:

```bash
go test ./field/clientbound/ -run 'BackEffect' -v
```

Expected: compile failure — `undefined: NewSetBackEffect`, `undefined: NewClearBackEffect`, `undefined: BackEffectShow`.

- [ ] **Step 3: Write `set_back_effect.go`**

```go
package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const SetBackEffectWriter = "SetBackEffect"

// Back-effect selectors. These are wire constants proven by the decompile
// (task-281 design §1.1): the client branches on nEffect and tweens the
// resolved back-layer page's alpha to 255 (show) or 0 (hide). Any other
// value makes CMapLoadable::OnSetBackEffect return without touching the
// field, which is why atlas-maps rejects it at the command consumer.
const (
	BackEffectShow byte = 0
	BackEffectHide byte = 1
)

// SetBackEffect is the clientbound CMapLoadable::OnSetBackEffect packet. The
// client reads four fields and tweens the alpha of every IWzGr2DLayer on
// back-layer page pageId to 255 (effect 0) or 0 (effect 1) over duration
// milliseconds. duration is a FADE length, not a lifetime. fieldId is decoded
// but unread by the v95 handler; it is kept because it occupies four wire
// bytes and omitting it would desynchronise the stream.
// packet-audit:fname CMapLoadable::OnSetBackEffect
type SetBackEffect struct {
	effect   byte
	fieldId  uint32
	pageId   byte
	duration uint32
}

func NewSetBackEffect(effect byte, fieldId uint32, pageId byte, duration uint32) SetBackEffect {
	return SetBackEffect{effect: effect, fieldId: fieldId, pageId: pageId, duration: duration}
}

func (m SetBackEffect) Effect() byte     { return m.effect }
func (m SetBackEffect) FieldId() uint32  { return m.fieldId }
func (m SetBackEffect) PageId() byte     { return m.pageId }
func (m SetBackEffect) Duration() uint32 { return m.duration }

func (m SetBackEffect) Operation() string { return SetBackEffectWriter }
func (m SetBackEffect) String() string {
	return fmt.Sprintf("effect [%d] fieldId [%d] pageId [%d] duration [%d]", m.effect, m.fieldId, m.pageId, m.duration)
}

func (m SetBackEffect) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.effect)
		w.WriteInt(m.fieldId)
		w.WriteByte(m.pageId)
		w.WriteInt(m.duration)
		return w.Bytes()
	}
}

func (m *SetBackEffect) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.effect = r.ReadByte()
		m.fieldId = r.ReadUint32()
		m.pageId = r.ReadByte()
		m.duration = r.ReadUint32()
	}
}
```

If `response.Writer` / `request.Reader` do not expose `WriteByte` / `ReadByte` under those exact names, grep a sibling in `libs/atlas-packet/field/clientbound/` that writes a single byte and use whatever it uses. Do not invent a method name.

- [ ] **Step 4: Write `clear_back_effect.go`**

Copy `field_obstacle_all_reset.go` verbatim and rename: `ClearBackEffectWriter = "ClearBackEffect"`, type `ClearBackEffect struct{}`, `NewClearBackEffect()`, `// packet-audit:fname CMapLoadable::OnClearBackEffect`. The doc comment must say the effect is **field-wide** — `CMapLoadable::OnClearBackEffect` thunks to `ReloadBack`, which rebuilds the whole back-layer set, so this clears every page at once. There is no per-page clear on the wire.

- [ ] **Step 5: Run the tests to verify they pass**

```bash
go test ./field/clientbound/ -run 'BackEffect' -v
```

Expected: PASS, all subtests.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-packet/field/clientbound/set_back_effect.go \
        libs/atlas-packet/field/clientbound/set_back_effect_test.go \
        libs/atlas-packet/field/clientbound/clear_back_effect.go \
        libs/atlas-packet/field/clientbound/clear_back_effect_test.go
git commit -m "feat(packet): add SetBackEffect and ClearBackEffect clientbound codecs"
```

---

## Task 4: Registry links, seed-template routes, evidence, and matrix promotion

Mechanical and repeated across 8 registry files, 8 templates and 20 cells. Deliberately left as one task: it is the same edit N times, and the matrix only checks out green once all of it lands. See `context.md`.

### Files

- `docs/packets/registry/gms_v72.yaml:836` — add `packet: field/clientbound/FieldSetBackEffect` to the `SET_BACK_EFFECT` entry
- `docs/packets/registry/gms_v79.yaml:857` — same
- `docs/packets/registry/gms_v83.yaml` — both ops (lines 657, 667)
- `docs/packets/registry/gms_v84.yaml` — both ops (lines 882, 898)
- `docs/packets/registry/gms_v87.yaml` — both ops (lines 714, 724)
- `docs/packets/registry/gms_v92.yaml` — both ops (lines 748, 760)
- `docs/packets/registry/gms_v95.yaml` — both ops (lines 761, 771)
- `docs/packets/registry/jms_v185.yaml` — both ops (lines 623, 633)
- `services/atlas-configurations/seed-data/templates/template_gms_72_1.json` — route `SetBackEffect` only
- `services/atlas-configurations/seed-data/templates/template_gms_79_1.json` — route `SetBackEffect` only
- `services/atlas-configurations/seed-data/templates/template_gms_83_1.json` — route both
- `services/atlas-configurations/seed-data/templates/template_gms_84_1.json` — route both
- `services/atlas-configurations/seed-data/templates/template_gms_87_1.json` — route both
- `services/atlas-configurations/seed-data/templates/template_gms_92_1.json` — route both
- `services/atlas-configurations/seed-data/templates/template_gms_95_1.json` — route both
- `services/atlas-configurations/seed-data/templates/template_jms_185_1.json` — route both
- `docs/packets/feature-families.yaml` — add the `back_effect` family
- `docs/packets/feature-na-evidence.yaml` — add six entries
- `docs/packets/audits/STATUS.md` — regenerated, do not hand-edit
- `docs/packets/audits/status.json` — regenerated, do not hand-edit

Patterns to copy: `services/atlas-configurations/seed-data/templates/template_gms_95_1.json:3645-3652` (the `{"opCode","writer","fname","services"}` route object), `docs/packets/registry/gms_v95.yaml:753` (a `packet:` link on a registry entry), `docs/packets/feature-na-evidence.yaml:9-20` (an `entries:` item).

Module root: repo root for `go run ./tools/packet-audit ...`.

- [ ] **Step 1: Add the `packet:` link to all 14 registry entries**

Each entry gains one line, e.g. in `docs/packets/registry/gms_v95.yaml`:

```yaml
- op: SET_BACK_EFFECT
  direction: clientbound
  opcode: 144
  fname: CMapLoadable::OnSetBackEffect
  packet: field/clientbound/FieldSetBackEffect
  provenance: csv-import
```

Do not change any existing `opcode`, `fname`, `ida`, `provenance` or `note` value.

- [ ] **Step 2: Add the seed-template routes**

One object per op per template, inserted in ascending `opCode` order among its neighbours:

```json
      {
        "opCode": "0x90",
        "writer": "SetBackEffect",
        "fname": "CMapLoadable::OnSetBackEffect",
        "services": [
          "channel"
        ]
      },
      {
        "opCode": "0x92",
        "writer": "ClearBackEffect",
        "fname": "CMapLoadable::OnClearBackEffect",
        "services": [
          "channel"
        ]
      },
```

`opCode` per template comes from the per-version opcode table in Global Constraints, written in the same casing the surrounding entries use (`0x90`, not `0x090`). **v48 and v61 templates get no route** — their cells are ⬜.

- [ ] **Step 3: Pin the evidence records**

From the repo root, once per cell (14 TIER1-FIXTURE + 6 VERSION-ABSENT):

```bash
go run ./tools/packet-audit evidence pin --packet field/clientbound/FieldSetBackEffect --version gms_v95 --ida "CMapLoadable::OnSetBackEffect" --category TIER1-FIXTURE
```

`--ida` takes the **fname string**, not an address; the tool resolves the address from the version's export. Categories: `TIER1-FIXTURE` for the 14 live cells, `VERSION-ABSENT` for the six blank ones (`FieldSetBackEffect` × gms_v48, gms_v61; `FieldClearBackEffect` × gms_v48, gms_v61, gms_v72, gms_v79). Records land at `docs/packets/evidence/<version>/field.clientbound.Field<Op>.yaml`. A VERSION-ABSENT record's `ida.address` reads `ABSENT` — see `docs/packets/evidence/jms_v185/guild.clientbound.GuildBBSEntryNotFound.yaml` for the shape.

- [ ] **Step 4: Declare the feature family and its absence evidence**

Add to `docs/packets/feature-families.yaml`:

```yaml
  back_effect:
    - SET_BACK_EFFECT
    - CLEAR_BACK_EFFECT
```

This makes `matrix --check` hold every ⬜ cell to positive absence evidence once a sibling is verified (`tools/packet-audit/cmd/na_consistency.go:120-205`). Add the six matching entries to `docs/packets/feature-na-evidence.yaml`, one per `op × version`, each with `evidence:` prose **copied from the Task 2 structures records** — the router arms present, which op owns the opcode slot, and the search hit counts. Do not paraphrase and do not write evidence Task 2 did not produce.

- [ ] **Step 5: Regenerate and check**

```bash
go run ./tools/packet-audit matrix
go run ./tools/packet-audit matrix --check
go run ./tools/packet-audit fname-doc --check
go run ./tools/packet-audit operations --check
```

Expected: `matrix` rewrites `docs/packets/audits/STATUS.md` and `status.json`; all three `--check` runs exit 0. Confirm the two rows now read ✅ on all 14 live cells and ⬜ on the six blanks:

```bash
grep -E '^\| (SET|CLEAR)_BACK_EFFECT ' docs/packets/audits/STATUS.md
```

- [ ] **Step 6: Commit**

```bash
git add docs/packets/ services/atlas-configurations/seed-data/templates/
git commit -m "feat(packets): promote SET/CLEAR_BACK_EFFECT cells and route both ops"
```

---

## Task 5: `atlas-maps` back-effect registry and processor

### Files

- `services/atlas-maps/atlas.com/maps/map/backeffect/registry.go` — new file
- `services/atlas-maps/atlas.com/maps/map/backeffect/registry_test.go` — new file
- `services/atlas-maps/atlas.com/maps/map/backeffect/processor.go` — new file
- `services/atlas-maps/atlas.com/maps/map/jukebox/registry.go:1-84` — read-only; the singleton + `FieldKey` shape to copy
- `services/atlas-maps/atlas.com/maps/map/jukebox/processor.go` — read-only; the tenant-resolution shape to copy

Patterns to copy: `services/atlas-maps/atlas.com/maps/map/jukebox/registry.go:22-56` (mutex + `sync.Once` singleton + Set/Get/Delete), `services/atlas-maps/atlas.com/maps/map/jukebox/registry_test.go:16-31` (test setup: `tenant.Create(uuid.New(), "GMS", 83, 1)` → `tenant.WithContext(context.Background(), ten)` → `field.NewBuilder(0, 1, 100000000).Build()`), `services/atlas-maps/atlas.com/maps/map/jukebox/registry_test.go:87-108` (tenant-isolation case).

Module root: `services/atlas-maps/atlas.com/maps`.

**Interfaces produced** (Tasks 6 and 7 consume these exact names):

```go
type FieldKey struct { Tenant tenant.Model; Field field.Model }
type BackEffectEntry struct { Effect byte; FieldId uint32; PageId byte; Duration uint32 }

type Processor interface {
	Set(f field.Model, entry BackEffectEntry)
	Clear(f field.Model) bool           // returns whether anything was removed
	GetActive(f field.Model) []BackEffectEntry
}
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor
```

- [ ] **Step 1: Write the failing test**

`registry_test.go` — four functions, setup copied from `map/jukebox/registry_test.go:16-31`.

| test | action | expected |
|---|---|---|
| `TestSetThenGetActive` | `Set(f, {Effect:0, FieldId:100000000, PageId:1, Duration:1000})`, then `GetActive(f)` | slice of length 1; `[0]` equals the entry field-for-field |
| `TestSetReplacesSamePageInPlace` | `Set` page 1 (`Effect:0`), `Set` page 2 (`Effect:0`), then `Set` page 1 again with `Effect:1, Duration:250` | `GetActive` length 2; `[0].PageId == 1 && [0].Effect == 1 && [0].Duration == 250`; `[1].PageId == 2` — the replaced page keeps position 0 |
| `TestClearRemovesEveryPage` | `Set` pages 1, 2, 3; `Clear(f)` returns `true`; `GetActive(f)` | length 0; a second `Clear(f)` returns `false` |
| `TestBackEffectIsTenantIsolated` | two tenants (`tenant.Create(uuid.New(), "GMS", 83, 1)` twice, distinct uuids), same `field.Model`; `Set` page 1 under tenant A only | `GetActive` under A has length 1; under B has length 0 |

Because the registry is a package-level singleton, each test must use a fresh tenant uuid so cases cannot see each other's entries — this is exactly what `map/jukebox/registry_test.go` does.

- [ ] **Step 2: Run the test to verify it fails**

```bash
cd services/atlas-maps/atlas.com/maps && go test ./map/backeffect/... -v
```

Expected: build failure — no such package.

- [ ] **Step 3: Write `registry.go`**

Copy `map/jukebox/registry.go` and change the value type to a slice. `FieldKey` is identical to `jukebox.FieldKey` — same two fields, same order. Drop `ExpiresAt`, `ExpiredEntry`, `GetExpired` and `DeleteEntry`; there is no reaper (design §3.1/§3.4) and a leftover expiry field would invite one.

```go
type BackEffectEntry struct {
	Effect   byte
	FieldId  uint32
	PageId   byte
	Duration uint32
}

type Registry struct {
	mutex   sync.RWMutex
	entries map[FieldKey][]BackEffectEntry
}
```

`Set(key FieldKey, entry BackEffectEntry)` scans the field's slice for a matching `PageId`; on a hit it overwrites in place (preserving index), otherwise it appends. `Get(key FieldKey) []BackEffectEntry` returns a **copy** of the slice — never the backing array, or a caller mutates registry state through it. `Clear(key FieldKey) bool` reports whether an entry existed, then deletes the map key.

Add a comment on `Duration` recording that it is the client's fade length in milliseconds and is deliberately **not** an expiry — the analogue of the `maxJukeboxDuration` comment at `services/atlas-maps/atlas.com/maps/kafka/consumer/map/consumer.go:72-76`.

- [ ] **Step 4: Write `processor.go`**

Copy the shape of `map/jukebox/processor.go` exactly: an interface, a `ProcessorImpl{l, ctx}`, `NewProcessor`, a `var _ Processor = (*ProcessorImpl)(nil)` assertion, and `tenant.MustFromContext(p.ctx)` to build the `FieldKey` in every method. Log `Set` and `Clear` at debug with map id, instance, page id and effect.

- [ ] **Step 5: Run the test to verify it passes**

```bash
cd services/atlas-maps/atlas.com/maps && go test ./map/backeffect/... -v
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-maps/atlas.com/maps/map/backeffect/
git commit -m "feat(atlas-maps): add per-field back-effect registry and processor"
```

---

## Task 6: `atlas-maps` command consumer and status-event producer

### Files

- `services/atlas-maps/atlas.com/maps/kafka/message/map/command.go` — add the two command type constants (const block, line 11) and the two body structs (end of file)
- `services/atlas-maps/atlas.com/maps/kafka/message/map/kafka.go` — add the two status-event type constants (const block, line 11) and the two event body structs (end of file)
- `services/atlas-maps/atlas.com/maps/map/backeffect/producer.go` — new file
- `services/atlas-maps/atlas.com/maps/kafka/consumer/map/consumer.go` — register two handlers in `InitHandlers` (line 32) and add two handler functions beside `handlePlayJukeboxCommand` (line 78)
- `services/atlas-maps/atlas.com/maps/kafka/consumer/map/consumer_test.go` — new test cases
- `services/atlas-maps/atlas.com/maps/map/jukebox/producer.go` — read-only; the provider shape to copy

Patterns to copy: `services/atlas-maps/atlas.com/maps/map/jukebox/producer.go:14-29` (`JukeboxStartEventProvider`), `services/atlas-maps/atlas.com/maps/kafka/consumer/map/consumer.go:78-101` (`handlePlayJukeboxCommand`), `services/atlas-maps/atlas.com/maps/kafka/consumer/map/consumer_test.go:18-48` (consumer test setup: hand-built `mapKafka.Command[...]` literal, call the handler, re-read via the processor).

Module root: `services/atlas-maps/atlas.com/maps`.

**Interfaces consumed:** `backeffect.NewProcessor`, `backeffect.BackEffectEntry`, `backeffect.Processor.Set/Clear/GetActive` (Task 5).

**Interfaces produced:**

```go
// kafka/message/map/command.go
const CommandTypeSetBackEffect   = "SET_BACK_EFFECT"
const CommandTypeClearBackEffect = "CLEAR_BACK_EFFECT"
type SetBackEffectCommandBody struct {
	Effect   uint8  `json:"effect"`
	FieldId  uint32 `json:"fieldId"`
	PageId   uint8  `json:"pageId"`
	Duration uint32 `json:"duration"`
}
type ClearBackEffectCommandBody struct{}

// kafka/message/map/kafka.go
const EventTopicMapStatusTypeBackEffectSet   = "BACK_EFFECT_SET"
const EventTopicMapStatusTypeBackEffectClear = "BACK_EFFECT_CLEAR"
type BackEffectSet struct {
	Effect   uint8  `json:"effect"`
	FieldId  uint32 `json:"fieldId"`
	PageId   uint8  `json:"pageId"`
	Duration uint32 `json:"duration"`
}
type BackEffectClear struct{}

// map/backeffect/producer.go
func BackEffectSetEventProvider(transactionId uuid.UUID, f field.Model, e BackEffectEntry) model.Provider[[]kafka.Message]
func BackEffectClearEventProvider(transactionId uuid.UUID, f field.Model) model.Provider[[]kafka.Message]
```

- [ ] **Step 1: Write the failing test**

New cases in `kafka/consumer/map/consumer_test.go`, setup copied from `consumer_test.go:18-48`.

| test | command | expected |
|---|---|---|
| `TestHandleSetBackEffectCommand_RecordsEntry` | `Type: mapKafka.CommandTypeSetBackEffect`, body `{Effect:0, FieldId:100000000, PageId:1, Duration:1000}` | `backeffect.NewProcessor(l, ctx).GetActive(f)` has length 1 and matches the body field-for-field |
| `TestHandleSetBackEffectCommand_RejectsInvalidEffect` | same, but `Effect: 2` | `GetActive(f)` has length 0 |
| `TestHandleSetBackEffectCommand_IgnoresWrongType` | `Type: mapKafka.CommandTypeWeatherStart`, valid body | `GetActive(f)` has length 0 |
| `TestHandleClearBackEffectCommand_RemovesEntries` | `Set` two pages first via the processor, then `Type: mapKafka.CommandTypeClearBackEffect` | `GetActive(f)` has length 0 |
| `TestHandleClearBackEffectCommand_EmptyFieldIsNotAnError` | no prior entries, `Type: mapKafka.CommandTypeClearBackEffect` | handler returns without panicking; `GetActive(f)` has length 0 |

Each test uses a fresh tenant uuid so the package-level registry cannot leak between cases. The existing tests do not assert on the produced Kafka message (there is no broker in this test binary) — neither do these; the event shape is covered by `map/backeffect/producer.go`'s own construction and by Task 8's channel-side consumer test.

- [ ] **Step 2: Run the test to verify it fails**

```bash
cd services/atlas-maps/atlas.com/maps && go test ./kafka/consumer/map/... -run BackEffect -v
```

Expected: compile failure — `undefined: mapKafka.CommandTypeSetBackEffect`, `undefined: handleSetBackEffectCommand`.

- [ ] **Step 3: Add the message types**

Append the constants and body structs listed under "Interfaces produced" to `kafka/message/map/command.go` and `kafka/message/map/kafka.go`, keeping the existing const-block and struct ordering style.

- [ ] **Step 4: Write `map/backeffect/producer.go`**

Copy `map/jukebox/producer.go` and substitute. Key stays `producer.CreateKey(int(f.MapId()))`; envelope stays `mapKafka.StatusEvent[T]` with `TransactionId/WorldId/ChannelId/MapId/Instance/Type/Body`.

- [ ] **Step 5: Write the two consumer handlers**

Both modelled on `handlePlayJukeboxCommand` (`consumer.go:78-101`). `handleSetBackEffectCommand`:

1. Return early unless `c.Type == mapKafka.CommandTypeSetBackEffect`.
2. Reject `c.Body.Effect` outside `{0, 1}` — log at **warn** naming the value, map id and instance, then return without mutating state or producing. The client handler returns without acting on any other value (design §1.1/§3.2), so broadcasting it would be a guaranteed no-op.
3. Build `f := field.NewBuilder(c.WorldId, c.ChannelId, c.MapId).SetInstance(c.Instance).Build()`.
4. `backeffect.NewProcessor(l, ctx).Set(f, backeffect.BackEffectEntry{...})`.
5. Produce `backeffect.BackEffectSetEventProvider(c.TransactionId, f, entry)` onto `mapKafka.EnvEventTopicMapStatus`; log the produce error at error.

**No clamp on `Duration`.** Add a comment saying why: it is a fade length bounded by the client's own tween, with no denial-of-service shape comparable to pinning a field's BGM — the counterpart to the `maxJukeboxDuration` comment above it.

`handleClearBackEffectCommand`: type guard, build `f`, call `Clear(f)`; when it returns `false`, log at **debug** that the field had nothing — and **produce the event anyway**, so a desynced client can be reset (PRD FR-4).

Register both in `InitHandlers` (`consumer.go:32-44`) with the same `rf(t, message.AdaptHandler(message.PersistentConfig(...)))` line shape as the jukebox handler.

- [ ] **Step 6: Run the tests to verify they pass**

```bash
cd services/atlas-maps/atlas.com/maps && go test ./kafka/consumer/map/... ./map/backeffect/... -v
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-maps/atlas.com/maps/kafka services/atlas-maps/atlas.com/maps/map/backeffect
git commit -m "feat(atlas-maps): consume back-effect commands and emit status events"
```

---

## Task 7: `atlas-maps` back-effect REST collection

### Files

- `services/atlas-maps/atlas.com/maps/map/backeffect/rest.go` — new file
- `services/atlas-maps/atlas.com/maps/map/backeffect/resource.go` — new file
- `services/atlas-maps/atlas.com/maps/map/backeffect/resource_test.go` — new file
- `services/atlas-maps/atlas.com/maps/main.go:15,149` — import the package and add `AddRouteInitializer(backeffect.InitResource(GetServer()))`
- `services/atlas-maps/atlas.com/maps/map/jukebox/rest.go` — read-only; the `RestModel` + `Transform` shape
- `services/atlas-maps/atlas.com/maps/map/jukebox/resource.go` — read-only; the route + handler shape

Patterns to copy: `services/atlas-maps/atlas.com/maps/map/jukebox/resource.go:23-27` (route registration via `rest.RegisterHandler(l)(si)(name, handler)`), `services/atlas-maps/atlas.com/maps/map/jukebox/resource.go:30-40` (the nested `rest.ParseWorldId`/`ParseChannelId`/`ParseMapId`/`ParseInstanceId` chain), `services/atlas-ban/atlas.com/ban/report/resource.go:62` (`server.MarshalResponse[[]RestModel]` — the collection form).

Module root: `services/atlas-maps/atlas.com/maps`.

**Interfaces produced** (Task 9's client mirrors this exactly):

```
GET /worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/backEffects
  200 -> JSON:API array of type "backEffect" (empty array when none)
```

```go
type RestModel struct {
	Id       string `json:"-"`
	Effect   uint8  `json:"effect"`
	FieldId  uint32 `json:"fieldId"`
	PageId   uint8  `json:"pageId"`
	Duration uint32 `json:"duration"`
}
func (m RestModel) GetName() string { return "backEffect" }
func Transform(e BackEffectEntry) (RestModel, error)
func InitResource(si jsonapi.ServerInformation) server.RouteInitializer
```

`Id` is `strconv.Itoa(int(e.PageId))` — the page is the natural per-field identity.

- [ ] **Step 1: Write the failing test**

`resource_test.go` — drive the real router, as `services/atlas-messengers/atlas.com/messengers/messenger/resource_paginate_test.go:210-240` does. Two cases:

| test | state | expected |
|---|---|---|
| `TestGetBackEffectsInMap_ReturnsEntries` | processor `Set` page 1 (`Effect:0, FieldId:100000000, Duration:1000`) then page 2 (`Effect:1, Duration:0`) | 200; JSON:API `data` array of length 2; `data[0].id == "1"`, `data[0].type == "backEffect"`, `data[0].attributes.effect == 0`, `data[0].attributes.duration == 1000`; `data[1].id == "2"`, `data[1].attributes.effect == 1` |
| `TestGetBackEffectsInMap_EmptyIsTwoHundred` | no entries for this tenant/field | **200**, not 404; `data` is an empty array |

The empty-is-200 case is the deviation from PRD §5 that design §3.3 settles — assert it explicitly so nobody "fixes" it back to a 404.

- [ ] **Step 2: Run the test to verify it fails**

```bash
cd services/atlas-maps/atlas.com/maps && go test ./map/backeffect/... -run Resource -v
```

Expected: compile failure — `undefined: InitResource`.

- [ ] **Step 3: Write `rest.go`**

Copy `map/jukebox/rest.go`. Include the `SetID` method. This is a server-side model, so `SetToOneReferenceID` / `SetToManyReferenceIDs` are not required here (the atlas-maps jukebox model omits them); Task 9's **client-side** model does need them.

- [ ] **Step 4: Write `resource.go`**

Copy `map/jukebox/resource.go` and change three things: the route path segment to `backEffects`, the handler name constant to `getBackEffectsInMap`, and the body to map the whole slice instead of branching on `ok`:

```go
entries := NewProcessor(d.Logger(), d.Context()).GetActive(f)
res := make([]RestModel, 0, len(entries))
for _, e := range entries {
	rm, err := Transform(e)
	if err != nil {
		d.Logger().WithError(err).Errorf("Creating REST model.")
		server.WriteErrorResponse(d.Logger())(w)(err)
		return
	}
	res = append(res, rm)
}
server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(r.URL.Query())(res)
```

`res` is allocated with `make(..., 0, len(entries))` so an empty result marshals as `[]`, not `null`.

- [ ] **Step 5: Register the route**

In `services/atlas-maps/atlas.com/maps/main.go`, add the import beside `"atlas-maps/map/jukebox"` (line 15) and `AddRouteInitializer(backeffect.InitResource(GetServer())).` beside the jukebox initializer (line 149).

- [ ] **Step 6: Run the tests to verify they pass**

```bash
cd services/atlas-maps/atlas.com/maps && go build ./... && go test ./map/backeffect/... -v
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-maps/atlas.com/maps/map/backeffect services/atlas-maps/atlas.com/maps/main.go
git commit -m "feat(atlas-maps): serve active back-effects over REST"
```

---

## Task 8: `atlas-channel` writers and status-event broadcast

### Files

- `services/atlas-channel/atlas.com/channel/kafka/message/map/kafka.go` — add the two status-event type constants (const block, line 11) and the two body structs (end of file), **byte-identical to Task 6's atlas-maps definitions**
- `services/atlas-channel/atlas.com/channel/socket/writer/set_back_effect.go` — new file
- `services/atlas-channel/atlas.com/channel/socket/writer/clear_back_effect.go` — new file
- `services/atlas-channel/atlas.com/channel/main.go` — add `fieldcb.SetBackEffectWriter` and `fieldcb.ClearBackEffectWriter` to the writer-name slice (after line 872)
- `services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go` — register two status-event handlers (line 113) and implement them beside `handleStatusEventJukeboxStart` (line 1104)
- `services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer_test.go` — new test cases

Patterns to copy: `services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go:1104-1127` (`handleStatusEventJukeboxStart`), `services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go:113-118` (handler registration), `services/atlas-channel/atlas.com/channel/socket/writer/play_jukebox.go:1-10` (writer body function), `services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer_test.go:502-524` (`stubDoorAnnounceForJukebox` — swap the package-level `doorAnnounce` var, record `{Writer, Body}`, return a `restore` func).

Module root: `services/atlas-channel/atlas.com/channel`.

**Interfaces consumed:** `fieldcb.SetBackEffectWriter`, `fieldcb.NewSetBackEffect`, `fieldcb.ClearBackEffectWriter`, `fieldcb.NewClearBackEffect` (Task 3); the `BACK_EFFECT_SET` / `BACK_EFFECT_CLEAR` event contract (Task 6).

- [ ] **Step 1: Write the failing test**

New cases in `kafka/consumer/map/consumer_test.go`, using `newTestCtx`, `newTestField`, `addFieldSession`, `newTestServerModel` (`consumer_test.go:32-66,530-534`) and a `stubDoorAnnounceForBackEffect` copied from `stubDoorAnnounceForJukebox` (`consumer_test.go:502-524`).

| test | event | expected |
|---|---|---|
| `TestHandleStatusEventBackEffectSet_BroadcastsToField` | `Type: BACK_EFFECT_SET`, body `{Effect:0, FieldId:100000000, PageId:1, Duration:1000}`, two sessions in the field | two recorded announces; each `Writer == fieldcb.SetBackEffectWriter`; each recorded body decodes (through `fieldcb.SetBackEffect.Decode`) to `Effect 0, FieldId 100000000, PageId 1, Duration 1000` |
| `TestHandleStatusEventBackEffectSet_IgnoresOtherChannel` | same event but `ChannelId` not matching the `server.Model` | zero recorded announces |
| `TestHandleStatusEventBackEffectClear_BroadcastsToField` | `Type: BACK_EFFECT_CLEAR`, one session in the field | one recorded announce; `Writer == fieldcb.ClearBackEffectWriter`; recorded body is zero-length |

Add a `decodeSetBackEffect` helper beside `decodePlayJukebox` (`consumer_test.go:517-524`) that decodes a captured wire body through the real codec — this is the cross-service seam assertion CLAUDE.md's "Done means verified" requires: the channel test asserts the exact contract `atlas-maps` produces.

- [ ] **Step 2: Run the test to verify it fails**

```bash
cd services/atlas-channel/atlas.com/channel && go test ./kafka/consumer/map/... -run BackEffect -v
```

Expected: compile failure — `undefined: handleStatusEventBackEffectSet`.

- [ ] **Step 3: Mirror the message types**

Add the same four declarations to `kafka/message/map/kafka.go` that Task 6 added on the atlas-maps side. The two files are kept keystroke-identical by convention; any drift is a wire bug.

- [ ] **Step 4: Add the writer body functions and register the writers**

`socket/writer/set_back_effect.go` and `clear_back_effect.go`, copied from `socket/writer/play_jukebox.go` and `socket/writer/field_obstacle_all_reset.go` respectively. These wrappers have no production call site today — the consumer calls `fieldcb.New...().Encode` directly — but the file-per-writer convention is uniform in this package, so match it.

In `main.go`, add `fieldcb.SetBackEffectWriter,` and `fieldcb.ClearBackEffectWriter,` to the writer-name slice immediately after `fieldcb.FieldObstacleAllResetWriter,` (line 872).

- [ ] **Step 5: Write the two status-event handlers**

Copy `handleStatusEventJukeboxStart` (`consumer.go:1104-1127`) twice. Each: type guard on `e.Type`, `sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId)` guard, debug log, `f := field.NewBuilder(...).SetInstance(e.Instance).Build()`, then

```go
err := _map.NewProcessor(l, ctx).ForSessionsInMap(f, func(s session.Model) error {
	return doorAnnounce(l, ctx, wp, fieldcb.SetBackEffectWriter, fieldcb.NewSetBackEffect(byte(e.Body.Effect), e.Body.FieldId, byte(e.Body.PageId), e.Body.Duration).Encode, s)
})
```

with an error log on failure. The clear handler is the same with `ClearBackEffectWriter` / `NewClearBackEffect()`. Register both in the handler block at `consumer.go:113-118`.

- [ ] **Step 6: Run the tests to verify they pass**

```bash
cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./kafka/consumer/map/... -run BackEffect -v
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka services/atlas-channel/atlas.com/channel/socket/writer services/atlas-channel/atlas.com/channel/main.go
git commit -m "feat(atlas-channel): broadcast back-effect set and clear packets"
```

---

## Task 9: `atlas-channel` REST client and map-entry replay

### Files

- `services/atlas-channel/atlas.com/channel/backeffect/rest.go` — new file
- `services/atlas-channel/atlas.com/channel/backeffect/requests.go` — new file
- `services/atlas-channel/atlas.com/channel/backeffect/processor.go` — new file
- `services/atlas-channel/atlas.com/channel/backeffect/mock/processor.go` — new file
- `services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go` — add the `routine.Go` dispatch (after line 383) and `announceActiveBackEffects` (after line 1162)
- `services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer_test.go` — new test cases

Patterns to copy: `services/atlas-channel/atlas.com/channel/events/requests.go:19-21` (`requests.GetRequest[[]RestModel]` — the collection request form), `services/atlas-channel/atlas.com/channel/events/processor.go:37` (`requests.SliceProvider[RestModel, RestModel](p.l, p.ctx)(req, Extract, model.Filters[RestModel]())()`), `services/atlas-channel/atlas.com/channel/events/rest.go:1-45` (client `RestModel` including the `SetToOneReferenceID` / `SetToManyReferenceIDs` no-ops api2go requires), `services/atlas-channel/atlas.com/channel/jukebox/requests.go:12-16` (the `worlds/%d/channels/%d/maps/%d/instances/%s` resource template and `requests.RootUrlFor(ctx, "MAPS")`), `services/atlas-channel/atlas.com/channel/jukebox/mock/processor.go:1-18` (the mock shape), `services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go:1153-1162` (`announceActiveJukebox`).

Module root: `services/atlas-channel/atlas.com/channel`.

**Interfaces produced:**

```go
type Processor interface { GetActive(f field.Model) ([]RestModel, error) }
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor
```

`RestModel` mirrors Task 7's server-side model field-for-field, `GetName()` returns `"backEffect"`, and the resource path is `worlds/%d/channels/%d/maps/%d/instances/%s/backEffects`.

- [ ] **Step 1: Write the failing test**

New cases in `kafka/consumer/map/consumer_test.go`, reusing `stubDoorAnnounceForBackEffect` from Task 8.

| test | client stub | expected |
|---|---|---|
| `TestAnnounceActiveBackEffects_ReplaysWithZeroDuration` | `GetActive` returns two models: `{PageId:1, Effect:0, FieldId:100000000, Duration:1000}`, `{PageId:2, Effect:1, FieldId:100000000, Duration:500}` | two announces to the single entering session, in that order; each `Writer == fieldcb.SetBackEffectWriter`; decoded bodies are `{Effect:0, FieldId:100000000, PageId:1, Duration:0}` and `{Effect:1, FieldId:100000000, PageId:2, Duration:0}` — **`Duration` is 0 on both**, `Effect`/`FieldId`/`PageId` replay as stored |
| `TestAnnounceActiveBackEffects_EmptyAnnouncesNothing` | `GetActive` returns an empty slice, nil error | zero announces |
| `TestAnnounceActiveBackEffects_LookupFailureIsFailOpen` | `GetActive` returns `nil, errors.New("boom")` | zero announces, **no panic and no returned error** — the fail-open assertion PRD FR-5 requires |

`announceActiveBackEffects` must therefore take its processor from a swappable package-level seam so the test can inject `backeffectmock.ProcessorMock`. Follow whatever `announceActiveVisuals` (`consumer.go:787`) already does for the `events` processor and match it; do not invent a second injection mechanism.

- [ ] **Step 2: Run the test to verify it fails**

```bash
cd services/atlas-channel/atlas.com/channel && go test ./kafka/consumer/map/... -run AnnounceActiveBackEffects -v
```

Expected: compile failure — `undefined: announceActiveBackEffects`.

- [ ] **Step 3: Write the client package**

Four files, each a direct copy of its `jukebox/` or `events/` counterpart with the collection type substituted:

- `rest.go` — `RestModel` with `GetID`/`GetName`/`SetID`/`SetToOneReferenceID`/`SetToManyReferenceIDs`/`Extract`, plus a comment stating that the field names and `GetName()` must match `services/atlas-maps/atlas.com/maps/map/backeffect/rest.go` exactly (the `events/rest.go` comment is the model).
- `requests.go` — `getBaseRequest` via `requests.RootUrlFor(ctx, "MAPS")`; `requestBackEffectsInMap(ctx, f) requests.Request[[]RestModel]` via `requests.GetRequest[[]RestModel]`.
- `processor.go` — `GetActive` via `requests.SliceProvider[RestModel, RestModel](p.l, p.ctx)(requestBackEffectsInMap(p.ctx, f), Extract, model.Filters[RestModel]())()`.
- `mock/processor.go` — `ProcessorMock{GetActiveFunc func(f field.Model) ([]backeffect.RestModel, error)}` with the `var _ backeffect.Processor = (*ProcessorMock)(nil)` assertion.

- [ ] **Step 4: Write `announceActiveBackEffects` and dispatch it**

Beside `announceActiveJukebox` (`consumer.go:1153-1162`):

```go
// announceActiveBackEffects replays the field's active back-effects to a
// single entering session. Duration is replayed as 0: the fade already
// happened for everyone else, so a late joiner should land on the end state
// rather than re-run the tween (design §4.3). Fails open — an unreachable
// atlas-maps costs the background, not the map entry.
func announceActiveBackEffects(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, f field.Model, s session.Model) {
	es, err := backeffect.NewProcessor(l, ctx).GetActive(f)
	if err != nil {
		return
	}
	for _, e := range es {
		_ = doorAnnounce(l, ctx, wp, fieldcb.SetBackEffectWriter, fieldcb.NewSetBackEffect(byte(e.Effect), e.FieldId, byte(e.PageId), 0).Encode, s)
	}
}
```

Dispatch it from its own `routine.Go` block immediately after the `announceActiveJukebox` block at `consumer.go:381-383`. Do **not** fold it into `announceActiveVisuals` — that replays event visuals from `atlas-events`, a different source with a different failure mode (design §4.3).

- [ ] **Step 5: Run the tests to verify they pass**

```bash
cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./kafka/consumer/map/... ./backeffect/... -v
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/backeffect services/atlas-channel/atlas.com/channel/kafka/consumer/map
git commit -m "feat(atlas-channel): replay active back-effects to entering sessions"
```

---

## Task 10: `libs/atlas-saga` actions and payloads

### Files

- `libs/atlas-saga/model.go:286-290` — add the two `Action` constants beside `PlayJukebox`
- `libs/atlas-saga/payloads.go:1311-1319` — add the two payload structs beside `PlayJukeboxPayload`
- `libs/atlas-saga/unmarshal.go:642-647` — add the two decode `case` arms
- `libs/atlas-saga/unmarshal_test.go:1649-1660` — new test cases

Patterns to copy: `libs/atlas-saga/model.go:286-290` (the `PlayJukebox` const with its explanatory comment), `libs/atlas-saga/payloads.go:1311-1319` (`PlayJukeboxPayload`), `libs/atlas-saga/unmarshal.go:642-647` (the `case PlayJukebox:` arm), `libs/atlas-saga/unmarshal_test.go:1649-1660` (`TestUnmarshalPlayJukeboxStep` — raw JSON literal, `json.Unmarshal` into `Step[any]`, type-assert the payload).

Module root: `libs/atlas-saga`.

**Interfaces produced:**

```go
const (
	SetBackEffect   Action = "set_back_effect"
	ClearBackEffect Action = "clear_back_effect"
)

type SetBackEffectPayload struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
	Effect    uint8      `json:"effect"`
	FieldId   uint32     `json:"fieldId"`
	PageId    uint8      `json:"pageId"`
	Duration  uint32     `json:"duration"`
}

type ClearBackEffectPayload struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
}
```

- [ ] **Step 1: Write the failing test**

Two cases in `unmarshal_test.go`, copied from `TestUnmarshalPlayJukeboxStep` (`:1649-1660`).

`TestUnmarshalSetBackEffectStep` unmarshals this exact JSON into a `Step[any]`:

```json
{"stepId":"back-effect-step","status":"pending","action":"set_back_effect","payload":{"worldId":0,"channelId":1,"mapId":100000000,"instance":"00000000-0000-0000-0000-000000000000","effect":0,"fieldId":100000000,"pageId":1,"duration":1000}}
```

and asserts `s.Payload` type-asserts to `SetBackEffectPayload` with `MapId == 100000000`, `Effect == 0`, `PageId == 1`, `Duration == 1000`.

`TestUnmarshalClearBackEffectStep` does the same for `"action":"clear_back_effect"` with a payload of only the four field-identity keys, asserting the type assertion to `ClearBackEffectPayload` succeeds and `MapId == 100000000`.

Use the same `stepId`/`status` key names the existing test uses; copy them from `unmarshal_test.go:1649-1660` rather than guessing the envelope's JSON tags.

- [ ] **Step 2: Run the test to verify it fails**

```bash
cd libs/atlas-saga && go test ./... -run BackEffect -v
```

Expected: compile failure — `undefined: SetBackEffectPayload`.

- [ ] **Step 3: Add the constants, payloads and decode arms**

Each `Action` constant carries a comment stating that `Duration` is a **fade length in milliseconds, not a lifetime** — the same distinction `PlayJukebox`'s comment draws about `DurationMs`. The `unmarshal.go` arms are byte-for-byte the `PlayJukebox` arm with the type substituted.

- [ ] **Step 4: Run the tests to verify they pass**

```bash
cd libs/atlas-saga && go test ./... -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-saga
git commit -m "feat(atlas-saga): add set/clear back-effect actions and payloads"
```

---

## Task 11: `atlas-saga-orchestrator` handlers and map commands

### Files

- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/model.go` — action aliases (line 254), payload aliases (line 395), payload-decode arms (line 1719)
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go` — interface methods (line 190), dispatch arms (line 1030), handler bodies (line 3744)
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/event_acceptance.go` — register both actions as self-completing (line 325)
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/map/kafka.go` — the command type constants (const block, line 11) and body structs (end of file)
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/map_command/processor.go` — two interface methods (line 16) and their implementations (line 38)
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/map_command/producer.go` — two command providers (beside `PlayJukeboxCommandProvider`, line 32)
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/map_command/producer_test.go` — new test cases
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler_test.go:1683-1702` — new test cases

Patterns to copy: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go:3744-3771` (`handlePlayJukebox`), `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/map_command/producer.go:32-48` (`PlayJukeboxCommandProvider`), `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/map_command/processor.go:38-40` (`PlayJukebox`), `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler_test.go:1683-1702` (`TestHandlePlayJukebox_InvalidPayload`).

Module root: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`.

**Interfaces consumed:** `sharedsaga.SetBackEffect`, `sharedsaga.ClearBackEffect`, `sharedsaga.SetBackEffectPayload`, `sharedsaga.ClearBackEffectPayload` (Task 10); the `SET_BACK_EFFECT` / `CLEAR_BACK_EFFECT` command contract (Task 6).

Note this service has **no `kafka/message/map/command.go`** — `Command[E]` and every body live in the single `kafka/message/map/kafka.go`.

- [ ] **Step 1: Write the failing tests**

In `map_command/producer_test.go`, beside `TestPlayJukeboxCommandProvider`:

| test | input | expected message value |
|---|---|---|
| `TestSetBackEffectCommandProvider` | `f = field.NewBuilder(0, 1, 100000000).SetInstance(instance).Build()`, `effect 0`, `fieldId 100000000`, `pageId 1`, `duration 1000` | key is `producer.CreateKey(100000000)`; value's `Type == mapKafka.CommandTypeSetBackEffect`; `WorldId 0`, `ChannelId 1`, `MapId 100000000`, `Instance` equal to the input uuid; body `{Effect:0, FieldId:100000000, PageId:1, Duration:1000}` |
| `TestClearBackEffectCommandProvider` | same field | `Type == mapKafka.CommandTypeClearBackEffect`; body is the zero `ClearBackEffectCommandBody{}` |

In `saga/handler_test.go`, beside `TestHandlePlayJukebox_InvalidPayload`:

| test | step | expected |
|---|---|---|
| `TestHandleSetBackEffect_InvalidPayload` | `NewStep[any]("set-back-effect-step", Pending, SetBackEffect, "invalid-payload-type")` | `handleSetBackEffect` returns an error whose message contains `"invalid payload"` |
| `TestHandleClearBackEffect_InvalidPayload` | `NewStep[any]("clear-back-effect-step", Pending, ClearBackEffect, "invalid-payload-type")` | `handleClearBackEffect` returns an error whose message contains `"invalid payload"` |

The happy path is deliberately not covered here — this package has no producer/env-topic fixture, which is why `TestHandlePlayJukebox_InvalidPayload` carries the same note. Message-shape coverage lives in `map_command/producer_test.go`.

- [ ] **Step 2: Run the tests to verify they fail**

```bash
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test ./saga/... ./map_command/... -run BackEffect -v
```

Expected: compile failure — `undefined: SetBackEffect`.

- [ ] **Step 3: Add the aliases, message types, providers and processor methods**

`saga/model.go`: `SetBackEffect = sharedsaga.SetBackEffect` (beside line 254), `SetBackEffectPayload = sharedsaga.SetBackEffectPayload` (beside line 395), and the two payload-decode `case` arms beside line 1719. Same for the clear pair.

`kafka/message/map/kafka.go`: add `CommandTypeSetBackEffect` / `CommandTypeClearBackEffect` and the two body structs, **field-for-field identical to Task 6's atlas-maps definitions**.

`map_command/producer.go` and `processor.go`: two providers and two methods, each a direct copy of the `PlayJukebox` pair with the type substituted.

- [ ] **Step 4: Add the handlers and register them**

`handler.go`: declare `handleSetBackEffect(s Saga, st Step[any]) error` and `handleClearBackEffect(...)` on the interface (beside line 190), add the two `case` arms to the dispatch table (beside line 1030), and write the bodies as direct copies of `handlePlayJukebox` — payload type-assert with `errors.New("invalid payload")`, structured debug log, `field.NewBuilder(...).SetInstance(payload.Instance).Build()`, the `h.mapCommandP` call, `h.logActionError` on failure, and `NewProcessor(h.l, h.ctx).StepCompleted(s.TransactionId(), true)` on success.

`event_acceptance.go`: register `sharedsaga.SetBackEffect: {}` and `sharedsaga.ClearBackEffect: {}` beside line 325, as fire-and-forget self-completing actions — matching `PlayJukebox`, which likewise has no acceptance event to await.

- [ ] **Step 5: Run the tests to verify they pass**

```bash
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go build ./... && go test ./saga/... ./map_command/... -v
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-saga-orchestrator/atlas.com/saga-orchestrator
git commit -m "feat(saga-orchestrator): dispatch set/clear back-effect saga actions"
```

---

## Task 12: `atlas-messages` GM chat commands

### Files

- `services/atlas-messages/atlas.com/messages/command/map/back_effect.go` — new file
- `services/atlas-messages/atlas.com/messages/command/map/back_effect_test.go` — new file
- `services/atlas-messages/atlas.com/messages/kafka/message/map/kafka.go` — add the two command type constants and the two body structs. **This service has no `command.go`**: `EnvCommandTopicMap`, `CommandTypeWeatherStart`, `Command[E]` and `WeatherStartCommandBody` all live in this one file
- `services/atlas-messages/atlas.com/messages/main.go` — register both producers via `command.Registry().Add(...)` beside the weather registration (line 93)
- `services/atlas-messages/atlas.com/messages/command/map/weather.go` — read-only; the producer shape to copy
- `services/atlas-messages/atlas.com/messages/command/map/commands_test.go` — read-only; `createTestCharacter` helper (lines 1-30)

Patterns to copy: `services/atlas-messages/atlas.com/messages/command/map/weather.go:21-49` (`WeatherCommandProducer` — regex match → `c.Gm()` gate → returned `command.Executor` closure), `services/atlas-messages/atlas.com/messages/command/map/weather.go:51-66` (`weatherStartCommandProvider`), `services/atlas-messages/atlas.com/messages/command/map/commands_test.go:1-30` (`createTestCharacter(t, id, name, isGm, mapId)`).

Module root: `services/atlas-messages/atlas.com/messages`.

**Command shapes (design §5.2):**

```
@backeffect <pageId> <effect> [durationMs]    # effect: 0 show, 1 hide; duration defaults to 0
@clearbackeffect
```

`fieldId` is filled from the invoking character's own field (`f.MapId()`), **not** taken as an argument: the client never reads it (design §1.1), so a GM-typed parameter would be surface area with no observable effect.

**Interfaces produced:**

```go
func BackEffectCommandProducer(_ logrus.FieldLogger) func(_ context.Context) func(f field.Model, c character.Model, m string) (command.Executor, bool)
func ClearBackEffectCommandProducer(_ logrus.FieldLogger) func(_ context.Context) func(f field.Model, c character.Model, m string) (command.Executor, bool)
```

- [ ] **Step 1: Write the failing test**

`back_effect_test.go`, table-driven, characters built with `createTestCharacter` from `commands_test.go:1-30`.

`TestBackEffectCommandProducer_Matching`:

| case | gm | message | expect matched |
|---|---|---|---|
| set with duration | true | `@backeffect 1 0 1000` | true |
| set without duration | true | `@backeffect 1 0` | true |
| hide | true | `@backeffect 2 1` | true |
| non-gm rejected | false | `@backeffect 1 0` | false |
| effect out of range | true | `@backeffect 1 2` | false |
| missing effect | true | `@backeffect 1` | false |
| unrelated message | true | `@weather 5120016 hi` | false |

`TestClearBackEffectCommandProducer_Matching`:

| case | gm | message | expect matched |
|---|---|---|---|
| gm clear | true | `@clearbackeffect` | true |
| non-gm rejected | false | `@clearbackeffect` | false |
| unrelated message | true | `@backeffect 1 0` | false |

Assert only on the returned `bool` (whether a `command.Executor` was produced). Do not invoke the executor: it produces onto Kafka and there is no broker in this test binary — the existing `TestWarpCommandProducer_RegexPatterns` is regex-only for the same reason. Command message shape is covered by Task 11's `map_command/producer_test.go`.

- [ ] **Step 2: Run the test to verify it fails**

```bash
cd services/atlas-messages/atlas.com/messages && go test ./command/map/... -run BackEffect -v
```

Expected: compile failure — `undefined: BackEffectCommandProducer`.

- [ ] **Step 3: Add the message types**

Add `CommandTypeSetBackEffect` / `CommandTypeClearBackEffect` and `SetBackEffectCommandBody` / `ClearBackEffectCommandBody` to `services/atlas-messages/atlas.com/messages/kafka/message/map/kafka.go`, **field-for-field identical to Task 6's atlas-maps definitions**.

- [ ] **Step 4: Write `back_effect.go`**

Copy `weather.go` twice into one file. Regexes:

- `regexp.MustCompile(`^@backeffect\s+(\d+)\s+([01])(?:\s+(\d+))?$`)` — group 1 `pageId`, group 2 `effect`, optional group 3 `durationMs`. `[01]` in the pattern is what makes the out-of-range case fail to match rather than needing a separate check. Parse `pageId` with `strconv.ParseUint(match[1], 10, 8)` and return `nil, false` on error; parse `durationMs` with `strconv.ParseUint(match[3], 10, 32)` only when `match[3] != ""`, defaulting to `0`.
- `regexp.MustCompile(`^@clearbackeffect$`)`.

Both gate on `c.Gm()` after the regex match, exactly as `weather.go:31` does. Both build their `mapKafka.Command[...]` inline in a `setBackEffectCommandProvider` / `clearBackEffectCommandProvider` copied from `weatherStartCommandProvider`, with `TransactionId: uuid.New()` and `FieldId: uint32(f.MapId())`.

- [ ] **Step 5: Register the producers**

In `services/atlas-messages/atlas.com/messages/main.go`, add `command.Registry().Add(_map.BackEffectCommandProducer)` and `command.Registry().Add(_map.ClearBackEffectCommandProducer)` beside the `WeatherCommandProducer` registration at line 93.

- [ ] **Step 6: Run the tests to verify they pass**

```bash
cd services/atlas-messages/atlas.com/messages && go build ./... && go test ./command/map/... -v
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-messages/atlas.com/messages
git commit -m "feat(atlas-messages): add @backeffect and @clearbackeffect GM commands"
```

---

## Task 13: Full verification gate

### Files

- `tools/verify.sh` — read-only; the gate to run
- `docs/tasks/task-281-map-back-effects/follow-ups.md` — new file; record `SET_MAP_OBJECT_VISIBLE` as the next-cheapest adjacent op

Module root: repo root.

- [ ] **Step 1: Record the follow-up**

Write `docs/tasks/task-281-map-back-effects/follow-ups.md` naming `SET_MAP_OBJECT_VISIBLE` (case 145 in the same `CMapLoadable::OnPacket` switch, ❌ on the same eight versions, `docs/packets/audits/STATUS.md:211`) as explicitly out of this task's scope (design §2.4) and cheap to do next because this task already decompiled the switch. Cite the per-version router addresses recorded in `docs/tasks/task-281-map-back-effects/structures/`.

- [ ] **Step 2: Run the packet gates**

```bash
go run ./tools/packet-audit matrix --check
go run ./tools/packet-audit fname-doc --check
go run ./tools/packet-audit operations --check
```

Expected: each exits 0. If `matrix --check` reports an n-a consistency problem, the `back_effect` family added in Task 4 is missing a `feature-na-evidence.yaml` entry — add it from the Task 2 structures record, never from memory.

- [ ] **Step 3: Run the flagless verification gate**

```bash
tools/verify.sh
```

Expected: exit 0. Per CLAUDE.md, `--quick` / `--no-docker` do **not** count for "done": they skip the bake and `-race`.

Dispatch this via `task-verifier` in its own context rather than running it inline — the build/vet/lint output should not land in an implementer's window.

- [ ] **Step 4: Trace the cross-service seam by hand**

Per CLAUDE.md's "Done means verified": follow `BACK_EFFECT_SET` from `services/atlas-maps/atlas.com/maps/map/backeffect/producer.go` into `services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go`, and confirm the two `kafka/message/map/kafka.go` files declare the constant and body struct identically:

```bash
grep -n 'BACK_EFFECT_SET\|BACK_EFFECT_CLEAR\|type BackEffectSet\|type BackEffectClear' \
  services/atlas-maps/atlas.com/maps/kafka/message/map/kafka.go \
  services/atlas-channel/atlas.com/channel/kafka/message/map/kafka.go
```

Expected: the same four declarations in both files. Task 8's `decodeSetBackEffect` assertion is the test that pins the new contract.

- [ ] **Step 5: Commit**

```bash
git add docs/tasks/task-281-map-back-effects/follow-ups.md
git commit -m "docs(task-281): record SET_MAP_OBJECT_VISIBLE follow-up"
```

- [ ] **Step 6: Code review, then PR**

Run code review before opening the PR — a green `tools/verify.sh` cannot see a cross-service seam defect. Dispatch `backend-guidelines-reviewer` over the changed Go packages and `packet-completeness-critic` for the packet-layer scope, per `docs/review-protocol.md`.

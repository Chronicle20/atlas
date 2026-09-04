# MobEscortFullPath Wire Model Correction — Design

Task: `task-295-mob-escort-full-path-fix`
PRD: [`prd.md`](prd.md)
Status: Draft
Created: 2026-09-04

---

## 1. Summary

The corrected wire layout of `MOB_ESCORT_FULL_PATH` is three leading `int32`s —
`count`, `m_Old_Dest.dp.x`, `m_Old_Dest.dp.y` — then the waypoint array, then
`m_nCurrentDestIndex`, then the two stop flags. The current codec's `mode` field is a
mis-derivation of `count` and is removed; the two fields at escort-struct offsets
`+1924` / `+1928` are added. The layout is byte-identical across `gms_v92`,
`gms_v95`, and `jms_v185`, so **no version gate is introduced**.

The `gms_v95` IDB carries full type information for the receiving structure —
`CVecCtrlMob` (1960 bytes, 80 members) and `CVecCtrlMob::EscortDest` (20 bytes) —
so every field in this packet gets a name taken directly from the client's own
member names rather than from behavioural inference. §9 (Open Questions) of the PRD
is fully closed by that: no field is named speculatively, and nothing needs to be
raised.

---

## 2. Live derivation (FR-1)

Read live from the IDBs on 2026-09-04 via `ida-pro-mcp`, not from the checked-in
exports:

| Version | IDB | Function | Address |
|---|---|---|---|
| `gms_v92` | `GMS_v92_1_DEVM.exe.i64` | `CMob::OnEscortFullPath` | `0x6374c0` |
| `gms_v95` | `GMS_v95.0_U_DEVM.exe.i64` | `CMob::OnEscortFullPath` | `0x643d90` |
| `jms_v185` | `MapleStory_dump_SCY.exe.i64` | `CMob::OnEscortFullPath` | `0x6efa01` |

### 2.1 The receiving structure

All three handlers write into `this->m_pvcActive - 12`, which the `gms_v95` IDB
types as `CVecCtrlMob *`. Its relevant members (`type_inspect CVecCtrlMob`):

| Offset | Member | Type |
|---|---|---|
| `0x75C` (1884) | `m_nStopDuration` | `int` |
| `0x760` (1888) | `m_bEscortStop` | `int` |
| `0x780` (1920) | `m_aEscorDest` | `ZArray<CVecCtrlMob::EscortDest>` |
| `0x784` (1924) | `m_Old_Dest` | `CVecCtrlMob::EscortDest` |
| `0x798` (1944) | `m_nCurrentDestIndex` | `int` |
| `0x79C` (1948) | `m_nCurrentDestIndexForce` | `int` |
| `0x7A0` (1952) | `m_bMoveMobBeforeEscortCurDest` | `int` |

`CVecCtrlMob::EscortDest` (20 bytes, `type_inspect`):

| Offset | Member | Type |
|---|---|---|
| `+0x0` | `dp` | `tagPOINT` (`dp.x` `+0x0`, `dp.y` `+0x4`) |
| `+0x8` | `nAttr` | `int` |
| `+0xC` | `ZMass` | `int` |
| `+0x10` | `nStopDuration` | `int` |

This resolves the PRD's `+1924` / `+1928` question exactly: they are
`m_Old_Dest.dp.x` and `m_Old_Dest.dp.y` — the x/y of the escort's *previous*
destination record. The client also zeroes `m_Old_Dest.nAttr` (`+1932`) and fills
`m_Old_Dest.ZMass` (`+1936`) from `CMob::GetZMass()`; neither comes from the wire.

### 2.2 Read order, all three versions

```
Decode4  -> count                    (loop bound; ZArray<EscortDest>::_Alloc(count))
Decode4  -> m_Old_Dest.dp.x
Decode4  -> m_Old_Dest.dp.y
         (m_Old_Dest.nAttr = 0;  m_Old_Dest.ZMass = this->GetZMass();  not wire)
         ZArray<EscortDest>::RemoveAll + _Alloc(count)
count ×  Decode4 -> dest[i].dp.x
         Decode4 -> dest[i].dp.y
         Decode4 -> dest[i].nAttr
         if nAttr == 1: dest[i].ZMass = 0
         else:          dest[i].ZMass = GetFootholdUnderneath(dp.x, dp.y).m_lZMass
                                        (fallback GetZMass())
                        if nAttr == 2: Decode4 -> dest[i].nStopDuration
Decode4  -> m_nCurrentDestIndex
Decode1  -> if set: Decode4 -> d;  m_nStopDuration = d + get_update_time();
                                   m_bEscortStop   = 1
Decode1  -> if set:                m_nStopDuration = 0;
                                   m_bEscortStop   = 1
```

Per-version confirmation of identity:

- `gms_v92` @ `0x6374c0` — offsets appear as `+1924` / `+1928` / `+1944` on the
  `m_pvcActive - 12` base (instruction `mov [esi+784h], eax` at
  `CMob::OnEscortFullPath+6D`). No named types in this IDB; offsets match v95's
  layout exactly.
- `gms_v95` @ `0x643d90` — decompiles with symbols:
  `ZArray<CVecCtrlMob::EscortDest>::_Alloc(..., v4, ...)` where `v4` is the first
  `Decode4`, `CWvsPhysicalSpace2D::GetFootholdUnderneath`, `get_update_time()`.
- `jms_v185` @ `0x6efa01` — same order at shifted offsets: array `+0x760`,
  old-dest `+0x764`/`+0x768`, `nAttr` slot `+0x76C`, ZMass slot `+0x770`,
  current index `+0x778`, stop duration `+1852`, stop flag `+1856`. The struct
  is laid out differently; **the wire order is identical**.

**Conclusion (FR-10):** no divergence among the three columns → no `MajorAtLeast`
gate. Adding one would be a defect.

### 2.3 What the fields are for (consumer evidence, `gms_v92`)

- `sub_975B10` (mob controller, escort reset path) refills the `m_Old_Dest` mirror
  from the array — `this[481] = wp[0]; this[482] = wp[1]; this[483] = wp[2];
  this[484] = wp[3]; this[485] = wp[4]` — and then drives the mob's position from
  it: `pos.x = (double)m_Old_Dest.dp.x`, `pos.y = (double)m_Old_Dest.dp.y - 10.0`.
  This is what proves `+1924`/`+1928` are an x/y point and not two scalars.
- `sub_976060` (per-tick escort update) indexes the array with
  `m_nCurrentDestIndex`: `*(array + 20 * this[486] + 8) == 1` (i.e.
  `dest[idx].nAttr`), and short-circuits on
  `m_bEscortStop || (m_nStopDuration && m_nStopDuration > get_update_time())`.
  When `m_nStopDuration <= now` it calls `CMob::SendEscortStopEndRequest` and
  clears it. This is what proves the post-loop `int32` is an **index**, and that
  the two trailing booleans are a *timed stop* and an *indefinite stop*.
- `sub_96FAD0` reads `m_Old_Dest.dp.x` (`fild [esi+784h]`) as the walk target when
  `m_Old_Dest.nAttr == 1`.

### 2.4 FR-3 — does `mode` survive?

No. The first wire `int32` is `count`: it is the `_Alloc` argument and the `do/while`
bound in all three functions. The codec's `mode` field never corresponded to a wire
field; it is deleted, not renamed.

### 2.5 FR-4 / FR-5 — tail and loop body

Re-confirmed live, not assumed. The post-loop section is structurally unchanged
(`Decode4`, `Decode1` [+`Decode4`], `Decode1`) and the loop body is structurally
unchanged (`x`, `y`, `nAttr`, plus one `Decode4` only when `nAttr == 2`). Only the
*names* change (§3).

---

## 3. Naming decision

The PRD (FR-2) requires names derived from observed behaviour and forbids
placeholders on exported accessors. The v95 IDB hands us the client's own member
names, so this design takes them verbatim wherever they exist. That means renaming
three fields the PRD's §5 signature sketch left alone (`tail`, and the waypoint's
`kind`/`extra`), plus the two trailing booleans.

| Current | New | Basis |
|---|---|---|
| `mode` | *(removed)* | no wire counterpart (§2.4) |
| — | `oldDestX` | `CVecCtrlMob::m_Old_Dest.dp.x` |
| — | `oldDestY` | `CVecCtrlMob::m_Old_Dest.dp.y` |
| `tail` | `currentDestIndex` | `CVecCtrlMob::m_nCurrentDestIndex` |
| `hasArrive` | `hasStopDuration` | gates the value written to `m_nStopDuration` |
| `arriveDelay` | `stopDuration` | `m_nStopDuration = value + get_update_time()` |
| `hasReset` | `stopIndefinitely` | sets `m_nStopDuration = 0`, `m_bEscortStop = 1` — the mob never auto-resumes |
| `MobEscortWaypoint.kind` | `attr` | `CVecCtrlMob::EscortDest::nAttr` |
| `MobEscortWaypoint.extra` | `stopDuration` | `CVecCtrlMob::EscortDest::nStopDuration` |
| `MobEscortWaypoint.x`/`y` | unchanged | `EscortDest::dp.x` / `dp.y` |

**Deviation from PRD §5, stated explicitly.** The PRD's illustrative Go signature
keeps `tail`, `hasArrive`, `arriveDelay`, `hasReset`, and does not mention the
waypoint accessors. This design renames them. Rationale: PRD §5 says the leading
parameter list "is fixed by the FR-1/FR-2 derivation; the design doc names them,"
and FR-2 forbids placeholder names on exported accessors — `tail`, `kind`, and
`extra` are exactly that, and `hasArrive`/`arriveDelay` are affirmatively wrong
(nothing about arrival; it is an escort *stop* timer). The cost is zero: the only
non-test consumers are `MobEscortFullPathBody` (no callers) and `main.go`'s
reference to the `MobEscortFullPathWriter` string constant, which does not change.

*Alternative considered and rejected:* rename only `mode`/`tail` and leave
`kind`/`extra`/`hasArrive` alone, to keep the diff inside PRD §5's letter. Rejected
because it deliberately leaves derived-and-known-wrong names on exported accessors
in the one file this task exists to correct, and a follow-up rename task would have
to re-derive the same evidence.

---

## 4. Codec design

`libs/atlas-packet/monster/clientbound/mob_escort_full_path.go`:

```go
type MobEscortWaypoint struct {
    x            int32
    y            int32
    attr         int32
    stopDuration int32
}

func NewMobEscortWaypoint(x, y, attr, stopDuration int32) MobEscortWaypoint

func (w MobEscortWaypoint) X() int32            { return w.x }
func (w MobEscortWaypoint) Y() int32            { return w.y }
func (w MobEscortWaypoint) Attr() int32         { return w.attr }
func (w MobEscortWaypoint) StopDuration() int32 { return w.stopDuration }

type MobEscortFullPath struct {
    oldDestX         int32
    oldDestY         int32
    waypoints        []MobEscortWaypoint
    currentDestIndex int32
    hasStopDuration  bool
    stopDuration     int32
    stopIndefinitely bool
}

func NewMobEscortFullPath(
    oldDestX int32,
    oldDestY int32,
    waypoints []MobEscortWaypoint,
    currentDestIndex int32,
    hasStopDuration bool,
    stopDuration int32,
    stopIndefinitely bool,
) MobEscortFullPath
```

Encode order (FR-6, FR-7, FR-8):

```
WriteInt32(int32(len(m.waypoints)))   // count — derived, never stored
WriteInt32(m.oldDestX)
WriteInt32(m.oldDestY)
for _, wp := range m.waypoints {
    WriteInt32(wp.x); WriteInt32(wp.y); WriteInt32(wp.attr)
    if wp.attr == 2 { WriteInt32(wp.stopDuration) }
}
WriteInt32(m.currentDestIndex)
WriteBool(m.hasStopDuration)
if m.hasStopDuration { WriteInt32(m.stopDuration) }
WriteBool(m.stopIndefinitely)
```

`Decode` mirrors it: read `count` into a local, then `oldDestX`, `oldDestY`, then
the loop, then the tail. The struct stays immutable with value-receiver accessors,
matching the existing file. `String()` is updated to the new field set.

The struct doc comment (FR-9) is rewritten to the layout in §2.2, cites all three
addresses, names the receiving `CVecCtrlMob` / `EscortDest` members, and states
"v92 + v95 + jms; escort family absent in v83/v84/v87." The stale
`8×Decode4 + Decode1 + Decode4 + Decode1` paragraph is replaced: with the corrected
shape a 2-waypoint, all-`attr!=2` example is `3 + 2×3 = 9 × Decode4`, then
`Decode1 + Decode4 + Decode1` — i.e. **9×Decode4**, not 8.

---

## 5. Fixture design

One test, three markers (FR-12). The byte layout is identical across the three
versions, so a second golden would assert the same bytes twice; the marker list is
what the matrix keys on, not the test count.

```go
// packet-audit:verify packet=monster/clientbound/MonsterMobEscortFullPath version=gms_v92 ida=0x6374c0
// packet-audit:verify packet=monster/clientbound/MonsterMobEscortFullPath version=gms_v95 ida=0x643d90
// packet-audit:verify packet=monster/clientbound/MonsterMobEscortFullPath version=jms_v185 ida=0x6efa01
func TestMobEscortFullPath(t *testing.T)
```

The golden input changes from "two waypoints, both `kind=1`" to **two waypoints,
`attr=1` and `attr=2`**, so the fixture exercises the conditional
`dest[i].nStopDuration` field — the current golden never encodes it, which is how a
whole conditional branch stayed unasserted. Per-line comments are rewritten to the
new order. The `pt.Variants` round-trip assertion is preserved unchanged (FR-15).

Expected golden (encode of `oldDest=(0x0A,0x14)`, wp0 `(100,200,attr=1)`,
wp1 `(250,300,attr=2,stopDuration=0x2BC)`, `currentDestIndex=1`,
`hasStopDuration=true`, `stopDuration=0x320`, `stopIndefinitely=false`):

```
02 00 00 00   count = 2
0A 00 00 00   oldDestX
14 00 00 00   oldDestY
64 00 00 00   wp0.x
C8 00 00 00   wp0.y
01 00 00 00   wp0.attr = 1
FA 00 00 00   wp1.x
2C 01 00 00   wp1.y
02 00 00 00   wp1.attr = 2
BC 02 00 00   wp1.stopDuration       (present only because attr == 2)
01 00 00 00   currentDestIndex = 1
01            hasStopDuration = true
20 03 00 00   stopDuration = 800
00            stopIndefinitely = false
```

Exact literals are the implementer's to fix; the shape above is the contract.

---

## 6. Evidence design (FR-13, FR-14)

All three records are written by the tool, never hand-edited:

```sh
go run ./tools/packet-audit evidence pin \
  --packet monster/clientbound/MonsterMobEscortFullPath \
  --version <gms_v92|gms_v95|jms_v185> \
  --ida "CMob::OnEscortFullPath" \
  --category TIER1-FIXTURE \
  --verifies libs/atlas-packet/monster/clientbound/mob_escort_full_path_test.go#TestMobEscortFullPath
```

`--verifies` must be passed on every re-pin: `cmd/evidence.go` rebuilds the record
from flags, so omitting it silently drops the `verifies:` link on the two existing
records.

**Finding that qualifies FR-13's wording.** `decompile_sha256` is computed by
`evidence.FunctionHash(exportPath, fname)` from the checked-in export JSON
(`docs/packets/ida-exports/<version>.json`), not from a live decompile. Since this
task changes no export, re-pinning `gms_v95` and `jms_v185` will reproduce the
*same* hash values (`fc79aaa5…` and `fb9623c9…`). That is correct behaviour, not a
carried-forward stale hash: the citation pins the export the model was derived
against, and that export is unchanged and independently confirmed live in §2. The
PRD's "regenerated from the current decompile" is satisfied by re-running the tool;
the design records here that an unchanged hash is the expected outcome so a reviewer
does not read it as a skipped step.

Export entries confirmed present for all three versions:
`gms_v92.json → 0x6374c0`, `gms_v95.json → 0x643d90`,
`gms_jms_185.json → 0x6efa01`.

---

## 7. Routing and matrix promotion

**FR-16 is already satisfied — no edit needed.** `docs/packets/registry/gms_v92.yaml`
already carries the row at line 1582:

```yaml
- op: MOB_ESCORT_FULL_PATH
  direction: clientbound
  opcode: 296          # 0x128
  fname: CMob::OnEscortFullPath
  provenance: csv-import
```

`status.json` confirms the registry side is live: the `gms_v92` cell is
`{"state": "incomplete", "note": "tier-1 without fixture; verdict 🔍", "opcode": 296}`
— routed in the registry, missing the fixture.

**FR-17 is the real gap.** `template_gms_92_1.json` has no `MobEscortFullPath`
writer row. A writers-array entry is added mirroring the v95/jms shape:

```json
{
  "opCode": "0x128",
  "writer": "MobEscortFullPath",
  "fname": "CMob::OnEscortFullPath",
  "services": ["channel"]
}
```

This is load-bearing, not cosmetic: `matrix --check` raises a
**template-wiring-gap conflict** (`cmd/matrix_test.go:127-197`) when an op is routed
in one version's template and reported-but-unrouted in another. Adding the marker
without the template row would turn the `gms_v92` cell into a conflict rather than
promoting it.

Then `go run ./tools/packet-audit matrix` regenerates `STATUS.md` / `status.json`.
Expected delta, and nothing else (FR-18, FR-19):

- `MOB_ESCORT_FULL_PATH` row, `gms_v92` column: ❌ → ✅
- the `gms_v92` summary-row totals

Verification of FR-19 is mechanical: `git diff docs/packets/audits/STATUS.md` must
show only those lines.

---

## 8. Consumer update (FR-20, FR-21)

`services/atlas-channel/.../socket/writer/mob_escort_full_path.go`:

```go
func MobEscortFullPathBody(
    oldDestX int32,
    oldDestY int32,
    waypoints []monsterpkt.MobEscortWaypoint,
    currentDestIndex int32,
    hasStopDuration bool,
    stopDuration int32,
    stopIndefinitely bool,
) packet.Encode
```

No shim, no deprecated overload — confirmed by grep that the only references to the
symbol are its own definition and the test. `main.go:758` references the
`MobEscortFullPathWriter` *string constant*, which is unchanged. The doc comment is
updated to the corrected field list and to record that the op is routed at
`gms_v92` (`0x128`), `gms_v95` (`0x130`), and `jms_v185` (`0x110`) after this
change. It remains an intentional unwired seam — no emitter is added (PRD §2
non-goal).

---

## 9. Work order

1. Codec: struct, constructor, accessors, `String`, `Encode`, `Decode`, doc comment.
2. Test: new golden (with an `attr == 2` waypoint), rewritten comments, three
   `packet-audit:verify` markers. `go test ./libs/atlas-packet/...`.
3. Writer: signature + doc comment. `go build ./services/atlas-channel/...`.
4. Template: `template_gms_92_1.json` writers row at `0x128`.
5. Evidence: three `evidence pin` invocations (§6).
6. `go run ./tools/packet-audit matrix`; diff `STATUS.md` for the single-cell delta;
   `go run ./tools/packet-audit matrix --check` → exit 0; `git status` clean after.
7. `tools/verify.sh` (flagless) → exit 0.
8. Code review before PR.

Steps 1–3 are one logical unit (the rename is atomic across codec + test + writer);
4–6 are the matrix unit. They are not independently committable in a compiling
state, so this is a two-commit branch, not a seven-commit one.

---

## 10. Risks

| Risk | Mitigation |
|---|---|
| Matrix regeneration picks up unrelated drift from other in-flight work | Diff `STATUS.md` before committing; FR-19 makes any extra cell movement a blocker to investigate, not to absorb. |
| Reviewer reads the renames as scope creep past PRD §5 | §3 states the deviation, the FR-2 basis, and the rejected minimal alternative up front. |
| Unchanged `decompile_sha256` on re-pin read as a skipped regeneration | §6 records the tool's actual hash source and states the expected values in advance. |
| Adding the marker before the template row | Work order puts the template row (step 4) before the matrix run (step 6); the conflict rule in `cmd/matrix_test.go` makes the failure loud, not silent. |
| A future escort emitter passes arguments positionally in the old order | Seven same-typed leading parameters is a real hazard, but the PRD forbids adding an emitter here. Noted for the emitter task; not addressed now. |

---

## 11. Out of scope (restated from PRD §2)

No escort emitter. No other escort op (`MobEscortStop`, `MobEscortStopSay`,
`MobEscortReturnBefore`) and no other `CMob` cell withheld by task-270. No
`gms_v83`/`gms_v84`/`gms_v87` addition — those cells stay ⬜. No behavioural change
in `atlas-channel` beyond the writer's parameter list.

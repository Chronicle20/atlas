# TOUCHING_REACTOR — opcode derivation (Task 1)

All addresses and opcode integers below are read directly from IDA decompile
or instruction-level output against the ten IDBs listed in
`.superpowers/sdd/plan/task-1-brief.md`. Session ids were re-confirmed via
`mcp__ida-pro__idb_list` at the start of this task and matched the plan-time
table exactly (no restart drift).

Wire layout (unchanged across every in-scope version, per design §1.1):

| offset | size | field      | meaning                                  |
|--------|------|------------|-------------------------------------------|
| 0      | 4    | `oid`      | reactor object id (`REACTOR::dwID`)      |
| 4      | 1    | `touching` | `1` on entering the touch area, `0` on leaving |

## Per-version table

| version | function address | symbol | COutPacket int (pseudocode) | field sequence | registry value | agreement |
|---|---|---|---|---|---|---|
| gms_v48 | — | not present | — | — | — (not in registry) | **n-a**, confirmed |
| gms_v61 | — | not present | — | — | — (not in registry) | **n-a**, confirmed |
| gms_v72 | `0x692bb0` | pre-existing `CReactorPool::FindTouchReactorAroundLocalUser` | `196` (enter and leave arms both) | `COutPacket(196); Encode4(dwID); Encode1(1\|0)` | not in registry (no gms_v72.yaml entry) | n/a — registry silent, binary is authoritative |
| gms_v79 | `0x6b8362` | pre-existing `CReactorPool::FindTouchReactorAroundLocalUser` | `198` (enter and leave arms both) | `COutPacket(198); Encode4(dwID); Encode1(1\|0)` | not in registry (no gms_v79.yaml entry) | n/a — registry silent, binary is authoritative |
| gms_v83 | `0x735D90` | **named by this task** (was `sub_735D90`) | `206` / `0xCE` (enter and leave arms both) | `COutPacket(0xCE); Encode4(dwID); Encode1(1\|0)` | `206` (`docs/packets/registry/gms_v83.yaml:3176`) | **agree** |
| gms_v84 | `0x753378` | **named by this task** (was `sub_753378`) | `212` / `0xD4` (enter and leave arms both) | `COutPacket(212); Encode4(dwID); Encode1(1\|0)` | `206` (`docs/packets/registry/gms_v84.yaml:3936`, csv-import, seeded from v83) | **disagree — registry is wrong**; measured value is 212 |
| gms_v87 | `0x77bca7` | pre-existing `CReactorPool::FindTouchReactorAroundLocalUser` | `219` / `0xDB` (enter and leave arms both) | `push 1/push ebx; push 0xDB; call COutPacket::ctor; ... call Encode4; ... call Encode1` (call-site instructions; body too large to decompile cleanly) | not checked against a registry cell in this task (design §1.2 records it as confirmed) | **agree** with design §1.2 |
| gms_v92 | `0x6C1630` | **named by this task** (was `sub_6C1630`); the design-brief candidate addresses `0x79f903`/`0x79f9f4` are **false positives** (unrelated GUI resource-registration code that happens to push the literal `0xF3` as an argument, not an opcode) | `243` / `0xF3` (enter and leave arms both) | `COutPacket(0xF3); Encode4(dwID); Encode1(1\|0)` | `243` (`docs/packets/registry/gms_v92.yaml:3649`) | **agree** (value agrees; the design brief's suspected address did not) |
| gms_v95 | `0x6cded0` | pre-existing `CReactorPool::FindTouchReactorAroundLocalUser` | `250` / `0xFA` (enter and leave arms both) | `COutPacket(250); Encode4(dwID); Encode1(1\|0)` | not checked against a registry cell in this task (design §1.1 primary derivation) | **agree** with design §1.1 |
| jms_v185 | `0x79f0aa` | pre-existing `CReactorPool::FindTouchReactorAroundLocalUser` | `217` / `0xD9` (enter and leave arms both) | `push 1/push ebx; push 0xD9; call COutPacket::ctor; ... call Encode4; ... call Encode1` (call-site instructions; body too large to decompile cleanly) | not checked against a registry cell in this task | **agree** with design §1.2 |

All eight in-scope versions match the client's edge-triggered `(uint32 oid,
byte touching)` two-field body exactly — no version deviates from the
`COutPacket(N); Encode4(dwID); Encode1(touching)` shape. No smoothing was
needed.

## Newly-established addresses — quoted decompile

### gms_v83 — `CReactorPool::FindTouchReactorAroundLocalUser` @ `0x735D90`

Function was already `define_func`-bounded (visible as `sub_735D90`, ending
before the enter/leave block at `0x735fb9`/`0x736021`) but unnamed. Renamed by
this task; `idb_save` completed.

```
COutPacket::COutPacket(&v23, 0xCE);      /*0x735fc3*/
COutPacket::Encode4(&v23, v16);          /*0x735fd4*/   v16 = *v30 (dwID)
COutPacket::Encode1(&v23, 1u);           /*0x735fdd*/
CClientSocket::SendPacket(..., &v23);    /*0x735fec*/
```
```
COutPacket::COutPacket(&v21, 206);       /*0x736029*/
COutPacket::Encode4(&v21, v17);          /*0x73603a*/   v17 = *v30 (dwID)
COutPacket::Encode1(&v21, 0);            /*0x736043*/
CClientSocket::SendPacket(..., &v21);    /*0x736052*/
```

`0xCE` and `206` are the same integer (206 decimal); the enter arm's
constructor literal is printed in hex by the decompiler, the leave arm's in
decimal — both read from the same compiled function, confirming internal
self-consistency as well as agreement with the registry.

### gms_v84 — `CReactorPool::FindTouchReactorAroundLocalUser` @ `0x753378`

Found by testing the `+6` shift hypothesis first: `find_bytes "68 D4 00 00 00"`
(212 = `0xD4`) hit `0x7535ab` and `0x753613`, immediately after
`FindHitReactor`'s end (`0x7530ac`, per design §1.4) — the expected enter/leave
pair. The general-signature sweep (`6A 01 68 ?? ?? 00 00`) was not needed; the
targeted hypothesis test succeeded on the first try. Function was already
`define_func`-bounded (`sub_753378`) but unnamed. Renamed by this task;
`idb_save` completed.

```
COutPacket::COutPacket((COutPacket *)v23, 212);   /*0x7535b5*/
COutPacket::Encode4((COutPacket *)v23, v16);      /*0x7535c6*/   v16 = *v36 (dwID)
COutPacket::Encode1((COutPacket *)v23, 1u);       /*0x7535cf*/
CClientSocket::SendPacket(...);                   /*0x7535de*/
```
```
COutPacket::COutPacket((COutPacket *)v21, 212);   /*0x75361b*/
COutPacket::Encode4((COutPacket *)v21, v17);      /*0x75362c*/   v17 = *v36 (dwID)
COutPacket::Encode1((COutPacket *)v21, 0);        /*0x753635*/
CClientSocket::SendPacket(...);                   /*0x753644*/
```

**This overturns the registry.** `docs/packets/registry/gms_v84.yaml:3936`
carries `opcode: 206`, explicitly annotated `"seeded from the v83 CSV column
— the CSVs have no v84 column; task-083 found v84 byte-identical to v83."`
That seed is wrong for this op: the binary shows **212**, a `+6` shift
consistent with `DAMAGE_REACTOR`'s v83→v84 shift (205→211). Task 3 must write
212 (`0xD4`), not the registry's 206, into the v84 template/registry.

### gms_v92 — `CReactorPool::FindTouchReactorAroundLocalUser` @ `0x6C1630`

The design brief's suspected addresses (`0x79f903`, `0x79f9f4`, both inside
`sub_79F590`) were checked first and are **false positives**: that function is
an unrelated GUI resource-registration routine that happens to push the
literal `243` (`0xF3`) as a positional argument (alongside `0x194`, `0x7DE`,
etc.) to an unrelated vtable call — not a `COutPacket` opcode. No `COutPacket`
constructor, `Encode4`, or `Encode1` call appears anywhere near either address.

The real send site was found by re-running `find_bytes "68 F3 00 00 00"`
(the constructor-literal signature alone, without the `6A 01` prefix) across
the whole binary and checking which hits fell inside a function with the
`ZMap`-gated enter/leave shape. `0x6c1949`/`0x6c19ae` (inside `sub_6C1630`,
already `define_func`-bounded, ends `0x6c1a65`) matched. Renamed by this task;
`idb_save` completed.

```
COutPacket::COutPacket((COutPacket *)&v29, 0xF3u);  /*0x6c1959*/
COutPacket::Encode4(&v29, v17);                     /*0x6c196d*/   v17 = *v4 (dwID)
COutPacket::Encode1(&v29, 1);                       /*0x6c1978*/
CClientSocket::SendPacket(...);                     /*0x6c1988*/
```
```
COutPacket::COutPacket((COutPacket *)&v31, 0xF3u);  /*0x6c19b7*/
COutPacket::Encode4(&v31, v18);                     /*0x6c19cb*/   v18 = *v4 (dwID)
COutPacket::Encode1(&v31, 0);                       /*0x6c19d5*/
CClientSocket::SendPacket(...);                     /*0x6c19e5*/
```

`0xF3` = 243, matching `docs/packets/registry/gms_v92.yaml:3649`. The registry
value is correct; only the design brief's candidate *address* was wrong.

## gms_v48 / gms_v61 — re-verified `n-a`

Design §1.3 measured both as genuinely absent via a name-scoped `func_query`
for `CReactorPool` (six symbols, none the touch function). Per
`VERIFYING_A_PACKET.md`'s "Is this cell `n-a`?" bar, a name search alone is
insufficient. Two independent checks were added by this task:

**1. Opcode-construction invariant sweep.** `find_bytes "6A 01 68 ?? ?? 00 00"`
(the general "push 1; push <opcode>" send-site signature) was run across each
whole binary and cross-checked against the `CReactorPool` cluster address
range:

- gms_v48: cluster spans `0x5a5390` (`OnPacket`) through `0x5a5eb4`
  (`LoadReactorLayer` end, `FindHitReactor` at `0x5a5a32`–`0x5a5d97` inside
  it). 160 whole-binary hits; **none** fall inside or adjacent to
  `[0x5a5390, 0x5a5eb4]` (nearest hits are `0x589cfc` below and `0x5aff0c`
  above — both outside).
- gms_v61: cluster spans `0x633133` (`OnPacket`) through `0x633d3f`
  (`GetAt` end, `FindHitReactor` at `0x6337df`–`0x633b44` inside it). 186
  whole-binary hits; **none** fall inside or adjacent to
  `[0x633133, 0x633d3f]` (nearest hits are `0x62959a` below and `0x6345eb`
  above — both outside).

**2. Mandatory sibling cross-check.** Decompiled `CReactorPool::OnPacket` and
`CReactorPool::OnReactorChangeState` for both versions:

- gms_v48 `OnPacket` @ `0x5a5390` dispatches only three opcodes: `210` →
  `OnReactorChangeState`, `212` → `OnReactorEnterField`, `213` →
  `OnReactorLeaveField`. No fourth arm, no touch/proximity acknowledgment.
- gms_v61 `OnPacket` @ `0x633133` dispatches only three opcodes: `214` →
  `OnReactorChangeState`, `216` → `OnReactorEnterField`, `217` →
  `OnReactorLeaveField`. Same three-opcode shape.
- Neither version's `OnReactorChangeState` body references a
  `m_reactorOnLocalUser`-style `ZMap` proximity-state field (v61's version
  does reference `ZMap<long,ZRef<REACTOR>,long>::GetAt`, but that is the
  *reactor object* lookup by id, unrelated to the touch/local-user proximity
  map that `FindTouchReactorAroundLocalUser` maintains in every in-scope
  version).

Both checks independently corroborate the name-search result. **gms_v48 and
gms_v61 remain genuinely `n-a`** — no send site exists in either binary, and
neither client's receive path implies one. Neither version enters scope for
Tasks 2–5; the plan does not need revision on this point.

## In-scope version list and opcodes

| version | opcode (decimal / hex) |
|---|---|
| gms_v72 | 196 / `0x0C4` |
| gms_v79 | 198 / `0x0C6` |
| gms_v83 | 206 / `0x0CE` |
| gms_v84 | **212 / `0x0D4`** (overturns registry's 206) |
| gms_v87 | 219 / `0x0DB` |
| gms_v92 | 243 / `0x0F3` |
| gms_v95 | 250 / `0x0FA` |
| jms_v185 | 217 / `0x0D9` |

Out of scope (measured `n-a`, positive absence evidence recorded above):
gms_v48, gms_v61.

Eight in-scope versions, matching design §1.7's count. The single correction
this task makes to the design's carried-forward table is gms_v84: **212, not
the registry's unconfirmed `0x0CE`/206.**

## Canvas origin convention (Task 6)

**Caveat on method.** This task's implementer session had no `mcp__ida-pro__*`
tools available, so it did not re-run a fresh decompile of
`CReactorPool::LoadReactorLayer` (gms_v83 `0x7348a0`). The confirmation below
draws on (a) the decompile evidence design §1.5 already recorded from earlier
work on this task, and (b) cross-checking that evidence against real WZ data
and this repo's own `xml.CanvasNode`/`point.RestModel` semantics. If a later
session has IDA access and wants to re-derive `LoadReactorLayer` from scratch,
that would strengthen this further, but nothing here overturns design §5.1's
formula.

Design §1.5 already recorded, from a decompile of the client's touch check,
that the client does **not** use the WZ `tl`/`rb` event vectors for hit
testing. It instead reads the reactor's *rendered layer* rectangle:

```
IWzGr2DLayer::Getlt(p->pLayer) -> rc.left,  rc.top
IWzGr2DLayer::Getrb(p->pLayer) -> rc.right, rc.bottom
PtInRect(&rc, pLocal->GetPos())
```

and that "the layer is the current state's canvas, placed at the reactor's
map position and anchored by the canvas `origin`" (design.md:158-159).

That "anchored by origin" placement is the standard `IWzGr2DLayer` convention
used identically for every drawable canvas in the client's asset pipeline —
mobs, NPCs, portals, items, reactors alike: a canvas of size `(w, h)` with an
`origin` vector `(ox, oy)` is drawn so that pixel `(ox, oy)` of the bitmap
lands on the object's logical position. In reactor-local coordinates (the
object's position as `(0,0)`), that places the rendered rect's corners at:

```
lt = pos - origin = (-ox, -oy)
rb = pos - origin + size = (w - ox, h - oy)
```

which is exactly design §5.1's formula. Three independent checks corroborate
it against this repo's own data and code, rather than against remembered
client behaviour:

1. **`xml.CanvasNode.GetPoint`/`point.RestModel` semantics** (`xml/model.go:206`,
   `point/rest.go`) apply no transform of their own — `origin` is read as a raw
   `(x, y)` pair verbatim from the WZ vector node. Nothing in the repo's XML
   layer inverts or rescales it; any sign convention is applied only at the
   point where the rectangle is derived, which is exactly what §5.1/Step 4 do.
2. **Self-consistency against real `Reactor.wz` data.** Reactor `2406000`'s
   three states (state 0: `115×45` origin `(53,-24)`; state 1: `122×137` origin
   `(56,68)`; state 2: `1×1` origin `(0,0)` — figures from design.md:160-162,
   reproduced verbatim in the Step 2 fixture below) run through the formula
   above to give exactly the `TL`/`BR` pairs in the Step 2 table: state 0 →
   `(-53,24)`/`(62,69)`, state 1 → `(-56,-68)`/`(66,69)`, state 2 → `(0,0)`/`(1,1)`.
   State 2's `1×1` stub collapsing to a single point at the origin matches
   design.md:161-162's independent read of it as "deliberately untouchable
   once spent" — a degenerate rectangle is the expected shape for an inert
   state, not an artifact of a wrong sign.
3. **Existing fixture precedent.** The pre-existing `testXML` fixture (reactor
   `1002000`, `reader_test.go:12`) carries a `158×211` canvas with origin
   `(79,105)` — origin sits within a few pixels of true centre (`79,105.5`),
   the expected shape for a reactor sprite anchored at its own visual middle,
   not at a corner or an arbitrarily offset point. A corner-anchored
   convention (`origin == (0,0)` or `origin == (w,h)`) would be the signal to
   distrust the sign in §5.1; a near-centre origin is exactly what the
   `-origin`/`size-origin` formula expects to see across ordinary reactor art.

No evidence found in this session contradicts design §5.1's formula; Step 3/4
proceed with it unchanged.

**Direct WZ cross-check.** The mounted Cosmic WZ tree's
`Reactor.wz/2406000.img.xml` (`<wz-mount-root>/wz/Reactor.wz/2406000.img.xml`)
confirms the Step 2 fixture figures verbatim: state `0` canvas `0` is
`115×45` origin `(53,-24)`; state `1` canvas `0` is `122×137` origin
`(56,68)`; state `2` canvas `0` is `1×1` origin `(0,0)`. This is not
fabricated test data — it is copied from the actual mounted WZ file.

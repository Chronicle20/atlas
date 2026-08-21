# Touch-Activated Reactors — Design

Version: v1
Status: Draft
Created: 2026-08-21
PRD: [prd.md](prd.md)

---

## 0. What this document settles

The PRD left six open questions and one enumeration (FR-20) explicitly to the
design phase. All seven are resolved here against primary evidence — client
binaries via ida-pro-mcp, `Reactor.wz`, and repository source. Section 1 records
the measurements; sections 2–9 build the architecture on top of them.

One PRD requirement does not survive contact with the evidence: **FR-14's
`[TL, BR]` bounds check cannot be implemented as written**, because all ten
`activateByTouch` templates leave `TL`/`BR` at the zero value under
`atlas-data`'s current derivation. §1.5 states the measurement and §5 states the
replacement. This is the one place the design deliberately deviates from the PRD.

---

## 1. Evidence

### 1.1 The wire layout (OQ-1)

Decompiled `CReactorPool::FindTouchReactorAroundLocalUser` in
`GMS_v95.0_U_DEVM.exe` at `0x6cded0`. The send path, verbatim from the
pseudocode:

```
COutPacket::COutPacket(&oPacket, 250);   // opcode 250 = 0x0FA
COutPacket::Encode4(&oPacket, dwID);     // reactor object id
COutPacket::Encode1(&oPacket, 1u);       // entering  -> 1
CClientSocket::SendPacket(...);
```

and, in the mirrored `else` arm of the same function:

```
COutPacket::COutPacket(&v32, 250);
COutPacket::Encode4(&v32, dwID);
COutPacket::Encode1(&v32, 0);            // leaving   -> 0
CClientSocket::SendPacket(...);
```

So `TOUCHING_REACTOR` is five bytes of body:

| offset | size | field      | meaning                                  |
|--------|------|------------|------------------------------------------|
| 0      | 4    | `oid`      | reactor object id (`REACTOR::dwID`)      |
| 4      | 1    | `touching` | `1` on entering the area, `0` on leaving |

The opcode constant in the `COutPacket` constructor matches the per-version
registry (`0x0FA` for gms_v95), which cross-validates both the registry and the
function identification.

**The client is edge-triggered, not level-triggered.** The enclosing loop keys a
`ZMap<long,long,long> m_reactorOnLocalUser` by `dwID`. It sends `touching=1`
only when `PtInRect` succeeds *and* the map has no entry for that reactor, and
sends `touching=0` only when `PtInRect` fails *and* the map *does* have an entry.
A character standing still inside the area produces exactly one packet, not one
per frame. This directly shapes the idempotence design (§6).

### 1.2 Layout stability across versions

The same two-field body was confirmed at both ends of the version range and at
two interior points:

| version    | evidence                                                     | opcode          |
|------------|--------------------------------------------------------------|-----------------|
| gms_v72    | full decompile @ `0x692bb0`: `COutPacket(196); Encode4; Encode1` | **`0x0C4` (196)** |
| gms_v79    | full decompile @ `0x6b8362`: `COutPacket(198); Encode4; Encode1` | **`0x0C6` (198)** |
| gms_v87    | call-site instructions in `0x77bca7`: `push 1; push 0DBh` / `push ebx; push 0DBh` | `0x0DB` |
| gms_v95    | full decompile @ `0x6cded0`                                  | `0x0FA`         |
| jms_v185   | call-site instructions in `0x79f0aa`: `push 1; push 0D9h` / `push ebx; push 0D9h` | `0x0D9` |

**The codec needs no version gate.** Unlike its sibling `HitRequest`, which
gained `isSkill` between v61/v72 and `skillId` between v72/v79, `TOUCHING_REACTOR`
has carried the identical `(uint32, byte)` body since it first appeared. The
implementation writes one ungated struct. (A `MajorAtLeast` gate must still not
be replaced with a raw numeric comparison anywhere the codec grows one later.)

### 1.3 The four `n-a` versions (OQ-3, FR-3)

`docs/packets/audits/support/{gms_v48,gms_v61,gms_v72,gms_v79}.md` all record
`n-a`. Two of those four are wrong:

| version | measurement | verdict |
|---|---|---|
| gms_v48 | `func_query` for `CReactorPool` returns six symbols (`OnPacket`, `OnReactorChangeState`, `OnReactorEnterField`, `OnReactorLeaveField`, `FindHitReactor`, `LoadReactorLayer`) and **no** `FindTouchReactorAroundLocalUser` | genuinely **n-a**, now measured |
| gms_v61 | same six symbols, same absence | genuinely **n-a**, now measured |
| gms_v72 | `FindTouchReactorAroundLocalUser` present @ `0x692bb0`, opcode `196` | **in scope, `0x0C4`** — support file is wrong |
| gms_v79 | present @ `0x6b8362`, opcode `198` | **in scope, `0x0C6`** — support file is wrong |

The v48/v61 IDBs carry full `CReactorPool` symbol coverage, so absence of the
symbol is a measurement of absence, not of a stripped IDB.

### 1.4 Versions whose IDB has not named the function

`gms_v83`, `gms_v84` and `gms_v92` return no `FindTouchReactorAroundLocalUser`
symbol, but unlike v48/v61 this is an IDB-naming gap, not evidence of absence:

- **gms_v83** — `CReactorPool` is otherwise fully symbolised (`Update`,
  `LoadReactorLayer`, `OnPacket`, `OnReactorChangeState`, `OnReactorEnterField`,
  `OnReactorLeaveField`, `FindHitReactor` @ `0x7356c7`, `FindSkillReactor` @
  `0x735ab7`). `find_bytes "68 CE 00 00 00"` yields `0x735fb9` and `0x736021`,
  immediately after `FindSkillReactor` ends (`0x735d13`) — the expected pair of
  `COutPacket(0xCE)` constructions in the enter and leave arms of an unnamed
  function. **Implementation must name that function and derive from it**; it is
  a prerequisite this task produces, not a blocker.
- **gms_v92** — `find_bytes "6A 01 68 F3 00 00 00"` yields `0x79f903`,
  `0x79f9f4`, `0x828dad`. The first two are adjacent and are the likely
  candidates; the third is unrelated. Name and confirm during implementation.
- **gms_v84** — `find_bytes "68 CE 00 00 00"` yields no match anywhere near the
  `CReactorPool` cluster (`FindHitReactor` @ `0x752cbc`, ending `0x7530ac`).
  **The registry's `0x0CE` for gms_v84 is therefore unconfirmed and possibly
  wrong.** Treat gms_v84's opcode as unknown until derived from the binary; do
  not copy v83's value across.

### 1.5 Reactor bounds are absent for every touch template (invalidates FR-14 as written)

Enumerated from `Reactor.wz` (Cosmic set — the ten-item list below is the
authoritative one; the nine-item lists in `docs/TODO.md:280` and task-019's PRD
undercount):

```
2406000  6109013  6109014  6109021  6109022  6109023
6109024  6109025  6109026  6109027
```

All ten carry `<int name="activateByTouch" value="1"/>`. **OQ-4 resolved: ten,
including `2406000`** (`나인스피릿의둥지` — the Nine Spirit nest, a Horntail
prequest reactor, not a GPQ one, which explains why the GPQ-focused lists missed
it). Other WZ sets on this machine disagree on count (`atlas-ros` 10,
`ms_1172` 9, `AtlasMS` 19), so the authoritative list is per-mounted-WZ; the ten
above are the list for the Cosmic/v83-era data Atlas reads.

The decisive finding: **none of the ten templates contains a type-100 event.**
`atlas-data` populates `TL`/`BR` exclusively inside the `if t == 100` branch at
`services/atlas-data/atlas.com/data/reactor/reader.go:111`, reading the event's
`tl`/`rb` child vectors. With no type-100 event, `TL` and `BR` stay at
`point.RestModel{}` — `(0,0)`/`(0,0)` — for every reactor this feature exists to
serve. An `[TL, BR]` bounds check as FR-14 specifies would reject 100% of
legitimate touches.

The client does not use `tl`/`rb` either. It reads the reactor's *rendered
layer* rectangle:

```
IWzGr2DLayer::Getlt(p->pLayer) -> rc.left,  rc.top
IWzGr2DLayer::Getrb(p->pLayer) -> rc.right, rc.bottom
PtInRect(&rc, pLocal->GetPos())
```

The layer is the current state's canvas, placed at the reactor's map position
and anchored by the canvas `origin`. That geometry *is* present in the WZ for
all ten templates, and it is per-state — `2406000` state 0 is `115×45` at origin
`(53,-24)`, state 1 is `122×137` at origin `(56,68)`, and state 2 is a `1×1`
stub, i.e. deliberately untouchable once spent. §5 builds the bounds check on
this instead.

### 1.6 Downstream service facts

- **`atlas-reactor-actions` ignores unknown command types** (OQ-5):
  `services/atlas-reactor-actions/atlas.com/reactor/script/consumer.go:70` falls
  through to `default: l.Warnf("Unknown command type: %s", ...)`. Nothing errors
  and nothing crashes. Ordering is therefore not a correctness constraint — but
  the consumer should still land with or before the producer to avoid a warn
  flood on every touch.
- **Rules are stored as JSONB**, not as columns
  (`script/entity.go`, keys `hitRules` / `actRules`). A third rule bucket is an
  additive JSON key; **no database migration is required.**
- **A cross-service character-position read already has a precedent**:
  `services/atlas-maps/atlas.com/maps/character/processor.go` exposes a narrow
  read-only `Snapshot(characterId) (x, y, hp, error)` over
  `requests.RootUrlFor(ctx, "CHARACTERS")`, added for the mist tick.
- **`atlas-character`'s REST `x`/`y` is live, not persisted-stale**:
  `character/rest.go:82` projects from `GetTemporalRegistry()`, which the
  `COMMAND_TOPIC_CHARACTER_MOVEMENT` consumer updates on every movement command.
  It is the authoritative current position.
- **`atlas-channel`'s `session.Model` carries no position** — only
  `characterId`, `field`, and connection state. The channel cannot perform a
  bounds check without a REST call of its own.
- **`reactor/data.Model` in `atlas-reactors` has no builder.** It is constructed
  directly by `Extract(RestModel)` in `reactor/data/rest.go`. FR-11's reference
  to "the package's existing builder" describes a builder that does not exist;
  plumbing goes through `RestModel` + `Extract`.
- **Two seed templates do not route `ReactorHitHandle`**: `template_gms_12_1.json`
  and `template_gms_92_1.json`. gms_12 is not a coverage-matrix column and is out
  of scope. gms_92's missing hit routing is a pre-existing gap this task does not
  fix; see §7.

### 1.7 Resulting version scope

| version   | opcode                     | in scope |
|-----------|----------------------------|----------|
| gms_v48   | —                          | no (measured n-a) |
| gms_v61   | —                          | no (measured n-a) |
| gms_v72   | `0x0C4`                    | **yes** (support file corrected) |
| gms_v79   | `0x0C6`                    | **yes** (support file corrected) |
| gms_v83   | `0x0CE` (registry; confirm at named fn near `0x735fb9`) | yes |
| gms_v84   | **unknown** — registry `0x0CE` unconfirmed (§1.4) | yes, opcode to be derived |
| gms_v87   | `0x0DB` (confirmed)        | yes |
| gms_v92   | `0x0F3` (registry; confirm near `0x79f903`) | yes |
| gms_v95   | `0x0FA` (confirmed)        | yes |
| jms_v185  | `0x0D9` (confirmed)        | yes |

Eight in-scope cells; two measured `n-a`.

---

## 2. Architecture overview

```
client
  │  TOUCHING_REACTOR { oid, touching }
  ▼
atlas-channel
  socket/handler/touch_reactor.go
  reactor.Processor.Touch(field, oid, characterId, touching)
  │  COMMAND_TOPIC_REACTOR  type=TOUCH  { reactorId, characterId, touching }
  ▼
atlas-reactors
  kafka consumer  ->  ProcessorImpl.Touch(reactorId, characterId, touching)
  ├─ registry lookup            (tenant-scoped)
  ├─ data.ActivateByTouch()     gate       ─┐
  ├─ character Snapshot(x,y)    + AABB      ├─ FR-14 rejections
  ├─ touch latch (Redis)        idempotence ─┘
  ├─ state progression (shared with Hit)
  ├─ EVENT_TOPIC_REACTOR_STATUS  type=HIT
  │  COMMAND_TOPIC_REACTOR_ACTIONS  type=TOUCH  { characterId }
  ▼
atlas-reactor-actions
  ProcessTouch -> touchRules (falling back to hitRules)
```

Two supporting data flows:

```
Reactor.wz ──> atlas-data reader ──> reactor RestModel
                                      + activateByTouch : bool
                                      + touchAreaInfo   : map[state]{tl,br}
                                          ──> atlas-reactors reactor/data.Model
```

The design keeps three properties:

1. **The authority does the validating.** Every FR-14 check lives in
   `atlas-reactors`, which owns reactor state; `atlas-channel` decodes and
   forwards and does not decide.
2. **Touch reuses state progression, not the hit predicate.** The state machine
   is factored out of `Hit` and shared; the *event-selection* predicate is not.
3. **Nothing on the hit path changes shape.** `Hit`'s signature, its skill
   gating, and its `TriggerAndDestroy` fall-throughs are preserved verbatim.

---

## 3. Packet layer

`libs/atlas-packet/reactor/serverbound/touching.go`:

```go
const TouchReactorHandle = "TouchReactorHandle"

// TouchingRequest - CReactorPool::FindTouchReactorAroundLocalUser
// packet-audit:fname CReactorPool::FindTouchReactorAroundLocalUser
type TouchingRequest struct {
    oid      uint32
    touching bool
}
```

`Encode` writes `WriteInt(oid)` then `WriteByte(1|0)`; `Decode` mirrors it with
`ReadUint32()` / `ReadByte() == 1`. No `MajorAtLeast` gate — §1.2. Accessors
`Oid()`, `Touching()`, plus `Operation()` and `String()` to match `hit.go`.

Per-version verification follows `docs/packets/audits/VERIFYING_A_PACKET.md`
unchanged: a `packet-audit:verify` byte fixture per in-scope cell, a pinned
evidence record, and a regenerated matrix, committed as one unit.

Two registry/support corrections ride along:

- `docs/packets/audits/support/gms_v72.md` and `gms_v79.md`: `n-a` →
  `0x0C4` / `0x0C6`, with the derivation addresses recorded.
- `docs/packets/audits/support/gms_v48.md` and `gms_v61.md`: keep `n-a`, but
  state that it was **measured** (symbol absent from a fully-symbolised
  `CReactorPool`), not interpolated.

### Alternative considered — reuse `HitRequest`

Rejected. The bodies differ (`uint32 + byte` vs. up to `uint32 ×3 + uint16`),
the opcodes differ, and the client's leave-notification has no analogue on the
hit path. Sharing the struct would force a mode discriminator into a codec that
has none on the wire.

---

## 4. Channel layer

`services/atlas-channel/atlas.com/channel/socket/handler/touch_reactor.go`
mirrors `reactor_hit.go`: decode, `l.Debugf("[%s] read [%s]", ...)`, delegate.
No validation, no session lookup beyond `s.Field()` and `s.CharacterId()`.

`reactor.Processor` gains:

```go
Touch(f field.Model, reactorId uint32, characterId uint32, touching bool) error
```

emitting `reactor2.CommandTypeTouch` with
`TouchCommandBody{ReactorId, CharacterId, Touching}` via a new
`TouchCommandProvider`, keyed on `reactorId` exactly as `HitCommandProvider` is —
so a reactor's touch and hit commands stay ordered relative to each other on the
same partition.

Registration: `handlerMap[reactorsb.TouchReactorHandle] = handler.TouchReactorHandleFunc`
in `main.go`, alongside the existing hit entry at line 1017.

### Why the leave notification is forwarded rather than dropped at the edge

`touching=0` is not an activation, and forwarding it costs a Kafka message that
changes no reactor state. It is forwarded anyway because it is the *only* signal
that clears the touch latch (§6). Dropping it at the channel would push the
latch's lifetime onto a TTL guess; forwarding it makes the server's latch an
exact mirror of the client's `m_reactorOnLocalUser` map.

---

## 5. Reactor data plumbing

### 5.1 `atlas-data`

`reactor.RestModel` gains two fields:

```go
ActivateByTouch bool                     `json:"activateByTouch"`
TouchAreaInfo   map[int8]AreaRestModel   `json:"touchAreaInfo"`
```

`ActivateByTouch` is populated from the value already read at `reader.go:80`
(the existing `loadArea` local is derived from the same node and its use at
`reader.go:111` is untouched, honouring the PRD's non-goal).

`TouchAreaInfo` is the FR-14 replacement. For each state directory, the reader
takes canvas `0` (`xml.CanvasNode`, which already exposes `Width`, `Height`, and
`GetPoint("origin", …)`) and derives a reactor-local rectangle:

```
tl = ( -origin.x,            -origin.y            )
br = ( -origin.x + width,    -origin.y + height   )
```

A state with no canvas contributes no entry. The map is emitted for every
reactor, not only touch ones — it is cheap, and gating it on `activateByTouch`
would make the field's presence depend on a flag consumers also read.

> **Implementation gate.** The origin sign convention above is inferred from the
> standard canvas anchor semantics and matches the shapes seen in the WZ
> (`6109013` centres on the reactor; `2406000` state 2 is a `1×1` stub). It MUST
> be confirmed against `CReactorPool::LoadReactorLayer` (gms_v83 `0x7348a0`,
> gms_v95 reachable from the same cluster) before the bounds check is trusted,
> and the confirmation recorded in the task folder. If the convention differs,
> only this formula changes — no other part of the design moves.

### 5.2 `atlas-reactors`

`reactor/data.Model` gains `activateByTouch bool` and
`touchAreaInfo map[int8]area.Model`, with accessors `ActivateByTouch() bool` and
`TouchArea(state int8) (area.Model, bool)`. Both are populated in
`Extract(RestModel)` in `reactor/data/rest.go` — there is no builder in this
package (§1.6).

FR-12 falls out of Go's zero values: an absent `activateByTouch` decodes to
`false` and an absent `touchAreaInfo` decodes to `nil`. A `nil` area map means
"no touch area known", which §6 treats as a rejection, not as an unbounded pass.

### Alternative considered — reuse `TL`/`BR`

Rejected. `TL`/`BR` already mean "item-drop area, taken from the type-100
event", they are per-reactor rather than per-state, and they are zero for all
ten touch templates. Overloading them would both break the drop semantics and
fail the exact reactors this task targets. A separate per-state field also keeps
the PRD's "`atlas-data` `tl`/`br` derivation stays as is" non-goal intact.

---

## 6. Touch activation in `atlas-reactors`

### 6.1 Entry point

```go
Touch(reactorId uint32, characterId uint32, touching bool) error
```

routed from a new `TOUCH` arm in the reactor command consumer.

`touching == false` short-circuits: clear the latch for
`(tenant, reactorId, characterId)`, log at debug, return `nil`. No lookup, no
state change, no actions command.

### 6.2 Rejection ladder (FR-14)

In order, each logging reactor id, character id, and the specific reason:

1. `GetById(reactorId)` fails → reject (reactor gone or wrong tenant).
2. `r.Data().ActivateByTouch()` is false → reject. This is the anti-cheat gate
   that makes a forged `TOUCHING_REACTOR` inert against ordinary reactors.
3. `r.Data().TouchArea(r.State())` is absent → reject. Unknown geometry is not a
   pass.
4. Character position: `character.NewProcessor(l, ctx).Snapshot(characterId)`
   (new read-only package mirroring `atlas-maps`'). Failure → reject.
5. AABB: reject unless
   `r.X()+tl.X ≤ cx ≤ r.X()+br.X` and `r.Y()+tl.Y ≤ cy ≤ r.Y()+br.Y`.
6. Latch already set for `(tenant, reactorId, characterId)` → no-op, not an
   error (FR-18).

Only after all six does the state progression run, and the latch is set as part
of accepting.

### 6.3 Idempotence (FR-18)

The client is already edge-triggered (§1.1), so the latch is a defence against a
modified client, not against normal traffic. It lives in the same Redis-backed
registry the reactor state does — `atlas-reactors` runs multi-pod, so an
in-process map would let two pods both accept the same touch. Shape: a set per
reactor, `touch:{tenant}:{reactorId}` with member `characterId` and a TTL, mirroring
the cooldown/spot helpers already on `Registry`. It is cleared on
`touching=0`, on `Registry.Remove`, and by `DestroyInField`'s sweep, alongside
the existing `CancelPendingActivation` / `cancelStateTimeout` calls.

Latching on `(reactor, character)` rather than `(reactor, character, state)`
matters for the cyclic templates: `6109013` state 0 →(type 6)→ state 1 →(type 7)→
state 0. A state-keyed latch would let a character standing still re-trigger
every time the cycle came back around to a state they had not yet touched. The
character-keyed latch matches the client's `m_reactorOnLocalUser` exactly: one
activation per entry, regardless of how the reactor's state moves underneath.

### 6.4 State progression (FR-15, FR-16, FR-19)

`Hit`'s body splits into two pieces:

- `selectNextState(stateEvents, skillId) (nextState int8, eventType int32)` —
  the existing `len(event.ActiveSkills()) == 0 || containsSkill(...)` predicate,
  used only by `Hit`.
- `advance(r Model, characterId uint32, nextState int8, matchedEventType int32) error` —
  everything from the `_, hasNextState := stateInfo[nextState]` check onward:
  the `persistsAtEndState` branch, the `isTerminalState` branch, the registry
  update, `scheduleStateTimeout`, `Trigger`, and the `hitStatusEventProvider`
  emission. Shared verbatim by both paths.

`Hit` after the split is `selectNextState` + `advance` and is byte-for-byte
equivalent in behaviour; its existing tests in `processor_test.go` must pass
unchanged, which is the FR-19 guard.

`Touch` selects differently:

```go
stateEvents := r.Data().StateInfo()[r.State()]
if len(stateEvents) == 0 {
    // FR-16 / OQ-6: no-op, do NOT TriggerAndDestroy
    return nil
}
next, eventType := stateEvents[0].NextState(), stateEvents[0].Type()
return p.advance(r, characterId, next, eventType)
```

**`activeSkills` is not consulted.** The `activateByTouch` flag is the gate;
`activeSkills` constrains only the hit path. Reusing the hit predicate on
templates whose events are type 5/6/7 with a non-empty `activeSkills` list would
leave `nextState == -1` and fall through to `TriggerAndDestroy` — destroying the
reactor on touch instead of advancing it. That inversion is the single most
important regression guard in the task and gets a dedicated test.

**OQ-6 resolved: a touch on a state with no events is a no-op, not a
`TriggerAndDestroy`.** `Hit`'s destroy-on-empty is defensible for a deliberate
attack; walking past a spent reactor is not a request to destroy it. The WZ
supports this directly: `2406000` state 2 has no `event` node and a `1×1`
canvas — the data itself says "this state is inert and untouchable". Preserving
`Hit`'s behaviour here while diverging on touch is intentional, and §6.4's split
keeps the divergence to one `if`.

### 6.5 Actions command (FR-17)

New `CommandTypeActionsTouch = "TOUCH"` on the existing `reactorActionsCommand[E]`
envelope with `touchActionsBody{CharacterId uint32}`, emitted through a
`touchActionsCommandProvider` mirroring `triggerActionsCommandProvider`. `Touch`
emits `TOUCH` and never `HIT`.

Ordering note: `Hit` emits its `HIT` actions command *before* the state
transition, so the script sees the pre-transition state. `Touch` follows the same
order for consistency — emit, then advance — so a script author reading
`reactorState` gets the same "state at time of activation" semantics on both
paths.

---

## 7. `atlas-reactor-actions`

`CommandTypeTouch = "TOUCH"` is added to `script/kafka.go` and a
`case CommandTypeTouch: handleTouchCommand(...)` arm to `handleCommandFunc`.
`handleTouchCommand` is `handleTriggerCommand` with `ProcessTouch` substituted;
its body shape (extract `characterId` from the `map[string]interface{}` body,
process, `executeOperations`) is unchanged.

`ProcessTouch` evaluates `script.TouchRules()` with `eventType == "touch"`, and
**falls back to `HitRules()` when `touchRules` is absent or empty.** The JSONB
`data` column gains a `touchRules` key; no migration (§1.6).

The fallback is the one judgement call here. Without it, this task ships a
mechanism that does nothing observable, because none of the ten templates has a
script yet and authoring them is an explicit PRD non-goal — a touch would
advance the reactor's state and then hit `no_match`. With it, a touch-activated
reactor that already has hit rules behaves sensibly the moment the mechanism
lands, and an author who wants divergent behaviour adds `touchRules` and takes
over. Distinguishability is preserved either way: the command type is `TOUCH`,
the log line says `touch`, and a script that declares `touchRules` never sees the
hit path.

### Alternative considered — reuse `HIT` for both

Rejected outright by FR-17, and correctly: a script could not tell a walk-over
from an attack, and the `isSkill`/`skillId` fields of `hitActionsBody` have no
meaningful value for a touch.

---

## 8. Configuration

`TouchReactorHandle` is routed at each version's opcode in the eight in-scope
seed templates: `template_gms_{72,79,83,84,87,92,95}_1.json` and
`template_jms_185_1.json`. `template_gms_{12,48,61}_1.json` gain nothing.

Two notes:

- gms_v84's entry waits on the opcode derivation of §1.4; it is **not** written
  as `0x0CE` on the registry's unconfirmed say-so.
- `template_gms_92_1.json` currently routes no `ReactorHitHandle` at all. This
  task adds `TouchReactorHandle` there (its opcode is a matrix column and the
  codec covers it) and records the missing hit routing as a pre-existing gap
  outside this task's scope. Adding a hit route to that template would be a
  behaviour change to a version this task is not otherwise touching.

---

## 9. Testing

| Layer | Test |
|---|---|
| packet | Byte fixture per in-scope version with a `packet-audit:verify` marker; encode/decode round-trip; both `touching=1` and `touching=0`. |
| atlas-data | `activateByTouch` true for `6109013`, false for a non-touch template; `touchAreaInfo` derived per state for `2406000` (three distinct states, including the `1×1` state 2). |
| atlas-reactors data | `Extract` with the fields absent yields `false` / `nil` (FR-12). |
| atlas-reactors touch | Accept: flag set, character inside, state advances. |
| | Reject: flag unset → no state change. |
| | Reject: character outside the AABB → no state change. |
| | Reject: `touchAreaInfo` missing for the current state → no state change. |
| | **FR-16 guard**: state whose only event is type 6 with a non-empty `activeSkills` list **advances** and is **not** destroyed. |
| | OQ-6 guard: state with zero events is a no-op; the reactor still exists afterward. |
| | Idempotence: two `touching=1` commands advance once; `touching=0` then `touching=1` advances again. |
| | Actions: exactly one `TOUCH` command emitted, zero `HIT` commands. |
| | Regression: existing `Hit` tests pass unchanged after the `selectNextState`/`advance` split. |
| atlas-reactor-actions | `TOUCH` routes to `ProcessTouch`; `touchRules` wins when present; falls back to `hitRules` when absent; unknown types still warn-and-ignore. |
| channel | Handler decodes and emits `TOUCH` with the `touching` flag intact. |

Test setup uses the repository's Builder pattern throughout; no
`*_testhelpers.go`.

---

## 10. Risks

| Risk | Mitigation |
|---|---|
| The canvas-origin sign convention is inferred, not measured. | §5.1 makes confirming it against `LoadReactorLayer` a gate before the bounds check is trusted. Wrong convention rejects every touch loudly rather than accepting a forged one. |
| gms_v84's opcode is unconfirmed and may not be `0x0CE`. | §1.4 / §8: derive from the binary; do not copy v83's value. A wrong opcode routes a live handler onto an unrelated packet. |
| One REST call to `atlas-character` per touch. | Bounded by the client's edge-triggering: one call per area entry, not per frame. The latch check runs after the position read, so a spamming client still costs one REST call per forged packet — acceptable, and no worse than any other unvalidated serverbound op. |
| `atlas-reactors` gains a dependency on `atlas-character`. | Narrow, read-only, and precedented (`atlas-maps` does exactly this). The alternative — checking in `atlas-channel` — puts anti-cheat on the edge and still needs the same REST call, since the session carries no position. |
| Splitting `Hit` risks a behaviour change on the hit path. | The split is mechanical and the existing `processor_test.go` suite is the guard; it must pass unedited. |

---

## 11. Deviations from the PRD

1. **FR-14's `[TL, BR]` bounds check is replaced by a per-state
   `touchAreaInfo` rectangle** derived from canvas geometry. `TL`/`BR` are
   `(0,0)` for all ten touch templates (§1.5), so the check as specified would
   reject every legitimate touch. The security property FR-14 asks for is
   preserved and, in fact, is only achievable this way.
2. **FR-3's `n-a` re-verification changes the version scope**: gms_v72 and
   gms_v79 are in scope at `0x0C4` / `0x0C6`; gms_v48 and gms_v61 remain `n-a`
   by measurement.
3. **FR-11's "existing builder" does not exist** in `reactor/data`; plumbing
   goes through `RestModel` + `Extract`.
4. **The `touching` flag is carried end to end**, which the PRD's §5.2
   `TouchCommandBody` does not include. OQ-1's answer makes it the natural latch
   signal (§6.3).
5. **§5.3's `TOUCH` actions command falls back to `hitRules`** when a script
   declares no `touchRules` (§7), so the mechanism is observable on day one
   without violating the "no script authoring" non-goal.

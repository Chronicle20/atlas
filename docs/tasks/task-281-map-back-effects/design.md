# Map Back-Effects — Design

Task: task-281-map-back-effects
Phase: 2 (design)
Input: `docs/tasks/task-281-map-back-effects/prd.md`
Status: Draft

---

## 0. What this design settles

The PRD deferred four things to a decompile (Open Questions 1–4) and one to this
phase (Open Question 5). Three of the four are now **derived, not assumed** — the
decompile was run during this phase because every downstream shape (registry key,
command body, event body, REST resource, GM command arguments) is a function of it,
and the IDBs are checked in. Evidence is in §1.

| PRD open question | Status after this phase |
|---|---|
| 1. Packet field list | **Resolved** — see §1.1 |
| 2. One back-effect per field, or several? | **Resolved: several**, keyed by page — see §1.1, §3.1 |
| 3. Does `CLEAR_BACK_EFFECT` carry a payload? | **Resolved: no.** Bare opcode — see §1.2 |
| 4. jms_v185 divergence | **Open.** GMS v72/v84/v95 proven identical; jms_v185 and v79/v83/v87/v92 confirmed during execution — see §1.3 |
| 5. Instance teardown hook | **Resolved by decision** (no reaper in v1) — see §3.4 |

---

## 1. Derived wire layout (evidence)

### 1.1 `SET_BACK_EFFECT`

GMS v95.1 (`ecc757f4`, `GMS_v95.0_U_DEVM.exe.i64`):

- `CMapLoadable::OnPacket` @ `0x61fd80` — `case 144 → OnSetBackEffect`,
  `case 145 → OnSetMapObjectVisible`, `case 146 → OnClearBackEffect`.
- `CMapLoadable::OnSetBackEffect` @ `0x612850` — calls `Field::BackEffect::Decode`
  and nothing else that touches the packet.
- `Field::BackEffect::Decode` @ `0x565500`, symbolized in the IDB:

```c
void Field::BackEffect::Decode(Field::BackEffect *this, CInPacket *iPacket)
{
  this->nEffect   = CInPacket::Decode1(iPacket);
  this->nFieldID  = CInPacket::Decode4(iPacket);
  this->nPageID   = CInPacket::Decode1(iPacket);
  this->tDuration = CInPacket::Decode4(iPacket);
}
```

Wire, in order — **10 bytes total**:

| # | Field | Width | Meaning (from the handler body, not from the name) |
|---|---|---|---|
| 1 | `nEffect` | byte | Branch selector. `0` → `IWzVector2D::RelMove(alpha, 255, …)` (fade the page **in**); `1` → `RelMove(alpha, 0, …)` (fade it **out**). Any other value: the handler returns without touching the field. |
| 2 | `nFieldID` | int32 | Decoded, **never read** by the v95 handler. Position-significant only. |
| 3 | `nPageID` | byte | The `long` key passed to `ZMap<long, ZRef<ZList<IWzGr2DLayer>>>::GetAt(&this->m_mlLayerBack, &key, &value)`. Selects one back-layer page of the current field. |
| 4 | `tDuration` | int32 | Added to `get_update_time()` to form the alpha tween's end time — a fade duration in client ticks (ms). |

Behaviour: the handler resolves page `nPageID` from the field's back-layer map,
walks its `IWzGr2DLayer` list, and tweens each layer's alpha to 255 (`nEffect==0`)
or 0 (`nEffect==1`) over `tDuration`. It is a **per-page show/hide with a fade**,
not a "load a different background" op.

Cross-version confirmation of the same four reads, same order:

| Version | Session | `OnSetBackEffect` | Decode body |
|---|---|---|---|
| gms_v72 | `99e435d8` | `0x5f5b4f` | `sub_54C265` @ `0x54c265` — `Decode1/Decode4/Decode1/Decode4` into `this[4..7]` |
| gms_v84 | `46c2a2eb` | `0x659e3c` | `sub_597B59` @ `0x597b59` — identical |
| gms_v95 | `ecc757f4` | `0x612850` | `Field::BackEffect::Decode` @ `0x565500` — identical |

v72 and v84 also reproduce the two-branch (`0`/`1`) alpha-255/alpha-0 structure
byte-for-byte in shape. **No divergence found so far; assume none, prove it per
version during execution.**

### 1.2 `CLEAR_BACK_EFFECT`

GMS v95.1, `CMapLoadable::OnClearBackEffect` @ `0x61f230`:

```c
// attributes: thunk
void CMapLoadable::OnClearBackEffect(CMapLoadable *this, CInPacket *iPacket)
{
  CMapLoadable::ReloadBack(this);   // 0x61f0c0
}
```

`iPacket` is untouched. **`CLEAR_BACK_EFFECT` is a bare opcode with an empty
body**, and its effect is field-wide: `ReloadBack` rebuilds the whole back-layer
set, so it clears *every* page's effect at once. It is not a per-page clear.

This asymmetry — per-page set, whole-field clear — is the single most important
fact for the server-side model and drives §3.1.

### 1.3 What execution still has to derive

- v79, v83, v87, v92, jms_v185: decompile `OnSetBackEffect` / `OnClearBackEffect`
  and their decode bodies; confirm the 4-field order and the empty clear.
- v48, v61 (⬜ in the matrix): the checked-in exports record
  `CMapLoadable::OnSetBackEffect` as `unresolved: true` ("function not found in
  IDB"), which is a *lead*, not evidence of absence. Read each IDB's
  `CMapLoadable::OnPacket` switch directly (sessions `12a398ce`, `921fdbb5`) and
  pin a VERSION-ABSENT record from the switch arms actually present.
- v72, v79 (`CLEAR_BACK_EFFECT` ⬜): the exports resolve `OnSetBackEffect` but
  carry no clear entry. Same treatment — read the `OnPacket` switch.

Per-version findings land in `docs/tasks/task-281-map-back-effects/structures/<version>.md`.

---

## 2. Packet layer

### 2.1 `SetBackEffect`

`libs/atlas-packet/field/clientbound/set_back_effect.go`, following
`field_obstacle_on_off.go`:

```go
const SetBackEffectWriter = "SetBackEffect"

// packet-audit:fname CMapLoadable::OnSetBackEffect
type SetBackEffect struct {
    effect   byte    // 0 = show (alpha -> 255), 1 = hide (alpha -> 0)
    fieldId  uint32  // decoded by the client, unread by the handler
    pageId   byte    // key into CMapLoadable::m_mlLayerBack
    duration uint32  // fade length in ms
}
```

Immutable, unexported fields, `NewSetBackEffect(...)`, accessors, `Operation()`,
`String()`, `Encode`, `Decode`. Encode order is exactly §1.1.

**Decision — model `effect` as a named type, not a bare byte.** Add
`BackEffectShow`/`BackEffectHide` constants in the packet package rather than
letting callers pass `0`/`1`. The values are wire constants proven by the
decompile, so they belong next to the codec.

**Decision — do not drop `fieldId` even though v95 ignores it.** It occupies four
wire bytes; the caller must supply it, and the natural value is the field the
effect is applied to. An earlier or later client may read it. Omitting it would
desynchronise the stream.

### 2.2 `ClearBackEffect`

`clear_back_effect.go`, an **empty-body** packet:

```go
const ClearBackEffectWriter = "ClearBackEffect"

// packet-audit:fname CMapLoadable::OnClearBackEffect
type ClearBackEffect struct{}
```

`Encode` returns the writer's bytes with nothing appended (opcode only, written by
the template layer); `Decode` reads nothing. Precedent for a zero-field
clientbound packet in this package: `field_obstacle_all_reset.go` — execution
should match its exact shape rather than inventing one.

### 2.3 Version gating

No divergence is known (§1.1), so **the first cut has no `MajorAtLeast` gate at
all**. If execution finds a per-version delta, gate it with the `MajorAtLeast`
idiom. Do not add a speculative gate now: an unexercised gate is a liability, and
adding one later is cheap.

### 2.4 Matrix work

14 ❌ → ✅ with per-cell byte fixtures (`// packet-audit:verify`), pinned tier-1
evidence, and audit reports; 6 ⬜ re-confirmed with fresh VERSION-ABSENT records
(§1.3). Both ops routed in every applicable seed template
(`services/atlas-configurations/seed-data/templates/template_gms_{72,79,83,84,87,92,95}_1.json`
and `template_jms_185_1.json` — v48/v61 templates get **no** route, matching ⬜).
`packet-audit matrix --check`, `fname-doc`, `operations --check` all exit 0.

**Scope note.** `SET_MAP_OBJECT_VISIBLE` is the sibling arm (case 145) in the same
`CMapLoadable::OnPacket` switch and is ❌ on the same eight versions. It is
**not** in this task's scope. The decompile of the switch that this task performs
makes it cheap to do next; record that as a follow-up, do not fold it in.

---

## 3. `atlas-maps` — state

New package `services/atlas-maps/atlas.com/maps/map/backeffect/`, files
`registry.go`, `processor.go`, `producer.go`, `rest.go`, `resource.go` —
the exact file set of `map/jukebox/`.

### 3.1 Registry shape — the central decision

The layout forces the shape:

- Set is addressed to **one page** (`nPageID`) of one field.
- Clear is **whole-field** (`ReloadBack`).

**Decision: `map[FieldKey][]BackEffectEntry` — an ordered slice per field, one
entry per `pageId`.**

```go
type FieldKey struct {           // identical to jukebox.FieldKey
    Tenant tenant.Model
    Field  field.Model
}

type BackEffectEntry struct {
    Effect   byte    // 0 show / 1 hide
    FieldId  uint32
    PageId   byte
    Duration uint32
}
```

- `Set(key, entry)`: replace in place if `PageId` already present (preserving its
  position), otherwise append. Last write per page wins; ordering across pages is
  first-set-first.
- `Clear(key)`: delete the whole slice. This is the only clear operation, because
  the client has no per-page clear.
- **No `ExpiresAt`, no reaper task.** Diverges from jukebox/weather deliberately;
  a back-effect is a state, not a timed effect. `tDuration` is a *fade* length,
  not a lifetime — reading it as an expiry would be a misreading of the decompile.

Alternatives considered:

| Alternative | Why rejected |
|---|---|
| One entry per field (PRD's fallback) | Contradicts the derived layout: two pages can be independently faded, and a single-entry model silently drops the first. |
| `map[FieldKey]map[byte]BackEffectEntry` | Loses replay order. Client alpha tweens are independent per page so order is *probably* immaterial — but "probably" is not evidence, and a slice costs nothing to keep deterministic. |
| Per-page clear command | There is no client-side per-page clear. Inventing one would mean lying about the wire. |

### 3.2 Command consumer

`CommandTypeSetBackEffect = "SET_BACK_EFFECT"`,
`CommandTypeClearBackEffect = "CLEAR_BACK_EFFECT"` added to
`kafka/message/map/command.go`, with bodies:

```go
type SetBackEffectCommandBody struct {
    Effect   uint8  `json:"effect"`
    FieldId  uint32 `json:"fieldId"`
    PageId   uint8  `json:"pageId"`
    Duration uint32 `json:"duration"`
}

type ClearBackEffectCommandBody struct{}
```

Two new arms in `kafka/consumer/map/consumer.go`, matching
`handlePlayJukeboxCommand` exactly: type guard → build `field.Model` → mutate
registry via the processor → produce the status event.

**Decision — validate `Effect` at the consumer.** A value other than 0/1 makes
the client handler return without doing anything; reject it (log at warn, drop
the command) rather than broadcasting a no-op packet. This is the analogue of the
existing `maxJukeboxDuration` clamp: bound a crafted or buggy command at the
service that owns the state.

**Decision — no clamp on `Duration`.** It is a fade length, bounded in practice
by the client's own tween; there is no denial-of-service shape here comparable to
pinning a field's BGM. Note the reasoning in a comment, as `maxJukeboxDuration`
does.

**Decision — clear on an empty field still emits.** Per PRD FR-4: a desynced
client must be able to be reset. Log at debug when the registry had nothing.

### 3.3 REST

**Decision — a collection endpoint, deviating from PRD §5.**

```
GET /worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/backEffects
  200 → JSON:API array of type "backEffect" (possibly empty)
```

The PRD specified a singular `/backEffect` returning 404 when none. That was
written under the "maybe one per field" assumption which FR-4 explicitly
conditioned on the decompile ("This is settled by the decompile, not by
preference"). With N-per-field proven, a collection with a 200-empty response is
the correct JSON:API shape and lets `atlas-channel` replay all pages in one call.
`RestModel.Id` is the `pageId` (the natural per-field identity), `GetName()`
returns `"backEffect"`.

### 3.4 Instance teardown (PRD Open Question 5)

**Decision: no teardown hook in v1.**

`atlas-maps` has no instance-destroyed event and no per-field occupancy state —
`character/location` resolves a character's field, not a field's roster; jukebox
and weather rely purely on expiry and neither reaps on teardown. Building
occupancy tracking here would be a materially larger change than the feature.

Accepted consequence: a destroyed instance leaves at most a few 10-byte entries
in a process-local map until `atlas-maps` restarts. Bounded by
(tenants × live fields × pages) and lost on restart anyway. Documented, not
hidden.

Rejected alternative: reap on `MAP_CHANGED` when the field empties — requires the
occupancy tracking that does not exist, and gets the semantics wrong for a
persistent-until-cleared effect on a field players cycle through.

---

## 4. `atlas-channel` — emit and replay

### 4.1 Writers

`SetBackEffectWriter` / `ClearBackEffectWriter` registered in `main.go` alongside
`fieldcb.FieldObstacleOnOffWriter`; body functions in `socket/writer/`
(`set_back_effect.go`, `clear_back_effect.go`).

### 4.2 Status-event arms

`EventTopicMapStatusTypeBackEffectSet = "BACK_EFFECT_SET"` /
`…Clear = "BACK_EFFECT_CLEAR"` in both `atlas-maps` and `atlas-channel`
`kafka/message/map/kafka.go`, with matching bodies. Two handlers in
`kafka/consumer/map/consumer.go` modelled on `handleStatusEventJukeboxStart` —
type guard, debug log, broadcast to sessions in the field via the existing
`ForSessionsInMap` + `doorAnnounce` composition.

### 4.3 Replay on map entry

**Decision — a new `announceActiveBackEffects`, sibling to
`announceActiveJukebox` (`consumer.go:1156`), dispatched from the same
`routine.Go` block at `consumer.go:~382`.**

Not folded into `announceActiveVisuals` (`consumer.go:787`): that function replays
*event* visuals sourced from the events service and is gated on
`event2.VisualContiMove`. Back-effects are field state from `atlas-maps`, a
different source with a different failure mode. Keeping them separate keeps each
replay's fail-open behaviour independently testable.

- New `services/atlas-channel/atlas.com/channel/backeffect/` REST client package
  (`requests.go`, `processor.go`, `rest.go`, `mock/processor.go`), mirroring
  `atlas-channel/jukebox/`.
- Fail-open: a lookup error logs at debug and returns, exactly as
  `announceActiveJukebox` does. An unreachable `atlas-maps` costs the background,
  never the map entry.

**Decision — replay with `Duration = 0`.** The fade already happened for everyone
else; a late joiner should see the end state immediately, not re-run the tween.
`Effect`, `FieldId`, and `PageId` are replayed as stored. Entries are replayed in
registry order.

Trade-off accepted: if a character enters *during* an in-flight fade they see the
final state a few hundred milliseconds early. The alternative — storing the
tween's start time and replaying the remainder — adds clock state to the registry
for a sub-second cosmetic difference.

---

## 5. Trigger paths

### 5.1 Saga (`libs/atlas-saga` + `atlas-saga-orchestrator`)

Mirrors `PlayJukebox` end to end:

- `libs/atlas-saga/model.go`: `SetBackEffect Action = "set_back_effect"`,
  `ClearBackEffect Action = "clear_back_effect"`.
- `libs/atlas-saga/payloads.go`: `SetBackEffectPayload` (world/channel/map/instance
  + effect/fieldId/pageId/duration), `ClearBackEffectPayload`
  (world/channel/map/instance only).
- `libs/atlas-saga/unmarshal.go`: two `case` arms.
- Orchestrator: aliases in `saga/model.go` (+ its payload-decode `case`),
  dispatch arms in `saga/handler.go`, registration in `saga/event_acceptance.go`,
  and `map_command` processor methods + command providers.

This is what makes the feature reachable from `atlas-map-actions` map-entry
scripts and NPC/quest conversations with **no change to those services** — they
already emit saga steps.

### 5.2 GM chat commands (`atlas-messages`)

`services/atlas-messages/atlas.com/messages/command/map/back_effect.go`, modelled
on `weather.go` — regex match, `c.Gm()` gate, produce onto `COMMAND_TOPIC_MAP`,
register in `commands.go`.

Argument shape follows §1.1:

```
@backeffect <pageId> <effect> [durationMs]   # effect: 0 show, 1 hide; duration default 0
@clearbackeffect
```

**Decision — the GM command fills `fieldId` from the invoking character's own
field**, not from an argument. The client never reads it (§1.1), so exposing it as
a GM-typed parameter would be surface area with no observable effect.

---

## 6. Test strategy

| Layer | Tests |
|---|---|
| `libs/atlas-packet` | Per-version byte fixtures with `packet-audit:verify` for both ops (14 cells); round-trip `Encode`→`Decode`. |
| `atlas-maps` registry | Set/replace-same-page/append-new-page/clear-all; tenant isolation (two tenants, same field). |
| `atlas-maps` consumer | Both arms: registry mutated + event produced; invalid `Effect` rejected; clear-on-empty still emits. |
| `atlas-maps` REST | 200 with entries, 200 with empty array. |
| `atlas-channel` consumer | Writer invoked for sessions in the field on each event (the `consumer_test.go` pattern). |
| `atlas-channel` replay | Active back-effects announced to the entering session with `Duration = 0`; **lookup failure does not block map entry** (fail-open assertion). |
| Orchestrator | Both handlers produce the command; invalid-payload rejection, per `TestHandlePlayJukebox_InvalidPayload`. |
| `atlas-messages` | Command produced for a GM; non-GM rejected. |

Cross-service seam (per CLAUDE.md "Done means verified"): `BACK_EFFECT_SET` is
traced by hand from the `atlas-maps` producer into the `atlas-channel` consumer,
and a channel-side test asserts the new contract.

---

## 7. Sequencing

The packet layer is a hard prerequisite for nothing except the writer bodies, but
the *derivation* (§1.3) is a prerequisite for the command/event/REST field names,
so it goes first.

1. Per-version derivation + `structures/<version>.md` (incl. the six ⬜ absence proofs).
2. Codecs + fixtures + evidence + templates + matrix regeneration.
3. `atlas-maps`: registry → processor → producer → consumer arms → REST.
4. `atlas-channel`: writers → status-event arms → REST client → replay.
5. `libs/atlas-saga` + orchestrator handlers.
6. `atlas-messages` GM commands.
7. Flagless `tools/verify.sh`, then code review, then PR.

Steps 3–6 are independent of each other once the field names from step 1 are
fixed, and can be planned as parallel units.

---

## 8. Risks

| Risk | Mitigation |
|---|---|
| A version diverges from the 4-field layout | Every version is decompiled in step 1 before any codec is written; a divergence becomes a `MajorAtLeast` gate, not a surprise at fixture time. |
| jms_v185 differs structurally (PRD Open Q4) | Same. If JMS diverges materially, its cells are handled as a separate gated shape — never by assuming GMS. |
| `nEffect` is not a two-value enum on some version | The rejection at §3.2 is on the *derived* set of accepted values; if a version accepts a third value, widen the validation with the evidence, do not guess. |
| Registry growth on destroyed instances | Accepted and documented (§3.4). |
| `SET_MAP_OBJECT_VISIBLE` looks adjacent and gets pulled in | Explicitly out of scope (§2.4); recorded as a follow-up. |

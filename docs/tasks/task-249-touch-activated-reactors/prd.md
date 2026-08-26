# Touch-Activated Reactors — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-21
---

## 1. Overview

MapleStory reactors can be configured to activate when a character walks into their
bounding area rather than when the character attacks them. The WZ data expresses this
with an `activateByTouch` flag on the reactor template's `info` node. The client
detects the overlap locally (`CReactorPool::FindTouchReactorAroundLocalUser`) and
notifies the server with a `TOUCHING_REACTOR` packet; the server then advances the
reactor's state exactly as it would for a hit.

Atlas has no touch-activation path at any layer. There is no `TOUCHING_REACTOR` codec
in `libs/atlas-packet`, no handler in `atlas-channel`, no routing entry in any of the
eleven seed templates, and no touch entry point in `atlas-reactors` — its only
activation method is `ProcessorImpl.Hit`. The `activateByTouch` flag is read by
`atlas-data` (`services/atlas-data/atlas.com/data/reactor/reader.go:80`) but only to
decide whether each state re-reads its own `tl`/`br` bounds; the flag itself is never
placed on the REST model, so `atlas-reactors` cannot see it. The result is that every
reactor whose only activation method is touch is permanently inert.

This task closes the gap end to end: derive and implement the `TOUCHING_REACTOR`
serverbound codec across every client version that defines it, wire a handler into
`atlas-channel` and every applicable seed template, expose `activateByTouch` through
`atlas-data` into the `atlas-reactors` reactor-data model, and add a touch activation
path in `atlas-reactors` that advances reactor state and emits a distinct `TOUCH`
command to `atlas-reactor-actions`.

## 2. Goals

Primary goals:

- A character walking into the bounding area of a reactor whose template sets
  `activateByTouch` causes that reactor to advance its state, exactly as a hit would,
  without the character performing an attack.
- The `TOUCHING_REACTOR` serverbound packet has a codec in `libs/atlas-packet` for
  every client version that defines the opcode, with each corresponding coverage-matrix
  cell promoted to verified.
- The `activateByTouch` flag is available to `atlas-reactors` as first-class reactor
  data and is enforced server-side: a touch request against a reactor whose template
  does not set the flag is rejected.
- Reactor scripts running in `atlas-reactor-actions` can distinguish a touch
  activation from a skill hit.
- The existing hit path (`REACTOR_HIT` / `ProcessorImpl.Hit`) is behaviourally
  unchanged.

Non-goals:

- Server-side movement-derived touch detection. Activation is driven exclusively by
  the client's `TOUCHING_REACTOR` packet. `atlas-reactors` will not subscribe to
  character movement or sweep reactor bounds on every move.
- Authoring or fixing reactor scripts for the affected GPQ reactor templates. This
  task delivers the mechanism; script content is separate work.
- Any `atlas-ui` change.
- Changing how `atlas-data` derives reactor `tl`/`br` bounds. The existing
  `loadArea` logic at `reader.go:111` stays as is; this task only additionally
  surfaces the flag.

## 3. User Stories

- As a player, I want to walk into a touch-activated reactor and have it fire, so that
  content built around walk-over triggers is completable.
- As a reactor script author, I want a touch activation to arrive as a distinguishable
  event, so that a script can respond differently to being walked into than to being
  attacked.
- As a server operator, I want a client that claims to be touching a reactor to be
  validated against the reactor's real template flag and real map position, so that a
  modified client cannot trigger arbitrary reactors.
- As a packet maintainer, I want the `TOUCHING_REACTOR` row in the coverage matrix to
  be verified rather than interpolated, so that future version bring-ups inherit a
  known-good codec.

## 4. Functional Requirements

### 4.1 Packet derivation and version coverage

- **FR-1.** The `TOUCHING_REACTOR` wire layout MUST be derived from the client binary
  (`CReactorPool::FindTouchReactorAroundLocalUser` and its send path), following
  `docs/packets/IMPLEMENTING_A_PACKET.md`. The field order MUST NOT be guessed from
  Cosmic's `TouchReactorHandler` or from general MapleStory knowledge; Cosmic may be
  used as a cross-check only.
- **FR-2.** The opcode for each version MUST come from the per-version registry under
  `docs/packets/registry/`. Known values already recorded: gms_v83 `0x0CE` (206),
  gms_v84 `0x0CE`, gms_v87 `0x0DB`, gms_v95 `0x0FA`, jms_v185 `0x0D9`. The ops CSV
  (`docs/packets/MapleStory Ops - ServerBound.csv:588`) additionally records gms_v92
  `0x0F3` and gms_v111 `0x169`.
- **FR-3.** The `n-a` markings for `gms_v48`, `gms_v61`, `gms_v72`, and `gms_v79` in
  `docs/packets/audits/support/*.md` MUST be re-verified against the actual client
  binaries, not accepted as given. The ops CSV has no columns for those four versions,
  so their `n-a` status is an interpolation rather than a measurement. Any version
  found to define the opcode MUST be treated as in scope for FR-4 and FR-5, and its
  support file corrected.
- **FR-4.** A codec MUST exist in `libs/atlas-packet/reactor/serverbound/` for every
  version determined to define the opcode, following the conventions established by
  the sibling `hit.go`: an immutable struct with both `Encode` and `Decode`, and
  version-gated fields expressed with the `MajorAtLeast` idiom rather than raw
  numeric comparisons.
- **FR-5.** Each `TOUCHING_REACTOR` × version cell that is in scope MUST be promoted
  to verified per `docs/packets/audits/VERIFYING_A_PACKET.md` — byte-fixture test with
  a `packet-audit:verify` marker, pinned evidence record, regenerated matrix — with
  the three artifacts committed together. Versions confirmed not to define the opcode
  remain `n-a` with the support file stating that this was measured.
- **FR-6.** No wire change may be made to any packet on an already-verified version as
  a side effect of this work.

### 4.2 Channel handling

- **FR-7.** `atlas-channel` MUST expose a `TouchReactorHandleFunc` in
  `services/atlas-channel/atlas.com/channel/socket/handler/`, following the shape of
  the existing `reactor_hit.go`: decode the request, log at debug, and delegate to the
  channel-side reactor processor.
- **FR-8.** The handler MUST be registered under a stable handler name (`TouchReactorHandle`)
  and routed in every seed template under
  `services/atlas-configurations/seed-data/templates/` whose version defines the
  opcode, at that version's opcode. Templates for versions that do not define the
  opcode MUST NOT gain an entry.
- **FR-9.** The channel-side reactor processor MUST gain a `Touch` method that emits a
  `TOUCH` command onto the reactor command topic. It MUST NOT reuse the `HIT` command.

### 4.3 Reactor data plumbing

- **FR-10.** `atlas-data`'s reactor `RestModel`
  (`services/atlas-data/atlas.com/data/reactor/rest.go`) MUST gain an
  `activateByTouch` boolean attribute populated from the value already read at
  `reader.go:80`. The existing `loadArea` use at `reader.go:111` is unchanged.
- **FR-11.** `atlas-reactors`' reactor-data model
  (`services/atlas-reactors/atlas.com/reactors/reactor/data/model.go`) MUST expose the
  flag via an `ActivateByTouch() bool` accessor, plumbed through the data REST model
  and builder alongside the existing `TL()`/`BR()` fields.
- **FR-12.** A reactor template that omits `activateByTouch` MUST decode as `false`.
  Existing persisted or cached reactor data with no such field MUST NOT fail to
  decode.

### 4.4 Touch activation in atlas-reactors

- **FR-13.** `atlas-reactors` MUST consume a `TOUCH` command on the reactor command
  topic and route it to a new `ProcessorImpl.Touch(reactorId, characterId)` entry
  point.
- **FR-14.** `Touch` MUST reject the request, with a logged reason and no state
  change, when any of the following hold:
  - the reactor id does not resolve in the registry for the tenant;
  - the reactor's template does not set `activateByTouch`;
  - the requesting character's last known position is outside the reactor's
    `[TL, BR]` axis-aligned bounding box, evaluated against the reactor's own `x`/`y`
    origin. *(See Open Question OQ-2 on the source of the character position.)*
- **FR-15.** On acceptance, `Touch` MUST perform the same state progression as `Hit`:
  cancel any pending state timeout, select the next state from the current state's
  event list, honour the `persistsAtEndState` / `isTerminalState` rules, re-arm the
  state timer, and emit the reactor status event.
- **FR-16.** State-event selection under touch MUST treat the touch as satisfying the
  activation predicate regardless of the event's `activeSkills` list. The
  `activateByTouch` template flag is the gate; `activeSkills` constrains only the hit
  path. Concretely, the `len(event.ActiveSkills()) == 0 || containsSkill(...)`
  condition at `processor.go:191` must not be reused verbatim for touch — under touch,
  the first event of the current state matches.

  *Rationale:* the affected GPQ reactor templates use type-5/6/7 skill-gated state
  events (`docs/tasks/task-019-reactor-type-semantics/prd.md:32`). Reusing the hit
  predicate would leave `nextState == -1`, fall through to `TriggerAndDestroy`, and
  *destroy* the reactor on touch instead of advancing it.
- **FR-17.** `Touch` MUST emit a `TOUCH` command to `atlas-reactor-actions` on
  `COMMAND_TOPIC_REACTOR_ACTIONS` — a new command type alongside the existing `HIT`
  and `TRIGGER` — carrying the character id. It MUST NOT emit a `HIT` actions command.
- **FR-18.** Repeated touch requests for the same character and reactor MUST NOT
  advance the state more than once per state occupancy. A touch arriving for a reactor
  already past the state it was sent for is a no-op, not an error.
- **FR-19.** The existing `Hit` path MUST remain behaviourally identical, including
  its skill-gating predicate and its `TriggerAndDestroy` fall-throughs.

### 4.5 Affected reactor templates

- **FR-20.** The set of templates carrying `activateByTouch` MUST be enumerated from
  the WZ data during design rather than taken from prose. Two prior enumerations
  disagree: `docs/TODO.md:280` and `docs/tasks/task-019-reactor-type-semantics/prd.md:32`
  list nine (`6109013`, `6109014`, `6109021`–`6109027`), while another enumeration
  lists ten, adding `2406000`. The design phase MUST resolve this against
  `Reactor.wz` and record the authoritative list.

## 5. API Surface

### 5.1 Serverbound packet

`TOUCHING_REACTOR` — fname `CReactorPool::FindTouchReactorAroundLocalUser`.
Field layout to be derived per FR-1; not specified here, because guessing it would
violate the repository's grounding rule.

### 5.2 Kafka — reactor command topic (`COMMAND_TOPIC_REACTOR`)

New command type on the existing `Command[E]` envelope in
`services/atlas-reactors/atlas.com/reactors/reactor/kafka.go` and its channel-side
counterpart:

```
CommandTypeTouch = "TOUCH"

type TouchCommandBody struct {
    ReactorId   uint32 `json:"reactorId"`
    CharacterId uint32 `json:"characterId"`
}
```

The envelope's existing `WorldId` / `ChannelId` / `MapId` / `Instance` fields carry
the field scoping, as they do for `HIT`.

### 5.3 Kafka — reactor actions topic (`COMMAND_TOPIC_REACTOR_ACTIONS`)

New command type on the existing `reactorActionsCommand[E]` envelope:

```
CommandTypeActionsTouch = "TOUCH"

type touchActionsBody struct {
    CharacterId uint32 `json:"characterId"`
}
```

This mirrors the existing `triggerActionsBody`. Existing `HIT` and `TRIGGER` consumers
in `atlas-reactor-actions` are unaffected; an unrecognised type must continue to be
ignored rather than error.

### 5.4 REST — atlas-data reactor resource

The reactor resource gains one attribute:

```
"activateByTouch": false
```

Additive and defaulted, so existing consumers are unaffected.

### 5.5 Error cases

Touch is fire-and-forget over Kafka; there is no synchronous error channel to the
client. All FR-14 rejections are logged and dropped. No clientbound error packet is
introduced.

## 6. Data Model

No database migration. Reactor state lives in the in-memory registry
(`services/atlas-reactors/atlas.com/reactors/reactor/registry.go`) and reactor
template data is fetched from `atlas-data`.

Changes:

- `atlas-data` reactor `RestModel` gains `ActivateByTouch bool` with json tag
  `activateByTouch`.
- `atlas-reactors` `reactor/data.Model` gains an unexported `activateByTouch bool`
  field with an `ActivateByTouch()` accessor, set through the package's existing
  builder. No new entity, no new relationship, no new constraint.

Multi-tenancy is unchanged: reactor lookups already resolve the tenant via
`tenant.MustFromContext` and the registry is keyed by `tenant.Model`.

## 7. Service Impact

| Service / library | Change |
|---|---|
| `libs/atlas-packet` | New `TOUCHING_REACTOR` serverbound codec in `reactor/serverbound/`, with byte-fixture tests per verified version. |
| `atlas-channel` | New `socket/handler/touch_reactor.go`; new `Touch` method on `reactor.Processor` plus the `TOUCH` command producer; handler registration. |
| `atlas-configurations` | `TouchReactorHandle` routing entry added to each seed template whose version defines the opcode. |
| `atlas-data` | `activateByTouch` added to the reactor REST model and populated in `reader.go`. |
| `atlas-reactors` | `TOUCH` command consumer; `ProcessorImpl.Touch`; `activateByTouch` plumbed through `reactor/data`; new `TOUCH` actions command producer. |
| `atlas-reactor-actions` | Consumes the new `TOUCH` actions command type. Verify during design whether its dispatch currently rejects or ignores unknown types. |
| `docs/packets` | Registry, support files, evidence records, `status.json` and regenerated `STATUS.md` for every in-scope cell. |
| `docs/TODO.md` | Remove the deferred `activateByTouch` item at line 280 once implemented. |

`atlas-ui` is unaffected.

## 8. Non-Functional Requirements

- **Security.** The `TOUCHING_REACTOR` packet is client-asserted. The server must
  never trust it as proof of proximity: FR-14's template-flag check and bounds check
  are the anti-cheat surface, and both are mandatory. A modified client must not be
  able to trigger a reactor it is not standing in, nor a reactor that is not
  touch-activated at all.
- **Performance.** Touch handling must add no per-tick or per-move cost. The only work
  happens on receipt of a client packet, and it is bounded by a single reactor lookup
  and one AABB comparison. No map-wide reactor sweep is permitted.
- **Traffic.** The handler must tolerate a client that sends `TOUCHING_REACTOR`
  repeatedly while the character remains inside the area. FR-18's idempotence is what
  keeps that from being amplified into repeated state transitions or repeated script
  invocations.
- **Observability.** Every FR-14 rejection logs the reactor id, character id, and the
  specific reason. Acceptance logs the state transition at debug, matching the
  existing `Hit` logging in `processor.go`.
- **Multi-tenancy.** Reactor resolution, registry access, and both Kafka producers
  operate under the request's tenant context, consistent with the existing hit path.
- **Backward compatibility.** Tenants on versions that do not define the opcode see no
  behavioural change. Reactor data lacking `activateByTouch` decodes as `false`,
  meaning every currently-working reactor keeps working and no reactor becomes
  touch-activated by accident.

## 9. Open Questions

- **OQ-1.** Does `TOUCHING_REACTOR` carry only the reactor object id, or also a
  touching/leaving boolean or a position? This determines whether FR-18's idempotence
  is enforced from the packet or purely server-side. Resolve by deriving the layout
  (FR-1) before designing the handler.
- **OQ-2.** What is the authoritative source of the character's current position for
  the FR-14 bounds check — session state in `atlas-channel` (checked before the
  command is emitted) or a lookup from `atlas-reactors`? Checking in `atlas-channel`
  is cheaper and keeps `atlas-reactors` free of a character dependency, but puts an
  anti-cheat check on the edge rather than at the authority. Decide in design.
- **OQ-3.** Do any of the four versions currently marked `n-a` (`gms_v48`, `gms_v61`,
  `gms_v72`, `gms_v79`) actually define the opcode? The user's expectation is that
  v72 and later do. This directly sets the codec's version spread and the template
  routing set (FR-3).
- **OQ-4.** Is `2406000` genuinely an `activateByTouch` template, or is the ten-item
  list in the research doc an overcount relative to the nine-item list in
  `docs/TODO.md`? (FR-20.)
- **OQ-5.** Does `atlas-reactor-actions` currently ignore or reject an unrecognised
  actions command type? If it rejects, the `TOUCH` consumer must land before or with
  the producer to avoid an error-log flood.
- **OQ-6.** Should a touch on a reactor in a state with *no* events fall through to
  `TriggerAndDestroy`, as `Hit` does at `processor.go:187`, or be a no-op? Destroying
  on touch is a larger side effect than destroying on a deliberate attack.

## 10. Acceptance Criteria

Packet layer:

- [ ] The `TOUCHING_REACTOR` layout is derived from the client binary and the
      derivation is recorded in the task's evidence artifacts.
- [ ] `gms_v48`, `gms_v61`, `gms_v72`, `gms_v79` have been checked against their actual
      client binaries, and each support file states the measured result rather than an
      interpolated `n-a`.
- [ ] A codec exists in `libs/atlas-packet/reactor/serverbound/` with both `Encode`
      and `Decode`, version-gated using `MajorAtLeast`.
- [ ] Every in-scope `TOUCHING_REACTOR` × version cell reads verified in
      `docs/packets/audits/STATUS.md`, each backed by a `packet-audit:verify` byte
      fixture and a pinned evidence record.
- [ ] `git diff` shows no wire change to any already-verified packet on any version.

Channel layer:

- [ ] `TouchReactorHandleFunc` exists and is registered.
- [ ] Each seed template for a version that defines the opcode routes
      `TouchReactorHandle` at that version's opcode; templates for versions that do
      not, have no such entry.

Data layer:

- [ ] The `atlas-data` reactor resource emits `activateByTouch`, and a test asserts it
      is `true` for a template that sets the flag and `false` for one that does not.
- [ ] `atlas-reactors`' `reactor/data.Model` exposes `ActivateByTouch()`, and reactor
      data with the field absent decodes as `false`.

Reactor behaviour (each with a test):

- [ ] A `TOUCH` command for a reactor whose template sets `activateByTouch`, from a
      character inside `[TL, BR]`, advances the reactor's state.
- [ ] The same touch against a reactor whose template does *not* set the flag is
      rejected with no state change.
- [ ] A touch from a character outside `[TL, BR]` is rejected with no state change.
- [ ] A touch against a state whose events are skill-gated (type 5/6/7, non-empty
      `activeSkills`) **advances the state** and does **not** destroy the reactor —
      this is the FR-16 regression guard.
- [ ] Repeated touches for the same character and reactor state advance the state
      exactly once.
- [ ] A `TOUCH` actions command is emitted to `COMMAND_TOPIC_REACTOR_ACTIONS`, and no
      `HIT` actions command is emitted for a touch.
- [ ] Existing `Hit` tests in
      `services/atlas-reactors/atlas.com/reactors/reactor/processor_test.go` pass
      unchanged.

Gate:

- [ ] The authoritative `activateByTouch` template list is recorded in the task folder,
      resolving the nine-vs-ten discrepancy.
- [ ] The deferred item at `docs/TODO.md:280` is removed.
- [ ] Flagless `tools/verify.sh` exits 0.
- [ ] Code review completed before the PR is opened.

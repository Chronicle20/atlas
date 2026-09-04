# Map Back-Effects (SET_BACK_EFFECT / CLEAR_BACK_EFFECT) — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-28
---

## 1. Overview

MapleStory clients support server-driven changes to a field's *background layer* — the
parallax art behind the tiles. The client entry points are
`CMapLoadable::OnSetBackEffect` and `CMapLoadable::OnClearBackEffect`, routed through
`CMapLoadable::OnPacket` (which `CField::OnPacket` dispatches into). Atlas has never
implemented either: `docs/packets/audits/STATUS.md:210,231` shows both ops ❌ on every
version where the client has them, there is no codec in `libs/atlas-packet`, no writer
registered in `atlas-channel`, and no service that can trigger one.

This task closes the gap end to end. It adds the two clientbound codecs to
`libs/atlas-packet/field/clientbound/`, promotes all 14 ❌ coverage-matrix cells to
verified, re-confirms the ⬜ (version-absent) cells with fresh IDB evidence, and wires a
trigger path so back-effects are actually reachable: a `COMMAND_TOPIC_MAP` command →
`atlas-maps` (which holds per-field back-effect state) → an `EVENT_TOPIC_MAP_STATUS`
event → `atlas-channel` writes the packet. Because a back-effect is a *set-until-cleared*
state rather than a timed effect, `atlas-maps` also exposes the active state over REST so
`atlas-channel` can replay it to a character entering the map.

The design deliberately mirrors the **jukebox** feature, which is the closest existing
end-to-end analogue in this repo: a per-field in-memory registry
(`services/atlas-maps/atlas.com/maps/map/jukebox/registry.go`), a command consumer, a
status-event producer, a REST resource (`.../jukebox/resource.go:26`), and a
channel-side replay on map entry (`services/atlas-channel/.../kafka/consumer/map/consumer.go:1156`
`announceActiveJukebox`). Reusing that shape keeps this feature inside patterns the repo
already reviews and tests.

## 2. Goals

Primary goals:
- Ship `SetBackEffect` and `ClearBackEffect` clientbound codecs in
  `libs/atlas-packet/field/clientbound/`, each with both `Encode` and `Decode`, version
  gates where the client layout diverges, and byte-fixture tests per applicable version.
- Drive all 14 ❌ cells for `SET_BACK_EFFECT` and `CLEAR_BACK_EFFECT` to ✅ verified, and
  re-confirm every ⬜ cell as IDB-evidenced VERSION-ABSENT.
- Make back-effects triggerable: a saga step (usable by map-entry scripts and
  NPC/quest conversations) and a GM chat command, both flowing through `atlas-maps`.
- Make back-effects correct for late joiners: a character entering a field with an active
  back-effect receives it.

Non-goals:
- Deciding *which* maps use back-effects. The claim that specific event maps use them is
  unverified general MapleStory knowledge; no map content, WZ-derived table, or seed data
  is authored by this task.
- Any serverbound packet. Both ops are clientbound only.
- Client-side layer semantics beyond what the decompiled read order proves.
- Reclassifying a live ❌ cell to ⬜ to "close" it — only genuine IDB-evidenced absence.
- Back-effect state surviving a service restart (in-memory registry only, matching
  jukebox).

## 3. User Stories

- As a content author, I want a map-entry script or NPC conversation to set a map's
  background layer and later clear it, so event maps can change appearance in response to
  progress.
- As a GM, I want an in-game chat command to set and clear a back-effect on my current
  map, so I can stage and test an event without a deploy.
- As a player entering a map where a back-effect is already active, I want to see the same
  background as everyone else already in the map.
- As a packet-coverage owner, I want `SET_BACK_EFFECT` and `CLEAR_BACK_EFFECT` fully ✅/⬜
  in the matrix, with pinned tier-1 evidence and audit reports, so two more rows leave the
  ❌ backlog with no silent gaps.

## 4. Functional Requirements

### FR-1 — Derive the wire layout (prerequisite, no invention)

The field list for both packets is **currently unknown to this task** and MUST be derived
before any codec is written. Do not infer it from other MapleStory servers.

- Decompile `CMapLoadable::OnSetBackEffect` and `CMapLoadable::OnClearBackEffect` per
  applicable version via `ida-pro-mcp` (or the checked-in export under
  `docs/packets/ida-exports/`), following `docs/packets/IMPLEMENTING_A_PACKET.md` §0–4.
  GMS v95.1 is the reference IDB; other versions are checked for divergence.
- The only recorded hint today is the v84 registry note
  (`docs/packets/registry/gms_v84.yaml:889`): `OnSetBackEffect` is
  `BackEffect::Decode` followed by two near-identical branches keyed on a state value
  (0 / 1) that each resolve a layer from the field's `ZMap` and adjust alpha. Treat this
  as a lead to confirm, not a finding. `BackEffect::Decode` itself must be decompiled —
  the codec's fields are whatever it reads, in that order.
- Record the ordered field list, per-version deltas, and export addresses in
  `docs/tasks/task-281-map-back-effects/structures/<version>.md`.
- Step 0 of `IMPLEMENTING_A_PACKET.md` still applies: confirm no existing codec already
  covers this shape before adding a new one.

### FR-2 — Packet codecs

- New files `libs/atlas-packet/field/clientbound/set_back_effect.go` and
  `clear_back_effect.go`, following the existing conventions in that package (see
  `field_obstacle_on_off.go`): an immutable struct with unexported fields, a `New…`
  constructor, accessors, `Operation()`, `String()`, `Encode`, `Decode`, and a
  `// packet-audit:fname` marker naming the client function.
- Writer constants `SetBackEffectWriter` and `ClearBackEffectWriter`.
- Version-divergent fields use the `MajorAtLeast` gate idiom, never a raw `> N`
  comparison.
- No wire change to any already-verified packet.

### FR-3 — Coverage matrix

Target cells, read from `docs/packets/audits/STATUS.md:210,231`:

| Op | v48 | v61 | v72 | v79 | v83 | v84 | v87 | v92 | v95 | JMS185 |
|---|---|---|---|---|---|---|---|---|---|---|
| SET_BACK_EFFECT | ⬜ | ⬜ | 0x075 ❌ | 0x079 ❌ | 0x080 ❌ | 0x083 ❌ | 0x088 ❌ | 0x08F ❌ | 0x090 ❌ | 0x07E ❌ |
| CLEAR_BACK_EFFECT | ⬜ | ⬜ | ⬜ | ⬜ | 0x082 ❌ | 0x085 ❌ | 0x08A ❌ | 0x091 ❌ | 0x092 ❌ | 0x080 ❌ |

- All 14 ❌ cells → ✅, each with a byte-fixture test carrying a `// packet-audit:verify`
  marker, a pinned tier-1 evidence record, and an audit report.
- All 6 ⬜ cells (`SET_BACK_EFFECT` on v48/v61; `CLEAR_BACK_EFFECT` on v48/v61/v72/v79)
  → re-confirmed VERSION-ABSENT with a freshly pinned evidence record derived by reading
  each older IDB's `CMapLoadable::OnPacket` switch. Existing prose in
  `docs/tasks/task-113-gms-legacy-versions/v72-packet-delta.md:179` and
  `v79-packet-delta.md:178` is a lead, not the evidence.
- Both ops routed in every applicable seed template.
- `go run ./tools/packet-audit matrix --check` (plus `fname-doc` and `operations --check`)
  exits 0.

### FR-4 — `atlas-maps`: back-effect state

New package `services/atlas-maps/atlas.com/maps/map/backeffect/`, modelled on
`map/jukebox/`:

- A tenant+field-keyed in-memory registry holding the active back-effect(s) for a field.
  Unlike jukebox there is **no expiry** — entries persist until explicitly cleared.
- Consume `SET_BACK_EFFECT` and `CLEAR_BACK_EFFECT` commands on `COMMAND_TOPIC_MAP`.
- On set: record the state, then emit a `BACK_EFFECT_SET` status event on
  `EVENT_TOPIC_MAP_STATUS`.
- On clear: remove the state, then emit `BACK_EFFECT_CLEAR`. Clearing a field with no
  active back-effect is a no-op that still emits (so a desynced client resets), and logs
  at debug.
- Whether a field can hold more than one simultaneous back-effect is determined by
  FR-1: if the derived layout keys the effect by a layer/slot value, the registry is keyed
  by field+slot and `CLEAR_BACK_EFFECT` clears accordingly; if the packet carries no
  slot, one entry per field. **This is settled by the decompile, not by preference.**
- REST: `GET /{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/backEffect`
  returning the active state (JSON:API, matching `map/jukebox/resource.go`), 404 when none.
- State is dropped when the field's instance is destroyed, matching how other per-field
  registries are reaped in this service.

### FR-5 — `atlas-channel`: emit and replay

- Register `SetBackEffectWriter` / `ClearBackEffectWriter` in `main.go` and add the
  corresponding `socket/writer/` body functions.
- Handle `BACK_EFFECT_SET` / `BACK_EFFECT_CLEAR` in the map status consumer, broadcasting
  to sessions in the field via `ForSessionsInMap`.
- On `CHARACTER_ENTER`, query `atlas-maps` for the field's active back-effect and announce
  it to that one session — the `announceActiveJukebox` pattern
  (`services/atlas-channel/.../kafka/consumer/map/consumer.go:1156`), including its
  fail-open behaviour: an unreachable `atlas-maps` costs the background, not the map entry.

### FR-6 — Saga step producer

- Add `SetBackEffect` and `ClearBackEffect` action types and payloads to
  `libs/atlas-saga`, mirroring `PlayJukebox` / `PlayJukeboxPayload`.
- Add handlers in `atlas-saga-orchestrator` (`saga/handler.go` dispatch + payload
  decode in `saga/model.go` + `saga/event_acceptance.go` registration) that call new
  `map_command` processor methods producing onto `COMMAND_TOPIC_MAP`.
- This is what makes the feature reachable from `atlas-map-actions` map-entry scripts and
  from NPC/quest conversation scripts, which already emit saga steps.

### FR-7 — GM chat command

- Add a command producer in `services/atlas-messages/atlas.com/messages/command/map/`,
  modelled on `weather.go`: GM-gated (`c.Gm()`), regex-matched, producing the
  `COMMAND_TOPIC_MAP` command for the invoking character's field.
- Two commands: one to set a back-effect and one to clear it. Exact argument shape follows
  the FR-1 derived field list.

## 5. API Surface

### Kafka — `COMMAND_TOPIC_MAP` (new command types)

Reusing the existing `Command[E]` envelope
(`services/atlas-maps/atlas.com/maps/kafka/message/map/command.go:17`):

```
CommandTypeSetBackEffect   = "SET_BACK_EFFECT"
CommandTypeClearBackEffect = "CLEAR_BACK_EFFECT"
```

Body field names follow the FR-1 derived packet fields; the envelope
(`transactionId`, `worldId`, `channelId`, `mapId`, `instance`, `type`, `body`) is
unchanged.

### Kafka — `EVENT_TOPIC_MAP_STATUS` (new event types)

Reusing the existing `StatusEvent[E]` envelope
(`services/atlas-maps/atlas.com/maps/kafka/message/map/kafka.go:22`):

```
EventTopicMapStatusTypeBackEffectSet   = "BACK_EFFECT_SET"
EventTopicMapStatusTypeBackEffectClear = "BACK_EFFECT_CLEAR"
```

The message-type constants and body structs must be added identically in every service
that participates: `atlas-maps`, `atlas-channel`, `atlas-saga-orchestrator`,
`atlas-messages` (command constants only).

### REST — `atlas-maps`

```
GET /{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/backEffect
  200 → JSON:API resource, type "backEffect", attributes = the active state
  404 → no active back-effect on that field
```

Errors follow the existing `rest.RegisterHandler` conventions in this service.

### Wire

Two clientbound packets only. Opcodes per version are already in the registry (FR-3
table); the payload layout is FR-1's output.

## 6. Data Model

No database tables and no migrations. State is a process-local, tenant-scoped in-memory
registry in `atlas-maps`:

- Key: `{Tenant tenant.Model, Field field.Model}` (plus a slot/layer discriminator if
  FR-1 proves the packet carries one) — the `jukebox.FieldKey` shape.
- Value: the back-effect's derived fields. **No `ExpiresAt`** — this state is cleared
  explicitly, not by a reaper task.
- Multi-tenancy: the tenant comes from `tenant.MustFromContext`, exactly as
  `jukebox/processor.go` does; no cross-tenant key collision is possible.
- Consequence, accepted: back-effects do not survive an `atlas-maps` restart. Same
  trade-off the jukebox already makes.

## 7. Service Impact

| Service / lib | Change |
|---|---|
| `libs/atlas-packet` | New `field/clientbound/set_back_effect.go`, `clear_back_effect.go` + fixture tests per version |
| `libs/atlas-saga` | New `SetBackEffect` / `ClearBackEffect` actions and payloads |
| `atlas-maps` | New `map/backeffect/` package (registry, processor, producer, REST resource); command consumer arms |
| `atlas-channel` | Writer registration + `socket/writer/` bodies; map status consumer arms; `CHARACTER_ENTER` replay; new `backeffect/` REST client package |
| `atlas-saga-orchestrator` | Two saga handlers + `map_command` processor methods + event-acceptance registration |
| `atlas-messages` | GM set/clear chat commands |
| `docs/packets/` | Registry `packet:` links, evidence records, audit reports, regenerated `STATUS.md` / `status.json`, seed-template routes |

`atlas-map-actions` and the NPC/quest script services need **no code change** — they reach
the feature through the new saga action types.

## 8. Non-Functional Requirements

- **Multi-tenancy:** every registry key, Kafka message, and REST call is tenant-scoped via
  the existing header/context propagation. No global back-effect state.
- **Performance:** a set/clear is a map write plus one broadcast over sessions already in
  the field; the character-enter replay is one REST call per map entry, on the same path
  that already makes the jukebox call. No new per-tick work and no reaper goroutine.
- **Resilience:** the character-enter replay fails open — a failed lookup logs and returns
  without blocking map entry.
- **Security:** the GM command is gated on `c.Gm()`, matching `weather.go:31`. The saga
  path is server-internal; there is no client-originated way to set a back-effect.
- **Observability:** set/clear log at debug with map id, instance, and the effect
  identity; broadcast failures log at error, matching the jukebox handlers.
- **Version safety:** a version that lacks the op must not route it; sending is gated by
  the seed-template route, and no already-verified packet's bytes change.

## 9. Open Questions

1. **Packet field list (blocking implementation, resolvable in-repo).** Unknown until
   FR-1's decompile. Everything downstream — command body, event body, registry key, REST
   attributes, GM command arguments — is shaped by it. This is a producible prerequisite,
   not an external blocker: the IDBs and exports are checked in.
2. **One back-effect per field, or several?** Determined by whether the derived layout
   carries a layer/slot discriminator (FR-4).
3. **Does `CLEAR_BACK_EFFECT` carry a payload at all,** or is it a bare opcode? The v84
   registry note describes only the set path. To be answered by the same decompile.
4. **jms_v185 divergence.** JMS versions have diverged from GMS elsewhere; whether the
   JMS layout matches GMS 0x090 is unverified until the JMS IDB is read.
5. **Instance teardown hook.** Which existing `atlas-maps` teardown path the registry
   should hook for reaping is a design-phase question; jukebox relies on expiry, which
   this feature does not have.

## 10. Acceptance Criteria

Packets:
- [ ] `structures/<version>.md` records the decompiled read order and export address for
      `OnSetBackEffect` / `OnClearBackEffect` on every applicable version.
- [ ] `set_back_effect.go` and `clear_back_effect.go` exist with `Encode` + `Decode`,
      `packet-audit:fname` markers, and version gates via `MajorAtLeast`.
- [ ] A byte-fixture test with a `packet-audit:verify` marker exists for each of the 14
      cells.
- [ ] All 14 cells read ✅ in a regenerated `STATUS.md`; all 6 ⬜ cells carry a fresh
      VERSION-ABSENT evidence record.
- [ ] Both ops routed in every applicable seed template.
- [ ] `go run ./tools/packet-audit matrix --check`, `fname-doc`, and
      `operations --check` each exit 0.

Trigger path:
- [ ] `atlas-maps` consumes `SET_BACK_EFFECT` / `CLEAR_BACK_EFFECT` on
      `COMMAND_TOPIC_MAP`, updates the registry, and emits `BACK_EFFECT_SET` /
      `BACK_EFFECT_CLEAR` on `EVENT_TOPIC_MAP_STATUS`, with unit tests for both arms.
- [ ] `atlas-maps` serves the active back-effect over the new REST endpoint, 404 when none.
- [ ] `atlas-channel` broadcasts both packets on the corresponding status events, with
      consumer tests asserting the writer is invoked for sessions in the field.
- [ ] A character entering a field with an active back-effect is sent `SET_BACK_EFFECT`;
      a lookup failure does not block map entry (test asserts fail-open).
- [ ] `SetBackEffect` / `ClearBackEffect` saga actions decode their payloads and produce
      the map commands, with handler tests including the invalid-payload rejection case
      (the `TestHandlePlayJukebox_InvalidPayload` pattern).
- [ ] The GM set and clear chat commands are gated on `c.Gm()` and produce the commands,
      with tests covering the non-GM rejection.

Gates:
- [ ] Flagless `tools/verify.sh` exits 0.
- [ ] Code review run before the PR, including a cross-service trace of the new event into
      its `atlas-channel` consumer with a test asserting the new contract.

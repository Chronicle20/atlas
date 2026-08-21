# Auto-Aggro / First-Attack Mobs — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-21
---

## 1. Overview

MapleStory mobs come in two flavours: passive mobs that ignore players until struck,
and aggressive mobs that acquire a target on proximity and attack unprovoked. The
distinction lives in WZ data as `Mob/<id>.img/info/firstAttack` — a non-zero value
marks the template aggressive. Atlas currently ingests that flag
(`services/atlas-data/atlas.com/data/monster/reader.go:81`, surfaced as
`first_attack` on three DTOs) but never acts on it: a repo-wide grep for
`FirstAttack` finds only the reader, the reader test, and the DTO fields. Atlas
aggro is therefore purely damage-driven — a monster's `ControllerHasAggro` flag
flips only after a character damages it. Every mob in the game behaves passively.

The client already knows how to do this. `CMob::TryFirstAttack` runs inside the
controlling client's mob update loop; when an aggressive, grounded, non-suspended
mob finds an eligible nearby character, the client sends the `AUTO_AGGRO`
serverbound packet (`fname CMob::ApplyControl`) naming the mob and asking the
server to make the sender the aggro'd controller. The server's job is to validate
that claim and re-issue the mob's control packet with the aggro byte set, at which
point the client's hostile AI takes over. Atlas has never handled that packet:
`AUTO_AGGRO` is present in all ten version registries with `provenance: csv-import`
and no codec, its `docs/packets/audits/STATUS.md` row is `❌` on every column that
carries an opcode, and it is routed in zero seed templates.

The supporting plumbing on the Atlas side is already built. `atlas-monsters` owns
`ControllerHasAggro` on the monster model, emits `AGGRO_CHANGED` on
`EVENT_TOPIC_MONSTER_STATUS`, and already accepts `CLEAR_AGGRO` and `FORCE_CONTROL`
commands on `COMMAND_TOPIC_MONSTER` (added for Monster Magnet). `atlas-channel`
mirrors the flag in its live monster mirror and already threads it into
`StartControlMonsterBody(m, aggro)`. `processor.go` even carries a guard written in
anticipation of aggro-at-spawn. What is missing is the inbound trigger: a codec, a
handler, a validated command, and the aggro flip. This task closes that loop.

## 2. Goals

Primary goals:

- Decode the `AUTO_AGGRO` serverbound packet in `libs/atlas-packet/monster/serverbound`
  with an IDA-derived byte layout, version-gated where the layout diverges.
- Route `AUTO_AGGRO` in every seed template that has a registry column for it, and
  promote every applicable `op × version` cell of the packet coverage matrix to
  `verified` (or record a grounded `n-a` where the client genuinely does not send it).
- Handle the packet in `atlas-channel`: resolve the mob, and emit a monster command
  that asks `atlas-monsters` to set aggro.
- Add a `SET_AGGRO` command to `COMMAND_TOPIC_MONSTER` and implement it in
  `atlas-monsters` with server-authoritative validation.
- Expose `FirstAttack` through the `atlas-monsters` monster-information model so
  the aggressive-template check is enforceable server-side.
- Re-issue the mob's control packet with `aggro = 1` so the controlling client's
  hostile AI engages, and keep the flag consistent across controller handover.

Non-goals:

- Server-side mob AI, pathing, or a proximity sweep in `atlas-monsters`. Auto-aggro
  is client-driven in this task: the controlling client detects proximity and the
  server validates the claim. A server-authoritative proximity tick is a separate,
  larger change.
- Mob attack initiation itself. Once aggro is set and the control packet re-issued,
  the existing mob-attack and mob-skill paths already carry the attack.
- Monster movement, `MOVE_MONSTER` semantics, or foothold/physics changes.
- Re-ingesting `firstAttack` from WZ — it is already parsed correctly.
- Changing the aggro decay model (`monster/aggro.go`) or the damage-driven aggro path.
- The `gms_12_1` seed template, which belongs to the in-flight v12 bring-up
  (task-175) and has no packet registry column yet.

## 3. User Stories

- As a player walking past a Ribbon Pig in Henesys, I want the passive mob to keep
  ignoring me, so that the world still reads as it should.
- As a player walking past a Jr. Necki, I want the aggressive mob to turn, chase,
  and attack me without my hitting it first, so that aggressive zones feel dangerous.
- As a player who aggros an aggressive mob and then walks away, I want the mob to
  eventually lose interest, so that aggro does not follow me across the map forever.
- As a player in Dark Sight, I want aggressive mobs not to notice me, so that the
  skill does what it says.
- As a server operator, I want a client that spams `AUTO_AGGRO` for mobs it does not
  control, for mobs in another map, or for passive templates, to be rejected without
  affecting other players.
- As an Atlas developer, I want the `AUTO_AGGRO` row in `STATUS.md` to be verified
  across every version column, so the coverage matrix reflects reality.

## 4. Functional Requirements

### 4.1 Packet decode (`libs/atlas-packet`)

- **FR-1.1** A new immutable codec `AutoAggro` lives in
  `libs/atlas-packet/monster/serverbound/auto_aggro.go`, carrying both `Encode` and
  `Decode`, an `Operation()` returning `AutoAggroHandle`, a `String()` for logging,
  and a `packet-audit:fname CMob::ApplyControl` marker.
- **FR-1.2** The byte layout MUST be derived from the client binary
  (`CMob::ApplyControl` in the GMS v95.1 IDB, cross-checked against v83/v84/v87/v92
  and jms_v185), not from remembered MapleStory knowledge and not from the CSV
  import. The derivation and the per-version addresses are recorded in the design doc.
- **FR-1.3** Where the field carrying the mob identity is a secured/CRC-fused value
  (as in the sibling `MobDropPickupRequest`, whose `mobCrc` is
  `_ZtlSecureFuse(m_dwMobID, m_dwMobID_CS)`), the codec MUST expose it under a name
  that says so, and the channel MUST resolve it to a monster unique id through the
  same path the other mob-serverbound handlers use.
- **FR-1.4** Version divergence, if any, is gated with the `MajorAtLeast` idiom.
  Raw `> N` comparisons are not acceptable.
- **FR-1.5** Byte-fixture tests exist for every version the op is routed on, each
  carrying a `packet-audit:verify` marker.

### 4.2 Packet routing and coverage

- **FR-2.1** `AUTO_AGGRO` is routed in the seed templates for every version whose
  registry entry carries an opcode. At the time of writing that is `gms_v83`
  (`0x0BD`), `gms_v84` (`0x0BD`), `gms_v87` (`0x0C9`), `gms_v92` (`0x0DD`),
  `gms_v95` (`0x0E4`), and `jms_v185` (`0x0C3`).
- **FR-2.2** For `gms_v48`, `gms_v61`, `gms_v72`, and `gms_v79` the registry
  currently records `n-a` with no opcode. This task MUST confirm that against the
  client binaries rather than inherit it: if any of those clients does emit an
  auto-aggro packet, it is routed and verified like the rest; if genuinely absent,
  the `n-a` is recorded with grounded evidence in `feature-na-evidence.yaml`.
- **FR-2.3** After implementation, `packet-audit` regeneration promotes each routed
  cell in `docs/packets/audits/status.json` / `STATUS.md`, and the corresponding
  `docs/packets/evidence/` record is pinned. A cell that does not promote is a
  failure, not a prose claim.
- **FR-2.4** No wire change is made to an already-verified packet on any version.

### 4.3 Channel handler (`atlas-channel`)

- **FR-3.1** `AutoAggroHandleFunc` in
  `services/atlas-channel/atlas.com/channel/socket/handler/auto_aggro.go` decodes the
  packet, logs it at debug like its siblings, and emits a `SET_AGGRO` command on
  `COMMAND_TOPIC_MONSTER`.
- **FR-3.2** The handler is registered in `main.go`'s `handlerMap` under
  `monstersb.AutoAggroHandle`.
- **FR-3.3** The handler performs only the cheap local checks it can make from
  session state and the live monster mirror — the session has a character and a
  field, and the named mob exists in that field's mirror. Authoritative validation
  belongs to `atlas-monsters` (FR-4.2); the channel MUST NOT be the only gate.
- **FR-3.4** A rejected or unresolvable request is dropped with a debug log and no
  response packet. `AUTO_AGGRO` has no client-visible failure path.
- **FR-3.5** The handler MUST NOT be left as decode-and-log; a stubbed handler does
  not satisfy this task.

### 4.4 Monster command and aggro flip (`atlas-monsters`)

- **FR-4.1** A `SET_AGGRO` command type is added to `COMMAND_TOPIC_MONSTER` in both
  `services/atlas-channel/.../kafka/message/monster/kafka.go` and
  `services/atlas-monsters/.../kafka/consumer/monster/kafka.go`, with a body carrying
  the requesting character id. It sits alongside the existing `CLEAR_AGGRO` and
  `FORCE_CONTROL` and does **not** move the controller.
- **FR-4.2** The `SET_AGGRO` consumer validates, in order, and drops with a debug log
  on any failure:
  1. the monster exists in the registry for the tenant;
  2. the monster is alive;
  3. the requesting character is in the monster's field;
  4. the requesting character is the monster's **current controller**;
  5. the monster's template has `firstAttack = true`.
- **FR-4.3** If the monster already has `ControllerHasAggro = true` under the same
  controller, `SET_AGGRO` is a no-op — no state write and no event emit. The command
  is idempotent; the client may send it repeatedly.
- **FR-4.4** On success the processor sets `ControllerHasAggro = true` on the monster
  and emits `AGGRO_CHANGED` on `EVENT_TOPIC_MONSTER_STATUS` with the existing
  `statusEventAggroChangedBody` shape (`controllerCharacterId`, `controllerHasAggro`).
- **FR-4.5** `SET_AGGRO` MUST NOT create or modify a damage entry. Auto-aggro confers
  no drop ownership and no kill credit; the existing `DamageEntries` model is
  untouched.

### 4.5 Template information plumbing

- **FR-5.1** `services/atlas-monsters/atlas.com/monsters/monster/information/model.go`
  gains a `firstAttack bool` field with a `FirstAttack() bool` accessor, populated
  from the existing `first_attack` JSON field in `information/rest.go:35` via the
  builder in `information/builder.go`.
- **FR-5.2** The lookup used by FR-4.2 step 5 uses the existing
  `information.NewProcessor(...).GetById(...)` cache path. A cache miss or upstream
  error is treated as **not aggressive** — auto-aggro is denied rather than granted
  on uncertainty.

### 4.6 Control re-issue and aggro lifecycle

- **FR-6.1** On receipt of `AGGRO_CHANGED` with `controllerHasAggro = true`, the
  channel updates its live monster mirror and re-issues the monster's control packet
  to the controller with `aggro = 1`, using the existing
  `StartControlMonsterBody(m, aggro)` writer path.
- **FR-6.2** When control transfers to a new character while a monster is aggro'd
  (controller leaves the field, or the picker re-picks), the new controller's control
  packet carries the aggro flag as it stands at handover — a mob that is aggro'd stays
  aggro'd through a controller change. `StatusEventStartControlBody` already carries
  `ControllerHasAggro`; this requirement is that it be populated truthfully rather
  than reset.
- **FR-6.3** Existing aggro decay (`monster/aggro.go`,
  `MonsterAggroDecayTask`) governs how auto-aggro releases. This task adds no new
  decay path. If auto-aggro produces no damage entry (FR-4.5) and the decay task
  operates only on damage entries, the design MUST state explicitly how an
  auto-aggro'd mob de-aggros when the player leaves, and implement it if the answer
  is "it currently never does".
- **FR-6.4** `CLEAR_AGGRO` continues to clear an auto-aggro'd mob's flag (Monster
  Magnet, and any future consumer), and the existing Monster Magnet ordering
  invariant — `CLEAR_AGGRO` strictly before `FORCE_CONTROL` — is preserved.

### 4.7 Behavioural outcome

- **FR-7.1** A player entering the proximity of a mob whose template has
  `firstAttack = 1` sees the mob turn and attack without being hit first.
- **FR-7.2** A player entering the proximity of a mob whose template has
  `firstAttack = 0` sees no change from today's behaviour.
- **FR-7.3** A character in Dark Sight does not trigger auto-aggro. The client-side
  gate in `CMob::TryFirstAttack` is the primary mechanism (see task-231's
  `client-analysis-dark-sight-touch-damage.md`); the design MUST state whether an
  additional server-side check is warranted, and add one if the client gate is not
  sufficient on its own.

## 5. API Surface

No REST surface changes. This task is packet-in / Kafka-across / packet-out.

### 5.1 Serverbound packet

| Op | Direction | fname | Versions |
|---|---|---|---|
| `AUTO_AGGRO` | serverbound | `CMob::ApplyControl` | `gms_v83` `0x0BD`, `gms_v84` `0x0BD`, `gms_v87` `0x0C9`, `gms_v92` `0x0DD`, `gms_v95` `0x0E4`, `jms_v185` `0x0C3`; `gms_v48`/`v61`/`v72`/`v79` pending FR-2.2 confirmation |

Body layout: **to be derived from IDA in the design phase** (FR-1.2). Do not
assume a shape here.

### 5.2 Kafka command — `COMMAND_TOPIC_MONSTER`

Producer: `atlas-channel`. Consumer: `atlas-monsters`.

```
type: "SET_AGGRO"
worldId, channelId, mapId, instance, monsterId   (existing command envelope)
body: { characterId: uint32 }
```

Failure is silent: an invalid `SET_AGGRO` is dropped with a debug log. There is no
command-failure event and no client-visible response.

### 5.3 Kafka event — `EVENT_TOPIC_MONSTER_STATUS`

No new event type. `AGGRO_CHANGED` (`EventMonsterStatusAggroChanged`) is reused with
its existing `statusEventAggroChangedBody`:

```
{ "controllerCharacterId": uint32, "controllerHasAggro": bool }
```

## 6. Data Model

No database entities, no migrations. All state is in-memory / Redis registry state
that already exists.

Changed in-memory shapes:

| Location | Change |
|---|---|
| `atlas-monsters/.../monster/information/model.go` | new unexported field `firstAttack bool` + `FirstAttack()` accessor |
| `atlas-monsters/.../monster/information/builder.go` | populate `firstAttack` from the existing `first_attack` REST field |
| `atlas-channel/.../kafka/message/monster/kafka.go` | new `CommandTypeSetAggro = "SET_AGGRO"` + command body struct |
| `atlas-monsters/.../kafka/consumer/monster/kafka.go` | matching `CommandTypeSetAggro` + body struct |

The monster registry's `ControllerHasAggro` field, the `AGGRO_CHANGED` event body,
and the channel's `LiveEntry.ControllerHasAggro` mirror field are all pre-existing
and unchanged in shape.

All command and event envelopes are tenant-scoped through the existing
`tenant.WithContext` / header path; no new tenancy surface is introduced.

## 7. Service Impact

**`libs/atlas-packet`** — new `monster/serverbound/auto_aggro.go` codec with
`Encode`/`Decode`, version gates, and per-version byte-fixture tests.

**`atlas-channel`** — new `socket/handler/auto_aggro.go`; registration in
`main.go`'s `handlerMap`; new `SET_AGGRO` command type and producer in
`kafka/message/monster`; `AGGRO_CHANGED` consumer extended (or confirmed sufficient)
to re-issue the control packet with `aggro = 1` per FR-6.1.

**`atlas-monsters`** — new `SET_AGGRO` command type and consumer arm; new processor
method implementing FR-4.2 through FR-4.5; `information.Model` gains `FirstAttack`.

**`atlas-data`** — no change expected. `first_attack` is already read from WZ and
already present on the outbound DTO (`monster/rest.go:40`).

**`atlas-monster-death`** — no change. Its DTO already carries `FirstAttack`.

**Configuration seed templates** (`services/atlas-configurations/seed-data/templates/`)
— `AUTO_AGGRO` handler routing added to the six (or more, per FR-2.2) applicable
templates.

**Packet coverage artifacts** (`docs/packets/`) — registry `provenance` updated off
`csv-import`, `status.json`/`STATUS.md` regenerated, evidence records pinned,
`feature-na-evidence.yaml` updated for any confirmed `n-a` column.

## 8. Non-Functional Requirements

**Performance.** `AUTO_AGGRO` is a per-mob, per-proximity-event packet and can
arrive frequently in a dense aggressive-mob map. The idempotent no-op path (FR-4.3)
MUST short-circuit before any Kafka emit or template lookup, so a repeat claim on an
already-aggro'd mob costs a registry read and nothing else. The `firstAttack` lookup
MUST go through the existing information cache, never a fresh REST call per packet.

**Security / anti-cheat.** The channel is not the authority. Every gate in FR-4.2
runs in `atlas-monsters`, which owns the monster registry. A client cannot aggro a
mob it does not control, a mob outside its field, a dead mob, or a passive mob.
Because auto-aggro writes no damage entry (FR-4.5), it cannot be used to claim drop
ownership or kill credit. An invalid request affects only the sender.

**Multi-tenancy.** All registry reads, information lookups, command consumption, and
event emission are tenant-scoped through the existing context path. The information
lookup in particular must pass a tenant-bearing context — the upstream `atlas-data`
middleware rejects requests without a `TENANT_ID` header (the same constraint that
shaped `bossLookupFn` in `aggro_task.go`).

**Observability.** Each validation rejection in FR-4.2 logs at debug with the
monster unique id, the requesting character id, and the specific gate that failed —
enough to distinguish "cheating client" from "we broke the controller bookkeeping"
from pod logs alone. Successful aggro flips log at debug.

**Version safety.** No behavioural or wire change to any currently-verified packet
cell on any version. Version divergence uses `MajorAtLeast`.

## 9. Open Questions

1. **Byte layout.** `CMob::ApplyControl`'s outbound body is not yet derived. Is it a
   single secured mob id (mirroring `MobDropPickupRequest.mobCrc`), or does it carry
   additional fields? Resolve in the design phase against the IDBs; do not assume.
2. **`n-a` on pre-v83 columns.** Registries record `AUTO_AGGRO` as `n-a` with no
   opcode for `gms_v48`/`v61`/`v72`/`v79`. Is that a genuine client-side absence or an
   artifact of the CSV import? FR-2.2 requires confirming it against the binaries.
3. **De-aggro for auto-aggro'd mobs.** The decay task operates on damage entries, and
   auto-aggro deliberately creates none (FR-4.5). If nothing else clears the flag, an
   auto-aggro'd mob stays aggro'd forever. FR-6.3 requires the design to answer this
   explicitly — the likely answer is a controller-proximity or idle-timer release,
   but the mechanism is not yet chosen.
4. **Dark Sight.** Is the client-side `CMob::TryFirstAttack` gate sufficient, or does
   a hidden character's client still send `AUTO_AGGRO`? Task-231's client analysis is
   the starting point; FR-7.3 requires a decision.
5. **`gms_v92` and `gms_v12`.** `gms_v92` has a registry entry (`0x0DD`) and a seed
   template but did not appear in the original backlog note's "five versions". It is
   in scope. `gms_v12` has a seed template but no registry column and belongs to
   task-175; it is out of scope. Confirm this split before implementation.
6. **Aggro on spawn.** `processor.go:192` guards a spawn-time re-pick on
   `ControllerHasAggro`, currently always false. Should an aggressive mob spawn
   already-aggro'd if a character is standing in range, or is the first client
   `AUTO_AGGRO` sufficient? Recommendation: leave spawn passive; the client will send
   `AUTO_AGGRO` on its next update tick.

## 10. Acceptance Criteria

**Packet layer**

- [ ] `libs/atlas-packet/monster/serverbound/auto_aggro.go` exists with `Encode`,
      `Decode`, `Operation()`, `String()`, and a `packet-audit:fname CMob::ApplyControl`
      marker.
- [ ] The byte layout in the codec's doc comment cites per-version IDA addresses for
      `CMob::ApplyControl`.
- [ ] Byte-fixture tests exist for every routed version, each carrying a
      `packet-audit:verify` marker.
- [ ] Any version divergence uses `MajorAtLeast`, not a raw numeric comparison.

**Coverage matrix**

- [ ] `AUTO_AGGRO` is routed in every seed template with an opcode for it.
- [ ] `docs/packets/audits/STATUS.md` shows the `AUTO_AGGRO` row as `verified` on
      every routed column — no `❌`, no `⬜` left unexplained.
- [ ] Any column left `n-a` has a grounded entry in `feature-na-evidence.yaml`
      citing the binary, not the CSV import.
- [ ] The registry `provenance` for `AUTO_AGGRO` is no longer `csv-import` on routed
      versions.
- [ ] `packet-audit` matrix/fname-doc/operations `--check` all exit 0.

**Channel**

- [ ] `AutoAggroHandleFunc` is registered in `main.go`'s `handlerMap`.
- [ ] The handler emits `SET_AGGRO`; it is not decode-and-log.
- [ ] A unit test asserts the handler emits `SET_AGGRO` with the sender's character
      id for a valid packet, and emits nothing when the mob is absent from the mirror.

**Monsters**

- [ ] `SET_AGGRO` is defined on both sides of `COMMAND_TOPIC_MONSTER` and consumed.
- [ ] A table-driven test covers each FR-4.2 rejection gate independently: unknown
      monster, dead monster, character not in field, character is not the controller,
      and passive template.
- [ ] A test asserts a repeat `SET_AGGRO` under the same controller emits no second
      `AGGRO_CHANGED` (FR-4.3).
- [ ] A test asserts `SET_AGGRO` leaves `DamageEntries` untouched (FR-4.5).
- [ ] `information.Model.FirstAttack()` returns the value of the DTO's `first_attack`
      field, covered by a test.
- [ ] An information-lookup failure denies aggro rather than granting it.

**Lifecycle**

- [ ] A test asserts the control packet re-issued after `AGGRO_CHANGED` carries
      `aggro = 1`.
- [ ] A test asserts an aggro'd mob's flag survives a controller handover (FR-6.2).
- [ ] The de-aggro mechanism decided in Open Question 3 is implemented and tested.
- [ ] The Monster Magnet `CLEAR_AGGRO`-before-`FORCE_CONTROL` ordering test still
      passes.

**Gate**

- [ ] Flagless `tools/verify.sh` exits 0.
- [ ] `backend-guidelines-reviewer` and `plan-adherence-reviewer` run clean before
      the PR.
- [ ] `packet-completeness-critic` reports no CHANGED-BUT-UNCLAIMED or
      CLAIMED-BUT-UNVERIFIED entries against this task's `coverage-manifest.yaml`.

**Manual verification**

- [ ] On a live channel, walking a character near an aggressive template (e.g. Jr.
      Necki) causes the mob to turn and attack without being struck.
- [ ] Walking near a passive template produces no change from current behaviour.

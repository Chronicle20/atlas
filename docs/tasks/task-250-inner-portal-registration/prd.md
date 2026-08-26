# Inner-Portal Registration (`USE_INNER_PORTAL`) — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-21
---

## 1. Overview

MapleStory maps contain "inner" portals — portals whose target map is the map the
player is already standing in. Walking into one teleports the character to another
coordinate pair inside the same field. Unlike a map change, the client performs this
move **locally**: it repositions the avatar itself and notifies the server with a
single serverbound packet, `USE_INNER_PORTAL` (client function
`CUserLocal::TryRegisterTeleport`).

Atlas does not decode that packet today. `docs/packets/audits/STATUS.md:640` shows
`USE_INNER_PORTAL | CUserLocal::TryRegisterTeleport` with **no codec** and an ❌ cell on
every version that assigns it an opcode (v83 `0x065`, v84 `0x065`, v87 `0x068`,
v92 `0x070`, v95 `0x071`, JMS185 `0x060`); the four older columns (v48, v61, v72, v79)
are ⬜. A repository-wide search for `InnerPortal` in `services/` and `libs/` returns
zero Go hits. The consequence is that between the teleport and the character's next
`MOVE` packet, the server's tracked position is stale: it still believes the character
is standing at the portal's entry coordinates. Every consumer of position during that
window — proximity, aggro, drop pickup range, and anything that later wants a
distance-based sanity check — reads a wrong value.

This task adds the missing codec across all applicable client versions, decodes the
packet in `atlas-channel`, registers the resulting position through the **same** path
that ordinary movement uses, and validates the reported teleport against the portal's
known coordinates before accepting it. The validation is what turns a cosmetic
desync-closing change into a real server-authority improvement: the reference
implementation in Cosmic (`InnerPortalHandler`) uses exactly this packet as its
anti-cheat distance check, and without it a crafted client can claim an arbitrary
in-map position with no server-side objection.

## 2. Goals

Primary goals:

- Decode `USE_INNER_PORTAL` in `libs/atlas-packet` with a field order derived from the
  client binary, for every version whose opcode registry assigns the op.
- Register the packet as a handled serverbound op in `atlas-channel` and route its
  per-version opcode in every seed template that carries the op.
- Update the server's tracked character position on receipt, via the same processor
  that `character_move.go` feeds, so exactly one authority owns character position.
- Validate the claimed teleport against the source portal's known coordinates and the
  character's last known position; reject (do not apply) and log an implausible claim.
- Promote the `USE_INNER_PORTAL` row in `docs/packets/audits/STATUS.md` from ❌ to ✅
  on every version that has an opcode, with byte-fixture evidence pinned per version.
- Re-derive, rather than assume, the ⬜ status of v48/v61/v72/v79 and JMS185 and record
  the finding.

Non-goals:

- Any change to cross-map warping (`CHANGE_MAP`), portal scripts
  (`CHANGE_MAP_SPECIAL` / `portal/serverbound/Script`), mystic doors, or teleport rocks.
- A general anti-cheat framework, speed-hack detection, ban/kick escalation, or a
  violation-tracking store. The distance check in this task logs and refuses to apply;
  it does not punish.
- New portal authoring, WZ extraction changes, or a new portal type in `atlas-data`
  beyond reading fields that already exist on the portal model.
- Client-side behaviour changes — the client already performs the teleport, and the
  server must not attempt to re-teleport the player in the accepted case.

## 3. User Stories

- As a **player**, I want the server to know where I am immediately after stepping
  through an in-map portal, so that monsters, drops, and other players react to my real
  position instead of the spot I left.
- As a **player near an inner portal**, I want to be able to use it repeatedly without
  being penalised, because a legitimate use is always accepted.
- As a **server operator**, I want an implausible inner-portal claim to be visible in
  logs with the character id, the claimed destination, and the reason it was refused, so
  that I can identify a crafted client.
- As an **Atlas developer**, I want `USE_INNER_PORTAL` to be a verified row in the
  coverage matrix with per-version byte fixtures, so nobody has to re-derive its layout.
- As an **Atlas developer**, I want inner-portal position updates to flow through the
  existing movement processor rather than a parallel path, so downstream consumers do
  not need to learn a second source of truth.

## 4. Functional Requirements

### 4.1 Packet derivation

- **FR-1.1** The field order for `USE_INNER_PORTAL` MUST be derived by decompiling
  `CUserLocal::TryRegisterTeleport` in the IDB for each applicable version, per
  `docs/packets/IMPLEMENTING_A_PACKET.md` Step 1. No field may be transcribed from
  memory or from another server's implementation.
- **FR-1.2** The derived ordered field list, widths, export address, and every
  per-version delta MUST be recorded in
  `docs/tasks/task-250-inner-portal-registration/structures/<version>.md#USE_INNER_PORTAL`.
- **FR-1.3** Before writing a new codec, Step 0 of the playbook MUST be executed: confirm
  no existing decoder (notably `portal/serverbound/Script`) already handles this opcode.
  If the op turns out to be a wrapper over an existing decoder, the task becomes
  verification plus a thin per-op wrapper, and the PRD's codec requirements are read
  accordingly.

### 4.2 Codec

- **FR-2.1** A new immutable struct MUST be added under
  `libs/atlas-packet/portal/serverbound/` (working name `InnerPortal`) following the
  conventions of the sibling `Script` codec: unexported fields, value-receiver accessors,
  an `Operation() string` returning a package-level handle constant, a `String()`, and
  BOTH `Encode` and `Decode`.
- **FR-2.2** The codec MUST carry a `// packet-audit:fname CUserLocal::TryRegisterTeleport`
  marker.
- **FR-2.3** Any field that differs between versions MUST be gated with the
  `MajorAtLeast` idiom used elsewhere in `libs/atlas-packet`; a raw `> N` version
  comparison is not acceptable.
- **FR-2.4** `Decode` MUST consume exactly the bytes the client writes for every
  supported version — no trailing-byte tolerance, no short-read tolerance.
- **FR-2.5** Encoding then decoding a value MUST round-trip to an equal value
  (round-trip test), and decoding a captured/derived golden byte string MUST produce the
  expected field values (golden-byte test) for each version.

### 4.3 Channel handling

- **FR-3.1** `atlas-channel` MUST register a handler for the new operation handle in
  `main.go`'s `handlerMap`, following the `portal2.PortalScriptHandle` precedent.
- **FR-3.2** The handler MUST decode the packet, log it at debug in the existing
  `[%s] read [%s]` form, and then invoke the validation and position-update flow below.
- **FR-3.3** The handler MUST NOT block on remote calls in a way that stalls the socket
  read loop beyond what the existing movement handler already does.
- **FR-3.4** The handler MUST be tenant-scoped through the request context in the same
  way as the existing handlers; no tenant identity may be derived from packet contents.

### 4.4 Validation (distance / plausibility check)

- **FR-4.1** The handler MUST resolve the portal identified by the packet within the
  character's current field, using the existing portal data path
  (`atlas-channel/data/portal`).
- **FR-4.2** If the portal cannot be resolved in the character's current field, the
  teleport MUST be refused: the position is not updated, and a warning is logged with
  the character id, field, and the unresolved identifier.
- **FR-4.3** If the resolved portal's target map is **not** the character's current map,
  the packet MUST be refused and logged — an inner portal by definition targets its own
  field, and a mismatch means the client is claiming the wrong portal.
- **FR-4.4** The character's last known position MUST be compared against the resolved
  portal's coordinates. If the distance exceeds a configured threshold, the teleport is
  refused and logged at warning with: character id, field, portal, last known position,
  portal position, computed distance, and threshold.
- **FR-4.5** The destination coordinates the server adopts MUST come from the resolved
  portal's target within the field — the **server's** data — not from coordinates
  supplied in the packet. Packet-supplied coordinates may be logged and used for the
  plausibility comparison but MUST NOT be trusted as the new position.
- **FR-4.6** A refused teleport MUST NOT disconnect, kick, ban, or forcibly reposition
  the character. It is a no-op plus a log line. (The next `MOVE` packet re-establishes
  position exactly as it does today, so a false positive degrades to current behaviour.)
- **FR-4.7** The distance threshold MUST be a named constant or configuration value with
  its unit documented, not an inline magic number.

### 4.5 Position registration

- **FR-5.1** On an accepted teleport, the character's tracked position MUST be updated
  through the same processor that `character_move.go` uses
  (`movement.NewProcessor(...).ForCharacter(...)` or the position-update entry point it
  fronts), so that inner-portal moves and walking moves converge on one authority.
- **FR-5.2** Downstream consumers of position (map/character position state and any
  Kafka position event that ordinary movement already emits) MUST observe the
  inner-portal move through their existing subscriptions — no new event type is
  introduced unless the movement path genuinely cannot express a teleport, in which case
  the design doc must state why.
- **FR-5.3** The update MUST NOT emit a clientbound movement broadcast that would cause
  the teleporting client to be re-positioned by its own message; other clients in the
  field observe the new position through whatever mechanism ordinary movement already
  uses.

### 4.6 Template routing and version coverage

- **FR-6.1** The per-version opcode for `USE_INNER_PORTAL` MUST be routed to the new
  handle in every seed template under
  `services/atlas-configurations/seed-data/templates/` whose client version assigns the
  op: `template_gms_83_1.json` (`0x065`), `template_gms_84_1.json` (`0x065`),
  `template_gms_87_1.json` (`0x068`), `template_gms_92_1.json` (`0x070`),
  `template_gms_95_1.json` (`0x071`), `template_jms_185_1.json` (`0x060`).
- **FR-6.2** For the columns currently marked ⬜ (v48, v61, v72, v79) and for
  `template_gms_12_1.json`, the ⬜ MUST be **re-derived**, not assumed: confirm from the
  version's opcode registry / IDB that the op genuinely does not exist in that client. If
  an opcode does exist, that version joins FR-6.1's list. The finding for each version
  MUST be recorded in the task folder.
- **FR-6.3** No wire behaviour of an already-verified op or version may change as a side
  effect of this task.

### 4.7 Coverage matrix promotion

- **FR-7.1** Each implemented version cell MUST be verified per
  `docs/packets/audits/VERIFYING_A_PACKET.md`: byte-fixture test with a
  `// packet-audit:verify` marker, evidence record pinned, matrix regenerated.
- **FR-7.2** `docs/packets/audits/STATUS.md` MUST be regenerated by `packet-audit matrix`
  (never hand-edited) and the `USE_INNER_PORTAL` row MUST show ✅ for every version in
  scope.
- **FR-7.3** `packet-audit` `matrix` / `fname-doc` / `operations --check` MUST exit 0.
- **FR-7.4** A `coverage-manifest.yaml` declaring the op × version cells this task claims
  MUST exist in the task folder for the completeness critic to diff against.

## 5. API Surface

No REST endpoint is added or modified.

**Wire surface (new, serverbound):** `USE_INNER_PORTAL`, opcodes per FR-6.1. The exact
field list is unknown until FR-1.1 completes and MUST NOT be guessed here. For reference,
the sibling in-map portal packet `portal/serverbound/Script`
(`CUserLocal::CheckPortal_Collision`) carries `fieldKey byte`, `portalName string`,
`x int16`, `y int16`; the derivation may or may not find the same shape and the
implementation follows the derivation, not this note.

**Error cases:** the packet has no response. Every failure mode (unresolvable portal,
target-map mismatch, distance exceeded) is a silent no-op to the client plus a
server-side log line, per FR-4.6.

**Internal call surface:** the handler consumes the existing portal data processor
(`atlas-channel/data/portal`) and the existing movement/position processor. Whether any
Kafka message is emitted is determined by the movement path already in use (FR-5.2).

## 6. Data Model

No new persisted entity, no schema migration, and no new `tenant_id`-scoped table.

The task reads the existing portal model
(`services/atlas-channel/atlas.com/channel/data/portal/model.go`: `id`, `name`, `target`,
`portalType`, `x`, `y`, `targetMapId`, `scriptName`) — specifically `x`/`y` for the
plausibility comparison and `target`/`targetMapId` for the same-field assertion and the
adopted destination.

If a distance threshold is made configurable rather than constant, it belongs with the
existing channel/tenant configuration mechanism and must be tenant-scoped like every
other configuration value; the design phase decides constant vs. configured (Open
Question 9.3).

## 7. Service Impact

| Service / lib | Change |
|---|---|
| `libs/atlas-packet` | New `portal/serverbound` codec + handle constant, round-trip and golden-byte tests, `packet-audit` markers. |
| `libs/atlas-opcodes` | Registry entry for the op if the registry is the source of the handle→opcode mapping (confirm during design). |
| `services/atlas-channel` | New `socket/handler/*.go` handler; `handlerMap` registration in `main.go`; validation logic; call into the existing movement/position processor. |
| `services/atlas-configurations` | Opcode routing added to each in-scope seed template under `seed-data/templates/`. |
| `docs/packets/audits` | Regenerated `STATUS.md` and `status.json`, new evidence records. |
| `services/atlas-data` / `atlas-maps` | Read-only consumers; no change expected. If portal lookup by the packet's identifier is not already possible, the design doc must say so explicitly rather than silently adding a field. |

## 8. Non-Functional Requirements

- **Performance:** the handler runs on the socket read path. Portal resolution MUST use
  the same cached data access the existing portal handlers use; it MUST NOT introduce an
  uncached per-packet remote fetch. Inner portals can be triggered repeatedly in quick
  succession, so the path must tolerate a burst without a per-call round trip.
- **Security / server authority:** the server adopts its own portal coordinates
  (FR-4.5); packet-supplied position is untrusted input. All packet fields must be
  treated as attacker-controlled — no unchecked indexing, no unbounded string/array read.
- **Multi-tenancy:** tenant comes from context, never from the packet. Portal data lookups
  and any registry state are tenant-scoped.
- **Observability:** accepted teleports log at debug; every refusal logs at warning with
  enough context to identify the character and the reason. Log lines must not be so
  chatty that a normal player using a portal repeatedly floods the log — accepted moves
  stay at debug.
- **Compatibility:** a version with no opcode for the op is unaffected; a version in
  scope gains only a new inbound handler. No clientbound packet changes.
- **Testing:** module-local `go build ./... && go test ./...` for each touched module
  during implementation; the flagless `tools/verify.sh` must exit 0 before the branch is
  claimed done.

## 9. Open Questions

- **9.1** What does `CUserLocal::TryRegisterTeleport` actually write? Portal id vs.
  portal name vs. raw coordinates is unknown until FR-1.1 is done, and the answer decides
  how FR-4.1's portal resolution is keyed. *(Resolved in the design phase by
  decompilation.)*
- **9.2** Does the existing movement processor accept a discrete "teleport to (x, y)"
  update, or does it only accept a decoded movement-path blob? If the latter, the design
  must choose between synthesising a minimal path and adding a position-set entry point —
  and justify it (FR-5.2).
- **9.3** Is the distance threshold a package constant or a tenant configuration value?
  Default assumption: a documented package constant, promoted to configuration only if
  the design finds real per-tenant variance.
- **9.4** Do any of v48/v61/v72/v79 (and the `gms_12` template) actually assign this
  opcode? FR-6.2 requires re-deriving rather than trusting the ⬜.
- **9.5** Does an inner-portal move need to be broadcast to other players in the field,
  or does the client-side teleport already propagate through the normal movement
  broadcast that follows? *(Affects FR-5.3; answer from client behaviour, not
  assumption.)*
- **9.6** Is JMS185's layout identical to the GMS line, or does it diverge enough to need
  a version gate (FR-2.3)?

## 10. Acceptance Criteria

- [ ] `docs/tasks/task-250-inner-portal-registration/structures/<version>.md` records the
      derived field order, widths, and export address for every in-scope version.
- [ ] A new immutable codec exists in `libs/atlas-packet/portal/serverbound/` with
      `Encode`, `Decode`, an operation handle constant, and a
      `packet-audit:fname CUserLocal::TryRegisterTeleport` marker.
- [ ] Round-trip and golden-byte tests pass for every in-scope version; version deltas
      are gated with `MajorAtLeast`.
- [ ] `atlas-channel` registers the handler in `main.go`'s `handlerMap` and decodes the
      packet without error against the fixtures.
- [ ] An unresolvable portal, a target map that is not the current map, and an
      over-threshold distance each result in **no position change** and a warning log —
      each covered by a test.
- [ ] A legitimate inner-portal use updates the tracked character position through the
      same processor `character_move.go` uses, verified by a test that asserts the
      processor call (not merely that the handler returned).
- [ ] The server-adopted destination comes from portal data, not from the packet
      (asserted by a test that feeds mismatched packet coordinates).
- [ ] Every in-scope seed template under
      `services/atlas-configurations/seed-data/templates/` routes the correct per-version
      opcode to the new handle.
- [ ] The ⬜ status of v48/v61/v72/v79 (+ `gms_12`) has been re-derived and the result
      documented in the task folder.
- [ ] `docs/packets/audits/STATUS.md` (regenerated, not hand-edited) shows ✅ for
      `USE_INNER_PORTAL` on every version in scope; evidence records are pinned.
- [ ] `packet-audit matrix`, `packet-audit fname-doc`, and `packet-audit operations
      --check` all exit 0.
- [ ] `docs/tasks/task-250-inner-portal-registration/coverage-manifest.yaml` declares the
      claimed op × version cells.
- [ ] Flagless `tools/verify.sh` exits 0.
- [ ] Code review completed before the PR is opened.

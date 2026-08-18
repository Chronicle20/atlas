# Evan Dragon Entity — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-13
---

## 1. Overview

Evan is a creatable job on Atlas tenants from v83 onward, but the class's defining
visual — the dragon companion that accompanies the character from its first growth
stage — does not exist anywhere in the server. The client is fully capable of
rendering it: `CDragon` is present in all six audited client columns (v83, v84,
v87, v92, v95, JMS185), and each of those columns has assigned opcodes for the
three clientbound dragon ops plus the one serverbound movement op. Atlas simply
never sends or accepts them.

The result is a class that is functionally creatable and visually broken. An Evan
player sees no dragon; other players in the map see no dragon; dragon movement is
never relayed. Every one of the four dragon matrix cells is `❌` in every
applicable version column, and no seed template binds a `Dragon`-named handler or
writer.

This task introduces a new `atlas-dragons` service that owns dragon lifecycle and
position state, wires the four packet codecs into `libs/atlas-packet` and
`atlas-channel`, routes them through all applicable seed templates, and promotes
all 24 matrix cells (4 ops × 6 versions) to `✅` verified with byte fixtures.

## 2. Goals

Primary goals:

- Every Evan character at job 2200 or above has a dragon entity spawned on the
  field, owned and tracked by a dedicated `atlas-dragons` service.
- The dragon is visible to the owner and to every other character in the same
  field, including characters who enter the field after the dragon exists.
- Dragon movement submitted by the owning client is relayed to all other clients
  in the field.
- The dragon is removed cleanly on job change away from the Evan dragon-bearing
  range, on map change, on channel change, and on logout.
- All four dragon packet ops are implemented as version-gated codecs and verified
  to `✅` in the coverage matrix for all six applicable version columns.

Non-goals:

- Dragon combat participation, dragon HP, or dragon damage. The dragon is a
  cosmetic/companion entity; it has no independent combat state.
- Evan mount-riding behavior (`CAvatar::IsRidingEvanDragon`, v95 IDB 0x50ac70).
  This is a separate Evan capability and is out of scope.
- Evan skill effects that animate the dragon (skill-triggered dragon animation
  packets are not among the four ops in scope).
- The Evan job-advancement NPC flow and growth-stage progression logic. This task
  reacts to job state; it does not create it.
- The `DRAGON_BALL_BOX` / `DRAGON_BALL_BOX_REQUEST` / `DRAGON_BALL_SUMMON_REQUEST`
  op family. Despite the name these are an unrelated event/item feature, present
  only in the v92 and v95 columns.
- Backfilling dragon support to v48, v61, v72, or v79. Those columns are `⬜`
  (n-a) — the clients have no `CDragon` opcode assignment.

## 3. User Stories

- As an Evan player, I want my dragon to appear beside me from my first growth
  stage onward, so that my class looks and feels like Evan rather than a
  dragonless beginner.
- As an Evan player, I want my dragon to follow me as I move around the map, so
  that it reads as a companion rather than a static decoration.
- As another player in the map, I want to see an Evan's dragon, so that the world
  is consistent between clients.
- As a player entering a map that already contains an Evan, I want that Evan's
  dragon to be visible immediately on entry, not only after the Evan next moves.
- As an Evan player who changes maps or channels, I want my dragon to accompany
  me without duplicating or orphaning entities in the map I left.
- As an operator, I want dragon state to be per-tenant isolated, so that one
  tenant's dragons are never visible to or addressable by another tenant.

## 4. Functional Requirements

### FR-1 — Dragon lifecycle

- **FR-1.1** A dragon exists for exactly one character at a time, and a character
  has at most one dragon. The dragon's identity is scoped by tenant and owning
  character id.
- **FR-1.2** A dragon is created when a character whose job id is in the
  dragon-bearing Evan range enters a field — on login, on map change, and on
  channel change.
- **FR-1.3** The dragon-bearing job range is job id `2200` through `2218`
  inclusive (`job.EvanStage1` … `job.EvanStage10`, per
  `libs/atlas-constants/job/identities_gen.go:83-92`). See OQ-3 — the exact lower
  bound must be confirmed before implementation.
- **FR-1.4** A dragon is created when a character's job changes *into* the
  dragon-bearing range, and destroyed when a character's job changes *out of* it.
- **FR-1.5** A dragon is destroyed when its owning character leaves the field
  (map change, channel change, or logout). No dragon may outlive its owner's
  presence in a field.
- **FR-1.6** Dragon creation and destruction are idempotent: a duplicate create
  for a character that already has a dragon must not produce a second entity or a
  second `SPAWN_DRAGON` broadcast, and a destroy for a character with no dragon
  must be a no-op rather than an error.

### FR-2 — Dragon state

- **FR-2.1** A dragon carries: owning character id, tenant, field (world, channel,
  map, instance), x, y, and stance.
- **FR-2.2** The dragon's initial position on spawn is derived from the owning
  character's position at spawn time.
- **FR-2.3** Dragon state is runtime-only, held in an in-memory registry keyed by
  tenant and character id. It is not persisted to a database — a dragon is fully
  reconstructible from the owning character's job and position. This mirrors
  `atlas-summons`, which is likewise registry-backed with no database.
- **FR-2.4** The registry is safe for concurrent access. Note that
  `atlas-channel`'s `ForEachInMap` runs handlers in parallel; no dragon
  broadcast path may rely on serialized per-map execution.

### FR-3 — Visibility and broadcast

- **FR-3.1** On dragon creation, `SPAWN_DRAGON` is sent to every character in the
  dragon's field, including the owner.
- **FR-3.2** When a character enters a field, that character receives one
  `SPAWN_DRAGON` for every dragon already present in that field.
- **FR-3.3** On dragon destruction, `REMOVE_DRAGON` is sent to every character in
  the dragon's field.
- **FR-3.4** A character must never receive a `SPAWN_DRAGON` for a dragon in a
  different field, nor for a dragon belonging to a different tenant.

### FR-4 — Movement

- **FR-4.1** The serverbound `MOVE_DRAGON` op is decoded by `atlas-channel` from
  the owning client and validated: the submitting character must be the dragon's
  owner, and the dragon must exist in the submitter's current field.
- **FR-4.2** The resulting position and stance update the dragon's registry state.
- **FR-4.3** The clientbound `MOVE_DRAGON` op is broadcast to every character in
  the field *except* the owner, matching the relay semantics of the existing
  summon and pet movement paths.
- **FR-4.4** A `MOVE_DRAGON` from a character with no dragon, or naming a dragon
  the submitter does not own, is rejected and logged; it must not create a dragon
  as a side effect.

### FR-5 — Packet codecs

- **FR-5.1** Four codecs are implemented in `libs/atlas-packet`, each with both
  `Encode` and `Decode`, following the immutable-struct convention:

  | Op | Direction | FName |
  |---|---|---|
  | `SPAWN_DRAGON` | clientbound | `CDragon::OnCreated` |
  | `MOVE_DRAGON` | clientbound | `CDragon::OnMove` |
  | `REMOVE_DRAGON` | clientbound | `CUser::OnDragonPacket` |
  | `MOVE_DRAGON` | serverbound | `CVecCtrlDragon::EndUpdateActive` |

- **FR-5.2** Field order and types are derived from the GMS v95.1 IDB, not from
  the registry fname alone. All four registry entries carry
  `provenance: csv-import` (`docs/packets/registry/gms_v95.yaml:1059-1073,3471-3475`),
  meaning no real decompile has ever backed these layouts. Symbol names must be
  distrusted per `docs/packets/IMPLEMENTING_A_PACKET.md`.
- **FR-5.3** Any field that diverges across versions is version-gated with the
  `MajorAtLeast` idiom. Raw `> N` major-version comparisons are prohibited — see
  the `MajorVersion()>83` off-by-one that misclassified v87.
- **FR-5.4** No wire change may be made to an already-verified cell of any other
  op as a side effect of this work.

### FR-6 — Template routing

- **FR-6.1** Handler and writer entries for the four ops are added to each seed
  template under `services/atlas-configurations/seed-data/templates/` for the six
  applicable versions: `template_gms_83_1.json`, `_84_`, `_87_`, `_92_`, `_95_`,
  `template_jms_185_1.json`.
- **FR-6.2** No entries are added to `template_gms_12_1.json`, `_48_`, `_61_`,
  `_72_`, or `_79_` — those columns are `⬜` n-a.
- **FR-6.3** Every new entry is inserted at its sorted position by `opCode`;
  `tools/template-opcode-order-guard.sh` must pass.
- **FR-6.4** Every new handler entry carries a validator. A handler with a
  missing validator is silently dropped at load time.
- **FR-6.5** Every new writer entry carries an `fname`.
- **FR-6.6** `tools/template-duplicate-binding-guard.sh` must pass — no
  leading-zero-padded duplicate bindings.
- **FR-6.7** The serverbound `MOVE_DRAGON` handler is a movement handler and must
  therefore carry a non-empty `options.types` movement table byte-identical to
  the other movement handlers in the same template, per
  `tools/template-movement-types-guard.sh`. See OQ-5.
- **FR-6.8** Live tenant socket configurations must be reconciled to the updated
  templates. New opcodes present only in the template and not in the live tenant
  config are silently dropped at runtime.

### FR-7 — Coverage matrix

- **FR-7.1** All 24 cells (4 ops × 6 versions) are promoted to `✅` verified.
- **FR-7.2** Each cell is verified per `docs/packets/audits/VERIFYING_A_PACKET.md`
  — client read order derived from the IDB, a byte-fixture test carrying a
  `packet-audit:verify` marker, and a pinned evidence record.
- **FR-7.3** A round-trip fixture alone does not constitute verification. Each
  cell's read order must be derived from the client, not from the Atlas encoder.

## 5. API Surface

`atlas-dragons` exposes a JSON:API REST surface following the project convention
(api2go, resource type from `GetName()`, `RegisterHandler(l)(si)`), plus Kafka
command and event topics.

### REST

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/dragons` | List dragons, filterable by field (world/channel/map) |
| `GET` | `/dragons/{characterId}` | Fetch the dragon owned by a character |

Both are tenant-scoped via `tenant.MustFromContext(ctx)`. `GET` for a character
with no dragon returns `404` — consumers must treat this as "no dragon," not as a
fetch failure. (This mirrors a known past defect where a `404` from a buff GET was
treated as an error.)

Dragon creation and destruction are not REST-driven; they are consequences of
character lifecycle events (FR-1). No `POST`/`DELETE` endpoints are exposed.

### Kafka

- **Command topic** `COMMAND_TOPIC_DRAGON` — `CREATE`, `DESTROY`, `MOVE`.
- **Event topic** `EVENT_TOPIC_DRAGON_STATUS` — `CREATED`, `DESTROYED`, `MOVED`.

Message bodies carry explicit tenant headers. The `TenantHeaderDecorator` has
previously dropped headers silently on some paths; header propagation must be
asserted in tests, not assumed.

`atlas-dragons` consumes character status events (login, logout, map change,
channel change, job change) to drive FR-1. `atlas-channel` consumes
`EVENT_TOPIC_DRAGON_STATUS` to drive the FR-3 and FR-4 broadcasts.

Exact request/response shapes and message bodies are to be settled in `design.md`.

## 6. Data Model

No database. `atlas-dragons` holds a singleton in-memory registry
(`sync.Once` + `sync.RWMutex`, per project convention) keyed by
`(tenant, characterId)`.

```
Dragon
  ownerCharacterId  uint32
  fld               field.Model   // world.Id (byte), channel.Id (byte),
                                  // _map.Id (uint32), instance uuid
  x                 int16
  y                 int16
  stance            byte
  jobId             uint16        // the Evan growth stage at spawn time
```

The model is immutable — private fields, getters, and a Builder. There is no
`*_testhelpers.go`; tests construct dragons through the Builder.

Because there is no database, this service is **not** added to
`tools/db-bootstrap.sh` and requires no migration. It is still subject to every
other registration list in `docs/adding-a-new-service.md` (see §7).

## 7. Service Impact

### `atlas-dragons` (new)

Full registration per `docs/adding-a-new-service.md`, which enumerates every
hand-maintained list. At minimum:

- `.github/config/services.json` (the single source of truth for CI and
  `docker-bake.hcl`)
- `go.work`
- `deploy/k8s/base/`
- `deploy/k8s/overlays/main/` **and** `deploy/k8s/overlays/pr/` — both
- Ingress (REST service)
- GHCR package visibility must be flipped to public after first publish;
  otherwise the pod lands in `ImagePullBackOff` with an anonymous-pull `401`
  while CI reports green.
- The image tag must be pinned, never `:latest`.
- `tools/service-registration-guard.sh` must pass.

### `libs/atlas-packet`

Four new codecs (FR-5), each version-gated across the six applicable columns,
each with byte fixtures.

### `services/atlas-channel`

- A `dragon` model and registry mirroring the existing `summon` model shape.
- Writers for the three clientbound ops.
- A handler for the serverbound `MOVE_DRAGON`.
- Field-enter fan-out: extend the existing map-enter spawn path
  (`kafka/consumer/map/consumer.go`, which already fans out `SummonSpawn`) to
  include dragons, satisfying FR-3.2.
- A consumer for `EVENT_TOPIC_DRAGON_STATUS`.

### `services/atlas-configurations`

Handler and writer entries in six seed templates (FR-6).

### `docs/packets/`

Registry entries gain real provenance; `status.json` / `STATUS.md` regenerate
with 24 cells promoted; evidence records pinned per cell.

### Not impacted

`atlas-summons` (skill summons only), `atlas-mounts`, `atlas-pets`,
`atlas-character`. Consuming character events does not require changes to
`atlas-character`'s producers if the needed events already exist — to be
confirmed in `design.md`.

## 8. Non-Functional Requirements

- **Multi-tenancy.** Every registry key, REST call, and Kafka message is
  tenant-scoped. Cross-tenant dragon visibility is a correctness failure, not a
  cosmetic one.
- **Concurrency.** The registry must tolerate parallel access. `ForEachInMap` in
  `atlas-channel` is parallel; broadcast paths must not assume serialization.
- **Idempotency.** Kafka delivery is at-least-once. Every dragon command handler
  must be idempotent (FR-1.6) — a redelivered `CREATE` must not spawn a second
  dragon, and a redelivered `MOVE` must not desynchronize position.
- **Version correctness.** All client-facing wire values are config-resolved, not
  hard-coded (DOM-25). Opcodes come from the tenant socket configuration.
- **Constants reuse.** Use `libs/atlas-constants` types directly — `world.Id`,
  `channel.Id`, `_map.Id`, `field.Model`, `job.Identity`. Do not introduce
  parallel local types (DOM-21).
- **Observability.** Dragon create/destroy/move are logged with tenant, character
  id, and field. Loki queries must use `service_name`, not `app`.
- **Performance.** Dragon movement relay is on the hot path of every Evan's
  motion. It must add no per-move REST call to another service; the channel-side
  registry serves the broadcast directly.
- **Goroutines.** Any concurrency uses `routine.Go`; bare `go` statements fail
  `tools/goroutine-guard.sh`.
- **Redis.** If any keyed Redis access is introduced it must go through
  `libs/atlas-redis` (`tools/redis-key-guard.sh`).

## 9. Open Questions

- **OQ-1 — Packet field layouts.** All four registry entries are
  `provenance: csv-import`; no decompile backs them. The exact field order of
  `CDragon::OnCreated`, `CDragon::OnMove`, `CUser::OnDragonPacket`, and
  `CVecCtrlDragon::EndUpdateActive` must be derived from the GMS v95.1 IDB during
  design. **Unverified — must not be guessed.**
- **OQ-2 — Dragon wire identity.** Cosmic is understood to reuse the owner's
  character id as the dragon id. This is **unverified**; confirm from
  `CDragon::OnCreated` in the v95 IDB.
- **OQ-3 — Growth-stage lower bound.** FR-1.3 assumes 2200. Cosmic gates
  `createDragon` on `hasSPTable(job)` (`client/Character.java:1141`). Whether
  that includes 2200 (Evan beginner) or begins at 2210 (1st growth) is
  **unverified** and must be confirmed against Cosmic source and WZ data before
  implementation.
- **OQ-4 — v83 applicability.** The v83 client has a `SPAWN_DRAGON` opcode
  assignment (`0x0B5`), but whether Evan is genuinely playable on v83 tenants is
  unconfirmed; the original backlog note said "creatable on v84+". If v83 Evan is
  not real, the v83 column should be `⬜` n-a rather than implemented — and the
  n-a consistency gate applies. Resolve before template work.
- **OQ-5 — Is serverbound `MOVE_DRAGON` a movement-table handler?** It is named
  `CVecCtrlDragon::EndUpdateActive` and plausibly carries a movement fragment
  list like `CharacterMoveHandle` / `PetMovementHandle` / `SummonMoveHandle`. If
  so, FR-6.7 applies and the handler needs the shared `options.types` table.
  Confirm from the IDB.
- **OQ-6 — Which character events already exist?** FR-1 needs login, logout, map
  change, channel change, and job change events. Whether `atlas-character` /
  `atlas-channel` already emit all five in a consumable form is unconfirmed.
- **OQ-7 — Cross-service seam coverage.** Green unit tests in `atlas-dragons`
  prove nothing about the `atlas-channel` consumer. The design must state how the
  dragon event → channel broadcast seam is actually exercised.

## 10. Acceptance Criteria

**Packets**

- [ ] All 24 matrix cells (`SPAWN_DRAGON`, clientbound `MOVE_DRAGON`,
      `REMOVE_DRAGON`, serverbound `MOVE_DRAGON` × v83, v84, v87, v92, v95,
      JMS185) read `✅` in `docs/packets/audits/STATUS.md`, subject to OQ-4.
- [ ] Each cell has a byte-fixture test with a `packet-audit:verify` marker and a
      pinned evidence record, committed together with the codec.
- [ ] No previously-`✅` cell of any other op has regressed.
- [ ] Registry entries for the four ops no longer carry bare `csv-import`
      provenance where a decompile was performed.

**Service**

- [ ] `atlas-dragons` builds, deploys to the PR overlay, and reports ready.
- [ ] `tools/service-registration-guard.sh` passes.
- [ ] The GHCR package for `atlas-dragons` is public and the pod pulls its image.
- [ ] The image is pinned to a commit sha in both overlays, not `:latest`.

**Behavior (verified against a live tenant, not only unit tests)**

- [ ] An Evan at a dragon-bearing job sees its dragon on login.
- [ ] A second player entering the map sees the existing Evan's dragon
      immediately, before that Evan moves.
- [ ] Dragon movement by the owner is visible to the second player.
- [ ] The owner does not receive an echo of its own `MOVE_DRAGON`.
- [ ] Changing maps produces exactly one `REMOVE_DRAGON` in the old map and
      exactly one `SPAWN_DRAGON` in the new — no duplicates, no orphans.
- [ ] Logging out produces `REMOVE_DRAGON` for remaining players in the map.
- [ ] A non-Evan character in the same map has no dragon and triggers no dragon
      traffic.
- [ ] A redelivered dragon `CREATE` command produces no second dragon.

**Guards and build**

- [ ] `go test -race ./...` clean in every changed module.
- [ ] `go vet ./...` clean in every changed module.
- [ ] `go build ./...` clean in every changed service.
- [ ] `docker buildx bake atlas-dragons` succeeds from the worktree root.
- [ ] `tools/lint.sh --check` clean.
- [ ] `tools/goroutine-guard.sh`, `tools/redis-key-guard.sh` clean.
- [ ] `tools/template-opcode-order-guard.sh`,
      `tools/template-duplicate-binding-guard.sh`,
      `tools/template-movement-types-guard.sh` clean.
- [ ] Code review run (`superpowers:requesting-code-review`) before PR.

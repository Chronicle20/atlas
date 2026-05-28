# GM Command: Spawn Monster at Position (`@mob spawn`) — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-05-28
---

## 1. Overview

Game Masters need a fast way to summon a specific monster on demand while in-game,
without editing map spawn data or relying on natural spawns. This feature adds a
new GM chat command, `@mob spawn <templateId> [count]`, to the **atlas-messages**
service. When a GM issues the command, the system spawns the requested monster
template at the GM's current position on their current map/channel/world.

The command joins the existing `@mob` command family in atlas-messages
(`@mob kill all`, `@mobstatus`, `@mobclear`), reusing the established command
producer/registry pattern: a regex-matched `command.Producer`, a `c.Gm()`
authorization gate, and pink-text feedback issued back to the GM.

Spawning is dispatched as a new Kafka command on the existing
`COMMAND_TOPIC_MONSTER` topic (a new `SPAWN` field-command type), consumed by
**atlas-monsters**, which already owns the monster-creation path
(`monster.Processor.Create`). Before dispatching, atlas-messages pre-validates the
template id and resolves the foothold beneath the GM's position via existing
**atlas-data** REST endpoints, so the GM gets immediate feedback on a bad id and
the spawned monster lands correctly on terrain.

## 2. Goals

Primary goals:
- Add a GM-only chat command `@mob spawn <templateId> [count]` in atlas-messages.
- Spawn the requested monster template at the issuing GM's current X/Y on their
  current world/channel/map (and instance).
- Pre-validate the `templateId` against monster data and give the GM a clear
  error message for an unknown template before anything is spawned.
- Resolve the foothold beneath the GM's position so the spawned monster rests on
  terrain rather than falling through the floor.
- Support an optional `count` to spawn multiple instances in one command, with a
  sane upper bound.
- Follow the existing `@mob`-family command, Kafka command, and consumer patterns
  exactly — no new architectural surface.

Non-goals:
- Spawning at arbitrary coordinates, on a different map, or for a target character
  other than the issuer.
- Persistent / respawning / scripted-wave spawns.
- Spawn-by-name lookup (template is referenced by numeric id only).
- Any atlas-ui or client changes.
- Changing how natural map spawns work.

## 3. User Stories

- As a GM, I want to type `@mob spawn 100100` so that a Snail appears at my feet
  for testing combat, drops, or mechanics.
- As a GM, I want to type `@mob spawn 100100 5` so that five of the monster spawn
  at once for AoE / crowd testing.
- As a GM, I want an immediate, clear error message when I type an invalid
  template id so that I know nothing was spawned and why.
- As a GM, I want spawned monsters to land on the ground where I am standing so
  that they behave normally instead of falling through the floor.
- As a non-GM player, I want the command to do nothing (and not leak its
  existence) so that ordinary players cannot summon monsters.

## 4. Functional Requirements

### 4.1 Command syntax and parsing (atlas-messages)
- FR-1: A new command producer matches the pattern
  `^@mob spawn\s+(\d+)(?:\s+(\d+))?$` (whitespace-tolerant), capturing the
  numeric `templateId` and an optional `count`.
- FR-2: The producer is registered in atlas-messages `main.go` alongside the
  other `@mob` producers (`MobKillAllCommandProducer`, etc.).
- FR-3: If the message does not match the pattern, the producer returns
  `(nil, false)` so message handling falls through to normal chat / other
  commands. This includes the existing `@mob kill all` — ensure the new regex
  does not capture `@mob kill all` and vice-versa.

### 4.2 Authorization
- FR-4: The command executes only when `c.Gm()` is true. For a non-GM issuer the
  producer returns `(nil, false)` (mirroring the existing `@mob` commands), so the
  command silently does not match and is not advertised.

### 4.3 Position resolution
- FR-5: The spawn position is the issuer's current position: `c.X()`, `c.Y()`,
  on `c.MapId()`, `c.WorldId()`, the field's channel, and the field's instance.
  All of these are already available on the atlas-messages `character.Model` and
  the `field.Model` passed to the producer. No additional position query is
  required.

### 4.4 Template validation (pre-dispatch)
- FR-6: Before dispatching, atlas-messages validates the `templateId` exists by
  calling atlas-data `GET /data/monsters/{monsterId}`.
- FR-7: If the template does not exist (404 / error), the command issues a
  pink-text error to the GM (e.g. `Unknown monster template: <id>`) and does NOT
  dispatch any spawn.
- FR-8: A successful validation lookup MAY be used to enrich the success message
  (e.g. include the monster name), but this is optional.

### 4.5 Foothold resolution
- FR-9: atlas-messages resolves the foothold beneath the GM's position via
  atlas-data `POST /data/maps/{mapId}/footholds/below` with body `{x, y}`,
  obtaining a foothold id (`Fh`).
- FR-10: If foothold resolution fails or returns no foothold, the system falls
  back to `Fh = 0` (the existing safe default) and still spawns; it does not abort
  the command on a foothold-lookup failure.

### 4.6 Spawn dispatch (Kafka)
- FR-11: atlas-messages emits a new `SPAWN` field-command on
  `COMMAND_TOPIC_MONSTER` for each monster to spawn, carrying world/channel/map/
  instance, the `monsterId`, and a body with `X`, `Y`, `Fh`, `Team`.
- FR-12: `Team` defaults to a neutral/hostile value consistent with normal map
  spawns (default `0` unless design determines otherwise).
- FR-13: For `count > 1`, atlas-messages emits `count` spawn commands (or a single
  command carrying count — design decides), each producing one monster at the
  resolved position.

### 4.7 Count bounds
- FR-14: `count` defaults to `1` when omitted.
- FR-15: `count` is clamped/validated to a maximum (proposed cap: **20**). A
  request above the cap is either rejected with a pink-text error or clamped to
  the cap — design decides the exact behavior, but the cap MUST be enforced to
  prevent accidental map flooding.
- FR-16: A `count` of `0` is treated as invalid (rejected with feedback) — the
  regex `(\d+)` already excludes negatives.

### 4.8 atlas-monsters SPAWN consumer
- FR-17: atlas-monsters adds a `SPAWN` command type constant and a
  `spawnCommandBody` struct (`X int16`, `Y int16`, `Fh int16`, `Team int8`) to its
  `COMMAND_TOPIC_MONSTER` Kafka contract.
- FR-18: A new handler `handleSpawnCommand` is registered in the monster command
  consumer's `InitHandlers`, discriminating on `Type == "SPAWN"`, mirroring the
  existing field-command handlers (`handleUseSkillFieldCommand`, etc.).
- FR-19: The handler builds the `field.Model` from the command's
  world/channel/map/instance and calls `monster.Processor.Create(f, RestModel{
  MonsterId, X, Y, Fh, Team})` — the existing spawn path. The Create method's
  existing template validation (`information.GetById`) remains as a server-side
  safety net.

### 4.9 Feedback
- FR-20: On successful dispatch, the GM receives a pink-text confirmation, e.g.
  `Spawned <count>x monster <id> at (<x>, <y>).` (monster name optional).
- FR-21: On validation failure, the GM receives a pink-text error and no spawn
  occurs (FR-7).
- FR-22: Feedback is issued via the existing `message.Processor.IssuePinkText`
  pattern used by the other `@mob` commands.

## 5. API Surface

No new externally-facing REST endpoints are introduced. The feature consumes
existing endpoints and adds one internal Kafka command type.

### 5.1 Consumed atlas-data endpoints (existing)
- `GET /data/monsters/{monsterId}` → `200` monster `RestModel` or `404`.
  Used for FR-6 template validation.
- `POST /data/maps/{mapId}/footholds/below` with body `{ "x": <int16>, "y": <int16> }`
  → `FootholdRestModel { Id uint32, First *Point, Second *Point }`.
  Used for FR-9 foothold resolution.

### 5.2 New Kafka command — `COMMAND_TOPIC_MONSTER` (atlas-monsters contract)
Extends the existing command envelope (no envelope change):
```go
// existing envelope
type command[E any] struct {
    WorldId   world.Id   `json:"worldId"`
    ChannelId channel.Id `json:"channelId"`
    MapId     _map.Id    `json:"mapId"`
    Instance  uuid.UUID  `json:"instance"`
    MonsterId uint32     `json:"monsterId"`
    Type      string     `json:"type"`   // new value: "SPAWN"
    Body      E          `json:"body"`
}

// new body
type spawnCommandBody struct {
    X    int16 `json:"x"`
    Y    int16 `json:"y"`
    Fh   int16 `json:"fh"`
    Team int8  `json:"team"`
}
```
A matching producer/provider is added on the atlas-messages side
(`monster.SpawnFieldCommandProvider(...)` analogous to the existing
`UseSkillFieldCommandProvider` / `DestroyFieldCommandProvider`).

## 6. Data Model

No new persistent entities, tables, or migrations. Monsters are created through
the existing atlas-monsters registry/Create path, which already persists runtime
monster state (Redis). All data is multi-tenant-scoped through the existing tenant
context propagated on the Kafka message headers and REST calls.

## 7. Service Impact

| Service | Change |
|---------|--------|
| **atlas-messages** | New command producer `MobSpawnCommandProducer` (regex match, GM gate, count parse). New atlas-data clients for monster-info validation (FR-6) and foothold-below resolution (FR-9), if not already present. New `SpawnFieldCommandProvider` + `SPAWN` constant in `kafka/message/monster`. Registration in `main.go`. Pink-text feedback. Help-text entry for `@mob spawn`. |
| **atlas-monsters** | New `SPAWN` command type constant + `spawnCommandBody` struct in `kafka/consumer/monster/kafka.go`. New `handleSpawnCommand` registered in the monster command consumer `InitHandlers`, calling existing `monster.Processor.Create`. |
| **atlas-data** | No change — existing `GET /data/monsters/{id}` and `POST /data/maps/{id}/footholds/below` endpoints are reused. |
| **atlas-saga-orchestrator** | No change — the saga spawn path is intentionally not used for this command. |

## 8. Non-Functional Requirements

- **Multi-tenancy**: All REST calls and the Kafka command must carry/propagate the
  tenant context already present in the message-handling flow
  (`tenant.MustFromContext`). Spawns must be scoped to the issuing tenant.
- **Security / authorization**: GM-only (FR-4). Non-GMs must not be able to detect
  or trigger the command. The numeric-only template parse prevents injection.
- **Safety**: The `count` cap (FR-15) prevents accidental or malicious map
  flooding. Foothold fallback (FR-10) prevents a foothold-lookup failure from
  blocking spawns.
- **Observability**: Failures (validation, foothold, dispatch) should be logged at
  an appropriate level in atlas-messages and atlas-monsters, consistent with the
  existing command handlers. Use the existing `logrus.FieldLogger` flow.
- **Latency**: Up to two synchronous atlas-data calls (validation + foothold) per
  command before dispatch. Acceptable for an interactive GM command; spawns
  themselves remain async via Kafka.
- **Parity**: Spawned monsters must rest on terrain like natural spawns. Note the
  known slope-spawn parity issue (mobs on slopes can fall through the floor if the
  ground y is off by one); the resolved foothold plus the GM's standing position
  should avoid this, but design/QA should verify on a sloped map.

## 9. Open Questions

1. **Where is the foothold resolved — atlas-messages or atlas-monsters?**
   Recommendation: resolve in **atlas-messages** (it already makes the validation
   call to atlas-data, and keeping `Fh` in the Kafka body keeps the atlas-monsters
   handler a thin pass-through). The alternative (resolve inside the atlas-monsters
   `handleSpawnCommand`) adds a REST dependency to atlas-monsters. Final call in
   design phase.
2. **One Kafka command with a `count` field, or N single-spawn commands?**
   Emitting N commands keeps the body identical to a single spawn and reuses the
   existing per-monster Create path; a `count` field would require loop logic in
   the consumer. Lean toward N commands unless design prefers otherwise.
3. **Over-cap behavior**: reject with an error message, or silently clamp to the
   cap (FR-15)? Proposed: clamp and inform (`Capped to 20.`).
4. **`Team` default value** for a GM-spawned monster — confirm `0` matches normal
   hostile map spawns (verify against atlas-monsters Create defaults / existing
   spawn callers).
5. **Stance**: `Create` uses a fixed stance (5) internally; confirm no stance input
   is needed from the command.

## 10. Acceptance Criteria

- [ ] `@mob spawn <templateId>` issued by a GM spawns exactly one monster of that
      template at the GM's current X/Y on their current map/channel/world/instance.
- [ ] `@mob spawn <templateId> <count>` spawns `count` monsters (1 ≤ count ≤ cap)
      at the GM's position.
- [ ] A `count` above the cap is clamped or rejected per FR-15, and the GM is
      informed.
- [ ] `@mob spawn <unknownId>` produces a pink-text error and spawns nothing.
- [ ] The spawned monster rests on terrain (foothold resolved; falls back to 0 on
      lookup failure without aborting).
- [ ] A non-GM issuing the command sees no effect and no acknowledgement; normal
      chat handling is unaffected.
- [ ] The new regex does not collide with `@mob kill all` (both still work).
- [ ] `@mob spawn` appears in the GM help command output.
- [ ] atlas-monsters consumes the new `SPAWN` command on `COMMAND_TOPIC_MONSTER`
      and creates the monster via the existing `Create` path.
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...` clean in both
      atlas-messages and atlas-monsters; `docker buildx bake atlas-messages` and
      `atlas-monsters` succeed; `tools/redis-key-guard.sh` clean.

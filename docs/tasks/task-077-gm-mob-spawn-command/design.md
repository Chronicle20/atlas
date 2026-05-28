# GM Command `@mob spawn` — Design

Task: task-077-gm-mob-spawn-command
PRD: `docs/tasks/task-077-gm-mob-spawn-command/prd.md` (approved)
Status: Design v1
Date: 2026-05-28

---

## 1. Summary

Add a GM chat command `@mob spawn <templateId> [count]` to **atlas-messages**.
It spawns 1–20 instances of a monster template at the issuing GM's current
position, on the GM's current world/channel/map/instance, by emitting a new
`SPAWN_FIELD` Kafka command on `COMMAND_TOPIC_MONSTER`. **atlas-monsters**
consumes the command and creates each monster through its existing
`monster.Processor.Create` path.

The command mirrors the existing `@mob` command family
(`MobKillAllCommandProducer`, `MobStatusCommandProducer`, `MobClearCommandProducer`)
exactly: a regex-matched `command.Producer`, a `c.Gm()` gate, pink-text feedback,
and a `FieldCommand`-envelope Kafka provider.

## 2. Key discovery that reshapes the PRD

**PRD FR-5 is incorrect.** It states the GM's `X()`/`Y()` are "already available
on the atlas-messages `character.Model` … No additional position query is
required." In reality:

- `character.Model.X()` and `Y()` are hardcoded stubs returning `0`
  (`services/atlas-messages/.../character/model.go:213-219`).
- The `Model` struct has **no** `x`/`y` fields (model.go:13-43).
- `character.Extract` (`character/rest.go:99-130`) silently drops `rm.X`/`rm.Y`
  even though the atlas-character REST resource **does** return them
  (`character/rest.go:41-43` — `X`, `Y`, `Stance` are present on the RestModel).

So the position is fetched from atlas-character on every `@mob` command (via
`message.HandleGeneral` → `character.GetById`) and then thrown away. Spawning at
the GM's feet therefore requires a small **plumb-through fix in atlas-messages**,
not a new service call. This is the approach chosen for this task (see §5.1).

Instance UUID does **not** need plumbing: the `field.Model f` passed into every
command producer already carries the live `world/channel/map/instance`
(`message.HandleGeneral(f field.Model, …)` at `message/processor.go:39`). The
existing `@mob` commands rebuild the field from `c.MapId()` and lose the instance
(`field.NewBuilder(ch.WorldId(), ch.Id(), c.MapId()).Build()` → `instance = Nil`);
the spawn command will instead use the **incoming `f`** directly so spawns work
correctly on instanced maps.

## 3. Resolved open questions (from PRD §9)

| # | Question | Decision | Rationale |
|---|----------|----------|-----------|
| — | GM position source (FR-5 wrong) | Plumb `X`/`Y` through atlas-messages `character.Model`; reuse data already on the REST response | Smallest change; no new service dependency (see §5.1) |
| 1 | Where is the foothold resolved? | **atlas-messages** | It already calls atlas-data for validation; keeps the atlas-monsters handler a thin pass-through. atlas-pets already calls the same endpoint (`data/position/requests.go`), so the client pattern is proven |
| 2 | One Kafka command with `count`, or N single-spawn commands? | **N single-spawn messages**, one per monster, all carrying the identical body | Body stays identical to a single spawn; the consumer loops nothing and reuses the existing per-monster `Create` path. Emitted as a single `Provider[[]kafka.Message]` of length `count` (one `Emit`) |
| 3 | Over-cap behavior | **Clamp to 20 and inform** ("Capped to 20.") | Confirmed with user; forgiving and still useful |
| 4 | `Team` default | **`0`** | Zero-value used by the existing REST/saga spawn path (`monster.RestModel.Team` is passed straight to `Create`; no GM team concept). `Create` does not override it (`monster/processor.go:189` passes `input.Team`) |
| 5 | Stance | **No stance input** | `Create` hard-codes stance `5` internally (`processor.go:189`) and ignores `RestModel.Stance`; nothing to pass |

**Envelope-shape deviation from PRD §5.2:** PRD sketched the `MonsterId`-carrying
`command[E]` envelope with `Type: "SPAWN"`. The actual `@mob` family in
atlas-messages emits only the **`FieldCommand[E]` envelope (no `MonsterId`)** with
`*_FIELD` type strings (`USE_SKILL_FIELD`, `DESTROY_FIELD`). A spawn's `monsterId`
is a *template* id, not a runtime unique-monster id (which is what the envelope's
`MonsterId` slot means for `DAMAGE`/`USE_SKILL`), so it belongs in the **body**.
The design therefore uses the field-command style: `Type: "SPAWN_FIELD"`, with
`monsterId` inside the body. This is consistent with the code as it exists.

## 4. Architecture & data flow

```
GM types "@mob spawn 100100 5"
        │  (chat Kafka event → message consumer)
        ▼
message.HandleGeneral(f, actorId, msg)            message/processor.go:39
        │  character.GetById(actorId)  ── REST ─▶ atlas-character GET /characters/{id}
        │      (now carries X/Y after §5.1 fix)
        ▼
command.Registry().Get(l, ctx, f, c, msg)
        ▼
MobSpawnCommandProducer  (NEW)                    command/monster/commands.go
   1. regex ^@mob spawn\s+(\d+)(?:\s+(\d+))?$  → templateId, count?
   2. c.Gm() gate                               (else return nil,false)
   3. count: default 1; reject 0; clamp >20 → 20 (+inform)
   ── executor ──────────────────────────────────────────────
   4. data/monster.GetById(templateId)  ── REST ─▶ atlas-data GET /data/monsters/{id}
         404/err → IssuePinkText "Unknown monster template: <id>"; STOP
   5. data/foothold.GetBelow(mapId,x,y) ── REST ─▶ atlas-data POST /data/maps/{id}/footholds/below
         err/none → Fh = 0 (fallback, continue)
   6. emit SpawnFieldCommandProvider(... , count)  ─ Kafka ─▶ COMMAND_TOPIC_MONSTER
         (count messages, identical body)
   7. IssuePinkText "Spawned 5x monster 100100 at (x, y). [Capped to 20.]"
        ▼
atlas-monsters  handleSpawnFieldCommand  (NEW)    kafka/consumer/monster/consumer.go
   - discriminate Type == "SPAWN_FIELD"
   - f := field.NewBuilder(c.WorldId, c.ChannelId, c.MapId).SetInstance(c.Instance).Build()
   - monster.NewProcessor(l,ctx).Create(f, RestModel{MonsterId,X,Y,Fh,Team})
        │  Create: information.GetById (server-side template safety net) → registry CreateMonster
        ▼
   monster persisted (Redis) + Created status event → channel spawns the mob
```

Position values flow: `X`/`Y` from `c.X()`/`c.Y()` (after §5.1 fix);
`world/channel/map/instance` from the incoming `field.Model f`; `Fh` from the
foothold lookup (or `0`); `Team` = `0`.

## 5. Component changes

### 5.1 atlas-messages — character position plumb-through (prerequisite)

File: `services/atlas-messages/atlas.com/messages/character/`

- **model.go**
  - Add `x int16`, `y int16` fields to `Model`.
  - Replace stub getters: `X()` → `return m.x`; `Y()` → `return m.y`.
  - `Clone(...)` must copy `x`/`y` into the builder (otherwise `SetSkills`, which
    round-trips through `Clone(...).Build()`, would zero the position).
  - `ModelBuilder.Build()` must copy `x`/`y` into the returned `Model`; add
    `SetX`/`SetY` setters for completeness/consistency with the other fields.
  - (`stance` is intentionally **not** plumbed — `Create` ignores stance.)
- **rest.go**
  - `Extract(rm)` copies `x: rm.X, y: rm.Y` into the `Model` literal.

This is a contained correctness fix; no other consumer of `character.Model`
depends on `X()`/`Y()` returning `0`.

### 5.2 atlas-messages — monster template validation client (NEW)

File: `services/atlas-messages/atlas.com/messages/data/monster/`
(model.go, rest.go, requests.go, processor.go) — mirrors the existing
`data/skill/` package.

- `requests.go`: `const monsterResource = "data/monsters/%d"`,
  `getBaseRequest() = requests.RootUrl("DATA")`,
  `requestById(id uint32) = requests.GetRequest[RestModel](…)`.
- `rest.go`: `RestModel{ Id uint32 \`json:"-"\`; Name string \`json:"name"\` }`,
  `GetName() = "monsters"`, JSON:API `GetID/SetID`. (Only `Id`+`Name` are needed;
  `Name` enriches the success message per FR-8.)
- `processor.go`: `Processor` interface with `GetById(id uint32) (Model, error)`
  + `ProcessorImpl`; `GetById` returns the upstream error on 404 (used as the
  "unknown template" signal). Interface form makes it mockable in tests.

### 5.3 atlas-messages — foothold-below client (NEW)

File: `services/atlas-messages/atlas.com/messages/data/foothold/`
— mirrors atlas-pets `data/position/`.

- `requests.go`: `const footholdBelowResource = "data/maps/%d/footholds/below"`,
  `getInMap(mapId, x, y) = requests.PostRequest[RestModel](…, PositionRestModel{X,Y})`.
- `rest.go`: `PositionRestModel{ X int16 \`json:"x"\`; Y int16 \`json:"y"\` }`,
  `RestModel{ Id uint32 \`json:"id"\`; First/Second *point \`json:"…,omitempty"\` }`,
  `GetName() = "footholds"`.
- `processor.go`: `Processor` interface `GetBelow(mapId, x, y) (Model, error)`;
  any error (the endpoint returns **500**, not 404, when no foothold is found)
  is non-fatal and maps to the `Fh = 0` fallback in the command executor.

### 5.4 atlas-messages — Kafka spawn provider

File: `services/atlas-messages/atlas.com/messages/kafka/message/monster/kafka.go`

- Add `CommandTypeSpawnField = "SPAWN_FIELD"`.
- Add body:
  ```go
  type SpawnFieldBody struct {
      MonsterId uint32 `json:"monsterId"`
      X         int16  `json:"x"`
      Y         int16  `json:"y"`
      Fh        int16  `json:"fh"`
      Team      int8   `json:"team"`
  }
  ```
- Add provider returning **`count`** identical messages (single `Emit`),
  partition key = `mapId` (matches existing providers):
  ```go
  func SpawnFieldCommandProvider(worldId world.Id, channelId channel.Id, mapId _map.Id,
      instance uuid.UUID, monsterId uint32, x, y, fh int16, team int8, count int) model.Provider[[]kafka.Message]
  ```
  Internally builds one `FieldCommand[SpawnFieldBody]{ Type: CommandTypeSpawnField, … }`
  and returns a slice of `count` messages with the same key/value.

### 5.5 atlas-messages — command producer (NEW)

File: `services/atlas-messages/atlas.com/messages/command/monster/commands.go`

`MobSpawnCommandProducer` mirroring `MobStatusCommandProducer`:

- Regex: `^@mob spawn\s+(\d+)(?:\s+(\d+))?$`. Whitespace-tolerant; cannot collide
  with `^@mob kill all$` ("spawn" ≠ "kill all"), `@mobstatus`, or `@mobclear`.
- `if !c.Gm() { return nil, false }` — non-GMs fall through, command not advertised.
- Parse: `templateId = match[1]` (uint32); `count = match[2]` else `1`.
  - `count == 0` → executor issues pink "Count must be at least 1." and dispatches
    nothing (FR-16; regex `\d+` does match `"0"`, so this is an explicit guard).
  - `count > 20` → set `count = 20`, remember a `capped` flag for the message.
- Executor (`command.Executor`):
  1. `data/monster.GetById(templateId)` → on error: pink
     `"Unknown monster template: <id>"`, return (no dispatch).
  2. `data/foothold.GetBelow(f.MapId(), c.X(), c.Y())` → on error/none: `fh = 0`,
     log at warn, continue.
  3. Emit `monster.SpawnFieldCommandProvider(f.WorldId(), f.ChannelId(), f.MapId(),
     f.Instance(), templateId, c.X(), c.Y(), fh, 0, count)` on
     `monster.EnvCommandTopic`. On emit error: pink `"Failed to spawn monster <id>."`.
  4. Success: pink `"Spawned <count>x monster <id> (<name>) at (<x>, <y>)."`
     + `" Capped to 20."` when `capped`.
- Uses the **incoming `f`** for world/channel/map/instance (preserves instance),
  and `c.X()/c.Y()` for position.

### 5.6 atlas-messages — registration & help text

- `main.go`: `command.Registry().Add(monster.MobSpawnCommandProducer)` alongside
  the other `@mob` registrations (~main.go:56-58).
- Help text: locate the existing GM help/`@`-command help output and add a
  `@mob spawn <templateId> [count]` line (acceptance criterion). The exact help
  producer is identified during planning; if no `@mob` help entry exists today,
  this reduces to a no-op and is noted in the plan.

### 5.7 atlas-monsters — SPAWN_FIELD consumer

Files: `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/{kafka.go,consumer.go}`

- **kafka.go**: add `CommandTypeSpawnField = "SPAWN_FIELD"` and
  ```go
  type spawnFieldCommandBody struct {
      MonsterId uint32 `json:"monsterId"`
      X         int16  `json:"x"`
      Y         int16  `json:"y"`
      Fh        int16  `json:"fh"`
      Team      int8   `json:"team"`
  }
  ```
  (uses the existing `fieldCommand[E]` envelope — no `MonsterId` in the envelope).
- **consumer.go**: register `handleSpawnFieldCommand` in `InitHandlers` (alongside
  `handleDestroyFieldCommand`), and:
  ```go
  func handleSpawnFieldCommand(l logrus.FieldLogger, ctx context.Context, c fieldCommand[spawnFieldCommandBody]) {
      if c.Type != CommandTypeSpawnField { return }
      f := field.NewBuilder(c.WorldId, c.ChannelId, c.MapId).SetInstance(c.Instance).Build()
      p := monster.NewProcessor(l, ctx)
      _, err := p.Create(f, monster.RestModel{
          MonsterId: c.Body.MonsterId, X: c.Body.X, Y: c.Body.Y, Fh: c.Body.Fh, Team: c.Body.Team,
      })
      if err != nil {
          l.WithError(err).Errorf("SPAWN_FIELD failed for template [%d] in field [%s].", c.Body.MonsterId, f.Id())
      }
  }
  ```
  `Create`'s `information.GetById` remains the server-side template safety net
  (FR-19).

### 5.8 atlas-data

No change. Both endpoints (`GET /data/monsters/{id}`,
`POST /data/maps/{id}/footholds/below`) already exist and are consumed by other
services.

## 6. Error handling

| Failure | Behavior |
|---------|----------|
| Non-GM issuer | Producer returns `(nil, false)`; normal chat handling proceeds; command not advertised |
| `count == 0` | Pink "Count must be at least 1."; no dispatch |
| `count > 20` | Clamp to 20; success message appends "Capped to 20." |
| Unknown template (atlas-data 404/err) | Pink "Unknown monster template: <id>"; no dispatch |
| Foothold lookup fails (atlas-data 500/none) | `Fh = 0` fallback; log warn; spawn proceeds |
| Kafka emit error | Pink "Failed to spawn monster <id>." |
| Template removed between validation and consume | `Create.information.GetById` fails server-side; logged in atlas-monsters; no monster created |

## 7. Testing

**atlas-messages (`command/monster`)** — table-driven tests on `MobSpawnCommandProducer`:
- matches `@mob spawn 100100` and `@mob spawn 100100 5`; whitespace tolerance.
- does **not** match `@mob kill all`, `@mobstatus …`, `@mobclear`, plain chat.
- non-GM (`c.Gm()==false`) → `(nil, false)`.
- count parsing: omitted → 1; `0` → rejected path; `>20` → clamped to 20.
- validation/dispatch paths exercised with mock `data/monster` and `data/foothold`
  `Processor` interfaces (unknown-template → no emit; valid → emit with expected
  body incl. resolved `Fh`, fallback `Fh=0` on foothold error).

**atlas-messages (`character`)** — `Extract` now sets `X()/Y()`; `Clone`+`Build`
round-trip preserves `x/y` (regression guard for the `SetSkills` path).

**atlas-monsters (`kafka/consumer/monster`)** — `handleSpawnFieldCommand`:
- ignores non-`SPAWN_FIELD` types.
- builds the field with the command's instance and constructs the expected
  `monster.RestModel` (MonsterId/X/Y/Fh/Team). Full `Create` behavior is already
  covered by existing `Create` tests.

## 8. Verification (per CLAUDE.md)

Changed Go modules: **atlas-messages**, **atlas-monsters**.
- `go test -race ./...`, `go vet ./...`, `go build ./...` clean in both.
- `docker buildx bake atlas-messages` and `docker buildx bake atlas-monsters`
  succeed (no new shared lib, so no Dockerfile/`go.work` edits expected).
- `tools/redis-key-guard.sh` clean (no new raw redis usage).

## 9. Out of scope (unchanged from PRD)

Arbitrary coordinates, cross-map/cross-character spawns, persistent/respawning
spawns, spawn-by-name, atlas-ui/client changes, natural-spawn changes.

# Context — task-077 `@mob spawn`

Companion to `plan.md`. Captures the key files, decisions, and dependencies an implementer needs.

## What this builds

A GM chat command `@mob spawn <templateId> [count]` in **atlas-messages** that spawns 1–20 instances of a monster template at the GM's current position by emitting `SPAWN_FIELD` Kafka commands consumed by **atlas-monsters**. No atlas-data, atlas-ui, or saga changes.

## Changed modules

| Module | Path | Why |
|--------|------|-----|
| atlas-messages | `services/atlas-messages/atlas.com/messages/` | Command, position plumb-through, two REST clients, Kafka provider, registration, help |
| atlas-monsters | `services/atlas-monsters/atlas.com/monsters/` | `SPAWN_FIELD` body + handler calling existing `Create` |

## Key existing files (read before editing)

- `command/monster/commands.go` — pattern source: `MobStatusCommandProducer` is the closest mirror (regex → `c.Gm()` gate → executor that emits a `FieldCommand` provider + `IssuePinkText`).
- `command/processor.go` — `Producer` / `Executor` type definitions (curried `func(l) func(ctx) func(f, c, m) (Executor, bool)`).
- `command/help/commands.go` — `@help` output is a `[]string` (`commandSyntaxList`); add one line.
- `message/processor.go` — `HandleGeneral` fetches the character (with X/Y once Task 1 lands) and passes the live `field.Model f` into `command.Registry().Get`. Use that `f` directly to preserve the instance.
- `character/model.go` + `character/rest.go` — position plumb-through target (see Decisions).
- `kafka/message/monster/kafka.go` — `FieldCommand[E]` envelope + existing `*FieldCommandProvider`s to mirror.
- `data/skill/` — mirror for the new `data/monster` client.
- atlas-pets `data/position/` (+ `mock/`) — mirror for the new `data/foothold` client.
- atlas-monsters `kafka/consumer/monster/{kafka.go,consumer.go}` — `fieldCommand[E]` envelope, `InitHandlers` registration, `handleDestroyFieldCommand` mirror.
- atlas-monsters `monster/processor.go:181` — `Create(f field.Model, input RestModel) (Model, error)`; `monster/rest.go:13` — `RestModel` (fields `MonsterId`, `X`, `Y`, `Fh`, `Team`, etc.). `Create` hard-codes stance 5 and passes `input.Team` through.

## Decisions (resolved from design)

1. **GM position is fetched then dropped — fix it.** `character.Model.X()`/`Y()` are `return 0` stubs; the `Model` struct has no `x`/`y`; `Extract` ignores `rm.X`/`rm.Y`. The `ModelBuilder` already declares **dead** `x`/`y`/`stance` fields (no setters, not copied in `Build`). Task 1 wires `x`/`y` end-to-end (struct field, getter, `Clone`, `SetX`/`SetY`, `Build`, `Extract`). `stance` stays unplumbed — `Create` ignores it.
2. **Instance preserved via incoming `f`.** Existing `@mob` commands rebuild the field from `ch.WorldId()/c.MapId()`, losing the instance. The new executor uses the incoming `f.WorldId()/f.ChannelId()/f.MapId()/f.Instance()` so spawns work on instanced maps.
3. **Envelope = `FieldCommand[E]`, type `SPAWN_FIELD`.** Not the PRD's `MonsterId`-carrying `command[E]`. `monsterId` is a *template* id and goes in the body, not the envelope's runtime-`MonsterId` slot.
4. **N identical messages, not a `count` field.** `SpawnFieldCommandProvider` returns a `count`-length `[]kafka.Message` (single `Emit`), each producing one monster via the existing per-monster `Create`.
5. **Count: default 1, reject 0, clamp >20 (cap 20) and inform.** Pure helpers `parseSpawnArgs` (regex/parse) and `normalizeCount` (validate/clamp) keep this unit-testable.
6. **Foothold resolved in atlas-messages, non-fatal.** `POST /data/maps/{id}/footholds/below` returns **500** (not 404) on no foothold; any error → `Fh = 0` fallback + warn log, spawn proceeds. The new `data/foothold` `Extract` only reads `Id` (avoids the nil-deref present in the pets `Extract`).
7. **`Team = 0`** (zero-value used by existing spawn path). **No stance input.**

## Dependencies / gotchas

- **Import collision:** `command/monster/commands.go` already imports `kafka/message/monster` as `monster` and `atlas-constants/monster` as `monster2`. Import the new template client as `monsterdata "atlas-messages/data/monster"`.
- **Foothold id cast:** foothold ids are `int16` in this system (`monster.RestModel.Fh` and `movementCommand.Fh` are `int16`); cast `int16(fhModel.Id())`.
- **`model.FixedProvider` + `producer.MessageProvider` + `producer.RawMessage`** are the building blocks for emitting N identical messages (see `libs/atlas-kafka/producer/message.go`).
- **Test idiom:** command tests in this repo cover regex/GM-gate/parse only (not executor REST/Kafka dispatch); body structs are tested via `json.Unmarshal`. The plan follows both idioms. `mock` subpackages are created for both REST clients to match convention, even though the executor isn't unit-tested.
- **Verification gate (CLAUDE.md):** `go test -race`, `go vet`, `go build` in both modules; `docker buildx bake atlas-messages` and `atlas-monsters` from the worktree root; `tools/redis-key-guard.sh`. No new shared lib → no Dockerfile/`go.work` edits.

## Acceptance criteria (from PRD §10)

GM `@mob spawn <id>` spawns 1 at position; `<id> <count>` spawns count (1–20); over-cap clamps + informs; unknown id → pink error, no spawn; monster rests on terrain (foothold, fallback 0); non-GM no-op; regex doesn't collide with `@mob kill all`; help line present; atlas-monsters consumes `SPAWN_FIELD` via `Create`; full verification gate green.

# Task 030 — Implementation context

For an engineer who has zero context for this codebase. Read this once
before working through `plan.md`.

## What this fix is

A multi-line player attack (e.g., L7) currently produces one Kafka `DAMAGE`
command per damage line per targeted monster. atlas-monsters processes each
independently and emits either a `damaged` or `killed` event for each. The
killing line emits `killed` instead of `damaged`, and the channel only writes
the HP-bar packet on `damaged` — so the HP bar visually skips the killing
line's drain.

We fix this by batching all lines for a single monster into one Kafka command
carrying `Damages []uint32`. atlas-monsters applies the lines in a Go loop in
a single goroutine and always emits a `damaged` event reflecting the final
state, plus a `killed` event when the attack lands a kill. The channel sees
`damaged → killed` and writes the final HP-bar drain before the death packet.

Read `design.md` in this folder before starting. Don't deviate from it
without a design update.

## Repo layout you'll touch

Two services, both Go modules:

- `services/atlas-channel/atlas.com/channel/` — module `atlas-channel`
- `services/atlas-monsters/atlas.com/monsters/` — module `atlas-monsters`

Shared libs sit in `libs/`. No shared lib changes are needed for this work.

## Cross-service contract

The contract is the JSON body of `COMMAND_TOPIC_MONSTER` messages of type
`DAMAGE`. Each service has its own copy of the body struct (no shared schema):

- Producer side (atlas-channel):
  `services/atlas-channel/atlas.com/channel/kafka/message/monster/kafka.go`
  → struct `DamageCommandBody` (exported, used by producer)
- Consumer side (atlas-monsters):
  `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/kafka.go`
  → struct `damageCommandBody` (unexported, used by consumer)

Both must be renamed `Damage uint32` → `Damages []uint32` in lockstep. The
JSON tag changes from `"damage"` to `"damages"`.

## Patterns the codebase follows (do not invent new ones)

- **Processor interface + Impl**: `monster.NewProcessor(l, ctx)` returns an
  interface; the implementation lives in the same package. When you change a
  method signature you change BOTH the interface and the impl.
- **Provider functions return `model.Provider[[]kafka.Message]`**: a
  zero-arg function that, when called, materialises the messages. Test by
  calling the provider and inspecting the resulting `kafka.Message.Value`
  (which is JSON bytes).
- **Curried logger/ctx**: `producer.ProviderImpl(l)(ctx)(topic)(provider)`.
- **Registry singletons**: `GetMonsterRegistry()` returns a redis-backed
  thread-safe singleton initialised in `TestMain` for tests via miniredis.
- **Tests live next to the file under test**: `foo_test.go` next to `foo.go`.

## What does and does not have unit-test coverage

- **Has tests**: registry redis ops (`monster/registry_test.go`), some
  builders, some channel handlers (e.g.
  `socket/handler/character_attack_projectile_test.go`).
- **No tests today**: `monster/processor.go` (atlas-monsters and
  atlas-channel both), kafka consumer handlers, kafka producer command
  providers, channel monster status-event handlers.

The plan adds tests where the existing surface is testable (JSON
marshal/unmarshal of the body structs, the channel-side `DamageCommandProvider`
output bytes). The processor-side change cannot be unit-tested without
introducing a Kafka producer abstraction, which is out of scope. We rely on
existing registry tests to cover damage accumulation, plus a manual in-game
smoke test for the end-to-end behaviour change.

## Build & test commands

Run from the service module root.

atlas-monsters:
```
cd services/atlas-monsters/atlas.com/monsters
go build ./...
go test ./...
```

atlas-channel:
```
cd services/atlas-channel/atlas.com/channel
go build ./...
go test ./...
```

Both modules must build and test green at the end.

## Manual smoke test

Required before merging. The unit tests do not cover the actual user-visible
behaviour change.

1. Bring up the local stack (atlas-channel, atlas-monsters, kafka, redis).
2. Log a character into a map with weak monsters (e.g., snails / orange
   mushrooms) the character can two-shot with L7.
3. Cast L7 at a single monster.
4. Observe the HP bar: it must drain twice (once per damage line) before
   the monster dies, not once.
5. Repeat for a single-shot kill: HP bar must drain to 0% before the
   death animation. (This is also a behaviour change vs. today.)

If you cannot run the full stack, say so in the PR description rather than
claiming the smoke test passed.

## Branch & commit conventions

- Branch already exists: `feature/task-030-multi-line-damage-batching`.
- Commit per logical step (the plan delineates these). Conventional commit
  prefixes: `feat:`, `fix:`, `refactor:`, `test:`, `chore:`.
- Never push to `main`. Project memory: branch protection blocks direct
  pushes to main.

## Reference: existing call sites you'll change

Lock these in your head — they are the entire surface area of the change.

atlas-channel (producer side):
1. `kafka/message/monster/kafka.go` — `DamageCommandBody` (struct)
2. `monster/producer.go` — `DamageCommandProvider` (line ~84)
3. `monster/processor.go` — `Processor.Damage` (line 43)
4. `socket/handler/character_attack_common.go` — inner loop at lines 67-73

atlas-monsters (consumer side):
5. `kafka/consumer/monster/kafka.go` — `damageCommandBody` (struct)
6. `kafka/consumer/monster/consumer.go` — `handleDamageCommand` (line 66)
7. `monster/processor.go` — `Processor` interface `Damage` (line 40),
   `ProcessorImpl.Damage` (line 230)

Nothing else needs to change. The status-event schema (`StatusEventDamagedBody`,
`StatusEventKilledBody`) is unchanged. The redis `applyDamageScript` is
unchanged. The channel-side status-event handler is unchanged (it already
writes the HP-bar packet on every `damaged` event).

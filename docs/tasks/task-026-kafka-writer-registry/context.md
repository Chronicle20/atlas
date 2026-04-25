# Kafka Writer Registry — Quick Reference Context

Companion to `prd.md`, `design.md`, `migration-plan.md`, `plan.md`.

## What this task does

Replace the per-publish `kafka.Writer` construct/close cycle in `libs/atlas-kafka/producer/` with a process-wide singleton registry of long-lived, per-topic Writers. Then migrate every service's producer wrapper and `main.go` to use it.

The visible payoff: the default `LeastBytes` balancer can finally distribute messages across partitions because Writer state persists across publishes. Today every publish gets a fresh Writer with all balancer counters at zero, which deterministically clusters `nil`-key publishes onto partition 0.

## Key facts the executor needs

- **Library home:** `libs/atlas-kafka/producer/` — module `github.com/Chronicle20/atlas/libs/atlas-kafka`. Go 1.25.
- **Singleton precedent:** `consumer.GetManager()` already exists in the same library; mirror it.
- **Teardown manager:** `service.GetTeardownManager()` from `libs/atlas-service/`. `TeardownFunc(f func())` registers a goroutine that fires on signal. **Teardown funcs run concurrently — there is no FIFO/LIFO ordering between them.**
- **`kafka.Writer` is documented as safe for concurrent use.** Many goroutines may share one Writer. The registry depends on this.
- **`topic.EnvProvider(l)(token)()` never returns an error** in the current code (missing env var falls back to using the token as the topic name). Tests cannot meaningfully exercise the resolution-error path with the live resolver.

## Files in scope

### Library (4 files)

```
libs/atlas-kafka/producer/manager.go         (new)
libs/atlas-kafka/producer/manager_test.go    (new)
libs/atlas-kafka/producer/producer.go        (modify: delete WriterProvider, drop w.Close())
libs/atlas-kafka/producer/producer_test.go   (no change — verify still passes)
```

### Per-service `kafka/producer/producer.go` wrappers — 47 files

Files containing `func ProviderImpl`. Each gets the identical mechanical edit:

- Replace `producer.WriterProvider(topic.EnvProvider(l)(token))` with `producer.ManagerWriterProvider(l)(token)`.
- Drop the now-unused `"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"` import.

Full list (alphabetical, generated `2026-04-25`):

```
services/atlas-account/atlas.com/account/kafka/producer/producer.go
services/atlas-asset-expiration/atlas.com/asset-expiration/kafka/producer/producer.go
services/atlas-ban/atlas.com/ban/kafka/producer/producer.go
services/atlas-buddies/atlas.com/buddies/kafka/producer/producer.go
services/atlas-buffs/atlas.com/buffs/kafka/producer/producer.go
services/atlas-cashshop/atlas.com/cashshop/kafka/producer/producer.go
services/atlas-chairs/atlas.com/chairs/kafka/producer/producer.go
services/atlas-chalkboards/atlas.com/chalkboards/kafka/producer/producer.go
services/atlas-channel/atlas.com/channel/kafka/producer/producer.go
services/atlas-character/atlas.com/character/kafka/producer/producer.go
services/atlas-character-factory/atlas.com/character-factory/kafka/producer/producer.go
services/atlas-consumables/atlas.com/consumables/kafka/producer/producer.go
services/atlas-data/atlas.com/data/kafka/producer/producer.go
services/atlas-drops/atlas.com/drops/kafka/producer/producer.go
services/atlas-effective-stats/atlas.com/effective-stats/kafka/producer/producer.go
services/atlas-expressions/atlas.com/expressions/kafka/producer/producer.go
services/atlas-fame/atlas.com/fame/kafka/producer/producer.go
services/atlas-families/atlas.com/family/kafka/producer/producer.go
services/atlas-guilds/atlas.com/guilds/kafka/producer/producer.go
services/atlas-inventory/atlas.com/inventory/kafka/producer/producer.go
services/atlas-invites/atlas.com/invites/kafka/producer/producer.go
services/atlas-keys/atlas.com/keys/kafka/producer/producer.go
services/atlas-login/atlas.com/login/kafka/producer/producer.go
services/atlas-map-actions/atlas.com/map-actions/kafka/producer/producer.go
services/atlas-maps/atlas.com/maps/kafka/producer/producer.go
services/atlas-marriages/atlas.com/marriages/kafka/producer/producer.go
services/atlas-merchant/atlas.com/merchant/kafka/producer/producer.go
services/atlas-messages/atlas.com/messages/kafka/producer/producer.go
services/atlas-messengers/atlas.com/messengers/kafka/producer/producer.go
services/atlas-monster-death/atlas.com/monster/kafka/producer/producer.go
services/atlas-monsters/atlas.com/monsters/kafka/producer/producer.go
services/atlas-notes/atlas.com/notes/kafka/producer/producer.go
services/atlas-npc-conversations/atlas.com/npc/kafka/producer/producer.go
services/atlas-npc-shops/atlas.com/npc/kafka/producer/producer.go
services/atlas-parties/atlas.com/parties/kafka/producer/producer.go
services/atlas-party-quests/atlas.com/party-quests/kafka/producer/producer.go
services/atlas-pets/atlas.com/pets/kafka/producer/producer.go
services/atlas-portal-actions/atlas.com/portal/kafka/producer/producer.go
services/atlas-portals/atlas.com/portals/kafka/producer/producer.go
services/atlas-reactor-actions/atlas.com/reactor/kafka/producer/producer.go
services/atlas-reactors/atlas.com/reactors/kafka/producer/producer.go
services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/producer/producer.go
services/atlas-skills/atlas.com/skills/kafka/producer/producer.go
services/atlas-storage/atlas.com/storage/kafka/producer/producer.go
services/atlas-tenants/atlas.com/tenants/kafka/producer/producer.go
services/atlas-transports/atlas.com/transports/kafka/producer/producer.go
services/atlas-world/atlas.com/world/kafka/producer/producer.go
```

### Non-standard producer callsites — 4 files

These call `producer.Produce(l)(producer.WriterProvider(topic.EnvProvider(l)(...)))` directly without going through `ProviderImpl`. Each gets the same `WriterProvider` → `ManagerWriterProvider` substitution:

```
services/atlas-quest/atlas.com/quest/kafka/producer/quest/producer.go         (emitEvent helper)
services/atlas-quest/atlas.com/quest/kafka/producer/saga/producer.go          (one direct call)
services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/party_quest/processor.go
services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/reactor/processor.go
```

### Service `main.go` files — 48 files

Add one teardown line to each `main.go` in the 47 services with a `ProviderImpl` wrapper plus `atlas-quest` (which uses the producer library via non-standard sub-wrappers):

```go
tdm.TeardownFunc(func() { _ = producer.GetManager().Close(l) })
```

Mapping `service → main.go`:

```
atlas-account              services/atlas-account/atlas.com/account/main.go
atlas-asset-expiration     services/atlas-asset-expiration/atlas.com/asset-expiration/main.go
atlas-ban                  services/atlas-ban/atlas.com/ban/main.go
atlas-buddies              services/atlas-buddies/atlas.com/buddies/main.go
atlas-buffs                services/atlas-buffs/atlas.com/buffs/main.go
atlas-cashshop             services/atlas-cashshop/atlas.com/cashshop/main.go
atlas-chairs               services/atlas-chairs/atlas.com/chairs/main.go
atlas-chalkboards          services/atlas-chalkboards/atlas.com/chalkboards/main.go
atlas-channel              services/atlas-channel/atlas.com/channel/main.go
atlas-character            services/atlas-character/atlas.com/character/main.go
atlas-character-factory    services/atlas-character-factory/atlas.com/character-factory/main.go
atlas-consumables          services/atlas-consumables/atlas.com/consumables/main.go
atlas-data                 services/atlas-data/atlas.com/data/main.go
atlas-drops                services/atlas-drops/atlas.com/drops/main.go
atlas-effective-stats      services/atlas-effective-stats/atlas.com/effective-stats/main.go
atlas-expressions          services/atlas-expressions/atlas.com/expressions/main.go
atlas-fame                 services/atlas-fame/atlas.com/fame/main.go
atlas-families             services/atlas-families/atlas.com/family/main.go
atlas-guilds               services/atlas-guilds/atlas.com/guilds/main.go
atlas-inventory            services/atlas-inventory/atlas.com/inventory/main.go
atlas-invites              services/atlas-invites/atlas.com/invites/main.go
atlas-keys                 services/atlas-keys/atlas.com/keys/main.go
atlas-login                services/atlas-login/atlas.com/login/main.go
atlas-map-actions          services/atlas-map-actions/atlas.com/map-actions/main.go
atlas-maps                 services/atlas-maps/atlas.com/maps/main.go
atlas-marriages            services/atlas-marriages/atlas.com/marriages/main.go
atlas-merchant             services/atlas-merchant/atlas.com/merchant/main.go
atlas-messages             services/atlas-messages/atlas.com/messages/main.go
atlas-messengers           services/atlas-messengers/atlas.com/messengers/main.go
atlas-monster-death        services/atlas-monster-death/atlas.com/monster/main.go
atlas-monsters             services/atlas-monsters/atlas.com/monsters/main.go
atlas-notes                services/atlas-notes/atlas.com/notes/main.go
atlas-npc-conversations    services/atlas-npc-conversations/atlas.com/npc/main.go
atlas-npc-shops            services/atlas-npc-shops/atlas.com/npc/main.go
atlas-parties              services/atlas-parties/atlas.com/parties/main.go
atlas-party-quests         services/atlas-party-quests/atlas.com/party-quests/main.go
atlas-pets                 services/atlas-pets/atlas.com/pets/main.go
atlas-portal-actions       services/atlas-portal-actions/atlas.com/portal/main.go
atlas-portals              services/atlas-portals/atlas.com/portals/main.go
atlas-quest                services/atlas-quest/atlas.com/quest/main.go
atlas-reactor-actions      services/atlas-reactor-actions/atlas.com/reactor/main.go
atlas-reactors             services/atlas-reactors/atlas.com/reactors/main.go
atlas-saga-orchestrator    services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/main.go
atlas-skills               services/atlas-skills/atlas.com/skills/main.go
atlas-storage              services/atlas-storage/atlas.com/storage/main.go
atlas-tenants              services/atlas-tenants/atlas.com/tenants/main.go
atlas-transports           services/atlas-transports/atlas.com/transports/main.go
atlas-world                services/atlas-world/atlas.com/world/main.go
```

## Decisions taken (settled in design.md)

| # | Question | Decision |
|---|---|---|
| 1 | Registry naming | `Manager` + `GetManager()` |
| 2 | Concurrency | `sync.RWMutex` + double-checked locking |
| 3 | `WriterProvider` fate | Deleted in this PR |
| 4 | Service-wrapper accessor | New `producer.ManagerWriterProvider(l)(token) model.Provider[Writer]` |
| 5 | `main.go` wiring | Lazy — no `Init`. One line: `tdm.TeardownFunc(func(){ _ = producer.GetManager().Close(l) })` |
| 6 | Test seam | `ResetInstance()` + `ConfigWriterFactory(WriterFactory)` configurator |
| 7 | Debug HTTP handler | Deferred |

## Deviation from design

The design's table of "Required new tests" lists `TestManager_TopicResolutionError`. The current `topic.EnvProvider` (`libs/atlas-kafka/topic/topic.go:13`) never returns an error — a missing env var falls back to using the token as the topic name. Without modifying `topic.EnvProvider` (out of scope) or adding another injection seam (unwarranted complexity), this test cannot be written meaningfully against the live resolver. The plan drops it. The error-propagation code path in `Manager.Writer` is still implemented, but it remains defensive against a future change to `topic.EnvProvider`'s contract.

## Verification commands

Per-affected-service build:

```bash
cd services/<svc>/atlas.com/<name> && go build ./... && go test ./...
```

Library:

```bash
cd libs/atlas-kafka && go test ./...
```

Pre-merge greps (from design §7):

```bash
grep -rn "producer\.WriterProvider" services/ libs/                              # expect 0
grep -rln "producer\.ManagerWriterProvider" services/ | wc -l                    # expect 47
grep -rln "producer\.GetManager().Close" services/                               # expect 47
grep -rn "kafka\.Writer{" services/                                              # expect 0
```

Smoke test runbook: `design.md` §8 (atlas-data + ≥4-partition `command.data` topic).

## Common pitfalls to avoid

- Do **not** add a per-Writer `Init(l)` ceremony — `sync.Once` inside `GetManager()` handles initialization on first use.
- Do **not** pre-resolve topic names in `ManagerWriterProvider`; the returned thunk must call `mgr.Writer(l, token)` lazily so `Produce`'s call-time error path keeps working.
- Do **not** keep `WriterProvider` as a deprecated alias. The single-PR migration deletes it. If a callsite is missed, the build will fail and surface it — that's a feature, not a bug.
- Do **not** reorder existing `tdm.TeardownFunc(tracing.Teardown(l)(tc))` calls. Their existing placement (after `Run()`) is a pre-existing pattern unrelated to this task. Add the producer teardown line where the design specifies (after consumer-handler init, before `server.New(l)...Run()`).
- Do **not** add `t.Parallel()` to manager tests — they share a singleton via `ResetInstance()` and must run serially.

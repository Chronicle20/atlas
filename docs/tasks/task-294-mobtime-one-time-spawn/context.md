# Context — task-294 one-time spawn for `mobTime == -1`

Companion to `plan.md`. Everything here was read out of this worktree, not recalled.

## Module and scope

- Single Go module: `services/atlas-maps/atlas.com/maps` (module name `atlas-maps`). Every
  `go build` / `go test` in the plan runs from there.
- `libs/atlas-redis` is **read-only**: `TenantKeyedHash` already exposes `Key`, `Len`, `Del`,
  `SetNX`, `Exists`, `GetAll`, `ClearForTenantId`, `ClearAllAcrossTenants`. Nothing in the lib
  needs to change.
- `services/atlas-monsters` is **read-only**: it already consumes `DESTROY_FIELD`.
- `services/atlas-maps/docs/domain.md` is the only non-Go file edited (Task 7).
- No migration, no schema change, no k8s manifest change. `COMMAND_TOPIC_MONSTER` is already
  in `deploy/k8s/base/env-configmap.yaml:68`, and `atlas-maps` gets the whole `atlas-env`
  ConfigMap via `envFrom` (`deploy/k8s/base/atlas-maps.yaml:21-23`).

## Key files and what they do today

| File | Role |
|---|---|
| `data/map/monster/rest.go:39` | `Extract` — currently drops `RestModel.Hide` |
| `data/map/monster/processor.go:46,54,58` | `SpawnableSpawnPointProvider` / `GetSpawnableSpawnPoints` / `Spawnable` — the `MobTime >= 0` gate that discards one-time points |
| `map/monster/registry.go:44-81` | `SpawnPointRegistry`, one `TenantKeyedHash` under namespace `maps:spawn`, plus a package-level `spawnHashKey` |
| `map/monster/registry.go:102-190` | Three Lua scripts: `initializeScript`, `reserveEligibleScript`, `resetCooldownScript` |
| `map/monster/registry.go:194-234` | `InitializeForMap` — guards on `Len > 0`, seeds from `GetSpawnableSpawnPoints` |
| `map/monster/processor.go:99-121` | `Count` → `getMonsterMax` → `toSpawn` — the block Task 4 rewrites, including the silent `totalCount == 0` return |
| `map/processor.go:105-110` | `Exit` — today only `cp.Exit` + a buffered `CHARACTER_EXIT` |
| `map/producer.go:47-63` | `enterMapActionsProvider` — the existing precedent for a cross-service command provider emitted from `_map` |
| `kafka/consumer/data/handler.go:37` | `DATA_UPDATED` → `FlushTenant`. **Unchanged by this task** (design D10) |

## Decisions carried from design.md, and the one placement deviation

- **Three keys, one namespace** (D1). `ClearForTenantId` SCANs
  `<prefix>:maps:spawn:<tenantId>:*`, which is namespace-wide per tenant, so `FlushTenant`
  and the `DATA_UPDATED` handler sweep all three key shapes with zero code change. Bumping to
  a second namespace would turn a self-reaping remnant into a permanent leak.
- **`v2:` key token** (D2). The JSON value shape does not change; the *content* would have
  gone stale, because `InitializeForMap`'s already-seeded guard would never re-seed a field
  that the old code had already populated with only the recurring subset.
- **`Classify` as a pure function, not more filtered providers** (D3). One `GetSpawnPoints`
  drain plus an in-memory partition is one paginated HTTP fetch per field init; two providers
  would be two.
- **`storedSpawnPoint` gains no `Hide` field** (D4). Hidden points never enter either hash,
  so "hidden" is not a state the registry can be in.
- **Deviation from D7's file placement.** The design put `DestroyFieldCommandProvider` in a new
  `monster/producer.go`. The plan puts an unexported `destroyFieldCommandProvider` in
  `map/producer.go` instead, next to `enterMapActionsProvider`, which is the identical shape
  (a command to another service's topic, emitted from `_map`). This avoids a third `monster`
  import alias inside `_map` and keeps the provider with its only emitter. The envelope, the
  topic, and the `RearmOneTime() == true` gating are exactly as D7 specifies — this is a
  placement-only change.
- **Latent bug fixed in passing** (Task 2 Step 3). `spawnHashKey` read
  `registryInstance.hashes` rather than the receiver's hash, silently bypassing any
  non-singleton registry — which is exactly what `registry_test.go`'s `newTestRegistry`
  builds. Replaced by three per-receiver key helpers.

## Test infrastructure

- Both `map/monster` and `map` have a `TestMain` that runs `miniredis.Run()` and wires
  `InitRegistry(rc)` (`map/monster/processor_test.go:29-39`, `map/processor_test.go:32-42`).
  The Lua paths therefore execute for real in tests, not against a fake.
- miniredis v2.38.0 registers `HSETNX`, `HDEL`, `HEXISTS` and `HLEN`
  (`cmd_hash.go:17,18,24,28`), so both new scripts run under test.
- `registry_test.go` builds its own registry via `newTestRegistry`; Task 2 rewires it through
  the extracted `newRegistry` constructor so the two never drift.
- `map/processor_test.go` already has `mockProducerProvider`, `createTestContext`,
  `createTestProcessor` and a `mockCharacterProcessor` with an injectable
  `getCharactersInMapFunc` — everything Task 6 needs.

## Cross-service seam (the review gate's own example)

`atlas-monsters`' consumer envelope, read verbatim from
`services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/kafka.go:252-259`:

```go
type fieldCommand[E any] struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
	Type      string     `json:"type"`
	Body      E          `json:"body"`
}
```

with `destroyFieldCommandBody struct{}` (`:103`) and
`CommandTypeDestroyField = "DESTROY_FIELD"` (`:26`), handled at `consumer.go:356-367` →
`DestroyInField(f)`. **There is no `transactionId`.** A drifted or extra key here fails open
with no error anywhere, which is why Task 5's test asserts the emitted key set exactly rather
than round-tripping through a struct.

## Task sizing

Seven tasks, strictly sequential. None exceeds three edited files or one service, so
`plan-lint.sh` F4 should be silent. The split points were chosen so every task leaves the
module compiling:

- Task 1 adds `Classify` but leaves the old `Spawnable*` methods in place; deleting them in
  the same task would break `registry.go:205` before Task 2 rewires it.
- Tasks 2 and 3 both edit `registry.go`, split at "state layout and seeding" vs. "claim and
  re-arm". They could be one task, but the review surface is meaningfully different: Task 2
  is the backward-compatibility risk (recurring path, key schema, flush coverage), Task 3 is
  the concurrency risk.
- Task 7 is the deletion sweep, deferred to last so no intermediate commit has a dangling
  caller. It also carries the flagless `tools/verify.sh` gate.

No task is deliberately oversized.

## Open risk carried into execution

The crash window in D5 is not mitigated: if the pod dies between `ClaimOneTimeSpawnPoints`
returning and `CreateMonster` being issued, the batch is lost until the field re-arms. This is
inherent to any claim-then-act split, `CreateMonster` is already fire-and-forget on the
recurring path, and no alternative considered in the design avoids it. Noted, not fixed.

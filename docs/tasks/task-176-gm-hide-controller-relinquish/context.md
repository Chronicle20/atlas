# task-176 — Context for Execution

Companion to [plan.md](plan.md). Key files, decisions, and dependencies gathered at plan time so executors don't re-derive them.

## What this task does

GM-hide (SuperGmHide 9101004 APPLIED/EXPIRED on `EVENT_TOPIC_CHARACTER_BUFF_STATUS`) must (a) relinquish the GM's monster controllers and exclude them from monster-controller election in **atlas-monsters**, and (b) introduce single-controller-per-NPC election in **atlas-channel** (today every session gets the controller grant) with the same hide semantics, plus an NPC action relay so non-controllers still see NPC motion.

## Key files

| Area | File | Why it matters |
|---|---|---|
| Monster election choke point | `services/atlas-monsters/atlas.com/monsters/monster/processor.go:260` (`getControllerCandidate`), `:305` (`FindNextController`) | ALL election paths funnel here; hidden filter + `ErrNoControllerCandidate` sentinel go here |
| DPS-leader switch | same file `:473` (in `Damage`) | assigns control OUTSIDE the choke point — needs its own hidden guard |
| Snapshot-first precedent | `services/atlas-monsters/atlas.com/monsters/kafka/consumer/map/consumer.go:53-83` | hide-relinquish must copy this two-step (snapshot → StopControl → FindNextController) and its provider-re-evaluation comment |
| Registry pattern to mirror | `services/atlas-monsters/atlas.com/monsters/monster/puppet_registry.go`, `registry.go` (`storedMonster`/`fromStored`, `GetMonsters`) | payload registry with tenant identity + SET index; `sync.Once` init from `main.go` |
| Buff event contract | `services/atlas-buffs/atlas.com/buffs/kafka/message/character/kafka.go:60-90` | `StatusEvent{WorldId, CharacterId, Type, Body}`; only APPLIED/EXPIRED exist; body carries `SourceId`; NO field on the event (by design — resolve live) |
| Location authority | `services/atlas-maps/atlas.com/maps/character/location/resource.go:35`, `rest.go` | `GET /characters/{characterId}/location`, JSON:API type `character-locations`, attrs worldId/channelId/mapId/instance; 404 = no location row |
| Channel NPC spawn path | `services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go:584` (`spawnNPCForSession`), `:221` (invocation in `SpawnForSelf`), `:554` (exit handler) | unconditional grant to every session is the thing being replaced; exit handler gains release/reassign |
| Channel buff consumer | `services/atlas-channel/atlas.com/channel/kafka/consumer/buff/consumer.go` | shape for the two new hide/reveal handlers (`IfPresentByCharacterId` + `s.Field()` — no maps lookup on channel side) |
| Hide semantics (task-156) | `services/atlas-channel/atlas.com/channel/character/buff/hidden.go` (`IsGmHidden`), `kafka/consumer/map/consumer.go:465-475` (suppression choke point) | winner-check = `buff.GetByCharacterId` + `IsGmHidden`; key on SourceId, never the DARK_SIGHT stat |
| NPC movement echo | `services/atlas-channel/atlas.com/channel/movement/processor.go:79` (`ForNPC`), `socket/handler/npc_action.go` | today echoes ONLY to the mover; needs controller guard + `ForOtherSessionsInMap` relay |
| Grant packet | `libs/atlas-packet/npc/clientbound/spawn_request_controller.go` | hard-codes flag byte `1`; writer name `SpawnNPCRequestController` reused by the new remove arm (same opcode) |
| Redis primitives | `libs/atlas-redis/keyed_set.go` (`TenantKeyedSet`), `keyed_hash.go` (`TenantKeyedHash`, **`SetNX` at :33 already exists**), `registry.go` (`Registry.GetAll`) | design's "add SetNX" is stale — no lib change needed |
| Channel session/map APIs | `services/atlas-channel/atlas.com/channel/map/processor.go:48` (`GetCharacterIdsInMap`), `:103` (`ForOtherSessionsInMap`); `session/processor.go:233` (`GetByCharacterId`), `IfPresentByCharacterId` | election candidate pool + announcement fan-out |

## Decisions (design D1-D6 + plan-time findings)

1. **atlas-channel owns NPC election** (D1) — no new service; state is one `TenantKeyedHash` per field (`npc-controller` namespace, suffix `<world>:<channel>:<map>:<instance>`, hash field npcId → charId). Uncontrolled = absent (D2). Claim = `HSETNX`; stale entries (controller not in field sessions) repaired lazily; Redis drops empty hashes → no teardown sweep.
2. **Monsters get a Redis hidden projection; channel does not** (D3). Channel winner-checks the single election winner against atlas-buffs REST (0-1 calls per election).
3. **Hidden registry stores tenant identity in the payload** (plan deviation from design's bare TenantKeyedSet): `Registry[string, storedHidden]` + `TenantKeyedSet` index, because the reconciliation sweep must enumerate tenants (`GetAll` mirrors `GetMonsters`). Election reads only the SET.
4. **Ordering (D5)**: set mutation BEFORE location resolution/relinquish, always. Location failure → debug log + skip, set stays mutated (FR-7.2). Reveal handler's own sweep closes the SREM-vs-concurrent-election race.
5. **Reconciliation (D4)**: leader-gated task in monsters, 5 min, one-way (remove members with no active SuperGmHide in atlas-buffs; keep on fetch error). Lost-APPLIED accepted limitation. `DEL` of the Redis keys = fail-open to pre-task behavior.
6. **All NPCs get election** (D6) — no movable/static filtering.
7. **DPS-leader guard added** (not in design §4's list): `Damage`'s direct `StartControl` to the damage leader must skip hidden characters or the "never selected" acceptance criterion fails.
8. **Remove-controller packet**: same opcode/writer name as the grant; flag byte 0 + uint32 npcId, nothing else. IDA: v95 `0x679730`, v83 `0x6d9a83` (byte-identical, `Decode1` + `Decode4` → `SetRemoteNpc`). New file `remove_controller.go`; matrix untouched (op `SPAWN_NPC_REQUEST_CONTROLLER` already verified on v83/v84/v87/v95/jms). `coverage-manifest.yaml` declares it for the completeness critic.
9. **Fail-open everywhere**: hidden-set read fails → election unfiltered; buffs winner-check fails → treat visible; `IsController` Redis failure/pre-init → allow motion; claim/release failure → skip, next trigger converges. No retry loops.
10. **Latent pool-leak fix folded in**: `getControllerCandidate`'s `controlCounts[m.ControlCharacterId()] += 1` inserts non-pool controllers into the candidate map (Go insert-on-increment); must only increment seeded ids or a mid-relinquish hidden GM re-enters the pool.

## Plan-time verified facts (design §9)

1. Maps location route/model — verified (see Key files).
2. atlas-buffs **removes buff state before emitting EXPIRED** (`character/processor.go:71` Cancel → registry cancel then emit; `GetExpired` prunes then returns) → reveal winner-check is safe, no hedge needed.
3. Channel consumer groups are **per-pod** (`"Channel Service - %s"` + `SERVICE_ID`, `main.go:156-163`) → every pod sees buff events, session-presence routes work. Monsters uses its shared group (`main.go:30`) → exactly-one-pod consumption for the Redis mutation.
4. Remove-arm IDB read order — verified (decision 8).
5. atlas-buffs has **no logout consumer** (only buff-command handlers) → hidden entries persist over logout; harmless + reconciliation-covered.

## Dependencies & environment

- `EVENT_TOPIC_CHARACTER_BUFF_STATUS`, `REDIS_URL`, `REDIS_PASSWORD` all ship in the shared `atlas-env` configmap (`deploy/k8s/base/env-configmap.yaml`); REST roots fall back to `BASE_SERVICE_URL` (`libs/atlas-rest/requests/url.go:14`). **No deploy/k8s changes.**
- `services/atlas-channel/atlas.com/channel/go.mod` gains `atlas-redis v0.0.0` (replace line already at :90); tidy pulls `go-redis v9.21.0` + `miniredis v2.38.0` (versions match monsters). **`docker buildx bake atlas-channel` is mandatory.** Monsters go.mod expected unchanged — verify with git diff.
- Channel has NO Redis usage today; `atlas.Connect(l)` + `controller.InitRegistry(rc)` added in `main.go`.
- Rollout note: controller entries build as sessions (re)enter maps; a channel redeploy drops all sessions, so claims repopulate on reconnect — no migration step.

## Verification gates

Per module: `go test -race ./...`, `go vet ./...`, `go build ./...`. Repo root: `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`, `tools/lint.sh --check`, `docker buildx bake atlas-channel`, `go run ./tools/packet-audit matrix --check`. Pre-PR: `superpowers:requesting-code-review` + `packet-completeness-critic` (libs/atlas-packet changed). The ≥2-replica live acceptance walk (PRD §10) happens at deploy/review time.

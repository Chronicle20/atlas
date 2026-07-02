# task-122 — Attack-Path Snapshots (PS-1): Context

Companion to `plan.md`. Key files, decisions, and dependencies for executors and reviewers. Line numbers are as of plan time (2026-07-02) and will drift — locate by function name.

## Dependency: task-120 (HARD GATE)

task-120 (monster-move-local-state) was **planned but not implemented or merged** when this plan was written. Plan Task 1 verifies it landed on main, rebases, and reconciles every consumed API name (`monster.GetLiveMirror`, `LiveEntry`, `LiveEntryFromModel`, `RecordMirrorFallback`, prometheus dep + `/api/metrics` mount). **If task-120 is not on main, execution is BLOCKED — stop and report.** Task 10 extends task-120's `LiveEntry` with `X/Y int16` + `UpdatePosition`.

## Key decisions (from design.md / event-coverage.md, encoded in the plan)

- **Composed-model snapshot** (design §3 alt A): the snapshot stores the three REST-shaped components (core `character.Model`, `[]skill.Model`, `inventory.Model`) + buffs + position overlay, and composes the exact decorated model `GetById(InventoryDecorator, SkillModelDecorator)` returns today via `core → SetX/SetY overlay → SetInventory → SetSkills`. Downstream (planner, damage pipeline, writers) untouched.
- **Generation counters** close the event-vs-backfill race: EVERY component mutation (in-place or invalidation) bumps the gen; REST backfills apply only at unchanged gen; a discarded backfill still serves its own caller (that value is exactly what REST returns today). In-place event updates apply only when the component is valid; on an invalid component they just bump the gen.
- **Lazy populate** (PRD OQ3): first `Get` pays today's 3 GETs once; entries created ONLY by the read path (`View`); event handlers are update-only (consumers run from `LastOffset`).
- **Eviction inside `session.Processor.Destroy`** (`session/processor.go:330`) + tenant evictor in `main.go` (`listener.RegisterEvictor`, ~:287). Covers logout/disconnect/channel-change.
- **Position** (PRD OQ1/FR-2.5): fed synchronously from atlas-channel's own movement fold in `movement.ForCharacter` (fold hoisted out of the goroutine; identical Kafka command bytes). MAP_CHANGED with `UseTargetPosition` sets the overlay; portal warps invalidate position AND core (next read's core refetch carries fresh REST X/Y = today's source).
- **Buffs join the snapshot** (PRD OQ2a): APPLIED/EXPIRED are rich; upsert/remove by `SourceId`; reads self-filter past `ExpiresAt` (bounds the atlas-buffs-restart residual). **Effective stats stay on REST** (PRD OQ2b): no event exists (event-coverage §6); venom keeps the per-swing memoized fetch.
- **STAT_CHANGED handling**: complete snake_case `Values` → apply absolute in place; nil/incomplete/unappliable (`AVAILABLE_SP`, unknown types) → invalidate core; `PET_SN_*` → skip (not base-model fields); empty `Updates` → no-op. No separate MESO/FAME handlers: every meso/fame mutation also emits STAT_CHANGED (atlas-character `processor.go:751,768,796,813`).
- **One producer change** (FR-2.4a): atlas-character populates `Values` on ALL ~21 `statChangedProvider` sites (site table in plan Task 13). Without it every MP-consuming swing self-invalidates (attack → CHANGE_MP → STAT_CHANGED(Mp,nil) → invalidate). Mixed-version rollout is safe: nil Values degrades to invalidate-and-refetch.
- **Monster dedup** (FR-4.2): `buildMonsterResolver` — per-swing memoized map over `LiveMirror.Lookup` with REST fallback + `Put` backfill; `deps.getMonster` retyped to return `monster.LiveEntry`; `mpEaterTryProc` takes the resolver (keeps `mp` for `DrainMp`). One resolve per damaged monster serves reflect + MP Eater.
- **Skill-data TTL cache** (FR-4.3): in-process re-expression of task-060 semantics (positive 5m / negative 30s / `requests.ErrNotFound`-only negative / env kill-switch / lazy expiry / no singleflight), fronting `data/skill.Processor.GetById` — covers attack `se`, MP-Eater effect, and writer mastery with zero writer changes.
- **Shadow verification** (PRD OQ5 — yes, diverging from task-121's "no"): `CHAR_SNAPSHOT_SHADOW_SAMPLE_RATE` (default 0), sampled full-hit reads async-compare vs REST through the same fetch seams; bounded by a 4-slot semaphore; position tolerance ±100px; `atlas_channel_char_snapshot_divergence_total`.

## Key files

| Area | File |
|---|---|
| Snapshot registry/processor/shadow | `services/atlas-channel/atlas.com/channel/character/snapshot/{registry,processor,shadow,metrics}.go` (new) |
| Builder enablers | `character/builder.go` (SetX/SetY), `character/skill/builder.go` (new) |
| Attack path | `socket/handler/character_attack_common.go` (`processAttack:266`, `processDamageInfoEntry`, `mpEaterTryProc`), `character_attack_projectile.go` (`Plan:64`, buff gate `:97`) |
| Writers (consume same model, no change) | `socket/writer/character_attack_common.go` (`preComputeAttackValues:19`, mastery `:214` → cached via data/skill) |
| Consumers gaining snapshot handlers | `kafka/consumer/{character,skill,asset,compartment,buff}/consumer.go` |
| Skill DELETED msg defs (new) | `kafka/message/skill/kafka.go` |
| Skill-data cache | `data/skill/{cache,metrics}.go` (new), `data/skill/processor.go:30` |
| Mirror extension | `monster/live_mirror.go` (task-120's, + X/Y), `movement/processor.go` (`ForCharacter:46` position feed, `ForMonster` mirror feed) |
| Lifecycle | `session/processor.go:330` (Destroy evict), `main.go:287` (tenant evictors) |
| Producer change | `services/atlas-character/atlas.com/character/character/processor.go` (all `statChangedProvider` sites; provider at `producer.go:249`) |

## Values key convention (channel↔character contract)

snake_case per stat type: `skin face hair level job strength dexterity intelligence luck hp max_hp mp max_mp available_ap available_sp experience fame meso gachapon_experience`. JSON numbers arrive as `float64` on the consumer side. Channel treats `AVAILABLE_SP` as un-appliable (per-book string table) → invalidate; producer still populates it for contract completeness.

## Metrics / env

- `atlas_channel_char_snapshot_reads_total{tenant,component,outcome}` — component ∈ core|skills|inventory|buffs|position; outcome ∈ hit|miss|fallback_success|fallback_failure
- `atlas_channel_char_snapshot_updates_total{tenant,component,kind}` — kind ∈ event_update|invalidation|backfill|backfill_discarded
- `atlas_channel_skill_data_cache_total{tenant,outcome}` — hit|negative_hit|miss
- `atlas_channel_char_snapshot_divergence_total{tenant,component}`
- Env: `SKILL_DATA_CACHE_ENABLED=true`, `SKILL_DATA_CACHE_TTL=5m [1s,24h]`, `SKILL_DATA_CACHE_NEGATIVE_TTL=30s [0s,5m]`, `CHAR_SNAPSHOT_SHADOW_SAMPLE_RATE=0`
- Endpoint: `/api/metrics` (mounted under the `/api/` base path — task-120 brings the mount; `/api/readyz` gotcha family)

## Test harness notes

- Consumer-test pattern: `newTestTenant` + `server.Register(tm, channel.NewModel(0,1), "127.0.0.1", 8484)` (see `kafka/consumer/monster/consumer_test.go:42-54`); handlers invoked directly with `tenant.WithContext`.
- Writer encode harness: `packet.Encode = func(l, ctx) func(options map[string]interface{}) []byte`; packet-test ctx via `pt.CreateContext("GMS", 83, 1)` (`socket/writer/character_info_test.go`).
- atlas-character has a DB-backed harness: `testDatabase(t)` (`character/provider_test.go:20`) + `kafka_integration_test.go` message-capture pattern — reuse for the Task 13 Values tests.
- Builder pattern only; no `*_testhelpers.go`. Test-only registry resets live in `_test.go` files (`resetRegistryForTest`, `resetShadowForTest`, `resetSkillCache`).

## VERIFY-AT-EXECUTION items (resolved in plan Task 1)

1. task-120 landed shape (API names, metrics bootstrap) — FR-5.2.
2. `GET /characters/{id}/inventory` does not net out reservations (event-coverage §4); if it does, RESERVED/RESERVATION_CANCELLED join Task 7 as invalidations.
3. `Values` key convention at the four populated sites.
4. MAP_CHANGED `UseTargetPosition=false` → position+core invalidate disposition.
5. Escalate (not fix): `RequestReserve` first-request-only loop (atlas-inventory `compartment/processor.go:767`) — pre-existing bug candidate, owner-visible.

## Escalations for owner

- `RequestReserve` processes only the first batched reserve request (atlas-inventory `compartment/processor.go:767`) — candidate pre-existing atlas-inventory bug, NOT addressed by task-122 (reservations are registry-only and excluded from the snapshot).

## Verification gate (Task 14)

`go test -race ./... && go vet ./... && go build ./...` in `services/atlas-channel/atlas.com/channel` AND `services/atlas-character/atlas.com/character`; `docker buildx bake atlas-channel atlas-character` from the worktree root; `tools/redis-key-guard.sh`; zero-REST grep sweep (no `GetById(cp.InventoryDecorator`, no `bp.GetByCharacterId` in handlers, no `mp.GetById` in the attack file); code review via `superpowers:requesting-code-review` before PR.

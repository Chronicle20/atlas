# Corsair Battleship (task-153) — Implementation Context

Companion to `plan.md`. Captures the code research, decisions, and deviations
an implementer (or reviewer) needs without re-deriving them.

## Key files

| Area | File | Why it matters |
|---|---|---|
| Mount classification | `libs/atlas-constants/skill/mount.go` | `IsBattleshipMountSkill` added; `SkillOnlyMountVehicleId` deliberately unchanged (atlas-data ingestion contract — adding battleship there would hardcode the wire value and force WZ re-ingestion) |
| Atomic counter | `libs/atlas-redis/counter.go` (new) | `TenantCounter` with Lua `DecrByIfExists`; second Lua script in the lib (precedent: `lock.go:15`) |
| uint32 resolver | `libs/atlas-packet/resolve.go` | `ResolveValue` sibling of `ResolveCode`; returns `(0,false)` on miss — no safe sentinel for 4-byte values |
| Options access | `<ch>/socket/writer/options_registry.go` (new) | per-tenant `writerName → options` tables, fed from `tenantCfg.Socket.Writers` in `buildListener` (`main.go:404`), evicted in `listener.RegisterEvictor` (`main.go:287`) |
| Ride mirror | `<ch>/battleship/mirror.go` (new) | pod-local riding truth; pattern copied from `<ch>/monster/status_mirror.go` |
| Processor | `<ch>/battleship/processor.go` (new) | sole owner of mirror + Redis pool; `Drain` holds the crossing predicate and break flow |
| Cast path | `<ch>/skill/handler/common.go:93-105`, `<ch>/skill/handler/mount.go`, `<ch>/socket/handler/character_skill_use.go:60-70` | carve-out, mount gate, battleship arm, cast rejection |
| Drain | `<ch>/socket/handler/character_damage.go:31` | replaces `// TODO decrease battleship hp`; `ChangeHP` (parallel pool) untouched |
| Gate | `<ch>/socket/handler/character_attack_common.go:660` | after ownership check (`processAttack` declared at `:636`); all attack types funnel through `processAttack` |
| Lifecycle | `<ch>/kafka/consumer/buff/consumer.go:114`, `<ch>/session/processor.go:405` | mirror put on APPLIED, Clear on EXPIRED and session Destroy |
| Cooldown transport | `<ch>/character/skill/processor.go:53` `ApplyCooldown` | already emits `SET_COOLDOWN`; atlas-skills applies + emits `COOLDOWN_APPLIED`; `<ch>/kafka/consumer/skill/consumer.go:111` already announces the client packet. Zero new Kafka surface. |
| Templates | `services/atlas-configurations/seed-data/templates/template_*_1.json` | options tables ×9 (gms_61…jms_185; gms_12/gms_48 n-a); v92 gets five new writer entries; gms_87/92/95/jms_185 each gain `CharacterUseSkillHandle` + `CharacterDamageHandle` — see plan.md R-2 |

`<ch>/` = `services/atlas-channel/atlas.com/channel/`.

## Verified facts the plan builds on

- `skill.CorsairBattleshipId/CannonId/TorpedoId` = 5221006/07/08 already exist
  (`libs/atlas-constants/skill/constants.go:3236-3238` — re-pinned after the `main` merge).
- `character/skill.Model` already carries `CooldownExpiresAt()` and `OnCooldown()`
  (`model.go:40-46`), decorated live via `SkillModelDecorator` — the cast
  rejection costs zero extra round-trips.
- `DamageTakenInfo.Damage()` is `int32` (`libs/atlas-packet/model/damage_taken_info.go:63`).
- `effect.Model.Cooldown() uint32`, `.Duration() int32` (ms).
- `character.Model.Level()` is `byte`.
- Buff events: `AppliedStatusEventBody{SourceId int32, Level byte, Changes []StatChange}`;
  `StatChange.Type` matches `charconst.TemporaryStatTypeMonsterRiding` = `"MONSTER_RIDING"`.
- `tamedMountStatups(e, vehicleId)` (mount.go:61) already does exactly the
  MONSTER_RIDING amount override the battleship arm needs — reused, not duplicated.
- Mounts are applied with `MountBuffDuration = MaxInt32` and cancelled at next
  login by the session consumer (`kafka/consumer/session/consumer.go:290-303`,
  "mounts are transient") — so a pod-local mirror can never miss a legitimate ride.
- atlas-redis tests use miniredis via `setupTestRedis(t)`
  (`libs/atlas-redis/registry_test.go:16`); miniredis executes Lua, so
  `DecrByIfExists` is testable in-process.
- `REDIS_URL`/`REDIS_PASSWORD` are read by `redis.Connect`
  (`libs/atlas-redis/connection.go`) from env; `REDIS_URL` lives in the shared
  `atlas-env` ConfigMap (`deploy/k8s/base/env-configmap.yaml:151`) which
  atlas-channel already mounts via `envFrom`.
- `go.work` already lists `./libs/atlas-redis`; atlas-channel `go.mod` already
  has the `replace` (line 82) but not the `require`.
- The shared Dockerfile already COPYs `libs/atlas-redis` (atlas-mounts builds
  against it) — bake is still mandatory verification.

## Decisions (including deviations from design.md)

1. **v92 template audit result (design predicted v87 risk; reality is v92).**
   All six templates were audited 2026-07-10: v83/v84/v87/v95/jms185 route both
   `CharacterBuffGive` and `CharacterSkillCooldown` correctly; `template_gms_92_1.json`
   routes NONE of the five buff/cooldown writers. Fixed in-scope with opcodes
   from `docs/packets/MapleStory Ops - ClientBound.csv` (the only v92 source —
   no v92 IDB or registry exists), cross-validated: the same CSV rows' values
   for v83/v87/v95/jms185 match those versions' verified templates exactly.
   **Out of scope:** v92's template is a skeleton overall (47 writers, 37
   handlers, no `CharacterUseSkillHandle`/attack/damage handlers at all) — a
   pre-existing tenant-wide gap. Battleship, like every other skill, cannot be
   exercised on v92 until that template is completed; the five entries added
   here make the battleship config complete on the writer side.

2. **No `WithResolvedValue` wrapper (design §3.3 suggested one).** Both wire
   values resolve through one mechanism: the tenant writer-options registry +
   `ResolveValue`. The gauge pseudo-id is resolved in the damage handler
   *before* deciding to announce, so a config miss sends **no packet at all**
   (vs. an encoder-wrapper which must emit something on miss). This matches the
   design's own fail-loud choice for the vehicle id ("abort the mount").

3. **Character level via `mountDeps.characterLevel` (one extra character GET
   per mount cast)** instead of threading the already-loaded model through the
   `UseSkill` signature (design §4.4 note). Avoids rippling a signature change
   through every UseSkill caller/test; mount casts are rare, the cast path is
   not hot. The seam keeps it offline-testable.

4. **State TTL rides in the mirror (`RideState.StateTTL`).** Drain must refresh
   the Redis TTL per touch (design §4.3) but must not call REST on the hot path
   (`data/skill.GetEffect` is an uncached REST call — verified). The buff
   consumer derives the TTL once per mount from effect data
   (`battleshipStateTTLFunc`) and stores it in the mirror; the processor falls
   back to `fallbackStateTTL = 35min` (documented WZ-derived constant) when the
   lookup fails or yields 0. Only lazy re-init (rare) and break (rare) do REST.

5. **Session-destroy cleanup is a direct `battleship.Clear` call in
   `session.Processor.Destroy`.** Import-cycle verified safe: `battleship`
   imports `character`, `character/buff`, `character/skill`, `data/skill` —
   none import `session`. The announce stays in the damage handler so
   `battleship` never imports `session`/`socket/writer`.

6. **Gauge announce lives in the damage handler, not the processor.** `Drain`
   returns a `DrainResult`; the handler (which already holds `s` and `wp`)
   announces on `DrainDrained`. Keeps the battleship package free of
   session/writer dependencies and keeps break side effects (which have no
   session dependency) inside the processor.

7. **No deployment manifest change** (design §4.3 assumed one). `REDIS_URL`
   already reaches atlas-channel through the `atlas-env` ConfigMap `envFrom` —
   same mechanism atlas-mounts uses.

8. **Redis error = degraded drain, never failed damage processing.** `Drain`
   logs warn and returns `DrainSkipped`; character HP flow unaffected;
   state self-repairs via lazy re-init after recovery (design §4.5).

9. **Exactly-once break** relies on the Lua-serialized decrement: the predicate
   `newHp <= 0 && newHp+damage > 0` is true for exactly one decrement per
   depletion. All break side effects are idempotent (Clear no-ops, cancelling
   an absent buff emits nothing, duplicate SET_COOLDOWN re-applies the same
   expiry), so even a theoretical lazy-re-init-window double-break is a no-op.

## Dependency order

Tasks 1–5 are independent of each other. 1→8; 2→6; 3→8,9; 4→8,9; 5→6,7,10;
6→7,8,9. Task 11 (templates) is independent of all code tasks. Task 12 last.

## Wire values (config-only — never in Go code)

| Value | Meaning | Config home | Verification |
|---|---|---|---|
| 5221999 | gauge pseudo-skill id in the skill-cooldown packet | `CharacterSkillCooldown` writer options, `skills.BATTLESHIP_HP_GAUGE` | IDA-verified in all 5 available IDBs (design §1.1); v92 unverified (no IDB), bracketed by v87/v95 |
| 1932000 | battleship vehicle item id (MONSTER_RIDING amount) | `CharacterBuffGive` writer options, `vehicles.CORSAIR_BATTLESHIP` | design §1.1 (Cosmic ItemId + client max-HP fn) |

## Post-merge ops

`backfill.md` (Task 12): PATCH every live tenant's channel socket config with
the two options tables (v92 additionally may need the five writers), then
restart atlas-channel — socket config is read at listener build and does not
hot-reload. Full six-tenant sweep, no spot-checking.

## Verification commands

```bash
(cd libs/atlas-constants && go test -race ./... && go vet ./...)
(cd libs/atlas-redis && go test -race ./... && go vet ./...)
(cd libs/atlas-packet && go test -race ./... && go vet ./...)
(cd services/atlas-channel/atlas.com/channel && go test -race ./... && go vet ./... && go build ./...)
(cd services/atlas-configurations/atlas.com/configurations && go test -race ./... && go vet ./...)
tools/redis-key-guard.sh          # repo root, NO GOWORK=off prefix
docker buildx bake atlas-channel  # worktree root; mandatory
```


## Post-merge note (2026-07-28)

`main` was merged into this branch after context.md was written. Line anchors above were
re-pinned; the authoritative, fully re-verified list is the **Post-merge reconciliation**
section (R-1…R-12) at the top of `plan.md`. Read that before starting any task.

# task-218 — Implementation Context

Companion to [`plan.md`](./plan.md). Everything below was read from the
worktree at plan time (file:line references are against
`task-218-player-cast-mists` at `efb865cfa`) or verified live during the
design phase (see [`design.md`](./design.md) §0).

---

## 1. What this task delivers

Four player-cast mist skills on top of task-200's mist mechanism:

| Skill | Identity (`libs/atlas-constants/skill/identities_gen.go`) | Wire id | Registry | Mist |
|---|---|---|---|---|
| Shadower Smokescreen | `ShadowerSmokescreen` (:292) | 4221006 | `Register` (USE_SKILL) | CHARACTER / PROTECTION |
| Blaze Wizard Flame Gear | `BlazeWizardStage3FlameGear` (:449) | 12111005 | `RegisterAttackCast` | MONSTER / DAMAGE_OVER_TIME |
| Night Walker Poison Bomb | `NightWalkerStage3PoisonBomb` (:490) | 14111006 | `RegisterAttackCast` | MONSTER / DAMAGE_OVER_TIME |
| Evan Recovery Aura | `EvanStage8RecoveryAura` (:590) | 22161003 | `Register` (USE_SKILL) | CHARACTER / RECOVERY |

Plus the FR-0 prerequisite (Evan wire ids are absent from every per-version
binding table, so `22161003` resolves to no Identity and Recovery Aura cannot
dispatch at all).

## 2. Key files

### atlas-maps (owns the mist contract and the tick)

| File | Role |
|---|---|
| `services/atlas-maps/atlas.com/maps/kafka/message/mist/kafka.go` | Contract owner. `TargetKind*`/`EffectKind*` consts :24-36, `CreateCommandBody` :46-70, `CreatedBody` :90-108. |
| `services/atlas-maps/atlas.com/maps/mist/model.go` | `Mist` :19-45, getters, `AffectedAreaType*` consts :99-139, `AffectedAreaTypeFor` :149-154, `Rect`/`Contains` :265-274, `ShouldTick` :283-288, `Builder` :297-442. |
| `services/atlas-maps/atlas.com/maps/mist/processor.go` | `Create` :63-114 — normalizes empty kinds, derives `nType` via `AffectedAreaTypeFor(body.OwnerType)` :96, rolls back the registry insert on emit failure. |
| `services/atlas-maps/atlas.com/maps/mist/producer.go` | `createdEventProvider` / `destroyedEventProvider`. |
| `services/atlas-maps/atlas.com/maps/mist/registry.go` | Tenant-keyed singleton, `sync.RWMutex`. |
| `services/atlas-maps/atlas.com/maps/tasks/mist_tick.go` | `monsterDotTickIntervalMs = 1000` :85, `PositionLookup` :107, `buffCommand`/`applyDiseaseBody` :111-133, `monsterCommand`/`applyStatusBody` :168-199, `tickOneMist` :348-371, `tickCharacters` :377-401, `tickMonsters` :410-439. |
| `services/atlas-maps/atlas.com/maps/character/rest.go` | Minimal projection of the atlas-character resource: `Id`, `X`, `Y` only. |
| `services/atlas-maps/atlas.com/maps/character/processor.go` | `Position(characterId) (int16, int16, error)`. |
| `services/atlas-maps/atlas.com/maps/main.go:114-127` | `posLookup` closure + `tasks.NewMistTick(l, 1000, posLookup)`. |

### atlas-channel (casts, mirrors the contract, consumes the events)

| File | Role |
|---|---|
| `services/atlas-channel/atlas.com/channel/kafka/message/mist/kafka.go` | **Partial** mirror of the atlas-maps contract — missing `CommandTypeCancel`, `CancelCommandBody`, `Reason*`, and ordered differently. Task 7 brings it to a full mirror. |
| `services/atlas-channel/atlas.com/channel/kafka/consumer/mist/consumer.go` | `SetStartOffset(kafka.LastOffset)` :29, `mistPhase = 0` :87, `handleMistCreated` :89-113, `handleMistDestroyed` :115-127, broadcaster seams :62-76. |
| `services/atlas-channel/atlas.com/channel/mist/processor.go` | `Create(body)` → `COMMAND_TOPIC_MIST`. |
| `services/atlas-channel/atlas.com/channel/skill/handler/registry.go` | Two registries. `Register`/`Lookup` (USE_SKILL, `Handler` takes `packetmodel.SkillUsageInfo`) :18-53; `RegisterAttackCast`/`LookupAttackCast` (`AttackCastHandler` takes `skillId`+`skillLevel`) :78-109. The doc comment at :55-77 explains why they are not interchangeable. |
| `services/atlas-channel/atlas.com/channel/skill/handler/poisonmist/poisonmist.go` | The template for all four handlers. `PlayerMistTickIntervalMs = 3000` :59, `MaxPlayerMistDurationMs = 300_000` :70, `loadCaster`/`emitCreate` seams :74-86, validation block :122-137. |
| `services/atlas-channel/atlas.com/channel/skill/handler/mprecovery/mprecovery.go` | The USE_SKILL handler shape (`channelhandler.Register`, `info packetmodel.SkillUsageInfo`). |
| `services/atlas-channel/atlas.com/channel/skill/handler/registrations/registrations.go` | Blank-import list. A handler package absent here never runs `init()`. |
| `services/atlas-channel/atlas.com/channel/socket/handler/character_damage.go` | `damageMitigationDeps` :34-44, wiring :83-97, `extractBuffAmounts` :132-162, `processDamageTaken` :168-325. |
| `services/atlas-channel/atlas.com/channel/party/processor.go` | `GetByMemberId(memberId) (Model, error)`, `MemberInMap` filter :95-99. `party.MemberModel` has `Id()`, `Online()`, `Field()`. |
| `services/atlas-channel/atlas.com/channel/character/processor.go:292` | `ChangeMP(f, characterId, amount int16)` → `COMMAND_TOPIC_CHARACTER` `CHANGE_MP`. |
| `services/atlas-channel/atlas.com/channel/kafka/message/character/kafka.go` | `Command[E]{WorldId, CharacterId, Type, Body}` :30-35, `CommandChangeMP = "CHANGE_MP"` :19, `ChangeMPCommandBody{ChannelId, Amount int16}` :70-73. This is the shape atlas-maps must mirror. |

### libs/atlas-constants (FR-0)

| File | Role |
|---|---|
| `libs/atlas-constants/gen/main.go` | `go run .` regenerates; `go run . -check` drift-checks. Emits `identities_gen.go` ×2, 22 `version_*_gen.go`, `constants/registry_gen.go`, `skill+job/baseline_gen.go`. |
| `libs/atlas-constants/gen/semantics.go:369-…` | `BuildSemantics` — joins the pinned snapshot with `semantics/<v>.yaml`: auto-binds every snapshot wire id whose value equals a canonical identity token, then overlays the divergent list. |
| `libs/atlas-constants/gen/wzsnapshot/snapshots.go` | `LoadSnapshot` (verifies the pinned sha256 on every load) and `HashIds` :77-97 — the canonical hash: sorted ids, `skills:N\n<id>\n…jobs:N\n<id>\n…`. |
| `libs/atlas-constants/gen/wzsnapshot/PROVENANCE.md` | Records the 2026-07-30 jobs-union drain, per-version tenant ids, and the gms_12 mirror-from-gms_48 policy. |

## 3. Decisions already made (design phase) that the plan assumes

- **Flame Gear applies `POISON` with magnitude 0**, same as Poison Mist and
  Poison Bomb. `monsterStatus` is `{}` for 12111005 on every version, and
  `atlas-monsters` has exactly two DoT statuses (`StatusPoison`,
  `StatusVenom` — `monster/status.go:20,23`). No `atlas-monsters` change.
- **Recovery Aura restores MP only**, magnitude `x` (38 at L1 → 80 at L15),
  already exposed as `effect.Model.X()`. `hp`/`mp`/`hpR`/`mpR` are 0 at every
  level on every version. No `atlas-data` change.
- **Smokescreen is party-scoped** — `CAffectedAreaPool::IsSmokeAreaByPoint`
  (v95 @0x434f40) accepts an area only if `dwOwnerID` is the local character
  or one of `adwPartyMemberID`, evaluated at hit time.
- **Smokescreen protection is mechanism (c)**: a channel-local protection-mist
  registry consulted on the damage path — not buff-mediated (atlas-maps has no
  party client, and a buff would shield everyone inside), not a synchronous
  REST call to atlas-maps on the hot damage path.
- **Recovery Aura party scoping is a cast-time snapshot** carried on the
  CREATE command, because atlas-maps has no party client and nothing
  client-side evaluates aura membership.

## 4. Facts verified at plan time (close design open items)

1. **`atlas-character`'s `ChangeMP` clamps** —
   `services/atlas-character/atlas.com/character/character/processor.go:1420`:
   `adjusted := enforceBounds(amount, c.Mp(), maxMP, 0)` against the effective
   MaxMP. Design open item 2 is **settled: the clamp exists**; FR-5.3's
   max-MP half needs no new code.
2. **`ChangeMP` does *not* skip dead characters** — it clamps to `[0, maxMP]`
   with no HP check. FR-5.3's "MUST NOT affect a dead character" is therefore
   **not** satisfied downstream; the plan closes it in `tickRecovery` by
   extending atlas-maps' character lookup to carry `hp` (Task 5) and skipping
   `hp == 0` (Task 6). This is a correction to design §6.3, which assumed the
   owning service covered it.
3. **`AffectedAreaCreated` / `AffectedAreaRemoved` are registered in all 11
   seed templates** — `grep -rlc` over
   `services/atlas-configurations/seed-data/templates/` returns 11 for each.
   No template change; Task 16 re-runs this as a gate.
4. **All four Identity constants exist** at the wire ids the PRD names
   (`identities_gen.go:292,449,490,590`).

## 5. Traps this task walks past

- **`nType` must never be 0 for a character-owned mist.** `nDamage` is written
  only on the mob-skill arms (`nSkillID == 130/131`), so a character-owned
  mist sent as 0 bills the caster an uninitialised value — the live
  1,434,803-damage self-hit task-200 diagnosed (`mist/model.go:125-133`).
- **`P > T` strictly.** The mist re-apply period (3000 ms) must exceed the DoT
  tick interval atlas-maps emits (`monsterDotTickIntervalMs`, 1000 ms), or the
  eligible damage window is exactly zero and the mist deals no damage at any
  tuning (`mist_tick.go:57-85`, `poisonmist.go:33-59`).
- **Reject, never clamp, an implausible lifetime.** The client computes its own
  `tEnd = tStart + 1000 * SKILLLEVELDATA::tTime`, so a server clamp leaves the
  client rendering a mist the server stopped ticking (`poisonmist.go:61-70`).
- **`COMMAND_TOPIC_MONSTER` is a shared topic** — every registered handler in
  atlas-monsters unmarshals every message on it. `applyStatusBody`'s key set is
  frozen: add nothing, rename nothing, retype nothing (`mist_tick.go:178-199`).
- **The mist contract lives in two Go modules.** A json tag changed in one and
  not the other compiles clean and decodes into a zero-valued body at runtime.
  Task 7 adds `tools/mist-contract-mirror-guard.sh` for exactly this.
- **A handler package missing from `registrations.go` never runs `init()`** and
  is silently absent.
- **Registering an attack-delivered skill on the USE_SKILL registry** both
  never fires *and* silently zeroes its MP cost (`registry.go:55-77`).
- **`buff-duration-guard.sh`**: the recovery path deliberately does not touch
  `COMMAND_TOPIC_CHARACTER_BUFF`, so it stays trivially clean. Do not
  "simplify" recovery into a buff apply.
- **`skill-job-id-guard.sh`** derives its ban list from the task-187 audit and
  depends on the binding tables FR-0 regenerates — run it after Task 2.

## 6. Live-environment dependencies (Task 2 only)

The FR-0 re-drain needs cluster access to `atlas-data` in namespace
`atlas-main`. Tenant ids **change on reprovision** — re-list them with
`GET /api/tenants` rather than reusing design §0's table verbatim. If the
cluster is unavailable, Task 2 is blocked; every other task is not, and Tasks
3–16 can proceed (only Recovery Aura's dispatch depends on Task 2 landing).

## 7. Verification gates (CLAUDE.md)

Changed modules: `libs/atlas-constants`, `libs/atlas-constants/gen`,
`services/atlas-maps`, `services/atlas-channel`. No `go.mod` changed unless a
new dependency is added, so the `docker buildx bake` step covers
`atlas-maps` and `atlas-channel` defensively.

```
go test -race ./...      # in libs/atlas-constants, libs/atlas-constants/gen, atlas-maps, atlas-channel
go vet ./...             # same four
go build ./...           # atlas-maps, atlas-channel
docker buildx bake atlas-maps atlas-channel
tools/redis-key-guard.sh
tools/goroutine-guard.sh
tools/buff-duration-guard.sh
tools/skill-job-id-guard.sh
tools/trade-contract-mirror-guard.sh
tools/mist-contract-mirror-guard.sh      # new, Task 7
tools/lint.sh --check
(cd libs/atlas-constants/gen && go run . -check)
```

`tools/lint.sh --check` needs nvm-provided node on PATH or it false-fails.

## 8. Open items the plan does NOT close

1. **Is Recovery Aura's `x` absolute MP or a percentage of max MP?** Not
   determinable from WZ; treated as absolute, consistent with every other `x`
   consumer in the repo. Settle by casting at a known level and reading the MP
   delta. If it is a percentage, the change is confined to `tickRecovery`'s
   amount and the `RecoveryMp` doc comment.
2. **Registry confirmation for the two USE_SKILL handlers.** The WZ-shape
   argument is an argument from total absence for Smokescreen and Recovery
   Aura (uniform across ten independently-ingested tenants). One live cast each
   confirms it; a mis-registration fails loudly (the handler never fires).
3. **Live v95 `dotInterval`/`dotTime` are stale pre-task-200 seconds.** Nothing
   in this task reads those fields. Fixing needs a re-ingest plus an
   atlas-data REST pod restart, and belongs to whichever task next depends on
   them.

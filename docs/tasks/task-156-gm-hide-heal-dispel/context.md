# Context — task-156 SuperGM Hide + Heal & Dispel

Companion to `plan.md`. Key files, verified facts, decisions, and dependencies gathered during planning. All paths under `services/atlas-channel/atlas.com/channel/` unless noted; module is `atlas-channel`.

## Goal in one line

Implement two SuperGM active skills in `atlas-channel`: **Heal + Dispel** (`9101000`) restores HP/MP and purges disease debuffs for every player in the caster's map; **Hide** (`9101004`) is a persistent-buff toggle that makes the caster invisible/untargetable and survives map changes.

## Scope (verified during planning)

- **Only `atlas-channel`'s `go.mod` is touched.** `atlas-data`, `atlas-buffs`, and `libs/atlas-constants` require **no** changes — all ids, types, and command handling already exist. This overturns the PRD's Service Impact section (which anticipated `atlas-data` edits); see design.md §1 "Grounding that reshapes the PRD."

## Key files (read these first)

**Pattern references (do not modify):**
- `skill/handler/mount.go` — the canonical toggle-via-persistent-buff handler with a `deps` struct for offline tests. Hide is modeled on it. `MountBuffDuration = int32(math.MaxInt32)` is the "permanent buff" convention.
- `skill/handler/heal/heal.go` — the Cleric Heal handler: recipient hydration from effective stats, `ChangeHP` clamp idiom (`effectiveMaxHpOrBase`), and the self/foreign skill-use announce (`socketHandler.AnnounceSkillUse` / `AnnounceForeignSkillUse` via `session.IfPresentByCharacterId` and `channelmap.ForOtherSessionsInMap`).
- `skill/handler/common.go` — `UseSkill` orchestrator: consumes MP/cooldown, runs the generic buff-apply path **only when `e.Duration() > 0 && len(e.StatUps()) > 0`** (line 107), then dispatches the per-skill handler via `Lookup` (line 117). Both new skills carry **no** statups in the WZ reader, so the generic path is skipped for them — a plain registered handler suffices (no `common.go` short-circuit like mounts need).
- `services/atlas-consumables/atlas.com/consumables/character/buff/producer.go:57` + `.../kafka/message/character/buff/kafka.go:44` — the working `CancelByTypes` producer template Task 3 mirrors.

**Files modified/created:** see plan.md "File Structure".

## Verified facts (grep/read evidence)

- **Constants exist** (`libs/atlas-constants`): `skill.SuperGmHealDispelId = Id(9101000)`, `skill.SuperGmHideId = Id(9101004)`, `skill.RogueDarkSightId = Id(4001003)`, `job.SuperGmId = Id(910)`, `character.TemporaryStatTypeDarkSight = "DARK_SIGHT"`, plus all 11 disease constants (`character/temporary_stat.go`).
- **SuperGM gate is correct:** `job.Is(900, 910)` (plain GM vs SuperGM) is **false** (`job/model.go:44` branch math); `job.Is(910, 910)` is true. Plain GM cannot pass. `character.Model.JobId()` returns `job.Id` already — no cast.
- **Effect recovery fields already parsed:** `data/skill/effect/rest.go:92-95` deserializes `Hp`/`Mp`/`HPR`/`MPR` into private `Model` fields. `HP()` accessor exists (`model.go:111`); `MP()/HpR()/MpR()` are the only gap (Task 1).
- **`ChangeHP`/`ChangeMP` exist** channel-side and take `int16`: `character/processor.go:271-277`, `character/producer.go:57-83`, command types `CHANGE_HP`/`CHANGE_MP` in `kafka/message/character/kafka.go:15-16`. Reuse both; no new HP/MP command.
- **`CANCEL_BY_TYPES` is consumed by atlas-buffs:** `services/atlas-buffs/.../kafka/consumer/character/consumer.go:81` `handleCancelByTypes` → `CancelByStatTypes`, registered at `:39`. The channel side has only `APPLY`/`CANCEL` today (`character/buff/processor.go:19-21`) — Task 3 adds the producer + processor method; **no atlas-buffs change**.
- **Disease set** (atlas-buffs authority): `services/atlas-buffs/.../buffs/character/immunity.go:7` — `STUN, POISON, SEAL, DARKNESS, WEAKEN, CURSE, SEDUCE, CONFUSE, UNDEAD, SLOW, STOP_PORTION` (11).
- **Buff `Model` accessors:** `SourceId() int32`, `Expired() bool`, `NewBuff(sourceId int32, level byte, duration int32, changes []stat.Model, createdAt, expiresAt time.Time)` (`character/buff/model.go`). `buff.Processor.Apply(f, fromId, sourceId, level, duration, statups) model.Operator[uint32]` then `(...)(characterId)`; `Cancel(f, characterId, sourceId)`.
- **Spawn choke point:** `kafka/consumer/map/consumer.go:427` `spawnCharacterForSession` is the single function every character spawn passes through — called at `:174` (SpawnForSelf-of-others) and `:370` (`enterMap`→others). It already fetches `buff.NewProcessor(...).GetByCharacterId(c.Id())` at `:432`. Both callers skip `k == s.CharacterId()`, so `c` is never the viewer's own character. `despawnForSession` at `:464`.
- **No import cycle:** `kafka/consumer/map` does **not** import `skill/handler` (grepped). Nothing imports the new `skill/handler/hide` except `registrations`. So `skill/handler/hide → kafka/consumer/map` is safe.
- **Selector seams:** `recipients.go` already has `inMapCharacterIdsFunc` (backed by `_map.ForSessionsInMap`) and `loadPartyMemberFunc` (`character.GetById`) as package-level test seams; the new `SelectAllCharactersInMap` reuses them. Test fixtures use `character.NewModelBuilder().Set*().MustBuild()` (setters confirmed: `SetId/SetLevel/SetJobId/SetHp/SetMaxHp/SetMp/SetMaxMp`; there is **no** `SetX`/`SetY` builder setter — the model has `X()/Y()` accessors that default to 0).
- **`effective_stats.GetByCharacterId(worldId, channelId, characterId) (RestModel, error)`**; `RestModel.MaxHp`/`MaxMp` are `uint32` (`effective_stats/rest.go:12-13`).

## Decisions (resolved open questions)

- **OQ-1 hide stat = `DARK_SIGHT`** (not `SNEAK`). `DARK_SIGHT` has a proven server→client apply/cancel pipeline (Rogue Dark Sight); `SNEAK` has no producer. Both foreign-encode as no-ops. Design §2.
- **OQ-2 hide state = a `DARK_SIGHT` buff sourced from `SuperGmHideId`, duration `math.MaxInt32`** (Option A). Persists per-character across maps; the suppression gate reads the buff list already fetched at the spawn choke point. Keying on `SourceId` (not the stat) is essential so Rogue Dark Sight stays visible → `buff.IsGmHidden` (Task 2).
- **OQ-3/FR-17 (hidden-cast broadcast) — RESOLVED IN PLAN.** The design's §3.4 code (`foreign only if !hidden`) contradicts its own prose ("suppressed both when hiding and revealing"). Per FR-17's intent ("MUST NOT reveal the caster's position"), the **Hide** handler broadcasts **self** animation only and **never** the foreign animation (both toggle directions). **Heal + Dispel** keeps the general rule: foreign only when the caster is currently visible (`!hidden`). This is the one place the plan makes a call the design left inconsistent — flagged for the reviewer.
- **OQ-4 self-Heal while hidden:** HP/MP/dispel are keyed by recipient id; none reference the caster's avatar, so a hidden caster stays hidden. No special handling.
- **OQ-5 `ChangeMP`:** exists — reuse.

## Architecture notes for the executor

- **Two `deps`-struct handlers** (`healdispel`, `hide`), each with an exported `Apply` (production wiring) and a tested core (`applyHealDispel` / `applyHide`). This mirrors `mount.go` and is the project's idiom for offline-testable handlers.
- **Shared predicate `buff.IsGmHidden`** lives in `character/buff` because three packages need it (heal FR-17, map gate, hide toggle) — DRY over per-package copies.
- **Broadcast helpers** (`SpawnCharacterInMap`/`DespawnCharacterInMap`) live in the map consumer package so spawn-body construction (buffs + guild + `enteringField=false`) stays in one place and the reveal packet is byte-identical to a normal map-entry spawn.
- **HP/MP clamp:** `restore = flat + floor(effMax * ratio)`, then clamp to `[0, effMax-current]` and to the `int16` ceiling before `ChangeHP`/`ChangeMP`. Effective max comes from `effective_stats`, falling back to base max (`effectiveMaxOrBase`), matching Cleric Heal.

## Dependencies & ordering

Task DAG: 1 (accessors) and 2 (predicate) are independent foundations. 3 (CancelByTypes) is independent. 4 (selector) is independent. 5 (heal handler) depends on 1+2+3+4. 6 (map gate + helpers) depends on 2. 7 (hide handler) depends on 2+6. 8 verifies everything. Follow the plan order.

## Verification gates (CLAUDE.md — mandatory)

From `services/atlas-channel/atlas.com/channel`: `go test -race ./...`, `go vet ./...`, `go build ./...`.
From worktree root: `docker buildx bake atlas-channel` (go.mod touched — required), `tools/redis-key-guard.sh`.
Execute-time: confirm live `9101000` WZ recovery values against WZ data; byte-verify spawn/despawn + `DARK_SIGHT` self buff-give packets (the design flags these as unverified in-repo — hard gates, not optional).

## Risks

- **`9101000` recovery magnitude unverified in-repo** (WZ mounted at runtime). The flat+ratio formula tolerates either shape (a zero field contributes nothing), but the actual values must be confirmed at execute time.
- **`DARK_SIGHT` self-give byte encoding** must serialize non-zero so `CUser::IsDarkSight` reads it — byte-verify at execute.
- **Mocks implementing `buff.Processor`** must gain `CancelByTypes` or the build breaks — grep `buff.Processor` and patch any mock (Task 3 Step 6).
- **Test-fixture constructors** (`SkillUsageInfo`, `character.Model`) — match the sibling `heal` tests if the bare literals don't compile (noted inline in plan Tasks 5/7).

# Task 057 — Context

> Companion to `plan.md`. Captures the files, contracts, and decisions an
> implementer needs without having to re-read PRD + design end-to-end.

## 1. Goal in One Paragraph

Eliminate the dual-apply path for Priest Doom (and any future skill in
`isMobAffectingBuff`) by making `applyToMobs` the single emitter of monster
status applies for buff-classified skills. Move the four server-authority
guarantees that today live only in the per-skill Doom handler — rect
verification, mobCount cap, prop roll, kind-aware reflect skip — into the
generic path, surface client/server divergence as structured warn logs, and
delete the per-skill Doom subpackage and its blank import.

## 2. Files Touched

### Modified

| Path | Why |
|---|---|
| `libs/atlas-packet/model/skill_usage_info.go` | Add `skill.PriestDoomId` to `isMobAffectingBuff` allowlist (precondition that opens the dual-apply window the consolidation closes). |
| `services/atlas-channel/atlas.com/channel/skill/handler/common.go` | Extend `applyToMobs` with rect verify, cap, prop, kind-aware reflect skip, anomaly + summary logging, mutual-exclusion of cancel/apply branches. Add test seams. |
| `services/atlas-channel/atlas.com/channel/skill/handler/registrations/registrations.go` | Drop the `_ "atlas-channel/skill/handler/doom"` blank import (heal stays). |

### Created

| Path | Purpose |
|---|---|
| `services/atlas-channel/atlas.com/channel/skill/handler/mob_select.go` | Pure helpers: `calculateBoundingBox`, `hasEffectBbox`, `intersectMobIds`, `mobBuffApplyKind`, `propAppliesTo`, `propBranch` enum. No I/O, no globals. |
| `services/atlas-channel/atlas.com/channel/skill/handler/mob_select_test.go` | Unit tests over the pure helpers (bbox, bbox-presence, intersection, kind classification, prop-carve-out). |
| `services/atlas-channel/atlas.com/channel/skill/handler/common_apply_to_mobs_test.go` | Orchestration tests via the package-level seam vars in `common.go`. |

### Deleted

| Path | Reason |
|---|---|
| `services/atlas-channel/atlas.com/channel/skill/handler/doom/doom.go` | Per-skill handler removed. |
| `services/atlas-channel/atlas.com/channel/skill/handler/doom/bbox.go` | Bbox math moves into `mob_select.go`. |
| `services/atlas-channel/atlas.com/channel/skill/handler/doom/doom_test.go` | Coverage migrates to `common_apply_to_mobs_test.go`. |
| `services/atlas-channel/atlas.com/channel/skill/handler/doom/bbox_test.go` | Coverage migrates to `mob_select_test.go`. |

## 3. Contracts the Plan Relies On

Confirmed read on this branch (`task-057-monster-buff-trust-verify`):

- `character.Model` exposes `X() int16`, `Y() int16`, `Stance() byte` (`character/model.go:243-251`).
- `monster.Model.UniqueId() uint32` (`monster/model.go:43`).
- `monster.Processor`:
  - `GetInMapRect(f field.Model, x1, y1, x2, y2 int16, limit uint32) ([]Model, error)` (`monster/processor.go:52`).
  - `ApplyStatus(f, monsterId, characterId, skillId, skillLevel uint32, statuses map[string]int32, duration uint32) error` (`monster/processor.go:83`).
  - `CancelStatus(f, monsterId uint32, statusTypes []string, sourceCharacterId, sourceSkillId uint32, sourceSkillClass string) error` (`monster/processor.go:96`).
- `monster.GetStatusMirror().GetReflect(t tenant.Model, monsterId uint32, kind string) (ReflectInfo, bool)` (`monster/status_mirror.go:201`).
- `effect.Model` exposes `LT()`, `RB()`, `MobCount()`, `Prop()`, `MonsterStatus()`, `Duration()` (`data/skill/effect/model.go:97-138`).
- `point.Model.X() int16`, `point.Model.Y() int16` (used in existing `doom/bbox.go:18-29`).
- `monster2.ReflectKindPhysical = "PHYSICAL"`, `ReflectKindMagical = "MAGICAL"`, `StatusDoom = "DOOM"` (`libs/atlas-constants/monster/skill.go:18-19`, `status.go:16`).
- `skill.PriestDoomId = Id(2311005)` (`libs/atlas-constants/skill/constants.go:3067`).
- `skill.Is(id, ids ...) bool` is the pattern already used in `common.go:107-112` and `dispelSkillClass`.
- `tenant.MustFromContext(ctx) tenant.Model` is the established way to surface tenant inside `applyToMobs` (matches the deleted `doom.go:109`).
- The blank-import line that needs removing reads exactly:
  `_ "atlas-channel/skill/handler/doom" // Priest Doom — task 047`
  in `registrations/registrations.go:7`.
- `isMobAffectingBuff` lives at `libs/atlas-packet/model/skill_usage_info.go:73`. Today PriestDoomId is **not** present (verified — neither the symbol nor the literal `2311005` appears in `libs/atlas-packet/`); the design's §2 makes adding it the responsibility of this task.

## 4. Decisions Locked by the Design

| Question | Locked answer |
|---|---|
| Where does adding `PriestDoomId` to `isMobAffectingBuff` live? | In this task. Same PR. |
| Helper organization | Sibling file `mob_select.go` in package `handler`, not an `internal/mobselect/` subpackage. |
| Failure policy on caster-load / rect-query errors | Drop the cast, error-log only. No warn. |
| Anomaly-warn condition | "Client list NOT contained by server query" only. Empty client list, empty server in-rect, prop misses, reflect skips do NOT warn. |
| Prop applies to apply AND cancel branches | Yes, default. Per-skill carve-out via `propAppliesTo(sid, branch)` table; initial table is **empty** (every today's skill takes the default). |
| Cancel + apply in the same cast | Mutually exclusive after consolidation (a tightening vs. today). |

## 5. Out of Scope (Recap)

- Auto-ban subsystem itself. This task only emits the FR-4.7 signals.
- Adding skills other than Doom to `isMobAffectingBuff`.
- atlas-monsters apply-side filters (boss / elemental immunity).
- Client-side polymorph rendering for Doom.
- Reworking Crash / Dispel beyond the trust-but-verify pass.

## 6. Test Seams (`common.go` package-level vars)

Six package-level variables, all `t.Cleanup`-restored in tests. One change vs.
the deleted Doom handler: a `cancelStatusFunc` is added because the consolidated
path now exercises the cancel branch as well.

```go
var (
    loadCasterFunc    = func(cp *character.ProcessorImpl, id uint32) (character.Model, error) { /* GetById()(id) */ }
    rectQueryFunc     = func(p *monster.Processor, f field.Model, x1, y1, x2, y2 int16, limit uint32) ([]monster.Model, error) { /* GetInMapRect */ }
    propRollFunc      = func(prop float64) bool { /* <=0 false; >=1 true; else rand.Float64() <= prop */ }
    reflectLookupFunc = func(t tenant.Model, mobId uint32, kind string) (monster.ReflectInfo, bool) { /* StatusMirror */ }
    applyStatusFunc   = func(p *monster.Processor, f field.Model, mobId, charId, sid, slvl uint32, m map[string]int32, dur uint32) error { /* ApplyStatus */ }
    cancelStatusFunc  = func(p *monster.Processor, f field.Model, mobId uint32, statusTypes []string, charId, sid uint32, class string) error { /* CancelStatus */ }
)
```

Pure helpers in `mob_select.go` are NOT seamed — tests call them directly.

## 7. Logging Matrix (operationally compact)

| Trigger | Level | Message string | Once-per |
|---|---|---|---|
| `len(mobIds) == 0` | (none) | — | — |
| Cap exceeded | warn | `client_target_count_exceeds_skill_cap` (event `monster_buff_anomaly_over_cap`) | cast |
| No-bbox WZ fallback | debug | `mob_buff_no_effect_bbox` | cast |
| Caster-load error | error | `mob_buff_caster_load_failed` | cast |
| Rect-query error | error | `mob_buff_rect_query_failed` | cast |
| Anomaly mob ids (client minus server) | warn | `client_targeted_mob_outside_server_rect` (event `monster_buff_anomaly_out_of_rect`) | cast |
| Unclassified kind | debug | `mob_buff_unclassified_kind` | per mob |
| Reflect skip | debug | (per-mob trace) | per mob |
| Per-cast summary | debug | `mob_buff_apply_summary` | cast |

All entries inherit `tenant`, `world.id`, `channel.id`, `service.name`,
`session`, `span.id`, `trace.id` from the request-scoped logger.

## 8. Key Risks (Locked Mitigations)

- **Crash/Priest Dispel become subject to atlas-monsters availability** — accepted in brainstorming (§10 of design).
- **`hasEffectBbox` conflates "no rect" with "literal zero-area rect"** — no v83 skill ships a literal zero-area effect; documented in design §5.2.
- **Wire-decoder change in `libs/atlas-packet` is in same PR** — the test order in the plan lands the decoder change first, then the orchestration extension, then the doom-package deletion in distinct commits.

## 9. Useful Read Paths for the Implementer

- `services/atlas-channel/atlas.com/channel/skill/handler/common.go:75-104` — current `applyToMobs` (this is what gets extended).
- `services/atlas-channel/atlas.com/channel/skill/handler/common.go:106-132` — `isCrashOrDispel` and `dispelSkillClass` (referenced by the new code; not modified).
- `services/atlas-channel/atlas.com/channel/skill/handler/doom/doom.go:25-130` — patterns for the test seams (load, rect, prop, reflect, apply) and the per-cast loop. Verbatim source for what the new orchestration mirrors.
- `services/atlas-channel/atlas.com/channel/skill/handler/doom/bbox.go:18-31` — `calculateBoundingBox` (verbatim move into `mob_select.go`).
- `services/atlas-channel/atlas.com/channel/skill/handler/doom/doom_test.go` and `bbox_test.go` — the existing assertions to mirror in the new test files.
- `services/atlas-channel/atlas.com/channel/skill/handler/registry.go` — Lookup registry stays; only the doom blank-import line goes.
- `libs/atlas-packet/model/skill_usage_info.go:73-128` — `isMobAffectingBuff` insertion site.

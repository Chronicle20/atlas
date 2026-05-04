# Monster-Buff Trust-but-Verify (Doom Handler Removal) — Design

Status: Draft
Created: 2026-05-04
PRD: [`prd.md`](prd.md)

---

## 1. Summary

Replace the per-skill Priest Doom handler (`skill/handler/doom/`) and the
"trust the client" `applyToMobs` block (`skill/handler/common.go`) with a single
server-authoritative path inside `applyToMobs`. The path verifies the client's
`affectedMobIds` against an atlas-monsters rect query, enforces the WZ-defined
`mobCount` cap, rolls `prop` per target, skips reflect-active mobs, and emits
structured warn logs whenever the client diverges from server expectations. As
part of the same change, `PriestDoomId` is added to `libs/atlas-packet`'s
`isMobAffectingBuff` allowlist (the precondition that creates the dual-apply
window the consolidation closes), and the entire `skill/handler/doom/`
subpackage is deleted.

Behavior parity with the deleted handler is preserved end-to-end: Doom still
applies to every in-rect mob whose magic-reflect is inactive and whose prop
roll succeeds, with the same status payload (`{DOOM:1}`) and WZ-derived
duration. The change is invisible on the wire (atlas-monsters envelopes
unchanged, `MonsterStatSet` packets unchanged) and observable only as half as
many duplicate commands per cast plus a new structured summary line.

## 2. Discrepancy Resolution

**Codebase fact-check.** The PRD asserts that `PriestDoomId` is already in
`libs/atlas-packet/model/skill_usage_info.go:isMobAffectingBuff`. On this
branch it is not (neither the symbol nor `2311005` appear anywhere under
`libs/atlas-packet/`). The user clarified during brainstorming that the entry
exists as an unstaged change on `main` — it is the change that surfaced the
dual-apply that motivated this task.

**Decision.** Adding `PriestDoomId` to `isMobAffectingBuff` is performed by
this task, in the same commit series as the consolidation. The PRD's §2
non-goal "Expanding `isMobAffectingBuff` to additional skills" is interpreted
as "do not add skills _other than Doom_"; Doom itself is the one entry whose
addition this task owns. Without that addition the consolidation has nothing
to consolidate (the generic path's `len(mobIds) == 0` guard at `common.go:77`
short-circuits before any of the new logic runs).

## 3. Architecture

### 3.1 File layout

```
services/atlas-channel/atlas.com/channel/skill/handler/
├── common.go                      # MODIFIED. UseSkill + applyToMobs (orchestration).
├── mob_select.go                  # NEW. Pure helpers + carve-out tables.
├── common_apply_to_mobs_test.go   # NEW. Orchestration tests via package vars.
├── mob_select_test.go             # NEW. Pure-helper tests.
├── registrations/registrations.go # MODIFIED. Drop the doom blank-import.
└── doom/                          # DELETED in entirety.

libs/atlas-packet/model/skill_usage_info.go   # MODIFIED. Add PriestDoomId to isMobAffectingBuff.
```

### 3.2 Why a sibling file, not an `internal/mobselect/` subpackage

`applyToMobs` already shares the `handler` package with `isCrashOrDispel`,
`dispelSkillClass`, and `applyToParty` — all small private helpers that live
beside their orchestrator. The new pure helpers (bbox, intersection, kind
classification, prop carve-out) are the same shape; promoting them to a
subpackage would force an exported API (`mobselect.Select(...)` returning a
struct) for ~150 LOC of code with one caller. The hybrid placement keeps
`common.go` thin (orchestration + I/O glue) while pinning the math in a
sibling file the test suite can target directly.

### 3.3 Symbols introduced in `mob_select.go`

All package-private. All pure (no globals, no I/O).

| Symbol | Purpose |
|---|---|
| `calculateBoundingBox(casterX, casterY int16, facingLeft bool, lt, rb point.Model) (x1, y1, x2, y2 int16)` | Verbatim move from `doom/bbox.go`. Mirrors Cosmic `StatEffect.calculateBoundingBox`. |
| `hasEffectBbox(lt, rb point.Model) bool` | Returns `false` only when **all four** components (`lt.X`, `lt.Y`, `rb.X`, `rb.Y`) are zero — that is the WZ "no rect" sentinel and drives the FR-4.2 fallback. |
| `intersectMobIds(client, server []uint32) (applied, anomaly []uint32)` | Walks `client` in order, classifying each id as `applied` (in `server`) or `anomaly` (not in `server`). Server-only ids are dropped per FR-4.1. Preserves client order (FR-4.4). |
| `mobBuffApplyKind(sid skill.Id) string` | Returns `"MAGICAL"` for `PriestDoomId`; `""` for unknown skills. (`""` triggers the FR-4.6 "skip reflect check, debug log" branch.) Crash/Dispel kinds continue to come from the existing `dispelSkillClass`. |
| `propAppliesTo(sid skill.Id, branch propBranch) bool` | Per-skill carve-out table for FR-4.5. Default `true`. Initial table is **empty**: every current skill (Doom apply, Crash family cancel, Priest Dispel cancel) takes the default. The table is the contract for future skills that need "prop only on apply" or "prop only on cancel". |
| `type propBranch int` with `propBranchApply` / `propBranchCancel` constants | Discriminates which branch the orchestrator is about to take when consulting the carve-out table. |

### 3.4 Test seams in `common.go`

Mirrors the existing Doom pattern (package-level `var`s, `t.Cleanup`-restored
in tests). One change vs. Doom: a `cancelStatusFunc` is added because the
consolidated path now exercises the cancel branch as well.

```go
var (
    loadCasterFunc    = func(cp *character.Processor, id uint32) (character.Model, error) { /* GetById */ }
    rectQueryFunc     = func(p *monster.Processor, f field.Model, x1, y1, x2, y2 int16, limit uint32) ([]monster.Model, error) { /* GetInMapRect */ }
    propRollFunc      = func(prop float64) bool { /* <=0 false; >=1 true; else rand.Float64() <= prop */ }
    reflectLookupFunc = func(t tenant.Model, mobId uint32, kind string) (monster.ReflectInfo, bool) { /* StatusMirror */ }
    applyStatusFunc   = func(p *monster.Processor, f field.Model, mobId, charId, sid, slvl uint32, m map[string]int32, dur uint32) error { /* ApplyStatus */ }
    cancelStatusFunc  = func(p *monster.Processor, f field.Model, mobId uint32, statusTypes []string, charId uint32, sid uint32, class string) error { /* CancelStatus */ }
)
```

These are the **only** mutable seams. `mob_select.go` helpers receive
inputs and return outputs — tests call them directly without monkey-patching.

## 4. Orchestration in `applyToMobs`

The new body (excluding logging plumbing) follows this order. Numeric
suffixes refer to the PRD's FR-IDs. Steps marked **drop** abort the cast with
no further emission for any mob.

1. **Empty client list** — if `len(info.AffectedMobIds()) == 0`, return.
   Preserves today's early-exit at `common.go:77-79`.
2. **mobCount cap (FR-4.3)** — if `len(mobIds) > e.MobCount()`:
   emit FR-4.7.2 warn log and **drop** the cast. Cap check runs before any
   atlas-monsters round-trip.
3. **Bbox classification** — if `hasEffectBbox(e.LT(), e.RB())` is `false`,
   skip to step 7 with `applied = mobIds`, `anomaly = nil`,
   `mobsInRectCount = -1` (FR-4.2 fallback). Emit a debug log noting
   bbox-fallback. No warn log.
4. **Caster load (FR-4.1)** — `loadCasterFunc(cp, characterId)`. On error,
   log `WithError(err).Errorf("applyToMobs: failed to load caster ...")` and
   **drop** the cast. (Section 5.1 records why the failure policy is
   bail-on-error.)
5. **Rect query (FR-4.1)** — derive `facingLeft = (c.Stance() & 1) == 1`,
   compute `(x1, y1, x2, y2)` via `calculateBoundingBox`, call
   `rectQueryFunc(mp, f, x1, y1, x2, y2, e.MobCount())`. On error, log
   `WithError(err).Errorf(...)` and **drop** the cast.
6. **Intersection (FR-4.1, FR-4.4)** —
   `applied, anomaly := intersectMobIds(mobIds, serverMobIds)`. If
   `len(anomaly) > 0`, emit the FR-4.7.1 warn log once. Cast continues with
   `applied` even when anomalies are present.
7. **Per-mob loop** — for each `mobId` in `applied` (preserves client
   order):
   1. **Reflect skip (FR-4.6)** — determine `kind`:
      - `isCrashOrDispel(sid)` → `dispelSkillClass(sid)`,
      - else → `mobBuffApplyKind(sid)`.
      If `kind == ""`, log debug "unclassified kind, reflect check skipped"
      and proceed without checking. Otherwise call
      `reflectLookupFunc(t, mobId, kind)`; if a reflect of the matching kind
      exists, increment `reflectSkipped` and `continue`.
   2. **Branch selection** — pick the emit branch:
      - `isCrashOrDispel(sid)` → `propBranchCancel`, will call `cancelStatusFunc`,
      - else if `len(e.MonsterStatus()) > 0` → `propBranchApply`, will call
        `applyStatusFunc`,
      - else → no emit branch applies; `continue` (defensive — should not
        occur for buff-classified skills).
   3. **Prop roll (FR-4.5)** — if `propAppliesTo(sid, branch)` is `true`,
      call `propRollFunc(e.Prop())`; on `false`, increment `propSkipped`
      and `continue`. If `propAppliesTo` returns `false`, skip the roll
      entirely (treat as "always pass" for this branch on this skill).
   4. **Emit** — call the chosen branch's seam:
      - apply: `applyStatusFunc(mp, f, mobId, characterId, sid, slvl, monsterStatuses, e.Duration())`,
      - cancel: `cancelStatusFunc(mp, f, mobId, nil, characterId, sid, dispelSkillClass(sid))` — `nil` for `statusTypes` matches today's call at `common.go:90`, which lets atlas-monsters infer the affected statuses from the source skill class.
      Increment `applied` counter on success.
8. **Summary log (FR-4.8)** — emit one debug `"mob_buff_apply_summary"`
   line with all of: `caster`, `skill_id`, `skill_level`, `mobs_in_rect`
   (or `-1` for fallback), `client_mob_count`, `applied`, `reflect_skipped`,
   `prop_skipped`, `out_of_rect_dropped` (= `len(anomaly)`).

### 4.1 `applyToMobs` signature

Unchanged from today:

```go
func applyToMobs(l logrus.FieldLogger, ctx context.Context, f field.Model,
    characterId uint32, info packetmodel.SkillUsageInfo, e effect.Model)
```

### 4.2 Today's CancelStatus + ApplyStatus dual-emit is removed

Today's `applyToMobs` (`common.go:87-103`) runs both branches when a
`isCrashOrDispel` skill happens to carry a non-empty `MonsterStatus` map: it
emits `CancelStatus` per mob and then `ApplyStatus` per mob. The PRD §4.9
asserts that "A skill MUST NOT trigger both branches in the same cast." The
new orchestration enforces this with the per-mob branch selection in 7.ii: a
crash/dispel skill takes the cancel branch and stops there; a non-crash buff
takes the apply branch only. v83 WZ data does not currently put a non-empty
`MonsterStatus` on Crash effects, so this tightening is a contract fix
without a behavior change on production data.

## 5. Failure Handling

### 5.1 Bail-on-error policy (decided during brainstorming)

| Failure | New policy | Rationale |
|---|---|---|
| `loadCasterFunc` returns error | Drop cast, `Errorf` log. | Without `(X, Y, Stance)` the rect cannot be computed; trust-but-verify is impossible. |
| `rectQueryFunc` returns error | Drop cast, `Errorf` log. | Falling back to "trust client" would let any atlas-monsters wobble bypass server authority — exactly the threat model FR-4.7 is built for. |
| `applyStatusFunc` / `cancelStatusFunc` returns error per mob | Log warning, continue with the next mob. | Per-mob Kafka emit failure should not poison the rest of the applied set. Matches Doom's current `_ = applyStatusFunc(...)` swallow. |
| `propRollFunc` panic | Not caught. | Pure function on a `float64`; no failure mode in production. |
| `reflectLookupFunc` returns `(zero, false)` | Treated as "no reflect"; cast continues normally. | This is the existing semantics of `StatusMirror.GetReflect`. |

This is a behavior change for crash/dispel skills relative to today: they
currently bypass the rect query entirely, so an atlas-monsters outage does
not stop a Crash. After consolidation, they are subject to the same
bail-on-error rule as Doom is today. Accepted in brainstorming.

### 5.2 No-bbox WZ sentinel

`hasEffectBbox` treats both points being all-zero as the sentinel for "no
rect contract in WZ data". Distinguishing a degenerate
`(0, 0)`–`(0, 0)` rect from a "no rect" effect is impossible from the
parsed `effect.Model`; no skill in v83 WZ data has a literal zero-area
effect bbox, so the conflation is safe. If a future skill ships a
deliberate point-rect, it will need to set `lt`/`rb` to non-zero values
that bound the point.

## 6. Logging Matrix

All log entries inherit `tenant`, `world.id`, `channel.id`, `service.name`,
`session`, `span.id`, `trace.id` from the request-scoped logger.

| Trigger | Level | Event/Message | Cardinality | Carries |
|---|---|---|---|---|
| `len(mobIds) == 0` | (none) | n/a | — | Early return; no log. |
| Cap exceeded (FR-4.3) | `warn` | `client_target_count_exceeds_skill_cap` (event `monster_buff_anomaly_over_cap`) | once per cast | `character_id`, `skill_id`, `skill_level`, `mob_count_cap`, `client_mob_count`, `client_mob_ids` |
| Bbox fallback (FR-4.2) | `debug` | `mob_buff_no_effect_bbox` | once per cast | `skill_id`, `skill_level`, `client_mob_count` |
| Caster-load fail | `error` | `mob_buff_caster_load_failed` | once per cast | `character_id`, `skill_id`, `error` |
| Rect-query fail | `error` | `mob_buff_rect_query_failed` | once per cast | `character_id`, `skill_id`, `rect`, `error` |
| Anomaly mob ids (FR-4.7.1) | `warn` | `client_targeted_mob_outside_server_rect` (event `monster_buff_anomaly_out_of_rect`) | once per cast | `character_id`, `skill_id`, `skill_level`, `rect={x1,y1,x2,y2}`, `mob_count_cap`, `client_mob_ids`, `server_mob_ids`, `anomaly_mob_ids` |
| Unclassified kind | `debug` | `mob_buff_unclassified_kind` | per affected mob (rare) | `skill_id`, `mob_id` |
| Reflect skip | `debug` | (per-mob trace, no event) | per skipped mob | `skill_id`, `mob_id`, `kind` |
| Prop skip | (counted only) | n/a | (counter increment) | rolled into summary |
| Per-cast summary (FR-4.8) | `debug` | `mob_buff_apply_summary` | once per cast | full PRD §4.8 field set |

`logrus.FieldLogger.WithFields(...)` is used for all multi-field entries.

## 7. Test Strategy

### 7.1 `mob_select_test.go` — pure helpers

| Test | Pins |
|---|---|
| `TestBoundingBox_FacingRight_SymmetricRect` | `calculateBoundingBox` mirroring (verbatim move from `doom/bbox_test.go`). |
| `TestBoundingBox_FacingLeft_SymmetricRect` | (verbatim move) |
| `TestBoundingBox_Asymmetric_FacingRight` | (verbatim move) |
| `TestBoundingBox_Asymmetric_FacingLeft` | (verbatim move) |
| `TestHasEffectBbox` | All-zero → false; any non-zero → true. |
| `TestIntersectMobIds_AllInRect` | `applied = client`, `anomaly = nil`. |
| `TestIntersectMobIds_ClientOrderPreserved` | Even if `server` is reordered, `applied` follows `client`. |
| `TestIntersectMobIds_AnomalySubset` | Mixed list yields correct partition. |
| `TestIntersectMobIds_ServerOnlyDropped` | Server ids absent from client are not in either slice. |
| `TestMobBuffApplyKind` | `PriestDoomId → "MAGICAL"`; unknown id `→ ""`. |
| `TestPropAppliesTo_DefaultsTrue` | Empty table; default branch returns `true`. |
| `TestPropAppliesTo_CarveOutHonored` | Insert one fixture entry inside the test (without mutating the production table) to demonstrate the contract — the test installs and tears down a private `propBranch` mapping via a `t.Cleanup`-restored override (or the helper accepts an optional table param). |

### 7.2 `common_apply_to_mobs_test.go` — orchestration

Uses the package-level seam vars; same `installFakes` style as today's
`doom_test.go`.

| Test | Pins |
|---|---|
| `TestApplyToMobs_EmptyClientList_NoOp` | No seams called. |
| `TestApplyToMobs_OverCap_Drops_AndWarns` | Cast with `len(mobIds) > MobCount()` produces zero `applyStatus`/`cancelStatus` calls and one warn log line `client_target_count_exceeds_skill_cap`. |
| `TestApplyToMobs_NoBbox_TrustsClient` | Effect with `lt = rb = (0,0)` → applied set equals client list, no rect query attempted. |
| `TestApplyToMobs_CasterLoadFails_Drops` | `loadCasterFunc` returns error → no rect query, no apply. |
| `TestApplyToMobs_RectQueryFails_Drops` | `rectQueryFunc` returns error → no apply. |
| `TestApplyToMobs_RectIntersectionApplied` | Server returns 3 ids, client lists 4 (one extra) → 3 applies in client order, one warn `client_targeted_mob_outside_server_rect`. |
| `TestApplyToMobs_DoomMagicReflectSkipped` | One in-rect mob has MAGICAL reflect → that mob is skipped; reflect counter increments. |
| `TestApplyToMobs_CrashFamily_PhysicalReflectSkipped` | Crusader Armor Crash, mob with PHYSICAL reflect → skipped. |
| `TestApplyToMobs_PriestDispel_MagicalReflectSkipped` | Priest Dispel, mob with MAGICAL reflect → skipped. |
| `TestApplyToMobs_PropZero_AppliesNothing` | `e.Prop() = 0`, `propRollFunc` honors it → all mobs skipped, prop counter equals applied set size. |
| `TestApplyToMobs_PropOne_AppliesAll` | `e.Prop() = 1` → no prop skips. |
| `TestApplyToMobs_DoomTakesApplyBranch` | Sid = `PriestDoomId`, `MonsterStatus` non-empty → `applyStatusFunc` called; `cancelStatusFunc` not called. |
| `TestApplyToMobs_CrashTakesCancelBranch` | Sid = `CrusaderArmorCrashId` → `cancelStatusFunc` called with class `"PHYSICAL"`; `applyStatusFunc` not called. |
| `TestApplyToMobs_PropCarveOutSuppressesPropOnCancel` | Inject a deny-table entry for a fake skill → cancel branch fires regardless of `propRollFunc` outcome. (Demonstrates the carve-out contract end-to-end.) |
| `TestApplyToMobs_PassesDoomStatusAndDuration` | (verbatim port of today's `TestDoom_Apply_PassesDoomStatusAndDuration`) |

### 7.3 Coverage equivalence to deleted `doom_test.go` / `bbox_test.go`

The four `Test_BoundingBox*` cases land verbatim in `mob_select_test.go`.
The four `TestDoom_Apply_*` cases map as follows:

| Deleted | Replacement |
|---|---|
| `TestDoom_Apply_AppliesToAllInRectMobs` | `TestApplyToMobs_RectIntersectionApplied` (with anomaly = []). |
| `TestDoom_Apply_SkipsMagicReflectMobs` | `TestApplyToMobs_DoomMagicReflectSkipped`. |
| `TestDoom_Apply_RespectsPropZero` | `TestApplyToMobs_PropZero_AppliesNothing`. |
| `TestDoom_Apply_PassesDoomStatusAndDuration` | `TestApplyToMobs_PassesDoomStatusAndDuration`. |

Every assertion the deleted suite makes is preserved in the new suite; no
behavior is unverified post-migration.

## 8. Migration Steps (rough order)

This is **not** the implementation plan — `/plan-task` produces that. It is
a sketch so the design is self-consistent.

1. Add `PriestDoomId` to `libs/atlas-packet/model/skill_usage_info.go`'s
   `isMobAffectingBuff` allowlist. (Wire-decoder change. After this,
   `affectedMobIds` is populated for Doom casts and the dual-apply window
   opens.)
2. Add `mob_select.go` with the pure helpers. (No call sites yet.)
3. Extend `applyToMobs` in `common.go` with the new orchestration
   (steps 1–8 of §4) and the new test seams. (Closes the dual-apply window
   for Doom; for Crash/Dispel, replaces today's "trust client unmodified"
   with the same trust-but-verify path.)
4. Land `mob_select_test.go` and `common_apply_to_mobs_test.go`.
5. Delete `services/atlas-channel/atlas.com/channel/skill/handler/doom/`
   in entirety.
6. Drop the `_ "atlas-channel/skill/handler/doom"` line in
   `registrations/registrations.go`. Verify the `heal` import remains.
7. Run `go build ./...` and `go test ./...` from
   `services/atlas-channel/atlas.com/channel/`. Verify atlas-channel still
   builds across all consumers.

Steps 1–4 may be combined into a single working state where Doom keeps
firing through both paths only inside the developer's working tree; once 5–6
land, the dual-apply is gone. Splitting commits is at the implementer's
discretion (driven by `/plan-task`).

## 9. Out of Scope (recap)

- Auto-ban subsystem (PRD §2). This task only emits the FR-4.7 signals.
- Adding skills other than Doom to `isMobAffectingBuff`.
- Per-skill dispatcher (`Lookup`) infrastructure for Heal/Cure/MPEater/Drain.
- atlas-monsters' apply-side filters (boss / elemental immunity).
- Client-side polymorph rendering. Whether v83 visually polymorphs all
  three Doom targets is unrelated to the dual-apply bug this task fixes
  (PRD §4.12).

## 10. Risks & Mitigations

| Risk | Mitigation |
|---|---|
| Crash / Priest Dispel become subject to atlas-monsters availability (vs. today's bypass). | Accepted in brainstorming. Dispels recover within seconds; outages already affect Doom and atlas-monsters status broadcasts generally. |
| The carve-out table is initially empty and could rot. | Test `TestApplyToMobs_PropCarveOutSuppressesPropOnCancel` exercises the table contract end-to-end with a fake skill, so the mechanism is provably wired even with no production entries. |
| `hasEffectBbox` conflates "no rect" with "literal zero-area rect". | No v83 skill has a literal zero-area effect; flagged in §5.2. Future skill that needs a point-rect must use non-zero deltas. |
| Adding `PriestDoomId` to `isMobAffectingBuff` is a wire-decoder change in a shared library; misorder of commits could expose a window where the channel ignores the new mob ids. | Order in §8 lands the decoder change first and the orchestration extension immediately after, both inside one PR. CI's atlas-channel `go test` exercises the consolidated path before merge. |

## 11. Decisions (from brainstorming)

| Question | Decision |
|---|---|
| Where does adding `PriestDoomId` to `isMobAffectingBuff` live? | In this task. (User clarification: it's an unstaged change on `main` today.) |
| Helper organization (FR-4.11 leaves it open). | Hybrid: pure helpers in `mob_select.go` sibling file in `handler` package; orchestration stays in `common.go`. (Approach C.) |
| Failure policy for caster-load / rect-query errors on the consolidated path. | Bail the cast on either failure (Approach A). Error-log only; no warn. |

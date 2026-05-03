# Priest Doom (Skill 2311005) Implementation Plan

Version: v2 (revised after wrong-channel discovery)
Status: Draft
Predecessor: see `postmortem.md`. v1 plan (Tasks 0–10 targeting the
magic-attack handler) is superseded; the previously-merged commits on
this branch are addressed in Task R (Revert) below.

> **For agentic workers:** REQUIRED SUB-SKILL: Use
> `superpowers:subagent-driven-development` (recommended) or
> `superpowers:executing-plans` to implement this plan task-by-task.
> Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal**: Make Priest Doom (skill `2311005`) work in-game on the v83
client. Each cast applies the DOOM monster status to up to `mobCount`
mobs in the caster's facing rectangle, consumes one Magic Rock, and
respects the existing elemental, boss, and magic-reflect immunities.

**Architecture**: per-skill handler under
`services/atlas-channel/atlas.com/channel/skill/handler/doom/`, registered
for `PriestDoomId`. Server-authoritative target selection via a new
atlas-monsters rect query. Generic `itemConsume` plumbing in
`handler.UseSkill`'s cost block (covers Doom + every other `itemConsume`
skill).

**Tech Stack**: Go (Go modules per service); existing patterns
(`skill/handler/heal/heal.go` for the per-skill handler shape, JSON:API
REST for the new rect endpoint, Kafka commands for status apply).

**Source design**: `docs/tasks/task-047-priest-doom/design.md`
**Source PRD**: `docs/tasks/task-047-priest-doom/prd.md`
**Postmortem**: `docs/tasks/task-047-priest-doom/postmortem.md`

---

## Task ordering rationale

1. **Task R (Revert)** runs first to delete the wrong-path code that
   merged on this branch. Subsequent tasks build on a clean baseline.
2. **Task A (atlas-monsters rect query)** is leaf-most: it adds new code
   that no atlas-channel code yet calls, so it can land independently
   and be tested in isolation.
3. **Task B (atlas-channel rect-query client)** wraps Task A; runs after
   it to avoid introducing a build-broken stub on atlas-channel.
4. **Task C (atlas-channel itemConsume in UseSkill cost block)** is
   independent of Tasks A/B; can run in parallel but is sequenced here
   for review simplicity.
5. **Task D (atlas-channel doom handler)** depends on B; builds the
   per-skill handler that consumes the rect query.
6. **Task E (effect.Model accessors)** is a small dependency Task D
   needs (`MobCount`, `Prop`); land it just-in-time before D.
7. **Task F (final cross-service build/test + manual verification handoff)**.

---

## File map

| File | Action | Responsibility |
|---|---|---|
| `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go` | Modify (revert) | Remove Doom-gated reflect probe and `itemConsume` cost-gate addition. Keep `processDamageInfoEntry` extraction. |
| `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common_test.go` | Modify (revert) | Remove 4 `TestProcessDamageInfoEntry_Doom_*` tests + helpers. |
| `services/atlas-monsters/atlas.com/monsters/monster/processor.go` | Modify (add) | New `GetInFieldRect` method on monster processor. |
| `services/atlas-monsters/atlas.com/monsters/monster/registry.go` | Modify (add) | New `GetMonstersInFieldRect` walk on the registry. |
| `services/atlas-monsters/atlas.com/monsters/monster/processor_test.go` | Modify (add) | Tests for `GetInFieldRect`. |
| `services/atlas-monsters/atlas.com/monsters/monster/resource.go` (or equivalent) | Modify (add) | New REST endpoint for the rect query. |
| `services/atlas-channel/atlas.com/channel/monster/processor.go` | Modify (add) | New `GetInFieldRect` client method (REST call wrapper). |
| `services/atlas-channel/atlas.com/channel/data/skill/effect/model.go` | Modify (add) | New `MobCount() uint32` and `Prop() float64` accessors if missing; populate from REST. |
| `services/atlas-channel/atlas.com/channel/data/skill/effect/rest.go` | Modify (add) | Wire `MobCount` and `Prop` from `RestModel` into `Model`. |
| `services/atlas-channel/atlas.com/channel/skill/handler/common.go` | Modify (add) | Generic `itemConsume` charge in the existing cost block. |
| `services/atlas-channel/atlas.com/channel/skill/handler/common_test.go` | Add (new) | Tests for the new `itemConsume` charge path. |
| `services/atlas-channel/atlas.com/channel/skill/handler/doom/doom.go` | Add (new) | Per-skill Doom handler + registration in `init()`. |
| `services/atlas-channel/atlas.com/channel/skill/handler/doom/bbox.go` | Add (new) | Pure `calculateBoundingBox` helper. |
| `services/atlas-channel/atlas.com/channel/skill/handler/doom/doom_test.go` | Add (new) | Handler tests (apply, reflect skip, prop, left-facing). |
| `services/atlas-channel/atlas.com/channel/skill/handler/doom/bbox_test.go` | Add (new) | Pure-function bbox tests. |
| `services/atlas-channel/atlas.com/channel/skill/handler/registrations/registrations.go` (or wherever heal is imported for side effects) | Modify (add) | Add a blank import for the new `doom` package so its `init()` registers the handler. |

No changes to `libs/atlas-packet`, `libs/atlas-constants`, atlas-data,
atlas-monsters' `ApplyStatusEffect` / `isElementallyImmune` /
`isBossAllowedStatus` / `information.ModelBuilder` (all of those are
already correct from the previous tasks on this branch and stay).

---

## Task R: Revert wrong-path Doom code

Remove the channel-side Doom production changes that target
`processAttack` / `processDamageInfoEntry`. They are unreachable for the
v83 Doom packet and would only confuse a reviewer of the corrected
implementation.

**Files**:
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go`
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common_test.go`

- [ ] **Step 1**: Remove the Doom-gated reflect probe in
  `processDamageInfoEntry`'s empty-damage branch.

  Locate the block currently between the `loadVenomStats` /
  `snapshotVenomDamagePerTick` recomputation and the
  `_ = deps.applyStatus(...)` call in the `if len(damages) == 0` branch.
  Delete the entire `if _, isDoom := ms[monster2.StatusDoom]; isDoom && attackKind != "" { ... }`
  guard plus its inner reflect probe and the trailing blank line.

  Verify with:
  ```bash
  grep -n "isDoom\|StatusDoom" services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go
  ```
  Expected: no hit (the symbol is no longer used in this file).

- [ ] **Step 2**: Remove the `itemConsume` charge in `processAttack`'s
  HP/MP cost gate.

  In `processAttack`, find the `if _, registered := handler.Lookup(skill3.Id(ai.SkillId())); !registered { ... }`
  block. Delete the entire `if itemId := se.ItemConsume(); itemId > 0 { ... }`
  branch (and its missing-item warning). Keep the surrounding HP and MP
  charges intact.

- [ ] **Step 3**: Drop now-unused imports.

  Run `goimports -w services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go`
  if available. Otherwise, edit the import block to drop:
  - `atlas-channel/consumable`
  - `inventoryconst "github.com/Chronicle20/atlas/libs/atlas-constants/inventory"`
  - `"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"`
  - `itemconst "github.com/Chronicle20/atlas/libs/atlas-constants/item"`
  - `charcon "github.com/Chronicle20/atlas/libs/atlas-constants/character"`

  Keep:
  - `"github.com/Chronicle20/atlas/libs/atlas-constants/field"` (used by
    the helper extraction)
  - `monster2 "github.com/Chronicle20/atlas/libs/atlas-constants/monster"`
    (used elsewhere in the file by `attackKindFromAttackType`)

  Verify with:
  ```bash
  ( cd services/atlas-channel/atlas.com/channel && go build ./socket/handler/... )
  ```
  Expected: success.

- [ ] **Step 4**: Remove the four `TestProcessDamageInfoEntry_Doom_*`
  tests + supporting helpers.

  In `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common_test.go`,
  delete:
  - `applyStatusCall` struct
  - `damageEntryFakes` struct + its `deps()` method
  - `newDoomEffect` function
  - `newDoomAttackInfo` function
  - `TestProcessDamageInfoEntry_Doom_EmptyDamagesAppliesStatus`
  - `TestProcessDamageInfoEntry_Doom_BlockedByReflect`
  - `TestProcessDamageInfoEntry_Doom_MultiTargetSpread`
  - `TestProcessDamageInfoEntry_NonDoom_EmptyDamagesIgnoresReflectProbe`

- [ ] **Step 5**: Drop test imports orphaned by Step 4.

  After the deletions, the test file should still need `monster`,
  `monster2`, `packetmodel`, `tenant`, `uuid`, `time` (the
  `TestComputeReflect_*` and `TestReflectFlow_*` tests use them).
  Drop:
  - `"atlas-channel/data/skill/effect"`
  - `"atlas-channel/effective_stats"`
  - `"errors"`
  - `"io"`
  - `"github.com/Chronicle20/atlas/libs/atlas-constants/channel"`
  - `"github.com/Chronicle20/atlas/libs/atlas-constants/field"`
  - `_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"`
  - `skillconst "github.com/Chronicle20/atlas/libs/atlas-constants/skill"`
  - `"github.com/Chronicle20/atlas/libs/atlas-constants/world"`
  - `"github.com/sirupsen/logrus"`

  (Verify each is unused after the test deletion before removing — some
  may still be referenced by the kept tests.)

- [ ] **Step 6**: Build and test atlas-channel; confirm no regressions.

  ```bash
  ( cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./... )
  ```
  Expected: green.

- [ ] **Step 7**: Commit.

  ```bash
  git add services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go \
          services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common_test.go
  git commit -m "revert(atlas-channel): drop wrong-path Doom probe and itemCon charge from processAttack"
  ```

---

## Task A: atlas-monsters — `GetInFieldRect` query

Add a registry walk + processor method that returns monsters in a field
within a rectangle, capped to a limit. The rectangle is given as
`(x1, y1, x2, y2)` and is normalized (any ordering of the corners is
accepted). Inclusive on all four edges.

**Files**:
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/registry.go`
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/processor.go`
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/processor_test.go`

- [ ] **Step 1**: Add the registry walk.

  In `registry.go`, add:

  ```go
  // GetMonstersInFieldRect returns monsters in the given field whose (x, y)
  // is inside the inclusive rectangle bounded by (x1, y1, x2, y2). The
  // corners may be passed in any order. The result is sorted by distance
  // from the rectangle center (ascending) and truncated to limit when
  // limit > 0; limit == 0 means "no cap".
  func (r *Registry) GetMonstersInFieldRect(t tenant.Model, f field.Model, x1, y1, x2, y2 int16, limit uint32) []Model {
      r.mu.RLock()
      defer r.mu.RUnlock()

      lx, hx := minInt16(x1, x2), maxInt16(x1, x2)
      ly, hy := minInt16(y1, y2), maxInt16(y1, y2)
      cx := (int32(lx) + int32(hx)) / 2
      cy := (int32(ly) + int32(hy)) / 2

      type scored struct {
          m   Model
          d2  int64
      }
      scoredAll := make([]scored, 0, 16)
      for _, m := range r.fieldMonsters(t, f) {
          if m.X() < lx || m.X() > hx || m.Y() < ly || m.Y() > hy {
              continue
          }
          dx := int64(int32(m.X()) - cx)
          dy := int64(int32(m.Y()) - cy)
          scoredAll = append(scoredAll, scored{m: m, d2: dx*dx + dy*dy})
      }
      sort.Slice(scoredAll, func(i, j int) bool { return scoredAll[i].d2 < scoredAll[j].d2 })
      if limit > 0 && uint32(len(scoredAll)) > limit {
          scoredAll = scoredAll[:limit]
      }
      out := make([]Model, len(scoredAll))
      for i, s := range scoredAll {
          out[i] = s.m
      }
      return out
  }
  ```

  `fieldMonsters(t, f)` is the existing internal accessor used by
  `GetMonstersInField` (look it up; if the existing method is named
  differently, mirror its accessor pattern). If `min`/`max` helpers for
  `int16` don't exist in the package, add them next to the new method.

- [ ] **Step 2**: Add the processor wrapper.

  In `processor.go`, add to `Processor` interface:

  ```go
  GetInFieldRect(f field.Model, x1, y1, x2, y2 int16, limit uint32) ([]Model, error)
  ```

  And to `ProcessorImpl`:

  ```go
  func (p *ProcessorImpl) GetInFieldRect(f field.Model, x1, y1, x2, y2 int16, limit uint32) ([]Model, error) {
      return GetMonsterRegistry().GetMonstersInFieldRect(p.t, f, x1, y1, x2, y2, limit), nil
  }
  ```

  Returns no error today (the registry walk cannot fail). The error
  return is reserved for the case where this becomes a remote call —
  matches the `GetById(uniqueId uint32) (Model, error)` shape.

- [ ] **Step 3**: Tests.

  Append to `processor_test.go`:

  ```go
  func TestGetInFieldRect_Inside(t *testing.T) {
      r := GetMonsterRegistry()
      ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
      ctx := context.Background()
      r.Clear(ctx)

      f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
      r.CreateMonster(ctx, ten, f, 9300018, 10, 10, 0, 0, 0, 100, 50)
      r.CreateMonster(ctx, ten, f, 9300018, 50, 50, 0, 0, 0, 100, 50)
      r.CreateMonster(ctx, ten, f, 9300018, 200, 200, 0, 0, 0, 100, 50)

      got := r.GetMonstersInFieldRect(ten, f, -100, -100, 100, 100, 0)
      if len(got) != 2 {
          t.Fatalf("len(got) = %d, want 2", len(got))
      }
  }

  func TestGetInFieldRect_LimitTruncates(t *testing.T) {
      r := GetMonsterRegistry()
      ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
      ctx := context.Background()
      r.Clear(ctx)

      f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
      for i := int16(0); i < 10; i++ {
          r.CreateMonster(ctx, ten, f, 9300018, i*10, 0, 0, 0, 0, 100, 50)
      }

      got := r.GetMonstersInFieldRect(ten, f, -50, -50, 200, 50, 3)
      if len(got) != 3 {
          t.Fatalf("len(got) = %d, want 3", len(got))
      }
  }

  func TestGetInFieldRect_BoundsInclusive(t *testing.T) {
      r := GetMonsterRegistry()
      ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
      ctx := context.Background()
      r.Clear(ctx)

      f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
      r.CreateMonster(ctx, ten, f, 9300018, 100, 100, 0, 0, 0, 100, 50)

      got := r.GetMonstersInFieldRect(ten, f, -100, -100, 100, 100, 0)
      if len(got) != 1 {
          t.Errorf("expected the corner-aligned monster to be included; got %d", len(got))
      }
  }

  func TestGetInFieldRect_OtherFieldsExcluded(t *testing.T) {
      r := GetMonsterRegistry()
      ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
      ctx := context.Background()
      r.Clear(ctx)

      f1 := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
      f2 := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40001)).Build()
      r.CreateMonster(ctx, ten, f1, 9300018, 0, 0, 0, 0, 0, 100, 50)
      r.CreateMonster(ctx, ten, f2, 9300018, 0, 0, 0, 0, 0, 100, 50)

      got := r.GetMonstersInFieldRect(ten, f1, -100, -100, 100, 100, 0)
      if len(got) != 1 {
          t.Errorf("expected only field 40000's monster; got %d", len(got))
      }
  }
  ```

  Run:
  ```bash
  ( cd services/atlas-monsters/atlas.com/monsters && go test ./monster -run 'TestGetInFieldRect_' -count=1 -v )
  ```
  Expected: all PASS.

- [ ] **Step 4**: Add the REST endpoint.

  Locate the existing monster REST handler (likely
  `services/atlas-monsters/atlas.com/monsters/monster/resource.go` or
  similar). Mirror the existing `GET .../monsters` (in-field listing)
  shape. Add query parameter parsing for `x1`, `y1`, `x2`, `y2`,
  `limit`. Emit JSON:API list using the existing transform.

  If the route registration is in a separate router file, add a route
  for the new query-string endpoint there.

- [ ] **Step 5**: Build and test the service.

  ```bash
  ( cd services/atlas-monsters/atlas.com/monsters && go build ./... && go test ./... )
  ```
  Expected: green.

- [ ] **Step 6**: Commit.

  ```bash
  git add services/atlas-monsters/atlas.com/monsters/monster/registry.go \
          services/atlas-monsters/atlas.com/monsters/monster/processor.go \
          services/atlas-monsters/atlas.com/monsters/monster/processor_test.go \
          services/atlas-monsters/atlas.com/monsters/monster/resource.go
  git commit -m "feat(atlas-monsters): GetInFieldRect query for AoE skill targeting"
  ```

---

## Task B: atlas-channel — rect-query client

Wrap the new atlas-monsters endpoint in a channel-side client method so
the Doom handler (Task D) can call it without knowing the REST URL.

**Files**:
- Modify: `services/atlas-channel/atlas.com/channel/monster/processor.go`
- (Possibly) Modify: `services/atlas-channel/atlas.com/channel/monster/requests.go` if there's a separate file for HTTP client wrappers.

- [ ] **Step 1**: Find the existing `GetById` and `GetInField` (if it
  exists) methods on `monster.Processor` to mirror their style:

  ```bash
  grep -n "func (p \*Processor) GetById\|func (p \*Processor) GetInField" services/atlas-channel/atlas.com/channel/monster/processor.go
  ```

- [ ] **Step 2**: Add the new client method. Signature:

  ```go
  func (p *Processor) GetInFieldRect(f field.Model, x1, y1, x2, y2 int16, limit uint32) ([]Model, error)
  ```

  Body: build the URL by formatting the existing monster-listing path
  with the rect query string; issue the GET; transform the response.
  Use the existing JSON:API decoding helper that `GetById` uses.

- [ ] **Step 3**: Build atlas-channel.

  ```bash
  ( cd services/atlas-channel/atlas.com/channel && go build ./... )
  ```
  Expected: success.

- [ ] **Step 4**: Commit.

  ```bash
  git add services/atlas-channel/atlas.com/channel/monster/processor.go
  git commit -m "feat(atlas-channel): GetInFieldRect client wrapper for atlas-monsters rect query"
  ```

---

## Task C: atlas-channel — generic `itemConsume` charge in `UseSkill`

Plumb `e.ItemConsume()` consumption into the existing `UseSkill` cost
block. Generic across all `itemConsume` skills (Doom, Mystic Door,
summons, mists). The previous Doom-specific placement in
`processAttack`'s cost gate was reverted in Task R.

**Files**:
- Modify: `services/atlas-channel/atlas.com/channel/skill/handler/common.go`
- Add: `services/atlas-channel/atlas.com/channel/skill/handler/common_test.go`

- [ ] **Step 1**: Extend the cost block.

  In `common.go`, after the existing `if e.MPConsume() > 0 { ... }`
  block (around lines 25-27), add:

  ```go
  if itemId := e.ItemConsume(); itemId > 0 {
      invType, typeOk := inventoryconst.TypeFromItemId(itemconst.Id(itemId))
      if a, found := character.NewProcessor(l, ctx).GetById(character.NewProcessor(l, ctx).InventoryDecorator)(characterId).Inventory().CompartmentByType(invType).FindFirstByItemId(itemId); typeOk && found {
          _ = consumable.NewProcessor(l, ctx).RequestItemConsume(f, charcon.Id(characterId), itemconst.Id(itemId), slot.Position(a.Slot()), 0)
      } else {
          l.Warnf("Character [%d] cast skill [%d] requiring item [%d] but no such item found in inventory; cast permitted (defense-in-depth gate only).", characterId, info.SkillId(), itemId)
      }
  }
  ```

  Important: the inline `character.NewProcessor(l, ctx).GetById(...)`
  invocation above is illustrative shorthand. The realized form should
  match how the file already loads characters (look near
  `applyToParty` or the per-skill handlers in this package for the
  canonical loader pattern). If the existing code uses a single-line
  loader like `cp.GetById()(characterId)`, mirror it. Don't double-call
  `NewProcessor`.

  Required imports (add if missing):
  - `"atlas-channel/character"` (likely already present)
  - `"atlas-channel/consumable"` (new)
  - `inventoryconst "github.com/Chronicle20/atlas/libs/atlas-constants/inventory"`
  - `"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"`
  - `itemconst "github.com/Chronicle20/atlas/libs/atlas-constants/item"`
  - `charcon "github.com/Chronicle20/atlas/libs/atlas-constants/character"`

- [ ] **Step 2**: Build atlas-channel.

  ```bash
  ( cd services/atlas-channel/atlas.com/channel && go build ./... )
  ```
  Expected: success.

- [ ] **Step 3**: Add tests.

  Create `services/atlas-channel/atlas.com/channel/skill/handler/common_test.go`
  with three tests (per design §5.3):
  - `TestUseSkill_ItemConsume_BurnsItem`
  - `TestUseSkill_ItemConsume_LogsWarningOnMissingItem`
  - `TestUseSkill_ItemConsume_ZeroItemConsume_Noop`

  Use a logrus test hook (`logrus/hooks/test.NewNullLogger`) for the
  warning capture. Use the asset / compartment / inventory builders the
  same way the previous (now-removed) `TestFindItemSlotInInventory_*`
  tests did before they were deleted in Task R; the asset builder
  requires a non-zero ID.

  Capturing `RequestItemConsume` calls cleanly requires the consumable
  processor to be testable. The simplest approach is to inject a fake
  for the duration of the test via a package-private hook variable, or
  to verify the Kafka-message-emit side via the existing producer
  fake harness if one exists in this package. If neither is feasible
  with reasonable effort, defer the assertion of the consume call to
  Task F's manual verification and keep the unit test focused on the
  inventory lookup (slot resolution).

- [ ] **Step 4**: Run tests.

  ```bash
  ( cd services/atlas-channel/atlas.com/channel && go test ./skill/handler -count=1 -v )
  ```
  Expected: green.

- [ ] **Step 5**: Commit.

  ```bash
  git add services/atlas-channel/atlas.com/channel/skill/handler/common.go \
          services/atlas-channel/atlas.com/channel/skill/handler/common_test.go
  git commit -m "feat(atlas-channel): generic itemConsume charge in UseSkill cost block"
  ```

---

## Task E: atlas-channel — `effect.Model.MobCount()` and `Prop()` accessors

The Doom handler reads `e.MobCount()` and `e.Prop()`. Verify they exist
and add getters if missing.

**Files**:
- Modify: `services/atlas-channel/atlas.com/channel/data/skill/effect/model.go`
- Modify: `services/atlas-channel/atlas.com/channel/data/skill/effect/rest.go` (only if `mobCount` or `prop` is not already wired into `Extract`)

- [ ] **Step 1**: Confirm whether the accessors exist.

  ```bash
  grep -n "func (m Model) MobCount\|func (m Model) Prop\|mobCount\b\|\bprop\b" services/atlas-channel/atlas.com/channel/data/skill/effect/model.go
  ```

  The struct already has `mobCount uint32` and `prop float64` fields
  (per the v1 read of the file). If the public getters are missing,
  add them:

  ```go
  // MobCount returns the cap on monsters affected by an AoE monster-buff
  // skill (e.g., Priest Doom's 6-mob target ceiling). Zero means "no cap".
  func (m Model) MobCount() uint32 { return m.mobCount }

  // Prop returns the per-target probability gate (0.0 - 1.0) used by
  // monster-buff skills like Doom. Zero means "never apply"; values
  // ≥ 1 mean "always apply".
  func (m Model) Prop() float64 { return m.prop }
  ```

- [ ] **Step 2**: Confirm `rest.go` populates both from `RestModel`.

  The `Extract` function already wires `mobCount: rm.MobCount` (line
  ~117 in rest.go) and `prop: rm.Prop` (line ~125). If either is
  missing, add the assignment.

- [ ] **Step 3**: Build atlas-channel.

  ```bash
  ( cd services/atlas-channel/atlas.com/channel && go build ./data/skill/effect/... )
  ```
  Expected: success.

- [ ] **Step 4**: Commit (if any change was made).

  ```bash
  git add services/atlas-channel/atlas.com/channel/data/skill/effect/model.go \
          services/atlas-channel/atlas.com/channel/data/skill/effect/rest.go
  git commit -m "feat(atlas-channel): expose effect.Model MobCount/Prop for AoE handlers"
  ```

---

## Task D: atlas-channel — Doom per-skill handler

Add the new Doom handler package and register it in the per-skill
registry.

**Files**:
- Add: `services/atlas-channel/atlas.com/channel/skill/handler/doom/bbox.go`
- Add: `services/atlas-channel/atlas.com/channel/skill/handler/doom/bbox_test.go`
- Add: `services/atlas-channel/atlas.com/channel/skill/handler/doom/doom.go`
- Add: `services/atlas-channel/atlas.com/channel/skill/handler/doom/doom_test.go`
- Modify: the file that imports the per-skill packages for side effects (registrations file or `main.go`); see design §2.4.

- [ ] **Step 1**: Pure `calculateBoundingBox` + tests (TDD).

  Write `bbox_test.go` first per design §5.1 (4 cases). Then implement
  `bbox.go`:

  ```go
  package doom

  import (
      "github.com/Chronicle20/atlas/libs/atlas-constants/point"
  )

  // calculateBoundingBox derives the inclusive (x1, y1, x2, y2) target
  // rectangle for a monster-buff skill cast. Mirrors Cosmic
  // StatEffect.calculateBoundingBox at server/StatEffect.java:1206-1218.
  // The returned tuple is not normalized — callers (or downstream
  // queries) must handle either ordering of the corners.
  func calculateBoundingBox(casterX, casterY int16, facingLeft bool, lt, rb point.Model) (x1, y1, x2, y2 int16) {
      if facingLeft {
          x1 = casterX + int16(lt.X())
          y1 = casterY + int16(lt.Y())
          x2 = casterX + int16(rb.X())
          y2 = casterY + int16(rb.Y())
      } else {
          x1 = casterX - int16(rb.X())
          y1 = casterY + int16(lt.Y())
          x2 = casterX - int16(lt.X())
          y2 = casterY + int16(rb.Y())
      }
      return
  }
  ```

  Adjust the `int16(lt.X())` cast to match `point.Model`'s actual
  accessor return type (it may already return `int16`).

  Run:
  ```bash
  ( cd services/atlas-channel/atlas.com/channel && go test ./skill/handler/doom -count=1 -v )
  ```
  Expected: 4 tests PASS.

- [ ] **Step 2**: Doom handler scaffold.

  Create `doom.go`. Use the design §2.4 sketch as a starting point.
  Two important details:
  - The package-private `propRollFunc` variable lets tests inject
    deterministic behavior. Do not export it.
  - `c.Stance() & 1 == 1` for facing-left. If atlas-channel's character
    model has a higher-level accessor (e.g., `c.IsFacingLeft()`), prefer
    it. Search:
    ```bash
    grep -n "FacingLeft\|isFacingLeft\|stance" services/atlas-channel/atlas.com/channel/character/model.go
    ```

- [ ] **Step 3**: Doom handler tests.

  Create `doom_test.go` per design §5.2. Each test injects its own
  `propRollFunc` (via the package-private variable, restored via
  `t.Cleanup`) and a fake `monster.Processor` (for `GetInFieldRect`,
  `GetStatusMirror`, `ApplyStatus`). Mirror the fake-injection pattern
  used in the (now-removed) `damageEntryFakes` if it makes sense; or,
  if the per-skill-handler package convention is different, follow that.

  Run:
  ```bash
  ( cd services/atlas-channel/atlas.com/channel && go test ./skill/handler/doom -count=1 -v )
  ```
  Expected: all PASS.

- [ ] **Step 4**: Wire the package's `init()` to register the handler
  by adding a blank import in the registrations file. Find it:

  ```bash
  grep -rln "_ \"atlas-channel/skill/handler/heal\"\|skill/handler/heal\"" services/atlas-channel/atlas.com/channel/
  ```

  Add the equivalent line for `_ "atlas-channel/skill/handler/doom"`
  in the same file.

- [ ] **Step 5**: Build and test full atlas-channel.

  ```bash
  ( cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./... )
  ```
  Expected: green.

- [ ] **Step 6**: Commit.

  ```bash
  git add services/atlas-channel/atlas.com/channel/skill/handler/doom/ \
          services/atlas-channel/atlas.com/channel/skill/handler/registrations/  # or wherever the blank import lives
  git commit -m "feat(atlas-channel): per-skill Doom handler with server-side bbox mob selection"
  ```

---

## Task F: Cross-service build, full test, manual verification handoff

**Files**: None modified.

- [ ] **Step 1**: Build and test all three affected services.

  ```bash
  ( cd services/atlas-monsters/atlas.com/monsters && go build ./... && go test ./... )
  ( cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./... )
  ( cd services/atlas-data/atlas.com/data && go build ./... && go test ./... )
  ```

  Expected: all green.

- [ ] **Step 2**: Verify the in-repo grep for the per-apply Doom log
  line is unique (it should still hit
  `services/atlas-channel/atlas.com/channel/monster/processor.go:73`):

  ```bash
  grep -rn '"Doom: caster=\[' services/
  ```

  Expected: one hit.

  And the per-cast summary log line is unique to the new handler:

  ```bash
  grep -rn '"Doom: caster=\[%d\] level=' services/
  ```

  Expected: one hit, in `services/atlas-channel/atlas.com/channel/skill/handler/doom/doom.go`.

- [ ] **Step 3**: Surface the manual end-to-end checklist (PRD §10).

  Tell the user:

  > Code changes complete. Manual verification needed against a
  > running channel + monsters + data stack:
  >
  > 1. Cast Doom on a regular mob in range. Confirm it renders as a
  >    snail in the v83 client for the duration and resumes its
  >    original sprite at expiry. Confirm one Magic Rock was consumed.
  > 2. Cast Doom on a fire-immune mob. Confirm DOOM still applies.
  > 3. Cast Doom on a boss. Confirm DOOM does not apply, and that
  >    atlas-monsters logs `Monster [..] is a boss. Status rejected.`.
  >    Confirm one Magic Rock was still consumed (the cast itself
  >    succeeded; only the per-target apply was rejected).
  > 4. Cast Doom on a group of mobs spread across the map. Confirm
  >    only mobs in the rectangle (≤ 6 of them) receive DOOM.
  > 5. Cast Doom against a magic-reflect mob. Confirm it is excluded
  >    from the apply and atlas-channel logs
  >    `Doom: monster [..] has MAGICAL reflect; status apply skipped.`.
  > 6. With zero Magic Rock, cast Doom (via a third-party client or
  >    by manipulating inventory). Confirm the warning
  >    `Character [..] cast skill [..] requiring item [..] but no
  >    such item found in inventory; cast permitted (defense-in-depth
  >    gate only).` appears, and the cast still runs.
  > 7. Confirm the per-cast summary line appears once per cast
  >    (`Doom: caster=[..] level=[..] mobsInRect=[..] applied=[..] reflectSkipped=[..] propSkipped=[..]`).
  > 8. Spot-check a non-Doom `itemConsume` skill (e.g., a summon).
  >    Confirm the required item is consumed once per cast.

  If observability MCPs are available, follow `reference_observability.md`
  to pull the relevant log lines from Loki rather than tailing pods.

- [ ] **Step 4**: No commit needed for this step.

---

## Self-review (run by writer; results inline)

**Spec coverage** (PRD §10 acceptance criteria):

- AC 1–3 (single mob, group, fire-immune): covered by Task D handler +
  Task A rect query + atlas-monsters' existing immunity short-circuit.
- AC 5 (boss): covered by atlas-monsters' existing
  `isBossAllowedStatus`. The cast itself charges Magic Rock; only the
  per-target apply is rejected. PRD §10 AC 5 was updated to reflect
  this realized behavior.
- AC 6 (magic-reflect): covered by the per-mob reflect probe in the
  Doom handler.
- AC 7 (no Magic Rock): covered by Task C's `itemConsume` cost block
  with the missing-item warning.
- AC 8 (per-apply log): the existing Debugf at
  `monster/processor.go:73` (kept from v1) plus the new per-cast
  summary line.

**Type consistency**:

- `e.ItemConsume() uint32`, `itemconst.Id(uint32)`, `slot.Position(int16)`,
  `charcon.Id(uint32)` — same casts used in `character_item_use.go:22`.
- `field.Model` is the same type used everywhere.
- `monster.GetStatusMirror().GetReflect(t, uniqueId, kind string) (ReflectInfo, bool)` —
  same signature used by `processDamageInfoEntry`.

**Notable design deviations called out inline**:

- AC 5 (boss): per-target rejection vs. cast-level rejection. The
  cast charges its `itemConsume` cost regardless of how many per-target
  applies succeed (matches Cosmic and matches HP/MP semantics for
  resisted casts in general). The PRD acceptance criterion 5 reflects
  this.
- `mobCount` ordering: atlas-monsters returns the N-closest in-rect
  mobs to the rect center; Cosmic returns the first N in registry
  iteration order. Functionally equivalent; ours is more deterministic.
- The `prop` per-mob probability is honored. The PRD goals section
  enumerates this; the design and tests reflect the deterministic
  injection seam.

If the implementer hits a realized-API mismatch (e.g., a `point.Model`
accessor returns a different type than expected, or the per-skill
registrations file isn't where the design assumes), the surrounding
step text directs them to grep for the actual symbol and adjust in
lockstep. No silent renames.

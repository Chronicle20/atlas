# Review: Task 18b-A — hand-write `Transform` for six forgotten `atlas-channel` packages

**Commit reviewed:** `56298756d` (`775e6aacf..56298756d`)
**Scope:** `buddylist/buddy`, `character`, `data/consumable`, `data/equipment`, `data/quest`, `minigame` — `rest.go` + `rest_test.go` in each, plus `docs/tasks/task-263-backend-guideline-conformance/handwork-notes.md`.

## Method

Read `git show 56298756d` per file; cross-checked every `Transform` field assignment against the
`model.go` and `RestModel` struct definitions for each of the six packages (not the working tree —
per instructions, since the tree is dirty with unrelated 18b-B work). Ran
`go build ./... && go test ./buddylist/buddy/... ./character/... ./data/consumable/... ./data/equipment/... ./data/quest/... ./minigame/...`
from `services/atlas-channel/atlas.com/channel` — all packages `ok`.

## Findings by package

### `buddylist/buddy` — PASS
- `RestModel` has 6 fields, `Transform` (rest.go) assigns all 6 by exact name/type match against
  `Model`'s unexported fields (`characterId`, `group`, `characterName`, `channelId`, `inShop`,
  `pending`). `Model.listId` (`model.go:6`, `uuid.UUID`) has no `RestModel` counterpart and correctly
  isn't referenced — not a dropped field, since `RestModel` never had a slot for it.
- No `Id` field on `RestModel`; `GetID()` derives the JSON:API resource id from `CharacterId`, which
  is mapped. Correctly reasoned as "no Id-mapping question" in the report.
- Bool pair `InShop`/`Pending` <-> `inShop`/`pending`: exact name match, unambiguous, correct.
- Test (`rest_test.go`) uses `reflect.DeepEqual(m, m2)` for a full `Model` round trip after asserting
  `rm2.CharacterId`. Real, per-field coverage.

### `character` — CHANGES REQUIRED (test honesty)
- Field mapping itself is **correct**: all 31 `RestModel` fields (`character/rest.go:12-41`) map
  1:1 by name/type to `Model`'s unexported fields (`character/rest.go:86-119` in the diff). Verified
  specifically that `SpawnPoint: m.spawnPoint` (the raw unexported field) is used, NOT
  `m.SpawnPoint()` — confirmed `Model.SpawnPoint()` (`character/model.go:240-242`) is a hardcoded
  `return 0` stub, so calling it would have silently zeroed the field. The implementer avoided this
  correctly.
- **Blocking:** `character/rest_test.go` (new file, 78 lines) does not use `reflect.DeepEqual` for
  the round trip, unlike all five sibling packages in this same commit. It asserts only
  `rm2.Id`, `m2.Id()`, `m2.Name()`, `m2.Level()`, `m2.Meso()` — 5 of 31 fields. A `Transform` that
  silently dropped any of the other 26 fields (`AccountId`, `WorldId`, `Experience`,
  `GachaponExperience`, `Strength`, `Dexterity`, `Intelligence`, `Luck`, `Hp`, `MaxHp`, `Mp`, `MaxMp`,
  `HpMpUsed`, `JobId`, `SkinColor`, `Gender`, `Fame`, `Hair`, `Face`, `Ap`, `Sp`, `SpawnPoint`, `Gm`,
  `X`, `Y`, `Fh`, `Stance`) would still pass this test — exactly the "smoke test that would pass
  against a half-empty `RestModel`" failure mode the brief calls out. The `SpawnPoint` case is the
  concrete instance: the test fixture sets `SpawnPoint: 0` (`character/rest_test.go:33`), which is
  indistinguishable from the buggy-getter value (`0`), so this test would not have caught the exact
  defect the brief warned about even if the implementer had made it. Fix: assert
  `reflect.DeepEqual(m, m2)` as the other five packages in this commit do — that compares the
  unexported struct fields directly and does not depend on `Model`'s (partially stubbed) getters, so
  it would correctly validate `spawnPoint` regardless of `SpawnPoint()`'s stub status.

### `data/consumable` — PASS
- `RestModel{Id, Spec, Npc, Script, RunOnPickup}` all map to `Model{id, spec, npc, script,
  runOnPickup}` by exact name/type (`data/consumable/rest.go` diff).
- `map[SpecType]int32` collection: `Transform` allocates `make(map[SpecType]int32, len(m.spec))` and
  copies key/value pairs — not aliased. Test actively mutates `rm2.Spec[SpecTypeHP]` and asserts
  `m.GetSpec(SpecTypeHP)` is unaffected (`data/consumable/rest_test.go:39-45`) — a real
  non-aliasing proof, not just a type check.
- Full `reflect.DeepEqual(m, m2)` round trip after the mutation is restored.

### `data/equipment` — PASS
- `RestModel{Id, PetAbilities, NotExtend}` map to `Model{id, petAbilities, notExtend}` by exact
  name/type.
- `[]string` collection: `Transform` allocates `make([]string, len(m.petAbilities))` + `copy` — not
  aliased. Test mutates `rm2.PetAbilities[0]` and asserts `m.PetAbilities()[0]` (the pre-existing
  getter, which does alias — but that's an existing `Model` accessor, not the new `Transform`) is
  unaffected — proves the `Transform` copy is real.
- Full `reflect.DeepEqual(m, m2)` round trip.

### `data/quest` — PASS
- Top-level `RestModel` (18 scalar/struct fields) and nested `RequirementsRestModel` (22 fields),
  `ActionsRestModel` (10 fields), and their sub-structs all map 1:1 by name/type, checked against
  `data/quest/model.go`. Bool fields `AutoStart`/`AutoPreComplete`/`AutoComplete`/`SelectedMob` and
  nested `NormalAutoStart` pair to `Model`'s identically-named unexported fields — unambiguous by
  name, no cross-wiring found.
- Slice handling: `jobs`/`fieldEnter`/`pet`/`dayOfWeek` copied via `copyUint16Slice`/
  `copyUint32Slice`/`copyStringSlice` helpers that explicitly preserve `nil` vs allocated-empty
  (`data/quest/rest.go`, "preserve the nil-vs-empty distinction" comment) — correct given
  `reflect.DeepEqual` distinguishes `nil` from `[]T{}`. `quests`/`items`/`mobs` and the
  `ActionsModel` reward slices are rebuilt element-by-element into freshly `make()`'d backing
  arrays inside `transformRequirements`/`transformActions` — not aliased.
- `transformRequirements`/`transformActions` read nested unexported fields on `RequirementsModel`/
  `ActionsModel` directly (`m.npcId`, `q.id`, `item.id`, etc.) — same-package private-field access,
  consistent with D1 (these are not exported-getter calls on the `Model` parameter itself).
- Test (`data/quest/rest_test.go`) populates every field on `StartRequirements`/`StartActions` and a
  partial (`NpcId`-only) `EndRequirements`/`EndActions` to exercise the nil-vs-empty slice path, then
  asserts full `reflect.DeepEqual(m, m2)`. Real per-field coverage including the nested structs.

### `minigame` — PASS
- `RestModel{Id, OwnerId, RoomType, Title, Private, HasPassword, PieceType, Occupancy, InProgress}`
  map 1:1 by name/type to `Model`'s unexported fields.
- Bool triple `Private`/`HasPassword`/`InProgress` pairs to `private`/`hasPassword`/`inProgress` by
  exact name — unambiguous.
- Full `reflect.DeepEqual(m, m2)` round trip after asserting `rm2.Id`.

## D1 compliance (direct unexported-field access, no getters on the `Model` parameter)

Checked every `Transform`/`transformRequirements`/`transformActions` body: all read `m.<field>` (or,
in `data/quest`, nested `q.id`/`item.id`/`mob.id`/`skill.id` on the sub-model parameters passed into
the helpers) — never `m.<ExportedGetter>()` on the `Model`/`*Model` parameter itself. No violation
found in any of the six packages.

## `Id` mapping

All five packages with a `RestModel.Id` field (`character`, `data/consumable`, `data/equipment`,
`data/quest`, `minigame`) map it to `m.id` and assert `rm2.Id` in their test. `buddylist/buddy` has
no `Id` field — `GetID()` derives the resource id from `CharacterId`, which is mapped and asserted.
No package leaves a resource id unset.

## Not evaluable

- Pre-existing `Extract` functions (not touched by this diff) were read only as needed to confirm
  field-type compatibility for the round trip; a defect in `Extract` itself (e.g. `data/quest`'s
  `Extract`-side aliasing of `rm.Jobs`/`rm.FieldEnter`/`rm.Pet`/`rm.DayOfWeek` directly into the
  `Model`, visible at `data/quest/rest.go` in the pre-existing `extractRequirements`) is outside this
  commit's diff and outside this task's brief (which scopes only `Transform`), so it is not scored
  here.

## Summary

Five of six packages (`buddylist/buddy`, `data/consumable`, `data/equipment`, `data/quest`,
`minigame`) are correct in both mapping and test rigor. `character`'s `Transform` mapping is also
correct, but its test is a partial-field smoke test inconsistent with the pattern established by
every other package in this same commit, and concretely fails to guard the one field
(`SpawnPoint`) where a real hazard (a hardcoded-0 getter) existed in this package.

---

```
verdict: CHANGES_REQUIRED
artifact: docs/tasks/task-263-backend-guideline-conformance/review-task-18b-A.md
scope_confirmed: reviewed commit 56298756d only (all 6 rest.go/rest_test.go pairs + handwork-notes.md); did not touch the dirty working tree
blocking: 1
  - services/atlas-channel/atlas.com/channel/character/rest_test.go:1-78 — TestTransformRoundTrip asserts only 5 of 31 RestModel fields (Id, Name, Level, Meso) instead of `reflect.DeepEqual(m, m2)` like the other five packages in this commit; would pass against a half-empty RestModel and specifically would not have caught a SpawnPoint mapping regression (the fixture value 0 is indistinguishable from the known-stub getter's return value). Replace the four ad hoc field checks with `reflect.DeepEqual(m, m2)`.
non_blocking: 0
not_evaluable: 1
```

# Task 16 batch `login-monsters-npcconv-qagg` — review

## Scope

Commits reviewed: `de782187a` (atlas-login), `127f007` (atlas-monsters), `04af499`
(atlas-npc-conversations), `8bdcf00` (atlas-query-aggregator). Brief:
`.superpowers/sdd/plan/task-16-brief-login-monsters-npcconv-qagg.md`. Implementer
report: `.superpowers/sdd/plan/task-16-login-monsters-npcconv-qagg-report.md`.

All 8 packages named in the brief were addressed: `atlas-login/character`,
`atlas-login/guild`, `atlas-monsters/character/buff`,
`atlas-monsters/monster/information`, `atlas-npc-conversations/pet`,
`atlas-npc-conversations/saved_location`, `atlas-query-aggregator/guild`,
`atlas-query-aggregator/pet`.

## Charge 1 — lossy round-trip carve-out (atlas-login/character)

Read `services/atlas-login/atlas.com/login/character/model.go` (Model, Builder)
and `rest.go` (`Extract` at line 129, `Transform` at line 97) directly.

- Confirmed `Extract`'s builder chain (`rest.go:130-157`) calls exactly 26
  setters: `SetId, SetAccountId, SetWorldId, SetName, SetLevel, SetExperience,
  SetGachaponExperience, SetStrength, SetDexterity, SetIntelligence, SetLuck,
  SetHp, SetMp, SetMaxHp, SetMaxMp, SetMeso, SetHpMpUsed, SetJobId,
  SetSkinColor, SetGender, SetFame, SetHair, SetFace, SetAp, SetSp, SetGm`. It
  never calls `SetSpawnPoint, SetPets, SetEquipment, SetInventory, SetRank,
  SetRankMove, SetJobRank, SetJobRankMove` — matches the implementer's claim
  exactly, and matches `model.go:44` (`spawnPoint`), `:47`-`:53` (`pets`,
  `equipment`, `inventory`, `rank`, `rankMove`, `jobRank`, `jobRankMove`).
- `rest_test.go:56-133` asserts exactly those same 26 fields, one `if` block
  per field, matching the 26 setters `Extract` calls 1:1. No round-trippable
  field is missing from the assertion list; no non-round-tripping field is
  wrongly included.
- `Transform` (`rest.go:97-127`) additionally emits `SpawnPoint: m.spawnPoint`
  into `RestModel`, but since `Extract` never reads `RestModel.SpawnPoint`
  back into `Model`, `spawnPoint` still does not round-trip through
  `Extract(Transform(m))` — correctly excluded from the 26-field assertion.
  This is "fidelity for a future consumer of the REST payload," not a round
  trip claim, and it does not misrepresent the test.
- `RestModel.X`, `.Y`, `.Stance` (`rest.go:40-42`) are confirmed unread by
  `Extract` and unset by `Transform` — verified by grep of both function
  bodies.
- **PASS** — premise verified, not just the conclusion.

## Charge 2 — field coverage, derived independently per package

Read each package's own `model.go` and cross-checked `Transform`/test:

| Package | Model fields | Transform covers | Notes |
|---|---|---|---|
| `atlas-login/character` | 34 (26 round-trip + 8 non-wired) | 26 asserted (+ spawnPoint emitted, non-round-trip) | see Charge 1 |
| `atlas-login/guild` | id, leaderId, members (3) | all 3 | lossless |
| `atlas-monsters/character/buff` | sourceId, expiresAt (2) | both | lossless |
| `atlas-monsters/monster/information` | 17 fields (hp, mp, boss, undead, friendly, firstAttack, weaponAttack, dropPeriod, resistances, animationTimes, skills, revives, banish, attacks, selfDestruction, hpRecovery, mpRecovery) | all 17 | `selfDestruction.present` correctly re-derived by `Extract`, not written by `Transform` (no such `RestModel` field) |
| `atlas-npc-conversations/pet` | id, templateId, name, level, slot (5) | all 5 | lossless |
| `atlas-npc-conversations/saved_location` | characterId, locationType, mapId, portalId (4) | all 4 | lossless |
| `atlas-query-aggregator/guild` | 13 fields (id, worldId, name, notice, points, capacity, logo, logoColor, logoBackground, logoBackgroundColor, leaderId, members, titles) | all 13 | full parity |
| `atlas-query-aggregator/pet` | id, slot, templateId, closeness (4) | all 4 | lossless |

**Twin-pair diff, `guild`:** `atlas-login/guild/model.go` Model has only 3
fields (`id`, `leaderId`, `members`); `atlas-query-aggregator/guild/model.go`
Model has 13 fields (`id, worldId, name, notice, points, capacity, logo,
logoColor, logoBackground, logoBackgroundColor, leaderId, members, titles`).
Genuinely different shapes despite the identical `RestModel` (both have the
full 13-field `RestModel`) — `atlas-login`'s `Extract` simply discards 10 of
13 `RestModel` fields, `atlas-query-aggregator`'s reads all 13. `Transform`
in each package was derived from its own `Extract`/`model.go`, not
cross-copied — confirmed no field mismatch in either.

**Twin-pair diff, `pet`:** `atlas-npc-conversations/pet/model.go` Model is
`{id, templateId, name, level, slot}`; `atlas-query-aggregator/pet/model.go`
Model is `{id, slot, templateId, closeness}` — disjoint field sets (`name`,
`level` vs `closeness`), confirming these are not interchangeable packages.
Both `Transform`s were independently correct against their own `model.go`.

**PASS** — all 8 packages independently verified; no cross-package field
borrowing found.

## Charge 3 — commit contents match declared scope

```
git show --stat de782187a   # only services/atlas-login/... + handwork-notes.md (declared, legitimate)
git show --stat 127f007     # only services/atlas-monsters/...
git show --stat 04af499     # only services/atlas-npc-conversations/...
git show --stat 8bdcf00     # only services/atlas-query-aggregator/...
```
Confirmed via `git show --stat` on all four commits: each touches only its
own service's `rest.go`/`rest_test.go` pairs, no unintended files. The one
non-service file (`docs/.../handwork-notes.md` in `de782187a`) is the
declared, legitimate carve-out documentation. **PASS**.

## Charge 4 — faithfulness to `Extract`, not the JSON surface

Verified per package above (Charge 2 table) that `Transform` only populates
`RestModel` fields `Extract` actually reads, with the single legitimate
exception of `atlas-login/character`'s `SpawnPoint` (documented, does not
falsely inflate the round-trip claim — see Charge 1). `RestModel.X`, `.Y`,
`.Stance` in `atlas-login/character` confirmed unread/unset as claimed.
**PASS**.

## Charge 5 — no behavior change outside the addition

For all 8 touched `rest.go` files, `git show <commit> -- <path> | grep '^-' | grep -v '^---'`
returned **zero deletion lines** in every case — the diffs are 100% additive
(new `Transform` function only). No `Extract` body changed, no `Build()`
validation rule touched, no `RestModel` field added. **PASS**.

## Charge 6 — no file overwritten

`git show --stat` confirms both pre-existing test files
(`atlas-monsters/monster/information/rest_test.go`,
`atlas-npc-conversations/pet/rest_test.go`) show partial diffs (+46/-1 and
+22/-1 respectively), not full-file rewrites; inspected the actual hunks —
each pre-existing test (`TestExtract_PopulatesAttacks`,
`TestExtract_PopulatesRecoveryFields`, `TestExtractFirstAttack`,
`TestBuilderSetFirstAttack` in monsters; `TestExtractPopulatesName` in
npc-conversations) is untouched; the single `-` line in each case is an
import-statement reformat (single import -> import block), not a lost test.
**PASS**.

## Charge 7 — tests actually constrain the code

Mutation test performed: in
`services/atlas-query-aggregator/atlas.com/query-aggregator/pet/rest.go`,
changed `Closeness: m.closeness` to `Closeness: 0` in `Transform`. Re-ran
`go test ./... -run TestTransformRoundTrip -v`:

```
rest_test.go:22: round trip mismatch. Expected {id:7 slot:3 templateId:5000029 closeness:250}, got {id:7 slot:3 templateId:5000029 closeness:0}
--- FAIL: TestTransformRoundTrip (0.00s)
```

Reverted with `git checkout -- rest.go`; `git status --short` on that file
shows clean. **PASS** — the test genuinely constrains `Transform`'s output.

## Module gates

Re-ran independently (not relying on implementer's report):
- `atlas-login`: `go build ./... && go vet ./...` clean; `go test ./character/... ./guild/...` — all `ok`.
- `atlas-monsters`: clean build/vet; `go test ./character/buff/... ./monster/information/...` — all `ok`.
- `atlas-npc-conversations`: clean build/vet; `go test ./pet/... ./saved_location/...` — all `ok`.
- `atlas-query-aggregator`: clean build/vet; `go test ./guild/... ./pet/...` — all `ok`.

## Findings

None blocking. None non-blocking.

## Not evaluable

None — all charges fully evaluable within the four-commit diff plus the
`model.go` files the diffs depend on (per brief, these were explicitly
read-only reference files for the round-trip field inventory).

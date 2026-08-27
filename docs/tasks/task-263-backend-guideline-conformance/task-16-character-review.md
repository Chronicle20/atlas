# Task 16 review — batch `character`

Scope: commits `1cf400a29` (feat(atlas-character): add Transform and round-trip
tests) and `f2a89d447` (feat(atlas-consumables): add Transform and round-trip
tests). Brief: `.superpowers/sdd/plan/task-16-brief-character.md`. Five
packages: `atlas-character/configuration`, `atlas-character/data/portal`,
`atlas-character/data/skill`, `atlas-character/data/skill/effect`,
`atlas-consumables/portal`.

## 1. Commit contents match declared scope

`git show --stat 1cf400a29`:
```
services/atlas-character/atlas.com/character/configuration/rest.go        |  8 ++
services/atlas-character/atlas.com/character/configuration/rest_test.go   | 16 ++++
services/atlas-character/atlas.com/character/data/portal/rest.go          | 15 ++++
services/atlas-character/atlas.com/character/data/portal/rest_test.go     | 31 ++++++++
services/atlas-character/atlas.com/character/data/skill/effect/rest.go    | 65 +++...
services/atlas-character/atlas.com/character/data/skill/effect/rest_test.go | 91 +++...
services/atlas-character/atlas.com/character/data/skill/rest.go           | 17 ++++
services/atlas-character/atlas.com/character/data/skill/rest_test.go      | 37 +++++++++
8 files changed, 280 insertions(+)
```
`git show --stat f2a89d447`:
```
services/atlas-consumables/atlas.com/consumables/portal/rest.go       | 15 +++++++++++
services/atlas-consumables/atlas.com/consumables/portal/rest_test.go  | 31 ++++++++++++++++++++++
2 files changed, 46 insertions(+)
```
Every file is under the two declared services, exactly matches the brief's file
list, and every hunk is a pure addition (0 deletions in both commits). PASS.
The implementer's self-reported git-state confusion (interleaved staging,
misattributed `handwork-notes.md` commit) left no trace in either commit —
both are exactly the declared scope.

## 2. Field coverage, per package (derived independently, not by analogy)

- **`configuration`** (`services/atlas-character/atlas.com/character/configuration/model.go`):
  `Model` has one field, `pendingExpiry time.Duration`. `Transform`
  (`rest.go:38-42`) sets `PendingExpiryHours: int(m.PendingExpiry().Hours())`.
  Matches `Extract`'s inverse (`rest.go:29-31`, which builds
  `DefaultConfig().WithPendingExpiry(time.Duration(r.PendingExpiryHours) * time.Hour)`).
  PASS.
- **`atlas-character/data/portal`** (`model.go`): 8 fields (`id`, `name`,
  `target`, `portalType`, `x`, `y`, `targetMapId`, `scriptName`). `Transform`
  (`rest.go:53-64`) covers all 8, rendering `id` via `strconv.Itoa`, matching
  `Extract`'s `strconv.Atoi` inverse. PASS.
- **`atlas-character/data/skill`** (`model.go`): 5 fields (`id`, `action`,
  `element`, `animationTime`, `effects []effect.Model`). `Transform`
  (`rest.go:53-67`) covers all 5, correctly delegating the nested slice to
  `effect.Transform` via `model.SliceMap`, mirroring `Extract`'s use of
  `effect.Extract`. PASS.
- **`atlas-character/data/skill/effect`** (`model.go`): 51 scalar/slice/map
  fields plus `statups []statup.Model`. `Transform` (`rest.go:132-192`) sets
  all 51 `RestModel` fields it can, delegates `statups` to `statup.Transform`,
  and deliberately omits `CardStats` (see §3 below). Verified field-by-field
  against both `model.go`'s field list and `RestModel`'s field list — every
  field `Extract` reads has a `Transform` counterpart, no field is dropped
  save the documented one. PASS.
- **`atlas-consumables/portal`** (`model.go`): 8 fields, structurally
  identical shape to `atlas-character/data/portal` but read from its own
  `model.go` — confirmed independently the two files' `Model` definitions are
  byte-for-byte the same package-local declaration (no cross-service
  cut/paste artifact left behind, e.g. no stray `_map` alias mismatch,
  `atlas-constants` import path is the same real import in both). `Transform`
  (`rest.go:38-49`) covers all 8. PASS.

No sibling-package analogy accepted as evidence; each package's own
`model.go`/`rest.go` was read and diffed against its own `Transform`.

## 3. Faithfulness to `Extract`, not the JSON surface — the `CardStats` claim

`services/atlas-character/atlas.com/character/data/skill/effect/rest.go:64`
declares `RestModel.CardStats cardItemUp`. `Extract` (`rest.go:67-127`)
constructs the `Model` literal from `rm.WeaponAttack` through
`rm.MonsterStatus` (lines 74-125) — `rm.CardStats` does not appear anywhere in
that body, and `Model` (`model.go`) has no `cardStats`/`CardStats` field to
receive it. The implementer's claim in `handwork-notes.md:34` is accurate:
`Transform` (`rest.go:132-192`) correctly does not populate `CardStats`. PASS
— verified against the actual `Extract` body, not accepted on the
implementer's word.

## 4. `configuration`'s signature

`Extract(r RestModel) Model` (`configuration/rest.go:29`) — single return
value, no `error`. The brief classified this as "return does not have two
results." `Transform(m Model) RestModel` (`rest.go:38`) matches: single
return, no `error` added. PASS — the implementer did not widen the signature.

## 5. No behavior change outside the addition

Both commits are purely additive per `--stat` (0 deletions, only new lines).
No `Extract` body, `Build()` validation rule, or `RestModel` field set is
touched — confirmed the diffs contain no modified line inside any pre-existing
function; every changed hunk is a new `Transform` function plus a new test
file. FR-17 and PRD §5 constraints hold. PASS.

## 6. No file overwritten

`--stat` for both commits shows only new files (`rest_test.go` in all five
packages, none pre-existing) and additive edits to `rest.go` (no deletions in
any hunk). No pre-existing test was destroyed. PASS.

## 7. Tests actually constrain the code (mutation test)

Performed an independent mutation on
`atlas-character/data/skill/effect/rest.go`: changed
`MonsterStatus: m.monsterStatus,` to `MonsterStatus: nil,` inside `Transform`.
Re-ran `go test ./data/skill/effect/... -run TestTransformRoundTrip -v`:
```
--- FAIL: TestTransformRoundTrip (0.00s)
    rest_test.go:89: round trip mismatch. ... got {... monsterStatus:map[]}
```
Test fails as expected, at field level (`reflect.DeepEqual` diff isolates
`monsterStatus`). Reverted with `git checkout --
services/atlas-character/atlas.com/character/data/skill/effect/rest.go` and
confirmed via `git status --short` that only the pre-existing
concurrent-agent files remain modified/untracked (none under
`atlas-character` or `atlas-consumables`); `handwork-notes.md` is the expected
`M` shared file. PASS.

## Full-module gate

`go build ./... && go vet ./... && go test ./...` passed clean for both
`services/atlas-character/atlas.com/character` and
`services/atlas-consumables/atlas.com/consumables` (all packages `ok` or `no
test files`, no failures).

## Not evaluable

None. All seven charges were fully evaluable from the two-commit diff plus
each package's own `model.go`.

## Verdict

All seven charges pass with cited evidence. No blocking or non-blocking
findings.

# Task 16 Review — messages-storage batch

Scope: commits `8167f428d` (feat(atlas-messages): add Transform and round-trip tests) and
`1d5205164` (feat(atlas-storage): add Transform and round-trip tests).

Brief: `.superpowers/sdd/plan/task-16-brief-messages-storage.md`
Implementer report: `.superpowers/sdd/plan/task-16-messages-storage-report.md`

## 1. Commit contents match declared scope

`git show --stat 8167f428d` — all 8 changed paths are under
`services/atlas-messages/atlas.com/messages/{data/skill,data/skill/effect,pet,rate}`. No
`atlas-summons` or other-service paths present.

`git show --stat 1d5205164` — all 6 changed paths are under
`services/atlas-storage/atlas.com/storage/data/{consumable,etc,setup}`. No leaked paths.

`git log --oneline c3ad6b3c4..HEAD -- services/atlas-summons` shows
`2bac98f26 feat(atlas-summons): add Transform and round-trip tests` — that service's own commit
is present and intact, confirming the implementer's self-reported unstage (`git restore --staged
services/atlas-summons`) worked and nothing was lost or leaked in either direction.

**PASS.**

## 2 & 3. Field coverage and faithfulness to Extract (per package)

### atlas-messages/data/skill/effect (`rest.go:129-181`)
`Model` (effect/model.go:8-59) has 51 unexported fields including `statups []statup.Model` and
`monsterStatus map[string]uint32`. `Transform` maps every one of them 1:1, including the nested
`SliceMap(statup.Transform)` call for `statups` (`rest.go:130`). `RestModel` additionally declares
`CardStats cardItemUp` (`rest.go:64`), which `Extract` never reads (`rest.go:67-127`) — `Transform`
correctly leaves `CardStats` unset (zero value), matching charge 3's requirement not to populate
fields `Extract` never consumes. Verified independently against this package's own `model.go`, not
borrowed from a sibling.

### atlas-messages/data/skill (`rest.go:36-50`)
`Model` (skill/model.go:7-13): `id, name, action, element, animationTime, effects`. `Transform`
covers all six, delegating `effects` through `model.SliceMap(effect.Transform)`. Matches `Extract`
(`rest.go:52-65`) exactly in reverse.

### atlas-messages/pet (`rest.go:50-56`)
`Model` (pet/model.go:4-8): `id, slot, name`. `Transform` covers all three, matching `Extract`
(`rest.go:46-48`), which itself only reads `Id, Slot, Name` via `NewModel`. `RestModel` has extra
unread fields (`CashId`, `TemplateId`, etc.) that neither `Extract` nor `Transform` touch — correct
since `Extract` never reads them either (pre-existing, untouched by this diff).

### atlas-messages/rate (`rest.go:35-53`)
`Model` (rate/model.go:23-30): `characterId, expRate, mesoRate, itemDropRate, questExpRate,
factors []Factor`. `Transform` covers all six, converting `characterId` back to the string `Id`
via `strconv.FormatUint` (inverse of `Extract`'s `strconv.ParseUint` at `rest.go:65`), and maps each
`Factor` to `FactorRestModel`. Matches `Extract` (`rest.go:55-75`).

### atlas-storage/data/consumable (`rest.go:24-30`)
`Model` (consumable/model.go:4-8): `id, slotMax, rechargeable`. `Transform` covers all three,
inverse of `Extract` (`rest.go:32-43`). Verified independently — this package's own `model.go`,
not the look-alike `etc`/`setup` siblings.

### atlas-storage/data/etc (`rest.go:23-28`)
`Model` (etc/model.go:4-7): `id, slotMax`. `Transform` covers both, inverse of `Extract`
(`rest.go:30-40`).

### atlas-storage/data/setup (`rest.go:23-28`)
`Model` (setup/model.go:4-7): `id, slotMax`. `Transform` covers both, inverse of `Extract`
(`rest.go:30-40`).

All seven `Transform` field lists were derived and checked against each package's own `model.go`
independently; none showed cross-package field borrowing (the failure mode this batch was flagged
for, given the `etc`/`setup`/`consumable` and `skill`/`skill/effect` look-alikes).

**PASS — all 7 packages.**

## 4. No behavior change outside the addition

`git show 8167f428d` and `git show 1d5205164` are purely additive: 287 and 104 insertions
respectively, 0 deletions (the only `^-` lines in each `git show` are the diff header `---` marker
lines, confirmed via `grep -c "^-"` = 8 and 6, matching file counts exactly, no content removed).
`Extract` bodies, `Build()` (n/a — none of these seven packages have a `Build()` method), and
`RestModel` field sets are untouched in both commits.

**PASS.**

## 5. No file overwritten

`--stat` output for both commits shows only new files (`rest_test.go`, all newly created) and
additive diffs to existing `rest.go` files (insertions only, confirmed above). No deletions, no
file replaced.

**PASS.**

## 6. Tests actually constrain the code

Reviewed `TestTransformRoundTrip` in all 7 `rest_test.go` files — each constructs a `Model` (or
uses `NewModel`) with distinct, non-zero values per field, round-trips through `Transform` then
`Extract`, and asserts `reflect.DeepEqual`. This is a genuine per-field constraint, not a
coincidental pass.

Independent mutation performed on `atlas-storage/data/consumable/rest.go`: changed
`Rechargeable: m.rechargeable,` to `Rechargeable: false,` in `Transform`. Ran `go test ./...`:

```
--- FAIL: TestTransformRoundTrip (0.00s)
    rest_test.go:26: round trip mismatch.
        Expected {id:123 slotMax:99 rechargeable:true}
        Got {id:123 slotMax:99 rechargeable:false}
FAIL
```

Reverted with `git checkout -- rest.go`; `git status --short` confirmed `consumable/rest.go` no
longer appears (remaining modified/untracked files belong to other concurrent agents' in-progress
work in this shared worktree, unrelated to this change).

Full test suites also run clean:
```
ok  atlas-messages/data/skill
ok  atlas-messages/data/skill/effect
ok  atlas-messages/pet
ok  atlas-messages/rate
ok  atlas-storage/data/consumable
ok  atlas-storage/data/etc
ok  atlas-storage/data/setup
```

**PASS.**

## Not evaluable

None — full review surface (both commits, all 7 packages' `Extract`/`Transform`/`model.go`) was
covered directly.

## Verdict

All six charges pass with direct evidence. No blocking or non-blocking findings.

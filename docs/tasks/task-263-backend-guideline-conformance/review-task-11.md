# Review: Task 11 — apply `transform` codemod to `atlas-channel`

Commit: `ed54e564e` (range `2146a4ab4..ed54e564e`)
Reviewer surface: the diff of this commit, plus the `Extract`/`Model` definitions in each
touched package (needed to judge whether `Transform` is the true inverse). The generator
itself (`docs/tasks/task-263-backend-guideline-conformance/codemod/transform.go`) was reviewed
and approved in Task 10 and is out of scope here, except to confirm it was not modified.

## 1. Generator not modified

```
git diff --stat 2146a4ab4..ed54e564e -- docs/tasks/task-263-backend-guideline-conformance/codemod/
```
produced no output. Only `services/atlas-channel/**` and the new ledger file changed. Task 10's
dry-run tally (76 APPLIED / 18 SKIPPED repo-wide) is not invalidated. **PASS.**

## 2. Transform is the true inverse of Extract (R1 check)

Read all 15 applied packages' `rest.go` files field-by-field against their `Extract`:
`chair`, `chalkboard`, `character/buff/stat`, `character/skill`, `data/monster`, `data/npc`,
`data/skill/effect/statup`, `drop`, `guild/member`, `guild/thread/reply`, `guild/title`,
`kite`, `mount`, `mts/holding`, `mts/transaction`.

Particular attention to same-typed-sibling risk:

- `drop/rest.go:54-92` — `X`/`Y` and `DropperX`/`DropperY` are both `int16`. Extract maps
  `x: rm.X, y: rm.Y, dropperX: rm.DropperX, dropperY: rm.DropperY`; Transform maps the same
  pairs back correctly by name, not position. **PASS.**
- `guild/member/rest.go:13-35` — `Level`, `Title`, `AllianceTitle` are all `byte` (a 3-way
  same-type collision). Extract/Transform pair each field with its correctly-named sibling.
  **PASS.**
- `mts/holding/rest.go:34-56` — the RestModel `WorldId` is declared `byte`; `Model.worldId`
  is `world.Id`. Extract does `world.Id(r.WorldId)`; Transform does `byte(m.worldId)` — the
  conversion target type is correctly drawn from the RestModel field's declared type (`byte`),
  not from `world.Id`'s underlying type by assumption. **PASS.**
- `character/skill/rest.go:35-53` — `RestModel.Id` is `uint32`; `Model.id` is `skill.Id`.
  Extract: `skill.Id(rm.Id)`; Transform: `uint32(m.id)`. Correct direction and type. **PASS.**
- `chair`, `chalkboard`, `character/buff/stat`, `data/skill/effect/statup`, `guild/title`,
  `guild/thread/reply`, `mount`, `mts/transaction`, `data/monster`, `data/npc`, `kite` — no
  same-typed-sibling fields beyond the ones above; each field name in `Extract` has an
  exact-name counterpart in `Transform`. **PASS** for all 15.

No case found where a field was mapped to the wrong same-typed sibling.

## 3. Round-trip tests actually discriminate

Read `chair`, `guild/member`, `drop`, `mts/holding`, `mts/transaction`, `data/monster`
`TestTransformRoundTrip` bodies directly (not just their PASS status). Values are distinct,
non-zero, sequential (11, 22, 33, ...) across every field of every struct, including multiple
`byte`/`int16` fields in the same struct (`drop`: `dropType:66`, `x:77`, `y:88`, `dropperX:143`,
`dropperY:154`; `guild/member`: `level:44`, `title:55`, `allianceTitle:77`). A field-swap bug
(e.g. `x`↔`dropperX`) would fail `reflect.DeepEqual` under these fixtures because the values
differ pairwise. **PASS** — tests genuinely discriminate, not zero-value-passing.

Independently re-ran (not trusted from the report):
```
cd services/atlas-channel/atlas.com/channel && go build ./... && go vet ./...
go test ./... -run TestTransformRoundTrip -v
```
Build and vet clean; 15 `--- PASS: TestTransformRoundTrip` lines, 0 `FAIL` lines. Matches the
report's Step 2/3 claims. **PASS.**

## 4. FR-17 — appends only

```
git diff -U0 2146a4ab4..ed54e564e -- services/atlas-channel | grep '^-' | grep -v '^---'
```
returns exactly two lines: `-import "testing"` (twice). Read both hunks directly:

- `data/monster/rest_test.go` — single-line `import "testing"` replaced by a grouped
  `import ("reflect"; "testing")` block (full diff hunk inspected: `testing` reappears inside
  the new group, zero functional lines removed).
- `mts/holding/rest_test.go` — identical pattern.
- `mts/transaction/rest_test.go` — already had a grouped import block; `reflect` was a pure
  insertion, no removed line at all (confirmed the implementer's claim; this file contributes
  zero lines to the `grep '^-'` output, consistent with what was found).

No other removed line exists anywhere in the 31-file diff. **PASS**, matches my own
independently-run `git diff -U0`, confirming the report's read.

## 5. Skip legitimacy (8 skips)

Ledger (`ledger-transform-channel.tsv`) has 23 rows: 15 APPLIED + 8 SKIPPED, matching the
report's histogram exactly — no reconciliation discrepancy found (the report's own histogram
text is worded confusingly — "8 packages named" vs "4+3+1" — but 4+3+1 = 8 and the ledger's 8
SKIPPED rows are exactly: `buddylist/buddy`, `character`, `data/consumable`, `data/equipment`,
`data/quest`, `minigame`, `mts/listing`, `mts/wish`. No discrepancy.

Spot-checked (source-verified, not trusted from the report):

- `buddylist/buddy` — `rest.go:12-13` has `InShop bool` and `Pending bool`; both are consumed
  into `Model` (`Extract` maps `inShop: rm.InShop, pending: rm.Pending`). Genuine 2-bool skip.
  **Legitimate.**
- `data/quest` — `rest.go` has 5 bool fields (`AutoStart`, `AutoPreComplete`, `AutoComplete`,
  `SelectedMob`, `NormalAutoStart`). **Legitimate** (well over the 2-bool threshold).
- `minigame` — `rest.go:13,14,17` has `Private`, `HasPassword`, `InProgress`, all consumed
  into `Model` per `Extract`. **Legitimate.**
- `data/equipment` — `rest.go:7` `PetAbilities []string`. **Legitimate** unsupported-type skip.
- `data/consumable` — `rest.go:16` `Spec map[SpecType]int32`. **Legitimate.**
- `mts/listing` — `rest.go:62` `EndsAt *time.Time`. **Legitimate.**
- `mts/wish` — `rest.go:19` `ExpiresAt *time.Time`. **Legitimate.**
- `character` — `rest.go` has 4 `byte` fields (`Level`, `SkinColor`, `Gender`, `Stance`); the
  reported failure is the generator's sequential test-value counter overflowing `byte` (value
  330 > 255) on one of them. This is the byte-range-overflow class Task 10's review already
  traced, and the controller's ruling explicitly anticipated this exact package skipping.
  **Legitimate**, and correctly not hand-patched.

All 8 skips confirmed legitimate against actual source, not merely trusted from the report.

## Not evaluable

- The remaining 27 non-scope tier-A/B/C `atlas-channel` packages (of 50 total) were not
  examined — out of this task's `-only services/atlas-channel` tier-A filter and correctly
  excluded from the ledger.
- Did not independently re-derive the codemod's internal field-matching algorithm (e.g. why
  it emits `byte(m.worldId)` rather than some other conversion) beyond confirming its output is
  correct on every applied package — that mechanism was Task 10's review surface, not this
  one's.

## Verdict

No blocking defects found. Generator untouched, all 15 `Transform` functions are correct
inverses of their `Extract` (including every same-typed-sibling and typed-conversion case
checked), all 15 round-trip tests genuinely discriminate, FR-17 holds exactly as reported, and
all 8 skips are legitimate against source. Independent re-run of build/vet/test matches the
report's claims.

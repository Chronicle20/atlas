# Task 15 review — atlas-inventory `data/consumable` (Transform/TransformReward)

Commit reviewed: `056cd3b5863f9467aa83c0b30427984637deb034` (range `4f82f105d..056cd3b`)
Brief: `.superpowers/sdd/plan/task-15-brief-atlas-inventory.md`
Files changed: `services/atlas-inventory/atlas.com/inventory/data/consumable/rest.go` (+88),
`services/atlas-inventory/atlas.com/inventory/data/consumable/rest_test.go` (+110, new file)

## 1. Copied-template risk — VERIFIED, not carried over

Compared `atlas-inventory`'s `RewardModel` (`model.go:202-206`: `itemId`, `count`, `prob` only)
against `atlas-consumables`' `RewardModel` (`services/atlas-consumables/.../data/consumable/model.go:257-264`:
`itemId`, `count`, `prob`, **`effect`, `worldMsg`, `period`**). The template package genuinely has
three more fields.

`TransformReward` in this commit (`rest.go:254-260`) emits exactly `ItemId`, `Count`, `Prob` —
matching this package's own `RewardModel` and `RewardRestModel` (`rest.go:248-252`, which also has
only three fields). No `effect`/`worldMsg`/`period` field was carried over from the consumables
template. Claim in the implementer's report confirmed field-by-field.

## 2. Transform/Extract and TransformReward/ExtractReward are true inverses

Read `Transform` (`rest.go:92-170`) against `Extract` (`rest.go:172-246`) field by field: every one
of the 55 top-level `RestModel` fields plus `MonsterSummons`/`Rewards` slice transforms maps
symmetrically in both directions (e.g. `TargetBlock`↔`tragetBlock`, the pre-existing misspelling,
is preserved consistently in both functions rather than "fixed" divergently). `TransformReward`
(`rest.go:254-260`) and `ExtractReward` (`rest.go:262-268`) map the same three fields both ways.
No field is silently dropped or invented in either direction — confirmed by the live-mutation test
below rather than by inspection alone.

## 3. Test honesty — live mutations performed, not taken on faith

Ran independently (not from the implementer's transcript):

- Mutated `rest.go:113` `Price: m.price,` → `Price: m.price + 1,`. `go test -run
  TestTransformRoundTrip -v` failed subtest `Transform` with `got.price:151` vs `want.price:150` —
  a genuine field-level diff, not a blanket failure. Reverted with a line-anchored `sed -i
  '113s/.../.../'` (not a blind pattern sed); `git diff --exit-code` on `rest.go` came back clean
  afterward.
- Mutated `rest.go:257` `Count: m.count,` → `Count: m.count + 1,`. Both subtests failed
  (`Transform` because `Reward` is nested inside the top-level round trip; `Reward` directly) with
  `count:3` vs `count:2` — again a specific field-level diff. Reverted the same way; `git diff
  --exit-code` clean.

Both subtests are genuinely load-bearing, not tautological.

## 4. `Extract*` outside `rest.go`

`grep -rn '^func Extract' services/atlas-inventory/atlas.com/inventory/data/consumable/` returns
only `Extract` and `ExtractReward`, both in `rest.go`. No delegated `Extract*` in a sibling file
(`model.go`, `processor.go`, `requests.go`) was missed. Package pair table is complete.

## 5. Fixtures — real identities

`rest_test.go`'s `Transform` fixture (lines 10-72) sets every bool `true`, every numeric field a
distinct non-zero value, non-empty strings, and non-empty maps/slices (`spec`, `monsterSummons`,
`morphs`, `skills`, `rewards`). No pointer-returning `Extract*` in this pair, so no nil-deref risk.
The `Reward` fixture (lines 90-94) is likewise fully non-zero and non-default. Round-trip assertion
via `reflect.DeepEqual` against the original `Model`/`RewardModel` is a strict identity check, not
a default-value match.

## 6. Scope discipline

`git show --stat 056cd3b` touches only the two files listed above, both under
`services/atlas-inventory`. No accessor methods were minted — `rest_test.go` is `package
consumable` and constructs `Model`/`RewardModel` via unexported field literals directly (per repo
convention D1), not through new exported accessors (confirmed: no new `func (m Model) ...` or
`func (m RewardModel) ...` lines in the commit). No `docs/` file is part of this commit (the
modified/untracked `docs/tasks/task-263-.../progress.md`, `agent-ledger.tsv`, and sibling review
files visible in `git status` are unrelated concurrent worktree activity, not part of commit
`056cd3b`). `atlas-consumables` and `atlas-npc-shops` packages are untouched by this commit.

## Not evaluable

None — the full review surface (both files in the commit, plus the `atlas-consumables` template
package read for the field-diff check) was directly inspected and exercised.

## Verdict

APPROVED. No blocking or non-blocking findings.

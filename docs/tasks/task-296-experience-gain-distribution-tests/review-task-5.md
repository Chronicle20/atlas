# Review: Task 5 — Mutation checks (task-296)

Commit range: `bdfb6c239..c4fc10ab7` (single commit `c4fc10ab7`)
Brief: `.superpowers/sdd/plan/task-5-brief.md`
Report: `.superpowers/sdd/plan/task-5-report.md`

## Scope

Task 5 is a proof-only task: apply three temporary local mutations, capture
failure output, revert, and commit only `progress.md`. Reviewed the commit
diff, the committed `progress.md`, and cross-checked its claims against the
current (unmutated) source of the two files it says were mutated.

## Findings

### 1. Commit contains only `progress.md` — PASS

`git diff --stat bdfb6c239..c4fc10ab7` shows exactly one file changed:
`docs/tasks/task-296-experience-gain-distribution-tests/progress.md`
(233 insertions). `git diff bdfb6c239..c4fc10ab7 -- services/atlas-channel/atlas.com/channel/kafka/message/character/kafka.go`
and the same for `kafka/consumer/character/consumer.go` both produce empty
output — neither mutated file carries any residual diff against the parent
commit. `git diff bdfb6c239..c4fc10ab7 --name-only` confirms the single-file
diff. No leftover mutation.

### 2. Captured output is genuine, not paraphrased — PASS

Cross-checked every specific claim in `progress.md` against the real,
unmutated test/source files:

- `consumer_test.go:392` — `t.Errorf("config mismatch\n got: %+v\nwant: %+v", ...)`
  matches the `TestBuildIncreaseExperienceConfig/Item_EquipItemBonusExpNotPrimary`
  and `.../PrimaryPlusBonuses_Accumulate` failure lines in `progress.md`
  verbatim (`grep -n` confirms line 392 is this exact statement).
- `consumer_test.go:413` — `t.Errorf("distribution type %q is in AllExperienceDistributionTypes but has no case in distributionMappingCases", dt)`
  matches the Step 1 captured output exactly, including line number.
- `consumer_test.go:419` — `t.Errorf("case %q covers distribution type %q, which is not in AllExperienceDistributionTypes", name, dt)`
  matches the Step 2 captured output exactly, including line number.
- The `Item_EquipItemBonusExpNotPrimary` test case (`consumer_test.go:291-297`)
  gives `ExperienceDistributionTypeItem, Amount: 7000` and
  `want: model2.IncreaseExperienceConfig{ItemBonusEXP: 7000}` — i.e. `Amount`
  defaults to 0 in `want`. The current (baseline) ITEM case at
  `consumer.go:410-417` only writes `c.ItemBonusEXP = int32(d.Amount)`, no
  `c.Amount` write — consistent with the brief's described mutation (adding
  `c.Amount = int32(d.Amount)`) producing `got.Amount:7000` against
  `want.Amount:0`, exactly as captured.
- The `PrimaryPlusBonuses_Accumulate` case (`consumer_test.go:342-348`) gives
  WHITE Amount 2500 then ITEM Amount 770; the captured failure
  (`got.Amount:770` vs `want.Amount:2500`) is exactly what the described
  mutation would produce (the ITEM branch overwriting `c.Amount` after WHITE
  set it).
- `kafka.go:158/182` confirms `ExperienceDistributionTypeCakePie = "CAKE_PIE"`
  is a real registry entry, consistent with Step 2's mutation (removing it)
  and the captured `"CAKE_PIE"` failure message.

All captured output is internally consistent with the real, currently-checked-
in source and test line numbers — this is genuine tool output, not a
reconstruction.

### 3. Working tree cleanliness — PASS

`git status --porcelain` at review time shows only the five pre-existing
untracked review/ledger artifacts (`agent-ledger.tsv`,
`review-task-1.md` … `review-task-4.md`), none of which this task touches or
is expected to touch. `progress.md`'s own Step 4 capture shows the same set
minus `review-task-5.md` (which did not exist yet at capture time) — no
mutated source file appears in either listing.

### 4. Step 4 clean/green confirmation present — PASS

`progress.md` records `git status --porcelain` output (matching the pattern
above) and `go test ./...` from `services/atlas-channel/atlas.com/channel`
exiting 0, with a captured log tail showing `ok`/`[no test files]` for every
remaining package and no `FAIL` line.

## Not evaluable

- The actual mutation-and-run sequence was not independently reproduced by
  this review (the brief and repo conventions direct the reviewer not to
  re-run `tools/verify.sh`, and re-running the mutations independently is
  outside the review's scope/budget). Genuineness was assessed via line-number
  and message cross-checks against the real source, which is strong
  circumstantial evidence but not a live re-run.

## Verdict

All four brief requirements are met: no leftover mutation, output is
internally consistent with real source (not a paraphrase), and the tree is
confirmed clean and green. No blocking findings.

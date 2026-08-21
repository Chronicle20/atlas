# Review: Task 7 — `atlas-channel` GM level semantics

Commit range: `e09eee916..98ec7fd7f` (single commit `98ec7fd7f8cb69699e026400675ae6947f6ae3ca`)
Brief: `.superpowers/sdd/plan/task-7-brief.md`

## Scope confirmed

`git diff --stat e09eee916..98ec7fd7f` shows exactly 4 files changed, all under
`services/atlas-channel/atlas.com/channel`:

- `character/builder_test.go` (+24)
- `character/model.go` (+12/-1)
- `session/model.go` (+6)
- `session/model_test.go` (new, +19)

This matches the brief's file list exactly. No other service, and no other file
in `atlas-channel`, was touched.

## Findings

### PASS — `character.Model.Gm()` widened to `m.gm > 0`, `GmLevel()` added

`services/atlas-channel/atlas.com/channel/character/model.go:69-77`:

```go
func (m Model) Gm() bool {
	return m.gm > 0
}

func (m Model) GmLevel() int {
	return m.gm
}
```

Matches brief Step 3 verbatim, including doc comments.

### PASS — `session.Model.Gm()` getter added next to `setGm`

`services/atlas-channel/atlas.com/channel/session/model.go` adds:

```go
func (s *Model) Gm() bool {
	return s.gm
}
```

immediately above the existing `setGm(gm bool) Model`. Matches brief Step 4.

### PASS — Scope containment: no other service touched

Verified via `grep -rln "gm == 1" services/` that the pattern still exists,
unmodified, in atlas-npc-shops, atlas-cashshop, atlas-login, atlas-pets,
atlas-consumables, atlas-messages, and atlas-query-aggregator (plus
atlas-rankings' processor_test.go), and confirmed `git diff --stat` for this
commit range touches none of them. This matches the plan's Global Constraint
that atlas-cashshop (and, by extension, every other service) is untouched on
this branch. `git status --porcelain` shows unrelated uncommitted changes in
`docs/packets/`, `libs/atlas-packet/` — these belong to the concurrent Task 9
implementer and are outside this review's range/scope, per the reviewer brief;
they are not part of commit `98ec7fd7f`.

### PASS — Three non-test callers of `Gm()` behave correctly under the widened predicate

- `kafka/consumer/session/consumer.go:212`: `s = sp.SetGm(s.SessionId(), c.Gm())`
  — passes the boolean straight through at session bootstrap. Widening from
  `== 1` to `> 0` only makes this more correct (a level-2+ GM's session now
  correctly carries `gm = true`), no behavioral regression.
- `kafka/consumer/message/consumer.go:99`: `c.Gm()` passed into
  `showGeneralChatForSession` for GM chat colouring — "is this player a GM" is
  exactly what `> 0` answers.
- `kafka/consumer/message/consumer.go:178`: `c.Gm()` passed into
  `NewWhisperReceive` — same "is a GM" boolean usage.

All three are consistent with the brief's characterization ("all want 'is this
player a GM', for which `> 0` is strictly more correct").

### PASS — New tests are non-vacuous

`TestGm_IsTrueForEveryLevelAboveZero`
(`character/builder_test.go`, appended after line 254) includes the case
`{2, true}`. Under the old predicate `m.gm == 1`, level 2 evaluates to
`2 == 1` → `false`, which mismatches `want = true`, so this test would fail
against the pre-change code. Confirmed by inspection of the removed line
(`git show` diff: `- return m.gm == 1`) — the test is not vacuous.

`TestGm_Accessor` (`session/model_test.go`, new file) exercises a getter that
did not previously exist (`s.Gm()` would not compile pre-change) — this test
is a compile-fail-then-pass test, also non-vacuous.

Both suites run and pass:

```
=== RUN   TestGm_IsTrueForEveryLevelAboveZero
--- PASS: TestGm_IsTrueForEveryLevelAboveZero (0.00s)
ok  	atlas-channel/character	(cached)
=== RUN   TestGm_Accessor
--- PASS: TestGm_Accessor (0.00s)
ok  	atlas-channel/session	(cached)
```

### PASS — `character/builder.go` unmodified

Not present in the commit's changed-file list; `SetGm(v int) *modelBuilder`
already existed at `builder.go:137` prior to this commit, confirmed by reading
the file (unchanged by the diff).

### Note (non-blocking) — builder_test.go uses `MustBuild()`, not `NewModelBuilder().Build()`

The brief's Step 1 pseudo-test used `.Build()` and told the implementer to
"replace `NewModelBuilder()` with the actual constructor". The implementer
correctly adapted this to the repo's actual API
(`character.NewModelBuilder().SetId(1).SetGm(c.level).MustBuild()`), which is
exactly what the brief asked for. Not a defect.

## Not evaluable

None. The full unit (diff + the three named caller sites + builder.go) was
read and verified in scope.

## Verdict

APPROVED. Requirement-by-requirement match to the brief, correct GM-level
semantics, non-vacuous tests, no scope leakage into other services, and all
downstream callers behave correctly under the widened predicate.

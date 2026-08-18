# Review: task-239-plan-lint-f5-symbols

## Scope

Requested range was `394ad8cf1..HEAD` (three commits: 394ad8cf1, be2bed596,
bf78ebcfb). `394ad8cf1` is not an ancestor common with `main` in the sense of
being excluded background — it is itself the tip of `main` (`main` = `6fb58bdc5`
is 394ad8cf1's parent), i.e. **394ad8cf1 is the F5 feature commit itself** ("F5
— flag symbols a plan's Go blocks call that resolve nowhere"). The two-dot range
`394ad8cf1..HEAD` therefore *excludes* the main risk item (the new F5 shell
logic) and only shows the doc-comment trim to it plus the plan-task.md/trim
commits. I reviewed the full `main..HEAD` diff (all three commits, matching the
task's own commit-list and the stated review priorities, which explicitly ask
about F5 shell correctness that only exists in 394ad8cf1) rather than the
literal two-dot range, since the literal range would have made the primary
review priority impossible to evaluate. Flagging this as a scope note, not a
defect in the change.

Diff reviewed: `git diff main..HEAD` —
`tools/plan-lint.sh`, `.claude/commands/plan-task.md`,
`.claude/commands/execute-task.md`, `docs/review-protocol.md`,
`.claude/agents/atlas-implementer.md`,
`.claude/agents/packet-completeness-critic.md`,
`.claude/agents/packet-verifier.md`, `.claude/commands/bringup-version.md`.

## F5 shell correctness (tools/plan-lint.sh)

All checks below run against the actual working tree at `bf78ebcfb` (HEAD of
the branch), executed with `bash tools/plan-lint.sh ...`.

- **Tempfile handling / cleanup**
  - `--no-symbols` (`run_symbols=0`): the entire F5 block is skipped, no
    `/tmp/plan-lint-*.$$` files created. Verified: `--no-symbols` on
    `docs/tasks/task-222-item-expiration-extenders/plan.md` → clean, exit 0,
    no F5 line printed.
  - Plan with no ` ```go ` blocks: `/tmp/plan-lint-go.$$` is created (possibly
    empty), the `[ -s ... ]` guard skips the indexing pipeline, and
    `rm -f /tmp/plan-lint-go.$$` at line 258 runs unconditionally afterward.
    Verified with a plan containing no go fences: no `/tmp/plan-lint-*` files
    left behind after the run.
  - Plan with go blocks: all five secondary tempfiles
    (`idx/vocab/known/sel/unk`) are created and removed together at
    `plan-lint.sh:255-256`, inside the `[ -s ... ]` branch that created them.
    Verified: no leftover files after a normal run with two F5 warnings.
  - Unclosed ` ```go ` fence (plan runs off the end of the file without a
    closing fence): `awk`'s `ing` flag just stays 1 to EOF, everything after
    the opening fence is treated as Go — no script error, no hang. Verified
    with a synthetic unclosed-fence plan; F5 warned correctly and exited 0.

- **`while read` loop is redirect, not pipe** — `plan-lint.sh:251` uses
  `done < /tmp/plan-lint-unk.$$`, matching the existing F2/F3 pattern and its
  own inline comment ("Redirect, not a pipe — same subshell trap as F2's
  loop.", `plan-lint.sh:250`). `f5warns` increments correctly outside any
  subshell — confirmed by the live run producing `f5warns=2` and the
  corresponding footer text firing.

- **`comm -23` sorted-input requirement** — both inputs
  (`/tmp/plan-lint-sel.$$` at line 246-247, `/tmp/plan-lint-known.$$` at line
  245) are built via `sort -u`, so the precondition holds.

- **`f4warns`/`f5warns` counters** — every `warn "F4 ...` and `warn "F5 ...`
  call site increments its dedicated counter first (`plan-lint.sh:105,108,253`).
  `warn "F3 skipped..."` (line 189) has no dedicated counter, but that call
  site predates this change and is out of scope. The footer prints the F4
  block only if `f4warns>0` and the F5 block only if `f5warns>0`
  (`plan-lint.sh:267,272`), and both are additive to the pre-existing generic
  `warnings` total used for the top-line count and the clean/non-clean branch
  — verified this is unchanged and correct via a live run (2 warnings, F5
  footer only, "0 error(s), 2 warning(s)", exit 0).

- **`set -u`** — script declares `set -u` (`plan-lint.sh:39`) and no
  `set -e`. `f4warns`/`f5warns` are initialized at declaration
  (`plan-lint.sh:64`) before any use, so no unbound-variable failures.
  Absence of `set -e` means a non-fatal `grep`/`sed` failure (e.g., no
  matches) would not abort the script even without the `|| true` fallbacks
  the author added anyway — belt and suspenders, not a bug.

- **`grep -r` over the tree cannot fail the script** — every `grep -r` in the
  F5 block is followed by `|| true` (`plan-lint.sh:224,234,243,247,249`),
  and `set -e` is not in effect anyway, so a "no matches" (`grep` exit 1) or
  transient I/O error cannot abort the run.

- **`--exclude-dir=.worktrees` given `$root` is a worktree when run inside
  one** — `$root` is `git rev-parse --show-toplevel` (`plan-lint.sh:56`).
  From the main repo this is `.../atlas`, and `--exclude-dir=.worktrees`
  correctly prevents `grep -r` from re-walking every checked-out worktree
  under `.worktrees/` (confirmed present: `git worktree list` shows 4+
  worktrees under `.../atlas/.worktrees/`). From inside a worktree,
  `git rev-parse --show-toplevel` returns that worktree's own root (e.g.
  `.../atlas/.worktrees/task-239.../`), which normally has no nested
  `.worktrees` directory, so the exclude is a correct no-op there — it does
  not (and should not) exclude the worktree's own tree from indexing.

- **`-h`/`--help` hard-coded line range** — `sed -n '28,33p' "$0"`
  (`plan-lint.sh:47`). Verified live: `bash tools/plan-lint.sh -h` prints
  exactly the 6-line usage block (lines 25-30 in the file's comment numbering
  before the flag was added shifted to 28-33 after `--no-symbols` was
  inserted into the usage comment) — output matches the current usage
  comment verbatim, including the new `--no-symbols` line. This is a
  self-referential range that will silently drift again if the usage comment
  block ever moves without updating this literal — same latent fragility as
  before this change (pre-existing pattern, not introduced by F5), but worth
  a passing note since F5 is exactly the kind of change that shifted it once
  already.

- **Self-declared vocabulary correctness** — a plan block that both defines
  and calls a helper (`func (h *Helper) DoThing() ...` then `h.DoThing()`)
  resolves clean; verified live, zero F5 warnings.

## Does F5 do what the docs claim?

- `tools/plan-lint.sh docs/tasks/task-222-item-expiration-extenders/plan.md --no-commands`
  → exactly 2 F5 warnings, `NewItemUseItemTag` and `SetPayload`, exit 0.
  Matches the review brief exactly.
- Same plan with `--no-symbols` → clean, exit 0. Matches.
- `-h` → prints the usage block including `--no-symbols`. Matches.
- Timing: `time bash tools/plan-lint.sh ... --no-commands` → 4.83s wall.
  Matches the plan-task.md claim of "~5s" (`plan-task.md:167`).

## Trim commit (bf78ebcfb) — did it remove any rule, threshold, procedure, path, or command?

Diffed `be2bed596..bf78ebcfb` file by file:

- `docs/review-protocol.md` — removed a two-table "cost audit" section (byte
  counts, percentages, a task-26 anecdote) under "Why this exists", replaced
  by a shorter "Why this shape" paragraph. No rule, threshold, or procedure
  removed — the normative content of the document (verdict vocabulary, return
  shape, artifact requirement) is untouched by this diff (confirmed those
  sections are outside the diff hunk).
- `.claude/commands/execute-task.md` — removed task-227 timing anecdotes
  ("48 of 109 minutes idle", "22 of those 48 idle minutes"), a task-231/232
  measurement aside, and reworded "6.9M tokens, ~24% of one controller
  session" into "never reverse-engineer the selection... ask it." All the
  underlying imperatives survive verbatim or near-verbatim: "do not poll, do
  not wait", "`atlas-reviewer`... never a bare `general-purpose` dispatch",
  "Read the review artifact only when the verdict is not `APPROVED`",
  "the fix commit joins the next gate's range", the ERROR-handling rule. No
  rule content lost, only the "why" evidence.
- `.claude/agents/atlas-implementer.md` — removed a task-227 timing anecdote
  ("6 minutes across five whole-filesystem sweeps... `atlas-rest`"); the rule
  itself ("Never root a `find` at `/`", "run `go list` instead") survives
  verbatim.
- `.claude/agents/packet-completeness-critic.md` — removed two bug-memory
  identifiers (`bug_majorversion_gt83`, `bug_reshift_csv_carryover`) used as
  retrospective examples under a "Why this exists" header, renamed to "Your
  role". The operative sentence ("gate-lint... gate-check... you are the
  SEMANTIC guard — you confirm the diff and the declared scope agree.")
  survives verbatim. These are illustrative bug-class labels, not a rule the
  agent is instructed to look up or act on elsewhere in the file — no
  functional loss found (not exhaustively cross-checked against other files
  in the repo that might reference these labels, since that is outside this
  unit's diff surface; see Not evaluable).
- `.claude/agents/packet-verifier.md` — one-line rewording, no content lost;
  "That playbook owns every rule below" replaces "every rule this agent used
  to restate" — same referent, same list of rules follows unchanged
  (untouched by the diff).
- `.claude/commands/bringup-version.md` — removed a task-113 anecdote
  ("This is the entry point task-113 lacked — it was hand-orchestrated.").
  The instruction itself ("You narrate and delegate... NOT do the whole pass
  inside one monolithic agent.") is untouched.

No normative rule, threshold, path, or command was found removed by this
commit. Everything removed reads as retrospective/measured justification,
consistent with the stated intent.

## plan-task.md (be2bed596) — F5 documentation and Step 5a addition

- F5 table row added at `plan-task.md:150` (was line ~146 before), consistent
  with implementation: selector-call resolution against repo
  definitions/calls/plan's own vocabulary, advisory not blocking, ~5s cost,
  `--no-symbols` escape hatch documented (`plan-task.md:167`). All claims
  verified live above.
- "Test blocks carry the spec, not the scaffolding" rule
  (`plan-task.md:85-118`) is new prose, not mechanically checked by
  `plan-lint.sh` (no F-check enforces it) — it is a Step 5a authoring rule,
  correctly scoped as such and not claimed otherwise in the diff.
- The rule explicitly states it "overrides `superpowers:writing-plans`' 'no
  test code without actual test code' rule, for scaffolding only" — this is
  a plan-authoring policy change worth the next planner being aware of; it is
  documented, scoped ("for scaffolding only"), and paired with a hard
  constraint ("A vague case table or expected value is a plan failure").
  No inconsistency found between this prose and the F5 mechanics.

## Markdown fence balance

Counted `^```` occurrences per file — all seven touched `.md` files have an
even count (fences pair):
`atlas-implementer.md` 2, `packet-completeness-critic.md` 8,
`packet-verifier.md` 0, `bringup-version.md` 0, `execute-task.md` 10,
`plan-task.md` 12, `review-protocol.md` 6. Manually inspected
`plan-task.md`'s fence line numbers (15/17, 73/81, 104/114, 143/145,
177/180, 184/187) — all open/close pairs correctly nested, including the new
"Test blocks carry the spec" example block (73-81) and its inner nested
```` ```markdown ```` fence. The mid-line 4-backtick inline spans
(```` ```go ````) used as inline-code escapes for the literal string ` ```go `
do not start at column 1 and were correctly excluded from the fence count.

## Not evaluable

- Whether the removed bug-memory identifiers
  (`bug_majorversion_gt83`, `bug_reshift_csv_carryover` in
  `packet-completeness-critic.md`) are referenced or looked up elsewhere in
  the repo (e.g. a memory/skill index) is outside this unit's diff surface
  and was not swept.
- Full regression sweep of `plan-lint.sh` against all 263 plans in
  `docs/tasks/` (to confirm the "107 times across 38 plans" F5 selector-rate
  claim in the trimmed comment) was not re-run — out of scope for a shell
  correctness/behavior review of this diff, and the claim is asserted in
  a comment, not tested code.

## Verdict rationale

F5's shell logic is correct on every path exercised (no-symbols, no-go-blocks,
normal warnings, unclosed fence, self-declared vocabulary), matches its own
documentation and the task's example exactly, and the trim commit removed only
retrospective narrative with no loss of rule/threshold/procedure/path/command.
Markdown fences balance. The only issue found is a scope note about the
review's own input range (394ad8cf1 is main's tip, not excluded background),
which does not reflect a defect in the change itself.

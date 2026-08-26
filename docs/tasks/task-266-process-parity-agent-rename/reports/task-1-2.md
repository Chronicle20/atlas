# Task 1 & 2 Report — Process Parity Agent Rename

Worktree: `.worktrees/task-266-process-parity-agent-rename`, branch
`task-266-process-parity-agent-rename`.

## Task 1 — Scripted rename sweep

Commit `b1b64f67c` — "rename atlas-implementer/verifier/reviewer to task-* on
the live surface".

1. `git mv`:
   - `.claude/agents/atlas-implementer.md` → `.claude/agents/task-implementer.md`
   - `.claude/agents/atlas-verifier.md` → `.claude/agents/task-verifier.md`
   - `.claude/agents/atlas-reviewer.md` → `.claude/agents/task-reviewer.md`
2. Applied `atlas-implementer→task-implementer`, `atlas-verifier→task-verifier`,
   `atlas-reviewer→task-reviewer` via `sed -i` to exactly the 17 files named in
   PRD §4.1/§4.2 (3 renamed + 14 edited): the three agent files,
   `.claude/commands/execute-task.md`, `.claude/commands/fix-pr-bug.md`,
   `.claude/hooks/commit-boundary.sh`, `.claude/hooks/turn-budget-guard.sh`,
   `.claude/hooks/turn-budget.sh`, `CLAUDE.md`, `docs/agent-dispatch.md`,
   `docs/post-implementation.md`, `docs/superpowers-integration.md`,
   `docs/review-protocol.md`, `docs/codemod-vs-agents.md`,
   `tools/agent-ledger_test.sh`, `tools/agent-ledger.sh`, `tools/task-step.sh`.
   Result: `git diff --cached --stat` reported 57 insertions/57 deletions
   across those 17 files, matching the plan's expected count exactly.
3. `docs/process-parity.md` was deliberately **excluded** from this sed sweep —
   it is the canonical, cross-repo spec for the rename and is not in the PRD
   §4.2 reference-count table; its "was `atlas-implementer`" annotations and
   the §5.1 rename-mapping prose describe the rename itself and would become
   nonsensical if mechanically substituted (e.g. line 77 would read
   "`task-implementer` (was `task-implementer`)"). Its only edit is the
   authored §7 amendment in Task 2.
4. All touched files were plain LF (`grep -c $'\r'` = 0 for each); no
   line-ending normalization occurred.

### Verification — no behavioral drift in the renamed agent files

Reversed each renamed file's substitutions (all three names at once) and
diffed against the pre-rename blob:

```
task-implementer: exit=0
task-verifier: exit=0
task-reviewer: exit=0
```

`diff` returned nothing for all three — the rename touched only the name, not
the tool lists, budgets, hand-back contracts, or fan-out constraints.

### Verification — live-surface sweep is complete

```
$ git grep -lE 'atlas-(implementer|verifier|reviewer)' | grep -v '^docs/tasks/'
docs/agent-dispatch.md
docs/process-parity.md
```

Both remaining hits are Task 2's own authored, historical-mention prose (the
cutover note and the rename spec's mapping table/§5.1 text) — see Task 2
below and the "Note" at the end of this report.

### Verification — §3.1 portable hooks contain no `atlas-` string

```
.claude/hooks/wait-loop-guard.sh: 0
.claude/hooks/wait-loop-guard_test.sh: 1   <- "kubectl get pods -n atlas-pr-1370" (a namespace example, unrelated to atlas-implementer/verifier/reviewer)
.claude/hooks/block-home-paths-in-docs.sh: 0
.claude/hooks/turn-budget.sh: 0
.claude/hooks/turn-budget-guard.sh: 0
.claude/hooks/fork-dispatch-guard.sh: 0
.claude/hooks/task-num-collision-detector.sh: 0
```

None of the seven hooks reference `atlas-implementer`, `atlas-verifier`, or
`atlas-reviewer`.

## Task 2 — Judgment edits

Commit `64e6b9368` — "narrow process-parity §7 check 3 to the live surface,
add historical-name note and CLAUDE.md route".

1. **`docs/process-parity.md` §7 check 3** — narrowed "anywhere" to the live
   surface (`.claude/`, `CLAUDE.md`, top-level `docs/*.md`, `tools/`),
   explicitly exempting `docs/tasks/` and pointing to `docs/agent-dispatch.md`
   for the cutover.
2. **`docs/agent-dispatch.md`** — added a "Historical-name cutoff" paragraph
   right after the intro, explaining that task-266 renamed the trio and that
   `docs/tasks/` artifacts predating it use the old `atlas-*` names because
   those dispatches actually happened under them.
3. **`CLAUDE.md`** — added a row to `## Where the procedures live` routing
   cross-repository process-parity questions to `docs/process-parity.md`.

### Note on the two remaining `atlas-` hits outside `docs/tasks/`

`git grep -lE 'atlas-(implementer|verifier|reviewer)'` (excluding
`docs/tasks/`) still returns `docs/agent-dispatch.md` and
`docs/process-parity.md` after both commits. Both are intentional and by
design, not a residual sweep gap:

- `docs/agent-dispatch.md` line 8-9 is the historical-cutoff note itself
  (PRD §4.4's explicit ask) — it necessarily names the old names once to
  explain what changed.
- `docs/process-parity.md` retains its "was `atlas-implementer`" table
  annotations (§3.5), its `atlas-implementer` → `task-implementer` mapping
  prose (§5.1), and the "currently hardcode" sentence (§2) — these describe
  the rename decision itself for the three other repositories
  (`home-hub`, `Harbormaster`, `MyFleet`) that have not yet executed it, and
  the file's own §4.2 reference table (57/17-file count) explicitly excludes
  this file from the mechanical sweep. §7 check 3, the only line that
  asserted "no reference to atlas-* anywhere," is now scoped to the live
  surface, which this file is not part of (it's the canonical cross-repo
  spec, not `docs/*.md` in the "repo's own procedures" sense checked by
  check 3's target repos).

If the dispatching controller intended a stricter reading (zero `atlas-*`
byte-string anywhere outside `docs/tasks/`, including in `docs/process-parity.md`
and the cutover note itself), that would require either scrubbing the
historical-name explanation of the very names it explains, or excluding
`docs/process-parity.md` from the grep check by name — flag this back if that
stricter reading was intended; I judged the PRD's own file inventory (which
excludes `docs/process-parity.md` from its 44-reference table) as authoritative
over the top-level dispatch brief's summary bullet.

## Tests run (module-local / task-local gates)

```
$ bash tools/agent-ledger_test.sh
...
agent-ledger_test.sh: all assertions passed
EXIT: 0

$ bash .claude/hooks/wait-loop-guard_test.sh
...
passed: 33  failed: 0
EXIT: 0

$ bash tools/task-numbers_test.sh
...
all task-numbers.sh tests passed
EXIT: 0
```

All three were green at baseline per the plan and remain green after both
commits — no regression.

## Files changed

Task 1 (`b1b64f67c`):
- `.claude/agents/atlas-implementer.md` → `.claude/agents/task-implementer.md` (renamed + 13 in-body refs across the trio)
- `.claude/agents/atlas-verifier.md` → `.claude/agents/task-verifier.md`
- `.claude/agents/atlas-reviewer.md` → `.claude/agents/task-reviewer.md`
- `.claude/commands/execute-task.md`
- `.claude/commands/fix-pr-bug.md`
- `.claude/hooks/commit-boundary.sh`
- `.claude/hooks/turn-budget-guard.sh`
- `.claude/hooks/turn-budget.sh`
- `CLAUDE.md` (1 mechanical rename hunk only)
- `docs/agent-dispatch.md` (3 mechanical rename hunks only)
- `docs/codemod-vs-agents.md`
- `docs/post-implementation.md`
- `docs/review-protocol.md`
- `docs/superpowers-integration.md`
- `tools/agent-ledger.sh`
- `tools/agent-ledger_test.sh`
- `tools/task-step.sh`

Task 2 (`64e6b9368`):
- `docs/process-parity.md` (§7 check 3 amendment)
- `docs/agent-dispatch.md` (historical-name cutoff note)
- `CLAUDE.md` (owner-table row)

## Self-review

- Confirmed the three renamed files' `name:` frontmatter matches their new
  filenames (`task-implementer`, `task-verifier`, `task-reviewer`).
- Confirmed reversing the rename on each of the three agent files yields a
  byte-identical diff against the pre-rename blob — no tool-list, budget, or
  constraint drift.
- Did not touch anything under `docs/tasks/` (verified via `git status` and
  the final `git grep` scoping).
- Did not run `tools/verify.sh`, `tools/lint.sh`, `-race`, or any docker
  command, per the dispatch's DO NOT list.
- Kept Task 1 (mechanical) and Task 2 (authored) as two separate commits by
  hand-splitting the CLAUDE.md and docs/agent-dispatch.md hunks with
  `git apply --cached` against extracted hunk ranges, since both files had one
  hunk from each task.
- Left the pre-existing untracked `docs/tasks/task-266-process-parity-agent-rename/plan.md`
  alone — it predates this dispatch and is not part of either task's scope.

## Issues / concerns

- See "Note on the two remaining atlas- hits" above — flag if the controller
  wanted a stricter (zero-tolerance outside docs/tasks/) reading applied to
  `docs/process-parity.md` and the new cutover note itself.

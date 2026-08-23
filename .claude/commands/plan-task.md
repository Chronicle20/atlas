---
description: Phase 3 — invoke superpowers:writing-plans to produce an implementation plan inside the task worktree
argument-hint: Task identifier — accepts "task-054-effect-duration-units", "task-054", "054", or "54"
---

You are starting Phase 3 of the Atlas four-phase development workflow. Argument: **$ARGUMENTS**

## The cost shape of this phase — read once, it explains every rule below

Measured across all 70 `/plan-task` sessions in this repo's transcript history:
**median 102 turns, median 220k peak context, 204k output tokens/session** —
the highest per-session output of any command here, above `/execute-task`'s
168k. It is not a bytes problem: median tool-result volume is 0.5 MB. It is a
**turn** problem. Each turn replays the whole ~220k window.

Where the turns go, per session: 44.8 Bash calls — 12.8 `cd`, 9.7 `grep`, 8.3
`sed`, 4.1 `cat`, 3.8 `ls`. That is ~39 of 102 turns spent establishing which
files exist, what module they build from, and what to copy — facts that are
mechanical and now come from one script.

The rest: 5.5 Write/Edit ops against `plan.md` for a ~50 KB file (median 91 KB
of payload — ~1.8x re-emission), driven by the write → lint → fix → fix loop.

**Do not economise by writing a smaller plan.** 94% of `plan.md` is `## Task N`
bodies (median 11 tasks x 4.4 KB), and Phase 4 extracts those verbatim as the
implementer's brief. Cutting them moves discovery into the implementer, and
`/execute-task` is already 55% of task-branch spend. Cut turns, not content.

## Process

### Step 1 — Resolve the task

Same as `/design-task` Step 1 — run the resolver, do NOT glob for the folder
yourself:

```sh
tools/task-resolve.sh "$ARGUMENTS"
```

It prints one tab-separated line: `<task-id>\t<task-dir>\t<worktree>`, and
accepts exact, number-only (`54`/`054`/`task-54`/`task-054`), or slug-fragment
identifiers.

**Never glob `.worktrees/*/docs/tasks/task-*`** — every worktree carries a full
copy of `docs/tasks/`, so that pattern returns thousands of mostly-duplicate
paths into context to resolve one ID.

Exit codes: **3** → no match, ask for correction. **4** → ambiguous, the
candidates are on stderr; list them and let the user pick. If `<worktree>` is
the main repo root the task has no worktree — stop and tell the user it needs
one.

### Step 2 — Enter the worktree once, then never `cd` again

Run `pwd`. If it does NOT match `<worktree>`, `cd <worktree>` yourself and
continue from there. Do NOT ask the user to re-run the command — per CLAUDE.md's
"Worktree Discipline" rule, cd into the task worktree yourself.

**That is the only `cd` this phase gets.** Sessions currently average 12.8, and
each one is a whole turn at ~220k context that discovers nothing. Every later
command uses a path relative to the worktree root you are already standing in,
or an absolute path under `<worktree>`. If you catch yourself typing `cd`, you
have lost your place — run `pwd` once and carry on from the answer.

### Step 3 — Validate inputs

1. Confirm both `prd.md` and `design.md` exist. If either is missing, stop and tell the user to complete the prior phase.
2. Confirm `plan.md` does NOT already exist. If it does, ask whether to overwrite.

`tools/plan-context.sh` (Step 4) exits **5** when `design.md` is absent, so this
check and Step 4 collapse into one call in the normal case.

### Step 4 — Load context in two calls, not forty

```sh
tools/task-facts.sh <id>
tools/plan-context.sh <id> --symbols
```

`task-facts.sh` gives branch, head, worktree, existing artifacts, change
surfaces, applicable guards, and toolchain.

`plan-context.sh` gives the code survey this phase used to assemble by hand: for
every path `design.md` names — whether it exists, its line count, its module
root (the `go build`/`go test` cwd), whether it already has a `_test.go`, the
sibling files that are "Patterns to copy:" candidates, and with `--symbols` the
exported declarations of every touched `.go` file. It ran 6.0 KB on task-241 (32
paths) and 8.3 KB on task-232; `--symbols` added 8.4 KB on task-241.

Then read, in one batched tool block:

- `<worktree>/docs/tasks/<id>/prd.md`
- `<worktree>/docs/tasks/<id>/design.md`
- `<worktree>/CLAUDE.md`

Read `docs/superpowers-integration.md` only if you need the fuzzy-resolution or
artifact-location rules; Steps 1–2 already applied them.

**Batch the remainder.** Whatever the survey did not answer, issue as
independent calls in a single tool block rather than one per turn — that is the
difference between 4 turns and 26. If you find yourself running `grep` for the
fourth time, stop: you are rediscovering something `--symbols` already printed.

### Step 4a — Delegate deep discovery when the survey is wide

`plan-context.sh` tells you what exists. It does not tell you how the code
*works*. If reading that yourself would take more than a handful of targeted
calls — the survey lists **more than ~15 paths**, or spans **more than one
service** — dispatch **one** `Explore` agent with `model: sonnet` instead of
reading it inline.

This inverts the usual inline-vs-delegate default, and the arithmetic is why:
the break-even is a ~35k dispatch floor against the ~20 controller turns this
replaces, each replaying ~220k. Below the threshold, read it yourself — a
dispatch to save three calls is a loss.

Dispatch at most one such agent. Give it the `plan-context.sh` output verbatim
(it is the inventory — the agent must not re-derive it) and ask for exactly:

- per file, what it currently does and the seam the design touches
- the concrete `Patterns to copy:` target, as `path:line`
- the existing test's setup shape per touched package, as `path:line-line`
- any signature the design assumes that does not match the code

Do NOT ask it to write plan tasks, size them, or draft `### Files` blocks. It
returns findings; you write the plan. A reviewer never fans out further, and
neither does this.

### Step 5 — Invoke writing-plans

Use the Skill tool to invoke `superpowers:writing-plans`. Pass:

- Spec at `<worktree>/docs/tasks/<id>/design.md` (PRD at `prd.md` for reference).
- Plan output MUST be saved to `<worktree>/docs/tasks/<id>/plan.md`.
- Also produce `<worktree>/docs/tasks/<id>/context.md` summarizing key files, decisions, dependencies.
- Do NOT auto-invoke execution.

Run the `writing-plans` skill's self-review (placeholder scan, type consistency, spec coverage) before saving.

**Write `plan.md` once.** Sessions average 5.5 Write/Edit ops and 91 KB of
payload for a ~50 KB file, because the plan gets written before its Step 5b
defects are known and then patched twice. Both error classes are already
answerable from Step 4: F1 is the survey's EXISTING/UNRESOLVED split, F5 is
`--symbols`. Settle every unresolved path and every invented symbol **before**
the first Write. Append later task sections if the plan is long; do not re-emit
sections you already wrote.

### Step 5a — Atlas plan-task format (required)

Phase 4 extracts each task section verbatim into the implementer's brief
(`tools/task-brief.sh` is an awk slice of the `## Task N` heading and its
body). Whatever you write into a task section IS the brief. Three rules follow
from that.

**Every task carries a `### Files` block.** You are reading the code to write
the plan; the implementer should not have to rediscover what you already
found. Naming the files removes the implementer's discovery phase — the phase
that inflates context before any editing starts.

```markdown
### Files

- `services/atlas-x/atlas.com/x/foo/processor.go` — add the Bar method here
- `services/atlas-x/atlas.com/x/foo/processor_test.go` — new test cases
- `libs/atlas-constants/item/id.go` — read-only; the constant to use

Patterns to copy: `services/atlas-y/atlas.com/y/baz/processor.go:88` (same shape)
```

Paths must be repo-relative and must exist (or be explicitly marked `new
file`). Mark read-only references as such so the implementer does not edit
them. Include the module root the task's `go build`/`go test` runs from when
it is not obvious from the paths — `plan-context.sh`'s "Module roots" section
is that answer.

**Test blocks carry the spec, not the scaffolding.** You have no compiler; the
implementer has one and an adjacent `_test.go` to copy from. In a ```` ```go ````
test block, spell out **in full** everything that only you know:

- the test function names, and the subtest name per case
- the case table — every case, with its exact inputs
- the exact expected values: byte fixtures, opcodes, version gates, error
  strings, field-by-field assertions

and name rather than spell out everything the repo already decides:

- imports, `t.Run` wrappers, `defer`/cleanup
- fixture and builder construction — `Patterns to copy:
  `services/atlas-y/.../baz_test.go:40-72` (same setup)`
- test-DB / tenant-context helpers

```markdown
- [ ] **Step 1: Write the failing test**

`TestFindDecision` — table-driven, setup copied from
`services/atlas-channel/.../cash_shop_check_name_change_test.go:1-90`.

| case | session state | gm | expect mode | expect payload |
|---|---|---|---|---|
| same field | `CashSceneNone`, field 100 | 0 | `0x09` | mapId 100 |
| in cash shop | `CashSceneCashShop` | 0 | `0x09` | `-1` |
```

This overrides `superpowers:writing-plans`' "no test code without actual test
code" rule, for scaffolding only. **A vague case table or expected value is a
plan failure.** If you cannot write the expected value down, you have not
finished designing the task — do not paper over it with a pointer.

**Size tasks to the implementer budget.** Implementers stop at 120 tool calls
and hand back `PARTIAL`; the controller then splits the task anyway, at a
worse moment and a higher cost. Split at plan time instead:

- A task touching **more than ~6 files**, or **more than one service**, gets
  split — unless the edits are the same mechanical change repeated, which
  batches fine.
- A task whose acceptance needs a new REST surface *and* a new Kafka consumer
  *and* their tests is three tasks.
- Prefer several small tasks over one large one. The review loop is per task,
  so smaller tasks also mean tighter review surfaces.
- **Before sizing a templated transformation into a second implementer task,
  check whether an AST codemod is cheaper than the manual dispatches it
  replaces** — see [docs/codemod-vs-agents.md](../../docs/codemod-vs-agents.md)
  for the break-even arithmetic and the worked example. This is dormant until
  a rewriter exists (no `tools/` codemod is built yet), but the sizing
  question still belongs at plan time, not discovered mid-execution.

Note in `context.md` any task you deliberately left large, and why.

### Step 5b — Lint the plan (required)

```
tools/plan-lint.sh docs/tasks/<id>/plan.md
```

Must exit 0 before you commit. It checks the rules Step 5a already states, and
which nothing previously enforced:

| | Check | Why |
|---|---|---|
| **F1** | every `### Files` path exists, or is marked `new file` | Step 5a's own rule |
| **F2** | the plan does not specify a stub to land | CLAUDE.md: no `// TODO`, stubbed handlers or 501s |
| **F3** | read-only commands the plan tells the implementer to run actually match something | |
| **F4** | task size (>6 files, or >1 service) — *warning* | Step 5a's splitting rule |
| **F5** | every symbol a ```` ```go ```` block calls resolves — *warning* | the code in the plan is written without a compiler |

F1–F3 are errors; fix them. F4 and F5 are advisory.

**If F1 or F5 fires, that is a Step 4 miss, not a normal outcome.** The survey
already listed every unresolved path and every exported symbol; a failure here
means the plan was written past evidence that was sitting in context. Note what
it caught, because that is the loop this phase is trying to stop paying for.

F4: a deliberately large task is allowed provided `context.md` says why, but
oversized tasks are what produce `PARTIAL` hand-backs and a mid-plan split at a
worse moment.

F5: a symbol resolves if the repo defines it, the repo calls it anywhere, or the
plan declares it. What survives is either the repo's first use of an external
API — confirm the signature — or a method you invented from memory, which an
implementer hits at dispatch time. **Grep each one before committing**; advisory
only because the first case is legitimate, not because it is optional.

F5 indexes every `.go` file in the tree, so it adds ~5s. `--no-symbols` skips it.

Every one of these defects is cheap to fix here and expensive at dispatch time,
where it surfaces as a `CONTROLLER RULING` at 150–250k context.

### Step 6 — Commit and summarize

```
git add docs/tasks/<id>/plan.md docs/tasks/<id>/context.md
git commit -m "plan(<id>): implementation plan and context"
```

Verify post-commit:

```
git rev-parse --show-toplevel  # must end with /.worktrees/<id>
git branch --show-current      # must be <id>
```

If either is wrong, STOP and report BLOCKED. Then tell the user:

> Plan and context saved and committed. Now run `/clear`, then `/execute-task <id>`. (You're already in the right worktree.)

## Important Rules

- All file I/O uses absolute paths under `<worktree>`.
- **One `cd`, in Step 2.** Everything after it is relative to the worktree root
  or absolute under it.
- **Independent commands go in one tool block.** A turn that runs a single `ls`
  costs the same context replay as a turn that runs twelve.
- Never write plan artifacts under main's `docs/tasks/`.
- At most one `Explore` dispatch (Step 4a), `model: sonnet`, and only above the
  stated threshold.
- DO NOT begin implementation. This phase produces planning documents only.

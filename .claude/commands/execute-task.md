---
description: Phase 4 — invoke superpowers:subagent-driven-development to implement a planned task in its existing worktree
argument-hint: Task identifier — accepts "task-054-effect-duration-units", "task-054", "054", or "54"
---

You are starting Phase 4 of the Atlas four-phase development workflow. Argument: **$ARGUMENTS**

## Process

### Step 1 — Resolve the task

Same fuzzy-match algorithm as `/design-task` Step 1:

1. Glob `docs/tasks/task-*` (main) and `.worktrees/*/docs/tasks/task-*` (sibling worktrees).
2. Match `$ARGUMENTS` against folder names — exact, number-only (`54`/`054`/`task-54`/`task-054`), or slug fragment.
3. Zero matches → ask for correction. Multiple matches → list and let the user pick.
4. If the task lives only on main with no worktree, stop and tell the user the task needs a worktree.
5. Resolve to `<worktree>/docs/tasks/<id>/`.

### Step 2 — Ensure we're in the right worktree

Run `pwd`. If it does NOT match `<worktree>`, `cd <worktree>` yourself and continue from there. Do NOT ask the user to re-run the command — per CLAUDE.md's "Worktree Discipline" rule, cd into the task worktree yourself.

Do NOT create a new worktree — the worktree was created by `/spec-task` and must be reused so phase artifacts stay co-located.

### Step 3 — Validate inputs

Confirm `<worktree>/docs/tasks/<id>/plan.md` AND `context.md` exist. If either is missing, tell the user to complete `/plan-task` first.

### Step 4 — Invoke subagent-driven-development

Use the Skill tool to invoke `superpowers:subagent-driven-development` (default). Pass:

- Plan path: `<worktree>/docs/tasks/<id>/plan.md`
- Context path: `<worktree>/docs/tasks/<id>/context.md`
- Project conventions: `<worktree>/CLAUDE.md`
- **Worktree absolute path** (`<worktree>`) for every dispatched implementer subagent. Subagent prompts MUST follow the cwd-discipline template from memory `feedback_subagent_worktree_cwd.md` — every Bash call prefixed with `cd <worktree> && ...`, post-commit branch verification, no destructive git ops, no `git add -A` / `git add .`.

**Every implementer dispatch uses `subagent_type: atlas-implementer`** — not
`general-purpose`. That agent carries the three Atlas contracts the generic
implementer template lacks (tool-call cap with `PARTIAL` hand-back,
verification scope, brief-first discovery). Its contracts override the
plugin's `implementer-prompt.md` wherever the two disagree — in particular,
the plugin template's "run the full suite once before committing" does NOT
apply here; see Step 4c.

Your dispatch prompt still supplies what the agent cannot know: where the
task fits, the brief path, the report path, interfaces and decisions from
earlier tasks, and your resolution of any ambiguity you noticed. Do not
restate the agent's contracts in the prompt.

### Step 4a — Model discipline for every dispatch

Pass an explicit `model` on **every** Agent/Task dispatch. Never rely on
inheritance: an unspecified model inherits the main-loop model (Opus), and an
Opus subagent turn costs ~7x a Sonnet one.

The pin is chosen by the **job the agent is doing**, not by its
`subagent_type`. Named reviewer agents carry a Sonnet pin in their frontmatter,
but an ad-hoc `general-purpose` dispatch carrying a review prompt does not —
that is the hole this rule closes.

| Job | Model | Notes |
|---|---|---|
| Review, verify, audit, re-review, whole-branch review | **`sonnet`** — always | No exceptions. Reviewing is reading against a checklist; Opus buys nothing and these run long |
| Scan, inventory, doc sweep, file-finding | `haiku` | |
| Run the verification gate (`atlas-verifier`) | `haiku` | Frontmatter pin; it runs one command and quotes the output |
| Implement a plan task (`atlas-implementer`) | `sonnet` | Default; frontmatter pin |
| Implement a plan task tagged `model: opus` in plan.md | `opus` | Opt-in only — see below; pass `model: opus` on the dispatch to override the frontmatter |

A plan task may be tagged `model: opus` in `plan.md` when it is genuinely
derivation-heavy: IDA/packet field-order derivation, saga orchestration across
services, or a cross-service contract change. `/plan-task` should apply that tag
sparingly and justify it in one line. Everything else — REST surfaces, GORM
entities, Kafka consumers, tests, template routing — runs Sonnet.

If an implementer comes back wrong twice on Sonnet, escalate that one task to
Opus and note it, rather than raising the default.

If the user explicitly requests inline mode this session (rare), invoke `superpowers:executing-plans` instead.

### Step 4b — Check the brief carries its file inventory

`scripts/task-brief` extracts the plan's task section verbatim, so a plan task
written per `/plan-task` already carries a `### Files` block naming every file
the task touches plus the patterns to copy. That block is what removes the
implementer's discovery phase — the phase that inflates context before a
single edit happens.

**Before each dispatch, check the generated brief for a `### Files` section.**
If it is missing (an older plan, or a task the planner under-specified),
produce the inventory yourself — once, in your own context — and append it to
the brief file before dispatching:

```markdown
### Files

- `path/to/file.go` — what this task changes here
- `path/to/other_test.go` — new test file

Patterns to copy: `path/to/reference.go:120` (handler shape)
```

Appending to the brief, not to the dispatch prompt, is deliberate: the brief
stays the single source of requirements, and a continuation dispatch inherits
it. One inventory pass in the controller costs a fraction of the same
discovery repeated inside a large implementer context.

`atlas-implementer` reports `NEEDS_CONTEXT` on a brief with no `### Files`
section rather than falling back to a repo sweep. If that comes back, this
step was skipped — do it and re-dispatch.

### Step 4c — Verification runs outside the implementer

`atlas-implementer` runs only module-local `go build ./... && go test ./...`.
It never runs `tools/verify.sh`, `tools/lint.sh`, `-race`, or docker bake —
those in a 400k-token implementer context cost a large multiple of the same
run in a clean 20k one, and their output is the biggest avoidable consumer of
an implementer's window.

After an implementer reports `DONE` / `DONE_WITH_CONCERNS`, and before the
task reviewer:

1. Dispatch `atlas-verifier` (`model: haiku`) with the worktree path and
   `tools/verify.sh --quick`.
2. **PASS** → proceed to the task review as normal.
3. **FAIL** → the quoted failing block becomes a review finding. Feed it into
   the existing fix loop (resume the implementer for rounds 1-3), then
   re-dispatch `atlas-verifier` for the fix commit. Never fix it yourself in
   the controller session.
4. **ERROR** → the gate did not run. Resolve it (wrong tree, timeout) and
   re-dispatch. Never treat ERROR as PASS.

The flagless `tools/verify.sh` still runs exactly once, at branch end, in
`superpowers:finishing-a-development-branch`. `--quick` per task is the inner
loop, not the gate — per CLAUDE.md only the flagless run counts as verified.

### Step 4d — Handle `PARTIAL`

`atlas-implementer` adds a fifth status to the plugin's four. `PARTIAL` means
the tool-call cap (120, warned at 100) was reached with work remaining: the
implementer committed what works and handed back the remainder. **This is the
designed outcome, not a failure — do not scold it, and do not re-dispatch the
same agent to "finish the job" in its now-large context.**

On `PARTIAL`:

1. Ledger it: `Task <N>: partial (commits <a7>..<b7>, cap reached — <what remains>)`.
2. Write a continuation brief beside the original — `task-N-brief-cont.md`,
   or `-cont2` for a second — containing: the remaining work file by file
   (from the report), the `### Files` inventory for just those files, and
   the interfaces and decisions the first implementer recorded.
3. Dispatch a **fresh** `atlas-implementer` with the continuation brief, the
   same report file path (it is the persistent memory across the split), and
   one line of framing: "A prior implementer completed part of this task and
   hit the tool-call cap. Read the report file for what was done."
4. The task review and `atlas-verifier` run once over the whole task range
   (BASE from before the first dispatch through the final HEAD) — not once
   per segment.

Two `PARTIAL`s on one task means the plan under-decomposed it. Rule on a
split, ledger the ruling, and carry it forward — the plan task was too big,
which is information `/plan-task` sizing should have caught.

### Step 5 — On completion

After all plan tasks complete and verify, the chosen skill hands off to `superpowers:finishing-a-development-branch`. Honor that handoff. Then suggest:

> All plan tasks complete. Recommend running `superpowers:requesting-code-review` next, which dispatches the appropriate reviewer agents (plan-adherence, backend-guidelines, frontend-guidelines) in parallel.

## Important Rules

- The worktree was created by `/spec-task`. NEVER create a new one here.
- Implementers are `atlas-implementer`, never `general-purpose`.
- Never run `tools/verify.sh` inside an implementer — that is `atlas-verifier`'s job (Step 4c).
- Never dispatch a brief with no `### Files` section (Step 4b).
- Never start implementation outside the task worktree.
- Follow plan steps exactly; stop and ask when blocked rather than guessing.
- Run the verification commands the plan specifies; don't claim completion based on assumption.

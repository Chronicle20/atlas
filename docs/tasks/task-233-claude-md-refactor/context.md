# Context — task-233 CLAUDE.md refactor

Companion to [`plan.md`](plan.md). Everything an implementer needs that the plan's task sections do not carry.

## What this task is

An information-architecture change to the repository's always-loaded context file, with an anti-regression obligation. `CLAUDE.md` (220 lines / 19,543 bytes / ~5k tokens at branch point) loads into every session before the agent knows what it has been asked to do. The content is not wrong; it is uniformly *eager*. The refactor relocates detail behind trigger-labeled breadcrumbs while proving no rule lost force.

**Documentation only.** No Go, TypeScript, test, build, or CI file is touched. There is no test cycle; the verification in each task is a reading check.

## Key files

### Rewritten

| Path | Role |
|---|---|
| `CLAUDE.md` | The subject. Becomes invariants + operating defaults + trigger-labeled routers |

### New owner documents (budget: exactly four)

| Path | Owns |
|---|---|
| `docs/git-workflow.md` | Branch safety, history rewrites, build triggering, `gh` auth |
| `docs/reverse-engineering.md` | IDA session resolution, `func_query`, version confirmation — narrow by acceptance criterion |
| `docs/tooling-conventions.md` | Locating Go module source, waiting on processes, shell/editing conventions |
| `docs/agent-dispatch.md` | Model selection, implementer budget, verification split, fork policy, context handoff |

### Extended owners

| Path | Change |
|---|---|
| `docs/verification.md` | Add "a flagged run is not a pass" + background-execution guidance. Most FR-4 verification material is **already there** — those rows are `drop-captured`, not relocations |
| `docs/superpowers-integration.md` | Gains task resolution + artifact-override + code-review detail; loses §Packet Work and §Phase 4 context budget to pointers |
| `docs/observability.md` | Gains a "Diagnosing a runtime failure" section after §Filtering by environment |
| `.claude/commands/execute-task.md` | Step 4a model table and Steps 4d/4e arithmetic → links to `docs/agent-dispatch.md`; procedural steps stay |

### Task artifacts

| Path | Role |
|---|---|
| `docs/tasks/task-233-claude-md-refactor/inventory.md` | FR-1 rule ledger — the working document for the whole refactor, built first |
| `docs/tasks/task-233-claude-md-refactor/ownership.md` | FR-2 ownership map |

### Unchanged, but read

`docs/packets/PROCESS.md` (confirmed sufficient — design §4.1), `docs/adding-a-new-service.md`, `docs/runbooks/ephemeral-pr-deployments.md` (cross-link target), `libs/atlas-constants/README.md`.

## Decisions already made (do not re-litigate)

All five PRD open questions are resolved in `design.md` §4:

1. **Runtime-debugging owner** → extend `docs/observability.md`, not a new runbook. A new runbook would exceed the four-document budget to hold four sentences and would split log access across two files. (§4.2)
2. **`docs/packets/PROCESS.md` sufficiency** → confirmed sufficient against the file. The root packet section collapses to one router line with **no content added** to `PROCESS.md`. This also makes `superpowers-integration.md` §Packet Work a duplicate, so it is reduced to a pointer. (§4.1)
3. **`agent-dispatch.md` vs `superpowers-integration.md`** → separate documents. Dispatch policy applies to every session including ad-hoc ones outside the four-phase workflow; folding it into a workflow doc makes ownership ambiguous in exactly the direction this task is fixing. Boundary is stated in **both** files. (§4.3)
4. **Constants-library and code-pattern rules** → stay in root as one-line Tier-2 bullets. They fit in one line, apply to every Go edit, and are cheaper to keep loaded than to route to. The enumeration of library contents and the `DOM-21` identifier drop. (§4.4)
5. **task-231 number collision** → out of scope, unchanged. Resolving it would mean renaming another task's branch and worktree from inside this one. (§4.5)

Also settled by the design:

- **Approach.** Pure-router (~40 lines, everything linked) was rejected: the branch-safety, no-invention, and flagless-verify rules govern decisions made *before* the agent knows it needs them — the exact condition under which a link is worthless. A second `@import`ed always-loaded file was rejected on a factual ground: an `@import` loads unconditionally alongside its importer, so the split saves nothing. (§2)
- **`(enforced)` markers** are permitted in root and carry no filename, threshold, or denial semantics. They exist so an agent hitting a denial reads it as policy rather than a tool malfunction. (§3.4)

## Dependencies and ordering

The order in `plan.md` is **forced**, not stylistic (design §5.3):

```
Task 1  ledgers
   ↓
Tasks 2-6  destinations  (2,3,4,5,6 are independent of each other)
   ↓
Task 7  rewrite CLAUDE.md      ← needs every destination to exist and hold its content
   ↓
Task 8  rewire execute-task.md ← needs docs/agent-dispatch.md (Task 4)
   ↓
Task 9  round-trip audit       ← needs everything
```

Task 8 depends only on Task 4 and could run earlier, but running it after Task 7 keeps the `CLAUDE.md`-never-links-to-a-command-file direction (FR-7) checkable in one pass.

Tasks 2–6 are independent and could be dispatched in parallel. They touch disjoint files; the only shared read is `inventory.md`, which is read-only after Task 1.

## Task sizing note

No task exceeds six files, and only Task 5 touches two owner documents at once (`verification.md` + `observability.md`) — grouped because both are small, additive edits sourced from adjacent `CLAUDE.md` sections. Task 2 groups two new documents for the same reason. Task 4 (`agent-dispatch.md`) is the largest single deliverable and is deliberately alone.

`tools/plan-lint.sh docs/tasks/task-233-claude-md-refactor/plan.md` exits 0 on the committed plan — no F1/F2/F3 errors, no F4 size warning.

Each task's `### Files` block is followed by a `### Steps` heading. That is not decoration: `plan-lint.sh` and `tools/task-brief.sh` both close the Files block on the next `###` heading, and without it every prose bullet in the task body is read as a file reference.

## Traps

- **Writing `CLAUDE.md` first.** The most tempting shortcut and the one FR-6.3 explicitly forbids — it produces breadcrumbs pointing at documents that do not yet answer the question.
- **`drop-captured` by assertion.** The disposition requires the destination text to already exist, verified by reading. Most of the verification block genuinely is already in `docs/verification.md`; some of it is not. Check each row.
- **Lossy copy of the model table.** Task 8 deletes the source table in `execute-task.md`. If Task 4's copy softened a row — particularly "review/verify/audit → sonnet, always, no exceptions" — that becomes a silent policy change with no source left to compare against.
- **Skipping the reverse round-trip.** Forward ("nothing lost") is the intuitive direction and the one that gets done. Reverse ("nothing invented") is what catches a rule that got *stronger* or *newer* during compression, which PRD §2 forbids just as firmly.
- **Optimizing for line count.** Size is an outcome reported in the PR description, never a target. Every close call goes to retention (PRD §8).
- **Home-path leakage.** The four new documents are the highest-risk files this branch produces for the home-path guard. Use repo-relative paths and placeholders throughout; `~/.config/atlas/gh.env` in `git-workflow.md` is a tilde path in prose, which is fine, but do not expand it.

## Verification

`tools/verify.sh` is change-gated and this branch touches no Go module, so a green run proves almost nothing here — but it does exercise the repo guards, including the home-path guard on the new documents, which is the one way this branch can fail mechanically. Run it flaglessly before PR, in the background.

**Acceptance is by human reading against PRD §10.** Explicitly declined by the PRD: any automated `CLAUDE.md` size cap, lint rule, or link checker. The two mechanical checks in Task 9 (link resolution, exclusion scan) are plan steps, not committed tooling.

Report in the PR description: before 220 lines / 19,543 bytes, after as measured, framed as an outcome; plus the total size added under `docs/`, so the trade is visible rather than implied.

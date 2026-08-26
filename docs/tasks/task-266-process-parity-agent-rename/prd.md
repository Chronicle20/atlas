# Process Parity Phase 1 — Generic Agent Naming — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-26
---

## 1. Overview

Four repositories — `atlas`, `home-hub`, `Harbormaster`, and `MyFleet` — share a
common agentic development process, but only `atlas` has the context-discipline
layer: enforcement hooks, owner documents, and a budget-capped
implementer/verifier/reviewer trio. The other three sit at the previous
generation, with no enforcement hooks beyond `skill-activation-prompt` and no
agent trio at all, so `/execute-task` there falls back to uncapped generic
dispatch.

`docs/process-parity.md` is the canonical specification for closing that gap. It
records the decisions taken: no sync mechanism, full parity in scope, generic
`task-*` agent naming in all four repositories, one four-phase task cycle per
repository, and a post-hoc mechanical consistency assertion. This task is phase 1
of four — the `atlas` cycle.

`atlas` goes first because it is the rename source. Today
`.claude/hooks/commit-boundary.sh`, `.claude/hooks/turn-budget.sh`,
`.claude/hooks/turn-budget-guard.sh`, and `docs/agent-dispatch.md` all hardcode
`atlas-implementer` / `atlas-verifier` / `atlas-reviewer`. Until those names are
generic, every portable file would need a per-repository edit forever. Once they
are generic, the portable set is byte-identical across all four repositories and
the remaining three ports become straight file copies.

This task carries no behavioral change. It renames, updates references, commits
the canonical spec, and amends one over-broad check in that spec.

## 2. Goals

Primary goals:

- Rename the agent trio to `task-implementer`, `task-verifier`, `task-reviewer`,
  including the on-disk filenames under `.claude/agents/`.
- Update all 57 live references so no live file names an `atlas-*` agent.
- Commit `docs/process-parity.md` to the repository as the canonical cross-repo
  specification.
- Amend `docs/process-parity.md` §7 check 3, which is currently unsatisfiable as
  written (see §4.4).
- Update `CLAUDE.md`'s `## Where the procedures live` table to route process-parity
  questions to the new document.

Non-goals:

- Changing any agent's behavior, tool list, budget, or protocol. This is a rename.
- Touching the 715 references in completed task folders under `docs/tasks/`.
- Any work in `home-hub`, `Harbormaster`, or `MyFleet`. Those are phases 2–4.
- Creating `tools/verify.sh`, `tools/task-brief.sh`, or `tools/task-numbers.sh`
  anywhere. `atlas` already has all three.
- Porting owner documents or hooks anywhere. That is phases 2–4.

## 3. User Stories

- As the orchestrating agent, I want the portable hook files to contain no
  repository-specific agent names, so that copying them into the other three
  repositories is a file copy rather than an edit.
- As a developer reading an old `agent-ledger.tsv`, I want the historical rows to
  still say `atlas-implementer`, so that the record matches what was actually
  dispatched.
- As a developer opening `CLAUDE.md`, I want a documented route to the parity
  specification, so that I can find the harmonization contract without knowing the
  filename.

## 4. Functional Requirements

### 4.1 Agent definitions

Rename all three files with `git mv`, preserving history:

| From | To |
|---|---|
| `.claude/agents/atlas-implementer.md` | `.claude/agents/task-implementer.md` |
| `.claude/agents/atlas-verifier.md` | `.claude/agents/task-verifier.md` |
| `.claude/agents/atlas-reviewer.md` | `.claude/agents/task-reviewer.md` |

Update the `name:` frontmatter field in each to match its new filename. Update
the `description:` field and every in-body self-reference and cross-reference
(13 references total across the three files).

Every other frontmatter field — `tools`, and the behavioral contract in the body
— is unchanged. The implementer keeps its 120 tool-call budget and `PARTIAL`
hand-back; the verifier keeps its never-edit constraint; the reviewer keeps its
no-recursive-fan-out constraint.

### 4.2 Live reference sweep

Update the remaining 44 live references:

| File | Refs | Nature |
|---|---|---|
| `.claude/commands/execute-task.md` | 12 | Dispatch instructions |
| `.claude/commands/fix-pr-bug.md` | 4 | Dispatch instructions |
| `.claude/hooks/commit-boundary.sh` | 2 | Operator-facing guidance text |
| `.claude/hooks/turn-budget-guard.sh` | 1 | Comment |
| `.claude/hooks/turn-budget.sh` | 1 | Comment |
| `CLAUDE.md` | 1 | Prose rule |
| `docs/agent-dispatch.md` | 4 | Dispatch policy |
| `docs/post-implementation.md` | 3 | Phase 5 procedure |
| `docs/superpowers-integration.md` | 2 | Skill routing |
| `docs/review-protocol.md` | 1 | Review procedure |
| `docs/codemod-vs-agents.md` | 1 | Prose |
| `tools/agent-ledger_test.sh` | 10 | Test fixtures |
| `tools/agent-ledger.sh` | 1 | Help text |
| `tools/task-step.sh` | 1 | Comment |

No validation logic in `tools/agent-ledger.sh` or `tools/task-step.sh` keys off
the agent-type string; both mention the names only in help text and comments, so
neither changes behavior. `tools/agent-ledger_test.sh` uses the names as sample
data; renaming the fixtures keeps the suite self-consistent.

### 4.3 Historical records

Leave all 715 references under `docs/tasks/task-NNN-*/` untouched — ledger rows,
bug reports, review artifacts, and plans from roughly 40 completed tasks. Those
record dispatches that actually happened under the `atlas-*` names; rewriting
them would make the record assert something untrue.

### 4.4 Specification amendment

`docs/process-parity.md` §7 check 3 currently reads:

> Each repository defines `task-implementer`, `task-verifier`, and
> `task-reviewer`, and no reference to `atlas-implementer`, `atlas-verifier`, or
> `atlas-reviewer` remains anywhere.

Given §4.3, "anywhere" is unsatisfiable. Narrow it to the live surface —
`.claude/`, `CLAUDE.md`, `docs/*.md` at the top level, and `tools/` — explicitly
excluding `docs/tasks/`. Record the historical-name cutoff in
`docs/agent-dispatch.md` so a future reader hitting an `atlas-implementer` row in
an old ledger understands why it is there.

### 4.5 Documentation routing

Add a row to `CLAUDE.md`'s `## Where the procedures live` table routing
cross-repository process-parity questions to `docs/process-parity.md`.

## 5. API Surface

None. No service, endpoint, or wire format is touched.

## 6. Data Model

None. No entity, migration, or schema change.

## 7. Service Impact

No service is affected. All changes are confined to `.claude/`, `CLAUDE.md`,
`docs/`, and `tools/` — repository tooling and documentation only. No Go or
TypeScript source file is modified.

## 8. Non-Functional Requirements

- **Behavior preservation.** Agent behavior must be byte-identical apart from the
  name. A diff of each agent file with the rename reversed should be empty.
- **History preservation.** Use `git mv` so the agent definitions keep their
  history. Do not rewrite completed task artifacts.
- **Hook integrity.** The three touched hooks must still pass their guards after
  the edit; `.claude/hooks/wait-loop-guard_test.sh` and
  `tools/agent-ledger_test.sh` must stay green.
- **Portability.** After this task, the seven portable hook files named in
  `docs/process-parity.md` §3.1 must contain no `atlas-` string, so that phases
  2–4 can copy them verbatim.

## 9. Open Questions

None blocking. Three items remain open in `docs/process-parity.md` §8, but all
three belong to phases 2–4 and none blocks this task:

- `home-hub` has no pinned Go linter config for `format-on-write.sh`.
- `Harbormaster`'s `CLAUDE.md` still claims the repository is unscaffolded.
- `home-hub`'s existing `docs/superpowers-integration.md` needs reconciling.

## 10. Acceptance Criteria

- [ ] `.claude/agents/` contains `task-implementer.md`, `task-verifier.md`, and
      `task-reviewer.md`, and no `atlas-*.md` agent file.
- [ ] Each renamed file's `name:` frontmatter matches its filename.
- [ ] Each renamed file's behavioral contract is unchanged — budget, tool list,
      and protocol identical to the pre-rename version.
- [ ] `git grep -lE 'atlas-(implementer|verifier|reviewer)'` returns only paths
      under `docs/tasks/`.
- [ ] The seven portable hook files from `docs/process-parity.md` §3.1 contain no
      `atlas-` string.
- [ ] `docs/process-parity.md` is committed, with §7 check 3 narrowed per §4.4.
- [ ] `docs/agent-dispatch.md` records the pre-task-266 historical-name cutoff.
- [ ] `CLAUDE.md`'s `## Where the procedures live` table routes to
      `docs/process-parity.md`, and every target file in that table exists.
- [ ] `tools/agent-ledger_test.sh`, `.claude/hooks/wait-loop-guard_test.sh`, and
      `tools/task-numbers_test.sh` pass.
- [ ] Flagless `tools/verify.sh` exits 0.

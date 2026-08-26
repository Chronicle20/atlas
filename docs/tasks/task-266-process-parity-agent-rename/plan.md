# Implementation Plan — task-266 Process Parity Phase 1

Input: `docs/tasks/task-266-process-parity-agent-rename/prd.md`
Canonical spec: `docs/process-parity.md`

No design phase. The PRD enumerates every affected file and reference count, and
the change is a rename with no design space.

## Task 1 — Scripted rename sweep (codemod, not agent judgment)

`docs/process-parity.md` §5.1 designates this the one place a scripted sweep is
appropriate, and `docs/codemod-vs-agents.md` puts a uniform repo-wide
transformation on the codemod side of the line. Do NOT hand this to a second
implementer file-by-file.

1. `git mv` the three agent definitions:
   - `.claude/agents/atlas-implementer.md` → `task-implementer.md`
   - `.claude/agents/atlas-verifier.md` → `task-verifier.md`
   - `.claude/agents/atlas-reviewer.md` → `task-reviewer.md`
2. Apply `atlas-implementer`→`task-implementer`, `atlas-verifier`→`task-verifier`,
   `atlas-reviewer`→`task-reviewer` across the live surface only:
   - `.claude/agents/*.md`, `.claude/commands/*.md`, `.claude/hooks/*.sh`
   - `CLAUDE.md`
   - top-level `docs/*.md`
   - `tools/*.sh`
   - **Excluded:** everything under `docs/tasks/` (PRD §4.3).
3. Preserve existing line endings; do not normalize CRLF→LF.

Expected: 57 references across 17 files (3 renamed + 14 edited).

Verify: `git grep -lE 'atlas-(implementer|verifier|reviewer)'` returns paths
under `docs/tasks/` only. The seven §3.1 portable hooks contain no `atlas-`
string. Each agent file's `name:` frontmatter matches its filename. A diff of
each agent file with the rename reversed is empty — no behavioral drift.

## Task 2 — Judgment edits

Not mechanical; these are three distinct authored changes.

1. **Amend `docs/process-parity.md` §7 check 3** per PRD §4.4. Narrow "anywhere"
   to the live surface — `.claude/`, `CLAUDE.md`, top-level `docs/*.md`, `tools/`
   — explicitly excluding `docs/tasks/`.
2. **Record the historical-name cutoff in `docs/agent-dispatch.md`**: artifacts
   predating task-266 use the `atlas-*` names because those dispatches actually
   happened under them.
3. **Add a `CLAUDE.md` owner-table row** routing cross-repository process-parity
   questions to `docs/process-parity.md`.

Verify: every target file in the `## Where the procedures live` table exists.

## Task 3 — Gates

1. `tools/agent-ledger_test.sh`, `.claude/hooks/wait-loop-guard_test.sh`,
   `tools/task-numbers_test.sh` pass (all three were green at baseline).
2. Flagless `tools/verify.sh` exits 0 — run by `task-verifier` in its own context.

## Dispatch note

The trio being renamed is the trio used to execute this task. Task 1 must land
before any verifier or reviewer dispatch, and those dispatches must then use the
NEW names. Dispatching `atlas-verifier` after Task 1 will fail.

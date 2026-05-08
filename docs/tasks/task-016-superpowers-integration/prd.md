# Superpowers Integration — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-04-21
---

## 1. Overview

Make the `superpowers` plugin (already installed and enabled in `.claude/settings.json`) the default development workflow for Atlas while preserving the project's artifact discipline (numbered task folders under `docs/tasks/`) and domain tooling (backend/frontend guideline skills, per-service audits, maintenance commands).

The project today ships two overlapping conventions. `.claude/commands/` contains inline commands (`spec-task`, `dev-docs`, `dev-docs-update`, `audit-plan`, `backend-audit`, `review-todos`, `service-doc`, and five `convert-*` commands) alongside a single `.claude/agents/documentation.md` that lacks frontmatter. Feature work splits across two folders: `dev/active/<slug>/` (51 folders, each with `<slug>-context.md` / `<slug>-plan.md` / `<slug>-tasks.md`) and `docs/tasks/task-NNN-slug/` (15 folders). The superpowers plugin provides skills covering the full lifecycle (`brainstorming`, `writing-plans`, `executing-plans`, `subagent-driven-development`, `systematic-debugging`, `test-driven-development`, `verification-before-completion`, `using-git-worktrees`, `finishing-a-development-branch`, `requesting-code-review`, `receiving-code-review`, `dispatching-parallel-agents`, `writing-skills`, `using-superpowers`) that atlas currently ignores.

This task merges the two conventions. Atlas adopts the four-phase workflow proven in home-hub (`task-044-superpowers-integration`), collapses `dev/active/` into `docs/tasks/legacy-<slug>/` as the single home for feature artifacts, deletes the legacy `dev/audits/` tree, and promotes inline audit/maintenance commands into proper agents with thin command wrappers.

## 2. Goals

Primary goals:
- Establish a single four-phase workflow (`/spec-task` → `/design-task` → `/plan-task` → `/execute-task`) as the canonical way to deliver feature work in atlas, backed by superpowers skills.
- Consolidate all feature artifacts under `docs/tasks/` with one filename convention (`prd.md`, `design.md`, `plan.md`, `context.md`, optional `audit.md`/`audit.json`).
- Promote five implicit agents (`plan-adherence-reviewer`, `backend-guidelines-reviewer`, `frontend-guidelines-reviewer`, `todo-scanner`, `service-documentation`) out of inline command bodies into named, independently-invokable agents dispatchable via `superpowers:requesting-code-review` or by hand.
- Close the hook gap so the `frontend-dev-guidelines` skill activates on frontend prompts/files (currently orphaned — the skill exists but `skill-rules.json` does not reference it).
- Eliminate the `dev/active/` and `dev/audits/` directories as active working locations; new feature work only uses `docs/tasks/`.

Non-goals:
- Reworking superpowers skills themselves or authoring new skills.
- Migrating `docs/tasks/task-001`…`task-015` content into a different shape (they remain as-is, including any `tasks.md` files).
- Modifying service code, shared libraries, or runtime behavior. No Go service under `services/` is touched.
- Modifying `settings.json`, the hook scripts (`hooks/skill-activation-prompt.{sh,py}`), or the resources under `skills/backend-dev-guidelines/resources/` and `skills/frontend-dev-guidelines/resources/`.
- Modifying the `convert-map`, `convert-npc`, `convert-portal`, `convert-quest`, `convert-reactor` commands — these stay untouched (atlas-specific domain converters, parallel to home-hub's `recipe-to-cooklang`).

## 3. User Stories

- As a developer starting a new feature, I want one command per phase (`/spec-task`, `/design-task`, `/plan-task`, `/execute-task`) invoked from a `/clear`'d session, so each phase reasons only from the prior phase's documented artifact rather than conversation residue.
- As a developer implementing a multi-step plan, I want `/execute-task` to dispatch subagents per task with review checkpoints between tasks, so the main context stays clean and each task has an isolated workspace.
- As a reviewer auditing a feature branch, I want `superpowers:requesting-code-review` to dispatch exactly the right subset of three reviewer agents (plan-adherence + backend-guidelines + frontend-guidelines) in parallel based on which files changed, so I get a combined report without running four commands serially.
- As a developer auditing a single service ad-hoc, I want to invoke `backend-guidelines-reviewer` directly by name with a service path, so I can get an audit without going through the full review orchestration.
- As a developer working on `atlas-ui` frontend code, I want the project hook to emit the `🎯 SKILL ACTIVATION CHECK` banner for `frontend-dev-guidelines` when frontend keywords or files are involved, so the domain skill is invoked reliably.
- As a developer searching for prior work on a topic, I want a single location (`docs/tasks/`) to grep through, so I don't miss artifacts stashed under `dev/active/`.
- As a developer kicking off implementation, I want `/execute-task` to strongly recommend `superpowers:using-git-worktrees` when the current branch is protected, so isolated work and parallelism are the default, not the exception.

## 4. Functional Requirements

### 4.1 Four-phase workflow commands

- **`/spec-task <idea>`** already exists; its handoff text currently points at `/dev-docs`. The handoff is updated to instruct the user to run `/clear` and then `/design-task <task-folder>`. No other behavior change.
- **`/design-task <task-folder>`** (new) validates that `docs/tasks/<task-folder>/prd.md` exists and `design.md` does not. Loads `prd.md`, `CLAUDE.md`, and `docs/superpowers-integration.md` as context. Invokes `superpowers:brainstorming` with explicit overrides: (a) skip the skill's default what/why questions because the PRD answers them; (b) focus on architecture, alternatives, and tradeoffs; (c) output MUST be saved to `docs/tasks/<task-folder>/design.md`, not the skill's default spec location. Does NOT auto-invoke `writing-plans`.
- **`/plan-task <task-folder>`** (new) validates that both `prd.md` and `design.md` exist and `plan.md` does not. Loads the three prior artifacts plus `CLAUDE.md` and `docs/superpowers-integration.md`. Invokes `superpowers:writing-plans` with overrides: output to `docs/tasks/<task-folder>/plan.md` and produce a companion `docs/tasks/<task-folder>/context.md` summarizing key files, decisions, and dependencies. Runs the `writing-plans` skill's self-review before saving. Does NOT auto-invoke execution.
- **`/execute-task <task-folder>`** (new) validates that `plan.md` and `context.md` exist. Asks the user once whether to use subagent-driven (default) or inline execution. If the current branch is `main` or `master`, strongly recommends `superpowers:using-git-worktrees` before proceeding. Invokes `superpowers:subagent-driven-development` (default) or `superpowers:executing-plans` (fallback), passing the plan path, context path, and `CLAUDE.md`. On completion, hands off to `superpowers:finishing-a-development-branch` and suggests the user run `superpowers:requesting-code-review`.

All four phases are invoked from fresh (`/clear`'d) sessions. No auto-chaining between phases.

### 4.2 Code-review agent trio

Three reviewer agents live under `.claude/agents/` with proper YAML frontmatter (`name`, `description` with `<example>` tags, `model: inherit`). Each is invokable directly by name or via `superpowers:requesting-code-review`.

- **`agents/plan-adherence-reviewer.md`** absorbs the body of the current `commands/audit-plan.md`. Given a task folder, walks checkboxes in `plan.md` against `git diff main...HEAD`, classifies each task as DONE / PARTIAL / SKIPPED / DEFERRED / NOT_APPLICABLE with file:line evidence, runs affected-service builds/tests, and writes `docs/tasks/<task-folder>/audit.md`.
- **`agents/backend-guidelines-reviewer.md`** absorbs the body of the current `commands/backend-audit.md`. Runs the DOM-* / SUB-* / SEC-* checklists on changed Go packages under `services/<service>/atlas.com/<module>/`, default-FAIL adversarial mindset, every PASS requires file:line citation. Produces `audit.md` + `audit.json` in the task folder (for a branch with `plan.md`) or under `docs/audits/<service>/` (for standalone service audits).
- **`agents/frontend-guidelines-reviewer.md`** (new) mirrors the backend reviewer for atlas-ui TypeScript/React. Derives its checklist from `skills/frontend-dev-guidelines/resources/` (anti-patterns and pattern files).

### 4.3 Maintenance agents

- **`agents/todo-scanner.md`** (new) absorbs the body of `commands/review-todos.md`. Whole-codebase scan for TODO / FIXME / XXX / HACK markers and unimplemented stubs; categorizes by service and priority; updates `docs/TODO.md`.
- **`agents/service-documentation.md`** (renamed from `agents/documentation.md`) gains proper frontmatter and absorbs the body of `commands/service-doc.md`. Documents one atlas service per invocation, following `DOCS.md` and the service's local `CLAUDE.md`. Operates only within the target service directory.

### 4.4 Slash command wrappers

`commands/audit-plan.md`, `commands/backend-audit.md`, `commands/review-todos.md`, `commands/service-doc.md` shrink to one-line dispatchers against their respective agents. Retained purely for `/` autocomplete discoverability.

### 4.5 Documentation

- **`CLAUDE.md`** — a new section documents the four-phase workflow as the canonical delivery path, declares `docs/tasks/task-NNN-slug/` as the artifact home (overriding superpowers defaults), and describes the modular code-review agent pattern. The existing Workflow Rules, Build & Verification, Code Patterns, and Documentation sections are preserved; the new section is additive.
- **`docs/superpowers-integration.md`** (new) — quick-reference companion to `CLAUDE.md`. Lists the four commands with their outputs, the three reviewer agents with their triggers, the two maintenance commands, the two domain skills (backend-dev-guidelines, frontend-dev-guidelines), the twelve self-activating superpowers skills, and a "when NOT to use superpowers" carve-out for trivial fixes.

### 4.6 Hook and skill-rules

- `skills/skill-rules.json` gains a `frontend-dev-guidelines` entry mirroring the backend entry. Keywords include `frontend`, `react`, `tsx`, `component`, `hook`, `form`, `tailwind`, `shadcn`, `react query`, `tanstack`, `zod`. `pathPatterns` target `atlas-ui/**/*.ts`, `atlas-ui/**/*.tsx`, `**/components/**/*.tsx`, `**/pages/**/*.tsx`, `**/lib/hooks/**/*.ts`, `**/lib/schemas/**/*.ts`, `**/services/api/**/*.ts` (matching atlas-ui's layout). `pathExclusions` cover `**/*.test.ts` and `**/*.test.tsx`.
- No entries are added for superpowers skills; they self-activate.
- `hooks/skill-activation-prompt.{sh,py}` remain unchanged.

### 4.7 dev/active migration (51 folders → docs/tasks/legacy-\<slug\>/)

Each folder under `dev/active/` migrates to `docs/tasks/legacy-<slug>/`. Inner files drop the redundant slug prefix:

- `dev/active/<slug>/<slug>-context.md` → `docs/tasks/legacy-<slug>/context.md`
- `dev/active/<slug>/<slug>-plan.md` → `docs/tasks/legacy-<slug>/plan.md`
- `dev/active/<slug>/<slug>-tasks.md` → `docs/tasks/legacy-<slug>/tasks.md`

The `legacy-` prefix preserves searchability without forcing fake task-NNN numbers. Any non-standard files inside a given folder are moved as-is. Renames must preserve git history (use `git mv`). Files inside that reference sibling filenames (e.g., a plan referencing its own context) are updated to the new names. Inbound references from outside the folder (e.g., `MEMORY.md`, `CLAUDE.md`, other docs) are left as a separate sweep — see §4.9.

After migration, the `dev/active/` directory is removed.

### 4.8 dev/audits deletion

`dev/audits/` is deleted entirely (25 folders, ~outputs of prior `/backend-audit` runs). Going forward, audit artifacts live inside the task folder (for feature-bound audits) or under `docs/audits/<service>/` (for standalone service audits). Existing `dev/audits/*` content is not migrated.

### 4.9 Reference updates

After the migration sweeps, any reference to `dev/active/<slug>/` or `dev/audits/` in tracked files is updated to the new path:

- `CLAUDE.md` — any references updated.
- `docs/TODO.md` — any references updated.
- Any other markdown under `docs/` that points into the old locations.
- Auto-memory files under `~/.claude/projects/-<workspace-pers-atlas>/memory/` that reference `dev/active/...` paths (e.g., `MEMORY.md`) are updated; the `dev/active/` references there today point at redis-migration, automatic-tenant-filtering, saga-orchestrator-durability, character-shop-merchant, writer-packet-extraction.

### 4.10 Removals

- `commands/dev-docs.md` — deleted. Its role is split between `/design-task` (design phase) and `/plan-task` (plan phase).
- `commands/dev-docs-update.md` — deleted. No equivalent; any future doc-update work flows through the regular four-phase pipeline.
- `agents/documentation.md` — renamed to `agents/service-documentation.md` (not a deletion per se).
- `dev/active/` directory — removed after content migration.
- `dev/audits/` directory — deleted outright.

### 4.11 tasks.md convention

Going forward, the canonical per-task file set is `prd.md`, `design.md`, `plan.md`, `context.md`, and optional `audit.md` / `audit.json`. `tasks.md` is NOT produced by the new workflow. Existing `tasks.md` files under `docs/tasks/task-001`…`task-015` remain as historical artifacts; migrated `docs/tasks/legacy-<slug>/tasks.md` files remain as historical artifacts. `CLAUDE.md` documents the new convention without retroactively deleting existing ones.

## 5. API Surface

Not applicable — this task modifies developer tooling under `.claude/` and documentation under `docs/`. No service APIs, no network surface, no Kafka topics, no JSON:API endpoints are added or changed.

## 6. Data Model

Not applicable — no database entities, schemas, or migrations are introduced. No GORM models are touched.

## 7. Service Impact

No Go service is modified. No shared library under `libs/` is modified. No frontend code under `atlas-ui/` is modified. This task is entirely confined to repo-root tooling and documentation:

- `.claude/commands/` — add 3, shrink 4, remove 2.
- `.claude/agents/` — add 4 new, rename 1 (documentation → service-documentation) with frontmatter.
- `.claude/skills/skill-rules.json` — add one entry.
- `CLAUDE.md` — add one section.
- `docs/superpowers-integration.md` — new file.
- `docs/tasks/` — add `legacy-<slug>/` folders via `git mv` from `dev/active/`.
- `dev/active/` — deleted after migration.
- `dev/audits/` — deleted outright.

The only indirect runtime impact is that subsequent `/execute-task` runs will be recommended to happen inside a git worktree rather than directly on the working branch; this is opt-in guidance, not enforced.

## 8. Non-Functional Requirements

### 8.1 Backwards compatibility

- Deleting `commands/dev-docs.md` and `commands/dev-docs-update.md` breaks muscle memory for any contributor accustomed to those commands. `CLAUDE.md` and `docs/superpowers-integration.md` explicitly document the replacement path so a developer who types `/dev-docs` and gets "command not found" can self-serve the fix.
- Existing `docs/tasks/task-001`…`task-015` folders are not rewritten. A developer reading an old task that still has `tasks.md` will see both `plan.md` and `tasks.md`; this is expected.

### 8.2 Git history preservation

All folder/file moves from `dev/active/` to `docs/tasks/legacy-<slug>/` use `git mv` so blame and log traversal survive. The commit(s) performing the migration use conventional commit prefixes (`refactor:` or `chore:`) and a message explicit about the move.

### 8.3 Agent frontmatter discipline

Every agent file under `.claude/agents/` must have YAML frontmatter matching the superpowers convention:

```yaml
---
name: <kebab-case-name>
description: |
  <when to invoke — specific enough that the main agent can decide to delegate>
  <1–2 <example> tags showing trigger scenarios>
model: inherit
---
```

Reference: `~/.claude/plugins/cache/claude-plugins-official/superpowers/5.0.7/agents/code-reviewer.md`.

### 8.4 Verification

`.claude/` config has no unit tests. Verification per change:

1. File presence + correct frontmatter (Read tool).
2. `skill-rules.json` JSON validity (`python3 -m json.tool < .claude/skills/skill-rules.json`).
3. Hook smoke test: pipe a sample frontend-flavored prompt into `python3 .claude/hooks/skill-activation-prompt.py` and verify the frontend-dev-guidelines banner appears.
4. Live invocation smoke test at the end: fresh session, run the four-phase flow on a tiny throwaway task to confirm artifacts land in the right folders and the handoffs work.
5. After dev/active migration: `git ls-files dev/active/` returns empty; `git log --follow -- docs/tasks/legacy-<slug>/context.md` shows pre-rename history.

### 8.5 Multi-tenancy

Not applicable — no runtime code, no tenant scoping, no tenant columns.

### 8.6 Security

No credentials, no network reach, no secrets in the new files. The hook already runs `python3` over user prompts; no new sandbox surface is introduced.

## 9. Open Questions

None blocking at the spec level. Design-phase items that will be worked out in `design.md`:

- Exact wording for the `CLAUDE.md` workflow section (tone, level of detail).
- How `/design-task` and `/plan-task` pass the task folder path into their underlying superpowers skills (skill argument, preamble prompt, or shared-context convention).
- Whether `frontend-guidelines-reviewer` targets only `atlas-ui` or also looks at any TS/React code elsewhere in the repo (expected: atlas-ui only, given atlas-ui is the sole frontend).
- Whether migrated `legacy-<slug>/` folders get a short `legacy.md` marker explaining their provenance (expected: no, the folder name is explanation enough).

## 10. Acceptance Criteria

- [ ] `.claude/commands/design-task.md`, `plan-task.md`, `execute-task.md` exist and each invokes the correct superpowers skill with the task-folder override.
- [ ] `.claude/agents/plan-adherence-reviewer.md`, `backend-guidelines-reviewer.md`, `frontend-guidelines-reviewer.md`, `todo-scanner.md`, `service-documentation.md` all exist with proper frontmatter (`name`, `description` with at least one `<example>` tag, `model: inherit`).
- [ ] `.claude/agents/documentation.md` no longer exists.
- [ ] `.claude/commands/audit-plan.md`, `backend-audit.md`, `review-todos.md`, `service-doc.md` are each shrunk to a thin wrapper that dispatches the corresponding agent.
- [ ] `.claude/commands/dev-docs.md` and `.claude/commands/dev-docs-update.md` no longer exist.
- [ ] `.claude/skills/skill-rules.json` contains a `frontend-dev-guidelines` entry; file is valid JSON; hook smoke test with a frontend-flavored prompt emits the expected banner.
- [ ] `CLAUDE.md` has a new section describing the four-phase workflow, the `docs/tasks/task-NNN-slug/` artifact home, and the modular code-review pattern, without overwriting the existing Workflow Rules / Build & Verification / Code Patterns / Documentation sections.
- [ ] `docs/superpowers-integration.md` exists and covers: four commands, three reviewer agents, two maintenance commands, two domain skills, self-activating superpowers skills, file-location cheat sheet, and a "when NOT to use superpowers" section.
- [ ] Every folder under `dev/active/` has been moved to `docs/tasks/legacy-<original-slug>/`; inner filenames drop the redundant slug prefix; `dev/active/` directory no longer exists; `git log --follow` on migrated files traverses pre-rename history.
- [ ] `dev/audits/` no longer exists.
- [ ] No reference to `dev/active/` or `dev/audits/` remains in tracked files (`git grep -E 'dev/(active|audits)/'` returns empty, excluding this PRD).
- [ ] Running the full four-phase flow on a throwaway task produces `prd.md` → `design.md` → `plan.md` + `context.md` → committed code, all under `docs/tasks/task-NNN-slug/`, with no mention of the old `dev-docs` flow and no `tasks.md` generated.
- [ ] Code review on a sample feature branch dispatches the correct subset of the three reviewer agents in parallel and aggregates into a single report.

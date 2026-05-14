# Superpowers Integration — Design

Status: Draft
Created: 2026-04-21
Follows: `prd.md`
---

## 1. Purpose

Design the mechanics for integrating the superpowers plugin as atlas's default development workflow while preserving the `docs/tasks/` artifact convention and domain tooling. This document resolves PRD §9's open architectural questions and pins down the concrete file shapes, invocation patterns, and migration mechanics that `plan.md` will translate into bite-sized tasks.

The reference implementation is home-hub (`~/source/pers/home-hub`, specifically its `task-044-superpowers-integration`). Where atlas diverges from home-hub, this document calls out the delta explicitly.

## 2. Architecture Overview

### 2.1 Command → skill handoff mechanism

**Decision: Preamble prompt.** Each phase command's body is a natural-language instruction to the main agent. The body instructs the agent to (a) validate inputs, (b) load prior artifacts plus `CLAUDE.md` and `docs/superpowers-integration.md`, (c) invoke the relevant superpowers skill via the Skill tool, and (d) pass overrides as in-prompt context inside the invocation ("output MUST be saved to `docs/tasks/<folder>/design.md`", "skip default what/why questions", "do NOT auto-invoke the next phase"). The skill itself is unmodified; the main agent carries the override context into the skill's flow.

Rejected alternatives:

- **Shared-context file** (command writes `.design-context.json`, skill reads it): invents a new convention that superpowers skills don't know about. Would require skill modification — violates PRD §2 non-goal "not reworking superpowers skills."
- **Environment variable**: skills don't read env vars; same reworking constraint.

The preamble-prompt pattern is what home-hub uses today and has proven out across its task-044 execution.

### 2.2 Phase command shape

All three new commands (`/design-task`, `/plan-task`, `/execute-task`) share a four-step body:

1. **Validate input** — resolve `$ARGUMENTS` to `docs/tasks/$ARGUMENTS/`; confirm the required prior artifacts exist; confirm this phase's output does NOT already exist (ask before overwrite).
2. **Load context** — the prior artifacts, `CLAUDE.md`, `docs/superpowers-integration.md`, and code areas implied by the PRD/design's Service Impact section.
3. **Invoke the superpowers skill via the Skill tool**, with overrides passed in natural language: output path, scope focus, no-auto-chain directive.
4. **Save and confirm** — after the skill produces its artifact, tell the user to `/clear` and run the next phase.

No auto-chaining between phases. The `/clear` between them is intentional: each phase consumes only the prior phase's documented artifact, not conversation residue.

`/execute-task` adds two extra steps between validation and invocation: (a) ask the user once whether to use subagent-driven (default) or inline execution; (b) if on `main`/`master`, strongly recommend `superpowers:using-git-worktrees`.

### 2.3 Reviewer-agent architecture

Three reviewer agents live under `.claude/agents/` with full YAML frontmatter. Each is independently invokable by name OR dispatched in parallel via `superpowers:requesting-code-review`. The orchestrator skill decides which subset applies based on `git diff main...HEAD` file extensions:

- `.go` files changed → `backend-guidelines-reviewer`
- `.ts`/`.tsx` files changed → `frontend-guidelines-reviewer`
- Any task folder with a `plan.md` → `plan-adherence-reviewer` (always)

Each agent writes its findings to a shared `docs/tasks/<task-folder>/audit.md` (appending sections) when invoked from a feature-branch context; standalone invocations write to `docs/audits/<service-name>/audit.md` instead.

### 2.4 Thin command wrappers

`commands/audit-plan.md`, `backend-audit.md`, `review-todos.md`, `service-doc.md` shrink to ~5-line dispatchers whose sole purpose is `/` autocomplete discoverability. Body format copied from home-hub:

```
---
description: <one-liner>
argument-hint: <hint>
---
Dispatch the `<agent-name>` agent against: **$ARGUMENTS**.
Pass the <thing> so the agent can <purpose>. Write results to <path>.
After completion, summarize <relevant metric> to the user.
```

## 3. File-by-file design

### 3.1 `.claude/commands/design-task.md` (new)

Body adapted from `~/source/pers/home-hub/.claude/commands/design-task.md`, with s/Home Hub/Atlas/. Validates `docs/tasks/$ARGUMENTS/prd.md` exists and `design.md` does not. Loads the three context documents. Invokes `superpowers:brainstorming` with overrides:

- PRD already exists at `docs/tasks/$ARGUMENTS/prd.md`; skip the skill's default what/why questions.
- Focus the conversation on architecture, alternatives, and tradeoffs.
- Design output MUST be saved to `docs/tasks/$ARGUMENTS/design.md`, NOT `docs/superpowers/specs/...`.
- After the design is approved, do NOT auto-invoke `writing-plans`. Hand off to the user: "run `/clear`, then `/plan-task $ARGUMENTS`".

### 3.2 `.claude/commands/plan-task.md` (new)

Body adapted from home-hub. Validates both `prd.md` and `design.md` exist and `plan.md` does not. Loads the four context documents (prd, design, CLAUDE.md, superpowers-integration.md). Invokes `superpowers:writing-plans` with overrides:

- Spec is at `docs/tasks/$ARGUMENTS/design.md`.
- Plan output MUST be saved to `docs/tasks/$ARGUMENTS/plan.md`, NOT `docs/superpowers/plans/...`.
- Also produce `docs/tasks/$ARGUMENTS/context.md` — a quick-reference summary of key files, decisions, and dependencies that an executing subagent can load instead of re-reading the full design.
- Run the `writing-plans` skill's self-review (placeholder scan, type consistency, spec coverage) before saving.
- Do NOT auto-invoke execution; tell the user to `/clear` and run `/execute-task $ARGUMENTS`.

### 3.3 `.claude/commands/execute-task.md` (new)

Body adapted from home-hub. Validates `plan.md` and `context.md` exist. Asks the user once: subagent-driven (default, `superpowers:subagent-driven-development`) or inline (`superpowers:executing-plans`). If on `main`/`master`, strongly recommends `superpowers:using-git-worktrees` before proceeding. Invokes the chosen skill, passing plan path, context path, `CLAUDE.md`. On completion, hands off to `superpowers:finishing-a-development-branch` and then suggests `superpowers:requesting-code-review`.

### 3.4 `.claude/commands/spec-task.md` (edited)

Only the Step 5 handoff text changes. Today it says "Run `/dev-docs task-NNN-slug`". New text: "Now run `/clear` to reset context, then `/design-task task-NNN-slug` to invoke the brainstorming/design phase." No other changes.

### 3.5 `.claude/agents/plan-adherence-reviewer.md` (new)

Absorbs the body of the current `commands/audit-plan.md`, reformatted with the superpowers agent frontmatter convention. Key adaptations:

- **Input**: task folder path (e.g., `docs/tasks/task-016-superpowers-integration`). Plan lives at `<task-folder>/plan.md`.
- **Process**: load plan → parse checkboxes → diff `main...HEAD` → classify each task DONE / PARTIAL / SKIPPED / DEFERRED / NOT_APPLICABLE with file:line evidence.
- **Build & test verification** (atlas adaptation): for each affected Go service, run `cd services/<service>/atlas.com/<module> && go build ./... && go test ./... -count=1`. For atlas-ui changes, run `cd services/atlas-ui && npm run build && npm test`.
- **Output**: `<task-folder>/audit.md` with the five-section template from home-hub (executive summary, task table, skipped/deferred detail, build & test results, overall assessment).

### 3.6 `.claude/agents/backend-guidelines-reviewer.md` (new)

Absorbs the body of the current `commands/backend-audit.md`. Same shape as home-hub's backend reviewer with atlas-specific adaptations:

- **Input**: either a service path (`services/atlas-account`) or a list of changed Go packages from `git diff`.
- **Guidelines source**: reads `.claude/skills/backend-dev-guidelines/resources/*.md` (ai-guidance, file-responsibilities, anti-patterns, testing-guide, patterns-provider, patterns-multitenancy-context, patterns-rest-jsonapi, patterns-functional, scaffolding-checklist).
- **Build/test gate**: `cd services/<service>/atlas.com/<module> && go build ./...` then `go test ./... -count=1`. Build or test failure → overall FAIL, skip to Phase 5.
- **Phase 2 Domain Discovery**: list packages under `<service-path>/atlas.com/<module>/internal/` (the atlas path depth differs from home-hub's `<service>/internal/`). Classify as domain (has `model.go`) / sub-domain (has `resource.go` only) / support.
- **Phase 3 checklists**: DOM-01 through DOM-20, SUB-01 through SUB-04, verbatim from home-hub. These IDs are content-derived from the backend-dev-guidelines skill which is shared in shape across home-hub and atlas.
- **Phase 4 security** (SEC-01 through SEC-04): applies only to auth-related services. For atlas, this means `services/atlas-login`, `services/atlas-account`.
- **Output**: `docs/audits/<service-name>/audit.{md,json}` for standalone invocations; `docs/tasks/<task-folder>/audit.{md,json}` for feature-branch invocations. Default FAIL, every PASS needs file:line evidence.

### 3.7 `.claude/agents/frontend-guidelines-reviewer.md` (new)

Mirrors home-hub's frontend reviewer, scoped to atlas-ui only (per §2 resolution). Adaptations:

- **Input**: either `services/atlas-ui/src` or a list of changed `.ts`/`.tsx` files.
- **Guidelines source**: `.claude/skills/frontend-dev-guidelines/SKILL.md` + `.claude/skills/frontend-dev-guidelines/resources/*.md`.
- **Build/test gate**: `cd services/atlas-ui && npm run build` (uses `tsc -b && vite build`), then `npm test` (uses `vitest run`). NOTE: differs from home-hub's Jest/`npm test -- --watchAll=false` form — atlas-ui uses Vitest.
- **Phase 2 File inventory**: classify each in-scope file as Page / Component / Hook / Service / Schema / Type / Other based on atlas-ui's `src/` layout.
- **Phase 3 checklists**: FE-01 through FE-18 verbatim from home-hub. The FE-* IDs are derived from the shared frontend-dev-guidelines skill.
- **Output**: `docs/tasks/<task-folder>/audit.md` (append) for feature-branch context; `docs/audits/frontend/audit.md` standalone.

### 3.8 `.claude/agents/todo-scanner.md` (new)

Absorbs the body of `commands/review-todos.md`. Verbatim from home-hub with s/Home Hub/Atlas/. Three-phase: parallel discovery (TODO markers, unimplemented stubs, project-structure analysis) → categorize by service + priority → update `docs/TODO.md` using the home-hub template. Atlas has more services (~26 under `services/`) so the "Services" section of the output will be longer.

### 3.9 `.claude/agents/service-documentation.md` (rename + reformat)

Renamed from `agents/documentation.md`. Gains full frontmatter per home-hub's `service-documentation.md`. Absorbs the body of `commands/service-doc.md`. Strict rules (code as single source of truth, no inference, no code modification, follow `DOCS.md`) kept verbatim from the current atlas agent. Argument shape: service name (`atlas-account`) or service path (`services/atlas-account`); resolve to `services/<name>/`.

### 3.10 `.claude/commands/audit-plan.md` (shrink)

```
---
description: Verify a plan was faithfully implemented — dispatches the plan-adherence-reviewer agent
argument-hint: Task folder name under docs/tasks/ (e.g., "task-016-superpowers-integration")
---

Dispatch the `plan-adherence-reviewer` agent against the task folder: **$ARGUMENTS**.

Pass the task folder path so the agent can locate `plan.md`, run the audit, and write findings to `docs/tasks/$ARGUMENTS/audit.md`.

After the agent completes, summarize the findings to the user — completion rate, blocking issues, and recommended next steps.
```

### 3.11 `.claude/commands/backend-audit.md`, `review-todos.md`, `service-doc.md` (shrink)

Same thin-wrapper pattern — verbatim from home-hub (file paths/IDs already match).

### 3.12 `.claude/commands/dev-docs.md`, `dev-docs-update.md` (delete)

Role split between `/design-task` and `/plan-task`. Removal is a conventional `git rm`. The corresponding superpowers skills (`superpowers:execute-plan`, `superpowers:write-plan`, `superpowers:brainstorm` are marked deprecated aliases in the plugin) are untouched — the removal here is atlas's own commands, not the plugin's.

### 3.13 `.claude/skills/skill-rules.json` (edit)

Add a `frontend-dev-guidelines` entry mirroring home-hub's, with atlas-ui path correction. **PRD §4.6 correction**: the PRD says `atlas-ui/**/*.ts` and `atlas-ui/**/*.tsx` at repo root, but atlas-ui actually lives at `services/atlas-ui/`. Use `services/atlas-ui/**/*.ts` and `services/atlas-ui/**/*.tsx` instead. The cross-cutting globs (`**/components/**/*.tsx`, `**/pages/**/*.tsx`, `**/lib/hooks/**/*.ts`, `**/lib/schemas/**/*.ts`, `**/services/api/**/*.ts`) stay as-is — they match regardless of where atlas-ui lives because they're unanchored.

Keywords: `frontend`, `react`, `tsx`, `component`, `hook`, `form`, `tailwind`, `shadcn`, `react query`, `tanstack`, `zod` (verbatim from home-hub).

Intent patterns: verbatim from home-hub.

Exclusions: `**/*.test.ts`, `**/*.test.tsx`.

Enforcement: `suggest`, priority `high`, same as the existing backend entry.

### 3.14 `CLAUDE.md` (additive edit)

Add one new top-level section (placement: between existing "Code Patterns" and "Documentation"). Body mirrors home-hub's `CLAUDE.md` section exactly, with s/Home Hub/Atlas/ and removal of home-hub's "Local Deployment" mention. Three subsections:

1. **Development Workflow** — describes the four-phase flow; each phase invoked from a `/clear`'d session.
2. **Artifact Location Override** — superpowers defaults to `docs/superpowers/specs/` and `docs/superpowers/plans/`; atlas uses `docs/tasks/task-NNN-slug/` instead. When invoking the skills directly (outside the phase commands), pass the task folder explicitly.
3. **Code Review Pattern** — three modular reviewer agents, dispatched in parallel via `superpowers:requesting-code-review` or individually by name. Each writes to `docs/tasks/task-NNN-slug/audit.md`.

Existing Workflow Rules / Build & Verification / Code Patterns / Documentation sections remain untouched.

### 3.15 `docs/superpowers-integration.md` (new)

Copied from home-hub's version (76 lines), with these atlas-specific substitutions:

- Title line + intro paragraph: s/Home Hub/Atlas/.
- Maintenance commands table: remove the `recipe-to-cooklang` row; add `convert-map`, `convert-npc`, `convert-portal`, `convert-quest`, `convert-reactor` rows under a "Domain Converters" subsection (no underlying agent, direct commands).
- File locations cheat sheet: s|recipes/|n/a| ; add `services/atlas-ui/` row for the frontend code location.
- Domain skills: unchanged (names match).
- Superpowers skills (self-activating): unchanged.
- When-NOT-to-use section: swap home-hub-specific example ("Personal recipe conversion") for atlas-relevant ("Running a domain-specific converter like `/convert-npc`").

## 4. Migration mechanics

### 4.1 `dev/active/` → `docs/tasks/legacy-<slug>/` (52 folders)

Per-folder procedure:

1. `git mv dev/active/<slug> docs/tasks/legacy-<slug>`
2. For each inner file matching the `<slug>-<suffix>.md` pattern, `git mv docs/tasks/legacy-<slug>/<slug>-<suffix>.md docs/tasks/legacy-<slug>/<suffix>.md`. Common suffixes: `-context.md`, `-plan.md`, `-tasks.md`.
3. Inspect inner cross-references (a plan referencing its own context by `<slug>-context.md`, etc.) and rewrite to the new bare names.
4. Non-standard inner files (README, notes, ad-hoc markdown) keep their names after the folder rename.
5. Commit: `refactor: migrate dev/active/<slug> to docs/tasks/legacy-<slug>` (one commit per folder).

**Edge case**: `dev/active/account-deletion-feature` may not follow the `<slug>-<suffix>` convention. Inspect first; if files use flat naming already, the folder rename is sufficient.

After all folders migrated: `rmdir dev/active` (via `git rm` of the empty directory — expected to be already handled because git does not track empty directories).

**Verification**:
- `git ls-files dev/active/` returns empty.
- `git log --follow -- docs/tasks/legacy-redis-registry-migration/context.md` shows pre-rename history crossing into `dev/active/redis-registry-migration/redis-registry-migration-context.md`.

**Commit count**: ~52 folder migrations. This is deliberate granularity — preserves per-task `git log --follow` clarity and keeps each commit's diff reviewable.

### 4.2 `dev/audits/` deletion (25 folders)

One commit: `chore: remove dev/audits (obsoleted by in-task and per-service audit locations)`. Body explains that audit artifacts now live at `docs/tasks/<task-folder>/audit.{md,json}` (feature-bound) or `docs/audits/<service>/audit.{md,json}` (standalone per-service). The `dev/audits/*` content is not migrated — those are historical `/backend-audit` outputs with no ongoing value.

**Verification**: `git ls-files dev/audits/` returns empty.

### 4.3 Reference sweep

After the folder migrations land, run:

```
git grep -E 'dev/(active|audits)/'
```

Update every match except this task's own PRD (which describes the migration itself and must keep its references). Known references:

- `CLAUDE.md` — any mentions of `dev/active/` or `dev/audits/`.
- `docs/TODO.md` — any mentions.
- Any other markdown under `docs/`.
- `~/.claude/projects/-<workspace-pers-atlas>/memory/MEMORY.md` — currently references `dev/active/redis-registry-migration/`, `dev/active/automatic-tenant-filtering/`, `dev/active/saga-orchestrator-durability/`, `dev/active/character-shop-merchant/`, `dev/active/writer-packet-extraction/`.
- Auto-memory sibling files under the same directory that the index points to (e.g., `redis-migration.md`) — grep those too for `dev/active/` references to the slug they describe.

Rewrite rule: `dev/active/<slug>/` → `docs/tasks/legacy-<slug>/`; file-inside references (`<slug>-context.md` etc.) update to the bare-name form per §4.1.

Commit: `docs: update references from dev/active and dev/audits to new locations` (one commit covering the sweep).

**Acceptance**: `git grep -E 'dev/(active|audits)/' -- ':!docs/tasks/task-016-superpowers-integration/'` returns empty.

## 5. Verification plan

Per PRD §8.4, there are five verification layers. This design specifies how each is invoked during `plan.md` execution:

1. **File-presence + frontmatter**: Read tool on each new/renamed agent and command file. Verify YAML frontmatter parses and contains the required keys (`name`, `description` with at least one `<example>` block, `model: inherit` for agents; `description`, optional `argument-hint` for commands).

2. **JSON validity**: `python3 -m json.tool < .claude/skills/skill-rules.json` returns zero exit code.

3. **Hook smoke test**: Pipe a sample frontend-flavored prompt through the hook. The hook scripts (`hooks/skill-activation-prompt.sh|py`) are untouched; this test only verifies that the new skill-rules entry is picked up.
   ```
   echo '{"prompt":"I need to add a new React component with a Zod form"}' | python3 .claude/hooks/skill-activation-prompt.py
   ```
   Expected output: contains "🎯 SKILL ACTIVATION CHECK" and names `frontend-dev-guidelines`.

4. **End-to-end workflow smoke test**: After all files land, on a throwaway branch, run the four-phase flow against a trivial throwaway idea. Expected: `docs/tasks/task-017-xxx/prd.md` from `/spec-task`, `design.md` from `/design-task`, `plan.md` + `context.md` from `/plan-task`, and either code changes or an explicit skip from `/execute-task`. Then `git reset --hard` the throwaway branch away.

5. **Migration verification**: `git log --follow` on one migrated file crosses the rename; `git ls-files dev/active/` and `git ls-files dev/audits/` both empty; `git grep -E 'dev/(active|audits)/' -- ':!docs/tasks/task-016-superpowers-integration/'` empty.

## 6. Resolution of PRD §9 Open Questions

| # | Question | Resolution |
|---|----------|------------|
| 1 | Exact wording for CLAUDE.md workflow section | Mirror home-hub's wording verbatim with s/Home Hub/Atlas/ and drop the "Local Deployment" subsection (atlas uses different local deploy). Additive — no existing section is rewritten. |
| 2 | How `/design-task` and `/plan-task` pass the task folder into their underlying superpowers skills | Preamble prompt (§2.1). Command body instructs main agent to pass task-folder-specific overrides as natural-language context inside the Skill-tool invocation. |
| 3 | Whether `frontend-guidelines-reviewer` targets only atlas-ui or also other TS/React | atlas-ui only. atlas-ui is the sole frontend in the repo (per PRD §7 and `services/` inspection). |
| 4 | Whether migrated `legacy-<slug>/` folders get a `legacy.md` marker | No. The folder-name prefix `legacy-` is self-documenting. Adding a marker file would be churn with no reader benefit. |

## 7. Additional design decisions (beyond §9)

### 7.1 PRD §4.6 correction — atlas-ui path

PRD §4.6 lists skill-rules path patterns `atlas-ui/**/*.ts` and `atlas-ui/**/*.tsx`. The actual location is `services/atlas-ui/`. Design uses `services/atlas-ui/**/*.ts` and `services/atlas-ui/**/*.tsx`. The PRD will be updated in-line as part of this task's implementation (no separate amendment doc) because the PRD remains the canonical requirements source.

### 7.2 atlas-ui test runner

atlas-ui uses Vitest (`npm test` → `vitest run`), not Jest. The `frontend-guidelines-reviewer` agent's Phase 1 gate uses `npm test` (which invokes Vitest) — not home-hub's `npm test -- --watchAll=false`. This is a single-line divergence in the agent body.

### 7.3 Migration commit granularity

One commit per `dev/active/<slug>` folder migration (~52 commits). Rationale:
- Preserves per-task `git log --follow` clarity for future archaeology.
- Each commit diff stays reviewable at a glance.
- Matches the home-hub precedent of preferring small, focused refactor commits.

Alternative considered: one big `refactor: migrate dev/active to docs/tasks/legacy-*` commit. Rejected — would bundle 52 unrelated task-folder renames into one diff and obscure per-task history. The extra commit count does not meaningfully affect repo size or `git log` readability.

### 7.4 Subagent model for reviewer agents

Agents use `model: inherit` (copied from home-hub and from the superpowers code-reviewer reference). This means each agent invocation uses the same model as the parent session. No per-agent model override; no Haiku downgrade for cheaper audits. Rationale: the adversarial-audit work benefits from the strongest model available, and `inherit` keeps the setup minimal.

### 7.5 Reviewer-trio orchestration mechanics

`superpowers:requesting-code-review` is the existing orchestrator skill from the plugin; no modification needed. The design relies on that skill's default behavior, which dispatches sub-agents based on repository-configured reviewer agents matching the changed file extensions. Atlas registers three reviewers under `.claude/agents/`; the orchestrator picks them up by convention. No custom dispatch wrapper.

### 7.6 `tasks.md` forward policy

Per PRD §4.11: the new workflow does not produce `tasks.md`. Existing `tasks.md` files (in `docs/tasks/task-001`…`task-015` and in migrated `legacy-<slug>/` folders) remain as historical artifacts — not deleted, not migrated. `CLAUDE.md`'s new section documents the new canonical file set (`prd.md`, `design.md`, `plan.md`, `context.md`, optional `audit.md`/`audit.json`) without retroactively pruning old `tasks.md` files.

### 7.7 `requesting-code-review` subset selection

The orchestrator decides which of the three reviewers to dispatch based on `git diff main...HEAD` file types. Expected behaviors:

| Branch contents | Dispatched reviewers |
|---|---|
| Only Go changes + plan.md | plan-adherence-reviewer, backend-guidelines-reviewer |
| Only atlas-ui TS changes + plan.md | plan-adherence-reviewer, frontend-guidelines-reviewer |
| Mixed changes + plan.md | all three |
| No plan.md (ad-hoc branch) | only the guideline reviewer(s) matching changed files |
| Only docs/config changes (no code) | plan-adherence-reviewer only |

Each reviewer writes to `docs/tasks/<task-folder>/audit.md` (append). The orchestrator aggregates into one summary response.

## 8. Out-of-scope confirmations (from PRD §2)

Restating what this design does NOT touch:

- superpowers skills themselves (in `~/.claude/plugins/cache/claude-plugins-official/superpowers/5.0.7/`).
- `hooks/skill-activation-prompt.sh|py`.
- `skills/backend-dev-guidelines/resources/*` and `skills/frontend-dev-guidelines/resources/*`.
- `settings.json` (the plugin is already enabled).
- Any Go service under `services/*` other than `services/atlas-ui/` (and atlas-ui's code is not modified — only its path is referenced).
- `libs/*`.
- The five `convert-*` commands (atlas-specific domain converters, untouched).
- `docs/tasks/task-001`…`task-015` existing content.

## 9. Next step

This design is ready to hand off to `/plan-task task-016-superpowers-integration`. The plan will decompose this design into bite-sized, independently-verifiable tasks suitable for subagent-driven execution, structured roughly as:

1. New agent files (5 tasks, one per agent).
2. Thin-wrapper commands (4 tasks, one per shrunk command).
3. New phase commands (3 tasks, one per new command).
4. `/spec-task` handoff edit (1 task).
5. `skill-rules.json` frontend entry (1 task).
6. `CLAUDE.md` section (1 task).
7. `docs/superpowers-integration.md` (1 task).
8. `dev/active/` migration (~52 tasks, one per folder, or grouped by shape).
9. `dev/audits/` deletion (1 task).
10. Reference sweep (1 task).
11. Deletions of `dev-docs.md`, `dev-docs-update.md`, and rename of `documentation.md` → `service-documentation.md` (bundled with the agent-creation tasks).
12. End-to-end verification (1 task).

Exact grouping and ordering is the plan phase's job.

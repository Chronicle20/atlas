# Superpowers Integration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Integrate the superpowers plugin as Atlas's default development workflow, promoting five reviewer/maintenance agents to proper `.claude/agents/` files, adding three phase commands, migrating 52 `dev/active/` folders to `docs/tasks/legacy-<slug>/`, and deleting `dev/audits/`.

**Architecture:** Phase commands invoke superpowers skills via the Skill tool, passing task-folder overrides as natural-language preamble. Reviewer agents are invokable by name or via `superpowers:requesting-code-review`. Home-hub's `task-044-superpowers-integration` is the verbatim reference implementation — atlas applies a small set of predictable substitutions (see `context.md` → "Adaptation rules").

**Tech Stack:** Shell + git + Python (for JSON validation). No Go or TypeScript code is produced or modified by this plan. Agent/command files are Markdown with YAML frontmatter.

---

## Pre-flight checklist

Before Task 1, confirm:

- [ ] Current branch is `task-016-superpowers-integration` (`git branch --show-current`).
- [ ] `docs/tasks/task-016-superpowers-integration/{prd.md,design.md,context.md}` all exist and are committed.
- [ ] `.claude/settings.json` has `enabledPlugins.superpowers@claude-plugins-official: true` on HEAD.
- [ ] Reference files under `~/source/pers/home-hub/.claude/` and `~/source/pers/home-hub/docs/superpowers-integration.md` are readable.
- [ ] You understand the atlas-vs-home-hub adaptation rules in `context.md`.

---

## File Structure

### Creating (9 files)

```
.claude/agents/plan-adherence-reviewer.md           # Task 1
.claude/agents/backend-guidelines-reviewer.md       # Task 2
.claude/agents/frontend-guidelines-reviewer.md      # Task 3
.claude/agents/todo-scanner.md                      # Task 4
.claude/agents/service-documentation.md             # Task 5 (replaces documentation.md)
.claude/commands/design-task.md                     # Task 9
.claude/commands/plan-task.md                       # Task 10
.claude/commands/execute-task.md                    # Task 11
docs/superpowers-integration.md                     # Task 15
```

### Modifying (6 files)

```
.claude/commands/spec-task.md                       # Task 12 — handoff text edit
.claude/commands/audit-plan.md                      # Task 6 — shrink to thin wrapper
.claude/commands/backend-audit.md                   # Task 7 — shrink to thin wrapper
.claude/commands/review-todos.md                    # Task 8a — shrink to thin wrapper
.claude/commands/service-doc.md                     # Task 8b — shrink to thin wrapper
.claude/skills/skill-rules.json                     # Task 13 — add frontend entry
CLAUDE.md                                           # Task 14 — additive Workflow section
```

### Deleting (3 files)

```
.claude/agents/documentation.md                     # Task 5 (replaced by service-documentation.md)
.claude/commands/dev-docs.md                        # Task 16
.claude/commands/dev-docs-update.md                 # Task 16
```

### Moving (52 folders + 1 directory deletion)

```
dev/active/<slug>/                                  # Task 17 — 52 folders → docs/tasks/legacy-<slug>/
dev/audits/                                         # Task 18 — removed outright
```

### Reference sweep (1 commit covering N files)

```
Tracked markdown referencing dev/(active|audits)/   # Task 19
~/.claude/projects/-<workspace-pers-atlas>/memory/*.md  # Task 20
```

### End-to-end verification

```
Throwaway four-phase smoke test                     # Task 21
```

---

## Task 1: Create `plan-adherence-reviewer` agent

**Files:**
- Create: `.claude/agents/plan-adherence-reviewer.md`
- Reference: `~/source/pers/home-hub/.claude/agents/plan-adherence-reviewer.md`

**Adaptation rules applied:**
- s/Home Hub/Atlas/ throughout.
- Phase 4 Build & Test Verification uses atlas service path depth: `cd services/<service>/atlas.com/<module> && go build ./... && go test ./... -count=1`.
- For frontend changes, `cd services/atlas-ui && npm run build && npm test` (not `cd frontend`).
- Preserve the five-section audit.md template verbatim (executive summary, task table, skipped/deferred, build & test, overall assessment).

- [ ] **Step 1: Copy home-hub reference as a starting point**

```bash
cp ~/source/pers/home-hub/.claude/agents/plan-adherence-reviewer.md .claude/agents/plan-adherence-reviewer.md
```

- [ ] **Step 2: Apply atlas adaptations**

Edit the file. Changes required (use Edit tool):

1. Replace "Home Hub project" → "Atlas project" (appears once in the agent body).
2. In the Step 4 Build & Test Verification section, change:
   - `Run 'go build ./...' (or the appropriate build command) from the service directory.` → `For each affected Go service, run 'go build ./...' and 'go test ./... -count=1' from 'services/<service>/atlas.com/<module>/' (atlas's deeper service path).`
   - `For frontend changes, run 'npm run build' and 'npm test' from the relevant app directory.` → `For atlas-ui changes, run 'npm run build' and 'npm test' from 'services/atlas-ui/'.`

- [ ] **Step 3: Verify frontmatter parses**

```bash
head -20 .claude/agents/plan-adherence-reviewer.md
```

Expected: first line is `---`, then `name: plan-adherence-reviewer`, then `description: |` followed by multi-line description with at least one `<example>` block, then `model: inherit`, then closing `---`.

- [ ] **Step 4: Commit**

```bash
git add .claude/agents/plan-adherence-reviewer.md
git commit -m "feat(agents): add plan-adherence-reviewer

Verifies every task in a plan.md was actually implemented, with file:line
evidence. Writes audit report to docs/tasks/<task-folder>/audit.md.
Adapted from home-hub/task-044 with atlas service path depth."
```

---

## Task 2: Create `backend-guidelines-reviewer` agent

**Files:**
- Create: `.claude/agents/backend-guidelines-reviewer.md`
- Reference: `~/source/pers/home-hub/.claude/agents/backend-guidelines-reviewer.md`

**Adaptation rules applied:**
- s/Home Hub/Atlas/ throughout.
- Phase 1 build/test commands use `cd services/<service>/atlas.com/<module>` (not `cd <service-path>`).
- Phase 2 Domain Discovery walks `<service-path>/atlas.com/<module>/internal/` (not `<service-path>/internal/`).
- Phase 5 audit output: `docs/audits/<service-name>/audit.{md,json}` for standalone; `docs/tasks/<task-folder>/audit.{md,json}` for feature-branch context.
- DOM-01 through DOM-20, SUB-01 through SUB-04, SEC-01 through SEC-04: **verbatim from home-hub** — these IDs are content-derived from the shared backend-dev-guidelines skill.
- Guideline resource paths unchanged (`.claude/skills/backend-dev-guidelines/resources/*.md`).

- [ ] **Step 1: Copy home-hub reference**

```bash
cp ~/source/pers/home-hub/.claude/agents/backend-guidelines-reviewer.md .claude/agents/backend-guidelines-reviewer.md
```

- [ ] **Step 2: Apply atlas adaptations**

Edit the file:

1. Replace "Home Hub microservice platform" → "Atlas microservice platform".
2. Example in frontmatter: `services/recipe-service` → `services/atlas-account` (or any real atlas service; pick one that exists).
3. Example section `Example: services/auth-service` → `services/atlas-login`.
4. Phase 0 Setup step 1: `services/auth-service` → `services/atlas-login`; update `last path segment` explanation to note atlas's depth:
   - Old: `Derive 'service-name' as the last path segment of the service path (e.g., 'services/auth-service' → 'auth-service').`
   - New: `Derive 'service-name' as the top-level service directory name under 'services/' (e.g., 'services/atlas-login/atlas.com/login' → 'atlas-login').`
5. Phase 1 Build & Test — replace the two code fences:
   ```
   cd <service-path> && go build ./...
   cd <service-path> && go test ./... -count=1
   ```
   with:
   ```
   cd <service-path>/atlas.com/<module> && go build ./...
   cd <service-path>/atlas.com/<module> && go test ./... -count=1
   ```
   (where `<module>` is the single directory name directly under `<service-path>/atlas.com/` — each atlas service typically has one).
6. Phase 2 Domain Discovery step 1: `<service-path>/internal/` → `<service-path>/atlas.com/<module>/internal/`.
7. Phase 5 first paragraph: update the file-path reasoning if the example paths mention home-hub specifics. Keep the audit location rules unchanged.

- [ ] **Step 3: Verify frontmatter and a spot-check on the DOM checklist**

```bash
head -20 .claude/agents/backend-guidelines-reviewer.md
grep -c '| DOM-' .claude/agents/backend-guidelines-reviewer.md
```

Expected: frontmatter valid (same shape as Task 1); DOM-* rows count = 20.

- [ ] **Step 4: Commit**

```bash
git add .claude/agents/backend-guidelines-reviewer.md
git commit -m "feat(agents): add backend-guidelines-reviewer

Adversarial Go auditor running DOM-*, SUB-*, and SEC-* checklists against
changed packages. Default-FAIL mindset; every PASS requires file:line.
Adapted from home-hub/task-044 for atlas's services/<svc>/atlas.com/<module>
service path depth."
```

---

## Task 3: Create `frontend-guidelines-reviewer` agent

**Files:**
- Create: `.claude/agents/frontend-guidelines-reviewer.md`
- Reference: `~/source/pers/home-hub/.claude/agents/frontend-guidelines-reviewer.md`

**Adaptation rules applied:**
- s/Home Hub/Atlas/.
- `frontend/` → `services/atlas-ui/` throughout.
- Build command: `cd services/atlas-ui && npm run build` (unchanged from home-hub's pattern, but path is deeper).
- Test command: `cd services/atlas-ui && npm test` (atlas-ui uses Vitest — this resolves to `vitest run`). Do NOT append `-- --watchAll=false` (that's a Jest flag, unsupported by Vitest).
- FE-01 through FE-18: **verbatim from home-hub**.
- Guideline resource paths unchanged (`.claude/skills/frontend-dev-guidelines/SKILL.md` + `resources/*.md`).
- Phase 2 File Inventory paths: `pages/*.tsx`, `components/**/*.tsx`, `lib/hooks/api/*.ts`, `services/api/*.ts`, `lib/schemas/*.ts`, `types/**/*.ts` — these are atlas-ui's `src/` subpaths and match home-hub's. Leave verbatim.

- [ ] **Step 1: Copy home-hub reference**

```bash
cp ~/source/pers/home-hub/.claude/agents/frontend-guidelines-reviewer.md .claude/agents/frontend-guidelines-reviewer.md
```

- [ ] **Step 2: Apply atlas adaptations**

Edit the file:

1. Replace "Home Hub UI" → "Atlas UI".
2. Frontmatter description example: `frontend/src/pages and frontend/src/services` → `services/atlas-ui/src/pages and services/atlas-ui/src/services`.
3. Frontmatter description example path `frontend/src` → `services/atlas-ui/src`.
4. Input section: `A frontend path (e.g., 'frontend/src')` → `An atlas-ui path (e.g., 'services/atlas-ui/src')`.
5. Phase 1 Build & Test replace:
   ```
   cd frontend && npm run build
   cd frontend && npm test -- --watchAll=false
   ```
   with:
   ```
   cd services/atlas-ui && npm run build
   cd services/atlas-ui && npm test
   ```
   Append a note after the code block: `Note: atlas-ui uses Vitest (via 'npm test' → 'vitest run'), not Jest. The '--watchAll' flag does not apply.`
6. Phase 4 standalone audit path: `docs/audits/frontend/audit.md` → `docs/audits/atlas-ui/audit.md`.

- [ ] **Step 3: Verify frontmatter and a spot-check on the FE checklist**

```bash
head -20 .claude/agents/frontend-guidelines-reviewer.md
grep -c '| FE-' .claude/agents/frontend-guidelines-reviewer.md
```

Expected: frontmatter valid; FE-* rows count = 18.

- [ ] **Step 4: Commit**

```bash
git add .claude/agents/frontend-guidelines-reviewer.md
git commit -m "feat(agents): add frontend-guidelines-reviewer

Adversarial atlas-ui auditor running FE-* anti-pattern, architecture, and
testing checklists. Default-FAIL mindset; cites file:line for every verdict.
Adapted from home-hub/task-044 for atlas-ui's services/atlas-ui/ location
and Vitest test runner."
```

---

## Task 4: Create `todo-scanner` agent

**Files:**
- Create: `.claude/agents/todo-scanner.md`
- Reference: `~/source/pers/home-hub/.claude/agents/todo-scanner.md`

**Adaptation rules applied:**
- s/Home Hub/Atlas/.
- All other content verbatim.

- [ ] **Step 1: Copy home-hub reference**

```bash
cp ~/source/pers/home-hub/.claude/agents/todo-scanner.md .claude/agents/todo-scanner.md
```

- [ ] **Step 2: Apply adaptations**

Edit: replace all instances of "Home Hub" with "Atlas" (including in the TODO.md header template: `# Home Hub Project TODO` → `# Atlas Project TODO`).

- [ ] **Step 3: Verify frontmatter**

```bash
head -20 .claude/agents/todo-scanner.md
```

Expected: frontmatter valid.

- [ ] **Step 4: Commit**

```bash
git add .claude/agents/todo-scanner.md
git commit -m "feat(agents): add todo-scanner

Whole-repo TODO/FIXME/XXX/HACK scan; categorizes by service and priority;
updates docs/TODO.md. Isolates heavy file-scanning from the main context.
Adapted verbatim from home-hub/task-044."
```

---

## Task 5: Replace `agents/documentation.md` with `agents/service-documentation.md`

**Files:**
- Create: `.claude/agents/service-documentation.md`
- Delete: `.claude/agents/documentation.md`
- Reference: `~/source/pers/home-hub/.claude/agents/service-documentation.md`

**Adaptation rules applied:**
- s/Home Hub/Atlas/ in both frontmatter description and body ("You are the Home Hub Documentation Agent" → "You are the Atlas Documentation Agent").
- Argument-shape example: `auth-service` → `atlas-account`.
- Argument-shape example service path: `services/auth-service` → `services/atlas-account`.
- Strict-rules block (MUST / MUST NOT lists) is verbatim.
- `DOCS.md` reference preserved — atlas has its own `DOCS.md` contract.

- [ ] **Step 1: Copy home-hub reference**

```bash
cp ~/source/pers/home-hub/.claude/agents/service-documentation.md .claude/agents/service-documentation.md
```

- [ ] **Step 2: Apply adaptations**

Edit: replace "Home Hub" with "Atlas" throughout (both in frontmatter `<example>` blocks and in body "You are the Home Hub Documentation Agent"). Replace `auth-service` with `atlas-account` in the example and argument-shape description.

- [ ] **Step 3: Delete the old `agents/documentation.md`**

```bash
git rm .claude/agents/documentation.md
```

- [ ] **Step 4: Verify**

```bash
ls .claude/agents/
head -20 .claude/agents/service-documentation.md
```

Expected: `documentation.md` absent; `service-documentation.md` present with valid frontmatter.

- [ ] **Step 5: Commit**

```bash
git add .claude/agents/service-documentation.md .claude/agents/documentation.md
git commit -m "refactor(agents): rename documentation → service-documentation with full frontmatter

Adds proper YAML frontmatter (name, description with <example> blocks,
model: inherit) so the agent is dispatchable by superpowers:requesting-code-review
and directly invokable by name. Adapted from home-hub/task-044."
```

---

## Task 6: Shrink `commands/audit-plan.md` to thin wrapper

**Files:**
- Modify: `.claude/commands/audit-plan.md`
- Reference: `~/source/pers/home-hub/.claude/commands/audit-plan.md`

The body shrinks from ~60 lines (full inline audit logic) to ~10 lines (dispatch to `plan-adherence-reviewer`).

- [ ] **Step 1: Replace the file with the thin-wrapper form**

Use the Write tool (the file is a full rewrite). Contents:

```markdown
---
description: Verify a plan was faithfully implemented — dispatches the plan-adherence-reviewer agent
argument-hint: Task folder name under docs/tasks/ (e.g., "task-016-superpowers-integration")
---

Dispatch the `plan-adherence-reviewer` agent against the task folder: **$ARGUMENTS**.

Pass the task folder path so the agent can locate `plan.md`, run the audit, and write findings to `docs/tasks/$ARGUMENTS/audit.md`.

After the agent completes, summarize the findings to the user — completion rate, blocking issues, and recommended next steps.
```

- [ ] **Step 2: Verify**

```bash
wc -l .claude/commands/audit-plan.md
head -5 .claude/commands/audit-plan.md
```

Expected: ~10 lines (was ~60); frontmatter present.

- [ ] **Step 3: Commit**

```bash
git add .claude/commands/audit-plan.md
git commit -m "refactor(commands): shrink audit-plan to plan-adherence-reviewer wrapper

Body now dispatches the agent. Keeps the /audit-plan slash command for
autocomplete discoverability."
```

---

## Task 7: Shrink `commands/backend-audit.md` to thin wrapper

**Files:**
- Modify: `.claude/commands/backend-audit.md`

- [ ] **Step 1: Replace the file**

Write tool, contents:

```markdown
---
description: Adversarially audit a Go service against backend developer guidelines — dispatches the backend-guidelines-reviewer agent
argument-hint: Path to the service to audit (e.g., "services/atlas-account")
---

Dispatch the `backend-guidelines-reviewer` agent against: **$ARGUMENTS**.

Pass the service path so the agent can run the build/test gate, the DOM-* / SUB-* checklists, and (if auth-related) SEC-* checks. The agent writes `audit.md` and `audit.json` under `docs/audits/<service-name>/` (or under the active task folder if invoked from a feature branch with a `plan.md`).

After the agent completes, summarize PASS / NEEDS-WORK / FAIL status and any blocking items.
```

- [ ] **Step 2: Verify**

```bash
wc -l .claude/commands/backend-audit.md
```

Expected: ~10 lines.

- [ ] **Step 3: Commit**

```bash
git add .claude/commands/backend-audit.md
git commit -m "refactor(commands): shrink backend-audit to backend-guidelines-reviewer wrapper"
```

---

## Task 8: Shrink `review-todos.md` and `service-doc.md` to thin wrappers

Two files, one commit (both are trivial wrappers).

**Files:**
- Modify: `.claude/commands/review-todos.md`
- Modify: `.claude/commands/service-doc.md`

- [ ] **Step 1: Rewrite `review-todos.md`**

```markdown
---
description: Scan the codebase for TODO/FIXME markers and unimplemented stubs; updates docs/TODO.md — dispatches the todo-scanner agent
---

Dispatch the `todo-scanner` agent.

The agent runs a full-repo scan, categorizes findings by service and priority, and updates `docs/TODO.md`. After it completes, surface the summary it returns: total findings, top critical items, and services with the most concentrated incomplete work.
```

- [ ] **Step 2: Rewrite `service-doc.md`**

```markdown
---
description: Generate or update documentation for one Atlas service — dispatches the service-documentation agent
argument-hint: Service name or path (e.g., "atlas-account" or "services/atlas-account")
---

Dispatch the `service-documentation` agent against: **$ARGUMENTS**.

The agent treats code as the single source of truth, follows `DOCS.md`, and operates only within the target service directory. It outputs only updated doc files — no commentary, no analysis.
```

- [ ] **Step 3: Verify**

```bash
wc -l .claude/commands/review-todos.md .claude/commands/service-doc.md
```

Expected: ~8 lines each.

- [ ] **Step 4: Commit**

```bash
git add .claude/commands/review-todos.md .claude/commands/service-doc.md
git commit -m "refactor(commands): shrink review-todos and service-doc to thin wrappers

Both dispatch their corresponding agents (todo-scanner, service-documentation)
while preserving / slash-command discoverability."
```

---

## Task 9: Create `commands/design-task.md` (Phase 2)

**Files:**
- Create: `.claude/commands/design-task.md`
- Reference: `~/source/pers/home-hub/.claude/commands/design-task.md`

**Adaptation rules applied:**
- s/Home Hub/Atlas/.
- All content otherwise verbatim — the command body is a natural-language preamble that works identically between projects.

- [ ] **Step 1: Copy home-hub reference**

```bash
cp ~/source/pers/home-hub/.claude/commands/design-task.md .claude/commands/design-task.md
```

- [ ] **Step 2: Apply adaptations**

Edit: replace "Home Hub four-phase development workflow" → "Atlas four-phase development workflow" (appears once in the body).

- [ ] **Step 3: Verify frontmatter**

```bash
head -5 .claude/commands/design-task.md
```

Expected: `---`, `description: Phase 2 — invoke superpowers:brainstorming ...`, `argument-hint: ...`, `---`.

- [ ] **Step 4: Commit**

```bash
git add .claude/commands/design-task.md
git commit -m "feat(commands): add /design-task for Phase 2 (brainstorming)

Invokes superpowers:brainstorming with task-folder-override preamble so the
design lands in docs/tasks/<folder>/design.md instead of the skill's default
docs/superpowers/specs/ location. No auto-chain to /plan-task; user runs /clear
between phases."
```

---

## Task 10: Create `commands/plan-task.md` (Phase 3)

**Files:**
- Create: `.claude/commands/plan-task.md`
- Reference: `~/source/pers/home-hub/.claude/commands/plan-task.md`

- [ ] **Step 1: Copy**

```bash
cp ~/source/pers/home-hub/.claude/commands/plan-task.md .claude/commands/plan-task.md
```

- [ ] **Step 2: Apply adaptations**

Edit: replace "Home Hub four-phase development workflow" → "Atlas four-phase development workflow".

- [ ] **Step 3: Verify**

```bash
head -5 .claude/commands/plan-task.md
```

- [ ] **Step 4: Commit**

```bash
git add .claude/commands/plan-task.md
git commit -m "feat(commands): add /plan-task for Phase 3 (writing-plans)

Invokes superpowers:writing-plans with task-folder-override preamble. Outputs
plan.md + context.md under docs/tasks/<folder>/. Runs the skill's built-in
self-review before saving."
```

---

## Task 11: Create `commands/execute-task.md` (Phase 4)

**Files:**
- Create: `.claude/commands/execute-task.md`
- Reference: `~/source/pers/home-hub/.claude/commands/execute-task.md`

- [ ] **Step 1: Copy**

```bash
cp ~/source/pers/home-hub/.claude/commands/execute-task.md .claude/commands/execute-task.md
```

- [ ] **Step 2: Apply adaptations**

Edit: replace "Home Hub four-phase development workflow" → "Atlas four-phase development workflow".

- [ ] **Step 3: Verify**

```bash
head -5 .claude/commands/execute-task.md
```

- [ ] **Step 4: Commit**

```bash
git add .claude/commands/execute-task.md
git commit -m "feat(commands): add /execute-task for Phase 4 (subagent-driven execution)

Asks once for subagent-driven (default) vs inline execution, recommends a
git worktree if on main/master, invokes superpowers:subagent-driven-development
or superpowers:executing-plans. Hands off to finishing-a-development-branch
on completion and suggests requesting-code-review."
```

---

## Task 12: Update `/spec-task` handoff text

**Files:**
- Modify: `.claude/commands/spec-task.md` (Step 5 only)

Current Step 5 suggests "Run `/dev-docs task-NNN-slug`". New text points at `/design-task`.

- [ ] **Step 1: Edit the Step 5 handoff**

Use Edit tool. Replace:

```
2. Suggested next step (e.g., "Run `/dev-docs task-NNN-slug` to create an implementation plan")
```

with:

```
2. Suggested next step: "Now run `/clear` to reset context, then `/design-task task-NNN-slug` to invoke the brainstorming/design phase"
```

- [ ] **Step 2: Verify diff is minimal**

```bash
git diff .claude/commands/spec-task.md
```

Expected: one-line change in Step 5 only; no other edits.

- [ ] **Step 3: Commit**

```bash
git add .claude/commands/spec-task.md
git commit -m "refactor(spec-task): hand off to /design-task instead of /dev-docs

Completes the Phase 1 → Phase 2 connection in the new four-phase workflow.
/dev-docs will be removed in a later commit."
```

---

## Task 13: Add `frontend-dev-guidelines` entry to `skill-rules.json`

**Files:**
- Modify: `.claude/skills/skill-rules.json`
- Reference: `~/source/pers/home-hub/.claude/skills/skill-rules.json`

**Adaptation rule:** home-hub uses `frontend/**/*.ts` and `frontend/**/*.tsx`. Atlas must use `services/atlas-ui/**/*.ts` and `services/atlas-ui/**/*.tsx`. The other cross-cutting globs (`**/components/**/*.tsx`, etc.) are unchanged.

- [ ] **Step 1: Read current atlas file to understand the shape**

```bash
cat .claude/skills/skill-rules.json
```

- [ ] **Step 2: Insert the frontend entry**

Use Edit tool. Find the closing `}` of the `backend-dev-guidelines` entry (inside `skills`) and add the new entry after the trailing `,`. Replace:

```json
  "skills": {
    "backend-dev-guidelines": {
      ...existing entry...
    }
  },
```

with:

```json
  "skills": {
    "backend-dev-guidelines": {
      ...existing entry...
    },
    "frontend-dev-guidelines": {
      "type": "domain",
      "enforcement": "suggest",
      "priority": "high",
      "description": "Frontend development patterns for React/TypeScript",
      "promptTriggers": {
        "keywords": [
          "frontend",
          "react",
          "tsx",
          "component",
          "hook",
          "form",
          "tailwind",
          "shadcn",
          "react query",
          "tanstack",
          "zod"
        ],
        "intentPatterns": [
          "(create|add|implement|build).*?(component|page|hook|form|dialog|table)",
          "(fix|handle|debug).*?(render|hydration|state|hook)",
          "(add|implement).*?(validation|schema|zod|form)",
          "(organize|structure|refactor).*?(frontend|component|hook|page)",
          "(how to|best practice).*?(frontend|component|hook|react|typescript)"
        ]
      },
      "fileTriggers": {
        "pathPatterns": [
          "services/atlas-ui/**/*.ts",
          "services/atlas-ui/**/*.tsx",
          "**/components/**/*.tsx",
          "**/pages/**/*.tsx",
          "**/lib/hooks/**/*.ts",
          "**/lib/schemas/**/*.ts",
          "**/services/api/**/*.ts"
        ],
        "pathExclusions": [
          "**/*.test.ts",
          "**/*.test.tsx"
        ],
        "contentPatterns": []
      }
    }
  },
```

(Preserve the exact existing `backend-dev-guidelines` contents — copy them verbatim; only add the new entry.)

- [ ] **Step 3: Validate JSON**

```bash
python3 -m json.tool < .claude/skills/skill-rules.json > /dev/null && echo "JSON valid"
```

Expected: `JSON valid`. If this fails, re-inspect for trailing-comma or brace issues.

- [ ] **Step 4: Hook smoke test**

```bash
echo '{"prompt":"I need to add a new React component with a Zod form"}' | python3 .claude/hooks/skill-activation-prompt.py
```

Expected: output contains `🎯 SKILL ACTIVATION CHECK` and mentions `frontend-dev-guidelines`. If the banner does not appear, inspect the hook script's parsing of skill-rules.json.

- [ ] **Step 5: Commit**

```bash
git add .claude/skills/skill-rules.json
git commit -m "feat(skill-rules): activate frontend-dev-guidelines on atlas-ui prompts/files

Adds a frontend-dev-guidelines entry so the hook fires on React/TS keywords
or on changes under services/atlas-ui/. Closes the gap where the skill
existed but was never triggered."
```

---

## Task 14: Add "Development Workflow" section to `CLAUDE.md`

**Files:**
- Modify: `CLAUDE.md`
- Reference: `~/source/pers/home-hub/CLAUDE.md` (Development Workflow + Artifact Location Override + Code Review Pattern sections)

Atlas's current `CLAUDE.md` has sections: Project Overview, Workflow Rules, Build & Verification, Code Patterns, Documentation. The new section is inserted between "Code Patterns" and "Documentation" and contains three subsections.

- [ ] **Step 1: Read current CLAUDE.md to locate insertion point**

```bash
cat CLAUDE.md
```

- [ ] **Step 2: Insert the new section using Edit**

Use Edit tool. Find the existing "## Documentation" heading and prepend the new section before it. Text to insert (before `## Documentation`):

```markdown
## Development Workflow

The canonical flow for any non-trivial change is four phases. Each phase is a separate slash command, each invoked from a fresh (`/clear`'d) session so the next phase consumes only the prior phase's documented artifacts:

1. `/spec-task <idea>` — interactive PRD interview. Output: `docs/tasks/task-NNN-slug/prd.md`.
2. `/clear`, then `/design-task <task-folder>` — invokes `superpowers:brainstorming` for architecture/tradeoffs. Output: `design.md` in same folder.
3. `/clear`, then `/plan-task <task-folder>` — invokes `superpowers:writing-plans` for bite-sized TDD steps. Output: `plan.md` + `context.md`.
4. `/clear`, then `/execute-task <task-folder>` — invokes `superpowers:subagent-driven-development` (default) or `superpowers:executing-plans` (fallback).

Skip `/spec-task` only for trivial fixes that don't warrant a PRD; document those directly via a brainstorming session.

### Artifact Location Override

Both `superpowers:brainstorming` and `superpowers:writing-plans` default to `docs/superpowers/specs/` and `docs/superpowers/plans/`. **In this project, both go under `docs/tasks/task-NNN-slug/` instead.** When invoking those skills directly (outside the phase commands), pass the task folder explicitly so artifacts land in the right place.

### Code Review Pattern

Code review uses three modular reviewer agents, dispatched in parallel:

- `plan-adherence-reviewer` — verifies plan tasks were actually implemented
- `backend-guidelines-reviewer` — Go DOM-* checklist (when Go files changed)
- `frontend-guidelines-reviewer` — TS/React FE-* checklist (when atlas-ui TS files changed)

Invoke via `superpowers:requesting-code-review` (it dispatches the appropriate subset), or invoke an individual agent directly for ad-hoc checks. Each agent writes its findings to `docs/tasks/task-NNN-slug/audit.md`.

See `docs/superpowers-integration.md` for a complete when-to-use-what reference.

```

- [ ] **Step 3: Verify existing sections are intact**

```bash
grep -c '^## ' CLAUDE.md
```

Expected: 6 top-level sections (Project Overview, Workflow Rules, Build & Verification, Code Patterns, Development Workflow, Documentation).

- [ ] **Step 4: Commit**

```bash
git add CLAUDE.md
git commit -m "docs(CLAUDE): add Development Workflow section

Documents the four-phase workflow (/spec-task → /design-task → /plan-task →
/execute-task), the docs/tasks/ artifact location override, and the modular
reviewer-agent pattern. Additive — existing sections preserved."
```

---

## Task 15: Create `docs/superpowers-integration.md`

**Files:**
- Create: `docs/superpowers-integration.md`
- Reference: `~/source/pers/home-hub/docs/superpowers-integration.md`

**Adaptation rules applied:**
- s/Home Hub/Atlas/.
- Maintenance commands table: drop the `recipe-to-cooklang` row; add a "Domain Converters" subsection listing atlas's five `convert-*` commands.
- File locations cheat sheet: drop the `recipes/` row; add a row for `services/atlas-ui/` as the frontend code location.
- When-NOT-to-use section: swap the recipe-conversion example for an atlas-specific domain-converter example.

- [ ] **Step 1: Copy home-hub reference**

```bash
cp ~/source/pers/home-hub/docs/superpowers-integration.md docs/superpowers-integration.md
```

- [ ] **Step 2: Apply adaptations**

Use Edit tool. Changes:

1. Title: `# Superpowers Integration — When to Use What` → unchanged.
2. First paragraph: `docs/tasks/task-044-superpowers-integration/design.md` → `docs/tasks/task-016-superpowers-integration/design.md`.
3. In the Maintenance Commands table, replace the `recipe-to-cooklang` row with:

    ```markdown
    | `/convert-map` | Convert map entry JavaScript script to JSON rules format | (direct command) |
    | `/convert-npc` | Convert NPC conversation JavaScript script to JSON state machine format | (direct command) |
    | `/convert-portal` | Convert portal JavaScript script to JSON rules format | (direct command) |
    | `/convert-quest` | Convert quest conversation JavaScript script to JSON state machine format | (direct command) |
    | `/convert-reactor` | Convert reactor JavaScript script to JSON rules format | (direct command) |
    ```

4. File Locations Cheat Sheet: remove the `Recipes | recipes/` row; add a row:

    ```markdown
    | atlas-ui frontend | `services/atlas-ui/` |
    ```

5. In the When-NOT-to-use section, replace:

    ```
    - **Personal recipe conversion** — use `/recipe-to-cooklang` directly.
    ```

    with:

    ```
    - **Domain script conversion** — use the appropriate `/convert-*` command directly (no workflow overhead).
    ```

- [ ] **Step 3: Verify the file looks right**

```bash
wc -l docs/superpowers-integration.md
grep -c 'convert-' docs/superpowers-integration.md
grep 'Home Hub' docs/superpowers-integration.md && echo "FAIL: found Home Hub reference" || echo "OK: no Home Hub references"
```

Expected: ~80 lines; 5 `convert-*` matches; no "Home Hub" references.

- [ ] **Step 4: Commit**

```bash
git add docs/superpowers-integration.md
git commit -m "docs: add superpowers-integration.md (when-to-use-what reference)

Quick-reference companion to CLAUDE.md. Lists the four phase commands, the
three reviewer agents, maintenance commands, domain converters, domain
skills, self-activating superpowers skills, and a when-NOT-to-use carve-out
for trivial fixes."
```

---

## Task 16: Delete `dev-docs.md` and `dev-docs-update.md`

**Files:**
- Delete: `.claude/commands/dev-docs.md`
- Delete: `.claude/commands/dev-docs-update.md`

These commands are replaced by `/design-task` + `/plan-task`.

- [ ] **Step 1: Delete both files**

```bash
git rm .claude/commands/dev-docs.md .claude/commands/dev-docs-update.md
```

- [ ] **Step 2: Verify**

```bash
ls .claude/commands/ | grep -E 'dev-docs'
```

Expected: no output (both files gone).

- [ ] **Step 3: Commit**

```bash
git commit -m "refactor(commands): remove dev-docs and dev-docs-update

Replaced by /design-task (architecture brainstorming) and /plan-task (plan
writing). Documented in CLAUDE.md and docs/superpowers-integration.md."
```

---

## Task 17: Migrate `dev/active/` to `docs/tasks/legacy-<slug>/`

**Files:**
- Move: every directory under `dev/active/` → `docs/tasks/legacy-<slug>/`
- Rename: every inner `<slug>-<suffix>.md` → `<suffix>.md`
- Delete: empty `dev/active/` directory at the end

**Scale:** 52 folders. One commit per folder. Use a shell loop for efficiency; inspect edge cases before running.

- [ ] **Step 1: Inventory and sanity-check the folders**

```bash
ls dev/active/ | wc -l
ls dev/active/
```

Expected: 52 folder names. Confirm none look unexpected (e.g., files rather than directories).

- [ ] **Step 2: Inspect the edge-case folder `account-deletion-feature`**

```bash
ls dev/active/account-deletion-feature/
```

If its inner files do NOT follow the `account-deletion-feature-*` pattern, note that — the rename step inside the loop will no-op on those files, which is correct (they stay under their original names).

- [ ] **Step 3: Run the migration loop**

Run in a shell. Each iteration does four things: `git mv` folder, rename inner prefixed files, rewrite in-file cross-references to the new names, commit.

```bash
set -e
for dir in dev/active/*/; do
  slug=$(basename "$dir")
  new_dir="docs/tasks/legacy-$slug"

  # 1. Move the folder
  git mv "dev/active/$slug" "$new_dir"

  # 2. Rename inner <slug>-<suffix>.md files to <suffix>.md
  for old_path in "$new_dir"/"$slug"-*.md; do
    [ -e "$old_path" ] || continue
    fname=$(basename "$old_path")
    new_fname=${fname#"$slug"-}
    git mv "$old_path" "$new_dir/$new_fname"
  done

  # 3. Rewrite in-file cross-references (sibling filenames) to the new bare names.
  # Use perl for in-place edit to avoid sed portability issues.
  for f in "$new_dir"/*.md; do
    [ -e "$f" ] || continue
    perl -i -pe "s|\\Q$slug-\\E([a-z0-9-]+\\.md)|\$1|g" "$f"
  done

  # 4. Stage all changes in this folder
  git add -A "$new_dir"
  # (the git mv commands above already staged the renames; this picks up the perl edits)

  # 5. Commit
  git commit -m "refactor: migrate dev/active/$slug to docs/tasks/legacy-$slug

Part of task-016 superpowers integration — consolidates feature artifacts
under docs/tasks/."
done
```

- [ ] **Step 4: Remove the now-empty `dev/active/` directory**

Git does not track empty directories, so removing the filesystem directory is enough — no commit needed for the directory itself.

```bash
rmdir dev/active
```

(If `rmdir` complains about non-empty, investigate — one of the loop iterations failed.)

- [ ] **Step 5: Verify the migration**

```bash
git ls-files dev/active/
```

Expected: empty.

```bash
git log --follow -- docs/tasks/legacy-redis-registry-migration/context.md | head -20
```

Expected: log shows pre-rename history, including a commit that touched `dev/active/redis-registry-migration/redis-registry-migration-context.md` (or similar pre-rename path).

```bash
ls docs/tasks/ | grep -c '^legacy-'
```

Expected: 52.

- [ ] **Step 6: Sanity-check one migrated folder**

```bash
ls docs/tasks/legacy-redis-registry-migration/
```

Expected: filenames like `context.md`, `plan.md`, `tasks.md` (no `redis-registry-migration-` prefix).

- [ ] **Step 7: No separate commit needed**

All 52 commits were made inside the loop. Verify with:

```bash
git log --oneline | grep -c 'migrate dev/active'
```

Expected: 52.

---

## Task 18: Delete `dev/audits/`

**Files:**
- Delete: all content under `dev/audits/`

- [ ] **Step 1: Inventory**

```bash
ls dev/audits/ | wc -l
```

Expected: 25.

- [ ] **Step 2: Delete**

```bash
git rm -r dev/audits/
```

- [ ] **Step 3: Verify**

```bash
git ls-files dev/audits/
ls dev/audits/ 2>/dev/null || echo "directory gone"
```

Expected: `git ls-files` empty; `ls` reports the directory is gone.

- [ ] **Step 4: Commit**

```bash
git commit -m "chore: remove dev/audits (obsoleted by in-task and per-service audit locations)

Going forward, audit artifacts live at docs/tasks/<task-folder>/audit.{md,json}
(feature-bound) or docs/audits/<service>/audit.{md,json} (standalone
per-service). The dev/audits/* content is historical /backend-audit output
with no ongoing value."
```

---

## Task 19: Reference sweep — tracked files

**Files:**
- Modify: any tracked file referencing `dev/active/<slug>/` or `dev/audits/<anything>/`.

**Exclusion**: `docs/tasks/task-016-superpowers-integration/` — its PRD and design describe the migration itself and must keep `dev/active` / `dev/audits` references.

- [ ] **Step 1: Inventory references**

```bash
git grep -E 'dev/(active|audits)/' -- ':!docs/tasks/task-016-superpowers-integration/'
```

Record the list. Expected candidates (from design §4.3):
- `CLAUDE.md` (may have none; inspect)
- `docs/TODO.md` (may or may not have references)
- Other markdown under `docs/` that points into the old locations

- [ ] **Step 2: Rewrite each reference**

For each match, apply the rewrite rule:

- `dev/active/<slug>/` → `docs/tasks/legacy-<slug>/`
- `dev/active/<slug>/<slug>-context.md` → `docs/tasks/legacy-<slug>/context.md` (and similarly for `-plan.md`, `-tasks.md`)
- `dev/audits/<anything>` → remove the reference or rewrite to `docs/audits/<service>/` if the audit described is ongoing (most `dev/audits` content is historical; prefer removal with a note explaining the reference is obsolete).

Use the Edit tool per file. For bulk rewrites in a single file:

```bash
# Example — replace one slug's old reference with new
perl -i -pe 's|dev/active/redis-registry-migration/redis-registry-migration-context\.md|docs/tasks/legacy-redis-registry-migration/context.md|g' path/to/file.md
```

Prefer targeted Edit calls over perl-in-place for predictability.

- [ ] **Step 3: Verify acceptance**

```bash
git grep -E 'dev/(active|audits)/' -- ':!docs/tasks/task-016-superpowers-integration/'
```

Expected: empty output.

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "docs: update references from dev/active and dev/audits to new locations

Sweeps CLAUDE.md, docs/TODO.md, and other tracked markdown so no reader
lands on a broken link. Task-016's PRD/design intentionally keep the old
paths (they describe the migration)."
```

---

## Task 20: Reference sweep — auto-memory files

**Files:**
- Modify: files under `~/.claude/projects/-<workspace-pers-atlas>/memory/` referencing `dev/active/`.

This sweep is OUTSIDE the repo — the auto-memory directory is user-level, not project-level. No git commit is made for these edits (the directory is not versioned inside this repo).

- [ ] **Step 1: Inventory references**

```bash
grep -rnE 'dev/(active|audits)/' ~/.claude/projects/-<workspace-pers-atlas>/memory/ || echo "none found"
```

Expected matches (per design §4.3 and MEMORY.md contents):
- `MEMORY.md` — 5 mentions: redis-registry-migration, automatic-tenant-filtering, saga-orchestrator-durability, character-shop-merchant, writer-packet-extraction.
- Possibly sibling files (e.g., `redis-migration.md`, `feedback_no_aliases.md`, `feedback_clean_dead_code.md`) if any reference `dev/active/` paths.

- [ ] **Step 2: Rewrite each reference**

Apply the same rewrite rule as Task 19:
- `dev/active/<slug>/` → `docs/tasks/legacy-<slug>/`

Use Edit tool per memory file.

- [ ] **Step 3: Verify**

```bash
grep -rE 'dev/(active|audits)/' ~/.claude/projects/-<workspace-pers-atlas>/memory/ || echo "OK: no references remain"
```

Expected: `OK: no references remain`.

- [ ] **Step 4: No commit**

Auto-memory is not versioned in this repo. No git action required.

---

## Task 21: End-to-end verification

A live smoke test of the four-phase workflow, the reviewer trio, and the domain-skill hook.

- [ ] **Step 1: Create a throwaway branch**

```bash
git checkout -b task-016-smoketest
```

- [ ] **Step 2: Hook smoke test (second time, from the clean branch)**

```bash
echo '{"prompt":"Let me refactor this React component and update the Zod schema"}' | python3 .claude/hooks/skill-activation-prompt.py
```

Expected: banner `🎯 SKILL ACTIVATION CHECK` mentioning `frontend-dev-guidelines`.

```bash
echo '{"prompt":"Add a new Go service endpoint with a Kafka consumer"}' | python3 .claude/hooks/skill-activation-prompt.py
```

Expected: banner mentioning `backend-dev-guidelines`.

- [ ] **Step 3: Phase-1 smoke test**

In a fresh Claude Code session on this branch, run:

```
/spec-task throwaway verification task for superpowers integration
```

Follow the prompts with minimal answers. Expected: a new folder `docs/tasks/task-NNN-throwaway-verification/` is created with `prd.md` inside. The handoff text at the end should mention `/design-task` (NOT `/dev-docs`).

- [ ] **Step 4: Phase-2 smoke test**

After `/clear`, run:

```
/design-task task-NNN-throwaway-verification
```

Expected: command loads `prd.md`, `CLAUDE.md`, `docs/superpowers-integration.md`, invokes `superpowers:brainstorming`, and ultimately writes `docs/tasks/task-NNN-throwaway-verification/design.md`. Handoff text mentions `/plan-task`.

- [ ] **Step 5: Phase-3 smoke test**

After `/clear`, run:

```
/plan-task task-NNN-throwaway-verification
```

Expected: produces `plan.md` AND `context.md` under the task folder. Handoff text mentions `/execute-task`. No `tasks.md` produced.

- [ ] **Step 6: Phase-4 smoke test (abbreviated)**

After `/clear`, run:

```
/execute-task task-NNN-throwaway-verification
```

Expected: asks about subagent-driven vs inline, recommends a worktree if on main. For the smoke test you can decline to actually execute; the goal is to verify the command itself wires up correctly.

- [ ] **Step 7: Reviewer-trio smoke test (optional, requires actual code changes)**

Skip if Steps 3-6 all validated. If you want to exercise `superpowers:requesting-code-review`, make a trivial atlas-ui TS change and one trivial Go change on the smoke-test branch, then invoke:

```
superpowers:requesting-code-review (via the Skill tool)
```

Expected: dispatches backend-guidelines-reviewer, frontend-guidelines-reviewer, and plan-adherence-reviewer (the last will note "no plan.md" and audit only the git diff). Aggregated report produced.

- [ ] **Step 8: Clean up the smoke test**

```bash
git checkout task-016-superpowers-integration
git branch -D task-016-smoketest
```

(Or preserve the smoke test branch for audit trail if preferred.)

- [ ] **Step 9: Final acceptance check — run all PRD §10 criteria**

```bash
# 1. Three new phase commands exist
ls .claude/commands/design-task.md .claude/commands/plan-task.md .claude/commands/execute-task.md

# 2. Five agents exist with frontmatter
for a in plan-adherence-reviewer backend-guidelines-reviewer frontend-guidelines-reviewer todo-scanner service-documentation; do
  head -5 ".claude/agents/$a.md" | grep -q "^name: $a" && echo "$a OK" || echo "$a FAIL"
done

# 3. Old documentation.md is gone
[ ! -f .claude/agents/documentation.md ] && echo "documentation.md gone OK"

# 4. Four thin-wrapper commands
for c in audit-plan backend-audit review-todos service-doc; do
  lines=$(wc -l < ".claude/commands/$c.md")
  [ "$lines" -lt 20 ] && echo "$c thin ($lines lines)" || echo "$c still big ($lines lines)"
done

# 5. dev-docs commands gone
[ ! -f .claude/commands/dev-docs.md ] && [ ! -f .claude/commands/dev-docs-update.md ] && echo "dev-docs commands gone OK"

# 6. skill-rules.json valid + has frontend entry
python3 -m json.tool < .claude/skills/skill-rules.json > /dev/null && echo "JSON valid"
grep -q 'frontend-dev-guidelines' .claude/skills/skill-rules.json && echo "frontend entry present"

# 7. CLAUDE.md has new section
grep -q '^## Development Workflow' CLAUDE.md && echo "CLAUDE.md section present"

# 8. docs/superpowers-integration.md exists
[ -f docs/superpowers-integration.md ] && echo "integration doc exists"

# 9. dev/active migration complete
[ -z "$(git ls-files dev/active/)" ] && echo "dev/active empty"
ls docs/tasks/ | grep -c '^legacy-' | xargs -I{} echo "legacy folders: {}"

# 10. dev/audits gone
[ -z "$(git ls-files dev/audits/)" ] && echo "dev/audits empty"

# 11. No dev/(active|audits) references remain
git grep -E 'dev/(active|audits)/' -- ':!docs/tasks/task-016-superpowers-integration/' || echo "refs all cleaned"
```

All ten checks must report OK. Any FAIL requires going back to the relevant earlier task and fixing it.

- [ ] **Step 10: Summary commit (optional)**

No code change in this task if all verifications pass. Optionally add a final marker:

```bash
git log --oneline main..task-016-superpowers-integration | wc -l
```

Record the commit count. Expected: ~75 (52 migration + 18 implementation + 4 setup + 1 audits-delete).

---

## Self-Review Notes

**Spec coverage (PRD §10 Acceptance Criteria → task mapping):**

| PRD criterion | Covered by task |
|---|---|
| design-task, plan-task, execute-task commands exist | Tasks 9, 10, 11 |
| Five agents exist with frontmatter | Tasks 1, 2, 3, 4, 5 |
| documentation.md absent | Task 5 |
| Four thin wrappers | Tasks 6, 7, 8 |
| dev-docs commands gone | Task 16 |
| skill-rules frontend entry + JSON valid + hook banner | Task 13 |
| CLAUDE.md new section preserving existing | Task 14 |
| superpowers-integration.md | Task 15 |
| dev/active fully migrated | Task 17 |
| dev/audits gone | Task 18 |
| No dev/(active|audits) refs remain | Task 19 + 20 |
| Four-phase flow works end-to-end | Task 21 |
| Reviewer trio dispatches correctly | Task 21 step 7 |

**Placeholder scan:** No `TBD` / `TODO` / "implement later" / "appropriate X" placeholders. Every step has actual content.

**Type consistency:** Agent names (`plan-adherence-reviewer`, `backend-guidelines-reviewer`, `frontend-guidelines-reviewer`, `todo-scanner`, `service-documentation`) are consistent across all tasks. Command names (`audit-plan`, `backend-audit`, `review-todos`, `service-doc`, `design-task`, `plan-task`, `execute-task`, `spec-task`) are consistent. File-path depth references (`services/<svc>/atlas.com/<module>/`) are consistent.

**Dependencies between tasks:**
- Tasks 1-5 (agents) must precede Tasks 6-8 (thin wrappers that dispatch them).
- Task 12 (/spec-task handoff) depends on Task 9 (design-task existing) for the handoff to be meaningful.
- Task 13 (skill-rules) is independent; can run anytime.
- Task 14 (CLAUDE.md) and Task 15 (superpowers-integration.md) reference the new commands/agents; safer to land after those exist.
- Tasks 17 (dev/active), 18 (dev/audits), 19 (repo reference sweep), 20 (auto-memory sweep) must run in that order — the sweeps depend on the migrations having landed.
- Task 21 (verification) is last.

The ordering in this plan satisfies all dependencies.

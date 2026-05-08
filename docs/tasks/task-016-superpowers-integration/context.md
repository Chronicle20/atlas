# task-016 Superpowers Integration — Execution Context

> Quick-reference for executing subagents. Full requirements in `prd.md`; full architecture in `design.md`. This file summarizes only what an executor needs at their fingertips.

## Goal (one sentence)

Replace atlas's dual-convention workflow (`dev/active/` + `docs/tasks/` + inline commands) with superpowers's four-phase flow (`/spec-task` → `/design-task` → `/plan-task` → `/execute-task`) backed by a modular reviewer-agent trio, consolidating all feature artifacts under `docs/tasks/task-NNN-slug/`.

## Reference implementation

All agent bodies, command bodies, CLAUDE.md wording, and `superpowers-integration.md` come from home-hub's `task-044-superpowers-integration`. Reference paths:

```
~/source/pers/home-hub/.claude/commands/{design-task,plan-task,execute-task,audit-plan,backend-audit,review-todos,service-doc,spec-task}.md
~/source/pers/home-hub/.claude/agents/{plan-adherence-reviewer,backend-guidelines-reviewer,frontend-guidelines-reviewer,todo-scanner,service-documentation}.md
~/source/pers/home-hub/.claude/skills/skill-rules.json
~/source/pers/home-hub/CLAUDE.md (workflow section)
~/source/pers/home-hub/docs/superpowers-integration.md
```

Atlas diverges from home-hub in predictable ways — see "Adaptation rules" below.

## Files touched by this task

### Created

| Path | Purpose |
|------|---------|
| `.claude/commands/design-task.md` | Phase 2 command — invokes `superpowers:brainstorming` |
| `.claude/commands/plan-task.md` | Phase 3 command — invokes `superpowers:writing-plans` |
| `.claude/commands/execute-task.md` | Phase 4 command — invokes subagent-driven-development |
| `.claude/agents/plan-adherence-reviewer.md` | Verifies plan.md tasks are implemented |
| `.claude/agents/backend-guidelines-reviewer.md` | Adversarial Go DOM/SUB/SEC audit |
| `.claude/agents/frontend-guidelines-reviewer.md` | Adversarial atlas-ui FE-* audit |
| `.claude/agents/todo-scanner.md` | Whole-codebase TODO scan |
| `.claude/agents/service-documentation.md` | Rename of `agents/documentation.md` with full frontmatter |
| `docs/superpowers-integration.md` | When-to-use-what quick reference |

### Modified

| Path | Change |
|------|--------|
| `.claude/commands/spec-task.md` | Step 5 handoff text: `/dev-docs` → `/design-task` (see design §3.4) |
| `.claude/commands/audit-plan.md` | Shrink to thin wrapper dispatching `plan-adherence-reviewer` |
| `.claude/commands/backend-audit.md` | Shrink to thin wrapper dispatching `backend-guidelines-reviewer` |
| `.claude/commands/review-todos.md` | Shrink to thin wrapper dispatching `todo-scanner` |
| `.claude/commands/service-doc.md` | Shrink to thin wrapper dispatching `service-documentation` |
| `.claude/skills/skill-rules.json` | Add `frontend-dev-guidelines` entry (see design §3.13) |
| `CLAUDE.md` | Add "Development Workflow" section (additive, see design §3.14) |

### Deleted

| Path | Reason |
|------|--------|
| `.claude/agents/documentation.md` | Replaced by `agents/service-documentation.md` with full frontmatter |
| `.claude/commands/dev-docs.md` | Split into `/design-task` + `/plan-task` |
| `.claude/commands/dev-docs-update.md` | No replacement; future doc-update work goes through the four-phase flow |

### Moved

52 folders under `dev/active/` → `docs/tasks/legacy-<slug>/` via per-folder `git mv` (see design §4.1). 25 folders under `dev/audits/` → deleted outright (see design §4.2).

## Adaptation rules — atlas vs home-hub

| Area | home-hub | atlas |
|------|----------|-------|
| Service path depth | `services/<service>/` | `services/<service>/atlas.com/<module>/` |
| Frontend dir | `frontend/` at repo root | `services/atlas-ui/` |
| Frontend test runner | `jest --watchAll=false` | `vitest run` (via `npm test`) |
| Frontend build | `npm run build` | `npm run build` (runs `tsc -b && vite build`) |
| Project name in copy | "Home Hub" | "Atlas" |
| Domain converters | `recipe-to-cooklang` | `convert-{map,npc,portal,quest,reactor}` |
| Backend audit scope | `services/<service>/internal/` | `services/<service>/atlas.com/<module>/internal/` |

Apply these substitutions mechanically when translating home-hub content into atlas content. Nothing else differs.

## Key decisions (from design §6)

1. **Command → skill handoff**: preamble prompt pattern (main agent invokes Skill tool with override context as natural-language; no skill modification).
2. **frontend-guidelines-reviewer scope**: `services/atlas-ui/` only — sole frontend in the repo.
3. **`legacy.md` marker files**: none — folder-name prefix `legacy-` is self-documenting.
4. **CLAUDE.md new section**: additive, placed between "Code Patterns" and "Documentation"; preserves existing sections verbatim.
5. **Migration commit granularity**: one commit per `dev/active/<slug>` folder (~52 commits); one commit for `dev/audits/` deletion; one commit for the reference sweep.

## Verification commands (atlas-specific)

### Go service build/test
```bash
cd services/<service>/atlas.com/<module>
go build ./...
go test ./... -count=1
```

### atlas-ui build/test
```bash
cd services/atlas-ui
npm run build     # tsc -b && vite build
npm test          # vitest run
```

### YAML/JSON validity
```bash
python3 -m json.tool < .claude/skills/skill-rules.json
# YAML frontmatter parseable (inspect via Read tool)
```

### Hook smoke test
```bash
echo '{"prompt":"I need to add a new React component with a Zod form"}' | python3 .claude/hooks/skill-activation-prompt.py
# Expected: output contains "🎯 SKILL ACTIVATION CHECK" and "frontend-dev-guidelines"
```

### Migration verification
```bash
git ls-files dev/active/    # must be empty after migration
git ls-files dev/audits/    # must be empty after deletion
git log --follow -- docs/tasks/legacy-redis-registry-migration/context.md
# Must show pre-rename history in dev/active/redis-registry-migration/
git grep -E 'dev/(active|audits)/' -- ':!docs/tasks/task-016-superpowers-integration/'
# Must return empty
```

## Out-of-scope reminders

Per PRD §2 and design §8:
- Do NOT modify any superpowers skill under `~/.claude/plugins/cache/claude-plugins-official/superpowers/`.
- Do NOT modify `.claude/hooks/skill-activation-prompt.{sh,py}`.
- Do NOT modify `.claude/skills/{backend,frontend}-dev-guidelines/resources/`.
- Do NOT modify `.claude/settings.json`.
- Do NOT modify any Go service under `services/<service>/atlas.com/<module>/`.
- Do NOT modify `services/atlas-ui/` code (only reference its path from agents/skill-rules).
- Do NOT modify `libs/*`.
- Do NOT modify the five `convert-*` commands.
- Do NOT modify content of existing `docs/tasks/task-001` through `task-015` folders (including their `tasks.md` files).

## Memory files to update in reference sweep

Files under `~/.claude/projects/-<workspace-pers-atlas>/memory/` that currently reference `dev/active/...`:
- `MEMORY.md` (the index) — has 5 mentions
- Any sibling file referenced by the index that describes one of the five migrated slugs (redis-migration.md, etc.)

Rewrite rule: `dev/active/<slug>/` → `docs/tasks/legacy-<slug>/`; stripped-prefix inner filenames (`<slug>-context.md` → `context.md`).

## Execution branch

Already created: `task-016-superpowers-integration`. The PRD and design are already committed on this branch. All implementation commits land here.

# CLAUDE.md rule inventory (FR-1)

One row per atomic rule in `CLAUDE.md` as of branch point (220 lines / 19,543
bytes). This is the working ledger for the whole refactor — produced before
any file is edited (design §5.1) — not a report written afterward.

Columns: `#` is a stable id assigned in source order; later tasks cite it.
Class is A global invariant · B global operating default · C task router · D
procedure/reference · E rationale/history (PRD FR-1). Disposition is exactly
one of `keep-verbatim` · `keep-compressed` · `relocate` · `drop-captured`.
`drop-captured` is only used where the destination text was opened and read,
and already carries the rule.

| # | Source | Rule | Class | Disposition | Destination |
|---|---|---|---|---|---|
| R-001 | Project Overview | Go microservices game server project, 14+ services; Go is the primary language, TypeScript only in atlas-ui | B | keep-compressed | `CLAUDE.md` §Atlas (orientation line) |
| R-002 | Workflow Rules | When asked to understand/plan, do not start implementing; wait for explicit approval before editing — planning and implementation are separate phases | A | keep-verbatim | `CLAUDE.md` §Development workflow |
| R-003 | Build & Verification | Before claiming "done"/"ready for PR"/invoking `finishing-a-development-branch`, run flagless `tools/verify.sh` and it must exit 0 | A | keep-verbatim | `CLAUDE.md` §Done means verified |
| R-004 | Build & Verification | Only the flagless invocation counts as verified; `--quick`/`--no-docker` also exit 0 but skip the bake and `-race`, so exiting 0 under a flag is not a pass | A | keep-verbatim | `CLAUDE.md` §Done means verified |
| R-005 | Build & Verification | The gate mirrors `.github/workflows/pr-validation.yml`: per-module build/vet/test `-race`, `docker buildx bake` per touched service, every repo guard, `tools/lint.sh --check`; change-gated the same way CI is | D | drop-captured | `docs/verification.md` §The Go layer / §The docker layer / §Guards / §Lint & format |
| R-006 | Build & Verification | Never claim verified from a subset; the bake step is not optional — `go build` against `go.work` cannot catch a missing `COPY libs/...` in the shared Dockerfile | D | drop-captured | `docs/verification.md` §The docker layer |
| R-007 | Build & Verification | Per-task gate runs should pass `--base <last-gated-commit>` so the change set is the increment, not the accumulated branch — one `libs/` commit otherwise fans every later run out to all 86 modules (~11 min instead of ~1) | E | drop-captured | `docs/verification.md` §The iteration gate |
| R-008 | Build & Verification | Launch the gate in the background and keep working; never idle waiting on it | B | relocate | `docs/verification.md` §The iteration gate |
| R-009 | Build & Verification | Router: rationale/per-guard invariants/CI drift → `docs/verification.md`; adding a service → `docs/adding-a-new-service.md` | C | keep-compressed | `CLAUDE.md` §Where the procedures live |
| R-010 | Code Patterns | When refactoring shared types/common libraries, prefer straightforward moves over re-exporting type aliases; don't call another layer's internals across service boundaries | B | keep-compressed | `CLAUDE.md` §Repository conventions |
| R-011 | Code Patterns | Before defining a new domain type/alias/numeric constant in a service, check `libs/atlas-constants/` for an existing equivalent (item ids, inventory/compartment types, weapon types, world/channel/map/character ids, jobs, skills, monster ids); enforced as DOM-21 | B | keep-compressed | `CLAUDE.md` §Repository conventions |
| R-012 | Development Workflow | The canonical flow for any non-trivial change is four phases, each a separate slash command invoked from a fresh (`/clear`'d) session, so the next phase consumes only the prior phase's documented artifacts | A | keep-compressed | `CLAUDE.md` §Development workflow |
| R-012a | Development Workflow | `/spec-task` creates a dedicated worktree at `.worktrees/task-NNN-slug/` on a `task-NNN-slug` branch; all subsequent phases run inside that worktree so docs, code, and the eventual PR are one unit | A | keep-verbatim | `CLAUDE.md` §Development workflow |
| R-013 | Development Workflow | Phase 1 — `/spec-task <idea>` runs from the main repo; interactive PRD interview that creates the worktree + branch and commits the PRD; output `<worktree>/docs/tasks/task-NNN-slug/prd.md` | A | keep-verbatim | `CLAUDE.md` §Development workflow |
| R-014 | Development Workflow | Phase 2 — `cd .worktrees/task-NNN-slug`, `/clear`, then `/design-task <task-id>`, which invokes `superpowers:brainstorming`; output `design.md` committed on the task branch | A | keep-verbatim | `CLAUDE.md` §Development workflow |
| R-015 | Development Workflow | Phase 3 — `/clear`, then `/plan-task <task-id>`, which invokes `superpowers:writing-plans`; output `plan.md` + `context.md` committed | A | keep-verbatim | `CLAUDE.md` §Development workflow |
| R-016 | Development Workflow | Phase 4 — `/clear`, then `/execute-task <task-id>`, which invokes `superpowers:subagent-driven-development`; reuses the existing worktree, never creates a new one | A | keep-verbatim | `CLAUDE.md` §Development workflow |
| R-017 | Development Workflow | Phase commands accept fuzzy task identifiers (`task-054-slug`, `task-054`, `054`, `54`) resolved via `tools/task-resolve.sh <identifier>`, printing `<task-id>\t<task-dir>\t<worktree>` | D | relocate | `docs/superpowers-integration.md` §Task lifecycle, task resolution |
| R-018 | Development Workflow | Never glob `.worktrees/*/docs/tasks/task-*` to find a task — every worktree carries a full copy of `docs/tasks/`, so that pattern returns tasks × worktrees mostly-duplicate paths | D | relocate | `docs/superpowers-integration.md` §Task lifecycle, task resolution |
| R-019 | Development Workflow | Skip `/spec-task` only for trivial fixes that don't warrant a PRD; document those directly via a brainstorming session | B | keep-compressed | `CLAUDE.md` §Development workflow |
| R-020 | Artifact Location Override | `superpowers:brainstorming`/`superpowers:writing-plans` default to `docs/superpowers/specs\|plans/`; in this project both go under `docs/tasks/task-NNN-slug/` instead — pass the task folder explicitly when invoking directly | D | relocate | `docs/superpowers-integration.md` §Artifact location override |
| R-021 | Code Review Pattern | Code review uses three modular reviewer agents dispatched in parallel: `plan-adherence-reviewer`, `backend-guidelines-reviewer`, `frontend-guidelines-reviewer` | D | drop-captured | `docs/superpowers-integration.md` §Code Review (lines 33-39 already list the same three agents) |
| R-022 | Code Review Pattern | Invoke via `superpowers:requesting-code-review`, or invoke an individual agent directly for ad-hoc checks | D | drop-captured | `docs/superpowers-integration.md` §Code Review (already states both invocation paths almost verbatim) |
| R-023 | Code Review Pattern | Each agent writes its findings to `docs/tasks/task-NNN-slug/audit.md` | D | relocate | `docs/superpowers-integration.md` §Code Review (the specific `audit.md` output path is not stated in that section) |
| R-024 | Code Review Pattern | See `docs/superpowers-integration.md` for a complete when-to-use-what reference | C | keep-compressed | `CLAUDE.md` §Where the procedures live |
| R-025 | Documentation | When updating TODO.md or other tracking docs, always use Glob/Grep to find the file first rather than assuming a path; updates follow the /dev-docs format. **Note (fix round 1, task 7):** the `/dev-docs` clause is deliberately dropped, not relocated — `/dev-docs` no longer exists (task-016 split it into `/design-task` + `/plan-task` and deleted `.claude/commands/dev-docs.md`); only the Glob/Grep-first clause is retained | B | keep-compressed | `CLAUDE.md` §Repository conventions |
| R-026 | Design/Plan Output Style | When producing design.md/plan.md, write the full document directly to file; do not walk through sections interactively or ask for per-section approval | B | keep-compressed | `CLAUDE.md` §Repository conventions |
| R-027 | Worktrees & Subagents | Before planning/designing/executing a task, verify cwd is the correct worktree; cd into it yourself rather than asking the user | A | keep-verbatim | `CLAUDE.md` §Where you work — branch & worktree |
| R-028 | Worktrees & Subagents | When searching for task PRDs/plans/designs, search across all worktrees (`git worktree list`) before concluding a file is missing | A | keep-verbatim | `CLAUDE.md` §Where you work — branch & worktree |
| R-029 | Worktrees & Subagents | Never edit files in the main repo when a task worktree exists for that work | A | keep-verbatim | `CLAUDE.md` §Where you work — branch & worktree |
| R-030 | Worktrees & Subagents | When dispatching subagents, ensure they operate inside the correct worktree — never write artifacts or edits into the main repo; verify the tree is clean after subagent runs | A | keep-verbatim | `CLAUDE.md` §Where you work — branch & worktree |
| R-031 | Code Review Before PR | Always run the code-review step before opening a PR; do not skip even when the task plan looks complete | A | keep-verbatim | `CLAUDE.md` §Done means verified |
| R-032 | Code Review Before PR | A green `tools/verify.sh` does not mean the branch is correct — the gate cannot see a service seam; when a change crosses a service boundary, trace the event into its consumers by hand and check that a test asserts the NEW contract | A | keep-compressed | `CLAUDE.md` §Done means verified |
| R-033 | Code Review Before PR | Example seam failures: a producer that empties a compartment the consumer still reads, a new saga action with no step handler in the orchestrator, a class of missing emits that existing tests actively pin as the old behavior | D | relocate | `docs/superpowers-integration.md` §Code Review |
| R-034 | Grounding, Verification & Finishing | Never invent values, names, opcodes, output, or behavior; unverified is "unknown/unverified," not a plausible guess — "I think it's X" is a lead to check, not a finding | A | keep-verbatim | `CLAUDE.md` §Evidence & grounding |
| R-035 | Grounding, Verification & Finishing | Quote the actual tool output before drawing a conclusion; do not paraphrase numbers from memory | A | keep-verbatim | `CLAUDE.md` §Evidence & grounding |
| R-036 | Grounding, Verification & Finishing | Verify, don't recall — for game data, packet encoding, protocol details, and service ownership, read the local WZ data or repo source; general MapleStory knowledge is not a source | A | keep-verbatim | `CLAUDE.md` §Evidence & grounding |
| R-037 | Grounding, Verification & Finishing | Confirm the exact server/tenant version being tested before investigating any bug; the wrong version sends the investigation down the wrong path | A | keep-verbatim | `CLAUDE.md` §Evidence & grounding |
| R-038 | Grounding, Verification & Finishing | Sweep, don't spot-check — a spot-check presented as a full sweep is a false "verified"; state findings as hypotheses until confirmed against real evidence | A | keep-verbatim | `CLAUDE.md` §Evidence & grounding |
| R-039 | Grounding, Verification & Finishing | Do not declare a "documented gap," "follow-up task," or "out of scope" when the blocker is a prerequisite you can produce yourself: an unnamed IDB function → name it; an unrouted template → wire it; a missing export → generate it | A | keep-verbatim | `CLAUDE.md` §Evidence & grounding |
| R-040 | Grounding, Verification & Finishing | Do not split work into a new task to avoid finishing this one — keep triage and fix on the same branch, and produce the clean PR branch by rebase at PR time | A | keep-verbatim | `CLAUDE.md` §Evidence & grounding |
| R-041 | Grounding, Verification & Finishing | No `// TODO`, stubbed handlers, or 501s in landed commits | A | keep-verbatim | `CLAUDE.md` §Evidence & grounding |
| R-042 | Grounding, Verification & Finishing | A genuine external blocker, an ambiguous design decision, or an unresolved packet-audit fname is different — surface it and ask; the bar is "can I produce this myself right now?" | A | keep-verbatim | `CLAUDE.md` §Evidence & grounding |
| R-043 | Test Helper Pattern | Use the project's Builder pattern for test setup; do not create `*_testhelpers.go` files with test-only constructors | B | keep-compressed | `CLAUDE.md` §Repository conventions |
| R-044 | File Writing / Conventions | When writing files, always use repo-relative paths or placeholders; never write literal home/absolute paths into committed files | A | keep-compressed | `CLAUDE.md` §Repository conventions |
| R-045 | Packet work | Packet-audit work has ONE canonical playbook per task type and an executable entry point; start at `docs/packets/PROCESS.md` — source of truth for version set, baseline status, CI gates, task-type table | C | keep-compressed | `CLAUDE.md` §Where the procedures live |
| R-046 | Packet work | Entry points: new feature codec → `/implement-packet` + `packet-implementer`; new client-version column → `/bringup-version`; dispatcher family → `family-auditor` then `dispatcher-family-implementer`; leaf step (all task types) → `/verify-packet` + `packet-verifier`; do not restate a playbook's procedure in prose elsewhere | D | drop-captured | `docs/packets/PROCESS.md` §Task type → entry point → canonical playbook |
| R-047 | Reverse Engineering / IDA | For IDA Pro lookups, use the `func_query` tool with `name_regex` (the documented method); do not improvise alternate lookup approaches | D | relocate | `docs/reverse-engineering.md` §Function lookup |
| R-047a | Reverse Engineering / IDA | See the IDA-MCP notes in project memory for the current API | D | relocate | `docs/reverse-engineering.md` §Function lookup |
| R-048 | Reverse Engineering / IDA | Confirm the IDA instance/version under investigation matches the version you're targeting before reading | A | drop-captured | `CLAUDE.md` §Evidence & grounding (already states the general confirm-version rule, R-037) |
| R-049 | Reverse Engineering / IDA | `select_instance(port)` and port-based selection are dead (since task-138); resolve the session from `idb_list` by binary name and pass it as the `database` parameter to subsequent calls | D | relocate | `docs/reverse-engineering.md` §Session resolution |
| R-050 | Task Workflow | Before planning or designing a task, verify the task is not already planned/implemented and its number does not collide with an in-flight task | B | keep-compressed | `CLAUDE.md` §Development workflow |
| R-051 | Task Workflow | Tool usage detail: `tools/task-numbers.sh next` to pick the number, `tools/task-resolve.sh --list` to see every existing task (one row per task, deduplicated across worktrees) | D | relocate | `docs/superpowers-integration.md` §Task lifecycle, task resolution |
| R-052 | Debugging / Kubernetes | For wedged deploys or runtime failures, read the relevant pod logs early rather than starting at packet-level fixes or bare pod listings; the logs usually name the real root cause directly | B | keep-compressed | `CLAUDE.md` §Repository conventions |
| R-053 | Debugging / Kubernetes | Read logs via `mcp__kubernetes__pods_log`, e.g. `atlas-character-factory`, `atlas-world` | D | relocate | `docs/observability.md` §Diagnosing a runtime failure |
| R-054 | Git Operations | Never commit or push directly to `main`; branch protection blocks the push, so a commit made on local `main` is stranded; check the branch before every commit | A | keep-verbatim | `CLAUDE.md` §Where you work — branch & worktree |
| R-055 | Git Operations | Setup work that must precede a feature branch still goes on the feature branch — create it first; it branches from the same HEAD | A | keep-verbatim | `CLAUDE.md` §Where you work — branch & worktree |
| R-056 | Git Operations | Recovery from a stray main commit: preserve the content on a branch (cherry-pick if needed), then `git fetch origin main && git reset --hard origin/main` | D | relocate | `docs/git-workflow.md` §Branch safety |
| R-057 | Git Operations | After completing a rebase/merge/history-rewrite, always push (force-push when history was rewritten) so the PR reflects the resolved state; do not stop at local-only completion | A | keep-verbatim | `CLAUDE.md` §Where you work — branch & worktree |
| R-058 | Git Operations | A plain push to a task branch DOES trigger the PR workflows and the ephemeral rollout; do not merge `origin/main` as a routine build-triggering ritual; exception: when the branch conflicts with main, merge `origin/main`, resolve, push the merge commit | B | keep-compressed | `CLAUDE.md` §Where you work — branch & worktree |
| R-059 | Git Operations | The `atlas-pr-<N>` ephemeral rollout behavior and the full conflict-vs-build-trigger mechanics | D | relocate | `docs/git-workflow.md` §Build triggering and the conflict exception |
| R-060 | Git Operations | Run `gh` with the token env explicitly cleared (`env -u GH_TOKEN -u GITHUB_TOKEN gh …`) so it uses stored `hosts.yml` auth; do NOT source `~/.config/atlas/gh.env` — its `GH_TOKEN` is expired and causes 401s; never echo the token | D | relocate | `docs/git-workflow.md` §`gh` authentication |
| R-061 | Interaction | Text emitted before an `AskUserQuestion` in the same turn does not render reliably; send substantive content as its own text-only message, then ask the question in a following turn | B | keep-compressed | `CLAUDE.md` §Repository conventions |
| R-062 | Interaction | Do not proactively pitch paid features (`/schedule`, remote/cloud agents, `/ultrareview`, scheduled routines); if genuinely the right tool, lead with what is known/unknown about billing; end-of-turn suggestions should be free/local | B | keep-compressed | `CLAUDE.md` §Repository conventions |
| R-063 | Model & Cost Preferences | Pass an explicit `model` on every Agent/Task dispatch; unspecified inherits Opus, and an Opus subagent turn costs ~7x a Sonnet one | A | keep-compressed | `CLAUDE.md` §Dispatching agents |
| R-064 | Model & Cost Preferences | The ~7x cost comparison detail | E | relocate | `docs/agent-dispatch.md` §Model selection |
| R-065 | Model & Cost Preferences | The pin follows the job, not the `subagent_type` — review/verify/audit always runs sonnet (including ad-hoc general-purpose agents carrying a review prompt); scans/inventories run haiku; implementers run sonnet unless tagged `model: opus` | B | keep-compressed | `CLAUDE.md` §Dispatching agents |
| R-066 | Model & Cost Preferences | The full job→model table (six rows), referenced from `.claude/commands/execute-task.md` Step 4a | D | relocate | `docs/agent-dispatch.md` §Model selection |
| R-067 | Model & Cost Preferences | Never use Fable for background/review workflows | A | keep-verbatim | `CLAUDE.md` §Dispatching agents |
| R-068 | Model & Cost Preferences | Long agents are the cost: context grows with turn count and every turn re-reads all of it, so one 600-turn agent costs far more than the same work split across fresh contexts | E | relocate | `docs/agent-dispatch.md` §The implementer budget |
| R-069 | Model & Cost Preferences | The implementer budget is 120 tool calls; at the cap the implementer commits and reports `PARTIAL`, and the controller dispatches a continuation | D | relocate | `docs/agent-dispatch.md` §The implementer budget |
| R-070 | Model & Cost Preferences | The budget is warned at 100 calls | D | relocate | `docs/agent-dispatch.md` §The implementer budget |
| R-071 | Model & Cost Preferences | Counted by `.claude/hooks/turn-budget.sh` and contracted in `.claude/agents/atlas-implementer.md`, `.claude/agents/packet-implementer.md`, and `.claude/agents/dispatcher-family-implementer.md` | D | relocate | `docs/agent-dispatch.md` §The implementer budget |
| R-072 | Model & Cost Preferences | The cap is binding: `.claude/hooks/turn-budget-guard.sh` (PreToolUse) denies subagent calls past CAP+5, exempting the commit-and-report path; controllers are never blocked | D | relocate | `docs/agent-dispatch.md` §The implementer budget |
| R-073 | Model & Cost Preferences | Change the number in the counting hook only | D | relocate | `docs/agent-dispatch.md` §The implementer budget |
| R-074 | Model & Cost Preferences | Implementers do not run repo-wide verification — `tools/verify.sh`, `tools/lint.sh`, `-race`, and docker bake belong to `atlas-verifier` in its own clean context; implementers run module-local `go build ./... && go test ./...` and nothing more | A | keep-compressed | `CLAUDE.md` §Dispatching agents |
| R-075 | Model & Cost Preferences | A `--quick` run inside a 400k-token implementer costs a large multiple of the same run in a 20k one | E | relocate | `docs/agent-dispatch.md` §Verification split |
| R-076 | Model & Cost Preferences | Fan out with fresh-context agents, not `subagent_type: "fork"` — a named agent type plus an explicit brief; fork only to continue an interactive debugging thread, and say why inline | A | keep-verbatim | `CLAUDE.md` §Dispatching agents |
| R-077 | Model & Cost Preferences | `fork-dispatch-guard.sh` denies an unjustified fork and states the cost | D | relocate | `docs/agent-dispatch.md` §Fork vs fresh context |
| R-078 | Model & Cost Preferences | The same arithmetic binds the controller, which is the one context that lives for a whole plan — every wake-up re-reads it; during `/execute-task`, hand off to a fresh session past ~250k tokens with tasks remaining | E | relocate | `docs/agent-dispatch.md` §Context handoff |
| R-079 | Model & Cost Preferences | The SDD ledger (`.superpowers/sdd/<plan>/progress.md`) is the resume point, so the cost is one plan re-read | D | relocate | `docs/agent-dispatch.md` §Context handoff |
| R-080 | Model & Cost Preferences | Procedure: `.claude/commands/execute-task.md` Step 4e | D | relocate | `docs/agent-dispatch.md` §Context handoff |
| R-081 | Context Handoff | The unit of work is a briefable task, not a conversation; context cost scales with turn count × context size, so 50 turns at 190k costs roughly ten times the same 50 turns at 19k | E | relocate | `docs/agent-dispatch.md` §Context handoff |
| R-082 | Context Handoff | At every durable boundary, ask whether the next unit of work depends materially on this conversation's history, or only on repository state | A | keep-verbatim | `CLAUDE.md` §Handing off context |
| R-083 | Context Handoff | If it can be resumed from repo state + task reports + a short written diagnosis, hand off; do not wait for a context threshold — the signal is dependency, not size | A | keep-verbatim | `CLAUDE.md` §Handing off context |
| R-084 | Context Handoff | Handing off means delegating, not clearing — `/clear` is a user action; dispatch the next unit to a fresh agent with a brief; only when genuinely controller-shaped, write the diagnosis and let the user `/clear` | A | keep-verbatim | `CLAUDE.md` §Handing off context |
| R-085 | Context Handoff | The diagnosis must be written before the handoff, not carried in your head — one paragraph into the task folder; a handoff whose reasoning survives only in conversation is not a handoff | A | keep-verbatim | `CLAUDE.md` §Handing off context |
| R-086 | Context Handoff | There is a floor as well as a backstop — context size thresholds are backstops; dependency is the primary signal | A | keep-compressed | `CLAUDE.md` §Handing off context |
| R-087 | Context Handoff | Specific thresholds: below ~60k a fresh agent re-discovers files you already hold; under ~40 tool calls prefer continuing (`commit-boundary.sh` raises the question past the floor); backstop ~250k for a controller | D | relocate | `docs/agent-dispatch.md` §Context handoff |
| R-088 | Context Handoff | The pattern already exists — `/execute-task` Steps 4d/4e are this handoff in concrete forms with durable artifacts; apply the same shape elsewhere; generate briefs with `tools/task-brief.sh`, never by hand out of `plan.md` | D | relocate | `docs/agent-dispatch.md` §Context handoff |
| R-089 | Shell & Editing Conventions | Prefer portable POSIX shell; avoid zsh/direnv-specific constructs and batch patch loops; for a multi-file edit, prefer per-file Edit/Write over a shell patch loop | B | keep-compressed | `CLAUDE.md` §Repository conventions |
| R-090 | Shell & Editing Conventions | Preserve line endings when editing; do not normalize CRLF→LF as a side effect — it inflates diffs with spurious changes | A | keep-compressed | `CLAUDE.md` §Repository conventions |
| R-091 | Shell & Editing Conventions | Never sweep the filesystem to locate a Go dependency's source; ask the toolchain — `go list -m -f '{{.Dir}}' <module>` | A | keep-compressed | `CLAUDE.md` §Repository conventions |
| R-092 | Shell & Editing Conventions | The same applies to `go doc <pkg>` for a symbol and `go list -m all` for the version set; the module-cache case-escaping tell; `find` is for paths you own, rooted at a directory you name, never at `/` | D | relocate | `docs/tooling-conventions.md` §Locating Go module source |
| R-093 | Shell & Editing Conventions | The `find /` timing measurement (~2 min on WSL2) and the task-227 anecdote (6 minutes across five sweeps hunting `atlas-rest`, which was `replace`d to `libs/atlas-rest` inside the worktree already in use) | E | relocate | `docs/tooling-conventions.md` §Locating Go module source |
| R-094 | Shell & Editing Conventions | Never spend inference turns waiting for a process; launch it once with a bound (`run_in_background`, or Monitor with an until-loop) and do something else or hand back; if it exceeds its bound, kill it and fall back, do not keep polling | A | keep-compressed | `CLAUDE.md` §Repository conventions |
| R-095 | Shell & Editing Conventions | The polling anti-pattern detail (repeated `sleep`/`ps aux \| grep`/`echo waiting`/`for … do sleep`), why it clusters expensively late in a session, and that per-tool hang fallbacks belong in that tool's agent doc | E | relocate | `docs/tooling-conventions.md` §Waiting on processes |

## Step 3 — cross-check against PRD FR-3 / FR-4

**FR-3 (retained) coverage** — every FR-3 bullet resolves to a `keep-verbatim`
or `keep-compressed` row above:

- Planning discipline → R-002.
- Evidence and grounding (5 bullets, now atomized to 9 rows since three of
  the five bullets each carried multiple obligations) → R-034–R-042.
- Branch and worktree safety (7 bullets) → R-027–R-030, R-054, R-055, R-057,
  R-058.
- Completion and verification claims (4 bullets) → R-003, R-004, R-031, R-032.
- Development lifecycle → R-012–R-016 (the four-phase overview plus one row
  per phase, per fix round 1 finding 1).
- Agent dispatch defaults (5 bullets) → R-063, R-065, R-067, R-074, R-076.
- Context handoff decision (4 bullets) → R-082–R-085, R-086.
- Task numbering → R-050.
- Debugging default → R-052.
- Short repository conventions (11 bullets) → R-010, R-011, R-025, R-026,
  R-043, R-044, R-061, R-062, R-089–R-091, R-094.

No FR-3 item is missing a `keep-*` row.

**FR-4 (relocated) coverage** — every FR-4 destination list is matched by
`relocate`/`drop-captured` rows naming that destination:

- `docs/verification.md` → R-005–R-008 (largely already captured, per design
  §6 — confirmed by reading the file: the iteration gate, Go layer, docker
  layer, guards, and lint sections all pre-exist the relocated text).
- `docs/superpowers-integration.md` → R-017, R-018, R-020–R-023, R-033, R-051.
  R-021/R-022 are `drop-captured` (confirmed by reading lines 33-39: the
  three-reviewer-agent inventory and the invoke-via-skill-or-directly pattern
  are already stated there almost verbatim); R-023 (`audit.md` output path)
  is `relocate` — that specific detail is absent from the section.
- `docs/packets/PROCESS.md` → R-046, `drop-captured` (confirmed sufficient by
  reading the file — design §4.1).
- `docs/reverse-engineering.md` → R-047, R-047a, R-049.
- `docs/git-workflow.md` → R-056, R-059, R-060.
- `docs/agent-dispatch.md` → R-064, R-066, R-068–R-073, R-075, R-077–R-080,
  R-081, R-087, R-088.
- `docs/tooling-conventions.md` → R-092, R-093, R-095.
- Runtime-debugging owner (`docs/observability.md`, per design §4.2) → R-053.

No disagreement found between this classification and the PRD; no note rows
were required.

## Round-trip verification (Task 9)

### Step 1 — forward round-trip: nothing lost

Walked all 97 rows (`R-001`–`R-095`, plus `R-012a`, `R-047a`). For every
`keep-verbatim`/`keep-compressed` row, the rule was located in the new
`CLAUDE.md`. For every `relocate`/`drop-captured` row, the destination file
was opened and the rule text confirmed present (not trusted from the row).

**Result: 97/97 pass, no failures, no fixes required.**

Two coexistences were confirmed as authorized, not defects:

- `R-004` (flagless `tools/verify.sh` is the only pass) stays `keep-verbatim`
  in `CLAUDE.md` §Done means verified; `R-008` (launch in background, never
  idle) is the row that relocates to `docs/verification.md` §The iteration
  gate. `docs/verification.md` additionally documents `--all`, which
  `CLAUDE.md` does not — an intentional elaboration, not drift.
- `R-025`'s dropped `/dev-docs` clause: confirmed `.claude/commands/dev-docs.md`
  does not exist in this worktree; the row's fix-round-1 note stands.

Every `relocate`/`drop-captured` destination was opened and read directly:
`docs/verification.md`, `docs/superpowers-integration.md`,
`docs/git-workflow.md`, `docs/reverse-engineering.md`,
`docs/tooling-conventions.md`, `docs/agent-dispatch.md`,
`docs/observability.md`, `docs/packets/PROCESS.md`,
`docs/adding-a-new-service.md`. All named destination sections carry the
rule text described by their row.

### Step 2 — reverse round-trip: nothing invented

Walked the finished `CLAUDE.md` sentence by sentence and mapped every
sentence to an `R-NNN` row.

| CLAUDE.md section | Sentence/bullet | Maps to |
|---|---|---|
| `# Atlas` intro | Go microservices monorepo, 14+ services, TS only in atlas-ui | R-001 |
| Never do this | Never commit/push to `main` | R-054 |
| Never do this | Never edit main repo when a task worktree exists | R-029 |
| Never do this | Never invent a value/name/opcode/output/behavior | R-034 |
| Never do this | Never claim verified from a flagged/partial run | R-004 |
| Never do this | Never open a PR without code review | R-031 |
| Never do this | Never dispatch an agent without explicit `model` | R-063 |
| Never do this | Never land a placeholder/stub/501 | R-041 |
| Evidence & grounding | Never invent… quote actual tool output | R-034, R-035 |
| Evidence & grounding | Repo/WZ/IDA/live output outrank memory | R-036 |
| Evidence & grounding | Confirm exact version before investigating | R-037 |
| Evidence & grounding | Sweep, don't spot-check | R-038 |
| Evidence & grounding | Finish producible work / don't split to avoid finishing | R-039, R-040 |
| Evidence & grounding | Genuine blocker → surface and ask | R-042 |
| Where you work | Check branch before every commit | R-054 |
| Where you work | Setup work goes on the feature branch | R-055 |
| Where you work | Verify cwd/worktree before planning/designing/executing | R-027 |
| Where you work | Search all worktrees before concluding missing | R-028 |
| Where you work | Subagents stay inside the correct worktree | R-030 |
| Where you work | Push after rebase/merge/history-rewrite | R-057 |
| Where you work | Plain push triggers PR workflow; conflict exception | R-058 |
| Done means verified | Flagless `verify.sh` must exit 0 before "done" | R-003 |
| Done means verified | Only flagless counts; `--quick`/`--no-docker` skip bake/-race | R-004 |
| Done means verified | Always run code review before PR | R-031 |
| Done means verified | Green `verify.sh` ≠ correct; trace seams by hand | R-032 |
| Development workflow | Don't implement during planning; wait for approval | R-002 |
| Development workflow | Four-phase flow, worktree/branch creation | R-012, R-012a |
| Development workflow | Phase 1–4 list | R-013, R-014, R-015, R-016 |
| Development workflow | Skip `/spec-task` only for trivial fixes | R-019 |
| Development workflow | Verify task not already planned / no number collision | R-050 |
| Development workflow | Task lifecycle mechanics → link | R-017, R-018, R-020–R-023, R-033, R-051 (router) |
| Dispatching agents | Explicit `model` on every dispatch | R-063 |
| Dispatching agents | Pin follows job not `subagent_type` | R-065 |
| Dispatching agents | Never use Fable for background/review | R-067 |
| Dispatching agents | Implementers don't run repo-wide verification | R-074 |
| Dispatching agents | Fresh-context agents, not fork, unless justified | R-076 |
| Dispatching agents | Agent dispatch mechanics → link | R-064, R-066, R-068–R-073, R-075, R-077 (router) |
| Handing off context | Ask whether next unit depends on history or repo state | R-082 |
| Handing off context | Resume from repo state + reports + diagnosis | R-083 |
| Handing off context | Handing off means delegating, not clearing | R-084 |
| Handing off context | Diagnosis written before handoff | R-085 |
| Handing off context | Thresholds are backstops, not triggers | R-086 |
| Handing off context | Context handoff mechanics → link | R-078–R-081, R-087, R-088 (router) |
| Repository conventions | Check `libs/atlas-constants/` first | R-011 |
| Repository conventions | Straightforward moves over re-exported aliases | R-010 |
| Repository conventions | Builder pattern, no `*_testhelpers.go` | R-043 |
| Repository conventions | Write design/plan.md directly, no per-section approval | R-026 |
| Repository conventions | Repo-relative paths only | R-044 |
| Repository conventions | Preserve line endings | R-090 |
| Repository conventions | Ask the toolchain, don't sweep the filesystem | R-091 |
| Repository conventions | Never poll; bound + hand back | R-094 |
| Repository conventions | Glob/Grep first for TODO.md et al. | R-025 |
| Repository conventions | Read pod logs early for wedged deploys | R-052 |
| Repository conventions | Text before `AskUserQuestion` doesn't render | R-061 |
| Repository conventions | Don't pitch paid features proactively | R-062 |
| Repository conventions | Prefer POSIX shell; per-file edits over patch loops | R-089 |
| Where the procedures live | 9-row router table | R-009, R-024, R-045, plus destination-doc summaries of R-047/R-049/R-053/R-056/R-059/R-060/R-064/R-066/R-068–R-095 subsets |

**Result: every sentence in the finished `CLAUDE.md` maps to at least one
`R-NNN` row. No unmapped/invented rule found.**

Two structural observations, neither a defect requiring a fix:

1. **Router pointers appear twice by design.** Each of "Development
   workflow", "Dispatching agents", and "Handing off context" ends with a
   bold inline pointer to its `docs/*.md` companion, and the same
   destination is repeated as a row in the "Where the procedures live"
   table at the end of the file. This is deliberate: the inline pointer
   serves a reader going section-by-section; the table serves the FR-8
   "where is the detailed procedure" scan. Both forms name the same
   destination and neither adds new normative content, so this is not the
   "same rule described normatively in two places" the acceptance
   criterion forbids — it is one router fact, indexed twice.
2. **One table row has no ledger source.** The "Where the procedures live"
   table's last row — "Go service patterns … `backend-dev-guidelines`
   skill, `backend-guidelines-reviewer` agent … DOM-* checklist
   enforcement" — points at pre-existing repo infrastructure (the skill and
   reviewer agent both predate this refactor) rather than at content this
   task relocated. No `R-NNN` row disposes to this destination; the closest
   related row is `R-011` (`libs/atlas-constants` / DOM-21), which is
   independently `keep-compressed` into §Repository conventions. This row
   does not restate any existing `CLAUDE.md` rule with different force —
   it is a discoverability pointer to infrastructure the refactor did not
   touch. Flagged as a concern rather than mechanically fixed, since
   removing or keeping it is an editorial call about the "where is the
   detailed procedure for this kind of work" scanability question, not a
   round-trip defect.

### Step 3 — link check

```
grep -oE '\]\(([^)]+)\)' CLAUDE.md
```

12 links captured, 9 distinct paths (`docs/agent-dispatch.md` linked 3×,
`docs/superpowers-integration.md` linked 2×). All 9 confirmed to exist:
`docs/superpowers-integration.md`, `docs/agent-dispatch.md`,
`docs/verification.md`, `docs/adding-a-new-service.md`,
`docs/packets/PROCESS.md`, `docs/git-workflow.md`,
`docs/reverse-engineering.md`, `docs/tooling-conventions.md`,
`docs/observability.md`.

**Result: pass — no dead links.**

### Step 4 — exclusion check

```
grep -n '\.claude/hooks/' CLAUDE.md
grep -nE '120|CAP\+5|250k|60k|7x|7×|86 modules' CLAUDE.md
```

Both greps produced no output.

**Result: pass — none of the excluded hook paths or magic numbers leaked
into root context.**

### Step 5 — scanability check (FR-8)

Each of the seven FR-8 questions resolves to exactly one `CLAUDE.md`
section, scannable without linear reading:

| FR-8 question | Section |
|---|---|
| What must I never do? | ## Never do this |
| Which workflow applies to this task? | ## Development workflow |
| What evidence standard applies? | ## Evidence & grounding |
| Which worktree and branch should I be in? | ## Where you work — branch & worktree |
| Which agent and model should handle this? | ## Dispatching agents |
| When should I hand work off? | ## Handing off context |
| Where is the detailed procedure for this kind of work? | ## Where the procedures live |

**Result: pass — all seven questions have exactly one home.**

### Step 6 — measurement

```
wc -l -c CLAUDE.md
```

| | Lines | Bytes |
|---|---|---|
| Before | 220 | 19,543 |
| After | 103 | 11,735 |
| Delta | -117 (-53.2%) | -7,808 (-39.96%) |

Reported as an outcome, not a goal.

New documents created by this refactor: exactly four —
`docs/git-workflow.md`, `docs/reverse-engineering.md`,
`docs/tooling-conventions.md`, `docs/agent-dispatch.md` — totaling 11,167
bytes. Three pre-existing documents also grew to absorb relocated/
drop-captured content: `docs/observability.md` (+825 bytes),
`docs/superpowers-integration.md` (+761 bytes), `docs/verification.md`
(+353 bytes). Total size added across `docs/`: **13,106 bytes**.

The trade made visible: root context shrank by 7,808 bytes; total
documentation grew by 13,106 bytes net (content moved out of the
always-loaded root file into task-scoped, load-on-demand documents — it did
not disappear).

### Step 7 — repo gate (Step 8 of the brief)

Not run in this task by controller instruction: repo-wide verification
(`tools/verify.sh`) runs in a separate clean context via the
`atlas-verifier` agent, dispatched by the controller after this task
reports. Treated as done-by-others.

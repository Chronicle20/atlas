# CLAUDE.md Refactor — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Restructure the always-loaded `CLAUDE.md` into invariants + operating defaults + trigger-labeled routers, relocating every procedure, measurement, and hook-implementation detail into an authoritative owner document — without weakening a single rule.

**Architecture:** Documentation-only change. Four new owner documents are created first (`docs/git-workflow.md`, `docs/reverse-engineering.md`, `docs/tooling-conventions.md`, `docs/agent-dispatch.md`), four existing owners are extended, and only then is `CLAUDE.md` rewritten against a rule ledger (`inventory.md`) that makes the relocation auditable in both directions. `.claude/commands/execute-task.md` is rewired to link to the new policy owner rather than restate it.

**Tech Stack:** Markdown only. No Go, TypeScript, test, build, or CI files are touched. Verification is `tools/verify.sh` (near-empty by construction on this branch — it still exercises the repo guards, notably the home-path guard) plus human reading against PRD §10.

**Spec:** [`design.md`](design.md) (PRD: [`prd.md`](prd.md))

## Global Constraints

Copied verbatim from the spec; every task's requirements implicitly include these.

- **No rule weakening.** PRD §2 non-goals: "Changing the meaning, strictness, or applicability of any current rule. This is a relocation and compression task, not a policy revision." When uncertain whether an item should remain direct, bias toward retaining it (PRD §8).
- **No new rules.** Every rule in the new `CLAUDE.md` must trace back to an `R-NNN` row in `inventory.md`. A rule with no source id is either a forbidden policy change or an accidental restatement (design §5.2, reverse direction).
- **Ordering is forced** (design §5.3): ledgers → destination documents → `CLAUDE.md` → `execute-task.md` → round-trip. Rewriting `CLAUDE.md` before its destinations exist produces breadcrumbs pointing at documents that do not answer the question — the exact failure FR-6.3 forbids.
- **Trigger-labeled breadcrumbs only** (design §3.3). `**Packet or protocol work** — start at [docs/packets/PROCESS.md](...)`, never `More information: docs/packets/PROCESS.md`.
- **`(enforced)` markers only** (design §3.4). A rule a hook enforces carries a bare `(enforced)` suffix in `CLAUDE.md` — no hook filename, no threshold, no denial semantics. The behavioral rule is always stated regardless; `(enforced)` never substitutes for it.
- **New-document budget is exactly four.** No fifth document. `docs/packets/PROCESS.md` and `docs/adding-a-new-service.md` are unchanged.
- **Path hygiene.** Repo-relative paths and placeholders only; never a literal `/home/...` or `/Users/...` path in a committed file (enforced by a repo guard).
- **Line-ending preservation** on every edited file. Do not normalize CRLF→LF as a side effect.
- **No service code, test, build file, or CI workflow changes.**
- **Commit per task.** Docs-only; there is no test cycle, so each task's verification is the reading check its steps name.

---

## Task 1: Rule ledger and ownership map

Builds the two durable artifacts that make the rest of the plan auditable. This is the working document for the whole refactor, produced **before** any file is edited (design §5.1) — not a write-up produced afterward.

### Files

- `docs/tasks/task-233-claude-md-refactor/inventory.md` — new file; the FR-1 rule ledger
- `docs/tasks/task-233-claude-md-refactor/ownership.md` — new file; the FR-2 ownership map
- `CLAUDE.md` — read-only in this task; the source being inventoried
- `docs/tasks/task-233-claude-md-refactor/design.md` — read-only; §4 ownership table, §5.1 ledger columns
- `docs/tasks/task-233-claude-md-refactor/prd.md` — read-only; FR-1 classes, FR-2 concerns, FR-4 destination lists

No module root; nothing to build.

### Steps

- [ ] **Step 1: Read the source in full**

Read `CLAUDE.md` end to end. It is 220 lines / 19,543 bytes at branch point. Every heading, paragraph, and bullet is in scope.

- [ ] **Step 2: Write `inventory.md`**

One row per **atomic rule** — not per paragraph. A paragraph carrying three distinct obligations produces three rows. Table columns exactly as design §5.1 specifies:

```markdown
| # | Source | Rule | Class | Disposition | Destination |
|---|---|---|---|---|---|
| R-001 | Workflow Rules | Planning and implementation are separate phases; wait for explicit approval before editing | A | keep-verbatim | `CLAUDE.md` §Development workflow |
| R-014 | Build & Verification | `--base <last-gated-commit>` scopes a per-task run to the increment; a `libs/` commit otherwise fans out to all 86 modules | E | drop-captured | `docs/verification.md` §The iteration gate |
```

Rules for the columns:

- `#` — `R-NNN`, assigned in **source order** through `CLAUDE.md`. Stable ids; later tasks cite them.
- Class — `A` global invariant · `B` global operating default · `C` task router · `D` procedure/reference · `E` rationale/history, per PRD FR-1.
- Disposition — exactly one of `keep-verbatim` · `keep-compressed` · `relocate` · `drop-captured`.
- Destination — owner document **plus section anchor**, or `CLAUDE.md §<section>` for `keep-*`.

`drop-captured` is the only disposition that permits removal without writing anything at the destination, and it requires the destination text to **already exist** — verified by opening the file and reading it, never asserted. If the destination text is thin or absent, the disposition is `relocate`, not `drop-captured`.

- [ ] **Step 3: Cross-check the ledger against PRD FR-3 and FR-4**

Every item PRD FR-3 lists as retained must appear with a `keep-verbatim` or `keep-compressed` disposition. Every item PRD FR-4 lists as relocated must appear with `relocate` or `drop-captured` and the destination FR-4 names. Any disagreement between your classification and the PRD is resolved in the PRD's favor, or recorded as an explicit note in the row.

- [ ] **Step 4: Write `ownership.md`**

The FR-2 map, one row per concern, using design §4's resolved table (which already resolves the PRD's open questions):

| Concern | Owner | Action |
|---|---|---|
| Verification gate | `docs/verification.md` | Extend / consolidate |
| Packet & protocol work | `docs/packets/PROCESS.md` | No change — confirmed sufficient |
| Task lifecycle, task resolution, code-review dispatch | `docs/superpowers-integration.md` | Extend |
| Adding a service | `docs/adding-a-new-service.md` | No change |
| Git / branch / PR workflow | `docs/git-workflow.md` | New |
| Reverse engineering / IDA | `docs/reverse-engineering.md` | New, narrow |
| Shell, editing, tooling conventions | `docs/tooling-conventions.md` | New |
| Agent dispatch, model selection, context handoff | `docs/agent-dispatch.md` | New |
| Runtime / Kubernetes debugging | `docs/observability.md` | Extend |
| Go service patterns (constants, boundaries, builders) | `backend-dev-guidelines` skill + `backend-guidelines-reviewer` | Existing; root keeps one-line bullets |

Add below the table the boundary statement from design §4.3, because it is the one ownership line a maintainer will get wrong:

> `superpowers-integration.md` — *which* command/agent/skill for a situation.
> `agent-dispatch.md` — *how* to dispatch any agent: model, budget, isolation, handoff.

Also record the answer to "where does a new rule go?" as a short decision list, so root context cannot silently re-accumulate procedure.

- [ ] **Step 5: Verify no rule was skipped**

Run:

```sh
grep -c '' CLAUDE.md
grep -n '^\(#\{2,3\} \|- \|\*\*\)' CLAUDE.md
```

Walk the second command's output top to bottom and confirm each heading and bullet is represented by at least one `R-NNN` row. Report the count of rows and the count of source sections covered.

- [ ] **Step 6: Commit**

```sh
git add docs/tasks/task-233-claude-md-refactor/inventory.md docs/tasks/task-233-claude-md-refactor/ownership.md
git commit -m "docs(task-233): rule inventory and ownership map"
```

---

## Task 2: New owners — git workflow and reverse engineering

Two small, tightly scoped new documents. They are grouped because each is short and neither depends on the other.

### Files

- `docs/git-workflow.md` — new file
- `docs/reverse-engineering.md` — new file
- `CLAUDE.md` — read-only; source text is §Git Operations and §Reverse Engineering / IDA
- `docs/tasks/task-233-claude-md-refactor/inventory.md` — read-only, new file created in Task 1; the `R-NNN` rows destined here
- `docs/packets/PROCESS.md` — read-only; the onward pointer target for RE work

Patterns to copy: `docs/verification.md` (heading style — a short lead paragraph stating what the document owns, then `##` sections, no numbered procedure unless the procedure is genuinely ordered).

### Steps

- [ ] **Step 1: Write `docs/git-workflow.md`**

Sections, per design §6:

1. **Branch safety** — never commit or push to `main`; branch protection blocks the push so a commit made on local `main` is stranded and never reaches the remote. Setup work that must precede a feature branch still goes *on* the feature branch — create it first; it branches from the same HEAD. Recovery from a stray `main` commit: preserve the content on a branch (cherry-pick if needed), then `git fetch origin main && git reset --hard origin/main`.
2. **Pushing and history rewrites** — after a rebase/merge/history rewrite, always push (force-push when history was rewritten) so the PR reflects the resolved state. A rebase resolved only locally leaves the PR still showing conflicts.
3. **Build triggering and the conflict exception** — a plain push to a task branch DOES trigger the PR workflows and the `atlas-pr-<N>` ephemeral rollout. Do not merge `origin/main` as a routine build-triggering ritual. The one exception: when the branch conflicts with `main`, the push does not start the build — merge `origin/main`, resolve, push the merge commit. The merge is the conflict resolution, not the trigger.
4. **`gh` authentication** — run `gh` with the token env explicitly cleared so it uses stored `hosts.yml` auth: `env -u GH_TOKEN -u GITHUB_TOKEN gh …`. Do NOT source `~/.config/atlas/gh.env` — its `GH_TOKEN` is expired and takes precedence, causing 401s. Never echo the token.

Cross-link to `docs/runbooks/ephemeral-pr-deployments.md` from section 3 for PR-environment lifecycle.

Keep the account name out of the committed text if it reads as personal configuration; describe it as "the stored `hosts.yml` account" instead.

- [ ] **Step 2: Write `docs/reverse-engineering.md`**

Deliberately **not** an RE tutorial — PRD §10 makes narrow scope an acceptance criterion. Scope, and nothing beyond it:

1. **Confirm the target version first** — the IDB/instance under investigation must match the version you are targeting. The wrong version sends the whole investigation down the wrong path.
2. **Session resolution** — `select_instance(port)` and port-based selection are dead. Resolve the session from `idb_list` by binary **name** and pass it as the `database` parameter to subsequent calls.
3. **Function lookup** — use the `func_query` tool with `name_regex`, the documented method. Do not improvise alternate lookup approaches.

End with an onward breadcrumb: **Deriving a packet field order** — see [`docs/packets/PROCESS.md`](packets/PROCESS.md), the dominant consumer of these mechanics (design §6).

- [ ] **Step 3: Verify the destinations are real, not stubs**

Run `wc -l docs/git-workflow.md docs/reverse-engineering.md` — neither may be a stub. Then confirm each file carries its material:

- `git-workflow.md`: search it for `GH_TOKEN`, `reset --hard`, and `atlas-pr-` — the auth, recovery, and build-trigger sections.
- `reverse-engineering.md`: search it for `idb_list`, `database`, and `name_regex` — session resolution and lookup.

Every rule the inventory routes to these two files must be findable by reading them. Confirm row by row against the `R-NNN` rows whose destination is one of these files.

- [ ] **Step 4: Commit**

```sh
git add docs/git-workflow.md docs/reverse-engineering.md
git commit -m "docs: add git-workflow and reverse-engineering owner documents"
```

---

## Task 3: New owner — tooling conventions

### Files

- `docs/tooling-conventions.md` — new file
- `CLAUDE.md` — read-only; source text is §Shell & Editing Conventions and §File Writing / Conventions
- `docs/tasks/task-233-claude-md-refactor/inventory.md` — read-only, new file created in Task 1; the `R-NNN` rows destined here

### Steps

- [ ] **Step 1: Write `docs/tooling-conventions.md`**

Three sections, per design §6:

1. **Locating Go module source.** Never sweep the filesystem for a dependency's source. `go list -m -f '{{.Dir}}' <module>` prints the directory in ~0.02s, whether the module resolves to the module cache or to a local `replace`. `find /` takes ~2 minutes on WSL2; one task-227 session burned 6 minutes across five sweeps hunting for `atlas-rest`, which `go.mod` had `replace`d to `libs/atlas-rest` inside the worktree the agents were already working in. Guessing at module-cache case-escaping (`!chronicle20`) is the tell that you should have asked `go list`. The same applies to `go doc <pkg>` for a symbol and `go list -m all` for the version set. `find` is for paths you own, rooted at a directory you name — never at `/`.
2. **Waiting on processes.** Never spend inference turns waiting. Launch once with a bound — background execution, or a monitor with an until-loop — and do something else or hand back. Repeated `sleep` / `ps aux | grep` / `echo waiting` / `for i in $(seq …); do sleep` calls are the anti-pattern: each re-reads the whole context to learn nothing, and they cluster late in a session where that is most expensive. If the process exceeds its bound, kill it and fall back; do not keep polling. When a tool has a known hang mode, the fallback belongs in that tool's agent doc, not in a longer wait.
3. **Shell and editing conventions.** Prefer portable POSIX shell; avoid zsh/direnv-specific constructs and batch patch loops that can produce garbled or unapplied output. For a multi-file edit prefer per-file Edit/Write over a shell patch loop. Preserve line endings when editing — normalizing CRLF→LF as a side effect inflates diffs with spurious changes. Use repo-relative paths or placeholders in committed files; never literal home or absolute paths.

The measurements and the task-227 anecdote belong **here**, not in `CLAUDE.md` — that is the point of the relocation (PRD FR-4).

- [ ] **Step 2: Verify coverage**

Search the new file for `go list -m`, `go doc`, and the `find /` anti-pattern (section 1), and for `CRLF`, `POSIX`, and `repo-relative` (section 3). Each must resolve to real text.

Confirm every `R-NNN` row destined here resolves to text in the file.

- [ ] **Step 3: Commit**

```sh
git add docs/tooling-conventions.md
git commit -m "docs: add tooling-conventions owner document"
```

---

## Task 4: New owner — agent dispatch and context handoff

The largest of the four new documents. It becomes the policy owner that `.claude/commands/execute-task.md` links to in Task 8.

### Files

- `docs/agent-dispatch.md` — new file
- `CLAUDE.md` — read-only; source text is §Model & Cost Preferences and §Context Handoff
- `.claude/commands/execute-task.md` — read-only in this task; Step 4a's model table (lines 67–96), Step 4c's verification split (147–207), Step 4d's `PARTIAL` handling (209–234), Step 4e's handoff arithmetic (236–274) are the source of the material this document takes ownership of
- `docs/superpowers-integration.md` — read-only; its "Phase 4 context budget" section (lines 16–29) is reduced to a pointer at this file in Task 6
- `docs/tasks/task-233-claude-md-refactor/inventory.md` — read-only, new file created in Task 1

### Steps

- [ ] **Step 1: Write `docs/agent-dispatch.md`**

Lead paragraph must state the ownership boundary explicitly (design §4.3), because this is the line a maintainer will otherwise get wrong:

> This document owns *how* to dispatch any agent — model, budget, isolation, handoff — for every dispatch in every session, including ad-hoc ones outside the four-phase workflow. `docs/superpowers-integration.md` owns *which* command, agent, or skill to reach for in a given situation.

Sections, per design §6:

1. **Model selection.** Pass an explicit `model` on every dispatch; an unspecified model inherits the main-loop model (Opus), and an Opus subagent turn costs ~7× a Sonnet one. The pin follows the **job**, not the `subagent_type` — named reviewer agents carry a Sonnet frontmatter pin, but an ad-hoc `general-purpose` dispatch carrying a review prompt does not; that is the hole this rule closes. Reproduce the full job → model table verbatim from `.claude/commands/execute-task.md` Step 4a (six rows: review/verify/audit → `sonnet` always; scan/inventory/doc sweep → `haiku`; `atlas-verifier` → `haiku`; `atlas-implementer` → `sonnet`; `packet-implementer` / `dispatcher-family-implementer` → `sonnet`; a plan task tagged `model: opus` → `opus`). Include the `model: opus` opt-in criteria (derivation-heavy work: IDA/packet field-order derivation, saga orchestration across services, cross-service contract change) and the escalate-after-two-failures rule. Never use Fable for background or review workflows.
2. **The implementer budget.** 120 tool calls, warned at 100, counted by `.claude/hooks/turn-budget.sh` and contracted in `.claude/agents/atlas-implementer.md`, `.claude/agents/packet-implementer.md`, and `.claude/agents/dispatcher-family-implementer.md`. At the cap the implementer commits and reports `PARTIAL`; the controller dispatches a continuation. The cap is binding: `.claude/hooks/turn-budget-guard.sh` (PreToolUse) denies subagent calls past CAP+5, exempting the commit-and-report path; controllers are never blocked. The number is changed in the counting hook only. State the underlying arithmetic: context grows with turn count and every turn re-reads all of it, so one 600-turn agent costs far more than the same work split across fresh contexts.
3. **Verification split.** Implementers never run `tools/verify.sh`, `tools/lint.sh`, `-race`, or docker bake — a `--quick` run inside a 400k-token implementer costs a large multiple of the same run in a clean 20k one, and its output is the biggest avoidable consumer of an implementer's window. Implementers run module-local `go build ./... && go test ./...` and nothing more; the repo gate belongs to `atlas-verifier`. Point at `/execute-task` Step 4c for the concurrency procedure (launch, keep going, reconcile, at most one gate in flight) — that is command mechanics and stays there.
4. **Fork vs fresh context.** Fan out with fresh-context named agents plus an explicit brief, not `subagent_type: "fork"`. Fork only to continue an interactive debugging thread, and say why inline. `.claude/hooks/fork-dispatch-guard.sh` denies an unjustified fork and states the cost.
5. **Context handoff.** The unit of work is a briefable task, not a conversation; cost scales with turn count × context size, so 50 turns carried at 190k cost roughly ten times the same 50 turns at 19k. The decision criterion at every durable boundary. Handing off means delegating, not clearing — `/clear` is a user action. The diagnosis is written down before the handoff, one paragraph into the task folder. The floor: below roughly 60k a fresh agent re-discovers files you already hold; under ~40 tool calls prefer continuing (`.claude/hooks/commit-boundary.sh` raises the question at commits past the floor). The backstop: ~250k for a controller, with the measured 18-task run (controller finished at 402k having produced 165KB of its own output across 157 calls; its last 42 self-contained turns ran at 360–400k each where a fresh session would have run them at ~80k). Generate briefs with `tools/task-brief.sh`, never by hand out of `plan.md`. The durable artifacts are `task-N-report.md` and the SDD ledger `.superpowers/sdd/<plan>/progress.md`. Point at `/execute-task` Steps 4c–4e for the procedural forms.

Hook filenames, thresholds, and denial semantics belong **here** — this is the document that may name them (PRD FR-5).

- [ ] **Step 2: Verify the model table matches its source exactly**

Print the source table with `sed -n '78,86p' .claude/commands/execute-task.md` and compare it against the copy in `docs/agent-dispatch.md` row by row (search the new file for `sonnet`, `haiku`, `opus`).

Every row must survive with the same job description, the same model, and the same "always / no exceptions" force on the review row. Task 8 deletes the source table; a lossy copy here becomes a silent policy change.

- [ ] **Step 3: Verify coverage**

Search the new file for `turn-budget`, `fork-dispatch-guard`, `commit-boundary`, and `task-brief`. All four are hook or tool names this document is the one place allowed to carry; a miss means a section is thin.

Confirm every `R-NNN` row destined here resolves to text in the file.

- [ ] **Step 4: Commit**

```sh
git add docs/agent-dispatch.md
git commit -m "docs: add agent-dispatch owner document"
```

---

## Task 5: Extend the verification and observability owners

### Files

- `docs/verification.md` — modify; add the flagged-run-is-not-a-pass statement and background-execution guidance
- `docs/observability.md` — modify; add a "Diagnosing a runtime failure" section
- `CLAUDE.md` — read-only; source text is §Build & Verification and §Debugging / Kubernetes
- `docs/runbooks/ephemeral-pr-deployments.md` — read-only; the cross-link target for env-lifecycle wedges
- `docs/tasks/task-233-claude-md-refactor/inventory.md` — read-only, new file created in Task 1

### Steps

- [ ] **Step 1: Read `docs/verification.md` and mark what is already captured**

Most of PRD FR-4's verification list is already in this file: the flag list (lines 6–13), CI equivalence and "CI is the authority" (15–17), the iteration gate with the 86-modules-vs-2 measurement (25–46), the Go layer (48–58), the docker layer and the `COPY libs/...` failure (60–80), guards, lint, and known CI drift. Those inventory rows take `drop-captured` — verified by reading, which you are doing now. This is where the largest single block of root text leaves with no new writing.

- [ ] **Step 2: Add the two genuinely missing statements to `docs/verification.md`**

The file presents the flag list neutrally. Add, near the top after the flag block, an explicit statement:

> Only the **flagless** invocation counts as verified. `--quick`, `--no-docker`, and `--all` also exit 0 — the first two print a caveat and skip the bake and `-race` — so "verify.sh exited 0" is not a pass unless it ran with no flags. Never claim verified from a subset.

And in "The iteration gate", add the background-execution guidance:

> Launch the gate in the background and keep working; never idle waiting on it.

Do not restate the check list — the file already says the list lives in the script, "do not maintain a second copy of it in `CLAUDE.md` or anywhere else." Honor that.

- [ ] **Step 3: Add "Diagnosing a runtime failure" to `docs/observability.md`**

Place it after "Filtering by environment" (which documents the Loki/Prometheus/Grafana access paths this section builds on), before "Log field naming". Content:

- For a wedged deploy or a runtime failure, read the relevant pod logs **early** — via `mcp__kubernetes__pods_log` — rather than starting at packet-level fixes or bare pod listings. The logs usually name the real root cause directly.
- Which services to read first for a wedged startup: `atlas-character-factory`, `atlas-world`.
- The Loki selector caveat that makes a query silently return nothing (there is no `app` label; select on `service_name`) if it is not already stated elsewhere in the file — check before adding, and do not duplicate.
- Cross-link: for an ephemeral PR environment that will not come up or will not tear down, see `docs/runbooks/ephemeral-pr-deployments.md` §9.3 / §9.4 — that runbook owns env lifecycle, this section owns reaching logs in any environment.

- [ ] **Step 4: Verify**

Search `docs/verification.md` for `flagless` and `not a pass`, and `docs/observability.md` for `pods_log` and the new `Diagnosing a runtime failure` heading.

- [ ] **Step 5: Commit**

```sh
git add docs/verification.md docs/observability.md
git commit -m "docs: consolidate verification gate wording and add runtime-failure diagnosis"
```

---

## Task 6: Extend and de-duplicate the superpowers-integration owner

Two directions at once: this file gains the task-lifecycle mechanics leaving `CLAUDE.md`, and loses two sections that duplicate other owners.

### Files

- `docs/superpowers-integration.md` — modify; add task-resolution and code-review mechanics, reduce two sections to pointers
- `CLAUDE.md` — read-only; source text is §Development Workflow, §Artifact Location Override, §Code Review Pattern, §Task Workflow, §Code Review Before PR
- `docs/packets/PROCESS.md` — read-only; the pointer target replacing §Packet Work
- `docs/agent-dispatch.md` — read-only, new file created in Task 4; the pointer target replacing §Phase 4 context budget
- `tools/task-resolve.sh` — read-only; the output format being documented
- `tools/task-numbers.sh` — read-only; the `next` / `list` subcommands being documented

### Steps

- [ ] **Step 1: Add task resolution mechanics**

New section (place it under "The Four-Phase Workflow"):

- Phase commands accept fuzzy identifiers: `task-054-slug`, `task-054`, `054`, `54` all resolve to the same folder.
- `tools/task-resolve.sh <identifier>` prints one tab-separated line: `<task-id>\t<task-dir>\t<worktree>`. Exit 3 → no match; exit 4 → ambiguous, candidates on stderr.
- `tools/task-resolve.sh --list` shows every existing task, one row per task, already deduplicated across worktrees.
- `tools/task-numbers.sh next` picks the number for a new task; check both before planning so a number does not collide with an in-flight task.
- **Never glob `.worktrees/*/docs/tasks/task-*`** — every worktree carries a full copy of `docs/tasks/` from its branch point, so that pattern returns (tasks × worktrees) mostly-duplicate paths into context to resolve a single id.
- Searching for a task artifact: search across all worktrees (`git worktree list`) before concluding a file is missing.

- [ ] **Step 2: Add the artifact-location override mechanics**

`superpowers:brainstorming` and `superpowers:writing-plans` default to `docs/superpowers/specs/` and `docs/superpowers/plans/`. In this project both go under `docs/tasks/task-NNN-slug/`. When invoking those skills directly, outside the phase commands, pass the task folder explicitly so artifacts land in the right place.

- [ ] **Step 3: Extend the Code Review section**

Add to the existing "Code Review" section:

- Each agent writes findings to `docs/tasks/task-NNN-slug/audit.md` (backend also writes `audit.json`).
- Code review is mandatory before opening a PR and is a **different gate** from verification. Reproduce the cross-service defect examples that `CLAUDE.md` currently carries: a producer empties a compartment the consumer still reads; a new saga action with no step handler in the orchestrator; a class of missing emits that existing tests actively pin as the old behavior. When a change crosses a service boundary, trace the event into its consumers by hand and check that a test asserts the NEW contract, not the old silent drop.

- [ ] **Step 4: Reduce §Packet Work to a pointer**

Design §4.1 confirmed `docs/packets/PROCESS.md` already carries every entry point, the leaf step, the version set, baseline status, and the CI gate list — plus a maintenance row this file never mentioned. Replace the duplicated table (lines 55–65) with a trigger-labeled pointer. **Keep** the `packet-completeness-critic` paragraph (line 67) — it is about *which agent to run before a packet PR*, which is this document's subject, not `PROCESS.md`'s.

- [ ] **Step 5: Reduce §Phase 4 context budget to a pointer**

Replace the three-row control table (lines 16–29) with a pointer at `docs/agent-dispatch.md`, keeping only the one sentence that is this document's own subject: `atlas-implementer` replaces `general-purpose` for every Phase 4 implementation dispatch, and its contracts override the plugin's `implementer-prompt.md` where they disagree.

- [ ] **Step 6: State the ownership boundary**

Add the design §4.3 boundary line to this file too, mirroring the one in `docs/agent-dispatch.md`, so the split is legible from either side.

- [ ] **Step 7: Verify**

Search `docs/superpowers-integration.md` for `task-resolve`, `task-numbers`, `agent-dispatch`, and `packets/PROCESS` — the added mechanics and the two new pointers.

Confirm no procedure is now described normatively in both this file and `PROCESS.md` or `agent-dispatch.md`.

- [ ] **Step 8: Commit**

```sh
git add docs/superpowers-integration.md
git commit -m "docs: superpowers-integration owns task resolution; packet and dispatch sections become pointers"
```

---

## Task 7: Rewrite `CLAUDE.md`

Every destination now exists and holds its content. This task rewrites root context against the ledger.

### Files

- `CLAUDE.md` — modify (rewritten)
- `docs/tasks/task-233-claude-md-refactor/inventory.md` — read-only, new file created in Task 1; the ledger this rewrite is written against
- `docs/tasks/task-233-claude-md-refactor/ownership.md` — read-only, new file created in Task 1; the source of the routing table
- `docs/tasks/task-233-claude-md-refactor/prd.md` — read-only; FR-3 is the floor for what stays, FR-8 the scanability contract

### Steps

- [ ] **Step 1: Lay down the section skeleton**

Exactly this order (design §3.2) — each section answers one FR-8 scanability question:

```
# Atlas
  (1-line orientation: Go microservices monorepo, 14+ services; TypeScript only in atlas-ui)

## Never do this                       → What must I never do?
## Evidence & grounding                → What evidence standard applies?
## Where you work — branch & worktree  → Which worktree and branch?
## Done means verified                 → completion gate + review gate
## Development workflow                → Which workflow applies?
## Dispatching agents                  → Which agent and model?
## Handing off context                 → When do I hand off?
## Repository conventions              → compact one-line bullets
## Where the procedures live           → Where is the detailed procedure?
```

- [ ] **Step 2: Write "Never do this" as the consolidated lead**

This is the one place the design deliberately **reorders** rather than relocates (design §3.2). Today the hard prohibitions are scattered across four sections. Consolidate them here as a short list; the topical sections below then carry only the positive procedure and must **not** restate the prohibition.

At minimum, from FR-3: never commit or push to `main`; never edit main-repo files when a task worktree exists for that work; never invent a value, name, opcode, or output; never claim verified from a flagged or partial run; never open a PR without code review; never dispatch an agent without an explicit `model`; never land a placeholder comment, a stubbed handler, or an unimplemented status response.

- [ ] **Step 3: Write the Tier-1 sections**

**Evidence & grounding** — never invent (unverified is labeled "unknown / unverified"; "I think it's X" is a lead to check, not a finding; quote actual tool output rather than paraphrasing numbers from memory); repo source, WZ data, IDA, and live output outrank remembered general MapleStory knowledge; confirm the exact server/tenant/client version before investigating; do not present a spot-check as a sweep; finish producible work — do not declare a documented gap, follow-up task, or "out of scope" when the blocker is a prerequisite you can produce yourself, and preserve the distinction between (i) producible now, (ii) a genuine external blocker, (iii) an ambiguous design decision, and (iv) evidence that cannot currently be obtained.

**Where you work — branch & worktree** — check the branch before every commit; non-trivial tasks live in their task worktree, verify cwd before task work; subagents operate inside the correct worktree and the tree is verified clean after they run; search all worktrees before concluding an artifact is missing; push after any history rewrite so the PR reflects the resolved state; do not merge `main` merely to trigger a build.

**Done means verified** — flagless `tools/verify.sh` is the authoritative completion gate; a flagged or subset run is not a pass; never claim done / ready-for-PR without it; code review is mandatory before a PR and is a *different gate* — a green gate does not mean the branch is correct, because every module can build, vet, test, and bake clean while the branch carries a cross-service seam defect; a cross-service change needs a test that asserts the NEW contract.

**Development workflow** — the canonical ordering `spec → design → plan → execute → review/verify/finish`, with the four slash commands and their outputs, and enough detail that an agent cannot skip a phase, create artifacts in the wrong repo or worktree, or create a second worktree during execution. Route the mechanics (fuzzy identifiers, resolver output format, artifact-location override, reviewer agents) to `docs/superpowers-integration.md`.

- [ ] **Step 4: Write the Tier-2 sections**

**Dispatching agents** — explicit `model` on every dispatch; review/verify/audit → Sonnet, scan/inventory → Haiku, implement → Sonnet unless the plan task is tagged `model: opus`; never Fable for background or review work; prefer fresh-context named agents over forks *(enforced)*; implementers verify module-locally only, the repo gate belongs to the verifier agent. Router to `docs/agent-dispatch.md`.

**Handing off context** — the decision criterion only: at a durable boundary, ask whether the next unit of work depends materially on this conversation's history or only on repository state; if repo state suffices, write the diagnosis down *before* handing off, then delegate to a fresh agent; `/clear` is a user action, an agent cannot clear itself; size thresholds are backstops, dependency is the primary signal *(enforced)*. Router to `docs/agent-dispatch.md` for the thresholds and mechanics.

**Repository conventions** — the FR-3 compact bullet list, one line each: check `libs/atlas-constants/` before defining a new domain type, alias, or numeric constant; prefer straightforward moves over re-exported type aliases and do not cross service boundaries into another layer's internals; Builder pattern for test setup, no `*_testhelpers.go`; write design/plan documents to file in full, no per-section approval; repo-relative paths or placeholders in committed files, never literal home/absolute paths *(enforced)*; preserve existing line endings; ask the toolchain (`go list -m -f '{{.Dir}}' <module>`) rather than sweeping the filesystem for a Go dependency; never spend inference turns polling a process — launch it with a bound; locate tracking docs with Glob/Grep rather than assuming a path; for a runtime or deploy failure read the relevant pod logs early; send substantive content as its own message before an `AskUserQuestion`; do not proactively pitch paid features.

Drop from the constants bullet the enumeration of what the library covers and the `DOM-21` identifier — keep the rule and the path (design §4.4).

- [ ] **Step 5: Write "Where the procedures live" as a table**

One row per concern: **trigger condition** · owner document · one clause on what is there. It is the reader-facing form of `ownership.md`. Rows for all ten concerns in the ownership map. Every destination is a `docs/` path — never a `.claude/commands/*` path for policy content (FR-7).

- [ ] **Step 6: Enforce the exclusions**

Read the finished file and delete anything matching:

- A `.claude/hooks/*` filename, a numeric enforcement threshold (120, 100, CAP+5, ~250k, ~60k, ~7×, 86 modules, 402k), or denial semantics. `(enforced)` markers only.
- Narrative incident history (task-227, task-231, the 48-of-109 idle minutes, the 18-task controller run).
- Any long explanatory paragraph, command tutorial, or deeply nested procedural step (FR-8).

- [ ] **Step 7: Run the reverse round-trip on this file**

For each rule in the new file, find its `R-NNN` in `inventory.md`. Any sentence with no source id is either a policy change — forbidden by PRD §2 — or an accidental restatement. Remove or correct it. This direction is the one that is easy to skip and the one that catches policy drift (design §5.2); it is a required step.

- [ ] **Step 8: Verify**

```sh
grep -n '\.claude/hooks/' CLAUDE.md
grep -oE '\]\(([^)]+)\)' CLAUDE.md
```

The first command must now print nothing. For the second, check each captured path exists on disk.

- [ ] **Step 9: Commit**

```sh
git add CLAUDE.md
git commit -m "docs: restructure CLAUDE.md into invariants, defaults, and trigger-labeled routers"
```

---

## Task 8: Rewire `.claude/commands/execute-task.md` to link, not restate

### Files

- `.claude/commands/execute-task.md` — modify; Step 4a's model table and Steps 4d/4e's handoff arithmetic replaced with links
- `docs/agent-dispatch.md` — read-only, new file created in Task 4; the link target
- `docs/tasks/task-233-claude-md-refactor/ownership.md` — read-only, new file created in Task 1; FR-7 direction of ownership

### Steps

- [ ] **Step 1: Replace Step 4a's policy with a link**

Delete the job → model table (lines 78–86) and the surrounding policy prose that `docs/agent-dispatch.md` now owns (the ~7× comparison, the pin-follows-the-job explanation, the `model: opus` opt-in criteria, the escalate-after-two-failures rule). Leave a short pointer:

> Model selection for every dispatch — the job → model table, the `model: opus` opt-in, and the escalation rule — is owned by [`docs/agent-dispatch.md`](../../docs/agent-dispatch.md). Pass an explicit `model` on every dispatch.

Verify the relative path resolves from `.claude/commands/`.

- [ ] **Step 2: Replace the handoff arithmetic in Steps 4d and 4e with links**

The **procedural** steps stay — they are command mechanics, not policy: what to ledger on a `PARTIAL`, how to write the continuation brief, the fresh-implementer dispatch, the single review over the whole task range; and in 4e, confirming the ledger, the message to the user, and stopping. What leaves: the measured 18-task controller numbers, the 402k / 165KB / 157-calls figures, the fresh-session comparison, and the cost arithmetic — all now in `docs/agent-dispatch.md` §Context handoff. Replace with a one-line pointer at that section, keeping the ~250k trigger threshold in the step because the step is where an agent acts on it.

- [ ] **Step 3: Leave Step 4c alone except for a pointer**

Step 4c's concurrency procedure (launch, keep going, reconcile, at most one gate in flight) is command mechanics and stays in full. Add a pointer at `docs/agent-dispatch.md` §Verification split for the *why*, and remove the duplicated cost explanation if it now reads as a restatement.

- [ ] **Step 4: Verify no policy is now stated in two places**

Search `.claude/commands/execute-task.md` for `agent-dispatch` (the pointers must be there) and for `sonnet` / `haiku`.

The second command should return only pointer text and any per-step model pin that is genuinely a dispatch instruction, not a policy table.

- [ ] **Step 5: Commit**

```sh
git add .claude/commands/execute-task.md
git commit -m "docs: execute-task links to agent-dispatch for model and handoff policy"
```

---

## Task 9: Round-trip audit, link check, and measurement

The acceptance pass. Nothing is written here except the audit record.

### Files

- `docs/tasks/task-233-claude-md-refactor/inventory.md` — modify (new file created in Task 1); append the round-trip result and the measurement
- `CLAUDE.md` — read-only; the subject of the audit
- `docs/tasks/task-233-claude-md-refactor/prd.md` — read-only; §10 acceptance criteria

### Steps

- [ ] **Step 1: Forward round-trip — nothing lost**

Walk every `R-NNN` row. For `keep-verbatim` / `keep-compressed`, the rule must be findable in the new `CLAUDE.md`. For `relocate` / `drop-captured`, it must be findable in the named destination section — verified by opening the file, not by trusting the row. Record any row that fails, fix it, and re-check.

- [ ] **Step 2: Reverse round-trip — nothing invented**

Walk the new `CLAUDE.md` sentence by sentence and map each rule to its `R-NNN`. Record the mapping. Any unmapped rule is fixed, not explained.

- [ ] **Step 3: Link check**

```sh
grep -oE '\]\(([^)]+)\)' CLAUDE.md
```

Confirm every captured path exists. A path that does not resolve is a hard failure of the "no dead links" NFR.

- [ ] **Step 4: Exclusion check**

```sh
grep -n '\.claude/hooks/' CLAUDE.md
grep -nE '120|CAP\+5|250k|60k|7x|7×|86 modules' CLAUDE.md
```

Both must print nothing. If a number is genuinely load-bearing in root context, that is a decision to surface, not to make silently.

- [ ] **Step 5: Scanability check**

Read the finished `CLAUDE.md` and answer each of the seven FR-8 questions by scanning to exactly one section. Record which section answers which question. A question with no single home means the skeleton needs a fix.

- [ ] **Step 6: Measure**

```sh
wc -l -c CLAUDE.md
```

Before: **220 lines / 19,543 bytes** (~5k tokens). Record after, and the delta, framed as an **outcome** measure — never as the goal. Also record the new-document count (must be exactly four) and the total size added across `docs/`, so the trade is visible: root context shrinks, total documentation grows.

- [ ] **Step 7: Append the audit record to `inventory.md`**

New `## Round-trip verification` section holding: forward result, reverse result, the reverse mapping, link-check result, exclusion-check result, the scanability question → section map, and the measurement. This is the artifact the PR description quotes.

- [ ] **Step 8: Run the repo gate**

```sh
tools/verify.sh
```

Flagless, must exit 0. It is near-empty by construction on this branch (no Go module changed) but it exercises the repo guards, including the home-path guard on the new documents — which is the one way this branch can fail mechanically. Launch it in the background and do the reading checks meanwhile; never idle waiting on it.

- [ ] **Step 9: Commit**

```sh
git add docs/tasks/task-233-claude-md-refactor/inventory.md
git commit -m "docs(task-233): round-trip audit and size measurement"
```

---

## Acceptance

The task is complete when every box above is checked and every PRD §10 criterion holds. The criteria that are not mechanically checkable — "every item in FR-3 is still present with undiminished force", "no procedure is described normatively in two places", "the four new documents are non-trivial" — are settled by human reading against `inventory.md`, which is exactly what the ledger exists to make possible.

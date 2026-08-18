# CLAUDE.md Refactor — Progressive Disclosure for Always-Loaded Context

Version: v1
Status: Draft
Created: 2026-08-16
---

## 1. Overview

The root `CLAUDE.md` is loaded into every Claude Code session in this repository, unconditionally, before the agent knows what it is being asked to do. It has grown to 220 lines and currently mixes four distinct kinds of material: behavioral invariants, operating defaults, task-specific procedures, and historical incident narrative. Sections such as *Build & Verification*, *Model & Cost Preferences*, *Context Handoff*, and *Shell & Editing Conventions* each carry multiple paragraphs of mechanics, thresholds, hook filenames, and post-mortem detail that only matter once an agent is already inside the corresponding workflow.

That material is not wrong — most of it is hard-won and still load-bearing. It is in the wrong place. Detail that is only actionable after entering a workflow costs context in every session that never enters it, and its bulk dilutes the invariants that genuinely must be visible up front. Meanwhile several concerns the root file describes in detail (Git workflow, IDA/reverse engineering, shell and tooling conventions, agent dispatch, context handoff) have **no authoritative document at all** — `CLAUDE.md` is their only home, which is precisely why they cannot currently be trimmed.

This task restructures `CLAUDE.md` into **invariants + routing + critical defaults**, and moves procedures, mechanics, rationale, and incident history into reference documents reachable by explicit trigger-labeled breadcrumbs. Where an authoritative document already exists (`docs/verification.md`, `docs/packets/PROCESS.md`, `docs/superpowers-integration.md`, `docs/adding-a-new-service.md`), material is consolidated into it. Where none exists, a minimal new document is created — but only for the four concerns that demonstrably lack one. The governing test for every retained line is: *could the agent reliably know it needs this information before it has already made the decision the information governs?*

Reduction in size is an **outcome** of better information architecture, not the objective. No rule may be weakened, dropped, or softened in order to make the file shorter.

## 2. Goals

Primary goals:

- Restructure `CLAUDE.md` so its content is limited to global invariants, global operating defaults, and trigger-labeled routers to authoritative documentation.
- Establish exactly one authoritative owner document for each major operational concern, and make that ownership explicit.
- Eliminate duplicated procedures between `CLAUDE.md` and reference docs, and between reference docs and `.claude/commands/*` / `.claude/agents/*`.
- Create the four missing authoritative documents (Git workflow, reverse engineering/IDA, tooling conventions, agent dispatch & context handoff) so that root-level detail has somewhere correct to go.
- Ensure every breadcrumb both (a) states the trigger condition that tells an agent when to follow it, and (b) resolves to a document that actually contains the answer.
- Preserve full semantic strength of every existing rule.

Non-goals:

- Changing the meaning, strictness, or applicability of any current rule. This is a relocation and compression task, not a policy revision.
- Modifying `.claude/hooks/*` behavior, thresholds, or enforcement logic.
- Modifying `~/.claude/RTK.md` or any user-global instruction file — those are outside the repository.
- Reorganizing `docs/` beyond the concerns enumerated in §4.
- Adding automated validation (lint rule, size cap, link checker) for `CLAUDE.md`. Explicitly declined; acceptance is by human reading.
- Rewriting `.claude/commands/*` or `.claude/agents/*` content, beyond replacing duplicated procedure blocks with links to the new owner documents.
- Any change to service code, tests, or build tooling.

## 3. User Stories

- As an agent starting a fresh session, I want the always-loaded context to be short and scannable so that I spend my context budget on the actual task rather than on procedures for workflows I will never enter.
- As an agent about to commit, I want the branch-safety invariant to be immediately visible so that I never strand a commit on local `main`.
- As an agent beginning a packet, IDA, Kubernetes, or new-service task, I want a labeled trigger that names both the situation and the destination document so that I can navigate to the authoritative procedure in one hop.
- As an agent dispatching a subagent, I want a compact model-selection rule in root context so that I never inherit Opus by omission.
- As an agent reaching a durable boundary, I want the handoff *decision criterion* in root context and the *mechanics* one link away.
- As a maintainer adding a new rule, I want an unambiguous answer to "which document owns this?" so that root context does not silently re-accumulate procedure.
- As a maintainer reading the refactored `CLAUDE.md`, I want to answer "what must I never do / which workflow applies / what evidence standard applies / which worktree and branch / which model / when do I hand off / where is the detailed procedure" by scanning, not by linear reading.

## 4. Functional Requirements

### FR-1 — Rule inventory and classification

Every rule, paragraph, and bullet currently in `CLAUDE.md` must be classified into exactly one of:

| Class | Meaning | Disposition |
|---|---|---|
| **A** Global invariant | Violation immediately produces incorrect, unsafe, or hard-to-recover work | Remains directly in `CLAUDE.md`, stated explicitly |
| **B** Global operating default | Broadly applicable behavior, expressible far more concisely | Governing rule stays; mechanics relocate |
| **C** Task router | Exists to recognize a category of work and direct to its workflow | Becomes a trigger → destination line |
| **D** Procedure / reference | Only actionable after entering a specific workflow | Relocates to the owner document |
| **E** Rationale / history | Incidents, measurements, cost comparisons, prior failures, hook origin stories | Relocates, or is dropped if already captured at the destination |

The classification must be recorded as a durable artifact in the task folder (`inventory.md`), with one row per source rule giving: source section, class, disposition, and destination document. This artifact is the evidence that nothing was silently dropped, and is a required deliverable.

### FR-2 — Documentation ownership map

The task must establish and record a single authoritative owner for each of the following concerns. `CLAUDE.md` links to these; it does not compete with them.

| Concern | Owner | Status |
|---|---|---|
| Verification gate | `docs/verification.md` | Exists — extend |
| Packet / protocol work | `docs/packets/PROCESS.md` | Exists — verify sufficiency, extend only if a breadcrumb would otherwise dangle |
| Task lifecycle, worktree/task resolution, code review dispatch | `docs/superpowers-integration.md` | Exists — extend |
| Adding a service | `docs/adding-a-new-service.md` | Exists — no change expected |
| Git / branch / PR workflow | `docs/git-workflow.md` | **New** |
| Reverse engineering / IDA | `docs/reverse-engineering.md` | **New**, narrowly scoped |
| Shell, editing, and tooling conventions | `docs/tooling-conventions.md` | **New** |
| Agent dispatch, model selection, context handoff | `docs/agent-dispatch.md` | **New** |
| Runtime / Kubernetes debugging | `docs/observability.md` or `docs/runbooks/` | Exists — confirm a home; extend the chosen one |

The ownership map must be recorded in the task folder (`ownership.md`) and summarized in whichever root section is most natural, so that a future maintainer can answer "where does this rule go?" without re-deriving the map.

Where two existing documents describe the same workflow, the task must consolidate them or explicitly designate one as authoritative and reduce the other to a pointer.

### FR-3 — Content retained directly in `CLAUDE.md`

The following must remain directly visible and difficult to overlook. This list is a floor, not a ceiling.

**Planning discipline**
- Planning and implementation are separate phases; do not implement while asked to understand or plan; wait for explicit approval before editing.

**Evidence and grounding policy** (retained prominently — it governs investigations before the agent knows which document it needs)
- Never invent values, names, opcodes, output, or behavior. Unverified information must be labeled unknown/unverified.
- Repository source, WZ data, IDA, and live output outrank remembered general MapleStory knowledge.
- Confirm the exact target server/tenant/client version before investigating.
- Do not present a spot-check as a sweep.
- Do not defer a prerequisite you can produce yourself as "out of scope" or a follow-up task. The distinction between (i) something producible now, (ii) a genuine external blocker, (iii) an unresolved design decision, and (iv) evidence that cannot currently be obtained must be preserved.

**Branch and worktree safety**
- Never commit or push directly to `main`; check the branch before every commit.
- Non-trivial tasks live in their task worktree; verify cwd before task work.
- Never edit main-repo files when a task worktree exists for that work.
- Subagents must operate inside the correct worktree.
- Search all worktrees before concluding a task artifact is missing.
- Push after any history-rewriting operation so the PR reflects resolved state.
- Do not merge `main` merely to trigger a build.

**Completion and verification claims**
- Flagless `tools/verify.sh` is the authoritative completion gate.
- `--quick`, `--no-docker`, and any subset do not constitute verification; exiting 0 under a flag is not a pass.
- Never claim done / ready-for-PR without the flagless gate.
- Code review is mandatory before opening a PR, and is a *different gate* from verification: a green gate does not mean the branch is correct.
- Cross-service changes require contract/seam reasoning that compilation cannot prove; a test must assert the new contract.

**Development lifecycle**
- The canonical ordering `spec → design → plan → execute → review/verify/finish`, with enough detail that an agent cannot skip a phase, create artifacts in the wrong repo or worktree, or create a second worktree during execution.

**Agent dispatch defaults**
- Every Agent/Task dispatch specifies an explicit `model`.
- Review / verify / audit → Sonnet; scan / inventory → Haiku; implement → Sonnet unless the plan task is tagged Opus.
- Never use Fable for background or review work.
- Prefer fresh-context named agents over forks.
- Implementers run module-local verification only; repo-wide verification belongs to the verifier agent.

**Context handoff decision**
- At durable boundaries, decide whether the next unit of work materially depends on conversation history or is resumable from durable repository state.
- If repo state suffices, write the diagnosis down *before* handing off, then delegate to a fresh agent.
- `/clear` is a user action; an agent cannot clear itself.
- Context size thresholds are backstops; dependency is the primary signal.

**Task numbering**
- Check existing and in-flight tasks with the canonical tooling before assigning or planning a task number.

**Debugging default**
- For runtime and deployment failures, read the relevant pod logs early rather than reasoning upward from packet behavior or pod status alone.

**Short repository conventions** — retained as compact bullets where they are one sentence, stable, broadly applicable, and cheaper to keep than to navigate to:
- Check `libs/atlas-constants/` before defining a new domain type, alias, or numeric constant.
- Prefer straightforward moves over re-exported type aliases; do not cross service boundaries into another layer's internals.
- Use the project's Builder pattern for test setup; no `*_testhelpers.go`.
- Write design/plan documents to file in full — no interactive per-section approval.
- Use repo-relative paths or placeholders in committed files; never literal home/absolute paths.
- Preserve existing line endings when editing.
- Do not sweep the filesystem to locate a Go dependency — ask the toolchain (`go list -m -f '{{.Dir}}' <module>`).
- Do not spend inference turns polling or waiting for a process; launch it with a bound and do something else.
- Locate tracking docs with Glob/Grep rather than assuming a path.
- Substantive content must be sent as its own message before an `AskUserQuestion`.
- Do not proactively pitch paid features.

### FR-4 — Content relocated out of `CLAUDE.md`

The following must be moved to the owner documents identified in FR-2, and must not remain duplicated in root context.

**To `docs/verification.md`:** CI equivalence detail; per-module build/vet/test/bake behavior; why the docker bake step is not optional and the `go.work` `COPY libs/...` failure example; the `--base <last-gated-commit>` incremental optimization and its 86-module fan-out arithmetic; background-execution guidance specific to verification; known CI drift; escape hatches and per-guard invariants (much of this already lives there — the task consolidates rather than duplicates).

**To `docs/superpowers-integration.md`:** fuzzy task-identifier resolution; `tools/task-resolve.sh` output format and `--list` behavior; `tools/task-numbers.sh next` usage; why globbing `.worktrees/*/docs/tasks/task-*` returns tasks × worktrees duplicates; the Superpowers default artifact paths and the `docs/tasks/task-NNN-slug/` override mechanics; the reviewer agent inventory (`plan-adherence-reviewer`, `backend-guidelines-reviewer`, `frontend-guidelines-reviewer`), their file-type dispatch rules, and `audit.md` output details; detailed cross-service defect examples (producer/consumer compartment, saga action without a step handler, tests pinning old silent-drop behavior).

**To `docs/packets/PROCESS.md`** (only if not already present): the `/implement-packet` + `packet-implementer` mapping; `/bringup-version`; `family-auditor` then `dispatcher-family-implementer`; `/verify-packet` + `packet-verifier` as the universal leaf step; packet × version matrix cell mechanics.

**To `docs/reverse-engineering.md` (new):** `func_query` with `name_regex` as the documented lookup method; the fact that `select_instance(port)` and port-based selection are dead; resolving the session from `idb_list` by binary name and passing it as the `database` parameter.

**To `docs/git-workflow.md` (new):** stray-`main`-commit recovery procedure; the exact `env -u GH_TOKEN -u GITHUB_TOKEN gh …` invocation and the reason `~/.config/atlas/gh.env` must not be sourced; the conflict-versus-build-trigger explanation and the `atlas-pr-<N>` ephemeral rollout behavior.

**To `docs/agent-dispatch.md` (new):** the ~7× Opus-versus-Sonnet cost comparison; the pin-follows-the-job rule and its full table reference; the 120 tool-call implementer budget, its warning threshold, the CAP+5 denial, the commit-and-report exemption, and the files implementing the contract; the verifier context-size comparison; the fork-dispatch guard; the controller ~250k backstop; token arithmetic for context handoff; the ~60k floor and ~40-tool-call heuristic; `tools/task-brief.sh`; the SDD ledger (`.superpowers/sdd/<plan>/progress.md`) and `task-N-report.md` artifacts; references to `/execute-task` Steps 4a/4d/4e.

**To `docs/tooling-conventions.md` (new):** the `find /` versus `go list` timing measurements and the task-227 anecdote; the module-cache case-escaping tell; `go doc <pkg>` and `go list -m all`; the full cost explanation for repeated `sleep` / `ps aux | grep` polling; guidance that per-tool hang-mode fallbacks belong in that tool's agent doc; POSIX-shell preference and per-file Edit/Write over shell patch loops.

**To the chosen runtime-debugging owner:** `mcp__kubernetes__pods_log` invocation specifics and the `atlas-character-factory` / `atlas-world` examples.

### FR-5 — Historical and enforcement material

- Every explanation of a past incident, prior agent failure, measurement, cost comparison, or hook origin must be evaluated against the test: *does this change current behavior?* If it does not, it relocates to the owner document, or is dropped if the destination already captures it. `CLAUDE.md` describes the current operating model, not an incident log.
- Where a rule is mechanically enforced by `.claude/hooks/*`, the **policy** stays in `CLAUDE.md` and the **enforcement implementation** (hook filename, threshold, exemption, denial semantics) relocates. Per the accepted decision, root context **may carry a brief `(enforced)` marker** on such rules so an agent is not surprised by a denial — but must not name the hook file or restate its thresholds.
- The existence of a hook never removes the need for the behavioral rule to be stated.

### FR-6 — Breadcrumb integrity

For every relocated item:

1. Its authoritative destination is identified.
2. An existing document is preferred over a new one.
3. The destination is verified to actually contain the required information — a breadcrumb must not lead to a document that does not answer the question. Where the destination is thin, the task adds the content rather than assuming it is there.
4. There is a clear path from `CLAUDE.md` to the destination.
5. Intermediate documents provide onward breadcrumbs where a further hop is needed.
6. Conflicting or duplicated versions of the same procedure are eliminated.

Every breadcrumb must be **trigger-labeled** — it states the situation that should cause an agent to follow it:

> **Packet / protocol work:** start at `docs/packets/PROCESS.md`.

not:

> More information: `docs/packets/PROCESS.md`.

### FR-7 — Bidirectional linking

Where a `.claude/commands/*` or `.claude/agents/*` file currently holds material that becomes owned by a `docs/` document (notably `.claude/commands/execute-task.md` Step 4a's model table and Steps 4d/4e's handoff mechanics), the command file must **link to the owner document** rather than restate the content. Per the accepted decision, `docs/` owns the content and command files reference it; `CLAUDE.md` links to `docs/`, never directly to a command file for policy content.

### FR-8 — Scanability

The final `CLAUDE.md` must use short sections, concise bullets, bold trigger and invariant phrases, direct links, and small decision tables. It must avoid long explanatory paragraphs, narrative incident history, deeply nested procedural steps, command tutorials, and repeated rationale.

An agent scanning the file must be able to answer, without linear reading:

- What must I never do?
- Which workflow applies to this task?
- What evidence standard applies?
- Which worktree and branch should I be in?
- Which agent and model should handle this?
- When should I hand work off?
- Where is the detailed procedure for this kind of work?

## 5. API Surface

Not applicable. This task changes documentation only; no service endpoints, request/response shapes, or error cases are affected.

## 6. Data Model

Not applicable. No entities, fields, relationships, constraints, or migrations.

## 7. Service Impact

No Atlas service is affected. No Go module, TypeScript source, test, build file, or CI workflow changes.

Files expected to change:

| Path | Change |
|---|---|
| `CLAUDE.md` | Rewritten |
| `docs/verification.md` | Extended / consolidated |
| `docs/superpowers-integration.md` | Extended |
| `docs/packets/PROCESS.md` | Extended only if a breadcrumb would otherwise dangle |
| `docs/observability.md` *or* a `docs/runbooks/` entry | Extended with the runtime-debugging default |
| `docs/git-workflow.md` | New |
| `docs/reverse-engineering.md` | New |
| `docs/tooling-conventions.md` | New |
| `docs/agent-dispatch.md` | New |
| `.claude/commands/execute-task.md` | Duplicated policy replaced with links |
| `docs/tasks/task-233-claude-md-refactor/inventory.md` | New — FR-1 artifact |
| `docs/tasks/task-233-claude-md-refactor/ownership.md` | New — FR-2 artifact |

The new-document budget is capped at the four listed above. If the design phase concludes a fifth is genuinely required, that is an explicit decision to surface, not a default.

## 8. Non-Functional Requirements

- **Context economy.** The purpose of the change is to reduce what every session pays for unconditionally. Size reduction is measured and reported, but is an outcome; no rule is weakened to achieve it.
- **No rule weakening.** When uncertain whether an item should remain direct, bias toward retaining it if violating it could cause: edits in the wrong tree, commits to the wrong branch, fabricated technical findings, incorrect-version reverse engineering, false verification claims, skipped review, destructive Git behavior, or runaway agent/context cost.
- **No dead links.** Every path referenced in `CLAUDE.md` must resolve to an existing file in the repository.
- **No parallel sources of truth.** After the change, no procedure is described normatively in two places.
- **Path hygiene.** No literal home or absolute paths in any committed file (`.claude/hooks/block-home-paths-in-docs.sh` enforces this).
- **Line-ending preservation** on all edited files.
- **Multi-tenancy / security / observability:** not applicable — no runtime behavior changes.

## 9. Open Questions

- **Runtime-debugging owner.** `docs/observability.md` exists (access paths, Loki/Grafana labels) and `docs/runbooks/` holds three operational runbooks, but neither currently owns "diagnose a wedged deploy." The design phase must pick one — extend `docs/observability.md`, or add `docs/runbooks/debugging-deployments.md`. Choosing the latter would exceed the four-new-document budget and therefore requires an explicit decision.
- **`docs/packets/PROCESS.md` sufficiency.** The breadcrumb can only be reduced to a pure router if `PROCESS.md` already names the entry points and agents currently listed in `CLAUDE.md`. Its outline (task type → entry point → playbook, version set, baseline status, CI gates, matrix cell states) suggests it does, but this must be confirmed against the file before anything is deleted from root.
- **`docs/agent-dispatch.md` versus `docs/superpowers-integration.md`.** The latter already carries a "Phase 4 context budget" section. The design phase must decide whether dispatch and handoff form a genuinely separate document or an expanded section of the existing one. The PRD assumes a separate document; collapsing to one is acceptable if it reads better and keeps the ownership map unambiguous.
- **Always-resident project facts.** `CLAUDE.md` currently contains no game-domain facts (those live in project memory), but the *Project Overview* and *Code Patterns* sections carry architecture that borders on it. Whether the constants-library rule belongs in root or in a backend-conventions doc is a judgment call for the design phase; the PRD retains it in root as a one-line bullet.
- **Pre-existing task-number collision.** Task number 231 is currently claimed by both `task-231-generalized-events-service` and `task-231-prepush-backup`. Unrelated to this task and not blocking it (233 is free), but it remains unresolved and the collision detector will keep firing.

## 10. Acceptance Criteria

- [ ] `inventory.md` exists and accounts for **every** rule currently in `CLAUDE.md`, each with a class (A–E), a disposition, and a destination.
- [ ] `ownership.md` exists and names exactly one authoritative owner for each of the ten concerns in FR-2.
- [ ] `CLAUDE.md` contains primarily invariants, operating defaults, and trigger-labeled routers.
- [ ] Every item listed in FR-3 is still present in `CLAUDE.md` with undiminished force.
- [ ] Every item listed in FR-4 is present at its destination document and absent from `CLAUDE.md`.
- [ ] The four new documents exist and are non-trivial — each actually contains the material relocated to it, not a stub.
- [ ] `docs/reverse-engineering.md` is narrowly scoped to session/API mechanics and version confirmation, not a general RE tutorial.
- [ ] Every path referenced in `CLAUDE.md` resolves to an existing file (verified, not assumed).
- [ ] Every breadcrumb in `CLAUDE.md` names its trigger condition, not just its destination.
- [ ] `.claude/commands/execute-task.md` links to `docs/` for model-selection and handoff policy rather than restating it.
- [ ] No procedure is described normatively in two places.
- [ ] No historical incident narrative, measurement, or cost comparison remains in `CLAUDE.md`.
- [ ] No hook filename or enforcement threshold remains in `CLAUDE.md`; `(enforced)` markers only.
- [ ] `CLAUDE.md` answers all seven scanability questions in FR-8 without linear reading.
- [ ] No service code, test, build file, or CI workflow changed.
- [ ] No literal home/absolute paths in any committed file.
- [ ] Before/after line and approximate token counts for `CLAUDE.md` are reported in the PR description, framed as an outcome measure.

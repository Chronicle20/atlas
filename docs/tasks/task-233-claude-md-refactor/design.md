# Design — CLAUDE.md Refactor: Progressive Disclosure for Always-Loaded Context

Task: `task-233-claude-md-refactor`
PRD: [`prd.md`](prd.md)
Status: Design
Created: 2026-08-16

---

## 1. Problem framing

`CLAUDE.md` is 220 lines / 19,543 bytes (~5k tokens) loaded unconditionally into
every session in this repository, before the agent knows what it has been asked
to do. The content is not wrong; it is uniformly *eager*. A session that never
touches a packet still pays for the packet entry-point table. A session that
never dispatches a subagent still pays for the CAP+5 denial semantics of
`.claude/hooks/turn-budget-guard.sh`.

The refactor is therefore not an editing task. It is an **information-architecture
task with an anti-regression obligation**: relocate detail behind trigger-labeled
breadcrumbs while proving that no rule lost force. Those two halves have opposite
failure modes — compression drops rules, and fear of dropping rules produces no
compression — so the design's central concern is a mechanism that lets us be
aggressive about relocation *because* the round-trip is auditable.

Four concerns described in `CLAUDE.md` (Git workflow, IDA/RE, shell & tooling
conventions, agent dispatch & context handoff) have **no owner document at all**.
That absence is the actual reason those sections cannot currently be trimmed —
there is nowhere for the detail to go. Creating the owners is a prerequisite, not
a side effect.

---

## 2. Approaches considered

### A — Pure router (rejected)

Reduce `CLAUDE.md` to ~40 lines of trigger → destination links; every rule,
including invariants, lives in an owner document.

Maximal context saving, and clean single-ownership. Rejected because it inverts
the cost model that motivates the task. The branch-safety rule, the no-invention
rule, and the "flagless verify.sh only" rule govern decisions the agent makes
*before it knows it needs them* — the exact condition under which a link is
worthless. A router-only file would trade ~3k tokens of context for a class of
unrecoverable errors (commits stranded on `main`, fabricated findings, false
"verified" claims). PRD §8 names this explicitly as the no-weakening bias.

### B — Invariants + operating defaults + trigger-labeled routers (recommended)

Three-tier file. Tier 1: invariants stated in full, directly. Tier 2: operating
defaults compressed to their governing sentence, mechanics relocated. Tier 3:
trigger-labeled routers to owner documents. Rationale, measurements, incident
history, and hook implementation detail leave entirely.

This is the PRD's model, and it matches the classification test in FR-1: an item
stays direct when the agent cannot reliably know it needs the information before
making the decision the information governs. Everything else is disclosable.

Recommended.

### C — Split across two always-loaded files (rejected)

`CLAUDE.md` plus a second file pulled in by `@import`.

Rejected on a factual ground: an `@import`ed file is loaded unconditionally
alongside its importer, so the split saves nothing. It would add a second place
to look while leaving the token cost identical. (The user-global `~/.claude/CLAUDE.md`
already demonstrates the mechanic with `@RTK.md`.) The only way to make context
conditional is a link an agent chooses to follow — which is approach B.

---

## 3. Architecture of the refactored `CLAUDE.md`

### 3.1 Tier model

| Tier | Contains | Test for membership |
|---|---|---|
| **1 — Invariants** | Rules whose violation produces unsafe, incorrect, or hard-to-recover work | Would an agent that has *not* read the owner doc make an unrecoverable mistake? |
| **2 — Operating defaults** | Broadly applicable behavior compressed to one governing sentence | Does it apply across most sessions regardless of task type? |
| **3 — Routers** | Trigger condition → owner document | Is the information only actionable after entering a specific workflow? |

Rationale, measurements, incident narrative, and hook implementation detail are
**tier 0 — not in the file at all**. They relocate, or drop when the destination
already captures them.

### 3.2 Section skeleton

The file is organized so the seven FR-8 scanability questions each map to exactly
one section, in the order an agent needs them:

```
# Atlas
  (1-line orientation: Go microservices monorepo; TS only in atlas-ui)

## Never do this                       → "What must I never do?"        [T1]
## Evidence & grounding                → "What evidence standard?"      [T1]
## Where you work — branch & worktree  → "Which worktree / branch?"     [T1]
## Done means verified                 → completion gate + review gate  [T1]
## Development workflow                → "Which workflow applies?"      [T1/T3]
## Dispatching agents                  → "Which agent and model?"       [T2]
## Handing off context                 → "When do I hand off?"          [T2]
## Repository conventions              → compact one-line bullets       [T2]
## Where the procedures live           → "Where is the detail?"         [T3]
```

Two structural choices worth naming:

**"Never do this" is a consolidated lead section, not a scatter.** Today the
hard prohibitions are spread across *Git Operations*, *Worktrees & Subagents*,
*Grounding*, and *Build & Verification* — an agent must read four sections to
learn what is forbidden. Consolidating them into a short opening list and then
restating each in its topical section would duplicate; instead the lead section
**is** the statement, and the topical sections carry only the positive procedure.
This is the one place the design deliberately reorders rather than relocates.

**"Where the procedures live" is a table, not prose.** One row per concern:
trigger condition, owner document, one-clause description of what is there. It
doubles as the reader-facing form of the FR-2 ownership map, satisfying the PRD's
"summarized in whichever root section is most natural."

### 3.3 Breadcrumb form

Every router names its trigger first, its destination second:

> **Packet or protocol work** — start at [`docs/packets/PROCESS.md`](docs/packets/PROCESS.md): version set, entry point per task type, CI gates.

Not `More information: docs/packets/PROCESS.md`. The trigger is the load-bearing
half; the agent is scanning for a situation match, not for a list of documents.

### 3.4 `(enforced)` markers

Per PRD FR-5, a rule that a hook mechanically enforces carries a bare `(enforced)`
suffix — no filename, no threshold, no denial semantics. Purpose: an agent that
hits a denial recognizes it as policy rather than a tool malfunction. The
behavioral rule is always stated regardless; `(enforced)` never substitutes for it.

Applies to: no-home-paths-in-docs, the implementer tool-call budget, the
fork-dispatch preference, and the commit-boundary handoff prompt.

---

## 4. Ownership map (FR-2) and open-question resolutions

| Concern | Owner | Action |
|---|---|---|
| Verification gate | `docs/verification.md` | Extend / consolidate |
| Packet & protocol work | `docs/packets/PROCESS.md` | **No change** — verified sufficient (§4.1) |
| Task lifecycle, task resolution, code-review dispatch | `docs/superpowers-integration.md` | Extend; hand its Phase-4-budget table to `agent-dispatch.md` |
| Adding a service | `docs/adding-a-new-service.md` | No change |
| Git / branch / PR workflow | `docs/git-workflow.md` | **New** |
| Reverse engineering / IDA | `docs/reverse-engineering.md` | **New**, narrow |
| Shell, editing, tooling conventions | `docs/tooling-conventions.md` | **New** |
| Agent dispatch, model selection, context handoff | `docs/agent-dispatch.md` | **New** |
| Runtime / Kubernetes debugging | `docs/observability.md` | Extend (§4.2) |
| Go service patterns (constants, boundaries, builders) | `backend-dev-guidelines` skill + `backend-guidelines-reviewer` | Existing; root keeps one-line bullets only (§4.4) |

New-document count: **four**. Within the PRD budget; no fifth document is proposed.

### 4.1 OQ — `docs/packets/PROCESS.md` sufficiency: **confirmed sufficient**

Verified against the file. `PROCESS.md` §"Task type → entry point → canonical
playbook" already carries every entry point `CLAUDE.md` currently lists —
`/implement-packet` + `packet-implementer`, `/bringup-version`, `family-auditor`
then `dispatcher-family-implementer` — plus the shared leaf step
(`/verify-packet` + `packet-verifier`), the 10-version set, baseline status, and
the CI gate list. It also carries a maintenance row (`RE_AUDITING_A_COLUMN.md`)
that `CLAUDE.md` never mentioned.

**Consequence:** the root *Packet work* section collapses to a single router line
with no content added to `PROCESS.md`. This also resolves a pre-existing
duplication: `docs/superpowers-integration.md` §"Packet Work" restates the same
table. That section is reduced to a pointer at `PROCESS.md`, satisfying the
"no parallel sources of truth" NFR for a duplication the PRD did not enumerate.

### 4.2 OQ — runtime-debugging owner: **extend `docs/observability.md`**

Neither candidate owns "diagnose a wedged deploy" today.
`docs/observability.md` is the telemetry *pipeline* document (spans, spanmetrics,
dashboards, Loki/Prometheus/Grafana access, label conventions).
`docs/runbooks/ephemeral-pr-deployments.md` is PR-env lifecycle — its §9.3
"Inspecting a stuck env" and §9.4 "Recovery when teardown wedges" are scoped to
ephemeral environments, not to a service crash-looping in any environment.

Decision: add a short **"Diagnosing a runtime failure"** section to
`docs/observability.md`. Justification: the relocated material is *how to reach
logs* (`mcp__kubernetes__pods_log`, which services to read first, the Loki
`service_name` selector), which is exactly the observability document's existing
subject — §"Filtering by environment" already documents the Loki/Prometheus/Grafana
access paths that this section builds on. A new runbook would exceed the
document budget to hold four sentences, and would split log access across two
files.

The section cross-links to the ephemeral runbook for env-lifecycle wedges, so
the two remain distinguishable.

The root-level rule itself is short, broadly applicable, and governs the *first*
move in an investigation — before the agent knows to open a doc. It therefore
stays in `CLAUDE.md` as a one-line Tier-2 default; only the tool specifics and
examples relocate.

### 4.3 OQ — `agent-dispatch.md` vs `superpowers-integration.md`: **separate document**

`docs/superpowers-integration.md` is scoped to the Superpowers workflow — which
command, agent, or skill to reach for within the four-phase lifecycle. Model
pinning, the tool-call budget, the fork preference, and the context-handoff
decision apply to **every** dispatch in **every** session, including ad-hoc ones
that never enter the four-phase workflow. Folding them into a workflow document
would make the ownership map ambiguous in exactly the direction the task is
trying to fix ("is this a Phase 4 rule or a global rule?").

Decision: `docs/agent-dispatch.md` is a separate owner. As part of the change,
`superpowers-integration.md`'s "Phase 4 context budget" section is reduced to a
pointer, so the two do not compete.

Boundary between the two, stated in both files:

- `superpowers-integration.md` — *which* command/agent/skill for a situation.
- `agent-dispatch.md` — *how* to dispatch any agent: model, budget, isolation, handoff.

### 4.4 OQ — constants library and code-pattern rules: **stay in root as bullets**

`libs/atlas-constants/` first-check, the no-re-exported-aliases preference, the
service-boundary rule, and the Builder-pattern test rule each fit in one line,
apply to every Go edit, and are cheaper to keep loaded than to route to. They
stay as Tier-2 bullets under *Repository conventions*. The detail (the package
index, DOM-21 enforcement) already lives in `libs/atlas-constants/README.md` and
the `backend-dev-guidelines` skill; root drops the enumeration of what the
library covers and the DOM-21 identifier, keeping the rule and the path.

### 4.5 OQ — task-231 number collision: **out of scope, unchanged**

Two task IDs claim number 231 (`task-231-generalized-events-service`,
`task-231-prepush-backup`). Unrelated to this task, does not block it, and
resolving it would mean renaming another task's branch and worktree from inside
this one. Left as-is; the collision detector continues to fire.

---

## 5. The anti-regression mechanism

This is the part of the design that makes aggressive relocation safe.

### 5.1 `inventory.md` is a ledger, not a report

The FR-1 artifact is produced **before** any file is edited, and is the working
document for the edit — not a write-up produced afterward to describe what
happened. One row per atomic rule in today's `CLAUDE.md`:

| Column | Content |
|---|---|
| `#` | Stable id, `R-NNN`, assigned in source order |
| Source | Current `CLAUDE.md` section |
| Rule | The rule in its own words, one sentence |
| Class | A / B / C / D / E per FR-1 |
| Disposition | `keep-verbatim` · `keep-compressed` · `relocate` · `drop-captured` |
| Destination | Owner document + section anchor, or `CLAUDE.md §<section>` |

`drop-captured` is the only disposition that permits removal without relocation,
and it requires the destination text to already exist — it is verified by
reading, not asserted.

### 5.2 The round-trip check

Two directions, both required, both mechanical enough to be checkable by reading:

1. **Forward (nothing lost):** every `R-NNN` row resolves to text that exists —
   in the new `CLAUDE.md` for `keep-*`, in the named destination section for
   `relocate` / `drop-captured`.
2. **Reverse (nothing invented):** every rule in the new `CLAUDE.md` traces back
   to an `R-NNN`. A rule with no source id is either a genuine policy change —
   forbidden by PRD §2 non-goals — or an accidental restatement.

The reverse direction is the one that is easy to skip and the one that catches
policy drift. It is a required step, not a nicety.

### 5.3 Ordering constraint

The edit sequence is forced by the round-trip:

1. Build `inventory.md` and `ownership.md` from the current file.
2. Create/extend all destination documents (four new, four extended) so every
   destination is real.
3. Only then rewrite `CLAUDE.md` against the inventory.
4. Update `.claude/commands/execute-task.md` to link rather than restate.
5. Run both round-trip directions; resolve dead links by reading, not assuming.

Rewriting `CLAUDE.md` first would produce breadcrumbs pointing at documents that
do not yet contain the answer — the precise failure FR-6.3 forbids.

---

## 6. Relocation plan by destination

Detail lists come from PRD FR-4; this section records the *shape* each
destination takes, which is what the plan phase needs.

### `docs/git-workflow.md` (new)

Sections: Branch safety (why a `main` commit is stranded; recovery procedure) ·
Pushing and history rewrites · Build triggering and the conflict exception
(`atlas-pr-<N>` ephemeral rollout) · `gh` authentication (the
`env -u GH_TOKEN -u GITHUB_TOKEN` invocation and why `~/.config/atlas/gh.env`
must not be sourced).

Root keeps: never commit/push to `main` (Tier 1), push after a history rewrite
(Tier 1), do not merge `main` to trigger a build (Tier 2), and the router.

### `docs/reverse-engineering.md` (new, narrow)

Deliberately **not** an RE tutorial (PRD acceptance criterion). Scope: session
resolution via `idb_list` by binary name and the `database` parameter;
`select_instance(port)` is dead; `func_query` + `name_regex` as the documented
lookup; confirm the IDB version matches the target version. Ends with a pointer
to `docs/packets/PROCESS.md` for packet-derivation work, which is the dominant
consumer.

Root keeps: confirm the version before investigating (Tier 1, already part of
Evidence & grounding) and the router.

### `docs/tooling-conventions.md` (new)

Sections: Locating Go module source (`go list -m -f '{{.Dir}}'`, `go doc`,
`go list -m all`; why `find /` is wrong, with the task-227 measurements) ·
Waiting on processes (bounded background launch; why polling is expensive; per-tool
hang fallbacks belong in that tool's agent doc) · Shell and editing conventions
(POSIX preference, per-file Edit/Write over patch loops, line-ending preservation).

Root keeps the three governing sentences as Tier-2 bullets plus the router.

### `docs/agent-dispatch.md` (new)

Sections: Model selection (the full job → model table, moved here from
`execute-task.md` Step 4a; the ~7× cost comparison; the pin-follows-the-job rule;
the `model: opus` opt-in) · The implementer budget (120 calls, warning threshold,
CAP+5 denial, commit-and-report exemption, which agent files carry the contract,
where the number is changed) · Verification split (why implementers do not run
the repo gate) · Fork vs fresh context · Context handoff mechanics (the floor,
the controller backstop, `tools/task-brief.sh`, the SDD ledger and
`task-N-report.md`, pointers to `/execute-task` Steps 4c-4e).

Root keeps: explicit `model` on every dispatch (Tier 1), the three-line
job→model rule, never Fable for background/review, prefer fresh agents over
forks, implementers verify module-locally only, and the handoff *decision
criterion* — dependency on conversation history vs repository state, write the
diagnosis before handing off, `/clear` is a user action.

### `docs/verification.md` (extend)

Already carries the iteration gate, the Go/docker layers, guards, lint, and CI
drift. Additions: an explicit statement that a flagged run is not a pass (today
the flag list is presented neutrally), and the background-execution guidance.
Most of FR-4's verification list is **already there** — those inventory rows take
`drop-captured`, verified by reading, and this is where the largest single block
of root text leaves without any new writing.

### `docs/superpowers-integration.md` (extend)

Additions: fuzzy task-identifier resolution and `tools/task-resolve.sh` output
format; `--list`; `tools/task-numbers.sh next`; why globbing
`.worktrees/*/docs/tasks/task-*` returns tasks × worktrees; the artifact-location
override mechanics; `audit.md` output detail; the cross-service defect examples
(producer/consumer compartment, saga action without a step handler, tests pinning
the old silent drop).
Reductions: §"Packet Work" → pointer (§4.1); §"Phase 4 context budget" → pointer
to `agent-dispatch.md` (§4.3).

### `docs/observability.md` (extend)

One new section per §4.2.

### `.claude/commands/execute-task.md` (edit)

Step 4a's model table and Steps 4d/4e's handoff arithmetic are replaced by links
to `docs/agent-dispatch.md`. The command retains its *procedural* steps — what to
do at a `PARTIAL`, what to confirm in the ledger — because those are command
mechanics, not policy. Direction of ownership is fixed by FR-7: `docs/` owns
policy, command files reference it, `CLAUDE.md` never links to a command file for
policy content.

---

## 7. Risks and mitigations

| Risk | Mitigation |
|---|---|
| A rule is silently softened while being compressed | Reverse round-trip (§5.2) — every new sentence traces to an `R-NNN`; the reviewer reads the two side by side |
| A breadcrumb points at a document that does not answer the question | Destinations written before `CLAUDE.md` is rewritten (§5.3); FR-6.3 verified by reading the destination, not by the link resolving |
| A new document is a stub | Each new doc must hold the full relocated text for its concern; an acceptance criterion, checked by reading |
| Root re-accumulates procedure later | `ownership.md` plus the root "Where the procedures live" table give a maintainer a one-hop answer to "which document owns this?" |
| Optimizing for line count | Size is reported as an outcome in the PR description, never as a target; the no-weakening bias in PRD §8 governs every close call |
| Home-path leakage in new docs | `.claude/hooks/block-home-paths-in-docs.sh` catches it on write; use repo-relative paths and placeholders throughout |

---

## 8. Verification for this task

`tools/verify.sh` is change-gated and this branch touches no Go module, so the
gate is near-empty by construction — a green run proves almost nothing here. It
is still run flaglessly before PR (it exercises the repo guards, including the
home-path guard on the new docs), but acceptance is by **human reading** against
PRD §10, as the PRD requires. Explicitly declined by the PRD: any automated
`CLAUDE.md` size cap, lint rule, or link checker.

Two checks are worth doing mechanically even without a lint rule, as steps in the
plan rather than committed tooling:

- Every path referenced in the new `CLAUDE.md` resolves to a file that exists.
- No `.claude/hooks/*` filename and no numeric enforcement threshold remains in
  `CLAUDE.md`.

Measurement to report in the PR description, framed as an outcome: before —
**220 lines / 19,543 bytes**; after — measured at completion.

---

## 9. Deliverables

| Path | Change |
|---|---|
| `docs/tasks/task-233-claude-md-refactor/inventory.md` | New — FR-1 ledger |
| `docs/tasks/task-233-claude-md-refactor/ownership.md` | New — FR-2 map |
| `docs/git-workflow.md` | New |
| `docs/reverse-engineering.md` | New |
| `docs/tooling-conventions.md` | New |
| `docs/agent-dispatch.md` | New |
| `docs/verification.md` | Extended |
| `docs/superpowers-integration.md` | Extended + two sections reduced to pointers |
| `docs/observability.md` | Extended — runtime-failure diagnosis |
| `.claude/commands/execute-task.md` | Policy replaced with links |
| `CLAUDE.md` | Rewritten |

`docs/packets/PROCESS.md` and `docs/adding-a-new-service.md` are unchanged.
No service code, test, build file, or CI workflow is touched.

# task-237 — Acting on the task-227 / task-232 agentic cost audits

**Not an audit.** The two audits are the evidence; this task is the state
transition they argue for. Spec of record: `task-227-agentic-cost-audit.md` and
`task-232-agentic-cost-audit.md` (both reproduced under `audits/` in this folder).

---

## Reconciliation — what already landed

Inspected before designing anything. Do not re-implement these.

| Audit recommendation | State | Evidence |
|---|---|---|
| Per-agent `tools:` restriction | **Landed** (task-234 Task 2) | all 12 `.claude/agents/*.md` carry a `tools:` line |
| Explicit `model` on every dispatch | **Landed** | `docs/agent-dispatch.md` §Model selection + job→model table |
| Shell glob quoting (`--include='*.go'`) | **Landed** (task-234 Task 3) | `docs/tooling-conventions.md` §Shell and editing conventions |
| "Never poll a process" prose rule | **Landed, unenforced** | `docs/tooling-conventions.md` §Waiting on processes; CLAUDE.md bullet |
| Controller context ceiling (~150k / 4 tasks) | **Landed** (task-234 Task 1) | `/execute-task` Step 4e, `docs/agent-dispatch.md` §Context handoff |
| Codemod-vs-agents break-even rule | **Landed** (task-234 Task 5a) | `docs/codemod-vs-agents.md` |
| Fork-dispatch guard | **Landed** | `.claude/hooks/fork-dispatch-guard.sh` |
| Implementer 120-call PARTIAL budget + guard | **Landed** | `turn-budget.sh` / `turn-budget-guard.sh` |
| Task resolution helper | **Landed** | `tools/task-resolve.sh` (+ `_test.sh`) |
| Per-plan-task brief extractor | **Landed** | `tools/task-brief.sh` — already the "extract one plan task" accessor |
| Verification split (implementer never gates) | **Landed** | `/execute-task` Step 4c, `atlas-verifier` |
| Review agent right-sizing | **Landed as documentation** | `/execute-task` Step 4c bullet |

**Therefore skipped:** task-227 T1(b) partially, task-232 opportunities #6 and #7
(as standalone), and every §8 "leave alone" item in both audits.

**What is genuinely missing** is the set below. Each workstream extends an
existing mechanism rather than adding a parallel one.

---

## Task 1 — Reviewer return protocol (WS1)

**Cause:** reviewer returns are 3.4–8 KB (227) / median 4,904 B with 0/84 durable
artifacts (232), while implementer (1,145–2,595 B) and verifier (478–1,546 B)
returns are healthy. Detail duplicates an artifact the reviewer just wrote, or —
for bare `general-purpose` per-unit review — is not written anywhere.

**Changes:**

1. New `docs/review-protocol.md` — single owner of the return block, the verdict
   vocabulary, the artifact requirement, and the controller's read rule.
2. Append a `## Return to the controller` section to
   `.claude/agents/backend-guidelines-reviewer.md`,
   `frontend-guidelines-reviewer.md`, `plan-adherence-reviewer.md`,
   `packet-completeness-critic.md`, `family-auditor.md` — each pointing at the
   protocol and naming its own artifact path.
   **Accepted exemption (code review, post-implementation):**
   `packet-completeness-critic.md` and `family-auditor.md` were left untouched.
   Both already return a one-line verdict plus counts and point the caller at
   their own artifact, so the append would have been churn against a contract
   they already satisfy. The other three, plus the new `atlas-reviewer.md`,
   carry the section.
3. New `.claude/agents/atlas-reviewer.md` — the named contract for per-unit /
   ad-hoc code review that currently rides bare `general-purpose` (84 of 93
   dispatches in task-232).
4. `/execute-task` Step 4c review dispatch → `atlas-reviewer`; controller reads
   the artifact only when `verdict != APPROVED`.
5. `docs/superpowers-integration.md` §Code Review → link the protocol.

**Not done:** implementer and verifier return contracts are untouched.

## Task 2 — Slice-first artifact access (WS2)

**Cause:** 74 whole-file reads of two documents = 69.3 MB-turns (7.5% of all
carry); median >12 KB result lands at position 0.10 of its stream.

**Changes:**

1. New `tools/doc-slice.sh` — one reusable accessor with three modes:
   `--outline` (headings + line ranges + byte sizes), `--section <pattern>`
   (one heading's body), `--rows <pattern>` (matching table/list rows with the
   header row preserved). Works on any Markdown/text file including
   `tool-results/*.txt` offloads.
2. `tools/doc-slice_test.sh`.
3. New `docs/slice-first.md` — the principle, the escalation clause, and the
   `git diff --stat` → per-file-hunk rule for review diffs.
4. Reference it from `.claude/agents/atlas-implementer.md`,
   `atlas-reviewer.md`, `plan-adherence-reviewer.md`, the two guideline
   reviewers, and `docs/tooling-conventions.md`.

**Explicitly a default with an escalation path, never a prohibition.**

## Task 3 — Deterministic facts (WS3)

**Cause:** ~30 turns at 170–290k reverse-engineering `verify.sh`'s own selection
logic (227 §A1); 2,191 Bash calls / 2.07 MB on mechanical repository facts (232
§A.2).

**Changes:**

1. `tools/verify.sh --facts` — implemented by neutering `step()` so the script's
   real body runs its real selection logic and executes nothing, then printing
   the fact block instead of the summary. **Same code path by construction**, not
   a reimplementation.
2. New `tools/change-surfaces.sh` — path classification of a change set
   (`rest_surface`, `kafka_surface`, `db_surface`, `deploy_surface`,
   `packet_surface`, `new_service`, `backend_audit_families`, `frontend_review`).
   **Additive and fail-open**: on any uncertainty it widens the family list and
   emits `classification=uncertain`.
3. New `tools/task-facts.sh` — the composer. `task-resolve.sh` +
   `change-surfaces.sh` + `verify.sh --facts` (for `applicable_guards`) + a
   toolchain line, in one <1 KB block. No new resolution framework.
4. `_test.sh` for each new script; `verify.sh --facts` covered by
   `tools/verify-facts_test.sh` asserting `--facts` agrees with a real run's
   selection. **Shipped as `tools/verify_test.sh`** — the suite also carries the
   structural anti-drift invariant that a gate label can only originate inside
   `step()`, which is a `verify.sh` property rather than a `--facts` one, so it
   is named for the script under test.
5. Inject into briefs: `/execute-task` Step 4b prepends the block;
   `superpowers:requesting-code-review` roster section consumes
   `change-surfaces.sh`.

## Task 4 — Waiting / polling / micro-delegation (WS4)

**Cause:** 30 `Bash true` no-ops in one reviewer (≈3.6M, 36% of that agent); 20 +
11 polling calls in two main threads at 170–290k; six micro-delegations costing
4.32M for 4,669 output tokens.

**Changes:**

1. New `.claude/hooks/wait-loop-guard.sh` (PreToolUse, Bash) denying bare
   no-ops and poll loops, with a `POLL-JUSTIFIED:` escape mirroring
   `FORK-JUSTIFIED:`. Legitimate inspection (`ps -p`, `kubectl`, `docker ps`,
   `sleep` inside a file being written) must pass.
2. `.claude/hooks/wait-loop-guard_test.sh`.
3. Register in `.claude/settings.json`.
4. Inline-vs-delegate break-even paragraph in `docs/agent-dispatch.md`; the
   no-nested-fan-out rule into the reviewer agent definitions.

## Task 5 — Post-implementation fresh context (WS5)

**Cause:** post-PR phase = 12.7% of task-227 at 94% main thread, 3 subagents
across 4 sessions, peaks 328k/274k. The four-phase flow ends at `/execute-task`.

**Changes:**

1. New `docs/post-implementation.md` — Phase 5. The durable-artifact loop:
   reproduce inline → write `docs/tasks/<task>/bug-<slug>.md` → dispatch a fresh
   `atlas-implementer` against that file → `atlas-verifier`. Reproduction stays
   inline (operator in the loop); *fix implementation* delegates.
2. New `/fix-pr-bug` command mechanizing it.
3. CLAUDE.md router row + `docs/superpowers-integration.md` phase table row.
4. Generalize `docs/agent-dispatch.md` §Context handoff to name the post-PR case.

**No arbitrary clearing, no user `/clear` requirement** — delegation and
artifacts are the mechanism.

## Task 6 — Measurement (WS6)

**Cause:** both audits needed hundreds of lines of ad-hoc transcript parsing.

**Changes:**

1. New `tools/agent-ledger.sh` — `append` / `summary` over
   `docs/tasks/<task>/agent-ledger.tsv`. Fields: `unit, agent_type, model,
   turns, tool_calls, tool_result_bytes, return_bytes, status, commit`, plus
   `verdict` and `caused_fix` for reviewer rows and a `HANDOFF` row type
   carrying context size. **Unknown fields are written `-`, never fabricated.**
2. `tools/agent-ledger_test.sh`.
3. `/execute-task` reconcile step appends one row per agent; the review protocol
   requires the reviewer's own row.

## Task 7 — Small findings

Only where they land naturally:

- `docs/reverse-engineering.md`: one `select:` line for the standard IDA tool set
  + "prefer `func_query`/bounded `insn_query` over full `decompile`".
- `docs/observability.md`: "never list a whole namespace when you know the
  service name".
- `docs/tooling-conventions.md`: rtk must not swallow `tools/*.sh` stdout
  (deterministic output eaten by a wrapper costs a whole turn); toolchain
  availability comes from `task-facts.sh`, not repeated probes.
- Commit `.envrc` so the `nvm.sh` prefix leaves the command strings.

Already fixed, so **not** re-done: glob quoting.

---

## Validation

| # | Claim | How |
|---|---|---|
| 1 | Clean reviewer return is compact, reasoning durable | worked example in `docs/review-protocol.md` |
| 2 | Failed review keeps blocking findings immediate | same, `CHANGES_REQUIRED` example |
| 3 | Plan task retrievable without loading `plan.md` | `tools/task-brief.sh` (existing) + `doc-slice.sh --section`, covered by `doc-slice_test.sh` |
| 4 | Task/change facts agree with repo state | `task-facts_test.sh`, `change-surfaces_test.sh` against a synthetic change set |
| 5 | `--facts` and real verification share selection logic | `verify-facts_test.sh` — asserts the module/guard/gate sets match a real `--quick` run's own reporting on the same base |
| 6 | Uncertain classification fails open | `change-surfaces_test.sh` novel-layout case → all families + `classification=uncertain` |
| 7 | Polling guard permits legitimate debugging | `wait-loop-guard_test.sh` allow-cases |
| 8 | Existing workflows still function | `tools/verify.sh` (flagless) at branch end |

## Out of scope

Everything both audits' §8 guards: implementer/verifier return contracts,
dispatch brief size, the design/plan whole-document loads, `progress.md`,
targeted read slices, the 120-call budget, the verification split, the
four-phase flow, and the authoritative reference documents' *content*.

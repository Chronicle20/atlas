# task-237 — Acting on the task-227 and task-232 agentic cost audits

**What this was.** Not a third audit. The two prior audits
(`audits/task-227-agentic-cost-audit.md`, `audits/task-232-agentic-cost-audit.md`)
are the evidence; this task reconciled them against current repository state and
implemented the smallest coherent set of changes that addresses the underlying
causes.

**Objective optimized:** minimum total model/context consumption required to make
the correct repository state transition — not token count in isolation. Every
change below either replaces expensive turns with a deterministic answer, moves
detail to a durable artifact so it is not re-billed on every subsequent turn, or
prevents a turn that produces no information. Nothing was removed that would
cause rediscovery later.

---

## 1. What was already implemented, and therefore skipped

Verified in the tree before designing anything.

| Audit recommendation | Where it already lives |
|---|---|
| Per-agent `tools:` restriction (task-234 Task 2) | all 12 `.claude/agents/*.md` carry `tools:` |
| Explicit `model` on every dispatch | `docs/agent-dispatch.md` §Model selection, job→model table |
| Shell glob quoting, `--include='*.go'` (227 §small, 232 #6) | `docs/tooling-conventions.md` §Shell and editing conventions — **task-234 Task 3 landed; not redone** |
| "Never poll a process" as prose | `docs/tooling-conventions.md` §Waiting on processes, CLAUDE.md bullet — landed, but **unenforced**; that gap is Workstream 4 |
| Controller context ceiling (~150k / 4 tasks) (227 T4 partial) | `/execute-task` Step 4e + `docs/agent-dispatch.md` §Context handoff |
| Codemod-vs-agents break-even | `docs/codemod-vs-agents.md` |
| Fork discipline, machine-checked | `.claude/hooks/fork-dispatch-guard.sh` |
| Implementer 120-call PARTIAL budget, machine-checked | `turn-budget.sh` / `turn-budget-guard.sh` |
| Fuzzy task resolution | `tools/task-resolve.sh` (+ `_test.sh`) |
| Per-plan-task extraction (232 Task A partial, 232 §C.3) | `tools/task-brief.sh` — **already the "one plan task without the whole plan" accessor**; wired into `plan-adherence-reviewer` rather than rebuilt |
| Verification split (implementer never gates) | `/execute-task` Step 4c, `atlas-verifier` |
| Review-agent right-sizing (task-234 Task 7) | `/execute-task` Step 4c |

Also **not touched**, per both audits' §8: implementer and verifier return
contracts, dispatch brief size, the `/plan-task` whole-document `prd.md` +
`design.md` loads, `progress.md`, targeted read slices, the 120-call budget, the
four-phase flow, and the *content* of every authoritative reference document.

---

## 2. What changed

### Workstream 1 — Compact reviewer protocol

**Finding addressed:** 227 §B2 / §9 #4 (reviewer returns 3.4–8 KB restating a
durable artifact; ~1.9 KB of one 3.4 KB return was unactionable PASS text, ×22
reviews) and 232 §B.1–B.4 / #2 (median review return 4,904 B; **0 of 84 wrote a
durable artifact**; only 26 of 84 verdict-first; 40.3 MB-turns of notification
carry).

| File | Change |
|---|---|
| `docs/review-protocol.md` | **New.** Single owner of the return block, verdict vocabulary, artifact requirement, controller read rule, and both worked examples (clean → 312 B against a measured 3,370 B; failed → both blocking findings dispatchable without opening the artifact). |
| `.claude/agents/atlas-reviewer.md` | **New.** The named contract for per-unit / ad-hoc review — the thing 84 of 93 review dispatches lacked by riding bare `general-purpose`. |
| `backend-guidelines-reviewer.md`, `frontend-guidelines-reviewer.md`, `plan-adherence-reviewer.md` | `## Return to the controller` appended, mapping each agent's existing status vocabulary onto the verdict and naming its artifact. |
| `.claude/commands/execute-task.md` | Step 4c: per-task review is `atlas-reviewer`; read the artifact only when the verdict is not `APPROVED`. |
| `docs/superpowers-integration.md`, `CLAUDE.md` | Roster entry, protocol link, router row. |

`APPROVED_WITH_FINDINGS` exists so a concern is never suppressed to look clean.
Blocking findings stay **enumerated with `file:line`**, not counted — the
controller must be able to dispatch a fix without opening the artifact.

`family-auditor` and `packet-completeness-critic` were **left alone**: both
already return a one-line verdict plus counts and explicitly point the caller at
their file. Changing them would have been churn.

### Workstream 2 — Slice-first artifact access

**Finding addressed:** 232 §C.4/§7/#1 — `service-wiring-recipe.md` (26 reads, 25
streams) and `query-scope-audit.md` (48 reads) cost **69.3 MB-turns, 7.5% of all
carry, from 74 of 11,102 tool calls**; median >12 KB result lands at position
**0.10** of its stream. Plus 232 §C.3 (`plan.md` read 13× inside one
`plan-adherence-reviewer`) and 227 §C4 (IDA/k8s results).

| File | Change |
|---|---|
| `tools/doc-slice.sh` + `_test.sh` | **New.** One reusable accessor: `--outline`, `--section`, `--rows` (Markdown table header preserved), `--grep --context`, `--lines`, `--max-bytes`. Works on Markdown, plain text, and offloaded `tool-results/*.txt`. Every slice prints the source path and size; truncation is always announced with an escalation hint. 39 assertions. |
| `docs/slice-first.md` | **New.** The principle, the measurement, and a per-situation table (plan task, recipe, audit matrix, review diff, offload, config, source). |
| `.claude/agents/atlas-implementer.md` | Contract 3 gains the slice-first bullet, explicitly exempting source files being edited. |
| `.claude/agents/plan-adherence-reviewer.md` | New "Reading the plan — slice, do not re-read" section pointing at `tools/task-brief.sh`. |
| `atlas-reviewer.md` | `git diff --stat` before hunks. |
| `.claude/commands/execute-task.md` | Step 4b: briefs name the *slice* ("Pattern C of…"), not the document. |
| `docs/reverse-engineering.md` | Prefer `func_query`/bounded `insn_query` over full `decompile`; one `select:` line for the whole IDA tool set. |
| `docs/observability.md` | Never list a whole namespace when the service name is known; the `pods_log` spill path is correct — do not `Read` a spilled result whole. |

Framed throughout as **a default with an explicit escalation path, never a
prohibition** — under-reading costs a fix round, which exceeds any read it saves.

### Workstream 3 — Deterministic task and change facts

**Finding addressed:** 227 §A1 (~30 turns at 170–290k reverse-engineering
`verify.sh`'s own selection logic ≈ 6.9M tokens, 24% of one session) and §A2
(13.6 KB `--stat` pair at reviewer turn 1–2 + ~12 applicability turns); 232 §A.2
(2,191 Bash calls / 2.07 MB on mechanical facts) and §A.4 (`task-resolve.sh`
used in 16 of 213 streams).

| File | Change |
|---|---|
| `tools/verify.sh` | **`--facts`.** Implemented by neutering `step()`, so the script's real body runs its real selection logic and executes nothing. **Same code path by construction** — not a reimplementation. Informational chatter moves to stderr; stdout is `key=value` only. |
| `tools/verify_test.sh` | **New**, 29 assertions: behavioural agreement (selected + skipped label sets identical to a real run over the same change set, across two flag sets and two synthetic probes) **and** the structural invariant that a gate label can only originate inside `step()`. |
| `tools/change-surfaces.sh` + `_test.sh` | **New.** Path/content classification → changed services and libs, REST/Kafka/DB/deploy/packet/tooling surfaces, `new_service`, `backend_audit_families`, `frontend_review`. **Additive and fail-open.** 49 assertions, including every fail-open path. |
| `tools/task-facts.sh` + `_test.sh` | **New.** The composer — `task-resolve.sh` + `change-surfaces.sh` + `verify.sh --facts` + a live toolchain line. **755 bytes** on a real task. No new resolution framework; exit codes 3/4 pass straight through. 19 assertions. |
| `.claude/commands/execute-task.md` | Step 1 offers `task-facts.sh`; Step 4b prepends the fact block to every brief; Step 4c reaches for `--facts` before investigating a surprising gate. |
| `docs/superpowers-integration.md` | Reviewer roster is read off `change-surfaces.sh`; the block is passed verbatim into each reviewer's brief as the minimum family set. |
| `docs/verification.md` | New `--facts` section with the output contract and the anti-drift argument. |
| `docs/tooling-conventions.md` | "Ask for a fact rather than deriving it" table; toolchain probing replaced by `task-facts.sh`. |

**Fail-open is enforced, not asserted.** An unresolvable base, a failed `git
diff`, or a Go file outside `services/`, `libs/`, `tools/` emits
`classification=uncertain` with all 17 families listed. `tools/change-surfaces_test.sh`
covers each of those paths, and every consumer doc states that a reviewer may
add a family but may never drop one.

### Workstream 4 — Waiting, polling, micro-delegation

**Finding addressed:** 227 §C1 (**30 `Bash true` no-ops in one reviewer** — 33%
of its tool calls, ≈3.6M tokens, ~36% of that agent), §C2 (20 + 11 + 22 polling
calls at 170–290k context), §B3 (six micro-delegations = 4.32M billed input for
4,669 output tokens; four returned <40 tokens).

| File | Change |
|---|---|
| `.claude/hooks/wait-loop-guard.sh` | **New** PreToolUse guard on Bash, modelled on `fork-dispatch-guard.sh`. Denies bare no-ops (`true`, `:`, `echo waiting`), sleep-driven polls, and broad `ps aux`/`ps -ef`/`pgrep` sweeps. Each refusal names the correct mechanism. |
| `.claude/hooks/wait-loop-guard_test.sh` | **New**, 33 cases — 15 deny, 18 allow. |
| `.claude/settings.json` | Registered on the `Bash` matcher. |
| `tools/verify.sh` | New gate: a `.claude/hooks/` change runs the hook suites (they were outside the `tools/`-changed gate). |
| `docs/agent-dispatch.md` | New §Inline vs delegate — the ~35–38k dispatch floor, the four-to-five-turn break-even, the six-children table, "never idle waiting on a child." |
| Three reviewer agent definitions, `docs/review-protocol.md` | "Do not fan out" — reviewers answer their own checklist. |
| `docs/tooling-conventions.md`, `CLAUDE.md` | Waiting rule extended to child agents; marked *(enforced)*. |

**Legitimate process debugging is explicitly preserved** and asserted in the
test: `ps -p <pid>`, `ps -o …`, `kill`/`pkill`, `kubectl`, `docker ps`, `top -b
-n1`, `journalctl`, `sleep` as file *content*, and `grep`ping for the word
"sleep" all pass. `POLL-JUSTIFIED:` is the escape hatch, mirroring
`FORK-JUSTIFIED:` — a considered wait costs one sentence.

*(The guard blocked a `sleep 45` from this very session mid-implementation; it
works.)*

### Workstream 5 — Fresh context after implementation

**Finding addressed:** 227 §3.1 / §7 / T4 — the post-PR phase was **120.5M
tokens, 12.7% of the task, at 94% main thread**, three subagents across four
sessions, peaks 328k and 274k. The four-phase flow ended at `/execute-task`, so
nothing told those sessions to delegate.

| File | Change |
|---|---|
| `docs/post-implementation.md` | **New** Phase 5: reproduce inline (operator in the loop) → diagnose to `docs/tasks/<task>/bug-<slug>.md` with a `## Fix` file inventory → dispatch a fresh `atlas-implementer` against that file → verify in a clean context → ledger. |
| `.claude/commands/fix-pr-bug.md` | **New** command mechanizing steps 2–5. |
| `docs/agent-dispatch.md` | §Context handoff explicitly extended past implementation. |
| `docs/superpowers-integration.md`, `CLAUDE.md` | Phase-5 row in the workflow table; router row. |

The bug-file habit **already existed** — that task folder holds four of them.
The delegation step after it did not; that is what was added. No new
context-clearing rule, and nothing requires the user to `/clear`.

### Workstream 6 — Lightweight measurement

**Finding addressed:** 232 §11 / Task E and 227 §11 — both audits were assembled
by hand-parsing transcripts; agent↔unit linkage, reviewer verdicts, "did a review
cause a fix", and declined-handoff cost were unrecoverable.

| File | Change |
|---|---|
| `tools/agent-ledger.sh` + `_test.sh` | **New.** `append` / `summary` / `path` over `docs/tasks/<task>/agent-ledger.tsv`. Fields: `unit, agent_type, model, turns, tool_calls, tool_result_bytes, return_bytes, status, commit`, plus `verdict`, `caused_fix`, `artifact`, and a `--kind handoff --context-tokens` row. `summary` reports per-type medians, verdict counts, fix causation, and handoff context. 27 assertions. |
| `/execute-task` Step 4f, `/fix-pr-bug` Step 6, `docs/review-protocol.md`, `docs/agent-dispatch.md` | One ledger line per agent at reconcile, batched with the dispatch that is already happening. |

**Unknown is `-`, never a guess.** Asserted in the test. No transcript parser was
built: 232 §11 explicitly warns against that, and a fabricated number would
poison the next audit worse than a gap.

### Small findings folded in

- **`grep -q` under `pipefail`** — see §3 below.
- **rtk swallowing `tools/*.sh` stdout** (227 §A0.1, a 280 B result costing a
  whole turn, twice) — stated as a requirement in `docs/tooling-conventions.md`.
  The wrapper itself is user-scope, outside this repo.
- **Toolchain probes** (232 §A.3, ~65 across 80 streams) — folded into
  `task-facts.sh`'s live `toolchain=` / `go_version=` lines, per 232 #7's
  "fold into #4, do not do separately."
- **MCP schema discovery** (227 §A3) — one `select:` line in
  `docs/reverse-engineering.md`.
- **The nvm command prefix** (227 §C6) — `tools/lib/node-env.sh`, sourced by
  `tools/lint.sh` `run_ui()` and `verify.sh` `ui_test_layer()`. No-op when node
  is already correct; locates nvm from `$NVM_DIR`/XDG/`$HOME` at runtime, so no
  home path is committed. The untracked `.envrc` was **not** committed — it is a
  personal `dotenv ~/.config/atlas/gh.env` line, not a Node shim.
- **Shell glob quoting** — already landed; not redone.

---

## 3. Defect found and fixed along the way

`tools/verify.sh`'s `touched()` used `printf | grep -q` under `set -o pipefail`.
`grep -q` exits on its first match, the still-writing `printf` takes SIGPIPE
(141), and `pipefail` reports that as the pipeline's status — **so a match reads
as "no match."** Reproduced directly:

```
grep -q:     NO MATCH (WRONG, rc=141)
grep w/o -q: MATCH
```

Consequence: past the 64 KB pipe buffer (~1,500 changed paths), every path-gated
guard — service registration, template guards, all four contract mirrors, the
tools suites, LB port drift, version coverage — silently **skips**, and the gate
still exits 0. It bites exactly on the large sweep branches that most need the
guards.

Fixed in `touched()` and the two other `grep -q` sites in `verify.sh`, and in
`change-surfaces.sh` where `$ADDED` is routinely ~1 MB (the bug was live there —
CONSTANTS, RESILIENCE and SEC never fired until it was fixed).
`tools/verify_test.sh` asserts no non-comment `grep -q` survives in the
change-detection path. This was not in either audit; it surfaced while building
`--facts`.

Two smaller ones, same class: a `--facts` assignment took its command
substitution's exit status and aborted under `set -e`; and `verify_test.sh`
initially recursed unboundedly, because `verify.sh` runs every changed
`tools/*_test.sh` and this test invokes `verify.sh`. Both fixed
(`ATLAS_VERIFY_TEST_INNER` breaks the recursion at depth one) and documented in
the file so the next editor does not reintroduce them.

---

## 4. Deliberately rejected

| Recommendation | Why not |
|---|---|
| 227 T5 / 232 A5: an authoritative `applicable_guards` mapping | `verify.sh` remains the authority. `task-facts.sh` reports what the gate *itself* selected via `--facts` rather than deriving a second mapping that could disagree. A wrong mapping silently skips a gate — the exact failure mode of §3. |
| 232 #7 standalone: a static toolchain line | A stale list is worse than a probe. Generated live inside `task-facts.sh`. |
| 227 #7: briefs carry symbol/line ranges, not just paths | Marked "Consider" in the audit; it needs a `task-brief.sh` output change plus a controller habit, and its evidence (0.85M per implementer) is an order of magnitude below Workstreams 1–5. Deferred rather than half-done. |
| 227 §11.2: fix or drop `<subagent_tokens>` | Harness-level, outside this repo. `agent-ledger.sh` routes around it by recording what the controller actually knows. |
| 227 §C4: an MCP field-projection flag | Tool-side, not repo-side. The documentation half (bound the query) landed. |
| 232 §11: per-turn context composition telemetry | Requires harness support; 227 §11.4 recommends a one-time measurement, not ongoing telemetry. Not built. |
| Committing `.envrc` | Personal config with a home path; CLAUDE.md forbids committed home paths. The actual problem — the nvm prefix — is solved by `tools/lib/node-env.sh`. |
| Trimming any reference document's content | Both audits' §8 are explicit: a thinner recipe would be re-derived by 25 agents at far greater cost. Access pattern changed; content untouched. |

---

## 5. Verification performed

| Suite | Result |
|---|---|
| `tools/verify_test.sh` | **29 assertions pass** — includes `--facts` vs real-run agreement on two flag sets and two synthetic probes, and the structural anti-drift invariants |
| `tools/change-surfaces_test.sh` | **49 pass** — hermetic fixture repo; every fail-open path covered |
| `tools/doc-slice_test.sh` | **39 pass** — all five modes, no-match exit 3, announced truncation, offload slicing |
| `tools/task-facts_test.sh` | **19 pass** — composition, exit-code pass-through, 1,200-byte budget |
| `tools/agent-ledger_test.sh` | **27 pass** — including "unmeasured is `-`, never 0" |
| `.claude/hooks/wait-loop-guard_test.sh` | **33 pass** — 15 deny, 18 allow (legitimate debugging preserved) |
| `shellcheck -x` on all new/changed scripts | No `error`-severity findings (the repo gate's threshold); the two style findings worth fixing were fixed |
| `tools/verify.sh` (flagless) | **PASS, exit 0** — 7 checks ran (shell tooling guard + all six suites), 13 skipped as unapplicable (no Go module, no `go.mod`, no `.go` file, no deploy or UI change). Verdict below. |

### The flagless gate

```
✓ shell tooling guard      ✓ agent-ledger_test.sh     ✓ change-surfaces_test.sh
✓ doc-slice_test.sh        ✓ task-facts_test.sh       ✓ verify_test.sh
✓ wait-loop-guard_test.sh
All checks passed.  GATE EXIT=0
```

The **first** flagless run FAILED and is worth recording, because the failure
mode was self-inflicted twice over. `shell tooling guard` went red on
`tools/verify_test.sh`: a comment line beginning `# shellcheck and lint, …` is
parsed by shellcheck as a malformed directive (SC1072/SC1073). Fixed in
`742dad384`, with a note in the file so it is not reintroduced.

And the run was initially *reported* as passing, because it was launched as
`tools/verify.sh > log 2>&1; echo "GATE EXIT=$?"` — the compound's exit status
is `echo`'s, so the harness saw 0 while the gate had exited 1. The re-run
propagates the gate's own status with `exit $rc`. **A background gate must
return the gate's exit code, not a wrapper's** — otherwise "the gate passed" is
a claim about the wrapper.

### Still outstanding

Per CLAUDE.md, **code review has not been run** — it is a separate gate from
verification, and it is required before a PR. This session was directed not to
dispatch agents, so the reviewer roster was not launched. Before opening a PR:

```sh
tools/change-surfaces.sh --base origin/main   # roster + families for this diff
```

then `superpowers:requesting-code-review`. On this branch the classifier reports
`go_changed=false`, `frontend_review=false`, `tooling_surface=true` — so the
review that matters is of the shell tooling and the workflow documents, not the
Go guideline checklists.

Mapping to the requested validation list:

1. *Clean reviewer → compact verdict-first return, reasoning durable* — worked
   example in `docs/review-protocol.md` (312 B vs a measured 3,370 B).
2. *Failed reviewer → blocking findings immediately actionable* —
   `CHANGES_REQUIRED` example; both findings dispatchable without opening the
   artifact.
3. *Plan task retrievable without loading the plan* — `tools/task-brief.sh`
   (existing) plus `doc-slice.sh --section`; `doc-slice_test.sh` asserts one
   section is under a third of the document and that `Task 1` does not match
   `Task 10`.
4. *Facts agree with repository state* — `task-facts_test.sh`,
   `change-surfaces_test.sh` against a synthetic repo with known answers.
5. *`--facts` and real verification share selection logic* — `verify_test.sh`,
   behavioural **and** structural.
6. *Uncertain classification fails open* — `change-surfaces_test.sh` novel-layout,
   bad-base, and `--all` cases each assert all 17 families plus
   `classification=uncertain`.
7. *Polling protections do not block legitimate debugging* —
   `wait-loop-guard_test.sh`, 18 allow-cases.
8. *Existing workflows still function* — `verify_test.sh` asserts real gate runs
   still select and execute their normal gates; `task-resolve_test.sh`,
   `task-numbers_test.sh`, `plan-lint_test.sh`, `gen-lb-ports_test.sh`,
   `check-version-coverage_test.sh` unchanged and still gated.

---

## 6. Baseline metrics for the next audit

Compare against these. Both audits' own numbers, restated so a follow-up does
not have to re-derive them.

### task-232 (sweep-shaped, 56 plan tasks, 17 sessions, 196 subagents)

| Metric | Baseline |
|---|---|
| Billed input | 1,259M (270M main / 988M subagent) |
| Turns | 1,725 main / 8,849 subagent |
| Tool calls | 11,102 (7,064 Bash) |
| Tool-result bytes | 20.36 MB |
| Total carry | 0.92 GB-turns ≈ 18% of billed input |
| Results >12 KB | 268 (26.8% of tool bytes), **median position 0.10** |
| Implementer orientation prefix | median **13** calls |
| Implementer return | median **1,145 B** (leave alone) |
| Review return | median **4,904 B**, p90 8,162, max 14,459 |
| Reviews writing a durable artifact | **0 of 84** |
| Reviews verdict-first | **26 of 84** |
| Notification ingest | 270 notifications, 725.5 KB, 40.3 MB-turns carry |
| Two hot reference docs | 74 reads → **69.3 MB-turns, 7.5% of all carry** |
| Mechanical-fact Bash calls | 2,191 / 2.07 MB |
| `task-resolve.sh` adoption | 16 of 213 streams |
| zsh glob failures | 113 / 11,102 (1.0%) |
| Subagent starting context | median 37.3k (task-234 target <28k) |

### task-227 (feature-shaped, 41 plan tasks, 18 sessions, 133 subagents)

| Metric | Baseline |
|---|---|
| Billed input | ≈949.3M; 6,808 turns (1,809 main / 4,999 subagent) |
| Per-turn floor | 23.3k main / ~35–38k subagent ≈ 23% of the task |
| Phase split | execute 84.3%, **post-PR 12.7%**, design+plan 2.0% |
| Post-PR main-thread share | **94%** (target <50%); peaks 328k / 274k (target <200k) |
| Reviewer returns | 2,988–7,946 B (target ≤1,200 B median) |
| No-op `Bash true` turns | 30 in one agent (33% of its calls, ≈3.6M) — **target 0** |
| Main-thread polling calls | 20 / 11 / 22 in three sessions — **target 0** |
| Gate-forensics turns | ~30 at 170–290k ≈ 6.9M; 19 non-invocation `verify.sh` touches in one session (target ≤2) |
| Reviewer discovery before first check | 2 + ~12 turns (target ≤3) |
| Micro-delegation | 6 children, 4.32M in for 4,669 out |

### Counter-metrics — watch these for over-correction

- **Controller reads of a review artifact.** If they rise by more than ~1 per
  review, the return contract is too tight and detail belongs back in it.
- **Rework rate after a sliced read.** Judge `slice-first` on fix rounds caused,
  not on bytes saved. Under-reading is more expensive than over-reading.
- **`classification=uncertain` frequency.** If it becomes the common case, the
  classifier has stopped being informative and needs a new known layout, not a
  wider default.
- **Reviews suppressed to `APPROVED`.** `APPROVED_WITH_FINDINGS` should be a
  regularly used verdict; if it is never emitted, concerns are being hidden to
  hit a byte target.

### How to read the answers next time

```sh
tools/agent-ledger.sh summary <task>
```

Gives agents by type, per-type median return bytes, verdict distribution,
reviews that caused a fix, and handoff count with median context — the questions
both audits had to reconstruct by hand.

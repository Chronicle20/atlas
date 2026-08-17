# Agentic cost audit — task-227 (cash shop name change / world transfer)

**Question this answers:** where did a real Atlas task spend model effort moving
information around or rediscovering mechanical facts rather than performing
useful semantic reasoning, and which changes are most likely to reduce that cost
across future tasks?

**Method.** All figures come from `<CLAUDE_HOME>/tools/session-digest.sh` digests and
direct `jq` queries over the retained session transcripts in
`<CLAUDE_HOME>/projects/<project-slug>/`. No token count in
this document is estimated unless labelled as such. "Billed input" means
`cache_read_input_tokens` summed over de-duplicated assistant turns — the context
re-read on every turn, which dominates the bill.

**Scope note.** A previous `/audit-session` run produced
`<CLAUDE_HOME>/audits/task-227-execution-chain.md`, covering the eight `/execute-task`
controller sessions through an orchestration lens (agent length, model pinning,
context drag, wall clock). This audit does not repeat that. It covers the **whole
lifecycle** — design, plan, execute, merge, and the post-PR live-testing phase the
earlier audit never saw — through three different lenses: deterministic
discovery, subagent→controller communication, and tool-result pollution.

---

## 1. Executive summary

Across 18 retained sessions, task-227 consumed **≈949M billed input tokens** and
**≈1.28M output tokens** over **6,808 assistant turns** (1,809 main-thread, 4,999
across 133 subagents). It produced a 466-file, +36,647/−557 branch and PR #1370,
which is still open and in live testing.

Three findings dominate, and none of them is "a tool returned too many bytes."

**1. Turns, not bytes, are the unit of waste.** Every main-thread turn carries a
measured fixed floor of **23.3k cached prefix tokens** (turn-1 `cache_read` is
23,271 in both `f6b5473a` and `51371205`, and 24,666 in `f777c2c8`) before the
session has loaded anything of its own. Every subagent dispatch carries a **~35–38k
floor** (`agent-a8ddff62240065648` turn 1: 38,178 input tokens; the whole 2-turn
`agent-a71266afaa0b2dfa0` cost 52,857). So a turn that produces nothing still costs
23k–120k depending on when it fires, and a subagent that answers one question still
costs ~50k. The largest single concentrations of waste in this task are both
turn-count phenomena: **30 `Bash true` no-op turns inside one reviewer agent** (33%
of its 92 tool calls, ≈3.6M billed input) and **20 process-polling calls in
`39f01959`'s main thread** at 170–290k context. Byte-level pollution is real but
secondary.

**2. The compact-return contract already works — for implementers and verifiers,
and not for reviewers.** Implementer notifications land at 1.6–2.6KB and are already
near-ideal (status, commit, test summary, one concern, report path). Verifiers land
at 0.8–1.5KB. **Reviewers land at 3.4–8KB** and duplicate a durable artifact they
themselves just wrote: `Review Task 26` streamed 3,370B into the controller
*and* wrote a 12.2KB `task-26-review.md`, with roughly 1.9KB of the streamed summary
being five restatements of PASS the controller could not act on. This is the one
agent class where a contract change is justified; recommending one for implementers
would be manufacturing work.

**3. The post-PR phase is a delegation blind spot.** The four live-testing/bug-fix
sessions (`fd6e44a3`, `f777c2c8`, `ce7eb6b5`, `34c712d5`) cost **120.5M billed
input — 12.7% of the whole task — of which 113.4M (94%) is main thread** with only
3 subagents between them. Compare the execute phase: 800.0M total, of which only
150.6M (19%) is main thread. Main-thread tokens are the expensive kind, because
context grows monotonically and never resets. `fd6e44a3` and `f777c2c8` alone burned
76.7M solo, peaking at 328k and 274k.

Deterministic discovery — the first area investigated — turned out to be the
**smallest** of the three opportunities at session-bootstrap level (~0.7% of total),
but a genuine one at two specific points: gate/verify forensics, and reviewer
applicability. Section 4 says so plainly rather than inflating it.

---

## 2. Audited task and evidence available

### Why task-227 is representative

It exercised every part of the workflow, at a scale that makes each part visible:

| Property | Value |
|---|---|
| Phase commands used | `/design-task`, `/plan-task`, `/execute-task` (×8 sessions) |
| Plan size | 41 tasks, 3,427 lines (`plan.md`, 166.6KB) |
| Branch | 91 commits, 466 files, +36,647 / −557 |
| Services touched | atlas-character, atlas-channel, atlas-cashshop, atlas-saga-orchestrator, atlas-parties, atlas-buddies, atlas-configurations, atlas-ui |
| Other surfaces | 9 packet codecs, ~70 coverage-matrix cells, 9 seed templates, Kafka contracts, REST resources, GORM entities |
| Agent classes exercised | `atlas-implementer`, `packet-implementer`, `atlas-verifier`, `backend-guidelines-reviewer`, `frontend-guidelines-reviewer`, ad-hoc `general-purpose` reviewers, one rejected `fork` |
| Handoffs | 8 controller sessions chained via `RESUME.md` + `.superpowers/sdd/plan/progress.md` |
| Terminal state | PR #1370 **open**, in live testing on `atlas-pr-1370` |

It is also the task with the richest retained evidence in the repo: 28 artifacts in
the task folder, a 236KB `progress.md` ledger, 22 review diffs, and 18 recoverable
session transcripts.

### Evidence used

**Directly observable:**
- Session transcripts (`<CLAUDE_HOME>/projects/.../<id>.jsonl`) and per-subagent
  transcripts (`<id>/subagents/agent-*.jsonl` + `.meta.json`), giving per-turn
  `cache_read` / `output` usage, agent type, model, tool ledgers with per-call
  result byte counts.
- Task artifacts in `docs/tasks/task-227-cash-name-change-world-transfer/`.
- `.superpowers/sdd/plan/` briefs, reports, progress ledger, review diffs.
- Git history on the branch; `tools/verify.sh` source.
- The prior audit `<CLAUDE_HOME>/audits/task-227-execution-chain.md`.

**Reasonable inference (labelled where used):** attribution of a specific token
figure to a specific *cause* (e.g. "this read was carried for 200 turns") is
arithmetic on observable quantities, not a measured counterfactual.

**Not available from retained telemetry:**
- Per-turn breakdown of *what* occupies the cached prefix (system prompt vs. tool
  schemas vs. CLAUDE.md vs. agent/skill listings). Only the total is recorded.
- Whether a given file read was actually used in a later decision.
- The `/spec-task` session for task-227 could not be located; `prd.md` exists
  (33.3KB) but its authoring session is not in the retained set. Phase-1 cost is
  therefore excluded from all totals.
- The `<usage><subagent_tokens>` figure embedded in completion notifications is
  **not** billed input: `Implement Task 37` reported 190,443 while its transcript
  shows 11,809,658 billed input. It is unusable as a cost signal (see §11).

---

## 3. Execution reconstruction

### 3.1 The chain and what each phase cost

| Session | Phase | Turns | Main billed-in | Subagents | Sub billed-in | Peak ctx |
|---|---|---|---|---|---|---|
| `3febd076` | `/design-task` | 56 | 6.99M | 0 | — | 184k |
| `51371205` | `/plan-task` | 76 | 11.85M | 0 | — | 252k |
| `f6b5473a` | execute 1 (Tasks 1–22) | 224 | 44.46M | 48 | 426.87M | 349k |
| `f7851213` | execute 2 (client cancel) | 139 | 21.14M | 19 | 51.73M | 264k |
| `2514add7` | execute 3 | 101 | 13.31M | 7 | 14.21M | 196k |
| `d6bf06aa` | execute 4 (Tasks 26–28) | 86 | 12.28M | 16 | 46.97M | 217k |
| `41bf8a26` | execute 5 | 94 | 12.72M | 12 | 22.60M | 209k |
| `5779cd80` | execute 6 | 70 | 8.21M | 6 | 7.47M | 169k |
| `b009d376` | execute 6′ (gate hang) | 82 | 10.15M | 0 | — | 182k |
| `39f01959` | execute 7 (Tasks 37–39 + branch review) | 164 | 28.35M | 21 | 79.54M | 291k |
| `9058e1fe`, `6fd7c405`, `a1e9cdd6`, `744a753b` | merges + follow-ups | 109 | 8.94M | 1 | 0.99M | — |
| `fd6e44a3` | post-PR live test/fix | 223 | 44.79M | 0 | — | 328k |
| `f777c2c8` | post-PR live test/fix | 172 | 31.86M | 0 | — | 274k |
| `ce7eb6b5` | post-PR bug fix (world-transfer crash) | 148 | 27.73M | 2 | 5.31M | 276k |
| `34c712d5` | post-PR live test/fix | 65 | 9.02M | 1 | 1.83M | — |
| **Total** | | **1,809** | **291.8M** | **133** | **657.5M** | |

**≈949.3M billed input, ≈1.28M output, 6,808 assistant turns.**

Phase shares of the 949.3M:

| Phase | Billed input | Share | Main-thread share of that phase |
|---|---|---|---|
| Design + plan | 18.8M | 2.0% | 100% |
| Execute (8 sessions) | 800.0M | 84.3% | 19% |
| Merges / follow-ups | 9.9M | 1.0% | 90% |
| **Post-PR live testing** | **120.5M** | **12.7%** | **94%** |

`5779cd80` and `b009d376` share a start timestamp and an identical first ~61 tool
calls — almost certainly one session recorded twice. Subtracting it moves the total
by ~2% and changes no conclusion.

### 3.2 Timeline of the execute loop (representative session `39f01959`)

Directly observable from the tool ledger:

1. Calls **#1–#6** — bootstrap: `tools/task-resolve.sh`, re-read the rtk tee log
   because the wrapper swallowed stdout, `ls` the task folder, `cat RESUME.md`
   (8.3KB), `tail -40 progress.md`, `tools/task-brief.sh … 37`.
2. Calls **#7–#20** — controller reads the brief and greps the Kafka/consumer/handler
   surface itself, then appends findings to the brief.
3. Call **#21** — dispatch `atlas-implementer[sonnet]` for Task 37 (89 turns, 11.8M).
4. Call **#22** — launch `tools/verify.sh --quick` backgrounded.
5. Call **#23** — read the 8.9KB `task-37-report.md` (the notification had already
   delivered the summary).
6. Calls **#26–#33** — locate the next plan task by grepping `plan.md` and running
   `task-brief.sh` again.
7. Calls **#83–#126** — a ~30-call forensic investigation into why `tools/verify.sh`
   behaved differently on this branch than on main (`git diff` of the script,
   `stat` mtimes, snapshot copies, narrowed re-runs with `--base`).
8. Calls **#94/#95** — the branch-level review: `backend-guidelines-reviewer` and
   `frontend-guidelines-reviewer` in parallel.
9. Calls **#101–#128** — three fix agents and a merge agent.

### 3.3 Retries, failed approaches, repeated investigation

- **Rejected forks (observable).** Inside `agent-a65fed95c965f0f84`
  (`backend-guidelines-reviewer`), calls #3–#8 dispatched six `fork[INHERIT]`
  children. All six were refused by `fork-dispatch-guard.sh` ("Refused: unjustified
  fork dispatch"). The agent correctly re-dispatched them as
  `general-purpose[sonnet]` at #9–#14. Cost: 6 wasted turns; benefit: six forks that
  would each have inherited a 46–158k context were prevented. **The guard paid for
  itself.**
- **Gate false FAIL (observable, from the prior audit's evidence).** `b009d376`
  #26/#27 corrected an `atlas-verifier` that reported "still executing"; #41 records
  it then produced a false FAIL the controller re-ran by hand.
- **Ten fix rounds across 41 plan tasks**, each traceable to a named reviewer
  finding. Productive iteration, not rework.
- **Post-PR rediscovery (observable).** `f777c2c8` #4/#5 re-grepped the repo for
  `namechange|name_change` and `pendingchange|pending_change` to relocate the code
  the same branch had written 48 hours earlier; `fd6e44a3` touched `plan.md` 4× for
  19.6KB of in-context text to re-establish what the plan had said.

### 3.4 Final repository state transition

91 commits on `task-227-cash-name-change-world-transfer`, PR #1370 open, HEAD
`3843015e5`. The transition is **not yet complete** — the 12.7% post-PR phase is
still accruing.

---

## 4. Investigation A — deterministic discovery

The premise: find work spent establishing *mechanical* facts that a script could
have produced. The honest headline is that **the routine bootstrap is already cheap
and largely already scripted**, and two specific non-routine cases are where the
money is.

### A0. The routine bootstrap — already solved, do not touch

Every session opens with 2–6 calls to re-establish worktree, task folder, branch,
and progress. Measured cost: 39f01959 calls #1–#6 returned 280B+274B+762B+8.3KB+2.6KB+145B
at 23–60k context ≈ **0.38M billed input**, and across ~18 sessions ≈ **7M ≈ 0.7% of
the task**. `tools/task-resolve.sh` (landed `64c023875`, 2026-08-14) and
`tools/task-brief.sh` already exist and do exactly this.

Two small, real defects inside it:

1. **The rtk wrapper swallows script output, costing a second turn.** `39f01959`
   #1 → `tools/task-resolve.sh "task-227"` returned 280B; #2 → `cat
   <XDG_DATA_HOME>/rtk/tee/1786795581_tools_task-resolve_sh_task-227.log`. The
   identical pair appears at `d6bf06aa` #1/#2. A deterministic tool defeated by a
   token-optimizing wrapper is a net loss: the wrapper saved a few hundred bytes and
   cost a full turn (~25–50k). *Belongs in: wrapper behavior (rtk filter exemption
   for `tools/*.sh`).*
2. **The tool is used in only half the sessions.** `3febd076` #2, `51371205` #2,
   `f777c2c8` #2, `fd6e44a3` #1 all hand-rolled `git worktree list && ls docs/tasks
   | grep 227` instead. Cost per instance is one call, so the impact is Low —
   but the inconsistency means the tool's own output contract is not what later
   turns are reasoning over. *Belongs in: agent instructions / phase commands.*

**Verdict: Low impact. Fix (1); mention (2). Do not build new bootstrap tooling.**

### A1. Verification-gate applicability and behavior — the real one

**What Claude was trying to determine.** Why `tools/verify.sh --quick` was building
all 86 modules on this branch, whether the script differed from `main`, what change
base it was using, and which modules it had actually selected.

**Investigation required.** `39f01959` calls **#83–#89, #103, #107–#126** — roughly
30 turns at 170–290k context. Directly observable examples:

```
#83  git log --oneline -3 -- tools/verify.sh; git diff --stat origin/main...HEAD -- tools/verify.sh
#84  git diff origin/main...HEAD -- tools/verify.sh
#85  bash -n tools/verify.sh; git show origin/main:tools/verify.sh > $CLAUDE_JOB_DIR/tmp/...
#87  git status --porcelain -- tools/verify.sh; git log -1 --format=... -- tools/verify.sh; date
#89  cp tools/verify.sh $CLAUDE_JOB_DIR/tmp/verify-snapshot.sh; ... tools/verify.sh --quick ...
#103 tools/verify.sh --quick --base 8c3736a > $CLAUDE_JOB_DIR/tmp/gate-n1.log
```

At the session's mean context over that range (~230k), 30 turns ≈ **6.9M billed
input, ~24% of that session's main thread**. Also visible in `fd6e44a3` (#157
`grep -n "atlas-ui" tools/verify.sh`, #166 grepping `.github/workflows`) and
`d6bf06aa` #67 (`grep -n "gua…"` inside `verify.sh` to learn how it invokes guards).

**Was interpretation needed?** Partly. Deciding *what to do* about an over-broad
gate is semantic. But every input to that decision is mechanical and already
computed inside the script: `tools/verify.sh:96–130` computes `CHANGED`;
`changed_modules()` (`:149`) selects the module set; `changed_tool_suites()`
(`:122`) selects guard suites; `UI_CHANGED` (`:394`) selects the frontend lane; the
`libs/` fan-out escape is at `:166`. Today the script prints only
`"verify.sh: change base <sha> — N changed path(s)"` (`:109`) and a warning at
`:173`. Claude reverse-engineered from source what the script already knew.

**Deterministic mechanism.** A `--facts` (or `--dry-run`) mode on `tools/verify.sh`
that runs the selection logic and exits without building. The logic exists; this is
a print statement, not an algorithm.

**Compact output contract:**

```
base=8c3736a
changed_paths=41
changed_services=atlas-cashshop,atlas-channel
changed_libs=none
fanout_reason=none            # or: libs/atlas-packet → all_modules
modules_selected=4
ui_changed=false
guard_suites=outbox,goroutine
gates=build,vet,test,lint
```

**Estimated benefit: High.** It removes the single largest block of mechanical
main-thread investigation in the chain, at the most expensive context depth. It
also makes the `--base` decision (which landed as `930ab4887` *during* `39f01959`)
self-explanatory instead of something to be rediscovered.

**Risk: Low.** Facts only; the model still decides what to do with them. The one
failure mode is drift between `--facts` output and the real selection path — avoided
by having `--facts` call the same functions and exit, not by duplicating them.

### A2. Reviewer applicability — which audit families apply

**What Claude was trying to determine.** Which reviewers to dispatch, and inside
`backend-guidelines-reviewer`, which DOM/FILE/SUB/EXT/SCAFFOLD/SEC families apply
to this diff.

**Investigation required.** `docs/superpowers-integration.md:59` says the skill
"dispatches the relevant subset of these agents" — the applicability decision is
left entirely to the model. Inside the reviewer,
`agent-a65fed95c965f0f84` calls #1 and #2 are:

```
#1  git diff --stat origin/main...HEAD -- '*.go' | tail -80      → 4.9KB
#2  git diff --stat origin/main...HEAD -- '*.go' | head -140     → 8.7KB
```

13.6KB of `--stat` output at the agent's turn 1–2, carried through all 83 of its
turns. It then spent #62–#67 rediscovering whether a Dockerfile exists and whether
`atlas-saga` is referenced in it (SCAFFOLD applicability), and #70–#73 grepping for
topic env vars and configmaps (EXT applicability) — six turns each on questions
answerable by a path glob.

**Was interpretation needed?** For *which rules to apply*, no. Whether a diff
touched a `resource.go`, a `kafka/message/*.go`, a `Dockerfile`, a `deploy/`
configmap, an `entity.go`, or a new service directory is a pure path classification.
For *whether the code satisfies* those rules — entirely semantic, and must stay
with the model.

**Deterministic mechanism.** A `tools/change-surfaces.sh [--base <rev>]` emitting a
classification block, consumed by `superpowers:requesting-code-review` to pick the
roster and passed into the reviewer's brief so it starts with the family list
instead of deriving it.

**Compact output contract:**

```
changed_services=atlas-cashshop,atlas-channel,atlas-character,atlas-saga-orchestrator
changed_libs=atlas-packet,atlas-saga
go_changed=true
ts_changed=true
rest_surface=true             # any */resource.go or */requests.go
kafka_surface=true            # any kafka/message|producer|consumer
db_surface=true               # any entity.go / administrator.go / migration
deploy_surface=true           # deploy/** or Dockerfile
packet_surface=true           # libs/atlas-packet/**
new_service=false
backend_audit_families=DOM,FILE,SUB,EXT,SEC
frontend_review=true
```

**Estimated benefit: Medium–High.** It replaces ~12 discovery turns inside a 10M
reviewer, drops the 13.6KB `--stat` pair for a ~400B block, and — more importantly
— removes an unstated judgement call from the dispatch step, where a missed
`frontend_review=true` costs a whole review pass.

**Risk: Low–Medium.** A path heuristic can under-classify a novel layout. Mitigate
by treating the block as *additive* — it tells the reviewer which families are
definitely in scope; the reviewer may still add one, but may not silently drop one.

### A3. Tool-schema discovery (`ToolSearch`) — small but avoidable

`ce7eb6b5` spent **4 `ToolSearch` turns** to load MCP schemas for 13 MCP calls
(#16 for three IDA tools, #23 for `xrefs_to`, #29 for `insn_query`). `fd6e44a3`
spent **5 `ToolSearch` turns for 5 MCP calls** — one schema-loading turn per unit
of work. The tool set needed for a packet/IDA investigation is knowable up front
from the task type.

**Mechanism:** the packet/RE playbooks (`docs/packets/PROCESS.md`,
`docs/reverse-engineering.md`) should name the exact tool set as a single
`select:` line so it is loaded in one turn. **Benefit: Low.** **Risk: none.**
*Belongs in: skill/documentation.*

### A4. Not worth automating

- Locating which handler implements a cash-shop opcode, or which consumer reads a
  topic: these look mechanical but the *right* grep depends on knowing the domain
  vocabulary. `39f01959` #8–#19 is 12 calls of this and every result was small
  (54B–4.1KB). Leave it.
- Deciding whether a diff crosses a service boundary in a *semantically* dangerous
  way. `docs/superpowers-integration.md` is explicit that the gate cannot see this;
  neither can a classifier.

---

## 5. Investigation B — subagent → controller communication

Agent results reach the controller as `<task-notification>` user messages, not as
tool results (the `Agent` tool result itself is a constant 1,096-byte launch stub).
This means **the digest's "tool result bytes" figure systematically excludes agent
returns** — in `f6b5473a` they are 145.3KB against 181.3KB of all tool results
combined.

### Measured return sizes

| Session | Notifications | Total | Median | Max |
|---|---|---|---|---|
| `f6b5473a` | 52 | 145.3KB | 2,682B | 5,992B |
| `39f01959` | 25 | 52.4KB | 1,552B | 7,946B |
| `d6bf06aa` | 12 | 23.1KB | 1,857B | 3,370B |
| `ce7eb6b5` | 12 | 13.6KB | 383B | 7,159B |
| `fd6e44a3` | 5 | 2.1KB | 409B | 416B |

Broken down by agent class (all three sessions with a full roster):

| Class | Return size range | Verdict |
|---|---|---|
| `atlas-verifier` | 841–1,546B | **Already minimal.** Leave alone. |
| `atlas-implementer` / `packet-implementer` | 1,645–2,595B | **Already near-ideal.** Leave alone. |
| Reviewers (`general-purpose`, guidelines) | 2,988–7,946B | **The opportunity.** |

### B1. Implementers — already compliant, no change recommended

`Implement Task 37` (89 turns, 11.8M billed input) returned 1,645B:

> **Status:** DONE_WITH_CONCERNS
> - Commit: `bcb5cf58a` — "feat(task-227): correlate cash purchase outcomes with a transaction id"
> - Test summary: atlas-cashshop … all pass; atlas-channel … all pass. New/mutated tests confirmed genuine RED→GREEN (quoted in report).
> - Concern: discovered a pre-existing defect … `message.Emit` only flushes buffered events when the wrapped closure returns `nil` …
> - Report: `…/.superpowers/sdd/plan/task-37-report.md`

Status, commit SHA, verification result, one blocking-adjacent concern, artifact
path. That *is* the compact contract. The controller needed all five fields
immediately: it used the concern to shape Task 38/39's brief (visible at `39f01959`
#33). **Recommending a schema here would be manufacturing work.**

One real duplication: the controller then Read the full 8.9KB `task-37-report.md`
at #23 anyway. Whether that was necessary is **inconclusive from the evidence** —
the notification's "quoted in report" phrasing invites the read, and the controller
did subsequently catch real gaps by independent checking. Flagging the phrasing, not
the read.

### B2. Reviewers — the one contract worth changing

`Review Task 26` (`d6bf06aa`) returned **3,370B** whose first line is:

> Review written to `…/docs/tasks/task-227-…/task-26-review.md`.

…followed by 2.9KB restating what is in that 12.2KB file. Five numbered PASS
paragraphs (credential version split, version gate not duplicated, fail-closed on
unset credential, credential never logged, RecordPicAttempt on both outcomes) — each
with file:line evidence, each already written to disk, and **none of them
actionable**. Only the last block ("Two findings beyond the four priorities:
**Blocking**: the FR-4.7 pink-text storage warning … is entirely absent") changed
what the controller did next.

Same shape at larger size: `Review Task 13 saga handlers` (5,992B in `f6b5473a`) —
a "Spec compliance: ✅" section, a seven-row consumer-enumeration table all marked
✅, three "PRIORITY … ✅" sections, then two Minor findings at the very end. And
`Review Task 38 commit` / `Review Task 39 commits` in `39f01959` at 7,946B and
7,894B.

**Information the controller genuinely needed immediately:** verdict; blocking and
non-blocking finding count; each blocking finding as one sentence plus `file:line`;
artifact path. **Information that could have stayed in the artifact:** every PASS
justification, every evidence table, every "not evaluable" enumeration.

**Did the controller later need something a compact return would have omitted?**
Checked, and the answer is no in the observed cases: after each review the controller
either dispatched a scoped fix agent naming the finding (`f6b5473a` #46, #55, #99,
#116, #124) or moved on. It never went back to the streamed PASS text. Where it
wanted more, it read the artifact — which is the correct behavior and stays possible.

**Recommended reviewer return contract:**

```
verdict: APPROVED | APPROVED_WITH_FINDINGS | CHANGES_REQUIRED
artifact: <path>
blocking: <n>
  - <file:line> — <one sentence>
non_blocking: <n>            # counts only; detail in artifact
not_evaluable: <n>           # counts only; detail in artifact
scope_confirmed: <diff range actually reviewed>
```

Applied to `Review Task 26`, that is ~450B against 3,370B — a **~2.9KB saving,
carried for the remaining ~58 turns of the session**. Across the 22+ reviews in the
chain the direct saving is on the order of 60–70KB of persistent context, most of it
loaded mid-session.

`scope_confirmed` is not padding: `docs/superpowers-integration.md` makes scope the
reviewer's contract, and it is the one fact a controller cannot recover from the
artifact path alone without reading the file.

### B3. Micro-delegation — the return was compact, the dispatch was not

Six children of `backend-guidelines-reviewer` in `39f01959`:

| Child | Turns | Billed input | Output tokens |
|---|---|---|---|
| Pending_change domain DOM/FILE checklist | 20 | 2,009,604 | 2,997 |
| DOM-21 atlas-constants reuse check | 10 | 713,475 | **25** |
| Orphan reconciliation severity assessment | 10 | 634,682 | **39** |
| Saga step handler coverage audit | 12 | 576,718 | 1,581 |
| Hand-mirrored cashshop kafka struct parity | 6 | 335,081 | **22** |
| Cashshop Purchase tx boundary audit | 2 | 52,857 | **5** |
| **Total** | 60 | **4.32M** | 4,669 |

Four of the six produced *fewer than 40 output tokens*. Their returns were maximally
compact; the cost was the ~35k dispatch floor plus 2–20 turns of context growth each.
`Cashshop Purchase tx boundary audit` made **one** tool call and cost 52,857 billed
input.

This is a distinct failure mode from B2 and needs a different rule: **a question
answerable in one or two tool calls should be answered inline, not delegated.** The
break-even is roughly 4–5 turns of the parent's own work, given a 35k dispatch floor
against a parent turn at 100–150k. *Belongs in: agent definitions
(`backend-guidelines-reviewer`) and `docs/agent-dispatch.md`.*

### B4. Continuation handoffs — working well, leave alone

The `RESUME.md` (8.3KB) + `progress.md` (236.9KB, tailed not read) pair carried state
across all eight execute sessions. `39f01959` reconstructs a 22-task history in two
calls (#4, #5) for 10.9KB. That is a genuinely cheap handoff and the discipline
should be preserved verbatim. Note the ledger tails rather than reads `progress.md`
— exactly right.

---

## 6. Investigation C — tool-result / context pollution

### C1. `Bash true` — 30 no-op turns inside one reviewer

The single worst item found. In `agent-a65fed95c965f0f84`
(`backend-guidelines-reviewer`, 83 turns, 92 tool calls, 9,994,299 billed input),
**30 of the 92 tool calls are `true`** returning 31 bytes — at ledger positions
#18–#22, #27–#29, #36–#38, #40–#41, #47–#48, #50–#51, #53–#54, #56–#58, #68–#69,
#74–#75, #84–#88. They cluster immediately after the six child dispatches: the agent
had nothing to do while its async children ran, and burned turns to stay alive.

At that agent's mean 120k billed input per turn, **30 turns ≈ 3.6M — ~36% of the
agent's entire cost**, for zero information.

Repo-wide check across the four sessions with subagents: 2,660 subagent `Bash` calls,
**35 no-ops, of which 30 are this one agent** (`f6b5473a`: 5 of 1,829; `d6bf06aa` and
`ce7eb6b5`: zero). So this is not a diffuse habit — it is one structural situation:
*an agent that fans out async children and then has no defined wait primitive.*

- **Better interface:** the same `Monitor`/until-loop discipline CLAUDE.md already
  mandates for processes, extended to child agents — or, simpler, forbid nested
  async fan-out from a reviewer and have it do the six checks inline (see B3).
- *Belongs in: agent instructions (`backend-guidelines-reviewer`) + `docs/agent-dispatch.md`.*
- **Impact: High** for reviewer agents; **Medium** overall.

### C2. Main-thread process polling — 20 calls at 170–290k context

`39f01959` main thread: 108 Bash calls, **20 matching `pgrep|pkill|ps -ef|sleep|tail
/tmp/claude`**. `fd6e44a3`: 173 Bash calls, **11 polling**. The largest single result
in `39f01959` is #114 at **9.8KB**: `pgrep -af "lint[.]sh" … kill -9 …`. The prior
audit found the same pattern at `b009d376` #47–#68 (22 consecutive calls, one 22KB
`ps -ef` dump).

The correct pattern is already demonstrated in the same task: `ce7eb6b5` #101, #125,
#131 use `Monitor` with an `until [ -f … ]` loop and a 2,400,000ms timeout, returning
209B each. Three sessions later, the anti-pattern is back. This is a **rule-adherence**
problem, not a knowledge problem — CLAUDE.md already forbids it.

**Impact: Medium–High** (20 turns × ~230k ≈ 4.6M in one session).
*Belongs in: workflow policy — a `PreToolUse` guard on Bash commands matching
`pgrep|pkill|^ps ` would make the existing rule machine-checked, the same way
`fork-dispatch-guard.sh` made the fork rule machine-checked.*

### C3. Whole-document reads at turn 4–8 — the classic multiplier

| Session | Call | Size | Turn | Turns carried | Est. persistent cost |
|---|---|---|---|---|---|
| `51371205` (plan) | #4 `prd.md` + #5 `design.md` | 35.2KB + 36.7KB | 4–5 | 71 | ≈18k tok × 71 ≈ **1.28M** (11% of session) |
| `3febd076` (design) | #4 `prd.md` | 35.2KB | 4 | 52 | ≈9k tok × 52 ≈ **0.46M** (7%) |
| `f6b5473a` (execute) | #5–#8 `plan.md` slices | 12.7+27.3+15+8 = 63KB | 5–8 | 216 | ≈16k tok × 216 ≈ **3.4M** (7.7%) |

The `f6b5473a` case is the actionable one: `tools/task-brief.sh` exists precisely to
avoid loading `plan.md` into a controller, and the later sessions use it correctly
(`39f01959` #6/#33/#65; `d6bf06aa` #40/#65). `f6b5473a` also re-touched `plan.md`
**44× via shell for 76.5KB total**.

The design/plan reads are a different matter — see §8; I am **not** recommending
those be trimmed.

### C4. MCP results — large, mid-session, and schema-loaded one call at a time

`ce7eb6b5`, 13 MCP calls totalling **71.7KB**, of which two calls are 52.8KB:

```
#30  mcp__ida-pro__insn_query   queries={"start":"0x7ef7f9","end":"0x7…}   24.7KB
#31  mcp__ida-pro__decompile    addr=0xa22785                              28.1KB
```

Both landed at turns 30–31 of a 148-turn session → ~13k tokens carried ~117 turns ≈
**1.5M, 5.5% of that session**. A `decompile` of a large function and a raw
instruction-range dump are both *bounded-window* requests: the useful portion was the
read order of one packet, not the whole function body.

Kubernetes, by contrast, shows the mechanism working and not working in the same task:

- `f777c2c8` #9 `pods_list_in_namespace(atlas-pr-1370)` → **17.5KB at turn 9 of 172**
  ≈ 4.4k tokens × 163 turns ≈ **0.72M**. A namespace pod list is almost entirely
  metadata the session never used; the three pod names it wanted were the only payload.
- The same session's `pods_log` calls (#11, #20, #23, #27) returned **1.3KB stubs**
  because the harness spilled the large payloads to
  `<session>/tool-results/mcp-kubernetes-pods_log-*`, and the model then sliced them
  with `grep`/`python3` from disk (#13–#15, #21–#22, #24–#26, #28). **That is the
  correct design and it worked** — bounded logs never entered the window.

The gap is that `pods_list_in_namespace` and `ida-pro` results are *just under* the
spill threshold, so they land whole.

- **Better interface:** a field-projection or `--brief` mode on
  `pods_list_in_namespace` (names + phase + restarts only); for IDA, prefer
  `func_query`/targeted `xrefs_to` over full `decompile` when the question is a read
  order, and bound `insn_query` ranges.
- *Belongs in: MCP/tool design for the k8s list; agent instructions + `docs/reverse-engineering.md` for IDA.*
- **Impact: Medium.**

### C5. Full-file reads inside implementers, duplicating the controller's own discovery

`agent-a8ddff62240065648` (Implement Task 37) calls #2–#6 read five files whole —
8.1 + 3.1 + 13.2 + 8.5 + 8.0 = **40.9KB at turns 2–6 of an 89-turn agent** ≈ 10k
tokens × 85 turns ≈ **0.85M, ~7% of the agent**. Four of those five files were
already inspected by the controller at `39f01959` #8–#19 while writing the brief.

The controller's discovery is therefore paid for twice: once at controller context
depth, once at agent context depth — and the brief transmits only the file *list*,
not what was learned. `docs/agent-dispatch.md`'s "brief-first discovery" contract
gets the agent to the right files fast; it does not stop it re-reading them whole.

- **Better behavior:** the brief's Files section should carry, per file, the
  specific symbol or line range that matters (the controller already knows it — that
  is what its greps found). *Belongs in: `tools/task-brief.sh` output shape + the
  controller step in `.claude/commands/execute-task.md`.*
- **Impact: Medium**, multiplied across every implementer in every task.

### C6. Repeated shell touches of the same file

| Session | File | Touches | In-context |
|---|---|---|---|
| `f6b5473a` | `plan.md` | 44× | 76.5KB |
| `39f01959` | `tools/verify.sh` | 19× | 5.5KB |
| `39f01959` / `fd6e44a3` | `$NVM_DIR/nvm.sh` | 10× / 12× | 2.8KB / 8.7KB |
| `fd6e44a3` | `plan.md` | 4× | 19.6KB |
| `51371205` | `character_cash_item_use.go` | 5× | 11.8KB |

The `nvm.sh` line is the interesting one: every gate invocation is prefixed with
`export NVM_DIR=… && . "$NVM_DIR/nvm.sh" && nvm use 22 >/dev/null && …`. That is
env bootstrap replicated into ~12 commands per session. An `.envrc` exists in the
working tree (untracked) — committing a direnv/`tools/` shim would delete the prefix
from every gate call. **Impact: Low** (bytes are small; it is the command-string
noise and the fragility that matter). *Belongs in: repo tooling.*

---

## 7. Multiplicative-cost observations

Cost pressure ≈ persistent context × subsequent turns × agent fan-out. Ranked by
that product, using the measured turn positions:

| Rank | Item | Size | Landed at | Turns carried | Est. persistent cost | Why it ranks here |
|---|---|---|---|---|---|---|
| 1 | **The per-turn floor itself** | 23.3k (main) / ~35–38k (agent) | turn 0 | every turn | ≈**42M main** (23.3k × 1,809) + ≈**180M subagent** (36k × 4,999) ≈ 23% of the task | Not removable by trimming reads. Only removable by having **fewer turns**. This is why C1/C2 (no-op turns) outrank every byte-level finding. |
| 2 | `f6b5473a` `plan.md` loads (#5–#8) | 63KB | turn 5–8 | 216 | ≈3.4M | Earliest possible position in the longest session, and `task-brief.sh` already existed. |
| 3 | Reviewer return verbosity (B2) | 2.9–7KB each, ~22 reviews | mid-session | 30–120 each | ≈2–4M aggregate | Each return is small; the fan-out multiplier is what makes it matter. |
| 4 | `ce7eb6b5` IDA pair (#30/#31) | 52.8KB | turn 30/148 | 117 | ≈1.5M | Large *and* early. |
| 5 | `51371205` prd+design (#4/#5) | 71.9KB | turn 4/76 | 71 | ≈1.28M | Large and earliest — but see §8; mostly justified. |
| 6 | `f777c2c8` `pods_list` (#9) | 17.5KB | turn 9/172 | 163 | ≈0.72M | Small payload, terrible position. |
| 7 | Implementer whole-file reads (C5) | ~41KB | turn 2–6 | 85 | ≈0.85M **per implementer** | The per-agent figure is modest; there were **133 agents**. |

**The contrast that proves the model.** `fd6e44a3` #180/#185 returned 7.6KB and
18.0KB from `namespaces_list` and `pods_list_in_namespace` — *larger* than
`f777c2c8` #9, but at turns 180 and 185 of a 223-turn session, so carried only
~40 turns ≈ 0.18M. The same call was **4× cheaper** purely by arriving late. Any
policy that ranks tool results by size alone will mis-prioritize these two.

**The dominant structural multiplier.** The post-PR phase (§3.1) is 12.7% of the
task at 94% main thread. A main-thread turn in `fd6e44a3` at turn 200 costs ~300k;
the same investigation inside a fresh subagent costs ~36k + its own growth. Nothing
in the workflow tells a debugging session to delegate, because the phase commands
stop at `/execute-task`.

---

## 8. Things that should remain unchanged

Explicitly: the following look expensive and are not the problem.

1. **The verification gate.** 17 `atlas-verifier` agents across the chain cost 3.65M
   = 0.49% of the execute phase, all on `haiku`, returning 841–1,546B. The prior
   audit tested and rejected the "verify.sh is the cost" hypothesis. Nothing to save.

2. **Review agents as a class.** ~37.9M ≈ 5.1% of the execute phase, and they found
   the Task 3 tenant-filter gap (Critical), the Task 4 vacuous test, the Task 11 gate,
   the Task 13 saga wiring, and the Task 14 missing saga starter. Five fix rounds
   trace to reviewer findings. §5's recommendation trims their **return format**, not
   their existence, their scope, or their thoroughness.

3. **`prd.md` and `design.md` loaded whole in `/plan-task`.** 71.9KB at turn 4–5 for
   ≈1.28M — the second-largest early load in the chain, and I am **not** recommending
   it be trimmed. `/plan-task` is the one phase whose entire output is a
   re-expression of those two documents; a plan written from excerpts is a plan with
   silent holes, and the resulting rediscovery would land in 41 implementer contexts
   rather than one controller. This is the canonical case of context that prevents
   expensive rediscovery.

4. **`RESUME.md` + `progress.md` continuation discipline.** 8.3KB + a tail, and it
   carried 41 plan tasks across eight `/clear` boundaries. `progress.md` is 236.9KB
   on disk and was never read whole. Exemplary; do not touch.

5. **`fork-dispatch-guard.sh`.** It cost 6 turns in `agent-a65fed95c965f0f84` and
   prevented six forks that would each have inherited a 46–158k parent context. Net
   strongly positive, and the model corrected itself immediately on being refused.
   This is the template for every other rule that is currently advisory.

6. **The tool-result spill mechanism.** `f777c2c8`'s `pods_log` payloads were spilled
   to `<session>/tool-results/` and entered context as 1.3KB stubs; the model then
   sliced them from disk. Nine bounded reads instead of four unbounded ones. Working
   as designed.

7. **Targeted greps during controller brief-writing.** `39f01959` #8–#19: twelve
   calls, results 54B–4.1KB, all feeding the brief. Cheap, semantic, and the reason
   the Task 37 implementer had a good starting point. Do not automate; do not cut.

8. **`atlas-implementer` / `atlas-verifier` return formats.** Already compact
   (§5, B1). Imposing a schema here is churn.

---

## 9. Prioritized opportunities

| # | Opportunity | Category | Evidence from task | Expected impact | Effort | Correctness risk | Recommendation |
|---|---|---|---|---|---|---|---|
| 1 | Eliminate no-op wait turns in agents that fan out async children | C / turn count | `agent-a65fed95c965f0f84`: 30 of 92 tool calls are `Bash true`, ≈3.6M ≈36% of that agent | **High** | Low | Low | **Do** — forbid nested async fan-out in reviewers; answer inline or use a bounded wait |
| 2 | Machine-check the "never poll a process" rule | C / turn count | `39f01959` 20 polling calls at 170–290k ≈4.6M; `fd6e44a3` 11; `b009d376` 22 (prior audit) | **High** | Low | Low | **Do** — `PreToolUse` Bash guard, modelled on `fork-dispatch-guard.sh` |
| 3 | `tools/verify.sh --facts` | A / deterministic | `39f01959` #83–#126, ~30 turns ≈6.9M reverse-engineering selection logic the script already computes at `:96–:194` | **High** | Low | Low | **Do** — print-only mode over existing functions |
| 4 | Compact reviewer return contract | B | `Review Task 26`: 3,370B streamed vs 12.2KB artifact, ~1.9KB unactionable PASS text; ×22 reviews | **Medium–High** | Low | Low–Medium | **Do** — verdict + blocking findings + counts + artifact path |
| 5 | Delegation discipline for the post-PR phase | Multiplicative | 120.5M (12.7% of task) at 94% main thread; `fd6e44a3`/`f777c2c8` peak 328k/274k with zero subagents | **High** | Medium | Low | **Do** — extend the handoff rule past `/execute-task` |
| 6 | `tools/change-surfaces.sh` + reviewer roster from it | A / deterministic | 13.6KB `git diff --stat` pair at reviewer turn 1–2; ~12 turns on SCAFFOLD/EXT applicability; roster choice is model-judgement today | **Medium** | Medium | Low–Medium | **Do** — facts only, additive to reviewer scope |
| 7 | Brief carries symbol/line ranges, not just file paths | C | Implementer #2–#6 read 40.9KB whole, four files the controller had already inspected | **Medium** | Medium | Low | **Consider** — needs a `task-brief.sh` output change plus a controller habit |
| 8 | Inline-vs-delegate break-even rule (~4–5 turns) | B | Six children of the backend reviewer: 4.32M billed input for 4,669 output tokens; four returned <40 tokens | **Medium** | Low | Low | **Do** — one paragraph in `docs/agent-dispatch.md` |
| 9 | Bound IDA / k8s-list result size | C | `ce7eb6b5` #30/#31 = 52.8KB at turn 30/148 ≈1.5M; `f777c2c8` #9 = 17.5KB at turn 9/172 | **Medium** | Low–Medium | Low | **Do** the guidance; **consider** the MCP projection flag |
| 10 | rtk exemption for `tools/*.sh` | A | `39f01959` #1/#2 and `d6bf06aa` #1/#2: a swallowed 280B result cost a whole turn | **Low** | Low | Low | **Do** — trivial, and it corrupts a deterministic contract |
| 11 | Preload MCP tool schemas per task type | A | `ce7eb6b5` 4 `ToolSearch`/13 MCP; `fd6e44a3` 5 `ToolSearch`/5 MCP | **Low** | Low | None | **Do** as documentation |
| 12 | Commit `.envrc` / drop the nvm prefix | C | `nvm.sh` in 10–12 command strings per session | **Low** | Low | Low | **Consider** |

---

## 10. Recommended implementation tasks

Five tasks. Each is justified by measured evidence above; nothing speculative.

### T1 — Kill no-op wait turns and machine-check the polling rule
*(opportunities 1, 2)*

- **Problem.** 30 `Bash true` turns in one reviewer (≈3.6M, 36% of that agent) and
  20 + 11 process-polling calls in two main threads at 170–290k context. CLAUDE.md
  already forbids polling; the rule is advisory and was violated in three separate
  sessions of one task.
- **Mechanism.** (a) A `PreToolUse` hook on `Bash` refusing commands that match
  `^\s*(true|:)\s*$` or contain `pgrep|pkill|^ps -ef|^ps aux|sleep [0-9]`, with a
  refusal message pointing at `Monitor` + `run_in_background` — exactly the shape of
  `fork-dispatch-guard.sh`. (b) Amend `.claude/agents/backend-guidelines-reviewer.md`
  to forbid async child fan-out: either answer inline or dispatch children
  synchronously.
- **Files.** `.claude/settings.json`, a new `.claude/hooks/wait-loop-guard.sh`,
  `.claude/agents/backend-guidelines-reviewer.md`, `docs/agent-dispatch.md`.
- **Benefit.** Removes the largest single concentration of pure-waste turns found.
- **Measurement.** Re-run the no-op query on the next completed task:
  `noop_bash / total_bash` across all subagents should be 0; `poll` count in main
  threads should be 0. Both are one `jq` line (§11).
- **Failure modes.** A legitimate `sleep` in a test harness gets refused — allow an
  explicit `POLL-JUSTIFIED:` prefix, mirroring `FORK-JUSTIFIED:`. An agent with
  nothing to do may emit some *other* no-op instead; the measurement query should be
  generalized to "tool calls returning <64B" if that appears.

### T2 — `tools/verify.sh --facts`
*(opportunity 3)*

- **Problem.** ~30 turns in `39f01959` (≈6.9M, 24% of that main thread) spent
  reverse-engineering the gate's own change-detection from its source, plus smaller
  instances in `fd6e44a3` and `d6bf06aa`.
- **Mechanism.** A flag that runs `CHANGED` computation, `changed_modules`,
  `changed_tool_suites`, and the `UI_CHANGED` / `libs/` fan-out decision, prints the
  `key=value` block from §A1, and exits 0 without building. Must call the existing
  functions, not restate them. Add the `fanout_reason` line so the `libs/` escape
  hatch (`:166`) explains itself instead of being discovered.
- **Files.** `tools/verify.sh`, `docs/verification.md`,
  `.claude/commands/execute-task.md` (use it before dispatching a gate).
- **Benefit.** Converts the most expensive block of mechanical investigation in the
  chain into one ~400B call.
- **Measurement.** Count tool calls whose command contains `verify.sh` but is not an
  *invocation* of the gate (i.e. `grep`/`git diff`/`sed` against the script) —
  19 in `39f01959`; target ≤2.
- **Failure modes.** Drift between `--facts` and the real selection path. Guard with
  a test in `tools/` asserting `--facts` module list equals the set the gate builds
  for a synthetic change set.

### T3 — Compact reviewer return contract
*(opportunities 4, 8)*

- **Problem.** Reviewer returns are 3.4–8KB and largely restate a durable artifact;
  ~1.9KB of `Review Task 26`'s 3.4KB was unactionable PASS text. Implementers and
  verifiers are already compact and must not be touched. Separately, six
  micro-delegations cost 4.32M for 4,669 output tokens.
- **Mechanism.** Add the §B2 return block to `.claude/agents/backend-guidelines-reviewer.md`,
  `.claude/agents/frontend-guidelines-reviewer.md`,
  `.claude/agents/plan-adherence-reviewer.md`, and the ad-hoc review prompt template
  in `.claude/commands/execute-task.md`. Add the inline-vs-delegate break-even
  paragraph (~4–5 parent turns, given a ~35k dispatch floor) to
  `docs/agent-dispatch.md`.
- **Files.** the three reviewer agent definitions,
  `.claude/commands/execute-task.md`, `docs/agent-dispatch.md`,
  `docs/superpowers-integration.md`.
- **Benefit.** ~2.5–7KB less persistent controller context per review, ~22 reviews
  per task of this size, at mid-session depth.
- **Measurement.** Median and max `<task-notification>` length by agent class (§11
  query). Target: reviewers ≤1,200B median. **Counter-metric:** count controller
  Read calls against `audit.md` / `task-NN-review.md` — if those rise by more than
  one per review, the contract is too tight and detail should move back.
- **Failure modes.** A reviewer compresses a genuine blocking finding into a
  one-liner the controller misreads. Mitigate by making `blocking` entries
  `file:line + one sentence` (not a count) and leaving only non-blocking and
  not-evaluable as bare counts.

### T4 — Extend delegation discipline past `/execute-task`
*(opportunity 5)*

- **Problem.** The post-PR live-testing phase is 120.5M (12.7% of the task) at 94%
  main thread, with 3 subagents across four sessions. `fd6e44a3` (223 turns, peak
  328k) and `f777c2c8` (172 turns, peak 274k) each ran a full debugging campaign solo.
  The four-phase flow ends at `/execute-task`, so nothing tells these sessions to
  delegate or hand off.
- **Mechanism.** A documented post-PR loop: reproduce → write the diagnosis to
  `docs/tasks/<task>/bug-<slug>.md` → dispatch a fresh implementer with that file as
  its brief → verify. The task folder already contains four such files
  (`bug-purchase-path-sets-assetid.md`, `bug-world-transfer-client-crash.md`,
  `name-check-gap.md`, `followup-check-time-eligibility.md`) — the artifact habit
  exists; the delegation habit does not. `ce7eb6b5` is the closest to the target
  shape: it opens by reading the bug file and does dispatch two agents.
- **Files.** `docs/superpowers-integration.md` (a "Phase 5 — post-PR" section),
  `CLAUDE.md` router table, possibly a `/fix-pr-bug` command.
- **Benefit.** Converts main-thread turns at 200–330k into subagent turns at 36–120k.
  On `fd6e44a3`'s profile that is a large fraction of 44.8M.
- **Measurement.** For the next task's post-PR sessions: main-thread share of phase
  billed input (94% today; target <50%) and peak main context (328k today; target
  <200k).
- **Failure modes.** Over-delegation of interactive debugging, where the operator is
  in the loop with a live client and round-trip latency matters more than tokens.
  The rule should trigger on *fix implementation*, not on reproduction.

### T5 — `tools/change-surfaces.sh` and a derived review roster
*(opportunity 6, with 9 and 11 folded in as documentation)*

- **Problem.** Reviewer applicability is model-judgement today
  (`docs/superpowers-integration.md:59`, "the relevant subset"), and the
  backend reviewer spends its first two calls on a 13.6KB `git diff --stat` pair plus
  ~12 later turns on SCAFFOLD/EXT applicability questions that are path
  classification.
- **Mechanism.** A script emitting the §A2 `key=value` block from
  `git diff --name-only <base>...HEAD`. `superpowers:requesting-code-review` picks
  the roster from `go_changed` / `ts_changed`; the block is passed verbatim into each
  reviewer's brief as the *minimum* family set. Alongside it, add to
  `docs/reverse-engineering.md` a one-line `select:` for the standard IDA tool set
  and a "prefer `func_query`/bounded `insn_query` over full `decompile`" rule, and to
  `docs/observability.md` a "never `pods_list` a whole namespace when you know the
  service name" rule.
- **Files.** new `tools/change-surfaces.sh` + `_test.sh`,
  `docs/superpowers-integration.md`, the reviewer agent definitions,
  `docs/reverse-engineering.md`, `docs/observability.md`.
- **Benefit.** Removes ~12 discovery turns from a 10M reviewer and ~13KB from its
  turn-1 context; removes a silent-miss risk from the roster decision.
- **Measurement.** Reviewer tool calls before its first substantive check (2 + ~12
  today; target ≤3); presence of the block in the reviewer brief.
- **Failure modes.** Under-classification on a novel layout silently narrows a
  review. Mitigate by making the block additive-only (a reviewer may add a family,
  never drop one) and by testing the classifier against the task-227 diff, whose
  correct answer is known.

**Not proposed as tasks:** anything touching the verification gate's coverage, the
review agents' scope, the design/plan document loads, or the implementer/verifier
return formats — see §8.

---

## 11. Measurement / telemetry gaps

What made this audit harder than it needed to be, and the minimal fix for each.
All of it is derivable from data the transcripts already contain; the gap is that
nothing aggregates it.

### What was hard, and why

1. **Agent returns are invisible to the tool ledger.** The `Agent` tool result is a
   constant 1,096-byte launch stub; the real return arrives later as a
   `<task-notification>` user message. `session-digest.sh`'s "tool result bytes"
   therefore excludes 145.3KB of the 326.6KB that actually entered `f6b5473a`'s
   context. Any cost audit that trusts that figure will miss the entire B category.
   **Fix:** teach `session-digest.sh` to fold notification lengths into the ledger
   and add a per-agent-class return-size table.

2. **The reported `<subagent_tokens>` figure is not a cost.** `Implement Task 37`
   reported 190,443 against 11,809,658 measured billed input — off by 62×. It is
   presumably an output or last-turn figure. **Fix:** either report billed input in
   the notification, or drop the field; a wrong number is worse than none.

3. **No per-phase attribution.** Reconstructing "design vs plan vs execute vs
   post-PR" required manually matching 18 session IDs to phases by reading opening
   prompts. **Fix:** stamp `task_id` and `phase` into the session record when a phase
   command runs — the phase commands already know both.

4. **Context-floor composition is unrecorded.** Turn-1 `cache_read` is 23,271 and
   that number is ~12–15% of a main session's bill, but nothing says how much is
   system prompt vs. tool schemas vs. CLAUDE.md vs. the agent/skill listings. Without
   it, "should we trim the preamble?" cannot be answered. **Fix:** one-time
   measurement, not ongoing telemetry — record the component sizes once per harness
   version.

5. **No counterfactual signal.** Whether a file read was later *used* is
   unknowable from the transcript. I marked such cases inconclusive rather than
   guessing. **Do not build tooling for this** — the cost of tracking it exceeds the
   value.

### Minimal recommended additions

Keep it to what `session-digest.sh` can compute from existing records:

| Metric | Why | Where |
|---|---|---|
| Notification size by agent class (median, max, total) | The only way to see B-category regressions | new digest section |
| No-op tool calls: count of calls returning <64B, split main/subagent | Catches C1 and its variants | new digest line |
| Polling calls: Bash commands matching `pgrep\|pkill\|ps -ef\|sleep [0-9]` | Catches C2; already a CLAUDE.md rule with no meter | new digest line |
| Turn-weighted result cost: `bytes × (total_turns − arrival_turn)` per tool call | Ranks results the way §7 does, instead of by raw size | replaces/augments LARGEST TOOL RESULTS |
| Per-agent-class rollup: n, turns, billed input, output, dispatch-floor share | Answers "controllers or workers?" directly | new digest section |
| `task_id` + `phase` stamped on the session | Answers "which phase dominates?" | phase commands |

Everything else the digest already provides. Three of the six are one `jq` line each;
none requires an observability platform.

### The questions this would then answer directly

- *Which workflow phases dominate inference cost?* — today: execute 84.3%, post-PR
  12.7%, design+plan 2.0% (computed by hand for this audit; should be a digest field).
- *Are controllers or workers more expensive?* — workers, 69% of the task
  (657.5M/949.3M) — but controllers are 100% of the post-PR phase, which is where
  the trend is going the wrong way.
- *Which tools inject the most persistent context?* — needs the turn-weighted metric;
  raw size gets it wrong (§7, the `fd6e44a3` vs `f777c2c8` `pods_list` contrast).
- *Did a new helper actually reduce discovery turns?* — needs the per-phase stamp
  plus the "calls mentioning `verify.sh` that are not invocations" counter from T2.
- *Did compact agent returns reduce controller growth?* — needs metric 1 plus the T3
  counter-metric (controller reads of the review artifact).

---

## Appendix — reproduction commands

```sh
# whole-chain totals
<CLAUDE_HOME>/tools/session-digest.sh find task-227
<CLAUDE_HOME>/tools/session-digest.sh digest f6b5473a

# agent return sizes by class
jq -rs 'map(select(.type=="user" and (.message.content|type=="string")
        and (.message.content|test("<task-notification")))|.message.content)
      | map({s:(capture("<summary>(?<x>[^<]*)").x), n:length})
      | sort_by(-.n) | .[] | "\(.n)\t\(.s)"' <session>.jsonl

# no-op turns across all subagents of a session
cat <session>/subagents/*.jsonl | jq -rs \
  'map(select(.type=="assistant")|.message.content[]?
     |select(.type=="tool_use" and .name=="Bash")|.input.command)
   | {total:length, noop:(map(select(test("^\\s*(true|:)\\s*$")))|length)}'

# polling calls in a main thread
jq -rs 'map(select(.type=="assistant")|.message.content[]?
        |select(.type=="tool_use" and .name=="Bash")|.input.command)
      | {bash:length, poll:(map(select(test("pgrep|pkill|ps -ef|ps aux|sleep [0-9]")))|length)}' \
  <session>.jsonl

# per-agent ledger
<CLAUDE_HOME>/tools/session-digest.sh agent 39f01959 a65fed95c965f0f84
```

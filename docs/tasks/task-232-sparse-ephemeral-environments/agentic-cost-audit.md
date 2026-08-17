# task-232 — agentic cost audit

**Audited:** 2026-08-16 · **Subject:** `task-232-sparse-ephemeral-environments`
**Scope:** the three questions posed — deterministic discovery (A), subagent →
controller communication (B), tool-result/context pollution (C).

**Relationship to prior work.** A session-level audit of this same task already
exists at `<CLAUDE_HOME>/audits/session-task-232-three-largest.md`, and task-234
(`docs/tasks/task-234-inference-efficiency-levers/`, landed on `main` as
`52e368269`) turned its findings into a spec. That audit answered *"how expensive
was each turn?"* and identified the fixed prefix (34%) and turn count (59%) as the
two dominant terms. **This audit answers a different question — *what was the
information moving through those turns, and how much of it was mechanically
derivable or duplicated?*** Findings here are new except where explicitly marked
as confirming the earlier work. Recommendations deliberately avoid re-proposing
task-234's FR-1…FR-5.

Every number below is measured from the retained session transcripts under
`<CLAUDE_HOME>/projects/…-atlas-ms-atlas/`. Nothing is estimated unless labelled
so.

---

## 1. Executive summary

Across 17 sessions and 196 subagents, task-232 consumed **1,259M billed input
tokens** (270M main-thread / 988M subagent) over **10,574 turns**. Tool results
contributed **20.36 MB** of raw text. That sounds small — until it is weighted by
how long each result is carried:

> **Total carry = 0.92 GB-turns.** At ~4 bytes/token that is **≈230M tokens,
> ≈18% of the task's entire billed input**, spent re-reading tool output the
> session had already finished with.

Three measured facts drive the whole report:

1. **Pollution is front-loaded.** The median >12 KB tool result lands at position
   **0.10** of its stream, and **75% of them land in the first quarter**. Large
   results are not a tail-end annoyance; they are inserted precisely where the
   multiplier is highest.
2. **Two reference documents cost more than all of `processor.go`.**
   `service-wiring-recipe.md` (26 reads) and `query-scope-audit.md` (48 reads)
   together carry **69.3 MB-turns — 7.5% of all carry — from 74 of 11,102 tool
   calls.** `processor.go`, read 155 times, costs 21.5 MB-turns.
3. **The implementer return contract is already right; the review return contract
   is not.** 101 `atlas-implementer` returns average **1,145 B** and 101/101 cite
   a commit sha. 84 `general-purpose` review returns average **4,904 B** (p90
   8,162 B, max 14,459 B), **0 of 84 wrote a durable artifact**, and only **26 of
   84** carry a verdict in their first 80 characters.

Investigation A (deterministic discovery) is real but **smaller than expected and
partly self-inflicted by non-adoption**: the repo already ships
`tools/task-resolve.sh` and `tools/task-brief.sh`, used in only **16 and 12 of 213
streams** respectively. The opportunity is adoption plus one new fact-block
script, not a new tooling layer.

Investigation C is the largest and most tractable. Investigation B is the
cleanest fix. Investigation A is worth doing but should be scoped modestly.

---

## 2. Audited task and evidence available

### Why task-232 is representative

| Criterion | task-232 |
|---|---|
| Four-phase flow | ✅ `/spec-task` → `/design-task` → `/plan-task` → `/execute-task` |
| Controller + implementer | ✅ 13 execute sessions, 101 implementer dispatches |
| Multiple implementation units | ✅ **56 plan tasks**, `plan.md` is 291 KB / 7,592 lines |
| Verifier / reviewer agents | ✅ 84 `general-purpose` reviews, 5 `plan-adherence-reviewer`, 4 `backend-guidelines-reviewer`, 1 `atlas-verifier` |
| Continuation / handoff | ✅ 17 sessions; PARTIAL/continuation splits throughout |
| Repository investigation | ✅ 40+ services touched, `query-scope-audit.md` is 126 KB of findings |
| Build / test / verification | ✅ `tools/verify.sh` invoked 325×, `tools/lint.sh` 114× |
| PR completion | ⚠️ branch complete and clean; **not yet merged to `main`** |

It is the largest agentic task in the repository's history by billed input, and
it is *sweep-shaped* — the dominant failure modes it exposes (templated repetition
across many services, shared reference documents, per-unit review) recur in every
Atlas migration task. That generality is the reason to audit it rather than a
smaller, tidier task.

### Evidence inventory

| Evidence | Location | Quality |
|---|---|---|
| 17 session transcripts | `<CLAUDE_HOME>/projects/…-atlas-ms-atlas/<sid>.jsonl` | **Direct.** Full tool calls, inputs, results, per-message `usage` |
| 196 subagent transcripts | `…/<sid>/subagents/agent-*.jsonl` (+ `.meta.json`) | **Direct.** Same fidelity |
| Offloaded large tool results | `…/<sid>/tool-results/*.txt` | Direct |
| Prior session audit | `<CLAUDE_HOME>/audits/session-task-232-three-largest.md` | Direct, independently reproduced below |
| Task artifacts | `docs/tasks/task-232-sparse-ephemeral-environments/` (17 files, 691 KB) | Direct |
| Git history | 55+ commits on `task-232-sparse-ephemeral-environments` | Direct |

**Session set.** Selected by `cwd` pointing inside the task worktree — 17
sessions, 196 subagents, matching the prior audit's counts exactly. (The prior
audit's 1,294M also included design/plan sessions `c9c462bf` and `0c353f15` whose
`cwd` was the main repo; hence 1,259M here. Same task, slightly narrower frame.)

### Independent reproduction of the headline figures

| Session | main-in | turns | peak ctx | sub-in | sub turns |
|---|---:|---:|---:|---:|---:|
| `9178dc81` (design) | 7.8M | 65 | 182k | — | — |
| `b3ac7967` (plan) | 26.0M | 107 | **370k** | — | — |
| `95966fe5` (exec) | 27.1M | 142 | 319k | 128.8M | 1,227 |
| `66df7cac` | 10.1M | 76 | 199k | 36.4M | 356 |
| `1563928e` | 13.0M | 99 | 209k | 53.8M | 560 |
| `0a47fb05` | 20.4M | 132 | 257k | 85.4M | 737 |
| `b1279ea1` | 15.1M | 97 | 233k | 42.8M | 364 |
| `3c4a2d8c` | 10.0M | 78 | 204k | 35.7M | 333 |
| `c63bbaa3` | 8.4M | 73 | 169k | 24.2M | 283 |
| `e2b4027c` | 22.2M | 149 | 242k | 133.3M | 1,107 |
| `854e6e87` | 27.9M | 164 | 272k | 103.6M | 922 |
| `accd84a6` | 20.6M | 127 | 255k | 120.3M | 1,036 |
| `3aa364a5` | 13.2M | 103 | 201k | 65.6M | 644 |
| `50d04935` | 21.9M | 129 | 282k | 91.6M | 682 |
| `3a494277` | 24.5M | 149 | 266k | 67.0M | 598 |
| `9cf46bb7` | 1.0M | 17 | 64k | — | — |
| `f445982a` | 1.0M | 18 | 65k | — | — |
| **total** | **270M** | **1,725** | | **988M** | **8,849** |

*Method: dedup assistant messages by `message.id`, sum
`cache_read + cache_creation + input`.* This confirms the prior audit's split
(subagents ≈ 78% of billed input; 84% of turns) from an independent
reconstruction.

---

## 3. Execution reconstruction

**Facts (directly observable).**

| Measure | Value |
|---|---|
| Sessions / subagents | 17 / 196 |
| Tool calls | 1,484 main + 9,618 subagent = **11,102** |
| Tool-result bytes | 1.78 MB main + 18.58 MB subagent = **20.36 MB** |
| Bash calls | 7,064 (64% of all calls) |
| Agent dispatches | 183 (`Agent` tool), 270 completion notifications (incl. `SendMessage` continuations) |
| Agent type mix | `atlas-implementer` 101 · `general-purpose` 84 · `plan-adherence-reviewer` 5 · `backend-guidelines-reviewer` 4 · `Explore` 1 · `atlas-verifier` 1 |
| Model mix | `sonnet` 181 · `haiku` 2 (all dispatches carried an explicit `model`) |
| Durable reports written by agents | **13 of 196** |
| `progress.md` writes by controllers | 248 (median 2,056 B, 545.6 KB total) |
| Plan size | `plan.md` 291 KB / 56 tasks |
| Final state | 55+ commits, branch clean, not yet merged |

**Reasonable inference.** The shape is: one long plan-authoring session
(`b3ac7967`, no subagents, 370k peak) followed by 13 execute sessions each running
a dispatch → gate → review → reconcile loop over 2–6 plan tasks, then two short
wrap-up sessions. Session boundaries correspond to context exhaustion rather than
work boundaries — the prior audit measured that **all 17 sessions ended at their
own peak context**, which this reconstruction is consistent with.

**Not recoverable from retained telemetry.**

- Per-turn context *composition* (how much of a 250k window was prefix vs. tool
  results vs. conversation) — inferable only by modelling, not measured.
- Wall-clock cost of background/async agent scheduling.
- Whether a given review finding changed the code, unless it produced a commit.

---

## 4. Investigation A — deterministic discovery

### A.1 The orientation prefix, measured

Defining the *orientation prefix* as every tool call a stream makes before its
first `Edit`/`Write`/`Agent`:

| | n | median prefix | p90 | total calls | prefix result bytes |
|---|---:|---:|---:|---:|---:|
| main threads | 17 | 12 | 22 | 291 | 0.79 MB |
| subagents | 195 | 18 | 37 | **3,915** | **13.48 MB** |

By agent type:

| Agent type | n | median prefix calls | prefix as % of its calls | prefix bytes |
|---|---:|---:|---:|---:|
| `atlas-implementer` | 101 | 13 | **23.8%** | 5.97 MB |
| `general-purpose` | 83 | 21 | 84.9%¹ | 6.19 MB |
| `plan-adherence-reviewer` | 5 | 27 | 96.0%¹ | 0.61 MB |
| `backend-guidelines-reviewer` | 4 | 40 | 92.1%¹ | 0.52 MB |
| `atlas-verifier` | 1 | 6 | 100%¹ | 0.01 MB |

¹ For read-only agents the "prefix" is essentially the whole agent — they read,
then write one report. **This is not waste for reviewers; the metric is only
meaningful for implementers.** Read the 23.8% row as the real signal:
`atlas-implementer` spends roughly a quarter of its tool budget before touching
code, at a median of 13 calls.

### A.2 What those calls actually establish

Classifying all 7,064 Bash calls by regex family:

| Family | calls | result bytes | streams |
|---|---:|---:|---:|
| read slice (`sed -n`/`head`/`tail`) | 3,158 | 5.35 MB | 209 |
| other | 2,490 | 4.77 MB | 207 |
| build/test (`go build`, `verify.sh`, `lint.sh`) | 1,251 | 0.81 MB | 165 |
| **worktree/branch state** | **649** | 0.44 MB | 161 |
| **changed-file set** (`git diff --name-only`, `status --short`) | **457** | 0.38 MB | 137 |
| **file enumeration** (`find`, `ls -R`, `git ls-files`) | **429** | 0.62 MB | 138 |
| **existence-check grep** (`-l`/`-q`/`-c`) | **380** | 0.36 MB | 120 |
| **commit/log inspection** | **276** | 0.27 MB | 125 |
| go module/package enumeration | 81 | 0.05 MB | 53 |
| task-artifact location | 68 | 0.06 MB | 46 |

The five bolded families are **2,191 calls / 2.07 MB establishing purely
mechanical repository facts** — which branch, which worktree, which files changed
since the base, which packages exist, does symbol X occur anywhere. No
interpretation is involved in producing any of them; interpretation begins only
once the fact is in hand.

### A.3 Concrete instances

**A verbatim implementer prefix** (`agent-a4e5d9ca06238e0cd`, session `1563928e`,
24 calls before first edit) — reading one DDD package file by file:

```
#1   3,150B  Read .superpowers/sdd/plan/task-12-brief.md
#2     678B  Read services/atlas-tenants/…/tenant/entity.go
#3   1,352B  Read services/atlas-tenants/…/tenant/model.go
#4   2,046B  Read services/atlas-tenants/…/tenant/builder.go
#5     262B  ls
#6   1,248B  Read services/atlas-tenants/…/tenant/rest.go
#7   2,569B  Read services/atlas-tenants/…/tenant/kafka.go
#8   8,520B  Read services/atlas-tenants/…/tenant/processor.go
#9  11,291B  Read services/atlas-tenants/…/tenant/processor_test.go
#10  1,698B  Read services/atlas-tenants/…/tenant/entity_builder.go
#11    750B  Read services/atlas-tenants/…/tenant/provider.go
#12  2,928B  Read services/atlas-tenants/…/tenants/main.go
#13  1,419B  Read services/atlas-tenants/…/tenants/test/database.go
```

The *file list* is mechanical — Atlas DDD packages have a fixed shape
(`entity/model/builder/rest/kafka/processor/provider/administrator/requests`).
The *contents* are not; the agent genuinely needs to read them. So the
deterministic win here is small (it would save the `ls` and a little sequencing
thought, not the reads).

**A second prefix** (`agent-aec35d01f4abfd73c`, session `50d04935`) is a better
target — it contains three toolchain capability probes:

```
#9     209B  awk --version 2>&1 | head -3; which awk; ls -la $(which awk)
#10  4,816B  docker manifest inspect nginx:latest … && echo has-docker || echo no-net
#13     17B  sed --version 2>&1 | head -1
```

Across the chain: **~65 toolchain-availability probes** (`command -v` 18,
`--version` 13, `which kustomize` 11, `which shellcheck` 7, `which bats` 4,
`which kubectl`/`docker`/`staticcheck`/`golangci`/`yq`, `go version`) spread over
**80 of 213 streams**, 0.12 MB. Individually trivial; collectively a fact the
environment could simply state once.

### A.4 The real finding: existing tooling is under-adopted

Atlas already ships 63 scripts under `tools/`, including exactly the resolvers
this investigation would otherwise propose:

| Existing tool | calls | distinct streams (of 213) |
|---|---:|---:|
| `tools/verify.sh` | 325 | 48 |
| `tools/lint.sh` | 114 | 41 |
| `tools/task-brief.sh` | 63 | **12** |
| `tools/mode-select.sh` | 58 | 11 |
| `tools/scope-guard.sh` | 55 | 17 |
| `tools/task-resolve.sh` | 19 | **16** |

`tools/task-resolve.sh` — whose entire purpose is turning a fuzzy task identifier
into a worktree path and branch — was invoked in **16 of 213 streams**. Meanwhile
649 `git worktree list`/`git status`/`git branch`/`git rev-parse` calls were
issued to derive the same facts by hand, and **1,606 Bash calls across 79 streams
carried a literal `cd /home/…/.worktrees/task-232-sparse-ephemeral-environments &&`
prefix** — an 86-character absolute path re-typed on every call because cwd was
never established once.

**This reframes Investigation A.** The gap is not "Atlas lacks deterministic
tooling." It is that the tooling exists, is not surfaced at the point of use
(agent briefs / agent definitions), and produces prose rather than a parseable
fact block that an agent can consume without a follow-up call.

### A.5 Per-candidate assessment

| # | What was being determined | Calls / investigation | Interpretation needed? | Deterministic mechanism | Compact output contract | Benefit | Risk |
|---|---|---|---|---|---|---|---|
| A1 | Worktree path, branch, base sha | 649 calls, 161 streams | **No** | `tools/task-resolve.sh`, already exists; emit into the brief | `worktree=…` `branch=…` `base=<sha>` | **Medium** — removes ~3 calls × 213 streams and the `cd` prefix tax | Very low; stale value if the controller rebases — mitigate by regenerating per dispatch |
| A2 | Which files/services changed since base | 457 calls, 137 streams | **No** | `git diff --name-only $BASE` behind a script | `changed_services=…` `changed_packages=N` | **Medium** | Low |
| A3 | Which packages/modules exist; enumerate files | 429 + 81 calls | **No** | `git ls-files` / `go list ./...` in the fact block | `modules=…` | Medium | Low — but do **not** pre-enumerate *contents*; agents must still read code |
| A4 | Does symbol/pattern X exist anywhere | 380 existence greps | **No** for existence; **yes** for what it means | Keep as agent work; fix the ergonomics (see A6) | — | Low | Replacing with a script risks answering the wrong question |
| A5 | Which guards/verification gates apply | folded into 1,251 build/test calls | **Partly** | Derive from changed paths: `guards=redis-key,scope,env-bootstrap` | `applicable_guards=…` | **Medium** — prevents both over- and under-running gates | Medium: a wrong mapping silently skips a gate. Must fail *open* (list all guards when unsure) |
| A6 | — (defect) | 113 `zsh: no matches found` errors, 1.0% of 11,102 calls | n/a | Quote globs: `--include='*.go'`; prefer `git ls-files` | — | Low but free | None. *Confirms the prior audit's finding on the full chain (it measured 60 over 5 sessions).* |
| A7 | Is tool T installed | ~65 probes, 80 streams | **No** | One line in the environment/agent preamble | `toolchain=go,docker,kubectl,yq,shellcheck,bats` | Low | Low; must be accurate or agents will assume a missing tool |

**Explicitly not recommended for automation.** The read-slice family (3,158 calls,
the single largest) is agents *choosing what to look at* — that is semantic work
and it is being done well (see §8). Package-content reads are semantic. Existence
greps are semantic in their framing even when mechanical in execution.

**Target output shape** (what a `tools/task-context.sh <task>` should print):

```
task=task-232-sparse-ephemeral-environments
worktree=.worktrees/task-232-sparse-ephemeral-environments
branch=task-232-sparse-ephemeral-environments
base=c8d44127cbb9e
changed_services=atlas-tenants,atlas-configurations
changed_packages=7
changed_files=23
rest_surface=true
kafka_surface=true
db_surface=false
new_service=false
applicable_guards=redis-key,scope,env-bootstrap,outbox
backend_audit_families=DOM,FILE,SUB
toolchain=go1.24,docker,kubectl,yq,shellcheck
```

Under 400 bytes. Compare against the ~2.07 MB and 2,191 calls currently spent
reaching a subset of the same conclusions.

**Honest sizing.** 2.07 MB of raw result bytes is ~10% of all tool bytes, and its
carry is modest because these results are small and evenly spread. The direct
saving is on the order of **1–2% of billed input**. The *indirect* value is larger
and harder to measure: the fact block also removes the ~13-call orientation
latency before an implementer's first edit, and A5 removes an entire class of
"which gate applies" reasoning. **Rank A below C and B.**

---

## 5. Investigation B — subagent → controller communication

### B.1 Return payload sizes

| Agent type | n | median | mean | p90 | max | total |
|---|---:|---:|---:|---:|---:|---:|
| `atlas-implementer` | 101 | **1,145 B** | 1,173 B | 1,643 B | 2,849 B | 115.7 KB |
| `general-purpose` | 84 | **4,904 B** | 5,097 B | 8,162 B | 14,459 B | **418.1 KB** |
| `backend-guidelines-reviewer` | 4 | 2,612 B | 2,592 B | 2,846 B | 2,944 B | 10.1 KB |
| `plan-adherence-reviewer` | 5 | 2,133 B | 2,194 B | 2,207 B | 2,561 B | 10.7 KB |
| `Explore` | 1 | 3,296 B | — | — | 3,296 B | 3.2 KB |
| `atlas-verifier` | 1 | **478 B** | — | — | 478 B | 0.5 KB |
| **total** | **196** | | | | | **558.3 KB** |

**`atlas-implementer` is already at contract.** 1.1 KB median, **101 of 101
mention a commit sha**, only 3 of 101 contain a fenced code block. Whatever
tightening happens elsewhere, this contract is working and should be left alone.

**`atlas-verifier` is the exemplar of the whole chain**: 6 tool calls, a 478-byte
return, and the build/lint output never left its context. One dispatch is too
small a sample to generalise from, but it demonstrates the target shape exists and
is achievable.

**`general-purpose` reviews are the outlier** — 3.6× the implementer's median, at
4.3× the volume (84 dispatches). They are 75% of all return bytes in the chain.

### B.2 Duplication: reviews leave no durable artifact

| | count |
|---|---:|
| Agents writing any `docs/tasks/**` artifact | **13 of 196** |
| — `plan-adherence-reviewer` | 5 |
| — `backend-guidelines-reviewer` | 4 |
| — `atlas-implementer` | 4 |
| — **`general-purpose`** | **0 of 84** |

So the 418 KB of review findings exists in exactly two places: the controller's
context (permanently, for the rest of that session) and whatever the controller
chose to hand-copy into `progress.md`. There is **no durable review artifact** for
any of the 84 reviews. If a session ends, the reasoning behind a review verdict is
recoverable only by re-reading the transcript.

Then the controller re-encodes it: **248 `progress.md` writes, median 2,056 B,
545.6 KB total**. A review finding is therefore paid for three times — the agent
generates it, the controller ingests it, the controller re-writes a summary of it.

### B.3 What the controller actually ingested

Agents ran asynchronously, so the `Agent` tool result is a 1,064-byte launch stub;
the real payload arrives later as a `<task-notification>`.

| | value |
|---|---:|
| Notifications (deduped by task-id) | **270** |
| Total bytes | **725.5 KB** |
| Mean / max | 2,751 B / 15,588 B |
| Median controller turns remaining after ingest | **51** |
| **Carry** | **40.3 MB-turns** ≈ 10M tokens |

The largest single ingest: an 11.0 KB notification in `854e6e87` carried across 95
subsequent controller turns = **1.07 MB-turns from one agent return**.

### B.4 Verdict burial

Only **26 of 84** review returns carry a verdict token (`VERDICT`/`PASS`/`FAIL`/
`✅`/`❌`/`Critical`) within their first 80 characters. The other 58 open with
narration:

```
| That's a false-positive match (`.worktrees/task-xxx` contains "xxx" as part …
| Confirmed — `atlas-kites` is nowhere in Task 1's or Task 3's scope either. …
| All matches. Now I have everything needed to write the final report.  ## PART A
```

The controller cannot short-circuit; it must ingest the full 4.9 KB to learn
whether anything is wrong. In the median case nothing is.

**A representative return** (`agent-a64868a95292e8c3c`, 4,865 B) contains, in
order: a false-positive dismissal, a "## Spec verdict" heading, a **17-line pasted
shell transcript** of two `redis-key-guard.sh` runs with their full stdout, a
description of a regression test the agent wrote and ran, a **5-line pasted test
failure block**, then a file-by-file source walk. Every fact in it is either (a)
already in the commit, (b) already in the gate log the controller ran itself, or
(c) a judgement that compresses to one line. Measured across all 93 review-ish
agents: only **10.5 KB of 438.9 KB (2.4%)** is inside fenced blocks — **the bulk
is prose reasoning, not evidence.**

### B.5 The reverse direction is load-bearing — do not cut it

| | n | median | mean | p90 | total |
|---|---:|---:|---:|---:|---:|
| Dispatch prompts (controller → agent) | 183 | 4,068 B | 4,041 B | 5,582 B | 722.2 KB |

These briefs are *why* implementers reach their first edit in 13 calls. The
largest (7,228 B, "Review Task 13") is still smaller than the largest review
return it produced. **Shrinking briefs would move cost into Investigation A, not
remove it.** See §8.

### B.6 Recommended return contracts

Derived from what the controllers demonstrably did next with each return.

**Implementer — keep as-is.** Already ~1.1 KB with a commit sha. Formalise only:

```
status: COMPLETE | PARTIAL | BLOCKED
commit: <sha>
files: <n> changed
verification: go build ok | go test ok (<n> pkgs)
residue: <one line, or none>          # PARTIAL only: what the continuation must do
```

**Reviewer — this is the change.** The controller's observed next action after a
review was always one of: mark the unit done in `progress.md`; dispatch a fix
agent; or re-gate. It needs the verdict and the actionable defects, nothing else.

```
verdict: PASS | FAIL | PASS_WITH_NOTES
artifact: docs/tasks/<task>/reviews/<unit>.md     # full reasoning lives here
critical: <n>
important: <n>
minor: <n>
blocking: <one sentence per critical/important finding, file:line>
```

Target ≤ 600 B for a clean review (vs. 4,904 B median today) with the full 5 KB
of reasoning written to a durable file the controller does not read unless
`verdict != PASS`.

**Verifier — already correct** at 478 B. Adopt its shape as the template.

**Continuation handoff — leave alone.** The prior audit measured continuation
re-orientation at ~18 KB / ≈1% of a continuation's spend, and this audit's
implementer-prefix data (median 13 calls) is consistent. It works.

### B.7 Did a compact contract have lost anything?

I checked the controller-side consequences. Two cases where a review return
mattered beyond its verdict:

- `854e6e87` #104–#106: a review surfaced an `analyzer_guard_hash` cache-key bug
  (a false-PASS hazard in the *verification tooling itself*). That is a
  `verdict: FAIL` + one `blocking:` line — **fully preserved** by the contract.
- `95966fe5` #54: the controller's own narrow-grep-root error was diagnosed partly
  from review detail. Under the contract this becomes a `FAIL` plus an artifact
  path; the controller would read the artifact. **One extra read, ~5 KB, in the
  rare case — against 84 unconditional 4.9 KB ingests.**

I found **no case** where a controller re-used review prose more than one turn
after ingesting it. The information was consumed immediately and then carried for
a median of 51 further turns as dead weight.

---

## 6. Investigation C — tool-result / context pollution

### C.1 Size distribution

| | n | total | median | p90 | p99 | max | top 1% share | top 5% share |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| main | 1,484 | 1.78 MB | 380 B | 3.0 KB | 10.6 KB | 50.5 KB | 16.6% | 38.0% |
| subagent | 9,618 | 18.58 MB | 475 B | 5.0 KB | 19.7 KB | 60.1 KB | 15.0% | 39.9% |

**Typical tool use is disciplined** — a 400–500 byte median across 11,102 calls is
good hygiene, and confirms the prior audit's conclusion that turn-level tool
misuse is not the cost driver. But the distribution has a heavy tail:
**268 results >12 KB carry 5.45 MB — 26.8% of all tool bytes from 2.4% of calls.**

### C.2 The largest results

| size | stream | call |
|---:|---|---|
| 58.6 KB | `plan-adherence-reviewer` | `Read docs/tasks/…/plan.md` |
| 57.0 KB | `atlas-implementer` | `Read docs/tasks/…/query-scope-audit.md` |
| 56.5 KB | `atlas-implementer` | `Read …/query-scope-audit.md` |
| 53.7 KB | `general-purpose` | `Read …/query-scope-audit.md` |
| 53.7 KB | `atlas-implementer` | `Read …/query-scope-audit.md` |
| 53.2 KB | `general-purpose` | `Read .superpowers/sdd/plan/review-3132fa29e..029a80667.diff` |
| 50.5 KB | `atlas-implementer` | `Read …/query-scope-audit.md` |
| 49.3 KB | main `b3ac7967` | `Read docs/tasks/…/design.md` |
| 48.4 KB | `backend-guidelines-reviewer` | `Read …/tool-results/bd3sc8ctl.txt` |
| 43.3 KB | main `b3ac7967` | `Read docs/tasks/…/prd.md` |

**Nine of the top ten are whole-file reads of task artifacts, not source code.**
The pollution surface is documentation and diffs, not the codebase.

### C.3 Repeated reads within one stream

75 stream×file pairs were read ≥3 times; ~1.13 MB is redundant copies. Worst
cases:

| bytes | reads | stream | file |
|---:|---:|---|---|
| 93.5 KB | **13×** | `plan-adherence-reviewer` | `plan.md` |
| 89.4 KB | 5× | `atlas-implementer` | `query-scope-audit.md` |
| 78.7 KB | 7× | `atlas-implementer` | `query-scope-audit.md` |
| 78.4 KB | 5× | `plan-adherence-reviewer` | `plan.md` |
| 31.0 KB | 7× | `atlas-implementer` | `analyzer.go` |
| 27.4 KB | 8× | `atlas-implementer` | `pr-validation.yml` |

A `plan-adherence-reviewer` reading a 291 KB `plan.md` thirteen times in one agent
is the sharpest single instance. It is also *understandable* — the agent must
check 56 tasks against the tree and the file does not fit — which is precisely why
it needs a per-task accessor rather than an instruction to "read less."

### C.4 Cross-agent re-reads of shared reference documents

| reads | streams | bytes | document |
|---:|---:|---:|---|
| 48 | 16 | 0.688 MB | `query-scope-audit.md` (126 KB on disk) |
| 34 | 9 | 0.346 MB | `plan.md` (291 KB) |
| 26 | 25 | 0.543 MB | `service-wiring-recipe.md` (23 KB) |
| 12 | 7 | 0.082 MB | `design.md` (46 KB) |

`service-wiring-recipe.md` is read by **25 distinct streams** — essentially every
implementer in the sweep — nearly always in full, ~22–24 KB at a time. That is
correct behaviour against the current interface (it is the authoritative recipe)
and it is exactly what makes it the most expensive object in the task once carry
is applied.

### C.5 Position: pollution is front-loaded

> **Median position of a >12 KB result: 0.10 of its stream. 75% land in the first
> 25%.**

This is the most consequential measurement in the report. A large result inserted
at 10% of an agent's life is re-billed on ~90% of that agent's turns. The heavy
tail is not merely large — it is optimally placed to be maximally expensive.

### C.6 Per-example assessment

| Tool/command | Why called | Magnitude | Portion actually needed | Better interface | Belongs in |
|---|---|---:|---|---|---|
| `Read query-scope-audit.md` | Learn this service's query-scope findings | 48 reads, 0.69 MB, 31.8 MB-turns | The **rows for the 1–8 services in this brief** — a few hundred bytes | Generate a per-batch extract at dispatch time, or a `--service` accessor | Deterministic helper + brief generation |
| `Read service-wiring-recipe.md` | Follow the wiring recipe | 26 reads, 0.54 MB, **37.5 MB-turns** | The **one Pattern (A/B/C/D)** this batch uses | Split into `recipe/pattern-a.md`… and name the pattern in the brief | Doc structure + brief generation |
| `Read plan.md` (13× in one agent) | Audit 56 tasks | 93.5 KB in one stream | The **one task** under audit | `tools/task-brief.sh`-style per-task extractor for reviewers | Agent instructions + helper |
| `Read …/review-*.diff` (53.2 KB, 39.7 KB, 34.3 KB) | Review a commit range | 3 results >30 KB | `--stat` first, then hunks for flagged files | `git diff --stat` → targeted `git diff -- <file>` | Agent instructions (reviewers) |
| `Read …/tool-results/*.txt` (48.4, 43.6, 42.7, 41.7, 41.4 KB) | Re-read an offloaded large result | 5 results >40 KB | A slice | Grep/sed the offload file rather than `Read` it whole | Agent instructions |
| `cat deploy/shared/routes.conf` | Understand routing table | 19.2 KB at call #5 of 38 | The 3–5 relevant `proxy_pass` lines (the agent did exactly this at #12) | Do the targeted grep first | Agent instructions |
| Unquoted `--include=*.go` under zsh | Search Go files | 113 failures / 11,102 calls | — | `--include='*.go'` or `git ls-files` | Guidelines (already specced in task-234 FR-1.4) |
| `go build ./... 2>&1 \| head -60` etc. | Module-local verification | 1,251 calls, 0.81 MB total | Already bounded | **No change — this is correct** | — |

Note the last row: build/test output is **already** consistently piped through
`head`/`tail`. 1,251 invocations for 0.81 MB is ~650 bytes each. This is a
discipline that is working and should be cited as the model for the doc reads.

---

## 7. Multiplicative-cost observations

Using **carry = result bytes × turns remaining in that stream**:

| | carry |
|---|---:|
| All tool results | **0.92 GB-turns** ≈ 230M tokens ≈ **18% of the task's 1,259M** |
| — `Read` | 440 MB-turns |
| — `Bash` | 449 MB-turns |
| Agent → controller notifications | 40.3 MB-turns ≈ 10M tokens |

*(Token figures assume ~4 bytes/token and are for ranking, not billing.)*

**Top carry by read target:**

| carry | reads | file |
|---:|---:|---|
| **37.5 MB-turns** | 26 | `service-wiring-recipe.md` |
| **31.8 MB-turns** | 48 | `query-scope-audit.md` |
| 21.5 MB-turns | 155 | `processor.go` |
| 17.8 MB-turns | 72 | `registry.go` |
| 13.7 MB-turns | 53 | `processor_test.go` |
| 12.7 MB-turns | 40 | `analyzer.go` |
| 12.0 MB-turns | 62 | `main.go` |
| 8.9 MB-turns | 34 | `plan.md` |
| **7.7 MB-turns** | **3** | `prd.md` |
| 6.4 MB-turns | 105 | `requests.go` |

**The ranking inverts the call counts.** `requests.go` was read 105 times and
costs 6.4 MB-turns. `prd.md` was read **three** times and costs 7.7 MB-turns —
because two of those reads were at turns 4–5 of the 107-turn plan-authoring
session `b3ac7967` and one at turn ~6 of `9178dc81`. Similarly `service-wiring-recipe.md`
outranks `processor.go` 37.5 : 21.5 despite 26 reads against 155.

**Top single carries:**

| carry | = size × turns | stream |
|---:|---|---|
| 6.7 MB-t | 53.7 KB × 122 turns | implementer reading `query-scope-audit.md` |
| 5.1 MB-t | 49.3 KB × 101 turns | `b3ac7967` reading `design.md` |
| 5.1 MB-t | 41.4 KB × 120 turns | implementer reading an offloaded result |
| 4.5 MB-t | 43.3 KB × 102 turns | `b3ac7967` reading `prd.md` |
| 3.6 MB-t | 23.9 KB × 147 turns | implementer reading `task-19-report.md` |
| ~3.0 MB-t each (×8) | ~23 KB × 110–142 turns | implementers reading `service-wiring-recipe.md` |

**Ranking rule that falls out of this data:** a fix that removes 20 KB from an
agent's *first ten calls* is worth roughly ten times the same 20 KB removed near
the end. Since 75% of >12 KB results land in the first quarter of their stream,
the entire heavy tail is in the high-multiplier zone.

**The two-document finding, stated plainly:** `service-wiring-recipe.md` +
`query-scope-audit.md` = **69.3 MB-turns, 7.5% of all carry (≈17M tokens, ≈1.4%
of the whole task), from 74 of 11,102 tool calls.** Two files. Slicing them is the
single highest ratio of benefit to effort in this report.

---

## 8. Things that should remain unchanged

Explicitly guarded, because each looks expensive and is not.

1. **Dispatch briefs (183 × ~4 KB = 722 KB).** These are why implementers reach
   first edit in a median of 13 calls, and why continuation agents re-orient in
   ~18 KB. Cutting them converts Investigation-B savings into Investigation-A
   costs at an unfavourable rate. Leave them; if anything the brief should grow
   by the ~400-byte fact block from §4.

2. **`atlas-implementer` return contract (1,145 B median, 101/101 cite a commit).**
   Already at target. Formalise, do not shrink.

3. **The 120-call PARTIAL/continuation split.** Prior audit measured
   re-orientation at ≈1% of a continuation's spend; this audit's median 13-call
   implementer prefix is consistent. It works.

4. **The verification split.** `atlas-verifier` returned 478 B from 6 tool calls
   and kept the entire build/lint/vet output out of the implementer's window.
   1,251 build/test invocations across the chain produced only 0.81 MB in-context
   because they are consistently bounded by `head`/`tail`. This is the discipline
   the doc reads should copy, not a cost to trim.

5. **Model discipline.** 183 dispatches, **100% carried an explicit `model`**,
   181 `sonnet` + 2 `haiku`, zero Opus leakage. Nothing to fix.

6. **`progress.md` as the durable handoff artifact.** 248 writes / 545.6 KB is
   real spend, but it is what makes a fresh controller resumable in ~14 calls. The
   duplication problem in §5 is that reviews *also* live only here — the fix is to
   give reviews their own artifact, not to write less progress.

7. **Targeted read slices (3,158 `sed -n`/`head`/`tail` calls).** The single
   largest Bash family, and it is agents choosing what to look at. `854e6e87`
   navigating an 8,929-line `progress.md` entirely by slice is the behaviour to
   propagate, not to police.

8. **`query-scope-audit.md` and `service-wiring-recipe.md` as documents.** They
   are named as the two costliest objects in the task, and they are also the
   reason 40+ services were wired consistently. **The recommendation is to change
   the *access pattern*, never to delete or thin the content.** A thinner recipe
   would be re-derived by 25 agents at far greater cost.

---

## 9. Prioritized opportunities

| # | Opportunity | Category | Evidence from task-232 | Expected impact | Effort | Correctness risk | Recommendation |
|---|---|---|---|---|---|---|---|
| 1 | **Slice the two hot reference docs** — per-batch extract or section accessor for `service-wiring-recipe.md` / `query-scope-audit.md` | C + multiplicative | 74 reads → **69.3 MB-turns, 7.5% of all carry**; recipe read by 25 of 213 streams, nearly always whole | **High** | Low | **Low** — content unchanged, only the slice is passed; brief names the pattern | **Do first** |
| 2 | **Compact reviewer return contract + durable review artifact** | B | 84 reviews, median 4,904 B, **0/84 wrote an artifact**, only 26/84 verdict-first, 40.3 MB-turns of notification carry | **High** | Low | **Low–Medium** — full reasoning is preserved in the artifact; risk is a controller not reading it on `FAIL` (mitigate: contract requires reading the artifact when `verdict != PASS`) | **Do first** |
| 3 | **No large whole-file `Read` early in an agent's life** — instruction + reviewer diff discipline (`--stat` then targeted hunks) | C | 268 results >12 KB = 26.8% of tool bytes; **median position 0.10, 75% in first quarter**; three review diffs >34 KB | **High** | Low | **Medium** — an agent that under-reads may miss context. Frame as "slice first, escalate to full read if the slice is insufficient," never as a hard ban | Do |
| 4 | **`tools/task-context.sh` fact block, injected into every brief** | A | 2,191 calls / 2.07 MB on mechanical facts; `task-resolve.sh` used in only 16 of 213 streams; 1,606 calls carry an 86-char `cd` prefix | **Medium** | Medium | **Medium** on the `applicable_guards` field — must fail open (list all guards when the mapping is uncertain) | Do |
| 5 | **Per-agent telemetry line** (type, model, turns, tool calls, result bytes, return bytes) | measurement | This audit needed ~250 lines of ad-hoc transcript parsing to establish facts a 200-byte ledger line would carry | **Medium** (enabling) | Low | None | Do |
| 6 | **Quote shell globs / prefer `git ls-files`** | A | 113 `zsh: no matches found` = 1.0% of 11,102 calls | **Low** | Trivial | None | Already specced (task-234 FR-1.4) — verify it landed |
| 7 | **Emit toolchain availability once** rather than probing | A | ~65 probes across 80 of 213 streams, 0.12 MB | **Low** | Low | Low — stale/wrong list is worse than probing | Fold into #4, do not do separately |

**Already addressed by task-234 — do not re-propose:** per-agent `tools:`
restriction (**landed** — 9 agent definitions now declare `tools:`), codemod
tooling for templated sweeps, controller context ceiling, review-agent *count*
right-sizing, prefix trim. Opportunity #2 above is about review *return format*
and is complementary to FR-4's review *volume*.

---

## 10. Recommended implementation tasks

### Task A — Per-batch slices for hot reference documents

- **Problem.** `service-wiring-recipe.md` (26 reads, 25 streams) and
  `query-scope-audit.md` (48 reads, 16 streams) are read whole by nearly every
  agent in the sweep and cost **69.3 MB-turns — 7.5% of all carry — from 74 of
  11,102 tool calls.** Each agent needs one Pattern section or a handful of
  service rows.
- **Mechanism.** (a) Split the recipe into `recipe/pattern-{a,b,c,d}.md` plus a
  short index, and have the brief name the pattern. (b) Add a
  `--service <name>…` accessor for audit-style documents that emits only the
  matching rows. (c) Brief generation embeds the slice, or its path, instead of
  the parent document.
- **Files/workflows.** `docs/tasks/<task>/service-wiring-recipe.md` (structure);
  brief generation in `.claude/commands/execute-task.md`; possibly
  `tools/task-brief.sh`.
- **Expected benefit.** Reduce those 74 reads from ~22–53 KB to ~2–4 KB. Carry
  drops from 69.3 to ~8 MB-turns; ≈1.2% of a task-232-sized task, concentrated in
  the highest-multiplier position.
- **Measure.** Re-run the carry analysis on the next sweep task: no `.md` should
  appear in the top-5 carry table.
- **Failure modes.** A slice that omits a cross-cutting caveat causes a wrong
  implementation and a fix round — which costs far more than the saving. Mitigate:
  every slice carries the document's short "invariants" preamble verbatim, and
  names the parent path so an agent can escalate.

### Task B — Reviewer return contract + durable review artifact

- **Problem.** 84 `general-purpose` reviews returned a median 4,904 B (p90 8,162,
  max 14,459) of prose; **0 of 84 wrote a durable artifact**; only 26 of 84 lead
  with a verdict; the controller then hand-summarised into `progress.md` (248
  writes, 545.6 KB). Findings are paid for three times and survive only in a
  transcript.
- **Mechanism.** Reviewers write full reasoning to
  `docs/tasks/<task>/reviews/<unit>.md` and return the ≤600 B block from §B.6
  (`verdict`/`artifact`/`critical`/`important`/`minor`/`blocking`). Controller
  reads the artifact only when `verdict != PASS`. Adopt `atlas-verifier`'s 478 B
  return as the reference shape.
- **Files/workflows.** `.claude/commands/execute-task.md` (review dispatch step);
  `.claude/agents/plan-adherence-reviewer.md`,
  `.claude/agents/backend-guidelines-reviewer.md`; the review brief template.
  Note that 84 of 93 review dispatches used bare `general-purpose` — this task
  should also give per-unit review a named agent type so the contract has
  somewhere to live.
- **Expected benefit.** Return bytes 418 KB → ~50 KB; notification carry
  40.3 → ~6 MB-turns; and review findings become durable and greppable across
  sessions.
- **Measure.** Median review return < 800 B; ≥95% verdict-first; a
  `reviews/` directory exists per task.
- **Failure modes.** (1) A compact `PASS` hides a real concern the prose would
  have surfaced — mitigate with `PASS_WITH_NOTES` routing to the artifact. (2)
  Reviewers pad the `blocking` field into a prose dump — mitigate with an explicit
  one-sentence-per-finding cap in the agent definition.

### Task C — Early-read discipline for large artifacts

- **Problem.** 268 results >12 KB carry 26.8% of all tool bytes, and their
  **median position is 0.10 of their stream (75% in the first quarter)** — the
  maximum-multiplier position. Instances: `plan.md` read 13× in one
  `plan-adherence-reviewer`; three review diffs of 53.2/39.7/34.3 KB read whole;
  five offloaded `tool-results/*.txt` read whole at >40 KB; `routes.conf` read
  whole (19.2 KB) at call #5 when the agent needed 5 lines (and greped for them at
  #12).
- **Mechanism.** Add to the reviewer/implementer definitions: for any file
  expected >20 KB, lead with `grep -n`/`sed -n`/`--stat` and escalate to a full
  read only if the slice is insufficient; for commit-range review, `git diff
  --stat` first, then per-file hunks. Give `plan-adherence-reviewer` a per-task
  plan extractor so it never re-reads a 291 KB `plan.md`.
- **Files/workflows.** `.claude/agents/*.md` (reviewer + implementer discovery
  sections); possibly `tools/task-brief.sh` for the plan extractor.
- **Expected benefit.** Convert a large share of the 5.45 MB heavy tail into
  slices, in the position where carry is highest.
- **Measure.** Count of >12 KB results per agent, and their median stream
  position, on the next comparable task.
- **Failure modes.** Under-reading → missed context → a fix round that costs more
  than the saving. **This must be phrased as a default with an explicit escalation
  path, never as a prohibition.** Judge it on rework rate, not on byte count
  alone.

### Task D — `tools/task-context.sh` deterministic fact block

- **Problem.** 2,191 Bash calls / 2.07 MB across the chain establish mechanical
  facts (branch, worktree, base sha, changed files, module list, symbol existence).
  `tools/task-resolve.sh` already answers part of this and was used in 16 of 213
  streams; 1,606 calls instead carried a literal 86-character `cd …` prefix.
- **Mechanism.** One script emitting the <400 B `key=value` block from §4.5,
  composed from existing tools where possible. Brief generation prepends it to
  every dispatch. `applicable_guards` derived from changed paths, **failing open**
  (emit all guards when the mapping is uncertain).
- **Files/workflows.** New `tools/task-context.sh`; brief generation in
  `.claude/commands/execute-task.md`; `tools/task-brief.sh`.
- **Expected benefit.** Direct saving is modest (~1–2%); the real gains are a
  shorter orientation prefix before first edit and removal of a class of
  "which gate applies" reasoning.
- **Measure.** Median implementer orientation prefix (13 calls in task-232) and
  the count of `git worktree list`/`status`/`branch` calls per stream.
- **Failure modes.** A stale fact block after a rebase — regenerate per dispatch,
  never cache across a session. A wrong `applicable_guards` silently skipping a
  gate — hence fail-open, and `tools/verify.sh` remains the authority.

### Task E — Lightweight per-agent telemetry

- **Problem.** Producing this report required ~250 lines of ad-hoc transcript
  parsing to recover facts (turns, tool calls, result bytes, return size, model)
  that a 200-byte ledger line would have carried. Nothing about per-turn context
  *composition* is recoverable at all.
- **Mechanism.** On each agent completion the controller appends one line to
  `docs/tasks/<task>/agent-ledger.tsv`:
  `unit · agent_type · model · turns · tool_calls · tool_result_bytes · return_bytes · status · commit`.
  All eight fields are available to the controller at reconcile time.
- **Files/workflows.** `.claude/commands/execute-task.md` reconcile step.
- **Expected benefit.** Makes "did helper X reduce discovery turns?" and "are
  controllers or workers more expensive?" answerable by reading one file.
- **Measure.** The next cost audit answers §11's questions without transcript
  parsing.
- **Failure modes.** Ledger drift if the controller forgets a line (accept —
  partial data still beats none). Scope creep into an observability platform —
  **one TSV line per agent, nothing more.**

**Not proposed.** Automating the read-slice family, the existence-grep family, or
package-content reads; shrinking dispatch briefs; touching the 120-call budget,
the verification split, or the four-phase flow. §8 states why for each.

---

## 11. Measurement / telemetry gaps

What was **directly recoverable** from retained evidence: per-message `usage`
(so billed input, output, and per-turn peak are exact); every tool call with its
full input; every tool result with its full text; agent type and model per
dispatch; agent return text; session `cwd` and timestamps. This is a genuinely
good telemetry substrate — the 1,259M / 270M / 988M split was reproduced
independently and matched the prior audit to ~3%.

What was **not** recoverable, in priority order:

| Gap | Why it mattered here | Minimal fix |
|---|---|---|
| **Context composition per turn** | I can measure that a 250k window existed and that 0.92 GB-turns of tool text flowed through it, but not what share of any given window was prefix vs. tool results vs. conversation. §7's token conversions are therefore modelled (4 bytes/token), not measured. | Record, per turn, a three-way byte split (system+tools / tool results / conversation). |
| **Agent identity ↔ plan unit** | Linking `agent-a64868a95292e8c3c` to "Task 9E review" required matching dispatch prompts by hand. | The Task-E ledger line. |
| **Cost of a *declined* handoff** | The prior audit reconstructed this for one session by hand. There is no marker for "a handoff was written and not taken." | Emit a `HANDOFF` ledger row with the context size at that moment. |
| **Whether a review finding changed anything** | Only inferable when a fix commit followed. 84 reviews produced 1 explicit Critical; how many of the rest were load-bearing is unknown. | Task-B's review artifact + `verdict` field makes this greppable. |
| **Offloaded tool results** | Five results >40 KB were re-reads of `tool-results/*.txt`; the original call's true size is one indirection away. | Record the original byte size alongside the offload pointer. |

**Recommended retention set** — deliberately eight fields, one line per agent
(Task E): `unit`, `agent_type`, `model`, `turns`, `tool_calls`,
`tool_result_bytes`, `return_bytes`, `status`+`commit`. Plus, if cheap at the
harness level, the per-turn context three-way split.

That set answers the questions posed:

- *Which workflow phases dominate?* → sum billed input by session, already
  available; the ledger attributes it to units.
- *Controllers or workers?* → **workers, 988M vs 270M (78/22)** — already
  answerable, and worth re-checking after task-234's ceiling lands.
- *Which tools inject the most persistent context?* → `tool_result_bytes` by
  agent + the carry weighting in §7. **Today: `Read` of task-artifact markdown,
  not source code.**
- *Did a new helper reduce discovery turns?* → orientation-prefix call count
  before/after (baseline: **median 13 for implementers, 18 for all subagents**).
- *Did compact returns reduce controller growth?* → `return_bytes` median by agent
  type (baseline: **implementer 1,145 B, general-purpose 4,904 B**).

---

## Appendix — reproduction

Analysis scripts are ad-hoc and were run from the job scratch directory; they are
not committed. All figures are reproducible from the transcripts by:

1. Selecting sessions whose modal `cwd` is the task worktree (yields the 17).
2. Deduping assistant messages by `message.id`; summing
   `usage.cache_read_input_tokens + cache_creation_input_tokens + input_tokens`.
3. Deduping tool calls by `tool_use.id`; pairing with `tool_result` by
   `tool_use_id` for result bytes.
4. Carry = result bytes × (count of later deduped assistant turns in that stream).
5. Return payload = the final distinct assistant text block of a subagent
   transcript.

Baselines any follow-up should be measured against:

| Metric | task-232 baseline |
|---|---|
| Billed input | 1,259M (270M main / 988M sub) |
| Turns | 1,725 main / 8,849 sub |
| Tool calls | 11,102 (7,064 Bash) |
| Tool-result bytes | 20.36 MB |
| Total carry | 0.92 GB-turns |
| Results >12 KB | 268 (26.8% of tool bytes), median position **0.10** |
| Implementer orientation prefix | median **13** calls |
| Implementer return | median **1,145 B** |
| Review return | median **4,904 B**, 0/84 durable artifacts |
| Notification ingest | 270, 725.5 KB, 40.3 MB-turns carry |
| zsh glob failures | 113 / 11,102 (1.0%) |

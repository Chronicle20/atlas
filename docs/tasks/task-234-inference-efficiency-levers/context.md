# Context — task-234 inference efficiency levers

Companion to [`plan.md`](plan.md). Everything an implementer needs that the plan's
task sections do not carry. Spec: [`prd.md`](prd.md). Design: [`design.md`](design.md).

## What this task is

A harness-configuration and process-documentation change aimed at the *cost* of
running Atlas tasks, not at any product behavior. Evidence base is a session-level
audit of task-232 (17 sessions, 1,294M billed input tokens):
`~/.claude/audits/session-task-232-three-largest.md` (user scope, outside the repo).

**No service code is touched.** Every edit lands in `.claude/`, `docs/`, or (for
Task 6 only) a user-scope script outside the repo. There is no Go or TypeScript
change, no test cycle, and no migration.

## The verification trap — read this first

`tools/verify.sh` will pass trivially for Tasks 1, 3, 5a, and 7 because those
tasks change only prose. **A green gate is not evidence any of this worked.**

- The real evidence for FR-1 (Task 2) is a live dispatch of each restricted agent
  completing without hitting a denied tool, plus the re-measured median subagent
  starting context.
- The real evidence for FR-3 (Task 1) and FR-4 (Task 7) is a reading check: is the
  rule unambiguous about *stopping* rather than *recommending*?
- Task 6 is the measurement harness that makes FR-5.2's before/after possible.

Do not report a prose task as "verified" on the gate alone. Say what was read and
why it is unambiguous.

## Key files

### Edited

| Path | Task | Role |
|---|---|---|
| `.claude/commands/execute-task.md` | 1, 5a, 7 | Gate-reconcile step gains the context ceiling; review step gains the mechanical/judgment distinction |
| `docs/agent-dispatch.md` | 1, 5a | Cross-reference so the ceiling and the codemod rule are discoverable from the dispatch doc |
| `.claude/agents/*.md` (all 12) | 2 | Frontmatter only — add an explicit `tools:` field |
| `.claude/commands/plan-task.md` | 5a (maybe) | Only if the codemod decision rule belongs at plan time as well as dispatch time |
| shell-conventions owner (find with `Grep`) | 3 | One-to-two-sentence glob-quoting note |
| `~/.claude/tools/session-digest.sh` | 6 | **User scope — outside the repo. Do not commit it.** |

### Created

| Path | Task | Role |
|---|---|---|
| `docs/codemod-vs-agents.md` | 5a | The decision rule plus the deferred rewriter's contract. May instead be a section in `docs/agent-dispatch.md` — either is acceptable; pick one and do not do both. |

### Explicitly NOT created

`tools/<name>/` — the retrospective AST rewriter. Q3 resolved to option (b); Task
5b is **dropped**. Do not create a Go module for it. Its contract survives as
written specification in Task 5a step 3.

## Decisions already made — do not relitigate

- **Q2 → both.** The FR-3 ceiling states a token threshold as the primary trigger
  *and* "after N completed plan tasks" as a fallback for a controller that cannot
  read its own context size. Not one or the other.
- **Q3 → option (b).** Ship the decision rule and the rewriter's written contract
  now; build the real rewriter against the next sweep-shaped task. Confirmed by the
  user (commit `010889154`). The rationale of record is in `plan.md` Task 4: the
  measured waste was not "we lacked a rewriter," it was that nobody asked whether
  one was cheaper than 6,231 implementer turns.
- **Three packet agents keep the wide tool set.** `packet-implementer`,
  `packet-verifier`, `dispatcher-family-implementer` drive IDA through MCP. FR-1.3
  requires the definition to say so inline, not silently.
- **Not changed by this task:** the 120-call implementer budget, the verification
  split (implementer module-local vs `atlas-verifier` repo-wide), and the
  four-phase flow. The audit found all three earning their cost.

## Numbers to cite (measured, not estimated)

Quote these rather than paraphrasing — several land in commit messages and in the
rules themselves.

| Figure | Value |
|---|---|
| task-232 total billed input | 1,294M (302M main + 993M subagent) |
| Fixed prefix share | ≈434M of 1,294M — 34% |
| Subagent starting context | median **37.3k** (n=197, p25 35.9k, p90 39.3k) |
| Of that: `CLAUDE.md` ~2.6k, agent definition ~2.7k, leaving ~32k base prompt + tool schemas |
| Implementer turns | 6,231 turns / 760M / 59% of the task / ~122k per turn |
| Review + audit agents | 2,616 turns / 227M / 17.6% |
| The declined handoff | `854e6e87` wrote HANDOFF #10 at 243k, then ran 26 more turns at 259k avg = 6.73M, for one plan task |
| Corroborating | `last context == peak context` in **all 17** sessions |
| Glob-quoting annoyance | 60 occurrences across 5 sessions |
| task-232 batch-4 commits (worked example for Task 5a) | `6a06ffae0`, `54e7e0c3d`, `8776709b8` |

Targets: subagent starting context **< 28k**; prefix share **< 22%**; no execute
session peak above ~180k.

## The mechanical/judgment split (Task 5a's worked example)

From task-232 batch 4 (`6a06ffae0`), the per-call-site transformation was six
steps:

1. `requests.RootUrl(` → `requests.RootUrlFor(ctx, ` — **AST**
2. add `"context"` to the import block — **AST**
3. thread `ctx` through the URL builder and its callers — **AST** (call-graph walk)
4. `main.go`: add `service.WithEnvironmentRegistry(serviceName)` to `Bootstrap` — **AST**
5. every Kafka consumer's `SetHeaderParsers` gains `consumer.EnvHeaderParser`,
   ordered after `TenantHeaderParser` — **AST**
6. error propagation: the URL builder now returns `(string, error)`, so each caller
   gains an `if err != nil` block with a service-appropriate log message —
   **judgment**, e.g. `"Unable to resolve base URL for character [%d]'s pets."`

That is the split FR-2.2 formalizes: rewrite what is derivable, **list** what is
not — never silently skip a site.

## Conventions that bite here

- **Repo-relative paths in committed files** — enforced under `docs/`. The
  `~/.claude/...` paths above appear in this context file and in `plan.md` as
  evidence references; if a path must appear in a rule under `docs/`, write it as a
  placeholder.
- **Preserve existing line endings.** Several `.claude/` files may be CRLF; do not
  normalize as a side effect of a frontmatter edit.
- **Quote glob arguments in shell calls** — `--include='*.go'`, not
  `--include=*.go`. This is literally Task 3's subject; it applies while doing the
  work too.
- **Frontmatter edits only** in Task 2. Do not rewrite agent bodies while adding
  `tools:` — the one exception is the inline "why wide" comment FR-1.3 requires on
  the three packet agents.

## Ordering

Tasks 1, 2, 3, and 6 are independent and can land in any order; 1 and 2 are the
cheap high-impact ones. Task 5a is independent of those. Task 7 depends on 5a
because "mechanically transformed" must be defined before the review step can key
off it — and, per the Q3 resolution, Task 7 likely reduces to documentation since
the `--check` mode it would have keyed on does not yet exist.

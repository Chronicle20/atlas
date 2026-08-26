# Process Parity Across the Four Repositories

Canonical specification for harmonizing the agentic development process across
`atlas`, `home-hub`, `Harbormaster`, and `MyFleet`.

This document is the single source of truth for the harmonization effort. Each
repository executes its own four-phase task cycle; each of those tasks embeds a
verbatim copy of this document in its task folder, plus that repository's row
from the binding table in §4. There is no sync mechanism — every repository ends
up self-contained. Consistency is asserted mechanically at the end (§7), not
maintained continuously.

## 1. Problem statement

The four repositories already share the *workflow* backbone: the four phase
commands, the worktree convention, the artifact-location override, the three
reviewer agents, and the two guideline skills. What they do not share is the
*context-discipline* layer that `atlas` grew afterwards — the enforcement hooks,
the owner-doc set, the budget-capped implementer, the isolated verifier, and the
terse rule-list `CLAUDE.md` shape that the owner-doc table makes possible.

`home-hub`, `Harbormaster`, and `MyFleet` are all sitting at the previous
generation: a prose-narrative `CLAUDE.md` of 69–107 lines, no enforcement hooks
beyond `skill-activation-prompt`, and no implementer/verifier/reviewer trio, so
`/execute-task` falls back to uncapped generic dispatch.

## 2. Decisions taken

| Decision | Choice |
|---|---|
| Sharing mechanism | None. Manual per-repo port; each repo self-contained; drift accepted and re-harmonized occasionally. |
| Scope | Full parity — hooks, agent trio, verify entrypoint, owner docs, `/fix-pr-bug`, and the `CLAUDE.md` restructure. |
| Agent naming | Generic `task-implementer` / `task-verifier` / `task-reviewer` in **all four**, including a rename in `atlas`. |
| Execution | One four-phase task cycle per repository, orchestrated against this document. |
| Consistency proof | Post-hoc mechanical assertion (§7), not a sync tool. |

The naming decision is what makes the rest cheap: `commit-boundary.sh` and
`docs/agent-dispatch.md` currently hardcode `atlas-implementer` /
`atlas-verifier`. Once those names are generic, the portable files are
byte-identical across all four repositories and a future re-harmonization is a
straight file copy rather than a per-repo edit.

## 3. File manifest

### 3.1 Hooks — portable verbatim

These contain no repository-specific paths. After the §5.1 rename they are
byte-identical in all four repositories.

| File | Lines | Wired at | Notes |
|---|---|---|---|
| `.claude/hooks/wait-loop-guard.sh` | 91 | `PreToolUse` / `Bash` | Blocks polling and `sleep` loops. |
| `.claude/hooks/wait-loop-guard_test.sh` | — | — | Ships with its subject. |
| `.claude/hooks/block-home-paths-in-docs.sh` | 34 | `PreToolUse` / `Write\|Edit` | Rejects literal home/absolute paths under `docs/`. |
| `.claude/hooks/turn-budget.sh` | 138 | `PostToolUse` / `*` | Counts tool calls per agent. |
| `.claude/hooks/turn-budget-guard.sh` | 109 | `PreToolUse` / `*` | Makes the implementer tool-call cap binding. |
| `.claude/hooks/fork-dispatch-guard.sh` | 53 | `PreToolUse` / `Agent` | Surfaces the cost of `subagent_type: "fork"`. |
| `.claude/hooks/task-num-collision-detector.sh` | 37 | `SessionStart` | Requires `tools/task-numbers.sh` (§3.4). |

`turn-budget.sh`, `turn-budget-guard.sh`, and `commit-boundary.sh` mention agent
names only in comments and operator-facing messages; the §5.1 rename resolves
those to the generic names.

### 3.2 Hooks — require per-repo adaptation

| File | Lines | What is repo-specific |
|---|---|---|
| `.claude/hooks/format-on-write.sh` | 45 | Hardcodes `services/atlas-ui` for prettier and sources `tools/toolchain.versions` for the pinned `golangci-lint`. Both must be rebound per §4. |
| `.claude/hooks/commit-boundary.sh` | 140 | References `tools/task-brief.sh` in its guidance text. Portable once that script exists (§3.4). |

### 3.3 Agents

Rename in `atlas`, create in the other three:

| Agent | Purpose | Present in |
|---|---|---|
| `task-implementer` (was `atlas-implementer`) | One plan task; 120 tool-call budget with a `PARTIAL` hand-back; module-local build/test only; brief-first discovery. | atlas only |
| `task-verifier` (was `atlas-verifier`) | Runs the repo-wide verification gate in its own clean context; returns `PASS` or the first failing block; never edits. | atlas only |
| `task-reviewer` (was `atlas-reviewer`) | Per-unit review of one commit range against its brief; durable artifact plus verdict-first return; no recursive fan-out. | atlas only |

Already present in all four, no change: `backend-guidelines-reviewer`,
`frontend-guidelines-reviewer`, `plan-adherence-reviewer`, `todo-scanner`.

Present in `atlas` and `home-hub` only, port to `Harbormaster` and `MyFleet`:
`service-documentation` + its `/service-doc` command.

### 3.4 Tools

| File | atlas | home-hub | Harbormaster | MyFleet |
|---|---|---|---|---|
| `tools/verify.sh` | ✅ | create (§4) | create (§4) | create (§4) |
| `tools/task-numbers.sh` | ✅ | port | port | ✅ |
| `tools/task-brief.sh` | ✅ | port | port | port |

`tools/verify.sh` must expose the same contract everywhere: **flagless run exits
0 means the branch may be called done**; `--quick` / `--no-docker` exit 0 too but
skip the slow gates and do not count as done.

### 3.5 Owner documents

Ported to `docs/` in each repository, genericized per §5.2. Total 1,519 lines in
`atlas` today; expect the ported versions to be shorter once Atlas-specific
examples are replaced.

| Document | Lines | Owns |
|---|---|---|
| `agent-dispatch.md` | 236 | Model pinning, fan-out vs. fork, handoff decision |
| `verification.md` | 368 | Gate failures, script/CI disagreement |
| `superpowers-integration.md` | 185 | Bare task numbers, skills outside a phase command |
| `review-protocol.md` | 178 | Dispatching a reviewer, writing up a review |
| `post-implementation.md` | 160 | Phase 5, `/fix-pr-bug` |
| `codemod-vs-agents.md` | 138 | Second implementer at the same transformation |
| `slice-first.md` | 107 | Reading a large document, diff, plan, or tool result |
| `tooling-conventions.md` | 95 | Long-running processes, mechanical repo facts, shell conventions |
| `git-workflow.md` | 52 | Committing, pushing, rebasing, stray `main` commits |

`home-hub` already has a `docs/superpowers-integration.md`; reconcile rather than
overwrite.

### 3.6 Commands

`/fix-pr-bug` (Phase 5) is missing from all three and is ported as-is.

### 3.7 Settings

`.claude/settings.json` gains `disableBundledSkills: true` and the full hook
wiring — `PreToolUse` (`Write|Edit`, `Agent`, `Bash`, `*`), `PostToolUse`
(`Write|Edit`, `*`, `Bash`), `SessionStart`, `UserPromptSubmit`.

## 4. Per-repository binding table

Everything above is identical across repositories. Everything below is not, and
is the only thing each repo's task may vary.

### atlas

| Binding | Value |
|---|---|
| Verify entrypoint | `tools/verify.sh` (exists) |
| Go layout | `go.work`, `libs/*` + `services/*` |
| Frontend path | `services/atlas-ui` (prettier) |
| Go formatter | pinned `golangci-lint` via `tools/toolchain.versions` |
| Task numbers | `tools/task-numbers.sh` (exists) |
| Work in this task | Rename the agent trio; extract this document; no new tooling |

### home-hub

| Binding | Value |
|---|---|
| Verify entrypoint | **create** `tools/verify.sh` wrapping `scripts/ci-build.sh`, `scripts/ci-test.sh`, `scripts/lint-all.sh` |
| Go layout | `go.work`, `services/*` + `shared/go/*` |
| Frontend path | `frontend/` |
| Go formatter | resolve — no pinned linter config found; decide during that repo's design phase |
| Task numbers | **port** `tools/task-numbers.sh` |
| Docker note | existing `CLAUDE.md` requires verifying Docker builds when shared libraries change — fold into `verify.sh`, do not drop |

### Harbormaster

| Binding | Value |
|---|---|
| Verify entrypoint | **create** `tools/verify.sh` from the existing prose checklist: backend `go test -race -count=1 ./...`, `go vet ./...`, `golangci-lint run`, `CGO_ENABLED=0 go build ./...` (cwd `apps/backend`); frontend `npm ci`, `npm run lint`, `npm run format`, `npm test`, `npm run build` (cwd `apps/frontend`); container `docker buildx build --platform linux/amd64,linux/arm64 -f deploy/docker/Dockerfile .` |
| `--no-docker` | skips the buildx step |
| On-demand, excluded from flagless | `HARBORMASTER_INTEGRATION=1 go test -tags=integration`, `npm run test:e2e` |
| Go layout | no `go.work`; single module at `apps/backend` |
| Frontend path | `apps/frontend` |
| Task numbers | **port** `tools/task-numbers.sh` |

### MyFleet

| Binding | Value |
|---|---|
| Verify entrypoint | **create** `tools/verify.sh` wrapping `make ci` plus the manifest gates |
| Manifest gates | `kustomize build deploy/k8s/overlays/local` and `.../main`, then **both** `kubectl apply --dry-run=server` runs when a cluster is reachable |
| `main` overlay assertion | renders with no PVCs, no Secrets, no ClusterRole, no placeholders |
| Node bootstrap | `export NVM_DIR="$HOME/.nvm" && . "$NVM_DIR/nvm.sh" && nvm use 22` when `npm` is absent |
| Go layout | `go.work`, `apps/*` + `packages/*` |
| Frontend path | `apps/web` |
| Go formatter | `tools/lint.sh` + `tools/lint.versions` |
| Task numbers | `tools/task-numbers.sh` (exists) |

## 5. Transformations

### 5.1 Agent rename

`atlas-implementer` → `task-implementer`, `atlas-verifier` → `task-verifier`,
`atlas-reviewer` → `task-reviewer`. Update every reference: the three agent
definition files, `.claude/hooks/commit-boundary.sh`,
`.claude/hooks/turn-budget.sh`, `.claude/hooks/turn-budget-guard.sh`,
`docs/agent-dispatch.md`, `docs/review-protocol.md`, `CLAUDE.md`, and any phase
command that names them. This is a mechanical repository-wide sweep and is the
one place a scripted rename is appropriate.

### 5.2 Owner-doc genericization

The *rules* in the owner docs are repository-agnostic; the *examples* are not.
Replace Atlas-specific illustrations — packet work, WZ data, IDA, service
opcodes, `tools/verify.sh` flag specifics — with the target repository's
equivalents, or with a neutral example where the target has none. Do not delete a
rule because its example does not transfer; find a new example.

`atlas`-only owner docs that do **not** port: anything under `docs/packets/`,
`docs/reverse-engineering.md`, `docs/adding-a-new-service.md` (port only if the
target repo has a service-scaffolding story), `docs/observability.md` (port only
if the target has a deploy story worth a runbook).

### 5.3 `CLAUDE.md` restructure

Each repository's `CLAUDE.md` is rewritten from prose narrative into the rule-list
shape: `# <Repo>`, then `## Never do this`, `## Evidence & grounding`,
`## Development workflow`, `## Done means verified`, `## Dispatching agents`,
`## Handing off context`, `## Repository conventions`, and
`## Where the procedures live` (the trigger → owner table).

Rules that are currently prose in the three repos and must survive the rewrite,
not be lost in it:

- home-hub: verify Docker builds when shared libraries change; `scripts/local-up.sh` for local deployment.
- Harbormaster: the full build/verification command set (moves into `verify.sh`, referenced from `## Done means verified`); the "repository is unscaffolded, update this file once settled" note — check whether it is now stale.
- MyFleet: the `make ci` target list; the Node/nvm bootstrap; the manifest render and dual dry-run requirement, including the recorded incident about the local overlay's missing `namespace:` — that incident is evidence for why both dry-runs are required and must be preserved.

Content that stays repo-specific and is not homogenized: project overview, build
commands, deployment specifics, domain conventions.

## 6. Execution plan

Four task cycles, `atlas` first because it is the rename source.

1. **atlas** — agent trio rename, `CLAUDE.md` owner-table update, commit this document. Unblocks the byte-identical portable set.
2. **home-hub**, **Harbormaster**, **MyFleet** — may run in any order or concurrently once (1) lands, since they do not touch each other.

Each cycle is the standard four phases (`/spec-task` → `/design-task` →
`/plan-task` → `/execute-task`) in that repository's own worktree, with this
document and the repo's §4 row as the PRD's input. The largest genuine
engineering in the whole effort is `tools/verify.sh` for `Harbormaster` and
`home-hub`; everything else is porting text.

## 7. Definition of done

The effort is complete when all of the following hold. Each is mechanically
checkable — this is what makes "orchestrated for consistency" an assertion rather
than an intention.

1. The seven §3.1 hook files are byte-identical across all four repositories
   (`diff` returns empty for each file, pairwise).
2. Each repository has `tools/verify.sh`, `tools/task-numbers.sh`, and
   `tools/task-brief.sh`, and the flagless `tools/verify.sh` exits 0 in each.
3. Each repository defines `task-implementer`, `task-verifier`, and
   `task-reviewer`, and no reference to `atlas-implementer`, `atlas-verifier`, or
   `atlas-reviewer` remains anywhere.
4. Each `.claude/settings.json` wires the same hook set at the same events,
   differing only where §4 says it may.
5. Each `CLAUDE.md` carries the same eight section headings, with only the
   repo-specific content of §5.3 differing, and each ends with a
   `## Where the procedures live` table whose every target file exists.
6. Each repository has the nine §3.5 owner documents, and every `docs/` link in
   every `CLAUDE.md` resolves.

## 8. Open items

- **home-hub Go formatter.** No pinned linter configuration was found; `format-on-write.sh` needs one. Resolve during that repository's design phase.
- **Harbormaster scaffolding note.** `CLAUDE.md` still claims the repository is unscaffolded while `apps/backend` and `apps/frontend` exist. Confirm current state before rewriting.
- **`docs/superpowers-integration.md` in home-hub.** Already present; reconcile against the Atlas version rather than overwriting.

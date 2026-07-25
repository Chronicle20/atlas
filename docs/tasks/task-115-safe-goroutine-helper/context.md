# task-115 Context — Safe Goroutine Helper (RR-6)

Companion to `plan.md`. Key files, verified facts, decisions, and dependencies gathered during planning (2026-07-02).

## Artifacts

- PRD: `docs/tasks/task-115-safe-goroutine-helper/prd.md`
- Design (normative): `docs/tasks/task-115-safe-goroutine-helper/design.md`
- Plan: `docs/tasks/task-115-safe-goroutine-helper/plan.md`
- Audit table (created in plan Task 3): `docs/tasks/task-115-safe-goroutine-helper/migration-audit.md`
- Worktree: `.worktrees/task-115-safe-goroutine-helper/`, branch `task-115-safe-goroutine-helper`

## Key files

| File | Role |
|---|---|
| `libs/atlas-routine/routine.go` (new) | The helper: `routine.Go(l, ctx, fn)`, package `routine`, module `github.com/Chronicle20/atlas/libs/atlas-routine` |
| `tools/goroutineguard/` (new) | `go/analysis` analyzer, twin of `tools/rediskeyguard` (same module layout, `GOWORK=off`, analysistest fixtures, singlechecker cmd) |
| `tools/goroutine-guard.sh` (new) | Wrapper: analyzer self-test → build → sweep every go.mod under `services/` AND `libs/` (redis-key-guard sweeps services only) |
| `tools/redis-key-guard.sh`, `tools/rediskeyguard/` | The precedent both guard components copy |
| `libs/atlas-lock/leader.go:155-190` | The one hand-rolled goroutine recover; replaced with completed-flag pattern (design §6.3). Existing `leader_test.go:288-320` pins panic→`lostTotal{reason="panic"}`+lease-release semantics |
| `libs/atlas-kafka/consumer/manager.go` | 3 spawn sites (:145, :523, :558). `safeHandle` (:577) is untouched by design |
| `libs/atlas-model/model/processor.go` (:155,:167,:208,:220,:441), `async/processor.go` (:72) | The `logrus.StandardLogger()` sites (module has no logger in its public API) |
| `libs/atlas-model/testutil/helpers.go:189` | The ONLY allowlist entry (`//goroutine-guard:allow` marker) — panic propagation is the point of a test harness |
| `libs/atlas-socket/server.go` (:125,:152,:173,:226), `libs/atlas-rest/server/server.go` (:171,:186), `libs/atlas-seeder/handlers.go` (:49) | Remaining lib sites, all mechanical |
| `Dockerfile` (repo root) | New-lib wiring needs **3** edits: 2 COPY lines + the `for L in ...` synthesized-go.work loop at line 91 (Dockerfile:15 comment is authoritative; design §2.4 undercounts at 2) |
| `go.work` | Gains `./libs/atlas-routine`. Does NOT gain `tools/goroutineguard` (rediskeyguard isn't in it either) |
| `.github/workflows/pr-validation.yml` | redis-key-guard job ends ~:98 (clone for goroutine-guard); `needs:` ~:480; results block ~:495-518 |
| `.claude/agents/backend-guidelines-reviewer.md` ~:101 | DOM-24 is the last DOM row; DOM-25 appends after it |
| `.claude/skills/backend-dev-guidelines/SKILL.md`, `resources/anti-patterns.md` | Skill-side DOM-25 content |

## Verified facts (planning-time)

- **Baseline count:** repo-root grep for statement-form `go` lines in non-test code under `services/` + `libs/` = **164** (design says ≈165). The analyzer's Task 3 run is the authoritative number; audit-table row count must equal it.
- **Affected services (33):** account, asset-expiration, ban, buffs, cashshop, channel, character, character-factory, configurations, data, doors, drops, expressions, families, guilds, invites, login, maps, marriages, merchant, monster-death, monsters, mounts, npc-conversations, party-quests, pets, reactors, renders, saga-orchestrator, skills, summons, transports, world. Heaviest single files: `atlas-channel kafka/consumer/map/consumer.go` (19), `movement/processor.go` (10), `party/consumer.go` (8), `session/consumer.go` (7).
- **Affected libs (6 + 1 allowlisted):** kafka, lock, model, rest, seeder, socket (+ model/testutil allowlisted).
- `libs/atlas-model/go.mod` currently has **no logrus dependency** — Task 6 adds a direct `github.com/sirupsen/logrus v1.9.4` require alongside atlas-routine.
- `SliceMap`'s parallel branch (`model/processor.go:441`) has **no ctx in scope** — the single `context.Background()` case (design §4.2 rule 3). `ExecuteForEach*` and `async.AwaitSlice` own a ctx.
- Lib→lib replace path is `../atlas-routine`; service→lib is `../../../../libs/atlas-routine`; require version string `v0.0.0-00010101000000-000000000000` (pattern verified in `services/atlas-monsters/.../go.mod` and `libs/atlas-kafka/go.mod`).
- `atlas-monsters` `ProcessorImpl` has `p.l`/`p.ctx` fields (processor.go:71-73) — the PRD's motivating site migrates with them.
- **RR-6 wrinkle:** `docs/architectural-improvements.md` on this branch has NO `RR-*` sections. The RR-6/7/8 text exists only in the main checkout's **uncommitted** working-copy rewrite of that doc (verified: `git diff --stat` in main shows 216 insertions uncommitted). Plan Task 12 Step 4 is therefore conditional: mark resolved if the rewrite has landed by then; otherwise record for rebase-time and re-check before PR. Do not invent an RR-6 section.
- logrus v1.9.4 across the repo; new modules use `go 1.25.5`; guard uses `golang.org/x/tools v0.47.0` (matches rediskeyguard).

## Decisions locked in design (do not relitigate)

1. Single spawn shape `routine.Go(l logrus.FieldLogger, ctx context.Context, fn func(context.Context))`; log message fixed: `Recovered panic in background goroutine.` with `panic` + `stack` fields; ctx pass-through only.
2. Guard = AST analyzer (not grep); allowlist = inline `//goroutine-guard:allow <justification>` marker (same line or line above), justification machine-required; exactly one initial entry (testutil).
3. atlas-model uses `logrus.StandardLogger()` — no `SetLogger` API, no breaking logger params. Accepted consequence: a recovered worker panic yields a nil/zero result or `ErrAwaitTimeout` instead of a returned error; the log line is the detection path.
4. atlas-lock keeps `setReason("panic")`/`cancelLeader()` semantics via the completed-flag pattern (no second recover). `safeHandle` in atlas-kafka stays.
5. Migration is mechanical: bodies byte-identical, sync primitives move inside the closure, bind `_ context.Context` and keep originally-captured ctx references.

## Dependencies / sequencing

- Task 1 (helper) blocks everything; Task 2 (analyzer) blocks Task 3 (wrapper/CI/baseline); Task 3's baseline blocks the audit table row-count check in Task 11.
- Tasks 4–10 (migration) are independent of each other once 1+3 exist, but run in plan order (protocol-heavy libs first, with explicit review — design §8).
- The CI guard job fails on this branch until Task 10 completes — expected; the PR lands whole.
- Task 13's `docker buildx bake all-go-services` is mandatory (every service go.mod touched — CLAUDE.md rule 4) and is the only check that validates the Task 1 Dockerfile edits.
- Code review (`superpowers:requesting-code-review`) before PR, per CLAUDE.md.

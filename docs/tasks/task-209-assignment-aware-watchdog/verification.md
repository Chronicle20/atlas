# task-209 — branch verification record

Evidence for plan Task 9. Every number below is the **actual** output of the
command named above it, captured on 2026-08-10 from the
`task-209-assignment-aware-watchdog` worktree. Where a step was not run, or was
run only partially, it says so explicitly.

---

## Step 2 — `libs/atlas-kafka` Go sweep

```bash
cd libs/atlas-kafka && go build ./... && go vet ./... && go test -race ./...
```

`go build` and `go vet` produced no output (clean). `go test -race`:

```
ok  	github.com/Chronicle20/atlas/libs/atlas-kafka/consumer	8.162s
ok  	github.com/Chronicle20/atlas/libs/atlas-kafka/consumergroup	(cached)
?   	github.com/Chronicle20/atlas/libs/atlas-kafka/handler	[no test files]
ok  	github.com/Chronicle20/atlas/libs/atlas-kafka/message	(cached)
ok  	github.com/Chronicle20/atlas/libs/atlas-kafka/producer	(cached)
?   	github.com/Chronicle20/atlas/libs/atlas-kafka/producer/producertest	[no test files]
ok  	github.com/Chronicle20/atlas/libs/atlas-kafka/retry	(cached)
?   	github.com/Chronicle20/atlas/libs/atlas-kafka/topic	[no test files]
```

Because the new engine is concurrency-heavy, `consumer` was additionally run at
`-count=5` under `-race` to check for flakes:

```
ok  	github.com/Chronicle20/atlas/libs/atlas-kafka/consumer	36.707s
```

No flake, no data race.

## Step 3 — every module that depends on `atlas-kafka`

```bash
for mod in $(grep -rl "atlas-kafka" --include=go.mod services libs | xargs -n1 dirname); do
  echo "== $mod"
  ( cd "$mod" && go build ./... && go vet ./... && go test -race ./... ) || echo "FAILED: $mod"
done
```

**63 modules swept, 0 failures.** `grep -c '^== '` → `63`; `grep -c FAILED` → `0`.
(The wrapper reported a non-zero shell status only because `grep -c` exits 1 when
it matches nothing — that is the passing case here, not a failure.)

## Step 4 — repo guards

| Guard | Result |
|---|---|
| `tools/redis-key-guard.sh` | exit 0 |
| `tools/goroutine-guard.sh` | exit 0 (self-test `ok …/goroutineguard 0.717s`) |
| `tools/lint.sh --check` | `lint.sh: OK` — 0 errors |

`tools/lint.sh --check` was run with nvm 22 on PATH. Its ESLint stage reports
**5 warnings, 0 errors**, all in `services/atlas-ui` files this branch does not
touch (`CreateTenantDialog.tsx`, `AccountsPage.tsx`, `QuestsPage.tsx` —
`react-hooks/incompatible-library` and `react-hooks/exhaustive-deps`). They are
the pre-existing atlas-ui lint baseline; the script still exits 0.

The service-registration, template-opcode-order, template-duplicate-binding,
template-movement-types, skill/job-id and buff-duration guards **do not apply**
and were not run: `git diff --name-only main...HEAD` is confined to
`libs/atlas-kafka/` and `docs/tasks/task-209-assignment-aware-watchdog/`, so no
`services.json`, deploy manifest, tenant template, or job/skill constant changed.

## Step 5 — service image bake

```bash
docker buildx bake all-go-services
```

Exit 0. All **62** declared `all-go-services` targets built — the count of
`naming to docker.io/library/<svc>:local` lines (62 distinct images) matches the
62 targets `docker buildx bake all-go-services --print` declares.

A naive `grep -cE "ERROR|error:"` over the build log returns 166, but every one
of those lines is BuildKit echoing the `build-env 48/51` step's own command
text, which contains the literal string `echo "ERROR: no module dir..."` as the
untaken failure branch of the module-directory guard. No step failed: there are
no `^ERROR` lines and no build-step non-zero exits.

This branch does not touch any `go.mod`, so no new `COPY libs/...` line was
required in the repo-root `Dockerfile`; the bake confirms nothing regressed.

## Step 6 — FR-3.2 and acceptance criteria, mechanically

| Check | Expected | Actual |
|---|---|---|
| `git diff --stat services/` | empty | empty |
| `git diff main...HEAD --name-only` | only `libs/atlas-kafka` + task docs | 19 files under `libs/atlas-kafka/consumer/`, 5 under `docs/tasks/task-209-assignment-aware-watchdog/`, nothing else |
| `grep -rn "GroupID:" libs/atlas-kafka/consumer/group.go` | no match | no match (exit 1) |
| `grep -rn "TODO\|FIXME" libs/atlas-kafka/consumer/` | no match | no match (exit 1) |

FR-3.2 holds: no service source changed, and the partition reader carries no
`GroupID`.

## Integration scenarios (plan Task 8, numbers recorded here)

```bash
cd libs/atlas-kafka && go test -race -tags integration -timeout 30m ./consumer/... -v
```

Result: `ok  github.com/Chronicle20/atlas/libs/atlas-kafka/consumer  636.696s`,
exit 0. Every scenario passed on **both** engines.

| Scenario | `consumergroup` | `reader` (legacy) |
|---|---|---|
| S1 steady-state latency | p99 **16.627 ms**, max **21.120 ms** (100 msgs) | p99 15.715 ms, max 16.106 ms |
| S2 idle-tick churn | p99 18.533 ms, max 18.533 ms, **recreates 0** | p99 14.564 ms, max 14.564 ms, **recreates 15** |
| S3 forced recreate | max dwell **523.27 ms**, recreates 1, timeToFirstFetch 8.15 ms, totalBackoff 1 s | max dwell **7.551 s**, recreates 1, timeToFirstFetch 7.025 s, totalBackoff 500 ms |
| S4 tick control | p99 18.537 ms, max 18.537 ms, recreates 0 | p99 15.301 ms, max 15.301 ms, recreates 0 |

S5 is engine-agnostic: idle fetch attempts in 30 s — `maxWait=50ms`: **641**;
`maxWait=10s`: **4**.

**S1 against the NFR.** The PRD names the task-136 S1 measurements as the
regression bar: p99 22.0 ms, max 87.1 ms. The `consumergroup` engine measured
p99 16.6 ms / max 21.1 ms — inside the bar on both statistics. It is ~0.9 ms
worse on p99 than the legacy engine in the same run (16.6 vs 15.7 ms) and
5.0 ms worse on max (21.1 vs 16.1 ms). Both engines are well under the NFR, so
risks.md **R6 is not triggered**; the numbers are reported as measured rather
than re-run for a better-looking sample.

**S2 is the headline.** Same workload, same ticks: the legacy engine recreated
its reader **15** times, the new engine **0**. That is the production churn this
task exists to remove, reproduced and eliminated in a test.

**S3 is the dwell payoff.** A forced recreate costs the new engine **523 ms** of
maximum dwell against the legacy engine's **7.55 s**, because the rebuild is
local to one partition reader and skips the group rejoin.

**S6 — `TestDwellS6_MembersExceedPartitions`: PASS (35.58 s).** The test does
not log its counter; its assertions are the record. It asserts
`unassignedSeen == true` (the members-exceed-partitions condition actually
reproduced), `require.Zero(t, totalRecreates)` — so **totalRecreates == 0** — and
that no consumer logged `wedged` or `stall suspect` (FR-2.5).

**Cross-engine offset round-trip — `TestCrossEngineOffsetRoundTrip`: PASS
(79.86 s)**, both directions: `consumergroup-then-reader` (36.50 s) and
`reader-then-consumergroup` (43.36 s). FR-5.3 holds — a rollback is safe in
either direction with no replay and no gap.

---

## Post-deploy measurements still outstanding

No test substitutes for these. Both are PRD acceptance criteria and remain
**unverified** until the branch runs on `atlas-main`:

1. **Recreate rate.**
   `count_over_time({namespace="atlas-main"} |= "Recreated reader for topic" [1h])`
   must be ~0 for all services. Baseline: **19–246 per hour per service**.
2. **No multi-second stall on the hot path.** An attack→drop-visible trace over
   a 10-minute play session with no multi-second gap. Baseline: **4.7 s** and
   **4.2 s** stalls in trace `bd9b801a…` on 2026-08-10.

When reading (1) after deploy, note that `recreateCount` is not comparable
across engines — on `reader` it counts group rejoins, on `consumergroup` it
counts local partition-reader rebuilds. See `libs/atlas-kafka/README.md`.

---

## Code review

Two reviewer agents were dispatched in parallel per CLAUDE.md's pre-PR
requirement. No frontend reviewer ran: the branch changes no TypeScript.

- `plan-adherence-reviewer` → `audit.md`. **9/9 plan tasks done.** Confirmed
  Task 1 was a byte-for-byte pure move (`git show e463478d8`), FR-3.2 holds, and
  none of the four PRD §9 open items was silently implemented or dropped.
- `backend-guidelines-reviewer` → `audit-backend.md`. **PASS**, no blocking
  findings. One Minor (FILE-06): `manager.go` is a 662-line multi-responsibility
  file — pre-existing, and this branch adds the new assignment/watchdog state to
  it rather than splitting it out. Deliberately not addressed here; splitting it
  would enlarge an otherwise tightly-scoped diff.

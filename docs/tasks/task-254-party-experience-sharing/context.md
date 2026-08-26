# Party EXP Sharing — Execution Context

Companion to [plan.md](plan.md). Read this before dispatching Task 1.

---

## 1. Where the work happens

| | |
|---|---|
| Worktree | `.worktrees/task-254-party-experience-sharing` |
| Branch | `task-254-party-experience-sharing` |
| Primary module | `services/atlas-monster-death/atlas.com/monster` (module name `atlas-monster-death`) |
| Secondary module | `services/atlas-monsters/atlas.com/monsters` |

**The directory trap.** `atlas-monster-death`'s Go module lives at
`services/atlas-monster-death/atlas.com/**monster**/` — singular, no `-death`
suffix. The design doc's change map abbreviates paths as `monster/party/model.go`;
the real path is `services/atlas-monster-death/atlas.com/monster/party/model.go`.
Every path in plan.md is already repo-relative and correct.

Two `monster` levels exist inside that module: the root-level `monster/` package
(the distributor, package `monster`) and `monster/information/`,
`monster/drop/`, `monster/drop/position/` beneath it. The distributor's files
are `monster/processor.go`, `monster/model.go`, and the new
`monster/experience.go`, `monster/interval.go`, `monster/config.go`.

---

## 2. Key files, by what they tell you

### Current behaviour being replaced

- `monster/processor.go:126-141` — `DistributeExperience`: interleaves I/O and
  arithmetic in one loop; solo-only.
- `monster/processor.go:143-205` — `produceDistribution`: holds both `// TODO
  parties` (`:161`) and `// TODO account for healing` (`:173`), and
  `totalDamage := mi.Hp()` (`:174`). **Deleted in Task 12.**
- `monster/processor.go:207-221` — `calculateExperienceStandardDeviationThreshold`.
  **Unchanged (D4).** Already covered by `characterization_test.go`.
- `monster/processor.go:223-229` — `isWhiteExperienceGain`. **Unchanged (D4).**
- `monster/processor.go:231-241` — `distributeCharacterExperience`, whose
  `totalPartyLevel byte` parameter is the latent overflow (D6). **Deleted in Task 12.**
- `monster/model.go:15-37` — `DamageDistributionModel`. **Deleted in Task 12**;
  `ExperiencePlan` supersedes it and carries the `TotalDamage` the PRD asked for.

### Reference implementations to copy from

| Need | Copy from |
|---|---|
| `With(...)` option DI | `services/atlas-pets/atlas.com/pets/pet/processor.go:161-217` |
| party `members` JSON:API relationship | `services/atlas-channel/atlas.com/channel/party/rest.go` (whole file) and `.../party/model.go:13-80` |
| `SHOW_HINT` producer + processor | `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/system_message/producer.go:94-110` and `.../system_message/processor.go:1-60` |
| `SHOW_HINT` struct definitions | `services/atlas-channel/atlas.com/channel/kafka/message/system_message/kafka.go:11-33`, `:62-67` |
| `Processor` + `mock` shape | `services/atlas-monster-death/atlas.com/monster/party/processor.go` and `.../party/mock/processor.go` |
| registry-style aggregation | `services/atlas-monsters/atlas.com/monsters/monster/registry.go:436-495` |
| tenant-in-context test setup | `services/atlas-monsters/atlas.com/monsters/monster/processor_test.go:1904-1905` |

### Consumers and contracts that must not move

- `monster/kafka/consumer/monster/consumer.go:63-89` — the `routine.Go`
  goroutine that calls `DistributeExperience`. Unchanged: `NewProcessor` binds
  the config itself, so nothing threads through the consumer.
- `monster/character/kafka.go` + `monster/character/producer.go` — the
  `AWARD_EXPERIENCE` contract. **Frozen by PRD §2.** Including its known warts:
  no `transactionId`, no `showEffect`, and a `PARTY` distribution with
  `Amount: 0` appended to every solo award. Recorded so a reviewer does not
  rediscover them as regressions introduced here.
- `deploy/k8s/base/atlas-monster-death.yaml:20-22` — already
  `envFrom: configMapRef: atlas-env`, and `env-configmap.yaml:92` already
  defines `COMMAND_TOPIC_SYSTEM_MESSAGE`. **No deployment change for the
  topic** (design §1.4). Task 8 adds only the three new gate keys.

---

## 3. Decisions the plan encodes

Design decisions D1–D16 are in [design.md §3](design.md). Two were open at the
end of design and have been resolved by the user before planning:

- **D12 — adopted.** Out-of-field damagers are never party-resolved. They
  contribute to `totalDamage` and add one each to `totalEntries`, and receive
  nothing. The PRD acceptance bullet reading "still contributes their damage to
  `partyDamage`" is **superseded**: crediting their damage to a party pool would
  require resolving their party, which FR-2.1 forbids, and would open a
  tag-and-walk-away leech vector.
- **D13 — adopted.** MVP is `argmax(damage)` over `expMembers` (a non-damager
  counts as 0), ties to lowest `characterId`. A literal FR-5.1 picks from the
  damager set, which can leave a party with no MVP and silently drop 20% of the
  pool.

### Plan-level choices not spelled out in design.md

1. **Config is loaded inside `NewProcessor`, not threaded from `main.go`.**
   Design §4 listed `main.go` and `kafka/consumer/monster/consumer.go` as
   changed files to carry an `ExperienceConfig` built at bootstrap. The plan
   instead has `NewProcessor` call `LoadExperienceConfig()`, with
   `WithExperienceConfig` as the test override. Rationale: threading it from
   `main.go` would change `InitConsumers`' signature for a value read from
   `os.Getenv`, and the injectable-config property design wanted is fully
   preserved. Net: two fewer files touched, same testability.

2. **`computeAward` clamps as well as guards.** FR-8.6 only requires a
   `NaN`/`Inf` guard. The plan also clamps a value at or above
   `math.MaxUint32`, because `uint32(v)` for an out-of-range `v` is
   implementation-defined in Go exactly as `uint32(NaN)` is. Both cases set the
   same `guarded` return so the caller warns identically.

3. **Builders added where tests need construction.** `party.NewBuilder` /
   `party.NewMemberBuilder` (Task 2) and `rates.NewBuilder` (Task 4) exist
   because both models have unexported fields and no other construction path.
   This is the repo's Builder convention, not a `*_testhelpers.go` file.

4. **The throttle lives in `system_message`, injected into the distributor.**
   D10 left the placement to the plan. A standalone `system_message.Throttle`
   with a process singleton keeps `ShowHint` itself unthrottled — so the
   "publish failure does not abort awards" test and the throttling test do not
   interfere — and lets the distributor inject a fake clock.

5. **`rates/mock` falls through to `rates.Default()`, not the zero value.** A
   zero `rates.Model` has `ExpRate() == 0`, which would silently award nothing
   in any test that forgets to set the func. Every other mock in this service
   falls through to a zero value; this one deliberately does not.

---

## 4. Task dependency order

Tasks 1–8 are independent of each other except that 6 extends the package 5
creates. Tasks 9→10→11→12→13 are strictly sequential.

```
1  atlas-monsters aggregation        (independent — different module)
2  party members + builders  ─┐
3  information Level/Name    ─┤
4  rates + map processors    ─┼──> 11 processor DI ──> 12 rewrite ──> 13 mock tests
5  system_message package ───┤                              ▲
6    └─ throttle           ──┘                              │
7  intervalSet             ─┐                               │
8  ExperienceConfig        ─┼──> 9 plan types + computeAward ┘
                            └──> 10 planDistribution
```

Task 10 depends on 7, 8, and 9. Task 11 depends on 3, 4, 5, 6, 8. Task 12
depends on everything.

Tasks 1–8 could be dispatched in parallel, but 9 and 10 both edit
`monster/experience.go` and 11 and 12 both edit `monster/processor.go` — those
pairs must be serialized.

---

## 5. Task sizing

No task is deliberately oversized. The two largest are:

- **Task 10 (`planDistribution`)** — one function in one file, but a large
  test table. It is the whole of FR-5 and FR-6 and does not decompose into
  independently-reviewable halves: every row of the table exercises the same
  function. Its file count is 2.
- **Task 12 (the rewrite)** — 3 files edited plus 1 test file, one service.
  Within budget.

Tasks 3 and 4 each touch 5–6 files, but every file is the same mechanical
`Processor` + `mock` shape repeated, which is the exception Step 5a allows.

---

## 6. Verification

- Implementers run **module-local** `go build ./... && go test ./...` from the
  module root of whatever they touched. Nothing more.
- Task 6 and Task 13 additionally run `-race` (the throttle mutex and the
  resolve phase's concurrent fetch).
- Repo-wide verification is the controller's gate, after Task 13: flagless
  `tools/verify.sh` must exit 0 — `--quick`/`--no-docker` also exit 0 but skip
  the bake and `-race`, so they do not count.
- `backend-guidelines-reviewer` must pass on both `services/atlas-monster-death`
  and `services/atlas-monsters` before the PR.

## 7. Things that will look like regressions but are not

- **White/yellow EXP changes for multi-hit kills.** `totalEntries` stops
  counting one per damage entry and starts counting one per participant
  (FR-2.5), which shifts the stddev threshold. Deliberate (FR-2.6).
- **Solo EXP numbers change.** `totalDamage` moves from `mi.Hp()` to the sum of
  aggregated entries (FR-3.1), and the full damage is credited rather than the
  last assignment winning (FR-1.4). Both are corrections.
- **A party member whose damage decayed to zero stops being a contributor.**
  `atlas-monsters`' `Registry.DecayDamageEntries` (`registry.go:713-745`) prunes
  entries idle past `AggroIdleThresholdMs`, so such a member neither appears in
  `partyDamage` nor widens the leech interval. Accepted, not worked around —
  it is consistent with how decay already governs drop ownership and control
  (design §1.2).
- **`atlas-monsters`' `AddDamageEntry` change is defence in depth, not the
  enabling fix.** `Model.Damage()` — its only caller — has no production
  caller; the live path is `Registry.ApplyDamage`, which already aggregates
  (design §1.1). Do not let a commit message or PR description claim otherwise.

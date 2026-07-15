# task-136 context — Kafka Consumer Fetch-Wedge

Companion to `plan.md`. Key files, decisions, and gotchas an implementer needs
beyond the plan's step-by-step content.

## Documents

- PRD: `docs/tasks/task-136-consumer-fetch-wedge/prd.md`
- Design: `docs/tasks/task-136-consumer-fetch-wedge/design.md` (Approach A chosen; §1 has file:line-verified facts F1–F10)
- Findings (produced by Tasks 3 and 6): `docs/tasks/task-136-consumer-fetch-wedge/findings.md`

## Key files

| File | Role |
|---|---|
| `libs/atlas-kafka/consumer/manager.go` | Consumer lifecycle, serial/parallel fetch loops, `errFetchWedged`, `Snapshot` — the core of the change |
| `libs/atlas-kafka/consumer/config.go` | Defaults (`maxWait`, `fetchTimeout`, `maxConsecutiveTimeouts`) + decorator API (signatures must not change) |
| `libs/atlas-kafka/consumer/debug.go` | `/debug/consumers` JSON:API serialization of `Snapshot` — additive fields only |
| `libs/atlas-kafka/consumer/manager_test.go` | Mock readers (`scriptedReader`, `ChannelMockReader`, `alternatingReader`, `controlledReader`), `readerFactory` helper, existing wedge tests |
| `libs/atlas-kafka/consumer/offsets_test.go` | The testcontainers pattern the dwell harness copies (`//go:build integration`, `confluentinc/cp-kafka:7.6.0`) |
| `libs/atlas-kafka/consumer/timing_test.go` | NEW (Task 1) — phase-timing test + shared `snapshotForTopic` helper |
| `libs/atlas-kafka/consumer/idle_stuck_test.go` | NEW (Task 4) — `statsStubReader` + classification tests |
| `libs/atlas-kafka/consumer/dwell_integration_test.go` | NEW (Task 2) — S1–S5 dwell scenarios |

## Load-bearing decisions

1. **Root-cause model (design §2):** the dwell driver is H1 — idle wedge →
   `Close()` → `LeaveGroup` → group-wide rebalance, multiplied by 15+ consumers
   sharing one GroupID per service. The per-call deadline itself does NOT
   touch the group session (kafka-go `reader.go:815-838`, hypothesis H3
   refuted in source; S4 is the measured control).
2. **Fix shape (design §3-A):** don't recreate on idle; detect genuine stalls
   by *absence of reader progress* via `Reader.Stats()` deltas. Rejected
   alternatives: per-topic group IDs (offset-replay incident risk, design §3-B)
   and one multi-topic reader via `GroupTopics` (too invasive; recorded as the
   library half of the potential cluster-infra follow-up, §3-C).
3. **Stats() ownership invariant (design R4):** kafka-go `Stats()` returns
   deltas since the previous call. The lib's fetch loop is the ONLY caller on
   lib-owned readers; external telemetry must read `Snapshot()`. S5
   deliberately uses raw `kafka.Reader`s (not the lib) so it may call `Stats()`
   itself.
4. **Legacy fallback:** mock readers that don't implement `StatsProvider` are
   treated as never-progressing, so every deadline tick counts toward the
   wedge threshold — exactly the old behavior. This is why ALL existing unit
   tests must pass unmodified; a broken existing test means the implementation
   is wrong, not the test.
5. **Wedge warn prefix is contract:** `TestFetchTimeoutEscalatesAfterMaxToWedge`
   greps Warn logs for `FetchMessage wedged` + topic + group. The new
   escalation message keeps that prefix and both identifiers.
6. **New defaults:** `maxWait` 10s (kafka-go's own default; `MinBytes=1` means
   zero delivery-latency cost — MaxWait only bounds the *empty* long-poll),
   `fetchTimeout` 1m (liveness tick). Rationale must live as a comment on
   `NewConfig` and in findings.
7. **Investigation-before-fix gate (design R1):** Task 3 runs the harness on
   pre-fix code; S2/S4 failing there IS the reproduction evidence. If S2 does
   NOT fail pre-fix, stop and escalate — do not proceed to the fix.

## Sequencing gotchas

- Between Task 2 and Task 6 the integration scenarios S2/S4 are expected-red.
  That is safe: `//go:build integration` tests are not run by CI or plain
  `go test` (verified — no workflow passes `-tags integration`). Unit suites
  stay green at every commit.
- Task 2's S2 cannot assert `IdleTicks` (field doesn't exist until Task 4);
  Task 6 adds that assertion.
- `snapshotForTopic` lives in `timing_test.go` (untagged) so it is visible to
  both unit and integration test builds of package `consumer_test`.
- The Manager is a process-wide singleton: every test starts with
  `consumer.ResetInstance()`, and integration tests must not use `t.Parallel()`.
- Group join with ~20 members on a fresh testcontainers broker can take tens
  of seconds — the harness warms up with one unmeasured message (120s
  `Eventually`) before measuring.

## Environment / commands

- Unit: `cd libs/atlas-kafka && go test -race ./...`
- Integration (needs local Docker):
  `cd libs/atlas-kafka && go test -tags integration -race -run TestDwell -v -timeout 60m ./consumer/`
- Vet both build modes: `go vet ./...` and `go vet -tags integration ./consumer/`
- Repo invariants from worktree root: `tools/redis-key-guard.sh` (no `GOWORK=off` prefix), `docker buildx bake all-go-services` (shared-lib bump ⇒ bake everything)
- kafka-go pinned at `v0.4.51` (module cache path referenced by design facts:
  `reader.go`, `consumergroup.go`)

## Dependencies

- No new Go dependencies: testcontainers-kafka, testify, logrus test hooks are
  already in `libs/atlas-kafka/go.mod`.
- No service, deploy, or broker-manifest changes. Services pick up the fix on
  the next image build (standard shared-lib bump).

## After implementation

- Code review via `superpowers:requesting-code-review` BEFORE any PR (repo rule).
- PRD §10 requires: if findings show broker topology is a material
  contributor, file the cluster-infra follow-up task (multi-broker / per-env
  topic reduction, optionally design §3-C) citing S5's extrapolated numbers.
  The design §7 decision rule: judge from post-deploy live observation —
  wedge logs gone and saga dwell < 1s ⇒ library fix sufficient.

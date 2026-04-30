# Context — task-039 Kafka FetchMessage Deadline + Tick-and-Escalate

> Quick-reference companion to [plan.md](./plan.md). Read before executing any task.

## Inputs (read these first)

- [prd.md](./prd.md) — locked requirements (file paths, line numbers, defaults, log strings, acceptance criteria).
- [design.md](./design.md) — architecture, state-machine diagram, locked tradeoffs.
- [risks.md](./risks.md) — six risks with mitigations; R2 (kafka-go ctx-cancel) is the one with a real test guard.

## Files in scope

All changes are local to `libs/atlas-kafka/consumer/`.

| File | Reason |
|---|---|
| `libs/atlas-kafka/consumer/manager.go` | Rewrite `runFetchLoop`, add 4 fields to `Consumer`, add `errFetchWedged`, add `recordTimeout`, modify `recordFetch` + `onReaderCreated`, extend `Snapshot` + `Snapshot()`, wire Config fields through `AddConsumer`, drop `retry` import. |
| `libs/atlas-kafka/consumer/config.go` | Add `fetchTimeout` + `maxConsecutiveTimeouts` fields, defaults in `NewConfig`, two new decorators. |
| `libs/atlas-kafka/consumer/debug.go` | Add 2 fields to `debugAttributes` + wire in `snapshotToAttributes`. |
| `libs/atlas-kafka/consumer/manager_test.go` | Fake fidelity fix (3 places: `MockReader`, `ChannelMockReader`, `scriptedReader`) + 3 new tests. |
| `libs/atlas-kafka/consumer/debug_test.go` | Extend in-test `debugAttributes` struct to include the 2 new fields. |

No service-side changes. No Dockerfile changes. No env or manifest changes.

## Locked decisions (do not re-litigate)

| Decision | Choice | Source |
|---|---|---|
| Detection mechanism | Per-call deadline on `FetchMessage` | PRD §1, §4.1 |
| Default `fetchTimeout` | `5 * time.Minute` | PRD §4.5 |
| Default `maxConsecutiveTimeouts` | `3` | PRD §4.5 |
| Inner `retry.Try` block | Removed entirely | PRD §4.3 |
| Sentinel export | Unexported `errFetchWedged` | PRD §4.4 |
| Sentinel file location | `manager.go` | design §4 |
| Counter reset sites | `recordFetch` + `onReaderCreated(attempt>0)` only | design §3.3 |
| Backoff reset on wedge recreate | None — existing 10s cap stands | design §3.4 |
| Tick log level | `Debug` | PRD §4.2 |
| Wedge log level | `Warn` (one-shot per wedge) | PRD §4.2 |
| Counter visibility | `Snapshot` + `/api/debug/consumers` | PRD §4.5, §5.2 |
| Process restart on wedge | None | PRD §2 non-goals |
| Existing-test scaffolding | Untouched (none need `SetFetchTimeout`) | design §9.3 |
| Test-fake `ctx.Err()` fix | Applied (3 fakes, 1 line each) | design §9.1 |

## Key invariants (will break things if violated)

1. **Eager `cancel()`** — call `cancel()` immediately after `FetchMessage` returns, never via `defer`. Deferred cancel inside an unbounded loop leaks one timer per iteration. Test 2 has a `runtime.NumGoroutine()` guard that catches violations.
2. **Counter authority** — `consecutiveTimeouts` lives on the `Consumer` struct (mutex-guarded), not just function-local in `runFetchLoop`. The function-local view is a convenience mirror; source of truth is `c.consecutiveTimeouts`.
3. **Idle ≠ error** — `recordTimeout` does NOT touch `lastError` / `lastErrorAt`. Only the wedge escalation (after the sentinel reaches the outer loop's `recordError`) writes to `lastError`.
4. **Counter reset is structural, not procedural** — `runFetchLoop` does NOT explicitly reset on non-deadline error. The reset happens in `onReaderCreated(attempt>0)` because every non-cancel error from `runFetchLoop` triggers an outer-loop reader recreate.

## State machine — `runFetchLoop` (locked from design §3)

```
                    enter loop
                        │
                        ▼
                ctx cancelled? ──yes──▶ return ctx.Err()
                        │no
                        ▼
        fetchCtx, cancel = WithTimeout(ctx, fetchTimeout)
        msg, err = reader.FetchMessage(fetchCtx)
        cancel()  ← eager, NOT deferred
                        │
        ┌───────────────┼────────────────────┐
        │               │                    │
     err == nil    DeadlineExceeded      other err
                  (parent ctx alive)         │
        │               │                    │
        ▼               ▼                    ▼
   recordFetch     recordTimeout       return err
   process(msg)    counter++           (transport err,
   commit          ┌─────┴──────┐       EOF, Canceled →
   continue       counter      counter outer recreate)
                   < max?       >= max?
                    │              │
                    ▼              ▼
                continue        log Warn,
                loop            return errFetchWedged
```

**Branch table:**

| Branch | Test | Action | Counter | Returns? |
|---|---|---|---|---|
| Success | `err == nil` | `recordFetch` (resets counter, clears lastError); `processMessage`; commit | reset to 0 | no, continue |
| Idle tick | `errors.Is(err, DeadlineExceeded) && ctx.Err() == nil && counter+1 < max` | `recordTimeout`; Debug log | counter++ | no, continue |
| Wedge escalate | `errors.Is(err, DeadlineExceeded) && ctx.Err() == nil && counter+1 >= max` | `recordTimeout`; Warn log | counter++ | yes, `errFetchWedged` |
| Parent cancel | `ctx.Err() != nil` (any err) | none | unchanged | yes, `ctx.Err()` |
| Other error | otherwise | none | unchanged | yes, `err` |

## Log strings (verbatim from PRD §4.2)

- Tick (Debug): `"FetchMessage deadline expired (consecutive=%d/%d); ticking."`
- Wedge (Warn): `"FetchMessage wedged: %d consecutive timeouts on topic [%s] (group [%s]); forcing reader recreate."`
- Sentinel message: `"consumer fetch wedged: exceeded consecutive timeouts"` — surfaces in `lastError` automatically through the outer loop's existing `c.recordError(err)`.

## Test fake fidelity fix (design §9.1)

Three fakes return the literal `context.Canceled` after `<-ctx.Done()`. Real kafka-go returns `ctx.Err()` (`DeadlineExceeded` on timeout, `Canceled` on cancel). The new state machine distinguishes them.

| Fake | Location | Fix |
|---|---|---|
| `MockReader.FetchMessage` | manager_test.go:32-40 | `return kafka.Message{}, ctx.Err()` |
| `ChannelMockReader.FetchMessage` | manager_test.go:117-124 | `return kafka.Message{}, ctx.Err()` |
| `scriptedReader.FetchMessage` | manager_test.go:443-461 | `return kafka.Message{}, ctx.Err()` |

This is a fidelity improvement, not a behavior change for existing tests — they only ever cancel the parent ctx, and `ctx.Err()` returns `Canceled` in that path (same value as the literal).

## New tests (design §9.2)

All three use `consumer.SetFetchTimeout(50*time.Millisecond)` and `consumer.SetMaxConsecutiveTimeouts(3)`. Wall-clock <250ms each.

1. **`TestFetchTimeoutTicksWithoutRecreate`** — empty `scriptedReader` (always blocks on ctx). Wait ~75ms. Assert `Snapshot.ConsecutiveTimeouts >= 1`, `RecreateCount == 0`, `LastError == ""`, `LastTimeoutAt` non-zero. Cancel ctx, assert reader closed once.
2. **`TestFetchTimeoutEscalatesAfterMaxToWedge`** — two readers via `readerFactory`: r1 empty, r2 delivers a message. Wait for handler invocation. Assert `r1.Closes() == 1`, `RecreateCount >= 1`, `LastError == "consumer fetch wedged: exceeded consecutive timeouts"`, `ConsecutiveTimeouts == 0`. **Goroutine-leak guard**: capture `runtime.NumGoroutine()` before and ~50ms after handler invocation; delta bounded.
3. **`TestFetchTimeoutResetsOnSuccessfulFetch`** — reader alternates: blocks-until-deadline, delivers message, repeats. ~6 iterations (3 timeouts/3 successes). Assert `RecreateCount == 0`, final `ConsecutiveTimeouts == 0`.

## Verification commands

After all tasks:

```bash
# Library build + tests
cd libs/atlas-kafka && go build ./... && go test ./consumer/... -race -count=1
cd libs/atlas-kafka && go vet ./...

# Workspace build
cd /home/tumidanski/source/atlas-ms/atlas && go build ./...
go vet ./libs/atlas-kafka/... && go vet ./services/...

# Docker builds for primary affected services (CLAUDE.md mandates this for shared-lib changes)
docker build -f services/atlas-maps/Dockerfile .
docker build -f services/atlas-monsters/Dockerfile .
docker build -f services/atlas-channel/Dockerfile .
```

## Service impact (no code changes)

The following 49 consumer-owning services pick up new behavior on Docker rebuild only:

`atlas-account, atlas-asset, atlas-barbarian, atlas-buddylist, atlas-cashshop, atlas-chairs, atlas-channel, atlas-character, atlas-character-presets, atlas-character-stat-bonus, atlas-compartment, atlas-data, atlas-drops, atlas-employees, atlas-equipables, atlas-events, atlas-friend-management, atlas-game-rng, atlas-guild, atlas-inventory, atlas-keymap, atlas-magicsword, atlas-marriage, atlas-maps, atlas-messages, atlas-monsters, atlas-mts, atlas-npcs, atlas-parties, atlas-portals, atlas-pets, atlas-quest, atlas-quick-access, atlas-recommendations, atlas-reports, atlas-saga-orchestrator, atlas-shops, atlas-skills, atlas-spell-cores, atlas-storage, atlas-tenants, atlas-titles, atlas-tradehouse, atlas-tradehouse-history, atlas-transports, atlas-ui, atlas-warps, atlas-weddings, atlas-world.`

Per CLAUDE.md, shared-library changes require Docker build verification. Plan covers atlas-maps, atlas-monsters, atlas-channel as the minimum set per PRD §10 acceptance criteria.

## Workflow rules (from CLAUDE.md + memory)

- Never commit directly to `main`. Branch protection blocks pushes; this work belongs on `task-039-kafka-fetch-deadline` branch.
- Run `superpowers:requesting-code-review` BEFORE opening a PR, not after.
- Plan/audit artifacts go under `docs/tasks/task-039-kafka-fetch-deadline/`, not `docs/superpowers/plans/`.

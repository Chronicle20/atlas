# Backend Audit — atlas-mini-games (Phase-1 fleet-pattern adoption)

- **Service Path:** services/atlas-mini-games/atlas.com/mini-games
- **Scope:** commit b38565315 (`feat(task-133): adopt fleet patterns in atlas-mini-games`) + full `git diff main..HEAD` over services/atlas-mini-games
- **Guidelines Source:** backend-dev-guidelines skill (ai-guidance, file-responsibilities, anti-patterns, testing-guide, patterns-provider/resilience/rest-jsonapi/functional/multitenancy)
- **Date:** 2026-07-16
- **Build:** PASS (`go build ./...` clean)
- **Vet:** PASS (`go vet ./...` clean)
- **Goroutine guard:** PASS (`tools/goroutine-guard.sh` exit 0; no bare `go` statements)
- **Tests:** PASS — `go test -race ./... -count=1` clean, all packages `ok`
- **Overall:** NEEDS-WORK (build/tests green, but FAIL-level findings below)

Mindset: FAIL until proven PASS. Every item below is either a cited PASS or a cited FAIL — nothing is graded on "the rest of the repo does this."

---

## Focus Area 1 — Transactional-outbox re-architecture (`game/processor.go`)

### Mechanism itself: sound

- `emit()` (processor.go:199-206) wraps the whole command in `database.ExecuteTransaction(p.db.WithContext(p.ctx), ...)` and enqueues the buffered events via `outbox.EmitProvider` inside that same tx — correct task-114 shape.
- `withTx()` (processor.go:213-217) is a safe shallow copy: only `db` is replaced; `reg` (registry pointer), `cp`/`mp`/`ip`/`chp` (REST-client seams), and `now`/`rng` are intentionally shared, not tx-scoped. This is correct — the registry is explicitly in-memory/non-transactional.
- `message.Emit` (kafka/message/message.go:35-50) only publishes to the outbox after the whole closure `f(b)` returns nil — a mid-command error means **zero** events are buffered/published. No double-emit hazard from the outbox path itself.
- Nested `database.ExecuteTransaction` calls correctly join the caller's tx (`libs/atlas-database/transaction.go:9-14`, `isTransaction` check on `Statement.ConnPool`), so `record.ApplyResult`'s own `ExecuteTransaction` call inside `endGame` rides the same outer tx rather than opening a second one.

### FINDING 1 (Important) — `endGame`'s "ApplyResult-before-swap" invariant only covers the WRITE, not the two READS that follow the swap

`game/processor.go:983-1034`. The documented invariant (processor.go:965-982) is: *"record.ApplyResult runs FIRST in code order, before the registry swap... If ApplyResult fails the room is left untouched."* That's true for the write:

```
998   if err := record.ApplyResult(p.db.WithContext(p.ctx), gameType, ownerId, visitorId, winnerSlot, tie); err != nil {
999       return err
1000  }
1001
1002  updated, err := p.reg.Update(p.t, roomId, func(cur Room) (Room, error) {   // <-- in-memory swap, immediately visible, NOT rolled back
1003      return resolvedRoom(cur, resultType, winnerSlot, now), nil
1004  })
...
1008  ownerRecord, err := record.GetOrZero(p.ctx, p.db.WithContext(p.ctx), ownerId, gameType)   // <-- fallible DB READ, AFTER the swap
1009  if err != nil {
1010      return err        // aborts the whole outer tx -> record.ApplyResult's writes ROLL BACK
1011  }
1012  visitorRecord, err := record.GetOrZero(p.ctx, p.db.WithContext(p.ctx), visitorId, gameType)
1013  if err != nil {
1014      return err
1015  }
```

If either `GetOrZero` call at line 1008 or 1012 fails (a real possibility — it's a live DB read, not a pure function), the outer `ExecuteTransaction` rolls back, **undoing `ApplyResult`'s Wins/Losses/Ties writes** — but the registry swap at line 1002-1007 already happened in memory and is **not** rolled back: the room is left with `InProgress=false`, board/deck wiped, session scores updated, `FirstMover` advanced. No `GAME_ENDED` or `BALLOON_UPDATED` event is ever emitted (the buffer aborts). The result: a room silently and permanently "ends" in memory with no persisted W/L/T record and no notification to either client, until the process restarts and the whole (unpersisted) registry is discarded. This is *worse* than the hazard the code's own comment claims to guard against — a swap with neither a persisted record nor an emitted event.

**Test coverage gap confirms this is unguarded:** the only test that exercises this invariant, `TestEndGame_ApplyResultFailureLeavesRoomInProgress` (`game/processor_test.go:1380-1410`), injects failure via a sqlite `BEFORE INSERT` trigger on `game_records` — i.e. it only tests the write path (line 998) failing. It does not, and cannot with that technique, exercise a failure in the two post-swap `GetOrZero` reads (lines 1008/1012). No test drives that branch.

**Fix shape:** fetch `ownerRecord`/`visitorRecord` (or read them from the same `getOrCreate` calls `ApplyResult` already performs internally) *before* the registry swap, so any DB-read failure aborts before touching in-memory state — mirroring the discipline already applied to the write.

### FINDING 2 (Important) — `create()` and `visit()` have the same ordering hazard: registry mutation precedes fallible DB reads

`game/processor.go:300-326` (`create`) and `game/processor.go:342-401` (`visit`).

`create()`:
```
300   if _, ok := p.reg.GetByMember(p.t, characterId); ok { ... }
...
313   if err := p.reg.Create(p.t, room); err != nil { ... }     // <-- registry mutation, visible immediately
...
317   ownerRecord, err := record.GetOrZero(p.ctx, p.db.WithContext(p.ctx), characterId, gameType)  // <-- fallible read AFTER mutation
318   if err != nil {
319       return err       // room stays registered; no CREATED and no CREATE_ERROR event ever reaches the client
320   }
```

`visit()` is the same shape: `p.reg.Update(...)` seats the visitor at line 374-383, and the two `record.GetOrZero` reads that can fail follow at lines 388 and 392.

Consequence: if the `GetOrZero` read fails after the registry mutation, the character is left registered as a room owner (`create`) or seated as a visitor (`visit`) with **no event of any kind emitted** (not even a `*_ERROR` event — the function returns a bare `err`, which only aborts the tx). The character is now stuck: `GetByMember` will find them "already in a room" on every future `Create`/`Visit` attempt, which today silently returns `errUnable`/`errCannotOpenMiniRoomHere`-style events forever, with no way out short of an explicit `Leave`/`Expel`/`TeardownCharacter` or a process restart. This is the identical class of defect as Finding 1, just in the CREATE/VISIT ladders instead of `endGame`.

**Fix shape:** same as Finding 1 — perform every fallible DB read before the registry mutation in both `create()` and `visit()`.

### FINDING 3 (Important) — the one consumer that would surface Findings 1/2 discards every processor error silently

`kafka/consumer/minigame/consumer.go:95-273` — all 17 command handlers (`handleCreate`, `handleVisit`, `handleLeave`, `handleChat`, `handleExpel`, `handleReady`, `handleUnready`, `handleStart`, `handleMoveStone`, `handleFlipCard`, `handleRequestTie`, `handleAnswerTie`, `handleGiveUp`, `handleRequestRetreat`, `handleAnswerRetreat`, `handleSkip`, `handleExitAfterGame`, `handleCancelExitAfterGame`) use the identical pattern, e.g. line 101:

```go
_ = game.NewProcessor(l, ctx, db).Create(c.TransactionId, f, c.CharacterId, c.Body.RoomType, c.Body.Title, c.Body.Private, c.Body.Password, c.Body.PieceType)
```

`message.Handler[M]` (`libs/atlas-kafka/message/handler.go:21`) is `func(l logrus.FieldLogger, ctx context.Context, m M)` — **no error return** — so this `_ = ...` is the only place in the call chain the error could be logged, and it isn't. Every outbox-tx failure, including the ones identified in Findings 1 and 2, vanishes with zero log line, zero metric, zero trace.

This is proven inconsistent within the same branch: the sibling teardown consumers correctly log:
- `kafka/consumer/session/consumer.go:51-53` — `if err := game.NewProcessor(l, ctx, db).TeardownCharacter(e.CharacterId); err != nil { l.WithError(err).Errorf(...) }`
- `kafka/consumer/character/consumer.go:54-56, 69-71, 89-91` — same pattern, three times.

The minigame command consumer is the odd one out, and it's the consumer that carries the highest-value/most-frequent traffic (every lifecycle and gameplay command).

**Fix shape:** `if err := ...; err != nil { l.WithError(err).Errorf("...") }` in all 17 handlers, matching the sibling consumers.

---

## Focus Area 2 — `service.Bootstrap` adoption (main.go)

PASS. `main.go:47` — `rt := service.Bootstrap(serviceName)`. No local `logger/` or `kafka/producer/` package exists under `services/atlas-mini-games/atlas.com/mini-games` (confirmed via directory listing — none present; the diff never introduced them). Shared producer repoint: `main.go:100` — `rt.TeardownFunc(func() { _ = producer.GetManager().Close(l) })` uses the shared `libs/atlas-kafka/producer` manager, no service-local producer wrapper. Outbox drainer goroutine correctly spawned via `routine.Go(l, rt.Context(), ...)` (main.go:70-72), not a bare `go` statement — confirmed by `tools/goroutine-guard.sh` exit 0 and a direct grep for bare `go` statements (none found).

---

## Focus Area 3 — 503 resilience / pagination / ExecuteTransaction / Gen3

| Item | Status | Evidence |
|---|---|---|
| `RegisterTransientErrorClassifier` wired in main.go | PASS | `main.go:53-59`, composes `database.IsTransientConnectionError` + `database.CountTransient` per patterns-resilience.md |
| Handlers use `server.WriteErrorResponse` not bare 500 | PASS | `record/resource.go:50,64`; `game/resource.go:74` |
| GET collections paginate | PASS | `record/resource.go:41,59,70` (`paginate.ParseParams` → `paginate.Slice` → `server.MarshalPaginatedResponse`); `game/resource.go:54,69,80` same shape |
| `record/administrator.go` `ApplyResult` uses `database.ExecuteTransaction` | PASS | `record/administrator.go:54-80`; correctly joins the caller's tx via `isTransaction` (`libs/atlas-database/transaction.go:10-12`) so the outbox tx and the two-row upsert commit atomically |
| Gen3 (`var _ Processor` + `mock/` package) for all 6 processor packages | PASS | `data/character/processor.go:32`, `data/map/processor.go:32`, `data/inventory/processor.go:29`, `data/chalkboard/processor.go:27`, `game/processor.go:158`, `record/processor.go:34` — each has a matching `mock/processor.go` with function-field mocks and a top-level `var _ X.Processor = (*ProcessorMock)(nil)` assertion (verified `game/mock/processor.go:173`, `record/mock/processor.go:18`) |

No findings in this focus area.

---

## Focus Area 4 — Test wiring

| Item | Status | Evidence |
|---|---|---|
| `outbox.Migration` in the two test DBs that drive command methods | PASS | `game/processor_test.go:93`; `kafka/consumer/character/consumer_test.go:50` |
| `database.RegisterTenantCallbacks` in test setup (not main.go) | PASS | `game/processor_test.go:73`; `kafka/consumer/character/consumer_test.go:34`; `record/administrator_test.go:31` — and correctly absent from `main.go` |
| DOM-24 Kafka producer stub | PASS | `game/testmain_test.go:11` and `kafka/consumer/character/testmain_test.go:11` both call `producertest.InstallNoop()`; no service-local no-op writer; no `t.Cleanup(producer.ResetInstance)` misuse found anywhere in the service |
| tx-scoped harness (`newHarness`/`setupTestDB`) | PASS | `game/processor_test.go:66-141` — builds a fresh tenant per test (registry isolation) and a fresh sqlite DB with tenant callbacks + outbox migration |

No findings in this focus area.

---

## Additional findings (outside the four numbered focus areas, caught in the full-service sweep)

### FINDING 5 (Important) — DOM-25: client-interpreted wire bytes emitted as Go literals by a domain service, then forwarded unresolved by the channel

`game/processor.go:45-50` (`resultWin`/`resultTie`/`resultForfeit`, explicitly documented at line 45 as "`§G5 mode-62 RESULT`" — i.e. the client's own wire mode byte) and `game/processor.go:73-78` (`leaveStatusClosed`/`leaveStatusLeft`/`leaveStatusExpelled` = 3/4/5) are put directly onto Kafka event bodies (`GameEndedEventBody.ResultType`, `LeftEventBody.Status`, `RoomClosedEventBody.VisitorStatus`, and `CardFlippedEventBody.ResultType` via `matchcards.FlipResultType`) by the **mini-games domain service**, not by atlas-channel.

Per `anti-patterns.md` ("Hardcoding client-interpreted wire values") and the DOM-25 checklist: *"Domain services (non-channel) emit SEMANTIC keys (strings), not client bytes... a byte field carrying a client code in a Kafka event produced by a domain service is a finding."* This is precisely that shape — contrast with the CREATE/VISIT validation ladder in the same file, which correctly emits semantic string keys (`errNotWhenDead = "NOT_WHEN_DEAD"`, etc., `processor.go:64-71`).

Confirmed end-to-end: atlas-channel forwards these bytes with **no** tenant-table resolution (no `WithResolvedCode`):
- `services/atlas-channel/atlas.com/channel/kafka/consumer/minigame/consumer.go:260` — `interactioncb.CharacterInteractionLeaveBody(e.Body.Slot, e.Body.Status)`
- `.../consumer.go:276` — `interactioncb.CharacterInteractionLeaveBody(1, e.Body.VisitorStatus)`
- `.../consumer.go:430` — `interactioncb.CharacterInteractionMiniGameResultBody(e.Body.ResultType, e.Body.WinnerSlot == 1, ownerRecord, visitorRecord)`
- `.../consumer.go:369` — `interactioncb.CharacterInteractionMiniGameCardSelectSecondBody(e.Body.Slot, e.Body.FirstSlot, e.Body.ResultType)`

Contrast with the correctly-built enterError path in the same consumer file, documented at `consumer.go:204` as *"resolved to the per-version"* key — i.e. the pattern this feature should have followed for these byte fields too. This mirrors the established `leaveReason`/`NoticeFailReason` precedent (personal-shop LEAVE reason codes, task-102/103): "version-stable" is explicitly not an exemption per the guideline.

**Fix shape:** either (a) mini-games emits semantic string keys for result/leave-status and the channel resolves them via a tenant writer-options table seeded into every supported version's template, or (b) if these bytes are genuinely dispatcher-internal and never routed through a per-version lookup on the real client, that needs to be established via IDA and documented as the deviation — it is not currently documented anywhere in this diff.

### FINDING 6 (Important, mechanical) — Cosmic source citations in code comments violate the project's explicit house rule

Project memory (`feedback_no_cosmic_in_code_comments.md`, referenced from CLAUDE.md context): *"No Cosmic citations in code comments — reference, not source of truth; cite IDA/WZ instead; scrub files you touch."* This diff introduces 19 Cosmic/`.java` citations across 8 files (non-test and test):

- `game/processor.go` — 9 instances: lines 30, 412, 542, 762, 815, 905, 961, 977, 1037 (e.g. line 30: `"(design §3.3 / Cosmic MiniGame.java:240-298, 5-minute tie-score cooldown)"`; line 815: `"PlayerInteractionHandler.java:411-425"`)
- `game/builder.go` — 2 instances (line 19-20: `"Cosmic MiniGame.java:52"`)
- `game/producer.go` — 1 instance (line 173)
- `game/matchcards/engine.go` — 3 instances (lines 6, 40-41)
- `game/omok/engine.go` — 1 instance (line 6, full path `<cosmic>/src/main/java/server/maps/MiniGame.java:431-516`)
- `game/processor_test.go`, `game/omok/engine_test.go`, `game/matchcards/engine_test.go` — 1 each

This is a mechanical, unambiguous violation of stated house style (not a judgment call) — every one of these needs to be rewritten to cite the IDA/WZ evidence directly (the design docs already do this in `ida-notes.md` per the branch's own commit history; the code comments should point there or restate the IDA fname/address instead of the Cosmic path).

### FINDING 7 (Minor) — EXT-03: `chalkboard.HasOpen` collapses every error into "false", not just 404

`data/chalkboard/processor.go:29-37`:
```go
// HasOpen fetches the character's chalkboard; a 404 (or any fetch failure)
// means there is no open chalkboard, so the check does not block the command.
func (p *ProcessorImpl) HasOpen(characterId uint32) (bool, error) {
    _, err := requestById(characterId)(p.l, p.ctx)
    if err != nil {
        return false, nil
    }
    return true, nil
}
```

This treats a genuine 404 (no open chalkboard — correct) identically to a transport failure, timeout, 5xx, or decode error from the chalkboards service (incorrect — should bubble up). EXT-03 requires: *"Only genuine 404s map to a domain-level 'not found' error; transport / decode / 5xx failures bubble up with their original error. Surfacing every error as 'not found' hides deploy bugs."* Practical impact: if atlas-chalkboards is down, `HasOpen` fail-opens to `false`, silently letting CREATE/VISIT through instead of surfacing an error (which callers, per Finding 3, would silently drop anyway). Contrast with `data/character`, `data/map`, `data/inventory` processors in the same diff, which all propagate the raw error (`return 0, err` / `return false, err`).

**Fix shape:** distinguish `requests.ErrNotFound` (→ `false, nil`) from every other error (→ `false, err`), matching the sibling processors.

### Note (non-finding) — DOM-05 `TransformSlice`

Neither `game/rest.go` nor `record/rest.go` defines a literal `TransformSlice` function; both resource handlers instead compose `model.SliceMap(Transform)(model.FixedProvider(...))(...)` (`game/resource.go:71`, `record/resource.go:61`). This is explicitly documented as the canonical alternative in `ai-guidance.md`'s "Useful Composition" section (lines 226-237: `res, err := ops.SliceMap(Transform)(ops.FixedProvider(models))(ops.ParallelMap())()`), so this is graded PASS via a documented alternate pattern — not a repo-convention rationalization.

---

## Summary

### Blocking (must fix)
- **Finding 1** — `game/processor.go:1002-1015`: `endGame`'s two post-swap `record.GetOrZero` reads can fail and roll back `ApplyResult` while the in-memory registry swap survives — stranded, unpersisted, unnotified game-end. Untested branch.
- **Finding 2** — `game/processor.go:313-320` (`create`) and `:374-395` (`visit`): registry mutation precedes fallible `record.GetOrZero` reads; a read failure strands a registered/seated character with zero notification.
- **Finding 3** — `kafka/consumer/minigame/consumer.go:101-271`: all 17 command handlers discard processor errors via `_ = ...` with no logging, unlike the sibling `session`/`character` consumers — Findings 1/2 would be completely invisible in production.
- **Finding 5** — DOM-25: `resultType`/leave-status bytes emitted as raw client wire values by the mini-games domain service, then forwarded unresolved by atlas-channel (`consumer.go:260,276,369,430`) with no tenant writer-options table, unlike the correctly-built `enterError` path in the same file.
- **Finding 6** — 19 Cosmic/`.java` source citations across 8 files violate the explicit project house rule against citing Cosmic as source-of-truth in code comments.

### Non-Blocking (should fix)
- **Finding 7** — `data/chalkboard/processor.go:31-37` `HasOpen` fail-opens on every error type, not just 404 (EXT-03).

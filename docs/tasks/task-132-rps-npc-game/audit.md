# Backend Audit — atlas-rps / task-132 RPS NPC minigame

- **Service Path:** `services/atlas-rps`, `services/atlas-channel`, `services/atlas-tenants`, `services/atlas-npc-conversations`, `services/atlas-saga-orchestrator`, `libs/atlas-packet/rps`, `libs/atlas-saga`
- **Guidelines Source:** backend-dev-guidelines skill (DOM-01..28, FILE-01..06, SUB-01..04, EXT-01..04)
- **Date:** 2026-07-17
- **Diff Scope:** `git diff c9490b72488bb446f15a4a44156563becab722b3 a1322a6a2a2e9a00b26f735ccea6e734f842829f -- '*.go'` (87 files, +9502/-1)
- **Build:** PASS (all 7 modules: `atlas-rps`, `atlas-channel`, `atlas-tenants`, `atlas-npc-conversations`, `atlas-saga-orchestrator`, `libs/atlas-packet`, `libs/atlas-saga`)
- **Tests:** PASS (all packages `ok`, zero `FAIL` across all 7 modules, `go test ./... -count=1`)
- **Overall:** NEEDS-WORK

## Backend guidelines review (round-loop + config)

*(This file did not previously exist in this task folder; the two related documents `audit-backend.md` and `audit-backend-recheck.md` are prior backend-guidelines passes from earlier in the branch's history — see the note below. Everything in this document is this review's own findings.)*

## Note on prior audit coverage

Two prior backend audits exist in this task folder: `audit-backend.md` (full-branch, at commit `7ae96ef4b`) and `audit-backend-recheck.md` (same base, confirms `IMP-1`/`IMP-2` pagination + error-response fixes landed). This document supplements those — it does not repeat their confirmed-PASS findings, and focuses on:

1. Independent verification of the highest-risk round-loop/consolation logic per this review's brief.
2. **Commits landed after `audit-backend-recheck.md`'s scope** — `7ae96ef4b..a1322a6a2` (`ea6ff6a2c` "streak certificates, loss consolation, and Retry restart", `e6845a1e0`, `b775bc012`, `92d85bf1b`, `3745d5c6b`, `78b6cafb6`, `db6a5e773`, `96ba615d0` and the reward-ladder `default.json` change bundled in `ea6ff6a2c`) — **none of this code has been through a backend-guidelines pass before now.**

## Build & Test Results

All modules built and tested clean from their own module root:

| Module | `go build ./...` | `go test ./... -count=1` |
|---|---|---|
| `services/atlas-rps/atlas.com/rps` | PASS | PASS (all packages `ok`) |
| `services/atlas-channel/atlas.com/channel` | PASS | PASS (all packages `ok`) |
| `services/atlas-tenants/atlas.com/tenants` | PASS | PASS (all packages `ok`) |
| `services/atlas-npc-conversations/atlas.com/npc` | PASS | PASS (all packages `ok`) |
| `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator` | PASS | PASS (all packages `ok`) |
| `libs/atlas-packet` | PASS | PASS (`rps/clientbound`, `rps/serverbound` `ok`) |
| `libs/atlas-saga` | PASS | PASS |

`tools/goroutine-guard.sh` from repo root: exits 1 under `GOWORK=off`, but the failure is a **pre-existing, unrelated environmental gap** — traced to `goroutineguard: ./... matched no packages` for `atlas-monster-book`, `atlas-doors`, `atlas-mts`, `atlas-mounts` (none touched by this diff; confirmed by re-running the built `goroutineguard` binary per-module with and without `GOWORK=off`). Manual sweep of every RPS-touched package (`grep -rnE '^\s*go (func|[A-Za-z_])'` excluding `_test.go`) returns **zero** matches; `services/atlas-rps/atlas.com/rps/main.go:55` and `services/atlas-rps/atlas.com/rps/tasks/task.go:19` both spawn via `routine.Go(l, ctx, ...)`. **DOM-26: PASS** for this diff's scope.

## Findings — Important

### IMP-1 (NEW, post-recheck): Unverified item IDs shipped in the `rps-rewards` default seed, contradicting the task's own verification doc and CLAUDE.md's grounding rule

`services/atlas-tenants/configurations/rps-rewards/default.json:8-17` (landed in commit `ea6ff6a2c`, part of this diff's scope) now ships:

```json
{ "rung": 1, "itemId": 4031332, "quantity": 1, "meso": 0 },
...
{ "rung": 10, "itemId": 4031341, "quantity": 1, "meso": 0 }
```

Ten sequential item ids (`4031332`–`4031341`) with **zero verification trail** anywhere in the repo — `grep -rn "4031332\|4031341"` across `*.md`/`*.json`/`*.go` returns only the seed file itself and three Go test-fixture literals (`services/atlas-tenants/atlas.com/tenants/configuration/rest_test.go:16`, `services/atlas-rps/atlas.com/rps/game/processor_test.go:399,506,630`) that simply reuse the seed's value, not an independent verification.

This directly contradicts `docs/tasks/task-132-rps-npc-game/reward-ladder.md`, a document this same task branch committed specifically to record that **no item ids would be shipped**:

> "**The `rps-rewards` config ships an operator-tunable, meso-only ladder.** No item-reward entries are seeded, because neither the authentic Cosmic reward set nor an item-id verification path was available in this execution environment, and the project rule **"Do not ship an unverified item id"** (CLAUDE.md → Grounding & Honesty) is binding." (`reward-ladder.md:5-8`, still committed, unmodified, at HEAD)

`reward-ladder.md`'s own documented seed table (`reward-ladder.md:12-17`) still shows `itemId: 0 (none)` for every rung — the document was never updated when `default.json` was changed to carry real item ids. `docs/tasks/task-132-rps-npc-game/verification.md:135-142` (parked-follow-ups item 4) repeats the same "meso-only, no item rewards seeded" claim and is now stale/contradicted by the shipped config.

Per CLAUDE.md → "Verification Over Memory": *"For game data values (props, item IDs, skill effects, WZ data), always verify against local WZ data or repo source. Do not cite values from general MapleStory knowledge or memory."* and → "Grounding & Honesty (No Inventing)": *"Never invent values, names, opcodes, command output, or behavior... say 'unknown / unverified'."* No WZ/atlas-data verification of items `4031332`–`4031341` is recorded anywhere in this branch's docs (`ida-rps-clientbound.md`, `ida-rps-serverbound.md`, `ida-rps-legacy-reaudit.md`, `reward-ladder.md` — none mention these ids).

**Failure scenario:** if any of `4031332`–`4031341` is not a real/appropriate item id in the live game's `Item.wz`/`atlas-data`, every tenant that seeds the default `rps-rewards` config grants players an invalid or wrong item on a win — silently, in production, with no operator warning, because the seed ships as the *default* (not opt-in).

**Severity: Important.** This is a policy violation with direct player-facing/data-integrity consequences, not a style nit — and it reverses a decision this task branch itself documented as binding two commits earlier in the same PR.

### IMP-2 (NEW, post-recheck): `Retry`'s fee-deduction saga failure is swallowed and the round proceeds unpaid — asymmetric with the payout path's fail-safe behavior

`services/atlas-rps/atlas.com/rps/game/processor.go:534-539`:

```go
// Re-charge the participation fee (best-effort - see the method doc).
if ladder.EntryCostMeso > 0 {
    if serr := p.sagaSubmitter(buildFeeDeductionSaga(m, ladder.EntryCostMeso)); serr != nil {
        p.l.WithError(serr).Warnf("Unable to submit RPS retry fee deduction for character [%d]; restarting anyway.", characterId)
    }
}

updated, err := CloneModelBuilder(m).SetRung(0).SetStatus(StatusAwaitingSelect).Build()
```

If `p.sagaSubmitter(...)` returns an error (saga-orchestrator unreachable, Kafka producer error, transient failure), the error is logged at `Warn` and **discarded** — execution falls through unconditionally to rebuild the session at rung 0 / `StatusAwaitingSelect` and buffer a `RoundStarted` event (`processor.go:547`), i.e. the round restarts regardless of whether the fee was ever charged.

This is the exact "money-path retry-safety" failure mode this review was asked to check for, and it is **inconsistent with this same file's own payout path**: `Collect`'s win branch (`processor.go:606-619`) returns the error immediately on a saga-submit failure — `if err := p.sagaSubmitter(s); err != nil { return Model{}, err }` — *before* `GetRegistry().Remove` and *before* the `GameEnded` event is buffered, deliberately leaving the session untouched so a retried `Collect` can attempt the payout again (per the method's own doc comment at `processor.go:574-575`: *"If saga submission fails, the session is left in place... so a retried Collect can attempt the payout again"*). `Retry`'s fee-deduction path does the opposite: on the identical class of failure, it proceeds anyway.

The method doc (`processor.go:512-515`) only discloses the *insufficient-funds* gap as a known follow-up ("Retry does not pre-check the player's balance... A balance-gated retry... is a documented follow-up") — it does not disclose or justify the separate, more serious gap that even a *failed saga submission* (the deduction was never attempted/confirmed) does not block the restart.

**Failure scenario:** the saga-orchestrator has a transient blip (pod restart, brief network partition, Kafka producer backoff) at the exact moment a player clicks Retry after a loss. `sagaSubmitter` returns an error, is logged and ignored, and the player's round restarts with **zero mesos deducted** — a free re-roll of the reward ladder, repeatable every time the player hits a transient failure window, with no compensating control anywhere else in the flow (the entry-cost fee for the *original* game-open Start is enforced via a synchronous, failure-checked saga step in `atlas-npc-conversations`; `Retry`'s fee is not).

**Severity: Important** (economy-integrity defect on the payout-adjacent money path — precisely the class of bug this review was scoped to find).

## Findings — Minor

### MIN-1 (NEW, post-recheck): Loss consolation is never paid on session abandonment/TTL-sweep — only reachable via explicit Retry or Exit

The consolation prize (`submitConsolation`, `services/atlas-rps/atlas.com/rps/game/processor.go:486-503`) fires from exactly two call sites: `Retry` (`processor.go:531-532`) and `Collect`'s `StatusEnded` branch (`processor.go:592-595`, only reached via the client's Exit sub-op). There is no third path.

`services/atlas-rps/atlas.com/rps/game/task.go:41-51` (`SweepTask.Run`) reaps any session past its 5-minute TTL (`registry.go:15` `defaultTTL = 5 * time.Minute`) via `GetRegistry().PopExpired`, and disposes it with a bare `GameEnded{disconnected}` event — it never calls `submitConsolation` (it cannot: `PopExpired` has already removed the session before `Run`'s loop body executes, and the sweep path doesn't have a `Processor`/`sagaSubmitter` in scope at all). `Processor.Dispose` (`processor.go:736-749`), the method that *would* run on an explicit disconnect hook, also does not call `submitConsolation` regardless of the session's status — and `grep -rn "DisposeAndEmit\|rps.NewProcessor" services/atlas-channel` confirms atlas-channel never actually calls `Dispose`/`DisposeAndEmit` on character disconnect in this diff, so `Dispose` is presently unreachable in production entirely.

**Failure scenario:** a player loses a round (session parked at `StatusEnded`, rung 0, per the `Select` loss branch at `processor.go:390-399`), sees the client's "consolation prize" dialog (client string `SP_3681`, per `ida-rps-serverbound.md:112`), and simply closes the dialog / disconnects / alt-tabs away instead of explicitly clicking "Exit" or "Retry" (a very common real-world player behavior). No code path ever calls `submitConsolation` for that session — it just times out 5 minutes later and is swept with `ReasonDisconnected` and no payout. The player who is shown a consolation-prize message by the client never actually receives it unless they click one of the two specific buttons. `task_test.go`'s sweep coverage (`TestSweepTask_Run_DisposesExpiredSessionWithNoPayout`, `task_test.go:140-184`) only exercises a `StatusAwaitingDecision`/rung-2 session — there is no test for a swept `StatusEnded`/rung-0 (post-loss) session, so this gap has no regression coverage either.

**Severity: Minor** (not an economy-integrity risk — mesos are under-granted, not over-granted — but it is a real, untested product-behavior gap directly on the "consolation isn't... omitted when it should fire" question this review was asked to verify, and the client-shown "consolation" message becomes misleading for any player who doesn't take the specific leave action).

### MIN-2 (NEW, post-recheck): Dead `ReasonLost` constant left after the loss-handling redesign

`services/atlas-rps/atlas.com/rps/kafka/message/rps/kafka.go:37` and the mirrored `services/atlas-channel/atlas.com/channel/kafka/message/rps/kafka.go:49` both still declare `ReasonLost = "lost"`. `grep -rn "ReasonLost" services/atlas-channel services/atlas-rps` shows these are the **only** two occurrences in either service — the constant is declared but never referenced. It was superseded by commit `92d85bf1b` ("fix(rps): keep session on loss so the client can show the result"), which removed the immediate `GameEnded{lost}` emission from `Select`'s loss branch (loss now only emits a `RoundResult` with the outcome; `GameEnded` is deferred to the eventual `Collect`/`quit` path, so `ReasonQuit` is what's actually emitted for a post-loss exit — see `processor.go:599`).

Per CLAUDE.md → "Migration & Refactoring Rules": *"Clean Up Dead Code After Extraction... review every modified service file for symbols that are no longer referenced... delete."* and `anti-patterns.md`: *"Leaving dead code after refactoring | Unused constants/structs/functions clutter the codebase and cause confusion."*

**Severity: Minor** (no functional impact — a future reader may reasonably (and incorrectly) assume `ReasonLost` is still a live wire value).

### MIN-3 (confirmed still-open from `audit-backend-recheck.md`): `RpsRewardRestModel` (rps side) still lacks `SetToOneReferenceID`/`SetToManyReferenceIDs` (EXT-01)

`services/atlas-rps/atlas.com/rps/configuration/rest.go:22` `RpsRewardRestModel` is consumed via `requests.SliceProvider[RpsRewardRestModel, game.Ladder]` (`configuration/processor.go:47`) and still has no `SetToOneReferenceID`/`SetToManyReferenceIDs` methods. This was already flagged as `MIN-2` in `audit-backend-recheck.md` and remains unfixed in this diff. Re-confirmed here rather than re-counted as new. Per EXT-01: *"Both methods present, even if no-op. Without them, api2go errors on any response with a `relationships` block."* Latent (no current relationships block on the atlas-tenants response), consistent with several other pre-existing sibling gaps per the prior audit — carried forward, not re-scored as blocking.

## Confirmed PASS (evidence for the round-loop/config risk areas this review targeted)

### PASS-1: Loss no longer double-emits / desyncs `GameEnded` — session correctly kept as `StatusEnded` pending player action

`processor.go:390-399` — the `Select` loss branch now buffers **only** `RoundResult` and keeps the session in the registry via `GetRegistry().Put(p.ctx, updated)` with `StatusEnded`. `Begin` (`processor.go:278`), `Select` (`processor.go:318`) and `Continue` (`processor.go:428`) all correctly reject a `StatusEnded` session with `ErrInvalidStatus`, and `Collect`'s two branches (`processor.go:587`, `:593`) and `Retry` (`processor.go:521`) are the only methods that accept it — confirmed no code path can re-enter a `StatusEnded` session into another round without going through `Retry`'s explicit rung-0 rebuild.

### PASS-2: No double-award under Kafka at-least-once redelivery

`services/atlas-channel/atlas.com/channel/rps/producer.go:16,29,42,56,68` (`producer.CreateKey(int(characterId))`) keys every `COMMAND_TOPIC_RPS` message by `characterId`, guaranteeing per-character ordering within a partition. Independently, the state-machine guards are idempotent to redelivery without relying on ordering: a redelivered `Retry` after a successful first application finds `m.Status() != StatusEnded` (now `StatusAwaitingSelect`) and returns `ErrInvalidStatus` (`processor.go:521-523`) with no second consolation/fee submission; a redelivered `Collect` after a successful first application finds no session at all (`GetRegistry().Get` returns `ok=false`) and returns `ErrSessionNotFound` (`processor.go:582-585`) with no second payout submission.

### PASS-3: `ConsolationMeso` config round-trips symmetrically on both sides

`services/atlas-tenants/atlas.com/tenants/configuration/rest.go:493-496` (`RpsRewardRestModel.ConsolationMeso`), `:529-532` (`TransformRpsReward` reads it), `:586` (`ExtractRpsReward` writes it) all match `services/atlas-rps/atlas.com/rps/configuration/rest.go:22-27` (`RpsRewardRestModel.ConsolationMeso`) and `:66-68` (`Extract` copies it into `game.Ladder.ConsolationMeso`, `game/ladder.go:22-24`). `services/atlas-tenants/atlas.com/tenants/configuration/rest_test.go:13-54` exercises the round trip explicitly, asserting `ConsolationMeso` survives `Transform`.

### PASS-4: Rung-0 consolation guard is enforced, not just commented

`processor.go:487-489` — `submitConsolation`'s first statement is `if m.Rung() != 0 { return }`, unconditionally returning before any ladder lookup or saga submission for `Rung() >= 1`. `services/atlas-rps/atlas.com/rps/game/processor_test.go` (post-`b775bc012`, "consolation only on a no-win loss") covers both the rung-0 and rung>0 cases per the new test additions in this diff (`processor_test.go` grew from ~774 to 1203 lines in this scope, adding dedicated Retry/consolation cases).

### PASS-5: Consolation is not double-submitted within a single Retry/Collect sequence

Traced: `Retry` calls `submitConsolation(m)` once (`processor.go:531-532`) using the pre-transition model snapshot, then immediately transitions the session out of `StatusEnded`. `Collect`'s `StatusEnded` branch likewise calls it exactly once (`processor.go:593-595`) before `GetRegistry().Remove`. No loop, no retry-of-the-consolation-call-itself, and (per PASS-2) redelivery cannot re-trigger it either.

### PASS-6: Mode-byte / sub-op config resolution (DOM-25) for the new START_SELECT arm and RPS_ACTION dispatch

`libs/atlas-packet/rps/operation_body.go:42-46` (`RPSGameStartSelectBody`) resolves the mode byte via `atlas_packet.WithResolvedCode("operations", RPSGameModeStartSelect, ...)` — no literal. `services/atlas-channel/atlas.com/channel/socket/handler/rps_action.go:132-154` (`isRPSAction`) resolves every serverbound sub-op (`START/SELECT/UPDATE/CONTINUE/EXIT/RETRY`) from the tenant `operations` table, never a hardcoded byte. Confirmed the `START_SELECT` row is seeded in **all five** supported version templates: `template_gms_83_1.json:3061`, `template_gms_84_1.json:3099`, `template_gms_87_1.json:2630`, `template_gms_95_1.json:2037`, `template_jms_185_1.json:2632`.

### PASS-7: `game/mock/processor.go` fully synchronized with the three new interface methods

`services/atlas-rps/atlas.com/rps/game/mock/processor.go:20-21,29-30` add `BeginFunc`/`BeginAndEmitFunc` and `RetryFunc`/`RetryAndEmitFunc` matching the interface additions at `processor.go:90-95,109-114`; `var _ game.Processor = (*ProcessorMock)(nil)` (`mock/processor.go:42`) confirms compile-time conformance. Interface-change workflow followed correctly.

### PASS-8: File-responsibilities — `atlas-rps/configuration` (REST-client support package) correctly split

`Processor`/`NewProcessor`/`GetLadder` all in `services/atlas-rps/atlas.com/rps/configuration/processor.go:20-55`; `RpsRewardRestModel`/`Extract`/JSON:API methods all in `configuration/rest.go:1-70`; `getBaseRequest`/`requestRewards` (the only two request-building functions) both in `configuration/requests.go:14-25`. No `<pkg>.go` catch-all file exists in this package (`find services/atlas-rps/atlas.com/rps/configuration -maxdepth 1 -name '*.go'` → `processor.go`, `processor_test.go`, `rest.go`, `requests.go`, `mock/`). FILE-01/02/03/06: PASS.

### PASS-9: Redis discipline (redis-key-guard invariant)

`grep -rn "goredis\.\|redis\.NewClient\|\.HGet(\|\.Get(ctx" services/atlas-rps/atlas.com/rps --include=*.go` (excluding tests) returns zero matches — `services/atlas-rps/atlas.com/rps/game/registry.go` uses only `libs/atlas-redis`'s `atlas.TTLRegistry`/`atlas.Set` wrapper types (`registry.go:9,21-22`), never a raw keyed `go-redis` call.

## Summary

### Blocking (must fix)

- **IMP-1**: Revert or independently verify item ids `4031332`–`4031341` in `services/atlas-tenants/configurations/rps-rewards/default.json:8-17` against WZ/atlas-data before this ships as a live default; update `reward-ladder.md`/`verification.md` to match whatever is actually shipped (both currently claim meso-only, contradicting the seed).
- **IMP-2**: `Retry`'s fee-deduction saga-submit failure (`services/atlas-rps/atlas.com/rps/game/processor.go:536-538`) must not silently allow the round to restart — either propagate the error (mirroring `Collect`'s win-path behavior at `processor.go:616-618`) or explicitly document why an unconfirmed-fee restart is acceptable and add a compensating control (e.g. a reconciliation sweep).

### Non-Blocking (should fix)

- **MIN-1**: Add a `submitConsolation` (or equivalent) call path for TTL-swept/disconnected `StatusEnded` sessions in `services/atlas-rps/atlas.com/rps/game/task.go`, or explicitly document the abandonment-forfeits-consolation behavior as an intentional design decision (it is currently undocumented) and add regression coverage for it.
- **MIN-2**: Delete the dead `ReasonLost` constant from `services/atlas-rps/atlas.com/rps/kafka/message/rps/kafka.go:37` and `services/atlas-channel/atlas.com/channel/kafka/message/rps/kafka.go:49`.
- **MIN-3**: Add no-op `SetToOneReferenceID`/`SetToManyReferenceIDs` to `services/atlas-rps/atlas.com/rps/configuration/rest.go`'s `RpsRewardRestModel` (EXT-01; carried forward from `audit-backend-recheck.md` MIN-2, still unfixed).

---

## Plan adherence review (session-added round-loop work, commits `3745d5c6b..a1322a6a2`)

**Date:** 2026-07-17 · **Branch:** `task-132-rps-npc-game` · **Base:** `c9490b724` · **HEAD:** `a1322a6a2`

Scope: verify the 7 items this session added on top of the already-merged RPS feature — START_SELECT wiring, loss/defer-close, consolation prize, Retry, streak certificates, ShowEffect, and the START_SELECT packet-audit verification pass — end to end (handler→command→processor→event→frame, config schema on both sides, saga steps), plus flag anything silently stubbed and any "documented follow-up" that is actually producible now. This section does not repeat the backend-guidelines findings above; it corroborates, extends, and in one case *corrects* IMP-1 with an independent WZ lookup, and separately confirms plan/task-doc adherence.

### 1. START_SELECT (mode 9) round-start handshake — CONFIRMED, fully wired

- Serverbound `START(0)` decoded and dispatched: `services/atlas-channel/atlas.com/channel/socket/handler/rps_action.go:107-113` (`RPSActionModeStart` branch) → `emitRPSBeginFunc` → `services/atlas-channel/atlas.com/channel/rps/processor.go:33-38` (`Begin`) → `BeginCommandProvider` (`producer.go:15-24`) → `COMMAND_TOPIC_RPS` `CommandTypeBegin`.
- Consumed: `services/atlas-rps/atlas.com/rps/kafka/consumer/rps/consumer.go:99-106` (`handleBeginCommand`) → `Processor.BeginAndEmit` → `Processor.Begin` (`game/processor.go:273-303`): requires `StatusOpen`, transitions to `StatusAwaitingSelect`, buffers `roundStartedEventProvider`.
- `Continue` (post-win, next round) emits the same `RoundStarted` event (`game/processor.go:448-453`) — confirms Continue re-arms the client too, per the task brief.
- Event → frame: `services/atlas-channel/atlas.com/channel/kafka/consumer/rps/consumer.go:112-126` (`handleRoundStartedEvent`) → `rpspkt.RPSGameStartSelectBody()` → `libs/atlas-packet/rps/clientbound/operation.go:71-111` (`StartSelect` struct, mode 9, bodyless, mode byte resolved via `WithResolvedCode`, never hardcoded).
- Tenant `operations` table: `START_SELECT: 9` seeded in all 9 `services/atlas-configurations/seed-data/templates/template_{gms_48,gms_61,gms_72,gms_79,gms_83,gms_84,gms_87,gms_95,jms_185}_1.json` (verified via `grep -c '"START_SELECT"'` = 1 in each of the 9 files; v92 correctly excluded per the plan's documented park).
- **Gap (operational, not code):** `docs/tasks/task-132-rps-npc-game/live-config-patch.md` (last touched at `c077b0e5f`, well before this session) was never updated with a `START_SELECT`/`operations` PATCH snippet for already-provisioned live tenants. Per project memory (`bug_new_opcodes_not_in_live_tenant_config`): seed templates only apply at tenant *creation* — an existing tenant's live socket config needs an explicit PATCH + channel restart before START_SELECT actually reaches its players. This is the same class of gap the memory note documents from a prior task; it is not a task-132 regression, but the runbook doc for *this* branch's new opcode was not extended to cover it.

### 2. Loss shows result then defers close to Exit — CONFIRMED

`game/processor.go:313-401` (`Select`), loss branch (`:366-399`): only `RoundResult` is buffered; the session is written back with `StatusEnded` and **kept** in the registry (`GetRegistry().Put`, not `Remove`). `GameEnded`/END is deliberately not emitted here (documented in the inline comment, `:367-389`, citing a live-confirmed 2026-07-17 finding that emitting `END` here tears the dialog down before the client shows the result). `Begin`/`Select`/`Continue` all reject a `StatusEnded` session (`ErrInvalidStatus`), so the state is unreachable except via the two designed exits: `Collect` (Exit sub-op) and `Retry`. Matches the backend audit's independent **PASS-1** finding.

### 3. Consolation prize (rung-0-only, deferred to leave action) — CONFIRMED, with one real gap

- Config: `consolationMeso` (default `500`) present on both sides — `services/atlas-tenants/configurations/rps-rewards/default.json:5`, `services/atlas-tenants/atlas.com/tenants/configuration/rest.go:493-496,529-532,586`, `services/atlas-rps/atlas.com/rps/configuration/rest.go:22-27,66-68` → `game.Ladder.ConsolationMeso` (`game/ladder.go:22-24`). Round-trip test: `services/atlas-tenants/atlas.com/tenants/configuration/rest_test.go:13-54`.
- Rung-0-only gate: `submitConsolation` (`game/processor.go:486-503`) returns immediately if `m.Rung() != 0` — a player who won at least one round and then lost gets nothing on top of the forfeited winnings, matching the task brief ("post-win retry costs the full fee with no consolation offset").
- Deferred to leave action: called from `Retry` (`:531-532`) and from `Collect`'s `StatusEnded` branch (`:592-595`, the Exit path) — **not** from the point of adjudication. The saga is `AwardMesos`-only, `ShowEffect: true` (via `buildPayoutSaga(m, Rung{Meso: ladder.ConsolationMeso})`, `:498`).
- **Gap (confirms backend audit MIN-1 independently):** there is a third way a `StatusEnded` (post-loss) session leaves the registry — the TTL sweeper. `game/task.go`'s `SweepTask.Run` calls `GetRegistry().PopExpired`, which removes the session and disposes it with a bare `GameEnded{disconnected}` — `submitConsolation` is never called on that path, and cannot be (the sweeper has no `sagaSubmitter`/`ladderProvider` in scope). A player who loses, is shown the "N mesos as a consolation prize" client string, and simply disconnects/alt-tabs/closes the dialog instead of clicking Exit or Retry never receives the consolation the client just promised them. This is a real, untested behavior gap on exactly the feature this session added — not previously flagged in any task doc as an intentional trade-off.

### 4. Retry (serverbound mode 5) — CONFIRMED wired, but the "documented follow-up" undersells a real gap

- Wiring confirmed end-to-end: `rps_action.go:118-124` (`RPSActionModeRetry`) → `emitRPSRetryFunc` → `rps/processor.go:49-51` → `RetryCommandProvider` (`producer.go:52-63`) → `CommandTypeRetry` → `kafka/consumer/rps/consumer.go:126-133` (`handleRetryCommand`) → `Processor.RetryAndEmit` → `Processor.Retry` (`game/processor.go:516-551`): requires `StatusEnded`, re-charges the fee, rebuilds at rung 0 / `StatusAwaitingSelect`, buffers `RoundStarted` (re-arms via `START_SELECT`, no new `OPEN` frame — correct per the IDA note that the board stays open on the loss screen).
- The plan/task prompt explicitly flags "the mode-6 FAIL_NOT_ENOUGH_MESO path" as a documented follow-up. I independently traced what that follow-up actually requires and it is **more than a missing client notice**:
  - `Retry`'s fee-deduction saga is submitted fire-and-forget (`processor.go:535-539`) and its *submission* error is logged-and-swallowed, but critically the round rebuild at `:541` runs **unconditionally**, regardless of whether the saga even reaches the orchestrator, let alone whether `atlas-character` actually has the funds. The entry flow this mirrors (`atlas-npc-conversations`' `processRPSActionState`) has a real failure loop — a `NOT_ENOUGH_MESO` saga failure routes back to `rpsAction_failureState` via the saga-failed consumer (FR-1.3) — but `Retry` has no equivalent feedback path at all; nothing in `atlas-rps` ever learns whether the deduction saga it submitted actually succeeded.
  - Net effect: a character with insufficient mesos can still click Retry and get a free fresh round, repeatably. This is not merely "the client shows no error frame" (a UX gap) — it is an **economy-integrity gap** (unlimited free retries when broke), which is the same conclusion the backend-guidelines pass reached independently as **IMP-2**. I concur with IMP-2's severity assessment.
  - Given CLAUDE.md's "No Deferring Producible Work" bar ("can I produce this myself right now?"), I assessed whether the full mode-6 fix (`FAIL_NOT_ENOUGH_MESO`, IDA-verified bodyless mode 6, `ida-rps-clientbound.md:65,305`) was a trivial deferred prerequisite. It is not: it requires either a synchronous balance pre-check (a new REST call to `atlas-character` before committing to the restart) or a saga-failure feedback loop mirroring the entry flow's `rpsAction_failureState` pattern (a new command/event type, a new channel-side codec wire-up, and holding the round-restart until the saga resolves) — genuinely feature-sized, correctly out of scope for a same-session bug-fix batch. **However**, the narrower, clearly-in-scope half of this — not restarting the round for free when the fee-deduction saga *submission itself* fails (mirroring `Collect`'s existing `return Model{}, err` pattern one method away, `processor.go:616-618`) — was producible in this session and was not done. I flag this as IMP-2's fix recommendation, not a new item.

### 5. Streak certificates 4031332-4031341 — CONFIRMED wired; item ids independently WZ-verified (refines backend audit IMP-1)

- Ladder wiring: `services/atlas-tenants/configurations/rps-rewards/default.json:6-16` (rungs 1-10, `itemId: 4031332`..`4031341`, `quantity: 1`, formula rung N → `4031331+N` matches the commit message). `game.Ladder.PrizeAt` (`ladder.go:29-39`) resolves by rung; `buildPayoutSaga` (`processor.go:657-691`) emits `AwardAsset` for any rung with `itemId != 0 && quantity > 0`.
- The backend-guidelines pass (**IMP-1**, above) flagged these ids as having "zero verification trail anywhere in the repo" and contradicting `reward-ladder.md`'s "meso-only, no items shipped" claim. Both halves of that finding are correct on the *documentation* trail (no `ida-rps-*.md`/`reward-ladder.md` file mentions these ids, and `reward-ladder.md`/`verification.md` are genuinely stale — see item 6 below). **I independently verified the underlying values, however, using the local WZ dump this repo already has checked out** (`tmp/ec876921-c363-4cc6-9c51-5bb8d57f9553/GMS/83.1/`, the documented `reference_atlas_data_wz_inspection` local-dump path):
  - `Item.wz/Etc/0403.img.xml` contains `<imgdir name="04031332">` through `<imgdir name="04031341">` — all ten ids exist as real Etc items in the v83 GMS item table.
  - `String.wz/Etc.img.xml:3850-3888` names them exactly `"Certificate of 1-straight Win"` through `"Certificate of 10-straight Wins"`, with a description confirming they are RPS-specific ("A document certifying N straight win(s) in Rock, Paper, Scissors. Take the ticket to the NPC's Paul, Jean, Martin, or Tony to exchange to another item.").
  - This is a positive match, not a coincidence of range — the semantics (RPS streak certificate) line up exactly with the ladder's use (`rung N` prize on a win streak).
  - **Conclusion:** the shipped item ids are correct and real, not invented. The gap is real but narrower than IMP-1's framing suggests: it is a **documentation/process** failure (the task's own sourcing-verification doc was not updated to record this WZ lookup, so a future reader has no way to know the ids were checked, and `reward-ladder.md`/`verification.md` actively assert the opposite of what's shipped), not a live risk of granting invalid items. I recommend the fix be "update `reward-ladder.md` to record this WZ citation" rather than "revert or treat as unverified."

### 6. Task-doc staleness (both pre-date this session, neither updated by any of the 8 session commits `3745d5c6b..a1322a6a2`)

- `docs/tasks/task-132-rps-npc-game/reward-ladder.md` (last touched `05660af75`) still states the ladder is meso-only and shows `itemId: 0 (none)` for every rung (`reward-ladder.md:12-17`) — contradicted by the shipped `default.json`. See item 5.
- `docs/tasks/task-132-rps-npc-game/verification.md` (last touched `1642c44ec`) lists "Parked follow-ups" #2 (Retry restart-with-fee) and #3 (loss consolation prize) as **not implemented** (`verification.md:124-134`) — both were implemented this session (items 3 and 4 above) and the doc was never refreshed to move them out of the parked list. Anyone reading `verification.md` alone would conclude Retry and consolation are still missing, which is no longer true.
- Neither staleness blocks functionality, but both directly violate the spirit of CLAUDE.md's grounding rule (docs must reflect what's actually shipped) and would mislead a reviewer or the next session picking up this task.

### 7. ShowEffect on entry + retry fee deductions — CONFIRMED

`services/atlas-npc-conversations/atlas.com/npc/conversation/processor.go:1047` (`deduct_entry_cost` `AwardMesosPayload.ShowEffect: true`) and `services/atlas-rps/atlas.com/rps/game/processor.go:652` (`buildFeeDeductionSaga`'s `ShowEffect: true`) both confirmed changed in commit `78b6cafb6`, with matching test updates (`processor_rps_test.go:134-135`). Payout/consolation saga steps were already `ShowEffect: true` from an earlier commit (`processor.go:672,683,498`) — this commit closes the asymmetry the commit message describes ("previously both were silent").

### 8. START_SELECT packet-audit verification across all 9 versions — CONFIRMED

- Codec: `libs/atlas-packet/rps/clientbound/operation.go:71-111` (`StartSelect` struct, `// packet-audit:fname CRPSGameDlg::OnPacket#START_SELECT`), `libs/atlas-packet/rps/operation_body.go:37-46` (`RPSGameStartSelectBody`, config-resolved mode byte).
- Fixture: `libs/atlas-packet/rps/clientbound/operation_test.go` `TestRPSGameStartSelect`, 9 `packet-audit:verify` markers (`gms_v48/v61/v72/v79/v83/v84/v87/v95`, `jms_v185`), run against `rpsVariants` = the 5 standard `pt.Variants` + 4 appended legacy versions (`operation_test.go:29-32`) = 9 total (the surrounding comment says "seven versions" at `operation_test.go:82` — a stale/incorrect count left over from an earlier draft; the marker list and the `rpsVariants` slice both correctly total 9 — cosmetic doc nit only, not a coverage gap).
- Per-version audit reports: `docs/packets/audits/{gms_v48,gms_v61,gms_v72,gms_v79,gms_v83,gms_v84,gms_v87,gms_v95,jms_v185}/RpsStartSelect.{md,json}` all present and ✅ (spot-checked `gms_v83/RpsStartSelect.md`: IDA `0x7402e9`, verdict ✅).
- `tools/packet-audit/cmd/run.go:1965-1966` adds the `CRPSGameDlg::OnPacket#START_SELECT` → `StartSelect` candidate mapping.
- Repo-root gates, re-run fresh for this review: `dispatcher-lint` → `clean` (exit 0); `matrix --check` → clean (exit 0); `fname-doc --check` → `fname-doc check OK (237 structs without an audit report carry no fname)` (exit 0); `operations --check` → `operations check OK (0 absent-writer note(s))` (exit 0). `docs/packets/audits/status.json`/`STATUS.md` diffs for this session are toolSha/exportHash-only (no row change) — expected, since `RPS_GAME` is tracked as a single family row (already ✅ pre-session) and per-mode verification lives in the dispatcher yaml + per-struct fixtures/reports checked above, not as separate matrix rows.

### Build & test verification (this review, run fresh)

| Module | `go build ./...` | `go vet ./...` | `go test -race -count=1 ./...` |
|---|---|---|---|
| `libs/atlas-packet` | PASS | PASS | PASS (all packages `ok`, incl. `rps/clientbound`, `rps/serverbound`) |
| `services/atlas-rps/atlas.com/rps` | PASS | PASS | PASS (all packages `ok`) |
| `services/atlas-channel/atlas.com/channel` | PASS | PASS | PASS (80 `ok`, 0 `FAIL`) |
| `services/atlas-tenants/atlas.com/tenants` | PASS | PASS | PASS (all packages `ok`) |
| `services/atlas-npc-conversations/atlas.com/npc` | PASS | PASS | PASS (all packages `ok`) |
| `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator` | PASS | PASS | PASS (all packages `ok`, unchanged this session but re-verified per instructions) |
| `tools/packet-audit` | PASS | PASS | PASS (all packages `ok`) |

Repo-root gates: `tools/redis-key-guard.sh` and `tools/goroutine-guard.sh` both **PASS (exit 0)** when invoked correctly (per project memory `reference_go_workspace_guard_footguns`: run them *without* a global `GOWORK=off` — the script itself scopes `GOWORK=off` only to its internal build step). **Correction to a footgun that produced false failures during this review and could mislead a future session:** running either guard with `GOWORK=off tools/<guard>.sh` (exporting the env var over the *whole* script, not just the build) makes the compiled analyzer binary's own `go/packages` load fail with `<dir>: matched no packages` for `atlas-mounts` and `atlas-mts` (redis guard) and `atlas-monster-book`/`atlas-doors`/`atlas-mts`/`atlas-mounts` (goroutine guard, per the backend-guidelines audit above) — because those modules resolve some import via the workspace's `go.work` replace graph rather than their own `go.mod` replace lines, and `GOWORK=off` at *run* time (not just build time) breaks that resolution. This is a pre-existing gap in those unrelated modules' own `go.mod`s, not a task-132 regression (task-132 never touches `atlas-mounts`/`atlas-mts`/`atlas-monster-book`/`atlas-doors`), but the correct invocation is the bare `tools/<guard>.sh` (no env override) — that passes clean, 0 findings, on this branch.

### Summary

All 8 session-added items (START_SELECT handshake, loss-defer, consolation, Retry, streak certificates, ShowEffect, and the 9-version packet-audit verification) are genuinely implemented end-to-end with no stubs/TODOs/501s. Two real, non-blocking-but-should-fix gaps were found that extend (not duplicate) the backend-guidelines pass's IMP-2/MIN-1: **Retry's fee-deduction failure is swallowed unconditionally** (economy gap, corroborates IMP-2) and **consolation is unreachable via TTL-sweep/disconnect** (corroborates MIN-1). One backend-guidelines finding (IMP-1) is refined here: the shipped streak-certificate item ids (`4031332`-`4031341`) are independently WZ-verified as correct ("Certificate of N-straight Win(s)", `Item.wz/Etc/0403.img.xml` + `String.wz/Etc.img.xml:3850-3888`) — the defect is that the task's own `reward-ladder.md`/`verification.md` docs were never updated to say so and still assert the opposite (meso-only, Retry/consolation unimplemented), which is a documentation-adherence failure worth fixing before merge even though the shipped values themselves are safe. All module builds/tests/packet-audit gates are green.

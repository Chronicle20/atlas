# Rock-Paper-Scissors NPC Game — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-07-03
---

## 1. Overview

The Rock-Paper-Scissors (RPS) minigame is a classic MapleStory attraction hosted by the
game-park NPC in Henesys (NPC template `9000019`). A player talks to the NPC, pays an
entry bet in meso, and plays rounds of Rock-Paper-Scissors against the server. The client
renders the interaction through its built-in `CRPSGameDlg` dialog; the server is the sole
authority for the opponent's throw, win/loss adjudication, the reward ladder, and all
economy mutations (meso debit, prize credit).

Atlas has **no implementation of this feature today**. A repository-wide search for
`RPS`, `RockPaper`, and `ROCK_PAPER` across `services/` and `libs/` returns zero code
hits. In the packet coverage matrix (`docs/packets/audits/STATUS.md`):

- **`RPS_GAME`** (clientbound, `CRPSGameDlg::OnPacket`) — v83 `0x138`, and distinct
  opcodes for v84/v87/v92/v95 — is ❌ across all versions (no codec, no writer).
- **`RPS_ACTION`** (serverbound, `CRPSGameDlg::OnBtStart`/`SendSelection`/`OnBtContinue`/
  `OnBtRetry`/`OnBtExit`/`Update`) — v83 `0x088`, distinct opcodes for the other four
  versions — is ❌ across all versions (no codec, no handler).

This task delivers the complete feature: the two packet codecs (across all supported
versions), a channel-side serverbound handler + clientbound writer, a **dedicated
microservice** (`atlas-rps`) that owns per-player game session state and the reward-ladder
logic, meso and prize economy integration, and the NPC-conversation entry point that opens
the dialog.

The gameplay model follows the Cosmic reference implementation
(`net/server/channel/handlers/RPSActionHandler.java` + `scripts/npc/9000019.js`): a single
meso bet buys entry to a game, each win lets the player either **collect** the current
prize or **continue** up an escalating reward ladder, and a loss ends the game with the
bet and any accumulated stake forfeited.

## 2. Goals

Primary goals:

- Let a player initiate the RPS game by talking to NPC `9000019` and paying a meso bet.
- Play server-authoritative Rock-Paper-Scissors: the server chooses the opponent throw and
  adjudicates the outcome; the client selection is never trusted to determine the result.
- Implement the Cosmic-style **escalating win-streak reward ladder**: on each win the
  player chooses to collect the current tier's prize or risk it to continue to the next
  tier; a loss forfeits everything.
- Deduct the meso bet at game start and grant prizes on collect.
- Support **all currently provisioned tenant versions** (v83, v84, v87, v92, v95) with the
  correct per-version opcodes and mode bytes.
- Enter the game through the existing NPC-conversation state-machine system, not a
  bespoke hardcoded entry.

Non-goals:

- Player-NPC ("Player NPC") spawning at the game park (out of scope, consistent with the
  rest of Atlas which does not yet implement it).
- Any client-side changes (the client already ships `CRPSGameDlg`).
- Gambling/rate-limiting policy beyond the meso bet itself (e.g. daily play caps).
- An admin UI for editing the reward ladder beyond the tenant configuration that the data
  model requires.
- Persisting game history or leaderboards.

## 3. User Stories

- As a player, I want to talk to the Henesys game-park NPC and start a Rock-Paper-Scissors
  game by paying a meso bet, so that I can gamble for prizes.
- As a player, I want to pick rock, paper, or scissors and immediately see whether I won,
  lost, or tied against the NPC, so that the game feels responsive.
- As a player who just won, I want to choose between collecting my current prize and
  risking it to climb to a bigger prize, so that I control my own risk.
- As a player who loses, I want the game to end clearly and understand that my stake is
  gone, so that the stakes are honest.
- As a player, I want to quit the game at any decision point, so that I am not trapped in
  the dialog.
- As a tenant operator, I want the reward ladder and entry cost to be configuration-driven,
  so that each world/version can tune the game without a code change.

## 4. Functional Requirements

Requirements are grouped by capability area. Exact opcode/mode byte values are deliberately
left to the design phase (IDA-verified per version); this PRD specifies **behavior and
protocol shape**, not byte constants.

### 4.1 Game entry (NPC conversation)

- FR-1.1 Talking to NPC `9000019` presents a conversation (via the existing
  `atlas-npc-conversations` state machine) that offers to start the RPS game.
- FR-1.2 Accepting the offer triggers the server to (a) verify the player can afford the
  entry bet, (b) debit the bet, and (c) send the clientbound `RPS_GAME` "open dialog"
  packet so the client renders `CRPSGameDlg`.
- FR-1.3 If the player cannot afford the bet, the conversation surfaces an appropriate
  message and does **not** open the dialog or debit meso.
- FR-1.4 Only one active RPS game session may exist per player at a time. Re-entering while
  a session is open resumes or resets according to the design (default: an already-open
  session is disposed before a new one starts).

### 4.2 Playing a round

- FR-2.1 After the dialog opens, the player's throw selection arrives as a serverbound
  `RPS_ACTION` packet carrying a "select" mode and the chosen throw (rock/paper/scissors).
- FR-2.2 The server generates the opponent throw using a server-side RNG and adjudicates
  the outcome (win / lose / tie). The client's selection MUST NOT be trusted to decide the
  result — the server independently computes it.
- FR-2.3 The server sends a clientbound `RPS_GAME` result packet conveying the opponent's
  throw and the outcome so the client animates it.
- FR-2.4 On a **tie**, the round is replayed at the current ladder rung with no additional
  meso charge (no bet is lost on a tie).
- FR-2.5 On a **win**, the player advances one rung on the reward ladder and is presented
  with the collect-or-continue choice (FR-3).
- FR-2.6 On a **loss**, the game ends immediately; the entry bet and any accumulated stake
  are forfeited; no prize is granted.

### 4.3 Reward ladder (Cosmic-style escalation)

- FR-3.1 The game maintains a per-session **win streak / ladder position** starting at rung 0.
- FR-3.2 Each rung maps to a prize (item id + quantity, and/or meso) defined by the reward
  ladder configuration (see §6).
- FR-3.3 After each win, the client offers **continue** (risk current winnings for the next
  rung) or **collect** (end the game and receive the current rung's prize). These arrive as
  serverbound `RPS_ACTION` "continue" and "collect/exit" modes.
- FR-3.4 Choosing **continue** starts another round (FR-2) at the next rung.
- FR-3.5 Choosing **collect** ends the game and grants the current rung's prize to the
  player (item and/or meso credit).
- FR-3.6 There is a maximum rung; reaching it forces a collect (the player cannot continue
  past the top of the ladder).
- FR-3.7 A loss at any rung > 0 forfeits the prize that would have been collected — the
  player leaves with nothing (the bet already having been spent at entry).

### 4.4 Quitting / disposal

- FR-4.1 The player may exit the game at any decision point via a serverbound `RPS_ACTION`
  "exit/quit" mode.
- FR-4.2 Quitting after a win but before collecting is treated as **collect** or
  **forfeit** per the design decision recorded in §9 (default, matching Cosmic: an explicit
  exit before collect forfeits; only an explicit collect pays out).
- FR-4.3 Disconnecting or leaving the map disposes the session server-side without paying a
  prize (the bet is already spent).

### 4.5 Multi-version protocol

- FR-5.1 Both `RPS_GAME` (clientbound) and `RPS_ACTION` (serverbound) codecs must be
  implemented for **v83, v84, v87, v92, v95** with each version's correct opcode.
- FR-5.2 The sub-mode bytes within each packet (open/select/result/continue/collect/exit)
  must be resolved per version. Where these are dispatcher-style mode prefixes, they must be
  driven by the tenant configuration `operations` table, never hardcoded literals
  (consistent with the project's dispatcher-family convention).
- FR-5.3 The new opcodes and handler/writer wiring must be added to the seed templates for
  every supported version **and** patched into any already-provisioned live tenant configs
  (new opcodes do not hot-load into existing tenants).

### 4.6 Economy authority

- FR-6.1 Meso debits (entry bet) and credits (meso prizes) are performed by the service that
  owns meso mutation (`atlas-character`), commanded via Kafka — `atlas-rps`/`atlas-channel`
  never mutate meso directly.
- FR-6.2 Item prizes are granted through the inventory service (`atlas-inventory`) via its
  existing award/command path.
- FR-6.3 Economy mutations must be ordered so a player is never charged without a game
  opening, and never granted a prize without a corresponding collect.

## 5. API Surface

`atlas-rps` follows the standard Atlas service shape (REST via JSON:API + Kafka
command/event topics). Exact resource attributes are finalized in design; this is the
intended surface.

### 5.1 REST (JSON:API)

- `GET /rps/games/{characterId}` — read the current RPS session state for a character
  (rung, current prize, status). Resource type e.g. `rps-games`. Tenant-scoped via header.
- Reward-ladder configuration is read from `atlas-tenants` (see §6), not exposed as a
  mutable REST surface in this task.

All request/response bodies use the JSON:API envelope
(`{data:{type, id, attributes}}`). Error cases: `404` when no active session; `409` /
domain error when starting a game the player cannot afford; standard validation errors.

### 5.2 Kafka — commands consumed by `atlas-rps`

- **StartGame** — `{characterId, worldId, channelId, npcId}`: verify affordability, debit
  bet, create session, emit GameOpened.
- **Select** — `{characterId, throw}`: adjudicate a round, advance/lose, emit RoundResult.
- **Continue** — `{characterId}`: begin next round.
- **Collect** — `{characterId}`: grant current rung prize, close session, emit GameEnded.
- **Quit/Dispose** — `{characterId}`: close session without payout.

### 5.3 Kafka — events emitted by `atlas-rps`

- **GameOpened** — channel opens the client dialog.
- **RoundResult** — `{characterId, opponentThrow, outcome, rung, prize}`: channel writes
  the clientbound result packet.
- **GameEnded** — `{characterId, reason: collected|lost|quit|disconnected, grantedPrize?}`.
- Prize/meso side effects are emitted as commands to `atlas-character` (meso) and
  `atlas-inventory` (items).

### 5.4 Channel socket packets

- Serverbound handler for `RPS_ACTION` — decodes the mode + payload, emits the appropriate
  `atlas-rps` command.
- Clientbound writer for `RPS_GAME` — encodes open/result/end frames from `atlas-rps`
  events, per version.

## 6. Data Model

### 6.1 RPS game session (owned by `atlas-rps`)

Per-player ephemeral session (persisted in `atlas-rps`'s store, scoped by `tenant_id`):

| Field           | Type          | Notes                                            |
|-----------------|---------------|--------------------------------------------------|
| `tenant_id`     | uuid          | Multi-tenant scope (surrogate PK + tenant index).|
| `character_id`  | uint32        | One active session per character.                |
| `npc_id`        | uint32        | Entry NPC (9000019).                             |
| `rung`          | int           | Current ladder position (0 = fresh).             |
| `status`        | enum          | `open`, `awaiting_select`, `awaiting_decision`, `ended`. |
| `created_at`    | timestamp     |                                                  |
| `updated_at`    | timestamp     |                                                  |

The session is short-lived; it is created on StartGame and removed on GameEnded. Follow the
project's tenant-safe PK convention (surrogate uuid PK + `(tenant_id, character_id)` unique
index) — do not use `character_id` alone as PK (known collision-on-second-tenant footgun).

### 6.2 Reward ladder configuration (tenant configuration resource)

Stored in `atlas-tenants` as a configuration resource (e.g. resource name `rps-rewards`),
so each tenant/version tunes entry cost and prizes without a code change. Shape:

```json
{
  "data": {
    "id": "rps-rewards",
    "attributes": {
      "entryCostMeso": 1000,
      "ladder": [
        { "rung": 1, "itemId": 0, "quantity": 0, "meso": 0 },
        { "rung": 2, "itemId": 0, "quantity": 0, "meso": 0 }
      ]
    }
  }
}
```

- `entryCostMeso` — the bet debited at StartGame (default 1000).
- `ladder` — ordered rungs; each maps to a prize (item id + quantity and/or meso). The
  concrete prize table is filled from the Cosmic `9000019.js` reward set during design (and
  verified against WZ/repo data — no invented item ids in the PRD).
- The top of the `ladder` array is the maximum rung (FR-3.6).

**Note:** actual item ids/quantities and the exact escalation curve are NOT specified in
this PRD; they will be sourced from the Cosmic reference and verified against local WZ /
item data during design, per the project's "verify, don't invent" rule.

## 7. Service Impact

### 7.1 New service: `atlas-rps`

A new Go microservice owning RPS session state and reward-ladder logic:

- Standard Atlas structure: immutable models + Builder, Processor (Interface + Impl,
  `NewProcessor(l, ctx)`, pure `Method(mb)` + `MethodAndEmit()`), Kafka
  `message.Buffer`/`message.Emit`, curried consumer registration, JSON:API REST handlers,
  tenant context from headers/consumers, config loading from `atlas-tenants`.
- **Registration checklist (new Go service):** add to `.github/config/services.json`
  **and** the hardcoded `go_services` list in `docker-bake.hcl` (HCL cannot read JSON —
  known hand-sync requirement); add the two `COPY` lines for any new shared lib to the
  repo-root `Dockerfile` and a `./libs/...` line to `go.work` only if a new lib is
  introduced; add k8s deploy manifests. `atlas-rps` is a REST+Kafka service (no raw
  login/channel socket), so **no LB socket port** wiring is required.

### 7.2 `libs/atlas-packet`

- New clientbound codec for `RPS_GAME` (open/result/end frames) for all five versions.
- New serverbound codec for `RPS_ACTION` (mode-prefixed) for all five versions.
- Reuse shared types from `libs/atlas-constants` (character/world/channel ids, item ids)
  rather than reinventing.

### 7.3 `atlas-channel`

- New serverbound handler for `RPS_ACTION` → emits `atlas-rps` commands.
- New clientbound writer for `RPS_GAME` ← consumes `atlas-rps` events.
- Register handler with a proper validator (LoggedInValidator) — a validator-less handler
  entry is silently dropped.
- Seed-template opcode/operations wiring for all versions **plus** a live-config patch for
  already-provisioned tenants.

### 7.4 `atlas-npc-conversations`

- NPC `9000019` conversation state machine that offers the game and, on accept, triggers the
  StartGame flow (affordability check → bet debit → open dialog).

### 7.5 `atlas-character`

- Consumes meso debit (entry bet) and meso credit (meso prizes) commands. Reuse existing
  meso mutation command path; no new economy authority elsewhere.

### 7.6 `atlas-inventory`

- Grants item prizes via its existing award/command path.

### 7.7 `atlas-tenants`

- New configuration resource `rps-rewards` (REST model + Transform/Extract + providers +
  processor methods + handlers + routes + mock) and seed data per version.

## 8. Non-Functional Requirements

- **Server authority / anti-cheat:** the opponent throw and outcome are computed
  server-side; the client-supplied selection never determines win/loss (FR-2.2). RNG lives
  server-side in `atlas-rps`.
- **Economy integrity:** no debit without an opened game; no prize without a collect;
  meso/item mutations flow only through their owning services (`atlas-character`,
  `atlas-inventory`). Ordering must prevent a charge-without-open or grant-without-collect.
- **Multi-tenancy:** all state and config are tenant-scoped (`tenant.MustFromContext`,
  header parsing in consumers); tenant-safe surrogate PKs.
- **Version correctness:** correct per-version opcodes/mode bytes; dispatcher-style modes
  resolved via the tenant `operations` table (never literal); new opcodes patched into live
  tenant configs, not just seeds.
- **Observability:** structured logging around StartGame/Select/Collect/GameEnded with
  characterId + tenant; log dropped/unhandled packets at appropriate levels.
- **Verification:** `RPS_GAME` and `RPS_ACTION` codecs validated with byte-fixture tests
  per version (matrix cells promoted from ❌), following
  `docs/packets/audits/VERIFYING_A_PACKET.md`. Standard gate: `go test -race ./...`,
  `go vet ./...`, `go build ./...`, `docker buildx bake atlas-rps` (and any other touched
  service), `tools/redis-key-guard.sh` — all clean before "done".

## 9. Open Questions

- **Quit-before-collect semantics (FR-4.2):** default to Cosmic behavior — an explicit exit
  before collect forfeits, only an explicit collect pays out. Confirm against the v83 client
  `CRPSGameDlg::OnBtExit` vs `OnBtContinue` semantics during design (IDA-verified).
- **Per-version mode bytes:** the exact `RPS_GAME`/`RPS_ACTION` sub-mode byte values and
  frame layouts per version must be IDA-verified in design; whether they use the shared
  `operations` dispatcher table or fixed inline bytes is determined then.
- **Reward ladder contents:** concrete item ids, quantities, and escalation curve are to be
  sourced from Cosmic `9000019.js` and verified against local WZ/item data (no invented
  values).
- **Session store:** whether `atlas-rps` persists sessions in Postgres (like other services)
  or keeps them purely in-memory (they are ephemeral) — decide in design, weighing crash
  recovery vs simplicity.
- **Tie handling on the wire:** confirm whether the v83 client expects a distinct "tie/redraw"
  result frame or reuses the result frame with a tie outcome code.

## 10. Acceptance Criteria

- [ ] Talking to NPC `9000019` offers the RPS game via the NPC-conversation system.
- [ ] Accepting debits the configured entry bet (default 1000 meso) and opens the client
      `CRPSGameDlg` dialog; an unaffordable player is refused without a debit.
- [ ] Selecting rock/paper/scissors produces a server-authoritative outcome; the client
      selection cannot force a win.
- [ ] A win advances the ladder and presents collect-or-continue; a tie replays the rung
      with no extra charge; a loss ends the game with nothing granted.
- [ ] Choosing collect grants the current rung's prize (item and/or meso) via the owning
      services; choosing continue starts the next rung; reaching the top rung forces a collect.
- [ ] Quitting or disconnecting disposes the session without a payout.
- [ ] `RPS_GAME` (clientbound) and `RPS_ACTION` (serverbound) codecs exist and are
      byte-fixture verified for **v83, v84, v87, v92, v95**; matrix cells promoted from ❌.
- [ ] New opcodes/handlers/writers are wired into every version's seed template and patched
      into already-provisioned live tenant configs.
- [ ] `atlas-rps` is registered in `services.json` and `docker-bake.hcl`, with k8s manifests.
- [ ] Reward ladder + entry cost are tenant-configurable via the `rps-rewards`
      configuration resource.
- [ ] Full verification gate passes for every changed module: `go test -race ./...`,
      `go vet ./...`, `go build ./...`, `docker buildx bake` for each touched service,
      `tools/redis-key-guard.sh` — all clean.

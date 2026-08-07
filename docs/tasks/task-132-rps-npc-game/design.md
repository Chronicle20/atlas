# Rock-Paper-Scissors NPC Game — Design

Task: task-132-rps-npc-game
Status: Draft (design phase)
Created: 2026-07-04
PRD: `docs/tasks/task-132-rps-npc-game/prd.md`

---

## 1. Summary

Deliver the Rock-Paper-Scissors (RPS) minigame hosted by the Henesys game-park NPC
(`9000019`). A player pays a meso bet, plays server-authoritative RPS rounds against the
server, and climbs an escalating collect-or-continue reward ladder. The feature spans a new
microservice (`atlas-rps`), two dispatcher-family packet codecs (`RPS_GAME` clientbound,
`RPS_ACTION` serverbound) across all IDA-tracked versions, channel handler/writer wiring, an
NPC-conversation entry point that reuses the existing saga machinery, a tenant-config reward
ladder resource, and the economy integrations for the bet and the prizes.

This design commits to concrete architecture and boundaries. Exact per-version packet mode
bytes, the concrete reward-ladder contents, and a couple of client-semantic confirmations
are IDA/WZ-verified during the plan/execute phases (§14, §16) — they are *not* invented here,
per the project's "verify, don't invent" rule.

---

## 2. Decisions & rationale

Three decisions were defaulted to the recommended option during design (the design interview
was unattended); each is reversible at plan time if desired.

### D1 — Version set: v83, v84, v87, v95, jms_v185; **park v92**

The PRD names "v83, v84, v87, v92, v95." The packet coverage matrix and the IDA exports
(`docs/packets/ida-exports/`) actually track **v83, v84, v87, v95, jms_v185** — there is **no
v92 column and no v92 IDA export**. The opcodes the PRD attributed to "v92/v95" are the
v95/jms185 columns:

| Packet     | Direction   | v83     | v84     | v87     | v95     | jms_v185 |
|------------|-------------|---------|---------|---------|---------|----------|
| RPS_GAME   | clientbound | `0x138` | `0x13F` | `0x149` | `0x173` | `0x151`  |
| RPS_ACTION | serverbound | `0x088` | `0x08C` | `0x090` | `0x0A0` | `0x08B`  |

(Source: `docs/packets/audits/STATUS.md`; both rows are `❌` in every version.)

RPS is a dispatcher family and requires (a) an IDA-verified per-version mode-byte table and
(b) an `operations` table in the tenant socket template. The `gms_92_1` socket template is a
15 KB stub with **no `operations` table** (v95/jms185 are complete at 36–41 KB *with*
`operations`), and no v92 IDB exists to verify mode bytes against. This is the identical
blocker that parked task-086's v92 mount-food handler ("no v92 IDB to verify"). Implementing
v92 would require inventing bytes — forbidden.

**Decision:** target the five IDA-tracked versions. Document a parked v92 follow-up (unblocks
when a v92 IDB exists), mirroring the mount-food precedent. jms_v185 — which the PRD omitted —
**is** included because it is tracked, provisioned, and has a complete template.

### D2 — Session store: **Redis TTL registry**, not Postgres

The PRD §6.1 data-model table is Postgres-shaped. The idiomatic Atlas store for ephemeral
per-character session state is a Redis TTL registry (the `atlas-expressions` pattern). A
crashed/expired session simply forfeits an in-progress game: the bet is already spent at
entry and prizes are granted only on an explicit collect, so there is **no economic-integrity
reason to persist**. Postgres would add a DB, a migration, and the tenant-safe-PK footgun for
state that is intentionally throw-away.

**Decision:** store sessions in a per-tenant Redis TTL registry with a background sweeper that
disposes abandoned sessions (no payout on expiry). The PRD's `created_at`/`updated_at`/PK
columns become registry-entry metadata, not SQL columns.

### D3 — Economy flow: **saga-orchestrated entry & payout**, direct round loop

`atlas-npc-conversations` has exactly two producer surfaces — client dialog commands and saga
steps to `atlas-saga-orchestrator`; it does **not** emit arbitrary service commands. All
NPC-initiated economy already flows through sagas (`award_mesos`, `open_storage`→`ShowStorage`,
gachapon→`DestroyAsset`+`SelectGachaponReward`). RPS mirrors the **gachapon integration
precedent** exactly:

- **Entry** is a saga `[AwardMesos(−entryCost) → StartRPSGame]`. `AwardMesos` is the existing
  atomic check-and-debit (it fails with `NOT_ENOUGH_MESO`, routing the conversation to a
  "can't afford" state — FR-1.3 for free). `StartRPSGame` is a new saga action dispatched to
  `atlas-rps`, exactly as `SelectGachaponReward` is dispatched to `atlas-gachapons`.
- **Payout** on collect is a second saga that `atlas-rps` initiates
  (`AwardMesos(+prizeMeso)` and/or `AwardAsset(prizeItem)`), then closes the session.
- **The round loop** (select → result, continue, tie-replay) is a high-frequency, no-economy
  interaction and runs **directly** channel ↔ atlas-rps ↔ channel — it does **not** go
  through the saga-orchestrator.

This keeps every meso/item mutation inside its owning service via the shared saga vocabulary,
gives atomic compensation on the two economy touchpoints for free, and adds no new producer
surface to `atlas-npc-conversations`.

---

## 3. Architecture overview

```
                       ┌─────────────────────────┐
   talk to NPC 9000019 │  atlas-npc-conversations │
   ───────────────────▶│  npc-9000019 state m/c   │
                       └───────────┬─────────────┘
                                   │ entry saga
                                   │ [AwardMesos(−cost) → StartRPSGame]
                                   ▼
        ┌──────────────────────────────────────────────┐
        │            atlas-saga-orchestrator            │
        │  AwardMesos ─▶ atlas-character (REQUEST_CHANGE_MESO) │
        │  StartRPSGame ─▶ atlas-rps (COMMAND_TOPIC_RPS)      │
        └───────────────┬──────────────────────────────┘
                        │ StartGame cmd
                        ▼
   ┌──────────────────────────────────┐        events (GameOpened / RoundResult / GameEnded)
   │             atlas-rps            │───────────────────────────────────────────┐
   │  • Redis TTL session registry    │                                           │
   │  • server RNG + adjudication     │◀───────────── commands ───────────┐       │
   │  • reward-ladder logic           │   (Select / Continue / Collect /   │       │
   │  • reads rps-rewards config      │    Quit) via COMMAND_TOPIC_RPS     │       │
   │  • payout saga on collect ───────┼──▶ atlas-saga-orchestrator ──▶ character / inventory
   └──────────────────────────────────┘                                    │       │
                                                                           │       ▼
   ┌──────────────────────────────────────────────────────────────────────┴───────────────┐
   │                                   atlas-channel                                         │
   │  serverbound RPS_ACTION handler ─▶ emits Select/Continue/Collect/Quit commands to rps   │
   │  clientbound RPS_GAME writer     ◀─ consumes rps events, writes open/result/end frames  │
   │  (LoggedInValidator; per-tenant opcode + operations mode-byte tables)                    │
   └───────────────────────────────────────┬─────────────────────────────────────────────────┘
                                            │ socket
                                            ▼
                                    client CRPSGameDlg
```

Component responsibilities:

- **atlas-rps** — sole authority for session state, opponent throw (server RNG), win/lose/tie
  adjudication, ladder position, and the collect/continue/quit lifecycle. Reads the reward
  ladder from tenant config. Initiates the payout saga on collect. Owns no packet encoding and
  no direct meso/item mutation.
- **atlas-channel** — the wire boundary. Decodes serverbound `RPS_ACTION` into atlas-rps
  commands; encodes atlas-rps events into clientbound `RPS_GAME` frames. Holds the per-tenant
  opcode/operations tables.
- **atlas-npc-conversations** — the entry point. The `9000019` state machine offers the game
  and builds the entry saga.
- **atlas-saga-orchestrator** + **libs/atlas-saga** — carry the new `StartRPSGame` action and
  dispatch the entry/payout economy steps.
- **atlas-tenants** — owns the `rps-rewards` configuration resource (entry cost + ladder).
- **atlas-character / atlas-inventory** — unchanged economy authorities, driven via existing
  saga actions.

---

## 4. End-to-end flows

### 4.1 Entry (talk → offer → bet → open dialog)

1. Player talks to NPC `9000019` → `START_CONVERSATION` → the `9000019` state machine runs.
2. A `dialogue` (yes/no) state offers the game (entry cost read from config, surfaced in text).
3. On **yes**, a state builds the **entry saga** `[AwardMesos(−entryCost), StartRPSGame]`.
   - `AwardMesos(−entryCost)` → `atlas-character` `REQUEST_CHANGE_MESO` (signed amount). If the
     player cannot afford it, `atlas-character` replies `NOT_ENOUGH_MESO`; the saga fails and
     the conversation routes to a "not enough meso" dialogue state. No dialog opens (FR-1.3).
   - `StartRPSGame` → `atlas-rps` creates the session (rung 0, status `open`), emits
     `GameOpened`, and reports saga success.
4. `atlas-channel` consumes `GameOpened` → writes the clientbound `RPS_GAME` **open** frame →
   the client renders `CRPSGameDlg`.
5. **Idempotency / re-entry (FR-1.4):** if a session already exists for the character,
   `StartRPSGame` disposes the stale session first (no payout) and starts fresh.

### 4.2 Play a round (select → adjudicate → result)

1. Player picks rock/paper/scissors → client sends serverbound `RPS_ACTION` (**select** mode +
   throw). Channel's handler decodes it and emits a **Select** command to atlas-rps.
2. atlas-rps generates the opponent throw with a server-side RNG and adjudicates independently
   (the client throw is an input, never the result — FR-2.2). Standard rules: rock>scissors,
   scissors>paper, paper>rock.
3. atlas-rps updates the session and emits **RoundResult** `{opponentThrow, outcome, rung, prize}`.
4. Channel consumes it and writes the clientbound `RPS_GAME` **result** frame; the client
   animates the throw and, on a win, shows collect/continue.

Outcomes:

- **Tie (FR-2.4):** rung unchanged, no charge, status returns to `awaiting_select`; the result
  frame carries a tie outcome so the client re-enables selection. (Whether the client expects a
  distinct tie/redraw frame or the result frame with a tie code is an IDA confirmation — §16.)
- **Win (FR-2.5):** rung += 1, status `awaiting_decision`, prize = ladder[rung]. The result
  frame conveys the win and the current prize; the client offers collect/continue.
- **Loss (FR-2.6):** session ends immediately, no payout; atlas-rps emits `GameEnded{reason:lost}`;
  channel writes the end frame.

### 4.3 Win decision (continue / collect)

- **Continue (FR-3.4):** serverbound `RPS_ACTION` continue mode → **Continue** command → rung is
  already advanced; status → `awaiting_select`; another round begins (§4.2). At the max rung
  (FR-3.6), continue is refused server-side and treated as a forced collect.
- **Collect (FR-3.5):** serverbound `RPS_ACTION` collect mode → **Collect** command → atlas-rps
  reads ladder[rung]'s prize, initiates the **payout saga**
  (`AwardMesos(+meso)` and/or `AwardAsset(item, qty)`), closes the session, emits
  `GameEnded{reason:collected, grantedPrize}`. Channel writes the end frame.

### 4.4 Quit / disposal

- **Explicit exit (FR-4.1/4.2):** serverbound `RPS_ACTION` exit mode → **Quit** command →
  session closed, **no payout** (Cosmic default: only an explicit collect pays). `GameEnded{reason:quit}`.
- **Disconnect / leave map (FR-4.3):** channel emits a dispose command (or the TTL sweeper
  reaps the session) → closed with no payout. The already-spent bet is not refunded.

---

## 5. atlas-rps service design

Structure (modeled on `atlas-expressions` spine + `atlas-chalkboards` REST layer). Module
`atlas-rps` at `services/atlas-rps/atlas.com/rps/`:

```
main.go                     # registry init, consumers, REST server, TTL sweeper task
logger/init.go
tasks/task.go               # generic task scheduler
game/
  model.go  builder.go      # immutable Session model + validating Builder
  registry.go               # Redis TTL registry keyed (tenant, characterId)
  processor.go              # Interface + Impl; Start/Select/Continue/Collect/Quit + …AndEmit
  adjudicate.go             # pure RPS rules + server RNG opponent throw (unit-tested)
  ladder.go                 # reward-ladder resolution from config
  producer.go               # GameOpened / RoundResult / GameEnded event providers
  task.go                   # sweeper: dispose expired sessions (no payout)
  mock/processor.go
  *_test.go
configuration/              # load rps-rewards from atlas-tenants (entry cost + ladder)
rest/handler.go             # ParseCharacterId etc.
game/resource.go            # InitResource: GET /rps/games/{characterId}
game/rest.go                # RestModel + GetName()="rps-games"
kafka/
  consumer/consumer.go
  consumer/rps/consumer.go  # StartGame/Select/Continue/Collect/Quit command handlers
  message/message.go        # Buffer + Emit + EmitWithResult
  message/rps/kafka.go      # COMMAND_TOPIC_RPS + EVENT_TOPIC_RPS + Command/Event structs
  message/character/kafka.go   # local copy of meso command types (for payout saga wiring, if direct)
  message/saga/kafka.go     # saga command envelope (payout saga + StartRPSGame status)
  producer/producer.go
```

### 5.1 Session model

Immutable model (private fields + getters + Builder). Fields:

| Field         | Type            | Notes                                              |
|---------------|-----------------|----------------------------------------------------|
| tenantId      | uuid            | from context; registry scope                       |
| characterId   | uint32          | registry key; one active session per character     |
| worldId       | world.Id (byte) | for downstream commands/events                     |
| channelId     | channel.Id      | for the channel writer routing                     |
| npcId         | uint32          | entry NPC (9000019)                                |
| rung          | int             | 0 = fresh; incremented on win                       |
| status        | enum            | `open` → `awaiting_select` → `awaiting_decision` → `ended` |
| lastThrow     | throw enum      | last opponent throw (for idempotent re-sends)      |
| createdAt     | time.Time       | registry metadata                                  |
| updatedAt     | time.Time       | registry metadata                                  |

Use `libs/atlas-constants` types (`world.Id`, `channel.Id`, character id, item id) — no new
numeric aliases (DOM-21).

### 5.2 Processor

Interface + Impl, `NewProcessor(l, ctx)` with `tenant.MustFromContext(ctx)`. Each op has a
buffered `Method(mb …)` and an emitting `…AndEmit()` wrapper (`message.EmitWithResult`):

- `Start(mb, characterId, worldId, channelId, npcId)` — dispose any stale session, create rung-0
  session, emit `GameOpened`.
- `Select(mb, characterId, throw)` — RNG opponent throw, adjudicate; on win advance rung and set
  `awaiting_decision`; on tie keep rung and `awaiting_select`; on loss end + emit
  `GameEnded{lost}`; emit `RoundResult`.
- `Continue(mb, characterId)` — validate `awaiting_decision` & rung < max; set `awaiting_select`.
  At max rung, force collect.
- `Collect(mb, characterId)` — resolve ladder[rung] prize; initiate payout saga; end session;
  emit `GameEnded{collected, prize}`.
- `Quit(mb, characterId)` / `Dispose(mb, characterId)` — end session, no payout.

`adjudicate.go` is pure and RNG-injectable so unit tests are deterministic (a seedable/mockable
throw source). Server authority (FR-2.2, NFR anti-cheat) lives here.

### 5.3 Registry (Redis TTL)

`atlas.NewTTLRegistry[uint32, Model]` keyed on characterId, tenant tracked in an `atlas.Set`,
per the expressions pattern. TTL bounds abandoned sessions (e.g. a few minutes of inactivity);
the sweeper task pops expired entries and disposes them with no payout. All access via
`atlas-redis` lib types (rediskeyguard invariant — no raw go-redis).

### 5.4 REST

`GET /rps/games/{characterId}` → JSON:API `rps-games` resource `{rung, status, prize}`; `404`
when no active session (FR/§5.1 of PRD). Read-only; the reward ladder is not a mutable REST
surface here.

### 5.5 Kafka topics

- `COMMAND_TOPIC_RPS` (inbound): `StartGame`, `Select`, `Continue`, `Collect`, `Quit/Dispose`.
- `EVENT_TOPIC_RPS` (outbound): `GameOpened`, `RoundResult`, `GameEnded`.
- Payout: emits saga command (or direct character/inventory commands — see D3; saga preferred).
- Consumes header parsers (span + tenant), curried `InitConsumers(l)(cmf)(groupId)`.

---

## 6. Packet design (`libs/atlas-packet`)

Both packets are **dispatcher families** — follow `docs/packets/DISPATCHER_FAMILY.md` to the
letter (INV-1..5; no AP-1..7 anti-patterns). One discrete struct per mode, `Encode` writes the
mode byte then the full arm body (every field cited to a decompile line), a per-mode body
function that fixes its op key and resolves the version mode via
`WithResolvedCode("operations", KEY, …)`, a `candidatesFromFName` case per mode, and per-mode
byte-fixture verification.

### 6.1 RPS_GAME — clientbound (`CRPSGameDlg::OnPacket`)

New package `libs/atlas-packet/rps/`, clientbound structs in `rps/clientbound/`, body functions
in `rps/rps_operation_body.go` (root). Frame vocabulary (server → client), mode bytes
IDA-verified per version in the plan:

- **OPEN** — open the dialog / start a game.
- **RESULT** — opponent throw + outcome (win/lose/tie) + current rung/prize for the client to
  render collect/continue.
- **END** — game over (collected / lost / quit), optional granted-prize echo.
- (Any additional frames — e.g. a distinct tie/redraw or a timeout — are enumerated from the
  `CRPSGameDlg::OnPacket` switch during IDA verification; §16.)

### 6.2 RPS_ACTION — serverbound (`CRPSGameDlg::OnBt*` / `SendSelection` / `Update`)

Serverbound arms in `rps/serverbound/`. The top-level `Operation` decodes only the leading mode
byte; per-arm structs decode the remaining body. Sub-actions, from the registry fnames:

- `OnBtStart` → **start** (begin a game at the current bet).
- `SendSelection` → **select** (chosen throw: rock/paper/scissors).
- `OnBtContinue` → **continue** (climb the ladder).
- `OnBtRetry` → **retry** (replay; semantics vs tie-redraw IDA-confirmed — §16).
- `OnBtExit` → **exit/quit**.
- `Update` → **update** (client tick/refresh; mapping confirmed in IDA — may be a no-op server-side).

The channel handler maps each decoded mode to the corresponding atlas-rps command. Modes are
resolved through the tenant `operations` table (never literal), per the dispatcher convention
and memory `feedback_dispatcher_config_drive_all_modes`.

### 6.3 Per-version opcodes

The opcode table in §2/D1 is the authoritative per-version opcode set (v83/v84/v87/v95/jms185).
Mode-byte tables per version are populated from each version's `CRPSGameDlg` switch during
verification (they are version-dependent, like other dispatcher families — memory
`bug_operations_mode_tables_missing_v87_v95_jms`).

---

## 7. atlas-channel wiring

- **Serverbound handler** `socket/handler/rps_action.go`: `RPSActionHandle` const +
  `RPSActionHandleFunc`; decodes `rps.Operation`, reads mode, compares against the tenant
  `operations` table (like `storage_operation.go`'s `isStorageOperation`), decodes the arm, and
  emits the matching atlas-rps command. Registered in `produceHandlers()` **with a validator**
  (`LoggedInValidator`) — a validator-less entry is silently dropped (memory
  `bug_socket_handler_missing_validator_silently_dropped`).
- **Clientbound writer** `socket/writer/rps_game.go`: body helpers per frame; writer consts
  (`RPSGameWriter`, `RPSGameOperation*` body funcs) registered in the `main.go` writer slice.
- **Event consumer** `kafka/consumer/rps/consumer.go`: subscribes to `EVENT_TOPIC_RPS`; on
  `GameOpened`/`RoundResult`/`GameEnded`, verifies tenant, resolves the session via
  `session.NewProcessor(...).GetByCharacterId(...)`, builds the frame body, and
  `session.Announce(...)(RPSGameWriter)(bp)(s)`.
- **Tenant config** — new opcode rows (handler + writer), the `operations` mode table, and the
  `HandlerConfig` (opcode→validator→handler) binding, for all five versions' seed templates
  **and** a live-config patch (§13).

---

## 8. atlas-npc-conversations — NPC 9000019

- **Definition:** add `npc-9000019.json` conversation to the seed set for each supported
  version (`deploy/seed/gms/{83,84,87,95}_1/npc-conversations/npc/` and
  `deploy/seed/jms/185_1/...`). State machine: a `dialogue` yes/no offer → on yes an action
  state that builds the entry saga → success/failure dialogue states.
- **Entry saga integration:** mirror the gachapon pattern. Options for the yes-branch:
  (a) a new state type `rpsAction` with a `processRPSActionState` that builds the saga and waits
  on `pendingSagaId`, or (b) a `genericAction` with a new `start_rps_game` operation `case` in
  `operation_executor.go` that appends `AwardMesos(−entryCost)` + `StartRPSGame` steps. **(b) is
  preferred** — it reuses the existing operation/saga machinery and the existing `award_mesos`
  case, adding only the `start_rps_game` case and re-exporting the `StartRPSGame` action const in
  `atlas.com/npc/saga/model.go`.
- **Config source of entry cost:** the conversation reads `entryCostMeso` from the `rps-rewards`
  config (via context injection or a `local:` helper) so the offer text and the debit stay in
  sync with tenant config.
- The affordability failure (`NOT_ENOUGH_MESO`) routes to a "not enough meso" dialogue
  (FR-1.3); the conversation ends cleanly.

---

## 9. atlas-saga-orchestrator + libs/atlas-saga

- **libs/atlas-saga:** add `StartRPSGame Action = "start_rps_game"` (§ near the other minigame
  actions), a `StartRPSGamePayload{ CharacterId, WorldId, ChannelId, NpcId }` in `payloads.go`,
  and unmarshal wiring in `unmarshal.go` (+ test).
- **atlas-saga-orchestrator:** add a dispatch handler for `StartRPSGame` that emits the
  `StartGame` command to `atlas-rps` (`COMMAND_TOPIC_RPS`) and awaits atlas-rps' saga-status
  reply (success once the session is created / `GameOpened` emitted), exactly as gachapon's
  `SelectGachaponReward` dispatches to `atlas-gachapons`.
- **Payout:** on collect, `atlas-rps` builds a saga `[AwardMesos(+meso)? , AwardAsset(item,qty)?]`
  (only the non-zero prize components) and submits it to the orchestrator. Reuses existing
  actions — no new action needed for the payout. (Alternatively, direct
  `REQUEST_CHANGE_MESO`/`CREATE_ASSET` commands from atlas-rps — the D3 fallback — if the saga
  round-trip proves unnecessary; default is the saga for compensation symmetry.)

---

## 10. atlas-tenants — `rps-rewards` configuration resource

Add a new configuration resource `rps-rewards` mirroring the "vessels" resource (the simplest
full resource), all under `services/atlas-tenants/atlas.com/tenants/`:

- `configuration/rest.go` — `RpsRewardRestModel` (`GetName()="rps-rewards"`), `TransformRpsReward`,
  `ExtractRpsReward` (`type:"rps-rewards"`), Create/Single JSON helpers, the 6 handlers, and
  `RegisterRoutes` wiring under `/tenants/{tenantId}/configurations/rps-rewards`.
- `configuration/processor.go` — interface + impl methods (`GetRpsRewards`, providers, and
  Create/Update/Delete/Seed `…AndEmit` variants).
- `configuration/provider.go` — `GetRpsRewardsProvider` reusing
  `GetByTenantIdAndResourceNameProvider(tenantId, "rps-rewards")`.
- `configuration/kafka.go` — `EventTypeRpsRewardsUpdated` etc. on the shared config-status topic.
- `configuration/seed.go` — `rps-rewards` seed loader (+ env override path).
- `configuration/mock/processor.go` — matching mock methods (compile-time interface check).
- `rest/handler.go` — `ParseRpsRewardId` helper.
- Seed data `services/atlas-tenants/configurations/rps-rewards/*.json` per version; optional
  Bruno collateral.

Config shape (per PRD §6.2):

```json
{ "data": { "id": "rps-rewards", "attributes": {
  "entryCostMeso": 1000,
  "ladder": [ { "rung": 1, "itemId": 0, "quantity": 0, "meso": 0 }, … ] } } }
```

`entryCostMeso` default 1000; `ladder` is ordered, top element = max rung (FR-3.6). Concrete
item ids/quantities/meso curve are sourced from the Cosmic `9000019.js` reward set and
**verified against local WZ/item data** during execution (§16) — not invented here. `atlas-rps`
reads this resource via the standard `atlas-tenants` config path.

---

## 11. Economy integrity (NFR)

- **No debit without an opened game:** entry saga step order is `AwardMesos(−cost)` *then*
  `StartRPSGame`; if `StartRPSGame` fails, the saga compensates the debit. If `AwardMesos` fails
  (unaffordable), no game is started.
- **No prize without a collect:** prizes are granted only by the payout saga, which is
  initiated solely by the `Collect` command; loss/quit/disconnect never pay.
- **Single mutation authority:** meso only via `atlas-character` `REQUEST_CHANGE_MESO`, items only
  via `atlas-inventory` `CREATE_ASSET`, both through saga actions. atlas-rps and atlas-channel
  never mutate economy directly.

---

## 12. Multi-version protocol & live-config patching

- Both codecs implemented for v83/v84/v87/v95/jms185 with the opcodes in §2/D1; mode bytes and
  `operations` tables populated per version from IDA.
- Seed templates (`services/atlas-configurations/seed-data/templates/template_gms_{83,84,87,95}_1.json`
  and `template_jms_185_1.json`) get the new handler/writer opcode rows, the RPS `operations`
  mode entries, and the `HandlerConfig` binding.
- **Live tenants are patched** (new opcodes/handlers/writers do not hot-load — memory
  `bug_new_opcodes_not_in_live_tenant_config`): the already-provisioned tenant configs are
  updated and the channel restarted as part of rollout.

---

## 13. Verification strategy

Per `docs/packets/audits/VERIFYING_A_PACKET.md` and `DISPATCHER_FAMILY.md`:

- Each mode × version gets a byte-fixture test with a `// packet-audit:verify` marker, an audit
  report under `docs/packets/audits/<version>/`, and (tier-1) a pinned evidence record; the
  matrix cell promotes from `❌` only when **every** arm for that version verifies. Dispatch is
  via the `packet-verifier` / `dispatcher-family-implementer` agents per family × version.
- Mode-byte enumeration alone is **not** a pass (memory
  `feedback_dispatcher_mode_byte_is_false_pass`): every arm with a body is fully encoded and
  byte-fixtured.
- Standard build gate for every changed module: `go test -race ./...`, `go vet ./...`,
  `go build ./...`, `docker buildx bake atlas-rps` (and every service whose `go.mod` was
  touched), `tools/redis-key-guard.sh` — all clean (CLAUDE.md build rules).

---

## 14. Testing strategy

- **atlas-rps unit tests:** `adjudicate.go` (all 9 throw combinations → win/lose/tie), ladder
  resolution (advance, max-rung force-collect, prize lookup), processor state transitions
  (start→select→win→continue→…; tie replay; loss end; quit no-payout), and the sweeper
  (expired → disposed, no payout). RNG injected for determinism. Builder-pattern test setup
  (no `*_testhelpers.go`).
- **Packet tests:** byte fixtures per mode × version (§13).
- **Channel handler tests:** mode decode → correct command emission; validator gating.
- **atlas-tenants tests:** `rps-rewards` transform/extract round-trip + mock interface parity.
- **Saga tests:** `StartRPSGame` payload marshal/unmarshal; entry-saga step ordering.

---

## 15. Service registration & infra

New Go service `atlas-rps` (REST + Kafka; **no LB socket port** — not a login/channel socket):

- `.github/config/services.json` — add the `atlas-rps` entry (mirroring chalkboards).
- `docker-bake.hcl` — add `"atlas-rps"` to the hand-synced `go_services` list (HCL can't read
  JSON — memory `reference_docker_bake_hand_synced`).
- Shared root `Dockerfile` — no change (parameterized by `SERVICE`); **no new shared lib**, so
  no new `COPY`/`go.work` lines. (libs/atlas-saga is an existing lib.)
- k8s manifests `deploy/k8s/base/atlas-rps.yaml` (+ overlay wiring), readiness probe on
  `/api/readyz` (memory `bug_readiness_probe_path_under_api_basepath`).
- `go.work` — add `./services/atlas-rps/atlas.com/rps` if the workspace enumerates services.

---

## 16. Open items carried to plan/execute (verify, don't invent)

These are IDA/WZ verifications, not design ambiguities — they are resolved with tooling during
plan/execute, and each is a hard gate before the corresponding cell/behavior is claimed done:

1. **Per-version mode bytes & frame layouts** for `RPS_GAME` (the `CRPSGameDlg::OnPacket` switch)
   and `RPS_ACTION` (the `OnBt*`/`SendSelection`/`Update` senders) — IDA-verified per version
   (v83/v84/v87/v95/jms185), populated into the per-version `operations` tables.
2. **OnBtRetry vs tie-redraw semantics** and whether the client expects a distinct tie frame or a
   tie outcome code in the result frame (FR-2.4, PRD §9) — IDA-confirmed against `CRPSGameDlg`.
3. **`Update` sub-action** meaning — confirm whether it is server-relevant or a client-only tick.
4. **Reward-ladder contents** — concrete item ids, quantities, and the meso escalation curve
   sourced from Cosmic `9000019.js` and verified against local WZ/item data.
5. **Quit-before-collect** stays Cosmic default (forfeit); confirm `OnBtExit` vs `OnBtContinue`
   client semantics (PRD §9).
6. **v92** parked with a documented blocker (needs a v92 IDB), to unblock later — mirrors
   task-086 mount-food.

---

## 17. Out of scope

Per PRD §2 non-goals: Player-NPC spawning at the game park; any client-side changes; gambling
rate-limiting beyond the meso bet; an admin UI for the ladder beyond the config resource;
persisting game history/leaderboards. v92 support (parked, §16.6).

---

## 18. Task breakdown preview (for /plan-task)

Rough implementation order (each a plan milestone):

1. `libs/atlas-saga` — `StartRPSGame` action + payload + unmarshal.
2. `atlas-rps` skeleton — service scaffold, session model/builder/registry, config load,
   REST read endpoint, service registration (services.json, docker-bake, k8s).
3. `atlas-rps` logic — adjudication + ladder + processor + kafka command/event topics + sweeper.
4. `libs/atlas-packet` — `RPS_GAME` clientbound + `RPS_ACTION` serverbound dispatcher families
   (v83 first, then v84/v87/v95/jms185), byte-fixture verified.
5. `atlas-channel` — handler (+validator) + writer + event consumer + tenant opcode/operations
   wiring; live-config patch.
6. `atlas-saga-orchestrator` — `StartRPSGame` dispatch; `atlas-rps` payout saga.
7. `atlas-tenants` — `rps-rewards` config resource + seed data.
8. `atlas-npc-conversations` — `start_rps_game` operation + npc-9000019 seed conversations.
9. Reward-ladder content (Cosmic-sourced, WZ-verified) + end-to-end verification + full gate.

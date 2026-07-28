# Miniroom Minigames: Omok and Match Cards — Design

Task: task-133-miniroom-minigames
Status: Approved design (Phase 2)
Date: 2026-07-04
Inputs: `prd.md` (approved), Cosmic source (local checkout, cited as `<cosmic>/src/main/java/...`), four codebase surveys (atlas-channel interaction surface, libs/atlas-packet inventory, Cosmic semantics extraction, new-service scaffolding survey).

---

## 1. Decision Summary

| # | Decision | Choice |
|---|---|---|
| D1 | Room/game state ownership | New `atlas-mini-games` service; **in-memory** tenant-partitioned registry (immutable Room models swapped under a `sync.Once` singleton + `sync.RWMutex`), per PRD §6. Redis `TenantRegistry` considered and rejected (§11-A). Consequence: `replicas: 1`. |
| D2 | Room identity | `roomId = ownerId` (character id). The balloon's `gameId` int and the client's VISIT `serialNumber` both carry it. No ID generator needed. |
| D3 | Validation placement | All CREATE/VISIT validation runs **in atlas-mini-games** on command receipt (REST fan-out to character/data/chalkboards/inventory), never in the channel. Channel arms are thin command emitters (merchant pattern). |
| D4 | Records persistence | GORM `game_records` table in atlas-mini-games; surrogate uuid PK + `(tenant_id, character_id, game_type)` unique index (gachapon PR #745 precedent). Both-player game-end update via **`db.Transaction` directly** — `database.ExecuteTransaction` is a confirmed no-op (`libs/atlas-database/transaction.go:9`) and task-119 has not landed. |
| D5 | Kafka keying / room serialization | Commands keyed by `characterId` (channel doesn't know roomId for bodyless modes); room mutations serialized by the registry write lock, not by partition ordering. Events keyed by `mapId` (chalkboards pattern). |
| D6 | Clientbound gameplay packets | New **discrete-per-mode** arms of the graduated `CMiniRoomBaseDlg::OnPacketBase` dispatcher family (`CharacterInteraction` writer), mode bytes resolved via `WithResolvedCode("operations", KEY)` — never literals. |
| D7 | Balloon packet | Implement `UPDATE_CHAR_BOX` (fname `CUser::OnMiniRoomBalloon`; registry opcodes exist unverified in all five versions) as a standard (non-dispatcher) packet; map the **already-registered but never-template-mapped** `MiniRoom` writer to it. |
| D8 | Balloon on map entry | Standalone `UPDATE_CHAR_BOX` announces from a `spawnMiniGamesForSession` step in `SpawnForSelf` (merchant/chalkboard precedent). The hardcoded `w.WriteByte(0) // mini room` at `libs/atlas-packet/character/clientbound/spawn.go:129` stays untouched (PRD Open Q5 resolved; fallback documented in §11-D). |
| D9 | Turn authority | Server tracks `currentTurn` and rejects out-of-turn/invalid moves by dropping the command (Cosmic sends no error packet for these either). Wire bytes mirror Cosmic exactly. |
| D10 | Retreat (take-back) | **Cosmic does not implement retreat at all** (verified: no redo/retreat opcode, packet, or logic anywhere in `PlayerInteractionHandler.java` / `MiniGame.java` / `PacketCreator.java`). Semantics and clientbound bodies must be IDA-derived from the v83 client (gate G2, §13). PRD FR-5.3's "per Cosmic semantics" is unsatisfiable as written; the authoritative reference is the client. |
| D11 | Cosmic error 5 (event instance) | Omitted entirely — Atlas has no event-instance concept; no check, no code path emits 5 (PRD Open Q2 resolved). |
| D12 | Tournament | Out of scope (PRD non-goal; Cosmic doesn't implement it; encode `tournament=false`, no round byte). |

---

## 2. Architecture Overview

```
 client ── PLAYER_INTERACTION (serverbound 0x7B @ v83) ──▶ atlas-channel
                                                             │ character_interaction.go arms
                                                             │ (thin emitters, merchant pattern)
                                                             ▼
                                              COMMAND_TOPIC_MINI_GAME (key: characterId)
                                                             │
                                                             ▼
                                                     atlas-mini-games
                                          ┌────────────────────────────────────┐
                                          │ game registry (in-mem, tenant map) │
                                          │ omok / matchcards engines (pure)   │
                                          │ game_records (GORM/Postgres)       │
                                          │ REST: records + rooms-in-field     │
                                          └────────────────────────────────────┘
                                                             │
                                              EVENT_TOPIC_MINI_GAME_STATUS (key: mapId)
                                                             │
                                                             ▼
 client ◀── CharacterInteraction / MiniRoom writers ── atlas-channel status consumer
            (session-targeted via IfPresentByCharacterId; field via ForSessionsInMap)
```

Validation dependencies (REST, from atlas-mini-games): atlas-character (`Hp() > 0` alive check), atlas-data (map `fieldLimit`), atlas-chalkboards (open-chalkboard check), atlas-inventory (`characters/%d/inventory`, item possession). Teardown inputs (Kafka, consumed by atlas-mini-games): `EVENT_TOPIC_SESSION_STATUS` `DESTROYED` (emitted by `session.Processor.Destroy`, `services/atlas-channel/atlas.com/channel/session/processor.go:329`) plus the character-status map-change/logout consumer mirroring `atlas-chalkboards/kafka/consumer/character/consumer.go`.

The channel-side vertical copies the merchant template end-to-end: serverbound stub → domain processor → `producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(provider)` → service → status topic → `kafka/consumer/minigame/consumer.go` (`InitConsumers`/`InitHandlers`) → `session.Announce(...)(writer)(body)`.

---

## 3. Game Semantics Specification (Cosmic-derived)

Everything in this section was extracted verbatim from the local Cosmic checkout on 2026-07-04. Server state below is authoritative in atlas-mini-games; wire bytes replicate Cosmic exactly.

### 3.1 Omok

- Board: 15×15, stored flat; Cosmic uses `piece[250]` with `slot = y*15 + x + 1` (`MiniGame.java:431-475`). Atlas engine: `[225]byte`, 0-indexed, same coordinate order.
- Placement validity: cell must be empty; that is the **only** placement rule. No double-three / renju / forbidden moves exist in the reference.
- Win detection: five **or more** in a row (a 5-window scan over 4 directions — horizontal, vertical, two diagonals — `searchCombo`/`searchCombo2`, `MiniGame.java:477-516`). No overline restriction.
- On win: award result, set the next-game first-mover byte (`0` after owner win, `1` after visitor win; initial value `1` — `MiniGame.java:52` default, assignments in `setPiece`), wipe the whole board for the rematch.
- Skip: broadcasts the skip arm and (Atlas addition) toggles server `currentTurn`. Cosmic tracks no turn state; we must, to satisfy FR-5.1's server-side turn validation.
- Retreat: absent from Cosmic (D10). Flow shape (from the existing serverbound decoders `operation_memory_game_retreat_answer.go`, fname `COmokDlg::OnRetreatRequest`): ASK_RETREAT (bodyless) forwarded to opponent; RETREAT_ANSWER carries `response bool`. On accept the server pops N stones and restores turn to the value the **client** displays — N and the turn rule come from IDA gate G2. On decline, forward the denial.

### 3.2 Match Cards

- `nGameSpec` 0/1/2 at create → `matchesToWin` 6/10/15 → deck of 12/20/30 cards (each pair id appears twice), shuffled **server-side at START**, not at create (`PlayerInteractionHandler.java:407`; deck build `MiniGame.java:342-396`).
- First flip (`turn==1`): record `firstSlot`, forward the pick **to the opponent only** (`getMatchCardSelect` with `turn=1`: mode, `1`, slot).
- Second flip (`turn==0`): compare card ids; broadcast mode, `0`, slot, firstSlot, `type` where type ∈ {0: owner mismatch, 1: visitor mismatch, 2: owner match, 3: visitor match} (`PlayerInteractionHandler.java:460-484`, `PacketCreator.java:4927-4939`).
- Turn retention is a **client convention driven by the type byte** (match retains, mismatch passes) — Cosmic tracks no turn. Atlas tracks `currentTurn` server-side with exactly that rule and validates FLIP_CARD senders.
- End: when `ownerPairs + visitorPairs == matchesToWin`; more pairs wins, equal pairs tie (`MiniGame.java:300-328`).

### 3.3 Tie, forfeit, scoring, records

- Tie request forwarded to opponent only, and only if the requester hasn't been denied this game (`isTieDenied`, deny bits OR'd into the in-progress bitfield, `MiniGame.java:220-238`). Answer: accept → draw; decline → set denier's bit + forward denial.
- Forfeit (GIVE_UP): opponent wins with forfeit flag (`PlayerInteractionHandler.java:411-425`). Visitor leaving/being expelled/disconnecting mid-game → owner forfeit-win (`removeVisitor`, `MiniGame.java:137-156`); owner leaving mid-game closes the room and the visitor takes the forfeit win (same resolution path, mirrored).
- Session score (per-room, **not persisted**, PRD FR-9.2): winner +50 (suppressed when the loser forfeited and already has ≥4 forfeits this session), loser +15 normal / −15 forfeit, tie +10 each with a 5-minute tie-score cooldown (`MiniGame.java:240-298`). Scores reset when a **different** visitor joins (`lastVisitor` tracking, `MiniGame.java:99-108`).
- Persistent records: winner `wins+1`, loser `losses+1`, tie `ties+1` each, per game type, both rows in **one `db.Transaction`** at each game end. Absent rows read as zeros.
- Exit-after-game flags: set/cancel any time; cleared on START; honored at game end by closing that player's side (`MiniGame.java:206-218`, `minigameMatchFinished`).
- Rematch: board/deck/pair-counters/deny-bits reset; room, visitor, session scores, and first-mover byte persist. `minigameMatchFinish()` is the double-resolution guard (`MiniGame.java:187-194`); the Atlas engine keeps the same idempotence (game-end resolution is a single state transition under the registry lock).

### 3.4 Status and error codes

- Leave statuses (existing `InteractionLeave` mode 10, slot + status): 3 = room closed (visitor on owner-leave), 4 = you left, 5 = expelled (`PacketCreator.java:4848`, `MiniGame.java:120-156`).
- Enter/create errors (existing `InteractionEnterResultError`: mode, `0`, code — byte-identical to Cosmic `getMiniRoomError`, `PacketCreator.java:4741-4747`): dead → 4, fieldLimit `0x80` → 11, chalkboard open → 13, item missing → 6, room gone → 1, room full / already inside → 2, wrong password → 22. "Already in a miniroom" on CREATE has no Cosmic code (its client prevents it); we send 6 ("character unable") as a chosen convention — flagged as such, not claimed as client parity.

---

## 4. atlas-mini-games Service Design

Layout (`services/atlas-mini-games/atlas.com/mini-games/`, module `atlas-mini-games`), composed from the two best references — atlas-chalkboards (registry + Kafka + REST skeleton) and atlas-buddies (GORM layer):

```
main.go                          bootstrap: db(SetMigrations), REST, consumers, readiness
logger/, rest/                   standard
kafka/message/minigame/kafka.go  Command[E]/StatusEvent[E] envelopes + type consts
kafka/consumer/minigame/         command consumer (InitConsumers/InitHandlers)
kafka/consumer/character/        teardown consumer (map-change/logout; chalkboards pattern)
kafka/consumer/session/          teardown consumer (EVENT_TOPIC_SESSION_STATUS DESTROYED)
kafka/producer/producer.go       generic ProviderImpl
game/model.go, builder.go        immutable Room model (see below)
game/registry.go                 singleton sync.Once + RWMutex, map[tenant]map[roomId]Room
game/processor.go                Processor iface + Impl; pure Method(mb) + MethodAndEmit
game/producer.go                 status-event providers (key = mapId)
game/resource.go, rest.go        GET rooms-in-field (JSON:API)
game/omok/engine.go              pure functions: Place, DetectWin, Retreat
game/matchcards/engine.go        pure functions: BuildDeck, Shuffle(rand), Flip
record/entity.go                 GORM entity + Migration (AutoMigrate + surrogate-PK DDL)
record/administrator.go          writes incl. ApplyGameEnd (both rows, one db.Transaction)
record/provider.go, model.go     reads (absent → zeros)
record/resource.go, rest.go      GET /characters/{id}/game-records
```

**Room model (immutable, swapped under registry write lock):** roomType (OMOK|MATCH_CARDS), ownerId, field (world/channel/map/instance), title, private, password, pieceType (Omok 0-11 / MatchCards 0-2), visitorId (0 = empty), visitorReady, inProgress, deny-tie bits, exitAfter flags (owner/visitor), firstMover byte, currentTurn slot, session scores + forfeit counters + lastVisitorId, and the game payload (Omok `[225]byte` board + move history for retreat; MatchCards deck `[]uint32`, firstSlot, pair counters). Copy-on-write via builder; the board array is a value type so copies are trivial at this scale.

**Registry:** `map[tenant.Model]map[uint32]Room` plus a `characterId → roomId` membership index (owner and visitor both). All command handling: take write lock → load room → validate → build next room → swap → collect emissions into `message.Buffer` → unlock → persist/`Emit`. DB writes (game end) happen after the state transition computes both records' deltas; the transaction commits before events emit so a crash never double-counts (re-emission is impossible: the in-memory game already reset; worst case is a lost broadcast, remedied by relog).

**Command handling (one handler per command type, chalkboards consumer idiom):** CREATE, VISIT, LEAVE, CHAT, READY, UNREADY, START, MOVE_STONE, FLIP_CARD, REQUEST_TIE, ANSWER_TIE, GIVE_UP, REQUEST_RETREAT, ANSWER_RETREAT, EXPEL, SKIP, EXIT_AFTER_GAME, CANCEL_EXIT_AFTER_GAME. Validation chains per §3.4; CREATE fan-out order: alive → fieldLimit → chalkboard → item → already-in-room. Invalid gameplay commands (out of turn, occupied cell, bad card index, non-owner START, START before visitor ready) are dropped with a structured warn log — no error packet, matching Cosmic.

**Teardown:** session DESTROYED or character map-leave/logout → same path as an explicit LEAVE for that character (forfeit resolution when a game is running, per §3.3).

**REST:**
- `GET /api/characters/{characterId}/game-records` — `game-records` resources (`characterId`, `gameType`, `wins`, `ties`, `losses`); absent rows returned as zeroed resources (decision: return them, so callers never special-case).
- `GET /api/worlds/{w}/channels/{c}/maps/{m}/instances/{i}/games` — open rooms in a field (id, type, title, private, pieceType, occupancy, inProgress) for the channel's map-entry balloon spawn. Mirrors the chalkboards field resource shape (`atlas-chalkboards/chalkboard/resource.go:22-27`).

---

## 5. Kafka Contract

Envelopes follow `atlas-chalkboards/kafka/message/chalkboard/kafka.go` exactly (generic `Command[E]`/`StatusEvent[E]` with `TransactionId/WorldId/ChannelId/MapId/Instance/CharacterId/Type/Body`; tenant in headers).

**`COMMAND_TOPIC_MINI_GAME`** (channel → mini-games, key `characterId`): the 18 types above. Bodies: CREATE{roomType, title, private, password, pieceType}; VISIT{roomId(serialNumber), password}; CHAT{message}; MOVE_STONE{x, y, stoneType}; FLIP_CARD{first, cardIndex}; ANSWER_TIE{accept}; ANSWER_RETREAT{accept}; all others empty.

**`EVENT_TOPIC_MINI_GAME_STATUS`** (mini-games → channel, key `mapId`). Every event body carries `roomId`, `ownerId`, `visitorId` (0 if none) so the channel targets sessions without lookups:

| Event | Extra body | Channel writes |
|---|---|---|
| `CREATED` | room snapshot + owner record | ENTER_RESULT success (GameRoom) to owner; balloon spawn to field |
| `CREATE_ERROR` / `ENTER_ERROR` | code | EnterResultError to the actor |
| `ENTERED` | visitor slot + both records + room snapshot | full room to visitor; `InteractionEnter` (game-visitor + record) to owner; balloon update |
| `LEFT` | slot, leaveStatus, expelled/forced flags | `InteractionLeave(slot, status)` to affected session(s); balloon update |
| `ROOM_CLOSED` | closeStatus for visitor | Leave to visitor (status 3/4), balloon **remove** to field |
| `CHAT` | slot, message | `CharacterInteractionChatBody(slot, msg)` to both |
| `READY` / `UNREADY` | — | ready/unready arm to both |
| `STARTED` | firstMover; MatchCards: deck (slot-ordered card ids) | start arm to both; balloon → in-progress |
| `STONE_PLACED` | x, y, stoneType | move arm to both |
| `CARD_FLIPPED` | phase(first/second), slot, firstSlot, resultType | first flip → **opponent only**; second → both |
| `TIE_REQUESTED` / `RETREAT_REQUESTED` | — | request arm to opponent only |
| `TIE_ANSWERED` / `RETREAT_ANSWERED` | accept (+retreat pop data per G2) | deny arm to requester, or game-end/board-pop to both |
| `SKIPPED` | who byte | skip arm to both |
| `GAME_ENDED` | resultType(win/tie/forfeit), winnerSlot, both refreshed records + session scores | GET_RESULT arm to both; balloon → open |
| `BALLOON_UPDATED` | full balloon payload (type, roomId, title, hasPassword, pieceType, occupancy, capacity 2, inProgress) | `UPDATE_CHAR_BOX` to field |

(`BALLOON_UPDATED` is folded into the lifecycle events where noted; it also exists standalone so the service can refresh the balloon without a second lifecycle meaning.)

---

## 6. Packet Work (libs/atlas-packet)

### 6.1 New dispatcher-family arms (CharacterInteraction / `CMiniRoomBaseDlg::OnPacketBase` — GRADUATED family, lint-clean required)

One consolidated file `interaction/clientbound/interaction_minigame.go`, discrete struct per mode per `docs/packets/DISPATCHER_FAMILY.md` (no shared-by-shape structs, no literal mode bytes, every body func fixes its key and resolves via `WithResolvedCode("operations", KEY)`):

| Struct | v83 mode | Body (Cosmic layout) |
|---|---|---|
| `InteractionMiniGameReady` / `UnReady` | 58 / 59 | mode only |
| `InteractionMiniGameStartOmok` | 61 | mode, firstMover byte |
| `InteractionMiniGameStartMatchCards` | 61 | mode, firstMover, count byte (12/20/30), count×int32 cardId |
| `InteractionMiniGameMoveStone` | 64 | mode, int32 x, int32 y, byte stoneType |
| `InteractionMiniGameCardSelect` | 68 | mode, byte turn, slot [, firstSlot, resultType when turn==0] |
| `InteractionMiniGameRequestTie` / `AnswerTie` | 50 / 51 | mode only / mode (deny forward) |
| `InteractionMiniGameSkip` | 63 | mode, byte who (owner 0x01 / visitor 0x00 — Cosmic writes the visitor variant as a short, byte-equivalent; meaning verified at G5) |
| `InteractionMiniGameResult` | 62 (`GET_RESULT`, clientbound-only key `MEMORY_GAME_RESULT`) | mode, resultType(0 win/1 tie/2 forfeit), bool visitorWon, then the tie/non-tie record-refresh layout of `PacketCreator.java:4785-4830` |
| `InteractionMiniGameRetreatRequest` / `RetreatAnswer` | 54 / 55 | **IDA-derived (G2)** — no Cosmic reference exists |

START uses one operations key with two structs (Omok vs MatchCards bodies) — same key, discrete arms, both fixtured. Reused as-is: `InteractionEnter`, `InteractionEnterResultSuccess`/`Error`, `InteractionChat`, `InteractionLeave` and their body funcs. The existing `interaction.Room` game branch and `GameRecord{marker, wins, ties, losses, points}` 5×int encoding already match Cosmic `getMiniGame` (marker = game type int, then W/T/L, then session score); the room-enter byte layout gets a fresh fixture against `getMiniGame`/`getMatchCard` (note the byte-2 difference: 0 for Omok, 2 for MatchCards) as part of G5.

Per mode: `run.go candidatesFromFName` case `CMiniRoomBaseDlg::OnPacketBase#<Mode>`, synthetic export entry, audit report, byte-fixture with `// packet-audit:verify` marker (pattern of `interaction/clientbound/interaction_test.go:27-76`), pinned evidence — for every version with an available IDB; others wired + bannered unverified. `packet-audit dispatcher-lint`, `matrix --check`, `fname-doc --check`, `operations --check` must all exit 0.

### 6.2 UPDATE_CHAR_BOX (balloon) — new standard packet

- New file `interaction/clientbound/mini_room_balloon.go`: `MiniRoomBalloon` (int32 characterId, byte roomType, int32 roomId, string title, bool hasPassword, byte pieceType, byte occupancy, byte capacity=2, byte inProgress — Cosmic `addAnnounceBox`, `PacketCreator.java:2199-2208`) and `MiniRoomBalloonRemove` (int32 characterId, byte 0).
- Writer name: reuse **`MiniRoom`** — already in the channel writer list (`main.go:783`) but mapped in **zero** templates today, which is why the merchant personal-shop balloon announces (`kafka/consumer/merchant/consumer.go:133,171`, `map/consumer.go:683`) currently resolve to nothing. Adding the template entry activates those paths too; the existing `MiniRoomBase.Spawn` shop-branch layout gets an IDA read-check on v83 as part of G3 so we don't crash clients via an unrelated feature. The game balloon itself routes through the new audited `MiniRoomBalloon` struct, not the legacy `Spawn` model.
- Opcodes per version already in the registries (all ❌/unverified): 0x0A5 (v83) / 0x0A8 (v84, corrected) / 0x0B0 (v87) / 0x0B8 (v95) / 0x0A3 (jms). Verify v83+v95 layouts in IDA (G3); fixture + evidence + matrix promotion for those; others wired + flagged.

### 6.3 Serverbound

All needed decoders exist (`operation_memory_game_{move_stone,flip_card,tie_answer,retreat_answer}.go`, `operation_create.go`, `operation_chat.go`; bodyless modes ride the base `Operation`). One gate: the game-room **VISIT-with-password** layout (Cosmic reads a password string; Atlas `operation_visit.go` decodes a trade-shaped body) — verify on v83 (G4) and extend the decoder or add a game-visit wrapper codec if the layouts diverge.

---

## 7. atlas-channel Integration

- **New domain package** `atlas.com/channel/minigame/`: `Processor` with `Create/Visit/Leave/Chat/Ready/.../Emit` methods, each `producer.ProviderImpl(l)(ctx)(minigame.EnvCommandTopic)(provider)` — clone of `merchant.Processor.PlaceShop`.
- **`character_interaction.go`**: CREATE arm's Omok/MatchCards branch, VISIT (game rooms), CHAT, EXIT, and all fourteen MemoryGame arms call the processor instead of decode-and-log. EXIT and CHAT are shared with trade/shop rooms and the channel doesn't know membership: emit to mini-games **in addition to** existing handling; the service ignores commands from non-members (logged at debug). The wire key typo `MEMORY_GAME_FIP_CARD` is load-bearing in live configs and code — keep it.
- **Status consumer** `kafka/consumer/minigame/consumer.go`: `InitConsumers`/`InitHandlers` (merchant idiom incl. tenant/world/channel guards), event→packet mapping per §5 table. Avatars/names for room/enter encodes fetched via `character.NewProcessor(l, ctx).GetById(...)` at encode time (existing pattern). Session targeting via `session.Processor.IfPresentByCharacterId`; field broadcasts via `_map.NewProcessor(l, ctx).ForSessionsInMap`.
- **Map entry**: add `spawnMiniGamesForSession` to the `SpawnForSelf` aggregator (`kafka/consumer/map/consumer.go:159`), REST-listing open rooms in the field from atlas-mini-games and announcing `MiniRoomBalloon` per room — exactly `spawnMerchantsForSession` (`map/consumer.go:666`) / `spawnChalkboardsForSession`.
- **Teardown**: nothing new to emit — `session.Processor.Destroy` already publishes `EVENT_TOPIC_SESSION_STATUS`/`DESTROYED`; mini-games consumes it (§4).

---

## 8. Seed Templates & Per-Version Wiring

Current state (verified by grep on 2026-07-04) and required work:

| Version | `CharacterInteractionHandle` entry | MEMORY_GAME sb rows | `CharacterInteraction` writer | Work |
|---|---|---|---|---|
| gms_83 | ✓ | ✓ (full, `template_gms_83_1.json:571-584`) | ✓ | add cb game-mode ops + `MiniRoom` writer entry |
| gms_84 | ✓ | ✓ | verify | same as 83 (+ writer if absent) |
| gms_87 | **missing** | — | ✓ | add handler (validator!) + sb rows + cb ops + balloon |
| gms_92 | **missing** | — | **missing** | add everything |
| gms_95 | **missing** | — | ✓ | add handler + sb rows + cb ops + balloon |
| jms_185 | ✓ | **missing** | ✓ | add sb MEMORY_GAME rows + cb ops + balloon |

Rules: every handler entry carries `"validator": "LoggedInValidator"` (validator-less entries are silently dropped — known bug); mode values are version-dependent and must be enumerated per version from that version's `CMiniRoomBaseDlg::OnPacketBase` / handler switch where an IDB exists (v83, v95 currently loaded; the instance set rotates — `list_instances` and match binary names first). The client uses a **single mode enum for both directions** (Cosmic's `Action` enum), so the already-seeded v83/v84 serverbound values are the primary evidence for the clientbound tables on those versions; versions without a loaded IDB get values derived from the nearest verified version, bannered UNVERIFIED in the registry per the reshift-carryover discipline. Live tenants don't pick up seed changes: the rollout includes a live-tenant config PATCH runbook (handler + writer + operations + balloon writer, per tenant) and a channel restart, committed in the task docs (task-127's deployment notes are the format precedent).

---

## 9. Data Model

Per PRD §6, unchanged: `game_records` (uuid surrogate PK, `tenant_id`, `character_id`, `game_type` enum string, `wins/ties/losses` uint32 default 0, timestamps, unique `(tenant_id, character_id, game_type)`). `Migration` = `AutoMigrate` via `database.SetMigrations` (buddies pattern, `atlas-buddies/main.go:57`); the unique index declared in GORM tags. Room/game state is in-memory only (D1); no Redis.

## 10. Deployment / Wiring Checklist

- `.github/config/services.json` entry + `docker-bake.hcl` `go_services` (hand-synced, both required) + `go.work` line.
- Repo-root `Dockerfile`: no change (no new lib).
- `deploy/k8s/base/atlas-mini-games.yaml`: Deployment (modeled on `atlas-buddies.yaml` DB env: `DB_NAME: atlas-mini-games`, creds from `db-credentials`) + Service :8080 + readiness probe **`/api/readyz`** (served via `server.MountReadiness`, atlas-world precedent) + **`replicas: 1`** (in-memory registry, D1). Add to `base/kustomization.yaml`.
- `env-configmap.yaml`: `COMMAND_TOPIC_MINI_GAME`, `EVENT_TOPIC_MINI_GAME_STATUS`. Channel deployment gets the same two vars if not inherited from the shared configmap.
- Overlays: `overlays/main` + `overlays/pr` env/db-name-suffix/consumer-group patches and `gen-cleanup-env.sh` (all enumerate services by hand).

## 11. Alternatives Considered

- **A. Redis `TenantRegistry` for rooms** (the newer convention — chalkboards/messengers): survives restarts and permits >1 replica, but serializes a 225-cell board + history to Redis on every stone, complicates atomic read-modify-write across two structures (room + membership index), and the PRD explicitly accepts restart-drop and mandates in-memory. Rejected; revisit only if mini-games ever needs HA.
- **B. Records as columns on atlas-character** (Cosmic's model): rejected — bloats a core service with leaf-feature data, needs a cross-service migration, and violates "new service owns its domain".
- **C. Channel-side validation before emitting CREATE**: saves one round-trip but splits authority (already-in-room can only be checked in the service) and duplicates REST fan-out per channel. Rejected; the service validates everything (D3).
- **D. Embedding the announce-box in the character spawn packet** (Cosmic embeds it; `spawn.go:129` is a hardcoded 0): higher blast radius (touches every character spawn on every version) for the same visible result the standalone balloon achieves, and the merchant precedent proves the standalone path. Rejected for v1; documented fallback if live testing shows late entrants missing balloons (acceptance criterion 1 covers this).
- **E. Kafka key = roomId for per-room ordering**: channel can't compute roomId for bodyless modes without local membership tracking; cross-player ordering isn't guaranteed by partitioning anyway (two clients race in real time). Rejected in favor of characterId keying + registry-lock serialization (D5) — correctness comes from validation under the lock, not arrival order.

## 12. Testing Strategy

- **Engines (pure, no mocks)**: Omok win detection (all 4 directions, board edges, overline-wins-not-blocked, wipe-on-win), placement rejection (occupied, out of turn, out of bounds); MatchCards deck build (6/10/15), injected-rand shuffle determinism, first/second flip resolution matrix (types 0-3), turn retention, end/tie conditions; tie-deny bits; scoring incl. forfeit-farm guard and tie cooldown (injected clock); retreat per G2 findings.
- **Registry/processor**: `go test -race` with concurrent commands on one room and across rooms; teardown-equals-leave; record transaction (both rows or neither).
- **Packets**: byte fixtures per new mode × version-with-IDB with `packet-audit:verify` markers; `dispatcher-lint` (family is graduated — no baseline), `matrix --check`, `operations --check` all clean. Full per-mode bodies — mode-byte enumeration alone is a known false pass.
- **Channel**: handler-arm tests per existing `character_interaction` test conventions; consumer handler tests for session targeting vs field broadcast.
- **Verification gates before "done"**: `go test -race ./...`, `go vet ./...`, `go build ./...` in atlas-mini-games, atlas-channel, libs/atlas-packet; `docker buildx bake atlas-mini-games atlas-channel`; `tools/redis-key-guard.sh` from repo root; live v83 acceptance pass per PRD §10.

## 13. Implementation-Gate IDA Verifications

All against matched instances (`list_instances`, match binary name first; v83 + v95 loaded as of 2026-06-30, no v84/87/jms):

- **G1** — start-byte turn semantics: Cosmic's `loser` field is written `0` on owner win / `1` on visitor win (i.e. raw value = winner slot despite the name) with initial `1`; verify in v83 `COmokDlg`/`CMemoryGameDlg` start handling which slot the client actually grants the first move, and initialize server `currentTurn` to match. Wire bytes mirror Cosmic regardless.
- **G2** — retreat: derive the clientbound ASK_RETREAT/RETREAT_ANSWER result bodies, the number of stones popped on accept, and the post-retreat turn from v83 `COmokDlg` (no server reference exists). Server board mirrors exactly what the client pops.
- **G3** — `UPDATE_CHAR_BOX` layout on v83 + v95 (`CUser::OnMiniRoomBalloon`), including the personal-shop branch the newly-mapped `MiniRoom` writer activates.
- **G4** — serverbound game-room VISIT-with-password layout on v83 (Atlas `operation_visit.go` is trade-shaped; Cosmic reads a password string).
- **G5** — clientbound mode values + minigame body layouts on v95 (enum stability vs v83); skip `who` byte meaning; room-enter (`getMiniGame`) fixture bytes.
- Unresolved fnames or contradictory decompiles are stop-and-ask, never substituted.

## 14. PRD Open Questions — Resolutions

1. Tournament: excluded (D12). 2. Error 5: omitted (D11). 3. Omok rules: extracted — 5+ in a row wins, no forbidden moves, board wiped on win; retreat is client-derived (D10/G2). 4. Per-version codes/modes: §8 strategy (IDB-verified where loaded, derived + UNVERIFIED-bannered elsewhere). 5. Announce-box in spawn: standalone balloon on map entry (D8), spawn packet untouched, fallback documented (§11-D).

# Miniroom Minigames: Omok and Match Cards — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-07-04
---

## 1. Overview

MapleStory's social minigames — Omok (five-in-a-row on a 15×15 board) and Match Cards (memory/pairs) — are played inside minirooms opened from Etc items. A player with an Omok set (items 4080000–4080011, twelve piece-set variants) or a Match Cards deck (item 4080100) opens a game room on the map; a balloon appears over their head; a second player clicks the balloon to join; both ready up and play, with per-character win/loss/tie records shown in the room UI. The games are bet-free and are a staple of town and Free Market social life.

Atlas currently decodes but discards every minigame packet. All fourteen `MemoryGame*` interaction modes in `services/atlas-channel/atlas.com/channel/socket/handler/character_interaction.go` are decode-and-log only, and the CREATE arm for `OmokMiniRoomType`/`MatchCardMiniRoomType` logs and returns. The channel model's `NewOmokMiniRoom`/`NewMatchCardMiniRoom` constructors (`socket/model/mini_room.go`) have no callers. No service holds game/room state (grep across `services/` and `libs/` for `omok|matchcard|minigame` hits only the packet lib, the channel model, and the stub handler), and no win/loss record storage exists anywhere — the `GameRecord`/`MiniGameRecord` structs exist only as packet encoders in `libs/atlas-packet/interaction/`.

This task delivers playable Omok and Match Cards end-to-end on all supported tenant versions, backed by a new `atlas-mini-games` service that owns room state, game rules, and persistent per-character game records. The behavioral reference is Cosmic (`net/server/channel/handlers/PlayerInteractionHandler.java`, `server/maps/MiniGame.java`), with packet layouts verified against client IDBs per the packet-audit discipline.

## 2. Goals

Primary goals:
- Playable Omok and Match Cards rooms: create from item, join via map balloon, ready/start, full move-by-move gameplay, and correct game-end resolution (win, loss, tie, forfeit).
- Complete miniroom social flow: field balloon lifecycle (spawn/update/remove), in-room chat, expel, leave, exit-after-game.
- Persistent per-character, per-game-type win/tie/loss records, tenant-scoped, owned by the new `atlas-mini-games` service and displayed in the room UI.
- Full server-side validation on room creation and joins (alive, map fieldLimit, chalkboard conflict, item possession, room capacity/password/in-progress).
- Working on **all supported tenant versions** (gms_v83, gms_v84, gms_v87, gms_v92, gms_v95, jms_v185): seed templates, per-version `operations` mode tables, and live-tenant config patches.

Non-goals:
- Trade rooms, personal shops, hired merchants (separate miniroom types; merchant flow already owned by `atlas-merchant`).
- Tournament mode (the room packet's `tournament` flag + `round` byte). See Open Questions.
- Meso betting / wagers (not part of v83-era Omok).
- Spectators beyond the 2-player capacity; observer lists.
- atlas-ui changes.

## 3. User Stories

- As a player with an Omok set in my Etc inventory, I want to open an Omok room with a title (and optional password) so that another player can join me for a game.
- As a player with a Match Cards deck, I want to open a Match Cards room choosing a board size (12/20/30 cards) so we can play a memory game.
- As a passer-by, I want to see a game balloon over the owner's head and click it to join (entering the password if set) so that I can play.
- As a room owner, I want the game to start only when my visitor has readied and I press start, so games begin fairly.
- As an Omok player, I want the server to validate turns and detect five-in-a-row so the outcome is authoritative and cheat-proof.
- As a Match Cards player, I want card flips, matches, and scoring resolved server-side so both clients stay in sync.
- As a player, I want to request a tie, forfeit, or ask to take back a move (retreat), and have my opponent answer, matching official game etiquette.
- As a room owner, I want to expel a visitor before a game starts.
- As a player, I want my wins/ties/losses per game type recorded on my character and shown next to my name in the room, so my record persists across sessions.
- As a player who disconnects mid-game, I accept that I forfeit, so my opponent is not stuck in a dead room.

## 4. Functional Requirements

Behavioral reference: Cosmic `PlayerInteractionHandler.java` and `server/maps/MiniGame.java`. Values below were verified against the Cosmic source on 2026-07-04; anything not listed here must be extracted during the design phase, not invented.

### FR-1 Room creation (CREATE arm, roomType 1 = Omok, 2 = Match Cards)

1. Validation chain, in order, each failure sending the miniroom error packet with the given code (Cosmic `establishMiniroomStatus` + CREATE branch):
   - Character dead → error 4.
   - Map disallows minigames: `fieldLimit` bit `0x80` (Cosmic `FieldLimit.CANNOTMINIGAME`) set on the map (available via atlas-data map attribute `fieldLimit`) → error 11.
   - Character has an open chalkboard (atlas-chalkboards) → error 13.
   - Item possession: Omok requires item `4080000 + pieceType` with pieceType clamped to [0,11]; Match Cards requires item `4080100`. Missing → error 6.
   - Character already in a miniroom → reject (no double-rooms).
   - (Cosmic also returns error 5 for "in event instance"; Atlas has no event-instance concept — see Open Questions.)
2. On success: create the room in `atlas-mini-games` (owner, title, private flag, password, pieceType; Match Cards `pieceType` 0/1/2 sets matches-to-win 6/10/15 = 12/20/30 cards), send the room packet to the owner (ENTER_RESULT success with `GameMiniRoom` encoding incl. owner's game record), and broadcast the field balloon (FR-3).
3. The Omok/Match Cards item is **not consumed**.

### FR-2 Visitor join (VISIT arm)

1. Validations: room exists on the character's field; room not full (capacity 2); game not in progress; password matches when the room is private; joiner passes the same alive/chalkboard checks. Failures send the corresponding enter-error code (enumerate per version during design; the clientbound `CharacterInteractionEnterErrorMode` table already exists in `libs/atlas-packet/interaction/clientbound/interaction_body.go`).
2. On success: visitor receives the full room packet (both players' records); owner receives the visitor-enter packet; the field balloon updates to "full".

### FR-3 Field balloon lifecycle

1. On room creation, broadcast the game balloon above the owner to the field (Cosmic `addOmokBox`/`addMatchCardBox`, `UPDATE_CHAR_BOX` opcode). **This clientbound packet does not exist in `libs/atlas-packet` yet and must be implemented for all supported versions.**
2. Balloon updates on occupancy/state change (open ↔ full, game in progress) and is removed when the room closes.
3. Players entering the map while a room is open must see the balloon (announce-box section of the character spawn packet — the spawn writer already models miniroom data; verify and wire).

### FR-4 Ready / start

1. Visitor toggles ready/unready; both states broadcast to the room.
2. Only the owner can start, and only when the visitor is ready. Start deals the initial state: Omok — empty 15×15 board; Match Cards — shuffled deck of 12/20/30 cards (server-side shuffle, layout sent in the start packet).
3. Turn order follows Cosmic `MiniGame.java` semantics (including who starts subsequent games in the same room).

### FR-5 Omok gameplay

1. MOVE_STONE (serverbound decoder exists: `operation_memory_game_move_stone.go`): server validates it is the sender's turn and the cell is empty, then broadcasts the placement.
2. Win detection server-side: five in a row horizontally/vertically/diagonally ends the game. Exact edge rules (overlines, forbidden double-three) must match Cosmic `MiniGame.java` — extract during design.
3. Retreat (take-back): requester asks, opponent answers (decoder `operation_memory_game_retreat_answer.go`); on accept the appropriate stones are removed and turn is restored per Cosmic semantics.
4. Skip: a player may pass their turn (SKIP mode); broadcast and switch turn.

### FR-6 Match Cards gameplay

1. FLIP_CARD (decoder exists: `operation_memory_game_flip_card.go`): server validates turn and card index, broadcasts the flip; second flip resolves match/mismatch, updates the in-room running score, and passes or retains the turn per Cosmic `MiniGame.java`.
2. Game ends when all pairs are matched; higher pair count wins, equal counts tie.

### FR-7 Game end, tie, forfeit

1. Tie: requester asks, opponent answers (decoder `operation_memory_game_tie_answer.go`); accept ends the game as a tie for both.
2. Forfeit (GIVE_UP): forfeiter takes a loss, opponent a win.
3. Leaving mid-game, being expelled mid-game, or disconnecting mid-game counts as a forfeit.
4. On every game end: emit the result packet (score/records refresh), update both characters' persistent records (FR-9), reset the board for a rematch without closing the room, and update the balloon.

### FR-8 Room membership and closure

1. Expel: owner removes the visitor pre-game (leave packet with the expel status code).
2. Exit-after-game: a player may flag (and unflag) leaving after the current game; honored at game end.
3. Owner leaving closes the room: visitor is ejected with the correct status code, balloon removed, room deleted.
4. In-room chat: routed through the room so both players see slot-attributed messages (clientbound `CharacterInteractionChatBody` exists; serverbound `operation_chat.go` exists).
5. Session logout/disconnect and channel/map departure must tear down membership exactly like an explicit leave (with FR-7.3 forfeit when a game is running).

### FR-9 Persistent game records

1. `atlas-mini-games` persists per character, per game type (`OMOK`, `MATCH_CARDS`): `wins`, `ties`, `losses` — tenant-scoped (Cosmic equivalent: `omokwins/omokties/omoklosses/matchcardwins/matchcardties/matchcardlosses` on the character row).
2. Records are read when encoding room enter/refresh packets. The in-room encoding per Cosmic `PacketCreator.getMiniGame` is: int marker `1`, wins, ties, losses, then the room-scoped running `score` — so `score` is **not** persisted; it is per-room session state.
3. Absent records read as zeros (no seeding required).
4. Updates are transactional per game end (both players updated; a crash must not double-count).

### FR-10 New service: atlas-mini-games

1. Owns: tenant-scoped in-memory room registry (singleton + `sync.RWMutex` per project pattern), Omok and Match Cards game engines, and the `game_records` table.
2. Follows the standard service layout (`services/atlas-mini-games/atlas.com/mini-games/`), immutable models + builders, Processor interface/Impl, curried Kafka consumer registration, JSON:API REST via api2go.
3. New-service wiring checklist: `.github/config/services.json` **and** the hand-synced `go_services` list in `docker-bake.hcl`; `go.work` entry; `deploy/k8s/base/atlas-mini-games.yaml` with readiness probe path **`/api/readyz`** (not `/readyz` — known wedge); Kafka topic env vars; DB migration.

### FR-11 Channel integration (all supported versions)

1. `character_interaction.go` arms for CREATE (Omok/Match Cards), VISIT (game rooms), CHAT (game rooms), and all fourteen `MemoryGame*` modes emit commands to `atlas-mini-games` instead of decode-and-log.
2. `atlas-channel` consumes minigame status events and writes the clientbound packets to the right sessions (room-scoped) or the field (balloon).
3. Missing clientbound bodies must be added to `libs/atlas-packet/interaction/clientbound/` (ready/unready/start-omok/start-matchcard/move/flip/tie/retreat-request+answer/skip/game-result/leave-status-codes) plus the new `UPDATE_CHAR_BOX` balloon packet — each with per-version layout verification and byte-fixture tests. Mode bytes must be config-resolved via the tenant `operations` table, never hard-coded (miniroom is a dispatcher-shaped family: `CMiniRoomBaseDlg::OnPacketBase`).
4. Seed templates for **all six versions** get the new handler entries (every handler entry MUST carry a `validator` — validator-less entries are silently dropped), writer entries, and miniroom `operations` mode-table rows. Mode values are version-dependent; verify per version, do not copy v83's table.
5. Live tenants do not pick up seed-template changes: document and apply the live-tenant config PATCH + channel restart as part of rollout.

## 5. API Surface

### REST (atlas-mini-games, JSON:API)

- `GET /api/characters/{characterId}/game-records` — list of `game-records` resources for the character (tenant from header). Attributes: `characterId`, `gameType` (`OMOK` | `MATCH_CARDS`), `wins`, `ties`, `losses`. Missing rows are returned as zeroed resources (or omitted — design decides; callers must treat absent as zero).
- No REST mutation of records (game-end driven only).

### Kafka

- `COMMAND_TOPIC_MINI_GAME` (channel → mini-games): `CREATE`, `VISIT`, `LEAVE`, `CHAT`, `READY`, `UNREADY`, `START`, `MOVE_STONE`, `FLIP_CARD`, `REQUEST_TIE`, `ANSWER_TIE`, `GIVE_UP`, `REQUEST_RETREAT`, `ANSWER_RETREAT`, `EXPEL`, `SKIP`, `EXIT_AFTER_GAME`, `CANCEL_EXIT_AFTER_GAME`. All carry tenant headers + field (worldId/channelId/mapId/instance) + characterId.
- `EVENT_TOPIC_MINI_GAME_STATUS` (mini-games → channel): `CREATED`, `CREATE_ERROR` (code), `ENTERED`, `ENTER_ERROR` (code), `LEFT` (slot + status code), `CHAT`, `READY`, `UNREADY`, `STARTED` (initial state), `STONE_PLACED`, `CARD_FLIPPED`, `TIE_REQUESTED`, `TIE_ANSWERED`, `RETREAT_REQUESTED`, `RETREAT_ANSWERED`, `SKIPPED`, `GAME_ENDED` (result + refreshed records), `BALLOON_UPDATED`, `ROOM_CLOSED`.
- Exact message schemas follow the project's command/event envelope conventions; final naming at design time.

### Dependencies consumed

- atlas-data: map `fieldLimit` attribute (creation validation).
- atlas-inventory: item possession check (4080000–4080011, 4080100).
- atlas-chalkboards: open-chalkboard check.
- atlas-character: alive check + avatar/name for room packet encoding (channel side already has this via `character.NewProcessor`).

## 6. Data Model

New table in atlas-mini-games' database:

`game_records`
- `id` — uuid, surrogate PK (never a business-key-only PK — known multi-tenant collision pattern)
- `tenant_id` — uuid, not null
- `character_id` — uint32, not null
- `game_type` — string enum (`OMOK`, `MATCH_CARDS`), not null
- `wins`, `ties`, `losses` — uint32, not null, default 0
- `created_at` / `updated_at`
- Unique index on (`tenant_id`, `character_id`, `game_type`)

Room and game state (board, deck, turn, running score, ready flags, exit-after flags) is **in-memory only** in the mini-games registry, keyed by tenant + field + ownerId. A service restart drops open rooms (acceptable; same class as other in-memory registries). No Redis; if caching is ever added it must route through `libs/atlas-redis`.

## 7. Service Impact

- **atlas-mini-games (new)** — room registry, game engines, records persistence, REST, Kafka command consumer + event producer. Full new-service checklist per FR-10.3.
- **atlas-channel** — replace stub arms in `character_interaction.go` with command emission; new Kafka consumer for minigame status events; writer registrations for new packets; wire announce-box data into the spawn path.
- **libs/atlas-packet** — new clientbound minigame bodies + `UPDATE_CHAR_BOX` packet; reuse existing serverbound decoders and `Room`/`Visitor`/`GameRecord` encoders; byte-fixture tests per version.
- **Tenant seed templates (all six)** — handler entries (with validators), writer entries, miniroom `operations` mode rows.
- **Live tenant configs** — PATCH runbook + channel restart at rollout.
- **deploy/k8s** — new service manifest (readiness `/api/readyz`), env wiring.
- **No impact**: atlas-ui, atlas-merchant (shop miniroom types untouched).

## 8. Non-Functional Requirements

- **Multi-tenancy**: every Kafka message carries tenant headers; registry partitions by tenant; all DB rows tenant-scoped; REST derives tenant from context.
- **Server-authoritative gameplay**: turn ownership, cell/card validity, win detection, and record mutation happen only in atlas-mini-games; the client is never trusted.
- **Concurrency**: registry access race-free (`go test -race` clean); simultaneous commands for one room serialize (per-room ordering — Kafka key by room owner or explicit lock).
- **Consistency**: record updates for both players commit in one transaction (mind the known `ExecuteTransaction` no-op pattern — use the working transaction mechanism current at implementation time).
- **Observability**: structured logs for room lifecycle (create/enter/start/end/close) with tenant, field, characterIds.
- **Performance**: negligible scale (2-player rooms); no special requirements beyond not blocking the channel event loop on REST lookups during packet encoding.

## 9. Open Questions

1. **Tournament mode** — the room encoding carries a `tournament` flag + `round` byte (`socket/model/mini_room.go`); it corresponds to the client's GM-run Omok tournament UI. Cosmic does not implement it and its client flow is unverified. Excluded from v1; candidate follow-up after IDA verification of the client event flow.
2. **Cosmic error 5 (event instance)** — Atlas has no event-instance concept. Omit, or map to an equivalent (party-quest instance membership)? Design decision.
3. **Omok forbidden-move rules** (double-three, overline) and exact retreat/turn-order semantics — extract from Cosmic `MiniGame.java` during design; do not invent.
4. **Per-version leave/enter status codes and miniroom `operations` mode values** — enumerate from each version's IDB during design/implementation. Currently loaded IDA instances cover v83 and v95; other versions may need instance rotation (list instances and match binary names before reading).
5. **Announce-box in spawn packet** — the spawn writer models miniroom balloon data; confirm it round-trips for game rooms on all versions or fix.

## 10. Acceptance Criteria

- [ ] On a live v83 tenant: create an Omok room with item 4080000-family in inventory (each of title, private+password verified), balloon appears; creation without the item fails with error 6; creation in a `fieldLimit & 0x80` map fails with error 11; creation with an open chalkboard fails with error 13.
- [ ] Second character joins via balloon (password enforced), readies; owner starts; a full Omok game plays to five-in-a-row; winner/loser records increment by exactly 1; tie flow, forfeit, retreat (accept + decline), skip, expel, exit-after-game, and owner-leave-closes-room all behave per FR-5/7/8.
- [ ] Match Cards playable end-to-end at all three board sizes with correct match/turn/score semantics and record updates.
- [ ] Disconnect mid-game forfeits and tears down the room; opponent receives win + correct leave packet.
- [ ] Records survive relog and are correct in the room UI encoding (marker 1, W/T/L, room-scoped score).
- [ ] `GET /api/characters/{id}/game-records` returns tenant-scoped records per §5.
- [ ] All six seed templates contain the new handler (with validator), writer, and `operations` entries; matrix/config checks pass; live-tenant PATCH runbook committed in the task docs.
- [ ] New clientbound packets have byte-fixture tests (full per-mode bodies — mode-byte enumeration alone is not verification) for every version with an available IDB; remaining versions wired and flagged unverified.
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...` clean in atlas-mini-games, atlas-channel, and libs/atlas-packet; `docker buildx bake atlas-mini-games atlas-channel` clean; `tools/redis-key-guard.sh` clean.
- [ ] atlas-mini-games present in `.github/config/services.json` AND `docker-bake.hcl` `go_services`; k8s manifest readiness probe is `/api/readyz`.
- [ ] Code review (plan-adherence + backend-guidelines) run before PR.

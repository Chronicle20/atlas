# Mini-Game Domain

## game

### Responsibility

Manages a mini-game room's lifecycle (create, visit, leave, chat, expel) and its in-game commands (ready, start, move stone, flip card, tie, retreat, give up, skip, exit-after-game) for the two supported room types, Omok and Match Cards. Room state is held in a process-wide, tenant-partitioned in-memory registry; it is never persisted, so it is rebuilt (empty) on restart.

### Core Models

#### Room

The immutable state of one mini-game room. Constructed via `Builder` (`NewBuilder`, `Clone`); `Registry.Update` swaps an old `Room` for a new one built from `Clone`.

| Field | Type | Description |
|-------|------|-------------|
| roomType | byte | 1 for Omok, 2 for Match Cards |
| ownerId | uint32 | Owner character id; also the room's identity (`Id()`) |
| field | field.Model | World, channel, map, and instance the room is registered in |
| title | string | Room title |
| private | bool | Whether the room requires a password to visit |
| password | string | Room password |
| pieceType | byte | Omok piece set, or Match Cards difficulty tier |
| visitorId | uint32 | Seated visitor's character id, or 0 when empty |
| visitorReady | bool | Whether the visitor has readied |
| inProgress | bool | Whether a game is currently running |
| deniedTie | [2]bool | Per-slot (0 owner, 1 visitor) flag: this slot has declined the current tie proposal |
| exitAfter | [2]bool | Per-slot flag: this slot has requested to leave once the current game ends |
| firstMover | byte | Slot (0 owner, 1 visitor) granted the first move of the next game |
| currentTurn | byte | Slot whose move is currently accepted; unset (0) until START |
| board | [omok.Cells]byte | Omok board (15x15 cells) |
| moves | []Move | Omok move history, in play order |
| deck | []uint32 | Match Cards deck (card ids, each appearing twice) |
| firstSlot | int16 | Deck index of the pending Match Cards first flip; -1 means none pending |
| ownerPairs | byte | Owner's matched pair count (Match Cards) |
| visitorPairs | byte | Visitor's matched pair count (Match Cards) |
| ownerScore | int32 | Owner's per-room session score; never persisted |
| visitorScore | int32 | Visitor's per-room session score; never persisted |
| ownerForfeits | byte | Owner's forfeit count this session |
| visitorForfeits | byte | Visitor's forfeit count this session |
| lastVisitorId | uint32 | Most recent non-zero `visitorId`, retained after a visitor leaves |
| tieCooldownUntil | time.Time | Point in time before which a tie result is not eligible for the tie-score bonus |
| skipCooldownUntil | time.Time | Point in time before which a further SKIP is ignored |
| gameType | record.GameType | Persisted-record game type this room maps to |

`Id()` returns `ownerId`. `SlotOf(characterId)` returns the occupied slot (0 owner, 1 visitor) or `(0, false)` if the character is not a member. `OpponentOf(characterId)` returns the character in the opposite slot, or 0. `Moves()` and `Deck()` return defensive copies of their backing slices.

#### Move

One Omok stone placement, in play order.

| Field | Type | Description |
|-------|------|-------------|
| X | uint32 | Column |
| Y | uint32 | Row |
| Stone | byte | Placed stone's 1-based color |

### Invariants

- `Room.Id()` always equals `Room.OwnerId()`: a character can own at most one room at a time.
- `Registry.Create` fails with `ErrOwnerHasRoom` when the owner is already a member (owner or visitor) of another room for the tenant.
- `Registry.Update` fails with `ErrRoomNotFound` when the room id does not exist for the tenant.
- Only the visitor may set `VisitorReady` (`Ready`/`Unready`); doing so while `InProgress` is a no-op.
- Only the owner may `Start`, and only when a visitor is present and ready, and no game is already running.
- Only the owner may `Expel`, and only when a visitor is present.
- A member's command against a room they do not occupy, or against the wrong turn slot, is silently dropped (no event).
- Stone color is derived server-side from the slot and `FirstMover` (`stoneColor`); it is never trusted from the client.
- The Omok double-three (renju) rule applies only to the first mover (color 1); a winning five is never forbidden.
- A tie proposal is dropped once the requester's slot has already been denied a tie this game (`DeniedTie`).
- A tie result is eligible for the tie-score bonus only outside the 5-minute `tieCooldownUntil` window.
- A winner's forfeit-win bonus is suppressed once the forfeiting loser already has 4 or more forfeits this session (anti-farm).
- A second SKIP within 3 seconds (`skipCooldownUntil`) of the first is dropped, to absorb the duplicate mode-63 time-over packet both clients send on a turn timeout.
- `endGame` is idempotent: a room that is not `InProgress` is a no-op.
- On `endGame`, the persisted-record write (`record.ApplyResult`) and the refreshed-record reads run before the in-memory registry swap; a failure at any of those steps leaves the room untouched and still `InProgress`.
- Session scores (`ownerScore`/`visitorScore`), `FirstMover`, `VisitorId`, and the forfeit counters persist across a game's end; the board, deck, move history, pair counts, deny-tie bits, exit-after bits, `InProgress`, and `CurrentTurn` reset for a rematch.
- Room creation requires the character to be alive (HP > 0), the field to not carry the no-mini-room field limit, the character to have no open chalkboard, and the character to possess the room-type's creation item (checked, never consumed).
- Room visiting additionally requires the room to exist, have no visitor, and (if private with a non-empty password) a matching password.
- Leaving or being expelled from a room mid-game forfeits the game to the opponent before the membership teardown runs.
- The owner leaving closes the room (removes it from the registry); the visitor leaving only frees the visitor slot.

### State Transitions

#### Room Lifecycle

1. **Create**: Validates alive, field limit, no open chalkboard, and item possession, then that the owner is not already a member of any room. Registers the room and emits `CREATED` + `BALLOON_UPDATED`.
2. **Visit**: Validates the visitor is not already a member of any room, then that the room exists, has no visitor, and (if applicable) the password matches, then alive and no open chalkboard. Seats the visitor and emits `ENTERED` + `BALLOON_UPDATED`. Session scores reset to 0 when a different visitor than `lastVisitorId` joins.
3. **Leave**: A non-member is a no-op. If in progress, resolves the leaver's forfeit first. The owner leaving removes the room and emits `ROOM_CLOSED` + `BALLOON_UPDATED` (remove); the visitor leaving frees the slot and emits `LEFT` + `BALLOON_UPDATED`.
4. **Expel**: Only the owner may expel the visitor. If in progress, resolves the visitor's forfeit first (owner wins). Frees the visitor slot and emits `LEFT` + `BALLOON_UPDATED`.
5. **Chat**: A member's message is rebroadcast to the room via `CHAT`; a non-member's is dropped.
6. **Teardown**: Map change, logout, channel change, and session destroy all release whatever room the character occupies on the same leave path as an explicit `Leave` (including mid-game forfeit resolution).

#### Game Lifecycle

1. **Ready / Unready**: The visitor toggles their ready flag (dropped mid-game or if sent by the owner). Emits `READY`/`UNREADY`.
2. **Start**: The owner starts once a visitor is present and ready and no game is running. Initializes the board (Omok) or a shuffled deck (Match Cards), sets `CurrentTurn` to the mover opposite `FirstMover`, and clears the exit-after and deny-tie bits. Emits `STARTED` + `BALLOON_UPDATED`.
3. **Move Stone (Omok)**: Dropped unless the game is running, the room is Omok, and the sender holds the current turn. A rejected placement (occupied/out of bounds, or a double-three by the first mover) emits `PUT_STONE_ERROR` to the acting character. A valid non-winning move flips the turn and emits `STONE_PLACED`; a winning move emits `STONE_PLACED` then resolves via end-game.
4. **Flip Card (Match Cards)**: Dropped unless the game is running, the room is Match Cards, the sender holds the current turn, and the card index is in range. A first flip records the pending index and emits `CARD_FLIPPED` (`secondFlip=false`). A second flip compares the two cards: a match increments that player's pair count and retains the turn; a mismatch passes the turn. When every pair is matched, the game ends (more pairs wins; equal pairs tie).
5. **Request / Answer Tie**: `RequestTie` forwards a tie proposal to the opponent unless the requester's slot has already been denied. `AnswerTie` accept ends the game as a tie; decline sets the requester's deny bit and emits `TIE_ANSWERED`.
6. **Give Up**: Forfeits the running game to the opponent.
7. **Request / Answer Retreat (Omok)**: `RequestRetreat` forwards an undo request to the opponent, only when the requester's own stone is the most recent move. `AnswerRetreat` accept pops the most recent stone from the board and move history and returns the turn to the requester; decline emits `RETREAT_ANSWERED` (`accept=false`).
8. **Skip**: Yields the current player's turn to the opponent; dropped unless the sender holds the current turn and the 3-second skip-debounce window has passed. Emits `SKIPPED`.
9. **Exit After Game / Cancel Exit After Game**: Sets or clears the member's exit-after-game flag; honored at the next game end. Emits no event.
10. **End Game** (`endGame`, internal): Shared resolution for a win (move/flip), a tie (accepted `ANSWER_TIE`), or a forfeit (`GIVE_UP` or a mid-game leave/expel/teardown). Upserts the win/tie/loss record for both participants, computes session score and forfeit-counter deltas, resets the room for a rematch, and emits `GAME_ENDED` + `BALLOON_UPDATED`. Then honors either side's exit-after-game flag by running that side's leave.

### Processors

#### Processor

Handles every mini-game lifecycle and gameplay command, plus the two read paths backing this service's REST endpoints (`RoomsInField`, `RoomForCharacter`). Constructed via `NewProcessor(l, ctx, db)`, which wires the process-wide `Registry` and the `character`, `map`, `inventory`, and `chalkboard` REST-client seams.

Every command runs through `emit`, which opens one database transaction, enqueues every buffered status event into the transactional outbox inside that transaction, and (for `endGame`) upserts the game record in the same transaction — so a command's persisted state and the events describing it commit and publish atomically.

#### Registry

The tenant-partitioned, in-memory, RWMutex-guarded store of `Room` values (`GetRegistry()` singleton). Indexes rooms by id and by member (owner or visitor) character id, so membership lookups and the double-room check are O(1). `Create`, `Update`, and `Remove` are the only mutators; a room is never mutated in place, only swapped.

#### Builder

Copy-on-write constructor for `Room`. `NewBuilder` starts a new room (`FirstMover=1`, `FirstSlot=-1`); `Clone` seeds a builder from an existing `Room` without mutating the source.

#### Omok engine (`game/omok`)

`Place` attempts a stone placement on the 15x15 board, rejecting an occupied/out-of-bounds cell or (for the first mover only) a double-three. `Wins` checks for five or more consecutive same-color stones from a given cell in any of the four line directions.

#### Match Cards engine (`game/matchcards`)

`MatchesToWin` maps a piece-set `pieceType` (0/1/2) to the pairs required to win (6/10/15). `BuildDeck` builds an unshuffled deck of the given pair count. `Shuffle` randomizes a deck in place (Fisher-Yates) using an injected `*rand.Rand`. `FlipResultType` maps a flip outcome to its wire result code.

#### Data-client seams

Small REST-client interfaces the `Processor` validation ladder reads through (`characterProvider`, `mapProvider`, `inventoryProvider`, `chalkboardProvider`), backed by the `data/character`, `data/map`, `data/inventory`, and `data/chalkboard` packages:

- `character`: `Hp(characterId)` — the alive check.
- `map`: `FieldLimit(mapId)` — the "cannot start game here" field-limit check.
- `inventory`: `HasItem(characterId, itemId)` — the room-creation item-possession check (item is never consumed).
- `chalkboard`: `HasOpen(characterId)` — whether the character has an open chalkboard (blocks opening or visiting a room). A 404 from the chalkboard service means "no open chalkboard"; any other error is propagated rather than treated as "no chalkboard."

---

## record

### Responsibility

Tracks each character's persistent win/tie/loss record for each mini-game type. A pure REST-and-persistence domain with no Kafka consumers; it is called directly by the `game` domain when a game ends.

### Core Models

#### Model

The immutable win/tie/loss record for one character and game type.

| Field | Type | Description |
|-------|------|-------------|
| tenantId | uuid.UUID | Tenant identifier |
| id | uuid.UUID | Record identifier; `uuid.Nil` means "never played" (no persisted row) |
| characterId | uint32 | Character the record belongs to |
| gameType | GameType | Which mini-game this record tracks |
| wins | uint32 | Win count |
| ties | uint32 | Tie count |
| losses | uint32 | Loss count |

#### GameType

`string` enum identifying which mini-game a record tracks: `GameTypeOmok` (`"OMOK"`), `GameTypeMatchCards` (`"MATCH_CARDS"`). `AllGameTypes` enumerates both.

### Invariants

- A `Model` with a zero `Id` (`uuid.Nil`) represents "no rows yet" for that character/game-type pair and is returned instead of a not-found error.
- `GetByCharacter` always returns exactly one `Model` per `GameType` in `AllGameTypes`, zero-filled for any game type the character has never played.
- `Model` is constructed only through `Builder`, which requires a non-nil `tenantId`, a non-zero `characterId`, and a non-empty `gameType`.
- `ApplyResult` upserts both participants' records in a single transaction: a tie increments both sides' `Ties`; otherwise the winning slot's `Wins` and the losing slot's `Losses` are incremented.
- `ApplyResult` joins the caller's existing transaction when one is supplied, so the record write and the `game` domain's outbox-enqueued events for the same game end commit atomically.

### Processors

#### Processor

Application-layer face for the `record` domain. `GetByCharacter(characterId)` returns one `Model` per `GameType`, zero-filled for unplayed types. Constructed via `NewProcessor(l, ctx, db)`.

#### administrator

`getOrCreate` returns (creating if absent) the `game_records` row for a character/game-type pair. `ApplyResult(db, gameType, ownerId, visitorId, winnerSlot, tie)` performs the two-row win/tie/loss upsert for a finished game inside one transaction (`database.ExecuteTransaction`).

#### provider

`GetOrZero(ctx, db, characterId, gameType)` returns the persisted record, or a zero-filled `Model` when none exists. `GetByCharacter(ctx, db, characterId)` calls `GetOrZero` once per `AllGameTypes` entry.

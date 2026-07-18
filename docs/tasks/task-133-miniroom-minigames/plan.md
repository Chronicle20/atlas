# Miniroom Minigames (Omok + Match Cards) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Playable, server-authoritative Omok and Match Cards minirooms on all supported tenant versions, backed by a new `atlas-mini-games` service with persistent per-character win/tie/loss records.

**Architecture:** atlas-channel's `character_interaction.go` arms become thin Kafka command emitters (merchant pattern) → new `atlas-mini-games` service owns an in-memory tenant-partitioned room registry, pure game engines, and a GORM `game_records` table → status events flow back to atlas-channel, which writes the clientbound packets (new discrete-per-mode arms of the graduated `CMiniRoomBaseDlg::OnPacketBase` dispatcher family, plus the new `UPDATE_CHAR_BOX` balloon packet). See `design.md` (same folder) for decisions D1–D12 and `context.md` for the reference-file map.

**Tech Stack:** Go, Kafka (atlas-kafka), GORM/Postgres, JSON:API (api2go), libs/atlas-packet, packet-audit tooling, ida-pro-mcp.

## Global Constraints

- Worktree: ALL work happens in `.worktrees/task-133-miniroom-minigames` on branch `task-133-miniroom-minigames`. Every subagent must `cd` there first and verify the branch after each commit.
- Immutable models + Builder pattern; Processor interface/Impl (`NewProcessor(l, ctx)`); curried Kafka consumer registration; no `*_testhelpers.go` files.
- `database.ExecuteTransaction` is a confirmed no-op (`libs/atlas-database/transaction.go:9`) — atomic multi-row writes MUST use `db.Transaction(func(tx *gorm.DB) error {...})` directly.
- Never hard-code a dispatcher mode byte — every clientbound arm resolves via `atlas_packet.WithResolvedCode("operations", KEY, ...)` (banned anti-patterns AP-1..AP-8, `docs/packets/DISPATCHER_FAMILY.md`).
- The wire key `MEMORY_GAME_FIP_CARD` (typo for FLIP_CARD) is load-bearing in live configs and code — keep it verbatim everywhere.
- Every seed-template `socket.handlers` entry MUST carry `"validator"` (validator-less entries are silently dropped).
- Game data / packet layouts: verify against Cosmic source, IDA, or repo registries — never from memory. Unresolved IDA fnames are stop-and-ask, never substituted.
- New shared lib: none planned. If one is added anyway: two `COPY` lines in the repo-root Dockerfile + one `go.work` line.
- Done-gates (repeated in Task 22): `go test -race ./...`, `go vet ./...`, `go build ./...` clean in atlas-mini-games, atlas-channel, libs/atlas-packet; `docker buildx bake atlas-mini-games atlas-channel`; `tools/redis-key-guard.sh` from repo root (no GOWORK=off prefix); `packet-audit dispatcher-lint` / `matrix --check` / `operations --check` exit 0.
- Commit after every task (at minimum); use `git commit` with the standard trailer.

## File Structure (locked decomposition)

```
libs/atlas-packet/interaction/clientbound/
  interaction_minigame.go        NEW  all discrete minigame dispatcher arms
  interaction_minigame_test.go   NEW  byte fixtures + packet-audit:verify markers
  mini_room_balloon.go           NEW  UPDATE_CHAR_BOX (balloon + remove)
  mini_room_balloon_test.go      NEW
  interaction_body.go            MOD  new mode consts + body funcs
libs/atlas-packet/interaction/serverbound/
  operation_visit_game.go        NEW (only if IDA gate G4 shows a distinct game-visit layout)
services/atlas-mini-games/atlas.com/mini-games/   NEW service (module atlas-mini-games)
  main.go, logger/, rest/, kafka/{message/minigame,consumer/...,producer}/
  game/{model,builder,registry,processor,producer,resource,rest}.go
  game/omok/engine.go            pure Omok logic
  game/matchcards/engine.go      pure Match Cards logic
  record/{entity,administrator,provider,model,builder,resource,rest}.go
  data/{character,map,inventory,chalkboard}/  REST clients for validation
services/atlas-channel/atlas.com/channel/
  kafka/message/minigame/kafka.go       NEW  command/event mirror
  minigame/{processor,producer,requests,model,rest}.go  NEW  emitters + REST client
  kafka/consumer/minigame/consumer.go   NEW  status-event consumer
  socket/handler/character_interaction.go  MOD  wire arms
  kafka/consumer/map/consumer.go        MOD  spawnMiniGamesForSession
  main.go                               MOD  consumer registration
services/atlas-configurations/seed-data/templates/template_*.json  MOD  all six
tools/packet-audit/cmd/run.go          MOD  candidatesFromFName cases
deploy/k8s/base/{atlas-mini-games.yaml,kustomization.yaml,env-configmap.yaml}  MOD/NEW
deploy/k8s/overlays/{main,pr}/...      MOD  per-service env lists
.github/config/services.json, docker-bake.hcl, go.work  MOD
docs/tasks/task-133-miniroom-minigames/{ida-notes.md,rollout.md}  NEW
```

---

### Task 1: IDA verification gates G1–G5 → `ida-notes.md`

**Files:**
- Create: `docs/tasks/task-133-miniroom-minigames/ida-notes.md`

**Interfaces:**
- Produces: `ida-notes.md` — the byte-layout source of truth consumed by Tasks 2–8 and 20. Sections: `## G1 start-byte`, `## G2 retreat`, `## G3 balloon`, `## G4 visit`, `## G5 modes+layouts`.

- [ ] **Step 1: Select the right IDA instances.** Run `mcp__ida-pro__list_instances` and match **binary names** (the set rotates; as of 2026-06-30 it was v83-dump on port 13342 and v95 on 13341). `select_instance(port)` for v83 first. If neither a v83 nor v95 IDB is loaded, STOP and ask the user to load them.
- [ ] **Step 2: G1 — start-byte turn semantics (v83).** `func_query` with `name_regex: "COmokDlg::|CMemoryGameDlg::"`, list all, then decompile the handler that consumes the START (0x3D) sub-op of `CMiniRoomBaseDlg::OnPacketBase`. Record verbatim: what the byte after the mode does (which slot gets the first move). Cosmic writes `1` initially, `0` after an owner win, `1` after a visitor win (`MiniGame.java:52`, `setPiece`). Write the resolved rule into ida-notes.md §G1 with decompile citation (address + snippet).
- [ ] **Step 3: G2 — retreat.** Decompile `COmokDlg::OnRetreatRequest` (fname already pinned in `libs/atlas-packet/interaction/serverbound/operation_memory_game_retreat_answer.go:12`) plus the clientbound arms for modes 54/55 inside `CMiniRoomBaseDlg::OnPacketBase`/`COmokDlg` vtable. Record: exact clientbound byte layout of retreat-request and retreat-answer(accept/decline), how many stones the client pops on accept, and whose turn follows. There is NO Cosmic reference — the client is the only authority.
- [ ] **Step 4: G3 — balloon.** Decompile `CUser::OnMiniRoomBalloon` on v83 AND v95 (`select_instance` swap). Record the full read order and compare against the candidate layout: `int32 characterId, byte roomType, int32 roomId, string title, bool private, byte pieceType, byte occupancy, byte capacity, byte inProgress` (Cosmic `addAnnounceBox`, `PacketCreator.java:2199-2208`; existing model `libs/atlas-packet/interaction/mini_room.go:67`). Also record the shop-branch (roomType 4/5) read order — mapping the `MiniRoom` writer activates merchant balloon sends too.
- [ ] **Step 5: G4 — serverbound VISIT for game rooms (v83).** Decompile the client's game-room join send path (start from `CMiniRoomBaseDlg::OnPacketBase` ENTER flow / the function that writes serverbound mode 4). Record whether the body is `int32 serialNumber` followed by a password string for private game rooms, vs the trade-shaped body `operation_visit.go` decodes today. Write the exact layout.
- [ ] **Step 6: G5 — clientbound mode values + gameplay body layouts.** On v83 and v95, decompile `CMiniRoomBaseDlg::OnPacketBase` and the Omok/MemoryGame dlg sub-op switch; record the mode value for each of: ASK_TIE, TIE_ANSWER, ASK_RETREAT, RETREAT_ANSWER, READY, UNREADY, START, RESULT(GET_RESULT), SKIP, MOVE_STONE, FIP_CARD (expected v83: 50/51/54/55/58/59/61/62/63/64/68 — the client uses one enum for both directions, matching the seeded serverbound table `template_gms_83_1.json:571-584`). Record read orders for START(omok/matchcards), MOVE_STONE, SELECT_CARD, RESULT (all three shapes), SKIP (meaning of the byte after the mode: Cosmic writes 0x01 for owner, 0x00 for visitor), and the room-enter (`getMiniGame`) blob.
- [ ] **Step 7: Commit.**
```bash
git add docs/tasks/task-133-miniroom-minigames/ida-notes.md
git commit -m "verify(task-133): IDA gates G1-G5 (start byte, retreat, balloon, visit, modes)"
```
Any fname that does not resolve, or a decompile that contradicts Cosmic AND the seeded tables: STOP and report to the user — do not guess.

---

### Task 2: Clientbound bodyless/simple game arms (Ready, Unready, RequestTie, AnswerTie, Skip)

**Files:**
- Create: `libs/atlas-packet/interaction/clientbound/interaction_minigame.go`
- Create: `libs/atlas-packet/interaction/clientbound/interaction_minigame_test.go`
- Modify: `libs/atlas-packet/interaction/clientbound/interaction_body.go` (append consts + body funcs)

**Interfaces:**
- Consumes: `ida-notes.md` §G5 (mode values, skip byte meaning); existing pattern `interaction.go:17-51` (struct + Encode/Decode + fname marker), `interaction_body.go:51-55` (`WithResolvedCode`).
- Produces (used by channel Task 18):
  `CharacterInteractionMiniGameReadyBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte`
  `CharacterInteractionMiniGameUnreadyBody()`, `CharacterInteractionMiniGameRequestTieBody()`, `CharacterInteractionMiniGameAnswerTieBody()` (deny forward — accept path emits RESULT instead), `CharacterInteractionMiniGameSkipBody(who byte)`.
  Mode-key consts: `CharacterInteractionModeMemoryGameAskTie/TieAnswer/AskRetreat/RetreatAnswer/Ready/Unready/Start/Result/Skip/MoveStone/FlipCard` with values `"MEMORY_GAME_ASK_TIE"`, `"MEMORY_GAME_TIE_ANSWER"`, `"MEMORY_GAME_ASK_RETREAT"`, `"MEMORY_GAME_RETREAT_ANSWER"`, `"MEMORY_GAME_READY"`, `"MEMORY_GAME_UNREADY"`, `"MEMORY_GAME_START"`, `"MEMORY_GAME_RESULT"`, `"MEMORY_GAME_SKIP"`, `"MEMORY_GAME_MOVE_STONE"`, `"MEMORY_GAME_FIP_CARD"` (same key strings as the serverbound handler table — one client enum).

- [ ] **Step 1: Write failing round-trip tests** in `interaction_minigame_test.go`, following `interaction_test.go` exactly (`test.Variants` + `test.RoundTrip`). One test per struct; literal mode bytes from ida-notes §G5 (v83 values shown):
```go
func TestInteractionMiniGameReadyRoundTrip(t *testing.T) {
	input := NewInteractionMiniGameReady(58)
	for _, v := range test.Variants {
		ctx := test.CreateTestContext(v.Region, v.MajorVersion, v.MinorVersion)()
		test.RoundTrip(t, ctx, input.Encode, (&InteractionMiniGameReady{}).Decode, nil)
	}
}
```
(same shape for Unready(59), RequestTie(50), AnswerTie(51), Skip(63, 0x01) and Skip(63, 0x00)).
- [ ] **Step 2: Run to verify failure.** `cd libs/atlas-packet && go test ./interaction/clientbound/ -run MiniGame -v` → FAIL (undefined types).
- [ ] **Step 3: Implement the structs** in `interaction_minigame.go`. Each is discrete (AP-1: no shared-by-shape struct), private fields, constructor, `Operation() string { return CharacterInteractionWriter }`, `String()`, Encode/Decode. Representative pair — replicate for each:
```go
// InteractionMiniGameReady - visitor readied up
// packet-audit:fname CMiniRoomBaseDlg::OnPacketBase#MemoryGameReady
type InteractionMiniGameReady struct{ mode byte }

func NewInteractionMiniGameReady(mode byte) InteractionMiniGameReady {
	return InteractionMiniGameReady{mode: mode}
}
func (m InteractionMiniGameReady) Operation() string { return CharacterInteractionWriter }
func (m InteractionMiniGameReady) String() string    { return "minigame ready" }
func (m InteractionMiniGameReady) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}
func (m *InteractionMiniGameReady) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) { m.mode = r.ReadByte() }
}

// InteractionMiniGameSkip - a player passed the turn
// packet-audit:fname CMiniRoomBaseDlg::OnPacketBase#MemoryGameSkip
type InteractionMiniGameSkip struct {
	mode byte
	who  byte // 0x01 owner skipped, 0x00 visitor (Cosmic getMiniGameSkipOwner/Visitor; confirm meaning in ida-notes §G5)
}

func NewInteractionMiniGameSkip(mode byte, who byte) InteractionMiniGameSkip {
	return InteractionMiniGameSkip{mode: mode, who: who}
}
func (m InteractionMiniGameSkip) Operation() string { return CharacterInteractionWriter }
func (m InteractionMiniGameSkip) String() string    { return fmt.Sprintf("minigame skip who [%d]", m.who) }
func (m InteractionMiniGameSkip) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.who)
		return w.Bytes()
	}
}
func (m *InteractionMiniGameSkip) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.who = r.ReadByte()
	}
}
```
`InteractionMiniGameUnready`, `InteractionMiniGameRequestTie`, `InteractionMiniGameAnswerTie` are mode-only clones of `Ready` with fname markers `#MemoryGameUnready`, `#MemoryGameRequestTie`, `#MemoryGameAnswerTie`.
- [ ] **Step 4: Append mode consts + body funcs** to `interaction_body.go` (after line 48, inside the const block and at file end):
```go
CharacterInteractionModeMemoryGameAskTie        CharacterInteractionMode = "MEMORY_GAME_ASK_TIE"        // 50
CharacterInteractionModeMemoryGameTieAnswer     CharacterInteractionMode = "MEMORY_GAME_TIE_ANSWER"     // 51
CharacterInteractionModeMemoryGameAskRetreat    CharacterInteractionMode = "MEMORY_GAME_ASK_RETREAT"    // 54
CharacterInteractionModeMemoryGameRetreatAnswer CharacterInteractionMode = "MEMORY_GAME_RETREAT_ANSWER" // 55
CharacterInteractionModeMemoryGameReady         CharacterInteractionMode = "MEMORY_GAME_READY"          // 58
CharacterInteractionModeMemoryGameUnready       CharacterInteractionMode = "MEMORY_GAME_UNREADY"        // 59
CharacterInteractionModeMemoryGameStart         CharacterInteractionMode = "MEMORY_GAME_START"          // 61
CharacterInteractionModeMemoryGameResult        CharacterInteractionMode = "MEMORY_GAME_RESULT"         // 62
CharacterInteractionModeMemoryGameSkip          CharacterInteractionMode = "MEMORY_GAME_SKIP"           // 63
CharacterInteractionModeMemoryGameMoveStone     CharacterInteractionMode = "MEMORY_GAME_MOVE_STONE"     // 64
CharacterInteractionModeMemoryGameFlipCard      CharacterInteractionMode = "MEMORY_GAME_FIP_CARD"       // 68 (typo is load-bearing)
```
```go
func CharacterInteractionMiniGameReadyBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CharacterInteractionModeMemoryGameReady, func(mode byte) packet.Encoder {
		return NewInteractionMiniGameReady(mode)
	})
}
```
(identical shape for `UnreadyBody`, `RequestTieBody`, `AnswerTieBody`, and `SkipBody(who byte)` closing over `who`).
- [ ] **Step 5: Run tests.** `go test ./interaction/clientbound/ -run MiniGame -v` → PASS. Then `go vet ./...` in `libs/atlas-packet`.
- [ ] **Step 6: Commit.**
```bash
git add libs/atlas-packet/interaction/clientbound/
git commit -m "feat(task-133): clientbound minigame ready/unready/tie/skip dispatcher arms"
```

---

### Task 3: Clientbound Start (Omok + MatchCards), MoveStone, CardSelect arms

**Files:**
- Modify: `libs/atlas-packet/interaction/clientbound/interaction_minigame.go` (+ test file)
- Modify: `libs/atlas-packet/interaction/clientbound/interaction_body.go`

**Interfaces:**
- Consumes: mode consts from Task 2; ida-notes §G5 layouts.
- Produces: `CharacterInteractionMiniGameStartOmokBody(firstMover byte)`, `CharacterInteractionMiniGameStartMatchCardsBody(firstMover byte, deck []uint32)`, `CharacterInteractionMiniGameMoveStoneBody(x uint32, y uint32, stoneType byte)`, `CharacterInteractionMiniGameCardSelectFirstBody(slot byte)`, `CharacterInteractionMiniGameCardSelectSecondBody(slot byte, firstSlot byte, resultType byte)` (all returning the standard body-func signature).

- [ ] **Step 1: Failing round-trip tests** (same fixture pattern; layouts from Cosmic verified against ida-notes §G5):
```go
func TestInteractionMiniGameStartMatchCardsRoundTrip(t *testing.T) {
	input := NewInteractionMiniGameStartMatchCards(61, 1, []uint32{0, 0, 1, 1, 2, 2, 3, 3, 4, 4, 5, 5})
	...
}
```
plus StartOmok(61, 1), MoveStone(64, 7, 8, 1), CardSelectFirst(68, 3), CardSelectSecond(68, 9, 3, 2).
- [ ] **Step 2: Run → FAIL.**
- [ ] **Step 3: Implement.** Layouts (each its own struct + `#`-fname marker):
  - `InteractionMiniGameStartOmok` (`#MemoryGameStartOmok`): mode, byte firstMover.
  - `InteractionMiniGameStartMatchCards` (`#MemoryGameStartMatchCards`): mode, byte firstMover, byte len(deck), then `len` × int32 cardId (Cosmic `getMatchCardStart`, `PacketCreator.java:4892-4911`).
  - `InteractionMiniGameMoveStone` (`#MemoryGameMoveStone`): mode, int32 x, int32 y, byte stoneType (`getMiniGameMoveOmok`, `:4755-4762`).
  - `InteractionMiniGameCardSelectFirst` (`#MemoryGameCardSelectFirst`): mode, byte 1, byte slot.
  - `InteractionMiniGameCardSelectSecond` (`#MemoryGameCardSelectSecond`): mode, byte 0, byte slot, byte firstSlot, byte resultType (0 owner-mismatch / 1 visitor-mismatch / 2 owner-match / 3 visitor-match; `getMatchCardSelect`, `:4927-4939`).
  Body funcs: StartOmok/StartMatchCards both resolve key `CharacterInteractionModeMemoryGameStart` (one key, two discrete arms — both fixtured); CardSelect First/Second both resolve `...ModeMemoryGameFlipCard`.
- [ ] **Step 4: Run → PASS; vet clean.**
- [ ] **Step 5: Commit** (`feat(task-133): clientbound minigame start/move/select arms`).

---

### Task 4: Clientbound Result arm (GET_RESULT)

**Files:**
- Modify: `libs/atlas-packet/interaction/clientbound/interaction_minigame.go` (+ test), `interaction_body.go`

**Interfaces:**
- Consumes: `interaction.GameRecord{Unknown, Wins, Ties, Losses, Points uint32}` (`libs/atlas-packet/interaction/visitor.go:20`) — reuse for the record refresh blocks.
- Produces: `CharacterInteractionMiniGameResultBody(resultType byte, visitorWon bool, ownerRecord interaction.GameRecord, visitorRecord interaction.GameRecord)` — resultType 0 = normal win, 1 = tie, 2 = forfeit win.

- [x] **Step 1: Failing tests** — three cases: owner normal win, visitor forfeit win, tie.
- [x] **Step 2: Run → FAIL.**
- [x] **Step 3: Implement** `InteractionMiniGameResult` (`#MemoryGameResult`).

**CORRECTED during implementation:** the draft Cosmic-derived table below (kept
for history) is **contradicted by ida-notes.md §G5 RESULT**
(`COmokDlg::OnGameResult` v83 @ 0x6e4463 / `CMemoryGameDlg::OnGameResult` v83 @
0x64e423) — no `bool` written on the tie shape, no `int32`/`int16` padding
blocks, and no trailing tie byte. Per the brief's override rule, the IDA read
order wins. The actual implemented layout is:
```
byte mode                      // 62 @ v83
byte resultType                // 0 win, 1 tie, 2 forfeit
if resultType != 1: byte winnerSlot   // 0 = owner won, 1 = visitor won; OMITTED for tie (resultType == 1)
<20-byte ownerRecord>           // 5 x int32: Unknown, Wins, Ties, Losses, Points
<20-byte visitorRecord>         // 5 x int32: Unknown, Wins, Ties, Losses, Points
```
`visitorWon bool` in `CharacterInteractionMiniGameResultBody`'s signature maps
1:1 onto `winnerSlot` (false→0, true→1) and is simply not serialized when
`resultType == 1`.

Superseded draft table (originally cross-checked against Cosmic
`getMiniGameResult`, `PacketCreator.java:4785-4830` — do not implement this,
see correction above):
```
byte mode                      // 62 @ v83
byte resultType                // 0 win, 1 tie, 2 forfeit
bool visitorWon                // false = owner won; written on ALL result types including tie
if resultType == 1: byte 0, int16 0
else:               int32 0
int32 ownerRecord marker(Unknown), Wins, Ties, Losses, Points   // 5 x int32 via GameRecord fields
int32 0
int32 visitorRecord marker, Wins, Ties, Losses, Points
if resultType == 1: byte 0
```
- [x] **Step 4: Run → PASS; vet clean.**
- [x] **Step 5: Commit** (`feat(task-133): clientbound minigame result arm`).

---

### Task 5: Clientbound Retreat arms (IDA-derived)

**Files:**
- Modify: `libs/atlas-packet/interaction/clientbound/interaction_minigame.go` (+ test), `interaction_body.go`

**Interfaces:**
- Consumes: ida-notes §G2 (the ONLY layout authority — Cosmic has no retreat).
- Produces: `CharacterInteractionMiniGameRetreatRequestBody()` and `CharacterInteractionMiniGameRetreatAnswerBody(...)` — final parameter list per §G2 (at minimum the accept/deny discriminator; plus stone-pop count/turn field if the client reads one).

- [ ] **Step 1: Read `ida-notes.md` §G2.** If Task 1 marked G2 unresolved, STOP — report BLOCKED (this is the one genuinely client-only feature).
- [ ] **Step 2: Failing round-trip tests** for `InteractionMiniGameRetreatRequest` (`#MemoryGameRetreatRequest`, mode 54 @ v83) and `InteractionMiniGameRetreatAnswer` (`#MemoryGameRetreatAnswer`, mode 55 @ v83) with the §G2 field list.
- [ ] **Step 3: Implement structs + body funcs** (keys `...ModeMemoryGameAskRetreat` / `...ModeMemoryGameRetreatAnswer`), exactly the §G2 read order.
- [ ] **Step 4: Run → PASS; vet clean.**
- [ ] **Step 5: Commit** (`feat(task-133): clientbound minigame retreat arms (IDA-derived)`).

---

### Task 6: UPDATE_CHAR_BOX balloon packet

**Files:**
- Create: `libs/atlas-packet/interaction/clientbound/mini_room_balloon.go`, `mini_room_balloon_test.go`

**Interfaces:**
- Consumes: ida-notes §G3; `interaction.MiniRoomWriter = "MiniRoom"` const (`libs/atlas-packet/interaction/mini_room.go:14`).
- Produces: `MiniRoomBalloonBody(characterId uint32, roomType byte, roomId uint32, title string, hasPassword bool, pieceType byte, occupancy byte, capacity byte, inProgress bool)` and `MiniRoomBalloonRemoveBody(characterId uint32)` — plain body funcs (no mode byte; this is NOT a dispatcher family), writer name `interaction.MiniRoomWriter`.

- [ ] **Step 1: Failing round-trip tests** for both structs.
- [ ] **Step 2: Run → FAIL.**
- [ ] **Step 3: Implement.**
```go
// MiniRoomBalloon - game/shop balloon over the owner's head
// packet-audit:fname CUser::OnMiniRoomBalloon
type MiniRoomBalloon struct {
	characterId uint32
	roomType    byte
	roomId      uint32
	title       string
	hasPassword bool
	pieceType   byte
	occupancy   byte
	capacity    byte
	inProgress  bool
}
```
Encode order per ida-notes §G3 (candidate: characterId, roomType, roomId, title, hasPassword, pieceType, occupancy, capacity, inProgress — Cosmic `addAnnounceBox`). `MiniRoomBalloonRemove`: int32 characterId, byte 0. `Operation()` returns `interaction.MiniRoomWriter` for both. Note in a comment that the legacy `MiniRoomBase.Spawn` (`mini_room.go:67`) encodes the same wire shape for shop types; the game path uses these audited structs.
- [ ] **Step 4: Run → PASS; vet clean.**
- [ ] **Step 5: Commit** (`feat(task-133): UPDATE_CHAR_BOX mini-room balloon packet`).

---

### Task 7: packet-audit wiring (candidates, exports, evidence, matrix, dispatcher-lint)

**Files:**
- Modify: `tools/packet-audit/cmd/run.go` (candidatesFromFName, near the existing `CMiniRoomBaseDlg::OnPacketBase#...` cases at `run.go:1895-1917`)
- Modify: `libs/atlas-packet/interaction/clientbound/interaction_minigame_test.go`, `mini_room_balloon_test.go` (add `// packet-audit:verify` markers)
- Modify: audit dirs under `docs/packets/audits/` (v83 + v95), `docs/packets/audits/STATUS.md` (regenerated)

**Interfaces:**
- Consumes: all Task 2–6 structs; the verify flow in `docs/packets/audits/VERIFYING_A_PACKET.md`; existing marker format `interaction_test.go:27-76` (`// packet-audit:verify packet=interaction/clientbound/Interaction<Struct> version=<ver> ida=<addr>`).
- Produces: matrix cells for the new arms (✅ on gms_v83 + gms_v95; other versions wired-unverified), lint-clean family.

- [ ] **Step 1: Add `candidatesFromFName` cases** — one per new arm: `CMiniRoomBaseDlg::OnPacketBase#MemoryGameReady|MemoryGameUnready|MemoryGameRequestTie|MemoryGameAnswerTie|MemoryGameSkip|MemoryGameStartOmok|MemoryGameStartMatchCards|MemoryGameMoveStone|MemoryGameCardSelectFirst|MemoryGameCardSelectSecond|MemoryGameResult|MemoryGameRetreatRequest|MemoryGameRetreatAnswer` plus `CUser::OnMiniRoomBalloon` → the two balloon structs. Mirror the existing 8 cases exactly (INV-3: no dangling candidates).
- [ ] **Step 2: Splice synthetic `#`-suffixed export entries** for v83 and v95 into the audit-dir exports (surgical splice — the export is NON-idempotent, never regenerate/overwrite; see `docs/packets/audits/VERIFYING_A_PACKET.md` §9–10) and write the per-arm audit reports with the ida-notes addresses.
- [ ] **Step 3: Add verify markers** above each fixture test (v83 + v95 per arm; balloon on both), pin evidence records.
- [ ] **Step 4: Run the checkers** from the tool's documented invocation (`--output docs/packets/audits`):
```
packet-audit dispatcher-lint   → exit 0
packet-audit matrix --check    → exit 0 (regen STATUS.md first if needed)
packet-audit operations --check → exit 0
```
Expected: `PLAYER_INTERACTION | CMiniRoomBaseDlg::OnPacketBase` stays ✅ with the new arms listed; `UPDATE_CHAR_BOX` row flips ✅ for v83/v95, remains ❌-with-banner elsewhere.
- [ ] **Step 5: Commit** (`verify(task-133): packet-audit wiring for minigame arms + balloon`).

---

### Task 8: Serverbound game-VISIT decoder (conditional on G4)

**Files:**
- Create (only if needed): `libs/atlas-packet/interaction/serverbound/operation_visit_game.go` (+ test)
- Modify (else): none — record the finding.

**Interfaces:**
- Consumes: ida-notes §G4.
- Produces: `OperationVisitGame` with `SerialNumber() uint32` and `Password() string` (channel Task 18 uses these; if G4 shows `operation_visit.go` already fits, Task 18 uses `OperationVisit.SerialNumber()` and an empty password).

- [ ] **Step 1: Read ida-notes §G4.** If the existing `OperationVisit` layout matches the game-room join, add a comment to `operation_visit.go` citing the G4 evidence, commit (`docs(task-133): G4 visit layout confirmation`), and SKIP the rest.
- [ ] **Step 2: Failing round-trip test** for `OperationVisitGame` with the §G4 field list (candidate: `serialNumber uint32`, then when the room is private a password string — exact gate/order per §G4).
- [ ] **Step 3: Implement** as a thin codec (fname marker = the §G4 send-path fname), following `operation_memory_game_move_stone.go` structure.
- [ ] **Step 4: Run → PASS; vet clean. Commit** (`feat(task-133): serverbound game-visit decoder`).

---

### Task 9: atlas-mini-games service scaffold

**Files:**
- Create: `services/atlas-mini-games/atlas.com/mini-games/{go.mod,main.go}`, `logger/init.go`, `rest/handler.go`, `kafka/consumer/consumer.go`, `kafka/producer/producer.go`
- Modify: `go.work` (add `./services/atlas-mini-games/atlas.com/mini-games`), `.github/config/services.json`, `docker-bake.hcl` (`go_services` list — BOTH places, hand-synced)

**Interfaces:**
- Produces: running skeleton other tasks fill in. `main.go` wires: `database.Connect(l, database.SetMigrations(record.Migration))`, consumer manager, REST server with `SetBasePath("/api/")` + `server.MountReadiness("/readyz", ...)` (atlas-world precedent — effective path `/api/readyz`), `AddRouteInitializer(record.InitResource(GetServer()))` + `AddRouteInitializer(game.InitResource(GetServer()))`.

- [ ] **Step 1: Copy the skeleton shape** from `services/atlas-chalkboards/atlas.com/chalkboards/` (`main.go`, `logger/`, `rest/handler.go`, `kafka/consumer/consumer.go`, `kafka/producer/producer.go` — adjust package/module name to `atlas-mini-games`, service name const `atlas-mini-games`, consumer group `consumergroup.Resolve("Mini Game Service")`). NO Redis: drop `atlas.Connect`/`InitRegistry` lines; ADD the buddies-style DB connect (`services/atlas-buddies/atlas.com/buddies/main.go:57`). Leave consumer/route initializers commented-in as later tasks land them (compile-clean at each commit — register only what exists; empty is fine now).
- [ ] **Step 2: go.mod** — module `atlas-mini-games`, same Go version + dep set as atlas-buddies' go.mod (copy, trim unused after code lands). Add the go.work entry.
- [ ] **Step 3: services.json + docker-bake.hcl.**
```json
{ "name": "atlas-mini-games", "type": "go-service", "path": "services/atlas-mini-games",
  "module_path": "services/atlas-mini-games/atlas.com/mini-games",
  "docker_image": "ghcr.io/chronicle20/atlas-mini-games/atlas-mini-games", "docker_context": "." }
```
and add `"atlas-mini-games"` to `go_services` in `docker-bake.hcl` (~line 35 list).
- [ ] **Step 4: Verify.** `cd services/atlas-mini-games/atlas.com/mini-games && go build ./... && go vet ./...` then from repo root `docker buildx bake atlas-mini-games` → all clean.
- [ ] **Step 5: Commit** (`feat(task-133): atlas-mini-games service scaffold`).

---

### Task 10: record domain (game_records) + REST

**Files:**
- Create: `services/atlas-mini-games/atlas.com/mini-games/record/{entity,administrator,provider,model,builder,resource,rest}.go` + `administrator_test.go`, `provider_test.go`

**Interfaces:**
- Produces:
  `record.GameType` (`string`; consts `record.GameTypeOmok = "OMOK"`, `record.GameTypeMatchCards = "MATCH_CARDS"`)
  `record.Migration(db *gorm.DB) error`
  `record.GetOrZero(db *gorm.DB, tenantId uuid.UUID, characterId uint32, gameType GameType) (Model, error)` — absent row → zeroed Model, nil error
  `record.GetByCharacter(db *gorm.DB, tenantId uuid.UUID, characterId uint32) ([]Model, error)` — zero-filled for both game types
  `record.ApplyResult(db *gorm.DB, tenantId uuid.UUID, gameType GameType, ownerId uint32, visitorId uint32, winnerSlot byte, tie bool) error` — winnerSlot 0 = owner, 1 = visitor; upserts BOTH rows inside one `db.Transaction`
  `record.Model` getters: `CharacterId() uint32`, `GameType() GameType`, `Wins() uint32`, `Ties() uint32`, `Losses() uint32`
  `record.InitResource(si jsonapi.ServerInformation) server.RouteInitializer` — `GET /characters/{characterId}/game-records`

- [ ] **Step 1: Failing tests** (sqlite-in-memory or the project's standard gorm test setup — mirror `services/atlas-buddies/atlas.com/buddies/list/administrator_test.go` DB bootstrap): `TestApplyResult_OwnerWin` (owner wins+1, visitor losses+1), `TestApplyResult_Tie` (both ties+1), `TestApplyResult_CreatesMissingRows`, `TestApplyResult_Atomic` (inject an error on the second write via a closed tx / constraint and assert the first row did not persist), `TestGetOrZero_Absent`.
- [ ] **Step 2: Run → FAIL.**
- [ ] **Step 3: Implement.** Entity:
```go
type Entity struct {
	TenantId    uuid.UUID `gorm:"not null;uniqueIndex:idx_record_tenant_char_game"`
	Id          uuid.UUID `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	CharacterId uint32    `gorm:"not null;uniqueIndex:idx_record_tenant_char_game"`
	GameType    string    `gorm:"not null;uniqueIndex:idx_record_tenant_char_game"`
	Wins        uint32    `gorm:"not null;default:0"`
	Ties        uint32    `gorm:"not null;default:0"`
	Losses      uint32    `gorm:"not null;default:0"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
func (e Entity) TableName() string { return "game_records" }
func Migration(db *gorm.DB) error  { return db.AutoMigrate(&Entity{}) }
```
`ApplyResult` core:
```go
func ApplyResult(db *gorm.DB, tenantId uuid.UUID, gameType GameType, ownerId uint32, visitorId uint32, winnerSlot byte, tie bool) error {
	return db.Transaction(func(tx *gorm.DB) error {
		or, err := getOrCreate(tx, tenantId, ownerId, gameType)
		if err != nil { return err }
		vr, err := getOrCreate(tx, tenantId, visitorId, gameType)
		if err != nil { return err }
		if tie {
			or.Ties++; vr.Ties++
		} else if winnerSlot == 0 {
			or.Wins++; vr.Losses++
		} else {
			or.Losses++; vr.Wins++
		}
		if err := tx.Save(&or).Error; err != nil { return err }
		return tx.Save(&vr).Error
	})
}
```
REST: chalkboards `resource.go` shape; RestModel `{Id (characterId-gameType), CharacterId, GameType, Wins, Ties, Losses}`, `GetName() = "game-records"`, list handler returns `GetByCharacter` (always two resources, zero-filled).
- [ ] **Step 4: Run → PASS; `go test -race ./record/ -v` clean.**
- [ ] **Step 5: Commit** (`feat(task-133): game_records domain + REST`).

---

### Task 11: Omok engine (pure)

**Files:**
- Create: `services/atlas-mini-games/atlas.com/mini-games/game/omok/{engine,engine_test}.go`

**Interfaces:**
- Produces:
  `omok.BoardSize = 15`, `omok.Cells = 225`
  `omok.Place(board [225]byte, x uint32, y uint32, stone byte) ([225]byte, bool)` — false on out-of-bounds/occupied/stone==0
  `omok.Wins(board [225]byte, x uint32, y uint32) bool` — 5+ in a row through (x,y), 4 directions, no overline restriction

- [ ] **Step 1: Failing tests:** horizontal/vertical/both-diagonal wins; win at each board edge/corner; run of 6 wins (overline allowed); 4-in-a-row does NOT win; occupied-cell and out-of-bounds placement rejected; broken run (gap) does not win.
- [ ] **Step 2: Run → FAIL.**
- [ ] **Step 3: Implement:**
```go
package omok

const BoardSize = 15
const Cells = BoardSize * BoardSize

// Semantics mirror Cosmic MiniGame.searchCombo/searchCombo2 (<cosmic>/src/main/java/server/maps/MiniGame.java:431-516):
// only rule is empty-cell; five or more consecutive wins; no forbidden moves.
func Place(board [Cells]byte, x uint32, y uint32, stone byte) ([Cells]byte, bool) {
	if x >= BoardSize || y >= BoardSize || stone == 0 {
		return board, false
	}
	idx := int(y)*BoardSize + int(x)
	if board[idx] != 0 {
		return board, false
	}
	board[idx] = stone
	return board, true
}

func Wins(board [Cells]byte, x uint32, y uint32) bool {
	stone := board[int(y)*BoardSize+int(x)]
	if stone == 0 {
		return false
	}
	dirs := [4][2]int{{1, 0}, {0, 1}, {1, 1}, {1, -1}}
	for _, d := range dirs {
		run := 1
		for _, sign := range [2]int{1, -1} {
			cx, cy := int(x), int(y)
			for {
				cx += d[0] * sign
				cy += d[1] * sign
				if cx < 0 || cx >= BoardSize || cy < 0 || cy >= BoardSize || board[cy*BoardSize+cx] != stone {
					break
				}
				run++
			}
		}
		if run >= 5 {
			return true
		}
	}
	return false
}
```
(last-move-centered scan is equivalent to Cosmic's whole-board scan: a new 5-run must pass through the placed stone.)
- [ ] **Step 4: Run → PASS. Commit** (`feat(task-133): omok engine`).

---

### Task 12: Match Cards engine (pure)

**Files:**
- Create: `services/atlas-mini-games/atlas.com/mini-games/game/matchcards/{engine,engine_test}.go`

**Interfaces:**
- Produces:
  `matchcards.MatchesToWin(pieceType byte) (byte, bool)` — 0→6, 1→10, 2→15, else false
  `matchcards.BuildDeck(pairs byte) []uint32` — ids 0..pairs-1 each twice, unshuffled
  `matchcards.Shuffle(deck []uint32, r *rand.Rand)` — in-place, injected rand
  `matchcards.FlipResultType(ownerFlipped bool, match bool) byte` — 2/3 match, 0/1 mismatch (owner even, visitor odd)

- [ ] **Step 1: Failing tests:** MatchesToWin mapping + invalid; deck length/content (each id exactly twice); deterministic shuffle with seeded rand; FlipResultType truth table (Cosmic `PlayerInteractionHandler.java:460-484`: owner match→2, visitor match→3, owner mismatch→0, visitor mismatch→1).
- [ ] **Step 2: Run → FAIL.**
- [ ] **Step 3: Implement** exactly the interface above (trivial bodies; FlipResultType: `if match { if ownerFlipped {return 2}; return 3 }; if ownerFlipped {return 0}; return 1`).
- [ ] **Step 4: Run → PASS. Commit** (`feat(task-133): match cards engine`).

---

### Task 13: Room model + registry

**Files:**
- Create: `services/atlas-mini-games/atlas.com/mini-games/game/{model,builder,registry}.go` + `registry_test.go`, `model_test.go`

**Interfaces:**
- Consumes: `omok.Cells`; `field.Model` (`libs/atlas-constants/field`); `tenant.Model`.
- Produces:
  `game.Move` struct `{X uint32; Y uint32; Stone byte}`
  `game.Room` immutable model. Getters (exact names): `RoomType() byte` (1 omok / 2 matchcards), `OwnerId() uint32`, `Id() uint32` (== OwnerId, D2), `Field() field.Model`, `Title() string`, `Private() bool`, `Password() string`, `PieceType() byte`, `VisitorId() uint32` (0 = empty), `VisitorReady() bool`, `InProgress() bool`, `DeniedTie(slot byte) bool`, `ExitAfter(slot byte) bool`, `FirstMover() byte`, `CurrentTurn() byte`, `Board() [225]byte`, `Moves() []Move`, `Deck() []uint32`, `FirstSlot() int16` (-1 = none), `OwnerPairs() byte`, `VisitorPairs() byte`, `OwnerScore() int32`, `VisitorScore() int32`, `OwnerForfeits() byte`, `VisitorForfeits() byte`, `LastVisitorId() uint32`, `TieCooldownUntil() time.Time`, `SlotOf(characterId uint32) (byte, bool)`, `OpponentOf(characterId uint32) uint32`, `GameType() record.GameType`
  `game.NewBuilder(roomType byte, ownerId uint32, f field.Model) *Builder` with a `Set<Field>` method for every field above, `Clone(r Room) *Builder`, `Build() Room`. Defaults: `FirstMover=1` (Cosmic `MiniGame.java:52`), `FirstSlot=-1`, `CurrentTurn` unset until START.
  `game.GetRegistry() *Registry` (sync.Once singleton, pattern of `services/atlas-channel/atlas.com/channel/account/registry.go`) with methods:
  `Create(t tenant.Model, r Room) error` (error if owner already in a room), `Get(t tenant.Model, roomId uint32) (Room, bool)`, `GetByMember(t tenant.Model, characterId uint32) (Room, bool)`, `GetInField(t tenant.Model, f field.Model) []Room`, `Update(t tenant.Model, roomId uint32, fn func(Room) (Room, error)) (Room, error)` (mutation under one write lock), `Remove(t tenant.Model, roomId uint32)`, `AddVisitor/RemoveVisitor` handled through `Update` + internal member index maintenance.

- [ ] **Step 1: Failing tests:** Create/Get round-trip; Create rejects double-room; GetByMember finds owner AND visitor; Remove clears member index for both; GetInField filters by field; `Update` swap visible to next Get; `-race` test hammering Update/Get/GetInField from 16 goroutines.
- [ ] **Step 2: Run → FAIL** (`go test -race ./game/ -v`).
- [ ] **Step 3: Implement.** Registry storage: `rooms map[tenant.Model]map[uint32]Room`, `members map[tenant.Model]map[uint32]uint32` (characterId → roomId), one `sync.RWMutex`; member index rebuilt inside `Update`/`Create`/`Remove` from the room's ownerId/visitorId. Model/builder: private fields + getters, builder holds a Room copy (channel `socket/model/mini_room.go` style, service conventions).
- [ ] **Step 4: Run → PASS (-race). Commit** (`feat(task-133): room model and registry`).

---

### Task 14: Kafka messages + lifecycle processing (create/visit/leave/chat/expel + validation)

**Files:**
- Create: `services/atlas-mini-games/atlas.com/mini-games/kafka/message/minigame/kafka.go`
- Create: `services/atlas-mini-games/atlas.com/mini-games/data/{character,map,inventory,chalkboard}/{model,requests,rest,processor}.go` (thin REST clients)
- Create: `services/atlas-mini-games/atlas.com/mini-games/game/{processor,producer}.go` + `processor_test.go`
- Create: `services/atlas-mini-games/atlas.com/mini-games/kafka/consumer/minigame/consumer.go`
- Modify: `main.go` (register consumer + handlers)

**Interfaces:**
- Consumes: registry (Task 13), record (Task 10), engines (11/12).
- Produces — message contract (mirrored verbatim by channel Task 17; envelopes clone `services/atlas-chalkboards/atlas.com/chalkboards/kafka/message/chalkboard/kafka.go`):
```go
const (
	EnvCommandTopic     = "COMMAND_TOPIC_MINI_GAME"
	CommandTypeCreate   = "CREATE"; CommandTypeVisit = "VISIT"; CommandTypeLeave = "LEAVE"
	CommandTypeChat     = "CHAT";   CommandTypeReady = "READY"; CommandTypeUnready = "UNREADY"
	CommandTypeStart    = "START";  CommandTypeMoveStone = "MOVE_STONE"; CommandTypeFlipCard = "FLIP_CARD"
	CommandTypeRequestTie = "REQUEST_TIE"; CommandTypeAnswerTie = "ANSWER_TIE"; CommandTypeGiveUp = "GIVE_UP"
	CommandTypeRequestRetreat = "REQUEST_RETREAT"; CommandTypeAnswerRetreat = "ANSWER_RETREAT"
	CommandTypeExpel = "EXPEL"; CommandTypeSkip = "SKIP"
	CommandTypeExitAfterGame = "EXIT_AFTER_GAME"; CommandTypeCancelExitAfterGame = "CANCEL_EXIT_AFTER_GAME"
)
type Command[E any] struct { // chalkboard envelope + tenant header
	TransactionId uuid.UUID; WorldId world.Id; ChannelId channel.Id; MapId _map.Id
	Instance uuid.UUID; CharacterId uint32; Type string; Body E   // json tags as in chalkboard/kafka.go
}
type CreateCommandBody struct{ RoomType byte; Title string; Private bool; Password string; PieceType byte }
type VisitCommandBody struct{ RoomId uint32; Password string }
type ChatCommandBody struct{ Message string }
type MoveStoneCommandBody struct{ X uint32; Y uint32; StoneType byte }
type FlipCardCommandBody struct{ First bool; CardIndex byte }
type AnswerCommandBody struct{ Accept bool } // ANSWER_TIE + ANSWER_RETREAT
type EmptyCommandBody struct{}
const (
	EnvEventTopicStatus = "EVENT_TOPIC_MINI_GAME_STATUS"
	EventTypeCreated = "CREATED"; EventTypeCreateError = "CREATE_ERROR"
	EventTypeEntered = "ENTERED"; EventTypeEnterError = "ENTER_ERROR"
	EventTypeLeft = "LEFT"; EventTypeRoomClosed = "ROOM_CLOSED"; EventTypeChat = "CHAT"
	EventTypeReady = "READY"; EventTypeUnready = "UNREADY"; EventTypeStarted = "STARTED"
	EventTypeStonePlaced = "STONE_PLACED"; EventTypeCardFlipped = "CARD_FLIPPED"
	EventTypeTieRequested = "TIE_REQUESTED"; EventTypeTieAnswered = "TIE_ANSWERED"
	EventTypeRetreatRequested = "RETREAT_REQUESTED"; EventTypeRetreatAnswered = "RETREAT_ANSWERED"
	EventTypeSkipped = "SKIPPED"; EventTypeGameEnded = "GAME_ENDED"; EventTypeBalloonUpdated = "BALLOON_UPDATED"
)
type StatusEvent[E any] struct {
	TransactionId uuid.UUID; WorldId world.Id; ChannelId channel.Id; MapId _map.Id; Instance uuid.UUID
	RoomId uint32; OwnerId uint32; VisitorId uint32; CharacterId uint32 // CharacterId = acting character
	Type string; Body E
}
type RecordBody struct{ GameType string; Wins uint32; Ties uint32; Losses uint32 }
type CreatedEventBody struct{ RoomType byte; Title string; PieceType byte; OwnerRecord RecordBody }
type ErrorEventBody struct{ Code string } // enterError KEY string, e.g. "NOT_WHEN_DEAD"
type EnteredEventBody struct{ Slot byte; RoomType byte; Title string; PieceType byte
	OwnerRecord RecordBody; VisitorRecord RecordBody; OwnerScore int32; VisitorScore int32 }
type LeftEventBody struct{ Slot byte; Status byte } // 3 closed / 4 left / 5 expelled
type RoomClosedEventBody struct{ VisitorStatus byte }
type ChatEventBody struct{ Slot byte; Message string }
type EmptyEventBody struct{}
type StartedEventBody struct{ RoomType byte; FirstMover byte; Deck []uint32 } // Deck empty for omok
type StonePlacedEventBody struct{ X uint32; Y uint32; StoneType byte }
type CardFlippedEventBody struct{ SecondFlip bool; Slot byte; FirstSlot byte; ResultType byte }
type AnswerEventBody struct{ Accept bool }
type SkippedEventBody struct{ Who byte }
type GameEndedEventBody struct{ ResultType byte; WinnerSlot byte // 0 win/1 tie/2 forfeit; slot 0 owner/1 visitor
	OwnerRecord RecordBody; VisitorRecord RecordBody; OwnerScore int32; VisitorScore int32 }
type BalloonEventBody struct{ Remove bool; RoomType byte; Title string; HasPassword bool
	PieceType byte; Occupancy byte; InProgress bool }
```
- Produces — processor API: `game.NewProcessor(l, ctx, db) Processor` with one method per command type (`Create(txId, f, characterId, roomType, title, private, password, pieceType) error`, `Visit(txId, f, characterId, roomId, password) error`, `Leave/Chat/Ready/Unready/Start/MoveStone/FlipCard/RequestTie/AnswerTie/GiveUp/RequestRetreat/AnswerRetreat/Expel/Skip/ExitAfterGame(cancel bool)`, plus `TeardownCharacter(characterId uint32) error` for Task 16). Status-event providers in `producer.go` keyed `producer.CreateKey(int(field.MapId()))` (chalkboards `producer.go:13` shape), every event populating RoomId/OwnerId/VisitorId.

- [ ] **Step 1: REST validation clients.** `data/character` (GET `characters/%d` via `requests.RootUrl("CHARACTERS")`, expose `Hp() uint16`, `Name() string`), `data/map` (atlas-data map by id, expose `FieldLimit() uint32`; consumer example `services/atlas-doors/.../data/map/`), `data/inventory` (GET `characters/%d/inventory` via `RootUrl("INVENTORY")`, expose `HasItem(itemId uint32) bool` — pattern `services/atlas-npc-shops/.../inventory/requests.go:10`), `data/chalkboard` (GET `chalkboards/%d` via `RootUrl("CHALKBOARDS")`, 404 = none — `services/atlas-channel/.../chalkboard/requests.go`). No new env vars: rely on `BASE_SERVICE_URL` fallback (known bug: never hard-code service URLs in overlays).
- [ ] **Step 2: Failing processor tests** (fake the four clients behind small interfaces injected into ProcessorImpl; message.Buffer assertions on emitted event types):
  - Create happy path → room in registry, `CREATED` + `BALLOON_UPDATED` emitted, owner record read (zeros).
  - Create validation ladder, in order, asserting the emitted enterError key (numeric code per the tenant `enterError` table): dead→`NOT_WHEN_DEAD`(4), fieldLimit `0x80`→`CANNOT_START_GAME_HERE`(11), chalkboard→`CANNOT_OPEN_MINI_ROOM_HERE`(13), missing item→`UNABLE`(6), already-in-room→`UNABLE`(6, design §3.4 convention). Omok item = `4080000 + pieceType` with pieceType clamped to [0,11]; MatchCards item = `4080100`, pieceType clamped [0,2]. Item NOT consumed.
  - Visit: room absent→`ROOM_CLOSED`(1); full→`FULL`(2); wrong password (private only; empty-password rooms always pass, case-insensitive compare — Cosmic `checkPassword`)→`INCORRECT_PASSWORD`(22); dead→4; chalkboard→13; success → `ENTERED` (slot 1, both records, scores; scores reset to 0 when `visitorId != lastVisitorId`) + `BALLOON_UPDATED` occupancy 2.
  - Leave (visitor, no game) → `LEFT{Slot:1, Status:4}` + balloon occupancy 1; Expel pre-game → `LEFT{Slot:1, Status:5}`; owner leave → `ROOM_CLOSED{VisitorStatus:3}` + balloon Remove + registry Remove.
  - Chat from member → `CHAT{Slot, Message}`; from non-member → dropped, no event.
- [ ] **Step 3: Run → FAIL.**
- [ ] **Step 4: Implement** processor + producer + consumer registration (`kafka/consumer/minigame/consumer.go` clones the chalkboards consumer: `InitConsumers` registering `NewConfig(l)("mini_game_command")(EnvCommandTopic)(groupId)` with Span+Tenant header parsers; `InitHandlers` registering one `message.AdaptHandler(message.PersistentConfig(handleX))` per command type; each handler early-returns unless `c.Type` matches, builds `field.NewBuilder(c.WorldId, c.ChannelId, c.MapId).SetInstance(c.Instance).Build()`, calls the processor). Wire into `main.go`. All registry mutation through `GetRegistry().Update/Create/Remove`; events emitted AFTER the swap via `producer.ProviderImpl`.
- [ ] **Step 5: Run → PASS (-race). Commit** (`feat(task-133): mini-games lifecycle commands (create/visit/leave/chat/expel)`).

---

### Task 15: Gameplay processing (ready/start/move/flip/skip/tie/retreat/forfeit/game-end)

**Files:**
- Modify: `services/atlas-mini-games/atlas.com/mini-games/game/{processor,producer}.go` + `processor_test.go`

**Interfaces:**
- Consumes: everything above; `record.ApplyResult`; ida-notes §G1 (initial `CurrentTurn`), §G2 (retreat pop count/turn).
- Produces: complete command coverage; `GAME_ENDED` semantics used by channel Task 18.

- [ ] **Step 1: Failing tests** (table-driven where possible):
  - Ready/Unready: visitor only; broadcast events; owner Start requires `VisitorReady && VisitorId != 0`, owner-only; Start emits `STARTED` (omok: empty board, FirstMover per room; matchcards: deck = `BuildDeck(MatchesToWin(pieceType))` shuffled) + `BALLOON_UPDATED{InProgress:true}`; Start clears exit-after flags and deny-tie bits; `CurrentTurn` initialized per §G1 rule from `FirstMover`.
  - MoveStone: out-of-turn dropped; occupied dropped; valid → `STONE_PLACED` + turn flip + move appended to history; winning move → `STONE_PLACED` then game-end path (owner win: FirstMover set to 0; visitor win: 1 — Cosmic `setPiece`), board wiped.
  - FlipCard: out-of-turn dropped; bad index (≥len(deck)) dropped; first flip → `CARD_FLIPPED{SecondFlip:false}`; second flip match → owner/visitor pairs++, `ResultType` from `matchcards.FlipResultType`, turn retained; mismatch → turn passes; last pair → game end (more pairs wins, equal → tie).
  - Skip: in-game member → `SKIPPED{Who: 0x01 owner / 0x00 visitor}` + turn flip.
  - Tie: request only when requester not denied; `TIE_REQUESTED` targeted at opponent; answer accept → tie game-end; decline → deny bit + `TIE_ANSWERED{Accept:false}`.
  - GiveUp → forfeit game-end (opponent wins, ResultType 2).
  - Retreat: request forwarded; accept → board/moves/turn adjusted per §G2, `RETREAT_ANSWERED{Accept:true}`; decline forwarded.
  - Mid-game leave/expel/teardown → forfeit game-end THEN membership teardown (`LEFT`/`ROOM_CLOSED`).
  - Game end (all paths): `record.ApplyResult` called once with correct winnerSlot/tie; scores per Cosmic — winner +50 (suppressed when forfeit && loser's forfeit count ≥4), loser +15 normal / −15 forfeit (forfeit count++), tie +10 both gated by `TieCooldownUntil` (5 min, injected clock `func() time.Time`); `GAME_ENDED` carries refreshed records + scores; room resets for rematch (board/deck/pairs/firstSlot/deny bits/inProgress) but keeps scores + FirstMover; exit-after flags honored (that side's leave processed after the result); `BALLOON_UPDATED{InProgress:false}`.
- [ ] **Step 2: Run → FAIL.**
- [ ] **Step 3: Implement.** One private `endGame(t, room, resultType, winnerSlot) (Room, []event)` transition used by win/tie/forfeit paths (single-resolution guard: no-op if `!InProgress()` — Cosmic `minigameMatchFinish` idempotence). `record.ApplyResult` commits BEFORE events emit.
- [ ] **Step 4: Run → PASS (-race). Commit** (`feat(task-133): omok and match-cards gameplay + records`).

---

### Task 16: Teardown consumers + rooms-in-field REST + service completion

**Files:**
- Create: `services/atlas-mini-games/atlas.com/mini-games/kafka/consumer/session/consumer.go`, `kafka/consumer/character/consumer.go`, `kafka/message/{session,character}/kafka.go`
- Create: `services/atlas-mini-games/atlas.com/mini-games/game/{resource,rest}.go`
- Modify: `main.go`

**Interfaces:**
- Consumes: `game.Processor.TeardownCharacter`; channel's session status contract (`services/atlas-channel/atlas.com/channel/kafka/message/session/kafka.go`: env `EVENT_TOPIC_SESSION_STATUS`, `StatusEvent{SessionId, AccountId, CharacterId, WorldId, ChannelId, Issuer, Type}`, `Type == "DESTROYED"`); atlas-chalkboards' character-status consumer (`services/atlas-chalkboards/atlas.com/chalkboards/kafka/consumer/character/consumer.go`) — copy ITS topic env + event shape verbatim for map-change/logout teardown.
- Produces: `GET /api/worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/games` returning `games` resources `{Id (roomId string), OwnerId, RoomType, Title, Private, PieceType, Occupancy, InProgress}` (channel Task 19 consumes).

- [ ] **Step 1: Message mirrors.** Copy the session StatusEvent struct into `kafka/message/session/kafka.go`; copy chalkboards' character event message shape into `kafka/message/character/kafka.go` (read the source — do not invent type names).
- [ ] **Step 2: Failing test:** processor test — `TeardownCharacter` on in-game visitor → forfeit + owner win + room stays (visitor slot cleared); on owner → room closed; on non-member → no-op.
- [ ] **Step 3: Implement consumers** (DESTROYED → `TeardownCharacter(e.CharacterId)`; character map-leave/logout → same; both registered in `main.go`) and the field REST resource (chalkboards `resource.go:55-84` shape over `GetRegistry().GetInField`).
- [ ] **Step 4: Run full service suite:** `go test -race ./... && go vet ./... && go build ./...` clean; `docker buildx bake atlas-mini-games` clean.
- [ ] **Step 5: Commit** (`feat(task-133): teardown consumers + rooms-in-field REST`).

---

### Task 17: atlas-channel — message mirror + minigame command processor

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/kafka/message/minigame/kafka.go` (byte-for-byte mirror of Task 14's contract — struct tags identical)
- Create: `services/atlas-channel/atlas.com/channel/minigame/{processor,producer}.go`

**Interfaces:**
- Consumes: Task 14 contract.
- Produces: `minigame.NewProcessor(l, ctx) *Processor` with emit methods (merchant pattern, `services/atlas-channel/atlas.com/channel/merchant/processor.go:45`):
  `Create(f field.Model, characterId uint32, roomType byte, title string, private bool, password string, pieceType byte) error`
  `Visit(f, characterId, roomId uint32, password string) error`, `Leave(f, characterId) error`, `Chat(f, characterId, message string) error`, `Ready/Unready/Start/GiveUp/RequestTie/RequestRetreat/Expel/Skip(f, characterId) error`, `MoveStone(f, characterId, x uint32, y uint32, stoneType byte) error`, `FlipCard(f, characterId, first bool, cardIndex byte) error`, `AnswerTie/AnswerRetreat(f, characterId, accept bool) error`, `ExitAfterGame(f, characterId, cancel bool) error`.
  Command providers keyed `producer.CreateKey(int(characterId))`, populating the full field envelope (WorldId/ChannelId/MapId/Instance).

- [ ] **Step 1: Write the mirror + processor + providers** (mechanical; each provider is the chalkboard/merchant provider shape).
- [ ] **Step 2: Verify:** `cd services/atlas-channel/atlas.com/channel && go build ./... && go vet ./...`.
- [ ] **Step 3: Commit** (`feat(task-133): channel minigame command emitters`).

---

### Task 18: atlas-channel — wire character_interaction arms + status consumer

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_interaction.go`
- Create: `services/atlas-channel/atlas.com/channel/kafka/consumer/minigame/consumer.go`
- Modify: `services/atlas-channel/atlas.com/channel/main.go` (InitConsumers/InitHandlers registration alongside the merchant consumer entries)

**Interfaces:**
- Consumes: Task 17 processor; Task 2–6 body funcs; `session.Processor.IfPresentByCharacterId(ch)(cid, op)`; `_map.NewProcessor(l, ctx).ForSessionsInMap(f, op)`; `character.NewProcessor(l, ctx).GetById()(cid)` + `model.NewFromCharacter` + `interaction.NewGameVisitor(slot, avatar, name, record)` / `interaction.NewGameRoom(...)` for room/enter encodes (existing merchant consumer `buildPersonalShopRoom` is the working example).
- Produces: fully wired vertical.

- [ ] **Step 1: Handler arms.** In the CREATE arm's Omok/MatchCards branch replace the log-and-return with:
```go
_ = minigame.NewProcessor(l, ctx).Create(s.Field(), s.CharacterId(), byte(roomType), sp.Title(), sp.Private(), sp.Password(), sp.NGameSpec())
return
```
VISIT arm: decode per Task 8 outcome (`OperationVisitGame` or existing `OperationVisit`), emit `Visit(s.Field(), s.CharacterId(), sp.SerialNumber(), password)` **in addition to** existing merchant handling (service drops non-members). CHAT arm: additionally emit `Chat(...)`. EXIT arm: additionally emit `Leave(...)`. Each of the fourteen MemoryGame arms replaces its `l.Debugf`-only body with the matching processor call (MoveStone splits the packed point: `x := uint32(sp.Point()); y := uint32(sp.Point() >> 32)` — low DWORD is x, written first by `PutStoneChecker`; FlipCard passes `sp.First(), sp.Index()`; TieAnswer/RetreatAnswer pass `sp.Response()`).
- [ ] **Step 2: Status consumer.** `kafka/consumer/minigame/consumer.go` cloning the merchant consumer skeleton (`InitConsumers` on `EnvEventTopicStatus`, `InitHandlers(l)(sc)(wp)(rf)`, tenant/world/channel guard `sc.Is(...)`). Per-event handlers with two helpers:
```go
func announceToRoom(...) // IfPresentByCharacterId for e.OwnerId and, when non-zero, e.VisitorId
func announceBalloon(...) // ForSessionsInMap(f, Announce(MiniRoomWriter)(MiniRoomBalloonBody(...)/RemoveBody(...)))
```
Event→packet mapping (design §5 table): CREATED → EnterResultSuccess(GameRoom w/ owner visitor+record) to owner + balloon; CREATE/ENTER_ERROR → `CharacterInteractionEnterResultErrorBody(e.Body.Code)` to `e.CharacterId`; ENTERED → EnterResultSuccess(full room) to visitor + `CharacterInteractionEnterBody(gameVisitor)` to owner + balloon; LEFT → `CharacterInteractionLeaveBody(slot, status)` to both (visitor's own copy uses its status; owner sees the departure); ROOM_CLOSED → Leave(status) to visitor + balloon remove; CHAT → ChatBody to room; READY/UNREADY/STARTED/STONE_PLACED/CARD_FLIPPED/TIE_*/RETREAT_*/SKIPPED/GAME_ENDED → the Task 2–5 bodies (CardFlipped `SecondFlip:false` goes to the opponent of `e.CharacterId` only; TIE/RETREAT_REQUESTED to opponent only; GAME_ENDED builds two `interaction.GameRecord{Unknown: uint32(roomType), Wins, Ties, Losses, Points: uint32(score)}`). Register in `main.go`.
- [ ] **Step 3: Verify:** `go build ./... && go vet ./... && go test -race ./...` in atlas-channel (existing handler tests must stay green).
- [ ] **Step 4: Commit** (`feat(task-133): channel minigame arms + status consumer`).

---

### Task 19: atlas-channel — map-entry balloon spawn + REST client

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/minigame/{requests,model,rest}.go`
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go`

**Interfaces:**
- Consumes: Task 16 REST endpoint; `spawnMerchantsForSession` (`map/consumer.go:666`) + `SpawnForSelf` aggregator (`map/consumer.go:159`).
- Produces: `minigame.Processor.InFieldModelProvider(f field.Model) model.Provider[[]Model]` + `ForEachInField(f, op)`; Model getters `Id() uint32`, `OwnerId() uint32`, `RoomType() byte`, `Title() string`, `Private() bool`, `PieceType() byte`, `Occupancy() byte`, `InProgress() bool`.

- [ ] **Step 1: REST client** — `requests.RootUrl("MINI_GAMES")` + resource `worlds/%d/channels/%d/maps/%d/instances/%s/games` (merchant `requests.go` shape; RestModel/Extract per chalkboards `rest.go`).
- [ ] **Step 2: `spawnMiniGamesForSession`** in `map/consumer.go`, added to `SpawnForSelf` next to the merchant call:
```go
_ = minigame.NewProcessor(l, ctx).ForEachInField(f, func(m minigame.Model) error {
	return session.Announce(l)(ctx)(wp)(interaction.MiniRoomWriter)(clientbound.MiniRoomBalloonBody(m.OwnerId(), m.RoomType(), m.Id(), m.Title(), m.Private(), m.PieceType(), m.Occupancy(), 2, m.InProgress()))(s)
})
```
- [ ] **Step 3: Verify build/vet/test; commit** (`feat(task-133): balloon spawn on map entry`).

---

### Task 20: Seed templates — all six versions

**Files:**
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_{83,84,87,92,95}_1.json`, `template_jms_185_1.json`

**Interfaces:**
- Consumes: ida-notes §G5 (v83/v95 verified mode values); per-version opcodes from `docs/packets/registries/*.yaml` (source of truth — grep `UPDATE_CHAR_BOX` and the serverbound `PLAYER_INTERACTION` handler rows per version; balloon: v83 0xA5, v84 0xA8, v87 0xB0, v95 0xB8, jms 0xA3, v92 from `gms_v92.yaml`).
- Produces: complete per-version socket config.

Current state (verified 2026-07-04): 83 ✓handler ✓sb-rows ✓writer; 84 same; **87 missing handler entirely**; **92 missing handler AND writer**; **95 missing handler**; jms ✓handler but missing MEMORY_GAME rows.

- [ ] **Step 1: gms_83.** To the existing `CharacterInteraction` writer entry (opCode 0x13A, `template_gms_83_1.json:2673-2710`) add operations rows: `"MEMORY_GAME_ASK_TIE": 50, "MEMORY_GAME_TIE_ANSWER": 51, "MEMORY_GAME_ASK_RETREAT": 54, "MEMORY_GAME_RETREAT_ANSWER": 55, "MEMORY_GAME_READY": 58, "MEMORY_GAME_UNREADY": 59, "MEMORY_GAME_START": 61, "MEMORY_GAME_RESULT": 62, "MEMORY_GAME_SKIP": 63, "MEMORY_GAME_MOVE_STONE": 64, "MEMORY_GAME_FIP_CARD": 68` (values re-checked against ida-notes §G5). Add the balloon writer entry:
```json
{ "opCode": "0xA5", "writer": "MiniRoom", "options": {} }
```
- [ ] **Step 2: gms_84.** Same additions (v84 sub-modes follow v83 — task-083 byte-identical precedent; opcode-table shift affects opCodes, not sub-modes); balloon opCode 0xA8. Verify the v84 writer entry exists first; add if missing (opcode from `gms_v84.yaml`).
- [ ] **Step 3: gms_87 / gms_92 / gms_95 / jms_185.** Add missing `CharacterInteractionHandle` handler entries (opCode from each version's serverbound registry row; `"validator": "LoggedInValidator"`; full operations table incl. CREATE/VISIT/CHAT/EXIT + all MEMORY_GAME rows), missing `CharacterInteraction` writer entry for 92, MEMORY_GAME writer rows for all four, and balloon writer entries. Mode values: v95 from ida-notes §G5; 87/92/jms derived from the nearest verified version and **bannered UNVERIFIED** (registry-note style of `bug_v84_opcode_table_shifted_vs_v83`). Do NOT copy v83 values blind where an IDB exists.
- [ ] **Step 4: Validate:** `jq . services/atlas-configurations/seed-data/templates/*.json > /dev/null` and `packet-audit operations --check` → exit 0.
- [ ] **Step 5: Commit** (`feat(task-133): seed-template minigame handler/writer/operations for all six versions`).

---

### Task 21: k8s + env wiring

**Files:**
- Create: `deploy/k8s/base/atlas-mini-games.yaml`
- Modify: `deploy/k8s/base/kustomization.yaml`, `deploy/k8s/base/env-configmap.yaml`, `deploy/k8s/overlays/main/{kustomization.yaml,patches/atlas-env-env.yaml,patches/db-name-suffix.yaml}`, `deploy/k8s/overlays/pr/{kustomization.yaml,patches/db-name-suffix.yaml,patches/consumer-group-env.yaml}`, `deploy/k8s/overlays/pr/scripts/gen-cleanup-env.sh`

- [ ] **Step 1: Deployment manifest** modeled on `deploy/k8s/base/atlas-buddies.yaml` (envFrom `atlas-env`, `DB_NAME: atlas-mini-games`, `DB_USER`/`DB_PASSWORD` from `db-credentials`, Service :8080) PLUS `replicas: 1` (in-memory registry — D1) and readiness probe:
```yaml
readinessProbe:
  httpGet: { path: /api/readyz, port: 8080 }
```
(`/api/readyz`, NOT `/readyz` — known rollout-wedge bug.)
- [ ] **Step 2: env-configmap:** add `COMMAND_TOPIC_MINI_GAME: mini-game-commands` style keys (match the existing naming convention in the file for CHALKBOARD/MERCHANT) + `EVENT_TOPIC_MINI_GAME_STATUS`. No `MINI_GAMES` URL override — BASE_SERVICE_URL fallback.
- [ ] **Step 3: Overlays:** add atlas-mini-games to every per-service list (main + pr kustomizations, db-name-suffix, consumer-group env, gen-cleanup-env.sh) — grep for `atlas-buddies` in `deploy/k8s/overlays/` and mirror every hit.
- [ ] **Step 4: Validate:** `kubectl kustomize deploy/k8s/overlays/main > /dev/null` and `kubectl kustomize deploy/k8s/overlays/pr > /dev/null`.
- [ ] **Step 5: Commit** (`feat(task-133): k8s + env wiring for atlas-mini-games`).

---

### Task 22: Full verification, rollout runbook, code review

**Files:**
- Create: `docs/tasks/task-133-miniroom-minigames/rollout.md`

- [ ] **Step 1: Full gates** (all must show clean output — quote it, don't summarize):
```bash
cd services/atlas-mini-games/atlas.com/mini-games && go test -race ./... && go vet ./... && go build ./...
cd services/atlas-channel/atlas.com/channel && go test -race ./... && go vet ./... && go build ./...
cd libs/atlas-packet && go test -race ./... && go vet ./...
# repo root:
docker buildx bake atlas-mini-games atlas-channel
tools/redis-key-guard.sh
packet-audit dispatcher-lint && packet-audit matrix --check && packet-audit operations --check
```
- [ ] **Step 2: rollout.md** — live-tenant PATCH runbook (format precedent: task-127 deployment notes): per-tenant config PATCH bodies for the handler operations rows, writer operations rows, and the new `MiniRoom` writer entry; rollout order (libs → atlas-mini-games deploy → atlas-channel deploy → tenant PATCH → channel restart — projection does not hot-reload handlers/writers); the v83 acceptance checklist from PRD §10 verbatim.
- [ ] **Step 3: Code review BEFORE PR** — invoke `superpowers:requesting-code-review` (dispatches plan-adherence-reviewer + backend-guidelines-reviewer; findings → `docs/tasks/task-133-miniroom-minigames/audit.md`). Address findings, re-run gates.
- [ ] **Step 4: Commit** (`docs(task-133): rollout runbook + verification evidence`). Do NOT open a PR in this task — that's a user decision via `superpowers:finishing-a-development-branch`.

---

## Self-Review Notes (performed at plan time)

- **Spec coverage:** design §3 → Tasks 11/12/14/15; §4 → 9/10/13–16; §5 → 14/17; §6 → 2–8; §7 → 17–19; §8 → 20; §9 → 10; §10 → 21; §12 → embedded per task + 22; §13 (G1–G5) → 1 (consumed by 2–8, 15, 20). PRD FR-1..FR-11 all land (FR-3 balloons: Tasks 6/18/19; FR-9 records: 10/15).
- **Known contract risks called out in-task:** result-packet shape (Task 4 defers to IDA on conflict), retreat entirely IDA-gated (Tasks 1/5/15), VISIT password layout (Tasks 1/8/18), MoveStone packed-point split (Task 18).
- **Type consistency spot-checks:** `RecordBody` name/fields identical in Tasks 14/17; body-func names in Task 2–6 match Task 18 usages; `record.ApplyResult(db, tenantId, gameType, ownerId, visitorId, winnerSlot, tie)` identical in Tasks 10/15; registry method set in Task 13 matches Task 14–16 usage; `minigame.Model` getters in Task 19 match Step 2 call.


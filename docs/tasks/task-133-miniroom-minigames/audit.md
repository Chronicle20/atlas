# Plan Audit — task-133-miniroom-minigames

**Plan Path:** docs/tasks/task-133-miniroom-minigames/plan.md
**Audit Date:** 2026-07-08
**Branch:** task-133-miniroom-minigames
**Base Branch:** main
**Section:** Plan adherence (plan-adherence-reviewer)

## Executive Summary

21 of 22 plan tasks are fully implemented with file-level evidence; Task 7
(packet-audit wiring) is PARTIAL on one cell: the clientbound Result arm
carries v83 tier-1 evidence only — the gms_v95 verify marker / evidence
record / audit report were never pinned, while every other new arm has both
versions. All done-gates were independently re-run during this audit and are
green: `go build/vet/test -race` clean in atlas-mini-games, atlas-channel,
and libs/atlas-packet; `docker buildx bake atlas-mini-games atlas-channel`
exit 0; `tools/redis-key-guard.sh` exit 0; `dispatcher-lint` / `matrix
--check` / `operations --check` all exit 0; `kubectl kustomize` clean for
both overlays. The two parent-flagged deviations (Result wire-layout
correction per IDA §G5; interim room-enter/enter-arm task replacing the
unimplementable legacy encode) are confirmed legitimate and fully executed.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | IDA gates G1–G5 → ida-notes.md | DONE | `ida-notes.md` §G1:30, §G2:101, §G3:174, §G4:243, §G5:304, coverage summary :612 ("No unresolved fnames"); commit 1d63571367 + review-fix a51e0f2e86 |
| 2 | Clientbound bodyless/simple arms | DONE | `libs/atlas-packet/interaction/clientbound/interaction_minigame.go:15,35,55,76,168` (Ready/Unready/RequestTie/AnswerTie/Skip); body funcs `interaction_body.go:148-199`; mode consts incl. load-bearing `MEMORY_GAME_FIP_CARD` |
| 3 | Start/MoveStone/CardSelect arms | DONE | `interaction_minigame.go:199` (StartOmok), `:231` (StartMatchCards), `:272` (MoveStone), `:310`/`:345` (CardSelect First/Second); bodies `interaction_body.go:202-236` |
| 4 | Result arm | DONE | `interaction_minigame.go:396`; layout corrected per IDA §G5 (plan.md:233-248 documents the in-branch correction — authorized deviation); 3 shape fixtures `interaction_minigame_test.go:200-243`; body `interaction_body.go:239` |
| 5 | Retreat arms (IDA-derived) | DONE | `interaction_minigame.go:99` (RetreatRequest), `:125` (RetreatAnswer); `CharacterInteractionMiniGameRetreatAnswerBody(accept, stoneCount, turnSlot)` `interaction_body.go:187` matches §G2 field list (ida-notes.md:136-149, v83 0x6e41f9 + v95 0x684620) |
| 6 | UPDATE_CHAR_BOX balloon | DONE | `mini_room_balloon.go` — `MiniRoomBalloonBody` :133, `MiniRoomBalloonRemoveBody` :137; v83+v95 verify markers in `mini_room_balloon_test.go` (ida 0x938ba5 / 0x8e8d30) |
| 7 | packet-audit wiring | PARTIAL | `tools/packet-audit/cmd/run.go:1921-1983` (all 13 arm cases + `CUser::OnMiniRoomBalloon`(+`#Remove`)); v83+v95 markers/evidence for 12 arms + balloon; checkers exit 0 (re-run this audit). **Gap: Result arm has gms_v83 evidence only** — see Skipped/Partial section |
| 8 | Serverbound game-VISIT decoder | DONE | `libs/atlas-packet/interaction/serverbound/operation_visit_game.go:12` (fname `CUserLocal::HandleLButtonDblClk`, §G4 layout: serialNumber, hasPassword, [password], trailing 0); round-trip tests; no verify marker by documented design (`operation_visit_game_test.go:9-15`) |
| 9 | atlas-mini-games scaffold | DONE | `services/atlas-mini-games/atlas.com/mini-games/main.go` (DB connect + `/api/readyz` under `/api/` base); `go.work:57`; `.github/config/services.json:273-277`; `docker-bake.hcl:69` (both hand-synced places) |
| 10 | record domain + REST | DONE | `record/administrator.go:44` `ApplyResult` uses `db.Transaction` directly (:45, no-op-aware comment :42); `record/provider.go:15,33`; `record/entity.go:11`; 5 tests incl. atomicity; REST `record/resource.go` (InitResource curried with db per 981c236f57) |
| 11 | Omok engine | DONE | `game/omok/engine.go:3-8` (BoardSize/Cells/Place/Wins); 19 tests in `engine_test.go` |
| 12 | Match Cards engine | DONE | `game/matchcards/engine.go:8,23,34,44` (MatchesToWin/BuildDeck/Shuffle/FlipResultType); 9 tests |
| 13 | Room model + registry | DONE | `game/{model,builder,registry}.go`; `GetRegistry` `registry.go:36`; 12 registry tests + model tests, `-race` clean; defensive-copy fix 11933ac5b4 |
| 14 | Kafka messages + lifecycle | DONE | `kafka/message/minigame/kafka.go` (full command/event contract); `data/{character,map,inventory,chalkboard}/` clients; `game/processor.go:223-485` (Create/Visit/Leave/Expel/Chat/TeardownCharacter); consumer `kafka/consumer/minigame/consumer.go` registered `main.go:66-67`; validation-ladder tests `processor_test.go:226,357,424` |
| 15 | Gameplay processing | DONE | `game/processor.go:494-800+` (Ready/Start/MoveStone/FlipCard/Tie/GiveUp/Retreat/Skip, endGame); record committed before registry swap (f5d4ffd5fd); 46 processor tests incl. `TestMoveStone_WinningMoveEndsGame:833`, `TestFlipCard_LastPairWinAndTie:939`, `TestTeardownCharacter_*:1203-1231` |
| 16 | Teardown consumers + field REST | DONE | `kafka/consumer/{session,character}/consumer.go` + message mirrors, registered `main.go:74-79`; `game/resource.go:32-33` (`GET .../instances/{instanceId}/games`); bonus channel-change teardown fix fd6f9a0dd8 |
| 17 | Channel mirror + command processor | DONE | `services/atlas-channel/atlas.com/channel/kafka/message/minigame/kafka.go` — struct-tag/const diff vs service side is empty (verified); `minigame/{processor,producer}.go` |
| 18 | Wire interaction arms + status consumer | DONE | 18 `minigame.NewProcessor` call sites in `socket/handler/character_interaction.go`; `kafka/consumer/minigame/consumer.go` (19 event-type references); registered `main.go:218,553`. Includes the plan-authorized interim task: `interaction_minigame_room.go:69` / `interaction_minigame_enter.go:35` (v83+v95 verified, `run.go:1954-1972`), version-gated jobCode (d4850dbfa4), legacy NewGameRoom/GameVisitor/GameMiniRoom paths removed — zero non-comment references remain (`libs/atlas-packet/interaction/mini_room.go:108`, `socket/model/mini_room.go:113` are explanatory comments only) |
| 19 | Map-entry balloon spawn + REST client | DONE | `minigame/{requests,model,rest}.go`; `spawnMiniGamesForSession` `kafka/consumer/map/consumer.go:703`, wired into spawn aggregation at `:278` (commit 3acab35e57) |
| 20 | Seed templates ×6 | DONE | All six templates: 25 `MEMORY_GAME` occurrences, 1 `"writer": "MiniRoom"`, `FIP_CARD` typo preserved; `jq` valid; 87/92/jms derivations documented in `seed-unverified-notes.md` + rollout.md caveats (9b213be4f6) |
| 21 | k8s + env wiring | DONE | `deploy/k8s/base/atlas-mini-games.yaml:9` (`replicas: 1`), `:43` (`/api/readyz`); `base/kustomization.yaml:40`; `env-configmap.yaml:46,125`; main+pr overlays incl. `db-name-suffix`, `consumer-group-env`, `gen-cleanup-env.sh:37`; `kubectl kustomize` main+pr clean (re-run this audit) |
| 22 | Verification + rollout + review | DONE | `rollout.md` (PATCH bodies :3, rollout order :164, PRD §10 acceptance checklist :176, quoted gate evidence :208); all gates independently reproduced green by this audit; code-review step satisfied by this audit pass; no PR opened (correct per plan) |

**Completion Rate:** 21/22 DONE + 1 PARTIAL (~98%)
**Skipped without approval:** 0
**Partial implementations:** 1 (Task 7, minor)

## Skipped / Deferred Tasks

**Task 7 — Result arm missing gms_v95 tier-1 evidence.**
`docs/packets/evidence/gms_v95/` contains evidence yamls for all 14 other
minigame-family cells but not `interaction.clientbound.InteractionInteractionMiniGameResult`;
the only verify marker is `interaction_minigame_test.go:199` (`version=gms_v83
ida=0x6e4463`), and no v95 audit report or export splice exists for
`#MemoryGameResult`. The ida-notes RESULT section (ida-notes.md §G5) cites
only v83 addresses (0x6e4463 / 0x64e423). Plan Task 7 expected "✅ on gms_v83
+ gms_v95" per arm, and the wiring commit (25841f297d) plus rollout.md's
caveat ("full per-mode byte-fixture tests for gms_v83 and gms_v95") both
overstate coverage for this one arm. Impact is low: the encoder is
version-uniform (no version gate) and the three shape fixtures round-trip
under all `test.Variants` including v95 context — the missing piece is the
IDA evidence pin, not code. No written justification for the omission was
found, so it reads as an oversight rather than a decision.

**Task 8 — no verify marker on OperationVisitGame (documented, acceptable).**
The decoder is implemented and tested; the test file explains
(`operation_visit_game_test.go:9-15`) that a verify marker requires the
heavier evidence-pin pass this task did not perform, and the fname is wired
into candidatesFromFName for a future pass. Plan Task 8 required only the
decoder, so this is compliant — noted for completeness.

## Build & Test Results

All commands re-run by this audit on 2026-07-08 (not taken from prior claims):

| Gate | Result | Notes |
|------|--------|-------|
| atlas-mini-games `go build && go vet && go test -race ./...` | PASS | exit 0 |
| atlas-channel `go build && go vet && go test -race ./...` | PASS | exit 0 |
| libs/atlas-packet `go vet && go test -race ./...` | PASS | exit 0 |
| `docker buildx bake atlas-mini-games atlas-channel` | PASS | exit 0 |
| `tools/redis-key-guard.sh` (repo root) | PASS | exit 0 |
| `packet-audit dispatcher-lint` | PASS | "dispatcher-lint: clean", exit 0 |
| `packet-audit matrix --check` | PASS | exit 0 |
| `packet-audit operations --check` | PASS | "operations check OK (0 absent-writer note(s))", exit 0 |
| `kubectl kustomize deploy/k8s/overlays/{main,pr}` | PASS | both build (deprecation warning only) |
| `jq .` all six seed templates | PASS | valid JSON |

## Overall Assessment

- **Plan Adherence:** MOSTLY_COMPLETE (one minor evidence-ceremony gap)
- **Recommendation:** NEEDS_FIXES (single small, producible item below; code itself is complete and green)

Both parent-flagged deviations are confirmed faithful to their authorization:
the Result wire-layout correction is documented inside plan.md Task 4 itself
with the superseded Cosmic table retained for history, and the interim
room-enter/enter-arm work (commits ed73966a5f → 75534e51d9) delivered
verified v83+v95 encoders while removing the legacy wrong-layout paths with
zero remaining callers.

Housekeeping note (not a gap): plan.md checkboxes were largely not ticked
during execution (5 of 103 checked — only Task 4's steps); completion was
established from code/commit evidence instead.

## Action Items

1. Pin the gms_v95 tier-1 evidence for the Result arm: resolve the v95
   `COmokDlg::OnGameResult`/`CMemoryGameDlg::OnGameResult` address in the v95
   IDB, splice the `#MemoryGameResult` export entry, add the
   `version=gms_v95` verify marker + evidence yaml + audit report, regenerate
   the matrix — or, if the v95 IDB genuinely cannot resolve it, document the
   omission in ida-notes.md §G5 and correct the rollout.md caveat that
   currently claims full v83+v95 fixture coverage.

---

## Whole-Branch Integration Review

**Verdict: PASS — ship-ready.** No Critical or Important cross-task defects.
The six named seams are all consistent; builds and tests are green
(`atlas-mini-games` build+vet+test, `atlas-channel` build, `atlas-packet`
interaction tests — all exit 0). Three Minor cleanups below, none blocking.

### Cross-task seam verification

1. **Command/event contract parity — PASS.**
   `services/atlas-mini-games/.../kafka/message/minigame/kafka.go` and
   `services/atlas-channel/.../kafka/message/minigame/kafka.go` are
   **byte-identical below line 1** (both `package minigame`; 217 lines each;
   `diff` with line 1 stripped is empty). Every command type (18), event type
   (20), and body struct/field/json-tag matches. No decode drift across the wire.

2. **Operations-key ↔ packet round-trip — PASS.** Every clientbound body func in
   `libs/atlas-packet/interaction/clientbound/interaction_body.go` resolves one of
   11 `MEMORY_GAME_*` keys via `WithResolvedCode("operations", …)`. All 11 keys
   (ASK_TIE 50, TIE_ANSWER 51, ASK_RETREAT 54, RETREAT_ANSWER 55, READY 58,
   UNREADY 59, START 61, RESULT 62, SKIP 63, MOVE_STONE 64, **FIP_CARD 68** — the
   load-bearing typo) are present in the **writer** operations map of all six seed
   templates (gms_83/84/87/92/95, jms_185). The 14-key serverbound **handler**
   operations map covers every arm the socket handler dispatches (adds FORFEIT,
   EXPEL, EXIT_AFTER_GAME, CANCEL_EXIT_AFTER_GAME; RESULT is correctly
   clientbound-only). GMS values match the constant comments exactly; jms uses its
   own shifted enum (ASK_TIE 47 … FIP_CARD 65) that is internally self-consistent
   between its handler and writer maps. First- and second-flip both resolve mode 68
   (`interaction_body.go:225,231`) — matches the single client `OnMemoryGameFlipCard`.

3. **Event→packet targeting — PASS** (owner/visitor never swapped),
   `services/atlas-channel/.../kafka/consumer/minigame/consumer.go`:
   - CARD_FLIPPED first flip (`SecondFlip:false`) → `announceTo(opponentOf(e),…)`
     (364); second flip → `announceToRoom` (367). `opponentOf` keys off
     `e.CharacterId` = the flipper (producer.go:139). ✔
   - TIE_REQUESTED / RETREAT_REQUESTED → `announceTo(opponentOf(e),…)` (312-314). ✔
   - CREATE_ERROR / ENTER_ERROR → `announceTo(e.CharacterId,…)` (213). ✔
   - GAME_ENDED → `CharacterInteractionMiniGameResultBody(resultType,
     e.Body.WinnerSlot == 1, ownerRecord, visitorRecord)` (428): winnerSlot 1 ⇒
     visitorWon, owner record first. Matches the event body doc. ✔

4. **Turn/slot convention — PASS** (one convention: 0 owner / 1 visitor across
   service + channel + packet). Service `Room.SlotOf`, channel `slotOf`
   (consumer.go:103 owner→0 else 1), packet `yourSlot`, and the `firstMover` byte
   all agree. `STARTED.FirstMover` carries the raw `room.FirstMover()`
   (producer.go:119); the service sets `CurrentTurn = 1 - FirstMover`
   (processor.go:568). Server `stoneColor(slot,firstMover)` (=2 if slot==firstMover
   else 1, processor.go:181) equals the client `m_nPlayerColor = 2 - (startByte !=
   mySlot)`. SKIPPED.Who and GAME_ENDED winnerSlot use the same basis.

5. **Room-enter path — PASS.** `grep` for the removed `NewGameRoom` /
   `GameVisitor` across `services/atlas-channel` + `libs/atlas-packet` returns
   **zero** hits. CREATED → `CharacterInteractionMiniGameRoomBody(…, yourSlot 0,
   [owner])` to the owner (consumer.go:196-197); ENTERED → full snapshot with
   `yourSlot = e.Body.Slot` (=1, producer.go:67) to the visitor **and** ENTER
   (avatar+record) to the owner (consumer.go:239-241).

6. **Teardown coverage — PASS.** All four lifecycle exits reach
   `TeardownCharacter` → `leave` (forfeit-then-teardown, processor.go:485-492):
   SESSION_DESTROYED (session consumer), LOGOUT / MAP_CHANGED / CHANNEL_CHANGED
   (character consumer). All three consumer groups registered in
   `services/atlas-mini-games/.../main.go:66-79`. The mid-game forfeit resolves
   first (`leave` calls `endGame` before the membership teardown,
   processor.go:385-397).

### Minor findings (new, non-blocking)

- **M1 — Balloon `hasPassword` computed two different ways across task-18/task-19.**
  Event path: `HasPassword: r.Private() && r.Password() != ""`
  (`services/atlas-mini-games/.../game/producer.go:200`). Map-entry path passes the
  raw `m.Private()` into the same wire slot
  (`services/atlas-channel/.../kafka/consumer/map/consumer.go:708`), because the
  rooms-in-field REST model exposes only `Private`, not a computed has-password
  (`services/atlas-mini-games/.../game/rest.go:13,48`). Failure scenario: a room
  created `private=true, password=""` (the VISIT gate at processor.go:320 treats it
  as *unlocked*) shows a lock icon on the balloon to a player walking into the map
  but no lock via the live BALLOON_UPDATED event — the two renders of the same room
  disagree. Cosmetic; the stock client forces a password when private is checked, so
  it only bites a hand-crafted CREATE. Fix: add a computed `hasPassword` to
  `RestModel`/`Transform` and pass it at consumer.go:708.

- **M2 — `record.Builder` is dead code.** `record.NewBuilder` /
  `SetWins/SetTies/SetLosses` (`services/atlas-mini-games/.../record/builder.go`)
  have **zero call sites** (incl. tests). `GetOrZero` builds `Model{}` with a struct
  literal (`record/provider.go:20-24`), contradicting the builder's own doc comment
  ("Used for zero-filled/absent-row results (GetOrZero)"). Recommend deleting
  builder.go; the comment is actively misleading.

- **M3 — Balloon + enter packet tests are RoundTrip-only.**
  `mini_room_balloon_test.go` and `interaction_minigame_enter_test.go` assert only
  full-consumption via `test.RoundTrip` (no field-value assert), unlike
  `interaction_minigame_test.go` / `interaction_minigame_room_test.go` which pin
  exact bytes. A symmetric field-order bug in the balloon/enter encoder+test-decoder
  pair would pass. IDA read order is documented in the `packet-audit:verify` markers,
  so risk is low; recommend one exact-byte golden for the balloon full-field case.

### Triage of accumulated per-task Minors

- **RoundTrip helper (no field-equality assert):** partially-resolved — room/body
  packets have exact-byte goldens; balloon+enter do not (see M3). **Accept-as-is**,
  optional golden.
- **record/builder.go possibly dead:** **confirmed dead → recommend drop** (M2).
- **chalkboard.HasOpen fails open on REST error:** **accept-as-is** (documented).
- **SKIP handler ungated:** **accept-as-is** — verified harmless: an out-of-turn
  SKIP sets `CurrentTurn = 1 - skipperSlot`, a no-op for the non-current player;
  only the current player's skip changes the turn (processor.go:874-891).
- **getOrCreate read-modify-write (replicas:1-safe only):** **accept-as-is** — the
  service pins `replicas: 1`.
- **v92 seed modes UNVERIFIED / jms room-enter not tier-1:** **accept-as-is** —
  documented in seed-unverified-notes.md; keys are structurally present and each
  version's handler/writer maps are internally self-consistent. Verify against a v92
  IDB when one exists before claiming those cells tier-1.

---

# Backend Guidelines Audit — task-133 (miniroom-minigames)

- **Scope:** `services/atlas-mini-games/atlas.com/mini-games` (new service), `services/atlas-channel/atlas.com/channel/{minigame,kafka/consumer/minigame,socket/handler/character_interaction.go,kafka/consumer/map/consumer.go}`, `libs/atlas-packet/interaction/**` additions, deploy wiring
- **Guidelines Source:** backend-dev-guidelines skill (all 11 resources read)
- **Date:** 2026-07-08
- **Build:** PASS — `go build ./...`, `go vet ./...` clean; `docker buildx bake atlas-mini-games` exit 0
- **Tests:** PASS — `go test ./... -count=1` all green (game, game/omok, game/matchcards, record, kafka/consumer/character)
- **Overall:** NEEDS-WORK (build+tests pass; blocking FAIL checks below)

## Build & Test Results

```
BUILD_OK / VET_OK
ok  atlas-mini-games/game 0.044s | game/matchcards 0.002s | game/omok 0.002s
ok  atlas-mini-games/kafka/consumer/character 0.008s | record 0.013s
docker buildx bake atlas-mini-games → exit 0
```

## Blocking (must fix)

### B1. EXT-01 — `data/map` client cannot decode any real atlas-data map response (feature-dead path, PROVEN)

`data/map/rest.go` `RestModel` implements only `GetName/GetID/SetID` — no
`SetToOneReferenceID` / `SetToManyReferenceIDs`. The upstream atlas-data map
resource ALWAYS emits a `relationships` block
(`services/atlas-data/atlas.com/data/map/rest.go:72-79` — `GetReferences()`
unconditionally returns portals/reactors/npcs/monsters).

**Empirically reproduced** during this audit with a temporary httptest probe
(fixture = JSON:API map doc with a `relationships` block):

```
decode failed with relationships block present:
struct *mapdata.RestModel does not implement UnmarshalToManyRelations
```

Consequence chain: `FieldLimit()` (`data/map/processor.go:39-45`) errors on
every fetch → `create()` aborts at the fieldLimit gate
(`game/processor.go:242-244`) → error swallowed by the command consumer
(`kafka/consumer/minigame/consumer.go:101`, `_ =`) → **no mini-game room can
ever be created against a real atlas-data**. This is the exact task-037
failure mode documented in `libs/atlas-rest/CLAUDE.md`. Note the in-game
acceptance checklist in `rollout.md:176-183` is still unchecked, consistent
with this never having been exercised live.

**Fix:** add the two no-op stubs to `data/map/rest.go` (see
`libs/atlas-rest/CLAUDE.md` boilerplate).

### B2. EXT-02 — zero httptest-backed integration tests for the four external clients

`data/chalkboard`, `data/character`, `data/inventory`, `data/map` have **no
test files at all** (`go test` reports `[no test files]`). The guideline
(EXT-02, `libs/atlas-rest/CLAUDE.md` "How to be sure you got it right")
requires an httptest fixture test per new external client precisely because
it catches B1. The audit probe that proved B1 is the test that should exist.

### B3. Ingress/service-URL wiring — both REST endpoints unreachable; channel reconciliation misrouted

- `deploy/shared/routes.conf` has **no route** for either endpoint.
- `GET /api/worlds/{w}/channels/{c}/maps/{m}/instances/{i}/games` falls through
  to `^/api/worlds(/.*)?$` → `atlas-world:8080` (routes.conf:517-519) → 404.
- `GET /api/characters/{id}/game-records` falls through to
  `^/api/characters(/.*)?$` → `atlas-character:8080` (routes.conf:327-329) → 404.
- No `MINI_GAMES_SERVICE_URL` exists anywhere under `deploy/` (grep: zero hits),
  so atlas-channel's `requests.RootUrl("MINI_GAMES")`
  (`channel/minigame/requests.go:15`) falls back to `BASE_SERVICE_URL`
  (`deploy/k8s/base/env-configmap.yaml:6` → nginx) and the map-entry balloon
  reconciliation (`channel/kafka/consumer/map/consumer.go` SpawnForSelf hook)
  will 404 on every map entry — swallowed at Debug level.

**Fix:** add `location` blocks for `/api/worlds/.../instances/[^/]+/games` and
`/api/characters/[^/]+/game-records` → `atlas-mini-games:8080` in
`deploy/shared/routes.conf` + regenerate the K8s ingress ConfigMap (the
`sync-k8s-ingress-routes.sh` script currently errors on repo layout —
`deploy/k8s/ingress.yaml` not found — so update `deploy/k8s/base/atlas-ingress.yaml`
by whatever the current mechanism is), or set `MINI_GAMES_SERVICE_URL`.

### B4. SCAFFOLD-06 — no docker-compose entry

`deploy/compose/docker-compose.core.yml` has no `atlas-mini-games` service
block (peers e.g. `atlas-chalkboards` at line 110). Compose environments will
not run the service.

### B5. SCAFFOLD-08 — no Bruno collection

`services/atlas-mini-games/.bruno/` does not exist. Required for REST services
(scaffolding-checklist.md §4).

### B6. No service README

`services/atlas-mini-games/atlas.com/mini-games/README.md` does not exist.
patterns-ingress-documentation.md requires REST endpoints +
`COMMAND_TOPIC_MINI_GAME` commands + `EVENT_TOPIC_MINI_GAME_STATUS` events
documented.

### B7. DOM-13/14 — `record` domain has no processor layer; handler calls provider directly

`record/resource.go:39` — `handleGetGameRecords` calls `GetByCharacter(db...)`
(a provider function) directly. There is no `record/processor.go`. The file's
own comment cites the buddies-list shape as precedent, but buddies has a
processor and its handler calls it
(`services/atlas-buddies/atlas.com/buddies/list/resource.go:44`
`NewProcessor(...).GetByCharacterId(...)`). Anti-patterns.md: "Handlers calling
provider functions directly" is a critical layer violation.

### B8. DOM-14 (game) — handler bypasses processor for the in-field read

`game/resource.go:46` — `handleGetGamesInField` calls
`GetRegistry().GetInField(t, f)` directly. The `game.Processor` interface
(`game/processor.go:104-124`) exposes no rooms-in-field read; the handler
reaches into the data layer (registry) itself.

### B9. DOM-11 / multi-tenancy anti-pattern — manual tenantId plumbing in `record`

- `record/provider.go:15-17` `GetOrZero(db, tenantId, ...)` +
  `Where("tenant_id = ? AND ...")`; `record/administrator.go:14-16` same in
  `getOrCreate`.
- `game/processor.go:283,354-358,959,969-973` pass bare `p.db` (never
  `p.db.WithContext(p.ctx)`), so the GORM tenant callback cannot apply and the
  manual WHERE is the only tenant filter.
- Anti-patterns.md explicitly bans "Passing TenantId to providers/update/delete"
  and "Manual `Where(\"tenant_id = ?\", ...)`"; providers are also eager
  `(Model, error)` functions rather than `database.EntityProvider` lazy
  providers (patterns-provider.md).

**Mitigating (verified):** every query IS tenant-scoped — reads via the manual
WHERE (provider.go:17, administrator.go:16), writes via `tx.Save` on rows
fetched under that filter, creates set `TenantId` explicitly
(administrator.go:24-29). This is a pattern violation, not a tenant leak.

### B10. Dead code — `record/builder.go`

`record.NewBuilder`/`SetId`/`SetWins`/`SetTies`/`SetLosses`/`Build` are
referenced nowhere (grep across the whole module: only builder.go itself).
`GetOrZero` constructs `Model` literals directly (provider.go:20-24) and the
write path builds `Entity` values (administrator.go:24-29). The earlier audit
section already recommended dropping it; still present. Anti-patterns.md:
"Leaving dead code after refactoring". Either route model construction through
the builder or delete the file.

## Non-Blocking (should fix)

- **W1. DOM-01 (game):** `game/builder.go:202-204` `Build()` returns `Room`
  with no validation (guideline: "Validation occurs in Build()"). The `record`
  builder does validate (builder.go:51-60) — but is dead (B10).
- **W2. EXT-01 (chalkboard/character clients):** `data/chalkboard/rest.go` and
  `data/character/rest.go` also lack the no-op relationship stubs. Currently
  benign — upstream chalkboards/character REST models have no `GetReferences`
  (grep: 0 hits) — but one upstream change away from B1. `data/inventory`
  implements the full include machinery correctly (rest.go
  `SetToManyReferenceIDs`/`SetReferencedStructs`).
- **W3. DOM-02/DOM-05:** no `ToEntity()` on `record.Model`, no
  `TransformSlice` in either rest.go. Calibrated against the repo: only 2
  rest.go files in all of `services/` define `TransformSlice`; the dominant
  convention is the guideline-blessed `model.SliceMap(Transform)(...)`
  composition, which both handlers use (record/resource.go:46,
  game/resource.go:48). Recorded as deviation-by-convention, not blocking.
- **W4. DOM-21 (roomType):** `game/processor.go:51-54` redeclares
  `RoomTypeOmok=1 / RoomTypeMatchCards=2`, duplicating
  `libs/atlas-packet/interaction/room.go:16-17` (`OmokRoomType`,
  `MatchCardRoomType`); `channel/kafka/consumer/minigame/consumer.go:140-149`
  (`gameTypeCode`) re-maps the same values a third time. Not in
  `libs/atlas-constants` (grep 408/Omok/MiniGame: 0 hits), so not a strict
  DOM-21 FAIL, but the byte contract now lives in three places. Item ids
  4080000/4080100 have no shared equivalent → local consts acceptable;
  inventory-type derivation correctly uses
  `inventory.TypeFromItemId(item.Id(...))` (data/inventory/processor.go:33).
- **W5. Command-consumer error swallowing:** all 18 handlers in
  `kafka/consumer/minigame/consumer.go` discard processor errors (`_ =`,
  e.g. :101). A failed `record.ApplyResult` inside `endGame` is invisible
  (the teardown consumers do log — consumer/session:52-54,
  consumer/character:57-59). Log at Error level like the teardown consumers.
- **W6. DOM-20:** table-driven tests present in `game/processor_test.go` (6
  tables) and `game/matchcards/engine_test.go` (2); `registry_test.go`,
  `model_test.go`, `omok/engine_test.go`, `record/*_test.go` are plain
  sequential tests without `t.Run` subtests.

## Verified PASS (evidence)

| Check | Evidence |
|---|---|
| Kafka Buffer/Emit pattern | `kafka/message/message.go:36-51` (Buffer + single flush); every processor method wraps via `p.emit` (`game/processor.go:188-190`); events buffered with `mb.Put` and flushed once |
| `db.Transaction` (not ExecuteTransaction) | `record/administrator.go:44-71` uses `db.Transaction` directly with comment citing the no-op bug; **atomicity regression-tested** via sqlite trigger fault injection (`record/administrator_test.go:139-175` `TestApplyResult_Atomic`) |
| DOM-24 producer stub | `game/testmain_test.go:11` + `kafka/consumer/character/testmain_test.go:11` `producertest.InstallNoop()`; no `t.Cleanup(producer.ResetInstance)` anywhere |
| DOM-10 tenant callbacks in tests | `record/administrator_test.go:29`, `game/processor_test.go:71` `database.RegisterTenantCallbacks(l, db)` |
| DOM-06/07 processor shape | `game/processor.go:126-157` `NewProcessor(l logrus.FieldLogger, ctx, db)`; consumers pass handler logger; data clients same shape (`data/*/processor.go`) |
| Immutability + Builder | `game/model.go` all-private fields, defensive copies for `Moves()/Deck()` (:145-162); `game/builder.go` copy-on-write Clone (:35-37), defensive `SetMoves/SetDeck` (:125-146); registry mutations only via `Update` under one write lock (`game/registry.go:109-129`) |
| Registry singleton | `game/registry.go:32-44` `sync.Once` + RWMutex, tenant-partitioned maps |
| DOM-23 topics | `COMMAND_TOPIC_MINI_GAME` / `EVENT_TOPIC_MINI_GAME_STATUS` in `deploy/k8s/base/env-configmap.yaml:46,125` as `KEY: "KEY"`, alphabetical; no literal topic env in `deploy/k8s/base/atlas-mini-games.yaml`; consumed via `topic.EnvProvider` |
| DOM-22 equivalent (shared root Dockerfile) | `docker buildx bake atlas-mini-games` exit 0; `go.work:57`, `docker-bake.hcl:69`, `.github/config/services.json:273-277` all wired |
| SCAFFOLD-01/02 | services.json entry ✓; `deploy/k8s/base/atlas-mini-games.yaml` — replicas:1 pinned with design comment, `envFrom: atlas-env`, db-credentials, **readinessProbe path `/api/readyz`** (:41-44, avoids the known probe-path bug) |
| SCAFFOLD-07 seed templates | All six templates (`gms_83/84/87/92/95, jms_185`) carry the `MiniRoom` writer + full `MEMORY_GAME_*` operations tables; Go constant strings (`interaction_body.go:53-63`) match template keys byte-for-byte, including the deliberately load-bearing `MEMORY_GAME_FIP_CARD` on both sides |
| Dispatcher mode rule | Every minigame clientbound body resolves its mode via `atlas_packet.WithResolvedCode("operations", ...)` (`interaction_body.go:133-240`); no literal mode bytes; `MiniRoomBalloon` documented non-dispatcher (`mini_room_balloon.go:13-16`) |
| Socket handler thinness | All new `character_interaction.go` arms decode then delegate to `minigame.NewProcessor(l, ctx).<Cmd>` — no business logic in handlers |
| DOM-12 / SUB-04 | No `os.Getenv` outside `main.go`/kafka consumer config; no `json.NewDecoder`/`json.Unmarshal`/`io.ReadAll` in any resource.go |
| No `*_testhelpers.go` | find across mini-games + channel/minigame + libs/atlas-packet/interaction: 0 hits |
| DOM-18/19 | `GetName/GetID/SetID` on all REST models (record/rest.go:18-29, game/rest.go:19-34); no nested Data/Type/Attributes anywhere |
| DOM-09 | Transform error paths checked in both handlers (record/resource.go:46-51, game/resource.go:48-53) |
| SEC | Not an auth/token service — SEC-01..03 N/A; no hardcoded secrets (room passwords are user data, in-memory only) |

**Accepted context (not re-flagged):** chalkboard.HasOpen fail-open,
ungated SKIP, getOrCreate read-modify-write under replicas:1.

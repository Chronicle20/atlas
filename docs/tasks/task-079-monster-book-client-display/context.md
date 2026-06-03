# Monster Book Client Display Wiring — Context

Companion to `plan.md`. Captures the verified facts, key files, and decisions an
implementer needs before touching code. Everything here was read from the repo on
2026-06-03 — verify against source, not memory (CLAUDE.md).

## Goal in one line

Wire the login `CharacterData` Monster Book section so the in-game window shows the
player's real cover + owned-card list, fetched per-login from `atlas-monster-book`,
with the version gate corrected (present GMS ≤ 87 + JMS, absent GMS v95).

## Key files (read before editing)

| File | Role | Change |
|---|---|---|
| `libs/atlas-constants/item/constants.go` | item id/classification constants | **Add** `MonsterBookCardBase = Id(2380000)`. Already has `ClassificationConsumableMonsterCard = Classification(238)`. |
| `libs/atlas-packet/character/data.go` | `CharacterData` encode/decode | **Add** `MonsterBookData`/`MonsterBookCard` types + `MonsterBook` field; **rewrite** `encodeMonsterBook`/`decodeMonsterBook` (lines 676–686); **split** the gate (encode line 131, decode line 186). |
| `libs/atlas-packet/character/data_test.go` | round-trip tests over `pt.Variants` | **Add** byte-level + round-trip tests. |
| `services/atlas-channel/atlas.com/channel/monsterbook/requests.go` | REST request builders | **Add** `CardsResource` + `requestCardsByCharacterId`. |
| `services/atlas-channel/atlas.com/channel/monsterbook/rest.go` | JSON:API wire models | **Add** `CardRestModel` + `ExtractCard`. |
| `services/atlas-channel/atlas.com/channel/monsterbook/processor.go` | REST consumption | **Add** `Card` domain type + `GetCardsByCharacterId` + provider; extend `Processor` interface. |
| `services/atlas-channel/atlas.com/channel/monsterbook/rest_test.go` | httptest round-trips | **Add** `CardRestModel` unmarshal + `GetCardsByCharacterId` tests. |
| `services/atlas-channel/atlas.com/channel/character/model.go` | immutable character model | **Add** `monsterBookCards []monsterbook.Card` field + getter/setter. |
| `services/atlas-channel/atlas.com/channel/character/builder.go` | model builder | **Thread** the new field (struct field, `CloneModel`, setter, `Build`). |
| `services/atlas-channel/atlas.com/channel/character/processor.go` | decorators | **Replace** `MonsterBookCoverDecorator` → `MonsterBookDecorator` (cover **and** cards, fail-open). |
| `services/atlas-channel/atlas.com/channel/character/mock/processor.go` | test mock | **Rename** mock method to match interface. |
| `services/atlas-channel/atlas.com/channel/socket/handler/character_info_request.go:31` | CharacterInfo handler | **Update** caller `cp.MonsterBookCoverDecorator` → `cp.MonsterBookDecorator`. |
| `services/atlas-channel/atlas.com/channel/socket/writer/character_data.go` | `BuildCharacterData` | **Populate** `cd.MonsterBook` from the model. |
| `services/atlas-channel/atlas.com/channel/kafka/consumer/session/consumer.go:166` | login decorator chain | **Add** `cp.MonsterBookDecorator` to the `GetById(...)` list. |
| `libs/atlas-packet/character/clientbound/info.go` | CharacterInfo packet (secondary) | **Thread** cover into the hardcoded `WriteInt(0)` at line 98. |
| `services/atlas-channel/atlas.com/channel/socket/writer/character_info.go` | `CharacterInfoBody` (secondary) | **Pass** `c.CoverCardId()` to `NewCharacterInfo`. |

## Verified facts (do not re-derive from memory)

- **Wire format (IDA-verified in design §2):** cover = full `int32` item id;
  then a **mode byte** (server emits `0`); then `short` count; then per card
  `short(cardId − 2380000)` + `byte(level)`. Mode 1 (bitmap+nibbles) is decoded by
  the client but the server need not emit it — emit mode 0 only.
- **Endianness:** `response.Writer` is **little-endian** (`binary.LittleEndian`,
  `libs/atlas-socket/response/writer.go:36-59`). `WriteInt`=4B, `WriteShort`=2B,
  `WriteByte`=1B.
- **Empty book is byte-identical today:** current stub writes `int(0) byte(0) short(0)`
  = mode-0 empty book. The change is strictly **additive** for v83/v87/JMS.
- **`pt.RoundTrip` asserts full byte consumption** (`reader.Available() > 0` → fail),
  so any gate/width mismatch surfaces as leftover bytes. This is the primary v95
  alignment guard.
- **`pt.Variants`** already includes GMS v28/v83/v87/v95 and JMS v185
  (`libs/atlas-packet/test/context.go:18`).
- **`/cards` endpoint** exists (`character/resource.go:35`, route
  `/characters/{characterId}/monster-book/cards`, handler `handleListCards`) and
  returns the **full** JSON:API collection — **no pagination** in the current
  handler. Card rest model: `card/rest.go` → `{ "id": cardId, "attributes": {
  "level", "isSpecial", "firstAcquiredAt" } }`.
- **`requests.SliceProvider[A, M]`** is the pattern for list endpoints
  (`libs/atlas-rest/requests/provider.go:22`; usage example `macro/processor.go:26`).
- **`character.CoverCardId()` already exists** (`model.go:352`) and is populated by the
  current `MonsterBookCoverDecorator`. The cover is modeled as `item.Id` (full id).

## Decisions carried from design

- **One unified `MonsterBookDecorator`** (2 REST calls, 1 chain entry, 1 fail-open
  boundary) — replaces the cover-only decorator (design §6-A, PRD §4.4).
- **Mode 0 simple list** on the wire (design §6-B). Cover sent as **full id**
  (design §6-C / §2.3.1) — flagged for live confirmation.
- **Single encoder for GMS v83/v87 and JMS** — no region branch in
  `encodeMonsterBook` (design §2.5, verified identical in the JMS IDB).

## ⚠️ Corrections to the design (found during planning)

1. **`MonsterBookCoverDecorator` is NOT unused.** Design §4.1.3 says "remove the
   now-unused cover-only decorator," but it is wired into
   `socket/handler/character_info_request.go:31` and stubbed in
   `character/mock/processor.go:75`. Replacing it **must** update both call sites or
   the build breaks. (Captured as Task 6.)

2. **The `>28` gate at `data.go:131` wraps more than the monster book.** It also
   guards `encodeNewYear` + `encodeArea` (GMS) / a JMS short + a trailing short.
   Design §3's snippet moves the *whole* block to `<= 87`, which would also drop
   newYear/area/trailing for v95 — and the design never verified v95 retains those.
   **Plan splits the gate**: monster book → `<= 87 || JMS`; newYear/area/trailing →
   unchanged `> 28 || JMS`. This is the minimal change: it removes only the monster
   book bytes for v95 and leaves the rest untouched. (Captured as Task 3, with an
   explicit v95-IDB verification step + a fallback if v95 also dropped the tail.)

## Dependencies / ordering

- Task 1 (constant) → Task 2 (encoder uses it).
- Task 2 (packet types) → Task 7 (`BuildCharacterData` maps to them).
- Task 4 (`Card` + `GetCardsByCharacterId`) → Task 6 (decorator uses it) → Task 5
  (model field the decorator sets) — implement 4 and 5 before 6.
- Task 6 (interface change) → Task 8 (chain wiring) and the mock/caller updates.
- Tasks 1–8 = primary fix. Task 9 = secondary CharacterInfo cover. Task 10 = full verify.

## Out of scope (PRD §2 / design §9)

`STATS_CHANGED` → client; atlas-ui admin widget; Fill-the-Book / card-exchange / card
scrolls; issue #655; any `atlas-monster-book` persistence or derived-stat change.

## Verification commands (Task 10)

Changed modules: `libs/atlas-constants`, `libs/atlas-packet`, `atlas-channel`
(only `atlas-channel`'s `go.mod` is a *service* module; libs are workspace modules).

```bash
# from each changed module dir:
go test -race ./...
go vet ./...
go build ./...
# from worktree root — only services whose go.mod changed:
docker buildx bake atlas-channel
# from repo root:
tools/redis-key-guard.sh
```

`atlas-monster-book` is **read-only** in this task (no `go.mod` change) → no bake.

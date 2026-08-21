# Jukebox Cash Item — Implementation Context

Companion to [plan.md](plan.md). PRD: [prd.md](prd.md). Design: [design.md](design.md).

---

## 1. The one-sentence shape

This is the **weather cash item pipeline with the BGM leg removed and a
client-supplied duration added**. Every hop already exists and works; nothing
here is a new architecture. If an implementer is unsure how something should
look, the answer is almost always "exactly like the weather equivalent" — and
the plan names that file and line for every task.

---

## 2. The two facts that shrink this relative to the PRD

The design phase resolved both against the client binaries, and they delete
work the PRD specified. An implementer reading only the PRD would build the
wrong thing.

1. **The client resolves the BGM itself.** `CMapLoadable::PlayNextMusic`
   @`0x61dab0` reads the used item's own WZ `info/path` node and hands it to
   `CSoundMan::PlayBGM`. The server never names a song. This deletes the entire
   atlas-data `info/path` workstream (PRD FR-2.1 – FR-2.4) and every
   `FieldEffect` BGM broadcast (FR-5.1.2, FR-5.2's restore).

   Sending a BGM `FieldEffect` is not merely redundant, it is a bug: that
   packet sets `m_sChangedBgmUOL`, and `CMapLoadable::RestoreBGM` @`0x61a4f0`
   prefers that string over the map's own music. The field would play the
   jukebox track forever after the song "ended".

2. **The client sends the song length.** The type-20 arm of
   `CWvsContext::SendConsumeCashItemUseRequest` encodes exactly one int32 —
   `IWzSound::length`, in milliseconds. Duration is client-supplied and
   server-capped, not the PRD's server constant (FR-4.2 / OQ-3).

---

## 3. Non-negotiables

| Rule | Why | Where enforced in the plan |
|---|---|---|
| The stop id is exactly `-1` | `PlayNextMusic` branches on `== -1`. Any other negative value fails `GetItemInfo`, returns without clearing `m_nJukeBoxItemID`, and `Update` re-enters `PlayNextMusic` every frame forever | Task 8: named constant `jukeboxStopItemId` + a test asserting the four wire bytes are `ff ff ff ff` |
| No `FieldEffect` BGM packet | Breaks `RestoreBGM` permanently (§2 above) | Task 8: called out in both handler comments; nothing in any task constructs one |
| `libs/atlas-packet/field/clientbound/play_jukebox.go` unchanged | Evidence-pinned for `gms_v95` and `jms_v185` | Final gate: `git diff --stat` over the codec and `docs/packets/audits` must be empty |
| Duration stays in **milliseconds** end to end | Weather's payload carries seconds; converting would truncate a real track | Task 2 payload field comment, Task 3 handler comment, Task 5 cap arithmetic |
| Arm keyed on cash-slot **type 20**, not an item id | `get_cashslot_item_type` maps 510→20 on every version examined and nothing else yields 20 | Task 7 const comment |

---

## 4. Task dependency order

```
Task 1 (packet codec) ─────────────┐
Task 2 (libs/atlas-saga action) ───┼──► Task 7 (channel arm)
                       └──► Task 3 (orchestrator) ──► [runtime only]
Task 4 (maps registry) ──► Task 5 (maps consumer/sweep/REST) ──► Task 8 (channel broadcast)
                                        └──► Task 6 (channel REST client) ──► Task 8
```

Compile-time dependencies are only within a module plus the two `libs`
packages. Tasks 4/5/6 and 1/2/3/7 can proceed in either order; Task 8 needs
Task 6 to compile. Tasks 5 and 8 have no compile-time coupling — they agree by
JSON contract, which is why both tasks' Interfaces blocks spell out the exact
message bodies rather than pointing at each other.

---

## 5. Key files by service

| Service / lib | Module root for `go build`/`go test` | New | Modified |
|---|---|---|---|
| `libs/atlas-packet` | `libs/atlas-packet` | `cash/serverbound/item_use_song_player{,_test}.go` | — |
| `libs/atlas-saga` | `libs/atlas-saga` | — | `model.go`, `payloads.go`, `unmarshal.go`, `unmarshal_test.go`, `world_transfer_test.go` |
| `atlas-saga-orchestrator` | `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator` | `map_command/producer_test.go` | `kafka/message/map/kafka.go`, `map_command/{processor,producer}.go`, `saga/{model,handler,event_acceptance}.go`, `saga/{handler,event_acceptance}_test.go` |
| `atlas-maps` | `services/atlas-maps/atlas.com/maps` | `map/jukebox/{registry,processor,producer,rest,resource}.go`, `map/jukebox/registry_test.go`, `tasks/jukebox{,_test}.go`, `kafka/consumer/map/{consumer,testmain}_test.go` | `kafka/message/map/{command,kafka}.go`, `kafka/consumer/map/consumer.go`, `main.go` |
| `atlas-channel` | `services/atlas-channel/atlas.com/channel` | `jukebox/{rest,requests,processor}.go`, `jukebox/mock/processor.go`, `jukebox/requests_test.go`, `socket/handler/character_cash_item_use_jukebox_test.go` | `socket/handler/character_cash_item_use.go`, `saga/model.go`, `kafka/message/map/kafka.go`, `kafka/consumer/map/consumer.go`, `kafka/consumer/map/consumer_test.go` |
| `atlas-data` | — | **none** | **none** |
| `libs/atlas-constants` | — | none | none — `ClassificationSongPlayer` already exists at `item/constants.go:91` |
| `atlas-ui` | — | none | none |

---

## 6. Test seams the plan relies on

These already exist. An implementer should not build new ones.

**atlas-channel handler tests** (`socket/handler`):
- `installCashItemInSlotSeam(t, slot, templateId)` — swaps the `cashItemInSlotFunc` package var (`character_cash_item_use_test.go:34`).
- `newCashItemUseTestSession(t, characterId)` — GMS v83 tenant + ctx + session with a `discardConn` (`:61`). v83 is deliberate: it makes `updateTimeFirst` resolve **false**, matching the raw sub-body bytes the tests build.
- `cashItemUsePrefix(slot, itemId)` — the common `ItemUse` header bytes (`:92`).
- `installCapturingProducer()` — captures Kafka messages per topic without a broker (`cash_item_gachapon_test.go:50`).
- `gaugeProducerRecorder` — counts and records socket announces (`character_damage_test.go:544`).
- `newKiteCharacterServer(t, id, name, x, y)` — a fake `CHARACTERS_SERVICE_URL` returning a JSON:API character (`character_cash_item_use_kite_test.go:20`). The jukebox arm resolves the player name the same way the kite arm does, so this fixture is reused rather than a new seam being added.

**atlas-channel map consumer tests** (`kafka/consumer/map`):
- `newTestCtx`, `newTestField`, `addFieldSession` (`consumer_test.go:28-63`).
- `doorAnnounce` — the package-level `session.Announce` seam (`consumer.go:759`). Despite the name it is already used for non-door writers (`ContiMove`, `FieldEffect`); the jukebox handlers announce through it so they are testable with the `nil`-conn sessions `addFieldSession` builds. **This is why the plan routes the broadcast through a closure rather than copying weather's direct `session.Announce(...)` call chain** — the weather handlers have no tests for exactly that reason.

**atlas-saga-orchestrator** (`saga`):
- `allActions` in `event_acceptance_test.go:23-59` is the single source of truth for two completeness tests (`TestAcceptanceTable_EveryActionRepresented`, `TestStepUnmarshal_EveryActionRepresented`). Adding one line there is Task 3's failing test.

**atlas-maps** (`tasks`):
- `tasks/weather_test.go` demonstrates the sweep-with-a-spy-`emit` pattern, including a test-local context-key type instead of importing `libs/atlas-env` — `tasks` sits outside env-domain-guard's permitted import list.

---

## 7. Deliberate design choices worth not re-litigating

- **A parallel `jukebox` package rather than generalising `weather`.** They differ in entry payload, duration source, end-event body, and companion packets. A generic abstraction parameterised on four axes after exactly two instances is premature, and it would put a refactor of an already-verified weather path inside a new-feature task. Extract at the third instance (design §3.4).
- **Registry stays in-process and unpersisted.** An `atlas-maps` restart drops active songs; clients keep playing until they change maps. Same acceptance weather already carries, for a cosmetic effect bounded at ten minutes (design §3.5).
- **`maxJukeboxDuration = 10 * time.Minute` is a judgement call**, an order of magnitude above any real WZ sound. If a real track exceeds it the song stops early — one line to retune.
- **The zero-length rejection lives in the channel arm, not atlas-maps.** By the time the command reaches atlas-maps the `DestroyAsset` step has already committed, so a rejection there would consume the item for nothing.
- **Broadcast includes the sending player.** Their client applies nothing locally (`m_bJukeBoxPlaying` is written only inside `PlayNextMusic`), and its own pre-gate prevents a second send, so no double-apply is possible (design §6.3).
- **`maxJukeboxDuration` is package-scoped** while the analogous `maxWeatherDuration` is function-scoped. Deliberate — the new constant is referenced from a test; weather's is left alone.

---

## 8. Task sizing notes

`tools/plan-lint.sh` reports F4 warnings on Tasks 4, 5, and 6. Those counts
include the read-only template files each task's `### Files` block names (the
weather equivalents an implementer copies from). Editable counts are 6, 8, and
5 — only Task 5 is genuinely over, and §8's third bullet records why. Listing
the templates is what removes the implementer's discovery phase, so they stay.

All eight tasks are at or under the ~6-editable-file guideline except where
noted:

- **Task 3 (orchestrator) touches 8 files in one service.** Left large deliberately: six of the eight edits are one-to-three-line insertions next to an existing `FieldEffectWeather` line, and splitting them would produce a non-compiling intermediate commit (the interface method, the switch case, and the implementation must land together; the two test-list edits are what make the completeness tests fail first). The single genuine unit of work is `handlePlayJukebox` plus its command provider.
- **Task 5 (atlas-maps) touches 8 editable files.** Left large deliberately. It is one deliverable — a song actually taking effect *and* expiring — and splitting the sweep from the consumer would leave an intermediate state where songs start and never stop. Six of the eight files are near-verbatim transcriptions of a named weather template (`producer.go`, `resource.go`, `tasks/jukebox.go`, `tasks/jukebox_test.go`, `testmain_test.go`, plus two localised insertions in `main.go`), which is exactly the "same mechanical change repeated" case Step 5a says batches fine. The realistic tool-call cost is well under the 120 budget, so this should not produce a PARTIAL.
- **Task 8 (atlas-channel) touches 3 files**, but one of them is a 50 KB consumer. The plan names the exact insertion points (lines ~97, ~346, ~974, ~1062) so no discovery sweep of that file is needed.

No task crosses a service boundary. Tasks 2/3 and 4/5 and 6/8 each split a
would-be cross-module task at the module line.

---

## 9. What "done" means here

Beyond each task's own green module-local tests:

1. Flagless `tools/verify.sh` exits 0 from the worktree root (controller-run, in a fresh context via `atlas-verifier`).
2. `grep -rn "PlayJukeboxWriter" services/` shows invoking call sites, not just the `main.go` registration — the PRD's explicit "no longer scaffold-only" criterion.
3. `git diff --stat main -- libs/atlas-packet/field/clientbound/play_jukebox.go docs/packets/audits` is empty.
4. Code review before the PR, per CLAUDE.md. The cross-service seams worth tracing by hand: the `PLAY_JUKEBOX` body contract between Task 3 and Task 5, and the `JUKEBOX_START`/`JUKEBOX_END` body contract between Task 5 and Task 8 — neither has a compile-time check, only the JSON tags agreeing.

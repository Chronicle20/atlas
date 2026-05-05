# Monster Book — Context Pack

This document is a quick-reference companion to `plan.md`. Read this first when picking up the task; it points you at the design decisions, key files, and code patterns the plan depends on.

---

## Authoritative documents

- **PRD:** `docs/tasks/task-056-monster-book/prd.md` — what we're building and why.
- **Design:** `docs/tasks/task-056-monster-book/design.md` — how the work is structured. Resolves all PRD open questions in §2.

If the plan and design disagree, design wins. If design and PRD disagree, PRD wins.

## Critical decisions locked at design phase

| # | Decision | Source |
|---|----------|--------|
| 1 | EXP bonus formula: `bonusPercent = bookLevel` (1% per book level). | design §6.4 |
| 2 | `consumeOnPickup` wiring lives in **atlas-inventory** (pre-insert check) and emits a generic `ITEM.CONSUMED_ON_PICKUP` event consumed by **atlas-consumables**. | design §4.1 |
| 3 | Foreign card-gain effect broadcasts to entire map (Cosmic parity). | design §5.3 |
| 4 | Idempotency uses per-row `last_event_id` (UUID) columns + `WHERE … IS DISTINCT FROM` guarded upserts. **No** dedup table. | design §7 |
| 5 | atlas-data is unchanged. `GET /data/consumables/{cardId}` already exposes `monsterId`. | design §4.5 |
| 6 | Quest condition wire shape: `{type: "monsterBookCount", operator, value}`. Evaluated in **atlas-query-aggregator**, not atlas-quest. | design §4.4, §2 (#8/#9) |
| 7 | New module name: `atlas-monster-book` (short form, mirrors `atlas-keys`/`atlas-fame`). Service path: `services/atlas-monster-book/atlas.com/monster-book/`. Template: atlas-keys. | design §3.1 |
| 8 | Login-time cover read uses the existing decorator pattern on atlas-channel `character.Processor`, alongside `PetAssetEnrichmentDecorator`/`SkillModelDecorator`/`QuestModelDecorator`. | design §5.4 |

## Scope guard rails

- **Card item id range:** `2380000`–`2389999` inclusive. `cardId / 1000 >= 2388` ⇒ *special*; else *normal*.
- **Per-card level cap:** 5. The 6th+ pickup acknowledges (`flag=0`) and does **not** mutate state.
- **Cover validation tightened beyond Cosmic:** `coverCardId == 0 || (in card range AND owned at level ≥ 1)`.
- **No backfill:** new tables are empty for all existing characters; that is correct.
- **No tenant-configurable knobs in v1.**
- **No card scrolls / set bonuses / decks.**
- **No "Fill the Book" / Barry NPC scripts.**
- **No HP/MP bonuses driven by book level.**

## Service map (touched files / packages)

| Service | What changes | Key paths |
|---|---|---|
| **atlas-monster-book** (new) | Full greenfield service. Two domain packages (`collection`, `card`), Kafka consumers/producer, REST, character lifecycle subscriber. | `services/atlas-monster-book/atlas.com/monster-book/` (entire tree) |
| **atlas-inventory** | Pre-insert `consumeOnPickup` branch in `compartment.Processor.AttemptItemPickUp`. Emits `ITEM.CONSUMED_ON_PICKUP`. | `services/atlas-inventory/atlas.com/inventory/compartment/processor.go` (lines `1148–1238`), new `kafka/message/pickup/`, `kafka/producer/` reuse |
| **atlas-consumables** | New consumer for `ITEM.CONSUMED_ON_PICKUP`. Card branch produces `MONSTER_BOOK.CARD_PICKED_UP`. | new `services/atlas-consumables/atlas.com/consumables/kafka/consumer/pickup/`, new `kafka/message/monsterbook/` |
| **atlas-channel** | `0x39` recv handler; `0x53`/`0x54` writers; effect packets on card add; consumers for `MONSTER_BOOK.CARD_ADDED` / `COVER_CHANGED`; `MonsterBookCoverDecorator` on character info. | `services/atlas-channel/atlas.com/channel/socket/handler/`, `kafka/consumer/monsterbook/` (new), `character/processor.go`, `socket/handler/character_info_request.go` |
| **atlas-quest** | Add `MonsterBookCountCondition` constant. Build `ConditionInput{Type: "monsterBookCount", Operator: ">=", Value: N}` from quest definitions where applicable. | `services/atlas-quest/atlas.com/quest/data/validation/model.go`, `…/processor.go` |
| **atlas-query-aggregator** | New `MonsterBookCountCondition` constant + `Condition.Evaluate` case calling atlas-monster-book REST. | `services/atlas-query-aggregator/atlas.com/query-aggregator/validation/model.go` (lines `20–56`, `124`, `380+`), `…/rest.go`, new `monsterbook/` REST client package |
| **libs/atlas-saga** | Add `MonsterBookCountCondition = "monsterBookCount"` constant to share with atlas-quest. | `libs/atlas-saga/validation.go` (lines `9–45`) |
| **libs/atlas-packet** | New `monsterbook/` clientbound writers for opcodes `0x53` `MonsterBookSetCard` and `0x54` `MonsterBookSetCover`. | new `libs/atlas-packet/character/clientbound/monsterbook/` |
| **libs/atlas-opcodes** | No code change — opcodes are wired via tenant configuration (atlas-tenants); the writer/handler **names** are the integration point. | `libs/atlas-opcodes/registry.go` (read-only) |
| **atlas-data** | **No change.** | n/a |
| **atlas-character** | **No change.** | n/a |
| **atlas-tenants** | **No change.** Tenant operators add three opcode mappings (`0x39` handler, `0x53`/`0x54` writers) using the new writer/handler names; this is a runtime data change, not a code change. | n/a |
| **atlas-ui** | New "Monster Book" widget on character detail page; consumes the new monster-book service + existing atlas-data endpoints. | `services/atlas-ui/src/components/features/characters/`, `services/atlas-ui/src/services/api/` |

## Code patterns to follow (atlas-keys is the template)

The new service mirrors atlas-keys exactly. Key reference files:

- `services/atlas-keys/atlas.com/keys/main.go` — bootstrap shape (logger, db connect with Migration, consumer manager, REST server).
- `services/atlas-keys/atlas.com/keys/key/processor.go` — `Processor` interface + `ProcessorImpl` + `NewProcessor(l, ctx, db)` pattern.
- `services/atlas-keys/atlas.com/keys/key/entity.go` — GORM entity with `TableName()` + `Migration(db *gorm.DB) error`.
- `services/atlas-keys/atlas.com/keys/key/builder.go` — `ModelBuilder` with `Build()` (validation) + `MustBuild()`.
- `services/atlas-keys/atlas.com/keys/key/administrator.go` — raw GORM `create`/`update`/`delete` helpers.
- `services/atlas-keys/atlas.com/keys/key/provider.go` — `EntityProvider`/`SliceQuery` patterns.
- `services/atlas-keys/atlas.com/keys/key/rest.go` — `RestModel` with `GetName()`/`GetID()`/`SetID()` + `Transform`.
- `services/atlas-keys/atlas.com/keys/character/resource.go` — `InitResource(si)(db)` + `RegisterHandler` / `RegisterInputHandler` route wiring.
- `services/atlas-keys/atlas.com/keys/rest/handler.go` — `ParseCharacterId` and friends.
- `services/atlas-keys/atlas.com/keys/kafka/producer/producer.go` — `Provider func(token string) producer.MessageProducer`.
- `services/atlas-keys/atlas.com/keys/kafka/message/message.go` — `Buffer`, `Emit(p)`, `EmitWithResult`.
- `services/atlas-keys/atlas.com/keys/kafka/consumer/consumer.go` — `NewConfig(l)(name)(token)(groupId)`.
- `services/atlas-keys/atlas.com/keys/kafka/consumer/character/consumer.go` — character lifecycle subscriber and `InitConsumers`/`InitHandlers` pattern.
- `services/atlas-keys/atlas.com/keys/kafka/message/character/kafka.go` — `EVENT_TOPIC_CHARACTER_STATUS` constant + `StatusEvent`/`DeletedStatusEventBody`.
- `services/atlas-keys/Dockerfile` — build pattern (go.work, replace directives, libs to copy).

## Cross-cutting Atlas conventions (CLAUDE.md / DOM-21)

- **Multi-tenancy:** every read/write lead with `tenant_id`; pull from `tenant.MustFromContext(ctx)` per-request (already in `Processor` constructor).
- **Immutable models:** private fields + getters + `ModelBuilder`. Never mutate; always clone-build.
- **Processors:** `NewProcessor(l, ctx, db) Processor` returns interface, impl is `*ProcessorImpl`. Pure logic in `Method(mb)(args)`; side-effecting wrappers in `MethodAndEmit(args) error`.
- **Kafka:** `message.Buffer` for batching, `message.Emit(p)` for atomic emission. Consumers register via curried `InitConsumers(l)(cmf)(groupId)` + `InitHandlers(l)(db)(rf)`.
- **REST:** JSON:API via `api2go/jsonapi`. `GetName()` returns resource type. `RegisterHandler(l)(si)` for GET, `RegisterInputHandler[M]` for PATCH/POST.
- **Constants library:** before adding any numeric type or item-classification helper, check `libs/atlas-constants/` (DOM-21). For Monster Book, reuse `world.Id` (byte), `channel.Id` (byte), `character.Id` (uint32). Card IDs are plain `uint32` — they are item ids and need no new wrapper.

## Pre-existing entry points the plan grafts onto

- **atlas-channel `character_info_request.go:27-31`** — decorator slice for character info packet. We append `MonsterBookCoverDecorator` here.
- **atlas-channel `kafka/consumer/character/consumer.go:269-270`** — already consumes `ExperienceDistributionTypeMonsterBook` and writes `experience_status.MonsterBookBonus`. **Do not touch.** atlas-monster-book is the new producer.
- **atlas-inventory `compartment/processor.go:1148`** — `AttemptItemPickUp` is where the consume-on-pickup branch goes (before the existing `compartment.GetByCharacterAndType` lookup).
- **atlas-data `consumable/rest.go:55`** — already exposes `consumeOnPickup`. We just *read* it.
- **atlas-data `consumable/reader.go:77`** (existing) — already extracts `MonsterId` from WZ `info/mob` and exposes via `monsterId` JSON field. UI consumes this.
- **libs/atlas-packet `character/effect_body.go:31`** — `CharacterEffectMonsterBookCardGet` mode constant already exists. The card-gain effect uses the existing `CharacterEffectWriter`/`CharacterEffectForeignWriter` (opcode `0x0D`).

## Things that look like decisions but aren't

- **Chat-line packet on card pickup:** PRD §4.2 says `SHOW_ITEM_GAIN_INCHAT` (`0x0D`) is sent on `ADDED`. Design §5.2 notes that in v83 the `EffectSimple{MonsterBookCardGet}` write often drives the chat-line render client-side and a separate packet may be redundant. **Plan assumes only `EffectSimple{MonsterBookCardGet}` is sent.** If end-to-end testing shows the chat line is missing, a follow-up task wires an `ItemGainInChat` writer next to the effect writer. This is *not* a v1 acceptance gate beyond the design's existing wording.

## Open follow-ups (out of scope for this task)

- Tenant-configurable EXP bonus formula and special-card threshold.
- Backfilling existing quest data with `monsterBookCount` requirements.
- "Fill the Book" / Barry NPC scripts.
- Card trading / dropping / destroying.
- Foreign-effect optimization beyond whole-map broadcast.

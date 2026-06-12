# Pet Evolution — Implementation Context

Companion to `plan.md`. Captures the key files, decisions, and cross-service contracts an implementer needs, all verified against the code in this worktree on 2026-06-12.

## Architecture in one paragraph

WZ evolution data is parsed by **atlas-data** and exposed as plain attributes on the pet REST resource. **atlas-pets** owns the random outcome roll (using the WZ relative weights) and the in-place mutation of its own pet row, then cascades a new **`CHANGE_TEMPLATE`** command to **atlas-inventory**, which swaps the asset's `templateId` in place and emits the existing **`UPDATED`** event (never `DELETED`, so atlas-pets' asset consumer does not destroy the pet). **Egg hatching** runs inside the `SPAWN` path (no saga). **NPC evolution** is a **`PetEvolution`** saga `[destroy_item → award_mesos → evolve_pet]` with reverse-walk compensation refunding Rock + mesos on failure. **atlas-channel** needs no code change.

## Confirmed WZ data (local `tmp/<uuid>/GMS/83.1/Item.wz/Pet/`)

| Template | evolNo | evol targets | evolProb | reqItemID | reqPetLvl | Classification |
|---|---|---|---|---|---|---|
| 5000028 Dragon Egg | 1 | 5000029 | 100 | 0 | — | **egg** |
| 5000029 Baby Dragon | 4 | 5000030–5000033 | 33,33,33,1 | 5380000 | 15 | evolvable |
| 5000047 Robo Egg | 1 | 5000048 | 1000 | 0 | — | **egg** |
| 5000048 Baby Robo | 5 | 5000049–5000053 | 330,330,330,9,1 | 5380000 | 15 | evolvable |

**Weights are relative, not percentages** (robo uses 1000-base). The roll sums weights and picks proportionally.

**Egg discriminator (data-driven):** egg = `len(evolutions)==1 && reqItemId==0 && reqPetLevel==0`. Evolvable = `len(evolutions)>0 && reqItemId!=0`. Non-evolvable = no evolution data.

## Key files & verified anchors

### atlas-data (`services/atlas-data/atlas.com/data/pet/`)
- `reader.go:30-80` `Read` — parses `info/` via `i.GetIntegerWithDefault(name, def int32)` / `i.GetBool`. `fmt`/`strconv` imported. Insert evol parsing after life (line 57).
- `rest.go:9-16` `RestModel` — add `ReqPetLevel`, `ReqItemId`, `Evolutions []EvolutionRestModel`. Evolutions are **plain attributes** (no JSON:API relationship) → `GetReferences` unchanged.
- `xml.Node` accessors live in `data/xml/model.go`: `ChildByName(name) (*Node,error)`, `GetBool(name,def) bool`, `GetIntegerWithDefault(name,def int32) int32`, `GetString`.

### atlas-pets (`services/atlas-pets/atlas.com/pets/`)
- `pet/model.go` / `pet/builder.go` — immutable model; `NewModelBuilder` defaults level=1, fullness=100, expiration `now+2160h`, slot=-1; `Clone` preserves all fields. **No `SetTemplateId` yet** (Task 7 adds it). `Build()` validates templateId!=0, name!="", level 1-30, slot -1..2, fullness<=100.
- `pet/processor.go:35-71` interface; `:73-85` struct (`cp` character, `dp` data, `kp` producer; add `ip` inventory + `rollEvolution`); `:87-102` `NewProcessor`; `:104-143` options (`WithTransaction`, `WithDataProcessor`, etc.); `:337-432` `Spawn` (egg branch inserts after ownership check ~line 349); `:638-714` `AwardCloseness*` (model for `Evolve`).
- `pet/administrator.go` — column updaters via `db.Model(&Entity{}).Where(...).Update(col,val)`. Add `updateOnEvolve` (template_id + expiration via map update).
- `pet/producer.go` — `*EventProvider` funcs returning `model.Provider[[]kafka.Message]`; add `evolvedEventProvider`.
- `kafka/message/pet/kafka.go` — `Command[E]` envelope has `TransactionId, ActorId, PetId, Type, Body`; `StatusEvent[E]` has `PetId, OwnerId, Type, Body`. Add `EVOLVE` cmd + `EVOLVED` event.
- `kafka/consumer/pet/consumer.go` — `handleSpawnCommand`/`handleAwardClosenessCommand` (threads `c.TransactionId`) registration pattern; add `handleEvolveCommand`.
- `data/pet/{model,rest}.go` — atlas-data client; add `reqPetLevel`, `reqItemId`, `evolutions`, `IsEgg()`, `IsEvolvable()`.
- `inventory/model.go` `Cash()` → `compartment.Model`; `compartment/model.go:47` `FindFirstByItemId(templateId)` (baby-owned check) and `:38` `FindBySlot`.
- **atlas-pets does NOT currently emit any inventory command** — Task 11 adds the `CHANGE_TEMPLATE` producer (new `kafka/message/compartment` + emitter). The cascade is keyed by **petId** because atlas-pets does not know the cash slot.

### atlas-inventory (`services/atlas-inventory/atlas.com/inventory/`)
- `asset/administrator.go:54-92` — `updateSlot`/`updateQuantity` use `.Select("Col").Updates(...)`; `updateEquipmentStats` deliberately omits `TemplateId`. Add `updateTemplate` selecting only `TemplateId`.
- `asset/processor.go:227-266` `UpdateEquipmentStats` (model for `ChangeTemplate`); `GetById`, `Clone(a).SetTemplateId(...)`. `asset/builder.go:102` already has `SetTemplateId`.
- `asset/model.go` — `IsPet()` = `IsCash() && petId>0`; `PetId()`, `CashId()`, `Slot()`.
- `asset/producer.go:116-131` `UpdatedEventStatusProvider` emits `StatusEventTypeUpdated` with full `AssetData` (does NOT include TemplateId in the body, but the envelope `TemplateId` field carries it).
- `kafka/message/compartment/kafka.go:13-31` command consts; `:34` `Command[E]` (`TransactionId, CharacterId, InventoryType, Type, Body`); `:141` `ModifyEquipmentCommandBody`. Add `CHANGE_TEMPLATE`.
- `kafka/consumer/compartment/consumer.go:84,333` registration + `handleModifyEquipmentCommand` pattern.
- `compartment/processor.go:1746-1765` `ModifyEquipment*` (model for `ChangeTemplate`); resolves asset, locks via `LockRegistry()`, `database.ExecuteTransaction`, `WithAssetProcessor`. Resolve pet asset by iterating cash compartment for `IsPet() && PetId()==petId` (use `DecorateAsset` to populate `Assets()`).

### libs/atlas-saga
- `model.go:14-25` `Type` consts (add `PetEvolution`); `:43-76` `Action` consts (add `EvolvePet` after `GainCloseness`). `Saga{TransactionId, SagaType, InitiatedBy, Timeout, Steps}`, `Step[T]{StepId, Status, Action, Payload, ...}`.
- `payloads.go:251-255` `GainClosenessPayload` (model for `EvolvePetPayload`).
- `unmarshal.go:180-185` `GainCloseness` case (model for `EvolvePet` case). **Confirm `DestroyAssetPayload` / `AwardMesosPayload` field names here** — needed by the compensator (Task 18).
- `builder.go` `NewBuilder().SetSagaType(...).SetInitiatedBy(...).AddStep(...).Build()`.

### atlas-saga-orchestrator (`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/`)
- `pet/processor.go` — `Processor` interface (currently `GainCloseness*` only). `ProcessorImpl{l,ctx,t,p}`; `*AndEmit` wraps `message.Emit(p.p)`; `Gain*` does `mb.Put(pet2.EnvCommandTopic, AwardClosenessProvider(...))`. Add `Evolve*` + `EvolveProvider`.
- `saga/handler.go` — interface ~line 103 (`handleGainCloseness`); `HandlerImpl` has `petP pet.Processor` (line 164) **already injected** + `WithPetProcessor`; `GetHandler` switch (add `case EvolvePet`); `handleGainCloseness` at 1205 (model). `h.logActionError(...)`.
- `saga/compensator.go` — `CompensateFailedStep:162`; `CharacterCreation` reverse-walk branch at **line 180** (add `PetEvolution` branch after it); default switch at 224 only compensates the **failed** step; `compensateCharacterCreation:908` + `DispatchCharacterCreationRollbacks:967` (model — handles `AwardAsset/CreateSkill/CreateCharacter` inverses, **NOT** DestroyAsset/AwardMesos). `compP compartment.Processor` (line 56) has `RequestCreateItem(txId,charId,templateId,qty,expiration)` and `RequestDestroyItem`. **Add a character processor field** for the mesos refund (`charP.AwardMesosAndEmit(txId, channel.Model, charId, actorId, "SYSTEM", amount, showEffect)` — see `character/processor.go:136`); mirror how `handleAwardMesos` (handler.go) builds the `channel.Model`.

### atlas-npc-conversations (`services/atlas-npc-conversations/atlas.com/npc/`)
- `conversation/operation_executor.go:314` `isLocalOperationType` = `strings.HasPrefix(type,"local:")`; `:268` `ExecuteOperations` runs locals first then **one** saga for all remote ops via `:823 createSagaForOperations` (currently hard-codes `SetSagaType(saga.InventoryTransaction)` at line 826 — Task 22 makes it conditional). `:1016 createStepForOperation` switch (line 1021) maps op type → `(stepId, status, action, payload)`; `destroy_item` case at 1403; `stepId = "<type>-<characterId>"`. Context via `:128 getContextValue` / `:145 setContextValue`; `evaluateContextValueAsInt` resolves `{context.x}`.
- `pet/processor.go` — `GetPets(characterId) model.Provider[[]Model]`, `GetPetIdBySlot`. `pet/model.go` `Model{id,slot}` + `IsSpawned()` — **extend with templateId+level** (Task 19); `pet/rest.go` `RestModel` already has `TemplateId`/`Level`/`Slot`, only `Extract` needs updating.
- Condition types (`validation/model.go:10-21`): `item`, `meso`, `questStatus`, `jobId`, `mapId`, `fame`, `buddyCapacity`. Operators `=,>,<,>=,<=`. No new condition type needed (Decision C).
- Seed conversations: `deploy/seed/gms/12_1/npc-conversations/npc/npc-*.json`; envelope `{"data":{"attributes":{npcId,startState,states[]},"id":"...","type":"npc-conversation"}}`. State types: `dialogue`, `genericAction` (operations[] + outcomes[] with conditions), `listSelection`, `askStyle`, etc.

### atlas-channel — verify-only
- `libs/atlas-packet/pet/clientbound/activated.go:55` writes `templateId` with no version branch.
- `services/atlas-channel/.../kafka/consumer/pet/consumer.go` handles `SPAWNED`/`DESPAWNED` → spawn/despawn packets.
- channel asset consumer handles `UPDATED` → `InventoryChangeWriter` (icon refresh).

## Cross-service contracts (must stay byte-compatible)

1. **pet `EVOLVE` command** — topic `COMMAND_TOPIC_PET`, `Command[EvolveCommandBody]` with `TransactionId, PetId, Type="EVOLVE"`. Emitter: saga-orchestrator `pet.EvolveProvider`. Consumer: atlas-pets `handleEvolveCommand`. Both services define their own copy of the envelope — keep JSON tags identical.
2. **pet `EVOLVED` event** — topic `EVENT_TOPIC_PET_STATUS`, `StatusEvent[EvolvedStatusEventBody]{Slot, OldTemplateId, NewTemplateId, TransactionId}`.
3. **inventory `CHANGE_TEMPLATE` command** — topic `COMMAND_TOPIC_COMPARTMENT`, `Command[ChangeTemplateCommandBody]{TransactionId, CharacterId, InventoryType=Cash, Type="CHANGE_TEMPLATE", Body{PetId, NewTemplateId}}`. Emitter: atlas-pets (new copy in `kafka/message/compartment`). Consumer: atlas-inventory `handleChangeTemplateCommand`. **JSON tags and `Type` string must match exactly across both copies.**
4. **inventory `UPDATED` event** — reused unchanged; the in-place swap must emit `UPDATED`, never `DELETED` (this is what keeps the pet alive — FR-3.5).
5. **`PetEvolution` saga type** — set by npc-conversations when the batch contains `evolve_pet`; consumed by the orchestrator's compensator for reverse-walk. If these drift, compensation silently degrades to single-step (no refund).

## Decisions (from design.md)

- **A1** — atlas-pets cascades the inventory swap itself (keyed by petId); the saga never threads the rolled id.
- **B** — egg hatch is a direct cascade inside `SPAWN`, no saga (nothing to compensate).
- **C** — eligibility filter lives in the npc enumeration; atlas-pets re-validates on `EVOLVE` as the authority. No new conversation condition type.
- **Roll location** — atlas-pets owns the weighted roll (injectable for tests); the conversation does NOT use `select_random_weighted`.

## Risks / watch-items

- **Mesos refund sign + channel.Model construction** (Task 18) — confirm `AwardMesosPayload.Amount` type/sign and mirror `handleAwardMesos` exactly. Getting the sign wrong double-charges or double-refunds.
- **Appearance refresh** (Task 12) — `Despawn`+`Spawn` re-runs slot logic; if fragile, emit `despawnEventProvider`+`spawnEventProvider` directly from the buffer using `TemporalData`.
- **Inventory producer wiring** (Task 11) — if injecting a producer into `inventory.ProcessorImpl` is awkward, emit the `CHANGE_TEMPLATE` provider directly from the pet processor's buffer.
- **Mocks** — growing the `Processor` interfaces (atlas-pets pet/inventory/data, orchestrator pet, npc petdata) requires updating their mocks; build will flag them.
- **No new lib / no new go.mod** — `Dockerfile`/`go.work` need no edits; bake the five touched services.

## Verification gate (CLAUDE.md)

Per changed module: `go test -race ./... && go vet ./... && go build ./...`.
Repo root: `GOWORK=off tools/redis-key-guard.sh`.
Worktree root: `docker buildx bake atlas-data atlas-pets atlas-inventory atlas-saga-orchestrator atlas-npc-conversations`.
Runtime acceptance: hatch + evolve verified on a v83 and a v95+ tenant.
